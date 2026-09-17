//+------------------------------------------------------------------+
//|                                  Predict-A-Trade-XAUUSD.mq5      |
//| Predict-A-Trade XAUUSD Institutional Intraday Confluence v1.00   |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade - XAUUSD Institutional Intraday Confluence v1.00"
#property version   "1.00"
#property strict
#property description "XAUUSD-only XAUUSD Institutional Intraday Confluence Suite"

//==============================================================
// Defines
//==============================================================
#define IES_MAX_LIQ     10
#define IES_MAX_FVG     10
#define IES_MAX_OB      10
#define IES_MTF_N       4
#define IES_HUD_PREFIX  "IES_HUD_"
#define IES_SMC_PREFIX  "IES_SMC_"

//==============================================================
// Enums
//==============================================================
enum ENUM_IES_REGIME
  {
   IES_REGIME_TRENDING_BULLISH = 0,
   IES_REGIME_TRENDING_BEARISH = 1,
   IES_REGIME_RANGE            = 2,
   IES_REGIME_MEAN_REVERSION   = 3,
   IES_REGIME_BREAKOUT         = 4,
   IES_REGIME_HIGH_VOLATILITY  = 5,
   IES_REGIME_SQUEEZE          = 6
  };

enum ENUM_IES_SESSION
  {
   IES_SESSION_SYDNEY    = 0,
   IES_SESSION_ASIAN     = 1,
   IES_SESSION_LONDON    = 2,
   IES_SESSION_NEW_YORK  = 3,
   IES_SESSION_OVERLAP   = 4,
   IES_SESSION_OFF_HOURS = 5
  };

enum ENUM_TRAIL_MODE { TRAIL_ATR = 0, TRAIL_SWING_STRUCTURE = 1, TRAIL_VWAP = 2 };
enum ENUM_UI_MODE    { UI_GRAPHICAL_HUD = 0, UI_TEXT_COMMENT = 1, UI_NONE = 2 };
enum ENUM_NEWS_IMPACT_FILTER { NEWS_DISABLED = 0, NEWS_HIGH_ONLY = 1, NEWS_HIGH_AND_MEDIUM = 2 };

enum ENUM_ASTRO_BODY
  {
   ASTRO_SUN = 0, ASTRO_MOON, ASTRO_MERCURY, ASTRO_VENUS, ASTRO_MARS,
   ASTRO_JUPITER, ASTRO_SATURN, ASTRO_RAHU, ASTRO_KETU, ASTRO_URANUS,
   ASTRO_NEPTUNE, ASTRO_PLUTO, ASTRO_BODY_COUNT
  };

enum ENUM_AYANAMSA_TYPE
  { AYANAMSA_LAHIRI = 0, AYANAMSA_RAMAN, AYANAMSA_KP, AYANAMSA_TRUE_CHITRA };

enum ENUM_SIGNAL_TIER
  { TIER_REJECT = 0, TIER_WATCH = 1, TIER_C = 2, TIER_B = 3, TIER_A = 4, TIER_APLUS = 5 };

//==============================================================
// Structs
//==============================================================
struct SAstroBody
  {
   string name;
   double tropLon, tropLat, sidLon, dailySpeed;
   bool   isRetro, isCombust;
   int    rashi, deg, min, sec, nakshatra, pada, totalPada;
   string dignity;
   double shadbala;
  };

struct SMicroscopicTradeMap
  {
   int    stateM1, stateM5, stateM15, stateM30, stateH1, stateH4, stateD1;
   double compositeBias;
   string rating;
   int    pada108Entry;
   string padaLord;
  };

struct SAstroContext
  {
   double      ayanamsa;
   SAstroBody  bodies[ASTRO_BODY_COUNT];
   int         horaPlanet;
   string      horaName;
   int         horaMinRemaining;
   bool        isRahuKalam, isGulika, isYamaganda;
   string      demonStatus;
   int         tithi;
   string      tithiName, paksha, lunarPhase;
   double      phaseAngle;
   bool        isRiktaTithi;
   int         moonNakshatra;
   string      moonNakName, moonNakLord;
   int         moonPada, moonTotalPada;
   string      taraBala;
   int         taraBalaScore;
   string      vimshottariLord, vimshottariSubLord;
   double      vimshottariBalance;
   string      shodshottariLord, ashtottariAridradiLord;
   string      ashtottariKartikadiLord, shashtiHayaniLord;
   string      saturnRashi, saturnAspectAlert;
   bool        saturnRetro;
   double      fearIndex;
   bool        hasVishYoga, hasGrahanYoga, hasYamaYoga, hasAngarakYoga;
   bool        isGandanta, isApocalypseTrigger, isObsessionGap;
   bool        hasGajakesari, hasBudhaditya, hasChandraMangala;
   bool        hasGuruMangala, hasDhanaYoga;
   double      cosmicScore, sizingMultiplier;
   SMicroscopicTradeMap microMap;
  };

struct SFvgItem
  {
   double   top, bottom, midpoint;
   datetime time;
   bool     isBullish, mitigated;
  };

struct SObItem
  {
   double   top, bottom;
   datetime time;
   bool     isBullish, mitigated, isBreaker;
  };

struct SIESParams
  {
   int    emaPeriods[5], smaPeriods[3];
   int    macdFast, macdSlow, macdSignal, adxPeriod;
   double sarStep, sarMax;
   int    ichiTenkan, ichiKijun, ichiSenkou, ichiChikouShift;
   int    rsiPeriod;
   int    stochK, stochD, stochSlow;
   int    srsiLen, srsiSmoothK, srsiSmoothD;
   int    cciPeriod, atrPeriod, bbPeriod;
   double bbDev;
   int    zLookback, vwapRollLen;
   double vwapBandSigma1, vwapBandSigma2, vwapBandSigma3;
   int    swingLR, structScanBars;
   double sweepMinATR, fvgMinATR, obDispATR;
   int    obScanBars;
   double eqTolATR;
   double fibLevels[6];
   double gzLo, gzHi;
   double confTolATR;
   double regAdxTrend, regAdxRange;
   double regBBWidthZ, regAtrZ;
   double chopTrend, chopRange;
   ENUM_TIMEFRAMES mtfTFs[IES_MTF_N];
   double mtfW[IES_MTF_N];
   int    newsWindowBeforeMin, newsWindowAfterMin;
   double dojiBodyMax, pinWickMin, dispBodyATR;
   double regCompRatio;
   int    streakScan, ibMinutes, linLen, chopLen;
  };

struct STradePlan
  {
   bool     valid;
   int      dir;
   double   entry, sl, tp1, tp2, tp3;
   double   slDist, slPts;
   double   r1, r2, r3;
   double   netRR1, netRR2, netRR3;
   double   grossRR1, grossRR2, grossRR3;
   double   lots, riskMoney, estCostMoney;
   string   reject;
  };

struct SFeatures
  {
   double ema9, ema21, ema50, ema100, ema200;
   int    emaCross921;
   double sma50, sma100, sma200;
   bool   emaStackBull, emaStackBear;
   double macdMain, macdSignal, macdHist;
   bool   macdBullCross, macdBearCross;
   double adx, adxPlusDI, adxMinusDI, adxSlope;
   double psar; bool psarLong;
   double ichiTenkan, ichiKijun, ichiSenkouA, ichiSenkouB;
   bool   ichiChikouBull;
   double ichiCloudTop, ichiCloudBot; int ichiCloudPos;
   double rsi, stochK, stochD, srsiRaw, srsiK, srsiD, cci;
   double atr, atrZ;
   double bbUpper, bbLower, bbMiddle, bbWidth, bbWidthZ;
   bool   bbBullRev, bbBearRev;
   bool   squeezeOn, squeezeRelease;
   double chop;
   double obv, obvZ, tickVolZ, cvd, cvdZ;
   bool   volumeSpike, absorption;
   double vwapSession, vwapUpper1, vwapLower1, vwapUpper2, vwapLower2;
   double vwapRoll, vwapZ;
   double swingHigh, swingLow, prevSwingHigh, prevSwingLow;
   int    trendState, bosDir, chochDir, mssDir;
   bool   mss;
   double liqHi[IES_MAX_LIQ], liqLo[IES_MAX_LIQ];
   int    liqHiN, liqLoN;
   bool   sweepHi, sweepLo, eqhDetected, eqlDetected;
   SFvgItem fvgsBull[IES_MAX_FVG]; int fvgBullN;
   SFvgItem fvgsBear[IES_MAX_FVG]; int fvgBearN;
   bool   priceInBullFvg, priceInBearFvg;
   SObItem obsBull[IES_MAX_OB]; int obBullN;
   SObItem obsBear[IES_MAX_OB]; int obBearN;
   double brkBullT, brkBullB, brkBearT, brkBearB;
   bool   breakerBull, breakerBear;
   bool   priceInBullOB, priceInBearOB;
   double fibHigh, fibLow, fibRetr, fibNear, fibExt1, fibExt2;
   bool   inGoldenZone;
   double marnieRetr, marnieNear, marnieExt1, marnieExt2;
   bool   marnieInGZ;
   double confluenceScore; int confluenceHits;
   double pivD, pivDR1, pivDR2, pivDR3, pivDS1, pivDS2, pivDS3;
   double pivW, pivWR1, pivWR2, pivWR3, pivWS1, pivWS2, pivWS3;
   double pdh, pdl, pdo, pdc;
   double asiaHi, asiaLo;
   double ibLonHi, ibLonLo, ibNyHi, ibNyLo;
   double ibWidthAtr;
   bool   inIB, brokeIBhigh, brokeIBlow, asiaSweepHi, asiaSweepLo;
   double linSlopeAtr, linR2;
   int    mtfStates[IES_MTF_N];
   int    mtfScore;
   int    regime, session;
   bool   isOverlap, isWeekend;
   bool   newsRisk;
   string newsEventName;
   double macroBias;
   double candleBody, candleUpWick, candleLoWick;
   double bodyRatio, candleATRNorm, clv, wickAtr;
   bool   patDoji, patPinBull, patPinBear, patEngulfBull, patEngulfBear;
   bool   patInside, patOutside;
   bool   displacement, rejection, breakoutBull, breakoutBear;
   bool   compression, expansion, microReclaimBull, microReclaimBear;
   int    bullStreak, bearStreak, todBucket;
   double scoreTrend, scoreMTF, scoreMomentum, scoreVolume;
   double scoreSMC, scoreGeometry, scoreCandle, scoreMacro, scoreAstro;
   double finalCompositeScore;
   ENUM_SIGNAL_TIER tier;
   string tierReason;
   SAstroContext astro;
  };

//==============================================================
// Inputs — v1.00 tuned
//==============================================================
input group "=== General ==="
input ENUM_TIMEFRAMES InpTimeframe         = PERIOD_M5;
input long            InpMagic             = 20260915;
input double          InpMinScore          = 18.0;
input double          InpAplusScore        = 55.0;
input double          InpATierScore        = 42.0;
input double          InpBTierScore        = 30.0;
input double          InpCTierScore        = 22.0;
input double          InpWatchScore        = 12.0;
input double          InpRR1               = 1.0;
input double          InpRR2               = 1.6;
input double          InpRR3               = 2.2;
input double          InpSLatrMult         = 1.0;
input bool            InpUseStructureSL    = true;
input double          InpMinSlAtr          = 0.8;
input double          InpMaxSlAtr          = 1.8;
input double          InpMaxSlUSD          = 8.0;
input bool            InpNewBarOnly        = false;
input uint            InpDeviationPts      = 100;
input bool            InpLogEveryBar       = true;
input bool            InpHeartbeat         = true;

input group "=== Score Diagnostics ==="
input bool            InpLogScoreParts     = true;
input bool            InpRelaxRange        = true;
input double          InpRelaxRangeScore   = 18.0;

input group "=== Partials & Trailing ==="
input bool            InpUsePartials       = true;
input double          InpPartial1Pct       = 50.0;
input double          InpPartial2Pct       = 30.0;
input bool            InpBEAfterTP1        = true;
input double          InpBEPlusCostPts     = 10.0;
input ENUM_TRAIL_MODE InpTrailMode         = TRAIL_ATR;
input double          InpTrailAtrMult      = 0.8;
input double          InpTrailStartR       = 1.0;
input bool            InpStepBrokerTP      = true;

input group "=== Risk & Protection (v1.00) ==="
input double          InpRiskPerTrade      = 0.5;
input bool            InpAllowMinLot       = true;
input double          InpDayLossPct        = 3.0;
input double          InpWeekLossPct       = 6.0;
input double          InpMaxDDPct          = 10.0;
input double          InpEquityFloor       = 0.0;
input double          InpProfitLockPct     = 10.0;   // v1.00: raised from 4
input int             InpMaxTradesDay      = 20;
input int             InpMaxConsecLosses   = 4;
input int             InpCooldownMin       = 20;
input double          InpRecoveryFactor    = 0.5;
input double          InpMaxSpreadAtrFrac  = 0.35;
input double          InpMaxSpreadUSD      = 1.20;
input double          InpMinMarginLvl      = 200.0;
input int             InpFridayCloseHour   = 20;
input bool            InpResumeOnInit      = true;   // v1.00: FULL state reset on restart

input group "=== Cost Model ==="
input double          InpCommissionPerLot  = -1.0;
input double          InpSlippageAllowPts  = 30.0;
input double          InpSwapPerLotPerDay  = 0.0;
input bool            InpAvoidNegativeSwap = true;

input group "=== Sessions ==="
input bool            InpUseSessionFilter  = false;
input string          InpSessions          = "SYDNEY,ASIAN,LONDON,OVERLAP,NEW_YORK";
input bool            InpUseTodFilter      = false;
input bool            InpAsianSweepFilter  = true;
input int             InpIBMinutes         = 60;
input int             InpBrokerGmtOffset   = 0;

input group "=== News ==="
input ENUM_NEWS_IMPACT_FILTER InpNewsImpact = NEWS_HIGH_ONLY;
input int             InpNewsWindowBefore  = 30;
input int             InpNewsWindowAfter   = 15;

input group "=== Drivers ==="
input bool            InpEnableDrivers     = true;
input string          InpDxySymbol         = "DXY";
input string          InpVixSymbol         = "VIX";
input string          InpBtcSymbol         = "BTCUSD";
input string          InpOilSymbol         = "USOIL";
input int             InpDriverMomBars     = 24;
input double          InpReal10y           = 0.0;
input double          InpFedCtx            = 0.0;
input double          InpCotNetChg         = 0.0;

input group "=== MTF ==="
input ENUM_TIMEFRAMES InpMtfTF0            = PERIOD_M5;
input ENUM_TIMEFRAMES InpMtfTF1            = PERIOD_M15;
input ENUM_TIMEFRAMES InpMtfTF2            = PERIOD_H1;
input ENUM_TIMEFRAMES InpMtfTF3            = PERIOD_H4;
input bool            InpStrictMtfFilter   = false;

input group "=== Astro ==="
input bool               InpEnableAstro      = true;
input ENUM_AYANAMSA_TYPE InpAstroAyanamsa    = AYANAMSA_LAHIRI;
input double             InpAstroWeight      = 25.0;
input bool               InpAstroSizing      = true;
input bool               InpFilterDemonHours = false;
input bool               InpFilterGandanta   = false;
input bool               InpShowAstroHUD     = true;

input group "=== Filters ==="
input bool            InpFilterAbsorption  = false;
input bool            InpFilterRangeNoise  = false;
input bool            InpDriverVeto        = false;

input group "=== UI ==="
input ENUM_UI_MODE    InpUiMode            = UI_GRAPHICAL_HUD;
input bool            InpDrawSMC           = true;
input bool            InpNotifyAlert       = true;
input bool            InpNotifyPush        = false;

//==============================================================
// Utilities
//==============================================================
double IES_Min(const double a, const double b) { return(a<b ? a : b); }
double IES_Max(const double a, const double b) { return(a>b ? a : b); }
int    IES_MinI(const int a, const int b)      { return(a<b ? a : b); }
int    IES_MaxI(const int a, const int b)      { return(a>b ? a : b); }
double IES_Clamp(const double v, const double lo, const double hi) { return(v<lo ? lo : (v>hi ? hi : v)); }
bool   IES_ValidNum(const double v) { return(MathIsValidNumber(v) && v != EMPTY_VALUE); }

double IES_ZScore(const double v, const double &arr[], const int n)
  {
   if(n < 3) return(0.0);
   double mean = 0, sq = 0;
   for(int i = 0; i < n; i++) mean += arr[i];
   mean /= n;
   for(int i = 0; i < n; i++) { double d = arr[i] - mean; sq += d*d; }
   double sd = MathSqrt(sq / (n - 1));
   if(sd < 1e-12) return(0.0);
   return((v - mean) / sd);
  }

bool IES_InRange(const int s, const int e, const int h) { return(s <= e ? (h >= s && h < e) : (h >= s || h < e)); }

bool IES_SymLooksLike(const string hay, const string needle)
  {
   string h = hay, n = needle;
   StringToUpper(h); StringToUpper(n);
   return(StringFind(h, n) >= 0);
  }

int IES_VolumeDigits(const string sym)
  {
   double step = SymbolInfoDouble(sym, SYMBOL_VOLUME_STEP);
   if(step <= 0) return(2);
   int d = 0;
   while(step < 0.999999 && d < 8) { step *= 10.0; d++; }
   return(d);
  }

double IES_NormLot(const string sym, double lots)
  {
   double minL = SymbolInfoDouble(sym, SYMBOL_VOLUME_MIN);
   double maxL = SymbolInfoDouble(sym, SYMBOL_VOLUME_MAX);
   double step = SymbolInfoDouble(sym, SYMBOL_VOLUME_STEP);
   if(minL <= 0) minL = 0.01;
   if(maxL <= 0) maxL = 100.0;
   if(step <= 0) step = 0.01;
   if(minL < step) minL = step;
   if(lots <= 0) return(0.0);
   double n = MathRound(lots / step) * step;
   int digs = IES_VolumeDigits(sym);
   n = NormalizeDouble(n, digs);
   if(n < minL) n = minL;
   if(n > maxL) n = maxL;
   return(n);
  }

ENUM_ORDER_TYPE_FILLING IES_Filling(const string sym)
  {
   long fm = (long)SymbolInfoInteger(sym, SYMBOL_FILLING_MODE);
   if((fm & SYMBOL_FILLING_IOC) == SYMBOL_FILLING_IOC) return(ORDER_FILLING_IOC);
   if((fm & SYMBOL_FILLING_FOK) == SYMBOL_FILLING_FOK) return(ORDER_FILLING_FOK);
   return(ORDER_FILLING_RETURN);
  }

string IES_RegimeToString(const int r)
  {
   switch(r)
     {
      case IES_REGIME_TRENDING_BULLISH: return("TRENDING_BULLISH");
      case IES_REGIME_TRENDING_BEARISH: return("TRENDING_BEARISH");
      case IES_REGIME_RANGE:            return("RANGE");
      case IES_REGIME_MEAN_REVERSION:   return("MEAN_REVERSION");
      case IES_REGIME_BREAKOUT:         return("BREAKOUT");
      case IES_REGIME_HIGH_VOLATILITY:  return("HIGH_VOLATILITY");
      case IES_REGIME_SQUEEZE:          return("SQUEEZE");
     }
   return("UNKNOWN");
  }

string IES_SessionToString(const int s)
  {
   switch(s)
     {
      case IES_SESSION_SYDNEY:   return("SYDNEY");
      case IES_SESSION_ASIAN:    return("ASIAN");
      case IES_SESSION_LONDON:   return("LONDON");
      case IES_SESSION_NEW_YORK: return("NEW_YORK");
      case IES_SESSION_OVERLAP:  return("OVERLAP");
     }
   return("OFF_HOURS");
  }

string IES_TierToString(const ENUM_SIGNAL_TIER t)
  {
   switch(t)
     {
      case TIER_APLUS: return("A+");
      case TIER_A:     return("A");
      case TIER_B:     return("B");
      case TIER_C:     return("C");
      case TIER_WATCH: return("WATCH");
     }
   return("REJECT");
  }

//==============================================================
// Forward declarations
//==============================================================
bool IES_ClosePosition(const ulong ticket);
bool IES_ClosePartial(const ulong ticket, const double volume);
bool IES_ModifyPosition(const ulong ticket, const double sl, const double tp);
bool IES_SendDeal(const ENUM_ORDER_TYPE ot, const string sym, const double lots,
                  const double price, const double sl, const double tp, const string comment);
double IES_GetInitialR(const ulong ticket, const double open, const double sl);

//==============================================================
// Broker info
//==============================================================
class CBrokerInfo
  {
public:
   string   sym;
   int      digits;
   double   point, tickSize, tickValue, contractSize;
   double   volMin, volMax, volStep;
   long     stopsLevel, freezeLevel, fillingMode;
   ENUM_SYMBOL_TRADE_MODE tradeMode;
   string   accountCurrency;
   double   accountLeverage;
   int      gmtOffsetHours;

   bool Init(const string s)
     {
      sym = s;
      if(!SymbolSelect(sym, true)) { Print("SymbolSelect failed for ", sym); return(false); }
      digits    = (int)SymbolInfoInteger(sym, SYMBOL_DIGITS);
      point     = SymbolInfoDouble(sym, SYMBOL_POINT);
      tickSize  = SymbolInfoDouble(sym, SYMBOL_TRADE_TICK_SIZE);
      tickValue = SymbolInfoDouble(sym, SYMBOL_TRADE_TICK_VALUE_LOSS);
      if(tickValue <= 0) tickValue = SymbolInfoDouble(sym, SYMBOL_TRADE_TICK_VALUE);
      if(tickValue <= 0) tickValue = SymbolInfoDouble(sym, SYMBOL_TRADE_TICK_VALUE_PROFIT);
      contractSize   = SymbolInfoDouble(sym, SYMBOL_TRADE_CONTRACT_SIZE);
      volMin         = SymbolInfoDouble(sym, SYMBOL_VOLUME_MIN);
      volMax         = SymbolInfoDouble(sym, SYMBOL_VOLUME_MAX);
      volStep        = SymbolInfoDouble(sym, SYMBOL_VOLUME_STEP);
      stopsLevel     = (long)SymbolInfoInteger(sym, SYMBOL_TRADE_STOPS_LEVEL);
      freezeLevel    = (long)SymbolInfoInteger(sym, SYMBOL_TRADE_FREEZE_LEVEL);
      fillingMode    = (long)SymbolInfoInteger(sym, SYMBOL_FILLING_MODE);
      tradeMode      = (ENUM_SYMBOL_TRADE_MODE)SymbolInfoInteger(sym, SYMBOL_TRADE_MODE);
      accountCurrency= AccountInfoString(ACCOUNT_CURRENCY);
      accountLeverage= (double)AccountInfoInteger(ACCOUNT_LEVERAGE);

      if(point <= 0) { Print("invalid point"); return(false); }
      if(tickSize <= 0) tickSize = point;
      if(volStep <= 0)  volStep = 0.01;
      if(volMin <= 0)   volMin = volStep;
      if(volMax <= 0)   volMax = 100.0;

      if(tickValue <= 0)
        {
         double ask = SymbolInfoDouble(sym, SYMBOL_ASK);
         double p = 0;
         if(ask > 0 && OrderCalcProfit(ORDER_TYPE_BUY, sym, 1.0, ask, ask + tickSize, p) && p > 0)
            tickValue = p;
        }
      if(tickValue <= 0 && contractSize > 0) tickValue = contractSize * tickSize;
      if(tickValue <= 0) { Print("tickValue unresolved"); return(false); }

      datetime srv = TimeTradeServer();
      datetime gmt = TimeGMT();
      gmtOffsetHours = (int)MathRound(((double)(srv - gmt)) / 3600.0);
      if(InpBrokerGmtOffset != 0) gmtOffsetHours = InpBrokerGmtOffset;
      return(true);
     }

   double MoneyPerPoint() const
     {
      if(tickSize <= 0) return(0.0);
      return(tickValue * (point / tickSize));
     }

   double MoneyRisk(const double slDistPrice, const double lots) const
     {
      double mpp = MoneyPerPoint();
      if(mpp <= 0 || lots <= 0) return(0.0);
      double pts = slDistPrice / point;
      return(pts * mpp * lots);
     }

   double CommissionPerLotRoundTurn()
     {
      if(InpCommissionPerLot > 0) return(InpCommissionPerLot);
      double total = 0; int cnt = 0;
      if(HistorySelect(TimeCurrent() - 30*24*60*60, TimeCurrent()))
        {
         int n = HistoryDealsTotal();
         for(int i = n-1; i >= MathMax(0, n-200); i--)
           {
            ulong t = HistoryDealGetTicket(i);
            if(t == 0) continue;
            if(HistoryDealGetString(t, DEAL_SYMBOL) != sym) continue;
            if((long)HistoryDealGetInteger(t, DEAL_MAGIC) != InpMagic) continue;
            double comm = HistoryDealGetDouble(t, DEAL_COMMISSION);
            double vol  = HistoryDealGetDouble(t, DEAL_VOLUME);
            if(vol > 0) { total += MathAbs(comm); cnt++; }
           }
        }
      if(cnt >= 5)
        {
         double avgPerSide = total / (double)cnt;
         return(avgPerSide * 2.0);
        }
      return(0.0);
     }

   double SwapPerLotPerNight() const
     {
      if(InpSwapPerLotPerDay != 0) return(InpSwapPerLotPerDay);
      double sl = SymbolInfoDouble(sym, SYMBOL_SWAP_LONG);
      double ss = SymbolInfoDouble(sym, SYMBOL_SWAP_SHORT);
      return(MathMax(MathAbs(sl), MathAbs(ss)));
     }

   bool IsSwapFree() const { return(SymbolInfoInteger(sym, SYMBOL_SWAP_MODE) == SYMBOL_SWAP_MODE_DISABLED); }
  };

