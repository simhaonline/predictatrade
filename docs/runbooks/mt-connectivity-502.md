# Runbook: MT Client Connectivity & 502 Prevention

> Added 2026-09-03 (commits 8d02a2c, ac005cd). Covers the MT-client
> "edge-poll failed: HTTP 502" incident, the HA fix, and the connectivity
> watchdog. Read this before touching nginx, pat-control, or the watchdog.

## Permanent delivery guardrails (2026-09-04, this repo)

The silent-drop incident exposed the core gap: **nothing verified that an
ACKed item was actually dispatched**, so a contract break looked like
success. Three permanent layers now exist (monitoring module, both control
replicas):

1. **Dispatch contract test** — `control/src/modules/monitoring/delivery-contract.spec.ts`.
   Pins the contract: every enqueued signal payload MUST carry
   `"type":"SIGNAL"`; SERVER_COMMAND/LICENSE_STATUS/CANARY keep their own
   types; HandleSignal fields survive injection. CI fails = delivery to old
   client builds would silently break — DO NOT deploy.
2. **Delivery reconciliation watchdog**
   (`delivery-reconciliation.service.ts`, migration 127) — runs every 60s:
   - CRITICAL `DELIVERY:empty-acks:<device>` when a device ACKs >3 signal
     items in 24h with an empty dispatch type (the exact incident
     signature; verified it flags 35ef87d0=52 / 4e3e8b15=112 for 09-03).
   - CRITICAL/WARNING `DELIVERY:backlog:<device>` when PENDING items age
     >5/20 min (device stopped polling).
   - Snapshot: `GET /api/v1/monitoring/delivery` (admin) — per-device
     delivered/dropped 24h counts, backlog age, last signal ack type.
   - State table: `system.delivery_reconciliation` (migration 127).
3. **Delivery canary** (`delivery-canary.service.ts`) — every 10 min
   enqueues one non-trade `type:"CANARY"` `test:true` item (no StrategyID
   → skipped by the entitlement clause) per active device and requires an
   ACK within the TTL window; CRITICAL `DELIVERY:canary-stale` otherwise.
   Canary rows self-clean. First live cycle 2026-09-03 17:02:55 UTC: 3/3
   devices ACKed in <2s — including the two MT5 builds that had dropped
   signals (they dispatch `type:CANARY` fine; only type-less SIGNAL
   payloads fell over, which is exactly what the engine fix addresses).

Alerts flow to ntfy topic `pat-connectivity` (CRITICAL = high priority) and
appear in the admin monitoring snapshot.

## Addendum (2026-09-03, 5730430): "Unknown queue item type:" + "ingest TICK failed: HTTP 1003"

Two client-log lines, two different causes:

1. **"Unknown queue item type: " (trailing blank)** — queued signal payloads
   carried no `"type"` key. Old MT5 client builds dispatch ONLY on
   `payload->>"type"`: real signals fell to UNKNOWN, were ACKed
   `PROCESSED` with `type:""` but never executed — silent total drop.
   Evidence: 126 empty-type ACKs (devices `35ef87d0`/`4e3e8b15`) vs 74
   correct `SIGNAL` ACKs (MT4 `3e27f366`, which promotes empty→SIGNAL).
   **Fix (server-side, no EA recompile needed):** `enqueueSignalForDevices`
   injects `"type":"SIGNAL"` into every queued signal payload
   (realtime v1.24.1, deployed 16:42 UTC). Wire-tested: typed payload →
   ack `{"type":"SIGNAL","status":"PROCESSED"}`. MT5 v1.26.1/MT4 v1.27.1
   also promote empty msgType via the `ID` key for future recompiles.
2. **"ingest TICK failed: HTTP 1003"** — NOT a server status. nginx logged
   13,325/13,325 ingest POSTs = 200 in that window; zero 1003 server-side.
   The status came from a client-side middlebox (local HTTP proxy /
   antivirus TLS interception on the MT machine; 1003 is the classic
   Cloudflare-style "direct access denied" body). Transient, self-healed —
   all three terminals POSTing 200 within minutes. If it recurs on one
   client only, check that machine's proxy/AV, not the server.

## Incident summary (2026-09-03)

MT clients (MT4/MT5 EAs) intermittently logged:

```
[Predict-A-Trade] edge-poll failed: HTTP 502 — <html>...502 Bad Gateway...nginx
```

Root cause chain:

1. **Single pat-control instance.** Every control restart (deploy, rebuild)
   left nginx with no reachable upstream for a few seconds → 502 to any EA
   polling in that window. The 502 clusters in nginx logs mapped 1:1 to
   deploy restarts.
2. **nginx cached container IPs at startup.** After a backend container was
   *recreated* (new IP), nginx kept proxying to the dead IP until it was
   restarted itself, producing 502s outside deploy windows too.

## HA fix (shipped)

### docker-compose.yml

- `pat-control-b`: second full replica of the control service. Both serve
  edge-poll; Docker DNS `control` resolves to both.
- Static host port bindings removed from control replicas (nginx reaches
  them over the docker network, so no port collision).

> The async backtest worker is replica-safe: `backtest_jobs` claims use
> `FOR UPDATE SKIP LOCKED`, so each replica claims distinct jobs.

### nginx (`nginx/sites-available/api.predictatrade.com.conf`)

- `resolver 127.0.0.11 valid=10s ipv6=off;` — Docker embedded DNS
  re-resolution every 10s (fixes stale container IPs after rebuilds).
