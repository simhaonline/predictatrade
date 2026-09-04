package features

import (
	"math"
	"time"

	"github.com/shopspring/decimal"
)

// currentTime returns the engine's authoritative "now" in the broker session
// timezone. All time-of-day logic (sessions, ORB, news windows) must run on
// the broker clock the EAs' TimeCurrent() runs on — never on the host's UTC
// wall clock. BrokerLocation resolves the live Master-Node-reported offset
// first (see SetLiveBrokerOffset), so classification cannot drift from the
// broker's actual server time.
func currentTime() time.Time { return time.Now().In(BrokerLocation()) }

func goSqrt(f float64) float64 { return math.Sqrt(f) }

// SafeDecimalDiv divides safely, returning zero on division by zero.
func SafeDecimalDiv(a, b decimal.Decimal) decimal.Decimal {
	if b.IsZero() {
		return decimal.Zero
	}
	return a.Div(b)
}
