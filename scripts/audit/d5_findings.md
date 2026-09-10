# AUDIT DOMAIN 5 — Database Schema · Deployment · Observability (2026-09-10)

Scope: /srv/predictatrade/xauusd. All findings verified against repo files AND live containers
(pat-postgres, pat-valkey, docker ps). Severity: CRITICAL/HIGH/MEDIUM/LOW. Class: SCHEMA/DEPLOY/OBS/SEC/RISK.
Scanners: scripts/audit/d5_scan_migrations.py, d5_scan_migrations2.py (created by this audit).

## (a) Migrations — 102 files on disk, 102 rows in audit.migration_history (verified live)

- OK — Ordering gaps: disk gaps are exactly 029→060, 100→110, 139→141; gaps 030–059 and 101–109
  documented intentional in database/migrations/MIGRATION_ORDER.md:42-49 ("never used", verified
  absent from history). Gap 139→141 is undocumented in that doc's gap list (139→141 not listed;
  only 030-059/101-109 are) — MEDIUM/DOC: add 139→141 to MIGRATION_ORDER.md gap section.
- OK — No duplicate table creations: 232 distinct qualified CREATE TABLE (IF NOT EXISTS) statements,
  zero same-name pairs. 3 unqualified tables in 080_devil_liquidity.sql (devil_liquidity_config/
  events/marks land in search_path=public; live catalog confirms public.devil_liquidity_events).
  MIGRATION_ORDER.md:6-20 documents the historical 018/089…095 renumber fix; check_migrations.sh guard exists.
- FINDING a1 — MEDIUM/SCHEMA — licensing.edge_signal_queue.device_id has NO FK to licensing.devices.id
  (database/migrations/117_edge_signal_queue.sql:13 declares "uuid NOT NULL" with comment only; live
  pg_constraint returns 0 FK rows). Same pattern: trading.signals.feature_snapshot_id UUID (no FK),
  audit.audit_events (only PK constraint), trading.trade_results (only PK + 1 unique).
- FINDING a2 — LOW/SCHEMA — Duplicate index name: idx_trade_results_account created identically in
  014:93 and 068:19 (harmless via IF NOT EXISTS, but 068 is a no-op re-create).
- Hot-path indexes (all verified in migrations + live pg_indexes):
  - trading.signals: idx_signals_strategy(005:535), idx_signals_created(005:537), composite
    (symbol,strategy_id,created_at DESC)(010:596), partial quality/unclosed/expectancy (020,090,095) — COVERED.
  - licensing.edge_signal_queue: idx_edge_signal_queue_device_pending(device_id,created_at) WHERE
    status IN('PENDING','IN_FLIGHT') (117:26-28) + (device_id,signal_id)(117:30) — covers the stated
    "device+status" hot path. Hot consumer query orders by created_at ASC (realtime main.go:343);
    the partial index matches. COVERED.
  - trading.trade_results: idx_trade_results_account(014:93) + (strategy_id,timeframe)(084:7) — COVERED.
  - audit.audit_events: idx_audit_entity(entity_type,entity_id)(005:1239), composite
    resource_time partial (010:608), timestamp DESC (005:1240) — COVERED.
- FINDING a3 — LOW/SCHEMA — trading.strategy_evaluations hypertable has NO compression policy
  (mig 010:120 created it; only 020:109 re-affirms hypertable). Live: chunks=43, compress=false.
  mig 010 did add compression for indicator_history (14d) but skipped strategy_evaluations.

## (b) TimescaleDB — 22 hypertables live (catalog verified)

- market.ticks: hypertable ✓, compressed ✓ (7d policy, 005:1011), retention 90d (005:1017), chunk 1h (005:1002), 370 chunks live.
- market.candles: hypertable ✓, compressed ✓ (30d, 005:1012), chunk 1d (005:1004), 5905 chunks live.
- FINDING b1 — MEDIUM/DEPLOY drift — candles retention: 081_market_candles_retention.sql:13 sets
  1095 days (3y, documented as v1.16.0 P2 closure) but LIVE policy job shows drop_after=180 days
  (hypertable_id 9). Someone changed it out-of-band after 081; MIGRATION_ORDER/history don't record it.
  Doc-vs-live drift on a data-retention guarantee.
- FINDING b2 — LOW/SCHEMA — trading.regime_history (010:192 hypertable) has NO compression AND NO
  retention policy live (compress=false, absent from policy jobs) — 49 chunks accumulating unbounded.
