# Predict-A-Trade v1.0.0 — Four Strategy Playbooks

## SOW Section 12A — Canonical Strategy Definitions

All four strategies are separate quantitative products with DISTINCT versioned behavior.
They are NOT aliases pointing to one generic signal.

### STANDARD_SCALPING (v1.0.0 seed)
- **Purpose:** Selective intraday scalp (NOT HFT)
- **Holding horizon:** 1–15 minutes
- **Frequency target:** 5–15 candidates/week (NOT a quota)
- **Entry timeframe:** M5
- **Execution timing:** M1/tick when healthy
- **HTF context:** H1 + M30
- **Min gross R:R:** 1.20
- **Max spread:** $0.35 + relative/cost gates
- **Signal TTL:** 15 minutes
- **Stop model:** Structure + volatility buffer ~1.0 ATR(M5)
- **Confluence threshold:** 75/100, separation 20
- **Mandatory pillars:** liquidity, structure

### ULTRA_SCALPING (v1.0.0 seed)
- **Purpose:** Very short-duration, cost-sensitive microstructure trading (NOT HFT)
- **Holding horizon:** 10 seconds–3 minutes
- **Frequency target:** 15–40 candidates/week (NOT a quota)
- **Entry timeframe:** M1
- **Execution timing:** tick/sub-minute if authoritative
- **HTF context:** M15 + M5
- **Min gross R:R:** 1.00 (only with validated positive net expectancy)
- **Max spread:** $0.25 hard seed + stricter relative/cost gates
- **Signal TTL:** 3 minutes
- **Stop model:** Tight structure + volatility buffer ~0.7 ATR(M1)
- **Confluence threshold:** 85/100, separation 25
- **Mandatory pillars:** flow_microstructure, liquidity_event, execution_cost_quality

### STANDARD_SWING (v1.0.0 seed)
- **Purpose:** Multi-hour swing trading
- **Holding horizon:** 4–48 hours
- **Frequency target:** 2–5 trades/week (NOT a quota)
- **Entry timeframe:** M15/H1
- **HTF context:** H4 + D1
- **Min gross R:R:** 1.80
- **Max spread:** $0.45 + relative/cost gates
- **Signal TTL:** 4 hours
- **Stop model:** Structure + volatility buffer ~1.5 ATR(H1)
- **Confluence threshold:** 70/100, separation 15
- **Mandatory pillars:** d1_h4_structure, macro_dxy_yield

### TREND_SWING (v1.0.0 seed)
- **Purpose:** Multi-day trend following
- **Holding horizon:** 1–10 days
- **Frequency target:** 2–6 trades/month (NOT a quota)
- **Entry timeframe:** H1/H4
- **HTF context:** D1 + W1
- **Min gross R:R:** 2.50
- **Max spread:** $0.50 + relative/cost gates
- **Signal TTL:** 24 hours
- **Stop model:** Weekly/HTF structure + volatility buffer ~2.0 ATR(H4)
- **Confluence threshold:** 75/100, separation 15
- **Mandatory pillars:** w1_d1_h4_trend_structure, macro_real_yield_dxy

## SOW Section 12C.5 — Strategy Frequency Is Not a Quota
NO-TRADE > frequency target when evidence or execution quality is insufficient.

## SOW Section 12C.2 — Hard Gates Override Score
A 95/100 score must still produce NO-TRADE if any hard gate fails.
