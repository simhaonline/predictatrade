---
name: release-gates
description: "Final promotion gates with GO/NO-GO decision."
---

# release-gates

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Promotion Ladder
research -> backtest -> OOS -> replay -> paper -> shadow -> signals -> limited execution

## Workflow
1. Collect build/lint/unit/integration/E2E, migrations/backup/restore, security/SBOM/signing evidence.
2. Deterministic/golden/parity, exit-aware cost backtest, OOS/calibration/sample sufficiency.
3. Gate latency/fail-closed, delivery reliability, broker qualification, ledger reconciliation.
4. Windows/MT, observability, compliance activation evidence.
5. Traceability PASS/FAIL/BLOCKED; GO only when all P0 gates pass.

## Docker-First Rules
- ALL services in Docker containers via docker compose.
- NEVER use systemd services.
- Build: docker compose build <srv> && docker compose up -d <srv>
- Logs: docker compose logs -f <srv>
- Status: docker compose ps

## Auto-Push Rule
After every code change: git add -A, git commit -m "...", git push origin main.

## Time Zone
Internal time truth is UTC. Broker time (UTC+3) kept as reference only.

## Output Contract
SOW sections, files examined/changed, tests/checks + results, unresolved risks, next action.
