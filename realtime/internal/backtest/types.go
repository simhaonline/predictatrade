// Package backtest implements historical strategy backtesting using the
// EXACT production feature engine and strategy evaluators.
//
// This is NOT a Python approximation. It feeds real historical candles
// through the real Go feature.Registry and strategy.Strategy implementations,
// guaranteeing 100% parity with live signal generation.
//
// Data source: market.candles table (PostgreSQL) or CSV files.
// Output: trading.backtest_runs + trading.backtest_trades tables.
package backtest

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// BacktestConfig configures a single backtest run.
type BacktestConfig struct {
	Symbol          string
	StrategyID      types.StrategyID
	PrimaryTimeframe types.Timeframe
	HigherTimeframes []types.Timeframe
	StartTime       time.Time
	EndTime         time.Time
	InitialBalance  decimal.Decimal

	// Execution costs
	Spread         decimal.Decimal // typical spread in price units
	Commission     decimal.Decimal // round-turn commission per lot
	ContractSize   decimal.Decimal // ounces per lot
	Slippage      decimal.Decimal // fixed slippage per trade
	MaxSlippagePoints decimal.Decimal // max slippage in price points (reject if exceeded)
	RolloverSlippageMult decimal.Decimal // slippage multiplier during rollover (spread widens)

	// Risk management
	MaxRiskPerTrade  decimal.Decimal // fraction of equity (e.g. 0.02)
	MaxPositions     int
	MaxDailyLossPct  decimal.Decimal // e.g. 2.0

	// Exit management
	TrailingStopEnabled bool
	TrailingATRMult      decimal.Decimal
	BreakEvenEnabled     bool
	BreakEvenTriggerR    decimal.Decimal

	// Same-bar SL/TP policy: if both SL and TP are within the candle range,
	// assume SL is hit first (conservative).
	ConservativeSLTP bool

		// Swap costs (NEW v1.06 — model overnight swap charges)
	SwapPerLotPerDay  decimal.Decimal // swap charge per lot per overnight holding
	TripleSwapDay     string         // day that gets 3x swap (e.g. "Wednesday")
	AvoidOvernight    bool           // close positions before end of day to avoid swap

	// Data source
	DataSource string // "database" or "csv"
	DBUrl      string
	CSVDir     string

	// Identification
	RunID    string
	GitCommit string
}

// DefaultConfig returns a sensible default for XAUUSD backtesting.
func DefaultConfig() BacktestConfig {
	return BacktestConfig{
		Symbol:            types.SymbolXAUUSD,
		StrategyID:        types.StrategyStandardScalping,
		PrimaryTimeframe:   types.TFM5,
		HigherTimeframes:   []types.Timeframe{types.TFM15, types.TFH1, types.TFH4, types.TFD1},
		InitialBalance:     decimal.NewFromInt(10000),
		Spread:            decimal.NewFromFloat(0.30),
		Commission:        decimal.NewFromFloat(7.0),
		ContractSize:      decimal.NewFromInt(100),
		Slippage:          decimal.NewFromFloat(0.05),
		MaxSlippagePoints: decimal.NewFromFloat(0.15), // max 15 cents slippage
		RolloverSlippageMult: decimal.NewFromFloat(3.0), // 3x slippage during rollover
		MaxRiskPerTrade:   decimal.NewFromFloat(0.02),
		MaxPositions:      3,
		MaxDailyLossPct:   decimal.NewFromFloat(5.0), // 5% max daily loss — capital protection
		TrailingStopEnabled: true,
		TrailingATRMult:      decimal.NewFromFloat(2.0),
		BreakEvenEnabled:     true,
		BreakEvenTriggerR:    decimal.NewFromInt(1),
		ConservativeSLTP:     true,
		SwapPerLotPerDay:    decimal.NewFromFloat(2.0), // $2/lot/day typical XAUUSD
		TripleSwapDay:       "Wednesday",
		AvoidOvernight:      true,
		DataSource:          "database",
	}
}

// Trade represents a single completed trade.
type Trade struct {
	TradeID      string
	StrategyID   types.StrategyID
	Direction    types.Direction
	EntryPrice   decimal.Decimal
	ExitPrice    decimal.Decimal
	StopLoss     decimal.Decimal
	TP1          decimal.Decimal
	TP2          decimal.Decimal
	TP3          decimal.Decimal
	EntryTime    time.Time
	ExitTime     time.Time
	ExitReason   string // "TP1", "TP2", "TP3", "SL", "TIMEOUT", "EOD"
	Size         decimal.Decimal
	RealizedPnL  decimal.Decimal
	RealizedR    decimal.Decimal // PnL / risk
	SpreadCost   decimal.Decimal
	CommissionCost decimal.Decimal
	SlippageCost decimal.Decimal
	Regime       types.Regime
	Session      string
	RawScore     decimal.Decimal
	EntryBarIdx  int
	ExitBarIdx   int
	HoldingBars  int
	SwapCost    decimal.Decimal // total swap charges accrued
}

// OpenPosition tracks a trade that hasn't closed yet.
type OpenPosition struct {
	TradeID      string
	StrategyID   types.StrategyID
	Direction    types.Direction
	EntryPrice   decimal.Decimal
	StopLoss     decimal.Decimal
	OriginalSL   decimal.Decimal
	TP1          decimal.Decimal
	TP2          decimal.Decimal
	TP3          decimal.Decimal
	EntryTime    time.Time
	EntryBarIdx  int
	Size         decimal.Decimal
	Regime       types.Regime
	Session      string
	RawScore     decimal.Decimal
	TP1Hit       bool
	TP2Hit       bool
	TP3Hit       bool
	BreakEvenSet bool
}

// Metrics holds computed performance metrics.
type Metrics struct {
	TotalTrades      int
	Wins             int
	Losses           int
	WinRate          decimal.Decimal
	NetProfit        decimal.Decimal
	TotalReturnPct   decimal.Decimal
	ProfitFactor     decimal.Decimal
	SharpeRatio      decimal.Decimal
	SortinoRatio     decimal.Decimal
	MaxDrawdownPct   decimal.Decimal
	AvgWin           decimal.Decimal
	AvgLoss          decimal.Decimal
	Expectancy       decimal.Decimal
	BestTrade        decimal.Decimal
	WorstTrade       decimal.Decimal
	AvgHoldingBars   int
	BuyCount         int
	SellCount        int
	FinalBalance     decimal.Decimal
	MaxConsecutiveWins  int
	MaxConsecutiveLosses int
}

// BacktestResult is the complete output of a backtest run.
type BacktestResult struct {
	RunID         string
	Config        BacktestConfig
	Trades       []Trade
	NoTradeCount  int
	BlockedCount  int
	BuySignals   int
	SellSignals  int
	BarsProcessed int
	Metrics      Metrics
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Status       string // "COMPLETED", "FAILED"
	Error        string
}

// SignalDecision records what each strategy decided at each bar.
type SignalDecision struct {
	BarIdx      int
	Timestamp   time.Time
	StrategyID  types.StrategyID
	Direction   types.Direction
	RawScore    decimal.Decimal
	ReasonCodes []types.NoTradeReason
	Regime      types.Regime
	Session     string
}
