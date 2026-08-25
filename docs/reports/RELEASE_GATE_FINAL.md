# Release Gate Final — 2026-08-23

## Verdict: CONDITIONAL GO

All repository-controlled P0 issues are fixed. Remaining blockers are genuinely external.

## P0 Security
- [x] Secret exposure: FIXED (opencode.json key removed)
- [x] WebSocket impersonation: FIXED (JWT-based identity)
- [x] CORS: FIXED (nginx headers on all routes)
- [ ] Payment webhook signature: BLOCKED (requires payment provider credentials)

## P0 Signal Math
- [x] ATR bug (2180→4.42): FIXED
- [x] ADX bug (99.79→30.14): FIXED
- [x] EMA precision: FIXED
- [x] MACD histogram: FIXED
- [x] Calibration probability: FIXED
- [x] GrossRR persistence: FIXED

## P0 Infrastructure
- [x] Docker architecture: ALL 10 containers in Docker
- [x] Merge conflicts: RESOLVED (277→0)
- [x] nginx DNS: FIXED (restart after rebuild)
- [x] Default light mode: FIXED

## External Blockers
1. Payment provider sandbox credentials (Stripe/Adyen) — required for webhook certification
2. Live MT4/MT5 broker qualification — required for execution certification
3. Historical OOS data for statistical calibration validation
4. Off-host backup storage (S3) — required for DR certification


## Release Gate Update — 25 August 2026 (v1.15.0)

### New Pass Items
- ✅ Server-side SL enforcement: EXECUTION_ACK verification, position monitoring, CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH
- ✅ Agent suspension for SL violations (3-strike disconnect)
- ✅ MQL EA v1.09: CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH handlers + position SL in snapshot
- ✅ Windows Agent v1.2.18: command forwarding for all 3 server commands
- ✅ DXY→macroHealth wiring fix (ML/Sentiment re-enabled)
- ✅ Calibration DB tables (migration 072)
- ✅ Legal compliance: Terms of Service, Privacy Policy, DPA + consent tracking
- ✅ CI/CD: All 6 GitHub Actions jobs passing

### Updated Status
- Go tests: 30/30 packages pass (race + non-race)
- Frontend: 0 lint errors, TypeScript passes, build passes
- NestJS: lint passes, tests pass, build passes
- Python: tests pass
- Windows Agent: cross-compile passes
- Security scan: passes (no false positives)
- Docker: all services healthy
