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
	MaxSpreadPips     float64
	MaxSlippagePoints int
	MinADX            float64
	MinRR             float64 // hard floor on reward:risk
	ExpiryMinutes     int
	CooldownMinutes   int
	DecisionTFs       []string
	AcceptedRegimes   []string
	AcceptedSessions  []string
	MinQualityState   string
}

// DefaultUltraScalping returns the v1 ULTRA_SCALPING profile.
// SL = 1.0 ATR, TP1 = 2.0 ATR => R:R = 2.0 (matches MinRR). This fixes the
// prior drift where the engine override produced a 1:1 R:R while the backtest
// assumed 2:1.
func DefaultUltraScalping() StrategyConfig {
	return StrategyConfig{
		StrategyID:        "ULTRA_SCALPING",
		MinConfluence:     65,
		MinMTFAlignment:   50,
		ATRMultiplierSL:   1.0,
		ATRMultiplierTP1:  2.0,
		ATRMultiplierTP2:  3.0,
		ATRMultiplierTP3:  5.0,
		MinSLATRFloor:     0.0,
		VolatilityScale:   1.0,
		MinSLSpreadMult:   3.0,
		SpreadATRGate:     0.4,
		MaxSpreadPips:     1.5,
		MaxSlippagePoints: 5,
		MinADX:            25,
		MinRR:             2.0,
		ExpiryMinutes:     5,
		CooldownMinutes:   5,
		DecisionTFs:       []string{"M1"},
		AcceptedRegimes:   []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "MEAN_REVERSION", "RANGE", "HIGH_VOLATILITY"},
		AcceptedSessions:  []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO", "SYDNEY"},
		MinQualityState:   "AUTHORITATIVE",
	}
}

// DefaultStandardScalping returns the STANDARD_SCALPING profile (M1/M5 decision,
// M15/M30 context). SL=1.5 ATR, TP1=2.5 ATR => R:R≈1.67; the RR gate enforces MinRR.
func DefaultStandardScalping() StrategyConfig {
	return StrategyConfig{
		StrategyID:        "STANDARD_SCALPING",
		MinConfluence:     65,
		MinMTFAlignment:   40,
		ATRMultiplierSL:   1.5,
		ATRMultiplierTP1:  2.5,
		ATRMultiplierTP2:  4.0,
		ATRMultiplierTP3:  6.0,
		MinSLATRFloor:     0.0,
		VolatilityScale:   1.0,
		MinSLSpreadMult:   3.0,
		SpreadATRGate:     0.5,
		MaxSpreadPips:     2.5,
		MaxSlippagePoints: 10,
		MinADX:            20,
		MinRR:             1.5,
		ExpiryMinutes:     10,
		CooldownMinutes:   15,
		DecisionTFs:       []string{"M1", "M5"},
		AcceptedRegimes:   []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "MEAN_REVERSION", "RANGE", "HIGH_VOLATILITY"},
		AcceptedSessions:  []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO", "SYDNEY"},
		MinQualityState:   "AUTHORITATIVE",
	}
}

// DefaultStandardSwing returns the STANDARD_SWING profile (M15/H1 decision, H4/D1
// context). SL=2.0 ATR, TP1=3.0 ATR => R:R=1.5; RR gate enforces MinRR.
func DefaultStandardSwing() StrategyConfig {
	return StrategyConfig{
		StrategyID:        "STANDARD_SWING",
		MinConfluence:     55,
		MinMTFAlignment:   30,
		ATRMultiplierSL:   2.0,
		ATRMultiplierTP1:  3.0,
		ATRMultiplierTP2:  5.0,
		ATRMultiplierTP3:  8.0,
		MinSLATRFloor:     0.0,
		VolatilityScale:   1.0,
		MinSLSpreadMult:   3.0,
		SpreadATRGate:     0.6,
		MaxSpreadPips:     4.0,
		MaxSlippagePoints: 20,
		MinADX:            20,
		MinRR:             1.5,
		ExpiryMinutes:     60,
		CooldownMinutes:   120,
		DecisionTFs:       []string{"M15", "M30", "H1"},
		AcceptedRegimes:   []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "MEAN_REVERSION", "RANGE", "HIGH_VOLATILITY"},
		AcceptedSessions:  []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO", "SYDNEY"},
		MinQualityState:   "AUTHORITATIVE",
	}
}

// DefaultTrendSwing returns the TREND_SWING profile (H1/H4 decision, D1/W1
// context). SL=2.5 ATR, TP1=4.0 ATR => R:R=1.6; RR gate enforces MinRR.
func DefaultTrendSwing() StrategyConfig {
	return StrategyConfig{
		StrategyID:        "TREND_SWING",
		MinConfluence:     50,
		MinMTFAlignment:   25,
		ATRMultiplierSL:   2.5,
		ATRMultiplierTP1:  4.0,
		ATRMultiplierTP2:  6.5,
		ATRMultiplierTP3:  10.0,
		MinSLATRFloor:     0.0,
		VolatilityScale:   1.0,
		MinSLSpreadMult:   3.0,
		SpreadATRGate:     0.8,
		MaxSpreadPips:     5.0,
		MaxSlippagePoints: 30,
		MinADX:            20,
		MinRR:             1.5,
		ExpiryMinutes:     240,
		CooldownMinutes:   360,
		DecisionTFs:       []string{"H1", "H4"},
		AcceptedRegimes:   []string{"TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "HIGH_VOLATILITY"},
		AcceptedSessions:  []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO", "SYDNEY"},
		MinQualityState:   "AUTHORITATIVE",
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
