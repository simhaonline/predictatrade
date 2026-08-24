package marketdata

import (
	"strconv"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Persister saves ticks and candles to TimescaleDB/PostgreSQL.
type Persister struct {
	db *sql.DB
}

func NewPersister(dbURL string) (*Persister, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Persister{db: db}, nil
}

func (p *Persister) Close() { p.db.Close() }

// GetDB returns the underlying *sql.DB for direct queries (e.g. signal replay).
func (p *Persister) GetDB() *sql.DB { return p.db }

// SaveTick persists a tick to market.ticks table.
func (p *Persister) SaveTick(ctx context.Context, tick *types.Tick) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO market.ticks (symbol, time, bid, ask, mid, spread, tick_volume, source, quality, source_timestamp, gateway_receipt_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		ON CONFLICT (time, symbol, source) DO NOTHING
	`, tick.Symbol, tick.GatewayTimestamp, tick.Bid.String(), tick.Ask.String(),
		tick.Mid.String(), tick.Spread.String(), tick.TickVolume,
		tick.Source, string(tick.Quality), tick.SourceTimestamp)
	if err != nil {
		log.Printf("[RT] SaveTick error: %v", err)
	}
	return err
}

// SaveCandle persists a candle to market.candles table.
func (p *Persister) SaveCandle(ctx context.Context, c *types.Candle) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO market.candles (symbol, timeframe, time, open, high, low, close, volume, source, quality, is_closed, alignment_profile)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (time, symbol, timeframe, source) DO UPDATE SET
			open = EXCLUDED.open, high = EXCLUDED.high, low = EXCLUDED.low,
			close = EXCLUDED.close, volume = EXCLUDED.volume, quality = EXCLUDED.quality,
			is_closed = EXCLUDED.is_closed
	`, c.Symbol, string(c.Timeframe), c.Time,
		c.Open.String(), c.High.String(), c.Low.String(), c.Close.String(),
		c.Volume, c.Source, string(c.Quality), c.IsClosed, string(c.Alignment))
	if err != nil {
		log.Printf("[RT] SaveCandle error: %v", err)
	}
	return err
}