//==============================================================
// Session engine
//==============================================================
class CSessionEngine
  {
private:
   int m_gmtOffsetHours;

   int ServerHourGmt()
     {
      MqlDateTime t; TimeToStruct(TimeTradeServer(), t);
      int h = t.hour - m_gmtOffsetHours;
      while(h < 0)   h += 24;
      while(h >= 24) h -= 24;
      return(h);
     }

public:
   void Init(const int brokerGmtOffset) { m_gmtOffsetHours = brokerGmtOffset; }

   int CurrentSession(bool &isOverlap, bool &isWeekend)
     {
      MqlDateTime g; TimeToStruct(TimeGMT(), g);
      isWeekend = (g.day_of_week == 6 || (g.day_of_week == 0 && g.hour < 21));
      int h = ServerHourGmt();
      bool inSyd = IES_InRange(21, 6, h);
      bool inTok = IES_InRange(0, 9, h);
      bool inLon = IES_InRange(7, 16, h);
      bool inNY  = IES_InRange(12, 21, h);
      isOverlap = (inLon && inNY);
      if(isOverlap) return(IES_SESSION_OVERLAP);
      if(inLon)     return(IES_SESSION_LONDON);
      if(inNY)      return(IES_SESSION_NEW_YORK);
      if(inTok)     return(IES_SESSION_ASIAN);
      if(inSyd)     return(IES_SESSION_SYDNEY);
      return(IES_SESSION_OFF_HOURS);
     }

   int TodBucket()
     {
      int h = ServerHourGmt();
      if((h >= 7 && h < 10) || (h >= 12 && h < 16)) return(1);
      if((h >= 2 && h < 4)  || (h >= 20 && h < 22)) return(-1);
      return(0);
     }
  };

//==============================================================
// Indicator engine
//==============================================================
class CIESIndicators
  {
private:
   string m_sym; ENUM_TIMEFRAMES m_tf; SIESParams m_p;
   int m_hEma[5], m_hSma[3];
   int m_hMacd, m_hAdx, m_hSar, m_hIchi, m_hRsi, m_hStoch, m_hCci, m_hAtr, m_hBands;
   int m_hMtfEmaFast[IES_MTF_N], m_hMtfEmaSlow[IES_MTF_N];
   int m_hMtfMacd[IES_MTF_N], m_hMtfRsi[IES_MTF_N];
   bool m_ok;

   bool Copy1(const int h, const int buf, const int shift, double &v)
     {
      if(h == INVALID_HANDLE) return(false);
      double t[1];
      if(CopyBuffer(h, buf, shift, 1, t) != 1) return(false);
      v = t[0];
      return(IES_ValidNum(v));
     }

public:
   void Init(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p)
     {
      m_sym = sym; m_tf = tf; m_p = p; m_ok = true;
      for(int i = 0; i < 5; i++)
        { m_hEma[i] = iMA(sym, tf, p.emaPeriods[i], 0, MODE_EMA, PRICE_CLOSE);
          if(m_hEma[i] == INVALID_HANDLE) m_ok = false; }
      for(int i = 0; i < 3; i++)
        { m_hSma[i] = iMA(sym, tf, p.smaPeriods[i], 0, MODE_SMA, PRICE_CLOSE);
          if(m_hSma[i] == INVALID_HANDLE) m_ok = false; }
      m_hMacd  = iMACD(sym, tf, p.macdFast, p.macdSlow, p.macdSignal, PRICE_CLOSE);
      m_hAdx   = iADX(sym, tf, p.adxPeriod);
      m_hSar   = iSAR(sym, tf, p.sarStep, p.sarMax);
      m_hIchi  = iIchimoku(sym, tf, p.ichiTenkan, p.ichiKijun, p.ichiSenkou);
      m_hRsi   = iRSI(sym, tf, p.rsiPeriod, PRICE_CLOSE);
      m_hStoch = iStochastic(sym, tf, p.stochK, p.stochD, p.stochSlow, MODE_SMA, STO_LOWHIGH);
      m_hCci   = iCCI(sym, tf, p.cciPeriod, PRICE_TYPICAL);
      m_hAtr   = iATR(sym, tf, p.atrPeriod);
      m_hBands = iBands(sym, tf, p.bbPeriod, 0, p.bbDev, PRICE_CLOSE);
      if(m_hMacd == INVALID_HANDLE || m_hAdx == INVALID_HANDLE || m_hSar == INVALID_HANDLE ||
         m_hIchi == INVALID_HANDLE || m_hRsi == INVALID_HANDLE || m_hStoch == INVALID_HANDLE ||
         m_hCci == INVALID_HANDLE  || m_hAtr == INVALID_HANDLE || m_hBands == INVALID_HANDLE)
         m_ok = false;
      for(int k = 0; k < IES_MTF_N; k++)
        {
         ENUM_TIMEFRAMES curTf = p.mtfTFs[k];
         m_hMtfEmaFast[k] = iMA(sym, curTf, 9,  0, MODE_EMA, PRICE_CLOSE);
         m_hMtfEmaSlow[k] = iMA(sym, curTf, 21, 0, MODE_EMA, PRICE_CLOSE);
         m_hMtfMacd[k]    = iMACD(sym, curTf, 12, 26, 9, PRICE_CLOSE);
         m_hMtfRsi[k]     = iRSI(sym, curTf, 14, PRICE_CLOSE);
        }
     }

   bool Ok() const { return(m_ok); }

   void Release()
     {
      for(int i = 0; i < 5; i++) if(m_hEma[i] != INVALID_HANDLE) IndicatorRelease(m_hEma[i]);
      for(int i = 0; i < 3; i++) if(m_hSma[i] != INVALID_HANDLE) IndicatorRelease(m_hSma[i]);
      if(m_hMacd  != INVALID_HANDLE) IndicatorRelease(m_hMacd);
      if(m_hAdx   != INVALID_HANDLE) IndicatorRelease(m_hAdx);
      if(m_hSar   != INVALID_HANDLE) IndicatorRelease(m_hSar);
      if(m_hIchi  != INVALID_HANDLE) IndicatorRelease(m_hIchi);
      if(m_hRsi   != INVALID_HANDLE) IndicatorRelease(m_hRsi);
      if(m_hStoch != INVALID_HANDLE) IndicatorRelease(m_hStoch);
      if(m_hCci   != INVALID_HANDLE) IndicatorRelease(m_hCci);
      if(m_hAtr   != INVALID_HANDLE) IndicatorRelease(m_hAtr);
      if(m_hBands != INVALID_HANDLE) IndicatorRelease(m_hBands);
      for(int k = 0; k < IES_MTF_N; k++)
        {
         if(m_hMtfEmaFast[k] != INVALID_HANDLE) IndicatorRelease(m_hMtfEmaFast[k]);
         if(m_hMtfEmaSlow[k] != INVALID_HANDLE) IndicatorRelease(m_hMtfEmaSlow[k]);
         if(m_hMtfMacd[k]    != INVALID_HANDLE) IndicatorRelease(m_hMtfMacd[k]);
         if(m_hMtfRsi[k]     != INVALID_HANDLE) IndicatorRelease(m_hMtfRsi[k]);
        }
     }

   bool Update(SFeatures &f)
     {
      if(!m_ok) return(false);
      double v = 0;

      if(!Copy1(m_hEma[0], 0, 1, v)) return(false); f.ema9   = v;
      if(!Copy1(m_hEma[1], 0, 1, v)) return(false); f.ema21  = v;
      if(!Copy1(m_hEma[2], 0, 1, v)) return(false); f.ema50  = v;
      if(!Copy1(m_hEma[3], 0, 1, v)) return(false); f.ema100 = v;
      if(!Copy1(m_hEma[4], 0, 1, v)) return(false); f.ema200 = v;

      f.emaStackBull = (f.ema9 > f.ema21 && f.ema21 > f.ema50 && f.ema50 > f.ema200);
      f.emaStackBear = (f.ema9 < f.ema21 && f.ema21 < f.ema50 && f.ema50 < f.ema200);

      double e9a, e9b, e21a, e21b;
      Copy1(m_hEma[0], 0, 2, e9b); Copy1(m_hEma[1], 0, 2, e21b);
      Copy1(m_hEma[0], 0, 1, e9a); Copy1(m_hEma[1], 0, 1, e21a);
      if(e9b <= e21b && e9a > e21a)      f.emaCross921 =  1;
      else if(e9b >= e21b && e9a < e21a) f.emaCross921 = -1;
      else                                f.emaCross921 =  0;

      Copy1(m_hSma[0], 0, 1, f.sma50);
      Copy1(m_hSma[1], 0, 1, f.sma100);
      Copy1(m_hSma[2], 0, 1, f.sma200);

      double mA, mB, sA, sB;
      if(!Copy1(m_hMacd, 0, 1, mA) || !Copy1(m_hMacd, 0, 2, mB) ||
         !Copy1(m_hMacd, 1, 1, sA) || !Copy1(m_hMacd, 1, 2, sB)) return(false);
      f.macdMain   = mA; f.macdSignal = sA; f.macdHist = mA - sA;
      f.macdBullCross = ((mB - sB) <= 0 && (mA - sA) > 0);
      f.macdBearCross = ((mB - sB) >= 0 && (mA - sA) < 0);

      Copy1(m_hAdx, 0, 1, f.adx);
      Copy1(m_hAdx, 1, 1, f.adxPlusDI);
      Copy1(m_hAdx, 2, 1, f.adxMinusDI);
      double adx5 = f.adx;
      Copy1(m_hAdx, 0, 5, adx5);
      f.adxSlope = f.adx - adx5;

      double sar;
      if(!Copy1(m_hSar, 0, 1, sar)) return(false);
      f.psar     = sar;
      f.psarLong = (iClose(m_sym, m_tf, 1) > sar);

      Copy1(m_hIchi, 0, 1, f.ichiTenkan);
      Copy1(m_hIchi, 1, 1, f.ichiKijun);
      Copy1(m_hIchi, 2, 1, f.ichiSenkouA);
      Copy1(m_hIchi, 3, 1, f.ichiSenkouB);
      f.ichiCloudTop = IES_Max(f.ichiSenkouA, f.ichiSenkouB);
      f.ichiCloudBot = IES_Min(f.ichiSenkouA, f.ichiSenkouB);
      double c1 = iClose(m_sym, m_tf, 1);
      f.ichiChikouBull = (c1 > iClose(m_sym, m_tf, 1 + m_p.ichiChikouShift));
      if(c1 > f.ichiCloudTop)      f.ichiCloudPos =  1;
      else if(c1 < f.ichiCloudBot) f.ichiCloudPos = -1;
      else                          f.ichiCloudPos =  0;

      Copy1(m_hRsi, 0, 1, f.rsi);
      double rsiArr[]; int n = IES_MinI(m_p.srsiLen + m_p.srsiSmoothK + m_p.srsiSmoothD + 5, 200);
      ArrayResize(rsiArr, n); ArraySetAsSeries(rsiArr, true);
      if(CopyBuffer(m_hRsi, 0, 1, n, rsiArr) == n)
        {
         int L = m_p.srsiLen;
         double lo = rsiArr[0], hi = rsiArr[0];
         for(int i = 0; i < L; i++) { lo = MathMin(lo, rsiArr[i]); hi = MathMax(hi, rsiArr[i]); }
         f.srsiRaw = ((hi - lo) > 1e-9 ? (rsiArr[0] - lo) / (hi - lo) : 0.5);
         double kArr[]; int nk = n - L + 1;
         ArrayResize(kArr, nk);
         for(int i = 0; i < nk; i++)
           {
            double l2 = rsiArr[i], h2 = rsiArr[i];
            for(int j = 0; j < L; j++) { l2 = MathMin(l2, rsiArr[i+j]); h2 = MathMax(h2, rsiArr[i+j]); }
            kArr[i] = ((h2 - l2) > 1e-9 ? (rsiArr[i] - l2) / (h2 - l2) : 0.5) * 100.0;
           }
         double k = 0; int K = m_p.srsiSmoothK;
         for(int i = 0; i < K && i < nk; i++) k += kArr[i];
         k /= IES_MaxI(K, 1); f.srsiK = k;
         double d = 0; int D = m_p.srsiSmoothD;
         for(int i = 0; i < D && i < nk; i++) d += kArr[i];
         d /= IES_MaxI(D, 1); f.srsiD = d;
        }

      Copy1(m_hStoch, 0, 1, f.stochK);
      Copy1(m_hStoch, 1, 1, f.stochD);
      Copy1(m_hCci,   0, 1, f.cci);

      if(!Copy1(m_hAtr, 0, 1, f.atr)) return(false);
      double atrArr[]; int zn = IES_MinI(m_p.zLookback, 200);
      ArrayResize(atrArr, zn); ArraySetAsSeries(atrArr, true);
      if(CopyBuffer(m_hAtr, 0, 1, zn, atrArr) == zn) f.atrZ = IES_ZScore(f.atr, atrArr, zn);

      Copy1(m_hBands, 0, 1, f.bbMiddle);
      Copy1(m_hBands, 1, 1, f.bbUpper);
      Copy1(m_hBands, 2, 1, f.bbLower);
      f.bbWidth = (f.bbMiddle > 0 ? (f.bbUpper - f.bbLower) / f.bbMiddle : 0);

      double upArr[], loArr[], midArr[];
      ArrayResize(upArr, zn); ArrayResize(loArr, zn); ArrayResize(midArr, zn);
      ArraySetAsSeries(upArr, true); ArraySetAsSeries(loArr, true); ArraySetAsSeries(midArr, true);
      if(CopyBuffer(m_hBands, 1, 1, zn, upArr) == zn &&
         CopyBuffer(m_hBands, 2, 1, zn, loArr) == zn &&
         CopyBuffer(m_hBands, 0, 1, zn, midArr) == zn)
        {
         double wArr[]; ArrayResize(wArr, zn);
         for(int i = 0; i < zn; i++) wArr[i] = (midArr[i] > 0 ? (upArr[i] - loArr[i]) / midArr[i] : 0);
         f.bbWidthZ = IES_ZScore(f.bbWidth, wArr, zn);
         double prevC = iClose(m_sym, m_tf, 2);
         f.bbBullRev = (prevC < loArr[1] && c1 > f.bbLower);
         f.bbBearRev = (prevC > upArr[1] && c1 < f.bbUpper);
        }

      double kel = 1.5 * f.atr;
      f.squeezeOn = (f.bbUpper > 0 && f.bbUpper < (f.ema21 + kel) && f.bbLower > (f.ema21 - kel));
      double atr2 = f.atr, bbU2 = f.bbUpper, bbL2 = f.bbLower, e212 = f.ema21;
      Copy1(m_hAtr, 0, 2, atr2);
      Copy1(m_hBands, 1, 2, bbU2);
      Copy1(m_hBands, 2, 2, bbL2);
      Copy1(m_hEma[1], 0, 2, e212);
      bool sqPrev = (bbU2 < (e212 + 1.5 * atr2) && bbL2 > (e212 - 1.5 * atr2));
      f.squeezeRelease = (sqPrev && !f.squeezeOn);
      return(true);
     }

   void UpdateMTF(SFeatures &f)
     {
      double wsum = 0, acc = 0;
      for(int k = 0; k < IES_MTF_N; k++)
        {
         double eF=0, eS=0, mMain=0, mSig=0, rsiVal=50;
         bool okF = Copy1(m_hMtfEmaFast[k], 0, 1, eF);
         bool okS = Copy1(m_hMtfEmaSlow[k], 0, 1, eS);
         bool okM1 = Copy1(m_hMtfMacd[k], 0, 1, mMain);
         bool okM2 = Copy1(m_hMtfMacd[k], 1, 1, mSig);
         bool okR  = Copy1(m_hMtfRsi[k],   0, 1, rsiVal);
         int st = 0;
         if(okF && okS && okM1 && okM2 && okR)
           {
            double mHist = mMain - mSig;
            bool emaB = (eF > eS), emaS = (eF < eS);
            bool macB = (mHist > 0), macS = (mHist < 0);
            if(emaB && macB && rsiVal > 52.0)      st =  2;
            else if(emaS && macS && rsiVal < 48.0) st = -2;
            else if(emaB && macB)                  st =  1;
            else if(emaS && macS)                  st = -1;
           }
         f.mtfStates[k] = st;
         acc  += st * m_p.mtfW[k];
         wsum += 2.0 * m_p.mtfW[k];
        }
      f.mtfScore = (wsum > 0 ? (int)MathRound(4.0 * acc / wsum) : 0);
     }
  };

//==============================================================
// Astro helpers
//==============================================================
double IES_AstroJulianDay(const datetime t)
  {
   MqlDateTime dt; TimeToStruct(t, dt);
   int y = dt.year, m = dt.mon, d = dt.day;
   if(m <= 2) { y -= 1; m += 12; }
   int a = y / 100;
   int b = 2 - a + (a / 4);
   double dayFrac = d + (dt.hour + dt.min/60.0 + dt.sec/3600.0) / 24.0;
   return(MathFloor(365.25*(y+4716)) + MathFloor(30.6001*(m+1)) + dayFrac + b - 1524.5);
  }

double IES_AstroAyanamsa(const double jd, const ENUM_AYANAMSA_TYPE ayaType)
  {
   double t = (jd - 2451545.0) / 36525.0;
   double lahiri = 23.85307 + 1.39697 * t + 0.000308 * t * t;
   switch(ayaType)
     {
      case AYANAMSA_RAMAN:       return(lahiri - 1.41);
      case AYANAMSA_KP:          return(lahiri - 0.04);
      case AYANAMSA_TRUE_CHITRA: return(lahiri + 0.02);
     }
   return(lahiri);
  }

string IES_GetRashiName(const int r)
  {
   static const string names[12] = {"Aries","Taurus","Gemini","Cancer","Leo","Virgo",
                                    "Libra","Scorpio","Sagittarius","Capricorn","Aquarius","Pisces"};
   return(names[r % 12]);
  }
string IES_GetNakshatraName(const int n)
  {
   static const string names[27] = {"Ashwini","Bharani","Krittika","Rohini","Mrigashira","Ardra",
      "Punarvasu","Pushya","Ashlesha","Magha","Purva Phalguni","Uttara Phalguni",
      "Hasta","Chitra","Swati","Vishakha","Anuradha","Jyeshtha",
      "Mula","Purva Ashadha","Uttara Ashadha","Shravana","Dhanishta","Shatabhisha",
      "Purva Bhadrapada","Uttara Bhadrapada","Revati"};
   return(names[n % 27]);
  }
string IES_GetNakshatraLord(const int n)
  {
   static const string lords[9] = {"Ketu","Venus","Sun","Moon","Mars","Rahu","Jupiter","Saturn","Mercury"};
   return(lords[n % 9]);
  }
string IES_Get108PadaLord(const int p)
  {
   static const string pL[36] =
     {"Mars","Venus","Mercury","Moon","Sun","Mercury","Venus","Mars","Jupiter",
      "Saturn","Saturn","Jupiter","Mars","Venus","Mercury","Moon","Sun","Mercury",
      "Venus","Mars","Jupiter","Saturn","Saturn","Jupiter","Mars","Venus","Mercury",
      "Moon","Sun","Mercury","Venus","Mars","Jupiter","Saturn","Saturn","Jupiter"};
   int rashi = (p % 108) / 9;
   int padaInRashi = (p % 108) % 9;
   int element = rashi % 4;
   return(pL[element * 9 + padaInRashi]);
  }
string IES_GetTithiName(const int t)
  {
   static const string names[15] = {"Pratipada","Dwitiya","Tritiya","Chaturthi","Panchami",
      "Shashthi","Saptami","Ashtami","Navami","Dashami","Ekadashi","Dvadashi","Trayodashi","Chaturdashi","Purnima"};
   if(t == 30) return("Amavasya (New Moon)");
   if(t == 15) return("Purnima (Full Moon)");
   return(names[(t-1) % 15]);
  }
string IES_GetTaraBalaName(const int i)
  {
   static const string names[9] = {"Janma (Volatile)","Sampat (Wealth +)","Vipat (Loss -)",
      "Kshema (Well-Being +)","Pratyak (Obstacle -)","Sadhana (Success +)",
      "Naidhana (Danger --)","Mitra (Friendly +)","Parama Mitra (Supreme +)"};
   return(names[i % 9]);
  }
string IES_GetChaldeanPlanetName(const int i)
  {
   static const string names[7] = {"Saturn","Jupiter","Mars","Sun","Venus","Mercury","Moon"};
   return(names[i % 7]);
  }

