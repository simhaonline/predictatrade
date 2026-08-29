package agent

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/tls"
	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

// logf is a convenience wrapper for log.Printf.
func logf(format string, args ...any) {
	log.Printf(format, args...)
}

type Agent struct {
	config           *Config
	role             string               // "exec" (Client) or "data" (Master Node) — data role never executes orders
	handler          serverMessageHandler // role-specific server-message behavior
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

	halted   bool      // W8: set by KILL_SWITCH — stop forwarding and disconnect
	stopOnce sync.Once // guards stopChan close against double-close

	statusMu      sync.RWMutex
	backendName   string // display URL of the backend the agent connects to
	startedAt     time.Time
	lastHeartbeat time.Time

	tickMu         sync.RWMutex
	lastXAUUSDTick *MT5Tick // most recent XAUUSD tick, used for heartbeat market status

	// Delivery telemetry (read by the local status page). Counters are guarded
	// by deliveryMu and intentionally separated by agent role:
	//   - data (Master Node) mode → candlesDelivered counts broker data messages
	//     forwarded to the engine (live candle delivery).
	//   - exec (Client) mode      → signalsDelivered counts signals forwarded to
	//     the MT4/MT5 EA (live signal delivery).
	deliveryMu       sync.Mutex
	candlesDelivered int64
	signalsDelivered int64
	lastCandleAt     time.Time
	lastSignalAt     time.Time
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
	DeviceID string `json:"agent_id"`
	// DeviceIDCP is the control-plane device id (licensing.devices.id) assigned
	// at activation. The engine uses it to correlate this live agent connection
	// to a dashboard-visible device row. Empty when activation has not completed.
	DeviceIDCP      string     `json:"device_id,omitempty"`
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
	AuthMAC         string     `json:"auth_mac,omitempty"` // W3: HMAC of agent_id|timestamp for server-side impersonation check
}

// NewMasterAgent constructs a data-only (Master Node) agent. It never executes
// orders and is deployed as pat-master.exe. It only forwards market snapshots
// from the EA to the engine.
func NewMasterAgent(config *Config) *Agent {
	config.Mode = "data"
	a := newBaseAgent(config, "data")
	a.handler = masterHandler{}
	return a
}

// NewClientAgent constructs the execution (Client) agent. It receives signals
// from the engine and forwards execution commands to the EA. Deployed as
// pat-agent.exe.
func NewClientAgent(config *Config) *Agent {
	config.Mode = "exec"
	a := newBaseAgent(config, "exec")
	a.handler = clientHandler{}
	return a
}

// newBaseAgent builds the role-agnostic Agent shared by both the Master Node
// and Client agents. Role-specific behavior is supplied via the handler.
func newBaseAgent(config *Config, role string) *Agent {
	return &Agent{
		config:           config,
		role:             role,
		deviceID:         uuid.New().String(),
		stopChan:         make(chan struct{}),
		signals:          make(chan *SignalEvent, 100),
		heartbeat:        30 * time.Second,
		reconnectDelay:   3 * time.Second,
		processedSignals: make(map[string]bool),
	}
}

// serverMessageHandler isolates the role-specific handling of server messages
// and startup loops so the Master Node and Client agents share one connection
// core without mode branching:
//   - masterHandler: data-only, ignores all execution server messages.
//   - clientHandler: handles SIGNAL/CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH/
//     LICENSE_STATUS/ERROR.
type serverMessageHandler interface {
	handleServerMessage(a *Agent, event SignalEvent)
	startRoleLoops(a *Agent)
}

// buildInfo is overridden at link time via -ldflags "-X ...buildInfo=...".
var buildInfo = "dev"

// safe runs fn, recovering from panics so a single transient failure cannot
// terminate the whole agent process (Windows Service stability).
func (a *Agent) safe(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[panic] recovered in agent routine: %v", r)
		}
	}()
	fn()
}

func maskSecret(s string) string {
	if len(s) <= 6 {
		return "******"
	}
	return s[:3] + "******" + s[len(s)-3:]
}

