# Market Data Trace Report — 2026-08-23

## Pipeline
```
MT5 Terminal → Master Node EA → FILE_COMMON pipes → Windows Agent
→ wss://live.predictatrade.com/ws/v1/agent → Go AgentProvider
→ Tick Validator → Aggregator → IndicatorEngine
→ Strategy Engines (×4) → Signal Engine → Hard Gates (×12)
→ Calibration → PostgreSQL → WebSocket + HTTP API
→ NestJS → Next.js Dashboards → Windows Agent → MT5 EA
```

## Provenance
- Source: MT5_MASTER (authoritative)
- Quality: AUTHORITATIVE
- Timestamps: UTC (ISO8601 from MT5 EA)
- No synthetic data in live path

## Current State (Saturday — market closed)
- agents_connected: 0 (market closed)
- Last tick: 2026-08-21 20:59 UTC (Friday close)
- ATR: 7.27 (from bootstrap candles)
- Regime: TRENDING_BULLISH
- Session: NEW_YORK (stale from Friday)
