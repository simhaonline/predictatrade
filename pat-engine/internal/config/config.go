// Package config holds strategy configuration. This is the SINGLE source of
// truth for SL/TP/RR per strategy — the backtest and the live engine MUST load
// the same values so the live execution matches the validated research.
package config

// StrategyConfig is the tunable profile for one strategy product.
type StrategyConfig struct {
	StrategyID        string
	MinConfluence     float64
	MinMTFAlignment   float64
	ATRMultiplierSL   float64
	ATRMultiplierTP1  float64
	ATRMultiplierTP2  float64
	ATRMultiplierTP3  float64
	MinSLATRFloor     float64
	VolatilityScale   float64 // 1.0 = use real ATR as-is (no understated-feed hack)
	MinSLSpreadMult   float64 // SL distance must dominate this multiple of spread
	SpreadATRGate     float64 // max spread/ATR ratio; above this => NO-TRADE
	SpreadMedianBaseline float64 // typical session spread (price units); >2x => NO-GO (playbook §2)
	MaxSpreadPips     float64
	MaxSlippagePoints int
	MinADX            float64
	MinRR             float64       // hard floor on reward:risk (gross, measured to TP2)
	PrimeWindowsUTC   [][2]int     // UTC [startH,endH) windows where the edge lives; empty = no restriction
	MinGrossCostMult  float64      // TP1 gross must be >= spread * this, else target too small to beat cost
	ExpiryMinutes     int
	CooldownMinutes   int
	DecisionTFs       []string
	AcceptedRegimes   []string
	AcceptedSessions  []string
	MinQualityState   string
	BacktestMaxBars   int // realistic trade horizon (bars) for the replay simulator
}

// primeWindows returns the high-probability XAUUSD scalping sessions from the
// external playbook (§3): London open through the London/NY overlap
// (07:00-17:00 UTC). ~80% of a scalper's edge concentrates here; trading the
// dead Tokyo/Sydney hours is excluded.
func primeWindows() [][2]int {
	return [][2]int{{7, 17}}
}

// DefaultUltraScalping returns the v1 ULTRA_SCALPING profile.
// SL = 1.0 ATR, TP1 = 1.0 ATR (close 50% + move stop to BE), TP2 = 2.0 ATR,
// TP3 = 3.0 ATR (trailed remainder). Implements the playbook exit ("half at 1R ->
// BE, rest to 2R/3R") so a trade is a WIN once price reaches 1R.
func DefaultUltraScalping() StrategyConfig {
	return StrategyConfig{
		StrategyID:           "ULTRA_SCALPING",
		MinConfluence:        0,
		MinMTFAlignment:      50,
		ATRMultiplierSL:   4.0,
		ATRMultiplierTP1:  4.0,
		ATRMultiplierTP2:  8.0,
		ATRMultiplierTP3:  12.0,
		MinSLATRFloor:        0.0,
		VolatilityScale:      1.0,
		MinSLSpreadMult:      3.0,
		SpreadATRGate:        0.4,
		SpreadMedianBaseline: 0.30,
		MaxSpreadPips:        1.5,
		MaxSlippagePoints:    5,
		MinADX:               25,
		MinRR:                2.0,
		PrimeWindowsUTC:      primeWindows(),
		MinGrossCostMult:     3.0,
		ExpiryMinutes:        5,
		CooldownMinutes:      5,
		DecisionTFs:          []string{"M1"},
		AcceptedRegimes:      []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "MEAN_REVERSION", "RANGE", "HIGH_VOLATILITY"},
		AcceptedSessions:     []string{"LONDON", "OVERLAP", "NEW_YORK"},
		MinQualityState:      "AUTHORITATIVE",
		BacktestMaxBars:      24,
	}
}

