---
name: broker-symbol-mapping
description: "Map broker-specific XAUUSD specs and lot sizing."
---

# broker-symbol-mapping

Use when configuring broker-specific XAUUSD symbol parameters in Predict-A-Trade.

## XAUUSD is NOT Standard Across Brokers
- Symbol name: XAUUSD, XAUUSDm, XAUUSD., GOLD
- Digits: 2 (most) or 1 (ECN)
- Tick size: 0.01 or 0.1
- Contract: 100oz (standard) or 10oz (mini) or 1oz (micro)
- Lot step: 0.01, 0.1, or 1.0
- Stop level: 0 to 200+ points
- Freeze level: 0 to 50+ points

## Discovery (Python)
```python
import MetaTrader5 as mt5
s = mt5.symbol_info("XAUUSD")
# digits, point, tick_size, tick_value, contract, vol_min/max/step, stops, freeze
```

## Broker Profiles (store in DB)
- IC Markets: XAUUSD, digits=2, tick=0.01, stops=0
- Pepperstone: XAUUSD, digits=2, tick=0.01
- XM: GOLD, digits=2
- Exness: XAUUSDm / XAUUSD.

## Validation
- [ ] Symbol name confirmed (chart vs order name may differ)
- [ ] Digits match tick_size
- [ ] Volume min/max/step tested
- [ ] Stop/freeze levels validated
- [ ] Margin verified with sample lot
- [ ] Swap rates + triple-swap day
- [ ] Session hours in UTC
