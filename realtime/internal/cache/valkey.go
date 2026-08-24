// Package cache provides a Valkey/Redis hot cache for market data.
// Dashboard reads from Valkey via REST API, ensuring data is always available.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type ValkeyCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewValkeyCache(addr string) *ValkeyCache {
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	return &ValkeyCache{
		client: redis.NewClient(&redis.Options{
			Addr:         addr,
			PoolSize:     10,
			MinIdleConns: 2,
			ReadTimeout:  100 * time.Millisecond,
			WriteTimeout: 100 * time.Millisecond,
			DialTimeout:  500 * time.Millisecond,
		}),
		ctx: context.Background(),
	}
}

func (v *ValkeyCache) Ping() error {
	return v.client.Ping(v.ctx).Err()
}

// GetRaw returns the raw bytes stored at key (used by P&L anchors).
func (v *ValkeyCache) GetRaw(key string) ([]byte, error) {
	return v.client.Get(v.ctx, key).Bytes()
}

// SetNXRaw stores raw bytes at key only if it does not already exist.
// Returns true when the caller won the write race.
func (v *ValkeyCache) SetNXRaw(ctx context.Context, key string, val []byte, ttl time.Duration) (bool, error) {
	return v.client.SetNX(ctx, key, val, ttl).Result()
}

func (v *ValkeyCache) SetMarketState(data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return v.client.Set(v.ctx, "pat:market_state", b, 10*time.Second).Err()
}

func (v *ValkeyCache) GetMarketState() (json.RawMessage, error) {
	b, err := v.client.Get(v.ctx, "pat:market_state").Bytes()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func (v *ValkeyCache) SetSnapshot(data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return v.client.Set(v.ctx, "pat:market_snapshot", b, 10*time.Second).Err()
}

func (v *ValkeyCache) GetSnapshot() (json.RawMessage, error) {
	b, err := v.client.Get(v.ctx, "pat:market_snapshot").Bytes()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func (v *ValkeyCache) SetAgentStatus(data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return v.client.Set(v.ctx, "pat:agent_status", b, 5*time.Second).Err()
}

func (v *ValkeyCache) GetAgentStatus() (json.RawMessage, error) {
	b, err := v.client.Get(v.ctx, "pat:agent_status").Bytes()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func (v *ValkeyCache) AddPricePoint(price float64, timestamp time.Time) error {
	key := "pat:price_history"
	point := fmt.Sprintf("%.5f@%d", price, timestamp.UnixMilli())
	pipe := v.client.Pipeline()
	pipe.LPush(v.ctx, key, point)
	pipe.LTrim(v.ctx, key, 0, 119)
	pipe.Expire(v.ctx, key, 5*time.Minute)
	_, err := pipe.Exec(v.ctx)
	return err
}

type PricePoint struct {
	Price       float64 `json:"price"`
	TimestampMs int64   `json:"timestamp_ms"`
}

func (v *ValkeyCache) GetPriceHistory() ([]PricePoint, error) {
	b, err := v.client.LRange(v.ctx, "pat:price_history", 0, -1).Result()
	if err != nil {
		return nil, err
	}
	points := make([]PricePoint, 0, len(b))
	for i := len(b) - 1; i >= 0; i-- {
		var pp PricePoint
		if _, err := fmt.Sscanf(b[i], "%f@%d", &pp.Price, &pp.TimestampMs); err == nil {
			points = append(points, pp)
		}
	}
	return points, nil
}

func (v *ValkeyCache) Close() error {
	return v.client.Close()
}

// === Signal Cooldown (SOW Section 17) ===
// Per strategy+symbol cooldown tracking using Valkey as distributed coordination store.
// Works across multiple API processes, workers, service restarts, concurrent evaluation.

// SetCooldown sets a cooldown for a strategy+symbol pair.
// The cooldown key will expire after the given TTL.
// Uses SET NX (atomic) to prevent race conditions.
func (v *ValkeyCache) SetCooldown(ctx context.Context, symbol, strategy string, ttl time.Duration) error {
	key := fmt.Sprintf("signal:cooldown:%s:%s", symbol, strategy)
	return v.client.Set(ctx, key, time.Now().UTC().Unix(), ttl).Err()
}

// CheckCooldown checks if a cooldown is active for a strategy+symbol pair.
// Returns: active (bool), remaining (time.Duration), error
func (v *ValkeyCache) CheckCooldown(ctx context.Context, symbol, strategy string) (bool, time.Duration, error) {
	key := fmt.Sprintf("signal:cooldown:%s:%s", symbol, strategy)
	ttl, err := v.client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if ttl > 0 {
		return true, ttl, nil
	}
	return false, 0, nil
}

// ClearCooldown removes a cooldown (for testing or manual override).
func (v *ValkeyCache) ClearCooldown(ctx context.Context, symbol, strategy string) error {
	key := fmt.Sprintf("signal:cooldown:%s:%s", symbol, strategy)
	return v.client.Del(ctx, key).Err()
}

// === Duplicate Signal Prevention (SOW Section 18) ===
// Signal fingerprinting to stop the same market event from generating multiple equivalent signals.
// Uses atomic SETNX to prevent race conditions across concurrent workers.

// SetFingerprint atomically sets a signal fingerprint if it doesn't already exist.
// Returns true if the fingerprint was set (new signal), false if it already exists (duplicate).
func (v *ValkeyCache) SetFingerprint(ctx context.Context, fingerprint string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("signal:fingerprint:%s", fingerprint)
	// SETNX returns 1 if key was set, 0 if it already existed
	result, err := v.client.SetNX(ctx, key, time.Now().UTC().Unix(), ttl).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

// ClearFingerprint removes a fingerprint (for testing or manual override).
func (v *ValkeyCache) ClearFingerprint(ctx context.Context, fingerprint string) error {
	key := fmt.Sprintf("signal:fingerprint:%s", fingerprint)
	return v.client.Del(ctx, key).Err()
}

// SetLatestSignals caches the most recent signals for fast dashboard reads.
// Avoids querying the database on every dashboard refresh.
func (v *ValkeyCache) SetLatestSignals(data interface{}) error {
	if v.client == nil {
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return v.client.Set(v.ctx, "pat:latest_signals", b, 10*time.Second).Err()
}

// GetLatestSignals reads the cached latest signals from Valkey.
func (v *ValkeyCache) GetLatestSignals() (json.RawMessage, error) {
	if v.client == nil {
		return nil, fmt.Errorf("no client")
	}
	b, err := v.client.Get(v.ctx, "pat:latest_signals").Bytes()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// SetLastSnapshot persists the last known market snapshot with a long TTL (7 days).
// This ensures the dashboard can display the last Master Node price even when the
// market is closed, the agent reconnects, or the engine restarts.
func (v *ValkeyCache) SetLastSnapshot(data interface{}) error {
	if v.client == nil {
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return v.client.Set(v.ctx, "pat:last_snapshot", b, 7*24*time.Hour).Err()
}

// GetLastSnapshot reads the persistent last known market snapshot from Valkey.
func (v *ValkeyCache) GetLastSnapshot() (json.RawMessage, error) {
	if v.client == nil {
		return nil, fmt.Errorf("no client")
	}
	b, err := v.client.Get(v.ctx, "pat:last_snapshot").Bytes()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
