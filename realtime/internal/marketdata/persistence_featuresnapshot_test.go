package marketdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// P0-2 (prompt.md): per-signal feature snapshots — the verifiable indicator
// reading flow. These tests hit a REAL Postgres when PERSISTENCE_TEST_URL is
// set (pattern for this repo's DB-bound tests); they skip otherwise so unit
// runs stay hermetic.
//
// 2026-09-17 incident: a round-trip run pointed PERSISTENCE_TEST_URL at the
// PRODUCTION predictatrade DB and left fixture rows in
// trading.predictions / trading.prediction_outcomes / feature snapshots,
// polluting the Phase-1 sample-sufficiency gate. The guard below refuses any
// target whose database name does not contain "test" — round-trips must run
// against a scratch DB (e.g. predictatrade_test), never prod.

func openFeatureSnapshotTestDB(t *testing.T) *sql.DB {
	t.Helper()
	target := os.Getenv("PERSISTENCE_TEST_URL")
	if target == "" {
		t.Skip("PERSISTENCE_TEST_URL not set — DB round-trip test skipped")
	}
	// Fail-safe: never write fixture rows into a production database.
	u, perr := url.Parse(target)
	if perr != nil {
		t.Skipf("PERSISTENCE_TEST_URL unparseable (%v) — DB round-trip test skipped", perr)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" || !strings.Contains(strings.ToLower(dbName), "test") {
		t.Skipf("PERSISTENCE_TEST_URL must point at a scratch DB whose name contains "+
			"\"test\" (got %q) — refusing to write fixture rows into %q", dbName, dbName)
	}
	db, err := sql.Open("pgx", target)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// SaveFeatureSnapshot must persist the indicator read set, and the snapshot
// row must carry the signal's own id (1:1 contract used by the API surface).
func TestFeatureSnapshotRoundTrip(t *testing.T) {
	db := openFeatureSnapshotTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inds := map[string]any{
		"rsi": 39.28, "adx": 43.82, "ema9": 4321.25, "vwap_z": 0.42, "atr": 518.49,
	}
	b, _ := json.Marshal(inds)
	sigID := "11111111-2222-3333-4444-555555555555"

	p := &Persister{db: db}
	if err := p.SaveFeatureSnapshot(ctx, sigID, "XAUUSD", "M5", "STANDARD_SCALPING", "1.0", b); err != nil {
		t.Fatalf("SaveFeatureSnapshot: %v", err)
	}
	got, err := p.GetFeatureSnapshot(ctx, sigID)
	if err != nil {
		t.Fatalf("GetFeatureSnapshot: %v", err)
	}
	if got["rsi"] != 39.28 || got["adx"] != 43.82 {
		t.Fatalf("round-trip mismatch: %v", got)
	}

	// schema-version metadata persisted
	var sv string
	if err := db.QueryRowContext(ctx,
		`SELECT schema_version FROM trading.signal_feature_snapshots WHERE id=$1::uuid`, sigID).Scan(&sv); err != nil {
		t.Fatalf("schema_version read: %v", err)
	}
	if sv == "" {
		t.Fatal("schema_version empty")
	}
}
