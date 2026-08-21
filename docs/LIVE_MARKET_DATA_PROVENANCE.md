# Live Market Data Provenance

## Version: v1.8.0 — Trade Management Audit + Broker Stop Validation + Cost-Aware Break-Even

## Production Data Flow

```
MT4/MT5 Terminal (Master Node)
    ↓ tick data via FILE_COMMON
Windows Agent (Go)
    ↓ WebSocket wss://.../ws/v1/agent
Realtime Engine (Go)
    ↓ AgentProvider → MarketSnapshot
    ↓ tick validation → aggregator → state manager
    ↓ feature engines → indicators/structure/liquidity/regime/MTF/session
    ↓ PTB engine → 20 advanced modules + synthesis (SHADOW)
    ↓ strategy engines (4 independent)
    ↓ signal engine → 12 hard gates → signal
    ↓ persistence (TimescaleDB) + WebSocket broadcast
    ↓ MT4/MT5 client delivery
```

## Data Authenticity Guard

Production signal generation requires `source_type = LIVE_MASTER_NODE`.

**Accepted:** LIVE_MASTER_NODE, REPLAY (backtesting only)

**Rejected:** TEST, MOCK, DEMO, FIXTURE, SYNTHETIC, PLACEHOLDER, UNKNOWN

## Gold Correlation Data Provenance

The correlation engine tracks provenance for each external factor:

```
source
source_type
timestamp
age (ms)
value
quality (OK, STALE, INSUFFICIENT, UNAVAILABLE, INVALID)
availability
```

### Current External Feed Status

| Feed | Status | Reason |
|------|--------|--------|
| DXY | UNAVAILABLE | No DXY symbol from Master Node |
| Silver (XAGUSD) | UNAVAILABLE | No silver symbol from Master Node |
| US10Y Yields | UNAVAILABLE | No yield feed from Master Node |
| Real Yields / TIPS | UNAVAILABLE | No TIPS feed |
| COT Report | NOT CONNECTED | No external feed configured |

When external data is unavailable:
- Correlation returns `quality = UNAVAILABLE`
- Gold role returns `UNKNOWN`
- No fabricated correlations or classifications
- PTB continues safely with available data

## No Fake Data Policy

No production signal may depend upon:
- demo values, example prices, sample values
- random generated values, fixture values, mock market data
- placeholder returns, hardcoded current market prices
- hardcoded correlation values (e.g., "DXY = -0.85")
- fabricated probability, fabricated volume, fake institutional data
- synthetic order-flow claims, default numeric values pretending missing data exists

Tests may use deterministic fixtures under the TEST environment only. Those fixtures are structurally incapable of reaching production runtime.

## Data Quality Measurement

Quality is measured from real attributes, not assumed:

```
EXCELLENT — fresh data, valid tick, positive spread, ATR available
GOOD — minor delays but all critical data present
DEGRADED — data age > 10s or ATR unavailable
STALE — data age > 30s or tick quality stale
ERROR — nil state, invalid tick, no candles, negative spread
```

No hardcoded `quality = 90` without measured inputs.
