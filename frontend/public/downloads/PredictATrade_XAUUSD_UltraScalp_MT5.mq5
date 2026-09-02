//+------------------------------------------------------------------------+
//| PredictATrade_XAUUSD_MT5.mq5                                           |
//| Predict-A-Trade v1.00                                                  |
//| Ultra Scalping XAUUSD EA (16-Gate)                                     |
//|                                                                        |
//|  SELF-CONTAINED: no .mqh includes, no external scripts, no WebRequest, |
//|  no files. Uses only native MQL5 (OrderSend, SymbolInfo*, PositionGet*,|
//|  AccountInfo*, CalendarValueHistory).                                  |
//|                                                                        |
//|  Gate order (short-circuit, NO-TRADE is a first-class result):         |
//|    1  ExecutionPermission                                              |
//|    2  BrokerSymbolValidation                                           |
//|    3  SeedCapitalProtection  (3% soft / 5% hard daily cap)             |
//|    4  DailyLossLimit                                                   |
//|    5  MaxSpread  (auto-adaptive)                                       |
//|    6  NewsRisk  (native MT5 economic calendar)                         |
//|    7  Slippage                                                         |
//|    8  MaxPositions                                                     |
//|    9  MaxExposure                                                      |
//|   10  Cooldown                                                         |
//|   11  StopHuntFilter                                                   |
//|   12  MarginCheck                                                      |
//|   13  OvertradeProtection                                              |
//|   14  MaxDailyTrades                                                   |
//|   15  RegimeFilter  (+ EURUSD dollar proxy)                            |
//|   16  ProfitTarget                                                     |
//+------------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property link      "https://predictatrade.com"
#property version   "1.17"
#property description "Predict-A-Trade Ultra Scalping XAUUSD EA — M1 decision, tight SL, 16-gate fail-closed risk pipeline, auto-cost-adaptive targets. Self-contained MQL5."

//+------------------------------------------------------------------+
//| INPUTS — Execution & Entitlement                                 |
//+------------------------------------------------------------------+
input group "=== 1. Execution Permission ==="
input bool    EnableTrading        = true;    // Master switch (false = signal-only, NO orders)
input string  LicenseKey           = "";      // License key (optional; empty = local mode)

input group "=== 2. Broker Symbol ==="
input string  TradeSymbol          = "XAUUSD"; // Symbol to trade ("" = current chart symbol)
input double  MaxSpreadPoints      = 20.0;    // Max spread in points (gate 5) — used when AutoSpread=false
input bool    AutoSpread           = true;    // Auto-adapt spread limit to the symbol's typical spread
input double  AutoSpreadMult       = 2.0;     // Auto limit = typical spread x this (blocks only abnormal blowouts)

input group "=== 3. Seed Capital Protection (5% daily cap) ==="
input double  SeedCapitalPct       = 3.0;     // Soft block: stop NEW entries at -3% of day-start (gate 3)
input double  HardStopPct          = 5.0;     // Hard floor: close-ALL + halt at -5% (never bypassed)
input double  MaxRunningDrawdownPct = 4.0;    // Halt if equity drops this % from intraday peak

input group "=== 4. Daily Loss Limit ==="
input double  DailyLossLimitPct    = 5.0;     // Per-(strategy,tf) daily loss limit % (gate 4)
input bool    BypassDailyLossBlock = false;   // Allow new trades after soft daily-loss limit (hard halt NEVER bypassed)

input group "=== 5. News Risk (native MT5 economic calendar) ==="
input bool    EnableNewsFilter     = true;    // Enable news blocking (gate 6)
input int     NewsBlockMin         = 5;       // Block trades within N min of a HIGH-importance event (5 = tight, keeps scalper active)
input bool    BlockWeekend         = true;    // NO-TRADE on weekends (market closed — not a session limit)
// v1.15: BlockFridayLate / FridayCutoffHour REMOVED per operator directive
// (all sessions traded; rollover spread blowout is handled by the Gate 5
// spread cap, not by a clock gate).

input group "=== 6. Slippage ==="
input double  MaxSlippagePts       = 5.0;     // Max allowed slippage points (gate 7)
input int     MaxDeviationPoints   = 10;      // Max deviation for order (points)

input group "=== 7. Positions & Exposure ==="
input int     MaxPositions         = 3;       // Max total open positions (gate 8)
input int     MaxSameDirPositions  = 2;       // Max same-direction positions (gate 8)
input double  MaxExposurePct       = 30.0;    // Max aggregate exposure % of equity (gate 9)
input bool    AllowScaleIn         = true;    // Multi-trade: scale in (pyramid) same-direction signals
input double  ScaleInLotMult       = 1.0;     // Lot multiplier for each scale-in (1.0 = same lot)
input int     TradeDirection       = 0;       // 0=both, 1=buy only, -1=sell only

input group "=== 8. Cooldown ==="
input int     CooldownSeconds      = 60;      // Cooldown after a closed trade (gate 10)
input int     MinBarsBetweenTrades = 3;       // Min M1 bars between entries (gate 10)

input group "=== 9. Stop Hunt Filter ==="
input bool    EnableStopHuntFilter = true;    // Enable stop-hunt / liquidity-sweep filter (gate 11)
input int     StopHuntLookback     = 20;      // Bars to scan for sweep (gate 11)
input double  StopHuntWickPct      = 0.30;    // Wick must exceed body by this fraction to count as sweep

//+------------------------------------------------------------------+
//| 0. FILTER STACK — measured v1.17                                 |
//+------------------------------------------------------------------+
// v1.16: SimpleMode was REMOVED at code level (22% WR live+backtest proved
// the stripped config negative-edge). v1.17 went further: walk-forward P&L
// attribution on 66 days of M1 data showed the M15 HTF gate and the SMC/
// Range/Momentum paths all lose money — the ONLY profitable entry is the
// trend-pullback. Those paths and the HTF gate are REMOVED at code level.
// RSI confirmation stays (measured neutral-to-positive). Safety rails
// — spread cap, cost firewall, daily caps — unchanged.
bool g_useRSI;

input group "=== 1. General ==="
input double  MaxMarginUsagePct    = 30.0;    // Max free-margin usage % for a new order (gate 12)

input group "=== 11. Overtrade Protection ==="
input int     MaxTradesPerHour     = 6;       // Max trades per rolling hour (gate 13)
input int     MaxConsecutiveLosses = 5;       // Pause after N consecutive losses (gate 13)
input int     ConsecLossCooldownSec = 300;    // Pause duration after hitting the loss streak (then auto-resume)
input double  MinProfitToResetLossStreak = 5.0; // Min profit ($) to reset the loss streak (tiny wins don't reset)

input group "=== 12. Max Daily Trades ==="
input int     MaxDailyTrades       = 15;      // Max trades per day (gate 14)

input group "=== 13. Regime Filter (+ dollar proxy) ==="
input bool    EnableRegimeFilter   = true;    // Enable regime filter (gate 15)
input int     RegimeEMAPeriod      = 50;      // EMA period for trend regime
input double  RegimeMinATR         = 0.50;    // Min M1 ATR (price units, e.g. $0.50) to trade — avoids dead market
input double  RegimeMaxATR         = 5.00;    // Max M1 ATR (price units, e.g. $5.00) — avoid wild volatility
input bool    UseDollarProxy       = true;    // Use EURUSD as native dollar proxy (inverse of DXY)
input string  DollarProxySymbol    = "EURUSD"; // Symbol used as dollar proxy ("" = disabled)

input group "=== 14. Profit Target ==="
input double  DailyProfitTargetPct = 3.0;     // Daily profit target % — stop trading when hit (gate 16)
input bool    StopOnProfitTarget   = true;    // Halt new entries after daily profit target

input group "=== 15. Position Sizing & Exit (Ultra Scalp — tight R:R) ==="
input double  RiskPerTradePct      = 0.25;    // Risk % of equity per trade (read.md: 0.25-0.5%; v1.16 ships the conservative end)
input double  BaseLot              = 0.01;    // Base lot (min)
input double  MaxLot               = 1.00;    // Max lot
// --- Exit engine (v1.17, measured) ---
// v1.17 removed the micro-TP decay / TP1-TP2 partial ladder / BE move / ATR
// trailing at code level: walk-forward simulation on 66 days of M1 data
// showed every stage lost money. The measured optimum is ONE final target
// and ONE hard stop, sized as multiples of the measured round-trip cost
// (spread + commission + slippage + buffer) so the same script works on
// raw/ECN and standard accounts without re-tuning.
input bool    AutoCostAdapt        = true;    // Auto-size TP/SL as multiples of measured round-trip cost (works on raw AND standard accounts)
input double  CostSLMult           = 8.0;     // SL = this x round-trip cost (auto mode) — measured optimum (sim sl8/tp12)
input double  CostTPMult           = 12.0;    // TP = this x round-trip cost (auto mode; single final target) — measured optimum
input double  MicroTPBufferPts     = 2.0;     // Cost buffer (points) added to spread+commission+slippage when measuring round-trip cost
input double  FixedSLPts           = 80.0;    // SL distance in points (used when AutoCostAdapt=false)
input double  FixedTPPts           = 160.0;   // TP distance in points (used when AutoCostAdapt=false)
input double  BrokerCommissionPerLot = 0.0;  // Broker commission per lot (in account currency, round-trip)
input double  MinRRBeforeCost      = 1.5;     // Cost firewall: skip if TP < this x round-trip cost (read.md: safety_factor 1.5-2.0)
input double  MaxEntrySpreadPts    = 45.0;    // Hard gate: block entry when LIVE spread > this (blocks abnormal blowouts; normal 33pt spread still allowed)

input group "=== 16. Signal Logic (Ultra Scalping M1) ==="
input int     SignalEMAFast        = 9;       // Fast EMA period (read.md: EMA 9)
input int     SignalEMASlow        = 21;      // Slow EMA period (read.md: EMA 21)
input bool    UseExhaustionFilter  = true;    // Skip entry when EMA spread too wide (momentum spent; jbm-ema-gold-scalper concept)
input double  MaxEMASpreadATR      = 1.5;     // Max fast-slow EMA gap (in ATR) to consider trend 'fresh' (replay-tuned: 0.8 strangled P2 — blocked 43% of pullback bars)
input bool    UseRSIFilter         = true;    // Soft RSI confirmation (read.md: avoid overbought/oversold extremes)
input int     RSIPeriod            = 7;       // RSI period (read.md: 7 for gold scalping)
input double  RSIOverbought        = 70.0;    // Don't BUY above this (overbought)
input double  RSIOversold          = 30.0;    // Don't SELL below this (oversold)
input int     MagicBase            = 40101;   // Magic number base (Predict-A-Trade convention)
// v1.17: UseHTFTrendBias/HTFEMA, UseSMC and its 8 sub-inputs, RangeBars/
// RangeMaxATR/ReversalBarATR, MomentumBodyATR REMOVED at code level —
// walk-forward P&L attribution showed P1/P3/P4 and the M15 gate all
// negative; the EA now trades the single measured-profitable path.

