// package hashperp provides the core backend functionality for the HashPerp platform,
// a decentralized Bitcoin hash rate derivatives trading system built on the Ark layer-2 protocol.
package hashperp

import (
	"context"
	"time"
)

// HashRateData represents Bitcoin network hash rate information used for pricing and settlement
type HashRateData struct {
	Timestamp     time.Time `json:"timestamp"`
	BlockHeight   uint64    `json:"block_height"`
	HashRate      float64   `json:"hash_rate"`      // Current hash rate in PH/s
	BTCPerPHPerDay float64   `json:"btc_ph_day"`    // BTC per PetaHash per Day rate
}

// ContractType defines whether a contract is a CALL or PUT option
type ContractType string

const (
	CALL ContractType = "CALL" // Buyer profits when hash rate increases
	PUT  ContractType = "PUT"  // Buyer profits when hash rate decreases
)

// ContractStatus represents the lifecycle state of a contract
type ContractStatus string

const (
	PENDING    ContractStatus = "PENDING"    // Contract created but not yet active
	ACTIVE     ContractStatus = "ACTIVE"     // Contract is currently active
	SETTLED    ContractStatus = "SETTLED"    // Contract has been settled
	EXITED     ContractStatus = "EXITED"     // Contract was exited before expiration
	ROLLED_OVER ContractStatus = "ROLLED_OVER" // Contract was rolled over to a new contract
)

// Contract represents a hash rate perpetual futures contract
type Contract struct {
	ID               string        `json:"id"`
	ContractType     ContractType  `json:"contract_type"`
	StrikeRate       float64       `json:"strike_rate"`      // Strike rate in BTC/PH/day
	ExpiryBlockHeight uint64        `json:"expiry_block_height"`
	ExpiryDate       time.Time     `json:"expiry_date"`     // Human-readable expiry date
	CreationTime     time.Time     `json:"creation_time"`
	Status           ContractStatus`json:"status"`
	BuyerID          string        `json:"buyer_id"`
	SellerID         string        `json:"seller_id"`
	Size             float64       `json:"size"`            // Contract size in BTC
	BuyerVTXO        string        `json:"buyer_vtxo"`      // VTXO identifier for buyer
	SellerVTXO       string        `json:"seller_vtxo"`     // VTXO identifier for seller
	SettlementTx     string        `json:"settlement_tx,omitempty"` // Settlement transaction ID if settled
	SettlementRate   float64       `json:"settlement_rate,omitempty"` // Settlement rate at contract completion
	RolledOverToID   string        `json:"rolled_over_to_id,omitempty"` // ID of contract this rolled into
}

// VTXO represents a Virtual Transaction Output used in the contract system
type VTXO struct {
	ID                string    `json:"id"`
	ContractID        string    `json:"contract_id"`
	OwnerID           string    `json:"owner_id"`
	Amount            float64   `json:"amount"`            // Amount in BTC
	ScriptPath        string    `json:"script_path"`       // The Taproot script path
	CreationTimestamp time.Time `json:"creation_timestamp"`
	SignatureData     []byte    `json:"signature_data"`    // Signature data for the VTXO
	SwappedFromID     string    `json:"swapped_from_id,omitempty"` // Previous VTXO ID if swapped
	IsActive          bool      `json:"is_active"`
}

// Order represents a buy or sell order in the order book
type OrderType string

const (
	BUY  OrderType = "BUY"
	SELL OrderType = "SELL"
)

// OrderStatus represents the current status of an order
type OrderStatus string

const (
	OPEN     OrderStatus = "OPEN"
	MATCHED  OrderStatus = "MATCHED"
	CANCELED OrderStatus = "CANCELED"
	EXPIRED  OrderStatus = "EXPIRED"
)

// Order represents an order in the orderbook
type Order struct {
	ID               string      `json:"id"`
	UserID           string      `json:"user_id"`
	OrderType        OrderType   `json:"order_type"`
	ContractType     ContractType`json:"contract_type"`
	StrikeRate       float64     `json:"strike_rate"`      // Strike rate in BTC/PH/day
	ExpiryBlockHeight uint64      `json:"expiry_block_height,omitempty"`
	ExpiryDate       time.Time   `json:"expiry_date,omitempty"` // Human-readable expiry for UI
	Size             float64     `json:"size"`             // Size in BTC
	Status           OrderStatus `json:"status"`
	CreationTime     time.Time   `json:"creation_time"`
	MatchedOrderID   string      `json:"matched_order_id,omitempty"`
	ResultingContractID string    `json:"resulting_contract_id,omitempty"`
}

// SwapOffer represents an offer to swap a VTXO with another user
type SwapOffer struct {
	ID           string    `json:"id"`
	OfferorID    string    `json:"offeror_id"`     // User offering the swap
	VTXOID       string    `json:"vtxo_id"`        // VTXO being offered
	ContractID   string    `json:"contract_id"`
	OfferedRate  float64   `json:"offered_rate"`   // Rate at which swap is offered
	CreationTime time.Time `json:"creation_time"`
	ExpiryTime   time.Time `json:"expiry_time"`
	Status       string    `json:"status"`         // "OPEN", "ACCEPTED", "EXPIRED", "CANCELED"
	AcceptorID   string    `json:"acceptor_id,omitempty"` // User who accepted the offer
}

