package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func TestValidateGeometry_BuyValid(t *testing.T) {
	cfg := strategy.StrategyConfig{ATRMultiplierSL: 2.0, ATRMultiplierTP1: 2.0, ATRMultiplierTP2: 4.0, ATRMultiplierTP3: 8.0}
	valid, reason := ValidateGeometry(types.DirectionBuy,
		decimal.NewFromFloat(2400), decimal.NewFromFloat(2396),
		decimal.NewFromFloat(2404), decimal.NewFromFloat(2408), decimal.NewFromFloat(2416), cfg)
	if !valid {
		t.Errorf("Valid BUY geometry should pass: %s", reason)
	}
}

func TestValidateGeometry_SellValid(t *testing.T) {
	cfg := strategy.StrategyConfig{ATRMultiplierSL: 2.0, ATRMultiplierTP1: 2.0, ATRMultiplierTP2: 4.0, ATRMultiplierTP3: 8.0}
	valid, reason := ValidateGeometry(types.DirectionSell,
		decimal.NewFromFloat(2400), decimal.NewFromFloat(2404),
		decimal.NewFromFloat(2396), decimal.NewFromFloat(2392), decimal.NewFromFloat(2384), cfg)
	if !valid {
		t.Errorf("Valid SELL geometry should pass: %s", reason)
	}
}

func TestValidateGeometry_SLOnWrongSide(t *testing.T) {
	cfg := strategy.StrategyConfig{ATRMultiplierSL: 2.0, ATRMultiplierTP1: 2.0}
	valid, reason := ValidateGeometry(types.DirectionBuy,
		decimal.NewFromFloat(2400), decimal.NewFromFloat(2405),
		decimal.NewFromFloat(2404), decimal.NewFromFloat(2408), decimal.NewFromFloat(2416), cfg)
	if valid {
		t.Error("SL above entry for BUY should fail")
	}
	if reason == "" {
		t.Error("Should have failure reason")
	}
}

func TestValidateGeometry_TPOrderWrong(t *testing.T) {
	cfg := strategy.StrategyConfig{ATRMultiplierSL: 2.0, ATRMultiplierTP1: 2.0, ATRMultiplierTP2: 4.0}
	valid, _ := ValidateGeometry(types.DirectionBuy,
		decimal.NewFromFloat(2400), decimal.NewFromFloat(2396),
		decimal.NewFromFloat(2404), decimal.NewFromFloat(2403), decimal.NewFromFloat(2416), cfg)
	if valid {
		t.Error("TP2 < TP1 for BUY should fail")
	}
}

func TestValidateGeometry_ZeroEntry(t *testing.T) {
	cfg := strategy.StrategyConfig{}
	valid, reason := ValidateGeometry(types.DirectionBuy,
		decimal.Zero, decimal.NewFromFloat(2396),
		decimal.NewFromFloat(2404), decimal.NewFromFloat(2408), decimal.NewFromFloat(2416), cfg)
	if valid {
		t.Error("Zero entry should fail")
	}
	if reason != "ENTRY_IS_ZERO" {
		t.Errorf("Expected ENTRY_IS_ZERO, got %s", reason)
	}
}

func TestValidateGeometry_RRMismatch(t *testing.T) {
	// Config says R:R1 should be 1.0 (TP1=2, SL=2), but actual is 2.0
	cfg := strategy.StrategyConfig{ATRMultiplierSL: 2.0, ATRMultiplierTP1: 2.0}
	valid, reason := ValidateGeometry(types.DirectionBuy,
		decimal.NewFromFloat(2400), decimal.NewFromFloat(2390),
		decimal.NewFromFloat(2420), decimal.NewFromFloat(2440), decimal.NewFromFloat(2480), cfg)
	if valid {
		t.Error("R:R mismatch should fail")
	}
	if reason == "" {
		t.Error("Should have failure reason for R:R mismatch")
	}
}
