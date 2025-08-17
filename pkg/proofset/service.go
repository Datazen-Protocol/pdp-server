package proofset

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"

	"github.com/storacha/piri/pkg/pdp/service"
	"github.com/storacha/piri/pkg/pdp/service/models"
)

// ProofSetService wraps Piri's PDPService to provide proof set management
type ProofSetService struct {
	piriService *service.PDPService
	db          *gorm.DB
	address     common.Address
}

// ProofSetInfo represents a proof set with its status
type ProofSetInfo struct {
	ID                    int64     `json:"id"`
	CreateMessageHash     string    `json:"create_message_hash"`
	CreatedAt             time.Time `json:"created_at"`
	ProofSetCreated       bool      `json:"proof_set_created"`
	InitReady             bool      `json:"init_ready"`
	ChallengeRequestEpoch *int64    `json:"challenge_request_epoch,omitempty"`
	ProveAtEpoch          *int64    `json:"prove_at_epoch,omitempty"`
	Status                string    `json:"status"`
}

// AddRootRequest represents a request to add roots to a proof set
type AddRootRequest struct {
	RootCID     string   `json:"root_cid" validate:"required"`
	SubrootCIDs []string `json:"subroot_cids" validate:"required,min=1"`
}

// NewProofSetService creates a new proof set service
func NewProofSetService(piriService *service.PDPService, db *gorm.DB, address common.Address) *ProofSetService {
	return &ProofSetService{
		piriService: piriService,
		db:          db,
		address:     address,
	}
}

// CreateProofSet creates a new proof set
func (p *ProofSetService) CreateProofSet(ctx context.Context) (*ProofSetInfo, error) {
	// Use the record keeper address (same as in your piri command)
	recordKeeper := common.HexToAddress("0x6170dE2b09b404776197485F3dc6c968Ef948505")

	// Use Piri's service to create the proof set
	txHash, err := p.piriService.ProofSetCreate(ctx, recordKeeper)
	if err != nil {
		return nil, fmt.Errorf("failed to create proof set: %w", err)
	}

	// Get the created proof set from database
	var proofSet models.PDPProofsetCreate
	if err := p.db.WithContext(ctx).
		Where("create_message_hash = ?", txHash.Hex()).
		First(&proofSet).Error; err != nil {
		return nil, fmt.Errorf("failed to get proof set from database: %w", err)
	}

	// Try to get the actual proof set ID from PDPProofSet table
	var pdpProofSet models.PDPProofSet
	var proofSetID int64 = 0
	if err := p.db.WithContext(ctx).
		Where("create_message_hash = ?", txHash.Hex()).
		First(&pdpProofSet).Error; err == nil {
		proofSetID = pdpProofSet.ID
	}

	return &ProofSetInfo{
		ID:                proofSetID,
		CreateMessageHash: proofSet.CreateMessageHash,
		CreatedAt:         proofSet.CreatedAt,
		ProofSetCreated:   proofSet.ProofsetCreated,
		Status:            "pending",
	}, nil
}

// ListProofSets lists all proof sets
func (p *ProofSetService) ListProofSets(ctx context.Context) ([]*ProofSetInfo, error) {
	var proofSets []models.PDPProofsetCreate
	if err := p.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&proofSets).Error; err != nil {
		return nil, fmt.Errorf("failed to list proof sets: %w", err)
	}

	result := make([]*ProofSetInfo, len(proofSets))
	for i, ps := range proofSets {
		status := "pending"
		if ps.ProofsetCreated {
			status = "created"
		}

		// Try to get the actual proof set ID from PDPProofSet table
		var pdpProofSet models.PDPProofSet
		var proofSetID int64 = 0
		if err := p.db.WithContext(ctx).
			Where("create_message_hash = ?", ps.CreateMessageHash).
			First(&pdpProofSet).Error; err == nil {
			proofSetID = pdpProofSet.ID
		}

		result[i] = &ProofSetInfo{
			ID:                proofSetID,
			CreateMessageHash: ps.CreateMessageHash,
			CreatedAt:         ps.CreatedAt,
			ProofSetCreated:   ps.ProofsetCreated,
			Status:            status,
		}
	}

	return result, nil
}

// GetProofSet gets a specific proof set by message hash
func (p *ProofSetService) GetProofSet(ctx context.Context, messageHash string) (*ProofSetInfo, error) {
	var proofSet models.PDPProofsetCreate
	if err := p.db.WithContext(ctx).
		Where("create_message_hash = ?", messageHash).
		First(&proofSet).Error; err != nil {
		return nil, fmt.Errorf("proof set not found: %w", err)
	}

	// Get additional status information from PDPProofSet table
	var pdpProofSet models.PDPProofSet
	if err := p.db.WithContext(ctx).
		Where("create_message_hash = ?", messageHash).
		First(&pdpProofSet).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get proof set status: %w", err)
	}

	status := "pending"
	if proofSet.ProofsetCreated {
		status = "created"
	}

	return &ProofSetInfo{
		ID:                    pdpProofSet.ID,
		CreateMessageHash:     proofSet.CreateMessageHash,
		CreatedAt:             proofSet.CreatedAt,
		ProofSetCreated:       proofSet.ProofsetCreated,
		InitReady:             pdpProofSet.InitReady,
		ChallengeRequestEpoch: pdpProofSet.ChallengeRequestTaskID, // Using TaskID as epoch
		ProveAtEpoch:          pdpProofSet.ProveAtEpoch,
		Status:                status,
	}, nil
}

