package hashperp

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// hashPerpService is the main service implementation that combines all components
type hashPerpService struct {
	contractManager   ContractManager
	vtxoManager       VTXOManager
	orderBookManager  OrderBookManager
	swapOfferManager  SwapOfferManager
	marketDataManager MarketDataManager
	transactionManager TransactionManager
	scriptGenerator    ScriptGenerator
	btcClient         BitcoinClient
}

// NewHashPerpService creates a new HashPerp service that implements the HashPerpService interface
func NewHashPerpService(
	contractManager ContractManager,
	vtxoManager VTXOManager,
	orderBookManager OrderBookManager,
	swapOfferManager SwapOfferManager,
	marketDataManager MarketDataManager,
	transactionManager TransactionManager,
	scriptGenerator ScriptGenerator,
	btcClient BitcoinClient,
) HashPerpService {
	return &hashPerpService{
		contractManager:   contractManager,
		vtxoManager:       vtxoManager,
		orderBookManager:  orderBookManager,
		swapOfferManager:  swapOfferManager,
		marketDataManager: marketDataManager,
		transactionManager: transactionManager,
		scriptGenerator:    scriptGenerator,
		btcClient:         btcClient,
	}
}

// ===========================
// ContractManager delegation
// ===========================

func (s *hashPerpService) CreateContract(
	ctx context.Context,
	buyerID string,
	sellerID string,
	contractType ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) (*Contract, error) {
	return s.contractManager.CreateContract(ctx, buyerID, sellerID, contractType, strikeRate, expiryBlockHeight, size)
}

func (s *hashPerpService) GetContract(ctx context.Context, contractID string) (*Contract, error) {
	return s.contractManager.GetContract(ctx, contractID)
}

func (s *hashPerpService) GetContractsByUser(ctx context.Context, userID string, status []ContractStatus) ([]*Contract, error) {
	return s.contractManager.GetContractsByUser(ctx, userID, status)
}

func (s *hashPerpService) SettleContract(ctx context.Context, contractID string) (*Transaction, error) {
	return s.contractManager.SettleContract(ctx, contractID)
}

func (s *hashPerpService) ExitContract(ctx context.Context, contractID string, userID string) (*Transaction, error) {
	return s.contractManager.ExitContract(ctx, contractID, userID)
}

func (s *hashPerpService) RolloverContract(ctx context.Context, contractID string, newExpiryBlockHeight uint64) (*Contract, *Transaction, error) {
	return s.contractManager.RolloverContract(ctx, contractID, newExpiryBlockHeight)
}

func (s *hashPerpService) ExecuteExitPath(ctx context.Context, contractID string, userID string, exitPathType string) (*Transaction, error) {
	return s.contractManager.ExecuteExitPath(ctx, contractID, userID, exitPathType)
}

// ===========================
// VTXOManager delegation
// ===========================

func (s *hashPerpService) CreateVTXO(
	ctx context.Context,
	contractID string,
	ownerID string,
	amount float64,
	scriptPath string,
	signatureData []byte,
) (*VTXO, error) {
	return s.vtxoManager.CreateVTXO(ctx, contractID, ownerID, amount, scriptPath, signatureData)
}

func (s *hashPerpService) GetVTXO(ctx context.Context, vtxoID string) (*VTXO, error) {
	return s.vtxoManager.GetVTXO(ctx, vtxoID)
}

func (s *hashPerpService) GetVTXOsByContract(ctx context.Context, contractID string) ([]*VTXO, error) {
	return s.vtxoManager.GetVTXOsByContract(ctx, contractID)
}

func (s *hashPerpService) GetVTXOsByUser(ctx context.Context, userID string, onlyActive bool) ([]*VTXO, error) {
	return s.vtxoManager.GetVTXOsByUser(ctx, userID, onlyActive)
}

func (s *hashPerpService) SwapVTXO(ctx context.Context, vtxoID string, newOwnerID string, newSignatureData []byte) (*VTXO, *Transaction, error) {
	return s.vtxoManager.SwapVTXO(ctx, vtxoID, newOwnerID, newSignatureData)
}

func (s *hashPerpService) CreatePresignedExitTransaction(ctx context.Context, vtxoID string, signatureData []byte) (string, error) {
	return s.vtxoManager.CreatePresignedExitTransaction(ctx, vtxoID, signatureData)
}

