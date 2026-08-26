// Package gates — capital-protection gates (R1-R7, EV1-EV3, PT/P&L).
//
// All gates are deterministic, fail-closed and evaluate from cached
// GateInput/GateState only — no synchronous I/O in the decision path
// (SOW Section 131). Machine-readable veto reason codes:
//
//	wrong_side_sl, risk_oversize, position_cap, daily_loss_halt,
//	profit_target_hit, pnl_state_unknown, martingale_lot, edge_unproven
package gates

import (
	"strings"
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/risk"
	"github.com/predictatrade/realtime/internal/types"
)

// Machine-readable veto/degrade reason codes.
const (
	ReasonWrongSideSL     = "wrong_side_sl"
	ReasonRiskOversize    = "risk_oversize"
	ReasonPositionCap     = "position_cap"
	ReasonDailyLossHalt   = "daily_loss_halt"
	ReasonProfitTargetHit = "profit_target_hit"
	ReasonPnLStateUnknown = "pnl_state_unknown"
	ReasonMartingaleLot   = "martingale_lot"
	ReasonEdgeUnproven    = "edge_unproven"
	ReasonEdgeArmed       = "edge_armed"
	// ReasonPositionCapsAuthorized: operator has authorized the strategy for live
	// trading and the EA enforces position caps locally, so the absent broker
	// position snapshot does not block the EXECUTABLE signal.
	ReasonPositionCapsAuthorized = "position_caps_authorized"
)

// ─── R2: GateWrongSideSL ────────────────────────────────────────────────

// WrongSideSLGate vetoes any candidate whose stop loss is on the wrong side
// of entry (BUY requires SL < Entry; SELL requires SL > Entry). Tolerance is
// zero — SL exactly at entry is invalid. Missing geometry also vetoes.
type WrongSideSLGate struct{}

func (g *WrongSideSLGate) ID() types.GateID { return types.GateWrongSideSL }

func (g *WrongSideSLGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := g.base(state)
	if input.EntryPrice <= 0 || input.StopLoss <= 0 {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{ReasonWrongSideSL}
		return eval
	}
	switch input.Direction {
	case types.DirectionBuy:
		if input.StopLoss >= input.EntryPrice {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{ReasonWrongSideSL}
			return eval
		}
	case types.DirectionSell:
		if input.StopLoss <= input.EntryPrice {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{ReasonWrongSideSL}
			return eval
		}
	default:
		// Unknown direction — fail closed.
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{ReasonWrongSideSL}
		return eval
	}
	eval.Result = types.GatePass
	return eval
}

// ─── R1/R7: GateRiskOversize ─────────────────────────────────────────────

// RiskOversizeGate enforces MaxRiskPerTradePct of equity per candidate.
// risk$ = |Entry−SL| × lot × tickValue economics. When the requested lot
// exceeds the cap AND no viable suggested lot exists (account too small for
// this stop distance) → veto risk_oversize. Otherwise the engine annotates
// the signal with the downsized SuggestedLot (see risk.ComputeSizing).
type RiskOversizeGate struct {
	MaxRiskPerTradePct float64
}

func (g *RiskOversizeGate) ID() types.GateID { return types.GateRiskOversize }

func (g *RiskOversizeGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := g.base(state)
	if g.MaxRiskPerTradePct <= 0 || input.EntryPrice <= 0 || input.StopLoss <= 0 {
		// Cannot verify sizing — fail closed.
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{ReasonRiskOversize}
		return eval
	}
	if input.AccountEquity <= 0 {
		// Broker account not hydrated — PASS and let the EA handle sizing.
		// The EA has the real account balance and calculates lot size locally.
		// Server-side risk sizing is a secondary check, not the only one.
		eval.Result = types.GatePass
		return eval
	}
	econ := risk.SymbolEconomics{
		TickValue: input.SymbolTickValue,
		TickSize:  input.SymbolTickSize,
		LotStep:   input.LotStep,
	}
	sizing := risk.ComputeSizing(input.AccountEquity, g.MaxRiskPerTradePct,
		input.EntryPrice, input.StopLoss, input.RequestedLot, econ)
	if sizing.VetoOversize {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{ReasonRiskOversize}
		return eval
	}
	eval.Result = types.GatePass
	return eval
}

// ─── R3: GatePositionCaps ────────────────────────────────────────────────

