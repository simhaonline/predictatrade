//+------------------------------------------------------------------+
//|                                PredictATrade_MasterNode_MT4.mq4  |
//|                            Predict-A-Trade v1.0.0                |
//|     Master Node: Live data collection for system & dashboard     |
//|     NO License Key · NO Trading · Data Collection Only           |
//|  IPC: FILE_COMMON folder (shared between all MT terminals)       |
//+------------------------------------------------------------------+
//| This EA collects comprehensive live market data — ticks,         |
//| multi-timeframe OHLC bars, technical indicators, account info,   |
//| symbol/broker specifications, and session detection — and writes |
//| it to a FILE_COMMON folder for the Windows Agent to forward to   |
//| the Go real-time engine and the dashboard/Command Center.        |
//|                                                                  |
//| This EA does NOT:                                                |
//|   - Require or check a license key                               |
//|   - Read or execute trading signals                              |
//|   - Place, modify, or close any orders                           |
//|   - Perform any financial operation                              |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property version   "1.00"
#property strict
#property description "Master Node: Live data collection for system & dashboard"
#property description "NO License Key · NO Trading · Data Collection Only"

//=== Input Parameters ===
input int     SnapshotIntervalMs = 10;     // HFT: 10ms snapshot (1-5ms co-located)
input int     TickIntervalMs     = 0;       // 0 = every tick (HFT: 1-5ms when co-located)
input string  BrokerSymbol      = "";     // Empty = auto-detect chart symbol
input bool    SendTickData      = true;   // Send tick data to Agent
input bool    SendSnapshots     = true;   // Send comprehensive market snapshots
input bool    SendIndicators    = true;   // Include indicator values in snapshots
input bool    SendMultiTF       = true;   // Include multi-timeframe bar data
input bool    SendAccountInfo   = true;   // Include account info in snapshots
input bool    SendSymbolInfo    = true;    // Include symbol/broker spec in snapshots
input bool    DebugMode         = false;  // Print debug messages to Experts log

// ─── Agent Status Notifications ───
input bool    EnableNotifications  = true;   // Send notifications when agent connects/disconnects
input string  TelegramBotToken     = "";     // Telegram bot token
input string  TelegramChatID       = "";     // Telegram chat ID
input string  EmailNotifyAddress   = "";     // Email for notifications (uses MT4 SendMail)
input int     NotifyCooldownSec    = 300;    // Min seconds between repeated notifications

//=== IPC Files (in FILE_COMMON folder — shared with Windows Agent) ===
#define PAT_MASTER_FILE  "PAT_master_data.txt"
#define PAT_HEARTBEAT    "PAT_heartbeat.txt"
// PAT_RESYNC is written by the Windows Agent (on engine REQUEST_SNAPSHOT nudge)
// and polled by this EA to force an immediate MARKET_SNAPSHOT re-emit.
#define PAT_RESYNC       "PAT_resync.txt"

//=== Timeframes for multi-TF bar data ===
// Per-TF broker CopyRates sync: the engine ingests these bars directly so its
// candles match MT5 exactly (no tick-re-aggregation drift).
#define TF_COUNT 9
int g_timeframes[TF_COUNT] = {PERIOD_M1, PERIOD_M5, PERIOD_M15, PERIOD_M30, PERIOD_H1, PERIOD_H4, PERIOD_D1, PERIOD_W1, PERIOD_MN1};
string g_tfNames[TF_COUNT] = {"M1", "M5", "M15", "M30", "H1", "H4", "D1", "W1", "MN1"};

//=== Global State ===
string  g_symbol;
string  g_connection   = "OFFLINE";
double  g_lastKnownBid = 0;   // last valid price — weekend market_closed snapshots
double  g_lastKnownAsk = 0;
string  g_accountID     = "—";
string  g_broker        = "";
uint    g_lastTickSend   = 0;
uint    g_lastSnapshot   = 0;
int     g_tickCount     = 0;
int     g_snapshotCount  = 0;
datetime g_lastBarTime  = 0;
uint    g_lastNotifyTime  = 0;
bool    g_lastAgentState  = false;

