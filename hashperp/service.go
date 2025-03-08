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
