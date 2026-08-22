package agent

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// logf is a convenience wrapper for log.Printf.
func logf(format string, args ...any) {
	log.Printf(format, args...)
}

type Agent struct {
	config           *Config
	deviceID         string
	deviceKey        *ecdsa.PrivateKey
	mu               sync.Mutex
	conn             *websocket.Conn
	running          bool
	stopChan         chan struct{}
	signals          chan *SignalEvent
	heartbeat        time.Duration
	lastSignal       time.Time
	reconnectDelay   time.Duration
	pipeManager      *PipeManager
	health           *healthServer
	updater          *Updater
	processedSignals map[string]bool // idempotency: track processed signal IDs
	clockDriftMs     int64           // clock drift (server - local) in ms
}

type SignalEvent struct {
	EventID       string          `json:"event_id"`
	StreamID      string          `json:"stream_id"`
	Sequence      int64           `json:"sequence"`
	SchemaVersion string          `json:"schema_version"`
	Timestamp     time.Time       `json:"timestamp"`
	Type          string          `json:"type"`
	Priority      string          `json:"priority"`
	Payload       json.RawMessage `json:"payload"`
}

type HeartbeatData struct {
	DeviceID        string     `json:"agent_id"`
	Version         string     `json:"version"`
	Hostname        string     `json:"hostname"`
	WindowsVersion  string     `json:"windows_version"`
	Status          string     `json:"status"` // ONLINE, DEGRADED, STALE, OFFLINE
	Timestamp       time.Time  `json:"timestamp"`
	MasterConnected bool       `json:"master_connected"`
	MT4Connected    bool       `json:"mt4_connected"`
	MT5Connected    bool       `json:"mt5_connected"`
	Broker          string     `json:"broker,omitempty"`
	AccountMasked   string     `json:"account_masked,omitempty"`
	BrokerSymbol    string     `json:"broker_symbol,omitempty"`
	CanonicalSymbol string     `json:"canonical_symbol,omitempty"`
	LastTickAt      *time.Time `json:"last_tick_at,omitempty"`
	LatencyMs       int64      `json:"latency_ms"`
	ClockDriftMs    int64      `json:"clock_drift_ms"`
	AgentVersion    string     `json:"agent_version"`
	MTConnected     bool       `json:"mt_connected"`
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

	// Claim the local health port before starting IPC or WebSocket workers. A
	// second installed copy must fail here instead of creating duplicate
	// backend connections and producing misleading handshake failures.
	a.health = newHealthServer()
	if err := a.health.start(); err != nil {
		return fmt.Errorf("health endpoint: %w (another Agent process may already be running)", err)
	}

	// Initialize named pipe manager for MT4/MT5 EA communication
	a.pipeManager = NewPipeManager(findCommonFolder(), a.sendToServer, a.config.APIURL)
	a.pipeManager.SetCallbacks(a.onTickFromEA, a.onLicenseCheck)
	a.pipeManager.SetTerminalCallback(func(term TerminalInfo) {
		// Register each new terminal with the NestJS control plane
		go a.registerTerminalWithBackend(term)
	})
	a.pipeManager.Start()
	log.Printf("File IPC started at: %s", findCommonFolder())

	// Connect to live.predictatrade.com WebSocket
	go a.connectLoop()

	// Start heartbeat
	go a.heartbeatLoop()

	// Start signal processor (receives signals from server → forwards to EA)
	go a.processSignals()

	// Start auto-updater (checks for updates every hour)
	manifestURL := getEnv("PAT_UPDATE_MANIFEST_URL", "https://downloads.predictatrade.com/windows-agent/update-manifest.json")
	installDir := getEnv("PAT_INSTALL_DIR", "C:\\Program Files\\PredictATrade\\XAUUSD")
	a.updater = NewUpdater(manifestURL, AgentVersion, a.config.AgentDataDir, installDir, a.config.UpdateChannel)
	go a.updateLoop()

	return nil
}

