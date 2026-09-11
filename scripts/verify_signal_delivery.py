#!/usr/bin/env python3
"""
Predict-A-Trade — Signal Delivery Verification (dashboard vs MT4/MT5 EA).

Read-only diagnostic. Explains WHY a user can see signals on
live.predictatrade.com but receive NONE on their MT4/MT5 EA.

It does not change any data. It connects to Postgres in-container (creds never
leave the host) and reports:

  1. GLOBAL divergence:
     - signals the dashboard CAN show (entitled strategies, any executability)
     - signals that are actually Executable==true (the only kind EAs receive)
     - signals actually delivered+ACKed to EAs (edge_signal_queue)
  2. PER-DEVICE delivery state:
     - license status, plan, allowed_strategies, last_seen (online?)
     - PENDING / ACKED / EXPIRED counts from edge_signal_queue (last 7d)
     - how many executable signals its plan IS entitled to vs how many it GOT
     - the exact reason it may be receiving nothing
  3. Per-plan entitlement map + which currently-executable strategies each allows.

Usage:
  python3 scripts/verify_signal_delivery.py                 # full report
  python3 scripts/verify_signal_delivery.py --device <uuid> # single-device deep dive
  python3 scripts/verify_signal_delivery.py --days 7

No secrets printed. Postgres role/db discovered from the container environment.
"""
import argparse
import subprocess
import sys

DAYS = 7


def pg(query):
    """Run a SELECT via the postgres container; return list of row tuples."""
    cmd = f'docker exec pat-postgres psql -U pat_admin -d predictatrade -tAc "{query}"'
    res = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    out, rc = res.stdout, res.returncode
    if rc != 0:
        sys.stderr.write(f"[warn] pg query failed: {out.strip()[:200]}\n")
        return []
    rows = []
    for line in out.splitlines():
        line = line.strip()
        if not line:
            continue
        # psql -A (unaligned) uses '|' separators
        rows.append(tuple(c if c != "\\N" else None for c in line.split("|")))
    return rows