// SaveSignal persists a signal to trading.signals table.
func (p *Persister) SaveSignal(ctx context.Context, s *types.Signal) error {
	evidenceJSON, _ := json.Marshal(s.Evidence)
	gateJSON, _ := json.Marshal(s.GateResults)
	reasonsJSON, _ := json.Marshal(s.ReasonCodes)
	secondaryBlockersJSON, _ := json.Marshal(s.SecondaryBlockers)
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.signals (
			id, signal_id, symbol, strategy_id, direction, grade,
			raw_score, long_score, short_score, calibrated_probability,
			entry_price, stop_loss, tp1, tp2, tp3,
			regime, session, news_risk, timeframe, status,
			reason_codes, evidence_summary, gate_results,
			created_at, expires_at, gate_policy_version,
			market_time, detected_at, candidate_detected_at, qualified_at, published_at,
			signal_class, candidate_threshold, trade_threshold, entry_type,
			exit_price, exit_reason, closed_at, realized_pnl, realized_r,
			geometry_version, conflict_penalty, parent_candidate_id,
			evaluation_sequence, signal_sequence, signal_reference,
			source_mode, source_sequence, source_timestamp, ingest_timestamp,
			market_bar_open_time, market_bar_close_time,
			bid_price, ask_price, bar_closed, provenance_state,
			calibration_status, score_status, dominance,
			transition_long_score, transition_short_score, transition_conflict, transition_final_score,
			is_transition_candidate, primary_blocker, secondary_blockers,
			input_hash, decision_hash, outbox_state,
			strategy_version, feature_version, risk_profile_version, regime_version,
			gross_rr_tp1, gross_rr_tp2, gross_rr_tp3,
			net_rr_tp1, net_rr_tp2, net_rr_tp3,
			expected_cost, executable, failed_production_reason
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,
			$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,$60,
			$61,$62,$63,$64,$65,$66,$67,$68,$69,$70,$71,$72,$73,
			$74,$75,$76,$77,$78,$79,$80,$81,$82
		)
		ON CONFLICT (id, created_at) DO UPDATE SET
			status = EXCLUDED.status,
			signal_class = EXCLUDED.signal_class,
			exit_price = EXCLUDED.exit_price,
			exit_reason = EXCLUDED.exit_reason,
			closed_at = EXCLUDED.closed_at,
			realized_pnl = EXCLUDED.realized_pnl,
			realized_r = EXCLUDED.realized_r,
			outbox_state = EXCLUDED.outbox_state
	`,
		s.ID, s.ID, s.Symbol, string(s.StrategyID), string(s.Direction), string(s.Grade),
		s.RawScore.String(), s.LongScore.String(), s.ShortScore.String(), s.CalibratedProbability.String(),
		s.EntryPrice.String(), s.StopLoss.String(), s.TP1.String(), s.TP2.String(), s.TP3.String(),
		string(s.Regime), s.Session, s.NewsRisk, string(s.Timeframe), string(s.Status),
		string(reasonsJSON), string(evidenceJSON), string(gateJSON),
		s.CreatedAt, s.ExpiresAt, s.GatePolicyVersion,
		s.MarketTime, s.DetectedAt, s.CandidateDetectedAt, s.QualifiedAt, s.PublishedAt,
		s.SignalClass, s.CandidateThreshold, s.TradeThreshold, s.EntryType,
		s.ExitPrice.String(), s.ExitReason, s.ClosedAt, s.RealizedPnL.String(), s.RealizedR.String(),
		s.GeometryVersion, s.ConflictPenalty.String(), s.ParentCandidateID,
		s.EvaluationSequence, s.SignalSequence, s.SignalReference,
		s.SourceMode, s.SourceSequence, s.SourceTimestamp, s.IngestTimestamp,
		s.MarketBarOpenTime, s.MarketBarCloseTime,
		s.BidPrice.String(), s.AskPrice.String(), string(s.BarClosed), string(s.ProvenanceState),
		string(s.CalibrationStatus), s.ScoreStatus, fmt.Sprintf("%.4f", s.Dominance),
		s.TransitionLongScore.String(), s.TransitionShortScore.String(), s.TransitionConflict.String(), s.TransitionFinalScore.String(),
		s.IsTransitionCandidate, s.PrimaryBlocker, string(secondaryBlockersJSON),
		s.InputHash, s.DecisionHash, s.OutboxState,
		s.StrategyVersion, s.FeatureVersion, s.RiskProfileVersion, s.RegimeVersion,
		s.GrossRRTP1.String(), s.GrossRRTP2.String(), s.GrossRRTP3.String(),
		s.NetRRTP1.String(), s.NetRRTP2.String(), s.NetRRTP3.String(),
		s.ExpectedCost.String(), s.Executable, s.FailedProductionReason,
	)
	if err != nil {
		// SOW Section 13: canonical idempotency — duplicate signal for same
		// (strategy_id, market_bar_close_time, direction, signal_class) is NOT an error.
		// The signal was already saved from a prior candle processing pass.
		// This is expected when the agent sends multiple snapshots for the same bar.
		if strings.Contains(err.Error(), "idx_signals_canonical_idempotency") {
			return nil // Already saved — not an error
		}
		log.Printf("[RT] SaveSignal error: %v", err)
	}
	return err
}

// GetRecentCandles retrieves recent candles from the database.
func (p *Persister) GetRecentCandles(ctx context.Context, symbol string, tf string, limit int) ([]*types.Candle, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT time, open, high, low, close, volume, source, quality, is_closed
		FROM market.candles
		WHERE symbol = $1 AND timeframe = $2
		ORDER BY time DESC LIMIT $3
	`, symbol, tf, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candles []*types.Candle
	for rows.Next() {
		var c types.Candle
		var openStr, highStr, lowStr, closeStr, source, qualityStr string
		var isClosed bool
		err := rows.Scan(&c.Time, &openStr, &highStr, &lowStr, &closeStr, &c.Volume, &source, &qualityStr, &isClosed)
		if err != nil {
			continue
		}
		c.Symbol = symbol
		c.Timeframe = types.Timeframe(tf)
		c.Open = parseDecimal(openStr)
		c.High = parseDecimal(highStr)
		c.Low = parseDecimal(lowStr)
		c.Close = parseDecimal(closeStr)
		c.Source = source
		c.Quality = types.CandleQuality(qualityStr)
		c.IsClosed = isClosed
		candles = append(candles, &c)
	}
	return candles, nil
}

