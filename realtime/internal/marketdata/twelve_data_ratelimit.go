// Package marketdata — shared TwelveData API rate limiter.
//
// Twelve Data's free tier allows 8 API credits per MINUTE and 800 credits
// per DAY. The engine has TWO consumers of the same API key: the DXY provider
// (6 component symbols, batched → 1 credit/call) and the macro-asset provider
// (VIX/BTC/Oil/EURUSD/USDCHF, batched → 1 credit/call). Before batching, the
// two consumers fired ~12 calls per 5-min cycle (≈3,456/day) and blew through
// BOTH limits — every fetch returned 429 and DXY (a MANDATORY strategy pillar)
// never became available, forcing all swing strategies into NO-TRADE.
//
// Fix (three layers):
//  1. Both providers batch their symbols into a SINGLE multi-symbol call
//     (1 credit each instead of N) — steady state ≈ 2 credits / 5 min.
//  2. Every Twelve Data HTTP request passes through this shared token-bucket
//     limiter (6/min, headroom under the 8/min burst cap) so the combined
//     call volume from both consumers stays under the per-minute ceiling.
//  3. A shared UTC-day credit budget (twelveDataDailyLimit) FAILS FAST once
//     the free tier's 800/day allowance is spent — instead of burning retries
//     against a hard 429 for the rest of the UTC day. The budget resets at
//     UTC midnight. Steady-state usage (~576/day) leaves headroom; this guard
//     only trips after abnormal consumption (e.g. retry storms or a legacy
//     un-batched build), preserving credits instead of hammering the API.
//
// The limiter is a package-level singleton; both providers share it so their
// call volumes are counted together against the same budget.
package marketdata

import (
	"context"
	"sync"
	"time"
)

// twelveDataLimit is the maximum Twelve Data API credits we allow per minute.
// Free tier = 8/min; we stay conservative at 6/min to leave headroom for
// retries and to avoid tripping the hard 429 cutoff.
const twelveDataLimit = 6

// twelveDataDailyLimit is the maximum Twelve Data API credits we allow per
// UTC day (free tier = 800/day). Steady-state batched usage is ~576/day
// (2 credits / 5 min × 288 five-minute windows); the guard only trips when
// something abnormal is burning credits, at which point failing fast (and
// letting DXY/macro go RATE_LIMITED → strategy NO-TRADE) preserves the rest
// of the day's budget instead of feeding a 429 retry storm.
const twelveDataDailyLimit = 640 // 800 free-tier × 0.8 safety margin

// twelveDataLimiter is the shared token bucket for all Twelve Data calls.
var twelveDataLimiter = newTokenBucket(twelveDataLimit, time.Minute)

// twelveDataDaily is the shared UTC-day credit budget for all Twelve Data calls.
var twelveDataDaily = newDailyBudget(twelveDataDailyLimit)

// tokenBucket is a simple thread-safe token-bucket rate limiter.
// capacity tokens are available; they refill at capacity per window.
type tokenBucket struct {
	mu       sync.Mutex
	capacity float64
	tokens   float64
	window   time.Duration
	last     time.Time
	waiters  int
}

func newTokenBucket(capacity int, window time.Duration) *tokenBucket {
	return &tokenBucket{
		capacity: float64(capacity),
		tokens:   float64(capacity),
		window:   window,
		last:     time.Now(),
	}
}

// refill tops up tokens based on elapsed time since the last refill.
func (b *tokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.last)
	if elapsed <= 0 {
		return
	}
	rate := b.capacity / float64(b.window.Nanoseconds())
	b.tokens += rate * float64(elapsed.Nanoseconds())
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
}

// Wait blocks until a token is available (or ctx is cancelled), then consumes one.
func (b *tokenBucket) Wait(ctx context.Context) error {
	for {
		b.mu.Lock()
		b.refill()
		if b.tokens >= 1 {
			b.tokens -= 1
			b.mu.Unlock()
			return nil
		}
		// Estimate how long until the next token is available.
		// tokensToWait = 1 - tokens; time = tokensToWait / rate.
		rate := b.capacity / float64(b.window.Nanoseconds())
		need := 1 - b.tokens
		if need <= 0 {
			need = 1
		}
		wait := time.Duration(need / rate)
		if wait < time.Second {
			wait = time.Second
		}
		b.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// dailyBudget tracks API credits consumed within the current UTC day and
// fails fast once the allowance is exhausted (no waiting — the caller should
// mark its data RATE_LIMITED and wait for the next UTC day).
type dailyBudget struct {
	mu        sync.Mutex
	limit     int
	used      int
	day       int64 // UTC day number (time.Now().UTC().Unix() / 86400)
	exhausted bool
}

func newDailyBudget(limit int) *dailyBudget {
	return &dailyBudget{limit: limit, day: time.Now().UTC().Unix() / 86400}
}

// roll advances the budget to the current UTC day (resets state on day change).
func (d *dailyBudget) roll() {
	today := time.Now().UTC().Unix() / 86400
	if today != d.day {
		d.day = today
		d.used = 0
		d.exhausted = false
	}
}

// tryConsume records one credit if the daily allowance has budget left.
// Returns false when the day's budget is spent (caller must NOT issue the
// HTTP request; treat it as a rate-limit condition).
func (d *dailyBudget) tryConsume() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll()
	if d.exhausted {
		return false
	}
	d.used++
	if d.used >= d.limit {
		d.exhausted = true
	}
	return true
}

// used reports credits consumed today (test introspection).
func (d *dailyBudget) usedToday() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll()
	return d.used
}

// isExhausted reports whether today's budget is spent (test introspection).
func (d *dailyBudget) isExhausted() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll()
	return d.exhausted
}

// acquireTwelveDataCredit blocks until one per-minute credit is available,
// then consumes one daily credit. It returns an error WITHOUT blocking the
// rest of the day once the UTC-day budget is exhausted (the caller marks its
// data RATE_LIMITED — strategies fail to NO-TRADE — and tomorrow's budget
// resets automatically at UTC midnight).
//
// All Twelve Data HTTP requests must call this before issuing the request so
// the combined call volume from both providers stays under both free-tier caps.
func acquireTwelveDataCredit(ctx context.Context) error {
	if !twelveDataDaily.tryConsume() {
		return ErrTwelveDataDailyBudgetSpent
	}
	return twelveDataLimiter.Wait(ctx)
}

// ErrTwelveDataDailyBudgetSpent is returned once the UTC-day Twelve Data
// credit budget is exhausted; callers should mark data RATE_LIMITED/STALE and
// retry after the next UTC midnight.
var ErrTwelveDataDailyBudgetSpent error = &twelveDataBudgetError{}

// twelveDataBudgetError is a typed, retryable-later error for the daily budget.
type twelveDataBudgetError struct{}

func (e *twelveDataBudgetError) Error() string {
	return "rate_limited: Twelve Data daily credit budget spent (resets next UTC midnight)"
}
