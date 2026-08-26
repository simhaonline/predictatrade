---
name: mt5-mcp-integration
description: "Connect AI agents to MetaTrader 5 via MCP protocol."
---

# mt5-mcp-integration

Use when connecting AI agents to MetaTrader 5 for live trading, market data,
position management, or strategy development.

## Two Approaches

### 1. Native MT5 MCP (Build 5955+)
MetaTrader 5 has built-in MCP support. The AI Assistant in MetaEditor can:
- Generate new MQL5 programs from prompts
- Analyze existing code and suggest fixes
- Access trading account, market data, and strategy tester
Setup: Log into MQL5 Community account in MT5 terminal. No additional config.

### 2. metatrader-mcp-server (ariadng, 690+ stars, MIT)
PyPI: pip install metatrader-mcp-server
GitHub: https://github.com/ariadng/metatrader-mcp-server

14 MCP Tools:
- Account: get_account_info
- Market: get_symbol_info, get_candles, get_current_price, get_all_symbols
- Positions: get_all_positions, close_position, close_all_profitable_positions
- Orders: place_market_order, place_pending_order, get_all_pending_orders
- History: get_order_history, get_deal_history
- WebSocket Quote Server for real-time tick streaming

Setup:
  pip install metatrader-mcp-server
  Copy MCP_EA.ex4/ex5 to MT5 Experts folder
  Set MT_HOST=127.0.0.1 MT_PORT=18080 API_KEY=<random>
  uvicorn mcp_server.app:app --host 0.0.0.0 --port 8000

Safety: Executes real trades. Use demo accounts for testing only.
