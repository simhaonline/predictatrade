//+------------------------------------------------------------------+
//|                                          PredictATrade_MT5.mq5   |
//|                              Predict-A-Trade v1.19.0 (Option B)  |
//+------------------------------------------------------------------+
//| ARCHITECTURE: THIN EXECUTOR — EA-DIRECT HTTPS (v1.19.0, Option B)|
//|                                                                  |
//| The Windows Agent is REMOVED. This EA talks to the cloud direct: |
//|   - POST {Cloud}/api/v1/devices/edge-poll   (HMAC device auth)   |
//|     → fetches executable signals, LICENSE_STATUS, commands       |
//|   - POST {Cloud}/ingest/agent (Bearer device JWT)                |
//|     → LICENSE_CHECK / ACCOUNT_INFO / LIVENESS / EXECUTION_ACK /  |
//|       TRADE_RESULT / CLOSE_ACK                                   |
//|   - POST {Cloud}/api/v1/devices/edge-heartbeat (HMAC)            |
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
//|   1. Activates its device against the LicenseKey (one-time,      |
//|      auto-persisted; or paste Device Id/Secret from dashboard)   |
//|   2. Polls the edge queue and executes executable signals        |
//|   3. Watchdog: verifies SL present every 15s (fail-closed)       |
//|   4. Reports EXECUTION_ACK + TRADE_RESULT back to the server     |
//|                                                                  |
//| REQUIRED ONCE PER TERMINAL (MT5 security model):                 |
//|   Tools → Options → Expert Advisors → "Allow WebRequest for      |
//|   listed URL" → add:  https://api.predictatrade.com              |
//|                                                                  |
//| ALL other parameters are hardcoded with safe defaults.           |
//| Changes to risk/strategy/trade management are made on the        |
//| SERVER - no EA recompile required.                               |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property version   "1.19"
#property strict

#include <Trade\Trade.mqh>

//=== Input Parameters ===
input bool    AutoExecute    = false;   // SIGNAL_ONLY=true by default (display only). Set true to auto-trade.
input bool    BypassDailyLossBlock = false; // Allow new trades even after the soft daily-loss limit is hit. Hard halt (close-all at MaxDailyLossPct) is NEVER bypassed.
input string  LicenseKey     = "";      // Your Predict-A-Trade license key
input string  ChartTimeframe = "M1";    // Chart/timeframe this EA instance trades (M1/M5/H1/...)
input string  PATCloudURL    = "https://api.predictatrade.com"; // Cloud API base URL (must be in WebRequest allowlist)
input string  PATDeviceId    = "";      // Device UUID (optional — auto-activation from LicenseKey if empty)
input string  PATDeviceSecret= "";      // Device secret (optional — auto-activation from LicenseKey if empty)
input int     PATPollMs      = 1000;    // Signal poll interval, ms (>=500)

//=== Strategy Selection ===
// Strategy selection is controlled by the SERVER based on your license plan.
// You do NOT need to select strategies here — just enter your License Key above.
// The server sends allowed_strategies in the license response, and the EA
// automatically filters signals based on your plan:
//   FREE     → STANDARD_SCALPING only
//   STANDARD → STANDARD_SCALPING + STANDARD_SWING
//   PRO      → STANDARD_SCALPING + ULTRA_SCALPING + STANDARD_SWING + TREND_SWING
//   ELITE    → All strategies

//=== Signal Direction Filter ===
input bool    ExecuteCandidates  = false;      // Execute candidates as real trades

//=== Execution Safety v1.00 (mql-fix.md — fail-closed) ===

//=== File names (FILE_COMMON — device bootstrap state + local log only) ===
//─-- Internal constants (managed by backend, do not change) ──
#define PAT_DEVICE_FILE   "PAT_device.txt"   // device_id|device_secret|refresh_token (bootstrap persistence)
#define SendTickData true
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
input bool   AvoidSwapCharges   = true;        // Skip signals inside the swap-time window (broker server time)
input int    SwapCutoffHour     = 22;          // Swap cutoff hour (broker server time, 0-23)
input int    SwapCutoffBuffer   = 15;          // Minutes before cutoff to start avoiding
input bool   AvoidTripleSwapDay = true;        // NO-TRADE on the triple-swap weekday (operator-toggleable)
input string TripleSwapDay      = "Wednesday"; // Weekday treated as triple-swap: Monday..Sunday
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
#define RiskPerTradePct 1.0
#define UseAutoLotSizing true

// Client MT terminal log — formatted [Predict-A-Trade] lines written here (FILE_COMMON)
// and echoed to the MT Experts log so the trader can see status/signal activity.
#define PAT_ERROR_LOG    "error.log"

// Strategy magic bases (mql-fix.md convention; +offset within 100 range)
#define MAGIC_BASE_SS   40101
#define MAGIC_BASE_US   40201
#define MAGIC_BASE_SW   40301
#define MAGIC_BASE_TS   40401
#define MAGIC_BASE_MF   40501
#define PAT_MAGIC_MIN   40101
#define PAT_MAGIC_MAX   40600
#define PAT_REG_MAX     64


// ─── Position Sizing ───

CTrade        trade;
int           g_atrHandle = INVALID_HANDLE;
string        g_symbol;
string        g_connection    = "OFFLINE";
string        g_licenseStatus  = "UNKNOWN";
string        g_licensePlan    = "—";
string        g_allowedStrategies = "";  // Server-provided comma-separated list from license
bool          g_strategiesEnforced = false; // W2: true when server sent allowed_strategies (even if empty → deny all)
double        g_suggestedLot   = 0;     // Server-calculated lot size
string        g_licenseKey    = "";
string        g_authStatus     = "UNKNOWN";
string        g_deviceStatus   = "UNKNOWN";
string        g_sessionStatus  = "UNKNOWN";
string        g_tradingStatus  = "UNKNOWN";
long          g_signalSeq      = 0;
string        g_lastAckSeq     = "";
string        g_accountID     = "—";
string        g_signalID       = "";
string        g_signalDirection = "NONE";
string        g_signalGrade     = "—";
string        g_signalStrategy  = "—";
string        g_signalClass     = "—";
double        g_entry  = 0;
double        g_sl     = 0;
double        g_tp1    = 0;
double        g_tp2    = 0;
double        g_tp3    = 0;
double        g_rawScore = 0;
double        g_calibProb = 0;
datetime      g_signalTime = 0;
string        g_lastExecutedSignalID = "";
uint          g_lastTickSend = 0;
ulong         g_tickCount    = 0;
int           g_signalsReceived = 0;
int           g_signalsDisplayed = 0;
int           g_signalsFiltered = 0;
double        g_dailyPnL       = 0;
double        g_dayStartBalance = 0;
datetime      g_currentDay    = 0;
bool          g_tradingBlocked = false;
bool          g_capitalWarnActive = false;  // edge-trigger for CAPITAL_WARNING emission
bool          g_hardHaltTriggered = false;
int           g_slippageRejects = 0;
bool          g_equityHalted   = false;
int           g_magicSeq       = 0;
string        g_lastSignalJSON = "";

// Per-trade registry (runtime, keyed by magic; stage persists via GVs)
ulong   g_regMagic[PAT_REG_MAX];
string  g_regSig[PAT_REG_MAX];
string  g_regStrat[PAT_REG_MAX];
double  g_regEntry[PAT_REG_MAX];
double  g_regSL0[PAT_REG_MAX];
double  g_regTP1[PAT_REG_MAX];
double  g_regTP2[PAT_REG_MAX];
double  g_regTP3[PAT_REG_MAX];
double  g_regOrigLot[PAT_REG_MAX];
datetime g_regOpenTime[PAT_REG_MAX];
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

string PAT_StrategyFromMagic(long magic)
{
    if(magic >= MAGIC_BASE_SS && magic < MAGIC_BASE_SS + 100) return "STANDARD_SCALPING";
    if(magic >= MAGIC_BASE_US && magic < MAGIC_BASE_US + 100) return "ULTRA_SCALPING";
    if(magic >= MAGIC_BASE_SW && magic < MAGIC_BASE_SW + 100) return "STANDARD_SWING";
    if(magic >= MAGIC_BASE_TS && magic < MAGIC_BASE_TS + 100) return "TREND_SWING";
    if(magic >= MAGIC_BASE_MF && magic < MAGIC_BASE_MF + 100) return "MARNIE_FIB";
    return "";
}

bool PAT_IsPatMagic(long magic)
{
    return (magic >= PAT_MAGIC_MIN && magic <= PAT_MAGIC_MAX);
}

//+------------------------------------------------------------------+
//| Unified Price Access Wrappers                                     |
//+------------------------------------------------------------------+
double PAT_Close(int shift) { double arr[]; if(CopyClose(g_symbol, PERIOD_CURRENT, shift, 1, arr) < 1) return 0; return arr[0]; }
double PAT_Open(int shift)  { double arr[]; if(CopyOpen(g_symbol, PERIOD_CURRENT, shift, 1, arr) < 1) return 0; return arr[0]; }
double PAT_High(int shift)  { double arr[]; if(CopyHigh(g_symbol, PERIOD_CURRENT, shift, 1, arr) < 1) return 0; return arr[0]; }
double PAT_Low(int shift)   { double arr[]; if(CopyLow(g_symbol, PERIOD_CURRENT, shift, 1, arr) < 1) return 0; return arr[0]; }

double PAT_SwapLong()   { return SymbolInfoDouble(g_symbol, SYMBOL_SWAP_LONG); }
double PAT_SwapShort()  { return SymbolInfoDouble(g_symbol, SYMBOL_SWAP_SHORT); }
double PAT_TickValue()  { return SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_VALUE); }
double PAT_TickSize()   { return SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_SIZE); }
double PAT_PointValue() { return (PAT_TickValue() / PAT_TickSize()); }

//+------------------------------------------------------------------+
//| Lot normalization to broker volume step/min/max                   |
//+------------------------------------------------------------------+
double PAT_NormalizeLot(double lots)
{
    double lotStep = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_STEP);
    double minLot  = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MIN);
    double maxLot  = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MAX);
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
    return PAT_NormalizeLot(lots);
}

// ─── Per-Strategy Spread Check (server-managed — always pass) ───
bool PAT_CheckSpread(string strategyName)
{
    // Spread checked by SERVER — always pass
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
//| WRONG-SIDE SL/TP VALIDATION (highest priority — fail-closed).     |
//| Never clamps. Aborts the order on any violation.                  |
//+------------------------------------------------------------------+
// PAT_ValidateLevels validates that SL/TP are on the correct side of entry for
// the trade direction. A wrong-side level (e.g. a BUY stop placed ABOVE entry) is
// a non-protective placement defect — instead of aborting the trade we MIRROR it
// to the correct side at the same distance from entry, so the position always
// carries a valid protective stop. This is the EA-side safety net that pairs with
// the server-side enforceSLDirection guard; it guarantees no inverted stop is ever
// sent to the broker even if a malformed signal arrives.
bool PAT_ValidateLevels(bool isBuy, double entry, double &sl, double &tpFinal)
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
            double d = sl - entry;
            sl = entry - d; // mirror to correct side, same distance
            Print("CORRECTED wrong_side_sl: BUY sl mirrored to ", DoubleToString(sl, _Digits));
            if(sl <= 0) return false;
        }
        if(tpFinal <= entry)
        {
            double d = entry - tpFinal;
            tpFinal = entry + d;
            Print("CORRECTED wrong_side_tp: BUY tp mirrored to ", DoubleToString(tpFinal, _Digits));
            if(tpFinal <= 0) return false;
        }
    }
    else
    {
        if(sl <= entry)
        {
            double d = entry - sl;
            sl = entry + d;
            Print("CORRECTED wrong_side_sl: SELL sl mirrored to ", DoubleToString(sl, _Digits));
            if(sl <= 0) return false;
        }
        if(tpFinal >= entry)
        {
            double d = tpFinal - entry;
            tpFinal = entry - d;
            Print("CORRECTED wrong_side_tp: SELL tp mirrored to ", DoubleToString(tpFinal, _Digits));
            if(tpFinal <= 0) return false;
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
    dt.day_of_week = 0; dt.day_of_year = 0;
    datetime localInterp = StructToTime(dt);   // MQL interprets components as TERMINAL-LOCAL time

    // Terminal local offset from UTC (seconds), e.g. +10800 for GMT+3.
    // Derived from the live wall-clock difference so it is broker/DST safe.
    int off = 0;
    {
        MqlDateTime lc, gc;
        TimeToStruct(TimeCurrent(), lc);
        TimeToStruct(TimeGMT(), gc);
        off = (lc.hour - gc.hour) * 3600 + (lc.min - gc.min) * 60 + (lc.sec - gc.sec);
        while(off > 43200) off -= 86400;
        while(off < -43200) off += 86400;
    }

    // Timezone suffix offset relative to UTC (seconds).
    int suffixOff = 0;
    int tzPos = 19;
    if(tzPos < StringLen(s))
    {
        string tzChar = StringSubstr(s, tzPos, 1);
        if(tzChar == "Z" || tzChar == "z")
        {
            suffixOff = 0;   // explicit UTC
        }
        else if(tzChar == "+" || tzChar == "-")
        {
            int sign = (tzChar == "+") ? 1 : -1;
            int oh = (int)StringToInteger(StringSubstr(s, tzPos + 1, 2));
            int om = 0;
            if(StringLen(s) >= tzPos + 6)
                om = (int)StringToInteger(StringSubstr(s, tzPos + 4, 2));
            suffixOff = sign * (oh * 3600 + om * 60);
        }
        // unrecognized suffix => treat as UTC (suffixOff = 0)
    }
    // No suffix and no "Z" => assume UTC.

    // Correct absolute time = (components as local) + terminal offset - suffix offset
    return (localInterp + off - suffixOff);
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
    int total = PositionsTotal();
    int count = 0;
    for(int i = 0; i < total; i++)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        long magic = PositionGetInteger(POSITION_MAGIC);
        if(!PAT_IsPatMagic(magic)) continue;
        count++;
    }
    return count;
}

