package crossmarket

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OutcomeResolver monitors XAUUSD price movement and resolves shadow signal outcomes.
// It tracks TP1/TP2/TP3/SL/Expiry for each unresolved XAUUSD shadow snapshot.
// IMPORTANT: Only XAUUSD price determines outcomes. Reference assets (BTC, Oil, DXY)
// never determine trade outcomes — they are reference inputs only.
type OutcomeResolver struct {
	mu     sync.Mutex
	db     *sql.DB
	tickFn func() (bid, ask float64, ts time.Time)
}

// NewOutcomeResolver creates a resolver that checks XAUUSD price against active shadow signals.
func NewOutcomeResolver(db *sql.DB, tickFn func() (bid, ask float64, ts time.Time)) *OutcomeResolver {
	return &OutcomeResolver{db: db, tickFn: tickFn}
}

// OutcomeEvent represents a single resolution event (TP hit, SL hit, etc).
type OutcomeEvent struct {
	SnapshotID uuid.UUID `json:"snapshot_id"`
	EventType  string    `json:"event_type"`   // ENTRY_TRIGGERED, TP1_HIT, TP2_HIT, TP3_HIT, SL_HIT, EXPIRED
	Price      float64   `json:"price"`
	Timestamp  time.Time `json:"timestamp"`
}

// Resolve checks all unresolved XAUUSD shadow snapshots against current XAUUSD price.
// Called periodically (not on every tick) to avoid performance impact.
func (r *OutcomeResolver) Resolve(ctx context.Context) ([]OutcomeEvent, error) {
	if r == nil || r.db == nil || r.tickFn == nil {
		return nil, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Get current XAUUSD price
	bid, ask, ts := r.tickFn()
	if bid <= 0 || ask <= 0 {
		return nil, nil
	}
	midPrice := (bid + ask) / 2

	// Fetch unresolved snapshots with valid entry/SL/TP
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, signal_id, strategy, direction, entry, stop_loss, tp1, tp2, tp3, expiry, timestamp
		FROM trading.cross_market_shadow_snapshots
		WHERE outcome = 'UNRESOLVED'
		  AND entry > 0 AND stop_loss > 0 AND tp1 > 0
		  AND direction IN ('BUY', 'SELL', 'BUY_CANDIDATE', 'SELL_CANDIDATE')
		ORDER BY timestamp ASC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutcomeEvent

	for rows.Next() {
		var snapID uuid.UUID
		var signalID string
		var strategy, direction string
		var entry, sl, tp1, tp2, tp3 float64
		var expiry, snapTime time.Time

		if err := rows.Scan(&snapID, &signalID, &strategy, &direction, &entry, &sl, &tp1, &tp2, &tp3, &expiry, &snapTime); err != nil {
			continue
		}

		isBuy := direction == "BUY" || direction == "BUY_CANDIDATE"

		// Check expiry first
		if !expiry.IsZero() && ts.After(expiry) {
			event := OutcomeEvent{
				SnapshotID: snapID,
				EventType:  "EXPIRED",
				Price:      midPrice,
				Timestamp:  ts,
			}
			events = append(events, event)
			r.updateOutcome(ctx, snapID, "EXPIRED", midPrice, midPrice, midPrice, 0, 0, 0)
			continue
		}

		// Check TP and SL based on direction
		tp1Hit := false
		slHit := false
		ambiguous := false

		if isBuy {
			tp1Hit = bid >= tp1
			slHit = ask <= sl
		} else {
			tp1Hit = ask <= tp1
			slHit = bid >= sl
		}

		// Same-bar ambiguity: both TP and SL in range
		if tp1Hit && slHit {
			ambiguous = true
		}

		if ambiguous {
			event := OutcomeEvent{
				SnapshotID: snapID,
				EventType:  "AMBIGUOUS_PATH",
				Price:      midPrice,
				Timestamp:  ts,
			}
			events = append(events, event)
			r.updateOutcome(ctx, snapID, "AMBIGUOUS", midPrice, midPrice, midPrice, 0, 0, 0)
			continue
		}

		if tp1Hit {
			// Determine which TP was hit
			outcomeType := "TP1_HIT"
			rMult := math.Abs(tp1-entry) / math.Abs(entry-sl)

			if tp2 > 0 {
				tp2Hit := isBuy && bid >= tp2
				if !isBuy {
					tp2Hit = ask <= tp2
				}
				if tp2Hit {
					outcomeType = "TP2_HIT"
					rMult = math.Abs(tp2-entry) / math.Abs(entry-sl)
				}
			}

			if tp3 > 0 {
				tp3Hit := isBuy && bid >= tp3
				if !isBuy {
					tp3Hit = ask <= tp3
				}
				if tp3Hit {
					outcomeType = "TP3_HIT"
					rMult = math.Abs(tp3-entry) / math.Abs(entry-sl)
				}
			}

			// Calculate MFE/MAE
			mfe := math.Abs(midPrice - entry)
			mae := 0.0 // would track from historical data in production

			event := OutcomeEvent{
				SnapshotID: snapID,
				EventType:  outcomeType,
				Price:      midPrice,
				Timestamp:  ts,
			}
			events = append(events, event)
			r.updateOutcome(ctx, snapID, outcomeType, midPrice, mfe, mae, rMult, int(ts.Sub(snapTime).Seconds()), 0)
		} else if slHit {
			rMult := -math.Abs(entry-sl) / math.Abs(entry-sl) // -1R

			mfe := 0.0
			mae := math.Abs(midPrice - entry)

			event := OutcomeEvent{
				SnapshotID: snapID,
				EventType:  "SL_HIT",
				Price:      midPrice,
				Timestamp:  ts,
			}
			events = append(events, event)
			r.updateOutcome(ctx, snapID, "SL_HIT", midPrice, mfe, mae, rMult, 0, int(ts.Sub(snapTime).Seconds()))
		}
	}

	return events, nil
}

// updateOutcome marks a shadow snapshot as resolved with the outcome details.
func (r *OutcomeResolver) updateOutcome(ctx context.Context, snapID uuid.UUID, outcome string, price, mfe, mae, rMult float64, timeToTP, timeToSL int) {
	if r.db == nil {
		return
	}
	now := time.Now().UTC()
	_, _ = r.db.ExecContext(ctx, `
		UPDATE trading.cross_market_shadow_snapshots
		SET outcome = $1, mfe = $2, mae = $3, r_multiple = $4,
		    time_to_tp = $5, time_to_sl = $6, resolved_at = $7
		WHERE id = $8 AND outcome = 'UNRESOLVED'
	`, outcome, mfe, mae, rMult, timeToTP, timeToSL, now, snapID)
}

// Start runs the outcome resolver in a background goroutine.
func (r *OutcomeResolver) Start(ctx context.Context, intervalSec int) {
	if intervalSec <= 0 {
		intervalSec = 30 // check every 30 seconds
	}
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			events, err := r.Resolve(resolveCtx)
			cancel()
			if err != nil {
				continue
			}
			_ = events // events are logged/persisted in updateOutcome
		}
	}
}

