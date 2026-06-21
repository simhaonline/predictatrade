# Predict-A-Trade — Institutional-Grade AI Trading Assistant

---

## 1. Project Overview

Predict-A-Trade is an institutional-grade, AI-driven trading ecosystem designed for XAUUSD / Gold Spot market analysis, decision support, broker connectivity, and controlled trade execution.

The platform will combine real-time market data, MetaTrader MT4/MT5 integration, multi-agent AI analysis, computer vision chart interpretation, technical strategy validation, risk management, and optional financial-astrology timing intelligence into a unified web-based trading dashboard.

The system will be optimized for Dubai, UAE timezone operations using GST / UTC+4 session logic, with primary focus on Tokyo, London, New York, and high-liquidity XAUUSD trading windows.

---

## 2. Project Objective

The objective of this Scope of Work is to define the full development, integration, testing, validation, and deployment requirements for a professional AI trading assistant capable of:

* Monitoring real-time XAUUSD market conditions.
* Reading and analyzing live chart data.
* Validating technical trade setups using multi-agent logic.
* Applying structured risk and execution rules before any trade action.
* Connecting with MT4/MT5 for broker-side synchronization and execution.
* Displaying all signals, risk status, system health, and trade decisions through a professional web dashboard.
* Operating safely across Observe, Backtest, Paper, Demo, and Live modes.
* Maintaining full auditability, logs, rejection reasons, and decision transparency.

---

## 3. Core System Pillars

### 3.1 MetaTrader MT4/MT5 Bridge

The system shall include a reliable broker bridge for MT4 and MT5 integration.

The bridge must support:

* Real-time price feed synchronization.
* Account balance, equity, margin, and open-position monitoring.
* Order placement in Demo and Live modes only.
* Trade modification, stop-loss, take-profit, and partial-close handling.
* Slippage protection.
* Spread monitoring.
* Broker connection status reporting.
* Execution confirmation and error handling.

---

### 3.2 Autonomous AI Agent System

The platform shall use a multi-agent architecture where each agent performs a dedicated analytical, risk, execution, or monitoring function.

The AI system must support:

* Real-time decision routing.
* Independent agent scoring.
* Multi-agent consensus validation.
* Confidence-based trade approval.
* Reject reason generation.
* Risk-aware trade filtering.
* Human-readable explanation for every accepted or rejected setup.

---

### 3.3 Vision AI Chart Analysis

The system shall integrate a Vision AI layer capable of reading live trading charts and validating visible technical structures.

The Vision AI module must analyze:

* Candlestick formations.
* OHLC bar structure.
* Line chart direction.
* Uptrend and downtrend behavior.
* Support and resistance zones.
* Liquidity sweeps.
* Break of Market Structure.
* Change of Character.
* Fair Value Gaps.
* Order Blocks.
* Continuation patterns.
* Reversal patterns.
* Fibonacci zones.
* Chart geometry and visual pattern consistency.

---

### 3.4 Technical Analysis Engine

The system shall include a technical analysis engine focused on institutional-style XAUUSD trading logic.

The technical engine must support:

* Market structure detection.
* HTF and LTF bias alignment.
* Smart Money Concepts.
* Liquidity pool identification.
* Buy-side liquidity and sell-side liquidity mapping.
* Fair Value Gap detection.
* Order Block validation.
* Fibonacci Premium / Discount analysis.
* Optimal Trade Entry zone calculation.
* DXY and US10Y correlation filtering.
* Volatility and session-based condition filtering.

---

### 3.5 Optional Financial-Astrology Timing Engine

The platform shall include an optional alternative timing module for financial-astrology bias analysis.

This module must be treated as an auxiliary bias engine and must not override mandatory risk-management controls.

The astrology engine may include:

* Vedic astrology cycles.
* KP / Krishnamurti Paddhati timing logic.
* Western planetary aspects.
* Chinese astrology cycle references.
* Lunar phase analysis.
* Numerology-based timing references.
* Dubai timezone-based daily bias windows.
* Historical market-cycle correlation logging.

