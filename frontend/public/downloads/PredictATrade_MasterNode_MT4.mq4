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
#property version   "1.29"
#property strict

// v1.27 account-type detection (additive; data-only node — tags snapshots)
//+------------------------------------------------------------------+
//| CAccountTypeDetector — INLINED (operator mandate: the MQL must    |
//| not reference any external file). Formerly                        |
//| mql/include/CAccountTypeDetector.mqh; identical content, same     |
//| PAT_ATD_* / CAccountTypeDetector API.                             |
//+------------------------------------------------------------------+

//--- Account type enum (stable string codes for wire payloads & DB)
enum ENUM_PAT_ACCOUNT_TYPE
{
   PAT_ATD_STANDARD = 0,   // Standard (default/baseline)
   PAT_ATD_DEMO,           // Demo account
   PAT_ATD_CONTEST,        // Contest/competition account
   PAT_ATD_ISLAMIC,        // Swap-free (Islamic) account
   PAT_ATD_MICRO_CENT,     // Micro / cent-denominated account
   PAT_ATD_ECN,            // ECN (market execution + commission)
   PAT_ATD_STP             // STP (straight-through processing)
};

//--- Cached detection state (globals: one instance per terminal)
long     g_patATD_login       = 0;       // account login the cache belongs to
int      g_patATD_type        = -1;      // cached type (-1 = not yet detected)
string   g_patATD_reason      = "";      // human-readable detection reason
int      g_patATD_confirms    = 0;       // confirmation count (rollover checks)
bool     g_patATD_verified    = false;   // verified = confirmed by observation
string   g_patATD_override    = "";      // manual override ("", or type code)
bool     g_patATD_swapRoll    = false;   // swap-free indicator observed
int      g_patATD_swapChecks  = 0;       // rollover checks performed
datetime g_patATD_lastRoll    = 0;       // last rollover check time
double   g_patATD_histComm    = 0;       // max commission seen in deal history

//--- Public inputs may be mapped to these by the host EA (optional).
//    Defaults keep detection fully automatic.
input string PAT_ATD_Override     = "";    // Manual account-type override ("" = auto)
input bool   PAT_ATD_EnableDetect = true;  // Enable account type detection

//+------------------------------------------------------------------+
//| Type → wire string (used in signals, ACKs, heartbeats, DB)         |
//+------------------------------------------------------------------+
string PAT_ATD_TypeName(int t)
{
   switch(t)
   {
      case PAT_ATD_DEMO:       return "Demo";
      case PAT_ATD_CONTEST:    return "Contest";
      case PAT_ATD_ISLAMIC:    return "Islamic";
      case PAT_ATD_MICRO_CENT: return "MicroCent";
      case PAT_ATD_ECN:        return "ECN";
      case PAT_ATD_STP:        return "STP";
      default:                 return "Standard";
   }
}

//+------------------------------------------------------------------+
//| String → type (for the manual override input)                     |
//+------------------------------------------------------------------+
int PAT_ATD_TypeFromName(const string name)
{
   string n = name;
   StringToLower(n);
   if(n == "demo")       return PAT_ATD_DEMO;
   if(n == "contest")    return PAT_ATD_CONTEST;
   if(n == "islamic")    return PAT_ATD_ISLAMIC;
   if(n == "micro" || n == "microcent" || n == "micro_cent" || n == "cent")
                         return PAT_ATD_MICRO_CENT;
   if(n == "ecn")        return PAT_ATD_ECN;
   if(n == "stp")        return PAT_ATD_STP;
   if(n == "standard")   return PAT_ATD_STANDARD;
   return -1;
}

//+------------------------------------------------------------------+
//| Empirical commission scan: max DEAL_COMMISSION in recent history  |
//| MT5: HistorySelect + HistoryDealGetDouble(DEAL_COMMISSION).       |
//| MT4: OrderSelect loop reading OrderCommission().                  |
//| Returns the largest absolute commission seen (account currency).  |
//+------------------------------------------------------------------+
double PAT_ATD_ScanCommission()
{
   double maxComm = 0;
#ifdef __MQL5__
   datetime from = TimeCurrent() - 30*24*60*60; // last 30 days
   if(!HistorySelect(from, TimeCurrent() + 60))
      return 0; // fail-open: treat as no commission data
   int total = HistoryDealsTotal();
   int scan  = MathMin(total, 200); // bounded scan
   for(int i = total - 1; i >= total - scan && i >= 0; i--)
   {
      ulong ticket = HistoryDealGetTicket(i);
      if(ticket == 0) continue;
      double c = MathAbs(HistoryDealGetDouble(ticket, DEAL_COMMISSION));
      if(c > maxComm) maxComm = c;
   }
#else
   int total = OrdersHistoryTotal();
   int scan  = MathMin(total, 200);
   for(int i = total - 1; i >= total - scan && i >= 0; i--)
   {
      if(!OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)) continue;
      double c = MathAbs(OrderCommission());
      if(c > maxComm) maxComm = c;
   }
#endif
   return maxComm;
}

