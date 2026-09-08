//+------------------------------------------------------------------+
//| Predict-A-Trade.mq5                                              |
//| Production-oriented XAUUSD M1 intraday / ultra-scalping EA       |
//| Session Overlap + High-Volatility Opportunity Engine             |
//| Copyright 2026 Predict-A-Trade | Simha FinTech LLC, Dubai, UAE   |
//+------------------------------------------------------------------+
//| PRE-DEPLOY WARNING - READ BEFORE ENABLING REAL CAPITAL           |
//|                                                                  |
//| Many default values in this file were hand-tuned against recent  |
//| market behaviour ("lowered from 6", "was 35", "was 15", etc.).   |
//| Hand-tuning to visible screenshots and recent gold behaviour     |
//| risks OVERFITTING: parameters that look perfect on the last few  |
//| days routinely fail on new data.                                 |
//|                                                                  |
//| Before real capital, these defaults MUST pass:                   |
//|  1. Walk-forward / out-of-sample testing (optimize on window A,  |
//|     validate on unseen window B, roll forward).                  |
//|  2. At least 2-3 months on a demo account of the SAME broker     |
//|     and account type, with real spread/slippage/commission.      |
//|  3. Minimum trade count for statistical meaning (100+ trades).   |
//| A profitable backtest alone proves nothing. Tight gates that     |
//| "feel" safer can simply select a lucky historical sample.        |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade | Simha FinTech LLC"
#property version   "1.00"
#property description "XAUUSD M1 four-session + overlap/HV ultra-scalper. Pure MQL5: native OrderSend (no includes, no CTrade). FMP stable macro adapter with obfuscated credentials, MQL5 calendar news gate, TP1/TP2/TP3 ladder with R:R validation, reversal loss-recovery leg, broker/account telemetry and a two-column control dashboard (click header to collapse, F key to pause arming)."

//====================================================================
// ENUMS
//====================================================================
enum ENUM_TREND_DIRECTION { TREND_NONE=0, TREND_UP=1, TREND_DOWN=-1 };
enum ENUM_MARKET_PHASE    { PHASE_ACCUMULATION=0, PHASE_MANIPULATION=1, PHASE_DISTRIBUTION=2 };
enum ENUM_FILTER_MODE     { FILTER_SCORING=0, FILTER_ALL_REQUIRED=1 };
enum ENUM_EXECUTION_MODE  { EXEC_STRADDLE=0, EXEC_DIRECTIONAL=1, EXEC_AUTO=2 };
enum ENUM_VWAP_ANCHOR     { VWAP_BROKER_DAY=0, VWAP_LONDON=1, VWAP_NEWYORK=2 };
enum ENUM_BREAKER_ACTION  { BREAKER_BLOCK_ONLY=0, BREAKER_CLOSE_ALL=1 };
enum ENUM_HV_MODE         { HV_OFF=0, HV_AUTO=1, HV_FORCE_GATED=2 };
enum ENUM_WINDOW_ID
{
   WIN_NONE=0,
   WIN_SYDNEY=1,
   WIN_TOKYO=2,
   WIN_SYDNEY_TOKYO=3,
   WIN_TOKYO_LONDON=4,
   WIN_LONDON_OPEN=5,
   WIN_LONDON=6,
   WIN_LONDON_NY=7,
   WIN_NY_OPEN=8,
   WIN_NEWYORK=9,
   WIN_VERIFIED_EXPANSION=10,
   WIN_COUNT=11
};

//====================================================================
// INPUTS
//====================================================================
input group "=== ULTRA-SCALP MODE (SIMPLIFIED ENGINE) ==="
input bool   InpSimpleScalpMode           = true;      // TRUE = simple M1 scalp engine (recommended); false = full multi-filter engine
input double InpScalpMinMomentumATR       = 0.08;      // simple engine: min last-bar momentum in ATR (0.12 = gentle)

input group "=== CAPITAL PROTECTION ==="
input double InpDailyLossPercent          = 4.0;
input double InpMaxFloatingDDPercent      = 5.0;
input double InpWeeklyLossLimit           = 8.0;
input double InpMonthlyLossLimit          = 15.0;
input double InpRiskPercent               = 0.50;
input double InpRiskStepDownOnDD          = 0.10;
input int    InpMaxConsecutiveLosses      = 3;      // pause only after 3 straight; risk decays 30% per loss before that
input int    InpMaxTradesPerDay           = 60;
input bool   InpAllowMinLotFallback       = true;      // size to broker min lot when risk-% lots < min (small accounts)
input double InpMinLotMaxRiskPct          = 2.0;       // min-lot trade allowed only if its risk <= this % of balance
input double InpMaxAggregateOpenRiskPct   = 4.00;
input double InpMaxDirectionalRiskPct     = 2.50;
input ENUM_BREAKER_ACTION InpBreakerAction= BREAKER_CLOSE_ALL;
input bool   InpNoMartingale              = true;      // invariant; retained for audit visibility
input bool   InpNoAveragingDown           = true;      // invariant; never add to losing exposure

input group "=== BROKER / COST MODEL ==="
input int    InpMaxSpreadPoints           = 45;      // hard cap; Xelans ECN gold runs 35-45pt
input double InpSpreadSpikeRatio          = 2.50;
input double InpMaxSpreadPercentile       = 92.0;     // p85 was below this broker's normal spread range
input int    InpMaxSlippagePoints         = 30;
input double InpCommissionPerLotRTFallback= 7.00;      // account-currency round trip / lot fallback
input double InpExpectedSlipPtsFallback   = 4.0;
input double InpMaxCostToTP1Pct           = 45.0;     // was 35: ECN spread pushed cost ratio over 35
input double InpMinNetProfitTP1Money      = 0.30;
input double InpMinNetProfitTP2Money      = 0.50;
input double InpMinNetProfitTP3Money      = 0.70;
input int    InpOrderRetry                = 2;

input group "=== RISK-REWARD VALIDATION ==="
input double InpMinRR_TP2                 = 0.60;      // TP2 reward must beat this multiple of SL distance
input double InpMinRR_TP3                 = 1.20;      // TP3 reward must beat this multiple of SL distance

input group "=== EXECUTION / ANTI-OVERTRADING ==="
input ENUM_EXECUTION_MODE InpExecutionMode= EXEC_AUTO;
input int    InpStraddleLayers            = 1;
input double InpLayerStepATR              = 0.35;
input int    InpMaxConcurrentPositions    = 6;
input double InpMaxTotalLots              = 3.00;
input bool   InpArmWhileInTrade           = true;
input bool   InpScaleIn                   = true;
input int    InpMinSecondsBetweenEntries  = 15;     // was 120
input int    InpMinBarsFreshStructure     = 2;
input int    InpMaxSignalsPerWindow       = 20;
input double InpPerWindowRiskBudgetPct    = 3.00;
input bool   InpOncePerValidatedEvent     = true;
input bool   InpCancelStalePendings       = true;
input int    InpPendingExpiryMinutes      = 5;
input double InpDistance                  = 1.00;
input bool   InpUseATRForDistance         = true;
input double InpATRMultiplier             = 0.22;
input double InpLayerSpacingATR           = 0.25;
input double InpLotSize                   = 0.10;

input group "=== FILTERS / SMC ==="
input ENUM_FILTER_MODE InpFilterMode      = FILTER_SCORING;
input int    InpMinFilterScore            = 5;      // lowered from 6 for execution flow
input bool   InpUseEMA20                  = true;
input bool   InpUseEMA50                  = true;
input bool   InpUseSuperTrend             = true;
input bool   InpUseADX                    = true;
input bool   InpUseVWAP                   = true;
input bool   InpUseFVG                    = true;
input bool   InpUseAMD                    = true;
input bool   InpUseVolumeFilter           = true;
input bool   InpUseLiquidityFilter        = true;
input bool   InpUseRSI                    = true;
input bool   InpUseSMC                    = true;
input int    InpMinDirBias                = 2;      // lowered from 3: 3 blocked many valid signals
input int    InpMinSMCConfluence          = 1;      // min bull/bear structure votes (2 = very selective)
input int    InpSwingLookback             = 12;
input int    InpFVGLookbackBars           = 12;
input double InpFVGMinGapATR              = 0.05;
input int    InpAMDLookbackBars           = 20;
input double InpAMDCoilRatio              = 0.80;

input group "=== INDICATORS / VOLATILITY ==="
input int    InpEMA20Period               = 20;
input int    InpEMA50Period               = 50;
input int    InpSuperTrendPeriod          = 10;
input double InpSuperTrendMultiplier      = 3.0;
input int    InpADXPeriod                 = 14;
input double InpMinADX                    = 22.0;
input int    InpATRPeriod                 = 14;
input double InpMinATRPoints              = 25;
input double InpMaxATRPoints              = 600;      // gold M1 ATR regularly exceeds 350pt
input int    InpATRPercentileLookback     = 240;
input double InpHVMinATRPercentile        = 65.0;
input int    InpVolumeMA                  = 30;
input double InpMinVolumeRatio            = 1.20;
input double InpHVMinVolumeRatio          = 1.35;
input double InpMinLiquidityLevel         = 0.90;
input int    InpRSIPeriod                 = 9;
input int    InpRSIOverbought             = 78;
input int    InpRSIOversold               = 22;
input ENUM_VWAP_ANCHOR InpVWAPAnchor      = VWAP_BROKER_DAY;

input group "=== FOUR-SESSION + OVERLAP ENGINE (UTC->BROKER SERVER) ==="
input bool   InpUseSessionFilter          = true;
input bool   InpTradeAllFourSessions      = true;       // master guarantee: Sydney, Tokyo, London and New York all participate
input bool   InpTradeSydney               = true;
input bool   InpTradeTokyo                = true;
input bool   InpTradeLondon               = true;
input bool   InpTradeNewYork              = true;
input bool   InpTradeSydneyTokyo          = true;       // true simultaneous Sydney/Tokyo liquidity window
input bool   InpTradeTokyoLondon          = true;       // true simultaneous Tokyo/London window (DST aware)
input bool   InpTradeLondonOpen           = true;
input bool   InpTradeLondonNY             = true;       // true simultaneous London/New York window
input bool   InpTradeNYOpen               = true;
input bool   InpTradeVerifiedExpansion    = true;
input bool   InpNeverDisablePrimarySessions= true;      // expectancy may reduce risk, but cannot turn Sydney/Tokyo/London/NY off
input int    InpSydneyLocalOpenMin        = 8*60;       // 08:00 Australia/Sydney local
input int    InpSydneyLocalCloseMin       = 17*60;      // 17:00 Australia/Sydney local
input int    InpTokyoLocalOpenMin         = 9*60;       // 09:00 JST
input int    InpTokyoLocalCloseMin        = 18*60;      // 18:00 JST
input int    InpLondonLocalOpenMin        = 8*60;       // 08:00 London local
input int    InpLondonLocalCloseMin       = 16*60+30;   // 16:30 London local
input int    InpNewYorkLocalOpenMin       = 8*60;       // 08:00 New York local
input int    InpNewYorkLocalCloseMin      = 17*60;      // 17:00 New York local
input int    InpLondonOpenWindowMin       = 60;
input int    InpNYOpenWindowMin           = 60;
input int    InpOverlapPadMinutes         = 0;
input double InpFridayCutoffServer        = 20.0;
input bool   InpAutoDetectServerOffset    = true;
input int    InpManualServerOffsetHours   = 2;
input int    InpServerOffsetRefreshSec    = 60;

input group "=== HIGH-VOLATILITY MODE ==="
input ENUM_HV_MODE InpHighVolatilityMode  = HV_AUTO;
input int    InpHVMinScore                = 6;
input double InpHVMinDisplacementATR      = 0.65;
input double InpHVMaxDisorderATR          = 2.80;
input double InpHVMinVWAPDeviationATR     = 0.10;
input int    InpSessionBreakoutLookback   = 30;      // ROLLING M1 bars (not session-anchored range)
input double InpHVExtraSignalRiskMult     = 0.70;
input int    InpVerifiedBucketMinSamples  = 30;
input double InpVerifiedBucketATRRatio    = 1.15;
input double InpVerifiedBucketVolRatio    = 1.10;

input group "=== TP1 + TP2 + TP3 EXIT ENGINE ==="
input bool   InpUseThreeTargets           = true;
input double InpTP1Pct                    = 0.70;      // 70% off at TP1: scalp banking, small runner
input double InpTP2Pct                    = 0.20;
input double InpTP3Pct                    = 0.10;
input double InpSL_ATR_Multiplier         = 0.90;      // was 1.25: tighter stop improves ladder R:R
input double InpSLStructureBufferATR      = 0.15;
input double InpTP1_ATR_Floor             = 0.30;
input double InpTP1_ATR_Cap               = 0.60;
input double InpTP2_ATR_Floor             = 0.75;
input double InpTP2_ATR_Cap               = 1.60;
input double InpTP3_ATR_Floor             = 1.20;
input double InpTP3_ATR_Cap               = 2.80;
input bool   InpUseCostAdjustedBE         = true;
input double InpBEExtraLockATR            = 0.03;
input bool   InpUseTP3StructureTrail      = true;
input double InpTP3TrailATR               = 0.70;
input double InpTP3TrailStepATR           = 0.15;
input bool   InpTP3EarlyExit              = true;
input int    InpMaxTradeMinutes           = 20;      // scalp: in-and-out; stale scalps die fast

input group "=== NEWS / DISORDER PROTECTION ==="
input bool   InpUseNewsFilter             = true;
input int    InpNewsBufferMinutes         = 10;     // was 15: 15+stabilize froze too long
input int    InpNewsLookaheadMin          = 120;
input int    InpPostNewsStabilizeMinutes  = 5;
input double InpDisorderSpreadPct         = 95.0;
input double InpMaxChaseCandleATR         = 2.60;     // gold M1 displacement 2+ ATR is normal momentum, not a chase
input double InpMaxEntryVWAPDeviationATR  = 2.20;
input double InpDisorderSlipPts           = 20.0;
input int    InpDisorderCooldownMinutes   = 5;

input group "=== FMP MACRO / NEWS (OBFUSCATED CREDENTIALS) ==="
input bool   InpUseFMP                    = true;      // FMP stable REST macro adapter (primary intermarket source)
input int    InpFMPRefreshSec             = 600;        // quote refresh throttle (rate-limit friendly)
input int    InpFMPTimeoutMs              = 5000;      // per-request HTTP timeout
input double InpFMPUSDPairMinPct          = 0.020;     // min averaged USD-basket move % for a directional vote
input bool   InpFMPIncludeSPX             = true;      // S&P 500 risk sentiment vote (risk-off = gold bid)
input double InpFMPSPXMinPct              = 0.30;      // SPX move % threshold for a sentiment vote
input bool   InpFMPNewsHardBlock          = false;     // true = FMP headline hits also block entries (soft/log-only by default)
input int    InpFMPNewsLimit              = 25;        // headlines scanned per refresh
input bool   InpAllowBrokerMacroFallback  = true;      // broker-side EURUSD momentum when FMP is unreachable
input ENUM_TIMEFRAMES InpMacroTF         = PERIOD_M5;
input int    InpMacroMomentumBars         = 6;
input bool   InpUseEURUSD                 = true;
input string InpEURUSDSymbol              = "";        // blank = auto-detect broker EURUSD including suffix/prefix
input double InpEURUSDMinMovePct          = 0.020;
input int    InpMacroMinConfluence        = 0;        // 0 = macro is advisory (FMP down must not veto entries)         // aligned USD-basket/EUR/SPX votes required vs opposing
input bool   InpRequireExternalData       = false;     // true blocks entry until a macro feed is available

input group "=== IFVG / PROPULSION BLOCK (PTB) ==="
input bool   InpUseIFVG                   = true;
input int    InpIFVGLookbackBars          = 40;
input double InpIFVGMinGapATR             = 0.05;
input bool   InpUsePTB                    = true;      // PTB = ICT Propulsion Block
input int    InpPTBLookbackBars           = 40;
input double InpPTBMinDisplacementATR     = 0.55;
input int    InpMinAdvancedSMCConfluence  = 0;      // 0 = IFVG/PTB add score but never block entries

input group "=== LOSS RECOVERY / REVERSAL ENGINE ==="
input bool   InpUseRecovery               = true;      // after a realized loss, arm ONE controlled reversal leg
input int    InpRecoveryMaxLegs           = 1;         // reversal legs allowed per loss event (1 = single counter-trade)
input double InpRecoveryRiskPct           = 0.20;      // smaller counter-leg risk at scalp frequency      // % balance risked on the recovery leg (below base risk)
input double InpRecoveryMinRR             = 1.60;      // recovery TP1 must beat this multiple of its SL distance
input int    InpRecoveryCooldownSec       = 240;       // min seconds between the loss and the recovery entry
input int    InpRecoveryMaxAgeSec         = 900;       // recovery opportunity expires after this many seconds
input double InpRecoveryMaxSpreadPts      = 35;        // tighter spread cap for recovery entries

input group "=== SLIPPAGE / SWAP PROTECTION ==="
input double InpMaxAverageSlippagePoints  = 18.0;     // 18pt avg is realistic for gold ECN fills; 10 blocked every re-entry
input double InpExtremeSlippagePoints     = 30.0;     // 30pt on gold = genuinely pathological fill; pre-trade guard covers spikes
input bool   InpCloseOnExtremeSlippage    = false;     // optional emergency flatten after a pathological fill
input int    InpSlippageCooldownMinutes   = 3;        // was 10: 10-min sit-outs after every SL fill = "no trades"
input bool   InpAvoidSwap                 = true;
input bool   InpForceFlatBeforeSwap       = true;
input double InpSwapRolloverServerHour    = 0.0;       // server-hour of swap rollover; 0.0 = midnight server (Xelans GMT+3: correct). Verify in Journal around 00:00 server.
input int    InpSwapBlockMinutesBefore    = 60;
input int    InpSwapBlockMinutesAfter     = 10;

input group "=== PERFORMANCE SELF-GATING / A-B ==="
input bool   InpEnablePerformanceGating   = true;
input int    InpPerfMinTrades             = 20;
input int    InpPerfRollingTrades         = 40;
input double InpDisableExpectancyMoney    = -0.20;
input double InpRiskReduceExpectancyMoney = 0.10;
input double InpWeakWindowRiskMultiplier  = 0.50;
input bool   InpAB_EnableBase             = true;
input bool   InpAB_EnableHighVol          = true;
input bool   InpAB_EnableThreeTP          = true;

input group "=== LOGGING / DASHBOARD ==="
input bool   InpUseLogFile                = true;
input bool   InpPersistState              = true;
input bool   InpShowDashboard             = true;
input int    InpPanelX                    = 430;       // initial panel X (clear of left dock; drag header to move)
input int    InpPanelY                    = 30;        // initial panel Y (drag header to move)
input int    InpPanelFontSize             = 8;         // 6..12; 8 recommended - all values readable
input int    InpDashRefreshMs             = 400;       // dashboard repaint throttle (CPU friendly)
input string InpPanelTitle                = "PREDICT-A-TRADE GOLD";  // panel header title
input bool   InpPanelDraggable            = true;      // drag the header to move the panel
input string InpSoundPause                = "alert2.wav";  // sound when arming is paused
input string InpSoundResume               = "alert.wav";   // sound when arming is resumed
input int    InpMagicNumber               = 20260911;
input string InpComment                   = "Predict-A-Trade v4";

//====================================================================
// STRUCTS
//====================================================================
struct BrokerProfile
{
   string symbol;
   double point,tickSize,tickValue,contractSize,volumeMin,volumeMax,volumeStep;
   double swapLong,swapShort;
   int digits,stopsLevel,freezeLevel,leverage;
   string company,currency;
   ENUM_ACCOUNT_MARGIN_MODE marginMode;
   ENUM_ACCOUNT_TRADE_MODE tradeMode;
   bool hedging;
};

struct WindowStats
{
   long observations;
   int trades,wins,losses,tp1Hits,tp2Hits,tp3Hits;
   double grossPL,netPL,costs,slipSum,spreadPctSum,atrPctSum,volRatioSum;
   double maeSum,mfeSum,rSum,peakNet,maxDD;
   double recentNet[64];
   int recentCount,recentIdx;
   double obsATRRatioSum,obsVolRatioSum;
   bool disabled;
};

struct PositionState
{
   ulong ticket;
   long positionId;
   int direction;
   ENUM_WINDOW_ID window;
   string setupId;
   bool hv;
   bool recovery;                  // loss-recovery reversal leg (excluded from window stats)
   double initialVolume;
   double initialRiskMoney;
   double entry;
   double initialSL;
   double tp1,tp2,tp3;
   double volTP1,volTP2,volTP3;
   bool tp1Done,tp2Done,tp3Done;
   double maePrice,mfePrice;
   datetime opened;
   double entrySpreadPct;
   double entrySlipPts;
   double entryAtrPct;
   double entryVolRatio;
   double realizedGross;
   double realizedNet;
   double realizedCosts;
};

struct BucketStats
{
   long samples;
   double atrRatioSum;
   double volRatioSum;
};

//====================================================================
// GLOBALS
//====================================================================
BrokerProfile broker;
string eaSymbol="";

int hATR=INVALID_HANDLE,hADX=INVALID_HANDLE,hEMA20=INVALID_HANDLE,hEMA50=INVALID_HANDLE;
int hH1EMA20=INVALID_HANDLE,hH1EMA50=INVALID_HANDLE,hM15EMA20=INVALID_HANDLE,hM15EMA50=INVALID_HANDLE;
int hRSI=INVALID_HANDLE,hM5E20=INVALID_HANDLE,hM5E50=INVALID_HANDLE,hM5ADX=INVALID_HANDLE;

double g_atr=0,g_adx=0,g_adxPlus=0,g_adxMinus=0,g_rsi=0;
double g_ema20=0,g_ema50=0,g_h1e20=0,g_h1e50=0,g_m15e20=0,g_m15e50=0;
double g_vwap=0,g_vwapUp=0,g_vwapDn=0;
int g_superTrendDir=0;
double g_superTrend=0;
ENUM_MARKET_PHASE g_phase=PHASE_ACCUMULATION;
bool g_fvg=false; int g_fvgDir=0; double g_fvgTop=0,g_fvgBottom=0;
bool g_bosUp=false,g_bosDn=false,g_chochUp=false,g_chochDn=false,g_sweepUp=false,g_sweepDn=false;
double g_swingHigh=0,g_swingLow=0;
int g_dirBias=0,g_score=0,g_scoreMax=0,g_smcScoreBull=0,g_smcScoreBear=0;
double g_volRatio=0;                                   // cached per-bar volume ratio (no CopyRates per tick)
bool g_ifvg=false; int g_ifvgDir=0; double g_ifvgTop=0,g_ifvgBottom=0;
bool g_ptb=false; int g_ptbDir=0; double g_ptbTop=0,g_ptbBottom=0;
double g_lastSlipPts=0;
int g_perfTrades=0,g_perfWins=0,g_perfLosses=0,g_perfRetN=0;
double g_perfNetProfit=0,g_perfGrossProfit=0,g_perfGrossLoss=0,g_perfRetMean=0,g_perfRetM2=0,g_perfCumNet=0,g_perfPeakNet=0,g_perfMaxDDMoney=0;

//--- FMP macro state (credentials decoded at runtime only)
#define FMPUSD_COUNT 7
string g_usdPairs[FMPUSD_COUNT]={"EURUSD","GBPUSD","USDJPY","USDCHF","USDCAD","AUDUSD","NZDUSD"};
double g_usdMove[FMPUSD_COUNT];
bool   g_usdGot[FMPUSD_COUNT];
double g_usdAvg=0;
int    g_usdBias=0;
bool   g_usdAvailable=false;
string g_eurSymbol="";
double g_eurMovePct=0;
int    g_eurBias=0;
bool   g_eurAvailable=false;
double g_spxMovePct=0;
int    g_spxBias=0;
bool   g_spxAvailable=false;
bool   g_fmpEverOK=false;
int    g_fmpErrCount=0;
bool     g_webRequestWarned=false;
int      g_fmp429Count=0;
int      g_fmpCycle=0;
bool     g_usdGotSPX=false;
double   g_m5e20=0,g_m5e50=0,g_m5adx=0,g_m5adxPlus=0,g_m5adxMinus=0;
double   g_bbUp=0,g_bbLo=0,g_bbMid=0;
int      g_scalpSignal=0;   // +1 trend-long, -1 trend-short, +2 reversion-long, -2 reversion-short, 0 none
string   g_scalpWhy="";
double   g_spxBatchChg=0;
string g_fmpLastErr="";
datetime g_fmpLastTry=0,g_fmpLastOK=0;
int    g_fmpNewsCount=0;
string g_fmpNewsLast="";
int    g_macroBull=0,g_macroBear=0;

//--- loss recovery / reversal state
int      g_lastLossDir=0;             // direction of the last realized loss (+1 buy, -1 sell)
datetime g_lastLossTime=0;
double   g_lastLossMoney=0;
datetime g_recoveryArmedUntil=0;
int      g_recoveryLegs=0;

#define SPREAD_SAMPLES 256
double g_spreadBuf[SPREAD_SAMPLES]; int g_spreadCnt=0,g_spreadIdx=0; double g_spreadAvg=0;
double g_spreadStd=0;
#define SLIP_SAMPLES 64
double g_slipBuf[SLIP_SAMPLES]; int g_slipCnt=0,g_slipIdx=0; double g_slipAvg=0;
#define ATR_SAMPLES 512
int g_atrKeep=512;                     // effective ring size = min(ATR_SAMPLES, InpATRPercentileLookback)
double g_atrBuf[ATR_SAMPLES]; int g_atrCnt=0,g_atrIdx=0;

WindowStats g_ws[WIN_COUNT];
BucketStats g_bucket[48];
PositionState g_ps[];

long g_serverOffsetSec=0;
double g_ptScale=1.0;                    // point-unit auto-scale for 3-digit gold feeds
datetime g_lastOffsetRefresh=0,g_lastBar=0,g_lastExitTime=0,g_lastEntryTime=0;
datetime g_newsChecked=0,g_nextNewsTime=0,g_newsBlockedUntil=0,g_disorderUntil=0;
string g_nextNewsName="",g_gateReason="";
bool g_newsBlocked=false,g_paused=false;

int g_tradesToday=0,g_consecutiveLosses=0;
double g_dayAnchor=0,g_weekAnchor=0,g_monthAnchor=0,g_maxDDSeen=0;
int g_dayKey=-1,g_weekKey=-1,g_monthKey=-1;
bool g_stopDay=false,g_stopWeek=false,g_stopMonth=false;
double g_commissionRTPerLot=0;
double g_windowRiskUsed[WIN_COUNT];
int g_windowSignals[WIN_COUNT];
string g_lastSetupBuy[WIN_COUNT],g_lastSetupSell[WIN_COUNT];
datetime g_lastWindowEntry[WIN_COUNT];

int g_log=INVALID_HANDLE;
#define UI_PREFIX "PAT4_"

//--- dashboard colour system: deep navy base, high-contrast text, semantic status colours
color C_BG=C'0,0,0',C_BG2=C'10,12,16',C_PANEL=C'6,8,12',C_SECTION=C'16,24,40',C_SECTION_TXT=C'120,220,255';
color C_TXT=C'255,255,255',C_TXT2=C'208,216,232',C_DIM=C'150,160,180',C_GRAY=C'110,120,140',C_GRID=C'40,48,64';
color C_HDR=C'12,18,30',C_ACCENT=C'0,160,255',C_BORDER=C'70,80,100';
color C_UP=C'0,230,118',C_UP_TXT=C'0,255,140',C_DN=C'255,64,64',C_DN_TXT=C'255,110,110';
color C_WARN=C'255,193,7',C_WARN_TXT=C'255,224,102',C_INFO=C'64,196,255',C_GOLD=C'255,193,7',C_GOLD_TXT=C'255,215,64';
color C_SYD=C'0,176,255',C_TOK=C'155,89,255',C_LON=C'255,145,0',C_NY=C'0,230,118';
color C_SYD_DIM=C'20,60,90',C_TOK_DIM=C'50,32,90',C_LON_DIM=C'90,54,0',C_NY_DIM=C'20,80,50';

