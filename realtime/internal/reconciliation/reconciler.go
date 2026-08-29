// Package reconciliation tracks signal lifecycle and verifies delivery.
// SOW Section 24: Signal lifecycle and reconciliation
//
// Reconciliation is fail-closed in the sense that it never blocks signal
// generation; it only observes and reports lifecycle gaps (delivery that was
// never acknowledged, or an acknowledged order that never reported a fill).
package reconciliation

import (
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// Reconciler tracks signal lifecycle states and delivery.
type Reconciler struct {
	mu      sync.RWMutex
	signals map[string]*SignalRecord
}

type SignalRecord struct {
	Signal *types.Signal

	DeliveredTo []string
	DeliveredAt time.Time

	AcknowledgedBy []string
	AcknowledgedAt time.Time

	FillID   string
	FilledAt time.Time

	Status    types.SignalStatus
	UpdatedAt time.Time
}

func NewReconciler() *Reconciler {
	return &Reconciler{signals: make(map[string]*SignalRecord)}
}

func (r *Reconciler) RecordSignal(signal *types.Signal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.signals[signal.ID]; !ok {
		r.signals[signal.ID] = &SignalRecord{
			Signal:    signal,
			Status:    signal.Status,
			UpdatedAt: time.Now().UTC(),
		}
		return
	}
	// Preserve delivery/ack/fill timestamps if the signal is re-recorded.
	rec := r.signals[signal.ID]
	rec.Signal = signal
	rec.Status = signal.Status
	rec.UpdatedAt = time.Now().UTC()
}

// RecordDelivery records that the signal was pushed to a client/agent.
// It captures the first delivery timestamp so ACK-timeout accounting is stable.
func (r *Reconciler) RecordDelivery(signalID, userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.signals[signalID]
	if !ok {
		return
	}
	if rec.DeliveredAt.IsZero() {
		rec.DeliveredAt = time.Now().UTC()
	}
	rec.DeliveredTo = append(rec.DeliveredTo, userID)
	rec.UpdatedAt = time.Now().UTC()
}

// RecordAcknowledgement records that a client/agent acknowledged (executed) the
// signal. Acknowledgement is terminal for the delivery leg of reconciliation.
func (r *Reconciler) RecordAcknowledgement(signalID, userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.signals[signalID]
	if !ok {
		return
	}
	if rec.AcknowledgedAt.IsZero() {
		rec.AcknowledgedAt = time.Now().UTC()
	}
	rec.AcknowledgedBy = append(rec.AcknowledgedBy, userID)
	rec.Status = types.SignalAcknowledged
	rec.UpdatedAt = time.Now().UTC()
}

// RecordFill records that the acknowledged order produced a fill. Without a fill
// confirmation, an acknowledged signal is treated as a fill-level gap.
func (r *Reconciler) RecordFill(signalID, fillID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.signals[signalID]
	if !ok {
		return
	}
	if rec.FilledAt.IsZero() {
		rec.FilledAt = time.Now().UTC()
	}
	rec.FillID = fillID
	rec.Status = types.SignalFilled
	rec.UpdatedAt = time.Now().UTC()
}

func (r *Reconciler) UpdateStatus(signalID string, status types.SignalStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.signals[signalID]; ok {
		rec.Status = status
		rec.UpdatedAt = time.Now().UTC()
	}
}

func (r *Reconciler) GetSignal(signalID string) *SignalRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.signals[signalID]
}

func (r *Reconciler) RecentSignals(limit int) []*SignalRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*SignalRecord
	for _, rec := range r.signals {
		result = append(result, rec)
		if len(result) >= limit {
			break
		}
	}
	return result
}

// Tracked returns the number of signals currently registered. Used by the
// reconciliation monitor to expose registry size as a metric.
func (r *Reconciler) Tracked() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.signals)
}

// UnacknowledgedOlderThan returns signals that were delivered but never
// acknowledged within ttl, measured from first delivery. This is the delivery
// leg of reconciliation (BE-6): a signal that left the server but the edge
// never confirmed execution.
func (r *Reconciler) UnacknowledgedOlderThan(ttl time.Duration) []*SignalRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now().UTC()
	var out []*SignalRecord
	for _, rec := range r.signals {
		if !rec.DeliveredAt.IsZero() && rec.AcknowledgedAt.IsZero() &&
			now.Sub(rec.DeliveredAt) > ttl {
			out = append(out, rec)
		}
	}
	return out
}

// UnfilledOlderThan returns signals that were acknowledged (an order was sent by
// the edge) but never reported a fill within ttl, measured from acknowledgement.
// This is the fill leg of reconciliation (BE-6): a position that should exist
// but the broker snapshot / fill confirmation never arrived.
func (r *Reconciler) UnfilledOlderThan(ttl time.Duration) []*SignalRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now().UTC()
	var out []*SignalRecord
	for _, rec := range r.signals {
		if rec.AcknowledgedAt.IsZero() || !rec.FilledAt.IsZero() {
			continue
		}
		// Only consider states where a fill is expected.
		switch rec.Status {
		case types.SignalOrderSent, types.SignalAcknowledged,
			types.SignalTriggered, types.SignalPartiallyFilled:
			if now.Sub(rec.AcknowledgedAt) > ttl {
				out = append(out, rec)
			}
		}
	}
	return out
}

// PruneOlderThan removes records whose last update is older than retention,
// bounding in-memory growth. Returns the number of removed records. It never
// removes records that are still open (delivered-but-unacked) to preserve
// active reconciliation state.
func (r *Reconciler) PruneOlderThan(retention time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	removed := 0
	for id, rec := range r.signals {
		if !rec.AcknowledgedAt.IsZero() && !rec.FilledAt.IsZero() &&
			now.Sub(rec.UpdatedAt) > retention {
			delete(r.signals, id)
			removed++
		}
	}
	return removed
}
