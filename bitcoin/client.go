package bitcoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
)

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
import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

// CreateAndBroadcastTx implements BitcoinClient.CreateAndBroadcastTx
// This builds an actual Bitcoin transaction from the provided script and broadcasts it to the network
func (c *BitcoinClientImpl) CreateAndBroadcastTx(
	ctx context.Context,
	scriptHex string,
	amount float64,
	destinationAddress string,
) (string, error) {
	// 1. Convert the amount from BTC to satoshis
	amountSatoshis := uint64(amount * 100000000) // 1 BTC = 100,000,000 satoshis
	
	// 2. Create a raw transaction with the script
	// Find appropriate UTXOs to use as inputs
	utxosReq := map[string]interface{}{
		"address": destinationAddress,
		"minconf": 1,
	}
	
	var utxosResp []map[string]interface{}
	if err := c.call(ctx, "listunspent", []interface{}{1, 9999999, []string{destinationAddress}}, &utxosResp); err != nil {
		return "", fmt.Errorf("failed to list unspent outputs: %w", err)
	}
	
	// Ensure we have enough funds
	var totalInput float64
	var inputs []map[string]interface{}
	for _, utxo := range utxosResp {
		totalInput += utxo["amount"].(float64)
		inputs = append(inputs, map[string]interface{}{
			"txid": utxo["txid"].(string),
			"vout": utxo["vout"].(float64),
		})
		
		if totalInput >= amount+0.0001 { // Adding a small fee
			break
		}
	}
	
	if totalInput < amount+0.0001 {
		return "", fmt.Errorf("insufficient funds: have %f BTC, need %f BTC", totalInput, amount+0.0001)
	}
	
	// Create the transaction outputs
	outputs := map[string]interface{}{
		destinationAddress: amount,
	}
	
	// Calculate change amount if necessary
	change := totalInput - amount - 0.0001 // Subtracting fee
	if change > 0.00001 { // Only add change output if it's not dust
		outputs[destinationAddress+"_change"] = change
	}
	
	// Create raw transaction
	var rawTxHex string
	if err := c.call(ctx, "createrawtransaction", []interface{}{inputs, outputs}, &rawTxHex); err != nil {
		return "", fmt.Errorf("failed to create raw transaction: %w", err)
	}
	
	// Sign the transaction
	signReq := map[string]interface{}{
		"hexstring": rawTxHex,
		"prevtxs":   []interface{}{},
		"privkeys":  []interface{}{},
		"script":    scriptHex,
	}
	
	var signResp map[string]interface{}
	if err := c.call(ctx, "signrawtransactionwithkey", []interface{}{signReq}, &signResp); err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}
	
	if !signResp["complete"].(bool) {
		return "", fmt.Errorf("failed to completely sign transaction")
	}
	
	signedTxHex := signResp["hex"].(string)
	
	// Send the transaction to the network
	var txid string
	if err := c.call(ctx, "sendrawtransaction", []interface{}{signedTxHex}, &txid); err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	
	return txid, nil
}

