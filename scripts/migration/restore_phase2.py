#!/usr/bin/env python3
"""restore_phase2.py — FINAL: correct restore path + built-in diagnostics.

Run ON THE NEW HOST from the repo root:
    git pull origin main
    python3 scripts/migration/restore_phase2.py

If postgres comes up: prints PROOF counts. If not: prints the FULL evidence
bundle (why it died, OOM flags, kernel lines, logs) — paste that block back.

Path bug fixed in this version: the restore_command must use the path INSIDE
the pat-postgres container (/var/lib/postgresql/data/pg_wal_bridge/%f), not the
helper-container mount (/pgdata). The earlier conf pointed at /pgdata — which
does not exist inside pat-postgres, so every WAL fetch failed.
"""
import subprocess, sys, os, time, re, io, tarfile, json

VOL = os.environ.get("PGDATA_VOLUME", "xauusd_pat-pgdata")
CONTAINER = "pat-postgres"

def run(argv, timeout=300):
    return subprocess.run(argv, capture_output=True, text=True, timeout=timeout)

def alpine(script, mounts=True):
    argv = ["docker", "run", "--rm"]
    if mounts:
        argv += ["-v", f"{VOL}:/pgdata", "-v", "/tmp/mig:/mnt/mig:ro"]
    argv += ["alpine", "sh", "-c", script]
    return run(argv)

def tar_into_volume(name, content_bytes):
    buf = io.BytesIO()
    tf = tarfile.open(fileobj=buf, mode="w")
    ti = tarfile.TarInfo(name)
    ti.size = len(content_bytes)
    ti.mtime = 0
    ti.mode = 0o600
    tf.addfile(ti, io.BytesIO(content_bytes))
    tf.close()
    p = subprocess.Popen(
        ["docker", "run", "--rm", "-i", "-v", f"{VOL}:/pgdata", "alpine",
         "sh", "-c",
         f"tar -x -C /pgdata && chown 1000:1000 /pgdata/{name} && echo TARWRITE-OK"],
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    o, e = p.communicate(buf.getvalue())
    return (o or b"").decode(), (e or b"").decode()

def sh(cmd, timeout=120):
    return subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=timeout)

def ok(m):   print(f"  PASS  {m}")

def die(msg, hint=""):
    print(f"\n!! ABORTED: {msg}")
    if hint:
        print(f"   hint: {hint}")
    print("\n===== DIAGNOSTIC BUNDLE (paste this whole block back) =====")
    try:
        print("-- container lifecycle --")
        d = json.loads(run(["docker", "inspect", CONTAINER]).stdout)[0]["State"]
        print(json.dumps({k: d.get(k) for k in
            ("Running","Status","ExitCode","OOMKilled","Error","StartedAt","FinishedAt")}, indent=1))
        print("-- docker logs (all attempts, last 120 lines) --")
        print(run(["docker", "logs", CONTAINER, "--tail", "120"]).stdout[-8000:])
        print("-- kernel OOM (last 15) --")
        print(run("dmesg 2>/dev/null | grep -iE 'oom|killed process|out of memory' | tail -15 || "
                  "journalctl -k --since '2 hours ago' 2>/dev/null | grep -iE 'oom|killed' | tail -15",
                  shell=True).stdout or "(none)")
        print("-- host memory --")
        print(run("free -m", shell=True).stdout)
        print("-- PGDATA pg_wal_bridge (via postgres-container mount path) --")
        print(run(["docker", "exec", CONTAINER, "sh", "-c",
                   "ls /var/lib/postgresql/data/pg_wal_bridge 2>&1 | head -4; "
                   "ls /var/lib/postgresql/data/pg_wal_bridge 2>/dev/null | wc -l"]).stdout)
        print("-- postmaster.pid --")
        print(run(["docker", "exec", CONTAINER, "sh", "-c",
                   "cat /var/lib/postgresql/data/postmaster.pid 2>/dev/null | head -3; echo rc=$?"]).stdout)
        print("-- process tree --")
        print(run(["docker", "top", CONTAINER]).stdout)
    except Exception as e:
        print("diag failed:", e)
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
run(["docker", "compose", "--env-file", "infra/env/.env", "stop", "postgres"])
ok("stopped")

# ─── 3. stage the bridge into the volume ─────────────────────────────────────
step(3, "stage WAL bridge into the volume")
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
# NOTE the path: inside pat-postgres the volume is at /var/lib/postgresql/data
RESTORE_CMD = "cp /var/lib/postgresql/data/pg_wal_bridge/%f %p"

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
lines += [f"restore_command = '{RESTORE_CMD}'",
          "recovery_target_timeline = 'current'"]
clean = "# rebuilt by restore_phase2.py 2026-09-17 v2\n" + "\n".join(lines) + "\n"
print(clean)

o, e = tar_into_volume("postgresql.auto.conf", clean.encode())
if "TARWRITE-OK" not in o:
    die("cannot write auto.conf", (e or "")[-300:])
o2, e2 = tar_into_volume("recovery.signal", b"")
if "TARWRITE-OK" not in o2:
    die("cannot create recovery.signal", (e2 or "")[-300:])
ok(f"auto.conf written (restore_command → {RESTORE_CMD}) + recovery.signal created")

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
    die("postgres not accepting after 180s", "diagnostic bundle above")
ok("postgres accepting connections")

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