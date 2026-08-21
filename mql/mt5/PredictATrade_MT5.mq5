//+------------------------------------------------------------------+
//|                                          PredictATrade_MT5.mq5   |
//|                            Predict-A-Trade v1.0.0                |
//|        Tick data collection + licensed signal execution EA       |
//|  IPC: FILE_COMMON folder (shared between all MT terminals)       |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property version   "1.07"
#property strict

#include <Trade\Trade.mqh>

//=== Input Parameters ===
input bool    AutoExecute    = true;    // SIGNAL_ONLY=false, AUTO=true
input bool    SendTickData   = true;     // Send real tick data to Windows Agent
input ulong   MagicNumber    = 20240001;
input int     TickIntervalMs = 0;      // 0 = every tick (HFT: 1-5ms co-located)
input string  BrokerSymbol   = "";       // Empty = auto-detect chart symbol
input string  LicenseKey     = "ee710bf6-5fe0-4b91-9b6b-a201348ea310";       // Your Predict-A-Trade license key

//=== Strategy Selection (set to true to receive that strategy's signals) ===
input bool    ReceiveStandardScalping = true;   // STANDARD_SCALPING (M1/M5 scalping)
input bool    ReceiveUltraScalping    = true;   // ULTRA_SCALPING (M1 ultra-fast scalping)
input bool    ReceiveStandardSwing   = true;   // STANDARD_SWING (M15/H1 swing trading)
input bool    ReceiveTrendSwing      = true;   // TREND_SWING (H1/H4 trend following)

//=== Signal Direction Filter ===
input bool    ReceiveBuy             = true;   // Receive BUY signals (qualified)
input bool    ReceiveSell            = true;   // Receive SELL signals (qualified)
input bool    ReceiveBuyCandidate    = true;   // Receive BUY_CANDIDATE (advisory)
input bool    ReceiveSellCandidate   = true;   // Receive SELL_CANDIDATE (advisory)
input bool    ExecuteCandidates  = false;   // Execute candidates as real trades

//=== Position Management (NEW v1.06) ===
input bool    UseTrailingStop   = true;     // Trail SL behind price after profit
input double  TrailingATRMult   = 2.0;      // Trailing distance = ATR * this
input bool    UseBreakEven      = true;     // Move SL to entry after 1R profit
input double  BreakEvenTriggerR  = 1.0;     // R multiples to trigger break-even
input bool    CloseAtTP2        = true;     // Close full position at TP2
input int     MaxHoldHours       = 4;        // Max holding time (0 = unlimited)

//=== Swap Avoidance (NEW v1.06) ===
input bool    AvoidSwapCharges   = true;     // Close positions before swap/rollover
input int     SwapCutoffHour     = 22;       // Server hour to close before
input int     SwapCutoffBuffer   = 15;       // Close N minutes before cutoff
input bool    AvoidTripleSwapDay  = true;     // Skip new trades on triple swap day
input string  TripleSwapDay      = "Wednesday"; // Triple swap day

//=== Slippage Control (NEW v1.07) ===
input int     MaxSlippagePoints = 3;
input bool    RejectOnHighSlippage = true;
input bool    AvoidRolloverSlippage = true;

//=== Capital Protection (NEW v1.07) ===
input double  MaxDailyLossPct   = 6.0; // Phase 4: hard halt threshold
input double  WarningLossPct    = 3.0; // warning threshold
input bool    EmergencyCloseAll = true;

//=== File names (in FILE_COMMON folder — shared with Windows Agent) ===
#define PAT_TICK_FILE    "PAT_ticks.txt"
#define PAT_SIGNAL_FILE  "PAT_signals.txt"
#define PAT_LICENSE_FILE "PAT_license.txt"
#define PAT_HEARTBEAT    "PAT_heartbeat.txt"
#define PAT_ACK_FILE     "PAT_ack.txt"

//=== Global State ===

// ─── Per-Strategy Spread/Slippage Limits (prompt.md Section 4.2) ───
input int     UltraScalp_MaxSpread    = 15;  // Ultra Scalping: max spread in points
input int     UltraScalp_MaxSlippage  = 5;   // Ultra Scalping: max slippage in points
input int     StdScalp_MaxSpread      = 25;  // Standard Scalping: max spread in points
input int     StdScalp_MaxSlippage    = 10;  // Standard Scalping: max slippage in points
input int     StdSwing_MaxSpread      = 40;  // Standard Swing: max spread in points
input int     StdSwing_MaxSlippage    = 20;  // Standard Swing: max slippage in points
input int     TrendSwing_MaxSpread    = 50;  // Trend Swing: max spread in points
input int     TrendSwing_MaxSlippage  = 30;  // Trend Swing: max slippage in points

// ─── Position Sizing (prompt.md Section 3.2) ───
input double  RiskPerTradePct  = 1.0;  // Risk per trade as % of equity (1%)
input bool    UseAutoLotSizing = true; // Calculate lot size from risk % and stop distance

// ─── Partial Close / Profit Locking (prompt.md Section 3.3) ───
input bool    UsePartialClose  = true; // Enable partial close at TP1/TP2/TP3
input double  TP1ClosePercent  = 50.0; // Close 50% at TP1, move SL to breakeven
input double  TP2ClosePercent  = 30.0; // Close 30% at TP2, move SL to TP1
input double  TP3ClosePercent  = 20.0; // Close remaining 20% at TP3
input double  TP3TrailATRMult  = 1.5;  // Trail remaining by 1.5*ATR after TP3

// ─── ADDON: Phase 5 — Pending Limit Orders & SL Updates ───
input bool    UsePendingLimit   = true;   // Place LIMIT orders instead of MARKET (zero slippage)
input int     PendingExpiryMin   = 5;      // Pending order expiry (minutes)
input double  SoftHaltLossPct   = 4.0;    // Phase 4: Soft halt (block new, keep existing)
input double  HardHaltLossPct   = 6.0;    // Phase 4: Hard halt (close all)