//==============================================================
// Astro engine
//==============================================================
class CIES_AstroEngine
  {
private:
   string             m_sym;
   ENUM_AYANAMSA_TYPE m_ayaType;
   SAstroContext      m_ctx;
   datetime           m_lastUpdate;

   double Norm360(const double d) { double r = MathMod(d, 360.0); return(r >= 0 ? r : r + 360.0); }
   double Rad(const double d) { return(d * M_PI / 180.0); }
   double Deg(const double r) { return(r * 180.0 / M_PI); }

   double SolveKepler(const double M_deg, const double e)
     {
      double m = Rad(M_deg), E = m;
      for(int i = 0; i < 15; i++)
        {
         double dE = (m - (E - e*MathSin(E))) / (1 - e*MathCos(E));
         E += dE;
         if(MathAbs(dE) < 1e-7) break;
        }
      return(E);
     }

   void GetHelioXYZ(const string body, const double cy, double &x, double &y, double &z)
     {
      double a0=0,ar=0,e0=0,er=0,i0=0,ir=0,l0=0,lr=0,w0=0,wr=0,o0=0,orr=0;
      if(body == "Earth")   { a0=1.00000261;ar=0.00000562;e0=0.01671123;er=-0.00004392;i0=0.00001531;ir=-0.01294668;l0=100.46457166;lr=35999.37244981;w0=102.93768193;wr=0.32327364;o0=0;orr=0; }
      else if(body == "Mercury") { a0=0.38709927;ar=0.00000037;e0=0.20563593;er=0.00001906;i0=7.00497902;ir=-0.00594749;l0=252.25032350;lr=149472.67411175;w0=77.45779628;wr=0.16047689;o0=48.33076593;orr=-0.12534081; }
      else if(body == "Venus") { a0=0.72333566;ar=0.00000390;e0=0.00677672;er=-0.00004107;i0=3.39467605;ir=-0.00078890;l0=181.97909950;lr=58517.81538729;w0=131.60246718;wr=0.00268329;o0=76.67984255;orr=-0.27769418; }
      else if(body == "Mars") { a0=1.52371034;ar=0.00001847;e0=0.09339410;er=0.00007882;i0=1.84969142;ir=-0.00813131;l0=-4.55343205;lr=19140.30268499;w0=-23.94362959;wr=0.44441088;o0=49.55953891;orr=-0.29257343; }
      else if(body == "Jupiter") { a0=5.20288700;ar=-0.00011607;e0=0.04838624;er=-0.00013253;i0=1.30439695;ir=-0.00183714;l0=34.39644051;lr=3034.74612775;w0=14.72847983;wr=0.21252668;o0=100.47390909;orr=0.20469106; }
      else if(body == "Saturn") { a0=9.53667594;ar=-0.00125060;e0=0.05386179;er=-0.00050991;i0=2.48599187;ir=0.00193609;l0=49.95424423;lr=1222.49362201;w0=92.59887831;wr=-0.41897216;o0=113.66242448;orr=-0.28867794; }
      else if(body == "Uranus") { a0=19.18916464;ar=-0.00196176;e0=0.04725744;er=-0.00004397;i0=0.77263783;ir=-0.00242939;l0=313.23810451;lr=428.48202785;w0=170.95427630;wr=0.40805281;o0=74.01692503;orr=0.04240589; }
      else if(body == "Neptune") { a0=30.06992276;ar=0.00026291;e0=0.00860610;er=0.00005105;i0=1.77004347;ir=0.00035372;l0=-55.12002969;lr=218.45945325;w0=44.96476227;wr=-0.32241464;o0=131.78422574;orr=-0.00508664; }
      else if(body == "Pluto") { a0=39.48211675;ar=-0.00031596;e0=0.24882730;er=0.00005170;i0=17.14001206;ir=0.00004818;l0=238.92903833;lr=145.20780515;w0=224.06879900;wr=-0.04062942;o0=110.30393608;orr=-0.01183482; }
      double a = a0 + ar*cy, e = e0 + er*cy, I = i0 + ir*cy;
      double L = l0 + lr*cy, w = w0 + wr*cy, O = o0 + orr*cy;
      double M = Norm360(L - w);
      double E = SolveKepler(M, e);
      double xp = a*(MathCos(E) - e);
      double yp = a*MathSqrt(1 - e*e)*MathSin(E);
      double omega = Rad(w - O), node = Rad(O), inc = Rad(I);
      x = (MathCos(omega)*MathCos(node) - MathSin(omega)*MathSin(node)*MathCos(inc))*xp
        + (-MathSin(omega)*MathCos(node) - MathCos(omega)*MathSin(node)*MathCos(inc))*yp;
      y = (MathCos(omega)*MathSin(node) + MathSin(omega)*MathCos(node)*MathCos(inc))*xp
        + (-MathSin(omega)*MathSin(node) + MathCos(omega)*MathCos(node)*MathCos(inc))*yp;
      z = (MathSin(omega)*MathSin(inc))*xp + (MathCos(omega)*MathSin(inc))*yp;
     }

   void SetupBody(const ENUM_ASTRO_BODY idx, const string name, const double tropLon, const double tropLat, const double speed)
     {
      m_ctx.bodies[idx].name = name;
      m_ctx.bodies[idx].tropLon = Norm360(tropLon);
      m_ctx.bodies[idx].tropLat = tropLat;
      m_ctx.bodies[idx].sidLon = Norm360(tropLon - m_ctx.ayanamsa);
      m_ctx.bodies[idx].dailySpeed = speed;
      m_ctx.bodies[idx].isRetro = (speed < 0);
      double sid = m_ctx.bodies[idx].sidLon;
      m_ctx.bodies[idx].rashi = (int)(sid / 30.0) % 12;
      double degIn = sid - m_ctx.bodies[idx].rashi * 30.0;
      m_ctx.bodies[idx].deg = (int)degIn;
      double mF = (degIn - m_ctx.bodies[idx].deg) * 60.0;
      m_ctx.bodies[idx].min = (int)mF;
      m_ctx.bodies[idx].sec = (int)((mF - m_ctx.bodies[idx].min) * 60.0);
      double nakSpan = 360.0 / 27.0;
      m_ctx.bodies[idx].nakshatra = (int)(sid / nakSpan) % 27;
      double rem = sid - m_ctx.bodies[idx].nakshatra * nakSpan;
      m_ctx.bodies[idx].pada = (int)(rem / (nakSpan / 4.0)) + 1;
      m_ctx.bodies[idx].totalPada = (int)(sid / (nakSpan / 4.0)) % 108;

      m_ctx.bodies[idx].isCombust = false;
      if(idx != ASTRO_SUN && idx != ASTRO_RAHU && idx != ASTRO_KETU)
        {
         double sun = m_ctx.bodies[ASTRO_SUN].tropLon;
         double d = MathAbs(m_ctx.bodies[idx].tropLon - sun);
         if(d > 180) d = 360 - d;
         double orb = 12.0;
         if(idx == ASTRO_MERCURY) orb = 14.0;
         if(idx == ASTRO_VENUS)   orb = 10.0;
         if(idx == ASTRO_MARS)    orb = 17.0;
         if(idx == ASTRO_JUPITER) orb = 11.0;
         if(idx == ASTRO_SATURN)  orb = 15.0;
         if(d <= orb) m_ctx.bodies[idx].isCombust = true;
        }
      int r = m_ctx.bodies[idx].rashi;
      string dig = "Neutral";
      if(idx == ASTRO_SUN) { if(r == 0) dig = "Exalted (Ucha)"; else if(r == 6) dig = "Debilitated (Neecha)"; else if(r == 4) dig = "Own Sign (Swakshetra)"; }
      else if(idx == ASTRO_MOON) { if(r == 1) dig = "Exalted (Ucha)"; else if(r == 7) dig = "Debilitated (Neecha)"; else if(r == 3) dig = "Own Sign (Swakshetra)"; }
      else if(idx == ASTRO_MARS) { if(r == 9) dig = "Exalted (Ucha)"; else if(r == 3) dig = "Debilitated (Neecha)"; else if(r == 0 || r == 7) dig = "Own Sign (Swakshetra)"; }
      else if(idx == ASTRO_MERCURY) { if(r == 5) dig = "Exalted (Ucha)"; else if(r == 11) dig = "Debilitated (Neecha)"; else if(r == 2) dig = "Own Sign (Swakshetra)"; }
      else if(idx == ASTRO_JUPITER) { if(r == 3) dig = "Exalted (Ucha)"; else if(r == 9) dig = "Debilitated (Neecha)"; else if(r == 8 || r == 11) dig = "Own Sign (Swakshetra)"; }
      else if(idx == ASTRO_VENUS) { if(r == 11) dig = "Exalted (Ucha)"; else if(r == 5) dig = "Debilitated (Neecha)"; else if(r == 1 || r == 6) dig = "Own Sign (Swakshetra)"; }
      else if(idx == ASTRO_SATURN) { if(r == 6) dig = "Exalted (Ucha)"; else if(r == 0) dig = "Debilitated (Neecha)"; else if(r == 9 || r == 10) dig = "Own Sign (Swakshetra)"; }
      else if(idx == ASTRO_RAHU) { if(r == 1 || r == 2) dig = "Exalted"; else if(r == 7 || r == 8) dig = "Debilitated"; }
      else if(idx == ASTRO_KETU) { if(r == 7 || r == 8) dig = "Exalted"; else if(r == 1 || r == 2) dig = "Debilitated"; }
      m_ctx.bodies[idx].dignity = dig;

      double sb = 50.0;
      if(StringFind(dig, "Exalted") >= 0) sb += 35;
      else if(StringFind(dig, "Own") >= 0) sb += 25;
      else if(StringFind(dig, "Debilitated") >= 0) sb -= 30;
      if(m_ctx.bodies[idx].isRetro && idx != ASTRO_RAHU && idx != ASTRO_KETU) sb += 12;
      if(m_ctx.bodies[idx].isCombust) sb -= 25;
      m_ctx.bodies[idx].shadbala = MathMax(5, MathMin(100, sb));
     }

   void CalcEphemeris(const double jd)
     {
      double cy = (jd - 2451545.0) / 36525.0;
      double L0 = 280.46646 + 36000.76983*cy + 0.0003032*cy*cy;
      double M_s = 357.52911 + 35999.05029*cy - 0.0001537*cy*cy;
      double C = (1.914602 - 0.004817*cy)*MathSin(Rad(M_s))
               + (0.019993 - 0.000101*cy)*MathSin(Rad(2*M_s))
               + 0.000289*MathSin(Rad(3*M_s));
      double sunTrop = Norm360(L0 + C);
      SetupBody(ASTRO_SUN, "Sun", sunTrop, 0, 0.9856);

      double xe, ye, ze; GetHelioXYZ("Earth", cy, xe, ye, ze);

      double L_m = 218.3164477 + 481267.88128*cy;
      double D_m = 297.8501921 + 445267.11140*cy;
      double M_m = 134.9633964 + 477198.86750*cy;
      double F_m = 93.2720950  + 483202.01752*cy;
      double moonLon = L_m + 6.289*MathSin(Rad(M_m)) - 1.274*MathSin(Rad(M_m - 2*D_m))
                     + 0.658*MathSin(Rad(2*D_m)) - 0.214*MathSin(Rad(2*M_m))
                     - 0.186*MathSin(Rad(M_s)) - 0.114*MathSin(Rad(2*F_m))
                     + 0.059*MathSin(Rad(2*D_m - M_m)) - 0.057*MathSin(Rad(M_m + M_s - 2*D_m))
                     + 0.053*MathSin(Rad(2*D_m + M_m)) + 0.046*MathSin(Rad(2*D_m - M_s))
                     + 0.041*MathSin(Rad(M_m - M_s)) - 0.035*MathSin(Rad(D_m))
                     - 0.031*MathSin(Rad(M_m + M_s));
      double moonLat = 5.128*MathSin(Rad(F_m)) + 0.280*MathSin(Rad(M_m + F_m));
      SetupBody(ASTRO_MOON, "Moon", moonLon, moonLat, 13.176);

      double rahuTrop = Norm360(125.04452 - 1934.136261*cy + 0.0020708*cy*cy);
      double ketuTrop = Norm360(rahuTrop + 180);
      SetupBody(ASTRO_RAHU, "Rahu", rahuTrop, 0, -0.0529);
      SetupBody(ASTRO_KETU, "Ketu", ketuTrop, 0, -0.0529);

      string pNames[8] = {"Mercury","Venus","Mars","Jupiter","Saturn","Uranus","Neptune","Pluto"};
      ENUM_ASTRO_BODY pEnums[8] = {ASTRO_MERCURY,ASTRO_VENUS,ASTRO_MARS,ASTRO_JUPITER,
                                    ASTRO_SATURN,ASTRO_URANUS,ASTRO_NEPTUNE,ASTRO_PLUTO};
      double cyN = ((jd + 0.25) - 2451545.0) / 36525.0;
      double xe2, ye2, ze2; GetHelioXYZ("Earth", cyN, xe2, ye2, ze2);
      for(int i = 0; i < 8; i++)
        {
         double xp, yp, zp; GetHelioXYZ(pNames[i], cy, xp, yp, zp);
         double X = xp - xe, Y = yp - ye, Z = zp - ze;
         double lon = Norm360(Deg(MathArctan2(Y, X)));
         double lat = Deg(MathArctan2(Z, MathSqrt(X*X + Y*Y)));
         double xp2, yp2, zp2; GetHelioXYZ(pNames[i], cyN, xp2, yp2, zp2);
         double X2 = xp2 - xe2, Y2 = yp2 - ye2;
         double lon2 = Norm360(Deg(MathArctan2(Y2, X2)));
         double dLon = lon2 - lon;
         if(dLon > 180) dLon -= 360; if(dLon < -180) dLon += 360;
         double speed = dLon * 4.0;
         SetupBody(pEnums[i], pNames[i], lon, lat, speed);
        }
     }

   void CalcHora(const datetime t)
     {
      MqlDateTime dt; TimeToStruct(t, dt);
      static const int dayRulersChaldean[7] = {3,6,2,5,1,4,0};
      int startRuler = dayRulersChaldean[dt.day_of_week % 7];
      m_ctx.horaPlanet = (startRuler + dt.hour) % 7;
      m_ctx.horaName   = IES_GetChaldeanPlanetName(m_ctx.horaPlanet);
      m_ctx.horaMinRemaining = 60 - dt.min;
     }

   void CalcDemonHours(const datetime t)
     {
      MqlDateTime dt; TimeToStruct(t, dt);
      int total = dt.hour*60 + dt.min;
      static const int rahuSeg[7] = {7,1,6,4,5,3,2};
      static const int yamaSeg[7] = {4,3,2,1,0,6,5};
      static const int guliSeg[7] = {6,5,4,3,2,1,0};
      m_ctx.isRahuKalam = false; m_ctx.isYamaganda = false; m_ctx.isGulika = false;
      m_ctx.demonStatus = "Clear (No Demon Hour)";
      if(total >= 360 && total < 1080)
        {
         int seg = (total - 360) / 90;
         int dow = dt.day_of_week % 7;
         if(seg == rahuSeg[dow])      { m_ctx.isRahuKalam = true; m_ctx.demonStatus = "RAHU KALAM (Toxic Shadow)"; }
         else if(seg == yamaSeg[dow]) { m_ctx.isYamaganda = true; m_ctx.demonStatus = "YAMAGANDA (Loss Period)"; }
         else if(seg == guliSeg[dow]) { m_ctx.isGulika    = true; m_ctx.demonStatus = "GULIKA KALAM (Mandi Poison)"; }
        }
     }

   void CalcLunarPhases()
     {
      double sunSid = m_ctx.bodies[ASTRO_SUN].sidLon;
      double moonSid = m_ctx.bodies[ASTRO_MOON].sidLon;
      double diff = Norm360(moonSid - sunSid);
      m_ctx.phaseAngle = diff;
      m_ctx.tithi = (int)(diff / 12.0) + 1;
      if(m_ctx.tithi < 1) m_ctx.tithi = 1;
      if(m_ctx.tithi > 30) m_ctx.tithi = 30;
      m_ctx.tithiName = IES_GetTithiName(m_ctx.tithi);
      m_ctx.paksha = (m_ctx.tithi <= 15 ? "Shukla (Waxing)" : "Krishna (Waning)");
      if(diff < 22.5 || diff >= 337.5)      m_ctx.lunarPhase = "New Moon (Amavasya)";
      else if(diff < 67.5)   m_ctx.lunarPhase = "Waxing Crescent";
      else if(diff < 112.5)  m_ctx.lunarPhase = "First Quarter";
      else if(diff < 157.5)  m_ctx.lunarPhase = "Waxing Gibbous";
      else if(diff < 202.5)  m_ctx.lunarPhase = "Full Moon (Purnima)";
      else if(diff < 247.5)  m_ctx.lunarPhase = "Waning Gibbous";
      else if(diff < 292.5)  m_ctx.lunarPhase = "Last Quarter";
      else                   m_ctx.lunarPhase = "Waning Crescent";
      m_ctx.isRiktaTithi = (m_ctx.tithi == 4 || m_ctx.tithi == 9 || m_ctx.tithi == 14 ||
                            m_ctx.tithi == 19 || m_ctx.tithi == 24 || m_ctx.tithi == 29);
     }

   void CalcDashas()
     {
      double moon = m_ctx.bodies[ASTRO_MOON].sidLon;
      double span = 360.0 / 27.0;
      int nak = (int)(moon / span) % 27;
      m_ctx.moonNakshatra = nak;
      m_ctx.moonNakName = IES_GetNakshatraName(nak);
      m_ctx.moonNakLord = IES_GetNakshatraLord(nak);
      m_ctx.moonPada = (int)(MathMod(moon, span) / (span / 4.0)) + 1;
      m_ctx.moonTotalPada = (int)(moon / (span / 4.0)) % 108;
      static const int vimYears[9] = {7,20,6,10,7,18,16,19,17};
      int lordIdx = nak % 9;
      m_ctx.vimshottariLord = IES_GetNakshatraLord(nak);
      double remSpan = span - MathMod(moon, span);
      m_ctx.vimshottariBalance = (remSpan / span) * vimYears[lordIdx];
      m_ctx.vimshottariSubLord = IES_GetNakshatraLord((lordIdx + 1) % 9);
      static const string shodLords[8] = {"Sun","Mars","Jupiter","Saturn","Ketu","Moon","Mercury","Venus"};
      m_ctx.shodshottariLord = shodLords[(nak >= 7 ? nak - 7 : nak + 20) % 8];
      static const string ashtLords[8] = {"Sun","Moon","Mars","Mercury","Saturn","Jupiter","Rahu","Venus"};
      m_ctx.ashtottariAridradiLord = ashtLords[(nak >= 5 ? nak - 5 : nak + 22) % 8];
      m_ctx.ashtottariKartikadiLord = ashtLords[(nak >= 2 ? nak - 2 : nak + 25) % 8];
      static const string shashtiLords[8] = {"Jupiter","Sun","Mars","Moon","Mercury","Venus","Saturn","Rahu"};
      m_ctx.shashtiHayaniLord = shashtiLords[nak % 8];
     }

   void CalcFearAndApocalypse()
     {
      double moon = m_ctx.bodies[ASTRO_MOON].sidLon;
      double sat  = m_ctx.bodies[ASTRO_SATURN].sidLon;
      double rahu = m_ctx.bodies[ASTRO_RAHU].sidLon;
      double mars = m_ctx.bodies[ASTRO_MARS].sidLon;
      double dMS = MathAbs(moon - sat); if(dMS > 180) dMS = 360 - dMS;
      m_ctx.hasVishYoga = (dMS <= 7 || MathAbs(dMS - 180) <= 7);
      double dMR = MathAbs(moon - rahu); if(dMR > 180) dMR = 360 - dMR;
      m_ctx.hasGrahanYoga = (dMR <= 8);
      double dMarsSat = MathAbs(mars - sat); if(dMarsSat > 180) dMarsSat = 360 - dMarsSat;
      m_ctx.hasYamaYoga = (dMarsSat <= 6 || MathAbs(dMarsSat - 90) <= 6);
      double dMarsRahu = MathAbs(mars - rahu); if(dMarsRahu > 180) dMarsRahu = 360 - dMarsRahu;
      m_ctx.hasAngarakYoga = (dMarsRahu <= 8);
      double f = 20.0;
      if(m_ctx.hasVishYoga)    f += 25;
      if(m_ctx.hasYamaYoga)    f += 25;
      if(m_ctx.hasGrahanYoga)  f += 15;
      if(m_ctx.hasAngarakYoga) f += 15;
      if(m_ctx.isRahuKalam)    f += 15;
      m_ctx.fearIndex = MathMin(100, f);
      m_ctx.isGandanta = ((moon >= 357 || moon <= 3) ||
                          (moon >= 117 && moon <= 123) ||
                          (moon >= 237 && moon <= 243));
      m_ctx.isApocalypseTrigger = false;
      if((MathAbs(m_ctx.bodies[ASTRO_MARS].dailySpeed) < 0.08 ||
          MathAbs(m_ctx.bodies[ASTRO_SATURN].dailySpeed) < 0.03) &&
         (dMarsRahu <= 4 || dMS <= 4))
         m_ctx.isApocalypseTrigger = true;
      m_ctx.isObsessionGap = (m_ctx.horaName == "Mars" && dMarsRahu <= 12);
     }

   void CalcSpecialYogas()
     {
      double moon = m_ctx.bodies[ASTRO_MOON].sidLon;
      double jup  = m_ctx.bodies[ASTRO_JUPITER].sidLon;
      double sun  = m_ctx.bodies[ASTRO_SUN].sidLon;
      double merc = m_ctx.bodies[ASTRO_MERCURY].sidLon;
      double mars = m_ctx.bodies[ASTRO_MARS].sidLon;
      double dMJ = Norm360(jup - moon);
      m_ctx.hasGajakesari = (dMJ <= 8 || dMJ >= 352 ||
                             MathAbs(dMJ - 90) <= 8 || MathAbs(dMJ - 180) <= 8 || MathAbs(dMJ - 270) <= 8);
      double dSM = MathAbs(sun - merc); if(dSM > 180) dSM = 360 - dSM;
      m_ctx.hasBudhaditya = (dSM <= 10 && !m_ctx.bodies[ASTRO_MERCURY].isCombust);
      double dMM = MathAbs(moon - mars); if(dMM > 180) dMM = 360 - dMM;
      m_ctx.hasChandraMangala = (dMM <= 8 || MathAbs(dMM - 180) <= 8);
      double dJM = MathAbs(jup - mars); if(dJM > 180) dJM = 360 - dJM;
      m_ctx.hasGuruMangala = (dJM <= 6 || MathAbs(dJM - 120) <= 6);
      m_ctx.hasDhanaYoga = (m_ctx.bodies[ASTRO_JUPITER].shadbala >= 70 ||
                            m_ctx.bodies[ASTRO_VENUS].shadbala >= 70);
     }

   void CalcSaturnTransit()
     {
      m_ctx.saturnRashi = IES_GetRashiName(m_ctx.bodies[ASTRO_SATURN].rashi);
      m_ctx.saturnRetro = m_ctx.bodies[ASTRO_SATURN].isRetro;
      m_ctx.saturnAspectAlert = StringFormat("Saturn in %s (%s) | Speed %+.2f deg/d",
                                             m_ctx.saturnRashi,
                                             (m_ctx.saturnRetro ? "RETROGRADE" : "DIRECT"),
                                             m_ctx.bodies[ASTRO_SATURN].dailySpeed);
     }

   void CalcTaraBala()
     {
      int dist = (m_ctx.moonNakshatra + 1) % 9;
      m_ctx.taraBala = IES_GetTaraBalaName(dist);
      if(dist == 1 || dist == 3 || dist == 5 || dist == 7 || dist == 8) m_ctx.taraBalaScore = 1;
      else if(dist == 2 || dist == 4 || dist == 6) m_ctx.taraBalaScore = -1;
      else m_ctx.taraBalaScore = 0;
     }

   void CalcMicroTradeMap(const string sym)
     {
      m_ctx.microMap.stateM1  = (iClose(sym,PERIOD_M1,1)  > iOpen(sym,PERIOD_M1,1)  ? 1 : -1);
      m_ctx.microMap.stateM5  = (iClose(sym,PERIOD_M5,1)  > iOpen(sym,PERIOD_M5,1)  ? 1 : -1);
      m_ctx.microMap.stateM15 = (iClose(sym,PERIOD_M15,1) > iOpen(sym,PERIOD_M15,1) ? 1 : -1);
      m_ctx.microMap.stateM30 = (iClose(sym,PERIOD_M30,1) > iOpen(sym,PERIOD_M30,1) ? 1 : -1);
      m_ctx.microMap.stateH1  = (iClose(sym,PERIOD_H1,1)  > iOpen(sym,PERIOD_H1,1)  ? 1 : -1);
      m_ctx.microMap.stateH4  = (iClose(sym,PERIOD_H4,1)  > iOpen(sym,PERIOD_H4,1)  ? 1 : -1);
      m_ctx.microMap.stateD1  = (iClose(sym,PERIOD_D1,1)  > iOpen(sym,PERIOD_D1,1)  ? 1 : -1);
      m_ctx.microMap.pada108Entry = m_ctx.moonTotalPada;
      m_ctx.microMap.padaLord     = IES_Get108PadaLord(m_ctx.moonTotalPada);
      double b = 0;
      b += m_ctx.microMap.stateM1*10 + m_ctx.microMap.stateM5*20 + m_ctx.microMap.stateM15*15
         + m_ctx.microMap.stateM30*15 + m_ctx.microMap.stateH1*15 + m_ctx.microMap.stateH4*15
         + m_ctx.microMap.stateD1*10;
      if(m_ctx.horaName == "Sun" || m_ctx.horaName == "Jupiter" || m_ctx.horaName == "Mars") b += 10;
      else if(m_ctx.horaName == "Saturn") b -= 10;
      if(m_ctx.hasGajakesari)  b += 15;
      if(m_ctx.hasGuruMangala) b += 12;
      if(m_ctx.hasVishYoga)    b -= 18;
      if(m_ctx.hasYamaYoga)    b -= 20;
      m_ctx.microMap.compositeBias = MathMax(-100, MathMin(100, b));
      if(m_ctx.microMap.compositeBias >= 50)       m_ctx.microMap.rating = "STRONG BULLISH";
      else if(m_ctx.microMap.compositeBias >= 15)  m_ctx.microMap.rating = "BULLISH";
      else if(m_ctx.microMap.compositeBias <= -50) m_ctx.microMap.rating = "STRONG BEARISH";
      else if(m_ctx.microMap.compositeBias <= -15) m_ctx.microMap.rating = "BEARISH";
      else                                          m_ctx.microMap.rating = "NEUTRAL";
     }

public:
   void Init(const string sym, const ENUM_AYANAMSA_TYPE ayaType)
     {
      m_sym = sym; m_ayaType = ayaType; m_lastUpdate = 0;
      ZeroMemory(m_ctx);
      Update(TimeCurrent());
     }

   void Update(const datetime t)
     {
      if(t - m_lastUpdate < 30 && m_lastUpdate != 0) return;
      m_lastUpdate = t;
      double jd = IES_AstroJulianDay(t);
      m_ctx.ayanamsa = IES_AstroAyanamsa(jd, m_ayaType);
      CalcEphemeris(jd);
      CalcHora(t);
      CalcDemonHours(t);
      CalcLunarPhases();
      CalcDashas();
      CalcFearAndApocalypse();
      CalcSpecialYogas();
      CalcSaturnTransit();
      CalcTaraBala();
      CalcMicroTradeMap(m_sym);
     }

   double GetCosmicScore(const string sym, const int dir)
     {
      double s = 0.0;
      bool gold = (StringFind(sym, "XAU") >= 0 || StringFind(sym, "GOLD") >= 0);
      double karakaSb = (gold ? m_ctx.bodies[ASTRO_SUN].shadbala
                              : m_ctx.bodies[ASTRO_MERCURY].shadbala);
      s += (karakaSb - 50.0) * 0.6;
      if(m_ctx.hasGajakesari)     s += 15.0;
      if(m_ctx.hasGuruMangala)    s += 12.0;
      if(m_ctx.hasBudhaditya)     s +=  8.0;
      if(m_ctx.hasChandraMangala) s += 10.0;
      if(m_ctx.hasDhanaYoga)      s += 10.0;
      if(m_ctx.hasVishYoga)       s -= 20.0;
      if(m_ctx.hasYamaYoga)       s -= 25.0;
      if(m_ctx.hasGrahanYoga)     s -= 15.0;
      if(m_ctx.isGandanta)        s -= 30.0;
      if(m_ctx.isRahuKalam)       s -= 15.0;
      s += m_ctx.taraBalaScore * 10.0;
      s += (double)dir * m_ctx.microMap.compositeBias * 0.10;
      return(MathMax(-100.0, MathMin(100.0, s)));
     }

   double GetSizingMultiplier(const string sym)
     {
      bool gold = (StringFind(sym, "XAU") >= 0 || StringFind(sym, "GOLD") >= 0);
      double karakaSb = (gold ? m_ctx.bodies[ASTRO_SUN].shadbala
                              : m_ctx.bodies[ASTRO_MERCURY].shadbala);
      double mult = 0.8 + (karakaSb / 100.0) * 0.5;
      if(m_ctx.isRahuKalam || m_ctx.isYamaganda) mult *= 0.65;
      return(MathMax(0.5, MathMin(1.5, mult)));
     }

   bool IsDemonHourActive() const { return(m_ctx.isRahuKalam || m_ctx.isYamaganda || m_ctx.isGulika); }
   bool IsGandantaActive()  const { return(m_ctx.isGandanta); }
   SAstroContext GetContext() const { return(m_ctx); }
  };