- FINDING b3 — LOW/SCHEMA — public.devil_liquidity_events (080:104) hypertable with NO compression,
  NO retention (compress=false, 3 chunks). Also created unqualified (public schema) unlike every other hypertable.
- FINDING b4 — INFO/SCHEMA — compliance.client_event_log hypertable exists (026:125) but has no
  compression/retention (026:160 leaves it as a commented-out TODO enable). chunks=0.
- Chunk sizing: 1h ticks / 1d candles is reasonable for this volume; audit tables 7d chunks (028).
- Continuous aggregates market.candles_m5_agg/h1_agg with refresh policies (010:721-729).

## (c) Compose — 18 services (not 20+), docker-compose.yml

- Restart: `restart: always` on all 17 run services ✓ (watchdog deliberately no depends_on — documented 465-472).
- Healthchecks: 9 of 17 have one (postgres, valkey, realtime, control, control-b, backtest, frontend,
  mail-relay, + live-terminal Dockerfile HEALTHCHECK). LACKING (docker-compose.yml lines): status(:261-279),
  nginx(:282-309), prometheus(:350-359), grafana(:361-378), ntfy(:381-392), nats(:400-409),
  backup-sync(:420-463), watchdog(:473-509), discord-bot(:514-529). MEDIUM/DEPLOY — watchdog supervises
  them via docker state only; a wedged-but-running Grafana/ntfy is never detected.
- FINDING c1 — HIGH/DEPLOY — ZERO resource limits: no deploy.resources / mem_limit / cpus anywhere in
  the file (whole 564-line file has no resources block). Postgres sorts/backtests can OOM the host
  (only mitigations: shm_size 2gb at :21). Also unbounded Prometheus TSDB (no retention flag, named volume).
- FINDING c2 — MEDIUM/DEPLOY — No per-service `logging:` blocks; container log rotation relies solely
  on host daemon.json (log-driver local, max-size 20m, max-file 5) — OK in practice but fragile
  (compose-level override on any service would silently disable it; nothing pins it in-repo).
- depends_on: correct service_healthy wiring for app tier; nginx depends_on omits control-b (:301-306)
  — LOW (nginx re-resolves via resolver 127.0.0.11 valid=10s so this is cosmetic).
- FINDING c3 — MEDIUM/DEPLOY — Unnecessarily exposed host ports (ss -tlnp verified on 0.0.0.0):
  realtime 13081:13081 (:117), frontend 13082:13082 (:248), status 13083:13083 (:273),
  live-terminal 13090:13090 (:77), nats 4222:4222 + 8222:8222 (:405-406). nginx fronts all app ports,
  so these bypass TLS/rate-limiting (only nginx 80/443 + localhost-bound 5432/6379/3001/8091/8088 are
  justified). NATS 4222 has no auth and is world-reachable — the worst of the set.
- Secrets: environment: carries ${POSTGRES_PASSWORD}, ${JWT_SECRET}, ${DATABASE_URL}, ${GF_SECURITY_ADMIN_PASSWORD}
  interpolated from infra/env/.env (gitignored, .gitignore:27, verified; env files 600 hermes-owned) —
  good pattern; no plaintext secrets committed in compose. mail-relay PAT_MAIL_USERS via env (:326) ✓.
- FINDING c4 — LOW/DEPLOY — valkey has NO auth (no --requirepass) and port 6379 is localhost-only, but
  any container on pat-net can reach it unauthenticated (acceptable for single-tenant bridge; note only).

## (d) nginx — docs/nginx/nginx.conf + nginx/snippets + sites-available

- TLS: TLSv1.2/1.3 (nginx.conf:47-48), per-site certs from /etc/letsencrypt, ssl_ciphers modern GCM
  (api:29), session cache 10m/1d, HSTS with preload (security-headers.conf:6), server_tokens off,
  http→https 301 everywhere. GOOD.
- Rate limiting: zones api 30r/s, auth 20r/s, ws 10r/s, general 60r/s + conn_per_ip (snippets/rate-limit.conf);
  applied per-site (api:59-60, auth:219). GOOD.
- WS: websocket-proxy.conf has Upgrade/Connection headers, proxy_buffering off, 120s read/send timeout
  (zombie protection) — but live.predictatrade.com /ws uses its own 300s block (:99-100) and
  platform /ws/v1/relay + /ws are 300s too. Inconsistent by design (EA long-poll vs browser).
