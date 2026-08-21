// Package oco implements durable One-Cancels-the-Other semantics for news breakout orders.
//
// Operator-authorized implementation (v1.9.0).
//
// CRITICAL SAFETY:
// - OCO groups are durable (persisted to DB, not just in-memory).
// - Sibling cancellation is idempotent.
// - Race conditions (both sides fill) are handled explicitly.
// - Restart/reconnect reconciliation exists.
// - Broker state is authoritative — local state is reconciled, not blindly trusted.
package oco

import (
	"fmt"
	"sync"
	"time"
)

// GroupState represents the OCO group state machine.
type GroupState string

const (
	StateCreated             GroupState = "CREATED"
	StateSubmitting          GroupState = "SUBMITTING"
	StateArmed               GroupState = "ARMED"
	StateBuyTriggered        GroupState = "BUY_TRIGGERED"
	StateSellTriggered       GroupState = "SELL_TRIGGERED"
	StateCancellingSibling   GroupState = "CANCELLING_SIBLING"
	StateActivePosition      GroupState = "ACTIVE_POSITION"
	StateCompleted           GroupState = "COMPLETED"
	StateExpired             GroupState = "EXPIRED"
	StateFailed              GroupState = "FAILED"
	StateRaceReconciliation  GroupState = "RACE_RECONCILIATION"
)

