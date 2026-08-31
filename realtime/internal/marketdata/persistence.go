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

	"github.com/predictatrade/realtime/internal/recovery"
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
			expected_cost, executable, failed_production_reason,
			ai_verification, risk_decision
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,
			$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,$60,
			$61,$62,$63,$64,$65,$66,$67,$68,$69,$70,$71,$72,$73,
			$74,$75,$76,$77,$78,$79,$80,$81,$82,$83,$84
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
		s.AiVerification, s.RiskDecision,
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

// RealignCandlesToBrokerOffset re-buckets historical UTC-aligned candles to the
// broker session timezone. A UTC-aligned bucket at time T maps to the broker
// bucket at T - off (because brokerLocal = T + off, then Truncate(period) yields
// a start that differs from the UTC start by exactly off). A uniform shift
// preserves bucket spacing, so no primary-key collisions occur. Idempotent:
// only UTC_ALIGNED rows are shifted, and they are marked BROKER_ALIGNED. Rows
// written by the live aggregator under a known offset are already BROKER_ALIGNED
// and are skipped. Recent (in-flight) candles are excluded to avoid clashing
// with candles the aggregator is actively writing.
func (p *Persister) RealignCandlesToBrokerOffset(off int) error {
	if off == 0 {
		return nil
	}
	db := p.GetDB()
	if db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := db.ExecContext(ctx, `
		UPDATE market.candles
		SET time = time - make_interval(hours => $1),
		    alignment_profile = 'BROKER_ALIGNED'
		WHERE alignment_profile = 'UTC_ALIGNED'
		  AND time < now() - interval '2 hours'
	`, off)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n > 0 {
		log.Printf("[RT] Realigned %d historical candles to broker offset %d (BROKER_ALIGNED)", n, off)
	}
	return nil
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
func (p *Persister) GetRecentSignals(ctx context.Context, limit int, strategy string) ([]*types.Signal, error) {
	query := `
		SELECT id, symbol, strategy_id, direction, grade, raw_score, long_score, short_score,
			calibrated_probability, entry_price, stop_loss, tp1, tp2, tp3,
			regime, session, news_risk, timeframe, status, created_at, expires_at,
			reason_codes, evidence_summary, gate_results,
			market_time, detected_at, signal_class, candidate_threshold, trade_threshold,
			entry_type, exit_price, exit_reason, closed_at, realized_pnl, realized_r,
			gross_rr_tp1, gross_rr_tp2, gross_rr_tp3,
			executable, failed_production_reason, ai_verification, risk_decision
		FROM trading.signals`
	args := []interface{}{limit}
	if strategy != "" {
		query += " WHERE strategy_id = $2"
		args = []interface{}{limit, strategy}
		query += " ORDER BY created_at DESC LIMIT $1"
	} else {
		query += " ORDER BY created_at DESC LIMIT $1"
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
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
		var aiVerificationStr, riskDecisionStr string
		var grossRR1Str, grossRR2Str, grossRR3Str string
		var reasonCodesJSON, evidenceJSON, gateJSON []byte
		err := rows.Scan(&s.ID, &s.Symbol, &strategyID, &direction, &grade,
			&rawScore, &longScore, &shortScore, &calProb, &entry, &sl, &tp1, &tp2, &tp3,
			&regime, &s.Session, &s.NewsRisk, &timeframe, &status, &s.CreatedAt, &s.ExpiresAt,
			&reasonCodesJSON, &evidenceJSON, &gateJSON,
			&s.MarketTime, &s.DetectedAt, &signalClass, &candidateThreshold, &tradeThreshold,
			&entryType, &exitPriceStr, &exitReason, &s.ClosedAt, &realizedPnLStr, &realizedRStr,
			&grossRR1Str, &grossRR2Str, &grossRR3Str,
			&s.Executable, &s.FailedProductionReason, &aiVerificationStr, &riskDecisionStr)
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
		s.AiVerification = aiVerificationStr
		s.RiskDecision = riskDecisionStr
		signals = append(signals, s)
	}
	return signals, nil
}

// TradeResult is the real executed-trade record surfaced to dashboards.
// It is sourced directly from trading.trade_results — no derived or
// estimated values are ever substituted.
type TradeResult struct {
	ID            string    `json:"id"`
	SignalID      string    `json:"signal_id,omitempty"`
	AccountID     string    `json:"account_id"`
	StrategyID    string    `json:"strategy_id"`
	Symbol        string    `json:"symbol"`
	Direction     string    `json:"direction"`
	BrokerTicket  string    `json:"broker_ticket,omitempty"`
	EntryPrice    string    `json:"entry_price"`
	ExitPrice     string    `json:"exit_price"`
	StopLoss      string    `json:"stop_loss,omitempty"`
	TakeProfit    string    `json:"take_profit,omitempty"`
	PnL           string    `json:"pnl"`
	PnLPoints     string    `json:"pnl_points"`
	PnLPercent    string    `json:"pnl_percent"`
	LotSize       string    `json:"lot_size,omitempty"`
	IsWin         bool      `json:"is_win"`
	IsLoss        bool      `json:"is_loss"`
	IsBreakeven   bool      `json:"is_breakeven"`
	CloseReason   string    `json:"close_reason,omitempty"`
	OpenedAt      time.Time `json:"opened_at,omitempty"`
	ClosedAt      time.Time `json:"closed_at"`
	TradingDay    string    `json:"trading_day,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	// Derived/enriched metrics (server-enriched when the EA omits them).
	TimeInTradeSeconds int64  `json:"time_in_trade_seconds"`
	MAE                string `json:"mae,omitempty"`
	MFE                string `json:"mfe,omitempty"`
	Timeframe          string `json:"timeframe,omitempty"`
}

// GetRecentTrades returns real executed trades from trading.trade_results.
// When strategy is non-empty it filters by strategy_id. Results are ordered
// by closed_at DESC so the newest closed trades appear first.
func (p *Persister) GetRecentTrades(ctx context.Context, limit int, strategy string) ([]*TradeResult, error) {
	query := `
		SELECT id, signal_id, account_id, strategy_id, symbol, direction,
			broker_ticket, entry_price, exit_price, stop_loss, take_profit,
			pnl, pnl_points, pnl_percent, lot_size,
			is_win, is_loss, is_breakeven, close_reason,
			opened_at, closed_at, trading_day, created_at,
			time_in_trade_seconds, mae, mfe, timeframe
		FROM trading.trade_results
	`
	args := []interface{}{}
	if strategy != "" {
		query += ` WHERE strategy_id = $1`
		args = append(args, strategy)
	}
	query += ` ORDER BY closed_at DESC LIMIT `
	if strategy != "" {
		query += `$2`
		args = append(args, limit)
	} else {
		query += `$1`
		args = append(args, limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*TradeResult
	for rows.Next() {
		t := &TradeResult{}
		var signalID, brokerTicket, sl, tp, lot, closeReason sql.NullString
		var openedAt, closedAt, createdAt sql.NullTime
		var tradingDay sql.NullTime
		var timeInTrade sql.NullInt64
		var maeN, fmeN, tfN sql.NullString
		err := rows.Scan(
			&t.ID, &signalID, &t.AccountID, &t.StrategyID, &t.Symbol, &t.Direction,
			&brokerTicket, &t.EntryPrice, &t.ExitPrice, &sl, &tp,
			&t.PnL, &t.PnLPoints, &t.PnLPercent, &lot,
			&t.IsWin, &t.IsLoss, &t.IsBreakeven, &closeReason,
			&openedAt, &closedAt, &tradingDay, &createdAt,
			&timeInTrade, &maeN, &fmeN, &tfN,
		)
		if err != nil {
			continue
		}
		if timeInTrade.Valid {
			t.TimeInTradeSeconds = timeInTrade.Int64
		}
		if maeN.Valid {
			t.MAE = maeN.String
		}
		if fmeN.Valid {
			t.MFE = fmeN.String
		}
		if tfN.Valid {
			t.Timeframe = tfN.String
		}
		if signalID.Valid {
			t.SignalID = signalID.String
		}
		if brokerTicket.Valid {
			t.BrokerTicket = brokerTicket.String
		}
		if sl.Valid {
			t.StopLoss = sl.String
		}
		if tp.Valid {
			t.TakeProfit = tp.String
		}
		if lot.Valid {
			t.LotSize = lot.String
		}
		if closeReason.Valid {
			t.CloseReason = closeReason.String
		}
		if openedAt.Valid {
			t.OpenedAt = openedAt.Time
		}
		if closedAt.Valid {
			t.ClosedAt = closedAt.Time
		}
		if tradingDay.Valid {
			t.TradingDay = tradingDay.Time.Format("2006-01-02")
		}
		if createdAt.Valid {
			t.CreatedAt = createdAt.Time
		}
		trades = append(trades, t)
	}
	return trades, nil
}

// GetUserAllowedStrategies returns the strategy IDs a user's active
// subscription entitles them to see. It mirrors the control plane's
// getEntitlements logic so signal visibility is server-authoritative and
// cannot be bypassed client-side. When no active subscription exists, the
// FREE default (STANDARD_SWING only) is returned.
func (p *Persister) GetUserAllowedStrategies(ctx context.Context, userID string) ([]string, error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT COALESCE(NULLIF(s.selected_strategies, '[]'::jsonb), p.allowed_strategies)::text
		FROM billing.subscriptions s
		JOIN control.plans p ON p.id = s.plan_id
		WHERE s.user_id = $1 AND s.status IN ('ACTIVE','TRIAL','GRACE','CANCEL_AT_PERIOD_END')
		ORDER BY s.created_at DESC LIMIT 1
	`, userID)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			// No active subscription -> FREE default (server-authoritative).
			return []string{"STANDARD_SCALPING"}, nil
		}
		return nil, err
	}
	var strategies []string
	if err := json.Unmarshal([]byte(raw), &strategies); err != nil {
		return nil, err
	}
	if len(strategies) == 0 {
		return []string{"STANDARD_SCALPING"}, nil
	}
	return strategies, nil
}

