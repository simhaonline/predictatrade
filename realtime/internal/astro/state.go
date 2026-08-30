package astro

import (
	"math"
	"time"
)

// State is the full Astro-Financial snapshot at an instant (Vedic + Western).
type State struct {
	Timestamp        time.Time                     `json:"timestamp"`
	Vedic            VedicState                    `json:"vedic"`
	Western          WesternState                  `json:"western"`
	CompositeScore   float64                       `json:"composite_score"` // −100..+100
	Confidence       float64                       `json:"confidence"`      // 0..1
	EligibleForTrade bool                          `json:"eligible_for_trade"`
	IneligibleReason string                        `json:"ineligible_reason,omitempty"`
	Factors          map[string]FactorContribution `json:"factors"`
	Apocalypse       *ApocalypseTrigger            `json:"apocalypse,omitempty"`
	MarketClosed     bool                          `json:"market_closed"`
	Note             string                        `json:"note,omitempty"`
}

type FactorContribution struct {
	Label  string  `json:"label"`
	Score  float64 `json:"score"`
	Detail string  `json:"detail,omitempty"`
	Weight float64 `json:"weight"`
}

type VedicState struct {
	SiderealLongitudes map[string]float64 `json:"sidereal_longitudes"`
	NakshatraIdx       int                `json:"nakshatra"`
	NakshatraName      string             `json:"nakshatra_name"`
	Pada               int                `json:"pada"`
	NakshatraBias      float64            `json:"nakshatra_bias"`
	HoraLord           string             `json:"hora_lord"`
	HoraBias           float64            `json:"hora_bias"`
	DashaL1            string             `json:"dasha_l1"`
	DashaL1Bias        float64            `json:"dasha_l1_bias"`
	DashaL2            string             `json:"dasha_l2"`
	DashaL2Bias        float64            `json:"dasha_l2_bias"`
	DashaProgressPct   float64            `json:"dasha_progress_pct"`
	Contamination      []string           `json:"contamination,omitempty"`
	Eclipse            bool               `json:"eclipse"`
	EclipseType        string             `json:"eclipse_type,omitempty"`
}

type WesternState struct {
	TropicalLongitudes map[string]float64 `json:"tropical_longitudes"`
	IsRetrograde       map[string]bool    `json:"is_retrograde"`
	Aspects            []WesternAspect    `json:"aspects"`
	LunarPhase         string             `json:"lunar_phase"` // new | waxing | full | waning
	TotalScore         float64            `json:"total_score"`
}

type WesternAspect struct {
	PlanetA string  `json:"planet_a"`
	PlanetB string  `json:"planet_b"`
	Type    string  `json:"type"`
	Orb     float64 `json:"orb"`
	Bias    float64 `json:"bias"`
}

type Apocalypse struct {
	Code     string `json:"code"`
	Severity int    `json:"severity"`
	Action   string `json:"action"`
	Duration string `json:"duration,omitempty"`
}

const (
	scoreMin = -100.0
	scoreMax = 100.0
	// XAUUSD Gold natal anchor (Nixon Shock 1971-08-15)
)

