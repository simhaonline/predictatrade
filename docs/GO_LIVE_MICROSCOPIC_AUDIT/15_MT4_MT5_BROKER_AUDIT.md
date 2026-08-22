# 15 — MT4/MT5 Broker Connectivity Audit

## What is proven

- One agent connected to engine (`/health agents:1`; Valkey `pat:agent_status`) during weekend — terminal-side heartbeat path works mechanically.
- `windows-agent/` Go bridge: WS client with backoff, device fingerprint, IPC pipe to EA, signed-updater manifest support, deploy artifacts published via nginx `/downloads`.
- MQL4+MQL5 EAs exist incl. MasterNode variants; symbol config + capital protection implemented in-code.

## What is NOT provable in this environment (EXTERNAL BLOCKERS)

1. Live terminal identity/account/broker spec (digits, tick value, stops/freeze levels, fill policy) — requires an actual MT4/MT5 terminal session.
2. Reconnect/stale-terminal semantics end-to-end: server marks ExecutionPermit PASS for 60s after any heartbeat (`main.go:638-666`, extended 30s) — a stale terminal CANNOT appear ONLINE beyond that window **provided** the only heartbeat source is a genuine agent; but because the agent WS is unauthenticated (F-003), any attacker can keep a fake terminal "online" indefinitely.
3. Account switching / multi-account binding: control-plane max-MT-account check is TOCTOU-racy (`licensing.service.ts:199-212`).

## Findings

| ID | Sev | Finding |
|---|---|---|
| 15-1 | P0 | Agent channel unauthenticated ⇒ terminal identity claims (account/equity/margin feeding Margin+ExecutionPermit gates) are spoofable (cross-ref F-003). |
| 15-2 | P1 | Legacy licensing revoke path leaves session leases/credentials alive (`licensing.service.ts:274-283`) vs correct cascade in device-auth service — divergent duplicate implementations. |
| 15-3 | P2 | Device access tokens minted by activation are unverifiable/dead; real continuity rides refresh tokens + 45s leases. |
| 15-4 | INFO | Local artifact `windows-agent/C:\ProgramData\PredictATrade\device.key` (gitignored) proves agent was test-run on Linux creating a literal Windows-path dir — key handling portability nit. |

**Status: PARTIAL.** No live-order behavior tested (per §25 prohibition without authorization).
