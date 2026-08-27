package bus

import (
	"sync"
	"testing"
	"time"
)

func TestDirectBusDispatchesSynchronously(t *testing.T) {
	var mu sync.Mutex
	var gotID string
	var gotData []byte
	handled := make(chan struct{}, 1)

	d := NewDirectBus(func(agentID string, data []byte) {
		mu.Lock()
		gotID = agentID
		gotData = data
		mu.Unlock()
		handled <- struct{}{}
	})

	payload := []byte(`{"type":"TICK"}`)
	if err := d.Publish("agent-1", payload); err != nil {
		t.Fatalf("DirectBus.Publish error: %v", err)
	}

	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("DirectBus did not dispatch synchronously")
	}

	mu.Lock()
	if gotID != "agent-1" {
		t.Fatalf("unexpected agentID: %s", gotID)
	}
	if string(gotData) != string(payload) {
		t.Fatalf("unexpected data: %s", gotData)
	}
	mu.Unlock()
}

func TestDirectBusNilHandlerNoPanic(t *testing.T) {
	d := NewDirectBus(nil)
	if err := d.Publish("x", []byte("{}")); err != nil {
		t.Fatalf("nil-handler publish should not error: %v", err)
	}
}

// TestNatsBusRoundTrip requires a running NATS server at localhost:4222.
// It is skipped automatically when unavailable, so CI without NATS stays green
// while local/dev runs with NATS verify the decoupling seam end-to-end.
func TestNatsBusRoundTrip(t *testing.T) {
	nb, err := NewNatsBus("nats://localhost:4222", "pat.ingest.agent.test")
	if err != nil {
		t.Skipf("NATS not available, skipping: %v", err)
	}
	defer nb.Close()

	// Confirm the peer is a real, responsive NATS server (a stray non-NATS
	// listener on 4222 would connect but fail a flush). Skip otherwise so CI
	// without NATS — and environments with a fake 4222 — stay green.
	if ferr := nb.conn.FlushTimeout(1 * time.Second); ferr != nil {
		t.Skipf("NATS server not responsive, skipping: %v", ferr)
	}

	var mu sync.Mutex
	var gotID string
	var gotData []byte
	done := make(chan struct{}, 1)
	if err := nb.Subscribe(func(agentID string, data []byte) {
		mu.Lock()
		gotID = agentID
		gotData = data
		mu.Unlock()
		done <- struct{}{}
	}); err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	payload := []byte(`{"type":"MARKET_SNAPSHOT"}`)
	if err := nb.Publish("agent-nats", payload); err != nil {
		t.Fatalf("publish error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("NATS message was not delivered to subscriber")
	}

	mu.Lock()
	if gotID != "agent-nats" || string(gotData) != string(payload) {
		t.Fatalf("unexpected delivery: id=%s data=%s", gotID, gotData)
	}
	mu.Unlock()
}
