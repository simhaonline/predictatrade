package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// P1-001: Verify the full gate hydration lifecycle from agent connection → broker account data.
// This simulates what happens when a Windows Agent connects and sends a MARKET_SNAPSHOT.
func TestFullGateHydrationLifecycleFromAgent(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	// Stage 1: After conservative seed — all safety-critical gates fail closed
	state := ResolveEntitlementState(reg)
	if state.ExecutionPermitted {
		t.Error("Execution should be denied after conservative seed")
	}

	exposureState, _ := reg.GetState(types.GateExposure)
	if exposureState.State == types.GatePass {
		t.Error("Exposure gate should be UNKNOWN after seed")
	}

	// Stage 2: Agent connects (TICK/HEARTBEAT received) — execution permit hydrated
	now := time.Now().UTC()
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State:         types.GatePass,
		EvaluatedAt:   now,
		ValidUntil:    now.Add(60 * time.Second),
		SourceVersion: "agent_connection",
		ReasonCode:    "terminal_connected",
	})

	state = ResolveEntitlementState(reg)
	if !state.ExecutionPermitted {
		t.Error("Execution should be permitted after agent connection")
	}
	// But entitlement and license still denied (not hydrated yet)
	if state.EntitlementOK {
		t.Error("Entitlement should still be denied — not hydrated from control plane yet")
	}

	// Stage 3: Agent sends MARKET_SNAPSHOT with account_info — exposure/margin hydrated
	reg.UpdateState(types.GateExposure, GateState{
		State:         types.GatePass,
		Value:         float64(0), // 0 open positions
		EvaluatedAt:   now,
		ValidUntil:    now.Add(30 * time.Second),
		SourceVersion: "broker_telemetry",
	})
	reg.UpdateState(types.GateMargin, GateState{
		State:         types.GatePass,
		Value:         true, // free margin > 0
		EvaluatedAt:   now,
		ValidUntil:    now.Add(30 * time.Second),
		SourceVersion: "broker_telemetry",
	})

	// Now a full gate evaluation with entitlement/license set should pass
	reg.UpdateState(types.GateEntitlement, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})
	reg.UpdateState(types.GateLicense, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})

	input := GateInput{
		Tick:           &types.Tick{Quality: types.QualityAuthoritative},
		SessionAllowed: true, NewsRisk: "LOW",
		Spread: 0.20, ATR: 3.0,
		EntryPrice: 2430, StopLoss: 2426, TakeProfit1: 2435,
		RoundTripCost: 0.30, CurrentExposure: 0, MaxExposure: 5,
		EntitlementOK: true, LicenseActive: true, ExecutionPermitted: true,
	}

	allPass, _, veto := reg.EvaluateAll(input)
	if !allPass {
		t.Errorf("Expected all gates to pass after full hydration, veto at: %s", func() string {
			if veto != nil {
				return string(veto.GateID)
			}
			return "nil"
		}())
	}

	// Stage 4: Agent disconnects — gates expire (simulated by waiting past ValidUntil)
	expired := now.Add(-1 * time.Second)
	reg.UpdateState(types.GateExposure, GateState{
		State: types.GatePass, Value: float64(0),
		EvaluatedAt: expired, ValidUntil: expired,
	})
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: expired, ValidUntil: expired,
	})

	// After expiry, gates fail closed
	state = ResolveEntitlementState(reg)
	if state.ExecutionPermitted {
		t.Error("Execution should be denied after gate expiry (agent disconnected)")
	}
}

// P2-003: Verify that gates expire and fail closed when agent stops sending data.
func TestGateExpiryOnAgentDisconnect(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	// Hydrate execution permit from agent connection
	now := time.Now().UTC()
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(5 * time.Second),
	})

	// Immediately after hydration — execution permitted
	state := ResolveEntitlementState(reg)
	if !state.ExecutionPermitted {
		t.Error("Execution should be permitted immediately after hydration")
	}

	// Simulate 10 seconds passing (no more heartbeats)
	// The ValidUntil has expired — ResolveEntitlementState should fail closed
	expired := time.Now().UTC().Add(-10 * time.Second)
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: expired, ValidUntil: expired.Add(5 * time.Second),
	})

	state = ResolveEntitlementState(reg)
	if state.ExecutionPermitted {
		t.Error("Execution should be denied after gate expiry")
	}
	if len(state.DenialReasons) == 0 {
		t.Error("Should have denial reasons for expired gate")
	}
}
