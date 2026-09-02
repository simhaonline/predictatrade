package gateway

// ingest_http.go — EA-direct HTTP ingest endpoint (Option B, v1.19.0).
//
// The Windows-agent WebSocket transport (ws/v1/agent, ws/v1/data) is REMOVED.
// Customer MT4/MT5 EAs now POST their messages (ticks, market snapshots,
// heartbeats, MASTER_INIT, TRADE_RESULT, EXECUTION_ACK) directly to:
//
//	POST /ingest/agent?agentId=<device-uuid>&role=data|exec
//	Authorization: Bearer <device access token>   (control-plane device JWT)
//	Content-Type: application/json                 (single object)
//	Content-Type: application/x-ndjson             (batch: one JSON object per line)
//
// Authentication is IDENTICAL to the old WS handshake: a per-device JWT minted
// by the control plane at device activation, verified locally with JWT_SECRET
// (validateAgentJWT). The connection identity (sub claim) binds to the
// agentId — a token for device A cannot ingest as device B.
//
// Messages are dispatched into the SAME AgentProvider.HandleAgentMessage
// pipeline the WS transport used, so all downstream processing (offset
// detection, candle merge, Valkey cache, state manager, bulkheads) is
// unchanged. Only the transport changed.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// maxIngestBody bounds a single ingest request (a 200-bar multi-TF snapshot
// with indicators is ~200KB; 4MB leaves generous headroom for batches).
const maxIngestBody = 4 << 20 // 4MB

// ingestRateLimit per request batch: EAs post at most this many messages per
// call; larger batches are rejected (EA should chunk).
const maxIngestBatch = 200

// IngestProvider is the subset of the market-data provider the ingest
// endpoint needs. *marketdata.AgentProvider satisfies it.
type IngestProvider interface {
	HandleAgentMessage(agentID string, data []byte)
	IsDataNode(agentID string) bool
}

// agentAuthJWT extracts and verifies the device JWT from the request
// (Authorization: Bearer header or ?token= query — same acceptance as the old
// WS handshake) and returns (subject=deviceId, role, ok).
func agentAuthJWT(r *http.Request) (string, string, bool) {
	var raw string
	if ah := r.Header.Get("Authorization"); strings.HasPrefix(ah, "Bearer ") {
		raw = strings.TrimPrefix(ah, "Bearer ")
	} else if t := r.URL.Query().Get("token"); t != "" {
		raw = t
	}
	if raw == "" {
		// No token material at all — log the header SHAPE (lengths only, never
		// the value) so a client sending an empty/missing Bearer is visible
		// without exposing secrets (live 401 storm 2026-09-02).
		ah := r.Header.Get("Authorization")
		log.Printf("[INGEST-AUTH] no token: authz_len=%d prefix_bearer=%v remote=%s",
			len(ah), strings.HasPrefix(ah, "Bearer "), r.RemoteAddr)
		return "", "", false
	}
	sub, role, err := validateAgentJWTRole(raw)
	if err != nil || sub == "" {
		// Token present but invalid — log the verification error reason plus
		// the token's SHAPE (length + dot count only, never the value) so a
		// client carrying the wrong credential type (device secret, refresh
		// token, truncated JWT) is immediately identifiable (2026-09-02).
		log.Printf("[INGEST-AUTH] token rejected: %v token_len=%d dots=%d remote=%s",
			err, len(raw), strings.Count(raw, "."), r.RemoteAddr)
		return "", "", false
	}
	return sub, role, true
}

// validateAgentJWTRole verifies the token and returns (sub, role).
func validateAgentJWTRole(token string) (string, string, error) {
	sub, role, err := validateJWTFull(token)
	if err != nil || sub == "" {
		return "", "", err
	}
	return sub, role, nil
}

