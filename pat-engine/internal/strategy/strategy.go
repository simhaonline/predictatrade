// Package strategy defines the strategy interface, shared result type, and the
// single-source SL/TP geometry used by every strategy product.
package strategy

import (
	"pat-engine/internal/config"
	"pat-engine/internal/types"
)

// StrategyResult is the output of a strategy evaluation.
type StrategyResult struct {
	StrategyID      types.StrategyID
	Direction       types.Direction
	RawScore        float64
	LongScore       float64
	ShortScore      float64
	EntryPrice      float64
	StopLoss        float64
	TP1, TP2, TP3   float64
	Reason          string
	ReasonCodes     []string
	ExpiryMinutes   int
	CooldownMinutes int
}

// Strategy is implemented by each distinct strategy product.
type Strategy interface {
	ID() types.StrategyID
	DecisionTimeframes() []types.Timeframe
	Evaluate(state *types.MarketState) StrategyResult
}

// computeEntrySLTP builds the trade geometry from the strategy config (the
// single source of truth). It applies VolatilityScale and guarantees the SL
// buffer dominates transaction cost via MinSLSpreadMult.
func computeEntrySLTP(state *types.MarketState, dir types.Direction, cfg config.StrategyConfig) (entry, sl, tp1, tp2, tp3 float64) {
	entry = state.CurrentPrice
	atr := state.ATR
	if cfg.VolatilityScale > 0 {
		atr = atr * cfg.VolatilityScale
	}
	if cfg.MinSLSpreadMult > 0 && state.Spread > 0 && cfg.ATRMultiplierSL > 0 {
		required := state.Spread * (cfg.MinSLSpreadMult / cfg.ATRMultiplierSL)
		if required > atr {
			atr = required
		}
	}
	slDist := atr * cfg.ATRMultiplierSL
	if cfg.MinSLATRFloor > 0 {
		floor := atr * cfg.MinSLATRFloor
		if floor > slDist {
			slDist = floor
		}
	}
	tp1d := atr * cfg.ATRMultiplierTP1
	tp2d := atr * cfg.ATRMultiplierTP2
	tp3d := atr * cfg.ATRMultiplierTP3
	if dir == types.DirBuy {
		sl = entry - slDist
		tp1 = entry + tp1d
		tp2 = entry + tp2d
		tp3 = entry + tp3d
	} else {
		sl = entry + slDist
		tp1 = entry - tp1d
		tp2 = entry - tp2d
		tp3 = entry - tp3d
	}
	return
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// scoreFromEvidence accumulates directional evidence, scales to 0-100, applies the
// confluence threshold, and enforces the mandatory H1 trend veto. It returns the
// resulting direction, raw/long/short scores, and any reason codes. This is shared
// by every strategy so scoring stays identical across products.
func scoreFromEvidence(ev []contrib, minConf float64, state *types.MarketState, cfg config.StrategyConfig) (types.Direction, float64, float64, float64, []string) {
	long, short := 0.0, 0.0
	for _, e := range ev {
		if e.dir == types.DirBuy {
			long += e.c
		} else {
			short += e.c
		}
	}
	long *= 100
	short *= 100

	var dir types.Direction
	var raw float64
	var reasons []string
	if long > short {
		raw = long
		if long > minConf {
			dir = types.DirBuy
		} else {
			dir = types.DirNoTrade
			reasons = append(reasons, "INSUFFICIENT_SCORE")
		}
	} else {
		raw = short
		if short > minConf {
			dir = types.DirSell
		} else {
			dir = types.DirNoTrade
			reasons = append(reasons, "INSUFFICIENT_SCORE")
		}
	}

	// H1 trend veto — higher-timeframe alignment is mandatory.
	if dir == types.DirBuy && state.CurrentPrice < state.H1Close {
		dir = types.DirNoTrade
		reasons = append(reasons, "HTF_BEARISH_VETO")
	}
	if dir == types.DirSell && state.CurrentPrice > state.H1Close {
		dir = types.DirNoTrade
		reasons = append(reasons, "HTF_BULLISH_VETO")
	}

	// Structural trigger (primary edge): only take a trade when a liquidity sweep
	// lines up with the trade direction — i.e. we FADE the liquidity grab in the
	// direction of the higher-timeframe bias (sell-side sweep -> BUY, buy-side
	// sweep -> SELL). This is the playbook's highest-expectancy XAUUSD setup;
	// trading without a sweep present is exactly what produced the coin-flip
	// win rate, so it is a hard requirement, not a soft confirmation.
	if bias := structuralBias(state); bias == types.DirNoTrade || bias != dir {
		dir = types.DirNoTrade
		reasons = append(reasons, "NO_STRUCTURAL_TRIGGER")
	}

	// Higher-timeframe alignment (EMA200 vs EMA400 proxy). Trade only with the
	// higher-timeframe bias — the dominant edge filter from the external scalping
	// research (trade with the map, not against it).
	if dir == types.DirBuy && state.HTFBias != types.Bullish {
		dir = types.DirNoTrade
		reasons = append(reasons, "HTF_BIAS_BEARISH")
	}
	if dir == types.DirSell && state.HTFBias != types.Bearish {
		dir = types.DirNoTrade
		reasons = append(reasons, "HTF_BIAS_BULLISH")
	}

	// Volatility-regime gate: skip dead/whippy low-volatility stretches where the
	// edge collapses (research: ATR < 0.8x its longer baseline = whipsaw land).
	if state.SlowATR > 0 && state.ATR < 0.8*state.SlowATR {
		dir = types.DirNoTrade
		reasons = append(reasons, "LOW_VOL_REGIME")
	}

	// RSI extreme filter (read.md §"Deliberately left off the chart"): RSI is NOT a
	// directional vote on gold — an OB/OS print at a liquidity sweep is exactly
	// where the sweep-&-reclaim reversal fires. Block entries INTO the extreme only
	// (validated on the Python backtest: +win-rate, +profit factor).
	if dir == types.DirBuy && state.Indicators.RSI > 72 {
		dir = types.DirNoTrade
		reasons = append(reasons, "RSI_OVERBOUGHT")
	}
	if dir == types.DirSell && state.Indicators.RSI < 28 {
		dir = types.DirNoTrade
		reasons = append(reasons, "RSI_OVERSOLD")
	}

	// Spread NO-GO: a blown spread (> 2x the session baseline) destroys the edge
	// (playbook §2: widened-spread = flat rule). Round-turn cost balloons and the
	// break-even win rate becomes unreachable.
	if cfg.SpreadMedianBaseline > 0 && state.Spread > 2*cfg.SpreadMedianBaseline {
		dir = types.DirNoTrade
		reasons = append(reasons, "SPREAD_BLOWN")
	}

	// Target must be large enough to beat transaction cost (playbook §1 consequence
	// #1: sub-$1 targets are structurally negative-expectancy). TP1 is the 1R partial.
	// Compared against the configured typical spread (the broker cost baseline), not
	// the live per-bar spread — the live spread is handled by the SPREAD_BLOWN gate.
	if cfg.MinGrossCostMult > 0 && cfg.SpreadMedianBaseline > 0 {
		scaledATR := state.ATR
		if cfg.VolatilityScale > 0 {
			scaledATR = state.ATR * cfg.VolatilityScale
		}
		tp1Gross := scaledATR * cfg.ATRMultiplierTP1
		if tp1Gross < cfg.SpreadMedianBaseline*cfg.MinGrossCostMult {
			dir = types.DirNoTrade
			reasons = append(reasons, "TARGET_TOO_SMALL_VS_COST")
		}
	}

	// Prime-window selectivity: the edge concentrates in two UTC windows (playbook §3:
	// London open 09:00-11:00 and London/NY overlap 14:00-17:00). Only enforced for
	// strategies that opt in (scalpers); swing products leave it empty.
	if len(cfg.PrimeWindowsUTC) > 0 {
		inWindow := false
		for _, w := range cfg.PrimeWindowsUTC {
			if state.UTCHour >= w[0] && state.UTCHour < w[1] {
				inWindow = true
				break
			}
		}
		if !inWindow {
			dir = types.DirNoTrade
			reasons = append(reasons, "OUTSIDE_PRIME_WINDOW")
		}
	}

	return dir, raw, long, short, reasons
}

// structuralBias returns a trade direction from the playbook's primary edge:
// the liquidity sweep & reclaim. A "sweep" is a wick that takes the sell-side or
// buy-side liquidity pool and then CLOSES BACK INSIDE it (the reclaim is already
// encoded in the sweep detection). The highest-expectancy gold scalp is to FADE
// that liquidity grab:
//   sell-side sweep taken  -> BUY  (fade the stop-hunt, expect continuation up)
//   buy-side  sweep taken  -> SELL (fade the stop-hunt, expect continuation down)
// This is deliberately the single, well-defined structural trigger — not "any BOS",
// which floods the book with low-quality breakout entries.
func structuralBias(s *types.MarketState) types.Direction {
	if s == nil {
		return types.DirNoTrade
	}
	var sweptSell, sweptBuy bool
	for _, sw := range s.Liquidity.RecentSweeps {
		switch sw.Direction {
		case "SELL_SIDE":
			sweptSell = true
		case "BUY_SIDE":
			sweptBuy = true
		}
	}
	if sweptSell {
		return types.DirBuy
	}
	if sweptBuy {
		return types.DirSell
	}
	return types.DirNoTrade
}
