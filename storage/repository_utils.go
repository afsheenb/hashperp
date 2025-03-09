package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashperp/hashperp"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Helper functions for model conversion must be updated to reflect all new fields

// Updated convertDBContractToContract to include all fields
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

// Updated convertDBVTXOToVTXO to include all fields
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

// Updated convertDBOrderToOrder to include all fields
func convertDBOrderToOrder(dbOrder *DBOrder) *hashperp.Order {
	order := &hashperp.Order{
		ID:                dbOrder.ID,
		UserID:            dbOrder.UserID,
		OrderType:         hashperp.OrderType(dbOrder.OrderType),
		ContractType:      hashperp.ContractType(dbOrder.ContractType),
		StrikeRate:        dbOrder.StrikeRate,
		ExpiryBlockHeight: dbOrder.ExpiryBlockHeight,
		ExpiryDate:        dbOrder.ExpiryDate,
		Size:              dbOrder.Size,
		Status:            hashperp.OrderStatus(dbOrder.Status),
		CreationTime:      dbOrder.CreationTime,
	}

	if dbOrder.MatchedOrderID.Valid {
		order.MatchedOrderID = dbOrder.MatchedOrderID.String
	}

	if dbOrder.ResultingContractID.Valid {
		order.ResultingContractID = dbOrder.ResultingContractID.String
	}

	return order
}

// Updated convertDBSwapOfferToSwapOffer to include all fields
func convertDBSwapOfferToSwapOffer(dbSwapOffer *DBSwapOffer) (*hashperp.SwapOffer, error) {
	swapOffer := &hashperp.SwapOffer{
		ID:           dbSwapOffer.ID,
		OfferorID:    dbSwapOffer.OfferorID,
		VTXOID:       dbSwapOffer.VTXOID,
		ContractID:   dbSwapOffer.ContractID,
		OfferedRate:  dbSwapOffer.OfferedRate,
		CreationTime: dbSwapOffer.CreationTime,
		ExpiryTime:   dbSwapOffer.ExpiryTime,
		Status:       dbSwapOffer.Status,
	}
	
	if dbSwapOffer.AcceptorID.Valid {
		swapOffer.AcceptorID = dbSwapOffer.AcceptorID.String
	}
	
	if dbSwapOffer.TargetUserID.Valid {
		swapOffer.TargetUserID = dbSwapOffer.TargetUserID.String
	}
	
	if dbSwapOffer.SwapType.Valid {
		swapOffer.SwapType = dbSwapOffer.SwapType.String
	}
	
	// Parse related entities if set
	if dbSwapOffer.RelatedEntities != nil {
		var relatedEntities map[string]string
		err := json.Unmarshal(dbSwapOffer.RelatedEntities, &relatedEntities)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal related entities: %w", err)
		}
		swapOffer.RelatedEntities = relatedEntities
	} else {
		swapOffer.RelatedEntities = make(map[string]string)
	}
	
	return swapOffer, nil
}

// Updated convertDBTransactionToTransaction to include all fields
func convertDBTransactionToTransaction(dbTransaction *DBTransaction) (*hashperp.Transaction, error) {
	tx := &hashperp.Transaction{
		ID:             dbTransaction.ID,
		Type:           hashperp.TransactionType(dbTransaction.Type),
		Timestamp:      dbTransaction.Timestamp,
		ContractID:     dbTransaction.ContractID,
		UserIDs:        []string(dbTransaction.UserIDs),
		TxHash:         dbTransaction.TxHash,
		Amount:         dbTransaction.Amount,
		BTCPerPHPerDay: dbTransaction.BTCPerPHPerDay,
		BlockHeight:    dbTransaction.BlockHeight,
		Status:         dbTransaction.Status,
	}
	
	// Parse related entities if set
	if dbTransaction.RelatedEntities != nil {
		var relatedEntities map[string]string
		err := json.Unmarshal(dbTransaction.RelatedEntities, &relatedEntities)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal related entities: %w", err)
		}
		tx.RelatedEntities = relatedEntities
	} else {
		tx.RelatedEntities = make(map[string]string)
	}
	
	return tx, nil
}

