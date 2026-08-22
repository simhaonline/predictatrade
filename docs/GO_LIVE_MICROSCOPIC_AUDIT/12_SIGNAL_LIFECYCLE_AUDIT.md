# 12 — Signal Lifecycle Audit

## Reference & uniqueness (DB-verified)

- `ID` = UUIDv4 per creation; `SignalReference = PAT-XAU-YYYYMMDD-%06d` from Postgres sequence `trading.signal_seq` — 10,402/10,402 non-empty refs unique.
- **6,451 signals (Aug 18–21) have empty reference** (pre-fix era) — traceability hole for history; sequence fetch errors ignored at 4 call sites (`main.go:1175,1335,1379,1601`) → on DB blip refs degrade to 0-collisions silently.

## Duplicate prevention

Canonical fingerprint sha256(symbol|strategy|ver|direction|entry@2dp|SL@2dp|BOS-min|CHoCH-min|bar-min) claimed via Valkey SETNX 30min TTL. **Fail-open when Valkey down** and publicly-writable Valkey (F-002) means dedup is bypassable — §86 requirement not met under adversarial/degraded conditions.

## TTL / expiry

ExpiresAt = CreatedAt + strategy ExpiryMinutes (3–240) or +15 NO-TRADE. Enforcement: resume endpoint filters expired; **no sweeper** (`MarkExpired` dead) → expired CONFIRMED signals linger in DB/API until natural filtering. Expired cannot re-execute via resume (filter verified).

## Delivery trace

Design chain signal→outbox→deliveries→receipts→acks exists in schema, but runtime shows: outbox 2,697 PENDING (dispatcher dead); deliveries/receipts/ledger = 0 rows; EA CLOSE_ACK/SLIPPAGE_EVENT logged-and-dropped (`agent_provider.go:519-533`). **End-to-end delivery acknowledgment is NOT operational** — a subscriber device may never receive a signal with no server-side knowledge (P1 for a signal-selling product).

## Reconciliation sample (live API vs DB)

Field-by-field match between `GET /api/v1/signals` payload and `trading.signals` row for sampled IDs (RawScore 39.67 / prob 0.596283 / geometry equal). WS envelope carries event_id/stream_id/per-client sequence/schema_version — but agent-bound envelopes omit `sequence`, and UI clients can't pass entitlement filter (see 25).

## Lifecycle state machine observed

DETECTED→(candidates APPROVED/REJECTED/ADVISORY persisted w/ RejectionGate)→CONFIRMED/blocked Grade=Blocked + PrimaryBlocker persisted; cooldown audit + duplicate audit rows written. Reason codes present for every NO-TRADE sampled (`NT_REGIME_MISMATCH`, `STRATEGY_COOLDOWN_ACTIVE`, `DUPLICATE_SIGNAL`) — §20 satisfied at persistence level.
