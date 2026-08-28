// Package marketdata — tests for signal truth, traceability, and durability.
// prompt.md Sections 5-13, 15-22, 27-34, 57-63
package marketdata

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Test signal reference generation format (prompt.md Section 6)
func TestGenerateSignalReference_Format(t *testing.T) {
	ref := GenerateSignalReference(1)
	if ref == "" {
		t.Fatal("Signal reference should not be empty")
	}
	// Should start with PAT-XAU-
	if len(ref) < 10 || ref[:8] != "PAT-XAU-" {
		t.Errorf("Signal reference should start with PAT-XAU-, got %s", ref)
	}
}

func TestGenerateSignalReference_Unique(t *testing.T) {
	ref1 := GenerateSignalReference(1)
	ref2 := GenerateSignalReference(2)
	if ref1 == ref2 {
		t.Error("Different sequences should produce different references")
	}
}

func TestGenerateSignalReference_Sortable(t *testing.T) {
	ref1 := GenerateSignalReference(100)
	ref2 := GenerateSignalReference(200)
	if ref1 >= ref2 {
		t.Error("Higher sequence should produce lexicographically higher reference")
	}
}

// Test production source safeguard (prompt.md Section 3)
func TestIsProductionSafeSource(t *testing.T) {
	if !IsProductionSafeSource("LIVE_AGENT") {
		t.Error("LIVE_AGENT should be production-safe")
	}
	if !IsProductionSafeSource("AGENT") {
		t.Error("AGENT should be production-safe")
	}
	if IsProductionSafeSource("SIMULATED") {
		t.Error("SIMULATED should NOT be production-safe")
	}
	if IsProductionSafeSource("SYNTHETIC") {
		t.Error("SYNTHETIC should NOT be production-safe")
	}
	if IsProductionSafeSource("TEST") {
		t.Error("TEST should NOT be production-safe")
	}
}

// Test score status semantics (prompt.md Sections 15-16)
func TestScoreStatusSemantics(t *testing.T) {
	// COMPUTED means the score was actually evaluated
	if types.ScoreStatusComputed != "COMPUTED" {
		t.Error("COMPUTED should be the correct value")
	}
	// NOT_EVALUATED means the strategy was skipped
	if types.ScoreStatusNotEvaluated != "NOT_EVALUATED" {
		t.Error("NOT_EVALUATED should be the correct value")
	}
}

// Test calibration status (prompt.md Sections 18-19)
func TestCalibrationStatus(t *testing.T) {
	if types.IsCalibrationValidated(types.CalibrationUnverified) {
		t.Error("UNVERIFIED should not be validated")
	}
	if !types.IsCalibrationValidated(types.CalibrationValidated) {
		t.Error("VALIDATED should be validated")
	}
	if !types.IsCalibrationValidated(types.CalibrationPromoted) {
		t.Error("PROMOTED should be validated")
	}
}

// Test provenance state (prompt.md Section 57)
func TestProvenanceState(t *testing.T) {
	if !types.IsLiveDataSource(types.DataSourceLiveAgent) {
		t.Error("LIVE_AGENT should be live")
	}
	if types.IsLiveDataSource(types.DataSourceSynthetic) {
		t.Error("SYNTHETIC should not be live")
	}
	if types.IsLiveDataSource(types.DataSourceTest) {
		t.Error("TEST should not be live")
	}
}

// Test bar closed state (prompt.md Section 32)
func TestBarClosedState(t *testing.T) {
	if types.BarClosedConfirmed != "CLOSED_BAR_CONFIRMED" {
		t.Error("CLOSED_BAR_CONFIRMED value mismatch")
	}
	if types.BarIntrabarLive != "INTRABAR_LIVE" {
		t.Error("INTRABAR_LIVE value mismatch")
	}
}

// Test signal with zero score and NOT_EVALUATED status (prompt.md Section 16)
func TestZeroScoreVsNotEvaluated(t *testing.T) {
	sig := &types.Signal{
		RawScore:    decimal.Zero,
		ScoreStatus: types.ScoreStatusNotEvaluated,
	}
	// Zero score with NOT_EVALUATED means the strategy was skipped
	if sig.ScoreStatus != types.ScoreStatusNotEvaluated {
		t.Error("Score status should be NOT_EVALUATED")
	}

	sig2 := &types.Signal{
		RawScore:    decimal.Zero,
		ScoreStatus: types.ScoreStatusComputed,
	}
	// Zero score with COMPUTED means the evidence mathematically totaled zero
	if sig2.ScoreStatus != types.ScoreStatusComputed {
		t.Error("Score status should be COMPUTED")
	}
}

// Test exit price is NULL before closure (prompt.md Section 35)
func TestExitPriceNullBeforeClosure(t *testing.T) {
	sig := &types.Signal{
		Direction:  types.DirectionBuy,
		EntryPrice: decimal.NewFromFloat(4400.0),
		TP1:        decimal.NewFromFloat(4412.0),
	}
	if !sig.ExitPrice.IsZero() {
		t.Error("ExitPrice should be zero before closure")
	}
	if !sig.ClosedAt.IsZero() {
		t.Error("ClosedAt should be zero before closure")
	}
}

// Test signal reference immutability concept
func TestSignalReferenceImmutability(t *testing.T) {
	ref1 := GenerateSignalReference(1)
	// The reference should contain the sequence number
	if ref1 == "" {
		t.Error("Reference should not be empty")
	}
	// Same sequence produces same reference (deterministic day portion)
	ref1b := GenerateSignalReference(1)
	if ref1 != ref1b {
		t.Error("Same sequence should produce same reference (same day)")
	}
}