CTrade        trade;
string        g_symbol;
string        g_connection    = "OFFLINE";
string        g_licenseStatus  = "UNKNOWN";
string        g_licensePlan    = "—";
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
bool          g_hardHaltTriggered = false; // Phase 4: hard halt flag
int           g_slippageRejects = 0;

//+------------------------------------------------------------------+
// ─── Unified Price Access Wrappers (prompt.md Section 8.1) ───
double PAT_Close(int shift) { double arr[]; CopyClose(g_symbol, PERIOD_CURRENT, shift, 1, arr); return arr[0]; }
double PAT_Open(int shift)  { double arr[]; CopyOpen(g_symbol, PERIOD_CURRENT, shift, 1, arr); return arr[0]; }
double PAT_High(int shift)  { double arr[]; CopyHigh(g_symbol, PERIOD_CURRENT, shift, 1, arr); return arr[0]; }
double PAT_Low(int shift)   { double arr[]; CopyLow(g_symbol, PERIOD_CURRENT, shift, 1, arr); return arr[0]; }

// ─── Unified Swap/Tick Info Wrappers (prompt.md Section 8.3) ───
double PAT_SwapLong()   { return SymbolInfoDouble(g_symbol, SYMBOL_SWAP_LONG); }
double PAT_SwapShort()  { return SymbolInfoDouble(g_symbol, SYMBOL_SWAP_SHORT); }
double PAT_TickValue()  { return SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_VALUE); }
double PAT_TickSize()   { return SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_SIZE); }
double PAT_PointValue() { return (PAT_TickValue() / PAT_TickSize()); }

// ─── Position Sizing (prompt.md Section 3.2) ───
double PAT_CalcLotSize(double equity, double stopDistancePrice)
{
    if(stopDistancePrice <= 0 || equity <= 0) return 0;
    double riskAmount = equity * (RiskPerTradePct / 100.0);
    double pointValue = PAT_PointValue();
    if(pointValue <= 0) return 0;
    double lots = riskAmount / (stopDistancePrice * pointValue);
    double lotStep = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_STEP);
    if(lotStep > 0) lots = MathFloor(lots / lotStep) * lotStep;
    double minLot = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MIN);
    double maxLot = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MAX);
    if(lots < minLot) return 0;
    if(lots > maxLot) lots = maxLot;
    return NormalizeDouble(lots, 2);
}

// ─── Per-Strategy Spread Check (prompt.md Section 4.2) ───
bool PAT_CheckSpread(string strategyName)
{
    int spread = (int)SymbolInfoInteger(g_symbol, SYMBOL_SPREAD);
    int maxSpread = 50;
    if(strategyName == "ULTRA_SCALPING") maxSpread = UltraScalp_MaxSpread;
    else if(strategyName == "STANDARD_SCALPING") maxSpread = StdScalp_MaxSpread;
    else if(strategyName == "STANDARD_SWING") maxSpread = StdSwing_MaxSpread;
    else if(strategyName == "TREND_SWING") maxSpread = TrendSwing_MaxSpread;
    if(spread > maxSpread) {
        Print("Spread check FAILED: ", spread, " > ", maxSpread, " for ", strategyName);
        return false;
    }
    return true;
}

// ─── Per-Strategy Slippage (prompt.md Section 4.2) ───
int PAT_GetMaxSlippage(string strategyName)
{
    if(strategyName == "ULTRA_SCALPING") return UltraScalp_MaxSlippage;
    if(strategyName == "STANDARD_SCALPING") return StdScalp_MaxSlippage;
    if(strategyName == "STANDARD_SWING") return StdSwing_MaxSlippage;
    if(strategyName == "TREND_SWING") return TrendSwing_MaxSlippage;
    return (int)MaxSlippagePoints;
}

// ─── Swap Protection Check (prompt.md Section 4.1) ───
bool PAT_CheckSwapProtection(ENUM_ORDER_TYPE orderType, double entry, double sl, double tp, double lots, bool isIntraday)
{
    double swapRate = (orderType == ORDER_TYPE_BUY) ? PAT_SwapLong() : PAT_SwapShort();
    if(swapRate >= 0) return true;
    if(isIntraday) return true;
    int expectedNights = 3;
    double swapCost = MathAbs(swapRate) * lots * expectedNights;
    double targetDist = MathAbs(tp - entry);
    double stopDist = MathAbs(entry - sl);
    if(stopDist <= 0) return false;
    double netProfit = targetDist - swapCost;
    double netRR = netProfit / stopDist;
    if(netRR < 2.0) {
        Print("Swap protection REJECT: net R:R=", DoubleToString(netRR, 2), " < 2.0");
        return false;
    }
    return true;
}