// GenerateBlockchainProof creates a proof that can be verified against the blockchain
// This is used for validating on-chain settlements and exit paths
func (c *BitcoinClientImpl) GenerateBlockchainProof(
	ctx context.Context,
	txid string,
	blockHash string,
) (map[string]interface{}, error) {
	// 1. Get the transaction data
	var txData map[string]interface{}
	if err := c.call(ctx, "gettransaction", []interface{}{txid}, &txData); err != nil {
		return nil, fmt.Errorf("failed to get transaction data: %w", err)
	}
	
	// 2. Get the block data
	var blockData map[string]interface{}
	if err := c.call(ctx, "getblock", []interface{}{blockHash, 2}, &blockData); err != nil {
		return nil, fmt.Errorf("failed to get block data: %w", err)
	}
	
	// 3. Generate Merkle proof
	// Find the transaction index in the block
	var txIndex int = -1
	transactions, ok := blockData["tx"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid block data format")
	}
	
	for i, tx := range transactions {
		if tx.(map[string]interface{})["txid"].(string) == txid {
			txIndex = i
			break
		}
	}
	
	if txIndex == -1 {
		return nil, fmt.Errorf("transaction not found in the specified block")
	}
	
	// Calculate Merkle proof nodes
	var merkleProof []string
	nodeCount := len(transactions)
	currentLayer := make([]string, nodeCount)
	
	// Initialize the first layer with transaction hashes
	for i, tx := range transactions {
		currentLayer[i] = tx.(map[string]interface{})["txid"].(string)
	}
	
	// Calculate the path from the transaction to the Merkle root
	currentTxPos := txIndex
	for nodeCount > 1 {
		// If we have an odd number of nodes, duplicate the last one
		if nodeCount%2 == 1 {
			currentLayer = append(currentLayer, currentLayer[nodeCount-1])
			nodeCount++
		}
		
		// Create the next layer
		nextLayer := make([]string, nodeCount/2)
		for i := 0; i < nodeCount; i += 2 {
			// Concatenate adjacent hashes and hash them
			combinedHash := currentLayer[i] + currentLayer[i+1]
			hashBytes := sha256.Sum256([]byte(combinedHash))
			nextLayer[i/2] = hex.EncodeToString(hashBytes[:])
			
			// If this node is part of our path, add the sibling to the proof
			if i == currentTxPos || i+1 == currentTxPos {
				siblingPos := i
				if currentTxPos == i {
					siblingPos = i + 1
				}
				merkleProof = append(merkleProof, currentLayer[siblingPos])
			}
		}
		
		// Update for the next iteration
		currentLayer = nextLayer
		currentTxPos = currentTxPos / 2
		nodeCount = len(nextLayer)
	}
	
	// 4. Assemble and return the proof
	proof := map[string]interface{}{
		"txid":         txid,
		"block_hash":   blockHash,
		"block_height": blockData["height"],
		"timestamp":    blockData["time"],
		"merkle_root":  blockData["merkleroot"],
		"merkle_proof": merkleProof,
		"tx_index":     txIndex,
	}
	
	return proof, nil
}

// VerifySignature implements BitcoinClient.ValidateSignature
// This provides a more robust implementation than the stub in the original
func (c *BitcoinClientImpl) ValidateSignature(
	ctx context.Context,
	message []byte,
	signature []byte,
	pubKey []byte,
) (bool, error) {
	// Convert the binary data to hex strings for Bitcoin RPC
	messageHex := hex.EncodeToString(message)
	signatureHex := hex.EncodeToString(signature)
	pubKeyHex := hex.EncodeToString(pubKey)
	
	// Call Bitcoin RPC verifymessage
	// verifymessage expects address, signature, and message
	// First, convert the public key to an address
	var address string
	err := c.call(ctx, "getaddressfrompubkey", []interface{}{pubKeyHex}, &address)
	if err != nil {
		return false, fmt.Errorf("failed to convert public key to address: %w", err)
	}
	
	// Now verify the signature
	var result bool
	err = c.call(ctx, "verifymessage", []interface{}{address, signatureHex, messageHex}, &result)
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}
	
	return result, nil
}

// GetTransactionConfirmations returns the number of confirmations for a transaction
func (c *BitcoinClientImpl) GetTransactionConfirmations(
	ctx context.Context,
	txHash string,
) (uint64, error) {
	var txData map[string]interface{}
	err := c.call(ctx, "gettransaction", []interface{}{txHash}, &txData)
	if err != nil {
		return 0, fmt.Errorf("failed to get transaction data: %w", err)
	}
	
	confirmations, ok := txData["confirmations"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid transaction data format")
	}
	
	return uint64(confirmations), nil
}

// GetNetworkFeeEstimate estimates the appropriate fee rate in satoshis per byte
func (c *BitcoinClientImpl) GetNetworkFeeEstimate(
	ctx context.Context,
	targetConfirmations int,
) (uint64, error) {
	// Bitcoin Core uses blocks for estimatefee, so convert confirmations to blocks
	targetBlocks := targetConfirmations
	if targetBlocks < 1 {
		targetBlocks = 1
	}
	
	var feeRate float64
	err := c.call(ctx, "estimatesmartfee", []interface{}{targetBlocks}, &feeRate)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate fee: %w", err)
	}
	
	// Convert from BTC/kB to satoshis/byte
	satoshisPerByte := uint64(feeRate * 100000000 / 1024)
	if satoshisPerByte < 1 {
		satoshisPerByte = 1 // Minimum of 1 satoshi per byte
	}
	
	return satoshisPerByte, nil
}

// CreateMultisigAddress creates a P2SH multisig address from public keys
func (c *BitcoinClientImpl) CreateMultisigAddress(
	ctx context.Context,
	requiredSignatures int,
	publicKeys []string,
) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.call(ctx, "createmultisig", []interface{}{requiredSignatures, publicKeys}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to create multisig address: %w", err)
	}
	
	return result, nil
}

