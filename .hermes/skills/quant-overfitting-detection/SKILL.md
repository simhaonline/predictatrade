---
name: quant-overfitting-detection
description: "Detect overfitting, lookahead, and bias in strategies."
---

# quant-overfitting-detection

Use for PAT strategy validation against overfitting.

Detection: lookahead (bar[0] vs bar[1]), survivorship bias, parameter instability, multiple testing, deflated Sharpe, sample insufficiency

Checks: Monte Carlo 200+ runs, walk-forward OOS, cross-symbol, time shift

Red flags: profit factor > 3, win rate > 70%, Sharpe > 3, zero losing months

Scripts: scripts/quant_validation.py, scripts/strategy_change_gate.py
