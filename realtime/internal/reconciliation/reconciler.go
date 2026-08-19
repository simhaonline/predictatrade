// Package reconciliation tracks signal lifecycle and verifies delivery.
// SOW Section 24: Signal lifecycle and reconciliation
package reconciliation

import (
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// Reconciler tracks signal lifecycle states and delivery.
type Reconciler struct {
	mu         sync.RWMutex
	signals    map[string]*SignalRecord
}

type SignalRecord struct {
	Signal        *types.Signal
	DeliveredTo   []string
	AcknowledgedBy []string
	Status        types.SignalStatus
	UpdatedAt     time.Time
}

func NewReconciler() *Reconciler {
	return &Reconciler{signals: make(map[string]*SignalRecord)}
}

func (r *Reconciler) RecordSignal(signal *types.Signal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.signals[signal.ID] = &SignalRecord{
		Signal:    signal,
		Status:    signal.Status,
		UpdatedAt: time.Now().UTC(),
	}
}

func (r *Reconciler) RecordDelivery(signalID, userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.signals[signalID]; ok {
		rec.DeliveredTo = append(rec.DeliveredTo, userID)
		rec.UpdatedAt = time.Now().UTC()
	}
}

func (r *Reconciler) RecordAcknowledgement(signalID, userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.signals[signalID]; ok {
		rec.AcknowledgedBy = append(rec.AcknowledgedBy, userID)
		rec.UpdatedAt = time.Now().UTC()
	}
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
