# Runbook — Email Deliverability (Gmail 5.7.28 blocking)

**Symptom:** mail-relay log shows `550 5.7.28 ... Gmail has detected an unusual rate of unsolicited mail originating from your IP address` on delivery to Gmail/Microsoft recipients. Recipients on other providers (own-domain, Century) deliver fine.

**First observed:** 2026-09-10, after the first bulk campaign sends.

**Root cause:** IP-reputation cooldown on `152.53.67.111`. Mail sent before 2026-09-10 09:00 UTC carried no DKIM signature and the domain had no SPF/DMARC — Gmail accumulated a negative reputation for the IP. Reputation recovery is **time-gated**, not config-gated.

## Fixed already (do not re-do)

| Date | Fix | Commit |
|---|---|---|
| 2026-09-10 | DKIM signing (pat1 selector) implemented in pat-mail-relay | 9b4f90c |
| 2026-09-10 | SPF record published (`ip4:152.53.67.111`), DKIM TXT `pat1._domainkey`, DMARC `p=reject` | operator (Plesk) |
| 2026-09-10 | HELO identity fix — relay announced `localhost`, now announces `pat.predictatrade.com` | 5db2e8a |

## Monitoring

- Relay log: `docker logs pat-mail-relay | grep -E '5.7.28|DEAD-LETTER'`
- Spool state: dead-lettered Gmail messages auto-retry on exponential backoff (5m → 6h); they clear automatically once reputation recovers. **No manual re-send needed.**
- Probe cadence: at most **one** small probe per day to a Gmail address. Hammering while blocked extends the cooldown.

## Expected recovery

Typical 5.7.28 cooldowns clear within **3 days to 2 weeks** of consistent clean, signed, low-volume sending. First sign of recovery: a relay log `delivered msg N to [...gmail.com]` with no 5.7.28.

## If still blocked after 2 weeks

1. **Google Postmaster Tools** (https://postmaster.google.com) — add `predictatrade.com`, verify by publishing the generated `google-site-verification` TXT on the domain (Plesk DNS). Gives IP reputation dashboards.
2. **Check rDNS**: `152.53.67.111` → `ops.simhaonline.com` (already set). Optional improvement: align PTR to a predictatrade hostname via the VPS provider, and add the PTR hostname to SPF if it sends.
3. **Volume warm-up**: if the first real newsletter burst re-triggers blocking, warm up volume (≤50/day for a week, then double weekly).
4. **Microsoft/Outlook** uses its own SmartNetworkDataServices — same pattern, same fix path (https://postmaster.live.com).

## Where mail state lives

- Relay spool: `pat-mail-relay:/var/lib/pat-mail/spool.db` (SQLite) — `messages` table, `attempts`, `last_error`.
- Dead-letter threshold: 30 attempts / 24h (`maxAttempts` in mail-relay/main.go).