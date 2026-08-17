// Package types defines canonical types used across the Predict-A-Trade real-time engine.
// SOW Sections 6, 8, 10, 11, 12, 15, 16, 17, 19, 24, 25, 27, 49
package types

import (
	"time"

	"github.com/shopspring/decimal"
)

// Symbol is the canonical trading instrument identifier.
const SymbolXAUUSD = "XAUUSD"

// Direction represents a trade direction.
type Direction string

const (
	DirectionBuy     Direction = "BUY"
	DirectionSell    Direction = "SELL"
	DirectionNoTrade Direction = "NO-TRADE"
)

// StrategyID identifies one of the four canonical strategy products.
type StrategyID string

const (
	StrategyStandardScalping StrategyID = "STANDARD_SCALPING"
	StrategyUltraScalping    StrategyID = "ULTRA_SCALPING"
	StrategyStandardSwing    StrategyID = "STANDARD_SWING"
	StrategyTrendSwing       StrategyID = "TREND_SWING"
)

// AllStrategies returns the four canonical strategy IDs.
func AllStrategies() []StrategyID {
	return []StrategyID{
		StrategyStandardScalping,
		StrategyUltraScalping,
		StrategyStandardSwing,
		StrategyTrendSwing,
	}
}

// Tick represents a single market data tick (SOW Section 6).
type Tick struct {
	Symbol           string
	Bid              decimal.Decimal
	Ask              decimal.Decimal
	Mid              decimal.Decimal
	Spread           decimal.Decimal
	TickVolume       int64
	Source           string
	SourceTimestamp  time.Time
	GatewayTimestamp time.Time
	Quality          QualityState
	Sequence         uint64
}

// QualityState represents data quality classification (SOW Section 6A.8).
type QualityState string

const (
	QualityAuthoritative QualityState = "AUTHORITATIVE"
	QualityDerived       QualityState = "DERIVED"
	QualityEstimated     QualityState = "ESTIMATED"
	QualityDegraded      QualityState = "DEGRADED"
	QualityStale         QualityState = "STALE"
	QualityUnavailable   QualityState = "UNAVAILABLE"
	QualityInvalid       QualityState = "INVALID"
)

// Candle represents an OHLC candle (SOW Section 8).
type Candle struct {
	Symbol      string
	Timeframe   Timeframe
	Time        time.Time
	Open        decimal.Decimal
	High        decimal.Decimal
	Low         decimal.Decimal
	Close       decimal.Decimal
	Volume      int64
	Source      string
	Quality     CandleQuality
	IsClosed    bool
	Alignment   AlignmentProfile
}

type CandleQuality string

const (
	CandleComplete  CandleQuality = "COMPLETE"
	CandlePartial   CandleQuality = "PARTIAL"
	CandleEstimated CandleQuality = "ESTIMATED"
	CandleStale     CandleQuality = "STALE"
	CandleInvalid   CandleQuality = "INVALID"
)

// Timeframe represents a candle timeframe (SOW Section 10).
type Timeframe string

const (
	TFM1  Timeframe = "M1"
	TFM5  Timeframe = "M5"
	TFM15 Timeframe = "M15"
	TFM30 Timeframe = "M30"
	TFH1  Timeframe = "H1"
	TFH4  Timeframe = "H4"
	TFD1  Timeframe = "D1"
	TFW1  Timeframe = "W1"
	TFMN1 Timeframe = "MN1"
)

// AlignmentProfile defines candle bucket alignment (SOW Section 150.3).
type AlignmentProfile string

const (
	AlignmentBrokerUTCPlus3 AlignmentProfile = "BROKER_ALIGNED_UTC_PLUS_3"
	AlignmentUTC            AlignmentProfile = "UTC_ALIGNED"
	AlignmentVenue          AlignmentProfile = "VENUE_ALIGNED"
	AlignmentSourceNative   AlignmentProfile = "SOURCE_NATIVE"
)

// Regime represents the current market regime (SOW Section 11).
type Regime string

