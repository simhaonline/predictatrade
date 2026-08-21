package health

import (
	"testing"
	"time"
)

func TestStaleChecker_Fresh(t *testing.T) {
	s := NewStaleChecker(60*time.Second, 120*time.Second)
	s.UpdateLastCandleTime(time.Now())
	stale, critical, _ := s.Check()
	if stale || critical {
		t.Error("Fresh data should not be stale")
	}
	if !s.IsHealthy() {
		t.Error("Fresh data should be healthy")
	}
}

func TestStaleChecker_Stale(t *testing.T) {
	s := NewStaleChecker(60*time.Second, 120*time.Second)
	s.UpdateLastCandleTime(time.Now().Add(-90 * time.Second))
	stale, critical, _ := s.Check()
	if !stale {
		t.Error("90s old should be stale (threshold 60s)")
	}
	if critical {
		t.Error("90s old should not be critical (threshold 120s)")
	}
}

func TestStaleChecker_Critical(t *testing.T) {
	s := NewStaleChecker(60*time.Second, 120*time.Second)
	s.UpdateLastCandleTime(time.Now().Add(-180 * time.Second))
	stale, critical, _ := s.Check()
	if !stale || !critical {
		t.Error("180s old should be both stale and critical")
	}
	if s.IsHealthy() {
		t.Error("Critical data should not be healthy")
	}
}

func TestStaleChecker_NoData(t *testing.T) {
	s := NewStaleChecker(60*time.Second, 120*time.Second)
	stale, critical, _ := s.Check()
	if !stale || !critical {
		t.Error("No data should be stale and critical")
	}
}
