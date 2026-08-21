package backtest

import (
	"math"

	"github.com/shopspring/decimal"
)

// ComputeMetrics calculates performance metrics from closed trades.
func ComputeMetrics(trades []Trade, initialBalance decimal.Decimal) Metrics {
	m := Metrics{}
	m.TotalTrades = len(trades)
	m.FinalBalance = initialBalance

	if len(trades) == 0 {
		return m
	}

	var totalWin decimal.Decimal
	var totalLoss decimal.Decimal
	var grossProfit decimal.Decimal
	var grossLoss decimal.Decimal
	var peakBalance decimal.Decimal
	var maxDD decimal.Decimal
	var currentBalance = initialBalance
	var consecutiveWins int
	var consecutiveLosses int
	var totalHoldingBars int

	// For Sharpe: collect per-trade returns
	var returns []float64

	for _, t := range trades {
		currentBalance = currentBalance.Add(t.RealizedPnL)
		m.NetProfit = m.NetProfit.Add(t.RealizedPnL)

		if t.Direction == "BUY" {
			m.BuyCount++
		} else {
			m.SellCount++
		}

		totalHoldingBars += t.HoldingBars

		if t.RealizedPnL.IsPositive() {
			m.Wins++
			totalWin = totalWin.Add(t.RealizedPnL)
			grossProfit = grossProfit.Add(t.RealizedPnL)
			consecutiveWins++
			consecutiveLosses = 0
			if consecutiveWins > m.MaxConsecutiveWins {
				m.MaxConsecutiveWins = consecutiveWins
			}
		} else {
			m.Losses++
			totalLoss = totalLoss.Add(t.RealizedPnL)
			grossLoss = grossLoss.Add(t.RealizedPnL.Abs())
			consecutiveLosses++
			consecutiveWins = 0
			if consecutiveLosses > m.MaxConsecutiveLosses {
				m.MaxConsecutiveLosses = consecutiveLosses
			}
		}

		// Track best/worst
		if t.RealizedPnL.GreaterThan(m.BestTrade) {
			m.BestTrade = t.RealizedPnL
		}
		if t.RealizedPnL.LessThan(m.WorstTrade) {
			m.WorstTrade = t.RealizedPnL
		}

		// Track drawdown
		if currentBalance.GreaterThan(peakBalance) {
			peakBalance = currentBalance
		}
		dd := peakBalance.Sub(currentBalance)
		if dd.GreaterThan(maxDD) {
			maxDD = dd
		}

		// Collect return for Sharhe
		if !initialBalance.IsZero() {
			ret, _ := t.RealizedPnL.Div(initialBalance).Float64()
			returns = append(returns, ret)
		}
	}

	m.FinalBalance = currentBalance

	// Win rate
	if m.TotalTrades > 0 {
		m.WinRate = decimal.NewFromInt(int64(m.Wins)).Div(decimal.NewFromInt(int64(m.TotalTrades))).Mul(decimal.NewFromInt(100))
	}

	// Avg win/loss
	if m.Wins > 0 {
		m.AvgWin = totalWin.Div(decimal.NewFromInt(int64(m.Wins)))
	}
	if m.Losses > 0 {
		m.AvgLoss = totalLoss.Div(decimal.NewFromInt(int64(m.Losses)))
	}

	// Profit factor
	if !grossLoss.IsZero() {
		m.ProfitFactor = grossProfit.Div(grossLoss)
	}

	// Total return %
	if !initialBalance.IsZero() {
		m.TotalReturnPct = m.NetProfit.Div(initialBalance).Mul(decimal.NewFromInt(100))
	}

	// Max drawdown %
	if !peakBalance.IsZero() {
		m.MaxDrawdownPct = maxDD.Div(peakBalance).Mul(decimal.NewFromInt(100))
	}

	// Expectancy
	if m.TotalTrades > 0 {
		m.Expectancy = m.NetProfit.Div(decimal.NewFromInt(int64(m.TotalTrades)))
	}

	// Avg holding bars
	if m.TotalTrades > 0 {
		m.AvgHoldingBars = totalHoldingBars / m.TotalTrades
	}

	// Sharpe ratio (simplified: per-trade, not per-bar)
	if len(returns) > 1 {
		m.SharpeRatio = computeSharpe(returns)
		m.SortinoRatio = computeSortino(returns)
	}

	return m
}

func computeSharpe(returns []float64) decimal.Decimal {
	n := float64(len(returns))
	if n < 2 {
		return decimal.Zero
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / n

	var sqSum float64
	for _, r := range returns {
		diff := r - mean
		sqSum += diff * diff
	}
	stdDev := math.Sqrt(sqSum / (n - 1))

	if stdDev == 0 {
		return decimal.Zero
	}

	// Annualize: ~252 trading days, ~5 trades/day → ~1260 trades/year
	// Sharpe = mean / stdDev * sqrt(n)
	sharpe := mean / stdDev * math.Sqrt(n)
	return decimal.NewFromFloat(sharpe)
}

func computeSortino(returns []float64) decimal.Decimal {
	n := float64(len(returns))
	if n < 2 {
		return decimal.Zero
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / n

	var downsideSqSum float64
	downsideCount := 0
	for _, r := range returns {
		if r < 0 {
			downsideSqSum += r * r
			downsideCount++
		}
	}

	if downsideCount == 0 {
		return decimal.Zero
	}
	downsideDev := math.Sqrt(downsideSqSum / float64(downsideCount))
	if downsideDev == 0 {
		return decimal.Zero
	}

	sortino := mean / downsideDev * math.Sqrt(n)
	return decimal.NewFromFloat(sortino)
}
