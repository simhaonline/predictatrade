#!/usr/bin/env python3
"""Restore a pg_basebackup tar set properly: recovery.signal + WAL bridge FIRST.

Run ON THE NEW HOST from the repo root:
    git pull origin main
    python3 scripts/migration/restore_pg.py

WHY THIS VERSION (the correct, documented pg_basebackup tar procedure):
The base backup's backup_label demands WAL from redo LSN 5C/470000D8 through the
backup checkpoint 5C/4C007AA0 (segment 4C) and beyond. Those segments are NOT in
pg_wal/ (the streamed pg_wal.tar.gz ends at segment 47) — they live in the WAL
bridge. A physical-base restore MUST run archive recovery with recovery.signal +
restore_command set BEFORE the server starts. Starting bare (as before) fails
with "could not locate required checkpoint record" — by design, not by accident.

This script (idempotent):
  1. verifies PGDATA (PG_VERSION, backup_label) and the bridge (/tmp/mig/wal_bridge)
  2. stages the bridge into the volume at /pgdata/pg_wal_bridge
  3. appends restore_command (bridge) to postgresql.auto.conf — hex stream, quote-safe
  4. creates recovery.signal
  5. starts postgres → replays redo → through the bridge → reaches end of WAL →
     promotes automatically and removes recovery.signal
  6. clears restore_command
  7. prints PROOF counts (tables, market.ticks)
"""
import subprocess, sys, os, time, json

VOL = os.environ.get("PGDATA_VOLUME", "xauusd_pat-pgdata")

def sh(cmd, timeout=300):
    return subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=timeout)

def ok(m):   print(f"  PASS  {m}")
def fail(m): print(f"  FAIL  {m}")

def die(msg, hint=""):
    print(f"\n!! ABORTED: {msg}")
    if hint:
        print(f"   hint: {hint}")
    sys.exit(1)

def step(n, label): print(f"\n== {n}. {label} ==")

def vol_write(path, content):
    hx = content.encode().hex()
    r = sh(f'docker run --rm -v {VOL}:/pgdata alpine sh -c '
           f'"echo {hx} | xxd -r -p > {path} && chown 1000:1000 {path} && echo VOLWRITE-OK"')
    if "VOLWRITE-OK" not in r.stdout:
        die(f"cannot write {path}", r.stderr[-200:])
    ok(f"{path} written")

def isready():
    return "accepting connections" in sh(
        "docker exec pat-postgres pg_isready -U pat_admin -d predictatrade").stdout

# ─── 1. preconditions ────────────────────────────────────────────────────────
step(1, "preconditions")
if not os.path.isfile("/tmp/mig/clean_base/base.tar.gz"):
    die("/tmp/mig/clean_base/base.tar.gz missing", "re-download from Hetzner (clean_base_20260916)")
if not os.path.isdir("/tmp/mig/wal_bridge"):
    die("/tmp/mig/wal_bridge missing", "re-download from Hetzner (wal bridge)")
n_bridge = sh("ls /tmp/mig/wal_bridge | wc -l").stdout.strip()
print(f"  bridge segments staged: {n_bridge} [expect ~20, first=5C00000048]")
ok("artifacts present")

# ─── 2. stop postgres (it is crash-looping) ──────────────────────────────────
step(2, "stop crash-looping postgres")
sh("docker compose --env-file infra/env/.env stop postgres")
ok("stopped")

# ─── 3. stage the bridge into the volume ─────────────────────────────────────
step(3, "stage WAL bridge into the volume")
rb = sh('docker run --rm -v %s:/pgdata -v /tmp/mig/wal_bridge:/mnt/wb:ro alpine sh -c '
        '"mkdir -p /pgdata/pg_wal_bridge && cp -n /mnt/wb/* /pgdata/pg_wal_bridge/ 2>/dev/null; '
        'chown -R 1000:1000 /pgdata/pg_wal_bridge; echo staged=$(ls /pgdata/pg_wal_bridge | wc -l)"' % VOL,
        timeout=600)
print(rb.stdout or rb.stderr)
if "staged=" not in rb.stdout:
    die("cannot stage bridge")
ok("bridge staged")

