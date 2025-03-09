package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/hashperp/hashperp"
)

// executeRPCMethod executes an RPC method with the given parameters
func (s *Server) executeRPCMethod(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	// Contract methods
	case "createContract":
		return s.rpcCreateContract(ctx, params)
	case "getContract":
		return s.rpcGetContract(ctx, params)
	case "getContractsByUser":
		return s.rpcGetContractsByUser(ctx, params)
	case "settleContract":
		return s.rpcSettleContract(ctx, params)
	case "exitContract":
		return s.rpcExitContract(ctx, params)
	case "rolloverContract":
		return s.rpcRolloverContract(ctx, params)
	case "executeExitPath":
		return s.rpcExecuteExitPath(ctx, params)

	// VTXO methods
	case "createVTXO":
		return s.rpcCreateVTXO(ctx, params)
	case "getVTXO":
		return s.rpcGetVTXO(ctx, params)
	case "getVTXOsByContract":
		return s.rpcGetVTXOsByContract(ctx, params)
	case "getVTXOsByUser":
		return s.rpcGetVTXOsByUser(ctx, params)
	case "swapVTXO":
		return s.rpcSwapVTXO(ctx, params)
	case "createPresignedExitTransaction":
		return s.rpcCreatePresignedExitTransaction(ctx, params)
	case "executeVTXOSweep":
		return s.rpcExecuteVTXOSweep(ctx, params)

	// Order methods
	case "placeOrder":
		return s.rpcPlaceOrder(ctx, params)
	case "cancelOrder":
		return s.rpcCancelOrder(ctx, params)
	case "getOrder":
		return s.rpcGetOrder(ctx, params)
	case "getOrdersByUser":
		return s.rpcGetOrdersByUser(ctx, params)
	case "getOrderBook":
		return s.rpcGetOrderBook(ctx, params)
	case "matchOrders":
		return s.rpcMatchOrders(ctx, params)

	// Swap offer methods
	case "createSwapOffer":
		return s.rpcCreateSwapOffer(ctx, params)
	case "acceptSwapOffer":
		return s.rpcAcceptSwapOffer(ctx, params)
	case "cancelSwapOffer":
		return s.rpcCancelSwapOffer(ctx, params)
	case "getSwapOffer":
		return s.rpcGetSwapOffer(ctx, params)
	case "getSwapOffersByUser":
		return s.rpcGetSwapOffersByUser(ctx, params)
	case "getSwapOffersByContract":
		return s.rpcGetSwapOffersByContract(ctx, params)

	// Market data methods
	case "getCurrentHashRate":
		return s.rpcGetCurrentHashRate(ctx, params)
	case "getHistoricalHashRate":
		return s.rpcGetHistoricalHashRate(ctx, params)
	case "getHashRateAtBlockHeight":
		return s.rpcGetHashRateAtBlockHeight(ctx, params)
	case "calculateBTCPerPHPerDay":
		return s.rpcCalculateBTCPerPHPerDay(ctx, params)

	// Transaction methods
	case "getTransaction":
		return s.rpcGetTransaction(ctx, params)
	case "getTransactionsByUser":
		return s.rpcGetTransactionsByUser(ctx, params)
	case "getTransactionsByContract":
		return s.rpcGetTransactionsByContract(ctx, params)

	// Utility methods
	case "getCurrentBlockHeight":
		return s.rpcGetCurrentBlockHeight(ctx, params)
	case "validateContractParameters":
		return s.rpcValidateContractParameters(ctx, params)

	default:
		return nil, &RPCError{
			Code:    -32601,
			Message: "Method not found",
			Data:    method,
		}
	}
}

// Contract RPC Methods

