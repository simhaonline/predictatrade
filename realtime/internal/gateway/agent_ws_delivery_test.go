package gateway

import (
	"encoding/json"
	"testing"
	"time"
)

// newTestAgent builds an AgentConnection with a buffered send channel so tests
// can read delivered payloads without a running writePump.
func newTestAgent(id string) *AgentConnection {
	return &AgentConnection{
		ID:   id,
		send: make(chan []byte, 16),
		done: make(chan struct{}),
	}
}

// TestSendFilteredSignalToAgentsDelivery verifies the core delivery pipeline: a
// connected agent that is entitled and passes the per-client risk check
// actually receives the serialized signal payload on its send channel.
func TestSendFilteredSignalToAgentsDelivery(t *testing.T) {
	hub := NewAgentHub(nil)
	hub.SetStrategyFilter(func(agentID, strategyID string) bool { return true })
	hub.SetRiskCheck(func(agentID string) bool { return true })

	a := newTestAgent("agent-A")
	hub.mu.Lock()
	hub.agents["agent-A"] = a
	hub.mu.Unlock()

	payload := []byte(`{"type":"SIGNAL","direction":"SELL"}`)
	hub.SendFilteredSignalToAgents("evt-1", "signals:ULTRA", "SIGNAL", "P0", "1.0.0", payload, "ULTRA_SCALPING")

	select {
	case got := <-a.send:
		var env map[string]json.RawMessage
		if err := json.Unmarshal(got, &env); err != nil {
			t.Fatalf("delivered payload not a valid envelope: %v", err)
		}
		if string(env["type"]) != `"SIGNAL"` {
			t.Fatalf("unexpected envelope type: %s", env["type"])
		}
	case <-time.After(time.Second):
		t.Fatal("agent did not receive delivered signal")
	}
}

// TestSendFilteredSignalToAgentsStrategySkip verifies the server-side
// entitlement filter drops a signal the agent's plan does not include.
func TestSendFilteredSignalToAgentsStrategySkip(t *testing.T) {
	hub := NewAgentHub(nil)
	hub.SetStrategyFilter(func(agentID, strategyID string) bool {
		return strategyID == "STANDARD_SCALPING" // agent only allowed STANDARD_SCALPING
	})
	hub.SetRiskCheck(func(agentID string) bool { return true })

	a := newTestAgent("agent-B")
	hub.mu.Lock()
	hub.agents["agent-B"] = a
	hub.mu.Unlock()

	// ULTRA_SCALPING signal -> should be skipped for this agent.
	hub.SendFilteredSignalToAgents("evt-2", "signals:ULTRA", "SIGNAL", "P0", "1.0.0", []byte(`{}`), "ULTRA_SCALPING")

	select {
	case <-a.send:
		t.Fatal("agent received a signal its plan does not entitle")
	case <-time.After(100 * time.Millisecond):
		// expected: no delivery
	}
}

// TestSendFilteredSignalToAgentsRiskSkip verifies the per-client risk filter
// isolates a blown client: a connected agent that fails the risk check does NOT
// receive executable signals, while a healthy agent on the same hub still does.
func TestSendFilteredSignalToAgentsRiskSkip(t *testing.T) {
	hub := NewAgentHub(nil)
	hub.SetStrategyFilter(func(agentID, strategyID string) bool { return true })
	// Risk check fails only for the blown agent.
	hub.SetRiskCheck(func(agentID string) bool { return agentID != "blown" })

	blown := newTestAgent("blown")
	healthy := newTestAgent("healthy")
	hub.mu.Lock()
	hub.agents["blown"] = blown
	hub.agents["healthy"] = healthy
	hub.mu.Unlock()

	hub.SendFilteredSignalToAgents("evt-3", "signals:ULTRA", "SIGNAL", "P0", "1.0.0", []byte(`{"type":"SIGNAL"}`), "ULTRA_SCALPING")

	select {
	case <-blown.send:
		t.Fatal("blown agent must NOT receive executable signal")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-healthy.send:
		// expected: healthy agent receives it
	case <-time.After(time.Second):
		t.Fatal("healthy agent did not receive signal despite passing risk check")
	}
}
