/* PAT Terminal — XAUUSD Data Engine (Matching https://live.predictatrade.com/) */

window.PAT = window.PAT || {};

PAT.state = {
  price: 2515.00,
  bid: 2514.85,
  ask: 2515.15,
  spread: 0.30,
  volume: 40,
  regime: "RANGE",
  session: "SYDNEY",
  mtfScore: +20,
  confidence: 60,
  apiConnected: true,
  masterNode: true
};

PAT.newsEvents = [
  "HIGH CB Consumer Confidence (Aug) @14:00Z",
  "MEDIUM Chicago Fed National Activity Index (Jul) @12:30Z",
  "MEDIUM S&P/Case-Shiller Home Price YoY (Jun) @13:00Z",
  "MEDIUM New Home Sales (Jul) @14:00Z",
  "MEDIUM API Crude Oil Stock Change @20:30Z"
];

PAT.mtfStates = {
  M1: "SELL",
  M5: "BUY",
  M15: "BUY",
  M30: "NEUT",
  H1: "SELL",
  H4: "BUY",
  D1: "NEUT",
  W1: "NEUT"
};

PAT.allIndicators = [
  { name: "EMA9", val: "2,514.26" },
  { name: "EMA21", val: "2,512.28" },
  { name: "EMA50", val: "2,508.49" },
  { name: "EMA100", val: "2,501.94" },
  { name: "EMA200", val: "2,495.67" },
  { name: "SMA50", val: "2,508.46" },
  { name: "SMA100", val: "2,502.05" },
  { name: "SMA200", val: "2,495.06" },
  { name: "MACD", val: "2.17" },
  { name: "MACD Sig", val: "2.29" },
  { name: "ADX", val: "48.62" },
  { name: "+DI", val: "19.18" },
  { name: "-DI", val: "8.80" },
  { name: "RSI", val: "64.30" },
  { name: "Stoch %K", val: "74.25" },
  { name: "Stoch %D", val: "75.87" },
  { name: "StochRSI", val: "0.64" },
  { name: "StochRSI K", val: "0.58" },
  { name: "StochRSI D", val: "0.46" },
  { name: "CCI", val: "51.28" },
  { name: "ATR", val: "1.84" },
  { name: "BB Upper", val: "2,522.22" },
  { name: "BB Mid", val: "2,512.18" },
  { name: "BB Lower", val: "2,502.14" },
  { name: "BB Width", val: "0.0080" },
  { name: "OBV", val: "-59,666,142" },
  { name: "OBV Z", val: "+1.25" },
  { name: "SAR", val: "2,502.18" },
  { name: "SAR Long", val: "YES" },
  { name: "Ich Tenkan", val: "2,511.98" },
  { name: "Ich Kijun", val: "2,506.15" },
  { name: "Ich SenA", val: "2,501.72" },
  { name: "Ich SenB", val: "2,514.32" },
  { name: "VWAP", val: "2,508.67" },
  { name: "Momentum", val: "100.07" },
  { name: "OsMA", val: "-0.12" },
  { name: "TickVol Z", val: "+1.82" },
  { name: "BB Width Z", val: "+0.94" },
  { name: "MACD Hist", val: "-0.12" },
  { name: "EMA Cross", val: "YES" },
  { name: "MACD Bull", val: "YES" },
  { name: "MACD Bear", val: "YES" },
  { name: "BB Bull Rev", val: "YES" },
  { name: "BB Bear Rev", val: "YES" }
];

PAT.hardGates = [
  { id: "data quality", status: "PASS" },
  { id: "session", status: "PASS" },
  { id: "news", status: "PASS" },
  { id: "spread", status: "PASS" },
  { id: "slippage", status: "PASS" },
  { id: "total cost", status: "PASS" },
  { id: "min atr", status: "PASS" },
  { id: "stop hunt filter", status: "PASS" },
  { id: "exposure", status: "PASS" },
  { id: "margin", status: "PASS" },
  { id: "rr net expectancy", status: "PASS" },
  { id: "entitlement", status: "PASS" },
  { id: "license", status: "PASS" },
  { id: "execution permission", status: "PASS" }
];

PAT.strategyEngines = [
  { id: "STD SCALP", name: "STD SCALP", status: "LIVE", decision: "NO-TRADE", score: "35.7", tf: "M5/M15", stats: "E:18 C:8 S:0" },
  { id: "ULTRA", name: "ULTRA", status: "LIVE", decision: "WAIT", score: "35.2", tf: "M1/M5", stats: "E:15 C:0 S:0" },
  { id: "STD SWING", name: "STD SWING", status: "LIVE", decision: "NO-TRADE", score: "81.2", tf: "M15/H1", stats: "E:1 C:0 S:0" },
  { id: "TREND SWING", name: "TREND SWING", status: "WAITING", decision: "—", score: "0.0", tf: "H1/H4", stats: "E:0 C:0 S:0" },
  { id: "MARNIE FIB", name: "MARNIE FIB", status: "LIVE", decision: "NO-TRADE", score: "0.0", tf: "M15/H1", stats: "E:1 C:0 S:0" }
];

PAT.crossMarketSummary = [
  { driver: "eurusd", status: "NEUTRAL" },
  { driver: "vix", status: "NEUTRAL" },
  { driver: "btc", status: "NEUTRAL" },
  { driver: "oil", status: "NEUTRAL" },
  { driver: "dxy", status: "BEARISH" },
  { driver: "cot", status: "BULLISH" },
  { driver: "real_yields", status: "NEUTRAL" }
];

PAT.signals = [
  { id: "MARNIE FIB", state: "NO-TRADE", score: "0.0", time: "18:15:00", entry: "—", sl: "—", tp: "—", target: "NO_TRADE" },
  { id: "STANDARD SWING", state: "NO-TRADE", score: "81.2", time: "18:15:00", entry: "2,514.80", sl: "2,500.68", tp: "2,528.64", target: "NO_TRADE" },
  { id: "STANDARD SCALPING", state: "BUY_CANDIDATE", score: "35.7", time: "18:15:00", entry: "2,514.80", sl: "2,512.88", tp: "2,517.69", target: "ADVISORY" },
  { id: "ULTRA SCALPING", state: "BUY_CANDIDATE", score: "35.2", time: "18:15:00", entry: "2,514.80", sl: "2,512.88", tp: "2,517.69", target: "ADVISORY" },
  { id: "STANDARD SCALPING", state: "BUY_CANDIDATE", score: "31.4", time: "18:15:00", entry: "2,514.80", sl: "2,512.88", tp: "2,517.69", target: "ADVISORY" }
];

PAT.agentsStatus = [
  { k: "Live Data", v: "CONNECTED" },
  { k: "Agents Connected", v: "2 agent(s)" },
  { k: "MT5 Terminals", v: "1 connected" },
  { k: "Snapshots Received", v: "1,995 total" },
  { k: "Data Quality", v: "AUTHORITATIVE" },
  { k: "WebSocket", v: "LIVE" },
  { k: "Positions", v: "0 open (0B/0S)" }
];