//+------------------------------------------------------------------+
//| FormatISO8601UTC — Convert datetime to ISO8601 UTC string        |
//| Returns: "2026-08-21T16:25:11Z" (proper RFC3339/ISO8601 format)  |
//| This replaces TimeToStr which produces "2026.08.21 19:25:11"      |
//| (dot separators, no timezone, broker time) — unparseable by JS   |
//+------------------------------------------------------------------+
string FormatISO8601UTC(datetime t)
{
    int year = TimeYear(t);
    int mon = TimeMonth(t);
    int day = TimeDay(t);
    int hour = TimeHour(t);
    int min = TimeMinute(t);
    int sec = TimeSeconds(t);
    return StringFormat("%04d-%02d-%02dT%02d:%02d:%02dZ",
        year, mon, day, hour, min, sec);
}



//+------------------------------------------------------------------+
int OnInit()
{
    Print("Predict-A-Trade Master Node v1.00 initializing (MT4)...");
    Print("Mode: DATA COLLECTION ONLY — NO License Key, NO Trading");

    g_symbol = BrokerSymbol;
    if(g_symbol == "") g_symbol = Symbol();
    g_broker = AccountCompany();
    g_accountID = DoubleToStr(AccountNumber(), 0);

    Print("Symbol: ", g_symbol);
    Print("Broker: ", g_broker);
    Print("Account: ", g_accountID);

    if(FileIsExist(PAT_HEARTBEAT, FILE_COMMON))
    {
        g_connection = "CONNECTED";
        Print("Windows Agent detected (heartbeat found in common folder)");
        SendMasterInit();
    }
    else
    {
        g_connection = "OFFLINE";
        Print("WARNING: Windows Agent not detected.");
        Print("Ensure pat-master.exe (Master Node agent) is running on this machine.");
    }

    UpdatePanel();

    // ─── Resilience: periodic timer ───
    // OnTick only fires when the broker streams quotes for the chart symbol. If
    // the terminal/connection hiccups and ticks stall, OnTick stops and the
    // engine goes silently blind. A 1-second OnTimer keeps emitting
    // MARKET_SNAPSHOT regardless of tick flow (terminal must be connected).
    EventSetTimer(1000);

    return(INIT_SUCCEEDED);
}

//+------------------------------------------------------------------+
 void OnDeinit(const int reason)
{
    EventKillTimer();
    MasterWrite("MASTER_DEINIT|{\"reason\":" + IntegerToString((long)reason) +
                ",\"symbol\":\"" + g_symbol + "\",\"account\":\"" + g_accountID + "\"}\n");
    Comment("");
}

//+------------------------------------------------------------------+
void OnTick()
{
    CheckAgentConnection();

    if(g_connection != "CONNECTED") { UpdatePanel(); return; }

    if(SendTickData)
        SendTickToAgent();

    if(SendSnapshots)
        SendMarketSnapshot();

    UpdatePanel();
}

//+------------------------------------------------------------------+
//| OnTimer — resilience fallback for market-data delivery.           |
//+------------------------------------------------------------------+
//| OnTick only fires while the broker streams quotes for the chart.  |
//| If quotes stall (terminal/connection hiccup) OnTick stops and the |
//| engine goes silently blind. This timer runs regardless of ticks   |
//| (as long as the terminal is alive) and re-emits MARKET_SNAPSHOT,  |
//| so the engine always has fresh data. It also honours a REQUEST_   |
//| SNAPSHOT nudge: the agent writes PAT_resync.txt when the engine   |
//| asks for a refresh; we delete it and force an immediate snapshot. |
//+------------------------------------------------------------------+
void OnTimer()
{
    CheckAgentConnection();

    if(g_connection == "CONNECTED" && SendSnapshots)
    {
        // Engine recovery nudge: if the agent dropped a REQUEST_SNAPSHOT flag,
        // force an immediate snapshot (bypass the snapshot throttle).
        if(FileIsExist(PAT_RESYNC, FILE_COMMON))
        {
            FileDelete(PAT_RESYNC, FILE_COMMON);
            g_lastSnapshot = 0;
        }
        SendMarketSnapshot();
    }

    UpdatePanel();
}

