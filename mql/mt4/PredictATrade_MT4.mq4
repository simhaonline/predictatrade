//+------------------------------------------------------------------+
//|                                          PredictATrade_MT4.mq4   |
//|                              Predict-A-Trade v1.19 (Option B)    |
//+------------------------------------------------------------------+
//| ARCHITECTURE: THIN EXECUTOR (FINAL - NO RECOMPILE NEEDED)        |
//|                                                                  |
//| The Go backend engine is the SOLE authority for:                 |
//|   - Risk calculation (lot size, risk%, position limits)          |
//|   - SL/TP calculation (ATR multipliers, percentage geometry)     |
//|   - Spread checks (SpreadGate)                                   |
//|   - Signal TTL / expiry / entry drift                            |
//|   - Margin / exposure / broker profile constraints               |
//|   - Subscription plan -> strategy allocation                     |
//|   - Trade management commands (CLOSE_POSITION, EMERGENCY_STOP)   |
//|                                                                  |
//| The EA ONLY does:                                                |
//|   1. Receives signal from server (via Windows Agent)             |
//|   2. Checks: LicenseKey is ACTIVE                                |
//|   3. Executes trade with server-provided SL/TP/lot               |
//|   4. Watchdog: verifies SL present every 15s (fail-closed)       |
//|   5. Reports EXECUTION_ACK + TRADE_RESULT back to server         |
//|                                                                  |
//| USER INPUTS (only 3 - shown in EA Properties dialog):            |
//|   - LicenseKey: Your Predict-A-Trade license key                 |
//|   - AutoExecute: true = auto-trade, false = display only         |
//|   - ExecuteCandidates: true = execute candidate signals too      |
//|                                                                  |
//| ALL other parameters are hardcoded with safe defaults.           |
//| Changes to risk/strategy/trade management are made on the        |
//| SERVER - no EA recompile required.                               |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property version   "1.27"
#property strict

// ─── Signal/Execution inputs ───
input bool    AutoExecute    = false;   // SIGNAL_ONLY=true by default (display only). Set true to auto-trade.
input bool    BypassDailyLossBlock = false; // Allow new trades even after the soft daily-loss limit is hit. Hard halt (close-all at MaxDailyLossPct) is NEVER bypassed.
input string  LicenseKey     = "";
input string  PATCloudURL    = "https://api.predictatrade.com"; // Cloud API base URL (add to WebRequest allowlist)
input string  ChartTimeframe = "M1";
input int     PATPollMs      = 3000;    // Edge-poll interval, ms (>=1000; ULTRA TTL is 3m so 3s is safe)    // Chart/timeframe this EA instance trades (M1/M5/H1/...)

// ─── Strategy/Direction filters ───
// Strategy selection is SERVER-CONTROLLED based on your license plan.
// Just enter your License Key — the server handles strategy filtering.


input bool    ExecuteCandidates  = false;  // Execute BUY_CANDIDATE/SELL_CANDIDATE as real trades


// ─── Execution Safety (mql-fix.md — fail-closed) ───


// ─── Position Sizing ───

// ─── Constants ───
//─-- Internal constants (managed by backend, do not change) ──
#define SendTickData true  // ticks POST straight to the cloud engine
#define TickIntervalMs 0
#define BrokerSymbol ""
#define ReceiveBuy true
#define ReceiveSell true
#define ReceiveBuyCandidate true
#define ReceiveSellCandidate true
#define UseTrailingStop true
#define TrailingATRMult 2.0
#define UseBreakEven true
#define MaxHoldHours 4
#define UsePartialClose true
#define TP1ClosePct 33.33
#define TP2ClosePct 33.33
#define TP3TrailATRMult 1.5
#define AvoidSwapCharges true
#define SwapCutoffHour 22
#define SwapCutoffBuffer 15
#define AvoidTripleSwapDay true
#define TripleSwapDay "Wednesday"
#define MaxSlippagePoints 3
#define RejectOnHighSlippage true
#define MaxDailyLossPct 6.0
#define WarningLossPct 3.0
#define EmergencyCloseAll true
#define BaseLot 0.01
#define MaxLotRatioVsBase 1.0
#define MaxSameDirPositions 1
#define MaxTotalPositions 2
#define MaxMarginUsagePct 30.0
#define MaxSignalAgeSeconds 300
#define MinEquityFloorPct 40.0
#define OnMissingSL "CLOSE"
#define ReEnableAfterHalt false
#define UltraScalp_MaxSlippage 5
#define StdScalp_MaxSlippage 10
#define StdSwing_MaxSlippage 20
#define TrendSwing_MaxSlippage 30
// v1.25: per-strategy entry-drift budgets (points). The EA executes at CURRENT
// market price with the signal's SL — if price ran from EntryPrice between the
// engine decision and this EA's poll (spikes move XAUUSD 5-10 pts in seconds),
// the R:R geometry is silently distorted. Beyond this budget the EA refuses
// the signal (fail-closed, counts as filtered). Budgets > per-strategy
// slippage so normal latency (2-15s) never trips them; spikes do.
#define UltraScalp_MaxEntryDrift 15
#define StdScalp_MaxEntryDrift 25
#define StdSwing_MaxEntryDrift 60
#define TrendSwing_MaxEntryDrift 80
#define RiskPerTradePct 1.0
#define UseAutoLotSizing true

// Client MT terminal log — formatted [Predict-A-Trade] lines written here (FILE_COMMON)
// and echoed to the MT Experts log so the trader can see status/signal activity.
#define PAT_ERROR_LOG   "error_mt4.log" // MT4-specific: MT5 client writes error.log in the same FILE_COMMON folder

// Strategy magic bases (mql-fix.md convention; +offset within 100 range)
#define MAGIC_BASE_SS   40101
#define MAGIC_BASE_US   40201
#define MAGIC_BASE_SW   40301
#define MAGIC_BASE_TS   40401
#define MAGIC_BASE_MF   40501
#define PAT_MAGIC_MIN   40101
#define PAT_MAGIC_MAX   40600
#define PAT_REG_MAX     64

// ─── Global state ───
string  g_connection      = "OFFLINE";
string  g_licenseStatus    = "PENDING";
string  g_licensePlan      = "";
string  g_allowedStrategies = "";  // Server-provided from license
bool    g_strategiesEnforced = false; // W2: true when server sent allowed_strategies (empty → deny all)
double  g_suggestedLot   = 0;     // Server-calculated lot size
string  g_licenseKey       = "";
string  g_authStatus       = "UNKNOWN";
string  g_deviceStatus     = "UNKNOWN";
string  g_sessionStatus    = "UNKNOWN";
string  g_tradingStatus    = "UNKNOWN";
string  g_accountID        = "";
string  g_symbol           = "";
string  g_signalID         = "";
string  g_signalDirection  = "NONE";
string  g_signalGrade      = "";
string  g_signalStrategy   = "";
string  g_signalClass       = "";
string  g_lastExecutedSignalID = "";
datetime g_signalTime      = 0;
double  g_entry  = 0;
double  g_sl     = 0;
double  g_tp1    = 0;
double  g_tp2    = 0;
double  g_tp3    = 0;
double  g_rawScore = 0;
double  g_calibProb = 0;
uint    g_lastTickSend     = 0;
int     g_tickCount        = 0;
int     g_signalsReceived = 0;
int     g_signalsDisplayed = 0;
int     g_signalsFiltered  = 0;
// Capital protection state
double  g_dailyPnL       = 0;
double  g_dayStartBalance = 0;
datetime g_currentDay    = 0;
 bool    g_tradingBlocked = false;
 bool    g_capitalWarnActive = false;  // edge-trigger for CAPITAL_WARNING emission
 bool    g_hardHaltTriggered = false;
int     g_slippageRejects = 0;
// Execution safety state
bool    g_equityHalted   = false;
int     g_magicSeq       = 0;
// Last raw signal payload (for ExpiresAt extraction)
string  g_lastSignalJSON = "";

// Per-trade registry (runtime, keyed by magic; survives reload via
// GlobalVariables + order-comment reconstruction)
int     g_regMagic[PAT_REG_MAX];
string  g_regSig[PAT_REG_MAX];
string  g_regStrat[PAT_REG_MAX];
double  g_regEntry[PAT_REG_MAX];
double  g_regSL0[PAT_REG_MAX];
double  g_regTP1[PAT_REG_MAX];
double  g_regTP2[PAT_REG_MAX];
double  g_regTP3[PAT_REG_MAX];
double  g_regOrigLot[PAT_REG_MAX];
int     g_regCount = 0;

//+------------------------------------------------------------------+
//| Strategy mapping                                                  |
//+------------------------------------------------------------------+
int PAT_StrategyMagicBase(string strategyName)
{
    if(strategyName == "ULTRA_SCALPING")    return MAGIC_BASE_US;
    if(strategyName == "STANDARD_SWING")    return MAGIC_BASE_SW;
    if(strategyName == "TREND_SWING")       return MAGIC_BASE_TS;
    if(strategyName == "MARNIE_FIB")        return MAGIC_BASE_MF;
    if(strategyName == "STANDARD_SCALPING") return MAGIC_BASE_SS;
    return 0; // unknown strategy — caller must reject (fail-closed)
}

string PAT_StrategyPrefix(string strategyName)
{
    if(strategyName == "ULTRA_SCALPING")    return "PAT-US:";
    if(strategyName == "STANDARD_SWING")    return "PAT-SW:";
    if(strategyName == "TREND_SWING")       return "PAT-TS:";
    if(strategyName == "MARNIE_FIB")        return "PAT-MF:";
    if(strategyName == "STANDARD_SCALPING") return "PAT-SS:";
    return "";
}

bool PAT_IsPatMagic(int magic)
{
    return (magic >= PAT_MAGIC_MIN && magic <= PAT_MAGIC_MAX);
}

//+------------------------------------------------------------------+
//| Unified Price Access Wrappers                                     |
//+------------------------------------------------------------------+
double PAT_Close(int shift) { return iClose(g_symbol, 0, shift); }
double PAT_Open(int shift)  { return iOpen(g_symbol, 0, shift); }
double PAT_High(int shift)  { return iHigh(g_symbol, 0, shift); }
double PAT_Low(int shift)   { return iLow(g_symbol, 0, shift); }

double PAT_SwapLong()   { return MarketInfo(g_symbol, MODE_SWAPLONG); }
double PAT_SwapShort()  { return MarketInfo(g_symbol, MODE_SWAPSHORT); }
double PAT_TickValue()  { return MarketInfo(g_symbol, MODE_TICKVALUE); }
double PAT_TickSize()   { return MarketInfo(g_symbol, MODE_TICKSIZE); }
double PAT_PointValue() { return (PAT_TickValue() / PAT_TickSize()); }

//+------------------------------------------------------------------+
//| Lot normalization to broker volume step/min/max                   |
//+------------------------------------------------------------------+
double PAT_NormalizeLot(double lots)
{
    double lotStep = MarketInfo(g_symbol, MODE_LOTSTEP);
    double minLot  = MarketInfo(g_symbol, MODE_MINLOT);
    double maxLot  = MarketInfo(g_symbol, MODE_MAXLOT);
    if(lotStep > 0) lots = MathFloor(lots / lotStep + 0.0000001) * lotStep;
    if(lots > maxLot) lots = maxLot;
    if(lots < minLot) return 0; // below broker minimum — caller rejects (fail-closed)
    return NormalizeDouble(lots, 2);
}

// ─── Position Sizing ───
double PAT_CalcLotSize(double equity, double stopDistancePrice)
{
    if(stopDistancePrice <= 0 || equity <= 0) return 0;
    double riskAmount = equity * (RiskPerTradePct / 100.0);
    double pointValue = PAT_PointValue();
    if(pointValue <= 0) return 0;
    double lots = riskAmount / (stopDistancePrice * pointValue);
    double norm = PAT_NormalizeLot(lots);
    return norm;
}

// ─── Per-Strategy Spread Check — SERVER HANDLES, always pass ───
bool PAT_CheckSpread(string strategyName)
{
    // Spread checked by SERVER SpreadGate — always pass
    return true;
}

int PAT_GetMaxSlippage(string strategyName)
{
    if(strategyName == "ULTRA_SCALPING") return UltraScalp_MaxSlippage;
    if(strategyName == "STANDARD_SCALPING") return StdScalp_MaxSlippage;
    if(strategyName == "STANDARD_SWING") return StdSwing_MaxSlippage;
    if(strategyName == "TREND_SWING") return TrendSwing_MaxSlippage;
    return MaxSlippagePoints;
}

//+------------------------------------------------------------------+
//| v1.25: ENTRY-DRIFT GATE — refuse signals whose entry geometry    |
//| is already stale. Compares CURRENT market price against the      |
//| signal's EntryPrice; beyond the per-strategy budget the R:R the  |
//| engine sized for no longer holds, so we fail closed.             |
//+------------------------------------------------------------------+
int PAT_GetMaxEntryDrift(string strategyName)
{
    if(strategyName == "ULTRA_SCALPING") return UltraScalp_MaxEntryDrift;
    if(strategyName == "STANDARD_SCALPING") return StdScalp_MaxEntryDrift;
    if(strategyName == "STANDARD_SWING") return StdSwing_MaxEntryDrift;
    if(strategyName == "TREND_SWING") return TrendSwing_MaxEntryDrift;
    if(strategyName == "MARNIE_FIB") return TrendSwing_MaxEntryDrift;   // 120-240m TTL zone strategy — widest budget
    if(strategyName == "ATEN") return TrendSwing_MaxEntryDrift;         // 60m TTL swing-class
    return StdSwing_MaxEntryDrift; // unknown strategy — conservative swing-class budget
}

bool PAT_EntryDriftOK(string strategyName, bool isBuy)
{
    if(g_entry <= 0) return true; // no entry reference in signal — nothing to compare
    double price = isBuy ? Ask : Bid;
    if(price <= 0) return false; // cannot verify market — fail closed
    double driftPts = MathAbs(price - g_entry) / Point;
    int budget = PAT_GetMaxEntryDrift(strategyName);
    if(driftPts > budget)
    {
        Print("SIGNAL REJECTED (entry drift): strategy=", strategyName,
              " dir=", (isBuy ? "BUY" : "SELL"),
              " signal_entry=", DoubleToString(g_entry, Digits),
              " market=", DoubleToString(price, Digits),
              " drift=", DoubleToString(driftPts, 1),
              "pt budget=", budget, "pt — R:R geometry stale, NO-TRADE");
        return false;
    }
    return true;
}

//+------------------------------------------------------------------+
//| WRONG-SIDE SL/TP VALIDATION (highest priority — fail-closed).     |
//| Never clamps. Aborts the order on any violation.                  |
//+------------------------------------------------------------------+
bool PAT_ValidateLevels(bool isBuy, double entry, double sl, double tpFinal)
{
    if(entry <= 0 || sl <= 0 || tpFinal <= 0)
    {
        Print("REJECTED invalid_levels: entry=", DoubleToString(entry, _Digits),
              " sl=", DoubleToString(sl, _Digits),
              " tp=", DoubleToString(tpFinal, _Digits));
        return false;
    }
    if(isBuy)
    {
        if(sl >= entry)
        {
            Print("REJECTED wrong_side_sl: BUY entry=", DoubleToString(entry, _Digits),
                  " sl=", DoubleToString(sl, _Digits), " — SL must be BELOW entry. Order ABORTED.");
            return false;
        }
        if(tpFinal <= entry)
        {
            Print("REJECTED wrong_side_tp: BUY entry=", DoubleToString(entry, _Digits),
                  " tp=", DoubleToString(tpFinal, _Digits), " — TP must be ABOVE entry. Order ABORTED.");
            return false;
        }
    }
    else
    {
        if(sl <= entry)
        {
            Print("REJECTED wrong_side_sl: SELL entry=", DoubleToString(entry, _Digits),
                  " sl=", DoubleToString(sl, _Digits), " — SL must be ABOVE entry. Order ABORTED.");
            return false;
        }
        if(tpFinal >= entry)
        {
            Print("REJECTED wrong_side_tp: SELL entry=", DoubleToString(entry, _Digits),
                  " tp=", DoubleToString(tpFinal, _Digits), " — TP must be BELOW entry. Order ABORTED.");
            return false;
        }
    }
    return true;
}

