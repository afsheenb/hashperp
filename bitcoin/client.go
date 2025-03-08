package bitcoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashperp/hashperp"
)

// BitcoinClientImpl implements the BitcoinClient interface
type BitcoinClientImpl struct {
	rpcURL      string
	rpcUser     string
	rpcPassword string
	httpClient  *http.Client
}

// RPCRequest represents a Bitcoin JSON-RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a Bitcoin JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// RPCError represents a Bitcoin JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewBitcoinClient creates a new Bitcoin client
func NewBitcoinClient(rpcURL, rpcUser, rpcPassword string) (hashperp.BitcoinClient, error) {
	if rpcURL == "" {
		return nil, errors.New("Bitcoin RPC URL is required")
	}

	return &BitcoinClientImpl{
		rpcURL:      rpcURL,
		rpcUser:     rpcUser,
		rpcPassword: rpcPassword,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GetCurrentBlockHeight implements BitcoinClient.GetCurrentBlockHeight
func (c *BitcoinClientImpl) GetCurrentBlockHeight(ctx context.Context) (uint64, error) {
	// Call Bitcoin RPC getblockcount
	var result uint64
	err := c.call(ctx, "getblockcount", nil, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to get block count: %w", err)
	}

	return result, nil
}

// GetBlockHashRate implements BitcoinClient.GetBlockHashRate
func (c *BitcoinClientImpl) GetBlockHashRate(ctx context.Context, blockHeight uint64) (float64, error) {
	// In a real implementation, this would fetch the hash rate at the specific block height
	// For this example, we'll use the network hash rate
	
	// Get block hash for the height
	var blockHash string
	err := c.call(ctx, "getblockhash", []interface{}{blockHeight}, &blockHash)
	if err != nil {
		return 0, fmt.Errorf("failed to get block hash: %w", err)
	}
	
	// Get block details
	var blockDetails struct {
		Difficulty float64 `json:"difficulty"`
		Time       int64   `json:"time"`
	}
	err = c.call(ctx, "getblock", []interface{}{blockHash}, &blockDetails)
	if err != nil {
		return 0, fmt.Errorf("failed to get block details: %w", err)
	}
	
	// Calculate estimated hash rate from difficulty
	// Bitcoin network targeting 600 seconds (10 minutes) per block
	// Hash rate (in hashes per second) = difficulty * 2^32 / 600
	hashRate := blockDetails.Difficulty * 4294967296 / 600
	
	// Convert to petahashes per second (PH/s)
	petaHashRate := hashRate / 1e15
	
	return petaHashRate, nil
}

// BroadcastTransaction implements BitcoinClient.BroadcastTransaction
func (c *BitcoinClientImpl) BroadcastTransaction(ctx context.Context, txHex string) (string, error) {
	// Call Bitcoin RPC sendrawtransaction
	var txid string
	err := c.call(ctx, "sendrawtransaction", []interface{}{txHex}, &txid)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return txid, nil
}

// ValidateSignature implements BitcoinClient.ValidateSignature
func (c *BitcoinClientImpl) ValidateSignature(ctx context.Context, message []byte, signature []byte, pubKey []byte) (bool, error) {
	// In a real implementation, this would validate the signature using Bitcoin cryptography
	// For this example, we'll assume the signature is valid
	
	// Convert the binary data to hex strings for Bitcoin RPC
	messageHex := fmt.Sprintf("%x", message)
	signatureHex := fmt.Sprintf("%x", signature)
	pubKeyHex := fmt.Sprintf("%x", pubKey)
	
	// Call Bitcoin RPC verifymessage
	var result bool
	err := c.call(ctx, "verifymessage", []interface{}{pubKeyHex, signatureHex, messageHex}, &result)
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}

	return result, nil
}

// GetBlockByHeight implements BitcoinClient.GetBlockByHeight
func (c *BitcoinClientImpl) GetBlockByHeight(ctx context.Context, height uint64) (map[string]interface{}, error) {
	// Get block hash for the height
	var blockHash string
	err := c.call(ctx, "getblockhash", []interface{}{height}, &blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block hash: %w", err)
	}
	
	// Get block details
	var blockDetails map[string]interface{}
	err = c.call(ctx, "getblock", []interface{}{blockHash, 2}, &blockDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to get block details: %w", err)
	}
	
	return blockDetails, nil
}

// EstimateNetworkDifficulty implements BitcoinClient.EstimateNetworkDifficulty
func (c *BitcoinClientImpl) EstimateNetworkDifficulty(ctx context.Context) (float64, error) {
	// Call Bitcoin RPC getdifficulty
	var difficulty float64
	err := c.call(ctx, "getdifficulty", nil, &difficulty)
	if err != nil {
		return 0, fmt.Errorf("failed to get network difficulty: %w", err)
	}

	return difficulty, nil
}

// call makes a Bitcoin JSON-RPC call
func (c *BitcoinClientImpl) call(ctx context.Context, method string, params []interface{}, result interface{}) error {
	// Create RPC request
	rpcReq := RPCRequest{
		JSONRPC: "1.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}
	
	// Marshal request to JSON
	reqBody, err := json.Marshal(rpcReq)
	if err != nil {
		return fmt.Errorf("failed to marshal RPC request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.rpcUser, c.rpcPassword)
	
	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status code: %d", resp.StatusCode)
	}
	
	// Decode response
	var rpcResp RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode RPC response: %w", err)
	}
	
	// Check for RPC error
	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error [%d]: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	
	// Unmarshal result
	if err := json.Unmarshal(rpcResp.Result, result); err != nil {
		return fmt.Errorf("failed to unmarshal RPC result: %w", err)
	}
	
	return nil
}
