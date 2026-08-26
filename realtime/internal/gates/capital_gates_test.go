package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/risk"
	"github.com/predictatrade/realtime/internal/types"
)

var capTestState = GateState{State: types.GatePass, EvaluatedAt: time.Now()}

// ─── R2: WrongSideSLGate ────────────────────────────────────────────────

func TestWrongSideSLGate(t *testing.T) {
	g := &WrongSideSLGate{}
	cases := []struct {
		name      string
		direction types.Direction
		entry     float64
		sl        float64
		want      types.GateResult
	}{
		{"buy valid", types.DirectionBuy, 2430, 2426, types.GatePass},
		{"buy SL==entry vetoed (tolerance zero)", types.DirectionBuy, 2430, 2430, types.GateVeto},
		{"buy wrong side", types.DirectionBuy, 2430, 2434, types.GateVeto},
		{"sell valid", types.DirectionSell, 2430, 2434, types.GatePass},
		{"sell SL==entry vetoed", types.DirectionSell, 2430, 2430, types.GateVeto},
		{"sell wrong side", types.DirectionSell, 2430, 2426, types.GateVeto},
		{"missing entry", types.DirectionBuy, 0, 2426, types.GateVeto},
		{"missing SL", types.DirectionBuy, 2430, 0, types.GateVeto},
		{"unknown direction fails closed", types.Direction("BUY_CANDIDATE"), 2430, 2426, types.GateVeto},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := g.Evaluate(GateInput{Direction: tc.direction, EntryPrice: tc.entry, StopLoss: tc.sl}, capTestState)
			if eval.Result != tc.want {
				t.Errorf("result = %s, want %s", eval.Result, tc.want)
			}
			if eval.Result == types.GateVeto && eval.ReasonCodes[0] != ReasonWrongSideSL {
				t.Errorf("reason = %s, want %s", eval.ReasonCodes[0], ReasonWrongSideSL)
			}
		})
	}
}

// ─── R1/R7: RiskOversizeGate ────────────────────────────────────────────

func TestRiskOversizeGate(t *testing.T) {
	g := &RiskOversizeGate{MaxRiskPerTradePct: 1.5} // $15 on $1000 equity
	cases := []struct {
		name         string
		equity       float64
		entry        float64
		sl           float64
		requestedLot float64
		want         types.GateResult
	}{
		// riskPerLot = 2.0/0.01*1 = $200/lot; 0.05 lot → $10 ≤ $15 → PASS
		{"small lot within cap", 1000, 2430, 2428, 0.05, types.GatePass},
		// 0.10 lot → $20 > $15 but suggested = floor(15/200/0.01)*0.01 = 0.07 ≥ minLot → PASS (downsize)
		{"oversize with viable suggestion", 1000, 2430, 2428, 0.10, types.GatePass},
		// stop distance 50 → riskPerLot $5000; suggested = floor(15/5000/0.01)=0 → VETO
		{"account too small for stop distance", 1000, 2430, 2380, 0.01, types.GateVeto},
		{"no equity fails closed", 0, 2430, 2428, 0.01, types.GateVeto},
		{"bad geometry fails closed", 1000, 0, 2428, 0.01, types.GateVeto},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := g.Evaluate(GateInput{
				AccountEquity: tc.equity, EntryPrice: tc.entry, StopLoss: tc.sl,
				RequestedLot: tc.requestedLot,
			}, capTestState)
			if eval.Result != tc.want {
				t.Errorf("result = %s, want %s", eval.Result, tc.want)
			}
			if eval.Result == types.GateVeto && eval.ReasonCodes[0] != ReasonRiskOversize {
				t.Errorf("reason = %s, want %s", eval.ReasonCodes[0], ReasonRiskOversize)
			}
		})
	}
}

// ─── R3: PositionCapsGate ───────────────────────────────────────────────