//+------------------------------------------------------------------+
//| Signal expiry: prefer server-provided ExpiresAt, else age input   |
//+------------------------------------------------------------------+
datetime PAT_ParseISO8601UTC(string s)
{
    // Handles "2026-08-24T16:25:11[.frac][Z|+03:00|-05:00]"
    if(StringLen(s) < 19) return 0;
    int y  = (int)StringToInteger(StringSubstr(s, 0, 4));
    int mo = (int)StringToInteger(StringSubstr(s, 5, 2));
    int d  = (int)StringToInteger(StringSubstr(s, 8, 2));
    int h  = (int)StringToInteger(StringSubstr(s, 11, 2));
    int mi = (int)StringToInteger(StringSubstr(s, 14, 2));
    int se = (int)StringToInteger(StringSubstr(s, 17, 2));
    if(y < 2000 || mo < 1 || mo > 12 || d < 1 || d > 31) return 0;
    MqlDateTime dt;
    dt.year = y; dt.mon = mo; dt.day = d; dt.hour = h; dt.min = mi; dt.sec = se;
    datetime gmt = StructToTime(dt);
    // Apply timezone offset if present (not plain 'Z')
    int tzPos = 19;
    if(tzPos < StringLen(s))
    {
        string tzChar = StringSubstr(s, tzPos, 1);
        if(tzChar == "+" || tzChar == "-")
        {
            int sign = (tzChar == "+") ? -1 : 1; // local ahead of UTC => subtract to get UTC
            int oh = (int)StringToInteger(StringSubstr(s, tzPos + 1, 2));
            int om = 0;
            if(StringLen(s) >= tzPos + 6)
                om = (int)StringToInteger(StringSubstr(s, tzPos + 4, 2));
            gmt += sign * (oh * 3600 + om * 60);
        }
    }
    // Convert UTC -> approximate server timeline using terminal GMT offset
    return (gmt + (TimeCurrent() - TimeGMT()));
}

bool PAT_SignalFresh()
{
    datetime expiry = PAT_ParseISO8601UTC(ExtractJSONString(g_lastSignalJSON, "ExpiresAt"));
    if(expiry > 0)
    {
        if(TimeCurrent() > expiry)
        {
            Print("SIGNAL EXPIRED (server expiry): signal=", g_signalID,
                  " expired=", TimeToString(expiry, TIME_DATE|TIME_SECONDS));
            return false;
        }
        return true;
    }
    if(g_signalTime > 0 && TimeCurrent() > g_signalTime + MaxSignalAgeSeconds)
    {
        Print("SIGNAL EXPIRED (age>): signal=", g_signalID, " older than ", MaxSignalAgeSeconds, "s");
        return false;
    }
    return true;
}

//+------------------------------------------------------------------+
//| Position counting by PAT magic range                              |
//+------------------------------------------------------------------+
int PAT_CountPatPositions()
{
    int total = OrdersTotal();
    int count = 0;
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(!PAT_IsPatMagic(OrderMagicNumber())) continue;
        count++;
    }
    return count;
}

//+------------------------------------------------------------------+
//| Pre-trade check — EMERGENCY HALT FLAGS ONLY.                     |
//| v1.20: ALL risk gates (spread, drift, TTL, caps, risk$,          |
//| martingale, margin) are enforced by the SERVER engine before a   |
//| signal is marked EXECUTABLE. Client-side duplicates removed —    |
//| clients execute what the server sends. Server = single source    |
//| of gating truth; this hook exists only for local kill-switches.  |
//+------------------------------------------------------------------+
//| Pre-trade check — EMERGENCY HALT FLAGS ONLY.                     |
//| v1.20: ALL risk gates (spread, drift, TTL, caps, risk$,          |
//| martingale, margin) are enforced by the SERVER engine before a   |
//| signal is marked EXECUTABLE. Client-side duplicates removed —    |
//| clients execute what the server sends. Server = single source    |
//| of gating truth; this hook exists only for local kill-switches.  |
//+------------------------------------------------------------------+
bool PAT_PreTradeGate(bool isBuy, double lot, string strategyName)
{
    if(g_equityHalted)
    {
        Print("REJECTED equity_floor_halt: trading halted until manual re-enable");
        return false;
    }
    if(g_tradingBlocked)
    {
        Print("REJECTED capital_protection_halt: daily loss limit reached");
        return false;
    }
    return true;
}
//+------------------------------------------------------------------+
//| Per-trade registry (runtime)                                      |
//+------------------------------------------------------------------+
int PAT_RegFind(int magic)
{
    for(int i = 0; i < g_regCount; i++)
        if(g_regMagic[i] == magic) return i;
    return -1;
}

int PAT_RegPut(int magic, string sig, string strat, double entry, double sl0,
               double tp1, double tp2, double tp3, double origLot)
{
    int idx = PAT_RegFind(magic);
    if(idx < 0)
    {
        if(g_regCount >= PAT_REG_MAX) return -1;
        idx = g_regCount++;
    }
    g_regMagic[idx]   = magic;
    g_regSig[idx]     = sig;
    g_regStrat[idx]   = strat;
    g_regEntry[idx]   = entry;
    g_regSL0[idx]     = sl0;
    g_regTP1[idx]     = tp1;
    g_regTP2[idx]     = tp2;
    g_regTP3[idx]     = tp3;
    g_regOrigLot[idx] = origLot;
    return idx;
}

//+------------------------------------------------------------------+
//| Stage persistence (survives EA reload via GlobalVariables)        |
//+------------------------------------------------------------------+
string PAT_GVName(int magic, string field) { return "PAT_M" + IntegerToString(magic) + "_" + field; }

void PAT_SaveStage(int magic, int stage)
{
    GlobalVariableSet(PAT_GVName(magic, "STAGE"), stage);
}

int PAT_LoadStage(int magic)
{
    string name = PAT_GVName(magic, "STAGE");
    if(GlobalVariableCheck(name)) return (int)GlobalVariableGet(name);
    return 0;
}

//+------------------------------------------------------------------+
//| Trade result reporting (TRADE_RESULT + CLOSE_ACK via IPC file)    |
//+------------------------------------------------------------------+
 void PAT_ReportResult(int magic, int ticket, string signalID, string strategyID, string exitReason,
                       double entry, double exitPx, double lots, double realizedPnl,
                       bool slCorrect,
                       string p_timeframe, string p_direction, string p_openedAt,
                       double p_sl, double p_tp, double p_pnlPoints, int p_timeInTradeSec)
 {
    if(signalID == "" && strategyID == "")
    {
        int idx = PAT_RegFind(magic);
        if(idx >= 0) { signalID = g_regSig[idx]; strategyID = g_regStrat[idx]; }
    }
    string msg = "TRADE_RESULT|{";
    msg += "\"type\":\"TRADE_RESULT\"";
    msg += ",\"signal_id\":\"" + signalID + "\"";
    msg += ",\"strategy_id\":\"" + strategyID + "\"";
    msg += ",\"magic\":" + IntegerToString(magic);
    msg += ",\"ticket\":" + IntegerToString(ticket);
    msg += ",\"timeframe\":\"" + p_timeframe + "\"";
    msg += ",\"direction\":\"" + p_direction + "\"";
    msg += ",\"opened_at\":\"" + p_openedAt + "\"";
    msg += ",\"exit_reason\":\"" + exitReason + "\"";
    msg += ",\"entry\":" + DoubleToString(entry, _Digits);
    msg += ",\"exit\":" + DoubleToString(exitPx, _Digits);
    msg += ",\"stop_loss\":" + DoubleToString(p_sl, _Digits);
    msg += ",\"take_profit\":" + DoubleToString(p_tp, _Digits);
    msg += ",\"lot\":" + DoubleToString(lots, 2);
    msg += ",\"realized_pnl\":" + DoubleToString(realizedPnl, 2);
    msg += ",\"pnl_points\":" + DoubleToString(p_pnlPoints, 2);
    msg += ",\"time_in_trade_seconds\":" + IntegerToString(p_timeInTradeSec);
    msg += ",\"mae\":0.0";
    msg += ",\"mfe\":0.0";
    msg += ",\"sl_correct\":" + (slCorrect ? "true" : "false");
    msg += "}";
    PAT_Send(msg);

    // CLOSE_ACK kept for pipeline compatibility (the engine core parses it).
    string ack = "CLOSE_ACK|{";
    ack += "\"ticket\":" + IntegerToString(ticket);
    ack += ",\"reason\":\"" + exitReason + "\"";
    ack += ",\"net_pnl\":" + DoubleToString(realizedPnl, 2);
    ack += ",\"signal_id\":\"" + signalID + "\"";
    ack += ",\"strategy_id\":\"" + strategyID + "\"";
    ack += ",\"magic\":" + IntegerToString(magic);
    ack += "}\n";
    PAT_Send(ack);

    Print("TRADE_RESULT reported: magic=", magic, " reason=", exitReason,
          " pnl=", DoubleToString(realizedPnl, 2), " signal=", signalID);
}

// Mark an EA-initiated close so the history poller attributes the reason.
// Keyed by TICKET (partial chunks of one magic each get their own ticket).
void PAT_SetForcedReason(int ticket, string reason)
{
    GlobalVariableSet("PAT_FR_" + IntegerToString(ticket), HashCode(reason));
}

int HashCode(string s)
{
    int h = 0;
    for(int i = 0; i < StringLen(s); i++) h = h * 31 + StringGetChar(s, i);
    return h;
}

//+------------------------------------------------------------------+
//| Exit-reason classification by close price proximity               |
//+------------------------------------------------------------------+
string PAT_ClassifyExit(bool isBuy, double entry, double sl, double tp1, double tp2,
                        double tp3, double exitPx, string forcedReason)
{
    if(forcedReason != "") return forcedReason;
    double point = MarketInfo(g_symbol, MODE_POINT);
    double spread = MarketInfo(g_symbol, MODE_SPREAD) * point;
    double tol = MathMax(spread, 10 * point);
    if(tol <= 0) tol = 0.10;
    if(isBuy)
    {
        if(tp3 > 0 && exitPx >= tp3 - tol) return "tp3";
        if(tp2 > 0 && exitPx >= tp2 - tol) return "tp2";
        if(tp1 > 0 && exitPx >= tp1 - tol) return "tp1";
        if(sl > 0 && exitPx <= sl + tol) return "sl";
    }
    else
    {
        if(tp3 > 0 && exitPx <= tp3 + tol) return "tp3";
        if(tp2 > 0 && exitPx <= tp2 + tol) return "tp2";
        if(tp1 > 0 && exitPx <= tp1 + tol) return "tp1";
        if(sl > 0 && exitPx >= sl - tol) return "sl";
    }
    return "manual";
}

// Forced-reason codes -> strings
string PAT_ForcedReasonFromCode(int code)
{
    if(code == HashCode("MAX_HOLD_TIME"))    return "expiry";
    if(code == HashCode("SWAP_AVOIDANCE"))   return "manual";
    if(code == HashCode("EMERGENCY_CAPITAL_PROTECTION")) return "manual";
    if(code == HashCode("EQUITY_FLOOR"))     return "manual";
    if(code == HashCode("WATCHDOG_NOSL"))    return "manual";
    if(code == HashCode("SLIPPAGE_REJECT"))  return "manual";
    return "";
}

//+------------------------------------------------------------------+
//| History poller: report every closed PAT order exactly once        |
//| (covers broker-side SL/TP fills AND MT4 partial-close chunks)     |
//+------------------------------------------------------------------+
void PAT_HistoryPoll()
{
    int total = OrdersHistoryTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(!PAT_IsPatMagic(OrderMagicNumber())) continue;
        if(OrderCloseTime() == 0) continue;

        int ticket = OrderTicket();
        string rptName = "PAT_RPT_" + IntegerToString(ticket);
        if(GlobalVariableCheck(rptName)) continue;

        int magic = OrderMagicNumber();
        string forced = "";
        string frName = "PAT_FR_" + IntegerToString(ticket);
        if(GlobalVariableCheck(frName))
        {
            forced = PAT_ForcedReasonFromCode((int)GlobalVariableGet(frName));
            GlobalVariableDel(frName);
        }

        // Direction/type from the closed order itself
        bool isBuy = (OrderType() == OP_BUY);
        int idx = PAT_RegFind(magic);
        double entry = OrderOpenPrice();
        double sl0 = OrderStopLoss();
        double tp1 = 0, tp2 = 0, tp3 = 0;
        if(idx >= 0)
        {
            entry = g_regEntry[idx];
            sl0 = g_regSL0[idx];
            tp1 = g_regTP1[idx];
            tp2 = g_regTP2[idx];
            tp3 = g_regTP3[idx];
        }
        // Partial-close chunks carry the ORIGINAL comment ("from #X" suffix
        // varies by broker) — classify purely by close-price proximity unless
        // an explicit forced reason exists.
        string reason = PAT_ClassifyExit(isBuy, entry, sl0, tp1, tp2, tp3,
                                         OrderClosePrice(), forced);
        double pnl = OrderProfit() + OrderSwap() + OrderCommission();
        string sig = "", strat = "";
        if(idx >= 0) { sig = g_regSig[idx]; strat = g_regStrat[idx]; }
        else
        {
            sig = PAT_SignalIDFromComment(OrderComment());
            strat = PAT_StrategyFromMagic(magic);
        }
        double point = MarketInfo(g_symbol, MODE_POINT);
        double pnlPoints = (point > 0) ? (OrderClosePrice() - entry) / point * (isBuy ? 1 : -1) : 0;
        string dir = isBuy ? "BUY" : "SELL";
        string openedAt = FormatISO8601UTC(OrderOpenTime());
        int timeInTrade = (int)(OrderCloseTime() - OrderOpenTime());
        PAT_ReportResult(magic, ticket, sig, strat, reason, entry, OrderClosePrice(),
                         OrderLots(), pnl, true, ChartTimeframe, dir, openedAt,
                         sl0, tp1, pnlPoints, timeInTrade);
        GlobalVariableSet(rptName, 1);
    }
}

string PAT_StrategyFromMagic(int magic)
{
    if(magic >= MAGIC_BASE_SS && magic < MAGIC_BASE_SS + 100) return "STANDARD_SCALPING";
    if(magic >= MAGIC_BASE_US && magic < MAGIC_BASE_US + 100) return "ULTRA_SCALPING";
    if(magic >= MAGIC_BASE_SW && magic < MAGIC_BASE_SW + 100) return "STANDARD_SWING";
    if(magic >= MAGIC_BASE_TS && magic < MAGIC_BASE_TS + 100) return "TREND_SWING";
    if(magic >= MAGIC_BASE_MF && magic < MAGIC_BASE_MF + 100) return "MARNIE_FIB";
    return "";
}

string PAT_SignalIDFromComment(string comment)
{
    // Comment format: PAT-XX:<signal_id>
    int colon = StringFind(comment, ":");
    if(colon < 0) return "";
    return StringSubstr(comment, colon + 1);
}