// rpcCreateContract creates a new contract
func (s *Server) rpcCreateContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		BuyerID           string  `json:"buyer_id"`
		SellerID          string  `json:"seller_id"`
		ContractType      string  `json:"contract_type"`
		StrikeRate        float64 `json:"strike_rate"`
		ExpiryBlockHeight uint64  `json:"expiry_block_height"`
		Size              float64 `json:"size"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	contract, err := s.service.CreateContract(
		ctx,
		req.BuyerID,
		req.SellerID,
		hashperp.ContractType(req.ContractType),
		req.StrikeRate,
		req.ExpiryBlockHeight,
		req.Size,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract: %w", err)
	}

	return contract, nil
}

// rpcGetContract retrieves a contract by ID
func (s *Server) rpcGetContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID string `json:"contract_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	contract, err := s.service.GetContract(ctx, req.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	return contract, nil
}

// rpcGetContractsByUser retrieves contracts for a user
func (s *Server) rpcGetContractsByUser(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		UserID string   `json:"user_id"`
		Status []string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Convert string statuses to ContractStatus
	var statuses []hashperp.ContractStatus
	for _, status := range req.Status {
		statuses = append(statuses, hashperp.ContractStatus(status))
	}

	contracts, err := s.service.GetContractsByUser(ctx, req.UserID, statuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get contracts by user: %w", err)
	}

	return contracts, nil
}

// rpcSettleContract settles a contract
func (s *Server) rpcSettleContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID string `json:"contract_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	tx, err := s.service.SettleContract(ctx, req.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to settle contract: %w", err)
	}

	return tx, nil
}

// rpcExitContract exits a contract
func (s *Server) rpcExitContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID string `json:"contract_id"`
		UserID     string `json:"user_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	tx, err := s.service.ExitContract(ctx, req.ContractID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to exit contract: %w", err)
	}

	return tx, nil
}

// rpcRolloverContract rolls over a contract
func (s *Server) rpcRolloverContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID          string `json:"contract_id"`
		NewExpiryBlockHeight uint64 `json:"new_expiry_block_height"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	contract, tx, err := s.service.RolloverContract(ctx, req.ContractID, req.NewExpiryBlockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to rollover contract: %w", err)
	}

	return map[string]interface{}{
		"contract":    contract,
		"transaction": tx,
	}, nil
}

// rpcExecuteExitPath executes an exit path
func (s *Server) rpcExecuteExitPath(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID   string `json:"contract_id"`
		UserID       string `json:"user_id"`
		ExitPathType string `json:"exit_path_type"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	tx, err := s.service.ExecuteExitPath(ctx, req.ContractID, req.UserID, req.ExitPathType)
	if err != nil {
		return nil, fmt.Errorf("failed to execute exit path: %w", err)
	}

	return tx, nil
}

// VTXO RPC Methods

// rpcCreateVTXO creates a new VTXO
func (s *Server) rpcCreateVTXO(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID    string  `json:"contract_id"`
		OwnerID       string  `json:"owner_id"`
		Amount        float64 `json:"amount"`
		ScriptPath    string  `json:"script_path"`
		SignatureData string  `json:"signature_data"` // Base64 encoded
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Decode signature data
	var signatureData []byte
	if req.SignatureData != "" {
		var err error
		signatureData, err = decodeBase64(req.SignatureData)
		if err != nil {
			return nil, &RPCError{
				Code:    -32602,
				Message: "Invalid signature data",
				Data:    err.Error(),
			}
		}
	}

	vtxo, err := s.service.CreateVTXO(
		ctx,
		req.ContractID,
		req.OwnerID,
		req.Amount,
		req.ScriptPath,
		signatureData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create VTXO: %w", err)
	}

	return vtxo, nil
}

// Additional RPC method implementations would follow a similar pattern.
// For brevity, I'm not including all method implementations here.

// decodeBase64 decodes a base64 string

// rpcGetVTXO retrieves a VTXO by ID
func (s *Server) rpcGetVTXO(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		VTXOID string `json:"vtxo_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	vtxo, err := s.service.GetVTXO(ctx, req.VTXOID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXO: %w", err)
	}

	return vtxo, nil
}

// rpcGetVTXOsByContract retrieves all VTXOs for a specific contract
func (s *Server) rpcGetVTXOsByContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID string `json:"contract_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	vtxos, err := s.service.GetVTXOsByContract(ctx, req.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXOs by contract: %w", err)
	}

	return vtxos, nil
}

