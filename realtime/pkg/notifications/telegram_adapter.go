package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramProvider sends notifications via Telegram Bot API.
type TelegramProvider struct {
	botToken string
	chatID   string
	client   *http.Client
}

// NewTelegramProvider creates a Telegram notification provider.
// Returns nil (not configured) if token or chat ID is empty.
func NewTelegramProvider(botToken, chatID string) *TelegramProvider {
	if botToken == "" || chatID == "" {
		return nil
	}
	return &TelegramProvider{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramProvider) Send(ctx context.Context, n *Notification) error {
	if t == nil {
		return fmt.Errorf("telegram provider not initialized")
	}

	text := fmt.Sprintf("🔔 *%s*\n\n%s\n\n`%s`", n.Title, n.Message, n.EventType)

	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}
	return nil
}

func (t *TelegramProvider) Channel() Channel { return ChannelTelegram }
func (t *TelegramProvider) IsConfigured() bool {
	return t != nil && t.botToken != "" && t.chatID != ""
}
func (t *TelegramProvider) Name() string { return "telegram" }
