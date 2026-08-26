---
name: security-supply-chain
description: "Threat model, secret scanning, supply chain security."
---

# security-supply-chain

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Workflow
1. Threat-model web/control/realtime/Windows/MQL/broker/payment/referral/update surfaces.
2. Review auth/session/MFA/RBAC/tenant and privileged admin paths.
3. Run secret/SAST/dependency/license checks and SBOM.
4. Validate signal/update signing, key separation/revocation and anti-replay.
5. Review input validation, rate/abuse and payment/referral/commission/payout fraud controls.
6. Review CI/release provenance/checksums/signatures.

## Known Issues (full-audit.md)
- F1 (HIGH): Access token in JS-readable cookie + window.__ACCESS_TOKEN__
- F2 (HIGH): No CSP/HSTS/X-Frame-Options
- W3 (HIGH): No signed IPC/WS, no replay protection
- H1 (HIGH): JWT secret dual-source

## Validate
Critical/high findings resolved or release-blocking. No production credentials for MCP/agents.
