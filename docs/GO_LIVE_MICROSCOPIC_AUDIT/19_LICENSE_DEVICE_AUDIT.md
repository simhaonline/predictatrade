# 19 — License / Device Activation Audit

## Lifecycle (code-audited)

`License Key → POST /devices/activate (fingerprint+account+terminal bind) → device_secret (AES-GCM at rest) → refresh rotation (family revocation on reuse) → 45s session leases via heartbeat → admin revoke cascade`.

DB state: licenses=1, devices=2, activation sessions schema present (`008_device_activation_sessions.sql`). Heartbeat returns independent connection/auth/license/device/session states — honest status surface.

## Verified strengths

- Device secret never stored plaintext; nonce replay table (`licensing.request_nonces`, ±30s window, TTL 120s) implemented with constant-time compare.
- Refresh reuse detection revokes token family + credentials + leases.
- Admin revoke cascades leases+credentials correctly in device-auth service.

## Defects

| ID | Sev | Finding |
|---|---|---|
| 19-1 | P1 | **IDOR**: `POST /licensing/devices/:id/revoke`, `PUT .../heartbeat` (legacy routes) have no ownership check — any JWT user can revoke/flood another user's device. |
| 19-2 | P1 | `verifyRequestSignature()` (the signed-request protocol incl. replay protection) is **never invoked** by any controller/guard. |
| 19-3 | P2 | Access tokens minted at activation/refresh are dead (never persisted/verifiable). |
| 19-4 | P2 | License expiry enforced only at activation; no reaper flips EXPIRED or purges nonces/leases. |
| 19-5 | P2 | Max-device/max-MT-account checks racy (read-check-insert, no lock/tx). |
| 19-6 | P2 | Duplicate divergent revoke semantics between licensing and device-auth services. |

Heartbeat also persists client-supplied account balances/equity/P&L unauthenticated-by-signature (spoofable; ties to F-003) and echoes client `trading_mode` as authoritative-looking state.

**Status: PARTIAL.** Core crypto/lifecycle sound; enforcement perimeter has holes that matter only once the network is locked down (18).
