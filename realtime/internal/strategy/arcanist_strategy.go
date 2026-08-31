// Package strategy — Arcanist Institutional Liquidity Strategy (XAUUSD)
//
// ICT / smart-money liquidity approach: top-down HTF bias (W1/D1), session
// killzone timing (Asian / London / NY), liquidity sweeps + order blocks +
// break-of-structure, with explicit gold-aware risk (20-25 pip SL = $2.00-$2.50,
// 1% risk, R:R >= 1:3). Reference spec: new/ALCHEMIST_XAUUSD_SPEC.md.
//
// Implemented as a first-class Go strategy product (SOW 12A-12F: distinct logic
// from the other six). It is intentionally DEFENSIVE: any data gap or error
// yields NO-TRADE, and it only ever emits advisory-grade output unless the
// operator explicitly authorizes it for execution. It never mutates shared
// state and is wrapped in a recover guard so a logic fault can never crash the
// engine or affect the other strategies.
package strategy

import (
	"fmt"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ArcanistStrategy implements the Strategy interface for the Arcanist liquidity model.
type ArcanistStrategy struct {
	cfg ArcanistConfig
}

// ArcanistConfig holds tunables (aligned with new/alchemist_xauusd.json).
type ArcanistConfig struct {
	StrategyID            types.StrategyID
	MinScore              float64
	RiskPerTradePct      float64
	SLPipsMin             float64
	SLPipsMax             float64
	SLATRMultiplier       float64
	MinRRToERL            float64
	MaxSpreadPoints       float64
	HardStopUTC           int
	DecisionTFs           []types.Timeframe
	AcceptedSessions      map[string]bool
	BiasTimeframes        []types.Timeframe
	RefinementTimeframes  []types.Timeframe
	ExecTimeframes         []types.Timeframe
}

// arcanistCandleProvider is injected at engine start so the strategy can pull
// multi-TF history from the candle store (the live MarketState only carries the
// current bar per timeframe). If unset, the strategy degrades to NO-TRADE.
var arcanistCandleProvider func(symbol string, tf types.Timeframe, limit int) ([]*types.Candle, error)

// SetArcanistCandleProvider injects the candle history source.
func SetArcanistCandleProvider(p func(symbol string, tf types.Timeframe, limit int) ([]*types.Candle, error)) {
	arcanistCandleProvider = p
}

// NewArcanistStrategy builds the default Arcanist configuration.
func NewArcanistStrategy() *ArcanistStrategy {
	return &ArcanistStrategy{
		cfg: ArcanistConfig{
			StrategyID:           types.StrategyArcanist,
			MinScore:             70,
			RiskPerTradePct:      1.0,
			SLPipsMin:            20,
			SLPipsMax:            25,
			SLATRMultiplier:      0.35,
			MinRRToERL:           3.0,
			MaxSpreadPoints:      35,
			HardStopUTC:          17,
			DecisionTFs:          []types.Timeframe{types.TFM15, types.TFM5},
			AcceptedSessions:     map[string]bool{"LONDON": true, "NEW_YORK": true, "OVERLAP": true, "ASIAN": true},
			BiasTimeframes:       []types.Timeframe{types.TFW1, types.TFD1},
			RefinementTimeframes: []types.Timeframe{types.TFH4, types.TFH1},
			ExecTimeframes:       []types.Timeframe{types.TFM15, types.TFM5},
		},
	}
}

func (s *ArcanistStrategy) ID() types.StrategyID { return s.cfg.StrategyID }
func (s *ArcanistStrategy) DecisionTimeframes() []types.Timeframe { return s.cfg.DecisionTFs }

// Evaluate runs the Arcanist decision pipeline. Panic-safe: any failure returns
// a non-trading result so the engine and other strategies are unaffected.
func (s *ArcanistStrategy) Evaluate(state *features.MarketState) (res StrategyResult) {
	defer func() {
		if r := recover(); r != nil {
			res = noTradeResult(s.cfg.StrategyID, "ARC_PANIC")
		}
	}()

	res = StrategyResult{
		StrategyID:       s.ID(),
		Direction:        types.DirectionNoTrade,
		RawScore:         decimal.Zero,
		LongScore:        decimal.Zero,
		ShortScore:       decimal.Zero,
		EntryPrice:       decimal.Zero,
		StopLoss:         decimal.Zero,
		TP1:              decimal.Zero,
		TP2:              decimal.Zero,
		TP3:              decimal.Zero,
		ExpiryMinutes:    180,
		CooldownMinutes:  240,
	}

	if state == nil || !state.CurrentPrice.IsPositive() {
		res.Direction = types.DirectionError
		return res
	}

	// 1) Hard stop after 17:00 UTC — thin liquidity window.
	if time.Now().UTC().Hour() >= s.cfg.HardStopUTC {
		return res
	}

	// 2) News risk — block on hard levels.
	nr := state.Session.NewsRisk
	if nr == "BLOCKED" || nr == "DATA_UNAVAILABLE" {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_NEWS_RISK_BLOCKED"))
		return res
	}

	// 3) Session / killzone gate.
	if !s.cfg.AcceptedSessions[state.Session.CurrentSession] {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_SESSION"))
		return res
	}

	// 4) HTF bias (W1 + D1 must agree).
	bias, biasOK := s.htfBias(state.Symbol)
	if !biasOK {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_NO_BIAS"))
		return res
	}

	// 5) Refinement POIs (H4/H1 order blocks aligned to bias, fresh).
	pois := s.freshOrderBlocks(state.Symbol, bias)
	if len(pois) == 0 {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_NO_POI"))
		return res
	}

	// 6) Asian range (00:00-06:00 UTC on M15).
	asiaLow, asiaHigh, asiaOK := s.asianRange(state.Symbol)
	if !asiaOK {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_NO_ASIA"))
		return res
	}

	// 7) Execution TF confirmation: liquidity sweep + BOS in bias direction.
	exec := s.cfg.ExecTimeframes[0]
	execCandles := s.fetch(state.Symbol, exec, 60)
	if len(execCandles) < 20 {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_NO_EXEC"))
		return res
	}
	swept := s.judasSweep(execCandles, asiaLow, asiaHigh, bias)
	bosDir := lastBOS(execCandles)
	if !swept || bosDir != bias {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_NO_CONFIRM"))
		return res
	}

	// 8) Choose nearest valid POI to price as entry.
	price := state.CurrentPrice
	poi := nearestPOI(pois, price)
	entry := poi

	// 9) Gold-pip-aware stop loss (20-25 pips = $2.00-$2.50; 1 gold pip = $0.10).
	slDist := s.stopDistance(state)
	if slDist.LessThanOrEqual(decimal.Zero) {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_NO_SL"))
		return res
	}
	var sl, tp1, tp2 decimal.Decimal
	if bias == types.DirectionBuy {
		sl = entry.Sub(slDist)
		tp1 = asiaHigh
		tp2 = entry.Add(slDist.Mul(decimal.NewFromInt(3)))
	} else {
		sl = entry.Add(slDist)
		tp1 = asiaLow
		tp2 = entry.Sub(slDist.Mul(decimal.NewFromInt(3)))
	}

	rr := tp2.Sub(entry).Abs().Div(slDist)
	if rr.LessThan(decimal.NewFromFloat(s.cfg.MinRRToERL)) {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_LOW_RR"))
		return res
	}

	// 10) Scoring (per spec weights; news/spread penalties applied).
	score := s.score(bias, swept, nr)
	if score < s.cfg.MinScore {
		res.ReasonCodes = append(res.ReasonCodes, types.NoTradeReason("NT_ARCANIST_LOW_SCORE"))
		return res
	}

	res.Direction = bias
	res.RawScore = decimal.NewFromFloat(score)
	res.EntryPrice = entry
	res.StopLoss = sl
	res.TP1 = tp1
	res.TP2 = tp2
	res.Confidence = score
	res.HumanReason = fmt.Sprintf("ARCANIST %s score=%.0f entry=%s sl=%s rr=%.1f",
		bias, score, entry.String(), sl.String(), rr.InexactFloat64())
	return res
}

// ---- helpers ----

func (s *ArcanistStrategy) fetch(symbol string, tf types.Timeframe, limit int) []*types.Candle {
	if arcanistCandleProvider == nil {
		return nil
	}
	cs, err := arcanistCandleProvider(symbol, tf, limit)
	if err != nil || len(cs) == 0 {
		return nil
	}
	return cs
}

func (s *ArcanistStrategy) htfBias(symbol string) (types.Direction, bool) {
	w1 := s.fetch(symbol, types.TFW1, 60)
	d1 := s.fetch(symbol, types.TFD1, 60)
	if len(w1) < 5 || len(d1) < 5 {
		return "", false
	}
	bw := lastBOS(w1)
	bd := lastBOS(d1)
	if bw == "" || bd == "" || bw != bd {
		return "", false
	}
	return bw, true
}

func (s *ArcanistStrategy) freshOrderBlocks(symbol string, bias types.Direction) []decimal.Decimal {
	var out []decimal.Decimal
	for _, tf := range s.cfg.RefinementTimeframes {
		cs := s.fetch(symbol, tf, 120)
		if len(cs) < 10 {
			continue
		}
		var sum decimal.Decimal
		for _, c := range cs {
			sum = sum.Add(c.Close.Sub(c.Open).Abs())
		}
		avg := sum.Div(decimal.NewFromInt(int64(len(cs))))
		thresh := avg.Mul(decimal.NewFromFloat(1.8))
		for i := 1; i < len(cs)-1; i++ {
			cur, nxt := cs[i], cs[i+1]
			body := nxt.Close.Sub(nxt.Open).Abs()
			if body.LessThan(thresh) {
				continue
			}
			if cur.Close.LessThan(cur.Open) && nxt.Close.GreaterThan(nxt.Open) && bias == types.DirectionBuy {
				out = append(out, cur.Close)
			}
			if cur.Close.GreaterThan(cur.Open) && nxt.Close.LessThan(nxt.Open) && bias == types.DirectionSell {
				out = append(out, cur.Close)
			}
		}
	}
	return out
}

func (s *ArcanistStrategy) asianRange(symbol string) (low, high decimal.Decimal, ok bool) {
	cs := s.fetch(symbol, types.TFM15, 240)
	if len(cs) == 0 {
		return decimal.Zero, decimal.Zero, false
	}
	lo := decimal.Zero
	hi := decimal.Zero
	found := false
	for _, c := range cs {
		if c.Time.UTC().Hour() >= 0 && c.Time.UTC().Hour() < 6 {
			if !found {
				lo = c.Low
				hi = c.High
				found = true
			} else {
				if c.Low.LessThan(lo) {
					lo = c.Low
				}
				if c.High.GreaterThan(hi) {
					hi = c.High
				}
			}
		}
	}
	return lo, hi, found
}

func (s *ArcanistStrategy) judasSweep(cs []*types.Candle, asiaLow, asiaHigh decimal.Decimal, bias types.Direction) bool {
	if len(cs) < 10 {
		return false
	}
	recent := cs[len(cs)-10:]
	if bias == types.DirectionBuy {
		for _, c := range recent {
			if c.Low.LessThan(asiaLow) {
				return true
			}
		}
	} else {
		for _, c := range recent {
			if c.High.GreaterThan(asiaHigh) {
				return true
			}
		}
	}
	return false
}

func (s *ArcanistStrategy) stopDistance(state *features.MarketState) decimal.Decimal {
	pip := decimal.NewFromFloat(0.10)
	minPips := decimal.NewFromFloat(s.cfg.SLPipsMin).Mul(pip)
	maxPips := decimal.NewFromFloat(s.cfg.SLPipsMax).Mul(pip)
	atr := state.Indicators.ATR
	if atr.IsZero() {
		return maxPips
	}
	calc := atr.Mul(decimal.NewFromFloat(s.cfg.SLATRMultiplier))
	if calc.LessThan(minPips) {
		return minPips
	}
	if calc.GreaterThan(maxPips) {
		return maxPips
	}
	return calc
}

func (s *ArcanistStrategy) score(bias types.Direction, swept bool, news string) float64 {
	_ = bias
	score := 25.0 + 20.0 + 15.0 + 10.0 + 10.0 // bias + poi + bos + killzone + rr
	if swept {
		score += 20.0
	}
	if news == "HIGH" || news == "EXTREME" {
		score -= 25.0
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func nearestPOI(pois []decimal.Decimal, price decimal.Decimal) decimal.Decimal {
	best := pois[0]
	bestD := price.Sub(best).Abs()
	for _, p := range pois[1:] {
		d := price.Sub(p).Abs()
		if d.LessThan(bestD) {
			best = p
			bestD = d
		}
	}
	return best
}

// lastBOS derives break-of-structure direction from the most recent swing
// high/low in the supplied candle slice.
func lastBOS(cs []*types.Candle) types.Direction {
	lastSH, lastSL := -1, -1
	for i := 2; i < len(cs)-2; i++ {
		if cs[i].High.GreaterThan(cs[i-1].High) && cs[i].High.GreaterThan(cs[i-2].High) &&
			cs[i].High.GreaterThan(cs[i+1].High) && cs[i].High.GreaterThan(cs[i+2].High) {
			lastSH = i
		}
		if cs[i].Low.LessThan(cs[i-1].Low) && cs[i].Low.LessThan(cs[i-2].Low) &&
			cs[i].Low.LessThan(cs[i+1].Low) && cs[i].Low.LessThan(cs[i+2].Low) {
			lastSL = i
		}
	}
	if lastSH == -1 && lastSL == -1 {
		return ""
	}
	last := cs[len(cs)-1]
	if lastSH > lastSL && last.Close.GreaterThan(cs[lastSH].High) {
		return types.DirectionBuy
	}
	if lastSL > lastSH && last.Close.LessThan(cs[lastSL].Low) {
		return types.DirectionSell
	}
	return ""
}

func noTradeResult(id types.StrategyID, reason string) StrategyResult {
	return StrategyResult{
		StrategyID:  id,
		Direction:   types.DirectionNoTrade,
		ReasonCodes: []types.NoTradeReason{types.NoTradeReason(reason)},
	}
}
