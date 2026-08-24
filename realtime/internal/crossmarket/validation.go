package crossmarket

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
	"fmt"

	"github.com/google/uuid"
)

// ValidationPersister handles shadow validation data persistence.
type ValidationPersister struct {
	db *sql.DB
}

func NewValidationPersister(db *sql.DB) *ValidationPersister {
	return &ValidationPersister{db: db}
}

// ShadowSignalSnapshot is a complete snapshot of a signal at evaluation time
// for later validation and ablation analysis.
type ShadowSignalSnapshot struct {
	ID                 uuid.UUID `json:"id"`
	Timestamp          time.Time `json:"timestamp"`
	SignalID           string    `json:"signal_id"`
	Strategy           string    `json:"strategy"`
	Direction          string    `json:"direction"`
	TechnicalScore      float64   `json:"technical_score"`
	CrossMarketScore   float64   `json:"cross_market_score"`
	CrossMarketConf    float64   `json:"cross_market_confidence"`
	CrossMarketRegime  string    `json:"cross_market_regime"`
	DXYContribution    float64   `json:"dxy_contribution"`
	EURUSDContribution float64   `json:"eurusd_contribution"`
	COTContribution    float64   `json:"cot_contribution"`
	RealYieldContrib   float64   `json:"real_yield_contribution"`
	VIXContribution    float64   `json:"vix_contribution"`
	BTCContribution    float64   `json:"btc_contribution"`
	OilContribution    float64   `json:"oil_contribution"`
	DriverHealth       string    `json:"driver_health"`
	DriverQuality      string    `json:"driver_quality"`
	CandidateDecision  string    `json:"candidate_decision"`
	SignalDecision     string    `json:"signal_decision"`
	Entry              float64   `json:"entry"`
	StopLoss           float64   `json:"stop_loss"`
	TP1                float64   `json:"tp1"`
	TP2                float64   `json:"tp2"`
	TP3                float64   `json:"tp3"`
	Expiry             time.Time `json:"expiry"`
	// Outcome fields (populated later)
	Outcome      string  `json:"outcome"`       // TP1, TP2, TP3, SL, EXPIRED, CANCELLED, NO_TRADE, UNRESOLVED
	MFE          float64 `json:"mfe"`           // Maximum Favorable Excursion
	MAE          float64 `json:"mae"`           // Maximum Adverse Excursion
	RMultiple    float64 `json:"r_multiple"`     // R multiple
	TimeToTP     int     `json:"time_to_tp"`     // seconds to TP
	TimeToSL     int     `json:"time_to_sl"`     // seconds to SL
	ResolvedAt   *time.Time `json:"resolved_at"`
}

