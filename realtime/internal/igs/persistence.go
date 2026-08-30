package igs

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Persister writes IGS composites and AI research reports to PostgreSQL.
// Best-effort: failures are returned but never block signal generation.
type Persister struct {
	db *sql.DB
}

// NewPersister wraps an existing DB handle.
func NewPersister(db *sql.DB) *Persister { return &Persister{db: db} }

// NewPersisterFromURL opens a dedicated low-pool connection.
func NewPersisterFromURL(dbURL string) (*Persister, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return &Persister{db: db}, nil
}

// Close releases the connection pool.
func (p *Persister) Close() error {
	if p == nil || p.db == nil {
		return nil
	}
	return p.db.Close()
}

// SaveResult persists one IGS evaluation.
func (p *Persister) SaveResult(ctx context.Context, c *Composite) error {
	if p == nil || p.db == nil {
		return nil
	}
	componentJSON, _ := json.Marshal(c.Components)
	missingJSON, _ := json.Marshal(c.MissingComponents)

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.igs_results (
			event_time, id, signal_id, symbol, score, classification, direction,
			confidence, agreement, conflict, data_quality,
			components_available, components_total, missing_components, warnings,
			score_adjustment, mode, model_version, weights_version, component_snapshot, created_at
		) VALUES ($1,$2,$3,'XAUUSD',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
	`, c.Timestamp, uuid.New(), nil, c.Score, c.Classification, c.Direction,
		c.Confidence, c.Agreement, c.Conflict, c.DataQuality,
		c.ComponentsAvailable, c.ComponentsTotal, missingJSON, nil,
		c.ScoreAdjustment, c.Mode, c.ModelVersion, c.WeightsVersion, compJSONOrNil(componentJSON), c.Timestamp)
	return err
}

func compJSONOrNil(b []byte) interface{} {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return b
}

// SaveAIReport upserts a TradingAgents-generated daily research report.
// Research artifact only — never an execution authority.
func (p *Persister) SaveAIReport(ctx context.Context, rep *AIReport) error {
	if p == nil || p.db == nil {
		return nil
	}
	fullJSON, _ := json.Marshal(rep)
	keyDrivers, _ := json.Marshal(rep.KeyDrivers)
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.ai_research_reports (
			created_at, run_date, symbol, framework, framework_version, model,
			bias, confidence, summary, bull_thesis, bear_thesis, risks,
			key_drivers, full_report, provenance, quality
		) VALUES (NOW(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (run_date, symbol, framework) DO UPDATE SET
			model = EXCLUDED.model,
			bias = EXCLUDED.bias,
			confidence = EXCLUDED.confidence,
			summary = EXCLUDED.summary,
			bull_thesis = EXCLUDED.bull_thesis,
			bear_thesis = EXCLUDED.bear_thesis,
			risks = EXCLUDED.risks,
			key_drivers = EXCLUDED.key_drivers,
			full_report = EXCLUDED.full_report,
			provenance = EXCLUDED.provenance,
			quality = EXCLUDED.quality
	`, rep.RunDate, rep.Symbol, rep.Framework, rep.FrameworkVersion, rep.Model,
		rep.Bias, rep.Confidence, rep.Summary, rep.BullThesis, rep.BearThesis,
		rep.Risks, keyDrivers, fullJSON, json.RawMessage(rep.Provenance), rep.Quality)
	return err
}

// GetLatestAIReportDay returns the run date of the most recent report, if any.
func (p *Persister) GetLatestAIReportDay(ctx context.Context) (time.Time, bool, error) {
	if p == nil || p.db == nil {
		return time.Time{}, false, nil
	}
	var d time.Time
	err := p.db.QueryRowContext(ctx,
		`SELECT MAX(run_date) FROM trading.ai_research_reports WHERE symbol='XAUUSD'`).Scan(&d)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	if d.IsZero() {
		return time.Time{}, false, nil
	}
	return d, true, nil
}

// AIReport mirrors a tradingagents adapter output row.
type AIReport struct {
	RunDate          string   `json:"run_date"`
	Symbol           string   `json:"symbol"`
	Framework        string   `json:"framework"`
	FrameworkVersion string   `json:"framework_version"`
	Model            string   `json:"model"`
	Bias             string   `json:"bias"`
	Confidence       float64  `json:"confidence"`
	Summary          string   `json:"summary"`
	BullThesis       string   `json:"bull_thesis"`
	BearThesis       string   `json:"bear_thesis"`
	Risks            string   `json:"risks"`
	KeyDrivers       []string `json:"key_drivers"`
	Provenance       string   `json:"provenance"`
	Quality          string   `json:"quality"`
}
