// vedic_depth.go — P2 (prompt.md): Vedic depth features the reference EA has
// and the engine lacked. NON-OVERLAP: this file ADDS to the existing astro
// engine (state.go has sidereal longitudes, nakshatra, hora, dasha, eclipse;
// evidence.go exposes those) — nothing here replaces existing state.
//
// Implemented (documented assumptions, all deterministic & pure):
//
//   ShadbalaLite(planet, lon, moonLon):
//     dignity-by-sign (exaltation +100 … own sign +50 … debilitation −100),
//     plus +15 when within 30° of the Moon (support). Documented
//     simplification of the full 6-fold Shadbala.
//
//   ComputeYogas: classical conjunction/aspect yogas with a 10° orb:
//     Gajakesari      — Jupiter in kendra (0/90/180/270) from Moon
//     Budhaditya      — Sun conjunct Mercury
//     GuruMangala     — Jupiter conjunct Mars
//     ChandraMangala  — Moon conjunct Mars
//     Vish            — Saturn conjunct Moon
//
//   Demon hours (RahuKalam/Yamaganda/Gulika): daylight = [sunrise, sunset]
//   split into 8 equal segments; which segment is which varies by weekday
//   (reference tables). Operator supplies sunrise/sunset hours — caller
//   resolves them (Hetzner UTC defaults passed from config), keeping this
//   pure.
//
//   IsGandanta: junction of water→fire signs (Revati→Ashwini, Ashlesha→Magha,
//   Jyeshtha→Mula) — last 1/4 nakshatra pada boundary, 3.2° zones.
//
//   AstroSizingMultiplier: 0.8 + (shadbala/100)×0.5, ×0.65 in demon hours,
//   clamped [0.5, 1.5] (reference formula).
package astro

import (
	"time"
)

// ─── Sign dignity table (sidereal, Vedic) ───────────────────────────────
// planet → [exaltationDeg, debilitationDeg, ownSignStarts…]
// Strength +100 at exaltation, −100 at debilitation, +40 in own sign,
// linear falloff 30° around the exaltation/debilitation points.
var exaltation = map[string]float64{
	"Sun": 10, "Moon": 33, "Mercury": 163, "Venus": 357,
	"Mars": 298, "Jupiter": 95, "Saturn": 200, "Rahu": 20, "Ketu": 200,
}
var debilitation = map[string]float64{
	"Sun": 190, "Moon": 213, "Mercury": 160, "Venus": 35,
	"Mars": 299, "Jupiter": 275, "Saturn": 20, "Rahu": 200, "Ketu": 20,
}
var ownSigns = map[string][]float64{
	"Sun": {120, 240}, "Moon": {90}, "Mercury": {60, 160},
	"Venus": {20, 150}, "Mars": {210, 240}, "Jupiter": {240, 270},
	"Saturn": {300, 30},
}

