// Package types defines the minimal market-state and signal contracts used by
// pat-engine. It is intentionally dependency-free so the engine can run with no
// external services or databases.
package types

// Direction is the trade direction emitted by a strategy.
type Direction string

const (
	DirBuy     Direction = "BUY"
	DirSell    Direction = "SELL"
	DirNoTrade Direction = "NO_TRADE"
	DirWait    Direction = "WAIT"
	DirError   Direction = "ERROR"
)

// Timeframe identifies a candle timeframe.
type Timeframe string

const (
	TFM1  Timeframe = "M1"
	TFM5  Timeframe = "M5"
	TFM15 Timeframe = "M15"
	TFH1  Timeframe = "H1"
)

// StrategyID uniquely identifies a strategy product.
type StrategyID string

const (
	StrategyUltraScalping     StrategyID = "ULTRA_SCALPING"
	StrategyStandardScalping  StrategyID = "STANDARD_SCALPING"
	StrategyStandardSwing     StrategyID = "STANDARD_SWING"
	StrategyTrendSwing        StrategyID = "TREND_SWING"
)

// Candle carries micro-structure flags used by scalping strategies.
type Candle struct {
	IsDisplacement     bool
	IsBullish          bool
	IsBearish          bool
	IsRejection        bool
	PinBarQuality      float64
	PinBarRejDirection string
}

// Sweep is a liquidity sweep observation.
type Sweep struct {
	Direction string // "BUY_SIDE" or "SELL_SIDE"
}

// Indicators holds the technical indicators a strategy consumes. Optional fields
// (SMA200, EMA100/200, CCI, ParabolicSAR) are zero when the data provider does
// not supply them — strategies simply skip the corresponding evidence.
type Indicators struct {
	EMA9, EMA21, EMA50      float64
	EMA100, EMA200          float64
	SMA200                  float64
	ADX, ADXPlusDI, ADXMinusDI float64
	RSI                     float64
	MACDMain, MACDSignal    float64
	OsMA                    float64
	StochMain, StochSignal  float64
	BollUpper, BollLower    float64
	CCI                     float64
	ParabolicSAR            float64
	ParabolicSARLong        bool
}

// Structure holds price-structure context.
type Structure struct {
	LastBOS     *BOS
	SwingLows   []float64
	SwingHighs  []float64
	LastCHoCH   *CHoCH
	LastMSS     *MSS
	CurrentTrend string
}

type BOS struct{ Direction string }
type CHoCH struct{}
type MSS struct{}

// Liquidity holds recent liquidity sweeps.
type Liquidity struct{ RecentSweeps []Sweep }

// Session describes the trading session context.
type Session struct {
	CurrentSession string
	IsOverlap      bool
	NewsRisk       string // "", "LOW", "MEDIUM", "HIGH", "BLOCKED"
}

// MarketState is a single point-in-time snapshot evaluated by strategies.
// It is the single contract between the data provider and the strategy engine.
type MarketState struct {
	Symbol       string
	Timeframe    Timeframe
	CurrentPrice float64
	Open         float64
	High         float64
	Low          float64
	Close        float64
	H1Close      float64 // H1 close used for the higher-timeframe trend veto
	Spread       float64
	ATR          float64
	Indicators   Indicators
	Structure    Structure
	Candle       Candle
	Liquidity    Liquidity
	MTFScore     float64
	Regime       string
	Session      Session
	Quality      string
	VWAP         float64
}

// Signal is the engine's output (executable or blocked).
type Signal struct {
	StrategyID  StrategyID
	Direction   Direction
	EntryPrice  float64
	StopLoss    float64
	TP1, TP2, TP3 float64
	RawScore    float64
	Reason      string
	ReasonCodes []string
	Executable  bool
}
