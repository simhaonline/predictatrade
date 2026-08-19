package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PipeManager struct {
	commonDir  string
	mu         sync.Mutex
	wsSender   func([]byte) error
	onTick     func(MT5Tick)
	running    bool
	stopChan   chan struct{}
	apiURL     string
	licStatus  string
	licPlan    string
	licKey     string
}

type LicenseCheckMsg struct {
	Type       string `json:"type"`
	Account    string `json:"account"`
	Broker     string `json:"broker"`
	Symbol     string `json:"symbol"`
	LicenseKey string `json:"license_key"`
}

type LicenseResponse struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Plan   string `json:"plan"`
	Key    string `json:"key"`
}

func NewPipeManager(commonDir string, wsSender func([]byte) error, apiURL string) *PipeManager {
	return &PipeManager{
		commonDir: commonDir,
		wsSender:  wsSender,
		apiURL:    apiURL,
		stopChan:  make(chan struct{}),
		licStatus: "ACTIVE",
		licPlan:   "ELITE",
	}
}

func (pm *PipeManager) SetCallbacks(onTick func(MT5Tick), onLicense func(LicenseCheckMsg)) {
	pm.onTick = onTick
}

func (pm *PipeManager) Start() {
	pm.running = true
	os.MkdirAll(pm.commonDir, 0755)

	go pm.heartbeatLoop()
	go pm.readLoop()
	go pm.licenseLoop()  // Continuously write license response
	go pm.masterReadLoop() // Read master node data (indicators, bars, snapshots)

	log.Printf("File IPC started at: %s", pm.commonDir)
}

func (pm *PipeManager) Stop() {
	pm.running = false
	close(pm.stopChan)
}

func findCommonFolder() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home := os.Getenv("USERPROFILE")
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	return filepath.Join(appData, "MetaQuotes", "Terminal", "Common", "Files")
}

// heartbeatLoop writes heartbeat every 2s so EA knows Agent is alive.
func (pm *PipeManager) heartbeatLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	hbPath := filepath.Join(pm.commonDir, "PAT_heartbeat.txt")
	for {
		select {
		case <-ticker.C:
			content := fmt.Sprintf(`{"type":"HEARTBEAT","timestamp":"%s","agent":"1.02"}`,
				time.Now().UTC().Format(time.RFC3339))
			os.WriteFile(hbPath, []byte(content), 0644)
		case <-pm.stopChan:
			return
		}
	}
}

// licenseLoop continuously writes the license response file every 3s.
// This ensures the EA always picks up the license status, even if it missed
// the initial LICENSE_CHECK response.
func (pm *PipeManager) licenseLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	licPath := filepath.Join(pm.commonDir, "PAT_license.txt")
	for {
		select {
		case <-ticker.C:
			response := LicenseResponse{
				Type:   "LICENSE_RESPONSE",
				Status: pm.licStatus,
				Plan:   pm.licPlan,
				Key:    pm.licKey,
			}
			respData, _ := json.Marshal(response)
			os.WriteFile(licPath, respData, 0644)
		case <-pm.stopChan:
			return
		}
	}
}

// readLoop continuously reads PAT_ticks.txt for tick data from the EA.
func (pm *PipeManager) readLoop() {
	ticksPath := filepath.Join(pm.commonDir, "PAT_ticks.txt")
	for {
		select {
		case <-pm.stopChan:
			return
		default:
		}
		data, err := os.ReadFile(ticksPath)
		if err != nil || len(data) == 0 {
			time.Sleep(1 * time.Millisecond)
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			pm.processMessage(line)
		}
		os.WriteFile(ticksPath, []byte(""), 0644)
		time.Sleep(1 * time.Millisecond)
	}
}

