package risk

import (
	"testing"
	"time"
)

// P3 (prompt.md): MaxSlUSD hard cap — SL distance can never exceed the cap.
// Pure; returns the (possibly clamped) distance and whether clamping happened.
func TestMaxSlUSDCap(t *testing.T) {
	cap8 := 8.0
	entry := 2400.0
	// SL 2410 → distance 10 > 8 → clamp to 8 (SL pulled to 2410-8=2402... for BUY SL is below:
	// cap applies to the DISTANCE only; caller picks the side).
	d, clamped := MaxSlUSDCap(entry, entry-10, cap8)
	if !clamped || d != cap8 {
		t.Fatalf("distance 10 must clamp to 8, got %v clamped=%v", d, clamped)
	}
	// BUY: SL below entry → new SL = entry − capped distance
	slNew := entry - d
	if slNew != 2392 {
		t.Fatalf("clamped SL = %v, want 2392", slNew)
	}
	// Within cap → untouched
	d2, clamped2 := MaxSlUSDCap(entry, entry-5, cap8)
	if clamped2 || d2 != 5 {
		t.Fatalf("distance 5 within cap 8 must pass through, got %v clamped=%v", d2, clamped2)
	}
	// Cap disabled (0) → pass-through
	d3, clamped3 := MaxSlUSDCap(entry, entry-50, 0)
	if clamped3 || d3 != 50 {
		t.Fatalf("cap 0 = disabled, got %v clamped=%v", d3, clamped3)
	}
	// Negative/zero SL → pass-through (invalid geometry handled upstream)
	d4, _ := MaxSlUSDCap(entry, 0, cap8)
	if d4 != entry {
		t.Fatalf("zero SL → distance = entry, got %v", d4)
	}
}

// P3: broker-constraint SL floor — distance must satisfy
// max(stopsLevelPoints+2, spread×1.5) as a MINIMUM (reference BuildTradePlan).
// This is a floor (widen), complementary to the MaxSlUSD cap (narrow) — both
// can apply: floor first, then cap.
func TestBrokerConstraintSLFloor(t *testing.T) {
	entry := 2400.0
	spread := 0.5
	stopsLevelPts := 15
	point := 0.1
	// floor = max((15+2)×0.1, 0.5×1.5) = max(1.7, 0.75) = 1.7
	floor := SLFloorPoints(entry, spread, stopsLevelPts, point)
	if floor < 1.699 || floor > 1.701 {
		t.Fatalf("floor = 1.7, got %v", floor)
	}
	// SL thinner than the floor → widened
	sl := entry - 1.0
	d := entry - sl
	got := EnforceSLFloor(entry, sl, floor)
	if got != entry-1.7 {
		t.Fatalf("SL widened to the floor: got %v", got)
	}
	_ = d
	// SL already beyond floor → unchanged
	slOk := entry - 3.0
	if EnforceSLFloor(entry, slOk, floor) != slOk {
		t.Fatal("SL beyond floor must be unchanged")
	}
}

// P3: Friday flatten — decision helper (server-side mirror of the EA's
// FridayCloseHour): after the cutoff on Friday, new entries are blocked.
func TestFridayFlatten(t *testing.T) {
	// Friday 2026-09-18 21:00 UTC after cutoff 20:00 → block
	fri := time.Date(2026, 9, 18, 21, 0, 0, 0, time.UTC)
	if !ShouldFlattenFriday(fri, 20) {
		t.Fatal("Friday 21:00 after 20:00 cutoff → flatten")
	}
	// Friday 10:00 → allow
	friAm := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	if ShouldFlattenFriday(friAm, 20) {
		t.Fatal("Friday morning → no flatten")
	}
	// Thursday → never
	thu := time.Date(2026, 9, 17, 21, 0, 0, 0, time.UTC)
	if ShouldFlattenFriday(thu, 20) {
		t.Fatal("Thursday → no flatten")
	}
	// Cutoff 0 = disabled
	if ShouldFlattenFriday(fri, 0) {
		t.Fatal("cutoff 0 disables the rule")
	}
}
