// feature_snapshot_builder.go — P0-2 (prompt.md): the per-signal feature
// snapshot builder. Converts the live features.MarketState indicator read set
// into a schema-versioned JSON payload persisted alongside each signal.
//
// Contract:
//   - every indicator the decision used appears with its ACTUAL raw value
//   - warmup (zero-valued) indicators are OMITTED, never emitted as
//     fabricated zeros — absence means "not ready", 0 means "read 0.0"
//   - version metadata travels inside the payload (schema/feature/indicator-set)
package marketdata

import (
	"encoding/json"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/shopspring/decimal"
)

// FeatureSnapshotSchemaVersion is bumped whenever the snapshot field set changes
// so consumers can branch on the exact shape they parsed.
const FeatureSnapshotSchemaVersion = "1.0"

// FeatureSnapshotIndicatorSetVersion tracks the indicator definition set
// (which indicators exist and their warmup windows).
const FeatureSnapshotIndicatorSetVersion = "1.0"

func fsDecFloat(d decimal.Decimal) (float64, bool) {
	if d.IsZero() {
		return 0, false // warmup/absent — omit
	}
	f, _ := d.Float64()
	return f, true
}

func putFloat(m map[string]any, key string, d decimal.Decimal) {
	if v, ok := fsDecFloat(d); ok {
		m[key] = v
	}
}

func putBool(m map[string]any, key string, b, known bool) {
	if known {
		m[key] = b
	}
}

