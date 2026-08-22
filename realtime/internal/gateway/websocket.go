// Package gateway implements the WebSocket and HTTP gateway.
// SOW Sections 19, 22, 23: Realtime delivery
// Production: Nginx proxies wss://live.predictatrade.com/ws/v1 → 127.0.0.1:8080
package gateway

import (
	"encoding/base64"
	"strings"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/predictatrade/realtime/internal/types"
)

// EventEnvelope is the versioned event envelope for WebSocket delivery.
type EventEnvelope struct {
	EventID       string          `json:"event_id"`
	StreamID      string          `json:"stream_id"`
	Sequence      int64           `json:"sequence"`
	SchemaVersion string          `json:"schema_version"`
	Timestamp     time.Time       `json:"timestamp"`
	Type          string          `json:"type"`
	Priority      string          `json:"priority"`
	Payload       json.RawMessage `json:"payload"`
	CorrelationID string          `json:"correlation_id,omitempty"`
}

// Client represents a connected WebSocket client.
type Client struct {
	ID           string
	UserID       string
	conn         *websocket.Conn
	send         chan *EventEnvelope
	sequences    map[string]*int64
	mu           sync.Mutex
	entitlements []types.StrategyID
	origin       string
}

func (c *Client) nextSeq(streamID string) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	seq, ok := c.sequences[streamID]
	if !ok {
		var s int64
		c.sequences[streamID] = &s
		seq = &s
	}
	*seq++
	return *seq
}

func (c *Client) isEntitled(strategyID types.StrategyID) bool {
	// P2-003: Fail-closed — a client with no positively-set entitlements
	// must NOT receive BUY/SELL signals. Only explicitly entitled strategies pass.
	if len(c.entitlements) == 0 {
		return false
	}
	for _, s := range c.entitlements {
		if s == strategyID {
			return true
		}
	}
	return false
}


// extractUserIDFromJWT parses a JWT token and extracts the user ID from claims.
// This is the ONLY way WebSocket client identity is established — never from query params.
func extractUserIDFromJWT(tokenString string) (string, error) {
	// Parse without verification for now — the Go engine trusts the NestJS-issued token
	// In production, this should verify the JWT signature using a shared secret.
	// For now, we extract the "sub" claim which contains the user ID.
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}

	if sub, ok := claims["sub"].(string); ok {
		return sub, nil
	}
	return "", fmt.Errorf("no sub claim in token")
}

// WebSocketHub manages WebSocket clients and broadcasts events.
type WebSocketHub struct {
	mu             sync.RWMutex
	clients        map[string]*Client
	register       chan *Client
	unregister     chan *Client
	broadcast      chan *EventEnvelope
	upgrader       websocket.Upgrader
	allowedOrigins map[string]bool
}

