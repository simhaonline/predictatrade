// Package news implements the economic calendar provider architecture,
// news risk engine, and news protection/breakout mode management.
//
// SOW Sections 12D, 131: News/calendar state, fail-closed gate architecture.
// Operator-authorized implementation (v1.9.0).
package news

import (
	"context"
	"time"
)

// ImpactLevel represents the importance of an economic event.
type ImpactLevel string

const (
	ImpactNone     ImpactLevel = "NONE"
	ImpactLow      ImpactLevel = "LOW"
	ImpactMedium   ImpactLevel = "MEDIUM"
	ImpactHigh     ImpactLevel = "HIGH"
	ImpactExtreme  ImpactLevel = "EXTREME"
)

// NewsMode represents the news operating mode.
type NewsMode string

const (
	NewsModeOff          NewsMode = "OFF"
	NewsModeProtectOnly  NewsMode = "PROTECT_ONLY"
	NewsModeEventBreakout NewsMode = "EVENT_BREAKOUT"
)

// FailPolicy determines behavior when the news provider is unavailable.
type FailPolicy string

const (
	FailPolicyBlockTrading FailPolicy = "BLOCK_TRADING" // Fail-safe: block when provider down
	FailPolicyAllowTrading FailPolicy = "ALLOW_TRADING" // Fail-open: allow when provider down (not recommended)
)

// NewsEvent is the normalized internal representation of an economic calendar event.
type NewsEvent struct {
	EventID         string      `json:"event_id"`
	Provider        string      `json:"provider"`
	ProviderEventID string      `json:"provider_event_id"`
	EventName       string      `json:"event_name"`
	Country         string      `json:"country"`
	Currency        string      `json:"currency"`
	Impact          ImpactLevel `json:"impact"`
	ScheduledAtUTC  time.Time   `json:"scheduled_at_utc"`
	Actual          string      `json:"actual,omitempty"`
	Forecast        string      `json:"forecast,omitempty"`
	Previous        string      `json:"previous,omitempty"`
	EventCategory   string      `json:"event_category,omitempty"`
	SourceTimestamp time.Time   `json:"source_timestamp"`
	ReceivedAt      time.Time   `json:"received_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	Revision        int         `json:"revision"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// IsUSDRelevant returns true if this event is relevant to XAUUSD (USD-denominated).
func (e *NewsEvent) IsUSDRelevant() bool {
	return e.Currency == "USD" || e.Country == "US"
}

// IsHighImpact returns true if the event is HIGH or EXTREME impact.
func (e *NewsEvent) IsHighImpact() bool {
	return e.Impact == ImpactHigh || e.Impact == ImpactExtreme
}

// ProviderHealth describes the current state of a calendar provider.
type ProviderHealth struct {
	ProviderName      string    `json:"provider_name"`
	Healthy           bool      `json:"healthy"`
	LastSuccessfulSync time.Time `json:"last_successful_sync"`
	LastError         string    `json:"last_error,omitempty"`
	EventCount        int       `json:"event_count"`
	StaleAfter        time.Duration `json:"stale_after"`
}

// IsStale returns true if the provider has not synced within the stale threshold.
func (h *ProviderHealth) IsStale() bool {
	if h.LastSuccessfulSync.IsZero() {
		return true
	}
	return time.Since(h.LastSuccessfulSync) > h.StaleAfter
}

// EconomicCalendarProvider is the interface for economic calendar data sources.
type EconomicCalendarProvider interface {
	// FetchEvents retrieves economic events within the given time range.
	FetchEvents(ctx context.Context, from, to time.Time) ([]NewsEvent, error)
	// Health returns the current provider health status.
	Health() ProviderHealth
	// ProviderName returns the provider identifier.
	ProviderName() string
	// SupportsHistorical returns true if the provider can fetch past events.
	SupportsHistorical() bool
	// SupportsRealtime returns true if the provider offers realtime/webhook updates.
	SupportsRealtime() bool
}

// NewsRiskLevel represents the computed news risk for trading decisions.
type NewsRiskLevel string

const (
	NewsRiskNone           NewsRiskLevel = "NONE"
	NewsRiskLow            NewsRiskLevel = "LOW"
	NewsRiskMedium         NewsRiskLevel = "MEDIUM"
	NewsRiskHigh           NewsRiskLevel = "HIGH"
	NewsRiskExtreme        NewsRiskLevel = "EXTREME"
	NewsRiskDataUnavailable NewsRiskLevel = "DATA_UNAVAILABLE"
)

// ShouldBlock returns true if this risk level should veto trading.
func (r NewsRiskLevel) ShouldBlock() bool {
	return r == NewsRiskHigh || r == NewsRiskExtreme || r == NewsRiskDataUnavailable
}

// NewsRiskResult contains the computed news risk and supporting evidence.
type NewsRiskResult struct {
	Level       NewsRiskLevel   `json:"level"`
	ReasonCode  string          `json:"reason_code"`
	Evidence    string          `json:"evidence"`
	NextEvent   *NewsEvent      `json:"next_event,omitempty"`
	ComputedAt  time.Time       `json:"computed_at"`
}

// Config holds news subsystem configuration.
type Config struct {
	Provider            string     `json:"provider"`              // disabled|fmp|...
	Mode                NewsMode   `json:"mode"`                  // OFF|PROTECT_ONLY|EVENT_BREAKOUT
	FailPolicy          FailPolicy `json:"fail_policy"`           // BLOCK_TRADING|ALLOW_TRADING
	SyncIntervalSec     int        `json:"sync_interval_sec"`     // how often to fetch events
	StaleAfterSec       int        `json:"stale_after_sec"`       // stale threshold
	PreBlackoutMinutes  int        `json:"pre_blackout_minutes"`  // minutes before event to block
	PostBlackoutMinutes int        `json:"post_blackout_minutes"` // minutes after event to block
	MinImpact           ImpactLevel `json:"min_impact"`           // minimum impact to consider
	ProviderAPIKey      string     `json:"-"`                     // never serialized
}

// DefaultConfig returns safe production defaults.
// News protection is enabled (PROTECT_ONLY) but requires a provider to be configured.
// If no provider is configured, NewsRisk resolves to DATA_UNAVAILABLE (fail-safe).
func DefaultConfig() Config {
	return Config{
		Provider:            "disabled",
		Mode:                NewsModeProtectOnly,
		FailPolicy:          FailPolicyBlockTrading,
		SyncIntervalSec:     300,   // 5 minutes
		StaleAfterSec:       900,   // 15 minutes
		PreBlackoutMinutes:  15,    // 15 min pre-event blackout
		PostBlackoutMinutes: 15,    // 15 min post-event blackout
		MinImpact:           ImpactMedium,
		ProviderAPIKey:      "",
	}
}

// IsEnabled returns true if the news mode is not OFF.
func (m NewsMode) IsEnabled(mode NewsMode) bool {
	return mode != NewsModeOff
}

// IsEnabled checks if EVENT_BREAKOUT mode is active.
func (NewsMode) IsEventBreakout(mode NewsMode) bool {
	return mode == NewsModeEventBreakout
}
