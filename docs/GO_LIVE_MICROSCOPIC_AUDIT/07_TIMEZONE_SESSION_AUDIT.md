# 07 — Timezone / Session Audit

## Canonical time

- Internal truth: **UTC** throughout (Go `time.Now().UTC()`, Postgres `timestamptz` everywhere, JWT/lease math UTC). OS: UTC. No local-time leakage found in DB writes.
- Display TZ handled client-side only; server emits RFC3339 Z timestamps.

## Session logic

- Implemented in `realtime/internal/features/session.go`: TOKYO/LONDON/NEW_YORK/OVERLAP classification with weekend OFF_HOURS blocking (`session.go:93-121`); gate state refreshed unconditionally PASS when market state exists — protection rides on live-input branch (`main.go:1736-1763`).
- Runtime evidence of correct DST-aware behavior: signals on Aug 21 carry `Session=NEW_YORK` at 23:45–00:00 UTC (EDT = UTC-4 ⇒ 19:45–20:00 ET NY afternoon) — consistent with US DST, not fixed-offset.
- Weekend close respected by engine (no evaluations producing directional signals after Friday close beyond final candle processing at Sat 00:00 — those are close-of-bar evaluations with NO-TRADE/regime-mismatch outcomes).

## Findings

| ID | Sev | Finding |
|---|---|---|
| 07-1 | P2 | Daily-loss reset boundary has **no implementation in the live plane** to audit (module unwired) → boundary semantics UNVERIFIED (ties F-004). |
| 07-2 | P3 | MT5-snapshot bars use ingest-time stamps (06-3) which also corrupts session attribution for those context candles. |
| 07-3 | PASS | Subscription period boundaries stored timestamptz (Aug 17→Sep 17); commission dates would derive from ledger rows (currently none). |
| 07-4 | UNVERIFIED | Broker-time (MT server) conversion inside EA not auditable without live terminal — EXTERNAL BLOCKER. |

No ambiguous timestamp formats found in API payloads (all RFC3339 with offset/Z).