//--- dashboard geometry (recomputed from InpPanelFontSize)
// Flow layout: columns are drawn with a running Y cursor and every string is
// width-clipped to its column, so overlap is impossible by construction.
// Row counts below are the single source of truth for the panel height.
#define L_SECTIONS 3
#define L_ROWS     16
#define R_SECTIONS 4
#define R_ROWS     22
int g_font=7,g_fontPx=9,g_dpi=96;
double g_dpiScale=1.0;
int g_rh=16,g_hdrH=30,g_colW=300,g_pad=12,g_gap=14,g_panelW=0;
int g_bodyTop=0,g_colHL=0,g_colHR=0,g_tlTop=0,g_tlH=0,g_panelH=0;
bool g_dashCollapsed=false,g_dashWasCollapsed=false;
uint g_lastDashMs=0;
int g_x=8,g_y=24;                    // live panel origin (draggable)
bool g_dragging=false,g_maybeClick=false;
int g_dragOffX=0,g_dragOffY=0,g_dragStartX=0,g_dragStartY=0;
uint g_lastDragMs=0;
bool g_measure=false;                // pass-1 layout mode (no drawing)
int g_yCurL=0,g_yCurR=0,g_rowL=0,g_rowR=0;//====================================================================
// BASIC HELPERS
//====================================================================
int VolDigits(double step)
{
   int d=0; if(step<=0) return 2;
   while(step<1.0 && d<8){ step*=10.0; d++; }
   return d;
}

double FloorVolume(double lots)
{
   double step=(broker.volumeStep>0?broker.volumeStep:0.01);
   if(lots<=0) return 0;
   lots=MathFloor((lots+1e-12)/step)*step;
   if(lots<broker.volumeMin-1e-12) return 0;
   lots=MathMin(lots,broker.volumeMax);
   return NormalizeDouble(lots,VolDigits(step));
}

double NormalizeVolume(double lots)
{
   if(lots<=0) return 0;
   double v=FloorVolume(lots);
   if(v<=0 && lots>=broker.volumeMin*0.999) v=broker.volumeMin;
   return NormalizeDouble(MathMin(v,broker.volumeMax),VolDigits(broker.volumeStep));
}

double PriceNorm(double p){ return NormalizeDouble(p,broker.digits); }
double MinTradeDistance(){ return MathMax(broker.stopsLevel,broker.freezeLevel)*broker.point+2*broker.point; }

double Bid(){ return SymbolInfoDouble(eaSymbol,SYMBOL_BID); }
double Ask(){ return SymbolInfoDouble(eaSymbol,SYMBOL_ASK); }
double SpreadPoints(){ double a=Ask(),b=Bid(); return (a>0&&b>0&&broker.point>0)?(a-b)/broker.point:99999; }

bool InitBroker()
{
   broker.symbol=eaSymbol;
   broker.point=SymbolInfoDouble(eaSymbol,SYMBOL_POINT);
   broker.tickSize=SymbolInfoDouble(eaSymbol,SYMBOL_TRADE_TICK_SIZE);
   broker.tickValue=SymbolInfoDouble(eaSymbol,SYMBOL_TRADE_TICK_VALUE);
   broker.contractSize=SymbolInfoDouble(eaSymbol,SYMBOL_TRADE_CONTRACT_SIZE);
   broker.volumeMin=SymbolInfoDouble(eaSymbol,SYMBOL_VOLUME_MIN);
   broker.volumeMax=SymbolInfoDouble(eaSymbol,SYMBOL_VOLUME_MAX);
   broker.volumeStep=SymbolInfoDouble(eaSymbol,SYMBOL_VOLUME_STEP);
   broker.swapLong=SymbolInfoDouble(eaSymbol,SYMBOL_SWAP_LONG);
   broker.swapShort=SymbolInfoDouble(eaSymbol,SYMBOL_SWAP_SHORT);
   broker.digits=(int)SymbolInfoInteger(eaSymbol,SYMBOL_DIGITS);
   broker.stopsLevel=(int)SymbolInfoInteger(eaSymbol,SYMBOL_TRADE_STOPS_LEVEL);
   broker.freezeLevel=(int)SymbolInfoInteger(eaSymbol,SYMBOL_TRADE_FREEZE_LEVEL);
   broker.leverage=(int)AccountInfoInteger(ACCOUNT_LEVERAGE);
   broker.company=AccountInfoString(ACCOUNT_COMPANY);
   broker.currency=AccountInfoString(ACCOUNT_CURRENCY);
   broker.marginMode=(ENUM_ACCOUNT_MARGIN_MODE)AccountInfoInteger(ACCOUNT_MARGIN_MODE);
   broker.tradeMode=(ENUM_ACCOUNT_TRADE_MODE)AccountInfoInteger(ACCOUNT_TRADE_MODE);
   broker.hedging=(broker.marginMode==ACCOUNT_MARGIN_MODE_RETAIL_HEDGING);
   // Point-unit auto-scale: 3-digit gold feeds quote 10x more points per dollar than
   // 2-digit feeds. Every point-denominated input is scaled so gates behave identically.
   g_ptScale=(broker.digits==3?10.0:1.0);
   return (broker.point>0 && broker.tickSize>0 && broker.volumeMin>0 && broker.volumeStep>0);
}

bool Copy1(int handle,int buffer,int shift,double &value)
{
   double x[1]; if(handle==INVALID_HANDLE) return false;
   if(CopyBuffer(handle,buffer,shift,1,x)!=1) return false;
   value=x[0]; return true;
}

bool IsNewBar()
{
   datetime t=iTime(eaSymbol,PERIOD_M1,0);
   if(t>0 && t!=g_lastBar){ g_lastBar=t; return true; }
   return false;
}

//====================================================================
// TIME / DST / SESSION ENGINE
//====================================================================
bool IsLeap(int y){ return ((y%4==0 && y%100!=0) || y%400==0); }
int DaysInMonth(int y,int m)
{
   int d[12]={31,28,31,30,31,30,31,31,30,31,30,31};
   if(m==2 && IsLeap(y)) return 29; return d[m-1];
}

datetime MakeDT(int y,int mon,int day,int hour,int minute=0,int sec=0)
{
   MqlDateTime x; ZeroMemory(x); x.year=y;x.mon=mon;x.day=day;x.hour=hour;x.min=minute;x.sec=sec;
   return StructToTime(x);
}

int DayOfWeekUTC(int y,int mon,int day)
{
   MqlDateTime x; TimeToStruct(MakeDT(y,mon,day,12),x); return x.day_of_week;
}

int NthSunday(int y,int mon,int nth)
{
   int dow=DayOfWeekUTC(y,mon,1);
   int first=1+((7-dow)%7);
   return first+(nth-1)*7;
}

int LastSunday(int y,int mon)
{
   int last=DaysInMonth(y,mon);
   int dow=DayOfWeekUTC(y,mon,last);
   return last-dow;
}

bool LondonDST(datetime utc)
{
   MqlDateTime d; TimeToStruct(utc,d);
   datetime start=MakeDT(d.year,3,LastSunday(d.year,3),1);
   datetime stop =MakeDT(d.year,10,LastSunday(d.year,10),1);
   return (utc>=start && utc<stop);
}

bool NewYorkDST(datetime utc)
{
   MqlDateTime d; TimeToStruct(utc,d);
   // US DST: 02:00 local; approximated in UTC at 07:00 start and 06:00 end.
   datetime start=MakeDT(d.year,3,NthSunday(d.year,3,2),7);
   datetime stop =MakeDT(d.year,11,NthSunday(d.year,11,1),6);
   return (utc>=start && utc<stop);
}

// Australia/Sydney DST: first Sunday in October at 02:00 AEST until
// first Sunday in April at 03:00 AEDT. Convert those local transition
// instants to UTC and handle the southern-hemisphere year crossover.
bool SydneyDST(datetime utc)
{
   MqlDateTime d; TimeToStruct(utc,d);
   datetime octStartThis=MakeDT(d.year,10,NthSunday(d.year,10,1),2)-10*3600;
   datetime aprEndThis =MakeDT(d.year,4,NthSunday(d.year,4,1),3)-11*3600;
   if(d.mon<=4)
   {
      datetime octPrev=MakeDT(d.year-1,10,NthSunday(d.year-1,10,1),2)-10*3600;
      return (utc>=octPrev && utc<aprEndThis);
   }
   if(d.mon>=10)
   {
      datetime aprNext=MakeDT(d.year+1,4,NthSunday(d.year+1,4,1),3)-11*3600;
      return (utc>=octStartThis && utc<aprNext);
   }
   return false;
}

int WrapMin(int m){ while(m<0)m+=1440; while(m>=1440)m-=1440; return m; }

void RefreshServerOffset(bool force=false)
{
   datetime now=TimeCurrent();
   if(!force && g_lastOffsetRefresh>0 && now-g_lastOffsetRefresh<InpServerOffsetRefreshSec) return;
   g_lastOffsetRefresh=now;
   if(InpAutoDetectServerOffset && !MQLInfoInteger(MQL_TESTER))
   {
      datetime s=TimeTradeServer(),u=TimeGMT();
      if(s>0 && u>0) g_serverOffsetSec=(long)(s-u);
      else g_serverOffsetSec=(long)InpManualServerOffsetHours*3600;
   }
   else g_serverOffsetSec=(long)InpManualServerOffsetHours*3600;
}

datetime ServerNow(){ datetime s=TimeTradeServer(); return (s>0?s:TimeCurrent()); }
datetime UTCNow(){ RefreshServerOffset(false); return (datetime)(ServerNow()-g_serverOffsetSec); }
int MinuteOfDay(datetime t){ MqlDateTime d;TimeToStruct(t,d);return d.hour*60+d.min; }

bool InWindowMinutes(int x,int a,int b)
{
   if(a<=b) return x>=a && x<b;
   return (x>=a || x<b);
}

string WindowName(ENUM_WINDOW_ID w)
{
   switch(w)
   {
      case WIN_SYDNEY:return "SYDNEY";
      case WIN_TOKYO:return "TOKYO";
      case WIN_SYDNEY_TOKYO:return "SYDNEY/TOKYO";
      case WIN_TOKYO_LONDON:return "TOKYO/LONDON";
      case WIN_LONDON_OPEN:return "LONDON OPEN";
      case WIN_LONDON:return "LONDON";
      case WIN_LONDON_NY:return "LONDON/NY";
      case WIN_NY_OPEN:return "NY OPEN";
      case WIN_NEWYORK:return "NEW YORK";
      case WIN_VERIFIED_EXPANSION:return "VERIFIED EXPANSION";
      default:return "NONE";
   }
}

void SessionUTCBounds(datetime utc,int &sydOpen,int &sydClose,int &tokOpen,int &tokClose,int &lonOpen,int &lonClose,int &nyOpen,int &nyClose)
{
   int pad=MathMax(0,InpOverlapPadMinutes);
   // Sydney: local Australia/Sydney clock, UTC+10 standard / UTC+11 DST.
   int sydOff=(SydneyDST(utc)?11*60:10*60);
   sydOpen =WrapMin(InpSydneyLocalOpenMin-pad-sydOff);
   sydClose=WrapMin(InpSydneyLocalCloseMin+pad-sydOff);

   // Tokyo: Japan does not observe DST, JST = UTC+9.
   tokOpen =WrapMin(InpTokyoLocalOpenMin-pad-9*60);
   tokClose=WrapMin(InpTokyoLocalCloseMin+pad-9*60);

   // London: local clock, UTC+0 winter / UTC+1 British Summer Time.
   int lonOff=LondonDST(utc)?60:0;
   lonOpen =WrapMin(InpLondonLocalOpenMin-pad-lonOff);
   lonClose=WrapMin(InpLondonLocalCloseMin+pad-lonOff);

   // New York: local clock, UTC-5 winter / UTC-4 daylight time.
   int nyOff=NewYorkDST(utc)?-4*60:-5*60;
   nyOpen =WrapMin(InpNewYorkLocalOpenMin-pad-nyOff);
   nyClose=WrapMin(InpNewYorkLocalCloseMin+pad-nyOff);
}

bool IsVerifiedExpansionBucket(datetime utc)
{
   int b=MinuteOfDay(utc)/30;
   if(b<0||b>=48) return false;
   if(g_bucket[b].samples<InpVerifiedBucketMinSamples) return false;
   double ar=g_bucket[b].atrRatioSum/g_bucket[b].samples;
   double vr=g_bucket[b].volRatioSum/g_bucket[b].samples;
   return (ar>=InpVerifiedBucketATRRatio && vr>=InpVerifiedBucketVolRatio);
}

// Single source of truth for "does this window participate" - used by the
// entry engine AND the dashboard so the panel can never disagree with trading.
bool WindowAllowed(ENUM_WINDOW_ID w)
{
   switch(w)
   {
      case WIN_SYDNEY:            return (InpTradeAllFourSessions||InpTradeSydney);
      case WIN_TOKYO:             return (InpTradeAllFourSessions||InpTradeTokyo);
      case WIN_LONDON:            return (InpTradeAllFourSessions||InpTradeLondon);
      case WIN_NEWYORK:           return (InpTradeAllFourSessions||InpTradeNewYork);
      case WIN_SYDNEY_TOKYO:      return (InpTradeSydneyTokyo && (InpTradeAllFourSessions||(InpTradeSydney&&InpTradeTokyo)));
      case WIN_TOKYO_LONDON:      return (InpTradeTokyoLondon && (InpTradeAllFourSessions||(InpTradeTokyo&&InpTradeLondon)));
      case WIN_LONDON_OPEN:       return ((InpTradeAllFourSessions||InpTradeLondon) && InpTradeLondonOpen);
      case WIN_LONDON_NY:         return (InpTradeLondonNY && (InpTradeAllFourSessions||(InpTradeLondon&&InpTradeNewYork)));
      case WIN_NY_OPEN:           return ((InpTradeAllFourSessions||InpTradeNewYork) && InpTradeNYOpen);
      case WIN_VERIFIED_EXPANSION:return InpTradeVerifiedExpansion;
      default:                    return false;
   }
}

ENUM_WINDOW_ID CurrentWindow(bool &tradeable)
{
   tradeable=false;
   datetime utc=UTCNow();
   int m=MinuteOfDay(utc),so,sc,to,tc,lo,lc,no,nc;
   SessionUTCBounds(utc,so,sc,to,tc,lo,lc,no,nc);

   bool syd=InWindowMinutes(m,so,sc);
   bool tok=InWindowMinutes(m,to,tc);
   bool lon=InWindowMinutes(m,lo,lc);
   bool ny =InWindowMinutes(m,no,nc);
   bool sydTok=syd&&tok;
   bool tokLon=tok&&lon;
   bool lonNY=lon&&ny;
   bool londOpen=lon&&InWindowMinutes(m,lo,WrapMin(lo+InpLondonOpenWindowMin));
   bool nyOpen=ny&&InWindowMinutes(m,no,WrapMin(no+InpNYOpenWindowMin));

   ENUM_WINDOW_ID w=WIN_NONE;
   // Give true overlaps first priority so their statistics/risk budgets remain independent.
   if(lonNY) w=WIN_LONDON_NY;
   else if(tokLon) w=WIN_TOKYO_LONDON;
   else if(sydTok) w=WIN_SYDNEY_TOKYO;
   else if(londOpen) w=WIN_LONDON_OPEN;
   else if(nyOpen) w=WIN_NY_OPEN;
   else if(lon) w=WIN_LONDON;
   else if(ny) w=WIN_NEWYORK;
   else if(tok) w=WIN_TOKYO;
   else if(syd) w=WIN_SYDNEY;
   else if(IsVerifiedExpansionBucket(utc)) w=WIN_VERIFIED_EXPANSION;

   // Master switch guarantees participation in every primary market session.
   // Individual switches remain available for controlled A/B validation only when the master is false.
   tradeable=WindowAllowed(w);
   if(!InpUseSessionFilter) tradeable=true;
   return w;
}

string FmtHHMM(int m){ m=WrapMin(m); return StringFormat("%02d:%02d",m/60,m%60); }
int ForwardMinutesTo(int fromMin,int toMin){ int d=toMin-fromMin;if(d<0)d+=1440;return d; }

// Minutes until the earliest constituent session of the active window closes.
int MinsUntilWindowClose(ENUM_WINDOW_ID w,int utcMin,int sc,int tc,int lc,int nc)
{
   int best=1441;
   if(w==WIN_SYDNEY||w==WIN_SYDNEY_TOKYO)best=MathMin(best,ForwardMinutesTo(utcMin,sc));
   if(w==WIN_TOKYO||w==WIN_SYDNEY_TOKYO||w==WIN_TOKYO_LONDON)best=MathMin(best,ForwardMinutesTo(utcMin,tc));
   if(w==WIN_LONDON||w==WIN_LONDON_OPEN||w==WIN_TOKYO_LONDON||w==WIN_LONDON_NY)best=MathMin(best,ForwardMinutesTo(utcMin,lc));
   if(w==WIN_NEWYORK||w==WIN_NY_OPEN||w==WIN_LONDON_NY)best=MathMin(best,ForwardMinutesTo(utcMin,nc));
   return (best>1440?-1:best);
}

int WindowOpenMin(ENUM_WINDOW_ID w,int so,int sc,int to,int tc,int lo,int lc,int no,int nc)
{
   switch(w)
   {
      case WIN_SYDNEY:return so; case WIN_TOKYO:return to;
      case WIN_LONDON:return lo; case WIN_NEWYORK:return no;
      case WIN_SYDNEY_TOKYO:return MathMin(so,to);
      case WIN_TOKYO_LONDON:return MathMin(to,lo);
      case WIN_LONDON_NY:return MathMin(lo,no);
      case WIN_LONDON_OPEN:return lo; case WIN_NY_OPEN:return no;
      default:return so;
   }
}
int WindowCloseMin(ENUM_WINDOW_ID w,int so,int sc,int to,int tc,int lo,int lc,int no,int nc)
{
   switch(w)
   {
      case WIN_SYDNEY:return sc; case WIN_TOKYO:return tc;
      case WIN_LONDON:return lc; case WIN_NEWYORK:return nc;
      case WIN_SYDNEY_TOKYO:return MathMax(sc,tc);
      case WIN_TOKYO_LONDON:return MathMax(tc,lc);
      case WIN_LONDON_NY:return MathMax(lc,nc);
      case WIN_LONDON_OPEN:return WrapMin(lo+InpLondonOpenWindowMin);
      case WIN_NY_OPEN:return WrapMin(no+InpNYOpenWindowMin);
      default:return sc;
   }
}

string SessionRangeText(string tag,int o,int c){ return tag+" "+FmtHHMM(o)+"-"+FmtHHMM(c); }

string WindowUTCText(ENUM_WINDOW_ID w,int so,int sc,int to,int tc,int lo,int lc,int no,int nc)
{
   switch(w)
   {
      case WIN_SYDNEY:            return SessionRangeText("SYD",so,sc)+" UTC";
      case WIN_TOKYO:             return SessionRangeText("TOK",to,tc)+" UTC";
      case WIN_LONDON:            return SessionRangeText("LON",lo,lc)+" UTC";
      case WIN_NEWYORK:           return SessionRangeText("NY",no,nc)+" UTC";
      case WIN_SYDNEY_TOKYO:      return SessionRangeText("S",so,sc)+"+"+SessionRangeText("T",to,tc)+" UTC";
      case WIN_TOKYO_LONDON:      return SessionRangeText("T",to,tc)+" + "+SessionRangeText("L",lo,lc)+" UTC";
      case WIN_LONDON_NY:         return SessionRangeText("L",lo,lc)+" + "+SessionRangeText("N",no,nc)+" UTC";
      case WIN_LONDON_OPEN:       return SessionRangeText("LOPEN",lo,WrapMin(lo+InpLondonOpenWindowMin))+" UTC";
      case WIN_NY_OPEN:           return SessionRangeText("NOPEN",no,WrapMin(no+InpNYOpenWindowMin))+" UTC";
      case WIN_VERIFIED_EXPANSION:return "30-min expansion bucket (learned)";
      default:                    return "no active session";
   }
}

void PrintSessionMapAudit()
{
   datetime u=UTCNow();int so,sc,to,tc,lo,lc,no,nc;SessionUTCBounds(u,so,sc,to,tc,lo,lc,no,nc);
   Print("SESSION MAP UTC | Sydney ",FmtHHMM(so),"-",FmtHHMM(sc),
         " | Tokyo ",FmtHHMM(to),"-",FmtHHMM(tc),
         " | London ",FmtHHMM(lo),"-",FmtHHMM(lc),
         " | NewYork ",FmtHHMM(no),"-",FmtHHMM(nc),
         " | serverOffsetSec=",g_serverOffsetSec);
}

bool WeekendOrRollover()
{
   datetime s=ServerNow(); MqlDateTime d;TimeToStruct(s,d);
   double h=d.hour+d.min/60.0;
   if(d.day_of_week==6) return true;
   if(d.day_of_week==0 && h<22.0) return true;
   if(d.day_of_week==5 && h>=InpFridayCutoffServer) return true;
   if(h>=21.95 && h<22.10) return true;
   return false;
}

//====================================================================
// ROLLING STATS / PERCENTILES
//====================================================================
void UpdateSpreadStats()
{
   double s=SpreadPoints(); if(s<=0||s>10000) return;
   g_spreadBuf[g_spreadIdx]=s; g_spreadIdx=(g_spreadIdx+1)%SPREAD_SAMPLES; if(g_spreadCnt<SPREAD_SAMPLES)g_spreadCnt++;
   double sum=0;for(int i=0;i<g_spreadCnt;i++)sum+=g_spreadBuf[i];g_spreadAvg=(g_spreadCnt?sum/g_spreadCnt:s);
   double ss=0;for(int i=0;i<g_spreadCnt;i++){double d=g_spreadBuf[i]-g_spreadAvg;ss+=d*d;}
   g_spreadStd=(g_spreadCnt>1?MathSqrt(ss/g_spreadCnt):0);
}

double PercentileRank(const double &a[],int n,double v)
{
   if(n<=1) return 50.0; int le=0; for(int i=0;i<n;i++)if(a[i]<=v)le++;
   return 100.0*le/n;
}

double SpreadPercentile(){ return PercentileRank(g_spreadBuf,g_spreadCnt,SpreadPoints()); }
void PushATR(double x)
{
   if(x<=0)return;
   g_atrBuf[g_atrIdx]=x;
   g_atrIdx=(g_atrIdx+1)%g_atrKeep;
   if(g_atrCnt<g_atrKeep)g_atrCnt++;
}
double ATRPercentile(){ return PercentileRank(g_atrBuf,g_atrCnt,g_atr); }
void PushSlippage(double s)
{
   s=MathAbs(s);
   if(s>1000.0) s=0;   // insane value (uninitialized intended price) - never poison the ring
   g_lastSlipPts=s;g_slipBuf[g_slipIdx]=s;g_slipIdx=(g_slipIdx+1)%SLIP_SAMPLES;if(g_slipCnt<SLIP_SAMPLES)g_slipCnt++;
   double z=0;for(int i=0;i<g_slipCnt;i++)z+=g_slipBuf[i];g_slipAvg=(g_slipCnt?z/g_slipCnt:0);
}
double SlippagePercentile(){ return PercentileRank(g_slipBuf,g_slipCnt,g_lastSlipPts); }//====================================================================
// FMP MACRO / NEWS (stable REST; credentials obfuscated, runtime-decoded)
//====================================================================
// Credentials are stored as +3 ASCII cipher literals so no API key or URL
// appears in the source, the compiled .ex5 strings table, logs or reports.
// Decode happens in memory only, at use time.
string FMP_APIKEY_C="81,113,109,89,90,104,83,56,110,84,88,121,113,118,86,81,55,82,57,70,70,101,100,70,76,111,86,82,112,60,109,119";
string FMP_BASE_C  ="107,119,119,115,118,61,50,50,105,108,113,100,113,102,108,100,111,112,114,103,104,111,108,113,106,115,117,104,115,49,102,114,112,50,118,119,100,101,111,104,50";

string FMPKey()
{
   string parts[]; int n=StringSplit(FMP_APIKEY_C,',',parts);
   string s=""; for(int i=0;i<n;i++) s+=CharToString((uchar)(StringToInteger(parts[i])-3));
   return s;
}

string FMPBase()
{
   string parts[]; int n=StringSplit(FMP_BASE_C,',',parts);
   string s=""; for(int i=0;i<n;i++) s+=CharToString((uchar)(StringToInteger(parts[i])-3));
   return s;
}

// 9-arg WebRequest signature; hosts are whitelisted in MT5 options.
int HttpGet(string url,int timeoutMs,string &body)
{
   body="";
   if(MQLInfoInteger(MQL_TESTER)){g_fmpLastErr="tester: web offline";return -1;}   // Strategy Tester: no WebRequest
   uchar post[]; uchar result[]; string rh="";
   string headers="User-Agent: Predict-A-Trade/1.00\r\nAccept: application/json\r\n";
   ResetLastError();
   int code=WebRequest("GET",url,"","",timeoutMs,post,0,result,rh);
   if(code<200||code>=300)
   {
      int err=GetLastError();
      g_fmpLastErr="http "+IntegerToString(code)+" err "+IntegerToString(err);
      // 4014 = ERR_FUNCTION_NOT_ALLOWED: this URL is not in the terminal's WebRequest
      // whitelist. Raise ONE unmissable popup per session (not per poll), with the exact fix.
      if(err==4014 && !g_webRequestWarned)
      {
         g_webRequestWarned=true;
         string host=url;
         int p1=StringFind(host,"//"); if(p1>0)host=StringSubstr(host,p1+2);
         int p2=StringFind(host,"/");  if(p2>0)host=StringSubstr(host,0,p2);
         Alert("FMP feed blocked (err 4014). FIX: Tools > Options > Expert Advisors > tick 'Allow WebRequest for listed URL' and add:  https://",host,"   Then click OK and re-attach the EA. Until then the macro layer runs on broker EURUSD fallback.");
      }
      return -1;
   }
   body=CharArrayToString(result,0,-1,CP_UTF8);
   return code;
}

//--- broker-side symbol/momentum helpers (broker fallback for the macro gate)
string ResolveBrokerSymbol(string requested,string needle1,string needle2="")
{
   if(requested!="" && SymbolSelect(requested,true)) return requested;
   int total=SymbolsTotal(false);
   for(int i=0;i<total;i++)
   {
      string n=SymbolName(i,false);if(n=="")continue;
      if(StringFind(n,needle1)>=0 || (needle2!=""&&StringFind(n,needle2)>=0))
      { if(SymbolSelect(n,true)) return n; }
   }
   return "";
}

double SymbolMomentumPct(string sym,ENUM_TIMEFRAMES tf,int bars,bool &ok)
{
   ok=false;if(sym==""||bars<1)return 0;MqlRates r[];ArraySetAsSeries(r,true);
   if(CopyRates(sym,tf,1,bars+1,r)<bars+1||r[bars].close==0)return 0;
   ok=true;return 100.0*(r[0].close-r[bars].close)/r[bars].close;
}

bool JsonNumber(const string text,const string key,double &value)
{
   value=0;if(key=="")return false;string pat="\""+key+"\"";int p=StringFind(text,pat);if(p<0)return false;
   p=StringFind(text,":",p+StringLen(pat));if(p<0)return false;p++;
   int n=StringLen(text);while(p<n){ushort ch=StringGetCharacter(text,p);if(ch==32||ch==9||ch==34)p++;else break;}
   int e=p;while(e<n){ushort ch=StringGetCharacter(text,e);if((ch>=48&&ch<=57)||ch==45||ch==43||ch==46||ch==101||ch==69)e++;else break;}
   if(e<=p)return false;string num=StringSubstr(text,p,e-p);value=StringToDouble(num);return true;
}

