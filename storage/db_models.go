package storage

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// DBContract is the database model for contracts
type DBContract struct {
	ID                  string          `gorm:"primary_key;type:uuid"`
	ContractType        string          `gorm:"type:varchar(10);not null"`
	StrikeRate          float64         `gorm:"type:decimal(18,8);not null"`
	ExpiryBlockHeight   uint64          `gorm:"not null"`
	ExpiryDate          time.Time       `gorm:"not null"`
	CreationTime        time.Time       `gorm:"not null"`
	Status              string          `gorm:"type:varchar(20);not null"`
	BuyerID             string          `gorm:"type:uuid;not null"`
	SellerID            string          `gorm:"type:uuid;not null"`
	Size                float64         `gorm:"type:decimal(18,8);not null"`
	BuyerVTXO           string          `gorm:"type:uuid"`
	SellerVTXO          string          `gorm:"type:uuid"`
	SettlementTx        sql.NullString  `gorm:"type:varchar(100)"`
	SettlementRate      sql.NullFloat64 `gorm:"type:decimal(18,8)"`
	RolledOverToID      sql.NullString  `gorm:"type:uuid"`
	CompletionTimestamp sql.NullTime    `gorm:"type:timestamp"`
	BuyerExited         bool            `gorm:"not null;default:false"`
	SellerExited        bool            `gorm:"not null;default:false"`
	BuyerExitTxHash     sql.NullString  `gorm:"type:varchar(100)"`
	SellerExitTxHash    sql.NullString  `gorm:"type:varchar(100)"`
	CreatedAt           time.Time       `gorm:"not null"`
	UpdatedAt           time.Time       `gorm:"not null"`
}

// TableName sets the table name for DBContract
func (DBContract) TableName() string {
	return "contracts"
}

// DBVTXO is the database model for VTXOs
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

// DBOrder is the database model for orders
type DBOrder struct {
	ID                  string         `gorm:"primary_key;type:uuid"`
	UserID              string         `gorm:"type:uuid;not null;index"`
	OrderType           string         `gorm:"type:varchar(10);not null"`
	ContractType        string         `gorm:"type:varchar(10);not null"`
	StrikeRate          float64        `gorm:"type:decimal(18,8);not null"`
	ExpiryBlockHeight   uint64         `gorm:"not null"`
	ExpiryDate          time.Time      `gorm:"not null"`
	Size                float64        `gorm:"type:decimal(18,8);not null"`
	Status              string         `gorm:"type:varchar(20);not null"`
	CreationTime        time.Time      `gorm:"not null"`
	MatchedOrderID      sql.NullString `gorm:"type:uuid"`
	ResultingContractID sql.NullString `gorm:"type:uuid"`
	CreatedAt           time.Time      `gorm:"not null"`
	UpdatedAt           time.Time      `gorm:"not null"`
}

// TableName sets the table name for DBOrder
func (DBOrder) TableName() string {
	return "orders"
}

// DBSwapOffer is the database model for swap offers
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

// DBTransaction is the database model for transactions
type DBTransaction struct {
	ID              string         `gorm:"primary_key;type:uuid"`
	Type            string         `gorm:"type:varchar(30);not null"`
	Timestamp       time.Time      `gorm:"not null"`
	ContractID      string         `gorm:"type:uuid;index"`
	UserIDs         pq.StringArray `gorm:"type:text[]"`
	TxHash          string         `gorm:"type:varchar(100)"`
	Amount          float64        `gorm:"type:decimal(18,8);not null"`
	BTCPerPHPerDay  float64        `gorm:"type:decimal(18,8)"`
	BlockHeight     uint64        
	Status          string         `gorm:"type:varchar(20);default:'COMPLETED'"`
	RelatedEntities json.RawMessage `gorm:"type:jsonb"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
}

// TableName sets the table name for DBTransaction
func (DBTransaction) TableName() string {
	return "transactions"
}

// DBHashRateData is the database model for hash rate data
type DBHashRateData struct {
	ID             uint64    `gorm:"primary_key;auto_increment"`
	Timestamp      time.Time `gorm:"not null;index"`
	BlockHeight    uint64    `gorm:"not null;unique_index"`
	HashRate       float64   `gorm:"type:decimal(18,8);not null"`
	BTCPerPHPerDay float64   `gorm:"type:decimal(18,8);not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

// TableName sets the table name for DBHashRateData
func (DBHashRateData) TableName() string {
	return "hash_rate_data"
}

// DBUser is the database model for users
type DBUser struct {
	ID        string    `gorm:"primary_key;type:uuid"`
	PublicKey []byte    `gorm:"type:bytea;not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName sets the table name for DBUser
func (DBUser) TableName() string {
	return "users"
}

// DBPreSignedExit is the database model for pre-signed exit transactions
type DBPreSignedExit struct {
	ID           string     `gorm:"primary_key;type:uuid"`
	VTXOID       string     `gorm:"type:uuid;not null;index"`
	ContractID   string     `gorm:"type:uuid;not null;index"`
	UserID       string     `gorm:"type:uuid;not null"`
	ExitTxHex    string     `gorm:"type:text;not null"`
	CreationTime time.Time  `gorm:"not null"`
	IsUsed       bool       `gorm:"not null;default:false"`
	UsedTime     *time.Time `gorm:""`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

// TableName sets the table name for DBPreSignedExit
func (DBPreSignedExit) TableName() string {
	return "pre_signed_exits"
}