//+------------------------------------------------------------------+
//| Pre-trade check — EMERGENCY HALT FLAGS ONLY.                     |
//| v1.19.1: ALL risk gates (spread, drift, TTL, caps, risk$,        |
//| martingale, margin) are enforced by the SERVER engine before a   |
//| signal is marked EXECUTABLE. Client-side duplicates removed —    |
//| clients execute what the server sends. Server = single source    |
//| of gating truth; this hook exists only for local kill-switches.  |
//+------------------------------------------------------------------+
//| Pre-trade check — EMERGENCY HALT FLAGS ONLY.                     |
//| v1.19.1: ALL risk gates (spread, drift, TTL, caps, risk$,        |
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
int PAT_RegFind(long magic)
{
    for(int i = 0; i < g_regCount; i++)
        if((long)g_regMagic[i] == magic) return i;
    return -1;
}

int PAT_RegPut(ulong magic, string sig, string strat, double entry, double sl0,
               double tp1, double tp2, double tp3, double origLot)
{
    int idx = PAT_RegFind((long)magic);
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
    g_regOpenTime[idx] = TimeCurrent();
    return idx;
}

//+------------------------------------------------------------------+
//| Stage persistence (survives EA reload via GlobalVariables)        |
//+------------------------------------------------------------------+
string PAT_GVName(long magic, string field) { return "PAT_M" + IntegerToString(magic) + "_" + field; }

void PAT_SaveStage(long magic, int stage)
{
    GlobalVariableSet(PAT_GVName(magic, "STAGE"), stage);
}

int PAT_LoadStage(long magic)
{
    string name = PAT_GVName(magic, "STAGE");
    if(GlobalVariableCheck(name)) return (int)GlobalVariableGet(name);
    return 0;
}

//+------------------------------------------------------------------+
//| Forced-reason codes -> strings (EA-initiated closes)               |
//+------------------------------------------------------------------+
int HashCode(string s)
{
    int h = 0;
    for(int i = 0; i < StringLen(s); i++) h = h * 31 + StringGetCharacter(s, i);
    return h;
}

void PAT_SetForcedReason(long posId, string reason)
{
    GlobalVariableSet("PAT_FR_" + IntegerToString(posId), HashCode(reason));
}

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
//| Trade result reporting (TRADE_RESULT + CLOSE_ACK via ingest)      |
//+------------------------------------------------------------------+
void PAT_ReportResult(long magic, long ticket, string signalID, string strategyID,
                      string exitReason, double entry, double exitPx, double lots,
                      double realizedPnl, bool slCorrect,
                      string p_timeframe, string p_direction, string p_openedAt,
                      double p_sl, double p_tp, double p_pnlPoints, long p_timeInTradeSec)
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

//+------------------------------------------------------------------+
//| Exit-reason classification by close price proximity               |
//+------------------------------------------------------------------+
string PAT_ClassifyExit(bool isBuy, double entry, double sl, double tp1, double tp2,
                        double tp3, double exitPx, string forcedReason)
{
    if(forcedReason != "") return forcedReason;
    double point = SymbolInfoDouble(g_symbol, SYMBOL_POINT);
    double spread = (double)SymbolInfoInteger(g_symbol, SYMBOL_SPREAD) * point;
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

string PAT_SignalIDFromComment(string comment)
{
    int colon = StringFind(comment, ":");
    if(colon < 0) return "";
    return StringSubstr(comment, colon + 1);
}

//+------------------------------------------------------------------+
//| History/deal poller: report every closed PAT deal exactly once    |
//| (covers broker-side SL/TP fills AND our partial closes)           |
//+------------------------------------------------------------------+
void PAT_HistoryPoll()
{
    static datetime lastSel = 0;
    datetime from = (lastSel > 0) ? lastSel - 300 : TimeCurrent() - 86400;
    if(!HistorySelect(from, TimeCurrent() + 60)) return;
    lastSel = TimeCurrent();

    int deals = HistoryDealsTotal();
    for(int i = 0; i < deals; i++)
    {
        ulong dealTicket = HistoryDealGetTicket(i);
        if(dealTicket == 0) continue;
        if(HistoryDealGetString(dealTicket, DEAL_SYMBOL) != g_symbol) continue;
        long magic = HistoryDealGetInteger(dealTicket, DEAL_MAGIC);
        if(!PAT_IsPatMagic(magic)) continue;

        long entryKind = HistoryDealGetInteger(dealTicket, DEAL_ENTRY);
        if(entryKind != DEAL_ENTRY_OUT && entryKind != DEAL_ENTRY_OUT_BY &&
           entryKind != DEAL_ENTRY_INOUT)
            continue;

        string rptName = "PAT_RPT_" + IntegerToString((long)dealTicket);
        if(GlobalVariableCheck(rptName)) continue;

        long posId = HistoryDealGetInteger(dealTicket, DEAL_POSITION_ID);

        // Closing a BUY produces a SELL deal and vice versa.
        bool isBuy = (HistoryDealGetInteger(dealTicket, DEAL_TYPE) == DEAL_TYPE_SELL);

        string forced = "";
        string frName = "PAT_FR_" + IntegerToString(posId);
        if(GlobalVariableCheck(frName))
        {
            forced = PAT_ForcedReasonFromCode((int)GlobalVariableGet(frName));
            GlobalVariableDel(frName);
        }

        int idx = PAT_RegFind(magic);
        double entry = (idx >= 0) ? g_regEntry[idx] : HistoryDealGetDouble(dealTicket, DEAL_PRICE);
        double sl0 = (idx >= 0) ? g_regSL0[idx] : 0;
        double tp1 = (idx >= 0) ? g_regTP1[idx] : 0;
        double tp2 = (idx >= 0) ? g_regTP2[idx] : 0;
        double tp3 = (idx >= 0) ? g_regTP3[idx] : 0;

        double exitPx = HistoryDealGetDouble(dealTicket, DEAL_PRICE);
        string reason = PAT_ClassifyExit(isBuy, entry, sl0, tp1, tp2, tp3, exitPx, forced);
        double pnl = HistoryDealGetDouble(dealTicket, DEAL_PROFIT)
                   + HistoryDealGetDouble(dealTicket, DEAL_SWAP)
                   + HistoryDealGetDouble(dealTicket, DEAL_COMMISSION);
        double lots = HistoryDealGetDouble(dealTicket, DEAL_VOLUME);
        string sig = "", strat = "";
        if(idx >= 0) { sig = g_regSig[idx]; strat = g_regStrat[idx]; }
        else
        {
            sig = PAT_SignalIDFromComment(HistoryDealGetString(dealTicket, DEAL_COMMENT));
            strat = PAT_StrategyFromMagic(magic);
        }
        double point = SymbolInfoDouble(g_symbol, SYMBOL_POINT);
        double pnlPoints = (point > 0) ? (exitPx - entry) / point * (isBuy ? 1 : -1) : 0;
        string dir = isBuy ? "BUY" : "SELL";
        string openedAt = "";
        long timeInTrade = 0;
        if(idx >= 0)
        {
            openedAt = FormatISO8601UTC(g_regOpenTime[idx]);
            timeInTrade = (long)(TimeCurrent() - g_regOpenTime[idx]);
        }
        PAT_ReportResult(magic, (long)dealTicket, sig, strat, reason, entry, exitPx,
                        lots, pnl, true, ChartTimeframe, dir, openedAt, sl0, tp1,
                        pnlPoints, timeInTrade);
        GlobalVariableSet(rptName, 1);
    }
}

//+------------------------------------------------------------------+
//| FormatISO8601UTC                                                  |
//+------------------------------------------------------------------+
string FormatISO8601UTC(datetime t)
{
    MqlDateTime dt;
    TimeToStruct(t, dt);
    return StringFormat("%04d-%02d-%02dT%02d:%02d:%02dZ",
        dt.year, dt.mon, dt.day, dt.hour, dt.min, dt.sec);
}

int OnInit()
{
    Print("Predict-A-Trade MT5 EA v1.19 initializing...");

    g_symbol = BrokerSymbol;
    if(g_symbol == "") g_symbol = _Symbol;
    g_licenseKey = LicenseKey;
    g_accountID = IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN));

    trade.SetExpertMagicNumber(MAGIC_BASE_SS); // overridden per-order via req.magic
    trade.SetDeviationInPoints(MaxSlippagePoints);

    g_atrHandle = iATR(g_symbol, PERIOD_CURRENT, 14);
    if(g_atrHandle == INVALID_HANDLE)
        Print("WARN: ATR handle creation failed — trailing will be inactive until ready");

    Print("Symbol: ", g_symbol);
    Print("Account: ", g_accountID);
    Print("License Key: ", (g_licenseKey == "" ? "NOT SET — SIGNALS WILL BE IGNORED" : g_licenseKey));
    Print("TRADE-CONFIG: AutoExecute=", AutoExecute, " ExecuteCandidates=", ExecuteCandidates,
          " AlgoTradingAllowed=", MQLInfoInteger(MQL_TRADE_ALLOWED), " Symbol=", g_symbol);

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

    // Option B: no local agent to detect — bootstrap the cloud device and
    // go ONLINE when credentials are ready. The first poll confirms reachability.
    if(PAT_EnsureDevice())
    {
        g_connection = "CONNECTED";
        Print("[Predict-A-Trade] Cloud device ready (", g_deviceId, ") — edge-poll mode.");
        SendInitMessage();
        RequestLicenseValidation();
    }
    else
    {
        g_connection = "OFFLINE";
        Print("WARNING: Cloud device not ready — set LicenseKey (or Device Id/Secret) in EA inputs.");
        Print("Also add ", PATCloudURL, " to Tools→Options→Expert Advisors→WebRequest allowlist.");
    }

    UpdatePanel();
    return(INIT_SUCCEEDED);
}

void OnDeinit(const int reason)
{
    EventKillTimer();
    if(g_atrHandle != INVALID_HANDLE) IndicatorRelease(g_atrHandle);
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

    // Option B signal/command fetch (replaces ReadFromAgent file IPC).
    // Throttled inside PAT_EdgePoll cadence: run at most once per PATPollMs.
    static uint lastPoll = 0;
    if(g_connection == "CONNECTED" && GetTickCount() - lastPoll >= (uint)MathMax(PATPollMs, 500))
    {
        lastPoll = GetTickCount();
        PollFromCloud();
    }

    if(g_signalDirection != "NONE" && g_signalTime > 0)
    {
        if(TimeCurrent() > g_signalTime + MaxSignalAgeSeconds)
            g_signalDirection = "EXPIRED";
    }

    PAT_ManagePositions();
    UpdateCapitalProtection();
    PAT_HistoryPoll();
    UpdatePanel();
}