//==============================================================
// News filter
//==============================================================
class CIESNewsFilter
  {
private:
   string m_sym;
   ENUM_NEWS_IMPACT_FILTER m_mode;
   int m_before, m_after;
   bool m_risk;
   string m_event;

public:
   void Init(const string sym, const ENUM_NEWS_IMPACT_FILTER mode, const int b, const int a)
     { m_sym = sym; m_mode = mode; m_before = b; m_after = a; m_risk = false; m_event = ""; }

   void Update(SFeatures &f)
     {
      f.newsRisk = false;
      f.newsEventName = "";
      if(m_mode == NEWS_DISABLED) { m_risk = false; m_event = ""; return; }

      string base = SymbolInfoString(m_sym, SYMBOL_CURRENCY_BASE);
      string prof = SymbolInfoString(m_sym, SYMBOL_CURRENCY_PROFIT);
      datetime from = TimeTradeServer() - m_after * 60;
      datetime to   = TimeTradeServer() + m_before * 60;

      string curs[3]; curs[0] = base; curs[1] = prof; curs[2] = "USD";
      for(int c = 0; c < 3; c++)
        {
         if(curs[c] == "") continue;
         MqlCalendarValue vals[];
         ResetLastError();
         int n = CalendarValueHistory(vals, from, to, NULL, curs[c]);
         if(n <= 0) continue;
         for(int i = 0; i < n; i++)
           {
            MqlCalendarEvent ev;
            if(!CalendarEventById(vals[i].event_id, ev)) continue;
            bool hi = (ev.importance == CALENDAR_IMPORTANCE_HIGH);
            bool me = (ev.importance == CALENDAR_IMPORTANCE_MODERATE);
            if(hi || (m_mode == NEWS_HIGH_AND_MEDIUM && me))
              {
               f.newsRisk = true;
               f.newsEventName = ev.name;
               m_risk = true; m_event = ev.name;
               return;
              }
           }
        }
      m_risk = false; m_event = "";
     }

   bool   IsRisk() const { return(m_risk); }
   string RiskEvent() const { return(m_event); }
  };

//==============================================================
// Drivers
//==============================================================
class CIESDrivers
  {
private:
   string m_sym, m_dxy, m_vix, m_btc, m_oil;
   int    m_mom;
   double m_real10y, m_fed, m_cot;
   double m_dxyMom, m_vixMom, m_btcMom, m_oilMom, m_corrDxy;
   bool   m_haveDxy, m_haveVix, m_haveBtc, m_haveOil, m_enabled;

   string Resolve(const string req)
     {
      if(req == "") return("");
      if(SymbolSelect(req, true)) return(req);
      string tryv[10]; int n = 0;
      if(IES_SymLooksLike(req, "DXY"))  { tryv[0]="DXY"; tryv[1]="USDX"; tryv[2]="USDIDX"; n=3; }
      else if(IES_SymLooksLike(req, "VIX")) { tryv[0]="VIX"; tryv[1]="VOLX"; n=2; }
      else if(IES_SymLooksLike(req, "BTC")) { tryv[0]="BTCUSD"; tryv[1]="BTCUSDT"; n=2; }
      else if(IES_SymLooksLike(req, "OIL") || IES_SymLooksLike(req, "WTI"))
        { tryv[0]="USOIL"; tryv[1]="WTI"; tryv[2]="XTIUSD"; n=3; }
      for(int i = 0; i < n; i++) if(SymbolSelect(tryv[i], true)) return(tryv[i]);
      return(req);
     }

   double MomPct(const string s, const int bars)
     {
      if(s == "") return(0.0);
      if((int)iBars(s, PERIOD_H1) < bars + 2) return(0.0);
      double now = iClose(s, PERIOD_H1, 1);
      double was = iClose(s, PERIOD_H1, 1 + bars);
      if(was <= 0) return(0.0);
      return((now - was) / was * 100.0);
     }

   double Corr(const string a, const int bars)
     {
      if(a == "") return(0.0);
      if((int)iBars(a, PERIOD_H1) < bars + 2 || (int)iBars(m_sym, PERIOD_H1) < bars + 2) return(0.0);
      double sa=0, sb=0, saa=0, sbb=0, sab=0;
      for(int i = 1; i <= bars; i++)
        {
         double ra = iClose(a, PERIOD_H1, i) - iClose(a, PERIOD_H1, i+1);
         double rb = iClose(m_sym, PERIOD_H1, i) - iClose(m_sym, PERIOD_H1, i+1);
         sa += ra; sb += rb; saa += ra*ra; sbb += rb*rb; sab += ra*rb;
        }
      double cov = bars*sab - sa*sb;
      double va = bars*saa - sa*sa;
      double vb = bars*sbb - sb*sb;
      if(va <= 0 || vb <= 0) return(0.0);
      return(cov / MathSqrt(va * vb));
     }

public:
   void Init(const string sym, const bool en, const string dxy, const string vix,
             const string btc, const string oil, const int mom, const double r10,
             const double fed, const double cot, const string url)
     {
      m_sym = sym; m_enabled = en;
      m_dxy = Resolve(dxy); m_vix = Resolve(vix);
      m_btc = Resolve(btc); m_oil = Resolve(oil);
      m_mom = mom; m_real10y = r10; m_fed = fed; m_cot = cot;
      Update();
     }

   void Update()
     {
      m_haveDxy = m_haveVix = m_haveBtc = m_haveOil = false;
      if(!m_enabled) return;
      if(m_dxy != "" && SymbolSelect(m_dxy, true) && iBars(m_dxy, PERIOD_H1) > 10)
        { m_dxyMom = MomPct(m_dxy, m_mom); m_corrDxy = Corr(m_dxy, 60); m_haveDxy = true; }
      if(m_vix != "" && SymbolSelect(m_vix, true) && iBars(m_vix, PERIOD_H1) > 10)
        { m_vixMom = MomPct(m_vix, m_mom); m_haveVix = true; }
      if(m_btc != "" && SymbolSelect(m_btc, true) && iBars(m_btc, PERIOD_H1) > 10)
        { m_btcMom = MomPct(m_btc, m_mom); m_haveBtc = true; }
      if(m_oil != "" && SymbolSelect(m_oil, true) && iBars(m_oil, PERIOD_H1) > 10)
        { m_oilMom = MomPct(m_oil, m_mom); m_haveOil = true; }
     }

   void FetchRemote() { /* external JSON feed not required */ }

   bool Live() const { return(m_haveDxy || m_haveVix || m_haveBtc || m_haveOil); }
   bool ManualOverlay() const { return(m_real10y != 0 || m_fed != 0 || m_cot != 0); }

   double Bias()
     {
      if(!m_enabled) return(0.0);
      double b = 0.0;
      if(m_haveDxy) b += -MathTanh(m_dxyMom / 2.0) * 0.6;
      if(m_real10y != 0.0) b += -MathTanh(m_real10y / 4.0) * 0.5;
      if(m_fed != 0.0)     b +=  MathTanh(m_fed) * 0.35;
      if(m_haveVix)        b +=  MathTanh(m_vixMom / 10.0) * 0.40;
      if(m_cot != 0.0)     b +=  MathTanh(m_cot / 10000.0) * 0.25;
      if(m_haveBtc)        b +=  MathTanh(m_btcMom / 10.0) * 0.20;
      if(m_haveOil)        b +=  MathTanh(m_oilMom / 8.0) * 0.20;
      return(IES_Clamp(b, -1.0, 1.0));
     }

   string Describe()
     {
      if(!m_enabled) return("Drivers: Disabled");
      return(StringFormat("DXY %.2f%% corr %.2f | VIX %.2f%% | BTC %.2f%% | Oil %.2f%% | Bias %.2f",
                          m_dxyMom, m_corrDxy, m_vixMom, m_btcMom, m_oilMom, Bias()));
     }
  };

//==============================================================
// Capital protection — v1.00 with full state reset on Resume()
//==============================================================
class CIESCapitalProtection
  {
private:
   string m_sym, m_prefix, m_haltReason;
   long   m_magic;
   double m_dayLossPct, m_weekLossPct, m_maxDDPct, m_equityFloor;
   double m_riskPct, m_recoveryFactor, m_profitLockPct, m_minMarginLvl;
   int    m_maxTradesDay, m_maxConsecLosses, m_cooldownMin;
   double m_dayStartEq, m_weekStartEq, m_peakEq, m_dayPnL, m_weekPnL;
   int    m_tradesToday, m_consecLosses;
   datetime m_blockedUntil;
   bool   m_halted;

   double G(const string k, const double d)
     {
      string key = m_prefix + k;
      if(GlobalVariableCheck(key)) return(GlobalVariableGet(key));
      GlobalVariableSet(key, d); return(d);
     }
   void GS(const string k, const double v) { GlobalVariableSet(m_prefix + k, v); }

   double DayKey(const datetime t)  { MqlDateTime d; TimeToStruct(t,d); return(d.year*10000 + d.mon*100 + d.day); }
   double WeekKey(const datetime t) { MqlDateTime d; TimeToStruct(t,d); return(d.year*100 + d.day_of_year/7); }

public:
   void Init(const string sym, const long magic,
             const double dayLossPct, const double weekLossPct, const double maxDDPct,
             const double equityFloor, const double riskPct, const int maxTradesDay,
             const int maxConsecLosses, const int cooldownMin, const double minMarginLvl,
             const double profitLockPct, const double recoveryFactor)
     {
      m_sym = sym; m_magic = magic;
      m_prefix = "IES_" + sym + "_" + IntegerToString(magic) + "_";
      m_dayLossPct = dayLossPct; m_weekLossPct = weekLossPct; m_maxDDPct = maxDDPct;
      m_equityFloor = equityFloor; m_riskPct = riskPct; m_recoveryFactor = recoveryFactor;
      m_maxTradesDay = maxTradesDay; m_maxConsecLosses = maxConsecLosses;
      m_cooldownMin = cooldownMin; m_minMarginLvl = minMarginLvl;
      m_profitLockPct = profitLockPct; m_haltReason = "";

      datetime now = TimeTradeServer();
      if(G("dayKey", -1) != DayKey(now))
        { GS("dayKey", DayKey(now)); GS("dayStartEq", AccountInfoDouble(ACCOUNT_EQUITY));
          GS("tradesToday", 0); GS("dayPnL", 0); }
      if(G("weekKey", -1) != WeekKey(now))
        { GS("weekKey", WeekKey(now)); GS("weekStartEq", AccountInfoDouble(ACCOUNT_EQUITY)); GS("weekPnL", 0); }

      m_dayStartEq  = G("dayStartEq", AccountInfoDouble(ACCOUNT_EQUITY));
      m_weekStartEq = G("weekStartEq", AccountInfoDouble(ACCOUNT_EQUITY));
      m_peakEq      = G("peakEq", AccountInfoDouble(ACCOUNT_EQUITY));
      m_tradesToday = (int)G("tradesToday", 0);
      m_consecLosses= (int)G("consecLosses", 0);
      m_dayPnL      = G("dayPnL", 0);
      m_weekPnL     = G("weekPnL", 0);
      m_halted      = (G("halted", 0) > 0.5);
      m_blockedUntil= (datetime)G("blockedUntil", 0);
     }

   void CloseAllPositions()
     {
      for(int i = PositionsTotal()-1; i >= 0; i--)
        {
         ulong t = PositionGetTicket(i);
         if(t == 0 || !PositionSelectByTicket(t)) continue;
         if(PositionGetString(POSITION_SYMBOL) != m_sym) continue;
         if((long)PositionGetInteger(POSITION_MAGIC) != m_magic) continue;
         IES_ClosePosition(t);
        }
     }

   void Update()
     {
      double eq = AccountInfoDouble(ACCOUNT_EQUITY);
      if(eq > m_peakEq) { m_peakEq = eq; GS("peakEq", eq); }

      datetime now = TimeTradeServer();
      if(G("dayKey", -1) != DayKey(now))
        {
         m_dayStartEq = eq; m_dayPnL = 0; m_tradesToday = 0; m_consecLosses = 0;
         GS("dayKey", DayKey(now)); GS("dayStartEq", eq); GS("dayPnL", 0);
         GS("tradesToday", 0); GS("consecLosses", 0); GS("halted", 0);
         m_halted = false; m_haltReason = "";
        }
      if(G("weekKey", -1) != WeekKey(now))
        { m_weekStartEq = eq; m_weekPnL = 0;
          GS("weekKey", WeekKey(now)); GS("weekStartEq", eq); GS("weekPnL", 0); }

      if(m_equityFloor > 0 && eq <= m_equityFloor) { Halt("Equity floor"); return; }
      if(m_peakEq > 0 && eq < m_peakEq * (1.0 - m_maxDDPct/100.0)) { Halt("Peak DD"); return; }
      if(m_dayStartEq > 0 && eq <= m_dayStartEq * (1.0 - m_dayLossPct/100.0)) { Halt("Daily loss"); return; }
      if(m_weekStartEq > 0 && eq <= m_weekStartEq * (1.0 - m_weekLossPct/100.0)) { Halt("Weekly loss"); return; }
     }

   void Halt(const string reason)
     {
      m_halted = true; m_haltReason = reason;
      GS("halted", 1.0);
      CloseAllPositions();
      Print("IES PROTECTION: ", reason);
     }

   // v1.00: FULL state reset — clears every persistent field and GlobalVariable
   void Resume()
     {
      double eqNow = AccountInfoDouble(ACCOUNT_EQUITY);

      m_halted       = false;
      m_haltReason   = "";
      m_blockedUntil = 0;
      m_consecLosses = 0;
      m_tradesToday  = 0;
      m_dayPnL       = 0.0;
      m_weekPnL      = 0.0;
      m_dayStartEq   = eqNow;
      m_weekStartEq  = eqNow;
      m_peakEq       = eqNow;

      GS("halted", 0.0);
      GS("blockedUntil", 0.0);
      GS("consecLosses", 0.0);
      GS("tradesToday", 0.0);
      GS("dayPnL", 0.0);
      GS("weekPnL", 0.0);
      GS("dayStartEq", eqNow);
      GS("weekStartEq", eqNow);
      GS("peakEq", eqNow);

      PrintFormat("IES Resume: state cleared. Baseline equity = %.2f", eqNow);
     }

   bool   Halted() const { return(m_halted); }
   string HaltReason() const { return(m_haltReason); }
   double DayPnL() const { return(m_dayPnL); }
   int    TradesToday() const { return(m_tradesToday); }
   int    ConsecLosses() const { return(m_consecLosses); }
   double PeakEq() const { return(m_peakEq); }

   bool CanOpen(string &reason, const double spreadPts, const double ask,
                const double bid, const double lots, const ENUM_ORDER_TYPE ot)
     {
      reason = "";
      if(m_halted) { reason = "Kill switch: " + m_haltReason; return(false); }
      if(TimeTradeServer() < m_blockedUntil)
        {
         int mins = (int)((m_blockedUntil - TimeTradeServer()) / 60);
         reason = StringFormat("Cooldown (%dm left)", mins);
         return(false);
        }

      double eq = AccountInfoDouble(ACCOUNT_EQUITY);
      if(m_equityFloor > 0 && eq <= m_equityFloor)
        { reason = StringFormat("Equity floor (%.2f)", m_equityFloor); return(false); }
      if(m_peakEq > 0 && eq < m_peakEq*(1.0 - m_maxDDPct/100.0))
        {
         PrintFormat("IES DD check | peakEq=%.2f eq=%.2f dropPct=%.2f threshold=%.1f%%",
                     m_peakEq, eq, (1.0 - eq/m_peakEq)*100.0, m_maxDDPct);
         reason = "Peak DD"; return(false);
        }
      if(m_dayStartEq > 0 && eq <= m_dayStartEq*(1.0 - m_dayLossPct/100.0))
        {
         PrintFormat("IES DayLoss check | dayStartEq=%.2f eq=%.2f dropPct=%.2f threshold=%.1f%%",
                     m_dayStartEq, eq, (1.0 - eq/m_dayStartEq)*100.0, m_dayLossPct);
         reason = "Day loss limit"; return(false);
        }
      if(m_weekStartEq > 0 && eq <= m_weekStartEq*(1.0 - m_weekLossPct/100.0))
        { reason = "Week loss limit"; return(false); }
      if(m_profitLockPct > 0 && m_dayStartEq > 0 &&
         eq >= m_dayStartEq*(1.0 + m_profitLockPct/100.0))
        {
         PrintFormat("IES ProfitLock check | dayStartEq=%.2f eq=%.2f gainPct=%.2f threshold=%.1f%%",
                     m_dayStartEq, eq, (eq/m_dayStartEq - 1.0)*100.0, m_profitLockPct);
         reason = "Profit locked"; return(false);
        }
      if(m_tradesToday >= m_maxTradesDay)
        { reason = StringFormat("Max trades/day (%d)", m_maxTradesDay); return(false); }

      double margin = 0;
      if(!OrderCalcMargin(ot, m_sym, lots, (ot == ORDER_TYPE_BUY ? ask : bid), margin))
        { reason = "Margin calc failed"; return(false); }
      if(margin > AccountInfoDouble(ACCOUNT_MARGIN_FREE) * 0.85) { reason = "Insufficient free margin"; return(false); }

      double ml = AccountInfoDouble(ACCOUNT_MARGIN_LEVEL);
      if(ml > 0 && ml < m_minMarginLvl)
        { reason = StringFormat("Margin level %.1f < min %.1f", ml, m_minMarginLvl); return(false); }
      return(true);
     }

   double LotSize(const double slPoints, const CBrokerInfo &br, const bool allowMinLot)
     {
      if(slPoints <= 0) return(allowMinLot ? IES_NormLot(m_sym, br.volMin) : 0.0);

      double eq = AccountInfoDouble(ACCOUNT_EQUITY);
      if(eq <= 0) eq = AccountInfoDouble(ACCOUNT_BALANCE);
      if(eq <= 0) eq = 1000.0;
      double risk = (m_riskPct / 100.0) * eq;
      if(m_consecLosses >= 2) risk *= m_recoveryFactor;
      if(risk <= 0) risk = (m_riskPct/100.0) * 1000.0;

      double mpp = br.MoneyPerPoint();
      if(mpp <= 0) mpp = 1.0;
      double moneyPerLot = mpp * slPoints;
      if(moneyPerLot <= 0) moneyPerLot = slPoints;

      double lots = risk / moneyPerLot;
      if(!MathIsValidNumber(lots) || lots <= 0)
         lots = (allowMinLot ? br.volMin : 0.0);

      if(lots > 0 && lots < br.volMin)
        { if(allowMinLot) lots = br.volMin; else return(0.0); }

      lots = IES_NormLot(m_sym, lots);
      if(lots <= 0 && allowMinLot) lots = br.volMin;
      return(lots);
     }

   void RegisterOpen() { m_tradesToday++; GS("tradesToday", m_tradesToday); }

   void RegisterClose(const double pnl)
     {
      m_dayPnL += pnl;  GS("dayPnL", m_dayPnL);
      m_weekPnL += pnl; GS("weekPnL", m_weekPnL);
      if(pnl < 0) m_consecLosses++; else m_consecLosses = 0;
      GS("consecLosses", m_consecLosses);
      if(m_consecLosses >= m_maxConsecLosses)
        {
         m_blockedUntil = TimeTradeServer() + m_cooldownMin * 60;
         GS("blockedUntil", (double)m_blockedUntil);
         m_consecLosses = 0; GS("consecLosses", 0);
         Print("IES COOLDOWN: ", m_cooldownMin, " min after ", m_maxConsecLosses, " losses.");
        }
     }
  };

