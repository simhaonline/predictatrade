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
	commonDirs  []string
	mu          sync.Mutex
	wsSender    func([]byte) error
	onTick      func(MT5Tick)
	running     bool
	stopChan    chan struct{}
	apiURL      string
	licStatus     string
	licPlan       string
	licKey        string
	licStrategies []string
	terminals   map[string]*TerminalInfo // keyed by "MT4:<account>" or "MT5:<account>"
	onTerminalConnect func(TerminalInfo) // callback when a new terminal connects
	onLicense   func(LicenseCheckMsg)    // callback to validate a license against the server
}

type LicenseCheckMsg struct {
	Type       string `json:"type"`
	Account    string `json:"account"`
	Broker     string `json:"broker"`
	Symbol     string `json:"symbol"`
	LicenseKey string `json:"license_key"`
}

type LicenseResponse struct {
	Type              string   `json:"type"`
	Status            string   `json:"status"`
	Plan              string   `json:"plan"`
	Key               string   `json:"key"`
	Valid             bool     `json:"valid"`
	MaxDevices        int      `json:"max_devices"`
	MaxMTAccounts     int      `json:"max_mt_accounts"`
	AllowedStrategies []string `json:"allowed_strategies"`
}

func NewPipeManager(commonDirs []string, wsSender func([]byte) error, apiURL string) *PipeManager {
	return &PipeManager{
		commonDirs: commonDirs,
		wsSender:   wsSender,
		apiURL:     apiURL,
		stopChan:   make(chan struct{}),
		licStatus:  "PENDING",
		licPlan:    "",
		terminals:  make(map[string]*TerminalInfo),
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
	pm.onLicense = onLicense
}

// SetLicenseResult records the authoritative license status returned by the
// server so licenseLoop() writes the real verdict (not a local self-approval)
// to PAT_license.txt for the EA to read.
func (pm *PipeManager) SetLicenseResult(status, plan string, strategies []string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if status != "" {
		pm.licStatus = status
	}
	if plan != "" {
		pm.licPlan = plan
	}
	if strategies != nil {
		pm.licStrategies = strategies
	}
}

// GetLicense returns the last authoritative license status and plan recorded
// by SetLicenseResult (or "" if none yet).
func (pm *PipeManager) GetLicense() (string, string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.licStatus, pm.licPlan
}

func (pm *PipeManager) Start() {
	pm.running = true
	// Only use folders that already exist. MetaTrader creates its
	// Common\Files directory (with the correct per-user ACL) on first run; if we
	// (running as LocalSystem) created it ourselves, the interactive user's EA
	// would be denied write access. Folders created by MT after agent start are
	// still picked up automatically by the write loops (WriteFile succeeds once
	// the parent exists).
	for _, d := range pm.commonDirs {
		if _, err := os.Stat(d); err == nil {
			_ = os.MkdirAll(d, 0755) // exists already -> no-op
		}
	}

	// Remove orphaned IPC files from previous agent versions so subscribers
	// upgrading the agent never have to clean up manually. This only touches
	// dead locations (never the live per-user Common\Files folders the EA uses).
	cleanupLegacyIpc()

	go pm.heartbeatLoop()
	go pm.readLoop()
	go pm.licenseLoop()  // Continuously write license response
	go pm.masterReadLoop() // Read master node data (indicators, bars, snapshots)

	log.Printf("File IPC started at %d folder(s): %v", len(pm.commonDirs), pm.commonDirs)
}

// cleanupLegacyIpc removes orphaned IPC files left behind by earlier agent
// versions. Upgrading the agent must be the ONLY step a subscriber performs —
// the EA (which uses MQL FILE_COMMON and never needed to change) keeps working.
//
// Dead locations removed:
//   - C:\ProgramData\PredictATrade\ipc  (v1.2.6 attempt; MQL cannot read
//     arbitrary absolute paths, so this folder was never usable by the EA)
//   - PAT_*.txt files in the LocalSystem %APPDATA% MetaQuotes Common\Files
//     folder (where the pre-fix agent wrote, but the user's EA never read them)
//
// The live per-user Common\Files folders are intentionally left untouched.
func cleanupLegacyIpc() {
	// 1. Remove the obsolete shared ProgramData\ipc directory.
	if base := os.Getenv("ProgramData"); base != "" {
		_ = os.RemoveAll(filepath.Join(base, "PredictATrade", "ipc"))
	}
	_ = os.RemoveAll(`C:\ProgramData\PredictATrade\ipc`)

	// 2. Remove stale PAT_*.txt orphaned in the LocalSystem MetaQuotes folder.
	if appData := os.Getenv("APPDATA"); appData != "" {
		legacy := filepath.Join(appData, "MetaQuotes", "Terminal", "Common", "Files")
		if entries, err := os.ReadDir(legacy); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := strings.ToUpper(e.Name())
				if strings.HasPrefix(name, "PAT_") && strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
					_ = os.Remove(filepath.Join(legacy, e.Name()))
				}
			}
		}
	}
}

func (pm *PipeManager) Stop() {
	pm.running = false
	close(pm.stopChan)
}

