#!/usr/bin/env python3
"""Repair postgresql.auto.conf + finish PHASE 2 on the new host — SAFE edition.

Run ON THE NEW HOST from the repo root (as root or hermes with docker perms):
    git pull origin main
    python3 scripts/migration/finish_phase2.py

Idempotent — safe to re-run. Stops at the first failure with the exact reason.

Fixes:  (1) rewrites a corrupted auto.conf (the pasted heredoc lost the '#' on
            the em-dash comment line → postgres "syntax error near token HOST"),
            writing via hex stream (no shell-quoting losses),
        (2) appends the 16GB-host memory profile (ASCII),
        (3) starts postgres, (4) replays the WAL bridge, (5) PROOF counts,
        (6) clears restore_command.
"""
import subprocess, sys, os, time

VOL = os.environ.get("PGDATA_VOLUME", "xauusd_pat-pgdata")

def sh(cmd, timeout=300):
    return subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=timeout)

def vol_write(path, content):
    """Write a file into the volume via hex stream (quote-safe, byte-exact)."""
    hx = content.encode().hex()
    r = sh(f'docker run --rm -v {VOL}:/pgdata alpine sh -c '
           f'"echo {hx} | xxd -r -p > {path} && chown 1000:1000 {path} && echo VOLWRITE-OK"')
    return r

def ok(m):   print(f"  PASS  {m}")
def fail(m): print(f"  FAIL  {m}")

def die(msg):
    print(f"\n!! ABORTED: {msg}")
    sys.exit(1)

def step(n, label): print(f"\n== {n}. {label} ==")

# ─── 1. read auto.conf ───────────────────────────────────────────────────────
step(1, "read postgresql.auto.conf from the volume")
r = sh(f'docker run --rm -v {VOL}:/pgdata alpine sh -c "cat /pgdata/postgresql.auto.conf"')
if r.returncode != 0:
    die(f"cannot read postgresql.auto.conf ({r.stderr.strip()[:200]})")
conf = r.stdout
print(conf)

# ─── 2. detect corruption (non-ascii or non-setting lines) ───────────────────
step(2, "corruption check")
bad = any(ord(c) > 127 for c in conf)
for i, ln in enumerate(conf.splitlines(), 1):
    s = ln.strip()
    if s and not s.startswith("#") and "=" not in s:
        bad = True
        print(f"  corrupt line {i}: {ln!r}")
print("  PASS  clean — no repair needed" if not bad else "  corrupt — will rewrite")

# ─── 3. build the clean conf ─────────────────────────────────────────────────
step(3, "build clean auto.conf")
keep = []
if not bad:
    clean = conf
else:
    for ln in conf.splitlines():
        s = ln.strip()
        if not s or s.startswith("#"):
            continue  # drop ALL comments (avoids any mangled remnant)
        if "=" in s and all(ord(c) < 128 for c in s):
            keep.append(ln)
    clean = "# rebuilt by finish_phase2.py 2026-09-17\n" + "\n".join(keep) + "\n"

if "shared_buffers = 3900MB" not in clean:
    clean += ("\n# 16GB HOST memory profile 2026-09-17\n"
              "shared_buffers = 3900MB\n"
              "max_connections = 80\n"
              "work_mem = 16MB\n"
              "maintenance_work_mem = 256MB\n"
              "effective_cache_size = 11700MB\n"
              "wal_buffers = 16MB\n"
              "max_wal_size = 2GB\n"
              "min_wal_size = 256MB\n")

# ensure archive_mode retained (needed post-recovery)
if "archive_mode" not in clean:
    clean = clean.rstrip("\n") + "\narchive_mode = on\narchive_command = 'test ! -f /var/lib/postgresql/wal_archive/%f && cp %p /var/lib/postgresql/wal_archive/%f'\n"

print(clean)