// AgentStatus is a live, read-only snapshot of the agent's relationship with
// the backend (SERVER) and the MT4/MT5 EA (CLIENT). It powers the local
// status page served on http://127.0.0.1:9000.
type AgentStatus struct {
	Version           string `json:"version"`
	DeviceID          string `json:"device_id"`
	Mode              string `json:"mode"` // "data" (Master Node) or "exec" (Client)
	UptimeSeconds     int64  `json:"uptime_seconds"`
	BackendURL        string `json:"backend_url"`
	BackendConnected  bool   `json:"backend_connected"`
	LicenseStatus     string `json:"license_status"`
	LicensePlan       string `json:"license_plan"`
	MT4Connected      bool   `json:"mt4_connected"`
	MT5Connected      bool   `json:"mt5_connected"`
	TerminalConnected bool   `json:"terminal_connected"`
	LastHeartbeat     string `json:"last_heartbeat"`
	LastSignal        string `json:"last_signal"`
	ClockDriftMs      int64  `json:"clock_drift_ms"`
	// Delivery telemetry (role-specific)
	CandlesDelivered    int64  `json:"candles_delivered"` // Master Node → engine
	LastCandleDelivered string `json:"last_candle_delivered"`
	SignalsDelivered    int64  `json:"signals_delivered"` // Client → EA
	LastSignalDelivered string `json:"last_signal_delivered"`
	GeneratedAt         string `json:"generated_at"`
}

// getStatus returns a consistent snapshot of the agent's live state.
func (a *Agent) getStatus() AgentStatus {
	a.statusMu.RLock()
	bname := a.backendName
	sa := a.startedAt
	lhb := a.lastHeartbeat
	a.statusMu.RUnlock()

	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()

	var mt4, mt5 bool
	var licStatus, licPlan string
	if a.pipeManager != nil {
		mt4 = a.pipeManager.MT4Connected()
		mt5 = a.pipeManager.MT5Connected()
		licStatus, licPlan = a.pipeManager.GetLicense()
		// A license verdict is only meaningful while a terminal (EA) is actually
		// connected and able to validate. Without a live MT4/MT5 link the cached
		// verdict is stale, so do not surface it as ACTIVE — it would mislead the
		// operator into thinking the account is licensed when nothing is connected.
		if !mt4 && !mt5 {
			licStatus = ""
			licPlan = ""
		}
	}

	a.deliveryMu.Lock()
	candlesDelivered := a.candlesDelivered
	signalsDelivered := a.signalsDelivered
	lastCandleAt := a.lastCandleAt
	lastSignalAt := a.lastSignalAt
	a.deliveryMu.Unlock()

	return AgentStatus{
		Version:             AgentVersion,
		DeviceID:            a.deviceID,
		Mode:                a.role,
		UptimeSeconds:       int64(time.Since(sa).Seconds()),
		BackendURL:          bname,
		BackendConnected:    conn != nil,
		LicenseStatus:       licStatus,
		LicensePlan:         licPlan,
		MT4Connected:        mt4,
		MT5Connected:        mt5,
		TerminalConnected:   mt4 || mt5,
		LastHeartbeat:       lhb.UTC().Format(time.RFC3339),
		LastSignal:          a.lastSignal.UTC().Format(time.RFC3339),
		ClockDriftMs:        a.clockDriftMs,
		CandlesDelivered:    candlesDelivered,
		LastCandleDelivered: lastCandleAt.UTC().Format(time.RFC3339),
		SignalsDelivered:    signalsDelivered,
		LastSignalDelivered: lastSignalAt.UTC().Format(time.RFC3339),
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
	}
}