func TestPositionCapsGate(t *testing.T) {
	g := &PositionCapsGate{MaxSameDirection: 1, MaxTotal: 2, MaxPerStrategy: 1}
	buyInput := func(buy, sell int, known bool) GateInput {
		return GateInput{
			Direction: types.DirectionBuy, PositionsKnown: known,
			OpenBuyPositions: buy, OpenSellPositions: sell,
		}
	}

	cases := []struct {
		name string
		in   GateInput
		want types.GateResult
	}{
		{"flat account passes", buyInput(0, 0, true), types.GatePass},
		{"same-direction cap hit", buyInput(1, 0, true), types.GateVeto},
		{"opposite direction allowed under same-direction cap", buyInput(0, 1, true), types.GatePass},
		{"total cap hit (one each side)", buyInput(0, 2, true), types.GateVeto},
		{"positions unknown degrades", buyInput(0, 0, false), types.GateDegraded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			in.StrategyID = "STANDARD_SCALPING"
			eval := g.Evaluate(in, capTestState)
			if eval.Result != tc.want {
				t.Errorf("result = %s (%v), want %s", eval.Result, eval.ReasonCodes, tc.want)
			}
			if tc.want == types.GateVeto && eval.ReasonCodes[0] != ReasonPositionCap+":same_direction" &&
				eval.ReasonCodes[0] != ReasonPositionCap+":total" {
				t.Errorf("unexpected reasons %v", eval.ReasonCodes)
			}
			if tc.want == types.GateDegraded && eval.ReasonCodes[0] != "positions_unknown" {
				t.Errorf("degraded reason = %s, want positions_unknown", eval.ReasonCodes[0])
			}
		})
	}

	// Per-strategy cap via engine-issued signal estimator.
	g2 := &PositionCapsGate{MaxSameDirection: 5, MaxTotal: 5, MaxPerStrategy: 1}
	g2.RecordIssued("TREND_SWING", time.Hour)
	eval := g2.Evaluate(GateInput{Direction: types.DirectionBuy, PositionsKnown: true,
		StrategyID: "TREND_SWING"}, capTestState)
	if eval.Result != types.GateVeto {
		t.Errorf("per-strategy cap: result = %s, want VETO", eval.Result)
	}
	// Expired issuance no longer counts.
	g3 := &PositionCapsGate{MaxSameDirection: 5, MaxTotal: 5, MaxPerStrategy: 1}
	g3.RecordIssued("TREND_SWING", -time.Minute)
	eval = g3.Evaluate(GateInput{Direction: types.DirectionBuy, PositionsKnown: true,
		StrategyID: "TREND_SWING"}, capTestState)
	if eval.Result != types.GatePass {
		t.Errorf("expired issuance should not count, got %s", eval.Result)
	}
}

// ─── R4 / PT: DailyLossGate + ProfitTargetGate ──────────────────────────

func TestDailyLossGate(t *testing.T) {
	g := &DailyLossGate{MaxDailyLossPct: 2, MaxWeeklyLossPct: 4, MaxMonthlyLossPct: 5}
	snap := func(day, week, month float64, known bool) GateState {
		return GateState{Value: PnLSnapshot{
			Known: known, Equity: 1000,
			PeriodPc: map[risk.Period]float64{
				risk.PeriodDay: day, risk.PeriodWeek: week, risk.PeriodMonth: month,
			},
		}}
	}
	cases := []struct {
		name   string
		state  GateState
		want   types.GateResult
		reason string
	}{
		{"healthy", snap(-0.5, -1, -2, true), types.GatePass, ""},
		{"daily halt at -2%", snap(-2.0, -1, -2, true), types.GateVeto, ReasonDailyLossHalt + ":daily"},
		{"weekly halt at -4%", snap(-1, -4.5, -2, true), types.GateVeto, ReasonDailyLossHalt + ":weekly"},
		{"monthly halt at -5%", snap(-1, -2, -6, true), types.GateVeto, ReasonDailyLossHalt + ":monthly"},
		{"unknown state vetoes pnl_state_unknown", GateState{}, types.GateVeto, ReasonPnLStateUnknown},
		{"known=false vetoes pnl_state_unknown", snap(0, 0, 0, false), types.GateVeto, ReasonPnLStateUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := g.Evaluate(GateInput{}, tc.state)
			if eval.Result != tc.want {
				t.Errorf("result = %s, want %s", eval.Result, tc.want)
			}
			if tc.reason != "" && eval.ReasonCodes[0] != tc.reason {
				t.Errorf("reason = %s, want %s", eval.ReasonCodes[0], tc.reason)
			}
		})
	}
}