//==============================================================
// Global instances
//==============================================================
SIESParams            g_p;
SFeatures             g_f;
CIESIndicators        g_ind;
CIESCapitalProtection g_cp;
CIESDrivers           g_drv;
CIESNewsFilter        g_news;
CIES_AstroEngine      g_astro;
CBrokerInfo           g_br;
CSessionEngine        g_sess;
string                g_sym;
datetime              g_lastBar = 0;
string                g_block   = "Init...";
double                g_score   = 0.0;
bool                  g_ready   = false;
int                   g_warmFails = 0;
double                g_commissionPerLot = 0.0;

//==============================================================
// Parameter defaults
//==============================================================
void IES_DefaultParams(SIESParams &p)
  {
   ZeroMemory(p);
   p.emaPeriods[0]=9;  p.emaPeriods[1]=21;  p.emaPeriods[2]=50;
   p.emaPeriods[3]=100;p.emaPeriods[4]=200;
   p.smaPeriods[0]=50; p.smaPeriods[1]=100; p.smaPeriods[2]=200;
   p.macdFast=12; p.macdSlow=26; p.macdSignal=9;
   p.adxPeriod=14;
   p.sarStep=0.02; p.sarMax=0.2;
   p.ichiTenkan=9; p.ichiKijun=26; p.ichiSenkou=52; p.ichiChikouShift=26;
   p.rsiPeriod=14;
   p.stochK=14; p.stochD=3; p.stochSlow=3;
   p.srsiLen=14; p.srsiSmoothK=3; p.srsiSmoothD=3;
   p.cciPeriod=20; p.atrPeriod=14; p.bbPeriod=20; p.bbDev=2.0;
   p.zLookback=200; p.vwapRollLen=50;
   p.vwapBandSigma1=1.0; p.vwapBandSigma2=2.0; p.vwapBandSigma3=3.0;
   p.swingLR=3; p.structScanBars=200;
   p.sweepMinATR=0.10; p.fvgMinATR=0.25;
   p.obDispATR=1.10; p.obScanBars=60;
   p.eqTolATR=0.25;
   p.fibLevels[0]=0.236; p.fibLevels[1]=0.382; p.fibLevels[2]=0.500;
   p.fibLevels[3]=0.618; p.fibLevels[4]=0.786; p.fibLevels[5]=1.000;
   p.gzLo=0.618; p.gzHi=0.786;
   p.confTolATR=0.35;
   p.regAdxTrend=25.0; p.regAdxRange=20.0;
   p.regBBWidthZ=1.5;  p.regAtrZ=2.5;
   p.chopTrend=38.2;   p.chopRange=61.8;
   p.mtfTFs[0]=InpMtfTF0; p.mtfTFs[1]=InpMtfTF1;
   p.mtfTFs[2]=InpMtfTF2; p.mtfTFs[3]=InpMtfTF3;
   p.mtfW[0]=0.5; p.mtfW[1]=1.0; p.mtfW[2]=2.0; p.mtfW[3]=2.5;
   p.newsWindowBeforeMin=InpNewsWindowBefore;
   p.newsWindowAfterMin=InpNewsWindowAfter;
   p.dojiBodyMax=0.10; p.pinWickMin=0.60; p.dispBodyATR=1.20;
   p.regCompRatio=0.70; p.streakScan=20;
   p.ibMinutes=InpIBMinutes; p.linLen=20; p.chopLen=14;
  }

//==============================================================
// Candle intelligence
//==============================================================
void IES_CandleIntel(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   double o1 = iOpen(sym,tf,1), h1 = iHigh(sym,tf,1), l1 = iLow(sym,tf,1), c1 = iClose(sym,tf,1);
   double o2 = iOpen(sym,tf,2), h2 = iHigh(sym,tf,2), l2 = iLow(sym,tf,2), c2 = iClose(sym,tf,2);
   double range = h1 - l1; if(range <= 0) range = _Point;

   f.candleBody   = MathAbs(c1 - o1);
   f.candleUpWick = h1 - MathMax(o1, c1);
   f.candleLoWick = MathMin(o1, c1) - l1;
   f.bodyRatio    = f.candleBody / range;
   f.candleATRNorm= (f.atr > 0 ? range / f.atr : 0);
   f.clv          = (c1 - l1) / range;
   f.wickAtr      = (f.atr > 0 ? IES_Max(f.candleUpWick, f.candleLoWick) / f.atr : 0);

   bool bull = (c1 > o1), bear = (c1 < o1);
   f.patDoji       = (f.bodyRatio <= p.dojiBodyMax);
   f.patPinBull    = (!bear && f.candleLoWick >= p.pinWickMin * range && f.candleLoWick >= 2.0 * f.candleBody);
   f.patPinBear    = (!bull && f.candleUpWick >= p.pinWickMin * range && f.candleUpWick >= 2.0 * f.candleBody);
   f.patEngulfBull = (c2 < o2 && c1 > o1 && c1 >= o2 && o1 <= c2);
   f.patEngulfBear = (c2 > o2 && c1 < o1 && c1 <= o2 && o1 >= c2);
   f.patInside     = (h1 < h2 && l1 > l2);
   f.patOutside    = (h1 > h2 && l1 < l2);
   f.rejection     = (f.patPinBull || f.patPinBear || f.patDoji);
   f.displacement  = (f.atr > 0 && f.candleBody >= p.dispBodyATR * f.atr && f.bodyRatio >= 0.60);
   f.absorption    = (f.tickVolZ > 1.8 && f.bodyRatio < 0.35 && range > 0 &&
                      (f.candleUpWick >= 0.35 * range || f.candleLoWick >= 0.35 * range));

   double h3 = iHigh(sym,tf,3), l3 = iLow(sym,tf,3);
   f.microReclaimBull = (l1 < l3 && c1 > h2 && c1 > o1);
   f.microReclaimBear = (h1 > h3 && c1 < l2 && c1 < o1);

   double avg = 0;
   for(int i = 1; i <= 20; i++) avg += (iHigh(sym,tf,i) - iLow(sym,tf,i));
   avg /= 20.0;
   f.compression = (avg > 0 && range < p.regCompRatio * avg);
   f.expansion   = (avg > 0 && range > 1.30 * avg);

   double hi20 = 0, lo20 = DBL_MAX;
   for(int i = 2; i <= 21; i++) { hi20 = IES_Max(hi20, iHigh(sym,tf,i)); lo20 = IES_Min(lo20, iLow(sym,tf,i)); }
   f.breakoutBull = (c1 > hi20 && f.bodyRatio >= 0.50);
   f.breakoutBear = (c1 < lo20 && f.bodyRatio >= 0.50);

   f.bullStreak = 0; f.bearStreak = 0;
   for(int i = 1; i <= p.streakScan; i++)
     {
      double cc = iClose(sym,tf,i), oo = iOpen(sym,tf,i);
      if(cc > oo && f.bullStreak == i - 1)      f.bullStreak = i;
      else if(cc < oo && f.bearStreak == i - 1) f.bearStreak = i;
      else break;
     }
  }

//==============================================================
// Volume / CVD / VWAP
//==============================================================
void IES_VolumeFeatures(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   int bars = IES_MinI(p.zLookback, (int)iBars(sym,tf) - 2);
   if(bars < 20) { f.obv = 0; f.cvd = 0; return; }

   double obvArr[], volArr[], cvdArr[];
   ArrayResize(obvArr, bars); ArrayResize(volArr, bars); ArrayResize(cvdArr, bars);
   double obv = 0, cvd = 0;
   double prevC = iClose(sym,tf,bars);
   for(int i = bars; i >= 1; i--)
     {
      double c = iClose(sym,tf,i), o = iOpen(sym,tf,i);
      double h = iHigh(sym,tf,i),  l = iLow(sym,tf,i);
      double v = (double)iVolume(sym,tf,i);
      if(c > prevC)      obv += v;
      else if(c < prevC) obv -= v;
      double rng = h - l;
      double clvBar = (rng > 1e-12 ? ((c - l) - (h - c)) / rng : 0);
      cvd += v * clvBar;
      prevC = c;
      obvArr[bars-i] = obv; volArr[bars-i] = v; cvdArr[bars-i] = cvd;
     }
   f.obv = obv; f.cvd = cvd;
   f.obvZ     = IES_ZScore(obv, obvArr, bars);
   f.cvdZ     = IES_ZScore(cvd, cvdArr, bars);
   f.tickVolZ = IES_ZScore((double)iVolume(sym,tf,1), volArr, bars);
   f.volumeSpike = (f.tickVolZ >= 2.0);

   int rl = IES_MinI(p.vwapRollLen, bars);
   double cpv = 0, cv = 0;
   for(int i = 1; i <= rl; i++)
     {
      double tp = (iHigh(sym,tf,i) + iLow(sym,tf,i) + iClose(sym,tf,i)) / 3.0;
      double v  = (double)iVolume(sym,tf,i);
      cpv += tp * v; cv += v;
     }
   f.vwapRoll = (cv > 0 ? cpv / cv : iClose(sym,tf,1));

   datetime dayStart = iTime(sym,tf,0);
   MqlDateTime dt; TimeToStruct(dayStart, dt);
   dt.hour = 0; dt.min = 0; dt.sec = 0;
   dayStart = StructToTime(dt);
   int startIdx = iBarShift(sym,tf,dayStart,false);
   if(startIdx < 0) startIdx = bars - 1;

   double cumTPV = 0, cumV = 0, cumTP2V = 0;
   for(int i = startIdx; i >= 0; i--)
     {
      double tp = (iHigh(sym,tf,i) + iLow(sym,tf,i) + iClose(sym,tf,i)) / 3.0;
      double v  = (double)iVolume(sym,tf,i);
      cumTPV += tp * v; cumV += v; cumTP2V += tp * tp * v;
     }
   if(cumV > 0)
     {
      f.vwapSession = cumTPV / cumV;
      double var = cumTP2V / cumV - f.vwapSession * f.vwapSession;
      double sd  = MathSqrt(IES_Max(var, 0));
      f.vwapUpper1 = f.vwapSession + p.vwapBandSigma1 * sd;
      f.vwapLower1 = f.vwapSession - p.vwapBandSigma1 * sd;
      f.vwapUpper2 = f.vwapSession + p.vwapBandSigma2 * sd;
      f.vwapLower2 = f.vwapSession - p.vwapBandSigma2 * sd;
     }
   if(f.atr > 0 && f.vwapSession > 0)
      f.vwapZ = (iClose(sym,tf,1) - f.vwapSession) / f.atr;
  }

//==============================================================
// Pivots
//==============================================================
void IES_Pivots(const string sym, SFeatures &f)
  {
   double dh = iHigh(sym,PERIOD_D1,1), dl = iLow(sym,PERIOD_D1,1), dc = iClose(sym,PERIOD_D1,1);
   double pd = (dh + dl + dc) / 3.0;
   f.pivD = pd;
   f.pivDR1 = 2*pd - dl; f.pivDS1 = 2*pd - dh;
   f.pivDR2 = pd + (dh - dl); f.pivDS2 = pd - (dh - dl);
   f.pivDR3 = dh + 2*(pd - dl); f.pivDS3 = dl - 2*(dh - pd);
   double wh = iHigh(sym,PERIOD_W1,1), wl = iLow(sym,PERIOD_W1,1), wc = iClose(sym,PERIOD_W1,1);
   double pw = (wh + wl + wc) / 3.0;
   f.pivW = pw;
   f.pivWR1 = 2*pw - wl; f.pivWS1 = 2*pw - wh;
   f.pivWR2 = pw + (wh - wl); f.pivWS2 = pw - (wh - wl);
   f.pivWR3 = wh + 2*(pw - wl); f.pivWS3 = wl - 2*(wh - pw);
   f.pdh = dh; f.pdl = dl; f.pdc = dc; f.pdo = iOpen(sym,PERIOD_D1,1);
  }

//==============================================================
// Session levels
//==============================================================
void IES_SessionLevels(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   f.asiaHi = 0; f.asiaLo = 0; f.ibLonHi = 0; f.ibLonLo = 0; f.ibNyHi = 0; f.ibNyLo = 0;
   f.inIB = false; f.brokeIBhigh = false; f.brokeIBlow = false;
   f.asiaSweepHi = false; f.asiaSweepLo = false;

   int bars = IES_MinI(400, (int)iBars(sym,tf) - 2);
   double aHi = 0, aLo = 0; bool aInit = false;
   double lHi = 0, lLo = 0; bool lInit = false; int lBars = 0;
   double nHi = 0, nLo = 0; bool nInit = false; int nBars = 0;
   int tfSec = IES_MaxI(1, (int)PeriodSeconds(tf));
   int ibNeed = IES_MaxI(1, (p.ibMinutes * 60) / tfSec);

   MqlDateTime tn; TimeToStruct(TimeGMT(), tn);
   int todayKey = tn.year * 10000 + tn.mon * 100 + tn.day;

   for(int i = 1; i < bars; i++)
     {
      datetime gmt = iTime(sym,tf,i) + (TimeGMT() - TimeCurrent());
      MqlDateTime tg; TimeToStruct(gmt, tg);
      int key = tg.year * 10000 + tg.mon * 100 + tg.day;
      bool asia = ((key == todayKey && tg.hour < 7) || (tg.hour >= 21));
      if(asia)
        {
         double h = iHigh(sym,tf,i), l = iLow(sym,tf,i);
         if(!aInit) { aHi = h; aLo = l; aInit = true; }
         else { aHi = IES_Max(aHi, h); aLo = IES_Min(aLo, l); }
        }
      if(key != todayKey) continue;
      if(tg.hour >= 7 && lBars < ibNeed)
        {
         double h = iHigh(sym,tf,i), l = iLow(sym,tf,i);
         if(!lInit) { lHi = h; lLo = l; lInit = true; }
         else { lHi = IES_Max(lHi,h); lLo = IES_Min(lLo,l); }
         lBars++;
        }
      if(tg.hour >= 12 && nBars < ibNeed)
        {
         double h = iHigh(sym,tf,i), l = iLow(sym,tf,i);
         if(!nInit) { nHi = h; nLo = l; nInit = true; }
         else { nHi = IES_Max(nHi,h); nLo = IES_Min(nLo,l); }
         nBars++;
        }
     }
   if(aInit) { f.asiaHi = aHi; f.asiaLo = aLo; }
   if(lInit) { f.ibLonHi = lHi; f.ibLonLo = lLo; }
   if(nInit) { f.ibNyHi  = nHi; f.ibNyLo  = nLo; }

   double ibH = (f.session == IES_SESSION_NEW_YORK || f.session == IES_SESSION_OVERLAP ? (nInit ? nHi : lHi) : lHi);
   double ibL = (f.session == IES_SESSION_NEW_YORK || f.session == IES_SESSION_OVERLAP ? (nInit ? nLo : lLo) : lLo);
   double c1  = iClose(sym,tf,1);
   if(ibH > ibL && ibL > 0)
     {
      f.ibWidthAtr  = (f.atr > 0 ? (ibH - ibL) / f.atr : 0);
      f.inIB        = (c1 <= ibH && c1 >= ibL);
      f.brokeIBhigh = (c1 > ibH);
      f.brokeIBlow  = (c1 < ibL);
     }
   if(aInit)
     {
      for(int i = 1; i <= 8; i++)
        {
         if(iHigh(sym,tf,i) > aHi && iClose(sym,tf,i) < aHi) f.asiaSweepHi = true;
         if(iLow(sym,tf,i)  < aLo && iClose(sym,tf,i) > aLo) f.asiaSweepLo = true;
        }
     }
  }

//==============================================================
// Chop & linreg
//==============================================================
void IES_ChopLinreg(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   int n = (p.chopLen < 3 ? 14 : p.chopLen);
   double sumTR = 0, hh = -1, ll = 1e100;
   for(int i = 1; i <= n; i++)
     {
      double h  = iHigh(sym,tf,i), l = iLow(sym,tf,i);
      double pc = iClose(sym,tf,i+1);
      double tr = IES_Max(h - l, IES_Max(MathAbs(h - pc), MathAbs(l - pc)));
      sumTR += tr;
      hh = IES_Max(hh, h); ll = IES_Min(ll, l);
     }
   f.chop = (hh > ll && n > 1 ? 100.0 * MathLog10(sumTR / (hh - ll)) / MathLog10((double)n) : 50.0);

   int m = (p.linLen < 5 ? 20 : p.linLen);
   double sx=0, sy=0, sxy=0, sx2=0;
   for(int i = 0; i < m; i++)
     { double x = (double)i; double y = iClose(sym,tf,m-i); sx+=x; sy+=y; sxy+=x*y; sx2+=x*x; }
   double den = m*sx2 - sx*sx;
   double slope = (den != 0 ? (m*sxy - sx*sy) / den : 0);
   double yMean = sy / m, ssTot = 0, ssRes = 0, b0 = (sy - slope*sx) / m;
   for(int i = 0; i < m; i++)
     {
      double y = iClose(sym,tf,m-i);
      double yhat = b0 + slope*i;
      ssTot += (y - yMean)*(y - yMean);
      ssRes += (y - yhat)*(y - yhat);
     }
   f.linR2       = (ssTot > 1e-12 ? IES_Max(0, 1 - ssRes / ssTot) : 0);
   f.linSlopeAtr = (f.atr > 0 ? (slope * m) / f.atr : 0);
  }

//==============================================================
// Fib & Marnie
//==============================================================
void IES_MarnieFib(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   double dh = iHigh(sym,PERIOD_D1,1), dl = iLow(sym,PERIOD_D1,1);
   double rng = dh - dl; if(rng <= 0) return;
   double c = iClose(sym,tf,1);
   f.marnieRetr = (c - dl) / rng;
   double lv[7] = {0.0, 0.236, 0.382, 0.500, 0.618, 0.786, 1.000};
   double best = lv[0], bd = DBL_MAX;
   for(int i = 0; i < 7; i++) { double d = MathAbs(f.marnieRetr - lv[i]); if(d < bd) { bd = d; best = lv[i]; } }
   f.marnieNear = best;
   f.marnieInGZ = (f.marnieRetr >= p.gzLo && f.marnieRetr <= p.gzHi);
   f.marnieExt1 = dh + 1.272 * rng;
   f.marnieExt2 = dh + 1.618 * rng;
  }

void IES_Fib(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   if(f.fibHigh <= 0) f.fibHigh = f.swingHigh;
   if(f.fibLow  <= 0) f.fibLow  = f.swingLow;
   double c = iClose(sym,tf,1);
   if(f.fibHigh <= 0 || f.fibLow <= 0 || f.fibHigh <= f.fibLow) return;
   double rng = f.fibHigh - f.fibLow;
   f.fibRetr = (f.trendState >= 0 ? (f.fibHigh - c) / rng : (c - f.fibLow) / rng);
   f.fibRetr = IES_Clamp(f.fibRetr, -0.5, 2.0);
   double bl = 0, bd = DBL_MAX;
   for(int i = 0; i < 6; i++)
     {
      double d = MathAbs(f.fibRetr - p.fibLevels[i]);
      if(d < bd)
        { bd = d; bl = (f.trendState >= 0 ? f.fibHigh - p.fibLevels[i]*rng : f.fibLow + p.fibLevels[i]*rng); }
     }
   f.fibNear      = bl;
   f.inGoldenZone = (f.fibRetr >= p.gzLo && f.fibRetr <= p.gzHi);
   f.fibExt1      = (f.trendState >= 0 ? f.fibLow + 1.272*rng : f.fibHigh - 1.272*rng);
   f.fibExt2      = (f.trendState >= 0 ? f.fibLow + 1.618*rng : f.fibHigh - 1.618*rng);
  }

//==============================================================
// Swings / Structure / Liquidity / FVG / OB / Confluence
//==============================================================
void IES_Swings(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   int lr = p.swingLR;
   int scan = IES_MinI(p.structScanBars, (int)iBars(sym,tf) - lr - 2);
   int foundH = 0, foundL = 0;
   for(int i = lr + 1; i < scan && (foundH < 2 || foundL < 2); i++)
     {
      bool isH = true, isL = true;
      double hv = iHigh(sym,tf,i), lv = iLow(sym,tf,i);
      for(int k = 1; k <= lr; k++)
        {
         if(iHigh(sym,tf,i+k) >= hv) isH = false;
         if(iHigh(sym,tf,i-k) >  hv) isH = false;
         if(iLow(sym,tf,i+k)  <= lv) isL = false;
         if(iLow(sym,tf,i-k)  <  lv) isL = false;
        }
      if(isH && foundH < 2) { if(foundH == 0) f.swingHigh = hv; else f.prevSwingHigh = hv; foundH++; }
      if(isL && foundL < 2) { if(foundL == 0) f.swingLow = lv; else f.prevSwingLow = lv; foundL++; }
     }
  }

void IES_MarketStructure(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   f.bosDir = 0; f.chochDir = 0; f.mssDir = 0; f.mss = false;
   int lr = p.swingLR;
   int scan = IES_MinI(p.structScanBars, (int)iBars(sym,tf) - lr - 2);
   int trend = 0, bosDirFound = 0, bosBar = 0;
   double atr = IES_Max(f.atr, _Point*10);
   int lastH = -1, lastL = -1;

   for(int i = scan; i >= lr + 1; i--)
     {
      bool isH = true, isL = true;
      double hv = iHigh(sym,tf,i), lv = iLow(sym,tf,i);
      for(int k = 1; k <= lr; k++)
        {
         if(iHigh(sym,tf,i+k) >= hv) isH = false;
         if(iHigh(sym,tf,i-k) >  hv) isH = false;
         if(iLow(sym,tf,i+k)  <= lv) isL = false;
         if(iLow(sym,tf,i-k)  <  lv) isL = false;
        }
      if(isH) lastH = i;
      if(isL) lastL = i;
      int confBar = i - lr;
      if(confBar < 1) continue;
      if(lastH >= 0)
        {
         double ph = iHigh(sym,tf,lastH);
         if(iClose(sym,tf,confBar) > ph && confBar < lastH)
           {
            if(trend == -1) f.chochDir = 1;
            if(iClose(sym,tf,confBar-1) <= ph ||
               (iClose(sym,tf,confBar) - iOpen(sym,tf,confBar)) >= p.obDispATR*atr)
              { if(f.bosDir == 0 || confBar > bosBar) { bosDirFound = 1; bosBar = confBar; } }
            trend = 1;
           }
        }
      if(lastL >= 0)
        {
         double pl = iLow(sym,tf,lastL);
         if(iClose(sym,tf,confBar) < pl && confBar < lastL)
           {
            if(trend == 1) f.chochDir = -1;
            if(iClose(sym,tf,confBar-1) >= pl ||
               (iOpen(sym,tf,confBar) - iClose(sym,tf,confBar)) >= p.obDispATR*atr)
              { if(f.bosDir == 0 || confBar > bosBar) { bosDirFound = -1; bosBar = confBar; } }
            trend = -1;
           }
        }
     }
   f.bosDir = bosDirFound;
   f.trendState = trend;
   if(bosBar > 0)
     {
      double body = MathAbs(iClose(sym,tf,bosBar) - iOpen(sym,tf,bosBar));
      if(body >= p.obDispATR * atr)
        { f.mss = true; f.mssDir = (iClose(sym,tf,bosBar) > iOpen(sym,tf,bosBar) ? 1 : -1); }
     }
  }

