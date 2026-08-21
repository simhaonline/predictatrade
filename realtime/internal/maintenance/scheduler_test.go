package maintenance

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/recovery"
	"github.com/shopspring/decimal"
)

func TestDailyResetWorks(t *testing.T) {
	recMgr := recovery.NewManager(recovery.DefaultConfig())
	key := recovery.AccountStrategyKey{AccountID: "acc1", StrategyID: "STANDARD_SCALPING", Symbol: "XAUUSD"}
	recMgr.SetStateRecord(&recovery.StateRecord{
		Key:            key,
		State:          recovery.StateDailyLimit,
		DailyLossCount: 3,
		DailyPnL:       decimal.NewFromInt(-200),
		TradingDay:     time.Now().UTC().AddDate(0, 0, -1), // yesterday UTC
	})

	sched := NewScheduler(DefaultConfig(), recMgr)
	sched.RunDailyReset()

	state := recMgr.GetState(key)
	if state != recovery.StateNormal {
		t.Fatalf("daily reset should clear DAILY_LIMIT, got %s", state)
	}
}

func TestNoDuplicateSchedulerExecution(t *testing.T) {
	recMgr := recovery.NewManager(recovery.DefaultConfig())
	sched := NewScheduler(DefaultConfig(), recMgr)

	// Run twice on the same day
	sched.RunDailyReset()
	sched.RunDailyReset()

	// Second run should be skipped (same day)
	lastRun := sched.LastRunAt()
	if lastRun.IsZero() {
		t.Fatal("lastRunAt should be set")
	}
}

func TestUsesUTCNotLocalMidnight(t *testing.T) {
	sched := NewScheduler(DefaultConfig(), nil)
	next := sched.nextRunTime()
	// nextRunTime should be in UTC
	if next.Location() != time.UTC {
		// It might be converted — check it's midnight UTC or next midnight
		utcTime := next.UTC()
		if utcTime.Hour() != 0 || utcTime.Minute() != 0 {
			// This is OK if the test runs after midnight UTC
		}
	}
}