func TestProfitTargetGate(t *testing.T) {
	g := &ProfitTargetGate{MaxDailyProfitPct: 5, MaxWeeklyProfitPct: 12}
	mk := func(day, week float64) GateState {
		return GateState{Value: PnLSnapshot{
			Known: true, Equity: 1000,
			PeriodPc: map[risk.Period]float64{
				risk.PeriodDay: day, risk.PeriodWeek: week, risk.PeriodMonth: 0,
			},
		}}
	}
	cases := []struct {
		name   string
		state  GateState
		want   types.GateResult
		reason string
	}{
		{"below targets", mk(2, 5), types.GatePass, ""},
		{"daily target hit", mk(5.1, 0), types.GateVeto, ReasonProfitTargetHit + ":daily"},
		{"weekly target hit", mk(0, 12.5), types.GateVeto, ReasonProfitTargetHit + ":weekly"},
		{"unknown state vetoes pnl_state_unknown", GateState{}, types.GateVeto, ReasonPnLStateUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := g.Evaluate(GateInput{}, tc.state)
			if eval.Result != tc.want {
				t.Errorf("result = %s, want %s", eval.Result, tc.want)
			}
			if tc.reason != "" && eval.ReasonCodes[0] != tc.reason {
				t.Errorf("reason = %s, want %s", eval.ReasonCodes[0], tc.reason)
			}
		})
	}
}

// ─── R5: MartingaleBanGate ──────────────────────────────────────────────

func TestMartingaleBanGate(t *testing.T) {
	baseLots := map[types.StrategyID]float64{
		"STANDARD_SCALPING": 0.01,
		"TREND_SWING":       0.02,
	}
	g := &MartingaleBanGate{MaxLotRatio: 1.0, BaseLots: baseLots}
	cases := []struct {
		name     string
		strategy types.StrategyID
		lot      float64
		want     types.GateResult
	}{
		{"base lot passes", "STANDARD_SCALPING", 0.01, types.GatePass},
		{"doubled lot vetoed", "STANDARD_SCALPING", 0.02, types.GateVeto},
		{"other strategy base ok", "TREND_SWING", 0.02, types.GatePass},
		{"unconfigured strategy fails closed", "MARNIE_FIB", 0.01, types.GateVeto},
		{"floating point safe at exact ratio", "STANDARD_SCALPING", 0.010000001, types.GatePass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := g.Evaluate(GateInput{StrategyID: tc.strategy, RequestedLot: tc.lot}, capTestState)
			if eval.Result != tc.want {
				t.Errorf("result = %s, want %s", eval.Result, tc.want)
			}
			if eval.Result == types.GateVeto && eval.ReasonCodes[0] != ReasonMartingaleLot {
				t.Errorf("reason = %s, want %s", eval.ReasonCodes[0], ReasonMartingaleLot)
			}
		})
	}
}

// ─── EV1-EV3: EdgeValidationGate ────────────────────────────────────────

