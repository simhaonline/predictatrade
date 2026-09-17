package features

import (
	"testing"

	"github.com/shopspring/decimal"
)

// P1 (prompt.md): CLV = (2C − H − L) / (H − L), 0 when H == L (doji-flat bar).
// 1 = close at high, 0 = close at mid, −1 = close at low.
func TestCloseLocationValue(t *testing.T) {
	h := decimal.NewFromFloat(100)
	l := decimal.NewFromFloat(90)

	if v, _ := CloseLocationValue(h, l, h).Float64(); v != 1 {
		t.Fatalf("close at high → CLV 1, got %v", v)
	}
	if v, _ := CloseLocationValue(h, l, l).Float64(); v != -1 {
		t.Fatalf("close at low → CLV −1, got %v", v)
	}
	if v, _ := CloseLocationValue(h, l, decimal.NewFromFloat(95)).Float64(); v != 0 {
		t.Fatalf("close at mid → CLV 0, got %v", v)
	}
	// flat bar: H == L → 0 (documented), never NaN/panic
	if v, _ := CloseLocationValue(h, h, h).Float64(); v != 0 {
		t.Fatalf("H==L → 0, got %v", v)
	}
}

// P1: bull/bear streaks — consecutive closes in one direction; 0 when flat.
func TestCloseStreak(t *testing.T) {
	up := []decimal.Decimal{
		decimal.NewFromInt(1), decimal.NewFromInt(2), decimal.NewFromInt(3),
	}
	// streak counts bars whose close > prior close (2 rising deltas)
	if s := CloseStreak(up); s != 2 {
		t.Fatalf("rising closes → bullStreak 2, got %d", s)
	}
	down := []decimal.Decimal{
		decimal.NewFromInt(3), decimal.NewFromInt(2), decimal.NewFromInt(1),
	}
	if s := CloseStreak(down); s != -2 {
		t.Fatalf("falling closes → bearStreak −2, got %d", s)
	}
	mixed := []decimal.Decimal{
		decimal.NewFromInt(1), decimal.NewFromInt(2), decimal.NewFromInt(2), decimal.NewFromInt(3),
	}
	if s := CloseStreak(mixed); s != 1 {
		t.Fatalf("flat break resets streak → 1, got %d", s)
	}
	if s := CloseStreak(nil); s != 0 {
		t.Fatalf("empty → 0, got %d", s)
	}
}

// P1: micro-reclaim — a wick pierces a level then the CLOSE is back on the
// original side. Bull reclaim: low < level AND close > level.
func TestMicroReclaim(t *testing.T) {
	h := decimal.NewFromFloat(101)
	l := decimal.NewFromFloat(99)
	c := decimal.NewFromFloat(100.5)
	level := decimal.NewFromFloat(100)

	if !MicroReclaim(h, l, c, level, true) {
		t.Fatal("bull reclaim: low pierced below 100, close back above → true")
	}
	if MicroReclaim(h, l, c, level, false) {
		t.Fatal("bear reclaim: high pierced above 100 but close also above → false (no bear reclaim)")
	}
	// bear reclaim: high > level AND close < level
	ch, cl, cc := decimal.NewFromFloat(101), decimal.NewFromFloat(99), decimal.NewFromFloat(99.5)
	if !MicroReclaim(ch, cl, cc, level, false) {
		t.Fatal("bear reclaim: high pierced above 100, close back below → true")
	}
	// no pierce → false
	if MicroReclaim(decimal.NewFromFloat(100.5), decimal.NewFromFloat(100.2), c, level, true) {
		t.Fatal("wick did not pierce the level → false")
	}
}
