package strategy

import (
	"pat-engine/internal/config"
	"pat-engine/internal/types"
)

// TrendSwing is the longest-horizon product (H1/H4 decision, D1/W1 context).
// Fewer signals, wider stops, larger targets, longer lifecycle. Only operates in
// trending/breakout regimes and never against the EMA100/EMA200 macro trend.
type TrendSwing struct {
	cfg config.StrategyConfig
}

func NewTrendSwing(cfg config.StrategyConfig) *TrendSwing {
	return &TrendSwing{cfg: cfg}
}

func (s *TrendSwing) ID() types.StrategyID { return types.StrategyTrendSwing }

func (s *TrendSwing) DecisionTimeframes() []types.Timeframe {
	tfs := make([]types.Timeframe, len(s.cfg.DecisionTFs))
	for i, t := range s.cfg.DecisionTFs {
		tfs[i] = types.Timeframe(t)
	}
	return tfs
}

func (s *TrendSwing) Evaluate(state *types.MarketState) StrategyResult {
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
	if state.ATR > 0 && (state.Spread/state.ATR) > s.cfg.SpreadATRGate {
		res.ReasonCodes = append(res.ReasonCodes, "HIGH_SPREAD")
		return res
	}

	// Trend Swing requires a genuine trend — weak ADX is an immediate NO-TRADE.
	if state.Indicators.ADX <= s.cfg.MinADX {
		res.ReasonCodes = append(res.ReasonCodes, "ADX_TOO_LOW")
		return res
	}
	// Never trade against the macro EMA100/EMA200 trend when both are present.
	if state.Indicators.EMA100 != 0 && state.Indicators.EMA200 != 0 {
		macroBull := state.Indicators.EMA100 > state.Indicators.EMA200
		macroBear := state.Indicators.EMA100 < state.Indicators.EMA200
		if !macroBull && !macroBear {
			res.ReasonCodes = append(res.ReasonCodes, "MACRO_TREND_UNCLEAR")
			return res
		}
	}

	var ev []contrib
	if state.Indicators.SMA200 != 0 {
		if state.CurrentPrice > state.Indicators.SMA200 {
			ev = append(ev, contrib{types.DirBuy, 0.18})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.18})
		}
	}
	if state.Indicators.EMA50 != 0 {
		if state.CurrentPrice > state.Indicators.EMA50 {
			ev = append(ev, contrib{types.DirBuy, 0.10})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.10})
		}
	}
	if state.Indicators.EMA21 > state.Indicators.EMA50 {
		ev = append(ev, contrib{types.DirBuy, 0.08})
	} else {
		ev = append(ev, contrib{types.DirSell, 0.08})
	}
	if state.Indicators.ADXPlusDI > state.Indicators.ADXMinusDI {
		ev = append(ev, contrib{types.DirBuy, 0.12})
	} else {
		ev = append(ev, contrib{types.DirSell, 0.12})
	}
	if state.Structure.LastBOS != nil {
		if state.Structure.LastBOS.Direction == "bullish" {
			ev = append(ev, contrib{types.DirBuy, 0.10})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.10})
		}
	}
	if state.Indicators.MACDMain > state.Indicators.MACDSignal {
		ev = append(ev, contrib{types.DirBuy, 0.06})
	} else if state.Indicators.MACDMain < state.Indicators.MACDSignal {
		ev = append(ev, contrib{types.DirSell, 0.06})
	}
	if state.Indicators.CCI > 0 {
		ev = append(ev, contrib{types.DirBuy, 0.05})
	} else if state.Indicators.CCI < 0 {
		ev = append(ev, contrib{types.DirSell, 0.05})
	}
	if state.MTFScore > 0 {
		ev = append(ev, contrib{types.DirBuy, 0.07})
	} else if state.MTFScore < 0 {
		ev = append(ev, contrib{types.DirSell, 0.07})
	}
	if state.VWAP > 0 {
		if state.CurrentPrice > state.VWAP {
			ev = append(ev, contrib{types.DirBuy, 0.05})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.05})
		}
	}
	if state.Indicators.ParabolicSAR != 0 {
		if state.Indicators.ParabolicSARLong {
			ev = append(ev, contrib{types.DirBuy, 0.05})
		} else {
			ev = append(ev, contrib{types.DirSell, 0.05})
		}
	}

	dir, raw, long, short, reasons := scoreFromEvidence(ev, s.cfg.MinConfluence, state)
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
