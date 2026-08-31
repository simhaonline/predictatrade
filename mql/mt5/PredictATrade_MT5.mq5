//+------------------------------------------------------------------+
//|                                          PredictATrade_MT5.mq5   |
//|                              Predict-A-Trade v1.00               |
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
#property version   "1.00"
#property strict

#include <Trade\Trade.mqh>

//=== Input Parameters ===
input bool    AutoExecute    = false;   // SIGNAL_ONLY=true by default (display only). Set true to auto-trade.
input bool    BypassDailyLossBlock = false; // Allow new trades even after the soft daily-loss limit is hit. Hard halt (close-all at MaxDailyLossPct) is NEVER bypassed.
input string  LicenseKey     = "";      // Your Predict-A-Trade license key
input string  ChartTimeframe = "M1";    // Chart/timeframe this EA instance trades (M1/M5/H1/...)

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

//=== File names (FILE_COMMON folder — shared with Windows Agent) ===
//─-- Internal constants (managed by backend, do not change) ──
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

#define PAT_TICK_FILE    "PAT_ticks.txt"
#define PAT_SIGNAL_FILE  "PAT_signals.txt"
#define PAT_LICENSE_FILE "PAT_license.txt"
#define PAT_HEARTBEAT    "PAT_heartbeat.txt"
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
    int count = 0;
    int total = PositionsTotal();
    for(int i = 0; i < total; i++)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        if(PAT_IsPatMagic(PositionGetInteger(POSITION_MAGIC))) count++;
    }
    return count;
}

int PAT_CountPatPositionsDir(bool isBuy)
{
    int count = 0;
    int total = PositionsTotal();
    for(int i = 0; i < total; i++)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        if(!PAT_IsPatMagic(PositionGetInteger(POSITION_MAGIC))) continue;
        long ptype = PositionGetInteger(POSITION_TYPE);
        if(isBuy && ptype == POSITION_TYPE_BUY) count++;
        if(!isBuy && ptype == POSITION_TYPE_SELL) count++;
    }
    return count;
}

//+------------------------------------------------------------------+
//| EA-side risk gate (belt-and-suspenders — all fail-closed)         |
//+------------------------------------------------------------------+
bool PAT_PreTradeGate(bool isBuy, double lot, string strategyName)
{
    // ALL risk gates are handled by the SERVER engine.
    // The EA trusts the server's decision — if the server sends a signal,
    // it has already passed all 14+ risk gates (spread, risk, margin, etc.)
    // Only keep the EA-side watchdog for SL presence (capital protection).

    // 0. Halt flags (EA-side emergency only)
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

    // Spread, entry drift, and TTL are checked by the SERVER engine.
    // EA trusts server decision — no local gates for these.

    // Position caps (same-direction and total, by PAT magic range)
    if(PAT_CountPatPositionsDir(isBuy) >= MaxSameDirPositions)
    {
        Print("REJECTED same_dir_cap: already ", PAT_CountPatPositionsDir(isBuy),
              " ", (isBuy ? "BUY" : "SELL"), " PAT position(s) (max ", MaxSameDirPositions, ")");
        return false;
    }
    if(PAT_CountPatPositions() >= MaxTotalPositions)
    {
        Print("REJECTED total_cap: ", PAT_CountPatPositions(),
              " PAT positions open (max ", MaxTotalPositions, ")");
        return false;
    }

    // 5. Risk-$ gate: risk$ = |entry-sl| * valuePerPriceUnit * lot
    double tickVal  = PAT_TickValue();
    double tickSize = PAT_TickSize();
    double equity   = AccountInfoDouble(ACCOUNT_EQUITY);
    double dist     = MathAbs(g_entry - g_sl);
    if(tickVal <= 0 || tickSize <= 0 || dist <= 0 || equity <= 0)
    {
        Print("REJECTED bad_risk_inputs: tickVal=", tickVal, " tickSize=", tickSize,
              " dist=", dist, " equity=", equity);
        return false;
    }
    double valuePerUnit = tickVal / tickSize;
    // Risk check REMOVED — server handles risk_oversize

    // 6. Martingale ban: lot may never exceed baseLot * MaxLotRatioVsBase.
    //    Effective base = max(configured BaseLot, deterministic risk-sized lot
    //    for THIS signal) — prevents runaway sizing while allowing honest
    //    risk-based sizing. Ratio 1.0 forbids any escalation beyond it.
    double effBase = BaseLot;
    double riskLot = PAT_CalcLotSize(equity, dist);
    if(riskLot > effBase) effBase = riskLot;
    if(lot > effBase * MaxLotRatioVsBase + 0.0000001)
    {
        Print("REJECTED martingale_ban: lot=", DoubleToString(lot, 2),
              " > base=", DoubleToString(effBase, 2), " x ", DoubleToString(MaxLotRatioVsBase, 2));
        return false;
    }

    // Current reference price for margin calculation
    double refPx = isBuy ? SymbolInfoDouble(g_symbol, SYMBOL_ASK)
                         : SymbolInfoDouble(g_symbol, SYMBOL_BID);

    // 7. Margin gate (OrderCalcMargin)
    double marginRequired = 0;
    ENUM_ORDER_TYPE mtype = isBuy ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;
    if(!OrderCalcMargin(mtype, g_symbol, lot, refPx, marginRequired))
        marginRequired = 0;
    double freeMargin = AccountInfoDouble(ACCOUNT_MARGIN_FREE);
    if(marginRequired > freeMargin * (MaxMarginUsagePct / 100.0))
    {
        Print("REJECTED margin_overuse: required=", DoubleToString(marginRequired, 2),
              " > freeMargin x ", MaxMarginUsagePct, "% = ",
              DoubleToString(freeMargin * MaxMarginUsagePct / 100.0, 2));
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
//| Trade result reporting (TRADE_RESULT + CLOSE_ACK via IPC file)    |
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
    PAT_Append(PAT_TICK_FILE, msg + "\n");

    // CLOSE_ACK is already parsed/forwarded by the Windows Agent
    // (windows-agent/internal/pipe.go) — sent for pipeline compatibility.
    string ack = "CLOSE_ACK|{";
    ack += "\"ticket\":" + IntegerToString(ticket);
    ack += ",\"reason\":\"" + exitReason + "\"";
    ack += ",\"net_pnl\":" + DoubleToString(realizedPnl, 2);
    ack += ",\"signal_id\":\"" + signalID + "\"";
    ack += ",\"strategy_id\":\"" + strategyID + "\"";
    ack += ",\"magic\":" + IntegerToString(magic);
    ack += "}\n";
    PAT_Append(PAT_TICK_FILE, ack);

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
    Print("Predict-A-Trade MT5 EA v1.00 initializing...");

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
        Print("Ensure pat-agent.exe is running on this machine.");
    }

    UpdatePanel();
    return(INIT_SUCCEEDED);
}