void IES_Liquidity(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   double atr = IES_Max(f.atr, _Point*10);
   double tol = p.eqTolATR * atr;
   f.liqHiN = 0; f.liqLoN = 0; f.sweepHi = false; f.sweepLo = false;
   f.eqhDetected = false; f.eqlDetected = false;
   int lr = p.swingLR;
   int scan = IES_MinI(p.structScanBars, (int)iBars(sym,tf) - lr - 2);
   double hiLv[IES_MAX_LIQ]; int hiN = 0;
   double loLv[IES_MAX_LIQ]; int loN = 0;
   for(int i = lr + 1; i < scan; i++)
     {
      bool isH = true, isL = true;
      double hv = iHigh(sym,tf,i), lv = iLow(sym,tf,i);
      for(int k = 1; k <= lr; k++)
        {
         if(iHigh(sym,tf,i+k) >= hv) isH = false;
         if(iHigh(sym,tf,i-k) >  hv) isH = false;
         if(iLow(sym,tf,i+k)  <= lv) isL = false;
         if(iLow(sym,tf,i-k)  <  lv) isL = false;
        }
      if(isH)
        {
         bool dup = false;
         for(int j = 0; j < hiN; j++) if(MathAbs(hiLv[j] - hv) <= tol) { dup = true; f.eqhDetected = true; break; }
         if(!dup && hiN < IES_MAX_LIQ) hiLv[hiN++] = hv;
        }
      if(isL)
        {
         bool dup = false;
         for(int j = 0; j < loN; j++) if(MathAbs(loLv[j] - lv) <= tol) { dup = true; f.eqlDetected = true; break; }
         if(!dup && loN < IES_MAX_LIQ) loLv[loN++] = lv;
        }
     }
   for(int j = 0; j < hiN && f.liqHiN < IES_MAX_LIQ; j++) f.liqHi[f.liqHiN++] = hiLv[j];
   for(int j = 0; j < loN && f.liqLoN < IES_MAX_LIQ; j++) f.liqLo[f.liqLoN++] = loLv[j];
   for(int j = 0; j < f.liqHiN; j++)
      for(int i = 1; i <= 5; i++)
         if(iHigh(sym,tf,i) > f.liqHi[j] + p.sweepMinATR*atr && iClose(sym,tf,i) < f.liqHi[j]) f.sweepHi = true;
   for(int j = 0; j < f.liqLoN; j++)
      for(int i = 1; i <= 5; i++)
         if(iLow(sym,tf,i)  < f.liqLo[j] - p.sweepMinATR*atr && iClose(sym,tf,i) > f.liqLo[j]) f.sweepLo = true;
  }

void IES_FVG(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   double atr = IES_Max(f.atr, _Point*10);
   f.fvgBullN = 0; f.fvgBearN = 0;
   f.priceInBullFvg = false; f.priceInBearFvg = false;
   int scan = IES_MinI(50, (int)iBars(sym,tf) - 3);
   double c1 = iClose(sym,tf,1);
   for(int s = 1; s < scan && (f.fvgBullN < IES_MAX_FVG || f.fvgBearN < IES_MAX_FVG); s++)
     {
      double gapUp = iLow(sym,tf,s) - iHigh(sym,tf,s+2);
      double gapDn = iLow(sym,tf,s+2) - iHigh(sym,tf,s);
      if(gapUp >= p.fvgMinATR * atr && f.fvgBullN < IES_MAX_FVG)
        {
         double top = iLow(sym,tf,s), bot = iHigh(sym,tf,s+2), ce = (top+bot)*0.5;
         bool mit = false;
         for(int j = s - 1; j >= 1; j--) if(iLow(sym,tf,j) <= ce) { mit = true; break; }
         f.fvgsBull[f.fvgBullN].top = top;
         f.fvgsBull[f.fvgBullN].bottom = bot;
         f.fvgsBull[f.fvgBullN].midpoint = ce;
         f.fvgsBull[f.fvgBullN].time = iTime(sym,tf,s);
         f.fvgsBull[f.fvgBullN].isBullish = true;
         f.fvgsBull[f.fvgBullN].mitigated = mit;
         if(!mit && c1 >= bot && c1 <= top) f.priceInBullFvg = true;
         f.fvgBullN++;
        }
      if(gapDn >= p.fvgMinATR * atr && f.fvgBearN < IES_MAX_FVG)
        {
         double top = iLow(sym,tf,s+2), bot = iHigh(sym,tf,s), ce = (top+bot)*0.5;
         bool mit = false;
         for(int j = s - 1; j >= 1; j--) if(iHigh(sym,tf,j) >= ce) { mit = true; break; }
         f.fvgsBear[f.fvgBearN].top = top;
         f.fvgsBear[f.fvgBearN].bottom = bot;
         f.fvgsBear[f.fvgBearN].midpoint = ce;
         f.fvgsBear[f.fvgBearN].time = iTime(sym,tf,s);
         f.fvgsBear[f.fvgBearN].isBullish = false;
         f.fvgsBear[f.fvgBearN].mitigated = mit;
         if(!mit && c1 >= bot && c1 <= top) f.priceInBearFvg = true;
         f.fvgBearN++;
        }
     }
  }

void IES_OrderBlocks(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   double atr = IES_Max(f.atr, _Point*10);
   f.obBullN = 0; f.obBearN = 0;
   f.breakerBull = false; f.breakerBear = false;
   f.priceInBullOB = false; f.priceInBearOB = false;
   int scan = IES_MinI(p.obScanBars, (int)iBars(sym,tf) - 6);
   double violBearT = 0, violBearB = 0;
   double violBullT = 0, violBullB = 0;
   double c1 = iClose(sym,tf,1);
   for(int i = scan; i >= 2; i--)
     {
      double o = iOpen(sym,tf,i), c = iClose(sym,tf,i);
      if(MathAbs(c - o) < p.obDispATR * atr) continue;
      if(c > o)
        {
         for(int j = IES_MinI(i+1,scan); j <= i+5 && j < (int)iBars(sym,tf)-1; j++)
           {
            double oj = iOpen(sym,tf,j), cj = iClose(sym,tf,j);
            if(cj < oj)
              {
               double top = IES_Max(oj,cj), bot = IES_Min(oj,cj);
               bool broken = false;
               for(int k = i; k >= 1; k--) if(iClose(sym,tf,k) < bot) { broken = true; break; }
               if(broken && violBearT == 0) { violBearT = top; violBearB = bot; }
               else if(!broken && f.obBullN < IES_MAX_OB)
                 {
                  f.obsBull[f.obBullN].top = top; f.obsBull[f.obBullN].bottom = bot;
                  f.obsBull[f.obBullN].time = iTime(sym,tf,j);
                  f.obsBull[f.obBullN].isBullish = true;
                  f.obsBull[f.obBullN].mitigated = false;
                  f.obsBull[f.obBullN].isBreaker = false;
                  if(c1 >= bot && c1 <= top) f.priceInBullOB = true;
                  f.obBullN++;
                 }
               break;
              }
           }
        }
      else
        {
         for(int j = IES_MinI(i+1,scan); j <= i+5 && j < (int)iBars(sym,tf)-1; j++)
           {
            double oj = iOpen(sym,tf,j), cj = iClose(sym,tf,j);
            if(cj > oj)
              {
               double top = IES_Max(oj,cj), bot = IES_Min(oj,cj);
               bool broken = false;
               for(int k = i; k >= 1; k--) if(iClose(sym,tf,k) > top) { broken = true; break; }
               if(broken && violBullT == 0) { violBullT = top; violBullB = bot; }
               else if(!broken && f.obBearN < IES_MAX_OB)
                 {
                  f.obsBear[f.obBearN].top = top; f.obsBear[f.obBearN].bottom = bot;
                  f.obsBear[f.obBearN].time = iTime(sym,tf,j);
                  f.obsBear[f.obBearN].isBullish = false;
                  f.obsBear[f.obBearN].mitigated = false;
                  f.obsBear[f.obBearN].isBreaker = false;
                  if(c1 >= bot && c1 <= top) f.priceInBearOB = true;
                  f.obBearN++;
                 }
               break;
              }
           }
        }
     }
   if(violBearT > 0)
     {
      bool retest = false;
      for(int i = 1; i <= 5; i++) if(iLow(sym,tf,i) <= violBearT && iClose(sym,tf,i) > violBearT) retest = true;
      if(retest) { f.breakerBull = true; f.brkBullT = violBearT; f.brkBullB = violBearB; }
     }
   if(violBullT > 0)
     {
      bool retest = false;
      for(int i = 1; i <= 5; i++) if(iHigh(sym,tf,i) >= violBullB && iClose(sym,tf,i) < violBullB) retest = true;
      if(retest) { f.breakerBear = true; f.brkBearT = violBullT; f.brkBearB = violBullB; }
     }
  }

double IES_Confluence(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   double atr = IES_Max(f.atr, _Point*10);
   double c   = iClose(sym,tf,1);
   double tol = p.confTolATR * atr;
   double lv[64]; int n = 0;
   if(f.swingHigh > 0 && n < 64)     lv[n++] = f.swingHigh;
   if(f.swingLow > 0 && n < 64)      lv[n++] = f.swingLow;
   if(f.prevSwingHigh > 0 && n < 64) lv[n++] = f.prevSwingHigh;
   if(f.prevSwingLow > 0 && n < 64)  lv[n++] = f.prevSwingLow;
   for(int i = 0; i < f.liqHiN && n < 64; i++) lv[n++] = f.liqHi[i];
   for(int i = 0; i < f.liqLoN && n < 64; i++) lv[n++] = f.liqLo[i];
   for(int i = 0; i < f.fvgBullN && n < 62; i++) { lv[n++] = f.fvgsBull[i].bottom; lv[n++] = f.fvgsBull[i].midpoint; }
   for(int i = 0; i < f.fvgBearN && n < 62; i++) { lv[n++] = f.fvgsBear[i].top;    lv[n++] = f.fvgsBear[i].midpoint; }
   for(int i = 0; i < f.obBullN && n < 64; i++)  lv[n++] = f.obsBull[i].top;
   for(int i = 0; i < f.obBearN && n < 64; i++)  lv[n++] = f.obsBear[i].bottom;
   if(f.vwapSession > 0 && n < 61) { lv[n++] = f.vwapSession; lv[n++] = f.vwapUpper1; lv[n++] = f.vwapLower1; }
   if(f.bbUpper > 0 && n < 61)     { lv[n++] = f.bbUpper; lv[n++] = f.bbLower; lv[n++] = f.bbMiddle; }
   if(f.fibNear > 0 && n < 64) lv[n++] = f.fibNear;
   if(f.fibExt1 > 0 && n < 64) lv[n++] = f.fibExt1;
   if(f.pivD > 0 && n < 60) { lv[n++] = f.pivDR1; lv[n++] = f.pivDS1; lv[n++] = f.pivDR2; lv[n++] = f.pivDS2; }
   if(f.pivW > 0 && n < 61) { lv[n++] = f.pivWR1; lv[n++] = f.pivWS1; lv[n++] = f.pivW; }
   if(f.brkBullT > 0 && n < 64) lv[n++] = f.brkBullT;
   if(f.brkBearB > 0 && n < 64) lv[n++] = f.brkBearB;
   int hits = 0;
   for(int i = 0; i < n; i++) if(MathAbs(lv[i] - c) <= tol) hits++;
   f.confluenceHits  = hits;
   f.confluenceScore = IES_Min((double)hits, 10.0);
   return(f.confluenceScore);
  }

int IES_Regime(const string sym, const ENUM_TIMEFRAMES tf, const SIESParams &p, SFeatures &f)
  {
   bool diBull = (f.adxPlusDI > f.adxMinusDI);
   double c1 = iClose(sym,tf,1);
   bool above = (f.ema50 > 0 && c1 > f.ema50);
   bool below = (f.ema50 > 0 && c1 < f.ema50);
   bool chopRange = (f.chop >= p.chopRange);
   int r = IES_REGIME_RANGE;
   if(f.squeezeRelease || (f.bbWidthZ >= p.regBBWidthZ && (f.breakoutBull || f.breakoutBear)))
      r = IES_REGIME_BREAKOUT;
   else if((diBull && above && f.linSlopeAtr >= 0) || (above && f.ema9 > f.ema21 && f.linSlopeAtr > 0))
      r = IES_REGIME_TRENDING_BULLISH;
   else if((!diBull && below && f.linSlopeAtr <= 0) || (below && f.ema9 < f.ema21 && f.linSlopeAtr < 0))
      r = IES_REGIME_TRENDING_BEARISH;
   else if(f.squeezeOn)
      r = IES_REGIME_SQUEEZE;
   else if((chopRange || f.adx < p.regAdxRange) && (f.bbBullRev || f.bbBearRev || MathAbs(f.vwapZ) >= 2.0))
      r = IES_REGIME_MEAN_REVERSION;
   else if(f.atrZ >= p.regAtrZ)
      r = IES_REGIME_HIGH_VOLATILITY;
   else
      r = IES_REGIME_RANGE;
   f.regime = r;
   return(r);
  }

//==============================================================
// Order execution primitives
//==============================================================
bool IES_OrderRaw(const ENUM_ORDER_TYPE ot, const string sym, const double lots,
                  const double price, const double sl, const double tp,
                  const ENUM_ORDER_TYPE_FILLING fill, const string comment,
                  MqlTradeResult &res)
  {
   MqlTradeRequest req; ZeroMemory(req); ZeroMemory(res);
   req.action = TRADE_ACTION_DEAL;
   req.symbol = sym; req.volume = lots; req.type = ot;
   req.price = price; req.sl = sl; req.tp = tp;
   req.deviation = InpDeviationPts; req.magic = InpMagic; req.comment = comment;
   req.type_filling = fill; req.type_time = ORDER_TIME_GTC;
   ResetLastError();
   if(!OrderSend(req, res))
      PrintFormat("OrderSend failed: fill=%s ret=%d err=%d",
                  EnumToString(fill), res.retcode, GetLastError());
   return(res.retcode == TRADE_RETCODE_DONE ||
          res.retcode == TRADE_RETCODE_PLACED ||
          res.retcode == TRADE_RETCODE_DONE_PARTIAL);
  }

bool IES_ModifyPosition(const ulong ticket, const double sl, const double tp)
  {
   if(!PositionSelectByTicket(ticket)) return(false);
   MqlTradeRequest req; MqlTradeResult res; ZeroMemory(req); ZeroMemory(res);
   req.action = TRADE_ACTION_SLTP;
   req.position = ticket;
   req.symbol = PositionGetString(POSITION_SYMBOL);
   req.sl = sl; req.tp = tp; req.magic = InpMagic;
   if(!OrderSend(req, res))
     { PrintFormat("Modify failed %I64u ret=%d err=%d", ticket, res.retcode, GetLastError()); return(false); }
   return(res.retcode == TRADE_RETCODE_DONE);
  }

ulong IES_FindOurPosition()
  {
   for(int i = PositionsTotal()-1; i >= 0; i--)
     {
      ulong t = PositionGetTicket(i);
      if(t == 0 || !PositionSelectByTicket(t)) continue;
      if(PositionGetString(POSITION_SYMBOL) == _Symbol &&
         (long)PositionGetInteger(POSITION_MAGIC) == InpMagic) return(t);
     }
   return(0);
  }

bool IES_ClosePartial(const ulong ticket, const double volume)
  {
   if(!PositionSelectByTicket(ticket)) return(false);
   ENUM_POSITION_TYPE pt = (ENUM_POSITION_TYPE)PositionGetInteger(POSITION_TYPE);
   string sym = PositionGetString(POSITION_SYMBOL);
   MqlTick tick;
   if(!SymbolInfoTick(sym, tick)) return(false);
   MqlTradeRequest req; MqlTradeResult res; ZeroMemory(req); ZeroMemory(res);
   req.action = TRADE_ACTION_DEAL;
   req.position = ticket; req.symbol = sym; req.volume = volume;
   req.deviation = InpDeviationPts; req.magic = InpMagic;
   req.comment = "IES Partial"; req.type_filling = IES_Filling(sym);
   if(pt == POSITION_TYPE_BUY) { req.type = ORDER_TYPE_SELL; req.price = tick.bid; }
   else                        { req.type = ORDER_TYPE_BUY;  req.price = tick.ask; }
   if(!OrderSend(req, res))
     { PrintFormat("Partial close failed ret=%d", res.retcode); return(false); }
   return(res.retcode == TRADE_RETCODE_DONE || res.retcode == TRADE_RETCODE_DONE_PARTIAL);
  }

bool IES_ClosePosition(const ulong ticket)
  {
   if(!PositionSelectByTicket(ticket)) return(false);
   return(IES_ClosePartial(ticket, PositionGetDouble(POSITION_VOLUME)));
  }

string IES_KeyPartial(const ulong t) { return("IES_P_" + IntegerToString(InpMagic) + "_" + IntegerToString((long)t)); }
string IES_KeyR(const ulong t)       { return("IES_R_" + IntegerToString(InpMagic) + "_" + IntegerToString((long)t)); }
bool   IES_WasPartialed(const ulong t) { return(GlobalVariableCheck(IES_KeyPartial(t))); }
void   IES_MarkPartialed(const ulong t) { GlobalVariableSet(IES_KeyPartial(t), 1.0); }
void   IES_StoreInitialR(const ulong t, const double r) { GlobalVariableSet(IES_KeyR(t), r); }
double IES_GetInitialR(const ulong t, const double open, const double sl)
  {
   string k = IES_KeyR(t);
   if(GlobalVariableCheck(k)) { double r = GlobalVariableGet(k); if(r > 0) return(r); }
   double r = MathAbs(open - sl);
   if(r > 0) IES_StoreInitialR(t, r);
   return(r);
  }
void IES_CleanGlobals(const ulong t)
  {
   if(GlobalVariableCheck(IES_KeyPartial(t)))              GlobalVariableDel(IES_KeyPartial(t));
   if(GlobalVariableCheck(IES_KeyPartial(t) + "_2"))       GlobalVariableDel(IES_KeyPartial(t) + "_2");
   if(GlobalVariableCheck(IES_KeyR(t)))                    GlobalVariableDel(IES_KeyR(t));
  }

bool IES_SendDeal(const ENUM_ORDER_TYPE ot, const string sym, const double lots,
                  const double price, const double sl, const double tp, const string comment)
  {
   ENUM_ORDER_TYPE_FILLING fills[3];
   fills[0] = IES_Filling(sym);
   fills[1] = ORDER_FILLING_IOC;
   fills[2] = ORDER_FILLING_FOK;
   MqlTradeResult res;

   for(int i = 0; i < 3; i++)
     {
      if(i > 0 && fills[i] == fills[0]) continue;
      if(IES_OrderRaw(ot, sym, lots, price, sl, tp, fills[i], comment, res))
        {
         ulong ticket = res.order;
         if(ticket == 0 && res.deal != 0 && HistoryDealSelect(res.deal))
            ticket = (ulong)HistoryDealGetInteger(res.deal, DEAL_POSITION_ID);
         if(ticket == 0) ticket = IES_FindOurPosition();
         if(ticket != 0) IES_StoreInitialR(ticket, MathAbs(price - sl));
         return(true);
        }
      if(res.retcode == TRADE_RETCODE_INVALID_STOPS ||
         res.retcode == TRADE_RETCODE_INVALID_PRICE) break;
     }

   Print("IES: Retrying entry without stops...");
   for(int i = 0; i < 3; i++)
     {
      if(i > 0 && fills[i] == fills[0]) continue;
      if(!IES_OrderRaw(ot, sym, lots, price, 0, 0, fills[i], comment, res)) continue;
      ulong ticket = 0;
      for(int a = 0; a < 5; a++)
        {
         Sleep(60);
         if(res.deal != 0 && HistoryDealSelect(res.deal))
            ticket = (ulong)HistoryDealGetInteger(res.deal, DEAL_POSITION_ID);
         if(ticket == 0) ticket = res.order;
         if(ticket == 0) ticket = IES_FindOurPosition();
         if(ticket != 0 && PositionSelectByTicket(ticket)) break;
        }
      if(ticket != 0)
        {
         IES_StoreInitialR(ticket, MathAbs(price - sl));
         if(!IES_ModifyPosition(ticket, sl, tp))
            PrintFormat("Warning: attached SL/TP failed for %I64u", ticket);
         return(true);
        }
      return(true);
     }
   return(false);
  }

