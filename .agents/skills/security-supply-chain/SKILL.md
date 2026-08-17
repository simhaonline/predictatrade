---
name: security-supply-chain
description: Perform threat modeling, IAM/session/tenant, secrets, SAST/dependency/SBOM, signing/updater, protocol, financial-abuse and release security.
---

# security-supply-chain

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Threat-model web/control/realtime/Windows/MQL/broker/payment/referral/update surfaces.
2. Review auth/session/MFA/RBAC/tenant and privileged admin paths.
3. Run secret/SAST/dependency/license checks and SBOM.
4. Validate signal/update signing, key separation/revocation and anti-replay.
5. Review input validation, rate/abuse and payment/referral/commission/payout fraud controls.
6. Review CI/release provenance/checksums/signatures.

## Validate
Critical/high findings resolved or release-blocking. No broad production credentials for MCP/agents; no private signing keys in repo/client.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