- Buffers: proxy_buffer_size 128k, proxy_buffers 4 256k, busy 256k (snippets/proxy-common.conf).
- FINDING d1 — LOW/DEPLOY — proxy-common.conf sets proxy_read/send 30s globally; only backtest route
  gets 330s; any future long endpoint will 504 (documented pattern, acceptable).
- FINDING d2 — LOW/DEPLOY — resolver-based HA relies on nginx `set $upstream_control` + proxy_next_upstream
  tries 2; verified `getent hosts control` returns ONE IP (control-b is a separate name, not an alias in
  the same DNS pool — compose:163 aliases control→both? No: control gets alias 'control', control-b gets
  alias 'control-b'). The compose comment (:152-158) claims "Docker DNS returns both A records for
  `control`" — that is FALSE in this compose (aliases are distinct names; live DNS confirmed 1 IP for
  control). Failover still works via error_page @control_b fallback (api.conf:319-327,334-342), but the
  comment's load-balancing claim is inaccurate — LOW/DOC.
- FINDING d3 — LOW/DEPLOY — no client_body_timeout/proxy_buffers tuning beyond defaults in server blocks;
  no HTTP/2 (`listen 443 ssl` without http2) — minor perf gap.

## (e) Observability

- Prometheus: infra/prometheus/prometheus.yml scrapes realtime:13081/metrics (handler verified
  realtime/internal/gateway/http.go:146) and control:13080/metrics (control/src/common/metrics.ts wired
  in main.ts) + self. GOOD — matches actual /metrics endpoints. But NO scrapes for live-terminal,
  backtest, mail-relay, nginx (stub_status), postgres/cadvisor — coverage is 2 app targets.
- Alert rules: exactly 2 alerts in infra/prometheus/rules.yml (ServiceScrapeDown, SignalPipelineStalled;
  TODO(P2) at rules.yml:3 for pat_agents_connected). MINIMAL.
- Grafana: ONE provisioned dashboard (infra/grafana/dashboards/gate-health.json). Coverage thin.
- FINDING e1 — MEDIUM/DEPLOY — Prometheus/Grafana have NO TSDB retention/config flags (grafana/prometheus
  services run with image defaults; prometheus TSDB default retention 15d — unbounded-ish disk growth
  is bounded but never sized; named volume pat-prometheus).
- FINDING e2 — LOW/DEPLOY — Structured logging is inconsistent: Go realtime/watchdog/mail-relay use
  zerolog / log.Printf, NestJS uses Logger, discord-bot/bot.mjs:56 and status/server.js:403 use raw
  console.log with hand-rolled timestamps. No log shipper (Loki/EFK) — logs live only in container
  local driver + nginx-logs volume.
- ntfy infra/ntfy/server.yml: auth-file + auth-default-access write-only, 15 lines, NO cache-message/
  attachment settings (default cache OK).

## (f) Backups/PITR — verified against live host

