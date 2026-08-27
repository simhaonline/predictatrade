package marketdata

import (
	"testing"
	"time"
)

// TestLastMarketDataAt tracks live market-data receipt. It must only advance on
// ticks/snapshots (not on heartbeats/other messages), and the getter must be
// safe to call and reflect the most recent receipt.
func TestLastMarketDataAt(t *testing.T) {
	p := NewAgentProvider()
	defer close(p.snapshotStop)

	// Initially zero.
	if !p.LastMarketDataAt().IsZero() {
		t.Fatalf("expected zero initial lastMarketDataAt")
	}

	// A heartbeat must NOT advance it.
	p.HandleAgentMessage("a1", []byte(`{"type":"HEARTBEAT"}`))
	if !p.LastMarketDataAt().IsZero() {
		t.Fatalf("heartbeat must not advance lastMarketDataAt")
	}

	// A tick must advance it.
	p.HandleAgentMessage("a1", []byte(`{"type":"TICK","bid":1,"ask":2}`))
	if p.LastMarketDataAt().IsZero() {
		t.Fatalf("tick must advance lastMarketDataAt")
	}
	first := p.LastMarketDataAt()
	if time.Since(first) > 5*time.Second {
		t.Fatalf("lastMarketDataAt should be recent, got %v", first)
	}

	// A snapshot must also advance it.
	time.Sleep(5 * time.Millisecond)
	p.HandleAgentMessage("a1", []byte(`{"type":"MARKET_SNAPSHOT","symbol":"XAUUSD"}`))
	if !p.LastMarketDataAt().After(first) {
		t.Fatalf("snapshot must advance lastMarketDataAt beyond %v", first)
	}
}
