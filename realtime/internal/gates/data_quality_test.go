package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func liveTick() *types.Tick {
	return &types.Tick{
		Symbol:           "XAUUSD",
		Bid:              mustDec(2412.1),
		Ask:              mustDec(2412.4),
		Source:           "MT5",
		SourceTimestamp:  time.Now().UTC(),
		GatewayTimestamp: time.Now().UTC(),
		Quality:          types.QualityAuthoritative,
	}
}

func mustDec(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

// MARKET_CLOSED regression (2026-08-29): the Master Node emits liveness-only
// snapshots during closed market carrying the last-known price. Those must
// NEVER allow EXECUTABLE evaluation — data-quality gate fails closed.
func TestDataQualityGateMarketClosed(t *testing.T) {
	g := &DataQualityGate{}
	state := GateState{State: types.GatePass}

	t.Run("live tick passes", func(t *testing.T) {
		eval := g.Evaluate(GateInput{Tick: liveTick()}, state)
		if eval.Result != types.GatePass {
			t.Fatalf("live tick should pass, got %s", eval.Result)
		}
	})

	t.Run("market_closed liveness tick vetoes", func(t *testing.T) {
		tick := liveTick()
		tick.MarketClosed = true
		eval := g.Evaluate(GateInput{Tick: tick}, state)
		if eval.Result != types.GateVeto {
			t.Fatalf("market_closed tick must VETO, got %s", eval.Result)
		}
		if len(eval.ReasonCodes) == 0 || eval.ReasonCodes[0] != "MARKET_CLOSED_LIVENESS_DATA" {
			t.Fatalf("expected MARKET_CLOSED_LIVENESS_DATA reason, got %v", eval.ReasonCodes)
		}
	})
}

// mustDec helper may already exist in another test file — guard via the
// existing testutil if compile complains. (kept local for isolation)
