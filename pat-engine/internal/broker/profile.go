package broker

import (
	"math"
	"os"
	"strconv"
	"time"
)

// ExecutionProfile is the broker-specific execution economics for one symbol. It is
// the single source of truth for transaction cost, margin and session math — ported
// from the old project's market.broker_execution_profiles + risk/sizing.go. Every
// field is needed so net R:R is total-cost aware (spread + commission + swap) and
// position sizing / margin are broker-exact.
type ExecutionProfile struct {
	Symbol          string  `json:"symbol"`
	Digits          int     `json:"digits"`
	TickSize        float64 `json:"tick_size"`
	TickValue       float64 `json:"tick_value"`   // account currency per 1 tick per lot
	ContractSize    float64 `json:"contract_size"` // units per lot (XAUUSD ~100)
	MinLot          float64 `json:"min_lot"`
	MaxLot          float64 `json:"max_lot"`
	LotStep         float64 `json:"lot_step"`
	StopLevel       int     `json:"stops_level"`  // min SL distance in points
	FreezeLevel     int     `json:"freeze_level"` // min distance to price to modify
	SwapLong        float64 `json:"swap_long"`    // points per lot per day
	SwapShort       float64 `json:"swap_short"`
	CommissionPerLot float64 `json:"commission_per_lot"` // account currency per lot (round-turn)
	TypicalSpread   float64 `json:"typical_spread"`      // points
	Leverage        float64 `json:"leverage"`
	TimezoneOffset  int     `json:"timezone_offset"`     // hours from UTC (broker server time)
	RolloverHour    int     `json:"rollover_hour"`       // broker-time hour swap is applied
	Sessions        []SessionWindow `json:"sessions"`
}

// SessionWindow is an active trading session in broker time.
type SessionWindow struct {
	Name    string `json:"name"`
	StartH  int    `json:"start_h"`
	EndH    int    `json:"end_h"`
	Overlap bool   `json:"overlap"`
}

// DefaultXAUUSDExecution returns an Equiti-like XAUUSD profile (broker server time
// UTC+4, per user). Override via env in production.
func DefaultXAUUSDExecution() ExecutionProfile {
	return ExecutionProfile{
		Symbol:           "XAUUSD",
		Digits:           2,
		TickSize:         0.01,
		TickValue:        1.0, // 1 point ($0.01) * 100 oz = $1 per lot
		ContractSize:     100,
		MinLot:           0.01,
		MaxLot:           50,
		LotStep:          0.01,
		StopLevel:        0,
		FreezeLevel:      0,
		SwapLong:         -1.5,
		SwapShort:        -1.5,
		CommissionPerLot: 7.0,
		TypicalSpread:    20, // 0.20 at 2 digits
		Leverage:         500,
		TimezoneOffset:   4,
		RolloverHour:     0,
		Sessions:         DefaultSessions(),
	}
}

// DefaultSessions are the FX sessions in broker time (UTC+offset applied externally).
func DefaultSessions() []SessionWindow {
	return []SessionWindow{
		{Name: "TOKYO", StartH: 0, EndH: 7, Overlap: false},
		{Name: "LONDON", StartH: 7, EndH: 13, Overlap: false},
		{Name: "OVERLAP", StartH: 13, EndH: 17, Overlap: true},
		{Name: "NEW_YORK", StartH: 17, EndH: 22, Overlap: false},
		{Name: "SYDNEY", StartH: 22, EndH: 24, Overlap: false},
	}
}

// PointSize is the price increment of one point (tick).
func (e ExecutionProfile) PointSize() float64 {
	if e.TickSize > 0 {
		return e.TickSize
	}
	return math.Pow10(-e.Digits)
}

// RoundToDigits rounds a price to the broker's decimal places (P1-001).
func (e ExecutionProfile) RoundToDigits(p float64) float64 {
	mult := math.Pow10(e.Digits)
	return math.Round(p*mult) / mult
}

// Points converts a price distance to points.
func (e ExecutionProfile) Points(priceDist float64) float64 {
	if e.TickSize <= 0 {
		return priceDist * math.Pow10(e.Digits)
	}
	return priceDist / e.TickSize
}

// CurrencyPerPoint is account currency per 1 price point per 1 lot.
func (e ExecutionProfile) CurrencyPerPoint() float64 {
	if e.TickValue > 0 && e.TickSize > 0 {
		return e.TickValue / e.TickSize
	}
	return e.TickValue
}

// CommissionPrice returns the commission cost expressed in PRICE units for a lot
// (P&L per lot = priceMove * ContractSize, so commission/lot offsets that move).
func (e ExecutionProfile) CommissionPrice(lot float64) float64 {
	if e.ContractSize <= 0 {
		return 0
	}
	return e.CommissionPerLot * lot / e.ContractSize
}

// SwapPrice returns the swap cost in PRICE units for a side over n days (per lot).
func (e ExecutionProfile) SwapPrice(side string, lot float64, days int) float64 {
	if days <= 0 || e.ContractSize <= 0 {
		return 0
	}
	s := e.SwapLong
	if side == "SELL" {
		s = e.SwapShort
	}
	return s * lot * float64(days) / e.ContractSize
}

// RequiredMargin returns the margin needed for a lot at price (leverage from profile).
func (e ExecutionProfile) RequiredMargin(lot, price float64) float64 {
	if e.Leverage <= 0 {
		return 0
	}
	return lot * e.ContractSize * price / e.Leverage
}

// MaxAffordableLot returns the largest lot whose margin fits free margin.
func (e ExecutionProfile) MaxAffordableLot(freeMargin, price float64) float64 {
	if e.Leverage <= 0 || price <= 0 {
		return 0
	}
	return freeMargin * e.Leverage / (e.ContractSize * price)
}

// RoundLot clamps/quantises a lot to [MinLot, MaxLot] on the broker step.
func (e ExecutionProfile) RoundLot(lot float64) float64 {
	if e.MinLot > 0 && lot < e.MinLot {
		lot = e.MinLot
	}
	if e.MaxLot > 0 && lot > e.MaxLot {
		lot = e.MaxLot
	}
	if e.LotStep > 0 {
		lot = math.Floor(lot/e.LotStep) * e.LotStep
	}
	if lot < 0 {
		lot = 0
	}
	return lot
}

// BrokerNow returns t in broker server time (FixedZone from profile offset).
func (e ExecutionProfile) BrokerNow(t time.Time) time.Time {
	loc := time.FixedZone("broker", e.TimezoneOffset*3600)
	return t.In(loc)
}

// LoadExecutionFromEnv overrides profile fields from env (production-grade config).
func (e ExecutionProfile) LoadExecutionFromEnv() ExecutionProfile {
	if v := os.Getenv("BROKER_DIGITS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			e.Digits = n
		}
	}
	if v := os.Getenv("BROKER_LEVERAGE"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			e.Leverage = n
		}
	}
	if v := os.Getenv("BROKER_COMMISSION_PER_LOT"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			e.CommissionPerLot = n
		}
	}
	if v := os.Getenv("BROKER_TYPICAL_SPREAD"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			e.TypicalSpread = n
		}
	}
	if v := os.Getenv("BROKER_TZ_OFFSET"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			e.TimezoneOffset = n
		}
	}
	if v := os.Getenv("BROKER_CONTRACT_SIZE"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			e.ContractSize = n
		}
	}
	return e
}