//+------------------------------------------------------------------+
//| Swap-free confirmation: does any OPEN position carry swap?        |
//| Returns true when at least one open position exists and ALL of    |
//| them report swap == 0.00. (Positions on the account, any symbol.) |
//+------------------------------------------------------------------+
bool PAT_ATD_OpenPositionsAllSwapFree(bool &anyOpen)
{
   anyOpen = false;
#ifdef __MQL5__
   int total = PositionsTotal();
   for(int i = 0; i < total; i++)
   {
      ulong ticket = PositionGetTicket(i);
      if(ticket == 0) continue;
      if(PositionGetDouble(POSITION_SWAP) != 0.0) return false;
      anyOpen = true;
   }
#else
   int total = OrdersTotal();
   for(int i = 0; i < total; i++)
   {
      if(!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
      if(OrderType() > OP_SELL) continue; // ignore pending orders
      if(OrderSwap() != 0.0) return false;
      anyOpen = true;
   }
#endif
   return true;
}

//+------------------------------------------------------------------+
//| Core detection (no caching — call Detect() instead)               |
//+------------------------------------------------------------------+
int PAT_ATD_DetectCore(string &reasonOut, int &confirms)
{
   confirms = 0;

   //--- Manual override has the highest priority of all (operator authority)
   string ov = PAT_ATD_Override;
   StringToLower(ov);
   if(StringLen(ov) > 0)
   {
      int t = PAT_ATD_TypeFromName(ov);
      if(t >= 0)
      {
         confirms = 1;
         g_patATD_reason = "manual override: " + ov;
         return t;
      }
   }

   //--- 1) Demo / 2) Contest — authoritative platform flags
   long tradeMode = -1;
#ifdef __MQL5__
   tradeMode = AccountInfoInteger(ACCOUNT_TRADE_MODE);
#else
   // MT4 has ACCOUNT_TRADE_MODE via AccountInfoInteger (build 600+).
   tradeMode = AccountInfoInteger(ACCOUNT_TRADE_MODE);
#endif
   // ACCOUNT_TRADE_MODE_DEMO=0, CONTEST=1, REAL=2 (both platforms)
   if(tradeMode == 0)
   {
      confirms = 1;
      g_patATD_reason = "ACCOUNT_TRADE_MODE=DEMO";
      return PAT_ATD_DEMO;
   }
   if(tradeMode == 1)
   {
      confirms = 1;
      g_patATD_reason = "ACCOUNT_TRADE_MODE=CONTEST";
      return PAT_ATD_CONTEST;
   }

   //--- 3) Islamic (swap-free) — symbol swap + rollover confirmation.
   //    The type itself is declared immediately (swap-free symbol) and
   //    CONFIRMED by observing open-position swap staying 0.00 across
   //    3 rollover boundaries (call PAT_ATD_OnRollover() from the timer).
   string sym = _Symbol;
#ifdef __MQL5__
   double swapL = SymbolInfoDouble(sym, SYMBOL_SWAP_LONG);
   double swapS = SymbolInfoDouble(sym, SYMBOL_SWAP_SHORT);
#else
   // MT4: MarketInfo swap values for the chart symbol
   double swapL = MarketInfo(Symbol(), MODE_SWAPLONG);
   double swapS = MarketInfo(Symbol(), MODE_SWAPSHORT);
#endif
   bool symbolSwapFree = (swapL == 0.0 && swapS == 0.0);
   if(symbolSwapFree && !g_patATD_swapRoll)
   {
      // Seed the rollover observer: check open-position swap right away.
      g_patATD_swapRoll = true;
      g_patATD_swapChecks = 0;
      g_patATD_lastRoll = TimeCurrent();
   }
   if(symbolSwapFree && g_patATD_swapChecks >= 3)
   {
      confirms = g_patATD_swapChecks;
      g_patATD_reason = "swap-free symbol + " + IntegerToString(g_patATD_swapChecks) + " rollover confirms";
      return PAT_ATD_ISLAMIC;
   }
   else if(symbolSwapFree)
   {
      // Provisional Islamic while rollover confirmation accumulates.
      confirms = g_patATD_swapChecks;
      g_patATD_reason = "swap-free symbol (pending " + IntegerToString(3 - g_patATD_swapChecks) + " rollover confirms)";
      return PAT_ATD_ISLAMIC;
   }

   //--- 4) Micro / Cent — lot/balance scaling pattern
   double volMin = 0;
   long   curDigits = 0;
#ifdef __MQL5__
   volMin = SymbolInfoDouble(_Symbol, SYMBOL_VOLUME_MIN);
   // ACCOUNT_CURRENCY_DIGITS is not a standard identifier; approximate the
   // cent-denomination check with balance/equity granularity: cent accounts
   // report balances like 123456.78 CENTS (= $78.56 standard). Heuristic:
   // balance with .00 precision AND large value AND tiny min-lot. The min-lot
   // check is the reliable half of the pair (spec: SYMBOL_VOLUME_MIN < 0.01).
#else
   volMin = MarketInfo(Symbol(), MODE_MINLOT);
#endif
   double balance = AccountInfoDouble(ACCOUNT_BALANCE);
   bool tinyLot   = (volMin > 0 && volMin < 0.01);
   bool hugeBalance = (balance >= 1000000.0); // 1e6 "units" = $10k on cent scale
   if(tinyLot || (curDigits > 2))
   {
      confirms = 1;
      g_patATD_reason = "min-lot " + DoubleToString(volMin, 3) + " < 0.01 (cent denomination)";
      return PAT_ATD_MICRO_CENT;
   }
   if(hugeBalance && volMin <= 0.01)
   {
      confirms = 1;
      g_patATD_reason = "balance " + DoubleToString(balance, 0) + " + min-lot " + DoubleToString(volMin, 3) + " (cent scaling pattern)";
      return PAT_ATD_MICRO_CENT;
   }

   //--- 5) ECN / 6) STP — execution-mode + commission fingerprint
   double histComm = PAT_ATD_ScanCommission();
   g_patATD_histComm = histComm;
   bool hasCommission = (histComm > 0.0);

   // Execution mode fingerprint.
   // MT5: SYMBOL_TRADE_MODE + fill-or-kill availability + zero stops level
   // (market execution allows SL/TP modification post-open; instant does not).
   long stopsLevel = 0;
   long fillMode = 0;
   bool marketExec = false;
   bool instantOrRequest = false;
#ifdef __MQL5__
   stopsLevel = SymbolInfoInteger(_Symbol, SYMBOL_TRADE_STOPS_LEVEL);
   fillMode   = SymbolInfoInteger(_Symbol, SYMBOL_FILLING_MODE);
   long symExec = SymbolInfoInteger(_Symbol, SYMBOL_TRADE_MODE);
   marketExec        = (symExec != 0); // tradable; execution mode refined below
   instantOrRequest  = (stopsLevel > 0); // instant/request brokers enforce stop distance at send time
   // Instant/Request execution is signaled by a non-zero stops level OR the
   // account forbidding market orders... MT5 exposes execution mode per symbol
   // via SYMBOL_TRADE_MODE bits; a directly-queryable EXECUTION_INSTANT flag
   // does not exist. Empirical: market execution brokers typically report
   // stops level 0 on majors + allow post-open modify. We use that.
   if(stopsLevel == 0) { marketExec = true; instantOrRequest = false; }
#else
   // MT4: MODE_PROHIBITED / instant by default; stop level > 0 ⇒ instant.
   stopsLevel = MarketInfo(Symbol(), MODE_STOPLEVEL);
   if(stopsLevel == 0) { marketExec = true; instantOrRequest = false; }
   else                { instantOrRequest = true; }
#endif

   if(marketExec && hasCommission)
   {
      confirms = 1;
      g_patATD_reason = "market execution + commission " + DoubleToString(histComm, 2) + "/lot (stops level " + IntegerToString((int)stopsLevel) + ")";
      return PAT_ATD_ECN;
   }
   if(instantOrRequest && !hasCommission)
   {
      confirms = 1;
      g_patATD_reason = "instant/request execution, no commission, stop level " + IntegerToString((int)stopsLevel);
      return PAT_ATD_STP;
   }

   //--- 7) Standard — default classification
   confirms = 1;
   g_patATD_reason = "default (marketExec=" + (marketExec ? "1" : "0") + " comm=" + DoubleToString(histComm, 2) + " stops=" + IntegerToString((int)stopsLevel) + ")";
   return PAT_ATD_STANDARD;
}

//+------------------------------------------------------------------+
//| Rollover confirmation — call from OnTimer (once per hour max).    |
//| Islamic confirmation needs 3 checks with open positions' swap 0. |
//| Cheap: one pass over open positions.                              |
//+------------------------------------------------------------------+
void PAT_ATD_OnRollover()
{
   if(!PAT_ATD_EnableDetect) return;
   if(g_patATD_type < 0) return;      // not detected yet
   if(!g_patATD_swapRoll) return;     // not tracking a swap-free candidate

   // At most one check per hour (rollover cadence proxy)
   if(TimeCurrent() - g_patATD_lastRoll < 3600) return;
   g_patATD_lastRoll = TimeCurrent();

   bool anyOpen = false;
   if(PAT_ATD_OpenPositionsAllSwapFree(anyOpen))
   {
      g_patATD_swapChecks++;
      g_patATD_confirms = g_patATD_swapChecks;
      if(g_patATD_swapChecks >= 3 && g_patATD_type == PAT_ATD_ISLAMIC)
      {
         g_patATD_verified = true;
         Print("[ACCT-DETECT] Islamic swap-free CONFIRMED after ",
               IntegerToString(g_patATD_swapChecks), " rollover checks (all open positions swap=0.00)");
      }
   }
   else
   {
      // Swap observed — the account is NOT swap-free after all. Downgrade.
      if(g_patATD_type == PAT_ATD_ISLAMIC)
      {
         Print("[ACCT-DETECT] Islamic hypothesis REJECTED: open position carries swap. Reclassifying.");
         g_patATD_swapRoll = false;
         g_patATD_swapChecks = 0;
         g_patATD_type = PAT_ATD_Detect(); // re-run detection (cached fields reset inside)
      }
   }
}

//+------------------------------------------------------------------+
//| Public API — lazy, cached detection.                              |
//| Returns the detected type; never throws; falls back to STANDARD.  |
//+------------------------------------------------------------------+
int PAT_ATD_Detect()
{
   long login = 0;
#ifdef __MQL5__
   login = AccountInfoInteger(ACCOUNT_LOGIN);
#else
   login = (long)AccountNumber();
#endif
   // Cache valid for the same login
   if(g_patATD_type >= 0 && g_patATD_login == login)
      return g_patATD_type;

   g_patATD_login = login;
   if(!PAT_ATD_EnableDetect)
   {
      g_patATD_type   = PAT_ATD_STANDARD;
      g_patATD_reason = "detection disabled";
      g_patATD_verified = true;
      return g_patATD_type;
   }

   int confirms = 0;
   int t = PAT_ATD_STANDARD;
   string reason = "";
   // MQL has no try/catch; the fail-safe is a defined fallback on any
   // unexpected path. Detection itself cannot throw — all calls return
   // values or 0 on failure.
   t = PAT_ATD_DetectCore(reason, confirms);

   if(t < 0) // belt & braces: never return an invalid type
   {
      t = PAT_ATD_STANDARD;
      reason = "fallback after detection anomaly";
   }
   g_patATD_type     = t;
   g_patATD_reason   = reason;
   g_patATD_confirms = confirms;
   g_patATD_verified = (t == PAT_ATD_STANDARD || t == PAT_ATD_DEMO || t == PAT_ATD_CONTEST || confirms >= 1);

   Print("[ACCT-DETECT] account ", IntegerToString(login),
         " → ", PAT_ATD_TypeName(t),
         " (", reason, "; confirms=", IntegerToString(confirms), ")");
   return t;
}

//--- Convenience getters (all O(1) cache reads after first call)
int    PAT_ATD_GetType()          { if(g_patATD_type < 0) PAT_ATD_Detect(); return g_patATD_type; }
string PAT_ATD_GetTypeName()      { return PAT_ATD_TypeName(PAT_ATD_GetType()); }
string PAT_ATD_GetReason()        { if(g_patATD_type < 0) PAT_ATD_Detect(); return g_patATD_reason; }
int    PAT_ATD_GetConfirms()      { if(g_patATD_type < 0) PAT_ATD_Detect(); return g_patATD_confirms; }
bool   PAT_ATD_IsVerified()       { if(g_patATD_type < 0) PAT_ATD_Detect(); return g_patATD_verified; }
long   PAT_ATD_GetLogin()         { if(g_patATD_type < 0) PAT_ATD_Detect(); return g_patATD_login; }

//+------------------------------------------------------------------+
//| ADAPTATION HELPERS — per-type math                                |
//+------------------------------------------------------------------+

//--- Position sizing scale factor. Micro/cent accounts: lots are divided
//--- by 100 (1.00 cent-lot = 0.01 standard lot).
double PAT_ATD_LotScale()
{
   int t = PAT_ATD_GetType();
   if(t == PAT_ATD_MICRO_CENT) return 0.01;
   return 1.0;
}

//--- Lot scale + broker min-lot floor respected.
double PAT_ATD_ScaleLot(double lot)
{
   double s = PAT_ATD_LotScale();
   if(s == 1.0) return lot;
   double scaled = lot * s;
   // A cent account's lot step is already 100× finer; the min lot on the
   // cent symbol is typically 0.002 standard-ish. Respect the SYMBOL's own
   // min (the EA's normalizer re-clamps to SymVolMin anyway).
   return scaled;
}

//--- Round-trip commission cost in account currency for a given lot.
//--- ECN: commission_per_lot × lot × 2 (entry + exit). Other types: use
//--- the empirically scanned commission if the broker charges per-deal.
double PAT_ATD_CommissionRoundTrip(double lot, double commissionPerLot)
{
   if(lot <= 0) return 0.0;
   double perLot = commissionPerLot;
   if(perLot <= 0) perLot = 0; // no commission model available
   int t = PAT_ATD_GetType();
   if(t == PAT_ATD_ECN)
      return perLot * lot * 2.0; // round trip
   if(t == PAT_ATD_MICRO_CENT)
      return perLot * lot * 2.0 * 0.01; // cent-scale commission
   return perLot * lot * 2.0; // standard/STP: charge both legs when present
}

//--- Extra slippage buffer in POINTS for risk/exit math.
double PAT_ATD_SlippageBufferPts(double baseBufferPts)
{
   int t = PAT_ATD_GetType();
   if(t == PAT_ATD_STP)
      return baseBufferPts + 2.0;  // STP: 1-2 pip slippage buffer (use 2)
   if(t == PAT_ATD_MICRO_CENT)
      return baseBufferPts + 1.0;  // cent feeds can print wider momentary spreads
   return baseBufferPts;
}

//--- Swap contribution to holding cost. Islamic: ALWAYS 0 in P&L math.
double PAT_ATD_SwapAdjust(double rawSwap)
{
   int t = PAT_ATD_GetType();
   if(t == PAT_ATD_ISLAMIC) return 0.0;
   return rawSwap;
}

//--- P&L adjustment: commission + swap handling per account type.
//--- netPnL = raw + swap_adjusted − commission_round_trip
double PAT_ATD_NetPnL(double rawProfit, double swap, double lot, double commissionPerLot)
{
   double pnl = rawProfit + PAT_ATD_SwapAdjust(swap);
   int t = PAT_ATD_GetType();
   if(t == PAT_ATD_ECN || t == PAT_ATD_MICRO_CENT)
      pnl -= PAT_ATD_CommissionRoundTrip(lot, commissionPerLot);
   else if(g_patATD_histComm > 0)
      pnl -= g_patATD_histComm; // broker-charged, already in deal P&L — no double count
   return pnl;
}

//--- Risk-reward adjustment for commission erosion (ECN).
//--- Returns the RR multiplier to apply so net RR meets the target.
double PAT_ATD_RRErosionMult(double tpDistPrice, double slDistPrice, double lot, double commissionPerLot, double valuePerUnit)
{
   if(tpDistPrice <= 0 || slDistPrice <= 0) return 1.0;
   int t = PAT_ATD_GetType();
   if(t != PAT_ATD_ECN) return 1.0;
   double comm = PAT_ATD_CommissionRoundTrip(lot, commissionPerLot);
   if(comm <= 0) return 1.0;
   double commPrice = comm / MathMax(valuePerUnit, 0.0000001); // $ → price units
   double erodedTP = tpDistPrice - commPrice;                  // net win distance
   double netRR = erodedTP / slDistPrice;
   if(netRR <= 0) return 1.0; // degenerate — do not amplify
   // Widen the TP so net RR equals the gross RR the strategy expects.
   double mult = tpDistPrice / MathMax(erodedTP, tpDistPrice * 0.1);
   return MathMax(1.0, mult);
}

//--- Demo flag for delivery payloads ("demo" tag on every signal).
bool PAT_ATD_IsDemo()
{
   return (PAT_ATD_GetType() == PAT_ATD_DEMO);
}

//--- One-line traceability tag for signals/ACKs.
string PAT_ATD_Tag()
{
   int t = PAT_ATD_GetType();
   string tag = "\"account_type\":\"" + PAT_ATD_TypeName(t) + "\"";
   if(t == PAT_ATD_DEMO) tag = "\"account_type\":\"Demo\",\"demo\":true";
   return tag;
}
//+------------------------------------------------------------------+
//| CAccountTypeDetector — facade class (name-compatible with spec).  |
//| MQL classes cannot use try/catch; the fail-safe lives in the       |
//| functions above. All methods are static-equivalent wrappers.       |
//+------------------------------------------------------------------+
class CAccountTypeDetector
{
public:
   static int    Detect()               { return PAT_ATD_Detect(); }
   static string TypeName()             { return PAT_ATD_GetTypeName(); }
   static string Reason()               { return PAT_ATD_GetReason(); }
   static int    ConfirmationCount()    { return PAT_ATD_GetConfirms(); }
   static bool   IsVerified()           { return PAT_ATD_IsVerified(); }
   static long   Login()                { return PAT_ATD_GetLogin(); }
   static double LotScale()             { return PAT_ATD_LotScale(); }
   static double ScaleLot(double lot)   { return PAT_ATD_ScaleLot(lot); }
   static double SlippageBufferPts(double base) { return PAT_ATD_SlippageBufferPts(base); }
   static double SwapAdjust(double s)   { return PAT_ATD_SwapAdjust(s); }
   static bool   IsDemo()               { return PAT_ATD_IsDemo(); }
   static string Tag()                  { return PAT_ATD_Tag(); }
   static void   RolloverCheck()        { PAT_ATD_OnRollover(); }
};
//+------------------------------------------------------------------+
#property description "Master Node: Live data collection for system & dashboard"
#property description "NO License Key · NO Trading · Data Collection Only"

//=== Input Parameters ===
input int     SnapshotIntervalMs = 10;     // HFT: 10ms snapshot (1-5ms co-located)
input int     TickIntervalMs     = 0;       // 0 = every tick (HFT: 1-5ms when co-located)
input string  BrokerSymbol      = "";     // Empty = auto-detect chart symbol
input string  PATCloudURL       = "https://api.predictatrade.com"; // Cloud API base URL (add to WebRequest allowlist)
input string  MasterDeviceId    = "";     // Device UUID (optional — auto-activation from MasterLicenseKey)
input string  MasterDeviceSecret= "";     // Device secret (optional)
input string  MasterLicenseKey  = "";     // Master (data-node) license key for auto-activation
input bool    SendTickData      = true;   // Send tick data to Cloud
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
// (resync flag file removed — REQUEST_SNAPSHOT arrives on the edge queue)

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
//+------------------------------------------------------------------+
//| v1.27 ADDITIVE HELPERS — account-type integration (MasterNode)    |
//+------------------------------------------------------------------+
void PAT_ATD_InitDetect()
{
   int t = CAccountTypeDetector::Detect();          // never throws; falls back Standard
   if(CAccountTypeDetector::IsVerified())
      Print("[MASTER] account_type=", CAccountTypeDetector::TypeName(),
            " login=", IntegerToString(AccountNumber()),
            " confirmed (", CAccountTypeDetector::Reason(), ")");
   else
      Print("[MASTER] account_type=", CAccountTypeDetector::TypeName(),
            " PROVISIONAL (", CAccountTypeDetector::Reason(), ")");
}

//+------------------------------------------------------------------+
int OnInit()
{
    Print("Predict-A-Trade Master Node v1.19 initializing (MT4)...");
    Print("Mode: DATA COLLECTION ONLY — NO License Key, NO Trading");

    g_symbol = BrokerSymbol;
    if(g_symbol == "") g_symbol = Symbol();
    g_broker = AccountCompany();
    g_accountID = DoubleToStr(AccountNumber(), 0);

    Print("Symbol: ", g_symbol);
    Print("Broker: ", g_broker);
    Print("Account: ", g_accountID);

    if(MasterEnsureDevice())
    {
        g_connection = "CONNECTED";
        Print("[MASTER_NODE] Cloud device ready (", g_deviceId, ") — edge ingest mode (v1.19).");
        SendMasterInit();
    }
    else
    {
        g_connection = "OFFLINE";
        Print("WARNING: Cloud device not ready — set MasterLicenseKey (or Device Id/Secret) in EA inputs.");
        Print("Also add ", PATCloudURL, " to Tools→Options→Expert Advisors→WebRequest allowlist.");
    }


    // ─── Resilience: periodic timer ───
    // OnTick only fires when the broker streams quotes for the chart symbol. If
    // the terminal/connection hiccups and ticks stall, OnTick stops and the
    // engine goes silently blind. A 1-second OnTimer keeps emitting
    // MARKET_SNAPSHOT regardless of tick flow (terminal must be connected).
    EventSetTimer(1000);

    // v1.27: account-type detection (read-only; tags every snapshot)
    PAT_ATD_InitDetect();

    PATUI_OnInit(); // v1.29.4: terminal dashboard (object panel)
    return(INIT_SUCCEEDED);
}

//+------------------------------------------------------------------+
 void OnDeinit(const int reason)
{
    EventKillTimer();
    MasterWrite("MASTER_DEINIT|{\"reason\":" + IntegerToString((long)reason) +
                ",\"symbol\":\"" + g_symbol + "\",\"account\":\"" + g_accountID + "\"}\n");
    PATUI_OnDeinit(); // v1.29.4: remove dashboard objects
    Comment("");
}

//+------------------------------------------------------------------+
void OnTick()
{
    CheckAgentConnection();

    if(g_connection != "CONNECTED") { PATUI_DashUpdate(true); return; }

    if(SendTickData)
        SendTickToAgent();

    if(SendSnapshots)
        SendMarketSnapshot();


    PATUI_DashUpdate(false); // v1.29.4: object dashboard (self-throttled)
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

    // Engine recovery nudge: REQUEST_SNAPSHOT commands arrive on the edge
    // queue (control plane) — poll, ack, and force an immediate snapshot.
    MasterEdgePoll();

    if(g_connection == "CONNECTED" && SendSnapshots)
    {
        SendMarketSnapshot();
    }


    PATUI_DashUpdate(true); // v1.29.4: dashboard repaint (works with no ticks)
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

    // Option B liveness: "connected" = cloud device credentials ready.
    // Reachability errors surface from ingest HTTP failures (g_ingestErrCount).
    bool deviceReady = MasterEnsureDevice();

    if(deviceReady)
    {
        g_connection = "CONNECTED";
        if(!g_lastAgentState)
        {
            g_lastAgentState = true;
            Print("[CLOUD] Master data device is now ACTIVE (device ", g_deviceId, ")");
            SendAgentNotification("ACTIVE", "Master Node cloud link is ACTIVE (edge ingest).");
        }
    }
    else
    {
        if(g_connection == "CONNECTED")
        {
            g_connection = "OFFLINE";
            Print("[CLOUD] Master data device credentials lost");
        }
        if(g_lastAgentState)
        {
            g_lastAgentState = false;
            Print("[CLOUD] Master data device is now OFFLINE (activation failed). Live data feed interrupted.");
            SendAgentNotification("OFFLINE", "WARNING: Master Node cloud link OFFLINE — device activation failed. Check MasterLicenseKey / WebRequest allowlist.");
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
        msg += ",\"account_type\":\"" + PAT_ATD_GetTypeName() + "\"";
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
//| External signal timestamp -> broker (TimeCurrent) timeline.       |
//| v1.28: ALL trading decisions must run on the broker clock that   |
//| TimeCurrent() returns (Master Node mandate). Converts an EXTERNAL|
//| timestamp — an upstream provider signal issued in a fixed        |
//| wall-clock zone (UTC, Dubai GMT+4, exchange local …) — onto the  |
//| broker timeline BEFORE comparing it against TimeCurrent(),       |
//| iTime()/iClose bar times, or an order expiration.                |
//| srcOffsetMinutes: minutes EAST of UTC of the source wall clock   |
//| (0 = UTC, 240 = Dubai GMT+4, -300 = New York GMT-5). A trailing  |
//| Z/+HH:MM/-HH:MM suffix in the string wins over srcOffsetMinutes. |
//| Returns 0 when the input is empty/unparseable (fail-closed).     |
//| DST: conversion uses ONLY live clock diffs (TimeLocal-TimeGMT,   |
//| TimeLocal-TimeCurrent) — no hardcoded offsets, so the Equiti     |
//| GMT+2 winter / GMT+3 summer change and the Windows PC's own DST  |
//| are both picked up automatically.                                |
//+------------------------------------------------------------------+
datetime PAT_LocalToBroker(string iso, int srcOffsetMinutes)
{
    datetime src = PAT_ParseISO8601Local(iso, srcOffsetMinutes);
    if(src <= 0) return 0;
    // Absolute UTC -> terminal-local -> broker, via live clock diffs only
    // (same bridge the client EAs use; DST-safe, no hardcoded offsets).
    return PAT_UTCToBrokerWall(src);
}

//+------------------------------------------------------------------+
//| DST-ADAPTIVE TIME BRIDGES — no hardcoded offsets anywhere.        |
//| Two live diffs do all the work:                                  |
//|   localOffset = TimeLocal() - TimeGMT()  (Windows PC zone, DST-  |
//|   aware: +4 Dubai summer / +3 Dubai winter, follows the OS);     |
//|   brokerOffset = TimeLocal() - TimeCurrent() (broker zone as the |
//|   SERVER sees it: GMT+2 winter / GMT+3 summer on Equiti, changes |
//|   automatically when the broker rolls DST — nothing to re-set).  |
//| Pure MQL4: no WebRequest, no files, no includes.                 |
//+------------------------------------------------------------------+
long PAT_LocalGMTOffsetSeconds()
{
    return ((long)TimeLocal() - (long)TimeGMT());
}

long PAT_BrokerOffsetSeconds()
{
    return ((long)TimeLocal() - (long)TimeCurrent());
}

datetime PAT_UTCToBrokerWall(long utcSeconds)
{
    // utc - localOffset = the same instant expressed as the PC's local
    // wall-clock; brokerOffset shifts that wall-clock onto the broker's.
    return (datetime)(utcSeconds - PAT_LocalGMTOffsetSeconds()
                                   + PAT_BrokerOffsetSeconds());
}

//+------------------------------------------------------------------+
//| Internal: parse ISO8601 with an explicit fixed source offset.     |
//| Used by PAT_LocalToBroker for external timestamps that carry no   |
//| Z/+HH:MM/-HH:MM suffix (components are in the source wall clock). |
//+------------------------------------------------------------------+
datetime PAT_ParseISO8601Local(string iso, int srcOffsetMinutes)
{
    // Handles "2026-08-24T16:25:11[.frac]" — components are in the SOURCE
    // wall clock (srcOffsetMinutes east of UTC) when no suffix is present;
    // a trailing Z/+HH:MM/-HH:MM suffix always wins over srcOffsetMinutes.
    if(StringLen(iso) < 19) return 0;
    int y  = (int)StringToInteger(StringSubstr(iso, 0, 4));
    int mo = (int)StringToInteger(StringSubstr(iso, 5, 2));
    int d  = (int)StringToInteger(StringSubstr(iso, 8, 2));
    int h  = (int)StringToInteger(StringSubstr(iso, 11, 2));
    int mi = (int)StringToInteger(StringSubstr(iso, 14, 2));
    int se = (int)StringToInteger(StringSubstr(iso, 17, 2));
    if(y < 2000 || mo < 1 || mo > 12 || d < 1 || d > 31) return 0;
    MqlDateTime dt;
    dt.year = y; dt.mon = mo; dt.day = d; dt.hour = h; dt.min = mi; dt.sec = se;
    dt.day_of_week = 0; dt.day_of_year = 0;
    datetime src = StructToTime(dt);
    int suffixOff = 0;
    if(StringLen(iso) >= 20)
    {
        string c = StringSubstr(iso, 19, 1);
        if(c == "Z" || c == "z") suffixOff = 0;                 // explicit UTC
        else if(c == "+" || c == "-")
        {
            int sign = (c == "+") ? 1 : -1;
            int oh = (int)StringToInteger(StringSubstr(iso, 20, 2));
            int om = 0;
            if(StringLen(iso) >= 25)
                om = (int)StringToInteger(StringSubstr(iso, 23, 2));
            suffixOff = sign * (oh * 3600 + om * 60);           // seconds east of UTC
        }
    }
    if(suffixOff != 0 || srcOffsetMinutes != 0)
        src = (datetime)((long)src - suffixOff - srcOffsetMinutes * 60);
    return src;                                                 // absolute UTC
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
    msg += ",\"ea_version\":\"1.19\"";
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
//| ══════════ OPTION B TRANSPORT (v1.19.0) ═══════════════════════ |
//| The Master EA posts market data DIRECTLY to the cloud engine over |
//| HTTPS — the Windows Agent and its IPC files are gone.             |
//| Outbound: POST /ingest/agent (Bearer device JWT, role=data).      |
//| Inbound:  HMAC edge-poll (REQUEST_SNAPSHOT / EMERGENCY_STOP …).   |
//+------------------------------------------------------------------+
string   g_deviceId      = "";
string   g_deviceSecret  = "";
string   g_refreshToken  = "";
string   g_accessToken   = "";
datetime g_tokenExpiry   = 0;
bool     g_masterNetShown = false;
int      g_ingestOkCount  = 0;
int      g_ingestErrCount = 0;
long     g_hmacCounter    = 0;

#define PAT_MASTER_DEVICE_FILE "PAT_master_device_mt4.txt" // device_id|device_secret|refresh_token (MT4-specific: MT4+MT5 share FILE_COMMON)

//--- MasterSHA256: pure-MQL4 SHA-256 (FIPS 180-4) over UTF-8 bytes
int MasterROTR(int x, int n) { return (int)(((uint)x >> n) | ((uint)x << (32 - n))); }

void MasterStoreU32BE(uint v, uchar &outp[], int pos)
{
    outp[pos]   = (uchar)((v >> 24) & 0xFF);
    outp[pos+1] = (uchar)((v >> 16) & 0xFF);
    outp[pos+2] = (uchar)((v >> 8) & 0xFF);
    outp[pos+3] = (uchar)(v & 0xFF);
}

void MasterSHA256K(uint &k[])
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

void MasterSHA256(const uchar &msg[], uchar &digest[])
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
    MasterSHA256K(k);

    uint w[64];
    for(int off = 0; off < paddedLen; off += 64)
    {
        for(int t = 0; t < 16; t++)
            w[t] = ((uint)padded[off + t*4] << 24) | ((uint)padded[off + t*4 + 1] << 16) |
                   ((uint)padded[off + t*4 + 2] << 8) | (uint)padded[off + t*4 + 3];
        for(int t = 16; t < 64; t++)
        {
            uint s0 = MasterROTR(w[t-15],7) ^ MasterROTR(w[t-15],18) ^ (w[t-15] >> 3);
            uint s1 = MasterROTR(w[t-2],17) ^ MasterROTR(w[t-2],19) ^ (w[t-2] >> 10);
            w[t] = w[t-16] + s0 + w[t-7] + s1;
        }
        uint a=h0, b=h1, c=h2, d=h3, e=h4, f=h5, g=h6, hh=h7;
        for(int t = 0; t < 64; t++)
        {
            uint S1 = MasterROTR(e,6) ^ MasterROTR(e,11) ^ MasterROTR(e,25);
            uint ch = (e & f) ^ ((~e) & g);
            uint temp1 = hh + S1 + ch + k[t] + w[t];
            uint S0 = MasterROTR(a,2) ^ MasterROTR(a,13) ^ MasterROTR(a,22);
            uint maj = (a & b) ^ (a & c) ^ (b & c);
            uint temp2 = S0 + maj;
            hh=g; g=f; f=e; e=d+temp1; d=c; c=b; b=a; a=temp1+temp2;
        }
        h0+=a; h1+=b; h2+=c; h3+=d; h4+=e; h5+=f; h6+=g; h7+=hh;
    }

    ArrayResize(digest, 32);
    MasterStoreU32BE(h0, digest, 0);  MasterStoreU32BE(h1, digest, 4);
    MasterStoreU32BE(h2, digest, 8);  MasterStoreU32BE(h3, digest, 12);
    MasterStoreU32BE(h4, digest, 16); MasterStoreU32BE(h5, digest, 20);
    MasterStoreU32BE(h6, digest, 24); MasterStoreU32BE(h7, digest, 28);
}

string MasterSHA256Hex(string text)
{
    uchar msg[];
    StringToCharArray(text, msg, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(msg, ArraySize(msg) - 1);
    uchar digest[];
    MasterSHA256(msg, digest);
    string hexchars = "0123456789abcdef";
    string outp = "";
    for(int i = 0; i < ArraySize(digest); i++)
    {
        outp += StringSubstr(hexchars, (digest[i] >> 4) & 0x0F, 1);
        outp += StringSubstr(hexchars, digest[i] & 0x0F, 1);
    }
    return outp;
}

//--- MasterHmacSha256Hex: HMAC-SHA256 = SHA256(opad || SHA256(ipad || msg))
string MasterHmacSha256Hex(string key, string message)
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
    MasterSHA256(innerMsg, innerDigest);
    uchar outerMsg[96];
    ArrayCopy(outerMsg, opad, 0, 0, 64);
    ArrayCopy(outerMsg, innerDigest, 64, 0, 32);
    uchar digest[];
    MasterSHA256(outerMsg, digest);
    string hexchars = "0123456789abcdef";
    string outp = "";
    for(int i = 0; i < ArraySize(digest); i++)
    {
        outp += StringSubstr(hexchars, (digest[i] >> 4) & 0x0F, 1);
        outp += StringSubstr(hexchars, digest[i] & 0x0F, 1);
    }
    return outp;
}

//--- MasterHMACSign: canonical v1 device signature (byte-identical to
//    DeviceAuthService.verifyRequestSignature in the control plane)
string MasterHMACSign(string path, string body, string deviceId, string deviceSecret, string ts, string nonce)
{
    string bodyHash = MasterSHA256Hex(body);
    string canonical = "v1\n" + ts + "\n" + nonce + "\nPOST\n" + path + "\n" + bodyHash + "\n" + deviceId;
    return MasterHmacSha256Hex(deviceSecret, canonical);
}

//--- MasterHTTPPost: plain JSON POST (no auth) → (status, response)
int MasterHTTPPost(string url, string body, string &response)
{
    string headers = "Content-Type: application/json\r\n";
    uchar data[];
    StringToCharArray(body, data, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(data, ArraySize(data) - 1);
    uchar result[];
    string resHeaders = "";
    int status = WebRequest("POST", url, headers, 8000, data, result, resHeaders);
    response = CharArrayToString(result, 0, WHOLE_ARRAY, CP_UTF8);
    return status;
}

//--- MasterSignedPost: HMAC-authenticated control-plane POST
int MasterSignedPost(string path, string body, string &response)
{
    string ts = IntegerToString((long)TimeGMT() * 1000 + (GetTickCount() % 1000));
    g_hmacCounter++;
    string nonce = MasterSHA256Hex(ts + IntegerToString(g_hmacCounter) + IntegerToString(MathRand()) + IntegerToString(GetTickCount()));
    string sig = MasterHMACSign(path, body, g_deviceId, g_deviceSecret, ts, nonce);

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

//--- MasterReadState / MasterWriteState: bootstrap persistence (FILE_COMMON)
string MasterReadState(string filename)
{
    if(!FileIsExist(filename, FILE_COMMON)) return "";
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

void MasterWriteState(string filename, string content)
{
    int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
    if(h != -1)
    {
        FileWriteString(h, content);
        FileClose(h);
    }
}

//--- MasterJSONString: minimal JSON string-field extractor
string MasterJSONString(string json, string key)
{
    string pat = "\"" + key + "\":\"";
    int p = StringFind(json, pat);
    if(p < 0) return "";
    p += StringLen(pat);
    int e = StringFind(json, "\"", p);
    if(e < 0) return "";
    return StringSubstr(json, p, e - p);
}

//--- MasterJSONEscape: minimal JSON string escaper
string MasterJSONEscape(string s)
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

//--- MasterClearState: truncate the device state file (re-activation reset)
void MasterClearState(string filename)
{
    int h = FileOpen(filename, FILE_WRITE | FILE_TXT | FILE_ANSI | FILE_COMMON);
    if(h != -1) FileClose(h);
}

//--- MasterDeviceFingerprint: stable per-terminal identity
string MasterDeviceFingerprint()
{
    string raw = "MT4-MASTER|" + AccountCompany()
               + "|" + TerminalPath()
               + "|" + IntegerToString((int)TerminalInfoInteger(TERMINAL_BUILD));
    return MasterSHA256Hex(raw);
}

//--- MasterEnsureDevice: bootstrap credentials (inputs → else persisted → activate)
bool MasterEnsureDevice()
{
    if(StringLen(g_deviceId) > 0 && StringLen(g_deviceSecret) > 0) return true;

    // 1) Persisted bootstrap state (manual split — MQL4 has no StringSplit)
    string saved = MasterReadState(PAT_MASTER_DEVICE_FILE);
    if(StringLen(saved) > 0)
    {
        string parts[4];
        int n = 0;
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
            return true;
        }
    }

    // 2) Auto-activate against the master license key
    if(StringLen(MasterLicenseKey) == 0)
    {
        if(!g_masterNetShown)
            Print("[MASTER_NODE] No device credentials and no MasterLicenseKey — set it in EA inputs.");
        return false;
    }
    string fp = MasterDeviceFingerprint();
    string body = "{\"license_key\":\"" + MasterLicenseKey + "\",\"client_type\":\"MT4\",\"role\":\"data\","
                  "\"fingerprint\":{\"machine_guid\":\"" + fp + "\",\"os\":\"Windows-MT4\"},"
                  "\"terminal\":{\"name\":\"" + MasterJSONEscape(AccountCompany()) + "\"}}";
    string response = "";
    int status = MasterHTTPPost(PATCloudURL + "/api/v1/devices/activate", body, response);
    if(status != 200)
    {
        Print("[MASTER_NODE] Device activation failed: HTTP ", status, " — ", StringSubstr(response, 0, 200));
        return false;
    }
    g_deviceId     = MasterJSONString(response, "device_id");
    g_deviceSecret = MasterJSONString(response, "device_secret");
    g_refreshToken = MasterJSONString(response, "refresh_token");
    g_accessToken  = MasterJSONString(response, "access_token");
    if(StringLen(g_deviceId) == 0 || StringLen(g_deviceSecret) == 0)
    {
        Print("[MASTER_NODE] Activation response missing device_id/device_secret.");
        return false;
    }
    MasterWriteState(PAT_MASTER_DEVICE_FILE, g_deviceId + "|" + g_deviceSecret + "|" + g_refreshToken + "\n");
    Print("[MASTER_NODE] Device activated: ", g_deviceId);
    return true;
}

//--- MasterEnsureAccessToken: rotate the access token via refresh_token grant.
bool MasterEnsureAccessToken()
{
    if(StringLen(g_accessToken) > 0 && TimeGMT() < g_tokenExpiry) return true;
    if(!MasterEnsureDevice()) return false;
    if(StringLen(g_refreshToken) == 0) return false;

    string body = "{\"refresh_token\":\"" + g_refreshToken + "\"}";
    string response = "";
    int status = MasterHTTPPost(PATCloudURL + "/api/v1/devices/refresh", body, response);
    if(status != 200)
    {
        // v1.29.2 production self-heal (parity with client EAs): a refresh
        // 401 means the stored token family is revoked (admin license reset,
        // reuse detection, device revocation). The old behavior returned
        // false and let the Master 401-loop forever with a dead feed — the
        // data node is the single source of market truth, so it MUST
        // self-recover. Clear ALL state and re-activate from MasterLicenseKey
        // on the next cycle.
        if(!g_masterNetShown)
            Print("[MASTER_NODE] Token refresh failed: HTTP ", status, " — clearing device state and re-activating.");
        g_deviceId = ""; g_deviceSecret = ""; g_refreshToken = ""; g_accessToken = ""; g_tokenExpiry = 0;
        MasterClearState(PAT_MASTER_DEVICE_FILE);
        MasterEnsureDevice();
        return false;
    }
    g_accessToken  = MasterJSONString(response, "access_token");
    string newRt   = MasterJSONString(response, "refresh_token");
    long expiresIn = StrToInteger(MasterJSONString(response, "expires_in"));
    if(StringLen(newRt) > 0) g_refreshToken = newRt;
    g_tokenExpiry = TimeGMT() + (expiresIn > 0 ? expiresIn - 60 : 82800);
    if(StringLen(g_accessToken) == 0) return false;
    MasterWriteState(PAT_MASTER_DEVICE_FILE, g_deviceId + "|" + g_deviceSecret + "|" + g_refreshToken + "\n");
    return true;
}

//+------------------------------------------------------------------+
//| File I/O using FILE_COMMON (legacy alias — now HTTP transport)    |
//+------------------------------------------------------------------+
void MasterWrite(string content)
{
    MasterAppend(content);
}

//+------------------------------------------------------------------+
//| MasterAppend — outbound message funnel (Option B, v1.19.0).       |
//| Takes the legacy "TYPE|{json}" wire line (or bare JSON for        |
//| bar-closed events) and POSTs it to the engine ingest endpoint     |
//| with the device Bearer JWT. Replaces the PAT_master_data.txt      |
//| pipe that the Windows Agent used to drain.                        |
//+------------------------------------------------------------------+
void MasterAppend(string content)
{
    if(!MasterEnsureDevice()) return;
    if(!MasterEnsureAccessToken()) return;

    string s = content;
    while(StringLen(s) > 0)
    {
        int c = StringGetChar(s, StringLen(s) - 1);
        if(c == '\n' || c == '\r' || c == 0) s = StringSubstr(s, 0, StringLen(s) - 1);
        else break;
    }
    if(StringLen(s) == 0) return;

    string msgType = "";
    string payload = s;
    int sep = StringFind(s, "|");
    if(sep > 0 && StringGetChar(s, 0) != '{')
    {
        // Legacy "TYPE|{json}" line.
        msgType = StringSubstr(s, 0, sep);
        payload = StringSubstr(s, sep + 1);
        if(StringFind(payload, "\"type\"") < 0 && StringGetChar(payload, 0) == '{')
            payload = "{\"type\":\"" + msgType + "\"," + StringSubstr(payload, 1);
    }
    // else: bare JSON (bar-closed event carries its own event_type).

    string headers = "Content-Type: application/json\r\n"
                     "Authorization: Bearer " + g_accessToken + "\r\n";
    uchar data[];
    StringToCharArray(payload, data, 0, WHOLE_ARRAY, CP_UTF8);
    ArrayResize(data, ArraySize(data) - 1);
    uchar result[];
    string resHeaders = "";
    string url = PATCloudURL + "/ingest/agent?agentId=" + MasterURLEncode(g_deviceId) + "&role=data";
    int status = WebRequest("POST", url, headers, 5000, data, result, resHeaders);
    if(status == 401)
    {
        // Access token expired — force refresh once and retry.
        g_accessToken = ""; g_tokenExpiry = 0;
        if(MasterEnsureAccessToken())
            MasterAppend(content);
        return;
    }
    if(status != 200)
    {
        g_ingestErrCount++;
        if(!g_masterNetShown)
        {
            // status == -1 means WebRequest could not complete at all (no HTTP
            // response): DNS failure, TLS handshake failure, blocked by the
            // WebRequest allowlist (Tools→Options→Expert Advisors), no network
            // route, or IPv6-only DNS on a terminal without IPv6. A 401/4xx/5xx
            // would be a real server reply. Capture GetLastError() to tell them
            // apart — 4011/4014 = allowlist/permissions, 6xxx = net/TLS.
            int err = GetLastError();
            if(status == -1)
                Print("[MASTER_NODE] ingest failed: HTTP -1 (transport failure, NO server reply) err=", err,
                      " url=", url, " — check WebRequest allowlist + DNS/TLS/IPv6 from the terminal host");
            else
                Print("[MASTER_NODE] ingest failed: HTTP ", status, " type=", msgType, " err=", err);
            g_masterNetShown = true;
        }
        return;
    }
    g_masterNetShown = false;
    g_ingestOkCount++;
}

//--- MasterURLEncode — percent-encoding for query values
string MasterURLEncode(string s)
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
//| MasterEdgePoll — fetch queued server commands (REQUEST_SNAPSHOT,  |
//| EMERGENCY_STOP …) for this data device via the control plane.     |
//| Replaces the PAT_resync.txt file nudge (agent transport removed). |
//+------------------------------------------------------------------+
void MasterEdgePoll()
{
    if(!MasterEnsureDevice()) return;
    string body = "{\"max_signals\":10}";
    string response = "";
    int status = MasterSignedPost("/api/v1/devices/edge-poll", body, response);
    if(status != 200) return;
    int pos = 0;
    while(true)
    {
        int qid = StringFind(response, "\"queue_id\":\"", pos);
        if(qid < 0) break;
        int qidEnd = StringFind(response, "\"", qid + 12);
        if(qidEnd < 0) break;
        string queueId = StringSubstr(response, qid + 12, qidEnd - (qid + 12));

        string typePat = "\"type\":\"";
        int tp = StringFind(response, typePat, qidEnd);
        int nextQ = StringFind(response, "\"queue_id\"", qidEnd);
        string msgType = "";
        if(tp >= 0 && (nextQ < 0 || tp < nextQ))
        {
            int te = StringFind(response, "\"", tp + StringLen(typePat));
            if(te > 0) msgType = StringSubstr(response, tp + StringLen(typePat), te - (tp + StringLen(typePat)));
        }

        // ACK first so the item leaves the queue permanently.
        string ackBody = "{\"queue_id\":\"" + queueId + "\",\"result\":{\"status\":\"PROCESSED\",\"type\":\"" + msgType + "\"}}";
        MasterSignedPost("/api/v1/devices/edge-ack", ackBody, response);

        if(msgType == "REQUEST_SNAPSHOT")
        {
            // Force an immediate snapshot (bypass the snapshot throttle).
            g_lastSnapshot = 0;
            Print("[MASTER_NODE] REQUEST_SNAPSHOT nudge received — forcing immediate snapshot.");
        }
        pos = qidEnd;
        if(pos > StringLen(response)) break;
    }
}
//+------------------------------------------------------------------+
//| Update on-chart panel                                             |
//+------------------------------------------------------------------+
//| TERMINAL DASHBOARD v2 — object-based two-column control panel     |
//| (design reference: MQL-REF.md flow layout; content adapted to the |
//| MASTER DATA NODE: cloud link, stream health, collection toggles,  |
//| account/symbol identity + DST-aware 24h session map. No trading.) |
//|                                                                   |
//| Overlap-proof by construction: rows are drawn with a running Y    |
//| cursor, every string is width-clipped with TextGetSize, and the   |
//| panel height is derived from the fixed row counts below (single   |
//| source of truth). Objects are fully cleared each frame — ghost    |
//| panels are impossible.                                            |
//|                                                                   |
//| Controls: drag header to move (persisted per symbol) · click      |
//| header to collapse · [PAUSE] button or F key toggles execution.   |
//+------------------------------------------------------------------+
#define PATUI_PREFIX "PATUI_"

// Terminal dashboard inputs
input bool   PAT_ShowDashboard  = true;   // show the terminal dashboard
input int    PAT_DashFontSize   = 8;      // dashboard font 6..12 (8 recommended)
input int    PAT_DashRefreshMs  = 400;    // dashboard repaint throttle ms (CPU friendly)
input int    PAT_DashX          = 430;    // initial panel X (drag to move; persisted)
input int    PAT_DashY          = 30;     // initial panel Y (drag to move; persisted)

//--- colour system: deep navy base, high-contrast text, semantic status
color PATUI_BG      = C'0,0,0';
color PATUI_PANEL   = C'6,8,12';
color PATUI_BG2     = C'10,12,16';
color PATUI_SECTION = C'16,24,40';
color PATUI_SECTXT  = C'120,220,255';
color PATUI_TXT     = C'255,255,255';
color PATUI_TXT2    = C'208,216,232';
color PATUI_DIM     = C'150,160,180';
color PATUI_GRAY    = C'110,120,140';
color PATUI_GRID    = C'40,48,64';
color PATUI_HDR     = C'12,18,30';
color PATUI_ACCENT  = C'0,160,255';
color PATUI_BORDER  = C'70,80,100';
color PATUI_UP      = C'0,255,140';
color PATUI_DN      = C'255,110,110';
color PATUI_WARN    = C'255,224,102';
color PATUI_GOLD    = C'255,215,64';
color PATUI_SYD     = C'0,176,255';
color PATUI_TOK     = C'155,89,255';
color PATUI_LON     = C'255,145,0';
color PATUI_NY      = C'0,230,118';
color PATUI_SYD_DIM = C'20,60,90';
color PATUI_TOK_DIM = C'50,32,90';
color PATUI_LON_DIM = C'90,54,0';
color PATUI_NY_DIM  = C'20,80,50';

//--- geometry (recomputed from PAT_DashFontSize each frame)
// Row counts are the single source of truth for the panel height:
//   left column:  3 sections + up to 19 rows (signal live + pause note)
//   right column: 3 sections + 12 rows + button clearance (2 rows)
int    PATUI_font = 8, PATUI_fontPx = 10, PATUI_dpi = 96;
double PATUI_dpiScale = 1.0;
int    PATUI_rh = 16, PATUI_hdrH = 30, PATUI_colW = 320, PATUI_pad = 12, PATUI_gap = 14, PATUI_panelW = 0;
int    PATUI_bodyTop = 0, PATUI_tlH = 0, PATUI_panelH = 0;
int    PATUI_x = 8, PATUI_y = 24;                 // live panel origin (draggable, persisted)
bool   PATUI_collapsed = false;
uint   PATUI_lastMs = 0, PATUI_lastDragMs = 0;
bool   PATUI_dragging = false, PATUI_maybeClick = false;
int    PATUI_dragOffX = 0, PATUI_dragOffY = 0, PATUI_dragStartX = 0, PATUI_dragStartY = 0;

int PATUI_SX(int v) { return (int)MathRound(v * PATUI_font / 9.0 * PATUI_dpiScale); }
int PATUI_FontOut(int fs) { return (int)MathMax(6, MathRound(fs / PATUI_dpiScale)); }

//--- account access guarded per platform (thin executor runs on both)
double PATUI_Equity()     {
#ifdef __MQL5__
   return AccountInfoDouble(ACCOUNT_EQUITY);
#else
   return AccountEquity();
#endif
}
double PATUI_Balance()    {
#ifdef __MQL5__
   return AccountInfoDouble(ACCOUNT_BALANCE);
#else
   return AccountBalance();
#endif
}
double PATUI_Margin()     {
#ifdef __MQL5__
   return AccountInfoDouble(ACCOUNT_MARGIN);
#else
   return AccountMargin();
#endif
}
double PATUI_FreeMargin() {
#ifdef __MQL5__
   return AccountInfoDouble(ACCOUNT_MARGIN_FREE);
#else
   return AccountFreeMargin();
#endif
}
long PATUI_Leverage()     {
#ifdef __MQL5__
   return AccountInfoInteger(ACCOUNT_LEVERAGE);
#else
   return (long)AccountLeverage();
#endif
}
long PATUI_TradeMode()    {
#ifdef __MQL5__
   return AccountInfoInteger(ACCOUNT_TRADE_MODE);
#else
   return AccountInfoInteger(ACCOUNT_TRADE_MODE); // MT4 build 600+
#endif
}
string PATUI_Currency()   {
#ifdef __MQL5__
   return AccountInfoString(ACCOUNT_CURRENCY);
#else
   return AccountCurrency();
#endif
}

//--- clip a string to maxW px, appending "…" when it does not fit
string PATUI_Clip(string s, int maxW, int fs)
{
    if(maxW <= 12) return (StringLen(s) == 0 ? s : "");
    if(StringLen(s) == 0) return s;
    TextSetFont("Consolas", PATUI_FontOut(fs), FW_DONTCARE, 0);
    uint w = 0, h = 0;
    if(!TextGetSize(s, w, h)) return s;
    if((int)w <= maxW) return s;
    while(StringLen(s) > 1)
    {
        s = StringSubstr(s, 0, StringLen(s) - 1);
        string t = s + "…";
        if(TextGetSize(t, w, h) && (int)w <= maxW) return t;
    }
    return "";
}

void PATUI_Recompute()
{
    PATUI_font = MathMax(6, MathMin(12, PAT_DashFontSize));
    long dpi = TerminalInfoInteger(TERMINAL_SCREEN_DPI);
    if(dpi < 96) dpi = 96;
    if(dpi > 480) dpi = 480;
    PATUI_dpi = (int)dpi;
    PATUI_dpiScale = (double)PATUI_dpi / 96.0;
    PATUI_fontPx = (int)MathRound(PATUI_font * PATUI_dpiScale);
    PATUI_rh = (int)MathRound((PATUI_font + 7) * PATUI_dpiScale);
    PATUI_hdrH = (int)MathRound((PATUI_font + 19) * PATUI_dpiScale);
    PATUI_colW = PATUI_SX(330);
    PATUI_pad = MathMax(8, PATUI_SX(12));
    PATUI_gap = MathMax(8, PATUI_SX(14));
    PATUI_panelW = PATUI_pad * 2 + PATUI_colW * 2 + PATUI_gap;
    PATUI_bodyTop = PATUI_hdrH + 6;
    PATUI_tlH = PATUI_SX(13) + 2 + 4 * PATUI_rh + PATUI_SX(14);
    int colHL = 3 * (PATUI_SX(20) + 2) + 19 * PATUI_rh + PATUI_SX(8);
    int colHR = 3 * (PATUI_SX(20) + 2) + 14 * PATUI_rh + PATUI_SX(8);
    int tlTop = PATUI_bodyTop + MathMax(colHL, colHR) + PATUI_SX(6);
    PATUI_panelH = tlTop + PATUI_tlH + 2 * PATUI_rh + PATUI_pad;
}

void PATUI_Rect(string n, int x, int y, int w, int h, color bg)
{
    string id = PATUI_PREFIX + n;
    if(ObjectFind(0, id) < 0) ObjectCreate(0, id, OBJ_RECTANGLE_LABEL, 0, 0, 0);
    ObjectSetInteger(0, id, OBJPROP_CORNER, CORNER_LEFT_UPPER);
    ObjectSetInteger(0, id, OBJPROP_XDISTANCE, x);
    ObjectSetInteger(0, id, OBJPROP_YDISTANCE, y);
    ObjectSetInteger(0, id, OBJPROP_XSIZE, MathMax(1, w));
    ObjectSetInteger(0, id, OBJPROP_YSIZE, MathMax(1, h));
    ObjectSetInteger(0, id, OBJPROP_BGCOLOR, bg);
    ObjectSetInteger(0, id, OBJPROP_COLOR, bg);
    ObjectSetInteger(0, id, OBJPROP_WIDTH, 1);
    ObjectSetInteger(0, id, OBJPROP_BACK, false);
    ObjectSetInteger(0, id, OBJPROP_HIDDEN, true);
    ObjectSetInteger(0, id, OBJPROP_SELECTABLE, false);
}

void PATUI_Label(string n, int x, int y, string txt, color c, int sz = -1)
{
    int req = (sz > 0 ? sz : PATUI_font);
    int fs = PATUI_FontOut(req);
    string id = PATUI_PREFIX + n;
    if(ObjectFind(0, id) < 0) ObjectCreate(0, id, OBJ_LABEL, 0, 0, 0);
    ObjectSetInteger(0, id, OBJPROP_CORNER, CORNER_LEFT_UPPER);
    ObjectSetInteger(0, id, OBJPROP_XDISTANCE, x);
    ObjectSetInteger(0, id, OBJPROP_YDISTANCE, y);
    ObjectSetInteger(0, id, OBJPROP_COLOR, c);
    ObjectSetInteger(0, id, OBJPROP_FONTSIZE, fs);
    ObjectSetString(0, id, OBJPROP_FONT, "Consolas");  // TextGetSize metrics == rendered metrics
    ObjectSetString(0, id, OBJPROP_TEXT, txt);
    ObjectSetInteger(0, id, OBJPROP_HIDDEN, true);
    ObjectSetInteger(0, id, OBJPROP_SELECTABLE, false);
}

//--- section bar at the current Y cursor
void PATUI_Section(string n, int col, int &y, string title)
{
    int x = PATUI_x + PATUI_pad + col * (PATUI_colW + PATUI_gap);
    PATUI_Rect("SB_" + n, x, y, PATUI_colW, PATUI_SX(20) - 3, PATUI_SECTION);
    PATUI_Rect("SL_" + n, x, y, 3, PATUI_SX(20) - 3, PATUI_ACCENT);
    PATUI_Label("ST_" + n, x + 8, y + ((PATUI_SX(20) - 3 - MathMax(6, PATUI_font - 1)) / 2),
                PATUI_Clip(PATUI_Upper(title), PATUI_colW - 14, MathMax(6, PATUI_font - 1)), PATUI_SECTXT, MathMax(6, PATUI_font - 1));
    y += PATUI_SX(20) + 2;
}

//--- one flowing row; text is clipped to the column so it can never bleed
void PATUI_Row(string n, int col, int &y, string txt, color c)
{
    int x = PATUI_x + PATUI_pad + col * (PATUI_colW + PATUI_gap) + 6;
    PATUI_Label("R_" + n, x, y + MathMax(0, (PATUI_rh - PATUI_fontPx) / 2), PATUI_Clip(txt, PATUI_colW - 12, PATUI_font), c, PATUI_font);
    y += PATUI_rh;
}

//--- two labels on one row (value left, status right-aligned in-column)
void PATUI_RowSplit(string n, int col, int &y, string left, string right, color cl, color cr)
{
    int x = PATUI_x + PATUI_pad + col * (PATUI_colW + PATUI_gap) + 6;
    PATUI_Label("R_" + n, x, y + MathMax(0, (PATUI_rh - PATUI_fontPx) / 2), PATUI_Clip(left, PATUI_colW - 100, PATUI_font), cl, PATUI_font);
    uint w = 0, h = 0;
    string r = PATUI_Clip(right, PATUI_SX(96), PATUI_font);
    TextSetFont("Consolas", PATUI_FontOut(PATUI_font), FW_DONTCARE, 0);
    TextGetSize(r, w, h);
    PATUI_Label("R_" + n + "b", x + PATUI_colW - 12 - PATUI_SX(4) - (int)w, y + MathMax(0, (PATUI_rh - PATUI_fontPx) / 2), r, cr, PATUI_font);
    y += PATUI_rh;
}

//--- one filled 24h segment (handles ranges that wrap past midnight)
void PATUI_Seg(string n, int bx, int by, int bw, int openMin, int closeMin, color col)
{
    int x1 = bx + (int)MathRound(openMin * bw / 1440.0);
    int x2 = bx + (int)MathRound(closeMin * bw / 1440.0);
    if(closeMin > openMin) PATUI_Rect(n, x1, by, MathMax(2, x2 - x1), 9, col);
    else { PATUI_Rect(n + "a", x1, by, MathMax(2, bx + bw - x1), 9, col); PATUI_Rect(n + "b", bx, by, MathMax(2, x2 - bx), 9, col); }
}

string PATUI_Upper(string s) { string t = s; StringToUpper(t); return t; }

// Chart timeframe to short string (portable MT4/MT5)
string ChartPeriodToString()
{
    int p = Period();
    if(p == PERIOD_M1)  return "M1";
    if(p == PERIOD_M5)  return "M5";
    if(p == PERIOD_M15) return "M15";
    if(p == PERIOD_M30) return "M30";
    if(p == PERIOD_H1)  return "H1";
    if(p == PERIOD_H4)  return "H4";
    if(p == PERIOD_D1)  return "D1";
    if(p == PERIOD_W1)  return "W1";
    if(p == PERIOD_MN1) return "MN";
    return "TF" + IntegerToString(p);
}

string PATUI_FmtMoney(double v) { return (v >= 0 ? "+" : "") + DoubleToString(v, 2); }

string PATUI_AccTypeName(long m)
{
    if(m == 0) return "DEMO";         // ACCOUNT_TRADE_MODE_DEMO
    if(m == 1) return "CONTEST";      // ACCOUNT_TRADE_MODE_CONTEST
    if(m == 2) return "REAL";         // ACCOUNT_TRADE_MODE_REAL
    return "UNKNOWN";
}

//===================== SESSION MAP (UTC, DST-aware) =====================
bool PATUI_IsLeap(int y) { return ((y % 4 == 0 && y % 100 != 0) || y % 400 == 0); }
int PATUI_DaysInMonth(int y, int m)
{
    int d[12] = {31,28,31,30,31,30,31,31,30,31,30,31};
    if(m == 2 && PATUI_IsLeap(y)) return 29;
    return d[m - 1];
}
datetime PATUI_MakeDT(int y, int mon, int day, int hour, int minute = 0, int sec = 0)
{
    MqlDateTime x;
    x.year = y; x.mon = mon; x.day = day; x.hour = hour; x.min = minute; x.sec = sec;
    x.day_of_week = 0; x.day_of_year = 0;
    return StructToTime(x);
}
int PATUI_DayOfWeekUTC(int y, int mon, int day)
{
    MqlDateTime x; TimeToStruct(PATUI_MakeDT(y, mon, day, 12), x); return x.day_of_week;
}
int PATUI_NthSunday(int y, int mon, int nth)
{
    int dow = PATUI_DayOfWeekUTC(y, mon, 1);
    int first = 1 + ((7 - dow) % 7);
    return first + (nth - 1) * 7;
}
int PATUI_LastSunday(int y, int mon)
{
    int last = PATUI_DaysInMonth(y, mon);
    int dow = PATUI_DayOfWeekUTC(y, mon, last);
    return last - dow;
}
bool PATUI_LondonDST(datetime utc)
{
    MqlDateTime d; TimeToStruct(utc, d);
    datetime start = PATUI_MakeDT(d.year, 3, PATUI_LastSunday(d.year, 3), 1);
    datetime stop  = PATUI_MakeDT(d.year, 10, PATUI_LastSunday(d.year, 10), 1);
    return (utc >= start && utc < stop);
}
bool PATUI_NewYorkDST(datetime utc)
{
    MqlDateTime d; TimeToStruct(utc, d);
    datetime start = PATUI_MakeDT(d.year, 3, PATUI_NthSunday(d.year, 3, 2), 7);
    datetime stop  = PATUI_MakeDT(d.year, 11, PATUI_NthSunday(d.year, 11, 1), 6);
    return (utc >= start && utc < stop);
}
// Australia/Sydney: first Sunday of October 02:00 AEST → first Sunday of April 03:00 AEDT
bool PATUI_SydneyDST(datetime utc)
{
    MqlDateTime d; TimeToStruct(utc, d);
    datetime octStartThis = PATUI_MakeDT(d.year, 10, PATUI_NthSunday(d.year, 10, 1), 2) - 10 * 3600;
    datetime aprEndThis   = PATUI_MakeDT(d.year, 4, PATUI_NthSunday(d.year, 4, 1), 3) - 11 * 3600;
    if(d.mon <= 4)
    {
        datetime octPrev = PATUI_MakeDT(d.year - 1, 10, PATUI_NthSunday(d.year - 1, 10, 1), 2) - 10 * 3600;
        return (utc >= octPrev && utc < aprEndThis);
    }
    if(d.mon >= 10)
    {
        datetime aprNext = PATUI_MakeDT(d.year + 1, 4, PATUI_NthSunday(d.year + 1, 4, 1), 3) - 11 * 3600;
        return (utc >= octStartThis && utc < aprNext);
    }
    return (utc >= octStartThis && utc < aprEndThis);
}
int PATUI_MinuteOfDay(datetime t) { MqlDateTime d; TimeToStruct(t, d); return d.hour * 60 + d.min; }
int PATUI_WrapMin(int m) { m %= 1440; if(m < 0) m += 1440; return m; }
bool PATUI_InWindow(int x, int a, int b) { if(a <= b) return (x >= a && x < b); return (x >= a || x < b); }
string PATUI_FmtHHMM(int m) { m = PATUI_WrapMin(m); return StringFormat("%02d:%02d", m / 60, m % 60); }

// Session UTC bounds (DST-aware: Sydney UTC+10/+11, Tokyo +9, London 0/+1, NY -5/-4)
void PATUI_SessionBounds(datetime utc, int &sydO, int &sydC, int &tokO, int &tokC, int &lonO, int &lonC, int &nyO, int &nyC)
{
    int sydOff = (PATUI_SydneyDST(utc) ? 11 * 60 : 10 * 60);
    sydO = PATUI_WrapMin(8 * 60 - sydOff);          sydC = PATUI_WrapMin(17 * 60 - sydOff);
    tokO = PATUI_WrapMin(9 * 60 - 9 * 60);          tokC = PATUI_WrapMin(18 * 60 - 9 * 60);
    int lonOff = (PATUI_LondonDST(utc) ? 60 : 0);
    lonO = PATUI_WrapMin(8 * 60 - lonOff);          lonC = PATUI_WrapMin((16 * 60 + 30) - lonOff);
    int nyOff = (PATUI_NewYorkDST(utc) ? 4 * 60 : 5 * 60);
    nyO  = PATUI_WrapMin(8 * 60 + nyOff);           nyC  = PATUI_WrapMin(17 * 60 + nyOff);
}

color PATUI_SessionColor(int i, bool active)
{
    if(i == 0) return (active ? PATUI_SYD : PATUI_SYD_DIM);
    if(i == 1) return (active ? PATUI_TOK : PATUI_TOK_DIM);
    if(i == 2) return (active ? PATUI_LON : PATUI_LON_DIM);
    return (active ? PATUI_NY : PATUI_NY_DIM);
}

//===================== DATA-NODE STATE HELPERS =====================
// Data-collection toggle short list (from MN inputs)
string PATUI_DataToggles()
{
    return (SendIndicators ? "[IND] " : "") + (SendMultiTF ? "[MTF] " : "") +
           (SendAccountInfo ? "[ACCT] " : "") + (SendSymbolInfo ? "[SYM]" : "");
}

//===================== DASHBOARD RENDER =====================
void PATUI_DashDestroy() { ObjectsDeleteAll(0, PATUI_PREFIX, 0, -1); }

void PATUI_OnDeinit() { PATUI_DashDestroy(); }

void PATUI_DashUpdate(bool force = false)
{
    if(!PAT_ShowDashboard) return;
    uint ms = GetTickCount();
    if(!force && PATUI_lastMs > 0 && ms - PATUI_lastMs < (uint)MathMax(100, PAT_DashRefreshMs)) return;
    PATUI_lastMs = ms;
    PATUI_Recompute();
    // Full clear every frame (reference pattern): stale objects from any
    // earlier layout can never linger, so ghost panels are impossible.
    ObjectsDeleteAll(0, PATUI_PREFIX, 0, -1);

    int x = PATUI_x, y = PATUI_y;

    //---- header (collapsed mode = single bar)
    if(PATUI_collapsed)
    {
        PATUI_Rect("BG", x, y, PATUI_panelW, PATUI_hdrH, PATUI_BG);
        PATUI_Rect("HDR", x, y, PATUI_panelW, PATUI_hdrH, PATUI_HDR);
        PATUI_Rect("HL", x, y, 3, PATUI_hdrH, PATUI_ACCENT);
        PATUI_Label("T", x + 10, y + ((PATUI_hdrH - PATUI_font - 2) / 2),
                    PATUI_Clip(PATUI_Upper("PREDICT-A-TRADE") + " [" + g_symbol + "]", PATUI_panelW - PATUI_SX(120), PATUI_font + 2), clrWhite, PATUI_font + 2);
        PATUI_Label("CS", x + PATUI_panelW - PATUI_SX(90), y + ((PATUI_hdrH - PATUI_font) / 2),
                    (g_connection == "CONNECTED" ? "ONLINE" : "OFFLINE"),
                    (g_connection == "CONNECTED" ? PATUI_UP : PATUI_DN), PATUI_font);
        PATUI_Label("CH", x + PATUI_panelW - PATUI_SX(18), y + ((PATUI_hdrH - PATUI_font) / 2), "+", PATUI_TXT, PATUI_font);
        ChartRedraw();
        return;
    }

    PATUI_Rect("BG", x, y, PATUI_panelW, PATUI_panelH, PATUI_BG);
    PATUI_Rect("HDR", x, y, PATUI_panelW, PATUI_hdrH, PATUI_HDR);
    PATUI_Rect("HL", x, y, 3, PATUI_hdrH, PATUI_ACCENT);
    PATUI_Label("T", x + 10, y + ((PATUI_hdrH - PATUI_font - 2) / 2),
                PATUI_Clip(PATUI_Upper("PAT MASTER DATA NODE") + "  [" + g_symbol + " " + ChartPeriodToString() + "]", PATUI_panelW - PATUI_SX(220), PATUI_font + 2), clrWhite, PATUI_font + 2);

    //---- header status chips (right-aligned, fixed slots)
    int chipX = x + PATUI_panelW - PATUI_SX(8);
    PATUI_Label("CH_COLL", chipX - PATUI_SX(20), y + ((PATUI_hdrH - PATUI_font) / 2), "[-]", PATUI_TXT2, PATUI_font);
    chipX -= PATUI_SX(28);
    PATUI_Label("CH_STATE", chipX - PATUI_SX(70), y + ((PATUI_hdrH - PATUI_font) / 2),
                (g_connection == "CONNECTED" ? "STREAMING" : "OFFLINE"),
                (g_connection == "CONNECTED" ? PATUI_UP : PATUI_DN), PATUI_font);

    //---- derived values
    double eq  = PATUI_Equity(), bal = PATUI_Balance();
    double point = SymbolInfoDouble(g_symbol, SYMBOL_POINT);
    double spread = 0;
#ifdef __MQL5__
    spread = (point > 0 ? (SymbolInfoDouble(g_symbol, SYMBOL_ASK) - SymbolInfoDouble(g_symbol, SYMBOL_BID)) / point : 0);
#else
    spread = MarketInfo(g_symbol, MODE_SPREAD);
#endif
    datetime utc = TimeGMT();
    int um = PATUI_MinuteOfDay(utc);
    int so, sc, to, tc, lo, lc, no, nc;
    PATUI_SessionBounds(utc, so, sc, to, tc, lo, lc, no, nc);
    MqlDateTime dtS; TimeToStruct(TimeCurrent(), dtS);
    string srvHM = StringFormat("%02d:%02d", dtS.hour, dtS.min);

    //================ LEFT COLUMN (flow) ==================
    int yL = PATUI_bodyTop;
    PATUI_Section("LS", 0, yL, "master data node");
    PATUI_RowSplit("L_MODE", 0, yL, "Mode DATA COLLECTION", "NO trading", PATUI_SECTXT, PATUI_GRAY);
    PATUI_RowSplit("L_LINK", 0, yL, "Link " + g_connection, "ticks " + IntegerToString((long)g_tickCount),
                   (g_connection == "CONNECTED" ? PATUI_UP : PATUI_DN), PATUI_TXT2);
    PATUI_Row("L_SNAP", 0, yL, "Snapshots sent " + IntegerToString((long)g_snapshotCount), PATUI_TXT2);
    PATUI_Row("L_TOG", 0, yL, "Feed " + PATUI_DataToggles(), PATUI_TXT);
    PATUI_RowSplit("L_SYM", 0, yL, "Symbol " + g_symbol, "Server " + srvHM, PATUI_TXT2, PATUI_DIM);

    PATUI_Section("LI", 0, yL, "identity");
    PATUI_Row("I_BRK", 0, yL, "Broker " + (g_broker == "" ? "—" : g_broker), PATUI_TXT2);
    PATUI_RowSplit("I_ACC", 0, yL, "Account " + g_accountID, PATUI_AccTypeName(PATUI_TradeMode()), PATUI_TXT2, PATUI_SECTXT);
    PATUI_Row("I_LEV", 0, yL, "Leverage 1:" + IntegerToString(PATUI_Leverage()) + "  " + PATUI_Currency(), PATUI_DIM);

    PATUI_Section("LH", 0, yL, "about");
    PATUI_Row("H_NOTE", 0, yL, "Streams ticks/snapshots to the engine.", PATUI_DIM);
    PATUI_Row("H_FREE", 0, yL, "License-free by design. No orders.", PATUI_DIM);
    PATUI_Row("H_TIME", 0, yL, "Now " + TimeToString(TimeCurrent(), TIME_DATE | TIME_SECONDS), PATUI_GRAY);

    //================ RIGHT COLUMN (flow) ==================
    int yR = PATUI_bodyTop;
    PATUI_Section("RS", 1, yR, "collection status");
    PATUI_Row("S_IND", 1, yR, "Indicators " + (SendIndicators ? "ON" : "OFF"), (SendIndicators ? PATUI_UP : PATUI_GRAY));
    PATUI_Row("S_MTF", 1, yR, "Multi-timeframe bars " + (SendMultiTF ? "ON" : "OFF"), (SendMultiTF ? PATUI_UP : PATUI_GRAY));
    PATUI_Row("S_ACCT", 1, yR, "Account info " + (SendAccountInfo ? "ON" : "OFF"), (SendAccountInfo ? PATUI_UP : PATUI_GRAY));
    PATUI_Row("S_SYM", 1, yR, "Symbol spec " + (SendSymbolInfo ? "ON" : "OFF"), (SendSymbolInfo ? PATUI_UP : PATUI_GRAY));

    PATUI_Section("RM", 1, yR, "market");
    PATUI_Row("M_EQ", 1, yR, "Equity " + DoubleToString(eq, 2) + "  Bal " + DoubleToString(bal, 2) + " " + PATUI_Currency(), PATUI_TXT2);
    PATUI_Row("M_SPR", 1, yR, "Spread " + DoubleToString(spread, 1) + " pts", PATUI_TXT2);
    string sessNow = "none";
    if(PATUI_InWindow(um, lo, lc) && PATUI_InWindow(um, no, nc)) sessNow = "LONDON/NY";
    else if(PATUI_InWindow(um, lo, lc)) sessNow = "LONDON";
    else if(PATUI_InWindow(um, no, nc)) sessNow = "NEW YORK";
    else if(PATUI_InWindow(um, to, tc)) sessNow = "TOKYO";
    else if(PATUI_InWindow(um, so, sc)) sessNow = "SYDNEY";
    PATUI_RowSplit("M_SES", 1, yR, "Session " + sessNow, "UTC " + PATUI_FmtHHMM(um),
                   (StringFind(sessNow, "/") >= 0 ? PATUI_GOLD : (sessNow != "none" ? PATUI_UP : PATUI_GRAY)), PATUI_DIM);
    string ovl = "";
    if(PATUI_InWindow(um, so, sc) && PATUI_InWindow(um, to, tc)) ovl = "SYD+TOK";
    else if(PATUI_InWindow(um, to, tc) && PATUI_InWindow(um, lo, lc)) ovl = "TOK+LDN";
    else if(PATUI_InWindow(um, lo, lc) && PATUI_InWindow(um, no, nc)) ovl = "LDN+NY";
    PATUI_Row("M_OVL", 1, yR, "Overlap " + (ovl == "" ? "--" : ovl), (ovl == "" ? PATUI_GRAY : PATUI_GOLD));

    PATUI_Section("RH", 1, yR, "stream health");
    PATUI_RowSplit("H_TICK", 1, yR, "Ticks sent " + IntegerToString((long)g_tickCount), "conn " + g_connection,
                   PATUI_UP, (g_connection == "CONNECTED" ? PATUI_UP : PATUI_DN));
    PATUI_RowSplit("H_SNAP", 1, yR, "Snapshots " + IntegerToString((long)g_snapshotCount),
                   "feed " + ((SendIndicators || SendMultiTF) ? "rich" : "lean"), PATUI_TXT2, PATUI_DIM);
    PATUI_Row("H_ROLE", 1, yR, "Role: passive data publisher", PATUI_DIM);
    PATUI_Row("H_SAFE", 1, yR, "No orders · No license · No keys", PATUI_DIM);

    //================ 24H UTC SESSION TIMELINE (flow) =====================
    int tlTop = MathMax(yL, yR) + PATUI_SX(6);
    int labW = PATUI_SX(34);
    int barX = x + PATUI_pad + labW, barW = PATUI_panelW - PATUI_pad * 2 - labW;
    PATUI_Rect("TL_BG", x + PATUI_pad, tlTop, PATUI_panelW - PATUI_pad * 2, PATUI_tlH, PATUI_BG2);
    PATUI_Label("TL_H", x + PATUI_pad + 4, tlTop + 2, "SESSION MAP (UTC, DST-AWARE)", PATUI_SECTXT, MathMax(6, PATUI_font - 1));
    int openM[4], closeM[4];
    openM[0] = so; closeM[0] = sc; openM[1] = to; closeM[1] = tc; openM[2] = lo; closeM[2] = lc; openM[3] = no; closeM[3] = nc;
    string names[4] = {"SYD", "TOK", "LDN", "NY"};
    int byT = tlTop + PATUI_SX(15);
    for(int i = 0; i < 4; i++)
    {
        int rowY = byT + i * PATUI_rh;
        bool act = PATUI_InWindow(um, openM[i], closeM[i]);
        PATUI_Label("TL_N" + IntegerToString(i), x + PATUI_pad + 4, rowY + ((PATUI_rh - MathMax(6, PATUI_font - 1)) / 2),
                    names[i], (act ? PATUI_SessionColor(i, true) : PATUI_TXT2), MathMax(6, PATUI_font - 1));
        PATUI_Rect("TL_T" + IntegerToString(i), barX, rowY + ((PATUI_rh - 9) / 2), barW, 9, PATUI_PANEL);
        for(int hh = 3; hh < 24; hh += 3)
            PATUI_Rect("TL_G" + IntegerToString(i) + "_" + IntegerToString(hh), barX + (int)MathRound(hh * barW / 24.0), rowY + ((PATUI_rh - 9) / 2), 1, 9, PATUI_GRID);
        PATUI_Seg("TL_S" + IntegerToString(i), barX, rowY + ((PATUI_rh - 9) / 2), barW, openM[i], closeM[i], PATUI_SessionColor(i, act));
    }
    int nowX = barX + (int)MathRound(um * barW / 1440.0);
    PATUI_Rect("TL_NOW", nowX, byT - 2, 2, 4 * PATUI_rh, clrWhite);
    PATUI_Label("TL_NOWL", MathMax(barX, MathMin(nowX - PATUI_SX(30), barX + barW - PATUI_SX(70))), tlTop + PATUI_tlH - PATUI_SX(13),
                "NOW " + PATUI_FmtHHMM(um) + " UTC", clrWhite, MathMax(6, PATUI_font - 1));

    //================ FOOTER ==============================
    int fy = tlTop + PATUI_tlH + PATUI_SX(2);
    string foot = "DATA NODE — no trading   drag header = move   click = collapse";
    PATUI_Label("FT", x + PATUI_pad + 4, fy, PATUI_Clip(foot, PATUI_panelW - PATUI_pad * 2 - 8, MathMax(6, PATUI_font - 1)), PATUI_GRAY, MathMax(6, PATUI_font - 1));
    ChartRedraw();
}

//===================== LIFECYCLE + EVENT HOOKS =====================
void PATUI_OnInit()
{
    PATUI_x = MathMax(0, PAT_DashX);
    PATUI_y = MathMax(0, PAT_DashY);
    // restore last dragged position (persisted per symbol, like the reference)
    string gvx = "PATUI_X_" + g_symbol, gvy = "PATUI_Y_" + g_symbol;
    if(GlobalVariableCheck(gvx) && GlobalVariableCheck(gvy))
    {
        int sx = (int)GlobalVariableGet(gvx), sy = (int)GlobalVariableGet(gvy);
        if(sx >= 0 && sy >= 0) { PATUI_x = sx; PATUI_y = sy; }
    }
    ChartSetInteger(0, CHART_EVENT_MOUSE_MOVE, true);
    PATUI_Recompute();
    PATUI_DashUpdate(true);
}

void PATUI_OnChartEvent(const int id, const long &lparam, const double &dparam, const string &sparam)
{
    if(!PAT_ShowDashboard) return;

    //--- header click = collapse / expand (if the press did not become a drag)
    if(id == CHARTEVENT_CLICK && PATUI_maybeClick)
    {
        int mx = (int)lparam, my = (int)dparam;
        PATUI_maybeClick = false;
        if(!PATUI_dragging && PATUI_panelW > 0 && mx >= PATUI_x && mx <= PATUI_x + PATUI_panelW && my >= PATUI_y && my <= PATUI_y + PATUI_hdrH)
        {
            PATUI_collapsed = !PATUI_collapsed;
            PATUI_DashUpdate(true);
        }
        return;
    }

    //--- header drag (persisted)
    if(id == CHARTEVENT_MOUSE_MOVE)
    {
        int mx = (int)lparam, my = (int)dparam;
        int flags = (int)StringToInteger(sparam);
        bool lmb = ((flags & 1) != 0);
        uint now = GetTickCount();

        if(lmb && !PATUI_dragging)
        {
            if(mx >= PATUI_x && mx <= PATUI_x + PATUI_panelW && my >= PATUI_y && my <= PATUI_y + PATUI_hdrH)
            {
                PATUI_maybeClick = true;
                PATUI_dragging = true;
                PATUI_dragOffX = mx - PATUI_x; PATUI_dragOffY = my - PATUI_y;
                PATUI_dragStartX = mx; PATUI_dragStartY = my;
            }
        }
        else if(PATUI_dragging && lmb)
        {
            if(MathAbs(mx - PATUI_dragStartX) > 4 || MathAbs(my - PATUI_dragStartY) > 4) PATUI_maybeClick = false;
            int nx = mx - PATUI_dragOffX, ny = my - PATUI_dragOffY;
            int cw = (int)ChartGetInteger(0, CHART_WIDTH_IN_PIXELS);
            int ch = (int)ChartGetInteger(0, CHART_HEIGHT_IN_PIXELS);
            if(nx < 0) nx = 0;
            if(ny < 0) ny = 0;
            if(nx + PATUI_panelW > cw) nx = MathMax(0, cw - PATUI_panelW);
            if(ny + PATUI_hdrH > ch) ny = MathMax(0, ch - PATUI_hdrH);
            if(nx != PATUI_x || ny != PATUI_y)
            {
                PATUI_x = nx; PATUI_y = ny;
                GlobalVariableSet("PATUI_X_" + g_symbol, PATUI_x);
                GlobalVariableSet("PATUI_Y_" + g_symbol, PATUI_y);
                if(now - PATUI_lastDragMs >= (uint)MathMax(30, PAT_DashRefreshMs / 4)) { PATUI_lastDragMs = now; PATUI_DashUpdate(true); }
            }
        }
        else if(PATUI_dragging && !lmb)
        {
            PATUI_dragging = false;
            PATUI_DashUpdate(true);
        }
    }
}
//+------------------------------------------------------------------+

//+------------------------------------------------------------------+
//| CHART EVENTS — dashboard drag / collapse (v1.29.4)                 |
//+------------------------------------------------------------------+
void OnChartEvent(const int id, const long &lparam, const double &dparam, const string &sparam)
{
    PATUI_OnChartEvent(id, lparam, dparam, sparam);
}
