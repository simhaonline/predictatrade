package crossmarket

import (
	"math"
	"time"
)

// NormalizeDXY converts a DXY value + trend into a bounded impact score.
// Conceptually: weakening USD → supportive for gold (positive).
// But this is NOT deterministic — it's a probabilistic influence.
func NormalizeDXY(dxyValue, dxyPrevValue float64, timestamp time.Time) DriverSnapshot {
	change := dxyValue - dxyPrevValue
	// Normalize: falling DXY = positive (bullish for gold)
	// Use a bounded transform: 1% DXY drop ≈ +40 impact
	impact := -change * 4000 // scale factor
	impact = math.Max(-100, math.Min(100, impact))

	dir := DirNeutral
	if impact > 15 {
		dir = DirBullish
	} else if impact < -15 {
		dir = DirBearish
	}

	return DriverSnapshot{
		Name:            DriverDXY,
		RawValue:        dxyValue,
		NormalizedValue: impact,
		ImpactScore:     impact,
		Direction:       dir,
		Confidence:      0.7,
		Quality:         QualityConnected,
		Source:          "twelvedata",
		Timeframe:       "intraday",
		Reason:          "DXY " + directionWord(change) + " → " + directionWord(impact) + " for gold",
		Timestamp:       timestamp,
	}
}

// NormalizeEURUSD converts EURUSD movement into a confirmation signal.
// EURUSD is a CONFIRMATION driver, not a primary driver (anti-double-counting).
func NormalizeEURUSD(eurusdValue, eurusdPrevValue float64, timestamp time.Time) DriverSnapshot {
	change := eurusdValue - eurusdPrevValue
	// Rising EURUSD = weakening USD = supportive for gold
	impact := change * 30000 // scale factor
	impact = math.Max(-60, math.Min(60, impact)) // capped lower than DXY (confirmation role)

	dir := DirNeutral
	if impact > 10 {
		dir = DirBullish
	} else if impact < -10 {
		dir = DirBearish
	}

	return DriverSnapshot{
		Name:            DriverEURUSD,
		RawValue:        eurusdValue,
		NormalizedValue: impact,
		ImpactScore:     impact,
		Direction:       dir,
		Confidence:      0.6,
		Quality:         QualityConnected,
		Source:          "twelvedata",
		Timeframe:       "intraday",
		Reason:          "EURUSD " + directionWord(change) + " — USD confirmation",
		Timestamp:       timestamp,
	}
}

// NormalizeCOT converts COT net positioning into a medium-term context score.
func NormalizeCOT(netPosition, percentile float64, timestamp time.Time) DriverSnapshot {
	// percentile: 0 = extreme short, 1 = extreme long
	// Convert to impact: extreme long = crowding risk (bearish for new longs)
	// extreme short = potential squeeze (bullish)
	impact := (0.5 - percentile) * 100 // inverted: high percentile = bearish crowding
	impact = math.Max(-50, math.Min(50, impact))

	dir := DirNeutral
	if impact > 15 {
		dir = DirBullish
	} else if impact < -15 {
		dir = DirBearish
	}

	quality := QualityConnected
	if timestamp.IsZero() {
		quality = QualityMissing
	}

	return DriverSnapshot{
		Name:            DriverCOT,
		RawValue:        netPosition,
		NormalizedValue: impact,
		ImpactScore:     impact,
		Direction:       dir,
		Confidence:      0.5, // lower confidence — weekly data
		Quality:         quality,
		Source:          "fmp_cot",
		Timeframe:       "weekly",
		Reason:          "COT percentile " + formatPercentile(percentile),
		Timestamp:       timestamp,
	}
}

// NormalizeVIX converts VIX level into a risk sentiment score.
func NormalizeVIX(vixValue, vixPrevValue float64, timestamp time.Time) DriverSnapshot {
	change := vixValue - vixPrevValue

	// VIX interpretation is regime-dependent:
	// Moderate VIX rise with gold rise = risk-off bid for gold (bullish)
	// Extreme VIX spike = liquidity stress (can be bearish for gold initially)
	var impact float64
	if vixValue > 40 {
		// Extreme fear — liquidity stress, gold may sell off
		impact = -20
	} else if vixValue > 25 && change > 0 {
		// Rising VIX — risk-off, potentially bullish for gold
		impact = 15
	} else if vixValue < 15 {
		// Low VIX — risk-on, neutral for gold
		impact = -5
	} else {
		impact = 0
	}

	impact = math.Max(-40, math.Min(40, impact))

	dir := DirNeutral
	if impact > 10 {
		dir = DirBullish
	} else if impact < -10 {
		dir = DirBearish
	}

	return DriverSnapshot{
		Name:            DriverVIX,
		RawValue:        vixValue,
		NormalizedValue: impact,
		ImpactScore:     impact,
		Direction:       dir,
		Confidence:      0.5,
		Quality:         QualityConnected,
		Source:          "market_feed",
		Timeframe:       "intraday",
		Reason:          "VIX at " + formatFloat(vixValue) + " — " + string(dir),
		Timestamp:       timestamp,
	}
}

