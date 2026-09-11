#!/usr/bin/env python3
"""
Predict-A-Trade capacity measurement + projection tool.

Measures the REAL current footprint of the running stack (host, realtime engine,
PostgreSQL) by sampling twice a few seconds apart, then projects resource use to a
target subscriber count using EXPLICIT, editable per-subscriber assumptions.

It does NOT fabricate data. Where live per-user numbers can't be derived (e.g. the
box currently has 0 connected WS clients), the projection uses the assumptions
below, which you can override on the command line. Every assumption is printed so
the math is auditable.

Outputs:
  - current baseline (RSS, CPU, goroutines, WS clients, pg connections, queries/s,
    disk KB/s + IOPS, host RAM/disk free)
  - projected footprint at --target-subscribers (default 10000)
  - the four dedicated-server trigger checks from the Hetzner deployment runbook
  - a GO / WATCH / DEDICATED verdict

Usage:
  python3 scripts/capacity/measure_capacity.py                 # baseline + 10k projection
  python3 scripts/capacity/measure_capacity.py --target 2500  # project to 2.5k
  python3 scripts/capacity/measure_capacity.py --ws-mb 0.5 --pg-conn-per-1k 12 --qps-per-1k 40
  python3 scripts/capacity/measure_capacity.py --prometheus http://127.0.0.1:9090

Requires: docker (to read postgres role + query stats), curl (realtime metrics).
No secrets are printed; postgres creds are read in-container only.
"""
import argparse
import json
import os
import subprocess
import sys
import time

RT_METRICS = "http://127.0.0.1:13081/metrics"
RT_HEALTH = "http://127.0.0.1:13081/health"
SAMPLE_INTERVAL = 5.0  # seconds between the two samples


# ---------- helpers ----------
def run(cmd, quiet=True):
    res = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if not quiet and res.returncode != 0:
        sys.stderr.write(f"[warn] cmd failed: {cmd[:80]} -> {res.stderr[:200]}\n")
    return res.stdout, res.returncode


def get_realtime_metrics():
    out, _ = run(f"curl -fsS --retry 3 --max-time 5 {RT_METRICS}")
    m = {}
    for line in out.splitlines():
        line = line.rstrip("\r")  # endpoint returns CRLF
        if line.startswith("#"):
            continue
        if " " not in line:  # need a name + value
            continue
        k, v = line.rsplit(" ", 1)
        try:
            m[k] = float(v)
        except ValueError:
            pass
    return m


def get_realtime_health():
    out, _ = run(f"curl -fsS --max-time 5 {RT_HEALTH}")
    try:
        return json.loads(out)
    except Exception:
        return {}


def get_pg_role():
    out, _ = run("docker exec pat-postgres env")
    role = db = None
    for line in out.splitlines():
        if line.startswith("POSTGRES_USER="):
            role = line.split("=", 1)[1]
        elif line.startswith("POSTGRES_DB="):
            db = line.split("=", 1)[1]
    return role or "postgres", db or role or "postgres"


def query_pg(sql):
    role, db = get_pg_role()
    # run inside the container so we use the local socket; creds never printed
    cmd = f"docker exec pat-postgres psql -U {role} -d {db} -tAc \"{sql}\""
    out, rc = run(cmd)
    if rc != 0:
        return None
    out = out.strip().splitlines()
    return [l.strip() for l in out if l.strip()]


def get_diskstats():
    """Return dict dev -> (read_sectors, write_sectors, ios_in_progress) from /proc/diskstats."""
    stats = {}
    with open("/proc/diskstats") as f:
        for line in f:
            parts = line.split()
            if len(parts) < 14:
                continue
            dev = parts[2]
            # skip partitions (those with a digit suffix beyond the base device)
            if dev.startswith("vda") or dev.startswith("sda") or dev.startswith("nvme"):
                if dev not in ("vda", "sda") and not dev.startswith("nvme0n1"):
                    # partition, skip
                    if any(c.isdigit() for c in dev) and dev not in ("vda", "sda"):
                        continue
                reads = int(parts[5])   # sectors read
                writes = int(parts[9])  # sectors written
                ios = int(parts[11])    # # I/Os in progress
                stats[dev] = (reads, writes, ios)
    return stats