// rpcGetVTXOsByUser retrieves all VTXOs for a specific user
func (s *Server) rpcGetVTXOsByUser(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		UserID     string `json:"user_id"`
		OnlyActive bool   `json:"only_active,omitempty"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	vtxos, err := s.service.GetVTXOsByUser(ctx, req.UserID, req.OnlyActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get VTXOs by user: %w", err)
	}

	return vtxos, nil
}

// rpcSwapVTXO swaps a VTXO between users
func (s *Server) rpcSwapVTXO(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		VTXOID          string `json:"vtxo_id"`
		NewOwnerID      string `json:"new_owner_id"`
		NewSignatureData string `json:"new_signature_data"` // Base64 encoded
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Decode the signature data
	signatureData, err := decodeBase64(req.NewSignatureData)
	if err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid signature data",
			Data:    err.Error(),
		}
	}

	vtxo, tx, err := s.service.SwapVTXO(ctx, req.VTXOID, req.NewOwnerID, signatureData)
	if err != nil {
		return nil, fmt.Errorf("failed to swap VTXO: %w", err)
	}

	return map[string]interface{}{
		"vtxo":        vtxo,
		"transaction": tx,
	}, nil
}

// rpcCreatePresignedExitTransaction creates a pre-signed exit transaction for a VTXO
func (s *Server) rpcCreatePresignedExitTransaction(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		VTXOID        string `json:"vtxo_id"`
		SignatureData string `json:"signature_data"` // Base64 encoded
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Decode the signature data
	signatureData, err := decodeBase64(req.SignatureData)
	if err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid signature data",
			Data:    err.Error(),
		}
	}

	txID, err := s.service.CreatePresignedExitTransaction(ctx, req.VTXOID, signatureData)
	if err != nil {
		return nil, fmt.Errorf("failed to create pre-signed exit transaction: %w", err)
	}

	return map[string]interface{}{
		"transaction_id": txID,
	}, nil
}

// rpcExecuteVTXOSweep executes a VTXO sweep
func (s *Server) rpcExecuteVTXOSweep(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		VTXOID string `json:"vtxo_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	tx, err := s.service.ExecuteVTXOSweep(ctx, req.VTXOID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute VTXO sweep: %w", err)
	}

	return tx, nil
}