// ─── Partial Close / Profit Locking (prompt.md Section 3.3) ───
void PAT_ProcessPartialClose(ulong ticket, ENUM_POSITION_TYPE posType, double openPrice, double sl, double tp1, double tp2, double tp3, double originalLots)
{
    if(!UsePartialClose) return;
    if(!PositionSelectByTicket(ticket)) return;

    double currentLots = PositionGetDouble(POSITION_VOLUME);
    double tp1CloseLots = NormalizeDouble(originalLots * (TP1ClosePercent / 100.0), 2);
    double tp2CloseLots = NormalizeDouble(originalLots * (TP2ClosePercent / 100.0), 2);
    double bid = SymbolInfoDouble(g_symbol, SYMBOL_BID);
    double ask = SymbolInfoDouble(g_symbol, SYMBOL_ASK);

    // TP1: close 50%, move SL to breakeven
    if(posType == POSITION_TYPE_BUY && bid >= tp1 && currentLots >= tp1CloseLots)
    {
        trade.PositionClosePartial(ticket, tp1CloseLots);
        Print("TP1 partial close: 50% closed at ", bid);
        if(trade.PositionModify(ticket, openPrice, PositionGetDouble(POSITION_TP)))
            Print("TP1: SL moved to breakeven (", openPrice, ")");
    }
    else if(posType == POSITION_TYPE_SELL && ask <= tp1 && currentLots >= tp1CloseLots)
    {
        trade.PositionClosePartial(ticket, tp1CloseLots);
        Print("TP1 partial close: 50% closed at ", ask);
        if(trade.PositionModify(ticket, openPrice, PositionGetDouble(POSITION_TP)))
            Print("TP1: SL moved to breakeven (", openPrice, ")");
    }

    // TP2: close 30%, move SL to TP1
    if(posType == POSITION_TYPE_BUY && bid >= tp2 && currentLots >= tp2CloseLots)
    {
        trade.PositionClosePartial(ticket, tp2CloseLots);
        Print("TP2 partial close: 30% closed at ", bid);
        if(trade.PositionModify(ticket, tp1, PositionGetDouble(POSITION_TP)))
            Print("TP2: SL moved to TP1 (", tp1, ")");
    }
    else if(posType == POSITION_TYPE_SELL && ask <= tp2 && currentLots >= tp2CloseLots)
    {
        trade.PositionClosePartial(ticket, tp2CloseLots);
        Print("TP2 partial close: 30% closed at ", ask);
        if(trade.PositionModify(ticket, tp1, PositionGetDouble(POSITION_TP)))
            Print("TP2: SL moved to TP1 (", tp1, ")");
    }
    // TP3: remaining 20% trails by 1.5*ATR (handled by existing trailing stop logic)
}


//+------------------------------------------------------------------+
//| FormatISO8601UTC — Convert datetime to ISO8601 UTC string        |
//| Returns: "2026-08-21T16:25:11Z" (proper RFC3339/ISO8601 format)  |
//| This replaces TimeToString which produces "2026.08.21 19:25:11"  |
//| (dot separators, no timezone, broker time) — unparseable by JS   |
//+------------------------------------------------------------------+
string FormatISO8601UTC(datetime t)
{
    MqlDateTime dt;
    TimeToStruct(t, dt);
    return StringFormat("%04d-%02d-%02dT%02d:%02d:%02dZ",
        dt.year, dt.mon, dt.day, dt.hour, dt.min, dt.sec);
}

//+------------------------------------------------------------------+
//| FormatISO8601Broker — Broker time as ISO8601 (for reference)     |
//| Returns: "2026-08-21T19:25:11+03:00" (with broker offset)        |
//+------------------------------------------------------------------+
string FormatISO8601Broker(datetime t)
{
    MqlDateTime dt;
    TimeToStruct(t, dt);
    // Calculate broker offset: TimeCurrent() - TimeGMT()
    long offsetSec = (long)TimeCurrent() - (long)TimeGMT();
    int offsetH = (int)(offsetSec / 3600);
    long absOffsetSec = offsetSec >= 0 ? offsetSec : -offsetSec;
    int offsetM = (int)((absOffsetSec % 3600) / 60);
    string sign = offsetSec >= 0 ? "+" : "-";
    return StringFormat("%04d-%02d-%02dT%02d:%02d:%02d%s%02d:%02d",
        dt.year, dt.mon, dt.day, dt.hour, dt.min, dt.sec, sign, offsetH >= 0 ? offsetH : -offsetH, offsetM);
}

int OnInit()
{
    Print("Predict-A-Trade MT5 EA v1.06 initializing...");

    g_symbol = BrokerSymbol;
    if(g_symbol == "") g_symbol = _Symbol;
    g_licenseKey = LicenseKey;
    g_accountID = IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN));

    trade.SetExpertMagicNumber(MagicNumber);

    Print("Symbol: ", g_symbol);
    Print("Account: ", g_accountID);
    Print("License Key: ", (g_licenseKey == "" ? "NOT SET — SIGNALS WILL BE IGNORED" : g_licenseKey));

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
//+------------------------------------------------------------------+
//| SLIPPAGE MONITORING (NEW v1.07)                                   |
//+------------------------------------------------------------------+
void CheckSlippage(ulong ticket, string direction, double requestedPrice)
{
    if(ticket == 0) return;
    if(!PositionSelectByTicket(ticket)) return;
    double filledPrice = PositionGetDouble(POSITION_PRICE_OPEN);
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
        ClosePosition(ticket, "SLIPPAGE_REJECT");
    }
}