// findCommonFolders returns the MetaQuotes "Common\Files" directory used for
// file-based IPC between the Windows Agent and the MetaTrader EAs.
//
// The MetaTrader EA writes/reads these files via the MQL FILE_COMMON flag,
// which always resolves to:
//
//	C:\Users\<user>\AppData\Roaming\MetaQuotes\Terminal\Common\Files
//
// The agent normally runs as a Windows service under LocalSystem, whose
// %APPDATA% resolves to C:\Windows\System32\config\systemprofile\... — a DIFFERENT
// folder than the one the user's terminal uses. So the agent must explicitly
// target the real user-profile Common\Files folder(s) rather than rely on its
// own %APPDATA%.
//
// We scan every local user profile for the MetaQuotes Common\Files directory and
// use all of them (so the agent works regardless of which user runs MT, and even
// before the EA has created the folder). The EA (running as the user) reads and
// writes in the same physical directory, so both sides finally meet.
//
// Override entirely with PAT_IPC_DIR (single dir) if needed.
func findCommonFolders() []string {
	if env := os.Getenv("PAT_IPC_DIR"); env != "" {
		return []string{env}
	}

	var dirs []string
	seen := map[string]bool{}

	add := func(d string) {
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		dirs = append(dirs, d)
	}

	// Candidate users directory (Windows). On non-Windows dev builds this is a
	// no-op and the caller simply gets an empty list (harmless).
	usersRoot := `C:\Users`
	if env := os.Getenv("PAT_USERS_ROOT"); env != "" {
		usersRoot = env
	}
	if entries, err := os.ReadDir(usersRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			// Skip well-known non-real profiles.
			switch strings.ToLower(name) {
			case "public", "default", "default user", "all users", "administrator":
				// "administrator" can be a real trading account, but is also a
				// default profile; include it anyway (harmless, just an extra dir).
			}
			cand := filepath.Join(usersRoot, name, "AppData", "Roaming",
				"MetaQuotes", "Terminal", "Common", "Files")
			add(cand)
		}
	}

	// Fallback: the agent's own profile (original behavior) — included so a
	// LocalSystem-only scenario still has a folder to write into.
	if appData := os.Getenv("APPDATA"); appData != "" {
		add(filepath.Join(appData, "MetaQuotes", "Terminal", "Common", "Files"))
	}

	return dirs
}

// heartbeatLoop writes heartbeat every 2s so EA knows Agent is alive.
func (pm *PipeManager) heartbeatLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			content := fmt.Sprintf(`{"type":"HEARTBEAT","timestamp":"%s","agent":"1.02"}`,
				time.Now().UTC().Format(time.RFC3339))
			for _, d := range pm.commonDirs {
				os.WriteFile(filepath.Join(d, "PAT_heartbeat.txt"), []byte(content), 0644)
			}
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
	for {
		select {
		case <-ticker.C:
			plan := pm.licPlan
			if plan == "" {
				plan = "ELITE" // never write a blank plan; EA always shows a type
			}
			response := LicenseResponse{
				Type:              "LICENSE_RESPONSE",
				Status:            pm.licStatus,
				Plan:              plan,
				Key:               pm.licKey,
				AllowedStrategies: pm.licStrategies,
			}
			respData, _ := json.Marshal(response)
			for _, d := range pm.commonDirs {
				os.WriteFile(filepath.Join(d, "PAT_license.txt"), respData, 0644)
			}
		case <-pm.stopChan:
			return
		}
	}
}