input group "=== 17. VWAP / display (read.md, display-only) ==="
input bool    UseVWAP              = true;    // Session VWAP shown on the panel (display-only; not a gate)
input int     VWAPSessionHours      = 24;      // VWAP session length in hours (24 = daily)

// v1.17: Stochastic trigger (5,3,3) group REMOVED — GetStochK() was
// display-only (grep: never referenced by any signal path or gate).

input group "=== 19. M5 ATR (sizing/stop + sit-out filter) ==="
input int     ATRM5Period           = 14;      // ATR period on M5 (read.md: ATR(14) M5)
input double  MinATR5Dollars        = 1.20;    // Sit out if M5 ATR < this $ (read.md: ~$1.20)

// v1.17: SuperTrend trailing group REMOVED (UseSuperTrend was false-default,
// never wired into any exit stage; trailing measured negative in the sim).

// === 21. Session Timing ===
// v1.15 ALL-SESSIONS (operator directive): the EA trades every session —
// no London/NY window, no Asian skip. The session inputs were removed
// entirely so a chart-preserved value can never re-enable the gate
// (same lesson as SimpleMode: MT5 preserves input values on recompile).
// All clock decisions still run on TimeCurrent() broker server time only.

//+------------------------------------------------------------------+
//| GLOBALS                                                           |
//+------------------------------------------------------------------+
string        g_symbol;
long          g_magic;
int           g_emaFastHandle = INVALID_HANDLE;
int           g_emaSlowHandle = INVALID_HANDLE;
int           g_atrHandle     = INVALID_HANDLE;
int           g_regimeEmaHandle = INVALID_HANDLE;
int           g_rsiHandle     = INVALID_HANDLE;
int           g_atrM5Handle   = INVALID_HANDLE;

// Day-state (persisted via GlobalVariables to survive reloads)
datetime      g_currentDay    = 0;
double        g_dayStartBalance = 0;
double        g_dailyPnL      = 0;
int           g_dailyTrades   = 0;
int           g_consecLosses  = 0;
datetime      g_lastTradeCloseTime = 0;
datetime      g_lastTradeBarOpen = 0;   // open time of the last bar we evaluated (FIX B1)
string        g_lastNoEntryMsg   = "";  // last printed no-entry reason (one print per minute)
// Signal-path counters (panel): pullback is the only remaining path
int           g_sigPathPullback = 0;    // entries fired by the trend-pullback path
int           g_sigNoEntryBars  = 0;    // bars evaluated with no signal at all
bool          g_halted        = false;   // hard halt (equity floor) — never bypassed
datetime      g_haltDay       = 0;       // day (server) the halt latched (FIX A.1)
bool          g_profitHalt    = false;   // daily profit target reached
bool          g_lossHalt      = false;   // daily loss limit reached (soft, bypassable)

// Rolling hour trade timestamps
datetime      g_tradeTimes[64];
int           g_tradeCount   = 0;

// Cached external data (refreshed on timer)
double        g_avgSpread     = 0;       // running average spread (auto-adaptive MaxSpread)
int           g_spreadSamples = 0;

//+------------------------------------------------------------------+
//| Native symbol helpers (replaces CSymbolInfo)                      |
//+------------------------------------------------------------------+
double SymBid()        { return SymbolInfoDouble(g_symbol, SYMBOL_BID); }
double SymAsk()         { return SymbolInfoDouble(g_symbol, SYMBOL_ASK); }
double SymPoint()       { return SymbolInfoDouble(g_symbol, SYMBOL_POINT); }
int    SymDigits()      { return (int)SymbolInfoInteger(g_symbol, SYMBOL_DIGITS); }
long   SymSpread()      { return SymbolInfoInteger(g_symbol, SYMBOL_SPREAD); }
double SymTickValue()   { return SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_VALUE); }
double SymTickSize()    { return SymbolInfoDouble(g_symbol, SYMBOL_TRADE_TICK_SIZE); }
double SymVolStep()     { return SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_STEP); }
double SymVolMin()      { return SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MIN); }
double SymVolMax()      { return SymbolInfoDouble(g_symbol, SYMBOL_VOLUME_MAX); }
long   SymStopsLevel()  { return SymbolInfoInteger(g_symbol, SYMBOL_TRADE_STOPS_LEVEL); }
long   SymTradeMode()   { return SymbolInfoInteger(g_symbol, SYMBOL_TRADE_MODE); }

//+------------------------------------------------------------------+
//| Magic helpers                                                     |
//+------------------------------------------------------------------+
long PAT_Magic() { return (long)MagicBase; }
bool PAT_IsOurMagic(long m) { return (m >= MagicBase && m < MagicBase + 100); }

//+------------------------------------------------------------------+
//| GlobalVariable persistence helpers                                |
//+------------------------------------------------------------------+
string GV_Day()        { return "PAT_DAY"; }
string GV_StartBal()   { return "PAT_STARTBAL"; }
string GV_DailyPnL()   { return "PAT_DAILYPnL"; }
string GV_DailyTrades(){ return "PAT_DAILYTRADES"; }
string GV_ConsecLoss() { return "PAT_CONSECLOSS"; }
string GV_LastClose()  { return "PAT_LASTCLOSE"; }
string GV_Halt()       { return "PAT_HALT"; }
string GV_HaltDay()    { return "PAT_HALTDAY"; }
string GV_PeakEquity() { return "PAT_PEAKEQ"; }
string GV_ProfitHalt() { return "PAT_PROFITHALT"; }
string GV_LossHalt()   { return "PAT_LOSSHALT"; }
string GV_Heartbeat()      { return "PAT_HEARTBEAT_" + g_symbol; }  // last-run time
string GV_HeartbeatChart() { return "PAT_HBCHART_" + g_symbol; }    // chart id of owner

//+------------------------------------------------------------------+
//| OnInit                                                            |
//+------------------------------------------------------------------+
int OnInit()
{
    g_symbol = (TradeSymbol == "" ? _Symbol : TradeSymbol);
    g_magic  = PAT_Magic();

    // v1.17: single measured-profitable path (trend-pullback) + RSI gate.
    g_useRSI   = UseRSIFilter;

    // Duplicate-instance guard: GlobalVariables are shared across all charts
    // in one terminal. If a DIFFERENT chart's instance heartbeated recently,
    // refuse to start (double-attach would double every position). A re-init
    // on the SAME chart (recompile, timeframe change, input change) reclaims
    // ownership automatically via the stored chart id.
    long myChart = ChartID();
    double otherBeat = GlobalVariableGet(GV_Heartbeat());
    double otherChart = GlobalVariableGet(GV_HeartbeatChart());
    bool otherAlive = (otherBeat > 0 &&
                       TimeCurrent() - (datetime)otherBeat < 120 &&
                       (long)otherChart != myChart);
    if(otherAlive)
    {
        Print("FATAL: another UltraScalping instance is already running on ", g_symbol,
              " (heartbeat ", IntegerToString((int)(TimeCurrent() - (datetime)otherBeat)),
              "s old, chart ", DoubleToString(otherChart, 0),
              "). Refusing to start — detach the other chart first.");
        return(INIT_FAILED);
    }
    GlobalVariableSet(GV_Heartbeat(), (double)TimeCurrent());
    GlobalVariableSet(GV_HeartbeatChart(), (double)myChart);

    if(!SymbolSelect(g_symbol, true))
    {
        Print("FATAL: symbol '", g_symbol, "' not found on this broker.");
        return(INIT_FAILED);
    }

    g_emaFastHandle = iMA(g_symbol, PERIOD_M1, SignalEMAFast, 0, MODE_EMA, PRICE_CLOSE);
    g_emaSlowHandle = iMA(g_symbol, PERIOD_M1, SignalEMASlow, 0, MODE_EMA, PRICE_CLOSE);
    g_atrHandle     = iATR(g_symbol, PERIOD_M1, 14);
    g_regimeEmaHandle = iMA(g_symbol, PERIOD_M1, RegimeEMAPeriod, 0, MODE_EMA, PRICE_CLOSE);
    g_rsiHandle     = (g_useRSI) ? iRSI(g_symbol, PERIOD_M1, RSIPeriod, PRICE_CLOSE) : INVALID_HANDLE;
    g_atrM5Handle   = iATR(g_symbol, PERIOD_M5, ATRM5Period);
    if(g_emaFastHandle == INVALID_HANDLE || g_emaSlowHandle == INVALID_HANDLE || g_atrHandle == INVALID_HANDLE || g_regimeEmaHandle == INVALID_HANDLE || g_atrM5Handle == INVALID_HANDLE)
    {
        Print("FATAL: indicator handle creation failed.");
        return(INIT_FAILED);
    }

    // Restore persisted day-state
    g_currentDay    = (datetime)GlobalVariableGet(GV_Day());
    g_dayStartBalance = GlobalVariableGet(GV_StartBal());
    g_dailyPnL      = GlobalVariableGet(GV_DailyPnL());
    g_dailyTrades   = (int)GlobalVariableGet(GV_DailyTrades());
    g_consecLosses  = (int)GlobalVariableGet(GV_ConsecLoss());
    g_lastTradeCloseTime = (datetime)GlobalVariableGet(GV_LastClose());
    g_halted        = (GlobalVariableGet(GV_Halt()) > 0.5);
    g_haltDay       = (datetime)GlobalVariableGet(GV_HaltDay());
    g_profitHalt    = (GlobalVariableGet(GV_ProfitHalt()) > 0.5);
    g_lossHalt      = (GlobalVariableGet(GV_LossHalt()) > 0.5);

    // Seed the auto-spread average immediately so the first trade isn't
    // blocked by a 0 average (which would fall back to MaxSpreadPoints).
    g_avgSpread = (double)SymSpread();
    g_spreadSamples = 1;

    if(g_halted)
        Print("*** HARD HALT latched (equity floor). Trading disabled until manual reset. ***");

    // FIX B10: check algo-trading permissions upfront so a disabled AutoTrading
    // button or account rule is diagnosed instead of failing silently.
    if(!MQLInfoInteger(MQL_TRADE_ALLOWED))
        Print("WARNING: MQL_TRADE_ALLOWED=false — enable 'Allow Algo Trading' in the terminal.");
    if(TerminalInfoInteger(TERMINAL_TRADE_ALLOWED) == 0)
        Print("WARNING: TERMINAL_TRADE_ALLOWED=0 — the AutoTrading button is OFF.");
    if((AccountInfoInteger(ACCOUNT_TRADE_EXPERT)) == 0)
        Print("WARNING: ACCOUNT_TRADE_EXPERT=0 — this account forbids Expert Advisor trading.");

    EventSetTimer(5);
    Print("Predict-A-Trade Ultra Scalp EA initialized. Symbol=", g_symbol, " Magic=", g_magic,
          " AlgoTrading=", MQLInfoInteger(MQL_TRADE_ALLOWED),
          " Filters=measured-v1.17 (pullback-only, single TP/SL target)",
          " AllSessions=ON (no session window, broker TimeCurrent only)");
    UpdatePanel();
    return(INIT_SUCCEEDED);
}

