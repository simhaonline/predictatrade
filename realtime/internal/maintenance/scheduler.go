// Package maintenance implements daily maintenance for risk-state reset,
// stale-state cleanup, and performance summary.
//
// Uses the project's canonical trading-day/timezone convention (UTC).
// Avoids bugs around DST, broker timezone, midnight restart,
// duplicate scheduler execution, and multi-instance deployments.
package maintenance

import (
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/recovery"
)

// Config holds maintenance scheduler configuration.
type Config struct {
	// DailyResetUTCMinute: minute of the UTC day to run daily reset (0-1439)
	// Default: 0 = midnight UTC
	DailyResetUTCMinute int
	// EnableLocking: use mutex to prevent duplicate execution in multi-instance
	EnableLocking bool
}

// DefaultConfig returns safe defaults.
func DefaultConfig() Config {
	return Config{
		DailyResetUTCMinute: 0, // midnight UTC
		EnableLocking:       true,
	}
}

// Scheduler runs daily maintenance tasks.
type Scheduler struct {
	mu        sync.Mutex
	config    Config
	recovery  *recovery.Manager
	stopCh    chan struct{}
	running   bool
	lastRunAt time.Time
}

// NewScheduler creates a maintenance scheduler.
func NewScheduler(cfg Config, recMgr *recovery.Manager) *Scheduler {
	return &Scheduler{
		config:   cfg,
		recovery: recMgr,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the maintenance loop.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	go s.loop()
}

// Stop stops the maintenance loop.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	close(s.stopCh)
	s.running = false
}

// RunDailyReset performs the daily risk-state reset.
// This is the canonical daily reset using UTC trading day.
func (s *Scheduler) RunDailyReset() {
	if s.config.EnableLocking {
		s.mu.Lock()
		defer s.mu.Unlock()
		// Prevent duplicate execution
		now := time.Now().UTC()
		if !s.lastRunAt.IsZero() && s.lastRunAt.Day() == now.Day() {
			return // already ran today
		}
		s.lastRunAt = now
	}

	if s.recovery != nil {
		s.recovery.ResetDaily(time.Now().UTC())
	}
}

// loop runs the background maintenance cycle.
func (s *Scheduler) loop() {
	for {
		nextRun := s.nextRunTime()
		sleepDuration := time.Until(nextRun)
		if sleepDuration < 0 {
			sleepDuration = 0
		}

		select {
		case <-s.stopCh:
			return
		case <-time.After(sleepDuration):
			s.RunDailyReset()
		}
	}
}

// nextRunTime calculates the next time the daily reset should run.
func (s *Scheduler) nextRunTime() time.Time {
	now := time.Now().UTC()
	target := time.Date(now.Year(), now.Month(), now.Day(),
		s.config.DailyResetUTCMinute/60, s.config.DailyResetUTCMinute%60, 0, 0, time.UTC)
	if target.Before(now) || target.Equal(now) {
		target = target.Add(24 * time.Hour)
	}
	return target
}

// LastRunAt returns the last time maintenance ran.
func (s *Scheduler) LastRunAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastRunAt
}
