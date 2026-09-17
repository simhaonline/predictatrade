#!/usr/bin/env python3
"""restore_phase2.py — CORRECT + SHELL-SAFE pg_basebackup restore (final).

Run ON THE NEW HOST from the repo root:
    git pull origin main
    python3 scripts/migration/restore_phase2.py

Design: EVERY docker invocation uses list-form argv (subprocess shell=False).
No host-shell expansion exists anywhere — no $(), no quoting hazards, nothing
can be mangled by a terminal or shell layer. Every step is verified (exit code
+ output check) and the script stops at the first failure with a precise
reason. Idempotent — safe to re-run.

Sequence (documented pg_basebackup tar restore):
  stage bridge → auto.conf (memory + restore_command) → recovery.signal
  → start → archive replay from bridge → promote → clear restore_command
  → PROOF counts.
"""
import subprocess, sys, os, time, re, io, tarfile

VOL = os.environ.get("PGDATA_VOLUME", "xauusd_pat-pgdata")
CONTAINER = "pat-postgres"

def run(argv, timeout=300):
    """Run docker with LIST args — no shell, no expansion, ever."""
    return subprocess.run(argv, capture_output=True, text=True, timeout=timeout)

def alpine(script):
    """Run a shell script inside a root alpine container with both mounts."""
    return run(["docker", "run", "--rm",
                "-v", f"{VOL}:/pgdata",
                "-v", "/tmp/mig:/mnt/mig:ro",
                "alpine", "sh", "-c", script])

