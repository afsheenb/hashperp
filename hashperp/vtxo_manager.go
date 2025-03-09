package hashperp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// These constants define the possible states for a contract during settlement
const (
	SETTLEMENT_PENDING     ContractStatus = "SETTLEMENT_PENDING"
	SETTLEMENT_IN_PROGRESS ContractStatus = "SETTLEMENT_IN_PROGRESS"
	COMPLETED              ContractStatus = "COMPLETED"
	CLOSE_TO_EXPIRY        ContractStatus = "CLOSE_TO_EXPIRY"
)



// Append to existing hashperp/vtxo_manager.go

// Updated VTXOService struct to include user and pre-signed exit repositories
type vtxoService struct {
	vtxoRepo         VTXORepository
	contractRepo     ContractRepository
	transactionRepo  TransactionRepository
	scriptGen        ScriptGenerator
	btcClient        BitcoinClient
	userRepo         UserRepository
	preSignedExitRepo PreSignedExitRepository
}

// NewVTXOService creates a new VTXO service
func NewVTXOService(
	vtxoRepo VTXORepository,
	contractRepo ContractRepository,
	transactionRepo TransactionRepository,
	scriptGen ScriptGenerator,
	btcClient BitcoinClient,
	userRepo UserRepository,
	preSignedExitRepo PreSignedExitRepository,
) VTXOManager {
	return &vtxoService{
		vtxoRepo:         vtxoRepo,
		contractRepo:     contractRepo,
		transactionRepo:  transactionRepo,
		scriptGen:        scriptGen,
		btcClient:        btcClient,
		userRepo:         userRepo,
		preSignedExitRepo: preSignedExitRepo,
	}
}

// CreateVTXO implements VTXOManager.CreateVTXO
func (s *vtxoService) CreateVTXO(
	ctx context.Context,
	contractID string,
	ownerID string,
	amount float64,
	scriptPath string,
	signatureData []byte,
) (*VTXO, error) {
	// 1. Validate contract exists
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 2. Validate amount is positive
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	// 3. Create the VTXO
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

	// 4. Save the VTXO
	if err := s.vtxoRepo.Create(ctx, vtxo); err != nil {
		return nil, fmt.Errorf("failed to create VTXO: %w", err)
	}

	return vtxo, nil
}

// GetVTXO implements VTXOManager.GetVTXO
func (s *vtxoService) GetVTXO(ctx context.Context, vtxoID string) (*VTXO, error) {
	vtxo, err := s.vtxoRepo.FindByID(ctx, vtxoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXO: %w", err)
	}
	if vtxo == nil {
		return nil, ErrVTXONotFound
	}
	return vtxo, nil
}

// GetVTXOsByContract implements VTXOManager.GetVTXOsByContract
func (s *vtxoService) GetVTXOsByContract(ctx context.Context, contractID string) ([]*VTXO, error) {
	// 1. Validate contract exists
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 2. Get VTXOs for the contract
	vtxos, err := s.vtxoRepo.FindByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXOs by contract: %w", err)
	}

	return vtxos, nil
}

// GetVTXOsByUser implements VTXOManager.GetVTXOsByUser
func (s *vtxoService) GetVTXOsByUser(ctx context.Context, userID string, onlyActive bool) ([]*VTXO, error) {
	vtxos, err := s.vtxoRepo.FindByUser(ctx, userID, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXOs by user: %w", err)
	}
	return vtxos, nil
}

