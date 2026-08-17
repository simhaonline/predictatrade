---
name: mt4-mt5-windows
description: Implement/review Go Windows Agent and lightweight MQL4/MQL5 licensing, signed signal stream, IPC, broker mapping, execution guards and signed updates.
---

# mt4-mt5-windows

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Keep predictive/risk truth server-side.
2. Validate license/device/account/entitlement before delivery/execution.
3. Secure stream with heartbeat, TTL, sequence, replay and idempotency protection.
4. Version local IPC between Agent and MT4/MT5.
5. Discover broker symbols/specifications and handle disconnect/reject/partial/manual intervention.
6. Maintain local audit/correlation IDs and signed updater/manifest/checksum/rollback.

## Validate
MT4/MT5 clean compile, Windows reconnect/restart, revocation propagation and duplicate-order protection. Never embed server/private signing credentials.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