func (s *hashPerpService) ExecuteVTXOSweep(ctx context.Context, vtxoID string) (*Transaction, error) {
	return s.vtxoManager.ExecuteVTXOSweep(ctx, vtxoID)
}

// ===========================
// OrderBookManager delegation
// ===========================

func (s *hashPerpService) PlaceOrder(
	ctx context.Context,
	userID string,
	orderType OrderType,
	contractType ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) (*Order, error) {
	return s.orderBookManager.PlaceOrder(ctx, userID, orderType, contractType, strikeRate, expiryBlockHeight, size)
}

func (s *hashPerpService) CancelOrder(ctx context.Context, orderID string, userID string) error {
	return s.orderBookManager.CancelOrder(ctx, orderID, userID)
}

func (s *hashPerpService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	return s.orderBookManager.GetOrder(ctx, orderID)
}

func (s *hashPerpService) GetOrdersByUser(ctx context.Context, userID string, status []OrderStatus) ([]*Order, error) {
	return s.orderBookManager.GetOrdersByUser(ctx, userID, status)
}

func (s *hashPerpService) GetOrderBook(ctx context.Context, contractType ContractType, expiryBlockHeight uint64) ([]*Order, error) {
	return s.orderBookManager.GetOrderBook(ctx, contractType, expiryBlockHeight)
}

func (s *hashPerpService) MatchOrders(ctx context.Context) ([]*Contract, error) {
	return s.orderBookManager.MatchOrders(ctx)
}

// ===========================
// SwapOfferManager delegation
// ===========================

func (s *hashPerpService) CreateSwapOffer(
	ctx context.Context,
	offerorID string,
	vtxoID string,
	offeredRate float64,
	expiryTime time.Time,
) (*SwapOffer, error) {
	return s.swapOfferManager.CreateSwapOffer(ctx, offerorID, vtxoID, offeredRate, expiryTime)
}

func (s *hashPerpService) AcceptSwapOffer(ctx context.Context, offerID string, acceptorID string) (*Transaction, error) {
	return s.swapOfferManager.AcceptSwapOffer(ctx, offerID, acceptorID)
}

func (s *hashPerpService) CancelSwapOffer(ctx context.Context, offerID string, offerorID string) error {
	return s.swapOfferManager.CancelSwapOffer(ctx, offerID, offerorID)
}

func (s *hashPerpService) GetSwapOffer(ctx context.Context, offerID string) (*SwapOffer, error) {
	return s.swapOfferManager.GetSwapOffer(ctx, offerID)
}

func (s *hashPerpService) GetSwapOffersByUser(ctx context.Context, userID string, isOfferor bool) ([]*SwapOffer, error) {
	return s.swapOfferManager.GetSwapOffersByUser(ctx, userID, isOfferor)
}

func (s *hashPerpService) GetSwapOffersByContract(ctx context.Context, contractID string) ([]*SwapOffer, error) {
	return s.swapOfferManager.GetSwapOffersByContract(ctx, contractID)
}

// ===========================
// MarketDataManager delegation
// ===========================

func (s *hashPerpService) GetCurrentHashRate(ctx context.Context) (*HashRateData, error) {
	return s.marketDataManager.GetCurrentHashRate(ctx)
}

func (s *hashPerpService) GetHistoricalHashRate(ctx context.Context, startTime, endTime time.Time) ([]*HashRateData, error) {
	return s.marketDataManager.GetHistoricalHashRate(ctx, startTime, endTime)
}

func (s *hashPerpService) GetHashRateAtBlockHeight(ctx context.Context, blockHeight uint64) (*HashRateData, error) {
	return s.marketDataManager.GetHashRateAtBlockHeight(ctx, blockHeight)
}

func (s *hashPerpService) CalculateBTCPerPHPerDay(ctx context.Context, hashRate float64, blockHeight uint64) (float64, error) {
	return s.marketDataManager.CalculateBTCPerPHPerDay(ctx, hashRate, blockHeight)
}

// ===========================
// TransactionManager delegation
// ===========================

