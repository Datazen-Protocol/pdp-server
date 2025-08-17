package proofset

import (
	"context"
	"fmt"
	"time"

	"github.com/your-org/pdp-server/pkg/models"
	"gorm.io/gorm"
)

// SimpleProofSetService provides basic proof set management for our isolated database
type SimpleProofSetService struct {
	db *gorm.DB
}

// NewSimpleProofSetService creates a new simple proof set service
func NewSimpleProofSetService(db *gorm.DB) *SimpleProofSetService {
	return &SimpleProofSetService{
		db: db,
	}
}

// CreateProofSetRequest represents a request to create a proof set
type CreateProofSetRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

// ProofSetResponse represents a proof set response
type ProofSetResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateProofSet creates a new proof set in our isolated database
func (s *SimpleProofSetService) CreateProofSet(ctx context.Context, req *CreateProofSetRequest) (*ProofSetResponse, error) {
	proofSet := &models.PDPProofSet{
		Name:        req.Name,
		Description: req.Description,
		Status:      "created",
	}

	if err := s.db.WithContext(ctx).Create(proofSet).Error; err != nil {
		return nil, fmt.Errorf("failed to create proof set: %w", err)
	}

	return &ProofSetResponse{
		ID:          proofSet.ID,
		Name:        proofSet.Name,
		Description: proofSet.Description,
		Status:      proofSet.Status,
		CreatedAt:   proofSet.CreatedAt,
	}, nil
}

// GetProofSet retrieves a proof set by ID
func (s *SimpleProofSetService) GetProofSet(ctx context.Context, id uint) (*ProofSetResponse, error) {
	var proofSet models.PDPProofSet
	if err := s.db.WithContext(ctx).First(&proofSet, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("proof set not found")
		}
		return nil, fmt.Errorf("failed to get proof set: %w", err)
	}

	return &ProofSetResponse{
		ID:          proofSet.ID,
		Name:        proofSet.Name,
		Description: proofSet.Description,
		Status:      proofSet.Status,
		CreatedAt:   proofSet.CreatedAt,
	}, nil
}

// ListProofSets lists all proof sets
func (s *SimpleProofSetService) ListProofSets(ctx context.Context) ([]*ProofSetResponse, error) {
	var proofSets []models.PDPProofSet
	if err := s.db.WithContext(ctx).Find(&proofSets).Error; err != nil {
		return nil, fmt.Errorf("failed to list proof sets: %w", err)
	}

	responses := make([]*ProofSetResponse, len(proofSets))
	for i, ps := range proofSets {
		responses[i] = &ProofSetResponse{
			ID:          ps.ID,
			Name:        ps.Name,
			Description: ps.Description,
			Status:      ps.Status,
			CreatedAt:   ps.CreatedAt,
		}
	}

	return responses, nil
}
