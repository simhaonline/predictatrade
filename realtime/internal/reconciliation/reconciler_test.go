package reconciliation

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

func newSignal(id string) *types.Signal {
	return &types.Signal{ID: id, Status: types.SignalActive}
}

func TestRecordSignalPreservesTimestamps(t *testing.T) {
	r := NewReconciler()
	sig := newSignal("s1")
	r.RecordSignal(sig)
	r.RecordDelivery("s1", "agentA")

	// Re-recording the same signal must not clear delivery state.
	r.RecordSignal(sig)
	if got := r.GetSignal("s1"); got == nil || got.DeliveredAt.IsZero() {
		t.Fatalf("re-record cleared delivery timestamp")
	}
}

func TestUnacknowledgedOlderThan(t *testing.T) {
	r := NewReconciler()
	r.RecordSignal(newSignal("s1"))
	r.RecordSignal(newSignal("s2"))
	r.RecordSignal(newSignal("s3"))

	r.RecordDelivery("s1", "a")
	r.RecordDelivery("s2", "a")
	// s3 never delivered.

	// Simulate s1 delivered > ttl ago, s2 just now.
	r.signals["s1"].DeliveredAt = time.Now().UTC().Add(-10 * time.Minute)

	stale := r.UnacknowledgedOlderThan(5 * time.Minute)
	if len(stale) != 1 || stale[0].Signal.ID != "s1" {
		t.Fatalf("expected only s1 stale, got %v", ids(stale))
	}

	// After acknowledging s1 it must no longer be reported.
	r.RecordAcknowledgement("s1", "a")
	if got := r.UnacknowledgedOlderThan(5 * time.Minute); len(got) != 0 {
		t.Fatalf("acknowledged signal still reported stale: %v", ids(got))
	}
}

func TestUnfilledOlderThan(t *testing.T) {
	r := NewReconciler()
	r.RecordSignal(newSignal("s1"))
	r.RecordSignal(newSignal("s2"))

	r.RecordAcknowledgement("s1", "a")
	r.RecordAcknowledgement("s2", "a")
	// s1 acknowledged long ago with no fill.
	r.signals["s1"].AcknowledgedAt = time.Now().UTC().Add(-10 * time.Minute)
	// s2 acknowledged recently.
	r.signals["s2"].AcknowledgedAt = time.Now().UTC()
	r.signals["s2"].Status = types.SignalOrderSent // expect fill

	unfilled := r.UnfilledOlderThan(5 * time.Minute)
	if len(unfilled) != 1 || unfilled[0].Signal.ID != "s1" {
		t.Fatalf("expected only s1 unfilled, got %v", ids(unfilled))
	}

	// Recording a fill clears s1 from the unfilled set.
	r.RecordFill("s1", "fill-1")
	if got := r.UnfilledOlderThan(5 * time.Minute); len(got) != 0 {
		t.Fatalf("filled signal still reported unfilled: %v", ids(got))
	}
}

func TestPruneLeavesOpenRecords(t *testing.T) {
	r := NewReconciler()
	r.RecordSignal(newSignal("open"))
	r.RecordSignal(newSignal("done"))
	r.RecordDelivery("open", "a")
	r.RecordDelivery("done", "a")
	r.RecordAcknowledgement("done", "a")
	r.RecordFill("done", "f1")

	old := time.Now().UTC().Add(-2 * time.Hour)
	r.signals["open"].UpdatedAt = old
	r.signals["done"].UpdatedAt = old

	removed := r.PruneOlderThan(time.Hour)
	if removed != 1 {
		t.Fatalf("expected 1 pruned, got %d", removed)
	}
	if r.GetSignal("open") == nil {
		t.Fatalf("open (unacked) record must not be pruned")
	}
	if r.GetSignal("done") != nil {
		t.Fatalf("fully-filled record should have been pruned")
	}
}

func ids(recs []*SignalRecord) []string {
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.Signal.ID)
	}
	return out
}
