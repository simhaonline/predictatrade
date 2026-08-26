---
name: ea-testing-validation
description: "EA robustness, Monte Carlo, and edge-case testing."
---

# ea-testing-validation

Use when testing Predict-A-Trade EAs for robustness, reliability, and edge-case handling before production.

## Unit Tests (MT5 Strategy Tester)
- Order placement: correct SL/TP sign for BUY/SELL
- Magic number: correct strategy-specific magic assigned
- Comment format: PAT-STRATEGY:signal_id present
- SL validation: wrong-side SL rejected
- Partial close: TP1 closes ~1/3, SL moves to breakeven
- Emergency stop: closes all on KILL_SWITCH
- Disconnect: stops trading when server unreachable

## Robustness Tests
- Monte Carlo: randomize tick sequence, spread, slippage (200+ runs)
- Parameter stability: small changes = small P&L delta
- Symbol cross-test: XAUUSD, XAGUSD, EURUSD (symbol leaks)
- Time shift: random start points within period

## Edge Cases
- Zero-volume tick (skip)
- Gap open (no market entry at gap price)
- Spread spike > 50 pips (NO-TRADE)
- Margin call (stop opening, close with error)
- News event during position (respect news gate)
- Partial fill (handle fill < requested lot)
- Duplicate signal ID (detect and reject)

## Performance
- OnTick < 2ms (must not block next tick)
- No memory leaks over 72h continuous
- CPU < 5% on typical 1-core VPS
- Non-blocking IPC file writes