// GetRecentSignals retrieves recent signals.
func (p *Persister) GetRecentSignals(ctx context.Context, limit int) ([]*types.Signal, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, symbol, strategy_id, direction, grade, raw_score, long_score, short_score,
			calibrated_probability, entry_price, stop_loss, tp1, tp2, tp3,
			regime, session, news_risk, timeframe, status, created_at, expires_at,
			reason_codes, evidence_summary, gate_results,
			market_time, detected_at, signal_class, candidate_threshold, trade_threshold,
			entry_type, exit_price, exit_reason, closed_at, realized_pnl, realized_r,
			gross_rr_tp1, gross_rr_tp2, gross_rr_tp3,
			executable, failed_production_reason
		FROM trading.signals ORDER BY created_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []*types.Signal
	for rows.Next() {
		s := &types.Signal{}
		var rawScore, longScore, shortScore, calProb, entry, sl, tp1, tp2, tp3 string
		var strategyID, direction, grade, regime, status, timeframe string
		var signalClass, entryType, exitReason string
		var candidateThreshold, tradeThreshold, exitPriceStr, realizedPnLStr, realizedRStr string
		var grossRR1Str, grossRR2Str, grossRR3Str string
		var reasonCodesJSON, evidenceJSON, gateJSON []byte
		err := rows.Scan(&s.ID, &s.Symbol, &strategyID, &direction, &grade,
			&rawScore, &longScore, &shortScore, &calProb, &entry, &sl, &tp1, &tp2, &tp3,
			&regime, &s.Session, &s.NewsRisk, &timeframe, &status, &s.CreatedAt, &s.ExpiresAt,
			&reasonCodesJSON, &evidenceJSON, &gateJSON,
			&s.MarketTime, &s.DetectedAt, &signalClass, &candidateThreshold, &tradeThreshold,
			&entryType, &exitPriceStr, &exitReason, &s.ClosedAt, &realizedPnLStr, &realizedRStr,
			&grossRR1Str, &grossRR2Str, &grossRR3Str,
			&s.Executable, &s.FailedProductionReason)
		if err != nil {
			continue
		}
		s.StrategyID = types.StrategyID(strategyID)
		s.Direction = types.Direction(direction)
		s.Grade = types.SignalGrade(grade)
		s.RawScore = parseDecimal(rawScore)
		s.LongScore = parseDecimal(longScore)
		s.ShortScore = parseDecimal(shortScore)
		s.CalibratedProbability = parseDecimal(calProb)
		s.EntryPrice = parseDecimal(entry)
		s.StopLoss = parseDecimal(sl)
		s.TP1 = parseDecimal(tp1)
		s.TP2 = parseDecimal(tp2)
		s.TP3 = parseDecimal(tp3)
		s.Regime = types.Regime(regime)
		s.Status = types.SignalStatus(status)
		s.Timeframe = types.Timeframe(timeframe)
		s.SignalClass = signalClass
		s.EntryType = entryType
		s.ExitReason = exitReason
		s.CandidateThreshold = parseFloatSafe(candidateThreshold)
		s.TradeThreshold = parseFloatSafe(tradeThreshold)
		s.ExitPrice = parseDecimal(exitPriceStr)
		s.RealizedPnL = parseDecimal(realizedPnLStr)
		s.RealizedR = parseDecimal(realizedRStr)
		s.GrossRRTP1 = parseDecimal(grossRR1Str)
		s.GrossRRTP2 = parseDecimal(grossRR2Str)
		s.GrossRRTP3 = parseDecimal(grossRR3Str)
		// Unmarshal JSON fields
		if len(reasonCodesJSON) > 0 {
			json.Unmarshal(reasonCodesJSON, &s.ReasonCodes)
		}
		if len(evidenceJSON) > 0 {
			json.Unmarshal(evidenceJSON, &s.Evidence)
		}
		if len(gateJSON) > 0 {
			json.Unmarshal(gateJSON, &s.GateResults)
		}
		signals = append(signals, s)
	}
	return signals, nil
}

// HealthCheck verifies database connectivity.
func (p *Persister) HealthCheck(ctx context.Context) error {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return p.db.PingContext(ctx2)
}

