# Server-Side SL Enforcement — Architecture & Flow

**Version:** v1.15.0 (25 August 2026)  
**Status:** ACTIVE — backend is the enforcement authority for S/L and TP

## Overview

The Predict-A-Trade backend is not just the calculation authority for stop-loss and take-profit levels — it is also the **enforcement authority**. The server can detect and respond to SL violations even if the MQL EA code is modified by the user.

## Signal Delivery Flow (no blocking)

```
Strategy Engine → broadcastSignalToAll()
    │
    ├── wsHub.BroadcastSignal()          → Frontend dashboard (always)
    │
    └── agentHub.BroadcastSignalToAgents()
        └── for each connected agent in h.agents:
            └── agent.send <- signal     → Windows Agent → EA
```

**Signal delivery is never blocked by SL enforcement.** Suspended agents are removed from `h.agents` by `DisconnectAgent()`, so `BroadcastSignalToAgents()` automatically skips them.

## SL Enforcement Flow

```
EA places trade → sends EXECUTION_ACK via IPC → Windows Agent → Server
    │
    ├── SL > 0 and matches server value? → ✅ Verified, log success
    │
    ├── SL = 0? → ❌ CRITICAL VIOLATION
    │   ├── Send CLOSE_POSITION to agent → EA closes position
    │   ├── Record violation (count++)
    │   ├── Log to audit.client_events
    │   └── If count >= 3 → DisconnectAgent()
    │
    └── SL mismatch (>0.5 point tolerance)? → ⚠️ WARNING
        ├── Record violation
        └── Log to audit.client_events
```

## Position SL Monitoring

```
Broker snapshot received (every ~1s from MT5 agent)
    │
    └── checkPositionSLs(positions)
        └── for each PAT position (magic 100000-199999):
            ├── SL > 0? → ✅ OK
            └── SL = 0? → ❌ Send CLOSE_POSITION + record violation
```

## Server Commands to EA

| Command | Trigger | EA Action | Scope |
|---------|---------|-----------|-------|
| `CLOSE_POSITION` | SL violation on specific position | Close by ticket or magic | Single position |
| `EMERGENCY_STOP` | Capital protection / system emergency | Close ALL PAT positions + halt | All positions |
| `KILL_SWITCH` | Security incident / permanent ban | Close all + ExpertRemove() + disconnect | Agent + EA |

## Agent Suspension

| Violations | Action |
|------------|--------|
| 1-2 | Logged, signals continue |
| 3 | Agent disconnected, no future signals |
| Other agents | Unaffected, signals continue normally |

## MQL EA Versions

| Version | Features |
|---------|----------|
| v1.08 | TRADE_RESULT reporting, wrong-side SL rejection, watchdog, partial TP1/2/3 |
| v1.09 | + CLOSE_POSITION handler, EMERGENCY_STOP handler, KILL_SWITCH handler, position SL in snapshot |

## Windows Agent Versions

| Version | Features |
|---------|----------|
| v1.2.16 | Terminal auto-recovery, signal delivery |
| v1.2.17 | + TRADE_RESULT forwarding |
| v1.2.18 | + CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH command forwarding |
