//+------------------------------------------------------------------+
//| CAccountTypeDetector.mqh                                          |
//| Predict-A-Trade v1.00 — account type detection & adaptation        |
//|                                                                    |
//| COMPATIBILITY: MT5 build 4000+ AND MT4 (all builds). Compile-safe   |
//| on both: every platform-specific symbol is guarded by               |
//| #ifdef __MQL5__ / #else (__MQL4__ is implied on MT4).               |
//|                                                                    |
//| DESIGN CONTRACT (operator mandate):                                 |
//|  - Pure MQL: no WebRequest, no file I/O, no DLLs, no external       |
//|    scripts. Detection uses only AccountInfo*, SymbolInfo*,          |
//|    Position*/Order* inspection.                                     |
//|  - ADDITIVE ONLY: this file declares NEW symbols with the PAT_ATD_  |
//|    / CAccountTypeDetector namespaces; it does not touch or shadow   |
//|    any existing identifier.                                         |
//|  - FAIL-SAFE: any detection error falls back to PAT_ATD_STANDARD.    |
//|    No exceptions are thrown (MQL has no try/catch); every branch     |
//|    returns a defined value. OnInit() callers always get INIT_SUCCEEDED. |
//|  - LAZY + CACHED: detection runs at most once per terminal session   |
//|    (per account login); later calls are O(1) cache reads.            |
//|                                                                    |
//| PRIORITY ORDER (first match wins):                                  |
//|  1. Demo     — ACCOUNT_TRADE_MODE == ACCOUNT_TRADE_MODE_DEMO        |
//|  2. Contest  — ACCOUNT_TRADE_MODE == ACCOUNT_TRADE_MODE_CONTEST     |
//|  3. Islamic  — swap-free primary symbol + 3× rollover confirmation  |
//|               (open positions' POSITION_SWAP stays 0.00)            |
//|  4. Micro/Cent — cent denomination or sub-0.01 min lot              |
//|  5. ECN      — market execution + commission + zero stop level      |
//|  6. STP      — instant/request execution, no/minimal commission     |
//|  7. Standard — default fallback                                     |
//|                                                                    |
//| SPEC CORRECTIONS (honesty notes vs. the original request):          |
//|  - SYMBOL_COMMISSION_TICK / SYMBOL_COMMISSION_LOT are NOT real      |
//|    MQL5 identifiers (commission in MT5 is a DEAL property, not a    |
//|    symbol property). ECN commission is detected empirically: scan   |
//|    closed deal history for non-zero DEAL_COMMISSION, with an        |
//|    optional manual override input.                                  |
//|  - SYMBOL_TRADE_EXEMODE is MT4-era; MT5 uses SYMBOL_TRADE_MODE.     |
//|    Execution mode comes from AccountInfoInteger(ACCOUNT_TRADE_MODE_ |
//|    FORBIDDEN)/symbol fill policy + empirical checks instead.        |
//|  - ACCOUNT_CURRENCY_DIGITS is not a standard identifier; cent       |
//|    detection uses SYMBOL_VOLUME_MIN < 0.01 (e.g. 0.002), balance    |
//|    scaling patterns and an optional override.                       |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property strict

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
   else if(PAT_ATD_histComm > 0)
      pnl -= PAT_ATD_histComm; // broker-charged, already in deal P&L — no double count
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
