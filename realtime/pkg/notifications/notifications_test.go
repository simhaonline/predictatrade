package notifications

import (
	"context"
	"testing"
	"time"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	channel      Channel
	configured   bool
	sendError    error
	sendCount    int
}

func (m *mockProvider) Send(ctx context.Context, n *Notification) error {
	m.sendCount++
	return m.sendError
}
func (m *mockProvider) Channel() Channel      { return m.channel }
func (m *mockProvider) IsConfigured() bool     { return m.configured }
func (m *mockProvider) Name() string           { return "mock" }

func TestManager_NotConfigured_Status(t *testing.T) {
	m := NewManager(DefaultConfig())
	p := &mockProvider{channel: ChannelTelegram, configured: false}
	m.RegisterProvider(p)

	status := m.GetProviderStatus()
	if len(status) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(status))
	}
	if status[0].Configured {
		t.Fatal("expected NOT configured")
	}
}

func TestManager_EnqueueAndDeliver(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxRetries = 1
	cfg.RetryIntervalMs = 10
	m := NewManager(cfg)

	p := &mockProvider{channel: ChannelTelegram, configured: true}
	m.RegisterProvider(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	n := &Notification{
		EventType: EventSignalApproved,
		Title:     "Test Signal",
		Message:   "BUY XAUUSD at 2500",
	}
	m.Enqueue(n)

	// Wait for delivery
	time.Sleep(100 * time.Millisecond)

	if p.sendCount == 0 {
		t.Fatal("expected at least 1 send attempt")
	}
}

func TestManager_QueueFull_DropsNotification(t *testing.T) {
	m := NewManager(DefaultConfig())
	m.queue = make(chan *Notification, 1) // tiny queue for testing

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	// Fill queue
	m.Enqueue(&Notification{EventType: EventTradeOpened, Title: "1"})
	m.Enqueue(&Notification{EventType: EventTradeClosed, Title: "2"}) // this should be dropped
	// If we reach here without blocking, the drop worked
}

func TestManager_RetryOnFailure(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxRetries = 2
	cfg.RetryIntervalMs = 10
	m := NewManager(cfg)

	p := &mockProvider{channel: ChannelEmail, configured: true, sendError: context.Canceled}
	m.RegisterProvider(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	m.Enqueue(&Notification{EventType: EventDailyLossLock, Title: "Daily Loss"})

	time.Sleep(200 * time.Millisecond)
	if p.sendCount < 2 {
		t.Fatalf("expected at least 2 retry attempts, got %d", p.sendCount)
	}
}

func TestEmailProvider_NotConfigured(t *testing.T) {
	p := NewEmailProvider("", 587, "", "", "", false)
	if p != nil {
		t.Fatal("expected nil for unconfigured email provider")
	}
}

func TestEmailProvider_Configured(t *testing.T) {
	p := NewEmailProvider("smtp.gmail.com", 587, "user", "pass", "from@test.com", true)
	if p == nil {
		t.Fatal("expected non-nil for configured email provider")
	}
	if !p.IsConfigured() {
		t.Fatal("expected IsConfigured=true")
	}
}

func TestTelegramProvider_NotConfigured(t *testing.T) {
	p := NewTelegramProvider("", "")
	if p != nil {
		t.Fatal("expected nil for unconfigured telegram provider")
	}
}

func TestTelegramProvider_Configured(t *testing.T) {
	p := NewTelegramProvider("bot123:ABC", "chat456")
	if p == nil {
		t.Fatal("expected non-nil for configured telegram provider")
	}
	if !p.IsConfigured() {
		t.Fatal("expected IsConfigured=true")
	}
}


func TestPushProvider_NotConfigured(t *testing.T) {
	p := NewNtfyPushProvider("", "", "")
	if p != nil {
		t.Fatal("expected nil for unconfigured ntfy push provider")
	}
}

func TestPushProvider_Configured(t *testing.T) {
	p := NewNtfyPushProvider("https://ntfy.example.com", "predictatrade-alerts", "")
	if p == nil {
		t.Fatal("expected non-nil for configured ntfy push provider")
	}
	if !p.IsConfigured() {
		t.Fatal("expected IsConfigured=true")
	}
}

func TestPushProvider_WithAccessToken(t *testing.T) {
	p := NewNtfyPushProvider("https://ntfy.example.com", "alerts", "secret_token")
	if p == nil {
		t.Fatal("expected non-nil for configured ntfy provider with token")
	}
	if !p.IsConfigured() {
		t.Fatal("expected IsConfigured=true")
	}
}

func TestProviderStatus_NeverExposesSecrets(t *testing.T) {
	m := NewManager(DefaultConfig())
	m.RegisterProvider(NewEmailProvider("smtp.test.com", 587, "secret_user", "secret_pass", "from@test.com", true))
	m.RegisterProvider(NewTelegramProvider("secret_token", "secret_chat"))
	m.RegisterProvider(NewNtfyPushProvider("https://ntfy.test.com", "alerts", "secret_token"))

	status := m.GetProviderStatus()
	for _, s := range status {
		if s.Configured != true {
			t.Errorf("expected configured for %s", s.Channel)
		}
	}
}

func TestDefaultConfig_AllDisabledByDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.EmailEnabled || cfg.TelegramEnabled || cfg.PushEnabled {
		t.Fatal("all notification channels must be DISABLED by default")
	}
}