// activateDevice calls the backend activation API with the hardware fingerprint.
// This registers the device and binds the license to this hardware.
func (a *Agent) activateDevice(fp *HardwareFingerprint) error {
	activationReq := map[string]interface{}{
		"license_key": a.config.LicenseKey,
		"client_type": "MT5", // Initial activation as MT5; individual terminals register separately
		"fingerprint": fp,
		"terminal": map[string]string{
			"name":       "PredictATrade Agent",
			"ea_version": "1.07",
		},
		"mt_account": map[string]string{
			"broker": a.config.BrokerName,
			"server": "",
			"login":  "",
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

// registerTerminalWithBackend registers an individual MT4/MT5 terminal with the NestJS control plane.
// This captures the broker name, account login, terminal build, and EA version for each connected terminal.
// Called when a new terminal sends a LICENSE_CHECK via the pipe.
func (a *Agent) registerTerminalWithBackend(term TerminalInfo) error {
	if a.config.LicenseKey == "" {
		log.Printf("Skipping terminal registration: no license key")
		return nil
	}

	fp := CollectFingerprint(a.config.AgentDataDir)

	activationReq := map[string]interface{}{
		"license_key": a.config.LicenseKey,
		"client_type": term.ClientType,
		"fingerprint": fp,
		"terminal": map[string]string{
			"name":       "PredictATrade Agent",
			"ea_version": "1.07",
			"build":      "",
		},
		"mt_account": map[string]string{
			"broker": term.Broker,
			"server": "",
			"login":  term.Account,
			"symbol": term.Symbol,
		},
	}

	body, _ := json.Marshal(activationReq)
	resp, err := http.Post(a.config.APIURL+"/devices/activate", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("terminal registration failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Terminal registration for %s/%s returned HTTP %d: %s", term.ClientType, term.Account, resp.StatusCode, string(respBody))
		// Don't return error — non-fatal, the agent continues operating
		return nil
	}

	log.Printf("Terminal registered: %s broker=%s account=%s", term.ClientType, term.Broker, term.Account)
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

	// Collect terminal info for heartbeat (including account data)
	var terminals []map[string]interface{}
	if a.pipeManager != nil {
		for _, t := range a.pipeManager.GetTerminals() {
			terminals = append(terminals, map[string]interface{}{
				"client_type":    t.ClientType,
				"broker":         t.Broker,
				"account":        t.Account,
				"symbol":         t.Symbol,
				"balance":        t.Balance,
				"equity":         t.Equity,
				"profit":         t.Profit,
				"currency":       t.Currency,
				"open_positions": t.OpenPositions,
				"buy_positions":  t.BuyPositions,
				"sell_positions": t.SellPositions,
				"total_lots":     t.TotalLots,
				"floating_pnl":   t.FloatingPnL,
			})
		}
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"device_id":    a.config.DeviceID,
		"session_id":   a.config.SessionID,
		"mt_connected": a.pipeManager != nil,
		"terminals":    terminals,
		"hostname":     hostname(),
	})

	req, err := http.NewRequest("POST", a.config.APIURL+"/devices/heartbeat", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Use access token if available
	if a.config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.config.AccessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Backend heartbeat HTTP %d: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("heartbeat failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func (a *Agent) Stop() {
	a.running = false
	close(a.stopChan)
	if a.health != nil {
		a.health.stop()
	}
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
	url := a.config.LiveWSURL + "?agentId=" + a.deviceID + "&agentVersion=" + AgentVersion
	log.Printf("Connecting to %s", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	log.Printf("Connected to live.predictatrade.com")

	// Read loop — receives signals and messages from Go RT server.
	// This runs synchronously so connectLoop cannot open a second WebSocket while
	// the current connection is still alive. Previously this was launched in a
	// goroutine and connect() returned nil immediately, causing a rapid dial loop
	// until Nginx rejected the agent with 503/connection-limit errors.
	defer func() {
		a.mu.Lock()
		if a.conn == conn {
			a.conn = nil
		}
		a.mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Connection lost: %v", err)
			return err
		}

		// Parse the event — check for ACK (contains server time for clock sync)
		var ackCheck struct {
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(msg, &ackCheck); err == nil {
			if ackCheck.Type == "ACK" || ackCheck.Type == "CONNECTED" {
				// Calculate clock drift: server_time - local_time
				if ackCheck.Timestamp != "" {
					if serverTime, err := time.Parse(time.RFC3339, ackCheck.Timestamp); err == nil {
						localTime := time.Now().UTC()
						a.mu.Lock()
						a.clockDriftMs = serverTime.Sub(localTime).Milliseconds()
						a.mu.Unlock()
					}
				}
				continue
			}
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
		Type:   "LICENSE_RESPONSE",
		Status: "ACTIVE",
		Plan:   "ELITE",
		Key:    msg.LicenseKey,
	}
	respData, _ := json.Marshal(response)
	if a.pipeManager != nil {
		a.pipeManager.WriteToPipe("LICENSE_RESPONSE", string(respData))
	}
}

// updateLoop periodically checks the download server for a new agent version.
// If an update is available, it downloads, verifies the checksum, and applies
// the update via a helper batch script that stops the service, swaps the binary,
// and restarts. The check runs every 1 hour (configurable via PAT_UPDATE_INTERVAL_MIN).
func (a *Agent) updateLoop() {
	intervalMin := 60
	if envInterval := getEnv("PAT_UPDATE_INTERVAL_MIN", ""); envInterval != "" {
		if v, err := strconv.Atoi(envInterval); err == nil && v > 0 {
			intervalMin = v
		}
	}

	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	defer ticker.Stop()

	// Initial check after 30 seconds (not immediately on startup)
	select {
	case <-a.stopChan:
		return
	case <-time.After(30 * time.Second):
	}

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.checkAndUpdate()
		}
	}
}

// checkAndUpdate performs a single update check and applies if available.
func (a *Agent) checkAndUpdate() {
	if a.updater == nil {
		return
	}

	manifest, err := a.updater.CheckForUpdate()
	if err != nil {
		logf("[updater] Update check failed: %v", err)
		return
	}
	if manifest == nil {
		logf("[updater] Already up to date (v%s)", AgentVersion)
		return
	}

	logf("[updater] Update available: v%s -> v%s (%s)", AgentVersion, manifest.Version, manifest.ReleaseNotes)

	// Download and verify
	stagedPath, err := a.updater.DownloadAndVerify(manifest)
	if err != nil {
		logf("[updater] Download/verify failed: %v", err)
		return
	}

	logf("[updater] Update verified (checksum OK), staged at %s", stagedPath)

	// Apply the update (Windows: helper batch script stops service, swaps, restarts)
	currentPath := filepath.Join(getEnv("PAT_INSTALL_DIR", "C:\\Program Files\\PredictATrade\\XAUUSD"), "agent.exe")
	if err := a.updater.ApplyUpdateOnWindows(stagedPath, currentPath, manifest); err != nil {
		logf("[updater] Apply failed: %v", err)
		return
	}

	logf("[updater] Update helper launched — service will restart shortly with v%s", manifest.Version)
	// The helper script will stop this process; no need to exit manually
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

			// Calculate clock drift: compare local UTC time with server time.
			// The server sends server_time in its responses. We track the
			// last known drift to include in heartbeat for monitoring.
			localNow := time.Now().UTC()
			hb := HeartbeatData{
				DeviceID:        a.deviceID,
				Version:         AgentVersion,
				Hostname:        hostname(),
				WindowsVersion:  "windows",
				Status:          "ONLINE",
				Timestamp:       localNow,
				MasterConnected: conn != nil,
				MT4Connected:    a.pipeManager != nil && a.pipeManager.MT4Connected(),
				MT5Connected:    a.pipeManager != nil && a.pipeManager.MT5Connected(),
				Broker:          a.config.BrokerName,
				CanonicalSymbol: "XAUUSD",
				AgentVersion:    AgentVersion,
				MTConnected:     a.pipeManager != nil,
				LatencyMs:       0, // Updated below after measuring round-trip
				ClockDriftMs:    a.clockDriftMs,
			}
			data, _ := json.Marshal(hb)
			sendStart := time.Now()
			conn.WriteMessage(websocket.TextMessage, data)

			// Measure round-trip latency from the ACK timestamp
			// The Go server responds with {"type":"ACK","timestamp":"<RFC3339>"}
			// We read it in the read loop, but we can estimate latency from
			// the time between sending and the next read.
			// For clock drift: compare server's ACK timestamp with our local time.
			latencyMs := time.Since(sendStart).Milliseconds()
			hb.LatencyMs = latencyMs

			// Log clock drift warning if significant
			absDrift := a.clockDriftMs
			if absDrift < 0 {
				absDrift = -absDrift
			}
			if absDrift > 120000 { // > 2 minutes
				log.Printf("WARN: Clock drift %dms — Windows clock may not be NTP-synced! Run 'w32tm /resync' to fix.", a.clockDriftMs)
			} else if absDrift > 30000 { // > 30 seconds
				log.Printf("WARN: Clock drift %dms — consider syncing Windows clock via NTP.", a.clockDriftMs)
			}

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