const (
	RegimeTrendingBullish Regime = "TRENDING_BULLISH"
	RegimeTrendingBearish Regime = "TRENDING_BEARISH"
	RegimeRange           Regime = "RANGE"
	RegimeBreakout        Regime = "BREAKOUT"
	RegimeMeanReversion   Regime = "MEAN_REVERSION"
	RegimeHighVolatility  Regime = "HIGH_VOLATILITY"
	RegimeLowVolatility   Regime = "LOW_VOLATILITY"
	RegimeLiquidityEvent  Regime = "LIQUIDITY_EVENT"
	RegimeNewsEvent       Regime = "NEWS_EVENT"
	RegimeUnstable        Regime = "UNSTABLE"
	RegimeNoTrade         Regime = "NO_TRADE"
)

// SignalGrade represents a signal quality grade (SOW Section 17).
type SignalGrade string

const (
	GradeAPlus  SignalGrade = "A+"
	GradeA      SignalGrade = "A"
	GradeB      SignalGrade = "B"
	GradeC      SignalGrade = "C"
	GradeNoTrade SignalGrade = "NO-TRADE"
	GradeResearch SignalGrade = "RESEARCH"
	GradeUnrated  SignalGrade = "UNRATED"
	GradeShadow   SignalGrade = "SHADOW"
)

// SignalStatus represents the signal lifecycle state (SOW Section 19).
type SignalStatus string

const (
	SignalDetected      SignalStatus = "DETECTED"
	SignalValidating    SignalStatus = "VALIDATING"
	SignalCandidate     SignalStatus = "CANDIDATE"
	SignalAIVerifying   SignalStatus = "AI_VERIFYING"
	SignalCalibrating   SignalStatus = "CALIBRATING"
	SignalRiskCheck     SignalStatus = "RISK_CHECK"
	SignalConfirmed     SignalStatus = "CONFIRMED"
	SignalActive        SignalStatus = "ACTIVE"
	SignalTriggered     SignalStatus = "TRIGGERED"
	SignalExpired       SignalStatus = "EXPIRED"
	SignalInvalidated   SignalStatus = "INVALIDATED"
	SignalCancelled     SignalStatus = "CANCELLED"
	SignalOrderSent     SignalStatus = "ORDER_SENT"
	SignalAcknowledged  SignalStatus = "ACKNOWLEDGED"
	SignalFilled        SignalStatus = "FILLED"
	SignalPartiallyFilled SignalStatus = "PARTIALLY_FILLED"
	SignalTP1           SignalStatus = "TP1"
	SignalTP2           SignalStatus = "TP2"
	SignalTP3           SignalStatus = "TP3"
	SignalStopped       SignalStatus = "STOPPED"
	SignalClosed        SignalStatus = "CLOSED"
)

// GateResult represents the outcome of a hard gate evaluation (SOW Section 131).
type GateResult string

const (
	GatePass     GateResult = "PASS"
	GateVeto     GateResult = "VETO"
	GateDegraded GateResult = "DEGRADED"
	GateUnknown  GateResult = "UNKNOWN"
)

// GateID identifies a specific hard gate (SOW Section 131.4).
type GateID string

const (
	GateDataQuality       GateID = "data_quality"
	GateSession           GateID = "session"
	GateNews              GateID = "news"
	GateSpread            GateID = "spread"
	GateSlippage          GateID = "slippage"
	GateTotalCost         GateID = "total_cost"
	GateExposure          GateID = "exposure"
	GateMargin            GateID = "margin"
	GateRRNetExpectancy   GateID = "rr_net_expectancy"
	GateEntitlement       GateID = "entitlement"
	GateLicense           GateID = "license"
	GateExecutionPermit   GateID = "execution_permission"
)

// NoTradeReason represents a standardized NO-TRADE reason (SOW Section 18).
type NoTradeReason string