// HandleIngest serves POST /ingest/agent.
func (h *HTTPServer) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// ── Authentication: device JWT (same credential model as the old WS) ──
	sub, role, ok := agentAuthJWT(r)
	if !ok || sub == "" {
		http.Error(w, `{"error":"device authentication required"}`, http.StatusUnauthorized)
		return
	}
	if role != "data" && role != "exec" {
		http.Error(w, `{"error":"role required (data|exec)"}`, http.StatusBadRequest)
		return
	}

	// Bind the identity to the verified credential. The query param is a
	// convenience echo; a mismatch is rejected so a token cannot post as
	// another device.
	agentID := r.URL.Query().Get("agentId")
	if agentID == "" {
		agentID = sub
	} else if agentID != sub {
		http.Error(w, `{"error":"agentId does not match token subject"}`, http.StatusForbidden)
		return
	}

	// Register the authenticated role with the market-data provider. The old
	// WS handshake called SetAgentRole on connect; the HTTP transport must do
	// the equivalent on every request (idempotent, once-"data"-always-"data").
	// WITHOUT THIS the role map stays empty, IsDataNode() is always false, and
	// the Master's MARKET_SNAPSHOT/MASTER_TICK messages are silently dropped
	// in HandleAgentMessage — engine shows "Data feed outage" while ingest
	// returns 200 (observed live 2026-09-02).
	if h.roleRegistrar != nil {
		h.roleRegistrar(agentID, role)
	}

	if h.ingestProvider == nil {
		http.Error(w, `{"error":"ingest not available"}`, http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxIngestBody+1))
	if err != nil {
		http.Error(w, `{"error":"read body failed"}`, http.StatusBadRequest)
		return
	}
	if len(body) > maxIngestBody {
		http.Error(w, `{"error":"body too large"}`, http.StatusRequestEntityTooLarge)
		return
	}

	// JSONL batch or single JSON object.
	accepted, processed := 0, 0
	trim := func(b []byte) []byte { return bytes.TrimSpace(b) }
	if ct := r.Header.Get("Content-Type"); bytes.Contains([]byte(ct), []byte("x-ndjson")) ||
		(len(trim(body)) > 0 && trim(body)[0] != '{' && trim(body)[0] != '[') {
		sc := bufio.NewScanner(bytes.NewReader(body))
		sc.Buffer(make([]byte, 0, 64*1024), maxIngestBody)
		for sc.Scan() {
			line := trim(sc.Bytes())
			if len(line) == 0 {
				continue
			}
			accepted++
			if accepted > maxIngestBatch {
				break
			}
			h.ingestProvider.HandleAgentMessage(agentID, line)
			processed++
		}
	} else {
		obj := trim(body)
		if len(obj) > 0 && obj[0] == '[' {
			var batch []json.RawMessage
			if err := json.Unmarshal(obj, &batch); err != nil {
				http.Error(w, `{"error":"invalid JSON array"}`, http.StatusBadRequest)
				return
			}
			for i, item := range batch {
				if i >= maxIngestBatch {
					break
				}
				accepted++
				h.ingestProvider.HandleAgentMessage(agentID, item)
				processed++
			}
		} else {
			accepted = 1
			h.ingestProvider.HandleAgentMessage(agentID, obj)
			processed = 1
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":          true,
		"accepted":    accepted,
		"processed":   processed,
		"agent_id":    agentID,
		"server_time": time.Now().UTC().Format(time.RFC3339),
	})
}

// RegisterIngestRoute wires the EA-direct ingest endpoint. Called from
// registerRoutes when an agent provider is active.
func (h *HTTPServer) RegisterIngestRoute() {
	h.mux.HandleFunc("/ingest/agent", h.HandleIngest)
}

// edgeDevicesOnline reports whether any EA-direct device has checked in
// (edge-poll, edge-heartbeat, or ingest) within the last 120 seconds. This is
// the connection-liveness signal that the old WS hub used to provide.
func (h *HTTPServer) edgeDevicesOnline() bool {
	if h.persister == nil {
		return false
	}
	var n int
	err := h.persister.GetDB().QueryRow(
		`SELECT count(*) FROM licensing.edge_device_state
		  WHERE GREATEST(COALESCE(last_poll_at, to_timestamp(0)),
		                 COALESCE(last_ack_at, to_timestamp(0)),
		                 COALESCE(last_heartbeat_at, to_timestamp(0))) > now() - interval '120 seconds'`).Scan(&n)
	if err != nil {
		return false
	}
	return n > 0
}

// queueCommand enqueues a server command for EA-direct devices by writing a
// PENDING row into licensing.edge_signal_queue. signal_id carries the command
// name (EMERGENCY_STOP / KILL_SWITCH / CLOSE_ALL / RESUME); payload carries the
// envelope. Best-effort: errors are logged by the caller.
func (h *HTTPServer) queueCommand(command string, envelope map[string]interface{}) error {
	if h.persister == nil {
		return fmt.Errorf("no database connection")
	}
	if envelope == nil {
		envelope = map[string]interface{}{}
	}
	envelope["type"] = "SERVER_COMMAND"
	envelope["command"] = command
	envelope["issued_at"] = time.Now().UTC().Format(time.RFC3339)
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = h.persister.GetDB().Exec(
		`INSERT INTO licensing.edge_signal_queue (device_id, signal_id, payload)
		 SELECT d.id, $1, $2::jsonb
		   FROM licensing.devices d
		  WHERE d.revoked_at IS NULL
		    AND d.connection_status = 'ONLINE'
		    AND NOT EXISTS (
		          SELECT 1 FROM licensing.edge_signal_queue q
		           WHERE q.device_id = d.id AND q.signal_id = $1
		             AND q.status IN ('PENDING','IN_FLIGHT'))`,
		command, string(payload))
	return err
}