// Updated convertContractToDBContract to include all fields
func convertContractToDBContract(contract *hashperp.Contract) *DBContract {
	dbContract := &DBContract{
		ID:                contract.ID,
		ContractType:      string(contract.ContractType),
		StrikeRate:        contract.StrikeRate,
		ExpiryBlockHeight: contract.ExpiryBlockHeight,
		ExpiryDate:        contract.ExpiryDate,
		CreationTime:      contract.CreationTime,
		Status:            string(contract.Status),
		BuyerID:           contract.BuyerID,
		SellerID:          contract.SellerID,
		Size:              contract.Size,
		BuyerVTXO:         contract.BuyerVTXO,
		SellerVTXO:        contract.SellerVTXO,
		BuyerExited:       contract.BuyerExited,
		SellerExited:      contract.SellerExited,
	}

	// Set nullable fields
	if contract.SettlementTx != "" {
		dbContract.SettlementTx = sql.NullString{
			String: contract.SettlementTx,
			Valid:  true,
		}
	}

	if contract.SettlementRate != 0 {
		dbContract.SettlementRate = sql.NullFloat64{
			Float64: contract.SettlementRate,
			Valid:   true,
		}
	}

	if contract.RolledOverToID != "" {
		dbContract.RolledOverToID = sql.NullString{
			String: contract.RolledOverToID,
			Valid:  true,
		}
	}
	
	if !contract.CompletionTimestamp.IsZero() {
		dbContract.CompletionTimestamp = sql.NullTime{
			Time:  contract.CompletionTimestamp,
			Valid: true,
		}
	}
	
	if contract.BuyerExitTxHash != "" {
		dbContract.BuyerExitTxHash = sql.NullString{
			String: contract.BuyerExitTxHash,
			Valid:  true,
		}
	}
	
	if contract.SellerExitTxHash != "" {
		dbContract.SellerExitTxHash = sql.NullString{
			String: contract.SellerExitTxHash,
			Valid:  true,
		}
	}

	return dbContract
}

// Updated convertVTXOToDBVTXO to include all fields
func convertVTXOToDBVTXO(vtxo *hashperp.VTXO) *DBVTXO {
	dbVTXO := &DBVTXO{
		ID:                vtxo.ID,
		ContractID:        vtxo.ContractID,
		OwnerID:           vtxo.OwnerID,
		Amount:            vtxo.Amount,
		ScriptPath:        vtxo.ScriptPath,
		CreationTimestamp: vtxo.CreationTimestamp,
		SignatureData:     vtxo.SignatureData,
		IsActive:          vtxo.IsActive,
	}

	if vtxo.SwappedFromID != "" {
		dbVTXO.SwappedFromID = sql.NullString{
			String: vtxo.SwappedFromID,
			Valid:  true,
		}
	}
	
	if vtxo.RolledFromID != "" {
		dbVTXO.RolledFromID = sql.NullString{
			String: vtxo.RolledFromID,
			Valid:  true,
		}
	}
	
	if vtxo.RolledToID != "" {
		dbVTXO.RolledToID = sql.NullString{
			String: vtxo.RolledToID,
			Valid:  true,
		}
	}
	
	if vtxo.ExitTxHash != "" {
		dbVTXO.ExitTxHash = sql.NullString{
			String: vtxo.ExitTxHash,
			Valid:  true,
		}
	}
	
	if !vtxo.ExitTimestamp.IsZero() {
		dbVTXO.ExitTimestamp = sql.NullTime{
			Time:  vtxo.ExitTimestamp,
			Valid: true,
		}
	}

	return dbVTXO
}

// Additional repository function to support the new VTXO methods

// FindActiveByContract finds all active VTXOs for a contract
func (r *PostgresVTXORepository) FindActiveByContract(ctx context.Context, contractID string) ([]*hashperp.VTXO, error) {
	var dbVTXOs []DBVTXO
	result := r.db.WithContext(ctx).
		Where("contract_id = ? AND is_active = true", contractID).
		Find(&dbVTXOs)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find active VTXOs: %w", result.Error)
	}
	
	vtxos := make([]*hashperp.VTXO, len(dbVTXOs))
	for i, dbVTXO := range dbVTXOs {
		vtxos[i] = convertDBVTXOToVTXO(&dbVTXO)
	}
	
	return vtxos, nil
}

// CountActiveByContract counts active VTXOs for a contract
func (r *PostgresVTXORepository) CountActiveByContract(ctx context.Context, contractID string) (int, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&DBVTXO{}).
		Where("contract_id = ? AND is_active = true", contractID).
		Count(&count)
	
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count active VTXOs: %w", result.Error)
	}
	
	return int(count), nil
}

// PostgresTransactionRepository implements the TransactionRepository interface
type PostgresTransactionRepository struct {
	db *gorm.DB
}

// NewPostgresTransactionRepository creates a new PostgreSQL-based transaction repository
func NewPostgresTransactionRepository(db *gorm.DB) hashperp.TransactionRepository {
	return &PostgresTransactionRepository{
		db: db,
	}
}

// Create creates a new transaction
func (r *PostgresTransactionRepository) Create(ctx context.Context, tx *hashperp.Transaction) error {
	// Convert related entities to JSON
	var relatedEntitiesJSON json.RawMessage
	if len(tx.RelatedEntities) > 0 {
		entitiesBytes, err := json.Marshal(tx.RelatedEntities)
		if err != nil {
			return fmt.Errorf("failed to marshal related entities: %w", err)
		}
		relatedEntitiesJSON = entitiesBytes
	}
	
	dbTransaction := &DBTransaction{
		ID:              tx.ID,
		Type:            string(tx.Type),
		Timestamp:       tx.Timestamp,
		ContractID:      tx.ContractID,
		UserIDs:         pq.StringArray(tx.UserIDs),
		TxHash:          tx.TxHash,
		Amount:          tx.Amount,
		BTCPerPHPerDay:  tx.BTCPerPHPerDay,
		BlockHeight:     tx.BlockHeight,
		Status:          tx.Status,
		RelatedEntities: relatedEntitiesJSON,
	}

	result := r.db.WithContext(ctx).Create(dbTransaction)
	if result.Error != nil {
		return fmt.Errorf("failed to create transaction: %w", result.Error)
	}

	return nil
}

