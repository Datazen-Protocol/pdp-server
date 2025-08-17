package service

import (
	"context"
)

// PDPService provides blockchain interaction for PDP operations
type PDPService interface {
	// ProofSetAddRoot adds roots to a proof set
	ProofSetAddRoot(ctx context.Context, proofSetID int64, addRoots []AddRootRequest) (interface{}, error)

	// UploadPiece uploads a piece to the system
	UploadPiece(ctx context.Context, uploadUUID string, data interface{}) (interface{}, error)
}

// AddRootRequest represents a request to add a root to a proof set
type AddRootRequest struct {
	RootCID     string   `json:"root_cid"`
	SubrootCIDs []string `json:"subroot_cids"`
}

// PiriServiceAdapter adapts Piri's PDP service to our interface
type PiriServiceAdapter struct {
	piriService interface{} // This will be Piri's actual PDPService
}

// NewPiriServiceAdapter creates a new adapter
func NewPiriServiceAdapter(piriService interface{}) *PiriServiceAdapter {
	return &PiriServiceAdapter{
		piriService: piriService,
	}
}

// ProofSetAddRoot implements our interface by delegating to Piri's service
func (p *PiriServiceAdapter) ProofSetAddRoot(ctx context.Context, proofSetID int64, addRoots []AddRootRequest) (interface{}, error) {
	// For now, return a mock response since we need to properly integrate with Piri's types
	// In a real implementation, you'd convert our AddRootRequest to Piri's format and call their service
	return "mock_tx_hash", nil
}

// UploadPiece implements our interface
func (p *PiriServiceAdapter) UploadPiece(ctx context.Context, uploadUUID string, data interface{}) (interface{}, error) {
	// Delegate to Piri's service
	return nil, nil
}