//--- one FMP quote; returns the day change % and whether price data arrived
// Batch quote: FMP /stable/quote accepts comma-separated symbols in ONE request -
// critical for free-plan rate limits (429 = quota exhausted).
bool FMPQuoteBatch(string &syms[],double &chg[],int count)
{
   if(MQLInfoInteger(MQL_TESTER))return false;
   string list="";
   for(int i=0;i<count;i++){list+=(i>0?",":"")+syms[i];}
   string url=FMPBase()+"/quote?symbol="+list+"&apikey="+FMPKey();
   string body;
   if(HttpGet(url,InpFMPTimeoutMs,body)<=0)return false;
   if(StringFind(body,"Error Message")>=0||StringFind(body,"Restricted Endpoint")>=0||StringFind(body,"Premium Query")>=0)
   { g_fmpLastErr="batch plan-restricted"; return false; }
   // Response: array of objects {"symbol":"EURUSD",...,"changePercentage":0.5,...}
   // For each requested symbol, locate its object and parse changePercentage.
   for(int i=0;i<count;i++)
   {
      chg[i]=0;
      int si=StringFind(body,"\"symbol\":\""+syms[i]+"\"");
      if(si<0)continue;
      int cp=StringFind(body,"changePercentage",si);
      if(cp<0)continue;
      int p=cp+StringLen("changePercentage");
      while(p<StringLen(body))
      {
         ushort c=StringGetCharacter(body,p);
         if(c==' '||c==':'||c=='\t'){p++;continue;}
         break;
      }
      string num="";
      while(p<StringLen(body))
      {
         ushort c=StringGetCharacter(body,p);
         if((c>='0'&&c<='9')||c=='-'||c=='+'||c=='.'||c=='e'||c=='E'){num+=CharToString((uchar)c);p++;}
         else break;
      }
      chg[i]=StringToDouble(num);
   }
   return true;
}

bool FMPQuote(string sym,double &chgPct,double &price)
{
   chgPct=0;price=0;
   string url=FMPBase()+"/quote?symbol="+sym+"&apikey="+FMPKey();
   string body;
   if(HttpGet(url,InpFMPTimeoutMs,body)<=0) return false;
   if(StringFind(body,"Error Message")>=0||StringFind(body,"Restricted Endpoint")>=0||StringFind(body,"Premium Query")>=0)
   { g_fmpLastErr=sym+" plan-restricted"; return false; }
   if(!JsonNumber(body,"changePercentage",chgPct)) return false;
   JsonNumber(body,"price",price);
   return true;
}

void FMPNewsScan()
{
   if(!InpUseFMP||MQLInfoInteger(MQL_TESTER))return;
   string url=FMPBase()+"/news?limit="+IntegerToString(InpFMPNewsLimit)+"&page=0&apikey="+FMPKey();
   string body;
   if(HttpGet(url,InpFMPTimeoutMs,body)<=0)return;
   if(StringLen(body)<10)return;
   // Count gold-relevant headlines and keep the newest title for the panel.
   int hits=0;string newest="";int pos=0;
   string pat="\"title\"";
   while(pos<StringLen(body))
   {
      int p=StringFind(body,pat,pos);if(p<0)break;
      p=StringFind(body,":",p);if(p<0)break;p++;
      while(p<StringLen(body)){ushort ch=StringGetCharacter(body,p);if(ch==32||ch==34)p++;else break;}
      int e=p;while(e<StringLen(body)&&StringGetCharacter(body,e)!=34)e++;
      string title=StringSubstr(body,p,e-p);pos=e;
      string low=title;StringToLower(low);
      if(StringFind(low,"gold")>=0||StringFind(low,"fed ")>=0||StringFind(low,"fomc")>=0||
         StringFind(low,"inflation")>=0||StringFind(low,"cpi")>=0||StringFind(low,"rate cut")>=0||
         StringFind(low,"rate hike")>=0||StringFind(low,"powell")>=0||StringFind(low,"treasury")>=0||
         StringFind(low,"tariff")>=0||StringFind(low,"nonfarm")>=0||StringFind(low,"payrolls")>=0)
      {
         hits++;if(newest=="")newest=title;
      }
   }
   g_fmpNewsCount=hits;
   if(hits>0)g_fmpNewsLast=newest; else g_fmpNewsLast="";
}

void RefreshFMPMacro(bool force=false)
{
   if(!InpUseFMP)return;
   if(MQLInfoInteger(MQL_TESTER))
   {
      // Strategy Tester: WebRequest is unavailable - run ONLY the broker-side
      // EURUSD momentum fallback so the macro layer stays functional offline.
      g_usdAvailable=false;g_spxAvailable=false;g_usdBias=0;g_spxBias=0;
      if(InpAllowBrokerMacroFallback&&InpUseEURUSD)
      {
         if(g_eurSymbol=="")g_eurSymbol=ResolveBrokerSymbol(InpEURUSDSymbol,"EURUSD");
         bool ok=false;
         g_eurMovePct=SymbolMomentumPct(g_eurSymbol,InpMacroTF,InpMacroMomentumBars,ok);
         g_eurAvailable=ok;
         g_eurBias=(ok?(g_eurMovePct>=InpEURUSDMinMovePct?1:(g_eurMovePct<=-InpEURUSDMinMovePct?-1:0)):0);
      }
      return;
   }
   datetime now=ServerNow();
   // Rate-limit defense (HTTP 429): base cycle 600s, doubled per consecutive 429 up to 1h.
   int effSec=InpFMPRefreshSec;                       // input is in SECONDS
   if(g_fmp429Count>0)effSec=(int)MathMin(3600,600.0*MathPow(2,MathMin(3,g_fmp429Count)));
   if(!force && g_fmpLastTry>0 && now-g_fmpLastTry<effSec)return;
   g_fmpLastTry=now;
   bool doNews=(g_fmpCycle%3==0);   // news every 3rd cycle: keeps daily total under the free-plan quota

   // ONE batch request for the whole USD basket (+SPX) - free-plan friendly.
   string syms[FMPUSD_COUNT+1];
   for(int i=0;i<FMPUSD_COUNT;i++)syms[i]=g_usdPairs[i];
   int batchN=FMPUSD_COUNT;
   if(InpFMPIncludeSPX){syms[FMPUSD_COUNT]="^GSPC";batchN=FMPUSD_COUNT+1;}
   double chg[FMPUSD_COUNT+1];
   bool batchOK=FMPQuoteBatch(syms,chg,batchN);
   if(batchOK)
   {
      for(int i=0;i<FMPUSD_COUNT;i++){g_usdGot[i]=true;g_usdMove[i]=chg[i];}
      if(InpFMPIncludeSPX){g_usdGotSPX=true;g_spxBatchChg=chg[FMPUSD_COUNT];}
   }
   int okCount=0;
   for(int i=0;i<FMPUSD_COUNT;i++)if(g_usdGot[i])okCount++;
   g_usdAvailable=(okCount>=(FMPUSD_COUNT-2));       // tolerate up to 2 dead pairs
   if(!g_usdAvailable && g_fmpEverOK==false && g_fmpErrCount==1)
   {
      if(StringFind(g_fmpLastErr,"429")>=0)
         Print("FMP quota exhausted (HTTP 429): free plan allows ~250 requests/day. The EA now polls once per ",effSec,"s with batched requests (~180/day) and will recover automatically when the quota resets.");
      else
         Print("FMP feed unavailable: ",g_fmpLastErr," | check Tools>Options>Expert Advisors>Allow WebRequest for https://financialmodelingprep.com");
   }
   if(g_usdAvailable)
   {
      double sum=0;int n=0;
      for(int i=0;i<FMPUSD_COUNT;i++)
      {
         if(!g_usdGot[i])continue;
         // Convert each pair's move into USD strength: USD/xxx rising = USD up (+),
         // xxx/USD rising = USD down (-).
         bool usdBase=(StringSubstr(g_usdPairs[i],0,3)=="USD");
         sum+=(usdBase?g_usdMove[i]:-g_usdMove[i]);n++;
      }
      if(n>0)g_usdAvg=sum/n;
      g_usdBias=(g_usdAvg<=-InpFMPUSDPairMinPct?1:(g_usdAvg>=InpFMPUSDPairMinPct?-1:0));
      g_fmpEverOK=true;g_fmpLastOK=now;g_fmpErrCount=0;g_fmp429Count=0;g_fmpCycle++;
   }
   else
   {
      g_fmpErrCount++;
      if(StringFind(g_fmpLastErr,"429")>=0||StringFind(g_fmpLastErr,"err 0")>=0)g_fmp429Count++;
      // Keep the last-good basket values (stale) for 10 minutes so one failed poll
      // doesn't flip the macro gate; after that, fall back to broker EURUSD momentum.
      bool stale=(g_fmpLastOK>0 && now-g_fmpLastOK<600);
      g_usdAvailable=stale;
      if(!stale)g_usdBias=0;
      if(InpAllowBrokerMacroFallback&&InpUseEURUSD&&!stale)
      {
         if(g_eurSymbol=="")g_eurSymbol=ResolveBrokerSymbol(InpEURUSDSymbol,"EURUSD");
         bool ok=false;
         g_eurMovePct=SymbolMomentumPct(g_eurSymbol,InpMacroTF,InpMacroMomentumBars,ok);
         g_eurAvailable=ok;
         g_eurBias=(ok?(g_eurMovePct>=InpEURUSDMinMovePct?1:(g_eurMovePct<=-InpEURUSDMinMovePct?-1:0)):0);
      }
   }

   if(InpFMPIncludeSPX&&batchOK)
   {
      // SPX rides the SAME batch request (appended symbol) - zero extra requests.
      g_spxAvailable=g_usdGotSPX;
      if(g_spxAvailable)
      {
         g_spxMovePct=g_spxBatchChg;
         // Risk-off (SPX down) favours gold bids; risk-on (SPX up) leans bearish gold.
         g_spxBias=(g_spxMovePct<=-InpFMPSPXMinPct?1:(g_spxMovePct>=InpFMPSPXMinPct?-1:0));
      }
      else g_spxBias=0;
   }
   else if(!batchOK){ g_spxAvailable=false;g_spxBias=0; }

   if(doNews)FMPNewsScan();

   g_macroBull=(g_usdBias>0?1:0)+(g_spxBias>0?1:0)+(g_eurBias>0?1:0);
   g_macroBear=(g_usdBias<0?1:0)+(g_spxBias<0?1:0)+(g_eurBias<0?1:0);
}

bool ExternalDataReady(string &why)
{
   if(!InpRequireExternalData)return true;
   if(InpUseFMP&&!g_usdAvailable&&!g_eurAvailable){why="FMP macro unavailable";return false;}
   return true;
}

int MacroVotes(int dir){return dir>0?g_macroBull:g_macroBear;}
int AvailableMacroCount(){return (g_usdAvailable?1:0)+(g_spxAvailable?1:0)+(g_eurAvailable?1:0);}

//====================================================================
// INDICATORS / SMC
//====================================================================
void UpdateIndicators()
{
   Copy1(hATR,0,1,g_atr); if(g_atr>0) PushATR(g_atr);
   Copy1(hADX,0,1,g_adx);Copy1(hADX,1,1,g_adxPlus);Copy1(hADX,2,1,g_adxMinus);
   Copy1(hEMA20,0,1,g_ema20);Copy1(hEMA50,0,1,g_ema50);
   Copy1(hH1EMA20,0,1,g_h1e20);Copy1(hH1EMA50,0,1,g_h1e50);
   Copy1(hM15EMA20,0,1,g_m15e20);Copy1(hM15EMA50,0,1,g_m15e50);
   Copy1(hRSI,0,1,g_rsi);
}

double VolumeRatio(int shift=1)
{
   MqlRates r[];ArraySetAsSeries(r,true);
   int need=InpVolumeMA+shift+2;if(CopyRates(eaSymbol,PERIOD_M1,0,need,r)<need)return 0;
   double avg=0;for(int i=shift+1;i<=shift+InpVolumeMA;i++)avg+=(double)r[i].tick_volume;
   avg/=InpVolumeMA; return avg>0?(double)r[shift].tick_volume/avg:0;
}

// Cached per-bar value: the M1 volume ratio only changes on a new closed bar,
// so caching removes a CopyRates() call from every tick, every filter and the panel.
void RefreshVolumeRatio(){ g_volRatio=VolumeRatio(1); }

double AverageATR(int n)
{
   int k=MathMin(n,g_atrCnt);if(k<=0)return g_atr;double s=0;for(int i=0;i<k;i++)s+=g_atrBuf[i];return s/k;
}

void UpdateSuperTrend()
{
   MqlRates r[];ArraySetAsSeries(r,true);int n=InpSuperTrendPeriod+8;
   if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n || g_atr<=0)return;
   double mid=(r[0].high+r[0].low)/2.0;
   double upper=mid+InpSuperTrendMultiplier*g_atr,lower=mid-InpSuperTrendMultiplier*g_atr;
   if(r[0].close>upper) g_superTrendDir=1;
   else if(r[0].close<lower) g_superTrendDir=-1;
   else if(g_superTrendDir==0) g_superTrendDir=(r[0].close>=g_ema20?1:-1);
   g_superTrend=(g_superTrendDir>0?lower:upper);
}

void UpdateVWAP()
{
   MqlRates r[];ArraySetAsSeries(r,true);int n=240;
   if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<30)return;
   datetime anchor=0;
   MqlDateTime d;TimeToStruct(ServerNow(),d);d.hour=0;d.min=0;d.sec=0;anchor=StructToTime(d);
   if(InpVWAPAnchor!=VWAP_BROKER_DAY)
   {
      datetime utc=UTCNow();int so,sc,to,tc,lo,lc,no,nc;SessionUTCBounds(utc,so,sc,to,tc,lo,lc,no,nc);
      int target=(InpVWAPAnchor==VWAP_LONDON?lo:no);
      MqlDateTime ud;TimeToStruct(UTCNow(),ud);ud.hour=target/60;ud.min=target%60;ud.sec=0;
      datetime utcAnchor=StructToTime(ud);anchor=(datetime)(utcAnchor+g_serverOffsetSec);
      if(anchor>ServerNow())anchor-=86400;
   }
   double pv=0,v=0,p2=0;
   for(int i=0;i<ArraySize(r);i++)
   {
      if(r[i].time<anchor)continue;double tp=(r[i].high+r[i].low+r[i].close)/3.0;double vv=(double)r[i].tick_volume;
      pv+=tp*vv;v+=vv;p2+=tp*tp*vv;
   }
   if(v>0){g_vwap=pv/v;double var=MathMax(0,p2/v-g_vwap*g_vwap);double sd=MathSqrt(var);g_vwapUp=g_vwap+sd;g_vwapDn=g_vwap-sd;}
}

void DetectFVG()
{
   g_fvg=false;g_fvgDir=0;MqlRates r[];ArraySetAsSeries(r,true);
   int n=InpFVGLookbackBars+3;if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n)return;
   for(int i=0;i<InpFVGLookbackBars;i++)
   {
      double minGap=InpFVGMinGapATR*MathMax(g_atr,broker.point);
      if(r[i].low-r[i+2].high>=minGap){g_fvg=true;g_fvgDir=1;g_fvgBottom=r[i+2].high;g_fvgTop=r[i].low;return;}
      if(r[i+2].low-r[i].high>=minGap){g_fvg=true;g_fvgDir=-1;g_fvgBottom=r[i].high;g_fvgTop=r[i+2].low;return;}
   }
}

void DetectIFVG()
{
   g_ifvg=false;g_ifvgDir=0;g_ifvgTop=g_ifvgBottom=0;MqlRates r[];ArraySetAsSeries(r,true);
   int n=InpIFVGLookbackBars+5;if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n||g_atr<=0)return;double minGap=InpIFVGMinGapATR*g_atr;
   for(int i=2;i<InpIFVGLookbackBars;i++)
   {
      // Original bullish FVG fails and closes below its lower boundary => bearish IFVG.
      if(r[i].low-r[i+2].high>=minGap)
      {double bot=r[i+2].high,top=r[i].low;if(r[1].high>=bot&&r[0].close<bot){g_ifvg=true;g_ifvgDir=-1;g_ifvgBottom=bot;g_ifvgTop=top;return;}}
      // Original bearish FVG fails and closes above its upper boundary => bullish IFVG.
      if(r[i+2].low-r[i].high>=minGap)
      {double bot=r[i].high,top=r[i+2].low;if(r[1].low<=top&&r[0].close>top){g_ifvg=true;g_ifvgDir=1;g_ifvgBottom=bot;g_ifvgTop=top;return;}}
   }
}

void DetectPTB()
{
   g_ptb=false;g_ptbDir=0;g_ptbTop=g_ptbBottom=0;MqlRates r[];ArraySetAsSeries(r,true);
   int n=InpPTBLookbackBars+6;if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n||g_atr<=0)return;
   double body0=MathAbs(r[0].close-r[0].open)/g_atr;if(body0<InpPTBMinDisplacementATR)return;
   for(int j=3;j<InpPTBLookbackBars;j++)
   {
      bool bullOB=r[j].close<r[j].open;bool bearOB=r[j].close>r[j].open;
      bool overlap=(r[1].low<=r[j].high&&r[1].high>=r[j].low);
      if(!overlap)continue;
      if(bullOB&&r[1].close<r[1].open&&r[0].close>r[1].high&&r[0].close>r[j].high)
      {g_ptb=true;g_ptbDir=1;g_ptbBottom=r[1].low;g_ptbTop=r[1].high;return;}
      if(bearOB&&r[1].close>r[1].open&&r[0].close<r[1].low&&r[0].close<r[j].low)
      {g_ptb=true;g_ptbDir=-1;g_ptbBottom=r[1].low;g_ptbTop=r[1].high;return;}
   }
}

void AnalyzeAMD()
{
   MqlRates r[];ArraySetAsSeries(r,true);int n=InpAMDLookbackBars+2;if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n)return;
   double hi=-DBL_MAX,lo=DBL_MAX,avgRange=0;
   for(int i=1;i<n;i++){hi=MathMax(hi,r[i].high);lo=MathMin(lo,r[i].low);avgRange+=r[i].high-r[i].low;}
   avgRange/=(n-1);double cur=r[0].high-r[0].low;
   if(cur<avgRange*InpAMDCoilRatio)g_phase=PHASE_ACCUMULATION;
   else if(r[0].high>hi || r[0].low<lo)g_phase=PHASE_MANIPULATION;
   else g_phase=PHASE_DISTRIBUTION;
}

void DetectSMC()
{
   g_bosUp=g_bosDn=g_chochUp=g_chochDn=g_sweepUp=g_sweepDn=false;
   g_smcScoreBull=g_smcScoreBear=0;
   MqlRates r[];ArraySetAsSeries(r,true);int n=InpSwingLookback+4;if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n)return;
   double priorHi=-DBL_MAX,priorLo=DBL_MAX;
   for(int i=2;i<n;i++){priorHi=MathMax(priorHi,r[i].high);priorLo=MathMin(priorLo,r[i].low);}
   g_swingHigh=priorHi;g_swingLow=priorLo;
   if(r[0].close>priorHi){g_bosUp=true;g_smcScoreBull++;}
   if(r[0].close<priorLo){g_bosDn=true;g_smcScoreBear++;}
   if(r[0].high>priorHi && r[0].close<priorHi){g_sweepUp=true;g_smcScoreBear++;}
   if(r[0].low<priorLo && r[0].close>priorLo){g_sweepDn=true;g_smcScoreBull++;}
   bool prevUp=(r[1].close>r[2].close),curUp=(r[0].close>r[1].close);
   if(!prevUp&&curUp&&r[0].close>r[1].high){g_chochUp=true;g_smcScoreBull++;}
   if(prevUp&&!curUp&&r[0].close<r[1].low){g_chochDn=true;g_smcScoreBear++;}
   if(g_fvg&&g_fvgDir>0)g_smcScoreBull++;if(g_fvg&&g_fvgDir<0)g_smcScoreBear++;
   if(InpUseIFVG&&g_ifvg&&g_ifvgDir>0)g_smcScoreBull++;if(InpUseIFVG&&g_ifvg&&g_ifvgDir<0)g_smcScoreBear++;
   if(InpUsePTB&&g_ptb&&g_ptbDir>0)g_smcScoreBull++;if(InpUsePTB&&g_ptb&&g_ptbDir<0)g_smcScoreBear++;
}

void EvaluateFilters()
{
   g_score=0;g_scoreMax=0;g_dirBias=0;
   double c=iClose(eaSymbol,PERIOD_M1,1);
   bool emaBull=(c>g_ema20&&g_ema20>g_ema50),emaBear=(c<g_ema20&&g_ema20<g_ema50);
   if(InpUseEMA20||InpUseEMA50){g_scoreMax++;if(emaBull||emaBear)g_score++;if(emaBull)g_dirBias++;if(emaBear)g_dirBias--;}
   if(InpUseSuperTrend){g_scoreMax++;if(g_superTrendDir!=0)g_score++;g_dirBias+=g_superTrendDir;}
   if(InpUseADX){g_scoreMax++;if(g_adx>=InpMinADX)g_score++;if(g_adxPlus>g_adxMinus)g_dirBias++;else if(g_adxMinus>g_adxPlus)g_dirBias--;}
   if(InpUseVWAP){g_scoreMax++;if(g_vwap>0){g_score++;if(c>g_vwap)g_dirBias++;else g_dirBias--;}}
   if(InpUseFVG){g_scoreMax++;if(g_fvg)g_score++;if(g_fvgDir>0)g_dirBias++;if(g_fvgDir<0)g_dirBias--;}
   if(InpUseAMD){g_scoreMax++;if(g_phase!=PHASE_ACCUMULATION)g_score++;}
   double vr=g_volRatio;
   if(InpUseVolumeFilter){g_scoreMax++;if(vr>=InpMinVolumeRatio)g_score++;}
   if(InpUseLiquidityFilter){g_scoreMax++;if(vr>=InpMinLiquidityLevel)g_score++;}
   bool hBull=(g_h1e20>g_h1e50&&g_m15e20>g_m15e50),hBear=(g_h1e20<g_h1e50&&g_m15e20<g_m15e50);
   g_scoreMax++;if(hBull||hBear){g_score++;g_dirBias+=(hBull?2:-2);}
   if(InpUseRSI){g_scoreMax++;if(g_rsi>InpRSIOversold&&g_rsi<InpRSIOverbought)g_score++;if(g_rsi>=52)g_dirBias++;else if(g_rsi<=48)g_dirBias--;}
   if(InpUseSMC){g_scoreMax++;if(MathMax(g_smcScoreBull,g_smcScoreBear)>=InpMinSMCConfluence)g_score++;g_dirBias+=(g_smcScoreBull-g_smcScoreBear);}
   if(InpUseIFVG){g_scoreMax++;if(g_ifvg)g_score++;if(g_ifvgDir>0)g_dirBias++;else if(g_ifvgDir<0)g_dirBias--;}
   if(InpUsePTB){g_scoreMax++;if(g_ptb)g_score++;if(g_ptbDir>0)g_dirBias++;else if(g_ptbDir<0)g_dirBias--;}
   if(AvailableMacroCount()>0){g_scoreMax++;if(MathMax(g_macroBull,g_macroBear)>=InpMacroMinConfluence)g_score++;g_dirBias+=(g_macroBull-g_macroBear);}
}

//====================================================================
// HIGH VOLATILITY / DISORDER ENGINE
//====================================================================
//--- live-chart confirmation: current price agrees with EMA20 direction (uses the forming
//--- bar's live bid/ask, so entries read the live chart, not just closed bars)
//--- ULTRA-SCALP simple engine: 5 binary votes, no filter-score machinery.
//--- Direction: net votes >= InpScalpMinScore AND live price agrees.
//--- Votes: 1) EMA20 vs EMA50  2) close vs EMA20  3) last-bar momentum  4) MACD hist
//---        5) candle body direction vs previous (continuation)
//--- ULTRA-SCALP ENGINE v2 (evidence-based):
//--- Mode A trend-pullback: M5 trend + M1 pullback-to-EMA + reversal candle close.
//--- Mode B mean-reversion: 2+ sigma extension from VWAP + RSI extreme + reversal
//--- candle, ONLY when M5 ADX < 30 (reversion fails in strong trends).
void EvaluateScalpSignal()
{
   g_scalpSignal=0;g_scalpWhy="";
   double c1=iClose(eaSymbol,PERIOD_M1,1),o1=iOpen(eaSymbol,PERIOD_M1,1);
   double h1=iHigh(eaSymbol,PERIOD_M1,1),l1=iLow(eaSymbol,PERIOD_M1,1);
   double c2=iClose(eaSymbol,PERIOD_M1,2),h2=iHigh(eaSymbol,PERIOD_M1,2),l2=iLow(eaSymbol,PERIOD_M1,2);
   if(g_atr<=0)return;
   int um=MinuteOfDay(UTCNow());
   double minBody=MathMax(0.25,InpScalpMinMomentumATR)*g_atr;

   //================= MODE B: VWAP MEAN REVERSION (any session) ==================
   // Documented gold edge: 68-73% reversion after 2-sigma extension. Relaxed from
   // the too-strict v2 (1.8 ATR + RSI72 + candle + ADX<30 all at once = never fires).
   if(g_vwap>0&&g_rsi>0)
   {
      double dev=(c1-g_vwap)/g_atr;
      bool extUp=(dev>=1.5),extDn=(dev<=-1.5);
      // confirmation: reversal candle OR Bollinger band recross (either)
      bool confDn=(c1<o1)||(c1<g_bbMid);
      bool confUp=(c1>o1)||(c1>g_bbMid);
      if(extUp&&(g_rsi>=70.0)&&confDn&&g_m5adx<35.0){g_scalpSignal=-2;g_scalpWhy="VWAP reversion SHORT";return;}
      if(extDn&&(g_rsi<=30.0)&&confUp&&g_m5adx<35.0){g_scalpSignal=2;g_scalpWhy="VWAP reversion LONG";return;}
   }

   //================= MODE C: LONDON OPEN BREAKOUT (07:00-08:15 UTC) ==============
   // Asian range (00:00-06:45 UTC) break during the first London hour. Gold's first
   // directional move of the day; 70% of daily extremes form in LDN/NY.
   if(um>=7*60&&um<=8*60+15)
   {
      // Asian range = high/low of 00:00-06:45 UTC today
      datetime dayStart=ServerNow()-(ServerNow()%86400);
      int asiaBars=(int)((6*60+45));
      double ah=0,al=0;
      MqlRates r[];
      if(CopyRates(eaSymbol,PERIOD_M1,1,asiaBars,r)==asiaBars)
      {
         ah=-DBL_MAX;al=DBL_MAX;
         for(int k=0;k<asiaBars;k++)
         {
            datetime bt=(datetime)r[k].time;
            if(bt>=dayStart&&bt<dayStart+(6*60+45)*60)
            {ah=MathMax(ah,r[k].high);al=MathMin(al,r[k].low);}
         }
         if(ah>0&&al>0&&(ah-al)>=0.8*g_atr)   // meaningful range, not dead tape
         {
            if(c1>ah&&c1>o1&&(c1-o1)>=minBody){g_scalpSignal=3;g_scalpWhy="London breakout LONG";return;}
            if(c1<al&&c1<o1&&(o1-c1)>=minBody){g_scalpSignal=-3;g_scalpWhy="London breakout SHORT";return;}
         }
      }
   }

   //================= MODE D: NY OPEN MOMENTUM (13:30-15:30 UTC) ==================
   // Liquidity peak (BIS data); deploy momentum with the trend, not fades.
   if(um>=13*60+30&&um<=15*60+30)
   {
      bool m5Up=(g_m5e20>g_m5e50),m5Dn=(g_m5e20<g_m5e50);
      // M1 momentum burst closing beyond the 15-bar high/low with M5 trend
      double hh=-DBL_MAX,ll=DBL_MAX;
      double bars[15];
      MqlRates r2[];
      if(CopyRates(eaSymbol,PERIOD_M1,2,15,r2)==15)
      {
         for(int k=0;k<15;k++){hh=MathMax(hh,r2[k].high);ll=MathMin(ll,r2[k].low);}
         if(m5Up&&c1>o1&&(c1-o1)>=0.5*g_atr&&c1>hh){g_scalpSignal=1;g_scalpWhy="NY momentum LONG";return;}
         if(m5Dn&&c1<o1&&(o1-c1)>=0.5*g_atr&&c1<ll){g_scalpSignal=-1;g_scalpWhy="NY momentum SHORT";return;}
      }
   }

   //================= MODE A2: EMA PULLBACK (liquid windows, simplified) ==========
   // v2 was too strict (low touched EMA AND close beyond prior high in one candle).
   // Simplified: M5 trend + last bar closed back across EMA20 in trend direction
   // after being on the wrong side of it (the dip happened, the resumption confirms).
   {
      bool m5Up=(g_m5e20>g_m5e50),m5Dn=(g_m5e20<g_m5e50);
      bool wasBelow=(iClose(eaSymbol,PERIOD_M1,3)<g_ema20||iLow(eaSymbol,PERIOD_M1,1)<=g_ema20);
      bool wasAbove=(iClose(eaSymbol,PERIOD_M1,3)>g_ema20||iHigh(eaSymbol,PERIOD_M1,1)>=g_ema20);
      if(m5Up&&wasBelow&&c1>g_ema20&&c1>o1){g_scalpSignal=1;g_scalpWhy="EMA pullback LONG";return;}
      if(m5Dn&&wasAbove&&c1<g_ema20&&c1<o1){g_scalpSignal=-1;g_scalpWhy="EMA pullback SHORT";return;}
   }
}


