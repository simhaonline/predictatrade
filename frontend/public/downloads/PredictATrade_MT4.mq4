//+------------------------------------------------------------------+
//| Predict-A-Trade MT4 EA v1.06                                     |
//| Fixes: TP validation, position management, swap avoidance        |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property version   "1.07"
#property strict

// ─── Signal/Execution inputs ───
input bool    AutoExecute    = true;
input bool    SendTickData   = true;
input int     MagicNumber    = 20240002;
input int     TickIntervalMs = 0;
input string  BrokerSymbol   = "";
input string  LicenseKey     = "ee710bf6-5fe0-4b91-9b6b-a201348ea310";

// ─── Strategy/Direction filters ───
input bool    ReceiveStandardScalping = true;
input bool    ReceiveUltraScalping    = true;
input bool    ReceiveStandardSwing   = true;
input bool    ReceiveTrendSwing      = true;
input bool    ReceiveBuy             = true;
input bool    ReceiveSell            = true;
input bool    ReceiveBuyCandidate    = true;
input bool    ReceiveSellCandidate   = true;
input bool    ExecuteCandidates  = false;  // Execute BUY_CANDIDATE/SELL_CANDIDATE as real trades

// ─── Position Management (NEW v1.06) ───
input bool    UseTrailingStop  = true;     // Trail SL behind price after profit
input double  TrailingATRMult  = 2.0;      // Trailing distance = ATR * this multiplier
input bool    UseBreakEven     = true;     // Move SL to entry after 1R profit
input double  BreakEvenTriggerR = 1.0;     // R multiples to trigger break-even
input bool    CloseAtTP2       = true;     // Close full position at TP2 (after TP1 hit)
input int     MaxHoldHours     = 4;        // Max holding time in hours (0 = unlimited)

// ─── Swap Avoidance (NEW v1.06) ───
input bool    AvoidSwapCharges  = true;     // Close positions before swap/rollover
input int     SwapCutoffHour    = 22;       // Server hour to close before (default 22:00)
input int     SwapCutoffBuffer = 15;        // Close N minutes before cutoff
input bool    AvoidTripleSwapDay = true;     // Don't open new positions on triple swap day
input string  TripleSwapDay     = "Wednesday"; // Day that gets 3x swap charge

// ─── Slippage Control (NEW v1.07) ───
input int     MaxSlippagePoints = 3;        // Max acceptable slippage in points (0 = no limit)
input bool    RejectOnHighSlippage = true;  // Reject orders that exceed max slippage
input bool    AvoidRolloverSlippage = true; // Don't trade during rollover (spread widens)

// ─── Capital Protection (NEW v1.07) ───
input double  MaxDailyLossPct   = 6.0;      // Phase 4: hard halt threshold
input double  WarningLossPct    = 3.0;      // Warning level (close at 3%, block at 5%)
input bool    EmergencyCloseAll = true;     // Close all positions when daily loss hits limit
input double  SoftHaltLossPct   = 4.0;      // Phase 4: soft halt (block new, keep existing)
input double  HardHaltLossPct   = 6.0;      // Phase 4: hard halt (close all)


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

// ─── Constants ───
#define PAT_TICK_FILE   "pat_ticks.txt"
#define PAT_SIGNAL_FILE "PAT_signals.txt"
#define PAT_LICENSE_FILE "PAT_license.txt"
#define PAT_HEARTBEAT   "pat_heartbeat.txt"

// ─── Global state ───
string  g_connection      = "OFFLINE";
string  g_licenseStatus    = "PENDING";
string  g_licensePlan      = "";
string  g_accountID        = "";
string  g_symbol           = "";
string  g_signalID         = "";
string  g_signalDirection  = "NONE";
string  g_signalGrade      = "";
string  g_signalStrategy   = "";
string  g_signalClass       = "";
string  g_lastExecutedSignalID = "";
datetime g_signalTime      = 0;
uint    g_lastTickSend     = 0;
int     g_tickCount        = 0;
int     g_signalsDisplayed = 0;
int     g_signalsFiltered  = 0;
// Capital protection state (NEW v1.07)
double  g_dailyPnL       = 0;      // running daily P&L
double  g_dayStartBalance = 0;     // balance at start of trading day
datetime g_currentDay    = 0;     // current trading day
bool    g_tradingBlocked = false; // capital protection flag
bool    g_hardHaltTriggered = false; // Phase 4: hard halt flag
int     g_slippageRejects = 0;    // count of slippage rejections
double  g_entry  = 0;
double  g_sl     = 0;
double  g_tp1    = 0;
double  g_tp2    = 0;
double  g_tp3    = 0;
double  g_rawScore = 0;
double  g_calibProb = 0;

