package marketdata

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestDailyBudgetConsumption verifies the UTC-day credit budget: consumes
// credits until exhausted, then fails fast without blocking.
func TestDailyBudgetConsumption(t *testing.T) {
	d := newDailyBudget(3)
	if d.isExhausted() {
		t.Fatal("fresh budget must not be exhausted")
	}
	for i := 0; i < 3; i++ {
		if !d.tryConsume() {
			t.Fatalf("consume %d: expected success", i+1)
		}
	}
	if !d.isExhausted() {
		t.Error("budget must be exhausted after limit is reached")
	}
	if d.tryConsume() {
		t.Error("consume beyond limit must fail fast")
	}
	if d.usedToday() != 3 {
		t.Errorf("used = %d, want 3 (no credit consumed after exhaustion)", d.usedToday())
	}
}

// TestDailyBudgetResetsOnDayRoll verifies the UTC-midnight reset logic by
// simulating a day-number advance.
func TestDailyBudgetResetsOnDayRoll(t *testing.T) {
	d := newDailyBudget(2)
	d.tryConsume()
	d.tryConsume()
	if !d.isExhausted() {
		t.Fatal("expected exhausted after 2 consumes")
	}
	// Simulate the next UTC day.
	d.mu.Lock()
	d.day -= 1
	d.mu.Unlock()
	if d.isExhausted() {
		t.Error("budget must reset when the UTC day rolls over")
	}
	if !d.tryConsume() {
		t.Error("consume after day roll must succeed")
	}
	if d.usedToday() != 1 {
		t.Errorf("used after roll = %d, want 1 (counter reset)", d.usedToday())
	}
}

// TestAcquireTwelveDataCreditDailyGuard verifies the combined gate: while the
// daily budget has room, acquire consumes both a daily credit and a
// per-minute token; once the daily budget is spent, acquire returns the typed
// error immediately (message classified as rate-limited downstream).
func TestAcquireTwelveDataCreditDailyGuard(t *testing.T) {
	// Save and restore the package singletons (tests share package state).
	origDaily := twelveDataDaily
	origLimiter := twelveDataLimiter
	defer func() {
		twelveDataDaily = origDaily
		twelveDataLimiter = origLimiter
	}()

	twelveDataDaily = newDailyBudget(2)
	twelveDataLimiter = newTokenBucket(10, time.Minute) // roomy: never blocks

	if err := acquireTwelveDataCredit(context.Background()); err != nil {
		t.Fatalf("first acquire should succeed, got %v", err)
	}
	if err := acquireTwelveDataCredit(ctxTest()); err != nil {
		t.Fatalf("second acquire should succeed, got %v", err)
	}
	err := acquireTwelveDataCredit(context.Background())
	if err == nil {
		t.Fatal("third acquire must fail fast on spent daily budget")
	}
	if !isRateLimited(err) {
		t.Errorf("budget error must classify as rate-limited downstream, got: %v", err)
	}
	if !strings.Contains(err.Error(), "UTC") {
		t.Errorf("error should explain the UTC reset, got: %v", err)
	}
}

// ctxTest returns a plain background context (named for clarity in the table
// above).
func ctxTest() context.Context { return context.Background() }
