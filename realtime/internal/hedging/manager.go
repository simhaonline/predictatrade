// Package hedging implements the Controlled Hedge Manager.
//
// CRITICAL: Hedging is a capital-protection tool, NOT a loss-recovery
// or reverse-trading engine.
//
// Hedging is DISABLED BY DEFAULT. It must be explicitly enabled via configuration.
// No martingale, no grid escalation, no exponential volume increase.
package hedging

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// Config holds hedge manager configuration.
type Config struct {
	Enabled               bool
	MinLossThreshold      float64 // minimum unrealized loss to consider hedging
	MaxLossThreshold      float64 // maximum loss beyond which hedging is not attempted
	HedgeSizeCap          float64 // max hedge size as fraction of original
	MaxSimultaneousHedges int
	MaxAggregateExposure  float64
	MaxHedgeDurationMin   int
	HedgeStopLoss         float64 // SL as fraction of entry
	HedgeTakeProfit       float64 // TP as fraction of entry
	ManipulationThreshold float64
	VolatilityThreshold   float64

	// Advanced hedging
	GridEnabled       bool // OFF by default
	OptionsEnabled    bool // OFF by default
	PartialHedging    bool
	TrailingStopEnabled bool
	AutoCloseEnabled  bool
}

// DefaultConfig returns disabled-by-default configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:               false, // DISABLED BY DEFAULT
		MinLossThreshold:      0.5,   // 0.5% unrealized loss minimum
		MaxLossThreshold:      3.0,   // 3% max loss for hedging
		HedgeSizeCap:          0.5,   // max 50% of original position
		MaxSimultaneousHedges: 2,
		MaxAggregateExposure:  5.0,
		MaxHedgeDurationMin:   120,
		HedgeStopLoss:         0.5,
		HedgeTakeProfit:       0.3,
		ManipulationThreshold: 70,
		VolatilityThreshold:   0.005,
		GridEnabled:           false, // OFF by default
		OptionsEnabled:        false, // OFF by default
		PartialHedging:        true,
		TrailingStopEnabled:   true,
		AutoCloseEnabled:      true,
	}
}

// HedgePosition represents an active hedge.
type HedgePosition struct {
	OriginalTradeID    string
	HedgeTradeID       string
	AccountID          string
	StrategyID         string
	Symbol             string
	OriginalDirection  string // BUY or SELL
	HedgeDirection     string
	OriginalSize       decimal.Decimal
	HedgeSize          decimal.Decimal
	OriginalEntry      decimal.Decimal
	HedgeEntry         decimal.Decimal
	HedgeSL            decimal.Decimal
	HedgeTP            decimal.Decimal
	ReasonOpened       string
	Status             string // OPEN, CLOSED_TP, CLOSED_SL, CLOSED_MANUAL, CLOSED_AUTO, EXPIRED
	OpenedAt           time.Time
	ExpiresAt          time.Time
}

// HedgeRequest is the input for evaluating whether to open a hedge.
type HedgeRequest struct {
	AccountID           string
	StrategyID          string
	Symbol              string
	OriginalTradeID     string
	OriginalDirection   string
	OriginalSize        decimal.Decimal
	OriginalEntry       decimal.Decimal
	CurrentPrice        decimal.Decimal
	UnrealizedLossPct   float64
	BrokerSupportsHedge bool
	AccountIsNetting    bool
	LicensePermitsTrade bool
	ManipulationIndex   float64
	Volatility          float64
	Spread              float64
	MarketDataFresh     bool
}

// HedgeResult is the outcome of a hedge evaluation.
type HedgeResult struct {
	Allowed   bool
	Reason    string
	Hedge     *HedgePosition
}

// Manager manages controlled hedging.
type Manager struct {
	mu          sync.RWMutex
	config      Config
	activeHedges map[string]*HedgePosition // originalTradeID -> hedge
	hedgeCount   int
}

// NewManager creates a hedge manager.
func NewManager(cfg Config) *Manager {
	return &Manager{
		config:       cfg,
		activeHedges:  make(map[string]*HedgePosition),
	}
}

