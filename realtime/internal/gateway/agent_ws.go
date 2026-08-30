package gateway

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/predictatrade/realtime/pkg/bus"
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
	ID       string
	conn     *websocket.Conn
	send     chan []byte
	done     chan struct{} // signals both read and write goroutines to stop
	doneOnce sync.Once     // guards close of done against double-close panics
}

// closeDone safely closes the done channel exactly once, even if both the
// Run() defer and the read goroutine defer (or DisconnectAgent) attempt it.
func (ac *AgentConnection) closeDone() {
	ac.doneOnce.Do(func() { close(ac.done) })
}

type AgentHub struct {
	mu         sync.RWMutex
	agents     map[string]*AgentConnection
	register   chan *AgentConnection
	unregister chan *AgentConnection
	provider   AgentDataProvider
	upgrader   websocket.Upgrader
	// Per-agent terminal link state, derived from agent heartbeats so the
	// status endpoint can report live MT4/MT5 connection liveness.
	mt4States map[string]bool
	mt5States map[string]bool
	// Strategy entitlement filter — set by main.go to enable per-agent
	// strategy filtering based on license allowed_strategies.
	// Returns true if the agent is allowed to receive signals for the given strategy.
	strategyFilter func(agentID, strategyID string) bool

	// Risk check — set by main.go to enforce PER-CLIENT risk isolation at
	// signal delivery. It reads the receiving agent's own account state so one
	// client's blown/over-exposed account can never contaminate another's
	// executable signals. Fail-open: nil check (or a nil-returning func) means
	// all clients pass.
	riskCheckFn func(agentID string) bool

	// Ingest bus — decouples data-collection (inbound agent messages) from the
	// signal engine. Default is a DirectBus (in-process, same as before). When
	// NATS is configured, the publisher enqueues and a subscriber dispatches.
	ingestBus bus.IngestBus
	ingestSub bus.IngestSubscriber

	// Live connection-state callbacks — set by main.go so the engine can
	// publish authoritative agent connect/disconnect/terminal-link state into
	// the control-plane DB (licensing.devices / device_activations) that the
	// Admin + User dashboards read.
	onConnect    func(agentID string)
	onDisconnect func(agentID string)
	onTerminals  func(agentID string, mt4, mt5 bool)
}

