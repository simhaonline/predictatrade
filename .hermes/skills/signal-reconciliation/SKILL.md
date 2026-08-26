---
name: signal-reconciliation
description: "Reconcile server signals with EA execution results."
---

# signal-reconciliation

Use when reconciling Predict-A-Trade server signals with EA execution results, tracking latency, or debugging trade attribution.

## Signal Lifecycle
1. Strategy Engine → signal (entry, SL, TP1/2/3, strategy_id, risk$)
2. Hard Gates → validate (14-gate, fail-closed)
3. SL Enforcement → server-side SL/TP
4. WS → Windows Agent → pipe → EA
5. EA validates SL sign, margin, duplicate ID
6. OrderSend / PositionOpen → broker
7. Order result → EA report → WS → server
8. Server matches signal_id → ticket → fill → P&L

## Reconciliation Per Signal
- signal_id: server log = EA comment = broker history
- Latency: signal_timestamp → order_open_time (target < 500ms)
- Fill: requested_lot vs filled_lot
- Slippage: signal_entry vs actual_fill
- SL: server_sl == broker_sl (+/- 0.5 points)

## Reconciliation Per Session
- Signals sent = EA received minus TTL expired
- Signals executed = broker positions opened
- Strategy attribution: every P&L has strategy_id
- No orphan positions (manual/unauthorized)

## Known Gaps (audit findings)
- No strategy_id in EA comment
- Wrong-side SL placed
- TP honored only ~9.5% of winners
