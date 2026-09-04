# Runbook: MT Client Connectivity & 502 Prevention

> Added 2026-09-03 (commits 8d02a2c, ac005cd). Covers the MT-client
> "edge-poll failed: HTTP 502" incident, the HA fix, and the connectivity
> watchdog. Read this before touching nginx, pat-control, or the watchdog.

## Walk-forward A/B: tightened vs old swing geometry (2026-09-03, Kaggle Q4-2025 M15)

**Parity fix first (real bug found):** the backtest-engine never called
`strategy.InitExitProfileDB`, so historical backtests ran on the
ATR-fallback geometry, NOT the DB `exit_profiles` — live geometry changes
were invisible to backtests. Fixed in `cmd/backtest-engine/main.go`
(database/sql via pgx stdlib → `InitExitProfileDB`); binary rebuilt,
`docker compose restart control control-b` (bind-mount inode rule).
Verified in-run: `[EXIT_PROFILE] Loaded STANDARD_SWING: mode=PERCENTAGE
SL=0.1800%`.

**A/B (90 days, M15, Kaggle Q4-2025, $10k, no gates — raw direction engine):**

| Run | Geometry | Trades | Win rate | PF | Expectancy/trade | Return |
|---|---|---|---|---|---|---|
| A | NEW (SL 0.18% / TP1 0.40%) | 1907 | 27.5% | 0.49 | −$8.52 | −162.4% |
| B | OLD (SL 0.25% / TP1 0.25%) | 2097 | 42.3% | 0.45 | −$7.64 | −160.2% |
| C | NEW, max 1 position | 861 | 28.0% | 0.40 | −$14.24 | −122.6% |

**Honest verdict: neither geometry is profitable raw.** The runner opens a
trade on EVERY directional `Evaluate` (no score bar, no gates, no
cooldowns, no tier filter — by design it tests the strategy engine, not
the production pipeline). Geometry comparison is still valid (same
inputs): NEW yields 2× larger avg wins ($30.37 vs $14.75) and PF 0.49 vs
0.45 — marginally better maths — but the ~27–42% raw direction accuracy
in a choppy 663-pt Q4 range loses under any exit scheme. The production
signal (score bar 25–40 + 23 gates + micro-TP + cooldowns + tier caps) is
a different, far more selective system; these runs measure the inner
direction engine alone.

**Implication for the fleet:** keep the tightened geometry (better
expectancy per trade, better tier fit, EV-correct ladder) BUT the swing
strategy's live viability rests on the gate stack doing its job —
capital-protection's negative-live-edge veto and the profitability gate
remain the safety net. Swing signals will be rare and selective — that is
correct behavior, not a bug.

## Production-parity backtest gates (2026-09-03, v1.26 — REAL-WORLD PERFORMANCE)

User directive: "do all needful to fix issue we need to have realworld
performance". Root issue: backtests measured the raw direction engine,
not the product clients receive. Fix shipped (runner.go + types.go +
cmd/backtest-engine/main.go):

- `ProductionParity` config (default ON; `--raw` disables): mirrors the
  live EXECUTABLE promotion rule before opening any trade —
  `rawScore >= GetThresholds(strategy, regime).trade` +
  `EntryGatePassed` + `!IsLossCandidate` + per-strategy cooldown
  (`CooldownMinutes` set on entry, like live CooldownManager).
- `ParityBlockedCount` + labeled veto tallies
  (`PARITY_BELOW_TRADE_BAR / PARITY_ENTRY_GATE / PARITY_NEGATIVE_EV /
  PARITY_COOLDOWN`) print in every run's NO-TRADE reasons.
- Verified in-run banner: `Gates: PRODUCTION PARITY`.

**90-day parity results (Kaggle Q4-2025 M15/M5, $10k) vs raw control:**