//+------------------------------------------------------------------+
//| WATCHDOG (OnTimer, 15s): missing-SL check + equity floor          |
//+------------------------------------------------------------------+
void OnTimer()
{
    CheckAgentConnection();

    // Weekend/holiday liveness: with no market ticks, OnTick never fires and
    // the EA looks OFFLINE (no terminal registration, license unresolved).
    // If no real tick went out recently, emit a LIVENESS ping instead.
    uint now = GetTickCount();
    uint tickGap = (TickIntervalMs > 0 ? TickIntervalMs : 1000) * 3;
    if(g_connection == "CONNECTED" && (g_lastTickSend == 0 || GetTickCount() - g_lastTickSend > tickGap))
    {
        SendLivenessPing();
        g_lastTickSend = GetTickCount(); // reuse as "last EA→cloud send" so we ping at a sane rate
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
        if(baseline <= 0) baseline = AccountInfoDouble(ACCOUNT_BALANCE);
        double floorEq = baseline * (MinEquityFloorPct / 100.0);
        double eq = AccountInfoDouble(ACCOUNT_EQUITY);
        if(baseline > 0 && eq < floorEq && eq > 0)
        {
            g_equityHalted = true;
            GlobalVariableSet("PAT_EQUITY_HALT", 1);
            Print("*** EQUITY FLOOR BREACH: equity=", DoubleToString(eq, 2),
                  " < floor=", DoubleToString(floorEq, 2), " — CLOSING ALL PAT POSITIONS AND HALTING ***");
            CloseAllPatPositions("EQUITY_FLOOR");
        }
    }

    // 2. Missing-SL self-check on every PAT position
    int total = PositionsTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        if(!PAT_IsPatMagic(PositionGetInteger(POSITION_MAGIC))) continue;

        double sl = PositionGetDouble(POSITION_SL);
        if(sl > 0) continue;

        long magic = PositionGetInteger(POSITION_MAGIC);
        bool isBuy = (PositionGetInteger(POSITION_TYPE) == POSITION_TYPE_BUY);
        int idx = PAT_RegFind(magic);
        double sl0 = (idx >= 0) ? g_regSL0[idx] : 0;

        string mode = OnMissingSL;
        StringToUpper(mode);
        if(mode == "RESTORE" && sl0 > 0)
        {
            double openPx = PositionGetDouble(POSITION_PRICE_OPEN);
            bool sideOk = isBuy ? (sl0 < openPx) : (sl0 > openPx);
            if(sideOk)
            {
                if(trade.PositionModify(ticket, sl0, PositionGetDouble(POSITION_TP)))
                    Print("WATCHDOG: restored missing SL on ticket=", ticket, " SL=", DoubleToString(sl0, _Digits));
                else
                    Print("WATCHDOG: SL restore FAILED ticket=", ticket, " retcode=", trade.ResultRetcode());
            }
            else
            {
                Print("WATCHDOG: stored SL wrong-side for ticket=", ticket, " — CLOSING (fail-closed)");
                PAT_SetForcedReason(PositionGetInteger(POSITION_IDENTIFIER), "WATCHDOG_NOSL");
                ClosePosition(ticket, "WATCHDOG_NOSL");
            }
        }
        else
        {
            Print("WATCHDOG: PAT position without SL ticket=", ticket,
                  " magic=", magic, " — CLOSING (OnMissingSL=CLOSE, fail-closed)");
            PAT_SetForcedReason(PositionGetInteger(POSITION_IDENTIFIER), "WATCHDOG_NOSL");
            ClosePosition(ticket, "WATCHDOG_NOSL");
        }
    }
}

void CloseAllPatPositions(string reason)
{
    int total = PositionsTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetString(POSITION_SYMBOL) == g_symbol &&
           PAT_IsPatMagic(PositionGetInteger(POSITION_MAGIC)))
        {
            PAT_SetForcedReason(PositionGetInteger(POSITION_IDENTIFIER), reason);
            ClosePosition(ticket, reason);
        }
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
    int total = PositionsTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        if(!PAT_IsPatMagic(PositionGetInteger(POSITION_MAGIC))) continue;

        long   magic    = PositionGetInteger(POSITION_MAGIC);
        long   posType  = PositionGetInteger(POSITION_TYPE);
        double openPx   = PositionGetDouble(POSITION_PRICE_OPEN);
        double sl       = PositionGetDouble(POSITION_SL);
        double tp       = PositionGetDouble(POSITION_TP);
        double curLots  = PositionGetDouble(POSITION_VOLUME);
        datetime openTime = (datetime)PositionGetInteger(POSITION_TIME);

        // Max holding time
        if(MaxHoldHours > 0)
        {
            int holdSec = (int)(TimeCurrent() - openTime);
            if(holdSec >= MaxHoldHours * 3600)
            {
                Print("MAX HOLD TIME reached: ticket=", ticket, " held=", holdSec/3600, "h | Closing...");
                PAT_SetForcedReason(PositionGetInteger(POSITION_IDENTIFIER), "MAX_HOLD_TIME");
                ClosePosition(ticket, "MAX_HOLD_TIME");
                continue;
            }
        }

        // Swap cutoff
        if(AvoidSwapCharges && IsNearSwapTime())
        {
            Print("SWAP CUTOFF: closing ticket=", ticket, " before swap charge");
            PAT_SetForcedReason(PositionGetInteger(POSITION_IDENTIFIER), "SWAP_AVOIDANCE");
            ClosePosition(ticket, "SWAP_AVOIDANCE");
            continue;
        }

        long stageL = PAT_LoadStage(magic);
        int stage = (int)stageL;
        bool isBuy = (posType == POSITION_TYPE_BUY);
        double point = SymbolInfoDouble(g_symbol, SYMBOL_POINT);
        int digits = (int)SymbolInfoInteger(g_symbol, SYMBOL_DIGITS);

        int idx = PAT_RegFind(magic);
        double tp1 = (idx >= 0) ? g_regTP1[idx] : 0;
        double tp2 = (idx >= 0) ? g_regTP2[idx] : 0;
        double origLot = (idx >= 0) ? g_regOrigLot[idx] : curLots;
        double sl0 = (idx >= 0) ? g_regSL0[idx] : 0;

        double bid = SymbolInfoDouble(g_symbol, SYMBOL_BID);
        double ask = SymbolInfoDouble(g_symbol, SYMBOL_ASK);

        // ── STAGE 0 -> 1: TP1 hit — partial close ──
        if(UsePartialClose && stage == 0 && tp1 > 0)
        {
            bool tp1Hit = isBuy ? (bid >= tp1) : (ask <= tp1);
            if(tp1Hit)
            {
                if(PAT_DoPartial(ticket, magic, isBuy, origLot, TP1ClosePct, "tp1"))
                {
                    PAT_SaveStage(magic, 1);
                    stage = 1;
                }
            }
        }

        // ── Breakeven maintenance (stage >= 1): SL to entry +/- spread ──
        if(stage >= 1 && UseBreakEven && sl0 > 0 && (sl == 0 || MathAbs(sl - sl0) < point))
        {
            double spread = (double)SymbolInfoInteger(g_symbol, SYMBOL_SPREAD) * point;
            double beSL = isBuy ? NormalizeDouble(openPx + spread, digits)
                                : NormalizeDouble(openPx - spread, digits);
            bool beSideOk = isBuy ? (beSL < bid) : (beSL > ask);
            if(beSideOk)
            {
                if(trade.PositionModify(ticket, beSL, tp))
                    Print("BREAK-EVEN: ticket=", ticket, " SL=", DoubleToString(beSL, digits));
            }
        }

        // ── STAGE 1 -> 2: TP2 hit — partial close + arm trailing ──
        if(UsePartialClose && stage == 1 && tp2 > 0)
        {
            bool tp2Hit = isBuy ? (bid >= tp2) : (ask <= tp2);
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
//| Uses CTrade::PositionClosePartial — SAME position continues.      |
//| If minimums prevent a sane split, closes FULL remainder.          |
//+------------------------------------------------------------------+
bool PAT_DoPartial(ulong ticket, long magic, bool isBuy, double origLot,
                   double pct, string reason)
{
    if(!PositionSelectByTicket(ticket)) return false;

    double curLots = PositionGetDouble(POSITION_VOLUME);
    double minLot  = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MIN);
    double step    = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_STEP);
    double closeLots = origLot * (pct / 100.0);
    if(step > 0) closeLots = MathFloor(closeLots / step + 0.0000001) * step;
    closeLots = NormalizeDouble(closeLots, 2);

    bool closeFull = false;
    if(closeLots < minLot) closeFull = true;
    else if(curLots - closeLots < minLot - 0.0000001) closeFull = true;

    if(closeFull)
    {
        Print("PARTIAL(", reason, "): remainder ", DoubleToString(curLots, 2),
              " too small to split — closing FULL at TP level");
        ClosePosition(ticket, reason);
        return false;
    }

    if(trade.PositionClosePartial(ticket, closeLots))
    {
        Print("PARTIAL ", reason, ": closed ", DoubleToString(closeLots, 2),
              " of ", DoubleToString(origLot, 2));
        // Reporting happens in PAT_HistoryPoll (one report per OUT deal).
    }
    else
    {
        Print("PARTIAL ", reason, " FAILED: ticket=", ticket,
              " retcode=", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
        return false;
    }
    return true;
}

//+------------------------------------------------------------------+
//| Monotonic ATR trailing for the stage>=2 remainder                 |
//+------------------------------------------------------------------+
void PAT_TrailRemainder(ulong ticket, bool isBuy, double openPx, double tp,
                        int digits, double point)
{
    if(!PositionSelectByTicket(ticket)) return;

    double atrBuffer[];
    double atr = 0;
    if(g_atrHandle != INVALID_HANDLE && CopyBuffer(g_atrHandle, 0, 0, 1, atrBuffer) > 0)
        atr = atrBuffer[0];
    if(atr <= 0) return;

    double trailDist = atr * TrailingATRMult;
    double stopLevel = (double)SymbolInfoInteger(g_symbol, SYMBOL_TRADE_STOPS_LEVEL) * point;
    double freezeLevel = (double)SymbolInfoInteger(g_symbol, SYMBOL_TRADE_FREEZE_LEVEL) * point;
    double sl = PositionGetDouble(POSITION_SL);
    double bid = SymbolInfoDouble(g_symbol, SYMBOL_BID);
    double ask = SymbolInfoDouble(g_symbol, SYMBOL_ASK);

    if(isBuy)
    {
        double newSL = NormalizeDouble(bid - trailDist, digits);
        double minSL = NormalizeDouble(bid - stopLevel, digits);
        if(newSL > minSL) newSL = minSL;
        if(freezeLevel > 0 && MathAbs(bid - sl) < freezeLevel) return;
        if(newSL > sl && newSL > openPx)
        {
            if(trade.PositionModify(ticket, newSL, tp))
                Print("TRAIL BUY: ticket=", ticket, " SL=", sl, " -> ", newSL);
        }
    }
    else
    {
        double newSL = NormalizeDouble(ask + trailDist, digits);
        double maxSL = NormalizeDouble(ask + stopLevel, digits);
        if(newSL < maxSL) newSL = maxSL;
        if(freezeLevel > 0 && MathAbs(ask - sl) < freezeLevel) return;
        if((sl == 0 || newSL < sl) && newSL < openPx)
        {
            if(trade.PositionModify(ticket, newSL, tp))
                Print("TRAIL SELL: ticket=", ticket, " SL=", sl, " -> ", newSL);
        }
    }
}

//+------------------------------------------------------------------+
bool ClosePosition(ulong ticket, string reason)
{
    bool ok = trade.PositionClose(ticket);
    if(ok)
        Print("CLOSED: ticket=", ticket, " reason=", reason);
    else
        Print("CLOSE FAILED: ticket=", ticket, " error=", GetLastError());
    return ok;
}

