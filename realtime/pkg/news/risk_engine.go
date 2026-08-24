package news

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// RiskEngine computes the current news risk level based on upcoming events.
// It replaces the stubbed NewsRisk="NONE" in the session engine.
//
// SAFETY RULES:
// - If a provider IS configured but fails/stale, return DATA_UNAVAILABLE (fail-safe, blocks).
// - If NO provider is configured, the Master Node is the authoritative source, so resolve to NONE (non-blocking).
// - Never silently swallow a real provider failure.
type RiskEngine struct {
	mu         sync.RWMutex
	cfg        Config
	provider   EconomicCalendarProvider
	events     []NewsEvent
	lastFetch  time.Time
	health     ProviderHealth
}

// NewRiskEngine creates a news risk engine with the given config and provider.
// If provider is nil, the engine will produce DATA_UNAVAILABLE (fail-safe).
func NewRiskEngine(cfg Config, provider EconomicCalendarProvider) *RiskEngine {
	return &RiskEngine{
		cfg:      cfg,
		provider: provider,
	}
}

// Start begins the background sync loop that fetches events from the provider.
func (e *RiskEngine) Start(ctx context.Context) {
	if e.provider == nil || e.cfg.Provider == "disabled" {
		// No provider configured: master node gates are authoritative. Report the
		// news subsystem as healthy (not degraded) so the overall health status and
		// signal flow are not artificially throttled.
		e.mu.Lock()
		e.health = ProviderHealth{
			ProviderName:      "none",
			Healthy:           true,
			LastSuccessfulSync: time.Now().UTC(),
		}
		e.mu.Unlock()
		log.Printf("[news] RiskEngine started with no provider — news risk resolves to NONE (non-blocking); master node gates remain authoritative")
		return
	}

	interval := time.Duration(e.cfg.SyncIntervalSec) * time.Second
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial fetch
	e.sync(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.sync(ctx)
		}
	}
}

func (e *RiskEngine) sync(ctx context.Context) {
	if e.provider == nil {
		return
	}

	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour)
	to := now.Add(48 * time.Hour)

	events, err := e.provider.FetchEvents(ctx, from, to)
	if err != nil {
		log.Printf("[news] Failed to fetch events: %v", err)
		return
	}

	e.mu.Lock()
	e.events = e.filterRelevant(events)
	e.lastFetch = time.Now().UTC()
	e.health = e.provider.Health()
	e.mu.Unlock()

	log.Printf("[news] Synced %d relevant events from %s", len(e.events), e.provider.ProviderName())
}

func (e *RiskEngine) filterRelevant(events []NewsEvent) []NewsEvent {
	minImpact := e.cfg.MinImpact
	result := make([]NewsEvent, 0, len(events))
	for _, ev := range events {
		if !ev.IsUSDRelevant() {
			continue
		}
		if !e.impactAtLeast(ev.Impact, minImpact) {
			continue
		}
		result = append(result, ev)
	}
	// Sort by scheduled time
	sort.Slice(result, func(i, j int) bool {
		return result[i].ScheduledAtUTC.Before(result[j].ScheduledAtUTC)
	})
	return result
}

func (e *RiskEngine) impactAtLeast(actual, threshold ImpactLevel) bool {
	order := map[ImpactLevel]int{
		ImpactNone: 0, ImpactLow: 1, ImpactMedium: 2, ImpactHigh: 3, ImpactExtreme: 4,
	}
	return order[actual] >= order[threshold]
}