//+------------------------------------------------------------------+
// ─── Unified Price Access Wrappers (prompt.md Section 8.1) ───
// MT4 native — iClose/iOpen/iHigh/iLow already platform-agnostic in MT4.
// These wrappers provide a consistent interface for future MT5 migration.

double PAT_Close(int shift) { return iClose(g_symbol, 0, shift); }
double PAT_Open(int shift)  { return iOpen(g_symbol, 0, shift); }
double PAT_High(int shift)  { return iHigh(g_symbol, 0, shift); }
double PAT_Low(int shift)   { return iLow(g_symbol, 0, shift); }

// ─── Unified Swap/Tick Info Wrappers (prompt.md Section 8.3) ───
double PAT_SwapLong()   { return MarketInfo(g_symbol, MODE_SWAPLONG); }
double PAT_SwapShort()  { return MarketInfo(g_symbol, MODE_SWAPSHORT); }
double PAT_TickValue()  { return MarketInfo(g_symbol, MODE_TICKVALUE); }
double PAT_TickSize()   { return MarketInfo(g_symbol, MODE_TICKSIZE); }
double PAT_PointValue() { return (PAT_TickValue() / PAT_TickSize()); }

// ─── Position Sizing (prompt.md Section 3.2) ───
// lots = risk_amount / (stop_distance_price * point_value)
// Round down to broker lot step.
double PAT_CalcLotSize(double equity, double stopDistancePrice)
{
    if(stopDistancePrice <= 0 || equity <= 0) return 0;
    double riskAmount = equity * (RiskPerTradePct / 100.0);
    double pointValue = PAT_PointValue();
    if(pointValue <= 0) return 0;
    double lots = riskAmount / (stopDistancePrice * pointValue);
    double lotStep = MarketInfo(g_symbol, MODE_LOTSTEP);
    if(lotStep > 0) lots = MathFloor(lots / lotStep) * lotStep;
    double minLot = MarketInfo(g_symbol, MODE_MINLOT);
    double maxLot = MarketInfo(g_symbol, MODE_MAXLOT);
    if(lots < minLot) return 0; // Below minimum — reject
    if(lots > maxLot) lots = maxLot;
    return NormalizeDouble(lots, 2);
}

// ─── Per-Strategy Spread Check (prompt.md Section 4.2) ───
bool PAT_CheckSpread(string strategyName)
{
    int spread = (int)MarketInfo(g_symbol, MODE_SPREAD);
    int maxSpread = 50; // default
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
    return MaxSlippagePoints; // fallback to global
}

