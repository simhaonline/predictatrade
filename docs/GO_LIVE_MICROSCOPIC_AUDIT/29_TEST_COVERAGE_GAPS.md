# 29 — Test Coverage Gaps

## Executed during audit

| Suite | Result |
|---|---|
| research pytest | **127 passed** (2.8s) |
| go vet ./internal/... ./pkg/... | clean |
| go test (features, strategy, engines, pkg/math, pkg/strategy) | all ok |

Repo claims (MANIFEST): 24/24 Go packages, 70 frontend tests — full-suite runs not repeated in audit window; targeted subset green.

## Production-critical logic with NO tests

1. Emergency-stop / kill-switch paths (code doesn't exist to test).
2. Daily-loss 5% boundary matrix (§22) — module unwired.
3. Billing webhook verification/idempotency/replay (no PSP integration).
4. Commission end-to-end (engine tested in isolation vs WRONG rate table — spec≠seed divergence).
5. Payout request/approve (would fail at runtime 42703 — proves tests never ran against schema).
6. Entitlement-filtered WS delivery (filter untestable: never populated).
7. Outbox dispatcher / expiry sweeper / delivery acks (dead code, zero coverage).
8. Multi-instance WebSocket sequencing (no test harness).
9. Candle aggregator zero-open bug (553 corrupted rows prove gap — property test missing).
10. MT4/MT5 EA execution idempotency under duplicate command delivery (EXTERNAL BLOCKER: needs terminal).

## Coverage theater found

- `control/spec/security-validation.spec.ts:21-26,35-38` asserts literals about themselves.
- Frontend tautological `engineAlive` check passes any 200.

## Golden dataset (§72)

Deterministic fixtures exist for indicator parity (research) but **no golden end-to-end snapshot fixture** (market state → expected indicators/regime/score/geometry/gate) is wired into CI. Required before GO per SOW.
