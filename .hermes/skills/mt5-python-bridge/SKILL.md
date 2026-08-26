---
name: mt5-python-bridge
description: "Python backtesting, data export, and MT5 Python API."
---

# mt5-python-bridge

Use when connecting Python research to MetaTrader 5 for data export, strategy validation, or Python-MQL parity testing.

## Setup
```bash
pip install MetaTrader5  # Windows only, MT5 terminal must be running
```

## Core Operations
### Data Export (MT5 to Python)
```python
mt5.initialize()
rates = mt5.copy_rates_from("XAUUSD", mt5.TIMEFRAME_M5, datetime(2026,1,1), 50000)
```

### Symbol Spec
```python
info = mt5.symbol_info("XAUUSD")
# digits, point, tick_size, tick_value, volume_min/max/step
```

### Account State
```python
acc = mt5.account_info()
# balance, equity, margin, margin_free, margin_level, leverage
```

## Predict-A-Trade Pipeline
MT5 export → research/data/raw/xauusd_m5.csv → backtester.py → Go parity check → signal validation

## Parity Testing
Go feature vector == Python feature vector (+/- epsilon)
Go signal output == Python signal output (same input)

## Safety
- Never call mt5.order_send() from research on live account
- Data export is safe on live terminal
- Use demo account for all order operations
