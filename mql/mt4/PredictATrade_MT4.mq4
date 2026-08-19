//+------------------------------------------------------------------+
//|                                          PredictATrade_MT4.mq4   |
//|                            Predict-A-Trade v1.0.0                |
//|        Tick data collection + licensed signal execution EA        |
//|  IPC: FILE_COMMON folder (shared between all MT terminals)       |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property version   "1.03"
#property strict

//=== Input Parameters ===
input bool    AutoExecute    = false;    // SIGNAL_ONLY=false, AUTO=true
input bool    SendTickData   = true;     // Send real tick data to Windows Agent
input int     MagicNumber    = 20240001;
input int     TickIntervalMs = 0;      // 0 = every tick (HFT: 1-5ms co-located)
input string  BrokerSymbol   = "";       // Empty = auto-detect chart symbol
input string  LicenseKey     = "";       // Your Predict-A-Trade license key

//=== File names (in FILE_COMMON folder — shared with Windows Agent) ===
#define PAT_TICK_FILE    "PAT_ticks.txt"
#define PAT_SIGNAL_FILE  "PAT_signals.txt"
#define PAT_LICENSE_FILE "PAT_license.txt"
#define PAT_HEARTBEAT    "PAT_heartbeat.txt"
#define PAT_ACK_FILE     "PAT_ack.txt"

//=== Global State ===
string        g_symbol;
string        g_connection    = "OFFLINE";
string        g_licenseStatus  = "UNKNOWN";
string        g_licensePlan    = "—";
string        g_licenseKey    = "";
string        g_authStatus     = "UNKNOWN";    // AUTHENTICATED, AUTH_DEGRADED, AUTH_DISCONNECTED
string        g_deviceStatus   = "UNKNOWN";    // AUTHORIZED, UNAUTHORIZED
string        g_sessionStatus  = "UNKNOWN";    // ACTIVE, EXPIRED, REVOKED
string        g_tradingStatus  = "UNKNOWN";    // ACTIVE, HALTED, KILL_SWITCH, EMERGENCY_HALT, MARKET_CLOSED
long          g_signalSeq      = 0;            // Last received signal sequence for recovery
string        g_lastAckSeq     = "";            // Last acknowledged signal sequence
string        g_accountID     = "—";
string        g_signalID       = "";
string        g_signalDirection = "NONE";
string        g_signalGrade     = "—";
string        g_signalStrategy  = "—";
double        g_entry  = 0;
double        g_sl     = 0;
double        g_tp1    = 0;
double        g_tp2    = 0;
double        g_tp3    = 0;
datetime      g_signalTime = 0;
string        g_lastExecutedSignalID = "";
uint          g_lastTickSend = 0;
int           g_tickCount    = 0;

//+------------------------------------------------------------------+
int OnInit()
{
    Print("Predict-A-Trade MT4 EA v1.03 initializing...");
    
    g_symbol = BrokerSymbol;
    if(g_symbol == "") g_symbol = Symbol();
    g_licenseKey = LicenseKey;
    g_accountID = DoubleToStr(AccountNumber(), 0);
    
    Print("Symbol: ", g_symbol);
    Print("Account: ", g_accountID);
    Print("License Key: ", (g_licenseKey == "" ? "NOT SET" : g_licenseKey));
    
    // Check if Windows Agent is running (heartbeat file in common folder)
    if(FileIsExist(PAT_HEARTBEAT, FILE_COMMON))
    {
        g_connection = "CONNECTED";
        Print("Windows Agent detected (heartbeat found in common folder)");
        SendInitMessage();
        RequestLicenseValidation();
    }
    else
    {
        g_connection = "OFFLINE";
        Print("WARNING: Windows Agent not detected.");
        Print("Ensure PredictATradeAgent.exe is running on this machine.");
    }
    
    UpdatePanel();
    return(INIT_SUCCEEDED);
}

//+------------------------------------------------------------------+
void OnDeinit(const int reason)
{
    PAT_Write(PAT_TICK_FILE, "DEINIT|{}\n");
    Comment("");
}

//+------------------------------------------------------------------+
void OnTick()
{
    CheckAgentConnection();
    
    if(SendTickData && g_connection == "CONNECTED")
        SendTickToAgent();
    
    if(g_connection == "CONNECTED")
        ReadFromAgent();
    
    if(g_signalDirection != "NONE" && g_signalTime > 0)
    {
        if(TimeCurrent() > g_signalTime + 300)
            g_signalDirection = "EXPIRED";
    }
    
    UpdatePanel();
}

