# pat-mail — Predict-A-Trade Mail Relay

Go **send-only SMTP submission relay** for the platform's outbound email
(onboarding, support, payment, alerts, news digests). Runs as the
`pat-mail-relay` container; SMTP submission host is
**`pat.predictatrade.com`**, sending domain **`predictatrade.com`**.

## Why our own relay
- Customer emails must come from `predictatrade.com` (not a third-party
  from-domain) for trust and deliverability control.
- Control plane (`nodemailer`) submits internally over SMTP AUTH — never
  exposed; the relay is the only component with outbound reach.
- Persistent spool (SQLite) + retry/backoff: payment/onboarding emails are
  transactional — they survive restarts and gateway hiccups (dead-letter at
  24h of retries, fully logged).

## Security posture (anti-scam / anti-relay)

| Control | Behavior |
|---|---|
| AUTH required | `AUTH PLAIN` against `SMTP_USERS` (`user:password` list). No anonymous submission. |
| From-domain enforcement | envelope `MAIL FROM` must end in `ALLOWED_FROM_DOMAINS` (`predictatrade.com`) |
| No open relay | unauthenticated MAIL/RCPT → 530; rcpt cap 25; message cap 10 MB |
| STARTTLS / implicit TLS | opportunistic STARTTLS for outbound (port 465 listener available for submission) |
| Retry / dead-letter | 5m→10m→…→6h cap backoff, dead-letter after 30 attempts (logged) |
| Queue persistence | SQLite spool volume `pat-mail-spool` |

## Deployment

```bash
docker compose --env-file infra/env/.env up -d mail-relay
```

The NestJS control plane already points at it:
`infra/env/control.env` → `SMTP_HOST=pat-mail-relay`, `SMTP_PORT=587`,
`SMTP_USERNAME=no-reply@predictatrade.com`, `SMTP_PASSWORD=<from infra/env>`.

## DNS records (operator, done once on `predictatrade.com`)

| Record | Value |
|---|---|
| A | `pat.predictatrade.com` → server IP |
| MX (optional) | `predictatrade.com` → `pat.predictatrade.com` (10) |
| SPF (TXT @) | `v=spf1 a:pat.predictatrade.com -all` |
| DKIM (TXT `pat1._domainkey`) | from `mail-relay/certs/dkim.key` public half (see below) |
| DMARC (TXT `_dmarc`) | `v=DMARC1; p=quarantine; rua=mailto:dmarc@predictatrade.com; fo=1` |

### DKIM key generation (operator, once)
```bash
openssl genrsa -out dkim.key 2048
openssl rsa -in dkim.key -pubout -out dkim.pub    # publish pat1._domainkey TXT
# place dkim.key at ./mail-relay/certs/dkim.key (mounted read-only)
```
Signing is wired via `DKIM_*` env (`pat1` / `predictatrade.com`).

## Env reference

| Var | Default | Purpose |
|---|---|---|
| `SMTP_LISTEN` | `:587` | submission (AUTH PLAIN; STARTTLS optional) |
| `SMTP_TLS_LISTEN` | `:465` | implicit-TLS submission (optional) |
| `SMTP_USERS` | — | `email:password` CSV of authenticated senders |
| `ALLOWED_FROM_DOMAINS` | `predictatrade.com` | From-domain allowlist |
| `MAIL_DOMAIN` | `pat.predictatrade.com` | our HELO/Received host |
| `DKIM_SELECTOR` / `DKIM_DNS_DOMAIN` / `DKIM_PRIVATE_KEY_PATH` | `pat1` / `predictatrade.com` / — | DKIM signing |
| `SQLITE_PATH` | `/var/lib/pat-mail/spool.db` | spool DB |
| `SMARTHOST` | — | optional upstream relay; empty = direct MX |
| `NOWPAYMENTS_UNDERPAY_TOLERANCE_PCT` | `2` | (control plane) allowed underpay % |
| `NOWPAYMENTS_REQUIRE_AMOUNT` | — | `strict` = refuse settlement without gateway amounts |

## Smoke test
```python
# AUTH PLAIN with no-reply creds → MAIL → RCPT → DATA → 250 queued as N
# then docker logs pat-mail-relay → 'delivered msg N'
```