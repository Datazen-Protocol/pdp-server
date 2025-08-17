package upload

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/piri/pkg/store/blobstore"
)

// UploadService handles file uploads and storage
type UploadService struct {
	blobStore    blobstore.Blobstore
	uploads      map[string]*UploadResult // Track uploaded files by CID
	metadataFile string                  // Path to persistent metadata file
	mutex        sync.RWMutex            // Protect concurrent access
}

// UploadResult contains information about the uploaded file
type UploadResult struct {
	CID        string    `json:"cid"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
	PieceID    string    `json:"piece_id,omitempty"`
}

// NewUploadService creates a new upload service
func NewUploadService(blobStore blobstore.Blobstore, dataDir string) *UploadService {
	metadataFile := filepath.Join(dataDir, "uploads.json")
	
	service := &UploadService{
		blobStore:    blobStore,
		uploads:      make(map[string]*UploadResult),
		metadataFile: metadataFile,
	}
	
	// Load existing metadata
	service.loadMetadata()
	
	return service
}

// loadMetadata loads existing upload metadata from file
func (s *UploadService) loadMetadata() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if metadata file exists
	if _, err := os.Stat(s.metadataFile); os.IsNotExist(err) {
		// File doesn't exist, start with empty map
		return nil
	}
	
	// Read metadata file
	data, err := os.ReadFile(s.metadataFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}
	
	// Parse JSON
	var uploads []*UploadResult
	if err := json.Unmarshal(data, &uploads); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}
	
	// Load into map
	for _, upload := range uploads {
		s.uploads[upload.CID] = upload
	}
	
	return nil
}

// saveMetadata saves upload metadata to file
func (s *UploadService) saveMetadata() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Convert map to slice
	uploads := make([]*UploadResult, 0, len(s.uploads))
	for _, upload := range s.uploads {
		uploads = append(uploads, upload)
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(uploads, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(s.metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	
	return nil
}

// UploadFile handles file upload and storage
func (s *UploadService) UploadFile(ctx context.Context, file *multipart.FileHeader) (*UploadResult, error) {
	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Read file data
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// Calculate SHA256 hash for the file
	hash := sha256.New()
	hash.Write(data)
	hashBytes := hash.Sum(nil)

	// Create multihash digest
	digest, err := multihash.Encode(hashBytes, multihash.SHA2_256)
	if err != nil {
		return nil, fmt.Errorf("failed to create multihash: %w", err)
	}

	// Store file in blob store
	err = s.blobStore.Put(ctx, digest, uint64(len(data)), io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to store file in blob store: %w", err)
	}

	// Create result with actual CID
	cidStr, err := multibase.Encode(multibase.Base58BTC, digest)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CID: %w", err)
	}

	result := &UploadResult{
		CID:        cidStr,
		Filename:   file.Filename,
		Size:       file.Size,
		UploadedAt: time.Now(),
	}

	// Track the uploaded file (with persistence)
	s.mutex.Lock()
	s.uploads[cidStr] = result
	s.mutex.Unlock()
	
	// Save metadata to disk
	if err := s.saveMetadata(); err != nil {
		// Log error but don't fail the upload
		fmt.Printf("Warning: failed to save metadata: %v\n", err)
	}

	return result, nil
}

// GetFile retrieves a file by CID
func (s *UploadService) GetFile(ctx context.Context, cidStr string) (io.ReadCloser, error) {
	// TODO: Implement file retrieval from Piri blob store
	return nil, fmt.Errorf("not implemented yet")
}

// ListFiles lists all uploaded files
func (s *UploadService) ListFiles(ctx context.Context) ([]*UploadResult, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	files := make([]*UploadResult, 0, len(s.uploads))
	for _, file := range s.uploads {
		files = append(files, file)
	}
	return files, nil
}
