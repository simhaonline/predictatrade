// Package gates implements the non-blocking, fail-closed hard gate architecture.
// SOW Section 131: Pure cached-state gate contract — no synchronous I/O in the decision path.
package gates

import (
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// GateScope keys a gate's cached state. Market-wide (global) gates use the
// zero-value scope (empty StrategyID/Timeframe). Strategy- and timeframe-scoped
// gates use {GateID, StrategyID, Timeframe} so that each strategy on each
// timeframe carries its OWN isolated, independently-fresh state. This prevents a
// single shared/central snapshot from going stale or erroring and silently
// blocking or degrading unrelated strategies (SOW Section 131, capital safety).
type GateScope struct {
	GateID     types.GateID
	StrategyID types.StrategyID
	Timeframe  types.Timeframe
}

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
	Tick               *types.Tick
	StrategyID         types.StrategyID
	// Timeframe is the decision timeframe of the triggering candle. Gate state is
	// resolved per (StrategyID, Timeframe) so timeframe-specific metrics (ATR,
	// structural levels, edge stats) are never conflated across timeframes.
	Timeframe          types.Timeframe
	Regime             types.Regime
	Spread             float64
	ATR                float64
	SignalScore        float64
	EntryPrice         float64
	StopLoss           float64
	TakeProfit1        float64
	RoundTripCost      float64
	CurrentExposure    float64
	MaxExposure        float64
	NewsRisk           string
	SessionAllowed     bool
	EntitlementOK      bool
	LicenseActive      bool
	ExecutionPermitted bool
	// Structural levels for StopHuntFilterGate
	StructuralLow  float64 // nearest swing low (0 = not available)
	StructuralHigh float64 // nearest swing high (0 = not available)

	// ─── Capital-protection inputs (R1-R7) ───
	Direction         types.Direction // BUY or SELL (zero for non-directional evaluation)
	AccountEquity     float64         // broker snapshot equity (0 = unavailable → fail-closed)
	AccountFreeMargin float64         // broker snapshot free margin
	AccountLeverage   float64         // broker leverage (0 → risk.DefaultLeverage)
	SymbolTickValue   float64         // broker symbol tick value (0 → XAUUSD default)
	SymbolTickSize    float64         // broker symbol tick size (0 → XAUUSD default)
	LotStep           float64         // broker lot step (0 → 0.01)
	RequestedLot      float64         // candidate base lot for this strategy
	// Position census from the broker snapshot. PositionsKnown=false means
	// no broker positions data has arrived — the caps gate must DEGRADE,
	// never claim safety it cannot verify.
	PositionsKnown        bool
	OpenBuyPositions      int
	OpenSellPositions     int
	StrategyOpenPositions int // per-strategy open-position estimate (engine-issued signals)

	// Refinement (prompt.md): strategy-computed profitability flags.
	// These are set by the signal engine from the strategy evaluation so the
	// delivery layer can eliminate loss-making candidates.
	EntryGatePassed  bool // strategy's unique entry gate passed
	IsLossCandidate  bool // strategy flagged this candidate as negative-EV
	// RefinementProvided is true only when the signal engine actually populated
	// the refinement flags. When false (e.g. direct GateInput construction in
	// tests or legacy paths), the gate falls back to its own EV computation and
	// must not veto solely on zero-valued default flags.
	RefinementProvided bool
}

// GateEvaluation records the result of a single gate check.
type GateEvaluation struct {
	GateID       types.GateID     `json:"gate_id"`
	Result       types.GateResult `json:"result"`
	ReasonCodes  []string         `json:"reason_codes"`
	EvaluatedAt  time.Time        `json:"evaluated_at"`
	FreshnessMs  int64            `json:"freshness_ms"`
	StateVersion string           `json:"state_version"`
}

// Registry holds all registered gates and their cached state.
type Registry struct {
	mu     sync.RWMutex
	gates  map[types.GateID]Gate
	states map[GateScope]GateState
	// globalGates are market-wide gates evaluated against a single shared scope
	// (empty StrategyID/Timeframe). All other gates are resolved per
	// (StrategyID, Timeframe) so strategy/timeframe state stays isolated.
	globalGates map[types.GateID]bool
	order  []types.GateID // short-circuit order (SOW Section 131.4)
}

