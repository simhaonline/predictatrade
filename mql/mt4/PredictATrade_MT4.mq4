//+------------------------------------------------------------------+
//|                                          PredictATrade_MT4.mq4   |
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

// ─── Signal/Execution inputs ───
input bool    AutoExecute    = true;
input string  LicenseKey     = "";
input string  ChartTimeframe = "M1";    // Chart/timeframe this EA instance trades (M1/M5/H1/...)

// ─── Strategy/Direction filters ───
// Strategy selection is SERVER-CONTROLLED based on your license plan.
// Just enter your License Key — the server handles strategy filtering.


input bool    ExecuteCandidates  = false;  // Execute BUY_CANDIDATE/SELL_CANDIDATE as real trades


// ─── Execution Safety v1.00 (mql-fix.md — fail-closed) ───


// ─── Position Sizing ───

// ─── Constants ───
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
#define RiskPerTradePct 1.0
#define UseAutoLotSizing true

#define PAT_TICK_FILE   "PAT_ticks.txt"
#define PAT_SIGNAL_FILE "PAT_signals.txt"
#define PAT_LICENSE_FILE "PAT_license.txt"
#define PAT_HEARTBEAT   "PAT_heartbeat.txt"
// Client MT terminal log — formatted [Predict-A-Trade] lines written here (FILE_COMMON)
// and echoed to the MT Experts log so the trader can see status/signal activity.
#define PAT_ERROR_LOG   "error.log"

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
    int count = 0;
    int total = OrdersTotal();
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(PAT_IsPatMagic(OrderMagicNumber())) count++;
    }
    return count;
}

int PAT_CountPatPositionsDir(bool isBuy)
{
    int count = 0;
    int total = OrdersTotal();
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() != g_symbol) continue;
        if(!PAT_IsPatMagic(OrderMagicNumber())) continue;
        if(isBuy && OrderType() == OP_BUY) count++;
        if(!isBuy && OrderType() == OP_SELL) count++;
    }
    return count;
}

