package hashperp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// OrderRepository defines the data access interface for orders
type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id string) (*Order, error)
	FindByUser(ctx context.Context, userID string, status []OrderStatus) ([]*Order, error)
	FindByContractType(ctx context.Context, contractType ContractType, expiryBlockHeight uint64) ([]*Order, error)
	FindOpenOrders(ctx context.Context) ([]*Order, error)
	Update(ctx context.Context, order *Order) error
	Delete(ctx context.Context, id string) error
}

// orderBookService implements the OrderBookManager interface
type orderBookService struct {
	orderRepo      OrderRepository
	contractRepo   ContractRepository
	contractMgr    ContractManager
	transactionRepo TransactionRepository
	btcClient      BitcoinClient
	blockHeight    uint64 // Current block height, regularly updated
}

// NewOrderBookService creates a new order book service
func NewOrderBookService(
	orderRepo OrderRepository,
	contractRepo ContractRepository,
	contractMgr ContractManager,
	transactionRepo TransactionRepository,
	btcClient BitcoinClient,
) OrderBookManager {
	return &orderBookService{
		orderRepo:      orderRepo,
		contractRepo:   contractRepo,
		contractMgr:    contractMgr,
		transactionRepo: transactionRepo,
		btcClient:      btcClient,
	}
}

// PlaceOrder implements OrderBookManager.PlaceOrder
func (s *orderBookService) PlaceOrder(
	ctx context.Context,
	userID string,
	orderType OrderType,
	contractType ContractType,
	strikeRate float64,
	expiryBlockHeight uint64,
	size float64,
) (*Order, error) {
	// 1. Validate inputs
	if orderType != BUY && orderType != SELL {
		return nil, errors.New("invalid order type")
	}

	if contractType != CALL && contractType != PUT {
		return nil, errors.New("invalid contract type")
	}

	if strikeRate <= 0 {
		return nil, errors.New("strike rate must be positive")
	}

	if size <= 0 {
		return nil, errors.New("size must be positive")
	}

	// 2. Validate expiry block height
	currentBlockHeight, err := s.btcClient.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}
	s.blockHeight = currentBlockHeight // Update cached block height

	if expiryBlockHeight <= currentBlockHeight {
		return nil, ErrInvalidBlockHeight
	}

	// 3. Calculate human-readable expiry date
	expiryDate := calculateExpiryDate(expiryBlockHeight, currentBlockHeight)

	// 4. Create the order
	order := &Order{
		ID:                generateUniqueID(),
		UserID:            userID,
		OrderType:         orderType,
		ContractType:      contractType,
		StrikeRate:        strikeRate,
		ExpiryBlockHeight: expiryBlockHeight,
		ExpiryDate:        expiryDate,
		Size:              size,
		Status:            OPEN,
		CreationTime:      time.Now().UTC(),
	}

	// 5. Save the order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 6. Try to match the order immediately
	matched, err := s.tryMatchOrder(ctx, order)
	if err != nil {
		// If matching fails, we still keep the order but log the error
		fmt.Printf("failed to match order: %v\n", err)
	}

	// 7. If the order was matched, update it
	if matched {
		order, err = s.orderRepo.FindByID(ctx, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated order: %w", err)
		}
	}

	return order, nil
}

// CancelOrder implements OrderBookManager.CancelOrder
func (s *orderBookService) CancelOrder(
	ctx context.Context,
	orderID string,
	userID string,
) error {
	// 1. Get the order
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return errors.New("order not found")
	}

	// 2. Validate the user is the owner of this order
	if order.UserID != userID {
		return errors.New("user is not the owner of this order")
	}

	// 3. Validate order status
	if order.Status != OPEN {
		return errors.New("order is not open for cancellation")
	}

	// 4. Update the order status to CANCELED
	order.Status = CANCELED
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// GetOrder implements OrderBookManager.GetOrder
func (s *orderBookService) GetOrder(
	ctx context.Context,
	orderID string,
) (*Order, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return nil, errors.New("order not found")
	}
	return order, nil
}

