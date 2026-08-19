package recovery

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func testConfig() Config {
	return DefaultConfig()
}

func makeResult(accountID, strategyID string, pnl float64, isWin, isLoss bool, closeID string, day time.Time) TradeResult {
	return TradeResult{
		AccountID:    accountID,
		StrategyID:   strategyID,
		Symbol:       "XAUUSD",
		CloseEventID: closeID,
		PnL:          decimal.NewFromFloat(pnl),
		IsWin:        isWin,
		IsLoss:       isLoss,
		ClosedAt:     time.Now(),
		TradingDay:   day,
	}
}

func TestSingleLossDoesNotTriggerRecovery(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	state := m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -10, false, true, "c1", day))
	if state != StateNormal {
		t.Fatalf("single loss should not trigger recovery, got %s", state)
	}
}

func TestConsecutiveLossesTriggerRecovery(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key: key, State: StateNormal,
		StartingEquity: decimal.NewFromInt(10000),
		TradingDay:     day,
	})
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -10, false, true, "c1", day))
	state := m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -10, false, true, "c2", day))
	if state != StateRecovery {
		t.Fatalf("two consecutive losses should trigger recovery, got %s", state)
	}
}

func TestRecoveryReducesRisk(t *testing.T) {
	m := NewManager(testConfig())
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key: key, State: StateRecovery,
		StartingEquity: decimal.NewFromInt(10000),
		TradingDay:     time.Now(),
	})
	mult := m.GetSizeMultiplier(key)
	if mult != 0.5 {
		t.Fatalf("recovery size multiplier should be 0.5, got %f", mult)
	}
}

func TestRecoveryBlocksLowConfluence(t *testing.T) {
	m := NewManager(testConfig())
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key: key, State: StateRecovery,
		TradingDay: time.Now(),
	})
	allowed, reason := m.CheckSignal(key, 70, "A", 80)
	if allowed {
		t.Fatal("should block low confluence in recovery")
	}
	if reason != BlockLowConfluenceRecovery {
		t.Fatalf("expected LOW_CONFLUENCE_RECOVERY, got %s", reason)
	}
}

func TestRecoveryBlocksLowQuality(t *testing.T) {
	m := NewManager(testConfig())
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key: key, State: StateRecovery,
		TradingDay: time.Now(),
	})
	allowed, reason := m.CheckSignal(key, 85, "B", 80)
	if allowed {
		t.Fatal("should block low quality in recovery")
	}
	if reason != BlockLowQualityRecovery {
		t.Fatalf("expected LOW_QUALITY_RECOVERY, got %s", reason)
	}
}

func TestRecoveryBlocksLowConfidence(t *testing.T) {
	m := NewManager(testConfig())
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key: key, State: StateRecovery,
		TradingDay: time.Now(),
	})
	allowed, reason := m.CheckSignal(key, 85, "A", 50)
	if allowed {
		t.Fatal("should block low confidence in recovery")
	}
	if reason != BlockLowConfidenceRecovery {
		t.Fatalf("expected LOW_CONFIDENCE_RECOVERY, got %s", reason)
	}
}

func TestRecoveryExitAfterWins(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	m.SetStateRecord(&StateRecord{
		Key:            AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"},
		State:          StateRecovery,
		TradingDay:     day,
		StartingEquity: decimal.NewFromInt(10000),
	})
	// Two wins should exit recovery
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", 10, true, false, "w1", day))
	state := m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", 10, true, false, "w2", day))
	if state != StateNormal {
		t.Fatalf("two wins should exit recovery, got %s", state)
	}
}

func TestMaxRecoveryTradesHalt(t *testing.T) {
	m := NewManager(testConfig())
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key: key, State: StateRecovery,
		RecoveryTradesTaken: 3,
		TradingDay: time.Now(),
	})
	allowed, reason := m.CheckSignal(key, 85, "A", 80)
	if allowed {
		t.Fatal("should block when max recovery trades reached")
	}
	if reason != BlockMaxRecoveryTrades {
		t.Fatalf("expected MAX_RECOVERY_TRADES, got %s", reason)
	}
}

func TestDailyLossLimitWorks(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key:            key,
		State:          StateNormal,
		StartingEquity: decimal.NewFromInt(10000),
		TradingDay:     day,
	})
	// Loss of 250 = 2.5% of 10000 → exceeds 2% limit
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -250, false, true, "d1", day))
	state := m.GetState(key)
	if state != StateDailyLimit {
		t.Fatalf("daily loss limit should trigger DAILY_LIMIT, got %s", state)
	}
}

func TestPositiveDailyProfitNeverTriggersLossLimit(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key:            key,
		State:          StateNormal,
		StartingEquity: decimal.NewFromInt(10000),
		TradingDay:     day,
	})
	// Big profit of 500 = +5% — must NOT trigger loss limit
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", 500, true, false, "p1", day))
	state := m.GetState(key)
	if state != StateNormal {
		t.Fatalf("positive profit must never trigger loss limit, got %s", state)
	}
}