//+------------------------------------------------------------------+
//| EA-side risk gate (belt-and-suspenders — all fail-closed)         |
//+------------------------------------------------------------------+
bool PAT_PreTradeGate(bool isBuy, double lot, string strategyName)
{
    // ALL risk gates handled by SERVER. EA only checks emergency halt flags.
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
    double equity   = AccountEquity();
    double dist     = MathAbs(g_entry - g_sl);
    if(tickVal <= 0 || tickSize <= 0 || dist <= 0 || equity <= 0)
    {
        Print("REJECTED bad_risk_inputs: tickVal=", tickVal, " tickSize=", tickSize,
              " dist=", dist, " equity=", equity);
        return false;
    }
    double valuePerUnit = tickVal / tickSize; // account currency per 1.0 price move per lot
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

    // 7. Margin gate
    double marginRequired = lot * MarketInfo(g_symbol, MODE_MARGINREQUIRED);
    double freeMargin     = AccountFreeMargin();
    if(marginRequired > freeMargin * (MaxMarginUsagePct / 100.0))
    {
        Print("REJECTED margin_overuse: required=", DoubleToString(marginRequired, 2),
              " > freeMargin x ", MaxMarginUsagePct, "% = ",
              DoubleToString(freeMargin * MaxMarginUsagePct / 100.0, 2));
        return false;
    }
    if(AccountFreeMarginCheck(g_symbol, isBuy ? OP_BUY : OP_SELL, lot) <= 0)
    {
        Print("REJECTED insufficient_margin: AccountFreeMarginCheck <= 0");
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
    g_connection = "OFFLINE";
    g_licenseStatus = "PENDING";

    Print("Predict-A-Trade MT4 EA v1.00 initializing...");
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
    PAT_Watchdog();
    PAT_HistoryPoll();
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
    PAT_Append(PAT_TICK_FILE, slipMsg + "\n");

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
                  + " | date(UTC): " + TimeToString(closeTime, TIME_DATE)
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

    // RECOVERY: if the daily loss is no longer beyond the soft halt, clear the
    // block so a recovered/healthy account is not stuck blocked for the day.
    // (Previously g_tradingBlocked was set true but never re-evaluated, so a
    // single early-in-the-day loss kept trading blocked even after recovery.)
    if(lossPct > -effSoftHalt)
    {
        if(g_tradingBlocked)
        {
            g_tradingBlocked = false;
            Print("CAPITAL PROTECTION (RECOVER): daily loss recovered to ", lossPct, "% — trading re-enabled");
            string recMsg = "CAPITAL_PROTECTION|{";
            recMsg += "\"event_type\":\"RECOVER\"";
            recMsg += ",\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
            recMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
            recMsg += ",\"action\":\"RESUMED\"";
            recMsg += "}";
            PAT_Append(PAT_TICK_FILE, recMsg + "\n");
            PAT_LogLine("CAPITAL | dayOpenBal: " + DoubleToString(dayOpenBalance, 2) + " | dailyPnL: " + DoubleToString(g_dailyPnL, 2) + " | lossPct: " + DoubleToString(lossPct, 2) + " | status: RESUMED (not blocked)");
        }
    }
    else
    {
        if(lossPct <= -effWarning && !g_tradingBlocked)
        {
            string warnMsg = "CAPITAL_WARNING|{";
            warnMsg += "\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
            warnMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
            warnMsg += ",\"max_loss_pct\":" + DoubleToString(MaxDailyLossPct, 1);
            warnMsg += ",\"balance\":" + DoubleToString(curBal, 2);
            warnMsg += ",\"action\":\"WARNED\"";
            warnMsg += "}";
            PAT_Append(PAT_TICK_FILE, warnMsg + "\n");
            Print("CAPITAL WARNING: daily P&L=", g_dailyPnL, " (", lossPct, "%)");
        }
        if(lossPct <= -effSoftHalt && !g_tradingBlocked)
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
        g_tradingBlocked = true;
        Print("*** CAPITAL PROTECTION (HARD): Daily loss ", lossPct, "% — CLOSING ALL POSITIONS ***");
        string blockMsg = "CAPITAL_PROTECTION|{";
        blockMsg += "\"event_type\":\"DAILY_LOSS_LIMIT_HIT\"";
        blockMsg += ",\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
        blockMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
        blockMsg += ",\"balance\":" + DoubleToString(curBal, 2);
        blockMsg += ",\"action\":\"BLOCKED_NEW_TRADES\"";
        blockMsg += "}";
        PAT_Append(PAT_TICK_FILE, blockMsg + "\n");
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
        if(elapsed < (uint)TickIntervalMs) return;
    }
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

    PAT_Append(PAT_TICK_FILE, msg);
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
    string msg = "INIT|{\"ea_version\":\"1.08\",\"broker\":\"" + AccountCompany() +
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
                 ",\"floating_pnl\":" + DoubleToString(AccountProfit(), 2) +
                 "}\n";
    PAT_Append(PAT_TICK_FILE, msg);
}

//+------------------------------------------------------------------+
void RequestLicenseValidation()
{
    string msg = "LICENSE_CHECK|{\"account\":\"" + g_accountID +
                 "\",\"broker\":\"" + AccountCompany() +
                 "\",\"symbol\":\"" + g_symbol +
                 "\",\"license_key\":\"" + g_licenseKey +
                 "\",\"balance\":" + DoubleToStr(AccountBalance(), 2) +
                 ",\"equity\":" + DoubleToStr(AccountEquity(), 2) +
                 ",\"profit\":" + DoubleToStr(AccountProfit(), 2) +
                 ",\"open_positions\":" + IntegerToString(OrdersTotal()) +
                 "}\n";
    PAT_Append(PAT_TICK_FILE, msg);
    Print("License validation requested - balance: ", AccountBalance());
}

//+------------------------------------------------------------------+
void ReadFromAgent()
{
    if(FileIsExist(PAT_LICENSE_FILE, FILE_COMMON))
    {
        int lh = FileOpen(PAT_LICENSE_FILE, FILE_READ|FILE_TXT|FILE_COMMON);
        if(lh != INVALID_HANDLE)
        {
            string licContent = "";
            while(!FileIsEnding(lh))
                licContent += FileReadString(lh);
            FileClose(lh);
            licContent = StringTrimLeft(StringTrimRight(licContent));
            if(StringLen(licContent) > 0)
                HandleLicenseResponse(licContent);
            FileDelete(PAT_LICENSE_FILE, FILE_COMMON);
        }
    }

    if(!FileIsExist(PAT_SIGNAL_FILE, FILE_COMMON))
        return;

    int h = FileOpen(PAT_SIGNAL_FILE, FILE_READ|FILE_TXT|FILE_COMMON);
    if(h == INVALID_HANDLE) return;

    string content = "";
    while(!FileIsEnding(h))
    {
        content += FileReadString(h) + "\n";
    }
    FileClose(h);
    FileDelete(PAT_SIGNAL_FILE, FILE_COMMON);

    if(StringLen(content) == 0) return;

    g_signalsReceived++;

    int pos = 0;
    while(pos < StringLen(content))
    {
        int next = StringFind(content, "\n", pos);
        if(next < 0) next = StringLen(content);

        string line = StringSubstr(content, pos, next - pos);
        line = StringTrimLeft(StringTrimRight(line));
        if(StringLen(line) > 0)
        {
            int sep = StringFind(line, "|");
            if(sep > 0)
            {
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
        }
        pos = next + 1;
    }
}

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
                    OrderClose(OrderTicket(), OrderLots(), closePrice, 10, clrRed);
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
                OrderClose(OrderTicket(), OrderLots(), closePrice, 10, clrRed);
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
                OrderClose(OrderTicket(), OrderLots(), closePrice, 10, clrRed);
            }
        }
    }

    g_tradingStatus = "KILL_SWITCH";
    GlobalVariableSet("PAT_EQUITY_HALT", 1);
    g_equityHalted = true;

    string deinitMsg = "DEINIT|{\"reason\":\"SERVER_KILL_SWITCH\"}\n";
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
        if(g_licenseStatus == "ACTIVE")
            return true;
        Print("Strategy check: license not validated — blocking ", strategyID);
        return false;
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

    if(oldStatus != g_licenseStatus)
        Print("License status: ", oldStatus, " -> ", g_licenseStatus,
              " Plan:", g_licensePlan);

    // Client MT terminal log — record license/access status.
    string access = (g_licenseStatus == "ACTIVE") ? "Access Granted" : "Access Denied";
    PAT_LogLine("STATUS: " + access + " | License: " + g_licenseStatus + " | Subscription: " + g_licensePlan);
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
        PAT_Append(PAT_TICK_FILE, ack);
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
        PAT_Append(PAT_TICK_FILE, ack);
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
//| File I/O — append-safe (FILE_READ|FILE_WRITE + SEEK_END)          |
//+------------------------------------------------------------------+
void PAT_Write(string filename, string content)
{
    int h = FileOpen(filename, FILE_WRITE|FILE_TXT|FILE_COMMON);
    if(h == INVALID_HANDLE) return;
    FileWriteString(h, content);
    FileClose(h);
}

 void PAT_Append(string filename, string content)
{
    int h = FileOpen(filename, FILE_READ|FILE_WRITE|FILE_TXT|FILE_COMMON);
    if(h == INVALID_HANDLE)
    {
        h = FileOpen(filename, FILE_WRITE|FILE_TXT|FILE_COMMON);
        if(h == INVALID_HANDLE) return;
    }
    FileSeek(h, 0, SEEK_END);
    FileWriteString(h, content);
    FileClose(h);
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