const (
	NTInsufficientScore      NoTradeReason = "INSUFFICIENT_SCORE"
	NTConflictingTimeframes  NoTradeReason = "CONFLICTING_TIMEFRAMES"
	NTHighNewsRisk           NoTradeReason = "HIGH_NEWS_RISK"
	NTExtremeVolatility      NoTradeReason = "EXTREME_VOLATILITY"
	NTLowLiquidity           NoTradeReason = "LOW_LIQUIDITY"
	NTHighSpread             NoTradeReason = "HIGH_SPREAD"
	NTPoorRR                 NoTradeReason = "POOR_RR"
	NTUnclearStructure       NoTradeReason = "UNCLEAR_STRUCTURE"
	NTDXYConflict            NoTradeReason = "DXY_CONFLICT"
	NTYieldConflict          NoTradeReason = "YIELD_CONFLICT"
	NTStaleData              NoTradeReason = "STALE_DATA"
	NTFeedDegraded           NoTradeReason = "FEED_DEGRADED"
	NTAIDisagreement         NoTradeReason = "AI_DISAGREEMENT"
	NTSignalExpired          NoTradeReason = "SIGNAL_EXPIRED"
	NTSessionUnsuitable      NoTradeReason = "SESSION_UNSUITABLE"
	NTExecutionUnavailable   NoTradeReason = "EXECUTION_UNAVAILABLE"
	NTBrokerUnavailable      NoTradeReason = "BROKER_UNAVAILABLE"
	NTRiskLimitReached       NoTradeReason = "RISK_LIMIT_REACHED"
	NTLicenseRestricted      NoTradeReason = "LICENSE_RESTRICTED"
	NTAccountNotAuthorized   NoTradeReason = "ACCOUNT_NOT_AUTHORIZED"
	NTSystemDegraded         NoTradeReason = "SYSTEM_DEGRADED"
	NTTotalCostExceeded      NoTradeReason = "TOTAL_COST_EXCEEDED"
	NTMarginInsufficient     NoTradeReason = "MARGIN_INSUFFICIENT"
	NTBrokerProfileMismatch  NoTradeReason = "BROKER_PROFILE_MISMATCH"
	NTGateDegraded           NoTradeReason = "GATE_DEGRADED"
	NTGateUnknown            NoTradeReason = "GATE_UNKNOWN"
)

// EvidenceContribution represents a single pillar's contribution to a signal score (SOW Section 12C.3).
type EvidenceContribution struct {
	Pillar         string          `json:"pillar"`
	Feature        string          `json:"feature"`
	RawValue       decimal.Decimal `json:"raw_value"`
	NormalizedValue decimal.Decimal `json:"normalized_value"`
	Direction      Direction       `json:"direction"`
	Weight         decimal.Decimal `json:"weight"`
	Contribution   decimal.Decimal `json:"contribution"`
	Mandatory      bool            `json:"mandatory"`
	Quality        QualityState    `json:"quality"`
	Source         string          `json:"source"`
	Version        string          `json:"version"`
	ReasonCode     string          `json:"reason_code"`
}

// Signal represents a complete trading signal (SOW Sections 15, 16, 64).
type Signal struct {
	ID                  string
	Symbol              string
	StrategyID          StrategyID
	StrategyDefinitionID string
	Direction           Direction
	Grade               SignalGrade
	RawScore            decimal.Decimal
	LongScore           decimal.Decimal
	ShortScore          decimal.Decimal
	CalibratedProbability decimal.Decimal
	EntryPrice          decimal.Decimal
	EntryZoneLow        decimal.Decimal
	EntryZoneHigh       decimal.Decimal
	StopLoss            decimal.Decimal
	TP1                 decimal.Decimal
	TP2                 decimal.Decimal
	TP3                 decimal.Decimal
	GrossRRTP1          decimal.Decimal
	GrossRRTP2          decimal.Decimal
	GrossRRTP3          decimal.Decimal
	NetRRTP1            decimal.Decimal
	NetRRTP2            decimal.Decimal
	NetRRTP3            decimal.Decimal
	ExpectedCost        decimal.Decimal
	Regime              Regime
	Session             string
	NewsRisk            string
	Timeframe           Timeframe
	TTL                 time.Duration
	Status              SignalStatus
	ReasonCodes         []NoTradeReason
	Evidence            []EvidenceContribution
	GateResults         []GateEvaluation
	CreatedAt           time.Time
	ExpiresAt           time.Time
	ExitProfileID       string
	GatePolicyVersion   string
}

// GateEvaluation records the result of a single gate check (SOW Section 131).
type GateEvaluation struct {
	GateID        GateID      `json:"gate_id"`
	Result        GateResult  `json:"result"`
	ReasonCodes   []string    `json:"reason_codes"`
	EvaluatedAt   time.Time   `json:"evaluated_at"`
	FreshnessMs   int64       `json:"freshness_ms"`
	StateVersion  string      `json:"state_version"`
}

// Capability represents a data feed capability (SOW Section 6A.1).
type Capability string