func (s *hashPerpService) RecordTransaction(
	ctx context.Context,
	transactionType TransactionType,
	contractID string,
	userIDs []string,
	txHash string,
	amount float64,
	btcPerPHPerDay float64,
	blockHeight uint64,
	relatedEntities map[string]string,
) (*Transaction, error) {
	return s.transactionManager.RecordTransaction(ctx, transactionType, contractID, userIDs, txHash, amount, btcPerPHPerDay, blockHeight, relatedEntities)
}

func (s *hashPerpService) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	return s.transactionManager.GetTransaction(ctx, transactionID)
}

func (s *hashPerpService) GetTransactionsByUser(ctx context.Context, userID string, transactionTypes []TransactionType) ([]*Transaction, error) {
	return s.transactionManager.GetTransactionsByUser(ctx, userID, transactionTypes)
}

func (s *hashPerpService) GetTransactionsByContract(ctx context.Context, contractID string) ([]*Transaction, error) {
	return s.transactionManager.GetTransactionsByContract(ctx, contractID)
}

// ===========================
// ScriptGenerator delegation
// ===========================

func (s *hashPerpService) GenerateContractScripts(ctx context.Context, contract *Contract) (map[string]string, error) {
	return s.scriptGenerator.GenerateContractScripts(ctx, contract)
}

func (s *hashPerpService) GenerateSetupTransaction(ctx context.Context, contract *Contract, buyerVTXO, sellerVTXO *VTXO) (string, error) {
	return s.scriptGenerator.GenerateSetupTransaction(ctx, contract, buyerVTXO, sellerVTXO)
}

func (s *hashPerpService) GenerateFinalTransaction(ctx context.Context, contract *Contract, setupTxID string) (string, error) {
	return s.scriptGenerator.GenerateFinalTransaction(ctx, contract, setupTxID)
}

func (s *hashPerpService) GenerateSettlementTransaction(ctx context.Context, contract *Contract, finalTxID string, winnerID string) (string, error) {
	return s.scriptGenerator.GenerateSettlementTransaction(ctx, contract, finalTxID, winnerID)
}

func (s *hashPerpService) GenerateExitPathScripts(ctx context.Context, contract *Contract) (map[string]string, error) {
	return s.scriptGenerator.GenerateExitPathScripts(ctx, contract)
}

// ===========================
// Additional service methods
// ===========================

// Healthcheck implements HashPerpService.Healthcheck
func (s *hashPerpService) Healthcheck(ctx context.Context) error {
	// Basic healthcheck: check if we can connect to Bitcoin node
	_, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Bitcoin node: %w", err)
	}
	return nil
}

// GetCurrentBlockHeight implements HashPerpService.GetCurrentBlockHeight
func (s *hashPerpService) GetCurrentBlockHeight(ctx context.Context) (uint64, error) {
	return s.btcClient.GetCurrentBlockHeight(ctx)
}

// ValidateContractParameters implements HashPerpService.ValidateContractParameters
func (s *hashPerpService) ValidateContractParameters(
	ctx context.Context,
	contractType ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) error {
	// 1. Validate contract type
	if contractType != CALL && contractType != PUT {
		return errors.New("invalid contract type, must be CALL or PUT")
	}

	// 2. Validate strike rate
	if strikeRate <= 0 {
		return errors.New("strike rate must be positive")
	}

	// 3. Validate expiry block height
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block height: %w", err)
	}

	if expiryBlockHeight <= currentBlockHeight {
		return errors.New("expiry block height must be in the future")
	}

	// Require at least 100 blocks (approx. 1 day) into the future
	if expiryBlockHeight < currentBlockHeight+100 {
		return errors.New("expiry block height must be at least 100 blocks in the future")
	}

	// 4. Validate contract size
	if size <= 0 {
		return errors.New("contract size must be positive")
	}

	// Add a minimum size requirement to prevent dust contracts
	if size < 0.001 { // 0.001 BTC minimum size
		return errors.New("contract size must be at least 0.001 BTC")
	}

	return nil
}

// ExecuteVTXOSweep adds input validation (continued)
func (s *hashPerpService) ExecuteVTXOSweep(
	ctx context.Context, 
	vtxoID string,
) (*Transaction, error) {
	if err := ValidateUUID(vtxoID); err != nil {
		return nil, fmt.Errorf("invalid VTXO ID: %w", err)
	}
	
	return s.vtxoManager.ExecuteVTXOSweep(ctx, vtxoID)
}

