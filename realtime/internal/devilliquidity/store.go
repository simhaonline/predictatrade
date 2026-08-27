package devilliquidity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store persists Devil's Marks and events. Persistence is best-effort: any
// failure is logged and never breaks the in-memory engine.
type Store struct {
	db *sql.DB
}

// NewStore opens a Postgres connection (pgx stdlib). A blank dbURL disables it.
func NewStore(dbURL string) (*Store, error) {
	if dbURL == "" {
		return &Store{}, nil
	}
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("devilliquidity store: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("devilliquidity store ping: %w", err)
	}
	return &Store{db: db}, nil
}

// Enabled reports whether persistence is active.
func (s *Store) Enabled() bool { return s.db != nil }

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// UpsertMark writes the current mark state.
func (s *Store) UpsertMark(m *DevilMark) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(context.Background(), upsertMarkSQL,
		m.ID, m.Symbol, m.Timeframe, string(m.Direction), m.MarkPrice,
		m.Open, m.High, m.Low, m.Close, m.Range, m.Body, m.BodyRatio,
		m.UpperWick, m.LowerWick, m.UpperWickRatio, m.LowerWickRatio,
		m.ATR, m.RangeATRRatio, m.BodyExpansion, m.Volume, m.VolumeRatio,
		m.Spread, m.Digits, m.TickSize,
		m.FVGPresent, m.FVGID, m.BOSPresent, m.MSSPresent, m.CHoCHPresent,
		m.FormationSession, m.FormationRegime,
		m.MarkQuality, m.PriorityScore, string(m.State),
		nullTime(m.FirstApproachAt), nullTime(m.FirstTouchAt), nullTime(m.FirstSweepAt),
		m.SweepLow, m.SweepHigh, nullTime(m.ReclaimAt), nullTime(m.ReversalConfirmedAt),
		m.SweepDepthATR, m.ReclaimStrength,
		m.ReversalScore, m.CombinedScore, m.DistanceATR,
		nullTime(m.ExpiredAt), nullTime(m.InvalidatedAt), nullTime(m.ResolvedAt),
		m.FeedSource, m.Broker, m.ServerID, m.ConfigVersion,
		m.DetectedAt, m.UpdatedAt,
	)
	return err
}

// InsertEvent appends a lifecycle event.
func (s *Store) InsertEvent(ev DevilEvent) error {
	if s.db == nil {
		return nil
	}
	meta, _ := json.Marshal(ev.Metadata)
	_, err := s.db.ExecContext(context.Background(), insertEventSQL,
		ev.MarkID, ev.Symbol, ev.Timeframe, ev.EventType,
		string(ev.StateFrom), string(ev.StateTo),
		ev.Price, ev.MarkPrice, ev.DistanceATR, ev.ATR, ev.Spread,
		ev.QualityScore, ev.ReversalScore, string(meta),
	)
	return err
}

// RecentMarks returns active marks for the API (terminal states excluded).
func (s *Store) RecentMarks(limit int) ([]*DevilMark, error) {
	if s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(context.Background(), recentMarksSQL, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*DevilMark
	for rows.Next() {
		m := &DevilMark{}
		var fApproach, fTouch, fSweep, fReclaim, fRev, fDetected, fUpdated sql.NullTime
		var dir, status string
		if err := rows.Scan(
			&m.ID, &m.Symbol, &m.Timeframe, &dir, &m.MarkPrice, &m.Open, &m.High, &m.Low, &m.Close,
			&m.Range, &m.Body, &m.BodyRatio, &m.UpperWick, &m.LowerWick, &m.UpperWickRatio, &m.LowerWickRatio,
			&m.ATR, &m.RangeATRRatio, &m.BodyExpansion, &m.Volume, &m.VolumeRatio,
			&m.Spread, &m.Digits, &m.TickSize, &m.FVGPresent, &m.BOSPresent, &m.MSSPresent, &m.CHoCHPresent,
			&m.FormationSession, &m.FormationRegime, &m.MarkQuality, &m.PriorityScore, &status,
			&fApproach, &fTouch, &fSweep, &fReclaim, &fRev,
			&m.SweepDepthATR, &m.ReclaimStrength, &m.ReversalScore, &m.CombinedScore, &m.DistanceATR,
			&fDetected, &fUpdated,
		); err != nil {
			return nil, err
		}
		m.Direction = MarkDirection(dir)
		m.State = MarkState(status)
		if fApproach.Valid {
			t := fApproach.Time
			m.FirstApproachAt = &t
		}
		if fTouch.Valid {
			t := fTouch.Time
			m.FirstTouchAt = &t
		}
		if fSweep.Valid {
			t := fSweep.Time
			m.FirstSweepAt = &t
		}
		if fReclaim.Valid {
			t := fReclaim.Time
			m.ReclaimAt = &t
		}
		if fRev.Valid {
			t := fRev.Time
			m.ReversalConfirmedAt = &t
		}
		m.DetectedAt = fDetected.Time
		m.UpdatedAt = fUpdated.Time
		out = append(out, m)
	}
	return out, nil
}