//+------------------------------------------------------------------+
//| FormatISO8601UTC                                                  |
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
    if(BrokerSymbol != "")
        g_symbol = BrokerSymbol;
    else
        g_symbol = Symbol();

    g_accountID = IntegerToString(AccountNumber());
    g_licenseKey = LicenseKey;
    g_deviceFile = PAT_ComputeDeviceFile(); // v1.26: per-terminal bootstrap state
    g_connection = "OFFLINE";
    g_licenseStatus = "PENDING";

    Print("Predict-A-Trade MT4 EA v1.27 initializing...");
    Print("Symbol: ", g_symbol);
    Print("Account: ", g_accountID);
    Print("License Key: ", (g_licenseKey == "" ? "NOT SET — SIGNALS WILL BE IGNORED" : g_licenseKey));

    // Restore equity-floor halt latch (persists across reloads)
    if(GlobalVariableCheck("PAT_EQUITY_HALT"))
        g_equityHalted = (GlobalVariableGet("PAT_EQUITY_HALT") > 0.5);
    if(g_equityHalted && !ReEnableAfterHalt)
        Print("*** TRADING HALTED: equity-floor breach latched. Set ReEnableAfterHalt=true to resume. ***");
    else if(g_equityHalted && ReEnableAfterHalt)
    {
        Print("Manual re-enable accepted — clearing equity-floor halt latch.");
        g_equityHalted = false;
        GlobalVariableSet("PAT_EQUITY_HALT", 0);
    }

    // Watchdog timer (every 15s)
    EventSetTimer(15);

    // Option B: no local agent — activate the cloud device and go ONLINE when
    // credentials are ready. First edge-poll confirms reachability.
    if(PAT_EnsureDevice())
    {
        g_connection = "CONNECTED";
        Print("[Predict-A-Trade] Cloud device ready (", g_deviceId, ") — edge mode (v1.19).");
        SendInitMessage();
        RequestLicenseValidation();
    }
    else
    {
        g_connection = "OFFLINE";
        Print("WARNING: Cloud device not ready — set LicenseKey in EA inputs.");
        Print("Also add ", PATCloudURL, " to Tools→Options→Expert Advisors→WebRequest allowlist.");
    }

    UpdatePanel();
    return(INIT_SUCCEEDED);
}

void OnDeinit(const int reason)
{
    EventKillTimer();
    PAT_Send("DEINIT|{}");
    Comment("");
}

//+------------------------------------------------------------------+
//| MAIN TICK                                                         |
//+------------------------------------------------------------------+
void OnTick()
{
    CheckAgentConnection();

    if(SendTickData && g_connection == "CONNECTED")
        SendTickToAgent();

    PollFromCloud();

    // Signal expiry (display state)
    if(g_signalDirection != "NONE" && g_signalTime > 0)
    {
        if(TimeCurrent() > g_signalTime + MaxSignalAgeSeconds)
            g_signalDirection = "EXPIRED";
    }

    // Staged position management (partial TP1/TP2, breakeven, trailing)
    PAT_ManagePositions();

    // Capital protection — check daily loss limit
    UpdateCapitalProtection();

    // Exit reporting (covers broker-side closes and partial chunks)
    PAT_HistoryPoll();

    UpdatePanel();
}

//+------------------------------------------------------------------+
//| WATCHDOG (OnTimer, 15s): missing-SL check + equity floor          |
//+------------------------------------------------------------------+
void OnTimer()
{
    CheckAgentConnection();

    uint tickGap = (TickIntervalMs > 0 ? TickIntervalMs : 1000) * 3;
    if(g_connection == "CONNECTED" && (g_lastTickSend == 0 || GetTickCount() - g_lastTickSend > tickGap))
    {
        SendLivenessPing();
        g_lastTickSend = GetTickCount();
    }

    PAT_Watchdog();
    PAT_HistoryPoll();
    SendAccountInfo();

    // Control-plane heartbeat (HMAC) — every watchdog cycle (15s) keeps the
    // device liveness fresh in edge_device_state even when the engine ingest
    // is healthy but quiet (weekend).
    PAT_EdgeHeartbeat();
}

void PAT_Watchdog()
{
    // 1. Equity floor check (fail-closed: close ALL PAT positions and halt)
    if(!g_equityHalted)
    {
        double baseline = g_dayStartBalance;
        if(baseline <= 0) baseline = AccountBalance();
        double floorEq = baseline * (MinEquityFloorPct / 100.0);
        if(baseline > 0 && AccountEquity() < floorEq && AccountEquity() > 0)
        {
            g_equityHalted = true;
            GlobalVariableSet("PAT_EQUITY_HALT", 1);
            Print("*** EQUITY FLOOR BREACH: equity=", DoubleToString(AccountEquity(), 2),
                  " < floor=", DoubleToString(floorEq, 2), " — CLOSING ALL PAT POSITIONS AND HALTING ***");
            CloseAllPatPositions("EQUITY_FLOOR");
        }
    }

    // 2. Missing-SL self-check on every PAT position
    int total = OrdersTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(!PAT_IsPatMagic(OrderMagicNumber())) continue;

        double sl = OrderStopLoss();
        if(sl > 0) continue;

        int magic = OrderMagicNumber();
        bool isBuy = (OrderType() == OP_BUY);
        int idx = PAT_RegFind(magic);
        double sl0 = (idx >= 0) ? g_regSL0[idx] : 0;

        string mode = OnMissingSL;
        StringToUpper(mode);
        if(mode == "RESTORE" && sl0 > 0)
        {
            // Validate restored SL side BEFORE modifying (fail-closed)
            double openPx = OrderOpenPrice();
            bool sideOk = isBuy ? (sl0 < openPx) : (sl0 > openPx);
            if(sideOk)
            {
                if(OrderModify(OrderTicket(), openPx, sl0, OrderTakeProfit(), 0, clrYellow))
                    Print("WATCHDOG: restored missing SL on ticket=", OrderTicket(), " SL=", DoubleToString(sl0, _Digits));
                else
                    Print("WATCHDOG: SL restore FAILED ticket=", OrderTicket(), " err=", GetLastError());
            }
            else
            {
                Print("WATCHDOG: stored SL wrong-side for ticket=", OrderTicket(), " — CLOSING (fail-closed)");
                ClosePosition(OrderTicket(), "WATCHDOG_NOSL");
            }
        }
        else
        {
            Print("WATCHDOG: PAT position without SL ticket=", OrderTicket(),
                  " magic=", magic, " — CLOSING (OnMissingSL=CLOSE, fail-closed)");
            ClosePosition(OrderTicket(), "WATCHDOG_NOSL");
        }
    }
}

void CloseAllPatPositions(string reason)
{
    int total = OrdersTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() == g_symbol && PAT_IsPatMagic(OrderMagicNumber()))
            ClosePosition(OrderTicket(), reason);
    }
}

//+------------------------------------------------------------------+
//| STAGED POSITION MANAGEMENT                                        |
//| ONE position per signal: TP1 -> close ~1/3 + SL to breakeven,     |
//| TP2 -> close another ~1/3 + ATR trail remainder, TP3/final SL     |
//| exits the rest. NEVER opens additional full positions.            |
//+------------------------------------------------------------------+
void PAT_ManagePositions()
{
    int total = OrdersTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(!PAT_IsPatMagic(OrderMagicNumber())) continue;

        int    ticket   = OrderTicket();
        int    magic    = OrderMagicNumber();
        int    type     = OrderType();
        double openPx   = OrderOpenPrice();
        double sl       = OrderStopLoss();
        double tp       = OrderTakeProfit();
        double curLots  = OrderLots();
        datetime openTime = OrderOpenTime();

        // Max holding time
        if(MaxHoldHours > 0)
        {
            int holdSec = (int)(TimeCurrent() - openTime);
            if(holdSec >= MaxHoldHours * 3600)
            {
                Print("MAX HOLD TIME reached: ticket=", ticket, " held=", holdSec/3600, "h | Closing...");
                PAT_SetForcedReason(ticket, "MAX_HOLD_TIME");
                ClosePosition(ticket, "MAX_HOLD_TIME");
                continue;
            }
        }

        // Swap cutoff
        if(AvoidSwapCharges && IsNearSwapTime())
        {
            Print("SWAP CUTOFF: closing ticket=", ticket, " before swap charge");
            PAT_SetForcedReason(ticket, "SWAP_AVOIDANCE");
            ClosePosition(ticket, "SWAP_AVOIDANCE");
            continue;
        }

        int stage = PAT_LoadStage(magic);
        bool isBuy = (type == OP_BUY);
        double point = MarketInfo(g_symbol, MODE_POINT);
        int digits = (int)MarketInfo(g_symbol, MODE_DIGITS);

        int idx = PAT_RegFind(magic);
        double tp1 = (idx >= 0) ? g_regTP1[idx] : 0;
        double tp2 = (idx >= 0) ? g_regTP2[idx] : 0;
        double origLot = (idx >= 0) ? g_regOrigLot[idx] : curLots;
        double sl0 = (idx >= 0) ? g_regSL0[idx] : 0;

        // ── STAGE 0 -> 1: TP1 hit — partial close ──
        if(UsePartialClose && stage == 0 && tp1 > 0)
        {
            bool tp1Hit = isBuy ? (Bid >= tp1) : (Ask <= tp1);
            if(tp1Hit)
            {
                if(PAT_DoPartial(ticket, magic, isBuy, origLot, TP1ClosePct, "tp1"))
                {
                    PAT_SaveStage(magic, 1);
                    stage = 1;
                }
            }
        }

        // ── Breakeven maintenance (stage >= 1): SL to entry +/- spread. ──
        // Runs every tick until applied, so it also covers the MT4 remainder
        // ticket created after a partial close (new ticket, same magic).
        if(stage >= 1 && UseBreakEven && sl0 > 0 && (sl == 0 || MathAbs(sl - sl0) < point))
        {
            double spread = MarketInfo(g_symbol, MODE_SPREAD) * point;
            double beSL = isBuy ? NormalizeDouble(openPx + spread, digits)
                                : NormalizeDouble(openPx - spread, digits);
            bool beSideOk = isBuy ? (beSL < Bid) : (beSL > Ask);
            if(beSideOk)
            {
                if(OrderModify(ticket, openPx, beSL, tp, 0, clrYellow))
                    Print("BREAK-EVEN: ticket=", ticket, " SL=", DoubleToString(beSL, digits));
            }
        }

        // ── STAGE 1 -> 2: TP2 hit — partial close + arm trailing ──
        if(UsePartialClose && stage == 1 && tp2 > 0)
        {
            bool tp2Hit = isBuy ? (Bid >= tp2) : (Ask <= tp2);
            if(tp2Hit)
            {
                if(PAT_DoPartial(ticket, magic, isBuy, origLot, TP2ClosePct, "tp2"))
                {
                    PAT_SaveStage(magic, 2);
                    stage = 2;
                }
            }
        }

        // ── STAGE >= 2: ATR trail the remainder (monotonic) ──
        if(stage >= 2 && UseTrailingStop)
            PAT_TrailRemainder(ticket, isBuy, openPx, tp, digits, point);
    }
}

//+------------------------------------------------------------------+
//| Partial close ~fraction% of ORIGINAL lot, respecting step/min.    |
//| Returns true if a partial close executed (remainder continues).   |
//| If broker minimums prevent a sane split, closes FULL remainder.   |
//+------------------------------------------------------------------+
bool PAT_DoPartial(int ticket, int magic, bool isBuy, double origLot,
                   double pct, string reason)
{
    if(!OrderSelect(ticket, SELECT_BY_TICKET)) return false;
    if(OrderCloseTime() != 0) return false;

    double curLots = OrderLots();
    double openPx  = OrderOpenPrice();
    double minLot  = MarketInfo(g_symbol, MODE_MINLOT);
    double step    = MarketInfo(g_symbol, MODE_LOTSTEP);
    double closeLots = origLot * (pct / 100.0);
    if(step > 0) closeLots = MathFloor(closeLots / step + 0.0000001) * step;
    closeLots = NormalizeDouble(closeLots, 2);

    bool closeFull = false;
    if(closeLots < minLot) closeFull = true;                       // slice below broker min
    else if(curLots - closeLots < minLot - 0.0000001) closeFull = true; // remainder below min

    RefreshRates();
    double px = isBuy ? Bid : Ask;

    if(closeFull)
    {
        Print("PARTIAL(", reason, "): remainder ", DoubleToString(curLots, 2),
              " too small to split — closing FULL at TP level");
        ClosePosition(ticket, reason);
        return false;
    }

    bool ok = OrderClose(ticket, closeLots, px, PAT_GetMaxSlippage(""), clrOrange);
    if(ok)
    {
        Print("PARTIAL ", reason, ": requested ", DoubleToString(closeLots, 2),
              " of ", DoubleToString(origLot, 2), " @ ~", DoubleToString(px, _Digits));
        // NOTE: MT4 splits the order — the closed chunk AND the remainder each
        // get NEW tickets. We therefore do NOT report here; PAT_HistoryPoll()
        // picks up the closed chunk by price-proximity (tp1/tp2) exactly once.
        // Stage state survives because it is keyed by magic.
    }
    else
        Print("PARTIAL ", reason, " FAILED: ticket=", ticket, " err=", GetLastError());
    return ok;
}

//+------------------------------------------------------------------+
//| Monotonic ATR trailing for the stage>=2 remainder                 |
//+------------------------------------------------------------------+
void PAT_TrailRemainder(int ticket, bool isBuy, double openPx, double tp,
                        int digits, double point)
{
    if(!OrderSelect(ticket, SELECT_BY_TICKET)) return;
    if(OrderCloseTime() != 0) return;

    double atr = GetATR(14);
    if(atr <= 0) return;
    double trailDist = atr * TrailingATRMult;
    double stopLevel = MarketInfo(g_symbol, MODE_STOPLEVEL) * point;
    double freezeLevel = MarketInfo(g_symbol, MODE_FREEZELEVEL) * point;
    double sl = OrderStopLoss();

    RefreshRates();
    if(isBuy)
    {
        double newSL = NormalizeDouble(Bid - trailDist, digits);
        double minSL = NormalizeDouble(Bid - stopLevel, digits);
        if(newSL > minSL) newSL = minSL;
        if(freezeLevel > 0 && MathAbs(Bid - sl) < freezeLevel) return;
        if(newSL > sl && newSL > openPx)
        {
            if(OrderModify(ticket, OrderOpenPrice(), newSL, tp, 0, clrAqua))
                Print("TRAIL BUY: ticket=", ticket, " SL=", sl, " -> ", newSL);
        }
    }
    else
    {
        double newSL = NormalizeDouble(Ask + trailDist, digits);
        double maxSL = NormalizeDouble(Ask + stopLevel, digits);
        if(newSL < maxSL) newSL = maxSL;
        if(freezeLevel > 0 && MathAbs(Ask - sl) < freezeLevel) return;
        if((sl == 0 || newSL < sl) && newSL < openPx)
        {
            if(OrderModify(ticket, OrderOpenPrice(), newSL, tp, 0, clrAqua))
                Print("TRAIL SELL: ticket=", ticket, " SL=", sl, " -> ", newSL);
        }
    }
}

//+------------------------------------------------------------------+
//| Close a position by ticket number                                 |
//+------------------------------------------------------------------+
bool ClosePosition(int ticket, string reason)
{
    if(!OrderSelect(ticket, SELECT_BY_TICKET, MODE_TRADES))
        return false;

    int type = OrderType();
    double closePrice = 0;
    if(type == OP_BUY)
        closePrice = Bid;
    else if(type == OP_SELL)
        closePrice = Ask;
    else
        return false;

    RefreshRates();
    if(type == OP_BUY) closePrice = Bid; else closePrice = Ask;

    bool ok = OrderClose(ticket, OrderLots(), closePrice, 5, clrOrange);
    if(ok)
    {
        double netProfit = OrderProfit() + OrderSwap() + OrderCommission();
        Print("CLOSED: ticket=", ticket, " reason=", reason, " | NET=", netProfit);
    }
    else
    {
        Print("CLOSE FAILED: ticket=", ticket, " error=", GetLastError());
    }
    return ok;
}

//+------------------------------------------------------------------+
//| Check if current time is near the swap/rollover cutoff             |
//+------------------------------------------------------------------+
bool IsNearSwapTime()
{
    int hour = TimeHour(TimeCurrent());
    int minute = TimeMinute(TimeCurrent());
    int dayOfWeek = TimeDayOfWeek(TimeCurrent());

    int cutoffMinute = SwapCutoffHour * 60 - SwapCutoffBuffer;
    int nowMinute = hour * 60 + minute;

    if(nowMinute >= cutoffMinute && nowMinute < SwapCutoffHour * 60 + 30)
    {
        if(dayOfWeek >= 1 && dayOfWeek <= 5)
            return true;
    }
    return false;
}

bool IsTripleSwapDay()
{
    if(!AvoidTripleSwapDay)
        return false;
    int dayOfWeek = TimeDayOfWeek(TimeCurrent());
    if(TripleSwapDay == "Wednesday" && dayOfWeek == 3)
        return true;
    if(TripleSwapDay == "Thursday" && dayOfWeek == 4)
        return true;
    if(TripleSwapDay == "Friday" && dayOfWeek == 5)
        return true;
    return false;
}