- backup-sync sidecar (compose:420-463): WAL archive → S3, dumps → S3 (filtered backup_2*.dump+sha256+base/*),
  60s loop, Hetzner-safe single-stream config inline. ✓ matches pat-backup-dr skill.
- WAL archiving LIVE: postgres archive_mode=on (container), pg_stat_archiver archived_count=10830,
  failed_count=3, last_archived_wal current. ✓
- Physical base backups LIVE: /var/backups/predictatrade/basebackup.log shows base_20260910_090001 (43G),
  keep-7. PITR chain complete (base+WAL on bucket per skill's 2026-09-02 verification).
- FINDING f1 — HIGH/OBS — NO restore-test evidence: scripts/backup/restore_test.sh exists but
  /var/backups/predictatrade/restore_test.log DOES NOT EXIST (checked live). Docs (BACKUP_RESTORE.md:210)
  say "run monthly" — never run (or never logged). Untested backups are not backups.
- FINDING f2 — LOW/OBS — Root-cron backup cadence (6h) is host-level, not in-repo (no infra/cron or
  systemd timer file for backup.sh/physical_backup.sh in repo — infra/systemd has only app units, all
  disabled). Config-as-code gap: DR depends on undocumented host crontab.

## (g) CI/CD — .github/workflows/ci.yml (single workflow, 4 jobs)

- go job: vet + test + test -race + build + govulncheck(best-effort `|| echo` — NOT a hard gate, ci.yml:26).
- nestjs: npm ci + lint + test + build ✓. frontend ✓. python ✓. security job: secret scan HARD gate
  (exit 1) + npm audit --omit=dev HARD gate (documented residual SUP-1).
- FINDING g1 — LOW/CICD — govulncheck is best-effort only (ci.yml:25-26 `|| echo "unavailable"`), and
  there is NO migration/DB check in CI (scripts/check_migrations.sh is referenced by make/db workflow
  docs but not invoked in ci.yml). Lint/test gates are otherwise genuinely wired.

## (h) DISCORD_WEBHOOK_URL — wired end-to-end ✓

- compose:500 passes ${DISCORD_WEBHOOK_URL:-} into watchdog env; watchdog/main.go:108 reads it,
  :66 cfg field, :489-570 fan-out posts discordPayload JSON via postJSON with error log on failure.
  Optional-empty disable semantics documented at compose:497. VERIFIED CONSUMER EXISTS. (Not verifiable
  that a real webhook URL is set in gitignored .env — that's operator-side by design.)

## (i) Valkey — live CONFIG GET verified

- maxmemory = 0 (unlimited), maxmemory-policy = noeviction, persistence = RDB only (--save 60 1),
  appendonly = no.
- FINDING i1 — MEDIUM/DEPLOY — Unlimited memory + noeviction: an unbounded keyspace growth (candle
  cache keys, session keys) will OOM the container instead of evicting; with noeviction and OOM, writes
  fail and the engine degrades. Caches DO use TTLs (realtime/internal/cache/valkey_candles.go 60s/5m
  TTLs) which bounds app keys, but no cap exists for foreign/unexpected keys.
- FINDING i2 — LOW/DEPLOY — RDB save 60/1 only: up to 60s of cache loss on crash (acceptable for a
  cache; sessions/bootstrap-candles are refetchable). AOF off by design.

## (j) Single points of failure

- pat-realtime: SINGLE instance (compose:87-134), no HA peer, no LB (nginx @realtime_retry is same-
  upstream retry only). All ingest, signal generation, WS relay go through it. SPOF — restart takes the
  full pipeline down (healthy restart ~29 min uptime pattern shows frequent deploys).
- pat-postgres: SINGLE postgres, no streaming replica; PITR exists but RTO is restore-time. Known/accepted.
- HA control pair: control + control-b with nginx failover = GOOD design (edge-poll no-502 goal), BUT:
  FINDING j1 — MEDIUM/DEPLOY — the "load-balanced via DNS" claim (compose:152-158, api.conf:41-46) is
  not what's deployed (distinct DNS names, verified); actual behavior is primary-only + error-page
  fallback. Failover correctness: backtest route uses proxy_next_upstream + named fallback with 330s
  budget ✓; generic /api/v1 uses error_page 502/503/504 → @control_b ✓ (only on 502/503/504, not
  timeouts>15s mid-body). Acceptable but the doc/comment mis-states the mechanism.
- watchdog is itself a SPOF (no peer) but has docker.sock self-heal + restart:always; acceptable.
- ntfy/mail-relay single instances — alert path single-homed (mitigated by watchdog Discord/Telegram mirrors).

## (k) Unbounded memory / resources

- FINDING k1 — MEDIUM/DEPLOY — Same as c1: NO container has memory/CPU limits (compose file has zero
  resources blocks). Highest-risk unbounded consumers: postgres (sorts, backtest engine running INSIDE
  control containers against the live DB), prometheus TSDB (15d default but no size cap), valkey (maxmemory 0).
- Bounded: mail-relay spool deletes on delivery/dead-letter (main.go:477-480, 24h cap) ✓;
  edge_signal_queue hygiene exists (dedupe DELETE at realtime/cmd/realtime-engine/main.go:338-350,
  MarkExpired loop :2279) ✓; valkey app keys TTL'd ✓; watchdog check loop is 30s polling ✓;
  nginx buffers capped ✓. Prometheus WAL/TSDB default-chunk retention = 15d (not configured in-repo).

## (l) TODO/FIXME in infra files

- Only ONE: infra/prometheus/rules.yml:3 "TODO(P2): export pat_agents_connected gauge … zero agents
  connected alert". Infra is otherwise TODO-clean (nginx/, docker-compose.yml, scripts/backup/, infra/ntfy clean).

## Summary counts
CRITICAL: 0 · HIGH: 2 (c1 no-resource-limits [merged with k1]; f1 no restore-test evidence) ·
MEDIUM: 9 (a1 FK gaps on queue, a3/b2/b3 policy gaps, b1 retention drift, c3 port exposure incl. NATS,
e1 prometheus sizing, i1 valkey unbounded, j1 HA-doc mismatch, e2 logging fragmentation) · LOW: 10.