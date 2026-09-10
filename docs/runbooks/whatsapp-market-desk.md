# Runbook — WhatsApp Market Desk (PAT WhatsApp AI)

The Predict-A-Trade WhatsApp assistant: button-driven market intelligence — live price, next-move bias, TA summaries, engine signals, risk tips.

## Architecture

```
WhatsApp provider (Meta Cloud API / Twilio / WAHA)
        │  webhook POST (HMAC-signed)
        ▼
control: /api/v1/whatsapp/inbound          ← signature verify + dedup
        ▼
WhatsAppAssistantEngine                    ← persona + flows
        │  fetch server-authoritative data: REALTIME_URL
        │    /market/snapshot · /signals · /cross-market/health
        ▼
WhatsAppSender (Meta Cloud API shape)      ← text + interactive buttons
        ▼
user's WhatsApp
```

- Engine NEVER recomputes indicators — everything is pulled from the realtime gateway (server-authoritative truth).
- Outbound failures are logged, never thrown — delivery is best-effort (same philosophy as notifications.Manager).

## Operator setup (go-live)

1. **Meta Business WhatsApp account** → get `PHONE_NUMBER_ID` + permanent `ACCESS_TOKEN`.
2. Set env in `infra/env/control.env`:
   ```
   WHATSAPP_TOKEN=<access token>
   WHATSAPP_PHONE_NUMBER_ID=<phone number id>
   WHATSAPP_WEBHOOK_SECRET=<random string — also set in Meta webhook config>
   WHATSAPP_API_URL=https://graph.facebook.com/v21.0   # default
   ```
3. Meta App Dashboard → WhatsApp → Webhooks:
   - Callback URL: `https://api.predictatrade.com/api/v1/whatsapp/inbound`
   - Verify token: any string (the GET handshake route lives at `/whatsapp/webhook/verify`)
   - Subscribe to `messages` field.
4. Restart control (`docker compose --env-file infra/env/.env up -d control control-b`).

Until `WHATSAPP_TOKEN`/`WHATSAPP_PHONE_NUMBER_ID` are set, the gateway runs in DEV mode: unsigned webhooks accepted (logged loudly), sends suppressed and logged — safe to smoke-test with curl.

## Smoke test (dev mode)

```bash
curl -X POST https://api.predictatrade.com/api/v1/whatsapp/inbound \
  -H 'Content-Type: application/json' \
  -d '{"from":"+9715xxxxxxx","text":"price"}'
# → control log shows the reply that WOULD be sent
```

## User features

| Button / keyword | Flow |
|---|---|
| 📈 Live Price (`price`) | H1/D1 OHLC from the server snapshot |
| 🔮 Prediction (`prediction`) | Engine bias (BULLISH/BEARISH/MIXED/NO-TRADE) from latest signals |
| 📊 Technical Analysis (`ta`) | H1 candle shape, close position, body strength, macro-driver health |
| ⚡ Signals (`signals`) | Last 5 engine prints with direction/strategy/grade |
| 🛡️ Risk Tips (`risk`) | Rotating position-sizing/stop/discipline tips |
| Alerts (`alerts`) | How to bind the number for push alerts |

Persona: concise, direct, button-first. **No liability disclaimers per message** — enforced by a dedicated test that asserts every reply flow against banned-phrase patterns; compliance lives in the platform footer/emails, not the chat.

## Security notes

- Inbound without `WHATSAPP_WEBHOOK_SECRET` set = DEV mode (accepts unsigned, suppresses sends). Never ship production without the secret.
- Signature: HMAC-SHA256 over raw body, `X-Hub-Signature-256` (Meta) or `X-Webhook-Signature`, timing-safe compare.
- Dedup by `message_id` (providers redeliver); rate limit 60/min/IP.
- Meta interactive buttons: max 3 — overflow renders as a numbered list in the body.