//+------------------------------------------------------------------+
// ─── Send notification via Email or Push (MT4 doesn't support WebRequest) ───
void SendAgentNotification(string status, string message)
{
    if(!EnableNotifications) return;
    
    if(GetTickCount() - g_lastNotifyTime < (uint)(NotifyCooldownSec * 1000)) return;
    g_lastNotifyTime = GetTickCount();
    
    string fullMsg = "[Predict-A-Trade Master Node] " + message;
    fullMsg += "\nHost: " + AccountInfoString(ACCOUNT_COMPANY);
    fullMsg += "\nBroker: " + g_broker;
    fullMsg += "\nSymbol: " + g_symbol;
    // Show the broker/local time (TimeCurrent) for the operator, but keep the UTC
    // value as reference. Internal/provenance time truth remains UTC (SOW): the
    // forwarded data fields are all UTC plus broker_offset.
    fullMsg += "\nTime: " + FormatISO8601UTC(TimeCurrent()) + " (broker/local)  [UTC " + FormatISO8601UTC(TimeGMT()) + "]";
    fullMsg += "\nAgent Status: " + status;
    
    Print("[NOTIFY] ", fullMsg);
    
    // 1. Email notification (MT4 built-in SendMail)
    if(EmailNotifyAddress != "")
    {
        string subject = "[Predict-A-Trade] Agent " + status;
        if(SendMail(subject, fullMsg))
            Print("[NOTIFY] Email sent to ", EmailNotifyAddress);
        else
            Print("[NOTIFY] Email failed: ", GetLastError());
    }
    
    // 2. Push notification (MT4 built-in SendNotification)
    if(TelegramChatID != "")
    {
        // Use SendNotification for push to mobile (MT4 doesn't have WebRequest for Telegram API)
        if(SendNotification(fullMsg))
            Print("[NOTIFY] Push notification sent");
        else
            Print("[NOTIFY] Push failed: ", GetLastError());
    }
    
    // 3. Write notification to file for Windows Agent to forward (Telegram/Discord)
    string notifLine = "NOTIFICATION|{\"type\":\"" + status + "\",\"message\":\"" + message + "\",\"timestamp\":\"" + FormatISO8601UTC(TimeGMT()) + "\"}\n";
    MasterAppend(notifLine);
}

void CheckAgentConnection()
{
    static uint lastCheck = 0;
    if(GetTickCount() - lastCheck < 2000) return;
    lastCheck = GetTickCount();

    bool agentOnline = FileIsExist(PAT_HEARTBEAT, FILE_COMMON);
    
    if(agentOnline)
    {
        g_connection = "CONNECTED";
        if(!g_lastAgentState)
        {
            g_lastAgentState = true;
            Print("[AGENT] Windows Agent is now ACTIVE (heartbeat detected)");
            SendAgentNotification("ACTIVE", "Windows Agent is now ACTIVE and connected to the Master Node.");
        }
    }
    else
    {
        if(g_connection == "CONNECTED")
        {
            g_connection = "OFFLINE";
            Print("[AGENT] Windows Agent heartbeat lost");
        }
        if(g_lastAgentState)
        {
            g_lastAgentState = false;
            Print("[AGENT] Windows Agent is now OFFLINE (heartbeat lost)");
            SendAgentNotification("OFFLINE", "WARNING: Windows Agent is OFFLINE! No heartbeat detected. Live data feed may be interrupted.");
        }
    }
}

//+------------------------------------------------------------------+
//| Send lightweight tick data                                        |
//+------------------------------------------------------------------+
void SendTickToAgent()
{
    if(TickIntervalMs > 0)
    {
        uint elapsed = GetTickCount() - g_lastTickSend;
        if(elapsed < (uint)TickIntervalMs) return;
    }
    g_lastTickSend = GetTickCount();

    double bid = MarketInfo(g_symbol, MODE_BID);
    double ask = MarketInfo(g_symbol, MODE_ASK);
    if(bid <= 0 || ask <= 0) return;

    g_lastKnownBid = bid;
    g_lastKnownAsk = ask;

    g_tickCount++;

    string msg = "MASTER_TICK|{\"type\":\"MASTER_TICK\"";
    msg += ",\"symbol\":\"" + g_symbol + "\"";
    msg += ",\"bid\":" + DoubleToStr(bid, 5);
    msg += ",\"ask\":" + DoubleToStr(ask, 5);
    msg += ",\"spread\":" + DoubleToStr(ask - bid, 5);
    msg += ",\"volume\":" + IntegerToString((long)Volume[0]);
    msg += ",\"timestamp\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += ",\"gmt\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += ",\"source\":\"MT4_MASTER\"";
    msg += ",\"broker\":\"" + EscapeJSON(g_broker) + "\"";
    msg += ",\"account\":\"" + g_accountID + "\"";
    msg += ",\"node\":\"MASTER\"";
    // Broker session timezone — collected live so the engine works on Broker TF
    // (not UTC). TimeGMTOffset() returns the broker's GMT offset in seconds.
    msg += ",\"broker_offset\":" + IntegerToString(TimeGMTOffset() / 3600);
    msg += "}\n";

    MasterAppend(msg);
}

