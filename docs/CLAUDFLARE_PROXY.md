# Cloudflare Proxy — Operations Note

Cloudflare proxies the following Predict-A-Trade subdomains (orange-cloud):

- `api.predictatrade.com`
- `live.predictatrade.com`
- `platform.predictatrade.com`
- `status.predictatrade.com`
- `downloads.predictatrade.com`
- `docs.predictatrade.com` (served off this server; CNAME elsewhere)

This note records the **nginx/wiring changes already applied** and the
**Cloudflare dashboard settings that MUST be set manually** (not reachable from
this repo).

---

## What is already done (nginx, committed)

1. **Real client IP restoration** — `nginx/snippets/cloudflare-realip.conf`
   trusts all Cloudflare IPv4+IPv6 ranges and sets
   `real_ip_header CF-Connecting-IP; real_ip_recursive on;`. Included in
   `nginx.conf` (http context) so every vhost sees the true visitor IP in
   `$remote_addr`, `X-Real-IP`, and `X-Forwarded-For`.

   Why: without this, `limit_req`/`limit_conn` keyed on Cloudflare's IP and
   would rate-limit/block ALL users at once, and audit logs stored Cloudflare
   IPs instead of the real visitor.

2. **API no-cache** — `Cache-Control: no-store` added to `cors-edge.conf`
   (covers every `/api/v1` + `/ingest/agent` route) and the api vhost's bare
   `/` block. Cloudflare will NOT cache dynamic data (prices/signals/state).

3. **WebSocket timeout alignment** — `live` and `platform` `/ws` proxy
   timeouts tightened `300s → 90s`. Cloudflare free plan cuts idle WebSockets
   at 100s; 90s keeps a 10s safety margin.

4. **Explicit `CF-Connecting-IP` forwarding** in `proxy-common.conf` and
   `websocket-proxy.conf` so the NestJS compliance/IP-trust logic always
   receives the real client IP.

5. **EA IPv4 fallback** — `api-ipv4.predictatrade.com` is in the api vhost
   `server_name`; EAs have a `PATCloudURLFallback` input (set it to
   `https://api-ipv4.predictatrade.com` and recompile) to retry over IPv4 if a
   primary request returns `HTTP -1`.

**No backend or database code change was required.** NestJS
`extractClientIp()` already trusts `cf-connecting-ip` + private docker ranges;
Go `livepreview` reads XFF→X-Real-IP→RemoteAddr. Real IP flows correctly once
nginx restores it.

---

## Cloudflare dashboard settings (MUST set manually)

1. **SSL/TLS → Overview → Mode: Full (Strict).**
   The origin has valid Let's Encrypt certs. "Flexible" would terminate TLS at
   CF and send plaintext to nginx, breaking the origin's HTTPS requirement.

2. **Cache rule for `api.predictatrade.com/*` (and `live`/`platform` dynamic
   routes):** Cache Level = Bypass, Edge Cache TTL = 0. This is belt-and-
   -suspenders with the `no-store` header already emitted. Do NOT cache
   `/api/v1/*` or `/ingest/*`.

3. **Verify proxy status** — each subdomain's DNS record must be
   orange-cloud (Proxied). `docs.predictatrade.com` is CNAME'd off this server;
   confirm it proxies wherever it lives.

4. **WebSocket support** — enabled on all plans by default; no action needed,
   but confirm no "Disable WebSockets" rule exists. The 90s nginx timeout keeps
   us under CF's 100s idle cut.

5. **Free-plan limits to know:**
   - Max request size 100 MB (EA POST bodies are tiny — fine).
   - Idle WebSocket cut at 100s (handled).
   - No custom WAF on free; if you later add rate-limiting WAF rules, allowlist
     your broker VPS IP ranges so EAs are not challenged.

---

## Gotchas (learned the hard way)

- **DNS propagation:** after orange-clouding, `dig` may still show the origin
  IP for a few minutes. The nginx config is correct for the post-propagation
  state (CF anycast). Wait for `dig api.predictatrade.com` to return a Cloudflare
  IP (e.g. `104.x` / `172.64.x`) before concluding.

- **MT4 `HTTP -1` root cause was IPv6-first DNS**, not Cloudflare. With CF
  proxy the hostname now resolves to Cloudflare anycast (solid IPv4), which
  fixes the EA connectivity. If a terminal still fails, check the EA log
  `err=` value: `4011/4014` = WebRequest allowlist, `6xxx` = network/TLS.

- **`pg_cron` is NOT bundled** in the TimescaleDB image — do not add it to
  `shared_preload_libraries` (postgres crash-loops). Scheduled jobs use the
  built-in TimescaleDB job scheduler (`timescaledb.add_job`), e.g. the daily
  retention-pruning job `pat_daily_pruning` (migration 147).

- **Real-IP + rate limit interaction:** because `$remote_addr` is now the real
  visitor, `limit_req_zone`/`limit_conn_zone` keyed on `$remote_addr` now
  correctly isolate per-user. If you ever see global 503s after a CF change,
  check that `cloudflare-realip.conf` is still included.

---

## Verification checklist

- [ ] `dig api.predictatrade.com` returns a Cloudflare IP (not origin).
- [ ] `curl -sI https://api.predictatrade.com/api/v1/health` → `200` and
      response headers include `cache-control: no-store`.
- [ ] `curl -sI https://api.predictatrade.com/` → API JSON + no-store.
- [ ] MT4/MT5 Master Node EA logs show successful `/ingest/agent` POSTs (no
      `HTTP -1`).
- [ ] Dashboard WS (live/platform) stays connected > 90s without mid-frame
      disconnects.
- [ ] `docker exec pat-nginx nginx -t` passes; `grep -c set_real_ip_from
      /etc/nginx/snippets/cloudflare-realip.conf` = 22.
