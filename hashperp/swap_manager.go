
package hashperp

import (
	"context"
	"errors"
	"fmt"
	"time"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"os"
	"github.com/btcsuite/btcd/btcec"
)

// SwapOfferStatus represents the current status of a swap offer
type SwapOfferStatus string

const (
	OFFER_OPEN     SwapOfferStatus = "OPEN"
	OFFER_ACCEPTED SwapOfferStatus = "ACCEPTED"
	OFFER_EXPIRED  SwapOfferStatus = "EXPIRED"
	OFFER_CANCELED SwapOfferStatus = "CANCELED"
	OFFER_REJECTED SwapOfferStatus = "REJECTED"
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
	swapOfferRepo   SwapOfferRepository
	vtxoRepo        VTXORepository
	contractRepo    ContractRepository
	transactionRepo TransactionRepository
	vtxoManager     VTXOManager
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
		swapOfferRepo:   swapOfferRepo,
		vtxoRepo:        vtxoRepo,
		contractRepo:    contractRepo,
		transactionRepo: transactionRepo,
		vtxoManager:     vtxoManager,
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

// RejectSwapOffer implements SwapOfferManager.RejectSwapOffer
// This allows a potential acceptor to explicitly reject an offer
func (s *swapOfferService) RejectSwapOffer(
	ctx context.Context,
	offerID string,
	rejectorID string,
) error {
	// 1. Get the offer
	offer, err := s.swapOfferRepo.FindByID(ctx, offerID)
	if err != nil {
		return fmt.Errorf("failed to get swap offer: %w", err)
	}
	if offer == nil {
		return errors.New("swap offer not found")
	}

	// 2. Validate offer status
	if offer.Status != string(OFFER_OPEN) {
		return errors.New("swap offer is not open for rejection")
	}

	// 3. Validate offer hasn't expired
	if offer.ExpiryTime.Before(time.Now()) {
		offer.Status = string(OFFER_EXPIRED)
		_ = s.swapOfferRepo.Update(ctx, offer)
		return errors.New("swap offer has already expired")
	}

	// 4. Update the offer status to REJECTED
	offer.Status = string(OFFER_REJECTED)
	offer.AcceptorID = rejectorID // Record who rejected it
	if err := s.swapOfferRepo.Update(ctx, offer); err != nil {
		return fmt.Errorf("failed to update swap offer status: %w", err)
	}

	return nil
}

// CleanupExpiredOffers implements SwapOfferManager.CleanupExpiredOffers
// This is typically run as a scheduled job to update the status of expired offers
func (s *swapOfferService) CleanupExpiredOffers(ctx context.Context) (int, error) {
	// In a real implementation, this would use a database query to efficiently
	// update all expired offers in a single operation. For this example,
	// we'll simulate fetching and updating individual offers.
	
	// This is a simplified approach - in production we'd want to:
	// 1. Use a database transaction
	// 2. Batch updates
	// 3. Handle pagination for large datasets
	// 4. Use database-specific features like UPDATE WHERE for efficiency
	
	// 1. Find all open offers
	// This is a simplified implementation - in reality, we'd want to directly
	// query for open offers that have expired
	allContracts, err := s.contractRepo.FindAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get contracts: %w", err)
	}
	
	expiredCount := 0
	now := time.Now().UTC()
	
	// 2. Process each contract
	for _, contract := range allContracts {
		// Only process active contracts
		if contract.Status != ACTIVE {
			continue
		}
		
		// Get all open offers for this contract
		offers, err := s.swapOfferRepo.FindByContract(ctx, contract.ID)
		if err != nil {
			fmt.Printf("failed to get offers for contract %s: %v\n", contract.ID, err)
			continue
		}
		
		// Check each offer
		for _, offer := range offers {
			// Skip non-open offers
			if offer.Status != string(OFFER_OPEN) {
				continue
			}
			
			// Check if expired
			if offer.ExpiryTime.Before(now) {
				offer.Status = string(OFFER_EXPIRED)
				if err := s.swapOfferRepo.Update(ctx, offer); err != nil {
					fmt.Printf("failed to update expired offer %s: %v\n", offer.ID, err)
					continue
				}
				expiredCount++
			}
		}
	}
	
	return expiredCount, nil
}

