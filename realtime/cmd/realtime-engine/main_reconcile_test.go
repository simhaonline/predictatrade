package main

import (
	"testing"
	"time"
)

func TestShouldAlertFirstObservation(t *testing.T) {
	alerted := make(map[string]time.Time)
	now := time.Now().UTC()

	if !shouldAlert(alerted, "s1", now, 10*time.Minute) {
		t.Fatalf("first observation must alert")
	}
	if shouldAlert(alerted, "s1", now.Add(time.Second), 10*time.Minute) {
		t.Fatalf("repeat within re-alert window must be suppressed")
	}
	if !shouldAlert(alerted, "s1", now.Add(11*time.Minute), 10*time.Minute) {
		t.Fatalf("gap older than re-alert window must re-alert")
	}
	if !shouldAlert(alerted, "s2", now, 10*time.Minute) {
		t.Fatalf("second signal is an independent first observation")
	}
}

func TestShouldAlertPerSignalDedup(t *testing.T) {
	alerted := make(map[string]time.Time)
	now := time.Now().UTC()

	_ = shouldAlert(alerted, "s1", now, time.Minute)
	_ = shouldAlert(alerted, "s2", now, time.Minute)
	// s1 re-alerting must not reset s2's clock or vice versa.
	if shouldAlert(alerted, "s2", now.Add(30*time.Second), time.Minute) {
		t.Fatalf("s2 dedup must be independent of s1")
	}
	if len(alerted) != 2 {
		t.Fatalf("expected 2 tracked gaps, got %d", len(alerted))
	}
}
