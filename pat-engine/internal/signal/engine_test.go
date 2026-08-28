package signal

import (
	"testing"

	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

func ultraBullState() *types.MarketState {
	return &types.MarketState{
		Symbol:       "XAUUSD",
		Timeframe:    types.TFM1,
		CurrentPrice: 2000.00,
		H1Close:      1998.00,
		VWAP:         1999.90,
		Spread:       0.20,
		ATR:          1.00,
		Indicators: types.Indicators{
			EMA9: 2001.0, EMA21: 2000.5, EMA50: 1999.0, EMA100: 1990.0, EMA200: 1980.0, SMA200: 1980.0,
			ADX: 30, ADXPlusDI: 25, ADXMinusDI: 10,
			RSI: 55, MACDMain: 0.5, MACDSignal: 0.3, OsMA: 0.2,
			StochMain: 0.6, StochSignal: 0.4, BollUpper: 2010, BollLower: 1990,
		},
		MTFScore: 15,
		Regime:   "TRENDING_BULLISH",
		Session:  types.Session{CurrentSession: "LONDON"},
		Quality:  "AUTHORITATIVE",
		Candle:   types.Candle{IsBullish: true, IsDisplacement: true},
	}
}

func TestUltraScalpingExecutable(t *testing.T) {
	st := strategy.Must("ULTRA_SCALPING")
	cfg := config.DefaultUltraScalping()
	pol := &broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2}

	d := Decide(ultraBullState(), st, cfg, pol)
	if !d.Signal.Executable {
		t.Fatalf("expected executable BUY, got BLOCKED reasons=%v", d.Reasons)
	}
	if d.Signal.Direction != types.DirBuy {
		t.Fatalf("expected BUY, got %s", d.Signal.Direction)
	}
	rr := (d.Signal.TP1 - d.Signal.EntryPrice) / (d.Signal.EntryPrice - d.Signal.StopLoss)
	if rr < cfg.MinRR-1e-6 {
		t.Fatalf("expected R:R >= %.2f, got %.2f", cfg.MinRR, rr)
	}
}

func TestBrokerScalpingForbiddenBlocksScalpers(t *testing.T) {
	st := strategy.Must("ULTRA_SCALPING")
	cfg := config.DefaultUltraScalping()
	pol := &broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: false, Digits: 2}

	d := Decide(ultraBullState(), st, cfg, pol)
	if d.Signal.Executable {
		t.Fatalf("broker forbids scalping; expected BLOCKED, got executable")
	}
	found := false
	for _, r := range d.Reasons {
		if r == "BROKER_SCALPING_NOT_ALLOWED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected BROKER_SCALPING_NOT_ALLOWED reason, got %v", d.Reasons)
	}
}

func TestHighSpreadBlocks(t *testing.T) {
	st := strategy.Must("ULTRA_SCALPING")
	cfg := config.DefaultUltraScalping()
	pol := &broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2}

	s := ultraBullState()
	s.Spread = 5.0 // far above ATR*0.4 gate
	d := Decide(s, st, cfg, pol)
	if d.Signal.Executable {
		t.Fatalf("expected HIGH_SPREAD block, got executable")
	}
}