// BuildFeatureSnapshotJSON serializes the indicator read set of a MarketState.
// Never fails: unknown/absent fields are simply omitted (warmup semantics).
func BuildFeatureSnapshotJSON(st *features.MarketState) ([]byte, error) {
	out := map[string]any{
		"_schema_version":       FeatureSnapshotSchemaVersion,
		"_indicator_set_version": FeatureSnapshotIndicatorSetVersion,
		"symbol":                st.Symbol,
	}

	if st == nil {
		return json.Marshal(out)
	}

	// ── Trend ──
	putFloat(out, "ema9", st.Indicators.EMA9)
	putFloat(out, "ema21", st.Indicators.EMA21)
	putFloat(out, "ema50", st.Indicators.EMA50)
	putFloat(out, "ema100", st.Indicators.EMA100)
	putFloat(out, "ema200", st.Indicators.EMA200)
	putFloat(out, "sma50", st.Indicators.SMA50)
	putFloat(out, "sma100", st.Indicators.SMA100)
	putFloat(out, "sma200", st.Indicators.SMA200)
	if st.Indicators.EMACross921 {
		out["ema_cross_9_21"] = "bull"
	} else {
		out["ema_cross_9_21"] = "bear"
	}

	// ── Momentum ──
	putFloat(out, "macd_main", st.Indicators.MACDMain)
	putFloat(out, "macd_signal", st.Indicators.MACDSignal)
	putFloat(out, "macd_hist", st.Indicators.MACDHistogram)
	putFloat(out, "rsi", st.Indicators.RSI)
	putFloat(out, "stoch_rsi_k", st.Indicators.StochRSIK)
	putFloat(out, "stoch_rsi_d", st.Indicators.StochRSID)
	putFloat(out, "cci", st.Indicators.CCI)
	putFloat(out, "adx", st.Indicators.ADX)
	putFloat(out, "adx_plus_di", st.Indicators.ADXPlusDI)
	putFloat(out, "adx_minus_di", st.Indicators.ADXMinusDI)
	putFloat(out, "psar", st.Indicators.ParabolicSAR)
	putBool(out, "psar_long", st.Indicators.ParabolicSARLong, !st.Indicators.ParabolicSAR.IsZero())
	putFloat(out, "ichimoku_tenkan", st.Indicators.IchimokuTenkan)
	putFloat(out, "ichimoku_kijun", st.Indicators.IchimokuKijun)
	putFloat(out, "ichimoku_senkou_a", st.Indicators.IchimokuSenkouA)
	putFloat(out, "ichimoku_senkou_b", st.Indicators.IchimokuSenkouB)
	putFloat(out, "ichimoku_cloud_top", st.Indicators.IchimokuCloudTop)
	putFloat(out, "ichimoku_cloud_bot", st.Indicators.IchimokuCloudBot)

	// ── Volatility ──
	putFloat(out, "atr", st.Indicators.ATR)
	putFloat(out, "choppiness", st.Indicators.Choppiness)
	putBool(out, "squeeze_on", st.Indicators.SqueezeOn, st.Indicators.SqueezeOn)
	putBool(out, "squeeze_release", st.Indicators.SqueezeRelease, st.Indicators.SqueezeRelease)
	putFloat(out, "lin_slope_atr", st.Indicators.LinSlopeATR)
	putFloat(out, "lin_r2", st.Indicators.LinR2)
	putFloat(out, "adx_slope", st.Indicators.ADXSlope)
	putFloat(out, "clv", st.Indicators.CLV)
	putFloat(out, "bb_upper", st.Indicators.BollUpper)
	putFloat(out, "boll_lower", st.Indicators.BollLower)
	putFloat(out, "boll_middle", st.Indicators.BollMiddle)
	putFloat(out, "boll_width", st.Indicators.BollWidth)
	putFloat(out, "bb_width_z", st.Indicators.BBWidthZScore)
	putBool(out, "boll_bull_rev", st.Indicators.BollBullRev, st.Indicators.BollBullRev)
	putBool(out, "boll_bear_rev", st.Indicators.BollBearRev, st.Indicators.BollBearRev)

	// ── Volume / flow ──
	putFloat(out, "obv", st.Indicators.OBV)
	putFloat(out, "obv_z", st.Indicators.OBVZScore)
	putFloat(out, "tick_volume_z", st.Indicators.TickVolumeZScore)

	// ── VWAP ──
	putFloat(out, "vwap_session", st.VWAP.SessionVWAP)
	putFloat(out, "vwap_upper", st.VWAP.UpperBand)
	putFloat(out, "vwap_lower", st.VWAP.LowerBand)
	putFloat(out, "vwap_rolling", st.VWAP.RollingVWAP)

	// ── Structure / regime / session (scalar reads where available) ──
	if st.Session.CurrentSession != "" {
		out["session"] = st.Session.CurrentSession
	}
	putBool(out, "is_overlap", st.Session.IsOverlap, st.Session.IsOverlap)
	putBool(out, "asian_sweep_high", st.SessionORB.AsianSweepHigh, st.SessionORB.AsianSweepHigh)
	putBool(out, "asian_sweep_low", st.SessionORB.AsianSweepLow, st.SessionORB.AsianSweepLow)
	if st.Session.NewsRisk != "" {
		out["news_risk"] = st.Session.NewsRisk
	}
	if st.Regime.Current != "" {
		out["regime"] = string(st.Regime.Current)
	}
	// P1: regime-direction allowance + squeeze veto state (strategy pkg
	// RegimeAllowsDirection mirrored here for the snapshot consumers).
	out["regime_direction_allowed"] = regimeAllowsDirectionSnapshot(string(st.Regime.Current))
	// P2 (prompt.md): crossmarket drivers + astro in the snapshot.
	if st.CrossMarketBiasX6 != 0 {
		out["macro_bias_x6"] = st.CrossMarketBiasX6
	}
	for name, v := range st.CrossMarketMomPct {
		out["mom_pct_"+name] = v
	}
	if as, ok := st.Astro.(features.AstroSnapshot); ok {
		out["astro_sizing_multiplier"] = as.Vedic.SizingMultiplier
		out["astro_yoga_bias"] = as.Vedic.YogaBias
		switch {
		case as.Vedic.DemonHours.RahuKalam:
			out["astro_demon_hour"] = "rahu_kalam"
		case as.Vedic.DemonHours.Yamaganda:
			out["astro_demon_hour"] = "yamaganda"
		case as.Vedic.DemonHours.Gulika:
			out["astro_demon_hour"] = "gulika"
		}
		out["astro_gandanta"] = as.Vedic.IsGandantaNow
	}

	return json.Marshal(out)
}
// regimeAllowsDirectionSnapshot mirrors strategy.RegimeAllowsDirection for the
// snapshot payload (marketdata cannot import strategy — dependency direction
// is strategy → marketdata). Kept in lockstep with the gate by tests.
func regimeAllowsDirectionSnapshot(regime string) string {
	switch regime {
	case "TRENDING_BULLISH":
		return "BUY_ONLY"
	case "TRENDING_BEARISH":
		return "SELL_ONLY"
	case "RANGE":
		return "NONE" // squeeze veto
	default:
		return "BOTH"
	}
}