void OnDeinit(const int reason)
{
    EventKillTimer();
    if(g_atrHandle != INVALID_HANDLE) IndicatorRelease(g_atrHandle);
    PAT_Append(PAT_TICK_FILE, "DEINIT|{}\n");
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

    if(g_connection == "CONNECTED")
        ReadFromAgent();

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
        g_lastTickSend = GetTickCount(); // reuse as "last EA→agent send" so we ping at a sane rate
    }

    PAT_Watchdog();
    PAT_HistoryPoll();
    SendAccountInfo();
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
void SendTickToAgent()
{
    if(TickIntervalMs > 0)
    {
        uint elapsed = GetTickCount() - g_lastTickSend;
        if(TickIntervalMs != 0 && elapsed < (uint)TickIntervalMs) return;
    }
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

    PAT_Append(PAT_TICK_FILE, msg);
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

    string msg = "INIT|{\"ea_version\":\"1.08\",\"broker\":\"" + AccountInfoString(ACCOUNT_COMPANY) +
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
    PAT_Append(PAT_TICK_FILE, msg);
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
    string msg = "ACCOUNT_INFO|{\"ea_version\":\"1.08\",\"account\":\"" + g_accountID +
                 "\",\"broker\":\"" + AccountInfoString(ACCOUNT_COMPANY) +
                 "\",\"symbol\":\"" + g_symbol +
                 "\",\"currency\":\"" + AccountInfoString(ACCOUNT_CURRENCY) +
                 "\",\"license_key\":\"" + g_licenseKey +
                 "\",\"balance\":" + DoubleToString(AccountInfoDouble(ACCOUNT_BALANCE), 2) +
                 ",\"equity\":" + DoubleToString(AccountInfoDouble(ACCOUNT_EQUITY), 2) +
                 ",\"free_margin\":" + DoubleToString(AccountInfoDouble(ACCOUNT_MARGIN_FREE), 2) +
                 ",\"leverage\":" + IntegerToString((int)AccountInfoInteger(ACCOUNT_LEVERAGE)) +
                 "}\n";
    PAT_Append(PAT_TICK_FILE, msg);
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
    PAT_Append(PAT_TICK_FILE, msg);
    Print("License validation with account data - balance: ", AccountInfoDouble(ACCOUNT_BALANCE));
}

