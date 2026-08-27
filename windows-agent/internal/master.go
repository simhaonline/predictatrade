package agent

import "log"

// masterHandler implements serverMessageHandler for the data-only (Master Node)
// agent. The Master Node never executes trades; it only forwards market
// snapshots from the EA to the engine. All execution server messages are
// ignored — this is a hard fail-closed boundary: a data agent cannot trade.
type masterHandler struct{}

// startRoleLoops is a no-op for the Master Node: it has no signal processor.
func (masterHandler) startRoleLoops(a *Agent) {}

// handleServerMessage ignores every server message. The Master Node has no
// execution concern (SIGNAL, CLOSE_POSITION, EMERGENCY_STOP, KILL_SWITCH,
// LICENSE_STATUS, ERROR) — those are strictly Client (execution) concerns.
func (masterHandler) handleServerMessage(a *Agent, event SignalEvent) {
	if event.Type != "" {
		log.Printf("DATA mode: ignoring server message type=%s", event.Type)
	}
}
