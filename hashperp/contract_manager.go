
package hashperp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Error definitions
var (
	ErrContractNotFound        = errors.New("contract not found")
	ErrVTXONotFound            = errors.New("VTXO not found")
	ErrInvalidOwner            = errors.New("user is not the owner of this VTXO")
	ErrInvalidContractStatus   = errors.New("contract is not in a valid status for this operation")
	ErrInvalidParameters       = errors.New("invalid contract parameters")
	ErrInsufficientFunds       = errors.New("insufficient funds for operation")
	ErrInvalidSignature        = errors.New("invalid signature data")
	ErrNonCooperativeSettlement = errors.New("non-cooperative settlement required")
	ErrInvalidBlockHeight      = errors.New("invalid block height")
	ErrSwapNotAvailable        = errors.New("swap not available or already accepted")
	ErrPositionAlreadyFilled   = errors.New("contract position already filled")
	ErrUserNotInContract       = errors.New("user is not a participant in this contract")
	ErrVTXONotActive           = errors.New("VTXO is not active")
	ErrDynamicJoinRejected     = errors.New("dynamic join request was rejected")
)

// contractService implements the ContractManager interface
type contractService struct {
	contractRepo    ContractRepository
	vtxoRepo        VTXORepository
	transactionRepo TransactionRepository
	scriptGen       ScriptGenerator
	btcClient       BitcoinClient
	blockHeight     uint64 // Current block height, regularly updated
	swapManager     SwapOfferManager // For handling VTXO swaps
}

// Below are additional helper methods that would typically be part of a complete implementation

// validateUserIsContractParty checks if a user is a buyer or seller in a contract
func (s *contractService) validateUserIsContractParty(contract *Contract, userID string) error {
	if contract.BuyerID != userID && contract.SellerID != userID {
		return ErrUserNotInContract
	}
	return nil
}

// GetCurrentBlockHeight provides the current block height for service consumers
func (s *contractService) GetCurrentBlockHeight(ctx context.Context) (uint64, error) {
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current block height: %w", err)
	}
	
	// Update cached block height
	s.blockHeight = currentBlockHeight
	return currentBlockHeight, nil
}

// ValidateContractParameters implements the HashPerpService interface method
func (s *contractService) ValidateContractParameters(
	ctx context.Context, 
	contractType ContractType,
	strikeRate float64, 
	expiryBlockHeight uint64, 
	size float64,
) error {
	return s.validateContractParameters(ctx, contractType, strikeRate, expiryBlockHeight, size)
}

