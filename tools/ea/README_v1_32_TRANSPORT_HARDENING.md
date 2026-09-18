# EA v1.32 — HTTP transport hardening (release notes)

**Files**: `mql/mt4/PredictATrade_MT4.mq4`, `mql/mt5/PredictATrade_MT5.mq5`
**Date**: 2026-09-18 · **Behavior change: NONE** — telemetry/transport only.

## What changed (all client-side, server untouched)

1. **Actionable failure text** — every transport failure now prints WHY:
   - `URL not in WebRequest allowlist — add https://api.predictatrade.com`
   - DNS resolution failed / connection refused / SSL-TLS error / timeout
   - `HTTP 1003`-style codes (WinINET artifacts some terminals propagate) are
     now explicitly recognized as transport-layer failures with guidance
     instead of a meaningless number.
2. **Automatic transport retries** — 3 attempts, linear backoff (0.5s, 1.0s).
   Applies to: device activation, signed edge-poll, Bearer ingest, edge-ack.
   Server statuses (401/404/409/…) are NOT retried — callers handle them.
3. **Re-sign on retry** — each signed-POST retry uses a fresh timestamp/nonce
   (nonce replay protection would reject a reused signature).
4. **Version bump** to v1.32 (single `PAT_EA_VERSION` source of truth).

## Why the errors you saw happened

- **MT4 `HTTP -1`**: WebRequest never reached the server. Causes, in order of
  likelihood: URL missing from that terminal's WebRequest allowlist; DNS;
  TLS (old Windows); internet blip. The old build printed only "HTTP -1";
  v1.32 prints the mapped cause + fix. Retries ride out transient blips.
- **MT5 `HTTP 1003`**: not a real HTTP status — a terminal transport artifact
  (WinINET layer propagation on some MT5 builds). v1.32 maps it and retries.

## Server side verified healthy (2026-09-18)

Both client endpoints verified end-to-end through the real Plesk edge:
- `POST https://api.predictatrade.com/api/v1/devices/activate` → 404
  "License not found" for an unknown key (correct; 200 with a real key)
- `POST https://api.predictatrade.com/ingest/agent` (Bearer) → 401
  "device authentication required" for a bad token (correct)
So the MT4 `-1` / MT5 `1003` errors were terminal-side, not server-side.

## Operator checklist when transport errors appear (either terminal)

1. Tools → Options → Expert Advisors → check "Allow WebRequest" AND the list
   contains exactly `https://api.predictatrade.com`
2. Confirm internet/DNS on the terminal (`ping api.predictatrade.com`)
3. Update Windows (old TLS stacks fail against modern TLS terminators)
4. Re-attach the EA; v1.32 retries automatically and prints the real cause
5. If errors persist >15 min with clean allowlist/DNS/TLS → send the Experts
   log excerpt (v1.32 names the exact failure reason)

## Recompile

Same procedure as `docs/guides/EA_RECOMPILE_RUNBOOK.md`: fresh repo copy →
apply patch 0001 → F7 (0 errors) → re-attach. Confirm `Predict-A-Trade EA
v1.32` in the Experts log.