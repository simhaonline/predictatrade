// Package risk implements capital-loss control: position sizing from broker economics,
// per-trade risk %, daily-loss halt bands, position caps and margin/leverage checks.
// Ported conceptually from the old realtime capital_gates / sizing, but self-contained.
package risk

import (
	"sync"
	"time"

	"pat-engine/internal/broker"
)

// RiskProfile is the trader/account risk mandate. Static config (per user/plan).
type RiskProfile struct {
	Equity           float64 `json:"equity"`
	FreeMargin       float64 `json:"free_margin"`
	RiskPerTradePct  float64 `json:"risk_per_trade_pct"`
	MaxDailyLossPct  float64 `json:"max_daily_loss_pct"`
	MaxPositions     int     `json:"max_positions"`
	MaxLeverage      float64 `json:"max_leverage"`
	MinRR            float64 `json:"min_rr"`
}

// DefaultRisk returns a conservative default (1% risk, 2% daily halt, 1.5x RR).
func DefaultRisk() RiskProfile {
	return RiskProfile{
		Equity:          10000,
		FreeMargin:      10000,
		RiskPerTradePct: 1.0,
		MaxDailyLossPct: 2.0,
		MaxPositions:    5,
		MaxLeverage:     -1, // -1 => no override; broker profile wins
		MinRR:           1.5,
	}
}

// PositionSize computes the lot size for a stop distance (price units) using the
// broker execution economics. P&L per lot for a price move d is d * ContractSize.
// Returns 0 if it cannot size safely.
func (r RiskProfile) PositionSize(stopDistancePrice float64, exec broker.ExecutionProfile) float64 {
	if stopDistancePrice <= 0 || exec.ContractSize <= 0 {
		return 0
	}
	riskAmount := r.Equity * (r.RiskPerTradePct / 100)
	lots := riskAmount / (stopDistancePrice * exec.ContractSize)
	// Broker affordability (margin) cap.
	if exec.Leverage > 0 && r.FreeMargin > 0 {
		affordable := exec.MaxAffordableLot(r.FreeMargin, stopDistancePrice+exec.RoundToDigits(0))
		if affordable > 0 && lots > affordable {
			lots = affordable
		}
	}
	lots = exec.RoundLot(lots)
	if lots < exec.MinLot {
		return 0
	}
	return lots
}

// RequiredMargin for a sized lot at price.
func (r RiskProfile) RequiredMargin(lot, price float64, exec broker.ExecutionProfile) float64 {
	return exec.RequiredMargin(lot, price)
}

// DailyLoss tracks per-day realized+unrealized loss for halt decisions. In-memory;
// in production this is hydrated from the DB equity/positions ledger.
type DailyLoss struct {
	mu       sync.Mutex
	day      string
	loss     float64
	count    int
	halted   bool
}

// Update records a P&L tick and re-anchors on UTC day rollover.
func (d *DailyLoss) Update(pnl float64, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	day := now.UTC().Format("2006-01-02")
	if day != d.day {
		d.day, d.loss, d.count, d.halted = day, 0, 0, false
	}
	d.loss += pnl
	if pnl < 0 {
		d.count++
	}
}

// Breached reports whether the daily loss cap is exceeded (and latches a halt).
func (d *DailyLoss) Breached(maxDailyLossPct, equity float64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if equity <= 0 {
		return false
	}
	if (d.loss/equity)*100 <= -maxDailyLossPct {
		d.halted = true
		return true
	}
	return false
}

// Halted reports the latched halt state.
func (d *DailyLoss) Halted() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.halted
}

// EvaluateOpenRisk returns a veto reason if open risk breaches mandates, else "".
func EvaluateOpenRisk(openPositions, maxPositions int, leverage, maxLeverage float64) string {
	if maxPositions > 0 && openPositions >= maxPositions {
		return "MAX_POSITIONS_REACHED"
	}
	if maxLeverage > 0 && leverage > maxLeverage {
		return "LEVERAGE_EXCEEDS_PLAN"
	}
	return ""
}