---

### 3.6 Web-Based Trading Dashboard

The system shall include a professional web dashboard for monitoring, analysis, control, and audit.

The dashboard must provide:

* Real-time XAUUSD price display.
* Live session and killzone status.
* Market bias summary.
* Active AI agent status.
* Gate decision status.
* Open trade monitoring.
* Risk exposure panel.
* Broker connection status.
* News blackout status.
* Astro/KP timing status.
* Vision AI validation result.
* Rejected trade registry.
* Accepted trade registry.
* System health and latency monitoring.
* Light and dark mode support.
* Responsive desktop and mobile layout.

---

## 4. Multi-Agent System Architecture

The system shall be organized into the following agent layers.

---

### 4.1 Data & Session Layer

| Agent ID | Agent Name               | Responsibility                                                                                                        |
| -------- | ------------------------ | --------------------------------------------------------------------------------------------------------------------- |
| AGT-01   | Market Data Agent        | Ingests real-time XAUUSD price data, volatility, spread, and tick information.                                        |
| AGT-02   | Session / Killzone Agent | Tracks Tokyo, London, New York, and Dubai GST liquidity windows.                                                      |
| AGT-07   | News Agent               | Processes macroeconomic events, USD news, inflation data, central bank updates, and high-impact event blackout rules. |
| AGT-18   | Health Monitor Agent     | Tracks system uptime, broker connectivity, latency, API status, and internal service health.                          |

---

### 4.2 Alternative Bias Layer

| Agent ID | Agent Name              | Responsibility                                                                  |
| -------- | ----------------------- | ------------------------------------------------------------------------------- |
| AGT-03   | Western Astrology Agent | Reviews planetary aspects, transits, and market-cycle correlations.             |
| AGT-04   | KP / Vedic Agent        | Applies KP and Vedic timing logic for optional trade-cycle bias.                |
| AGT-06   | Lunar Agent             | Reviews lunar phase, apogee/perigee, and short-term sentiment cycle references. |

---

### 4.3 Technical Analysis Layer

| Agent ID | Agent Name              | Responsibility                                                                   |
| -------- | ----------------------- | -------------------------------------------------------------------------------- |
| AGT-08   | Structure / Bias Agent  | Detects HTF direction, BMS, CHoCH, and structural trend bias.                    |
| AGT-09   | Liquidity Agent         | Identifies buy-side liquidity, sell-side liquidity, inducement, and sweep zones. |
| AGT-10   | FVG Agent               | Detects Fair Value Gaps, imbalance quality, and price inefficiency.              |
| AGT-11   | Order Block Agent       | Validates institutional supply and demand zones.                                 |
| AGT-12   | Fibonacci Agent         | Calculates Premium / Discount zones and Optimal Trade Entry areas.               |
| AGT-13   | Correlation Agent       | Reviews DXY, US10Y, and other gold-sensitive correlation inputs.                 |
| AGT-17   | Vision Validation Agent | Uses Vision AI to confirm chart geometry and visible technical setup quality.    |

---

### 4.4 Execution & Risk Layer

| Agent ID | Agent Name              | Responsibility                                                                     |
| -------- | ----------------------- | ---------------------------------------------------------------------------------- |
| AGT-14   | Risk / Lot Sizing Agent | Calculates position size, risk percentage, drawdown limits, and margin safety.     |
| AGT-15   | Execution Agent         | Handles MT4/MT5 order routing, slippage control, and execution response handling.  |
| AGT-16   | Trade Management Agent  | Manages trailing stop, break-even movement, partial close, and TP/SL updates.      |
| AGT-19   | Notification Agent      | Sends alerts through dashboard, webhook, email, Telegram, or mobile push channels. |

---

## 5. Gate Decision Cascade Engine

