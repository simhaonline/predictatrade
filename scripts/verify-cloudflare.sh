#!/usr/bin/env bash
# Verify the Cloudflare proxy is correctly configured for Predict-A-Trade.
# Checks: DNS resolves to Cloudflare anycast, API returns 200 with no-store,
# and the EA ingest endpoint is reachable (401 = auth required = alive).
#
# Usage: ./scripts/verify-cloudflare.sh [--quiet]
set -uo pipefail

QUIET=0
[[ "${1:-}" == "--quiet" ]] && QUIET=1
log() { [[ $QUIET -eq 0 ]] && echo "$*"; }
pass() { log "  [PASS] $*"; PASS=$((PASS+1)); }
fail() { log "  [FAIL] $*"; FAIL=$((FAIL+1)); }

PASS=0; FAIL=0

# Cloudflare anycast ranges are 104.16.0.0/12, 172.64.0.0/13, 162.158.0.0/15,
# plus the 104.24/14 and 198.41/17 etc. We accept any of the well-known CF IPv4
# prefixes as "proxied". IPv6 2400:cb00::/32 / 2606:4700::/32 also count.
is_cloudflare_ip() {
  local ip="$1"
  [[ "$ip" =~ ^(104\.(1[6-9]|2[0-9]|3[0-1])\.|^172\.6[4-7]\.|^162\.15[0-9]\.|^104\.2[4-7]\.|^198\.4[01]\.|^141\.101\.|^108\.162\.|^190\.93\.|^188\.114\.|^173\.245\.|^103\.2[12]\.|^197\.234\.|^131\.0\.72\.) ]] && return 0
  [[ "$ip" =~ ^(2400:cb00|2606:4700|2803:f800|2405:b500|2405:8100|2a06:98c0|2c0f:f248): ]] && return 0
  return 1
}

API_HOST="api.predictatrade.com"
HEALTH="https://${API_HOST}/api/v1/health"
INGEST="https://${API_HOST}/ingest/agent"

log "== Cloudflare proxy verification =="

# 1) DNS resolves to Cloudflare anycast (proxied)
log "1) DNS for ${API_HOST} (expect Cloudflare anycast):"
resolved=$(dig +short A "$API_HOST" 2>/dev/null | grep -v '^;' | head -5)
if [[ -z "$resolved" ]]; then
  fail "no A record resolved for ${API_HOST}"
else
  while IFS= read -r ip; do log "     $ip"; done <<< "$resolved"
  hit=0
  while IFS= read -r ip; do is_cloudflare_ip "$ip" && hit=1; done <<< "$resolved"
  if [[ $hit -eq 1 ]]; then pass "resolves to Cloudflare anycast (proxied)"; else fail "resolves to NON-Cloudflare IP — proxy not active yet (DNS may still be propagating)"; fi
fi

# 2) API health returns 200 with no-store
log "2) GET ${HEALTH} (expect 200 + cache-control: no-store):"
body=$(curl -s -o /dev/null -w "http=%{http_code}" --max-time 25 "$HEALTH" 2>/dev/null)
http_code="${body#http=}"
cc=$(curl -s -D - -o /dev/null --max-time 25 "$HEALTH" 2>/dev/null | grep -i '^cache-control:' | tr -d '\r')
log "     status=$http_code  $cc"
[[ "$http_code" == "200" ]] && pass "API health 200" || fail "API health not 200 (got '$http_code')"
[[ "$cc" == *no-store* ]] && pass "Cache-Control: no-store present" || fail "missing no-store (Cloudflare could cache dynamic data)"

# 3) EA ingest endpoint reachable (401 = alive, auth required)
log "3) POST ${INGEST} (expect 401 = backend alive & auth enforced):"
ing=$(curl -s -o /dev/null -w "http=%{http_code}" -X POST --max-time 25 "$INGEST" 2>/dev/null)
ing_code="${ing#http=}"
log "     status=$ing_code"
[[ "$ing_code" == "401" ]] && pass "ingest endpoint alive (401 auth enforced)" || fail "ingest endpoint unexpected status '$ing_code'"

# 4) nginx real_ip ranges loaded (if running locally)
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^pat-nginx$'; then
  log "4) nginx real_ip ranges loaded:"
  n=$(docker exec -i pat-nginx grep -c "set_real_ip_from" /etc/nginx/snippets/cloudflare-realip.conf 2>/dev/null | tr -d '\r')
  if [[ "$n" =~ ^[0-9]+$ ]] && [[ "$n" -ge 20 ]]; then pass "$n Cloudflare CIDRs trusted"; else fail "real_ip ranges missing or incomplete (count='$n')"; fi
else
  log "4) skipped: pat-nginx not running locally"
fi

log "== Result: ${PASS} passed, ${FAIL} failed =="
[[ $FAIL -eq 0 ]] && exit 0 || exit 1