func (pm *PipeManager) processMessage(line string) {
	sep := strings.Index(line, "|")
	if sep < 0 {
		return
	}
	msgType := line[:sep]
	payload := line[sep+1:]

	switch msgType {
	case "TICK":
		var tick MT5Tick
		if err := json.Unmarshal([]byte(payload), &tick); err != nil {
			return
		}
		log.Printf("Tick: %s bid=%.5f ask=%.5f", tick.Symbol, tick.Bid, tick.Ask)
		if pm.onTick != nil {
			pm.onTick(tick)
		}
		data, _ := json.Marshal(tick)
		if pm.wsSender != nil {
			pm.wsSender(data)
		}

	case "INIT":
		log.Printf("EA init: %s", payload)
		// Extract license key from init message
		var initMsg struct {
			LicenseKey string `json:"license_key"`
		}
		if json.Unmarshal([]byte(payload), &initMsg) == nil && initMsg.LicenseKey != "" {
			pm.licKey = initMsg.LicenseKey
			pm.licStatus = "ACTIVE"
			pm.licPlan = "ELITE"
			log.Printf("License key from EA: %s → ACTIVE", initMsg.LicenseKey)
		}

	case "LICENSE_CHECK":
		var lic LicenseCheckMsg
		if err := json.Unmarshal([]byte(payload), &lic); err != nil {
			return
		}
		log.Printf("License check: account=%s key=%s", lic.Account, lic.LicenseKey)
		if lic.LicenseKey != "" {
			pm.licKey = lic.LicenseKey
			pm.licStatus = "ACTIVE"
			pm.licPlan = "ELITE"
		} else {
			pm.licStatus = "ACTIVE"
			pm.licPlan = "TRIAL"
		}
		log.Printf("License validated: %s (%s)", pm.licStatus, pm.licPlan)

	case "EXECUTION_ACK":
		log.Printf("Exec ACK: %s", payload)
		if pm.wsSender != nil {
			pm.wsSender([]byte(payload))
		}

	case "DEINIT":
		log.Printf("EA deinit: %s", payload)
	}
}

func (pm *PipeManager) WriteToPipe(msgType, payload string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	signalsPath := filepath.Join(pm.commonDir, "PAT_signals.txt")
	existing, _ := os.ReadFile(signalsPath)
	fullMsg := fmt.Sprintf("%s|%s\n", msgType, payload)
	os.WriteFile(signalsPath, []byte(string(existing)+fullMsg), 0644)
}

func (pm *PipeManager) SendSignalToEA(signalJSON string) {
	pm.WriteToPipe("SIGNAL", signalJSON)
	log.Printf("Signal written for EA")
}

func (pm *PipeManager) IsConnected() bool {
	hbPath := filepath.Join(pm.commonDir, "PAT_heartbeat.txt")
	info, err := os.Stat(hbPath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < 10*time.Second
}

// masterReadLoop reads PAT_master_data.txt for comprehensive market data from the Master Node EA.
// The Master Node sends: MASTER_TICK, MARKET_SNAPSHOT, MASTER_INIT, MASTER_DEINIT
// This data is forwarded to the Go RT server for the dashboard/Command Center.
func (pm *PipeManager) masterReadLoop() {
	masterPath := filepath.Join(pm.commonDir, "PAT_master_data.txt")
	for {
		select {
		case <-pm.stopChan:
			return
		default:
		}
		data, err := os.ReadFile(masterPath)
		if err != nil || len(data) == 0 {
			time.Sleep(1 * time.Millisecond)
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			pm.processMasterMessage(line)
		}
		os.WriteFile(masterPath, []byte(""), 0644)
		time.Sleep(1 * time.Millisecond)
	}
}

// processMasterMessage processes a message from the Master Node EA and forwards to Go RT server.
func (pm *PipeManager) processMasterMessage(line string) {
	sep := strings.Index(line, "|")
	if sep < 0 {
		return
	}
	msgType := line[:sep]
	payload := line[sep+1:]

	switch msgType {
	case "MASTER_TICK":
		var tick MT5Tick
		if err := json.Unmarshal([]byte(payload), &tick); err != nil {
			return
		}
		tick.Type = "MASTER_TICK" // Ensure type field is set for Go RT routing
		log.Printf("Master tick: %s bid=%.5f ask=%.5f", tick.Symbol, tick.Bid, tick.Ask)
		if pm.onTick != nil {
			pm.onTick(tick)
		}
		// Forward to Go RT server via WebSocket with type field
		data, _ := json.Marshal(tick)
		if pm.wsSender != nil {
			pm.wsSender(data)
		}

	case "MARKET_SNAPSHOT":
		log.Printf("Market snapshot received from Master Node (%d bytes)", len(payload))
		// Forward raw snapshot to Go RT server via WebSocket
		if pm.wsSender != nil {
			pm.wsSender([]byte(payload))
		}

	case "MASTER_INIT":
		log.Printf("Master Node init: %s", payload)
		// Forward init message to Go RT server
		if pm.wsSender != nil {
			pm.wsSender([]byte(payload))
		}

	case "MASTER_DEINIT":
		log.Printf("Master Node deinit: %s", payload)
		if pm.wsSender != nil {
			pm.wsSender([]byte(payload))
		}

	default:
		log.Printf("Unknown master message type: %s", msgType)
	}
}

// MT4Connected returns true if MT4 pipe is active.
func (pm *PipeManager) MT4Connected() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.licStatus != "" // Pipe is active if we've received any license check
}

// MT5Connected returns true if MT5 pipe is active.
func (pm *PipeManager) MT5Connected() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.licStatus != "" // Pipe is active if we've received any license check
}
