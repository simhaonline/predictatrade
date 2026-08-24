// Session P&L anchoring (R4 / PT1-PT4 / P&L1-P&L6).
//
// Start-of-period equity anchors are persisted in Valkey under
// pat:pnl_anchor:{day|week|month} on the first observation of each period.
// P&L is measured as account-equity delta vs the anchor (equity already
// includes floating P&L from open positions). Periods roll over by UTC
// calendar day, ISO-8601 week and UTC month.
//
// Fail-closed: when no anchor can be established or read, the snapshot is
// marked Known=false and the P&L gates veto with pnl_state_unknown.
package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/predictatrade/realtime/internal/cache"
)

// Period identifies a P&L anchor window.
type Period string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
)

// AnchorKey is the Valkey key for a period anchor: pat:pnl_anchor:{period}.
func AnchorKey(p Period) string { return "pat:pnl_anchor:" + string(p) }

// AnchorTTL covers a full month so daily/weekly anchors never expire early.
const AnchorTTL = 40 * 24 * time.Hour

// PeriodAnchor is the persisted start-of-period equity record.
type PeriodAnchor struct {
	PeriodID string  `json:"period_id"`
	Equity   float64 `json:"equity"`
	SetAt    string  `json:"set_at"`
}

// AnchorStore abstracts the persistence of period anchors.
type AnchorStore interface {
	Get(key string) ([]byte, error)
	SetNX(key string, val []byte, ttl time.Duration) (bool, error)
}

// ValkeyAnchorStore adapts the shared Valkey cache to AnchorStore.
type ValkeyAnchorStore struct {
	cache *cache.ValkeyCache
}

func NewValkeyAnchorStore(c *cache.ValkeyCache) *ValkeyAnchorStore {
	return &ValkeyAnchorStore{cache: c}
}

func (s *ValkeyAnchorStore) Get(key string) ([]byte, error) {
	if s == nil || s.cache == nil {
		return nil, fmt.Errorf("valkey unavailable")
	}
	return s.cache.GetRaw(key)
}

func (s *ValkeyAnchorStore) SetNX(key string, val []byte, ttl time.Duration) (bool, error) {
	if s == nil || s.cache == nil {
		return false, fmt.Errorf("valkey unavailable")
	}
	return s.cache.SetNXRaw(context.Background(), key, val, ttl)
}

// PnLSnapshot carries session P&L percentages to the gates.
type PnLSnapshot struct {
	Known    bool             `json:"known"` // false → gates must veto pnl_state_unknown
	Equity   float64          `json:"equity"`
	PeriodPc map[Period]float64 `json:"period_pct"` // % change vs period anchor
	AsOf     time.Time        `json:"as_of"`
}

// PeriodID returns the canonical period identifier at `now` (UTC-based;
// ISO-8601 week numbering for weeks).
func PeriodID(now time.Time, p Period) string {
	now = now.UTC()
	switch p {
	case PeriodDay:
		return now.Format("2006-01-02")
	case PeriodWeek:
		y, w := now.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", y, w)
	case PeriodMonth:
		return now.Format("2006-01")
	default:
		return "unknown"
	}
}

// PnLTracker maintains period anchors and produces PnLSnapshots.
type PnLTracker struct {
	store AnchorStore
}

func NewPnLTracker(store AnchorStore) *PnLTracker {
	return &PnLTracker{store: store}
}

// Update records the current equity and returns the session P&L snapshot.
// On the first observation of a period the current equity becomes the
// anchor (P&L = 0). Any storage failure marks the snapshot unknown —
// fail-closed.
func (t *PnLTracker) Update(equity float64, now time.Time) PnLSnapshot {
	snap := PnLSnapshot{
		Known:    equity > 0,
		Equity:   equity,
		PeriodPc: make(map[Period]float64, 3),
		AsOf:     now.UTC(),
	}
	if t == nil || t.store == nil {
		snap.Known = false
		return snap
	}
	if !snap.Known {
		return snap
	}
	for _, p := range []Period{PeriodDay, PeriodWeek, PeriodMonth} {
		pct, ok := t.periodPct(p, equity, now)
		if !ok {
			snap.Known = false
			return snap
		}
		snap.PeriodPc[p] = pct
	}
	return snap
}

func (t *PnLTracker) periodPct(p Period, equity float64, now time.Time) (float64, bool) {
	key := AnchorKey(p)
	id := PeriodID(now, p)

	raw, err := t.store.Get(key)
	if err == nil && len(raw) > 0 {
		var anchor PeriodAnchor
		if jsonErr := json.Unmarshal(raw, &anchor); jsonErr == nil &&
			anchor.PeriodID == id && anchor.Equity > 0 {
			return (equity - anchor.Equity) / anchor.Equity * 100.0, true
		}
		// Corrupt record or period rollover → fall through to re-anchor.
	}

	// First observation of this period (or rollover): persist anchor atomically.
	blob, mErr := json.Marshal(PeriodAnchor{PeriodID: id, Equity: equity, SetAt: now.UTC().Format(time.RFC3339)})
	if mErr != nil {
		return 0, false
	}
	set, sErr := t.store.SetNX(key, blob, AnchorTTL)
	if sErr != nil {
		return 0, false
	}
	if !set {
		// Another writer won the race — re-read once; if it is a valid
		// anchor for this period use it, otherwise fail closed.
		raw, err = t.store.Get(key)
		if err != nil || len(raw) == 0 {
			return 0, false
		}
		var anchor PeriodAnchor
		if json.Unmarshal(raw, &anchor) != nil || anchor.PeriodID != id || anchor.Equity <= 0 {
			return 0, false
		}
		return (equity - anchor.Equity) / anchor.Equity * 100.0, true
	}
	// We set the anchor: P&L for this period starts at zero.
	return 0, true
}