func parseFloatSafe(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseDecimal(s string) (d decimal.Decimal) {
	d, _ = decimal.NewFromString(s)
	return d
}

// decStr converts a decimal to a string, returning "0" for zero values
// to avoid PostgreSQL "invalid input syntax for type numeric: \"\"" errors.
func decStr(d decimal.Decimal) string {
	s := d.String()
	if s == "" {
		return "0"
	}
	return s
}

// decStrOrZero converts a string to a safe numeric string, returning "0" for empty.
func decStrOrZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

// SaveCandidate persists a signal candidate to trading.signal_candidates.
func (p *Persister) SaveCandidate(ctx context.Context, c *CandidateRecord) error {
	reasonJSON, _ := json.Marshal(c.ReasonCodes)
	structureJSON, _ := json.Marshal(c.StructureState)
	readinessJSON, _ := json.Marshal(c.FeatureReadiness)
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.signal_candidates (
			candidate_uuid, symbol, strategy_id, strategy_version, direction,
			entry_price, stop_loss, tp1, tp2, tp3, calculated_rr,
			raw_score, long_score, short_score, calibrated_prob,
			regime, market_session, timeframe, structure_state, feature_readiness,
			reason_codes, approval_state, rejection_gate, signal_id, created_at, expires_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26
		) ON CONFLICT (candidate_uuid) DO NOTHING
	`, c.CandidateUUID, c.Symbol, c.StrategyID, c.StrategyVersion, c.Direction,
		decStrOrZero(c.EntryPrice), decStrOrZero(c.StopLoss), decStrOrZero(c.TP1), decStrOrZero(c.TP2), decStrOrZero(c.TP3), decStrOrZero(c.CalculatedRR),
		decStrOrZero(c.RawScore), decStrOrZero(c.LongScore), decStrOrZero(c.ShortScore), decStrOrZero(c.CalibratedProb),
		c.Regime, c.MarketSession, c.Timeframe, string(structureJSON), string(readinessJSON),
		string(reasonJSON), c.ApprovalState, c.RejectionGate, sql.NullString{String: c.SignalID, Valid: c.SignalID != ""}, c.CreatedAt, c.ExpiresAt)
	if err != nil {
		log.Printf("[RT] SaveCandidate error: %v (dir=%s regime=%s session=%s approval=%s reject=%s)", err, c.Direction, c.Regime, c.MarketSession, c.ApprovalState, c.RejectionGate)
	}
	return err
}

// SaveRiskDecision persists a risk gate decision to trading.risk_decisions.
func (p *Persister) SaveRiskDecision(ctx context.Context, r *RiskDecisionRecord) error {
	reasonJSON, _ := json.Marshal(r.ReasonCodes)
	metaJSON, _ := json.Marshal(r.Metadata)
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.risk_decisions (
			signal_id, decision, gate_id, gate_result, reason_codes, metadata,
			threshold_value, observed_value, gate_version, config_version, evaluated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, sql.NullString{String: r.SignalID, Valid: r.SignalID != ""}, r.Decision, r.GateID, r.GateResult, string(reasonJSON), string(metaJSON),
		decStrOrZero(r.ThresholdValue), decStrOrZero(r.ObservedValue), r.GateVersion, r.ConfigVersion, r.EvaluatedAt)
	if err != nil {
		log.Printf("[RT] SaveRiskDecision error: %v", err)
	}
	return err
}

// SaveStrategyEvaluation persists a strategy evaluation to trading.strategy_evaluations.
func (p *Persister) SaveStrategyEvaluation(ctx context.Context, e *StrategyEvalRecord) error {
	condPassedJSON, _ := json.Marshal(e.ConditionsPassed)
	condFailedJSON, _ := json.Marshal(e.ConditionsFailed)
	inputJSON, _ := json.Marshal(e.InputFeatures)
	// Truncate reason to prevent DB overflow
	reason := e.Reason
	if len(reason) > 1900 {
		reason = reason[:1900]
	}
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.strategy_evaluations (
			strategy_id, strategy_version, symbol, timeframe, timestamp,
			input_features, score, long_score, short_score,
			conditions_passed, conditions_failed, candidate_generated, direction, reason,
			evaluation_sequence, score_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`, e.StrategyID, e.StrategyVersion, e.Symbol, e.Timeframe, e.Timestamp,
		string(inputJSON), decStrOrZero(e.Score), decStrOrZero(e.LongScore), decStrOrZero(e.ShortScore),
		string(condPassedJSON), string(condFailedJSON), e.CandidateGenerated, e.Direction, reason,
		e.EvaluationSequence, e.ScoreStatus)
	if err != nil {
		log.Printf("[RT] SaveStrategyEvaluation error: %v", err)
	}
	return err
}

