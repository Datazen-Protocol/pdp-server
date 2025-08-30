package piece

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"errors"

	"github.com/Datazen-Protocol/pdp-server/pkg/blobstore"
	"github.com/Datazen-Protocol/pdp-server/pkg/models"
	"github.com/Datazen-Protocol/pdp-server/pkg/service"
	"github.com/filecoin-project/go-commp-utils/nonffi"
	commcid "github.com/filecoin-project/go-fil-commcid"
	commp "github.com/filecoin-project/go-fil-commp-hashhash"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PieceService handles piece preparation and upload using our own system
type PieceService struct {
	piriService service.PDPService
	blobStore   blobstore.Blobstore
	db          *gorm.DB
	mutex       sync.RWMutex
	pieces      map[string]*PieceInfo
}

// PieceInfo represents information about a prepared piece
type PieceInfo struct {
	ID                   string    `json:"id"`
	FilePath             string    `json:"file_path"`
	Size                 int64     `json:"size"`
	CommP                string    `json:"comm_p"`
	PieceCID             string    `json:"piece_cid"`
	DataCID              string    `json:"data_cid"`
	ProofSetID           int64     `json:"proof_set_id,omitempty"`
	Status               string    `json:"status"` // "prepared", "uploaded", "added_to_proofset"
	ErrorMessage         string    `json:"error_message,omitempty"`
	UploadURL            string    `json:"upload_url,omitempty"` // Piri's upload URL
	TransactionHash      string    `json:"transaction_hash,omitempty"`
	TransactionTimestamp time.Time `json:"transaction_timestamp,omitempty"`
}

// NewPieceService creates a new piece service
func NewPieceService(piriService service.PDPService, blobStore blobstore.Blobstore, db *gorm.DB) *PieceService {
	return &PieceService{
		piriService: piriService,
		blobStore:   blobStore,
		db:          db,
		pieces:      make(map[string]*PieceInfo),
	}
}

// PreparePiece prepares a file for upload using our own system
func (p *PieceService) PreparePiece(ctx context.Context, filePath string) (*PieceInfo, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Read file and calculate SHA256 hash
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Calculate SHA256 hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %v", err)
	}
	hashBytes := hash.Sum(nil)

	// Generate a unique ID for our piece tracking
	pieceID := fmt.Sprintf("%x", hashBytes[:8]) // Use first 8 bytes of hash as ID

	// Create piece info
	pieceInfo := &PieceInfo{
		ID:        pieceID,
		FilePath:  filePath,
		Size:      fileInfo.Size(),
		Status:    "prepared",
		UploadURL: "", // No upload URL needed for our system
	}

	p.pieces[pieceID] = pieceInfo
	log.Printf("Prepared piece %s for file %s", pieceID, filePath)
	return pieceInfo, nil
}

// UploadPiece uploads a piece to the blob store
func (p *PieceService) UploadPiece(ctx context.Context, pieceID string, fileContent []byte) (*PieceInfo, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if piece already exists
	if _, exists := p.pieces[pieceID]; exists {
		return nil, fmt.Errorf("piece %s already exists", pieceID)
	}

	// Pad file content to next power of 2 for Filecoin compatibility
	paddedContent := padToPowerOfTwo(fileContent)

	// Calculate the piece commitment (CommP)
	cp := &commp.Calc{}
	if _, err := io.Copy(cp, bytes.NewReader(paddedContent)); err != nil {
		return nil, fmt.Errorf("failed to calculate commp: %v", err)
	}
	digest, paddedPieceSize, err := cp.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to get commp digest: %v", err)
	}
	pieceCID, err := commcid.DataCommitmentV1ToCID(digest)
	if err != nil {
		return nil, fmt.Errorf("failed to convert commp to piece CID: %v", err)
	}

	// Create piece info
	piece := &PieceInfo{
		ID:       pieceID,
		Size:     int64(len(paddedContent)), // Use padded size for Filecoin compatibility
		CommP:    hex.EncodeToString(digest),
		PieceCID: pieceCID.String(),
		DataCID:  pieceCID.String(),
		Status:   "uploaded",
	}

	// Store the padded file in blob store using piece CID as key
	if err := p.blobStore.Put(ctx, piece.PieceCID, bytes.NewReader(paddedContent)); err != nil {
		return nil, fmt.Errorf("failed to store piece in blob store: %v", err)
	}

	// Store piece info in memory
	p.pieces[pieceID] = piece

	log.Printf("Successfully uploaded piece %s with CommP: %s, PieceCID: %s, PaddedSize: %d",
		pieceID, piece.CommP, piece.PieceCID, paddedPieceSize)

	return piece, nil
}