//==============================================================
// Confluence score
//==============================================================
double ConfluenceScore(SFeatures &f, CIESDrivers &drv)
  {
   double sTrend=0, sMtf=0, sMom=0, sVol=0, sSmc=0, sGeom=0, sCandle=0, sMacro=0;

   sTrend += (f.ema9  > f.ema21  ?  4 : -4);
   sTrend += (f.ema21 > f.ema50  ?  3 : -3);
   sTrend += (f.ema50 > f.ema200 ?  2 : -2);
   sTrend += (f.sma50 > f.sma200 ?  1 : -1);
   if(f.emaStackBull) sTrend += 3;
   if(f.emaStackBear) sTrend -= 3;
   if(f.emaCross921 > 0) sTrend += 2;
   if(f.emaCross921 < 0) sTrend -= 2;
   sTrend += (f.trendState > 0 ? 3 : (f.trendState < 0 ? -3 : 0));
   if(f.bosDir   > 0) sTrend += 2;  if(f.bosDir   < 0) sTrend -= 2;
   if(f.chochDir > 0) sTrend += 2;  if(f.chochDir < 0) sTrend -= 2;
   if(f.mss) sTrend += f.mssDir * 2;

   sMtf += (f.mtfScore > 0 ? 4 : (f.mtfScore < 0 ? -4 : 0));
   if(f.mtfStates[2] > 0) sMtf += 2;  if(f.mtfStates[2] < 0) sMtf -= 2;
   if(f.mtfStates[3] > 0) sMtf += 2;  if(f.mtfStates[3] < 0) sMtf -= 2;

   sMom += (f.psarLong ? 2 : -2);
   sMom += (f.macdHist > 0 ? 2 : -2);
   if(f.macdBullCross) sMom += 3;
   if(f.macdBearCross) sMom -= 3;
   sMom += (f.adxPlusDI > f.adxMinusDI ? 1 : -1);
   if(f.adxSlope > 0 && f.adx >= 20) sMom += (f.adxPlusDI > f.adxMinusDI ? 2 : -2);
   sMom += (f.ichiCloudPos > 0 ? 2 : (f.ichiCloudPos < 0 ? -2 : 0));
   sMom += (f.ichiTenkan > f.ichiKijun ? 1 : -1);
   sMom += (f.ichiChikouBull ? 1 : -1);
   sMom += (f.rsi > 52 ? 2 : (f.rsi < 48 ? -2 : 0));
   sMom += (f.stochK > f.stochD && f.stochK < 80 ? 1 :
            (f.stochK < f.stochD && f.stochK > 20 ? -1 : 0));
   sMom += (f.srsiK > f.srsiD ? 1 : -1);
   sMom += (f.cci > 0 ? 1 : -1);

   sVol += (f.cvdZ > 0.6 ? 2 : (f.cvdZ < -0.6 ? -2 : 0));
   sVol += (f.obvZ > 0.5 ? 1 : (f.obvZ < -0.5 ? -1 : 0));
   if(f.volumeSpike) sVol += (f.clv >= 0.5 ? 1 : -1);
   if(f.absorption)  sVol += (f.clv >= 0.6 ? 1 : (f.clv <= 0.4 ? -1 : 0));

   if(f.sweepLo) sSmc += 3;
   if(f.sweepHi) sSmc -= 3;
   if(InpAsianSweepFilter && f.asiaSweepLo) sSmc += 3;
   if(InpAsianSweepFilter && f.asiaSweepHi) sSmc -= 3;
   if(f.breakerBull) sSmc += 2;
   if(f.breakerBear) sSmc -= 2;
   if(f.priceInBullFvg) sSmc += 2;
   if(f.priceInBearFvg) sSmc -= 2;
   if(f.priceInBullOB)  sSmc += 2;
   if(f.priceInBearOB)  sSmc -= 2;
   if(f.brokeIBhigh && f.clv > 0.6) sSmc += 3;
   if(f.brokeIBlow  && f.clv < 0.4) sSmc -= 3;

   if(f.vwapZ >  0.3 && f.vwapZ <  1.5) sGeom += 2;
   else if(f.vwapZ < -0.3 && f.vwapZ > -1.5) sGeom -= 2;
   else if(f.vwapZ >=  2.2) sGeom -= 2;
   else if(f.vwapZ <= -2.2) sGeom += 2;
   if(f.bbBullRev) sGeom += 2;
   if(f.bbBearRev) sGeom -= 2;
   if(f.inGoldenZone) sGeom += (f.trendState >= 0 ? 2 : -2);
   if(f.marnieInGZ)   sGeom += (f.trendState >= 0 ? 1 : -1);
   if(f.confluenceHits >= 3) sGeom += (f.trendState >= 0 ? 2 : -2);

   if(f.patEngulfBull || f.microReclaimBull) sCandle += 2;
   if(f.patEngulfBear || f.microReclaimBear) sCandle -= 2;
   if(f.patPinBull && f.wickAtr >= 0.4) sCandle += 2;
   if(f.patPinBear && f.wickAtr >= 0.4) sCandle -= 2;
   if(f.clv >= 0.80) sCandle += 1;
   if(f.clv <= 0.20) sCandle -= 1;
   if(f.breakoutBull && !f.absorption) sCandle += 2;
   if(f.breakoutBear && !f.absorption) sCandle -= 2;
   if(f.displacement) sCandle += (iClose(_Symbol, InpTimeframe, 1) > iOpen(_Symbol, InpTimeframe, 1) ? 2 : -2);
   if(f.squeezeRelease) sCandle += (f.linSlopeAtr >= 0 ? 2 : -2);
   if(f.linR2 >= 0.45)
      sCandle += (f.linSlopeAtr > 0 ? 2 : (f.linSlopeAtr < 0 ? -2 : 0));
   sCandle += f.todBucket;

   double db = drv.Bias();
   f.macroBias = db;
   sMacro += (double)MathRound(db * 6.0);

   f.scoreTrend = sTrend; f.scoreMTF = sMtf; f.scoreMomentum = sMom;
   f.scoreVolume = sVol;  f.scoreSMC = sSmc; f.scoreGeometry = sGeom;
   f.scoreCandle = sCandle; f.scoreMacro = sMacro;

   double sAstro = 0;
   if(InpEnableAstro)
     {
      int d = (sTrend > 0 ? 1 : (sTrend < 0 ? -1 : 0));
      double a = g_astro.GetCosmicScore(_Symbol, d);
      sAstro = a * (InpAstroWeight / 100.0) * 0.15;
     }
   f.scoreAstro = sAstro;

   double total = (sTrend + sMtf + sMom + sVol + sSmc + sGeom + sCandle + sMacro + sAstro) * 1.5;
   f.finalCompositeScore = IES_Clamp(total, -100.0, 100.0);
   return(f.finalCompositeScore);
  }

bool RegimeAllows(const int regime, const int dir)
  {
   switch(regime)
     {
      case IES_REGIME_TRENDING_BULLISH: return(dir > 0);
      case IES_REGIME_TRENDING_BEARISH: return(dir < 0);
      case IES_REGIME_SQUEEZE:          return(false);
     }
   return(true);
  }

//==============================================================
// BuildTradePlan
//==============================================================
void BuildTradePlan(const int dir, STradePlan &plan, const CBrokerInfo &br,
                    const double spread, const double commissionPerLotRT,
                    const double slippageEstPts, const double swapPerLotNight,
                    const bool allowMinLot)
  {
   ZeroMemory(plan);
   plan.valid = false; plan.dir = dir; plan.reject = "";

   double ask = SymbolInfoDouble(br.sym, SYMBOL_ASK);
   double bid = SymbolInfoDouble(br.sym, SYMBOL_BID);
   if(ask <= 0 || bid <= 0) { plan.reject = "Invalid quote"; return; }
   plan.entry = (dir > 0 ? ask : bid);

   double atr = g_f.atr;
   if(atr <= 0) { plan.reject = "ATR unavailable"; return; }

   double atrDist = InpSLatrMult * atr;
   double structDist = atrDist;
   if(InpUseStructureSL)
     {
      if(dir > 0 && g_f.swingLow  > 0) structDist = plan.entry - g_f.swingLow;
      if(dir < 0 && g_f.swingHigh > 0) structDist = g_f.swingHigh - plan.entry;
      if(structDist < atrDist * InpMinSlAtr) structDist = atrDist * InpMinSlAtr;
      if(structDist > atrDist * InpMaxSlAtr) structDist = atrDist * InpMaxSlAtr;
     }
   double rawDist = IES_Max(structDist, atrDist * InpMinSlAtr);

   if(InpMaxSlUSD > 0 && rawDist > InpMaxSlUSD) rawDist = InpMaxSlUSD;

   double minStopPrice = (br.stopsLevel + 2) * br.point;
   double dist = IES_Max(rawDist, minStopPrice);
   dist += spread * 1.5;

   plan.slDist = dist;
   plan.slPts  = dist / br.point;

   double tpDist1 = dist * InpRR1;
   double tpDist2 = dist * InpRR2;
   double tpDist3 = dist * InpRR3;

   double stepPrice = IES_Max(br.point * 20.0, minStopPrice);
   if(tpDist2 <= tpDist1 + stepPrice) tpDist2 = tpDist1 + stepPrice;
   if(tpDist3 <= tpDist2 + stepPrice) tpDist3 = tpDist2 + stepPrice;

   if(dir > 0)
     {
      plan.sl  = plan.entry - dist;
      plan.tp1 = plan.entry + tpDist1;
      plan.tp2 = plan.entry + tpDist2;
      plan.tp3 = plan.entry + tpDist3;
     }
   else
     {
      plan.sl  = plan.entry + dist;
      plan.tp1 = plan.entry - tpDist1;
      plan.tp2 = plan.entry - tpDist2;
      plan.tp3 = plan.entry - tpDist3;
     }
   plan.sl  = NormalizeDouble(plan.sl,  br.digits);
   plan.tp1 = NormalizeDouble(plan.tp1, br.digits);
   plan.tp2 = NormalizeDouble(plan.tp2, br.digits);
   plan.tp3 = NormalizeDouble(plan.tp3, br.digits);

   double mpp = br.MoneyPerPoint();
   if(mpp <= 0) { plan.reject = "tick value invalid"; return; }
   double slMoneyPerLot = plan.slPts * mpp;

   plan.lots = g_cp.LotSize(plan.slPts, br, allowMinLot);
   if(plan.lots <= 0) { plan.reject = "Sizing produced 0 lots"; return; }

   plan.riskMoney = plan.lots * slMoneyPerLot;

   double costPtsEquiv = (spread / br.point) + slippageEstPts +
                         (commissionPerLotRT + swapPerLotNight) / IES_Max(mpp, 1e-9);
   plan.estCostMoney = plan.lots * costPtsEquiv * mpp;

   plan.r1 = plan.slPts > 0 ? (tpDist1 / br.point) / plan.slPts : 0;
   plan.r2 = plan.slPts > 0 ? (tpDist2 / br.point) / plan.slPts : 0;
   plan.r3 = plan.slPts > 0 ? (tpDist3 / br.point) / plan.slPts : 0;

   double costR = costPtsEquiv / IES_Max(plan.slPts, 1e-9);
   plan.netRR1 = plan.r1 - costR;
   plan.netRR2 = plan.r2 - costR;
   plan.netRR3 = plan.r3 - costR;
   plan.grossRR1 = plan.r1;
   plan.grossRR2 = plan.r2;
   plan.grossRR3 = plan.r3;

   if(plan.netRR1 < 0.5)  { plan.reject = StringFormat("Net R:R1 %.2f too low", plan.netRR1); return; }
   if(plan.netRR3 < 1.5)  { plan.reject = StringFormat("Net R:R3 %.2f too low", plan.netRR3); return; }
   plan.valid = true;
  }

//==============================================================
// Chart/HUD — v1.00: CleanChartObjects scans ALL windows
//==============================================================
void CleanChartObjects()
  {
   int total = ObjectsTotal(0, -1, -1);
   for(int i = total - 1; i >= 0; i--)
     {
      string n = ObjectName(0, i, -1, -1);
      if(StringFind(n, IES_HUD_PREFIX) == 0 || StringFind(n, IES_SMC_PREFIX) == 0)
         ObjectDelete(0, n);
     }
   ChartRedraw(0);
  }

void autoSetLabel(const int x, const int y, const string id, const string text,
                  const color col, const int fs = 9, const bool bold = false)
  {
   string name = IES_HUD_PREFIX + id;
   if(ObjectFind(0, name) < 0)
     {
      ObjectCreate(0, name, OBJ_LABEL, 0, 0, 0);
      ObjectSetInteger(0, name, OBJPROP_CORNER, CORNER_LEFT_UPPER);
      ObjectSetInteger(0, name, OBJPROP_SELECTABLE, false);
     }
   ObjectSetInteger(0, name, OBJPROP_XDISTANCE, x);
   ObjectSetInteger(0, name, OBJPROP_YDISTANCE, y);
   ObjectSetString(0, name, OBJPROP_TEXT, text);
   ObjectSetInteger(0, name, OBJPROP_COLOR, col);
   ObjectSetInteger(0, name, OBJPROP_FONTSIZE, fs);
   ObjectSetString(0, name, OBJPROP_FONT, (bold ? "Arial Bold" : "Arial"));
  }

void DrawSMCChartObjects(const SFeatures &f)
  {
   if(!InpDrawSMC) return;
   int total = ObjectsTotal(0, -1, -1);
   for(int i = total-1; i >= 0; i--)
     {
      string n = ObjectName(0, i, -1, -1);
      if(StringFind(n, IES_SMC_PREFIX) == 0) ObjectDelete(0, n);
     }
   for(int i = 0; i < f.fvgBullN && i < 3; i++)
     {
      if(f.fvgsBull[i].mitigated) continue;
      string n = IES_SMC_PREFIX + "FVG_BULL_" + IntegerToString(i);
      datetime t1 = f.fvgsBull[i].time;
      datetime t2 = t1 + PeriodSeconds(InpTimeframe) * 15;
      ObjectCreate(0, n, OBJ_RECTANGLE, 0, t1, f.fvgsBull[i].top, t2, f.fvgsBull[i].bottom);
      ObjectSetInteger(0, n, OBJPROP_COLOR, clrMediumSeaGreen);
      ObjectSetInteger(0, n, OBJPROP_BACK, true);
     }
   for(int i = 0; i < f.fvgBearN && i < 3; i++)
     {
      if(f.fvgsBear[i].mitigated) continue;
      string n = IES_SMC_PREFIX + "FVG_BEAR_" + IntegerToString(i);
      datetime t1 = f.fvgsBear[i].time;
      datetime t2 = t1 + PeriodSeconds(InpTimeframe) * 15;
      ObjectCreate(0, n, OBJ_RECTANGLE, 0, t1, f.fvgsBear[i].top, t2, f.fvgsBear[i].bottom);
      ObjectSetInteger(0, n, OBJPROP_COLOR, clrIndianRed);
      ObjectSetInteger(0, n, OBJPROP_BACK, true);
     }
  }

void Panel()
  {
   // UI_NONE = no dashboard at all
   if(InpUiMode == UI_NONE)
     {
      Comment("");
      return;
     }

   double pt = g_br.point > 0 ? g_br.point : _Point;
   double spread = (pt > 0 ? (SymbolInfoDouble(g_sym, SYMBOL_ASK) - SymbolInfoDouble(g_sym, SYMBOL_BID)) / pt : 0);

   if(InpUiMode == UI_TEXT_COMMENT)
     {
      string t;
      t  = "========================================================\n";
      t += "  XAUUSD IES v1.00 | " + g_sym + " (" + EnumToString(InpTimeframe) + ")\n";
      t += "========================================================\n";
      t += StringFormat("  Score %.1f | Tier %s | Regime %s | Session %s\n",
                        g_score, IES_TierToString(g_f.tier),
                        IES_RegimeToString(g_f.regime), IES_SessionToString(g_f.session));
      t += StringFormat("  Trend %d | BOS %d | CHoCH %d | MTF %d\n",
                        g_f.trendState, g_f.bosDir, g_f.chochDir, g_f.mtfScore);
      t += StringFormat("  ADX %.1f | CHOP %.1f | R2 %.2f | ATR %.3f\n",
                        g_f.adx, g_f.chop, g_f.linR2, g_f.atr);
      t += StringFormat("  VWAPz %.2f | CVDz %.2f | OBVz %.2f | Squeeze %s\n",
                        g_f.vwapZ, g_f.cvdZ, g_f.obvZ,
                        (g_f.squeezeRelease ? "REL" : (g_f.squeezeOn ? "ON" : "OFF")));
      t += "  " + g_drv.Describe() + "\n";
      t += StringFormat("  DayPnL %.2f | Trades %d/%d | Consec %d\n",
                        g_cp.DayPnL(), g_cp.TradesToday(), InpMaxTradesDay, g_cp.ConsecLosses());
      t += StringFormat("  Spread %.0f pts | Status %s\n",
                        spread, (g_cp.Halted() ? "HALT" : g_block));
      if(g_f.newsRisk) t += "  *** NEWS: " + g_f.newsEventName + " ***\n";
      Comment(t);
      return;
     }

   Comment("");
   int x = 20, y = 30;
   int rowH = 18;

   string idBox = IES_HUD_PREFIX + "BG";
   if(ObjectFind(0, idBox) < 0)
     {
      ObjectCreate(0, idBox, OBJ_RECTANGLE_LABEL, 0, 0, 0);
      ObjectSetInteger(0, idBox, OBJPROP_XDISTANCE, x - 10);
      ObjectSetInteger(0, idBox, OBJPROP_YDISTANCE, y - 10);
      ObjectSetInteger(0, idBox, OBJPROP_XSIZE, 400);
      ObjectSetInteger(0, idBox, OBJPROP_YSIZE, 300);
      ObjectSetInteger(0, idBox, OBJPROP_BGCOLOR, C'20,24,30');
      ObjectSetInteger(0, idBox, OBJPROP_BORDER_COLOR, C'50,60,75');
      ObjectSetInteger(0, idBox, OBJPROP_BORDER_TYPE, BORDER_FLAT);
      ObjectSetInteger(0, idBox, OBJPROP_CORNER, CORNER_LEFT_UPPER);
      ObjectSetInteger(0, idBox, OBJPROP_BACK, false);
      ObjectSetInteger(0, idBox, OBJPROP_SELECTABLE, false);
     }

   autoSetLabel(x, y, "L0", "XAUUSD IES v1.00 | " + g_sym, clrGold, 10, true); y += rowH + 4;
   color scCol = (g_score >= InpMinScore ? clrSpringGreen :
                  (g_score <= -InpMinScore ? clrCrimson : clrSilver));
   autoSetLabel(x, y, "L1",
                StringFormat("Score %.1f | Tier %s | Min |%.0f|",
                             g_score, IES_TierToString(g_f.tier), InpMinScore),
                scCol, 9, true); y += rowH;
   autoSetLabel(x, y, "L2", "Regime: " + IES_RegimeToString(g_f.regime), clrAqua, 9); y += rowH;
   autoSetLabel(x, y, "L3",
                StringFormat("Session %s | ToD %d",
                             IES_SessionToString(g_f.session), g_f.todBucket), clrWhite, 9); y += rowH;
   autoSetLabel(x, y, "L4",
                StringFormat("MTF %+d [%+d %+d %+d %+d]",
                             g_f.mtfScore, g_f.mtfStates[0], g_f.mtfStates[1],
                             g_f.mtfStates[2], g_f.mtfStates[3]),
                (g_f.mtfScore > 0 ? clrLightGreen : (g_f.mtfScore < 0 ? clrLightCoral : clrWhiteSmoke)), 9); y += rowH;
   autoSetLabel(x, y, "L5",
                StringFormat("Trend %+d BOS %+d CHoCH %+d MSS %s",
                             g_f.trendState, g_f.bosDir, g_f.chochDir,
                             (g_f.mss ? "Y" : "N")), clrDodgerBlue, 9); y += rowH;
   autoSetLabel(x, y, "L6",
                StringFormat("VWAPz %.2f CVDz %.2f OBVz %.2f",
                             g_f.vwapZ, g_f.cvdZ, g_f.obvZ), clrCornflowerBlue, 9); y += rowH;
   autoSetLabel(x, y, "L7",
                StringFormat("ADX %.1f CHOP %.1f R2 %.2f",
                             g_f.adx, g_f.chop, g_f.linR2), clrKhaki, 9); y += rowH;
   autoSetLabel(x, y, "L8",
                StringFormat("Liq SwHi %d SwLo %d | Asia %d/%d | IB %s",
                             (int)g_f.sweepHi, (int)g_f.sweepLo,
                             (int)g_f.asiaSweepHi, (int)g_f.asiaSweepLo,
                             (g_f.brokeIBhigh ? "Up" : (g_f.brokeIBlow ? "Dn" : (g_f.inIB ? "In" : "-")))),
                clrPlum, 9); y += rowH;
   autoSetLabel(x, y, "L9", StringFormat("Driver %+.2f | Spread %.0f pts",
                                         g_drv.Bias(), spread), clrOrange, 9); y += rowH;
   autoSetLabel(x, y, "L10",
                StringFormat("DayPnL %.2f Trades %d/%d CL %d",
                             g_cp.DayPnL(), g_cp.TradesToday(),
                             InpMaxTradesDay, g_cp.ConsecLosses()),
                (g_cp.DayPnL() >= 0 ? clrLimeGreen : clrSalmon), 9); y += rowH;
   color stCol = (g_cp.Halted() ? clrRed : (g_f.newsRisk ? clrOrangeRed : clrLightGray));
   string stTxt = (g_cp.Halted() ? "HALT " + g_cp.HaltReason()
                  : (g_f.newsRisk ? "NEWS " + g_f.newsEventName : "STATUS " + g_block));
   autoSetLabel(x, y, "L11", stTxt, stCol, 9, true);

   if(InpEnableAstro && InpShowAstroHUD)
     {
      int ax = 435, ay = 30;
      string aBox = IES_HUD_PREFIX + "ASTRO_BG";
      if(ObjectFind(0, aBox) < 0)
        {
         ObjectCreate(0, aBox, OBJ_RECTANGLE_LABEL, 0, 0, 0);
         ObjectSetInteger(0, aBox, OBJPROP_XDISTANCE, ax - 10);
         ObjectSetInteger(0, aBox, OBJPROP_YDISTANCE, ay - 10);
         ObjectSetInteger(0, aBox, OBJPROP_XSIZE, 420);
         ObjectSetInteger(0, aBox, OBJPROP_YSIZE, 300);
         ObjectSetInteger(0, aBox, OBJPROP_BGCOLOR, C'18,20,28');
         ObjectSetInteger(0, aBox, OBJPROP_BORDER_COLOR, C'65,55,85');
         ObjectSetInteger(0, aBox, OBJPROP_BORDER_TYPE, BORDER_FLAT);
         ObjectSetInteger(0, aBox, OBJPROP_CORNER, CORNER_LEFT_UPPER);
         ObjectSetInteger(0, aBox, OBJPROP_BACK, false);
         ObjectSetInteger(0, aBox, OBJPROP_SELECTABLE, false);
        }
      autoSetLabel(ax, ay, "A0", "COSMIC ASTRO & VEDIC MATRIX", clrMediumOrchid, 10, true); ay += rowH + 4;
      autoSetLabel(ax, ay, "A1",
                   StringFormat("Micro: %+d %+d %+d %+d %+d %+d %+d",
                                g_f.astro.microMap.stateM1, g_f.astro.microMap.stateM5,
                                g_f.astro.microMap.stateM15, g_f.astro.microMap.stateM30,
                                g_f.astro.microMap.stateH1, g_f.astro.microMap.stateH4,
                                g_f.astro.microMap.stateD1), clrCyan, 9); ay += rowH;
      autoSetLabel(ax, ay, "A2",
                   StringFormat("Bias %+.1f (%s) | Size x%.2f",
                                g_f.astro.microMap.compositeBias,
                                g_f.astro.microMap.rating,
                                g_f.astro.sizingMultiplier), clrLimeGreen, 9, true); ay += rowH;
      autoSetLabel(ax, ay, "A3",
                   StringFormat("Pada #%03d %s P%d Lord %s",
                                g_f.astro.moonTotalPada, g_f.astro.moonNakName,
                                g_f.astro.moonPada, g_f.astro.microMap.padaLord),
                   clrSpringGreen, 9); ay += rowH;
      autoSetLabel(ax, ay, "A4",
                   StringFormat("Hora %s (%dm) Tara %s",
                                g_f.astro.horaName, g_f.astro.horaMinRemaining,
                                g_f.astro.taraBala), clrWhite, 9); ay += rowH;
      autoSetLabel(ax, ay, "A5", "Demon: " + g_f.astro.demonStatus,
                   (g_f.astro.isRahuKalam || g_f.astro.isYamaganda ? clrRed : clrLimeGreen), 9); ay += rowH;
      autoSetLabel(ax, ay, "A6",
                   StringFormat("Tithi %d %s | %s",
                                g_f.astro.tithi, g_f.astro.tithiName, g_f.astro.paksha),
                   clrGold, 9); ay += rowH;
      autoSetLabel(ax, ay, "A7",
                   StringFormat("Vim %s (%.1fy) | Shod %s",
                                g_f.astro.vimshottariLord, g_f.astro.vimshottariBalance,
                                g_f.astro.shodshottariLord), clrPlum, 9); ay += rowH;
      autoSetLabel(ax, ay, "A8",
                   StringFormat("Fear %.0f/100 | Gand %d Apoc %d",
                                g_f.astro.fearIndex, (int)g_f.astro.isGandanta,
                                (int)g_f.astro.isApocalypseTrigger),
                   (g_f.astro.fearIndex >= 50 ? clrOrangeRed : clrDeepSkyBlue), 9); ay += rowH;
      autoSetLabel(ax, ay, "A9", g_f.astro.saturnAspectAlert, clrOrange, 9);
     }
   ChartRedraw(0);
  }

void LogBar(const string msg)
  {
   if(InpLogEveryBar) Print("IES: ", msg);
   g_block = msg;
   Panel();
  }

//==============================================================
// Position management
//==============================================================
void newSLBE(const ulong ticket, const double be, const double tp)
  {
   if(!PositionSelectByTicket(ticket)) return;
   double bid = SymbolInfoDouble(g_sym, SYMBOL_BID);
   double ask = SymbolInfoDouble(g_sym, SYMBOL_ASK);
   bool isBuy = (PositionGetInteger(POSITION_TYPE) == POSITION_TYPE_BUY);
   double pt = g_br.point;
   double minStop = (g_br.stopsLevel + 2) * pt;
   double px = (isBuy ? bid : ask);
   if(isBuy && (px - be) < minStop) return;
   if(!isBuy && (be - px) < minStop) return;
   IES_ModifyPosition(ticket, NormalizeDouble(be, g_br.digits), tp);
  }