// ExecuteExitPath implements ContractManager.ExecuteExitPath
func (s *contractService) ExecuteExitPath(
	ctx context.Context,
	contractID string,
	userID string,
	exitPathType string,
) (*Transaction, error) {
	// 1. Get the contract
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 2. Verify the user is a party to the contract
	if contract.BuyerID != userID && contract.SellerID != userID {
		return nil, ErrUserNotInContract
	}

	// 3. Validate contract status
	if contract.Status != ACTIVE {
		return nil, ErrInvalidContractStatus
	}

	// 4. Get current block height and hash rate for the transaction record
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	s.blockHeight = currentBlockHeight // Update cached block height

	// 5. Determine counterparty
	var counterpartyID string
	if userID == contract.BuyerID {
		counterpartyID = contract.SellerID
	} else {
		counterpartyID = contract.BuyerID
	}

	// 6. Get VTXOs for the contract
	buyerVTXO, err := s.vtxoRepo.FindByID(ctx, contract.BuyerVTXO)
	if err != nil || buyerVTXO == nil {
		return nil, fmt.Errorf("failed to get buyer VTXO: %w", err)
	}

	sellerVTXO, err := s.vtxoRepo.FindByID(ctx, contract.SellerVTXO)
	if err != nil || sellerVTXO == nil {
		return nil, fmt.Errorf("failed to get seller VTXO: %w", err)
	}

	// 7. Execute the appropriate exit path based on type
	var exitTxHex, exitTxID string
	var relatedEntities map[string]string = make(map[string]string)

	switch exitPathType {
	case "early_exit":
		// Voluntary early exit (typically with a fee)
		// Generate exit transaction
		scripts, err := s.scriptGen.GenerateExitPathScripts(ctx, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to generate exit scripts: %w", err)
		}
		
		exitTxHex = scripts["early_exit"]
		relatedEntities["exit_reason"] = "voluntary_early_exit"
		
	case "mutual_agreement":
		// Both parties agree to exit (often with no fee)
		// Check if both parties have signed agreement (in a real system, would verify signatures)
		scripts, err := s.scriptGen.GenerateExitPathScripts(ctx, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to generate exit scripts: %w", err)
		}
		
		exitTxHex = scripts["mutual_agreement"]
		relatedEntities["exit_reason"] = "mutual_agreement"
		
	case "timeout":
		// Contract expired but wasn't settled normally
		if currentBlockHeight < contract.ExpiryBlockHeight {
			return nil, fmt.Errorf("cannot use timeout exit path before contract expiry")
		}
		
		// Ensure sufficient time has passed since expiry (e.g., 144 blocks / 1 day)
		const timeoutBuffer = 144 // blocks
		if currentBlockHeight < contract.ExpiryBlockHeight + timeoutBuffer {
			return nil, fmt.Errorf("timeout exit path requires waiting %d blocks after expiry", timeoutBuffer)
		}
		
		scripts, err := s.scriptGen.GenerateExitPathScripts(ctx, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to generate exit scripts: %w", err)
		}
		
		exitTxHex = scripts["timeout"]
		relatedEntities["exit_reason"] = "settlement_timeout"
		relatedEntities["blocks_since_expiry"] = fmt.Sprintf("%d", currentBlockHeight - contract.ExpiryBlockHeight)
		
	case "forced_settlement":
		// One party forces settlement (usually after contract expiry)
		if currentBlockHeight < contract.ExpiryBlockHeight {
			return nil, fmt.Errorf("cannot force settlement before contract expiry")
		}
		
		scripts, err := s.scriptGen.GenerateExitPathScripts(ctx, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to generate exit scripts: %w", err)
		}
		
		exitTxHex = scripts["forced_settlement"]
		relatedEntities["exit_reason"] = "forced_settlement"
		relatedEntities["initiated_by"] = userID
		
	case "dispute_resolution":
		// Use third-party oracle or arbiter for dispute resolution
		scripts, err := s.scriptGen.GenerateExitPathScripts(ctx, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to generate exit scripts: %w", err)
		}
		
		exitTxHex = scripts["dispute_resolution"]
		relatedEntities["exit_reason"] = "dispute_resolution"
		relatedEntities["dispute_initiator"] = userID
		
	case "emergency_exit":
		// Used for security measures or protocol emergencies
		scripts, err := s.scriptGen.GenerateExitPathScripts(ctx, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to generate exit scripts: %w", err)
		}
		
		exitTxHex = scripts["emergency"]
		relatedEntities["exit_reason"] = "emergency_protocol_action"
		
	default:
		return nil, fmt.Errorf("unknown exit path type: %s", exitPathType)
	}

	// 8. Broadcast the exit transaction
	exitTxID, err = s.btcClient.BroadcastTransaction(ctx, exitTxHex)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast exit transaction: %w", err)
	}

	// 9. Update contract status to EXITED
	contract.Status = EXITED
	if err := s.contractRepo.Update(ctx, contract); err != nil {
		return nil, fmt.Errorf("failed to update contract status: %w", err)
	}

	// 10. Mark VTXOs as inactive
	buyerVTXO.IsActive = false
	sellerVTXO.IsActive = false

	if err := s.vtxoRepo.Update(ctx, buyerVTXO); err != nil {
		return nil, fmt.Errorf("failed to update buyer VTXO: %w", err)
	}

	if err := s.vtxoRepo.Update(ctx, sellerVTXO); err != nil {
		return nil, fmt.Errorf("failed to update seller VTXO: %w", err)
	}

	// 11. Get current hash rate for transaction record
	hashRate, err := s.btcClient.GetBlockHashRate(ctx, currentBlockHeight)
	if err != nil {
		// Non-critical error, can continue with settlement
		hashRate = 0 // Default value if hash rate retrieval fails
	}
	currentBTCPerPHPerDay := calculateBTCPerPHPerDay(hashRate)

	// 12. Add core related entities
	relatedEntities["exit_path_type"] = exitPathType
	relatedEntities["buyer_vtxo"] = buyerVTXO.ID
	relatedEntities["seller_vtxo"] = sellerVTXO.ID
	relatedEntities["initiated_by"] = userID
	relatedEntities["counterparty"] = counterpartyID

	// 13. Record the exit transaction
	tx := &Transaction{
		ID:              generateUniqueID(),
		Type:            EXIT_PATH_EXECUTION,
		Timestamp:       time.Now().UTC(),
		ContractID:      contractID,
		UserIDs:         []string{contract.BuyerID, contract.SellerID},
		TxHash:          exitTxID,
		Amount:          contract.Size,
		BTCPerPHPerDay:  currentBTCPerPHPerDay,
		BlockHeight:     currentBlockHeight,
		RelatedEntities: relatedEntities,
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to record exit transaction: %w", err)
	}

	return tx, nil
}

