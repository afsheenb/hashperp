package hashperp

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Common validation errors
var (
	ErrEmptyID          = errors.New("ID cannot be empty")
	ErrInvalidID        = errors.New("invalid ID format")
	ErrEmptyUserID      = errors.New("user ID cannot be empty")
	ErrInvalidUserID    = errors.New("invalid user ID format")
	ErrNegativeAmount   = errors.New("amount must be positive")
	ErrInvalidRate      = errors.New("rate must be positive")
	ErrMissingSignature = errors.New("signature data is required")
	ErrInvalidTimeRange = errors.New("invalid time range")
)

// Basic regex for UUID v4 validation
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// ValidateUUID validates that a string is a valid UUID
func ValidateUUID(id string) error {
	if id == "" {
		return ErrEmptyID
	}
	
	if !uuidRegex.MatchString(id) {
		return ErrInvalidID
	}
	
	return nil
}

// ValidateUserID validates a user ID
func ValidateUserID(userID string) error {
	if userID == "" {
		return ErrEmptyUserID
	}
	
	if !uuidRegex.MatchString(userID) {
		return ErrInvalidUserID
	}
	
	return nil
}

// ValidateAmount validates that an amount is positive and within bounds
func ValidateAmount(amount float64, min, max float64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	
	if amount < min {
		return fmt.Errorf("amount below minimum: %v", min)
	}
	
	if max > 0 && amount > max {
		return fmt.Errorf("amount exceeds maximum: %v", max)
	}
	
	return nil
}

// ValidateRate validates that a rate is positive and within bounds
func ValidateRate(rate float64, min, max float64) error {
	if rate <= 0 {
		return ErrInvalidRate
	}
	
	if rate < min {
		return fmt.Errorf("rate below minimum: %v", min)
	}
	
	if max > 0 && rate > max {
		return fmt.Errorf("rate exceeds maximum: %v", max)
	}
	
	return nil
}

// ValidateSignatureData validates that signature data is present and has minimum length
func ValidateSignatureData(data []byte, minLength int) error {
	if data == nil || len(data) == 0 {
		return ErrMissingSignature
	}
	
	if len(data) < minLength {
		return fmt.Errorf("signature data too short (minimum %d bytes required)", minLength)
	}
	
	return nil
}

// ValidateTimeRange validates a time range
func ValidateTimeRange(startTime, endTime time.Time) error {
	if startTime.IsZero() || endTime.IsZero() {
		return ErrInvalidTimeRange
	}
	
	if endTime.Before(startTime) {
		return fmt.Errorf("%w: end time is before start time", ErrInvalidTimeRange)
	}
	
	return nil
}

// ValidateContractType validates that a contract type is valid
func ValidateContractType(contractType ContractType) error {
	if contractType != CALL && contractType != PUT {
		return fmt.Errorf("invalid contract type: %s", contractType)
	}
	
	return nil
}

// ValidateOrderType validates that an order type is valid
func ValidateOrderType(orderType OrderType) error {
	if orderType != BUY && orderType != SELL {
		return fmt.Errorf("invalid order type: %s", orderType)
	}
	
	return nil
}

// ValidateContractStatus validates that a contract status is valid
func ValidateContractStatus(status ContractStatus) error {
	validStatuses := map[ContractStatus]bool{
		PENDING:              true,
		ACTIVE:               true,
		SETTLED:              true,
		EXITED:               true,
		ROLLED_OVER:          true,
		SETTLEMENT_PENDING:   true,
		SETTLEMENT_IN_PROGRESS: true,
		COMPLETED:            true,
		CLOSE_TO_EXPIRY:      true,
	}
	
	if !validStatuses[status] {
		return fmt.Errorf("invalid contract status: %s", status)
	}
	
	return nil
}

// ValidateOrderStatus validates that an order status is valid
func ValidateOrderStatus(status OrderStatus) error {
	validStatuses := map[OrderStatus]bool{
		OPEN:     true,
		MATCHED:  true,
		CANCELED: true,
		EXPIRED:  true,
	}
	
	if !validStatuses[status] {
		return fmt.Errorf("invalid order status: %s", status)
	}
	
	return nil
}

// ValidateSwapOfferStatus validates that a swap offer status is valid
func ValidateSwapOfferStatus(status SwapOfferStatus) error {
	validStatuses := map[SwapOfferStatus]bool{
		OFFER_OPEN:     true,
		OFFER_ACCEPTED: true,
		OFFER_EXPIRED:  true,
		OFFER_CANCELED: true,
		OFFER_REJECTED: true,
	}
	
	if !validStatuses[status] {
		return fmt.Errorf("invalid swap offer status: %s", status)
	}
	
	return nil
}

// ValidateTransactionType validates that a transaction type is valid
func ValidateTransactionType(txType TransactionType) error {
	validTypes := map[TransactionType]bool{
		CONTRACT_CREATION:   true,
		CONTRACT_SETTLEMENT: true,
		VTXO_SWAP:           true,
		VTXO_ROLLOVER:       true,
		CONTRACT_ROLLOVER:   true,
		EXIT_PATH_EXECUTION: true,
	}
	
	if !validTypes[txType] {
		return fmt.Errorf("invalid transaction type: %s", txType)
	}
	
	return nil
}
