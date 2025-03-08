package hashperp

import (
	"context"
	"time"
)

// BitcoinClient defines the interface for interacting with the Bitcoin network
type BitcoinClient interface {
	// GetCurrentBlockHeight returns the current block height of the Bitcoin network
	GetCurrentBlockHeight(ctx context.Context) (uint64, error)
	
	// GetBlockHashRate returns the estimated hash rate at a specific block height
	GetBlockHashRate(ctx context.Context, blockHeight uint64) (float64, error)
	
	// BroadcastTransaction broadcasts a raw transaction to the Bitcoin network
	BroadcastTransaction(ctx context.Context, txHex string) (string, error)
	
	// ValidateSignature validates a signature against a message and public key
	ValidateSignature(ctx context.Context, message []byte, signature []byte, pubKey []byte) (bool, error)
	
	// GetBlockByHeight retrieves block data for a specific height
	GetBlockByHeight(ctx context.Context, height uint64) (map[string]interface{}, error)
	
	// EstimateNetworkDifficulty estimates the current network difficulty
	EstimateNetworkDifficulty(ctx context.Context) (float64, error)
}

// ContractRepository defines the data access interface for contracts
type ContractRepository interface {
	// Create creates a new contract
	Create(ctx context.Context, contract *Contract) error
	
	// FindByID retrieves a contract by ID
	FindByID(ctx context.Context, id string) (*Contract, error)
	
	// FindByUser retrieves all contracts for a specific user
	FindByUser(ctx context.Context, userID string, status []ContractStatus) ([]*Contract, error)
	
	// FindActiveContracts retrieves all active contracts
	FindActiveContracts(ctx context.Context) ([]*Contract, error)
	
	// FindByExpiryRange retrieves contracts expiring within a certain block height range
	FindByExpiryRange(ctx context.Context, fromHeight, toHeight uint64) ([]*Contract, error)
	
	// Update updates an existing contract
	Update(ctx context.Context, contract *Contract) error
	
	// Delete deletes a contract by ID
	Delete(ctx context.Context, id string) error
}

// VTXORepository defines the data access interface for VTXOs
type VTXORepository interface {
	// Create creates a new VTXO
	Create(ctx context.Context, vtxo *VTXO) error
	
	// FindByID retrieves a VTXO by ID
	FindByID(ctx context.Context, id string) (*VTXO, error)
	
	// FindByContract retrieves all VTXOs for a specific contract
	FindByContract(ctx context.Context, contractID string) ([]*VTXO, error)
	
	// FindByUser retrieves all VTXOs for a specific user
	FindByUser(ctx context.Context, userID string, onlyActive bool) ([]*VTXO, error)
	
	// FindActiveVTXOs retrieves all active VTXOs
	FindActiveVTXOs(ctx context.Context) ([]*VTXO, error)
	
	// Update updates an existing VTXO
	Update(ctx context.Context, vtxo *VTXO) error
	
	// Delete deletes a VTXO by ID
	Delete(ctx context.Context, id string) error
}

// OrderRepository defines the data access interface for orders
type OrderRepository interface {
	// Create creates a new order
	Create(ctx context.Context, order *Order) error
	
	// FindByID retrieves an order by ID
	FindByID(ctx context.Context, id string) (*Order, error)
	
	// FindByUser retrieves all orders for a specific user
	FindByUser(ctx context.Context, userID string, status []OrderStatus) ([]*Order, error)
	
	// FindByContractType retrieves orders for a specific contract type and expiry
	FindByContractType(ctx context.Context, contractType ContractType, expiryBlockHeight uint64) ([]*Order, error)
	
	// FindOpenOrders retrieves all open orders
	FindOpenOrders(ctx context.Context) ([]*Order, error)
	
	// FindMatchingOrders finds orders that could potentially match with the given one
	FindMatchingOrders(ctx context.Context, orderID string) ([]*Order, error)
	
	// Update updates an existing order
	Update(ctx context.Context, order *Order) error
	
	// Delete deletes an order by ID
	Delete(ctx context.Context, id string) error
}

// SwapOfferRepository defines the data access interface for swap offers
type SwapOfferRepository interface {
	// Create creates a new swap offer
	Create(ctx context.Context, offer *SwapOffer) error
	
	// FindByID retrieves a swap offer by ID
	FindByID(ctx context.Context, id string) (*SwapOffer, error)
	
	// FindByUser retrieves all swap offers for a specific user
	FindByUser(ctx context.Context, userID string, isOfferor bool) ([]*SwapOffer, error)
	
	// FindByContract retrieves all swap offers for a specific contract
	FindByContract(ctx context.Context, contractID string) ([]*SwapOffer, error)
	
	// FindOpenOffersByVTXO retrieves all open swap offers for a specific VTXO
	FindOpenOffersByVTXO(ctx context.Context, vtxoID string) ([]*SwapOffer, error)
	
	// Update updates an existing swap offer
	Update(ctx context.Context, offer *SwapOffer) error
	
	// Delete deletes a swap offer by ID
	Delete(ctx context.Context, id string) error
}

// TransactionRepository defines the data access interface for transactions
type TransactionRepository interface {
	// Create creates a new transaction
	Create(ctx context.Context, tx *Transaction) error
	
	// FindByID retrieves a transaction by ID
	FindByID(ctx context.Context, id string) (*Transaction, error)
	
	// FindByUser retrieves all transactions for a specific user
	FindByUser(ctx context.Context, userID string, types []TransactionType) ([]*Transaction, error)
	
	// FindByContract retrieves all transactions for a specific contract
	FindByContract(ctx context.Context, contractID string) ([]*Transaction, error)
	
	// FindByType retrieves all transactions of a specific type
	FindByType(ctx context.Context, transactionType TransactionType) ([]*Transaction, error)
	
	// FindByTimeRange retrieves all transactions within a time range
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*Transaction, error)
	
	// Update updates an existing transaction
	Update(ctx context.Context, tx *Transaction) error
}

// HashRateRepository defines the data access interface for hash rate data
type HashRateRepository interface {
	// Create creates a new hash rate data entry
	Create(ctx context.Context, data *HashRateData) error
	
	// FindByBlockHeight retrieves hash rate data for a specific block height
	FindByBlockHeight(ctx context.Context, blockHeight uint64) (*HashRateData, error)
	
	// FindByTimeRange retrieves hash rate data within a time range
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*HashRateData, error)
	
	// GetLatest retrieves the most recent hash rate data
	GetLatest(ctx context.Context) (*HashRateData, error)
	
	// Update updates existing hash rate data
	Update(ctx context.Context, data *HashRateData) error
}

// UserRepository defines the data access interface for user data
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, userID string, publicKey []byte) error
	
	// FindByID retrieves a user by ID
	FindByID(ctx context.Context, userID string) (map[string]interface{}, error)
	
	// GetPublicKey retrieves a user's public key
	GetPublicKey(ctx context.Context, userID string) ([]byte, error)
	
	// Update updates a user's data
	Update(ctx context.Context, userID string, data map[string]interface{}) error
	
	// Delete deletes a user by ID
	Delete(ctx context.Context, userID string) error
}
