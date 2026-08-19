// Package recovery implements the Loss Recovery / Capital Protection Manager.
//
// CRITICAL SAFETY RULES:
// - No martingale, no doubling after losses
// - No automatic reverse trades (a loss never creates an opposite BUY/SELL)
// - Recovery reduces risk, never increases it
// - Daily loss circuit breaker uses CORRECT PnL sign logic:
//   daily_pnl_percent <= -max_daily_loss_percent (never abs())
// - State is isolated per account+strategy, not a single global
// - Duplicate broker close events are deduplicated
// - State survives restart via persistence
package recovery

import (
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// RecoveryState represents the loss-recovery state machine.
type RecoveryState string

const (
	StateNormal     RecoveryState = "NORMAL"
	StateRecovery   RecoveryState = "RECOVERY"
	StateHalted     RecoveryState = "HALTED"
	StateDailyLimit RecoveryState = "DAILY_LIMIT"
)

// Config holds loss-recovery configuration. All values are configurable.
type Config struct {
	MaxDailyLossPercent    float64 `json:"max_daily_loss_percent"`
	MaxDailyLossCount      int     `json:"max_daily_loss_count"`
	MaxConsecutiveLosses   int     `json:"max_consecutive_losses"`

	RecoverySizeMultiplier float64 `json:"recovery_size_multiplier"`
	RecoveryMinConfluence  float64 `json:"recovery_min_confluence"`
	RecoveryMinSetupGrade  string  `json:"recovery_min_setup_quality"`
	RecoveryMinConfidence  float64 `json:"recovery_min_confidence"`
	RecoveryMaxTrades      int     `json:"recovery_max_trades"`
	RecoveryExitAfterWins  int     `json:"recovery_exit_after_wins"`

	NormalCooldownMinutes   int `json:"normal_cooldown_minutes"`
	RecoveryCooldownMinutes int `json:"recovery_cooldown_minutes"`
	HaltCooldownMinutes     int `json:"halt_cooldown_minutes"`
}

// DefaultConfig returns conservative reference defaults.
// Existing production configuration that is more conservative takes precedence.
func DefaultConfig() Config {
	return Config{
		MaxDailyLossPercent:     2.0,
		MaxDailyLossCount:       3,
		MaxConsecutiveLosses:    2,
		RecoverySizeMultiplier:  0.50,
		RecoveryMinConfluence:   80,
		RecoveryMinSetupGrade:   "A",
		RecoveryMinConfidence:    75,
		RecoveryMaxTrades:       3,
		RecoveryExitAfterWins:    2,
		NormalCooldownMinutes:    5,
		RecoveryCooldownMinutes:  30,
		HaltCooldownMinutes:      60,
	}
}

// AccountStrategyKey isolates state per account+strategy+symbol.
type AccountStrategyKey struct {
	AccountID  string
	StrategyID string
	Symbol     string
}

// StateRecord holds the persisted recovery state for one account+strategy.
type StateRecord struct {
	Key                  AccountStrategyKey
	State                RecoveryState
	ConsecutiveLosses    int
	DailyLossCount       int
	DailyLossPercent     float64
	DailyPnL            decimal.Decimal
	StartingEquity      decimal.Decimal
	RecoveryTradesTaken int
	RecoveryWins        int
	CooldownUntil       time.Time
	HaltUntil           time.Time
	HaltReason          string
	LastTradeAt         time.Time
	LastLossAt          time.Time
	LastCloseEventID    string
	TradingDay          time.Time
}

// TradeResult represents a closed trade outcome fed to the recovery manager.
type TradeResult struct {
	AccountID    string
	StrategyID   string
	Symbol       string
	SignalID     string
	CloseEventID string
	PnL          decimal.Decimal
	IsWin        bool
	IsLoss       bool
	IsBreakeven  bool
	ClosedAt     time.Time
	TradingDay   time.Time
}

// BlockReason explains why a signal was blocked by recovery.
type BlockReason string

const (
	BlockRecoveryMode           BlockReason = "RECOVERY_MODE"
	BlockDailyLimit             BlockReason = "DAILY_LIMIT"
	BlockHalt                   BlockReason = "HALT"
	BlockCooldown               BlockReason = "COOLDOWN"
	BlockLowConfluenceRecovery  BlockReason = "LOW_CONFLUENCE_RECOVERY"
	BlockLowQualityRecovery     BlockReason = "LOW_QUALITY_RECOVERY"
	BlockLowConfidenceRecovery  BlockReason = "LOW_CONFIDENCE_RECOVERY"
	BlockMaxRecoveryTrades      BlockReason = "MAX_RECOVERY_TRADES"
)

// Manager manages loss recovery state per account+strategy.
type Manager struct {
	mu     sync.RWMutex
	config Config
	states map[AccountStrategyKey]*StateRecord
	processedCloseEvents map[string]bool
}

// NewManager creates a loss recovery manager with the given config.
func NewManager(cfg Config) *Manager {
	return &Manager{
		config:               cfg,
		states:               make(map[AccountStrategyKey]*StateRecord),
		processedCloseEvents: make(map[string]bool),
	}
}

// GetState returns the current recovery state for an account+strategy.
func (m *Manager) GetState(key AccountStrategyKey) RecoveryState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if rec, ok := m.states[key]; ok {
		return rec.State
	}
	return StateNormal
}