// rpcPlaceOrder places a new order in the order book
func (s *Server) rpcPlaceOrder(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		UserID           string  `json:"user_id"`
		OrderType        string  `json:"order_type"`
		ContractType     string  `json:"contract_type"`
		StrikeRate       float64 `json:"strike_rate"`
		ExpiryBlockHeight uint64  `json:"expiry_block_height"`
		Size             float64 `json:"size"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	order, err := s.service.PlaceOrder(
		ctx,
		req.UserID,
		hashperp.OrderType(req.OrderType),
		hashperp.ContractType(req.ContractType),
		req.StrikeRate,
		req.ExpiryBlockHeight,
		req.Size,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	return order, nil
}

// rpcCancelOrder cancels an existing order
func (s *Server) rpcCancelOrder(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		OrderID string `json:"order_id"`
		UserID  string `json:"user_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	err := s.service.CancelOrder(ctx, req.OrderID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// rpcGetOrder retrieves an order by ID
func (s *Server) rpcGetOrder(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		OrderID string `json:"order_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	order, err := s.service.GetOrder(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

// rpcGetOrdersByUser retrieves all orders for a specific user
func (s *Server) rpcGetOrdersByUser(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		UserID string   `json:"user_id"`
		Status []string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Convert string statuses to OrderStatus
	var statuses []hashperp.OrderStatus
	for _, status := range req.Status {
		statuses = append(statuses, hashperp.OrderStatus(status))
	}

	orders, err := s.service.GetOrdersByUser(ctx, req.UserID, statuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by user: %w", err)
	}

	return orders, nil
}

// rpcGetOrderBook retrieves the current order book for a given contract type
func (s *Server) rpcGetOrderBook(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractType     string `json:"contract_type"`
		ExpiryBlockHeight uint64 `json:"expiry_block_height"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	orders, err := s.service.GetOrderBook(
		ctx,
		hashperp.ContractType(req.ContractType),
		req.ExpiryBlockHeight,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	// Group orders by type for easier client-side rendering
	buyOrders := []*hashperp.Order{}
	sellOrders := []*hashperp.Order{}
	
	for _, order := range orders {
		if order.OrderType == hashperp.BUY {
			buyOrders = append(buyOrders, order)
		} else {
			sellOrders = append(sellOrders, order)
		}
	}

	return map[string]interface{}{
		"buy_orders":  buyOrders,
		"sell_orders": sellOrders,
	}, nil
}

// rpcMatchOrders attempts to match buy and sell orders
func (s *Server) rpcMatchOrders(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// This endpoint doesn't require any parameters as it processes all available orders
	contracts, err := s.service.MatchOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to match orders: %w", err)
	}

	return map[string]interface{}{
		"matched_contracts": contracts,
	}, nil
}

// rpcCreateSwapOffer creates a new swap offer
func (s *Server) rpcCreateSwapOffer(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		OfferorID   string  `json:"offeror_id"`
		VTXOID      string  `json:"vtxo_id"`
		OfferedRate float64 `json:"offered_rate"`
		ExpiryHours int     `json:"expiry_hours"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Calculate expiry time
	expiryTime := time.Now().UTC().Add(time.Duration(req.ExpiryHours) * time.Hour)

	offer, err := s.service.CreateSwapOffer(
		ctx,
		req.OfferorID,
		req.VTXOID,
		req.OfferedRate,
		expiryTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create swap offer: %w", err)
	}

	return offer, nil
}

// rpcAcceptSwapOffer accepts a swap offer
func (s *Server) rpcAcceptSwapOffer(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		OfferID    string `json:"offer_id"`
		AcceptorID string `json:"acceptor_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	tx, err := s.service.AcceptSwapOffer(ctx, req.OfferID, req.AcceptorID)
	if err != nil {
		return nil, fmt.Errorf("failed to accept swap offer: %w", err)
	}

	return tx, nil
}

// rpcCancelSwapOffer cancels a swap offer
func (s *Server) rpcCancelSwapOffer(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		OfferID   string `json:"offer_id"`
		OfferorID string `json:"offeror_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	err := s.service.CancelSwapOffer(ctx, req.OfferID, req.OfferorID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel swap offer: %w", err)
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// rpcGetSwapOffer retrieves a swap offer by ID
func (s *Server) rpcGetSwapOffer(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		OfferID string `json:"offer_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	offer, err := s.service.GetSwapOffer(ctx, req.OfferID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offer: %w", err)
	}

	return offer, nil
}

// rpcGetSwapOffersByUser retrieves all swap offers for a specific user
func (s *Server) rpcGetSwapOffersByUser(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		UserID    string `json:"user_id"`
		IsOfferor bool   `json:"is_offeror"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	offers, err := s.service.GetSwapOffersByUser(ctx, req.UserID, req.IsOfferor)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offers by user: %w", err)
	}

	return offers, nil
}

// rpcGetSwapOffersByContract retrieves all swap offers for a specific contract
func (s *Server) rpcGetSwapOffersByContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID string `json:"contract_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	offers, err := s.service.GetSwapOffersByContract(ctx, req.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap offers by contract: %w", err)
	}

	return offers, nil
}

// rpcGetCurrentHashRate retrieves the current hash rate
func (s *Server) rpcGetCurrentHashRate(ctx context.Context, params json.RawMessage) (interface{}, error) {
	hashRate, err := s.service.GetCurrentHashRate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current hash rate: %w", err)
	}

	return hashRate, nil
}