//+------------------------------------------------------------------+
//| CAPITAL PROTECTION (NEW v1.07)                                    |
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
    // FIX: Only count today's closed deals (not last 24h) — matches MT4 behavior
    MqlDateTime todayDt;
    TimeToStruct(TimeCurrent(), todayDt);
    datetime dayStart = (datetime)(todayDt.year * 10000 + todayDt.mon * 100 + todayDt.day);
    HistorySelect(0, TimeCurrent());
    int deals = HistoryDealsTotal();
    for(int i = 0; i < deals; i++)
    {
        ulong dealTicket = HistoryDealGetTicket(i);
        if(dealTicket == 0) continue;
        if(HistoryDealGetString(dealTicket, DEAL_SYMBOL) != g_symbol) continue;
        if(HistoryDealGetInteger(dealTicket, DEAL_MAGIC) != (long)MagicNumber) continue;
        // Only count deals from today
        datetime dealTime = (datetime)HistoryDealGetInteger(dealTicket, DEAL_TIME);
        MqlDateTime dealDt;
        TimeToStruct(dealTime, dealDt);
        datetime dealDay = (datetime)(dealDt.year * 10000 + dealDt.mon * 100 + dealDt.day);
        if(dealDay != dayStart) continue;
        g_dailyPnL += HistoryDealGetDouble(dealTicket, DEAL_PROFIT)
                     + HistoryDealGetDouble(dealTicket, DEAL_SWAP)
                     + HistoryDealGetDouble(dealTicket, DEAL_COMMISSION);
    }
    double lossPct = 0;
    if(g_dayStartBalance > 0) lossPct = (g_dailyPnL / g_dayStartBalance) * 100;

    // Dynamic thresholds for small accounts — tiered for micro accounts
    // Tier 1 (< $100): 3.5x — $50 * 14% = $7 loss before soft halt (2-3 losing trades)
    // Tier 2 ($100-$200): 2x — $150 * 8% = $12 loss before soft halt
    // Tier 3 (>= $200): normal — 4% soft halt
    double effSoftHalt = SoftHaltLossPct;
    double effHardHalt = HardHaltLossPct;
    double effWarning  = WarningLossPct;
    double minAbsLoss   = 1.0;
    if(AccountInfoDouble(ACCOUNT_BALANCE) < 100)
    {
        effSoftHalt = SoftHaltLossPct * 3.5;  // 4% -> 14% ($7 on $50)
        effHardHalt = HardHaltLossPct * 3.5;  // 6% -> 21% ($10.50 on $50)
        effWarning  = WarningLossPct * 3.5;   // 3% -> 10.5% ($5.25 on $50)
        minAbsLoss   = 3.0;                    // Don't block unless loss > $3
    }
    else if(AccountInfoDouble(ACCOUNT_BALANCE) < 200)
    {
        effSoftHalt = SoftHaltLossPct * 2.0;  // 4% -> 8%
        effHardHalt = HardHaltLossPct * 2.0;  // 6% -> 12%
        effWarning  = WarningLossPct * 2.0;   // 3% -> 6%
        minAbsLoss   = 2.0;                    // Don't block unless loss > $2
    }

    // Minimum absolute loss floor — prevents blocking on tiny losses
    if(g_dailyPnL > -minAbsLoss)
    {
        return; // Loss too small to trigger protection
    }

    if(lossPct <= -effWarning && !g_tradingBlocked)
    {
        string warnMsg = "CAPITAL_WARNING|{\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2)
                       + ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + "}";
        PAT_Append(PAT_TICK_FILE, warnMsg + "\n");
    }
    // ADDON Phase 4: Two-stage halt system
    // Soft halt at -4%: block new entries, let existing trades run naturally
    if(lossPct <= -effSoftHalt && !g_tradingBlocked)
    {
        g_tradingBlocked = true;
        Print("*** CAPITAL PROTECTION (SOFT): Daily loss ", lossPct, "% — new entries blocked, existing trades continue ***");
        string softMsg = "CAPITAL_PROTECTION|\"event_type\":\"SOFT_HALT\""
                        + ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2)
                        + ",\"action\":\"BLOCKED_NEW_ENTRIES_ONLY\"}";
        PAT_Append(PAT_TICK_FILE, softMsg + "\n");
        // Do NOT close existing positions at soft halt
    }
    // Hard halt at -6%: emergency close all positions
    if(lossPct <= -effHardHalt)
    {
        if(!g_hardHaltTriggered)
        {
            g_hardHaltTriggered = true;
            Print("*** CAPITAL PROTECTION (HARD): Daily loss ", lossPct, "% — CLOSING ALL POSITIONS ***");
            string hardMsg = "CAPITAL_PROTECTION|\"event_type\":\"HARD_HALT\""
                           + ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2)
                           + ",\"action\":\"EMERGENCY_CLOSE_ALL\"}";
            PAT_Append(PAT_TICK_FILE, hardMsg + "\n");
            if(EmergencyCloseAll)
            {
                int total = PositionsTotal();
                for(int i = total - 1; i >= 0; i--)
                {
                    ulong ticket = PositionGetTicket(i);
                    if(ticket == 0) continue;
                    if(!PositionSelectByTicket(ticket)) continue;
                    if(PositionGetString(POSITION_SYMBOL) == g_symbol &&
                       PositionGetInteger(POSITION_MAGIC) == (long)MagicNumber)
                        ClosePosition(ticket, "EMERGENCY_CAPITAL_PROTECTION");
                }
            }
        }
    }
}