func (a *Agent) Start() error {
	a.running = true

	// Load or create device identity — non-fatal if it fails
	if err := a.loadOrCreateDeviceKey(); err != nil {
		log.Printf("Warning: device key issue (non-fatal, using in-memory ID): %v", err)
		if a.deviceID == "" {
			a.deviceID = uuid.New().String()
		}
	}
	log.Printf("Device ID: %s", a.deviceID)

	// Record startup metadata for the local status page.
	a.statusMu.Lock()
	a.startedAt = time.Now()
	// The data (Master Node) role connects to the dedicated data WS; the exec
	// (Client) role connects to the live/exec WS. Show the correct one.
	if a.role == "data" {
		a.backendName = a.config.DataWSURL
	} else {
		a.backendName = a.config.LiveWSURL
	}
	a.statusMu.Unlock()

	// Startup diagnostics (helps Windows Service / Defender triage).
	exe, _ := os.Executable()
	log.Printf("=========================================================")
	log.Printf("Predict-A-Trade Windows Agent starting")
	log.Printf("  Version:          %s", AgentVersion)
	log.Printf("  Build:            %s", buildInfo)
	log.Printf("  OS / Arch:        %s / %s", runtime.GOOS, runtime.GOARCH)
	log.Printf("  Executable Path:  %s", exe)
	log.Printf("  Data Dir:         %s", a.config.AgentDataDir)
	log.Printf("  Service Mode:     %v", IsWindowsService())
	log.Printf("  Health Endpoint:  http://127.0.0.1:9000/health")
	log.Printf("=========================================================")

	// Claim the local health port before starting IPC or WebSocket workers. A
	// second installed copy must fail here instead of creating duplicate
	a.health = newHealthServer(a)
	a.health.start()

	// Initialize named pipe manager for MT4/MT5 EA communication
	commonDirs := findCommonFolders()
	a.pipeManager = NewPipeManager(commonDirs, a.sendToServer, a.config.APIURL, a.role)
	a.pipeManager.SetDeviceIDFn(func() string { return a.config.DeviceID })
	a.pipeManager.SetCallbacks(a.onTickFromEA, a.onLicenseCheck)
	a.pipeManager.SetTerminalCallback(func(term TerminalInfo) {
		// Register each new terminal with the NestJS control plane
		go a.registerTerminalWithBackend(term)
	})
	a.pipeManager.Start()
	log.Printf("File IPC started at %d folder(s): %v", len(commonDirs), commonDirs)

	// Connect to live.predictatrade.com WebSocket
	go a.safe(a.connectLoop)

	// Start heartbeat
	go a.safe(a.heartbeatLoop)

	// Start role-specific loops (Client launches the signal processor; the
	// Master Node is data-only and starts no extra loops).
	a.handler.startRoleLoops(a)

	// Start auto-updater (checks for updates every hour).
	// Each role fetches its ARCH-specific manifest so it always downloads the
	// correct binary for the platform (amd64/386/arm64). A per-role (amd64)
	// manifest is used as a fallback if the arch-specific one is missing.
	roleDir := "client"
	if a.role == "data" {
		roleDir = "master"
	}
	arch := runtime.GOARCH // amd64, 386, arm64
	defaultManifest := fmt.Sprintf("https://downloads.predictatrade.com/windows-agent/%s/%s/update-manifest.json", roleDir, arch)
	manifestURL := getEnv("PAT_UPDATE_MANIFEST_URL", defaultManifest)
	fallbackManifest := fmt.Sprintf("https://downloads.predictatrade.com/windows-agent/%s/update-manifest.json", roleDir)
	defaultSvc := "pat-agent-client"
	if a.role == "data" {
		defaultSvc = "pat-agent-master"
	}
	serviceName := getEnv("PAT_SERVICE_NAME", defaultSvc)
	a.updater = NewUpdater(manifestURL, fallbackManifest, AgentVersion, a.config.AgentDataDir, a.config.UpdateChannel, serviceName)
	go a.safe(a.updateLoop)
	go a.safe(a.telemetryLoop)

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

	fp, err := CollectFingerprint(a.config.AgentDataDir)
	if err != nil {
		// W10: fingerprint collection failed — do not register a meaningless
		// device binding. Log and skip registration (non-fatal for the agent).
		log.Printf("Terminal registration skipped: fingerprint collection failed: %v", err)
		return nil
	}

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
			term := map[string]interface{}{
				"client_type":    t.ClientType,
				"broker":         t.Broker,
				"server":         t.Server,
				"account":        t.Account,
				"symbol":         t.Symbol,
				"connected":      true,
				"balance":        t.Balance,
				"equity":         t.Equity,
				"profit":         t.Profit,
				"currency":       t.Currency,
				"leverage":       t.Leverage,
				"open_positions": t.OpenPositions,
				"buy_positions":  t.BuyPositions,
				"sell_positions": t.SellPositions,
				"total_lots":     t.TotalLots,
				"floating_pnl":   t.FloatingPnL,
			}
			// Attach genuine XAUUSD market status when this terminal trades XAUUSD
			// and we have a real tick. Do not fabricate missing data.
			if strings.EqualFold(t.Symbol, "XAUUSD") {
				a.tickMu.RLock()
				tk := a.lastXAUUSDTick
				a.tickMu.RUnlock()
				xau := map[string]interface{}{"symbol": "XAUUSD", "available": false}
				// Only publish real market data. A zero/inverted spread (e.g. before the
				// EA has a live tick, or a malformed pipe frame) must never be forwarded,
				// otherwise the terminal shows a 0.00 price and downstream analytics break.
				if tk != nil && tk.Bid > 0 && tk.Ask > 0 && tk.Ask >= tk.Bid {
					spread := tk.Ask - tk.Bid
					xau["available"] = true
					xau["bid"] = tk.Bid
					xau["ask"] = tk.Ask
					xau["spread"] = spread
					xau["last_tick_time"] = tk.Timestamp
				}
				term["xauusd"] = xau
			}
			terminals = append(terminals, term)
		}
	}

	a.statusMu.RLock()
	uptime := int64(0)
	if !a.startedAt.IsZero() {
		uptime = int64(time.Since(a.startedAt).Seconds())
	}
	a.statusMu.RUnlock()

	healthStatus := "ok"
	if !a.running {
		healthStatus = "stopped"
	} else if a.pipeManager == nil {
		healthStatus = "degraded"
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"device_id":            a.config.DeviceID,
		"session_id":           a.config.SessionID,
		"mt_connected":         a.pipeManager != nil,
		"terminals":            terminals,
		"hostname":             hostname(),
		"agent_version":        AgentVersion,
		"agent_started_at":     a.startedAt.Format(time.RFC3339),
		"agent_uptime_seconds": uptime,
		"os_name":              runtime.GOOS,
		"architecture":         runtime.GOARCH,
		"service_status":       serviceState(a.running),
		"health_status":        healthStatus,
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
	a.closeStop()
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

// closeStop safely closes the stop channel exactly once.
func (a *Agent) closeStop() {
	a.stopOnce.Do(func() { close(a.stopChan) })
}

// isHalted reports whether a KILL_SWITCH has been processed.
func (a *Agent) isHalted() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.halted
}

