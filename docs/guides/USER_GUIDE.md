# User Guide
## v1.16.0 — 26 August 2026

### Overview

Predict-A-Trade generates XAUUSD trading signals using 5 strategy engines, 42 technical indicators, and 16 risk gates. This guide covers the user dashboard, signal interpretation, account management, and Windows/MT4-MT5 setup.

---

## 1. Getting Started

### Registration
1. Go to `https://platform.predictatrade.com/signup`
2. Enter email, password, and accept Terms of Service + Privacy Policy
3. Verify email (check inbox for verification link)
4. Log in at `https://platform.predictatrade.com/login`

### Plan Tiers
| Plan | Price | Strategies | Signals/Day | Features |
|------|:-----:|:---------:|:-----------:|----------|
| FREE | $0 | Standard Scalping | 2 | Basic dashboard, delayed signals |
| STANDARD | $99/mo | All 5 engines | Unlimited | Real-time signals, indicators, analysis |
| PRO | $299/mo | All 5 engines | Unlimited | + Backtesting, priority support, ML insights |
| ELITE | $699/mo | All 5 engines | Unlimited | + Personal account manager, API access, custom strategies |

---

## 2. Dashboard

### Live Dashboard (`/dashboard/live`)
Real-time trading view showing:
- **Current Price:** XAUUSD bid/ask spread
- **Active Signals:** Live signals from all enabled strategies — displayed in the Signal Pipeline card
- **Signal Cards:** Direction (BUY/SELL), entry price, SL, TP1/TP2/TP3, grade (A+/A/B), score, R:R per TP level
- **Engine Status:** Which strategies are generating signals
- **Market Regime:** Current market condition (TRENDING_BULLISH, TRENDING_BEARISH, RANGE)
- **Session:** Active trading session (TOKYO/LONDON/NY/OVERLAP)

#### Signal Card Example
```
┌─────────────────────────────────────────┐
│ STANDARD SCALPING          Grade: A     │
│ BUY XAUUSD                 Score: 72.5  │
│ Entry: 2430.50                          │
│ SL:    2425.00  TP1: 2442.00  R:R 2.0x  │
│                TP2: 2453.75  R:R 3.1x  │
│                TP3: 2465.50  R:R 4.7x  │
│ Regime: TRENDING_BULLISH                │
│ Expires: 10:10 UTC                      │
└─────────────────────────────────────────┘
```

### Signals Page (`/dashboard/signals`)
Complete signal table with per-strategy filtering and evidence chain expansion:
- **Table columns:** Direction, Strategy, Score, Calibrated Probability, Entry, SL, TP1, TP2, TP3, Quality Grade, Status, Regime, Date
- **Client-side pagination:** 15 signals per page with Prev/Next navigation (prevents browser lockup with large signal volumes)
- **Expand rows:** Click any signal to view full evidence chain, lot sizing, risk metrics
- **Filters:** Strategy, Direction, Regime
- **Color coding:** BUY (green), SELL (red), BUY_CANDIDATE/SELL_CANDIDATE (amber), NO-TRADE (grey)
- **Export to CSV:** Download filtered signal history

### Signal Status Lifecycle
1. **DETECTED** — Strategy identified a setup
2. **QUALIFIED** — Passed all 16 risk gates
3. **EXECUTED** — Sent to your MT4/MT5 for execution
4. **CLOSED** — Trade closed with outcome (WIN/LOSS/BREAKEVEN)
5. **EXPIRED** — Signal timed out before execution

---

## 3. Understanding Strategies

### Standard Scalping (M1/M5)
- **Personality:** Quick trades, held minutes not hours
- **Best during:** London and NY sessions
- **Min score:** 65
- **Expiry:** 10 minutes (act fast)

### Ultra Scalping (M1)
- **Personality:** Ultra-fast, very short holds
- **Best during:** High volatility periods
- **Min score:** 60
- **Expiry:** 5 minutes

### Standard Swing (M15/H1)
- **Personality:** Medium-term, held 30 min to hours
- **Best during:** Clear trends with pullbacks
- **Min score:** 68
- **Expiry:** 30 minutes

### Trend Swing (H1/H4)
- **Personality:** Long-term trend following
- **Best during:** Strong trending markets
- **Min score:** 70
- **Expiry:** 60 minutes

### MARNIE_FIB (H1) — SHADOW
- **Personality:** Fibonacci confluence trader
- **Status:** Shadow mode — signals for observation only, not live trading
- **Expiry:** 60 minutes

### Signal Grades
| Grade | Meaning | Action |
|:-----:|---------|--------|
| **A** | Strong confluence, high confidence | Preferred entry |
| **B** | Good setup, moderate confidence | Consider with caution |
| **C** | Marginal setup, low confidence | Observe or skip |

---

## 4. Evidence & Scoring

Every signal has a full evidence breakdown showing WHY it was generated:

- **Trend:** EMA alignment, ADX direction
- **Momentum:** MACD, RSI, Stoch crossover
- **Structure:** Support/resistance breaks (BOS/CHoCH)
- **Liquidity:** Stop hunts, liquidity sweep detection
- **SMC:** Fair value gaps (FVG), imbalances
- **Candles:** Rejection wicks, pin bars, engulfing patterns
- **Regime:** Market condition filter
- **VWAP:** Price relative to volume-weighted average
- **P2 Features:** Opening range breakouts, pullback depth, pin bar quality

View the evidence chain by clicking any signal in the dashboard.

---

## 5. Account Management

### Profile (`/dashboard/settings`)
- Update name, email, phone
- Change password
- Enable/disable MFA (TOTP authenticator app)

### MFA Setup
1. Go to Settings → Security → Enable MFA
2. Scan QR code with authenticator app (Google Authenticator, Authy)
3. Enter verification code to confirm
4. Save backup codes in a secure location