double GetATR(int period)
{
    return iATR(g_symbol, 0, period, 0);
}

//+------------------------------------------------------------------+
//| SLIPPAGE MONITORING                                               |
//+------------------------------------------------------------------+
void CheckSlippage(int ticket, string direction, double requestedPrice)
{
    if(ticket <= 0) return;
    if(!OrderSelect(ticket, SELECT_BY_TICKET, MODE_TRADES)) return;

    double filledPrice = OrderOpenPrice();
    double slippage = MathAbs(filledPrice - requestedPrice);
    double point = MarketInfo(g_symbol, MODE_POINT);
    double slippagePoints = 0;
    if(point > 0) slippagePoints = slippage / point;

    string slipMsg = "SLIPPAGE_EVENT|{";
    slipMsg += "\"ticket\":" + IntegerToString(ticket);
    slipMsg += ",\"symbol\":\"" + g_symbol + "\"";
    slipMsg += ",\"direction\":\"" + direction + "\"";
    slipMsg += ",\"requested\":" + DoubleToString(requestedPrice, 5);
    slipMsg += ",\"filled\":" + DoubleToString(filledPrice, 5);
    slipMsg += ",\"slippage_points\":" + DoubleToString(slippagePoints, 2);
    slipMsg += ",\"spread\":" + DoubleToString(MarketInfo(g_symbol, MODE_SPREAD) * point, 5);
    slipMsg += ",\"is_rollover\":" + (IsNearSwapTime() ? "true" : "false");
    slipMsg += ",\"strategy\":\"" + g_signalStrategy + "\"";
    slipMsg += ",\"signal_id\":\"" + g_signalID + "\"";
    slipMsg += "}";
    PAT_Send(slipMsg);

    if(RejectOnHighSlippage && MaxSlippagePoints > 0 && slippagePoints > MaxSlippagePoints)
    {
        g_slippageRejects++;
        Print("SLIPPAGE EXCEEDED: ticket=", ticket, " requested=", requestedPrice,
              " filled=", filledPrice, " slip=", slippagePoints, " points (max=", MaxSlippagePoints, ")");
        PAT_SetForcedReason(ticket, "SLIPPAGE_REJECT");
        ClosePosition(ticket, "SLIPPAGE_REJECT");
    }
    else
    {
        Print("Fill OK: ticket=", ticket, " requested=", requestedPrice, " filled=", filledPrice,
              " slippage=", slippagePoints, " points");
    }
}

//+------------------------------------------------------------------+
//| Log each PAT/XAUUSD closed order counted as "today" so the trader |
//| can verify whether a prior-day close is leaking into the daily    |
//| loss calc.                                                        |
//+------------------------------------------------------------------+
void PAT_LogDailyLossDeals()
{
   int today = TimeDay(TimeCurrent()) + TimeMonth(TimeCurrent()) * 100 + TimeYear(TimeCurrent()) * 10000;
   int n = 0;
   for(int i = OrdersHistoryTotal() - 1; i >= 0; i--)
   {
      if(!OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)) continue;
      if(OrderSymbol() != g_symbol) continue;
      if(!PAT_IsPatMagic(OrderMagicNumber())) continue;
      datetime closeTime = OrderCloseTime();
      if(closeTime == 0) continue; // still open
      int dy = TimeDay(closeTime) + TimeMonth(closeTime) * 100 + TimeYear(closeTime) * 10000;
      if(dy != today) continue;
      n++;
      PAT_LogLine("CAPITAL DEAL #" + IntegerToString(n)
                  + " | date(Broker): " + TimeToString(closeTime, TIME_DATE)
                  + " | profit: " + DoubleToString(OrderProfit(), 2)
                  + " | swap: " + DoubleToString(OrderSwap(), 2)
                  + " | commission: " + DoubleToString(OrderCommission(), 2));
   }
   if(n == 0) PAT_LogLine("CAPITAL DEAL: none counted as today");
}

//+------------------------------------------------------------------+
//| CAPITAL PROTECTION                                                |
//+------------------------------------------------------------------+
void UpdateCapitalProtection()
{
    datetime today = TimeDay(TimeCurrent()) + TimeMonth(TimeCurrent()) * 100 + TimeYear(TimeCurrent()) * 10000;

    if(g_currentDay != today)
    {
        g_currentDay = today;
        g_dayStartBalance = AccountBalance();
        g_dailyPnL = 0;
        g_tradingBlocked = false;
        g_hardHaltTriggered = false;
        Print("NEW TRADING DAY: start balance=", g_dayStartBalance);
    }

    g_dailyPnL = 0;
    int total = OrdersHistoryTotal();
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)) continue;
        if(OrderSymbol() != g_symbol || !PAT_IsPatMagic(OrderMagicNumber())) continue;
        if(OrderCloseTime() == 0) continue;

        datetime closeDay = TimeDay(OrderCloseTime()) + TimeMonth(OrderCloseTime()) * 100 + TimeYear(OrderCloseTime()) * 10000;
        if(closeDay == today)
            g_dailyPnL += OrderProfit() + OrderSwap() + OrderCommission();
    }

    // Daily loss % is measured against the balance at the start of the broker
    // day. Derive it from realized P&L so it is correct even if the EA is
    // attached/restarted mid-day (the captured baseline would otherwise be the
    // already-reduced balance and overstate the loss %).
    double curBal = AccountBalance();
    double dayOpenBalance = curBal - g_dailyPnL;
    if(dayOpenBalance <= 0) dayOpenBalance = curBal; // deposit/withdrawal guard
    g_dayStartBalance = dayOpenBalance;
    double lossPct = 0;
    if(dayOpenBalance > 0)
        lossPct = (g_dailyPnL / dayOpenBalance) * 100;

    double effSoftHalt = WarningLossPct;
    double effHardHalt = MaxDailyLossPct;
    double effWarning  = WarningLossPct;
    if(curBal < 100)
    {
        effSoftHalt = WarningLossPct * 3.5;
        effHardHalt = MaxDailyLossPct * 3.5;
        effWarning  = WarningLossPct * 3.5;
    }
    else if(curBal < 200)
    {
        effSoftHalt = WarningLossPct * 2.0;
        effHardHalt = MaxDailyLossPct * 2.0;
        effWarning  = WarningLossPct * 2.0;
    }

    // Client override: if BypassDailyLossBlock is enabled, never keep the soft
    // block active (immediate unblock when the operator toggles it on).
    if(BypassDailyLossBlock && g_tradingBlocked)
    {
        g_tradingBlocked = false;
        Print("CAPITAL PROTECTION: soft daily-loss block BYPASSED by client input — trading re-enabled");
    }

    // RECOVERY: if the daily loss is no longer beyond the soft halt, clear the
    // block so a recovered/healthy account is not stuck blocked for the day.
    // (Previously g_tradingBlocked was set true but never re-evaluated, so a
    // single early-in-the-day loss kept trading blocked even after recovery.)
    if(lossPct > -effSoftHalt)
    {
        if(g_tradingBlocked)
        {
            g_tradingBlocked = false;
            g_capitalWarnActive = false;  // allow a fresh warning next time we re-enter the band
            Print("CAPITAL PROTECTION (RECOVER): daily loss recovered to ", lossPct, "% — trading re-enabled");
            string recMsg = "CAPITAL_PROTECTION|{";
            recMsg += "\"event_type\":\"RECOVER\"";
            recMsg += ",\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
            recMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
            recMsg += ",\"action\":\"RESUMED\"";
            recMsg += "}";
            PAT_Send(recMsg);
            PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: RESUMED (not blocked)");
        }
    }
    else
    {
        // Edge-triggered CAPITAL_WARNING: emit only on the transition into the
        // warning band (not on every tick). Resets when the account recovers
        // above the warning threshold so a fresh warning can fire later.
        if(lossPct <= -effWarning && !g_tradingBlocked)
        {
            if(!g_capitalWarnActive)
            {
                g_capitalWarnActive = true;
                string warnMsg = "CAPITAL_WARNING|{";
                warnMsg += "\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
                warnMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
                warnMsg += ",\"max_loss_pct\":" + DoubleToString(MaxDailyLossPct, 1);
                warnMsg += ",\"balance\":" + DoubleToString(curBal, 2);
                warnMsg += ",\"action\":\"WARNED\"";
                warnMsg += "}";
                PAT_Send(warnMsg);
                Print("CAPITAL WARNING: daily P&L=", g_dailyPnL, " (", lossPct, "%)");
            }
        }
        else
        {
            g_capitalWarnActive = false;
        }
        if(!BypassDailyLossBlock && lossPct <= -effSoftHalt && !g_tradingBlocked)
        {
            g_tradingBlocked = true;
            Print("*** CAPITAL PROTECTION (SOFT): Daily loss ", lossPct, "% — new entries blocked ***");
            string softMsg = "CAPITAL_PROTECTION|{\"event_type\":\"SOFT_HALT\",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + ",\"action\":\"BLOCKED_NEW_ENTRIES_ONLY\"}";
            PAT_Send(softMsg);
            PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: BLOCKED (daily loss)");
            PAT_LogDailyLossDeals();
        }
    }
    if(lossPct <= -effHardHalt && !g_hardHaltTriggered)
    {
        g_hardHaltTriggered = true;
        g_tradingBlocked = true;
        Print("*** CAPITAL PROTECTION (HARD): Daily loss ", lossPct, "% — CLOSING ALL POSITIONS ***");
        string blockMsg = "CAPITAL_PROTECTION|{";
        blockMsg += "\"event_type\":\"DAILY_LOSS_LIMIT_HIT\"";
        blockMsg += ",\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
        blockMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
        blockMsg += ",\"balance\":" + DoubleToString(curBal, 2);
        blockMsg += ",\"action\":\"BLOCKED_NEW_TRADES\"";
        blockMsg += "}";
        PAT_Send(blockMsg);
        PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: HARD HALT (closing all)");
        PAT_LogDailyLossDeals();

        if(EmergencyCloseAll)
            CloseAllPatPositions("EMERGENCY_CAPITAL_PROTECTION");
    }
}

//+------------------------------------------------------------------+
void CheckAgentConnection()
{
    static uint lastCheck = 0;
    if(GetTickCount() - lastCheck < 2000) return;
    lastCheck = GetTickCount();

    // Option B liveness: "connected" = cloud device credentials ready.
    // Reachability errors surface from ingest/poll HTTP failures.
    if(PAT_EnsureDevice())
    {
        g_connection = "CONNECTED";
        g_netDiagnosticsShown = false;
    }
    else
    {
        if(g_connection == "CONNECTED")
        {
            Print("[Predict-A-Trade] Cloud device credentials lost");
            g_connection = "OFFLINE";
        }
    }
}

//+------------------------------------------------------------------+
void SendTickToAgent()
{
    g_lastTickSend = GetTickCount();

    double bid = MarketInfo(g_symbol, MODE_BID);
    double ask = MarketInfo(g_symbol, MODE_ASK);
    if(bid <= 0 || ask <= 0) return;

    g_tickCount++;

    string msg = "TICK|{\"type\":\"TICK\"";
    msg += ",\"symbol\":\"" + g_symbol + "\"";
    msg += ",\"bid\":" + DoubleToString(bid, 5);
    msg += ",\"ask\":" + DoubleToString(ask, 5);
    msg += ",\"volume\":" + IntegerToString((long)Volume[0]);
    msg += ",\"timestamp\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += ",\"source\":\"MT4\"";
    msg += ",\"broker\":\"" + AccountCompany() + "\"";
    msg += ",\"account\":\"" + g_accountID + "\"";
    msg += ",\"license_key\":\"" + LicenseKey + "\"";
    // Broker session timezone — collected live so the engine works on Broker TF
    // (not UTC). TimeGMTOffset() returns the broker's GMT offset in seconds.
    msg += ",\"broker_offset\":" + IntegerToString(TimeGMTOffset() / 3600);
    msg += "}\n";

    PAT_Send(msg);
}

//+------------------------------------------------------------------+
void SendInitMessage()
{
    int totalPos = OrdersTotal();
    int buyCount = 0, sellCount = 0;
    double totalLots = 0;
    for(int i = 0; i < totalPos; i++)
    {
        if(OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            if(OrderType() == OP_BUY) { buyCount++; totalLots += OrderLots(); }
            else if(OrderType() == OP_SELL) { sellCount++; totalLots += OrderLots(); }
        }
    }
    string msg = "INIT|{\"ea_version\":\"1.27\",\"broker\":\"" + AccountCompany() +
                 "\",\"account\":\"" + g_accountID + "\",\"symbol\":\"" + g_symbol +
                 "\",\"license_key\":\"" + LicenseKey +
                 "\",\"balance\":" + DoubleToString(AccountBalance(), 2) +
                 ",\"equity\":" + DoubleToString(AccountEquity(), 2) +
                 ",\"profit\":" + DoubleToString(AccountProfit(), 2) +
                 ",\"currency\":\"" + AccountCurrency() +
                 "\",\"leverage\":" + IntegerToString(AccountLeverage()) +
                 ",\"open_positions\":" + IntegerToString(totalPos) +
                 ",\"buy_positions\":" + IntegerToString(buyCount) +
                 ",\"sell_positions\":" + IntegerToString(sellCount) +
                 ",\"total_lots\":" + DoubleToString(totalLots, 2) +
                 ",\"free_margin\":" + DoubleToString(AccountFreeMargin(), 2) +
                 ",\"floating_pnl\":" + DoubleToString(AccountProfit(), 2) +
                 "}\n";
    PAT_Send(msg);
}

//+------------------------------------------------------------------+
//| Periodic account telemetry → engine (enables executable signals).   |
//+------------------------------------------------------------------+
void SendAccountInfo()
{
    string msg = "ACCOUNT_INFO|{\"ea_version\":\"1.27\",\"account\":\"" + g_accountID +
                 "\",\"broker\":\"" + AccountCompany() +
                 "\",\"symbol\":\"" + g_symbol +
                 "\",\"currency\":\"" + AccountCurrency() +
                 "\",\"license_key\":\"" + g_licenseKey +
                 ",\"balance\":" + DoubleToString(AccountBalance(), 2) +
                 ",\"equity\":" + DoubleToString(AccountEquity(), 2) +
                 ",\"free_margin\":" + DoubleToString(AccountFreeMargin(), 2) +
                 ",\"leverage\":" + IntegerToString(AccountLeverage()) +
                 ",\"open_positions\":" + IntegerToString(OrdersTotal()) +
                 "}\n";
    PAT_Send(msg);
}

//+------------------------------------------------------------------+
void RequestLicenseValidation()
{
    string msg = "LICENSE_CHECK|{\"account\":\"" + g_accountID +
                 "\",\"broker\":\"" + AccountCompany() +
                 "\",\"symbol\":\"" + g_symbol +
                 "\",\"license_key\":\"" + g_licenseKey +
                 "\",\"balance\":" + DoubleToString(AccountBalance(), 2) +
                 ",\"equity\":" + DoubleToString(AccountEquity(), 2) +
                 ",\"profit\":" + DoubleToString(AccountProfit(), 2) +
                 ",\"free_margin\":" + DoubleToString(AccountFreeMargin(), 2) +
                 ",\"open_positions\":" + IntegerToString(OrdersTotal()) +
                 "}\n";
    PAT_Send(msg);
    Print("License validation requested - balance: ", AccountBalance());
}

//+------------------------------------------------------------------+
//| LIVENESS ping — OnTimer fallback when the market produces no ticks |
//| (weekend/holiday): keeps terminal visible + license resolvable.    |
//+------------------------------------------------------------------+
void SendLivenessPing()
{
    string msg = "LIVENESS|{\"type\":\"LIVENESS\"";
    msg += ",\"symbol\":\""+g_symbol+"\"";
    msg += ",\"source\":\"MT4\"";
    msg += ",\"account\":\""+g_accountID+"\"";
    msg += ",\"broker\":\"" + AccountCompany() + "\"";
    msg += ",\"market_closed\":true";
    msg += ",\"timestamp\":\""+FormatISO8601UTC(TimeGMT())+"\"}\n";
    PAT_Send(msg);

    RequestLicenseValidation();
}