void ManagePositions()
  {
   double atr = g_f.atr; if(atr <= 0) return;
   double pt = g_br.point;
   long stopsLvl = g_br.stopsLevel;
   long freezeLvl = g_br.freezeLevel;
   double minStop = (stopsLvl + 2) * pt;

   for(int i = PositionsTotal()-1; i >= 0; i--)
     {
      ulong ticket = PositionGetTicket(i);
      if(ticket == 0 || !PositionSelectByTicket(ticket)) continue;
      if(PositionGetString(POSITION_SYMBOL) != g_sym) continue;
      if((long)PositionGetInteger(POSITION_MAGIC) != InpMagic) continue;

      double open = PositionGetDouble(POSITION_PRICE_OPEN);
      double sl   = PositionGetDouble(POSITION_SL);
      double tp   = PositionGetDouble(POSITION_TP);
      double vol  = PositionGetDouble(POSITION_VOLUME);
      double bid  = SymbolInfoDouble(g_sym, SYMBOL_BID);
      double ask  = SymbolInfoDouble(g_sym, SYMBOL_ASK);
      bool isBuy  = (PositionGetInteger(POSITION_TYPE) == POSITION_TYPE_BUY);

      double rDist = IES_GetInitialR(ticket, open, sl);
      if(rDist <= 0) continue;

      double prog = (isBuy ? bid - open : open - ask);
      double r    = prog / rDist;

      double tp1Target = (isBuy ? open + rDist * InpRR1 : open - rDist * InpRR1);
      double tp2Target = (isBuy ? open + rDist * InpRR2 : open - rDist * InpRR2);
      double tp3Target = (isBuy ? open + rDist * InpRR3 : open - rDist * InpRR3);

      double volMin = g_br.volMin;
      double step   = g_br.volStep;

      if(InpUsePartials && !IES_WasPartialed(ticket))
        {
         bool hit1 = (isBuy ? bid >= tp1Target : ask <= tp1Target);
         if(hit1)
           {
            double pVol = MathFloor((vol * (InpPartial1Pct/100.0)) / step) * step;
            if(pVol >= volMin && (vol - pVol) >= volMin)
              {
               if(IES_ClosePartial(ticket, pVol))
                 {
                  IES_MarkPartialed(ticket);
                  PrintFormat("Partial1 %.2f lots closed @ %.5f for %I64u", pVol, tp1Target, ticket);
                  if(InpStepBrokerTP)
                     IES_ModifyPosition(ticket, sl, NormalizeDouble(tp2Target, g_br.digits));
                 }
              }
           }
        }

      string keyP2 = IES_KeyPartial(ticket) + "_2";
      bool didP2 = GlobalVariableCheck(keyP2);
      if(InpUsePartials && IES_WasPartialed(ticket) && !didP2 && r >= InpRR2)
        {
         double pVol2 = MathFloor((vol * (InpPartial2Pct/100.0)) / step) * step;
         if(pVol2 >= volMin && (vol - pVol2) >= volMin)
           {
            if(IES_ClosePartial(ticket, pVol2))
              {
               GlobalVariableSet(keyP2, 1.0);
               PrintFormat("Partial2 %.2f lots closed @ %.5f for %I64u", pVol2, tp2Target, ticket);
               if(InpStepBrokerTP)
                  IES_ModifyPosition(ticket, sl, NormalizeDouble(tp3Target, g_br.digits));
              }
           }
        }

      if(InpBEAfterTP1 && r >= InpRR1)
        {
         double mpp = g_br.MoneyPerPoint();
         double costPrice = 0;
         if(mpp > 0) costPrice = g_commissionPerLot / mpp;
         double buffer = IES_Max(InpBEPlusCostPts * pt, minStop * 0.25) + costPrice;
         double be = (isBuy ? open + buffer : open - buffer);
         if(isBuy && sl < be - pt) newSLBE(ticket, be, tp);
         if(!isBuy && (sl == 0 || sl > be + pt)) newSLBE(ticket, be, tp);
        }

      if(InpTrailStartR > 0 && r >= InpTrailStartR)
        {
         double trail = sl;
         if(InpTrailMode == TRAIL_ATR)
            trail = (isBuy ? bid - InpTrailAtrMult * atr : ask + InpTrailAtrMult * atr);
         else if(InpTrailMode == TRAIL_SWING_STRUCTURE)
           {
            if(isBuy && g_f.swingLow > 0) trail = g_f.swingLow - minStop;
            else if(!isBuy && g_f.swingHigh > 0) trail = g_f.swingHigh + minStop;
           }
         else if(InpTrailMode == TRAIL_VWAP)
           {
            if(isBuy && g_f.vwapLower1 > 0) trail = g_f.vwapLower1;
            else if(!isBuy && g_f.vwapUpper1 > 0) trail = g_f.vwapUpper1;
           }

         double newSL = sl;
         if(isBuy && trail > newSL + pt) newSL = trail;
         if(!isBuy && (newSL == 0 || trail < newSL - pt)) newSL = trail;
         newSL = NormalizeDouble(newSL, g_br.digits);
         sl = NormalizeDouble(sl, g_br.digits);
         if(newSL != sl)
           {
            double px = (isBuy ? bid : ask);
            if(!(freezeLvl > 0 && MathAbs(px - open) <= freezeLvl*pt))
              if((isBuy && (px - newSL) >= minStop) || (!isBuy && (newSL - px) >= minStop))
                 IES_ModifyPosition(ticket, newSL, tp);
           }
        }
     }
  }

//==============================================================
// Build features pipeline
//==============================================================
bool BuildFeatures()
  {
   ZeroMemory(g_f);
   g_f.regime = -1;
   g_f.session = IES_SESSION_OFF_HOURS;

   if(!g_ind.Update(g_f)) return(false);

   IES_VolumeFeatures(g_sym, InpTimeframe, g_p, g_f);
   IES_ChopLinreg(g_sym, InpTimeframe, g_p, g_f);
   IES_Swings(g_sym, InpTimeframe, g_p, g_f);
   IES_MarketStructure(g_sym, InpTimeframe, g_p, g_f);
   IES_Liquidity(g_sym, InpTimeframe, g_p, g_f);
   IES_FVG(g_sym, InpTimeframe, g_p, g_f);
   IES_OrderBlocks(g_sym, InpTimeframe, g_p, g_f);
   IES_Pivots(g_sym, g_f);
   IES_Fib(g_sym, InpTimeframe, g_p, g_f);
   IES_MarnieFib(g_sym, InpTimeframe, g_p, g_f);
   IES_Confluence(g_sym, InpTimeframe, g_p, g_f);
   IES_CandleIntel(g_sym, InpTimeframe, g_p, g_f);
   g_ind.UpdateMTF(g_f);

   bool overlap, weekend;
   g_f.session = g_sess.CurrentSession(overlap, weekend);
   g_f.isOverlap = overlap; g_f.isWeekend = weekend;
   g_f.todBucket = g_sess.TodBucket();

   IES_SessionLevels(g_sym, InpTimeframe, g_p, g_f);
   IES_Regime(g_sym, InpTimeframe, g_p, g_f);

   g_news.Update(g_f);

   if(InpEnableAstro)
     {
      g_astro.Update(TimeCurrent());
      g_f.astro = g_astro.GetContext();
      g_f.astro.sizingMultiplier = g_astro.GetSizingMultiplier(g_sym);
     }

   g_score = ConfluenceScore(g_f, g_drv);
   DrawSMCChartObjects(g_f);
   return(true);
  }

void AssignTier()
  {
   double a = MathAbs(g_score);
   if(a >= InpAplusScore)      { g_f.tier = TIER_APLUS; g_f.tierReason = "A+"; }
   else if(a >= InpATierScore) { g_f.tier = TIER_A;     g_f.tierReason = "A"; }
   else if(a >= InpBTierScore) { g_f.tier = TIER_B;     g_f.tierReason = "B"; }
   else if(a >= InpCTierScore) { g_f.tier = TIER_C;     g_f.tierReason = "C"; }
   else if(a >= InpWatchScore) { g_f.tier = TIER_WATCH; g_f.tierReason = "Watch"; }
   else                         { g_f.tier = TIER_REJECT; g_f.tierReason = "Below minimum"; }
  }

//==============================================================
// Entry logic
//==============================================================
void TryEntry()
  {
   AssignTier();

   if(InpRelaxRange && g_f.tier == TIER_WATCH &&
      (g_f.regime == IES_REGIME_RANGE || g_f.regime == IES_REGIME_MEAN_REVERSION) &&
      MathAbs(g_score) >= InpRelaxRangeScore)
     {
      g_f.tier = TIER_C;
      g_f.tierReason = "C(relaxed-range)";
     }

   if(g_f.tier == TIER_REJECT || g_f.tier == TIER_WATCH)
     {
      if(InpLogScoreParts)
         PrintFormat("IES ScoreParts | total=%.1f | trend=%.1f mtf=%.1f mom=%.1f vol=%.1f smc=%.1f geom=%.1f candle=%.1f macro=%.1f astro=%.1f | rawSum=%.1f | ATR=%.3f ADX=%.1f RSI=%.1f MTF=%d | regime=%s",
                     g_score,
                     g_f.scoreTrend, g_f.scoreMTF, g_f.scoreMomentum, g_f.scoreVolume,
                     g_f.scoreSMC, g_f.scoreGeometry, g_f.scoreCandle, g_f.scoreMacro, g_f.scoreAstro,
                     g_f.scoreTrend + g_f.scoreMTF + g_f.scoreMomentum + g_f.scoreVolume +
                     g_f.scoreSMC + g_f.scoreGeometry + g_f.scoreCandle + g_f.scoreMacro + g_f.scoreAstro,
                     g_f.atr, g_f.adx, g_f.rsi, g_f.mtfScore,
                     IES_RegimeToString(g_f.regime));
      LogBar(StringFormat("STANDBY score=%.1f tier=%s", g_score, g_f.tierReason));
      return;
     }

   int dir = (g_score > 0 ? 1 : -1);

   if(!RegimeAllows(g_f.regime, dir))
     { LogBar("VETO regime " + IES_RegimeToString(g_f.regime)); return; }
   if(g_f.isWeekend) { LogBar("VETO weekend"); return; }
   if(g_f.newsRisk)  { LogBar("VETO news " + g_f.newsEventName); return; }
   if(InpUseSessionFilter)
     {
      string tok = "," + IES_SessionToString(g_f.session) + ",";
      string hay = "," + InpSessions + ",";
      StringReplace(hay, " ", "");
      if(StringFind(hay, tok) < 0) { LogBar("VETO session " + IES_SessionToString(g_f.session)); return; }
     }
   if(InpUseTodFilter && g_f.todBucket < 0) { LogBar("VETO ToD"); return; }
   if(InpStrictMtfFilter)
     {
      if((dir > 0 && g_f.mtfStates[2] < 0) || (dir < 0 && g_f.mtfStates[2] > 0))
        { LogBar("VETO MTF H1"); return; }
     }
   if(InpEnableAstro && InpFilterDemonHours && g_astro.IsDemonHourActive())
     { LogBar("VETO Demon Hour"); return; }
   if(InpEnableAstro && InpFilterGandanta && g_astro.IsGandantaActive())
     { LogBar("VETO Gandanta"); return; }
   if(InpFilterAbsorption && g_f.absorption)
     {
      double r1 = iHigh(g_sym, InpTimeframe, 1) - iLow(g_sym, InpTimeframe, 1);
      if(dir > 0 && g_f.candleUpWick >= 0.40*r1 && g_f.clv < 0.40) { LogBar("VETO absorption"); return; }
      if(dir < 0 && g_f.candleLoWick >= 0.40*r1 && g_f.clv > 0.60) { LogBar("VETO absorption"); return; }
     }
   if(InpFilterRangeNoise && g_f.linR2 < 0.18 && g_f.chop > g_p.chopRange &&
      g_f.regime == IES_REGIME_RANGE)
     { LogBar("VETO range noise"); return; }
   if(InpDriverVeto && (g_drv.Live() || g_drv.ManualOverlay()) && g_drv.Bias() * dir < -0.35)
     { LogBar("VETO macro driver"); return; }

   if(IES_FindOurPosition() != 0) { LogBar("VETO already in position"); return; }
   if(g_f.atr <= 0) { LogBar("VETO ATR unset"); return; }
   if(!MQLInfoInteger(MQL_TRADE_ALLOWED) || !TerminalInfoInteger(TERMINAL_TRADE_ALLOWED))
     { LogBar("VETO AutoTrading off"); return; }
   if(!AccountInfoInteger(ACCOUNT_TRADE_ALLOWED)) { LogBar("VETO account"); return; }
   if(g_br.tradeMode == SYMBOL_TRADE_MODE_DISABLED) { LogBar("VETO sym disabled"); return; }

   MqlTick tick;
   if(!SymbolInfoTick(g_sym, tick)) { LogBar("VETO no tick"); return; }
   double spreadPts = (tick.ask - tick.bid) / g_br.point;
   double spreadAtr = (g_f.atr > 0 ? (tick.ask - tick.bid) / g_f.atr : 0);

   if(InpMaxSpreadUSD > 0 && (tick.ask - tick.bid) > InpMaxSpreadUSD)
     { LogBar(StringFormat("VETO spread %.2f > %.2f", tick.ask - tick.bid, InpMaxSpreadUSD)); return; }
   if(InpMaxSpreadAtrFrac > 0 && spreadAtr > InpMaxSpreadAtrFrac)
     { LogBar(StringFormat("VETO spread/ATR %.2f > %.2f", spreadAtr, InpMaxSpreadAtrFrac)); return; }

   STradePlan plan;
   BuildTradePlan(dir, plan, g_br, (tick.ask - tick.bid),
                  g_commissionPerLot, InpSlippageAllowPts,
                  g_br.SwapPerLotPerNight(), InpAllowMinLot);
   if(!plan.valid) { LogBar("VETO plan: " + plan.reject); return; }

   if(InpAvoidNegativeSwap && !g_br.IsSwapFree())
     {
      MqlDateTime dt; TimeToStruct(TimeTradeServer(), dt);
      if(dt.hour >= 20 && g_br.SwapPerLotPerNight() < 0)
        { LogBar("VETO adverse overnight swap"); return; }
     }

   if(InpEnableAstro && InpAstroSizing)
     {
      double m = g_astro.GetSizingMultiplier(g_sym);
      plan.lots = IES_NormLot(g_sym, plan.lots * m);
      if(plan.lots < g_br.volMin && InpAllowMinLot) plan.lots = g_br.volMin;
      if(plan.lots <= 0) { LogBar("VETO astro sizing"); return; }
     }

   string why;
   if(!g_cp.CanOpen(why, spreadPts, tick.ask, tick.bid, plan.lots,
                    (dir > 0 ? ORDER_TYPE_BUY : ORDER_TYPE_SELL)))
     { LogBar("VETO " + why); return; }

   double price = (dir > 0 ? tick.ask : tick.bid);
   double brokerTP = InpStepBrokerTP ? plan.tp1 : plan.tp3;

   bool ok = IES_SendDeal((dir > 0 ? ORDER_TYPE_BUY : ORDER_TYPE_SELL),
                          g_sym, plan.lots, price, plan.sl,
                          brokerTP, "IES");
   if(ok)
     {
      g_cp.RegisterOpen();
      string msg = StringFormat("IES %s %.2f lots @ %.5f | SL %.5f | TP1 %.5f TP2 %.5f TP3 %.5f | Score %.1f Tier %s | StepTP=%s",
                                (dir > 0 ? "LONG" : "SHORT"), plan.lots, price,
                                plan.sl, plan.tp1, plan.tp2, plan.tp3,
                                g_score, g_f.tierReason,
                                (InpStepBrokerTP ? "ON" : "OFF"));
      LogBar(msg);
      if(InpNotifyAlert) Alert(msg);
      if(InpNotifyPush)  SendNotification(msg);
     }
   else LogBar("ORDER FAILED");
  }

void CheckFridayClose()
  {
   if(InpFridayCloseHour <= 0) return;
   MqlDateTime dt; TimeToStruct(TimeTradeServer(), dt);
   if(dt.day_of_week == 5 && dt.hour >= InpFridayCloseHour)
     {
      for(int i = PositionsTotal()-1; i >= 0; i--)
        {
         ulong t = PositionGetTicket(i);
         if(t == 0 || !PositionSelectByTicket(t)) continue;
         if(PositionGetString(POSITION_SYMBOL) != g_sym) continue;
         if((long)PositionGetInteger(POSITION_MAGIC) != InpMagic) continue;
         IES_ClosePosition(t);
        }
     }
  }

//==============================================================
// Lifecycle handlers
//==============================================================
int OnInit()
  {
   CleanChartObjects();          // v1.00: purge stale HUD/SMC from any window BEFORE anything else

   g_sym = _Symbol;
   if(!g_br.Init(g_sym))
     { Print("FATAL: broker init failed"); return(INIT_FAILED); }

   g_sess.Init(g_br.gmtOffsetHours);
   IES_DefaultParams(g_p);
   g_p.newsWindowBeforeMin = InpNewsWindowBefore;
   g_p.newsWindowAfterMin  = InpNewsWindowAfter;

   g_ind.Init(g_sym, InpTimeframe, g_p);
   if(!g_ind.Ok()) { Print("FATAL: indicator init"); return(INIT_FAILED); }

   g_cp.Init(g_sym, InpMagic, InpDayLossPct, InpWeekLossPct, InpMaxDDPct,
             InpEquityFloor, InpRiskPerTrade, InpMaxTradesDay, InpMaxConsecLosses,
             InpCooldownMin, InpMinMarginLvl, InpProfitLockPct, InpRecoveryFactor);

   g_drv.Init(g_sym, InpEnableDrivers, InpDxySymbol, InpVixSymbol, InpBtcSymbol,
              InpOilSymbol, InpDriverMomBars, InpReal10y, InpFedCtx, InpCotNetChg,
              "");

   g_news.Init(g_sym, InpNewsImpact, InpNewsWindowBefore, InpNewsWindowAfter);
   if(InpEnableAstro) g_astro.Init(g_sym, InpAstroAyanamsa);
   if(InpResumeOnInit) g_cp.Resume();  // v1.00: FULL state reset (peak, dayStart, all counters)

   g_commissionPerLot = g_br.CommissionPerLotRoundTurn();

   g_ready = false; g_lastBar = 0; g_block = "Warming up...";
   EventSetTimer(5);
   Panel();
   PrintFormat("Predict-A-Trade XAUUSD IES v1.00 init OK | %s | digits=%d point=%.5f tickSize=%.5f tickVal=%.5f volMin=%.2f volStep=%.2f stopsLvl=%d gmtOffset=%d",
               g_sym, g_br.digits, g_br.point, g_br.tickSize, g_br.tickValue,
               g_br.volMin, g_br.volStep, (int)g_br.stopsLevel, g_br.gmtOffsetHours);
   PrintFormat("IES Config: RR=%.1f/%.1f/%.1f SLatr=%.2f MaxSlAtr=%.2f MaxSlUSD=%.2f Partial1=%.0f%% Partial2=%.0f%% TrailStartR=%.2f StepTP=%s | RelaxRange=%s(>=%.1f) | MaxTrades=%d MaxLossStreak=%d Cooldown=%dm ProfitLock=%.1f%%",
               InpRR1, InpRR2, InpRR3, InpSLatrMult, InpMaxSlAtr, InpMaxSlUSD,
               InpPartial1Pct, InpPartial2Pct, InpTrailStartR,
               (InpStepBrokerTP ? "ON" : "OFF"),
               (InpRelaxRange ? "ON" : "OFF"), InpRelaxRangeScore,
               InpMaxTradesDay, InpMaxConsecLosses, InpCooldownMin, InpProfitLockPct);
   return(INIT_SUCCEEDED);
  }

void OnDeinit(const int reason)
  {
   EventKillTimer();
   g_ind.Release();
   CleanChartObjects();
   Comment("");
   Print("Predict-A-Trade XAUUSD IES v1.00 deinit: ", reason);
  }

void OnTimer()
  {
   g_cp.Update();

   static int lastFetch = -1;
   MqlDateTime t; TimeToStruct(TimeCurrent(), t);
   if(t.min % 15 == 0 && t.min != lastFetch)
     { lastFetch = t.min; g_drv.Update(); g_drv.FetchRemote(); }

   if(!g_ready)
     {
      if(BuildFeatures())
        {
         g_ready = true;
         AssignTier();
         PrintFormat("IES sync ATR=%.5f Score=%.1f Session=%s",
                     g_f.atr, g_score, IES_SessionToString(g_f.session));
        }
      else
        {
         g_warmFails++;
         g_block = StringFormat("Warming up retry #%d", g_warmFails);
        }
     }

   if(InpHeartbeat)
     {
      static datetime last = 0;
      if(TimeCurrent() - last >= 30)
        {
         last = TimeCurrent();
         double pt = g_br.point > 0 ? g_br.point : _Point;
         double spr = pt > 0 ? (SymbolInfoDouble(g_sym,SYMBOL_ASK) - SymbolInfoDouble(g_sym,SYMBOL_BID)) / pt : 0;
         PrintFormat("IES Beat ready=%d score=%.1f session=%s spread=%.0f atr=%.3f status=%s",
                     (int)g_ready, g_score, IES_SessionToString(g_f.session),
                     spr, g_f.atr, g_block);
        }
     }

   Panel();
  }

void OnTick()
  {
   ManagePositions();
   CheckFridayClose();

   if(!g_ready)
     {
      if(!BuildFeatures()) return;
      g_ready = true;
     }

   datetime bt = iTime(g_sym, InpTimeframe, 0);
   if(bt == 0) return;

   if(InpNewBarOnly)
     {
      if(bt == g_lastBar) return;
      if(!BuildFeatures()) { Panel(); return; }
      g_lastBar = bt;
      TryEntry();
     }
   else
     {
      if(IES_FindOurPosition() == 0)
        {
         if(BuildFeatures()) TryEntry();
        }
      g_lastBar = bt;
     }
  }

void OnTradeTransaction(const MqlTradeTransaction &trans,
                        const MqlTradeRequest &request,
                        const MqlTradeResult &result)
  {
   if(trans.type != TRADE_TRANSACTION_DEAL_ADD) return;
   ulong deal = trans.deal;
   if(!HistoryDealSelect(deal)) return;
   if(HistoryDealGetString(deal, DEAL_SYMBOL) != g_sym) return;
   if((long)HistoryDealGetInteger(deal, DEAL_MAGIC) != InpMagic) return;

   ENUM_DEAL_ENTRY entry = (ENUM_DEAL_ENTRY)HistoryDealGetInteger(deal, DEAL_ENTRY);
   if(entry == DEAL_ENTRY_OUT || entry == DEAL_ENTRY_INOUT)
     {
      double pnl = HistoryDealGetDouble(deal, DEAL_PROFIT)
                 + HistoryDealGetDouble(deal, DEAL_SWAP)
                 + HistoryDealGetDouble(deal, DEAL_COMMISSION);
      g_cp.RegisterClose(pnl);
      ulong posId = (ulong)HistoryDealGetInteger(deal, DEAL_POSITION_ID);
      if(!PositionSelectByTicket(posId)) IES_CleanGlobals(posId);
     }
  }
//+------------------------------------------------------------------+
// END OF FILE
//+------------------------------------------------------------------+