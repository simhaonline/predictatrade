package signal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/predictatrade/realtime/internal/cache"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// CooldownManager manages per-strategy+symbol cooldowns using Valkey.
// SOW Section 17: Signal cooldown must work across restarts, concurrent workers, multiple processes.
// Cooldown durations come from strategy configuration (StrategyResult.CooldownMinutes).
type CooldownManager struct {
	cache *cache.ValkeyCache
}

// NewCooldownManager creates a new cooldown manager.
// If cache is nil, cooldown enforcement is disabled (fail-open for testing only).
// In production, cache must be non-nil (fail-safe: see CheckCooldown).
func NewCooldownManager(cache *cache.ValkeyCache) *CooldownManager {
	return &CooldownManager{cache: cache}
}

// CheckCooldown checks if a cooldown is active for a strategy+symbol pair.
// Returns: active (bool), remaining (time.Duration), error
// FAIL-SAFE: if Valkey is unavailable, returns active=false (allows evaluation)
// to prevent total signal blackout during infrastructure outage.
// The duplicate prevention layer provides additional protection.
func (cm *CooldownManager) CheckCooldown(ctx context.Context, symbol string, strategyID types.StrategyID) (bool, time.Duration, error) {
	if cm.cache == nil {
		return false, 0, nil
	}
	return cm.cache.CheckCooldown(ctx, symbol, string(strategyID))
}

// SetCooldown sets a cooldown for a strategy+symbol pair.
func (cm *CooldownManager) SetCooldown(ctx context.Context, symbol string, strategyID types.StrategyID, cooldownMinutes int) error {
	if cm.cache == nil || cooldownMinutes <= 0 {
		return nil
	}
	ttl := time.Duration(cooldownMinutes) * time.Minute
	return cm.cache.SetCooldown(ctx, symbol, string(strategyID), ttl)
}

// CooldownReason creates a standardized cooldown reason.
func CooldownReason(strategy types.StrategyID, symbol string, remaining time.Duration) types.NoTradeReason {
	return types.NoTradeReason(fmt.Sprintf("STRATEGY_COOLDOWN_ACTIVE:%s:%s:%ds", strategy, symbol, int(remaining.Seconds())))
}

// === Duplicate Signal Prevention (SOW Section 18) ===

// DuplicateChecker prevents duplicate signals using fingerprinting in Valkey.
// Fingerprints use meaningful event identity (symbol, strategy, direction, structural anchors, entry zone).
// NOT volatile timestamps that would make every evaluation unique.
type DuplicateChecker struct {
	cache *cache.ValkeyCache
}

// NewDuplicateChecker creates a new duplicate checker.
func NewDuplicateChecker(cache *cache.ValkeyCache) *DuplicateChecker {
	return &DuplicateChecker{cache: cache}
}

// ComputeFingerprint creates a deterministic fingerprint for a signal candidate.
// Inputs: symbol, strategy, direction, entry zone (rounded), structural anchor timestamps, BOS/CHoCH event.
// Does NOT include volatile timestamps or tiny price differences that would make every evaluation unique.
func ComputeFingerprint(symbol string, strategyID types.StrategyID, direction types.Direction,
	entryPrice, stopLoss decimal.Decimal, bosTime, chochTime time.Time) string {
	// Round entry and SL to 2 decimal places to avoid fingerprint churn from micro-movements
	entryStr := fmt.Sprintf("%.2f", roundTo2(entryPrice))
	slStr := fmt.Sprintf("%.2f", roundTo2(stopLoss))

	// Structural anchor timestamps (these change when structure actually changes)
	bosStr := ""
	if !bosTime.IsZero() {
		bosStr = bosTime.UTC().Format("2006-01-02T15:04Z") // Minute precision
	}
	chochStr := ""
	if !chochTime.IsZero() {
		chochStr = chochTime.UTC().Format("2006-01-02T15:04Z")
	}

	// Deterministic canonical representation
	canonical := strings.Join([]string{
		symbol,
		string(strategyID),
		string(direction),
		entryStr,
		slStr,
		bosStr,
		chochStr,
	}, "|")

	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:16]) // 16 bytes = 32 hex chars, sufficient uniqueness
}

// CheckDuplicate atomically checks if a signal with this fingerprint has already been published.
// Returns true if this is a NEW signal (not duplicate), false if duplicate.
// FAIL-SAFE: if Valkey is unavailable, returns true (allows signal) to prevent total blackout.
// The cooldown layer provides additional protection.
func (dc *DuplicateChecker) CheckDuplicate(ctx context.Context, fingerprint string, ttl time.Duration) (isNew bool, err error) {
	if dc.cache == nil {
		return true, nil // No cache — allow (testing mode)
	}
	return dc.cache.SetFingerprint(ctx, fingerprint, ttl)
}

// DuplicateReason creates a standardized duplicate reason.
func DuplicateReason(strategy types.StrategyID) types.NoTradeReason {
	return types.NoTradeReason(fmt.Sprintf("DUPLICATE_SIGNAL:%s", strategy))
}

// roundTo2 rounds a decimal to 2 decimal places.
func roundTo2(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return float64(int(f*100+0.5)) / 100.0
}

// ComputeFingerprintWithBar creates a canonical closed-bar idempotency fingerprint.
// prompt.md Section 28: Include market_bar_time for canonical closed-bar decisions.
// symbol + strategy + strategy_version + direction + closed bar timestamp + signal class
// This ensures only one canonical closed-bar decision exists per bar.
func ComputeFingerprintWithBar(symbol string, strategyID types.StrategyID, direction types.Direction,
	entryPrice, stopLoss decimal.Decimal, bosTime, chochTime time.Time, barTime time.Time) string {
	// Round entry and SL to 2 decimal places to avoid fingerprint churn from micro-movements
	entryStr := fmt.Sprintf("%.2f", roundTo2(entryPrice))
	slStr := fmt.Sprintf("%.2f", roundTo2(stopLoss))

	// Structural anchor timestamps (these change when structure actually changes)
	bosStr := ""
	if !bosTime.IsZero() {
		bosStr = bosTime.UTC().Format("2006-01-02T15:04Z") // Minute precision
	}
	chochStr := ""
	if !chochTime.IsZero() {
		chochStr = chochTime.UTC().Format("2006-01-02T15:04Z")
	}

	// Market bar time — canonical closed-bar idempotency (prompt.md Section 28)
	barStr := ""
	if !barTime.IsZero() {
		barStr = barTime.UTC().Format("2006-01-02T15:04Z")
	}

	// Deterministic canonical representation with market_bar_time
	canonical := strings.Join([]string{
		symbol,
		string(strategyID),
		"1.0", // strategy_version
		string(direction),
		entryStr,
		slStr,
		bosStr,
		chochStr,
		barStr, // market_bar_time for canonical idempotency
	}, "|")

	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:16])
}
