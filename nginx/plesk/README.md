# Predict-A-Trade — Plesk Edge Proxy Kit

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
PLESK SERVER (nginx vhosts, TLS)        ← THIS KIT (nginx/plesk/vhosts/)
   │ HTTP  → origin 2.29.23.42:8443
   ▼
pat-nginx :8443 (nginx/plesk/origin-8443.conf, HTTP only)
   │ HTTP (docker network)
   ▼
containers (control ×2 / realtime / frontend / live-terminal / status / files)
```

Direct-origin failover kept for EAs:
`api-ipv4.predictatrade.com` still terminates TLS on the origin :443
(unchanged site config) — EAs with `PATCloudURLFallback` survive a Plesk
outage.

## Files

| File | Where it goes |
|------|---------------|
| `vhosts/api.predictatrade.com.plesk.conf` | Plesk → domain → Apache & nginx Settings → Additional nginx directives |
| `vhosts/platform.predictatrade.com.plesk.conf` | same |
| `vhosts/live.predictatrade.com.plesk.conf` | same |
| `vhosts/status.predictatrade.com.plesk.conf` | same |
| `vhosts/downloads.predictatrade.com.plesk.conf` | same |
| `vhosts/docs.predictatrade.com.plesk.conf` | same |
| `00-global.conf` | Tools & Settings → Nginx Settings (server-wide) OR /etc/nginx/conf.d/zz-pat-proxy.conf |

## Origin-side (this server, committed)

| File | Purpose |
|------|---------|
| `../plesk/origin-8443.conf` | :8443 HTTP-only edge — mirrors each site's 443 logic minus TLS |
| `../snippets/plesk-edge-realip.conf` | trusts the Plesk IP for real-IP restoration |
| `../snippets/forwarded-proto-https.conf` | X-Forwarded-Proto https on :8443 |

## Deploy order (zero-downtime)

1. Origin: firewall :8443 to PLESK_IP only (Hetzner firewall / ufw).
2. Origin: fill `__PLESK_SERVER_IP__` in `nginx/snippets/plesk-edge-realip.conf`,
   include both new snippets + `plesk/origin-8443.conf`, verify, reload.
3. Plesk: paste `00-global.conf` server-wide; create the 6 vhosts; verify.
4. DNS: switch the 6 subdomains' A records from origin-IP (CF → origin) to
   CF → PLESK_IP. Keep `api-ipv4` pointed DIRECTLY at origin IP (grey cloud,
   DNS-only) — EA failover must bypass both Plesk and CF.
5. Run `scripts/verify-plesk-proxy.sh` on the origin.

## TLS on Plesk

Use the Cloudflare Origin CA cert per domain (or Plesk Let's Encrypt).
CF SSL mode: Full (Strict).

## SMTP — NOT proxied

pat.predictatrade.com :465/:587 stay DNS-direct to the origin IP. Do not
proxy mail through Plesk (Plesk runs its own mail stack).
