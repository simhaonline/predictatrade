// Package breakout implements the News Breakout pending-order planning engine.
//
// Operator-authorized implementation (v1.9.0).
// DISABLED BY DEFAULT — must be explicitly enabled via NEWS_BREAKOUT_ENABLED=true.
//
// This package creates LOGICAL breakout plans. It does NOT place broker orders.
// Order execution is handled by the MT4/MT5 EA via the Windows Agent.
package breakout

import (
	"fmt"
	"sync"
	"time"

	"github.com/predictatrade/realtime/pkg/news"
	"github.com/shopspring/decimal"
)

// PlanStatus represents the lifecycle state of a breakout plan.
type PlanStatus string

const (
	StatusCreated   PlanStatus = "CREATED"
	StatusValidated PlanStatus = "VALIDATED"
	StatusArmed     PlanStatus = "ARMED"
	StatusTriggered PlanStatus = "TRIGGERED"
	StatusExpired   PlanStatus = "EXPIRED"
	StatusRejected  PlanStatus = "REJECTED"
	StatusCancelled PlanStatus = "CANCELLED"
)

// BreakoutPlan is the logical pending-order plan for a news event.
type BreakoutPlan struct {
	PlanID           string          `json:"plan_id"`
	EventID          string          `json:"event_id"`
	Symbol           string          `json:"symbol"`
	Strategy         string          `json:"strategy"`
	CreatedAt        time.Time       `json:"created_at"`
	EventTime        time.Time       `json:"event_time"`
	BuyStopEntry     decimal.Decimal `json:"buy_stop_entry"`
	SellStopEntry    decimal.Decimal `json:"sell_stop_entry"`
	BuyStopSL        decimal.Decimal `json:"buy_stop_sl"`
	SellStopSL       decimal.Decimal `json:"sell_stop_sl"`
	BuyStopTP        decimal.Decimal `json:"buy_stop_tp"`
	SellStopTP       decimal.Decimal `json:"sell_stop_tp"`
	Volume           decimal.Decimal `json:"volume"`
	RiskPct          float64         `json:"risk_pct"`
	Expiry           time.Time       `json:"expiry"`
	OcoGroupID       string          `json:"oco_group_id"`
	Status           PlanStatus      `json:"status"`
	RejectionReason  string          `json:"rejection_reason,omitempty"`
}

// Config holds news breakout configuration.
type Config struct {
	Enabled             bool    `json:"enabled"`
	PrepareSeconds      int     `json:"prepare_seconds"`       // how early to prepare before event
	ExpirySeconds       int     `json:"expiry_seconds"`        // how long after event to expire
	EntryATRMultiplier  float64 `json:"entry_atr_multiplier"`  // entry distance as ATR multiple
	MaxSpread           float64 `json:"max_spread"`            // max spread to allow breakout
	MaxRiskPct          float64 `json:"max_risk_pct"`          // max risk per breakout
	SLATRMultiplier     float64 `json:"sl_atr_multiplier"`     // SL distance as ATR multiple
	TPATRMultiplier     float64 `json:"tp_atr_multiplier"`     // TP distance as ATR multiplier
}

// DefaultConfig returns disabled-by-default configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:            false, // DISABLED BY DEFAULT
		PrepareSeconds:     120,   // prepare 2 min before event
		ExpirySeconds:      300,   // expire 5 min after event
		EntryATRMultiplier: 0.5,   // 0.5×ATR entry distance
		MaxSpread:          3.0,   // max 3 pips spread
		MaxRiskPct:         1.0,   // max 1% risk
		SLATRMultiplier:    1.0,   // 1.0×ATR stop loss
		TPATRMultiplier:    2.0,   // 2.0×ATR take profit
	}
}

// EligibilityInput provides all inputs for breakout eligibility checking.
type EligibilityInput struct {
	NewsMode         news.NewsMode
	ProviderHealthy  bool
	Event            *news.NewsEvent
	CurrentPrice     decimal.Decimal
	ATR              decimal.Decimal
	Spread           float64
	MaxSpread        float64
	SessionAllowed   bool
	DailyLossClear   bool
	DrawdownClear    bool
	ExposureClear    bool
	MarginSufficient bool
	LicenseActive    bool
	EntitlementOK    bool
	Equity           decimal.Decimal
	StopDistancePrice decimal.Decimal
	TickSize         decimal.Decimal
	TickValue        decimal.Decimal
}

// EligibilityResult contains the eligibility decision and rejection reason.
type EligibilityResult struct {
	Eligible     bool   `json:"eligible"`
	RejectReason string `json:"reject_reason,omitempty"`
}

