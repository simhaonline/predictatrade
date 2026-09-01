# Predict-A-Trade — Development Summary

> Snapshot as of 2026-09-01. Macroscopic view of architecture, current state, recent work, and open issues.

## 1. What the system is

Predict-A-Trade is a real-time XAUUSD (gold) trading intelligence + execution platform with a strict, multi-plane architecture (see `AGENTS.md`):

| Plane | Tech | Responsibility |
|-------|------|----------------|
| Real-Time Trading | **Go** (`realtime/`) | Market data ingest, features, strategy, signal generation, hard-risk gates, SL/TP enforcement, execution authorization, reconciliation. Fail-closed. |
| Intelligence/Research | **Python** (`research/`) | Datasets, backtesting, walk-forward/OOS, calibration, ML/NLP/vision research. Not on the live tick path. |
| SaaS/Control | **NestJS** (`control/`) | IAM/MFA/RBAC, tenants, subscriptions, billing, licensing, devices, MT accounts, referrals, commissions, payouts, audit. |
| Presentation | **Next.js** (`frontend/`) | Public site, user portal, admin console, XAUUSD Live Command Center. Renders server truth; never authoritative for risk/entitlement/finance. |
| Edge | **Windows Agent + MQL4/5** (`windows-agent/`, `mql/`) | Lightweight adapters. Agent runs as **Master Node** (data feed → `MARKET_SNAPSHOT`) or **Client Node** (signal execution). |

All services run in Docker via `docker compose` (`--env-file infra/env/.env`). No systemd.

## 2. Runtime topology (containers)

`pat-postgres` (TimescaleDB) · `pat-valkey` (cache) · `pat-realtime` (Go) · `pat-control` (NestJS) · `pat-frontend` (Next) · `pat-nginx` (reverse proxy/SSL) · `pat-prometheus` · `pat-grafana` · `pat-ntfy` (alerts) · `pat-nats` (optional ingest bus) · `pat-backtest` · `pat-mail-relay` · `pat-status` · `pat-live-terminal` · `pat-backup-sync`.

## 3. Strategy products (versioned, config-backed)

`STANDARD_SCALPING` · `ULTRA_SCALPING` · `STANDARD_SWING` · `TREND_SWING`. Raw score is **not** probability; subscriber-facing probability must be calibrated to a named target + exit profile.

## 4. Recent development focus (last commits)

- **Windows Agent hardening & shipping** (v1.2.44 → v1.2.53): CA code-signing pipeline, SmartScreen warm-up, stale `install.ps1` prevention, stray-process kill on install/update, MQL consolidation, and "dedupe windows-agent deploy tree; canonical role installers + per-arch only" (`3f288af3`).
- **Agent WS routing** moved to `api.predictatrade.com` (internal API) + installer fixes (`a1304228`).
- **Server-side SL/TP enforcement (v1.15.0)**: `EXECUTION_ACK` verification, position SL monitoring → `CLOSE_POSITION`, `EMERGENCY_STOP`, `KILL_SWITCH`, 3-strike agent suspension, signal delivery continues (suspension via disconnect, not broadcast filtering).
- **Market-feed health hardening**: data-feed outage detection independent of candle flow, ntfy alerts + `REQUEST_SNAPSHOT` nudge to agents (`startHealthMonitor`).
- **Co-located roles regression fix (2026-08-29)**: only the `data` role may consume `PAT_master_data.txt`; the `exec` role skips it (rename-race fix so the engine consistently receives snapshots).
- **Docs refresh + root cleanup** (`2bc93c51`).

## 5. Current operational state (2026-09-01)

- All containers healthy except `pat-backup-sync` (Restarting — unrelated AWS-cli loop issue).
- Realtime engine restarted at 07:58 UTC and has been in **"Data feed outage"** (fail-closed, correct behavior) since warmup because **no `MARKET_SNAPSHOT` has arrived this session**.
- Valkey `pat:last_snapshot` is frozen at **03:29 UTC** (pre-restart) — the Master Node stopped streaming market data ~03:29.

### Connected Windows agents (telemetry)
| Agent | Role | Ver | MT4 | MT5 | License | Note |
|-------|------|-----|-----|-----|---------|------|
| `00589b08…-data` | master | 1.2.53 | true | false | PENDING | WS up, `candles_delivered` frozen @6925 → EA not writing `PAT_master_data.txt` |
| `00589b08…-exec` | client | 1.2.53 | **false** | **false** | — | no MT terminal linked to agent |
| `be414a86…` | client | 1.2.44 | false | true | ACTIVE/ELITE | older agent; MT5 linked, healthy |

**Key finding:** both broken nodes are the new **v1.2.53** build on the same machine (`00589b08`). The older v1.2.44 client still links fine → the server path is sound; the v1.2.53 update (role-installer/per-arch dedupe) is the prime suspect for the Master EA not emitting snapshots and the Client MT not linking. `license=PENDING` on the master is **benign** for data (engine authorizes data nodes regardless).

## 6. Known / open issues

1. **[P1] Master Node data feed dead** — EA not writing `PAT_master_data.txt` after v1.2.53 update; engine shows permanent outage; no live signals. Action: check agent logs for `[IPC] PAT_master_data.txt not present` vs `present — forwarding`; re-attach Master EA on XAUUSD chart; verify v1.2.53 master installer landed the EA files.
2. **[P1] Client Node MT unlinked** — `mt4=false mt5=false`; no terminal attached to agent. Action: attach Client EA to MT4/MT5; confirm role installer used.
3. **[P3] `pat-backup-sync` restart loop** — needs investigation (likely AWS creds/env).
4. **[P3] Stray caches** — `realtime/.ruff_cache`, `research/**/__pycache__` present on disk (gitignored, harmless).

## 7. Guardrails (non-negotiable, from AGENTS.md)

- `NO-TRADE` is a first-class valid result; never force a trade for frequency.
- Hard-risk, news/session, spread/slippage, margin, license/entitlement, TTL/replay/idempotency, emergency-stop, financial-ledger correctness, and security outrank convenience and AI output.
- No live automated trading / real orders / financial mutation without explicit operator authorization.
- Every code change must be committed and pushed to `origin/main` (`git add -A && commit && push`).
- All `docker compose` commands must use `--env-file infra/env/.env`; secrets live only in `infra/env/*.env` (gitignored).

## 8. How to verify / move forward

- Realtime logs: `docker compose --env-file infra/env/.env logs -f realtime`
- Agent/feed health: inspect Valkey `pat:last_snapshot` timestamp; if it ages >180s the engine is correctly in outage.
- Next step on Windows `00589b08` machine: confirm v1.2.53 role installers placed the correct Master/Client EAs and that the EAs are attached + the `Common\Files` path the agent watches matches the EA's `FILE_COMMON` output.
