package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hashperp/hashperp"
)

// Server represents the API server
type Server struct {
	router     *mux.Router
	service    hashperp.HashPerpService
	upgrader   websocket.Upgrader
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(service hashperp.HashPerpService) *Server {
	router := mux.NewRouter()
	
	server := &Server{
		router:  router,
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				return true
			},
		},
	}
	
	// Set up routes
	server.setupRoutes()
	
	return server
}

// setupRoutes configures the API endpoints
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealthCheck).Methods(http.MethodGet)
	
	// JSONRPC endpoint
	s.router.HandleFunc("/rpc", s.handleRPC).Methods(http.MethodPost)
	
	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.handleWebSocket)
}

// Start starts the API server
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	
	log.Printf("Starting HashPerp API server on %s", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// handleHealthCheck handles health check requests
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	err := s.service.Healthcheck(r.Context())
	if err != nil {
		http.Error(w, "Service Unhealthy", http.StatusServiceUnavailable)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// handleRPC handles JSON-RPC requests
func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, &RPCError{
			Code:    -32700,
			Message: "Parse error",
			Data:    err.Error(),
		}, nil)
		return
	}
	
	// Ensure JSON-RPC version is 2.0
	if req.JSONRPC != "2.0" {
		writeRPCError(w, &RPCError{
			Code:    -32600,
			Message: "Invalid Request",
			Data:    "JSONRPC version must be 2.0",
		}, req.ID)
		return
	}
	
	result, err := s.executeRPCMethod(r.Context(), req.Method, req.Params)
	if err != nil {
		var rpcError *RPCError
		if errors.As(err, &rpcError) {
			writeRPCError(w, rpcError, req.ID)
		} else {
			writeRPCError(w, &RPCError{
				Code:    -32603,
				Message: "Internal error",
				Data:    err.Error(),
			}, req.ID)
		}
		return
	}
	
	writeRPCResult(w, result, req.ID)
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()
	
	// Create a context that will be canceled when the connection closes
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	
	// Handle incoming messages
	go s.handleWebSocketMessages(ctx, conn)
	
	// Keep the connection alive with pings
	go s.keepWebSocketAlive(ctx, conn)
	
	// Block until context is canceled (connection closed)
	<-ctx.Done()
}

// handleWebSocketMessages handles incoming WebSocket messages
func (s *Server) handleWebSocketMessages(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg WebSocketMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("Error reading WebSocket message: %v", err)
				return
			}
			
			if msg.Type == "rpc" {
				var rpcReq RPCRequest
				if err := json.Unmarshal(msg.Payload, &rpcReq); err != nil {
					s.sendWebSocketError(conn, "rpc_error", &RPCError{
						Code:    -32700,
						Message: "Parse error",
						Data:    err.Error(),
					}, rpcReq.ID)
					continue
				}
				
				result, err := s.executeRPCMethod(ctx, rpcReq.Method, rpcReq.Params)
				if err != nil {
					var rpcError *RPCError
					if errors.As(err, &rpcError) {
						s.sendWebSocketError(conn, "rpc_error", rpcError, rpcReq.ID)
					} else {
						s.sendWebSocketError(conn, "rpc_error", &RPCError{
							Code:    -32603,
							Message: "Internal error",
							Data:    err.Error(),
						}, rpcReq.ID)
					}
					continue
				}
				
				s.sendWebSocketResult(conn, "rpc_result", result, rpcReq.ID)
			} else if msg.Type == "subscribe" {
				// Handle subscription requests
				s.handleSubscription(ctx, conn, msg.Payload)
			}
		}
	}
}

// keepWebSocketAlive sends periodic pings to keep the WebSocket connection alive
func (s *Server) keepWebSocketAlive(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Printf("Error sending ping: %v", err)
				return
			}
		}
	}
}

// handleSubscription handles WebSocket subscription requests
func (s *Server) handleSubscription(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	var subscription struct {
		Channel string          `json:"channel"`
		Params  json.RawMessage `json:"params"`
	}
	
	if err := json.Unmarshal(payload, &subscription); err != nil {
		s.sendWebSocketError(conn, "subscription_error", &RPCError{
			Code:    -32700,
			Message: "Parse error",
			Data:    err.Error(),
		}, nil)
		return
	}
	
	// Create a context that will be canceled when the parent context is canceled
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	// Start a goroutine to handle the subscription
	go func() {
		switch subscription.Channel {
		case "contracts":
			s.subscribeToContracts(subCtx, conn, subscription.Params)
		case "orders":
			s.subscribeToOrders(subCtx, conn, subscription.Params)
		case "hash_rate":
			s.subscribeToHashRate(subCtx, conn, subscription.Params)
		default:
			s.sendWebSocketError(conn, "subscription_error", &RPCError{
				Code:    -32601,
				Message: "Unknown subscription channel",
				Data:    subscription.Channel,
			}, nil)
		}
	}()
}

