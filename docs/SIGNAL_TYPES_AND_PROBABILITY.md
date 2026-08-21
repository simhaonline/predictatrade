# Predict-A-Trade — Signal Types and Probability Reference

## Signal Direction Types

The real-time engine produces six direction types. Each represents a distinct stage in the signal lifecycle:

| Direction | Color (UI) | Meaning | Executable? | Source |
|-----------|-----------|---------|-------------|--------|
| **BUY** | Green | Qualified long signal — raw score ≥ trade threshold AND all 12 hard gates passed | ✅ Yes (if license/entitlement/device permits) | Strategy engine + gate evaluation |
| **SELL** | Red | Qualified short signal — raw score ≥ trade threshold AND all 12 hard gates passed | ✅ Yes (if license/entitlement/device permits) | Strategy engine + gate evaluation |
| **BUY_CANDIDATE** | Amber/Yellow | Advisory long — raw score ≥ candidate threshold but < trade threshold. Directional conviction exists but is not strong enough for automatic execution. | ❌ No (advisory only) | Candidate threshold system |
| **SELL_CANDIDATE** | Orange | Advisory short — raw score ≥ candidate threshold but < trade threshold. | ❌ No (advisory only) | Candidate threshold system |
| **NO-TRADE** | Gray | No meaningful opportunity — raw score < candidate threshold, OR strategy returned NO-TRADE due to conflicting timeframes, regime mismatch, etc. | ❌ No | Strategy engine |
| **BLOCKED** | Gray (preserves original direction) | Signal had a direction (BUY/SELL) but was blocked by a hard gate veto, cooldown, duplicate detection, or other safety mechanism. The original market direction is preserved for transparency. | ❌ No | Gate engine / safety system |

### BUY vs BUY_CANDIDATE — Detailed Explanation

The distinction between **BUY** and **BUY_CANDIDATE** is fundamental to the system's safety design:

#### BUY (Qualified / Executable)
1. **Strategy scoring**: The strategy engine computed a raw score that meets or exceeds the **trade threshold** (strategy-specific, regime-aware).
2. **Direction confirmed**: Long score > short score with sufficient dominance (no flip-flopping).
3. **Multi-timeframe alignment**: Required timeframes agree on direction.
4. **All 12 hard gates passed**: data quality, session, news, spread, slippage, total cost, exposure, margin, R:R net expectancy, entitlement, license, execution permission.
5. **Signal class**: `EXECUTABLE`
6. **Grade**: Assigned based on calibration sufficiency (A+, A, B, C, or UNRATED)

#### BUY_CANDIDATE (Advisory / Not Executable)
1. **Strategy scoring**: The raw score is between the **candidate threshold** and the **trade threshold**.
2. **Direction inferred**: Direction is determined from long/short score dominance (if dominance is sufficient).
3. **No gate evaluation**: Hard gates are NOT run on candidates (they would fail anyway due to insufficient score).
4. **Signal class**: `ADVISORY`
5. **Grade**: `RESEARCH`
6. **Purpose**: Informational — shows the user that a directional setup is forming but has not reached the qualification threshold. This is NOT a trade signal.

### Candidate Thresholds per Strategy

| Strategy | Candidate Threshold | Trade Threshold | Max Theoretical Score |
|----------|-------------------|-----------------|---------------------|
| STANDARD_SCALPING | 40 | 65 | ~80 |
| ULTRA_SCALPING | 40 | 65 | ~78 |
| STANDARD_SWING | 35 | 55 | ~92 |
| TREND_SWING | 30 | 50 | ~75 |

**Note**: Thresholds are regime-aware. In RANGE regime, thresholds may be lower because the evidence budget is smaller. See `strategy/candidate_threshold.go` and `strategy/regime_thresholds.go`.

### Signal Status Lifecycle

Signals progress through lifecycle statuses:

```
DETECTED → VALIDATING → CANDIDATE → CALIBRATING → RISK_CHECK → CONFIRMED → ACTIVE → TRIGGERED → FILLED → TP1/TP2/TP3 → CLOSED
                                    ↓
                              INVALIDATED / CANCELLED / EXPIRED / STOPPED
```

**DETECTED** is the initial status for all signals (including NO-TRADE and candidates).
**CONFIRMED** is set only when all gates pass for a BUY/SELL signal.
**BLOCKED** grade indicates a gate veto or safety block (direction is preserved).