// NormalizeBTC converts BTC movement into a low-weight crypto sentiment score.
func NormalizeBTC(btcValue, btcPrevValue float64, timestamp time.Time) DriverSnapshot {
	change := btcValue - btcPrevValue
	pctChange := 0.0
	if btcPrevValue > 0 {
		pctChange = (change / btcPrevValue) * 100
	}

	// BTC is LOW priority — small impact, capped
	impact := pctChange * 2 // scale down significantly
	impact = math.Max(-20, math.Min(20, impact))

	dir := DirNeutral
	if impact > 5 {
		dir = DirBullish
	} else if impact < -5 {
		dir = DirBearish
	}

	return DriverSnapshot{
		Name:            DriverBTC,
		RawValue:        btcValue,
		NormalizedValue: impact,
		ImpactScore:     impact,
		Direction:       dir,
		Confidence:      0.3, // LOW confidence
		Quality:         QualityConnected,
		Source:          "market_feed",
		Timeframe:       "intraday",
		Reason:          "BTC " + formatFloat(pctChange) + "% — crypto sentiment",
		Timestamp:       timestamp,
	}
}

func directionWord(v float64) string {
	if v > 0 {
		return "rising"
	}
	if v < 0 {
		return "falling"
	}
	return "flat"
}

func formatPercentile(p float64) string {
	return formatFloat(p*100) + "%"
}

// NormalizeOil converts WTI/Brent price movement into an inflation context score.
func NormalizeOil(oilValue, oilPrevValue float64, timestamp time.Time) DriverSnapshot {
	change := oilValue - oilPrevValue
	pctChange := 0.0
	if oilPrevValue > 0 {
		pctChange = (change / oilPrevValue) * 100
	}

	// Rising oil = inflation pressure = mildly supportive for gold as inflation hedge
	// But this is LOW weight — oil is a secondary contextual input
	impact := pctChange * 1.5 // scale down significantly
	impact = math.Max(-30, math.Min(30, impact))

	dir := DirNeutral
	if impact > 8 {
		dir = DirBullish
	} else if impact < -8 {
		dir = DirBearish
	}

	return DriverSnapshot{
		Name:        DriverOil,
		RawValue:    oilValue,
		NormalizedValue: impact,
		ImpactScore: impact,
		Direction:   dir,
		Confidence:  0.3, // LOW confidence
		Quality:     QualityConnected,
		Source:      "twelvedata",
		Timeframe:   "intraday",
		Reason:      "Oil " + formatFloat(pctChange) + "% — inflation context",
		Timestamp:   timestamp,
	}
}

// NormalizeRealYield converts a 10-year real yield value into a bounded impact score.
// Real yield is the 10-Year Treasury Inflation-Indexed Security yield (TIPS).
// This is the REAL yield, not nominal — they are semantically separate.
//
// Conceptually: falling real yields → supportive for gold (lower opportunity cost)
//               rising real yields → negative for gold (higher opportunity cost)
// But this is probabilistic and regime-dependent, NOT deterministic.
func NormalizeRealYield(realYield, prevRealYield float64, timestamp time.Time) DriverSnapshot {
	change := realYield - prevRealYield

	// Falling real yields → bullish for gold
	// Rising real yields → bearish for gold
	impact := -change * 30 // scale factor for yield changes
	impact = math.Max(-80, math.Min(80, impact))

	dir := DirNeutral
	if impact > 15 {
		dir = DirBullish
	} else if impact < -15 {
		dir = DirBearish
	}

	return DriverSnapshot{
		Name:            DriverRealYields,
		RawValue:        realYield,
		NormalizedValue: impact,
		ImpactScore:     impact,
		Direction:       dir,
		Confidence:      0.6, // medium-high confidence — real yield is a strong gold driver
		Quality:         QualityConnected,
		Source:          "fred",
		Timeframe:       "daily",
		Reason:          "Real yield " + directionWord(change) + " (" + formatFloat(realYield) + "%) → " + directionWord(impact) + " for gold",
		Timestamp:       timestamp,
	}
}