// SwapVTXO implements VTXOManager.SwapVTXO
// This is the core functionality for dynamic contract participation
func (s *vtxoService) SwapVTXO(
	ctx context.Context,
	vtxoID string,
	newOwnerID string,
	newSignatureData []byte,
) (*VTXO, *Transaction, error) {
	// 1. Get the existing VTXO
	vtxo, err := s.vtxoRepo.FindByID(ctx, vtxoID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get VTXO: %w", err)
	}
	if vtxo == nil {
		return nil, nil, ErrVTXONotFound
	}

	// 2. Validate the VTXO is active
	if !vtxo.IsActive {
		return nil, nil, ErrVTXONotActive
	}

	// 3. Get the associated contract
	contract, err := s.contractRepo.FindByID(ctx, vtxo.ContractID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, nil, ErrContractNotFound
	}

	// 4. Validate contract status
	if contract.Status != ACTIVE {
		return nil, nil, ErrInvalidContractStatus
	}

	// 5. Verify signature data (in a real implementation, this would validate against the contract terms)
	// For this example, we'll just check if the signature data is not empty
	if newSignatureData == nil || len(newSignatureData) == 0 {
		return nil, nil, ErrInvalidSignature
	}

	// 6. Create a new VTXO with the new owner
	newVTXO := &VTXO{
		ID:                generateUniqueID(),
		ContractID:        vtxo.ContractID,
		OwnerID:           newOwnerID,
		Amount:            vtxo.Amount,
		ScriptPath:        vtxo.ScriptPath,
		CreationTimestamp: time.Now().UTC(),
		SignatureData:     newSignatureData,
		SwappedFromID:     vtxo.ID,
		IsActive:          true,
	}

	// 7. Save the new VTXO
	if err := s.vtxoRepo.Create(ctx, newVTXO); err != nil {
		return nil, nil, fmt.Errorf("failed to create new VTXO: %w", err)
	}

	// 8. Mark the old VTXO as inactive
	oldVTXO := vtxo // Keep a reference to the old VTXO
	vtxo.IsActive = false
	if err := s.vtxoRepo.Update(ctx, vtxo); err != nil {
		// If we fail to update the old VTXO, delete the new one to maintain consistency
		_ = s.vtxoRepo.Delete(ctx, newVTXO.ID)
		return nil, nil, fmt.Errorf("failed to update old VTXO: %w", err)
	}

	// 9. Update the contract with the new VTXO ID
	var positionType string
	if contract.BuyerVTXO == vtxoID {
		contract.BuyerVTXO = newVTXO.ID
		contract.BuyerID = newOwnerID
		positionType = "buyer"
	} else if contract.SellerVTXO == vtxoID {
		contract.SellerVTXO = newVTXO.ID
		contract.SellerID = newOwnerID
		positionType = "seller"
	} else {
		// This should never happen if our data integrity is maintained
		_ = s.vtxoRepo.Delete(ctx, newVTXO.ID)
		oldVTXO.IsActive = true
		_ = s.vtxoRepo.Update(ctx, oldVTXO)
		return nil, nil, errors.New("VTXO is not associated with this contract's buyer or seller")
	}

	if err := s.contractRepo.Update(ctx, contract); err != nil {
		// If we fail to update the contract, revert the VTXO changes
		_ = s.vtxoRepo.Delete(ctx, newVTXO.ID)
		oldVTXO.IsActive = true
		_ = s.vtxoRepo.Update(ctx, oldVTXO)
		return nil, nil, fmt.Errorf("failed to update contract: %w", err)
	}

	// 10. Record the swap transaction
	tx := &Transaction{
		ID:         generateUniqueID(),
		Type:       VTXO_SWAP,
		Timestamp:  time.Now().UTC(),
		ContractID: contract.ID,
		UserIDs:    []string{vtxo.OwnerID, newOwnerID},
		Amount:     vtxo.Amount,
		RelatedEntities: map[string]string{
			"old_vtxo":       vtxo.ID,
			"new_vtxo":       newVTXO.ID,
			"position_type":  positionType,
			"old_owner_id":   vtxo.OwnerID,
			"new_owner_id":   newOwnerID,
		},
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		// If recording the transaction fails, we'll still proceed with the swap
		// but log the error
		fmt.Printf("failed to record swap transaction: %v\n", err)
	}

	return newVTXO, tx, nil
}

