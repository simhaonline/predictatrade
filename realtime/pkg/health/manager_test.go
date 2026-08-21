package health

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/pkg/macro"
)

func TestManager_Healthy(t *testing.T) {
	stale := NewStaleChecker(60*time.Second, 120*time.Second)
	stale.UpdateLastCandleTime(time.Now())
	flow := NewSignalFlowMonitor(5)
	flow.OnSignalGenerated("BUY")
	mac := macro.NewMacroHealth()
	mac.OnDXYFetchSuccess(98.5)

	mgr := NewManager(stale, flow, mac)
	mgr.Update()
	if mgr.IsDegraded() {
		t.Error("Should not be degraded when all healthy")
	}
}

func TestManager_StaleData(t *testing.T) {
	stale := NewStaleChecker(60*time.Second, 120*time.Second)
	stale.UpdateLastCandleTime(time.Now().Add(-180 * time.Second))
	flow := NewSignalFlowMonitor(5)
	mac := macro.NewMacroHealth()

	mgr := NewManager(stale, flow, mac)
	mgr.Update()
	if !mgr.IsDegraded() {
		t.Error("Should be degraded with stale data")
	}
	if mgr.DegradedReason() != "STALE_DATA_CRITICAL" {
		t.Errorf("Reason: got %s, want STALE_DATA_CRITICAL", mgr.DegradedReason())
	}
}

func TestManager_SignalBlocked(t *testing.T) {
	stale := NewStaleChecker(60*time.Second, 120*time.Second)
	stale.UpdateLastCandleTime(time.Now())
	flow := NewSignalFlowMonitor(5)
	for i := 0; i < 5; i++ {
		flow.OnCandleProcessed()
	}
	mac := macro.NewMacroHealth()

	mgr := NewManager(stale, flow, mac)
	mgr.Update()
	if !mgr.IsDegraded() {
		t.Error("Should be degraded with blocked signal flow")
	}
}

func TestManager_SetDataStale(t *testing.T) {
	stale := NewStaleChecker(60*time.Second, 120*time.Second)
	stale.UpdateLastCandleTime(time.Now())
	flow := NewSignalFlowMonitor(5)
	mac := macro.NewMacroHealth()

	mgr := NewManager(stale, flow, mac)
	mgr.SetDataStale(true)
	if !mgr.IsDegraded() {
		t.Error("Should be degraded after SetDataStale(true)")
	}
	if mgr.DegradedReason() != "MT5_CONNECTION_LOST" {
		t.Errorf("Reason: got %s, want MT5_CONNECTION_LOST", mgr.DegradedReason())
	}
}

func TestManager_Recovery(t *testing.T) {
	stale := NewStaleChecker(60*time.Second, 120*time.Second)
	stale.UpdateLastCandleTime(time.Now().Add(-180 * time.Second))
	flow := NewSignalFlowMonitor(5)
	mac := macro.NewMacroHealth()

	mgr := NewManager(stale, flow, mac)
	mgr.Update()
	if !mgr.IsDegraded() {
		t.Error("Should be degraded initially")
	}

	// Recover: update candle time to now, set DXY
	stale.UpdateLastCandleTime(time.Now())
	flow.OnSignalGenerated("BUY")
	mac.OnDXYFetchSuccess(98.5)
	mgr.Update()
	if mgr.IsDegraded() {
		t.Error("Should recover after health improves")
	}
}
