// Create hashperp/presigned_exit.go

package hashperp

import (
	"context"
	"time"
)

// PreSignedExit represents a pre-signed exit transaction
type PreSignedExit struct {
	ID           string    `json:"id"`
	VTXOID       string    `json:"vtxo_id"`
	ContractID   string    `json:"contract_id"`
	UserID       string    `json:"user_id"`
	ExitTxHex    string    `json:"exit_tx_hex"`
	CreationTime time.Time `json:"creation_time"`
	IsUsed       bool      `json:"is_used"`
	UsedTime     *time.Time `json:"used_time,omitempty"`
}

// PreSignedExitRepository defines the data access interface for pre-signed exits
type PreSignedExitRepository interface {
	// Create creates a new pre-signed exit
	Create(ctx context.Context, preSignedExit *PreSignedExit) error
	
	// FindByID retrieves a pre-signed exit by ID
	FindByID(ctx context.Context, id string) (*PreSignedExit, error)
	
	// FindByVTXO retrieves pre-signed exits for a VTXO
	FindByVTXO(ctx context.Context, vtxoID string) ([]*PreSignedExit, error)
	
	// MarkAsUsed marks a pre-signed exit as used
	MarkAsUsed(ctx context.Context, id string) error
}