// GetStateRecord returns the full state record (for API/monitoring).
func (m *Manager) GetStateRecord(key AccountStrategyKey) *StateRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if rec, ok := m.states[key]; ok {
		cp := *rec
		return &cp
	}
	return nil
}

// SetStateRecord replaces the state record (used for restore from persistence).
func (m *Manager) SetStateRecord(rec *StateRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[rec.Key] = rec
}

// RecordTradeResult processes a closed trade outcome.
// Returns the updated state. Duplicate close events are ignored.
// A loss NEVER automatically creates an opposite BUY/SELL signal.
func (m *Manager) RecordTradeResult(result TradeResult) RecoveryState {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := AccountStrategyKey{
		AccountID:  result.AccountID,
		StrategyID: result.StrategyID,
		Symbol:     result.Symbol,
	}

	// Dedup: ignore duplicate broker close events
	if result.CloseEventID != "" {
		if m.processedCloseEvents[result.CloseEventID] {
			return m.currentStateLocked(key)
		}
		m.processedCloseEvents[result.CloseEventID] = true
	}

	rec := m.getOrCreateStateLocked(key, result.TradingDay)

	// Check for trading day rollover
	if !result.TradingDay.IsZero() && !isSameDay(rec.TradingDay, result.TradingDay) {
		rec.TradingDay = result.TradingDay
		rec.DailyLossCount = 0
		rec.DailyLossPercent = 0
		rec.DailyPnL = decimal.Zero
		rec.RecoveryTradesTaken = 0
		rec.RecoveryWins = 0
		if rec.State == StateDailyLimit {
			rec.State = StateNormal
		}
	}

	// Accumulate daily PnL
	rec.DailyPnL = rec.DailyPnL.Add(result.PnL)

	// Calculate daily PnL percent
	if !rec.StartingEquity.IsZero() {
		pnlPct, _ := rec.DailyPnL.Div(rec.StartingEquity).Mul(decimal.NewFromInt(100)).Float64()
		rec.DailyLossPercent = pnlPct
	}

	// Update counters
	if result.IsLoss {
		rec.ConsecutiveLosses++
		rec.DailyLossCount++
		rec.LastLossAt = result.ClosedAt
	} else if result.IsWin {
		rec.ConsecutiveLosses = 0
		if rec.State == StateRecovery {
			rec.RecoveryWins++
		}
	}
	rec.LastTradeAt = result.ClosedAt
	if result.CloseEventID != "" {
		rec.LastCloseEventID = result.CloseEventID
	}

	m.transitionLocked(rec)
	return rec.State
}

// transitionLocked applies deterministic state machine transitions.
func (m *Manager) transitionLocked(rec *StateRecord) {
	cfg := m.config

	// 1. Check halt expiry
	if rec.State == StateHalted {
		if !rec.HaltUntil.IsZero() && time.Now().After(rec.HaltUntil) {
			rec.State = StateRecovery
			rec.HaltUntil = time.Time{}
			rec.HaltReason = ""
		} else {
			return
		}
	}

	// 2. Check daily loss limit — CORRECT sign logic
	// daily_pnl_percent <= -max_daily_loss_percent
	// A profitable day must NEVER trigger a loss circuit breaker.
	if rec.DailyLossPercent <= -cfg.MaxDailyLossPercent {
		rec.State = StateDailyLimit
		rec.HaltReason = fmt.Sprintf("Daily loss limit: %.2f%% <= -%.2f%%", rec.DailyLossPercent, cfg.MaxDailyLossPercent)
		return
	}

	if rec.DailyLossCount >= cfg.MaxDailyLossCount {
		rec.State = StateDailyLimit
		rec.HaltReason = fmt.Sprintf("Daily loss count: %d >= %d", rec.DailyLossCount, cfg.MaxDailyLossCount)
		return
	}

	// 3. Consecutive losses → enter recovery
	if rec.ConsecutiveLosses >= cfg.MaxConsecutiveLosses && rec.State == StateNormal {
		rec.State = StateRecovery
		rec.RecoveryTradesTaken = 0
		rec.RecoveryWins = 0
		return
	}

	// 4. In recovery: check exit/halt conditions
	if rec.State == StateRecovery {
		if rec.RecoveryWins >= cfg.RecoveryExitAfterWins {
			rec.State = StateNormal
			rec.RecoveryTradesTaken = 0
			rec.RecoveryWins = 0
			rec.ConsecutiveLosses = 0
			return
		}
		if rec.RecoveryTradesTaken >= cfg.RecoveryMaxTrades {
			rec.State = StateHalted
			rec.HaltUntil = time.Now().Add(time.Duration(cfg.HaltCooldownMinutes) * time.Minute)
			rec.HaltReason = fmt.Sprintf("Max recovery trades: %d >= %d", rec.RecoveryTradesTaken, cfg.RecoveryMaxTrades)
			return
		}
	}

	if rec.State == StateDailyLimit {
		return
	}
}

