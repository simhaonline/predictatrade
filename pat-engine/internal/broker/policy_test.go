package broker

import "testing"

func TestStrategyAllowedScalping(t *testing.T) {
	allow := &BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true}
	if ok, _ := allow.StrategyAllowed("ULTRA_SCALPING"); !ok {
		t.Fatalf("scalping allowed: ULTRA_SCALPING should be allowed")
	}
	if ok, _ := allow.StrategyAllowed("TREND_SWING"); !ok {
		t.Fatalf("non-scalping strategy should always be allowed")
	}

	forbid := &BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: false}
	if ok, why := forbid.StrategyAllowed("ULTRA_SCALPING"); ok {
		t.Fatalf("scalping forbidden: ULTRA_SCALPING must be blocked")
	} else if why != "BROKER_SCALPING_NOT_ALLOWED" {
		t.Fatalf("unexpected reason %q", why)
	}
	if ok, _ := forbid.StrategyAllowed("TREND_SWING"); !ok {
		t.Fatalf("non-scalping strategy must remain allowed under no-scalping broker")
	}
}
