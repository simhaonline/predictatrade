package agent

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

type Agent struct {
	config        *Config
	deviceID      string
	deviceKey     *ecdsa.PrivateKey
	mu            sync.Mutex
	conn          *websocket.Conn
	running       bool
	stopChan      chan struct{}
	signals       chan *SignalEvent
	heartbeat     time.Duration
	lastSignal    time.Time
	reconnectDelay time.Duration
	pipeManager   *PipeManager
	processedSignals map[string]bool  // idempotency: track processed signal IDs
}

type SignalEvent struct {
	EventID       string          `json:"event_id"`
	StreamID      string          `json:"stream_id"`
	Sequence      int64           `json:"sequence"`
	SchemaVersion string          `json:"schema_version"`
	Timestamp     time.Time       `json:"timestamp"`
	Type          string          `json:"type"`
	Priority      string          `json:"priority"`
	Payload       json.RawMessage  `json:"payload"`
}

type HeartbeatData struct {
	DeviceID         string    `json:"agent_id"`
	Version          string    `json:"version"`
	Hostname         string    `json:"hostname"`
	WindowsVersion   string    `json:"windows_version"`
	Status           string    `json:"status"` // ONLINE, DEGRADED, STALE, OFFLINE
	Timestamp        time.Time `json:"timestamp"`
	MasterConnected  bool      `json:"master_connected"`
	MT4Connected     bool      `json:"mt4_connected"`
	MT5Connected     bool      `json:"mt5_connected"`
	Broker           string    `json:"broker,omitempty"`
	AccountMasked    string    `json:"account_masked,omitempty"`
	BrokerSymbol     string    `json:"broker_symbol,omitempty"`
	CanonicalSymbol  string    `json:"canonical_symbol,omitempty"`
	LastTickAt       *time.Time `json:"last_tick_at,omitempty"`
	LatencyMs        int64     `json:"latency_ms"`
	ClockDriftMs     int64     `json:"clock_drift_ms"`
	AgentVersion     string    `json:"agent_version"`
	MTConnected      bool      `json:"mt_connected"`
}

func NewAgent(config *Config) *Agent {
	return &Agent{
		config:           config,
		deviceID:         uuid.New().String(),
		stopChan:         make(chan struct{}),
		signals:          make(chan *SignalEvent, 100),
		heartbeat:        30 * time.Second,
		reconnectDelay:   3 * time.Second,
		processedSignals: make(map[string]bool),
	}
}

func (a *Agent) Start() error {
	a.running = true

	// Load or create device identity
	if err := a.loadOrCreateDeviceKey(); err != nil {
		return fmt.Errorf("device key: %w", err)
	}
	log.Printf("Device ID: %s", a.deviceID)

	// Initialize named pipe manager for MT4/MT5 EA communication
	a.pipeManager = NewPipeManager(findCommonFolder(), a.sendToServer, a.config.APIURL)
	a.pipeManager.SetCallbacks(a.onTickFromEA, a.onLicenseCheck)
	a.pipeManager.Start()
	log.Printf("File IPC started at: %s", findCommonFolder())

	// Connect to live.predictatrade.com WebSocket
	go a.connectLoop()

	// Start heartbeat
	go a.heartbeatLoop()

	// Start signal processor (receives signals from server → forwards to EA)
	go a.processSignals()

	return nil
}