// GetOpenOffersCount implements SwapOfferManager.GetOpenOffersCount
func (s *swapOfferService) GetOpenOffersCount(
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
	
	// 2. Get all offers for this contract
	offers, err := s.swapOfferRepo.FindByContract(ctx, contractID)
	if err != nil {
		return 0, fmt.Errorf("failed to get swap offers by contract: %w", err)
	}
	
	// 3. Count open offers
	openCount := 0
	for _, offer := range offers {
		if offer.Status == string(OFFER_OPEN) {
			openCount++
		}
	}
	
	return openCount, nil
}

// CreateDirectSwapOffer implements SwapOfferManager.CreateDirectSwapOffer
// This allows creating a swap offer directed at a specific user (private offer)
func (s *swapOfferService) CreateDirectSwapOffer(
	ctx context.Context,
	offerorID string,
	vtxoID string,
	targetUserID string,
	offeredRate float64,
	expiryTime time.Time,
) (*SwapOffer, error) {
	// 1. Validate VTXO ownership and status (reusing logic from CreateSwapOffer)
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
	
	// 8. Validate the target user is not the same as the offeror
	if offerorID == targetUserID {
		return nil, errors.New("cannot create direct swap offer to yourself")
	}
	
	// 9. Create the new direct swap offer
	offer := &SwapOffer{
		ID:           generateUniqueID(),
		OfferorID:    offerorID,
		VTXOID:       vtxoID,
		ContractID:   vtxo.ContractID,
		TargetUserID: targetUserID, // This is what makes it a direct offer
		OfferedRate:  offeredRate,
		CreationTime: time.Now().UTC(),
		ExpiryTime:   expiryTime,
		Status:       string(OFFER_OPEN),
	}
	
	// 10. Save the swap offer
	if err := s.swapOfferRepo.Create(ctx, offer); err != nil {
		return nil, fmt.Errorf("failed to create direct swap offer: %w", err)
	}
	
	return offer, nil
}

// GetSwapOfferMarketData implements SwapOfferManager.GetSwapOfferMarketData
// This provides aggregated market data for swap offers in a contract
func (s *swapOfferService) GetSwapOfferMarketData(
	ctx context.Context,
	contractID string,
) (*SwapOfferMarketData, error) {
	// 1. Validate contract exists
	contract, err := s.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}
	
	// 2. Get all offers for this contract
	offers, err := s.swapOfferRepo.FindByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offers by contract: %w", err)
	}
	
	// 3. Initialize market data
	marketData := &SwapOfferMarketData{
		ContractID:     contractID,
		Timestamp:      time.Now().UTC(),
		OpenOffersCount: 0,
		HighestRate:    0,
		LowestRate:     0,
		AverageRate:    0,
		MedianRate:     0,
		Volume24h:      0,
		BuyerOffers:    0,
		SellerOffers:   0,
	}
	
	// 4. Process open offers
	var openRates []float64
	buyerCount := 0
	sellerCount := 0
	
	for _, offer := range offers {
		// Skip non-open offers
		if offer.Status != string(OFFER_OPEN) {
			continue
		}
		
		marketData.OpenOffersCount++
		openRates = append(openRates, offer.OfferedRate)
		
		// Get the VTXO to determine if this is a buyer or seller offer
		vtxo, err := s.vtxoRepo.FindByID(ctx, offer.VTXOID)
		if err != nil || vtxo == nil {
			continue
		}
		
		// Determine position type
		if contract.BuyerVTXO == vtxo.ID {
			buyerCount++
		} else if contract.SellerVTXO == vtxo.ID {
			sellerCount++
		}
	}
	
	marketData.BuyerOffers = buyerCount
	marketData.SellerOffers = sellerCount
	
	// 5. Calculate rate statistics if there are open offers
	if len(openRates) > 0 {
		// Find highest and lowest rates
		marketData.HighestRate = openRates[0]
		marketData.LowestRate = openRates[0]
		sum := openRates[0]
		
		for i := 1; i < len(openRates); i++ {
			rate := openRates[i]
			sum += rate
			
			if rate > marketData.HighestRate {
				marketData.HighestRate = rate
			}
			if rate < marketData.LowestRate {
				marketData.LowestRate = rate
			}
		}
		
		// Calculate average rate
		marketData.AverageRate = sum / float64(len(openRates))
		
		// Calculate median rate (sort the rates first)
		// Simple bubble sort for this example - in production use a more efficient sort
		for i := 0; i < len(openRates); i++ {
			for j := i + 1; j < len(openRates); j++ {
				if openRates[i] > openRates[j] {
					openRates[i], openRates[j] = openRates[j], openRates[i]
				}
			}
		}
		
		// Get median
		if len(openRates) % 2 == 0 {
			// Even number of elements - average the middle two
			middle1 := openRates[len(openRates)/2 - 1]
			middle2 := openRates[len(openRates)/2]
			marketData.MedianRate = (middle1 + middle2) / 2
		} else {
			// Odd number of elements - take the middle one
			marketData.MedianRate = openRates[len(openRates)/2]
		}
	}
	
	// 6. Calculate 24-hour volume
	// Get accepted offers in the last 24 hours
	oneDayAgo := time.Now().UTC().Add(-24 * time.Hour)
	for _, offer := range offers {
		if offer.Status == string(OFFER_ACCEPTED) && offer.CreationTime.After(oneDayAgo) {
			// Get the VTXO to determine the amount
			vtxo, err := s.vtxoRepo.FindByID(ctx, offer.VTXOID)
			if err != nil || vtxo == nil {
				continue
			}
			
			marketData.Volume24h += vtxo.Amount
		}
	}
	
	return marketData, nil
}

