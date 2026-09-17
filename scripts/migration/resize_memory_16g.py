#!/usr/bin/env python3
"""Right-size compose mem_limits for the 16GB new host — service-aware rewrite.

Run ON THE NEW HOST:  python3 scripts/migration/resize_memory_16g.py
Idempotent: maps every container's mem_limit by container_name block.
Verifies compose config after writing; refuses to write on parse failure.
"""
import re, sys

TARGET = {
    "pat-postgres":     "12g",
    "pat-realtime":     "2g",
    "pat-valkey":       "1g",
    "pat-backtest":     "1g",
    "pat-prometheus":   "1g",
    "pat-control":      "1g",
    "pat-control-b":    "1g",
    "pat-frontend":     "1g",
    "pat-grafana":      "768m",
    "pat-backup-sync":  "512m",
    "pat-live-terminal":"512m",
    "pat-nginx":        "640m",
    "pat-mail-relay":   "512m",
    "pat-ntfy":         "256m",
    "pat-watchdog":     "256m",
    "pat-discord-bot":  "512m",
    "pat-status":       "512m",
    "pat-mail-spool":   None,  # volumes have no mem_limit
}

COMPOSE = "docker-compose.yml"
src = open(COMPOSE).read()

# Walk service blocks: container_name: pat-X ... next mem_limit within same block
def rewrite(text):
    lines = text.split("\n")
    current = None
    out = []
    changed = 0
    for ln in lines:
        m = re.match(r"\s*container_name:\s*(pat-[a-z0-9-]+)", ln)
        if m:
            current = m.group(1)
        if "mem_limit:" in ln and current in TARGET and TARGET[current]:
            new = re.sub(r"mem_limit:\s*\S+", f"mem_limit: {TARGET[current]}", ln)
            if new != ln:
                changed += 1
            out.append(new)
        else:
            out.append(ln)
    return "\n".join(out), changed

new_text, n = rewrite(src)
print(f"rewrote {n} mem_limit lines")

# validate compose before writing
import subprocess
open("/tmp/compose_candidate.yml", "w").write(new_text)
r = subprocess.run(["docker","compose","--env-file","infra/env/.env","-f","/tmp/compose_candidate.yml","config","-q"],
                   capture_output=True, text=True)
if r.returncode != 0:
    print("COMPOSE VALIDATION FAILED — not written:", r.stderr[-500:])
    sys.exit(1)

open(COMPOSE, "w").write(new_text)
print("COMPOSE-OK — written. Summary:")
for svc, v in TARGET.items():
    if v:
        print(f"  {svc:22s} {v}")