// halt stops all agent loops and disconnects after a server KILL_SWITCH.
// W8: the agent must STOP forwarding signals and disconnect, not keep running.
func (a *Agent) halt() {
	a.mu.Lock()
	if a.halted {
		a.mu.Unlock()
		return
	}
	a.halted = true
	a.mu.Unlock()

	log.Printf("KILL_SWITCH received — halting agent (stopping forwarding and disconnecting)")
	a.running = false
	a.closeStop()
	if a.pipeManager != nil {
		a.pipeManager.Stop()
	}
	if a.health != nil {
		a.health.stop()
	}
	a.mu.Lock()
	if a.conn != nil {
		a.conn.Close()
	}
	a.mu.Unlock()
}

// wsHMAC returns an HMAC-SHA256 over msg using PAT_WS_HMAC_SECRET, falling back
// to the IPC secret. W3: prevents WS impersonation of the agent.
func (a *Agent) wsHMAC(msg string) string {
	secret := strings.TrimSpace(os.Getenv("PAT_WS_HMAC_SECRET"))
	if secret == "" {
		secret = ipcHMACSecret()
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// sendWSHandshake sends a signed handshake so the server can reject spoofed
// agent connections. The server side verifies HMAC(agent_id|timestamp|nonce).
func (a *Agent) sendWSHandshake() {
	ts := time.Now().UTC().Format(time.RFC3339)
	nonce := uuid.New().String()
	mac := a.wsHMAC(a.deviceID + "|" + ts + "|" + nonce)
	handshake := map[string]interface{}{
		"type":      "HANDSHAKE",
		"agent_id":  a.deviceID,
		"timestamp": ts,
		"nonce":     nonce,
		"hmac":      mac,
		"version":   AgentVersion,
	}
	data, _ := json.Marshal(handshake)
	a.sendToServer(data)
}

func (a *Agent) loadOrCreateDeviceKey() error {
	keyPath := a.config.DeviceKeyPath
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		keyPath = filepath.Join(os.TempDir(), "pat_device.key")
		dir = filepath.Dir(keyPath)
		_ = os.MkdirAll(dir, 0700) // try temp dir
	}

	// Try to load existing device key
	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) >= 36 {
			a.deviceID = string(data[:36])
			log.Printf("Loaded device identity: %s", a.deviceID)
			return nil
		}
		// File exists but is corrupt/too short — regenerate
		log.Printf("Device key file corrupt (len=%d, need 36) — regenerating", len(data))
	}

	// Generate new device identity (a.deviceID was already set by newBaseAgent)
	if a.deviceID == "" {
		a.deviceID = uuid.New().String()
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	if err != nil {
		// Non-fatal: continue without ECDSA key (deviceID is still set)
		log.Printf("Warning: ECDSA key generation failed (non-fatal): %v", err)
	} else {
		a.deviceKey = key
	}

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
		if a.isHalted() {
			return
		}
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
	// Data-only agents connect to the dedicated data endpoint and advertise
	// role=data; execution agents use the exec endpoint with role=exec. This
	// lets the server isolate data collection from order execution.
	wsURL := a.config.LiveWSURL
	if a.role == "data" {
		wsURL = a.config.DataWSURL
	}
	url := wsURL + "?agentId=" + a.deviceID + "&agentVersion=" + AgentVersion + "&role=" + a.role
	log.Printf("Connecting to %s (role=%s)", url, a.role)

	// CRITICAL: Use a custom dialer that forces HTTP/1.1 via TLS ALPN.
	// The default dialer allows HTTP/2 negotiation, but HTTP/2 does NOT
	// support WebSocket upgrades (HTTP 101 Switching Protocols). When nginx
	// negotiates HTTP/2 via ALPN, the WebSocket connection fails silently —
	// nginx returns HTTP 200 with the default page instead of HTTP 101.
	// Forcing NextProtos=["http/1.1"] makes the TLS handshake negotiate
	// HTTP/1.1 only, which properly supports WebSocket upgrades.
	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"},
		},
	}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	log.Printf("Connected to live.predictatrade.com")

	// W3: send a signed handshake so the server can reject impersonated agents.
	a.sendWSHandshake()

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

		// Delegate role-specific handling to the configured handler. The
		// Master Node ignores execution messages; the Client processes them.
		a.handler.handleServerMessage(a, event)
	}
}

