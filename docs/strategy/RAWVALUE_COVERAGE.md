# RawValue Coverage — definitive checklist (Phase 0.75 Task A)

Scope: every evidence row's `RawValue` must be the REAL indicator read (never
a direction, score, or placeholder). Warmup semantics: an indicator that is
not ready emits NO evidence row — absence means "not ready", never a
fabricated zero. Legacy `addEvidence` stays for boolean/structural facts.

Status after Phase 0.75 Task A wiring (27 sites converted, TDD-verified):
RawValue non-zero on all indicator-bearing reads across the 4 core strategies.

## Wired (addEvidenceRaw, RawValue = live read)

| Strategy | Pillar | Feature | RawValue source |
|---|---|---|---|
| StandardScalping | TREND | EMA9_ABOVE/BELOW_EMA21 | `Indicators.EMA9` |
| StandardScalping | VWAP | ABOVE/BELOW_VWAP | `VWAP.SessionVWAP` |
| StandardScalping | MOMENTUM | MACD_BULLISH/BEARISH | `Indicators.MACDHistogram` |
| StandardScalping | MOMENTUM | RSI_BULLISH/BEARISH_MID | `Indicators.RSI` |
| StandardScalping | TREND | ADX_BULLISH/BEARISH | `Indicators.ADX` |
| StandardScalping | MOMENTUM | OSMA_POSITIVE/NEGATIVE | `Indicators.OsMA` |
| StandardScalping | MTF | ALIGNMENT_BULLISH/BEARISH | `MTF.Score` |
| StandardScalping | STRUCTURE | ABOVE/BELOW_DAILY_PIVOT | `Pivots.Daily.P` |
| UltraScalping | CANDLE | M1_BULLISH/BEARISH_DISPLACEMENT | `Candle.Range` |
| UltraScalping | MOMENTUM | OSMA_POSITIVE/NEGATIVE | `Indicators.OsMA` |
| UltraScalping | MOMENTUM | STOCH_BULLISH/BEARISH_CROSS | `Indicators.StochMain` |
| UltraScalping | MTF | ALIGNMENT_* | `MTF.Score` |
| UltraScalping | TREND | ADX_STRONG_* | `Indicators.ADX` |
| UltraScalping | VOLATILITY | BOLL_LOWER/UPPER_TOUCH | touched band (`BollLower`/`BollUpper`) |
| StandardSwing | TREND | EMA21_ABOVE/BELOW_EMA50 | `Indicators.EMA21` |
| StandardSwing | TREND | ABOVE/BELOW_SMA200 | `Indicators.SMA200` |
| StandardSwing | TREND | ADX_BULLISH/BEARISH | `Indicators.ADX` |
| StandardSwing | MOMENTUM | RSI_ABOVE/BELOW_50 | `Indicators.RSI` |
| StandardSwing | CANDLE | BULLISH/BEARISH_BREAKOUT_CLOSE | `Candle.Range` |
| StandardSwing | MTF | ALIGNMENT_* | `MTF.Score` |
| StandardSwing | TREND | ICHIMOKU_*_CROSS | `Indicators.IchimokuTenkan` |
| StandardSwing | TREND | ABOVE/BELOW_ICHIMOKU_CLOUD | `Indicators.IchimokuKijun` |
| StandardSwing | STRUCTURE | FIB_618_BOUNCE/REJECTION | `Fibonacci.Levels["0.618"]` |
| TrendSwing | TREND | ABOVE/BELOW_SMA200 | `Indicators.SMA200` |
| TrendSwing | TREND | ABOVE/BELOW_EMA50 | `Indicators.EMA50` |
| TrendSwing | TREND | EMA21_ABOVE/BELOW_EMA50 | `Indicators.EMA21` |
| TrendSwing | TREND | ADX_STRONG_* | `Indicators.ADX` |
| TrendSwing | MOMENTUM | MACD_*_CONTINUATION | `Indicators.MACDHistogram` |
| TrendSwing | MOMENTUM | CCI_BULLISH/BEARISH | `Indicators.CCI` |
| TrendSwing | MTF | ALIGNMENT_* | `MTF.Score` |
| TrendSwing | VWAP | ABOVE/BELOW_VWAP | `VWAP.SessionVWAP` |
| TrendSwing | TREND | SAR_BULLISH/BEARISH | `Indicators.ParabolicSAR` |
| MarnieFib | FIBONACCI | NEAR_GOLDEN_ZONE_BULL/BEAR | `ConfluenceScore` |
| MarnieFib | FIBONACCI | DISTANT_GOLDEN_BULL/BEAR | `ConfluenceScore` |
| MarnieFib | FIBONACCI | AT_FIB_LEVEL_BULL/BEAR | `NearestLevelPrice` |
| MarnieFib | FIBONACCI | HIGH_CONFLUENCE | `ConfluenceScore` |
| MarnieFib | TREND | EMA21_ABOVE/BELOW_EMA50 | `Indicators.EMA21` |
| MarnieFib | MOMENTUM | RSI_OVERSOLD/OVERBOUGHT | `Indicators.RSI` |
| MarnieFib | MOMENTUM | MACD_BULLISH/BEARISH | `Indicators.MACDHistogram` |
| TrendTransition (aux) | TREND | ADX_EXPANSION_BULL/BEAR | `Indicators.ADX` |
| TrendTransition (aux) | TREND | EMA_SLOPE_BULL/BEAR | `Indicators.EMA9` |
| TrendTransition (aux) | VOLATILITY | BB_EXPANSION_UPPER/LOWER | `Indicators.BollWidth` |
| TrendTransition (aux) | VOLATILITY | ATR_EXPANSION/_S | `Indicators.ATR` |

