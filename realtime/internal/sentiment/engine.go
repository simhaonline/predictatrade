// Package sentiment implements the real-time sentiment engine.
//
// CRITICAL: Sentiment data is collected ASYNCHRONOUSLY via a background
// refresh goroutine. The signal-generation hot path only reads the cached
// snapshot — it NEVER performs slow external HTTP calls synchronously.
//
// If sentiment is unavailable or stale: neutral/no-impact fallback.
// Never create random sentiment values.
package sentiment

import (
	"context"
	"math"
	"sync"
	"time"
)

// Config holds sentiment engine configuration.
type Config struct {
	Enabled             bool
	RefreshIntervalSec   int
	TimeoutSec           int
	MaxRetries           int
	StaleThresholdSec    int
	MinConfidenceThreshold float64
	ProviderWeights      map[string]float64
}

// DefaultConfig returns safe defaults. Sentiment is disabled by default.
func DefaultConfig() Config {
	return Config{
		Enabled:               false, // disabled by default — requires API credentials
		RefreshIntervalSec:    300,   // 5 minutes
		TimeoutSec:            10,
		MaxRetries:            3,
		StaleThresholdSec:     600, // 10 minutes
		MinConfidenceThreshold: 0.5,
		ProviderWeights: map[string]float64{
			"gdelt":    0.25,
			"reuters":  0.30,
			"fed":      0.20,
			"reddit":   0.10,
			"twitter":  0.10,
			"internal": 0.05,
		},
	}
}

// SentimentItem represents a single sentiment data point with provenance.
type SentimentItem struct {
	Source          string    // gdelt, reuters, fed, reddit, twitter_x, internal
	Provider        string
	HeadlineID      string    // URL or unique identifier
	Score           float64   // -100 to +100
	Confidence      float64   // 0 to 1
	Category        string    // BULLISH, BEARISH, NEUTRAL, MIXED
	TextPreview     string
	SourceTimestamp time.Time
	FetchedAt       time.Time
	AgeSeconds      int
}

// Snapshot is the cached sentiment state read by the signal engine.
type Snapshot struct {
	OverallScore       float64 // -100 to +100
	OverallConfidence  float64 // 0 to 1
	Category           string  // BULLISH, BEARISH, NEUTRAL, MIXED
	ItemCount          int
	SourceCount         int
	ProviderHealth     map[string]ProviderHealth
	LastSuccessfulUpdate time.Time
	DataAgeSeconds     int
	Items              []SentimentItem
}

// ProviderHealth tracks the health of each sentiment provider.
type ProviderHealth struct {
	Status       string // OK, DEGRADED, ERROR
	LastSuccess  time.Time
	ErrorCount   int
	LastErrorMessage string
}

// Provider is the interface for sentiment data providers.
// One unavailable provider does not break the system.
type Provider interface {
	Name() string
	Fetch(ctx context.Context) ([]SentimentItem, error)
}

// Engine manages the sentiment subsystem with async refresh.
type Engine struct {
	mu         sync.RWMutex
	config     Config
	snapshot   *Snapshot
	providers  []Provider
	stopCh     chan struct{}
	running    bool
}

// NewEngine creates a sentiment engine.
func NewEngine(cfg Config, providers []Provider) *Engine {
	return &Engine{
		config:    cfg,
		providers: providers,
		stopCh:    make(chan struct{}),
		snapshot: &Snapshot{
			OverallScore:      0,
			OverallConfidence: 0,
			Category:          "NEUTRAL",
			ProviderHealth:    make(map[string]ProviderHealth),
		},
	}
}

// Start begins the background refresh goroutine.
func (e *Engine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return
	}
	e.running = true
	go e.refreshLoop()
}

// Stop stops the background refresh.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}
	close(e.stopCh)
	e.running = false
}

// GetSnapshot returns the current cached sentiment snapshot.
// This is the ONLY function called from the signal hot path.
// It NEVER performs external HTTP calls.
func (e *Engine) GetSnapshot() *Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.snapshot == nil {
		return &Snapshot{
			OverallScore:      0,
			OverallConfidence: 0,
			Category:          "NEUTRAL",
			ProviderHealth:    make(map[string]ProviderHealth),
		}
	}
	cp := *e.snapshot
	return &cp
}

