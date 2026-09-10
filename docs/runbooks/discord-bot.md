# Runbook — Discord Bot (alerts · portfolio · price prediction)

discord.js v14 standalone service (`discord-bot/`) as compose service `discord-bot` (`pat-discord-bot`).

## What it does

| Slash command | Rich embed content |
|---|---|
| `/price` | Live XAU/USD tick + H1/D1 OHLC from the server snapshot |
| `/predict` | Weighted-bias prediction: EMA/MACD/RSI/Bollinger/Volume-Profile votes + engine signal balance + BTC risk sentiment, with confidence % and the full vote ledger |
| `/indicators` | EMA(9/21/50), MACD(12,26,9), RSI(14), Bollinger(20,2) + %B, Volume Profile (POC + 70% value area) |
| `/signals` | Last 5 gate-cleared engine prints (entry/SL/TP ladder, R:R, regime) |
| `/btc` | BTC driver + liquidity-regime desk read |
| `/mtlog` | Agent mesh, feed health, strategy-engine liveness |
| `/portfolio [account]` | Net P&L, win rate, per-strategy breakdown from `trading.trade_results` (read-only DB) |
| `/alerts` | Alert-stream map (signals / system-health / risk events) + channel |

Plus: **refresh buttons** on every embed (Price · Predict · Signals · Indicators) and an automatic **alert mirror** — new gate-cleared signal prints post to `ALERT_CHANNEL_ID` within 30s of hitting the gateway.

## Data authority

- All market/prediction data comes from the realtime gateway (server-authoritative; MT4_MASTER feed).
- Portfolio reads `trading.trade_results` via a read-only PostgreSQL connection (max 3 pooled clients).
- The bot NEVER executes trades and NEVER recomputes strategy scores — the realtime engine's 14-gate registry stays the execution authority (same boundary as the dashboards).

## Operator setup (go-live)

1. **Discord Developer Portal** → New Application → Bot → copy token.
2. Get the Application (Client) ID from General Information.
3. OAuth2 → URL Generator: scope `bot` + `applications.commands`; permissions `Send Messages`, `Embed Links`; open the generated URL to invite the bot to your server.
4. In `infra/env/.env` (gitignored):
   ```
   DISCORD_BOT_TOKEN=<bot token>
   DISCORD_CLIENT_ID=<application id>
   DISCORD_GUILD_ID=<your server id>     # optional: instant slash registration
   DISCORD_ALERT_CHANNEL_ID=<channel id for the alert mirror>
   ```
5. Deploy: `docker compose --env-file infra/env/.env up -d discord-bot`.
6. Verify: `docker logs pat-discord-bot` → "Logged in as …" + "Registered 8 guild slash commands".

Without a token the container exits with a clear FATAL line — it never half-runs.

## Notes

- Global slash commands can take up to 1h to propagate; set `DISCORD_GUILD_ID` for instant availability during testing.
- Slash registration is idempotent (PUT) — safe on every boot.
- Discord embeds cap 4096 chars description / 1024 per field — all builders truncate.
- Graceful SIGTERM shutdown closes the pg pool.
- Alert dedup: the mirror tracks the newest signal `ID` and only posts on change; first poll seeds without posting.