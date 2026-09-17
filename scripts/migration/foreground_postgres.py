#!/usr/bin/env python3
"""One-shot: run postgres MANUALLY in the foreground inside the container so the
raw failure output appears directly on the terminal (docker logs on the new host
swallows everything — this bypasses that entirely).

Run ON THE NEW HOST:
    python3 scripts/migration/foreground_postgres.py
Watch the terminal: the real error prints there. Ctrl+C when done reading.
If it reaches "ready to accept connections", you are THROUGH — open another
terminal and run python3 scripts/migration/restore_phase2.py to continue.
"""
import subprocess, sys, os, io, tarfile, time

VOL = os.environ.get("PGDATA_VOLUME", "xauusd_pat-pgdata")

print("This drops you into a shell inside a postgres container with the restored")
print("PGDATA mounted at /var/lib/postgresql/data. Then run:")
print("  postgres -D /var/lib/postgresql/data")
print("and the real error prints on your screen. (Ctrl+C / 'exit' when done.)")
print("Extra mounts: bridge at /var/lib/postgresql/data/pg_wal_bridge, helpers in /tmp/mig.\n")

env_args = [
    "-e", "PGDATA=/var/lib/postgresql/data",
    "-e", "POSTGRES_USER=pat_admin",
    "-e", "POSTGRES_PASSWORD=mig-tmp",
    "-e", "POSTGRES_DB=predictatrade",
]
argv = ["docker", "run", "--rm", "-it", "--name", "pat-postgres-fg",
        "--shm-size", "2gb",
        "-v", f"{VOL}:/var/lib/postgresql/data",
        "-v", "/tmp/mig/wal_bridge:/var/lib/postgresql/data/pg_wal_bridge:ro",
        ] + env_args + ["timescale/timescaledb-ha:pg17"]
os.execvp(argv[0], argv)