// PlaceOrder adds input validation
func (s *hashPerpService) PlaceOrder(
	ctx context.Context,
	userID string,
	orderType OrderType,
	contractType ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) (*Order, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	if err := ValidateOrderType(orderType); err != nil {
		return nil, err
	}
	
	if err := ValidateContractType(contractType); err != nil {
		return nil, err
	}
	
	if err := ValidateRate(strikeRate, 0.0001, 1000); err != nil {
		return nil, fmt.Errorf("invalid strike rate: %w", err)
	}
	
	if err := ValidateAmount(size, 0.001, 100); err != nil {
		return nil, fmt.Errorf("invalid order size: %w", err)
	}
	
	// Validate expiry block height
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	
	if expiryBlockHeight <= currentBlockHeight {
		return nil, errors.New("expiry block height must be in the future")
	}
	
	// Minimum and maximum durations
	minBlocks := uint64(144)   // ~1 day
	maxBlocks := uint64(52560) // ~1 year
	
	if expiryBlockHeight < currentBlockHeight+minBlocks {
		return nil, fmt.Errorf("expiry must be at least %d blocks in the future", minBlocks)
	}
	
	if expiryBlockHeight > currentBlockHeight+maxBlocks {
		return nil, fmt.Errorf("expiry must not exceed %d blocks in the future", maxBlocks)
	}
	
	return s.orderBookManager.PlaceOrder(
		ctx, userID, orderType, contractType, strikeRate, expiryBlockHeight, size)
}

// CancelOrder adds input validation
func (s *hashPerpService) CancelOrder(
	ctx context.Context,
	orderID string,
	userID string,
) error {
	if err := ValidateUUID(orderID); err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}
	
	if err := ValidateUserID(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	
	return s.orderBookManager.CancelOrder(ctx, orderID, userID)
}

// GetOrder adds input validation
func (s *hashPerpService) GetOrder(
	ctx context.Context,
	orderID string,
) (*Order, error) {
	if err := ValidateUUID(orderID); err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}
	
	return s.orderBookManager.GetOrder(ctx, orderID)
}

// GetOrdersByUser adds input validation
func (s *hashPerpService) GetOrdersByUser(
	ctx context.Context,
	userID string,
	status []OrderStatus,
) ([]*Order, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	// Validate each order status if provided
	if len(status) > 0 {
		for _, s := range status {
			if err := ValidateOrderStatus(s); err != nil {
				return nil, err
			}
		}
	}
	
	return s.orderBookManager.GetOrdersByUser(ctx, userID, status)
}

// GetOrderBook adds input validation
func (s *hashPerpService) GetOrderBook(
	ctx context.Context,
	contractType ContractType,
	expiryBlockHeight uint64,
) ([]*Order, error) {
	if err := ValidateContractType(contractType); err != nil {
		return nil, err
	}
	
	// Validate expiry block height
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	
	if expiryBlockHeight <= currentBlockHeight {
		return nil, errors.New("expiry block height must be in the future")
	}
	
	return s.orderBookManager.GetOrderBook(ctx, contractType, expiryBlockHeight)
}

// MatchOrders adds input validation
func (s *hashPerpService) MatchOrders(ctx context.Context) ([]*Contract, error) {
	// No inputs to validate for this method
	return s.orderBookManager.MatchOrders(ctx)
}

// CreateSwapOffer adds input validation
func (s *hashPerpService) CreateSwapOffer(
	ctx context.Context,
	offerorID string,
	vtxoID string,
	offeredRate float64,
	expiryTime time.Time,
) (*SwapOffer, error) {
	if err := ValidateUserID(offerorID); err != nil {
		return nil, fmt.Errorf("invalid offeror ID: %w", err)
	}
	
	if err := ValidateUUID(vtxoID); err != nil {
		return nil, fmt.Errorf("invalid VTXO ID: %w", err)
	}
	
	if err := ValidateRate(offeredRate, 0.0001, 1000); err != nil {
		return nil, fmt.Errorf("invalid offered rate: %w", err)
	}
	
	// Validate expiry time
	now := time.Now().UTC()
	minExpiry := now.Add(1 * time.Hour)    // Minimum 1 hour in the future
	maxExpiry := now.Add(30 * 24 * time.Hour) // Maximum 30 days in the future
	
	if expiryTime.Before(minExpiry) {
		return nil, errors.New("expiry time must be at least 1 hour in the future")
	}
	
	if expiryTime.After(maxExpiry) {
		return nil, errors.New("expiry time must not exceed 30 days in the future")
	}
	
	return s.swapOfferManager.CreateSwapOffer(ctx, offerorID, vtxoID, offeredRate, expiryTime)
}