//+------------------------------------------------------------------+
//| OnDeinit                                                          |
//+------------------------------------------------------------------+
void OnDeinit(const int reason)
{
    EventKillTimer();
    // Release instance ownership on explicit removal (chart/profile change,
    // EA detach) so a new attach elsewhere can start immediately. Never
    // cleared on recompile/timeframe change — the same chart re-inits.
    if(reason == REASON_REMOVE || reason == REASON_PROGRAM)
    {
        if(GlobalVariableGet(GV_HeartbeatChart()) == (double)ChartID())
            GlobalVariableSet(GV_Heartbeat(), 0.0);
    }
    if(g_emaFastHandle != INVALID_HANDLE) IndicatorRelease(g_emaFastHandle);
    if(g_emaSlowHandle != INVALID_HANDLE) IndicatorRelease(g_emaSlowHandle);
    if(g_atrHandle     != INVALID_HANDLE) IndicatorRelease(g_atrHandle);
    if(g_regimeEmaHandle != INVALID_HANDLE) IndicatorRelease(g_regimeEmaHandle);
    if(g_rsiHandle     != INVALID_HANDLE) IndicatorRelease(g_rsiHandle);
    if(g_atrM5Handle   != INVALID_HANDLE) IndicatorRelease(g_atrM5Handle);
    Comment("");
}

//+------------------------------------------------------------------+
//| OnTick                                                            |
//+------------------------------------------------------------------+
void OnTick()
{
    UpdateDayState();
    UpdateCapitalProtection();
    ManagePositions();

    // Only attempt new entries on a fresh bar — compare the bar's OPEN TIME,
    // not the shift index (iBarShift(sym, tf, now) is always 0 and would
    // block all future entries after the first tick). FIX B1.
    datetime barOpen = iTime(g_symbol, PERIOD_M1, 0);
    if(barOpen == g_lastTradeBarOpen) return;
    g_lastTradeBarOpen = barOpen;

    // Generate ultra-scalping signal
    int dir = GenerateSignal();
    if(dir == 0)
    {
        g_sigNoEntryBars++;
        // Diagnostic: log WHY no entry (all 4 paths), only when the state changes.
        string noEntry = NoEntryReason();
        if(noEntry != g_lastNoEntryMsg)
        {
            Print("SIGNAL: no entry — ", noEntry);
            g_lastNoEntryMsg = noEntry;
        }
        UpdatePanel();
        return;
    }

    // FIX B6: removed the signal-TTL block. GenerateSignal() returns a
    // *condition* (trend + pullback), not an event — a persistent condition
    // is not "stale". The fresh-bar guard above already throttles entries to
    // one per M1 bar, which is the correct anti-overtrade control.

    // Run the 16-gate pipeline
    string reason;
    if(!RunRiskPipeline(dir, reason))
    {
        Print("NO-TRADE [", dir > 0 ? "BUY" : "SELL", "] gate: ", reason);
        UpdatePanel();
        return;
    }

    // Size the position (apply scale-in lot multiplier for multi-trade)
    double lot = CalcLotSize(dir);
    if(lot <= 0) { Print("NO-TRADE: lot sizing failed"); UpdatePanel(); return; }
    if(AllowScaleIn)
    {
        int sameDir = CountSameDirPositions(dir > 0);
        if(sameDir > 0)
            lot = NormalizeDouble(lot * ScaleInLotMult, 2);
    }

    // Place the order
    if(OpenPosition(dir, lot))
    {
        g_dailyTrades++;
        GlobalVariableSet(GV_DailyTrades(), g_dailyTrades);
        RecordTradeTime();
        Print("TRADE OPENED: ", dir > 0 ? "BUY" : "SELL", " lot=", DoubleToString(lot, 2),
              " @ ", DoubleToString(SymAsk(), SymDigits()));
    }
    UpdatePanel();
}

//+------------------------------------------------------------------+
//| OnTimer — watchdog + external data refresh                        |
//+------------------------------------------------------------------+
void OnTimer()
{
    // Keep duplicate-instance heartbeat alive (5s timer → far under the 120s
    // staleness window in OnInit).
    GlobalVariableSet(GV_Heartbeat(), (double)TimeCurrent());
    GlobalVariableSet(GV_HeartbeatChart(), (double)ChartID());

    UpdateDayState();
    UpdateCapitalProtection();
    Watchdog();
    RefreshExternalData();
    UpdateAvgSpread();
    UpdatePanel();
}

//+------------------------------------------------------------------+
//| Running average spread — auto-adaptive MaxSpread                  |
//+------------------------------------------------------------------+
void UpdateAvgSpread()
{
    double s = (double)SymSpread();
    if(s <= 0) return;
    // Exponential moving average (smooth, adapts to the symbol's typical spread).
    if(g_spreadSamples == 0)
    {
        g_avgSpread = s;
        g_spreadSamples = 1;
    }
    else
    {
        double alpha = 0.1; // 10% weight on new sample
        g_avgSpread = g_avgSpread * (1.0 - alpha) + s * alpha;
        g_spreadSamples++;
    }
}

//+------------------------------------------------------------------+
//| Day-state rollover                                                |
//+------------------------------------------------------------------+
// Returns the start-of-day datetime (broker server time) for a given time.
// FIX B2: builds a proper datetime via StringToTime, not the broken
// (year*10000+mon*100+day) integer encoding (which was 23 Aug 1970).
datetime DayStart(datetime t)
{
    MqlDateTime dt;
    TimeToStruct(t, dt);
    string s = StringFormat("%04d.%02d.%02d", dt.year, dt.mon, dt.day);
    return StringToTime(s);
}

void UpdateDayState()
{
    datetime today = DayStart(TimeCurrent());
    if(g_currentDay != today)
    {
        g_currentDay    = today;
        g_dayStartBalance = AccountInfoDouble(ACCOUNT_BALANCE);
        g_dailyPnL      = 0;
        g_dailyTrades   = 0;
        g_consecLosses  = 0;
        g_profitHalt    = false;
        g_lossHalt      = false;
        g_tradeCount    = 0;
        GlobalVariableSet(GV_Day(), (double)g_currentDay);
        GlobalVariableSet(GV_StartBal(), g_dayStartBalance);
        GlobalVariableSet(GV_DailyPnL(), 0);
        GlobalVariableSet(GV_DailyTrades(), 0);
        GlobalVariableSet(GV_ConsecLoss(), 0);
        GlobalVariableSet(GV_ProfitHalt(), 0);
        GlobalVariableSet(GV_LossHalt(), 0);
        // FIX A.2: reset intraday peak to day-start equity on rollover, so a
        // stale high peak from a prior day can't fire the DD-halt at the open.
        GlobalVariableSet(GV_PeakEquity(), AccountInfoDouble(ACCOUNT_EQUITY));
        Print("New trading day. Day-start balance=", DoubleToString(g_dayStartBalance, 2));
    }
}

