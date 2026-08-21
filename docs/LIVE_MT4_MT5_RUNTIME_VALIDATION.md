# Live MT4/MT5 Runtime Validation Runbook

**Version:** v1.9.0  
**Status:** SOFTWARE_VERIFIED_RUNTIME_PENDING  

This document describes the exact runtime validation steps required to verify
the Predict-A-Trade platform against a live MT4/MT5 terminal.

**CRITICAL: All diagnostics default to READ-ONLY / DRY-RUN / TEST MODE.**
Do NOT send live orders as part of validation without separate explicit authorization.

---

## Prerequisites

1. MT4 or MT5 terminal installed and connected to a broker
2. PredictATrade MT4.mq4 or PredictATrade_MT5.mq5 EA attached to XAUUSD chart
3. Windows Agent running and connected to the Go engine via WebSocket
4. Go engine running (port 13081)
5. Frontend running (port 13082)
6. Database (PostgreSQL) running (port 5432)
7. Valkey running (port 6379)

---

## Validation Steps

### Step 1: Terminal Connection

```bash
# Check that Windows Agent is connected
curl -s http://127.0.0.1:13081/api/v1/agents/status | python3 -m json.tool
# Expected: agents > 0, connected: true
```

### Step 2: Broker Connection

```bash
# Check that market data is flowing (live ticks)
curl -s http://127.0.0.1:13081/api/v1/market/snapshot | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'Symbol: {d.get(\"symbol\",\"?\")}')
print(f'Bid: {d.get(\"bid\",\"?\")}')
print(f'Ask: {d.get(\"ask\",\"?\")}')
print(f'Spread: {d.get(\"spread\",\"?\")}')
print(f'Source: {d.get(\"source_type\",\"?\")}')
"
# Expected: source_type = LIVE_MASTER_NODE, bid/ask are real prices
```

### Step 3: Account Identification

```bash
# Check broker account info from agent snapshot
curl -s http://127.0.0.1:13081/api/v1/agents/status | python3 -c "
import sys, json
d = json.load(sys.stdin)
for a in d.get('agents', []):
    print(f'Agent: {a.get(\"agent_id\",\"?\")}')
    acct = a.get('account_info', {})
    print(f'  Login: {acct.get(\"login\",\"?\")}')
    print(f'  Balance: {acct.get(\"balance\",\"?\")}')
    print(f'  Equity: {acct.get(\"equity\",\"?\")}')
    print(f'  Leverage: {acct.get(\"leverage\",\"?\")}')
"
```

### Step 4: Symbol Mapping

```bash
# Verify XAUUSD symbol mapping
curl -s http://127.0.0.1:13081/api/v1/market/snapshot | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'Symbol: {d.get(\"symbol\",\"?\")}')
print(f'Digits: {d.get(\"digits\",\"?\")}')
print(f'Tick size: {d.get(\"tick_size\",\"?\")}')
"
# Expected: symbol=XAUUSD, digits=2, tick_size=0.01
```

### Step 5: Tick Flow

```bash
# Monitor tick flow for 30 seconds
for i in $(seq 1 10); do
    curl -s http://127.0.0.1:13081/api/v1/market/snapshot | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'{d.get(\"bid\",\"?\")}/{d.get(\"ask\",\"?\")} spread={d.get(\"spread\",\"?\")}')
"
    sleep 3
done
# Expected: bid/ask change over time (live market)
```

### Step 6: Spread Check

```bash
# Verify spread is within acceptable range
curl -s http://127.0.0.1:13081/api/v1/market/snapshot | python3 -c "
import sys, json
d = json.load(sys.stdin)
spread = d.get('spread', 999)
print(f'Spread: {spread}')
if spread > 3.0:
    print('WARNING: Spread exceeds 3.0 pips')
else:
    print('OK: Spread within limits')
"
```

### Step 7: Signal Generation

```bash
# Check signal generation
curl -s http://127.0.0.1:13081/api/v1/signals | python3 -c "
import sys, json
d = json.load(sys.stdin)
sigs = d.get('signals', [])
directional = [s for s in sigs if s.get('Direction') != 'NO-TRADE']
print(f'Total signals: {len(sigs)}')
print(f'Directional: {len(directional)}')
for s in directional[:3]:
    print(f'  {s.get(\"Direction\")} {s.get(\"StrategyID\")} entry={s.get(\"EntryPrice\")} SL={s.get(\"StopLoss\")}')
"
```

### Step 8: Signal Delivery to EA

```bash
# Check that signals are being delivered to the Windows Agent
journalctl -u predictatrade-realtime -n 50 --no-pager -o cat | grep -i "Signal broadcast"
# Expected: log entries showing signal broadcast to Windows Agents
```

### Step 9: EA Signal Reception

```
// On the MT4/MT5 terminal, check the EA chart panel:
// - "Signals Received" counter should increment
// - "Signals Displayed" should show recent signals
// - Check PAT_signals.txt for received signal data
```

### Step 10: DB Persistence

```bash
# Verify signals are persisted in the database
PGPASSWORD=pat_local_dev_only psql -h 127.0.0.1 -U pat_admin -d predictatrade -c "
SELECT id, strategy_id, direction, entry_price, stop_loss, created_at
FROM trading.signals
ORDER BY created_at DESC LIMIT 5;
"
```

### Step 11: Health Check

```bash
curl -s http://127.0.0.1:13081/health | python3 -m json.tool
# Expected: status=ok, agents>0
```

### Step 12: News Breakout (if enabled)

```bash
# Check news breakout status (if NEWS_BREAKOUT_ENABLED=true)
curl -s http://127.0.0.1:13081/api/v1/admin/regime-diagnostics | python3 -m json.tool
# Check OCO groups
# This endpoint would show active breakout plans and OCO groups
```

### Step 13: Restart Recovery

```bash
# 1. Note current signals
curl -s http://127.0.0.1:13081/api/v1/signals | python3 -c "import sys,json; print(len(json.load(sys.stdin)['signals']))"

# 2. Restart the Go engine
systemctl restart predictatrade-realtime

# 3. Wait for startup
sleep 10

# 4. Verify signals are restored from DB
curl -s http://127.0.0.1:13081/api/v1/signals | python3 -c "import sys,json; print(len(json.load(sys.stdin)['signals']))"

# 5. Verify agent reconnects
curl -s http://127.0.0.1:13081/health | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'agents={d[\"agents\"]}')"
```

---

## Validation Checklist

| Check | Method | Pass Criteria |
|-------|--------|---------------|
| Terminal connection | API agents/status | agents > 0 |
| Broker connection | market/snapshot | source_type = LIVE_MASTER_NODE |
| Account ID | agents/status | login, balance present |
| Symbol mapping | market/snapshot | XAUUSD, digits=2 |
| Tick flow | 30s monitoring | bid/ask change |
| Spread | market/snapshot | spread < 3.0 |
| Signal generation | signals endpoint | directional signals > 0 |
| Signal delivery | journalctl | broadcast log entries |
| EA reception | chart panel | signals received counter > 0 |
| DB persistence | psql | signals in trading.signals |
| Health | /health | status=ok |
| Restart recovery | restart + verify | signals restored, agent reconnects |

---

## Post-Validation

After successful runtime validation, record:
- Signal IDs tested
- Broker name and account
- Terminal version
- Order IDs (if test orders were placed with separate authorization)
- Fill timestamps
- Slippage measurements
- DB record verification

**Do NOT mark runtime validation as complete without evidence from a live terminal.**