//+------------------------------------------------------------------+
//| Send comprehensive market snapshot                                |
//+------------------------------------------------------------------+
void SendMarketSnapshot()
{
    if(SnapshotIntervalMs > 0)
    {
        uint elapsed = GetTickCount() - g_lastSnapshot;
        if(elapsed < (uint)SnapshotIntervalMs) return;
    }
    g_lastSnapshot = GetTickCount();

    double bid = MarketInfo(g_symbol, MODE_BID);
    double ask = MarketInfo(g_symbol, MODE_ASK);
    bool marketClosed = (bid <= 0 || ask <= 0);
    if(marketClosed)
    {
        if(g_lastKnownBid <= 0 || g_lastKnownAsk <= 0) return; // EA just attached — no price known yet
        bid = g_lastKnownBid;
        ask = g_lastKnownAsk;
    }
    else
    {
        g_lastKnownBid = bid;
        g_lastKnownAsk = ask;
    }

    g_snapshotCount++;

    string msg = "MARKET_SNAPSHOT|{";
    msg += "\"type\":\"MARKET_SNAPSHOT\"";
    msg += ",\"symbol\":\"" + g_symbol + "\"";
    msg += ",\"timestamp\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += ",\"gmt\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += ",\"source\":\"MT4_MASTER\"";
    msg += ",\"broker\":\"" + EscapeJSON(g_broker) + "\"";
    msg += ",\"account\":\"" + g_accountID + "\"";
    msg += ",\"node\":\"MASTER\"";
    if(marketClosed) msg += ",\"market_closed\":true";
    // Broker session timezone — collected live so the engine works on Broker TF
    // (not UTC). TimeGMTOffset() returns the broker's GMT offset in seconds.
    msg += ",\"broker_offset\":" + IntegerToString(TimeGMTOffset() / 3600);

    //--- Tick data
    msg += ",\"tick\":{";
    msg += "\"bid\":" + DoubleToStr(bid, 5);
    msg += ",\"ask\":" + DoubleToStr(ask, 5);
    msg += ",\"spread\":" + DoubleToStr(ask - bid, 5);
    msg += ",\"spread_points\":" + IntegerToString((long)MarketInfo(g_symbol, MODE_SPREAD));
    msg += ",\"volume\":" + IntegerToString((long)Volume[0]);
    msg += ",\"time\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += "}";

    //--- Multi-timeframe bar data
    if(SendMultiTF)
    {
        msg += ",\"bars\":{";
        for(int i = 0; i < TF_COUNT; i++)
        {
            if(i > 0) msg += ",";
            msg += "\"" + g_tfNames[i] + "\":" + GetBarJSON(g_timeframes[i]);
        }
        msg += "}";
    }

    //--- Technical indicators
    if(SendIndicators)
    {
        msg += ",\"indicators\":" + GetIndicatorsJSON();
    }

    //--- VWAP
    msg += ",\"vwap\":{";
    msg += "\"session_vwap\":" + DoubleToStr(CalculateSessionVWAP(), 5);
    msg += "}";

    //--- Account info
    if(SendAccountInfo)
    {
        msg += ",\"account_info\":{";
        msg += "\"balance\":" + DoubleToStr(AccountBalance(), 2);
        msg += ",\"equity\":" + DoubleToStr(AccountEquity(), 2);
        msg += ",\"margin\":" + DoubleToStr(AccountMargin(), 2);
        msg += ",\"free_margin\":" + DoubleToStr(AccountFreeMargin(), 2);
        msg += ",\"profit\":" + DoubleToStr(AccountProfit(), 2);
        msg += ",\"currency\":\"" + AccountCurrency() + "\"";
        msg += ",\"leverage\":" + IntegerToString((long)AccountLeverage());
        msg += ",\"server\":\"" + EscapeJSON(AccountServer()) + "\"";
        msg += "}";
    }

    //--- Symbol/broker info
    if(SendSymbolInfo)
    {
        msg += ",\"symbol_info\":{";
        msg += "\"digits\":" + IntegerToString((long)MarketInfo(g_symbol, MODE_DIGITS));
        msg += ",\"point\":" + DoubleToStr(MarketInfo(g_symbol, MODE_POINT), 5);
        msg += ",\"spread\":" + IntegerToString((long)MarketInfo(g_symbol, MODE_SPREAD));
        msg += ",\"stops_level\":" + IntegerToString((long)MarketInfo(g_symbol, MODE_STOPLEVEL));
        msg += ",\"freeze_level\":" + IntegerToString((long)MarketInfo(g_symbol, MODE_FREEZELEVEL));
        msg += ",\"contract_size\":" + DoubleToStr(MarketInfo(g_symbol, MODE_LOTSIZE), 0);
        msg += ",\"min_lot\":" + DoubleToStr(MarketInfo(g_symbol, MODE_MINLOT), 2);
        msg += ",\"max_lot\":" + DoubleToStr(MarketInfo(g_symbol, MODE_MAXLOT), 2);
        msg += ",\"lot_step\":" + DoubleToStr(MarketInfo(g_symbol, MODE_LOTSTEP), 2);
        msg += ",\"swap_long\":" + DoubleToStr(MarketInfo(g_symbol, MODE_SWAPLONG), 2);
        msg += ",\"swap_short\":" + DoubleToStr(MarketInfo(g_symbol, MODE_SWAPSHORT), 2);
        msg += ",\"tick_value\":" + DoubleToStr(MarketInfo(g_symbol, MODE_TICKVALUE), 5);
        msg += ",\"tick_size\":" + DoubleToStr(MarketInfo(g_symbol, MODE_TICKSIZE), 5);
        msg += ",\"margin_init\":" + DoubleToStr(MarketInfo(g_symbol, MODE_MARGININIT), 2);
        msg += ",\"margin_maint\":" + DoubleToStr(MarketInfo(g_symbol, MODE_MARGININIT), 2);
        msg += "}";
    }

    //--- Session info
    msg += ",\"session\":" + GetSessionJSON();

    //--- Open positions summary (read-only, no trading)
    msg += ",\"positions\":" + GetPositionsJSON();

    msg += "}\n";

    MasterAppend(msg);

    if(DebugMode)
        Print("Snapshot #", g_snapshotCount, " sent");
}

