# 14 — Trade Management Audit

## Architecture reality

Go engine is a **signal emitter only** — it places no broker orders. Execution path: signal → AgentHub broadcast → Windows Agent → named pipe → MQL EA. Therefore §23 lifecycle (order prep→fill→partial→breakeven→trail→TP1-3→reconcile) lives in `windows-agent/` + `mql/*` + `trading.*` tables.

## Verified server-side pieces

- `execution_commands` / `execution_events` / `positions` / `trade_results` / `sl_modification_history` / `hedge_*` tables exist (schema complete).
- EA loopback messages CLOSE_ACK, SLIPPAGE_EVENT, CAPITAL_WARNING, CAPITAL_PROTECTION arrive at Go and are **logged and dropped** (`agent_provider.go:519-533`) — no ACK ingestion, no fill capture, no reconciliation (`reconciler` memory-only, ack methods dead).
- `signal_deliveries`=0 rows ⇒ server never records that any device received/executed anything.

## Windows agent + MQL (static review)

- Agent: connect/backoff, fingerprint, signed-update manifest support present; **it authenticates nothing to the engine** (none required by server) — device binding enforced only toward control plane.
- MQL EAs implement order send with deviation, SL/TP placement, capital-protection percent checks client-side. Broker spec adaptation (stops level/freeze/tick value) present in code but UNVERIFIED against a real terminal — EXTERNAL BLOCKER (no live MT4/MT5 terminal available to this audit).
- Idempotency of execution commands: command GUIDs exist in schema; EA-side dedup logic present in code but cannot be proven without terminal replay — EXTERNAL BLOCKER.

## Verdict

TRADE MANAGEMENT = PARTIAL / NOT VERIFIED.
Blocking gaps before live execution: delivery/ack ledger unwired (12), reconciliation stub, no server-side position/fill truth, duplicate-execution protection unproven end-to-end, emergency stop absent (13). Per §100 ("duplicate trade execution", "unsafe live-order behavior") these are NO-GO conditions for broker execution regardless of signal quality.
