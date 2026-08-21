// Package gates implements the non-blocking, fail-closed hard gate architecture.
// SOW Section 131: Pure cached-state gate contract — no synchronous I/O in the decision path.
package gates

import (
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// GateState is the cached, generation-stamped state for a gate (SOW Section 131.2).
type GateState struct {
	GateID        types.GateID
	Scope         string
	Value         any
	State         types.GateResult
	ReasonCode    string // human-readable reason for the current state (observability)
	EvaluatedAt   time.Time
	EventTime     time.Time
	FreshnessMs   int64
	ValidUntil    time.Time
	SourceVersion string
	ConfigVersion string
	Quality       types.QualityState
	Generation    uint64
}

// Gate is the interface for a deterministic pure-evaluation gate.
// SOW Section 131.1: gate(input_snapshot, gate_state_snapshot) -> PASS | VETO | DEGRADED | UNKNOWN
type Gate interface {
	ID() types.GateID
	Evaluate(input GateInput, state GateState) GateEvaluation
}

// GateInput is the snapshot provided to each gate evaluation.
type GateInput struct {
	Tick           *types.Tick
	StrategyID     types.StrategyID
	Regime         types.Regime
	Spread         float64
	ATR            float64
	SignalScore    float64
	EntryPrice     float64
	StopLoss       float64
	TakeProfit1    float64
	RoundTripCost  float64
	CurrentExposure float64
	MaxExposure    float64
	NewsRisk       string
	SessionAllowed bool
	EntitlementOK  bool
	LicenseActive  bool
	ExecutionPermitted bool
	// Structural levels for StopHuntFilterGate
	StructuralLow  float64 // nearest swing low (0 = not available)
	StructuralHigh float64 // nearest swing high (0 = not available)
}

// GateEvaluation records the result of a single gate check.
type GateEvaluation struct {
	GateID       types.GateID      `json:"gate_id"`
	Result       types.GateResult  `json:"result"`
	ReasonCodes  []string          `json:"reason_codes"`
	EvaluatedAt  time.Time         `json:"evaluated_at"`
	FreshnessMs  int64             `json:"freshness_ms"`
	StateVersion string            `json:"state_version"`
}

// Registry holds all registered gates and their cached state.
type Registry struct {
	mu     sync.RWMutex
	gates  map[types.GateID]Gate
	states map[types.GateID]GateState
	order  []types.GateID // short-circuit order (SOW Section 131.4)
}

// NewRegistry creates a gate registry with the canonical short-circuit ordering.
func NewRegistry() *Registry {
	r := &Registry{
		gates:  make(map[types.GateID]Gate),
		states: make(map[types.GateID]GateState),
		order: []types.GateID{
			types.GateDataQuality,
			types.GateSession,
			types.GateNews,
			types.GateSpread,
			types.GateSlippage,
			types.GateTotalCost,
			types.GateExposure,
			types.GateMargin,
			types.GateRRNetExpectancy,
			types.GateEntitlement,
			types.GateLicense,
			types.GateExecutionPermit,
		},
	}
	return r
}

// Register adds a gate to the registry.
func (r *Registry) Register(g Gate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gates[g.ID()] = g
}

// UpdateState updates the cached state for a gate (called by background goroutines).
func (r *Registry) UpdateState(gateID types.GateID, state GateState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[gateID] = state
}

// GetState returns the current cached state for a gate.
func (r *Registry) GetState(gateID types.GateID) (GateState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.states[gateID]
	return s, ok
}

// EvaluateAll runs all gates in short-circuit order.
// SOW Section 131.4: The first hard veto terminates gate evaluation.
func (r *Registry) EvaluateAll(input GateInput) (allPass bool, evaluations []GateEvaluation, firstVeto *GateEvaluation) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allPass = true
	evaluations = make([]GateEvaluation, 0, len(r.order))

	for _, gateID := range r.order {
		gate, exists := r.gates[gateID]
		if !exists {
			// Gate not registered — fail closed
			eval := GateEvaluation{
				GateID:      gateID,
				Result:      types.GateUnknown,
				ReasonCodes: []string{"GATE_NOT_REGISTERED"},
				EvaluatedAt: time.Now(),
			}
			evaluations = append(evaluations, eval)
			allPass = false
			if firstVeto == nil {
				v := eval
				firstVeto = &v
			}
			continue
		}

		state, stateExists := r.states[gateID]
		if !stateExists {
			// No cached state — fail closed (SOW Section 131.1)
			// Distinguish NOT_INITIALIZED from missing
			eval := GateEvaluation{
				GateID:      gateID,
				Result:      types.GateUnknown,
				ReasonCodes: []string{"GATE_NOT_INITIALIZED"},
				EvaluatedAt: time.Now(),
			}
			evaluations = append(evaluations, eval)
			allPass = false
			if firstVeto == nil {
				v := eval
				firstVeto = &v
			}
			continue
		}

		// Check freshness (SOW Section 131.7)
		// Stale gate state: fail closed for risk-critical gates,
		// degrade (still allow) for non-critical gates if state was previously PASS
		if !state.ValidUntil.IsZero() && time.Now().After(state.ValidUntil) {
			// Risk-critical gates: exposure, margin — must have fresh state
			isRiskCritical := gateID == types.GateExposure || gateID == types.GateMargin
			if isRiskCritical {
				eval := GateEvaluation{
					GateID:      gateID,
					Result:      types.GateVeto,
					ReasonCodes: []string{"GATE_STATE_STALE"},
					EvaluatedAt: time.Now(),
					FreshnessMs: time.Since(state.EvaluatedAt).Milliseconds(),
				}
				evaluations = append(evaluations, eval)
				allPass = false
				if firstVeto == nil {
					v := eval
					firstVeto = &v
				}
				continue
			}
			// Non-critical gates: mark degraded but continue evaluation
			eval := GateEvaluation{
				GateID:      gateID,
				Result:      types.GateDegraded,
				ReasonCodes: []string{"GATE_STATE_STALE"},
				EvaluatedAt: time.Now(),
				FreshnessMs: time.Since(state.EvaluatedAt).Milliseconds(),
			}
			evaluations = append(evaluations, eval)
			// Degraded is not a hard veto — continue evaluation
		}

		// Evaluate the gate
		eval := gate.Evaluate(input, state)
		evaluations = append(evaluations, eval)

		if eval.Result == types.GateVeto || eval.Result == types.GateUnknown {
			allPass = false
			if firstVeto == nil {
				v := eval
				firstVeto = &v
			}
			// Short-circuit: first hard veto terminates evaluation
			break
		}
		// GateDegraded = stale but previously PASS — don't short-circuit, just mark as not all-pass
		if eval.Result == types.GateDegraded {
			allPass = false
			// Don't set firstVeto for degraded — it's not a hard veto
		}
	}

	return allPass, evaluations, firstVeto
}
