# 06 — Market Data Connectivity Audit (end-to-end trace)

## Live path (verified)

Windows Agent/EA → `WS /ws/v1/agent` (**unauthenticated** — F-003) → `HandleAgentMessage` → validation → `processTick` → Aggregator M1..D1 → Valkey + Timescale (`market.ticks`, `market.candles`) → `/api/v1/market/state|snapshot|candles` → dashboards.

Runtime evidence:
- `market.ticks` n=7,109,183, last `2026-08-21 20:59:57+00`; M5 candles to 20:55 — matches Friday XAUUSD close; current UTC Sat 21:41 ⇒ gap is **correct weekend behavior**, not staleness. Engine healthy, 1 agent connected.
- News provider live: FMP sync every 5 min in logs through audit time.
- `/api/v1/market/state` during closed market returns zero-value top-level fields (`Timestamp:0001-01-01`, Bid/Ask "0") with honest per-candle `Quality/IsClosed` — but **no explicit MARKET_CLOSED state flag**; frontends render 0/hardcoded placeholders instead of a labeled closed state (see 24).

## Findings

| ID | Sev | Finding |
|---|---|---|
| 06-1 | P0 | Agent feed accepts any WS client as a data source; forged ticks/snapshots would flow into candles and gate hydration (Exposure/Margin/ExecutionPermit flip PASS on forged account_info for 30–60s). |
| 06-2 | P1 | 553 corrupted candles (`open=0`,`low=0`,`quality=COMPLETE`) Aug 18–21 prove aggregator seeding bug reached the live DB; downstream structure/FVG/pattern features read open/low. |
| 06-3 | P1 | Snapshot bars stamped `time.Now()` instead of bar time (`main.go:332,864`) distort candle-age/staleness math on MT5-sourced bars. |
| 06-4 | P2 | No broker symbol-alias table found; single hardcoded `XAUUSD` symbol mapping — multi-broker aliasing UNVERIFIED. |
| 06-5 | PASS | Duplicate/out-of-order tick handling exists in validation path; backpressure drop-on-full; provenance fields (`SourceMode/SourceSequence/GatewayReceiptTime`) persisted. |
| 06-6 | INFO | TwelveData DXY + FMP COT loops live and optional; COT weight=0; PTB marks unsupported capabilities honestly (`UNSUPPORTED_BY_DATA_SOURCE`). |

## Timestamp ladder per stage (design)

`SourceTimestamp→IngestTimestamp→MarketBarCloseTime→DetectedAt→QualifiedAt→PublishedAt` columns exist and populate on directional signals; NO-TRADE rows carry zero geometry (honest).
