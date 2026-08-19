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

type AgentConnection struct {
	ID   string
	conn *websocket.Conn
	send chan []byte
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
			h.agents[agent.ID] = agent
			h.mu.Unlock()
		case agent := <-h.unregister:
			h.mu.Lock()
			delete(h.agents, agent.ID)
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

func (h *AgentHub) HandleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	agentID := r.URL.Query().Get("agentId")
	if agentID == "" {
		agentID = uuid.New().String()
	}
	agent := &AgentConnection{ID: agentID, conn: conn, send: make(chan []byte, 64)}
	h.register <- agent

	confirm, _ := json.Marshal(map[string]interface{}{
		"type": "CONNECTED", "agentId": agentID, "timestamp": time.Now().UTC(),
	})
	conn.WriteMessage(websocket.TextMessage, confirm)

	go func() {
		defer conn.Close()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-agent.send:
				if !ok { return }
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil { return }
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil { return }
			}
		}
	}()

	go func() {
		defer func() {
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
			_, data, err := conn.ReadMessage()
			if err != nil { break }
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
