package strategy

import (
	"pat-engine/internal/config"
	"pat-engine/internal/types"
)

// StandardScalping is M1/M5 scalping with M15/M30 context. Higher quality bar than
// Ultra, slightly wider stops. Ported faithfully from the existing engine.
type StandardScalping struct {
	cfg config.StrategyConfig
}

func NewStandardScalping(cfg config.StrategyConfig) *StandardScalping {
	return &StandardScalping{cfg: cfg}
}

func (s *StandardScalping) ID() types.StrategyID { return types.StrategyStandardScalping }

func (s *StandardScalping) DecisionTimeframes() []types.Timeframe {
	tfs := make([]types.Timeframe, len(s.cfg.DecisionTFs))
	for i, t := range s.cfg.DecisionTFs {
		tfs[i] = types.Timeframe(t)
	}
	return tfs
}

func (s *StandardScalping) Evaluate(state *types.MarketState) StrategyResult {
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

	var ev []contrib
	if state.Indicators.EMA9 > state.Indicators.EMA21 {
		ev = append(ev, contrib{types.DirBuy, 0.12})
	} else if state.Indicators.EMA9 < state.Indicators.EMA21 {
		ev = append(ev, contrib{types.DirSell, 0.12})
	}
	if state.VWAP > 0 {
		if state.CurrentPrice > state.VWAP {
			ev = append(ev, contrib{types.DirBuy, 0.08})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.08})
		}
	}
	if state.Structure.LastBOS != nil {
		if state.Structure.LastBOS.Direction == "bullish" {
			ev = append(ev, contrib{types.DirBuy, 0.14})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.14})
		}
	}
	if state.Candle.IsDisplacement && state.Candle.IsBullish {
		ev = append(ev, contrib{types.DirBuy, 0.10})
	}
	if state.Candle.IsDisplacement && state.Candle.IsBearish {
		ev = append(ev, contrib{types.DirSell, 0.10})
	}
	if state.Candle.IsRejection && state.Candle.IsBullish {
		ev = append(ev, contrib{types.DirBuy, 0.08})
	}
	if state.Candle.IsRejection && state.Candle.IsBearish {
		ev = append(ev, contrib{types.DirSell, 0.08})
	}
	if state.Indicators.MACDMain > state.Indicators.MACDSignal {
		ev = append(ev, contrib{types.DirBuy, 0.06})
	} else if state.Indicators.MACDMain < state.Indicators.MACDSignal {
		ev = append(ev, contrib{types.DirSell, 0.06})
	}
	if state.Indicators.OsMA > 0 {
		ev = append(ev, contrib{types.DirBuy, 0.05})
	} else if state.Indicators.OsMA < 0 {
		ev = append(ev, contrib{types.DirSell, 0.05})
	}
	rsi := state.Indicators.RSI
	if rsi > 50 && rsi < 70 {
		ev = append(ev, contrib{types.DirBuy, 0.05})
	} else if rsi < 50 && rsi > 30 {
		ev = append(ev, contrib{types.DirSell, 0.05})
	}
	if state.Indicators.ADX > s.cfg.MinADX {
		if state.Indicators.ADXPlusDI > state.Indicators.ADXMinusDI {
			ev = append(ev, contrib{types.DirBuy, 0.07})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.07})
		}
	}
	if len(state.Liquidity.RecentSweeps) > 0 {
		sw := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		if sw.Direction == "SELL_SIDE" {
			ev = append(ev, contrib{types.DirBuy, 0.08})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.08})
		}
	}
	if state.MTFScore > 0 {
		ev = append(ev, contrib{types.DirBuy, 0.05})
	} else if state.MTFScore < 0 {
		ev = append(ev, contrib{types.DirSell, 0.05})
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