//+------------------------------------------------------------------+
//| Get OHLC bar data for a timeframe as JSON                         |
//+------------------------------------------------------------------+
string GetBarJSON(int timeframe)
{
    string s = "{";
    s += "\"open\":" + DoubleToStr(iOpen(g_symbol, timeframe, 0), 5);
    s += ",\"high\":" + DoubleToStr(iHigh(g_symbol, timeframe, 0), 5);
    s += ",\"low\":" + DoubleToStr(iLow(g_symbol, timeframe, 0), 5);
    s += ",\"close\":" + DoubleToStr(iClose(g_symbol, timeframe, 0), 5);
    s += ",\"volume\":" + IntegerToString((long)iVolume(g_symbol, timeframe, 0));
    // Use the actual broker bar open time (iTime) converted to UTC so the
    // engine candle time matches MT5 exactly (not the current wall-clock time).
    s += ",\"time\":\"" + FormatISO8601UTC((datetime)((long)iTime(g_symbol, timeframe, 0) - (long)TimeCurrent() + (long)TimeGMT())) + "\"";

    // Previous closed bar
    s += ",\"prev_open\":" + DoubleToStr(iOpen(g_symbol, timeframe, 1), 5);
    s += ",\"prev_high\":" + DoubleToStr(iHigh(g_symbol, timeframe, 1), 5);
    s += ",\"prev_low\":" + DoubleToStr(iLow(g_symbol, timeframe, 1), 5);
    s += ",\"prev_close\":" + DoubleToStr(iClose(g_symbol, timeframe, 1), 5);
    s += ",\"prev_volume\":" + IntegerToString((long)iVolume(g_symbol, timeframe, 1));
    s += "}";

    return s;
}