// Transaction represents a record of all on-chain and off-chain transactions
type TransactionType string

const (
	CONTRACT_CREATION TransactionType = "CONTRACT_CREATION"
	CONTRACT_SETTLEMENT TransactionType = "CONTRACT_SETTLEMENT"
	VTXO_SWAP TransactionType = "VTXO_SWAP"
	CONTRACT_ROLLOVER TransactionType = "CONTRACT_ROLLOVER"
	EXIT_PATH_EXECUTION TransactionType = "EXIT_PATH_EXECUTION"
)

// Transaction represents a transaction in the system
type Transaction struct {
	ID              string         `json:"id"`
	Type            TransactionType`json:"type"`
	Timestamp       time.Time      `json:"timestamp"`
	ContractID      string         `json:"contract_id,omitempty"`
	UserIDs         []string       `json:"user_ids"`
	TxHash          string         `json:"tx_hash,omitempty"`    // On-chain transaction hash, if applicable
	Amount          float64        `json:"amount"`              // Amount in BTC
	BTCPerPHPerDay  float64        `json:"btc_ph_day,omitempty"`// Rate at transaction time
	BlockHeight     uint64         `json:"block_height,omitempty"`
	RelatedEntities map[string]string `json:"related_entities,omitempty"` // Related VTXOs, contracts, etc.
}

// =============================================================================
// CONTRACT MANAGEMENT API INTERFACES
// =============================================================================

// ContractManager handles the lifecycle of contracts
type ContractManager interface {
	// CreateContract creates a new contract between two parties
	CreateContract(ctx context.Context, buyerID, sellerID string, contractType ContractType, 
		strikeRate float64, expiryBlockHeight uint64, size float64) (*Contract, error)
	
	// GetContract retrieves a contract by ID
	GetContract(ctx context.Context, contractID string) (*Contract, error)
	
	// GetContractsByUser retrieves all contracts for a specific user
	GetContractsByUser(ctx context.Context, userID string, status []ContractStatus) ([]*Contract, error)
	
	// SettleContract settles a contract based on the current hash rate data
	SettleContract(ctx context.Context, contractID string) (*Transaction, error)
	
	// ExitContract allows a user to exit a contract before expiration
	ExitContract(ctx context.Context, contractID string, userID string) (*Transaction, error)
	
	// RolloverContract rolls over a contract to a new expiration
	RolloverContract(ctx context.Context, contractID string, newExpiryBlockHeight uint64) (*Contract, *Transaction, error)
	
	// ExecuteExitPath handles non-cooperative settlement via an exit path
	ExecuteExitPath(ctx context.Context, contractID string, userID string, exitPathType string) (*Transaction, error)
}

// =============================================================================
// VTXO MANAGEMENT API INTERFACES
// =============================================================================

// VTXOManager handles the creation and management of VTXOs
type VTXOManager interface {
	// CreateVTXO creates a new VTXO for a contract
	CreateVTXO(ctx context.Context, contractID string, ownerID string, amount float64, scriptPath string, 
		signatureData []byte) (*VTXO, error)
	
	// GetVTXO retrieves a VTXO by ID
	GetVTXO(ctx context.Context, vtxoID string) (*VTXO, error)
	
	// GetVTXOsByContract retrieves all VTXOs for a specific contract
	GetVTXOsByContract(ctx context.Context, contractID string) ([]*VTXO, error)
	
	// GetVTXOsByUser retrieves all VTXOs for a specific user
	GetVTXOsByUser(ctx context.Context, userID string, onlyActive bool) ([]*VTXO, error)
	
	// SwapVTXO swaps a VTXO between two users (off-chain)
	SwapVTXO(ctx context.Context, vtxoID string, newOwnerID string, newSignatureData []byte) (*VTXO, *Transaction, error)
	
	// CreatePresignedExitTransaction creates a pre-signed exit transaction for a VTXO
	CreatePresignedExitTransaction(ctx context.Context, vtxoID string, signatureData []byte) (string, error)
	
	// ExecuteVTXOSweep executes a VTXO sweep in case of failure or non-cooperation
	ExecuteVTXOSweep(ctx context.Context, vtxoID string) (*Transaction, error)
}

// =============================================================================
// ORDER BOOK API INTERFACES
// =============================================================================

// OrderBookManager handles the order book functionality
type OrderBookManager interface {
	// PlaceOrder places a new order in the order book
	PlaceOrder(ctx context.Context, userID string, orderType OrderType, contractType ContractType, 
		strikeRate float64, expiryBlockHeight uint64, size float64) (*Order, error)
	
	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string, userID string) error
	
	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	
	// GetOrdersByUser retrieves all orders for a specific user
	GetOrdersByUser(ctx context.Context, userID string, status []OrderStatus) ([]*Order, error)
	
	// GetOrderBook retrieves the current order book for a given contract type and parameters
	GetOrderBook(ctx context.Context, contractType ContractType, expiryBlockHeight uint64) ([]*Order, error)
	
	// MatchOrders attempts to match buy and sell orders
	MatchOrders(ctx context.Context) ([]*Contract, error)
}

