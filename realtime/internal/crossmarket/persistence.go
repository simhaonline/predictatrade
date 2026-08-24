package crossmarket

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Persister writes cross-market results to PostgreSQL/TimescaleDB.
// All writes are best-effort — failures are logged but never block signal generation.
type Persister struct {
	db *sql.DB
}

func NewPersister(db *sql.DB) *Persister {
	return &Persister{db: db}
}

func NewPersisterFromURL(dbURL string) (*Persister, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	return &Persister{db: db}, nil
}

func (p *Persister) Close() error {
	if p == nil || p.db == nil {
		return nil
	}
	return p.db.Close()
}

// SaveConfluenceResult persists a confluence result with signal linkage.
func (p *Persister) SaveConfluenceResult(ctx context.Context, result *ConfluenceResult, signalID string) error {
	if p == nil || p.db == nil {
		return nil
	}

	id := uuid.New()
	now := time.Now().UTC()

	primaryJSON, _ := json.Marshal(result.PrimaryDrivers)
	opposingJSON, _ := json.Marshal(result.OpposingDrivers)
	missingJSON, _ := json.Marshal(result.MissingDrivers)
	warningsJSON, _ := json.Marshal(result.Warnings)
	driverJSON, _ := json.Marshal(result.DriverSnapshot)

	var sigID interface{}
	if signalID != "" {
		sigID = signalID
	}

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.cross_market_confluence_results (
			event_time, id, signal_id, symbol, score, direction, confidence,
			agreement, conflict, data_quality, regime, event_risk, correlation_regime,
			primary_drivers, opposing_drivers, missing_drivers, warnings,
			divergence_severity, score_adjustment, mode, model_version, weights_version,
			driver_snapshot, created_at
		) VALUES ($1, $2, $3, 'XAUUSD', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`, now, id, sigID, result.Score, result.Direction, result.Confidence,
		result.Agreement, result.Conflict, result.DataQuality, result.Regime, result.EventRisk, result.CorrelationRegime,
		primaryJSON, opposingJSON, missingJSON, warningsJSON,
		result.DivergenceSeverity, result.ScoreAdjustment, result.Mode, result.ModelVersion, result.WeightsVersion,
		driverJSON, now)
	return err
}

// SaveDriverSnapshot persists a single driver snapshot.
func (p *Persister) SaveDriverSnapshot(ctx context.Context, snap *DriverSnapshot) error {
	if p == nil || p.db == nil {
		return nil
	}

	id := uuid.New()
	now := time.Now().UTC()

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.cross_market_driver_snapshots (
			event_time, id, driver, raw_value, normalized_value, impact_score,
			direction, confidence, freshness, quality, source, timeframe, reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, now, id, snap.Name, snap.RawValue, snap.NormalizedValue, snap.ImpactScore,
		snap.Direction, snap.Confidence, snap.Freshness, snap.Quality,
		snap.Source, snap.Timeframe, snap.Reason, nil)
	return err
}

// SaveCorrelationRegime persists a correlation regime observation.
func (p *Persister) SaveCorrelationRegime(ctx context.Context, pair string, corr float64, window int, regime CorrelationRegime, stability float64) error {
	if p == nil || p.db == nil {
		return nil
	}

	id := uuid.New()
	now := time.Now().UTC()

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.cross_market_correlation_regimes (
			event_time, id, pair, correlation, window_size, regime, stability
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, now, id, pair, corr, window, regime, stability)
	return err
}

// SaveProviderHealth persists a provider health record.
func (p *Persister) SaveProviderHealth(ctx context.Context, provider string, status DataQuality, lastSuccess time.Time, lastError string, errorCount int, latencyMs int) error {
	if p == nil || p.db == nil {
		return nil
	}

	id := uuid.New()
	now := time.Now().UTC()

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.cross_market_provider_health (
			event_time, id, provider, status, last_success, last_error, error_count, latency_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, now, id, provider, status, lastSuccess, lastError, errorCount, latencyMs)
	return err
}