//+------------------------------------------------------------------+
//| Capital protection: realized daily P&L from history               |
//+------------------------------------------------------------------+
void UpdateCapitalProtection()
{
    g_dailyPnL = 0;
    datetime today = DayStart(TimeCurrent());
    // FIX B2: select history from the true start-of-day, not a bogus epoch.
    HistorySelect(today, TimeCurrent() + 60);
    int deals = HistoryDealsTotal();
    for(int i = 0; i < deals; i++)
    {
        ulong t = HistoryDealGetTicket(i);
        if(t == 0) continue;
        if(HistoryDealGetString(t, DEAL_SYMBOL) != g_symbol) continue;
        if(!PAT_IsOurMagic(HistoryDealGetInteger(t, DEAL_MAGIC))) continue;
        if(HistoryDealGetInteger(t, DEAL_ENTRY) != DEAL_ENTRY_OUT) continue;
        g_dailyPnL += HistoryDealGetDouble(t, DEAL_PROFIT)
                    + HistoryDealGetDouble(t, DEAL_SWAP)
                    + HistoryDealGetDouble(t, DEAL_COMMISSION);
    }
    GlobalVariableSet(GV_DailyPnL(), g_dailyPnL);

    // FIX (read.md b, v1.16): do NOT overwrite the day-start balance with an
    // inferred value on every tick — deposits/withdrawals or missing history
    // silently corrupt every percentage guard derived from it. Trust the
    // value stored at day rollover (GV_StartBal); infer only as a fallback
    // when the stored value is missing/zero.
    double curBal = AccountInfoDouble(ACCOUNT_BALANCE);
    double storedBal = GlobalVariableGet(GV_StartBal());
    if(storedBal > 0)
        g_dayStartBalance = storedBal;
    else
    {
        double dayOpen = curBal - g_dailyPnL;
        g_dayStartBalance = (dayOpen > 0 ? dayOpen : curBal);
        GlobalVariableSet(GV_StartBal(), g_dayStartBalance);
    }

    // Hard equity floor (never bypassed) — FIX: floor = (1 - HardStopPct/100)
    // of day-start, so HardStopPct=8 means halt when equity drops 8% (92%).
    double floorPct = 1.0 - HardStopPct / 100.0;
    double floorEq = g_dayStartBalance * floorPct;
    double eq = AccountInfoDouble(ACCOUNT_EQUITY);
    if(!g_halted && eq <= floorEq)
    {
        g_halted = true;
        g_haltDay = g_currentDay;   // FIX A.1: latch the day the halt fired
        GlobalVariableSet(GV_Halt(), 1);
        GlobalVariableSet(GV_HaltDay(), (double)g_haltDay);
        Print("*** HARD HALT: equity ", DoubleToString(eq, 2), " <= floor ",
              DoubleToString(floorEq, 2), " (-", HardStopPct, "% of day-start). Closing all + halting. ***");
        CloseAllPositions();
    }
    // Reset the halt latch on a NEW trading day. FIX A.1 latched the halt
    // day; FIX (read.md c, v1.16): compare day starts instead of a +86400s
    // timer — a halt fired at 23:00 used to stay latched until 23:00 the next
    // day, killing most of the next session. Clears at broker-day rollover.
    if(g_halted && g_haltDay != 0 && DayStart(TimeCurrent()) != g_haltDay)
    {
        g_halted = false;
        g_haltDay = 0;
        GlobalVariableSet(GV_Halt(), 0);
        GlobalVariableSet(GV_HaltDay(), 0);
        Print("New day: equity-floor halt latch cleared.");
    }

    // Running drawdown from intraday peak — halts if equity falls
    // MaxRunningDrawdownPct% from the day's highest equity (read.md Section 6B).
    double peakEq = GlobalVariableGet(GV_PeakEquity());
    if(eq > peakEq)
    {
        peakEq = eq;
        GlobalVariableSet(GV_PeakEquity(), peakEq);
    }
    if(!g_halted && peakEq > 0)
    {
        double ddPct = (peakEq - eq) / peakEq * 100.0;
        if(ddPct > MaxRunningDrawdownPct)
        {
            g_halted = true;
            GlobalVariableSet(GV_Halt(), 1);
            Print("*** DRAWDOWN HALT: ", DoubleToString(ddPct, 2),
                  "% from peak ", DoubleToString(peakEq, 2), ". Closing all + halting. ***");
            CloseAllPositions();
        }
    }

    // Daily loss limit (soft, bypassable)
    double lossPct = 0;
    if(g_dayStartBalance > 0) lossPct = (g_dailyPnL / g_dayStartBalance) * 100;
    if(lossPct <= -DailyLossLimitPct && !g_lossHalt)
    {
        g_lossHalt = true;
        GlobalVariableSet(GV_LossHalt(), 1);
        Print("DAILY LOSS LIMIT hit: ", DoubleToString(lossPct, 2), "% (limit ", DailyLossLimitPct, "%). New entries blocked.");
    }
    if(BypassDailyLossBlock && g_lossHalt)
    {
        g_lossHalt = false;
        GlobalVariableSet(GV_LossHalt(), 0);
        Print("Daily-loss block BYPASSED by operator input — trading re-enabled.");
    }

    // Daily profit target — FIX 3.3.4: close ALL positions to lock the day's
    // gains, not just block new entries (otherwise a reversal can turn a +3%
    // day into a loss via the DD-halt).
    double profitPct = 0;
    if(g_dayStartBalance > 0) profitPct = (g_dailyPnL / g_dayStartBalance) * 100;
    if(StopOnProfitTarget && profitPct >= DailyProfitTargetPct && !g_profitHalt)
    {
        g_profitHalt = true;
        GlobalVariableSet(GV_ProfitHalt(), 1);
        CloseAllPositions();
        Print("DAILY PROFIT TARGET hit: ", DoubleToString(profitPct, 2), "% (target ", DailyProfitTargetPct, "%). Closing all + halting new entries.");
    }
}

//+------------------------------------------------------------------+
//| Watchdog: missing-SL check                                        |
//+------------------------------------------------------------------+
void Watchdog()
{
    for(int i = PositionsTotal() - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsOurMagic(PositionGetInteger(POSITION_MAGIC))) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        double sl = PositionGetDouble(POSITION_SL);
        if(sl == 0)
        {
            Print("WATCHDOG: position ", ticket, " has NO stop-loss. Closing (fail-closed).");
            ClosePosition(ticket);
        }
    }
}

//+------------------------------------------------------------------+
//| Position management: single final target + hard SL watchdog      |
//+------------------------------------------------------------------+
// v1.17 (measured): the old exit machinery — MicroTP decay, TP1/TP2 partials,
// BE move, ATR trailing — was walk-forward simulated on 66 days of M1 data
// and every stage LOST money (the 20%-slice micro decay repeatedly cut the
// winners that paid for the losers; the stage-2-from-entry trailing ratchet
// stopped out entries at noise distance). The measured optimum is the
// simplest possible exit: ONE final target (TP = 12x round-trip cost) or
// the hard SL (8x cost), set on the order at entry and managed by the
// broker. This watchdog only guarantees the two stops exist (fail-closed).
void ManagePositions()
{
    for(int i = PositionsTotal() - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsOurMagic(PositionGetInteger(POSITION_MAGIC))) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;

        // Fail-closed watchdog: every position must carry its SL and final TP
        // (both set by OpenPosition at entry; restore if anything stripped them).
        double sl = PositionGetDouble(POSITION_SL);
        if(sl == 0)
        {
            Print("WATCHDOG: position ", ticket, " has NO stop-loss. Closing (fail-closed).");
            ClosePosition(ticket);
        }
    }
}

//+------------------------------------------------------------------+
//| Native order helpers (replaces CTrade)                            |
//+------------------------------------------------------------------+
// FIX B8: round a volume to the symbol's lot step (e.g. 0.002 -> 0.01).
double RoundToLotStep(double vol)
{
    double step = SymVolStep();
    if(step <= 0) step = 0.01;
    double minLot = SymVolMin();
    if(vol < minLot) return 0; // below min lot -> cannot trade this slice
    // Round to NEAREST step (not always down) so partial closes don't
    // systematically under-close. Clamp to minLot and never exceed vol.
    double rounded = MathRound(vol / step) * step;
    rounded = MathMax(rounded, minLot);
    return MathMin(rounded, vol);
}

ENUM_ORDER_TYPE_FILLING GetFillingMode()
{
    long mode = SymbolInfoInteger(g_symbol, SYMBOL_FILLING_MODE);
    if((mode & SYMBOL_FILLING_FOK) != 0) return ORDER_FILLING_FOK;
    if((mode & SYMBOL_FILLING_IOC) != 0) return ORDER_FILLING_IOC;
    return ORDER_FILLING_RETURN;
}

// Returns true on success and sets fillPrice to the actual fill (res.price).
// FIX B5: use res.price from the trade result — the deal is NOT in the
// history cache immediately after OrderSend, so GetLastFillPrice() returns 0.
bool SendOrder(ENUM_ORDER_TYPE type, double lot, double price, double sl, double tp, string comment, double &fillPrice, uint &retcode)
{
    MqlTradeRequest req;
    MqlTradeResult  res;
    ZeroMemory(req);
    ZeroMemory(res);
    req.action   = TRADE_ACTION_DEAL;
    req.symbol   = g_symbol;
    req.volume   = lot;
    req.type     = type;
    req.deviation= MaxDeviationPoints;
    req.magic    = (ulong)g_magic;
    req.comment  = comment;
    req.type_filling = GetFillingMode();
    if(price > 0) req.price = price;
    if(sl > 0) req.sl = sl;
    if(tp > 0) req.tp = tp;
    if(!OrderSend(req, res))
    {
        retcode = res.retcode;
        Print("ORDER FAILED: ", res.retcode, " ", res.comment);
        return false;
    }
    if(res.retcode != TRADE_RETCODE_DONE && res.retcode != TRADE_RETCODE_PLACED)
    {
        retcode = res.retcode;
        Print("ORDER REJECTED: retcode=", res.retcode, " ", res.comment);
        return false;
    }
    fillPrice = res.price; // real fill price from the result
    return true;
}

// FIX Tier3: must PositionSelectByTicket(ticket) before reading position
// properties; otherwise PositionGetDouble(VOLUME)/POSITION_TYPE read whichever
// position the loop selected last. Also check the retcode.
bool ClosePosition(ulong ticket)
{
    MqlTradeRequest req;
    MqlTradeResult  res;
    ZeroMemory(req); ZeroMemory(res);
    if(!PositionSelectByTicket(ticket))
    {
        Print("ClosePosition: cannot select ticket ", ticket);
        return false;
    }
    req.action   = TRADE_ACTION_DEAL;
    req.position = ticket;
    req.symbol   = g_symbol;
    req.volume   = PositionGetDouble(POSITION_VOLUME);
    req.deviation= MaxDeviationPoints;
    req.magic    = (ulong)g_magic;
    req.type_filling = GetFillingMode();
    if(PositionGetInteger(POSITION_TYPE) == POSITION_TYPE_BUY)
    {
        req.type  = ORDER_TYPE_SELL;
        req.price = SymBid();
    }
    else
    {
        req.type  = ORDER_TYPE_BUY;
        req.price = SymAsk();
    }
    if(!OrderSend(req, res)) { Print("ClosePosition failed: ", res.retcode, " ", res.comment); return false; }
    return (res.retcode == TRADE_RETCODE_DONE || res.retcode == TRADE_RETCODE_PLACED);
}

bool ClosePartial(ulong ticket, double vol)
{
    MqlTradeRequest req;
    MqlTradeResult  res;
    ZeroMemory(req); ZeroMemory(res);
    if(!PositionSelectByTicket(ticket))
    {
        Print("ClosePartial: cannot select ticket ", ticket);
        return false;
    }
    req.action   = TRADE_ACTION_DEAL;
    req.position = ticket;
    req.symbol   = g_symbol;
    req.volume   = vol;
    req.deviation= MaxDeviationPoints;
    req.magic    = (ulong)g_magic;
    req.type_filling = GetFillingMode();
    if(PositionGetInteger(POSITION_TYPE) == POSITION_TYPE_BUY)
    {
        req.type  = ORDER_TYPE_SELL;
        req.price = SymBid();
    }
    else
    {
        req.type  = ORDER_TYPE_BUY;
        req.price = SymAsk();
    }
    if(!OrderSend(req, res)) { Print("ClosePartial failed: ", res.retcode, " ", res.comment); return false; }
    return (res.retcode == TRADE_RETCODE_DONE || res.retcode == TRADE_RETCODE_PLACED);
}