// CreatePresignedExitTransaction implements VTXOManager.CreatePresignedExitTransaction
func (s *vtxoService) CreatePresignedExitTransaction(
	ctx context.Context,
	vtxoID string,
	signatureData []byte,
) (string, error) {
	// 1. Get the VTXO
	vtxo, err := s.vtxoRepo.FindByID(ctx, vtxoID)
	if err != nil {
		return "", fmt.Errorf("failed to get VTXO: %w", err)
	}
	if vtxo == nil {
		return "", ErrVTXONotFound
	}

	// 2. Validate the VTXO is active
	if !vtxo.IsActive {
		return "", ErrVTXONotActive
	}

	// 3. Get the associated contract
	contract, err := s.contractRepo.FindByID(ctx, vtxo.ContractID)
	if err != nil {
		return "", fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return "", ErrContractNotFound
	}

	// 4. Validate contract status
	if contract.Status != ACTIVE {
		return "", ErrInvalidContractStatus
	}

	// 5. Validate signature data is present
	if signatureData == nil || len(signatureData) == 0 {
		return "", ErrInvalidSignature
	}
	
	// 6. Construct verification message
	verificationMessage := fmt.Sprintf("exit:%s:%s:%d", 
		vtxoID, vtxo.OwnerID, contract.ExpiryBlockHeight)
	messageBytes := []byte(verificationMessage)
	
	// 7. Get user's public key from repository
	userRepo := s.userRepo
	pubKey, err := userRepo.GetPublicKey(ctx, vtxo.OwnerID)
	if err != nil {
		return "", fmt.Errorf("failed to get user public key: %w", err)
	}
	
	// 8. Verify the signature
	isValid, err := s.btcClient.ValidateSignature(ctx, messageBytes, signatureData, pubKey)
	if err != nil {
		return "", fmt.Errorf("signature validation error: %w", err)
	}
	
	if !isValid {
		return "", ErrInvalidSignature
	}

	// 9. Generate the exit script
	exitScript, err := s.scriptGen.GenerateExitScript(ctx, vtxo.ScriptPath, signatureData)
	if err != nil {
		return "", fmt.Errorf("failed to generate exit script: %w", err)
	}

	// 10. Generate a unique ID for the pre-signed transaction
	exitTxID := generateExitTransactionID(vtxo, contract)

	// 11. Store the pre-signed exit transaction
	preSignedExit := &PreSignedExit{
		ID:           generateUniqueID(),
		VTXOID:       vtxoID,
		ContractID:   vtxo.ContractID,
		UserID:       vtxo.OwnerID,
		ExitTxHex:    exitScript,
		CreationTime: time.Now().UTC(),
		IsUsed:       false,
	}

	if err := s.preSignedExitRepo.Create(ctx, preSignedExit); err != nil {
		return "", fmt.Errorf("failed to store pre-signed exit transaction: %w", err)
	}
	
	// 12. Record the transaction
	tx := &Transaction{
		ID:         generateUniqueID(),
		Type:       EXIT_PATH_EXECUTION,
		Timestamp:  time.Now().UTC(),
		ContractID: contract.ID,
		UserIDs:    []string{vtxo.OwnerID},
		Amount:     vtxo.Amount,
		Status:     "PREPARED",
		RelatedEntities: map[string]string{
			"vtxo_id":     vtxo.ID,
			"exit_type":   "pre_signed",
			"exit_tx_id":  exitTxID,
			"owner_id":    vtxo.OwnerID,
		},
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return "", fmt.Errorf("failed to record pre-signed exit transaction: %w", err)
	}

	return exitTxID, nil
}

