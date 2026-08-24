// Forward-test edge statistics (EV1-EV3).
//
// EdgeValidation requires, per strategy over the last N=50 closed trades:
//   - profit factor >= 1.2
//   - expectancy    >= 0.2R
//   - sample size   >= 50
//
// Strategies that do not meet the bar still publish signals but are forced
// to SignalClass=ADVISORY with reason edge_unproven. Empty history →
// ADVISORY (never a hard veto; never fabricated evidence).
package risk

// TradeRecord is one closed trade used for edge statistics.
type TradeRecord struct {
	StrategyID string  `json:"strategy_id"`
	Direction  string  `json:"direction"` // BUY | SELL
	PnL        float64 `json:"pnl"`
	EntryPrice float64 `json:"entry_price"`
	StopLoss   float64 `json:"stop_loss"`
	LotSize    float64 `json:"lot_size"`
}

// EdgeStats is the computed forward-test edge summary for one strategy.
type EdgeStats struct {
	SampleSize        int     `json:"sample_size"`
	Wins              int     `json:"wins"`
	Losses            int     `json:"losses"`
	ProfitFactor      float64 `json:"profit_factor"`      // grossWin / |grossLoss|
	ExpectancyR       float64 `json:"expectancy_r"`       // mean R over R-computable trades
	RComputableCount  int     `json:"r_computable_count"` // trades with entry/sl/lot data
	ShortExpectancyR  float64 `json:"short_expectancy_r"` // expectancy over SELL trades only
	ShortSampleSize   int     `json:"short_sample_size"`
	Proven            bool    `json:"proven"`
}

// ComputeEdgeStats derives EdgeStats from closed-trade records.
// Trades without valid geometry (entry/SL/lot) contribute P&L to the
// profit factor but not to R-based expectancy.
func ComputeEdgeStats(trades []TradeRecord) EdgeStats {
	var s EdgeStats
	s.SampleSize = len(trades)

	var grossWin, grossLoss float64
	var rSum, shortRSum float64
	rCount := 0
	shortCount := 0

	for _, t := range trades {
		if t.PnL > 0 {
			s.Wins++
			grossWin += t.PnL
		} else if t.PnL < 0 {
			s.Losses++
			grossLoss += -t.PnL
		}

		r := tradeR(t)
		if computable := t.EntryPrice > 0 && t.StopLoss > 0 && t.LotSize > 0; computable {
			rCount++
			rSum += r
			if t.Direction == "SELL" {
				shortCount++
				shortRSum += r
			}
		}
	}

	if grossLoss > 0 {
		s.ProfitFactor = grossWin / grossLoss
	} else if grossWin > 0 {
		s.ProfitFactor = float64(s.Wins) // all-win: cap PF at win count (no infinite values)
	}
	if rCount > 0 {
		s.ExpectancyR = rSum / float64(rCount)
		s.RComputableCount = rCount
	}
	if shortCount > 0 {
		s.ShortExpectancyR = shortRSum / float64(shortCount)
		s.ShortSampleSize = shortCount
	}
	return s
}

// IsProven applies the EV1-EV3 thresholds.
func (s EdgeStats) IsProven(minProfitFactor, minExpectancyR float64, minSampleSize int) bool {
	if s.SampleSize < minSampleSize {
		return false
	}
	if s.RComputableCount < minSampleSize {
		// Expectancy must be evidenced over the required sample, not a subset.
		return false
	}
	return s.ProfitFactor >= minProfitFactor && s.ExpectancyR >= minExpectancyR
}

// tradeR converts one closed trade into an R multiple using its recorded
// geometry: R = pnl / risk$ where risk$ = |entry-SL| × lot × 100oz.
func tradeR(t TradeRecord) float64 {
	risk := absF(t.EntryPrice-t.StopLoss) * t.LotSize * DefaultContractSize
	if risk <= 0 {
		return 0
	}
	return t.PnL / risk
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