// FindByID retrieves a transaction by ID
func (r *PostgresTransactionRepository) FindByID(ctx context.Context, id string) (*hashperp.Transaction, error) {
	var dbTransaction DBTransaction
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&dbTransaction)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find transaction: %w", result.Error)
	}

	return convertDBTransactionToTransaction(&dbTransaction)
}

// FindByUser retrieves all transactions for a specific user
func (r *PostgresTransactionRepository) FindByUser(ctx context.Context, userID string, types []hashperp.TransactionType) ([]*hashperp.Transaction, error) {
	var dbTransactions []DBTransaction
	
	query := r.db.WithContext(ctx).Where("? = ANY(user_ids)", userID)
	
	if len(types) > 0 {
		// Convert TransactionType to string for the query
		var typeStrings []string
		for _, t := range types {
			typeStrings = append(typeStrings, string(t))
		}
		query = query.Where("type IN ?", typeStrings)
	}
	
	result := query.Find(&dbTransactions)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find transactions by user: %w", result.Error)
	}

	transactions := make([]*hashperp.Transaction, 0, len(dbTransactions))
	for _, dbTx := range dbTransactions {
		tx, err := convertDBTransactionToTransaction(&dbTx)
		if err != nil {
			// Log the error but continue with other transactions
			fmt.Printf("error converting transaction %s: %v\n", dbTx.ID, err)
			continue
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// FindByContract retrieves all transactions for a specific contract
func (r *PostgresTransactionRepository) FindByContract(ctx context.Context, contractID string) ([]*hashperp.Transaction, error) {
	var dbTransactions []DBTransaction
	result := r.db.WithContext(ctx).Where("contract_id = ?", contractID).Find(&dbTransactions)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find transactions by contract: %w", result.Error)
	}

	transactions := make([]*hashperp.Transaction, 0, len(dbTransactions))
	for _, dbTx := range dbTransactions {
		tx, err := convertDBTransactionToTransaction(&dbTx)
		if err != nil {
			// Log the error but continue with other transactions
			fmt.Printf("error converting transaction %s: %v\n", dbTx.ID, err)
			continue
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// FindByType retrieves all transactions of a specific type
func (r *PostgresTransactionRepository) FindByType(ctx context.Context, transactionType hashperp.TransactionType) ([]*hashperp.Transaction, error) {
	var dbTransactions []DBTransaction
	result := r.db.WithContext(ctx).Where("type = ?", string(transactionType)).Find(&dbTransactions)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find transactions by type: %w", result.Error)
	}

	transactions := make([]*hashperp.Transaction, 0, len(dbTransactions))
	for _, dbTx := range dbTransactions {
		tx, err := convertDBTransactionToTransaction(&dbTx)
		if err != nil {
			// Log the error but continue with other transactions
			fmt.Printf("error converting transaction %s: %v\n", dbTx.ID, err)
			continue
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// FindByTimeRange retrieves all transactions within a time range
func (r *PostgresTransactionRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*hashperp.Transaction, error) {
	var dbTransactions []DBTransaction
	result := r.db.WithContext(ctx).
		Where("timestamp BETWEEN ? AND ?", startTime, endTime).
		Find(&dbTransactions)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find transactions by time range: %w", result.Error)
	}

	transactions := make([]*hashperp.Transaction, 0, len(dbTransactions))
	for _, dbTx := range dbTransactions {
		tx, err := convertDBTransactionToTransaction(&dbTx)
		if err != nil {
			// Log the error but continue with other transactions
			fmt.Printf("error converting transaction %s: %v\n", dbTx.ID, err)
			continue
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// Update updates an existing transaction
func (r *PostgresTransactionRepository) Update(ctx context.Context, tx *hashperp.Transaction) error {
	// Convert related entities to JSON
	var relatedEntitiesJSON json.RawMessage
	if len(tx.RelatedEntities) > 0 {
		entitiesBytes, err := json.Marshal(tx.RelatedEntities)
		if err != nil {
			return fmt.Errorf("failed to marshal related entities: %w", err)
		}
		relatedEntitiesJSON = entitiesBytes
	}
	
	dbTransaction := &DBTransaction{
		ID:              tx.ID,
		Type:            string(tx.Type),
		Timestamp:       tx.Timestamp,
		ContractID:      tx.ContractID,
		UserIDs:         pq.StringArray(tx.UserIDs),
		TxHash:          tx.TxHash,
		Amount:          tx.Amount,
		BTCPerPHPerDay:  tx.BTCPerPHPerDay,
		BlockHeight:     tx.BlockHeight,
		Status:          tx.Status,
		RelatedEntities: relatedEntitiesJSON,
	}

	result := r.db.WithContext(ctx).Save(dbTransaction)
	if result.Error != nil {
		return fmt.Errorf("failed to update transaction: %w", result.Error)
	}

	return nil
}
