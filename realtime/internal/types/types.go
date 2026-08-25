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
	DirectionWait    Direction = "WAIT"
	DirectionNoTrade Direction = "NO-TRADE"
	DirectionBlocked Direction = "BLOCKED"
	DirectionError   Direction = "ERROR"
)

// StrategyID identifies one of the four canonical strategy products.
type StrategyID string

const (
	StrategyStandardScalping StrategyID = "STANDARD_SCALPING"
	StrategyUltraScalping    StrategyID = "ULTRA_SCALPING"
	StrategyStandardSwing    StrategyID = "STANDARD_SWING"
	StrategyTrendSwing       StrategyID = "TREND_SWING"
	StrategyMarnieFib        StrategyID = "MARNIE_FIB"
)

// AllStrategies returns the four canonical strategy IDs.
func AllStrategies() []StrategyID {
	return []StrategyID{
		StrategyStandardScalping,
		StrategyUltraScalping,
		StrategyStandardSwing,
		StrategyTrendSwing,
		StrategyMarnieFib,
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

// Duration returns the bar period covered by one closed candle of this
// timeframe. Returns 0 for unknown timeframes (callers must treat the bar
// close time as unknown rather than guessing).
func (t Timeframe) Duration() time.Duration {
	switch t {
	case TFM1:
		return time.Minute
	case TFM5:
		return 5 * time.Minute
	case TFM15:
		return 15 * time.Minute
	case TFM30:
		return 30 * time.Minute
	case TFH1:
		return time.Hour
	case TFH4:
		return 4 * time.Hour
	case TFD1:
		return 24 * time.Hour
	case TFW1:
		return 7 * 24 * time.Hour
	case TFMN1:
		return 30 * 24 * time.Hour
	default:
		return 0
	}
}

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
	GradeWait    SignalGrade = "WAIT"
	GradeBlocked SignalGrade = "BLOCKED"
	GradeError   SignalGrade = "ERROR"
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
	GateStopHuntFilter    GateID = "stop_hunt_filter"
	GateMinATR            GateID = "min_atr"

	// Capital-protection gates (R1-R7, EV1-EV3, PT1-PT4)
	GateWrongSideSL      GateID = "wrong_side_sl"
	GateRiskOversize     GateID = "risk_oversize"
	GatePositionCaps     GateID = "position_caps"
	GateDailyLoss        GateID = "daily_loss"
	GateProfitTarget     GateID = "profit_target"
	GateMartingaleBan    GateID = "martingale_ban"
	GateEdgeValidation   GateID = "edge_validation"
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

	// Phase 2: Granular NO-TRADE reason codes (SOW Section 32)
	NTNoDirection           NoTradeReason = "NT_NO_DIRECTION"
	NTScoreBelowThreshold   NoTradeReason = "NT_SCORE_BELOW_THRESHOLD"
	NTRegimeMismatch        NoTradeReason = "NT_REGIME_MISMATCH"
	NTFeatureWarmup         NoTradeReason = "NT_FEATURE_WARMUP"
	NTLowATR                NoTradeReason = "LOW_ATR_COST_RISK"
	NTStructuralStopHunt    NoTradeReason = "STRUCTURAL_STOP_HUNT"
	NTRegimeMismatchNew     NoTradeReason = "REGIME_MISMATCH"
	NTMinLotExceedsEquity   NoTradeReason = "MIN_LOT_EXCEEDS_EQUITY"
	NTDailySoftCap          NoTradeReason = "DAILY_SOFT_CAP"
	NTMTFUnavailable        NoTradeReason = "NT_MTF_UNAVAILABLE"
	NTStructureUnavailable  NoTradeReason = "NT_STRUCTURE_UNAVAILABLE"
	NTATRNotReady           NoTradeReason = "NT_ATR_NOT_READY"
	NTDataStale             NoTradeReason = "NT_DATA_STALE"
	NTBrokerConstraint      NoTradeReason = "NT_BROKER_CONSTRAINT"
	NTCalibrationUnavailable NoTradeReason = "NT_CALIBRATION_UNAVAILABLE"
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

	// ML & Sentiment injection (v1.7.0) — default 0, does not affect existing tests
	ML       float64 `json:"ml,omitempty"`
	Sentiment float64 `json:"sentiment,omitempty"`
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

	// Probability is the calibrated win probability derived from a research-trained
	// calibration model loaded from CALIBRATION_DIR (env, default ./calibration).
	// It is set ONLY when a matching, schema-versioned calibration file exists;
	// otherwise it stays 0 and ProbabilityCalibrated stays false, so subscribers
	// NEVER see a fabricated probability (AGENTS.md / SOW Section 16).
	Probability          float64
	ProbabilityCalibrated bool
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

	// Phase 2: Versioning (SOW Section 33)
	RegimeEngineVersion  string
	StrategyVersion      string
	ScoringVersion       string
	GateConfigVersion    string

	// Phase 2: Shadow signal marker
	ShadowOnly           bool
	Executable           bool
	FailedProductionReason string

	// Phase 2: Detailed timestamp model (SOW Sections 26-30)
	// Each timestamp captures a distinct lifecycle stage.
	// Do NOT populate stages that have not occurred.
	MarketTime           time.Time // source/market candle time (broker time, UTC)
	MarketBarOpenTime    time.Time // candle open time
	MarketBarCloseTime   time.Time // candle close time
	DetectedAt           time.Time // strategy evaluation processing time
	CandidateDetectedAt  time.Time // candidate threshold crossed
	QualifiedAt          time.Time // trade threshold crossed + gates passed
	PublishedAt          time.Time // signal published to delivery layer
	DeliveryQueuedAt     time.Time
	DeliveredAt          time.Time
	AcknowledgedAt       time.Time
	ExecutionSubmittedAt time.Time
	BrokerFillAt         time.Time

	// Exit lifecycle (SOW Sections 2, 34) — NULL/zero until trade actually closes
	ExitPrice            decimal.Decimal
	ExitReason           string // TP1, TP2, TP3, SL, TIMEOUT, MANUAL, SAFETY_EXIT, BROKER_CLOSE
	ClosedAt             time.Time
	RealizedPnL          decimal.Decimal
	RealizedR            decimal.Decimal

	// Candidate/advisory classification (SOW Sections 12, 31-35)
	SignalClass          string // ADVISORY, EXECUTABLE
	CandidateThreshold   float64
	TradeThreshold       float64
	EntryType            string // MARKET, LIMIT, STOP
	ConflictPenalty      decimal.Decimal

	// Versioning for reproducibility (SOW Section 44)
	GeometryVersion      string
	RiskProfileVersion   string
	FeatureVersion       string
	RegimeVersion        string

	// Parent linkage for transition candidates (SOW Section 24)
	ParentCandidateID    string

	// Transition analysis scores (prompt.md Sections 6, 54)
	TransitionLongScore    decimal.Decimal
	TransitionShortScore   decimal.Decimal
	TransitionConflict      decimal.Decimal
	TransitionFinalScore    decimal.Decimal
	TransitionCandidateThreshold float64
	IsTransitionCandidate   bool

	// Blocker tracking (prompt.md Sections 17-18)
	PrimaryBlocker    string
	SecondaryBlockers []string

	// Signal provenance (prompt.md Sections 30-31, 43)
	SourceMode        string    // LIVE_MASTER_NODE, AGENT, SIMULATED, etc.
	SourceAgentID     string    // agent/device safe identifier
	SourceSequence    uint64    // source tick/snapshot sequence
	SourceTimestamp   time.Time // source timestamp from provider
	IngestTimestamp   time.Time // when data entered our pipeline
	BarClosed         BarClosedState // CLOSED_BAR_CONFIRMED or INTRABAR_LIVE
	BidPrice          decimal.Decimal
	AskPrice          decimal.Decimal
	ProvenanceState   ProvenanceState // LIVE_VERIFIED, UNVERIFIED, etc.
	CalibrationStatus CalibrationStatus

	// Deterministic hash (prompt.md Section 38)
	InputHash   string
	DecisionHash string

	// Capital-protection sizing annotations (R1/R7) — populated by the engine
	// from broker-account snapshot data; zero until account data is available.
	SuggestedLot     decimal.Decimal // recommended lot (floored to lot step)
	RiskDollars      decimal.Decimal // $ risk at requested lot
	RiskPctOfEquity  decimal.Decimal // requested-lot risk as % of equity
	SLDistancePoints decimal.Decimal // |entry - SL| in price points

	// Dominance (prompt.md Section 23)
	Dominance   float64

	// Traceability identifiers (prompt.md Sections 5-9)
	EvaluationSequence int64
	SignalSequence     int64
	SignalReference    string // PAT-XAU-YYYYMMDD-NNNNNN

	// Score status (prompt.md Section 15)
	ScoreStatus string // COMPUTED, NOT_EVALUATED, INSUFFICIENT_FEATURES, NOT_APPLICABLE, ERROR

	// Calibration metadata (prompt.md Section 18)
	CalibrationModelID     string
	CalibrationModelVersion string
	CalibrationTarget      string
	CalibrationSampleCount  int
	CalibrationArtifactHash string

	// Outbox state (prompt.md Section 34)
	OutboxState string // PENDING, PROCESSING, PUBLISHED, FAILED, RETRYING, DEAD_LETTER
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

// DataSourceType represents the provenance of market data (Stage 4 Section 1).
// Production signal generation requires LIVE_MASTER_NODE.
// Any other source must FAIL CLOSED.
type DataSourceType string

const (
	DataSourceLiveMasterNode DataSourceType = "LIVE_MASTER_NODE"
	DataSourceTest           DataSourceType = "TEST"
	DataSourceMock           DataSourceType = "MOCK"
	DataSourceDemo           DataSourceType = "DEMO"
	DataSourceFixture        DataSourceType = "FIXTURE"
	DataSourceSynthetic      DataSourceType = "SYNTHETIC"
	DataSourcePlaceholder    DataSourceType = "PLACEHOLDER"
	DataSourceUnknown        DataSourceType = "UNKNOWN"
	DataSourceReplay         DataSourceType = "REPLAY"
)

// IsLiveDataSource returns true only if the data source is production-live.
func IsLiveDataSource(src DataSourceType) bool {
	return src == DataSourceLiveMasterNode
}

// ModuleMode represents the activation state of an advanced intelligence module.
// Stage 4 Section 29-30: New modules start in SHADOW mode with zero score impact.
type ModuleMode string

const (
	ModuleOff       ModuleMode = "OFF"
	ModuleShadow    ModuleMode = "SHADOW"
	ModuleActive    ModuleMode = "ACTIVE"
	ModuleDisabled  ModuleMode = "DISABLED"
	ModuleUnsupported ModuleMode = "UNSUPPORTED"
	ModuleResearch  ModuleMode = "RESEARCH"
)

// ModuleProvenance records the metadata needed to reconstruct any advanced feature.
// Stage 4 Section 44: Complete module provenance.
type ModuleProvenance struct {
	Module         string          `json:"module"`
	ModuleVersion  string          `json:"module_version"`
	Timestamp      time.Time       `json:"timestamp"`
	SourceSnapshotID string       `json:"source_snapshot_id"`
	InputTimeframes []string      `json:"input_timeframes"`
	Lookback       int             `json:"lookback"`
	SampleCount    int             `json:"sample_count"`
	ParametersVersion string      `json:"parameters_version"`
	Value          interface{}     `json:"value"`
	State          string          `json:"state"`
	Availability   string          `json:"availability"`
	WindowStart    time.Time       `json:"window_start"`
	WindowEnd      time.Time       `json:"window_end"`
	CalcLatencyMs  int64           `json:"calculation_latency_ms"`
}

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

// Phase 2: Additional granular NO-TRADE reason codes (prompt.md Sections 6, 10, 23)
const (
	NTNoTrendTransition       NoTradeReason = "NT_NO_TREND_TRANSITION"
	NTConflictingDirection    NoTradeReason = "NT_CONFLICTING_DIRECTION"
	NTScoreBelowTradeThreshold NoTradeReason = "SCORE_BELOW_TRADE_THRESHOLD"
	NTScoreBelowCandidate     NoTradeReason = "SCORE_BELOW_CANDIDATE_THRESHOLD"
)

// ProvenanceState represents the authenticity verification state of a signal.
// prompt.md Section 43: Backend is authoritative for provenance.
type ProvenanceState string

const (
	ProvenanceLiveVerified  ProvenanceState = "LIVE_VERIFIED"
	ProvenanceRealReplay    ProvenanceState = "REAL_REPLAY"
	ProvenanceSynthetic     ProvenanceState = "SYNTHETIC"
	ProvenanceUnverified    ProvenanceState = "UNVERIFIED"
)

// CalibrationStatus represents the validation state of a calibration model.
// prompt.md Section 36: Until valid calibration exists, probability must be NULL.
type CalibrationStatus string

const (
	CalibrationUnverified  CalibrationStatus = "UNVERIFIED"
	CalibrationShadow      CalibrationStatus = "SHADOW"
	CalibrationValidated   CalibrationStatus = "VALIDATED"
	CalibrationPromoted    CalibrationStatus = "PROMOTED"
)

// IsCalibrationValidated returns true only if calibration is VALIDATED or PROMOTED.
func IsCalibrationValidated(status CalibrationStatus) bool {
	return status == CalibrationValidated || status == CalibrationPromoted
}

// BarClosedState distinguishes intrabar vs closed-bar semantics.
// prompt.md Section 32: Do NOT treat shift/index 0 as confirmed closed bar.
type BarClosedState string

const (
	BarClosedConfirmed  BarClosedState = "CLOSED_BAR_CONFIRMED"
	BarIntrabarLive     BarClosedState = "INTRABAR_LIVE"
)

// Additional Signal fields for transition analysis, provenance, and blocker tracking.
// These are appended after the main Signal struct definition via a separate struct
// that is embedded. However, since Go doesn't allow adding fields to an existing
// struct from a separate file, we define helper accessors instead.
// The actual fields are added directly to the Signal struct above.

// ScoreStatus values (prompt.md Section 15)
const (
	ScoreStatusComputed           = "COMPUTED"
	ScoreStatusNotEvaluated       = "NOT_EVALUATED"
	ScoreStatusInsufficientFeatures = "INSUFFICIENT_FEATURES"
	ScoreStatusNotApplicable      = "NOT_APPLICABLE"
	ScoreStatusError              = "ERROR"
)