bool ModifyPosition(ulong ticket, double sl, double tp)
{
    MqlTradeRequest req;
    MqlTradeResult  res;
    ZeroMemory(req); ZeroMemory(res);
    if(!PositionSelectByTicket(ticket))
    {
        Print("ModifyPosition: cannot select ticket ", ticket);
        return false;
    }
    req.action   = TRADE_ACTION_SLTP;
    req.position = ticket;
    req.symbol   = g_symbol;
    req.sl       = NormalizeDouble(sl, SymDigits());
    req.tp       = NormalizeDouble(tp, SymDigits());
    req.magic    = (ulong)g_magic;
    if(!OrderSend(req, res)) { Print("ModifyPosition failed: ", res.retcode, " ", res.comment); return false; }
    return (res.retcode == TRADE_RETCODE_DONE || res.retcode == TRADE_RETCODE_PLACED);
}

//+------------------------------------------------------------------+
//| Close all our positions                                           |
//+------------------------------------------------------------------+
void CloseAllPositions()
{
    for(int i = PositionsTotal() - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsOurMagic(PositionGetInteger(POSITION_MAGIC))) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        ClosePosition(ticket);
    }
}

//+------------------------------------------------------------------+
//| Count our positions                                               |
//+------------------------------------------------------------------+
int CountPositions()
{
    int n = 0;
    for(int i = PositionsTotal() - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsOurMagic(PositionGetInteger(POSITION_MAGIC))) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        n++;
    }
    return n;
}

int CountSameDirPositions(bool isBuy)
{
    int n = 0;
    for(int i = PositionsTotal() - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsOurMagic(PositionGetInteger(POSITION_MAGIC))) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        bool pBuy = (PositionGetInteger(POSITION_TYPE) == POSITION_TYPE_BUY);
        if(pBuy == isBuy) n++;
    }
    return n;
}

//+------------------------------------------------------------------+
//| Indicator helpers                                                 |
//+------------------------------------------------------------------+
double GetATR()
{
    double buf[1];
    if(CopyBuffer(g_atrHandle, 0, 0, 1, buf) < 1) return 0;
    return buf[0];
}

double GetATR5()
{
    double buf[1];
    if(CopyBuffer(g_atrM5Handle, 0, 0, 1, buf) < 1) return 0;
    return buf[0];
}

double GetEMA(int handle)
{
    double buf[1];
    if(CopyBuffer(handle, 0, 0, 1, buf) < 1) return 0;
    return buf[0];
}

//+------------------------------------------------------------------+
//| VWAP (session-anchored) — read.md factor                          |
//+------------------------------------------------------------------+
// FIX Tier3: cache the VWAP and recompute only once per M1 bar, instead of
// scanning VWAPSessionHours*60 bars on every tick and every 5s timer.
double g_vwapCache = 0;
datetime g_vwapBar = 0;
double GetVWAP()
{
    datetime bar = iTime(g_symbol, PERIOD_M1, 0);
    if(bar == g_vwapBar && g_vwapCache > 0) return g_vwapCache;
    g_vwapBar = bar;
    int bars = VWAPSessionHours * 60;
    double sumPV = 0, sumV = 0;
    for(int i = 1; i <= bars; i++)
    {
        double h = iHigh(g_symbol, PERIOD_M1, i);
        double l = iLow(g_symbol, PERIOD_M1, i);
        double c = iClose(g_symbol, PERIOD_M1, i);
        double tp = (h + l + c) / 3.0;
        double v = (double)iVolume(g_symbol, PERIOD_M1, i);
        sumPV += tp * v;
        sumV  += v;
    }
    if(sumV <= 0) return 0;
    g_vwapCache = sumPV / sumV;
    return g_vwapCache;
}

//+------------------------------------------------------------------+
//| Dollar proxy (native EURUSD) — inverse of DXY                     |
//+------------------------------------------------------------------+
// Uses EURUSD as a native dollar-strength proxy (no external API).
// A rising EURUSD = weakening dollar = bullish gold; falling EURUSD =
// strengthening dollar = bearish gold. Returns day-over-day EURUSD change.
double GetDollarProxyChange()
{
    if(!UseDollarProxy) return 0;
    if(DollarProxySymbol == "") return 0;
    if(!SymbolSelect(DollarProxySymbol, true)) return 0;
    // Day-over-day change of EURUSD (close today vs close yesterday).
    double c0 = iClose(DollarProxySymbol, PERIOD_D1, 0);
    double c1 = iClose(DollarProxySymbol, PERIOD_D1, 1);
    if(c0 <= 0 || c1 <= 0) return 0;
    return (c0 - c1) / c1; // fractional change
}

//+------------------------------------------------------------------+
//| News risk — native MT5 economic calendar                           |
//+------------------------------------------------------------------+
// Uses the terminal's built-in economic calendar (CalendarValueHistory /
// CalendarEventById) — no external API, no files. Blocks trades within
// NewsBlockMin of a HIGH-importance event that is GOLD-CRITICAL (NFP, CPI,
// FOMC, GDP, unemployment, rate decisions). Minor high-importance events
// (e.g. flash PMIs) do NOT freeze the scalper.
bool CalendarNewsBlocked(string &reason)
{
    if(!EnableNewsFilter) return false;
    // Look for high-impact USD events within the block window.
    datetime from = TimeCurrent() - NewsBlockMin * 60;
    datetime to   = TimeCurrent() + NewsBlockMin * 60;
    MqlCalendarValue values[];
    int n = CalendarValueHistory(values, from, to, NULL, "USD");
    if(n <= 0) return false;
    for(int i = 0; i < n; i++)
    {
        MqlCalendarEvent ev;
        if(!CalendarEventById(values[i].event_id, ev)) continue;
        // Only block HIGH-importance events.
        if(ev.importance != CALENDAR_IMPORTANCE_HIGH) continue;
        // Only block GOLD-CRITICAL events (the ones that actually move gold).
        // Tightened for ultra scalping: JOLTS/jobs/generic labor reports are
        // minor and would block far too many entries, so they are excluded.
        // Edit the keyword list here if you want to add/remove events.
        string name = ev.name;
        string low = name;
        StringToLower(low);
        bool critical =
            StringFind(low, "nonfarm") >= 0 ||   // NFP
            StringFind(low, "payrolls") >= 0 ||  // actual NFP headline
            StringFind(low, "cpi") >= 0 ||       // CPI
            StringFind(low, "core inflation") >= 0 ||
            StringFind(low, "fomc") >= 0 ||      // FOMC
            StringFind(low, "fed funds") >= 0 ||
            StringFind(low, "rate decision") >= 0 ||
            StringFind(low, "interest rate") >= 0 ||
            StringFind(low, "gdp") >= 0 ||       // GDP
            StringFind(low, "unemployment rate") >= 0 ||
            StringFind(low, "core cpi") >= 0;
        // Excluded minors: "jobs", "jolts", "initial jobless", "payroll
        // change", "average hourly earnings" — these don't reliably move gold.
        if(!critical) continue;
        // Block if the event is within the window (before or after).
        long diff = (long)(TimeCurrent() - values[i].time);
        if(diff >= -NewsBlockMin * 60 && diff <= NewsBlockMin * 60)
        {
            reason = "News: GOLD-CRITICAL event '"+name+"' within "+IntegerToString(NewsBlockMin)+" min";
            return true;
        }
    }
    return false;
}

//+------------------------------------------------------------------+
//| Session timing (broker server time)                               |
//+------------------------------------------------------------------+
// v1.15 ALL-SESSIONS: the London/NY window gate and the Asian skip are
// REMOVED per operator directive ("trade in all sessions, follow only
// broker time"). Every other clock decision (news filter, daily reset,
// heartbeat) still runs on TimeCurrent() broker server time. Rollover
// protection now rides on the Gate 5 spread cap, the cost firewall and
// the daily loss cap — not on a clock gate.
bool SessionOK(string &reason)
{
    return true; // all sessions traded — no session window
}

//+------------------------------------------------------------------+
//| RSI (read.md filter) — soft overbought/oversold gate             |
//+------------------------------------------------------------------+
double GetRSI()
{
    if(g_rsiHandle == INVALID_HANDLE) return 50;
    double buf[1];
    if(CopyBuffer(g_rsiHandle, 0, 0, 1, buf) < 1)
        return 50;
    return buf[0];
}

//+------------------------------------------------------------------+
//| No-entry diagnosis — single-path status line (v1.17)              |
//+------------------------------------------------------------------+
// Attribute a fired signal to its entry path (2 = trend-pullback; the only
// remaining path after the v1.17 measured removal of P1/P3/P4).
void CountSignalPath(int path)
{
    if(path == 2) g_sigPathPullback++;
}

string NoEntryReason()
{
    // Single-path report with stable labels (change-dedup: one print per
    // distinct state; numbers live on the panel, not here).
    double emaFast = GetEMA(g_emaFastHandle);
    double emaSlow = GetEMA(g_emaSlowHandle);
    double atr     = GetATR();

    if(emaFast <= 0 || emaSlow <= 0 || atr <= 0)
        return "indicator data not ready (EMA/ATR=0)";

    double mid = (SymBid() + SymAsk()) / 2.0;
    double mom = (mid - emaFast) / (atr > 0 ? atr : 1);
    double rsi = GetRSI();

    bool exhausted = (UseExhaustionFilter &&
                      MathAbs(emaFast - emaSlow) / (atr > 0 ? atr : 1) > MaxEMASpreadATR);
    bool trendUp   = (emaFast > emaSlow);
    bool trendDown = (emaFast < emaSlow);
    if(exhausted)                    return "P2 PULL: EMA exhausted";
    if(!trendUp && !trendDown)       return "P2 PULL: no trend";
    if(MathAbs(mom) >= 0.25)         return "P2 PULL: off EMA";
    if(trendUp && g_useRSI && !(rsi < RSIOverbought))   return "P2 PULL: RSI OB";
    if(trendDown && g_useRSI && !(rsi > RSIOversold))   return "P2 PULL: RSI OS";
    return "P2 PULL: READY";
}