//+------------------------------------------------------------------+
void CheckAgentConnection()
{
    static uint lastCheck = 0;
    if(GetTickCount() - lastCheck < 2000) return;
    lastCheck = GetTickCount();
    
    if(FileIsExist(PAT_HEARTBEAT, FILE_COMMON))
        g_connection = "CONNECTED";
    else
    {
        if(g_connection == "CONNECTED")
        {
            Print("Windows Agent heartbeat lost");
            g_connection = "OFFLINE";
        }
    }
}

//+------------------------------------------------------------------+
//| Send real tick data — EA writes to FILE_COMMON, Agent reads      |
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
    
    g_tickCount++;
    
    string msg = "TICK|{\"type\":\"TICK\"";
    msg += ",\"symbol\":\"" + g_symbol + "\"";
    msg += ",\"bid\":" + DoubleToStr(bid, 5);
    msg += ",\"ask\":" + DoubleToStr(ask, 5);
    msg += ",\"volume\":" + IntegerToString((long)Volume[0]);
    msg += ",\"timestamp\":\"" + TimeToStr(TimeCurrent(), TIME_DATE|TIME_SECONDS) + "\"";
    msg += ",\"source\":\"MT4\"";
    msg += ",\"broker\":\"" + AccountCompany() + "\"";
    msg += ",\"account\":\"" + g_accountID + "\"";
    msg += ",\"license_key\":\"" + g_licenseKey + "\"";
    msg += "}\n";
    
    PAT_Append(PAT_TICK_FILE, msg);
}

//+------------------------------------------------------------------+
void SendInitMessage()
{
    string msg = "INIT|{\"ea_version\":\"1.02\",\"broker\":\"" + AccountCompany() +
                 "\",\"account\":\"" + g_accountID + "\",\"symbol\":\"" + g_symbol +
                 "\",\"license_key\":\"" + g_licenseKey + "\"}\n";
    PAT_Write(PAT_TICK_FILE, msg);
}

//+------------------------------------------------------------------+
void RequestLicenseValidation()
{
    string msg = "LICENSE_CHECK|{\"account\":\"" + g_accountID +
                 "\",\"broker\":\"" + AccountCompany() +
                 "\",\"symbol\":\"" + g_symbol +
                 "\",\"license_key\":\"" + g_licenseKey + "\"}\n";
    PAT_Append(PAT_TICK_FILE, msg);
    Print("License validation requested for key: ", g_licenseKey);
}

//+------------------------------------------------------------------+
void ReadFromAgent()
{
    // Read license response
    if(FileIsExist(PAT_LICENSE_FILE, FILE_COMMON))
    {
        string content = PAT_Read(PAT_LICENSE_FILE);
        if(StringLen(content) > 0)
        {
            HandleLicenseResponse(content);
            PAT_Clear(PAT_LICENSE_FILE);
        }
    }
    
    // Read signals
    if(FileIsExist(PAT_SIGNAL_FILE, FILE_COMMON))
    {
        string content = PAT_Read(PAT_SIGNAL_FILE);
        if(StringLen(content) > 0)
        {
            string lines[];
            int count = StringSplit(content, '\n', lines);
            for(int i = 0; i < count; i++)
            {
                string line = lines[i];
                if(StringLen(line) == 0) continue;
                int sep = StringFind(line, "|");
                if(sep < 0) continue;
                string msgType = StringSubstr(line, 0, sep);
                string payload = StringSubstr(line, sep + 1);
                if(msgType == "SIGNAL") HandleSignal(payload);
            }
            PAT_Clear(PAT_SIGNAL_FILE);
        }
    }
}