// ReadFromAgent removed (v1.19.0 Option B): inbound traffic now arrives via
void ReadFromAgent() { PollFromCloud(); }


//+------------------------------------------------------------------+
//+------------------------------------------------------------------+
//| Server-side SL enforcement: CLOSE_POSITION command               |
//+------------------------------------------------------------------+
void HandleClosePosition(string payload)
{
    int ticket = (int)StringToInteger(ExtractJSONString(payload, "ticket"));
    long magic = (long)StringToInteger(ExtractJSONString(payload, "magic"));
    string reason = ExtractJSONString(payload, "reason");

    Print("CLOSE_POSITION from server: ticket=", ticket, " magic=", magic, " reason=", reason);

    if(ticket > 0)
    {
        if(OrderSelect(ticket, SELECT_BY_TICKET))
        {
            // W6 FIX: only close if the position belongs to PAT (within our magic
            // range) AND matches this EA's symbol. Prevents closing an arbitrary
            // user position that happens to share the ticket number.
            if(!PAT_IsPatMagic(OrderMagicNumber()) || OrderSymbol() != g_symbol)
            {
                Print("CLOSE_POSITION: ticket=", ticket, " ignored — not a PAT position (magic=", OrderMagicNumber(), " symbol=", OrderSymbol(), ")");
                return;
            }
            int type = OrderType();
            bool isBuy = (type == OP_BUY);
            double closePrice = isBuy ? MarketInfo(g_symbol, MODE_BID) : MarketInfo(g_symbol, MODE_ASK);
            if(OrderClose(ticket, OrderLots(), closePrice, 10, clrRed))
            {
                Print("CLOSE_POSITION: ticket=", ticket, " closed");
                PAT_SetForcedReason(ticket, "SERVER_CLOSE_POSITION");
            }
            else
                Print("CLOSE_POSITION: FAILED ticket=", ticket, " err=", GetLastError());
            return;
        }
    }

    if(magic > 0)
    {
        int total = OrdersTotal();
        for(int i = total - 1; i >= 0; i--)
        {
            if(OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
            {
                if(OrderMagicNumber() == magic && OrderSymbol() == g_symbol)
                {
                    bool isBuy = (OrderType() == OP_BUY);
                    double closePrice = isBuy ? MarketInfo(g_symbol, MODE_BID) : MarketInfo(g_symbol, MODE_ASK);
                    PAT_SetForcedReason(OrderTicket(), "SERVER_CLOSE_POSITION");
                    if(!OrderClose(OrderTicket(), OrderLots(), closePrice, 10, clrRed))
                        Print("OrderClose failed during SERVER_CLOSE_POSITION: ticket=", OrderTicket(), " err=", GetLastError());
                }
            }
        }
    }
}

//+------------------------------------------------------------------+
//| Server-side emergency stop: close ALL PAT positions and halt     |
//+------------------------------------------------------------------+
void HandleEmergencyStop(string payload)
{
    string reason = ExtractJSONString(payload, "reason");
    Print("*** EMERGENCY_STOP from server: reason=", reason, " — CLOSING ALL ***");

    int total = OrdersTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        if(OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            if(PAT_IsPatMagic(OrderMagicNumber()) && OrderSymbol() == g_symbol)
            {
                bool isBuy = (OrderType() == OP_BUY);
                double closePrice = isBuy ? MarketInfo(g_symbol, MODE_BID) : MarketInfo(g_symbol, MODE_ASK);
                PAT_SetForcedReason(OrderTicket(), "SERVER_EMERGENCY_STOP");
                if(!OrderClose(OrderTicket(), OrderLots(), closePrice, 10, clrRed))
                    Print("OrderClose failed during SERVER_EMERGENCY_STOP: ticket=", OrderTicket(), " err=", GetLastError());
            }
        }
    }

    g_tradingStatus = "EMERGENCY_HALT";
    GlobalVariableSet("PAT_EQUITY_HALT", 1);
    g_equityHalted = true;
    Print("*** EMERGENCY_STOP complete — trading HALTED ***");
}

//+------------------------------------------------------------------+
//| Server-side kill switch: close ALL and stop EA                   |
//+------------------------------------------------------------------+
void HandleKillSwitch(string payload)
{
    Print("*** KILL_SWITCH from server — CLOSING ALL AND STOPPING EA ***");

    int total = OrdersTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        if(OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            if(PAT_IsPatMagic(OrderMagicNumber()) && OrderSymbol() == g_symbol)
            {
                bool isBuy = (OrderType() == OP_BUY);
                double closePrice = isBuy ? MarketInfo(g_symbol, MODE_BID) : MarketInfo(g_symbol, MODE_ASK);
                PAT_SetForcedReason(OrderTicket(), "SERVER_KILL_SWITCH");
                if(!OrderClose(OrderTicket(), OrderLots(), closePrice, 10, clrRed))
                    Print("OrderClose failed during SERVER_KILL_SWITCH: ticket=", OrderTicket(), " err=", GetLastError());
            }
        }
    }

    g_tradingStatus = "KILL_SWITCH";
    GlobalVariableSet("PAT_EQUITY_HALT", 1);
    g_equityHalted = true;

    string deinitMsg = "DEINIT|{\"reason\":\"SERVER_KILL_SWITCH\"}\n";
    PAT_Send(deinitMsg);
    Print("*** KILL_SWITCH complete — EA stopped ***");
    ExpertRemove();
}

//+------------------------------------------------------------------+
//| Build position details JSON for MARKET_SNAPSHOT (server SL monitor)|
//+------------------------------------------------------------------+
string PAT_BuildPositionDetails()
{
    string details = "[";
    bool first = true;
    int total = OrdersTotal();
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(!PAT_IsPatMagic(OrderMagicNumber())) continue;
        if(!first) details += ",";
        first = false;
        double sl = OrderStopLoss();
        double tp = OrderTakeProfit();
        double vol = OrderLots();
        double openPx = OrderOpenPrice();
        double profit = OrderProfit();
        bool isBuy = (OrderType() == OP_BUY);
        string typeStr = isBuy ? "BUY" : "SELL";
        details += "{\"ticket\":" + IntegerToString(OrderTicket());
        details += ",\"magic\":" + IntegerToString(OrderMagicNumber());
        details += ",\"type\":\"" + typeStr + "\"";
        details += ",\"volume\":\"" + DoubleToString(vol, 2) + "\"";
        details += ",\"open_price\":\"" + DoubleToString(openPx, Digits) + "\"";
        details += ",\"sl\":\"" + DoubleToString(sl, Digits) + "\"";
        details += ",\"tp\":\"" + DoubleToString(tp, Digits) + "\"";
        details += ",\"profit\":\"" + DoubleToString(profit, 2) + "\"";
        details += ",\"symbol\":\"" + g_symbol + "\"}";
    }
    details += "]";
    return details;
}

//+------------------------------------------------------------------+
bool IsStrategyEnabled(string strategyID)
{
    // SERVER-CONTROLLED: Check if strategy is in server-provided allowed_strategies.
    // The server ALSO filters signals before sending to the agent (primary defense).
    // This EA check is a secondary defense layer.

    // If the server OMITTED allowed_strategies entirely (legacy backend /
    // not yet received), allow all — server-side filtering is primary defense.
    if(!g_strategiesEnforced)
    {
        // PENDING = activation round-trip not finished yet. The signal still
        // came through the HMAC-authenticated per-device edge-poll and was
        // ALREADY filtered server-side by license + plan allowed_strategies
        // (fail-closed enqueue). Dropping it here only punished terminal
        // restarts ("license not validated — blocking STANDARD_SWING").
        // Block only explicit negative license states.
        if(g_licenseStatus == "REVOKED" || g_licenseStatus == "SUSPENDED" ||
           g_licenseStatus == "EXPIRED" || g_licenseStatus == "DENIED")
        {
            Print("Strategy check: license ", g_licenseStatus, " — blocking ", strategyID);
            return false;
        }
        return true;
    }

    // Server sent an explicit list. An EMPTY list means NO strategies allowed
    // (fail closed) — never fall through to allow-all.
    if(StringLen(g_allowedStrategies) == 0)
    {
        Print("Strategy check: empty allowed list (deny-all) — blocking ", strategyID);
        return false;
    }

    string search = "," + strategyID + ",";
    string list = "," + g_allowedStrategies + ",";
    if(StringFind(list, search) >= 0)
        return true;
    Print("Strategy check: ", strategyID, " NOT allowed (", g_allowedStrategies, ")");
    return false;
}

bool IsDirectionEnabled(string direction)
{
    if(direction == "BUY" && ReceiveBuy) return true;
    if(direction == "SELL" && ReceiveSell) return true;
    if(direction == "BUY_CANDIDATE" && ReceiveBuyCandidate) return true;
    if(direction == "SELL_CANDIDATE" && ReceiveSellCandidate) return true;
    return false;
}

//+------------------------------------------------------------------+
void HandleSignal(string json)
{
    g_lastSignalJSON   = json;
    g_signalID        = ExtractJSONString(json, "ID");
    g_signalDirection = ExtractJSONString(json, "Direction");
    g_signalGrade     = ExtractJSONString(json, "Grade");
    g_signalStrategy  = ExtractJSONString(json, "StrategyID");
    g_signalClass     = ExtractJSONString(json, "SignalClass");
    g_entry  = ExtractJSONDouble(json, "EntryPrice");
    g_sl     = ExtractJSONDouble(json, "StopLoss");
    g_tp1    = ExtractJSONDouble(json, "TP1");
    g_suggestedLot = ExtractJSONDouble(json, "SuggestedLot");
    g_tp2    = ExtractJSONDouble(json, "TP2");
    g_tp3    = ExtractJSONDouble(json, "TP3");
    g_rawScore = ExtractJSONDouble(json, "RawScore");
    g_calibProb = ExtractJSONDouble(json, "CalibratedProbability");
    g_signalTime = TimeCurrent();

    // Client MT terminal log — record every signal received from the engine.
    string logType = g_signalDirection;
    if(logType == "BUY_CANDIDATE") logType = "BUY";
    else if(logType == "SELL_CANDIDATE") logType = "SELL";
    string logPrice = (g_entry > 0) ? DoubleToString(g_entry, 2) : "—";
    string logLot = (g_suggestedLot > 0) ? DoubleToString(g_suggestedLot, 2) : "—";
    PAT_LogLine("SIGNAL RECEIVED | Symbol: " + g_symbol + " | Type: " + logType + " | Price: " + logPrice + " | Lot: " + logLot);

    if(g_signalID == g_lastExecutedSignalID)
        return;

    if(!IsStrategyEnabled(g_signalStrategy))
    {
        g_signalsFiltered++;
        return;
    }

    if(!IsDirectionEnabled(g_signalDirection))
    {
        g_signalsFiltered++;
        return;
    }

    if(g_connection != "CONNECTED")
        return;

    if(g_licenseStatus != "ACTIVE")
        return;

    if(AvoidSwapCharges && IsNearSwapTime())
    {
        Print("SWAP AVOIDANCE: skipping signal near swap cutoff — ", g_signalDirection, " ", g_signalStrategy);
        g_signalsFiltered++;
        return;
    }

    if(IsTripleSwapDay())
    {
        Print("TRIPLE SWAP DAY: skipping signal — ", g_signalDirection, " ", g_signalStrategy);
        g_signalsFiltered++;
        return;
    }

    if(g_tradingBlocked)
    {
        Print("CAPITAL PROTECTION: trading blocked — daily loss limit reached");
        g_signalsFiltered++;
        return;
    }

    g_signalsDisplayed++;

    // Only auto-trade CONFIRMED executable signals. ADVISORY / NO_TRADE signals
    // are displayed for context but must never open a position — this prevents
    // trading on non-confirmed reads and duplicate fills when both advisory and
    // executable signals are delivered to the same agent.
    if(g_signalClass != "EXECUTABLE")
    {
        Print("SIGNAL DISPLAY-ONLY: class=", g_signalClass, " — not EXECUTABLE, skip auto-trade");
        return;
    }

    // v1.25: TTL freshness gate — never execute a signal past its server
    // ExpiresAt (or MaxSignalAgeSeconds fallback). Restores the intended
    // defense-in-depth: the server already sweeps expired PENDING rows at
    // poll time, but this EA-side check protects against any future server
    // regression delivering stale payloads (fail-closed).
    if(!PAT_SignalFresh())
    {
        g_signalsFiltered++;
        return;
    }

    // v1.25: entry-drift gate — direction resolved first (candidates are
    // normalized below), then current market is compared against the signal
    // EntryPrice before any order is considered.
    bool driftBuy = (g_signalDirection == "BUY" || g_signalDirection == "BUY_CANDIDATE");
    bool driftSell = (g_signalDirection == "SELL" || g_signalDirection == "SELL_CANDIDATE");
    if(driftBuy && !PAT_EntryDriftOK(g_signalStrategy, true))  { g_signalsFiltered++; return; }
    if(driftSell && !PAT_EntryDriftOK(g_signalStrategy, false)) { g_signalsFiltered++; return; }

    if(AutoExecute && g_signalDirection == "BUY")
        ExecuteBuy();
    else if(AutoExecute && g_signalDirection == "SELL")
        ExecuteSell();
    else if(AutoExecute && ExecuteCandidates && g_signalDirection == "BUY_CANDIDATE")
    {
        Print("Executing BUY_CANDIDATE as real trade (ExecuteCandidates=true)");
        g_signalDirection = "BUY";
        ExecuteBuy();
    }
    else if(AutoExecute && ExecuteCandidates && g_signalDirection == "SELL_CANDIDATE")
    {
        Print("Executing SELL_CANDIDATE as real trade (ExecuteCandidates=true)");
        g_signalDirection = "SELL";
        ExecuteSell();
    }
}

//+------------------------------------------------------------------+
void HandleLicenseResponse(string json)
{
    string oldStatus = g_licenseStatus;
    string oldPlan   = g_licensePlan;
    string oldAuth   = g_authStatus;
    string oldDevice = g_deviceStatus;
    string oldSess   = g_sessionStatus;
    string oldTrade  = g_tradingStatus;
    g_licenseStatus = ExtractJSONString(json, "status");
    g_licensePlan   = ExtractJSONString(json, "plan");
    g_authStatus    = ExtractJSONString(json, "auth");
    g_deviceStatus  = ExtractJSONString(json, "device");
    g_sessionStatus = ExtractJSONString(json, "session");
    g_tradingStatus = ExtractJSONString(json, "trading");

    // Parse allowed_strategies from server license response.
    // Backend sends a JSON ARRAY: ["STANDARD_SCALPING",...]. W2 FIX: the old
    // ExtractJSONString only matched a quoted STRING, so the array was never
    // parsed and all strategies stayed enabled. Use ExtractJSONArrayRaw, with
    // legacy quoted-string fallback.
    bool listPresent = (StringFind(json, "\"allowed_strategies\":") >= 0);
    if(listPresent)
    {
        g_strategiesEnforced = true;
        string strategiesRaw = ExtractJSONArrayRaw(json, "allowed_strategies");
        if(StringLen(strategiesRaw) == 0)
            strategiesRaw = ExtractJSONString(json, "allowed_strategies"); // legacy string
        string cleaned = strategiesRaw;
        StringReplace(cleaned, "[", "");
        StringReplace(cleaned, "]", "");
        StringReplace(cleaned, "\"", "");
        StringReplace(cleaned, " ", "");
        g_allowedStrategies = cleaned;
    }

    // Only log when the license state actually changes — the agent re-sends
    // license responses on every heartbeat, so an unconditional log would spam
    // the terminal every few seconds.
    bool licChanged = (oldStatus != g_licenseStatus) || (oldPlan != g_licensePlan)
                     || (oldAuth != g_authStatus) || (oldDevice != g_deviceStatus)
                     || (oldSess != g_sessionStatus) || (oldTrade != g_tradingStatus);
    if(licChanged)
    {
        Print("License status: ", oldStatus, " -> ", g_licenseStatus,
              " Plan:", g_licensePlan);

        // Client MT terminal log — record license/access status.
        string access = (g_licenseStatus == "ACTIVE") ? "Access Granted" : "Access Denied";
        PAT_LogLine("STATUS: " + access + " | License: " + g_licenseStatus + " | Subscription: " + g_licensePlan);
    }
}