## Pre-existing wired sites (audited — RawValue is the real read, unchanged)

- StandardScalping/UltraScalping: TREND EMA9_ABOVE/BELOW_EMA21 → EMA9 (site 968/970, 1206/1208)
- VWAP sites (976/978, 1222/1224) → SessionVWAP
- MACD_BULLISH/BEARISH (1007/1009, 1447/1449) → MACDHistogram
- RSI_BULLISH/BEARISH_MID (1020/1022) → RSI
- ADX_BULLISH/BEARISH (1029/1031) → ADX

## Deliberate zero-RawValue (boolean/structural — documented exceptions)

BOS / HTF_BOS / BOS_CONFIRMED / HTF_CHoCH (structure events), displacement
and rejection candle flags (StandardScalping/StandardSwing variants),
liquidity sweeps (SELL/BUY_SIDE_SWEEP and reclaim/HTF variants), MTF alignment
was converted to raw (mtfScore is a real scalar), regime flags
(TRENDING_BULLISH/BEARISH — categorical), OB/FVG presence
(BULLISH/BEARISH_OB, *_FVG, CONTINUATION_*), HH_HL / LH_LL sequence,
PULLBACK_*_CONTINUATION (candle streak boolean), P1 volatility extras and P2
pullback/ORB/pin-bar helpers, RANGE evidence helpers (computeRangeEvidence /
computeUltraRangeEvidence — directional facts, no single scalar read).

## Warmup contract

- Any not-ready indicator (zero value / insufficient bars) emits NO evidence row.
- Covered by TestRawValueWarmupOmission (RSI=0 → no RSI evidence row).
- Endpoint and feature snapshot omit not-ready values (parity test:
  TestFeatureSnapshotRoundTrip family).

## Coverage measurement

- Live: evidence rows carry RawValue into `signal_feature_snapshots` (migration
  149) — `rawvalue coverage` = share of indicator-bearing evidence rows with
  non-zero RawValue post-warmup, visible in metrics pack (Task C adds the query).
- Test-level: per-strategy equality asserts RawValue == state field
  (TestRawValueStandardScalping et al.).
- Target: > 90% non-zero on post-warmup reads. Pre-wiring audit (Phase 0
  baseline): 98.6% of RawValues were zero because only TREND/VWAP were wired;
  the wiring above converts every indicator-bearing pillar row to carry its read.