//+------------------------------------------------------------------+
void HandleSignal(string json)
{
    g_signalID        = ExtractJSONString(json, "ID");
    g_signalDirection = ExtractJSONString(json, "Direction");
    g_signalGrade     = ExtractJSONString(json, "Grade");
    g_signalStrategy  = ExtractJSONString(json, "StrategyID");
    g_entry  = ExtractJSONDouble(json, "EntryPrice");
    g_sl     = ExtractJSONDouble(json, "StopLoss");
    g_tp1    = ExtractJSONDouble(json, "TP1");
    g_tp2    = ExtractJSONDouble(json, "TP2");
    g_tp3    = ExtractJSONDouble(json, "TP3");
    g_signalTime = TimeCurrent();
    
    if(g_signalID == g_lastExecutedSignalID) return;
    // Trading halt does NOT mean disconnected — check states independently
    if(g_connection != "CONNECTED") { return; }
    if(g_licenseStatus != "ACTIVE") { Print("License not active — signal ignored"); return; }
    if(g_tradingStatus == "HALTED" || g_tradingStatus == "KILL_SWITCH" || g_tradingStatus == "EMERGENCY_HALT") {
        Print("Trading is ", g_tradingStatus, " — signal received but not executed. Connection remains ONLINE.");
        return;
    }
    if(g_tradingStatus == "MARKET_CLOSED") {
        Print("Market is closed — signal ignored. Connection remains ONLINE.");
        return;
    }
    
    Print("Signal: ", g_signalDirection, " | ", g_signalStrategy, " | ", g_signalGrade);
    
    if(AutoExecute && g_signalDirection == "BUY")  ExecuteBuy();
    else if(AutoExecute && g_signalDirection == "SELL") ExecuteSell();
}

//+------------------------------------------------------------------+
void HandleLicenseResponse(string json)
{
    g_licenseStatus = ExtractJSONString(json, "status");
    g_licensePlan   = ExtractJSONString(json, "plan");
    g_authStatus    = ExtractJSONString(json, "auth");
    g_deviceStatus  = ExtractJSONString(json, "device");
    g_sessionStatus = ExtractJSONString(json, "session");
    g_tradingStatus = ExtractJSONString(json, "trading");
    long newSeq = (long)ExtractJSONDouble(json, "seq");
    if(newSeq > 0) g_signalSeq = newSeq;
    Print("License: ", g_licenseStatus, " Auth: ", g_authStatus, " Device: ", g_deviceStatus, " Session: ", g_sessionStatus, " Trading: ", g_tradingStatus);
}

//+------------------------------------------------------------------+
void ExecuteBuy()
{
    if(g_entry <= 0 || g_sl <= 0) return;
    double vol = CalculateLotSize();
    int ticket = OrderSend(g_symbol, OP_BUY, vol, g_entry, 5, g_sl, g_tp1,
                           "PAT:" + g_signalID, MagicNumber, 0, clrGreen);
    if(ticket > 0)
    {
        g_lastExecutedSignalID = g_signalID;
        Print("BUY executed: ticket ", ticket);
        PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|{\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"}\n");
    }
    else
    {
        Print("BUY FAILED: error ", GetLastError());
    }
}

//+------------------------------------------------------------------+
void ExecuteSell()
{
    if(g_entry <= 0 || g_sl <= 0) return;
    double vol = CalculateLotSize();
    int ticket = OrderSend(g_symbol, OP_SELL, vol, g_entry, 5, g_sl, g_tp1,
                           "PAT:" + g_signalID, MagicNumber, 0, clrRed);
    if(ticket > 0)
    {
        g_lastExecutedSignalID = g_signalID;
        Print("SELL executed: ticket ", ticket);
        PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|{\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"}\n");
    }
    else
    {
        Print("SELL FAILED: error ", GetLastError());
    }
}

//+------------------------------------------------------------------+
double CalculateLotSize()
{
    double balance = AccountBalance();
    double risk = balance * 0.01;
    double slDist = MathAbs(g_entry - g_sl);
    double tickVal = MarketInfo(g_symbol, MODE_TICKVALUE);
    double tickSize = MarketInfo(g_symbol, MODE_TICKSIZE);
    if(slDist <= 0 || tickVal <= 0 || tickSize <= 0) return 0.01;
    double lot = risk / ((slDist / tickSize) * tickVal);
    double minLot = MarketInfo(g_symbol, MODE_MINLOT);
    double maxLot = MarketInfo(g_symbol, MODE_MAXLOT);
    double step = MarketInfo(g_symbol, MODE_LOTSTEP);
    lot = MathMax(minLot, MathMin(maxLot, lot));
    return MathRound(lot / step) * step;
}

