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
		// Operator-authorized (2026-09-11): unknown snapshot + ZERO issued/open
		// positions is trivially within every cap → PASS, not DEGRADED.
		{"positions unknown zero issued passes", buyInput(0, 0, false), types.GatePass},
		// Unknown snapshot WITH reported open positions still degrades.
		{"positions unknown with open positions degrades", buyInput(1, 0, false), types.GateDegraded},
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
		// Soft recovery policy: loss within cap enters recovery mode (PASS, sized
		// down by recovery manager) instead of a hard halt.
		{"daily recovery band at -2%", snap(-2.0, -1, -2, true), types.GatePass, ReasonDailyLossHalt + ":daily:recovery"},
		{"weekly recovery band at -4.5%", snap(-1, -4.5, -2, true), types.GatePass, ReasonDailyLossHalt + ":weekly:recovery"},
		{"monthly recovery band at -6%", snap(-1, -2, -6, true), types.GatePass, ReasonDailyLossHalt + ":monthly:recovery"},
		// Severe blowout (loss ≥ cap × multiplier, default 2×) hard-halts.
		{"daily severe halt at -5%", snap(-5.0, -1, -2, true), types.GateVeto, ReasonDailyLossHalt + ":daily:severe"},
		{"weekly severe halt at -9%", snap(-1, -9, -2, true), types.GateVeto, ReasonDailyLossHalt + ":weekly:severe"},
		{"monthly severe halt at -11%", snap(-1, -2, -11, true), types.GateVeto, ReasonDailyLossHalt + ":monthly:severe"},
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
		// Operator-authorized (2026-09-11): unconfigured base lot falls back to
		// the XAUUSD minimum (0.01) instead of hard-vetoing every candidate.
		// Martingale protection is preserved: any lot above base×ratio still vetoes.
		{"unconfigured strategy falls back to min lot", "MARNIE_FIB", 0.01, types.GatePass},
		{"unconfigured strategy martingale lot still vetoed", "MARNIE_FIB", 0.02, types.GateVeto},
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
		TakeProfit1:        2438, // 2R target so the RR gate passes and evaluation reaches edge_validation
		AccountEquity:      10000,
		RequestedLot:       0.01,
		PositionsKnown:     false,
		EntitlementOK:      true,
		LicenseActive:      true,
		ExecutionPermitted: true,
	}
	allPass, evals, _ := reg.EvaluateAll(input)

	// The bootstrap-deadlock edge gate (no live closed-trade stats yet) must
	// still force DEGRADED so a fresh engine cannot go executable immediately.
	if allPass {
		t.Error("must NOT pass all gates while edge-validation stats are unhydrated")
	}
	results := map[types.GateID]types.GateResult{}
	for _, e := range evals {
		results[e.GateID] = e.Result
	}
	// Operator-authorized (2026-09-11): unknown positions + zero issued → PASS
	// (trivially within caps); a fresh account is seeded with KNOWN zero PnL
	// anchors so daily/weekly/monthly caps hold at 0 loss until the tracker
	// hydrates real values.
	if results[types.GatePositionCaps] != types.GatePass {
		t.Errorf("position_caps = %s, want PASS (unknown + zero issued)", results[types.GatePositionCaps])
	}
	if results[types.GateWrongSideSL] != types.GatePass {
		t.Errorf("wrong_side_sl = %s, want PASS for valid geometry", results[types.GateWrongSideSL])
	}
	if results[types.GateDailyLoss] != types.GatePass {
		t.Errorf("daily_loss = %s, want PASS (fresh account seeded with zero PnL anchors)", results[types.GateDailyLoss])
	}
	if results[types.GateEdgeValidation] != types.GateDegraded {
		t.Errorf("edge_validation = %s, want DEGRADED (bootstrap deadlock until live stats hydrate)", results[types.GateEdgeValidation])
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

// TestPositionCapsAuthorized verifies the 2026-09-11 operator-authorized
// semantics: an unknown broker position snapshot with ZERO reported/issued
// positions is trivially within every cap → PASS regardless of authorization
// (the authorized/armed flags are no longer consulted by Evaluate). Real
// over-cap exposure is still caught whenever PositionsKnown=true, and unknown
// WITH reported open positions still degrades.
func TestPositionCapsAuthorized(t *testing.T) {
	g := &PositionCapsGate{MaxSameDirection: 1, MaxTotal: 2, MaxPerStrategy: 1}
	g.SetAuthorized(true)
	g.SetArmed([]string{"STANDARD_SCALPING"})

	// Unknown + zero issued: trivially within caps → PASS (operator-authorized).
	eval := g.Evaluate(GateInput{Direction: types.DirectionBuy, StrategyID: types.StrategyID("STANDARD_SCALPING"), PositionsKnown: false}, GateState{})
	if eval.Result != types.GatePass || eval.ReasonCodes[0] != "positions_unknown_zero_issued" {
		t.Errorf("authorized+armed zero-issued: result=%s reason=%v, want PASS/positions_unknown_zero_issued", eval.Result, eval.ReasonCodes)
	}

	// Unknown WITH reported open positions → DEGRADED (fail-closed).
	eval2 := g.Evaluate(GateInput{Direction: types.DirectionBuy, StrategyID: types.StrategyID("STANDARD_SCALPING"),
		PositionsKnown: false, OpenBuyPositions: 1}, GateState{})
	if eval2.Result != types.GateDegraded || eval2.ReasonCodes[0] != "positions_unknown" {
		t.Errorf("unknown with open positions: result=%s reason=%v, want DEGRADED/positions_unknown", eval2.Result, eval2.ReasonCodes)
	}

	// Verified positions still enforce caps: 1 open buy + max 1 same-direction → VETO.
	g3 := &PositionCapsGate{MaxSameDirection: 1, MaxTotal: 2, MaxPerStrategy: 1}
	eval3 := g3.Evaluate(GateInput{Direction: types.DirectionBuy, StrategyID: types.StrategyID("STANDARD_SCALPING"),
		PositionsKnown: true, OpenBuyPositions: 1, OpenSellPositions: 0}, GateState{})
	if eval3.Result != types.GateVeto {
		t.Errorf("verified same-direction cap: result=%s, want VETO", eval3.Result)
	}
}