//+------------------------------------------------------------------+
//| EXECUTION — BUY                                                   |
//| Fail-closed: validates SL/TP sides + full pre-trade gate first.   |
//| Opens ONE position with the TOTAL lot; TP set to TP3 (final).     |
//| TP1/TP2 are taken as EA-managed PARTIAL closes.                   |
//+------------------------------------------------------------------+
void ExecuteBuy()
{
    // 1. WRONG-SIDE SL/TP REJECT (highest priority — abort, never clamp)
    double finalTP = (g_tp3 > 0) ? g_tp3 : g_tp1;
    if(!PAT_ValidateLevels(true, g_entry, g_sl, finalTP)) return;

    // 2. Strategy magic + comment
    int magicBase = PAT_StrategyMagicBase(g_signalStrategy);
    if(magicBase == 0)
    {
        Print("REJECTED unknown_strategy: ", g_signalStrategy);
        return;
    }
    int magic = PAT_NextMagic(magicBase);
    string comment = PAT_StrategyPrefix(g_signalStrategy) + PAT_ShortSignalID(g_signalID);

    // W4 FIX: BUY must size identically to SELL — use PAT_CalcLotSize when
    // auto lot sizing is enabled, else fall back to the base lot.
    double vol = 0;
    if(UseAutoLotSizing)
        vol = PAT_CalcLotSize(AccountEquity(), MathAbs(g_entry - g_sl));
    if(vol <= 0) vol = PAT_NormalizeLot(BaseLot);
    if(vol <= 0)
    {
        Print("REJECTED lot_below_min: computed lot below broker minimum — refusing to force size");
        return;
    }

    // 4. EA-side risk gate (spread, drift, TTL, caps, risk$, martingale, margin)
    // Risk gates handled by SERVER — EA trusts server decision

    // Enforce EA-side position caps / halt flags. PAT_PreTradeGate was defined
    // but never invoked here — that omission allowed duplicate/over-positioning
    // (multiple positions per signal). Wire it in (fail-closed).
    if(!PAT_PreTradeGate(true, vol, g_signalStrategy))
    {
        Print("SIGNAL NOT EXECUTED: EA pre-trade gate rejected BUY (cap/halt)");
        g_signalsFiltered++;
        return;
    }

    Print("ExecuteBuy: vol=", DoubleToString(vol, 2), " entry=", DoubleToString(Ask, _Digits),
          " sl=", DoubleToString(g_sl, _Digits), " tp3=", DoubleToString(finalTP, _Digits),
          " magic=", magic, " comment=", comment);

    RefreshRates();
    int ticket = OrderSend(g_symbol, OP_BUY, vol, Ask, PAT_GetMaxSlippage(g_signalStrategy),
                           g_sl, finalTP, comment, magic, 0, clrGreen);
    if(ticket > 0)
    {
        g_lastExecutedSignalID = g_signalID;
        PAT_RegPut(magic, g_signalID, g_signalStrategy, Ask, g_sl, g_tp1, g_tp2, g_tp3, vol);
        PAT_SaveStage(magic, 0);
        GlobalVariableSet(PAT_GVName(magic, "OL"), vol);
        Print("BUY executed: ticket=", ticket, " magic=", magic, " vol=", DoubleToString(vol, 2),
              " SL=", DoubleToString(g_sl, _Digits), " TP3=", DoubleToString(finalTP, _Digits));
        string ack = "EXECUTION_ACK|{";
        ack += "\"signal_id\":\"" + g_signalID + "\"";
        ack += ",\"status\":\"FILLED\"";
        ack += ",\"strategy_id\":\"" + g_signalStrategy + "\"";
        ack += ",\"magic\":" + IntegerToString(magic);
        ack += ",\"ticket\":" + IntegerToString(ticket);
        ack += ",\"entry\":" + DoubleToString(Ask, _Digits);
        ack += ",\"sl\":" + DoubleToString(g_sl, _Digits);
        ack += ",\"tp\":" + DoubleToString(finalTP, _Digits);
        ack += "}\n";
        PAT_Send(ack);
        CheckSlippage(ticket, "BUY", Ask);
    }
    else
    {
        Print("BUY FAILED: error=", GetLastError(), " entry=", g_entry, " sl=", g_sl, " tp=", finalTP);
    }
}

void ExecuteSell()
{
    double finalTP = (g_tp3 > 0) ? g_tp3 : g_tp1;
    if(!PAT_ValidateLevels(false, g_entry, g_sl, finalTP)) return;

    int magicBase = PAT_StrategyMagicBase(g_signalStrategy);
    if(magicBase == 0)
    {
        Print("REJECTED unknown_strategy: ", g_signalStrategy);
        return;
    }
    int magic = PAT_NextMagic(magicBase);
    string comment = PAT_StrategyPrefix(g_signalStrategy) + PAT_ShortSignalID(g_signalID);

    double vol = 0;
    if(UseAutoLotSizing)
        vol = PAT_CalcLotSize(AccountEquity(), MathAbs(g_entry - g_sl));
    if(vol <= 0) vol = PAT_NormalizeLot(BaseLot);
    if(vol <= 0)
    {
        Print("REJECTED lot_below_min: computed lot below broker minimum — refusing to force size");
        return;
    }

    // Risk gates handled by SERVER — EA trusts server decision

    // Enforce EA-side position caps / halt flags (same fix as ExecuteBuy).
    if(!PAT_PreTradeGate(false, vol, g_signalStrategy))
    {
        Print("SIGNAL NOT EXECUTED: EA pre-trade gate rejected SELL (cap/halt)");
        g_signalsFiltered++;
        return;
    }

    Print("ExecuteSell: vol=", DoubleToString(vol, 2), " entry=", DoubleToString(Bid, _Digits),
          " sl=", DoubleToString(g_sl, _Digits), " tp3=", DoubleToString(finalTP, _Digits),
          " magic=", magic, " comment=", comment);

    RefreshRates();
    int ticket = OrderSend(g_symbol, OP_SELL, vol, Bid, PAT_GetMaxSlippage(g_signalStrategy),
                           g_sl, finalTP, comment, magic, 0, clrRed);
    if(ticket > 0)
    {
        g_lastExecutedSignalID = g_signalID;
        PAT_RegPut(magic, g_signalID, g_signalStrategy, Bid, g_sl, g_tp1, g_tp2, g_tp3, vol);
        PAT_SaveStage(magic, 0);
        GlobalVariableSet(PAT_GVName(magic, "OL"), vol);
        Print("SELL executed: ticket=", ticket, " magic=", magic, " vol=", DoubleToString(vol, 2),
              " SL=", DoubleToString(g_sl, _Digits), " TP3=", DoubleToString(finalTP, _Digits));
        string ack = "EXECUTION_ACK|{";
        ack += "\"signal_id\":\"" + g_signalID + "\"";
        ack += ",\"status\":\"FILLED\"";
        ack += ",\"strategy_id\":\"" + g_signalStrategy + "\"";
        ack += ",\"magic\":" + IntegerToString(magic);
        ack += ",\"ticket\":" + IntegerToString(ticket);
        ack += ",\"entry\":" + DoubleToString(Bid, _Digits);
        ack += ",\"sl\":" + DoubleToString(g_sl, _Digits);
        ack += ",\"tp\":" + DoubleToString(finalTP, _Digits);
        ack += "}\n";
        PAT_Send(ack);
        CheckSlippage(ticket, "SELL", Bid);
    }
    else
    {
        Print("SELL FAILED: error=", GetLastError(), " entry=", g_entry, " sl=", g_sl, " tp=", finalTP);
    }
}

//+------------------------------------------------------------------+
//| Magic allocation: strategy base + offset within its 100-range     |
//+------------------------------------------------------------------+
int PAT_NextMagic(int magicBase)
{
    string seqName = "PAT_MAGIC_SEQ";
    if(GlobalVariableCheck(seqName))
        g_magicSeq = (int)GlobalVariableGet(seqName);
    for(int attempt = 0; attempt < 100; attempt++)
    {
        int magic = magicBase + (g_magicSeq % 100);
        g_magicSeq++;
        GlobalVariableSet(seqName, g_magicSeq);
        if(!PAT_MagicInUse(magic) && PAT_RegFind(magic) < 0) return magic;
    }
    return magicBase; // fallback: all offsets busy (should not happen with caps of 2)
}

bool PAT_MagicInUse(int magic)
{
    int total = OrdersTotal();
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderMagicNumber() == magic && OrderSymbol() == g_symbol) return true;
    }
    return false;
}

string PAT_ShortSignalID(string signalID)
{
    // MT4 comments are limited (~31 chars); prefix is 7, keep up to 22 of id
    if(StringLen(signalID) > 22) return StringSubstr(signalID, 0, 22);
    return signalID;
}

//+------------------------------------------------------------------+
void UpdatePanel()
{
    string p = "=== Predict-A-Trade v1.00 ===\n";
    p += "Agent:    " + g_connection + "\n";
    p += "License:  " + g_licenseStatus + " (" + g_licensePlan + ")\n";
    p += "Lic.Key:  " + (g_licenseKey == "" ? "NOT SET" : StringSubstr(g_licenseKey, 0, 12) + "...") + "\n";
    p += "Account:  " + g_accountID + "\n";
    p += "Symbol:   " + g_symbol + "\n";
    p += "Mode:     " + (AutoExecute ? "AUTO EXECUTE" : "SIGNAL ONLY") + "\n";
    p += "Open Pos: " + IntegerToString(PAT_CountPatPositions()) + "\n";
    p += "-----------------------------\n";
    p += "Signals:  " + IntegerToString(g_signalsReceived) + " recv, " + IntegerToString(g_signalsDisplayed) + " shown, " + IntegerToString(g_signalsFiltered) + " filtered\n";
    p += "Strats:   ";
    if(StringFind("," + g_allowedStrategies + ",", ",STANDARD_SCALPING,") >= 0) p += "SS ";
    if(StringFind("," + g_allowedStrategies + ",", ",ULTRA_SCALPING,") >= 0) p += "US ";
    if(StringFind("," + g_allowedStrategies + ",", ",STANDARD_SWING,") >= 0) p += "SW ";
    if(StringFind("," + g_allowedStrategies + ",", ",TREND_SWING,") >= 0) p += "TW\n";
    p += "-----------------------------\n";
    p += "Signal:   " + g_signalDirection + "\n";
    if(g_signalDirection != "NONE" && g_signalDirection != "EXPIRED")
    {
        p += "Strategy: " + g_signalStrategy + "\n";
        p += "Grade:    " + g_signalGrade + "\n";
        p += "Class:    " + g_signalClass + "\n";
        p += "Score:    " + DoubleToString(g_rawScore, 1) + "\n";
        p += "Prob:     " + (g_calibProb > 0 ? DoubleToString(g_calibProb * 100, 1) + "%" : "Pending") + "\n";
        p += "Entry:    " + DoubleToString(g_entry, 2) + "\n";
        p += "SL:       " + DoubleToString(g_sl, 2) + "\n";
        p += "TP1:      " + DoubleToString(g_tp1, 2) + "\n";
        p += "TP2:      " + DoubleToString(g_tp2, 2) + "\n";
        p += "TP3:      " + DoubleToString(g_tp3, 2) + "\n";
    }
    p += "-----------------------------\n";
    p += "Ticks:    " + IntegerToString(g_tickCount) + "\n";
    p += "Time:     " + TimeToString(TimeCurrent(), TIME_DATE|TIME_SECONDS) + "\n";
    p += "Slip rejects: " + IntegerToString(g_slippageRejects) + "\n";
    p += "Daily P&L: " + DoubleToString(g_dailyPnL, 2) + "\n";
    if(BypassDailyLossBlock) p += "DailyLoss guard: BYPASSED (client override)\n";
    if(g_tradingBlocked) p += "*** TRADING BLOCKED (daily loss) ***\n";
    if(g_equityHalted)   p += "*** HALTED: EQUITY FLOOR ***\n";
    if(AvoidSwapCharges)
    {
        p += "Swap cutoff: " + IntegerToString(SwapCutoffHour) + ":00 (-" + IntegerToString(SwapCutoffBuffer) + "min)\n";
        if(IsNearSwapTime()) p += "*** SWAP CUTOFF ACTIVE ***\n";
        if(IsTripleSwapDay()) p += "*** TRIPLE SWAP DAY ***\n";
    }
    Comment(p);
}

//+------------------------------------------------------------------+
//| JSON helpers                                                      |
//+------------------------------------------------------------------+
string ExtractJSONString(string json, string key)
{
    string search = "\"" + key + "\":\"";
    int start = StringFind(json, search);
    if(start < 0) return "";
    start += StringLen(search);
    int end = StringFind(json, "\"", start);
    if(end < 0) return "";
    return StringSubstr(json, start, end - start);
}

// ExtractJSONArrayRaw returns the INNER content of a JSON array value for `key`.
// Example: for {"allowed_strategies":["A","B"]} with key="allowed_strategies",
// returns "A","B" (brackets/quotes/spaces stripped by the caller).
// Returns "" if the value is not a JSON array. W2 FIX.
string ExtractJSONArrayRaw(string json, string key)
{
    string sk = "\"" + key + "\":[";
    int s = StringFind(json, sk);
    if(s < 0) return "";
    s += StringLen(sk);
    int depth = 1;
    int n = StringLen(json);
    int i = s;
    while(i < n)
    {
        int c = StringGetCharacter(json, i);
        if(c == '[') depth++;
        else if(c == ']') { depth--; if(depth == 0) break; }
        i++;
    }
    if(i >= n) return "";
    return StringSubstr(json, s, i - s);
}

double ExtractJSONDouble(string json, string key)
{
    string sk = "\""+ key + "\":";
    int s = StringFind(json, sk);
    if(s < 0) return 0;
    s += StringLen(sk);
    // Skip leading spaces or quotes (decimal.Decimal serializes as "4519.03")
    while(s < StringLen(json))
    {
        int c = StringGetChar(json, s);
        if(c == 32 || c == 34) { s++; continue; }
        break;
    }
    string v = "";
    for(int i = s; i < StringLen(json); i++)
    {
        int c = StringGetChar(json, i);
        if(c == 44 || c == 125 || c == 32 || c == 34) break;
        v += CharToString((uchar)c);
    }
    return StringToDouble(v);
}

//+------------------------------------------------------------------+
//| ══════════ OPTION B TRANSPORT (v1.19.0) ═══════════════════════ |
//| EAs talk to the cloud DIRECTLY over HTTPS — the Windows Agent and |
//| its IPC pipe files (PAT_ticks/PAT_signals/PAT_license) are gone.  |
//| Outbound: POST /ingest/agent (Bearer device JWT) on the Go engine.|
//| Inbound: HMAC edge-poll on the control plane (signals, license    |
//| verdicts, server commands).                                       |
//+------------------------------------------------------------------+
string g_deviceId       = "";
string g_deviceSecret   = "";
string g_refreshToken   = "";
string g_accessToken    = "";
datetime g_tokenExpiry  = 0;
bool   g_netDiagnosticsShown = false;
int    g_pollOkCount    = 0;
int    g_pollErrCount   = 0;
long   g_hmacCounter    = 0;

#define PAT_DEVICE_FILE "PAT_device_mt4.txt" // LEGACY shared name (v1.26 reads it only for one-time migration)
// v1.26: per-terminal state file. The old fixed "PAT_device_mt4.txt" in
// FILE_COMMON is shared by EVERY MT4 terminal on the machine — two broker
// terminals overwrite each other's device credentials, each then presents
// the other's (rotated) refresh token, the server's reuse detection revokes
// the whole token family, and both terminals 401-loop. State is now keyed
// per terminal: broker + account + terminal path, set in OnInit.
string  g_deviceFile     = PAT_DEVICE_FILE; // per-terminal state file, set in OnInit (v1.26)
string  g_deviceFileSet  = "";     // which file the in-memory creds were loaded from

