---
name: mt4-mt5-windows
description: "Implement Go Windows Agent and MQL4/MQL5 EAs."
---

# mt4-mt5-windows

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Components
- Windows Agent (Go, port 9000 health, WebSocket to realtime engine)
- MT4 EA (PredictATrade_MT4.mq4, 2068 lines)
- MT5 EA (PredictATrade_MT5.mq5, 2089 lines)
- Named-pipe IPC via MetaQuotes Common/Files

## Magic-Number Ranges
- STANDARD_SCALPING 40101, ULTRA_SCALPING 40201, STANDARD_SWING 40301, TREND_SWING 40401, MARNIE_FIB 40501

## Workflow
1. Keep predictive/risk truth server-side.
2. Validate license/device/account/entitlement before delivery.
3. Secure stream with heartbeat, TTL, sequence, replay/idempotency.
4. Version local IPC between Agent and MT4/MT5.
5. Broker symbol discovery, disconnect/reject handling.

## Validate
MT4/MT5 clean compile, reconnect, revocation propagation. Never embed server/private credentials.
