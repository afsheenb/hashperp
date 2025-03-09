package storage

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Updated DBSwapOffer struct to support additional fields
type DBSwapOffer struct {
	ID              string          `gorm:"primary_key;type:uuid"`
	OfferorID       string          `gorm:"type:uuid;not null;index"`
	VTXOID          string          `gorm:"type:uuid;not null;index"`
	ContractID      string          `gorm:"type:uuid;not null;index"`
	OfferedRate     float64         `gorm:"type:decimal(18,8);not null"`
	CreationTime    time.Time       `gorm:"not null"`
	ExpiryTime      time.Time       `gorm:"not null"`
	Status          string          `gorm:"type:varchar(20);not null"`
	AcceptorID      sql.NullString  `gorm:"type:uuid"`
	TargetUserID    sql.NullString  `gorm:"type:uuid"`
	SwapType        sql.NullString  `gorm:"type:varchar(20)"`
	RelatedEntities json.RawMessage `gorm:"type:jsonb"`
	CreatedAt       time.Time       `gorm:"not null"`
	UpdatedAt       time.Time       `gorm:"not null"`
}

// TableName sets the table name for DBSwapOffer
func (DBSwapOffer) TableName() string {
	return "swap_offers"
}

// Add missing fields to the DBVTXO struct for rollover functionality
type DBVTXO struct {
	ID                string         `gorm:"primary_key;type:uuid"`
	ContractID        string         `gorm:"type:uuid;not null;index"`
	OwnerID           string         `gorm:"type:uuid;not null;index"`
	Amount            float64        `gorm:"type:decimal(18,8);not null"`
	ScriptPath        string         `gorm:"type:text;not null"`
	CreationTimestamp time.Time      `gorm:"not null"`
	SignatureData     []byte         `gorm:"type:bytea"`
	SwappedFromID     sql.NullString `gorm:"type:uuid"`
	RolledFromID      sql.NullString `gorm:"type:uuid"`
	RolledToID        sql.NullString `gorm:"type:uuid"`
	IsActive          bool           `gorm:"not null;default:true"`
	ExitTxHash        sql.NullString `gorm:"type:varchar(100)"`
	ExitTimestamp     sql.NullTime   `gorm:"type:timestamp"`
	CreatedAt         time.Time      `gorm:"not null"`
	UpdatedAt         time.Time      `gorm:"not null"`
}

// TableName sets the table name for DBVTXO
func (DBVTXO) TableName() string {
	return "vtxos"
}

// Updated DBContract struct to support additional fields
type DBContract struct {
	ID                  string         `gorm:"primary_key;type:uuid"`
	ContractType        string         `gorm:"type:varchar(10);not null"`
	StrikeRate          float64        `gorm:"type:decimal(18,8);not null"`
	ExpiryBlockHeight   uint64         `gorm:"not null"`
	ExpiryDate          time.Time      `gorm:"not null"`
	CreationTime        time.Time      `gorm:"not null"`
	Status              string         `gorm:"type:varchar(20);not null"`
	BuyerID             string         `gorm:"type:uuid;not null"`
	SellerID            string         `gorm:"type:uuid;not null"`
	Size                float64        `gorm:"type:decimal(18,8);not null"`
	BuyerVTXO           string         `gorm:"type:uuid"`
	SellerVTXO          string         `gorm:"type:uuid"`
	SettlementTx        sql.NullString `gorm:"type:varchar(100)"`
	SettlementRate      sql.NullFloat64 `gorm:"type:decimal(18,8)"`
	RolledOverToID      sql.NullString `gorm:"type:uuid"`
	CompletionTimestamp sql.NullTime   `gorm:"type:timestamp"`
	BuyerExited         bool           `gorm:"not null;default:false"`
	SellerExited        bool           `gorm:"not null;default:false"`
	BuyerExitTxHash     sql.NullString `gorm:"type:varchar(100)"`
	SellerExitTxHash    sql.NullString `gorm:"type:varchar(100)"`
	CreatedAt           time.Time      `gorm:"not null"`
	UpdatedAt           time.Time      `gorm:"not null"`
}

// TableName sets the table name for DBContract
func (DBContract) TableName() string {
	return "contracts"
}

// Enhanced convertDBVTXOToVTXO to support all fields
func convertDBVTXOToVTXO(dbVTXO *DBVTXO) *hashperp.VTXO {
	vtxo := &hashperp.VTXO{
		ID:                dbVTXO.ID,
		ContractID:        dbVTXO.ContractID,
		OwnerID:           dbVTXO.OwnerID,
		Amount:            dbVTXO.Amount,
		ScriptPath:        dbVTXO.ScriptPath,
		CreationTimestamp: dbVTXO.CreationTimestamp,
		SignatureData:     dbVTXO.SignatureData,
		IsActive:          dbVTXO.IsActive,
	}

	if dbVTXO.SwappedFromID.Valid {
		vtxo.SwappedFromID = dbVTXO.SwappedFromID.String
	}
	
	if dbVTXO.RolledFromID.Valid {
		vtxo.RolledFromID = dbVTXO.RolledFromID.String
	}
	
	if dbVTXO.RolledToID.Valid {
		vtxo.RolledToID = dbVTXO.RolledToID.String
	}
	
	if dbVTXO.ExitTxHash.Valid {
		vtxo.ExitTxHash = dbVTXO.ExitTxHash.String
	}
	
	if dbVTXO.ExitTimestamp.Valid {
		vtxo.ExitTimestamp = dbVTXO.ExitTimestamp.Time
	}

	return vtxo
}

// Enhanced convertDBContractToContract to support all fields
func convertDBContractToContract(dbContract *DBContract) *hashperp.Contract {
	contract := &hashperp.Contract{
		ID:                dbContract.ID,
		ContractType:      hashperp.ContractType(dbContract.ContractType),
		StrikeRate:        dbContract.StrikeRate,
		ExpiryBlockHeight: dbContract.ExpiryBlockHeight,
		ExpiryDate:        dbContract.ExpiryDate,
		CreationTime:      dbContract.CreationTime,
		Status:            hashperp.ContractStatus(dbContract.Status),
		BuyerID:           dbContract.BuyerID,
		SellerID:          dbContract.SellerID,
		Size:              dbContract.Size,
		BuyerVTXO:         dbContract.BuyerVTXO,
		SellerVTXO:        dbContract.SellerVTXO,
		BuyerExited:       dbContract.BuyerExited,
		SellerExited:      dbContract.SellerExited,
	}

	if dbContract.SettlementTx.Valid {
		contract.SettlementTx = dbContract.SettlementTx.String
	}

	if dbContract.SettlementRate.Valid {
		contract.SettlementRate = dbContract.SettlementRate.Float64
	}

	if dbContract.RolledOverToID.Valid {
		contract.RolledOverToID = dbContract.RolledOverToID.String
	}
	
	if dbContract.CompletionTimestamp.Valid {
		contract.CompletionTimestamp = dbContract.CompletionTimestamp.Time
	}
	
	if dbContract.BuyerExitTxHash.Valid {
		contract.BuyerExitTxHash = dbContract.BuyerExitTxHash.String
	}
	
	if dbContract.SellerExitTxHash.Valid {
		contract.SellerExitTxHash = dbContract.SellerExitTxHash.String
	}

	return contract
}