//--- PAT_SHA256: pure-MQL4 SHA-256 (FIPS 180-4) over UTF-8 bytes
int PAT_ROTR(int x, int n) { return (int)(((uint)x >> n) | ((uint)x << (32 - n))); }

void PAT_StoreU32BE(uint v, uchar &outp[], int pos)
{
    outp[pos]   = (uchar)((v >> 24) & 0xFF);
    outp[pos+1] = (uchar)((v >> 16) & 0xFF);
    outp[pos+2] = (uchar)((v >> 8) & 0xFF);
    outp[pos+3] = (uchar)(v & 0xFF);
}

void PAT_SHA256K(uint &k[])
{
    static uint K[64];
    K[0]=0x428a2f98;  K[1]=0x71374491;  K[2]=0xb5c0fbcf;  K[3]=0xe9b5dba5;
    K[4]=0x3956c25b;  K[5]=0x59f111f1;  K[6]=0x923f82a4;  K[7]=0xab1c5ed5;
    K[8]=0xd807aa98;  K[9]=0x12835b01;  K[10]=0x243185be; K[11]=0x550c7dc3;
    K[12]=0x72be5d74; K[13]=0x80deb1fe; K[14]=0x9bdc06a7; K[15]=0xc19bf174;
    K[16]=0xe49b69c1; K[17]=0xefbe4786; K[18]=0x0fc19dc6; K[19]=0x240ca1cc;
    K[20]=0x2de92c6f; K[21]=0x4a7484aa; K[22]=0x5cb0a9dc; K[23]=0x76f988da;
    K[24]=0x983e5152; K[25]=0xa831c66d; K[26]=0xb00327c8; K[27]=0xbf597fc7;
    K[28]=0xc6e00bf3; K[29]=0xd5a79147; K[30]=0x06ca6351; K[31]=0x14292967;
    K[32]=0x27b70a85; K[33]=0x2e1b2138; K[34]=0x4d2c6dfc; K[35]=0x53380d13;
    K[36]=0x650a7354; K[37]=0x766a0abb; K[38]=0x81c2c92e; K[39]=0x92722c85;
    K[40]=0xa2bfe8a1; K[41]=0xa81a664b; K[42]=0xc24b8b70; K[43]=0xc76c51a3;
    K[44]=0xd192e819; K[45]=0xd6990624; K[46]=0xf40e3585; K[47]=0x106aa070;
    K[48]=0x19a4c116; K[49]=0x1e376c08; K[50]=0x2748774c; K[51]=0x34b0bcb5;
    K[52]=0x391c0cb3; K[53]=0x4ed8aa4a; K[54]=0x5b9cca4f; K[55]=0x682e6ff3;
    K[56]=0x748f82ee; K[57]=0x78a5636f; K[58]=0x84c87814; K[59]=0x8cc70208;
    K[60]=0x90befffa; K[61]=0xa4506ceb; K[62]=0xbef9a3f7; K[63]=0xc67178f2;
    ArrayCopy(k, K);
}

void PAT_SHA256(const uchar &msg[], uchar &digest[])
{
    ulong bitLen = (ulong)ArraySize(msg) * 8;
    // FIPS 180-4 §5.1: 0x80 + 8-byte length fit inside the 64-alignment of
    // (len + 9); the old ((len+8)/64+1)*64 form mis-pads len=55/119/...
    int paddedLen = (int)(((ArraySize(msg) + 9 + 63) / 64)) * 64;
    uchar padded[];
    ArrayResize(padded, paddedLen);
    ArrayInitialize(padded, 0);
    ArrayCopy(padded, msg, 0, 0, ArraySize(msg));
    padded[ArraySize(msg)] = 0x80;
    for(int i = 0; i < 8; i++)
        padded[paddedLen - 1 - i] = (uchar)((bitLen >> (8 * i)) & 0xFF);

    uint h0=0x6a09e667, h1=0xbb67ae85, h2=0x3c6ef372, h3=0xa54ff53a;
    uint h4=0x510e527f, h5=0x9b05688c, h6=0x1f83d9ab, h7=0x5be0cd19;
    uint k[64];
    PAT_SHA256K(k);

    uint w[64];
    ArrayInitialize(w, 0); // silence 'possible use of uninitialized variable' (filled per 64-byte block below)
    for(int off = 0; off < paddedLen; off += 64)
    {
        for(int t = 0; t < 16; t++)
            w[t] = ((uint)padded[off + t*4] << 24) | ((uint)padded[off + t*4 + 1] << 16) |
                   ((uint)padded[off + t*4 + 2] << 8) | (uint)padded[off + t*4 + 3];
        for(int t = 16; t < 64; t++)
        {
            uint s0 = PAT_ROTR(w[t-15],7) ^ PAT_ROTR(w[t-15],18) ^ (w[t-15] >> 3);
            uint s1 = PAT_ROTR(w[t-2],17) ^ PAT_ROTR(w[t-2],19) ^ (w[t-2] >> 10);
            w[t] = w[t-16] + s0 + w[t-7] + s1;
        }
        uint a=h0, b=h1, c=h2, d=h3, e=h4, f=h5, g=h6, hh=h7;
        for(int t = 0; t < 64; t++)
        {
            uint S1 = PAT_ROTR(e,6) ^ PAT_ROTR(e,11) ^ PAT_ROTR(e,25);
            uint ch = (e & f) ^ ((~e) & g);
            uint temp1 = hh + S1 + ch + k[t] + w[t];
            uint S0 = PAT_ROTR(a,2) ^ PAT_ROTR(a,13) ^ PAT_ROTR(a,22);
            uint maj = (a & b) ^ (a & c) ^ (b & c);
            uint temp2 = S0 + maj;
            hh=g; g=f; f=e; e=d+temp1; d=c; c=b; b=a; a=temp1+temp2;
        }
        h0+=a; h1+=b; h2+=c; h3+=d; h4+=e; h5+=f; h6+=g; h7+=hh;
    }

    ArrayResize(digest, 32);
    PAT_StoreU32BE(h0, digest, 0);  PAT_StoreU32BE(h1, digest, 4);
    PAT_StoreU32BE(h2, digest, 8);  PAT_StoreU32BE(h3, digest, 12);
    PAT_StoreU32BE(h4, digest, 16); PAT_StoreU32BE(h5, digest, 20);
    PAT_StoreU32BE(h6, digest, 24); PAT_StoreU32BE(h7, digest, 28);
}

