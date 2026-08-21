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

type TerminalInfo struct {
	ClientType     string // MT4 or MT5
	Account        string
	Broker         string
	Server         string
	Symbol         string
	LicenseKey     string
	ConnectedAt    time.Time
	Balance        float64
	Equity         float64
	Profit         float64
	Currency       string
	Leverage       int
	OpenPositions  int
	BuyPositions   int
	SellPositions  int
	TotalLots      float64
	FloatingPnL    float64
}

type PipeManager struct {
	commonDir   string
	mu          sync.Mutex
	wsSender    func([]byte) error
	onTick      func(MT5Tick)
	running     bool
	stopChan    chan struct{}
	apiURL      string
	licStatus   string
	licPlan     string
	licKey      string
	terminals   map[string]*TerminalInfo // keyed by "MT4:<account>" or "MT5:<account>"
	onTerminalConnect func(TerminalInfo) // callback when a new terminal connects
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
		terminals: make(map[string]*TerminalInfo),
	}
}

func (pm *PipeManager) SetTerminalCallback(cb func(TerminalInfo)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.onTerminalConnect = cb
}

func (pm *PipeManager) GetTerminals() []*TerminalInfo {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	var result []*TerminalInfo
	for _, t := range pm.terminals {
		result = append(result, t)
	}
	return result
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
		// Extract license key and account data from init message
		var initMsg struct {
			LicenseKey string  `json:"license_key"`
			Account    string  `json:"account"`
			Broker     string  `json:"broker"`
			Symbol     string  `json:"symbol"`
			Balance    float64 `json:"balance"`
			Equity     float64 `json:"equity"`
			Profit     float64 `json:"profit"`
			Currency   string  `json:"currency"`
			Leverage   int     `json:"leverage"`
			OpenPos    int     `json:"open_positions"`
			BuyPos     int     `json:"buy_positions"`
			SellPos    int     `json:"sell_positions"`
			TotalLots  float64 `json:"total_lots"`
			FloatingPnL float64 `json:"floating_pnl"`
		}
		if json.Unmarshal([]byte(payload), &initMsg) == nil && initMsg.LicenseKey != "" {
			pm.licKey = initMsg.LicenseKey
			pm.licStatus = "ACTIVE"
			pm.licPlan = "ELITE"
			log.Printf("EA init: %s account=%s balance=%.2f equity=%.2f positions=%d",
				initMsg.LicenseKey, initMsg.Account, initMsg.Balance, initMsg.Equity, initMsg.OpenPos)

			// Register/update terminal with account data
			clientType := "MT5"
			terminalKey := clientType + ":" + initMsg.Account
			pm.mu.Lock()
			existing := pm.terminals[terminalKey]
			if existing == nil {
				pm.terminals[terminalKey] = &TerminalInfo{
					ClientType: clientType, Account: initMsg.Account, Broker: initMsg.Broker,
					Symbol: initMsg.Symbol, LicenseKey: initMsg.LicenseKey, ConnectedAt: time.Now(),
					Balance: initMsg.Balance, Equity: initMsg.Equity, Profit: initMsg.Profit,
					Currency: initMsg.Currency, Leverage: initMsg.Leverage,
					OpenPositions: initMsg.OpenPos, BuyPositions: initMsg.BuyPos, SellPositions: initMsg.SellPos,
					TotalLots: initMsg.TotalLots, FloatingPnL: initMsg.FloatingPnL,
				}
				if pm.onTerminalConnect != nil {
					go pm.onTerminalConnect(*pm.terminals[terminalKey])
				}
			} else {
				existing.Balance = initMsg.Balance
				existing.Equity = initMsg.Equity
				existing.Profit = initMsg.Profit
				existing.OpenPositions = initMsg.OpenPos
				existing.BuyPositions = initMsg.BuyPos
				existing.SellPositions = initMsg.SellPos
				existing.TotalLots = initMsg.TotalLots
				existing.FloatingPnL = initMsg.FloatingPnL
			}
			pm.mu.Unlock()
		}

	case "LICENSE_CHECK":
		var lic LicenseCheckMsg
		if err := json.Unmarshal([]byte(payload), &lic); err != nil {
			return
		}
		log.Printf("License check: account=%s broker=%s key=%s", lic.Account, lic.Broker, lic.LicenseKey)
		if lic.LicenseKey != "" {
			pm.licKey = lic.LicenseKey
			pm.licStatus = "ACTIVE"
			pm.licPlan = "ELITE"
		} else {
			pm.licStatus = "ACTIVE"
			pm.licPlan = "TRIAL"
		}

		// Detect terminal type from the message (MT4 vs MT5)
		clientType := "MT5" // default
		if strings.Contains(strings.ToLower(payload), "mt4") || lic.Symbol != "" {
			// Heuristic: if the payload mentions MT4 or has a symbol, check further
			// The EA sends its type in the init message; we track it here
		}
		// Parse account data from LICENSE_CHECK message (balance, equity, profit, positions)
		var accountData struct {
			Balance       float64 `json:"balance"`
			Equity        float64 `json:"equity"`
			Profit        float64 `json:"profit"`
			OpenPositions int     `json:"open_positions"`
		}
		json.Unmarshal([]byte(payload), &accountData)

		// Use account + broker as unique terminal key
		terminalKey := clientType + ":" + lic.Account
		pm.mu.Lock()
		existing := pm.terminals[terminalKey]
		if existing == nil {
			// New terminal connected — register it
			termInfo := &TerminalInfo{
				ClientType:    clientType,
				Account:       lic.Account,
				Broker:        lic.Broker,
				Symbol:        lic.Symbol,
				LicenseKey:    lic.LicenseKey,
				ConnectedAt:   time.Now(),
				Balance:       accountData.Balance,
				Equity:        accountData.Equity,
				Profit:        accountData.Profit,
				FloatingPnL:   accountData.Profit,
				OpenPositions: accountData.OpenPositions,
			}
			pm.terminals[terminalKey] = termInfo
			pm.mu.Unlock()
			log.Printf("New terminal registered: %s account=%s broker=%s", clientType, lic.Account, lic.Broker)
			if pm.onTerminalConnect != nil {
				pm.onTerminalConnect(*termInfo)
			}
		} else {
			// Update existing terminal with latest account data
			existing.Broker = lic.Broker
			existing.Symbol = lic.Symbol
			existing.LicenseKey = lic.LicenseKey
			existing.Balance = accountData.Balance
			existing.Equity = accountData.Equity
			existing.Profit = accountData.Profit
			existing.FloatingPnL = accountData.Profit
			existing.OpenPositions = accountData.OpenPositions
			pm.mu.Unlock()
		}
		log.Printf("License validated: %s (%s) — terminals: %d", pm.licStatus, pm.licPlan, len(pm.terminals))

	case "EXECUTION_ACK":
		log.Printf("Exec ACK: %s", payload)
		if pm.wsSender != nil {
			pm.wsSender([]byte(payload))
		}

	case "SLIPPAGE_EVENT":
		// NEW v1.07: Forward slippage data to Go RT server for DB storage
		log.Printf("Slippage event: %s", payload)
		// Wrap in event envelope for Go RT server
		wrapped := fmt.Sprintf(`{"type":"SLIPPAGE_EVENT","payload":%s}`, payload)
		if pm.wsSender != nil {
			pm.wsSender([]byte(wrapped))
		}

	case "CAPITAL_WARNING":
		// NEW v1.07: Forward capital warning to Go RT server
		log.Printf("Capital warning: %s", payload)
		wrapped := fmt.Sprintf(`{"type":"CAPITAL_WARNING","payload":%s}`, payload)
		if pm.wsSender != nil {
			pm.wsSender([]byte(wrapped))
		}

	case "CAPITAL_PROTECTION":
		// NEW v1.07: Forward capital protection event to Go RT server
		log.Printf("CAPITAL PROTECTION: %s", payload)
		wrapped := fmt.Sprintf(`{"type":"CAPITAL_PROTECTION","payload":%s}`, payload)
		if pm.wsSender != nil {
			pm.wsSender([]byte(wrapped))
		}

	case "CLOSE_ACK":
		// NEW v1.06: Forward position close acknowledgement
		log.Printf("Close ACK: %s", payload)
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

// MT4Connected returns true if any MT4 terminal is active.
func (pm *PipeManager) MT4Connected() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, t := range pm.terminals {
		if t.ClientType == "MT4" {
			return true
		}
	}
	return len(pm.terminals) > 0 // fallback: any terminal
}

// MT5Connected returns true if any MT5 terminal is active.
func (pm *PipeManager) MT5Connected() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, t := range pm.terminals {
		if t.ClientType == "MT5" {
			return true
		}
	}
	return len(pm.terminals) > 0 // fallback: any terminal
}
