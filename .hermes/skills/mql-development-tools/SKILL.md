---
name: mql-development-tools
description: "MQL4/MQL5 coding, linting, compiling, and debugging."
---

# mql-development-tools

Use when developing MQL4/MQL5 Expert Advisors, indicators, or scripts.

## VS Code Integration
Extension: MQL Tools (L-I-V) — syntax highlighting, compile from VS Code.
Install: code --install-extension L-I-V.mql-tools

## MetaEditor (Built-in)
F7=compile, F5=strategy tester, F5+debugger with breakpoints.

## Project EAs
- mql/mt5/PredictATrade_MT5.mq5 (2089 lines)
- mql/mt4/PredictATrade_MT4.mq4 (2068 lines)

## Key Patterns
Magic numbers: SS=40101, US=40201, SW=40301, TS=40401, MF=40501
Order comment: PAT-<STRATEGY>:<signal_id>
SL validation: bool slWrong = isBuy ? (sl >= entry) : (sl <= entry);
Partial TP: ONE position, close ~1/3 at TP1, move SL to breakeven.

## Known MQL Bugs (mql-fix.md)
- Wrong-side S/L placement
- TP rarely honored (~9.5%)
- TP1/2/3 opens 3 positions instead of scaling one
- No strategy_id in comment/magic
