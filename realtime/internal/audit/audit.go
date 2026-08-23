// Package audit provides execution logging for the signal pipeline.
// Logs pipeline executions, engine steps, score components, and final signal decisions
// to PostgreSQL/TimescaleDB audit tables using database/sql (matching existing persister).
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Logger writes audit execution records to PostgreSQL.
type Logger struct {
	db *sql.DB
}

// NewLogger creates a new audit logger from an existing *sql.DB connection.
func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// NewLoggerFromURL creates a new audit logger with its own connection pool.
func NewLoggerFromURL(dbURL string) (*Logger, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("audit logger connect: %w", err)
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("audit logger ping: %w", err)
	}
	return &Logger{db: db}, nil
}

// Close releases the database connection.
func (l *Logger) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}

// StartPipeline creates a pipeline execution record and returns its ID.
func (l *Logger) StartPipeline(ctx context.Context, asset, timeframe string) (uuid.UUID, error) {
	id := uuid.New()
	now := time.Now().UTC()
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO audit.pipeline_executions (
			event_time, pipeline_execution_id, asset, timeframe,
			started_at, status
		) VALUES ($1, $2, $3, $4, $5, 'RUNNING')
	`, now, id, asset, timeframe, now)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// CompletePipeline marks a pipeline execution as completed.
func (l *Logger) CompletePipeline(ctx context.Context, id uuid.UUID, signalID uuid.UUID, status string, metadata map[string]interface{}) error {
	now := time.Now().UTC()
	var metaJSON interface{}
	if len(metadata) > 0 {
		b, _ := json.Marshal(metadata)
		metaJSON = string(b)
	}
	_, err := l.db.ExecContext(ctx, `
		UPDATE audit.pipeline_executions
		SET completed_at = $1, status = $2, signal_id = $3,
		    latency_ms = EXTRACT(EPOCH FROM ($1 - started_at)) * 1000,
		    metadata = $4
		WHERE pipeline_execution_id = $5
	`, now, status, signalID, metaJSON, id)
	return err
}

// PipelineStep represents one engine/indicator execution within a pipeline.
type PipelineStep struct {
	ID                  uuid.UUID
	PipelineExecutionID uuid.UUID
	EngineName          string
	EngineVersion       string
	Timeframe           string
	StartedAt           time.Time
	Status              string
	RawValue            float64
	NormalizedValue     float64
	Direction           string
	Confidence          float64
	Weight              float64
	FeatureName         string
}

// LogStep records a single engine/indicator execution step.
func (l *Logger) LogStep(ctx context.Context, step PipelineStep) error {
	if step.ID == uuid.Nil {
		step.ID = uuid.New()
	}
	now := time.Now().UTC()
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO audit.pipeline_steps (
			event_time, step_id, pipeline_execution_id,
			engine_name, engine_version, timeframe,
			started_at, completed_at, latency_ms, status,
			raw_value, normalized_value, direction, confidence, weight
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, now, step.ID, step.PipelineExecutionID,
		step.EngineName, step.EngineVersion, step.Timeframe,
		step.StartedAt, now, int64(now.Sub(step.StartedAt).Milliseconds()), step.Status,
		step.RawValue, step.NormalizedValue, step.Direction, step.Confidence, step.Weight)
	return err
}

// ScoreComponent represents one pillar's contribution to a score.
type ScoreComponent struct {
	ID                   uuid.UUID
	ScoreExecutionID     uuid.UUID
	PillarName           string
	RawScore             float64
	Weight               float64
	WeightedContribution float64
	NormalizedScore      float64
	Confidence           float64
	Direction            string
	Status               string
	FeatureName          string
}

// ScoreExecution represents a score calculation.
type ScoreExecution struct {
	ID                  uuid.UUID
	PipelineExecutionID uuid.UUID
	ScoreVersion        string
	RawScore            float64
	NormalizedScore     float64
	BullishScore        float64
	BearishScore        float64
	Confidence          float64
	Signal              string
	SignalGrade         string
	StrategyID          string
	Asset               string
	Timeframe           string
}

// LogScore records a score execution with its pillar components.
func (l *Logger) LogScore(ctx context.Context, exec ScoreExecution, components []ScoreComponent) error {
	if exec.ID == uuid.Nil {
		exec.ID = uuid.New()
	}
	now := time.Now().UTC()
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO audit.score_executions (
			event_time, score_execution_id, pipeline_execution_id,
			score_version, raw_score, normalized_score,
			bullish_score, bearish_score, confidence,
			signal, signal_grade, strategy_id, asset, timeframe
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, now, exec.ID, exec.PipelineExecutionID,
		exec.ScoreVersion, exec.RawScore, exec.NormalizedScore,
		exec.BullishScore, exec.BearishScore, exec.Confidence,
		exec.Signal, exec.SignalGrade, exec.StrategyID, exec.Asset, exec.Timeframe)
	if err != nil {
		return err
	}

	for _, c := range components {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		_, err := l.db.ExecContext(ctx, `
			INSERT INTO audit.score_components (
				event_time, component_id, score_execution_id,
				pillar_name, raw_score, weight, weighted_contribution,
				normalized_score, confidence, direction, status, feature_name
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, now, c.ID, exec.ID,
			c.PillarName, c.RawScore, c.Weight, c.WeightedContribution,
			c.NormalizedScore, c.Confidence, c.Direction, c.Status, c.FeatureName)
		if err != nil {
			return err
		}
	}
	return nil
}

// SignalExecution represents the final signal decision.
type SignalExecution struct {
	SignalID            uuid.UUID
	PipelineExecutionID uuid.UUID
	ScoreExecutionID    uuid.UUID
	Asset               string
	Timeframe           string
	Signal              string
	Decision            string
	Score               float64
	Confidence          float64
	SignalGrade         string
	Entry               float64
	StopLoss            float64
	TakeProfit          float64
	RiskReward          float64
	DecisionReason      string
	StrategyID          string
	MarketDataTimestamp time.Time
	DataSource          string
	ApplicationVersion  string
}

// LogSignal records the final signal decision.
func (l *Logger) LogSignal(ctx context.Context, exec SignalExecution) error {
	if exec.SignalID == uuid.Nil {
		exec.SignalID = uuid.New()
	}
	now := time.Now().UTC()
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO audit.signal_executions (
			event_time, signal_id, pipeline_execution_id, score_execution_id,
			asset, timeframe, signal, decision, score, confidence, signal_grade,
			entry, stop_loss, take_profit, risk_reward,
			decision_reason, strategy_id, market_data_timestamp, data_source, application_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, now, exec.SignalID, exec.PipelineExecutionID, exec.ScoreExecutionID,
		exec.Asset, exec.Timeframe, exec.Signal, exec.Decision, exec.Score, exec.Confidence, exec.SignalGrade,
		exec.Entry, exec.StopLoss, exec.TakeProfit, exec.RiskReward,
		exec.DecisionReason, exec.StrategyID, exec.MarketDataTimestamp, exec.DataSource, exec.ApplicationVersion)
	return err
}