// CheckEligibility verifies all conditions for news breakout preparation.
// Returns eligible=true ONLY when ALL conditions pass.
func CheckEligibility(input EligibilityInput) EligibilityResult {
	// Check news mode FIRST before any other gate
	if input.NewsMode != news.NewsModeEventBreakout {
		return EligibilityResult{Eligible: false, RejectReason: "NEWS_MODE_NOT_EVENT_BREAKOUT"}
	}
	if !input.ProviderHealthy {
		return EligibilityResult{Eligible: false, RejectReason: "NEWS_PROVIDER_UNHEALTHY"}
	}
	if input.Event == nil {
		return EligibilityResult{Eligible: false, RejectReason: "NO_EVENT"}
	}
	if !input.Event.IsHighImpact() {
		return EligibilityResult{Eligible: false, RejectReason: "EVENT_IMPACT_BELOW_THRESHOLD"}
	}
	if !input.Event.IsUSDRelevant() {
		return EligibilityResult{Eligible: false, RejectReason: "EVENT_NOT_USD_RELEVANT"}
	}
	if !input.SessionAllowed {
		return EligibilityResult{Eligible: false, RejectReason: "SESSION_NOT_ALLOWED"}
	}
	if input.Spread > input.MaxSpread {
		return EligibilityResult{Eligible: false, RejectReason: "SPREAD_TOO_HIGH"}
	}
	if !input.DailyLossClear {
		return EligibilityResult{Eligible: false, RejectReason: "DAILY_LOSS_GATE_BLOCKED"}
	}
	if !input.DrawdownClear {
		return EligibilityResult{Eligible: false, RejectReason: "DRAWDOWN_GATE_BLOCKED"}
	}
	if !input.ExposureClear {
		return EligibilityResult{Eligible: false, RejectReason: "EXPOSURE_GATE_BLOCKED"}
	}
	if !input.MarginSufficient {
		return EligibilityResult{Eligible: false, RejectReason: "MARGIN_INSUFFICIENT"}
	}
	if !input.LicenseActive {
		return EligibilityResult{Eligible: false, RejectReason: "LICENSE_NOT_ACTIVE"}
	}
	if !input.EntitlementOK {
		return EligibilityResult{Eligible: false, RejectReason: "ENTITLEMENT_NOT_OK"}
	}
	if input.CurrentPrice.IsZero() || input.ATR.IsZero() {
		return EligibilityResult{Eligible: false, RejectReason: "MARKET_DATA_INVALID"}
	}
	return EligibilityResult{Eligible: true}
}

// CreatePlan generates a logical breakout plan from the eligibility input.
// This does NOT place broker orders — it creates the plan for OCO coordination.
func CreatePlan(input EligibilityInput, cfg Config, planID, ocoGroupID string) (*BreakoutPlan, error) {
	elig := CheckEligibility(input)
	if !elig.Eligible {
		return nil, fmt.Errorf("breakout not eligible: %s", elig.RejectReason)
	}

	atr := input.ATR
	entryDistance := atr.Mul(decimal.NewFromFloat(cfg.EntryATRMultiplier))
	slDistance := atr.Mul(decimal.NewFromFloat(cfg.SLATRMultiplier))
	tpDistance := atr.Mul(decimal.NewFromFloat(cfg.TPATRMultiplier))

	// Calculate position size using money-at-risk (reuses capital_protection logic)
	riskAmount := input.Equity.Mul(decimal.NewFromFloat(cfg.MaxRiskPct / 100.0))
	pointValue := input.TickValue.Div(input.TickSize)
	volume := decimal.Zero
	if !slDistance.IsZero() && !pointValue.IsZero() {
		volume = riskAmount.Div(slDistance.Mul(pointValue))
	}

	// Round volume to lot step (simplified — real rounding uses broker lot step)
	if !volume.IsZero() {
		volume = volume.Round(2) // 0.01 lot step
		if volume.LessThan(decimal.NewFromFloat(0.01)) {
			volume = decimal.NewFromFloat(0.01) // minimum lot
		}
	}

	now := time.Now().UTC()
	plan := &BreakoutPlan{
		PlanID:        planID,
		EventID:       input.Event.EventID,
		Symbol:        "XAUUSD",
		Strategy:      "NEWS_BREAKOUT",
		CreatedAt:     now,
		EventTime:     input.Event.ScheduledAtUTC,
		BuyStopEntry:  input.CurrentPrice.Add(entryDistance),
		SellStopEntry: input.CurrentPrice.Sub(entryDistance),
		BuyStopSL:     input.CurrentPrice.Add(entryDistance).Sub(slDistance),
		SellStopSL:    input.CurrentPrice.Sub(entryDistance).Add(slDistance),
		BuyStopTP:     input.CurrentPrice.Add(entryDistance).Add(tpDistance),
		SellStopTP:    input.CurrentPrice.Sub(entryDistance).Sub(tpDistance),
		Volume:        volume,
		RiskPct:       cfg.MaxRiskPct,
		Expiry:        input.Event.ScheduledAtUTC.Add(time.Duration(cfg.ExpirySeconds) * time.Second),
		OcoGroupID:    ocoGroupID,
		Status:        StatusCreated,
	}
	return plan, nil
}

// Engine manages breakout plan lifecycle.
type Engine struct {
	mu     sync.RWMutex
	cfg    Config
	plans  map[string]*BreakoutPlan
}

// NewEngine creates a new breakout engine.
func NewEngine(cfg Config) *Engine {
	return &Engine{
		cfg:   cfg,
		plans: make(map[string]*BreakoutPlan),
	}
}

// RegisterPlan adds a plan to the engine.
func (e *Engine) RegisterPlan(plan *BreakoutPlan) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.plans[plan.PlanID] = plan
}

// GetPlan retrieves a plan by ID.
func (e *Engine) GetPlan(planID string) (*BreakoutPlan, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	plan, ok := e.plans[planID]
	return plan, ok
}

// ExpirePlans marks plans past their expiry time as EXPIRED.
func (e *Engine) ExpirePlans(now time.Time) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var expired []string
	for id, plan := range e.plans {
		if plan.Status == StatusArmed && now.After(plan.Expiry) {
			plan.Status = StatusExpired
			expired = append(expired, id)
		}
	}
	return expired
}

// ActivePlans returns all plans in ARMED status.
func (e *Engine) ActivePlans() []*BreakoutPlan {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var result []*BreakoutPlan
	for _, p := range e.plans {
		if p.Status == StatusArmed || p.Status == StatusCreated || p.Status == StatusValidated {
			result = append(result, p)
		}
	}
	return result
}

// IsEnabled returns true if breakout is enabled in config.
func (e *Engine) IsEnabled() bool {
	return e.cfg.Enabled
}