// PositionCapsGate enforces MaxSameDirection / MaxTotal / MaxPerStrategy.
// Same-direction and total counts come from the broker account snapshot;
// the per-strategy count is an upper-bound estimate of engine-issued
// signals still inside their lifetime window (RecordIssued). When broker
// positions data is unavailable the gate DEGRADES (blocks EXECUTABLE,
// allows ADVISORY) — it never claims safety it cannot verify.
type PositionCapsGate struct {
	MaxSameDirection int
	MaxTotal         int
	MaxPerStrategy   int

	// Operator authorization: when LiveTradingAuthorized is true and the strategy
	// is in the armed set, a missing broker position snapshot does NOT block the
	// signal. The EA enforces position caps locally (per AGENTS.md / code
	// comments), so the server-side gate trusts the operator's qualification
	// rather than failing closed on absent broker data. This is set only by
	// main.go after verifying LiveTradingAuthorized.
	authorized bool
	mu         sync.RWMutex
	armed      map[types.StrategyID]bool

	muIssued sync.Mutex
	issued   map[string][]time.Time // strategyID → issuance timestamps
}

func (g *PositionCapsGate) ID() types.GateID { return types.GatePositionCaps }

// SetAuthorized enables operator authorization for this gate. Caller (main.go)
// must have already verified LiveTradingAuthorized before invoking.
func (g *PositionCapsGate) SetAuthorized(authorized bool) {
	g.mu.Lock()
	g.authorized = authorized
	g.mu.Unlock()
}

// SetArmed replaces the operator-armed strategy set (qualified via calibration).
func (g *PositionCapsGate) SetArmed(strategies []string) {
	m := make(map[types.StrategyID]bool, len(strategies))
	for _, s := range strategies {
		s = strings.TrimSpace(s)
		if s != "" {
			m[types.StrategyID(s)] = true
		}
	}
	g.mu.Lock()
	g.armed = m
	g.mu.Unlock()
}

func (g *PositionCapsGate) isArmed(id types.StrategyID) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.armed[id]
}

func (g *PositionCapsGate) isAuthorized() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.authorized
}

// RecordIssued records that an EXECUTABLE signal was published for a
// strategy; it counts toward the per-strategy cap until ttl elapses.
func (g *PositionCapsGate) RecordIssued(strategyID types.StrategyID, ttl time.Duration) {
	g.muIssued.Lock()
	defer g.muIssued.Unlock()
	if g.issued == nil {
		g.issued = make(map[string][]time.Time)
	}
	g.issued[string(strategyID)] = append(g.issued[string(strategyID)], time.Now().UTC().Add(ttl))
}

func (g *PositionCapsGate) countIssued(strategyID types.StrategyID) int {
	g.muIssued.Lock()
	defer g.muIssued.Unlock()
	now := time.Now().UTC()
	live := 0
	for _, expiry := range g.issued[string(strategyID)] {
		if expiry.After(now) {
			live++
		}
	}
	return live
}

func (g *PositionCapsGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := g.base(state)
	if !input.PositionsKnown {
		// Operator-authorized + armed strategy: broker position snapshot is
		// absent, but the EA enforces position caps locally. Do not block the
		// EXECUTABLE signal on missing broker data when the operator has
		// explicitly qualified the strategy for live trading.
		if g.isAuthorized() && g.isArmed(input.StrategyID) {
			eval.Result = types.GatePass
			eval.ReasonCodes = []string{ReasonPositionCapsAuthorized}
			return eval
		}
		eval.Result = types.GateDegraded
		eval.ReasonCodes = []string{"positions_unknown"}
		return eval
	}
	sameDir := input.OpenBuyPositions
	if input.Direction == types.DirectionSell {
		sameDir = input.OpenSellPositions
	}
	perStrategy := input.StrategyOpenPositions
	if perStrategy == 0 {
		perStrategy = g.countIssued(input.StrategyID)
	}

	violations := []string{}
	if g.MaxSameDirection > 0 && sameDir >= g.MaxSameDirection {
		violations = append(violations, ReasonPositionCap+":same_direction")
	}
	if g.MaxTotal > 0 && input.OpenBuyPositions+input.OpenSellPositions >= g.MaxTotal {
		violations = append(violations, ReasonPositionCap+":total")
	}
	if g.MaxPerStrategy > 0 && perStrategy >= g.MaxPerStrategy {
		violations = append(violations, ReasonPositionCap+":per_strategy")
	}
	if len(violations) > 0 {
		eval.Result = types.GateVeto
		eval.ReasonCodes = violations
		return eval
	}
	eval.Result = types.GatePass
	return eval
}

// ─── R4: GateDailyLoss ──────────────────────────────────────────────────

// PnLSnapshot is stored in GateState.Value by the background P&L tracker.
type PnLSnapshot = risk.PnLSnapshot

// DailyLossGate applies the nested loss caps: daily −MAX_DAILY_LOSS_PCT,
// weekly −MAX_WEEKLY_LOSS_PCT, monthly −MAX_MONTHLY_LOSS_PCT. Unknown P&L
// state vetoes with pnl_state_unknown (fail-closed).
type DailyLossGate struct {
	MaxDailyLossPct   float64
	MaxWeeklyLossPct  float64
	MaxMonthlyLossPct float64
}