// GetUserSignalEntitlement returns the strategy IDs a user may view plus the
// plan's per-day signal cap (0 = unlimited). It mirrors the control plane's
// getEntitlements so signal visibility and quota are server-authoritative and
// cannot be bypassed client-side. No active subscription -> FREE default
// (STANDARD_SCALPING, cap 5/day per the MASTER PROMPT spec).
func (p *Persister) GetUserSignalEntitlement(ctx context.Context, userID string) (allowed []string, maxPerDay int, err error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT COALESCE(NULLIF(s.selected_strategies, '[]'::jsonb), p.allowed_strategies)::text,
		       COALESCE(p.max_signals_per_day, 0)::int
		FROM billing.subscriptions s
		JOIN control.plans p ON p.id = s.plan_id
		WHERE s.user_id = $1 AND s.status IN ('ACTIVE','TRIAL','GRACE','CANCEL_AT_PERIOD_END')
		ORDER BY s.created_at DESC LIMIT 1
	`, userID)
	var raw string
	var max int
	if err := row.Scan(&raw, &max); err != nil {
		if err == sql.ErrNoRows {
			// No active subscription -> FREE default (server-authoritative).
			return []string{"STANDARD_SCALPING"}, 5, nil
		}
		return nil, 0, err
	}
	var strategies []string
	if err := json.Unmarshal([]byte(raw), &strategies); err != nil {
		return nil, 0, err
	}
	if len(strategies) == 0 {
		strategies = []string{"STANDARD_SCALPING"}
	}
	return strategies, max, nil
}

// HealthCheck verifies database connectivity.
func (p *Persister) HealthCheck(ctx context.Context) error {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if p.db == nil {
		return fmt.Errorf("no database configured")
	}
	return p.db.PingContext(ctx2)
}

// DBHealth returns a simple, non-panicking health label for the underlying
// database so HTTP health/readiness endpoints can report DB status without
// crashing when the engine runs without a database attached.
//   - "not_configured" — no DB attached (fail-open: engine may run DB-less)
//   - "down"           — ping failed
//   - "ok"             — ping succeeded
func (p *Persister) DBHealth(ctx context.Context) string {
	if p == nil || p.db == nil {
		return "not_configured"
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := p.db.PingContext(ctx2); err != nil {
		return "down"
	}
	return "ok"
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
	case "LIVE_AGENT", "AGENT", "LIVE":
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

// ─── Recovery state persistence (so loss-recovery halts survive engine restarts) ───

func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

func mustDec(s sql.NullString) decimal.Decimal {
	if s.Valid && s.String != "" {
		if d, err := decimal.NewFromString(s.String); err == nil {
			return d
		}
	}
	return decimal.Zero
}

// SaveRecoveryState upserts a per-account+strategy recovery state machine record.
// The UNIQUE(account_id, strategy_id, symbol, trading_day) constraint makes this
// idempotent across restarts.
func (p *Persister) SaveRecoveryState(ctx context.Context, rec recovery.StateRecord) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO trading.recovery_states
			(account_id, strategy_id, symbol, state, consecutive_losses, daily_loss_count,
			 daily_loss_percent, daily_pnl, starting_equity, recovery_trades_taken, recovery_wins,
			 cooldown_until, halt_until, halt_reason, last_trade_at, last_loss_at, last_close_event_id, trading_day)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (account_id, strategy_id, symbol, trading_day)
		DO UPDATE SET
			state=EXCLUDED.state,
			consecutive_losses=EXCLUDED.consecutive_losses,
			daily_loss_count=EXCLUDED.daily_loss_count,
			daily_loss_percent=EXCLUDED.daily_loss_percent,
			daily_pnl=EXCLUDED.daily_pnl,
			starting_equity=EXCLUDED.starting_equity,
			recovery_trades_taken=EXCLUDED.recovery_trades_taken,
			recovery_wins=EXCLUDED.recovery_wins,
			cooldown_until=EXCLUDED.cooldown_until,
			halt_until=EXCLUDED.halt_until,
			halt_reason=EXCLUDED.halt_reason,
			last_trade_at=EXCLUDED.last_trade_at,
			last_loss_at=EXCLUDED.last_loss_at,
			last_close_event_id=EXCLUDED.last_close_event_id,
			updated_at=now()`,
		rec.Key.AccountID, rec.Key.StrategyID, rec.Key.Symbol, string(rec.State),
		rec.ConsecutiveLosses, rec.DailyLossCount, rec.DailyLossPercent, rec.DailyPnL.String(), rec.StartingEquity.String(),
		rec.RecoveryTradesTaken, rec.RecoveryWins,
		nullTime(rec.CooldownUntil), nullTime(rec.HaltUntil), rec.HaltReason,
		nullTime(rec.LastTradeAt), nullTime(rec.LastLossAt), rec.LastCloseEventID, rec.TradingDay,
	)
	return err
}