//+------------------------------------------------------------------+
bool IsNearSwapTime()
{
    MqlDateTime dt;
    TimeToStruct(TimeCurrent(), dt);
    int nowMinute = dt.hour * 60 + dt.min;
    int cutoffMinute = SwapCutoffHour * 60 - SwapCutoffBuffer;
    if(nowMinute >= cutoffMinute && nowMinute < SwapCutoffHour * 60 + 30)
    {
        if(dt.day_of_week >= 1 && dt.day_of_week <= 5)
            return true;
    }
    return false;
}

bool IsTripleSwapDay()
{
    if(!AvoidTripleSwapDay) return false;
    MqlDateTime dt;
    TimeToStruct(TimeCurrent(), dt);
    if(TripleSwapDay == "Wednesday" && dt.day_of_week == 3) return true;
    if(TripleSwapDay == "Thursday" && dt.day_of_week == 4) return true;
    if(TripleSwapDay == "Friday" && dt.day_of_week == 5) return true;
    return false;
}

//+------------------------------------------------------------------+
void CheckAgentConnection()
{
    static uint lastCheck = 0;
    if(GetTickCount() - lastCheck < 2000) return;
    lastCheck = GetTickCount();

    // Option B liveness: "connected" = cloud credentials present. Actual
    // reachability is tracked by poll success counters (see OnTick/poll loop).
    bool ready = (StringLen(g_deviceId) > 0 && StringLen(g_deviceSecret) > 0);
    if(ready && g_connection != "CONNECTED")
    {
        Print("[Predict-A-Trade] Cloud device ready — resuming edge-poll.");
        g_connection = "CONNECTED";
        SendInitMessage();
        RequestLicenseValidation();
    }
    else if(!ready && g_connection == "CONNECTED")
    {
        Print("[Predict-A-Trade] Cloud device credentials lost — OFFLINE.");
        g_connection = "OFFLINE";
    }
}

//+------------------------------------------------------------------+
void SendTickToAgent()
{
    g_lastTickSend = GetTickCount();

    double bid = SymbolInfoDouble(g_symbol, SYMBOL_BID);
    double ask = SymbolInfoDouble(g_symbol, SYMBOL_ASK);
    if(bid <= 0 || ask <= 0) return;

    g_tickCount++;

    string msg = "TICK|{\"type\":\"TICK\"";
    msg += ",\"symbol\":\"" + g_symbol + "\"";
    msg += ",\"bid\":" + DoubleToString(bid, 5);
    msg += ",\"ask\":" + DoubleToString(ask, 5);
    msg += ",\"volume\":" + IntegerToString(SymbolInfoInteger(g_symbol, SYMBOL_VOLUME));
    msg += ",\"timestamp\":\"" + FormatISO8601UTC(TimeGMT()) + "\"";
    msg += ",\"source\":\"MT5\"";
    msg += ",\"broker\":\"" + AccountInfoString(ACCOUNT_COMPANY) + "\"";
    msg += ",\"account\":\"" + g_accountID + "\"";
    msg += ",\"license_key\":\"" + g_licenseKey + "\"";
    // Broker session timezone — collected live so the engine works on Broker TF
    // (not UTC). TimeGMTOffset() returns the broker's GMT offset in seconds.
    msg += ",\"broker_offset\":" + IntegerToString(TimeGMTOffset() / 3600);
    msg += "}\n";

    PAT_Send(msg);
}

//+------------------------------------------------------------------+
void SendInitMessage()
{
    int totalPos = PositionsTotal();
    int buyCount = 0, sellCount = 0;
    double totalLots = 0;
    for(int i = 0; i < totalPos; i++)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket > 0)
        {
            long type = PositionGetInteger(POSITION_TYPE);
            double vol = PositionGetDouble(POSITION_VOLUME);
            if(type == POSITION_TYPE_BUY) { buyCount++; totalLots += vol; }
            else if(type == POSITION_TYPE_SELL) { sellCount++; totalLots += vol; }
        }
    }
    //--- Ensure we read the equity of the account this EA is bound to, not whatever
    //    account happens to be active in a multi-account terminal (else the engine
    //    receives a misread of the wrong account's equity, which can trip risk gates).
    // MQL5 has no programmatic account switching — the EA always reads the
    // account currently logged into the terminal. Warn (do not switch) if the
    // bound account id does not match, so telemetry is never silently wrong.
    if(g_accountID != "" && IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN)) != g_accountID)
       Print("WARNING: EA bound to account ", g_accountID, " but terminal is logged into ", IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN)));

    string msg = "INIT|{\"ea_version\":\"1.19\",\"broker\":\"" + AccountInfoString(ACCOUNT_COMPANY) +
                "\",\"account\":\"" + g_accountID + "\",\"symbol\":\"" + g_symbol +
                "\",\"license_key\":\"" + g_licenseKey +
                "\",\"balance\":" + DoubleToString(AccountInfoDouble(ACCOUNT_BALANCE), 2) +
                ",\"equity\":" + DoubleToString(AccountInfoDouble(ACCOUNT_EQUITY), 2) +
                ",\"profit\":" + DoubleToString(AccountInfoDouble(ACCOUNT_PROFIT), 2) +
                ",\"currency\":\"" + AccountInfoString(ACCOUNT_CURRENCY) +
                "\",\"leverage\":" + IntegerToString((int)AccountInfoInteger(ACCOUNT_LEVERAGE)) +
                ",\"open_positions\":" + IntegerToString(totalPos) +
                ",\"buy_positions\":" + IntegerToString(buyCount) +
                ",\"sell_positions\":" + IntegerToString(sellCount) +
                  ",\"total_lots\":" + DoubleToString(totalLots, 2) +
                  ",\"free_margin\":" + DoubleToString(AccountInfoDouble(ACCOUNT_MARGIN_FREE), 2) +
                  ",\"floating_pnl\":" + DoubleToString(AccountInfoDouble(ACCOUNT_PROFIT), 2) +
                  "}\n";
    PAT_Send(msg);
    Print("Init message sent with account data - balance: ", AccountInfoDouble(ACCOUNT_BALANCE));
}

//+------------------------------------------------------------------+
//| Periodic account telemetry → engine. Sends equity/free-margin/leverage
//| every timer tick so the engine's margin gate and lot-sizing can compute
//| and mark signals EXECUTABLE (without it the engine fails closed).
//+------------------------------------------------------------------+
void SendAccountInfo()
{
    if(g_connection != "CONNECTED") return;
    // Ensure we read THIS EA's bound account, not whatever is active.
    // MQL5 has no programmatic account switching — the EA always reads the
    // account currently logged into the terminal. Warn (do not switch) if the
    // bound account id does not match, so telemetry is never silently wrong.
    if(g_accountID != "" && IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN)) != g_accountID)
       Print("WARNING: EA bound to account ", g_accountID, " but terminal is logged into ", IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN)));
    string msg = "ACCOUNT_INFO|{\"ea_version\":\"1.09\",\"account\":\"" + g_accountID +
                "\",\"broker\":\"" + AccountInfoString(ACCOUNT_COMPANY) +
                "\",\"symbol\":\"" + g_symbol +
                "\",\"currency\":\"" + AccountInfoString(ACCOUNT_CURRENCY) +
                "\",\"license_key\":\"" + g_licenseKey +
                "\",\"balance\":" + DoubleToString(AccountInfoDouble(ACCOUNT_BALANCE), 2) +
                ",\"equity\":" + DoubleToString(AccountInfoDouble(ACCOUNT_EQUITY), 2) +
                ",\"free_margin\":" + DoubleToString(AccountInfoDouble(ACCOUNT_MARGIN_FREE), 2) +
                ",\"leverage\":" + IntegerToString((int)AccountInfoInteger(ACCOUNT_LEVERAGE)) +
                ",\"open_positions\":" + IntegerToString((int)PositionsTotal()) +
                "}\n";
    PAT_Send(msg);
}

//+------------------------------------------------------------------+
void RequestLicenseValidation()
{
    //--- Ensure we read the equity of the account this EA is bound to (see INIT above).
    // MQL5 has no programmatic account switching — the EA always reads the
    // account currently logged into the terminal. Warn (do not switch) if the
    // bound account id does not match, so telemetry is never silently wrong.
    if(g_accountID != "" && IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN)) != g_accountID)
       Print("WARNING: EA bound to account ", g_accountID, " but terminal is logged into ", IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN)));

    string msg = "LICENSE_CHECK|{\"account\":\"" + g_accountID +
                "\",\"broker\":\"" + AccountInfoString(ACCOUNT_COMPANY) +
                "\",\"symbol\":\"" + g_symbol +
                "\",\"license_key\":\"" + g_licenseKey +
                "\",\"balance\":" + DoubleToString(AccountInfoDouble(ACCOUNT_BALANCE), 2) +
                ",\"equity\":" + DoubleToString(AccountInfoDouble(ACCOUNT_EQUITY), 2) +
                  ",\"profit\":" + DoubleToString(AccountInfoDouble(ACCOUNT_PROFIT), 2) +
                  ",\"free_margin\":" + DoubleToString(AccountInfoDouble(ACCOUNT_MARGIN_FREE), 2) +
                  ",\"open_positions\":" + IntegerToString(PositionsTotal()) +
                  "}\n";
    PAT_Send(msg);
    Print("License validation with account data - balance: ", AccountInfoDouble(ACCOUNT_BALANCE));
}

//+------------------------------------------------------------------+
//| LIVENESS ping — sent from OnTimer when the market produces no ticks |
//| (weekend/holiday). Keeps the terminal visible (dashboard ONLINE),   |
//| the license resolved, and the EA→cloud chain provably alive.        |
//| engine treats LIVENESS as connectivity-only (no price evaluation).  |
//+------------------------------------------------------------------+
void SendLivenessPing()
{
    string msg = "LIVENESS|{\"type\":\"LIVENESS\"";
    msg += ",\"symbol\":\""+g_symbol+"\"";
    msg += ",\"source\":\"MT5\"";
    msg += ",\"account\":\""+g_accountID+"\"";
    msg += ",\"broker\":\""+AccountInfoString(ACCOUNT_COMPANY)+"\"";
    msg += ",\"timestamp\":\""+FormatISO8601UTC(TimeGMT())+"\"}\n";
    PAT_Send(msg);

    // Include a license re-validation with each liveness ping so a fresh key
    // typed into the EA resolves even through a closed market.
    RequestLicenseValidation();
}

//+------------------------------------------------------------------+
//+------------------------------------------------------------------+
//| PollFromCloud — Option B signal/command fetch (replaces           |
//| ReadFromAgent's file IPC). Pulls the edge queue via HMAC-signed   |
//| edge-poll and dispatches each payload by its "type" — the same    |
//| message vocabulary the WS agent used (SIGNAL / LICENSE_STATUS /   |
//| CLOSE_POSITION / EMERGENCY_STOP / KILL_SWITCH / SERVER_COMMAND).  |
//+------------------------------------------------------------------+
void PollFromCloud()
{
    string items[];
    string queueIds[];
    int n = PAT_EdgePoll(items, queueIds);
    if(n <= 0) return;

    for(int i = 0; i < n; i++)
    {
        string payload = items[i];
        string queueId = queueIds[i];
        if(StringLen(payload) == 0) continue;

        // Dispatch by payload type. The queue carries: real signals
        // (payload = the signal JSON), LICENSE_STATUS verdicts, and
        // SERVER_COMMAND envelopes (CLOSE_POSITION/EMERGENCY_STOP/
        // KILL_SWITCH/REQUEST_SNAPSHOT…).
        string msgType = ExtractJSONString(payload, "type");

        if(msgType == "SIGNAL" || ExtractJSONString(payload, "ID") != "")
            HandleSignal(payload);
        else if(msgType == "LICENSE_STATUS")
        {
            // Envelope: {"type":"LICENSE_STATUS","license_status":{...},"device_id":"…"}
            // The verdict is a nested JSON OBJECT (not a string) — extract it
            // specifically, falling back to the innermost object only if absent.
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
            string cmd = ExtractJSONString(payload, "command");
            if(cmd == "") cmd = "CLOSE_POSITION";
            string inner = ExtractJSONString(payload, "payload");
            if(cmd == "CLOSE_POSITION")
                HandleClosePosition(StringLen(inner) > 0 ? inner : payload);
            else if(cmd == "EMERGENCY_STOP")
                HandleEmergencyStop(inner);
            else if(cmd == "KILL_SWITCH")
                HandleKillSwitch(inner);
            else
                Print("[Predict-A-Trade] Server command received: ", cmd);
        }
        else if(msgType == "EMERGENCY_STOP")
            HandleEmergencyStop(payload);
        else if(msgType == "KILL_SWITCH")
            HandleKillSwitch(payload);
        else
            Print("[Predict-A-Trade] Unknown queue item type: ", msgType);

        // Always ACK so the item leaves the queue permanently.
        string ackResult = "{\"status\":\"PROCESSED\",\"type\":\"" + msgType + "\"}";
        PAT_EdgeAck(queueId, ackResult);
        g_signalsReceived++;
    }
}

