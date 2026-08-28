package store

import (
	"context"
	"testing"
	"time"

	"pat-engine/internal/backtest"
)

// TestDegradedStore proves the engine keeps working (no panic, in-memory buffer
// retains records) when Postgres/Valkey are unreachable.
func TestDegradedStore(t *testing.T) {
	s := New(context.Background(), "", "") // no DSN => degraded
	defer s.Close()

	pg, vk := s.Healthy()
	if pg || vk {
		t.Fatalf("expected degraded store, got pg=%v vk=%v", pg, vk)
	}

	s.SaveBar(context.Background(), backtest.Bar{Close: 2000, High: 2001, Low: 1999, Open: 2000, Spread: 0.2})
	s.SaveSignal(context.Background(), SignalRecord{
		ID: "TREND_SWING-1", TS: time.Now(), StrategyID: "TREND_SWING",
		Direction: "BUY", Entry: 2000, SL: 1995, TP1: 2008, RawScore: 76, Status: "EXECUTABLE",
	})

	recent := s.RecentSignals(context.Background(), 10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 in-memory signal, got %d", len(recent))
	}
	if recent[0].ID != "TREND_SWING-1" {
		t.Fatalf("unexpected signal id: %s", recent[0].ID)
	}
}
