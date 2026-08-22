# 10 — Regime Engine Audit

Implementation: `realtime/internal/features/regime.go` v2.0.0 (+ `regime_thresholds.go`).

## Classification math (quoted from source)

```
bullish  = EMA9>EMA21 && EMA21>EMA50
bearish  = EMA9<EMA21 && EMA21<EMA50
trending = ADX>25 ; ranging = ADX<20
overbought = RSI>70 ; oversold = RSI<30   (RSI>0 guarded)
highVol  = ATR/Close > 0.002

priority: MEAN_REVERSION(0.7) > TRENDING_BULL/BEAR(0.8) > RANGE(0.6) > HIGH_VOLATILITY(0.5) > default RANGE(0.4)
```

## State reachability

Defined type: 11 regimes (`types.go:125-136`). `classifyRaw` can emit only 5: TRENDING_BULLISH, TRENDING_BEARISH, RANGE, MEAN_REVERSION, HIGH_VOLATILITY.
**BREAKOUT, LOW_VOLATILITY, LIQUIDITY_EVENT, NEWS_EVENT, UNSTABLE, NO_TRADE are unreachable** — yet BREAKOUT appears in `AcceptedRegimes` and threshold tables (`regime_thresholds.go:39-67`) → dead config branches (P2; misleading ops surface).

## Noise control (verified)

- Hysteresis: candidate must repeat on **3 consecutive evaluations**; min dwell **5 minutes**; forced switch when confidence ≤0.25 else decay ×0.92 per mismatched candle; first-candle bootstrap accepts raw regime; history capped 100.
- Fallback state: RANGE.

## Wiring into strategies (all verified)

1. Gate `checkRegimeSession` → NT_REGIME_MISMATCH (persisted ReasonCode seen in live API payload).
2. Regime-selects scoring thresholds via `GetThresholds(strat, regime)`.
3. Conflict penalty in scorer while RANGE.
4. TrendSwing transition mode for RANGE.
5. Persistence to `trading.regime_history` + `/api/v1/admin/regime-diagnostics` (endpoint unauthenticated — see 25).

Runtime sample: NO-TRADE TREND_SWING with `Regime=HIGH_VOLATILITY`, reason NT_REGIME_MISMATCH — consistent with code paths.

**Verdict:** engine VERIFIED-WIRED and deterministic; defects are unreachable-state config debt (P2) and unauthenticated diagnostics exposure (P1 security cross-ref).
