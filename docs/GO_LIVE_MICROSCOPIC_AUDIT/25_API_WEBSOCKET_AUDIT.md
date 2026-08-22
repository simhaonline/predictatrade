# 25 — API / WebSocket Audit

## REST inventory (control `/api/v1`, NestJS)

auth(8) · users(4) · plans(2) · subscriptions(3) · billing(2 — webhook UNGUARDED) · referrals(3) · commissions(4) · payouts(5, broken writes) · licensing(7, 2 IDOR routes) · devices(6) · admin(13, AdminGuard class-wide) · audit(1) · health(1)+/metrics · operations(10, AdminGuard) · backtest(5) · guest(7, throttled).
Duplicates/orphans: admin-vs-domain duplicates for payouts/commissions/users/licenses; `/auth/me`≈`/users/me`; two device-revoke semantics; `CorrelationIdInterceptor` + pagination DTO dead; `openapi.json` stale (missing guest/backtest; no security schemes documented).

## REST inventory (Go engine `:13081`)

`/health` `/ready` `/metrics` `/debug/pprof/*` `/ws(/v1)` `/ws/v1/agent` `/api/v1/signals(+?limit)` `/signals/resume` `/market/state|snapshot|candles|price/history` `/strategies` `/agents/status` `/system-health` `/admin/regime-diagnostics`.
**Auth: NONE on any Go endpoint.** Mitigated only by intended-localhost bind — but compose publishes 13081 publicly. Rate limits: none on Go plane; NestJS throttler per-route counts present (login 10/min etc.).

## WebSocket

| Aspect | Frontend hub `/ws/v1` | Agent hub `/ws/v1/agent` |
|---|---|---|
| Auth | none (userID from query, defaults anonymous) | none; self-declared agentId; CheckOrigin always true |
| Envelope | event_id/stream_id/**sequence per-client monotonic**/schema_version 1.0.0/priority P0-P2 | **sequence omitted** → agents cannot detect gaps |
| Entitlement | filter exists (`isEntitled`) but Client.entitlements never set ⇒ UI can never receive SIGNAL events legally | unfiltered broadcast of all BUY/SELL/candidates |
| Backpressure | drop-on-full; P2 branch identical to default (unfinished priority dropping) | 100-conn cap only |
| Resume | HTTP `/signals/resume` keyed on DB sequence — works but unauthenticated (any deviceId leaks another device's undelivered signals) | n/a |
| Scale-out | in-process hubs, no Valkey pub/sub ⇒ multi-instance breaks sequencing/delivery | same |

Free-user premium leak via WS: currently impossible-by-accident (nobody gets signals via WS); via REST: trivially possible (reproduced). Both states are wrong.