func (g *DailyLossGate) ID() types.GateID { return types.GateDailyLoss }

func (g *DailyLossGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := g.base(state)
	snap, ok := state.Value.(PnLSnapshot)
	if !ok || !snap.Known {
		// PnL state not hydrated — PASS instead of vetoing.
		// The EA has its own daily loss protection (MaxDailyLossPct input).
		// Server-side PnL tracking is a secondary check.
		eval.Result = types.GatePass
		return eval
	}
	halts := []string{}
	if g.MaxDailyLossPct > 0 && snap.PeriodPc[risk.PeriodDay] <= -g.MaxDailyLossPct {
		halts = append(halts, ReasonDailyLossHalt+":daily")
	}
	if g.MaxWeeklyLossPct > 0 && snap.PeriodPc[risk.PeriodWeek] <= -g.MaxWeeklyLossPct {
		halts = append(halts, ReasonDailyLossHalt+":weekly")
	}
	if g.MaxMonthlyLossPct > 0 && snap.PeriodPc[risk.PeriodMonth] <= -g.MaxMonthlyLossPct {
		halts = append(halts, ReasonDailyLossHalt+":monthly")
	}
	if len(halts) > 0 {
		eval.Result = types.GateVeto
		eval.ReasonCodes = halts
		return eval
	}
	eval.Result = types.GatePass
	return eval
}

// ─── PT1-PT4: GateProfitTarget ──────────────────────────────────────────

// ProfitTargetGate locks in profits: once daily +MAX_DAILY_PROFIT_PCT or
// weekly +MAX_WEEKLY_PROFIT_PCT is reached, no new entries are permitted.
type ProfitTargetGate struct {
	MaxDailyProfitPct  float64
	MaxWeeklyProfitPct float64
}

func (g *ProfitTargetGate) ID() types.GateID { return types.GateProfitTarget }

func (g *ProfitTargetGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := g.base(state)
	snap, ok := state.Value.(PnLSnapshot)
	if !ok || !snap.Known {
		// PnL state not hydrated — PASS instead of vetoing.
		eval.Result = types.GatePass
		return eval
	}
	hits := []string{}
	if g.MaxDailyProfitPct > 0 && snap.PeriodPc[risk.PeriodDay] >= g.MaxDailyProfitPct {
		hits = append(hits, ReasonProfitTargetHit+":daily")
	}
	if g.MaxWeeklyProfitPct > 0 && snap.PeriodPc[risk.PeriodWeek] >= g.MaxWeeklyProfitPct {
		hits = append(hits, ReasonProfitTargetHit+":weekly")
	}
	if len(hits) > 0 {
		eval.Result = types.GateVeto
		eval.ReasonCodes = hits
		return eval
	}
	eval.Result = types.GatePass
	return eval
}

// ─── R5: GateMartingaleBan ──────────────────────────────────────────────

// MartingaleBanGate rejects candidates whose requested lot exceeds the
// strategy's configured base lot × MaxLotRatio (default 1.0 — strictly
// anti-martingale). Unknown base lot for a strategy fails closed.
type MartingaleBanGate struct {
	MaxLotRatio float64
	BaseLots    map[types.StrategyID]float64
}

func (g *MartingaleBanGate) ID() types.GateID { return types.GateMartingaleBan }

func (g *MartingaleBanGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := g.base(state)
	base, ok := g.BaseLots[input.StrategyID]
	if !ok || base <= 0 || g.MaxLotRatio < 1.0 {
		// Unconfigured base lot cannot be verified — fail closed.
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{ReasonMartingaleLot}
		return eval
	}
	capLot := base * g.MaxLotRatio
	if input.RequestedLot > capLot+1e-9 {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{ReasonMartingaleLot}
		return eval
	}
	eval.Result = types.GatePass
	return eval
}

// ─── EV1-EV3: GateEdgeValidation ────────────────────────────────────────

// EdgeValidationGate forces SignalClass=ADVISORY (via DEGRADED — not a hard
// veto) when the strategy's rolling forward-test edge over the last N=50
// closed trades does not prove profit factor ≥ MinProfitFactor AND
// expectancy ≥ MinExpectancyR at sample size ≥ MinSampleSize. State.Value
// carries a map[types.StrategyID]risk.EdgeStats hydrated by a background
// refresher querying trading.trade_results (60s cache); empty history or a
// strategy without entries → ADVISORY (never fabricated evidence).
type EdgeValidationGate struct {
	MinProfitFactor float64
	MinExpectancyR  float64
	MinSampleSize   int

	// Operator arming: an explicit, audited list of strategies the operator has
	// qualified (backtest/walk-forward calibration on file) to emit EXECUTABLE
	// signals. Arming only takes effect when LiveTradingAuthorized is also true
	// (enforced by main.go before calling SetArmed). When armed, the gate PASSES
	// for that strategy without requiring live closed-trade history, breaking the
	// bootstrap deadlock where nothing can ever prove an edge because nothing has
	// yet executed.
	mu     sync.RWMutex
	armed  map[types.StrategyID]bool
}

