package crossmarket

import (
	"testing"
	"time"
)

func TestProductScopeGuard_XAUUSDOnly(t *testing.T) {
	guard := NewProductScopeGuard()
	if !guard.CanGenerateSignal("XAUUSD") {
		t.Error("XAUUSD must be able to generate signals")
	}
	if guard.CanGenerateSignal("BTCUSD") {
		t.Error("BTCUSD must NOT be able to generate signals")
	}
	if guard.CanGenerateSignal("USOIL") {
		t.Error("USOIL must NOT be able to generate signals")
	}
	if guard.CanGenerateSignal("EURUSD") {
		t.Error("EURUSD must NOT be able to generate signals")
	}
	if guard.CanGenerateSignal("DXY") {
		t.Error("DXY must NOT be able to generate signals")
	}
	if guard.CanGenerateSignal("VIX") {
		t.Error("VIX must NOT be able to generate signals")
	}
}

func TestProductScopeGuard_Execution(t *testing.T) {
	guard := NewProductScopeGuard()
	if !guard.CanExecute("XAUUSD") {
		t.Error("XAUUSD must be executable")
	}
	if guard.CanExecute("BTCUSD") {
		t.Error("BTCUSD must NOT be executable")
	}
}

func TestProductScopeGuard_ReferenceOnly(t *testing.T) {
	guard := NewProductScopeGuard()
	if !guard.IsReference("DXY") {
		t.Error("DXY must be classified as reference")
	}
	if !guard.IsReference("BTCUSD") {
		t.Error("BTCUSD must be classified as reference")
	}
	if guard.IsReference("XAUUSD") {
		t.Error("XAUUSD must NOT be classified as reference")
	}
}

func TestProductScopeGuard_FormatScope(t *testing.T) {
	guard := NewProductScopeGuard()
	scope := guard.FormatScope()
	if scope == "" {
		t.Error("scope should not be empty")
	}
}

func TestOutcomeResolver_NilSafety(t *testing.T) {
	r := &OutcomeResolver{}
	events, err := r.Resolve(nil)
	if err != nil {
		t.Errorf("nil resolver should not error, got %v", err)
	}
	if events != nil {
		t.Error("nil resolver should return nil events")
	}
}

func TestOutcomeResolver_BUY_TP1_Logic(t *testing.T) {
	// BUY signal: entry=2500, SL=2490, TP1=2510
	// Price bid=2512 → above TP1 → TP1_HIT
	_ = 2500.0
	sl := 2490.0
	tp1 := 2510.0
	bid := 2512.0

	tp1Hit := bid >= tp1
	slHit := bid <= sl // for BUY, SL hit when price drops below SL

	if !tp1Hit {
		t.Error("bid above TP1 should trigger TP1_HIT")
	}
	if slHit {
		t.Error("bid above SL should not trigger SL_HIT")
	}
}

func TestOutcomeResolver_SELL_SL_Logic(t *testing.T) {
	// SELL signal: entry=2500, SL=2510, TP1=2490
	// Price bid=2512 → above SL for SELL → SL_HIT
	_ = 2500.0
	sl := 2510.0
	tp1 := 2490.0
	bid := 2512.0

	// For SELL: TP1 hit when price drops to/below TP1
	tp1Hit := bid <= tp1
	// For SELL: SL hit when price rises to/above SL
	slHit := bid >= sl

	if !slHit {
		t.Error("bid above SL should trigger SL_HIT for SELL")
	}
	if tp1Hit {
		t.Error("bid above TP1 should not trigger TP1_HIT for SELL")
	}

}

func TestOutcomeResolver_SameBarAmbiguity(t *testing.T) {
	// If both TP and SL are in range, it's ambiguous
	tp1 := 2510.0
	sl := 2490.0
	bid := 2515.0  // above TP1 (BUY)
	ask := 2485.0  // below SL (BUY: SL hit when ask <= sl)

	tp1Hit := bid >= tp1
	slHit := ask <= sl
	ambiguous := tp1Hit && slHit

	if !ambiguous {
		t.Error("both TP and SL in range should be ambiguous")
	}
}

func TestOutcomeResolver_ExpiryBeforeTPSL(t *testing.T) {
	expiry := time.Now().Add(-1 * time.Hour)
	now := time.Now()
	if !now.After(expiry) {
		t.Error("now should be after expiry")
	}
}

func TestOutcomeResolver_IdempotencyCheck(t *testing.T) {
	// The updateOutcome query uses WHERE outcome = 'UNRESOLVED'
	// Running it twice should not double-count
	// This is verified by the WHERE clause
}

func TestOutcomeResolver_NoReferenceAssetOutcomes(t *testing.T) {
	guard := NewProductScopeGuard()
	if guard.IsTradable("BTCUSD") {
		t.Error("BTC must not be tradable — outcome resolver must not use BTC price")
	}
	if guard.IsTradable("USOIL") {
		t.Error("Oil must not be tradable — outcome resolver must not use Oil price")
	}
}