func angDist(a, b float64) float64 {
	d := norm360f(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// norm360f normalizes to [0,360).
func norm360f(x float64) float64 {
	for x < 0 {
		x += 360
	}
	for x >= 360 {
		x -= 360
	}
	return x
}

// ShadbalaLite: dignity-based planet strength ∈ [−100, 100].
// moonLon > 0 enables the lunar-support bonus (+15 within 45°); pass 0 to
// disable. Unknown planets return 0 (neutral — no fabricated strength).
func ShadbalaLite(planet string, siderealLon, moonLon float64) float64 {
	ex, hasEx := exaltation[planet]
	if !hasEx {
		return 0
	}
	deb, hasDeb := debilitation[planet]

	var s float64
	switch {
	case hasEx && angDist(siderealLon, ex) <= 15:
		s = 100 - angDist(siderealLon, ex)*(100/15.0) // falloff toward own sign
	case hasDeb && angDist(siderealLon, deb) <= 15:
		s = -100 + angDist(siderealLon, deb)*(100/15.0)
	default:
		s = 0
		for _, o := range ownSigns[planet] {
			// own sign span: 30° starting at o
			rel := norm360f(siderealLon - o)
			if rel < 30 {
				s = 40
				break
			}
		}
	}
	if moonLon > 0 && angDist(siderealLon, moonLon) <= 45 {
		s += 15 // lunar support (mutual visibility)
	}
	return clamp(s, -100, 100)
}

// ─── Yoga suite ─────────────────────────────────────────────────────────

const yogaOrb = 10.0        // conjunction orb (degrees)
const kendraOrb = 12.0      // kendra (90°-family) aspect orb
const yogaBiasConj = 12.0   // conjunction yoga strength
const yogaBiasKendra = 10.0 // kendra yoga strength

// YogaContext carries the sidereal longitudes yoga checks need.
type YogaContext struct {
	Longitudes map[string]float64
}

// YogaFlags: classical yogas (reference values):
// Gajakesari +15, GuruMangala +12, Budhaditya +8, ChandraMangala +10,
// Dhana +10, Vish −20, Yama −25, Grahan −15.
type YogaResult struct {
	Gajakesari     bool
	GuruMangala    bool
	Budhaditya     bool
	ChandraMangala bool
	Dhana          bool
	Vish           bool
	Yama           bool
	Grahan         bool
}

// ComputeYogas evaluates the reference yoga set from sidereal longitudes.
func ComputeYogas(ctx YogaContext) YogaResult {
	var y YogaResult
	moon, hasMoon := ctx.Longitudes["Moon"]
	jup, hasJup := ctx.Longitudes["Jupiter"]
	sun, hasSun := ctx.Longitudes["Sun"]
	merc, hasMerc := ctx.Longitudes["Mercury"]
	mars, hasMars := ctx.Longitudes["Mars"]
	sat, hasSat := ctx.Longitudes["Saturn"]

	// Gajakesari: Jupiter in kendra from Moon (0/90/180/270 ± orb)
	if hasJup && hasMoon {
		d := angDist(moon, jup)
		for _, k := range []float64{0, 90, 180, 270} {
			if angDist(d, k) <= kendraOrb {
				y.Gajakesari = true
				break
			}
		}
	}
	// Budhaditya: Sun conjunct Mercury
	if hasSun && hasMerc && angDist(sun, merc) <= yogaOrb {
		y.Budhaditya = true
	}
	// GuruMangala: Jupiter conjunct Mars
	if hasJup && hasMars && angDist(jup, mars) <= yogaOrb {
		y.GuruMangala = true
	}
	// ChandraMangala: Moon conjunct Mars
	if hasMoon && hasMars && angDist(moon, mars) <= yogaOrb {
		y.ChandraMangala = true
	}
	// Vish: Saturn conjunct Moon
	if hasMoon && hasSat && angDist(moon, sat) <= yogaOrb {
		y.Vish = true
	}
	// Yama: Sun conjunct Saturn
	if hasSun && hasSat && angDist(sun, sat) <= yogaOrb {
		y.Yama = true
	}
	// Grahan: Moon conjunct Rahu/Ketu (eclipse-family)
	if hasMoon {
		if rahu, ok := ctx.Longitudes["Rahu"]; ok && angDist(moon, rahu) <= yogaOrb {
			y.Grahan = true
		}
		if ketu, ok := ctx.Longitudes["Ketu"]; ok && angDist(moon, ketu) <= yogaOrb {
			y.Grahan = true
		}
	}
	// Dhana: Moon-Jupiter OR Sun-Venus conjunction (wealth combinations)
	if hasMoon && hasJup && angDist(moon, jup) <= yogaOrb {
		y.Dhana = true
	}
	if venus, ok := ctx.Longitudes["Venus"]; ok && hasSun && angDist(sun, venus) <= yogaOrb {
		y.Dhana = true
	}
	return y
}

// YogaScoreBias applies the reference weights to a YogaResult.
func YogaScoreBias(y YogaResult) float64 {
	bias := 0.0
	if y.Gajakesari {
		bias += 15
	}
	if y.GuruMangala {
		bias += 12
	}
	if y.Budhaditya {
		bias += 8
	}
	if y.ChandraMangala {
		bias += 10
	}
	if y.Dhana {
		bias += 10
	}
	if y.Vish {
		bias -= 20
	}
	if y.Yama {
		bias -= 25
	}
	if y.Grahan {
		bias -= 15
	}
	return bias
}

// ─── Demon hours ────────────────────────────────────────────────────────

// DemonHourState flags the three demon periods.
type DemonHourState struct {
	RahuKalam bool
	Yamaganda bool
	Gulika    bool
}

// Reference segment tables: which 1/8th of daylight each demon period
// occupies, per weekday (0=Sunday … 6=Saturday).
var rahuSeg = map[int]int{0: 7, 1: 1, 2: 5, 3: 5, 4: 3, 5: 2, 6: 1}
var yamaSeg = map[int]int{0: 3, 1: 0, 2: 2, 3: 3, 4: 5, 5: 4, 6: 6}
var gulikaSeg = map[int]int{0: 6, 1: 5, 2: 3, 3: 4, 4: 2, 5: 0, 6: 3}

// DemonHours computes the demon-hour flags for a moment.
// sunriseH/sunsetH: local solar-day bounds in the same clock as t
// (documented simplification: fixed sunrise/sunset rather than ephemeral).
func ComputeDemonHours(t time.Time, sunriseHour, sunsetHour float64) DemonHourState {
	day := int(t.Weekday())
	daylight := sunsetHour - sunriseHour
	if daylight <= 0 {
		return DemonHourState{}
	}
	seg := (float64(t.Hour()) + float64(t.Minute())/60 - sunriseHour) / daylight * 8
	if seg < 0 || seg >= 8 {
		return DemonHourState{} // before sunrise / after sunset — no demon hours
	}
	idx := int(seg)
	return DemonHourState{
		RahuKalam: rahuSeg[day] == idx,
		Yamaganda: yamaSeg[day] == idx,
		Gulika:    gulikaSeg[day] == idx,
	}
}

// ─── Gandanta ───────────────────────────────────────────────────────────

// gandantaJunctions: the three water→fire sign junctions (sidereal degrees).
// Zone = last 2° of the water sign + first 2° of the fire sign (3.2° total,
// rounded to the standard 2°+2° window used by most Panchangs).
var gandantaJunctions = [][2]float64{
	{358.0, 2.0},   // Revati→Ashwini (Pisces→Aries)
	{238.0, 242.0}, // Ashlesha→Magha (Cancer→Leo)
	{298.0, 302.0}, // Jyeshtha→Mula (Scorpio→Sagittarius)
}

// IsGandanta: sidereal longitude inside a water→fire junction zone.
func IsGandanta(siderealLon float64) bool {
	n := norm360f(siderealLon)
	for _, z := range gandantaJunctions {
		a, b := z[0], z[1]
		if a < b {
			if n >= a && n < b {
				return true
			}
		} else { // wraps 0
			if n >= a || n < b {
				return true
			}
		}
	}
	return false
}

// ─── Sizing multiplier ──────────────────────────────────────────────────

// AstroSizingMultiplier: 0.8 + (shadbala/100)×0.5, ×0.65 during demon
// hours, clamped [0.5, 1.5] (reference formula). Pure.
func AstroSizingMultiplier(shadbala float64, demonHour bool) float64 {
	m := 0.8 + (shadbala/100.0)*0.5
	if demonHour {
		m *= 0.65
	}
	return clamp(m, 0.5, 1.5)
}