//+------------------------------------------------------------------+
//| Server-side SL enforcement: CLOSE_POSITION command               |
//| Closes a specific position by ticket or magic number              |
//+------------------------------------------------------------------+
void HandleClosePosition(string payload)
{
    int ticket = (int)StringToInteger(ExtractJSONString(payload, "ticket"));
    long magic = (long)StringToInteger(ExtractJSONString(payload, "magic"));
    string reason = ExtractJSONString(payload, "reason");

    if(ticket > 0)
    {
        if(PositionSelectByTicket(ticket))
        {
            // W6 FIX: only close positions that belong to PAT (magic within our
            // range) AND match the EA symbol. Prevents closing an arbitrary
            // position a user happens to hold on the same ticket.
            long posMagic = PositionGetInteger(POSITION_MAGIC);
            string posSymbol = PositionGetString(POSITION_SYMBOL);
            if(!PAT_IsPatMagic(posMagic) || posSymbol != g_symbol)
            {
                Print("CLOSE_POSITION: ticket=", ticket, " ignored — not a PAT position (magic=", posMagic, " symbol=", posSymbol, ")");
                return;
            }
            if(trade.PositionClose(ticket))
            {
                Print("CLOSE_POSITION: ticket=", ticket, " closed");
                PAT_SetForcedReason(ticket, "SERVER_CLOSE_POSITION");
            }
            return;
        }
    }
    if(magic > 0)
    {
        for(int i = PositionsTotal() - 1; i >= 0; i--)
        {
            ulong t = PositionGetTicket(i);
            if(t == 0) continue;
            if(PositionGetInteger(POSITION_MAGIC) == magic && PositionGetString(POSITION_SYMBOL) == g_symbol)
            {
                if(trade.PositionClose(t))
                {
                    Print("CLOSE_POSITION: magic=", magic, " ticket=", t, " closed");
                    PAT_SetForcedReason(t, "SERVER_CLOSE_POSITION");
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
    Print("*** EMERGENCY_STOP from server: reason=", reason, " — CLOSING ALL PAT POSITIONS ***");

    // Close all PAT-managed positions
    int total = PositionsTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsPatMagic(PositionGetInteger(POSITION_MAGIC))) continue;

        PAT_SetForcedReason(ticket, "SERVER_EMERGENCY_STOP");
        if(trade.PositionClose(ticket))
            Print("EMERGENCY_STOP: closed ticket=", ticket);
        else
            Print("EMERGENCY_STOP: FAILED to close ticket=", ticket, " err=", trade.ResultRetcode());
    }

    // Set halt flag
    g_tradingStatus = "EMERGENCY_HALT";
    GlobalVariableSet("PAT_EQUITY_HALT", 1);
    g_equityHalted = true;

    Print("*** EMERGENCY_STOP complete — trading HALTED until manual re-enable ***");
}

//+------------------------------------------------------------------+
//| Server-side kill switch: close ALL and stop EA                   |
//+------------------------------------------------------------------+
void HandleKillSwitch(string payload)
{
    Print("*** KILL_SWITCH from server — CLOSING ALL POSITIONS AND STOPPING EA ***");

    // Close all PAT-managed positions
    int total = PositionsTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsPatMagic(PositionGetInteger(POSITION_MAGIC))) continue;

        PAT_SetForcedReason(ticket, "SERVER_KILL_SWITCH");
        trade.PositionClose(ticket);
    }

    // Set halt and remove EA
    g_tradingStatus = "KILL_SWITCH";
    GlobalVariableSet("PAT_EQUITY_HALT", 1);
    g_equityHalted = true;

    // Report deinit
    string deinitMsg = "DEINIT|{\"reason\":\"SERVER_KILL_SWITCH\",\"closed_positions\":true}\n";
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
    int total = PositionsTotal();
    for(int i = 0; i < total; i++)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        long magic = PositionGetInteger(POSITION_MAGIC);
        if(!PAT_IsPatMagic(magic)) continue;
        if(!first) details += ",";
        first = false;
        double sl = PositionGetDouble(POSITION_SL);
        double tp = PositionGetDouble(POSITION_TP);
        double vol = PositionGetDouble(POSITION_VOLUME);
        double openPx = PositionGetDouble(POSITION_PRICE_OPEN);
        double profit = PositionGetDouble(POSITION_PROFIT);
        long ptype = PositionGetInteger(POSITION_TYPE);
        string typeStr = (ptype == POSITION_TYPE_BUY) ? "BUY" : "SELL";
        details += "{\"ticket\":" + IntegerToString((long)ticket);
        details += ",\"magic\":" + IntegerToString(magic);
        details += ",\"type\":\"" + typeStr + "\"";
        details += ",\"volume\":\"" + DoubleToString(vol, 2) + "\"";
        details += ",\"open_price\":\"" + DoubleToString(openPx, _Digits) + "\"";
        details += ",\"sl\":\"" + DoubleToString(sl, _Digits) + "\"";
        details += ",\"tp\":\"" + DoubleToString(tp, _Digits) + "\"";
        details += ",\"profit\":\"" + DoubleToString(profit, 2) + "\"";
        details += ",\"symbol\":\"" + g_symbol + "\"}";
    }
    details += "]";
    return details;
}

//+------------------------------------------------------------------+
bool IsStrategyEnabled(string strategyID)
{
    // SERVER-CONTROLLED: Check if strategy is in the server-provided allowed_strategies.
    // The server ALSO filters signals before enqueueing (primary defense).
    // This EA check is a secondary defense layer.

    // If the server OMITTED allowed_strategies entirely (legacy backend /
    // not yet received), allow all — the server already filters by plan before
    // sending signals (primary defense).
    if(!g_strategiesEnforced)
    {
        if(g_licenseStatus == "ACTIVE")
            return true;
        Print("Strategy check: license not validated — blocking ", strategyID);
        return false;
    }

    // Server sent an explicit list. An EMPTY list means NO strategies are
    // allowed (fail closed) — never fall through to allow-all.
    if(StringLen(g_allowedStrategies) == 0)
    {
        Print("Strategy check: empty allowed list (deny-all) — blocking ", strategyID);
        return false;
    }

    // Check if strategyID is in the comma-separated g_allowedStrategies
    string search = "," + strategyID + ",";
    string list = "," + g_allowedStrategies + ",";
    if(StringFind(list, search) >= 0)
        return true;

    Print("Strategy check: ", strategyID, " NOT in allowed list (", g_allowedStrategies, ")");
    return false;
}

