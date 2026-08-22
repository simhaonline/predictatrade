# 08 — Indicator Mathematical Audit

Scope: `realtime/internal/features` (13 engines, 42-feature vector) vs `research/src/patresearch/reference_math.py` parity baseline + `pkg/math` (Wilder smoothing).

## Method & evidence

- Independent implementations exist and are TESTED: research pytest suite **127/127 passed** during audit (includes indicator parity fixtures); Go targeted suites (`internal/features`, `pkg/math`) pass.
- Warm-up/NaN handling verified in code review: RSI=0 guard documented (`regime.go:36-37`); capability-honest labels when insufficient history; closed-bar-only evaluation (no forming-candle inputs) — look-ahead negative at feature layer (see 12).
- Live DB spot-check: OHLC invariant violations = 0 across 418,041 M5 candles; BUT 553 bars carry open=0/low=0 (06-2) which any range-based feature consumes as valid — contamination is real for Aug 18–21 windows.

## Indicator inventory status

| Class | Status | Note |
|---|---|---|
| EMA 9/21/50/100/200, SMA | WIRED+TESTED | regime uses 9/21/50 |
| MACD 12/26/9, ADX 14 (Wilder), RSI 14, ATR 14 | WIRED+TESTED | Wilder parity in pkg/math |
| Bollinger 20/2 + width, Stoch, StochRSI, CCI, Ichimoku | WIRED | covered by registry tests |
| OBV / Volume Profile / CumDelta / VWAP | PARTIAL | tick-volume semantics labeled broker-local; no centralized-volume claim (SOW-compliant) |
| Structure: BOS/CHoCH/FVG/swings/liquidity/sweeps | WIRED | consume open/low ⇒ affected by 06-2 corruption window |
| Fibonacci/pivots/confluence | WIRED | confluence weights config-versioned |

## Findings

- **08-1 P1:** correctness of math is evidenced by parity tests, but **no golden end-to-end fixture ties a persisted historical signal to independently recomputed indicators** (§98) — reproducibility UNVERIFIED beyond unit level.
- **08-2 P1:** corrupted-bar ingestion (open/low=0, quality COMPLETE) invalidates "indicators correct" for affected periods until backfill/reseed + quality-gate fix.
- **08-3 PASS:** no look-ahead detected: features evaluate on closed bars only; replay engine labels SYNTHETIC_REPLAY and is not imported live.