// ExecuteVTXOSweep implements VTXOManager.ExecuteVTXOSweep
func (s *vtxoService) ExecuteVTXOSweep(
	ctx context.Context,
	vtxoID string,
) (*Transaction, error) {
	// 1. Get the VTXO
	vtxo, err := s.vtxoRepo.FindByID(ctx, vtxoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXO: %w", err)
	}
	if vtxo == nil {
		return nil, ErrVTXONotFound
	}

	// 2. Validate the VTXO is active
	if !vtxo.IsActive {
		return nil, ErrVTXONotActive
	}

	// 3. Get the associated contract
	contract, err := s.contractRepo.FindByID(ctx, vtxo.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 4. Validate the contract status allows for exits
	// Typically, only ACTIVE or SETTLEMENT_PENDING contracts can be exited
	if contract.Status != ACTIVE && contract.Status != SETTLEMENT_PENDING {
		return nil, ErrInvalidContractStatus
	}

	// 5. Get current block height for validation
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}

	// 6. Validate that VTXO sweep is allowed
	// For emergency exits, we typically require the contract to be close to expiry
	// or a certain timeout to have passed
	if currentBlockHeight < contract.ExpiryBlockHeight - 144 { // 144 blocks = ~1 day
		// Check if the contract has a special flag or status that allows early exit
		if contract.Status != SETTLEMENT_PENDING {
            return nil, errors.New("VTXO sweep is only allowed within 1 day of expiry or for settlement pending contracts")
        }
	}

	// 7. Generate the emergency exit script based on the VTXO's script path
	exitScript, err := s.scriptGen.GenerateExitScript(ctx, vtxo.ScriptPath, vtxo.SignatureData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate exit script: %w", err)
	}

	// 8. Create and broadcast a Bitcoin transaction to execute the VTXO sweep
	// This would create and sign a transaction that sends the funds to the VTXO owner's address
	txHash, err := s.btcClient.BroadcastTransaction(ctx, exitScript)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast exit transaction: %w", err)
	}

	// 9. Mark the VTXO as inactive
	vtxo.IsActive = false
	vtxo.ExitTxHash = txHash
	vtxo.ExitTimestamp = time.Now().UTC()
	
	if err := s.vtxoRepo.Update(ctx, vtxo); err != nil {
		// If we fail to update the VTXO, log the error but don't fail the operation
		// since the Bitcoin transaction has already been broadcast
		fmt.Printf("failed to update VTXO after sweep: %v\n", err)
		// In a real implementation, this should be handled by a reconciliation process
	}

	// 10. Update the contract if necessary
	// If this VTXO is part of the contract's core positions, update accordingly
	if contract.BuyerVTXO == vtxoID || contract.SellerVTXO == vtxoID {
		if contract.BuyerVTXO == vtxoID {
			contract.BuyerVTXO = ""
			contract.BuyerExited = true
			contract.BuyerExitTxHash = txHash
		} else {
			contract.SellerVTXO = ""
			contract.SellerExited = true
			contract.SellerExitTxHash = txHash
		}
		
		// If both parties have exited, mark the contract as completed
		if contract.BuyerExited && contract.SellerExited {
			contract.Status = COMPLETED
			contract.CompletionTimestamp = time.Now().UTC()
		} else if contract.Status != SETTLEMENT_PENDING {
			// Otherwise, mark it as pending settlement if not already
			contract.Status = SETTLEMENT_PENDING
		}
		
		if err := s.contractRepo.Update(ctx, contract); err != nil {
			fmt.Printf("failed to update contract after VTXO sweep: %v\n", err)
			// In a real implementation, this should be handled by a reconciliation process
		}
	}

	// 11. Record the sweep transaction
	tx := &Transaction{
		ID:         generateUniqueID(),
		Type:       EXIT_PATH_EXECUTION,
		Timestamp:  time.Now().UTC(),
		ContractID: contract.ID,
		UserIDs:    []string{vtxo.OwnerID},
		TxHash:     txHash,
		Amount:     vtxo.Amount,
		BlockHeight: currentBlockHeight,
		RelatedEntities: map[string]string{
			"vtxo_id":     vtxo.ID,
			"exit_type":   "vtxo_sweep",
			"owner_id":    vtxo.OwnerID,
			"btc_tx_hash": txHash,
		},
		Status:     "COMPLETED",
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		// Log the error but don't fail the operation since the Bitcoin transaction
		// has already been broadcast and the VTXO updated
		fmt.Printf("failed to record sweep transaction: %v\n", err)
	}

	return tx, nil
}

