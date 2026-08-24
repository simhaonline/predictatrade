package risk

import (
	"math"
	"testing"
	"time"
)

// ─── Sizing (R1/R7, MA1-MA2) ────────────────────────────────────────────

func TestSuggestedLot(t *testing.T) {
	cases := []struct {
		name     string
		equity   float64
		pct      float64
		dist     float64
		wantLots float64
	}{
		// $10k equity × 1.5% = $150; $2 stop → $200/lot → floor(0.75) = 0.75
		{"standard", 10000, 1.5, 2.0, 0.75},
		// rounds DOWN to lot step
		{"rounds down", 10000, 1.5, 1.5, 0.50}, // $150/300 = 0.5
		{"too small returns zero", 100, 1.5, 30.0, 0}, // $1.5/$3000 → 0
		{"zero equity", 0, 1.5, 2.0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SuggestedLot(tc.equity, tc.pct, tc.dist, SymbolEconomics{})
			if math.Abs(got-tc.wantLots) > 1e-9 {
				t.Errorf("SuggestedLot = %v, want %v", got, tc.wantLots)
			}
		})
	}
}

func TestRiskPerLotDefaultsMatchContractMath(t *testing.T) {
	// XAUUSD defaults: tickValue $1 / tickSize 0.01 → $100 per $1 move per lot.
	if got := RiskPerLot(1.0, SymbolEconomics{}); got != 100 {
		t.Errorf("RiskPerLot($1 dist) = %v, want 100 (100oz contract)", got)
	}
}

func TestMarginAwareLotCap(t *testing.T) {
	// requiredMargin = lot×100×price/leverage
	// 0.10 lot @ 2400 @ 1:500 → 0.10×100×2400/500 = $48; budget = 1000×0.3 = $300 → OK
	mc := MarginAwareLotCap(1000, 1000, 0.10, 2400, 500)
	if !mc.Allowed {
		t.Errorf("expected allowed, got %+v", mc)
	}
	// 1.00 lot @ 2400 @ 1:100 → 1×100×2400/100 = $2400 > $300 budget
	mc = MarginAwareLotCap(1000, 1000, 1.0, 2400, 100)
	if mc.Allowed || mc.Reason != "margin_exceeded" {
		t.Errorf("expected margin_exceeded, got %+v", mc)
	}
	// capped lot: floor(300×100/(100×2400)/0.01)*0.01 = floor(0.125→12.5 steps)=0.12
	if math.Abs(mc.CappedLot-0.12) > 1e-9 {
		t.Errorf("CappedLot = %v, want 0.12", mc.CappedLot)
	}
	// no free margin fails closed
	mc = MarginAwareLotCap(1000, 0, 0.01, 2400, 500)
	if mc.Allowed || mc.Reason != "margin_state_unknown" {
		t.Errorf("zero free margin must fail closed, got %+v", mc)
	}
}

func TestComputeSizingVetoRule(t *testing.T) {
	// Veto only when over cap AND suggested lot == 0.
	res := ComputeSizing(1000, 1.5, 2430, 2380, 0.01, SymbolEconomics{})
	if !res.VetoOversize {
		t.Errorf("account too small for $50 stop should veto, got %+v", res)
	}
	res = ComputeSizing(10000, 1.5, 2430, 2428, 1.0, SymbolEconomics{})
	if res.VetoOversize || !res.Oversize {
		t.Errorf("oversize-but-viable should annotate not veto: %+v", res)
	}
	if res.SuggestedLot <= 0 {
		t.Errorf("suggested lot should be viable: %v", res.SuggestedLot)
	}
	if math.Abs(res.SLDistancePoints-2.0) > 1e-9 {
		t.Errorf("SLDistancePoints = %v, want 2.0", res.SLDistancePoints)
	}
}

// ─── P&L anchors (R4) ──────────────────────────────────────────────────

type memStore struct {
	data map[string][]byte
	fail bool
}

func (m *memStore) Get(key string) ([]byte, error) {
	if m.fail {
		return nil, errStoreFailed
	}
	v := m.data[key]
	return v, nil // missing key → empty bytes, handled by caller
}