// AddPieceToProofSet adds a piece to a proof set by creating the necessary database entries and using Piri's method
func (p *PieceService) AddPieceToProofSet(ctx context.Context, pieceID string, proofSetID int64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	piece, exists := p.pieces[pieceID]
	if !exists {
		return fmt.Errorf("piece not found: %s", pieceID)
	}

	if piece.Status != "uploaded" {
		return fmt.Errorf("piece %s is not uploaded", pieceID)
	}

	// Parse the piece CID
	pieceCID, err := cid.Decode(piece.PieceCID)
	if err != nil {
		return fmt.Errorf("invalid piece CID: %v", err)
	}

	// Calculate the padded piece size (must be a multiple of 127 bytes)
	// For simplicity, we'll use the original file size as the padded size
	// In a real implementation, this should be calculated properly
	paddedSize := abi.PaddedPieceSize(piece.Size)

	// Create piece info for the unsealed CID generation
	proofType := abi.RegisteredSealProof_StackedDrg64GiBV1_1
	pieceInfo := abi.PieceInfo{
		Size:     paddedSize,
		PieceCID: pieceCID,
	}

	// Generate the unsealed CID (this is what Piri does)
	generatedCID, err := nonffi.GenerateUnsealedCID(proofType, []abi.PieceInfo{pieceInfo})
	if err != nil {
		return fmt.Errorf("failed to generate unsealed CID: %v", err)
	}

	log.Printf("Generated unsealed CID: %s for piece: %s", generatedCID, piece.PieceCID)

	// Create the database entries that Piri expects
	// We need to create entries in parked_pieces, parked_piece_refs, and pdp_piecerefs
	if err := p.createPiriDatabaseEntries(ctx, piece); err != nil {
		return fmt.Errorf("failed to create Piri database entries: %v", err)
	}

	// Create the AddRootRequest that Piri expects
	addRootReq := service.AddRootRequest{
		RootCID:     generatedCID.String(),    // Use our generated CID as the root
		SubrootCIDs: []string{piece.PieceCID}, // Use our piece CID as the subroot
	}

	// Use Piri's ProofSetAddRoot method
	result, err := p.piriService.ProofSetAddRoot(ctx, proofSetID, []service.AddRootRequest{addRootReq})
	if err != nil {
		piece.Status = "error"
		piece.ErrorMessage = fmt.Sprintf("failed to add root to proof set: %v", err)
		return err
	}

	// Store the transaction hash for monitoring
	if txHash, ok := result.(string); ok {
		piece.TransactionHash = txHash
		log.Printf("Transaction sent: %s", txHash)
	}

	// Update piece status to pending
	piece.ProofSetID = proofSetID
	piece.Status = "pending_confirmation"
	piece.TransactionTimestamp = time.Now()

	log.Printf("Added piece %s to proof set %d, transaction pending confirmation", pieceID, proofSetID)
	return nil
}

// createPiriDatabaseEntries creates the necessary database entries that Piri expects
func (p *PieceService) createPiriDatabaseEntries(ctx context.Context, piece *PieceInfo) error {
	log.Printf("Creating Piri database entries for piece: %s", piece.PieceCID)

	// We need to create entries in the following order:
	// 1. ParkedPiece - stores the piece information
	// 2. ParkedPieceRef - references the parked piece
	// 3. PDPPieceRef - links to our service with the piece CID

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: Create ParkedPiece entry
		parkedPiece := &models.ParkedPiece{
			PieceCID:        piece.PieceCID,
			PiecePaddedSize: piece.Size, // Use the file size as padded size
			PieceRawSize:    piece.Size, // Use the file size as raw size
			Complete:        true,       // Mark as complete since we have the data
			LongTerm:        true,       // Mark as long term storage
		}

		if err := tx.Create(parkedPiece).Error; err != nil {
			return fmt.Errorf("failed to create parked piece: %v", err)
		}

		log.Printf("Created ParkedPiece with ID: %d", parkedPiece.ID)

		// Step 2: Create ParkedPieceRef entry
		parkedPieceRef := &models.ParkedPieceRef{
			PieceID:     parkedPiece.ID,
			DataURL:     "",                   // We don't have a data URL since we're using our own storage
			DataHeaders: datatypes.JSON("{}"), // Empty JSON headers
			LongTerm:    true,
		}

		if err := tx.Create(parkedPieceRef).Error; err != nil {
			return fmt.Errorf("failed to create parked piece ref: %v", err)
		}

		log.Printf("Created ParkedPieceRef with RefID: %d", parkedPieceRef.RefID)

		// Step 3: Create PDPPieceRef entry
		pdpPieceRef := &models.PDPPieceRef{
			Service:          "storacha", // Our service name
			PieceCID:         piece.PieceCID,
			PieceRef:         parkedPieceRef.RefID,
			ProofsetRefcount: 0, // Start with 0 references
		}

		if err := tx.Create(pdpPieceRef).Error; err != nil {
			return fmt.Errorf("failed to create PDP piece ref: %v", err)
		}

		log.Printf("Created PDPPieceRef with ID: %d", pdpPieceRef.ID)

		return nil
	})
}