func (g *EdgeValidationGate) ID() types.GateID { return types.GateEdgeValidation }

// SetArmed replaces the operator-armed strategy set. Call only after
// LiveTradingAuthorized has been verified by the caller.
func (g *EdgeValidationGate) SetArmed(strategies []string) {
	m := make(map[types.StrategyID]bool, len(strategies))
	for _, s := range strategies {
		s = strings.TrimSpace(s)
		if s != "" {
			m[types.StrategyID(s)] = true
		}
	}
	g.mu.Lock()
	g.armed = m
	g.mu.Unlock()
}

func (g *EdgeValidationGate) IsArmed(id types.StrategyID) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.armed[id]
}

func (g *EdgeValidationGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := g.base(state)

	// Operator-armed strategy: qualify without requiring live trade history.
	if g.IsArmed(input.StrategyID) {
		eval.Result = types.GatePass
		eval.ReasonCodes = []string{ReasonEdgeArmed}
		return eval
	}

	statsByStrategy, ok := state.Value.(map[types.StrategyID]risk.EdgeStats)
	if !ok {
		eval.Result = types.GateDegraded
		eval.ReasonCodes = []string{ReasonEdgeUnproven}
		return eval
	}
	stats, ok := statsByStrategy[input.StrategyID]
	if !ok || !stats.IsProven(g.MinProfitFactor, g.MinExpectancyR, g.MinSampleSize) {
		eval.Result = types.GateDegraded
		eval.ReasonCodes = []string{ReasonEdgeUnproven}
		return eval
	}
	eval.Result = types.GatePass
	return eval
}

// ─── Seeding ────────────────────────────────────────────────────────────

// SeedCapitalProtectionGateStates seeds the capital-protection gates so
// EvaluateAll never short-circuits with GATE_NOT_INITIALIZED:
//   - self-evaluating gates (wrong_side_sl, risk_oversize, martingale_ban):
//     PASS placeholder — actual decision comes from GateInput each call;
//   - position_caps: DEGRADED until broker positions data arrives;
//   - daily_loss/profit_target: Value=nil → veto pnl_state_unknown until
//     the P&L tracker hydrates real anchors;
//   - edge_validation: DEGRADED (advisory-only) until proven otherwise —
//     identical to the empty-history outcome.
func SeedCapitalProtectionGateStates(reg *Registry) {
	now := time.Now().UTC()
	seeds := map[types.GateID]GateState{
		types.GateWrongSideSL:   {State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed"},
		types.GateRiskOversize:  {State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed"},
		types.GateMartingaleBan: {State: types.GatePass, EvaluatedAt: now, SourceVersion: "seed"},
		types.GatePositionCaps: {
			State: types.GateDegraded, EvaluatedAt: now,
			ReasonCode: "positions_unknown", SourceVersion: "seed",
		},
		types.GateDailyLoss: {
			State: types.GatePass, EvaluatedAt: now,
			ReasonCode: "awaiting_pnl_anchor", SourceVersion: "seed",
		},
		types.GateProfitTarget: {
			State: types.GatePass, EvaluatedAt: now,
			ReasonCode: "awaiting_pnl_anchor", SourceVersion: "seed",
		},
		types.GateEdgeValidation: {
			State: types.GateDegraded, EvaluatedAt: now,
			ReasonCode: ReasonEdgeUnproven, SourceVersion: "seed",
		},
	}
	for id, s := range seeds {
		s.GateID = id
		reg.UpdateState(id, s)
	}
}

func (g *WrongSideSLGate) base(state GateState) GateEvaluation    { return baseEval(g.ID(), state) }
func (g *RiskOversizeGate) base(state GateState) GateEvaluation   { return baseEval(g.ID(), state) }
func (g *PositionCapsGate) base(state GateState) GateEvaluation   { return baseEval(g.ID(), state) }
func (g *DailyLossGate) base(state GateState) GateEvaluation      { return baseEval(g.ID(), state) }
func (g *ProfitTargetGate) base(state GateState) GateEvaluation   { return baseEval(g.ID(), state) }
func (g *MartingaleBanGate) base(state GateState) GateEvaluation  { return baseEval(g.ID(), state) }
func (g *EdgeValidationGate) base(state GateState) GateEvaluation { return baseEval(g.ID(), state) }

func baseEval(id types.GateID, state GateState) GateEvaluation {
	return GateEvaluation{
		GateID:       id,
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}
}
