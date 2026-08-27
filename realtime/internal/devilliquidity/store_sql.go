package devilliquidity

// SQL for the Devil Liquidity persistence layer (prompt.md Sections 51-55).
// Tables are created by migration 080_devil_liquidity.sql.

const upsertMarkSQL = `
INSERT INTO devil_liquidity_marks (
	id, symbol, timeframe, direction, mark_price,
	open, high, low, close, range, body, body_ratio,
	upper_wick, lower_wick, upper_wick_ratio, lower_wick_ratio,
	atr, range_atr_ratio, body_expansion_ratio, volume, volume_ratio,
	spread, digits, tick_size,
	fvg_present, fvg_id, bos_present, mss_present, choch_present,
	formation_session, formation_regime,
	mark_quality_score, priority_score, status,
	first_approach_at, first_touch_at, first_sweep_at,
	sweep_low, sweep_high, reclaim_at, reversal_confirmed_at,
	sweep_depth_atr, reclaim_strength,
	reversal_score, combined_score, distance_atr,
	expired_at, invalidated_at, resolved_at,
	feed_source, broker, server_identifier, config_version,
	detected_at, updated_at
) VALUES (
	$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
	$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,
	$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52
)
ON CONFLICT (id) DO UPDATE SET
	mark_price=EXCLUDED.mark_price, high=EXCLUDED.high, low=EXCLUDED.low,
	close=EXCLUDED.close, atr=EXCLUDED.atr, body_ratio=EXCLUDED.body_ratio,
	lower_wick=EXCLUDED.lower_wick, upper_wick=EXCLUDED.upper_wick,
	volume=EXCLUDED.volume, volume_ratio=EXCLUDED.volume_ratio,
	mark_quality_score=EXCLUDED.mark_quality_score, priority_score=EXCLUDED.priority_score,
	status=EXCLUDED.status, first_approach_at=EXCLUDED.first_approach_at,
	first_touch_at=EXCLUDED.first_touch_at, first_sweep_at=EXCLUDED.first_sweep_at,
	sweep_low=EXCLUDED.sweep_low, sweep_high=EXCLUDED.sweep_high,
	reclaim_at=EXCLUDED.reclaim_at, reversal_confirmed_at=EXCLUDED.reversal_confirmed_at,
	sweep_depth_atr=EXCLUDED.sweep_depth_atr, reclaim_strength=EXCLUDED.reclaim_strength,
	reversal_score=EXCLUDED.reversal_score, combined_score=EXCLUDED.combined_score,
	distance_atr=EXCLUDED.distance_atr, expired_at=EXCLUDED.expired_at,
	invalidated_at=EXCLUDED.invalidated_at, resolved_at=EXCLUDED.resolved_at,
	updated_at=EXCLUDED.updated_at
`

const insertEventSQL = `
INSERT INTO devil_liquidity_events (
	mark_id, symbol, timeframe, event_type, state_from, state_to,
	price, mark_price, distance_atr, atr, spread,
	quality_score, reversal_score, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
`

const recentMarksSQL = `
SELECT id, symbol, timeframe, direction, mark_price, open, high, low, close,
	range, body, body_ratio, upper_wick, lower_wick, upper_wick_ratio, lower_wick_ratio,
	atr, range_atr_ratio, body_expansion_ratio, volume, volume_ratio,
	spread, digits, tick_size, fvg_present, bos_present, mss_present, choch_present,
	formation_session, formation_regime, mark_quality_score, priority_score, status,
	first_approach_at, first_touch_at, first_sweep_at, reclaim_at, reversal_confirmed_at,
	sweep_depth_atr, reclaim_strength, reversal_score, combined_score, distance_atr,
	detected_at, updated_at
FROM devil_liquidity_marks
WHERE status NOT IN ('INVALIDATED','EXPIRED','FAILED','MITIGATED')
ORDER BY detected_at DESC
LIMIT $1
`