// AcceptSwapOffer adds input validation
func (s *hashPerpService) AcceptSwapOffer(
	ctx context.Context,
	offerID string,
	acceptorID string,
) (*Transaction, error) {
	if err := ValidateUUID(offerID); err != nil {
		return nil, fmt.Errorf("invalid offer ID: %w", err)
	}
	
	if err := ValidateUserID(acceptorID); err != nil {
		return nil, fmt.Errorf("invalid acceptor ID: %w", err)
	}
	
	return s.swapOfferManager.AcceptSwapOffer(ctx, offerID, acceptorID)
}

// CancelSwapOffer adds input validation
func (s *hashPerpService) CancelSwapOffer(
	ctx context.Context,
	offerID string,
	offerorID string,
) error {
	if err := ValidateUUID(offerID); err != nil {
		return fmt.Errorf("invalid offer ID: %w", err)
	}
	
	if err := ValidateUserID(offerorID); err != nil {
		return fmt.Errorf("invalid offeror ID: %w", err)
	}
	
	return s.swapOfferManager.CancelSwapOffer(ctx, offerID, offerorID)
}

// GetSwapOffer adds input validation
func (s *hashPerpService) GetSwapOffer(
	ctx context.Context,
	offerID string,
) (*SwapOffer, error) {
	if err := ValidateUUID(offerID); err != nil {
		return nil, fmt.Errorf("invalid offer ID: %w", err)
	}
	
	return s.swapOfferManager.GetSwapOffer(ctx, offerID)
}

// GetSwapOffersByUser adds input validation
func (s *hashPerpService) GetSwapOffersByUser(
	ctx context.Context,
	userID string,
	isOfferor bool,
) ([]*SwapOffer, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	return s.swapOfferManager.GetSwapOffersByUser(ctx, userID, isOfferor)
}

// GetSwapOffersByContract adds input validation
func (s *hashPerpService) GetSwapOffersByContract(
	ctx context.Context,
	contractID string,
) ([]*SwapOffer, error) {
	if err := ValidateUUID(contractID); err != nil {
		return nil, fmt.Errorf("invalid contract ID: %w", err)
	}
	
	return s.swapOfferManager.GetSwapOffersByContract(ctx, contractID)
}

// GetCurrentHashRate adds input validation
func (s *hashPerpService) GetCurrentHashRate(ctx context.Context) (*HashRateData, error) {
	// No inputs to validate for this method
	return s.marketDataManager.GetCurrentHashRate(ctx)
}

// GetHistoricalHashRate adds input validation
func (s *hashPerpService) GetHistoricalHashRate(
	ctx context.Context,
	startTime, endTime time.Time,
) ([]*HashRateData, error) {
	if err := ValidateTimeRange(startTime, endTime); err != nil {
		return nil, err
	}
	
	// Add a reasonable limit to the time range to prevent excessive queries
	maxDuration := 90 * 24 * time.Hour // 90 days
	if endTime.Sub(startTime) > maxDuration {
		return nil, fmt.Errorf("time range exceeds maximum allowed duration of %v", maxDuration)
	}
	
	return s.marketDataManager.GetHistoricalHashRate(ctx, startTime, endTime)
}

// GetHashRateAtBlockHeight adds input validation
func (s *hashPerpService) GetHashRateAtBlockHeight(
	ctx context.Context,
	blockHeight uint64,
) (*HashRateData, error) {
	// Validate block height is reasonable
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	
	if blockHeight > currentBlockHeight {
		return nil, errors.New("block height cannot be in the future")
	}
	
	// Limit how far back we can query to prevent excessive historical queries
	const maxHistoricalBlocks = 52560 * 2 // ~2 years
	if currentBlockHeight - blockHeight > maxHistoricalBlocks {
		return nil, fmt.Errorf("requested block height is too far in the past (max %d blocks)", maxHistoricalBlocks)
	}
	
	return s.marketDataManager.GetHashRateAtBlockHeight(ctx, blockHeight)
}