// GetProofSetByID gets a specific proof set by ID
func (p *ProofSetService) GetProofSetByID(ctx context.Context, proofSetID int64) (*ProofSetInfo, error) {
	// First try to get from PDPProofSet table
	var pdpProofSet models.PDPProofSet
	if err := p.db.WithContext(ctx).
		Where("id = ?", proofSetID).
		First(&pdpProofSet).Error; err != nil {
		return nil, fmt.Errorf("proof set not found: %w", err)
	}

	// Get the creation record from PDPProofsetCreate table
	var proofSet models.PDPProofsetCreate
	if err := p.db.WithContext(ctx).
		Where("create_message_hash = ?", pdpProofSet.CreateMessageHash).
		First(&proofSet).Error; err != nil {
		return nil, fmt.Errorf("proof set creation record not found: %w", err)
	}

	status := "pending"
	if proofSet.ProofsetCreated {
		status = "created"
	}

	return &ProofSetInfo{
		ID:                    pdpProofSet.ID,
		CreateMessageHash:     proofSet.CreateMessageHash,
		CreatedAt:             proofSet.CreatedAt,
		ProofSetCreated:       proofSet.ProofsetCreated,
		InitReady:             pdpProofSet.InitReady,
		ChallengeRequestEpoch: pdpProofSet.ChallengeRequestTaskID, // Using TaskID as epoch
		ProveAtEpoch:          pdpProofSet.ProveAtEpoch,
		Status:                status,
	}, nil
}

// AddRootsToProofSet adds roots to a proof set
func (p *ProofSetService) AddRootsToProofSet(ctx context.Context, proofSetID int64, requests []AddRootRequest) error {
	// Convert our requests to Piri's format
	piriRequests := make([]service.AddRootRequest, len(requests))
	for i, req := range requests {
		piriRequests[i] = service.AddRootRequest{
			RootCID:     req.RootCID,
			SubrootCIDs: req.SubrootCIDs,
		}
	}

	// Use Piri's service to add roots
	_, err := p.piriService.ProofSetAddRoot(ctx, proofSetID, piriRequests)
	if err != nil {
		return fmt.Errorf("failed to add roots to proof set: %w", err)
	}

	return nil
}

// GetProofSetRoots gets the roots for a proof set
func (p *ProofSetService) GetProofSetRoots(ctx context.Context, proofSetID int64) ([]map[string]interface{}, error) {
	var rootAdds []models.PDPProofsetRootAdd
	if err := p.db.WithContext(ctx).
		Where("proofset_id = ?", proofSetID).
		Order("add_message_hash ASC").
		Find(&rootAdds).Error; err != nil {
		return nil, fmt.Errorf("failed to get proof set roots: %w", err)
	}

	result := make([]map[string]interface{}, len(rootAdds))
	for i, rootAdd := range rootAdds {
		result[i] = map[string]interface{}{
			"proofset_id":       rootAdd.ProofsetID,
			"root":              rootAdd.Root,
			"subroot":           rootAdd.Subroot,
			"subroot_offset":    rootAdd.SubrootOffset,
			"subroot_size":      rootAdd.SubrootSize,
			"add_message_hash":  rootAdd.AddMessageHash,
			"add_message_index": rootAdd.AddMessageIndex,
		}
	}

	return result, nil
}

// GetProofSetStatus gets detailed status of a proof set
func (p *ProofSetService) GetProofSetStatus(ctx context.Context, proofSetID int64) (map[string]interface{}, error) {
	// Get proof set info from PDPProofSet table
	var pdpProofSet models.PDPProofSet
	if err := p.db.WithContext(ctx).
		Where("id = ?", proofSetID).
		First(&pdpProofSet).Error; err != nil {
		return nil, fmt.Errorf("proof set not found: %w", err)
	}

	// Get create info
	var proofSetCreate models.PDPProofsetCreate
	if err := p.db.WithContext(ctx).
		Where("create_message_hash = ?", pdpProofSet.CreateMessageHash).
		First(&proofSetCreate).Error; err != nil {
		return nil, fmt.Errorf("failed to get proof set create info: %w", err)
	}

	// Get roots
	roots, err := p.GetProofSetRoots(ctx, proofSetID)
	if err != nil {
		return nil, err
	}

	// Get transaction status
	var messageWait models.MessageWaitsEth
	if err := p.db.WithContext(ctx).
		Where("signed_tx_hash = ?", pdpProofSet.CreateMessageHash).
		First(&messageWait).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get transaction status: %w", err)
	}

	status := "pending"
	if proofSetCreate.ProofsetCreated {
		status = "created"
	}

	return map[string]interface{}{
		"proof_set": map[string]interface{}{
			"id":                      pdpProofSet.ID,
			"create_message_hash":     pdpProofSet.CreateMessageHash,
			"init_ready":              pdpProofSet.InitReady,
			"challenge_request_epoch": pdpProofSet.ChallengeRequestTaskID,
			"prove_at_epoch":          pdpProofSet.ProveAtEpoch,
			"status":                  status,
		},
		"roots":     roots,
		"tx_status": messageWait.TxStatus,
	}, nil
}