Every trade setup must pass through a strict decision cascade before execution.

No order shall be sent to MT4/MT5 unless all mandatory gates pass.

---

### 5.1 Gate Flow

```text
[Trade Setup Triggered]
        |
        |-- Gate 01: Circuit Breaker
        |-- Gate 02: Risk Management
        |-- Gate 03: News Blackout
        |-- Gate 04: Spread / Broker Status
        |-- Gate 05: Session Validation
        |-- Gate 06: Correlation Filter
        |-- Gate 07: Market Structure
        |-- Gate 08: Liquidity Sweep
        |-- Gate 09: FVG Quality Check
        |-- Gate 10: Fibonacci Zone
        |-- Gate 11: Order Block Validation
        |-- Gate 12: Astro / KP Cycle Alignment
        |-- Gate 13: Multi-Agent Confidence
        |-- Gate 14: Vision AI Validation
        |
[Execute Order via MT4/MT5]
```

---

### 5.2 Gate Specifications & Reject Registry

| Sequence | Gate Name              | Target Subsystem         | Failure Code     | Description                                                                                   |
| -------- | ---------------------- | ------------------------ | ---------------- | --------------------------------------------------------------------------------------------- |
| 01       | Circuit Breaker        | AGT-18                   | CB_ACTIVE        | Rejects trades when drawdown limits, system health rules, or emergency locks are active.      |
| 02       | Risk Management        | AGT-14                   | RISK_FAIL        | Rejects trades if margin, exposure, position size, or risk percentage violates configuration. |
| 03       | News Blackout          | AGT-07                   | NEWS_BLACKOUT    | Blocks trading during restricted high-impact USD/XAU news windows.                            |
| 04       | Spread / Broker Status | AGT-15                   | SPREAD_FAIL      | Rejects trades if spread, slippage, broker status, or execution quality is unacceptable.      |
| 05       | Session Validation     | AGT-02                   | SESSION_FAIL     | Blocks trades outside approved Dubai GST trading sessions and killzones.                      |
| 06       | Correlation Filter     | AGT-13                   | CORRELATION_FAIL | Rejects trades when DXY, US10Y, or related inputs conflict with XAUUSD direction.             |
| 07       | Market Structure       | AGT-08                   | STRUCTURE_FAIL   | Rejects trades that do not align with HTF order flow or structural bias.                      |
| 08       | Liquidity Sweep        | AGT-09                   | NO_SWEEP         | Rejects setups without a valid liquidity sweep or institutional trigger.                      |
| 09       | FVG Quality            | AGT-10                   | FVG_FAIL         | Rejects weak or invalid Fair Value Gap setups.                                                |
| 10       | Fibonacci Zone         | AGT-12                   | FIB_FAIL         | Rejects trades outside Premium / Discount or Optimal Trade Entry zones.                       |
| 11       | Order Block Validation | AGT-11                   | OB_INVALID       | Rejects trades if price is not reacting from a validated Order Block.                         |
| 12       | Astro / KP Alignment   | AGT-03 / AGT-04 / AGT-06 | ASTRO_FAIL       | Optional reject or warning based on alternative timing-cycle conflict.                        |
| 13       | Multi-Agent Confidence | Orchestrator             | CONFIDENCE_LOW   | Rejects trades when total confidence score is below the configured threshold.                 |
| 14       | Vision AI Validation   | AGT-17                   | VISION_FAIL      | Rejects trades when chart-image validation does not confirm the setup.                        |

---

## 6. System Execution Modes

The system shall support five execution modes.

The default mode must always be `OBSERVE`.

Live trading must remain locked unless explicitly enabled by authorized configuration and administrative approval.

---

### 6.1 Execution Mode Matrix