// IsStale checks if the current snapshot is stale.
func (e *Engine) IsStale() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.snapshot == nil || e.snapshot.LastSuccessfulUpdate.IsZero() {
		return true
	}
	age := time.Since(e.snapshot.LastSuccessfulUpdate)
	return age > time.Duration(e.config.StaleThresholdSec)*time.Second
}

// GetInfluence returns the sentiment influence factor for PTB.
// If stale or unavailable, returns 0 (neutral/no-impact).
func (e *Engine) GetInfluence() float64 {
	if !e.config.Enabled {
		return 0
	}
	snap := e.GetSnapshot()
	if e.IsStale() {
		return 0 // neutral fallback
	}
	if snap.OverallConfidence < e.config.MinConfidenceThreshold {
		return 0
	}
	// Normalize score from -100..100 to -1..1
	return snap.OverallScore / 100.0
}

// refreshLoop runs the background refresh cycle.
func (e *Engine) refreshLoop() {
	ticker := time.NewTicker(time.Duration(e.config.RefreshIntervalSec) * time.Second)
	defer ticker.Stop()

	// Initial refresh
	e.refresh()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.refresh()
		}
	}
}

// refresh fetches sentiment from all providers with timeout and retry.
func (e *Engine) refresh() {
	if len(e.providers) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(e.config.TimeoutSec)*time.Second)
	defer cancel()

	var allItems []SentimentItem
	providerHealth := make(map[string]ProviderHealth)

	for _, p := range e.providers {
		items, err := e.fetchWithRetry(ctx, p)
		health := ProviderHealth{
			Status:      "OK",
			LastSuccess: time.Now(),
		}
		if err != nil {
			health.Status = "ERROR"
			health.ErrorCount++
			health.LastErrorMessage = err.Error()
		}
		providerHealth[p.Name()] = health
		if err == nil {
			allItems = append(allItems, items...)
		}
	}

	if len(allItems) == 0 {
		// No data — keep existing snapshot, mark providers as degraded
		e.mu.Lock()
		if e.snapshot != nil {
			e.snapshot.ProviderHealth = providerHealth
		}
		e.mu.Unlock()
		return
	}

	// Compute weighted overall score
	overallScore, overallConfidence := e.computeOverall(allItems)
	category := classifyScore(overallScore)

	// Count unique sources
	sources := make(map[string]bool)
	for _, item := range allItems {
		sources[item.Source] = true
	}

	e.mu.Lock()
	e.snapshot = &Snapshot{
		OverallScore:        overallScore,
		OverallConfidence:   overallConfidence,
		Category:            category,
		ItemCount:           len(allItems),
		SourceCount:         len(sources),
		ProviderHealth:     providerHealth,
		LastSuccessfulUpdate: time.Now(),
		Items:              allItems,
	}
	e.mu.Unlock()
}

// fetchWithRetry fetches from a provider with exponential backoff retry.
func (e *Engine) fetchWithRetry(ctx context.Context, p Provider) ([]SentimentItem, error) {
	var lastErr error
	for attempt := 0; attempt < e.config.MaxRetries; attempt++ {
		items, err := p.Fetch(ctx)
		if err == nil {
			return items, nil
		}
		lastErr = err
		// Exponential backoff
		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

// computeOverall calculates the weighted overall sentiment score.
func (e *Engine) computeOverall(items []SentimentItem) (float64, float64) {
	if len(items) == 0 {
		return 0, 0
	}
	totalWeight := 0.0
	weightedScore := 0.0
	totalConfidence := 0.0

	for _, item := range items {
		weight := e.config.ProviderWeights[item.Source]
		if weight == 0 {
			weight = 0.1 // default weight for unknown sources
		}
		weightedScore += item.Score * weight * item.Confidence
		totalWeight += weight * item.Confidence
		totalConfidence += item.Confidence
	}

	if totalWeight == 0 {
		return 0, 0
	}

	overallScore := weightedScore / totalWeight
	overallConfidence := totalConfidence / float64(len(items))

	// Clamp
	overallScore = math.Max(-100, math.Min(100, overallScore))
	overallConfidence = math.Max(0, math.Min(1, overallConfidence))

	return overallScore, overallConfidence
}

// classifyScore converts a numeric score to a category.
func classifyScore(score float64) string {
	if score > 20 {
		return "BULLISH"
	}
	if score < -20 {
		return "BEARISH"
	}
	if math.Abs(score) < 5 {
		return "NEUTRAL"
	}
	return "MIXED"
}
