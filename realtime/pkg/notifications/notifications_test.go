package notifications

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	mu         sync.Mutex
	channel    Channel
	configured bool
	sendError  error
	sendCount  int
}

func (m *mockProvider) Send(ctx context.Context, n *Notification) error {
	m.mu.Lock()
	m.sendCount++
	m.mu.Unlock()
	return m.sendError
}
func (m *mockProvider) Channel() Channel  { return m.channel }
func (m *mockProvider) IsConfigured() bool { return m.configured }
func (m *mockProvider) Name() string       { return "mock" }

func (m *mockProvider) getSendCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendCount
}

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

	if p.getSendCount() == 0 {
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
	if p.getSendCount() < 2 {
		t.Fatalf("expected at least 2 retry attempts, got %d", p.getSendCount())
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
