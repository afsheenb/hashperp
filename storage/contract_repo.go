package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hashperp/hashperp"
	"gorm.io/gorm"
)

// PostgresContractRepository implements the ContractRepository interface using PostgreSQL
type PostgresContractRepository struct {
	db *gorm.DB
}

// NewPostgresContractRepository creates a new PostgreSQL-based contract repository
func NewPostgresContractRepository(db *gorm.DB) hashperp.ContractRepository {
	return &PostgresContractRepository{
		db: db,
	}
}

// Create creates a new contract
func (r *PostgresContractRepository) Create(ctx context.Context, contract *hashperp.Contract) error {
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

	result := r.db.WithContext(ctx).Create(dbContract)
	if result.Error != nil {
		return fmt.Errorf("failed to create contract: %w", result.Error)
	}

	return nil
}

// FindByID retrieves a contract by ID
func (r *PostgresContractRepository) FindByID(ctx context.Context, id string) (*hashperp.Contract, error) {
	var dbContract DBContract
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&dbContract)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find contract: %w", result.Error)
	}

	return convertDBContractToContract(&dbContract), nil
}

// FindByUser retrieves all contracts for a specific user
func (r *PostgresContractRepository) FindByUser(ctx context.Context, userID string, status []hashperp.ContractStatus) ([]*hashperp.Contract, error) {
	var dbContracts []DBContract
	
	// Convert string statuses to string array
	var statusStrings []string
	for _, s := range status {
		statusStrings = append(statusStrings, string(s))
	}
	
	query := r.db.WithContext(ctx).
		Where("buyer_id = ? OR seller_id = ?", userID, userID)
	
	if len(statusStrings) > 0 {
		query = query.Where("status IN ?", statusStrings)
	}
	
	result := query.Find(&dbContracts)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find contracts for user: %w", result.Error)
	}

	contracts := make([]*hashperp.Contract, len(dbContracts))
	for i, dbContract := range dbContracts {
		contracts[i] = convertDBContractToContract(&dbContract)
	}

	return contracts, nil
}

// FindActiveContracts retrieves all active contracts
func (r *PostgresContractRepository) FindActiveContracts(ctx context.Context) ([]*hashperp.Contract, error) {
	var dbContracts []DBContract
	result := r.db.WithContext(ctx).Where("status = ?", string(hashperp.ACTIVE)).Find(&dbContracts)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find active contracts: %w", result.Error)
	}

	contracts := make([]*hashperp.Contract, len(dbContracts))
	for i, dbContract := range dbContracts {
		contracts[i] = convertDBContractToContract(&dbContract)
	}

	return contracts, nil
}

// FindByExpiryRange retrieves contracts expiring within a certain block height range
func (r *PostgresContractRepository) FindByExpiryRange(ctx context.Context, fromHeight, toHeight uint64) ([]*hashperp.Contract, error) {
	var dbContracts []DBContract
	result := r.db.WithContext(ctx).
		Where("status = ? AND expiry_block_height BETWEEN ? AND ?", 
		      string(hashperp.ACTIVE), fromHeight, toHeight).
		Find(&dbContracts)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find contracts by expiry range: %w", result.Error)
	}

	contracts := make([]*hashperp.Contract, len(dbContracts))
	for i, dbContract := range dbContracts {
		contracts[i] = convertDBContractToContract(&dbContract)
	}

	return contracts, nil
}

// Update updates an existing contract
func (r *PostgresContractRepository) Update(ctx context.Context, contract *hashperp.Contract) error {
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

	result := r.db.WithContext(ctx).Save(dbContract)
	if result.Error != nil {
		return fmt.Errorf("failed to update contract: %w", result.Error)
	}

	return nil
}

// Delete deletes a contract by ID
func (r *PostgresContractRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&DBContract{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete contract: %w", result.Error)
	}
	return nil
}

// Helper function to convert DB model to domain model
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

	return contract
}

// PostgresVTXORepository implements the VTXORepository interface using PostgreSQL
type PostgresVTXORepository struct {
	db *gorm.DB
}

// NewPostgresVTXORepository creates a new PostgreSQL-based VTXO repository
func NewPostgresVTXORepository(db *gorm.DB) hashperp.VTXORepository {
	return &PostgresVTXORepository{
		db: db,
	}
}

