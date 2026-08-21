# Self-Hosted Notifications Setup Guide

This guide covers setting up **ntfy** (push notifications) and **wppconnect-server** (WhatsApp) as self-hosted services — no third-party cloud providers required.

---

## 1. ntfy — Self-Hosted Push Notifications

**Project:** https://github.com/binwiederhier/ntfy  
**Android app:** https://github.com/binwiederhier/ntfy-android  
**iOS app:** https://github.com/binwiederhier/ntfy-ios

### Quick Setup (Docker)

```bash
# Create data directory
mkdir -p /var/lib/ntfy

# Run ntfy server
docker run -d \
  --name ntfy \
  --restart=unless-stopped \
  -p 8090:80 \
  -v /var/lib/ntfy:/var/lib/ntfy \
  -v /etc/ntfy:/etc/ntfy \
  binwiederhier/ntfy serve

# Or install as binary:
# sudo apt install ntfy
# sudo systemctl enable --now ntfy
```

### Configuration

Create `/etc/ntfy/server.yml`:

```yaml
base-url: "http://your-server-ip:8090"
listen: ":80"
cache-file: "/var/lib/ntfy/cache.db"
behind-proxy: true

# Optional: require auth token for your topic
auth-file: "/var/lib/ntfy/auth.db"
auth-default-access: "deny-all"
```

### Create Protected Topic (optional)

```bash
# Grant access to your topic
ntfy access predictatrade-alerts your-secret-token ro

# The token goes in NTFY_ACCESS_TOKEN in your env file
```

### Phone App Setup

1. Install ntfy app from Google Play / F-Droid / App Store
2. Open the app → tap "+" to add subscription
3. Enter your server URL: `http://your-server-ip:8090/predictatrade-alerts`
4. If using auth token: enter it when prompted

### Environment Variables

```env
NTFY_SERVER_URL=http://your-server-ip:8090
NTFY_TOPIC=predictatrade-alerts
NTFY_ACCESS_TOKEN=your-secret-token   # optional, only if topic is protected
NOTIFICATION_PUSH_ENABLED=true
```

### Test It

```bash
# Send a test notification
curl -H "Title: Test" -d "Hello from Predict-A-Trade" \
  http://your-server-ip:8090/predictatrade-alerts
```

---

## 2. wppconnect-server — Self-Hosted WhatsApp

**Project:** https://github.com/wppconnect-team/wppconnect-server

### Quick Setup (Docker)

```bash
# Create data directory
mkdir -p /var/lib/wppconnect

# Run wppconnect-server
docker run -d \
  --name wppconnect \
  --restart=unless-stopped \
  -p 21465:21465 \
  -v /var/lib/wppconnect:/usr/src/app/tokens \
  -v /var/lib/wppconnect/sessions:/usr/src/app/sessions \
  wppconnectteam/wppconnect-server:latest

# Or install from source:
# git clone https://github.com/wppconnect-team/wppconnect-server.git
# cd wppconnect-server
# npm install
# npm run build
# npm start
```

### Configuration

Create a config file or use environment variables. Key settings:

```json
{
  "server": {
    "port": 21465,
    "host": "0.0.0.0"
  },
  "secretKey": "your-secret-api-token",
  "maxQr": 5,
  "webhook": {
    "url": "http://127.0.0.1:13081/api/v1/whatsapp/webhook",
    "events": ["onmessage"]
  }
}
```

### Start Session and Scan QR

```bash
# Start a session
curl -X POST http://localhost:21465/api/predictatrade/start-session \
  -H "Authorization: Bearer your-secret-api-token" \
  -H "Content-Type: application/json" \
  -d '{"webhook": "http://127.0.0.1:13081/api/v1/whatsapp/webhook"}'

# Get the QR code
curl http://localhost:21465/api/predictatrade/qrcode-session \
  -H "Authorization: Bearer your-secret-api-token"

# Open the QR code URL in a browser
# Scan it with WhatsApp on your phone (Settings → Linked Devices → Link a Device)
```

### Environment Variables

```env
WPPCONNECT_SERVER_URL=http://localhost:21465
WPPCONNECT_TOKEN=your-secret-api-token
WPPCONNECT_SESSION=predictatrade
WPPCONNECT_PHONE=1234567890              # recipient phone with country code, no +
NOTIFICATION_WHATSAPP_ENABLED=true
```

### Test It

```bash
# Send a test message
curl -X POST http://localhost:21465/api/predictatrade/send-message \
  -H "Authorization: Bearer your-secret-api-token" \
  -H "Content-Type: application/json" \
  -d '{"phone": "1234567890", "message": "Hello from Predict-A-Trade"}'
```

---

## 3. Restart Engine After Configuration

After setting up both services and filling in the env vars:

```bash
systemctl restart predictatrade-realtime
```

Check logs to verify providers are registered:

```bash
journalctl -u predictatrade-realtime -n 20 --no-pager | grep "notifications"
```

Expected output:
```
[notifications] Provider smtp (EMAIL) registered and configured
[notifications] Provider telegram (TELEGRAM) registered and configured
[notifications] Provider ntfy (PUSH) registered and configured
[notifications] Provider wppconnect (WHATSAPP) registered and configured
```

If a provider shows "NOT configured", it means the credentials are missing or empty in the env file.
