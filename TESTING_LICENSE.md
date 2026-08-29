# Testing License Key (DEV ONLY)

This key seeds an `ACTIVE` license in the local `licensing.licenses` table and is
provided only to validate the end-to-end agent → control-plane licensing flow.

```
PAT-A1B2C3D4-0001-4000-8000-000000000001
```

## Constraints
- Plan: `FREE`
- `max_devices`: 1
- `max_mt_accounts`: 1
- `allowed_execution_modes`: `SIGNAL_ONLY`
- `allowed_strategies`: `STANDARD_SCALPING`

Use it on **one** terminal/account to confirm the license badge flips to `ACTIVE`.
For production, issue a proper key via the control plane (pat-control) for the device.

## How to use
1. In MT4 and/or MT5, open the `PredictATrade` EA → **Inputs** → `LicenseKey`.
2. Paste the key above (no leading/trailing spaces).
3. Re-attach the EA (or restart the terminal) so the input loads.
4. Enable **Algo Trading**.
5. The agent validates via `POST /api/v1/licensing/validate` and the badge becomes
   `LICENSE ACTIVE`.

## Notes
- The control-plane `/licensing/validate` does **not** require device activation —
  a valid key returns `{"valid":true,"status":"ACTIVE"}` directly.
- If the badge shows `INVALID`/`NOT_FOUND`, the key entered is wrong (typo or
  whitespace). Paste the exact key above to confirm the pipeline.
- This is NOT a production key. Never use it where a real issued license is required.
