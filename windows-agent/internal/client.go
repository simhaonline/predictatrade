package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// clientHandler implements serverMessageHandler for the execution (Client)
// agent. It receives trading signals from the engine and forwards execution
// commands (CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH) and license status to
// the MT4/MT5 EA.
type clientHandler struct{}

// startRoleLoops launches the signal processor that forwards server signals to
// the EA. This loop is Client-only; the Master Node starts no such loop.
func (clientHandler) startRoleLoops(a *Agent) {
	go a.safe(a.processSignals)
}

// handleServerMessage processes execution-relevant server messages.
func (clientHandler) handleServerMessage(a *Agent, event SignalEvent) {
	switch event.Type {
	case "SIGNAL":
		// Idempotency check
		if a.processedSignals[event.EventID] {
			log.Printf("Duplicate signal ignored: %s", event.EventID)
			return
		}
		a.processedSignals[event.EventID] = true

		a.lastSignal = time.Now()
		select {
		case a.signals <- &event:
		default:
			log.Printf("Signal buffer full, dropping: %s", event.EventID)
		}

	case "CLOSE_POSITION":
		var closePayload struct {
			Ticket   int64  `json:"ticket"`
			Magic    int64  `json:"magic"`
			Reason   string `json:"reason"`
			SignalID string `json:"signal_id"`
		}
		json.Unmarshal(event.Payload, &closePayload)
		log.Printf("CLOSE_POSITION from server: ticket=%d magic=%d reason=%s", closePayload.Ticket, closePayload.Magic, closePayload.Reason)
		if a.pipeManager != nil {
			a.pipeManager.WriteToPipe("CLOSE_POSITION", fmt.Sprintf(`{"ticket":%d,"magic":%d,"reason":"%s","signal_id":"%s"}`, closePayload.Ticket, closePayload.Magic, closePayload.Reason, closePayload.SignalID))
		}

	case "EMERGENCY_STOP":
		var emergencyPayload struct {
			Reason string `json:"reason"`
		}
		json.Unmarshal(event.Payload, &emergencyPayload)
		log.Printf("EMERGENCY_STOP from server: reason=%s", emergencyPayload.Reason)
		if a.pipeManager != nil {
			a.pipeManager.WriteToPipe("EMERGENCY_STOP", fmt.Sprintf(`{"reason":"%s"}`, emergencyPayload.Reason))
		}

	case "KILL_SWITCH":
		// W8: STOP forwarding signals and disconnect — do not keep running.
		log.Printf("KILL_SWITCH from server - halting agent and disconnecting")
		if a.pipeManager != nil {
			a.pipeManager.WriteToPipe("KILL_SWITCH", `{"reason":"SERVER_KILL_SWITCH"}`)
		}
		a.halt()
		return

	case "LICENSE_STATUS":
		var lic struct {
			Valid      bool     `json:"valid"`
			Status     string   `json:"status"`
			Plan       string   `json:"plan"`
			Strategies []string `json:"allowed_strategies"`
		}
		if json.Unmarshal(event.Payload, &lic) != nil {
			return
		}
		status := lic.Status
		if lic.Valid {
			status = "ACTIVE"
		}
		log.Printf("LICENSE_STATUS received: status=%s plan=%s", status, lic.Plan)
		if a.pipeManager != nil {
			plan := lic.Plan
			if plan == "" {
				plan = "ELITE"
			}
			a.pipeManager.SetLicenseResult(status, plan, lic.Strategies)
		}

	case "ERROR", "DENIAL":
		// P1-001: Distinguish distinct failure types — never conflate
		// auth failures with signal halts, license issues, etc.
		var errPayload struct {
			ErrorCode  string `json:"error_code"`
			Reason     string `json:"reason"`
			SignalID   string `json:"signal_id,omitempty"`
			StrategyID string `json:"strategy_id,omitempty"`
		}
		_ = json.Unmarshal(event.Payload, &errPayload)
		switch errPayload.ErrorCode {
		case "AUTH_TOKEN_EXPIRED", "AUTH_INVALID":
			log.Printf("AUTH FAILURE: %s — token refresh required", errPayload.ErrorCode)
			go a.refreshToken()
		case "LICENSE_EXPIRED", "LICENSE_DEVICE_MISMATCH":
			log.Printf("LICENSE FAILURE: %s — %s", errPayload.ErrorCode, errPayload.Reason)
		case "SUBSCRIPTION_NOT_ENTITLED":
			log.Printf("ENTITLEMENT DENIED: strategy=%s — %s", errPayload.StrategyID, errPayload.Reason)
		case "TERMINAL_DISCONNECTED":
			log.Printf("TERMINAL DISCONNECTED: %s", errPayload.Reason)
		case "ACCOUNT_STATE_STALE":
			log.Printf("ACCOUNT STATE STALE: %s", errPayload.Reason)
		case "SYSTEM_HALTED":
			log.Printf("SYSTEM HALTED: %s — all execution suspended", errPayload.Reason)
		case "SIGNAL_NOT_EXECUTABLE", "SIGNAL_EXPIRED":
			log.Printf("SIGNAL REJECTED: %s signal=%s — %s", errPayload.ErrorCode, errPayload.SignalID, errPayload.Reason)
		case "RISK_REJECTED":
			log.Printf("RISK REJECTED: signal=%s — %s", errPayload.SignalID, errPayload.Reason)
		case "MARKET_CLOSED":
			log.Printf("MARKET CLOSED: %s", errPayload.Reason)
		default:
			log.Printf("SERVER ERROR: code=%s reason=%s", errPayload.ErrorCode, errPayload.Reason)
		}
	}
}

// processSignals handles received signals and forwards them to the MT4/MT5 EA.
// It is the Client (execution) role's signal-delivery loop.
func (a *Agent) processSignals() {
	for {
		select {
		case event := <-a.signals:
			log.Printf("Signal received from server: type=%s priority=%s", event.Type, event.Priority)

			// Parse signal payload
			var signal map[string]interface{}
			if err := json.Unmarshal(event.Payload, &signal); err != nil {
				log.Printf("Failed to parse signal: %v", err)
				continue
			}

			direction, _ := signal["Direction"].(string)
			strategyID, _ := signal["StrategyID"].(string)
			log.Printf("Signal: %s %s (strategy: %s)", direction, signal["Symbol"], strategyID)

			// Forward signal to EA via named pipe
			if a.pipeManager != nil {
				a.pipeManager.SendSignalToEA(string(event.Payload))
				a.deliveryMu.Lock()
				a.signalsDelivered++
				a.lastSignalAt = time.Now()
				a.deliveryMu.Unlock()
			}

			// Send acknowledgement back to Go RT server
			ack := map[string]interface{}{
				"type":      "ACK",
				"signal_id": signal["ID"],
				"device_id": a.deviceID,
				"timestamp": time.Now().UTC(),
				"status":    "FORWARDED_TO_EA",
			}
			data, _ := json.Marshal(ack)
			a.sendToServer(data)

		case <-a.stopChan:
			return
		}
	}
}