func TestEdgeValidationGate(t *testing.T) {
	g := &EdgeValidationGate{MinProfitFactor: 1.2, MinExpectancyR: 0.2, MinSampleSize: 50}

	provenStats := risk.EdgeStats{SampleSize: 60, RComputableCount: 60, ProfitFactor: 1.5, ExpectancyR: 0.3}
	unprovenStats := risk.EdgeStats{SampleSize: 10, RComputableCount: 10, ProfitFactor: 0.9, ExpectancyR: -0.1}

	cases := []struct {
		name  string
		state GateState
		strat types.StrategyID
		want  types.GateResult
	}{
		{
			"proven edge allows executable",
			GateState{Value: map[types.StrategyID]risk.EdgeStats{"STANDARD_SCALPING": provenStats}},
			"STANDARD_SCALPING", types.GatePass,
		},
		{
			"unproven edge forces advisory",
			GateState{Value: map[types.StrategyID]risk.EdgeStats{"STANDARD_SCALPING": unprovenStats}},
			"STANDARD_SCALPING", types.GateDegraded,
		},
		{
			"empty history forces advisory",
			GateState{Value: map[types.StrategyID]risk.EdgeStats{}},
			"TREND_SWING", types.GateDegraded,
		},
		{
			"no hydrated stats (nil value) forces advisory",
			GateState{},
			"STANDARD_SCALPING", types.GateDegraded,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := g.Evaluate(GateInput{StrategyID: tc.strat}, tc.state)
			if eval.Result != tc.want {
				t.Errorf("result = %s, want %s", eval.Result, tc.want)
			}
			if eval.Result == types.GateDegraded && eval.ReasonCodes[0] != ReasonEdgeUnproven {
				t.Errorf("reason = %s, want %s", eval.ReasonCodes[0], ReasonEdgeUnproven)
			}
			if eval.Result == types.GateDegraded {
				// Advisory downgrade must NOT be a hard veto.
				if eval.Result == types.GateVeto {
					t.Error("edge_unproven must never be a hard veto")
				}
			}
		})
	}
}

// ─── Seeding: fail-closed defaults for capital-protection gates ─────────

func TestSeedCapitalProtectionGateStates(t *testing.T) {
	reg := NewRegistry()
	// Mirror production wiring (main.go): standard gates + self-evaluating
	// precision gates registered and seeded, then capital-protection gates
	// inserted at their canonical positions.
	registerAllGates(reg)
	reg.Register(&MinAbsoluteATRGate{MinATR: 2.0})
	reg.Register(&StopHuntFilterGate{MinDistanceATR: 1.5})
	setAllGateStatesPass(reg)
	posCaps := &PositionCapsGate{}
	reg.RegisterOrdered(&WrongSideSLGate{}, types.GateDataQuality)
	reg.RegisterOrdered(&RiskOversizeGate{MaxRiskPerTradePct: 1.5}, types.GateMargin)
	reg.RegisterOrdered(posCaps, types.GateRiskOversize)
	reg.RegisterOrdered(&DailyLossGate{MaxDailyLossPct: 2}, types.GatePositionCaps)
	reg.RegisterOrdered(&ProfitTargetGate{MaxDailyProfitPct: 5}, types.GateDailyLoss)
	reg.RegisterOrdered(&MartingaleBanGate{MaxLotRatio: 1, BaseLots: map[types.StrategyID]float64{"X": 0.01}}, types.GateProfitTarget)
	reg.RegisterOrdered(&EdgeValidationGate{MinSampleSize: 50}, types.GateExecutionPermit)
	now := time.Now()
	reg.UpdateState(types.GateMinATR, GateState{State: types.GatePass, EvaluatedAt: now})
	reg.UpdateState(types.GateStopHuntFilter, GateState{State: types.GatePass, EvaluatedAt: now})
	SeedCapitalProtectionGateStates(reg)

	input := GateInput{
		Tick:               &types.Tick{Quality: types.QualityAuthoritative},
		SessionAllowed:     true,
		NewsRisk:           "LOW",
		Spread:             0.20,
		ATR:                3.0,
		Direction:          types.DirectionBuy,
		EntryPrice:         2430,
		StopLoss:           2426,
		AccountEquity:      10000,
		RequestedLot:       0.01,
		PositionsKnown:     false,
		EntitlementOK:      true,
		LicenseActive:      true,
		ExecutionPermitted: true,
	}
	allPass, evals, _ := reg.EvaluateAll(input)

	if allPass {
		t.Error("must NOT pass while positions are unknown and P&L anchors missing")
	}
	results := map[types.GateID]types.GateResult{}
	for _, e := range evals {
		results[e.GateID] = e.Result
	}
	if results[types.GatePositionCaps] != types.GateDegraded {
		t.Errorf("position_caps = %s, want DEGRADED (positions unknown)", results[types.GatePositionCaps])
	}
	if results[types.GateWrongSideSL] != types.GatePass {
		t.Errorf("wrong_side_sl = %s, want PASS for valid geometry", results[types.GateWrongSideSL])
	}
	// Daily loss no longer hard-vetoes on unknown PnL — it defers to the EA's
	// own MaxDailyLossPct (server-side PnL tracking is a secondary check). The
	// fail-closed posture for missing broker data is carried by position_caps
	// (DEGRADED), which keeps AllGatesPass=false.
	if results[types.GateDailyLoss] != types.GatePass {
		t.Errorf("daily_loss = %s, want PASS when PnL anchor unknown (EA enforces locally)", results[types.GateDailyLoss])
	}
}

