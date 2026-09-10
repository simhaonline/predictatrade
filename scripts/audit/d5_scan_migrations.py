#!/usr/bin/env python3
"""Audit Domain 5 scanner: migrations, timescale, compose, nginx, infra."""
import re, os, sys, json, collections

ROOT = "/srv/predictatrade/xauusd"
MIG = os.path.join(ROOT, "database/migrations")

# ---------- (a) migration table inventory ----------
tables = collections.defaultdict(list)   # qualified name -> [(mig, line, kind)]
indexes = collections.defaultdict(list)  # qualified table -> [(mig, indexname, columns)]
fks = collections.defaultdict(list)      # table -> fk info
timescale = []                           # hypertable/compression/retention lines
for fn in sorted(os.listdir(MIG)):
    if not fn.endswith(".sql"): continue
    path = os.path.join(MIG, fn)
    text = open(path, encoding="utf-8", errors="replace").read()
    for m in re.finditer(r'CREATE\s+(UNLOGGED\s+)?TABLE\s+(IF\s+NOT\s+EXISTS\s+)?((?:\w+)\.(?:\w+))', text, re.I):
        kind = "UNLOGGED" if m.group(1) else "normal"
        tables[m.group(3).lower()].append((fn, text[:m.start()].count("\n")+1, kind))
    # indexes
    for m in re.finditer(r'CREATE\s+(UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s+ON\s+((?:\w+)\.(?:\w+))\s*(USING\s+\w+\s*)?\(?([^;]*?)\)?\s*;', text, re.I | re.S):
        indexes[m.group(3).lower()].append((fn, m.group(2), " ".join(m.group(5).split())))
    # timescale statements
    for m in re.finditer(r'(?i)(create_hypertable|add_retention_policy|add_compression_policy|compress hypertable|set_chunk_time_interval|create_materialized_view|add_continuous_aggregate_policy)', text):
        line_no = text[:m.start()].count("\n")+1
        line = text.splitlines()[line_no-1].strip()
        timescale.append((fn, line_no, line[:160]))
    # FKs
    for m in re.finditer(r'(?i)FOREIGN\s+KEY\s*\(([^)]+)\)\s+REFERENCES\s+((?:\w+)\.(?:\w+))', text):
        fks[fn].append((m.group(1), m.group(2)))

print("== DUPLICATE TABLE CREATIONS ==")
for t, occ in sorted(tables.items()):
    if len(occ) > 1:
        print(f"  {t}: {occ}")
print(f"\n== TABLE COUNT: {len(tables)} distinct, {sum(len(v) for v in tables.values())} creations ==")

print("\n== UNLOGGED TABLES ==")
for t, occ in sorted(tables.items()):
    for mig, ln, kind in occ:
        if kind == "UNLOGGED": print(f"  {t} @ {mig}:{ln}")

print("\n== TIMESCALE STATEMENTS (all) ==")
for fn, ln, line in timescale:
    print(f"  {fn}:{ln}: {line}")

print("\n== FK COUNT PER MIG (files with 0 FKs) ==")
nofk = [fn for fn in sorted(os.listdir(MIG)) if fn.endswith(".sql") and not fks.get(fn)]
print("  " + ", ".join(nofk))