// SaveCooldownAudit persists a cooldown event to trading.cooldown_audit.
func (p *Persister) SaveCooldownAudit(ctx context.Context, c *CooldownAuditRecord) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.cooldown_audit (
			symbol, strategy_id, event_type, event_timestamp,
			cooldown_start, cooldown_expiry, remaining_seconds, fingerprint
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, c.Symbol, c.StrategyID, c.EventType, c.EventTimestamp,
		c.CooldownStart, c.CooldownExpiry, c.RemainingSeconds, c.Fingerprint)
	if err != nil {
		log.Printf("[RT] SaveCooldownAudit error: %v", err)
	}
	return err
}

// SaveDuplicateAudit persists a duplicate signal event to trading.duplicate_audit.
func (p *Persister) SaveDuplicateAudit(ctx context.Context, d *DuplicateAuditRecord) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.duplicate_audit (
			fingerprint, symbol, strategy_id, direction, event_type, event_timestamp, candidate_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, d.Fingerprint, d.Symbol, d.StrategyID, d.Direction, d.EventType, d.EventTimestamp,
		sql.NullString{String: d.CandidateID, Valid: d.CandidateID != ""})
	if err != nil {
		log.Printf("[RT] SaveDuplicateAudit error: %v", err)
	}
	return err
}

// SaveIndicatorHistory persists an indicator value to trading.indicator_history.
func (p *Persister) SaveIndicatorHistory(ctx context.Context, h *IndicatorHistoryRecord) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.indicator_history (
			symbol, timeframe, timestamp, indicator_name, indicator_version,
			value, value_secondary, value_tertiary, quality, source
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, h.Symbol, h.Timeframe, h.Timestamp, h.IndicatorName, h.IndicatorVersion,
		decStrOrZero(h.Value), decStrOrZero(h.ValueSecondary), decStrOrZero(h.ValueTertiary), h.Quality, h.Source)
	if err != nil {
		log.Printf("[RT] SaveIndicatorHistory error: %v", err)
	}
	return err
}

// SaveRegimeHistory persists a regime classification to trading.regime_history.
func (p *Persister) SaveRegimeHistory(ctx context.Context, r *RegimeHistoryRecord) error {
	featuresJSON, _ := json.Marshal(r.ContributingFeatures)
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.regime_history (
			symbol, timeframe, timestamp, regime, confidence, contributing_features, algorithm_version
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, r.Symbol, r.Timeframe, r.Timestamp, r.Regime, decStrOrZero(r.Confidence), string(featuresJSON), r.AlgorithmVersion)
	if err != nil {
		log.Printf("[RT] SaveRegimeHistory error: %v", err)
	}
	return err
}

// === Record Types ===