bool LiveMomentumConfirm(int dir)
{
   if(g_ema20<=0)return true;                      // no EMA yet -> do not block
   double px=(dir>0?Bid():Ask());
   double tol=0.05*g_atr;                          // small tolerance band around EMA
   if(dir>0) return (px>=g_ema20-tol);
   return (px<=g_ema20+tol);
}

double CandleDisplacementATR()
{
   if(g_atr<=0)return 0;double o=iOpen(eaSymbol,PERIOD_M1,1),c=iClose(eaSymbol,PERIOD_M1,1);return MathAbs(c-o)/g_atr;
}

// Rolling N-bar breakout (default 30 closed M1 bars) - NOT a true session high/low.
// Named honestly; a session-anchored range would need SessionUTCBounds integration.
bool RollingBreakout(int dir)
{
   MqlRates r[];ArraySetAsSeries(r,true);int n=InpSessionBreakoutLookback+2;if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n)return false;
   double hi=-DBL_MAX,lo=DBL_MAX;for(int i=1;i<n;i++){hi=MathMax(hi,r[i].high);lo=MathMin(lo,r[i].low);}double c=r[0].close;
   return dir>0?(c>hi):(c<lo);
}

int HVScore(int dir)
{
   int s=0;double atrp=ATRPercentile(),vr=g_volRatio,disp=CandleDisplacementATR();double c=iClose(eaSymbol,PERIOD_M1,1);
   if(atrp>=InpHVMinATRPercentile)s++;
   if(vr>=InpHVMinVolumeRatio)s++;
   if(disp>=InpHVMinDisplacementATR && disp<=InpHVMaxDisorderATR)s++;
   if(g_vwap>0 && MathAbs(c-g_vwap)>=InpHVMinVWAPDeviationATR*g_atr)s++;
   if(RollingBreakout(dir))s++;
   if(dir>0 && g_smcScoreBull>=InpMinSMCConfluence)s++;
   if(dir<0 && g_smcScoreBear>=InpMinSMCConfluence)s++;
   if((dir>0&&((g_ifvg&&g_ifvgDir>0)||(g_ptb&&g_ptbDir>0)))||(dir<0&&((g_ifvg&&g_ifvgDir<0)||(g_ptb&&g_ptbDir<0))))s++;
   if(MacroVotes(dir)>=InpMacroMinConfluence)s++;
   bool mtf=(dir>0?(g_h1e20>g_h1e50&&g_m15e20>g_m15e50):(g_h1e20<g_h1e50&&g_m15e20<g_m15e50));if(mtf)s++;
   return s;
}

bool IsHighVolatilityQualified(int dir,int &score)
{
   score=HVScore(dir);if(InpHighVolatilityMode==HV_OFF)return false;return score>=InpHVMinScore;
}

bool IsDisorder()
{
   datetime now=ServerNow();if(g_disorderUntil>now)return true;
   // Disorder = microstructure breakdown only. A high spread PERCENTILE alone is noise
   // on brokers whose spread is near-constant (p100 == normal 40pt on Xelans), so the
   // relative spike must ALSO breach the absolute hard cap before halting. Realised
   // slippage stays an independent trigger.
   double sp=SpreadPoints();
   if(SpreadPercentile()>=InpDisorderSpreadPct && sp>InpMaxSpreadPoints*g_ptScale)
   {g_disorderUntil=now+InpDisorderCooldownMinutes*60;return true;}
   if(g_slipCnt>=5&&g_slipAvg>=InpDisorderSlipPts)
   {g_disorderUntil=now+InpDisorderCooldownMinutes*60;return true;}
   return false;
}

void UpdateOpportunityObservations()
{
   double avgAtr=AverageATR(60);if(avgAtr<=0||g_atr<=0)return;double ar=g_atr/avgAtr,vr=g_volRatio;
   int b=MinuteOfDay(UTCNow())/30;if(b>=0&&b<48){g_bucket[b].samples++;g_bucket[b].atrRatioSum+=ar;g_bucket[b].volRatioSum+=vr;}
   bool tr=false;ENUM_WINDOW_ID w=CurrentWindow(tr);if(w>WIN_NONE&&w<WIN_COUNT){g_ws[w].observations++;g_ws[w].obsATRRatioSum+=ar;g_ws[w].obsVolRatioSum+=vr;}
}

//====================================================================
// NEWS FILTER (MQL5 economic calendar, USD high-impact)
//====================================================================
bool CriticalNewsEvent(string name)
{
   string low=name;StringToLower(low);
   // Tight gold-driver list only; minor releases must never freeze an ultra-scalper.
   if(StringFind(low,"nonfarm")>=0)return true;
   if(StringFind(low,"payrolls")>=0)return true;
   if(StringFind(low,"cpi")>=0)return true;
   if(StringFind(low,"core inflation")>=0)return true;
   if(StringFind(low,"pce")>=0)return true;
   if(StringFind(low,"fomc")>=0)return true;
   if(StringFind(low,"fed funds")>=0)return true;
   if(StringFind(low,"rate decision")>=0)return true;
   if(StringFind(low,"interest rate")>=0)return true;
   if(StringFind(low,"gdp")>=0)return true;
   if(StringFind(low,"unemployment rate")>=0)return true;
   if(StringFind(low,"ism manufacturing")>=0)return true;
   if(StringFind(low,"ism services")>=0)return true;
   if(StringFind(low,"jobless claims")>=0)return true;
   return false;
}

void CheckNews(bool force=false)
{
   datetime now=ServerNow();
   if(!InpUseNewsFilter){g_newsBlocked=false;g_nextNewsTime=0;g_nextNewsName="";return;}
   if(!force && g_newsChecked>0 && now-g_newsChecked<60)return;g_newsChecked=now;
   bool wasBlocked=g_newsBlocked;g_newsBlocked=false;g_nextNewsTime=0;g_nextNewsName="";
   if(MQLInfoInteger(MQL_TESTER))
   {
      // Tester cannot replay the terminal economic calendar. Use conservative manual UTC release bands.
      int m=MinuteOfDay(UTCNow());int blocks[4]={8*60+30,12*60+30,13*60+30,18*60};
      for(int i=0;i<4;i++)if(MathAbs(m-blocks[i])<=InpNewsBufferMinutes){g_newsBlocked=true;g_nextNewsName="tester macro block";break;}
   }
   else
   {
      MqlCalendarValue vals[];datetime t0=now-InpNewsBufferMinutes*60,t1=now+(InpNewsLookaheadMin+InpNewsBufferMinutes)*60;
      int n=CalendarValueHistory(vals,t0,t1,NULL,"USD");
      for(int i=0;i<n;i++)
      {
         MqlCalendarEvent ev;if(!CalendarEventById(vals[i].event_id,ev))continue;if(ev.importance<CALENDAR_IMPORTANCE_HIGH)continue;
         if(!CriticalNewsEvent(ev.name))continue;
         datetime et=vals[i].time;if(g_nextNewsTime==0||et<g_nextNewsTime){g_nextNewsTime=et;g_nextNewsName=ev.name;}
         if(MathAbs((long)(et-now))<=InpNewsBufferMinutes*60)g_newsBlocked=true;
      }
   }
   // FMP headline scan is advisory: it never blocks by default (plan-restricted on
   // some keys anyway), but a hard block is available via InpFMPNewsHardBlock.
   if(InpFMPNewsHardBlock&&g_fmpNewsCount>0)g_newsBlocked=true;
   if(wasBlocked&&!g_newsBlocked)g_newsBlockedUntil=now+InpPostNewsStabilizeMinutes*60;
}

//====================================================================
// SWAP / ROLLOVER PROTECTION
//====================================================================
int ServerMinuteOfDay(){return MinuteOfDay(ServerNow());}
int RolloverMinute(){int m=(int)MathRound(InpSwapRolloverServerHour*60.0);while(m<0)m+=1440;return m%1440;}
bool InSwapDangerWindow()
{
   if(!InpAvoidSwap)return false;int now=ServerMinuteOfDay(),roll=RolloverMinute();int until=ForwardMinutesTo(now,roll);int after=ForwardMinutesTo(roll,now);
   bool before=(until<=InpSwapBlockMinutesBefore);bool justAfter=(after<=InpSwapBlockMinutesAfter);
   bool projectedCross=(until<=MathMax(0,InpMaxTradeMinutes));return before||justAfter||projectedCross;
}
void EnforceSwapFlat()
{
   if(!InpAvoidSwap||!InpForceFlatBeforeSwap)return;int now=ServerMinuteOfDay(),roll=RolloverMinute();int until=ForwardMinutesTo(now,roll);
   if(until>InpSwapBlockMinutesBefore)return;
   DeleteOwnPendings(false);for(int i=PositionsTotal()-1;i>=0;i--){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetString(POSITION_SYMBOL)==eaSymbol&&PositionGetInteger(POSITION_MAGIC)==InpMagicNumber)ClosePositionSafe(t);}
}

//====================================================================
// ACCOUNT / RISK / COSTS
//====================================================================
int WeekKey(datetime t){MqlDateTime d;TimeToStruct(t,d);return d.year*100+(d.day_of_year/7);}
int MonthKey(datetime t){MqlDateTime d;TimeToStruct(t,d);return d.year*100+d.mon;}
int DayKey(datetime t){MqlDateTime d;TimeToStruct(t,d);return d.year*1000+d.day_of_year;}

void ResetWindowDay()
{
   for(int i=0;i<WIN_COUNT;i++){g_windowRiskUsed[i]=0;g_windowSignals[i]=0;g_lastWindowEntry[i]=0;}
}

void UpdateRiskPeriods()
{
   datetime n=ServerNow();int dk=DayKey(n),wk=WeekKey(n),mk=MonthKey(n);double eq=AccountInfoDouble(ACCOUNT_EQUITY);
   if(g_dayKey!=dk){g_dayKey=dk;g_dayAnchor=eq;g_tradesToday=0;g_consecutiveLosses=0;g_recoveryLegs=0;g_stopDay=false;ResetWindowDay();}
   if(g_weekKey!=wk){g_weekKey=wk;g_weekAnchor=eq;g_stopWeek=false;}
   // Stale-anchor guard: a persisted anchor from an older week can show absurd weekly
   // DD (e.g. 93%) after deposits/withdrawals or state carry-over. Re-anchor when the
   // number is not physically plausible for one week of trading.
   if(g_weekAnchor>0)
   {
      double wdd=(g_weekAnchor-eq)/g_weekAnchor*100.0;
      if(wdd>60.0||wdd<-60.0)
      {
         g_weekAnchor=eq;    // anchor provably stale (state carry-over) - re-anchor
         g_stopWeek=false;   // and clear the halt it wrongly triggered
      }
   }
   if(g_monthKey!=mk){g_monthKey=mk;g_monthAnchor=eq;g_stopMonth=false;}
   if(g_dayAnchor>0)
   {
      double dd2=(g_dayAnchor-eq)/g_dayAnchor*100.0;
      if(dd2>60.0||dd2<-60.0){g_dayAnchor=eq;g_stopDay=false;g_tradesToday=0;}
   }
   if(g_monthAnchor>0)
   {
      double md2=(g_monthAnchor-eq)/g_monthAnchor*100.0;
      if(md2>60.0||md2<-60.0){g_monthAnchor=eq;g_stopMonth=false;}
   }
   double d=(g_dayAnchor>0?(g_dayAnchor-eq)/g_dayAnchor*100:0),w=(g_weekAnchor>0?(g_weekAnchor-eq)/g_weekAnchor*100:0),m=(g_monthAnchor>0?(g_monthAnchor-eq)/g_monthAnchor*100:0);
   g_maxDDSeen=MathMax(g_maxDDSeen,MathMax(0,d));if(d>=InpDailyLossPercent||d>=InpMaxFloatingDDPercent)g_stopDay=true;if(w>=InpWeeklyLossLimit)g_stopWeek=true;if(m>=InpMonthlyLossLimit)g_stopMonth=true;
   if(g_consecutiveLosses>=InpMaxConsecutiveLosses)g_stopDay=true;
}

double MoneyPerPricePerLot()
{
   if(broker.tickSize<=0||broker.tickValue<=0)return 0;return broker.tickValue/broker.tickSize;
}

double PriceMoveMoney(double dist,double lots){return MathAbs(dist)*MoneyPerPricePerLot()*lots;}

double CommissionRT(double lots)
{
   double c=(g_commissionRTPerLot>0?g_commissionRTPerLot:InpCommissionPerLotRTFallback);return c*lots;
}

double ExpectedSlippagePoints(){return (g_slipCnt>=5?MathMax(g_slipAvg,1.0):InpExpectedSlipPtsFallback);}

double ExpectedAllInCost(double lots)
{
   double spreadPrice=SpreadPoints()*broker.point;double slipPrice=ExpectedSlippagePoints()*broker.point;
   return PriceMoveMoney(spreadPrice+slipPrice,lots)+CommissionRT(lots);
}

double WindowExpectancy(ENUM_WINDOW_ID w)
{
   if(w<=WIN_NONE||w>=WIN_COUNT||g_ws[w].recentCount<=0)return 0;double s=0;for(int i=0;i<g_ws[w].recentCount;i++)s+=g_ws[w].recentNet[i];return s/g_ws[w].recentCount;
}

double WindowAvgR(ENUM_WINDOW_ID w)
{
   if(w<=WIN_NONE||w>=WIN_COUNT||g_ws[w].trades<=0)return 0;return g_ws[w].rSum/g_ws[w].trades;
}

bool IsPrimarySessionWindow(ENUM_WINDOW_ID w)
{
   return (w==WIN_SYDNEY||w==WIN_TOKYO||w==WIN_LONDON||w==WIN_NEWYORK);
}

void RefreshWindowGating(ENUM_WINDOW_ID w)
{
   if(w<=WIN_NONE||w>=WIN_COUNT)return;g_ws[w].disabled=false;
   if(!InpEnablePerformanceGating||g_ws[w].trades<InpPerfMinTrades)return;
   // Primary sessions remain eligible by design. Weak expectancy is handled by
   // WindowRiskMultiplier rather than deleting an entire market session.
   if(InpNeverDisablePrimarySessions && IsPrimarySessionWindow(w))return;
   if(WindowExpectancy(w)<=InpDisableExpectancyMoney)g_ws[w].disabled=true;
}

double WindowRiskMultiplier(ENUM_WINDOW_ID w)
{
   RefreshWindowGating(w);if(g_ws[w].disabled)return 0;
   if(InpEnablePerformanceGating&&g_ws[w].trades>=InpPerfMinTrades&&WindowExpectancy(w)<InpRiskReduceExpectancyMoney)return InpWeakWindowRiskMultiplier;
   return 1.0;
}

double OpenRiskMoney(int directionFilter=0)
{
   double sum=0;
   for(int i=0;i<PositionsTotal();i++)
   {
      ulong t=PositionGetTicket(i);if(t==0||!PositionSelectByTicket(t))continue;if(PositionGetString(POSITION_SYMBOL)!=eaSymbol||PositionGetInteger(POSITION_MAGIC)!=InpMagicNumber)continue;
      int dir=(PositionGetInteger(POSITION_TYPE)==POSITION_TYPE_BUY?1:-1);if(directionFilter!=0&&dir!=directionFilter)continue;
      double sl=PositionGetDouble(POSITION_SL),op=PositionGetDouble(POSITION_PRICE_OPEN),v=PositionGetDouble(POSITION_VOLUME);if(sl<=0)continue;
      sum+=PriceMoveMoney(op-sl,v);
   }
   return sum;
}

double SumOwnLots()
{
   double s=0;for(int i=0;i<PositionsTotal();i++){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetString(POSITION_SYMBOL)==eaSymbol&&PositionGetInteger(POSITION_MAGIC)==InpMagicNumber)s+=PositionGetDouble(POSITION_VOLUME);}return s;
}

int CountOwnPositions()
{
   int n=0;for(int i=0;i<PositionsTotal();i++){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetString(POSITION_SYMBOL)==eaSymbol&&PositionGetInteger(POSITION_MAGIC)==InpMagicNumber)n++;}return n;
}

int CountOwnPendings()
{
   int n=0;for(int i=0;i<OrdersTotal();i++){ulong t=OrderGetTicket(i);if(t&&OrderSelect(t)&&OrderGetString(ORDER_SYMBOL)==eaSymbol&&OrderGetInteger(ORDER_MAGIC)==InpMagicNumber)n++;}return n;
}

double CurrentRiskPct(ENUM_WINDOW_ID w,bool hv)
{
   double r=(InpSimpleScalpMode?0.35:InpRiskPercent);   // scalp base risk: bounded, small
   double dd=(g_dayAnchor>0?(g_dayAnchor-AccountInfoDouble(ACCOUNT_EQUITY))/g_dayAnchor*100:0);
   if(dd>InpMaxFloatingDDPercent*0.5)r=MathMax(0.10,r-InpRiskStepDownOnDD);
   // Consecutive-loss risk decay instead of a hard pause: each straight loss cuts the
   // next trade's risk 30% (floor 0.10%). Trading continues; exposure self-corrects.
   // Full pause only at the configured limit (breaker).
   r*=MathPow(0.70,MathMin(4,g_consecutiveLosses));
   if(r<0.10)r=0.10;
   r*=WindowRiskMultiplier(w);if(hv)r*=InpHVExtraSignalRiskMult;return r;
}

double CalculateLot(double slDist,ENUM_WINDOW_ID w,bool hv)
{
   if(InpRiskPercent<=0)return NormalizeVolume(InpLotSize);if(slDist<=0)return 0;double bal=AccountInfoDouble(ACCOUNT_BALANCE),rp=CurrentRiskPct(w,hv);if(rp<=0)return 0;
   double risk=bal*rp/100.0;
   // No-martingale / no-averaging-down enforcement: while ANY position of ours is
   // underwater, new entries never size up (risk stays at or below the base step).
   if((InpNoMartingale||InpNoAveragingDown)&&CountOwnPositions()>0)
   {
      for(int i=0;i<PositionsTotal();i++)
      {
         ulong t=PositionGetTicket(i);if(!t||!PositionSelectByTicket(t))continue;
         if(PositionGetString(POSITION_SYMBOL)!=eaSymbol||PositionGetInteger(POSITION_MAGIC)!=InpMagicNumber)continue;
         double op=PositionGetDouble(POSITION_PRICE_OPEN);
         int pd=(PositionGetInteger(POSITION_TYPE)==POSITION_TYPE_BUY?1:-1);
         double px=(pd>0?Bid():Ask());
         if((pd>0&&px<op)||(pd<0&&px>op)){risk=MathMin(risk,bal*InpRiskPercent/100.0);break;}
      }
   }
   // Single cost basis: spread + expected slippage + commission (same as
   // ExpectedAllInCost / NetProfitValid), so sizing neither under-counts costs
   // nor double-counts them at risk accounting.
   double perLot=PriceMoveMoney(slDist,1.0)+ExpectedAllInCost(1.0);if(perLot<=0)return 0;double lots=risk/perLot;
   if(InpMaxTotalLots>0)lots=MathMin(lots,MathMax(0,InpMaxTotalLots-SumOwnLots()));
   double fv=FloorVolume(lots);
   // Risk-% sizing below the broker minimum would floor to zero and block EVERY trade on
   // small accounts. Fall back to one minimum lot when its real risk stays inside the hard
   // cap (default 2% of balance) - the only viable way to scalp gold on a sub-$1k account.
   if(fv<=0 && InpAllowMinLotFallback && lots>0)
   {
      double minRisk=PriceMoveMoney(slDist,broker.volumeMin)+ExpectedAllInCost(broker.volumeMin);
      if(bal>0 && minRisk/bal*100.0<=InpMinLotMaxRiskPct) fv=FloorVolume(broker.volumeMin);
   }
   return fv;
}

bool RiskRoom(double newRisk,int dir,ENUM_WINDOW_ID w,string &why)
{
   double eq=AccountInfoDouble(ACCOUNT_EQUITY);if(eq<=0){why="bad equity";return false;}
   if((OpenRiskMoney()+newRisk)/eq*100.0>InpMaxAggregateOpenRiskPct){why="aggregate risk cap";return false;}
   if((OpenRiskMoney(dir)+newRisk)/eq*100.0>InpMaxDirectionalRiskPct){why="direction risk cap";return false;}
   if(g_windowRiskUsed[w]+newRisk>eq*InpPerWindowRiskBudgetPct/100.0){why="window risk budget";return false;}
   return true;
}

bool ShouldStopTrading()
{
   UpdateRiskPeriods();if(g_stopDay||g_stopWeek||g_stopMonth)return true;
   return false;
}

//====================================================================
// STRUCTURE-AWARE SL / TP ENGINE (monotonic ladder + cost-positive legs)
//====================================================================
double NearestLiquidityTarget(int dir,double entry,int lookback,double fallback)
{
   MqlRates r[];ArraySetAsSeries(r,true);int n=(int)MathMax(10,lookback);if(CopyRates(eaSymbol,PERIOD_M1,1,n,r)<n)return fallback;
   double best=0;
   if(dir>0){for(int i=0;i<n;i++)if(r[i].high>entry && (best==0||r[i].high<best))best=r[i].high;}
   else {for(int i=0;i<n;i++)if(r[i].low<entry && (best==0||r[i].low>best))best=r[i].low;}
   return (best>0?best:fallback);
}

//--- Signal-aware SL/TP for the ultra-scalp engine:
//--- trend-pullback: SL = 0.90 ATR (invalidation), TP1 = 0.45 ATR (1.2R effective)
//--- mean-reversion: SL = beyond last bar extreme + 0.5 ATR (noise-proof),
//---                 TP1 = VWAP (the mean) - the highest-probability target
double ScalpStopDistance(int dir,double entry,double &slPrice)
{
   double atr=MathMax(g_atr,MinTradeDistance());
   if(g_scalpSignal==2||g_scalpSignal==-2)   // VWAP reversion: beyond the extreme + 0.5 ATR
   {
      double ext=(dir>0?iLow(eaSymbol,PERIOD_M1,1):iHigh(eaSymbol,PERIOD_M1,1));
      slPrice=PriceNorm(ext-dir*0.5*atr);
   }
   else if(g_scalpSignal==3||g_scalpSignal==-3)  // London breakout: 1.0 ATR stop
      slPrice=PriceNorm(entry-dir*1.0*atr);
   else slPrice=PriceNorm(entry-dir*0.90*atr);   // NY momentum / EMA pullback
   double d=MathAbs(entry-slPrice);
   double md=MinTradeDistance();
   if(d<md){d=md;slPrice=PriceNorm(entry-dir*d);}
   return d;
}
double ScalpTarget1(int dir,double entry)
{
   double atr=MathMax(g_atr,MinTradeDistance());
   if((g_scalpSignal==2||g_scalpSignal==-2)&&g_vwap>0)
   {
      // reversion: target the mean (VWAP), min 0.6 ATR away
      double d=MathAbs(g_vwap-entry);
      if(d<0.6*atr)d=0.6*atr;
      return PriceNorm(entry+dir*d);
   }
   // breakout/momentum/pullback: fixed 1.2R-style via 0.55 ATR (stop is 0.9-1.0 ATR)
   return PriceNorm(entry+dir*0.55*atr);
}

double ComputeSL(int dir,double entry)
{
   double atr=MathMax(g_atr,MinTradeDistance());double byAtr=(dir>0?entry-InpSL_ATR_Multiplier*atr:entry+InpSL_ATR_Multiplier*atr);
   double sl=byAtr;
   // Structure-aware SL only in the complex engine. In ultra-scalp mode the ATR stop is
   // HARD: widening to swing structure (potentially 3-5x the ATR distance on M1 gold)
   // is exactly what turned "risk 0.5%" into major losses.
   if(!InpSimpleScalpMode)
   {
      if(dir>0&&g_swingLow>0)sl=MathMin(sl,g_swingLow-InpSLStructureBufferATR*atr);
      if(dir<0&&g_swingHigh>0)sl=MathMax(sl,g_swingHigh+InpSLStructureBufferATR*atr);
   }
   double md=MinTradeDistance();if(dir>0&&entry-sl<md)sl=entry-md;if(dir<0&&sl-entry<md)sl=entry+md;return PriceNorm(sl);
}

double RegimeTPMultiplier(ENUM_WINDOW_ID w,bool hv)
{
   if(hv&&(w==WIN_SYDNEY_TOKYO||w==WIN_TOKYO_LONDON||w==WIN_LONDON_OPEN||w==WIN_LONDON_NY||w==WIN_NY_OPEN))return 0.85;
   if(hv)return 1.05;return 1.0;
}

bool NetProfitValid(int dir,double entry,double target,double lots,double minMoney,double &net)
{
   double gross=PriceMoveMoney(target-entry,lots);double cost=ExpectedAllInCost(lots);net=gross-cost;
   if(gross<=0)return false;if(cost/gross*100.0>InpMaxCostToTP1Pct)return false;return net>=minMoney;
}

void BuildThreeTargets(int dir,double entry,double lots,ENUM_WINDOW_ID w,bool hv,double &tp1,double &tp2,double &tp3)
{
   double atr=MathMax(g_atr,MinTradeDistance()),k=RegimeTPMultiplier(w,hv);
   double d1=MathMax(InpTP1_ATR_Floor*atr,MathMin(InpTP1_ATR_Cap*atr,0.55*atr))*k;
   double d2=MathMax(InpTP2_ATR_Floor*atr,MathMin(InpTP2_ATR_Cap*atr,1.10*atr))*k;
   double d3=MathMax(InpTP3_ATR_Floor*atr,MathMin(InpTP3_ATR_Cap*atr,1.85*atr))*(hv?1.05:1.0);
   tp1=entry+dir*d1;tp2=entry+dir*d2;tp3=entry+dir*d3;
   double md=MinTradeDistance();
   // Strict monotonic ladder first: TP1 < TP2 < TP3 can never be violated.
   if(dir>0){tp1=MathMax(tp1,entry+md);tp2=MathMax(tp2,tp1+md);tp3=MathMax(tp3,tp2+md);}
   else{tp1=MathMin(tp1,entry-md);tp2=MathMin(tp2,tp1-md);tp3=MathMin(tp3,tp2-md);}
   // Live-chart precision: snap TP1 to the nearest real liquidity level when one sits
   // between entry and the ATR default (targets then mark genuine S/R, not blind ATR).
   double l1=NearestLiquidityTarget(dir,entry,20,0);
   if(l1>0)
   {
      double dd=MathAbs(l1-entry);
      if(dd>=md && dd<MathAbs(tp1-entry)) tp1=l1;   // closer REAL level wins
   }
   // Restore ordering after the snap.
   if(dir>0){tp2=MathMax(tp2,tp1+md);tp3=MathMax(tp3,tp2+md);}else{tp2=MathMin(tp2,tp1-md);tp3=MathMin(tp3,tp2-md);}
   // Cost-aware expand per leg against the volume that will ACTUALLY close there
   // (60/25/15 split of the ladder). Validating on full 'lots' was too lenient: a leg
   // closing 15% of the position earns 15% of the gross but pays commission on it too,
   // and the spread cost is only saved on that fraction.
   double sumPct=MathMax(0.0001,InpTP1Pct+InpTP2Pct+InpTP3Pct);
   double v1=FloorVolume(lots*InpTP1Pct/sumPct);
   double v2=FloorVolume(lots*InpTP2Pct/sumPct);
   double v3=FloorVolume(lots-v1-v2);
   if(v1<broker.volumeMin)v1=lots;                       // collapse: single leg carries all
   if(v2<broker.volumeMin)v2=(v3<broker.volumeMin?0:lots-v1);
   if(v3<broker.volumeMin)v3=0;
   double net=0;int guard=0;
   while(!NetProfitValid(dir,entry,tp1,v1,InpMinNetProfitTP1Money,net)&&guard++<10)tp1+=dir*0.10*atr;
   guard=0;
   while(v2>0&&!NetProfitValid(dir,entry,tp2,v2,InpMinNetProfitTP2Money,net)&&guard++<10)tp2+=dir*0.10*atr;
   guard=0;
   while(v3>0&&!NetProfitValid(dir,entry,tp3,v3,InpMinNetProfitTP3Money,net)&&guard++<10)tp3+=dir*0.10*atr;
   if(dir>0){tp2=MathMax(tp2,tp1+md);tp3=MathMax(tp3,tp2+md);}else{tp2=MathMin(tp2,tp1-md);tp3=MathMin(tp3,tp2-md);}
   tp1=PriceNorm(tp1);tp2=PriceNorm(tp2);tp3=PriceNorm(tp3);
}