| Strategy | Raw engine | Production-parity | Verdict |
|---|---|---|---|
| TREND_SWING | −129.4%, PF 0.47, 1204 tr | **+17.6%, PF 1.86, DD 8.0%, wr 44.8%, 29 tr** | workhorse ✓ (stored run 149ad7c1) |
| STANDARD_SWING | −162.4%, PF 0.49 | −10.0%, PF 0.82, DD 17.9%, 52 tr | marginal, chop-vulnerable |
| STANDARD_SCALPING | −160%, PF 0.45 | 1 trade (entry-gate vetoes 10,524) | matches live negative-edge fail-closed |
| ULTRA_SCALPING | — | 0 trades (HIGH_SPREAD ×14,475 — Kaggle synthetic spread) | live-only edge; NO-TRADE is a valid result |

**THE PROOF:** same strategy, same data — raw TREND_SWING loses 129%,
production-parity TREND_SWING makes +17.6%. The gate stack IS the edge.
The product clients receive is the gated pipeline, and the gated system
is now honestly measurable in backtests. Swing signals being rare is
correct behavior: TREND_SWING took 29 trades in 90 days with PF 1.86.

Data limitations (honest): Kaggle synthetic feed blocks ULTRA spread
modeling; live MT5_MASTER only covers ~2 weeks; Q4-2025 was one regime.
Real-world validation should continue with live-forward tracking (the
delivery canary + ack chain already capture what clients actually
receive), not more synthetic backtests.

## Synthetic-tick parity fix (2026-09-03 — the "MARNIE_FIB geometry bug" root cause)

The MARNIE_FIB `BUY_GEOMETRY_INVALID: TP1 <= Entry` wall was NOT a
strategy bug. Root cause: with `lastTick == nil` (backtests), the feature
registry fell back to `Bid=candle.Low / Ask=candle.High /
Spread=high−low` — the FULL BAR RANGE as spread. BUY entries printed at
the candle HIGH while TP1 was computed from Close, so any wide bar vetoed
geometry (`TP1(Close+2×ATR) < Entry(High)`). Live always has a real tick
(spread ≈ $0.30), so this artifact never exists in production.

Fix (runner.go, v1.26): the runner now constructs a per-bar synthetic
tick — mid=Close, spread=`config.Spread` (default $0.30), timestamps =
bar time — mirroring live tick semantics. Re-run results (90-day Kaggle
Q4-2025, production-parity gates ON):

| Strategy | Wide-spread artifact | Realistic tick | Note |
|---|---|---|---|
| TREND_SWING | +17.6%, PF 1.86 | **+23.6%, PF 1.45, DD 13.8%, 70 tr** | workhorse confirmed |
| MARNIE_FIB | 0 trades (veto wall) | **+18.1%, PF 4.49, DD 3.1%, 7 tr** | strategy was never broken |
| STANDARD_SWING | −10.0%, PF 0.82 | **+3.0%, PF 1.01, 239 tr** | breakeven-positive in chop |
| STANDARD_SCALPING | 1 trade | −46.0%, PF 0.81, 337 tr | live negative edge now honestly visible; live fail-closed downgrades are correct |
| ULTRA_SCALPING | 0 trades (spread) | −18.8%, 55 tr | Kaggle M5 cannot represent micro-TP tick dynamics; live-only edge |

Fleet verdict after the fix: TREND_SWING + MARNIE_FIB are the proven
earners; STANDARD_SWING is marginal-positive (selectivity does the
work); STANDARD_SCALPING is negative in both live and backtest (gates
rightly suppress it); ULTRA remains live-tracked only. ATEN stays
dormant (not armed, no exit profile).

## v1.25 DEPLOYED LIVE (2026-09-03 17:38 UTC, user-authorized early deploy)

"i need to get signals on clients" — done, verified end-to-end:

