// Package broker encodes per-account broker execution policy. This is the new
// core logic called out by the product research: many brokers forbid scalping
// or impose a minimum trade duration / stop distance. The policy decides which
// strategies are even ELIGIBLE for a given account before any signal is built.
package broker

// BrokerPolicy describes what an account's broker permits.
type BrokerPolicy struct {
	Symbol              string
	AllowsScalping      bool    // false => scalping strategies are excluded
	MinTradeDurationSec int     // broker-imposed minimum hold time
	StopLevelPoints     float64 // broker StopsLevel (points)
	FreezeLevelPoints   float64 // broker FreezeLevel (points)
	Digits              int     // price digits (2 for XAUUSD)
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