// ContractRepository defines the data access interface for contracts
type ContractRepository interface {
	Create(ctx context.Context, contract *Contract) error
	FindByID(ctx context.Context, id string) (*Contract, error)
	FindByUser(ctx context.Context, userID string, status []ContractStatus) ([]*Contract, error)
	Update(ctx context.Context, contract *Contract) error
	Delete(ctx context.Context, id string) error
}

// VTXORepository defines the data access interface for VTXOs
type VTXORepository interface {
	Create(ctx context.Context, vtxo *VTXO) error
	FindByID(ctx context.Context, id string) (*VTXO, error)
	FindByContract(ctx context.Context, contractID string) ([]*VTXO, error)
	FindByUser(ctx context.Context, userID string, onlyActive bool) ([]*VTXO, error)
	Update(ctx context.Context, vtxo *VTXO) error
	Delete(ctx context.Context, id string) error
}

// TransactionRepository defines the data access interface for transactions
type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) error
	FindByID(ctx context.Context, id string) (*Transaction, error)
	FindByUser(ctx context.Context, userID string, types []TransactionType) ([]*Transaction, error)
	FindByContract(ctx context.Context, contractID string) ([]*Transaction, error)
}

// BitcoinClient defines the interface for interacting with the Bitcoin network
type BitcoinClient interface {
	GetCurrentBlockHeight(ctx context.Context) (uint64, error)
	GetBlockHashRate(ctx context.Context, blockHeight uint64) (float64, error)
	BroadcastTransaction(ctx context.Context, txHex string) (string, error)
	ValidateSignature(ctx context.Context, message []byte, signature []byte, pubKey []byte) (bool, error)
}

// NewContractService creates a new contract service
func NewContractService(
	contractRepo ContractRepository,
	vtxoRepo VTXORepository,
	transactionRepo TransactionRepository,
	scriptGen ScriptGenerator,
	btcClient BitcoinClient,
	swapManager SwapOfferManager,
) ContractManager {
	return &contractService{
		contractRepo:    contractRepo,
		vtxoRepo:        vtxoRepo,
		transactionRepo: transactionRepo,
		scriptGen:       scriptGen,
		btcClient:       btcClient,
		swapManager:     swapManager,
	}
}

// Helper function to generate a unique ID
func generateUniqueID() string {
	return uuid.New().String()
}

// Helper function to calculate expiry date based on block height
func calculateExpiryDate(expiryBlockHeight, currentBlockHeight uint64) time.Time {
	// Assuming average 10 minutes per block
	blockDifference := expiryBlockHeight - currentBlockHeight
	minutesUntilExpiry := blockDifference * 10
	return time.Now().UTC().Add(time.Duration(minutesUntilExpiry) * time.Minute)
}

// calculateBTCPerPHPerDay calculates BTC per PetaHash per day based on hash rate
func calculateBTCPerPHPerDay(hashRate float64) float64 {
	// Simplified calculation - in a real implementation this would be more complex
	// and consider difficulty, block rewards, etc.
	const blocksPerDay = 144 // 6 blocks per hour * 24 hours
	const blockRewardBTC = 6.25 // Current block reward as of 2022-2023
	const networkHashRatePH = 400000 // Example network hash rate in PH/s

	// Calculate BTC per PH per day
	return (blocksPerDay * blockRewardBTC) / networkHashRatePH
}

// Helper method to validate contract parameters
func (s *contractService) validateContractParameters(
	ctx context.Context,
	contractType ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) error {
	// Validate contract type
	if contractType != CALL && contractType != PUT {
		return fmt.Errorf("%w: invalid contract type", ErrInvalidParameters)
	}

	// Validate strike rate
	if strikeRate <= 0 {
		return fmt.Errorf("%w: strike rate must be positive", ErrInvalidParameters)
	}

	// Validate size
	if size <= 0 {
		return fmt.Errorf("%w: size must be positive", ErrInvalidParameters)
	}

	// Validate expiry block height
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block height: %w", err)
	}
	s.blockHeight = currentBlockHeight // Update cached block height

	if expiryBlockHeight <= currentBlockHeight {
		return fmt.Errorf("%w: expiry block height must be in the future", ErrInvalidBlockHeight)
	}

	// Validate minimum contract duration (e.g., at least 100 blocks ~16 hours)
	const minBlockDuration = 100
	if expiryBlockHeight < currentBlockHeight+minBlockDuration {
		return fmt.Errorf("%w: contract duration too short, minimum %d blocks", ErrInvalidParameters, minBlockDuration)
	}

	return nil
}