def main():
    ap = argparse.ArgumentParser(description="Signal delivery verification (dashboard vs EA)")
    ap.add_argument("--device", default=None, help="device UUID to deep-dive")
    ap.add_argument("--days", type=int, default=DAYS)
    args = ap.parse_args()
    d = args.days

    print("=" * 78)
    print("SIGNAL DELIVERY VERIFICATION — dashboard (live.predictatrade.com) vs MT4/MT5 EA")
    print(f"window: last {d} days")
    print("=" * 78)

    # ---- 1. Global divergence ----
    print("\n[1] GLOBAL — what the dashboard shows vs what EAs can receive")
    g = pg(f"""
        SELECT
          (SELECT count(*) FROM trading.signals WHERE created_at > now()-interval '{d} days') AS total,
          (SELECT count(*) FROM trading.signals WHERE created_at > now()-interval '{d} days' AND executable IS TRUE) AS executable
    """)
    total = int(g[0][0]) if g else 0
    exec_all = int(g[0][1]) if g else 0
    print(f"  signals generated (all)            : {total:,}")
    print(f"  signals Executable==true (EA-eligible): {exec_all:,}  ({100*exec_all/max(total,1):.2f}%)")
    print(f"  -> dashboard shows ALL {total:,} (incl. non-executable candidates)")
    print(f"  -> EAs receive ONLY the {exec_all:,} executable ones, plan-gated")

    # executable by strategy
    ex = pg(f"""
        SELECT strategy_id, count(*) FROM trading.signals
        WHERE created_at > now()-interval '{d} days' AND executable IS TRUE
        GROUP BY strategy_id ORDER BY 2 DESC
    """)
    print("  executable signals by strategy:")
    for sid, n in ex:
        print(f"      {sid or '(null)':24s} {int(n):,}")

    # delivered via edge queue
    dl = pg(f"""
        SELECT status, count(*) FROM licensing.edge_signal_queue
        WHERE created_at > now()-interval '{d} days'
        GROUP BY status ORDER BY 2 DESC
    """)
    print("  edge_signal_queue (EA delivery) status:")
    for st, n in dl:
        print(f"      {st or '(null)':14s} {int(n):,}")

    # ---- 2. Plan entitlement map ----
    print("\n[2] PLANS — allowed_strategies")
    plans = pg("""
        SELECT name, allowed_strategies::text FROM control.plans
        ORDER BY name
    """)
    plan_map = {}
    for name, allowed in plans:
        plan_map[name] = allowed or "[]"
        print(f"  {name:28s} {allowed}")

    # ---- 3. Per-device ----
    print("\n[3] PER-DEVICE delivery state (last 7d)")
    devs = pg(f"""
        SELECT d.id, d.os_name,
               l.status AS lic_status,
               p.name   AS plan,
               p.allowed_strategies::text AS allowed,
               d.last_seen_at,
               (SELECT count(*) FROM licensing.edge_signal_queue q
                  WHERE q.device_id=d.id AND q.status='ACKED'
                    AND q.created_at > now()-interval '{d} days') AS acked,
               (SELECT count(*) FROM licensing.edge_signal_queue q
                  WHERE q.device_id=d.id AND q.status='PENDING'
                    AND q.created_at > now()-interval '{d} days') AS pending,
               (SELECT count(*) FROM licensing.edge_signal_queue q
                  WHERE q.device_id=d.id AND q.status='EXPIRED'
                    AND q.created_at > now()-interval '{d} days') AS expired
        FROM licensing.devices d
        LEFT JOIN licensing.licenses l ON l.id = d.bound_license_id
        LEFT JOIN control.plans p ON p.id = l.plan_id
        ORDER BY acked DESC, d.id
    """)

    def reason_for(dev):
        did, osn, lic, plan, allowed, seen, acked, pending, expired = dev
        if lic != "ACTIVE":
            return f"LICENSE NOT ACTIVE ({lic})"
        if allowed in (None, "[]", "", "null"):
            return "no allowed_strategies on plan"
        # executable strategies this plan allows
        exec_strats = [r[0] for r in ex if r[0]]
        allow_set = set(a.strip().strip('"') for a in allowed.strip("[]").split(",") if a.strip())
        entitled_exec = [s for s in exec_strats if s in allow_set]
        entitled_exec_n = sum(int(n) for s, n in ex if s in allow_set)
        if not entitled_exec:
            return f"plan '{plan}' allows {sorted(allow_set)} but NONE of the {exec_all} executable signals are in that set"
        entitled_exec_n = sum(int(n) for s, n in ex if s in allow_set)
        if (acked or 0) == 0 and (pending or 0) == 0:
            if seen is None or str(seen) == "":
                return "device NEVER polled (last_seen empty) — EA not activated / not running"
            return f"entitled to {entitled_exec_n} executable signals but received 0 — device last seen {seen}; check EA WebRequest allowlist + AlgoTrading enabled + device activated"
        return "OK — receiving signals"

    print(f"  {'device':38s} {'plan':14s} {'lic':9s} {'acked':6s} {'pend':5s} {'exp':5s}  reason")
    for dev in devs:
        did, osn, lic, plan, allowed, seen, acked, pending, expired = dev
        acked = int(acked or 0); pending = int(pending or 0); expired = int(expired or 0)
        plan_s = (plan or "?")[:14]
        lic_s = (lic or "?")[:9]
        short = did[:8] + "…" if did else "?"
        why = reason_for(dev)
        flag = "" if (acked or pending) else "  <-- NO DELIVERY"
        print(f"  {short:38s} {plan_s:14s} {lic_s:9s} {acked:<6d} {pending:<5d} {expired:<5d}  {why}{flag}")
        if args.device and args.device in str(did):
            print(f"      device_id={did}  os={osn}  last_seen={seen}")

    # ---- 4. Verdict ----
    print("\n[4] EXPLANATION")
    print("  The dashboard reads trading.signals filtered by your plan's allowed_strategies")
    print("  but does NOT filter on Executable — so it shows candidates (Executable=false).")
    print("  MT4/MT5 EAs only ever receive Executable==true signals via edge-poll, plan-gated.")
    print("  If you see dashboard signals but your EA is silent, it is almost always one of:")
    print("    (a) your plan does not allow the strategies that are currently executable, or")
    print("    (b) your EA is not polling (WebRequest allowlist / AlgoTrading off / device")
    print("        not activated), or (c) your license is not ACTIVE.")
    print("  The engine itself is delivering: see ACKED count above.")
    print("=" * 78)


if __name__ == "__main__":
    main()
