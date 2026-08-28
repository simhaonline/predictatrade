// Package broker encodes per-account broker execution policy. This is the new
// core logic called out by the product research: many brokers forbid scalping
// or impose a minimum trade duration / stop distance. The policy decides which
// strategies are even ELIGIBLE for a given account before any signal is built.
package broker

import "time"

// BrokerPolicy describes what an account's broker permits, plus the execution
// economics used for cost-aware sizing and net R:R.
type BrokerPolicy struct {
	Symbol              string
	AllowsScalping      bool    // false => scalping strategies are excluded
	MinTradeDurationSec int     // broker-imposed minimum hold time
	StopLevelPoints     float64 // broker StopsLevel (points)
	FreezeLevelPoints   float64 // broker FreezeLevel (points)
	Digits              int     // price digits (2 for XAUUSD)
	MinNetRR            float64 // hard floor on NET R:R after costs (0 = block only if unprofitable)
	Execution           ExecutionProfile
}

// IsScalpingStrategy reports whether a strategy ID is a scalping product.
func IsScalpingStrategy(id string) bool {
	return id == "ULTRA_SCALPING" || id == "STANDARD_SCALPING"
}

// StrategyAllowed reports whether the broker policy permits running a strategy.
// Returns (true, "") when allowed, or (false, reason) when excluded.
func (p *BrokerPolicy) StrategyAllowed(id string) (bool, string) {
	if IsScalpingStrategy(id) && !p.AllowsScalping {
		return false, "BROKER_SCALPING_NOT_ALLOWED"
	}
	if p.MinTradeDurationSec > 0 && IsScalpingStrategy(id) {
		// A scalping signal with a short expiry cannot honour the broker's
		// minimum hold time without being force-closed; exclude it.
		return false, "BROKER_MIN_HOLD_EXCEEDS_SIGNAL_LIFETIME"
	}
	return true, ""
}

// Session classifies the current time into a trading session using BROKER server
// time (profile timezone offset). This is the single place session/overlap logic
// lives so signal generation is always in broker time, never local time.
func (p *BrokerPolicy) Session(now time.Time) (name string, overlap bool) {
	prof := p.Execution
	bt := now.In(time.FixedZone("broker", prof.TimezoneOffset*3600))
	h := bt.Hour()
	for _, s := range prof.Sessions {
		if h >= s.StartH && h < s.EndH {
			return s.Name, s.Overlap
		}
	}
	// wrap-around (e.g., 23:00 in TOKYO/SYDNEY window)
	if len(prof.Sessions) > 0 {
		s := prof.Sessions[0]
		if h >= s.StartH {
			return s.Name, s.Overlap
		}
	}
	return "UNKNOWN", false
}

// IsOverlap reports whether now is inside the London/NY overlap window.
func (p *BrokerPolicy) IsOverlap(now time.Time) bool {
	_, o := p.Session(now)
	return o
}
