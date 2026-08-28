// Package provider is the live signal gateway: it ingests bars from a Windows
// Agent over HTTP, runs the SAME strategy + broker-policy + gate pipeline used by
// the backtest, and writes the resulting executable signal to the EA signal file
// (PAT_signals.txt, "SIGNAL|<json>" format the MQL EA already parses).
//
// It is deliberately zero-dependency (stdlib net/http) so the project never gets
// stuck on external module fetches.
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"pat-engine/internal/backtest"
	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/license"
	"pat-engine/internal/signal"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

// SignalDTO is the JSON payload written to the EA signal file (field names match
// the MQL ExtractJSONString keys exactly).
type SignalDTO struct {
	ID                    string  `json:"ID"`
	Direction             string  `json:"Direction"`
	Grade                 string  `json:"Grade"`
	StrategyID            string  `json:"StrategyID"`
	SignalClass           string  `json:"SignalClass"`
	EntryPrice            float64 `json:"EntryPrice"`
	StopLoss              float64 `json:"StopLoss"`
	TP1                   float64 `json:"TP1"`
	TP2                   float64 `json:"TP2"`
	TP3                   float64 `json:"TP3"`
	SuggestedLot          float64 `json:"SuggestedLot"`
	RawScore              float64 `json:"RawScore"`
	CalibratedProbability float64 `json:"CalibratedProbability"`
}

// Gateway keeps a rolling window of bars per symbol and emits signals.
type Gateway struct {
	mu       sync.Mutex
	bars     []backtest.Bar
	policy   *broker.BrokerPolicy
	lic      *license.License
	outPath  string
	lastID   string
	lastJSON string
	seq      int
}

// New creates a Gateway writing signals to outPath (PAT_signals.txt). It starts with
// a dev license that allows all strategies; call LoadLicense to enforce real
// entitlements.
func New(policy *broker.BrokerPolicy, outPath string) *Gateway {
	if policy == nil {
		policy = &broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2}
	}
	dev, _, _ := license.DevLicense(license.DefaultDevSecret, nil, nil)
	_ = os.MkdirAll(filepath.Dir(outPath), 0o755)
	return &Gateway{policy: policy, lic: dev, outPath: outPath}
}

// LoadLicense parses and installs a signed license token; non-entitled strategies
// are filtered out of signal selection.
func (g *Gateway) LoadLicense(token, secret string) error {
	l, err := license.Parse(token, secret)
	if err != nil {
		return err
	}
	g.lic = l
	return nil
}

// IngestBar adds a bar and, if an executable signal results, writes the signal file.
func (g *Gateway) IngestBar(b backtest.Bar) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.bars = append(g.bars, b)
	if len(g.bars) > 1000 {
		g.bars = g.bars[len(g.bars)-1000:]
	}
	state := backtest.StateFromBars(g.bars)
	if state == nil {
		return
	}

	best := g.bestExecutable(state)
	if best == nil {
		return
	}
	if best.ID == g.lastID {
		return // de-dupe: same signal already emitted
	}
	g.lastID = best.ID
	g.lastJSON = best.line
	if err := os.WriteFile(g.outPath, []byte(best.line+"\n"), 0o644); err != nil {
		fmt.Println("gateway: write signal file:", err)
	}
}

// Latest returns the last emitted signal line (for the GET /signal endpoint).
func (g *Gateway) Latest() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.lastJSON
}

type emit struct {
	ID   string
	line string
}

func (g *Gateway) bestExecutable(state *types.MarketState) *emit {
	cfgs := config.AllDefaults()
	strats := strategy.All()
	var best *emit
	var bestScore float64
	for _, st := range strats {
		// Entitlement gate: a license may only narrow what the broker policy allows.
		if g.lic != nil && !g.lic.AllowsStrategy(string(st.ID())) {
			continue
		}
		cfg := cfgs[string(st.ID())]
		d := signal.Decide(state, st, cfg, g.policy)
		if !d.Signal.Executable {
			continue
		}
		if d.Signal.RawScore <= bestScore {
			continue
		}
		bestScore = d.Signal.RawScore
		g.seq++
		best = &emit{ID: fmt.Sprintf("%s-%d", d.Signal.StrategyID, g.seq)}
		cls := signalClass(string(d.Signal.StrategyID))
		grade := gradeOf(d.Signal.RawScore)
		dto := SignalDTO{
			ID:          best.ID,
			Direction:   string(d.Signal.Direction),
			Grade:       grade,
			StrategyID:  string(d.Signal.StrategyID),
			SignalClass: cls,
			EntryPrice:  d.Signal.EntryPrice,
			StopLoss:    d.Signal.StopLoss,
			TP1:         d.Signal.TP1,
			TP2:         d.Signal.TP2,
			TP3:         d.Signal.TP3,
			RawScore:    d.Signal.RawScore,
		}
		b, _ := json.Marshal(dto)
		best.line = "SIGNAL|" + string(b)
	}
	return best
}

func signalClass(id string) string {
	switch id {
	case "ULTRA_SCALPING", "STANDARD_SCALPING":
		return "SCALP"
	case "STANDARD_SWING":
		return "SWING"
	case "TREND_SWING":
		return "TREND"
	}
	return "SIGNAL"
}

func gradeOf(score float64) string {
	switch {
	case score >= 80:
		return "A"
	case score >= 70:
		return "B"
	default:
		return "C"
	}
}
