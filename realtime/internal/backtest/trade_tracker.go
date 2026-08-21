package backtest

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// TradeTracker manages open positions and records closed trades.
type TradeTracker struct {
	openPositions []*OpenPosition
	closedTrades  []Trade
	balance       decimal.Decimal
	initialBalance decimal.Decimal

	// Risk state
	consecutiveLosses int
	dailyPnL         decimal.Decimal
	currentDay       time.Time
	recoveryMode      bool
	recoverySizeMult  decimal.Decimal

	// Config
	config BacktestConfig
}

// NewTradeTracker creates a new trade tracker.
func NewTradeTracker(config BacktestConfig) *TradeTracker {
	return &TradeTracker{
		balance:        config.InitialBalance,
		initialBalance: config.InitialBalance,
		config:        config,
		recoverySizeMult: decimal.NewFromInt(1),
	}
}

// OpenPosition opens a new position from a strategy signal.
func (t *TradeTracker) OpenPosition(pos *OpenPosition) {
	t.openPositions = append(t.openPositions, pos)
}

// OpenCount returns the number of currently open positions.
func (t *TradeTracker) OpenCount() int {
	return len(t.openPositions)
}

// TotalExposure returns the total exposure across all open positions.
func (t *TradeTracker) TotalExposure() decimal.Decimal {
	total := decimal.Zero
	for _, p := range t.openPositions {
		total = total.Add(p.EntryPrice.Mul(p.Size))
	}
	return total
}

// ConsecutiveLosses returns the current consecutive loss count.
func (t *TradeTracker) ConsecutiveLosses() int {
	return t.consecutiveLosses
}

// IsRecoveryMode returns whether recovery mode is active.
func (t *TradeTracker) IsRecoveryMode() bool {
	return t.recoveryMode
}

// RecoverySizeMultiplier returns the current position size multiplier.
func (t *TradeTracker) RecoverySizeMultiplier() decimal.Decimal {
	return t.recoverySizeMult
}

// DailyLossPct returns the current daily loss as a percentage.
func (t *TradeTracker) DailyLossPct() decimal.Decimal {
	if t.initialBalance.IsZero() {
		return decimal.Zero
	}
	return t.dailyPnL.Div(t.initialBalance).Mul(decimal.NewFromInt(100))
}

// CheckDayRollover resets daily PnL counters on a new trading day.
func (t *TradeTracker) CheckDayRollover(ts time.Time) {
	day := ts.UTC().Truncate(24 * time.Hour)
	if t.currentDay.IsZero() {
		t.currentDay = day
	} else if !day.Equal(t.currentDay) {
		t.currentDay = day
		t.dailyPnL = decimal.Zero
	}
}

// UpdatePositions checks all open positions against the current candle
// for SL/TP hits, trailing stops, and break-even moves.
// Returns the positions that were closed this bar.
func (t *TradeTracker) UpdatePositions(candle *types.Candle, barIdx int) []Trade {
	var newlyClosed []Trade
	var stillOpen []*OpenPosition

	for _, pos := range t.openPositions {
		closed := t.checkPosition(pos, candle, barIdx)
		if closed != nil {
			t.closedTrades = append(t.closedTrades, *closed)
			newlyClosed = append(newlyClosed, *closed)
			t.balance = t.balance.Add(closed.RealizedPnL)
			t.dailyPnL = t.dailyPnL.Add(closed.RealizedPnL)

			// Update consecutive losses
			if closed.RealizedPnL.IsNegative() {
				t.consecutiveLosses++
			} else if closed.RealizedPnL.IsPositive() {
				t.consecutiveLosses = 0
				if t.recoveryMode {
					// Exit recovery after a win
					t.recoveryMode = false
					t.recoverySizeMult = decimal.NewFromInt(1)
				}
			}
		} else {
			// Apply trailing stop and break-even
			t.applyTradeManagement(pos, candle)
			stillOpen = append(stillOpen, pos)
		}
	}

	t.openPositions = stillOpen
	return newlyClosed
}

