# Live Dashboard (live.predictatrade.com) — Issues, Fixes & Missing API Calls

Generated: 2026-08-23
Scope: The public view-only live dashboard served at `live.predictatrade.com`
(static file `/var/www/pat-live/index.html`, sourced from `realtime/web/live.html`).
Backend: Go realtime engine (`realtime/`) exposed via nginx: `/api/v1/*` and `/ws` → `realtime:13081`.

## 1. Issues Found (verified against live API responses)

| # | Issue | Root cause | Severity |
|---|-------|-----------|----------|
| 1 | Regime, Session, MTF score, VWAP, Confidence never render (always `—`) | `loadMarketState()` parsed `market/state` fields at the **top level**, but the engine nests them under `states[0]` (`{states:[{Regime, MTF, Session, VWAP, Indicators, Candles,...}]}`) | HIGH |
| 2 | Same bug in `handleWS()` MARKET_STATE branch (EventEnvelope payload also wrapped in `states[0]`) | identical parse bug | HIGH |
| 3 | Price chart shows "Loading chart data…" forever | `STATE.candles` only built from `/api/v1/price/history` (empty); real aggregated candles in `market/state.states[0].Candles` were never read | HIGH |
| 4 | Misleading "● LIVE" + "AUTHORITATIVE" labels when no live feed | No honest data-source state; with no MT5 master node the engine reports `source: LOCAL_COMPUTE_ONLY` but UI still claimed LIVE | MEDIUM (SOW honesty) |
| 5 | No indication of observation/demo mode | Users could mistake aggregated historical indicators for live trading data | MEDIUM (SOW honesty) |
| 6 | Indicators (RSI/ATR/ADX/MACD/CCI) actually DID render (snapshot `indicators` shape matched) — confirmed working | — | OK |

Live API snapshot (2026-08-23): `agents_connected:1`, `master_node_connected:true` but `snapshot_count:0`,
`market/snapshot.source = LOCAL_COMPUTE_ONLY`, `price/history = {prices:[]}`, `candles = []` (table empty).
So the engine computes indicators locally from **aggregated historical candles** but has **no live tick stream** persisted.

## 2. Fixes Applied (to `realtime/web/live.html` + deployed `/var/www/pat-live/index.html`)

- **Fixed `loadMarketState()`** to unwrap `states[0]` and map `Regime.Current/Volatility/Confidence`,
  `MTF.Score/States`, `Session.CurrentSession/IsOverlap/IsWeekend/NewsRisk`, `VWAP.SessionVWAP`,
  and `Candles` → now populates regime/session/MTF/VWAP/chart.
- **Fixed `handleWS()` MARKET_STATE** to unwrap `m.payload.states[0]`.
- **Added `buildCandlesFromMarketState()`** — renders the real aggregated OHLC candles already returned by
  `market/state` (M5/H1/D1…) into `STATE.candles`, sets day-open/high/low and volume.
- **Added `loadCandles()`** calling `/api/v1/candles?symbol=XAUUSD&timeframe=M5&limit=200` (TimescaleDB);
  used as the preferred chart source when available, falls back to aggregated market-state candles.
- **Added honest mode banner** (`#modeBanner`): `LIVE` (master node + live source) / `OBSERVATION MODE`
  (no master node — aggregated historical, not live trading data) / `DATA UNAVAILABLE`.
- **Corrected header `● LIVE` pill and `AUTHORITATIVE` → `LIVE FEED` / `AGGREGATE`** to reflect reality.
- **Chart source label** (`SRC: AGGREGATE | DB`) added to the price chart.
- JS syntax validated with `node --check`.

## 3. Missing / Pending API Calls & Data Gaps

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

## 4. Verification

- `node --check` on extracted inline scripts: PASS.
- Live endpoints confirmed returning data: `market/state` (states[0] with Regime/MTF/Session/VWAP/Candles),
  `market/snapshot` (indicators), `agents/status`, `strategies`, `system-health`.
- After deploy, the dashboard now renders regime/session/MTF/VWAP/indicators and aggregated candles,
  and shows an honest OBSERVATION MODE banner instead of a false LIVE state.