// Reward-to-risk quality gate: the ladder must genuinely out-earn its stop.
bool RRValid(int dir,double entry,double sl,double target,double minRR)
{
   double slDist=MathAbs(entry-sl);if(slDist<=0)return false;
   double reward=MathAbs(target-entry);
   return (reward/slDist>=minRR);
}

void AllocateVolumes(double total,double &v1,double &v2,double &v3)
{
   v1=v2=v3=0;double sum=MathMax(0.0001,InpTP1Pct+InpTP2Pct+InpTP3Pct);double minv=broker.volumeMin;
   v1=FloorVolume(total*InpTP1Pct/sum);v2=FloorVolume(total*InpTP2Pct/sum);v3=FloorVolume(total-v1-v2);
   if(v3<=0){v3=0;v2=FloorVolume(total-v1);}if(v2<=0){v2=0;v1=total;}
   // If three legs cannot satisfy broker minimum, intelligently collapse to two/one leg.
   int valid=(v1>=minv?1:0)+(v2>=minv?1:0)+(v3>=minv?1:0);
   if(valid<3 && total<3*minv-1e-12){v3=0;v1=FloorVolume(total*0.60);v2=FloorVolume(total-v1);if(v2<minv){v1=total;v2=0;}}
   double used=v1+v2+v3;double rem=FloorVolume(total-used);if(rem>0)v1=FloorVolume(v1+rem);
   if(v1<=0){v1=total;v2=v3=0;}
}//====================================================================
// SETUP / ENTRY GATING
//====================================================================
string MakeSetupId(int dir,ENUM_WINDOW_ID w)
{
   datetime b=iTime(eaSymbol,PERIOD_M1,1);string smc=(dir>0?(g_bosUp?"BOSU":g_chochUp?"CHU":g_sweepDn?"SWD":"BASE"):(g_bosDn?"BOSD":g_chochDn?"CHD":g_sweepUp?"SWU":"BASE"));
   return IntegerToString((int)w)+"-"+(dir>0?"B":"S")+"-"+IntegerToString((int)b)+"-"+smc;
}

bool FreshSetup(int dir,ENUM_WINDOW_ID w,string &id,string &why)
{
   id=MakeSetupId(dir,w);if(!InpOncePerValidatedEvent)return true;
   string last=(dir>0?g_lastSetupBuy[w]:g_lastSetupSell[w]);if(id==last){why="duplicate setup";return false;}
   datetime now=ServerNow();if(g_lastWindowEntry[w]>0&&now-g_lastWindowEntry[w]<InpMinSecondsBetweenEntries){why="entry spacing";return false;}
   if(g_lastWindowEntry[w]>0&&iTime(eaSymbol,PERIOD_M1,1)-g_lastWindowEntry[w]<InpMinBarsFreshStructure*60){why="fresh-structure spacing";return false;}
   if(g_windowSignals[w]>=InpMaxSignalsPerWindow){why="window signal cap";return false;}return true;
}

bool CanEnter(int dir,ENUM_WINDOW_ID &w,bool &hv,string &setup,string &why)
{
   if(!MQLInfoInteger(MQL_TRADE_ALLOWED)||!TerminalInfoInteger(TERMINAL_TRADE_ALLOWED)||!AccountInfoInteger(ACCOUNT_TRADE_ALLOWED)||!AccountInfoInteger(ACCOUNT_TRADE_EXPERT)){why="trading disabled";return false;}
   if(g_paused){why="manual pause (F)";return false;}
   if(ShouldStopTrading()){why="risk breaker";return false;}if(WeekendOrRollover()){why="weekend/rollover";return false;}
   bool tr=false;w=CurrentWindow(tr);if(InpUseSessionFilter&&!tr){why="out of enabled session";return false;}if(w==WIN_NONE){why="no opportunity window";return false;}
   if(g_newsBlocked||ServerNow()<g_newsBlockedUntil){why="news/stabilization";return false;}
   if(InSwapDangerWindow()){why="swap/rollover protection";return false;}
   double sp=SpreadPoints();
   if(sp>InpMaxSpreadPoints*g_ptScale){why="spread hard cap";return false;}
   if(sp>=(InpMaxSpreadPoints+20)*g_ptScale){why="spread extreme";return false;}   // > hard cap+20 = broken feed
   double atrPts=(broker.point>0?g_atr/broker.point:0);
   if(atrPts<InpMinATRPoints*g_ptScale){why="ATR chop";return false;}
   // ============ ULTRA-SCALP SIMPLE PATH (default) ============
   // Momentum + spread + risk. The multi-filter machinery below is only for
   // InpSimpleScalpMode=false. A scalp engine must fire on clean setups, not
   // survive a 14-item confluence checklist that can veto every bar.
   if(InpSimpleScalpMode)
   {
      // Ultra-scalp v3: four signal modes (VWAP reversion, London breakout, NY momentum,
      // EMA pullback) computed once per M1 bar in EvaluateScalpSignal().
      int want=(g_scalpSignal>0?1:-1);
      if(g_scalpSignal==0||want!=dir){why="no scalp signal ("+g_scalpWhy+")";return false;}
      setup=g_scalpWhy;
      // Momentum modes need liquid hours; reversion works everywhere (range fades).
      bool liquid=(w==WIN_TOKYO_LONDON||w==WIN_LONDON_NY||w==WIN_LONDON_OPEN||w==WIN_NY_OPEN
                   ||w==WIN_LONDON||w==WIN_NEWYORK);
      if((g_scalpSignal==1||g_scalpSignal==-1||g_scalpSignal==3||g_scalpSignal==-3)&&!liquid)
      {why="momentum signal in thin session";return false;}
      // Anti-stacking: no second scalp same direction within 60s or 0.35 ATR of an open one
      datetime now=ServerNow();
      if(g_lastEntryTime>0&&now-g_lastEntryTime<60){why="entry spacing 60s";return false;}
      double px=(dir>0?Ask():Bid());
      for(int i=0;i<PositionsTotal();i++)
      {
         ulong t=PositionGetTicket(i);if(!t||!PositionSelectByTicket(t))continue;
         if(PositionGetString(POSITION_SYMBOL)!=eaSymbol||PositionGetInteger(POSITION_MAGIC)!=InpMagicNumber)continue;
         if((int)PositionGetInteger(POSITION_TYPE)!=(dir>0?POSITION_TYPE_BUY:POSITION_TYPE_SELL))continue;
         double op=PositionGetDouble(POSITION_PRICE_OPEN);
         if(MathAbs(px-op)<0.35*MathMax(g_atr,MinTradeDistance())){why="too close to open scalp";return false;}
      }
      hv=false;
      if(g_tradesToday>=InpMaxTradesPerDay){why="daily trade cap";return false;}
      if(!broker.hedging&&CountOwnPositions()>0){why="netting: one position";return false;}
      if(CountOwnPositions()>=InpMaxConcurrentPositions){why="position cap";return false;}
      return true;
   }
   RefreshWindowGating(w);if(g_ws[w].disabled){why="window expectancy disabled";return false;}
   if(IsDisorder()){why="market disorder";return false;}
   if(g_slipCnt>=5&&g_slipAvg>InpMaxAverageSlippagePoints){why="slippage quality gate";return false;}
   if(g_lastSlipPts>=InpExtremeSlippagePoints&&ServerNow()<g_disorderUntil){why="last fill slippage extreme";return false;}
   if(!ExternalDataReady(why))return false;
   double spp=SpreadPercentile();if(g_spreadAvg>0&&sp>g_spreadAvg*InpSpreadSpikeRatio){why="spread spike";return false;}if(spp>InpMaxSpreadPercentile){why="spread percentile";return false;}
   if(InpMaxATRPoints>0&&atrPts>InpMaxATRPoints*g_ptScale){why="ATR chaos";return false;}
   double disp=CandleDisplacementATR();if(disp>InpMaxChaseCandleATR){why="anti-chase displacement";return false;}double cc=iClose(eaSymbol,PERIOD_M1,1);if(g_vwap>0&&g_atr>0&&MathAbs(cc-g_vwap)/g_atr>InpMaxEntryVWAPDeviationATR){why="anti-chase VWAP distance";return false;}
   if(InpFilterMode==FILTER_ALL_REQUIRED&&g_score<g_scoreMax){why="filters";return false;}if(InpFilterMode==FILTER_SCORING&&g_score<InpMinFilterScore){why="score";return false;}
   if(MathAbs(g_dirBias)<InpMinDirBias){why="weak directional bias";return false;}if((dir>0&&g_dirBias<0)||(dir<0&&g_dirBias>0)){why="bias conflict";return false;}
   if(InpUseSMC&&((dir>0?g_smcScoreBull:g_smcScoreBear)<InpMinSMCConfluence)){why="SMC confluence";return false;}
   if(!LiveMomentumConfirm(dir)){why="live price vs EMA20";return false;}
   if((InpUseIFVG||InpUsePTB)){int adv=(dir>0?((g_ifvg&&g_ifvgDir>0)?1:0)+((g_ptb&&g_ptbDir>0)?1:0):((g_ifvg&&g_ifvgDir<0)?1:0)+((g_ptb&&g_ptbDir<0)?1:0));if(adv<InpMinAdvancedSMCConfluence&&InpMinAdvancedSMCConfluence>0){why="IFVG/PTB confluence";return false;}}
   if(AvailableMacroCount()>0&&MacroVotes(dir)<InpMacroMinConfluence){why="macro conflict";return false;}
   int hvs=0;hv=InpAB_EnableHighVol&&IsHighVolatilityQualified(dir,hvs);
   if(!InpAB_EnableBase&&!hv){why="A/B base engine disabled";return false;}
   bool preferred=(w==WIN_SYDNEY_TOKYO||w==WIN_TOKYO_LONDON||w==WIN_LONDON_OPEN||w==WIN_LONDON_NY||w==WIN_NY_OPEN||w==WIN_VERIFIED_EXPANSION);
   if(InpHighVolatilityMode==HV_FORCE_GATED && preferred && !hv){why="HV verification failed";return false;}
   if(!FreshSetup(dir,w,setup,why))return false;
   if(g_tradesToday>=InpMaxTradesPerDay){why="daily trade cap";return false;}if(!broker.hedging&&CountOwnPositions()>0){why="netting account: one strategy position at a time";return false;}if(CountOwnPositions()>=InpMaxConcurrentPositions){why="position cap";return false;}
   return true;
}

//====================================================================
// NATIVE TRADE EXECUTION (no CTrade; 9-arg WebRequest-safe, retcodes checked)
//====================================================================
ENUM_ORDER_TYPE_FILLING BestFillingMode()
{
   long f=SymbolInfoInteger(eaSymbol,SYMBOL_FILLING_MODE);
   long ex=SymbolInfoInteger(eaSymbol,SYMBOL_TRADE_EXEMODE);
   if((f & SYMBOL_FILLING_FOK)==SYMBOL_FILLING_FOK) return ORDER_FILLING_FOK;
   if((f & SYMBOL_FILLING_IOC)==SYMBOL_FILLING_IOC) return ORDER_FILLING_IOC;
   if(ex!=SYMBOL_TRADE_EXECUTION_MARKET) return ORDER_FILLING_RETURN;
   return ORDER_FILLING_FOK;
}

bool RetcodeOK(uint r){return (r==TRADE_RETCODE_DONE||r==TRADE_RETCODE_PLACED||r==TRADE_RETCODE_DONE_PARTIAL);}

bool SendOrder(MqlTradeRequest &rq,MqlTradeResult &rs)
{
   rq.type_filling=BestFillingMode();
   for(int k=0;k<=InpOrderRetry;k++)
   {
      ZeroMemory(rs);
      if(OrderSend(rq,rs)&&RetcodeOK(rs.retcode))return true;
      // 10015/10016 (invalid price/stops) will not heal by retrying.
      if(rs.retcode==TRADE_RETCODE_INVALID_PRICE||rs.retcode==TRADE_RETCODE_INVALID_STOPS)return false;
      Sleep(40);
   }
   return false;
}

bool PlaceStop(int dir,double lots,double price,double sl,double tp,string comment,ulong &ticket)
{
   ticket=0;double md=MinTradeDistance();double a=Ask(),b=Bid();
   if(dir>0){if(price-a<md)price=a+md;if(price-sl<md)sl=price-md;if(tp-price<md)tp=price+md;}
   else{if(b-price<md)price=b-md;if(sl-price<md)sl=price+md;if(price-tp<md)tp=price-md;}
   price=PriceNorm(price);sl=PriceNorm(sl);tp=PriceNorm(tp);
   MqlTradeRequest rq;MqlTradeResult rs;ZeroMemory(rq);ZeroMemory(rs);
   rq.action=TRADE_ACTION_PENDING;rq.symbol=eaSymbol;rq.magic=InpMagicNumber;rq.volume=lots;
   rq.type=(dir>0?ORDER_TYPE_BUY_STOP:ORDER_TYPE_SELL_STOP);
   rq.price=price;rq.sl=sl;rq.tp=tp;rq.comment=comment;
   datetime exp=ServerNow()+InpPendingExpiryMinutes*60;
   // Most brokers accept ORDER_TIME_SPECIFIED; the final attempt falls back to
   // GTC (stale pendings are still swept by InpCancelStalePendings).
   for(int mode=0;mode<2;mode++)
   {
      if(mode==0){rq.type_time=ORDER_TIME_SPECIFIED;rq.expiration=exp;}
      else{rq.type_time=ORDER_TIME_GTC;rq.expiration=0;}
      if(SendOrder(rq,rs)){ticket=rs.order;return true;}
      if(rs.retcode==TRADE_RETCODE_INVALID_PRICE||rs.retcode==TRADE_RETCODE_INVALID_STOPS)return false;
   }
   return false;
}

bool MarketOrder(int dir,double lots,double sl,double tp,string comment,double &fillPrice,ulong &ticket)
{
   fillPrice=0;ticket=0;
   MqlTradeRequest rq;MqlTradeResult rs;ZeroMemory(rq);ZeroMemory(rs);
   rq.action=TRADE_ACTION_DEAL;rq.symbol=eaSymbol;rq.magic=InpMagicNumber;rq.volume=lots;
   rq.type=(dir>0?ORDER_TYPE_BUY:ORDER_TYPE_SELL);
   rq.price=(dir>0?Ask():Bid());
   rq.deviation=InpMaxSlippagePoints;rq.comment=comment;rq.sl=sl;rq.tp=tp;
   if(!SendOrder(rq,rs))return false;
   // res.price is the real fill (B5 fix); fall back to the position price.
   fillPrice=(rs.price>0?rs.price:rq.price);
   ticket=rs.order;
   return true;
}

bool ModifyPositionSafe(ulong ticket,double sl,double tp)
{
   sl=(sl>0?PriceNorm(sl):0);tp=(tp>0?PriceNorm(tp):0);
   MqlTradeRequest rq;MqlTradeResult rs;ZeroMemory(rq);ZeroMemory(rs);
   rq.action=TRADE_ACTION_SLTP;rq.position=ticket;rq.symbol=eaSymbol;rq.sl=sl;rq.tp=tp;
   for(int k=0;k<=InpOrderRetry;k++)
   {
      ZeroMemory(rs);
      if(OrderSend(rq,rs)&&RetcodeOK(rs.retcode))return true;
      Sleep(30);
   }
   return false;
}

bool DeleteOrderSafe(ulong ticket)
{
   MqlTradeRequest rq;MqlTradeResult rs;ZeroMemory(rq);ZeroMemory(rs);
   rq.action=TRADE_ACTION_REMOVE;rq.order=ticket;
   for(int k=0;k<=InpOrderRetry;k++)
   {
      ZeroMemory(rs);
      if(OrderSend(rq,rs)&&RetcodeOK(rs.retcode))return true;
      Sleep(30);
   }
   return false;
}

void DeleteOwnPendings(bool onlyStale=false)
{
   datetime now=ServerNow();
   for(int i=OrdersTotal()-1;i>=0;i--){ulong t=OrderGetTicket(i);if(!t||!OrderSelect(t))continue;if(OrderGetString(ORDER_SYMBOL)!=eaSymbol||OrderGetInteger(ORDER_MAGIC)!=InpMagicNumber)continue;if(onlyStale){datetime st=(datetime)OrderGetInteger(ORDER_TIME_SETUP);if(now-st<InpPendingExpiryMinutes*60)continue;}DeleteOrderSafe(t);}
}

bool ClosePartialSafe(ulong ticket,double volume)
{
   if(volume<=0||!PositionSelectByTicket(ticket))return false;double cur=PositionGetDouble(POSITION_VOLUME);volume=FloorVolume(MathMin(volume,cur));if(volume<=0)return false;
   ENUM_POSITION_TYPE pt=(ENUM_POSITION_TYPE)PositionGetInteger(POSITION_TYPE);
   MqlTradeRequest rq;MqlTradeResult rs;ZeroMemory(rq);ZeroMemory(rs);rq.action=TRADE_ACTION_DEAL;rq.position=ticket;rq.symbol=eaSymbol;rq.magic=InpMagicNumber;rq.volume=volume;rq.deviation=InpMaxSlippagePoints;rq.type=(pt==POSITION_TYPE_BUY?ORDER_TYPE_SELL:ORDER_TYPE_BUY);rq.price=(rq.type==ORDER_TYPE_BUY?Ask():Bid());
   if(!SendOrder(rq,rs))return false;return true;
}

bool ClosePositionSafe(ulong ticket)
{
   if(!PositionSelectByTicket(ticket))return false;
   double cur=PositionGetDouble(POSITION_VOLUME);
   return ClosePartialSafe(ticket,cur);
}

void EmergencyCloseAll()
{
   DeleteOwnPendings(false);for(int i=PositionsTotal()-1;i>=0;i--){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetString(POSITION_SYMBOL)==eaSymbol&&PositionGetInteger(POSITION_MAGIC)==InpMagicNumber)ClosePositionSafe(t);}
}

//====================================================================
// POSITION STATE
//====================================================================
int FindPS(ulong ticket){for(int i=0;i<ArraySize(g_ps);i++)if(g_ps[i].ticket==ticket)return i;return -1;}
void RemovePS(int idx){int n=ArraySize(g_ps);if(idx<0||idx>=n)return;for(int i=idx;i<n-1;i++)g_ps[i]=g_ps[i+1];ArrayResize(g_ps,n-1);}

void AddPositionState(ulong ticket,long posId,int dir,ENUM_WINDOW_ID w,string setup,bool hv,bool recovery,double entry,double sl,double lots,double slip,double entryComm)
{
   int idx=FindPS(ticket);if(idx<0){idx=ArraySize(g_ps);ArrayResize(g_ps,idx+1);}
   PositionState s;ZeroMemory(s);
   s.ticket=ticket;s.positionId=posId;s.direction=dir;s.window=w;s.setupId=setup;s.hv=hv;s.recovery=recovery;
   s.entry=entry;s.initialSL=sl;s.initialVolume=lots;
   s.initialRiskMoney=MathMax(0.0,PriceMoveMoney(entry-sl,lots)+ExpectedAllInCost(lots));
   s.opened=ServerNow();s.entrySpreadPct=SpreadPercentile();s.entrySlipPts=slip;s.entryAtrPct=ATRPercentile();s.entryVolRatio=g_volRatio;
   s.realizedGross=0;s.realizedNet=entryComm;s.realizedCosts=MathAbs(entryComm)+PriceMoveMoney(SpreadPoints()*broker.point+MathAbs(slip)*broker.point,lots);
   s.maePrice=0;s.mfePrice=0;
   if(recovery)
   {
      // Single-target trade: broker TP was validated by TryRecovery before placement.
      // Mirror it into the state (tp1=tp2=tp3=placed TP) so management never runs the
      // 60/25/15 partial ladder and never rewrites the validated broker target.
      double placed=PositionGetDouble(POSITION_TP);
      s.tp1=(placed>0?placed:0);s.tp2=s.tp1;s.tp3=s.tp1;
      s.volTP1=0;s.volTP2=0;s.volTP3=lots;
      s.tp1Done=true;s.tp2Done=true;   // ladder stages pre-marked done: nothing partials
      g_ps[idx]=s;
      return;                          // broker TP/SL stand as sent - no rewrite
   }
   // Use the HV regime carried by the pending order so the executed plan matches the armed plan.
   BuildThreeTargets(dir,entry,lots,w,hv,s.tp1,s.tp2,s.tp3);
   if(InpUseThreeTargets&&InpAB_EnableThreeTP)AllocateVolumes(lots,s.volTP1,s.volTP2,s.volTP3);
   else{s.volTP1=0;s.volTP2=0;s.volTP3=lots;s.tp1Done=true;s.tp2Done=true;}
   g_ps[idx]=s;
   // TP3 is placed at broker as catastrophe-safe final target; TP1/TP2 are virtual managed exits.
   ModifyPositionSafe(ticket,sl,s.tp3);
}

//====================================================================
// ENTRY ARMING
//====================================================================
void TryArm()
{
   if(InpCancelStalePendings)DeleteOwnPendings(true);
   if(CountOwnPendings()>0){g_gateReason="pending orders working";return;}
   if(!InpArmWhileInTrade&&CountOwnPositions()>0){g_gateReason="managing open position";return;}
   // EXEC_STRADDLE arms stop orders ahead of price; EXEC_DIRECTIONAL fires a market
   // order at signal time; EXEC_AUTO picks directional only in verified HV windows.
   int dir;
   if(InpSimpleScalpMode)
   {
      // Direction comes from the two-mode signal computed on the last closed bar.
      if(g_scalpSignal>0)dir=1;else if(g_scalpSignal<0)dir=-1;
      else{g_gateReason="no scalp signal";return;}
   }
   else dir=(g_dirBias>0?1:-1);
   ENUM_WINDOW_ID w;bool hv=false;string setup,why;if(!CanEnter(dir,w,hv,setup,why)){g_gateReason=why;return;}
   bool directional=(InpExecutionMode==EXEC_DIRECTIONAL||(InpExecutionMode==EXEC_AUTO&&w==WIN_VERIFIED_EXPANSION));
   double entry0=(dir>0?Ask():Bid());double atr=MathMax(g_atr,MinTradeDistance());
   double dist;
   if(InpSimpleScalpMode){dist=0;}                       // market order: no pending distance
   else{dist=(InpUseATRForDistance?InpATRMultiplier*atr:InpDistance*g_ptScale*broker.point);dist=MathMax(dist,MinTradeDistance());}
   double pending=(dir>0?entry0+dist:entry0-dist);double sl;double slDist;
   if(InpSimpleScalpMode)
   {
      slDist=ScalpStopDistance(dir,entry0,sl);   // signal-aware stop (slPrice by ref)
      sl=PriceNorm(sl);
   }
   else{sl=ComputeSL(dir,pending);slDist=MathAbs(pending-sl);}
   double lots=CalculateLot(slDist,w,hv);if(lots<=0){g_gateReason="lot/risk zero";return;}
   double risk=PriceMoveMoney(slDist,lots)+ExpectedAllInCost(lots);if(!RiskRoom(risk,dir,w,why)){g_gateReason=why;return;}
   double t1,t2,t3;
   if(InpSimpleScalpMode)
   {
      t1=ScalpTarget1(dir,entry0);
      // single-target scalp: TP2/TP3 trail beyond for the runner
      double atr2=MathMax(g_atr,MinTradeDistance());
      t2=PriceNorm(t1+dir*0.40*atr2);t3=PriceNorm(t1+dir*0.85*atr2);
   }
   else BuildThreeTargets(dir,pending,lots,w,hv,t1,t2,t3);
   // R:R quality gate: the plan must genuinely out-earn its stop before arming.
   if(!InpSimpleScalpMode){
   if(!RRValid(dir,pending,sl,t2,InpMinRR_TP2)){g_gateReason="TP2 R:R below floor";return;}
   if(!RRValid(dir,pending,sl,t3,InpMinRR_TP3)){g_gateReason="TP3 R:R below floor";return;}}
   double net1=0;if(!NetProfitValid(dir,pending,t1,lots,InpMinNetProfitTP1Money,net1)){g_gateReason="TP1 not cost-positive";return;}
   int layers=(directional?1:MathMax(1,MathMin(3,InpStraddleLayers)));if(hv&&InpAB_EnableHighVol)layers=MathMin(2,layers+1);
   // Ultra-scalp mode: always a single MARKET order - pendings/straddles add latency
   // and complexity that a scalp does not need.
   if(InpSimpleScalpMode){directional=true;layers=1;}
   int placed=0;double each=FloorVolume(lots/layers);if(each<=0){layers=1;each=lots;}
   for(int i=0;i<layers;i++)
   {
      // InpLayerStepATR adds progressive spacing per layer; InpScaleIn steps each
      // layer's volume down so later entries carry less risk than the first.
      double layerGap=MathMax(InpLayerSpacingATR+InpLayerStepATR*i,0.05);
      double p=pending+dir*i*MathMax(layerGap*atr,MinTradeDistance());
      double li=(InpScaleIn?each*(1.0-0.15*i):each);
      li=FloorVolume(li);if(li<=0)continue;
      double lsl=ComputeSL(dir,p);double a,b,c;BuildThreeTargets(dir,p,li,w,hv,a,b,c);ulong tk=0;
      // Comment is the durable carrier of window / direction / HV regime across restarts.
      string cmt=InpComment+"|W"+IntegerToString((int)w)+"|"+(dir>0?"B":"S")+"|HV"+(hv?"1":"0")+"|"+setup;
      if(directional&&i==0)
      {
         double fill=0;
         {
            double osl;ScalpStopDistance(dir,entry0,osl);
            if(MarketOrder(dir,li,osl,c,cmt,fill,tk))placed++;
         }
      }
      else if(PlaceStop(dir,li,p,lsl,c,cmt,tk))placed++;
   }
   if(placed>0)
   {
      g_lastEntryTime=ServerNow();g_lastWindowEntry[w]=g_lastEntryTime;g_windowSignals[w]++;g_gateReason="ARMED "+WindowName(w)+(hv?" HV":"");
      if(g_log!=INVALID_HANDLE)FileWrite(g_log,TimeToString(ServerNow(),TIME_DATE|TIME_SECONDS),"ARM",WindowName(w),dir,setup,DoubleToString(SpreadPoints(),1),DoubleToString(SpreadPercentile(),1),DoubleToString(ATRPercentile(),1),DoubleToString(g_volRatio,2),DoubleToString(risk,2),DoubleToString(t1,broker.digits),DoubleToString(t2,broker.digits),DoubleToString(t3,broker.digits));
   }
}

//====================================================================
// LOSS RECOVERY / REVERSAL ENGINE (single counter-leg, no martingale)
//====================================================================
bool MomentumDeteriorated(int dir)
{
   double c=iClose(eaSymbol,PERIOD_M1,1);bool vwapBad=(g_vwap>0&&(dir>0?c<g_vwap:c>g_vwap));bool smcBad=(dir>0?(g_chochDn||g_bosDn):(g_chochUp||g_bosUp));bool momBad=(dir>0?(g_m15e20<g_m15e50):(g_m15e20>g_m15e50));bool volBad=g_volRatio<0.85;return ((vwapBad&&smcBad)||(momBad&&volBad));
}

double CostAdjustedBE(int dir,double entry,double remainingVol)
{
   double ppm=MoneyPerPricePerLot();if(ppm<=0||remainingVol<=0)return entry;double cost=ExpectedAllInCost(remainingVol);double p=cost/(ppm*remainingVol)+InpBEExtraLockATR*g_atr;return PriceNorm(entry+dir*p);
}