// Helper method to create a VTXO for a contract
func (s *contractService) createContractVTXO(
	ctx context.Context,
	contractID string,
	ownerID string,
	amount float64,
	scriptPath string,
	signatureData []byte,
) (*VTXO, error) {
	vtxo := &VTXO{
		ID:                generateUniqueID(),
		ContractID:        contractID,
		OwnerID:           ownerID,
		Amount:            amount,
		ScriptPath:        scriptPath,
		CreationTimestamp: time.Now().UTC(),
		SignatureData:     signatureData,
		IsActive:          true,
	}

	if err := s.vtxoRepo.Create(ctx, vtxo); err != nil {
		return nil, fmt.Errorf("failed to create VTXO: %w", err)
	}

	return vtxo, nil
}

// Helper method to record contract creation transaction
func (s *contractService) recordContractCreationTransaction(
	ctx context.Context,
	contract *Contract,
	buyerVTXOID string,
	sellerVTXOID string,
) (*Transaction, error) {
	tx := &Transaction{
		ID:              generateUniqueID(),
		Type:            CONTRACT_CREATION,
		Timestamp:       time.Now().UTC(),
		ContractID:      contract.ID,
		UserIDs:         []string{contract.BuyerID, contract.SellerID},
		Amount:          contract.Size,
		BTCPerPHPerDay:  contract.StrikeRate, // Using strike rate at creation time
		BlockHeight:     s.blockHeight,
		RelatedEntities: map[string]string{
			"buyer_vtxo":  buyerVTXOID,
			"seller_vtxo": sellerVTXOID,
		},
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to record contract creation transaction: %w", err)
	}

	return tx, nil
}

// CreateContract implements ContractManager.CreateContract
func (s *contractService) CreateContract(
	ctx context.Context,
	buyerID, sellerID string,
	contractType ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) (*Contract, error) {
	// 1. Validate parameters
	if err := s.validateContractParameters(ctx, contractType, strikeRate, expiryBlockHeight, size); err != nil {
		return nil, err
	}

	// 2. Generate a unique contract ID
	contractID := generateUniqueID()

	// 3. Calculate human-readable expiry date based on block height and average block time
	expiryDate := calculateExpiryDate(expiryBlockHeight, s.blockHeight)

	// 4. Create the contract
	contract := &Contract{
		ID:               contractID,
		ContractType:     contractType,
		StrikeRate:       strikeRate,
		ExpiryBlockHeight: expiryBlockHeight,
		ExpiryDate:       expiryDate,
		CreationTime:     time.Now().UTC(),
		Status:           PENDING,
		BuyerID:          buyerID,
		SellerID:         sellerID,
		Size:             size,
		// VTXO IDs will be set later
	}

	// 5. Save the contract to the repository
	if err := s.contractRepo.Create(ctx, contract); err != nil {
		return nil, fmt.Errorf("failed to create contract: %w", err)
	}

	// 6. Generate scripts for the contract
	scripts, err := s.scriptGen.GenerateContractScripts(ctx, contract)
	if err != nil {
		// If script generation fails, delete the contract
		_ = s.contractRepo.Delete(ctx, contractID)
		return nil, fmt.Errorf("failed to generate contract scripts: %w", err)
	}

	// 7. Create VTXOs for buyer and seller
	buyerVTXO, err := s.createContractVTXO(ctx, contractID, buyerID, size/2, scripts["buyerScriptPath"], nil)
	if err != nil {
		_ = s.contractRepo.Delete(ctx, contractID)
		return nil, fmt.Errorf("failed to create buyer VTXO: %w", err)
	}

	sellerVTXO, err := s.createContractVTXO(ctx, contractID, sellerID, size/2, scripts["sellerScriptPath"], nil)
	if err != nil {
		_ = s.contractRepo.Delete(ctx, contractID)
		_ = s.vtxoRepo.Delete(ctx, buyerVTXO.ID)
		return nil, fmt.Errorf("failed to create seller VTXO: %w", err)
	}

	// 8. Update contract with VTXO IDs
	contract.BuyerVTXO = buyerVTXO.ID
	contract.SellerVTXO = sellerVTXO.ID
	contract.Status = ACTIVE

	if err := s.contractRepo.Update(ctx, contract); err != nil {
		return nil, fmt.Errorf("failed to update contract with VTXOs: %w", err)
	}

	// 9. Record transaction
	_, err = s.recordContractCreationTransaction(ctx, contract, buyerVTXO.ID, sellerVTXO.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to record contract creation transaction: %w", err)
	}

	return contract, nil
}