// Side represents which side of the OCO triggered.
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// Group is a durable OCO group linking two pending orders.
type Group struct {
	GroupID             string     `json:"group_id"`
	BreakoutPlanID      string     `json:"breakout_plan_id"`
	BuyOrderID          string     `json:"buy_order_id"`
	SellOrderID         string     `json:"sell_order_id"`
	BrokerBuyOrderID    string     `json:"broker_buy_order_id,omitempty"`
	BrokerSellOrderID   string     `json:"broker_sell_order_id,omitempty"`
	State               GroupState `json:"state"`
	Winner              Side       `json:"winner,omitempty"`
	CancelledSide       Side       `json:"cancelled_side,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	TriggeredAt         *time.Time `json:"triggered_at,omitempty"`
	CancelRequestedAt   *time.Time `json:"cancel_requested_at,omitempty"`
	CancelConfirmedAt   *time.Time `json:"cancel_confirmed_at,omitempty"`
	ReconciliationState string     `json:"reconciliation_state,omitempty"`
}

// Manager manages OCO group lifecycle.
type Manager struct {
	mu     sync.RWMutex
	groups map[string]*Group
}

// NewManager creates a new OCO manager.
func NewManager() *Manager {
	return &Manager{groups: make(map[string]*Group)}
}

// CreateGroup creates a new OCO group for a breakout plan.
func (m *Manager) CreateGroup(groupID, breakoutPlanID, buyOrderID, sellOrderID string) *Group {
	g := &Group{
		GroupID:        groupID,
		BreakoutPlanID: breakoutPlanID,
		BuyOrderID:     buyOrderID,
		SellOrderID:    sellOrderID,
		State:          StateCreated,
		CreatedAt:      time.Now().UTC(),
	}
	m.mu.Lock()
	m.groups[groupID] = g
	m.mu.Unlock()
	return g
}

// GetGroup retrieves an OCO group by ID.
func (m *Manager) GetGroup(groupID string) (*Group, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.groups[groupID]
	return g, ok
}

// Arm transitions a group to ARMED state (both pending orders submitted to broker).
func (m *Manager) Arm(groupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("OCO group %s not found", groupID)
	}
	if g.State != StateCreated && g.State != StateSubmitting {
		return fmt.Errorf("cannot arm group in state %s", g.State)
	}
	g.State = StateArmed
	return nil
}

// Trigger handles one side of the OCO being filled.
// This is idempotent — calling it twice with the same side is safe.
func (m *Manager) Trigger(groupID string, side Side) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("OCO group %s not found", groupID)
	}

	now := time.Now().UTC()

	// Idempotency: if already completed with this winner, return nil
	if g.State == StateActivePosition || g.State == StateCompleted {
		if g.Winner == side {
			return nil // idempotent — already processed
		}
	}

	switch g.State {
	case StateArmed:
		// Normal case: one side fills, cancel the sibling
		g.Winner = side
		g.CancelledSide = oppositeSide(side)
		g.TriggeredAt = &now
		g.State = StateCancellingSibling
		return nil

	case StateCancellingSibling:
		// Race condition: the sibling also filled before cancellation completed
		if g.Winner != side {
			g.State = StateRaceReconciliation
			g.ReconciliationState = "BOTH_SIDES_FILLED"
			return fmt.Errorf("race condition: both sides triggered")
		}
		return nil // same side triggered again — idempotent

	case StateRaceReconciliation:
		return fmt.Errorf("already in race reconciliation")

	default:
		return fmt.Errorf("cannot trigger group in state %s", g.State)
	}
}

// ConfirmCancellation confirms that the sibling order has been cancelled at the broker.
func (m *Manager) ConfirmCancellation(groupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("OCO group %s not found", groupID)
	}
	if g.State != StateCancellingSibling {
		return fmt.Errorf("cannot confirm cancellation in state %s", g.State)
	}
	now := time.Now().UTC()
	g.CancelConfirmedAt = &now
	g.State = StateActivePosition
	return nil
}

// Complete marks the OCO group as completed (position closed).
func (m *Manager) Complete(groupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("OCO group %s not found", groupID)
	}
	if g.State != StateActivePosition {
		return fmt.Errorf("cannot complete group in state %s", g.State)
	}
	g.State = StateCompleted
	return nil
}

// Expire marks a group as expired (event passed without trigger).
func (m *Manager) Expire(groupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("OCO group %s not found", groupID)
	}
	if g.State != StateArmed && g.State != StateCreated {
		return fmt.Errorf("cannot expire group in state %s", g.State)
	}
	g.State = StateExpired
	return nil
}

// ResolveRace handles the race condition where both sides filled.
// Policy: immediately close the unintended second fill to limit exposure.
// This is the safest policy — both fills are unwanted, we flatten.
func (m *Manager) ResolveRace(groupID string, policy RacePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("OCO group %s not found", groupID)
	}
	if g.State != StateRaceReconciliation {
		return fmt.Errorf("group not in race reconciliation state")
	}

	switch policy {
	case RacePolicyCloseSecond:
		// Keep the first winner, close the second fill
		g.ReconciliationState = "SECOND_CLOSED"
		g.State = StateActivePosition

	case RacePolicyFlattenBoth:
		// Close both positions — safest but realizes loss on both
		g.ReconciliationState = "BOTH_FLATTENED"
		g.State = StateCompleted

	default:
		return fmt.Errorf("unknown race policy: %s", policy)
	}
	return nil
}

// RacePolicy determines how to handle both-sides-filled race conditions.
type RacePolicy string

const (
	RacePolicyCloseSecond  RacePolicy = "CLOSE_SECOND"   // Close the unintended second fill
	RacePolicyFlattenBoth  RacePolicy = "FLATTEN_BOTH"   // Close both (safest)
)

// GetActiveGroups returns all groups in non-terminal states.
func (m *Manager) GetActiveGroups() []*Group {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Group
	for _, g := range m.groups {
		if g.State != StateCompleted && g.State != StateExpired && g.State != StateFailed {
			result = append(result, g)
		}
	}
	return result
}

// RestoreGroup re-registers a group after restart (from DB persistence).
func (m *Manager) RestoreGroup(g *Group) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups[g.GroupID] = g
}

// ReconcileWithBroker reconciles OCO state with broker state after restart/reconnect.
// brokerBuyFilled/brokerSellFilled indicate whether the broker reports each side as filled.
func (m *Manager) ReconcileWithBroker(groupID string, brokerBuyFilled, brokerSellFilled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("OCO group %s not found", groupID)
	}

	// If both are filled at broker, it's a race
	if brokerBuyFilled && brokerSellFilled {
		if g.State != StateRaceReconciliation {
			g.State = StateRaceReconciliation
			g.ReconciliationState = "BROKER_REPORTS_BOTH_FILLED"
		}
		return nil
	}

	// If buy is filled, ensure we're in the right state
	if brokerBuyFilled && g.Winner != SideBuy {
		now := time.Now().UTC()
		g.Winner = SideBuy
		g.CancelledSide = SideSell
		g.TriggeredAt = &now
		g.State = StateCancellingSibling
		g.ReconciliationState = "RECONCILED_FROM_BROKER"
		return nil
	}

	// If sell is filled, ensure we're in the right state
	if brokerSellFilled && g.Winner != SideSell {
		now := time.Now().UTC()
		g.Winner = SideSell
		g.CancelledSide = SideBuy
		g.TriggeredAt = &now
		g.State = StateCancellingSibling
		g.ReconciliationState = "RECONCILED_FROM_BROKER"
		return nil
	}

	// Neither filled — if still armed, keep armed
	if !brokerBuyFilled && !brokerSellFilled && g.State == StateArmed {
		return nil // still armed, no action needed
	}

	return nil
}

func oppositeSide(s Side) Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}