func (m *memStore) SetNX(key string, val []byte, _ time.Duration) (bool, error) {
	if m.fail {
		return false, errStoreFailed
	}
	if m.data == nil {
		m.data = map[string][]byte{}
	}
	if _, exists := m.data[key]; exists {
		return false, nil
	}
	m.data[key] = val
	return true, nil
}

var errStoreFailed = &storeError{}

type storeError struct{}

func (*storeError) Error() string { return "store failed" }

func TestPeriodID(t *testing.T) {
	now := time.Date(2026, 8, 24, 15, 4, 5, 0, time.UTC) // Monday, ISO week 35
	if got := PeriodID(now, PeriodDay); got != "2026-08-24" {
		t.Errorf("day = %s", got)
	}
	if got := PeriodID(now, PeriodWeek); got != "2026-W35" {
		t.Errorf("week = %s, want 2026-W35", got)
	}
	if got := PeriodID(now, PeriodMonth); got != "2026-08" {
		t.Errorf("month = %s", got)
	}
	// ISO week rollover: Jan 1 2026 is a Thursday → ISO week 1 of 2026.
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := PeriodID(jan1, PeriodWeek); got != "2026-W01" {
		t.Errorf("week = %s, want 2026-W01", got)
	}
}

func TestPnLTrackerAnchorsAndRollover(t *testing.T) {
	store := &memStore{}
	tracker := NewPnLTracker(store)
	day1 := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)

	snap := tracker.Update(1000, day1)
	if !snap.Known || snap.PeriodPc[PeriodDay] != 0 {
		t.Errorf("first observation should anchor at 0%%: %+v", snap)
	}

	// Equity drops to 970 on the same day → −3% day, −3% week/month.
	snap = tracker.Update(970, day1.Add(time.Hour))
	if !snap.Known {
		t.Fatal("snapshot should be known")
	}
	if math.Abs(snap.PeriodPc[PeriodDay]+3) > 1e-9 {
		t.Errorf("day pct = %v, want -3", snap.PeriodPc[PeriodDay])
	}
	if math.Abs(snap.PeriodPc[PeriodMonth]+3) > 1e-9 {
		t.Errorf("month pct = %v, want -3", snap.PeriodPc[PeriodMonth])
	}

	// Next UTC day: daily anchor rolls over (pct resets vs new day anchor);
	// weekly and monthly anchors persist (start-of-period equity = 1000).
	day2 := time.Date(2026, 8, 25, 0, 30, 0, 0, time.UTC)
	snap = tracker.Update(1000, day2)
	if !snap.Known {
		t.Fatal("snapshot should be known after rollover")
	}
	if snap.PeriodPc[PeriodDay] != 0 {
		t.Errorf("new day should re-anchor to 0%%, got %v", snap.PeriodPc[PeriodDay])
	}
	if math.Abs(snap.PeriodPc[PeriodWeek]) > 1e-9 || math.Abs(snap.PeriodPc[PeriodMonth]) > 1e-9 {
		t.Errorf("week/month pct measured against their own anchors should be 0%%, got week=%v month=%v",
			snap.PeriodPc[PeriodWeek], snap.PeriodPc[PeriodMonth])
	}
	// Weekly loss persists relative to the Monday anchor when equity drops again.
	snap = tracker.Update(950, day2.Add(time.Hour))
	if math.Abs(snap.PeriodPc[PeriodWeek]+5) > 1e-9 {
		t.Errorf("week pct = %v, want -5 vs Monday anchor", snap.PeriodPc[PeriodWeek])
	}
	if snap.PeriodPc[PeriodDay] != 0 {
		t.Errorf("day pct should reset on the new day's anchor, got %v", snap.PeriodPc[PeriodDay])
	}
}

func TestPnLTrackerFailClosedOnStoreFailure(t *testing.T) {
	tracker := NewPnLTracker(&memStore{fail: true})
	snap := tracker.Update(1000, time.Now())
	if snap.Known {
		t.Error("store failure must produce Known=false (fail-closed)")
	}
	var nilTracker *PnLTracker
	snap = nilTracker.Update(1000, time.Now())
	if snap.Known {
		t.Error("nil tracker must fail closed")
	}
}