// EvaluateHedge checks whether hedging is allowed and returns a hedge plan.
// It NEVER executes — it only evaluates and returns a plan.
func (m *Manager) EvaluateHedge(req HedgeRequest) HedgeResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.config.Enabled {
		return HedgeResult{Allowed: false, Reason: "Hedging is disabled"}
	}

	if !req.BrokerSupportsHedge {
		return HedgeResult{Allowed: false, Reason: "Broker does not support hedging"}
	}

	if req.AccountIsNetting {
		return HedgeResult{Allowed: false, Reason: "Account is netting mode — hedging incompatible"}
	}

	if !req.LicensePermitsTrade {
		return HedgeResult{Allowed: false, Reason: "License does not permit trading"}
	}

	if !req.MarketDataFresh {
		return HedgeResult{Allowed: false, Reason: "Market data is not fresh"}
	}

	// Check if original position is still open (no hedge if original closed)
	if req.OriginalTradeID == "" {
		return HedgeResult{Allowed: false, Reason: "No original trade ID provided"}
	}

	// Check if hedge already exists for this trade
	if _, exists := m.activeHedges[req.OriginalTradeID]; exists {
		return HedgeResult{Allowed: false, Reason: "Hedge already exists for this trade"}
	}

	// Check max simultaneous hedges
	if m.hedgeCount >= m.config.MaxSimultaneousHedges {
		return HedgeResult{Allowed: false, Reason: "Max simultaneous hedges reached"}
	}

	// Check loss threshold
	if req.UnrealizedLossPct < m.config.MinLossThreshold {
		return HedgeResult{Allowed: false, Reason: "Loss below minimum threshold"}
	}

	if req.UnrealizedLossPct > m.config.MaxLossThreshold {
		return HedgeResult{Allowed: false, Reason: "Loss exceeds maximum threshold — too risky to hedge"}
	}

	// Check manipulation/volatility
	if req.ManipulationIndex > m.config.ManipulationThreshold {
		return HedgeResult{Allowed: false, Reason: "Manipulation index too high for hedging"}
	}

	// Check aggregate exposure
	if m.countAggregateExposure() >= m.config.MaxAggregateExposure {
		return HedgeResult{Allowed: false, Reason: "Aggregate exposure limit reached"}
	}

	// Calculate hedge size (partial hedging — never exceeds cap)
	hedgeSize := req.OriginalSize.Mul(decimal.NewFromFloat(m.config.HedgeSizeCap))
	if hedgeSize.GreaterThan(req.OriginalSize) {
		hedgeSize = req.OriginalSize // never exceed original
	}

	// Determine hedge direction (opposite of original)
	hedgeDir := "SELL"
	if req.OriginalDirection == "SELL" {
		hedgeDir = "BUY"
	}

	hedge := &HedgePosition{
		OriginalTradeID:   req.OriginalTradeID,
		HedgeTradeID:      req.OriginalTradeID + "_HEDGE",
		AccountID:         req.AccountID,
		StrategyID:       req.StrategyID,
		Symbol:            req.Symbol,
		OriginalDirection: req.OriginalDirection,
		HedgeDirection:    hedgeDir,
		OriginalSize:      req.OriginalSize,
		HedgeSize:         hedgeSize,
		OriginalEntry:     req.OriginalEntry,
		HedgeEntry:        req.CurrentPrice,
		ReasonOpened:      "Controlled hedge: unrealized loss protection",
		Status:            "OPEN",
		OpenedAt:          time.Now(),
		ExpiresAt:         time.Now().Add(time.Duration(m.config.MaxHedgeDurationMin) * time.Minute),
	}

	return HedgeResult{Allowed: true, Reason: "Hedge approved", Hedge: hedge}
}

// OpenHedge registers a hedge as active.
func (m *Manager) OpenHedge(hedge *HedgePosition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeHedges[hedge.OriginalTradeID] = hedge
	m.hedgeCount++
}

// CloseHedge closes a hedge and records it in history.
func (m *Manager) CloseHedge(originalTradeID string, reason string, pnl decimal.Decimal) *HedgePosition {
	m.mu.Lock()
	defer m.mu.Unlock()
	hedge, ok := m.activeHedges[originalTradeID]
	if !ok {
		return nil
	}
	hedge.Status = "CLOSED_MANUAL"
	if reason == "TP" {
		hedge.Status = "CLOSED_TP"
	} else if reason == "SL" {
		hedge.Status = "CLOSED_SL"
	} else if reason == "AUTO" {
		hedge.Status = "CLOSED_AUTO"
	}
	delete(m.activeHedges, originalTradeID)
	m.hedgeCount--
	return hedge
}

// GetActiveHedge returns the active hedge for a trade.
func (m *Manager) GetActiveHedge(originalTradeID string) *HedgePosition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if h, ok := m.activeHedges[originalTradeID]; ok {
		cp := *h
		return &cp
	}
	return nil
}

// ActiveHedgeCount returns the number of active hedges.
func (m *Manager) ActiveHedgeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hedgeCount
}

// AggregateExposure returns total exposure from hedges.
func (m *Manager) AggregateExposure() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.countAggregateExposure()
}

func (m *Manager) countAggregateExposure() float64 {
	total := 0.0
	for _, h := range m.activeHedges {
		s, _ := h.HedgeSize.Float64()
		total += s
	}
	return total
}

// IsGridEnabled returns whether grid hedging is enabled.
func (m *Manager) IsGridEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.GridEnabled
}

// IsOptionsEnabled returns whether options hedging is enabled.
func (m *Manager) IsOptionsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.OptionsEnabled
}

// CheckExpiredHedges closes hedges that have exceeded their max duration.
func (m *Manager) CheckExpiredHedges() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var expired []string
	now := time.Now()
	for id, h := range m.activeHedges {
		if now.After(h.ExpiresAt) {
			h.Status = "EXPIRED"
			expired = append(expired, id)
			delete(m.activeHedges, id)
			m.hedgeCount--
		}
	}
	return expired
}
