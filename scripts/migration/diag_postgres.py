#!/usr/bin/env python3
"""Diagnose why pat-postgres is silent on the new host (no FATAL, no LOG lines).

Run ON THE NEW HOST from the repo root:
    python3 scripts/migration/diag_postgres.py

Prints a single evidence bundle: container state, full logs, OOM flags,
cgroup memory pressure, PGDATA sanity, shm size, and the exact command the
container runs. Paste the whole output back.
"""
import subprocess, json, sys

def sh(cmd, timeout=120):
    return subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=timeout)

def out(label, res, limit=4000):
    print(f"\n===== {label} =====")
    txt = (res.stdout or res.stderr or "").strip()
    print(txt[:limit] if txt else "(empty)")

r = sh("docker ps -a --filter name=pat-postgres --format '{{.ID}} | {{.Status}} | {{.Image}}'")
out("container state", r)

r = sh("docker inspect pat-postgres")
try:
    d = json.loads(r.stdout)[0]
    st = d["State"]
    print("\n===== inspect: lifecycle =====")
    print(json.dumps({
        "Running": st["Running"], "Status": st["Status"], "ExitCode": st["ExitCode"],
        "OOMKilled": st["OOMKilled"], "Error": st["Error"],
        "StartedAt": st["StartedAt"], "FinishedAt": st["FinishedAt"],
        "RestartCount": st["RestartCount"],
    }, indent=1))
    hc = d["Config"].get("Healthcheck")
    print("healthcheck:", json.dumps(hc))
    shm = d["HostConfig"].get("ShmSize")
    print(f"ShmSize: {shm} ({int(shm)/1024/1024:.0f} MB)" if shm else "ShmSize: ?")
    ml = d["HostConfig"].get("Memory")
    print(f"Memory limit: {ml} ({(ml or 0)/1024/1024/1024:.1f} GB)")
    print("LogPath:", d["LogPath"])
except Exception as e:
    print("inspect parse failed:", e)

out("FULL docker logs (all)", sh("docker logs pat-postgres 2>&1"), 12000)

out("kernel OOM evidence (last 20)", sh(
    "dmesg 2>/dev/null | grep -iE 'oom|killed process|out of memory' | tail -20 "
    "|| journalctl -k --since '2 hours ago' 2>/dev/null | grep -iE 'oom|killed' | tail -20"))

out("host memory now", sh("free -m"))

out("PGDATA contents + ownership", sh(
    "docker run --rm -v xauusd_pat-pgdata:/pgd alpine sh -c "
    "'ls -ld /pgd; ls /pgd | head; echo PG_VERSION=$(cat /pgd/PG_VERSION 2>/dev/null); "
    "echo postmaster.pid:; cat /pgd/postmaster.pid 2>/dev/null | head -3; "
    "ls /pgd/log 2>/dev/null | tail -2; echo; echo ---auto.conf---; cat /pgd/postgresql.auto.conf'"))

out("pg_isready direct", sh("docker exec pat-postgres pg_isready -U pat_admin -d predictatrade; echo rc=$?"))

out("process tree inside container", sh("docker top pat-postgres 2>&1 | head -8"))

print("\n== paste ALL of the above back ==")