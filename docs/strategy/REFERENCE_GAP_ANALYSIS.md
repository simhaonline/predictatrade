# reference.md → Predict-A-Trade Engine Gap Analysis

Source: `reference.md` (3,480-line MQL5 EA "Predict-A-Trade XAUUSD Institutional
Intraday Confluence v1.00"). Audited against `realtime/internal/features`,
`strategy`, `astro`, `gates`, `risk`, `cmd/realtime-engine` (2026-09-17).

## 1. Indicator reading flow — status

The indicator reading flow **works** but has two delivery defects:

| # | Finding | Severity | Where |
|---|---------|----------|-------|
| 1 | `EvidenceContribution.RawValue` is never set by `addEvidence()` — every evidence row ships `raw_value: 0` while `normalized_value` carries the real contribution. Indicator *readings* (RSI=43.8, ADX=43.8, ATR…) never leave the engine. | HIGH (observability/EA contract) | `internal/strategy/strategies.go addEvidence()` |
| 2 | No surface exposes raw indicator values: no `/api/v1/indicators`-style endpoint, WS hub has zero indicator streaming, `feature_snapshot_id` is NULL on 100% of recent signals (column exists, never written). Reference EA's HUD/`LogScoreParts` equivalent is missing server-side. | HIGH | gateway WS + REST |
| 3 | `feature_version` is always `"1.0"` — there is no versioned feature-snapshot artifact to replay a signal's exact indicator state. | MEDIUM | persistence |

Live check: regime engine reads live ADX/RSI/ATR (43.8/39.3/518) — the
computation path is alive; only the *exposure* of reads is broken.

## 2. Features the reference has that we lack

### Candle anatomy / volume flow (scoreCandle + sVol families)
- `clv` — close-location value `(2C−H−L)/(H−L)`, drives ±1 sCandle and gates breakout/absorption signs
- `wickAtr` — wick size normalized by ATR (pin-bar strength ≥0.4 threshold)
- `candleATRNorm` / `bodyRatio` decomposition (body vs upper/lower wick separately)
- `absorption` — high volume, no price progress (used +1/−1 in sVol, +candle gate)
- `volumeSpike` flag feeding direction via CLV
- `microReclaimBull/Bear` — wick pierces level then closes back inside
- `bullStreak/bearStreak` — consecutive close runs

### Volatility/structure
- `atrZ` — ATR z-score (expansion/compression context)
- `chop` — Choppiness Index (regime filter, squeeze family)
- `squeezeOn / squeezeRelease` — BB-inside-KC state + release direction (scored ±2 with slope sign)
- `linSlopeAtr, linR2` — linear-regression slope (ATR-normalized) + R², used in sCandle (R²≥0.45 gate)
- `adxSlope` — ADX delta across bars (momentum of momentum)
- `pivD..pivDS3, pivW..pivWS3` — floor-trader pivots R1-R3/S1-S3 daily AND weekly (we have daily/weekly pivots — verify the R3/S3 depth)
- `pdh, pdl, pdo, pdc` — previous-day OHLC as levels
- `ibLonHi/Lo, ibNyHi/Lo, ibWidthAtr, inIB, brokeIBhigh/low` — initial balance (London/NY) + break scoring (±3)
- `asiaSweepHi/Lo` — Asian-session sweep filter (±3, operator-gated)
- `sweepHi/sweepLo` — generic liquidity sweeps (we have sweep detection in liquidity.go — check array form)
- `eqhDetected/eqlDetected` — equal highs/lows
- `prevSwingHigh/Low` — second-order structure reference for SL placement
- `breakerBull/Bear, brkBullT/B, brkBearT/B` — breaker blocks with zone boundaries
- `fvgsBull/Bear[]` arrays with per-zone fill state (we have FVG detection — verify zone-array + in-zone checks)
- `obsBull/Bear[]` order-block arrays with top/bottom + in-zone checks
- `liqHi[]/liqLo[]` liquidity pool arrays (n levels each side)

### Geometry
- `vwapSession, vwapUpper1/2, vwapLower1/2, vwapRoll, vwapZ` — full VWAP band stack (we have session+rolling VWAP; verify band ±1σ/±2σ and z-score)
- `fibHigh/Low/Retr/Near/Ext1/Ext2, inGoldenZone` — fib retracement + golden zone + extensions (we have golden zone; verify ext1/ext2 levels)
- `marnieRetr/Near/Ext1/Ext2, marnieInGZ` — Marnie variant levels + its own golden zone (we have MarnieFib strategy — check inGZ flag exists as feature)
- `confluenceHits` — count of simultaneous level touches, ≥3 adds ±2

### Scoring model (the big structural gap)
- **8-family signed-sum composite** (Trend/MTF/Momentum/Volume/SMC/Geometry/Candle/Macro + optional Astro), scaled ×1.5, clamped ±100
- Per-family sub-scores persisted (`scoreTrend…scoreAstro`) for score-part diagnostics
- Tier ladder A+/A/B/C/Watch with distinct thresholds (55/42/30/22/12) + `tierReason`
- `todBucket` — time-of-day bucket added INTO the candle score
- `RegimeAllows()` — regime-direction gating (trending-bull only longs, squeeze = no trade)
- Astro score spliced as `GetCosmicScore(dir) × weight% × 0.15`

### Astro (we have: nakshatra, hora, dasha, western score — missing the depth)
- `shadbala` per-planet strength (drives both score and sizing)
- Planetary karaka selection: Sun for gold, Mercury otherwise
- Yoga suite: Gajakesari (+15), GuruMangala (+12), Budhaditya (+8), ChandraMangala (+10), Dhana (+10), Vish (−20), Yama (−25), Grahan (−15)
- `isGandanta` (−30), `isRahuKalam` (−15), Yamaganda/Gulika demon hours
- `CalcTaraBala` score ×10, `CalcMicroTradeMap` composite bias ×0.10
- **Astro sizing multiplier** `0.8 + (shadbala/100)×0.5`, ×0.65 in demon hours, clamp [0.5,1.5] — position-size modulation, we have nothing equivalent
- `FilterDemonHours / FilterGandanta` operator toggles

### Drivers (crossmarket — we have DXY/VIX/BTC/OIL; verify momentum form)
- `MomPct` momentum over N bars per driver + Corr matrix + combined `Bias()` scaled ×6 into sMacro
- `InpReal10y / InpFedCtx / InpCotNetChg` manual overlays (we have COT/FRED — verify they feed the same bias channel)

### News
- `newsRisk + newsEventName` carried into features; impact filter levels (HIGH_ONLY etc.), before/after windows as EA inputs

### Execution/risk model (EA-side, informs our EA contract)
- SL: structure distance clamped to [0.8,1.8]×ATR, hard cap `MaxSlUSD` ($8), broker stops-level +2pt floor, +1.5×spread buffer
- Partials: TP1 50% / TP2 30%, BE+cost after TP1, ATR trail starting at 1R, step-broker-TP
- Risk: 0.5%/trade, day −3%/week −6% loss caps, maxDD 10%, equity floor, profit lock +10%, 20 trades/day, 4 consecutive losses, 20min cooldown, recovery factor 0.5
- Cost model: commission/lot round-turn, slippage allowance pts, swap/lot/night, avoid-negative-swap side selection
- Friday close hour flatten; full state reset on restart

## 3. Indicator reading flow — correctness checks (our side)

Live-verified 2026-09-17:
- regime engine: ADX 43.8 / RSI 39.3 / ATR 518 / RANGE / BEARISH — live reads working
- evidence pillars reaching signals: TREND, MOMENTUM, SMC, STRUCTURE, MTF, VWAP, VOLATILITY, CANDLE, REGIME, WESTERN_ASTRO, DI_DASHA/HORA/NAKSHATRA, CONSENSUS, CROSS_MARKET
- 41 indicator unit tests across indicators_test.go + new_indicators_test.go
- defects: `raw_value` always 0 (never set), no feature-snapshot persistence, no raw-indicator exposure endpoint, WS does not stream indicator reads

## 4. Priority order

P0 (breaks the "reading flow works perfectly" requirement):
1. Fill `RawValue` in addEvidence (pass the actual indicator read per feature)
2. Persist a per-signal feature snapshot (indicator values + version) and set `feature_snapshot_id`
3. Expose `GET /api/v1/indicators/snapshot` (or WS stream) with the SFeatures-equivalent read set

P1 (score parity with reference):
4. Missing indicators: CLV, wickAtr-as-feature, absorption, micro-reclaim, chop, squeeze state/release, linreg slope/R², adxSlope, IB breaks, asiaSweep, PDH/PDL/PDO/PDC, streaks
5. VWAP bands ±1σ/±2σ + vwapZ; fib ext1/ext2; Marnie inGZ; confluenceHits counter
6. RegimeAllows-style regime-direction gating + squeeze veto

P2 (depth):
7. Astro: shadbala, yoga suite, demon hours/gandanta filters, sizing multiplier
8. Macro driver momentum% + correlation matrix feeding sMacro
9. News event name + impact windows as first-class feature fields

P3 (EA contract parity):
10. MaxSlUSD cap, partials/BE/trail spec on EA commands, cost-model parity