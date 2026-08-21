package main

import (
	"fmt"
	"github.com/predictatrade/realtime/internal/gates"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func main() {
	fmt.Println("=== STRATEGY CONFIG AUDIT ===")
	
	strategies := []strategy.Strategy{
		strategy.NewStandardScalping(),
		strategy.NewUltraScalping(),
		strategy.NewStandardSwing(),
		strategy.NewTrendSwing(),
	}
	
	for _, s := range strategies {
		cfg := strategy.GetStrategyConfig(s)
		fmt.Printf("\n%s:\n", s.ID())
		fmt.Printf("  SL=%.1f*ATR TP1=%.1f TP2=%.1f TP3=%.1f\n", cfg.ATRMultiplierSL, cfg.ATRMultiplierTP1, cfg.ATRMultiplierTP2, cfg.ATRMultiplierTP3)
		fmt.Printf("  MinRR=%.1f MinADX=%.0f MaxSpread=%.1f MaxSlippage=%dpts\n", cfg.MinRR, cfg.MinADX, cfg.MaxSpreadPips, cfg.MaxSlippagePoints)
		rr1 := cfg.ATRMultiplierTP1 / cfg.ATRMultiplierSL
		rr2 := cfg.ATRMultiplierTP2 / cfg.ATRMultiplierSL
		rr3 := cfg.ATRMultiplierTP3 / cfg.ATRMultiplierSL
		fmt.Printf("  R:R at TP1=%.2f TP2=%.2f TP3=%.2f\n", rr1, rr2, rr3)
		if cfg.MinRR == 2.0 { fmt.Printf("  PASS: MinRR=2.0\n") } else { fmt.Printf("  FAIL: MinRR=%.1f (expected 2.0)\n", cfg.MinRR) }
		hvFound := false
		for _, r := range cfg.AcceptedRegimes { if r == types.RegimeHighVolatility { hvFound = true; break } }
		if hvFound { fmt.Printf("  PASS: HIGH_VOLATILITY accepted\n") } else { fmt.Printf("  FAIL: HIGH_VOLATILITY not accepted\n") }
	}

	fmt.Println("\n=== CANDIDATE MICROPROFIT GEOMETRY ===")
	for stratID, cc := range strategy.DefaultCandidateGeometry() {
		fmt.Printf("\n%s: SL=%.1f TP1=%.1f TP2=%.1f TP3=%.1f R:R1=%.2f\n", stratID, cc.ATRMultiplierSL, cc.ATRMultiplierTP1, cc.ATRMultiplierTP2, cc.ATRMultiplierTP3, cc.ATRMultiplierTP1/cc.ATRMultiplierSL)
	}

	fmt.Println("\n=== CAPITAL PROTECTION ===")
	capCfg := gates.DefaultCapitalProtectionConfig()
	fmt.Printf("MaxDailyLossPct: %.1f (expect 5.0)\n", capCfg.MaxDailyLossPct)
	fmt.Printf("MaxPerTradeRiskPct: %.1f (expect 1.0)\n", capCfg.MaxPerTradeRiskPct)
	fmt.Printf("MaxTotalOpenRiskPct: %.1f (expect 5.0)\n", capCfg.MaxTotalOpenRiskPct)
	fmt.Printf("MinRR: %.1f (expect 2.0)\n", capCfg.MinRR)
	
	if capCfg.MaxDailyLossPct == 5.0 { fmt.Println("  PASS: DailyLoss=5%") } else { fmt.Println("  FAIL: DailyLoss") }
	if capCfg.MaxPerTradeRiskPct == 1.0 { fmt.Println("  PASS: PerTrade=1%") } else { fmt.Println("  FAIL: PerTrade") }
	if capCfg.MinRR == 2.0 { fmt.Println("  PASS: MinRR=2.0") } else { fmt.Println("  FAIL: MinRR") }

	si := gates.DefaultXAUSymbolInfo()
	lots := gates.CalculatePositionSize(decimal.NewFromFloat(10000), decimal.NewFromFloat(2.0), si)
	fmt.Printf("\nPositionSize(10000, 2.0): %s lots (expect 0.50)\n", lots.String())
	lots0 := gates.CalculatePositionSize(decimal.NewFromFloat(10000), decimal.Zero, si)
	fmt.Printf("PositionSize(10000, 0): %s (expect 0)\n", lots0.String())

	schedule := gates.BuildPartialCloseSchedule(decimal.NewFromFloat(2400), decimal.NewFromFloat(2404), decimal.NewFromFloat(2408), decimal.NewFromFloat(2.0))
	fmt.Printf("\nPartial Close Schedule:\n")
	for _, s := range schedule {
		fmt.Printf("  Stage=%d Close%%=%.0f SL=%s TrailATR=%.1f\n", s.Stage, s.ClosePercent, s.NewStopLoss.String(), s.TrailATRMultiplier)
	}
	if len(schedule) == 3 && schedule[0].ClosePercent == 50 && schedule[1].ClosePercent == 30 && schedule[2].ClosePercent == 20 {
		fmt.Println("  PASS: 50/30/20 schedule")
	} else {
		fmt.Println("  FAIL: schedule not 50/30/20")
	}

	swapResult := gates.CheckSwapProtection(types.DirectionBuy, decimal.NewFromFloat(2400), decimal.NewFromFloat(2396), decimal.NewFromFloat(2404), decimal.NewFromFloat(-0.5), decimal.NewFromFloat(1.0), 3, false)
	fmt.Printf("\nSwapProtection(neg,swing): Allowed=%v Reason=%s\n", swapResult.Allowed, swapResult.ReasonCode)
	if !swapResult.Allowed { fmt.Println("  PASS: rejects negative swap with low R:R") } else { fmt.Println("  FAIL: should reject") }

	slipResult := gates.CheckSpreadSlippage(30, 25)
	fmt.Printf("SpreadSlippage(30>25): Allowed=%v\n", slipResult.Allowed)
	if !slipResult.Allowed { fmt.Println("  PASS: rejects spread > max") } else { fmt.Println("  FAIL: should reject") }

	slipResult2 := gates.CheckSpreadSlippage(20, 25)
	fmt.Printf("SpreadSlippage(20<25): Allowed=%v\n", slipResult2.Allowed)
	if slipResult2.Allowed { fmt.Println("  PASS: allows spread < max") } else { fmt.Println("  FAIL: should allow") }
}