//+------------------------------------------------------------------+
//| File I/O using FILE_COMMON (shared between all MT terminals)     |
//+------------------------------------------------------------------+
void PAT_Write(string filename, string content)
{
    int retry = 0;
    while(retry < 3)
    {
        int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
        if(h != -1)
        {
            FileWriteString(h, content);
            FileClose(h);
            return;
        }
        retry++;
        Sleep(5);
    }
    Print("FileOpen WRITE failed after 3 retries: ", filename, " error=", GetLastError());
}

void PAT_Append(string filename, string content)
{
    int retry = 0;
    while(retry < 3)
    {
        int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
        if(h != -1)
        {
            FileWriteString(h, content);
            FileClose(h);
            return;
        }
        retry++;
        Sleep(5);
    }
}

string PAT_Read(string filename)
{
    int h = FileOpen(filename, FILE_READ | FILE_TXT | FILE_ANSI | FILE_COMMON);
    if(h == -1) return "";
    string content = "";
    while(!FileIsEnding(h))
    {
        string line = FileReadString(h);
        if(StringLen(line) > 0)
        {
            if(StringLen(content) > 0) content += "\n";
            content += line;
        }
    }
    FileClose(h);
    return content;
}

void PAT_Clear(string filename)
{
    int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
    if(h != -1) FileClose(h);
}

//+------------------------------------------------------------------+
string ExtractJSONString(string json, string key)
{
    string sk = "\"" + key + "\":\"";
    int s = StringFind(json, sk);
    if(s < 0) return "";
    s += StringLen(sk);
    int e = StringFind(json, "\"", s);
    if(e < 0) return "";
    return StringSubstr(json, s, e - s);
}

double ExtractJSONDouble(string json, string key)
{
    string sk = "\"" + key + "\":";
    int s = StringFind(json, sk);
    if(s < 0) return 0;
    s += StringLen(sk);
    string v = "";
    for(int i = s; i < StringLen(json); i++)
    {
        int c = StringGetCharacter(json, i);
        if(c == 44 || c == 125 || c == 32) break;
        v += CharToString((uchar)c);
    }
    return StrToDouble(v);
}

//+------------------------------------------------------------------+
void SendSignalAck(string signalID, long seq)
{
    if(seq <= 0) return;
    string ack = "SIGNAL_ACK|{\"signal_id\":\"" + signalID + "\",\"seq\":" + IntegerToString(seq) + ",\"status\":\"EXECUTED\",\"timestamp\":" + IntegerToString(TimeCurrent()) + "}\n";
    PAT_Write(PAT_ACK_FILE, ack);
    g_lastAckSeq = IntegerToString(seq);
    Print("Signal ACK sent: signal=", signalID, " seq=", seq);
}

//+------------------------------------------------------------------+
void UpdatePanel()
{
    string p = "=== Predict-A-Trade v1.03 ===\n";
    p += "Connection: " + g_connection + "\n";
    p += "Auth:       " + g_authStatus + "\n";
    p += "License:    " + g_licenseStatus + " (" + g_licensePlan + ")\n";
    p += "Device:     " + g_deviceStatus + "\n";
    p += "Session:    " + g_sessionStatus + "\n";
    p += "Trading:    " + g_tradingStatus + "\n";
    p += "License Key: " + (g_licenseKey == "" ? "NOT SET" : g_licenseKey) + "\n";
    p += "Account:  " + g_accountID + "\n";
    p += "Symbol:   " + g_symbol + "\n";
    p += "Mode:     " + (AutoExecute ? "AUTO EXECUTE" : "SIGNAL ONLY") + "\n";
    p += "------------------------------\n";
    p += "Signal:   " + g_signalDirection + "\n";
    if(g_signalDirection != "NONE" && g_signalDirection != "EXPIRED")
    {
        p += "Strategy: " + g_signalStrategy + "\n";
        p += "Grade:    " + g_signalGrade + "\n";
        p += "Entry:    " + DoubleToStr(g_entry, 2) + "\n";
        p += "SL:       " + DoubleToStr(g_sl, 2) + "\n";
        p += "TP1:      " + DoubleToStr(g_tp1, 2) + "\n";
    }
    p += "------------------------------\n";
    p += "Ticks:    " + DoubleToStr(g_tickCount, 0) + "\n";
    Comment(p);
}
//+------------------------------------------------------------------+