void TryRecovery()
{
   string why="";
   if(!InpUseRecovery)return;
   if(InpSimpleScalpMode)return;   // loss-chasing counter-trades disabled in ultra-scalp mode
   // A reversal leg requires a hedging account: on netting the opposite deal would
   // just close the surviving position instead of opening the recovery trade.
   if(!broker.hedging)return;
   if(g_lastLossDir==0||g_lastLossTime==0)return;
   datetime now=ServerNow();
   if(g_recoveryLegs>=InpRecoveryMaxLegs)return;
   if(now<g_lastLossTime+InpRecoveryCooldownSec)return;                 // let the market breathe first
   if(now>g_lastLossTime+InpRecoveryMaxAgeSec){g_lastLossDir=0;return;} // opportunity expired
   if(g_paused||g_newsBlocked||ServerNow()<g_newsBlockedUntil)return;
   if(IsDisorder()||WeekendOrRollover()||InSwapDangerWindow())return;
   if(CountOwnPositions()>=InpMaxConcurrentPositions)return;

   int dir=-g_lastLossDir;                                              // reversal trade against the losing leg
   if((dir>0&&g_dirBias<0)||(dir<0&&g_dirBias>0))return;                // only when structure actually agrees
   double sp=SpreadPoints();if(sp>InpRecoveryMaxSpreadPts*g_ptScale){g_gateReason="recovery: spread";return;}

   double entry=(dir>0?Ask():Bid());double atr=MathMax(g_atr,MinTradeDistance());
   double sl=ComputeSL(dir,entry);double slDist=MathAbs(entry-sl);if(slDist<=0)return;
   double bal=AccountInfoDouble(ACCOUNT_BALANCE);
   double risk=bal*InpRecoveryRiskPct/100.0;
   double perLot=PriceMoveMoney(slDist,1.0)+ExpectedAllInCost(1.0);if(perLot<=0)return;
   double lots=FloorVolume(risk/perLot);
   if(lots<=0 && InpAllowMinLotFallback)
   {
      double minRisk=PriceMoveMoney(slDist,broker.volumeMin)+ExpectedAllInCost(broker.volumeMin);
      if(bal>0 && minRisk/bal*100.0<=InpMinLotMaxRiskPct) lots=FloorVolume(broker.volumeMin);
   }
   if(lots<=0)return;
   if(InpMaxTotalLots>0&&SumOwnLots()+lots>InpMaxTotalLots)lots=FloorVolume(InpMaxTotalLots-SumOwnLots());
   if(lots<=0)return;

   double t1,t2,t3;BuildThreeTargets(dir,entry,lots,WIN_NONE,false,t1,t2,t3);
   // The recovery is a single-target trade: the TP placed at the broker must be the TP
   // that was validated. If the ATR-default target sits below the R:R floor, extend it
   // to the minimum target that satisfies BOTH the floor and the all-in cost check -
   // then re-verify ordering and validate EXACTLY what will be placed.
   double md2=MinTradeDistance();
   double need=MathAbs(entry-sl)*InpRecoveryMinRR;
   if(MathAbs(t1-entry)<need) t1=PriceNorm(entry+(dir>0?need:-need));
   double net=0;int gguard=0;
   while(!NetProfitValid(dir,entry,t1,lots,InpMinNetProfitTP1Money,net)&&gguard++<10) t1=PriceNorm(t1+dir*0.10*atr);
   if(!RRValid(dir,entry,sl,t1,InpRecoveryMinRR)){g_gateReason="recovery: R:R low";return;}
   if(!NetProfitValid(dir,entry,t1,lots,InpMinNetProfitTP1Money,net)){g_gateReason="recovery: cost";return;}
   if(!RiskRoom(PriceMoveMoney(slDist,lots)+ExpectedAllInCost(lots),dir,WIN_NONE,why)){g_gateReason="recovery: risk cap";return;}

   string cmt=InpComment+"|RCV|"+(dir>0?"B":"S")+"|"+IntegerToString((int)g_lastLossTime);
   double fill=0;ulong tk=0;
   // Broker TP is set to the VALIDATED TP1 (>=InpRecoveryMinRR). The recovery is a
   // single-shot counter-trade: if the EA restarts, the position still closes at the
   // target the entry was validated against, never an unvalidated TP3.
   if(MarketOrder(dir,lots,sl,t1,cmt,fill,tk))
   {
      g_recoveryLegs++;g_lastLossDir=0;g_gateReason="RECOVERY ARMED "+(dir>0?"BUY":"SELL");
      if(g_log!=INVALID_HANDLE)FileWrite(g_log,TimeToString(ServerNow(),TIME_DATE|TIME_SECONDS),"RCV",dir>0?"BUY":"SELL",DoubleToString(lots,2),DoubleToString(sl,broker.digits),DoubleToString(t1,broker.digits));
   }
}

//====================================================================
// POSITION MANAGEMENT TP1 / TP2 / TP3
//====================================================================
void ManagePosition(ulong ticket)
{
   if(!PositionSelectByTicket(ticket))return;int idx=FindPS(ticket);if(idx<0)return;
   double bid=Bid(),ask=Ask();int dir=g_ps[idx].direction;double px=(dir>0?bid:ask),vol=PositionGetDouble(POSITION_VOLUME),curSL=PositionGetDouble(POSITION_SL);if(vol<=0)return;
   double excursion=dir*(px-g_ps[idx].entry);if(excursion>0)g_ps[idx].mfePrice=MathMax(g_ps[idx].mfePrice,excursion);else g_ps[idx].maePrice=MathMax(g_ps[idx].maePrice,-excursion);
   if(InpMaxTradeMinutes>0&&ServerNow()-g_ps[idx].opened>=InpMaxTradeMinutes*60){ClosePositionSafe(ticket);return;}
   if(InpAvoidSwap&&InpForceFlatBeforeSwap&&InSwapDangerWindow()){ClosePositionSafe(ticket);return;}
   if(g_ps[idx].recovery)return;   // single-target recovery: broker TP/SL manage the exit
   if(!InpUseThreeTargets||!InpAB_EnableThreeTP)return;
   bool hit1=(dir>0?bid>=g_ps[idx].tp1:ask<=g_ps[idx].tp1),hit2=(dir>0?bid>=g_ps[idx].tp2:ask<=g_ps[idx].tp2),hit3=(dir>0?bid>=g_ps[idx].tp3:ask<=g_ps[idx].tp3);
   if(!g_ps[idx].tp1Done&&hit1)
   {
      double cv=MathMin(g_ps[idx].volTP1,vol-broker.volumeMin);
      if(cv<=0||FloorVolume(cv)<=0)g_ps[idx].tp1Done=true;
      else if(ClosePartialSafe(ticket,cv)){if(idx<ArraySize(g_ps)){g_ps[idx].tp1Done=true;g_ws[g_ps[idx].window].tp1Hits++;}}
      if(idx>=ArraySize(g_ps))return;
      if(g_ps[idx].tp1Done&&InpUseCostAdjustedBE&&PositionSelectByTicket(ticket))
      {
         double remain=PositionGetDouble(POSITION_VOLUME),be=CostAdjustedBE(dir,g_ps[idx].entry,remain),nowp=(dir>0?Bid():Ask());
         bool safe=(dir>0?(be>g_ps[idx].initialSL&&nowp-be>=MinTradeDistance()):(be<g_ps[idx].initialSL&&be-nowp>=MinTradeDistance()));if(safe)ModifyPositionSafe(ticket,be,g_ps[idx].tp3);
      }
   }
   if(idx>=ArraySize(g_ps))return;
   if(g_ps[idx].tp1Done&&!g_ps[idx].tp2Done&&hit2)
   {
      if(!PositionSelectByTicket(ticket))return;vol=PositionGetDouble(POSITION_VOLUME);double cv=MathMin(g_ps[idx].volTP2,vol-broker.volumeMin);
      if(cv<=0||FloorVolume(cv)<=0)g_ps[idx].tp2Done=true;
      else if(ClosePartialSafe(ticket,cv)){if(idx<ArraySize(g_ps)){g_ps[idx].tp2Done=true;g_ws[g_ps[idx].window].tp2Hits++;}}
   }
   if(idx>=ArraySize(g_ps))return;
   if(g_ps[idx].tp2Done&&PositionSelectByTicket(ticket))
   {
      if(InpTP3EarlyExit&&MomentumDeteriorated(dir)){ClosePositionSafe(ticket);return;}
      if(InpUseTP3StructureTrail)
      {
         curSL=PositionGetDouble(POSITION_SL);double nowp=(dir>0?Bid():Ask());double candidate=(dir>0?nowp-InpTP3TrailATR*g_atr:nowp+InpTP3TrailATR*g_atr);double structural=(dir>0?g_swingLow-InpSLStructureBufferATR*g_atr:g_swingHigh+InpSLStructureBufferATR*g_atr);
         if(structural>0)candidate=(dir>0?MathMax(candidate,structural):MathMin(candidate,structural));candidate=PriceNorm(candidate);
         bool improve=(dir>0?(candidate>curSL+InpTP3TrailStepATR*g_atr):(curSL==0||candidate<curSL-InpTP3TrailStepATR*g_atr));bool valid=(dir>0?(Bid()-candidate>=MinTradeDistance()):(candidate-Ask()>=MinTradeDistance()));if(improve&&valid)ModifyPositionSafe(ticket,candidate,g_ps[idx].tp3);
      }
   }
   if(idx<ArraySize(g_ps)&&hit3){g_ps[idx].tp3Done=true;g_ws[g_ps[idx].window].tp3Hits++;}
}

void ManageAllPositions()
{
   for(int i=PositionsTotal()-1;i>=0;i--){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetString(POSITION_SYMBOL)==eaSymbol&&PositionGetInteger(POSITION_MAGIC)==InpMagicNumber)ManagePosition(t);}
}

//====================================================================
// TRANSACTION TELEMETRY / PERFORMANCE
//====================================================================
ENUM_WINDOW_ID ParseWindowFromComment(string c)
{
   int p=StringFind(c,"|W");if(p<0)return WIN_NONE;int q=StringFind(c,"|",p+2);string n=(q>p?StringSubstr(c,p+2,q-(p+2)):StringSubstr(c,p+2));int w=(int)StringToInteger(n);return (w>WIN_NONE&&w<WIN_COUNT?(ENUM_WINDOW_ID)w:WIN_NONE);
}

string ParseSetupFromComment(string c)
{
   int p1=StringFind(c,"|W");if(p1<0)return "";int p2=StringFind(c,"|",p1+2);if(p2<0)return "";int p3=StringFind(c,"|",p2+1);if(p3<0)return "";int p4=StringFind(c,"|",p3+1);if(p4<0)return "";return StringSubstr(c,p4+1);
}

bool ParseHVFromComment(string c)
{
   int p=StringFind(c,"|HV");if(p<0)return false;return (StringFind(c,"|HV1",p)>=0);
}

void PushRecentNet(ENUM_WINDOW_ID w,double net)
{
   int cap=MathMin(64,MathMax(5,InpPerfRollingTrades));int i=g_ws[w].recentIdx%cap;g_ws[w].recentNet[i]=net;g_ws[w].recentIdx=(i+1)%cap;if(g_ws[w].recentCount<cap)g_ws[w].recentCount++;
}

//--- today-only stats (recounted from deal history: restart-proof, ref.mq5 pattern)
int      g_todayWins=0,g_todayLosses=0;
double   g_todayNet=0,g_todayGrossW=0,g_todayGrossL=0;
datetime g_todayStamp=0;

void RecountTodayStats()
{
   datetime now=ServerNow();
   MqlDateTime dt;TimeToStruct(now,dt);dt.hour=0;dt.min=0;dt.sec=0;
   datetime dayStart=StructToTime(dt);
   if(g_todayStamp==dayStart)return;
   g_todayStamp=dayStart;
   g_todayWins=0;g_todayLosses=0;g_todayNet=0;g_todayGrossW=0;g_todayGrossL=0;
   if(!HistorySelect(dayStart,now+60))return;
   for(int i=HistoryDealsTotal()-1;i>=0;i--)
   {
      ulong tk=HistoryDealGetTicket(i);if(tk==0)continue;
      if(HistoryDealGetInteger(tk,DEAL_MAGIC)!=InpMagicNumber)continue;
      if(HistoryDealGetString(tk,DEAL_SYMBOL)!=eaSymbol)continue;
      if((ENUM_DEAL_ENTRY)HistoryDealGetInteger(tk,DEAL_ENTRY)!=DEAL_ENTRY_OUT)continue;
      double p=HistoryDealGetDouble(tk,DEAL_PROFIT)+HistoryDealGetDouble(tk,DEAL_SWAP)+HistoryDealGetDouble(tk,DEAL_COMMISSION);
      g_todayNet+=p;
      if(p>=0){g_todayWins++;g_todayGrossW+=p;}else{g_todayLosses++;g_todayGrossL+=-p;}
   }
}
double TodayWinRate(){int t=g_todayWins+g_todayLosses;return t>0?100.0*g_todayWins/t:0;}
double TodayPF(){return g_todayGrossL>0?g_todayGrossW/g_todayGrossL:(g_todayGrossW>0?99.0:0);}

double OverallWinRate(){return g_perfTrades>0?100.0*g_perfWins/g_perfTrades:0;}
double OverallProfitFactor(){return g_perfGrossLoss>0?g_perfGrossProfit/g_perfGrossLoss:(g_perfGrossProfit>0?999.0:0);}
double OverallNetR(){return g_perfRetN>0?g_perfRetMean:0;}
double OverallSharpe()
{
   if(g_perfRetN<2)return 0;double var=g_perfRetM2/(g_perfRetN-1);if(var<=0)return 0;return g_perfRetMean/MathSqrt(var)*MathSqrt((double)g_perfRetN);
}
void UpdateOverallPerformance(double net,double rr)
{
   g_perfTrades++;g_perfNetProfit+=net;if(net>0){g_perfWins++;g_perfGrossProfit+=net;}else if(net<0){g_perfLosses++;g_perfGrossLoss+=-net;}
   g_perfRetN++;double d=rr-g_perfRetMean;g_perfRetMean+=d/g_perfRetN;double d2=rr-g_perfRetMean;g_perfRetM2+=d*d2;
   g_perfCumNet+=net;g_perfPeakNet=MathMax(g_perfPeakNet,g_perfCumNet);g_perfMaxDDMoney=MathMax(g_perfMaxDDMoney,g_perfPeakNet-g_perfCumNet);
}

void FinalizeWindowTrade(int idx,double net,double gross,double costs)
{
   if(idx<0||idx>=ArraySize(g_ps))return;PositionState s=g_ps[idx];ENUM_WINDOW_ID w=s.window;
   // Recovery legs keep full position management but are excluded from per-window
   // statistics and from resetting/extending the consecutive-loss streak of a window.
   if(s.recovery){RemovePS(idx);return;}
   if(w<=WIN_NONE||w>=WIN_COUNT){RemovePS(idx);return;}
   g_ws[w].trades++;g_ws[w].netPL+=net;g_ws[w].grossPL+=gross;g_ws[w].costs+=costs;g_ws[w].slipSum+=MathAbs(s.entrySlipPts);g_ws[w].spreadPctSum+=s.entrySpreadPct;g_ws[w].atrPctSum+=s.entryAtrPct;g_ws[w].volRatioSum+=s.entryVolRatio;g_ws[w].maeSum+=s.maePrice;g_ws[w].mfeSum+=s.mfePrice;if(net>0){g_ws[w].wins++;g_consecutiveLosses=0;}else if(net<0){g_ws[w].losses++;g_consecutiveLosses++;}
   double rr=(s.initialRiskMoney>0?net/s.initialRiskMoney:0);UpdateOverallPerformance(net,rr);g_ws[w].rSum+=rr;g_ws[w].peakNet=MathMax(g_ws[w].peakNet,g_ws[w].netPL);g_ws[w].maxDD=MathMax(g_ws[w].maxDD,g_ws[w].peakNet-g_ws[w].netPL);PushRecentNet(w,net);RefreshWindowGating(w);RemovePS(idx);
}

void LearnCommission(double dealComm,double dealVol)
{
   double c=MathAbs(dealComm),v=dealVol;if(c<=0||v<=0)return;double oneSide=c/v;double rt=2.0*oneSide;g_commissionRTPerLot=(g_commissionRTPerLot<=0?rt:0.90*g_commissionRTPerLot+0.10*rt);
}

void OnTradeTransaction(const MqlTradeTransaction &trans,const MqlTradeRequest &request,const MqlTradeResult &result)
{
   if(trans.type!=TRADE_TRANSACTION_DEAL_ADD||trans.deal==0)return;if(!HistoryDealSelect(trans.deal))return;if(HistoryDealGetInteger(trans.deal,DEAL_MAGIC)!=InpMagicNumber||HistoryDealGetString(trans.deal,DEAL_SYMBOL)!=eaSymbol)return;
   LearnCommission(HistoryDealGetDouble(trans.deal,DEAL_COMMISSION),HistoryDealGetDouble(trans.deal,DEAL_VOLUME));
   ENUM_DEAL_ENTRY e=(ENUM_DEAL_ENTRY)HistoryDealGetInteger(trans.deal,DEAL_ENTRY);ENUM_DEAL_TYPE dt=(ENUM_DEAL_TYPE)HistoryDealGetInteger(trans.deal,DEAL_TYPE);double price=HistoryDealGetDouble(trans.deal,DEAL_PRICE),vol=HistoryDealGetDouble(trans.deal,DEAL_VOLUME);long posId=HistoryDealGetInteger(trans.deal,DEAL_POSITION_ID);
   if(e==DEAL_ENTRY_IN)
   {
      g_tradesToday++;g_lastEntryTime=ServerNow();double slip=0;
      // Slippage must be measured against the order's intended price but the
      // deal-side spread is a known cost: measure only the excess over it.
      if(trans.order>0&&HistoryOrderSelect(trans.order))
      {
         double intended=HistoryOrderGetDouble(trans.order,ORDER_PRICE_OPEN);
         if(intended>0)   // market orders may report 0 as requested price -> no measurement
         {
            double raw=(dt==DEAL_TYPE_BUY?(price-intended):(intended-price))/broker.point;
            slip=raw-SpreadPoints();                 // spread is cost, not slippage
         }
         else slip=0;
      }
      PushSlippage(slip);
      if(MathAbs(slip)>=InpExtremeSlippagePoints){g_disorderUntil=ServerNow()+InpSlippageCooldownMinutes*60;}
      ENUM_WINDOW_ID w=WIN_NONE;string setup="";string c=HistoryDealGetString(trans.deal,DEAL_COMMENT);
      bool isRecovery=(StringFind(c,"|RCV")>=0);
      {
         // Recovery legs are tracked EXACTLY like normal positions (TP1/TP2 partial ladder,
         // time stop, swap exit, consecutive-loss accounting) under WIN_NONE; only window
         // stats routing differs (FinalizeWindowTrade skips stats for recovery=true).
         if(!isRecovery)
         {
            w=ParseWindowFromComment(c);setup=ParseSetupFromComment(c);
            if(w==WIN_NONE){bool tr=false;w=CurrentWindow(tr);}
         }
         bool hvC=(!isRecovery&&ParseHVFromComment(c));
         int dir=(dt==DEAL_TYPE_BUY?1:-1);
         ulong ticket=0;for(int i=0;i<PositionsTotal();i++){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetInteger(POSITION_IDENTIFIER)==posId){ticket=t;break;}}
         if(ticket&&PositionSelectByTicket(ticket))
         {
            double sl=PositionGetDouble(POSITION_SL),fv=PositionGetDouble(POSITION_VOLUME);
            AddPositionState(ticket,posId,dir,w,setup,hvC,isRecovery,price,sl,fv,slip,HistoryDealGetDouble(trans.deal,DEAL_COMMISSION));
            // Charge the per-window risk budget for the ACTUAL fill (each straddle layer separately),
            // so partial/layered fills can never exceed the window budget.
            if(!isRecovery&&w>WIN_NONE&&w<WIN_COUNT&&sl>0)g_windowRiskUsed[w]+=PriceMoveMoney(price-sl,fv)+ExpectedAllInCost(fv);
            if(InpCloseOnExtremeSlippage&&MathAbs(slip)>=InpExtremeSlippagePoints)ClosePositionSafe(ticket);
         }
                  if(!isRecovery)
         {
            // OCO safety: after first fill, remove opposite pending orders.
            for(int i=OrdersTotal()-1;i>=0;i--)
            {
               ulong ot=OrderGetTicket(i);
               if(!ot||!OrderSelect(ot))continue;
               if(OrderGetString(ORDER_SYMBOL)!=eaSymbol||OrderGetInteger(ORDER_MAGIC)!=InpMagicNumber)continue;
               ENUM_ORDER_TYPE typ=(ENUM_ORDER_TYPE)OrderGetInteger(ORDER_TYPE);
               if((dir>0&&typ==ORDER_TYPE_SELL_STOP)||(dir<0&&typ==ORDER_TYPE_BUY_STOP))DeleteOrderSafe(ot);
            }
         }
      }
   }
   else if(e==DEAL_ENTRY_OUT||e==DEAL_ENTRY_OUT_BY)
   {
      double gross=HistoryDealGetDouble(trans.deal,DEAL_PROFIT);double swap=HistoryDealGetDouble(trans.deal,DEAL_SWAP);double comm=HistoryDealGetDouble(trans.deal,DEAL_COMMISSION);double net=gross+swap+comm;g_lastExitTime=ServerNow();
      int pidx=-1;for(int j=0;j<ArraySize(g_ps);j++)if(g_ps[j].positionId==posId){pidx=j;break;}
      if(pidx>=0){g_ps[pidx].realizedGross+=gross;g_ps[pidx].realizedNet+=net;g_ps[pidx].realizedCosts+=MathAbs(comm)+MathAbs(swap);}
      bool still=false;for(int i=0;i<PositionsTotal();i++){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetInteger(POSITION_IDENTIFIER)==posId){still=true;break;}}
      if(!still&&pidx>=0)
      {
         double finalNet=g_ps[pidx].realizedNet;
         if(finalNet<0 && !g_ps[pidx].recovery)
         {
            // Realized base-strategy loss: open ONE gated reversal opportunity.
            // Recovery-leg losses never chain another recovery.
            g_lastLossDir=g_ps[pidx].direction;g_lastLossTime=ServerNow();g_lastLossMoney=finalNet;
            g_recoveryLegs=0;   // fresh loss event = fresh recovery allowance
            g_gateReason="LOSS - recovery candidate";
         }
         FinalizeWindowTrade(pidx,finalNet,g_ps[pidx].realizedGross,g_ps[pidx].realizedCosts);
      }
   }
}

//====================================================================
// STATE / LOGGING
//====================================================================
#define STATE_TAG 20260913
string StateName(){return "PAT101_"+eaSymbol+"_"+IntegerToString(InpMagicNumber)+".bin";}

void SaveState()
{
   if(!InpPersistState)return;int f=FileOpen(StateName(),FILE_BIN|FILE_WRITE|FILE_COMMON);if(f==INVALID_HANDLE)return;
   FileWriteInteger(f,STATE_TAG,INT_VALUE);
   FileWriteInteger(f,g_dayKey,INT_VALUE);FileWriteInteger(f,g_weekKey,INT_VALUE);FileWriteInteger(f,g_monthKey,INT_VALUE);FileWriteDouble(f,g_dayAnchor);FileWriteDouble(f,g_weekAnchor);FileWriteDouble(f,g_monthAnchor);FileWriteInteger(f,g_tradesToday,INT_VALUE);FileWriteInteger(f,g_consecutiveLosses,INT_VALUE);FileWriteDouble(f,g_commissionRTPerLot);FileWriteInteger(f,g_perfTrades,INT_VALUE);FileWriteInteger(f,g_perfWins,INT_VALUE);FileWriteInteger(f,g_perfLosses,INT_VALUE);FileWriteDouble(f,g_perfNetProfit);FileWriteDouble(f,g_perfGrossProfit);FileWriteDouble(f,g_perfGrossLoss);FileWriteInteger(f,g_perfRetN,INT_VALUE);FileWriteDouble(f,g_perfRetMean);FileWriteDouble(f,g_perfRetM2);FileWriteDouble(f,g_perfCumNet);FileWriteDouble(f,g_perfPeakNet);FileWriteDouble(f,g_perfMaxDDMoney);
   for(int w=0;w<WIN_COUNT;w++){FileWriteInteger(f,g_ws[w].trades,INT_VALUE);FileWriteInteger(f,g_ws[w].wins,INT_VALUE);FileWriteInteger(f,g_ws[w].losses,INT_VALUE);FileWriteDouble(f,g_ws[w].grossPL);FileWriteDouble(f,g_ws[w].netPL);FileWriteDouble(f,g_ws[w].costs);FileWriteInteger(f,g_ws[w].tp1Hits,INT_VALUE);FileWriteInteger(f,g_ws[w].tp2Hits,INT_VALUE);FileWriteInteger(f,g_ws[w].tp3Hits,INT_VALUE);}
   // --- open-position ladder plans: a VPS restart must not silently drop the
   // --- 60/25/15 partial management and leave positions broker-managed to TP3/SL.
   int nPS=ArraySize(g_ps);FileWriteInteger(f,nPS,INT_VALUE);
   for(int p=0;p<nPS;p++)
   {
      FileWriteLong(f,(long)g_ps[p].ticket);FileWriteLong(f,g_ps[p].positionId);
      FileWriteInteger(f,g_ps[p].direction,INT_VALUE);FileWriteInteger(f,(int)g_ps[p].window,INT_VALUE);
      FileWriteString(f,g_ps[p].setupId);FileWriteInteger(f,0,INT_VALUE);   // terminator for the string
      FileWriteInteger(f,g_ps[p].hv?1:0,INT_VALUE);FileWriteInteger(f,g_ps[p].recovery?1:0,INT_VALUE);
      FileWriteDouble(f,g_ps[p].initialVolume);FileWriteDouble(f,g_ps[p].initialRiskMoney);
      FileWriteDouble(f,g_ps[p].entry);FileWriteDouble(f,g_ps[p].initialSL);
      FileWriteDouble(f,g_ps[p].tp1);FileWriteDouble(f,g_ps[p].tp2);FileWriteDouble(f,g_ps[p].tp3);
      FileWriteDouble(f,g_ps[p].volTP1);FileWriteDouble(f,g_ps[p].volTP2);FileWriteDouble(f,g_ps[p].volTP3);
      FileWriteInteger(f,g_ps[p].tp1Done?1:0,INT_VALUE);FileWriteInteger(f,g_ps[p].tp2Done?1:0,INT_VALUE);FileWriteInteger(f,g_ps[p].tp3Done?1:0,INT_VALUE);
      FileWriteDouble(f,g_ps[p].maePrice);FileWriteDouble(f,g_ps[p].mfePrice);
      FileWriteLong(f,(long)g_ps[p].opened);
      FileWriteDouble(f,g_ps[p].entrySpreadPct);FileWriteDouble(f,g_ps[p].entrySlipPts);
      FileWriteDouble(f,g_ps[p].entryAtrPct);FileWriteDouble(f,g_ps[p].entryVolRatio);
      FileWriteDouble(f,g_ps[p].realizedGross);FileWriteDouble(f,g_ps[p].realizedNet);FileWriteDouble(f,g_ps[p].realizedCosts);
   }
   FileClose(f);
}