# ─── 4. write it back (hex stream — no quoting hazards) ──────────────────────
step(4, "write repaired conf")
if bad:
    hx = clean.encode().hex()
    rw = sh(f'docker run --rm -v {VOL}:/pgdata alpine sh -c '
            f'"echo {hx} | xxd -r -p > /pgdata/postgresql.auto.conf && '
            f'chown 1000:1000 /pgdata/postgresql.auto.conf && echo VOLWRITE-OK"')
    if "VOLWRITE-OK" not in rw.stdout:
        die(f"cannot write auto.conf ({rw.stderr.strip()[:200]})")
    ok("written via hex stream (byte-exact)")
else:
    ok("unchanged")

# ─── 5. start postgres ───────────────────────────────────────────────────────
step(5, "start postgres")
sh("docker compose --env-file infra/env/.env up -d postgres")
ready = False
for _ in range(30):
    import time; time.sleep(3)
    if "accepting connections" in sh("docker exec pat-postgres pg_isready -U pat_admin -d predictatrade").stdout:
        ready = True
        break
if ready:
    ok("postgres accepting connections")
else:
    print(sh("docker logs pat-postgres --tail 25").stdout)
    die("postgres not accepting after 90s — logs above")

# ─── 6. WAL bridge + restore_command + replay ────────────────────────────────
step(6, "WAL bridge replay")
if not os.path.isdir("/tmp/mig/wal_bridge"):
    die("/tmp/mig/wal_bridge missing — re-download from Hetzner (PHASE 2 step 2.1)")
rb = sh('docker run --rm -v xauusd_pat-pgdata:/pgdata -v /tmp/mig/wal_bridge:/mnt/wb:ro '
        'alpine sh -c "mkdir -p /pgdata/pg_wal_bridge && cp -n /mnt/wb/* /pgdata/pg_wal_bridge/ 2>/dev/null; '
        'chown -R 1000:1000 /pgdata/pg_wal_bridge; echo staged=$(ls /pgdata/pg_wal_bridge | wc -l)"',
        timeout=600)
print(rb.stdout or rb.stderr)
if "staged=" not in rb.stdout:
    die("cannot stage wal bridge")

ra = sh("docker exec pat-postgres psql -U pat_admin -d postgres -c "
        "\"ALTER SYSTEM SET restore_command = 'cp /pgdata/pg_wal_bridge/%f %p 2>/dev/null || exit 1';\"")
print(ra.stdout or ra.stderr)
if ra.returncode != 0:
    die("ALTER SYSTEM restore_command failed")

sh("docker compose --env-file infra/env/.env restart postgres")
print("  waiting for replay (up to 120s)...")
import time
for _ in range(40):
    time.sleep(3)
    if "accepting connections" in sh("docker exec pat-postgres pg_isready -U pat_admin -d predictatrade").stdout:
        break
if "accepting connections" not in sh("docker exec pat-postgres pg_isready -U pat_admin -d predictatrade").stdout:
    print(sh("docker logs pat-postgres --tail 25").stdout)
    die("postgres did not come back after restart")

# ─── 7. PROOF ─────────────────────────────────────────────────────────────────
step(7, "PROOF counts")
tables = sh('docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc '
            '"SELECT count(*) FROM pg_tables WHERE schemaname NOT IN '
            '(\'pg_catalog\',\'information_schema\',\'_timescaledb_internal\',\'_timescaledb_catalog\');"').stdout.strip()
ticks = sh('docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc '
           '"SELECT count(*) FROM market.ticks;"').stdout.strip()
print(f"  tables (non-timescale): {tables}   [expect ~235]")
print(f"  market.ticks: {ticks}   [expect ~30.9M]")

# ─── 8. clear restore_command ─────────────────────────────────────────────────
step(8, "clear restore_command")
rc = sh("docker exec pat-postgres psql -U pat_admin -d postgres -c "
        "\"ALTER SYSTEM SET restore_command = ''; SELECT pg_reload_conf();\"")
print(rc.stdout or rc.stderr)
print("\nDONE — PHASE 2 complete. Next: PHASE 3 (docker compose --env-file infra/env/.env up -d --build)")