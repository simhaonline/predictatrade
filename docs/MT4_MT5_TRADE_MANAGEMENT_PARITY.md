# MT4/MT5 Trade Management Parity

| Capability | MT4 | MT5 | Status |
|-----------|----|----|--------|
| Initial SL | OrderSend SL param | CTrade SL param | EXISTS_AND_WIRED |
| TP1/TP2/TP3 | OrderSend TP + EA manages partial | CTrade TP + EA manages partial | EXISTS_AND_WIRED |
| Break-even | OrderModify(ticket, openPx+spread, ...) | trade.PositionModify(ticket, openPx+spread, ...) | EXISTS_AND_WIRED (fixed: cost-aware) |
| Profit lock | Partial close at TP1 (50%) → SL to breakeven | Partial close at TP1 (50%) → SL to breakeven | EXISTS_AND_WIRED |
| Dynamic trailing | ATR-based, newSL = Bid - ATR*mult | ATR-based, newSL = Bid - ATR*mult | EXISTS_AND_WIRED |
| Monotonic SL | newSL > sl (BUY), newSL < sl (SELL) | newSL > sl (BUY), newSL < sl (SELL) | EXISTS_AND_WIRED |
| Broker ACK | GetLastError() on OrderModify failure | trade.ResultRetcode() on failure | EXISTS_AND_WIRED |
| Retry handling | Not implemented (single attempt) | Not implemented (single attempt) | PARTIALLY_IMPLEMENTED |
| Reconnect recovery | EA reads SL from broker on restart | EA reads SL from broker on restart | EXISTS_AND_WIRED |
| Stop-level check | MarketInfo(MODE_STOPLEVEL) | SymbolInfoInteger(SYMBOL_TRADE_STOPS_LEVEL) | FIXED (added in this audit) |
| Freeze-level check | MarketInfo(MODE_FREEZELEVEL) | SymbolInfoInteger(SYMBOL_TRADE_FREEZE_LEVEL) | FIXED (added in this audit) |
| Partial close | OrderClose (partial volume) | trade.PositionClosePartial | EXISTS_AND_WIRED |
| SL normalization | NormalizeDouble(SL, MarketInfo(MODE_DIGITS)) | NormalizeDouble(SL, SymbolInfoInteger(SYMBOL_DIGITS)) | EXISTS_AND_WIRED |
| Swap avoidance | IsNearSwapTime() → ClosePosition | IsNearSwapTime() → ClosePosition | EXISTS_AND_WIRED |
| Max hold time | MaxHoldHours → ClosePosition | MaxHoldHours → ClosePosition | EXISTS_AND_WIRED |
| TP2 full close | CloseAtTP2 → ClosePosition | CloseAtTP2 → ClosePosition | EXISTS_AND_WIRED |
| Capital protection | UpdateCapitalProtection() daily loss check | UpdateCapitalProtection() daily loss check | EXISTS_AND_WIRED |
| Slippage monitoring | SLIPPAGE_EVENT sent to server | SLIPPAGE_EVENT sent to server | EXISTS_AND_WIRED |
| Spread filter | PAT_CheckSpread() before entry | PAT_CheckSpread() before entry | EXISTS_AND_WIRED |
| Position sizing | PAT_CalcLotSize() with tick value/size | PAT_CalcLotSize() with tick value/size | EXISTS_AND_WIRED |
