#!/bin/bash
# ─── Predict-A-Trade — Certbot SSL for docs.predictatrade.com ───
# Run: chmod +x docs/scripts/setup-docs-ssl.sh && sudo ./docs/scripts/setup-docs-ssl.sh
set -e

DOMAIN="docs.predictatrade.com"
EMAIL="admin@predictatrade.com"
CERTBOT_WEBROOT="/var/www/certbot"

echo "=== SSL Setup for ${DOMAIN} ==="

echo "[1/5] Checking DNS..."
if command -v dig &>/dev/null; then
    IP=$(dig +short "${DOMAIN}" A)
elif command -v nslookup &>/dev/null; then
    IP=$(nslookup "${DOMAIN}" | awk '/^Address: / {print $2}' | tail -1)
else
    echo "WARNING: Cannot verify DNS. Ensure A record points to this server."
    IP="unknown"
fi
if [ -z "${IP}" ] || [ "${IP}" = "unknown" ]; then
    echo "ERROR: ${DOMAIN} does not resolve."
    echo "Create A record: ${DOMAIN} → $(curl -s ifconfig.me 2>/dev/null || echo 'YOUR_SERVER_IP')"
    exit 1
fi
echo "  ${DOMAIN} → ${IP}"

echo "[2/5] Creating webroot..."
mkdir -p "${CERTBOT_WEBROOT}/.well-known/acme-challenge"

echo "[3/5] Requesting certificate..."
if command -v certbot &>/dev/null; then
    certbot certonly --webroot --webroot-path="${CERTBOT_WEBROOT}" \
        --domain "${DOMAIN}" --email "${EMAIL}" --agree-tos --non-interactive --keep-until-expiring
elif command -v docker &>/dev/null; then
    docker run --rm -v "${CERTBOT_WEBROOT}:/var/www/certbot" -v "/etc/letsencrypt:/etc/letsencrypt" \
        certbot/certbot certonly --webroot --webroot-path="/var/www/certbot" \
        --domain "${DOMAIN}" --email "${EMAIL}" --agree-tos --non-interactive --keep-until-expiring
else
    echo "ERROR: Install certbot: sudo apt install -y certbot"
    exit 1
fi

CERT_PATH="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
[ ! -f "${CERT_PATH}" ] && echo "ERROR: Certificate not created." && exit 1
echo "  Certificate created"

echo "[4/5] Enabling HTTPS in nginx config..."
CONF="/srv/predictatrade/xauusd/docs/nginx/docs.predictatrade.com.conf"
if [ -f "${CONF}" ]; then
    sed -i 's/^# server {/server {/' "${CONF}"
    sed -i 's/^#     listen/    listen/' "${CONF}"
    sed -i 's/^#     ssl/    ssl/' "${CONF}"
    sed -i 's/^#     include/    include/' "${CONF}"
    sed -i 's/^#     root/    root/' "${CONF}"
    sed -i 's/^#     index/    index/' "${CONF}"
    sed -i 's/^#     location/    location/' "${CONF}"
    sed -i 's/^#         try_files/        try_files/' "${CONF}"
    sed -i 's/^#     }/    }/' "${CONF}"
    sed -i 's/^# }/}/' "${CONF}"
    sed -i 's/^    # return 301/    return 301/' "${CONF}"
    echo "  HTTPS enabled"
fi

echo "[5/5] Reloading nginx..."
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q pat-nginx; then
    docker compose -f /srv/predictatrade/xauusd/docker-compose.yml restart nginx
    echo "  Nginx restarted"
elif systemctl is-active --quiet nginx 2>/dev/null; then
    nginx -t && systemctl reload nginx
    echo "  Nginx reloaded"
fi

echo ""
echo "=== DONE ==="
echo "  http://${DOMAIN}  →  https://${DOMAIN}"
echo "  Verify: curl -I https://${DOMAIN}"
echo ""
echo "Auto-renewal: add to crontab:"
echo "  0 3 * * * certbot renew --quiet --post-hook 'docker compose -f /srv/predictatrade/xauusd/docker-compose.yml restart nginx'"