// Create creates a new VTXO
func (r *PostgresVTXORepository) Create(ctx context.Context, vtxo *hashperp.VTXO) error {
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

	result := r.db.WithContext(ctx).Create(dbVTXO)
	if result.Error != nil {
		return fmt.Errorf("failed to create VTXO: %w", result.Error)
	}

	return nil
}

// FindByID retrieves a VTXO by ID
func (r *PostgresVTXORepository) FindByID(ctx context.Context, id string) (*hashperp.VTXO, error) {
	var dbVTXO DBVTXO
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&dbVTXO)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find VTXO: %w", result.Error)
	}

	return convertDBVTXOToVTXO(&dbVTXO), nil
}

// FindByContract retrieves all VTXOs for a specific contract
func (r *PostgresVTXORepository) FindByContract(ctx context.Context, contractID string) ([]*hashperp.VTXO, error) {
	var dbVTXOs []DBVTXO
	result := r.db.WithContext(ctx).Where("contract_id = ?", contractID).Find(&dbVTXOs)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find VTXOs for contract: %w", result.Error)
	}

	vtxos := make([]*hashperp.VTXO, len(dbVTXOs))
	for i, dbVTXO := range dbVTXOs {
		vtxos[i] = convertDBVTXOToVTXO(&dbVTXO)
	}

	return vtxos, nil
}

// FindByUser retrieves all VTXOs for a specific user
func (r *PostgresVTXORepository) FindByUser(ctx context.Context, userID string, onlyActive bool) ([]*hashperp.VTXO, error) {
	var dbVTXOs []DBVTXO
	query := r.db.WithContext(ctx).Where("owner_id = ?", userID)
	
	if onlyActive {
		query = query.Where("is_active = true")
	}
	
	result := query.Find(&dbVTXOs)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find VTXOs for user: %w", result.Error)
	}

	vtxos := make([]*hashperp.VTXO, len(dbVTXOs))
	for i, dbVTXO := range dbVTXOs {
		vtxos[i] = convertDBVTXOToVTXO(&dbVTXO)
	}

	return vtxos, nil
}

// FindActiveVTXOs retrieves all active VTXOs
func (r *PostgresVTXORepository) FindActiveVTXOs(ctx context.Context) ([]*hashperp.VTXO, error) {
	var dbVTXOs []DBVTXO
	result := r.db.WithContext(ctx).Where("is_active = true").Find(&dbVTXOs)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find active VTXOs: %w", result.Error)
	}

	vtxos := make([]*hashperp.VTXO, len(dbVTXOs))
	for i, dbVTXO := range dbVTXOs {
		vtxos[i] = convertDBVTXOToVTXO(&dbVTXO)
	}

	return vtxos, nil
}

// Update updates an existing VTXO
func (r *PostgresVTXORepository) Update(ctx context.Context, vtxo *hashperp.VTXO) error {
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

	result := r.db.WithContext(ctx).Save(dbVTXO)
	if result.Error != nil {
		return fmt.Errorf("failed to update VTXO: %w", result.Error)
	}

	return nil
}

// Delete deletes a VTXO by ID
func (r *PostgresVTXORepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&DBVTXO{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete VTXO: %w", result.Error)
	}
	return nil
}

// Helper function to convert DB model to domain model
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

	return vtxo
}

// PostgresOrderRepository implements the OrderRepository interface using PostgreSQL
type PostgresOrderRepository struct {
	db *gorm.DB
}

// NewPostgresOrderRepository creates a new PostgreSQL-based order repository
func NewPostgresOrderRepository(db *gorm.DB) hashperp.OrderRepository {
	return &PostgresOrderRepository{
		db: db,
	}
}