//+------------------------------------------------------------------+
//| Get all technical indicators as JSON                              |
//+------------------------------------------------------------------+
string GetIndicatorsJSON()
{
    string s = "{";

    // ATR (14)
    s += "\"atr\":" + DoubleToStr(iATR(g_symbol, PERIOD_CURRENT, 14, 0), 5);

    // RSI (14)
    s += ",\"rsi\":" + DoubleToStr(iRSI(g_symbol, PERIOD_CURRENT, 14, PRICE_CLOSE, 0), 2);

    // EMA 9, 21, 50
    s += ",\"ema9\":" + DoubleToStr(iMA(g_symbol, PERIOD_CURRENT, 9, 0, MODE_EMA, PRICE_CLOSE, 0), 5);
    s += ",\"ema21\":" + DoubleToStr(iMA(g_symbol, PERIOD_CURRENT, 21, 0, MODE_EMA, PRICE_CLOSE, 0), 5);
    s += ",\"ema50\":" + DoubleToStr(iMA(g_symbol, PERIOD_CURRENT, 50, 0, MODE_EMA, PRICE_CLOSE, 0), 5);

    // SMA 200
    s += ",\"sma200\":" + DoubleToStr(iMA(g_symbol, PERIOD_CURRENT, 200, 0, MODE_SMA, PRICE_CLOSE, 0), 5);

    // ADX (14)
    s += ",\"adx\":" + DoubleToStr(iADX(g_symbol, PERIOD_CURRENT, 14, PRICE_CLOSE, MODE_MAIN, 0), 2);
    s += ",\"adx_plus_di\":" + DoubleToStr(iADX(g_symbol, PERIOD_CURRENT, 14, PRICE_CLOSE, MODE_PLUSDI, 0), 2);
    s += ",\"adx_minus_di\":" + DoubleToStr(iADX(g_symbol, PERIOD_CURRENT, 14, PRICE_CLOSE, MODE_MINUSDI, 0), 2);

    // Bollinger Bands (20, 2)
    s += ",\"boll_upper\":" + DoubleToStr(iBands(g_symbol, PERIOD_CURRENT, 20, 2, 0, PRICE_CLOSE, MODE_UPPER, 0), 5);
    s += ",\"boll_lower\":" + DoubleToStr(iBands(g_symbol, PERIOD_CURRENT, 20, 2, 0, PRICE_CLOSE, MODE_LOWER, 0), 5);
    s += ",\"boll_middle\":" + DoubleToStr(iBands(g_symbol, PERIOD_CURRENT, 20, 2, 0, PRICE_CLOSE, MODE_MAIN, 0), 5);

    // MACD (12, 26, 9)
    s += ",\"macd_main\":" + DoubleToStr(iMACD(g_symbol, PERIOD_CURRENT, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0), 5);
    s += ",\"macd_signal\":" + DoubleToStr(iMACD(g_symbol, PERIOD_CURRENT, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0), 5);

    // Stochastic (14, 3, 3)
    s += ",\"stoch_main\":" + DoubleToStr(iStochastic(g_symbol, PERIOD_CURRENT, 14, 3, 3, MODE_SMA, 0, MODE_MAIN, 0), 2);
    s += ",\"stoch_signal\":" + DoubleToStr(iStochastic(g_symbol, PERIOD_CURRENT, 14, 3, 3, MODE_SMA, 0, MODE_SIGNAL, 0), 2);

    // CCI (14)
    s += ",\"cci\":" + DoubleToStr(iCCI(g_symbol, PERIOD_CURRENT, 14, PRICE_TYPICAL, 0), 2);

    // Momentum (14)
    s += ",\"mom\":" + DoubleToStr(iMomentum(g_symbol, PERIOD_CURRENT, 14, PRICE_CLOSE, 0), 5);

    // OsMA (12, 26, 9)
    s += ",\"osma\":" + DoubleToStr(iOsMA(g_symbol, PERIOD_CURRENT, 12, 26, 9, PRICE_CLOSE, 0), 5);

    s += "}";
    return s;
}

//+------------------------------------------------------------------+
//| Calculate session VWAP using today's bars                        |
//+------------------------------------------------------------------+
double CalculateSessionVWAP()
{
    double sumPV = 0;
    double sumV = 0;
    int maxBars = MathMin(iBars(g_symbol, PERIOD_M1), 1440); // Up to 24h of M1 bars

    for(int i = 0; i < maxBars; i++)
    {
        double h = iHigh(g_symbol, PERIOD_M1, i);
        double l = iLow(g_symbol, PERIOD_M1, i);
        double c = iClose(g_symbol, PERIOD_M1, i);
        long v = iVolume(g_symbol, PERIOD_M1, i);
        if(v <= 0) continue;
        double typicalPrice = (h + l + c) / 3.0;
        sumPV += typicalPrice * v;
        sumV  += v;
    }
    if(sumV <= 0) return 0;
    return sumPV / sumV;
}

