// ASTRO evidence contribution — integrates Astro-Financial Intelligence into
// the signal engine (check.md 2026-08-30: "add to our scoring system as well").
//
// The ASTRO factor is computed per candle close and injected into the evidence
// pillars list with weight 0.10 (10% of composite), the 5th intelligence engine.
// It contributes to ALL plans but only ELITE gets the full breakdown
// (entitlement gate enforced downstream at /astro endpoints and delivery).
package astro

import (
	"fmt"
	"time"
)

// GetEvidence returns the ASTRO evidence contribution for the current instant.
// Returns label, score (−100..+100), and a deterministic human reason.
func GetEvidence(t time.Time, marketClosed bool) (label string, score float64, detail string, ok bool) {
	st := Compute(t, marketClosed)
	label = "ASTRO"
	score = st.CompositeScore
	detail = st.Vedic.NakshatraName + " · " + st.Vedic.HoraLord + " hora · " +
		st.Vedic.DashaL1 + " dasha · Western " + fmt2f(st.Western.TotalScore)
	if marketClosed {
		detail = "Market closed — ASTRO liveness only, no signal contribution."
		ok = false
	}
	_ = st.Confidence
	return label, score, detail, !marketClosed
}

func fmt2f(v float64) string {
	// tiny local formatter to avoid fmt import cycle in tests
	s := fmt.Sprintf("%.1f", v)
	return s
}
