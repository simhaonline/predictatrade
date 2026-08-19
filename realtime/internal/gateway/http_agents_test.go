package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleAgentsStatusLivePrecedence verifies that the /api/v1/agents/status
// endpoint always returns the LIVE agent count from agentHub, not a stale
// Valkey cache entry. This is a regression test for the screenshot bug where
// the dashboard showed "Master Node OFFLINE" while the MT5 terminal showed
// "Agent: CONNECTED" and /health reported agents:1.
func TestHandleAgentsStatusLivePrecedence(t *testing.T) {
	// Build a minimal HTTPServer with a live agentHub (1 agent registered)
	// and a nil valkeyCache to force the live path.
	agentHub := NewAgentHub(nil)
	// Manually register a fake agent to make AgentCount() == 1
	agentHub.mu.Lock()
	agentHub.agents["test-agent-1"] = &AgentConnection{
		ID:   "test-agent-1",
		send: make(chan []byte, 8),
	}
	agentHub.mu.Unlock()

	server := &HTTPServer{
		hub:         NewWebSocketHub([]string{"*"}),
		agentHub:    agentHub,
		valkeyCache: nil, // no cache — must use live data
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/status", nil)
	w := httptest.NewRecorder()
	server.handleAgentsStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	agentsConnected, ok := resp["agents_connected"].(float64)
	if !ok {
		t.Fatalf("agents_connected missing or wrong type: %v", resp["agents_connected"])
	}
	if agentsConnected != 1 {
		t.Errorf("expected agents_connected=1 (live), got %v", agentsConnected)
	}

	// When agentHub has >0 connections, master_node_connected must be true
	// even if the provider hasn't registered yet (race during handshake).
	masterConnected, ok := resp["master_node_connected"].(bool)
	if !ok {
		t.Fatalf("master_node_connected missing or wrong type: %v", resp["master_node_connected"])
	}
	if !masterConnected {
		t.Error("expected master_node_connected=true when agents_connected > 0")
	}
}

// TestHandleAgentsStatusZeroAgents verifies correct output when no agents are connected.
func TestHandleAgentsStatusZeroAgents(t *testing.T) {
	agentHub := NewAgentHub(nil)

	server := &HTTPServer{
		hub:         NewWebSocketHub([]string{"*"}),
		agentHub:    agentHub,
		valkeyCache: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/status", nil)
	w := httptest.NewRecorder()
	server.handleAgentsStatus(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	agentsConnected, _ := resp["agents_connected"].(float64)
	if agentsConnected != 0 {
		t.Errorf("expected agents_connected=0, got %v", agentsConnected)
	}

	masterConnected, _ := resp["master_node_connected"].(bool)
	if masterConnected {
		t.Error("expected master_node_connected=false when no agents connected")
	}
}