// sendToServer sends data (ticks, acks) to the Go RT server via WebSocket
func (a *Agent) sendToServer(data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn == nil {
		return fmt.Errorf("not connected")
	}
	// Count live data delivery to the engine — but ONLY real market data.
	// Counting every send here inflated the telemetry "candles" field with
	// heartbeats/telemetry (uptime/60 ≈ candles), masking a dead Master feed
	// as healthy during the 2026-08-29 co-located-roles incident.
	if a.role == "data" {
		var t struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &t) == nil &&
			(t.Type == "MASTER_TICK" || t.Type == "MARKET_SNAPSHOT") {
			a.deliveryMu.Lock()
			a.candlesDelivered++
			a.lastCandleAt = time.Now()
			a.deliveryMu.Unlock()
		}
	}
	return a.conn.WriteMessage(websocket.TextMessage, data)
}

// sendTelemetry pushes a periodic health/usage snapshot to the backend so the
// server has visibility into each client agent (uptime, connectivity, license,
// data delivery). It rides the same engine WebSocket as market data.
func (a *Agent) sendTelemetry() {
	if a.conn == nil {
		return
	}
	st := a.getStatus()
	payload := map[string]interface{}{
		"type":              "AGENT_TELEMETRY",
		"agent_id":          a.deviceID,
		"version":           AgentVersion,
		"role":              a.role,
		"goos":              runtime.GOOS,
		"goarch":            runtime.GOARCH,
		"mt4_connected":     st.MT4Connected,
		"mt5_connected":     st.MT5Connected,
		"backend_connected": st.BackendConnected,
		"uptime_seconds":    st.UptimeSeconds,
		"candles_delivered": st.CandlesDelivered,
		"license_status":    st.LicenseStatus,
		"license_plan":      st.LicensePlan,
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if err := a.sendToServer(data); err != nil {
		logf("[telemetry] send failed: %v", err)
	}
}

// telemetryLoop emits a telemetry snapshot every minute for server-side observability.
func (a *Agent) telemetryLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	a.sendTelemetry() // emit immediately on connect
	for {
		select {
		case <-ticker.C:
			a.sendTelemetry()
		case <-a.stopChan:
			return
		}
	}
}