// Helper function to generate a unique ID for an exit transaction
func generateExitTransactionID(vtxo *VTXO, contract *Contract) string {
	// Create a unique identifier that's stable across implementations
	data := fmt.Sprintf("exit_%s_%s_%s_%d",
		vtxo.ID,
		vtxo.OwnerID,
		contract.ID,
		time.Now().UnixNano())
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// VerifyVTXOOwnership implements VTXOManager.VerifyVTXOOwnership
func (s *vtxoService) VerifyVTXOOwnership(
	ctx context.Context,
	vtxoID string,
	ownerID string,
) (bool, error) {
	// 1. Get the VTXO
	vtxo, err := s.vtxoRepo.FindByID(ctx, vtxoID)
	if err != nil {
		return false, fmt.Errorf("failed to get VTXO: %w", err)
	}
	if vtxo == nil {
		return false, ErrVTXONotFound
	}
	
	// 2. Verify the owner matches
	return vtxo.OwnerID == ownerID, nil
}

// GetVTXOHistory implements VTXOManager.GetVTXOHistory
func (s *vtxoService) GetVTXOHistory(
	ctx context.Context,
	vtxoID string,
) ([]*VTXO, error) {
	// 1. Get the current VTXO
	current, err := s.vtxoRepo.FindByID(ctx, vtxoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXO: %w", err)
	}
	if current == nil {
		return nil, ErrVTXONotFound
	}
	
	// 2. Initialize history with the current VTXO
	history := []*VTXO{current}
	
	// 3. Trace back through the swap chain
	var swapID = current.SwappedFromID
	for swapID != "" {
		// Get the previous VTXO in the chain
		prev, err := s.vtxoRepo.FindByID(ctx, swapID)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous VTXO in swap chain: %w", err)
		}
		if prev == nil {
			// If the previous VTXO doesn't exist (data integrity issue),
			// break the chain but return what we have so far
			break
		}
		
		// Add to history and continue tracing back
		history = append(history, prev)
		swapID = prev.SwappedFromID
	}
	
	// 4. Reverse the history so it's in chronological order
	// (earliest to latest)
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	
	return history, nil
}