// checkPosition checks if a position's SL or TP was hit by the candle.
func (t *TradeTracker) checkPosition(pos *OpenPosition, candle *types.Candle, barIdx int) *Trade {
	var exitPrice decimal.Decimal
	var exitReason string
	hit := false

	if pos.Direction == types.DirectionBuy {
		// Check SL first (conservative) or TP first
		if t.config.ConservativeSLTP {
			// SL hit? (guard against zero SL)
			if !pos.StopLoss.IsZero() && candle.Low.LessThanOrEqual(pos.StopLoss) {
				exitPrice = pos.StopLoss
				exitReason = "SL"
				hit = true
			} else if !pos.TP1.IsZero() && candle.High.GreaterThanOrEqual(pos.TP1) && !pos.TP1Hit {
				exitPrice = pos.TP1
				exitReason = "TP1"
				pos.TP1Hit = true
				hit = true
			} else if !pos.TP2.IsZero() && candle.High.GreaterThanOrEqual(pos.TP2) && pos.TP1Hit && !pos.TP2Hit {
				exitPrice = pos.TP2
				exitReason = "TP2"
				pos.TP2Hit = true
				hit = true
			} else if !pos.TP3.IsZero() && candle.High.GreaterThanOrEqual(pos.TP3) && pos.TP2Hit && !pos.TP3Hit {
				exitPrice = pos.TP3
				exitReason = "TP3"
				pos.TP3Hit = true
				hit = true
			}
		} else {
			// Non-conservative: check TP first
			if !pos.TP1.IsZero() && candle.High.GreaterThanOrEqual(pos.TP1) && !pos.TP1Hit {
				exitPrice = pos.TP1
				exitReason = "TP1"
				pos.TP1Hit = true
				hit = true
			} else if candle.Low.LessThanOrEqual(pos.StopLoss) {
				exitPrice = pos.StopLoss
				exitReason = "SL"
				hit = true
			}
		}
	} else { // SELL
		if t.config.ConservativeSLTP {
			// SL hit? (for sell, SL is above) — guard against zero SL
			if !pos.StopLoss.IsZero() && candle.High.GreaterThanOrEqual(pos.StopLoss) {
				exitPrice = pos.StopLoss
				exitReason = "SL"
				hit = true
			} else if !pos.TP1.IsZero() && candle.Low.LessThanOrEqual(pos.TP1) && !pos.TP1Hit {
				exitPrice = pos.TP1
				exitReason = "TP1"
				pos.TP1Hit = true
				hit = true
			} else if !pos.TP2.IsZero() && candle.Low.LessThanOrEqual(pos.TP2) && pos.TP1Hit && !pos.TP2Hit {
				exitPrice = pos.TP2
				exitReason = "TP2"
				pos.TP2Hit = true
				hit = true
			} else if !pos.TP3.IsZero() && candle.Low.LessThanOrEqual(pos.TP3) && pos.TP2Hit && !pos.TP3Hit {
				exitPrice = pos.TP3
				exitReason = "TP3"
				pos.TP3Hit = true
				hit = true
			}
		} else {
			if !pos.TP1.IsZero() && candle.Low.LessThanOrEqual(pos.TP1) && !pos.TP1Hit {
				exitPrice = pos.TP1
				exitReason = "TP1"
				pos.TP1Hit = true
				hit = true
			} else if candle.High.GreaterThanOrEqual(pos.StopLoss) {
				exitPrice = pos.StopLoss
				exitReason = "SL"
				hit = true
			}
		}
	}

	if !hit {
		return nil
	}

	// Calculate PnL
	var pnl decimal.Decimal
	var risk decimal.Decimal

	if pos.Direction == types.DirectionBuy {
		pnl = exitPrice.Sub(pos.EntryPrice).Mul(pos.Size).Mul(t.config.ContractSize)
		risk = pos.EntryPrice.Sub(pos.OriginalSL)
	} else {
		pnl = pos.EntryPrice.Sub(exitPrice).Mul(pos.Size).Mul(t.config.ContractSize)
		risk = pos.OriginalSL.Sub(pos.EntryPrice)
	}

	// Subtract costs
	spreadCost := t.config.Spread.Mul(pos.Size)
	commissionCost := t.config.Commission.Mul(pos.Size)

	// Swap cost: charge per overnight holding period
	// Each day held overnight = 1 swap charge. Triple swap on the configured day.
	swapCost := decimal.Zero
	if !t.config.SwapPerLotPerDay.IsZero() {
		holdingHours := candle.Time.Sub(pos.EntryTime).Hours()
		overnightDays := int(holdingHours / 24.0)
		if overnightDays > 0 {
			swapCost = t.config.SwapPerLotPerDay.Mul(pos.Size).Mul(decimal.NewFromInt(int64(overnightDays)))
		}
	}

	totalCost := spreadCost.Add(commissionCost).Add(swapCost)
	pnl = pnl.Sub(totalCost)

	var realizedR decimal.Decimal
	if !risk.IsZero() {
		if pos.Direction == types.DirectionBuy {
			realizedR = exitPrice.Sub(pos.EntryPrice).Div(risk)
		} else {
			realizedR = pos.EntryPrice.Sub(exitPrice).Div(risk)
		}
	}

	return &Trade{
		TradeID:        pos.TradeID,
		StrategyID:     pos.StrategyID,
		Direction:      pos.Direction,
		EntryPrice:     pos.EntryPrice,
		ExitPrice:      exitPrice,
		StopLoss:       pos.StopLoss,
		TP1:            pos.TP1,
		TP2:            pos.TP2,
		TP3:            pos.TP3,
		EntryTime:      pos.EntryTime,
		ExitTime:       candle.Time,
		ExitReason:     exitReason,
		Size:           pos.Size,
		RealizedPnL:     pnl,
		RealizedR:      realizedR,
		SpreadCost:     spreadCost,
		CommissionCost: commissionCost,
		SwapCost:       swapCost,
		Regime:         pos.Regime,
		Session:        pos.Session,
		RawScore:       pos.RawScore,
		EntryBarIdx:    pos.EntryBarIdx,
		ExitBarIdx:     barIdx,
		HoldingBars:    barIdx - pos.EntryBarIdx,
	}
}

