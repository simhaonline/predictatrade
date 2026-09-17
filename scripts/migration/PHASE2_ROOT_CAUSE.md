# PHASE 2 restore ROOT CAUSE + permanent fixes — final (2026-09-17)

## The root cause (proven RED→GREEN on an isolated volume)

The foreground boot (`foreground_postgres.py`) finally surfaced the real error:

```
FATAL:  recovery aborted because of insufficient parameter settings
DETAIL: max_connections = 80 is a lower setting than on the primary server, where its value was 100.
```

**Archive recovery requires standby params ≥ primary values.** The memory
right-sizing (Fix 2) set `max_connections = 80` for the 16 GB host, but the
primary ran `max_connections = 100`. Postgres refuses to enter archive recovery
with a lower value — by design, to prevent WAL replay incompatibilities. This
was the cause of the silent restart loop on the new host (ExitCode 1, ~2 s
lifetime, no useful logs through docker logs).

## Regression test (isolated volume, both directions verified live)

- RED: conf with `max_connections = 80` + recovery.signal + bridge →
  `FATAL: recovery aborted because of insufficient parameter settings`
- GREEN: same volume, conf with `max_connections = 100`,
  `max_locks_per_transaction = 1024`, `max_wal_senders = 10`,
  `max_worker_processes = 35` (the old host's values) →
  `database system is ready to accept connections`, then
  **238 tables / 30,921,448 market.ticks** restored.

Primary values (from the old host, 2026-09-17):
`max_connections=100, max_locks_per_transaction=1024,
max_prepared_transactions=0, max_wal_senders=10, max_worker_processes=35`.

## Rule going forward (never again)

1. **Never lower standby-critical GUCs below primary values before recovery:**
   `max_connections`, `max_prepared_transactions`, `max_locks_per_transaction`,
   `max_wal_senders`, `max_worker_processes`. Check the restored
   `pg_control`/primary values first (`pg_controldata` / `pg_settings` on the
   source). Memory right-sizing on a standby/restore must NOT touch these.
2. **Conf writes into a volume go through tar-stream or hex-stream only** —
   never printf/heredoc paste (Fix 3 class) and never with host-shell expansion.
3. **Every restore procedure must be validated on an isolated throwaway volume
   with the REAL artifacts before it runs on the target** — this caught the
   permission-denied on staged bridge files (mode 600 + wrong owner) and the
   max_connections violation in one afternoon, with zero risk to prod.
4. **`docker compose stop` ≠ container removal** — `volume rm` needs
   `compose rm -f <svc>` first.
5. **Bridge/WAL files staged into PGDATA must be `chmod 644` + `chown 1000:1000`**
   (archive `cp` runs as the postgres OS user; 600+root = unreadable).
6. **restore_command paths must match the postgres container's mount layout**
   (`/var/lib/postgresql/data/pg_wal_bridge/%f`), not helper-container paths.
7. **When docker logs shows nothing but the entrypoint loop, boot postgres in
   the foreground** (`foreground_postgres.py`) — the raw FATAL prints to the
   terminal and ends the diagnosis in one pass.

## Final restore profile for the 16 GB new host (PHASE 2 — temporary)

```
shared_buffers = 3900MB
max_connections = 100        # MUST match primary (recovery rule); retune AFTER promote
work_mem = 16MB
maintenance_work_mem = 256MB
effective_cache_size = 11700MB
wal_buffers = 16MB
max_wal_size = 2GB / min_wal_size = 256MB
max_locks_per_transaction = 1024
max_wal_senders = 10
max_worker_processes = 35
```

After cutover (server running, promoted, restore_command cleared), the memory
profile can be re-tuned ONLINE via `ALTER SYSTEM SET work_mem/…` + reload —
never by editing auto.conf under a live postmaster, and never by lowering
standby-critical params below primary values while the backup chain is in use.

## Verified proof (isolated volume, real data)

```
tables (non-timescale): 238
market.ticks: 30,921,448
db time: 2026-09-17 (current)
```

Commits: `53b0abc` (finish_phase2.py + hex-stream writes), `b07a439`
(compose-candidate env_file fix), `3457748` (list-args everywhere, tar-stream
conf/signal writes), `fcb3116` (chmod 644 staged bridge), `9b64d0f`
(restore_command path fix + diagnostic bundle on failure), `fcc6496`
(max_connections 100 + standby params — THE root-cause fix).