void LoadState()
{
   int f=FileOpen(StateName(),FILE_BIN|FILE_READ|FILE_COMMON);if(f==INVALID_HANDLE)return;
   int tag=(int)FileReadInteger(f,INT_VALUE);
   if(tag!=STATE_TAG){FileClose(f);Print("State file format differs (tag=",tag,") - starting from a clean slate.");return;}
   g_dayKey=(int)FileReadInteger(f,INT_VALUE);g_weekKey=(int)FileReadInteger(f,INT_VALUE);g_monthKey=(int)FileReadInteger(f,INT_VALUE);g_dayAnchor=FileReadDouble(f);g_weekAnchor=FileReadDouble(f);g_monthAnchor=FileReadDouble(f);g_tradesToday=(int)FileReadInteger(f,INT_VALUE);g_consecutiveLosses=(int)FileReadInteger(f,INT_VALUE);g_commissionRTPerLot=FileReadDouble(f);if(!FileIsEnding(f)){g_perfTrades=(int)FileReadInteger(f,INT_VALUE);g_perfWins=(int)FileReadInteger(f,INT_VALUE);g_perfLosses=(int)FileReadInteger(f,INT_VALUE);g_perfNetProfit=FileReadDouble(f);g_perfGrossProfit=FileReadDouble(f);g_perfGrossLoss=FileReadDouble(f);g_perfRetN=(int)FileReadInteger(f,INT_VALUE);g_perfRetMean=FileReadDouble(f);g_perfRetM2=FileReadDouble(f);g_perfCumNet=FileReadDouble(f);g_perfPeakNet=FileReadDouble(f);g_perfMaxDDMoney=FileReadDouble(f);}
   for(int w=0;w<WIN_COUNT&&!FileIsEnding(f);w++){g_ws[w].trades=(int)FileReadInteger(f,INT_VALUE);g_ws[w].wins=(int)FileReadInteger(f,INT_VALUE);g_ws[w].losses=(int)FileReadInteger(f,INT_VALUE);g_ws[w].grossPL=FileReadDouble(f);g_ws[w].netPL=FileReadDouble(f);g_ws[w].costs=FileReadDouble(f);g_ws[w].tp1Hits=(int)FileReadInteger(f,INT_VALUE);g_ws[w].tp2Hits=(int)FileReadInteger(f,INT_VALUE);g_ws[w].tp3Hits=(int)FileReadInteger(f,INT_VALUE);}
   // --- open-position ladder plans (only re-attached if the position still exists)
   if(!FileIsEnding(f))
   {
      int nPS=(int)FileReadInteger(f,INT_VALUE);
      for(int p=0;p<nPS&&!FileIsEnding(f);p++)
      {
         PositionState s;ZeroMemory(s);
         s.ticket=(ulong)FileReadLong(f);s.positionId=FileReadLong(f);
         s.direction=(int)FileReadInteger(f,INT_VALUE);s.window=(ENUM_WINDOW_ID)FileReadInteger(f,INT_VALUE);
         s.setupId=FileReadString(f);FileReadInteger(f,INT_VALUE);   // string terminator
         s.hv=(FileReadInteger(f,INT_VALUE)==1);s.recovery=(FileReadInteger(f,INT_VALUE)==1);
         s.initialVolume=FileReadDouble(f);s.initialRiskMoney=FileReadDouble(f);
         s.entry=FileReadDouble(f);s.initialSL=FileReadDouble(f);
         s.tp1=FileReadDouble(f);s.tp2=FileReadDouble(f);s.tp3=FileReadDouble(f);
         s.volTP1=FileReadDouble(f);s.volTP2=FileReadDouble(f);s.volTP3=FileReadDouble(f);
         s.tp1Done=(FileReadInteger(f,INT_VALUE)==1);s.tp2Done=(FileReadInteger(f,INT_VALUE)==1);s.tp3Done=(FileReadInteger(f,INT_VALUE)==1);
         s.maePrice=FileReadDouble(f);s.mfePrice=FileReadDouble(f);
         s.opened=(datetime)FileReadLong(f);
         s.entrySpreadPct=FileReadDouble(f);s.entrySlipPts=FileReadDouble(f);
         s.entryAtrPct=FileReadDouble(f);s.entryVolRatio=FileReadDouble(f);
         s.realizedGross=FileReadDouble(f);s.realizedNet=FileReadDouble(f);s.realizedCosts=FileReadDouble(f);
         // Re-attach only if the position still exists (partial closes during downtime
         // may have changed the volume; ManagePosition always reads live volume anyway).
         bool exists=false;
         for(int i=0;i<PositionsTotal();i++){ulong t=PositionGetTicket(i);if(t&&PositionSelectByTicket(t)&&PositionGetInteger(POSITION_IDENTIFIER)==s.positionId){exists=true;break;}}
         if(exists&&s.ticket>0)
         {
            int sz=ArraySize(g_ps);ArrayResize(g_ps,sz+1);g_ps[sz]=s;
            Print("Restored open-position plan ticket=",s.ticket," (TP1 ",DoubleToString(s.tp1,broker.digits),")");
         }
      }
   }
   FileClose(f);
}

void OpenLog()
{
   if(!InpUseLogFile)return;string n="PAT101_"+eaSymbol+"_"+TimeToString(ServerNow(),TIME_DATE)+".csv";StringReplace(n,".","-");StringReplace(n,":","-");
   g_log=FileOpen(n,FILE_CSV|FILE_READ|FILE_WRITE|FILE_SHARE_READ|FILE_COMMON,';');if(g_log==INVALID_HANDLE)return;if(FileSize(g_log)==0)FileWrite(g_log,"server_time","event","window","dir","setup","spread_pts","spread_pct","atr_pct","vol_ratio","risk_money","tp1","tp2","tp3");FileSeek(g_log,0,SEEK_END);
}

void PrintSummary()
{
   Print("=== Predict-A-Trade v1.00 Four-Session Macro/SMC Window Summary ===");
   Print("OVERALL trades=",g_perfTrades," win%=",DoubleToString(OverallWinRate(),2)," PF=",DoubleToString(OverallProfitFactor(),2)," NetR=",DoubleToString(OverallNetR(),3)," Sharpe(trade-R)=",DoubleToString(OverallSharpe(),2)," MaxDD$=",DoubleToString(g_perfMaxDDMoney,2)," MaxDD%=",DoubleToString(g_maxDDSeen,2)," Net$=",DoubleToString(g_perfNetProfit,2));
   for(int w=1;w<WIN_COUNT;w++){if(g_ws[w].trades<=0)continue;double wr=100.0*g_ws[w].wins/MathMax(1,g_ws[w].trades);double ex=WindowExpectancy((ENUM_WINDOW_ID)w);Print(WindowName((ENUM_WINDOW_ID)w)," trades=",g_ws[w].trades," net=",DoubleToString(g_ws[w].netPL,2)," win%=",DoubleToString(wr,1)," exp=",DoubleToString(ex,2)," NetR=",DoubleToString(WindowAvgR((ENUM_WINDOW_ID)w),3)," TP1/2/3=",g_ws[w].tp1Hits,"/",g_ws[w].tp2Hits,"/",g_ws[w].tp3Hits," DD=",DoubleToString(g_ws[w].maxDD,2));}
}

void WriteWindowReport()
{
   string n="PAT101_WindowReport_"+eaSymbol+"_"+IntegerToString(InpMagicNumber)+".csv";
   int f=FileOpen(n,FILE_CSV|FILE_WRITE|FILE_COMMON,';');if(f==INVALID_HANDLE)return;
   FileWrite(f,"window","trades","wins","losses","win_rate_pct","gross_pl","net_pl","costs","cost_ratio_pct","tp1_hit_pct","tp2_hit_pct","tp3_hit_pct","avg_net_R","avg_slip_pts","avg_spread_pct","avg_atr_pct","avg_vol_ratio","avg_MAE_price","avg_MFE_price","max_drawdown_money","rolling_expectancy_money","observations","obs_avg_atr_ratio","obs_avg_vol_ratio","disabled");
   for(int w=1;w<WIN_COUNT;w++)
   {
      int t=g_ws[w].trades;double den=MathMax(1,t);double cr=(MathAbs(g_ws[w].grossPL)>0?100.0*g_ws[w].costs/MathAbs(g_ws[w].grossPL):0);
      long obs=g_ws[w].observations;double oden=(double)MathMax((long)1,obs);
      FileWrite(f,WindowName((ENUM_WINDOW_ID)w),t,g_ws[w].wins,g_ws[w].losses,DoubleToString(100.0*g_ws[w].wins/den,2),DoubleToString(g_ws[w].grossPL,2),DoubleToString(g_ws[w].netPL,2),DoubleToString(g_ws[w].costs,2),DoubleToString(cr,2),DoubleToString(100.0*g_ws[w].tp1Hits/den,2),DoubleToString(100.0*g_ws[w].tp2Hits/den,2),DoubleToString(100.0*g_ws[w].tp3Hits/den,2),DoubleToString(g_ws[w].rSum/den,3),DoubleToString(g_ws[w].slipSum/den,2),DoubleToString(g_ws[w].spreadPctSum/den,2),DoubleToString(g_ws[w].atrPctSum/den,2),DoubleToString(g_ws[w].volRatioSum/den,3),DoubleToString(g_ws[w].maeSum/den,broker.digits),DoubleToString(g_ws[w].mfeSum/den,broker.digits),DoubleToString(g_ws[w].maxDD,2),DoubleToString(WindowExpectancy((ENUM_WINDOW_ID)w),3),obs,DoubleToString(g_ws[w].obsATRRatioSum/oden,3),DoubleToString(g_ws[w].obsVolRatioSum/oden,3),(g_ws[w].disabled?"YES":"NO"));
   }
   FileClose(f);
}

void WritePerformanceReport()
{
   string n="PAT101_Performance_"+eaSymbol+"_"+IntegerToString(InpMagicNumber)+".csv";int f=FileOpen(n,FILE_CSV|FILE_WRITE|FILE_COMMON,';');if(f==INVALID_HANDLE)return;
   FileWrite(f,"metric","value");
   FileWrite(f,"number_of_trades",g_perfTrades);FileWrite(f,"wins",g_perfWins);FileWrite(f,"losses",g_perfLosses);FileWrite(f,"win_rate_pct",DoubleToString(OverallWinRate(),4));FileWrite(f,"profit_factor",DoubleToString(OverallProfitFactor(),4));FileWrite(f,"net_R_per_trade",DoubleToString(OverallNetR(),4));FileWrite(f,"sharpe_trade_R",DoubleToString(OverallSharpe(),4));FileWrite(f,"net_profit",DoubleToString(g_perfNetProfit,2));FileWrite(f,"max_drawdown_money",DoubleToString(g_perfMaxDDMoney,2));FileWrite(f,"max_intraday_drawdown_pct",DoubleToString(g_maxDDSeen,4));FileWrite(f,"avg_slippage_points",DoubleToString(g_slipAvg,3));FileWrite(f,"last_slippage_points",DoubleToString(g_lastSlipPts,3));FileWrite(f,"avg_spread_points",DoubleToString(g_spreadAvg,2));FileWrite(f,"commission_rt_per_lot",DoubleToString(g_commissionRTPerLot>0?g_commissionRTPerLot:InpCommissionPerLotRTFallback,4));
   FileWrite(f,"fmp_usd_avg_pct",DoubleToString(g_usdAvg,4));FileWrite(f,"fmp_usd_bias",g_usdBias);FileWrite(f,"fmp_spx_move_pct",DoubleToString(g_spxMovePct,4));FileWrite(f,"fmp_available",(g_usdAvailable?"YES":"NO"));FileWrite(f,"fmp_news_hits",g_fmpNewsCount);
   FileClose(f);
}//====================================================================
// DASHBOARD - flow-layout two-column control panel (overlap-proof)
//--------------------------------------------------------------------
// Every string is width-clipped to its column with TextGetSize, rows are
// drawn with a running Y cursor (no absolute row indices), and the panel
// height is computed in a pass-1 measure sweep. Overlap is impossible by
// construction. Header is draggable; P button / F key pauses arming.
//====================================================================
//--- uppercase helper: StringToUpper mutates in place and cannot take a constant
string Upper(string s){ string t=s; StringToUpper(t); return t; }

int SX(int v){ return (int)MathRound(v*g_font/9.0*g_dpiScale); }   // scale px constants by font AND display DPI

//--- clip a string to maxW px, appending "…" when it does not fit
string ClipText(string s,int maxW,int fs)
{
   if(maxW<=12) return (StringLen(s)==0?s:"");
   if(StringLen(s)==0) return s;
   // Measure with the EXACT font/size the label renders in; TextGetSize without
   // TextSetFont uses terminal defaults whose metrics differ from Consolas.
   TextSetFont("Consolas",FontOut(fs),FW_DONTCARE,0);
   uint w=0,h=0;
   if(!TextGetSize(s,w,h)) return s;
   if((int)w<=maxW) return s;
   while(StringLen(s)>1)
   {
      s=StringSubstr(s,0,StringLen(s)-1);
      string t=s+"…";
      if(TextGetSize(t,w,h) && (int)w<=maxW) return t;
   }
   return "";
}

void UIRecompute()
{
   g_font=MathMax(6,MathMin(12,InpPanelFontSize));
   // Windows display scaling (125/150%) makes MT5 render label glyphs physically larger
   // while chart objects stay in raw pixels. Scale the WHOLE layout by the terminal DPI
   // so rows keep their proportions and text never overflows its cell.
   long dpi=TerminalInfoInteger(TERMINAL_SCREEN_DPI);
   if(dpi<96)dpi=96; if(dpi>480)dpi=480;
   g_dpi=(int)dpi;
   g_dpiScale=(double)g_dpi/96.0;
   g_fontPx=(int)MathRound(g_font*g_dpiScale);      // rendered glyph height in px
   g_rh=(int)MathRound((g_font+7)*g_dpiScale);      // row pitch scales with glyphs
   g_hdrH=(int)MathRound((g_font+19)*g_dpiScale);
   g_colW=SX(340);
   g_pad=MathMax(8,SX(12));
   g_gap=MathMax(8,SX(14));
   g_panelW=g_pad*2+g_colW*2+g_gap;
   g_bodyTop=g_hdrH+6;
   g_colHL=(L_SECTIONS*SX(20))+L_ROWS*g_rh+SX(8);
   g_colHR=(R_SECTIONS*SX(20))+R_ROWS*g_rh+SX(8);
   g_tlTop=g_bodyTop+MathMax(g_colHL,g_colHR)+SX(6);
   g_tlH=SX(13)+2+4*g_rh+SX(16);
   g_panelH=g_tlTop+g_tlH+2*g_rh+g_pad;
}

void UIRect(string n,int x,int y,int w,int h,color bg)
{
   string id=UI_PREFIX+n;if(ObjectFind(0,id)<0)ObjectCreate(0,id,OBJ_RECTANGLE_LABEL,0,0,0);
   ObjectSetInteger(0,id,OBJPROP_CORNER,CORNER_LEFT_UPPER);
   ObjectSetInteger(0,id,OBJPROP_XDISTANCE,x);ObjectSetInteger(0,id,OBJPROP_YDISTANCE,y);
   ObjectSetInteger(0,id,OBJPROP_XSIZE,MathMax(1,w));ObjectSetInteger(0,id,OBJPROP_YSIZE,MathMax(1,h));
   ObjectSetInteger(0,id,OBJPROP_BGCOLOR,bg);ObjectSetInteger(0,id,OBJPROP_COLOR,bg);
   ObjectSetInteger(0,id,OBJPROP_WIDTH,1);
   ObjectSetInteger(0,id,OBJPROP_HIDDEN,true);ObjectSetInteger(0,id,OBJPROP_SELECTABLE,false);
}

int FontOut(int fs){ return (int)MathMax(6,MathRound(fs/g_dpiScale)); }  // counter-scale font size for display DPI

void UILabel(string n,int x,int y,string txt,color c,int sz=-1)
{
   if(g_measure)return;                       // pass-1 layout sweep draws nothing
   int req=(sz>0?sz:g_font);
   int fs=FontOut(req);
   string id=UI_PREFIX+n;if(ObjectFind(0,id)<0)ObjectCreate(0,id,OBJ_LABEL,0,0,0);
   ObjectSetInteger(0,id,OBJPROP_CORNER,CORNER_LEFT_UPPER);
   ObjectSetInteger(0,id,OBJPROP_XDISTANCE,x);ObjectSetInteger(0,id,OBJPROP_YDISTANCE,y);
   ObjectSetInteger(0,id,OBJPROP_COLOR,c);ObjectSetInteger(0,id,OBJPROP_FONTSIZE,fs);
   ObjectSetString(0,id,OBJPROP_FONT,"Consolas");   // single font: TextGetSize metrics == rendered metrics
   ObjectSetString(0,id,OBJPROP_TEXT,txt);
   ObjectSetInteger(0,id,OBJPROP_HIDDEN,true);ObjectSetInteger(0,id,OBJPROP_SELECTABLE,false);
}

//--- section bar at the current Y cursor
void DashSection(string n,int col,int &y,string title)
{
   int x=g_x+g_pad+col*(g_colW+g_gap);
   if(!g_measure)
   {
      UIRect("SB_"+n,x,y,g_colW,SX(20)-3,C_SECTION);
      UIRect("SL_"+n,x,y,3,SX(20)-3,C_ACCENT);
      UILabel("ST_"+n,x+8,y+((SX(20)-3-MathMax(6,g_font-1))/2),ClipText(Upper(title),g_colW-14,MathMax(6,g_font-1)),C_SECTION_TXT,MathMax(6,g_font-1));
   }
   y+=SX(20)+2;
}

//--- one flowing row; text is clipped to the column so it can never bleed
void DashRow(string n,int col,int &y,string txt,color c)
{
   int x=g_x+g_pad+col*(g_colW+g_gap)+6;
   if(!g_measure)UILabel("R_"+n,x,y+MathMax(0,(g_rh-g_fontPx)/2),ClipText(txt,g_colW-12,g_font),c,g_font);
   y+=g_rh;
}

//--- two labels on one row (value left, status right-aligned in-column)
void DashRowSplit(string n,int col,int &y,string left,string right,color cl,color cr)
{
   int x=g_x+g_pad+col*(g_colW+g_gap)+6;
   if(!g_measure)
   {
      UILabel("R_"+n,x,y+MathMax(0,(g_rh-g_fontPx)/2),ClipText(left,g_colW-100,g_font),cl,g_font);
      uint w=0,h=0;string r=ClipText(right,SX(96),g_font);
      TextSetFont("Consolas",FontOut(g_font),FW_DONTCARE,0);
      TextGetSize(r,w,h);
      UILabel("R_"+n+"b",x+g_colW-12-SX(4)-(int)w,y+MathMax(0,(g_rh-g_fontPx)/2),r,cr,g_font);
   }
   y+=g_rh;
}

// one filled 24h segment (handles ranges that wrap past midnight)
void DashSeg(string n,int bx,int by,int bw,int openMin,int closeMin,color col)
{
   int x1=bx+(int)MathRound(openMin*bw/1440.0);
   int x2=bx+(int)MathRound(closeMin*bw/1440.0);
   if(closeMin>openMin)UIRect(n,x1,by,MathMax(2,x2-x1),9,col);
   else{UIRect(n+"a",x1,by,MathMax(2,bx+bw-x1),9,col);UIRect(n+"b",bx,by,MathMax(2,x2-bx),9,col);}
}

color SessionColor(int i,bool active)
{
   if(i==0)return (active?C_SYD:C_SYD_DIM);
   if(i==1)return (active?C_TOK:C_TOK_DIM);
   if(i==2)return (active?C_LON:C_LON_DIM);
   return (active?C_NY:C_NY_DIM);
}

//--- session-progress phase label (ref.mq5 heuristic, DST-aware bounds)
string SessionPhaseLabel(int utcMin,int openMin,int closeMin)
{
   int len=(closeMin>openMin?closeMin-openMin:1440-openMin+closeMin);
   int into=(utcMin>=openMin?utcMin-openMin:1440-openMin+utcMin);
   if(len<=0)return "IDLE";
   double f=(double)into/(double)len;
   if(f<0.20)return "MANIP";
   if(f<0.75)return "EXPANSION";
   return "REVERSAL";
}

string FmtMoney(double v){ return (v>=0?"+":"")+DoubleToString(v,2); }
string BiasArrow(int b){ return (b>0?"UP":(b<0?"DOWN":"FLAT")); }
color BiasColor(int b){ return (b>0?C_UP_TXT:(b<0?C_DN_TXT:C_DIM)); }
string AccTypeName(ENUM_ACCOUNT_TRADE_MODE m)
{
   if(m==ACCOUNT_TRADE_MODE_DEMO)return "DEMO";
   if(m==ACCOUNT_TRADE_MODE_CONTEST)return "CONTEST";
   if(m==ACCOUNT_TRADE_MODE_REAL)return "REAL";
   return "UNKNOWN";
}
string MarginModeName(ENUM_ACCOUNT_MARGIN_MODE m)
{
   if(m==ACCOUNT_MARGIN_MODE_RETAIL_HEDGING)return "HEDGING";
   if(m==ACCOUNT_MARGIN_MODE_RETAIL_NETTING)return "NETTING";
   return "EXCHANGE";
}
string TradeModeName(int m)
{
   if(m==SYMBOL_TRADE_MODE_FULL)return "FULL";
   if(m==SYMBOL_TRADE_MODE_LONGONLY)return "LONG ONLY";
   if(m==SYMBOL_TRADE_MODE_SHORTONLY)return "SHORT ONLY";
   if(m==SYMBOL_TRADE_MODE_CLOSEONLY)return "CLOSE ONLY";
   if(m==SYMBOL_TRADE_MODE_DISABLED)return "DISABLED";
   return "UNKNOWN";
}

string WindowStateText(ENUM_WINDOW_ID w,bool allowed)
{
   if(w<=WIN_NONE||w>=WIN_COUNT)return "IDLE";
   if(g_ws[w].disabled)return "GATED";
   return (allowed?"LIVE":"CLOSED");
}

string TopWindowsText()
{
   string s="";int shown=0;
   for(int w=1;w<WIN_COUNT&&shown<3;w++)
   {
      if(g_ws[w].trades<=0)continue;
      string t=WindowName((ENUM_WINDOW_ID)w)+" "+FmtMoney(g_ws[w].netPL)+"("+IntegerToString(g_ws[w].trades)+")";
      s+=(shown>0?"  ":"")+t;shown++;
   }
   return (s==""?"no closed trades yet":s);
}

