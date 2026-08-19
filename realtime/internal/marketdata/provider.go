// Package marketdata implements market data ingestion, tick normalization, and candle aggregation.
// SOW Sections 6, 6A, 8, 150
package marketdata

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Provider is the interface for market data sources.
type Provider interface {
	Name() string
	Connect(ctx context.Context) error
	Subscribe(symbol string) error
	Stream() <-chan *types.Tick
	Close() error
}

// TickValidator validates and normalizes raw ticks.
type TickValidator struct {
	lastSequence map[string]uint64
	mu           sync.Mutex
}

func NewTickValidator() *TickValidator {
	return &TickValidator{lastSequence: make(map[string]uint64)}
}

func (v *TickValidator) Validate(tick *types.Tick) (valid bool, reason string) {
	if tick == nil {
		return false, "nil_tick"
	}
	if tick.Bid.LessThanOrEqual(decimal.Zero) || tick.Ask.LessThanOrEqual(decimal.Zero) {
		return false, "zero_or_negative_price"
	}
	if tick.Bid.GreaterThan(tick.Ask) {
		return false, "inverted_spread"
	}
	mid := tick.Bid.Add(tick.Ask).Div(decimal.NewFromInt(2))
	if mid.GreaterThan(decimal.Zero) {
		spreadPct := tick.Spread.Div(mid)
		if spreadPct.GreaterThan(decimal.NewFromFloat(0.01)) {
			return false, "unreasonable_spread"
		}
	}
	return true, ""
}

func NormalizeTick(tick *types.Tick) {
	if tick == nil {
		return
	}
	tick.Mid = tick.Bid.Add(tick.Ask).Div(decimal.NewFromInt(2))
	tick.Spread = tick.Ask.Sub(tick.Bid)
	now := time.Now().UTC()
	if tick.GatewayTimestamp.IsZero() {
		tick.GatewayTimestamp = now
	}
}

// NewProvider creates a provider based on mode.
// "agent" = real MT5 data from Windows Agent (PRODUCTION)
// "simulated" = fake data for DEV/TEST ONLY
// "replay" = historical replay
func NewProvider(mode, symbol string, basePrice float64, tickRateMs int) Provider {
	switch mode {
	case "agent":
		return NewAgentProvider()
	case "replay":
		return NewReplayProvider(symbol)
	case "simulated":
		return NewSimulatedProvider(symbol, basePrice, tickRateMs)
	default:
		return NewAgentProvider() // Default to agent (real data) — NO fake data in production
	}
}

// StaleDetector monitors tick freshness.
type StaleDetector struct {
	maxAge       time.Duration
	lastTickTime map[string]time.Time
	mu           sync.RWMutex
}

func NewStaleDetector(maxAge time.Duration) *StaleDetector {
	return &StaleDetector{maxAge: maxAge, lastTickTime: make(map[string]time.Time)}
}

func (d *StaleDetector) Update(symbol string, t time.Time) {
	d.mu.Lock()
	d.lastTickTime[symbol] = t
	d.mu.Unlock()
}

func (d *StaleDetector) IsStale(symbol string) bool {
	d.mu.RLock()
	last, ok := d.lastTickTime[symbol]
	d.mu.RUnlock()
	if !ok {
		return true
	}
	return time.Since(last) > d.maxAge
}

func (d *StaleDetector) Staleness(symbol string) time.Duration {
	d.mu.RLock()
	last, ok := d.lastTickTime[symbol]
	d.mu.RUnlock()
	if !ok {
		return time.Duration(math.MaxInt64)
	}
	return time.Since(last)
}

// SimulatedProvider — DEV/TEST ONLY. Generates fake ticks.
// NEVER used in production. Clearly labeled as ESTIMATED quality.
type SimulatedProvider struct {
	name       string
	symbol     string
	basePrice  float64
	tickRateMs int
	mu         sync.Mutex
	rng        *rand.Rand
	ch         chan *types.Tick
	stopChan   chan struct{}
	running    atomic.Bool
	validator  *TickValidator
	sequence   uint64
}

func NewSimulatedProvider(symbol string, basePrice float64, tickRateMs int) *SimulatedProvider {
	return &SimulatedProvider{
		name: "SIMULATED", symbol: symbol, basePrice: basePrice, tickRateMs: tickRateMs,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
		ch: make(chan *types.Tick, 1024), stopChan: make(chan struct{}),
		validator: NewTickValidator(),
	}
}

func (p *SimulatedProvider) Name() string { return p.name }
func (p *SimulatedProvider) Connect(ctx context.Context) error {
	p.running.Store(true)
	go p.generate()
	return nil
}
func (p *SimulatedProvider) Subscribe(symbol string) error { p.mu.Lock(); p.symbol = symbol; p.mu.Unlock(); return nil }
func (p *SimulatedProvider) Stream() <-chan *types.Tick { return p.ch }
func (p *SimulatedProvider) Close() error {
	if p.running.CompareAndSwap(true, false) { close(p.stopChan) }
	return nil
}

func (p *SimulatedProvider) generate() {
	ticker := time.NewTicker(time.Duration(p.tickRateMs) * time.Millisecond)
	defer ticker.Stop()
	currentPrice := p.basePrice
	for {
		select {
		case <-p.stopChan:
			close(p.ch)
			return
		case <-ticker.C:
			p.mu.Lock()
			drift := (p.basePrice - currentPrice) * 0.001
			noise := p.rng.NormFloat64() * 0.15
			currentPrice += drift + noise
			p.mu.Unlock()
			if currentPrice <= 0 { currentPrice = p.basePrice }
			spread := 0.15 + p.rng.Float64()*0.20
			bid := currentPrice - spread/2
			ask := currentPrice + spread/2
			p.sequence++
			tick := &types.Tick{
				Symbol: p.symbol, Bid: decimal.NewFromFloat(bid), Ask: decimal.NewFromFloat(ask),
				TickVolume: int64(1 + p.rng.Intn(100)), Source: p.name,
				SourceTimestamp: time.Now().UTC(), GatewayTimestamp: time.Now().UTC(),
				Quality: types.QualityEstimated, Sequence: p.sequence,
			}
			NormalizeTick(tick)
			if valid, _ := p.validator.Validate(tick); valid {
				select { case p.ch <- tick: default: }
			}
		}
	}
}

// ReplayProvider replays historical ticks from the database.
type ReplayProvider struct {
	name string; symbol string; ch chan *types.Tick; stopChan chan struct{}; running atomic.Bool
}

func NewReplayProvider(symbol string) *ReplayProvider {
	return &ReplayProvider{name: "REPLAY", symbol: symbol, ch: make(chan *types.Tick, 1024), stopChan: make(chan struct{})}
}
func (p *ReplayProvider) Name() string { return p.name }
func (p *ReplayProvider) Connect(ctx context.Context) error { return nil }
func (p *ReplayProvider) Subscribe(symbol string) error { p.symbol = symbol; return nil }
func (p *ReplayProvider) Stream() <-chan *types.Tick { return p.ch }
func (p *ReplayProvider) Close() error { if p.running.CompareAndSwap(true, false) { close(p.stopChan) }; return nil }