### Subscription (`/dashboard/billing`)
- View current plan and renewal date
- Upgrade/downgrade plan
- View billing history (invoices, payments)
- Cancel subscription (takes effect at period end)

### Referrals (`/dashboard/referrals`)
- Your unique referral code
- Earnings summary (total earned, pending, paid)
- Referral tree (who signed up under your code)
- Commission rates: Standard 10%, Pro 15%, Elite 18%

---

## 6. Windows Agent & MT4/MT5 Setup

### Prerequisites
- Windows 10/11 (64-bit)
- MetaTrader 4 or MetaTrader 5 installed
- Active Predict-A-Trade subscription (STANDARD tier or higher)
- Broker account with XAUUSD symbol

### Installation Steps

1. **Download the Windows Agent**
   - From your dashboard: Settings → Downloads → Windows Agent
   - Or: `https://platform.predictatrade.com/downloads/windows-agent`

2. **Install and configure**
   ```bash
   # Run the installer
   PredictATrade-Agent-Setup.exe

   # Or manual setup
   unzip windows-agent.zip
   cd windows-agent
   ```

3. **Enter your license key**
   - Find your license key at: Dashboard → Settings → License
   - Enter in the Agent configuration window

4. **Connect to MT4/MT5**
   - Agent auto-detects running MT4/MT5 terminals
   - Select your terminal from the dropdown
   - Verify XAUUSD symbol is available
   - Click "Connect"

5. **Verify connection**
   - Agent status shows "Connected" with green indicator
   - Dashboard shows "Agent Online" with device name
   - Test connection: Agent sends heartbeat every 30s

### MQL Expert Advisor Installation
The Windows Agent installs the MQL EA automatically. Manual steps:
1. Copy `mql/mt5/Experts/PAT_SignalExecutor.ex5` to your MT5 Experts folder
2. Copy `mql/mt4/Experts/PAT_SignalExecutor.ex4` to your MT4 Experts folder
3. Restart MetaTrader
4. Attach EA to XAUUSD chart (M1 timeframe)
5. Enable "Allow Automated Trading" in MT4/MT5

### Agent Features
- **Auto-trade mode:** EA executes signals automatically (configurable)
- **Manual mode:** Agent shows signals, you place trades manually
- **Risk controls:** Max lot size, max spread, daily loss cap
- **Capital protection:** EA auto-closes all on daily loss limit breach
- **SL enforcement:** Server verifies stop losses are set correctly
- **Emergency commands:** CLOSE_POSITION, EMERGENCY_STOP, KILL_SWITCH from server

### Safety Features (mandatory, cannot be disabled)
| Feature | Behaviour |
|---------|-----------|
| Server-side SL | Server verifies SL is set; closes position if missing |
| Max daily loss | EA closes all positions if daily loss cap hit |
| Max spread gate | Signals blocked if spread exceeds limit |
| Slippage guard | Post-fill slippage check, reports violations |
| Margin check | OrderCalcMargin before every order |
| Martingale ban | MaxLotRatioVsBase = 1.0 (no doubling) |
| License enforcement | EA checks license status, fails closed |

---

## 7. Backtesting (`/dashboard/backtest`)

Available on PRO and ELITE plans.

### Running a Backtest
1. Select strategy (e.g., Standard Scalping)
2. Choose date range
3. Set parameters: starting capital, risk %, commission
4. Click "Run Backtest"

### Reading Results
- **Total Trades:** Number of signals generated
- **Win Rate:** Percentage of profitable trades
- **Profit Factor:** Gross profit / gross loss (>1.0 is profitable)
- **Max Drawdown:** Largest peak-to-trough decline
- **Sharpe Ratio:** Risk-adjusted return (>1.0 is good)
- **Equity Curve:** Visual chart of account growth

---

## 8. Trading Reports (`/dashboard/trading-reports`)

- **Performance Summary:** Win rate, total P&L, average R:R
- **Per-Strategy Breakdown:** Which strategies perform best
- **Session Analysis:** Performance by trading session
- **Regime Analysis:** Performance by market condition
- **Daily Summary:** Trades per day, daily P&L
- Export reports to CSV/PDF

---

## 9. Notifications

Configure alerts at Settings → Notifications:

| Channel | Available on | Example |
|---------|:-----------:|---------|
| Email | All plans | Signal alerts to your inbox |
| Telegram | STANDARD+ | Instant signal notifications via bot |
| Push | STANDARD+ | Browser push notifications |
| ntfy | STANDARD+ | Self-hosted notification server |

Alert types:
- **New Signal:** When a grade A/B signal fires
- **Signal Executed:** When your EA executes a trade
- **Trade Closed:** Outcome notification (WIN/LOSS)
- **Daily Summary:** End-of-day performance recap
- **System Alerts:** Engine degradation, maintenance windows

---

## 10. Troubleshooting

### No signals appearing
- Check Engine Status on dashboard (DEGRADED = market data issue)
- Verify your plan tier includes active strategies
- Check if market is closed (weekend, holiday)
- Verify MT4/MT5 connection if using auto-trade

### Agent won't connect
- Verify license key is valid (Dashboard → Settings → License)
- Check Windows firewall allows outbound WebSocket (port 443)
- Ensure internet connection is stable
- Restart Windows Agent as administrator

### MT4/MT5 EA not working
- Confirm "Allow Automated Trading" is enabled
- Check EA is attached to XAUUSD M1 chart
- Look for smiley face icon in MT4/MT5 (green = active)
- Check Experts tab for error messages

### Support
- Email: support@predictatrade.com
- Dashboard: Help → Contact Support
- Knowledge base: https://predictatrade.com/docs
