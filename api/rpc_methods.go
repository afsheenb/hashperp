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
func decodeBase64(encoded string) ([]byte, error) {
	// Implementation would go here
	return []byte(encoded), nil
}