// GetContract implements ContractManager.GetContract
func (s *contractService) GetContract(ctx context.Context, contractID string) (*Contract, error) {
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}
	return contract, nil
}

// GetContractsByUser implements ContractManager.GetContractsByUser
func (s *contractService) GetContractsByUser(ctx context.Context, userID string, status []ContractStatus) ([]*Contract, error) {
	contracts, err := s.contractRepo.FindByUser(ctx, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get contracts by user: %w", err)
	}
	return contracts, nil
}

// SettleContract implements ContractManager.SettleContract
func (s *contractService) SettleContract(ctx context.Context, contractID string) (*Transaction, error) {
	// 1. Get the contract
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 2. Validate contract status
	if contract.Status != ACTIVE {
		return nil, ErrInvalidContractStatus
	}

	// 3. Check if the contract has expired
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	s.blockHeight = currentBlockHeight // Update cached block height

	if currentBlockHeight < contract.ExpiryBlockHeight {
		return nil, fmt.Errorf("contract has not expired yet, current height %d < expiry height %d", 
		    currentBlockHeight, contract.ExpiryBlockHeight)
	}

	// 4. Get the hash rate at expiry block
	hashRate, err := s.btcClient.GetBlockHashRate(ctx, contract.ExpiryBlockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash rate at expiry block: %w", err)
	}

	// 5. Calculate BTC per PH per day rate at settlement
	btcPerPHPerDay := calculateBTCPerPHPerDay(hashRate)

	// 6. Determine winner (buyer or seller)
	var winnerID, loserID string
	if (contract.ContractType == CALL && btcPerPHPerDay > contract.StrikeRate) ||
		(contract.ContractType == PUT && btcPerPHPerDay < contract.StrikeRate) {
		winnerID = contract.BuyerID
		loserID = contract.SellerID
	} else {
		winnerID = contract.SellerID
		loserID = contract.BuyerID
	}

	// 7. Generate settlement transaction
	buyerVTXO, err := s.vtxoRepo.FindByID(ctx, contract.BuyerVTXO)
	if err != nil {
		return nil, fmt.Errorf("failed to get buyer VTXO: %w", err)
	}

	sellerVTXO, err := s.vtxoRepo.FindByID(ctx, contract.SellerVTXO)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller VTXO: %w", err)
	}

	// 8. Generate and broadcast settlement transaction
	setupTx, err := s.scriptGen.GenerateSetupTransaction(ctx, contract, buyerVTXO, sellerVTXO)
	if err != nil {
		return nil, fmt.Errorf("failed to generate setup transaction: %w", err)
	}

	setupTxID, err := s.btcClient.BroadcastTransaction(ctx, setupTx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast setup transaction: %w", err)
	}

	finalTx, err := s.scriptGen.GenerateFinalTransaction(ctx, contract, setupTxID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate final transaction: %w", err)
	}

	finalTxID, err := s.btcClient.BroadcastTransaction(ctx, finalTx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast final transaction: %w", err)
	}

	settlementTx, err := s.scriptGen.GenerateSettlementTransaction(ctx, contract, finalTxID, winnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate settlement transaction: %w", err)
	}

	settlementTxID, err := s.btcClient.BroadcastTransaction(ctx, settlementTx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast settlement transaction: %w", err)
	}

	// 9. Update contract status
	contract.Status = SETTLED
	contract.SettlementTx = settlementTxID
	contract.SettlementRate = btcPerPHPerDay

	if err := s.contractRepo.Update(ctx, contract); err != nil {
		return nil, fmt.Errorf("failed to update contract status: %w", err)
	}

	// 10. Mark VTXOs as inactive
	buyerVTXO.IsActive = false
	sellerVTXO.IsActive = false

	if err := s.vtxoRepo.Update(ctx, buyerVTXO); err != nil {
		return nil, fmt.Errorf("failed to update buyer VTXO: %w", err)
	}

	if err := s.vtxoRepo.Update(ctx, sellerVTXO); err != nil {
		return nil, fmt.Errorf("failed to update seller VTXO: %w", err)
	}

	// 11. Record settlement transaction
	tx := &Transaction{
		ID:              generateUniqueID(),
		Type:            CONTRACT_SETTLEMENT,
		Timestamp:       time.Now().UTC(),
		ContractID:      contractID,
		UserIDs:         []string{contract.BuyerID, contract.SellerID},
		TxHash:          settlementTxID,
		Amount:          contract.Size,
		BTCPerPHPerDay:  btcPerPHPerDay,
		BlockHeight:     contract.ExpiryBlockHeight,
		RelatedEntities: map[string]string{
			"buyer_vtxo":  contract.BuyerVTXO,
			"seller_vtxo": contract.SellerVTXO,
			"winner_id":   winnerID,
			"loser_id":    loserID,
		},
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to record settlement transaction: %w", err)
	}

	return tx, nil
}