type CandidateRecord struct {
	CandidateUUID    string
	Symbol           string
	StrategyID      string
	StrategyVersion string
	Direction       string
	EntryPrice      string
	StopLoss        string
	TP1             string
	TP2             string
	TP3             string
	CalculatedRR    string
	RawScore        string
	LongScore       string
	ShortScore      string
	CalibratedProb  string
	Regime          string
	MarketSession   string
	Timeframe       string
	StructureState  interface{}
	FeatureReadiness interface{}
	ReasonCodes     interface{}
	ApprovalState   string
	RejectionGate   string
	SignalID        string
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

type RiskDecisionRecord struct {
	SignalID       string
	Decision       string
	GateID         string
	GateResult     string
	ReasonCodes    interface{}
	Metadata       interface{}
	ThresholdValue string
	ObservedValue  string
	GateVersion    string
	ConfigVersion  string
	EvaluatedAt    time.Time
}

type StrategyEvalRecord struct {
	StrategyID      string
	StrategyVersion string
	Symbol          string
	Timeframe       string
	Timestamp       time.Time
	InputFeatures   interface{}
	Score           string
	LongScore       string
	ShortScore      string
	ConditionsPassed interface{}
	ConditionsFailed interface{}
	CandidateGenerated bool
	Direction      string
	Reason          string
	EvaluationSequence int64
	ScoreStatus     string
}

type CooldownAuditRecord struct {
	Symbol          string
	StrategyID     string
	EventType      string
	EventTimestamp time.Time
	CooldownStart  time.Time
	CooldownExpiry time.Time
	RemainingSeconds int
	Fingerprint    string
}

type DuplicateAuditRecord struct {
	Fingerprint    string
	Symbol         string
	StrategyID    string
	Direction      string
	EventType      string
	EventTimestamp time.Time
	CandidateID    string
}

type IndicatorHistoryRecord struct {
	Symbol          string
	Timeframe       string
	Timestamp       time.Time
	IndicatorName   string
	IndicatorVersion string
	Value          string
	ValueSecondary  string
	ValueTertiary   string
	Quality        string
	Source         string
}

type RegimeHistoryRecord struct {
	Symbol             string
	Timeframe          string
	Timestamp          time.Time
	Regime             string
	Confidence         string
	ContributingFeatures interface{}
	AlgorithmVersion   string
}

// GenerateSignalReference creates a human-readable signal reference.
// Format: PAT-XAU-YYYYMMDD-NNNNNN (prompt.md Section 6)
func GenerateSignalReference(signalSeq int64) string {
	return fmt.Sprintf("PAT-XAU-%s-%06d", time.Now().UTC().Format("20060102"), signalSeq)
}

// SaveOutboxEvent persists a signal event to the transactional outbox.
func (p *Persister) SaveOutboxEvent(ctx context.Context, signalID string, signalReference string, payload interface{}) error {
	payloadJSON, _ := json.Marshal(payload)
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.signal_outbox (signal_id, signal_reference, payload, state)
		VALUES ($1, $2, $3, 'PENDING')
	`, signalID, signalReference, string(payloadJSON))
	if err != nil {
		log.Printf("[RT] SaveOutboxEvent error: %v", err)
	}
	return err
}

// GetPendingOutboxEvents retrieves pending outbox events for dispatch.
func (p *Persister) GetPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, signal_id, signal_reference, payload, state, attempt_count, last_error
		FROM trading.signal_outbox
		WHERE state IN ('PENDING', 'RETRYING') AND next_retry_at <= NOW()
		ORDER BY created_at ASC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		var payloadJSON []byte
		err := rows.Scan(&e.ID, &e.SignalID, &e.SignalReference, &payloadJSON, &e.State, &e.AttemptCount, &e.LastError)
		if err != nil {
			continue
		}
		e.Payload = string(payloadJSON)
		events = append(events, e)
	}
	return events, nil
}

// MarkOutboxPublished marks an outbox event as published.
func (p *Persister) MarkOutboxPublished(ctx context.Context, eventID string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE trading.signal_outbox SET state = 'PUBLISHED', published_at = NOW() WHERE id = $1
	`, eventID)
	return err
}

// MarkOutboxFailed marks an outbox event as failed and schedules retry.
func (p *Persister) MarkOutboxFailed(ctx context.Context, eventID string, errorMsg string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE trading.signal_outbox 
		SET state = CASE WHEN attempt_count >= max_attempts - 1 THEN 'DEAD_LETTER' ELSE 'RETRYING' END,
		    attempt_count = attempt_count + 1,
		    last_attempt_at = NOW(),
		    last_error = $2,
		    next_retry_at = NOW() + INTERVAL '5 seconds'
		WHERE id = $1
	`, eventID, errorMsg)
	return err
}

// NextEvaluationSequence gets the next evaluation sequence number.
func (p *Persister) NextEvaluationSequence(ctx context.Context) (int64, error) {
	var seq int64
	err := p.db.QueryRowContext(ctx, "SELECT nextval('trading.evaluation_seq')").Scan(&seq)
	return seq, err
}

// NextSignalSequence gets the next signal sequence number.
func (p *Persister) NextSignalSequence(ctx context.Context) (int64, error) {
	var seq int64
	err := p.db.QueryRowContext(ctx, "SELECT nextval('trading.signal_seq')").Scan(&seq)
	return seq, err
}

// OutboxEvent represents a pending outbox event for dispatch.
type OutboxEvent struct {
	ID              string
	SignalID        string
	SignalReference string
	Payload         string
	State           string
	AttemptCount    int
	LastError       string
}

// Production safeguard: prevent synthetic/test data from being persisted as LIVE (prompt.md Section 3)
func IsProductionSafeSource(sourceMode string) bool {
	switch sourceMode {
	case "LIVE_MASTER_NODE", "AGENT", "LIVE":
		return true
	default:
		return false
	}
}

// SaveSignalSafe persists a signal with production safeguards (prompt.md Section 3)
func (p *Persister) SaveSignalSafe(ctx context.Context, s *types.Signal) error {
	// Production safeguard: reject synthetic/test signals from being persisted as production
	if !IsProductionSafeSource(s.SourceMode) && s.SourceMode != "" {
		// Allow non-production signals but mark them as UNVERIFIED
		if s.ProvenanceState != types.ProvenanceSynthetic {
			s.ProvenanceState = types.ProvenanceUnverified
		}
	}
	return p.SaveSignal(ctx, s)
}