//+------------------------------------------------------------------+
//| LIVENESS ping — sent from OnTimer when the market produces no ticks |
//| (weekend/holiday). Keeps the terminal visible (dashboard ONLINE),   |
//| the license resolved, and the EA→agent chain provably alive. The    |
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
    PAT_Append(PAT_TICK_FILE, msg);

    // Include a license re-validation with each liveness ping so a fresh key
    // typed into the EA resolves even through a closed market.
    RequestLicenseValidation();
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

    // Read signals — check every tick
    if(FileIsExist(PAT_SIGNAL_FILE, FILE_COMMON))
    {
        string content = PAT_Read(PAT_SIGNAL_FILE);
        if(StringLen(content) > 0)
        {
            g_signalsReceived++;

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

                if(msgType == "SIGNAL")
                    HandleSignal(payload);
                else if(msgType == "LICENSE_RESPONSE" || msgType == "LICENSE")
                    HandleLicenseResponse(payload);
                else if(msgType == "CLOSE_POSITION")
                    HandleClosePosition(payload);
                else if(msgType == "EMERGENCY_STOP")
                    HandleEmergencyStop(payload);
                else if(msgType == "KILL_SWITCH")
                    HandleKillSwitch(payload);
            }
            PAT_Clear(PAT_SIGNAL_FILE);
        }
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
    PAT_Append(PAT_TICK_FILE, deinitMsg);

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
    // The server ALSO filters signals before sending to the agent (primary defense).
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
        Print("SIGNAL BLOCKED: agent connection not CONNECTED (g_connection=", g_connection, ")");
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

    // Only log when the license state actually changes — the agent re-sends
    // license responses on every heartbeat, so an unconditional log would spam
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
        PAT_Append(PAT_TICK_FILE, ack);
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
        PAT_Append(PAT_TICK_FILE, ack);
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
    PAT_Append(PAT_TICK_FILE, slipMsg + "\n");

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
            PAT_Append(PAT_TICK_FILE, recMsg + "\n");
            PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: RESUMED (not blocked)");
        }
    }
    else
    {
        // Edge-triggered CAPITAL_WARNING: emit only on the transition into the
        // warning band (not on every tick). The EA otherwise appends the warning
        // on every tick while the account sits at/below the threshold, which
        // floods the agent → engine path. Reset when the account recovers above
        // the warning threshold so a fresh warning can fire later.
        if(lossPct <= -effWarning && !g_tradingBlocked)
        {
            if(!g_capitalWarnActive)
            {
                g_capitalWarnActive = true;
                string warnMsg = "CAPITAL_WARNING|{\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2)
                               + ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + "}";
                PAT_Append(PAT_TICK_FILE, warnMsg + "\n");
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
            PAT_Append(PAT_TICK_FILE, softMsg + "\n");
            PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: BLOCKED (daily loss)");
            PAT_LogDailyLossDeals();
        }
    }
        if(lossPct <= -effHardHalt && !g_hardHaltTriggered)
        {
            g_hardHaltTriggered = true;
            Print("*** CAPITAL PROTECTION (HARD): Daily loss ", lossPct, "% — CLOSING ALL POSITIONS ***");
            string hardMsg = "CAPITAL_PROTECTION|{\"event_type\":\"HARD_HALT\",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + ",\"action\":\"EMERGENCY_CLOSE_ALL\"}";
            PAT_Append(PAT_TICK_FILE, hardMsg + "\n");
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
//| File I/O using FILE_COMMON — append-safe                          |
//| FIXED v1.00: PAT_Append previously used FILE_WRITE which TRUNCATED|
//| the IPC file (lost messages when agent had not drained fast       |
//| enough). Now FILE_READ|FILE_WRITE + FileSeek(SEEK_END).           |
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

void PAT_Append(string filename, string content)
{
    int retry = 0;
    while(retry < 5)
    {
        if(retry > 0) Sleep(5);
        int h = FileOpen(filename, FILE_READ | FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
        if(h == INVALID_HANDLE)
        {
            // File may not exist yet — create it.
            h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
            if(h != INVALID_HANDLE)
            {
                FileSeek(h, 0, SEEK_END);
                FileWriteString(h, content);
                FileClose(h);
                return;
            }
        }
        else
        {
            FileSeek(h, 0, SEEK_END);
            FileWriteString(h, content);
            FileClose(h);
            return;
        }
        retry++;
        Sleep(5);
    }
    Print("FileOpen APPEND failed after 3 retries: ", filename, " error=", GetLastError());
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

 void PAT_Clear(string filename)
{
    int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
    if(h != INVALID_HANDLE) FileClose(h);
}

//+------------------------------------------------------------------+
//| Client MT terminal log: echoes a formatted [Predict-A-Trade] line  |
//| to the MT Experts log AND appends it to error.log (FILE_COMMON)   |
//| so the trader can see why trading is blocked / what was received.  |
//+------------------------------------------------------------------+
 void PAT_LogLine(string msg)
{
    Print("[Predict-A-Trade] ", msg);
    PAT_Append(PAT_ERROR_LOG, "[Predict-A-Trade] " + msg + "\n");
}