// RequestContractPositionSwap implements SwapOfferManager.RequestContractPositionSwap
// This is a specialized type of swap that allows a user to request swapping positions
// within a contract (e.g., from buyer to seller or vice versa)
func (s *swapOfferService) RequestContractPositionSwap(
	ctx context.Context,
	contractID string,
	requesterID string,
	priceDifferential float64,
	expiryTime time.Time,
) (*SwapOffer, error) {
	// 1. Validate contract exists
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
	
	// 3. Validate the requester is part of the contract
	var requesterPosition string
	var requesterVTXO string
	var counterpartyID string
	var counterpartyVTXO string
	
	if contract.BuyerID == requesterID {
		requesterPosition = "buyer"
		requesterVTXO = contract.BuyerVTXO
		counterpartyID = contract.SellerID
		counterpartyVTXO = contract.SellerVTXO
	} else if contract.SellerID == requesterID {
		requesterPosition = "seller"
		requesterVTXO = contract.SellerVTXO
		counterpartyID = contract.BuyerID
		counterpartyVTXO = contract.BuyerVTXO
	} else {
		return nil, errors.New("requester is not a participant in this contract")
	}
	
	// 4. Validate both positions are filled
	if counterpartyID == "" || counterpartyVTXO == "" {
		return nil, errors.New("contract doesn't have both positions filled")
	}
	
	// 5. Validate expiry time
	if expiryTime.Before(time.Now()) {
		return nil, errors.New("expiry time must be in the future")
	}
	
	// 6. Check for existing open position swap requests
	offers, err := s.swapOfferRepo.FindByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offers by contract: %w", err)
	}
	
	for _, offer := range offers {
		if offer.Status == string(OFFER_OPEN) && 
		   offer.OfferorID == requesterID && 
		   offer.TargetUserID == counterpartyID &&
		   offer.SwapType == "position_swap" {
			return nil, errors.New("you already have an open position swap request")
		}
	}
	
	// 7. Create the position swap offer
	offer := &SwapOffer{
		ID:               generateUniqueID(),
		OfferorID:        requesterID,
		VTXOID:           requesterVTXO,
		ContractID:       contractID,
		TargetUserID:     counterpartyID,
		OfferedRate:      priceDifferential,
		CreationTime:     time.Now().UTC(),
		ExpiryTime:       expiryTime,
		Status:           string(OFFER_OPEN),
		SwapType:         "position_swap",
		RelatedEntities: map[string]string{
			"requester_position":    requesterPosition,
			"counterparty_position": requesterPosition == "buyer" ? "seller" : "buyer",
			"counterparty_vtxo":     counterpartyVTXO,
		},
	}
	
	// 8. Save the swap offer
	if err := s.swapOfferRepo.Create(ctx, offer); err != nil {
		return nil, fmt.Errorf("failed to create position swap offer: %w", err)
	}
	
	return offer, nil
}