//+------------------------------------------------------------------+
//| Get session info as JSON (based on GMT time)                      |
//+------------------------------------------------------------------+
string GetSessionJSON()
{
    datetime gmt = TimeGMT();
    int hour = TimeHour(gmt);
    int dow = DayOfWeek();

    bool isWeekend = (dow == 0 || dow == 6);
    string sessionName = "OFF_HOURS";
    bool isOverlap = false;

    // Sydney: 22:00-07:00 GMT
    // Tokyo:  00:00-09:00 GMT
    // London: 08:00-17:00 GMT
    // New York: 13:00-22:00 GMT
    // London/NY overlap: 13:00-17:00 GMT

    if(!isWeekend)
    {
        bool london = (hour >= 8 && hour < 17);
        bool newYork = (hour >= 13 && hour < 22);
        bool tokyo = (hour >= 0 && hour < 9);
        bool sydney = (hour >= 22 || hour < 7);

        if(london && newYork) { sessionName = "LONDON_NEWYORK_OVERLAP"; isOverlap = true; }
        else if(london)  sessionName = "LONDON";
        else if(newYork) sessionName = "NEW_YORK";
        else if(tokyo)   sessionName = "TOKYO";
        else if(sydney)  sessionName = "SYDNEY";
        else             sessionName = "OFF_HOURS";
    }

    string s = "{";
    s += "\"name\":\"" + sessionName + "\"";
    s += ",\"is_overlap\":" + (isOverlap ? "true" : "false");
    s += ",\"is_weekend\":" + (isWeekend ? "true" : "false");
    s += ",\"gmt_hour\":" + IntegerToString((long)hour);
    s += ",\"gmt_dow\":" + IntegerToString((long)dow);
    s += "}";
    return s;
}

//+------------------------------------------------------------------+
//| Get open positions summary as JSON (read-only)                   |
//+------------------------------------------------------------------+
string GetPositionsJSON()
{
    int total = OrdersTotal();
    int patOrders = 0;
    double totalProfit = 0;
    double totalVolume = 0;
    int buyCount = 0;
    int sellCount = 0;

    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(OrderType() == OP_BUY)  buyCount++;
        if(OrderType() == OP_SELL) sellCount++;
        totalProfit += OrderProfit() + OrderSwap() + OrderCommission();
        totalVolume += OrderLots();
        patOrders++;
    }

    string s = "{";
    s += "\"total_orders\":" + IntegerToString((long)patOrders);
    s += ",\"buy_count\":" + IntegerToString((long)buyCount);
    s += ",\"sell_count\":" + IntegerToString((long)sellCount);
    s += ",\"total_lots\":" + DoubleToStr(totalVolume, 2);
    s += ",\"floating_profit\":" + DoubleToStr(totalProfit, 2);
    s += "}";
    return s;
}

//+------------------------------------------------------------------+
//| Send initialization message                                       |
//+------------------------------------------------------------------+
void SendMasterInit()
{
    string msg = "MASTER_INIT|{";
    msg += "\"type\":\"MASTER_INIT\"";
    msg += ",\"ea_version\":\"1.00\"";
    msg += ",\"node\":\"MASTER\"";
    msg += ",\"platform\":\"MT4\"";
    msg += ",\"broker\":\"" + EscapeJSON(g_broker) + "\"";
    msg += ",\"account\":\"" + g_accountID + "\"";
    msg += ",\"symbol\":\"" + g_symbol + "\"";
    msg += ",\"currency\":\"" + AccountCurrency() + "\"";
    msg += ",\"leverage\":" + IntegerToString((long)AccountLeverage());
    msg += ",\"balance\":" + DoubleToStr(AccountBalance(), 2);
    msg += ",\"equity\":" + DoubleToStr(AccountEquity(), 2);
    msg += ",\"digits\":" + IntegerToString((long)MarketInfo(g_symbol, MODE_DIGITS));
    msg += ",\"no_license\":true";
    msg += ",\"no_trading\":true";
    msg += ",\"timestamp\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += "}\n";

    MasterWrite(msg);
    Print("Master Node initialized — data collection mode (no license, no trading)");
}