//--- PAT_SHA256Hex / PAT_HmacSha256Hex (HMAC = SHA256(opad || SHA256(ipad||msg)))
string PAT_SHA256Hex(string text)
{
    uchar msg[];
    StringToCharArray(text, msg, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(msg, ArraySize(msg) - 1);
    uchar digest[];
    PAT_SHA256(msg, digest);
    string hexchars = "0123456789abcdef";
    string outp = "";
    for(int i = 0; i < ArraySize(digest); i++)
    {
        outp += StringSubstr(hexchars, (digest[i] >> 4) & 0x0F, 1);
        outp += StringSubstr(hexchars, digest[i] & 0x0F, 1);
    }
    return outp;
}

string PAT_HmacSha256Hex(string key, string message)
{
    uchar keyBytes[];
    int klen = StringToCharArray(key, keyBytes, 0, WHOLE_ARRAY, CP_UTF8) - 1;
    uchar keyBlock[64];
    ArrayInitialize(keyBlock, 0);
    ArrayCopy(keyBlock, keyBytes, 0, 0, MathMin(klen, 64));
    uchar ipad[64], opad[64];
    for(int i = 0; i < 64; i++)
    {
        ipad[i] = keyBlock[i] ^ 0x36;
        opad[i] = keyBlock[i] ^ 0x5C;
    }
    uchar msgBytes[];
    StringToCharArray(message, msgBytes, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(msgBytes, ArraySize(msgBytes) - 1);
    uchar innerMsg[96];
    ArrayCopy(innerMsg, ipad, 0, 0, 64);
    ArrayCopy(innerMsg, msgBytes, 64, 0, ArraySize(msgBytes));
    uchar innerDigest[];
    PAT_SHA256(innerMsg, innerDigest);
    uchar outerMsg[96];
    ArrayCopy(outerMsg, opad, 0, 0, 64);
    ArrayCopy(outerMsg, innerDigest, 64, 0, 32);
    uchar digest[];
    PAT_SHA256(outerMsg, digest);
    string hexchars = "0123456789abcdef";
    string outp = "";
    for(int i = 0; i < ArraySize(digest); i++)
    {
        outp += StringSubstr(hexchars, (digest[i] >> 4) & 0x0F, 1);
        outp += StringSubstr(hexchars, digest[i] & 0x0F, 1);
    }
    return outp;
}

//--- PAT_HTTPPost: plain JSON POST (no auth) → (status, response)
int PAT_HTTPPost(string url, string body, string &response)
{
    string headers = "Content-Type: application/json\r\n";
    uchar data[];
    StringToCharArray(body, data, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(data, ArraySize(data) - 1); // strip trailing NUL
    uchar result[];
    string resHeaders = "";
    int status = WebRequest("POST", url, headers, 8000, data, result, resHeaders);
    response = CharArrayToString(result, 0, WHOLE_ARRAY, CP_UTF8);
    return status;
}

//--- PAT_HMACSign: canonical v1 device signature over an outgoing request
//    canonical = "v1\n<ts>\n<nonce>\nPOST\n<path>\n<sha256(body)>\n<device_id>"
//    (byte-identical to DeviceAuthService.verifyRequestSignature)
string PAT_HMACSign(string path, string body, string deviceId, string deviceSecret, string ts, string nonce)
{
    string bodyHash = PAT_SHA256Hex(body);
    string canonical = "v1\n" + ts + "\n" + nonce + "\nPOST\n" + path + "\n" + bodyHash + "\n" + deviceId;
    return PAT_HmacSha256Hex(deviceSecret, canonical);
}

//--- PAT_SignedPost: HMAC-authenticated control-plane POST
int PAT_SignedPost(string path, string body, string &response)
{
    string ts = IntegerToString((long)TimeGMT() * 1000 + (GetTickCount() % 1000));
    g_hmacCounter++;
    string nonce = PAT_SHA256Hex(ts + IntegerToString(g_hmacCounter) + IntegerToString(MathRand()) + IntegerToString(GetTickCount()));
    string sig = PAT_HMACSign(path, body, g_deviceId, g_deviceSecret, ts, nonce);

    string headers = "Content-Type: application/json\r\n"
                     "X-Device-Id: " + g_deviceId + "\r\n"
                     "X-Device-Timestamp: " + ts + "\r\n"
                     "X-Device-Nonce: " + nonce + "\r\n"
                     "X-Device-Signature: " + sig + "\r\n";
    uchar data[];
    StringToCharArray(body, data, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(data, ArraySize(data) - 1);
    uchar result[];
    string resHeaders = "";
    int status = WebRequest("POST", PATCloudURL + path, headers, 8000, data, result, resHeaders);
    response = CharArrayToString(result, 0, WHOLE_ARRAY, CP_UTF8);
    return status;
}

//--- PAT_ReadFile / PAT_WriteFile / PAT_ClearFile: FILE_COMMON state (bootstrap + log)
string PAT_ReadFile(string filename)
{
    if(!FileIsExist(filename, FILE_COMMON)) return "";
    int h = FileOpen(filename, FILE_READ|FILE_TXT|FILE_ANSI|FILE_COMMON);
    if(h == INVALID_HANDLE) return "";
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

void PAT_WriteFile(string filename, string content)
{
    int h = FileOpen(filename, FILE_WRITE|FILE_TXT|FILE_ANSI|FILE_COMMON);
    if(h == INVALID_HANDLE) return;
    FileWriteString(h, content);
    FileClose(h);
}

//--- PAT_DeviceFingerprint: stable per-terminal identity
string PAT_DeviceFingerprint()
{
    string raw = "MT4|" + AccountCompany()
               + "|" + TerminalPath()
               + "|" + IntegerToString((int)TerminalInfoInteger(TERMINAL_BUILD));
    return PAT_SHA256Hex(raw);
}

//--- PAT_EnsureDevice: bootstrap credentials (inputs → else persisted → activate)
bool PAT_EnsureDevice()
{
    if(StringLen(g_deviceId) > 0 && StringLen(g_deviceSecret) > 0) return true;

    // 2) Persisted bootstrap state (per-terminal file)
    string saved = PAT_ReadFile(g_deviceFile);
    if(StringLen(saved) == 0 && StringLen(g_deviceFileSet) == 0)
    {
        // v1.26 one-time migration: adopt legacy shared PAT_device_mt4.txt
        // state into this terminal's own file (see PAT_ComputeDeviceFile).
        string legacy = PAT_ReadFile(PAT_DEVICE_FILE);
        if(StringLen(legacy) > 0)
        {
            string lparts[4];
            int ln = 0;
            string lrest = legacy;
            while(true)
            {
                int lp = StringFind(lrest, "|");
                if(lp < 0) { lparts[ln] = lrest; ln++; break; }
                lparts[ln] = StringSubstr(lrest, 0, lp);
                ln++;
                lrest = StringSubstr(lrest, lp + 1);
                if(ln >= 4) break;
            }
            if(ln >= 2 && StringLen(lparts[0]) > 0 && StringLen(lparts[1]) > 0)
            {
                g_deviceId = lparts[0];
                g_deviceSecret = lparts[1];
                if(ln >= 3) g_refreshToken = lparts[2];
                g_deviceFileSet = PAT_DEVICE_FILE;
                Print("[Predict-A-Trade] Adopted legacy device state (v1.25 migration): ", g_deviceId);
                return true;
            }
        }
    }
    if(StringLen(saved) > 0)
    {
        string parts[4];
        int n = 0;
        // MQL4 has no StringSplit — manual split on '|'
        string rest = saved;
        while(true)
        {
            int p = StringFind(rest, "|");
            if(p < 0) { parts[n] = rest; n++; break; }
            parts[n] = StringSubstr(rest, 0, p);
            n++;
            rest = StringSubstr(rest, p + 1);
            if(n >= 4) break;
        }
        if(n >= 2 && StringLen(parts[0]) > 0 && StringLen(parts[1]) > 0)
        {
            g_deviceId = parts[0];
            g_deviceSecret = parts[1];
            if(n >= 3) g_refreshToken = parts[2];
            g_deviceFileSet = g_deviceFile;
            return true;
        }
    }

    // 3) Auto-activate against the license key
    if(StringLen(LicenseKey) == 0)
    {
        if(!g_netDiagnosticsShown)
            Print("[Predict-A-Trade] No device credentials and no LicenseKey — set LicenseKey in EA inputs.");
        return false;
    }
    string fp = PAT_DeviceFingerprint();
    string body = "{\"license_key\":\"" + LicenseKey + "\",\"client_type\":\"MT4\",\"role\":\"exec\","
                  "\"fingerprint\":{\"machine_guid\":\"" + fp + "\",\"os\":\"Windows-MT4\"},"
                  "\"terminal\":{\"name\":\"" + PAT_JSONEscape(AccountCompany()) + "\"}}";
    string response = "";
    int status = PAT_HTTPPost(PATCloudURL + "/api/v1/devices/activate", body, response);
    if(status != 200)
    {
        Print("[Predict-A-Trade] Device activation failed: HTTP ", status, " — ", StringSubstr(response, 0, 200));
        return false;
    }
    g_deviceId     = ExtractJSONString(response, "device_id");
    g_deviceSecret = ExtractJSONString(response, "device_secret");
    g_refreshToken = ExtractJSONString(response, "refresh_token");
    g_accessToken  = ExtractJSONString(response, "access_token");
    if(StringLen(g_deviceId) == 0 || StringLen(g_deviceSecret) == 0)
    {
        Print("[Predict-A-Trade] Activation response missing device_id/device_secret.");
        return false;
    }
    PAT_WriteFile(g_deviceFile, g_deviceId + "|" + g_deviceSecret + "|" + g_refreshToken + "\n");
    Print("[Predict-A-Trade] Device activated: ", g_deviceId);
    return true;
}

//--- PAT_EnsureAccessToken: rotate the access token via refresh_token grant.
bool PAT_EnsureAccessToken()
{
    if(StringLen(g_accessToken) > 0 && TimeGMT() < g_tokenExpiry) return true;
    if(!PAT_EnsureDevice()) return false;
    if(StringLen(g_refreshToken) == 0) return false;

    // v1.27 multi-instance self-heal: two EA instances on the SAME terminal
    // (two charts) share one per-terminal state file. An instance holding an
    // older in-memory refresh token would present it and trip the server's
    // reuse detector (family revoked → re-activation churn). Re-read the
    // file before every refresh and adopt the newest persisted token when
    // it differs from memory — instances converge on one rotation chain.
    string state = PAT_ReadFile(g_deviceFile);
    if(StringLen(state) > 0)
    {
        string sparts[];
        int sn = StringSplit(state, '|', sparts);
        if(sn >= 3 && sparts[0] == g_deviceId && StringLen(sparts[2]) > 0 &&
           sparts[2] != g_refreshToken)
        {
            Print("[Predict-A-Trade] Adopting newer refresh token from device state file (another instance rotated).");
            g_refreshToken = sparts[2];
        }
    }

    string body = "{\"refresh_token\":\"" + g_refreshToken + "\"}";
    string response = "";
    int status = PAT_HTTPPost(PATCloudURL + "/api/v1/devices/refresh", body, response);
    if(status != 200)
    {
        if(!g_netDiagnosticsShown)
            Print("[Predict-A-Trade] Token refresh failed: HTTP ", status);
        return false;
    }
    g_accessToken  = ExtractJSONString(response, "access_token");
    string newRt   = ExtractJSONString(response, "refresh_token");
    long expiresIn = StrToInteger(ExtractJSONString(response, "expires_in"));
    if(StringLen(newRt) > 0) g_refreshToken = newRt;
    g_tokenExpiry = TimeGMT() + (expiresIn > 0 ? (datetime)(expiresIn - 60) : 82800);
    if(StringLen(g_accessToken) == 0) return false;
    PAT_WriteFile(g_deviceFile, g_deviceId + "|" + g_deviceSecret + "|" + g_refreshToken + "\n");
    return true;
}

//--- PAT_JSONEscape: minimal JSON string escaper for activation body
string PAT_JSONEscape(string s)
{
    string outp = "";
    for(int i = 0; i < StringLen(s); i++)
    {
        int c = StringGetChar(s, i);
        if(c == '"') outp += "\\\"";
        else if(c == '\\') outp += "\\\\";
        else outp += StringSubstr(s, i, 1);
    }
    return outp;
}

//+------------------------------------------------------------------+
//--- PAT_SanitizeFileTag: make broker/terminal strings safe for filenames
string PAT_SanitizeFileTag(string s)
{
    string outp = "";
    for(int i = 0; i < StringLen(s); i++)
    {
        int c = StringGetChar(s, i);
        if((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'))
            outp += CharToStr((uchar)c);
        else if(c == ' ' || c == '-' || c == '_' || c == '.')
            outp += "_";
        // everything else dropped
    }
    if(StringLen(outp) > 24) outp = StringSubstr(outp, 0, 24);
    return outp;
}

//--- PAT_ComputeDeviceFile: per-terminal state filename. Distinct per
//    broker+account+terminal-path, stable across restarts. This is what
//    stops two MT4 terminals on one machine from swapping refresh tokens.
string PAT_ComputeDeviceFile()
{
    string tag = PAT_SanitizeFileTag(AccountCompany()) + "_" +
                 g_accountID + "_" +
                 PAT_SanitizeFileTag(TerminalInfoString(TERMINAL_PATH));
    return "PAT_device_mt4_" + tag + ".txt";
}

//--- PAT_URLEncode — percent-encoding for query values
//+------------------------------------------------------------------+
string PAT_URLEncode(string s)
{
    string outp = "";
    for(int i = 0; i < StringLen(s); i++)
    {
        int c = StringGetChar(s, i);
        if((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
           c == '-' || c == '_' || c == '.' || c == '~')
            outp += CharToStr((uchar)c);
        else
            outp += StringFormat("%%%02X", c);
    }
    return outp;
}

//+------------------------------------------------------------------+
//| PAT_Send — outbound message funnel (Option B). Takes the legacy   |
//| "TYPE|{json}" wire line and POSTs its JSON payload to the engine  |
//| ingest endpoint. Injects "type" if the payload omits it.          |
//+------------------------------------------------------------------+
void PAT_Send(string line)
{
    string s = line;
    while(StringLen(s) > 0)
    {
        int c = StringGetChar(s, StringLen(s) - 1);
        if(c == '\n' || c == '\r' || c == 0) s = StringSubstr(s, 0, StringLen(s) - 1);
        else break;
    }
    int sep = StringFind(s, "|");
    if(sep <= 0) return;
    string msgType = StringSubstr(s, 0, sep);
    string payload = StringSubstr(s, sep + 1);
    if(StringLen(msgType) == 0 || StringLen(payload) == 0) return;
    if(StringFind(payload, "\"type\"") < 0 && StringGetChar(payload, 0) == '{')
        payload = "{\"type\":\"" + msgType + "\"," + StringSubstr(payload, 1);
    PAT_PostIngest(msgType, payload);
}

//--- PAT_PostIngest: POST one already-built payload to the Go engine
//    (Bearer device JWT) with one-shot 401 retry.
void PAT_PostIngest(string msgType, string payload)
{
    if(!PAT_EnsureDevice()) return;
    if(!PAT_EnsureAccessToken()) return;

    string headers = "Content-Type: application/json\r\n"
                     "Authorization: Bearer " + g_accessToken + "\r\n";
    uchar data[];
    StringToCharArray(payload, data, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(data, ArraySize(data) - 1);
    uchar result[];
    string resHeaders = "";
    string url = PATCloudURL + "/ingest/agent?agentId=" + PAT_URLEncode(g_deviceId) + "&role=exec";
    int status = WebRequest("POST", url, headers, 5000, data, result, resHeaders);
    if(status == 401)
    {
        g_accessToken = ""; g_tokenExpiry = 0;
        if(PAT_EnsureAccessToken())
            PAT_PostIngest(msgType, payload);
        return;
    }
    if(status != 200)
    {
        if(!g_netDiagnosticsShown)
        {
            Print("[Predict-A-Trade] ingest failed: HTTP ", status, " type=", msgType);
            g_netDiagnosticsShown = true;
        }
        return;
    }
    g_netDiagnosticsShown = false;
}

//+------------------------------------------------------------------+
//| PAT_EdgePoll — fetch the next batch of queued items for this      |
//| device. Returns number of items; fills items[]/queueIds[].        |
//+------------------------------------------------------------------+
int PAT_EdgePoll(string &items[], string &queueIds[])
{
    ArrayResize(items, 0);
    ArrayResize(queueIds, 0);
    if(!PAT_EnsureDevice()) return -1;

    string body = "{\"max_signals\":10}";
    string response = "";
    int status = PAT_SignedPost("/api/v1/devices/edge-poll", body, response);
    if(status != 200)
    {
        g_pollErrCount++;
        if(!g_netDiagnosticsShown)
        {
            Print("[Predict-A-Trade] edge-poll failed: HTTP ", status, " — ", StringSubstr(response, 0, 200));
            g_netDiagnosticsShown = true;
        }
        return -1;
    }
    g_netDiagnosticsShown = false;
    g_pollOkCount++;

    // Response: {"ok":true,"pending":[{"queue_id":"…","signal_id":"…","signal":{…}}, …]}
    int pos = 0;
    while(true)
    {
        int qid = StringFind(response, "\"queue_id\":\"", pos);
        if(qid < 0) break;
        int qidEnd = StringFind(response, "\"", qid + 12);
        if(qidEnd < 0) break;
        string queueId = StringSubstr(response, qid + 12, qidEnd - (qid + 12));

        int sigKey = StringFind(response, "\"signal\":{", qidEnd);
        int nextQ  = StringFind(response, "\"queue_id\"", qidEnd);
        if(sigKey < 0 || (nextQ >= 0 && sigKey > nextQ))
        {
            pos = qidEnd;
            continue;
        }
        // v1.21 FIX: sigKey+9 points AT the '{' ("signal":{ is 10 chars) —
        // the old sigKey+10 started inside the object, so MQL4's no-guard
        // extractor returned a TRUNCATED cut (missing SignalClass/Direction)
        // -> every signal was display-only + UNKNOWN-acked, never executed.
        string payload = PAT_ExtractJSONObject(response, sigKey + 9);
        int cnt = ArraySize(items);
        ArrayResize(items, cnt + 1);
        ArrayResize(queueIds, cnt + 1);
        items[cnt] = payload;
        queueIds[cnt] = queueId;
        pos = sigKey + 9 + StringLen(payload);
        if(ArraySize(items) >= 20) break;
    }
    return ArraySize(items);
}

//--- PAT_ExtractJSONObject: balanced-brace JSON object extractor (MQL4)
string PAT_ExtractJSONObject(string s, int start)
{
    if(start < 0 || StringGetChar(s, start) != '{') return "";  // v1.21: guard (parity with MT5)
    int depth = 0;
    bool inStr = false;
    int len = StringLen(s);
    for(int i = start; i < len; i++)
    {
        int c = StringGetChar(s, i);
        if(c == '\\' && inStr) { i++; continue; }
        if(c == '"') inStr = !inStr;
        if(inStr) continue;
        if(c == '{') depth++;
        else if(c == '}')
        {
            depth--;
            if(depth == 0) return StringSubstr(s, start, i - start + 1);
        }
    }
    return "";
}

//+------------------------------------------------------------------+
//| PAT_EdgeAck — confirm a queue item (executed / rejected / error). |
//+------------------------------------------------------------------+
void PAT_EdgeAck(string queueId, string resultJSON)
{
    if(StringLen(queueId) == 0) return;
    string body = "{\"queue_id\":\"" + queueId + "\",\"result\":" + resultJSON + "}";
    string response = "";
    int status = PAT_SignedPost("/api/v1/devices/edge-ack", body, response);
    if(status != 200)
        Print("[Predict-A-Trade] edge-ack failed: HTTP ", status, " — ", StringSubstr(response, 0, 120));
}

//+------------------------------------------------------------------+
//| PAT_EdgeHeartbeat — liveness + terminal/account metadata.         |
//+------------------------------------------------------------------+
void PAT_EdgeHeartbeat()
{
    if(!PAT_EnsureDevice()) return;
    // v1.24: stream equity with every heartbeat so the platform can classify
    // the device's capital tier (MICRO/STANDARD/PRO) and deliver signals
    // suitable for the account size. Equity is account currency.
    string body = "{\"terminal\":\"MT4\",\"account\":\"" + g_accountID + "\","
                  "\"symbol\":\"" + g_symbol + "\",\"build\":" + IntegerToString((int)TerminalInfoInteger(TERMINAL_BUILD)) + ","
                  "\"equity\":" + DoubleToString(AccountEquity(), 2) + "}";
    string response = "";
    int status = PAT_SignedPost("/api/v1/devices/edge-heartbeat", body, response);
    if(status != 200 && !g_netDiagnosticsShown)
        Print("[Predict-A-Trade] edge-heartbeat failed: HTTP ", status);
}

//+------------------------------------------------------------------+
//| PollFromCloud — Option B signal/command fetch (replaces           |
//| ReadFromAgent's file IPC). Pulls the edge queue via HMAC, then    |
//| dispatches SIGNAL / LICENSE_STATUS / CLOSE_POSITION /             |
//| EMERGENCY_STOP / KILL_SWITCH exactly like the old pipe reader.    |
//+------------------------------------------------------------------+
void PollFromCloud()
{
    // v1.23: cadence guard — polling on every tick hammered the cloud with
    // 100-290 req/min per terminal and tripped the shared-IP HTTP 429 throttle
    // for ALL clients behind the same IP. MT5 has had this guard since v1.10;
    // MT4 was the unguarded regression.
    static uint lastPoll = 0;
    uint nowMs = GetTickCount();
    if(nowMs - lastPoll < (uint)MathMax(PATPollMs, 1000)) return;
    lastPoll = nowMs;

    string items[];
    string queueIds[];
    int n = PAT_EdgePoll(items, queueIds);
    if(n <= 0) return;

    for(int i = 0; i < n; i++)
    {
        string payload = items[i];
        string msgType = ExtractJSONString(payload, "type");
        // v1.21b: the poller extracts the INNER signal object (post off-by-one fix).
        // Signal payloads carry "ID" (no "type", no "signal_id") — detect them the
        // same way the MT5 client does, else every real signal fell to UNKNOWN and
        // was acked-but-never-executed.
        // v1.27.1: promote an empty msgType BEFORE dispatch (parity with MT5
        // v1.26.1): type-less payload with "ID" = a real signal.
        if(StringLen(msgType) == 0)
        {
            if(StringFind(payload, "\"signal_id\"") >= 0 || ExtractJSONString(payload, "ID") != "")
                msgType = "SIGNAL";
            else
                msgType = "UNKNOWN";
        }

        if(msgType == "SIGNAL")
        {
            g_signalsReceived++;
            HandleSignal(payload);
        }
        else if(msgType == "LICENSE_STATUS" || msgType == "LICENSE_RESPONSE" || msgType == "LICENSE")
        {
            // Envelope: {"type":"LICENSE_STATUS","license_status":{...},"device_id":"…"}
            // The verdict is a nested JSON OBJECT (not a string) — extract it
            // specifically, falling back to the whole payload only if absent.
            int licKey = StringFind(payload, "\"license_status\":{");
            if(licKey >= 0)
            {
                int licStart = licKey + StringLen("\"license_status\":");
                string lic = PAT_ExtractJSONObject(payload, licStart);
                if(StringLen(lic) > 0)
                    HandleLicenseResponse(lic);
            }
            else
                HandleLicenseResponse(payload);
        }
        else if(msgType == "SERVER_COMMAND" || msgType == "CLOSE_POSITION")
        {
            // v1.27: server command envelopes (queueCommandForDevice wraps
            // CLOSE_POSITION / EMERGENCY_STOP / KILL_SWITCH / REQUEST_SNAPSHOT
            // as {"type":"SERVER_COMMAND","command":…}). Dispatch by the inner
            // command, mirroring the MT5 client; unknown commands log and ACK.
            string cmd = ExtractJSONString(payload, "command");
            if(cmd == "") cmd = (msgType == "CLOSE_POSITION" ? "CLOSE_POSITION" : "");
            string inner = ExtractJSONString(payload, "payload");
            if(cmd == "CLOSE_POSITION")
                HandleClosePosition(StringLen(inner) > 0 ? inner : payload);
            else if(cmd == "EMERGENCY_STOP")
                HandleEmergencyStop(inner);
            else if(cmd == "KILL_SWITCH")
                HandleKillSwitch(inner);
            else if(cmd == "REQUEST_SNAPSHOT")
                Print("[Predict-A-Trade] Server command received: REQUEST_SNAPSHOT (master data refresh requested)");
            else if(StringLen(cmd) > 0)
                Print("[Predict-A-Trade] Server command received: ", cmd);
            else
                Print("[Predict-A-Trade] Server command envelope missing 'command' field — acked.");
        }
        else if(msgType == "EMERGENCY_STOP")
            HandleEmergencyStop(payload);
        else if(msgType == "KILL_SWITCH")
            HandleKillSwitch(payload);
        else
            Print("[Predict-A-Trade] Unknown queue item type: ", msgType);

        // Always ACK so the item leaves the queue permanently.
        string ackResult = "{\"status\":\"PROCESSED\",\"type\":\"" + msgType + "\"}";
        PAT_EdgeAck(queueIds[i], ackResult);
    }
}

//+------------------------------------------------------------------+
//| Client MT terminal log: echoes a formatted [Predict-A-Trade] line  |
//| to the MT Experts log AND appends it to error.log (FILE_COMMON)   |
//| so the trader can see why trading is blocked / what was received.  |
//+------------------------------------------------------------------+
 void PAT_LogLine(string msg)
{
    Print("[Predict-A-Trade] ", msg);
    PAT_WriteFile(PAT_ERROR_LOG, PAT_ReadFile(PAT_ERROR_LOG) + "[Predict-A-Trade] " + msg + "\n");
}