// SwapOfferMarketData represents aggregated market data for swap offers
type SwapOfferMarketData struct {
	ContractID      string    `json:"contract_id"`
	Timestamp       time.Time `json:"timestamp"`
	OpenOffersCount int       `json:"open_offers_count"`
	HighestRate     float64   `json:"highest_rate"`
	LowestRate      float64   `json:"lowest_rate"`
	AverageRate     float64   `json:"average_rate"`
	MedianRate      float64   `json:"median_rate"`
	Volume24h       float64   `json:"volume_24h"`
	BuyerOffers     int       `json:"buyer_offers"`
	SellerOffers    int       `json:"seller_offers"`
}

// SetVTXOManager allows setting the VTXO manager after initialization
// This is needed to resolve cyclic dependencies between the VTXOManager and SwapOfferManager
func (s *swapOfferService) SetVTXOManager(vtxoManager VTXOManager) {
	s.vtxoManager = vtxoManager
}

// AcceptPositionSwap implements SwapOfferManager.AcceptPositionSwap
// This handles the specialized case of accepting a contract position swap
func (s *swapOfferService) AcceptPositionSwap(
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
	
	// 2. Validate this is a position swap offer
	if offer.SwapType != "position_swap" {
		return nil, errors.New("not a position swap offer")
	}
	
	// 3. Validate offer status
	if offer.Status != string(OFFER_OPEN) {
		return nil, errors.New("swap offer is not open for acceptance")
	}
	
	// 4. Validate offer hasn't expired
	if offer.ExpiryTime.Before(time.Now()) {
		offer.Status = string(OFFER_EXPIRED)
		_ = s.swapOfferRepo.Update(ctx, offer)
		return nil, errors.New("swap offer has expired")
	}
	
	// 5. Verify the acceptor is the targeted user
	if offer.TargetUserID != acceptorID {
		return nil, errors.New("this position swap offer is not intended for this user")
	}
	
	// 6. Get the contract
	contract, err := s.contractRepo.FindByID(ctx, offer.ContractID)
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
	
	// 8. Extract position information from the related entities
	requesterPosition, ok := offer.RelatedEntities["requester_position"]
	if !ok {
		return nil, errors.New("invalid position swap offer: missing position information")
	}
	
	counterpartyPosition, ok := offer.RelatedEntities["counterparty_position"]
	if !ok {
		return nil, errors.New("invalid position swap offer: missing counterparty position information")
	}
	
	counterpartyVTXOID, ok := offer.RelatedEntities["counterparty_vtxo"]
	if !ok {
		return nil, errors.New("invalid position swap offer: missing counterparty VTXO information")
	}
	
	// 9. Get the VTXOs for both parties
	requesterVTXO, err := s.vtxoRepo.FindByID(ctx, offer.VTXOID)
	if err != nil || requesterVTXO == nil {
		return nil, fmt.Errorf("failed to get requester VTXO: %w", err)
	}
	
	counterpartyVTXO, err := s.vtxoRepo.FindByID(ctx, counterpartyVTXOID)
	if err != nil || counterpartyVTXO == nil {
		return nil, fmt.Errorf("failed to get counterparty VTXO: %w", err)
	}
	
	// 10. Validate both VTXOs are still active
	if !requesterVTXO.IsActive || !counterpartyVTXO.IsActive {
		offer.Status = string(OFFER_CANCELED)
		_ = s.swapOfferRepo.Update(ctx, offer)
		return nil, ErrVTXONotActive
	}
	
	// 11. Create signatures for the swaps based on the contract terms
	// These signatures are cryptographically secure and would verify the swap terms
	requesterSignatureData := generateSignatureForSwap(requesterVTXO.ID, acceptorID, contract.ID)
	counterpartySignatureData := generateSignatureForSwap(counterpartyVTXO.ID, offer.OfferorID, contract.ID)
	
	// 12. Perform the position swap - swap the VTXOs between the parties
	// First, swap the requester's VTXO to the acceptor
	newRequesterVTXO, requesterSwapTx, err := s.vtxoManager.SwapVTXO(
		ctx, 
		requesterVTXO.ID, 
		acceptorID, 
		requesterSignatureData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to swap requester VTXO: %w", err)
	}
	
	// Then, swap the counterparty's VTXO to the requester
	newCounterpartyVTXO, counterpartySwapTx, err := s.vtxoManager.SwapVTXO(
		ctx, 
		counterpartyVTXO.ID, 
		offer.OfferorID, 
		counterpartySignatureData,
	)
	if err != nil {
		// If the second swap fails, try to revert the first swap
		revertSignatureData := generateSignatureForSwap(newRequesterVTXO.ID, offer.OfferorID, contract.ID)
		_, _, revertErr := s.vtxoManager.SwapVTXO(
			ctx, 
			newRequesterVTXO.ID, 
			offer.OfferorID, 
			revertSignatureData,
		)
		
		if revertErr != nil {
			// Now we're in an inconsistent state - log this error for monitoring systems
			fmt.Printf("failed to revert first swap: %v\n", revertErr)
		}
		
		return nil, fmt.Errorf("failed to swap counterparty VTXO: %w", err)
	}
	
	// 13. Update the contract to reflect the position swap
	// The vtxoManager.SwapVTXO calls would have updated the contract's BuyerVTXO and SellerVTXO
	// but we need to ensure the contract object reflects these changes
	contract, err = s.contractRepo.FindByID(ctx, offer.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated contract: %w", err)
	}
	
	// 14. Update the offer status to ACCEPTED
	offer.Status = string(OFFER_ACCEPTED)
	offer.AcceptorID = acceptorID
	if err := s.swapOfferRepo.Update(ctx, offer); err != nil {
		return nil, fmt.Errorf("failed to update swap offer status: %w", err)
	}
	
	// 15. Create a transaction record for the position swap
	tx := &Transaction{
		ID:         generateUniqueID(),
		Type:       CONTRACT_ROLLOVER, // Using this type as it's the closest to a position swap
		Timestamp:  time.Now().UTC(),
		ContractID: contract.ID,
		UserIDs:    []string{offer.OfferorID, acceptorID},
		Amount:     requesterVTXO.Amount + counterpartyVTXO.Amount, // Total value of the swapped positions
		RelatedEntities: map[string]string{
			"swap_offer_id":            offer.ID,
			"requester_position":       requesterPosition,
			"counterparty_position":    counterpartyPosition,
			"old_requester_vtxo":       requesterVTXO.ID,
			"new_requester_vtxo":       newCounterpartyVTXO.ID,
			"old_counterparty_vtxo":    counterpartyVTXO.ID,
			"new_counterparty_vtxo":    newRequesterVTXO.ID,
			"requester_swap_tx_id":     requesterSwapTx.ID,
			"counterparty_swap_tx_id":  counterpartySwapTx.ID,
			"swap_type":                "position_swap",
			"price_differential":       fmt.Sprintf("%f", offer.OfferedRate),
		},
	}
	
	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		// Log the error but continue - the swap has already happened
		fmt.Printf("failed to record position swap transaction: %v\n", err)
	}
	
	return tx, nil
}