// ExitContract implements ContractManager.ExitContract
func (s *contractService) ExitContract(
	ctx context.Context,
	contractID string,
	userID string,
) (*Transaction, error) {
	// 1. Get the contract
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 2. Verify the user is a party to the contract
	if contract.BuyerID != userID && contract.SellerID != userID {
		return nil, ErrUserNotInContract
	}

	// 3. Validate contract status
	if contract.Status != ACTIVE {
		return nil, ErrInvalidContractStatus
	}

	// 4. Get current block height and hash rate
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	s.blockHeight = currentBlockHeight // Update cached block height

	hashRate, err := s.btcClient.GetBlockHashRate(ctx, currentBlockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get current hash rate: %w", err)
	}
	currentBTCPerPHPerDay := calculateBTCPerPHPerDay(hashRate)

	// 5. Determine counterparty
	var counterpartyID string
	if userID == contract.BuyerID {
		counterpartyID = contract.SellerID
	} else {
		counterpartyID = contract.BuyerID
	}

	// 6. Calculate exit fee and settlement amount
	// For early exit, apply a penalty to the exiting party
	exitFee := contract.Size * 0.05 // 5% exit fee as an example
	
	// Calculate settlement based on current market conditions
	// This is a simplified approach - in a real system, this would involve more complex pricing
	var settlementAmount float64
	if userID == contract.BuyerID {
		if contract.ContractType == CALL && currentBTCPerPHPerDay > contract.StrikeRate {
			// Buyer is in profit on a CALL
			profit := (currentBTCPerPHPerDay - contract.StrikeRate) / contract.StrikeRate * contract.Size
			settlementAmount = (contract.Size / 2) + profit - exitFee
		} else if contract.ContractType == PUT && currentBTCPerPHPerDay < contract.StrikeRate {
			// Buyer is in profit on a PUT
			profit := (contract.StrikeRate - currentBTCPerPHPerDay) / contract.StrikeRate * contract.Size
			settlementAmount = (contract.Size / 2) + profit - exitFee
		} else {
			// Buyer is not in profit
			settlementAmount = (contract.Size / 2) - exitFee
		}
	} else {
		// Seller exit logic - inverse of buyer logic
		if contract.ContractType == CALL && currentBTCPerPHPerDay < contract.StrikeRate {
			// Seller is in profit on a CALL
			profit := (contract.StrikeRate - currentBTCPerPHPerDay) / contract.StrikeRate * contract.Size
			settlementAmount = (contract.Size / 2) + profit - exitFee
		} else if contract.ContractType == PUT && currentBTCPerPHPerDay > contract.StrikeRate {
			// Seller is in profit on a PUT
			profit := (currentBTCPerPHPerDay - contract.StrikeRate) / contract.StrikeRate * contract.Size
			settlementAmount = (contract.Size / 2) + profit - exitFee
		} else {
			// Seller is not in profit
			settlementAmount = (contract.Size / 2) - exitFee
		}
	}

	// 7. Generate early exit transaction using a mutual agreement exit path
	exitPathType := "early_exit"
	tx, err := s.ExecuteExitPath(ctx, contractID, userID, exitPathType)
	if err != nil {
		return nil, fmt.Errorf("failed to execute exit path: %w", err)
	}

	// 8. Update transaction with exit-specific details
	tx.RelatedEntities["exit_fee"] = fmt.Sprintf("%.8f", exitFee)
	tx.RelatedEntities["settlement_amount"] = fmt.Sprintf("%.8f", settlementAmount)
	tx.RelatedEntities["exit_initiator"] = userID
	tx.RelatedEntities["current_btc_ph_day"] = fmt.Sprintf("%.8f", currentBTCPerPHPerDay)

	// 9. Update the transaction in the repository
	if err := s.transactionRepo.Update(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to update exit transaction: %w", err)
	}

	return tx, nil
}

