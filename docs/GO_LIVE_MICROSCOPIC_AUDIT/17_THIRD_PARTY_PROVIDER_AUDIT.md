# 17 — Third-Party Provider Audit

| Provider | Purpose | Owner code | Credential | Timeout/Retry | Live? | Failure behavior |
|---|---|---|---|---|---|---|
| FMP | COT + economic calendar (news) | `pkg/macro`, `pkg/news` | `FMP_API_KEY` in `infra/env/realtime.env` **plaintext** | 5-min sync; stale>900s | **LIVE** — logs "Synced 1 relevant events" every 5 min through audit time | NEWS_FAIL_POLICY=BLOCK_TRADING ⇒ fail-closed veto |
| Twelve Data | DXY index | `main.go` DXY loop | key in env plaintext | refresh loop, fail-safe neutral | LIVE (value observed in MANIFEST; loop active) | degrade to unavailable weight |
| Ollama (local) | sentiment | pkg/ollama | none | 2s timeout | **UNREACHABLE from container** (localhost bind) → always neutral | silent fallback 0.0 |
| ntfy | push notifications | pkg/notifications | optional token | n/a | **MISCONFIGURED**: env points to `127.0.0.1:8090`; actual service publishes 8091 | pushes fail silently |
| SMTP mail.predictatrade.com:587 | transactional email | control nodemailer | password plaintext env | provider-specific | UNVERIFIED (no send test performed) | dev fallback drops mail silently if insecure config |
| Telegram bot | notifications | realtime notifier | token+chat id plaintext env | n/a | UNVERIFIED (no test message sent) | logged errors only |
| Stripe | payments | — | — | — | **NOT INTEGRATED** despite DB row claiming STRIPE provider | webhook stub accepts unverified POSTs |

## §32 failure-mode posture (static)

News: DATA_UNAVAILABLE vetoes ⇒ provider failure cannot become trading evidence. COT/DXY: weight-0/optional ⇒ degrade honestly. Sentiment: neutral fallback. Payments: absent.

## Findings

- **17-1 P0/P2:** all third-party credentials sit in world-readable-on-host plaintext env files and several are also effective secrets for a publicly reachable host (see 30 for rotation list).
- **17-2 P1:** Stripe appears in financial data without any integration code — provenance of the SUCCEEDED payment must be explained by operator or purged.
- **17-3 PASS:** no fabricated external data detected entering scoring (COT/DXY/sentiment are zero-weight/dead).