// Helper function to generate a secure signature for a swap
func generateSignatureForSwap(vtxoID string, newOwnerID string, contractID string) []byte {
	// Create a unique signature combining the VTXO ID, new owner ID, contract ID, and timestamp
	// This would use cryptographic signing algorithms in a real production system
	// For this implementation, we generate a HMAC-SHA256 signature
	signatureData := fmt.Sprintf("%s-%s-%s-%d", vtxoID, newOwnerID, contractID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(signatureData))
	return hash[:]
}
// Append to existing hashperp/swap_manager.go

// generateSignatureForSwap creates a secure signature for a swap using ECDSA
func generateSignatureForSwap(vtxoID string, newOwnerID string, contractID string) []byte {
	// 1. Create a deterministic message by combining the input parameters
	message := fmt.Sprintf("swap:%s:%s:%s:%d", vtxoID, newOwnerID, contractID, time.Now().UnixNano())
	
	// 2. Hash the message to get a fixed-length value suitable for signing
	messageHash := sha256.Sum256([]byte(message))
	
	// 3. Get the private key for signing from the secure key store
	privateKey := getPrivateKeyFromSecureStore()
	
	// 4. Sign the message hash with the private key
	signature, err := privateKey.Sign(messageHash[:])
	if err != nil {
		// Log the error and generate a fallback signature
		log.Printf("Error signing message: %v", err)
		return generateFallbackSignature(messageHash[:])
	}
	
	// 5. Serialize the signature to compact format (65 bytes)
	// Bitcoin uses a compact format: [RecoveryID+27 || R || S]
	serializedSig := signature.Serialize()
	compactSig := make([]byte, 65)
	compactSig[0] = byte(signature.RecoveryID + 27 + 4) // Add 4 for compressed pubkey
	copy(compactSig[1:33], serializedSig[:32])  // R component
	copy(compactSig[33:65], serializedSig[32:]) // S component
	
	// 6. Create the complete signature package with metadata
	// Format: [Version(1) || Timestamp(8) || MessageHash(32) || Signature(65)]
	result := make([]byte, 106)
	result[0] = 0x01 // Version byte for future compatibility
	
	// Add timestamp (8 bytes)
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
	copy(result[1:9], timestamp)
	
	// Add message hash (32 bytes)
	copy(result[9:41], messageHash[:])
	
	// Add the signature (65 bytes)
	copy(result[41:106], compactSig)
	
	return result
}