// GetOrdersByUser implements OrderBookManager.GetOrdersByUser
func (s *orderBookService) GetOrdersByUser(
	ctx context.Context,
	userID string,
	status []OrderStatus,
) ([]*Order, error) {
	orders, err := s.orderRepo.FindByUser(ctx, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by user: %w", err)
	}
	return orders, nil
}

// GetOrderBook implements OrderBookManager.GetOrderBook
func (s *orderBookService) GetOrderBook(
	ctx context.Context,
	contractType ContractType,
	expiryBlockHeight uint64,
) ([]*Order, error) {
	// Get orders for the specified contract type and expiry
	orders, err := s.orderRepo.FindByContractType(ctx, contractType, expiryBlockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for contract type: %w", err)
	}

	// Filter for only open orders
	var openOrders []*Order
	for _, order := range orders {
		if order.Status == OPEN {
			openOrders = append(openOrders, order)
		}
	}

	return openOrders, nil
}

// MatchOrders implements OrderBookManager.MatchOrders
// This is the core function that attempts to match open buy and sell orders
func (s *orderBookService) MatchOrders(ctx context.Context) ([]*Contract, error) {
	// 1. Get all open orders
	allOrders, err := s.orderRepo.FindOpenOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	// 2. Group orders by contract type and expiry
	orderGroups := make(map[string][]*Order)
	for _, order := range allOrders {
		key := fmt.Sprintf("%s-%d", order.ContractType, order.ExpiryBlockHeight)
		orderGroups[key] = append(orderGroups[key], order)
	}

	// 3. Match orders within each group
	var matchedContracts []*Contract

	for _, orders := range orderGroups {
		// Separate buy and sell orders
		var buyOrders []*Order
		var sellOrders []*Order

		for _, order := range orders {
			if order.OrderType == BUY {
				buyOrders = append(buyOrders, order)
			} else {
				sellOrders = append(sellOrders, order)
			}
		}

		// Sort buy orders by price (highest first)
		sort.Slice(buyOrders, func(i, j int) bool {
			return buyOrders[i].StrikeRate > buyOrders[j].StrikeRate
		})

		// Sort sell orders by price (lowest first)
		sort.Slice(sellOrders, func(i, j int) bool {
			return sellOrders[i].StrikeRate < sellOrders[j].StrikeRate
		})

		// Match orders
		for _, buyOrder := range buyOrders {
			if buyOrder.Status != OPEN {
				continue
			}

			for _, sellOrder := range sellOrders {
				if sellOrder.Status != OPEN {
					continue
				}

				// Check if the buy price is >= sell price
				if buyOrder.StrikeRate >= sellOrder.StrikeRate {
					// Match the orders
					contract, err := s.createContractFromOrders(ctx, buyOrder, sellOrder)
					if err != nil {
						fmt.Printf("failed to create contract from orders: %v\n", err)
						continue
					}

					matchedContracts = append(matchedContracts, contract)

					// Update order statuses
					buyOrder.Status = MATCHED
					buyOrder.MatchedOrderID = sellOrder.ID
					buyOrder.ResultingContractID = contract.ID

					sellOrder.Status = MATCHED
					sellOrder.MatchedOrderID = buyOrder.ID
					sellOrder.ResultingContractID = contract.ID

					if err := s.orderRepo.Update(ctx, buyOrder); err != nil {
						fmt.Printf("failed to update buy order: %v\n", err)
					}

					if err := s.orderRepo.Update(ctx, sellOrder); err != nil {
						fmt.Printf("failed to update sell order: %v\n", err)
					}

					break // Move to the next buy order
				}
			}
		}
	}

	return matchedContracts, nil
}

