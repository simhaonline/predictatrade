// Package mt5 provides MT5 WebSocket auto-reconnection with exponential backoff.
package mt5

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// ReconnectConfig holds reconnection parameters.
type ReconnectConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	MaxRetries   int
}

// DefaultReconnectConfig returns safe defaults: 1s, 2s, 4s, 8s, 16s, 32s max.
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     32 * time.Second,
		MaxRetries:   10,
	}
}

// ReconnectManager manages WebSocket reconnection with exponential backoff.
type ReconnectManager struct {
	mu          sync.Mutex
	config      ReconnectConfig
	attempt     int
	connected   bool
	lastConnect time.Time
}

// NewReconnectManager creates a reconnection manager.
func NewReconnectManager(cfg ReconnectConfig) *ReconnectManager {
	return &ReconnectManager{config: cfg}
}

// NextDelay returns the next backoff delay.
func (r *ReconnectManager) NextDelay() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	delay := time.Duration(math.Pow(2, float64(r.attempt))) * r.config.InitialDelay
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}
	r.attempt++
	return delay
}

// OnConnected marks the connection as successful and resets the attempt counter.
func (r *ReconnectManager) OnConnected() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connected = true
	r.attempt = 0
	r.lastConnect = time.Now()
}

// OnDisconnected marks the connection as lost.
func (r *ReconnectManager) OnDisconnected() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connected = false
}

// IsConnected returns the current connection state.
func (r *ReconnectManager) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connected
}

// AttemptCount returns the current reconnection attempt count.
func (r *ReconnectManager) AttemptCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.attempt
}

// ShouldRetry returns true if reconnection should be attempted.
func (r *ReconnectManager) ShouldRetry() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.attempt < r.config.MaxRetries
}

// PingURL checks if a URL is reachable with a 1s timeout.
func PingURL(url string) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// String returns a human-readable status.
func (r *ReconnectManager) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fmt.Sprintf("connected=%v attempts=%d lastConnect=%s", r.connected, r.attempt, r.lastConnect.Format(time.RFC3339))
}