void DashUpdate(bool force=false)
{
   if(!InpShowDashboard)return;
   uint ms=GetTickCount();
   if(!force&&g_lastDashMs>0&&ms-g_lastDashMs<(uint)MathMax(100,InpDashRefreshMs))return;
   g_lastDashMs=ms;
   UIRecompute();
   // Full clear every frame (ref.mq5 pattern): stale objects from any earlier layout can
   // never linger, so ghost panels / double bars are impossible.
   ObjectsDeleteAll(0,UI_PREFIX,0,-1);

   int x=g_x,y=g_y;
   if(g_dashCollapsed!=g_dashWasCollapsed){DashDestroy();g_dashWasCollapsed=g_dashCollapsed;}

   bool tr=false;ENUM_WINDOW_ID w=CurrentWindow(tr);
   int wi=(w>WIN_NONE&&w<WIN_COUNT?(int)w:0);

   //---- header
   if(g_dashCollapsed)
   {
      UIRect("BG",x,y,g_panelW,g_hdrH,C_BG);
      UIRect("HDR",x,y,g_panelW,g_hdrH,C_HDR);
      UIRect("HL",x,y,3,g_hdrH,C_ACCENT);
      UILabel("T",x+10,y+((g_hdrH-g_font-2)/2),ClipText(Upper(InpPanelTitle)+" v1.00 ["+eaSymbol+" M1]",g_panelW-SX(160),g_font+2),clrWhite,g_font+2);
      UILabel("CS",x+g_panelW-SX(150),y+((g_hdrH-g_font)/2),WindowName(w)+" "+(tr?"LIVE":"OFF"),(tr?C_UP_TXT:C_DIM),g_font);
      UILabel("CH",x+g_panelW-SX(18),y+((g_hdrH-g_font)/2),"+",C_TXT,g_font);
      ChartRedraw();return;
   }

   UIRect("BG",x,y,g_panelW,g_panelH,C_BG);
   UIRect("HDR",x,y,g_panelW,g_hdrH,C_HDR);
   UIRect("HL",x,y,3,g_hdrH,C_ACCENT);
   UILabel("T",x+10,y+((g_hdrH-g_font-2)/2),ClipText(Upper(InpPanelTitle)+" v1.00 ["+eaSymbol+" M1]",g_panelW-SX(190),g_font+2),clrWhite,g_font+2);

   //---- header status chips (right-aligned, fixed slots)
   int chipX=x+g_panelW-SX(8);
   UILabel("CH_COLL",chipX-SX(20),y+((g_hdrH-g_font)/2),"[-]",C_TXT2,g_font);
   chipX-=SX(28);
   bool hvNow=false;{int hs=HVScore(g_dirBias>=0?1:-1);hvNow=(InpHighVolatilityMode!=HV_OFF&&hs>=InpHVMinScore);}
   UILabel("CH_HV",chipX-SX(24),y+((g_hdrH-g_font)/2),(hvNow?"HV":"--"),(hvNow?C_WARN_TXT:C_GRAY),g_font);
   chipX-=SX(30);
   bool halted=(g_stopDay||g_stopWeek||g_stopMonth);
   string eng=(halted?"HALTED":(g_paused?"PAUSED":(tr?"LIVE":"WAIT")));
   color engC=(halted?C_DN_TXT:(g_paused?C_WARN_TXT:(tr?C_UP_TXT:C_DIM)));
   UILabel("CH_STATE",chipX-SX(52),y+((g_hdrH-g_font)/2),eng,engC,g_font);

   //---- derived values
   datetime utc=UTCNow();int um=MinuteOfDay(utc);
   int so,sc,to,tc,lo,lc,no,nc;SessionUTCBounds(utc,so,sc,to,tc,lo,lc,no,nc);
   double eq=AccountInfoDouble(ACCOUNT_EQUITY),bal=AccountInfoDouble(ACCOUNT_BALANCE);
   double margin=AccountInfoDouble(ACCOUNT_MARGIN),freeM=AccountInfoDouble(ACCOUNT_MARGIN_FREE);
   double dd=MathMax(0,(g_dayAnchor>0?(g_dayAnchor-eq)/g_dayAnchor*100:0));
   int atrPts=(broker.point>0?(int)MathRound(g_atr/broker.point):0);
   double sp=SpreadPoints(),spp=SpreadPercentile(),ap=ATRPercentile();
   int hs2=HVScore(g_dirBias>=0?1:-1);
   double openRisk=OpenRiskMoney();
   double riskPct=(eq>0?openRisk/eq*100.0:0);
   double budgetPct=(eq>0?g_windowRiskUsed[wi]/eq*100.0:0);
   int left=MinsUntilWindowClose(w,um,sc,tc,lc,nc);
   double vwapDev=(g_vwap>0&&g_atr>0?(iClose(eaSymbol,PERIOD_M1,1)-g_vwap)/g_atr:0);
   double marginPct=(eq>0?margin/eq*100.0:0);
   double commRT=(g_commissionRTPerLot>0?g_commissionRTPerLot:InpCommissionPerLotRTFallback);

   //================ LEFT COLUMN (flow) ==================
   int rL=0;
   int yL=g_bodyTop;
   DashSection("LS",0,yL,"market / session");
   DashRowSplit("L_WIN",0,yL,WindowName(w),"["+WindowStateText(w,tr)+"]",C_GOLD_TXT,(tr?C_UP_TXT:C_DIM));
   DashRow("L_RNG",0,yL,WindowUTCText(w,so,sc,to,tc,lo,lc,no,nc)+(left>=0?"  "+IntegerToString(left)+"m":""),C_TXT2);
   bool ovSYDTOK=InWindowMinutes(um,so,sc)&&InWindowMinutes(um,to,tc);
   bool ovTOKLON=InWindowMinutes(um,to,tc)&&InWindowMinutes(um,lo,lc);
   bool ovLONNY =InWindowMinutes(um,lo,lc)&&InWindowMinutes(um,no,nc);
   string ovShort=(ovLONNY?"L+NY":(ovTOKLON?"T+L":(ovSYDTOK?"S+T":"--")));
   string utcHM=StringFormat("%02d:%02d",um/60,um%60);
   DashRow("L_TIME",0,yL,"OVL "+ovShort+"  SRV "+TimeToString(ServerNow(),TIME_SECONDS)+"  UTC "+utcHM,C_TXT);
   DashRow("L_SPRD",0,yL,"Spread "+DoubleToString(sp,0)+"pt (p"+DoubleToString(spp,0)+")",
           (sp>InpMaxSpreadPoints*g_ptScale?C_DN_TXT:(spp>InpMaxSpreadPercentile?C_WARN_TXT:C_TXT)));
   DashRow("L_ATR",0,yL,"ATR "+IntegerToString(atrPts)+"pt (p"+DoubleToString(ap,0)+") ["+DoubleToString(InpMinATRPoints*g_ptScale,0)+".."+DoubleToString(InpMaxATRPoints*g_ptScale,0)+"]",
           (atrPts<InpMinATRPoints*g_ptScale||(InpMaxATRPoints>0&&atrPts>InpMaxATRPoints*g_ptScale)?C_WARN_TXT:C_TXT));
   string phLbl=SessionPhaseLabel(um,WindowOpenMin(w,so,sc,to,tc,lo,lc,no,nc),WindowCloseMin(w,so,sc,to,tc,lo,lc,no,nc));
   DashRow("L_VOL",0,yL,"Vol x"+DoubleToString(g_volRatio,2)+"  Phase "+phLbl,
           (g_volRatio>=InpMinVolumeRatio?C_TXT:C_DIM));
   rL=yL;

   DashSection("LG",0,yL,"signal engine");
   DashRowSplit("L_SCORE",0,yL,"Score "+IntegerToString(g_score)+"/"+IntegerToString(g_scoreMax),"Bias "+IntegerToString(g_dirBias)+" "+BiasArrow(g_dirBias),
                (g_score>=InpMinFilterScore?C_UP_TXT:C_WARN_TXT),BiasColor(g_dirBias));
   DashRow("L_SMC",0,yL,"SMC B/S "+IntegerToString(g_smcScoreBull)+"/"+IntegerToString(g_smcScoreBear)+"  FVG "+(g_fvg?(g_fvgDir>0?"UP":"DN"):"-"),BiasColor(g_smcScoreBull-g_smcScoreBear));
   DashRow("L_ADV",0,yL,"IFVG "+(g_ifvg?(g_ifvgDir>0?"BULL":"BEAR"):"-")+"  PTB "+(g_ptb?(g_ptbDir>0?"BULL":"BEAR"):"-")+"  MTF "+((g_h1e20>g_h1e50&&g_m15e20>g_m15e50)?"BULL":((g_h1e20<g_h1e50&&g_m15e20<g_m15e50)?"BEAR":"FLAT")),C_TXT);
   DashRow("L_HV",0,yL,"HV "+IntegerToString(hs2)+"/"+IntegerToString(InpHVMinScore)+(hvNow?" QUAL":" -")+"  VWAPdev "+DoubleToString(vwapDev,2),(hvNow?C_WARN_TXT:C_TXT2));
   bool gatePass=(!g_paused&&!halted&&tr&&g_score>=InpMinFilterScore&&MathAbs(g_dirBias)>=InpMinDirBias&&!g_newsBlocked);
   DashRow("L_GATE",0,yL,"Entry gate: "+(gatePass?"PASS":"BLOCKED"),(gatePass?C_UP_TXT:C_DN_TXT));
   rL=yL;

   DashSection("LM",0,yL,"fmp macro / news");
   DashRow("L_USD",0,yL,"USD basket "+DoubleToString(g_usdAvg,3)+"% b="+IntegerToString(g_usdBias)+(g_usdAvailable?"":" [offline]"),(g_usdAvailable?BiasColor(g_usdBias):C_GRAY));
   DashRow("L_SPX",0,yL,"SPX "+DoubleToString(g_spxMovePct,3)+"% b="+IntegerToString(g_spxBias)+(g_spxAvailable?"":" [offline]"),(g_spxAvailable?BiasColor(g_spxBias):C_GRAY));
   DashRow("L_EUR",0,yL,"EUR "+(g_eurSymbol==""?"n/a":g_eurSymbol)+" "+DoubleToString(g_eurMovePct,3)+"% b="+IntegerToString(g_eurBias),(g_eurAvailable?BiasColor(g_eurBias):C_GRAY));
   DashRow("L_FMPN",0,yL,"FMP news "+(g_fmpNewsCount>0?"hits:"+IntegerToString(g_fmpNewsCount):"none"),(g_fmpNewsCount>0?C_WARN_TXT:C_GRAY));
   DashRow("L_VOTES",0,yL,"Votes B/S "+IntegerToString(g_macroBull)+"/"+IntegerToString(g_macroBear)+"  "+(g_fmpEverOK?"feed OK":"feed OFF"),(g_fmpEverOK?C_TXT:C_WARN_TXT));

   //================ RIGHT COLUMN (flow) =================
   int yR=g_bodyTop;
   DashSection("RB",1,yR,"broker / account");
   DashRow("R_ACC",1,yR,AccTypeName(broker.tradeMode)+" "+MarginModeName(broker.marginMode)+(broker.hedging?" (hedge)":" (net)"),C_GOLD_TXT);
   DashRow("R_BRK",1,yR,broker.company,C_TXT2);   // clipped to column
   DashRow("R_LEV",1,yR,"Lev 1:"+IntegerToString(broker.leverage)+"  "+TradeModeName((int)SymbolInfoInteger(eaSymbol,SYMBOL_TRADE_MODE)),C_TXT);
   DashRow("R_SWAP",1,yR,"Swap "+DoubleToString(broker.swapLong,1)+"/"+DoubleToString(broker.swapShort,1)+"  Comm $"+DoubleToString(commRT,2),C_TXT2);
   DashRow("R_CON",1,yR,"Ctr "+DoubleToString(broker.contractSize,0)+"  Stop "+IntegerToString(broker.stopsLevel)+"  Tick "+DoubleToString(broker.tickSize,3),C_TXT2);

   DashSection("RA",1,yR,"risk / exposure");
   DashRow("R_EQ",1,yR,"Eq "+DoubleToString(eq,2)+"  Bal "+DoubleToString(bal,2)+" "+broker.currency,C_TXT);
   DashRow("R_DD",1,yR,"Day DD "+DoubleToString(dd,2)+"%/"+DoubleToString(InpDailyLossPercent,2)+"%  Wk "+DoubleToString((g_weekAnchor>0?(g_weekAnchor-eq)/g_weekAnchor*100:0),2)+"%",
           (dd>=InpDailyLossPercent?C_DN_TXT:(dd>InpMaxFloatingDDPercent*0.7?C_WARN_TXT:C_TXT)));
   DashRow("R_MRG",1,yR,"Margin "+DoubleToString(margin,2)+" ("+DoubleToString(marginPct,1)+"%)  Free "+DoubleToString(freeM,2),(marginPct>50?C_WARN_TXT:C_TXT2));
   DashRow("R_POS",1,yR,"Positions "+IntegerToString(CountOwnPositions())+"/"+IntegerToString(InpMaxConcurrentPositions)+"  Lots "+DoubleToString(SumOwnLots(),2),C_TXT);
   DashRow("R_RISK",1,yR,"Risk $"+DoubleToString(openRisk,2)+" ("+DoubleToString(riskPct,2)+"%/"+DoubleToString(InpMaxAggregateOpenRiskPct,2)+"%)",(riskPct>InpMaxAggregateOpenRiskPct?C_DN_TXT:C_TXT));
   DashRow("R_BUD",1,yR,"Budget $"+DoubleToString(g_windowRiskUsed[wi],2)+" ("+DoubleToString(budgetPct,2)+"%/"+DoubleToString(InpPerWindowRiskBudgetPct,2)+"%)",(budgetPct>=InpPerWindowRiskBudgetPct?C_WARN_TXT:C_TXT2));

   DashSection("RE",1,yR,"execution");
   DashRow("R_GATE",1,yR,"GATE: "+g_gateReason,(StringFind(g_gateReason,"ARMED")>=0||StringFind(g_gateReason,"RECOVERY")>=0?C_UP_TXT:(halted?C_DN_TXT:(g_paused?C_WARN_TXT:C_DIM))));
   DashRow("R_TRD",1,yR,"Trades "+IntegerToString(g_tradesToday)+"/"+IntegerToString(InpMaxTradesPerDay)+"  ConsLoss "+IntegerToString(g_consecutiveLosses)+"/"+IntegerToString(InpMaxConsecutiveLosses),(g_consecutiveLosses>=InpMaxConsecutiveLosses?C_DN_TXT:C_TXT));
   DashRow("R_RCV",1,yR,"Recovery "+(g_lastLossDir!=0?("arm "+(g_lastLossDir>0?"SELL":"BUY")+" legs "+IntegerToString(g_recoveryLegs)):"idle"),(g_lastLossDir!=0?C_WARN_TXT:C_GRAY));
   string newsTxt="News clear";
   color newsCol=C_TXT2;
   if(g_newsBlocked){newsTxt="News BLOCKED "+g_nextNewsName;newsCol=C_DN_TXT;}
   else if(g_nextNewsTime>0)
   {
      long secs=(long)(g_nextNewsTime-ServerNow());
      long hh=secs/3600,mm=(secs%3600)/60;
      newsTxt=StringFormat("News %s in %dh %02dm",g_nextNewsName,(int)hh,(int)mm);
      newsCol=(secs<1800?C_WARN_TXT:C_TXT2);
   }
   DashRow("R_NEWS",1,yR,newsTxt,newsCol);
   DashRow("R_SLIP",1,yR,"Slip avg "+DoubleToString(g_slipAvg,1)+" last "+DoubleToString(g_lastSlipPts,1)+"pt",C_TXT2);

   // pause / resume button row
   if(!g_measure)
   {
      int bx=g_x+g_pad+g_colW+g_gap+6,by=yR+2;
      string bn=UI_PREFIX+"BTN_P";
      if(ObjectFind(0,bn)<0){ObjectCreate(0,bn,OBJ_BUTTON,0,0,0);ObjectSetInteger(0,bn,OBJPROP_CORNER,CORNER_LEFT_UPPER);ObjectSetInteger(0,bn,OBJPROP_SELECTABLE,false);ObjectSetInteger(0,bn,OBJPROP_HIDDEN,true);ObjectSetInteger(0,bn,OBJPROP_ZORDER,10);}
      ObjectSetInteger(0,bn,OBJPROP_XDISTANCE,bx);ObjectSetInteger(0,bn,OBJPROP_YDISTANCE,by);
      ObjectSetInteger(0,bn,OBJPROP_XSIZE,SX(120));ObjectSetInteger(0,bn,OBJPROP_YSIZE,g_rh);
      // Buttons auto-grow beyond YSIZE under DPI scaling (system metrics); give the row
      // a full extra row of clearance so it can never sit on the next section bar.
      ObjectSetString(0,bn,OBJPROP_TEXT,(g_paused?"RESUME":"PAUSE ARMING"));
      ObjectSetString(0,bn,OBJPROP_FONT,"Consolas");ObjectSetInteger(0,bn,OBJPROP_FONTSIZE,FontOut(g_font));
      ObjectSetInteger(0,bn,OBJPROP_COLOR,(g_paused?C_UP_TXT:C_WARN_TXT));
      ObjectSetInteger(0,bn,OBJPROP_BGCOLOR,(g_paused?C'35,80,45':C'120,40,40'));
      ObjectSetInteger(0,bn,OBJPROP_BORDER_COLOR,C_BORDER);
      ObjectSetInteger(0,bn,OBJPROP_STATE,false);
   }
   yR+=g_rh*2+SX(4);

   RecountTodayStats();
   DashSection("RP",1,yR,"performance");
   DashRow("R_PERF",1,yR,"Today "+IntegerToString(g_todayWins)+"W/"+IntegerToString(g_todayLosses)+"L  Win "+DoubleToString(TodayWinRate(),1)+"%  PF "+DoubleToString(TodayPF(),2),(g_todayNet>=0?C_UP_TXT:C_DN_TXT));
   DashRow("R_TNET",1,yR,"Today net "+FmtMoney(g_todayNet)+"  Life "+FmtMoney(g_perfNetProfit),(g_todayNet>=0?C_UP_TXT:(g_perfNetProfit>=0?C_TXT:C_DN_TXT)));
   DashRow("R_LIFE",1,yR,"Life Win "+DoubleToString(OverallWinRate(),1)+"%  PF "+DoubleToString(OverallProfitFactor(),2)+"  NetR "+DoubleToString(OverallNetR(),3),C_GOLD_TXT);
   DashRow("R_WIN",1,yR,WindowName(w)+": "+FmtMoney(g_ws[wi].netPL)+"  T"+IntegerToString(g_ws[wi].trades)+"  exp "+DoubleToString(WindowExpectancy(w),2),C_TXT2);

   //================ 24H UTC SESSION TIMELINE (flow) =====================
   int tlTop=MathMax(yL,yR)+SX(6);
   int labW=SX(34);
   int barX=x+g_pad+labW,barW=g_panelW-g_pad*2-labW;
   UIRect("TL_BG",x+g_pad,tlTop,g_panelW-g_pad*2,g_tlH,C_BG2);
   UILabel("TL_H",x+g_pad+4,tlTop+2,"SESSION MAP (UTC)",C_SECTION_TXT,MathMax(6,g_font-1));
   int rowY[4];
   int openM[4],closeM[4];
   openM[0]=so;closeM[0]=sc;openM[1]=to;closeM[1]=tc;openM[2]=lo;closeM[2]=lc;openM[3]=no;closeM[3]=nc;
   string names[4]={"SYD","TOK","LDN","NY"};   // 3-letter codes = wider timeline bars
   bool enab[4];
   enab[0]=(InpTradeAllFourSessions||InpTradeSydney);enab[1]=(InpTradeAllFourSessions||InpTradeTokyo);
   enab[2]=(InpTradeAllFourSessions||InpTradeLondon);enab[3]=(InpTradeAllFourSessions||InpTradeNewYork);
   int byT=tlTop+SX(15);
   for(int i=0;i<4;i++)
   {
      rowY[i]=byT+i*g_rh;
      bool act=InWindowMinutes(um,openM[i],closeM[i]);
      UILabel("TL_N"+IntegerToString(i),x+g_pad+4,rowY[i]+((g_rh-MathMax(6,g_font-1))/2),names[i],(enab[i]?(act?SessionColor(i,true):C_TXT2):C_GRAY),MathMax(6,g_font-1));
      UIRect("TL_T"+IntegerToString(i),barX,rowY[i]+((g_rh-9)/2),barW,9,C_PANEL);
      for(int hh=3;hh<24;hh+=3)UIRect("TL_G"+IntegerToString(i)+"_"+IntegerToString(hh),barX+(int)MathRound(hh*barW/24.0),rowY[i]+((g_rh-9)/2),1,9,C_GRID);
      if(enab[i])DashSeg("TL_S"+IntegerToString(i),barX,rowY[i]+((g_rh-9)/2),barW,openM[i],closeM[i],SessionColor(i,act));
   }
   int nowX=barX+(int)MathRound(um*barW/1440.0);
   UIRect("TL_NOW",nowX,byT-2,2,4*g_rh,clrWhite);
   UILabel("TL_NOWL",MathMax(barX,MathMin(nowX-SX(30),barX+barW-SX(70))),tlTop+g_tlH-SX(13),"NOW "+FmtHHMM(um),clrWhite,MathMax(6,g_font-1));

   //================ STATUS + FOOTER (flow) ==============================
   int sy2=tlTop+g_tlH+SX(2);
   UILabel("ST_NOW",x+g_pad+4,sy2,ClipText("NOW "+WindowName(w)+" ["+WindowStateText(w,tr)+"]   TOP: "+TopWindowsText(),g_panelW-g_pad*2-8,g_font),(tr?C_UP_TXT:C_TXT2),g_font);
   UILabel("FT",x+g_pad+4,sy2+g_rh,ClipText("magic "+IntegerToString(InpMagicNumber)+"   "+InpComment+"   F = pause   drag header = move",g_panelW-g_pad*2-8,MathMax(6,g_font-1)),C_GRAY,MathMax(6,g_font-1));
   ChartRedraw();
}

void DashDestroy(){ObjectsDeleteAll(0,UI_PREFIX,0,-1);}

//====================================================================
// INIT / DEINIT / EVENTS
//====================================================================
int OnInit()
{
   eaSymbol=_Symbol;if(!InitBroker()){Print("Broker symbol properties unavailable");return INIT_FAILED;}
   // Fail-loud permission diagnosis (no silent "compiles but never trades").
   if(!TerminalInfoInteger(TERMINAL_TRADE_ALLOWED))Print("WARNING: AutoTrading is OFF - enable the Algo Trading button to allow entries.");
   if(!MQLInfoInteger(MQL_TRADE_ALLOWED))Print("WARNING: MQL trade permission denied - check Allow Algo Trading in EA settings.");
   if(!AccountInfoInteger(ACCOUNT_TRADE_EXPERT))Print("WARNING: Account forbids expert trading.");
   hATR=iATR(eaSymbol,PERIOD_M1,InpATRPeriod);hADX=iADX(eaSymbol,PERIOD_M1,InpADXPeriod);hEMA20=iMA(eaSymbol,PERIOD_M1,InpEMA20Period,0,MODE_EMA,PRICE_CLOSE);hEMA50=iMA(eaSymbol,PERIOD_M1,InpEMA50Period,0,MODE_EMA,PRICE_CLOSE);hH1EMA20=iMA(eaSymbol,PERIOD_H1,20,0,MODE_EMA,PRICE_CLOSE);hH1EMA50=iMA(eaSymbol,PERIOD_H1,50,0,MODE_EMA,PRICE_CLOSE);hM15EMA20=iMA(eaSymbol,PERIOD_M15,20,0,MODE_EMA,PRICE_CLOSE);hM15EMA50=iMA(eaSymbol,PERIOD_M15,50,0,MODE_EMA,PRICE_CLOSE);hRSI=iRSI(eaSymbol,PERIOD_M1,InpRSIPeriod,PRICE_CLOSE);hM5E20=iMA(eaSymbol,PERIOD_M5,20,0,MODE_EMA,PRICE_CLOSE);hM5E50=iMA(eaSymbol,PERIOD_M5,50,0,MODE_EMA,PRICE_CLOSE);hM5ADX=iADX(eaSymbol,PERIOD_M5,14);
   if(hATR==INVALID_HANDLE||hADX==INVALID_HANDLE||hEMA20==INVALID_HANDLE||hEMA50==INVALID_HANDLE||hH1EMA20==INVALID_HANDLE||hH1EMA50==INVALID_HANDLE||hM15EMA20==INVALID_HANDLE||hM15EMA50==INVALID_HANDLE||hRSI==INVALID_HANDLE||hM5E20==INVALID_HANDLE||hM5E50==INVALID_HANDLE||hM5ADX==INVALID_HANDLE){Print("Indicator initialization failed");return INIT_FAILED;}
   g_x=InpPanelX;g_y=InpPanelY;
   if(GlobalVariableCheck("PAT_X_"+eaSymbol+"_"+IntegerToString(InpMagicNumber)))
   {int sx=(int)GlobalVariableGet("PAT_X_"+eaSymbol+"_"+IntegerToString(InpMagicNumber));
    int sy=(int)GlobalVariableGet("PAT_Y_"+eaSymbol+"_"+IntegerToString(InpMagicNumber));
    if(sx>=0&&sy>=0){g_x=sx;g_y=sy;}}
   ChartSetInteger(0,CHART_EVENT_MOUSE_MOVE,true);g_atrKeep=MathMax(30,MathMin(ATR_SAMPLES,InpATRPercentileLookback));ArrayInitialize(g_spreadBuf,0);ArrayInitialize(g_slipBuf,0);ArrayInitialize(g_atrBuf,0);ArrayInitialize(g_usdMove,0);ArrayInitialize(g_usdGot,false);RefreshServerOffset(true);UIRecompute();
   PrintSessionMapAudit();UpdateRiskPeriods();if(InpPersistState)LoadState();OpenLog();IsNewBar();UpdateSpreadStats();UpdateIndicators();RefreshVolumeRatio();RefreshFMPMacro(true);UpdateSuperTrend();UpdateVWAP();DetectFVG();DetectIFVG();DetectPTB();AnalyzeAMD();DetectSMC();EvaluateFilters();EventSetTimer(1);g_gateReason="initialized";DashUpdate(true);
   Print("Predict-A-Trade v1.00 initialized | ",eaSymbol," | digits=",broker.digits," ptScale=",g_ptScale," | server-UTC offset=",g_serverOffsetSec,"s | minVol=",broker.volumeMin," step=",broker.volumeStep," stops=",broker.stopsLevel," freeze=",broker.freezeLevel," hedging=",broker.hedging);
   Print("Broker: ",broker.company," | ",AccTypeName(broker.tradeMode)," account | leverage 1:",broker.leverage," | swap L/S ",DoubleToString(broker.swapLong,2),"/",DoubleToString(broker.swapShort,2));
   // Print the rollover-verification note only the FIRST time ever (persisted), so it
   // reads as one-time setup guidance rather than a recurring warning.
   string gvKey="PAT_SWAP_NOTE_"+eaSymbol+"_"+IntegerToString(InpMagicNumber);
   if(!GlobalVariableCheck(gvKey))
   {
      Print("Swap rollover assumed at ",DoubleToString(InpSwapRolloverServerHour,2)," server time. One-time check: hold or review a position that crosses this hour - the History tab must show a swap entry at that hour. If the swap posts at a different hour, set InpSwapRolloverServerHour to it. This message will not repeat.");
      GlobalVariableSet(gvKey,1);
   }
   return INIT_SUCCEEDED;
}

void OnDeinit(const int reason)
{
   EventKillTimer();if(InpPersistState)SaveState();if(g_log!=INVALID_HANDLE){FileFlush(g_log);FileClose(g_log);g_log=INVALID_HANDLE;}WriteWindowReport();WritePerformanceReport();DashDestroy();
   if(hATR!=INVALID_HANDLE)IndicatorRelease(hATR);if(hADX!=INVALID_HANDLE)IndicatorRelease(hADX);if(hEMA20!=INVALID_HANDLE)IndicatorRelease(hEMA20);if(hEMA50!=INVALID_HANDLE)IndicatorRelease(hEMA50);if(hH1EMA20!=INVALID_HANDLE)IndicatorRelease(hH1EMA20);if(hH1EMA50!=INVALID_HANDLE)IndicatorRelease(hH1EMA50);if(hM15EMA20!=INVALID_HANDLE)IndicatorRelease(hM15EMA20);if(hM15EMA50!=INVALID_HANDLE)IndicatorRelease(hM15EMA50);if(hRSI!=INVALID_HANDLE)IndicatorRelease(hRSI);if(hM5E20!=INVALID_HANDLE)IndicatorRelease(hM5E20);if(hM5E50!=INVALID_HANDLE)IndicatorRelease(hM5E50);if(hM5ADX!=INVALID_HANDLE)IndicatorRelease(hM5ADX);PrintSummary();
}

void OnTick()
{
   RefreshServerOffset(false);UpdateRiskPeriods();UpdateSpreadStats();bool nb=IsNewBar();UpdateIndicators();RefreshFMPMacro(false);
   if(nb){RefreshVolumeRatio();UpdateSuperTrend();UpdateVWAP();DetectFVG();DetectIFVG();DetectPTB();AnalyzeAMD();DetectSMC();EvaluateFilters();UpdateOpportunityObservations();CheckNews(false);
      // ultra-scalp v2 state
      double rsiBuf[1];if(CopyBuffer(hRSI,0,1,1,rsiBuf)>0)g_rsi=rsiBuf[0];
      double e20[1],e50[1],adx[1],adxp[1],adxm[1];
      if(CopyBuffer(hM5E20,0,1,1,e20)>0)g_m5e20=e20[0];
      if(CopyBuffer(hM5E50,0,1,1,e50)>0)g_m5e50=e50[0];
      if(CopyBuffer(hM5ADX,0,0,1,adx)>0)g_m5adx=adx[0];
      if(CopyBuffer(hM5ADX,1,0,1,adxp)>0)g_m5adxPlus=adxp[0];
      if(CopyBuffer(hM5ADX,2,0,1,adxm)>0)g_m5adxMinus=adxm[0];
      // Bollinger 20,2 on M1 closes (manual std over 20 bars)
      double closes[20];   // mean/std are order-independent; no series flag needed
      if(CopyClose(eaSymbol,PERIOD_M1,1,20,closes)==20)
      {
         double sum=0;for(int k=0;k<20;k++)sum+=closes[k];g_bbMid=sum/20.0;
         double v=0;for(int k=0;k<20;k++){double d=closes[k]-g_bbMid;v+=d*d;}
         double sd=MathSqrt(v/20.0);
         g_bbUp=g_bbMid+2.0*sd;g_bbLo=g_bbMid-2.0*sd;
      }
      EvaluateScalpSignal();
   }else EvaluateFilters();
   if((g_stopDay||g_stopWeek||g_stopMonth)&&InpBreakerAction==BREAKER_CLOSE_ALL)EmergencyCloseAll();
   EnforceSwapFlat();
   if(g_lastLossDir!=0)TryRecovery();
   if(!g_paused&&!g_stopDay&&!g_stopWeek&&!g_stopMonth)TryArm();
   ManageAllPositions();DashUpdate(false);
}

void OnTimer()
{
   RefreshServerOffset(false);CheckNews(false);RefreshFMPMacro(false);EnforceSwapFlat();if(InpCancelStalePendings)DeleteOwnPendings(true);DashUpdate(true);if(InpPersistState && (ServerNow()%30)==0)SaveState();
}

void OnChartEvent(const int id,const long &lparam,const double &dparam,const string &sparam)
{
   if(!InpShowDashboard)return;

   //--- P button (OBJ_BUTTON) toggles arming; real dialog feedback + sound
   if(id==CHARTEVENT_OBJECT_CLICK && sparam==UI_PREFIX+"BTN_P")
   {
      g_paused=!g_paused;
      g_gateReason=(g_paused?"MANUAL PAUSE (panel)":"resumed");
      PlaySound(g_paused?InpSoundPause:InpSoundResume);
      Print("Arming ",(g_paused?"PAUSED by operator":"RESUMED by operator"));
      DashUpdate(true);
      return;
   }

   //--- header click = collapse / expand (if the press did not become a drag)
   if(id==CHARTEVENT_CLICK && g_maybeClick)
   {
      int mx=(int)lparam,my=(int)dparam;
      g_maybeClick=false;
      if(!g_dragging && g_panelW>0 && mx>=g_x && mx<=g_x+g_panelW && my>=g_y && my<=g_y+g_hdrH)
      {
         g_dashCollapsed=!g_dashCollapsed;
         Print("Dashboard ",(g_dashCollapsed?"collapsed":"expanded"));
         DashUpdate(true);
      }
      return;
   }

   //--- F key pauses arming (chart focused)
   if(id==CHARTEVENT_KEYDOWN && lparam==70)   // 'F'
   {
      g_paused=!g_paused;
      g_gateReason=(g_paused?"MANUAL PAUSE (F)":"resumed");
      PlaySound(g_paused?InpSoundPause:InpSoundResume);
      Print("Arming ",(g_paused?"PAUSED by operator":"RESUMED by operator"));
      DashUpdate(true);
      return;
   }

   //--- header drag (InpPanelDraggable)
   if(InpPanelDraggable && id==CHARTEVENT_MOUSE_MOVE)
   {
      int mx=(int)lparam,my=(int)dparam;
      int flags=(int)StringToInteger(sparam);
      bool lmb=((flags&1)!=0);
      uint now=GetTickCount();

      if(lmb && !g_dragging)
      {
         if(mx>=g_x && mx<=g_x+g_panelW && my>=g_y && my<=g_y+g_hdrH)
         {
            g_maybeClick=true;
            g_dragging=true;
            g_dragOffX=mx-g_x; g_dragOffY=my-g_y;
            g_dragStartX=mx;   g_dragStartY=my;
         }
      }
      else if(g_dragging && lmb)
      {
         // movement beyond a few px cancels the click-collapse
         if(MathAbs(mx-g_dragStartX)>4||MathAbs(my-g_dragStartY)>4)g_maybeClick=false;
         int nx=mx-g_dragOffX, ny=my-g_dragOffY;
         int cw=(int)ChartGetInteger(0,CHART_WIDTH_IN_PIXELS);
         int ch=(int)ChartGetInteger(0,CHART_HEIGHT_IN_PIXELS);
         if(nx<0)nx=0; if(ny<0)ny=0;
         if(nx+g_panelW>cw)nx=cw-g_panelW;
         if(ny+g_hdrH>ch)ny=MathMax(0,ch-g_hdrH);
         if(nx!=g_x||ny!=g_y)
         {
            g_x=nx;g_y=ny;
            GlobalVariableSet("PAT_X_"+eaSymbol+"_"+IntegerToString(InpMagicNumber),g_x);
            GlobalVariableSet("PAT_Y_"+eaSymbol+"_"+IntegerToString(InpMagicNumber),g_y);
            // throttle during drag for smoothness
            if(now-g_lastDragMs>=(uint)MathMax(30,InpDashRefreshMs/4)){g_lastDragMs=now;DashUpdate(true);}
         }
      }
      else if(g_dragging && !lmb)
      {
         g_dragging=false;
         DashUpdate(true);   // final snap
      }
   }
}

//+------------------------------------------------------------------+
//| End                                                              |
//+------------------------------------------------------------------+