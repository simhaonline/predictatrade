package marketdata

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// stubStateUpdater is a concurrency-safe stand-in for the real StateManager used
// by AgentProvider.HandleAgentMessage during the load test.
type stubStateUpdater struct {
	mu    sync.Mutex
	calls int
}

func (s *stubStateUpdater) Update(symbol string, update func(any)) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	update(map[string]string{"symbol": symbol}) // mergeFn is nil in the test, so this is unused
}

// TestAgentProviderConcurrentIngest hammers HandleAgentMessage from many agents
// concurrently (snapshots + ticks) to exercise the ingest bulkhead maps and the
// single-writer snapshotMergeLoop under load. Run with: go test -race.
func TestAgentProviderConcurrentIngest(t *testing.T) {
	p := NewAgentProvider()
	defer close(p.snapshotStop)
	p.stateMgr = &stubStateUpdater{}

	var wg sync.WaitGroup
	for a := 0; a < 32; a++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			aid := fmt.Sprintf("agent-%d", id)
			for i := 0; i < 500; i++ {
				snap := []byte(`{"type":"MARKET_SNAPSHOT","symbol":"XAUUSD","bars":{"M5":{"time":"2026-01-01T00:00:00Z"}},"account_info":{"balance":1000}}`)
				p.HandleAgentMessage(aid, snap)
				tick := []byte(fmt.Sprintf(`{"type":"TICK","bid":%d,"ask":%d}`, 100+i, 101+i))
				p.HandleAgentMessage(aid, tick)
			}
		}(a)
	}
	wg.Wait()

	// Let the single-writer merge loop drain any pending snapshots.
	time.Sleep(200 * time.Millisecond)
}

// TestIngestGuardDedupe verifies that identical repeated low-value messages
// (CAPITAL_WARNING / CAPITAL_PROTECTION) are dropped after the first, while a
// changed body still passes. This is the bulkhead that prevents a per-tick
// emitter from flooding the engine.
func TestIngestGuardDedupe(t *testing.T) {
	p := NewAgentProvider()
	defer close(p.snapshotStop)

	body := []byte(`{"type":"CAPITAL_WARNING","daily_pnl_pct":-3.0}`)
	if !p.ingestGuard("agent1", "CAPITAL_WARNING", body) {
		t.Fatal("first CAPITAL_WARNING should be allowed")
	}
	// Identical repeat must be dropped.
	if p.ingestGuard("agent1", "CAPITAL_WARNING", body) {
		t.Fatal("identical repeated CAPITAL_WARNING must be dropped")
	}
	// A changed body (state change) must pass.
	changed := []byte(`{"type":"CAPITAL_WARNING","daily_pnl_pct":-4.0}`)
	if !p.ingestGuard("agent1", "CAPITAL_WARNING", changed) {
		t.Fatal("changed CAPITAL_WARNING must be allowed")
	}
	// Different agent with same body must still pass (per-agent dedupe).
	if !p.ingestGuard("agent2", "CAPITAL_WARNING", body) {
		t.Fatal("different agent should be allowed")
	}
}

// TestIngestGuardRateLimitQuarantine verifies a flooding agent is quarantined
// and stays quarantined, while a normal agent is unaffected.
func TestIngestGuardRateLimitQuarantine(t *testing.T) {
	p := NewAgentProvider()
	defer close(p.snapshotStop)

	// Normal agent: 100 ordinary messages -> all allowed.
	for i := 0; i < 100; i++ {
		if !p.ingestGuard("normal", "TICK", []byte(`{"type":"TICK","bid":1,"ask":2}`)) {
			t.Fatalf("normal agent message %d should be allowed", i)
		}
	}

	// Flooding agent: exceed the per-window cap -> quarantined.
	flood := []byte(`{"type":"HEARTBEAT","x":1}`)
	allowed := 0
	for i := 0; i < maxAgentMsgsPerWindow+50; i++ {
		if p.ingestGuard("flooder", "HEARTBEAT", flood) {
			allowed++
		}
	}
	if allowed > maxAgentMsgsPerWindow {
		t.Fatalf("expected at most %d allowed before quarantine, got %d", maxAgentMsgsPerWindow, allowed)
	}
	// While quarantined, everything is dropped.
	for i := 0; i < 10; i++ {
		if p.ingestGuard("flooder", "TICK", []byte(`{"type":"TICK","bid":1,"ask":2}`)) {
			t.Fatal("quarantined agent must be dropped")
		}
	}
	// A different agent is not affected by the flooder's quarantine.
	if !p.ingestGuard("other", "TICK", []byte(`{"type":"TICK","bid":1,"ask":2}`)) {
		t.Fatal("other agent must not be affected by flooder quarantine")
	}
}
