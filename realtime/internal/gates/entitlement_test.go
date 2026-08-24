package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// P1-001 / P2-003: ResolveEntitlementState must fail closed when state is missing.
func TestResolveEntitlementStateFailClosedWhenNotInitialized(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	state := ResolveEntitlementState(reg)
	if state.EntitlementOK {
		t.Error("EntitlementOK must be false when gate is NOT_INITIALIZED")
	}
	if state.LicenseActive {
		t.Error("LicenseActive must be false when gate is NOT_INITIALIZED")
	}
	if state.ExecutionPermitted {
		t.Error("ExecutionPermitted must be false when gate is NOT_INITIALIZED")
	}
	if len(state.DenialReasons) == 0 {
		t.Error("Expected denial reasons when gates are not initialized")
	}
}

// P1-001: ResolveEntitlementState must pass only when all three gates are fresh PASS.
func TestResolveEntitlementStatePassWhenAllVerified(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	now := time.Now().UTC()
	// Set all three safety-critical gates to fresh PASS
	reg.UpdateState(types.GateEntitlement, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
		SourceVersion: "control_plane",
	})
	reg.UpdateState(types.GateLicense, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
		SourceVersion: "control_plane",
	})
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
		SourceVersion: "system_config",
	})

	state := ResolveEntitlementState(reg)
	if !state.EntitlementOK {
		t.Error("EntitlementOK should be true when gate is fresh PASS")
	}
	if !state.LicenseActive {
		t.Error("LicenseActive should be true when gate is fresh PASS")
	}
	if !state.ExecutionPermitted {
		t.Error("ExecutionPermitted should be true when gate is fresh PASS")
	}
}

// P2-003: Stale gate state must fail closed.
func TestResolveEntitlementStateStaleFailsClosed(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	// Set gates to PASS but with expired validity
	past := time.Now().UTC().Add(-10 * time.Minute)
	reg.UpdateState(types.GateEntitlement, GateState{
		State: types.GatePass, EvaluatedAt: past, ValidUntil: past.Add(1 * time.Minute),
	})
	reg.UpdateState(types.GateLicense, GateState{
		State: types.GatePass, EvaluatedAt: past, ValidUntil: past.Add(1 * time.Minute),
	})
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: past, ValidUntil: past.Add(1 * time.Minute),
	})

	state := ResolveEntitlementState(reg)
	if state.EntitlementOK {
		t.Error("Stale entitlement gate must fail closed")
	}
	if state.LicenseActive {
		t.Error("Stale license gate must fail closed")
	}
	if state.ExecutionPermitted {
		t.Error("Stale execution permit must fail closed")
	}
}

// P2-003: VETO state must fail closed.
func TestResolveEntitlementStateVetoFailsClosed(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	now := time.Now().UTC()
	reg.UpdateState(types.GateEntitlement, GateState{
		State: types.GateVeto, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})
	reg.UpdateState(types.GateLicense, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})

	state := ResolveEntitlementState(reg)
	if state.EntitlementOK {
		t.Error("VETO entitlement gate must fail closed")
	}
	// License and execution should still pass (they are PASS)
	if !state.LicenseActive {
		t.Error("License should still pass")
	}
	if !state.ExecutionPermitted {
		t.Error("Execution should still pass")
	}
}

// P2-003: SeedConservativeGateStates must seed safety-critical gates as UNKNOWN.
func TestSeedConservativeGateStatesSafetyCriticalAreUnknown(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	safetyCritical := []types.GateID{
		types.GateExposure, types.GateMargin,
		types.GateEntitlement, types.GateLicense, types.GateExecutionPermit,
	}
	for _, gid := range safetyCritical {
		state, exists := reg.GetState(gid)
		if !exists {
			t.Errorf("Gate %s should have seeded state", gid)
			continue
		}
		if state.State != types.GateUnknown {
			t.Errorf("Gate %s should be UNKNOWN after conservative seed, got %s", gid, state.State)
		}
		// ValidUntil should be in the past (already expired) → forces fail-closed
		if !time.Now().After(state.ValidUntil) {
			t.Errorf("Gate %s should have expired validity after conservative seed", gid)
		}
	}
}