// GetOutcomeStats returns outcome statistics by strategy.
type OutcomeStats struct {
	Strategy   string
	Total      int
	TP1Hits    int
	TP2Hits    int
	TP3Hits    int
	SLHits     int
	Expired    int
	Ambiguous  int
	Unresolved int
	WinRate    float64
	AvgR       float64
}

// GetStats queries outcome statistics from the shadow dataset.
func (r *OutcomeResolver) GetStats(ctx context.Context) ([]OutcomeStats, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT strategy,
			count(*) as total,
			count(CASE WHEN outcome = 'TP1_HIT' THEN 1 END) as tp1,
			count(CASE WHEN outcome = 'TP2_HIT' THEN 1 END) as tp2,
			count(CASE WHEN outcome = 'TP3_HIT' THEN 1 END) as tp3,
			count(CASE WHEN outcome = 'SL_HIT' THEN 1 END) as sl,
			count(CASE WHEN outcome = 'EXPIRED' THEN 1 END) as expired,
			count(CASE WHEN outcome = 'AMBIGUOUS' THEN 1 END) as ambiguous,
			count(CASE WHEN outcome = 'UNRESOLVED' THEN 1 END) as unresolved,
			COALESCE(avg(CASE WHEN outcome != 'UNRESOLVED' THEN r_multiple END), 0) as avg_r
		FROM trading.cross_market_shadow_snapshots
		WHERE entry > 0 AND stop_loss > 0 AND tp1 > 0
		GROUP BY strategy
		ORDER BY strategy
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []OutcomeStats
	for rows.Next() {
		var s OutcomeStats
		if err := rows.Scan(&s.Strategy, &s.Total, &s.TP1Hits, &s.TP2Hits, &s.TP3Hits, &s.SLHits, &s.Expired, &s.Ambiguous, &s.Unresolved, &s.AvgR); err != nil {
			continue
		}
		resolved := s.Total - s.Unresolved
		if resolved > 0 {
			wins := s.TP1Hits + s.TP2Hits + s.TP3Hits
			s.WinRate = float64(wins) / float64(resolved)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// ProductScopeGuard enforces that only XAUUSD signals are generated.
// This is a defensive check to prevent reference assets from accidentally
// entering signal generation, execution, or notification paths.
type ProductScopeGuard struct {
	TradableSymbols    []string
	ReferenceSymbols   []string
}

// NewProductScopeGuard creates the product scope guard.
func NewProductScopeGuard() *ProductScopeGuard {
	return &ProductScopeGuard{
		TradableSymbols:  []string{"XAUUSD"},
		ReferenceSymbols: []string{"DXY", "EURUSD", "BTCUSD", "WTI", "VIX", "US10Y", "US10Y_REAL_YIELD", "COT"},
	}
}

// IsTradable returns true if the symbol is a tradable product (XAUUSD only).
func (g *ProductScopeGuard) IsTradable(symbol string) bool {
	for _, s := range g.TradableSymbols {
		if s == symbol {
			return true
		}
	}
	return false
}

// IsReference returns true if the symbol is a reference-only asset.
func (g *ProductScopeGuard) IsReference(symbol string) bool {
	for _, s := range g.ReferenceSymbols {
		if s == symbol {
			return true
		}
	}
	return false
}

// CanGenerateSignal returns true only for XAUUSD.
func (g *ProductScopeGuard) CanGenerateSignal(symbol string) bool {
	return g.IsTradable(symbol)
}

// CanExecute returns true only for XAUUSD.
func (g *ProductScopeGuard) CanExecute(symbol string) bool {
	return g.IsTradable(symbol)
}

// FormatScope returns a human-readable scope description.
func (g *ProductScopeGuard) FormatScope() string {
	return fmt.Sprintf("Tradable: %v | Reference-only: %v", g.TradableSymbols, g.ReferenceSymbols)
}
