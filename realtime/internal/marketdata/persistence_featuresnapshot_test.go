package marketdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// P0-2 (prompt.md): per-signal feature snapshots — the verifiable indicator
// reading flow. These tests hit the REAL control-plane DB when
// PERSISTENCE_TEST_URL is set (pattern for this repo's DB-bound tests);
// they skip otherwise so unit runs stay hermetic.

func openFeatureSnapshotTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("PERSISTENCE_TEST_URL")
	if url == "" {
		t.Skip("PERSISTENCE_TEST_URL not set — DB round-trip test skipped")
	}
	db, err := sql.Open("pgx", url)
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