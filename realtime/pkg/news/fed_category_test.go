package news

import "testing"

// TestCategorizeFedDecision verifies FMP Fed event names map to RATE_DECISION
// (the category the crossmarket fed_context driver filters on).
func TestCategorizeFedDecision(t *testing.T) {
	p := NewFMPProvider("k")
	cases := map[string]string{
		"Fed Interest Rate Decision":  "RATE_DECISION",
		"FOMC Economic Projections":   "RATE_DECISION",
		"Fed Press Conference":        "SPEECH",
		"FOMC Minutes":                "RATE_DECISION",
		"Federal Reserve Rate Decision": "RATE_DECISION",
	}
	for name, want := range cases {
		got := p.categorizeEvent(name)
		if got != want {
			t.Errorf("categorizeEvent(%q) = %q, want %q", name, got, want)
		}
	}
}