// Create creates a new order
func (r *PostgresOrderRepository) Create(ctx context.Context, order *hashperp.Order) error {
	dbOrder := &DBOrder{
		ID:                order.ID,
		UserID:            order.UserID,
		OrderType:         string(order.OrderType),
		ContractType:      string(order.ContractType),
		StrikeRate:        order.StrikeRate,
		ExpiryBlockHeight: order.ExpiryBlockHeight,
		ExpiryDate:        order.ExpiryDate,
		Size:              order.Size,
		Status:            string(order.Status),
		CreationTime:      order.CreationTime,
	}

	if order.MatchedOrderID != "" {
		dbOrder.MatchedOrderID = sql.NullString{
			String: order.MatchedOrderID,
			Valid:  true,
		}
	}

	if order.ResultingContractID != "" {
		dbOrder.ResultingContractID = sql.NullString{
			String: order.ResultingContractID,
			Valid:  true,
		}
	}

	result := r.db.WithContext(ctx).Create(dbOrder)
	if result.Error != nil {
		return fmt.Errorf("failed to create order: %w", result.Error)
	}

	return nil
}

// FindAll returns all contracts in the system
func (r *PostgresContractRepository) FindAll(ctx context.Context) ([]*hashperp.Contract, error) {
	var dbContracts []DBContract
	result := r.db.WithContext(ctx).Find(&dbContracts)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find contracts: %w", result.Error)
	}
	
	contracts := make([]*hashperp.Contract, len(dbContracts))
	for i, dbContract := range dbContracts {
		contracts[i] = convertDBContractToContract(&dbContract)
	}
	
	return contracts, nil
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

// FindActiveByContract finds active VTXOs for a contract
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

// PostgresSwapOfferRepository implements the SwapOfferRepository interface
type PostgresSwapOfferRepository struct {
	db *gorm.DB
}

// NewPostgresSwapOfferRepository creates a new PostgreSQL-based swap offer repository
func NewPostgresSwapOfferRepository(db *gorm.DB) hashperp.SwapOfferRepository {
	return &PostgresSwapOfferRepository{
		db: db,
	}
}

// Create creates a new swap offer
func (r *PostgresSwapOfferRepository) Create(ctx context.Context, offer *hashperp.SwapOffer) error {
	dbSwapOffer := &DBSwapOffer{
		ID:           offer.ID,
		OfferorID:    offer.OfferorID,
		VTXOID:       offer.VTXOID,
		ContractID:   offer.ContractID,
		OfferedRate:  offer.OfferedRate,
		CreationTime: offer.CreationTime,
		ExpiryTime:   offer.ExpiryTime,
		Status:       offer.Status,
	}
	
	if offer.AcceptorID != "" {
		dbSwapOffer.AcceptorID = sql.NullString{
			String: offer.AcceptorID,
			Valid:  true,
		}
	}
	
	// Store additional fields if set
	if offer.TargetUserID != "" {
		dbSwapOffer.TargetUserID = sql.NullString{
			String: offer.TargetUserID,
			Valid:  true,
		}
	}
	
	if offer.SwapType != "" {
		dbSwapOffer.SwapType = sql.NullString{
			String: offer.SwapType,
			Valid:  true,
		}
	}
	
	if len(offer.RelatedEntities) > 0 {
		relatedEntitiesJSON, err := json.Marshal(offer.RelatedEntities)
		if err != nil {
			return fmt.Errorf("failed to marshal related entities: %w", err)
		}
		
		dbSwapOffer.RelatedEntities = relatedEntitiesJSON
	}
	
	result := r.db.WithContext(ctx).Create(dbSwapOffer)
	if result.Error != nil {
		return fmt.Errorf("failed to create swap offer: %w", result.Error)
	}
	
	return nil
}

// FindByID retrieves a swap offer by ID
func (r *PostgresSwapOfferRepository) FindByID(ctx context.Context, id string) (*hashperp.SwapOffer, error) {
	var dbSwapOffer DBSwapOffer
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&dbSwapOffer)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find swap offer: %w", result.Error)
	}
	
	return convertDBSwapOfferToSwapOffer(&dbSwapOffer)
}

// FindByUser retrieves all swap offers for a specific user
func (r *PostgresSwapOfferRepository) FindByUser(ctx context.Context, userID string, isOfferor bool) ([]*hashperp.SwapOffer, error) {
	var dbSwapOffers []DBSwapOffer
	var query *gorm.DB
	
	if isOfferor {
		query = r.db.WithContext(ctx).Where("offeror_id = ?", userID)
	} else {
		query = r.db.WithContext(ctx).Where("acceptor_id = ? OR target_user_id = ?", userID, userID)
	}
	
	result := query.Find(&dbSwapOffers)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find swap offers by user: %w", result.Error)
	}
	
	swapOffers := make([]*hashperp.SwapOffer, 0, len(dbSwapOffers))
	for _, dbSwapOffer := range dbSwapOffers {
		swapOffer, err := convertDBSwapOfferToSwapOffer(&dbSwapOffer)
		if err != nil {
			continue // Skip offers we can't convert
		}
		swapOffers = append(swapOffers, swapOffer)
	}
	
	return swapOffers, nil
}