//+------------------------------------------------------------------+
//| POSITION MANAGEMENT (NEW v1.06)                                   |
//+------------------------------------------------------------------+
void ManageOpenPositions()
{
    int total = PositionsTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PositionSelectByTicket(ticket)) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        if(PositionGetInteger(POSITION_MAGIC) != (long)MagicNumber) continue;

        long   posType   = PositionGetInteger(POSITION_TYPE);
        double openPx   = PositionGetDouble(POSITION_PRICE_OPEN);
        double sl       = PositionGetDouble(POSITION_SL);
        double tp       = PositionGetDouble(POSITION_TP);
        datetime openTime = (datetime)PositionGetInteger(POSITION_TIME);
        double profit   = PositionGetDouble(POSITION_PROFIT);
        double swap     = PositionGetDouble(POSITION_SWAP);

        // 1. Max holding time
        if(MaxHoldHours > 0)
        {
            int holdSec = (int)(TimeCurrent() - openTime);
            if(holdSec >= MaxHoldHours * 3600)
            {
                Print("MAX HOLD TIME: ticket=", ticket, " held=", holdSec/3600, "h | Closing");
                ClosePosition(ticket, "MAX_HOLD_TIME");
                continue;
            }
        }

        // 2. Swap cutoff
        if(AvoidSwapCharges && IsNearSwapTime())
        {
            Print("SWAP CUTOFF: closing ticket=", ticket, " | profit=", profit, " swap=", swap);
            ClosePosition(ticket, "SWAP_AVOIDANCE");
            continue;
        }

        // 3. Break-even (cost-aware: SL = entry + spread buffer)
        if(UseBreakEven && sl != openPx)
        {
            double risk = MathAbs(openPx - g_sl);
            if(risk > 0)
            {
                double profitR = 0;
                double spread = (double)SymbolInfoInteger(g_symbol, SYMBOL_SPREAD) * SymbolInfoDouble(g_symbol, SYMBOL_POINT);
                int digits = (int)SymbolInfoInteger(g_symbol, SYMBOL_DIGITS);
                if(posType == POSITION_TYPE_BUY)
                    profitR = (SymbolInfoDouble(g_symbol, SYMBOL_BID) - openPx) / risk;
                else
                    profitR = (openPx - SymbolInfoDouble(g_symbol, SYMBOL_ASK)) / risk;

                if(profitR >= BreakEvenTriggerR)
                {
                    double beSL = 0;
                    if(posType == POSITION_TYPE_BUY)
                        beSL = NormalizeDouble(openPx + spread, digits);
                    else
                        beSL = NormalizeDouble(openPx - spread, digits);
                    Print("BREAK-EVEN: ticket=", ticket, " profitR=", profitR, " BE_SL=", beSL);
                    if(trade.PositionModify(ticket, beSL, tp))
                        Print("Break-even set: ticket=", ticket, " SL=", beSL);
                    else
                        Print("Break-even FAILED: retcode=", trade.ResultRetcode());
                }
            }
        }

        // 4. Trailing stop (with broker stop level validation)
        if(UseTrailingStop)
        {
            int atrHandle = iATR(g_symbol, PERIOD_CURRENT, 14);
            double atrBuffer[];
            double atr = 0;
            if(atrHandle != INVALID_HANDLE && CopyBuffer(atrHandle, 0, 0, 1, atrBuffer) > 0)
                atr = atrBuffer[0];
            if(atrHandle != INVALID_HANDLE) IndicatorRelease(atrHandle);
            if(atr > 0)
            {
                double trailDist = atr * TrailingATRMult;
                int digits = (int)SymbolInfoInteger(g_symbol, SYMBOL_DIGITS);
                double point = SymbolInfoDouble(g_symbol, SYMBOL_POINT);
                double stopLevel = (double)SymbolInfoInteger(g_symbol, SYMBOL_TRADE_STOPS_LEVEL) * point;
                double freezeLevel = (double)SymbolInfoInteger(g_symbol, SYMBOL_TRADE_FREEZE_LEVEL) * point;
                double bid = SymbolInfoDouble(g_symbol, SYMBOL_BID);
                double ask = SymbolInfoDouble(g_symbol, SYMBOL_ASK);

                if(posType == POSITION_TYPE_BUY)
                {
                    double newSL = NormalizeDouble(bid - trailDist, digits);
                    double minSL = NormalizeDouble(bid - stopLevel, digits);
                    if(newSL > minSL)
                        newSL = minSL;
                    if(MathAbs(bid - sl) < freezeLevel)
                        continue; // too close to freeze level
                    if(newSL > sl && newSL > openPx)
                    {
                        if(trade.PositionModify(ticket, newSL, tp))
                            Print("Trailing BUY: ticket=", ticket, " SL=", sl, " -> ", newSL);
                        else
                            Print("Trailing BUY FAILED: ticket=", ticket, " retcode=", trade.ResultRetcode());
                    }
                }
                else
                {
                    double newSL = NormalizeDouble(ask + trailDist, digits);
                    double maxSL = NormalizeDouble(ask + stopLevel, digits);
                    if(newSL < maxSL)
                        newSL = maxSL;
                    if(MathAbs(ask - sl) < freezeLevel)
                        continue;
                    if(newSL < sl && newSL < openPx)
                    {
                        if(trade.PositionModify(ticket, newSL, tp))
                            Print("Trailing SELL: ticket=", ticket, " SL=", sl, " -> ", newSL);
                        else
                            Print("Trailing SELL FAILED: ticket=", ticket, " retcode=", trade.ResultRetcode());
                    }
                }
            }
        }

        // 5. Close at TP2
        if(CloseAtTP2 && g_tp2 > 0)
        {
            double bid = SymbolInfoDouble(g_symbol, SYMBOL_BID);
            double ask = SymbolInfoDouble(g_symbol, SYMBOL_ASK);
            if(posType == POSITION_TYPE_BUY && bid >= g_tp2)
            {
                Print("TP2 HIT: closing BUY ticket=", ticket);
                ClosePosition(ticket, "TP2");
                continue;
            }
            else if(posType == POSITION_TYPE_SELL && ask <= g_tp2)
            {
                Print("TP2 HIT: closing SELL ticket=", ticket);
                ClosePosition(ticket, "TP2");
                continue;
            }
        }
    }
}

//+------------------------------------------------------------------+
bool ClosePosition(ulong ticket, string reason)
{
    bool ok = trade.PositionClose(ticket);
    if(ok)
    {
        Print("CLOSED: ticket=", ticket, " reason=", reason);
        PAT_Append(PAT_TICK_FILE, "CLOSE_ACK|\"ticket\":" + IntegerToString((long)ticket) +
                   ",\"reason\":\"" + reason + "\"}\n");
    }
    else
    {
        Print("CLOSE FAILED: ticket=", ticket, " error=", GetLastError());
    }
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

//+------------------------------------------------------------------+
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
int CountOpenPositions()
{
    int count = 0;
    int total = PositionsTotal();
    for(int i = 0; i < total; i++)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PositionSelectByTicket(ticket)) continue;
        if(PositionGetString(POSITION_SYMBOL) == g_symbol &&
           PositionGetInteger(POSITION_MAGIC) == (long)MagicNumber)
            count++;
    }
    return count;
}