// RolloverContract implements ContractManager.RolloverContract
func (s *contractService) RolloverContract(
	ctx context.Context,
	contractID string,
	newExpiryBlockHeight uint64,
) (*Contract, *Transaction, error) {
	// 1. Get the original contract
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, nil, ErrContractNotFound
	}

	// 2. Validate contract status
	if contract.Status != ACTIVE {
		return nil, nil, ErrInvalidContractStatus
	}

	// 3. Validate new expiry block height
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	s.blockHeight = currentBlockHeight // Update cached block height

	if newExpiryBlockHeight <= currentBlockHeight {
		return nil, nil, fmt.Errorf("%w: new expiry must be in the future", ErrInvalidBlockHeight)
	}

	if newExpiryBlockHeight <= contract.ExpiryBlockHeight {
		return nil, nil, fmt.Errorf("%w: new expiry must be later than current expiry", ErrInvalidBlockHeight)
	}

	// 4. Get current hash rate for pricing
	hashRate, err := s.btcClient.GetBlockHashRate(ctx, currentBlockHeight)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current hash rate: %w", err)
	}
	currentBTCPerPHPerDay := calculateBTCPerPHPerDay(hashRate)

	// 5. Create a new contract with the same parameters but new expiry
	newContract := &Contract{
		ID:                generateUniqueID(),
		ContractType:      contract.ContractType,
		StrikeRate:        contract.StrikeRate, // Could adjust based on market conditions
		ExpiryBlockHeight: newExpiryBlockHeight,
		ExpiryDate:        calculateExpiryDate(newExpiryBlockHeight, currentBlockHeight),
		CreationTime:      time.Now().UTC(),
		Status:            PENDING,
		BuyerID:           contract.BuyerID,
		SellerID:          contract.SellerID,
		Size:              contract.Size,
	}

	// 6. Save the new contract
	if err := s.contractRepo.Create(ctx, newContract); err != nil {
		return nil, nil, fmt.Errorf("failed to create new contract: %w", err)
	}

	// 7. Generate scripts for the new contract
	scripts, err := s.scriptGen.GenerateContractScripts(ctx, newContract)
	if err != nil {
		// Clean up if script generation fails
		_ = s.contractRepo.Delete(ctx, newContract.ID)
		return nil, nil, fmt.Errorf("failed to generate contract scripts: %w", err)
	}

	// 8. Get original VTXOs
	origBuyerVTXO, err := s.vtxoRepo.FindByID(ctx, contract.BuyerVTXO)
	if err != nil || origBuyerVTXO == nil {
		_ = s.contractRepo.Delete(ctx, newContract.ID)
		return nil, nil, fmt.Errorf("failed to get original buyer VTXO: %w", err)
	}

	origSellerVTXO, err := s.vtxoRepo.FindByID(ctx, contract.SellerVTXO)
	if err != nil || origSellerVTXO == nil {
		_ = s.contractRepo.Delete(ctx, newContract.ID)
		return nil, nil, fmt.Errorf("failed to get original seller VTXO: %w", err)
	}

	// 9. Create new VTXOs for the new contract
	newBuyerVTXO, err := s.createContractVTXO(
		ctx, 
		newContract.ID, 
		newContract.BuyerID, 
		newContract.Size/2, 
		scripts["buyerScriptPath"], 
		nil,
	)
	if err != nil {
		_ = s.contractRepo.Delete(ctx, newContract.ID)
		return nil, nil, fmt.Errorf("failed to create new buyer VTXO: %w", err)
	}

	newSellerVTXO, err := s.createContractVTXO(
		ctx, 
		newContract.ID, 
		newContract.SellerID, 
		newContract.Size/2, 
		scripts["sellerScriptPath"], 
		nil,
	)
	if err != nil {
		_ = s.contractRepo.Delete(ctx, newContract.ID)
		_ = s.vtxoRepo.Delete(ctx, newBuyerVTXO.ID)
		return nil, nil, fmt.Errorf("failed to create new seller VTXO: %w", err)
	}

	// 10. Update new contract with VTXO IDs and set to ACTIVE
	newContract.BuyerVTXO = newBuyerVTXO.ID
	newContract.SellerVTXO = newSellerVTXO.ID
	newContract.Status = ACTIVE

	if err := s.contractRepo.Update(ctx, newContract); err != nil {
		return nil, nil, fmt.Errorf("failed to update new contract: %w", err)
	}

	// 11. Mark original contract as ROLLED_OVER and reference new contract
	contract.Status = ROLLED_OVER
	contract.RolledOverToID = newContract.ID

	if err := s.contractRepo.Update(ctx, contract); err != nil {
		return nil, nil, fmt.Errorf("failed to update original contract: %w", err)
	}

	// 12. Mark original VTXOs as inactive
	origBuyerVTXO.IsActive = false
	origSellerVTXO.IsActive = false

	if err := s.vtxoRepo.Update(ctx, origBuyerVTXO); err != nil {
		return nil, nil, fmt.Errorf("failed to update original buyer VTXO: %w", err)
	}

	if err := s.vtxoRepo.Update(ctx, origSellerVTXO); err != nil {
		return nil, nil, fmt.Errorf("failed to update original seller VTXO: %w", err)
	}

	// 13. Record rollover transaction
	tx := &Transaction{
		ID:              generateUniqueID(),
		Type:            CONTRACT_ROLLOVER,
		Timestamp:       time.Now().UTC(),
		ContractID:      contractID,
		UserIDs:         []string{contract.BuyerID, contract.SellerID},
		Amount:          contract.Size,
		BTCPerPHPerDay:  currentBTCPerPHPerDay,
		BlockHeight:     currentBlockHeight,
		RelatedEntities: map[string]string{
			"original_contract_id":  contractID,
			"new_contract_id":       newContract.ID,
			"original_expiry":       fmt.Sprintf("%d", contract.ExpiryBlockHeight),
			"new_expiry":            fmt.Sprintf("%d", newExpiryBlockHeight),
			"original_buyer_vtxo":   contract.BuyerVTXO,
			"original_seller_vtxo":  contract.SellerVTXO,
			"new_buyer_vtxo":        newContract.BuyerVTXO,
			"new_seller_vtxo":       newContract.SellerVTXO,
		},
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return nil, nil, fmt.Errorf("failed to record rollover transaction: %w", err)
	}

	return newContract, tx, nil
}