// subscribeToContracts subscribes to contract updates
func (s *Server) subscribeToContracts(ctx context.Context, conn *websocket.Conn, params json.RawMessage) {
	// Parse subscription parameters
	var contractParams struct {
		UserID string `json:"user_id"`
	}
	
	if err := json.Unmarshal(params, &contractParams); err != nil {
		s.sendWebSocketError(conn, "subscription_error", &RPCError{
			Code:    -32700,
			Message: "Parse error",
			Data:    err.Error(),
		}, nil)
		return
	}
	
	// Send confirmation
	s.sendWebSocketMessage(conn, "subscription_started", map[string]string{
		"channel": "contracts",
		"user_id": contractParams.UserID,
	})
	
	// Implement a polling mechanism to check for contract updates
	// In a real implementation, this might use a messaging system or database triggers
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	var lastKnownContracts []*hashperp.Contract
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Fetch current contracts for the user
			contracts, err := s.service.GetContractsByUser(ctx, contractParams.UserID, nil)
			if err != nil {
				log.Printf("Error fetching contracts: %v", err)
				continue
			}
			
			// Check for changes
			if hasContractChanges(lastKnownContracts, contracts) {
				s.sendWebSocketMessage(conn, "contracts_update", contracts)
				lastKnownContracts = contracts
			}
		}
	}
}

// subscribeToOrders subscribes to order updates
func (s *Server) subscribeToOrders(ctx context.Context, conn *websocket.Conn, params json.RawMessage) {
	// Implementation similar to subscribeToContracts
	// Omitted for brevity
}

// subscribeToHashRate subscribes to hash rate updates
func (s *Server) subscribeToHashRate(ctx context.Context, conn *websocket.Conn, params json.RawMessage) {
	// Implementation similar to subscribeToContracts
	// Omitted for brevity
}

// sendWebSocketMessage sends a WebSocket message
func (s *Server) sendWebSocketMessage(conn *websocket.Conn, msgType string, payload interface{}) {
	msg := WebSocketMessage{
		Type:    msgType,
		Payload: nil,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling WebSocket payload: %v", err)
		return
	}
	msg.Payload = payloadBytes
	
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Error writing WebSocket message: %v", err)
	}
}

// sendWebSocketResult sends a WebSocket RPC result
func (s *Server) sendWebSocketResult(conn *websocket.Conn, msgType string, result interface{}, id interface{}) {
	rpcResponse := RPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	
	s.sendWebSocketMessage(conn, msgType, rpcResponse)
}

// sendWebSocketError sends a WebSocket RPC error
func (s *Server) sendWebSocketError(conn *websocket.Conn, msgType string, err *RPCError, id interface{}) {
	rpcResponse := RPCResponse{
		JSONRPC: "2.0",
		Error:   err,
		ID:      id,
	}
	
	s.sendWebSocketMessage(conn, msgType, rpcResponse)
}

// writeRPCResult writes a JSON-RPC result response
func writeRPCResult(w http.ResponseWriter, result interface{}, id interface{}) {
	response := RPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeRPCError writes a JSON-RPC error response
func writeRPCError(w http.ResponseWriter, err *RPCError, id interface{}) {
	response := RPCResponse{
		JSONRPC: "2.0",
		Error:   err,
		ID:      id,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Always 200 for JSON-RPC errors
	json.NewEncoder(w).Encode(response)
}

// hasContractChanges checks if there are changes between two contract lists
func hasContractChanges(oldContracts, newContracts []*hashperp.Contract) bool {
	if len(oldContracts) != len(newContracts) {
		return true
	}
	
	// Create a map of old contracts by ID
	oldContractMap := make(map[string]*hashperp.Contract)
	for _, contract := range oldContracts {
		oldContractMap[contract.ID] = contract
	}
	
	// Check for changes
	for _, newContract := range newContracts {
		oldContract, exists := oldContractMap[newContract.ID]
		if !exists {
			return true
		}
		
		// Check for status changes
		if oldContract.Status != newContract.Status {
			return true
		}
		
		// Check for other relevant changes
		// This is a simplified check, in a real implementation
		// we would check all relevant fields
	}
	
	return false
}