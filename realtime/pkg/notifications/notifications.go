// Package notifications implements external notification provider adapters.
//
// Operator-authorized implementation (v1.9.0).
//
// The internal WebSocket notification path is preserved and not replaced.
// External adapters (email, Telegram, WhatsApp, push) are built around it.
//
// CRITICAL SAFETY:
// - Notification failure must NOT crash the trading engine.
// - Missing credentials produce NOT_CONFIGURED status, not fake success.
// - All adapters use async delivery with retry.
// - No secrets are logged.
package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Channel represents a notification delivery channel.
type Channel string

const (
	ChannelWebSocket Channel = "INTERNAL_WEBSOCKET"
	ChannelEmail     Channel = "EMAIL"
	ChannelTelegram  Channel = "TELEGRAM"
	ChannelPush      Channel = "PUSH"
)

// DeliveryStatus represents the delivery state of a notification.
type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "PENDING"
	StatusSent      DeliveryStatus = "SENT"
	StatusFailed    DeliveryStatus = "FAILED"
	StatusNotConfigured DeliveryStatus = "NOT_CONFIGURED"
	StatusDeadLetter DeliveryStatus = "DEAD_LETTER"
)

// EventType represents the type of notification event.
type EventType string

const (
	EventSignalApproved       EventType = "SIGNAL_APPROVED"
	EventSignalRejected       EventType = "SIGNAL_REJECTED"
	EventTradeOpened          EventType = "TRADE_OPENED"
	EventTradeClosed          EventType = "TRADE_CLOSED"
	EventPendingOrderCreated  EventType = "PENDING_ORDER_CREATED"
	EventPendingOrderFilled   EventType = "PENDING_ORDER_FILLED"
	EventPendingOrderCancelled EventType = "PENDING_ORDER_CANCELLED"
	EventOCOTriggered         EventType = "OCO_TRIGGERED"
	EventOCOReconciled        EventType = "OCO_RECONCILED"
	EventTP1Hit               EventType = "TP1_HIT"
	EventTP2Hit               EventType = "TP2_HIT"
	EventTP3Hit               EventType = "TP3_HIT"
	EventTrailingStopUpdated  EventType = "TRAILING_STOP_UPDATED"
	EventDailyLossLock        EventType = "DAILY_LOSS_LOCK"
	EventDrawdownLock         EventType = "DRAWDOWN_LOCK"
	EventEmergencyClose       EventType = "EMERGENCY_CLOSE"
	EventNewsProtectionActive EventType = "NEWS_PROTECTION_ACTIVE"
	EventNewsBreakoutArmed    EventType = "NEWS_BREAKOUT_ARMED"
	EventNewsBreakoutTriggered EventType = "NEWS_BREAKOUT_TRIGGERED"
	EventBrokerDisconnected   EventType = "BROKER_DISCONNECTED"
	EventBrokerReconnected    EventType = "BROKER_RECONNECTED"
	EventExecutionRejected    EventType = "EXECUTION_REJECTED"
)