// GetPiece retrieves piece information
func (p *PieceService) GetPiece(ctx context.Context, pieceID string) (*PieceInfo, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	piece, exists := p.pieces[pieceID]
	if !exists {
		return nil, fmt.Errorf("piece not found: %s", pieceID)
	}

	return piece, nil
}

// ListPieces returns all pieces
func (p *PieceService) ListPieces(ctx context.Context) ([]*PieceInfo, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	pieces := make([]*PieceInfo, 0, len(p.pieces))
	for _, piece := range p.pieces {
		pieces = append(pieces, piece)
	}

	return pieces, nil
}

// GetPieceContent retrieves piece content from the blob store
func (p *PieceService) GetPieceContent(ctx context.Context, pieceID string) ([]byte, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Check if piece exists
	piece, exists := p.pieces[pieceID]
	if !exists {
		return nil, fmt.Errorf("piece not found: %s", pieceID)
	}

	// Get from blob store using piece CID as key
	obj, err := p.blobStore.Get(ctx, piece.PieceCID)
	if err != nil {
		return nil, fmt.Errorf("failed to get piece content: %v", err)
	}
	defer obj.Close()

	// Read content
	content, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read piece content: %v", err)
	}

	return content, nil
}

// MonitorTransactionStatus checks the status of a pending transaction
func (p *PieceService) MonitorTransactionStatus(ctx context.Context, pieceID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	piece, exists := p.pieces[pieceID]
	if !exists {
		return fmt.Errorf("piece not found: %s", pieceID)
	}

	if piece.Status != "pending_confirmation" {
		return fmt.Errorf("piece %s is not in pending confirmation status", pieceID)
	}

	if piece.TransactionHash == "" {
		return fmt.Errorf("no transaction hash found for piece %s", pieceID)
	}

	// Check transaction status in Piri's database
	var messageWait models.MessageWaitsEth
	err := p.db.WithContext(ctx).
		Where("signed_tx_hash = ?", piece.TransactionHash).
		First(&messageWait).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Transaction %s not found in database, still pending", piece.TransactionHash)
			return nil
		}
		return fmt.Errorf("failed to query transaction status: %v", err)
	}

	// Update piece status based on transaction status
	switch messageWait.TxStatus {
	case "confirmed":
		if messageWait.TxSuccess != nil && *messageWait.TxSuccess {
			piece.Status = "added_to_proofset"
			log.Printf("Transaction %s confirmed successfully", piece.TransactionHash)
		} else {
			piece.Status = "transaction_failed"
			piece.ErrorMessage = "Transaction failed on blockchain"
			log.Printf("Transaction %s failed on blockchain", piece.TransactionHash)
		}
	case "pending":
		log.Printf("Transaction %s still pending", piece.TransactionHash)
	default:
		log.Printf("Transaction %s has status: %s", piece.TransactionHash, messageWait.TxStatus)
	}

	return nil
}

// padToPowerOfTwo pads data to the next power of 2 size for Filecoin compatibility
func padToPowerOfTwo(data []byte) []byte {
	size := len(data)
	if size == 0 {
		return data
	}

	// Find next power of 2
	nextPowerOfTwo := 1
	for nextPowerOfTwo < size {
		nextPowerOfTwo <<= 1
	}

	// If already a power of 2, no padding needed
	if nextPowerOfTwo == size {
		return data
	}

	// Pad with zeros
	padded := make([]byte, nextPowerOfTwo)
	copy(padded, data)

	return padded
}