func NewWebSocketHub(allowedOrigins []string) *WebSocketHub {
	originsMap := make(map[string]bool)
	for _, o := range allowedOrigins {
		originsMap[o] = true
	}
	// Allow localhost for dev/testing
	originsMap["https://platform.predictatrade.com"] = true
	originsMap["https://predictatrade.com"] = true

	return &WebSocketHub{
		clients:    make(map[string]*Client),
		register:    make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *EventEnvelope, 512),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // Non-browser clients (Windows Agent) have no Origin
				}
				return originsMap[origin]
			},
		},
		allowedOrigins: originsMap,
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.send)
			}
			h.mu.Unlock()
		case event := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				if event.Priority == "P2" {
					select {
					case client.send <- event:
					default:
					}
				} else {
					select {
					case client.send <- event:
					default:
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

var wsConnCount atomic.Int64

func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// P0 SECURITY: Identity MUST come from validated token, not caller-supplied query param.
	// The old code accepted ?userId=... which allowed user impersonation.
	// Now we extract identity from the JWT token in Authorization header or query param token.
	userID := "anonymous"

	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if parsedID, err := extractUserIDFromJWT(token); err == nil && parsedID != "" {
			userID = parsedID
		}
	}

	// Fall back to token query param (for browser WebSocket which can't set headers)
	if userID == "anonymous" {
		if token := r.URL.Query().Get("token"); token != "" {
			if parsedID, err := extractUserIDFromJWT(token); err == nil && parsedID != "" {
				userID = parsedID
			}
		}
	}

	// Reject caller-supplied userId — it must NOT determine identity
	// (old ?userId= param is now ignored for identity purposes)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	clientID := uuid.New().String()

	client := &Client{
		ID:        clientID,
		UserID:    userID,
		conn:      conn,
		send:      make(chan *EventEnvelope, 256),
		sequences: make(map[string]*int64),
		origin:    r.Header.Get("Origin"),
	}

	h.register <- client
	wsConnCount.Add(1)

	// Initial snapshot
	snapshot := EventEnvelope{
		EventID:       uuid.New().String(),
		StreamID:      "market_state",
		Sequence:      client.nextSeq("market_state"),
		SchemaVersion: "1.0.0",
		Timestamp:     time.Now().UTC(),
		Type:          "SNAPSHOT",
		Priority:      "P0",
		Payload:       json.RawMessage(`{"status":"connected","client_id":"` + clientID + `"}`),
	}
	client.send <- &snapshot

	// Writer goroutine
	go func() {
		defer func() {
			h.unregister <- client
			wsConnCount.Add(-1)
			conn.Close()
		}()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case event, ok := <-client.send:
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteJSON(event); err != nil {
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Reader goroutine
	go func() {
		defer conn.Close()
		conn.SetReadLimit(4096)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *WebSocketHub) BroadcastSignal(signal *types.Signal) {
	if signal == nil {
		return
	}
	payload, err := json.Marshal(signal)
	if err != nil {
		return
	}
	priority := "P1"
	if signal.Direction != types.DirectionNoTrade {
		priority = "P0"
	}
	event := EventEnvelope{
		EventID:       uuid.New().String(),
		StreamID:      fmt.Sprintf("signals:%s", signal.StrategyID),
		SchemaVersion: "1.0.0",
		Timestamp:     time.Now().UTC(),
		Type:          "SIGNAL",
		Priority:      priority,
		Payload:       payload,
	}
	h.mu.RLock()
	for _, client := range h.clients {
		if !client.isEntitled(signal.StrategyID) {
			continue
		}
		event.Sequence = client.nextSeq(event.StreamID)
		eventCopy := event
		select {
		case client.send <- &eventCopy:
		default:
		}
	}
	h.mu.RUnlock()
}

func (h *WebSocketHub) BroadcastMarketState(state interface{}) {
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	event := EventEnvelope{
		EventID:       uuid.New().String(),
		StreamID:      "market_state",
		SchemaVersion: "1.0.0",
		Timestamp:     time.Now().UTC(),
		Type:          "MARKET_STATE",
		Priority:      "P1",
		Payload:       payload,
	}
	h.mu.RLock()
	for _, client := range h.clients {
		event.Sequence = client.nextSeq("market_state")
		eventCopy := event
		select {
		case client.send <- &eventCopy:
		default:
		}
	}
	h.mu.RUnlock()
}

func (h *WebSocketHub) ClientCount() int64 { return wsConnCount.Load() }

// BroadcastMarketSnapshot broadcasts a Master Node market snapshot to all connected clients.
// This includes comprehensive data: multi-TF bars, indicators, account info, symbol info, session.
func (h *WebSocketHub) BroadcastMarketSnapshot(snapshot interface{}) {
	if snapshot == nil {
		return
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	event := EventEnvelope{
		EventID:       uuid.New().String(),
		StreamID:     "market_snapshot",
		SchemaVersion: "1.0.0",
		Timestamp:    time.Now().UTC(),
		Type:         "MARKET_SNAPSHOT",
		Priority:     "P1",
		Payload:      payload,
	}
	h.mu.RLock()
	for _, client := range h.clients {
		event.Sequence = client.nextSeq("market_snapshot")
		eventCopy := event
		select {
		case client.send <- &eventCopy:
		default:
		}
	}
	h.mu.RUnlock()
}

// AgentStatus represents the connection status of Windows Agents and Master Nodes.
type AgentStatus struct {
	AgentsConnected      int   `json:"agents_connected"`
	MasterNodeConnected  bool  `json:"master_node_connected"`
	SnapshotCount        uint64 `json:"snapshot_count"`
	LastSnapshotTime     *time.Time `json:"last_snapshot_time,omitempty"`
	Timestamp            time.Time `json:"timestamp"`
}

// BroadcastAgentStatus broadcasts agent connection status to all connected clients.
func (h *WebSocketHub) BroadcastAgentStatus(status AgentStatus) {
	payload, err := json.Marshal(status)
	if err != nil {
		return
	}
	event := EventEnvelope{
		EventID:       uuid.New().String(),
		StreamID:     "agent_status",
		SchemaVersion: "1.0.0",
		Timestamp:    time.Now().UTC(),
		Type:         "AGENT_STATUS",
		Priority:     "P2",
		Payload:      payload,
	}
	h.mu.RLock()
	for _, client := range h.clients {
		event.Sequence = client.nextSeq("agent_status")
		eventCopy := event
		select {
		case client.send <- &eventCopy:
		default:
		}
	}
	h.mu.RUnlock()
}