void OnTick()
{
    CheckAgentConnection();

    if(SendTickData && g_connection == "CONNECTED")
        SendTickToAgent();

    if(g_connection == "CONNECTED")
        ReadFromAgent();

    // NEW v1.06: Manage open positions every tick
    ManageOpenPositions();

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
void SendTickToAgent()
{
    if(TickIntervalMs > 0)
    {
        uint elapsed = GetTickCount() - g_lastTickSend;
        if(elapsed < (uint)TickIntervalMs) return;
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
                 ",\"floating_pnl\":" + DoubleToString(AccountInfoDouble(ACCOUNT_PROFIT), 2) +
                 "}\n";
    PAT_Write(PAT_TICK_FILE, msg);
    Print("Init message sent with account data - balance: ", AccountInfoDouble(ACCOUNT_BALANCE));
}

//+------------------------------------------------------------------+
void RequestLicenseValidation()
{
    string msg = "LICENSE_CHECK|{\"account\":\"" + g_accountID +
                 "\",\"broker\":\"" + AccountInfoString(ACCOUNT_COMPANY) +
                 "\",\"symbol\":\"" + g_symbol +
                 "\",\"license_key\":\"" + g_licenseKey +
                 "\",\"balance\":" + DoubleToString(AccountInfoDouble(ACCOUNT_BALANCE), 2) +
                 ",\"equity\":" + DoubleToString(AccountInfoDouble(ACCOUNT_EQUITY), 2) +
                 ",\"profit\":" + DoubleToString(AccountInfoDouble(ACCOUNT_PROFIT), 2) +
                 ",\"open_positions\":" + IntegerToString(PositionsTotal()) +
                 "}\n";
    PAT_Append(PAT_TICK_FILE, msg);
    Print("License validation with account data - balance: ", AccountInfoDouble(ACCOUNT_BALANCE));
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
            Print("=== SIGNAL FILE READ === content length: ", StringLen(content));

            string lines[];
            int count = StringSplit(content, '\n', lines);
            Print("Lines found: ", count);

            for(int i = 0; i < count; i++)
            {
                string line = lines[i];
                if(StringLen(line) == 0) continue;
                Print("Line ", i, ": ", StringSubstr(line, 0, MathMin(80, StringLen(line))), "...");

                int sep = StringFind(line, "|");
                if(sep < 0) continue;
                string msgType = StringSubstr(line, 0, sep);
                string payload = StringSubstr(line, sep + 1);

                Print("Message type: ", msgType, " payload length: ", StringLen(payload));

                if(msgType == "SIGNAL")
                {
                    HandleSignal(payload);
                }
            }
            PAT_Clear(PAT_SIGNAL_FILE);
        }
    }
}