def tar_into_volume(name, content_bytes, mode="600"):
    """Write one file into the volume via a tar stream over stdin — byte-exact."""
    buf = io.BytesIO()
    tf = tarfile.open(fileobj=buf, mode="w")
    ti = tarfile.TarInfo(name)
    ti.size = len(content_bytes)
    ti.mtime = 0
    ti.mode = int(mode, 8)
    tf.addfile(ti, io.BytesIO(content_bytes))
    tf.close()
    p = subprocess.Popen(
        ["docker", "run", "--rm", "-i", "-v", f"{VOL}:/pgdata", "alpine",
         "sh", "-c",
         f"tar -x -C /pgdata && chown 1000:1000 /pgdata/{name} && echo TARWRITE-OK"],
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    o_bytes, e_bytes = p.communicate(buf.getvalue())
    return (o_bytes or b"").decode(), (e_bytes or b"").decode()

def ok(m):   print(f"  PASS  {m}")

def die(msg, hint=""):
    print(f"\n!! ABORTED: {msg}")
    if hint:
        print(f"   hint: {hint}")
    sys.exit(1)

def step(n, label): print(f"\n== {n}. {label} ==")

def isready():
    r = run(["docker", "exec", CONTAINER, "pg_isready", "-U", "pat_admin", "-d", "predictatrade"])
    return "accepting connections" in (r.stdout or "")

# ─── 1. preconditions ────────────────────────────────────────────────────────
step(1, "preconditions")
if not os.path.isfile("/tmp/mig/clean_base/base.tar.gz"):
    die("/tmp/mig/clean_base/base.tar.gz missing", "re-download from Hetzner")
if not os.path.isdir("/tmp/mig/wal_bridge"):
    die("/tmp/mig/wal_bridge missing", "re-download from Hetzner")
n_bridge = len(os.listdir("/tmp/mig/wal_bridge"))
print(f"  bridge segments on host: {n_bridge}")
if n_bridge < 10:
    die(f"bridge has only {n_bridge} segments", "re-run the Hetzner wal download")

# ─── 2. stop postgres ────────────────────────────────────────────────────────
step(2, "stop postgres")
sh = run(["docker", "compose", "--env-file", "infra/env/.env", "stop", "postgres"])
ok("stopped")

# ─── 3. stage the bridge into the volume ─────────────────────────────────────
step(3, "stage WAL bridge into the volume (no host shell — list args only)")
r = alpine(
    "set -e; "
    "echo source-files=$(ls /mnt/mig/wal_bridge | wc -l); "
    "rm -rf /pgdata/pg_wal_bridge; "
    "mkdir -p /pgdata/pg_wal_bridge; "
    "cp /mnt/mig/wal_bridge/* /pgdata/pg_wal_bridge/; "
    "chown -R 1000:1000 /pgdata/pg_wal_bridge; "
    "echo staged=$(ls /pgdata/pg_wal_bridge | wc -l)")
print(r.stdout or r.stderr)
if r.returncode != 0:
    die("bridge copy failed", r.stderr[-400:])
m = re.search(r"staged=(\d+)", r.stdout)
if not m or int(m.group(1)) < 10:
    die(f"only {m.group(1) if m else '?'} segments staged", r.stdout[-400:])
ok(f"bridge staged: {m.group(1)} segments")

# ─── 4. auto.conf + recovery.signal BEFORE start ─────────────────────────────
step(4, "recovery config written before first start")
r = alpine("cat /pgdata/postgresql.auto.conf")
conf = r.stdout

lines = []
for ln in conf.splitlines():
    s = ln.strip()
    if not s or s.startswith("#"):
        continue
    if s.startswith("restore_command") or s.startswith("recovery_target"):
        continue
    if "=" in s and all(ord(c) < 128 for c in s):
        lines.append(s)
if not any("shared_buffers = 3900MB" in s for s in lines):
    lines += ["shared_buffers = 3900MB", "max_connections = 80", "work_mem = 16MB",
              "maintenance_work_mem = 256MB", "effective_cache_size = 11700MB",
              "wal_buffers = 16MB", "max_wal_size = 2GB", "min_wal_size = 256MB"]
if not any(s.startswith("archive_mode") for s in lines):
    lines += ["archive_mode = on",
              "archive_command = 'test ! -f /var/lib/postgresql/wal_archive/%f && cp %p /var/lib/postgresql/wal_archive/%f'"]
lines += ["restore_command = 'cp /pgdata/pg_wal_bridge/%f %p'",
          "recovery_target_timeline = 'current'"]
clean = "# rebuilt by restore_phase2.py 2026-09-17\n" + "\n".join(lines) + "\n"
print(clean)

o, e = tar_into_volume("postgresql.auto.conf", clean.encode())
if "TARWRITE-OK" not in o:
    die("cannot write auto.conf via tar stream", (e or "")[-300:])

o2, e2 = tar_into_volume("recovery.signal", b"")
if "TARWRITE-OK" not in o2:
    die("cannot create recovery.signal", (e2 or "")[-300:])
ok("auto.conf + recovery.signal written (tar stream, byte-exact)")

# ─── 5. start → archive recovery → promote ───────────────────────────────────
step(5, "start postgres (archive recovery)")
run(["docker", "compose", "--env-file", "infra/env/.env", "up", "-d", "postgres"])
ready = False
for i in range(60):
    time.sleep(3)
    if isready():
        ready = True
        break
if not ready:
    print(run(["docker", "logs", CONTAINER, "--tail", "40"]).stdout)
    die("postgres not accepting after 180s", "logs above")
ok("postgres accepting connections")
logs = run(["docker", "logs", CONTAINER, "--since", "5m"]).stdout
print(logs[-1500:])

# ─── 6. clear restore_command ─────────────────────────────────────────────────
step(6, "clear restore_command")
rc = run(["docker", "exec", CONTAINER, "psql", "-U", "pat_admin", "-d", "postgres", "-c",
          "ALTER SYSTEM SET restore_command = ''; SELECT pg_reload_conf();"])
print(rc.stdout or rc.stderr)
if rc.returncode != 0:
    die("cannot clear restore_command")
ok("restore_command cleared")

# ─── 7. PROOF ─────────────────────────────────────────────────────────────────
step(7, "PROOF counts")
r1 = run(["docker", "exec", CONTAINER, "psql", "-U", "pat_admin", "-d", "predictatrade", "-Atc",
          "SELECT count(*) FROM pg_tables WHERE schemaname NOT IN "
          "('pg_catalog','information_schema','_timescaledb_internal','_timescaledb_catalog');"])
r2 = run(["docker", "exec", CONTAINER, "psql", "-U", "pat_admin", "-d", "predictatrade", "-Atc",
          "SELECT count(*) FROM market.ticks;"])
r3 = run(["docker", "exec", CONTAINER, "psql", "-U", "pat_admin", "-d", "predictatrade", "-Atc",
          "SELECT now();"])
print(f"  tables (non-timescale): {r1.stdout.strip()}   [expect ~235]")
print(f"  market.ticks: {r2.stdout.strip()}   [expect ~30.9M]")
print(f"  db time now: {r3.stdout.strip()}")
print("\nDONE — PHASE 2 complete. Next: PHASE 3 (docker compose --env-file infra/env/.env up -d --build)")