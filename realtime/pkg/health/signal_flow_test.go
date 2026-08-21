package health

import (
	"testing"
)

func TestSignalFlow_Normal(t *testing.T) {
	m := NewSignalFlowMonitor(5)
	m.OnSignalGenerated("BUY")
	m.OnCandleProcessed()
	if m.IsBlocked() {
		t.Error("Should not be blocked after 1 blank candle")
	}
}

func TestSignalFlow_Blocked(t *testing.T) {
	m := NewSignalFlowMonitor(5)
	for i := 0; i < 5; i++ {
		m.OnCandleProcessed()
	}
	if !m.IsBlocked() {
		t.Error("Should be blocked after 5 blank candles")
	}
}

func TestSignalFlow_Reset(t *testing.T) {
	m := NewSignalFlowMonitor(5)
	for i := 0; i < 4; i++ {
		m.OnCandleProcessed()
	}
	m.OnSignalGenerated("BUY")
	m.OnCandleProcessed()
	if m.IsBlocked() {
		t.Error("Should not be blocked after signal resets counter")
	}
}

func TestSignalFlow_BlockageAlert(t *testing.T) {
	m := NewSignalFlowMonitor(3)
	for i := 0; i < 2; i++ {
		m.OnCandleProcessed()
	}
	blocked := m.OnCandleProcessed()
	if !blocked {
		t.Error("Should return true on 3rd blank candle")
	}
}