// readLoop continuously reads PAT_ticks.txt for tick data from the EA. It
// scans every candidate Common\Files folder so the agent picks up whichever
// user profile the MetaTrader terminal is running under.
func (pm *PipeManager) readLoop() {
	for {
		select {
		case <-pm.stopChan:
			return
		default:
		}
		for _, d := range pm.commonDirs {
			ticksPath := filepath.Join(d, "PAT_ticks.txt")
			data, err := os.ReadFile(ticksPath)
			if err != nil || len(data) == 0 {
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
		}
		time.Sleep(5 * time.Millisecond)
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
		// The INIT message does not carry a platform tag, but every TICK does
		// ("source":"MT4"|"MT5"). Reclassify the terminal so MT4/MT5 detection
		// is correct without requiring an EA recompile.
		clientType := "MT5"
		if src := strings.ToUpper(tick.Source); src == "MT4" || src == "MT5" {
			clientType = src
			pm.setTerminalClientType(tick.Account, src)
		}
		// Auto-register terminal from TICK data so the agent recovers terminal
		// state after an agent restart without requiring the EA to re-send INIT.
		// The EA only sends INIT in OnInit(); if the agent restarts while the EA
		// is already running, pm.terminals would stay empty and the heartbeat
		// would report mt4_connected: false / mt5_connected: false even though
		// the terminal IS connected and sending ticks.
		if tick.Account != "" {
			terminalKey := clientType + ":" + tick.Account
			pm.mu.Lock()
			if existing := pm.terminals[terminalKey]; existing == nil {
				pm.terminals[terminalKey] = &TerminalInfo{
					ClientType: clientType, Account: tick.Account,
					Broker: tick.Broker, Symbol: tick.Symbol,
					ConnectedAt: time.Now(),
				}
				pm.mu.Unlock()
				log.Printf("Terminal auto-registered from tick: %s account=%s broker=%s", clientType, tick.Account, tick.Broker)
				if pm.onTerminalConnect != nil {
					go pm.onTerminalConnect(*pm.terminals[terminalKey])
				}
			} else {
				existing.Broker = tick.Broker
				existing.Symbol = tick.Symbol
				pm.mu.Unlock()
			}
		}
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
			// Do NOT self-approve. Mark pending and validate against the server.
			pm.licStatus = "PENDING"
			log.Printf("EA init: validating license %s account=%s balance=%.2f equity=%.2f positions=%d",
				initMsg.LicenseKey, initMsg.Account, initMsg.Balance, initMsg.Equity, initMsg.OpenPos)
			if pm.onLicense != nil {
				go pm.onLicense(LicenseCheckMsg{
					Type:       "LICENSE_CHECK",
					Account:    initMsg.Account,
					Broker:     initMsg.Broker,
					Symbol:     initMsg.Symbol,
					LicenseKey: initMsg.LicenseKey,
				})
			}

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
		// Do NOT self-approve. Mark pending and validate against the server.
		pm.licKey = lic.LicenseKey
		pm.licStatus = "PENDING"
		if pm.onLicense != nil {
			go pm.onLicense(lic)
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

	case "TRADE_RESULT":
		// EA v1.08 exit reconciliation (mql-fix.md Bug 5): forward the full
		// outcome record — signal_id, strategy_id, magic, exit_reason,
		// realized_pnl, sl_correct — to the Go RT server for the
		// expected-vs-actual reconciliation table.
		log.Printf("Trade result: %s", payload)
		wrapped := fmt.Sprintf(`{"type":"TRADE_RESULT","payload":%s}`, payload)
		if pm.wsSender != nil {
			pm.wsSender([]byte(wrapped))
		}

	case "DEINIT":
		log.Printf("EA deinit: %s", payload)
	}
}

func (pm *PipeManager) WriteToPipe(msgType, payload string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	fullMsg := fmt.Sprintf("%s|%s\n", msgType, payload)
	for _, d := range pm.commonDirs {
		signalsPath := filepath.Join(d, "PAT_signals.txt")
		existing, _ := os.ReadFile(signalsPath)
		os.WriteFile(signalsPath, []byte(string(existing)+fullMsg), 0644)
	}
}

func (pm *PipeManager) SendSignalToEA(signalJSON string) {
	pm.WriteToPipe("SIGNAL", signalJSON)
	log.Printf("Signal written for EA")
}

func (pm *PipeManager) IsConnected() bool {
	for _, d := range pm.commonDirs {
		info, err := os.Stat(filepath.Join(d, "PAT_heartbeat.txt"))
		if err == nil && time.Since(info.ModTime()) < 10*time.Second {
			return true
		}
	}
	return false
}

// masterReadLoop reads PAT_master_data.txt for comprehensive market data from the Master Node EA.
// The Master Node sends: MASTER_TICK, MARKET_SNAPSHOT, MASTER_INIT, MASTER_DEINIT
// This data is forwarded to the Go RT server for the dashboard/Command Center.
//
// Diagnostic note: a throttled (every 30s) log reports whether PAT_master_data.txt is
// present in each scanned Common\Files folder. If it is NEVER present, the Master Node EA
// is not writing it (attach/enable the PredictATrade_MasterNode_MT5 EA). If it IS present
// but "Forwarding MARKET_SNAPSHOT to engine" never logs, the file content is malformed.
var lastMasterDiag time.Time

func (pm *PipeManager) masterReadLoop() {
	for {
		select {
		case <-pm.stopChan:
			return
		default:
		}
		now := time.Now()
		doDiag := now.Sub(lastMasterDiag) > 30*time.Second
		if doDiag {
			lastMasterDiag = now
		}
		for _, d := range pm.commonDirs {
			masterPath := filepath.Join(d, "PAT_master_data.txt")
			data, err := os.ReadFile(masterPath)
			if err != nil || len(data) == 0 {
				if doDiag {
					log.Printf("[IPC] PAT_master_data.txt not present in %s", d)
				}
				continue
			}
			if doDiag {
				log.Printf("[IPC] PAT_master_data.txt present in %s (%d bytes) — forwarding to engine", d, len(data))
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
		}
		time.Sleep(5 * time.Millisecond)
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
// setTerminalClientType reclassifies a terminal by account once the EA's
// platform ("source") is learned from a TICK message, keeping the lookup key
// consistent with the real ClientType.
func (pm *PipeManager) setTerminalClientType(account, clientType string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for k, t := range pm.terminals {
		if t.Account == account && t.ClientType != clientType {
			t.ClientType = clientType
			delete(pm.terminals, k)
			pm.terminals[clientType+":"+account] = t
			log.Printf("Terminal %s reclassified as %s", account, clientType)
		}
	}
}

func (pm *PipeManager) MT4Connected() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, t := range pm.terminals {
		if t.ClientType == "MT4" {
			return true
		}
	}
	return false
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
	return false
}
