# 09 — Strategy Engine Audit (4 products)

Configs: `realtime/configs/` per-strategy YAML (timeframes, thresholds, expiry 3–240min, cooldowns); code `pkg/strategy/*.go` + engines addon. All four registered and evaluated every closed candle.

| Strategy | TF focus | Distinct logic evidence | BUY reachable | SELL reachable | NO-TRADE |
|---|---|---|---|---|---|
| STANDARD_SCALPING | M1–M5 | own thresholds/session profile | YES (BUY_CANDIDATE rows in DB) | YES (SELL rows) | YES (NT_* blockers) |
| ULTRA_SCALPING | M1 | tighter cost/spread caps, shortest TTL | YES | YES (DB sample) | YES |
| STANDARD_SWING | H1–H4 | structure+regime trend filters | YES | YES | YES (DB sample) |
| TREND_SWING | H1–D1 | excludes RANGE/MEAN_REVERSION (`strategies.go:1118`), transition mode | YES (DB BUY_CANDIDATE) | YES | YES (majority of its rows) |

DB proof: all four strategy IDs present in `trading.signals` with direction distribution including BUY_CANDIDATE, SELL_CANDIDATE and NO-TRADE; grades RESEARCH/CONFIRMED/NO-TRADE/Blocked observed. No permanently-unreachable directional branch found (unlike regime BREAKOUT).

## Gates inside strategies

- Regime/session check `checkRegimeSession` (`strategies.go:495-521`) → `NT_REGIME_MISMATCH` reason persisted (verified in API payload ReasonCodes).
- Conflict penalty +10 while RANGE (`strategies.go:325-328`).
- Per-strategy MinAbsATR via engines addon survives despite dead global gate.

## Findings

- **09-1 PASS:** four distinct versioned behaviors confirmed; NO-TRADE is first-class with named reason codes (`PrimaryBlocker`, `RejectionGate`) persisted to candidates/rejections tables.
- **09-2 P2:** threshold provenance: configs versioned on disk, but historical signals do not all snapshot config hash (InputHash/DecisionHash declared, never populated — `types.go`) ⇒ §97 partial.
- **09-3 P2:** WAIT state exists alongside NO-TRADE; dashboards conflate some states (frontend V5) — semantics preserved server-side only.
