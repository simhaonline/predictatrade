// Package gates implements the hard, fail-closed risk gate pipeline. A gate may
// only VETO; it never invents a trade. Every veto carries a stable ID so the
// downstream UI / EA can render an exact, machine-readable reason.
package gates

import (
	"strconv"

	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

// Veto is a single gate rejection.
type Veto struct {
	ID     string
	Reason string
}

func format(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }

func pow10(n int) int {
	p := 1
	for i := 0; i < n; i++ {
		p *= 10
	}
	return p
}

// EvaluateAll runs the gate set against a directional strategy result. It returns
// the list of vetoes (empty = pass). Order is irrelevant; any veto blocks.
func EvaluateAll(state *types.MarketState, res strategy.StrategyResult, cfg config.StrategyConfig, pol *broker.BrokerPolicy) []Veto {
	var vetoes []Veto
	if res.Direction != types.DirBuy && res.Direction != types.DirSell {
		return vetoes
	}

	var risk, reward float64
	if res.Direction == types.DirBuy {
		risk = res.EntryPrice - res.StopLoss
		reward = res.TP2 - res.EntryPrice // measured to the 2R target (TP1 is the 1R partial)
	} else {
		risk = res.StopLoss - res.EntryPrice
		reward = res.EntryPrice - res.TP2
	}

	// 1) R:R floor — the dominant cause of prior client stop-outs.
	if risk > 0 {
		rr := reward / risk
		if rr < cfg.MinRR {
			vetoes = append(vetoes, Veto{"RR_BELOW_MIN", "R:R " + format(rr) + " < MinRR " + format(cfg.MinRR)})
		}
	} else {
		vetoes = append(vetoes, Veto{"INVALID_SL", "SL on wrong side of entry"})
	}

	// 1b) NET R:R after TOTAL transaction cost (spread + commission + swap), in
	// PRICE units (same scale as the SL/TP distances). This is the authoritative
	// gate — gross R:R alone hides the cost that erodes edge. Cost uses the BROKER
	// execution profile (the real execution cost), matching the backtest simulator;
	// the per-bar data spread is only used by the SPREAD_BLOWN NO-GO gate.
	if pol != nil {
		exec := pol.Execution
		side := "BUY"
		if res.Direction == types.DirSell {
			side = "SELL"
		}
		holdDays := 1
		if cfg.ExpiryMinutes > 0 {
			holdDays = int(cfg.ExpiryMinutes / 1440)
			if holdDays < 1 {
				holdDays = 1
			}
		}
		costPrice := exec.TypicalSpread*2*exec.TickSize + // round-turn spread
			exec.CommissionPrice(1.0)*2 + // round-turn commission
			exec.SwapPrice(side, 1.0, holdDays)

		netRisk := risk + costPrice
		netReward := reward - costPrice
		if netReward <= 0 {
			vetoes = append(vetoes, Veto{"NET_RR_NEGATIVE", "trade unprofitable after spread+commission+swap"})
		} else if netRisk > 0 && pol.MinNetRR > 0 {
			netRR := netReward / netRisk
			if netRR < pol.MinNetRR {
				vetoes = append(vetoes, Veto{"NET_RR_BELOW_MIN", "net R:R " + format(netRR) + " < min net " + format(pol.MinNetRR)})
			}
		}
	}

	// 2) Broker stop-level compliance (points). SL distance must exceed the
	//    broker's StopsLevel.
	if pol != nil && pol.Digits > 0 && pol.StopLevelPoints > 0 {
		slPoints := risk * float64(pow10(pol.Digits))
		if slPoints < pol.StopLevelPoints {
			vetoes = append(vetoes, Veto{"BROKER_STOP_LEVEL", "SL " + format(slPoints) + "pts < broker min " + format(pol.StopLevelPoints) + "pts"})
		}
	}

	// 3) Broker freeze-level compliance (entry must clear the freeze zone).
	if pol != nil && pol.Digits > 0 && pol.FreezeLevelPoints > 0 {
		distToEntry := risk // conservative: use SL distance as proxy for execution clearance
		if distToEntry*float64(pow10(pol.Digits)) < pol.FreezeLevelPoints {
			vetoes = append(vetoes, Veto{"BROKER_FREEZE_LEVEL", "entry clearance < freeze level"})
		}
	}

	return vetoes
}