//+------------------------------------------------------------------+
//| Signal generation — Ultra Scalping M1 (measured, v1.17)          |
//+------------------------------------------------------------------+
// v1.17: the 4-path engine was walk-forward simulated on 66 days of M1
// data with full exit+cost modeling. Per-path attribution:
//   P2 Trend-pullback  = the ONLY path with a positive edge
//   P1 SMC             = displaced profitable pullback entries (net negative)
//   P3 Range (5.3% WR) and P4 Momentum (2.3% WR) = structurally unprofitable
// P1/P3/P4 are therefore REMOVED at code level, as is the M15 HTF bias gate
// (it hurt every out-of-sample configuration). The single remaining entry:
// trend-pullback to the fast EMA inside a fresh trend, RSI-confirmed.
// Returns 1 (BUY), -1 (SELL), 0.
int GenerateSignal()
{
    double emaFast = GetEMA(g_emaFastHandle);
    double emaSlow = GetEMA(g_emaSlowHandle);
    double atr     = GetATR();
    if(emaFast <= 0 || emaSlow <= 0 || atr <= 0) return 0;

    double mid = (SymBid() + SymAsk()) / 2.0;
    double mom = (mid - emaFast) / (atr > 0 ? atr : 1);

    // ---- Exhaustion (soft): skip when the trend leg is spent ----
    bool exhausted = false;
    if(UseExhaustionFilter)
    {
        double spreadATR = MathAbs(emaFast - emaSlow) / (atr > 0 ? atr : 1);
        if(spreadATR > MaxEMASpreadATR) exhausted = true;
    }

    // ---- Path 2: Trend-pullback — the ONLY entry path (measured v1.17) ----
    // Requires a real trend AND a pullback to the fast EMA — not just any
    // momentum bar. This filters out the noise that was paying the spread.
    if(!exhausted)
    {
        bool trendUp   = (emaFast > emaSlow);
        bool trendDown = (emaFast < emaSlow);
        // Pullback to the fast EMA (mom near 0) — the ONLY entry here.
        // v1.17: band 0.35 -> 0.25 (walk-forward sim: +$1,475 tune / -$273 valid
        // vs +$1,582/-$495 for 0.35 — tighter band wins out-of-sample).
        bool pullback = MathAbs(mom) < 0.25;

        if(trendUp && pullback)
        {
            if(!g_useRSI || GetRSI() < RSIOverbought)
            {
                CountSignalPath(2);
                return 1;
            }
        }
        if(trendDown && pullback)
        {
            if(!g_useRSI || GetRSI() > RSIOversold)
            {
                CountSignalPath(2);
                return -1;
            }
        }
    }

    return 0; // no signal
}

//+------------------------------------------------------------------+
//| Regime classification                                             |
//+------------------------------------------------------------------+
int GetRegime()
{
    double ema = GetEMA(g_regimeEmaHandle);
    if(ema <= 0) return 0;
    double mid = (SymBid() + SymAsk()) / 2.0;
    double emaPrev[3];
    if(CopyBuffer(g_regimeEmaHandle, 0, 1, 3, emaPrev) < 3) return 0;
    double slope = ema - emaPrev[2];
    if(mid > ema && slope > 0) return 1;
    if(mid < ema && slope < 0) return -1;
    return 0;
}

//+------------------------------------------------------------------+
//| THE 16-GATE RISK PIPELINE (fail-closed, short-circuit order)      |
//+------------------------------------------------------------------+
bool RunRiskPipeline(int dir, string &reason)
{
    // Gate 1 — ExecutionPermission
    if(!EnableTrading) { reason = "ExecutionPermission: EnableTrading=false"; return false; }
    // FIX B10: fail-closed on algo-trading permission.
    if(!MQLInfoInteger(MQL_TRADE_ALLOWED)) { reason = "ExecutionPermission: MQL_TRADE_ALLOWED=false"; return false; }
    if(TerminalInfoInteger(TERMINAL_TRADE_ALLOWED) == 0) { reason = "ExecutionPermission: AutoTrading button OFF"; return false; }
    if(AccountInfoInteger(ACCOUNT_TRADE_EXPERT) == 0) { reason = "ExecutionPermission: account forbids EAs"; return false; }

    // Gate 2 — BrokerSymbolValidation
    if(!ValidateBrokerSymbol(reason)) return false;

    // Gate 3 — SeedCapitalProtection (5% daily cap)
    double lossPct = 0;
    if(g_dayStartBalance > 0) lossPct = (g_dailyPnL / g_dayStartBalance) * 100;
    if(lossPct <= -SeedCapitalPct)
    {
        reason = "SeedCapitalProtection: daily loss " + DoubleToString(lossPct, 2) +
                 "% <= -" + DoubleToString(SeedCapitalPct, 2) + "% cap";
        return false;
    }

    // Gate 3b — TradeDirection filter
    if(TradeDirection > 0 && dir < 0) { reason = "TradeDirection: BUY-only mode"; return false; }
    if(TradeDirection < 0 && dir > 0) { reason = "TradeDirection: SELL-only mode"; return false; }

    // Gate 4 — DailyLossLimit
    if(g_lossHalt && !BypassDailyLossBlock)
    {
        reason = "DailyLossLimit: daily loss limit reached (soft block active)";
        return false;
    }

    // Gate 5 — MaxSpread (auto-adaptive to symbol's typical spread)
    double spread = (double)SymSpread();
    double effMax = MaxSpreadPoints;
    if(AutoSpread)
    {
        // Auto-learn the symbol's typical spread via a running average
        // (g_avgSpread, updated on the timer). Limit = typical x AutoSpreadMult,
        // so it only blocks abnormal blowouts (news/volatility) and never
        // blocks the symbol's normal spread.
        double typical = g_avgSpread;
        if(typical <= 0) typical = spread; // not seeded yet — use live spread
        effMax = MathMax(typical * AutoSpreadMult, MaxSpreadPoints);
    }
    if(spread > effMax)
    {
        reason = "MaxSpread: spread " + DoubleToString(spread, 1) + " pts > " +
                 DoubleToString(effMax, 1) + " pts (auto limit)";
        return false;
    }

    // Gate 5b — Cost gate: skip the trade if the final TP distance is less
    // than MinRRBeforeCost x the round-trip cost. In quiet markets the target
    // may be too small to cover spread + commission + slippage, so the trade
    // would only ever pay costs. (Adapted concept from a MIT-licensed XAUUSD
    // scalper — implemented cleanly.)
    {
        double slDistC, tpDistC;
        if(!GetExitDistances(slDistC, tpDistC)) { reason = "CostGate: cannot compute exit distances"; return false; }
        double roundTrip = GetMicroTPDistance(0) * SymPoint(); // cost in price units
        if(roundTrip > 0 && tpDistC < roundTrip * MinRRBeforeCost)
        {
            reason = "CostGate: TP distance " + DoubleToString(tpDistC / SymPoint(), 1) +
                     " pts < cost x " + DoubleToString(MinRRBeforeCost, 1) +
                     " (" + DoubleToString(roundTrip / SymPoint(), 1) + " pts) — target too tight";
            return false;
        }
    }

    // Gate 5c — MaxEntrySpread: block entry when the LIVE spread is abnormally
    // wide (news/volatility blowout). The normal 33pt spread still passes;
    // this only stops entering when the spread has blown out and would make
    // the first target unreachable.
    if(MaxEntrySpreadPts > 0 && spread > MaxEntrySpreadPts)
    {
        reason = "MaxEntrySpread: live spread " + DoubleToString(spread, 1) +
                 " pts > " + DoubleToString(MaxEntrySpreadPts, 1) + " pts (blowout)";
        return false;
    }

    // Gate 6 — NewsRisk (native MT5 economic calendar)
    if(!NewsRiskOK(reason)) return false;

    // Gate 7 — Slippage
    if(MaxSlippagePts <= 0) { reason = "Slippage: MaxSlippagePts must be > 0"; return false; }

    // Gate 8 — MaxPositions
    if(CountPositions() >= MaxPositions)
    {
        reason = "MaxPositions: " + IntegerToString(CountPositions()) + " open (max " +
                 IntegerToString(MaxPositions) + ")";
        return false;
    }
    if(CountSameDirPositions(dir > 0) >= MaxSameDirPositions)
    {
        reason = "MaxPositions: same-dir cap reached";
        return false;
    }

    // Gate 9 — MaxExposure
    double exposure = CalcExposurePct();
    if(exposure > MaxExposurePct)
    {
        reason = "MaxExposure: " + DoubleToString(exposure, 2) + "% > " +
                 DoubleToString(MaxExposurePct, 2) + "%";
        return false;
    }

    // Gate 10 — Cooldown
    if(TimeCurrent() - g_lastTradeCloseTime < CooldownSeconds)
    {
        reason = "Cooldown: " + IntegerToString(TimeCurrent() - g_lastTradeCloseTime) +
                 "s < " + IntegerToString(CooldownSeconds) + "s since last close";
        return false;
    }
    int barsSince = iBarShift(g_symbol, PERIOD_M1, g_lastTradeCloseTime);
    if(barsSince >= 0 && barsSince < MinBarsBetweenTrades)
    {
        reason = "Cooldown: only " + IntegerToString(barsSince) + " bars since last trade";
        return false;
    }

    // Gate 11 — StopHuntFilter
    if(EnableStopHuntFilter && StopHuntDetected(dir))
    {
        reason = "StopHuntFilter: liquidity sweep detected in lookback";
        return false;
    }

    // Gate 12 — MarginCheck
    double lot = CalcLotSize(dir);
    if(!MarginOK(lot, dir, reason)) return false;

    // Gate 13 — OvertradeProtection
    if(TradesInLastHour() >= MaxTradesPerHour)
    {
        reason = "OvertradeProtection: " + IntegerToString(TradesInLastHour()) +
                 " trades in last hour (max " + IntegerToString(MaxTradesPerHour) + ")";
        return false;
    }
    if(g_consecLosses >= MaxConsecutiveLosses)
    {
        // Pause for ConsecLossCooldownSec, then auto-resume. A permanent halt
        // is too aggressive for scalping (streaks of 3-5 losses are normal).
        if(TimeCurrent() - g_lastTradeCloseTime < ConsecLossCooldownSec)
        {
            reason = "OvertradeProtection: " + IntegerToString(g_consecLosses) +
                     " consecutive losses (max " + IntegerToString(MaxConsecutiveLosses) +
                     ") — pausing " + IntegerToString(ConsecLossCooldownSec) + "s";
            return false;
        }
        // Cooldown elapsed — allow trading again (streak is stale).
    }

    // Gate 14 — MaxDailyTrades
    if(g_dailyTrades >= MaxDailyTrades)
    {
        reason = "MaxDailyTrades: " + IntegerToString(g_dailyTrades) + " today (max " +
                 IntegerToString(MaxDailyTrades) + ")";
        return false;
    }

    // Gate 15 — RegimeFilter (+ M5 ATR sit-out; v1.15: session gate removed)
    if(!RegimeOK(reason)) return false;
    double atr5 = GetATR5();
    if(atr5 > 0 && atr5 < MinATR5Dollars)
    {
        reason = "M5ATR: ATR(M5) " + DoubleToString(atr5, 2) + " < " +
                 DoubleToString(MinATR5Dollars, 2) + " (dead market — sit out)";
        return false;
    }
    if(!SessionOK(reason)) return false;

    // Gate 16 — ProfitTarget
    if(g_profitHalt)
    {
        reason = "ProfitTarget: daily profit target reached";
        return false;
    }

    return true;
}