func NewAgentHub(provider AgentDataProvider) *AgentHub {
	return &AgentHub{
		agents:     make(map[string]*AgentConnection),
		register:   make(chan *AgentConnection),
		unregister: make(chan *AgentConnection),
		provider:   provider,
		mt4States:  make(map[string]bool),
		mt5States:  make(map[string]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

// updateAgentTerminals records the latest MT4/MT5 link state reported by an
// agent heartbeat.
func (h *AgentHub) updateAgentTerminals(agentID string, mt4, mt5 bool) {
	h.mu.Lock()
	h.mt4States[agentID] = mt4
	h.mt5States[agentID] = mt5
	h.mu.Unlock()
	if h.onTerminals != nil {
		h.onTerminals(agentID, mt4, mt5)
	}
}

// MT4ConnectedCount returns the number of connected agents that currently
// report a live MT4 terminal link.
func (h *AgentHub) MT4ConnectedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, v := range h.mt4States {
		if v {
			n++
		}
	}
	return n
}

// MT5ConnectedCount returns the number of connected agents that currently
// report a live MT5 terminal link.
func (h *AgentHub) MT5ConnectedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, v := range h.mt5States {
		if v {
			n++
		}
	}
	return n
}

// SetStrategyFilter sets the per-agent strategy entitlement filter.
// When set, SendFilteredSignalToAgents will only send signals to agents
// whose license allows the given strategy.
func (h *AgentHub) SetStrategyFilter(filter func(agentID, strategyID string) bool) {
	h.strategyFilter = filter
}

// SetRiskCheck sets the per-client risk filter applied at signal delivery.
// It must read the receiving agent's OWN account/risk state and return true
// if that client may receive EXECUTABLE signals. It is intentionally
// fail-open: an unset check (or a nil receiver) allows all clients, so this
// can never introduce a global signal blackout.
func (h *AgentHub) SetRiskCheck(fn func(agentID string) bool) {
	h.riskCheckFn = fn
}

// SetIngestBus configures the inbound message transport. Passing a DirectBus
// preserves the original in-process behavior; passing a NatsBus decouples
// ingestion from processing.
func (h *AgentHub) SetIngestBus(b bus.IngestBus) {
	h.ingestBus = b
}

// SetIngestSubscriber configures the consumer side (NATS only). When set, Run
// starts a dispatcher that feeds inbound messages to the engine handler.
func (h *AgentHub) SetIngestSubscriber(s bus.IngestSubscriber) {
	h.ingestSub = s
}

// SetOnConnect registers a callback fired when an agent connection is registered.
func (h *AgentHub) SetOnConnect(fn func(agentID string)) { h.onConnect = fn }

// SetOnDisconnect registers a callback fired when an agent connection is unregistered.
func (h *AgentHub) SetOnDisconnect(fn func(agentID string)) { h.onDisconnect = fn }

// SetOnTerminals registers a callback fired when an agent reports its MT4/MT5 link state.
func (h *AgentHub) SetOnTerminals(fn func(agentID string, mt4, mt5 bool)) { h.onTerminals = fn }

func (h *AgentHub) Run() {
	// If a NATS (or other external) subscriber is configured, start the
	// dispatcher that feeds inbound messages to the engine handler. The
	// DirectBus path needs no subscriber (Publish dispatches in-process).
	if h.ingestSub != nil {
		go func() {
			_ = h.ingestSub.Subscribe(h.provider.HandleAgentMessage)
		}()
	}
	for {
		select {
		case agent := <-h.register:
			h.mu.Lock()
			// Close any existing connection with the same agent ID
			// to prevent duplicate connections and goroutine leaks
			if existing, ok := h.agents[agent.ID]; ok {
				existing.closeDone()
				existing.conn.Close()
			}
			// Enforce max connection limit
			if len(h.agents) >= maxAgentConnections {
				h.mu.Unlock()
				agent.closeDone()
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
			delete(h.mt4States, agent.ID)
			delete(h.mt5States, agent.ID)
			h.mu.Unlock()
			if h.onDisconnect != nil {
				h.onDisconnect(agent.ID)
			}
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

func (h *AgentHub) GetAgentIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	return ids
}

// DisconnectAgent forcibly removes an agent connection (e.g. failed license
// validation). Safe to call for unknown IDs.
func (h *AgentHub) DisconnectAgent(agentID, reason string) {
	h.mu.Lock()
	agent, ok := h.agents[agentID]
	if ok {
		agent.closeDone()
		agent.conn.Close()
		delete(h.agents, agent.ID)
	}
	h.mu.Unlock()
	if !ok {
		return
	}
	log.Printf("[AGENT] Disconnected %s: %s", agentID, reason)
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

// SendFilteredSignalToAgents sends a signal only to agents whose license
// allows the given strategy. This is server-side entitlement enforcement —
// unauthorized agents never receive signals for strategies they haven't paid for.
func (h *AgentHub) SendFilteredSignalToAgents(eventID, streamID, eventType, priority, schemaVersion string, payload []byte, strategyID string) {
	if len(payload) == 0 {
		return
	}

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
	sent := 0
	skipped := 0
	for agentID, agent := range h.agents {
		// Check if this agent's license allows this strategy
		allowed := true
		if h.strategyFilter != nil {
			allowed = h.strategyFilter(agentID, strategyID)
		}
		if !allowed {
			skipped++
			continue
		}
		// Per-client risk isolation: a client whose own account has no buying
		// power must not receive executable signals. This is evaluated on the
		// RECEIVING agent's own account state and is fail-open, so one client's
		// blown/over-exposed account can never contaminate another client's
		// signals or cause a global blackout.
		if h.riskCheckFn != nil && !h.riskCheckFn(agentID) {
			skipped++
			continue
		}
		select {
		case agent.send <- data:
			sent++
		default:
			// Agent buffer full — drop
		}
	}
	h.mu.RUnlock()

	if skipped > 0 {
		log.Printf("[SIGNAL-FILTER] strategy=%s sent=%d skipped=%d (plan not entitled)", strategyID, sent, skipped)
	}
}

func (h *AgentHub) HandleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	// P0-RT1 (partial mitigation): shared-token authentication.
	// When AGENT_WS_TOKEN is configured, agents MUST present it via the
	// X-Agent-Token header or ?token= query param. Full per-device crypto
	// identity remains open work (see full-audit.md Batch 2).
	if expected := os.Getenv("AGENT_WS_TOKEN"); expected != "" {
		provided := r.Header.Get("X-Agent-Token")
		if provided == "" {
			provided = r.URL.Query().Get("token")
		}
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			http.Error(w, "agent authentication required", http.StatusUnauthorized)
			return
		}
	}

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
			// Signal done to stop the write goroutine (guarded against
			// double-close by closeDone's sync.Once)
			agent.closeDone()
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
			// Track live MT4/MT5 terminal link state from agent heartbeats.
			// BUG FIX: Only update terminal state from actual heartbeat messages.
			// Previously, every incoming message (TICK, MARKET_SNAPSHOT, etc.) was
			// unmarshaled into a fresh hb struct with zero-value false fields, then
			// updateAgentTerminals was called — overwriting the true values set by
			// the real heartbeat. Since the Master Node sends snapshots every ~1s
			// but heartbeats only every 30s, the terminal status was reset to false
			// ~30 times between heartbeats, making the dashboard always show
			// mt4_connected: 0, mt5_connected: 0 even when terminals were connected.
			var typeCheck struct {
				Type         string `json:"type"`
				AgentID      string `json:"agent_id"`
				MT4Connected bool   `json:"mt4_connected"`
				MT5Connected bool   `json:"mt5_connected"`
			}
			if json.Unmarshal(data, &typeCheck) == nil {
				// Heartbeat messages have an "agent_id" field but NO "type" field.
				// TICK and MARKET_SNAPSHOT messages have "type" but no "agent_id".
				// Only update terminal state from heartbeat messages to avoid
				// zero-value overwrites from tick/snapshot messages.
				if typeCheck.AgentID != "" {
					log.Printf("[AGENT-WS] Heartbeat from %s: mt4=%v mt5=%v", agentID, typeCheck.MT4Connected, typeCheck.MT5Connected)
					h.updateAgentTerminals(agentID, typeCheck.MT4Connected, typeCheck.MT5Connected)
				}
			}
			// Route inbound agent message through the ingest bus. Default is the
			// in-process DirectBus (behavior identical to the previous direct call);
			// when NATS is configured the message is enqueued and dispatched by the
			// subscriber, decoupling data-collection from the signal engine.
			if h.ingestBus != nil {
				if err := h.ingestBus.Publish(agentID, data); err != nil {
					log.Printf("[AGENT-WS] ingest publish error: %v", err)
				}
			} else if h.provider != nil {
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
		log.Printf("[SendToAgent] marshal error: agent=%s type=%s err=%v", agentID, msgType, err)
		return
	}
	h.mu.RLock()
	agent, ok := h.agents[agentID]
	count := len(h.agents)
	h.mu.RUnlock()
	if !ok {
		log.Printf("[SendToAgent] agent NOT FOUND: agent=%s type=%s (total agents=%d)", agentID, msgType, count)
		return
	}
	select {
	case agent.send <- data:
		log.Printf("[SendToAgent] SENT: agent=%s type=%s", agentID, msgType)
	default:
		log.Printf("[SendToAgent] DROPPED (buffer full): agent=%s type=%s", agentID, msgType)
	}
}

// BroadcastToAllAgents sends a JSON message to ALL connected agents.
// Used for EMERGENCY_STOP and KILL_SWITCH commands.
func (h *AgentHub) BroadcastToAllAgents(msgType string, payload interface{}) {
	data, err := json.Marshal(map[string]interface{}{
		"type":      msgType,
		"timestamp": time.Now().UTC(),
		"payload":   payload,
	})
	if err != nil {
		return
	}
	h.mu.RLock()
	sent := 0
	for _, agent := range h.agents {
		select {
		case agent.send <- data:
			sent++
		default:
		}
	}
	h.mu.RUnlock()
	log.Printf("[AGENT-WS] Broadcast %s to %d agents", msgType, sent)
}
