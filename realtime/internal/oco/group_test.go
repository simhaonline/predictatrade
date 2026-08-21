package oco

import (
	"testing"
	"time"
)

func TestOCO_CreateAndArm(t *testing.T) {
	m := NewManager()
	g := m.CreateGroup("g1", "plan1", "buy1", "sell1")
	if g.State != StateCreated {
		t.Fatalf("expected CREATED, got %s", g.State)
	}
	if err := m.Arm("g1"); err != nil {
		t.Fatalf("arm failed: %v", err)
	}
	g2, _ := m.GetGroup("g1")
	if g2.State != StateArmed {
		t.Fatalf("expected ARMED, got %s", g2.State)
	}
}

func TestOCO_BuyTriggered_SiblingCancelled(t *testing.T) {
	m := NewManager()
	m.CreateGroup("g1", "plan1", "buy1", "sell1")
	m.Arm("g1")

	// Buy side triggers
	if err := m.Trigger("g1", SideBuy); err != nil {
		t.Fatalf("trigger failed: %v", err)
	}
	g, _ := m.GetGroup("g1")
	if g.State != StateCancellingSibling {
		t.Fatalf("expected CANCELLING_SIBLING, got %s", g.State)
	}
	if g.Winner != SideBuy {
		t.Fatal("expected winner BUY")
	}
	if g.CancelledSide != SideSell {
		t.Fatal("expected cancelled SELL")
	}

	// Confirm cancellation
	if err := m.ConfirmCancellation("g1"); err != nil {
		t.Fatalf("confirm cancellation failed: %v", err)
	}
	g, _ = m.GetGroup("g1")
	if g.State != StateActivePosition {
		t.Fatalf("expected ACTIVE_POSITION, got %s", g.State)
	}

	// Complete
	if err := m.Complete("g1"); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	g, _ = m.GetGroup("g1")
	if g.State != StateCompleted {
		t.Fatalf("expected COMPLETED, got %s", g.State)
	}
}

func TestOCO_SellTriggered(t *testing.T) {
	m := NewManager()
	m.CreateGroup("g1", "plan1", "buy1", "sell1")
	m.Arm("g1")

	if err := m.Trigger("g1", SideSell); err != nil {
		t.Fatalf("trigger failed: %v", err)
	}
	g, _ := m.GetGroup("g1")
	if g.Winner != SideSell || g.CancelledSide != SideBuy {
		t.Fatalf("expected winner SELL, cancelled BUY")
	}
}

func TestOCO_IdempotentTrigger(t *testing.T) {
	m := NewManager()
	m.CreateGroup("g1", "plan1", "buy1", "sell1")
	m.Arm("g1")

	// Trigger buy
	m.Trigger("g1", SideBuy)
	m.ConfirmCancellation("g1")
	m.Complete("g1")

	// Trigger buy again — should be idempotent (no error)
	if err := m.Trigger("g1", SideBuy); err != nil {
		t.Fatalf("idempotent trigger should not error: %v", err)
	}
}

func TestOCO_RaceCondition_BothTriggered(t *testing.T) {
	m := NewManager()
	m.CreateGroup("g1", "plan1", "buy1", "sell1")
	m.Arm("g1")

	// Buy triggers first
	m.Trigger("g1", SideBuy)

	// Sell triggers before cancellation completes — race condition
	err := m.Trigger("g1", SideSell)
	if err == nil {
		t.Fatal("expected race condition error")
	}
	g, _ := m.GetGroup("g1")
	if g.State != StateRaceReconciliation {
		t.Fatalf("expected RACE_RECONCILIATION, got %s", g.State)
	}

	// Resolve with flatten both policy (safest)
	if err := m.ResolveRace("g1", RacePolicyFlattenBoth); err != nil {
		t.Fatalf("resolve race failed: %v", err)
	}
	g, _ = m.GetGroup("g1")
	if g.State != StateCompleted {
		t.Fatalf("expected COMPLETED after flatten, got %s", g.State)
	}
	if g.ReconciliationState != "BOTH_FLATTENED" {
		t.Fatalf("expected BOTH_FLATTENED, got %s", g.ReconciliationState)
	}
}

func TestOCO_RaceCondition_CloseSecondPolicy(t *testing.T) {
	m := NewManager()
	m.CreateGroup("g1", "plan1", "buy1", "sell1")
	m.Arm("g1")

	m.Trigger("g1", SideBuy)
	m.Trigger("g1", SideSell) // race

	if err := m.ResolveRace("g1", RacePolicyCloseSecond); err != nil {
		t.Fatalf("resolve race failed: %v", err)
	}
	g, _ := m.GetGroup("g1")
	if g.State != StateActivePosition {
		t.Fatalf("expected ACTIVE_POSITION after close second, got %s", g.State)
	}
}

func TestOCO_Expire(t *testing.T) {
	m := NewManager()
	m.CreateGroup("g1", "plan1", "buy1", "sell1")
	m.Arm("g1")

	if err := m.Expire("g1"); err != nil {
		t.Fatalf("expire failed: %v", err)
	}
	g, _ := m.GetGroup("g1")
	if g.State != StateExpired {
		t.Fatalf("expected EXPIRED, got %s", g.State)
	}
}

func TestOCO_RestartReconciliation(t *testing.T) {
	m := NewManager()
	g := &Group{
		GroupID:        "g1",
		BreakoutPlanID: "plan1",
		BuyOrderID:     "buy1",
		SellOrderID:    "sell1",
		State:          StateArmed,
		CreatedAt:      time.Now().UTC(),
	}
	m.RestoreGroup(g)

	// Simulate restart: broker reports buy filled
	if err := m.ReconcileWithBroker("g1", true, false); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	g2, _ := m.GetGroup("g1")
	if g2.State != StateCancellingSibling {
		t.Fatalf("expected CANCELLING_SIBLING after reconcile, got %s", g2.State)
	}
	if g2.Winner != SideBuy {
		t.Fatal("expected winner BUY after reconcile")
	}
}

func TestOCO_RestartReconciliation_BothFilled(t *testing.T) {
	m := NewManager()
	g := &Group{
		GroupID:   "g1",
		State:     StateArmed,
		CreatedAt: time.Now().UTC(),
	}
	m.RestoreGroup(g)

	if err := m.ReconcileWithBroker("g1", true, true); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	g2, _ := m.GetGroup("g1")
	if g2.State != StateRaceReconciliation {
		t.Fatalf("expected RACE_RECONCILIATION, got %s", g2.State)
	}
}

func TestOCO_RestartReconciliation_NeitherFilled(t *testing.T) {
	m := NewManager()
	g := &Group{
		GroupID:   "g1",
		State:     StateArmed,
		CreatedAt: time.Now().UTC(),
	}
	m.RestoreGroup(g)

	if err := m.ReconcileWithBroker("g1", false, false); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	g2, _ := m.GetGroup("g1")
	if g2.State != StateArmed {
		t.Fatalf("expected ARMED (no change), got %s", g2.State)
	}
}

func TestOCO_GetActiveGroups(t *testing.T) {
	m := NewManager()
	m.CreateGroup("g1", "plan1", "buy1", "sell1")
	m.CreateGroup("g2", "plan2", "buy2", "sell2")
	m.Arm("g1")
	m.Arm("g2")
	m.Trigger("g1", SideBuy)
	m.ConfirmCancellation("g1")
	m.Complete("g1")

	active := m.GetActiveGroups()
	if len(active) != 1 {
		t.Fatalf("expected 1 active group, got %d", len(active))
	}
	if active[0].GroupID != "g2" {
		t.Fatalf("expected g2 to be active, got %s", active[0].GroupID)
	}
}
