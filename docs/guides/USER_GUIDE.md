# User Guide

**Version:** v1.4.0 — Color Palette + Signal Delivery + TP/SL Geometry Fix  
**Date:** 18 August 2026

---

## Getting Started

### Registration

1. Visit `https://platform.predictatrade.com/register`
2. Enter email, password, and referral code (optional)
3. Verify email
4. Choose a subscription plan
5. Download MT4/MT5 EA and Windows Agent

### Login

1. Visit `https://platform.predictatrade.com/login`
2. Enter email and password
3. If MFA is enabled, enter OTP code
4. You will be redirected to the dashboard

### Password Recovery

- Forgot password: `https://platform.predictatrade.com/forgot-password`
- Reset password: `https://platform.predictatrade.com/reset-password` (using token from email)

---

## Dashboard

The user dashboard is at `https://platform.predictatrade.com/dashboard` and includes:

### Dashboard Pages

| Page | Route | Description |
|------|-------|-------------|
| Overview | `/dashboard/overview` | Account summary, KPIs, recent activity |
| Live Chart | `/dashboard/chart` | Live XAUUSD chart with candlesticks and overlays |
| Signals | `/dashboard/signals` | Live signal feed with direction, grade, entry/SL/TP |
| Market Pulse | `/dashboard/market-pulse` | Real-time market state, regime, session |
| Positions | `/dashboard/positions` | Open and closed positions |
| Performance | `/dashboard/performance` | Trading performance metrics and history |
| Subscription | `/dashboard/subscription` | Current plan, billing, upgrade options |
| License | `/dashboard/license` | License keys, active devices, MT accounts |
| MT Setup | `/dashboard/mt-setup` | Step-by-step MT4/MT5 installation guide |
| Referral | `/dashboard/referral` | Referral network, commissions earned |
| Payouts | `/dashboard/payouts` | Payout requests and history |
| Security | `/dashboard/security` | MFA setup, password change, session management |
| Notifications | `/dashboard/notifications` | Signal and system notifications |
| Support | `/dashboard/support` | Support ticket creation and history |

---

## Signal Direction Types

The engine produces six direction types, each with distinct meaning:

| Direction | Color | Meaning | Executable? | Action Required |
|-----------|-------|---------|-------------|----------------|
| **BUY** | 🟢 Green | Qualified long — score ≥ trade threshold + all 12 gates passed | ✅ Yes | Execute via MT4/MT5 EA (if licensed) |
| **SELL** | 🔴 Red | Qualified short — score ≥ trade threshold + all 12 gates passed | ✅ Yes | Execute via MT4/MT5 EA (if licensed) |
| **BUY_CANDIDATE** | 🟡 Amber | Advisory long — score ≥ candidate threshold but below trade threshold | ❌ No | Monitor — setup forming, not yet qualified |
| **SELL_CANDIDATE** | 🟠 Orange | Advisory short — score ≥ candidate threshold but below trade threshold | ❌ No | Monitor — setup forming, not yet qualified |
| **NO-TRADE** | ⚪ Gray | Score below candidate threshold, or strategy conflict | ❌ No | No action needed |
| **BLOCKED** | ⚪ Gray | Had direction (BUY/SELL) but blocked by gate veto/safety | ❌ No | Check risk settings, data feed, or news |
| **WAIT** | ⚪ Gray | Insufficient features or MTF conflict | ❌ No | Wait for confirmation |
| **ERROR** | ⚪ Gray | System issue | ❌ No | Check data feed and agent connection |

### BUY vs BUY_CANDIDATE

- **BUY**: The strategy found sufficient evidence (score ≥ trade threshold), all 12 hard gates passed, and the signal is qualified for execution.
- **BUY_CANDIDATE**: The strategy found a meaningful directional setup (score ≥ candidate threshold) but the conviction is not strong enough for automatic execution (score < trade threshold). This is an **advisory** signal showing a setup is forming.

**Important**: BUY_CANDIDATE and SELL_CANDIDATE are NOT trade signals. They are informational only. Do not execute trades based on candidate signals.

---

## Strategies

Four independent strategies with different timeframes and risk profiles:

| Strategy | Timeframes | Threshold | Min RR | Cooldown |
|----------|-----------|-----------|--------|----------|
| **STANDARD_SCALPING** | M1/M5 + M15/M30 | 65 | 1.2 | 15m |
| **ULTRA_SCALPING** | M1 + M5 | 85 | 1.0 | 15m |
| **STANDARD_SWING** | M15/M30/H1 + H4/D1 | 55 | 1.8 | 120m |
| **TREND_SWING** | H1/H4 + D1/W1 | 50 | 2.5 | 360m |

All four evaluate independently every eligible cycle. No master strategy copies results.

---

## Probability and Grading

### Calibrated Probability (PROB)

The calibrated probability represents the model's confidence that the signal's prediction target (typically TP1 hit) will be achieved. It is computed from the raw score using a sigmoid calibration function.

**Why PROB may show "Pending"**: Until a calibration model is validated against historical data, the probability MUST be NULL (zero) per SOW §16, §36. The UI shows "Pending" to make this state clear. The raw score is always shown in the Score column — it is a weighted evidence sum (0-100+), NOT a probability.

