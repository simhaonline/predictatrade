package features

import (
	"math"
	"time"

	"github.com/shopspring/decimal"
)

func currentTime() time.Time { return time.Now().UTC() }

func goSqrt(f float64) float64 { return math.Sqrt(f) }

// SafeDecimalDiv divides safely, returning zero on division by zero.
func SafeDecimalDiv(a, b decimal.Decimal) decimal.Decimal {
	if b.IsZero() {
		return decimal.Zero
	}
	return a.Div(b)
}