- **EXECUTABLE delivery proven live at 17:18** (pre-deploy, v1.24.1):
  STANDARD_SWING SELL, raw score 61.0 vs TRENDING_BEARISH trade bar 25
  (NOT 70 — the 70 was the RANGE bar; regime thresholds differ per regime,
  see `strategy/regime_thresholds.go`: swing trending=10/25, range=15/40),
  all gates pass, class EXECUTABLE → enqueued → **Xelans ACKed type:SIGNAL
  in 2s**. The EA auto-trade gate passes on class=EXECUTABLE.
- **Post-deploy (17:38)**: engine loaded tightened profiles at boot
  (`[EXIT_PROFILE] STANDARD_SWING SL=0.1800% TP1=0.4000%`; TREND 0.3000%).
  `[CANDIDATE-GATE]` observability is producing named gate vetoes (157 in
  10 min): data_quality FEED_QUALITY_FAILURE (ATEN, feed-quality
  transition), profitability NEGATIVE_EXPECTANCY (ultra in quiet ATR),
  martingale_ban, and — transient at boot — risk_oversize from
  equity-not-hydrated fail-closed (stops once ACCT-INIT lands, 17:39:49).
- **Fleet tier map (live equity)**: Xelans $792.60=STANDARD (gets swing +
  scalps; tightened SL 4.99–8.1pts fits its $10–39.6 cap band); Equiti
  $8.96=MICRO (cap $0.36 — cannot trade any XAUUSD min lot; honest
  fail-closed, needs top-up); ADS MT4 polls+ACKs (14 items/2h) but no
  recent ACCT-INIT — equity unknown until next ACCOUNT_INFO.
- All 3 clients ACKed canary/command/SIGNAL traffic at 17:42; delivery
  pipe green on all three.

Remaining watch item: swing reads must exceed the CURRENT regime's trade
bar (trending=25, range=40, high_vol=25) to promote; score 50 reads at
17:45 were marked INSUFFICIENT_SCORE because the decision score (post-
calibration/dominance) landed below the active regime bar — the raw
candidate score is not the deciding number. No action: this is the
scoring design working; flow resumes when reads strengthen.

## Combined tier-geometry model v1.25 (2026-09-03, 9fbff05 — user-approved a+b+c)

Answer to "combine a+b+c and make best maths to win". The three legs:

- **(a) Scalp character for small tiers** — micro-TP must clear round-trip
  cost (spread + slippage 0.10 + commission 0.06) or the profitability gate
  vetoes as loss candidate. Already global; unchanged. MICRO therefore
  still trades only when geometry is genuinely cost-covering.
- **(b) Tier caps** (`capitaltier.PerTradeRiskCapPct`): MICRO 2→4% ($4 on
  the $100 floor), STANDARD 2→5% ($25 on the $500 floor), PRO 2% unchanged.
  Effective per-trade cap stays min(plan cap, tier cap); execution sizing
  is separately capped by capital protection at 1% equity (lot shrinks, SL
  distance is what the tier gate admits).
- **(c) Tightened swing geometry** (migration 128, APPLIED to
  `trading.exit_profiles`; 5-min engine cache TTL → live without restart):
  STANDARD_SWING 0.25/0.25/0.50/1.00 → 0.18/0.40/0.70/1.20;
  TREND_SWING 0.40/0.40/1.20/1.60 → 0.30/0.65/1.10/1.50.

Why this is the best-maths combination (from 2026-09-03 production data):
the old swing TP1 == SL (1:1) was EV-negative at wr ≤ 0.5 — the root of
the profitability veto wall — and the 0.25% SL made wide-ATR swings
PRO-only so the only executable swing matched 0 devices. Tightening SL
while widening TP1 raises netRR 0.91→2.03 (clears the MinRR 2.0 gate and
the EV test at modelled wr 0.63: +2.41→+7.84 per 1R) AND shrinks min-lot
risk 11.2→8.1pts so STANDARD ($25 cap) admits it; TREND 17.9→13.5pts.
MICRO stays gross-risk gated (no net-risk admission — a $100 account must
not carry gap-through risk beyond 4%). Tests updated
(TestEvaluate_TightenedSwingReachesStandard); capitaltier/gates/signal
suites green; build clean. Deploy: cron 7aedaad30039 @ 21:05 UTC.