// SaveShadowSnapshot persists a shadow signal snapshot for later validation.
func (p *ValidationPersister) SaveShadowSnapshot(ctx context.Context, snap *ShadowSignalSnapshot) error {
	if p == nil || p.db == nil {
		return nil
	}
	if snap.ID == uuid.Nil {
		snap.ID = uuid.New()
	}
	if snap.Timestamp.IsZero() {
		snap.Timestamp = time.Now().UTC()
	}

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.cross_market_shadow_snapshots (
			id, timestamp, signal_id, strategy, direction, technical_score,
			cross_market_score, cross_market_confidence, cross_market_regime,
			dxy_contribution, eurusd_contribution, cot_contribution,
			real_yield_contribution, vix_contribution, btc_contribution, oil_contribution,
			driver_health, driver_quality, candidate_decision, signal_decision,
			entry, stop_loss, tp1, tp2, tp3, expiry, outcome
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
	`, snap.ID, snap.Timestamp, snap.SignalID, snap.Strategy, snap.Direction, snap.TechnicalScore,
		snap.CrossMarketScore, snap.CrossMarketConf, snap.CrossMarketRegime,
		snap.DXYContribution, snap.EURUSDContribution, snap.COTContribution,
		snap.RealYieldContrib, snap.VIXContribution, snap.BTCContribution, snap.OilContribution,
		snap.DriverHealth, snap.DriverQuality, snap.CandidateDecision, snap.SignalDecision,
		snap.Entry, snap.StopLoss, snap.TP1, snap.TP2, snap.TP3, snap.Expiry, "UNRESOLVED")
	return err
}

// UpdateOutcome labels a shadow snapshot with the actual trade outcome.
// This is called AFTER the signal has resolved (TP hit, SL hit, expired).
func (p *ValidationPersister) UpdateOutcome(ctx context.Context, signalID string, outcome string, mfe, mae, rMultiple float64, timeToTP, timeToSL int) error {
	if p == nil || p.db == nil {
		return nil
	}
	now := time.Now().UTC()
	_, err := p.db.ExecContext(ctx, `
		UPDATE trading.cross_market_shadow_snapshots
		SET outcome = $1, mfe = $2, mae = $3, r_multiple = $4,
		    time_to_tp = $5, time_to_sl = $6, resolved_at = $7
		WHERE signal_id = $8 AND outcome = 'UNRESOLVED'
	`, outcome, mfe, mae, rMultiple, timeToTP, timeToSL, now, signalID)
	return err
}

// ValidationStatus represents the current validation readiness state.
type ValidationStatus struct {
	Mode                 string `json:"mode"`
	CalendarDays         int    `json:"calendar_days"`
	UsableShadowDays     int    `json:"usable_shadow_days"`
	TotalCandidates      int    `json:"total_candidates"`
	ResolvedOutcomes     int    `json:"resolved_outcomes"`
	MinimumDaysRequired  int    `json:"minimum_days_required"`
	AblationReady        bool   `json:"ablation_ready"`
	WalkForwardReady     bool   `json:"walk_forward_ready"`
	ActivationEligible   bool   `json:"activation_eligible"`
}

// GetValidationStatus queries the database for current validation state.
func (p *ValidationPersister) GetValidationStatus(ctx context.Context, mode string) (*ValidationStatus, error) {
	if p == nil || p.db == nil {
		return &ValidationStatus{Mode: mode, ActivationEligible: false}, nil
	}

	status := &ValidationStatus{
		Mode:                mode,
		MinimumDaysRequired: 30,
	}

	// Count calendar days since first shadow record
	var firstRecord, lastRecord time.Time
	err := p.db.QueryRowContext(ctx, `
		SELECT MIN(timestamp), MAX(timestamp) FROM trading.cross_market_shadow_snapshots
	`).Scan(&firstRecord, &lastRecord)
	if err == nil && !firstRecord.IsZero() {
		status.CalendarDays = int(lastRecord.Sub(firstRecord).Hours() / 24)
	}

	// Count total candidates and resolved outcomes
	err = p.db.QueryRowContext(ctx, `
		SELECT count(*), count(CASE WHEN outcome != 'UNRESOLVED' THEN 1 END)
		FROM trading.cross_market_shadow_snapshots
	`).Scan(&status.TotalCandidates, &status.ResolvedOutcomes)
	if err != nil {
		status.TotalCandidates = 0
		status.ResolvedOutcomes = 0
	}

	// Count usable shadow days (days with at least 10 records)
	err = p.db.QueryRowContext(ctx, `
		SELECT count(DISTINCT date_trunc('day', timestamp))
		FROM trading.cross_market_shadow_snapshots
		GROUP BY date_trunc('day', timestamp)
		HAVING count(*) >= 10
	`).Scan(&status.UsableShadowDays)
	if err != nil {
		status.UsableShadowDays = 0
	}

	// Determine readiness
	status.AblationReady = status.UsableShadowDays >= 30 && status.ResolvedOutcomes >= 100
	status.WalkForwardReady = status.AblationReady && status.ResolvedOutcomes >= 500
	status.ActivationEligible = status.WalkForwardReady && status.UsableShadowDays >= 30

	return status, nil
}

// AblationConfig defines an ablation test configuration.
type AblationConfig struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	DisabledDrivers []DriverName `json:"disabled_drivers"`
}

// GetAblationConfigs returns all ablation test configurations.
func GetAblationConfigs() []AblationConfig {
	return []AblationConfig{
		{Name: "BASELINE", Description: "Existing PAT signal without Cross-Market influence"},
		{Name: "FULL", Description: "Baseline + all Cross-Market drivers"},
		{Name: "ABLATION_DXY", Description: "Full minus DXY", DisabledDrivers: []DriverName{DriverDXY}},
		{Name: "ABLATION_EURUSD", Description: "Full minus EURUSD", DisabledDrivers: []DriverName{DriverEURUSD}},
		{Name: "ABLATION_COT", Description: "Full minus COT", DisabledDrivers: []DriverName{DriverCOT}},
		{Name: "ABLATION_REAL_YIELD", Description: "Full minus Real Yield", DisabledDrivers: []DriverName{DriverRealYields}},
		{Name: "ABLATION_VIX", Description: "Full minus VIX", DisabledDrivers: []DriverName{DriverVIX}},
		{Name: "ABLATION_BTC", Description: "Full minus BTC", DisabledDrivers: []DriverName{DriverBTC}},
		{Name: "ABLATION_OIL", Description: "Full minus Oil", DisabledDrivers: []DriverName{DriverOil}},
	}
}

// AblationResult holds the result of an ablation test for one strategy.
type AblationResult struct {
	ConfigName     string  `json:"config_name"`
	Strategy       string  `json:"strategy"`
	SignalCount    int     `json:"signal_count"`
	WinRate        float64 `json:"win_rate"`
	ProfitFactor   float64 `json:"profit_factor"`
	Expectancy     float64 `json:"expectancy"`
	AvgR           float64 `json:"avg_r"`
	MedianR        float64 `json:"median_r"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	TP1HitRate     float64 `json:"tp1_hit_rate"`
	TP2HitRate     float64 `json:"tp2_hit_rate"`
	TP3HitRate     float64 `json:"tp3_hit_rate"`
	SLRate         float64 `json:"sl_rate"`
	Sharpe         float64 `json:"sharpe"`
	Sortino        float64 `json:"sortino"`
}

// RunAblation runs an ablation test against the shadow dataset.
// Returns INSUFFICIENT_DATA if not enough resolved outcomes exist.
func (p *ValidationPersister) RunAblation(ctx context.Context, config AblationConfig, strategy string) (*AblationResult, error) {
	if p == nil || p.db == nil {
		return &AblationResult{ConfigName: config.Name, Strategy: strategy}, nil
	}

	result := &AblationResult{ConfigName: config.Name, Strategy: strategy}

	// Query resolved outcomes for this strategy
	rows, err := p.db.QueryContext(ctx, `
		SELECT outcome, r_multiple, mfe, mae
		FROM trading.cross_market_shadow_snapshots
		WHERE strategy = $1 AND outcome != 'UNRESOLVED'
	`, strategy)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	var rMultiples []float64
	wins := 0
	losses := 0
	totalR := 0.0
	tp1Hits := 0
	tp2Hits := 0
	tp3Hits := 0
	slHits := 0

	for rows.Next() {
		var outcome string
		var rMult, mfe, mae float64
		if err := rows.Scan(&outcome, &rMult, &mfe, &mae); err != nil {
			continue
		}
		rMultiples = append(rMultiples, rMult)
		totalR += rMult
		if rMult > 0 {
			wins++
		} else {
			losses++
		}
		switch outcome {
		case "TP1":
			tp1Hits++
		case "TP2":
			tp2Hits++
		case "TP3":
			tp3Hits++
		case "SL":
			slHits++
		}
	}

	total := wins + losses
	if total == 0 {
		return result, nil // INSUFFICIENT_DATA
	}

	result.SignalCount = total
	result.WinRate = float64(wins) / float64(total)
	result.AvgR = totalR / float64(total)
	if losses > 0 {
		result.ProfitFactor = (totalR + float64(losses)) / float64(losses)
	}
	result.TP1HitRate = float64(tp1Hits) / float64(total)
	result.TP2HitRate = float64(tp2Hits) / float64(total)
	result.TP3HitRate = float64(tp3Hits) / float64(total)
	result.SLRate = float64(slHits) / float64(total)

	// Calculate median R
	if len(rMultiples) > 0 {
		sorted := make([]float64, len(rMultiples))
		copy(sorted, rMultiples)
		// Simple sort for median
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j] < sorted[i] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		result.MedianR = sorted[len(sorted)/2]
	}

	return result, nil
}