## Calibrated Probability (PROB)

### Why PROB May Show "Pending"

The **PROB** column displays the calibrated probability of the signal achieving its prediction target (typically TP1 hit). Per SOW §16 and §36:

> **Until a calibration model is VALIDATED or PROMOTED, probability MUST be NULL (zero).**

The system uses a sigmoid calibration model:
```
calibrated_probability = sigmoid(a × (raw_score/100) + b)
```

Default calibration models are seeded with `Status: "UNVERIFIED"`. This is intentional and correct — unverified models cannot produce trustworthy probabilities. The probability will show **"Pending"** in the UI until:

1. A calibration model is trained on sufficient historical data
2. The model passes validation (Brier score, ECE, sample sufficiency)
3. The model is promoted to production status

### What PROB Is NOT

- **Raw score is NOT probability**: The raw score (0-100+) is a weighted evidence confluence sum, not a probability. It is shown in the separate **Score** column.
- **Probability cannot be fabricated**: The system must NOT produce fake probabilities. Zero/null is the honest state when calibration is unverified.
- **Demo/replay data must be labeled**: Probability from non-live data sources carries an `UNVERIFIED` provenance state.

### Calibration Model Status

| Status | Meaning | PROB Display |
|--------|---------|--------------|
| `UNVERIFIED` | Default seeded models — NOT validated against historical outcomes | "Pending" |
| `SHADOW` | Model running in shadow mode — predictions tracked but not used | "Pending" |
| `VALIDATED` | Model passed validation checks (Brier, ECE, sample size, Wilson lower bound) | Shows percentage |
| `PROMOTED` | Model validated and promoted to production use | Shows percentage |

### Calibration Parameters (Default Models)

| Strategy | Sigmoid A | Sigmoid B | Prediction Target |
|----------|-----------|----------|-------------------|
| STANDARD_SCALPING | 2.5 | -0.5 | TP1_HIT |
| ULTRA_SCALPING | 3.0 | -0.8 | TP1_HIT |
| STANDARD_SWING | 2.0 | -0.3 | TP1_HIT |
| TREND_SWING | 1.8 | -0.2 | TP1_HIT |

## TP/SL Geometry (v1.4.0 Fix)

### How TP/SL Are Computed

Entry, Stop Loss, and Take Profit levels are computed using **ATR-based multipliers** — the same volatility measure for both SL and TP, ensuring balanced geometry:

| Level | Formula (BUY) | Formula (SELL) |
|-------|---------------|----------------|
| **Entry** | Ask price | Bid price |
| **SL** | Entry − (ATRMultiplierSL × ATR) | Entry + (ATRMultiplierSL × ATR) |
| **TP1** | Entry + (ATRMultiplierTP1 × ATR) | Entry − (ATRMultiplierTP1 × ATR) |
| **TP2** | Entry + (ATRMultiplierTP2 × ATR) | Entry − (ATRMultiplierTP2 × ATR) |
| **TP3** | Entry + (ATRMultiplierTP3 × ATR) | Entry − (ATRMultiplierTP3 × ATR) |

### Strategy ATR Multipliers

| Strategy | SL × ATR | TP1 × ATR | TP2 × ATR | TP3 × ATR | MinRR (gate) |
|----------|---------|-----------|-----------|-----------|-------------|
| STANDARD_SCALPING | 1.0 | 1.0 | 1.5 | 2.0 | 1.2 |
| ULTRA_SCALPING | 0.5 | 0.5 | 0.75 | 1.0 | 1.0 |
| STANDARD_SWING | 1.5 | 1.5 | 2.5 | 3.5 | 1.8 |
| TREND_SWING | 2.0 | 2.0 | 4.0 | 6.0 | 2.5 |

### What Changed in v1.4.0

**Before (broken):** TP1 was computed as `MinRR × SL_distance`, making TP1 2.5x further than SL. This caused trades to hit the tight SL before reaching the distant TP1 — "shows profit then closes with loss."

**After (fixed):** TP1 is computed as `ATRMultiplierTP1 × ATR` — the same ATR basis as SL. This gives R:R ≈ 1:1 for TP1, with TP2/TP3 providing larger upside. The MinRR gate validates the resulting R:R and rejects signals where R:R is insufficient, rather than inflating TP to force an artificial R:R.

### Minimum SL Distance

SL distance is enforced to be at least `ATRMultiplierSL × ATR` from entry. When structural levels (swing lows/highs) are close to entry, the ATR-based minimum prevents SL from being too tight.

