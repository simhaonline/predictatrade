#!/usr/bin/env bash
# ============================================================
# scripts/deploy-plesk-proxy.sh — origin-side Plesk edge wiring
# ============================================================
# Prepares and activates the :8443 Plesk-facing listener on pat-nginx.
# Safe to re-run (idempotent). Does NOT touch DNS — that is the operator's
# cutover step after verification.
#
# Steps:
#   1. Read PLESK_SERVER_IP from infra/env/.env (required).
#   2. Substitute __PLESK_SERVER_IP__ in nginx/snippets/plesk-edge-realip.conf.
#   3. Add the :8443 listener include to the LIVE nginx.conf
#      (docs/nginx/nginx.conf — the file mounted into pat-nginx).
#   4. docker compose up -d nginx (recreate with same mounts = no-op if
#      unchanged; nginx -t runs before reload via container entrypoint).
#   5. Verify :8443 answers for all 6 domains.
#
# Usage: PLESK_SERVER_IP=x.x.x.x ./scripts/deploy-plesk-proxy.sh [--dry-run]
# ============================================================
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

DRY_RUN=0
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=1

ENV_FILE="infra/env/.env"
[[ -f "$ENV_FILE" ]] || { echo "FAIL: $ENV_FILE not found"; exit 1; }
PLESK_IP="$(grep -E '^PLESK_SERVER_IP=' "$ENV_FILE" | tail -1 | cut -d= -f2- | tr -d '\"' | tr -d "'" | xargs)"
[[ -n "$PLESK_IP" ]] || { echo "FAIL: PLESK_SERVER_IP not set in $ENV_FILE"; exit 2; }
echo "== Plesk edge IP: $PLESK_IP =="

REALIP_SNIPPET="nginx/snippets/plesk-edge-realip.conf"
LIVE_CONF="docs/nginx/nginx.conf"

# --- 2. substitute IP into realip snippet (idempotent replace) ---
sed -i "s/^set_real_ip_from .*/set_real_ip_from ${PLESK_IP};/" "$REALIP_SNIPPET"
grep -q "set_real_ip_from ${PLESK_IP};" "$REALIP_SNIPPET" || { echo "FAIL: realip substitution"; exit 3; }
echo "[ok] plesk-edge-realip.conf trusts ${PLESK_IP}"

# --- 3. ensure :8443 listener include present in LIVE nginx.conf ---
ORIGIN_CONF="nginx/plesk/origin-8443.conf"
[[ -f "$ORIGIN_CONF" ]] || { echo "FAIL: $ORIGIN_CONF missing"; exit 4; }

if ! grep -q 'plesk/origin-8443.conf' "$LIVE_CONF"; then
  if [[ $DRY_RUN -eq 0 ]]; then
    # insert include before the sites-available include (last include line)
    sed -i "s|    # Include site configurations|    # Plesk-facing :8443 HTTP edge (CF → Plesk → origin). Plain HTTP only —\n    # firewall :8443 to the Plesk server IP ONLY (PLESK_SERVER_IP in env).\n    include /etc/nginx/plesk/origin-8443.conf;\n\n    # Include site configurations|" "$LIVE_CONF"
    grep -q 'plesk/origin-8443.conf' "$LIVE_CONF" || { echo "FAIL: include insert"; exit 5; }
    echo "[ok] origin-8443.conf include added to $LIVE_CONF"
  else
    echo "[dry-run] would add origin-8443.conf include to $LIVE_CONF"
  fi
else
  echo "[ok] origin-8443.conf include already present"
fi

# --- 3b. realip include into LIVE nginx.conf (only once IP is real) ---
if [[ "$PLESK_IP" == __* ]]; then
  echo "[skip] plesk-edge-realip include — PLESK_SERVER_IP is still a placeholder"
elif ! grep -q 'plesk-edge-realip.conf' "$LIVE_CONF"; then
  sed -i "s|    include /etc/nginx/snippets/cloudflare-realip.conf;|    include /etc/nginx/snippets/cloudflare-realip.conf;\n    include /etc/nginx/snippets/plesk-edge-realip.conf;|" "$LIVE_CONF"
  grep -q 'plesk-edge-realip.conf' "$LIVE_CONF" || { echo "FAIL: realip include insert"; exit 9; }
  echo "[ok] plesk-edge-realip include added to $LIVE_CONF"
else
  echo "[ok] plesk-edge-realip include already present"
fi

# --- 3c. mount the plesk dir into the container (compose volume) ---
if ! grep -q './nginx/plesk:/etc/nginx/plesk' docker-compose.yml; then
  if [[ $DRY_RUN -eq 0 ]]; then
    python3 - <<'PY'
import re, pathlib
p = pathlib.Path("docker-compose.yml")
t = p.read_text()
old = "      - ./nginx/snippets:/etc/nginx/snippets:ro"
new = old + "\n      - ./nginx/plesk:/etc/nginx/plesk:ro"
assert old in t
p.write_text(t.replace(old, new, 1))
print("[ok] compose volume ./nginx/plesk added")
PY
  else
    echo "[dry-run] would add ./nginx/plesk volume to compose nginx service"
  fi
else
  echo "[ok] compose volume already present"
fi

if [[ $DRY_RUN -eq 1 ]]; then
  echo "== dry-run complete — no changes applied =="
  exit 0
fi

# --- 4. syntax-check inside the container, then apply ---
echo "== applying (compose up -d nginx; recreate + config test) =="
docker compose --env-file "$ENV_FILE" up -d nginx
sleep 2
docker exec pat-nginx nginx -t || { echo "FAIL: nginx -t inside container"; exit 6; }
docker exec pat-nginx nginx -s reload || { echo "FAIL: nginx reload"; exit 7; }
echo "[ok] nginx reloaded with :8443 listener"

# --- 5. verify :8443 answers for all 6 domains ---
sleep 1
FAIL=0
for d in api platform live status downloads docs; do
  host="$d.predictatrade.com"
  code="$(docker exec pat-nginx curl -s -o /dev/null -w '%{http_code}' -H "Host: $host" http://127.0.0.1:8443/ 2>/dev/null || echo 000)"
  echo "  :8443 Host $host -> HTTP $code"
  [[ "$code" == "000" ]] && FAIL=1
done
[[ $FAIL -eq 0 ]] && echo "== ALL :8443 vhosts answer ==" || { echo "== some vhosts did not answer — check logs =="; docker logs pat-nginx --tail 20; exit 8; }