// Notification is the normalized notification event.
type Notification struct {
	NotificationID   string          `json:"notification_id"`
	EventType        EventType       `json:"event_type"`
	Severity         string          `json:"severity"`
	UserAccount      string          `json:"user_account,omitempty"`
	TradeID          string          `json:"trade_id,omitempty"`
	SignalID         string          `json:"signal_id,omitempty"`
	OrderID          string          `json:"order_id,omitempty"`
	PositionID       string          `json:"position_id,omitempty"`
	Title            string          `json:"title"`
	Message          string          `json:"message"`
	StructuredPayload json.RawMessage `json:"structured_payload,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	DeliveryChannel  Channel         `json:"delivery_channel"`
	DeliveryStatus   DeliveryStatus  `json:"delivery_status"`
	AttemptCount     int             `json:"attempt_count"`
	LastError        string          `json:"last_error,omitempty"`
}

// Provider is the interface for notification delivery channels.
type Provider interface {
	// Send delivers a notification through this channel.
	Send(ctx context.Context, n *Notification) error
	// Channel returns the channel type.
	Channel() Channel
	// IsConfigured returns true if the provider has valid credentials.
	IsConfigured() bool
	// Name returns the provider name for logging.
	Name() string
}

// Config holds notification configuration.
type Config struct {
	EmailEnabled    bool   `json:"email_enabled"`
	TelegramEnabled bool   `json:"telegram_enabled"`
	WhatsAppEnabled bool   `json:"whatsapp_enabled"`
	PushEnabled     bool   `json:"push_enabled"`
	MaxRetries      int    `json:"max_retries"`
	RetryIntervalMs int    `json:"retry_interval_ms"`

	// SMTP
	SMTPHost     string `json:"-"`
	SMTPPort     int    `json:"-"`
	SMTPUsername string `json:"-"`
	SMTPPassword string `json:"-"`
	SMTPFrom     string `json:"-"`
	SMPTTLS      bool   `json:"-"`

	// Telegram
	TelegramBotToken string `json:"-"`
	TelegramChatID   string `json:"-"`

	// WhatsApp (provider-dependent)
	WhatsAppAPIURL string `json:"-"`
	WhatsAppToken  string `json:"-"`

	// Push
	PushProviderURL string `json:"-"`
	PushAPIKey      string `json:"-"`
}

// DefaultConfig returns disabled-by-default notification configuration.
func DefaultConfig() Config {
	return Config{
		EmailEnabled:    false,
		TelegramEnabled: false,
		WhatsAppEnabled: false,
		PushEnabled:     false,
		MaxRetries:      3,
		RetryIntervalMs: 5000,
	}
}

// Manager orchestrates notification delivery across all configured channels.
// Trading execution must NOT fail because of notification delivery issues.
type Manager struct {
	mu        sync.RWMutex
	cfg       Config
	providers map[Channel]Provider
	queue     chan *Notification
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager creates a notification manager.
func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:       cfg,
		providers: make(map[Channel]Provider),
		queue:     make(chan *Notification, 256),
	}
}

// RegisterProvider adds a delivery provider.
func (m *Manager) RegisterProvider(p Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Channel()] = p
	if p.IsConfigured() {
		log.Printf("[notifications] Provider %s (%s) registered and configured", p.Name(), p.Channel())
	} else {
		log.Printf("[notifications] Provider %s (%s) registered but NOT configured", p.Name(), p.Channel())
	}
}

// Start begins the background delivery loop.
func (m *Manager) Start(ctx context.Context) {
	m.ctx, m.cancel = context.WithCancel(ctx)
	go m.deliveryLoop()
}

// Stop shuts down the notification manager.
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// Enqueue adds a notification to the delivery queue (non-blocking).
// If the queue is full, the notification is dropped (trading must not block).
func (m *Manager) Enqueue(n *Notification) {
	if n.NotificationID == "" {
		n.NotificationID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	}
	n.CreatedAt = time.Now().UTC()
	n.DeliveryStatus = StatusPending

	select {
	case m.queue <- n:
		// queued successfully
	default:
		log.Printf("[notifications] Queue full — dropping notification %s (%s)", n.NotificationID, n.EventType)
	}
}

func (m *Manager) deliveryLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case n := <-m.queue:
			m.deliver(n)
		}
	}
}

func (m *Manager) deliver(n *Notification) {
	m.mu.RLock()
	providers := make([]Provider, 0, len(m.providers))
	for _, p := range m.providers {
		providers = append(providers, p)
	}
	m.mu.RUnlock()

	for _, p := range providers {
		if !p.IsConfigured() {
			n.DeliveryChannel = p.Channel()
			n.DeliveryStatus = StatusNotConfigured
			continue
		}

		n.DeliveryChannel = p.Channel()
		n.AttemptCount = 0

		for attempt := 0; attempt <= m.cfg.MaxRetries; attempt++ {
			n.AttemptCount = attempt + 1
			ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
			err := p.Send(ctx, n)
			cancel()

			if err == nil {
				n.DeliveryStatus = StatusSent
				break
			}

			n.LastError = err.Error()
			if attempt < m.cfg.MaxRetries {
				time.Sleep(time.Duration(m.cfg.RetryIntervalMs) * time.Millisecond)
			}
		}

		if n.DeliveryStatus != StatusSent {
			n.DeliveryStatus = StatusFailed
			log.Printf("[notifications] Delivery failed for %s via %s: %s (attempts=%d)",
				n.NotificationID, p.Channel(), n.LastError, n.AttemptCount)
		}
	}
}

// GetProviderStatus returns the status of all configured providers (for admin inspection).
func (m *Manager) GetProviderStatus() []ProviderStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []ProviderStatus
	for _, p := range m.providers {
		result = append(result, ProviderStatus{
			Channel:     p.Channel(),
			Name:        p.Name(),
			Configured:  p.IsConfigured(),
			// Never expose credentials — only show configured/not-configured
		})
	}
	return result
}

// ProviderStatus shows the configuration state of a notification provider.
type ProviderStatus struct {
	Channel    Channel `json:"channel"`
	Name       string  `json:"name"`
	Configured bool    `json:"configured"`
}
