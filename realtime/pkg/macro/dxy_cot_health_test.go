package macro

import (
	"testing"
)

func TestMacroHealth_DXYSuccess(t *testing.T) {
	m := NewMacroHealth()
	m.OnDXYFetchSuccess(98.5)
	val, ok := m.DXYValue()
	if !ok || val != 98.5 {
		t.Errorf("DXY should be available: val=%f ok=%v", val, ok)
	}
}

func TestMacroHealth_DXYFallback(t *testing.T) {
	m := NewMacroHealth()
	m.OnDXYFetchSuccess(98.5)
	m.OnDXYFetchFailure()
	m.OnDXYFetchFailure()
	m.OnDXYFetchFailure()
	val, ok := m.DXYValue()
	if ok {
		t.Error("DXY should be unavailable after 3 failures")
	}
	if val != 98.5 {
		t.Error("Should return last known good value even when unavailable")
	}
}

func TestMacroHealth_COTSuccess(t *testing.T) {
	m := NewMacroHealth()
	m.OnCOTFetchSuccess(141636)
	val, ok := m.COTValue()
	if !ok || val != 141636 {
		t.Errorf("COT should be available: val=%f ok=%v", val, ok)
	}
}

func TestMacroHealth_COTFailure(t *testing.T) {
	m := NewMacroHealth()
	m.OnCOTFetchSuccess(141636)
	for i := 0; i < 3; i++ {
		m.OnCOTFetchFailure()
	}
	_, ok := m.COTValue()
	if ok {
		t.Error("COT should be unavailable after 3 failures")
	}
}

func TestMacroHealth_IsHealthy(t *testing.T) {
	m := NewMacroHealth()
	m.OnDXYFetchSuccess(98.5)
	if !m.IsHealthy() {
		t.Error("Should be healthy with DXY available")
	}
}