| Calibration Status | Meaning | PROB Display |
|-------------------|---------|---------------|
| UNVERIFIED | Default models — not validated | "Pending" |
| SHADOW | Model running in shadow mode | "Pending" |
| VALIDATED | Model passed validation checks | Shows percentage |
| PROMOTED | Model promoted to production | Shows percentage |

**Important**: Calibrated probability is **NOT** a guaranteed win rate. It is a statistical estimate based on historical calibration data.

---

## TP/SL Levels (v1.4.0)

Take Profit and Stop Loss levels are computed using ATR (Average True Range) — a volatility-based measure that adapts to market conditions:

| Strategy | SL Distance | TP1 Distance | TP2 Distance | TP3 Distance |
|----------|-------------|---------------|--------------|--------------|
| STANDARD_SCALPING | 1.0 × ATR | 1.0 × ATR | 1.5 × ATR | 2.0 × ATR |
| ULTRA_SCALPING | 0.5 × ATR | 0.5 × ATR | 0.75 × ATR | 1.0 × ATR |
| STANDARD_SWING | 1.5 × ATR | 1.5 × ATR | 2.5 × ATR | 3.5 × ATR |
| TREND_SWING | 2.0 × ATR | 2.0 × ATR | 4.0 × ATR | 6.0 × ATR |

TP1 is approximately the same distance as SL (balanced R:R ≈ 1:1). TP2 and TP3 provide larger targets for trend continuation.

---

## MT4/MT5 EA Strategy Selection (v1.05)

When attaching the Predict-A-Trade EA to a chart, you can select which strategies and signal directions to receive:

### Strategy Selection
- `ReceiveStandardScalping` (default: true) — M1/M5 scalping signals
- `ReceiveUltraScalping` (default: true) — M1 ultra-fast scalping signals
- `ReceiveStandardSwing` (default: true) — M15/H1 swing trading signals
- `ReceiveTrendSwing` (default: true) — H1/H4 trend following signals

### Direction Filter
- `ReceiveBuy` (default: true) — Qualified BUY signals (executable)
- `ReceiveSell` (default: true) — Qualified SELL signals (executable)
- `ReceiveBuyCandidate` (default: true) — Advisory BUY_CANDIDATE signals
- `ReceiveSellCandidate` (default: true) — Advisory SELL_CANDIDATE signals

All are enabled by default. Set any to `false` to filter out those signals. The chart panel shows a counter: "X recv, Y shown, Z filtered" and which strategies are active: "Strats: SS US SW TW".

### Signal Grades

| Grade | Meaning |
|-------|---------|
| A+ | Highest quality setup |
| A | High quality setup |
| B | Good quality setup |
| C | Marginal setup |
| NO-TRADE/WAIT/BLOCKED/ERROR | Non-tradeable states |
| UNRATED | Not enough data to grade |

---

## MT4/MT5 Setup

### Prerequisites

- MetaTrader 4 or MetaTrader 5 terminal installed
- Active Predict-A-Trade subscription
- Valid license key
- Windows Agent running (for Master Node data)

### Installation Steps

1. **Install Windows Agent** — Download `pat-pat-agent.exe` and run on the MT4/MT5 terminal machine
2. **Install EA** — Copy `PredictATrade_MT5.mq5` (or `PredictATrade_MT4.mq4`) to `MQL5/Experts/` (or `MQL4/Experts/`)
3. **Install Master Node EA** — Copy `PredictATrade_MasterNode_MT5.mq5` (or MT4) to another chart
4. **Configure** — Enter license key in EA inputs
5. **Enable** — Enable AutoTrading in terminal
6. **AutoExecute** — Set `AutoExecute = true` for automated trading (default: false = signal display only)

### Signal Delivery

Signals are delivered via shared file (FILE_COMMON) as JSON. The EA reads the file, displays the signal, and optionally executes the trade.

### Master Node EA

The Master Node EA publishes tick data from the MT terminal to the Windows Agent via named pipe, which forwards it to the realtime engine via WebSocket.

---

## Referral System

### How It Works

1. Share your referral link/code from the Referral page
2. When someone registers with your code, they become your referral
3. You earn commissions on their subscription payments
4. View your referral network and commissions on the Referral page

### Commission Structure

Commissions are calculated based on the canonical eligible revenue policy. The commission engine is exact-decimal, transactional, and auditable. See your plan for commission rates.

### Payouts

1. Request a payout from the Payouts page
2. Payouts are processed according to your plan's payout schedule
3. View payout history and status on the Payouts page

---

## Security

### MFA (Multi-Factor Authentication)

1. Go to Security page
2. Click "Enable MFA"
3. Scan QR code with authenticator app (Google Authenticator, Authy, etc.)
4. Enter verification code to confirm
5. Save backup codes in a safe place

### Session Management

- View active sessions on Security page
- Revoke any suspicious sessions
- Session tokens are rotated automatically

### Password Security

- Change password from Security page
- Strong password requirements enforced

---

## Data Limitations

- **Volume data** is broker tick volume (proxy, not real exchange volume)
- **Institutional footprint** is not available (no DOM/Level2/Time&Sales)
- **DXY/silver/yield correlations** are not available (awaiting data feed)
- **ML is in research mode** (no AI predictions in production)
- **COT report** is not connected
- **Volume Profile / CVD**: UNSUPPORTED — cleanly disabled

---

## Support

- Create a support ticket from the Support page
- Check system status at `https://status.predictatrade.com`
- Contact support via the Support page for any issues