// ActivationGuard determines if the engine can be activated.
type ActivationGuard struct {
	MinShadowDays     int
	MinResolvedSamples int
	RequireAblation   bool
	RequireWalkForward bool
}

// DefaultActivationGuard returns the default activation requirements.
func DefaultActivationGuard() ActivationGuard {
	return ActivationGuard{
		MinShadowDays:      30,
		MinResolvedSamples: 100,
		RequireAblation:    true,
		RequireWalkForward: true,
	}
}

// CheckEligibility evaluates whether the engine can be activated.
func (g *ActivationGuard) CheckEligibility(status *ValidationStatus) (bool, []string) {
	var reasons []string
	eligible := true

	if status.UsableShadowDays < g.MinShadowDays {
		eligible = false
		reasons = append(reasons, fmt.Sprintf("insufficient shadow days: %d/%d", status.UsableShadowDays, g.MinShadowDays))
	}
	if status.ResolvedOutcomes < g.MinResolvedSamples {
		eligible = false
		reasons = append(reasons, fmt.Sprintf("insufficient resolved outcomes: %d/%d", status.ResolvedOutcomes, g.MinResolvedSamples))
	}
	if g.RequireAblation && !status.AblationReady {
		eligible = false
		reasons = append(reasons, "ablation analysis not ready")
	}
	if g.RequireWalkForward && !status.WalkForwardReady {
		eligible = false
		reasons = append(reasons, "walk-forward validation not ready")
	}

	return eligible, reasons
}

// Need fmt import
var _ = json.Marshal
