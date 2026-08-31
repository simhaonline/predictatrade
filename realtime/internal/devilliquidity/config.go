// Package devilliquidity implements the "Devil's Mark" / Devil Liquidity
// market-structure intelligence engine described in prompt.md.
//
// Lifecycle: Detection -> Qualification -> Tracking -> Approach -> Touch ->
// Sweep -> Reclaim/Reject -> Reversal Confirmation -> Outcome Resolution.
//
// The engine operates ONLY on completed (IsClosed) candles to satisfy the
// non-repaint rule (prompt.md Section 5). It is self-contained for core
// detection/qualification/lifecycle and accepts optional structural context
// (FVG/BOS/MSS/CHoCH) injected by the caller when available.
package devilliquidity

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// MarkState enumerates the Devil's Mark lifecycle (prompt.md Section 15).
type MarkState string

const (
	StateDetected          MarkState = "DETECTED"
	StateActive            MarkState = "ACTIVE"
	StateApproaching       MarkState = "APPROACHING"
	StateTouched           MarkState = "TOUCHED"
	StateSwept             MarkState = "SWEPT"
	StateReclaiming        MarkState = "RECLAIMING"
	StateRejected          MarkState = "REJECTED"
	StateReversalConfirmed MarkState = "REVERSAL_CONFIRMED"
	StateSignalEligible    MarkState = "SIGNAL_ELIGIBLE"
	StateMitigated         MarkState = "MITIGATED"
	StateInvalidated       MarkState = "INVALIDATED"
	StateExpired           MarkState = "EXPIRED"
	StateFailed            MarkState = "FAILED"
)

// MarkDirection is the expected reversal bias of a mark.
type MarkDirection string

const (
	DirBullish MarkDirection = "BULLISH" // support sweep -> up
	DirBearish MarkDirection = "BEARISH" // resistance sweep -> down
)

// Config holds all Devil Liquidity tunables (prompt.md Sections 9-11, 18-22, 41-42, 52).
type Config struct {
	Enabled bool

	// Flat-edge tolerance (Section 9).
	FlatWickRatio  float64 // max wick/range ratio allowed (e.g. 0.03)
	FlatWickATRTol float64 // wick tolerance as fraction of ATR
	MinimumTickTol int64   // minimum ticks considered "flat"
	TickSize       float64 // instrument tick size (XAUUSD default 0.01)

	// Expansion / displacement (Section 10).
	MinBodyRatio     float64 // e.g. 0.70
	MinRangeATR      float64 // e.g. 1.20
	MinBodyExpansion float64 // e.g. 1.50

	// Close-location (Section 11).
	CloseExtremeRatio float64 // e.g. 0.15

	// Lifecycle distances (Sections 18, 20-22).
	ApproachDistanceATR float64 // enter APPROACHING within N*ATR
	TouchToleranceATR   float64 // price reaching mark within N*ATR
	MinSweepDepthATR    float64 // min penetration below/above mark (ATR)
	MaxSweepDepthATR    float64 // beyond this => structural failure
	ReclaimMaxBars      int     // bars to reclaim before FAILED
	ReversalBodyRatio   float64 // body dominance for reversal confirmation

	// Expiry (Section 41).
	MarkExpiryBars int

	// Scoring floors (Sections 14, 26, 52).
	MinMarkQuality float64
	MinSignalScore float64

	// Volume contribution (Section 12).
	VolumeWeight float64

	// Operational.
	Mode string // disabled|shadow|confluence|soft_filter|hard_filter

	ConfigVersion string
}

// DefaultConfig returns the prompt.md initial defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:             true,
		FlatWickRatio:       0.03,
		FlatWickATRTol:      0.15,
		MinimumTickTol:      2,
		TickSize:            0.01,
		MinBodyRatio:        0.70,
		MinRangeATR:         1.20,
		MinBodyExpansion:    1.50,
		CloseExtremeRatio:   0.15,
		ApproachDistanceATR: 3.0,
		TouchToleranceATR:   0.5,
		MinSweepDepthATR:    0.20,
		MaxSweepDepthATR:    2.5,
		ReclaimMaxBars:      10,
		ReversalBodyRatio:   0.50,
		MarkExpiryBars:      240,
		MinMarkQuality:      40.0,
		MinSignalScore:      60.0,
		VolumeWeight:        10.0,
		Mode:                "confluence",
		ConfigVersion:       "1.0.0",
	}
}

// ScoreComponents records the individual weighted parts so later optimization
// never requires rebuilding history (prompt.md Section 14).
type ScoreComponents struct {
	FlatEdge     float64 `json:"flat_edge"`
	Displacement float64 `json:"displacement"`
	BodyDom      float64 `json:"body_dominance"`
	Volume       float64 `json:"volume"`
	Structure    float64 `json:"structure"`
	FVG          float64 `json:"fvg"`
	HTF          float64 `json:"htf_alignment"`
	Session      float64 `json:"session_quality"`
	Regime       float64 `json:"regime_compat"`

	Total float64 `json:"total"`
}

// DevilMark is the canonical in-memory + persisted mark record.
type DevilMark struct {
	ID        string
	Symbol    string
	Timeframe string
	Direction MarkDirection
	MarkPrice float64

	Open           float64
	High           float64
	Low            float64
	Close          float64
	Range          float64
	Body           float64
	BodyRatio      float64
	UpperWick      float64
	LowerWick      float64
	UpperWickRatio float64
	LowerWickRatio float64

	ATR           float64
	DetectedATR   float64
	RangeATRRatio float64
	BodyExpansion float64
	Volume        int64
	VolumeRatio   float64
	VolumeZScore  float64

	Spread   float64
	Digits   int
	TickSize float64

	// Structural context (optional, injected by caller).
	FVGPresent   bool
	FVGID        string
	BOSPresent   bool
	MSSPresent   bool
	CHoCHPresent bool

	FormationSession string
	FormationRegime  string

	MarkQuality   float64
	PriorityScore float64

	State MarkState

	FirstApproachAt     *time.Time
	FirstTouchAt        *time.Time
	FirstSweepAt        *time.Time
	SweepLow            float64
	SweepHigh           float64
	ReclaimAt           *time.Time
	ReversalConfirmedAt *time.Time
	SweepDepthATR       float64
	ReclaimStrength     float64

	ReversalScore float64
	CombinedScore float64

	DistanceATR float64

	ExpiredAt     *time.Time
	InvalidatedAt *time.Time
	ResolvedAt    *time.Time

	FeedSource    string
	Broker        string
	ServerID      string
	ConfigVersion string

	DetectedAt time.Time
	UpdatedAt  time.Time

	BarsSinceDetect int
}

// DevilEvent is an append-only lifecycle event.
type DevilEvent struct {
	MarkID        string
	Symbol        string
	Timeframe     string
	EventType     string
	StateFrom     MarkState
	StateTo       MarkState
	Price         float64
	MarkPrice     float64
	DistanceATR   float64
	ATR           float64
	Spread        float64
	Regime        string
	Session       string
	QualityScore  float64
	ReversalScore float64
	Metadata      map[string]interface{}
}

// MarshalJSON ensures events serialize cleanly.
func (e DevilEvent) MarshalJSON() ([]byte, error) {
	type alias DevilEvent
	return json.Marshal(alias(e))
}

func decF(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}