// ─── Operator arming / authorization (live auto-trading enablement) ─────────

// TestEdgeValidationGateArmed verifies that an operator-armed strategy passes
// the edge gate without requiring live closed-trade history (breaks the
// bootstrap deadlock), while unarmed strategies still force ADVISORY.
func TestEdgeValidationGateArmed(t *testing.T) {
	g := &EdgeValidationGate{MinProfitFactor: 1.2, MinExpectancyR: 0.2, MinSampleSize: 50}
	g.SetArmed([]string{"STANDARD_SCALPING"})

	// Armed strategy — no stats available → still PASS (armed).
	eval := g.Evaluate(GateInput{StrategyID: types.StrategyID("STANDARD_SCALPING")}, GateState{State: types.GateDegraded})
	if eval.Result != types.GatePass || eval.ReasonCodes[0] != ReasonEdgeArmed {
		t.Errorf("armed strategy: result=%s reason=%v, want PASS/edge_armed", eval.Result, eval.ReasonCodes)
	}

	// Unarmed strategy — no stats → DEGRADED (advisory-only).
	eval2 := g.Evaluate(GateInput{StrategyID: types.StrategyID("ULTRA_SCALPING")}, GateState{State: types.GateDegraded})
	if eval2.Result != types.GateDegraded || eval2.ReasonCodes[0] != ReasonEdgeUnproven {
		t.Errorf("unarmed strategy: result=%s reason=%v, want DEGRADED/edge_unproven", eval2.Result, eval2.ReasonCodes)
	}
}

// TestPositionCapsAuthorized verifies that when the operator has authorized live
// trading and the strategy is armed, a missing broker position snapshot does NOT
// block the signal (EA enforces caps locally). Without authorization it still
// degrades (fail-closed).
func TestPositionCapsAuthorized(t *testing.T) {
	g := &PositionCapsGate{MaxSameDirection: 1, MaxTotal: 2, MaxPerStrategy: 1}
	g.SetAuthorized(true)
	g.SetArmed([]string{"STANDARD_SCALPING"})

	// Authorized + armed, positions unknown → PASS (EA enforces locally).
	eval := g.Evaluate(GateInput{Direction: types.DirectionBuy, StrategyID: types.StrategyID("STANDARD_SCALPING"), PositionsKnown: false}, GateState{})
	if eval.Result != types.GatePass || eval.ReasonCodes[0] != ReasonPositionCapsAuthorized {
		t.Errorf("authorized+armed: result=%s reason=%v, want PASS/position_caps_authorized", eval.Result, eval.ReasonCodes)
	}

	// Authorized but NOT armed → still DEGRADED (positions_unknown).
	eval2 := g.Evaluate(GateInput{Direction: types.DirectionBuy, StrategyID: types.StrategyID("ULTRA_SCALPING"), PositionsKnown: false}, GateState{})
	if eval2.Result != types.GateDegraded {
		t.Errorf("authorized but unarmed: result=%s, want DEGRADED", eval2.Result)
	}

	// Not authorized at all → DEGRADED regardless of arming.
	g2 := &PositionCapsGate{MaxSameDirection: 1, MaxTotal: 2, MaxPerStrategy: 1}
	g2.SetArmed([]string{"STANDARD_SCALPING"})
	eval3 := g2.Evaluate(GateInput{Direction: types.DirectionBuy, StrategyID: types.StrategyID("STANDARD_SCALPING"), PositionsKnown: false}, GateState{})
	if eval3.Result != types.GateDegraded {
		t.Errorf("not authorized: result=%s, want DEGRADED", eval3.Result)
	}
}
