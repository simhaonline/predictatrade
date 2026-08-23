package gateway

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// AgentDataProvider is implemented by marketdata.AgentProvider.
type AgentDataProvider interface {
	HandleAgentMessage(agentID string, data []byte)
	UnregisterAgent(agentID string)
}

// maxAgentConnections limits the number of concurrent agent WebSocket
// connections to prevent goroutine leaks from misconfigured clients
// that open many connections without closing old ones.
const maxAgentConnections = 100

type AgentConnection struct {
	ID   string
	conn *websocket.Conn
	send chan []byte
	done chan struct{} // signals both read and write goroutines to stop
}

type AgentHub struct {
	mu         sync.RWMutex
	agents     map[string]*AgentConnection
	register   chan *AgentConnection
	unregister chan *AgentConnection
	provider   AgentDataProvider
	upgrader   websocket.Upgrader
}

func NewAgentHub(provider AgentDataProvider) *AgentHub {
	return &AgentHub{
		agents:     make(map[string]*AgentConnection),
		register:   make(chan *AgentConnection),
		unregister: make(chan *AgentConnection),
		provider:   provider,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (h *AgentHub) Run() {
	for {
		select {
		case agent := <-h.register:
			h.mu.Lock()
			// Close any existing connection with the same agent ID
			// to prevent duplicate connections and goroutine leaks
			if existing, ok := h.agents[agent.ID]; ok {
				close(existing.done)
				existing.conn.Close()
			}
			// Enforce max connection limit
			if len(h.agents) >= maxAgentConnections {
				h.mu.Unlock()
				close(agent.done)
				agent.conn.Close()
				continue
			}
			h.agents[agent.ID] = agent
			h.mu.Unlock()
		case agent := <-h.unregister:
			h.mu.Lock()
			// Only delete if this is still the current agent (not replaced)
			if current, ok := h.agents[agent.ID]; ok && current == agent {
				delete(h.agents, agent.ID)
			}
			h.mu.Unlock()
			if h.provider != nil {
				h.provider.UnregisterAgent(agent.ID)
			}
		}
	}
}

func (h *AgentHub) AgentCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.agents)
}

// BroadcastSignalToAgents sends a trading signal to ALL connected Windows Agents.
// The agent forwards it to the MT4/MT5 EA via named pipe.
// Only directional signals (BUY/SELL/BUY_CANDIDATE/SELL_CANDIDATE) are sent —
// NO-TRADE signals are not forwarded to reduce noise.
// The signal is wrapped in the same EventEnvelope format used by the WebSocketHub.
func (h *AgentHub) BroadcastSignalToAgents(eventID, streamID, eventType, priority, schemaVersion string, payload []byte) {
	if len(payload) == 0 {
		return
	}

	// Build the event envelope (same format the Windows Agent expects)
	envelope := map[string]interface{}{
		"event_id":       eventID,
		"stream_id":      streamID,
		"schema_version": schemaVersion,
		"timestamp":      time.Now().UTC(),
		"type":           eventType,
		"priority":       priority,
		"payload":        json.RawMessage(payload),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return
	}

	h.mu.RLock()
	for _, agent := range h.agents {
		select {
		case agent.send <- data:
		default:
			// Agent buffer full — drop to avoid blocking signal processing
		}
	}
	h.mu.RUnlock()
}

func (h *AgentHub) HandleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	agentID := r.URL.Query().Get("agentId")
	if agentID == "" {
		agentID = uuid.New().String()
	}
	agent := &AgentConnection{
		ID:   agentID,
		conn: conn,
		send: make(chan []byte, 64),
		done: make(chan struct{}),
	}
	h.register <- agent

	confirm, _ := json.Marshal(map[string]interface{}{
		"type": "CONNECTED", "agentId": agentID, "timestamp": time.Now().UTC(),
	})
	conn.WriteMessage(websocket.TextMessage, confirm)

	// Write goroutine — sends signals and pings; exits on done or write error
	go func() {
		defer conn.Close()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-agent.done:
				return
			case msg, ok := <-agent.send:
				if !ok {
					return
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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

	// Read goroutine — reads agent messages; exits on done, read error, or deadline
	go func() {
		defer func() {
			// Signal done to stop the write goroutine
			select {
			case <-agent.done:
				// already closed
			default:
				close(agent.done)
			}
			h.unregister <- agent
			conn.Close()
		}()
		conn.SetReadLimit(65536)
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			return nil
		})
		for {
			select {
			case <-agent.done:
				return
			default:
			}
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if h.provider != nil {
				h.provider.HandleAgentMessage(agentID, data)
			}
			ack, _ := json.Marshal(map[string]interface{}{
				"type": "ACK", "agentId": agentID, "timestamp": time.Now().UTC(),
			})
			select {
			case agent.send <- ack:
			default:
			}
		}
	}()
}

// SendToAgent sends a JSON message to a specific agent by ID.
func (h *AgentHub) SendToAgent(agentID string, msgType string, payload interface{}) {
	data, err := json.Marshal(map[string]interface{}{
		"type":      msgType,
		"timestamp": time.Now().UTC(),
		"payload":   payload,
	})
	if err != nil {
		return
	}
	h.mu.RLock()
	agent, ok := h.agents[agentID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case agent.send <- data:
	default:
	}
}