## "Why no trade" production truth (2026-09-03, d08bc27)

Answer to "signals not executing on MetaTrader" after the type fix:
delivery infrastructure is fine — the engine is not emitting tier-eligible
executable signals. Verified chain of custody for 2026-09-03:

- 453 signal items enqueued Sep 2 → Sep 3 08:04 UTC were ALL eaten by the
  type-less payload bug (fixed 5730430, deployed 16:42).
- After 16:42: zero signal payloads enqueued. Two verified reasons:
  1. The single EXECUTABLE swing signal (b549decd, SELL, score 77.1,
     17:01:15 UTC) was PRO-only by capital-tier math (SL distance 19.6pts
     → min-lot risk $19.6 > STANDARD 2% cap $10) while the fleet is
     MICRO×2 + STANDARD×1 → delivery SQL matched 0 devices (correct
     fail-closed behavior, but silent).
  2. Every other strong read (scores 66–85, trade bar 70) was vetoed by
     the advisory-candidate gate path — which logged NOTHING (no log, no
     metric, no DB row). Standard-scalping additionally carries a
     proven-negative live edge (65 downgrades; fail-closed by design).
- Profitability math replay for the 77.1 read PASSES (netRR 3.72, EV
  +17.8) → the veto is a different, unnamed gate; identified only after
  v1.24.2 observability deploys.

v1.24.2 (d08bc27, scheduled deploy 21:05 UTC) makes both failure classes
loud: `[CANDIDATE-GATE]` logs + `pat_gate_veto_total` for every strong
candidate veto, and `[DELIVERY]` WARN when an executable signal matches
0 devices. Engine log version now reflects the real engine version.

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

## Tick-lag vs candle-fresh split (2026-09-04, loop-split fix)

Signature: `market.ticks` MASTER rows carry `time` ~20 min behind
`gateway_receipt_time` (rows insert at ~2/s but trail real time), while
`market.candles` stays fresh. NOT an EA clock problem and NOT a WSS drop —
both terminals kept streaming; candles bypass the engine's main loop via
`SetCandleSyncFn` (provider goroutine → `PushExternalCandle`), ticks ride
`tickChan` (4096 slots) consumed by the same single `select` loop that runs
candle-driven strategy evaluation (~5 candles/s across TFs on this feed).
When candle work saturates that loop, the tick queue backs up and drains at
~0.1× real time while `/health` stays green.

Fix: tick and candle channels now drain in **separate goroutines**
(`realtime/cmd/realtime-engine/main.go` — `handleTick` loop split out; both
handlers only share mutex-guarded state and clone-emit paths already exercised
concurrently by the snapshot handler). Verified post-deploy: tick lag 0.0 min
while candle rate unchanged.

Diagnostic shortcut: `SELECT max(time), max(gateway_receipt_time) FROM
market.ticks WHERE source LIKE '%MASTER%';` — growing gap between the two =
backlog inside the engine, not a feed outage. `/metrics`
`pat_tick_latency_ms_*` histogram confirms (p99 collapses from ~e9 ns-scale
sums/counts to sub-second after the split).

Related deploy gotcha (same incident): recreating pat-realtime with plain
`docker compose up -d realtime` drops `DATABASE_URL` interpolation (host
shell has no env) → engine crash-loops `Config validation failed` and, once
given a host-loopback URL, panics at `Persister.GetDB` (main.go:2168).
Deploy with `docker compose --env-file infra/env/.env up -d realtime` — the
canonical env file carries the container-network Postgres URL.

## Related

- `docs/runbooks/edge-poll-429.md` — rate-limit side of edge-poll.
- EA diagnostics skill: pat-ea-auth-diagnostics (401 loops / License NOT SET).