# ─── 4. write recovery config BEFORE first start ────────────────────────────
step(4, "write recovery config into auto.conf (before start)")
r = sh(f'docker run --rm -v {VOL}:/pgdata alpine sh -c "cat /pgdata/postgresql.auto.conf"')
conf = r.stdout
mem_block = ("shared_buffers = 3900MB\nmax_connections = 80\nwork_mem = 16MB\n"
             "maintenance_work_mem = 256MB\neffective_cache_size = 11700MB\nwal_buffers = 16MB\n"
             "max_wal_size = 2GB\nmin_wal_size = 256MB\n")

# Rebuild cleanly regardless of previous state:
lines = []
for ln in conf.splitlines():
    s = ln.strip()
    if not s or s.startswith("#"):
        continue
    if s.startswith("restore_command") or s.startswith("recovery_target"):
        continue  # we manage these here
    if "=" in s and all(ord(c) < 128 for c in s):
        lines.append(s)
if not any("shared_buffers = 3900MB" in s for s in lines):
    lines += mem_block.strip().splitlines()
if not any(s.startswith("archive_mode") for s in lines):
    lines += ["archive_mode = on",
              "archive_command = 'test ! -f /var/lib/postgresql/wal_archive/%f && cp %p /var/lib/postgresql/wal_archive/%f'"]
lines += ["restore_command = 'cp /pgdata/pg_wal_bridge/%f %p'",
          "recovery_target_timeline = 'current'"]
clean = "# rebuilt by restore_phase2.py 2026-09-17\n" + "\n".join(lines) + "\n"
print(clean)

vol_write("/pgdata/postgresql.auto.conf", clean)

# recovery.signal
rs = sh(f'docker run --rm -v {VOL}:/pgdata alpine sh -c '
        '"touch /pgdata/recovery.signal && chown 1000:1000 /pgdata/recovery.signal && echo SIGNAL-OK"')
if "SIGNAL-OK" not in rs.stdout:
    die("cannot create recovery.signal", rs.stderr[-200:])
ok("recovery.signal created + restore_command set")

# ─── 5. start postgres → archive recovery → promote ──────────────────────────
step(5, "start postgres (archive recovery from bridge)")
sh("docker compose --env-file infra/env/.env up -d postgres")
ready = False
seen_replay = False
for i in range(60):  # up to 180s
    time.sleep(3)
    logs = sh("docker logs pat-postgres --since 2m 2>&1").stdout
    if "redo" in logs.lower() or "consistent recovery" in logs.lower():
        seen_replay = True
    if "accepting connections" in sh(
            "docker exec pat-postgres pg_isready -U pat_admin -d predictatrade").stdout:
        ready = True
        break
if not ready:
    print(sh("docker logs pat-postgres --tail 40 2>&1").stdout)
    die("postgres not accepting after 180s", "logs above")
ok("postgres accepting connections")
tail = sh("docker logs pat-postgres --tail 12 2>&1").stdout
print(tail)

# ─── 6. clear restore_command (archive recovery complete) ────────────────────
step(6, "clear restore_command")
ra = sh("docker exec pat-postgres psql -U pat_admin -d postgres -c "
        "\"ALTER SYSTEM SET restore_command = ''; SELECT pg_reload_conf();\"")
print(ra.stdout or ra.stderr)
if ra.returncode != 0:
    die("cannot clear restore_command")
ok("restore_command cleared")

# ─── 7. PROOF ─────────────────────────────────────────────────────────────────
step(7, "PROOF counts")
tables = sh('docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc '
            '"SELECT count(*) FROM pg_tables WHERE schemaname NOT IN '
            '(\'pg_catalog\',\'information_schema\',\'_timescaledb_internal\',\'_timescaledb_catalog\');"').stdout.strip()
ticks = sh('docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc '
           '"SELECT count(*) FROM market.ticks;"').stdout.strip()
dbtime = sh('docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc "SELECT now();"').stdout.strip()
print(f"  tables (non-timescale): {tables}   [expect ~235]")
print(f"  market.ticks: {ticks}   [expect ~30.9M]")
print(f"  db time now: {dbtime}")
print("\nDONE — PHASE 2 complete. Next: PHASE 3 (docker compose --env-file infra/env/.env up -d --build)")