// CalculateBTCPerPHPerDay adds input validation
func (s *hashPerpService) CalculateBTCPerPHPerDay(
	ctx context.Context,
	hashRate float64,
	blockHeight uint64,
) (float64, error) {
	// Validate hash rate
	if hashRate <= 0 {
		return 0, errors.New("hash rate must be positive")
	}
	
	// Validate hash rate is within reasonable bounds
	// Current global hash rate is around 500-600 EH/s (500,000-600,000 PH/s)
	// Set a generous upper bound of 10 million PH/s for future growth
	const maxHashRate = 10000000.0 // 10 million PH/s
	if hashRate > maxHashRate {
		return 0, fmt.Errorf("hash rate exceeds maximum allowed value (%f PH/s)", maxHashRate)
	}
	
	// Validate block height
	if blockHeight > 0 {
		currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get current block height: %w", err)
		}
		
		if blockHeight > currentBlockHeight {
			return 0, errors.New("block height cannot be in the future")
		}
	}
	
	return s.marketDataManager.CalculateBTCPerPHPerDay(ctx, hashRate, blockHeight)
}

// RecordTransaction adds input validation
func (s *hashPerpService) RecordTransaction(
	ctx context.Context,
	transactionType TransactionType,
	contractID string,
	userIDs []string,
	txHash string,
	amount float64,
	btcPerPHPerDay float64,
	blockHeight uint64,
	relatedEntities map[string]string,
) (*Transaction, error) {
	if err := ValidateTransactionType(transactionType); err != nil {
		return nil, err
	}
	
	if contractID != "" {
		if err := ValidateUUID(contractID); err != nil {
			return nil, fmt.Errorf("invalid contract ID: %w", err)
		}
	}
	
	// Validate user IDs
	if len(userIDs) == 0 {
		return nil, errors.New("at least one user ID is required")
	}
	
	for _, userID := range userIDs {
		if err := ValidateUserID(userID); err != nil {
			return nil, fmt.Errorf("invalid user ID (%s): %w", userID, err)
		}
	}
	
	// Validate amount
	if err := ValidateAmount(amount, 0, 0); err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	
	// Validate btcPerPHPerDay if provided
	if btcPerPHPerDay != 0 {
		if btcPerPHPerDay < 0 {
			return nil, errors.New("BTC per PH per day cannot be negative")
		}
	}
	
	// Validate block height if provided
	if blockHeight > 0 {
		currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current block height: %w", err)
		}
		
		if blockHeight > currentBlockHeight {
			return nil, errors.New("block height cannot be in the future")
		}
	}
	
	return s.transactionManager.RecordTransaction(
		ctx, transactionType, contractID, userIDs, txHash, amount, btcPerPHPerDay, blockHeight, relatedEntities)
}

// GetTransaction adds input validation
func (s *hashPerpService) GetTransaction(
	ctx context.Context,
	transactionID string,
) (*Transaction, error) {
	if err := ValidateUUID(transactionID); err != nil {
		return nil, fmt.Errorf("invalid transaction ID: %w", err)
	}
	
	return s.transactionManager.GetTransaction(ctx, transactionID)
}

// GetTransactionsByUser adds input validation
func (s *hashPerpService) GetTransactionsByUser(
	ctx context.Context,
	userID string,
	transactionTypes []TransactionType,
) ([]*Transaction, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	// Validate transaction types if provided
	if len(transactionTypes) > 0 {
		for _, txType := range transactionTypes {
			if err := ValidateTransactionType(txType); err != nil {
				return nil, err
			}
		}
	}
	
	return s.transactionManager.GetTransactionsByUser(ctx, userID, transactionTypes)
}

// GetTransactionsByContract adds input validation
func (s *hashPerpService) GetTransactionsByContract(
	ctx context.Context,
	contractID string,
) ([]*Transaction, error) {
	if err := ValidateUUID(contractID); err != nil {
		return nil, fmt.Errorf("invalid contract ID: %w", err)
	}
	
	return s.transactionManager.GetTransactionsByContract(ctx, contractID)
}

