package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

func registerAllGates(r *Registry) {
	r.Register(&DataQualityGate{})
	r.Register(&SessionGate{})
	r.Register(&NewsGate{})
	r.Register(&SpreadGate{MaxSpreadAbsolute: 0.35, MaxSpreadToATR: 0.5})
	r.Register(&SlippageGate{MaxSlippage: 0.10})
	r.Register(&TotalCostGate{MaxCostToTarget: 0.25})
	r.Register(&ExposureGate{MaxExposure: 5})
	r.Register(&MarginGate{})
	r.Register(&RRNetExpectancyGate{MinGrossRR: 1.20})
	r.Register(&ProfitabilityGate{})
	r.Register(&EntitlementGate{})
	r.Register(&LicenseGate{})
	r.Register(&ExecutionPermissionGate{})
	// Self-evaluating precision gates (in canonical order; evaluated from
	// live GateInput, seeded PASS like production main.go B-04).
	r.Register(&MinAbsoluteATRGate{MinATR: 2.0})
	r.Register(&StopHuntFilterGate{MinDistanceATR: 1.5})
}

func setAllGateStatesPass(r *Registry) {
	now := time.Now()
	validUntil := now.Add(time.Minute)
	for _, gid := range []types.GateID{
		types.GateDataQuality, types.GateSession, types.GateNews,
		types.GateSpread, types.GateSlippage, types.GateTotalCost,
		types.GateExposure, types.GateMargin, types.GateRRNetExpectancy,
		types.GateEntitlement, types.GateLicense, types.GateExecutionPermit,
		types.GateMinATR, types.GateStopHuntFilter,
	} {
		val := any(true)
		if gid == types.GateSlippage {
			val = any(0.05) // low slippage
		}
		r.UpdateState(gid, GateState{
			GateID:        gid,
			State:         types.GatePass,
			Value:         val,
			EvaluatedAt:   now,
			ValidUntil:    validUntil,
			SourceVersion: "v1",
		})
	}
}

func TestGateAllPass(t *testing.T) {
	registry := NewRegistry()
	registerAllGates(registry)
	setAllGateStatesPass(registry)

	input := GateInput{
		Tick: &types.Tick{
			Symbol:  "XAUUSD",
			Quality: types.QualityAuthoritative,
		},
		SessionAllowed:     true,
		NewsRisk:           "LOW",
		Spread:             0.20,
		ATR:                3.0,
		EntryPrice:         2430,
		StopLoss:           2426,
		TakeProfit1:        2435,
		RoundTripCost:      0.50,
		CurrentExposure:    1,
		MaxExposure:        5,
		EntitlementOK:      true,
		LicenseActive:      true,
		ExecutionPermitted: true,
	}

	allPass, evals, veto := registry.EvaluateAll(input)
	if !allPass {
		t.Error("Expected all gates to pass")
	}
	if veto != nil {
		t.Errorf("Expected no veto, got veto at gate %s", veto.GateID)
	}
	if len(evals) < 12 {
		t.Errorf("Expected 12 evaluations, got %d", len(evals))
	}
}

func TestGateVetoStopsEvaluation(t *testing.T) {
	registry := NewRegistry()
	registerAllGates(registry)
	setAllGateStatesPass(registry)

	// Override news to veto
	registry.UpdateState(types.GateNews, GateState{
		GateID:      types.GateNews,
		State:       types.GatePass,
		EvaluatedAt: time.Now(),
		ValidUntil:  time.Now().Add(time.Minute),
	})

	input := GateInput{
		Tick:           &types.Tick{Quality: types.QualityAuthoritative},
		SessionAllowed: true,
		NewsRisk:       "HIGH", // This should trigger news veto
	}

	allPass, _, veto := registry.EvaluateAll(input)
	if allPass {
		t.Error("Expected gates to NOT all pass")
	}
	if veto == nil {
		t.Error("Expected a veto")
	}
	if veto.GateID != types.GateNews {
		t.Errorf("Expected veto at news gate, got %s", veto.GateID)
	}
}

func TestGateMissingStateFailsClosed(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&DataQualityGate{})

	input := GateInput{}
	allPass, evals, veto := registry.EvaluateAll(input)
	if allPass {
		t.Error("Expected fail-closed when no state")
	}
	if veto == nil {
		t.Error("Expected a veto when state missing")
	}
	if evals[0].Result != types.GateUnknown {
		t.Errorf("Expected UNKNOWN result, got %s", evals[0].Result)
	}
}

func TestSpreadGate(t *testing.T) {
	gate := &SpreadGate{MaxSpreadAbsolute: 0.35, MaxSpreadToATR: 0.5}
	state := GateState{State: types.GatePass}

	// Within limits
	input := GateInput{Spread: 0.20, ATR: 3.0}
	eval := gate.Evaluate(input, state)
	if eval.Result != types.GatePass {
		t.Errorf("Expected PASS, got %s", eval.Result)
	}

	// Exceeds absolute
	input.Spread = 0.40
	eval = gate.Evaluate(input, state)
	if eval.Result != types.GateVeto {
		t.Errorf("Expected VETO for high spread, got %s", eval.Result)
	}

	// Exceeds spread/ATR
	input.Spread = 0.20
	input.ATR = 0.30
	eval = gate.Evaluate(input, state)
	if eval.Result != types.GateVeto {
		t.Errorf("Expected VETO for high spread/ATR, got %s", eval.Result)
	}
}

func TestStaleGateState(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&DataQualityGate{})

	registry.UpdateState(types.GateDataQuality, GateState{
		GateID:      types.GateDataQuality,
		State:       types.GatePass,
		EvaluatedAt: time.Now().Add(-time.Hour),
		ValidUntil:  time.Now().Add(-time.Minute),
	})

	input := GateInput{
		Tick: &types.Tick{Quality: types.QualityAuthoritative},
	}

	allPass, evals, _ := registry.EvaluateAll(input)
	if allPass {
		t.Error("Expected fail for stale state")
	}
	if evals[0].Result != types.GateDegraded {
		t.Errorf("Expected DEGRADED for stale state, got %s", evals[0].Result)
	}
}