// rpcGetHistoricalHashRate retrieves historical hash rate data
func (s *Server) rpcGetHistoricalHashRate(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		StartTime int64 `json:"start_time"` // Unix timestamp
		EndTime   int64 `json:"end_time"`   // Unix timestamp
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	startTime := time.Unix(req.StartTime, 0).UTC()
	endTime := time.Unix(req.EndTime, 0).UTC()

	hashRates, err := s.service.GetHistoricalHashRate(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical hash rate: %w", err)
	}

	return hashRates, nil
}

// rpcGetHashRateAtBlockHeight retrieves hash rate data at a specific block height
func (s *Server) rpcGetHashRateAtBlockHeight(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		BlockHeight uint64 `json:"block_height"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	hashRate, err := s.service.GetHashRateAtBlockHeight(ctx, req.BlockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash rate at block height: %w", err)
	}

	return hashRate, nil
}

// rpcCalculateBTCPerPHPerDay calculates the BTC per PetaHash per Day rate
func (s *Server) rpcCalculateBTCPerPHPerDay(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		HashRate    float64 `json:"hash_rate"`
		BlockHeight uint64  `json:"block_height,omitempty"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// If block height is not provided, use current block height
	blockHeight := req.BlockHeight
	if blockHeight == 0 {
		var err error
		blockHeight, err = s.service.GetCurrentBlockHeight(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current block height: %w", err)
		}
	}

	btcPerPHPerDay, err := s.service.CalculateBTCPerPHPerDay(ctx, req.HashRate, blockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate BTC per PH per day: %w", err)
	}

	return map[string]interface{}{
		"btc_per_ph_per_day": btcPerPHPerDay,
	}, nil
}

// rpcGetTransaction retrieves a transaction by ID
func (s *Server) rpcGetTransaction(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TransactionID string `json:"transaction_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	tx, err := s.service.GetTransaction(ctx, req.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return tx, nil
}

// rpcGetTransactionsByUser retrieves all transactions for a specific user
func (s *Server) rpcGetTransactionsByUser(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		UserID string   `json:"user_id"`
		Types  []string `json:"types,omitempty"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Convert string types to TransactionType
	var types []hashperp.TransactionType
	for _, t := range req.Types {
		types = append(types, hashperp.TransactionType(t))
	}

	txs, err := s.service.GetTransactionsByUser(ctx, req.UserID, types)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by user: %w", err)
	}

	return txs, nil
}

// rpcGetTransactionsByContract retrieves all transactions for a specific contract
func (s *Server) rpcGetTransactionsByContract(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractID string `json:"contract_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	txs, err := s.service.GetTransactionsByContract(ctx, req.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by contract: %w", err)
	}

	return txs, nil
}

// rpcGetCurrentBlockHeight retrieves the current block height
func (s *Server) rpcGetCurrentBlockHeight(ctx context.Context, params json.RawMessage) (interface{}, error) {
	blockHeight, err := s.service.GetCurrentBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block height: %w", err)
	}

	return map[string]interface{}{
		"block_height": blockHeight,
	}, nil
}

// rpcValidateContractParameters validates contract parameters
func (s *Server) rpcValidateContractParameters(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ContractType      string  `json:"contract_type"`
		StrikeRate        float64 `json:"strike_rate"`
		ExpiryBlockHeight uint64  `json:"expiry_block_height"`
		Size              float64 `json:"size"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	err := s.service.ValidateContractParameters(
		ctx,
		hashperp.ContractType(req.ContractType),
		req.StrikeRate,
		req.ExpiryBlockHeight,
		req.Size,
	)
	if err != nil {
		return map[string]interface{}{
			"valid":  false,
			"reason": err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"valid": true,
	}, nil
}

// decodeBase64 decodes a base64 string
func decodeBase64(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, errors.New("empty string")
	}
	
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 string: %w", err)
	}
	
	return decoded, nil
}
