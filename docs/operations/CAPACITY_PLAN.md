# Capacity Plan & Measurement — Predict-A-Trade XAUUSD

**Status:** v1.0 (2026-09-11) · companion to [HETZNER_DEPLOYMENT.md](HETZNER_DEPLOYMENT.md)
**Tool:** `scripts/capacity/measure_capacity.py`

This turns the "VPS vs Dedicated at 10k subscribers" question into **measured
numbers + explicit assumptions**, not guesswork. It does not fabricate data: it
reads the live stack, reports the real baseline, and projects using
overridable per-subscriber estimates.

---

## 1. What the script measures (live, no secrets printed)

| Signal | Source |
|---|---|
| Host CPUs / RAM used / disk used | `free`, `df`, `nproc` |
| Realtime engine RSS / CPU / goroutines / net bytes | `realtime:13081/metrics` (Go process metrics) |
| WS clients connected | `realtime:13081/health` (`ws_clients`) |
| Postgres active connections | `pg_stat_activity` (in-container, creds never printed) |
| Postgres queries/sec | `pg_stat_database` commit+rollback delta |
| Disk KB/s + IOPS | `/proc/diskstats` (sector deltas) |

It samples twice ~5s apart to derive per-second rates.

## 2. How to run

```bash
# baseline + projection to 10,000 subscribers (default)
python3 scripts/capacity/measure_capacity.py

# project to a different size
python3 scripts/capacity/measure_capacity.py --target 2500

# calibrate with measured per-user assumptions (see §4)
python3 scripts/capacity/measure_capacity.py \
  --ws-mb 0.6 --pg-conn-per-1k 12 --qps-per-1k 45 \
  --rt-ram-mb-per-1k 300 --target 10000
```

Requires `docker` (to read the Postgres role + query stats) and `curl`. Prometheus
port is not host-published, so the script scrapes `realtime` directly; pass
`--prometheus <url>` only if you expose it.

## 3. The four dedicated-server triggers

Mirrors the decision logic in HETZNER_DEPLOYMENT.md:

| # | Trigger | Fires when |
|---|---|---|
| T1 | **RAM ceiling** | projected realtime RAM > 70% of a 128 GB VPS cap (~90 GB) |
| T2 | **CPU headroom** | projected DB q/s sustained > ~60% of 16-core capacity |
| T3 | **Disk IOPS** | projected write q/s > 8000 (WAL + data + pg_cron contention) |
| T4 | **HA / replica** | target ≥ 2000 subscribers → recommend primary + streaming replica pair |

Output ends with a verdict: `VPS OK` / `DEDICATED SOON (scale-out)` /
`DEDICATED (metal)`.

## 4. Calibrating the assumptions (do this at peak)

The default per-subscriber estimates are conservative placeholders. They become
real once you run the script **during market open with real clients connected**:

1. Note the measured `WS clients` and `Postgres queries/sec` from the baseline.
2. Derive: `qps-per-1k = measured_qps / (ws_clients/1000)` (or per current subs).
3. Same for `--pg-conn-per-1k` and `--ws-mb` (net egress/1k from the baseline).
4. Re-run with those values. Now the projection is **data-driven**, not assumed.

Until then, treat the projection as an upper-bound sanity check, not a forecast.

## 5. Current measured baseline (reference, 2026-09-11)

Single Hetzner VPS, 16 vCPU / 62 GB RAM / 2 TB NVMe, ~9.4 GB RAM used:

```
realtime RSS      : 73 MB      realtime CPU : 8.8%
goroutines        : 51         WS clients   : 0 (no live clients on this box)
Postgres conns    : 18         queries/sec  : ~55
disk read/write   : ~3.3 / 3.7 MB/s   IOPS ~14k
```

With default assumptions the 10k projection stays **well within VPS limits**
(RAM 2 GB vs 90 GB cap, 300 q/s vs ~38k sustainable). The only standing
recommendation at 10k is **T4: add a Postgres streaming replica** for HA — which
the R2 WAL archive already makes straightforward.

> Reality check: this box currently has 0 connected WS clients, so the per-user
> footprint can't yet be derived from live data. Calibrate at peak before trusting
> the 10k numbers (§4). The architecture (stateless fan-out behind Cloudflare,
> separated stateful services) scales horizontally regardless.

## 6. Scaling playbook (when a trigger fires)

- **T1 (RAM):** move Postgres to its own node, or buy dedicated EX/AX metal.
- **T2/T3 (CPU/IOPS):** add **PgBouncer** (connection pooling), a **streaming
  replica** (read offload), and split `pat-backtest` / `pat-realtime` onto their
  own nodes (services are already separated in compose).
- **T4 (HA):** primary + replica pair; keep R2 WAL archive as PITR floor.

All of the above are host-agnostic thanks to the R2 `wal/ db/ code/` backups.
