# Predict-A-Trade — Plesk Edge Proxy (per-domain, zero dependencies)

Plesk (separate server) terminates public TLS for the 6 predictatrade.com
subdomains and reverse-proxies to the Docker origin. Cloudflare stays
orange-cloud in front of Plesk.

```
Browser / EA
   │ HTTPS
   ▼
Cloudflare (orange-cloud, TLS)          ← unchanged
   │ HTTPS
   ▼
PLESK SERVER (per-domain nginx directives, TLS)
   │ HTTP  → origin 2.29.23.42:8443
   ▼
pat-nginx :8443 (origin side, already active) → containers
```

## Install — one file per domain, nothing global

For each domain: **Plesk → Domains → <domain> → Apache & nginx Settings
→ "Additional nginx directives" → paste the whole file → OK → Apply**

| Domain | File to paste |
|--------|---------------|
| api.predictatrade.com | `vhosts/api.predictatrade.com.plesk.conf` |
| platform.predictatrade.com | `vhosts/platform.predictatrade.com.plesk.conf` |
| live.predictatrade.com | `vhosts/live.predictatrade.com.plesk.conf` |
| status.predictatrade.com | `vhosts/status.predictatrade.com.plesk.conf` |
| downloads.predictatrade.com | `vhosts/downloads.predictatrade.com.plesk.conf` |
| docs.predictatrade.com | `vhosts/docs.predictatrade.com.plesk.conf` |

Each file is fully standalone: no `include`, no `map`, no global directives,
no extra files on the Plesk server. Plesk keeps `listen`/`ssl_certificate`/
`server_name`; the directives only add proxy locations. WebSocket upgrade
uses a literal `Connection "upgrade"` header — no `$connection_upgrade` map
(whose http-level definition Plesk's per-domain box cannot host).

## TLS on Plesk

Cloudflare Origin CA cert per domain (or Plesk Let's Encrypt).
Cloudflare SSL mode: **Full (Strict)**.

## DNS cutover

- The 6 subdomains: A record → PLESK server IP (orange-cloud).
- `api-ipv4.predictatrade.com`: stays grey-cloud, A → origin `2.29.23.42`
  (EA direct-origin failover — bypasses Plesk AND Cloudflare).
- `pat.predictatrade.com` (:465/:587 SMTP): stays DNS-direct to the origin.
  Never proxy mail through Plesk.

## Origin side (this server — already active, verified)

| File | Purpose |
|------|---------|
| `../plesk/origin-8443.conf` | :8443 HTTP-only edge — mirrors each site's 443 logic minus TLS (included from the live nginx.conf; compose publishes 8443:8443) |
| `../snippets/plesk-edge-realip.conf` | trusts the Plesk IP for real client-IP restore (activate: set `PLESK_SERVER_IP` in `infra/env/.env`, run `scripts/deploy-plesk-proxy.sh`) |

Deploy/verify on the origin: `scripts/deploy-plesk-proxy.sh`,
`scripts/verify-plesk-proxy.sh`.

## After cutover

Run `scripts/verify-plesk-proxy.sh` from the origin — checks :8443 directly,
the Plesk edge per domain, the public CF path, api health/ingest, and the
api-ipv4 failover record.