// =============================================================================
// SWAP OFFER API INTERFACES
// =============================================================================

// SwapOfferManager handles VTXO swap offers
type SwapOfferManager interface {
	// CreateSwapOffer creates a new swap offer
	CreateSwapOffer(ctx context.Context, offerorID string, vtxoID string, offeredRate float64, 
		expiryTime time.Time) (*SwapOffer, error)
	
	// AcceptSwapOffer accepts a swap offer
	AcceptSwapOffer(ctx context.Context, offerID string, acceptorID string) (*Transaction, error)
	
	// CancelSwapOffer cancels a swap offer
	CancelSwapOffer(ctx context.Context, offerID string, offerorID string) error
	
	// GetSwapOffer retrieves a swap offer by ID
	GetSwapOffer(ctx context.Context, offerID string) (*SwapOffer, error)
	
	// GetSwapOffersByUser retrieves all swap offers for a specific user
	GetSwapOffersByUser(ctx context.Context, userID string, isOfferor bool) ([]*SwapOffer, error)
	
	// GetSwapOffersByContract retrieves all swap offers for a specific contract
	GetSwapOffersByContract(ctx context.Context, contractID string) ([]*SwapOffer, error)
}

// =============================================================================
// MARKET DATA API INTERFACES
// =============================================================================

// MarketDataManager handles hash rate data retrieval and storage
type MarketDataManager interface {
	// GetCurrentHashRate retrieves the current Bitcoin hash rate data
	GetCurrentHashRate(ctx context.Context) (*HashRateData, error)
	
	// GetHistoricalHashRate retrieves historical hash rate data for a given time range
	GetHistoricalHashRate(ctx context.Context, startTime, endTime time.Time) ([]*HashRateData, error)
	
	// GetHashRateAtBlockHeight retrieves hash rate data at a specific block height
	GetHashRateAtBlockHeight(ctx context.Context, blockHeight uint64) (*HashRateData, error)
	
	// CalculateBTCPerPHPerDay calculates the BTC per PetaHash per Day rate
	CalculateBTCPerPHPerDay(ctx context.Context, hashRate float64, blockHeight uint64) (float64, error)
}

// =============================================================================
// TRANSACTION HISTORY API INTERFACES
// =============================================================================

// TransactionManager handles transaction recording and retrieval
type TransactionManager interface {
	// RecordTransaction records a new transaction
	RecordTransaction(ctx context.Context, transactionType TransactionType, contractID string, 
		userIDs []string, txHash string, amount float64, btcPerPHPerDay float64, 
		blockHeight uint64, relatedEntities map[string]string) (*Transaction, error)
	
	// GetTransaction retrieves a transaction by ID
	GetTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	
	// GetTransactionsByUser retrieves all transactions for a specific user
	GetTransactionsByUser(ctx context.Context, userID string, transactionTypes []TransactionType) ([]*Transaction, error)
	
	// GetTransactionsByContract retrieves all transactions for a specific contract
	GetTransactionsByContract(ctx context.Context, contractID string) ([]*Transaction, error)
}

// =============================================================================
// BITCOIN SCRIPT GENERATION API INTERFACES
// =============================================================================

// ScriptGenerator handles the generation of Bitcoin scripts for contracts
type ScriptGenerator interface {
	// GenerateContractScripts generates all necessary scripts for a contract
	GenerateContractScripts(ctx context.Context, contract *Contract) (map[string]string, error)
	
	// GenerateSetupTransaction generates the setup transaction script for a contract
	GenerateSetupTransaction(ctx context.Context, contract *Contract, buyerVTXO, sellerVTXO *VTXO) (string, error)
	
	// GenerateFinalTransaction generates the final transaction script for a contract
	GenerateFinalTransaction(ctx context.Context, contract *Contract, setupTxID string) (string, error)
	
	// GenerateSettlementTransaction generates the settlement transaction script for a contract
	GenerateSettlementTransaction(ctx context.Context, contract *Contract, finalTxID string, winnerID string) (string, error)
	
	// GenerateExitPathScripts generates scripts for all exit paths
	GenerateExitPathScripts(ctx context.Context, contract *Contract) (map[string]string, error)
}

// =============================================================================
// SERVICE PROVIDER API INTERFACE
// =============================================================================

// HashPerpService is the main entry point for the HashPerp backend API
type HashPerpService interface {
	ContractManager
	VTXOManager
	OrderBookManager
	SwapOfferManager
	MarketDataManager
	TransactionManager
	ScriptGenerator
	
	// Healthcheck verifies the service is functioning properly
	Healthcheck(ctx context.Context) error
	
	// GetCurrentBlockHeight retrieves the current Bitcoin block height
	GetCurrentBlockHeight(ctx context.Context) (uint64, error)
	
	// ValidateContractParameters validates that contract parameters are valid
	ValidateContractParameters(ctx context.Context, contractType ContractType, 
		strikeRate float64, expiryBlockHeight uint64, size float64) error
}