| Mode     | Capital Type             | Order Execution           | Primary Purpose                                                                                  | Operational Status         |
| -------- | ------------------------ | ------------------------- | ------------------------------------------------------------------------------------------------ | -------------------------- |
| OBSERVE  | None                     | Disabled                  | Watch market, run agents, log simulated decisions, and monitor signals without order generation. | Default Safe Mode          |
| BACKTEST | Simulated                | Historical Simulator      | Test strategies on historical M1, tick, and session data.                                        | Research Mode              |
| PAPER    | Simulated                | Mock Routing              | Forward-test live signals without broker execution.                                              | Validation Mode            |
| DEMO     | Simulated Broker Capital | MT4/MT5 Demo Gateway      | Validate broker bridge, execution speed, slippage, and trade lifecycle.                          | Broker Testing Mode        |
| LIVE     | Real Capital             | MT4/MT5 Production Bridge | Execute real trades only after all gates pass and live lock is authorized.                       | Restricted Production Mode |

---

## 7. Dashboard Scope

The dashboard shall be developed as a modern, responsive, professional trading interface.

---

### 7.1 Dashboard Pages

The dashboard must include the following pages or sections:

* Overview Dashboard
* Live Market Data
* AI Agent Console
* Gate Decision Monitor
* Signal Center
* Trade Execution Panel
* Risk Management Panel
* MT4/MT5 Broker Status
* Vision AI Chart Analysis
* News & Economic Calendar
* Astro / KP Bias Dashboard
* Backtest Results
* Paper Trading Results
* Demo Trading Results
* Live Trading Control
* System Logs
* Audit Trail
* Settings & Configuration
* User Management
* Notification Settings

---

### 7.2 Dashboard UI/UX Requirements

The dashboard must be:

* Professional and institutional-grade.
* Clean, modern, and elegant.
* Responsive for desktop, tablet, and mobile.
* Optimized for light and dark mode.
* Built with clear visual hierarchy.
* Designed for real-time trading decisions.
* Capable of showing alerts, warnings, failures, and approvals clearly.
* Easy to navigate during fast market conditions.

---

## 8. API Scope

The backend must expose structured APIs for all major platform functions.

---

### 8.1 Required API Modules

* Authentication API
* User Management API
* Market Data API
* Broker Bridge API
* Signal API
* AI Agent API
* Gate Decision API
* Risk Management API
* Trade Execution API
* Trade Management API
* Backtesting API
* Paper Trading API
* Demo Trading API
* Live Trading Control API
* News API
* Astrology Bias API
* Vision AI API
* Notification API
* Audit Log API
* System Health API
* Configuration API

---

## 9. Database Scope

The database shall store all operational, analytical, trading, audit, and configuration data.

---

### 9.1 Required Database Areas

The database must support:

* Users and roles.
* API keys and permissions.
* Broker account mappings.
* Market data snapshots.
* Agent outputs.
* Signal history.
* Gate decisions.
* Rejected setup logs.
* Accepted setup logs.
* Trade lifecycle events.
* Risk calculations.
* Backtest results.
* Paper trading results.
* Demo trading results.
* Live trading records.
* News events.
* Astro/KP bias records.
* Vision AI validation records.
* System health logs.
* Error logs.
* Audit trails.
* Configuration versioning.

---

## 10. Risk Management Scope

Risk management is mandatory and must override all other system logic.

The system must include:

* Maximum daily drawdown limit.
* Maximum weekly drawdown limit.
* Maximum open-risk limit.
* Maximum lot-size limit.
* Maximum trades per day.
* Maximum consecutive losses rule.
* Minimum risk-reward ratio.
* Spread threshold.
* Slippage threshold.
* Margin safety check.
* Equity protection rule.
* News blackout lock.
* Manual emergency stop.
* Live trading lock.
* Account-level exposure cap.
* Symbol-level exposure cap.

---

## 11. Trade Execution Scope

The execution engine shall manage the full trade lifecycle.

---

### 11.1 Execution Requirements

The execution system must support:

