# Live Dashboard (live.predictatrade.com) — Missing API Calls & Data Gaps

Generated: 2026-08-23
Scope: The public view-only live dashboard served at `live.predictatrade.com`
(static file `/var/www/pat-live/index.html`, sourced from `realtime/web/live.html`).
Backend: Go realtime engine (`realtime/`) exposed via nginx: `/api/v1/*` and `/ws` → `realtime:13081`.

## Context

The dashboard already calls these endpoints (all implemented in the Go engine and verified returning data):
- `GET /api/v1/market/snapshot` — indicators (42 keys: RSI/ATR/ADX/MACD/CCI/…)
- `GET /api/v1/market/state` — `states[0]` with Regime, MTF, Session, VWAP, Indicators, Candles
- `GET /api/v1/agents/status` — agents_connected, master_node_connected, snapshot_count
- `GET /api/v1/strategies` — the 4 strategies
- `GET /api/v1/candles?symbol=XAUUSD&timeframe=M5&limit=200` — TimescaleDB candle history
- `GET /api/v1/price/history` — tick price history
- `GET /api/v1/system-health` — health of engine/PG/Valkey
- `WS /ws` — EventEnvelope broadcasts (MARKET_STATE, MARKET_SNAPSHOT, SIGNAL, AGENT_STATUS)

Live API snapshot (2026-08-23): `agents_connected:1`, `master_node_connected:true` but
`snapshot_count:0`, `market/snapshot.source = LOCAL_COMPUTE_ONLY`, `price/history = {prices:[]}`,
`candles = []` (table empty). The engine computes indicators locally from **aggregated historical
candles** but has **no live tick stream persisted**.

## Missing / Pending API Calls & Data Gaps

These are capabilities the dashboard needs but the backend does **not** currently provide live data for.
They are genuine gaps (not frontend bugs) — required for full production grade.

- [ ] **Live tick price (bid/ask/spread)** — only present when an MT5 **Master Node** is streaming.
      `market/snapshot` returns `tick` only with a connected node; currently `snapshot_count:0`.
      → Connect/qualify the MT5 Windows Agent master node, or verify the agent→`/ws/v1/agent` stream is flowing.
- [ ] **Persisted candle history** — `/api/v1/candles` and `/api/v1/price/history` both return empty because
      no ticks are being persisted to `market.candles` / price history. The `market/state` endpoints expose
      *aggregated* candles, but the time-series tables the chart APIs read are empty.
      → Fix the ingestion/persistence pipeline (ticks → `market.candles` + price history) so `/api/v1/candles` returns rows.
- [ ] **Real signals** — `/api/v1/signals` exists but the dashboard does **not** call it; strategy "BUY/SELL/HOLD"
      is currently *derived heuristically* from the MTF score (could be misread as a trade signal).
      → Dashboard should consume `/api/v1/signals` and label them explicitly as engine signals (not derived).
- [ ] **Broker name / volume / open positions / account info** — only available via `market/snapshot.account_info`
      / `symbol_info` / `positions` when a master node is connected. Currently absent → UI shows `—` (honest).
- [ ] **WebSocket push to browser clients** — engine broadcasts `MARKET_STATE`/`MARKET_SNAPSHOT`/`SIGNAL`/
      `AGENT_STATUS` EventEnvelopes, but without a master node there is no periodic push, so the WS indicator
      stays `OFFLINE` while REST polling carries the data. With a live master node this resolves automatically.
      → Confirm `BroadcastMarketState` is invoked on a cadence (or on each snapshot) for browser clients.
- [ ] **News / macro risk** — `Session.NewsRisk` returns `DATA_UNAVAILABLE`; no news provider wired. Optional for v1.0.0.

## Related Fixes Already Applied (see live-api-gaps.md)

- `loadMarketState()` / `handleWS()` MARKET_STATE now unwrap `states[0]` → regime/session/MTF/VWAP render.
- `buildCandlesFromMarketState()` + `loadCandles()` populate the price chart from aggregated/DB candles.
- Honest mode banner (LIVE / OBSERVATION MODE / DATA UNAVAILABLE) replaces false "LIVE" labeling.
