// Create storage/presigned_exit_repo.go

package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashperp/hashperp"
	"gorm.io/gorm"
)

// PostgresPreSignedExitRepository implements the PreSignedExitRepository interface
type PostgresPreSignedExitRepository struct {
	db *gorm.DB
}

// NewPostgresPreSignedExitRepository creates a new PostgreSQL-based repository
func NewPostgresPreSignedExitRepository(db *gorm.DB) hashperp.PreSignedExitRepository {
	return &PostgresPreSignedExitRepository{
		db: db,
	}
}

// Create creates a new pre-signed exit
func (r *PostgresPreSignedExitRepository) Create(ctx context.Context, preSignedExit *hashperp.PreSignedExit) error {
	dbPreSignedExit := &DBPreSignedExit{
		ID:           preSignedExit.ID,
		VTXOID:       preSignedExit.VTXOID,
		ContractID:   preSignedExit.ContractID,
		UserID:       preSignedExit.UserID,
		ExitTxHex:    preSignedExit.ExitTxHex,
		CreationTime: preSignedExit.CreationTime,
		IsUsed:       preSignedExit.IsUsed,
	}
	
	if preSignedExit.UsedTime != nil {
		dbPreSignedExit.UsedTime = preSignedExit.UsedTime
	}
	
	result := r.db.WithContext(ctx).Create(dbPreSignedExit)
	if result.Error != nil {
		return fmt.Errorf("failed to create pre-signed exit: %w", result.Error)
	}
	
	return nil
}

// FindByID retrieves a pre-signed exit by ID
func (r *PostgresPreSignedExitRepository) FindByID(ctx context.Context, id string) (*hashperp.PreSignedExit, error) {
	var dbPreSignedExit DBPreSignedExit
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&dbPreSignedExit)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find pre-signed exit: %w", result.Error)
	}
	
	return convertDBPreSignedExitToPreSignedExit(&dbPreSignedExit), nil
}

// FindByVTXO retrieves all pre-signed exits for a VTXO
func (r *PostgresPreSignedExitRepository) FindByVTXO(ctx context.Context, vtxoID string) ([]*hashperp.PreSignedExit, error) {
	var dbPreSignedExits []DBPreSignedExit
	result := r.db.WithContext(ctx).Where("vtxo_id = ?", vtxoID).Find(&dbPreSignedExits)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find pre-signed exits for VTXO: %w", result.Error)
	}
	
	preSignedExits := make([]*hashperp.PreSignedExit, len(dbPreSignedExits))
	for i, dbPreSignedExit := range dbPreSignedExits {
		preSignedExits[i] = convertDBPreSignedExitToPreSignedExit(&dbPreSignedExit)
	}
	
	return preSignedExits, nil
}

// MarkAsUsed marks a pre-signed exit as used
func (r *PostgresPreSignedExitRepository) MarkAsUsed(ctx context.Context, id string) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&DBPreSignedExit{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_used":   true,
			"used_time": now,
		})
	
	if result.Error != nil {
		return fmt.Errorf("failed to mark pre-signed exit as used: %w", result.Error)
	}
	
	return nil
}
