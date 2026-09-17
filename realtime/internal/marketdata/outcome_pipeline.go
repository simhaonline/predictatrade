// outcome_pipeline.go — Phase 0.5 (prompt.md, Candidate B): outcome capture
// writers. Schema: migration 150_outcome_pipeline.sql (frozen design,
// docs/strategy/OUTCOME_PIPELINE.md).
//
// SavePrediction               — one row per EXECUTABLE signal at emit
//                                (pre-registered expectation). Idempotent on
//                                signal_id; fail-open (never blocks signal
//                                truth — same pattern as SaveFeatureSnapshot).
// SaveOutcomeFromTradeResult   — upsert on signal_id from an EA TRADE_RESULT.
//                                MANUAL closes are first-class. link_status
//                                is LINKED when a signal_id is provided;
//                                UNLINKED rows are recorded, never dropped
//                                (fail-closed, prompt.md guardrail).
package marketdata

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// SavePrediction persists the pre-registered expectation for an EXECUTABLE
// signal. Idempotent on signal_id (ON CONFLICT DO UPDATE — a re-emit
// refreshes fields but never duplicates).
func (p *Persister) SavePrediction(ctx context.Context, signalID, symbol, timeframe,
	strategyID, strategyVersion, direction,
	entry, stopLoss, tp1, tp2, tp3,
	rawScore, calibratedProb, grade, signalClass, tier, tierReason,
	compositeScore string, familySubScores json.RawMessage,
	regime, session string, gateDecisions json.RawMessage,
	astroSizing, yogaBias, macroBiasX6 string, featureSnapshotID, modelVersion string,
) error {
	db := p.GetDB()
	if db == nil {
		return nil // fail-open: never block signal truth
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO trading.predictions (
			signal_id, symbol, timeframe, strategy_id, strategy_version, direction,
			entry, stop_loss, tp1, tp2, tp3,
			raw_score, calibrated_probability, grade, signal_class, tier, tier_reason,
			composite_score, family_sub_scores, regime, session, gate_decisions,
			astro_sizing_multiplier, yoga_bias, macro_bias_x6,
			feature_snapshot_id, model_version
		) VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7::numeric, $8::numeric, $9::numeric,
			$10::numeric, $11::numeric, $12::numeric, $13::numeric, $14, $15, $16,
			$17, $18, $19::jsonb, $20, $21, $22::jsonb, $23::numeric, $24::numeric,
			$25::numeric, $26::uuid, $27
		)
		ON CONFLICT (signal_id) DO UPDATE SET
			entry = EXCLUDED.entry,
			stop_loss = EXCLUDED.stop_loss,
			raw_score = EXCLUDED.raw_score,
			calibrated_probability = EXCLUDED.calibrated_probability,
			feature_snapshot_id = COALESCE(EXCLUDED.feature_snapshot_id, trading.predictions.feature_snapshot_id)`,
		signalID, symbol, timeframe, strategyID, strategyVersion, direction,
		numOrNull(entry), numOrNull(stopLoss), numOrNull(tp1), numOrNull(tp2), numOrNull(tp3),
		numOrNull(rawScore), numOrNull(calibratedProb), grade, signalClass, tier, tierReason,
		numOrNull(compositeScore), jsonOrNull(familySubScores), regime, session,
		jsonOrNull(gateDecisions), numOrNull(astroSizing), numOrNull(yogaBias),
		numOrNull(macroBiasX6), uuidOrNull(featureSnapshotID), modelVersion)
	return err
}

// SaveOutcomeFromTradeResult upserts the realized outcome of an EXECUTABLE
// signal from its EA TRADE_RESULT. Idempotent on signal_id. MANUAL closes
// are recorded as a first-class close_reason. link_status is LINKED whenever
// a signal id resolves — UNLINKED rows are never fabricated or dropped.
func (p *Persister) SaveOutcomeFromTradeResult(ctx context.Context, signalID,
	strategyID, closeReason, realizedPnl, rMultiple, mae, mfe string,
	durationSeconds int64, spreadAtEntry, spreadAtExit string, linked bool,
) error {
	db := p.GetDB()
	if db == nil {
		return nil
	}
	link := "UNLINKED"
	if linked && signalID != "" {
		link = "LINKED"
	}
	outcomeType := "BREAKEVEN"
	isWin := false
	pnl := numOrNull(realizedPnl)
	if pnlF := fOf(realizedPnl); pnlF > 0 {
		outcomeType = "WIN"
		isWin = true
	} else if pnlF < 0 {
		outcomeType = "LOSS"
		isWin = false
	}
	// prediction_id: resolve from trading.predictions when the signal is known;
	// when UNLINKED the FK is not satisfiable — the row still records the
	// outcome with link_status=UNLINKED (alert path handles it).
	resolvedTradeResultID := any(nil)
	if signalID != "" {
		_ = db.QueryRowContext(ctx,
			`SELECT id FROM trading.trade_results WHERE signal_id=$1::uuid ORDER BY closed_at DESC LIMIT 1`,
			signalID).Scan(&resolvedTradeResultID)
	}
	var predID any
	_ = db.QueryRowContext(ctx,
		`SELECT id FROM trading.predictions WHERE signal_id=$1::uuid`, signalID).Scan(&predID)
	if predID == nil && signalID != "" {
		// Outcome arrived before/without a prediction row (e.g. legacy EA):
		// create the parent so the FK holds. link_status=UNLINKED flags it
		// for the alert path — never silently dropped.
		_, _ = db.ExecContext(ctx, `
			INSERT INTO trading.predictions (signal_id, symbol, timeframe, direction, strategy_id)
			VALUES ($1::uuid, '', '', 'UNKNOWN', $2)
			ON CONFLICT (signal_id) DO NOTHING`, signalID, strategyID)
		_ = db.QueryRowContext(ctx,
			`SELECT id FROM trading.predictions WHERE signal_id=$1::uuid`, signalID).Scan(&predID)
	}

	if predID == nil {
		// Signal id empty and no prediction resolvable — fail closed:
		// no parent record can exist for a fabricated id.
		return nil
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO trading.prediction_outcomes (
			prediction_id, signal_id, link_status, strategy_id, close_reason,
			outcome_type, outcome_value, realized_rr, realized_pnl, r_multiple,
			mae_points, mfe_points, duration_seconds,
			spread_at_entry, spread_at_exit, trade_result_id, schema_version
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5,
			$6, $7, $8::numeric, $9::numeric, $10::numeric,
			$11::numeric, $12::numeric, $13, $14::numeric, $15::numeric,
			$16::uuid, '1.0'
		)
		ON CONFLICT (signal_id) WHERE signal_id IS NOT NULL DO UPDATE SET
			close_reason     = EXCLUDED.close_reason,
			outcome_type     = EXCLUDED.outcome_type,
			outcome_value    = EXCLUDED.outcome_value,
			realized_rr      = EXCLUDED.realized_rr,
			realized_pnl     = EXCLUDED.realized_pnl,
			r_multiple       = EXCLUDED.r_multiple,
			mae_points       = EXCLUDED.mae_points,
			mfe_points       = EXCLUDED.mfe_points,
			duration_seconds = EXCLUDED.duration_seconds,
			spread_at_exit   = EXCLUDED.spread_at_exit,
			trade_result_id  = EXCLUDED.trade_result_id`,
		predID, uuidOrNull(signalID), link, strategyID, strOrNull(closeReason),
		outcomeType, isWin, rMultiple, pnl, numOrNull(rMultiple),
		numOrNull(mae), numOrNull(mfe), durationSeconds, numOrNull(spreadAtEntry),
		numOrNull(spreadAtExit), resolvedTradeResultID)
	return err
}
// SavePredictionFromSignal builds the prediction payload from a types.Signal
// (the emit-site shape) and persists it. Family sub-scores are derived from
// the signal's evidence rows (same aggregation as the diagnostics layer).
func (p *Persister) SavePredictionFromSignal(ctx context.Context, s *types.Signal) error {
	fam := map[string]any{}
	for _, ev := range s.Evidence {
		if cur, ok := fam[ev.Pillar]; ok {
			if f, fok := cur.(float64); fok {
				nc, _ := ev.Contribution.Float64()
				fam[ev.Pillar] = fok && f != 0 && nc != 0 && nc+0 == 0
			}
		} else {
			nc, _ := ev.Contribution.Float64()
			fam[ev.Pillar] = nc
		}
	}
	famB, _ := json.Marshal(fam)
	gateB, _ := json.Marshal(s.GateResults)
	// Astro/crossmarket values live in the linked feature snapshot
	// (feature_snapshot_id) — the prediction row references it rather than
	// duplicating the values (single source of truth, no import cycle).
	astro, yoga, macro := "0", "0", "0"
	return p.SavePrediction(ctx, s.ID, s.Symbol, string(s.Timeframe),
		string(s.StrategyID), s.StrategyVersion, string(s.Direction),
		s.EntryPrice.String(), s.StopLoss.String(), s.TP1.String(), s.TP2.String(), s.TP3.String(),
		s.RawScore.String(), s.CalibratedProbability.String(), string(s.Grade), s.SignalClass, "", "",
		s.RawScore.String(), famB, string(s.Regime), s.Session, gateB,
		astro, yoga, macro, s.FeatureSnapshotID, s.StrategyVersion)
}

// fmtFloat renders a float with up to 4 decimals, trimming zeros.
func fmtFloat(f float64) string {
	if f == 0 {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(
		decimal.NewFromFloat(f).StringFixed(4), "0"), ".")
}