// tryMatchOrder attempts to match a single order with existing open orders
func (s *orderBookService) tryMatchOrder(
	ctx context.Context,
	order *Order,
) (bool, error) {
	// 1. Get open orders of the opposite type with the same contract type and expiry
	oppositeType := SELL
	if order.OrderType == SELL {
		oppositeType = BUY
	}

	// 2. Get all open orders
	allOrders, err := s.orderRepo.FindOpenOrders(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get open orders: %w", err)
	}

	// 3. Filter for compatible orders
	var compatibleOrders []*Order
	for _, o := range allOrders {
		if o.OrderType == oppositeType &&
			o.ContractType == order.ContractType &&
			o.ExpiryBlockHeight == order.ExpiryBlockHeight &&
			o.Status == OPEN {
			compatibleOrders = append(compatibleOrders, o)
		}
	}

	// 4. Sort compatible orders by price
	if order.OrderType == BUY {
		// For buy orders, sort sell orders by price (lowest first)
		sort.Slice(compatibleOrders, func(i, j int) bool {
			return compatibleOrders[i].StrikeRate < compatibleOrders[j].StrikeRate
		})
	} else {
		// For sell orders, sort buy orders by price (highest first)
		sort.Slice(compatibleOrders, func(i, j int) bool {
			return compatibleOrders[i].StrikeRate > compatibleOrders[j].StrikeRate
		})
	}

	// 5. Try to match with each compatible order
	for _, compatibleOrder := range compatibleOrders {
		// Check if the prices match
		if (order.OrderType == BUY && order.StrikeRate >= compatibleOrder.StrikeRate) ||
			(order.OrderType == SELL && order.StrikeRate <= compatibleOrder.StrikeRate) {
			
			// Determine buyer and seller
			var buyOrder, sellOrder *Order
			if order.OrderType == BUY {
				buyOrder = order
				sellOrder = compatibleOrder
			} else {
				buyOrder = compatibleOrder
				sellOrder = order
			}

			// Create a contract from the matched orders
			contract, err := s.createContractFromOrders(ctx, buyOrder, sellOrder)
			if err != nil {
				return false, fmt.Errorf("failed to create contract from orders: %w", err)
			}

			// Update order statuses
			buyOrder.Status = MATCHED
			buyOrder.MatchedOrderID = sellOrder.ID
			buyOrder.ResultingContractID = contract.ID

			sellOrder.Status = MATCHED
			sellOrder.MatchedOrderID = buyOrder.ID
			sellOrder.ResultingContractID = contract.ID

			if err := s.orderRepo.Update(ctx, buyOrder); err != nil {
				return false, fmt.Errorf("failed to update buy order: %w", err)
			}

			if err := s.orderRepo.Update(ctx, sellOrder); err != nil {
				return false, fmt.Errorf("failed to update sell order: %w", err)
			}

			return true, nil
		}
	}

	return false, nil
}

// createContractFromOrders creates a new contract from matched buy and sell orders
func (s *orderBookService) createContractFromOrders(
	ctx context.Context,
	buyOrder *Order,
	sellOrder *Order,
) (*Contract, error) {
	// Validate that the orders are compatible
	if buyOrder.ContractType != sellOrder.ContractType {
		return nil, errors.New("orders have different contract types")
	}

	if buyOrder.ExpiryBlockHeight != sellOrder.ExpiryBlockHeight {
		return nil, errors.New("orders have different expiry block heights")
	}

	// Determine the contract size (minimum of the two orders)
	size := buyOrder.Size
	if sellOrder.Size < size {
		size = sellOrder.Size
	}

	// Create the contract using the ContractManager
	contract, err := s.contractMgr.CreateContract(
		ctx,
		buyOrder.UserID,    // Buyer ID
		sellOrder.UserID,   // Seller ID
		buyOrder.ContractType,
		buyOrder.StrikeRate, // Use the buyer's strike rate (ensures the buyer got a price they're ok with)
		buyOrder.ExpiryBlockHeight,
		size,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create contract from orders: %w", err)
	}

	return contract, nil
}