// activateDevice calls the backend activation API with the hardware fingerprint.
func (a *Agent) activateDevice(fp *HardwareFingerprint) error {
	activationReq := map[string]interface{}{
		"license_key": a.config.LicenseKey,
		"client_type": "MT5",
		"fingerprint": fp,
		"terminal": map[string]string{
			"name":       "PredictATrade Agent",
			"ea_version": "1.02",
		},
	}

	body, _ := json.Marshal(activationReq)
	resp, err := http.Post(a.config.APIURL+"/devices/activate", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("activation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("activation failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if v, ok := result["device_id"].(string); ok {
		a.config.DeviceID = v
	}
	if v, ok := result["session_id"].(string); ok {
		a.config.SessionID = v
	}
	if v, ok := result["device_secret"].(string); ok {
		a.config.DeviceSecret = v
	}
	if v, ok := result["refresh_token"].(string); ok {
		a.config.RefreshToken = v
	}
	if v, ok := result["access_token"].(string); ok {
		a.config.AccessToken = v
	}

	a.saveDeviceCredentials()
	return nil
}

func (a *Agent) saveDeviceCredentials() {
	credPath := filepath.Join(a.config.AgentDataDir, "device_creds.json")
	creds := map[string]string{
		"device_id":     a.config.DeviceID,
		"session_id":    a.config.SessionID,
		"refresh_token": a.config.RefreshToken,
	}
	data, _ := json.Marshal(creds)
	os.WriteFile(credPath, data, 0600)
}

func (a *Agent) loadDeviceCredentials() bool {
	credPath := filepath.Join(a.config.AgentDataDir, "device_creds.json")
	data, err := os.ReadFile(credPath)
	if err != nil {
		return false
	}
	var creds map[string]string
	json.Unmarshal(data, &creds)
	a.config.DeviceID = creds["device_id"]
	a.config.SessionID = creds["session_id"]
	a.config.RefreshToken = creds["refresh_token"]
	return a.config.DeviceID != ""
}

func (a *Agent) refreshToken() error {
	reqBody, _ := json.Marshal(map[string]string{
		"refresh_token": a.config.RefreshToken,
		"device_id":     a.config.DeviceID,
	})
	resp, err := http.Post(a.config.APIURL+"/devices/refresh", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("refresh failed (HTTP %d)", resp.StatusCode)
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if v, ok := result["access_token"].(string); ok {
		a.config.AccessToken = v
	}
	if v, ok := result["refresh_token"].(string); ok {
		a.config.RefreshToken = v
	}
	a.saveDeviceCredentials()
	return nil
}

func (a *Agent) sendHeartbeatToBackend() error {
	if a.config.DeviceID == "" {
		return nil // Not activated yet
	}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"device_id":   a.config.DeviceID,
		"session_id":  a.config.SessionID,
		"mt_connected": a.pipeManager != nil,
	})
	resp, err := http.Post(a.config.APIURL+"/devices/heartbeat", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("heartbeat failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func (a *Agent) Stop() {
	a.running = false
	close(a.stopChan)
	if a.pipeManager != nil {
		a.pipeManager.Stop()
	}
	a.mu.Lock()
	if a.conn != nil {
		a.conn.Close()
	}
	a.mu.Unlock()
}

func (a *Agent) loadOrCreateDeviceKey() error {
	keyPath := a.config.DeviceKeyPath
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		keyPath = filepath.Join(os.TempDir(), "pat_device.key")
	}

	if data, err := os.ReadFile(keyPath); err == nil {
		a.deviceID = string(data[:36])
		log.Printf("Loaded device identity: %s", a.deviceID)
		return nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	if err != nil {
		return err
	}
	a.deviceKey = key

	if err := os.WriteFile(keyPath, []byte(a.deviceID), 0600); err != nil {
		log.Printf("Warning: could not save device key: %v", err)
	}
	log.Printf("Generated new device identity: %s", a.deviceID)
	return nil
}

// connectLoop maintains WebSocket connection to live.predictatrade.com
func (a *Agent) connectLoop() {
	for {
		select {
		case <-a.stopChan:
			return
		default:
		}

		err := a.connect()
		if err != nil {
			log.Printf("WARN: Connection failed: %v, retrying in %v", err, a.reconnectDelay)
			select {
			case <-time.After(a.reconnectDelay + time.Duration(mrand.Intn(int(a.reconnectDelay/2)))):
				// Bounded exponential backoff: 1s → 2s → 5s → 10s → 30s with jitter
				backoffSteps := []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second}
				stepIdx := 0
				for i, step := range backoffSteps {
					if a.reconnectDelay < step {
						stepIdx = i
						break
					}
					stepIdx = i
				}
				if stepIdx < len(backoffSteps)-1 {
					a.reconnectDelay = backoffSteps[stepIdx+1]
				} else {
					a.reconnectDelay = backoffSteps[len(backoffSteps)-1]
				}
			case <-a.stopChan:
				return
			}
			log.Printf("INFO: Reconnecting to backend (delay: %v)...", a.reconnectDelay)
			continue
		}
		a.reconnectDelay = 3 * time.Second
	}
}

func (a *Agent) connect() error {
	url := a.config.LiveWSURL + "?agentId=" + a.deviceID + "&agentVersion=1.0.0"
	log.Printf("Connecting to %s", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	log.Printf("Connected to live.predictatrade.com")

	// Read loop — receives signals and messages from Go RT server
	go func() {
		defer func() {
			a.mu.Lock()
			a.conn = nil
			a.mu.Unlock()
			conn.Close()
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Connection lost: %v", err)
				return
			}

			// Parse the event
			var event SignalEvent
			if err := json.Unmarshal(msg, &event); err != nil {
				// Might be a tick ack or other message
				continue
			}

			// Handle signal events
			if event.Type == "SIGNAL" {
				// Idempotency check
				if a.processedSignals[event.EventID] {
					log.Printf("Duplicate signal ignored: %s", event.EventID)
					continue
				}
				a.processedSignals[event.EventID] = true

				a.lastSignal = time.Now()
				select {
				case a.signals <- &event:
				default:
					log.Printf("Signal buffer full, dropping: %s", event.EventID)
				}
			} else if event.Type == "ERROR" || event.Type == "DENIAL" {
				// P1-001: Distinguish distinct failure types — never conflate
				// auth failures with signal halts, license issues, etc.
				var errPayload struct {
					ErrorCode    string `json:"error_code"`
					Reason       string `json:"reason"`
					SignalID     string `json:"signal_id,omitempty"`
					StrategyID   string `json:"strategy_id,omitempty"`
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
	}()

	return nil
}

// sendToServer sends data (ticks, acks) to the Go RT server via WebSocket
func (a *Agent) sendToServer(data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn == nil {
		return fmt.Errorf("not connected")
	}
	return a.conn.WriteMessage(websocket.TextMessage, data)
}

// onTickFromEA is called when the EA sends real MT5 tick data via named pipe
func (a *Agent) onTickFromEA(tick MT5Tick) {
	// Forward tick to Go RT server via WebSocket
	data, err := MarshalTick(tick)
	if err != nil {
		return
	}
	a.sendToServer(data)
}

// onLicenseCheck is called when the EA requests license validation
func (a *Agent) onLicenseCheck(msg LicenseCheckMsg) {
	log.Printf("License check requested: account=%s broker=%s", msg.Account, msg.Broker)
	// In production: send HTTP request to https://api.predictatrade.com/api/v1/licensing/validate
	// For now, send a default ACTIVE response back to EA
	response := LicenseResponse{
		Type:     "LICENSE_RESPONSE",
		Status:   "ACTIVE",
		Plan:     "ELITE",
		Key: msg.LicenseKey,
	}
	respData, _ := json.Marshal(response)
	if a.pipeManager != nil {
		a.pipeManager.WriteToPipe("LICENSE_RESPONSE", string(respData))
	}
}

func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(a.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.mu.Lock()
			conn := a.conn
			a.mu.Unlock()
			if conn == nil {
				continue
			}

			hb := HeartbeatData{
				DeviceID:        a.deviceID,
				Version:         "1.0.0",
				Hostname:        hostname(),
				WindowsVersion:  "windows",
				Status:          "ONLINE",
				Timestamp:       time.Now().UTC(),
				MasterConnected: conn != nil,
				MT4Connected:    a.pipeManager != nil && a.pipeManager.MT4Connected(),
				MT5Connected:    a.pipeManager != nil && a.pipeManager.MT5Connected(),
				Broker:          a.config.BrokerName,
				CanonicalSymbol: "XAUUSD",
				AgentVersion:    "1.0.0",
				MTConnected:     a.pipeManager != nil,
				LatencyMs:       time.Since(time.Now().Add(-a.heartbeat)).Milliseconds(),
			}
			data, _ := json.Marshal(hb)
			conn.WriteMessage(websocket.TextMessage, data)

			// Also send heartbeat to EA via pipe
			if a.pipeManager != nil {
			}

		case <-a.stopChan:
			return
		}
	}
}

// processSignals handles received signals and forwards them to the MT4/MT5 EA
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

// hostname returns the machine hostname for heartbeat.
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