// Compute returns the full Astro-Financial state for a timestamp.
func Compute(t time.Time, marketClosed bool) *State {
	st := &State{Timestamp: t, Factors: map[string]FactorContribution{}, MarketClosed: marketClosed}
	if marketClosed {
		st.Note = "Market closed — ASTRO state computed for reference; no signals until re-open."
	}

	// ── Vedic ──
	sidMoon := siderealOf(mustLon("Moon", t), t)

	nakIdx, pada := nakshatraOf(sidMoon)
	st.Vedic.NakshatraName = nakshatraNames[nakIdx-1]
	st.Vedic.Pada = pada
	st.Vedic.NakshatraBias = nakshatraBias[st.Vedic.NakshatraName]

	// Hora: hour within day (broker-agnostic UTC), day lord from local weekday
	dayLord := weekdayLord[t.Weekday()]
	dayStart := time.Date(t.Year(), t.Month(), t.Day(), 6, 0, 0, 0, time.UTC) // 06:00 sunrise
	hoursFromSunrise := int(t.Sub(dayStart).Hours()) % 24
	startIdx := indexOf(horaOrder, dayLord)
	st.Vedic.HoraLord = horaOrder[(startIdx+hoursFromSunrise)%7]
	st.Vedic.HoraBias = horaBias[st.Vedic.HoraLord]

	// Vimshottari dasha from a reference epoch 1971-08-15 + known MahaDasha start
	refEpoch := time.Date(1971, 8, 15, 0, 0, 0, 0, time.UTC)
	elapsed := t.Sub(refEpoch).Hours() / 24 / 365.24219
	total := 0.0
	for _, lord := range vimshottariOrder {
		total += vimshottariYears[lord]
	}
	p := math.Mod(elapsed, total)
	cursor := 0.0
	prog1 := 0.0
	for i, lord := range vimshottariOrder {
		yrs := vimshottariYears[lord]
		if p < cursor+yrs {
			prog1 = (p - cursor) / yrs
			var subLord string
			acc := 0.0
			for j := 0; j < len(vimshottariOrder); j++ {
				sub := vimshottariOrder[(i+j)%len(vimshottariOrder)]
				yrsSub := yrs * vimshottariYears[sub] / 120.0
				if p < cursor+acc+yrsSub {
					subLord = sub
					break
				}
				acc += yrsSub
			}
			st.Vedic.DashaL1 = lord
			st.Vedic.DashaL1Bias = dashaBias[lord]
			st.Vedic.DashaProgressPct = prog1 * 100
			st.Vedic.DashaL2 = subLord
			st.Vedic.DashaL2Bias = dashaBias[subLord]
			break
		}
		cursor += yrs
	}

	// Contamination + eclipse
	eclipse, etype := isInEclipseWindow(t)
	st.Vedic.Eclipse = eclipse
	st.Vedic.EclipseType = etype
	if eclipse {
		st.Vedic.Contamination = append(st.Vedic.Contamination, etype)
	}

	// Western: tropical longitudes + aspects to Gold natal
	st.Western.TropicalLongitudes = map[string]float64{}
	st.Western.IsRetrograde = map[string]bool{}
	for name := range planets {
		lon, rx := approxLongitude(name, t)
		st.Western.TropicalLongitudes[name] = norm360(lon)
		st.Western.IsRetrograde[name] = rx
	}
	lonMoon, _ := approxLongitude("Moon", t)
	st.Western.TropicalLongitudes["Moon"] = lonMoon

	totalWestern := 0.0
	// transit-to-natal aspects
	for planet, lon := range st.Western.TropicalLongitudes {
		for natalName, natalLon := range goldNatalTropical {
			diff := angularDiff(lon, natalLon)
			for _, asp := range aspects {
				if math.Abs(diff-aspectDeg(asp)) <= asp.orbs {
					bias := asp.bias(planet, natalName)
					st.Western.Aspects = append(st.Western.Aspects, WesternAspect{
						PlanetA: planet, PlanetB: natalName, Type: asp.name,
						Orb: math.Abs(diff - aspectDeg(asp)), Bias: bias,
					})
					totalWestern += bias
				}
			}
		}
	}
	// Mercury Rx caution
	if st.Western.IsRetrograde["Mercury"] {
		totalWestern -= 10
		st.Factors["WEST_MERCURY_RX"] = FactorContribution{Label: "Mercury Retrograde", Score: -10, Weight: 0.15}
	}
	st.Western.TotalScore = math.Max(-100, math.Min(100, totalWestern))
	st.Factors["WESTERN_SCORE"] = FactorContribution{Label: "Western Tropical Aspects", Score: st.Western.TotalScore, Weight: 0.30}

	// Nakshatra factor
	nak := nakshatraBias[st.Vedic.NakshatraName]
	st.Factors["VEDIC_NAKSHATRA"] = FactorContribution{Label: st.Vedic.NakshatraName, Score: nak, Weight: 0.25}
	// Hora factor
	hr := horaBias[st.Vedic.HoraLord]
	st.Factors["VEDIC_HORA"] = FactorContribution{Label: st.Vedic.HoraLord + " hora", Score: hr, Weight: 0.12}
	// Dasha L1 factor
	d1 := dashaBias[st.Vedic.DashaL1]
	d2 := dashaBias[st.Vedic.DashaL2]
	dashCombined := (d1*(1-st.Vedic.DashaProgressPct/100) + d2*(st.Vedic.DashaProgressPct/100)) / 2.2
	st.Factors["VEDIC_DASHA"] = FactorContribution{Label: st.Vedic.DashaL1 + "/" + st.Vedic.DashaL2, Score: dashCombined, Weight: 0.20}

	// Composite
	composite := nak*0.25 + hr*0.12 + dashCombined*0.22 + st.Western.TotalScore*0.28
	for range st.Vedic.Contamination {
		composite -= 20
	}
	// normalise −100..+100
	st.CompositeScore = clamp(composite, scoreMin, scoreMax)

	// Eligibility rules — deterministic fail-closed
	st.EligibleForTrade = true
	if marketClosed {
		st.EligibleForTrade = false
		st.IneligibleReason = "MARKET_CLOSED_LIVENESS_DATA"
		return st
	}
	for _, trig := range apocalypseTriggers(t, st.Vedic) {
		st.EligibleForTrade = false
		st.IneligibleReason = trig
		st.Apocalypse = &ApocalypseTrigger{Code: trig, Severity: 100, Action: "MANDATORY_NO_ASTRO_SIGNAL"}
		return st
	}
	if eclipse {
		st.EligibleForTrade = false
		st.IneligibleReason = "ECLIPSE_WINDOW"
		st.Apocalypse = &ApocalypseTrigger{Code: "ECLIPSE_WINDOW", Severity: 75, Action: "NO_ASTRO_SIGNAL"}
	}
	return st
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

// apocalypseTriggers (subset of check.md §6.5 — deterministic ones)
func apocalypseTriggers(t time.Time, vs VedicState) []string {
	var out []string
	for _, c := range vs.Contamination {
		if c == "solar_eclipse_window" {
			out = append(out, "APO_ECLIPSE_WINDOW")
		}
	}
	// Great conjunction approximation: Saturn-Jupiter same sign, ~20yr cycle
	if isSatJupiterConj(t) {
		out = append(out, "APO_GREAT_CONJUNCTION")
	}
	return out
}

func isSatJupiterConj(t time.Time) bool {
	d := t.Sub(j2000).Hours() / 24
	jup := norm360(planets["Jupiter"].meanLonJ2000 + planets["Jupiter"].perDay*d)
	sat := norm360(planets["Saturn"].meanLonJ2000 + planets["Saturn"].perDay*d)
	return math.Abs(angularDiff(jup, sat)) < 2
}

// ---- helpers ---------------------------------------------------------------
func aspectDeg(a aspectDef) float64 { return a.deg }

func natalPlanetFor(lon float64) string {
	// find closest natal planet name by longitude
	closest := "—"
	best := 1e9
	for name, nl := range goldNatalTropical {
		if math.Abs(angularDiff(lon, nl)) < best {
			best = math.Abs(angularDiff(lon, nl))
			closest = name
		}
	}
	return closest
}

func angularDiff(a, b float64) float64 {
	d := math.Abs(norm360(a - b))
	if d > 180 {
		d = 360 - d
	}
	return d
}

func mustLon(planet string, t time.Time) float64 {
	lon, _ := approxLongitude(planet, t)
	return lon
}

func indexOf[T comparable](arr []T, v T) int {
	for i, x := range arr {
		if x == v {
			return i
		}
	}
	return -1
}