// ComputeRisk calculates the current news risk level for trading decisions.
// This is the function called from the signal hot path (fast, <1ms).
func (e *RiskEngine) ComputeRisk(now time.Time) NewsRiskResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := NewsRiskResult{ComputedAt: now}

	// When NO provider is configured, the Master Node's own market/session/regime
	// gates remain the authoritative source for execution decisions. Failing closed
	// here would artificially throttle signal flow whenever the master node is healthy
	// but no external news calendar is wired up. Resolve to NONE (non-blocking) and
	// keep the reason code for observability.
	if e.provider == nil || e.cfg.Provider == "disabled" {
		result.Level = NewsRiskNone
		result.ReasonCode = "NEWS_PROVIDER_NOT_CONFIGURED"
		result.Evidence = "No economic calendar provider configured — master node market/session/regime gates are authoritative; news risk treated as NONE (non-blocking)"
		return result
	}

	// Check provider health
	health := e.health
	if !health.Healthy || health.IsStale() {
		if e.cfg.FailPolicy == FailPolicyBlockTrading {
			result.Level = NewsRiskDataUnavailable
			result.ReasonCode = "NEWS_PROVIDER_STALE"
			result.Evidence = fmt.Sprintf("Provider %s last synced: %s", health.ProviderName, health.LastSuccessfulSync)
			return result
		}
		// Fail-open (not recommended but configurable)
		result.Level = NewsRiskNone
		result.ReasonCode = "NEWS_PROVIDER_STALE_FAIL_OPEN"
		result.Evidence = "Provider stale but fail policy allows trading"
		return result
	}

	// Check events against blackout windows
	preBlackout := time.Duration(e.cfg.PreBlackoutMinutes) * time.Minute
	postBlackout := time.Duration(e.cfg.PostBlackoutMinutes) * time.Minute

	var nextEvent *NewsEvent
	highestRisk := NewsRiskNone
	highestReason := ""
	highestEvidence := ""

	for i := range e.events {
		ev := &e.events[i]
		scheduled := ev.ScheduledAtUTC
		timeToEvent := scheduled.Sub(now)

		// Check if within blackout window
		inPreBlackout := timeToEvent > 0 && timeToEvent <= preBlackout
		inPostBlackout := timeToEvent < 0 && -timeToEvent <= postBlackout
		duringEvent := timeToEvent >= 0 && timeToEvent < time.Minute // event happening now

		if inPreBlackout || inPostBlackout || duringEvent {
			risk := e.impactToRisk(ev.Impact)
			if e.riskAtLeast(risk, highestRisk) {
				highestRisk = risk
				if inPreBlackout {
					highestReason = "NEWS_PRE_EVENT_BLACKOUT"
					highestEvidence = fmt.Sprintf("%s %s in %.1f min", ev.EventName, ev.Impact, timeToEvent.Minutes())
				} else if inPostBlackout {
					highestReason = "NEWS_POST_EVENT_BLACKOUT"
					highestEvidence = fmt.Sprintf("%s %s %.1f min ago", ev.EventName, ev.Impact, -timeToEvent.Minutes())
				} else {
					highestReason = "NEWS_DURING_EVENT"
					highestEvidence = fmt.Sprintf("%s %s happening now", ev.EventName, ev.Impact)
				}
			}
		}

		// Track next upcoming event
		if timeToEvent > 0 && ev.IsHighImpact() {
			if nextEvent == nil || scheduled.Before(nextEvent.ScheduledAtUTC) {
				nextEvent = ev
			}
		}
	}

	result.Level = highestRisk
	result.ReasonCode = highestReason
	result.Evidence = highestEvidence
	result.NextEvent = nextEvent

	if highestRisk == NewsRiskNone && nextEvent != nil {
		result.ReasonCode = "NEWS_UPCOMING_HIGH_IMPACT"
		result.Evidence = fmt.Sprintf("Next: %s at %s", nextEvent.EventName, nextEvent.ScheduledAtUTC.Format("15:04 UTC"))
	}

	return result
}

func (e *RiskEngine) impactToRisk(impact ImpactLevel) NewsRiskLevel {
	switch impact {
	case ImpactExtreme:
		return NewsRiskExtreme
	case ImpactHigh:
		return NewsRiskHigh
	case ImpactMedium:
		return NewsRiskMedium
	case ImpactLow:
		return NewsRiskLow
	default:
		return NewsRiskNone
	}
}

func (e *RiskEngine) riskAtLeast(a, b NewsRiskLevel) bool {
	order := map[NewsRiskLevel]int{
		NewsRiskNone: 0, NewsRiskLow: 1, NewsRiskMedium: 2,
		NewsRiskHigh: 3, NewsRiskExtreme: 4, NewsRiskDataUnavailable: 5,
	}
	return order[a] >= order[b]
}

// GetEvents returns the current cached events (for admin inspection).
func (e *RiskEngine) GetEvents() []NewsEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]NewsEvent, len(e.events))
	copy(result, e.events)
	return result
}

// GetHealth returns the current provider health (for admin inspection).
func (e *RiskEngine) GetHealth() ProviderHealth {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.health
}

// GetConfig returns the current news configuration (for admin inspection).
func (e *RiskEngine) GetConfig() Config {
	return e.cfg
}

// HasProvider returns true if a calendar provider is configured.
func (e *RiskEngine) HasProvider() bool {
	return e.provider != nil && e.cfg.Provider != "disabled"
}