// DecodeRawTransaction decodes a raw transaction hex string into its components
func (c *BitcoinClientImpl) DecodeRawTransaction(
	ctx context.Context,
	rawTransactionHex string,
) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.call(ctx, "decoderawtransaction", []interface{}{rawTransactionHex}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode raw transaction: %w", err)
	}
	
	return result, nil
}

// GetBlockHeader retrieves the header information for a block by its hash
func (c *BitcoinClientImpl) GetBlockHeader(
	ctx context.Context,
	blockHash string,
) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.call(ctx, "getblockheader", []interface{}{blockHash, true}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get block header: %w", err)
	}
	
	return result, nil
}

// CalculateNetworkHashPower calculates the network hash power based on difficulty and time between blocks
func (c *BitcoinClientImpl) CalculateNetworkHashPower(
	ctx context.Context,
	blockHeight uint64,
) (float64, error) {
	// Get the block hash for the specified height
	var blockHash string
	err := c.call(ctx, "getblockhash", []interface{}{blockHeight}, &blockHash)
	if err != nil {
		return 0, fmt.Errorf("failed to get block hash: %w", err)
	}
	
	// Get the block details
	var blockDetails map[string]interface{}
	err = c.call(ctx, "getblock", []interface{}{blockHash, true}, &blockDetails)
	if err != nil {
		return 0, fmt.Errorf("failed to get block details: %w", err)
	}
	
	// Extract the difficulty
	difficulty, ok := blockDetails["difficulty"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid block data format: difficulty not found")
	}
	
	// Calculate hash power using the formula: hashrate = difficulty * 2^32 / 600
	// 600 seconds is the target time between blocks
	// 2^32 is the normalization factor
	hashPower := difficulty * math.Pow(2, 32) / 600
	
	// Convert to Petahash per second (PH/s)
	petaHashPerSecond := hashPower / 1e15
	
	return petaHashPerSecond, nil
}

// SignMessage signs a message with a private key
func (c *BitcoinClientImpl) SignMessage(
	ctx context.Context,
	privateKeyWIF string,
	message string,
) (string, error) {
	var signature string
	err := c.call(ctx, "signmessage", []interface{}{privateKeyWIF, message}, &signature)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}
	
	return signature, nil
}

// ImportPrivateKey imports a private key to the wallet
func (c *BitcoinClientImpl) ImportPrivateKey(
	ctx context.Context,
	privateKeyWIF string,
	label string,
	rescan bool,
) error {
	err := c.call(ctx, "importprivkey", []interface{}{privateKeyWIF, label, rescan}, nil)
	if err != nil {
		return fmt.Errorf("failed to import private key: %w", err)
	}
	
	return nil
}

// CreateRawTransaction creates a raw transaction with the specified inputs and outputs
func (c *BitcoinClientImpl) CreateRawTransaction(
	ctx context.Context,
	inputs []map[string]interface{},
	outputs map[string]interface{},
	locktime int,
) (string, error) {
	var rawTx string
	err := c.call(ctx, "createrawtransaction", []interface{}{inputs, outputs, locktime}, &rawTx)
	if err != nil {
		return "", fmt.Errorf("failed to create raw transaction: %w", err)
	}
	
	return rawTx, nil
}

// VerifyChainTip ensures the blockchain is in a valid state and returns the current chain height
func (c *BitcoinClientImpl) VerifyChainTip(ctx context.Context) (uint64, error) {
	// First, check if the blockchain is in a valid state
	var chainInfo map[string]interface{}
	err := c.call(ctx, "getblockchaininfo", []interface{}{}, &chainInfo)
	if err != nil {
		return 0, fmt.Errorf("failed to get blockchain info: %w", err)
	}
	
	// Check if the chain is in an initial block download state
	ibd, ok := chainInfo["initialblockdownload"].(bool)
	if ok && ibd {
		return 0, fmt.Errorf("blockchain is still in initial block download mode")
	}
	
	// Get the current block count
	var blockCount uint64
	err = c.call(ctx, "getblockcount", []interface{}{}, &blockCount)
	if err != nil {
		return 0, fmt.Errorf("failed to get block count: %w", err)
	}
	
	// Verify that the chain is valid
	var verified bool
	err = c.call(ctx, "verifychain", []interface{}{}, &verified)
	if err != nil {
		return 0, fmt.Errorf("failed to verify blockchain: %w", err)
	}
	
	if !verified {
		return 0, fmt.Errorf("blockchain verification failed")
	}
	
	return blockCount, nil
}
