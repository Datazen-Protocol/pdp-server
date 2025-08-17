package models

import (
	"time"

	"gorm.io/datatypes"
)

// ParkedPiece represents a piece that has been parked in the system
type ParkedPiece struct {
	ID              uint   `gorm:"primaryKey"`
	PieceCID        string `gorm:"uniqueIndex;not null"`
	PiecePaddedSize int64  `gorm:"not null"`
	PieceRawSize    int64  `gorm:"not null"`
	Complete        bool   `gorm:"not null;default:false"`
	LongTerm        bool   `gorm:"not null;default:false"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ParkedPieceRef represents a reference to a parked piece
type ParkedPieceRef struct {
	RefID       uint           `gorm:"primaryKey"`
	PieceID     uint           `gorm:"not null"`
	DataURL     string         `gorm:"not null"`
	DataHeaders datatypes.JSON `gorm:"not null"`
	LongTerm    bool           `gorm:"not null;default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PDPPieceRef represents a piece reference in the PDP system
type PDPPieceRef struct {
	ID               uint   `gorm:"primaryKey"`
	Service          string `gorm:"not null"`
	PieceCID         string `gorm:"uniqueIndex;not null"`
	PieceRef         uint   `gorm:"not null"`
	ProofsetRefcount int    `gorm:"not null;default:0"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// MessageWaitsEth represents an Ethereum transaction waiting for confirmation
type MessageWaitsEth struct {
	ID                   uint   `gorm:"primaryKey"`
	SignedTxHash         string `gorm:"uniqueIndex;not null"`
	ConfirmedTxHash      string
	TxStatus             string `gorm:"not null;default:'pending'"`
	TxSuccess            *bool
	ConfirmedBlockNumber *int64
	ConfirmedTxData      []byte
	TxReceipt            []byte
	WaiterMachineID      *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// PDPProofSet represents a proof set in our system
type PDPProofSet struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string
	Status      string `gorm:"not null;default:'created'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PDPProofSetCreate tracks proof set creation requests
type PDPProofSetCreate struct {
	ID                uint   `gorm:"primaryKey"`
	ProofSetID        uint   `gorm:"not null"`
	CreateMessageHash string `gorm:"uniqueIndex;not null"`
	Service           string `gorm:"not null;default:'pdp-server'"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
