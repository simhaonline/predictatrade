#!/usr/bin/env bash
# ============================================================
# scripts/verify-plesk-proxy.sh — end-to-end Plesk chain check
# ============================================================
# Verifies the CF → Plesk → origin :8443 → containers chain for all 6
# predictatrade.com subdomains. Run AFTER the Plesk vhosts are installed and
# DNS is cut over. Also validates the origin-side :8443 listeners directly.
#
# Usage: ./scripts/verify-plesk-proxy.sh [PLESK_IP]
#   PLESK_IP defaults to $PLESK_SERVER_IP from infra/env/.env
# ============================================================
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

PASS=0; FAIL=0
pass() { echo "  [PASS] $*"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $*"; FAIL=$((FAIL+1)); }

PLESK_IP="${1:-}"
if [[ -z "$PLESK_IP" && -f infra/env/.env ]]; then
  PLESK_IP="$(grep -E '^PLESK_SERVER_IP=' infra/env/.env | tail -1 | cut -d= -f2- | tr -d '\"' | tr -d "'" | xargs)"
fi
ORIGIN_IP="$(curl -4 -s --max-time 5 ifconfig.me 2>/dev/null || echo unknown)"

DOMAINS=(api platform live status downloads docs)

echo "== Predict-A-Trade Plesk proxy verification =="
echo "origin: $ORIGIN_IP   plesk: ${PLESK_IP:-NOT SET}"

# ── 1. Origin-side :8443 direct checks (Plesk-independent) ──
echo "── 1. origin :8443 listeners (from pat-nginx container) ──"
for d in "${DOMAINS[@]}"; do
  host="$d.predictatrade.com"
  code="$(docker exec pat-nginx curl -s -o /dev/null -w '%{http_code}' -H "Host: $host" http://127.0.0.1:8443/ 2>/dev/null || echo 000)"
  [[ "$code" != "000" ]] && pass ":8443 $host -> $code" || fail ":8443 $host unreachable"
done

# ── 2. Plesk edge TLS (if IP given): vhost answers with origin upstream ──
if [[ -n "$PLESK_IP" ]]; then
  echo "── 2. Plesk edge ($PLESK_IP) per-domain ──"
  for d in "${DOMAINS[@]}"; do
    host="$d.predictatrade.com"
    code="$(curl -sk -o /dev/null -w '%{http_code}' --max-time 10 --resolve "$host:443:$PLESK_IP" "https://$host/" 2>/dev/null || echo 000)"
    [[ "$code" != "000" && "$code" != "502" && "$code" != "504" ]] \
      && pass "plesk $host -> $code" \
      || fail "plesk $host -> $code (TLS/vhost/upstream problem)"
  done
else
  echo "── 2. Plesk edge: SKIPPED (no PLESK_SERVER_IP) ──"
fi

# ── 3. Public path through Cloudflare ──
echo "── 3. public path (through Cloudflare) ──"
for d in "${DOMAINS[@]}"; do
  host="$d.predictatrade.com"
  out="$(curl -s -o /dev/null -w '%{http_code} %{remote_ip}' --max-time 10 "https://$host/" 2>/dev/null || echo "000 -")"
  code="${out%% *}"; ip="${out#* }"
  if [[ "$code" != "000" && "$code" != "5xx" && "$code" != "502" && "$code" != "504" ]]; then
    pass "public $host -> $code via $ip"
  else
    fail "public $host -> $out"
  fi
done

# ── 4. api deep checks: health, ingest, real-IP chain ──
echo "── 4. api deep checks ──"
h="$(curl -s --max-time 10 https://api.predictatrade.com/api/v1/health 2>/dev/null | head -c 200)"
[[ "$h" == *'"status"'*''*'"ok"'* || "$h" == *status* ]] && pass "api health body: ${h:0:80}" || fail "api health: '$h'"
ing="$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 -X POST https://api.predictatrade.com/ingest/agent 2>/dev/null || echo 000)"
[[ "$ing" != "000" && "$ing" != "502" && "$ing" != "503" && "$ing" != "504" ]] && pass "ingest/agent reachable -> $ing (401 expected unauthenticated)" || fail "ingest/agent -> $ing"
cf="$(curl -sI --max-time 10 https://api.predictatrade.com/api/v1/health 2>/dev/null | grep -i '^cf-ray' | head -1)"
[[ -n "$cf" ]] && pass "Cloudflare in chain ($cf" || echo "  [INFO] no CF-Ray header — api may be grey-cloud or CF bypassed"

# ── 5. api-ipv4 direct-origin failover intact ──
echo "── 5. EA direct-IP failover (api-ipv4, must bypass Plesk) ──"
res="$(dig +short api-ipv4.predictatrade.com A 2>/dev/null | tail -1)"
if [[ -n "$res" && "$res" == "$ORIGIN_IP" ]]; then
  pass "api-ipv4 resolves DIRECT to origin ($res) — grey-cloud"
elif [[ -n "$res" ]]; then
  fail "api-ipv4 resolves to $res, expected origin $ORIGIN_IP (set DNS-only + A record to origin)"
else
  echo "  [INFO] dig unavailable or no record — verify api-ipv4 manually"
fi

echo
echo "== RESULT: $PASS passed, $FAIL failed =="
[[ $FAIL -eq 0 ]]