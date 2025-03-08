package hashperp

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SwapOfferStatus represents the current status of a swap offer
type SwapOfferStatus string

const (
	OFFER_OPEN     SwapOfferStatus = "OPEN"
	OFFER_ACCEPTED SwapOfferStatus = "ACCEPTED"
	OFFER_EXPIRED  SwapOfferStatus = "EXPIRED"
	OFFER_CANCELED SwapOfferStatus = "CANCELED"
)

// SwapOfferRepository defines the data access interface for swap offers
type SwapOfferRepository interface {
	Create(ctx context.Context, offer *SwapOffer) error
	FindByID(ctx context.Context, id string) (*SwapOffer, error)
	FindByUser(ctx context.Context, userID string, isOfferor bool) ([]*SwapOffer, error)
	FindByContract(ctx context.Context, contractID string) ([]*SwapOffer, error)
	FindOpenOffersByVTXO(ctx context.Context, vtxoID string) ([]*SwapOffer, error)
	Update(ctx context.Context, offer *SwapOffer) error
	Delete(ctx context.Context, id string) error
}

// swapOfferService implements the SwapOfferManager interface
type swapOfferService struct {
	swapOfferRepo  SwapOfferRepository
	vtxoRepo       VTXORepository
	contractRepo   ContractRepository
	transactionRepo TransactionRepository
	vtxoManager    VTXOManager
}

// NewSwapOfferService creates a new swap offer service
func NewSwapOfferService(
	swapOfferRepo SwapOfferRepository,
	vtxoRepo VTXORepository,
	contractRepo ContractRepository,
	transactionRepo TransactionRepository,
	vtxoManager VTXOManager,
) SwapOfferManager {
	return &swapOfferService{
		swapOfferRepo:  swapOfferRepo,
		vtxoRepo:       vtxoRepo,
		contractRepo:   contractRepo,
		transactionRepo: transactionRepo,
		vtxoManager:    vtxoManager,
	}
}