// P2-003: SeedConservativeGateStates must seed market-data gates as PASS (they use live input).
func TestSeedConservativeGateStatesMarketDataArePass(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	marketDataGates := []types.GateID{
		types.GateDataQuality, types.GateSession, types.GateNews,
		types.GateSpread, types.GateSlippage, types.GateTotalCost,
		types.GateRRNetExpectancy,
	}
	for _, gid := range marketDataGates {
		state, exists := reg.GetState(gid)
		if !exists {
			t.Errorf("Gate %s should have seeded state", gid)
			continue
		}
		if state.State != types.GatePass {
			t.Errorf("Market-data gate %s should be PASS after seed, got %s", gid, state.State)
		}
	}
}

// P2-003: After conservative seed, a full gate evaluation must fail (safety-critical gates are UNKNOWN).
func TestEvaluateAllFailsAfterConservativeSeed(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	input := GateInput{
		Tick:           &types.Tick{Quality: types.QualityAuthoritative},
		SessionAllowed: true, NewsRisk: "LOW",
		Spread: 0.20, ATR: 3.0,
		EntryPrice: 2430, StopLoss: 2426, TakeProfit1: 2435,
		RoundTripCost: 0.30, CurrentExposure: 0, MaxExposure: 5,
		EntitlementOK: true, LicenseActive: true, ExecutionPermitted: true,
	}

	allPass, evals, veto := reg.EvaluateAll(input)
	if allPass {
		t.Error("EvaluateAll must NOT pass after conservative seed — safety-critical gates are UNKNOWN")
	}
	if veto == nil {
		t.Error("Expected a veto from uninitialized safety-critical gates")
	}

	// The veto should come from a safety-critical gate (exposure is first in order)
	safetyCritical := map[types.GateID]bool{
		types.GateExposure: true, types.GateMargin: true,
		types.GateEntitlement: true, types.GateLicense: true,
		types.GateExecutionPermit: true,
	}
	if !safetyCritical[veto.GateID] {
		t.Errorf("Expected veto from safety-critical gate, got %s", veto.GateID)
	}

	// Verify at least one eval is UNKNOWN
	foundUnknown := false
	for _, e := range evals {
		if e.Result == types.GateUnknown || e.Result == types.GateVeto {
			foundUnknown = true
			break
		}
	}
	if !foundUnknown {
		t.Error("Expected at least one UNKNOWN/VETO result in evaluations")
	}
}

// P2-003: After hydrating broker + entitlement state, EvaluateAll should pass.
func TestEvaluateAllPassesAfterFullHydration(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	now := time.Now().UTC()
	fresh := now.Add(60 * time.Second)

	// Hydrate all safety-critical gates
	reg.UpdateState(types.GateExposure, GateState{
		State: types.GatePass, Value: float64(0), EvaluatedAt: now, ValidUntil: fresh,
		SourceVersion: "broker",
	})
	reg.UpdateState(types.GateMargin, GateState{
		State: types.GatePass, Value: true, EvaluatedAt: now, ValidUntil: fresh,
		SourceVersion: "broker",
	})
	reg.UpdateState(types.GateEntitlement, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: fresh, SourceVersion: "control_plane",
	})
	reg.UpdateState(types.GateLicense, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: fresh, SourceVersion: "control_plane",
	})
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: fresh, SourceVersion: "system_config",
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
		t.Error("Expected all gates to pass after full hydration")
	}
	if veto != nil {
		t.Errorf("Expected no veto after full hydration, got %s", veto.GateID)
	}
}

// P2-003: Restart scenario — fresh registry starts fail-closed, no transient allow.
func TestRestartDoesNotTransitionallyAllowTrading(t *testing.T) {
	// Simulate restart: new registry, conservative seed
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	// Immediately check entitlement state — must be denied
	state := ResolveEntitlementState(reg)
	if state.EntitlementOK || state.LicenseActive || state.ExecutionPermitted {
		t.Error("After restart, no trading should be allowed until state is re-verified")
	}
}

// P2-003: Reconnect scenario — re-hydration after reconnect allows trading.
func TestReconnectReVerification(t *testing.T) {
	reg := NewRegistry()
	registerAllGates(reg)
	SeedConservativeGateStates(reg)

	// Initially denied
	state := ResolveEntitlementState(reg)
	if state.EntitlementOK {
		t.Error("Initially should be denied")
	}

	// Simulate reconnect — state arrives
	now := time.Now().UTC()
	reg.UpdateState(types.GateEntitlement, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})
	reg.UpdateState(types.GateLicense, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})
	reg.UpdateState(types.GateExecutionPermit, GateState{
		State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
	})

	// Now allowed
	state = ResolveEntitlementState(reg)
	if !state.EntitlementOK || !state.LicenseActive || !state.ExecutionPermitted {
		t.Error("After re-verification, trading should be allowed")
	}
}