* Buy order.
* Sell order.
* Pending order.
* Market order.
* Stop-loss placement.
* Take-profit placement.
* Trailing stop.
* Break-even movement.
* Partial close.
* Full close.
* Order modification.
* Order cancellation.
* Broker rejection handling.
* Retry logic.
* Execution latency logging.
* Slippage logging.
* Trade confirmation logging.

---

## 12. Vision AI Scope

The Vision AI system shall support chart-reading and technical validation.

---

### 12.1 Vision AI Input Sources

Vision AI must support:

* Uploaded chart screenshots.
* Live dashboard chart captures.
* MT4/MT5 screenshot captures.
* TradingView screenshot captures where applicable.
* OHLC chart visualization snapshots.

---

### 12.2 Vision AI Validation Outputs

Vision AI must return:

* Detected trend direction.
* Support and resistance zones.
* Liquidity zones.
* Pattern recognition result.
* Candle structure assessment.
* Market structure validation.
* FVG presence or absence.
* Order Block presence or absence.
* Fibonacci zone confirmation.
* Trade setup confidence score.
* Invalid setup reason, if applicable.

---

## 13. News & Macro Scope

The system shall monitor high-impact market events that may affect XAUUSD.

The news module must include:

* USD high-impact news.
* Federal Reserve events.
* CPI, PPI, NFP, FOMC, GDP, unemployment, and interest-rate events.
* Gold-sensitive geopolitical headlines.
* Risk-on / risk-off sentiment monitoring.
* Configurable blackout window before and after events.
* Dashboard warning status.
* Trade rejection during restricted news periods.

---

## 14. Astrology / Alternative Bias Scope

The astrology module shall be optional and configurable.

It must never bypass mandatory technical, execution, or risk controls.

The module may include:

* Daily XAUUSD astrology bias.
* Vedic planetary transit bias.
* Western planetary aspect bias.
* KP event-timing references.
* Chinese astrology cycle bias.
* Lunar cycle bias.
* Numerology timing bias.
* Dubai GST timezone alignment.
* Bullish, bearish, neutral, and caution classifications.
* Historical comparison logging.
* Confidence weighting controls.

---

## 15. Security Scope

The system must be designed with strict operational security.

Security requirements include:

* Secure authentication.
* Role-based access control.
* Admin-only live trading unlock.
* API key encryption.
* Broker credential encryption.
* Environment-variable protection.
* Audit log protection.
* Session timeout.
* Failed-login protection.
* Secure deployment configuration.
* Production and demo environment separation.
* Manual emergency shutdown.
* Live mode confirmation flow.

---

## 16. Audit & Logging Scope

Every material system decision must be logged.

The audit system must record:

* User login activity.
* Configuration changes.
* Agent outputs.
* Signal generation.
* Gate pass/fail status.
* Reject reason codes.
* Risk calculations.
* Broker connection events.
* Order placement attempts.
* Order success or failure.
* Trade modification.
* Trade closure.
* System errors.
* API errors.
* AI model responses.
* Vision AI results.
* Manual override actions.
* Live mode unlock events.

---

## 17. Notification Scope

The platform shall support real-time notification delivery.

Notifications may include:

* New signal generated.
* Trade setup rejected.
* Trade setup approved.
* Demo trade executed.
* Live trade executed.
* Stop-loss hit.
* Take-profit hit.
* Partial close completed.
* Break-even moved.
* Broker disconnected.
* Spread too high.
* News blackout active.
* Risk limit reached.
* Circuit breaker activated.
* System health degraded.

Notification channels may include:

* Dashboard alerts.
* Email.
* Telegram.
* Webhook.
* Mobile push notification, if supported.

---

## 18. Deliverables

The project deliverables shall include:

