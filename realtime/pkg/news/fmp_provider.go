package news

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FMPProvider implements EconomicCalendarProvider using the Financial Modeling Prep API.
// API docs: https://site.financialmodelingprep.com/developer/docs/economic-calendar
// Uses the stable endpoint: https://financialmodelingprep.com/stable/economic-calendar
type FMPProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	health     ProviderHealth
}

// fmpEvent represents the raw JSON response from FMP's economic-calendar endpoint.
// Fields can be numbers, strings, or null — use json.RawMessage for flexibility.
type fmpEvent struct {
	Event           string          `json:"event"`
	Country         string          `json:"country"`
	Currency        string          `json:"currency"`
	Actual          json.RawMessage `json:"actual"`
	Previous        json.RawMessage `json:"previous"`
	Estimate        json.RawMessage `json:"estimate"`
	Change          json.RawMessage `json:"change"`
	ChangePct       json.RawMessage `json:"changePercentage"`
	Impact          string          `json:"impact"`
	DateTime        string          `json:"date"`
	EventID         string          `json:"eventId"`
	Unit            string          `json:"unit"`
}

func NewFMPProvider(apiKey string) *FMPProvider {
	return &FMPProvider{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://financialmodelingprep.com/stable",
		health: ProviderHealth{
			ProviderName: "fmp",
			Healthy:      false,
			StaleAfter:   15 * time.Minute,
		},
	}
}

func (p *FMPProvider) ProviderName() string { return "fmp" }

func (p *FMPProvider) SupportsHistorical() bool { return true }
func (p *FMPProvider) SupportsRealtime() bool { return false }

func (p *FMPProvider) Health() ProviderHealth { return p.health }

func (p *FMPProvider) FetchEvents(ctx context.Context, from, to time.Time) ([]NewsEvent, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	fromStr := from.UTC().Format("2006-01-02")
	toStr := to.UTC().Format("2006-01-02")

	url := fmt.Sprintf("%s/economic-calendar?from=%s&to=%s&apikey=%s",
		p.baseURL, fromStr, toStr, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		p.updateHealth(false, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.updateHealth(false, err.Error())
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.updateHealth(false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("FMP API returned status %d", resp.StatusCode)
	}

	var rawEvents []fmpEvent
	if err := json.NewDecoder(resp.Body).Decode(&rawEvents); err != nil {
		p.updateHealth(false, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	events := make([]NewsEvent, 0, len(rawEvents))
	for _, raw := range rawEvents {
		event := p.normalizeEvent(raw)
		if event.IsUSDRelevant() {
			events = append(events, event)
		}
	}

	p.updateHealth(true, "")
	p.health.EventCount = len(events)
	return events, nil
}

// rawToString converts a json.RawMessage (which can be a number, string, or null) to a string.
func rawToString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Otherwise return the raw JSON value as string (number, etc.)
	return string(raw)
}

func (p *FMPProvider) normalizeEvent(raw fmpEvent) NewsEvent {
	now := time.Now().UTC()
	scheduledAt, _ := time.Parse("2006-01-02 15:04:05", raw.DateTime)
	if scheduledAt.IsZero() {
		scheduledAt, _ = time.Parse("2006-01-02", raw.DateTime)
	}
	scheduledAt = scheduledAt.UTC()

	eventID := raw.EventID
	if eventID == "" {
		eventID = fmt.Sprintf("fmp_%s_%s", raw.Country, raw.DateTime)
	}

	return NewsEvent{
		EventID:         fmt.Sprintf("fmp_%s", eventID),
		Provider:        "fmp",
		ProviderEventID: eventID,
		EventName:       raw.Event,
		Country:         raw.Country,
		Currency:        raw.Currency,
		Impact:          p.normalizeImpact(raw.Impact),
		ScheduledAtUTC:  scheduledAt,
		Actual:          rawToString(raw.Actual),
		Forecast:        rawToString(raw.Estimate),
		Previous:        rawToString(raw.Previous),
		EventCategory:   p.categorizeEvent(raw.Event),
		SourceTimestamp: now,
		ReceivedAt:      now,
		UpdatedAt:       now,
		Metadata: map[string]string{
			"change":     rawToString(raw.Change),
			"change_pct": rawToString(raw.ChangePct),
			"unit":       raw.Unit,
		},
	}
}

func (p *FMPProvider) normalizeImpact(impact string) ImpactLevel {
	switch strings.ToLower(impact) {
	case "high":
		return ImpactHigh
	case "medium":
		return ImpactMedium
	case "low":
		return ImpactLow
	default:
		return ImpactNone
	}
}

// categorizeEvent maps FMP event names to our internal categories.
func (p *FMPProvider) categorizeEvent(name string) string {
	nameLower := strings.ToLower(name)
	switch {
	case contains(nameLower, "fomc"), contains(nameLower, "rate decision"), contains(nameLower, "federal reserve"):
		return "RATE_DECISION"
	case contains(nameLower, "nonfarm"), contains(nameLower, "non-farm"), contains(nameLower, "payroll"), contains(nameLower, "jobless"):
		return "EMPLOYMENT"
	case contains(nameLower, "cpi"), contains(nameLower, "pce"), contains(nameLower, "ppi"), contains(nameLower, "inflation"):
		return "INFLATION"
	case contains(nameLower, "gdp"):
		return "GDP"
	case contains(nameLower, "unemployment"):
		return "UNEMPLOYMENT"
	case contains(nameLower, "powell"), contains(nameLower, "speech"), contains(nameLower, "yellen"), contains(nameLower, "fed chair"):
		return "SPEECH"
	case contains(nameLower, "ism"):
		return "ISM"
	case contains(nameLower, "consumer confidence"), contains(nameLower, "sentiment"):
		return "SENTIMENT"
	case contains(nameLower, "housing"), contains(nameLower, "existing home"), contains(nameLower, "new home"):
		return "HOUSING"
	case contains(nameLower, "durable goods"):
		return "DURABLE_GOODS"
	case contains(nameLower, "trade balance"):
		return "TRADE_BALANCE"
	case contains(nameLower, "treasury"), contains(nameLower, "bond auction"):
		return "TREASURY"
	case contains(nameLower, "retail sales"):
		return "OTHER"
	default:
		return "OTHER"
	}
}

func (p *FMPProvider) updateHealth(healthy bool, errMsg string) {
	p.health.Healthy = healthy
	p.health.LastError = errMsg
	if healthy {
		p.health.LastSuccessfulSync = time.Now().UTC()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