// FindByContract retrieves all swap offers for a specific contract
func (r *PostgresSwapOfferRepository) FindByContract(ctx context.Context, contractID string) ([]*hashperp.SwapOffer, error) {
	var dbSwapOffers []DBSwapOffer
	result := r.db.WithContext(ctx).Where("contract_id = ?", contractID).Find(&dbSwapOffers)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find swap offers by contract: %w", result.Error)
	}
	
	swapOffers := make([]*hashperp.SwapOffer, 0, len(dbSwapOffers))
	for _, dbSwapOffer := range dbSwapOffers {
		swapOffer, err := convertDBSwapOfferToSwapOffer(&dbSwapOffer)
		if err != nil {
			continue // Skip offers we can't convert
		}
		swapOffers = append(swapOffers, swapOffer)
	}
	
	return swapOffers, nil
}

// FindOpenOffersByVTXO retrieves all open swap offers for a specific VTXO
func (r *PostgresSwapOfferRepository) FindOpenOffersByVTXO(ctx context.Context, vtxoID string) ([]*hashperp.SwapOffer, error) {
	var dbSwapOffers []DBSwapOffer
	result := r.db.WithContext(ctx).
		Where("vtxo_id = ? AND status = ?", vtxoID, string(hashperp.OFFER_OPEN)).
		Find(&dbSwapOffers)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find open swap offers by VTXO: %w", result.Error)
	}
	
	swapOffers := make([]*hashperp.SwapOffer, 0, len(dbSwapOffers))
	for _, dbSwapOffer := range dbSwapOffers {
		swapOffer, err := convertDBSwapOfferToSwapOffer(&dbSwapOffer)
		if err != nil {
			continue // Skip offers we can't convert
		}
		swapOffers = append(swapOffers, swapOffer)
	}
	
	return swapOffers, nil
}

// Update updates an existing swap offer
func (r *PostgresSwapOfferRepository) Update(ctx context.Context, offer *hashperp.SwapOffer) error {
	dbSwapOffer := &DBSwapOffer{
		ID:           offer.ID,
		OfferorID:    offer.OfferorID,
		VTXOID:       offer.VTXOID,
		ContractID:   offer.ContractID,
		OfferedRate:  offer.OfferedRate,
		CreationTime: offer.CreationTime,
		ExpiryTime:   offer.ExpiryTime,
		Status:       offer.Status,
	}
	
	if offer.AcceptorID != "" {
		dbSwapOffer.AcceptorID = sql.NullString{
			String: offer.AcceptorID,
			Valid:  true,
		}
	}
	
	// Store additional fields if set
	if offer.TargetUserID != "" {
		dbSwapOffer.TargetUserID = sql.NullString{
			String: offer.TargetUserID,
			Valid:  true,
		}
	}
	
	if offer.SwapType != "" {
		dbSwapOffer.SwapType = sql.NullString{
			String: offer.SwapType,
			Valid:  true,
		}
	}
	
	if len(offer.RelatedEntities) > 0 {
		relatedEntitiesJSON, err := json.Marshal(offer.RelatedEntities)
		if err != nil {
			return fmt.Errorf("failed to marshal related entities: %w", err)
		}
		
		dbSwapOffer.RelatedEntities = relatedEntitiesJSON
	}
	
	result := r.db.WithContext(ctx).Save(dbSwapOffer)
	if result.Error != nil {
		return fmt.Errorf("failed to update swap offer: %w", result.Error)
	}
	
	return nil
}

// Delete deletes a swap offer by ID
func (r *PostgresSwapOfferRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&DBSwapOffer{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete swap offer: %w", result.Error)
	}
	return nil
}

// convertDBSwapOfferToSwapOffer converts a database model to domain model
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