* Complete frontend dashboard.
* Complete backend API.
* Database schema and migrations.
* MT4/MT5 bridge integration.
* Multi-agent orchestration engine.
* Technical analysis engine.
* Vision AI integration.
* Optional astrology bias engine.
* Risk management engine.
* Gate decision cascade engine.
* Trade execution engine.
* Trade management engine.
* News blackout module.
* Notification module.
* Audit logging system.
* Configuration management system.
* Backtesting module.
* Paper trading module.
* Demo trading module.
* Live trading lock module.
* Deployment scripts.
* Environment configuration templates.
* README.md documentation.
* Installation guide.
* Operations guide.
* API documentation.
* Testing report.
* Final audit report.

---

## 19. Acceptance Criteria

The work shall be considered complete only when the following acceptance criteria are met.

---

### 19.1 Functional Acceptance

* All dashboard pages are implemented and accessible.
* All AI agents are registered and operational.
* All gate decisions return pass/fail status correctly.
* Reject reason codes are stored and displayed.
* MT4/MT5 bridge connects successfully in Demo mode.
* Demo trade execution works end-to-end.
* Risk management blocks invalid trades.
* News blackout blocks restricted trading periods.
* Vision AI returns structured chart validation output.
* Astro/KP module works as optional bias input.
* Backtest mode runs without broker dependency.
* Paper mode generates simulated live results.
* Observe mode does not send any order payload.
* Live mode remains locked unless explicitly unlocked.

---

### 19.2 Technical Acceptance

* APIs return structured and documented responses.
* Database migrations run cleanly.
* Logs are stored correctly.
* Audit trail captures critical events.
* Error handling is implemented.
* Configuration is environment-based.
* Services start successfully.
* Health checks pass.
* No hardcoded secrets exist.
* Codebase is clean and organized.
* Unused files are removed.
* README.md is complete and accurate.

---

### 19.3 UI/UX Acceptance

* Dashboard is professional and visually polished.
* Light mode works correctly.
* Dark mode works correctly.
* Layout is responsive.
* Key trading information is visible without clutter.
* Status indicators are clear.
* Warning and rejection messages are easy to understand.
* Charts, cards, panels, and tables are aligned and consistent.
* Navigation is smooth and logical.

---

### 19.4 Trading Safety Acceptance

* Live trading is disabled by default.
* Circuit breaker works.
* Risk limits work.
* Spread filter works.
* Slippage filter works.
* News blackout works.
* Manual emergency stop works.
* No trade can bypass the gate cascade.
* All live-order attempts are logged.
* All rejected trades include clear reasons.

---

## 20. Out of Scope

The following items are excluded unless approved separately:

* Guarantee of trading profit.
* Fully autonomous live trading without administrative unlock.
* Circumvention of broker rules.
* Financial advisory or investment advisory services.
* Regulatory licensing services.
* Custody of client funds.
* Broker account creation.
* Guaranteed accuracy of astrology-based predictions.
* Guaranteed performance of external AI model providers.
* Guaranteed uninterrupted third-party API availability.

---

## 21. Compliance & Risk Disclaimer

The platform is a trading technology and decision-support system.

It does not provide guaranteed profits, personalized financial advice, or investment advisory services.

All trading involves substantial risk, including the possible loss of capital.

Live trading functionality must remain restricted, audited, and manually controlled.

The astrology and numerology modules are alternative analytical inputs only and must not be treated as guaranteed predictive systems or standalone execution triggers.

---

## 22. Final Completion Requirements

Before final handover, the development team must perform a full deep audit covering:

* Codebase structure.
* Frontend UI/UX.
* Backend APIs.
* Database schema.
* MT4/MT5 bridge.
* AI agent orchestration.
* Vision AI integration.
* Astrology bias engine.
* Risk engine.
* Gate cascade.
* Trade execution flow.
* Trade management flow.
* Logs and audit trail.
* Security configuration.
* Environment files.
* Deployment scripts.
* README.md.
* Installation guide.
* Operations guide.
* Testing report.
* Final issue-resolution report.

The final handover must confirm that the system is stable, documented, secure, operational, and ready for controlled Demo-mode validation before any Live-mode activation.
