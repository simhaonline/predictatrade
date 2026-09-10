#!/usr/bin/env python3
"""D5 part 2: unqualified tables, inline FKs, hot-path indexes, migration order doc."""
import re, os, collections

ROOT = "/srv/predictatrade/xauusd"
MIG = os.path.join(ROOT, "database/migrations")

unq = collections.defaultdict(list)
inline_fk = collections.defaultdict(list)
hot = collections.defaultdict(list)
for fn in sorted(os.listdir(MIG)):
    if not fn.endswith(".sql"): continue
    text = open(os.path.join(MIG, fn), encoding="utf-8", errors="replace").read()
    # unqualified CREATE TABLE (no dot before the paren/space)
    for m in re.finditer(r'(?i)CREATE\s+(UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?!\s*")(?!IF)(\w+)\s*\(', text):
        name = m.group(2)
        if "." not in name and name.lower() not in ("if",):
            unq[name].append((fn, text[:m.start()].count("\n")+1))
    # inline REFERENCES
    for m in re.finditer(r'(?i)REFERENCES\s+((?:\w+)\.(?:\w+))', text):
        inline_fk[fn].append(m.group(1))
    # hot path index targets
    for m in re.finditer(r'(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s+ON\s+((?:trading|audit)\.(?:signals|edge_signal_queue|trade_results|audit_events|signal_performance))\s*\(?([^;]*?);', text, re.S):
        hot[m.group(3).lower()].append((fn, text[:m.start()].count("\n")+1, m.group(2), " ".join(m.group(4).split())[:120]))

print("== UNQUALIFIED CREATE TABLE ==")
for t, occ in sorted(unq.items()):
    print(f"  {t}: {occ}")

print("\n== HOT PATH INDEXES ==")
for t in ("trading.signals", "trading.edge_signal_queue", "trading.trade_results", "audit.audit_events", "trading.signal_performance"):
    print(f"  -- {t} --")
    for fn, ln, name, cols in hot.get(t, []):
        print(f"    {fn}:{ln}: {name} ({cols})")

print("\n== INLINE FK per file (count) ==")
for fn in sorted(inline_fk):
    print(f"  {fn}: {len(inline_fk[fn])}")
print("files with ZERO FK of any kind:", sorted(set(f for f in os.listdir(MIG) if f.endswith('.sql')) - set(inline_fk))[:20], "...")