// applyTradeManagement applies trailing stop and break-even adjustments.
func (t *TradeTracker) applyTradeManagement(pos *OpenPosition, candle *types.Candle) {
	if !t.config.TrailingStopEnabled && !t.config.BreakEvenEnabled {
		return
	}

	if pos.Direction == types.DirectionBuy {
		// Break-even: if price has moved 1R in favor, move SL to entry
		if t.config.BreakEvenEnabled && !pos.BreakEvenSet {
			risk := pos.EntryPrice.Sub(pos.OriginalSL)
			if risk.IsPositive() && candle.Close.Sub(pos.EntryPrice).GreaterThanOrEqual(risk.Mul(t.config.BreakEvenTriggerR)) {
				pos.StopLoss = pos.EntryPrice
				pos.BreakEvenSet = true
			}
		}
		// Trailing stop: move SL to close - ATR*mult (simplified: use candle range)
		if t.config.TrailingStopEnabled {
			candleRange := candle.High.Sub(candle.Low)
			newSL := candle.Close.Sub(candleRange.Mul(t.config.TrailingATRMult))
			if newSL.GreaterThan(pos.StopLoss) {
				pos.StopLoss = newSL
			}
		}
	} else {
		// SELL
		if t.config.BreakEvenEnabled && !pos.BreakEvenSet {
			risk := pos.OriginalSL.Sub(pos.EntryPrice)
			if risk.IsPositive() && pos.EntryPrice.Sub(candle.Close).GreaterThanOrEqual(risk.Mul(t.config.BreakEvenTriggerR)) {
				pos.StopLoss = pos.EntryPrice
				pos.BreakEvenSet = true
			}
		}
		if t.config.TrailingStopEnabled {
			candleRange := candle.High.Sub(candle.Low)
			newSL := candle.Close.Add(candleRange.Mul(t.config.TrailingATRMult))
			if newSL.LessThan(pos.StopLoss) {
				pos.StopLoss = newSL
			}
		}
	}
}

// CloseAllPositions force-closes all open positions at candle close.
func (t *TradeTracker) CloseAllPositions(candle *types.Candle, barIdx int) []Trade {
	var closed []Trade
	for _, pos := range t.openPositions {
		exitPrice := candle.Close
		var pnl decimal.Decimal
		var risk decimal.Decimal

		if pos.Direction == types.DirectionBuy {
			pnl = exitPrice.Sub(pos.EntryPrice).Mul(pos.Size).Mul(t.config.ContractSize)
			risk = pos.EntryPrice.Sub(pos.OriginalSL)
		} else {
			pnl = pos.EntryPrice.Sub(exitPrice).Mul(pos.Size).Mul(t.config.ContractSize)
			risk = pos.OriginalSL.Sub(pos.EntryPrice)
		}

		spreadCost := t.config.Spread.Mul(pos.Size)
		commissionCost := t.config.Commission.Mul(pos.Size)

		// Swap cost for EOD close
		swapCost := decimal.Zero
		if !t.config.SwapPerLotPerDay.IsZero() {
			holdingHours := candle.Time.Sub(pos.EntryTime).Hours()
			overnightDays := int(holdingHours / 24.0)
			if overnightDays > 0 {
				swapCost = t.config.SwapPerLotPerDay.Mul(pos.Size).Mul(decimal.NewFromInt(int64(overnightDays)))
			}
		}

		pnl = pnl.Sub(spreadCost.Add(commissionCost).Add(swapCost))

		var realizedR decimal.Decimal
		if !risk.IsZero() {
			if pos.Direction == types.DirectionBuy {
				realizedR = exitPrice.Sub(pos.EntryPrice).Div(risk)
			} else {
				realizedR = pos.EntryPrice.Sub(exitPrice).Div(risk)
			}
		}

		trade := Trade{
			TradeID:        pos.TradeID,
			StrategyID:     pos.StrategyID,
			Direction:      pos.Direction,
			EntryPrice:     pos.EntryPrice,
			ExitPrice:      exitPrice,
			StopLoss:       pos.StopLoss,
			TP1:            pos.TP1,
			TP2:            pos.TP2,
			TP3:            pos.TP3,
			EntryTime:      pos.EntryTime,
			ExitTime:       candle.Time,
			ExitReason:     "EOD",
			Size:           pos.Size,
			RealizedPnL:    pnl,
			RealizedR:      realizedR,
			SpreadCost:     spreadCost,
			CommissionCost: commissionCost,
			Regime:         pos.Regime,
			Session:       pos.Session,
			RawScore:       pos.RawScore,
			EntryBarIdx:    pos.EntryBarIdx,
			ExitBarIdx:     barIdx,
			HoldingBars:    barIdx - pos.EntryBarIdx,
		}
		t.closedTrades = append(t.closedTrades, trade)
		closed = append(closed, trade)
		t.balance = t.balance.Add(pnl)
	}
	t.openPositions = nil
	return closed
}

// ClosedTrades returns all closed trades.
func (t *TradeTracker) ClosedTrades() []Trade {
	return t.closedTrades
}

// Balance returns the current balance.
func (t *TradeTracker) Balance() decimal.Decimal {
	return t.balance
}