// GenerateContractScripts adds input validation
func (s *hashPerpService) GenerateContractScripts(
	ctx context.Context,
	contract *Contract,
) (map[string]string, error) {
	if contract == nil {
		return nil, errors.New("contract cannot be nil")
	}
	
	if err := ValidateUUID(contract.ID); err != nil {
		return nil, fmt.Errorf("invalid contract ID: %w", err)
	}
	
	if err := ValidateContractType(contract.ContractType); err != nil {
		return nil, err
	}
	
	if err := ValidateUserID(contract.BuyerID); err != nil {
		return nil, fmt.Errorf("invalid buyer ID: %w", err)
	}
	
	if err := ValidateUserID(contract.SellerID); err != nil {
		return nil, fmt.Errorf("invalid seller ID: %w", err)
	}
	
	return s.scriptGenerator.GenerateContractScripts(ctx, contract)
}

// GenerateSetupTransaction adds input validation
func (s *hashPerpService) GenerateSetupTransaction(
	ctx context.Context,
	contract *Contract,
	buyerVTXO, sellerVTXO *VTXO,
) (string, error) {
	if contract == nil {
		return "", errors.New("contract cannot be nil")
	}
	
	if err := ValidateUUID(contract.ID); err != nil {
		return "", fmt.Errorf("invalid contract ID: %w", err)
	}
	
	if buyerVTXO == nil {
		return "", errors.New("buyer VTXO cannot be nil")
	}
	
	if err := ValidateUUID(buyerVTXO.ID); err != nil {
		return "", fmt.Errorf("invalid buyer VTXO ID: %w", err)
	}
	
	if sellerVTXO == nil {
		return "", errors.New("seller VTXO cannot be nil")
	}
	
	if err := ValidateUUID(sellerVTXO.ID); err != nil {
		return "", fmt.Errorf("invalid seller VTXO ID: %w", err)
	}
	
	return s.scriptGenerator.GenerateSetupTransaction(ctx, contract, buyerVTXO, sellerVTXO)
}

// GenerateFinalTransaction adds input validation
func (s *hashPerpService) GenerateFinalTransaction(
	ctx context.Context,
	contract *Contract,
	setupTxID string,
) (string, error) {
	if contract == nil {
		return "", errors.New("contract cannot be nil")
	}
	
	if err := ValidateUUID(contract.ID); err != nil {
		return "", fmt.Errorf("invalid contract ID: %w", err)
	}
	
	if setupTxID == "" {
		return "", errors.New("setup transaction ID cannot be empty")
	}
	
	return s.scriptGenerator.GenerateFinalTransaction(ctx, contract, setupTxID)
}

// GenerateSettlementTransaction adds input validation
func (s *hashPerpService) GenerateSettlementTransaction(
	ctx context.Context,
	contract *Contract,
	finalTxID string,
	winnerID string,
) (string, error) {
	if contract == nil {
		return "", errors.New("contract cannot be nil")
	}
	
	if err := ValidateUUID(contract.ID); err != nil {
		return "", fmt.Errorf("invalid contract ID: %w", err)
	}
	
	if finalTxID == "" {
		return "", errors.New("final transaction ID cannot be empty")
	}
	
	if err := ValidateUserID(winnerID); err != nil {
		return "", fmt.Errorf("invalid winner ID: %w", err)
	}
	
	// Verify the winner is part of the contract
	if winnerID != contract.BuyerID && winnerID != contract.SellerID {
		return "", errors.New("winner ID must be either the buyer or seller of the contract")
	}
	
	return s.scriptGenerator.GenerateSettlementTransaction(ctx, contract, finalTxID, winnerID)
}

// GenerateExitPathScripts adds input validation
func (s *hashPerpService) GenerateExitPathScripts(
	ctx context.Context,
	contract *Contract,
) (map[string]string, error) {
	if contract == nil {
		return nil, errors.New("contract cannot be nil")
	}
	
	if err := ValidateUUID(contract.ID); err != nil {
		return nil, fmt.Errorf("invalid contract ID: %w", err)
	}
	
	return s.scriptGenerator.GenerateExitPathScripts(ctx, contract)
}

// Healthcheck adds input validation
func (s *hashPerpService) Healthcheck(ctx context.Context) error {
	// No inputs to validate for this method
	return nil
}

// GetCurrentBlockHeight adds input validation
func (s *hashPerpService) GetCurrentBlockHeight(ctx context.Context) (uint64, error) {
	// No inputs to validate for this method
	return s.btcClient.GetCurrentBlockHeight(ctx)
}