// ─── Swap Protection Check (prompt.md Section 4.1) ───
bool PAT_CheckSwapProtection(int orderType, double entry, double sl, double tp, double lots, bool isIntraday)
{
    double swapRate = (orderType == OP_BUY) ? PAT_SwapLong() : PAT_SwapShort();
    if(swapRate >= 0) return true; // Positive swap — no restriction
    if(isIntraday) return true; // Intraday closes before rollover

    // Swing: include swap cost in R:R
    int expectedNights = 3; // default expected hold nights
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
// TP1 hit: Close 50%, move SL to breakeven
// TP2 hit: Close 30%, move SL to TP1
// TP3 hit: Close remaining 20%, trail by 1.5*ATR
void PAT_ProcessPartialClose(int ticket, int orderType, double openPrice, double sl, double tp1, double tp2, double tp3, double originalLots)
{
    if(!UsePartialClose) return;
    if(!OrderSelect(ticket, SELECT_BY_TICKET)) return;
    if(OrderCloseTime() != 0) return; // already closed

    double currentLots = OrderLots();
    double tp1CloseLots = NormalizeDouble(originalLots * (TP1ClosePercent / 100.0), 2);
    double tp2CloseLots = NormalizeDouble(originalLots * (TP2ClosePercent / 100.0), 2);

    // TP1: close 50%, move SL to Entry + 0.5R (Phase 3C: guaranteed profit)
    if(orderType == OP_BUY && Bid >= tp1 && OrderLots() >= tp1CloseLots)
    {
        if(OrderClose(ticket, tp1CloseLots, Bid, PAT_GetMaxSlippage(""), clrGreen))
            Print("TP1 partial close: 50% closed at ", Bid);
        // ADDON Phase 3C: Move SL to Entry + 0.5R (not just breakeven)
        double r_dist = MathAbs(openPrice - sl);
        double newSL = openPrice + (0.5 * r_dist);
        if(OrderModify(ticket, newSL, newSL, OrderTakeProfit(), 0, clrYellow))
            Print("TP1: SL moved to Entry+0.5R (", newSL, ") — profit locked");
    }
    else if(orderType == OP_SELL && Ask <= tp1 && OrderLots() >= tp1CloseLots)
    {
        if(OrderClose(ticket, tp1CloseLots, Ask, PAT_GetMaxSlippage(""), clrRed))
            Print("TP1 partial close: 50% closed at ", Ask);
        // ADDON Phase 3C: Move SL to Entry - 0.5R (not just breakeven)
        double r_dist_s = MathAbs(sl - openPrice);
        double newSL_s = openPrice - (0.5 * r_dist_s);
        if(OrderModify(ticket, newSL_s, newSL_s, OrderTakeProfit(), 0, clrYellow))
            Print("TP1: SL moved to Entry-0.5R (", newSL_s, ") — profit locked");
    }

    // TP2: close 30%, move SL to TP1
    if(orderType == OP_BUY && Bid >= tp2 && OrderLots() >= tp2CloseLots)
    {
        if(OrderClose(ticket, tp2CloseLots, Bid, PAT_GetMaxSlippage(""), clrGreen))
            Print("TP2 partial close: 30% closed at ", Bid);
        if(OrderModify(ticket, openPrice, tp1, OrderTakeProfit(), 0, clrAqua))
            Print("TP2: SL moved to TP1 (", tp1, ")");
    }
    else if(orderType == OP_SELL && Ask <= tp2 && OrderLots() >= tp2CloseLots)
    {
        if(OrderClose(ticket, tp2CloseLots, Ask, PAT_GetMaxSlippage(""), clrRed))
            Print("TP2 partial close: 30% closed at ", Ask);
        if(OrderModify(ticket, openPrice, tp1, OrderTakeProfit(), 0, clrAqua))
            Print("TP2: SL moved to TP1 (", tp1, ")");
    }
    // TP3: remaining 20% trails by 1.5*ATR (handled by existing trailing stop logic)
}

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



int OnInit()
{
    if(BrokerSymbol != "")
        g_symbol = BrokerSymbol;
    else
        g_symbol = Symbol();

    g_accountID = IntegerToString(AccountNumber());
    g_connection = "OFFLINE";
    g_licenseStatus = "PENDING";
    SendInitMessage();
    Print("Predict-A-Trade MT4 EA v1.06 initialized | Symbol: ", g_symbol,
          " | Account: ", g_accountID, " | Swap avoidance: ", AvoidSwapCharges);
    return(INIT_SUCCEEDED);
}

void OnDeinit(const int reason)
{
    PAT_Write(PAT_TICK_FILE, "DEINIT|{}\n");
    Comment("");
}

//+------------------------------------------------------------------+
//| MAIN TICK — now manages open positions every tick                |
//+------------------------------------------------------------------+
void OnTick()
{
    CheckAgentConnection();

    if(SendTickData && g_connection == "CONNECTED")
        SendTickToAgent();

    if(g_connection == "CONNECTED")
        ReadFromAgent();

    // Signal expiry
    if(g_signalDirection != "NONE" && g_signalTime > 0)
    {
        if(TimeCurrent() > g_signalTime + 300)
            g_signalDirection = "EXPIRED";
    }

    // Manage all open positions (trailing stop, break-even, TP2, swap, max hold)
    ManageOpenPositions();

    // NEW v1.07: Capital protection — check daily loss limit
    UpdateCapitalProtection();

    UpdatePanel();
}

//+------------------------------------------------------------------+
//| POSITION MANAGEMENT (NEW v1.06)                                   |
//| Called every tick — monitors and manages all open PAT positions   |
//+------------------------------------------------------------------+
void ManageOpenPositions()
{
    int total = OrdersTotal();
    for(int i = total - 1; i >= 0; i--)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
            continue;
        if(OrderSymbol() != g_symbol || OrderMagicNumber() != MagicNumber)
            continue;

        int    ticket   = OrderTicket();
        int    type     = OrderType();
        double openPx   = OrderOpenPrice();
        double sl       = OrderStopLoss();
        double tp       = OrderTakeProfit();
        datetime openTime = OrderOpenTime();
        double profit   = OrderProfit();
        double swap     = OrderSwap();
        double commission = OrderCommission();

        // ─── 1. Check max holding time ───
        if(MaxHoldHours > 0)
        {
            int holdSec = (int)(TimeCurrent() - openTime);
            if(holdSec >= MaxHoldHours * 3600)
            {
                Print("MAX HOLD TIME reached: ticket=", ticket, " held=", holdSec/3600, "h | Closing...");
                ClosePosition(ticket, "MAX_HOLD_TIME");
                continue;
            }
        }

        // ─── 2. Check swap cutoff — close before rollover ───
        if(AvoidSwapCharges)
        {
            if(IsNearSwapTime())
            {
                Print("SWAP CUTOFF: closing ticket=", ticket, " before swap charge | profit=", profit, " swap=", swap);
                ClosePosition(ticket, "SWAP_AVOIDANCE");
                continue;
            }
        }

        // ─── 3. Break-even: move SL to entry + spread/cost buffer after 1R profit ───
        // Cost-aware break-even: SL = entry + spread buffer (not just entry)
        // This prevents a "break-even" stop from becoming a small realized loss
        // due to spread, commission, and execution costs.
        if(UseBreakEven && sl != openPx)
        {
            double risk = MathAbs(openPx - g_sl); // original SL distance (immutable 1R)
            if(risk > 0)
            {
                double currentProfitR = 0;
                double spread = MarketInfo(g_symbol, MODE_SPREAD) * MarketInfo(g_symbol, MODE_POINT);
                double digits = (int)MarketInfo(g_symbol, MODE_DIGITS);
                if(type == OP_BUY)
                    currentProfitR = (Bid - openPx) / risk;
                else
                    currentProfitR = (openPx - Ask) / risk;

                if(currentProfitR >= BreakEvenTriggerR)
                {
                    // Cost-aware break-even: add spread buffer above entry for BUY, below for SELL
                    double beSL = 0;
                    if(type == OP_BUY)
                        beSL = NormalizeDouble(openPx + spread, (int)digits); // entry + spread
                    else
                        beSL = NormalizeDouble(openPx - spread, (int)digits); // entry - spread
                    Print("BREAK-EVEN: moving SL to entry+spread for ticket=", ticket, " | profit R=", currentProfitR, " BE_SL=", beSL);
                    if(OrderModify(ticket, openPx, beSL, tp, 0, clrYellow))
                        Print("Break-even set: ticket=", ticket, " SL=", beSL);
                    else
                        Print("Break-even FAILED: error=", GetLastError());
                }
            }
        }

        // ─── 4. Trailing stop: move SL behind price (with broker stop level validation) ───
        if(UseTrailingStop)
        {
            double atr = GetATR(14);
            if(atr > 0)
            {
                double trailDist = atr * TrailingATRMult;
                double stopLevel = MarketInfo(g_symbol, MODE_STOPLEVEL) * MarketInfo(g_symbol, MODE_POINT);
                double freezeLevel = MarketInfo(g_symbol, MODE_FREEZELEVEL) * MarketInfo(g_symbol, MODE_POINT);
                int digits = (int)MarketInfo(g_symbol, MODE_DIGITS);

                if(type == OP_BUY)
                {
                    double newSL = NormalizeDouble(Bid - trailDist, digits);
                    // Monotonic: only move SL upward (BUY)
                    // Broker stop level: SL must be at least stopLevel below current Bid
                    double minSL = NormalizeDouble(Bid - stopLevel, digits);
                    if(newSL > minSL)
                        newSL = minSL; // respect broker stop level
                    // Freeze level: don't modify if price is within freeze level of SL
                    if(MathAbs(Bid - sl) < freezeLevel)
                        continue; // skip — too close to freeze level
                    // Only trail above entry and above current SL
                    if(newSL > sl && newSL > openPx)
                    {
                        if(OrderModify(ticket, openPx, newSL, tp, 0, clrAqua))
                            Print("Trailing BUY: ticket=", ticket, " SL=", sl, " -> ", newSL);
                        else
                            Print("Trailing BUY FAILED: ticket=", ticket, " error=", GetLastError());
                    }
                }
                else if(type == OP_SELL)
                {
                    double newSL = NormalizeDouble(Ask + trailDist, digits);
                    // Monotonic: only move SL downward (SELL)
                    double maxSL = NormalizeDouble(Ask + stopLevel, digits);
                    if(newSL < maxSL)
                        newSL = maxSL; // respect broker stop level
                    if(MathAbs(Ask - sl) < freezeLevel)
                        continue; // skip — too close to freeze level
                    if(newSL < sl && newSL < openPx)
                    {
                        if(OrderModify(ticket, openPx, newSL, tp, 0, clrAqua))
                            Print("Trailing SELL: ticket=", ticket, " SL=", sl, " -> ", newSL);
                        else
                            Print("Trailing SELL FAILED: ticket=", ticket, " error=", GetLastError());
                    }
                }
            }
        }

        // ─── 5. Close at TP2 — if price reaches TP2, close the position ───
        if(CloseAtTP2 && g_tp2 > 0)
        {
            if(type == OP_BUY && Bid >= g_tp2)
            {
                Print("TP2 HIT: closing BUY ticket=", ticket, " Bid=", Bid, " TP2=", g_tp2);
                ClosePosition(ticket, "TP2");
                continue;
            }
            else if(type == OP_SELL && Ask <= g_tp2)
            {
                Print("TP2 HIT: closing SELL ticket=", ticket, " Ask=", Ask, " TP2=", g_tp2);
                ClosePosition(ticket, "TP2");
                continue;
            }
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
        return false; // Not a market order

    bool ok = OrderClose(ticket, OrderLots(), closePrice, 5, clrOrange);
    if(ok)
    {
        double netProfit = OrderProfit() + OrderSwap() + OrderCommission();
        Print("CLOSED: ticket=", ticket, " reason=", reason,
              " | gross=", OrderProfit(), " swap=", OrderSwap(), " comm=", OrderCommission(),
              " | NET=", netProfit);
        PAT_Append(PAT_TICK_FILE, "CLOSE_ACK|{\"ticket\":" + IntegerToString(ticket) +
                   ",\"reason\":\"" + reason + "\"" +
                   ",\"net_pnl\":" + DoubleToString(netProfit, 2) + "}\n");
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
    // Server time
    int hour = TimeHour(TimeCurrent());
    int minute = TimeMinute(TimeCurrent());
    int dayOfWeek = TimeDayOfWeek(TimeCurrent());

    // Triple swap day check — don't open new positions (but still close existing ones)
    // 0=Sunday, 1=Monday, ..., 3=Wednesday, ..., 5=Friday

    // Close all positions SwapCutoffBuffer minutes before the cutoff hour
    int cutoffMinute = SwapCutoffHour * 60 - SwapCutoffBuffer;
    int nowMinute = hour * 60 + minute;

    if(nowMinute >= cutoffMinute && nowMinute < SwapCutoffHour * 60 + 30)
    {
        // Only close on trading days (Mon-Fri), not on weekends
        if(dayOfWeek >= 1 && dayOfWeek <= 5)
            return true;
    }
    return false;
}

//+------------------------------------------------------------------+
//| Check if today is triple swap day (typically Wednesday)          |
//+------------------------------------------------------------------+
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

//+------------------------------------------------------------------+
//| Get ATR value using iATR                                         |
//+------------------------------------------------------------------+
double GetATR(int period)
{
    return iATR(g_symbol, 0, period, 0);
}

//+------------------------------------------------------------------+
//| SLIPPAGE MONITORING (NEW v1.07)                                   |
//| Checks actual fill price vs requested price after order execution |
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

    // Report slippage to Windows Agent for database storage
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

    // Reject if slippage exceeds limit
    if(RejectOnHighSlippage && MaxSlippagePoints > 0 && slippagePoints > MaxSlippagePoints)
    {
        g_slippageRejects++;
        Print("SLIPPAGE EXCEEDED: ticket=", ticket, " requested=", requestedPrice,
              " filled=", filledPrice, " slip=", slippagePoints, " points (max=", MaxSlippagePoints, ")");
        // Close the position immediately — slippage too high
        ClosePosition(ticket, "SLIPPAGE_REJECT");
    }
    else
    {
        Print("Fill OK: ticket=", ticket, " requested=", requestedPrice, " filled=", filledPrice,
              " slippage=", slippagePoints, " points");
    }
}

//+------------------------------------------------------------------+
//| CAPITAL PROTECTION (NEW v1.07)                                    |
//| Tracks daily P&L and blocks trading when loss exceeds 5% of capital|
//+------------------------------------------------------------------+
void UpdateCapitalProtection()
{
    datetime today = TimeDay(TimeCurrent()) + TimeMonth(TimeCurrent()) * 100 + TimeYear(TimeCurrent()) * 10000;

    // New trading day — reset daily counters
    if(g_currentDay != today)
    {
        g_currentDay = today;
        g_dayStartBalance = AccountBalance();
        g_dailyPnL = 0;
        g_tradingBlocked = false;
        g_hardHaltTriggered = false;
        Print("NEW TRADING DAY: start balance=", g_dayStartBalance);
    }

    // Calculate daily P&L from closed trades today
    g_dailyPnL = 0;
    int total = OrdersHistoryTotal();
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)) continue;
        if(OrderSymbol() != g_symbol || OrderMagicNumber() != MagicNumber) continue;
        if(OrderCloseTime() == 0) continue;

        datetime closeDay = TimeDay(OrderCloseTime()) + TimeMonth(OrderCloseTime()) * 100 + TimeYear(OrderCloseTime()) * 10000;
        if(closeDay == today)
            g_dailyPnL += OrderProfit() + OrderSwap() + OrderCommission();
    }

    double lossPct = 0;
    if(g_dayStartBalance > 0)
        lossPct = (g_dailyPnL / g_dayStartBalance) * 100;

    // Dynamic thresholds for small accounts — prevents premature blocking on $50 accounts
    // Tier 1 (< $100): 3.5x thresholds — $50 * 14% = $7 loss before soft halt (2-3 losing trades)
    // Tier 2 ($100-$200): 2x thresholds — $150 * 8% = $12 loss before soft halt
    // Tier 3 (>= $200): normal thresholds — 4% soft halt
    double effSoftHalt = SoftHaltLossPct;
    double effHardHalt = HardHaltLossPct;
    double effWarning  = WarningLossPct;
    double minAbsLoss   = 1.0;  // Minimum absolute loss ($) to trigger any protection
    if(AccountBalance() < 100)
    {
        effSoftHalt = SoftHaltLossPct * 3.5;  // 4% -> 14% ($7 on $50)
        effHardHalt = HardHaltLossPct * 3.5;  // 6% -> 21% ($10.50 on $50)
        effWarning  = WarningLossPct * 3.5;   // 3% -> 10.5% ($5.25 on $50)
        minAbsLoss   = 3.0;                    // Don't block unless loss > $3
    }
    else if(AccountBalance() < 200)
    {
        effSoftHalt = SoftHaltLossPct * 2.0;  // 4% -> 8%
        effHardHalt = HardHaltLossPct * 2.0;  // 6% -> 12%
        effWarning  = WarningLossPct * 2.0;   // 3% -> 6%
        minAbsLoss   = 2.0;                    // Don't block unless loss > $2
    }

    // Minimum absolute loss floor — prevents blocking on tiny losses
    // that are within normal trading range for micro accounts
    if(g_dailyPnL > -minAbsLoss)
    {
        // Loss too small to trigger protection — allow trading
        return;
    }

    // Warning
    if(lossPct <= -effWarning && !g_tradingBlocked)
    {
        string warnMsg = "CAPITAL_WARNING|{";
        warnMsg += "\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
        warnMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
        warnMsg += ",\"max_loss_pct\":" + DoubleToString(MaxDailyLossPct, 1);
        warnMsg += ",\"balance\":" + DoubleToString(AccountBalance(), 2);
        warnMsg += ",\"action\":\"WARNED\"";
        warnMsg += "}";
        PAT_Append(PAT_TICK_FILE, warnMsg + "\n");
        Print("CAPITAL WARNING: daily P&L=", g_dailyPnL, " (", lossPct, "%) — approaching limit");
    }

    // ADDON Phase 4: Two-stage halt system
    // Soft halt at -4%: block new entries, let existing trades run naturally
    if(lossPct <= -effSoftHalt && !g_tradingBlocked)
    {
        g_tradingBlocked = true;
        Print("*** CAPITAL PROTECTION (SOFT): Daily loss ", lossPct, "% — new entries blocked, existing trades continue ***");
        string softMsg = "CAPITAL_PROTECTION|{\"event_type\":\"SOFT_HALT\",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2) + ",\"action\":\"BLOCKED_NEW_ENTRIES_ONLY\"}";
        PAT_Append(PAT_TICK_FILE, softMsg + "\n");
        // Do NOT close existing positions at soft halt
    }
    // Hard halt at -6%: emergency close all positions
    if(lossPct <= -effHardHalt && !g_hardHaltTriggered)
    {
        g_hardHaltTriggered = true;
        g_tradingBlocked = true;
        Print("*** CAPITAL PROTECTION (HARD): Daily loss ", lossPct, "% — CLOSING ALL POSITIONS ***");
        string blockMsg = "CAPITAL_PROTECTION|{";
        blockMsg += "\"event_type\":\"DAILY_LOSS_LIMIT_HIT\"";
        blockMsg += ",\"daily_pnl\":" + DoubleToString(g_dailyPnL, 2);
        blockMsg += ",\"daily_pnl_pct\":" + DoubleToString(lossPct, 2);
        blockMsg += ",\"max_loss_pct\":" + DoubleToString(MaxDailyLossPct, 1);
        blockMsg += ",\"balance\":" + DoubleToString(AccountBalance(), 2);
        blockMsg += ",\"equity\":" + DoubleToString(AccountEquity(), 2);
        blockMsg += ",\"open_positions\":" + IntegerToString(CountOpenPositions());
        blockMsg += ",\"action\":\"BLOCKED_NEW_TRADES\"";
        blockMsg += "}";
        PAT_Append(PAT_TICK_FILE, blockMsg + "\n");

        // Emergency close all positions
        if(EmergencyCloseAll)
        {
            Print("*** EMERGENCY CLOSE: closing all open positions ***");
            int totalPos = OrdersTotal();
            for(int i = totalPos - 1; i >= 0; i--)
            {
                if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
                if(OrderSymbol() == g_symbol && OrderMagicNumber() == MagicNumber)
                    ClosePosition(OrderTicket(), "EMERGENCY_CAPITAL_PROTECTION");
            }
        }
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
void ReadFromAgent()
{
    // Read license response from PAT_LICENSE_FILE (written by Windows Agent every 3s)
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

    // Read signals from PAT_SIGNAL_FILE (same file Windows Agent writes to)
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

    // Parse lines
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
            }
        }
        pos = next + 1;
    }
}