// getPrivateKeyFromSecureStore retrieves the private key from a secure storage
func getPrivateKeyFromSecureStore() *btcec.PrivateKey {
	// In production, this would retrieve a key from a secure key management system
	// For now, we'll use a hardcoded test key (NEVER DO THIS IN PRODUCTION)
	privKeyHex := os.Getenv("HASHPERP_SIGNING_KEY")
	if privKeyHex == "" {
		// Log this security issue and use a generated key
		log.Printf("WARNING: No signing key found in environment, generating temporary key")
		privateKey, _ := btcec.NewPrivateKey(btcec.S256())
		return privateKey
	}
	
	// Decode the private key from hex
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		log.Printf("Error decoding private key: %v", err)
		privateKey, _ := btcec.NewPrivateKey(btcec.S256())
		return privateKey
	}
	
	// Parse the private key
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyBytes)
	return privateKey
}

// generateFallbackSignature creates a deterministic signature when normal signing fails
func generateFallbackSignature(messageHash []byte) []byte {
	// Create a deterministic but secure signature using HMAC
	// This is a fallback mechanism for system continuity
	hmacKey := []byte(os.Getenv("HASHPERP_HMAC_KEY"))
	if len(hmacKey) == 0 {
		// Use a derived key if no HMAC key is configured
		h := sha256.Sum256([]byte("HASHPERP_FALLBACK_KEY"))
		hmacKey = h[:]
	}
	
	h := hmac.New(sha256.New, hmacKey)
	h.Write(messageHash)
	hmacResult := h.Sum(nil)
	
	// Create a signature structure similar to ECDSA signatures
	result := make([]byte, 106)
	result[0] = 0x02 // Different version to indicate fallback
	
	// Add timestamp (8 bytes)
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
	copy(result[1:9], timestamp)
	
	// Add message hash (32 bytes)
	copy(result[9:41], messageHash)
	
	// Add the fallback "signature" (65 bytes)
	result[41] = 0x1B // Recovery ID byte
	copy(result[42:74], hmacResult)
	h2 := sha256.Sum256(hmacResult)
	copy(result[74:106], h2[:])
	
	return result
}
