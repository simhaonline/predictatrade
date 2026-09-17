// regime_direction_gate.go — P1 (prompt.md): RegimeAllows + squeeze-veto as
// a proper ordered hard gate.
//
// NON-OVERLAP: checkRegimeSession() (strategies.go) already vetoes when a
// strategy's AcceptedRegimes don't include the current regime — that is the
// per-strategy soft/config gate. THIS gate is different, complementary, and
// fail-closed-by-design: it enforces the REFERENCE direction rule
// (trending regimes admit only the aligned direction; compressed/squeeze
// regimes admit nothing) regardless of any strategy's own list. It runs
// ordered, after Session, before Spread — per-(strategy,timeframe) isolated
// state like every other gate.
//
// Gate: types.GateRegimeDirection ("regime_direction")
// Veto reason codes: regime_direction_mismatch, regime_squeeze_veto
package gates

import (
	"github.com/predictatrade/realtime/internal/types"
)

// Machine-readable reasons for the regime-direction gate.
const (
	ReasonRegimeDirMismatch = "regime_direction_mismatch"
	ReasonRegimeSqueezeVeto = "regime_squeeze_veto"
)

// GateRegimeDirectionID — new GateID for the ordered chain.
const GateRegimeDirectionID types.GateID = "regime_direction"

// RegimeDirectionGate implements RegimeAllows() (prompt.md P1):
//   - TRENDING_BULLISH vetoes SELL
//   - TRENDING_BEARISH vetoes BUY
//   - RANGE vetoes BOTH (reference squeeze veto — no trades while compressed)
//   - all other regimes pass (the strategy's own AcceptedRegimes check still
//     applies via checkRegimeSession)
type RegimeDirectionGate struct{}

func (g *RegimeDirectionGate) ID() types.GateID { return GateRegimeDirectionID }

func (g *RegimeDirectionGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := baseEval(g.ID(), state)
	if input.Direction != types.DirectionBuy && input.Direction != types.DirectionSell {
		// Non-directional evaluation — nothing to gate.
		eval.Result = types.GatePass
		return eval
	}

	switch input.Regime {
	case types.RegimeTrendingBullish:
		if input.Direction == types.DirectionSell {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{ReasonRegimeDirMismatch}
			return eval
		}
	case types.RegimeTrendingBearish:
		if input.Direction == types.DirectionBuy {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{ReasonRegimeDirMismatch}
			return eval
		}
	case types.RegimeRange:
		// Squeeze veto: compressed market — no directional trades (fail closed).
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{ReasonRegimeSqueezeVeto}
		return eval
	}
	eval.Result = types.GatePass
	return eval
}