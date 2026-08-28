package strategy

import (
	"pat-engine/internal/config"
	"pat-engine/internal/types"
)

// UltraScalping is the v1 target strategy: fastest, M1 decision with M5/M15
// context, extremely cost-sensitive, very tight SL. Ported faithfully from the
// existing engine but trimmed to a clean, single-config implementation.
type UltraScalping struct {
	cfg config.StrategyConfig
}

func NewUltraScalping(cfg config.StrategyConfig) *UltraScalping {
	return &UltraScalping{cfg: cfg}
}

func (s *UltraScalping) ID() types.StrategyID { return types.StrategyUltraScalping }

func (s *UltraScalping) DecisionTimeframes() []types.Timeframe {
	tfs := make([]types.Timeframe, len(s.cfg.DecisionTFs))
	for i, t := range s.cfg.DecisionTFs {
		tfs[i] = types.Timeframe(t)
	}
	return tfs
}

type contrib struct {
	dir types.Direction
	c   float64
}

// Evaluate runs the ULTRA_SCALPING decision on a MarketState snapshot.
func (s *UltraScalping) Evaluate(state *types.MarketState) StrategyResult {
	res := StrategyResult{
		StrategyID:      s.ID(),
		Direction:       types.DirNoTrade,
		ExpiryMinutes:   s.cfg.ExpiryMinutes,
		CooldownMinutes: s.cfg.CooldownMinutes,
	}
	if state == nil || state.ATR == 0 {
		res.Direction = types.DirError
		res.ReasonCodes = append(res.ReasonCodes, "ATR_NOT_READY")
		return res
	}

	// Regime / session / news filters
	if !contains(s.cfg.AcceptedRegimes, state.Regime) {
		res.ReasonCodes = append(res.ReasonCodes, "REGIME_MISMATCH")
		return res
	}
	if !contains(s.cfg.AcceptedSessions, state.Session.CurrentSession) && !state.Session.IsOverlap {
		res.ReasonCodes = append(res.ReasonCodes, "SESSION_UNSUITABLE")
		return res
	}
	if state.Session.NewsRisk == "HIGH" || state.Session.NewsRisk == "BLOCKED" {
		res.ReasonCodes = append(res.ReasonCodes, "HIGH_NEWS_RISK")
		return res
	}

	// EMA hierarchy is used only as soft evidence below (not a hard veto): the
	// primary edge is the liquidity-sweep & reclaim, which by definition occurs
	// during a pullback (M1 EMA9<EMA21) while the HTF bias stays intact. A hard
	// EMA-alignment veto would reject exactly the pullback-fade entries we want.

	var ev []contrib
	if state.Indicators.EMA9 > state.Indicators.EMA21 {
		ev = append(ev, contrib{types.DirBuy, 0.15})
	} else if state.Indicators.EMA9 < state.Indicators.EMA21 {
		ev = append(ev, contrib{types.DirSell, 0.15})
	}
	if state.Candle.IsDisplacement && state.Candle.IsBullish {
		ev = append(ev, contrib{types.DirBuy, 0.15})
	}
	if state.Candle.IsDisplacement && state.Candle.IsBearish {
		ev = append(ev, contrib{types.DirSell, 0.15})
	}
	if state.VWAP > 0 {
		if state.CurrentPrice > state.VWAP {
			ev = append(ev, contrib{types.DirBuy, 0.10})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.10})
		}
	}
	if len(state.Liquidity.RecentSweeps) > 0 {
		sw := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		if sw.Direction == "SELL_SIDE" {
			ev = append(ev, contrib{types.DirBuy, 0.12})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.12})
		}
	}
	if state.Indicators.OsMA > 0 {
		ev = append(ev, contrib{types.DirBuy, 0.08})
	} else if state.Indicators.OsMA < 0 {
		ev = append(ev, contrib{types.DirSell, 0.08})
	}
	if state.Indicators.StochMain > state.Indicators.StochSignal {
		ev = append(ev, contrib{types.DirBuy, 0.06})
	} else {
		ev = append(ev, contrib{types.DirSell, 0.06})
	}
	if state.MTFScore > 0 {
		ev = append(ev, contrib{types.DirBuy, 0.08})
	} else if state.MTFScore < 0 {
		ev = append(ev, contrib{types.DirSell, 0.08})
	}
	if state.Indicators.ADX > s.cfg.MinADX {
		if state.Indicators.ADXPlusDI > state.Indicators.ADXMinusDI {
			ev = append(ev, contrib{types.DirBuy, 0.08})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.08})
		}
	}
	if state.Indicators.BollLower > 0 {
		if state.CurrentPrice <= state.Indicators.BollLower {
			ev = append(ev, contrib{types.DirBuy, 0.05})
		} else if state.CurrentPrice >= state.Indicators.BollUpper {
			ev = append(ev, contrib{types.DirSell, 0.05})
		}
	}

	dir, raw, long, short, reasons := scoreFromEvidence(ev, s.cfg.MinConfluence, state, s.cfg)
	res.ReasonCodes = append(res.ReasonCodes, reasons...)
	res.Direction = dir
	res.RawScore = raw
	res.LongScore = long
	res.ShortScore = short

	if dir == types.DirBuy || dir == types.DirSell {
		entry, sl, tp1, tp2, tp3 := computeEntrySLTP(state, dir, s.cfg)
		res.EntryPrice = entry
		res.StopLoss = sl
		res.TP1 = tp1
		res.TP2 = tp2
		res.TP3 = tp3
	}
	return res
}