// LoadRecoveryStates reads all persisted recovery state records for restoration
// on engine startup.
func (p *Persister) LoadRecoveryStates(ctx context.Context) ([]recovery.StateRecord, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT account_id, strategy_id, symbol, state, consecutive_losses, daily_loss_count,
		       daily_loss_percent, daily_pnl, starting_equity, recovery_trades_taken, recovery_wins,
		       cooldown_until, halt_until, halt_reason, last_trade_at, last_loss_at, last_close_event_id, trading_day
		FROM trading.recovery_states`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []recovery.StateRecord
	for rows.Next() {
		var (
			accountID, strategyID, symbol, state, haltReason, lastCloseEventID sql.NullString
			dlp, dpnl, seq                                                          sql.NullString
			consLoss, dailyLossCount, recTrades, recWins                           int
			cooldown, halt, lastTrade, lastLoss, tradingDay                        sql.NullTime
		)
		if err := rows.Scan(
			&accountID, &strategyID, &symbol, &state, &consLoss, &dailyLossCount,
			&dlp, &dpnl, &seq, &recTrades, &recWins,
			&cooldown, &halt, &haltReason, &lastTrade, &lastLoss, &lastCloseEventID, &tradingDay,
		); err != nil {
			return nil, err
		}
		r := recovery.StateRecord{
			Key:                recovery.AccountStrategyKey{AccountID: accountID.String, StrategyID: strategyID.String, Symbol: symbol.String},
			State:              recovery.RecoveryState(state.String),
			ConsecutiveLosses:  consLoss,
			DailyLossCount:     dailyLossCount,
			RecoveryTradesTaken: recTrades,
			RecoveryWins:       recWins,
			CooldownUntil:      cooldown.Time,
			HaltUntil:          halt.Time,
			HaltReason:         haltReason.String,
			LastTradeAt:        lastTrade.Time,
			LastLossAt:         lastLoss.Time,
			LastCloseEventID:   lastCloseEventID.String,
			TradingDay:         tradingDay.Time,
		}
		if f, err := strconv.ParseFloat(dlp.String, 64); err == nil {
			r.DailyLossPercent = f
		}
		r.DailyPnL = mustDec(dpnl)
		r.StartingEquity = mustDec(seq)
		out = append(out, r)
	}
	return out, rows.Err()
}