const (
	CapSpotBidAsk          Capability = "SPOT_BID_ASK"
	CapSpotBrokerTickVolume Capability = "SPOT_BROKER_TICK_VOLUME"
	CapGCTrades            Capability = "GC_TRADES"
	CapGCTopOfBook         Capability = "GC_TOP_OF_BOOK"
	CapGCMarketByPrice     Capability = "GC_MARKET_BY_PRICE"
	CapGCMarketByOrder     Capability = "GC_MARKET_BY_ORDER"
	CapDXY                 Capability = "DXY"
	CapNominalYields       Capability = "NOMINAL_YIELDS"
	CapRealYields          Capability = "REAL_YIELDS"
	CapEconomicCalendar    Capability = "ECONOMIC_CALENDAR"
	CapNews                Capability = "NEWS"
	CapCOT                 Capability = "COT"
	CapETFFlows            Capability = "ETF_FLOWS"
	CapCentralBankFlow     Capability = "CENTRAL_BANK_FLOW"
)

// ExecutionMode represents the execution permission level (SOW Section 26).
type ExecutionMode string

const (
	ExecSignalOnly      ExecutionMode = "SIGNAL_ONLY"
	ExecManual          ExecutionMode = "MANUAL_EXECUTION"
	ExecAssisted        ExecutionMode = "ASSISTED_EXECUTION"
	ExecAuto            ExecutionMode = "AUTO_EXECUTION"
	ExecPaper           ExecutionMode = "PAPER"
	ExecShadow          ExecutionMode = "SHADOW"
	ExecEmergencyStop   ExecutionMode = "EMERGENCY_STOP"
)

// EntitlementLease represents a signed short-lived entitlement (SOW Section 41).
type EntitlementLease struct {
	Subject        string        `json:"subject"`
	LicenseID      string        `json:"license_id"`
	DeviceID       string        `json:"device_id"`
	Plan           string        `json:"plan"`
	Features       []string      `json:"features"`
	ExecutionModes []ExecutionMode `json:"execution_modes"`
	Strategies     []StrategyID  `json:"strategies"`
	IssuedAt       time.Time     `json:"issued_at"`
	ExpiresAt      time.Time     `json:"expires_at"`
	TokenID        string        `json:"token_id"`
	Issuer         string        `json:"issuer"`
	Audience       string        `json:"audience"`
}

// BrokerProfile represents broker-specific XAUUSD execution economics (SOW Section 103A).
type BrokerProfile struct {
	Broker              string          `json:"broker"`
	Server              string          `json:"server"`
	Platform            string          `json:"platform"`
	CanonicalSymbol     string          `json:"canonical_symbol"`
	BrokerSymbol        string          `json:"broker_symbol"`
	Digits              int             `json:"digits"`
	Point               decimal.Decimal `json:"point"`
	TickSize            decimal.Decimal `json:"tick_size"`
	TickValue           decimal.Decimal `json:"tick_value"`
	TickValueCurrency   string          `json:"tick_value_currency"`
	ContractSize        decimal.Decimal `json:"contract_size"`
	MinimumLot          decimal.Decimal `json:"minimum_lot"`
	MaximumLot          decimal.Decimal `json:"maximum_lot"`
	LotStep             decimal.Decimal `json:"lot_step"`
	StopsLevel          int             `json:"stops_level"`
	FreezeLevel         int             `json:"freeze_level"`
	FillModes           []string        `json:"fill_modes"`
	SwapLong            decimal.Decimal `json:"swap_long"`
	SwapShort           decimal.Decimal `json:"swap_short"`
	SwapMethod          string          `json:"swap_calculation_method"`
	TripleSwapDay       string          `json:"triple_swap_day"`
	Commission          decimal.Decimal `json:"commission_round_turn"`
	TypicalSpread       decimal.Decimal `json:"typical_spread"`
	SpreadP95           decimal.Decimal `json:"spread_p95"`
}

// KillSwitch represents an emergency control (SOW Section 85).
type KillSwitch struct {
	Scope   string `json:"scope"`   // GLOBAL, STRATEGY, SYMBOL, BROKER, ACCOUNT, LICENSE, DEVICE
	Target  string `json:"target"`  // specific ID or "ALL"
	Active  bool   `json:"active"`
	Reason  string `json:"reason"`
	SetBy   string `json:"set_by"`
	SetAt   time.Time `json:"set_at"`
}
