# Runbook — Telegram Gold Desk (PAT Telegram AI)

Professional XAU/USD Telegram assistant: indicator engine, AI chart reads, BTC correlation, MetaTrader logging, verified signals, micro-trade desk.

## Architecture (plane-safe)

```
Telegram (long polling getUpdates)
        ▼
control: TelegramBot poller          ← TELEGRAM_BOT_TOKEN (shared with engine; engine only sends)
        ▼
TelegramAssistantEngine              ← computes indicator DISPLAY from server candles
        │  data: REALTIME_URL /candles · /signals · /cross-market/current · /agents/status
        ▼
TelegramBot.sendMessage (markdown + inline keyboards)
```

- **Plane boundaries:** the assistant never recomputes strategy scores and never talks to brokers. The realtime engine stays the execution authority (14-gate registry). Execution from chat = an *intent record* only, gated by operator ARM + the engine's veto authority.
- The realtime engine's notification adapter uses the same token SEND-only → no getUpdates conflict.

## Feature map

| Menu | Content |
|---|---|
| 📊 Indicator Engine | EMA(9/21/50) + trend, MACD(12/26/9) histogram, RSI(14) label, Bollinger(20,2) bands + %B, Volume Profile POC + 70% value area — all from server M15 candles |
| 🧠 AI Chart Read | Weighted structure vote across the indicator set (score ±4) |
| ₿ BTC Correlation | Live BTC driver + liquidity-regime desk read |
| 🖥 MetaTrader Log | Agent mesh count/online, feed health, last tick UTC |
| ✅ Verified Signals | Last 5 gate-cleared engine prints |
| 💼 Micro-Trade Desk | `plan <equity> <buy|sell> <structure stop>` → exact micro-lots, SL, TP 1R/2R/3R |
| 🛡 10% Framework | The capital stop-loss rules |

## The 10% maximum capital stop-loss framework (hard rules)

1. **Envelope = 10% of account equity** at risk per position — hard cap, not configurable above 10 from any chat surface.
2. **Stop distance defines size**: lots = envelope ÷ (stopPoints × point value). Floor to broker lot step.
3. **Micro-lot refusal**: if the computed size < broker min lot (0.01), the plan is REFUSED — never upsized to fit.
4. **Automated hedging**: requesting a hedge SPLITS the envelope — primary takes (100−hedge)%, the counter-leg takes the remainder, so **combined exposure ≤ 10% always** (e.g. 30% hedge → 70/30 split).
5. **Armable execution**: with `TELEGRAM_TRADING_ARMED=true`, approved plans emit execution *intents* into the platform gate flow. The realtime engine's 14-gate registry holds final veto authority (exposure, margin, session, news, entitlement, license) — NO-TRADE remains a valid outcome. Kill-switch/emergency-stop always win.

## Operator setup / state

Env (in `infra/env/control.env`):
```
TELEGRAM_BOT_TOKEN=<shared bot token>
TELEGRAM_ASSISTANT_ENABLED=true      # opt-in: starts the poller
TELEGRAM_ASSISTANT_CHAT_IDS=<operator chat id(s), comma-separated>
TELEGRAM_TRADING_ARMED=false         # execution intents LOCKED until armed
```

Status check (admin): `GET /api/v1/admin/telegram/status` → configured / enabled / armed.
Broadcast (admin): `GET /api/v1/admin/telegram/broadcast?text=...` → pushes to operator chats.

## Arming execution (operator decision)

Set `TELEGRAM_TRADING_ARMED=true` + restart control. Verify via `/admin/telegram/status` → `tradingArmed: true`. Disarm any time by flipping back to `false` (intents stop instantly).

## Monitoring

- Poller: `docker logs pat-control | grep '\[TG\]'` (poll errors would surface here)
- Bot identity: `curl https://api.telegram.org/bot$TOKEN/getMe`
- Test send: use the admin broadcast endpoint.

## Test coverage

23 tests: indicator math vs known answers (EMA seed/smooth, RSI extremes, MACD convergence on linear ramps — float-noise tolerance, Bollinger %B, volume profile POC/value area), full-snapshot trend labels, 10% framework math (exact envelope fill, step rounding, min-lot refusal, envelope-split hedging with combined ≤10%), keyboard flows, and the persona rule.