func TestDailyLossCountWorks(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key:            key,
		State:          StateNormal,
		StartingEquity: decimal.NewFromInt(100000),
		TradingDay:     day,
	})
	// 3 losses (each small enough to not hit percent limit)
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -5, false, true, "l1", day))
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -5, false, true, "l2", day))
	state := m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -5, false, true, "l3", day))
	if state != StateDailyLimit {
		t.Fatalf("3 daily losses should trigger DAILY_LIMIT, got %s", state)
	}
}

func TestDailyResetWorks(t *testing.T) {
	m := NewManager(testConfig())
	day1 := time.Now()
	day2 := day1.Add(24 * time.Hour)
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key:            key,
		State:          StateDailyLimit,
		DailyLossCount: 3,
		TradingDay:     day1,
		StartingEquity: decimal.NewFromInt(10000),
	})
	m.ResetDaily(day2)
	state := m.GetState(key)
	if state != StateNormal {
		t.Fatalf("daily reset should clear DAILY_LIMIT, got %s", state)
	}
}

func TestHaltExpiryWorks(t *testing.T) {
	m := NewManager(testConfig())
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	// Set halt that expired 1 minute ago
	m.SetStateRecord(&StateRecord{
		Key:       key,
		State:     StateHalted,
		HaltUntil: time.Now().Add(-1 * time.Minute),
		TradingDay: time.Now(),
	})
	// CheckSignal with expired halt should allow (transitions to recovery on next trade result)
	// But CheckSignal doesn't transition — it checks. Halt expiry happens on RecordTradeResult.
	// However, CheckSignal should not block if halt has expired.
	allowed, _ := m.CheckSignal(key, 85, "A", 80)
	if !allowed {
		t.Fatal("expired halt should not block")
	}
}

func TestDuplicateCloseEventIgnored(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key:            key,
		State:          StateNormal,
		StartingEquity: decimal.NewFromInt(100000),
		TradingDay:     day,
	})
	// Process first loss
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -10, false, true, "dup1", day))
	rec := m.GetStateRecord(key)
	lossCount := rec.DailyLossCount
	// Process same close event again — should be ignored
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -10, false, true, "dup1", day))
	rec2 := m.GetStateRecord(key)
	if rec2.DailyLossCount != lossCount {
		t.Fatalf("duplicate close event should be ignored: before=%d after=%d", lossCount, rec2.DailyLossCount)
	}
}

func TestRestartPersistenceDoesNotBypassHalt(t *testing.T) {
	m1 := NewManager(testConfig())
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m1.SetStateRecord(&StateRecord{
		Key:       key,
		State:     StateHalted,
		HaltUntil: time.Now().Add(30 * time.Minute),
		HaltReason: "test halt",
		TradingDay: time.Now(),
	})
	// Simulate restart: export and restore
	states := m1.AllStates()
	m2 := NewManager(testConfig())
	m2.RestoreStates(states)
	state := m2.GetState(key)
	if state != StateHalted {
		t.Fatalf("halt should survive restart, got %s", state)
	}
}

func TestDifferentAccountStateIsolated(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	// Account 1 has 2 losses
	m.SetStateRecord(&StateRecord{
		Key:            AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"},
		State:          StateRecovery,
		StartingEquity: decimal.NewFromInt(10000),
		TradingDay:     day,
	})
	// Account 2 should be NORMAL
	state2 := m.GetState(AccountStrategyKey{"acc2", "STANDARD_SCALPING", "XAUUSD"})
	if state2 != StateNormal {
		t.Fatalf("account2 should be NORMAL, got %s", state2)
	}
}

func TestLossDoesNotCreateReverseSignal(t *testing.T) {
	m := NewManager(testConfig())
	day := time.Now()
	key := AccountStrategyKey{"acc1", "STANDARD_SCALPING", "XAUUSD"}
	m.SetStateRecord(&StateRecord{
		Key:            key,
		State:          StateNormal,
		StartingEquity: decimal.NewFromInt(10000),
		TradingDay:     day,
	})
	// Process a loss
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -10, false, true, "r1", day))
	m.RecordTradeResult(makeResult("acc1", "STANDARD_SCALPING", -10, false, true, "r2", day))
	state := m.GetState(key)
	// In recovery — but no trade was created. CheckSignal only blocks.
	if state != StateRecovery {
		t.Fatalf("expected recovery, got %s", state)
	}
	// Verify no trade is created — CheckSignal returns false, not a direction
	allowed, _ := m.CheckSignal(key, 85, "A", 80)
	// In recovery with valid signal, it's allowed but with reduced size
	if !allowed {
		t.Fatal("valid signal in recovery should be allowed (with reduced size), not blocked")
	}
	mult := m.GetSizeMultiplier(key)
	if mult != 0.5 {
		t.Fatalf("recovery should reduce size to 0.5, got %f", mult)
	}
}
