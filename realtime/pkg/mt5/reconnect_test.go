package mt5

import (
	"testing"
	"time"
)

func TestReconnect_ExponentialBackoff(t *testing.T) {
	r := NewReconnectManager(DefaultReconnectConfig())
	d1 := r.NextDelay()
	d2 := r.NextDelay()
	d3 := r.NextDelay()
	if d1 != 1*time.Second {
		t.Errorf("First delay: got %v, want 1s", d1)
	}
	if d2 != 2*time.Second {
		t.Errorf("Second delay: got %v, want 2s", d2)
	}
	if d3 != 4*time.Second {
		t.Errorf("Third delay: got %v, want 4s", d3)
	}
}

func TestReconnect_MaxDelay(t *testing.T) {
	r := NewReconnectManager(DefaultReconnectConfig())
	for i := 0; i < 10; i++ {
		r.NextDelay()
	}
	d := r.NextDelay()
	if d > 32*time.Second {
		t.Errorf("Delay should be capped at 32s, got %v", d)
	}
}

func TestReconnect_OnConnected(t *testing.T) {
	r := NewReconnectManager(DefaultReconnectConfig())
	r.NextDelay()
	r.NextDelay()
	r.OnConnected()
	if r.AttemptCount() != 0 {
		t.Error("Attempt count should reset on connect")
	}
	if !r.IsConnected() {
		t.Error("Should be connected after OnConnected")
	}
}

func TestReconnect_OnDisconnected(t *testing.T) {
	r := NewReconnectManager(DefaultReconnectConfig())
	r.OnConnected()
	r.OnDisconnected()
	if r.IsConnected() {
		t.Error("Should not be connected after OnDisconnected")
	}
}

func TestReconnect_ShouldRetry(t *testing.T) {
	cfg := DefaultReconnectConfig()
	cfg.MaxRetries = 3
	r := NewReconnectManager(cfg)
	for i := 0; i < 3; i++ {
		if !r.ShouldRetry() {
			t.Error("Should retry before max retries")
		}
		r.NextDelay()
	}
	if r.ShouldRetry() {
		t.Error("Should not retry after max retries")
	}
}
