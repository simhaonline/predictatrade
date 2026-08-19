package signal

import (
	"context"
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/cache"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// === Cooldown Tests (SOW Section 17) ===

func TestCooldownManager_NoCache_AllowsAll(t *testing.T) {
	cm := NewCooldownManager(nil)
	active, _, err := cm.CheckCooldown(context.Background(), "XAUUSD", types.StrategyStandardScalping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active {
		t.Error("cooldown should not be active when cache is nil (test mode)")
	}
}

func TestCooldownManager_SetAndCheck(t *testing.T) {
	vc := tryConnectValkey()
	if vc == nil {
		t.Skip("Valkey not available — skipping integration test")
	}
	defer vc.Close()

	cm := NewCooldownManager(vc)
	ctx := context.Background()

	// Clear any existing cooldown from previous test runs
	for _, strat := range []types.StrategyID{types.StrategyStandardScalping, types.StrategyStandardSwing, types.StrategyUltraScalping, types.StrategyTrendSwing} {
		vc.ClearCooldown(ctx, "XAUUSD", string(strat))
	}

	// Check — should not be active
	active, _, err := cm.CheckCooldown(ctx, "XAUUSD", types.StrategyStandardScalping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active {
		t.Error("cooldown should not be active initially")
	}

	// Set cooldown for 1 minute
	err = cm.SetCooldown(ctx, "XAUUSD", types.StrategyStandardScalping, 1)
	if err != nil {
		t.Fatalf("unexpected error setting cooldown: %v", err)
	}

	// Check — should be active
	active, remaining, err := cm.CheckCooldown(ctx, "XAUUSD", types.StrategyStandardScalping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !active {
		t.Error("cooldown should be active after setting")
	}
	if remaining <= 0 || remaining > time.Minute {
		t.Errorf("expected remaining ~60s, got %v", remaining)
	}

	// Different strategy should NOT be affected
	active2, _, err := cm.CheckCooldown(ctx, "XAUUSD", types.StrategyStandardSwing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active2 {
		t.Error("cooldown for STANDARD_SCALPING should not affect STANDARD_SWING")
	}
}

// === Duplicate Prevention Tests (SOW Section 18) ===

func TestFingerprint_Deterministic(t *testing.T) {
	fp1 := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})
	fp2 := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})

	if fp1 != fp2 {
		t.Error("fingerprints should be deterministic for same inputs")
	}
}

func TestFingerprint_DifferentDirection(t *testing.T) {
	fpBuy := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})
	fpSell := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionSell,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})

	if fpBuy == fpSell {
		t.Error("fingerprints should differ for different directions")
	}
}

func TestFingerprint_DifferentStrategy(t *testing.T) {
	fp1 := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})
	fp2 := ComputeFingerprint("XAUUSD", types.StrategyUltraScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})

	if fp1 == fp2 {
		t.Error("fingerprints should differ for different strategies")
	}
}

func TestFingerprint_MicroPriceChange_Ignored(t *testing.T) {
	// Prices rounded to 2 decimals — micro-changes should produce same fingerprint
	fp1 := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.501), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})
	fp2 := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.504), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})

	if fp1 != fp2 {
		t.Error("micro price changes (within rounding) should produce same fingerprint")
	}
}

func TestFingerprint_StructuralChange_Different(t *testing.T) {
	fp1 := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC), time.Time{})
	fp2 := ComputeFingerprint("XAUUSD", types.StrategyStandardScalping, types.DirectionBuy,
		decimal.NewFromFloat(2000.50), decimal.NewFromFloat(1995.00),
		time.Date(2024, 6, 3, 14, 0, 0, 0, time.UTC), time.Time{})

	if fp1 == fp2 {
		t.Error("different structural anchors should produce different fingerprints")
	}
}

func TestDuplicateChecker_NoCache_AllowsAll(t *testing.T) {
	dc := NewDuplicateChecker(nil)
	isNew, err := dc.CheckDuplicate(context.Background(), "test-fp", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isNew {
		t.Error("should allow when cache is nil (test mode)")
	}
}

func TestDuplicateChecker_SetAndCheck(t *testing.T) {
	vc := tryConnectValkey()
	if vc == nil {
		t.Skip("Valkey not available — skipping integration test")
	}
	defer vc.Close()

	dc := NewDuplicateChecker(vc)
	ctx := context.Background()
	fp := "test-fp-duplicate-check"

	// Clear any existing fingerprint
	vc.ClearFingerprint(ctx, fp)

	// First check — should be new
	isNew, err := dc.CheckDuplicate(ctx, fp, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isNew {
		t.Error("first check should be new signal")
	}

	// Second check — should be duplicate
	isNew, err = dc.CheckDuplicate(ctx, fp, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isNew {
		t.Error("second check should be duplicate")
	}
}

// tryConnectValkey attempts to connect to a local Valkey instance for integration tests.
func tryConnectValkey() *cache.ValkeyCache {
	v := cache.NewValkeyCache("127.0.0.1:6379")
	if err := v.Ping(); err != nil {
		v.Close()
		return nil
	}
	return v
}