// NewRegistry creates a gate registry with the canonical short-circuit ordering.
func NewRegistry() *Registry {
	r := &Registry{
		gates:  make(map[types.GateID]Gate),
		states: make(map[GateScope]GateState),
		globalGates: map[types.GateID]bool{
			types.GateDataQuality:     true,
			types.GateSession:         true,
			types.GateNews:            true,
			types.GateSpread:          true,
			types.GateSlippage:        true,
			types.GateTotalCost:       true,
			types.GateExposure:        true,
			types.GateMargin:          true,
			types.GateEntitlement:     true,
			types.GateLicense:         true,
			types.GateExecutionPermit: true,
		},
		order: []types.GateID{
			types.GateDataQuality,
			types.GateSession,
			types.GateNews,
			types.GateSpread,
			types.GateSlippage,
			types.GateTotalCost,
			types.GateMinATR,
			types.GateStopHuntFilter,
			types.GateExposure,
			types.GateMargin,
			types.GateRRNetExpectancy,
			types.GateProfitability,
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

// RegisterOrdered adds a gate and inserts it into the short-circuit
// evaluation order immediately after `after`. If `after` is not present in
// the order slice the gate is appended at the end. Gates registered with
// plain Register are evaluated only if already present in the order.
func (r *Registry) RegisterOrdered(g Gate, after types.GateID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gates[g.ID()] = g
	idx := -1
	for i, id := range r.order {
		if id == after {
			idx = i
			break
		}
	}
	if idx < 0 {
		r.order = append(r.order, g.ID())
		return
	}
	next := append([]types.GateID{g.ID()}, r.order[idx+1:]...)
	r.order = append(r.order[:idx+1], next...)
}

// UpdateState updates the cached state for a gate at the GLOBAL scope (market-wide
// gates). Strategy/timeframe-scoped gates should use UpdateStateScoped.
func (r *Registry) UpdateState(gateID types.GateID, state GateState) {
	r.UpdateStateScoped(GateScope{GateID: gateID}, state)
}

// UpdateStateScoped writes cached state for an explicit (gate, strategy, timeframe) scope.
func (r *Registry) UpdateStateScoped(scope GateScope, state GateState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state.GateID = scope.GateID
	r.states[scope] = state
}

// GetState returns the current cached state for a gate at the GLOBAL scope.
func (r *Registry) GetState(gateID types.GateID) (GateState, bool) {
	return r.GetStateScoped(GateScope{GateID: gateID})
}

// GetStateScoped returns cached state for an explicit scope.
func (r *Registry) GetStateScoped(scope GateScope) (GateState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.states[scope]
	return s, ok
}

// GetEdgeState resolves the edge-validation state for a specific (strategy,
// timeframe) using the same fallback chain as EvaluateAll: (strategy, tf) →
// (strategy, "") → global. Returns (zero, false) if no state is cached anywhere.
// Used by capital-protection paths that must read the isolated per-strategy edge
// stats directly (the global scope no longer carries the shared stats map).
func (r *Registry) GetEdgeState(strategyID types.StrategyID, tf types.Timeframe) (GateState, bool) {
	if s, ok := r.GetStateScoped(GateScope{GateID: types.GateEdgeValidation, StrategyID: strategyID, Timeframe: tf}); ok {
		return s, true
	}
	if tf != "" {
		if s, ok := r.GetStateScoped(GateScope{GateID: types.GateEdgeValidation, StrategyID: strategyID}); ok {
			return s, true
		}
	}
	return r.GetStateScoped(GateScope{GateID: types.GateEdgeValidation})
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

		// Resolve this gate's cached state. Strategy/timeframe-scoped gates use
		// their own isolated scope; market-wide gates use the global scope. The
		// fallback chain keeps each (strategy, timeframe) independent so a stale or
		// errored scope can NEVER veto or degrade unrelated strategies.
		scope := GateScope{GateID: gateID, StrategyID: input.StrategyID, Timeframe: input.Timeframe}
		state, stateExists := r.GetStateScoped(scope)
		if !stateExists && input.Timeframe != "" {
			// Fall back to the strategy-wide scope (legacy strategies that do not
			// declare a specific timeframe).
			state, stateExists = r.GetStateScoped(GateScope{GateID: gateID, StrategyID: input.StrategyID})
		}
		if !stateExists {
			// Final fallback: the global scope (covers market-wide gates and any
			// globally-seeded strategy-gate states).
			state, stateExists = r.GetStateScoped(GateScope{GateID: gateID})
		}

	if !stateExists {
		// No cached state for this scope after the full fallback chain. Fail
		// CLOSED for this (gate, strategy, timeframe) scope. Because state is now
		// scoped per (strategy, timeframe), this veto isolates a single strategy's
		// missing data and cannot cascade to unrelated strategies or timeframes —
		// which is the core fix for the old central-gate "block everything" risk.
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