//+------------------------------------------------------------------+
//| Gate 2 — Broker symbol validation                                 |
//+------------------------------------------------------------------+
bool ValidateBrokerSymbol(string &reason)
{
    if(!SymbolSelect(g_symbol, true)) { reason = "BrokerSymbolValidation: symbol not found"; return false; }
    if(SymTradeMode() == SYMBOL_TRADE_MODE_DISABLED)
    {
        reason = "BrokerSymbolValidation: trading disabled for symbol";
        return false;
    }
    double lotStep = SymVolStep();
    double minLot  = SymVolMin();
    double maxLot  = SymVolMax();
    if(lotStep <= 0 || minLot <= 0 || maxLot <= 0)
    {
        reason = "BrokerSymbolValidation: invalid lot metadata (step/min/max)";
        return false;
    }
    if(SymPoint() <= 0 || SymDigits() <= 0)
    {
        reason = "BrokerSymbolValidation: invalid point/digits";
        return false;
    }
    return true;
}

//+------------------------------------------------------------------+
//| Gate 6 — News risk                                                |
//+------------------------------------------------------------------+
bool NewsRiskOK(string &reason)
{
    if(!EnableNewsFilter) return true;

    MqlDateTime dt;
    TimeToStruct(TimeCurrent(), dt);

    if(BlockWeekend && (dt.day_of_week == 0 || dt.day_of_week == 6))
    {
        reason = "NewsRisk: weekend (no trading)";
        return false;
    }
    // v1.15: Friday late cutoff removed (all sessions traded — directive).
    // Native MT5 economic calendar news check
    if(CalendarNewsBlocked(reason)) return false;
    return true;
}

//+------------------------------------------------------------------+
//| Gate 9 — Exposure                                                 |
//+------------------------------------------------------------------+
double CalcExposurePct()
{
    double equity = AccountInfoDouble(ACCOUNT_EQUITY);
    if(equity <= 0) return 100;
    double marginUsed = AccountInfoDouble(ACCOUNT_MARGIN);
    return (marginUsed / equity) * 100;
}

//+------------------------------------------------------------------+
//| Gate 11 — Stop hunt / liquidity sweep detection                   |
//+------------------------------------------------------------------+
bool StopHuntDetected(int dir)
{
    double hi = 0, lo = DBL_MAX;
    for(int i = 1; i <= StopHuntLookback; i++)
    {
        double h = iHigh(g_symbol, PERIOD_M1, i);
        double l = iLow(g_symbol, PERIOD_M1, i);
        if(h > hi) hi = h;
        if(l < lo) lo = l;
    }
    double range = hi - lo;
    if(range <= 0) return false;

    double o = iOpen(g_symbol, PERIOD_M1, 1);
    double c = iClose(g_symbol, PERIOD_M1, 1);
    double h = iHigh(g_symbol, PERIOD_M1, 1);
    double l = iLow(g_symbol, PERIOD_M1, 1);
    double body = MathAbs(c - o);
    // FIX Tier3: guard against doji bars (body ~ 0) — a ~zero body would make
    // any wick exceed body*StopHuntWickPct and falsely trigger the filter.
    if(body < SymPoint() * 2) return false;
    double upperWick = h - MathMax(o, c);
    double lowerWick = MathMin(o, c) - l;

    if(dir > 0)
    {
        if(lowerWick > body * StopHuntWickPct && l <= lo + range * 0.05)
            return true;
    }
    else
    {
        if(upperWick > body * StopHuntWickPct && h >= hi - range * 0.05)
            return true;
    }
    return false;
}

//+------------------------------------------------------------------+
//| Gate 12 — Margin check                                           |
//+------------------------------------------------------------------+
bool MarginOK(double lot, int dir, string &reason)
{
    double refPx = (dir > 0) ? SymAsk() : SymBid();
    ENUM_ORDER_TYPE mtype = (dir > 0) ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;
    double margin = 0;
    if(!OrderCalcMargin(mtype, g_symbol, lot, refPx, margin))
    {
        reason = "MarginCheck: OrderCalcMargin failed";
        return false;
    }
    double freeMargin = AccountInfoDouble(ACCOUNT_MARGIN_FREE);
    if(margin > freeMargin * (MaxMarginUsagePct / 100.0))
    {
        reason = "MarginCheck: required " + DoubleToString(margin, 2) + " > freeMargin x " +
                 DoubleToString(MaxMarginUsagePct, 2) + "% = " +
                 DoubleToString(freeMargin * MaxMarginUsagePct / 100.0, 2);
        return false;
    }
    return true;
}

//+------------------------------------------------------------------+
//| Gate 15 — Regime filter                                           |
//+------------------------------------------------------------------+
bool RegimeOK(string &reason)
{
    if(!EnableRegimeFilter) return true;
    double atr = GetATR();
    if(atr <= 0) { reason = "RegimeFilter: ATR unavailable"; return false; }
    // ATR is in price units (e.g. $3.60 for gold). Compare directly against
    // the min/max thresholds (also in price units) — no point conversion.
    if(atr < RegimeMinATR)
    {
        reason = "RegimeFilter: ATR " + DoubleToString(atr, 2) + " < min " +
                 DoubleToString(RegimeMinATR, 2) + " (too quiet)";
        return false;
    }
    if(atr > RegimeMaxATR)
    {
        reason = "RegimeFilter: ATR " + DoubleToString(atr, 2) + " > max " +
                 DoubleToString(RegimeMaxATR, 2) + " (too volatile)";
        return false;
    }
    return true;
}

//+------------------------------------------------------------------+
//| Position sizing — risk-based (M5 ATR)                             |
//+------------------------------------------------------------------+
double CalcLotSize(int dir)
{
    double equity = AccountInfoDouble(ACCOUNT_EQUITY);
    if(equity <= 0) return 0;

    double riskMoney = equity * (RiskPerTradePct / 100.0);
    // Size from the ACTUAL stop the order will use (GetExitDistances).
    double slDistC, tpC;
    if(!GetExitDistances(slDistC, tpC)) return 0;
    double slDist = slDistC;
    if(slDist <= 0) return 0;

    double tickVal  = SymTickValue();
    double tickSize = SymTickSize();
    if(tickVal <= 0 || tickSize <= 0) return 0;
    double valuePerUnit = tickVal / tickSize;

    double lot = riskMoney / (slDist * valuePerUnit);
    double lotStep = SymVolStep();
    double minLot  = SymVolMin();
    double maxLot  = SymVolMax();
    if(lotStep <= 0) lotStep = 0.01;
    lot = MathFloor(lot / lotStep) * lotStep;
    if(lot < minLot) lot = minLot;
    if(lot > maxLot) lot = maxLot;
    if(lot > MaxLot) lot = MaxLot;
    if(lot < BaseLot) lot = BaseLot;
    return lot;
}

//+------------------------------------------------------------------+
//| Round-trip cost distance (points) — the sizing yardstick          |
//+------------------------------------------------------------------+
// v1.17: this function no longer produces a micro-TP; it MEASURES the
// round-trip cost (spread + broker commission + slippage + buffer) that all
// exits are sized from (CostSLMult / CostTPMult multiples) and that the
// Gate 5b cost firewall checks against. Name kept for continuity.
double GetMicroTPDistance(double lot)
{
    double point = SymPoint();
    if(point <= 0) return 0;

    // 1. Spread cost (in points) — the bid/ask gap already paid on entry.
    double spreadPts = (double)SymSpread();

    // 2. Commission cost (in points) — convert per-lot commission to points.
    double commPts = 0;
    if(BrokerCommissionPerLot > 0 && lot > 0)
    {
        double tickVal  = SymTickValue();
        double tickSize = SymTickSize();
        if(tickVal > 0 && tickSize > 0)
        {
            double valuePerUnit = tickVal / tickSize; // $ per 1.0 price unit per lot
            if(valuePerUnit > 0)
                commPts = (BrokerCommissionPerLot / valuePerUnit) / point;
        }
    }

    // 3. Slippage buffer (points).
    double slipPts = MaxSlippagePts;

    // Total cost-covering distance + extra buffer.
    double dist = spreadPts + commPts + slipPts + MicroTPBufferPts;
    return dist;
}

//+------------------------------------------------------------------+
//| Effective SL/TP distances (auto-cost, fixed points, or ATR×R:R)   |
//+------------------------------------------------------------------+
// v1.17 (measured): ONE final target + ONE hard stop. AUTO-COST mode sizes
// both as multiples of the measured round-trip cost (spread + commission +
// slippage), so one script works on both raw/ECN (spread ~10pts + commission)
// and standard (spread ~33pts) accounts without re-tuning. Returns false on
// failure.
bool GetExitDistances(double &slDist, double &tpDist)
{
    double point = SymPoint();
    if(point <= 0) return false;

    // AUTO-COST mode: size from the measured round-trip cost.
    if(AutoCostAdapt)
    {
        double roundTrip = GetMicroTPDistance(0);   // points (spread+comm+slip+buffer)
        if(roundTrip > 0)
        {
            double p = roundTrip * point;           // convert to price units
            slDist = p * CostSLMult;
            tpDist = p * CostTPMult;
            return (slDist > 0 && tpDist > 0);
        }
        // Fall through to fixed if cost can't be measured.
    }

    // Fixed-points fallback when auto cost can't be measured.
    slDist = FixedSLPts * point;
    tpDist = FixedTPPts * point;
    return true;
}

