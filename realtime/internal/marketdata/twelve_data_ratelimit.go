// Package marketdata — shared TwelveData API rate limiter.
//
// Twelve Data's free tier allows only 8 API credits per minute. The engine has
// TWO consumers of the same API key: the DXY provider (6 component symbols) and
// the macro-asset provider (VIX/BTC/Oil/EURUSD/USDCHF). Firing all of those as
// separate calls in the same minute blew past the 8/min cap, so every fetch
// returned HTTP 429 and DXY — a MANDATORY strategy pillar — never became
// available, which forced the engine into NO-TRADE for all swing strategies.
//
// Fix (two parts):
//   1. Both providers now batch their symbols into a SINGLE multi-symbol call
//      (1 credit each instead of N), and
//   2. Every Twelve Data HTTP request passes through this shared token-bucket
//      limiter so the combined rate stays safely under the free-tier ceiling.
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

// twelveDataLimiter is the shared token bucket for all Twelve Data calls.
var twelveDataLimiter = newTokenBucket(twelveDataLimit, time.Minute)

// tokenBucket is a simple thread-safe token-bucket rate limiter.
// capacity tokens are available; they refill at capacity per window.
type tokenBucket struct {
	mu        sync.Mutex
	capacity  float64
	tokens    float64
	window    time.Duration
	last      time.Time
	waiters   int
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

// acquire blocks until one Twelve Data API credit is available, then consumes it.
// All Twelve Data HTTP requests must call this before issuing the request so the
// combined call volume from both providers stays under the free-tier cap.
func acquireTwelveDataCredit(ctx context.Context) error {
	return twelveDataLimiter.Wait(ctx)
}