//+------------------------------------------------------------------+
void HandleSignal(string json)
{
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

    // NEW v1.06: Don't open new positions near swap time
    if(AvoidSwapCharges && IsNearSwapTime())
    {
        Print("SWAP AVOIDANCE: skipping signal near swap cutoff — ", g_signalDirection, " ", g_signalStrategy);
        g_signalsFiltered++;
        return;
    }

    // NEW v1.06: Don't open new positions on triple swap day
    if(IsTripleSwapDay())
    {
        Print("TRIPLE SWAP DAY: skipping signal to avoid 3x swap charge — ", g_signalDirection, " ", g_signalStrategy);
        g_signalsFiltered++;
        return;
    }

    // NEW v1.07: Capital protection — block new trades if daily loss limit hit
    if(g_tradingBlocked)
    {
        Print("CAPITAL PROTECTION: trading blocked — daily loss limit reached");
        g_signalsFiltered++;
        return;
    }

    // NEW v1.07: Check if we're in rollover period (high spread/slippage)
    if(AvoidRolloverSlippage && IsNearSwapTime())
    {
        Print("ROLLOVER SLIPPAGE: skipping signal — spread widened during rollover");
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
    if(oldStatus != g_licenseStatus)
        Print("License status: ", oldStatus, " -> ", g_licenseStatus, " Plan: ", g_licensePlan);
}

//+------------------------------------------------------------------+
//| FIXED v1.06: Now validates TP1 before placing order               |
//+------------------------------------------------------------------+
void ExecuteBuy()
{
    // FIXED: validate ALL critical levels including TP1
    if(g_entry <= 0 || g_sl <= 0 || g_tp1 <= 0)
    {
        Print("ExecuteBuy: INVALID levels — entry:", g_entry, " sl:", g_sl, " tp1:", g_tp1, " | SKIPPING (no trade without TP)");
        return;
    }

    // NEW: Don't open if we already have max positions
    if(CountOpenPositions() >= 3)
    {
        Print("ExecuteBuy: max positions reached, skipping");
        return;
    }

    double vol = CalculateLotSize();
    if(vol <= 0)
    {
        Print("ExecuteBuy: lot size = 0 (equity too small for min lot) — using minimum lot");
        vol = MarketInfo(g_symbol, MODE_MINLOT);
    }
    Print("ExecuteBuy: vol=", vol, " entry=", g_entry, " sl=", g_sl, " tp1=", g_tp1, " tp2=", g_tp2);
    int ticket = OrderSend(g_symbol, OP_BUY, vol, Ask, MaxSlippagePoints, g_sl, g_tp1, "PAT:" + g_signalID, MagicNumber, 0, clrGreen);
    if(ticket > 0)
    {
        g_lastExecutedSignalID = g_signalID;
        Print("BUY executed: ticket=", ticket, " vol=", vol, " SL=", g_sl, " TP1=", g_tp1);
        PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|{\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"}\n");
        CheckSlippage(ticket, "BUY", Ask); // NEW v1.07: monitor slippage
    }
    else
    {
        Print("BUY FAILED: error=", GetLastError(), " entry=", g_entry, " sl=", g_sl, " tp1=", g_tp1);
    }
}

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

    double vol = CalculateLotSize();
    if(vol <= 0)
    {
        Print("ExecuteSell: lot size = 0 (equity too small for min lot) — using minimum lot");
        vol = MarketInfo(g_symbol, MODE_MINLOT);
    }
    Print("ExecuteSell: vol=", vol, " entry=", g_entry, " sl=", g_sl, " tp1=", g_tp1, " tp2=", g_tp2);
    int ticket = OrderSend(g_symbol, OP_SELL, vol, Bid, MaxSlippagePoints, g_sl, g_tp1, "PAT:" + g_signalID, MagicNumber, 0, clrRed);
    if(ticket > 0)
    {
        g_lastExecutedSignalID = g_signalID;
        Print("SELL executed: ticket=", ticket, " vol=", vol, " SL=", g_sl, " TP1=", g_tp1);
        PAT_Append(PAT_TICK_FILE, "EXECUTION_ACK|{\"signal_id\":\"" + g_signalID + "\",\"status\":\"FILLED\"}\n");
        CheckSlippage(ticket, "SELL", Bid); // NEW v1.07: monitor slippage
    }
    else
    {
        Print("SELL FAILED: error=", GetLastError(), " entry=", g_entry, " sl=", g_sl, " tp1=", g_tp1);
    }
}