//+------------------------------------------------------------------+
bool IsStrategyEnabled(string strategyID)
{
    if(strategyID == "STANDARD_SCALPING" && ReceiveStandardScalping) return true;
    if(strategyID == "ULTRA_SCALPING"    && ReceiveUltraScalping)    return true;
    if(strategyID == "STANDARD_SWING"   && ReceiveStandardSwing)     return true;
    if(strategyID == "TREND_SWING"      && ReceiveTrendSwing)        return true;
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
    Print("=== HandleSignal called === json length: ", StringLen(json));
    Print("JSON preview: ", StringSubstr(json, 0, MathMin(200, StringLen(json))));

    g_signalID        = ExtractJSONString(json, "ID");
    g_signalDirection = ExtractJSONString(json, "Direction");
    g_signalGrade     = ExtractJSONString(json, "Grade");
    g_signalStrategy  = ExtractJSONString(json, "StrategyID");
    g_signalClass     = ExtractJSONString(json, "SignalClass");
    g_entry  = ExtractJSONDouble(json, "EntryPrice");
    g_sl     = ExtractJSONDouble(json, "StopLoss");
    g_tp1    = ExtractJSONDouble(json, "TP1");
    g_tp2    = ExtractJSONDouble(json, "TP2");
    g_tp3    = ExtractJSONDouble(json, "TP3");
    g_rawScore = ExtractJSONDouble(json, "RawScore");
    g_calibProb = ExtractJSONDouble(json, "CalibratedProbability");
    g_signalTime = TimeCurrent();

    Print("Parsed: ID=", g_signalID, " Dir=", g_signalDirection, " Strategy=", g_signalStrategy,
          " Entry=", g_entry, " SL=", g_sl, " TP1=", g_tp1);

    if(g_signalID == g_lastExecutedSignalID)
    {
        Print("Duplicate signal ID — skipping: ", g_signalID);
        return;
    }

    // Strategy filter — skip if strategy not enabled
    if(!IsStrategyEnabled(g_signalStrategy))
    {
        g_signalsFiltered++;
        Print("Strategy FILTERED OUT: ", g_signalStrategy, " (not enabled in EA inputs)");
        return;
    }

    // Direction filter — skip if direction not enabled
    if(!IsDirectionEnabled(g_signalDirection))
    {
        g_signalsFiltered++;
        Print("Direction FILTERED OUT: ", g_signalDirection, " (not enabled in EA inputs)");
        return;
    }

    // Check connection
    if(g_connection != "CONNECTED")
    {
        Print("Agent not connected — signal ignored. Connection: ", g_connection);
        return;
    }

    // Check license
    if(g_licenseStatus != "ACTIVE")
    {
        Print("License not active — signal ignored. License status: ", g_licenseStatus);
        return;
    }

    // Check trading halt
    if(g_tradingStatus == "HALTED" || g_tradingStatus == "KILL_SWITCH" || g_tradingStatus == "EMERGENCY_HALT")
    {
        Print("Trading is ", g_tradingStatus, " — signal received but not executed.");
        return;
    }

    // Log all signal types for display
    if(g_tradingBlocked) { g_signalsFiltered++; return; }
    if(AvoidRolloverSlippage && IsNearSwapTime()) { g_signalsFiltered++; return; }
    g_signalsDisplayed++;
    Print(">>> SIGNAL DISPLAYED: ", g_signalDirection, " | ", g_signalStrategy, " | Grade:", g_signalGrade,
          " | Class:", g_signalClass, " | Score:", g_rawScore, " | Entry:", g_entry, " | SL:", g_sl, " | TP1:", g_tp1);

    // Execute only qualified BUY/SELL (not candidates)
    if(AutoExecute && g_signalDirection == "BUY")
    {
        Print("AutoExecute BUY triggered");
        ExecuteBuy();
    }
    else if(AutoExecute && g_signalDirection == "SELL")
    {
        Print("AutoExecute SELL triggered");
        ExecuteSell();
    }
    else if(g_signalDirection == "BUY_CANDIDATE" || g_signalDirection == "SELL_CANDIDATE")
    {
        Print("Advisory signal (not executable): ", g_signalDirection);
    }
    else if(g_signalDirection == "NO-TRADE")
    {
        // NO-TRADE signals are not forwarded by the engine — this should not happen
        Print("NO-TRADE signal received (unexpected)");
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

    if(oldStatus != g_licenseStatus)
    {
        Print("License status changed: ", oldStatus, " → ", g_licenseStatus,
              " Plan:", g_licensePlan, " Auth:", g_authStatus,
              " Device:", g_deviceStatus, " Session:", g_sessionStatus,
              " Trading:", g_tradingStatus);
    }
}

//+------------------------------------------------------------------+
void ExecuteBuy()
{
    if(g_entry <= 0 || g_sl <= 0 || g_tp1 <= 0)
    {
        Print("ExecuteBuy: INVALID levels — entry:", g_entry, " sl:", g_sl, " tp1:", g_tp1, " | SKIPPING (no trade without TP)");
        return;
    }
    if(CountOpenPositions() >= 3)
    {
        Print("ExecuteBuy: max positions reached, skipping");
        return;
    }
    if(AvoidSwapCharges && IsNearSwapTime())
    {
        Print("SWAP AVOIDANCE: skipping BUY signal near cutoff");
        return;
    }
    if(IsTripleSwapDay())
    {
        Print("TRIPLE SWAP DAY: skipping BUY signal to avoid 3x charge");
        return;
    }
    double vol = CalculateLotSize();
    Print("ExecuteBuy: vol=", vol, " entry=", g_entry, " sl=", g_sl, " tp1=", g_tp1);
    // ADDON Phase 5: Hybrid limit + market fallback (zero slippage when possible)
    if(UsePendingLimit && MathAbs(SymbolInfoDouble(g_symbol, SYMBOL_ASK) - g_entry) < (0.5 * SymbolInfoDouble(g_symbol, SYMBOL_POINT)))
    {
        // Price is close to entry — use LIMIT order for zero slippage
        datetime expiry = TimeCurrent() + (PendingExpiryMin * 60);
        MqlTradeRequest req;
        MqlTradeResult res;
        ZeroMemory(req);
        ZeroMemory(res);
        req.action    = TRADE_ACTION_PENDING;
        req.symbol    = g_symbol;
        req.volume    = vol;
        req.type      = ORDER_TYPE_BUY_LIMIT;
        req.price     = g_entry;
        req.sl        = g_sl;
        req.tp        = g_tp1;
        req.type_time = ORDER_TIME_SPECIFIED;
        req.expiration = expiry;
        req.magic     = MagicNumber;
        req.comment   = "PAT:" + g_signalID;
        if(OrderSend(req, res) && (res.retcode == TRADE_RETCODE_DONE || res.retcode == TRADE_RETCODE_PLACED))
        {
            g_lastExecutedSignalID = g_signalID;
            Print("BUY LIMIT placed: ticket ", res.order, " expiry=", PendingExpiryMin, "min");
            PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|\"signal_id\":\"" + g_signalID + "\",\"status\":\"PENDING_LIMIT\"\n");
        }
        else
        {
            Print("BUY LIMIT FAILED: ", res.retcode, " ", res.comment, " — falling back to MARKET");
            if(trade.Buy(vol, g_symbol, g_entry, g_sl, g_tp1, "PAT:" + g_signalID))
            {
                g_lastExecutedSignalID = g_signalID;
                PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"\n");
            }
            else
            {
                Print("BUY MARKET FALLBACK FAILED: ", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
            }
        }
    }
    else if(trade.Buy(vol, g_symbol, g_entry, g_sl, g_tp1, "PAT:" + g_signalID))
    {
        g_lastExecutedSignalID = g_signalID;
        Print("BUY executed: ticket ", trade.ResultOrder());
        PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|{\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"}\n");
        CheckSlippage(trade.ResultOrder(), "BUY", SymbolInfoDouble(g_symbol, SYMBOL_ASK));
    }
    else
    {
        Print("BUY FAILED: ", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
    }
}

//+------------------------------------------------------------------+
void ExecuteSell()
{
    if(g_entry <= 0 || g_sl <= 0 || g_tp1 <= 0)
    {
        Print("ExecuteSell: INVALID levels — entry:", g_entry, " sl:", g_sl, " tp1:", g_tp1, " | SKIPPING (no trade without TP)");
        return;
    }
    if(CountOpenPositions() >= 3)
    {
        Print("ExecuteSell: max positions reached, skipping");
        return;
    }
    if(AvoidSwapCharges && IsNearSwapTime())
    {
        Print("SWAP AVOIDANCE: skipping SELL signal near cutoff");
        return;
    }
    if(IsTripleSwapDay())
    {
        Print("TRIPLE SWAP DAY: skipping SELL signal to avoid 3x charge");
        return;
    }
    double vol = CalculateLotSize();
    Print("ExecuteSell: vol=", vol, " entry=", g_entry, " sl=", g_sl, " tp1=", g_tp1);
    // ADDON Phase 5: Hybrid limit + market fallback (zero slippage when possible)
    if(UsePendingLimit && MathAbs(SymbolInfoDouble(g_symbol, SYMBOL_BID) - g_entry) < (0.5 * SymbolInfoDouble(g_symbol, SYMBOL_POINT)))
    {
        // Price is close to entry — use LIMIT order for zero slippage
        datetime expiry = TimeCurrent() + (PendingExpiryMin * 60);
        MqlTradeRequest req;
        MqlTradeResult res;
        ZeroMemory(req);
        ZeroMemory(res);
        req.action    = TRADE_ACTION_PENDING;
        req.symbol    = g_symbol;
        req.volume    = vol;
        req.type      = ORDER_TYPE_SELL_LIMIT;
        req.price     = g_entry;
        req.sl        = g_sl;
        req.tp        = g_tp1;
        req.type_time = ORDER_TIME_SPECIFIED;
        req.expiration = expiry;
        req.magic     = MagicNumber;
        req.comment   = "PAT:" + g_signalID;
        if(OrderSend(req, res) && (res.retcode == TRADE_RETCODE_DONE || res.retcode == TRADE_RETCODE_PLACED))
        {
            g_lastExecutedSignalID = g_signalID;
            Print("SELL LIMIT placed: ticket ", res.order, " expiry=", PendingExpiryMin, "min");
            PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|\"signal_id\":\"" + g_signalID + "\",\"status\":\"PENDING_LIMIT\"\n");
        }
        else
        {
            Print("SELL LIMIT FAILED: ", res.retcode, " ", res.comment, " — falling back to MARKET");
            if(trade.Sell(vol, g_symbol, g_entry, g_sl, g_tp1, "PAT:" + g_signalID))
            {
                g_lastExecutedSignalID = g_signalID;
                PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"\n");
            }
            else
            {
                Print("SELL MARKET FALLBACK FAILED: ", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
            }
        }
    }
    else if(trade.Sell(vol, g_symbol, g_entry, g_sl, g_tp1, "PAT:" + g_signalID))
    {
        g_lastExecutedSignalID = g_signalID;
        Print("SELL executed: ticket ", trade.ResultOrder());
        PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|{\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"}\n");
        CheckSlippage(trade.ResultOrder(), "SELL", SymbolInfoDouble(g_symbol, SYMBOL_BID));
    }
    else
    {
        Print("SELL FAILED: ", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
    }
}

//+------------------------------------------------------------------+
double CalculateLotSize()
{
    double balance = AccountInfoDouble(ACCOUNT_BALANCE);
    double risk = balance * 0.01;
    double slDist = MathAbs(g_entry - g_sl);
    double tickVal = SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_VALUE);
    double tickSize = SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_SIZE);
    if(slDist <= 0 || tickVal <= 0 || tickSize <= 0) return 0.01;
    double lot = risk / ((slDist / tickSize) * tickVal);
    double minLot = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MIN);
    double maxLot = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MAX);
    double step = SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_STEP);
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
    // Skip leading spaces or quotes
    while(s < StringLen(json))
    {
        int c = StringGetCharacter(json, s);
        if(c == 32 || c == 34) { s++; continue; }  // skip space or quote
        break;
    }
    string v = "";
    for(int i = s; i < StringLen(json); i++)
    {
        int c = StringGetCharacter(json, i);
        if(c == 44 || c == 125 || c == 32 || c == 34) break;  // comma, brace, space, quote
        v += CharToString((uchar)c);
    }
    return StringToDouble(v);
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
    string p = "═══ Predict-A-Trade v1.05 ═══\n";
    p += "Agent:    " + g_connection + "\n";
    p += "License:  " + g_licenseStatus + " (" + g_licensePlan + ")\n";
    p += "Lic.Key:  " + (g_licenseKey == "" ? "NOT SET" : StringSubstr(g_licenseKey, 0, 12) + "...") + "\n";
    p += "Account:  " + g_accountID + "\n";
    p += "Symbol:   " + g_symbol + "\n";
    p += "Mode:     " + (AutoExecute ? "AUTO EXECUTE" : "SIGNAL ONLY") + "\n";
    p += "──────────────────────────────\n";
    p += "Signals:  " + IntegerToString(g_signalsReceived) + " recv, " + IntegerToString(g_signalsDisplayed) + " shown, " + IntegerToString(g_signalsFiltered) + " filtered\n";
    p += "Strats:   ";
    if(ReceiveStandardScalping) p += "SS ";
    if(ReceiveUltraScalping)    p += "US ";
    if(ReceiveStandardSwing)    p += "SW ";
    if(ReceiveTrendSwing)      p += "TW\n";
    p += "──────────────────────────────\n";
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
    p += "──────────────────────────────\n";
    p += "Ticks:    " + IntegerToString((long)g_tickCount) + "\n";
    p += "Time:     " + TimeToString(TimeCurrent(), TIME_DATE|TIME_SECONDS) + "\n";
    p += "Slip rejects: " + IntegerToString(g_slippageRejects) + "\n";
    p += "Daily P&L: " + DoubleToString(g_dailyPnL, 2) + "\n";
    if(g_tradingBlocked) p += "*** TRADING BLOCKED ***\n";
    if(AvoidSwapCharges)
    {
        p += "Swap cutoff: " + IntegerToString(SwapCutoffHour) + ":00 (-" + IntegerToString(SwapCutoffBuffer) + "min)\n";
        if(IsNearSwapTime()) p += "*** SWAP CUTOFF ACTIVE ***\n";
        if(IsTripleSwapDay()) p += "*** TRIPLE SWAP DAY ***\n";
    }
    Comment(p);
}
//+------------------------------------------------------------------+
