package astro

import (
	"testing"
	"time"
)

func TestComputeDeterministic(t *testing.T) {
	t1 := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	a := Compute(t1, true)
	b := Compute(t2, true)
	if a.CompositeScore != b.CompositeScore {
		t.Fatalf("non-deterministic: %v vs %v", a.CompositeScore, b.CompositeScore)
	}
	if !a.MarketClosed || a.IneligibleReason != "MARKET_CLOSED_LIVENESS_DATA" {
		t.Fatalf("market closed should be ineligible")
	}
}

func TestComputeRange(t *testing.T) {
	now := time.Now().UTC()
	st := Compute(now, false)
	if st.CompositeScore < scoreMin || st.CompositeScore > scoreMax {
		t.Fatalf("composite out of range: %v", st.CompositeScore)
	}
	if st.Vedic.NakshatraName == "" || st.Vedic.HoraLord == "" || st.Vedic.DashaL1 == "" {
		t.Fatalf("missing vedic fields")
	}
	if len(st.Western.TropicalLongitudes) == 0 {
		t.Fatalf("western longitudes empty")
	}
}

func TestSiderealShift(t *testing.T) {
	// verify ayanamsa grows monotonically
	a1 := ayanamsa(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	a2 := ayanamsa(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if a2 <= a1 {
		t.Fatalf("ayanamsa must increase with time")
	}
}

func TestNakshatra27(t *testing.T) {
	for i := 0; i < 27; i++ {
		name := nakshatraNames[i]
		if nakshatraBias[name] == 0 && name == "Pushya" {
			t.Fatalf("Pushya bias missing")
		}
	}
}