// RolloverVTXO implements VTXOManager.RolloverVTXO
// This function allows rolling over a VTXO from an expiring contract to a new one
func (s *vtxoService) RolloverVTXO(
	ctx context.Context,
	oldVTXOID string,
	newContractID string,
	newSignatureData []byte,
) (*VTXO, *Transaction, error) {
	// 1. Get the original VTXO
	oldVTXO, err := s.vtxoRepo.FindByID(ctx, oldVTXOID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old VTXO: %w", err)
	}
	if oldVTXO == nil {
		return nil, nil, ErrVTXONotFound
	}
	
	// 2. Validate the VTXO is active
	if !oldVTXO.IsActive {
		return nil, nil, ErrVTXONotActive
	}
	
	// 3. Get the old contract
	oldContract, err := s.contractRepo.FindByID(ctx, oldVTXO.ContractID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old contract: %w", err)
	}
	if oldContract == nil {
		return nil, nil, ErrContractNotFound
	}
	
	// 4. Validate old contract status
	// Rollover is typically only allowed for contracts that are active or close to expiration
	if oldContract.Status != ACTIVE && oldContract.Status != CLOSE_TO_EXPIRY {
		return nil, nil, ErrInvalidContractStatus
	}
	
	// 5. Get the new contract
	newContract, err := s.contractRepo.FindByID(ctx, newContractID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new contract: %w", err)
	}
	if newContract == nil {
		return nil, nil, ErrContractNotFound
	}
	
	// 6. Validate new contract status
	if newContract.Status != ACTIVE {
		return nil, nil, ErrInvalidContractStatus
	}
	
	// 7. Determine position type in the old contract
	var positionType string
	if oldContract.BuyerVTXO == oldVTXOID {
		positionType = "buyer"
		// Verify the buyer position isn't already filled in the new contract
		if newContract.BuyerVTXO != "" {
			return nil, nil, errors.New("new contract already has a buyer")
		}
	} else if oldContract.SellerVTXO == oldVTXOID {
		positionType = "seller"
		// Verify the seller position isn't already filled in the new contract
		if newContract.SellerVTXO != "" {
			return nil, nil, errors.New("new contract already has a seller")
		}
	} else {
		return nil, nil, errors.New("VTXO is not associated with this contract's buyer or seller")
	}
	
	// 8. Verify signature data
	if newSignatureData == nil || len(newSignatureData) == 0 {
		return nil, nil, ErrInvalidSignature
	}
	
	// 9. Create a new VTXO for the new contract
	newVTXO := &VTXO{
		ID:                generateUniqueID(),
		ContractID:        newContractID,
		OwnerID:           oldVTXO.OwnerID,
		Amount:            oldVTXO.Amount, // Typically the amount would be adjusted based on new contract terms
		ScriptPath:        oldVTXO.ScriptPath, // This might need to be regenerated for the new contract
		CreationTimestamp: time.Now().UTC(),
		SignatureData:     newSignatureData,
		RolledFromID:      oldVTXO.ID,
		IsActive:          true,
	}
	
	// 10. Save the new VTXO
	if err := s.vtxoRepo.Create(ctx, newVTXO); err != nil {
		return nil, nil, fmt.Errorf("failed to create new VTXO: %w", err)
	}
	
	// 11. Mark the old VTXO as inactive
	oldVTXO.IsActive = false
	oldVTXO.RolledToID = newVTXO.ID
	if err := s.vtxoRepo.Update(ctx, oldVTXO); err != nil {
		// If we fail to update the old VTXO, delete the new one to maintain consistency
		_ = s.vtxoRepo.Delete(ctx, newVTXO.ID)
		return nil, nil, fmt.Errorf("failed to update old VTXO: %w", err)
	}
	
	// 12. Update the new contract with the new VTXO ID
	if positionType == "buyer" {
		newContract.BuyerVTXO = newVTXO.ID
		newContract.BuyerID = oldVTXO.OwnerID
	} else {
		newContract.SellerVTXO = newVTXO.ID
		newContract.SellerID = oldVTXO.OwnerID
	}
	
	if err := s.contractRepo.Update(ctx, newContract); err != nil {
		// If we fail to update the contract, revert the VTXO changes
		_ = s.vtxoRepo.Delete(ctx, newVTXO.ID)
		oldVTXO.IsActive = true
		oldVTXO.RolledToID = ""
		_ = s.vtxoRepo.Update(ctx, oldVTXO)
		return nil, nil, fmt.Errorf("failed to update new contract: %w", err)
	}
	
	// 13. Record the rollover transaction
	tx := &Transaction{
		ID:         generateUniqueID(),
		Type:       CONTRACT_ROLLOVER,
		Timestamp:  time.Now().UTC(),
		ContractID: newContract.ID,
		UserIDs:    []string{oldVTXO.OwnerID},
		Amount:     newVTXO.Amount,
		RelatedEntities: map[string]string{
			"old_vtxo":        oldVTXO.ID,
			"new_vtxo":        newVTXO.ID,
			"old_contract_id": oldContract.ID,
			"new_contract_id": newContract.ID,
			"position_type":   positionType,
			"owner_id":        oldVTXO.OwnerID,
		},
		Status: "COMPLETED",
	}
	
	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		// If recording the transaction fails, we'll still proceed with the rollover
		// but log the error
		fmt.Printf("failed to record rollover transaction: %v\n", err)
	}
	
	return newVTXO, tx, nil
}

// GetActiveVTXOsCount implements VTXOManager.GetActiveVTXOsCount
func (s *vtxoService) GetActiveVTXOsCount(
	ctx context.Context,
	contractID string,
) (int, error) {
	// 1. Validate contract exists
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return 0, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return 0, ErrContractNotFound
	}
	
	// 2. Get the count of active VTXOs for this contract
	count, err := s.vtxoRepo.CountActiveByContract(ctx, contractID)
	if err != nil {
		return 0, fmt.Errorf("failed to count active VTXOs: %w", err)
	}
	
	return count, nil
}

// GetVTXOsForSettlement implements VTXOManager.GetVTXOsForSettlement
func (s *vtxoService) GetVTXOsForSettlement(
	ctx context.Context,
	contractID string,
) ([]*VTXO, error) {
	// 1. Validate contract exists
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}
	
	// 2. Validate contract status is appropriate for settlement
	if contract.Status != ACTIVE && contract.Status != SETTLEMENT_PENDING && contract.Status != SETTLEMENT_IN_PROGRESS {
		return nil, ErrInvalidContractStatus
	}
	
	// 3. Get all active VTXOs for this contract
	vtxos, err := s.vtxoRepo.FindActiveByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active VTXOs for settlement: %w", err)
	}
	
	return vtxos, nil
}