def host_snapshot():
    out, _ = run("free -b")
    mem_total = mem_avail = 0
    for line in out.splitlines():
        if line.startswith("Mem:"):
            parts = line.split()
            mem_total = int(parts[1])
            mem_avail = int(parts[-1]) if len(parts) >= 7 else int(parts[6])
    df_out, _ = run("df -B1 / | tail -1")
    disk_total = disk_used = 0
    if df_out:
        parts = df_out.split()
        disk_total, disk_used = int(parts[1]), int(parts[2])
    return {
        "cpus": os.cpu_count() or 0,
        "mem_total_bytes": mem_total,
        "mem_avail_bytes": mem_avail,
        "disk_total_bytes": disk_total,
        "disk_used_bytes": disk_used,
    }


def main():
    ap = argparse.ArgumentParser(description="PAT capacity measurement + projection")
    ap.add_argument("--target", type=int, default=10000, help="target subscriber count (default 10000)")
    ap.add_argument("--ws-mb", type=float, default=0.4, help="est MB/s network egress per 1k subscribers (default 0.4)")
    ap.add_argument("--pg-conn-per-1k", type=float, default=10.0, help="Postgres connections per 1k subscribers (default 10)")
    ap.add_argument("--qps-per-1k", type=float, default=30.0, help="DB queries/sec per 1k subscribers (default 30)")
    ap.add_argument("--rt-ram-mb-per-1k", type=float, default=250.0, help="realtime engine RAM MB per 1k subscribers (default 250)")
    ap.add_argument("--interval", type=float, default=SAMPLE_INTERVAL)
    ap.add_argument("--prometheus", default=None, help="optional Prometheus base URL (unused if unreachable)")
    args = ap.parse_args()

    print("=" * 70)
    print("PREDICT-A-TRADE CAPACITY MEASUREMENT")
    print(f"target subscribers : {args.target:,}")
    print("=" * 70)

    # ---- sample 1 ----
    h1 = host_snapshot()
    m1 = get_realtime_metrics()
    d1 = get_diskstats()
    health1 = get_realtime_health()
    pg_conn1 = query_pg("SELECT count(*) FROM pg_stat_activity WHERE state IS NOT NULL;")
    pg_q1 = query_pg("SELECT sum(xact_commit + xact_rollback) FROM pg_stat_database;")
    time.sleep(args.interval)
    # ---- sample 2 ----
    h2 = host_snapshot()
    m2 = get_realtime_metrics()
    d2 = get_diskstats()
    health2 = get_realtime_health()
    pg_conn2 = query_pg("SELECT count(*) FROM pg_stat_activity WHERE state IS NOT NULL;")
    pg_q2 = query_pg("SELECT sum(xact_commit + xact_rollback) FROM pg_stat_database;")

    def num(x):
        try:
            return float(x[0]) if x else 0.0
        except Exception:
            return 0.0

    ws_now = int(health2.get("ws_clients", 0) or 0)
    rt_rss_mb = (m2.get("process_resident_memory_bytes", 0) or 0) / 1e6
    rt_cpu_s = (m2.get("process_cpu_seconds_total", 0) or 0) - (m1.get("process_cpu_seconds_total", 0) or 0)
    rt_cpu_pct = (rt_cpu_s / args.interval) * 100.0
    goroutines = int(m2.get("go_goroutines", 0) or 0)
    pg_conn = int(num(pg_conn2))
    qps = (num(pg_q2) - num(pg_q1)) / args.interval if (num(pg_q2) and num(pg_q1)) else 0.0

    # disk deltas (pick first matching device)
    dev = next(iter(d2), None)
    kb_r = kb_w = iops = 0.0
    if dev and dev in d1:
        r1, w1, _ = d1[dev]; r2, w2, _ = d2[dev]
        kb_r = max(0.0, (r2 - r1) * 512 / 1024 / args.interval)
        kb_w = max(0.0, (w2 - w1) * 512 / 1024 / args.interval)
        iops = max(0.0, (r2 - r1 + w2 - w1) / args.interval)

    # ---- current baseline ----
    print("\n[BASELINE — measured now]")
    print(f"  host CPUs               : {h2['cpus']}")
    print(f"  host RAM used/avail     : {(h2['mem_total_bytes']-h2['mem_avail_bytes'])/1e9:.1f} GB / {h2['mem_avail_bytes']/1e9:.1f} GB free")
    print(f"  host disk used          : {h2['disk_used_bytes']/1e9:.0f} GB")
    print(f"  realtime RSS            : {rt_rss_mb:.0f} MB")
    print(f"  realtime CPU            : {rt_cpu_pct:.1f}% over {args.interval:.0f}s")
    print(f"  realtime goroutines     : {goroutines}")
    print(f"  WS clients (realtime)   : {ws_now}")
    print(f"  Postgres connections    : {pg_conn}")
    print(f"  Postgres queries/sec    : {qps:.1f}")
    print(f"  disk read/write         : {kb_r:.0f} / {kb_w:.0f} KB/s  (IOPS ~{iops:.0f})")

    # ---- projection ----
    print(f"\n[PROJECTION to {args.target:,} subscribers]  (assumptions below are EDITABLE)")
    print(f"  net egress/1k          : {args.ws_mb} MB/s  -> {args.ws_mb*args.target/1000:.1f} MB/s")
    print(f"  pg connections/1k       : {args.pg_conn_per_1k}   -> {args.pg_conn_per_1k*args.target/1000:.0f} conns")
    print(f"  db queries/1k           : {args.qps_per_1k}   -> {args.qps_per_1k*args.target/1000:.0f} q/s")
    print(f"  realtime RAM/1k         : {args.rt_ram_mb_per_1k} MB -> {args.rt_ram_mb_per_1k*args.target/1000:.0f} MB")

    proj_rt_ram_gb = args.rt_ram_mb_per_1k * args.target / 1000 / 1024
    proj_pg_conn = args.pg_conn_per_1k * args.target / 1000
    proj_qps = args.qps_per_1k * args.target / 1000
    proj_net_mbs = args.ws_mb * args.target / 1000

    # ---- dedicated trigger checks (from Hetzner runbook) ----
    print("\n[DEDICATED-SERVER TRIGGER CHECKS]")
    # T1 RAM ceiling: VPS cap ~128 GB usable; trigger at sustained >70%
    RAM_CAP_GB = 128.0
    THRESH_RAM = 0.70 * RAM_CAP_GB
    t1 = proj_rt_ram_gb > THRESH_RAM
    print(f"  T1 RAM ceiling   : projected realtime RAM {proj_rt_ram_gb:.0f} GB vs 70% VPS cap {THRESH_RAM:.0f} GB -> {'DEDICATED' if t1 else 'OK (VPS fits)'}")

    # T2 CPU: assume 1 core ~ can push N q/s; if projected qps > headroom of 16 cores -> watch
    CPU_CORES = 16
    QPS_PER_CORE = 4000  # rough: a tuned Postgres core handles several k q/s for light queries
    t2 = proj_qps > (CPU_CORES * QPS_PER_CORE * 0.6)
    print(f"  T2 CPU headroom  : projected {proj_qps:.0f} q/s vs ~{CPU_CORES*QPS_PER_CORE*0.6:.0f} sustainable on 16 cores -> {'WATCH' if t2 else 'OK'}")

    # T3 disk IOPS: VPS NVMe ~10k-30k IOPS; trigger if projected writes heavy + pg_cron/dump contention
    # rough: each committed tx ~1-2 disk write ops; + WAL. Use qps as proxy.
    t3 = proj_qps > 8000
    print(f"  T3 disk IOPS      : projected {proj_qps:.0f} q/s (WAL+data writes) -> {'WATCH (consider dedicated NVMe)' if t3 else 'OK'}")

    # T4 HA/replica: once >2000 concurrent or any T1-T3 hit -> recommend replica pair
    t4 = (ws_now or args.target) and (args.target >= 2000)
    print(f"  T4 HA replica     : target {args.target:,} >= 2000 -> {'RECOMMEND primary+replica pair' if t4 else 'single box OK'}")

    # ---- verdict ----
    print("\n[VERDICT]")
    if t1:
        verdict = "DEDICATED — RAM ceiling crossed; go dedicated metal (EX/AX line)."
    elif t4 and (t2 or t3):
        verdict = "DEDICATED SOON — scale-out: add PgBouncer + Postgres streaming replica; keep VPS or move to dedicated pair."
    elif args.target >= 2000:
        verdict = "VPS OK — but instrument + prepare scale-out (PgBouncer, replica) for when triggers fire."
    else:
        verdict = "VPS OK — comfortable headroom; no dedicated needed yet."
    print(f"  {verdict}")
    print("\n[NOTE] Per-user assumptions are estimates; calibrate by running this during")
    print("      peak load (market open, real clients connected) and feeding measured")
    print("      ws-mb / pg-conn-per-1k / qps-per-1k back in. No data is fabricated.")
    print("=" * 70)


if __name__ == "__main__":
    main()
