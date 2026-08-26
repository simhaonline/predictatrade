// Package gates — Authoritative entitlement/license/execution state provider.
// P1-001 / P2-003: Replaces hardcoded `true` production gate values with
// state derived from the authoritative gate registry cached state.
//
// The signal hot path uses ONLY cached state — no synchronous external I/O.
// Background goroutines update the gate registry from authoritative sources
// (control plane DB, agent heartbeat, system configuration).
//
// Fail-closed: unknown/stale/missing state ALWAYS denies execution.
package gates

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// EntitlementState is the authoritative execution-eligibility snapshot
// derived from the gate registry. It is computed on every signal decision
// from cached gate state — never from a hardcoded default.
type EntitlementState struct {
	EntitlementOK      bool
	LicenseActive      bool
	ExecutionPermitted bool
	// DenialReasons records WHY each gate denied (for observability).
	DenialReasons []string
}

// ResolveEntitlementState derives the entitlement/license/execution-permit
// flags from the gate registry's cached state.
//
// A gate is considered positively verified ONLY when:
//   - cached state exists
//   - state is GatePass
//   - state is not stale (ValidUntil not exceeded)
//
// Any other condition (missing, unknown, veto, stale) → false (fail closed).
func ResolveEntitlementState(reg *Registry) EntitlementState {
	result := EntitlementState{}

	result.EntitlementOK = gatePass(reg, types.GateEntitlement, &result.DenialReasons)
	result.LicenseActive = gatePass(reg, types.GateLicense, &result.DenialReasons)
	result.ExecutionPermitted = gatePass(reg, types.GateExecutionPermit, &result.DenialReasons)

	return result
}

// gatePass returns true only when the gate has a fresh, positively-verified PASS state.
func gatePass(reg *Registry, gateID types.GateID, denialReasons *[]string) bool {
	state, exists := reg.GetState(gateID)
	if !exists {
		*denialReasons = append(*denialReasons, string(gateID)+"_NOT_INITIALIZED")
		return false
	}
	if state.State != types.GatePass {
		*denialReasons = append(*denialReasons, string(gateID)+"_"+string(state.State))
		return false
	}
	// Check freshness — stale state fails closed for safety-critical gates
	if !state.ValidUntil.IsZero() && time.Now().After(state.ValidUntil) {
		*denialReasons = append(*denialReasons, string(gateID)+"_STALE")
		return false
	}
	return true
}

// SeedConservativeGateStates initializes gate states to fail-closed defaults.
//
// P2-003: Production-sensitive gates must initialize conservatively.
// Market-data gates (data_quality, session, news, spread, slippage, total_cost,
// rr_net_expectancy) evaluate from live GateInput and may start as PASS because
// their actual evaluation uses the live tick/candle, not the cached state.
//
// Safety-critical gates (exposure, margin, entitlement, license, execution_permit)
// MUST start as UNKNOWN and fail closed until authoritative data arrives.
func SeedConservativeGateStates(reg *Registry) {
	now := time.Now().UTC()

	// Market-data gates — evaluated from live input; state marks freshness.
	// Short validity so they refresh quickly from the live feed.
	for gateID, state := range map[types.GateID]GateState{
		types.GateDataQuality: {
			State: types.GatePass, EvaluatedAt: now,
			ValidUntil: now.Add(10 * time.Second), SourceVersion: "seed",
		},
		types.GateSession: {
			State: types.GatePass, EvaluatedAt: now,
			ValidUntil: now.Add(10 * time.Second), SourceVersion: "seed",
		},
		types.GateNews: {
			// Self-evaluating from live GateInput.NewsRisk — does not use cached
			// state, so no ValidUntil (refreshGateStates does not refresh it; an
			// expiring seed would otherwise go GATE_STATE_STALE → DEGRADED →
			// fail the whole chain forever).
			State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed",
		},
		types.GateSpread: {
			State: types.GatePass, EvaluatedAt: now,
			ValidUntil: now.Add(5 * time.Second), SourceVersion: "seed",
		},
		types.GateSlippage: {
			// Self-evaluating from live GateInput — no ValidUntil (see GateNews).
			State: types.GatePass, Value: float64(0), EvaluatedAt: now, SourceVersion: "seed",
		},
		types.GateTotalCost: {
			// Self-evaluating from live GateInput — no ValidUntil (see GateNews).
			State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed",
		},
		types.GateRRNetExpectancy: {
			// Self-evaluating from live GateInput — no ValidUntil (see GateNews).
			State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed",
		},
		// ── Safety-critical gates: FAIL CLOSED until authoritative data arrives ──
		types.GateExposure: {
			State: types.GateUnknown, Value: float64(0), EvaluatedAt: now,
			ValidUntil: now, // already expired → forces fail-closed
			ReasonCode: "NOT_INITIALIZED", SourceVersion: "seed",
		},
		types.GateMargin: {
			State: types.GateUnknown, Value: false, EvaluatedAt: now,
			ValidUntil: now,
			ReasonCode: "NOT_INITIALIZED", SourceVersion: "seed",
		},
		types.GateEntitlement: {
			State: types.GateUnknown, EvaluatedAt: now,
			ValidUntil: now,
			ReasonCode: "NOT_INITIALIZED", SourceVersion: "seed",
		},
		types.GateLicense: {
			State: types.GateUnknown, EvaluatedAt: now,
			ValidUntil: now,
			ReasonCode: "NOT_INITIALIZED", SourceVersion: "seed",
		},
		types.GateExecutionPermit: {
			State: types.GateUnknown, EvaluatedAt: now,
			ValidUntil: now,
			ReasonCode: "NOT_INITIALIZED", SourceVersion: "seed",
		},
		// ── Self-evaluating precision gates (B-04): evaluated from live
		// GateInput each call; seeded PASS so EvaluateAll does not fail with
		// GATE_NOT_INITIALIZED on the first signal before refresh ticks fire.
		types.GateMinATR:         {State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed"},
		types.GateStopHuntFilter: {State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed"},
		// Profitability gate is pure-input (evaluated from live GateInput each
		// call); seed PASS so EvaluateAll does not fail closed on the first signal.
		types.GateProfitability: {State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed"},
		// Broker symbol validation: seeded PASS — degrades when broker
		// metadata is unavailable, but must not block signals.
		types.GateBrokerSymbolValidation: {State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed"},
	} {
		reg.UpdateState(gateID, state)
	}
}
