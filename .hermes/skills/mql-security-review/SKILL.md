---
name: mql-security-review
description: "Audit MQL EAs for credential leaks and DLL risks."
---

# mql-security-review

Use when reviewing MQL4/MQL5 EAs for security vulnerabilities, credential exposure, or unsafe DLL usage.

## Forbidden in EAs
- Server DB credentials (postgresql://...)
- API keys (Twelve Data, FMP, NOWPayments, Stripe)
- JWT secrets or private signing keys
- License server private keys
- Payment processor secrets
- AWS/Azure/GCP credentials
- Telegram bot tokens (use input params, not hardcoded)
- SMTP passwords

## Audit Steps
1. Credential scan: rg -n '(password|secret|api.key|token|postgres|redis|mongodb)' mql/
2. DLL audit: rg -n '#import' mql/ — kernel32/user32 ok, custom DLLs signed, never urlmon/wininet
3. WebRequest: HTTPS only, no credentials in URL, MT4 max 20-char URL
4. File ops: FILE_COMMON only, never MQL4/Files with credentials
5. Input params: no default secret values, string inputs validated

## Known Issues
- W3 (HIGH): No signed IPC/WS, no replay protection
- TelegramBotToken as input param — correct approach, not hardcoded
