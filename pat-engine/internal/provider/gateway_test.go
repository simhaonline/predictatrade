package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"pat-engine/internal/backtest"
	"pat-engine/internal/license"
)

func TestLicenseRestrictsStrategySelection(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "PAT_signals.txt")

	// License that only entitles TREND_SWING.
	_, tok, err := license.DevLicense(license.DefaultDevSecret, []string{"TREND_SWING"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Build a gateway via New then LoadLicense (New installs a dev-allow-all license).
	gw := New(nil, out)
	if err := gw.LoadLicense(tok, license.DefaultDevSecret); err != nil {
		t.Fatal(err)
	}

	for _, b := range backtest.Generate(2000, 7) {
		gw.IngestBar(b)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		// No signal emitted at all is acceptable; the key check is suppression.
		t.Logf("no signal file produced (ok): %v", err)
		return
	}
	var dto SignalDTO
	if err := json.Unmarshal([]byte(data[7:]), &dto); err != nil { // skip "SIGNAL|"
		t.Fatalf("bad signal line: %v", err)
	}
	if dto.StrategyID != "TREND_SWING" {
		t.Fatalf("license must suppress non-entitled strategies; got %s", dto.StrategyID)
	}
}