- `proxy_next_upstream error timeout http_502 http_503 http_504;` in
  `nginx/snippets/proxy-common.conf` — retry inside the resolved IP set.
- `error_page 502 503 504 = @control_b;` on `/api/v1/`,
  `/api/v1/backtest/`, and `/api/v1/health` — the intercept fallback
  re-dispatches the ORIGINAL request to the control-b peer (named location
  `@control_b`, 330s read timeout so a backtest failover can complete).

**nginx conf is BIND-MOUNTED from the repo into xauusd-nginx-1** — edit the
repo file, then `docker exec xauusd-nginx-1 nginx -t && docker exec
xauusd-nginx-1 nginx -s reload`. The host's /etc/nginx is stale — never edit
it.

### Verification (do this after any nginx/control change)

Hammer through real HTTPS while restarting the primary:

```bash
# in background: for i in $(seq 1 150); do curl -sk -o /dev/null \
#   -w "%{http_code}\n" --max-time 6 \
#   "https://api.predictatrade.com/api/v1/health"; sleep 0.1; done
docker restart pat-control   # mid-hammer
# expect: 100% 200s (proven 150/150 on 2026-09-03)
```

## Connectivity watchdog

**Guarantee:** if an MT client stops polling or the signal feed stalls, the
user AND admin are notified — clients never silently lose signals.

- Migration `database/migrations/126_connectivity_alerts.sql`:
  `system.connectivity_alerts` (dedup-keyed OPEN/RESOLVED, occurrences,
  notified_at).
- `control/src/modules/monitoring/connectivity-watchdog.service.ts` — runs
  **in both control replicas**, 60s cycles:
  - Realtime engine probe (`http://realtime:13081/health`, 5s timeout) →
    CRITICAL `ENGINE:realtime-down`.
  - Tick-feed staleness via **`gateway_receipt_time`** in `market.ticks`
    (3 min → CRITICAL `ENGINE:tick-feed-stale`). NOT the `time` column:
    the master terminal's own clock can lag many minutes while transport
    stays fresh (observed live: 15+ min source lag, 0.2s receipt lag).
  - Master clock drift: source `time` >5 min behind → WARNING
    `ENGINE:master-clock-drift` (ask the user to sync the terminal PC clock;
    signals are unaffected).
  - Device freshness: device bound to an ACTIVE/TRIALING license with no
    edge-poll for >3 min → WARNING `DEVICE:<id>`; auto-resolves when
    polling resumes.
- Notifications: ntfy POST to topic `pat-connectivity` (priority high for
  CRITICAL), max one push per alert key per 10 min. Dedup + resolve state
  in the alerts table. ntfy publish works (200); ntfy GET/read is
  ACL-restricted (403) — don't "verify" by reading back.
- `ws_clients == 0` in /health is **NORMAL** in Option B — EAs poll/ingest
  over HTTP and never connect to the WS hub (dashboard live-stream only).
  Do not re-add a "no agents connected" alert from that field; it is a
  false positive.
- Admin API: `GET /api/v1/monitoring/connectivity` (JwtAuthGuard +
  AdminGuard) — open alerts, 24h history, per-device seconds-since-poll.
- UI: `ConnectivityCard` on Admin → Backtesting (live status dot, open
  alerts, per-device last-poll freshness, 30s auto-refresh).

### Subscribing to alerts

Users/admins subscribe their phone/dashboard to the ntfy topic
`pat-connectivity` (self-hosted ntfy at pat-ntfy). Critical alerts arrive
with high priority.

## Quick triage — "why no signals on MT client?"

1. `docker ps` — all four of pat-control, pat-control-b, pat-realtime,
   xauusd-nginx-1 healthy?
2. Admin → Backtesting → Connectivity card (or
   `GET /api/v1/monitoring/connectivity`): open alerts + device freshness.
3. `SELECT alert_key, severity, status, occurrences FROM
   system.connectivity_alerts ORDER BY last_seen_at DESC LIMIT 10;`
4. Tick feed: `SELECT now() - max(gateway_receipt_time) FROM market.ticks;`
   — should be seconds, not minutes.
5. nginx error log:
   `docker exec xauusd-nginx-1 sh -c 'grep "connect() failed"
   /var/log/nginx/error.log | tail'` — a stream of these during steady
   state (not deploys) means a replica is actually down.
6. EA side: edge-poll errors in Experts log; token self-heal should recover
   401s (see pat-ea-auth-diagnostics skill).

## Realtime ingest 502s (2026-09-03, 5a78258)

`ingest TICK failed: HTTP 502` on the Master/data EA maps 1:1 to
pat-realtime deploy windows (e.g. 19:44:57 journal time = 15:44 UTC
engine rebuild; tick flow 318/min → 2 for ~40s → recovered by 15:47).

Realtime is a **stateful singleton** — a second engine instance would
split in-memory candle/strategy state, so there is no failover peer.
Mitigation (5a78258): `/ingest/agent` retries ONCE via
`error_page 502 503 504 = @realtime_retry` (fresh DNS resolve +
reconnect). Ticks are per-tick idempotent; the EA's next POST succeeds
once the container is back.

**Ops rule: schedule pat-realtime rebuilds in low-liquidity windows.**
Control-plane deploys are safe anytime (dual-replica failover proven).

## Related

- `docs/runbooks/edge-poll-429.md` — rate-limit side of edge-poll.
- EA diagnostics skill: pat-ea-auth-diagnostics (401 loops / License NOT SET).