package marketdata

import (
	"testing"
	"time"
)

// TestAgentAccountOKIsolation verifies the per-client risk isolation primitive:
//   - an unknown client is allowed (fail-open, preserves default behavior);
//   - a client with real buying power is allowed;
//   - a client with NO free margin (blown account) is isolated (rejected);
//   - a stale client record is allowed (fail-open on stale data).
//
// This guarantees one client's blown account can never block another's signals.
func TestAgentAccountOKIsolation(t *testing.T) {
	p := NewAgentProvider()
	defer close(p.snapshotStop)

	// Unknown agent -> fail-open (allowed).
	if p.AgentAccountOK("unknown") {
		t.Fatalf("unknown agent must fail closed")
	}

	// Healthy client with free margin -> allowed.
	p.RecordAgentAccount("healthy", &SnapshotAccount{Balance: 1000, Equity: 1000, FreeMargin: 500, Leverage: 100}, &SnapshotPositions{TotalPositions: 2})
	if !p.AgentAccountOK("healthy") {
		t.Fatalf("healthy agent with free margin should be allowed")
	}

	// Blown client (no free margin) -> isolated.
	p.RecordAgentAccount("blown", &SnapshotAccount{Balance: 0, Equity: 0, FreeMargin: 0, Leverage: 100}, &SnapshotPositions{TotalPositions: 4})
	if p.AgentAccountOK("blown") {
		t.Fatalf("blown agent with no free margin must be isolated (rejected)")
	}

	// Different healthy client must NOT be affected by the blown one.
	if !p.AgentAccountOK("healthy") {
		t.Fatalf("healthy agent must remain allowed even when another agent is blown")
	}
}

// TestAgentAccountOKStale verifies a stale (freshness window exceeded) client
// record is treated as fail-open rather than blocking signals indefinitely.
func TestAgentAccountOKStale(t *testing.T) {
	p := NewAgentProvider()
	defer close(p.snapshotStop)

	p.RecordAgentAccount("stale", &SnapshotAccount{Balance: 0, Equity: 0, FreeMargin: 0, Leverage: 100}, nil)
	// Simulate staleness by backdating the record.
	p.agentAccMu.Lock()
	if accts, ok := p.agentAccounts["stale"]; ok {
		if st, ok := accts["default"]; ok {
			st.updatedAt = time.Now().UTC().Add(-90 * time.Second)
		}
	}
	p.agentAccMu.Unlock()

	if p.AgentAccountOK("stale") {
		t.Fatalf("stale agent record must fail closed")
	}
}
