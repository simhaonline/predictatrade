//+------------------------------------------------------------------+
//| PAT_Config.mqh — Internal configuration (DO NOT MODIFY)           |
//| All trading parameters are managed by the Go backend engine.     |
//| This file is auto-generated — changes here have no effect on     |
//| risk, strategy, or trade management.                             |
//+------------------------------------------------------------------+
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