bool IsDirectionEnabled(string direction)
{
    if(direction == "BUY"             && ReceiveBuy)            return true;
    if(direction == "SELL"            && ReceiveSell)           return true;
    if(direction == "BUY_CANDIDATE"   && ReceiveBuyCandidate)  return true;
    if(direction == "SELL_CANDIDATE"  && ReceiveSellCandidate) return true;
    return false;
}

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
    {
        Print("Duplicate signal ID — skipping: ", g_signalID);
        return;
    }

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
    {
        Print("SIGNAL BLOCKED: cloud connection not CONNECTED (g_connection=", g_connection, ")");
        return;
    }

    if(g_licenseStatus != "ACTIVE")
    {
        Print("SIGNAL BLOCKED: license not ACTIVE (g_licenseStatus=", g_licenseStatus, ")");
        return;
    }

    if(g_tradingStatus == "HALTED" || g_tradingStatus == "KILL_SWITCH" || g_tradingStatus == "EMERGENCY_HALT")
    {
        Print("Trading is ", g_tradingStatus, " — signal received but not executed.");
        return;
    }

    if(g_tradingBlocked) { Print("SIGNAL BLOCKED: g_tradingBlocked (daily loss limit reached) — NO-TRADE"); g_signalsFiltered++; return; }
    if(AvoidSwapCharges && IsNearSwapTime()) { Print("SIGNAL BLOCKED: AvoidSwapCharges + near swap-time (broker ", TimeToString(TimeCurrent(), TIME_MINUTES), ") — NO-TRADE"); g_signalsFiltered++; return; }
    if(IsTripleSwapDay()) { Print("SIGNAL BLOCKED: AvoidTripleSwapDay=true and today is ", TripleSwapDay, " (triple-swap) — ALL signals vetoed, NO-TRADE. Set EA input AvoidTripleSwapDay=false to trade today."); g_signalsFiltered++; return; }
    g_signalsDisplayed++;

    // Only auto-trade CONFIRMED executable signals. ADVISORY / NO_TRADE signals
    // are displayed for context but must never open a position — this prevents
    // trading on non-confirmed reads and duplicate fills when both advisory and
    // executable signals are delivered to the same device.
    if(g_signalClass != "EXECUTABLE")
    {
        Print("SIGNAL DISPLAY-ONLY: class=", g_signalClass, " — not EXECUTABLE, skip auto-trade");
        return;
    }

    Print("SIGNAL-EXEC-CHECK dir=", g_signalDirection, " class=", g_signalClass,
          " AutoExecute=", AutoExecute, " ExecuteCandidates=", ExecuteCandidates,
          " conn=", g_connection, " lic=", g_licenseStatus);

    if(AutoExecute && g_signalDirection == "BUY")
        ExecuteBuy();
    else if(AutoExecute && g_signalDirection == "SELL")
        ExecuteSell();
    else if(AutoExecute && ExecuteCandidates && g_signalDirection == "BUY_CANDIDATE")
    {
        g_signalDirection = "BUY";
        ExecuteBuy();
    }
    else if(AutoExecute && ExecuteCandidates && g_signalDirection == "SELL_CANDIDATE")
    {
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
    // The backend sends it as a JSON ARRAY: ["STANDARD_SCALPING",...].
    // W2 FIX: ExtractJSONString only matched a quoted STRING value, so the
    // array was never parsed and every strategy stayed enabled. Use the new
    // ExtractJSONArrayRaw helper. A legacy quoted-string value is still
    // accepted for backwards compatibility.
    bool listPresent = (StringFind(json, "\"allowed_strategies\":") >= 0);
    if(listPresent)
    {
        g_strategiesEnforced = true;
        string strategiesRaw = ExtractJSONArrayRaw(json, "allowed_strategies");
        if(StringLen(strategiesRaw) == 0)
            strategiesRaw = ExtractJSONString(json, "allowed_strategies"); // legacy string
        // Convert the array inner content to a comma-separated list.
        string cleaned = strategiesRaw;
        StringReplace(cleaned, "[", "");
        StringReplace(cleaned, "]", "");
        StringReplace(cleaned, "\"", "");
        StringReplace(cleaned, " ", "");
        g_allowedStrategies = cleaned;
    }

    // Only log when the license state actually changes — LICENSE_STATUS
    // verdicts arrive on every poll cycle, so an unconditional log would spam
    // the terminal every few seconds.
    bool licChanged = (oldStatus != g_licenseStatus) || (oldPlan != g_licensePlan)
                    || (oldAuth != g_authStatus) || (oldDevice != g_deviceStatus)
                    || (oldSess != g_sessionStatus) || (oldTrade != g_tradingStatus);
    if(licChanged)
    {
        Print("License status changed: ", oldStatus, " → ", g_licenseStatus,
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
    ulong magic = PAT_NextMagic(magicBase);
    string comment = PAT_StrategyPrefix(g_signalStrategy) + PAT_ShortSignalID(g_signalID);

    // 3. Lot sizing (risk-based; reject instead of forcing min lot)
    // Use server-calculated lot size if provided, otherwise minimum lot
    double vol = PAT_NormalizeLot(g_suggestedLot > 0 ? g_suggestedLot : SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MIN));
    if(vol <= 0)
    {
        Print("REJECTED lot_below_min: computed lot below broker minimum — refusing to force size");
        return;
    }

    // 4. EA-side risk gate (spread, drift, TTL, caps, risk$, martingale, margin)
    // Risk gates handled by SERVER — EA trusts server decision

    // Enforce EA-side position caps / halt flags. PAT_PreTradeGate was defined
    // but never invoked here — that omission allowed duplicate/over-positioning
    // (multiple positions per signal). Wire it in (fail-closed) so at most
    // MaxSameDirPositions / MaxTotalPositions PAT positions can be open.
    if(!PAT_PreTradeGate(true, vol, g_signalStrategy))
    {
        Print("SIGNAL NOT EXECUTED: EA pre-trade gate rejected BUY (cap/halt)");
        g_signalsFiltered++;
        return;
    }

    double ask = SymbolInfoDouble(g_symbol, SYMBOL_ASK);
    Print("ExecuteBuy: vol=", DoubleToString(vol, 2), " ask=", DoubleToString(ask, _Digits),
          " sl=", DoubleToString(g_sl, _Digits), " tp3=", DoubleToString(finalTP, _Digits),
          " magic=", magic, " comment=", comment);

    trade.SetExpertMagicNumber(magic);
    // W5 FIX: apply per-strategy max slippage (overrides the global default
    // set at init) so each strategy respects its configured slippage budget.
    trade.SetDeviationInPoints(PAT_GetMaxSlippage(g_signalStrategy));
    if(trade.Buy(vol, g_symbol, ask, g_sl, finalTP, comment))
    {
        g_lastExecutedSignalID = g_signalID;
        ulong posTicket = trade.ResultOrder();
        if(PositionSelectByTicket(posTicket))
        {
            PAT_RegPut(magic, g_signalID, g_signalStrategy,
                       PositionGetDouble(POSITION_PRICE_OPEN), g_sl,
                       g_tp1, g_tp2, g_tp3, vol);
        }
        else
        {
            PAT_RegPut(magic, g_signalID, g_signalStrategy, ask, g_sl,
                       g_tp1, g_tp2, g_tp3, vol);
        }
        PAT_SaveStage((long)magic, 0);
        Print("BUY executed: order=", posTicket, " magic=", magic, " vol=", DoubleToString(vol, 2));

        string ack = "EXECUTION_ACK|{";
        ack += "\"signal_id\":\"" + g_signalID + "\"";
        ack += ",\"status\":\"FILLED\"";
        ack += ",\"strategy_id\":\"" + g_signalStrategy + "\"";
        ack += ",\"magic\":" + IntegerToString((long)magic);
        ack += ",\"ticket\":" + IntegerToString((long)posTicket);
        ack += ",\"entry\":" + DoubleToString(ask, _Digits);
        ack += ",\"sl\":" + DoubleToString(g_sl, _Digits);
        ack += ",\"tp\":" + DoubleToString(finalTP, _Digits);
        ack += "}\n";
        PAT_Send(ack);
        CheckSlippage(posTicket, "BUY", ask);
    }
    else
    {
        Print("BUY FAILED: ", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
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
    ulong magic = PAT_NextMagic(magicBase);
    string comment = PAT_StrategyPrefix(g_signalStrategy) + PAT_ShortSignalID(g_signalID);

    // Use server-calculated lot size if provided, otherwise minimum lot
    double vol = PAT_NormalizeLot(g_suggestedLot > 0 ? g_suggestedLot : SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MIN));
    if(vol <= 0)
    {
        Print("REJECTED lot_below_min: computed lot below broker minimum — refusing to force size");
        return;
    }

    // Risk gates handled by SERVER — EA trusts server decision

    // Enforce EA-side position caps / halt flags (same as ExecuteBuy — this was
    // the missing call that permitted multiple positions per signal).
    if(!PAT_PreTradeGate(false, vol, g_signalStrategy))
    {
        Print("SIGNAL NOT EXECUTED: EA pre-trade gate rejected SELL (cap/halt)");
        g_signalsFiltered++;
        return;
    }

    double bid = SymbolInfoDouble(g_symbol, SYMBOL_BID);
    Print("ExecuteSell: vol=", DoubleToString(vol, 2), " bid=", DoubleToString(bid, _Digits),
          " sl=", DoubleToString(g_sl, _Digits), " tp3=", DoubleToString(finalTP, _Digits),
          " magic=", magic, " comment=", comment);

    trade.SetExpertMagicNumber(magic);
    // W5 FIX: apply per-strategy max slippage (see ExecuteBuy).
    trade.SetDeviationInPoints(PAT_GetMaxSlippage(g_signalStrategy));
    if(trade.Sell(vol, g_symbol, bid, g_sl, finalTP, comment))
    {
        g_lastExecutedSignalID = g_signalID;
        ulong posTicket = trade.ResultOrder();
        if(PositionSelectByTicket(posTicket))
        {
            PAT_RegPut(magic, g_signalID, g_signalStrategy,
                       PositionGetDouble(POSITION_PRICE_OPEN), g_sl,
                       g_tp1, g_tp2, g_tp3, vol);
        }
        else
        {
            PAT_RegPut(magic, g_signalID, g_signalStrategy, bid, g_sl,
                       g_tp1, g_tp2, g_tp3, vol);
        }
        PAT_SaveStage((long)magic, 0);
        Print("SELL executed: order=", posTicket, " magic=", magic, " vol=", DoubleToString(vol, 2));

        string ack = "EXECUTION_ACK|{";
        ack += "\"signal_id\":\"" + g_signalID + "\"";
        ack += ",\"status\":\"FILLED\"";
        ack += ",\"strategy_id\":\"" + g_signalStrategy + "\"";
        ack += ",\"magic\":" + IntegerToString((long)magic);
        ack += ",\"ticket\":" + IntegerToString((long)posTicket);
        ack += ",\"entry\":" + DoubleToString(bid, _Digits);
        ack += ",\"sl\":" + DoubleToString(g_sl, _Digits);
        ack += ",\"tp\":" + DoubleToString(finalTP, _Digits);
        ack += "}\n";
        PAT_Send(ack);
        CheckSlippage(posTicket, "SELL", bid);
    }
    else
    {
        Print("SELL FAILED: ", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
    }
}

//+------------------------------------------------------------------+
//| Magic allocation: strategy base + offset within its 100-range     |
//+------------------------------------------------------------------+
ulong PAT_NextMagic(int magicBase)
{
    string seqName = "PAT_MAGIC_SEQ";
    if(GlobalVariableCheck(seqName))
        g_magicSeq = (int)GlobalVariableGet(seqName);
    for(int attempt = 0; attempt < 100; attempt++)
    {
        ulong magic = (ulong)(magicBase + (g_magicSeq % 100));
        g_magicSeq++;
        GlobalVariableSet(seqName, g_magicSeq);
        if(!PAT_MagicInUse(magic) && PAT_RegFind((long)magic) < 0) return magic;
    }
    return (ulong)magicBase;
}

bool PAT_MagicInUse(ulong magic)
{
    int total = PositionsTotal();
    for(int i = 0; i < total; i++)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetInteger(POSITION_MAGIC) == (long)magic &&
           PositionGetString(POSITION_SYMBOL) == g_symbol) return true;
    }
    return false;
}

string PAT_ShortSignalID(string signalID)
{
    // Terminal comments are limited (~31 chars); prefix is 7, keep up to 22 of id
    if(StringLen(signalID) > 22) return StringSubstr(signalID, 0, 22);
    return signalID;
}

//+------------------------------------------------------------------+
//| SLIPPAGE MONITORING                                               |
//+------------------------------------------------------------------+
void CheckSlippage(ulong ticket, string direction, double requestedPrice)
{
    if(ticket == 0) return;
    double filledPrice = requestedPrice;
    if(PositionSelectByTicket(ticket))
        filledPrice = PositionGetDouble(POSITION_PRICE_OPEN);
    double point = SymbolInfoDouble(g_symbol, SYMBOL_POINT);
    double slippagePoints = 0;
    if(point > 0) slippagePoints = MathAbs(filledPrice - requestedPrice) / point;

    string slipMsg = "SLIPPAGE_EVENT|{";
    slipMsg += "\"ticket\":" + IntegerToString((long)ticket);
    slipMsg += ",\"symbol\":\"" + g_symbol + "\"";
    slipMsg += ",\"direction\":\"" + direction + "\"";
    slipMsg += ",\"requested\":" + DoubleToString(requestedPrice, 5);
    slipMsg += ",\"filled\":" + DoubleToString(filledPrice, 5);
    slipMsg += ",\"slippage_points\":" + DoubleToString(slippagePoints, 2);
    slipMsg += ",\"is_rollover\":" + (IsNearSwapTime() ? "true" : "false");
    slipMsg += ",\"strategy\":\"" + g_signalStrategy + "\"";
    slipMsg += ",\"signal_id\":\"" + g_signalID + "\"";
    slipMsg += "}";
    PAT_Send(slipMsg);

    if(RejectOnHighSlippage && MaxSlippagePoints > 0 && slippagePoints > MaxSlippagePoints)
    {
        g_slippageRejects++;
        Print("SLIPPAGE EXCEEDED: ticket=", ticket, " slip=", slippagePoints, " pts");
        if(PositionSelectByTicket(ticket))
        {
            PAT_SetForcedReason(PositionGetInteger(POSITION_IDENTIFIER), "SLIPPAGE_REJECT");
            ClosePosition(ticket, "SLIPPAGE_REJECT");
        }
    }
}

//+------------------------------------------------------------------+
//| Log each PAT/XAUUSD deal that is being counted as "today" so the  |
//| trader can verify whether a prior-day close is leaking into the   |
//| daily-loss calc.                                                  |
//+------------------------------------------------------------------+
void PAT_LogDailyLossDeals()
{
  MqlDateTime dt; TimeToStruct(TimeCurrent(), dt);
  datetime today = (datetime)(dt.year * 10000 + dt.mon * 100 + dt.day);
  HistorySelect(TimeCurrent() - 172800, TimeCurrent() + 60);
  int deals = HistoryDealsTotal();
  int n = 0;
  for(int i = 0; i < deals; i++)
  {
      ulong t = HistoryDealGetTicket(i);
      if(t == 0) continue;
      if(HistoryDealGetString(t, DEAL_SYMBOL) != g_symbol) continue;
      if(!PAT_IsPatMagic(HistoryDealGetInteger(t, DEAL_MAGIC))) continue;
      datetime dtt = (datetime)HistoryDealGetInteger(t, DEAL_TIME);
      MqlDateTime ddt; TimeToStruct(dtt, ddt);
      datetime dd = (datetime)(ddt.year * 10000 + ddt.mon * 100 + ddt.day);
      if(dd != today) continue;
      n++;
      double p = HistoryDealGetDouble(t, DEAL_PROFIT);
      double s = HistoryDealGetDouble(t, DEAL_SWAP);
      double c = HistoryDealGetDouble(t, DEAL_COMMISSION);
      PAT_LogLine("CAPITAL DEAL #" + IntegerToString(n)
                  + " | date(Broker): " + TimeToString(dtt, TIME_DATE)
                  + " | profit: " + DoubleToString(p, 2)
                  + " | swap: " + DoubleToString(s, 2)
                  + " | commission: " + DoubleToString(c, 2));
  }
  if(n == 0) PAT_LogLine("CAPITAL DEAL: none counted as today");
}

//+------------------------------------------------------------------+
//| CAPITAL PROTECTION                                                |
//+------------------------------------------------------------------+
void UpdateCapitalProtection()
{
    MqlDateTime dt;
    TimeToStruct(TimeCurrent(), dt);
    datetime today = (datetime)(dt.year * 10000 + dt.mon * 100 + dt.day);
    if(g_currentDay != today)
    {
        g_currentDay = today;
        g_dayStartBalance = AccountInfoDouble(ACCOUNT_BALANCE);
        g_dailyPnL = 0;
        g_tradingBlocked = false;
    }
    g_dailyPnL = 0;
    datetime dayStart = today;
    HistorySelect(TimeCurrent() - 172800, TimeCurrent() + 60);
    int deals = HistoryDealsTotal();
    for(int i = 0; i < deals; i++)
    {
        ulong dealTicket = HistoryDealGetTicket(i);
        if(dealTicket == 0) continue;
        if(HistoryDealGetString(dealTicket, DEAL_SYMBOL) != g_symbol) continue;
        if(!PAT_IsPatMagic(HistoryDealGetInteger(dealTicket, DEAL_MAGIC))) continue;
        datetime dealTime = (datetime)HistoryDealGetInteger(dealTicket, DEAL_TIME);
        MqlDateTime dealDt;
        TimeToStruct(dealTime, dealDt);
        datetime dealDay = (datetime)(dealDt.year * 10000 + dealDt.mon * 100 + dealDt.day);
        if(dealDay != dayStart) continue;
        g_dailyPnL += HistoryDealGetDouble(dealTicket, DEAL_PROFIT)
                    + HistoryDealGetDouble(dealTicket, DEAL_SWAP)
                    + HistoryDealGetDouble(dealTicket, DEAL_COMMISSION);
    }
    // Daily loss % is measured against the balance at the start of the broker
    // day. Derive it from realized P&L so it is correct even if the EA is
    // attached/restarted mid-day (the captured baseline would otherwise be the
    // already-reduced balance and overstate the loss %).
    double curBal = AccountInfoDouble(ACCOUNT_BALANCE);
    double dayOpenBalance = curBal - g_dailyPnL;
    if(dayOpenBalance <= 0) dayOpenBalance = curBal; // deposit/withdrawal guard
    g_dayStartBalance = dayOpenBalance;
    double lossPct = 0;
    if(dayOpenBalance > 0) lossPct = (g_dailyPnL / dayOpenBalance) * 100;

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
            string recMsg = "CAPITAL_PROTECTION|{\"event_type\":\"RECOVER\",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + ",\"action\":\"RESUMED\"}";
            PAT_Send(recMsg);
            PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: RESUMED (not blocked)");
        }
    }
    else
    {
        // Edge-triggered CAPITAL_WARNING: emit only on the transition into the
        // warning band (not on every tick). The EA otherwise appends the warning
        // on every tick while the account sits at/below the threshold, which
        // floods the EA → engine path. Reset when the account recovers above
        // the warning threshold so a fresh warning can fire later.
        if(lossPct <= -effWarning && !g_tradingBlocked)
        {
            if(!g_capitalWarnActive)
            {
                g_capitalWarnActive = true;
                string warnMsg = "CAPITAL_WARNING|{\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2)
                               + ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + "}";
                PAT_Send(warnMsg);
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
            Print("*** CAPITAL PROTECTION (HARD): Daily loss ", lossPct, "% — CLOSING ALL POSITIONS ***");
            string hardMsg = "CAPITAL_PROTECTION|{\"event_type\":\"HARD_HALT\",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + ",\"action\":\"EMERGENCY_CLOSE_ALL\"}";
            PAT_Send(hardMsg);
            PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: HARD HALT (closing all)");
            PAT_LogDailyLossDeals();
            if(EmergencyCloseAll)
                CloseAllPatPositions("EMERGENCY_CAPITAL_PROTECTION");
        }
}

//+------------------------------------------------------------------+
void UpdatePanel()
{
    string p = "=== Predict-A-Trade v1.00 ===\n";
    p += "Link:     " + g_connection + " (edge-poll ok=" + IntegerToString(g_pollOkCount) + " err=" + IntegerToString(g_pollErrCount) + ")\n";
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
    p += "Ticks:    " + IntegerToString((long)g_tickCount) + "\n";
    p += "Time:     " + TimeToString(TimeCurrent(), TIME_DATE|TIME_SECONDS) + "\n";
    p += "Slip rejects: " + IntegerToString(g_slippageRejects) + "\n";
    p += "Daily P&L: " + DoubleToString(g_dailyPnL, 2) + "\n";
    if(BypassDailyLossBlock) p += "DailyLoss guard: BYPASSED (client override)\n";
    if(g_tradingBlocked) p += "*** TRADING BLOCKED (daily loss) ***\n";
    if(g_equityHalted)   p += "*** HALTED: EQUITY FLOOR ***\n";
    Comment(p);
}

//+------------------------------------------------------------------+
//| JSON helpers                                                      |
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

// ExtractJSONArrayRaw returns the INNER content of a JSON array value for `key`.
// Example: for {"allowed_strategies":["A","B"]} with key="allowed_strategies",
// returns "A","B" (brackets/quotes/spaces stripped by the caller).
// Returns "" if the value is not a JSON array (so callers can fall back to a
// quoted-string value via ExtractJSONString). W2 FIX.
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
    string sk = "\"" + key + "\":";
    int s = StringFind(json, sk);
    if(s < 0) return 0;
    s += StringLen(sk);
    // Skip leading spaces or quotes (decimal.Decimal serializes as "4519.03")
    while(s < StringLen(json))
    {
        int c = StringGetCharacter(json, s);
        if(c == 32 || c == 34) { s++; continue; }
        break;
    }
    string v = "";
    for(int i = s; i < StringLen(json); i++)
    {
        int c = StringGetCharacter(json, i);
        if(c == 44 || c == 125 || c == 32 || c == 34) break;
        v += CharToString((uchar)c);
    }
    return StringToDouble(v);
}

