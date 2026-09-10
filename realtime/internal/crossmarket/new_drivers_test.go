package crossmarket

import (
	"testing"
	"time"
)

// TestNormalizeUSDCHFDirection — rising USD/CHF (stronger USD) must be
// bearish for gold; falling bullish; flat neutral.
func TestNormalizeUSDCHFDirection(t *testing.T) {
	now := time.Now().UTC()

	// Rising USDCHF: 0.8100 → 0.8120 (+0.0020)
	up := NormalizeUSDCHF(0.8120, 0.8100, now)
	if up.Direction != DirBearish {
		t.Errorf("rising USDCHF should be bearish for gold, got %s (impact %.1f)", up.Direction, up.ImpactScore)
	}
	if up.Quality != QualityConnected {
		t.Errorf("expected CONNECTED quality, got %s", up.Quality)
	}
	if up.Name != DriverUSDCHF {
		t.Errorf("expected driver name usdchf, got %s", up.Name)
	}

	// Falling USD/CHF → bullish for gold
	down := NormalizeUSDCHF(0.8080, 0.8100, now)
	if down.Direction != DirBullish {
		t.Errorf("falling USDCHF should be bullish for gold, got %s (impact %.1f)", down.Direction, down.ImpactScore)
	}

	// Flat → neutral
	flat := NormalizeUSDCHF(0.8100, 0.8100, now)
	if flat.Direction != DirNeutral {
		t.Errorf("flat USDCHF should be neutral, got %s", flat.Direction)
	}
}

// TestNormalizeUSDCHFCapped — a wild move must clamp to the ±50 bound.
func TestNormalizeUSDCHFClamped(t *testing.T) {
	now := time.Now().UTC()
	extreme := NormalizeUSDCHF(0.9000, 0.8000, now)
	if extreme.ImpactScore > 50 || extreme.ImpactScore < -50 {
		t.Errorf("impact %.1f outside ±50 clamp", extreme.ImpactScore)
	}
}

// TestNormalizeFedContextEventWindows — event-risk scaling by proximity.
func TestNormalizeFedContextEventWindows(t *testing.T) {
	now := time.Now().UTC()

	// No events → neutral
	none := NormalizeFedContext(nil, nil, now)
	if none.Direction != DirNeutral {
		t.Errorf("no events should be neutral, got %s", none.Direction)
	}

	// FOMC within 4h → strongly risk-off
	imminent := now.Add(2 * time.Hour)
	soon := NormalizeFedContext(&imminent, nil, now)
	if soon.ImpactScore != -45 {
		t.Errorf("FOMC within 4h should be -45, got %.1f", soon.ImpactScore)
	}
	if soon.Quality != QualityConnected {
		t.Errorf("expected CONNECTED, got %s", soon.Quality)
	}

	// FOMC within 24h → -20
	dayTs := now.Add(20 * time.Hour)
	day := NormalizeFedContext(&dayTs, nil, now)
	if day.ImpactScore != -20 {
		t.Errorf("FOMC within 24h should be -20, got %.1f", day.ImpactScore)
	}

	// FOMC within 48h → -8
	twoDay := now.Add(40 * time.Hour)
	mild := NormalizeFedContext(&twoDay, nil, now)
	if mild.ImpactScore != -8 {
		t.Errorf("FOMC within 48h should be -8, got %.1f", mild.ImpactScore)
	}

	// Past FOMC within 24h → -10 digestion
	yesterday := now.Add(-12 * time.Hour)
	digest := NormalizeFedContext(nil, &yesterday, now)
	if digest.ImpactScore != -10 {
		t.Errorf("recent past FOMC should be -10, got %.1f", digest.ImpactScore)
	}

	// Old FOMC (>24h past) → neutral
	old := now.Add(-72 * time.Hour)
	staleEv := NormalizeFedContext(nil, &old, now)
	if staleEv.Direction != DirNeutral {
		t.Errorf("old FOMC should be neutral, got %s", staleEv.Direction)
	}
}