## Frontend Display

### Admin Signals Panel (`/admin/signals`)
- Full signal table with all columns: Time, Direction, Strategy, Symbol, Prob, Score, Entry, SL, TP1, TP2, TP3, Regime, Session, Status
- Direction filters: ALL, BUY, BUY_CANDIDATE, SELL, SELL_CANDIDATE, NO-TRADE
- Strategy tabs: ALL, STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING
- Expandable rows show: evidence breakdown, gate results, NO-TRADE reasons, full entry/SL/TP grid
- PROB column shows "Pending" when calibration is unverified

### User Signals Page (`/dashboard/signals`)
- Simplified signal table: Direction, Strategy, Probability, Entry, SL, TP, Status, Date
- Direction color-coded: BUY (green), SELL (red), BUY_CANDIDATE (amber), SELL_CANDIDATE (orange), NO-TRADE (gray)
- PROB column shows "Pending" with tooltip explaining calibration requirement

### WebSocket Live Updates
Both pages subscribe to WebSocket signal events. The Go engine broadcasts:
- `SIGNAL` events with priority P0 (directional) or P1 (NO-TRADE)
- Event envelope includes: event ID, stream ID, schema version, timestamp, sequence, priority
- Entitlement filtering: clients only receive signals for strategies they are entitled to

## MQL EA Strategy Selection (v1.05)

Both MT4 and MT5 EAs (v1.05) include input parameters for strategy and direction filtering:

### Strategy Selection Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `ReceiveStandardScalping` | `true` | STANDARD_SCALPING (M1/M5 scalping) |
| `ReceiveUltraScalping` | `true` | ULTRA_SCALPING (M1 ultra-fast scalping) |
| `ReceiveStandardSwing` | `true` | STANDARD_SWING (M15/H1 swing trading) |
| `ReceiveTrendSwing` | `true` | TREND_SWING (H1/H4 trend following) |

### Direction Filter Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `ReceiveBuy` | `true` | Receive BUY signals (qualified, executable) |
| `ReceiveSell` | `true` | Receive SELL signals (qualified, executable) |
| `ReceiveBuyCandidate` | `true` | Receive BUY_CANDIDATE (advisory only) |
| `ReceiveSellCandidate` | `true` | Receive SELL_CANDIDATE (advisory only) |

All 4 strategies and all 4 directions are enabled by default. Subscribers can disable specific strategies or directions via EA inputs — no server-side change needed.

### Signal Delivery Chain

```
Go Engine → WebSocket → Windows Agent → Named Pipe (PAT_signals.txt) → MT4/MT5 EA
```

1. Go engine generates signal → `broadcastSignalToAll()` sends to both WebSocketHub (frontend) and AgentHub (Windows Agent)
2. Windows Agent receives `SIGNAL` event → `processSignals()` → `SendSignalToEA()` writes to `PAT_signals.txt`
3. MT4/MT5 EA reads `PAT_signals.txt` → `HandleSignal()` → filters by strategy/direction → displays or executes

## Related Files

| Component | File |
|-----------|------|
| Signal types | `realtime/internal/types/types.go` |
| Signal engine (gate evaluation) | `realtime/internal/signal/engine.go` |
| Candidate threshold logic | `realtime/internal/strategy/candidate_threshold.go` |
| Regime-specific thresholds | `realtime/internal/strategy/regime_thresholds.go` |
| Calibration consumer | `realtime/internal/calibration/consumer.go` |
| Main engine pipeline | `realtime/cmd/realtime-engine/main.go` |
| Admin signals page | `frontend/src/app/(admin)/admin/signals/page.tsx` |
| User signals page | `frontend/src/app/(user)/dashboard/signals/page.tsx` |
| WebSocket broadcast | `realtime/internal/gateway/websocket.go` |
| Signal persistence | `realtime/internal/marketdata/persistence.go` |
| TP/SL geometry builder | `realtime/internal/strategy/geometry.go` |
| Strategy configs (ATR multipliers) | `realtime/internal/strategy/strategies.go` |
| Agent hub (signal delivery) | `realtime/internal/gateway/agent_ws.go` |
| MT4 EA (v1.05) | `mql/mt4/PredictATrade_MT4.mq4` |
| MT5 EA (v1.05) | `mql/mt5/PredictATrade_MT5.mq5` |