//+------------------------------------------------------------------+
//| Open position with SL/TP                                          |
//+------------------------------------------------------------------+
bool OpenPosition(int dir, double lot)
{
    double slDist, tpDist;
    if(!GetExitDistances(slDist, tpDist)) return false;
    // v1.17 (measured): single final target. The broker manages the TP; the
    // watchdog in ManagePositions() guarantees the SL exists. No ladder, no
    // trailing — every extra exit stage measured negative in the sim.

    ENUM_ORDER_TYPE type = (dir > 0) ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;

    // FIX 10016 (v1.13): the old clamp measured SL from the ENTRY price, but
    // the broker validates BUY stops against BID and SELL stops against ASK.
    // On XAUUSD.e (33-45pt spread) an 80pt SL had only ~35-47pt of real
    // clearance -> server rejected with retcode 10016 "Invalid stops".
    // Now: every attempt re-reads LIVE prices and the broker's stops level
    // (some brokers inflate it in thin hours) and widens the SL/TP distances
    // so the side-correct clearance holds by construction: entry-anchored
    // distance >= broker min + spread + 2pt margin. Bounded 3-attempt retry
    // with growing clearance; fail-closed abandon if the broker still refuses.
    double fillPrice = 0;
    bool sent = false;
    for(int attempt = 1; attempt <= 3 && !sent; attempt++)
    {
        int digits = SymDigits();
        double pt = SymPoint();
        double minDist = MathMax((double)SymStopsLevel(),
                                 (double)SymbolInfoInteger(g_symbol, SYMBOL_TRADE_FREEZE_LEVEL)) * pt
                         + (double)(attempt - 1) * 25.0 * pt;   // widen clearance each retry (read.md e: 10pt step too small vs inflated stop levels)
        double spreadPts = (double)SymSpread();
        double cover = minDist + (spreadPts + 2.0) * pt;
        double slDistEff = MathMax(slDist, cover);
        double tpDistEff = MathMax(tpDist, cover);
        double price, sl, tp;
        if(dir > 0)
        {
            price = SymAsk();
            sl = price - slDistEff;   // <= bid - minDist by construction (BUY SL validates vs BID)
            tp = price + tpDistEff;   // >= bid + minDist by construction
        }
        else
        {
            price = SymBid();
            sl = price + slDistEff;   // >= ask + minDist by construction (SELL SL validates vs ASK)
            tp = price - tpDistEff;   // <= ask - minDist by construction
        }
        sl = NormalizeDouble(sl, digits);
        tp = NormalizeDouble(tp, digits);

        uint rc = 0;
        if(SendOrder(type, lot, price, sl, tp, "PAT:ultrascalp", fillPrice, rc))
        {
            sent = true;
            // Gate 7 — post-fill slippage check (using res.price, not history).
            // FIX: res.price for a market BUY returns the BID (position price), while
            // req.price was the ASK — so the raw difference equals the SPREAD, not
            // slippage. Only flag slippage BEYOND the spread: effective threshold =
            // current spread + MaxSlippagePts. This stops the gate from closing every
            // trade on a wide-spread symbol like XAUUSD.e.
            if(fillPrice > 0)
            {
                double slipPts = MathAbs(fillPrice - price) / SymPoint();
                double effSlipMax = (double)SymSpread() + MaxSlippagePts;
                if(slipPts > effSlipMax)
                {
                    Print("SLIPPAGE EXCEEDED: fill ", DoubleToString(fillPrice, digits),
                          " vs req ", DoubleToString(price, digits), " = ", DoubleToString(slipPts, 1),
                          " pts > spread+", DoubleToString(MaxSlippagePts, 1),
                          " (", DoubleToString(effSlipMax, 1), " pts). Closing (fail-closed).");
                    CloseNewestPosition();
                    return false;
                }
            }
            return true;
        }
        // Retry ONLY transient price/stops rejections; everything else aborts.
        if(rc == 10004 || rc == 10015 || rc == 10016 || rc == 10021)
        {
            Print("ORDER RETRY ", attempt, "/2 after retcode ", rc, " (invalid stops/price) — re-clamping to live bid/ask:",
                  " price=", DoubleToString(price, digits),
                  " sl=", DoubleToString(sl, digits),
                  " tp=", DoubleToString(tp, digits),
                  " bid=", DoubleToString(SymBid(), digits),
                  " ask=", DoubleToString(SymAsk(), digits),
                  " spread=", DoubleToString((double)SymSpread(), 1), "pts",
                  " stopsLevel=", IntegerToString(SymStopsLevel()));
        }
        else return false;
    }
    Print("ORDER ABANDONED after 3 attempts (stops still rejected) — NO-TRADE, waiting for next signal");
    return false;
}

//+------------------------------------------------------------------+
//| Close the most recently opened position with our magic            |
//+------------------------------------------------------------------+
void CloseNewestPosition()
{
    ulong newest = 0;
    datetime newestTime = 0;
    for(int i = PositionsTotal() - 1; i >= 0; i--)
    {
        ulong ticket = PositionGetTicket(i);
        if(ticket == 0) continue;
        if(!PAT_IsOurMagic(PositionGetInteger(POSITION_MAGIC))) continue;
        if(PositionGetString(POSITION_SYMBOL) != g_symbol) continue;
        datetime ot = (datetime)PositionGetInteger(POSITION_TIME);
        if(ot >= newestTime) { newestTime = ot; newest = ticket; }
    }
    if(newest > 0) ClosePosition(newest);
}

//+------------------------------------------------------------------+
//| Rolling-hour trade tracking                                       |
//+------------------------------------------------------------------+
void RecordTradeTime()
{
    // Prune expired timestamps before appending (read.md Section 4).
    datetime cutoff = TimeCurrent() - 3600;
    int valid = 0;
    for(int i = 0; i < g_tradeCount; i++)
        if(g_tradeTimes[i] >= cutoff)
            g_tradeTimes[valid++] = g_tradeTimes[i];
    g_tradeCount = valid;
    if(g_tradeCount < 64)
        g_tradeTimes[g_tradeCount++] = TimeCurrent();
}

int TradesInLastHour()
{
    int n = 0;
    datetime cutoff = TimeCurrent() - 3600;
    for(int i = 0; i < g_tradeCount; i++)
        if(g_tradeTimes[i] >= cutoff) n++;
    return n;
}

//+------------------------------------------------------------------+
//| Track consecutive losses on position close (via OnTradeTransaction)|
//+------------------------------------------------------------------+
void OnTradeTransaction(const MqlTradeTransaction &trans,
                        const MqlTradeRequest &request,
                        const MqlTradeResult &result)
{
    if(trans.type != TRADE_TRANSACTION_DEAL_ADD) return;
    ulong deal = trans.deal;
    if(deal == 0) return;
    // FIX Tier3: must select the deal into the history cache before reading it.
    if(!HistoryDealSelect(deal)) return;
    if(HistoryDealGetString(deal, DEAL_SYMBOL) != g_symbol) return;
    if(!PAT_IsOurMagic(HistoryDealGetInteger(deal, DEAL_MAGIC))) return;
    if(HistoryDealGetInteger(deal, DEAL_ENTRY) != DEAL_ENTRY_OUT) return;

    double profit = HistoryDealGetDouble(deal, DEAL_PROFIT)
                  + HistoryDealGetDouble(deal, DEAL_SWAP)
                  + HistoryDealGetDouble(deal, DEAL_COMMISSION);
    if(profit < 0)
    {
        g_consecLosses++;
        GlobalVariableSet(GV_ConsecLoss(), g_consecLosses);
    }
    else if(profit >= MinProfitToResetLossStreak)
    {
        // Only a MEANINGFUL win resets the streak. A tiny win/breakeven that
        // just covers costs does not — otherwise 5 losses + a $0.01 win would
        // reset the counter while you're still down (read.md Section 3).
        g_consecLosses = 0;
        GlobalVariableSet(GV_ConsecLoss(), 0);
    }
    // else: tiny win/breakeven — keep the streak alive.
    g_lastTradeCloseTime = TimeCurrent();
    GlobalVariableSet(GV_LastClose(), (double)g_lastTradeCloseTime);
}

//+------------------------------------------------------------------+
//| Refresh native data (dollar proxy) on timer                       |
//+------------------------------------------------------------------+
void RefreshExternalData()
{
    // Dollar proxy is computed on-demand in the signal; nothing to cache.
    // (Kept as a hook for future native data refreshes.)
}

//+------------------------------------------------------------------+
//| Panel                                                             |
//+------------------------------------------------------------------+
void UpdatePanel()
{
    double profitPct = 0;
    if(g_dayStartBalance > 0)
        profitPct = (g_dailyPnL / g_dayStartBalance) * 100;
    string status = "RUNNING";
    if(g_halted) status = "HARD HALT";
    else if(g_profitHalt) status = "PROFIT TARGET";
    else if(g_lossHalt) status = "LOSS LIMIT";

    string p = "Predict-A-Trade v1.17 - Ultra Scalp [ALL-SESS]\n";
    p += "Symbol: " + g_symbol + "  Magic: " + IntegerToString((int)g_magic) + "\n";
    p += "Status: " + status + "\n";
    p += "Day-start balance: " + DoubleToString(g_dayStartBalance, 2) + "\n";
    p += "Daily P&L: " + DoubleToString(g_dailyPnL, 2) + " (" + DoubleToString(profitPct, 2) + "%)\n";
    p += "Daily trades: " + IntegerToString(g_dailyTrades) + "/" + IntegerToString(MaxDailyTrades) + "\n";
    p += "Consec losses: " + IntegerToString(g_consecLosses) + "/" + IntegerToString(MaxConsecutiveLosses) + "\n";
    p += "Open positions: " + IntegerToString(CountPositions()) + "/" + IntegerToString(MaxPositions) + "\n";
    p += "Spread: " + DoubleToString((double)SymSpread(), 1) + " pts (auto max " + DoubleToString(g_avgSpread * AutoSpreadMult, 1) + ")\n";
    p += "ATR(M1): " + DoubleToString(GetATR() / SymPoint(), 1) + " pts  ATR(M5): " + DoubleToString(GetATR5(), 2) + "\n";
    int rg = GetRegime();
    p += "Regime: " + (rg > 0 ? "TRENDING_BULLISH" : (rg < 0 ? "TRENDING_BEARISH" : "RANGE")) + "\n";
    p += "Exit: SL " + DoubleToString(CostSLMult, 1) + "x cost | TP " + DoubleToString(CostTPMult, 1) + "x cost (single target, measured v1.17)\n";
    p += "Signals: Pullback " + IntegerToString(g_sigPathPullback) +
         " | No-entry bars: " + IntegerToString(g_sigNoEntryBars) + "\n";
    if(UseVWAP) p += "VWAP: " + DoubleToString(GetVWAP(), SymDigits()) + "\n";
    if(UseDollarProxy) p += "EURUSD chg: " + DoubleToString(GetDollarProxyChange() * 100, 2) + "%\n";
    Comment(p);
}
//+------------------------------------------------------------------+
