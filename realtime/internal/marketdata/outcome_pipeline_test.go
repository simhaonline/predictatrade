package marketdata

import (
	"fmt"
	"strings"
	"context"
	"encoding/json"
	"testing"
	"time"
)

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// P0.5 Candidate B: prediction + outcome writers.
// DB round-trips run against the real DB when PERSISTENCE_TEST_URL is set
// (P0 pattern); they verify schema, idempotency, and linkage — the exact
// data that calibration (Phase 1) will consume.
func TestSavePredictionRoundTrip(t *testing.T) {
	db := openFeatureSnapshotTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := &Persister{db: db}
	sigID := "22222222-3333-4444-5555-666666666666"

	// 27 args: sigID, symbol, tf, strat, stratVer, dir, entry, sl, tp1, tp2, tp3,
	// raw, calib, grade, class, tier, tierReason, composite, famJSON, regime,
	// session, gatesJSON, astro, yoga, macro, fsID, modelVer
	err := p.SavePrediction(ctx, sigID,
		"XAUUSD", "M5", "STANDARD_SCALPING", "1.0", "BUY",
		"2400.5", "2396.0", "2404.0", "2408.0", "2412.0",
		"55.2", "0.62", "A+", "EXECUTABLE", "A+", "composite=0.84 dominant=TREND",
		"0.84", mustJSON(t, map[string]any{"TREND": 0.12, "SMC": 0.10}),
		"TRENDING_BULLISH", "LONDON", mustJSON(t, map[string]any{}),
		"1.2", "0", "6.0", "33333333-4444-5555-6666-777777777777", "1.0")
	if err != nil {
		t.Fatalf("SavePrediction: %v", err)
	}
	// idempotency: second emit must not duplicate or error
	err2 := p.SavePrediction(ctx, sigID,
		"XAUUSD", "M5", "STANDARD_SCALPING", "1.0", "BUY",
		"2400.5", "2396.0", "2408.0", "2412.0", "2412.0",
		"55.2", "0.62", "A+", "EXECUTABLE", "A+", "idempotent-refresh",
		"0.84", mustJSON(t, map[string]any{"TREND": 0.12}),
		"TRENDING_BULLISH", "LONDON", mustJSON(t, map[string]any{}),
		"1.0", "0", "0", "33333333-4444-5555-6666-777777777777", "1.0")
	if err2 != nil {
		t.Fatalf("SavePrediction (idempotent): %v", err2)
	}
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM trading.predictions WHERE signal_id=$1::uuid`, sigID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("idempotency broken: %d rows for one signal", n)
	}
	// snapshot linkage stored
	var fsID *string
	_ = db.QueryRowContext(ctx,
		`SELECT feature_snapshot_id::text FROM trading.predictions WHERE signal_id=$1::uuid`, sigID).Scan(&fsID)
	if fsID == nil || *fsID != "33333333-4444-5555-6666-777777777777" {
		t.Fatalf("feature_snapshot_id = %v", fsID)
	}
}

func TestSaveOutcomeFromTradeResultRoundTrip(t *testing.T) {
	db := openFeatureSnapshotTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := &Persister{db: db}
	sigID := "44444444-5555-6666-7777-888888888888"
	// upsert twice → one row, idempotent on signal_id
	for i := 0; i < 2; i++ {
		oerr := p.SaveOutcomeFromTradeResult(ctx, sigID, "STANDARD_SCALPING",
			"TP1", "12.34", "1.48", "5.2", "18.9", 3600, "0.12", "0.31", true)
		if oerr != nil {
			t.Fatalf("SaveOutcomeFromTradeResult (pass %d): %v", i, oerr)
		}
	}
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM trading.prediction_outcomes WHERE signal_id=$1::uuid`, sigID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("outcome idempotency broken: %d rows", n)
	}
	var linkStatus, closeReason, schemaVersion string
	var rMult float64
	err := db.QueryRowContext(ctx,
		`SELECT link_status, close_reason, schema_version, r_multiple
		 FROM trading.prediction_outcomes WHERE signal_id=$1::uuid`, sigID).
		Scan(&linkStatus, &closeReason, &schemaVersion, &rMult)
	if err != nil {
		t.Fatal(err)
	}
	if linkStatus != "LINKED" || closeReason != "TP1" || rMult != 1.48 {
		t.Fatalf("outcome row mismatch: status=%s reason=%s r=%v", linkStatus, closeReason, rMult)
	}
	if schemaVersion == "" {
		t.Fatal("schema_version empty")
	}
	// MANUAL closes are first-class, not filtered
	merr := p.SaveOutcomeFromTradeResult(ctx, "66666666-7777-8888-9999-000000000000",
		"XAUUSD", "MANUAL", "-1.20", "-1.0", "0", "0", 120, "", "", true)
	if merr != nil {
		t.Fatalf("MANUAL outcome write: %v", merr)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT close_reason FROM trading.prediction_outcomes WHERE signal_id=$1::uuid`,
		"66666666-7777-8888-9999-000000000000").Scan(&closeReason); err != nil {
		t.Fatal(err)
	}
	if closeReason != "MANUAL" {
		t.Fatalf("MANUAL close must be recorded, got %q", closeReason)
	}
}

// ─── Phase 0.75 Task B: writer hardening ───

// TestOutcomeReasonMappingLowercase — legacy EA lowercase close reasons map to
// the canonical enum (sl→STOP etc.) instead of tripping the CHECK constraint.
func TestOutcomeReasonMappingLowercase(t *testing.T) {
	db := openFeatureSnapshotTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p := &Persister{db: db}

	cases := map[string]string{
		"sl":         "STOP",
		"tp1":        "TP1",
		"manual":     "MANUAL",
		"be":         "BE",
		"trail":      "TRAIL",
		"friday":     "FRIDAY_FLATTEN",
		"hard_stop":  "HARD_STOP",
		"timeout":    "TIMEOUT",
		"tp2":        "TP2",
		"tp3":        "TP3",
	}
	for raw, want := range cases {
		sig := "70000000-0000-0000-0000-0000000000" + fmt.Sprintf("%02d", len(want))
		if err := p.SaveOutcomeFromTradeResult(ctx, sig, "STANDARD_SCALPING",
			raw, "1.0", "0.5", "0", "0", 60, "", "", true); err != nil {
			t.Fatalf("reason %q write: %v", raw, err)
		}
		var got string
		if err := db.QueryRowContext(ctx,
			`SELECT close_reason FROM trading.prediction_outcomes WHERE signal_id=$1::uuid`,
			sig).Scan(&got); err != nil {
			t.Fatalf("reason %q read: %v", raw, err)
		}
		if got != want {
			t.Fatalf("reason %q mapped to %q, want %q", raw, got, want)
		}
	}
}

// TestSchemaDriftGuardDetectsMissingColumn — the startup guard must fail when
// trading.predictions / trading.prediction_outcomes lack a column the writer
// expects (Phase 0.5 lesson: CREATE TABLE IF NOT EXISTS silently no-op'd on a
// legacy table).
func TestSchemaDriftGuardDetectsMissingColumn(t *testing.T) {
	db := openFeatureSnapshotTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p := &Persister{db: db}

	if err := p.VerifyOutcomeSchema(ctx); err != nil {
		t.Fatalf("expected schema OK, got drift: %v", err)
	}

	// Simulate drift NON-DESTRUCTIVELY: create a scratch table missing one column
	// and point the guard's column list at it via a copy of the expected map.
	scratch := "trading.prediction_outcomes_drift_probe"
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS ` + scratch)
	defer db.ExecContext(ctx, `DROP TABLE IF EXISTS ` + scratch)
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE ` + scratch + ` AS
		SELECT * FROM trading.prediction_outcomes LIMIT 0`); err != nil {
		t.Fatalf("scratch table: %v", err)
	}
	// remove one column to simulate legacy drift
	if _, err := db.ExecContext(ctx, `ALTER TABLE ` + scratch + ` DROP COLUMN r_multiple`); err != nil {
		t.Skipf("cannot drop column (permissions): %v", err)
	}

	// Run the guard against the scratch table by temporarily extending the expected map.
	orig := outcomeSchemaColumns
	outcomeSchemaColumns = map[string][]string{
		scratch: orig["trading.prediction_outcomes"],
	}
	defer func() { outcomeSchemaColumns = orig }()

	err := p.VerifyOutcomeSchema(ctx)
	if err == nil {
		t.Fatal("schema drift NOT detected — guard must fail closed")
	}
	if !strings.Contains(err.Error(), "r_multiple") {
		t.Fatalf("drift error must name the missing column: %v", err)
	}
}