// onTickFromEA is called when the EA sends real MT5 tick data via named pipe
func (a *Agent) onTickFromEA(tick MT5Tick) {
	// Capture latest XAUUSD tick for the heartbeat market-status payload (do not
	// fabricate; only store ticks the EA actually sends).
	if strings.EqualFold(tick.Symbol, "XAUUSD") {
		a.tickMu.Lock()
		t := tick
		a.lastXAUUSDTick = &t
		a.tickMu.Unlock()
	}
	// Forward tick to Go RT server via WebSocket
	data, err := MarshalTick(tick)
	if err != nil {
		return
	}
	a.sendToServer(data)
}

// onLicenseCheck is called when the EA requests license validation.
// It calls the NestJS control plane API to validate the license key.
func (a *Agent) onLicenseCheck(msg LicenseCheckMsg) {
	log.Printf("License check requested: account=%s broker=%s key=%s", msg.Account, msg.Broker, maskSecret(msg.LicenseKey))

	// Propagate the EA-provided key into the agent's own license config so that
	// device activation (registerTerminalWithBackend -> /devices/activate) uses
	// it. This is what makes typing the license key ONCE in the MT4/MT5 EA the
	// only manual step — the agent then activates the device automatically,
	// instead of requiring a separate PAT_LICENSE_KEY env var on the agent.
	if msg.LicenseKey != "" {
		a.config.LicenseKey = msg.LicenseKey
		// Terminals detected before the key arrived were skipped by
		// registerTerminalWithBackend (it bails when no license key is set).
		// Re-register them now so the control plane binds the license to the
		// device and subsequent validation verdicts become ACTIVE.
		for _, t := range a.pipeManager.GetTerminals() {
			go a.registerTerminalWithBackend(*t)
		}
	}

	// Build the API URL for license validation
	validateURL := strings.Replace(a.config.APIURL, "/api/v1", "/api/v1/licensing/validate", 1)
	if !strings.Contains(validateURL, "licensing/validate") {
		validateURL = a.config.APIURL + "/licensing/validate"
	}

	// POST license_key + MT account info to the control plane for validation
	// The server uses mt_account to enforce max_mt_accounts and prevent
	// the same license key from being used on unlimited terminals.
	reqBody, _ := json.Marshal(map[string]string{
		"license_key":    msg.LicenseKey,
		"mt_account":     msg.Account,
		"broker_name":    msg.Broker,
		"terminal_build": "",
		"ea_version":     "",
	})
	req, err := http.NewRequest("POST", validateURL, bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("License validation request error: %v", err)
		a.pipeManager.SetLicenseResult("UNKNOWN", "", nil)
		a.sendLicenseResponse(msg.LicenseKey, "ERROR", "UNKNOWN", nil)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("License validation HTTP error: %v", err)
		// Fallback: send UNKNOWN status so EA knows validation failed
		a.pipeManager.SetLicenseResult("UNKNOWN", "", nil)
		a.sendLicenseResponse(msg.LicenseKey, "ERROR", "UNKNOWN", nil)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("License validation response: HTTP %d — %s", resp.StatusCode, string(respBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("License validation failed: HTTP %d", resp.StatusCode)
		a.pipeManager.SetLicenseResult("UNKNOWN", "", nil)
		a.sendLicenseResponse(msg.LicenseKey, "ERROR", "UNKNOWN", nil)
		return
	}

	// Parse the API response
	var apiResp struct {
		Valid             bool     `json:"valid"`
		Status            string   `json:"status"`
		Plan              string   `json:"plan"`
		MaxDevices        int      `json:"max_devices"`
		MaxMTAccounts     int      `json:"max_mt_accounts"`
		AllowedStrategies []string `json:"allowed_strategies"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		log.Printf("License response parse error: %v", err)
		a.pipeManager.SetLicenseResult("UNKNOWN", "", nil)
		a.sendLicenseResponse(msg.LicenseKey, "ERROR", "UNKNOWN", nil)
		return
	}

	status := apiResp.Status
	if !apiResp.Valid {
		status = "INVALID"
		log.Printf("License INVALID: status=%s", apiResp.Status)
	} else {
		log.Printf("License VALID: status=%s plan=%s devices=%d", apiResp.Status, apiResp.Plan, apiResp.MaxDevices)
	}

	// Always surface a license type to the EA. The DB-backed plan is authoritative,
	// but if it ever comes back empty (e.g. a license row with no plan assigned,
	// or a transient validation hiccup) fall back to a sensible default so the
	// EA's "License Type" field is never blank.
	plan := apiResp.Plan
	if plan == "" {
		plan = "ELITE"
		log.Printf("License plan empty from server — defaulting display plan to ELITE")
	}

	// Record the authoritative verdict so licenseLoop() writes it to PAT_license.txt.
	a.pipeManager.SetLicenseResult(status, plan, apiResp.AllowedStrategies)
	a.sendLicenseResponse(msg.LicenseKey, status, plan, apiResp.AllowedStrategies)
}

// sendLicenseResponse writes the license validation result back to the EA via the pipe.
func (a *Agent) sendLicenseResponse(key, status, plan string, strategies []string) {
	response := LicenseResponse{
		Type:              "LICENSE_RESPONSE",
		Status:            status,
		Plan:              plan,
		Key:               key,
		Valid:             status == "ACTIVE",
		AllowedStrategies: strategies,
	}
	respData, _ := json.Marshal(response)
	if a.pipeManager != nil {
		a.pipeManager.WriteToPipe("LICENSE_RESPONSE", string(respData))
		// W3: write detached HMAC signature alongside the license file so the
		// EA can reject spoofed license status (operator-compiled follow-up).
		a.pipeManager.writeLicenseSig(status, plan)
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

	// Apply the update (Windows: helper batch script stops service, swaps, restarts).
	// Determine the exact path of the running binary so the swap targets the real
	// installed exe (pat-agent.exe OR pat-master.exe, in its actual install dir).
	exePath, err := os.Executable()
	if err != nil {
		logf("[updater] cannot determine own executable path: %v", err)
		return
	}
	currentPath := exePath
	if err := a.updater.ApplyUpdateOnWindows(stagedPath, currentPath, manifest); err != nil {
		logf("[updater] Apply failed: %v", err)
		return
	}

	logf("[updater] Update helper launched — service will restart shortly with v%s", manifest.Version)
	// The helper script will stop this process; no need to exit manually
}

// RequestUpdate triggers an immediate background update check. It is invoked by the
// dashboard "Check for Update" button so operators don't have to wait for the
// next hourly cycle. Runs in a goroutine; if an update is found it downloads,
// verifies, and swaps the binary automatically (the service restarts itself).
func (a *Agent) RequestUpdate() string {
	if a.updater == nil {
		return "Auto-update is not configured on this agent."
	}
	go a.safe(a.checkAndUpdate)
	return "Update check started in the background — watch the agent log; the service restarts itself if a new version is found."
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
				DeviceIDCP:      a.config.DeviceID,
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
				AuthMAC:         a.wsHMAC(a.deviceID + "|" + localNow.Format(time.RFC3339)),
			}
			data, _ := json.Marshal(hb)
			sendStart := time.Now()
			conn.WriteMessage(websocket.TextMessage, data)
			a.lastHeartbeat = time.Now()

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

		case <-a.stopChan:
			return
		}
	}
}

// hostname returns the machine hostname for heartbeat.
// serviceState maps the running flag to a heartbeat service status string.
func serviceState(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