func TestPnLTrackerNonPositiveEquityFailsClosed(t *testing.T) {
	tracker := NewPnLTracker(&memStore{})
	for _, eq := range []float64{0, -5} {
		if snap := tracker.Update(eq, time.Now()); snap.Known {
			t.Errorf("equity=%v must be unknown", eq)
		}
	}
}

// ─── Edge stats (EV1-EV3) ──────────────────────────────────────────────

func mkTrade(dir string, pnl, entry, sl, lot float64) TradeRecord {
	return TradeRecord{StrategyID: "S", Direction: dir, PnL: pnl,
		EntryPrice: entry, StopLoss: sl, LotSize: lot}
}

func TestComputeEdgeStatsProven(t *testing.T) {
	// 60 trades risking $200 each ($2 SL × 1.0 lot): 60% win rate at +$300/−$100
	// → PF = 36×300/(24×100) = 4.5; expectancy = (0.6×1.5 − 0.4×0.5) = 0.7R
	var trades []TradeRecord
	for i := 0; i < 60; i++ {
		if i%5 < 3 { // 36 wins
			trades = append(trades, mkTrade("BUY", 300, 2400, 2398, 1.0))
		} else {
			trades = append(trades, mkTrade("BUY", -100, 2400, 2398, 1.0))
		}
	}
	stats := ComputeEdgeStats(trades)
	if stats.SampleSize != 60 || stats.Wins != 36 || stats.Losses != 24 {
		t.Fatalf("unexpected counts: %+v", stats)
	}
	if stats.ProfitFactor < 4.49 || stats.ProfitFactor > 4.51 {
		t.Errorf("PF = %v, want ~4.5", stats.ProfitFactor)
	}
	if math.Abs(stats.ExpectancyR-0.7) > 1e-9 {
		t.Errorf("expectancy = %v, want 0.7", stats.ExpectancyR)
	}
	if !stats.IsProven(1.2, 0.2, 50) {
		t.Error("edge should be proven")
	}
}

func TestComputeEdgeStatsShortExpectancy(t *testing.T) {
	trades := []TradeRecord{
		mkTrade("SELL", 200, 2400, 2402, 1.0), // +1R
		mkTrade("SELL", 100, 2400, 2402, 1.0), // +0.5R
		mkTrade("BUY", -200, 2400, 2398, 1.0),
	}
	stats := ComputeEdgeStats(trades)
	if math.Abs(stats.ShortExpectancyR-0.75) > 1e-9 {
		t.Errorf("short expectancy = %v, want 0.75", stats.ShortExpectancyR)
	}
	if stats.ShortSampleSize != 2 {
		t.Errorf("short sample = %d, want 2", stats.ShortSampleSize)
	}
}

func TestEdgeNotProvenBelowThresholds(t *testing.T) {
	stats := EdgeStats{SampleSize: 49, RComputableCount: 49, ProfitFactor: 3.0, ExpectancyR: 1.0}
	if stats.IsProven(1.2, 0.2, 50) {
		t.Error("sample below minimum must not be proven")
	}
	stats = EdgeStats{SampleSize: 60, RComputableCount: 20, ProfitFactor: 3.0, ExpectancyR: 1.0}
	if stats.IsProven(1.2, 0.2, 50) {
		t.Error("expectancy evidenced on a subset of the sample must not be proven")
	}
	stats = EdgeStats{SampleSize: 60, RComputableCount: 60, ProfitFactor: 1.0, ExpectancyR: 0.5}
	if stats.IsProven(1.2, 0.2, 50) {
		t.Error("profit factor below threshold must not be proven")
	}
}

func TestComputeEdgeStatsEmptyHistory(t *testing.T) {
	stats := ComputeEdgeStats(nil)
	if stats.SampleSize != 0 || stats.IsProven(1.2, 0.2, 50) {
		t.Error("empty history must never be proven")
	}
}