// DefaultStandardScalping returns the STANDARD_SCALPING profile (M1/M5 decision,
// M15/M30 context). SL=1.5 ATR, TP1=1.5 ATR (1R partial), TP2=3.0 ATR (2R), TP3=4.5 ATR (3R).
func DefaultStandardScalping() StrategyConfig {
	return StrategyConfig{
		StrategyID:           "STANDARD_SCALPING",
		MinConfluence:        0,
		MinMTFAlignment:      40,
		ATRMultiplierSL:   3.0,
		ATRMultiplierTP1:  3.0,
		ATRMultiplierTP2:  6.0,
		ATRMultiplierTP3:  9.0,
		MinSLATRFloor:        0.0,
		VolatilityScale:      1.0,
		MinSLSpreadMult:      3.0,
		SpreadATRGate:        0.5,
		SpreadMedianBaseline: 0.35,
		MaxSpreadPips:        2.5,
		MaxSlippagePoints:    10,
		MinADX:               25,
		MinRR:                2.0,
		PrimeWindowsUTC:      primeWindows(),
		MinGrossCostMult:     3.0,
		ExpiryMinutes:        10,
		CooldownMinutes:      15,
		DecisionTFs:          []string{"M1", "M5"},
		AcceptedRegimes:      []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "MEAN_REVERSION", "RANGE", "HIGH_VOLATILITY"},
		AcceptedSessions:     []string{"LONDON", "OVERLAP", "NEW_YORK"},
		MinQualityState:      "AUTHORITATIVE",
		BacktestMaxBars:      40,
	}
}

// DefaultStandardSwing returns the STANDARD_SWING profile (M15/H1 decision, H4/D1
// context). SL=2.0 ATR, TP1=2.0 ATR (1R partial), TP2=4.0 ATR (2R), TP3=6.0 ATR (3R).
// Swing trades are not restricted to the scalping prime windows.
func DefaultStandardSwing() StrategyConfig {
	return StrategyConfig{
		StrategyID:           "STANDARD_SWING",
		MinConfluence:        0,
		MinMTFAlignment:      30,
		ATRMultiplierSL:      2.0,
		ATRMultiplierTP1:     2.0,
		ATRMultiplierTP2:     4.0,
		ATRMultiplierTP3:     6.0,
		MinSLATRFloor:        0.0,
		VolatilityScale:      1.0,
		MinSLSpreadMult:      3.0,
		SpreadATRGate:        0.6,
		SpreadMedianBaseline: 0.50,
		MaxSpreadPips:        4.0,
		MaxSlippagePoints:    20,
		MinADX:               20,
		MinRR:                2.0,
		PrimeWindowsUTC:      nil,
		MinGrossCostMult:     3.0,
		ExpiryMinutes:        60,
		CooldownMinutes:      120,
		DecisionTFs:          []string{"M15", "M30", "H1"},
		AcceptedRegimes:      []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "MEAN_REVERSION", "RANGE", "HIGH_VOLATILITY"},
		AcceptedSessions:     []string{"LONDON", "OVERLAP", "NEW_YORK"},
		MinQualityState:      "AUTHORITATIVE",
		BacktestMaxBars:      160,
	}
}

// DefaultTrendSwing returns the TREND_SWING profile (H1/H4 decision, D1/W1
// context). SL=2.5 ATR, TP1=2.5 ATR (1R partial), TP2=5.0 ATR (2R), TP3=7.5 ATR (3R).
func DefaultTrendSwing() StrategyConfig {
	return StrategyConfig{
		StrategyID:           "TREND_SWING",
		MinConfluence:        0,
		MinMTFAlignment:      25,
		ATRMultiplierSL:   5.0,
		ATRMultiplierTP1:  5.0,
		ATRMultiplierTP2:  10.0,
		ATRMultiplierTP3:  15.0,
		MinSLATRFloor:        0.0,
		VolatilityScale:      1.0,
		MinSLSpreadMult:      3.0,
		SpreadATRGate:        0.8,
		SpreadMedianBaseline: 0.60,
		MaxSpreadPips:        5.0,
		MaxSlippagePoints:    30,
		MinADX:               20,
		MinRR:                2.0,
		PrimeWindowsUTC:      nil,
		MinGrossCostMult:     3.0,
		ExpiryMinutes:        240,
		CooldownMinutes:      360,
		DecisionTFs:          []string{"H1", "H4"},
		AcceptedRegimes:      []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "HIGH_VOLATILITY"},
		AcceptedSessions:     []string{"LONDON", "OVERLAP", "NEW_YORK"},
		MinQualityState:      "AUTHORITATIVE",
		BacktestMaxBars:      240,
	}
}

// AllDefaults returns every strategy config (single source of truth for backtest
// and live). Add a strategy here and it is instantly available to the engine.
func AllDefaults() map[string]StrategyConfig {
	return map[string]StrategyConfig{
		"ULTRA_SCALPING":     DefaultUltraScalping(),
		"STANDARD_SCALPING":  DefaultStandardScalping(),
		"STANDARD_SWING":     DefaultStandardSwing(),
		"TREND_SWING":        DefaultTrendSwing(),
	}
}
