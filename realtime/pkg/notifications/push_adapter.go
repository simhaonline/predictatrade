// Package notifications — ntfy push notification adapter.
//
// ntfy is a self-hosted HTTP-based pub-sub notification service.
// https://github.com/binwiederhier/ntfy
//
// Send notifications via HTTP POST to a self-hosted ntfy server:
//   POST https://your-ntfy-server/topic
//   Headers: Title, Priority, Tags
//   Body: message text
//
// This is a self-service push notification solution — no third-party
// cloud provider (FCM/APNs) required. Users install the ntfy Android/iOS app
// and subscribe to the topic.
package notifications

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// NtfyPushProvider sends push notifications via a self-hosted ntfy server.
//
// Setup:
//  1. Self-host ntfy server (Docker, binary, or ntfy.sh hosted)
//  2. Set NTFY_SERVER_URL to your server (e.g., https://ntfy.yourdomain.com)
//  3. Set NTFY_TOPIC to your notification topic (e.g., predictatrade-alerts)
//  4. Optionally set NTFY_ACCESS_TOKEN for authenticated topics
//  5. Users install ntfy Android/iOS app and subscribe to the topic
//
// The ntfy Android app: https://github.com/binwiederhier/ntfy-android
// The ntfy iOS app: https://github.com/binwiederhier/ntfy-ios
type NtfyPushProvider struct {
	serverURL   string // e.g., https://ntfy.yourdomain.com
	topic       string // e.g., predictatrade-alerts
	accessToken string // optional auth token for protected topics
	httpClient  *http.Client
}

// NewNtfyPushProvider creates a push notification provider using a self-hosted ntfy server.
// Returns nil (not configured) if serverURL or topic is empty.
func NewNtfyPushProvider(serverURL, topic, accessToken string) *NtfyPushProvider {
	if serverURL == "" || topic == "" {
		return nil
	}
	// Normalize URL — strip trailing slash
	serverURL = strings.TrimRight(serverURL, "/")
	return &NtfyPushProvider{
		serverURL:   serverURL,
		topic:       topic,
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Send delivers a notification via ntfy HTTP POST.
func (p *NtfyPushProvider) Send(ctx context.Context, n *Notification) error {
	if p == nil {
		return fmt.Errorf("ntfy push provider not initialized")
	}

	// Build the ntfy endpoint URL: POST {serverURL}/{topic}
	url := fmt.Sprintf("%s/%s", p.serverURL, p.topic)

	// Prepare the request body — the message text
	body := fmt.Sprintf("%s\n\n%s", n.Title, n.Message)
	if n.EventType != "" {
		body = fmt.Sprintf("[%s] %s\n\n%s", n.EventType, n.Title, n.Message)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create ntfy request: %w", err)
	}

	// Set ntfy-specific headers
	req.Header.Set("Title", fmt.Sprintf("Predict-A-Trade: %s", n.Title))

	// Map severity to ntfy priority
	switch strings.ToUpper(n.Severity) {
	case "CRITICAL", "EMERGENCY":
		req.Header.Set("Priority", "5") // max priority
		req.Header.Set("Tags", "rotating_light,triangle_exclamation")
	case "HIGH", "WARNING":
		req.Header.Set("Priority", "4") // high priority
		req.Header.Set("Tags", "warning,chart_with_upwards_trend")
	case "MEDIUM", "INFO":
		req.Header.Set("Priority", "3") // default priority
		req.Header.Set("Tags", "bell")
	default:
		req.Header.Set("Priority", "2") // low priority
	}

	// Set auth token if configured (for protected topics)
	if p.accessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ntfy server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *NtfyPushProvider) Channel() Channel { return ChannelPush }
func (p *NtfyPushProvider) IsConfigured() bool {
	return p != nil && p.serverURL != "" && p.topic != ""
}
func (p *NtfyPushProvider) Name() string { return "ntfy" }