// CheckSignal evaluates whether a signal should be blocked by recovery.
// Returns (allowed, blockReason). This NEVER creates a trade — it only blocks.
func (m *Manager) CheckSignal(key AccountStrategyKey, confluence float64, setupGrade string, confidence float64) (bool, BlockReason) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, ok := m.states[key]
	if !ok {
		return true, ""
	}

	switch rec.State {
	case StateHalted:
		if !rec.HaltUntil.IsZero() && time.Now().Before(rec.HaltUntil) {
			return false, BlockHalt
		}
	case StateDailyLimit:
		return false, BlockDailyLimit
	case StateRecovery:
		if confluence < m.config.RecoveryMinConfluence {
			return false, BlockLowConfluenceRecovery
		}
		if setupGrade != "A" && setupGrade != "A+" {
			return false, BlockLowQualityRecovery
		}
		if confidence < m.config.RecoveryMinConfidence {
			return false, BlockLowConfidenceRecovery
		}
		if rec.RecoveryTradesTaken >= m.config.RecoveryMaxTrades {
			return false, BlockMaxRecoveryTrades
		}
	}

	if !rec.CooldownUntil.IsZero() && time.Now().Before(rec.CooldownUntil) {
		return false, BlockCooldown
	}

	return true, ""
}

// IncrementRecoveryTrade tracks that a recovery-mode trade was placed.
func (m *Manager) IncrementRecoveryTrade(key AccountStrategyKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, ok := m.states[key]; ok {
		rec.RecoveryTradesTaken++
	}
}

// SetCooldown applies a cooldown based on current state.
func (m *Manager) SetCooldown(key AccountStrategyKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.states[key]
	if !ok {
		return
	}
	var minutes int
	switch rec.State {
	case StateRecovery:
		minutes = m.config.RecoveryCooldownMinutes
	case StateHalted:
		minutes = m.config.HaltCooldownMinutes
	default:
		minutes = m.config.NormalCooldownMinutes
	}
	rec.CooldownUntil = time.Now().Add(time.Duration(minutes) * time.Minute)
}

// GetSizeMultiplier returns the position sizing multiplier for the current state.
func (m *Manager) GetSizeMultiplier(key AccountStrategyKey) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.states[key]
	if !ok {
		return 1.0
	}
	switch rec.State {
	case StateRecovery:
		return m.config.RecoverySizeMultiplier
	case StateHalted, StateDailyLimit:
		return 0.0
	default:
		return 1.0
	}
}

// ResetDaily resets daily counters for a new trading day.
func (m *Manager) ResetDaily(tradingDay time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rec := range m.states {
		if !isSameDay(rec.TradingDay, tradingDay) {
			rec.TradingDay = tradingDay
			rec.DailyLossCount = 0
			rec.DailyLossPercent = 0
			rec.DailyPnL = decimal.Zero
			rec.RecoveryTradesTaken = 0
			rec.RecoveryWins = 0
			if rec.State == StateDailyLimit {
				rec.State = StateNormal
				rec.HaltReason = ""
			}
		}
	}
}

// AllStates returns a snapshot of all state records.
func (m *Manager) AllStates() []StateRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]StateRecord, 0, len(m.states))
	for _, rec := range m.states {
		result = append(result, *rec)
	}
	return result
}

// RestoreStates restores state from persistence (restart safety).
func (m *Manager) RestoreStates(records []StateRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rec := range records {
		r := rec
		m.states[r.Key] = &r
	}
}

// ClearHalt clears halt state (admin operation).
func (m *Manager) ClearHalt(key AccountStrategyKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, ok := m.states[key]; ok {
		rec.State = StateNormal
		rec.HaltUntil = time.Time{}
		rec.HaltReason = ""
		rec.ConsecutiveLosses = 0
		rec.RecoveryTradesTaken = 0
		rec.RecoveryWins = 0
	}
}

func (m *Manager) getOrCreateStateLocked(key AccountStrategyKey, tradingDay time.Time) *StateRecord {
	if rec, ok := m.states[key]; ok {
		return rec
	}
	rec := &StateRecord{
		Key:       key,
		State:     StateNormal,
		TradingDay: tradingDay,
	}
	if tradingDay.IsZero() {
		rec.TradingDay = time.Now().UTC()
	}
	m.states[key] = rec
	return rec
}

func (m *Manager) currentStateLocked(key AccountStrategyKey) RecoveryState {
	if rec, ok := m.states[key]; ok {
		return rec.State
	}
	return StateNormal
}

func isSameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