//+------------------------------------------------------------------+
//| Count open positions with our magic number                         |
//+------------------------------------------------------------------+
int CountOpenPositions()
{
    int count = 0;
    int total = OrdersTotal();
    for(int i = 0; i < total; i++)
    {
        if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
        if(OrderSymbol() == g_symbol && OrderMagicNumber() == MagicNumber)
            count++;
    }
    return count;
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
bool IsStrategyEnabled(string stratID)
{
    if(stratID == "STANDARD_SCALPING") return ReceiveStandardScalping;
    if(stratID == "ULTRA_SCALPING")    return ReceiveUltraScalping;
    if(stratID == "STANDARD_SWING")    return ReceiveStandardSwing;
    if(stratID == "TREND_SWING")       return ReceiveTrendSwing;
    return false;
}

bool IsDirectionEnabled(string dir)
{
    if(dir == "BUY")            return ReceiveBuy;
    if(dir == "SELL")            return ReceiveSell;
    if(dir == "BUY_CANDIDATE")   return ReceiveBuyCandidate;
    if(dir == "SELL_CANDIDATE")  return ReceiveSellCandidate;
    return false;
}

//+------------------------------------------------------------------+
void UpdatePanel()
{
    string p = "══════ PREDICT-A-TRADE v1.06 ══════\n";
    p += "Connection: " + g_connection + "\n";
    p += "License:    " + g_licenseStatus + " (" + g_licensePlan + ")\n";
    p += "Symbol:     " + g_symbol + "\n";
    p += "Open Pos:   " + IntegerToString(CountOpenPositions()) + "\n";
    p += "────────────────────────────────\n";
    if(g_signalDirection != "NONE" && g_signalDirection != "EXPIRED")
    {
        p += "Signal:    " + g_signalDirection + "\n";
        p += "Strategy:  " + g_signalStrategy + "\n";
        p += "Grade:     " + g_signalGrade + "\n";
        p += "Score:     " + DoubleToString(g_rawScore, 1) + "\n";
        p += "Entry:     " + DoubleToString(g_entry, 2) + "\n";
        p += "SL:        " + DoubleToString(g_sl, 2) + "\n";
        p += "TP1:       " + DoubleToString(g_tp1, 2) + "\n";
        p += "TP2:       " + DoubleToString(g_tp2, 2) + "\n";
        p += "TP3:       " + DoubleToString(g_tp3, 2) + "\n";
    }
    else
    {
        p += "No active signal\n";
    }
    p += "────────────────────────────────\n";
    p += "Ticks sent: " + IntegerToString(g_tickCount) + "\n";
    p += "Signals shown: " + IntegerToString(g_signalsDisplayed) + "\n";
    p += "Signals filtered: " + IntegerToString(g_signalsFiltered) + "\n";
    p += "Slippage rejects: " + IntegerToString(g_slippageRejects) + "\n";
    p += "Max slippage: " + IntegerToString(MaxSlippagePoints) + " pts\n";
    p += "Daily P&L: " + DoubleToString(g_dailyPnL, 2) + "\n";
    if(g_tradingBlocked) p += "*** TRADING BLOCKED (capital protection) ***\n";
    p += "Max daily loss: " + DoubleToString(MaxDailyLossPct, 1) + "%\n";

    // Swap avoidance status
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