// CreateSwapOffer implements SwapOfferManager.CreateSwapOffer
func (s *swapOfferService) CreateSwapOffer(
	ctx context.Context,
	offerorID string,
	vtxoID string,
	offeredRate float64,
	expiryTime time.Time,
) (*SwapOffer, error) {
	// 1. Validate the VTXO exists and belongs to the offeror
	vtxo, err := s.vtxoRepo.FindByID(ctx, vtxoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXO: %w", err)
	}
	if vtxo == nil {
		return nil, ErrVTXONotFound
	}

	// 2. Validate the offeror owns this VTXO
	if vtxo.OwnerID != offerorID {
		return nil, ErrInvalidOwner
	}

	// 3. Validate the VTXO is active
	if !vtxo.IsActive {
		return nil, ErrVTXONotActive
	}

	// 4. Get the associated contract
	contract, err := s.contractRepo.FindByID(ctx, vtxo.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 5. Validate contract status
	if contract.Status != ACTIVE {
		return nil, ErrInvalidContractStatus
	}

	// 6. Validate offered rate
	if offeredRate <= 0 {
		return nil, errors.New("offered rate must be positive")
	}

	// 7. Validate expiry time is in the future
	if expiryTime.Before(time.Now()) {
		return nil, errors.New("expiry time must be in the future")
	}

	// 8. Check if there are already open offers for this VTXO
	existingOffers, err := s.swapOfferRepo.FindOpenOffersByVTXO(ctx, vtxoID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing offers: %w", err)
	}

	// 9. Cancel any existing offers for this VTXO
	for _, existingOffer := range existingOffers {
		existingOffer.Status = string(OFFER_CANCELED)
		if err := s.swapOfferRepo.Update(ctx, existingOffer); err != nil {
			return nil, fmt.Errorf("failed to cancel existing offer: %w", err)
		}
	}

	// 10. Create the new swap offer
	offer := &SwapOffer{
		ID:           generateUniqueID(),
		OfferorID:    offerorID,
		VTXOID:       vtxoID,
		ContractID:   vtxo.ContractID,
		OfferedRate:  offeredRate,
		CreationTime: time.Now().UTC(),
		ExpiryTime:   expiryTime,
		Status:       string(OFFER_OPEN),
	}

	// 11. Save the swap offer
	if err := s.swapOfferRepo.Create(ctx, offer); err != nil {
		return nil, fmt.Errorf("failed to create swap offer: %w", err)
	}

	return offer, nil
}

// AcceptSwapOffer implements SwapOfferManager.AcceptSwapOffer
func (s *swapOfferService) AcceptSwapOffer(
	ctx context.Context,
	offerID string,
	acceptorID string,
) (*Transaction, error) {
	// 1. Get the offer
	offer, err := s.swapOfferRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offer: %w", err)
	}
	if offer == nil {
		return nil, errors.New("swap offer not found")
	}

	// 2. Validate offer status
	if offer.Status != string(OFFER_OPEN) {
		return nil, errors.New("swap offer is not open for acceptance")
	}

	// 3. Validate offer hasn't expired
	if offer.ExpiryTime.Before(time.Now()) {
		offer.Status = string(OFFER_EXPIRED)
		_ = s.swapOfferRepo.Update(ctx, offer)
		return nil, errors.New("swap offer has expired")
	}

	// 4. Get the VTXO
	vtxo, err := s.vtxoRepo.FindByID(ctx, offer.VTXOID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXO: %w", err)
	}
	if vtxo == nil {
		return nil, ErrVTXONotFound
	}

	// 5. Validate the VTXO is still active
	if !vtxo.IsActive {
		offer.Status = string(OFFER_CANCELED)
		_ = s.swapOfferRepo.Update(ctx, offer)
		return nil, ErrVTXONotActive
	}

	// 6. Get the contract
	contract, err := s.contractRepo.FindByID(ctx, vtxo.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 7. Validate contract status
	if contract.Status != ACTIVE {
		offer.Status = string(OFFER_CANCELED)
		_ = s.swapOfferRepo.Update(ctx, offer)
		return nil, ErrInvalidContractStatus
	}

	// 8. Determine which position is being swapped
	var positionType string
	if contract.BuyerVTXO == vtxo.ID {
		positionType = "buyer"
	} else if contract.SellerVTXO == vtxo.ID {
		positionType = "seller"
	} else {
		// This should never happen if our data integrity is maintained
		return nil, errors.New("VTXO is not associated with this contract's buyer or seller")
	}

	// 9. Generate dummy signature data for the new VTXO
	// In a real implementation, the acceptor would provide their signature
	dummySignatureData := []byte("dummy_signature_data")

	// 10. Execute the VTXO swap
	newVTXO, tx, err := s.vtxoManager.SwapVTXO(ctx, vtxo.ID, acceptorID, dummySignatureData)
	if err != nil {
		return nil, fmt.Errorf("failed to execute VTXO swap: %w", err)
	}

	// 11. Update the offer status to ACCEPTED
	offer.Status = string(OFFER_ACCEPTED)
	offer.AcceptorID = acceptorID
	if err := s.swapOfferRepo.Update(ctx, offer); err != nil {
		return nil, fmt.Errorf("failed to update swap offer status: %w", err)
	}

	// 12. Add offer details to the transaction's related entities
	tx.RelatedEntities["swap_offer_id"] = offer.ID
	tx.RelatedEntities["offered_rate"] = fmt.Sprintf("%f", offer.OfferedRate)

	if err := s.transactionRepo.Update(ctx, tx); err != nil {
		// If updating the transaction fails, we'll still proceed with the swap
		// but log the error
		fmt.Printf("failed to update transaction with offer details: %v\n", err)
	}

	return tx, nil
}

// CancelSwapOffer implements SwapOfferManager.CancelSwapOffer
func (s *swapOfferService) CancelSwapOffer(
	ctx context.Context,
	offerID string,
	offerorID string,
) error {
	// 1. Get the offer
	offer, err := s.swapOfferRepo.FindByID(ctx, offerID)
	if err != nil {
		return fmt.Errorf("failed to get swap offer: %w", err)
	}
	if offer == nil {
		return errors.New("swap offer not found")
	}

	// 2. Validate the offeror is the owner of this offer
	if offer.OfferorID != offerorID {
		return errors.New("user is not the offeror of this swap")
	}

	// 3. Validate offer status
	if offer.Status != string(OFFER_OPEN) {
		return errors.New("swap offer is not open for cancellation")
	}

	// 4. Update the offer status to CANCELED
	offer.Status = string(OFFER_CANCELED)
	if err := s.swapOfferRepo.Update(ctx, offer); err != nil {
		return fmt.Errorf("failed to update swap offer status: %w", err)
	}

	return nil
}

// GetSwapOffer implements SwapOfferManager.GetSwapOffer
func (s *swapOfferService) GetSwapOffer(
	ctx context.Context,
	offerID string,
) (*SwapOffer, error) {
	offer, err := s.swapOfferRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offer: %w", err)
	}
	if offer == nil {
		return nil, errors.New("swap offer not found")
	}
	return offer, nil
}

// GetSwapOffersByUser implements SwapOfferManager.GetSwapOffersByUser
func (s *swapOfferService) GetSwapOffersByUser(
	ctx context.Context,
	userID string,
	isOfferor bool,
) ([]*SwapOffer, error) {
	offers, err := s.swapOfferRepo.FindByUser(ctx, userID, isOfferor)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offers by user: %w", err)
	}
	return offers, nil
}

// GetSwapOffersByContract implements SwapOfferManager.GetSwapOffersByContract
func (s *swapOfferService) GetSwapOffersByContract(
	ctx context.Context,
	contractID string,
) ([]*SwapOffer, error) {
	// 1. Validate contract exists
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	// 2. Get offers for the contract
	offers, err := s.swapOfferRepo.FindByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offers by contract: %w", err)
	}

	return offers, nil
}
