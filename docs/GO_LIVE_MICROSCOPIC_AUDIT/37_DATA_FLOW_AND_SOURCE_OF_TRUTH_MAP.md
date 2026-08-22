# 37 — Data Flow & Source-of-Truth Map

```
                    ┌──────────────── SOURCE OF TRUTH REGISTRY ───────────────┐
                    │ Market truth ......... market.ticks/candles (Timescale) │
                    │ Signal truth ......... trading.signals (+candidates)    │
                    │ Finance truth ........ referral.commission_ledger (0)   │
                    │                       billing.payments/invoices (1 fake)│
                    │ Identity truth ....... iam.users/sessions/roles         │
                    │ Device/license truth . licensing.*                      │
                    │ Ephemeral ............ Valkey (cache/dedup/cooldown)    │
                    └─────────────────────────────────────────────────────────┘

EA/Agent ──WS(UNAUTH)──▶ Go engine ──▶ Timescale + Valkey
                             │
                             ├─▶ /api/v1/signals (NO AUTH) ◀── Next.js dashboards (poll)
                             └─▶ AgentHub broadcast ──▶ Windows Agent ──pipe──▶ EA orders
                                          ▲                              │
                                          └── CLOSE_ACK dropped ✗        │
Control plane (NestJS) ── SQL ─▶ iam/licensing/billing/referral/audit schemas
      ▲ Next.js admin/user pages (REST, JWT cookie)
      ✗ billing webhook: unverified POST → can create financial rows (proven)

Contradiction traces performed (§122):
- UI "Operational" engine tile ← tautological check (admin dashboard:108) — fixed-truth source = HTTP 200 only.
- DB prob=0.450166 on ALL NO-TRADE rows ← seed sigmoid default — traced to consumer.go seeder.
- ACTIVE ELITE subscription with zero payment code ← fabricated webhook row signature_verified=false.
- "12 hard gates" panel ← static dots; real count evaluated = 12 of 14 registered (2 unreachable).
- Signals visible without login ← REST entitlement bypass at Go plane (design gap, not cache).
- 553 candles open/low=0 but COMPLETE ← aggregator seeding bug, not provider data.
```

Every contradiction was traced to a single root cause; none remain unexplained. Cross-layer field consistency for sampled signals (engine↔DB↔API): MATCH except frontend-invented derivatives listed in 24.