//+------------------------------------------------------------------+
//| Escape JSON string values                                         |
//+------------------------------------------------------------------+
string EscapeJSON(string s)
{
    string result = "";
    for(int i = 0; i < StringLen(s); i++)
    {
        int c = StringGetCharacter(s, i);
        if(c == 34) result += "\\\"";        // "
        else if(c == 92) result += "\\\\";   // backslash
        else if(c == 10) result += "\\n";    // newline
        else if(c == 13) result += "\\r";    // carriage return
        else if(c == 9) result += "\\t";     // tab
        else result += CharToString((uchar)c);
    }
    return result;
}

//+------------------------------------------------------------------+
//| File I/O using FILE_COMMON (shared between all MT terminals)     |
//+------------------------------------------------------------------+
void MasterWrite(string content)
{
    int retry = 0;
    while(retry < 3)
    {
        int h = FileOpen(PAT_MASTER_FILE, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON | FILE_SHARE_READ | FILE_SHARE_WRITE);
        if(h != -1)
        {
            FileWriteString(h, content);
            FileClose(h);
            return;
        }
        retry++;
        Sleep(5);
    }
    Print("FileOpen WRITE failed after 3 retries: ", PAT_MASTER_FILE, " error=", GetLastError());
}

 void MasterAppend(string content)
{
    // Append (do NOT truncate). The Windows Agent polls PAT_master_data.txt every
    // ~5ms, reads ALL lines and clears the file. Because MASTER_TICK is written on
    // every tick while MARKET_SNAPSHOT / MASTER_INIT are written less often, a
    // truncating write would clobber the snapshot before the agent can read it —
    // which made the engine receive ticks but never snapshots (silent data feed).
    // Opening read+write and seeking to the end lets ticks AND snapshots coexist
    // in the file until the agent drains them.
    int retry = 0;
    while(retry < 3)
    {
        int h = FileOpen(PAT_MASTER_FILE, FILE_READ | FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON | FILE_SHARE_READ | FILE_SHARE_WRITE);
        if(h != -1)
        {
            FileSeek(h, 0, SEEK_END);
            FileWriteString(h, content);
            FileClose(h);
            return;
        }
        retry++;
        Sleep(5);
    }
    // Self-heal: if the append keeps failing (err 5004 = file too long, or a
    // transient lock race with the Agent), reset the file with a truncating
    // write and record the message so the market-data feed never dies.
    int h = FileOpen(PAT_MASTER_FILE, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON | FILE_SHARE_READ | FILE_SHARE_WRITE);
    if(h != -1)
    {
        FileWriteString(h, content);
        FileClose(h);
        Print("MasterAppend self-heal: reset ", PAT_MASTER_FILE, " (prev err ", GetLastError(), ")");
        return;
    }
    Print("FileOpen APPEND failed after retries + self-heal: ", PAT_MASTER_FILE, " error=", GetLastError());
}

//+------------------------------------------------------------------+
//| Update on-chart panel                                             |
//+------------------------------------------------------------------+
void UpdatePanel()
{
    string p = "=== PAT Master Node v1.00 ===\n";
    p += "Mode: DATA COLLECTION ONLY\n";
    p += "NO License Key · NO Trading\n";
    p += "Agent:    " + g_connection + "\n";
    p += "Broker:   " + g_broker + "\n";
    p += "Account:  " + g_accountID + "\n";
    p += "Symbol:   " + g_symbol + "\n";
    p += "-----------------------------\n";
    p += "Ticks Sent:    " + IntegerToString((long)g_tickCount) + "\n";
    p += "Snapshots Sent: " + IntegerToString((long)g_snapshotCount) + "\n";
    p += "-----------------------------\n";
    p += "Indicators: " + (SendIndicators ? "ON" : "OFF") + "\n";
    p += "Multi-TF:  " + (SendMultiTF ? "ON" : "OFF") + "\n";
    p += "Acct Info: " + (SendAccountInfo ? "ON" : "OFF") + "\n";
    p += "Sym Info:  " + (SendSymbolInfo ? "ON" : "OFF") + "\n";
    Comment(p);
}
//+------------------------------------------------------------------+
