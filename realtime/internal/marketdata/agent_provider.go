// Package marketdata — AgentProvider receives real tick data from Windows MT5 Agent.
// Architecture: MT5 Terminal → Windows Agent → wss://live.predictatrade.com/ws/v1/agent → Go RT engine
// This provider does NOT generate fake data. It only processes real ticks from connected agents.
package marketdata

import (
	"log"
	"encoding/json"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// parseMQLTimestamp parses timestamps from MQL EAs.
// Handles both old format ("2026.08.21 19:25:11" with dots, broker time)
// and new ISO8601 format ("2026-08-21T16:25:11Z" UTC).
// Returns the parsed time in UTC, or time.Now().UTC() if parsing fails.
func parseMQLTimestamp(s string) time.Time {
	if s == "" {
		return time.Now().UTC()
	}
	// Try ISO8601 first (new format from updated EAs)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	// Try parsing with dots → convert to dashes
	dashFormat := strings.ReplaceAll(s, ".", "-")
	// Try "2026-08-21 19:25:11" (without timezone — assume broker time)
	// We can't know the broker offset, so use gateway time instead
	if t, err := time.Parse("2006-01-02 15:04:05", dashFormat); err == nil {
		// This is broker time without timezone info — we can't trust it as UTC
		// Use gateway time instead for accuracy
		_ = t // discard — we use gateway time
		return time.Now().UTC()
	}
	// Fallback to gateway time
	return time.Now().UTC()
}

// AgentTickMessage is the message format the Windows Agent sends with real MT5 tick data.
type AgentTickMessage struct {
	Type      string          `json:"type"`       // "TICK", "HEARTBEAT", "BAR"
	Symbol    string          `json:"symbol"`
	Bid       float64         `json:"bid"`
	Ask       float64         `json:"ask"`
	Volume    int64           `json:"volume"`
	Timestamp string `json:"timestamp"`
	Source    string          `json:"source"`     // "MT5", "MT4"
	Broker    string          `json:"broker"`
	Account   string          `json:"account"`
}

// MarketSnapshot is a comprehensive market data message from the Master Node EA.
// It includes ticks, multi-timeframe bars, indicators, account info, and symbol info.
// SOW Section 9: Normalized market state for dashboard/Command Center.
type MarketSnapshot struct {
	Type        string          `json:"type"`
	Symbol      string          `json:"symbol"`
	Timestamp   string          `json:"timestamp"`
	GMT         string          `json:"gmt"`
	Source      string          `json:"source"`
	Broker      string          `json:"broker"`
	Account     string          `json:"account"`
	Node        string          `json:"node"`
	Tick        SnapshotTick    `json:"tick"`
	Bars        map[string]SnapshotBar `json:"bars,omitempty"`
	Indicators  SnapshotIndicators `json:"indicators,omitempty"`
	VWAP        SnapshotVWAP    `json:"vwap"`
	AccountInfo SnapshotAccount `json:"account_info,omitempty"`
	SymbolInfo  SnapshotSymbol  `json:"symbol_info,omitempty"`
	Session     SnapshotSession `json:"session"`
	Positions   SnapshotPositions `json:"positions,omitempty"`
}

type SnapshotTick struct {
	Bid          float64 `json:"bid"`
	Ask          float64 `json:"ask"`
	Spread       float64 `json:"spread"`
	SpreadPoints int64   `json:"spread_points"`
	Volume       int64   `json:"volume"`
	Time         string  `json:"time"`
}

type SnapshotBar struct {
	Open       float64 `json:"open"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	Close      float64 `json:"close"`
	Volume     int64   `json:"volume"`
	Time       string  `json:"time"`
	PrevOpen   float64 `json:"prev_open,omitempty"`
	PrevHigh   float64 `json:"prev_high,omitempty"`
	PrevLow    float64 `json:"prev_low,omitempty"`
	PrevClose  float64 `json:"prev_close,omitempty"`
	PrevVolume int64   `json:"prev_volume,omitempty"`
}

type SnapshotIndicators struct {
	// Original MT5 EA fields
	ATR         float64 `json:"atr"`
	RSI         float64 `json:"rsi"`
	EMA9        float64 `json:"ema9"`
	EMA21       float64 `json:"ema21"`
	EMA50       float64 `json:"ema50"`
	SMA200      float64 `json:"sma200"`
	ADX         float64 `json:"adx"`
	ADXPlusDI   float64 `json:"adx_plus_di"`
	ADXMinusDI  float64 `json:"adx_minus_di"`
	BollUpper   float64 `json:"boll_upper"`
	BollLower   float64 `json:"boll_lower"`
	BollMiddle  float64 `json:"boll_middle"`
	MACDMain    float64 `json:"macd_main"`
	MACDSignal  float64 `json:"macd_signal"`
	StochMain   float64 `json:"stoch_main"`
	StochSignal float64 `json:"stoch_signal"`
	CCI         float64 `json:"cci"`
	Mom         float64 `json:"mom"`
	OsMA        float64 `json:"osma"`

	// Locally-computed indicators (enriched by Go engine, prompt.md Section 1)
	EMA100        float64 `json:"ema100,omitempty"`
	EMA200        float64 `json:"ema200,omitempty"`
	EMACross921   bool    `json:"ema_cross_9_21,omitempty"`
	SMA50         float64 `json:"sma50,omitempty"`
	SMA100        float64 `json:"sma100,omitempty"`
	MACDHistogram float64 `json:"macd_histogram,omitempty"`
	MACDBullCross bool    `json:"macd_bull_cross,omitempty"`
	MACDBearCross bool    `json:"macd_bear_cross,omitempty"`
	BollWidth     float64 `json:"boll_width,omitempty"`
	BollBullRev   bool    `json:"boll_bull_rev,omitempty"`
	BollBearRev   bool    `json:"boll_bear_rev,omitempty"`
	OBV           float64 `json:"obv,omitempty"`
	TickVolume    float64 `json:"tick_volume,omitempty"`
	VWAP          float64 `json:"vwap,omitempty"`
	ParabolicSAR  float64 `json:"psar,omitempty"`
	PSARLong      bool    `json:"psar_long,omitempty"`
	StochRSI      float64 `json:"stoch_rsi,omitempty"`
	StochRSIK     float64 `json:"stoch_rsi_k,omitempty"`
	StochRSID     float64 `json:"stoch_rsi_d,omitempty"`
	IchimokuTenkan  float64 `json:"ichimoku_tenkan,omitempty"`
	IchimokuKijun   float64 `json:"ichimoku_kijun,omitempty"`
	IchimokuSenkouA float64 `json:"ichimoku_senkou_a,omitempty"`
	IchimokuSenkouB float64 `json:"ichimoku_senkou_b,omitempty"`
}

type SnapshotVWAP struct {
	SessionVWAP float64 `json:"session_vwap"`
}

type SnapshotAccount struct {
	Balance    float64 `json:"balance"`
	Equity     float64 `json:"equity"`
	Margin     float64 `json:"margin"`
	FreeMargin float64 `json:"free_margin"`
	Profit     float64 `json:"profit"`
	Currency   string  `json:"currency"`
	Leverage   int64   `json:"leverage"`
	Server     string  `json:"server"`
}

type SnapshotSymbol struct {
	Digits       int64   `json:"digits"`
	Point        float64 `json:"point"`
	Spread       int64   `json:"spread"`
	StopsLevel   int64   `json:"stops_level"`
	FreezeLevel  int64   `json:"freeze_level"`
	ContractSize float64 `json:"contract_size"`
	MinLot       float64 `json:"min_lot"`
	MaxLot       float64 `json:"max_lot"`
	LotStep      float64 `json:"lot_step"`
	SwapLong     float64 `json:"swap_long"`
	SwapShort    float64 `json:"swap_short"`
	TickValue    float64 `json:"tick_value"`
	TickSize     float64 `json:"tick_size"`
	MarginInit   float64 `json:"margin_init"`
	MarginMaint  float64 `json:"margin_maint"`
}

type SnapshotSession struct {
	Name      string `json:"name"`
	IsOverlap bool   `json:"is_overlap"`
	IsWeekend bool   `json:"is_weekend"`
	GMTHour   int64  `json:"gmt_hour"`
	GMTDow    int64  `json:"gmt_dow"`
}

type SnapshotPositions struct {
	TotalPositions int64   `json:"total_positions"`
	BuyCount        int64   `json:"buy_count"`
	SellCount       int64   `json:"sell_count"`
	TotalLots       float64 `json:"total_lots"`
	FloatingProfit  float64 `json:"floating_profit"`
}

// AgentProvider receives real tick data from connected Windows MT5 Agents.
// It does NOT generate fake data. If no agent is connected, it produces NO ticks
// and the system degrades to NO-TRADE (SOW: data quality gate fails closed).
type AgentProvider struct {
	name      string
	mu        sync.Mutex
	agents    map[string]chan *AgentTickMessage // agentID → tick channel
	tickChan  chan *types.Tick
	stopChan  chan struct{}
	running   atomic.Bool
	validator *TickValidator

	// Market snapshot storage from Master Node
	snapshotMu     sync.RWMutex
	lastSnapshot    *MarketSnapshot
	snapshotCount   uint64

	// Valkey cache — write snapshots directly (not via delayed broadcast loop)
	valkeyCache interface{ SetSnapshot(interface{}) error; SetMarketState(interface{}) error; AddPricePoint(float64, time.Time) error }

	// State manager — merge snapshot indicators/bars into market state
	stateMgr StateUpdater

	// Merge function — set by main.go to avoid import cycle
	mergeSnapshotFn func(any, *MarketSnapshot)

	// BrokerAccountHydrateFn — called when a snapshot with account_info arrives.
	// This callback hydrates safety-critical gates (exposure, margin, execution)
	// from live broker account data. Set by main.go to avoid import cycle.
	brokerAccountHydrateFn func(account *SnapshotAccount, positions *SnapshotPositions)

	// AgentConnectFn — called when an agent connects or sends heartbeat.
	// This hydrates the execution permit gate (terminal connected = PASS).
	agentConnectFn func(agentID string, msgType string)

	// LicenseValidateFn — validates a license key against the DB and sends
	// a LICENSE_STATUS response back to the agent. Set by main.go.
	licenseValidateFn func(agentID, licenseKey string) LicenseValidationResult
}

// LicenseValidationResult holds the result of license key validation.
type LicenseValidationResult struct {
	Valid       bool   `json:"valid"`
	Status      string `json:"status"`       // ACTIVE, EXPIRED, REVOKED, NOT_FOUND
	Plan        string `json:"plan"`         // FREE, STANDARD, PRO, ELITE
	MaxDevices  int    `json:"max_devices"`
	MaxMTAccts  int    `json:"max_mt_accounts"`
	Strategies  []string `json:"allowed_strategies"`
	Error       string `json:"error,omitempty"`
}

func (p *AgentProvider) SetValkeyCache(v interface{ SetSnapshot(interface{}) error; SetMarketState(interface{}) error; AddPricePoint(float64, time.Time) error }) {
	p.valkeyCache = v
}

// SetStateManager gives the provider access to the state manager so it can
// merge authoritative MT5 snapshot indicators and bars into MarketState.
type StateUpdater interface {
	Update(symbol string, update func(any))
}

func (p *AgentProvider) SetStateManager(sm StateUpdater) {
	p.stateMgr = sm
}

// SetMergeFunction sets the callback that merges snapshot data into MarketState.
// This avoids import cycle between marketdata and features packages.
func (p *AgentProvider) SetMergeFunction(fn func(any, *MarketSnapshot)) {
	p.mergeSnapshotFn = fn
}

// SetBrokerAccountHydrateFn sets the callback that hydrates safety-critical gates
// from live broker account data when a MARKET_SNAPSHOT with account_info arrives.
func (p *AgentProvider) SetBrokerAccountHydrateFn(fn func(account *SnapshotAccount, positions *SnapshotPositions)) {
	p.brokerAccountHydrateFn = fn
}

// SetAgentConnectFn sets the callback that hydrates the execution permit gate
// when an agent connects or sends a heartbeat.
func (p *AgentProvider) SetAgentConnectFn(fn func(agentID string, msgType string)) {
	p.agentConnectFn = fn
}

// SetLicenseValidateFn sets the license validation callback.
func (p *AgentProvider) SetLicenseValidateFn(fn func(agentID, licenseKey string) LicenseValidationResult) {
	p.licenseValidateFn = fn
}

func NewAgentProvider() *AgentProvider {
	return &AgentProvider{
		name:      "MT5_AGENT",
		agents:    make(map[string]chan *AgentTickMessage),
		tickChan:  make(chan *types.Tick, 4096),
		stopChan:  make(chan struct{}),
		validator: NewTickValidator(),
	}
}

func (p *AgentProvider) Name() string { return p.name }
func (p *AgentProvider) Connect(ctx context.Context) error {
	p.running.Store(true)
	return nil
}
func (p *AgentProvider) Subscribe(symbol string) error { return nil }
func (p *AgentProvider) Stream() <-chan *types.Tick { return p.tickChan }
func (p *AgentProvider) Close() error {
	if p.running.CompareAndSwap(true, false) {
		close(p.stopChan)
	}
	return nil
}

// RegisterAgent creates a channel for a new Windows Agent connection.
func (p *AgentProvider) RegisterAgent(agentID string) chan *AgentTickMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	ch := make(chan *AgentTickMessage, 256)
	p.agents[agentID] = ch

	// Start processing goroutine for this agent
	go p.processAgentTicks(agentID, ch)
	return ch
}

// UnregisterAgent removes an agent when it disconnects.
func (p *AgentProvider) UnregisterAgent(agentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ch, ok := p.agents[agentID]; ok {
		close(ch)
		delete(p.agents, agentID)
	}
}

// HasConnectedAgents returns true if at least one Windows Agent is connected.
func (p *AgentProvider) HasConnectedAgents() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.agents) > 0
}

// processAgentTicks converts AgentTickMessages into validated types.Tick and sends to stream.
func (p *AgentProvider) processAgentTicks(agentID string, ch chan *AgentTickMessage) {
	for {
		select {
		case <-p.stopChan:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.Type != "TICK" && msg.Type != "MASTER_TICK" {
				continue // Skip heartbeats and other non-tick messages
			}

			// Parse the MT5 timestamp — new EAs send ISO8601 UTC, old EAs send
			// broker time with dots. parseMQLTimestamp handles both and falls
			// back to gateway time for old format (broker time without TZ info).
			sourceTime := parseMQLTimestamp(msg.Timestamp)
			tick := &types.Tick{
				Symbol:           normalizeSymbol(msg.Symbol),
				Bid:              decimal.NewFromFloat(msg.Bid),
				Ask:              decimal.NewFromFloat(msg.Ask),
				TickVolume:       msg.Volume,
				Source:           msg.Source,
				SourceTimestamp:  sourceTime,
				GatewayTimestamp: time.Now().UTC(),
				Quality:          types.QualityAuthoritative, // Real MT5 data is AUTHORITATIVE
			}
			NormalizeTick(tick)

			if valid, _ := p.validator.Validate(tick); !valid {
				continue // Reject bad ticks
			}

			select {
			case p.tickChan <- tick:
			default:
				// Backpressure — drop tick if buffer full
			}
		}
	}
}

// normalizeSymbol converts broker-specific XAUUSD variants to canonical "XAUUSD".
// Brokers use different suffixes: XAUUSD, XAUUSD.sd, XAUUSD.e, XAUUSD.m, etc.
// All should be treated as the same instrument for strategy evaluation.
func normalizeSymbol(s string) string {
	if len(s) >= 6 && s[:6] == "XAUUSD" {
		return "XAUUSD"
	}
	return s
}

// HandleAgentMessage processes a raw JSON message from a Windows Agent WebSocket connection.
// It detects the message type and routes accordingly:
//   - "TICK" / "MASTER_TICK" → tick processing pipeline
//   - "MARKET_SNAPSHOT" → market snapshot storage for dashboard/Command Center
//   - "MASTER_INIT" / "MASTER_DEINIT" / "HEARTBEAT" → logged and stored
func (p *AgentProvider) HandleAgentMessage(agentID string, data []byte) {
	// First, detect the message type
	var typeDetector struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeDetector); err != nil {
		return
	}

	msgType := typeDetector.Type
	if msgType == "" {
		// Fallback: old agent sends ticks without type field — detect by bid/ask presence
		var tickCheck struct {
			Bid float64 `json:"bid"`
			Ask float64 `json:"ask"`
		}
		if err := json.Unmarshal(data, &tickCheck); err == nil && tickCheck.Bid > 0 && tickCheck.Ask > 0 {
			msgType = "MASTER_TICK" // Treat as master tick
		}
	}

	switch msgType {
	case "TICK", "MASTER_TICK", "HEARTBEAT":
		// Notify main loop that an agent is active — hydrate execution permit gate
		if p.agentConnectFn != nil {
			p.agentConnectFn(agentID, msgType)
		}
		// Process as tick data (MASTER_TICK has the same structure as TICK)
		var msg AgentTickMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		if msg.Type == "" {
			msg.Type = msgType // Ensure type is set
		}
		p.mu.Lock()
		ch, ok := p.agents[agentID]
		if !ok {
			// Auto-register agent on first tick — creates processing channel
			ch = make(chan *AgentTickMessage, 256)
			p.agents[agentID] = ch
			go p.processAgentTicks(agentID, ch)
		}
		p.mu.Unlock()
		if ch == nil {
			return
		}
		// Write tick to Valkey immediately
		if p.valkeyCache != nil && msg.Bid > 0 && msg.Ask > 0 {
			mid := (msg.Bid + msg.Ask) / 2
			p.valkeyCache.AddPricePoint(mid, time.Now().UTC())
		}
		select {
		case ch <- &msg:
		default:
		}

	case "MARKET_SNAPSHOT":
		// Process as comprehensive market snapshot from Master Node
		var snapshot MarketSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return
		}
		p.snapshotMu.Lock()
		p.lastSnapshot = &snapshot
		p.snapshotCount++
		p.snapshotMu.Unlock()
		// Write directly to Valkey
		if p.valkeyCache != nil {
			p.valkeyCache.SetSnapshot(&snapshot)
			if snapshot.Tick.Bid > 0 && snapshot.Tick.Ask > 0 {
				mid := (snapshot.Tick.Bid + snapshot.Tick.Ask) / 2
				p.valkeyCache.AddPricePoint(mid, time.Now().UTC())
			}
		}

		// Hydrate safety-critical gates from live broker account data (P1-001).
		// When the Windows Agent sends account_info with the snapshot, the
		// exposure and margin gates are hydrated from real broker state.
		if p.brokerAccountHydrateFn != nil {
			// Only hydrate if we have meaningful account data (balance > 0 means a real account is connected)
			if snapshot.AccountInfo.Balance > 0 || snapshot.AccountInfo.Equity > 0 {
				p.brokerAccountHydrateFn(&snapshot.AccountInfo, &snapshot.Positions)
			}
		}

		// CRITICAL: Merge authoritative MT5 indicators and bars into MarketState
		// The Master Node EA computes all 14 indicators natively in MT5.
		// These are AUTHORITATIVE values — do not use locally computed approximations.
		if p.stateMgr != nil {
			p.stateMgr.Update(normalizeSymbol(snapshot.Symbol), func(stateRaw any) {
				// Type-assert to *features.MarketState via reflection-free approach
				// Since stateMgr uses a func(any) update callback, we need a concrete type
				// The actual type is *features.MarketState but we can't import it here
				// Instead we use a helper that the main loop provides
				if mergeFn := p.mergeSnapshotFn; mergeFn != nil {
					mergeFn(stateRaw, &snapshot)
				}
			})
		}

		// CRITICAL: Also create a tick from the snapshot's tick data
		// This ensures ticks flow even when MASTER_TICK messages are overwritten
		// by MARKET_SNAPSHOT in the shared file (Master Node writes both to same file)
		if snapshot.Tick.Bid > 0 && snapshot.Tick.Ask > 0 {
			tickMsg := AgentTickMessage{
				Type:    "MASTER_TICK",
				Symbol:  normalizeSymbol(snapshot.Symbol),
				Bid:     snapshot.Tick.Bid,
				Ask:     snapshot.Tick.Ask,
				Volume:  snapshot.Tick.Volume,
				Source:  snapshot.Source,
				Broker:  snapshot.Broker,
				Account: snapshot.Account,
			}
			p.mu.Lock()
			ch, ok := p.agents[agentID]
			if !ok {
				ch = make(chan *AgentTickMessage, 256)
				p.agents[agentID] = ch
				go p.processAgentTicks(agentID, ch)
			}
			p.mu.Unlock()
			if ch != nil {
				select {
				case ch <- &tickMsg:
				default:
				}
			}
		}

	case "MASTER_INIT":
		// Master Node EA initialized — agent terminal is connected and ready
		if p.agentConnectFn != nil {
			p.agentConnectFn(agentID, "MASTER_INIT")
		}
		// Parse license key from the MASTER_INIT message and validate
		if p.licenseValidateFn != nil {
			var initMsg struct {
				LicenseKey string `json:"license_key"`
			}
			_ = json.Unmarshal(data, &initMsg)
			if initMsg.LicenseKey != "" {
				p.licenseValidateFn(agentID, initMsg.LicenseKey)
			}
		}
	case "MASTER_DEINIT":
		// Master Node lifecycle events — logged but no tick processing needed
		// These are informational messages from the data collection EA

	case "SLIPPAGE_EVENT":
		// NEW v1.07: Slippage monitoring from EA — log and forward
		log.Printf("[AGENT] Slippage event from %s: type=%s", agentID, msgType)

	case "CAPITAL_WARNING":
		// NEW v1.07: Capital protection warning (3% loss)
		log.Printf("[AGENT] CAPITAL WARNING from %s: type=%s", agentID, msgType)

	case "CAPITAL_PROTECTION":
		// NEW v1.07: Capital protection triggered (5% loss) — trading blocked
		log.Printf("[AGENT] CAPITAL PROTECTION from %s: type=%s", agentID, msgType)

	case "CLOSE_ACK":
		// NEW v1.06: Position close acknowledgement from EA
		log.Printf("[AGENT] Close ACK from %s: type=%s", agentID, msgType)

	default:
		// Unknown message type — try to parse as tick (backward compatibility)
		var msg AgentTickMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		if msg.Type == "" {
			return
		}
		p.mu.Lock()
		ch, ok := p.agents[agentID]
		p.mu.Unlock()
		if !ok {
			return
		}
		select {
		case ch <- &msg:
		default:
		}
	}
}

// GetLastSnapshot returns the most recent market snapshot from the Master Node.
func (p *AgentProvider) GetLastSnapshot() interface{} {
	p.snapshotMu.RLock()
	defer p.snapshotMu.RUnlock()
	return p.lastSnapshot
}

// GetSnapshotCount returns the total number of snapshots received.
func (p *AgentProvider) GetSnapshotCount() uint64 {
	p.snapshotMu.RLock()
	defer p.snapshotMu.RUnlock()
	return p.snapshotCount
}