// validateContractParameters implements a more robust validation of contract parameters
func (s *contractService) validateContractParameters(
	ctx context.Context,
	contractType hashperp.ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) error {
	// 1. Validate contract type
	if contractType != hashperp.CALL && contractType != hashperp.PUT {
		return fmt.Errorf("%w: contract type must be CALL or PUT, got %s", ErrInvalidParameters, contractType)
	}

	// 2. Validate strike rate - must be positive and reasonable
	if strikeRate <= 0 {
		return fmt.Errorf("%w: strike rate must be positive", ErrInvalidParameters)
	}
	
	// Add upper bound to prevent unreasonable values that might indicate errors
	if strikeRate > 1000 {
		return fmt.Errorf("%w: strike rate too high (> 1000), please verify", ErrInvalidParameters)
	}

	// 3. Validate size - must be positive and within reasonable bounds
	if size <= 0 {
		return fmt.Errorf("%w: size must be positive", ErrInvalidParameters)
	}
	
	// Add minimum size to prevent dust contracts
	const minSize = 0.001 // 0.001 BTC
	if size < minSize {
		return fmt.Errorf("%w: size must be at least %v BTC", ErrInvalidParameters, minSize)
	}
	
	// Add upper bound to prevent unreasonable values
	const maxSize = 100 // 100 BTC
	if size > maxSize {
		return fmt.Errorf("%w: size exceeds maximum allowed (%v BTC)", ErrInvalidParameters, maxSize)
	}

	// 4. Validate expiry block height
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block height: %w", err)
	}
	s.blockHeight = currentBlockHeight // Update cached block height

	if expiryBlockHeight <= currentBlockHeight {
		return fmt.Errorf("%w: expiry block height must be in the future", ErrInvalidBlockHeight)
	}

	// Validate minimum contract duration (e.g., at least 100 blocks ~16 hours)
	const minBlockDuration = 100
	if expiryBlockHeight < currentBlockHeight+minBlockDuration {
		return fmt.Errorf("%w: contract duration too short, minimum %d blocks", ErrInvalidParameters, minBlockDuration)
	}
	
	// Validate maximum contract duration (e.g., at most 52560 blocks ~1 year)
	const maxBlockDuration = 52560 // ~1 year at 10 min per block
	if expiryBlockHeight > currentBlockHeight+maxBlockDuration {
		return fmt.Errorf("%w: contract duration too long, maximum %d blocks (~1 year)", ErrInvalidParameters, maxBlockDuration)
	}

	return nil
}