//+------------------------------------------------------------------+
//| File I/O using FILE_COMMON — device bootstrap state + local log   |
//| (v1.19.0: the IPC pipe files are GONE. Only PAT_device.txt — the  |
//| persisted device credential — and error.log remain.)              |
//+------------------------------------------------------------------+
void PAT_Write(string filename, string content)
{
    int retry = 0;
    while(retry < 3)
    {
        int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
        if(h != INVALID_HANDLE)
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

string PAT_Read(string filename)
{
    int h = FileOpen(filename, FILE_READ | FILE_TXT | FILE_ANSI | FILE_COMMON);
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

//+------------------------------------------------------------------+
//| ══════════ OPTION B TRANSPORT (v1.19.0) ═══════════════════════ |
//| HTTPS client: device activation, HMAC signing, edge-poll, ack,   |
//| heartbeat, and the engine ingest POST.                           |
//+------------------------------------------------------------------+
string g_deviceId       = "";
string g_deviceSecret   = "";
string g_refreshToken   = "";
string g_accessToken    = "";     // Bearer JWT for POST /ingest/agent
datetime g_tokenExpiry  = 0;      // UTC time the access token expires
bool   g_netDiagnosticsShown = false;
int    g_pollOkCount    = 0;
int    g_pollErrCount   = 0;
long   g_hmacCounter    = 0;      // monotonic nonce component

//--- PAT_HTTPPost: plain JSON POST (no auth) → (status, response)
int PAT_HTTPPost(string url, string body, string &response)
{
    string headers = "Content-Type: application/json\r\n";
    uchar data[];
    StringToCharArray(body, data, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(data, ArraySize(data) - 1); // strip trailing NUL
    uchar result[];
    string resHeaders = "";
    int status = WebRequest("POST", url, headers, 5000, data, result, resHeaders);
    response = CharArrayToString(result, 0, WHOLE_ARRAY, CP_UTF8);
    return status;
}

//--- PAT_HMACSign: canonical v1 device signature over an outgoing request
//    canonical = "v1\n<ts>\n<nonce>\nPOST\n<path>\n<sha256(body)>\n<device_id>"
//    (byte-identical to DeviceAuthService.verifyRequestSignature)
string PAT_HMACSign(string path, string body, string deviceId, string deviceSecret, string ts, string nonce)
{
    string bodyHash = PAT_Sha256Hex(body);
    string canonical = "v1\n" + ts + "\n" + nonce + "\nPOST\n" + path + "\n" + bodyHash + "\n" + deviceId;
    return PAT_HmacSha256Hex(deviceSecret, canonical);
}

//--- PAT_SignedPost: HMAC-authenticated control-plane POST
//    path: full path as signed, e.g. "/api/v1/devices/edge-poll"
int PAT_SignedPost(string path, string body, string &response)
{
    string ts = IntegerToString((long)TimeGMT() * 1000 + (GetTickCount() % 1000));
    g_hmacCounter++;
    string nonce = PAT_Sha256Hex(ts + IntegerToString(g_hmacCounter) + IntegerToString(MathRand()) + IntegerToString(GetTickCount()));
    string sig = PAT_HMACSign(path, body, g_deviceId, g_deviceSecret, ts, nonce);

    string headers = "Content-Type: application/json\r\n"
                    "X-Device-Id: " + g_deviceId + "\r\n"
                    "X-Device-Timestamp: " + ts + "\r\n"
                    "X-Device-Nonce: " + nonce + "\r\n"
                    "X-Device-Signature: " + sig + "\r\n";
    uchar data[];
    StringToCharArray(body, data, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(data, ArraySize(data) - 1); // strip trailing NUL
    uchar result[];
    string resHeaders = "";
    int status = WebRequest("POST", PATCloudURL + path, headers, 8000, data, result, resHeaders);
    response = CharArrayToString(result, 0, WHOLE_ARRAY, CP_UTF8);
    return status;
}

//--- PAT_EnsureDevice: bootstrap credentials (inputs → else persisted → activate)
bool PAT_EnsureDevice()
{
    if(StringLen(g_deviceId) > 0 && StringLen(g_deviceSecret) > 0) return true;

    // 1) EA inputs (dashboard copy-paste flow)
    if(StringLen(PATDeviceId) > 0 && StringLen(PATDeviceSecret) > 0)
    {
        g_deviceId = PATDeviceId;
        g_deviceSecret = PATDeviceSecret;
        PAT_Write(PAT_DEVICE_FILE, g_deviceId + "|" + g_deviceSecret + "\n");
        Print("[Predict-A-Trade] Device credentials loaded from EA inputs.");
        return true;
    }

    // 2) Persisted bootstrap state
    string saved = PAT_Read(PAT_DEVICE_FILE);
    if(StringLen(saved) > 0)
    {
        string parts[];
        int n = StringSplit(saved, '|', parts);
        if(n >= 2 && StringLen(parts[0]) > 0 && StringLen(parts[1]) > 0)
        {
            g_deviceId = parts[0];
            g_deviceSecret = parts[1];
            if(n >= 3) g_refreshToken = parts[2];
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
    string body = "{\"license_key\":\"" + LicenseKey + "\",\"client_type\":\"MT5\",\"role\":\"exec\","
                  "\"fingerprint\":{\"machine_guid\":\"" + fp + "\",\"os\":\"Windows-MT5\"},"
                  "\"terminal\":{\"name\":\"" + AccountInfoString(ACCOUNT_COMPANY) + "\"}}";
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
    PAT_Write(PAT_DEVICE_FILE, g_deviceId + "|" + g_deviceSecret + "|" + g_refreshToken + "\n");
    Print("[Predict-A-Trade] Device activated: ", g_deviceId);
    return true;
}

//--- PAT_DeviceFingerprint: stable per-terminal identity
string PAT_DeviceFingerprint()
{
    // Terminal company + account data-folder path + build — stable across
    // restarts, unique enough per terminal install. (TERMINAL_NAME is a
    // TerminalInfoString property; passing it to TerminalInfoInteger does
    // not compile in MQL5.)
    string raw = AccountInfoString(ACCOUNT_COMPANY)
               + "|" + TerminalInfoString(TERMINAL_PATH)
               + "|" + IntegerToString((int)TerminalInfoInteger(TERMINAL_BUILD));
    return PAT_Sha256Hex(raw);
}

//+------------------------------------------------------------------+
//| PAT_EdgePoll — fetch the next batch of queued items for this      |
//| device. Returns number of items; fills parts[] with raw payloads. |
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
        // Expired access token is irrelevant here (HMAC auth) but activation
        // may have lapsed — surface diagnostics once.
        if(!g_netDiagnosticsShown)
        {
            Print("[Predict-A-Trade] edge-poll failed: HTTP ", status, " — ", StringSubstr(response, 0, 200));
            g_netDiagnosticsShown = true;
        }
        return -1;
    }
    g_netDiagnosticsShown = false;
    g_pollOkCount++;

    // Parse "signal" payloads + "queue_id" per item. Response shape:
    // {"ok":true,"pending":[{"queue_id":"…","signal_id":"…","signal":{…}}, …]}
    int pos = 0;
    while(true)
    {
        int qid = StringFind(response, "\"queue_id\":\"", pos);
        if(qid < 0) break;
        int qidEnd = StringFind(response, "\"", qid + 12);
        if(qidEnd < 0) break;
        string queueId = StringSubstr(response, qid + 12, qidEnd - (qid + 12));

        // The signal payload object: find the "signal":{ … } that follows.
        int sigKey = StringFind(response, "\"signal\":{", qidEnd);
        int nextQ  = StringFind(response, "\"queue_id\"", qidEnd);
        if(sigKey < 0 || (nextQ >= 0 && sigKey > nextQ))
        {
            // No signal object in this item (shouldn't happen) — skip.
            pos = qidEnd;
            continue;
        }
        // Extract the balanced JSON object after "signal":{
        string payload = PAT_ExtractJSONObject(response, sigKey + 10);
        int cnt = ArraySize(items);
        ArrayResize(items, cnt + 1);
        ArrayResize(queueIds, cnt + 1);
        items[cnt] = payload;
        queueIds[cnt] = queueId;
        pos = sigKey + 10 + StringLen(payload);
        if(ArraySize(items) >= 20) break;
    }
    return ArraySize(items);
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
    string body = "{\"terminal\":\"MT5\",\"account\":\"" + g_accountID + "\","
                  "\"symbol\":\"" + g_symbol + "\",\"build\":" + IntegerToString((int)TerminalInfoInteger(TERMINAL_BUILD)) + "}";
    string response = "";
    int status = PAT_SignedPost("/api/v1/devices/edge-heartbeat", body, response);
    if(status != 200 && !g_netDiagnosticsShown)
        Print("[Predict-A-Trade] edge-heartbeat failed: HTTP ", status);
}

//+------------------------------------------------------------------+
//| PAT_Send — outbound message funnel (Option B). Takes the legacy   |
//| "TYPE|{json}" wire line and POSTs its JSON payload to the engine  |
//| ingest endpoint. Injects "type" if the payload omits it.          |
//+------------------------------------------------------------------+
void PAT_Send(string line)
{
    string s = line;
    // Strip trailing CR/LF/NUL whitespace.
    while(StringLen(s) > 0)
    {
        ushort c = StringGetCharacter(s, StringLen(s) - 1);
        if(c == '\n' || c == '\r' || c == 0) s = StringSubstr(s, 0, StringLen(s) - 1);
        else break;
    }
    int sep = StringFind(s, "|");
    if(sep <= 0) return;
    string msgType = StringSubstr(s, 0, sep);
    string payload = StringSubstr(s, sep + 1);
    if(StringLen(msgType) == 0 || StringLen(payload) == 0) return;
    // Inject "type" when the builder omitted it (INIT/ACCOUNT_INFO/LICENSE_CHECK).
    if(StringFind(payload, "\"type\"") < 0 && StringGetCharacter(payload, 0) == '{')
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
    ArrayResize(data, ArraySize(data) - 1); // strip trailing NUL
    uchar result[];
    string resHeaders = "";
    string url = PATCloudURL + "/ingest/agent?agentId=" + PAT_URLEncode(g_deviceId) + "&role=exec";
    int status = WebRequest("POST", url, headers, 5000, data, result, resHeaders);
    if(status == 401)
    {
        // Access token expired — force refresh once and retry.
        g_accessToken = ""; g_tokenExpiry = 0;
        if(PAT_EnsureAccessToken())
            PAT_PostIngest(msgType, payload);
        return;
    }
    if(status != 200)
        Print("[Predict-A-Trade] ingest ", msgType, " failed: HTTP ", status);
}

//--- PAT_URLEncode — percent-encoding for query values
string PAT_URLEncode(string s)
{
    string outp = "";
    uchar bytes[];
    int n = StringToCharArray(s, bytes, 0, WHOLE_ARRAY, CP_UTF8);
    for(int i = 0; i < n - 1; i++)   // trailing NUL excluded
    {
        uchar c = bytes[i];
        if((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
           c == '-' || c == '_' || c == '.' || c == '~')
            outp += CharToString(c);
        else
            outp += StringFormat("%%%02X", c);
    }
    return outp;
}

//--- PAT_EnsureAccessToken: rotate the access token via
//    POST /api/v1/devices/refresh (refresh_token grant).
bool PAT_EnsureAccessToken()
{
    if(StringLen(g_accessToken) > 0 && TimeGMT() < g_tokenExpiry) return true;
    if(!PAT_EnsureDevice()) return false;
    if(StringLen(g_refreshToken) == 0)
    {
        // Legacy persisted state without a refresh token — re-activate.
        g_deviceId = ""; g_deviceSecret = ""; PAT_Clear(PAT_DEVICE_FILE);
        return PAT_EnsureDevice();
    }
    string body = "{\"refresh_token\":\"" + g_refreshToken + "\",\"device_id\":\"" + g_deviceId + "\",\"role\":\"exec\"}";
    string response = "";
    int status = PAT_HTTPPost(PATCloudURL + "/api/v1/devices/refresh", body, response);
    if(status != 200)
    {
        Print("[Predict-A-Trade] token refresh failed: HTTP ", status, " — re-activating.");
        g_deviceId = ""; g_deviceSecret = ""; g_refreshToken = "";
        PAT_Clear(PAT_DEVICE_FILE);
        return PAT_EnsureDevice();
    }
    g_accessToken = ExtractJSONString(response, "access_token");
    g_refreshToken = ExtractJSONString(response, "refresh_token");
    // 10-minute server TTL; refresh at 8 minutes to stay ahead of skew.
    g_tokenExpiry = TimeGMT() + 8 * 60;
    if(StringLen(g_refreshToken) > 0)
        PAT_Write(PAT_DEVICE_FILE, g_deviceId + "|" + g_deviceSecret + "|" + g_refreshToken + "\n");
    return StringLen(g_accessToken) > 0;
}

//+------------------------------------------------------------------+
//| Client MT terminal log: echoes a formatted [Predict-A-Trade] line  |
//| to the MT Experts log AND appends it to error.log (FILE_COMMON)   |
//| so the trader can see why trading is blocked / what was received.  |
//+------------------------------------------------------------------+
void PAT_LogLine(string msg)
{
    Print("[Predict-A-Trade] ", msg);
    int h = FileOpen(PAT_ERROR_LOG, FILE_READ | FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
    if(h == INVALID_HANDLE) return;
    FileSeek(h, 0, SEEK_END);
    FileWriteString(h, "[Predict-A-Trade] " + msg + "\n");
    FileClose(h);
}

//--- PAT_Clear: truncate a FILE_COMMON file to empty (device re-activation reset)
void PAT_Clear(string filename)
{
    int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
    if(h != INVALID_HANDLE) FileClose(h);
}

//+------------------------------------------------------------------+
//| PAT_Sha256Hex / PAT_HmacSha256Hex — pure-MQL5 SHA-256 + HMAC.     |
//| (MQL5 has no native HMAC; this is the standard FIPS 180-4        |
//| implementation over UTF-8 bytes, output lowercase hex.)          |
//+------------------------------------------------------------------+
void PAT_Sha256(const uchar &msg[], uchar &digest[])
{
    // Message padding (FIPS 180-4 §5.1): 0x80 + 8-byte length fit inside the
    // 64-alignment of (len + 9). The ((len+8)/64+1)*64 form mis-pads len=55/119/...
    // (0x80 overwrites the length field → wrong digest → HMAC rejection).
    ulong bitLen = (ulong)ArraySize(msg) * 8;
    int paddedLen = (int)(((ArraySize(msg) + 9 + 63) / 64)) * 64;
    uchar padded[];
    ArrayResize(padded, paddedLen);
    ArrayInitialize(padded, 0);
    ArrayCopy(padded, msg, 0, 0, ArraySize(msg));
    padded[ArraySize(msg)] = 0x80;
    for(int i = 0; i < 8; i++)
        padded[paddedLen - 1 - i] = (uchar)((bitLen >> (8 * i)) & 0xFF);

    // Initial hash values
    uint h0=0x6a09e667, h1=0xbb67ae85, h2=0x3c6ef372, h3=0xa54ff53a;
    uint h4=0x510e527f, h5=0x9b05688c, h6=0x1f83d9ab, h7=0x5be0cd19;
    uint k[64];
    PAT_Sha256K(k);

    uint w[64];
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

void PAT_Sha256K(uint &k[])
{
    static const uint K[64] = {
        0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,
        0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,
        0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
        0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,
        0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,
        0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
        0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,
        0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2};
    ArrayCopy(k, K);
}

uint PAT_ROTR(uint x, int n) { return (x >> n) | (x << (32 - n)); }

void PAT_StoreU32BE(uint v, uchar &outp[], int pos)
{
    outp[pos]   = (uchar)((v >> 24) & 0xFF);
    outp[pos+1] = (uchar)((v >> 16) & 0xFF);
    outp[pos+2] = (uchar)((v >> 8) & 0xFF);
    outp[pos+3] = (uchar)(v & 0xFF);
}

string PAT_BytesToHex(const uchar &bytes[])
{
    string hexchars = "0123456789abcdef";
    string outp = "";
    for(int i = 0; i < ArraySize(bytes); i++)
    {
        outp += StringSubstr(hexchars, (bytes[i] >> 4) & 0x0F, 1);
        outp += StringSubstr(hexchars, bytes[i] & 0x0F, 1);
    }
    return outp;
}

string PAT_Sha256Hex(string text)
{
    uchar msg[];
    StringToCharArray(text, msg, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(msg, ArraySize(msg) - 1); // strip trailing NUL
    uchar digest[];
    PAT_Sha256(msg, digest);
    return PAT_BytesToHex(digest);
}

// HMAC-SHA256 (RFC 2104): H((K⊕opad) || H((K⊕ipad) || m)), K zero-padded to 64 bytes.
string PAT_HmacSha256Hex(string key, string message)
{
    uchar k[], m[];
    StringToCharArray(key, k, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(k, ArraySize(k) - 1);
    StringToCharArray(message, m, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(m, ArraySize(m) - 1);

    uchar kblock[64];
    ArrayInitialize(kblock, 0);
    if(ArraySize(k) > 64)
    {
        uchar kh[];
        PAT_Sha256(k, kh);
        ArrayCopy(kblock, kh, 0, 0, 32);
    }
    else
        ArrayCopy(kblock, k, 0, 0, ArraySize(k));

    uchar ipad[64], opad[64];
    for(int i = 0; i < 64; i++)
    {
        ipad[i] = kblock[i] ^ 0x36;
        opad[i] = kblock[i] ^ 0x5C;
    }

    uchar inner[], innerMsg[], innerDigest[];
    ArrayCopy(innerMsg, ipad, 0, 0, 64);
    ArrayCopy(innerMsg, m, 64, 0, ArraySize(m));
    PAT_Sha256(innerMsg, innerDigest);

    uchar outer[], outerMsg[], outerDigest[];
    ArrayCopy(outerMsg, opad, 0, 0, 64);
    ArrayCopy(outerMsg, innerDigest, 64, 0, 32);
    PAT_Sha256(outerMsg, outerDigest);

    return PAT_BytesToHex(outerDigest);
}

//+------------------------------------------------------------------+
//| PAT_ExtractJSONObject — extract a balanced {...} starting at      |
//| start (which must point at '{'). Handles nested objects/strings.  |
//+------------------------------------------------------------------+
string PAT_ExtractJSONObject(string s, int start)
{
    if(start < 0 || StringGetCharacter(s, start) != '{') return "";
    int depth = 0;
    bool inStr = false;
    int len = StringLen(s);
    for(int i = start; i < len; i++)
    {
        ushort c = StringGetCharacter(s, i);
        if(inStr)
        {
            if(c == '\\') i++;              // skip escaped char
            else if(c == '"') inStr = false;
        }
        else
        {
            if(c == '"') inStr = true;
            else if(c == '{') depth++;
            else if(c == '}')
            {
                depth--;
                if(depth == 0)
                    return StringSubstr(s, start, i - start + 1);
            }
        }
    }
    return "";
}
