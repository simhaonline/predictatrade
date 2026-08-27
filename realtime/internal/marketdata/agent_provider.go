// Package marketdata — AgentProvider receives real tick data from Windows MT5 Agent.
// Architecture: MT5 Terminal → Windows Agent → wss://live.predictatrade.com/ws/v1/agent → Go RT engine
// This provider does NOT generate fake data. It only processes real ticks from connected agents.
package marketdata

import (
	"log"
	"encoding/json"
	"context"
	"math"
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

// ParseSnapshotTime exposes MQL snapshot timestamp parsing (ISO8601 preferred,
// gateway-time fallback) so consumers merging snapshot data into market state
// can carry genuine source timestamps instead of processing timestamps.
func ParseSnapshotTime(s string) time.Time {
	return parseMQLTimestamp(s)
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
	// BrokerOffset is the broker's UTC offset in hours, reported live by the
	// Master Node (TimeGMTOffset). Used to align candles to broker session TF.
	BrokerOffset int `json:"broker_offset"`
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
	// BrokerOffset is the broker's UTC offset in hours, reported live by the
	// Master Node (TimeGMTOffset). Authoritative broker session timezone.
	BrokerOffset int `json:"broker_offset"`
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
	TotalPositions int64             `json:"total_positions"`
	BuyCount        int64             `json:"buy_count"`
	SellCount       int64             `json:"sell_count"`
	TotalLots       float64           `json:"total_lots"`
	FloatingProfit  float64           `json:"floating_profit"`
	// Per-position details for server-side SL verification (v1.09+)
	Details         []PositionDetail  `json:"details,omitempty"`
}

// PositionDetail captures individual position SL/TP for server-side enforcement.
type PositionDetail struct {
	Ticket  int64   `json:"ticket"`
	Magic   int64   `json:"magic"`
	Type    string  `json:"type"`    // "BUY" or "SELL"
	Volume  float64 `json:"volume"`
	OpenPx  float64 `json:"open_price"`
	SL      float64 `json:"sl"`
	TP      float64 `json:"tp"`
	Profit  float64 `json:"profit"`
	Symbol  string  `json:"symbol"`
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
	valkeyCache interface{ SetSnapshot(interface{}) error; SetLastSnapshot(interface{}) error; SetMarketState(interface{}) error; AddPricePoint(float64, time.Time) error }

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
	// deviceID (when provided by the agent) is the control-plane device id
	// (licensing.devices.id) so the engine can correlate its live agent
	// connection to a dashboard-visible device row.
	licenseValidateFn func(agentID, licenseKey, deviceID string) LicenseValidationResult

	// TradeResultFn — receives EA exit-reconciliation records (TRADE_RESULT)
	// for persistence into the expected-vs-actual outcome table. Set by main.go.
	tradeResultFn   func(agentID string, data []byte)
	executionAckFn  func(agentID string, data []byte)

	// Broker session alignment. The Master Node sends the broker's server
	// time per tick; we derive the broker UTC offset from it so candles align
	// to BROKER session boundaries (not UTC). cfgOffset is an operator override
	// (BROKER_UTC_OFFSET); otherwise the offset is taken from the Master Node's
	// authoritative live time (masterOffset) and falls back to auto-detection.
	cfgOffset      int
	brokerOffset   atomic.Int32 // auto-detected from naive broker-local ticks
	masterOffset   atomic.Int32 // authoritative offset reported by Master Node
	offsetMu       sync.Mutex
	offsetSamples  map[int]int
}

// SetConfiguredOffset overrides auto-detection with an explicit broker UTC
// offset (hours). 0 enables detection from the Master Node live time.
func (p *AgentProvider) SetConfiguredOffset(hours int) {
	p.cfgOffset = hours
}

// BrokerOffsetHours returns the broker UTC offset in hours (e.g. 3 = UTC+3).
// Operator override (BROKER_UTC_OFFSET) wins; otherwise the authoritative
// offset collected live from the Master Node is used, then auto-detection.
func (p *AgentProvider) BrokerOffsetHours() int {
	if p.cfgOffset != 0 {
		return p.cfgOffset
	}
	if mo := int(p.masterOffset.Load()); mo != 0 {
		return mo
	}
	return int(p.brokerOffset.Load())
}

// ObserveMasterOffset records the broker UTC offset reported live by the Master
// Node (derived from TimeGMTOffset on the EA). This is the authoritative source
// of the broker session timezone so the engine works on Broker TF, not UTC.
func (p *AgentProvider) ObserveMasterOffset(hours int) {
	if hours < -12 || hours > 14 {
		return
	}
	if p.cfgOffset != 0 {
		return // operator override always wins
	}
	p.masterOffset.Store(int32(hours))
	log.Printf("[marketdata] broker UTC offset set from Master Node live time = %d (candles align to broker sessions)", hours)
}

// BrokerNow returns the current time in the broker's session timezone, collected
// live from the Master Node. This is the engine's authoritative "now" for any
// time-of-day / session / candle-completion logic so it runs on Broker TF.
func (p *AgentProvider) BrokerNow() time.Time {
	return time.Now().UTC().Add(time.Duration(p.BrokerOffsetHours()) * time.Hour)
}

// observeOffset records a candidate offset from a live tick and promotes it to
// the stable broker offset once enough consistent evidence arrives.
func (p *AgentProvider) observeOffset(cand int) {
	if cand < -12 || cand > 14 {
		return
	}
	p.offsetMu.Lock()
	defer p.offsetMu.Unlock()
	if p.brokerOffset.Load() != 0 {
		return // already locked in
	}
	p.offsetSamples[cand]++
	best, bestN := 0, 0
	for k, n := range p.offsetSamples {
		if n > bestN {
			bestN = n
			best = k
		}
	}
	if bestN >= 5 {
		p.brokerOffset.Store(int32(best))
		log.Printf("[marketdata] auto-detected broker UTC offset = %d (from live Master Node ticks)", best)
	}
}

// resolveSourceTime converts the EA's MT5 server timestamp into a true UTC
// time while preserving broker-session alignment. For naive broker-local
// timestamps (old EAs) it derives the UTC offset; ISO8601 UTC timestamps are
// used as-is and aligned downstream by the aggregator using the broker offset.
func (p *AgentProvider) resolveSourceTime(raw string) time.Time {
	// New EAs send ISO8601 UTC — already a true UTC instant.
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC()
	}
	// Old EAs send naive broker-local time "YYYY-MM-DD HH:MM:SS" (dots/dashes).
	dash := strings.ReplaceAll(raw, ".", "-")
	if t, err := time.Parse("2006-01-02 15:04:05", dash); err == nil {
		off := p.currentBrokerOffset(t)
		// broker-local wall clock -> true UTC
		return t.Add(-time.Duration(off) * time.Hour)
	}
	return time.Now().UTC()
}

// currentBrokerOffset returns the effective broker offset for a naive
// broker-local tick time, auto-detecting it on the fly when not overridden.
func (p *AgentProvider) currentBrokerOffset(brokerLocal time.Time) int {
	if p.cfgOffset != 0 {
		return p.cfgOffset
	}
	// brokerLocal - gatewayUTC ≈ offset (latency is sub-second, rounds away).
	cand := int(math.Round(brokerLocal.Sub(time.Now().UTC()).Hours()))
	p.observeOffset(cand)
	return p.BrokerOffsetHours()
}

func truncateForLog(data []byte) string {
	if len(data) > 300 {
		return string(data[:300]) + "...(truncated)"
	}
	return string(data)
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

func (p *AgentProvider) SetValkeyCache(v interface{ SetSnapshot(interface{}) error; SetLastSnapshot(interface{}) error; SetMarketState(interface{}) error; AddPricePoint(float64, time.Time) error }) {
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
func (p *AgentProvider) SetLicenseValidateFn(fn func(agentID, licenseKey, deviceID string) LicenseValidationResult) {
	p.licenseValidateFn = fn
}

func (p *AgentProvider) GetLicenseValidateFn() func(agentID, licenseKey, deviceID string) LicenseValidationResult {
	return p.licenseValidateFn
}

// SetTradeResultFn registers the exit-reconciliation callback (TRADE_RESULT).
func (p *AgentProvider) SetTradeResultFn(fn func(agentID string, data []byte)) {
	p.tradeResultFn = fn
}

// SetExecutionAckFn registers the execution acknowledgement callback (EXECUTION_ACK).
// The server uses this to verify that trades were placed with the correct SL/TP.
func (p *AgentProvider) SetExecutionAckFn(fn func(agentID string, data []byte)) {
	p.executionAckFn = fn
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

			// Resolve the MT5 server timestamp into true UTC while preserving the
			// broker's session alignment (auto-detects the broker UTC offset from
			// live Master Node ticks so candles align to broker TFs, not UTC).
			sourceTime := p.resolveSourceTime(msg.Timestamp)
			// Collect the broker UTC offset reported live by the Master Node so
			// the engine runs on Broker TF rather than UTC.
			if msg.BrokerOffset != 0 {
				p.ObserveMasterOffset(msg.BrokerOffset)
			}
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
		// Collect the broker UTC offset reported live by the Master Node so the
		// engine runs on Broker TF rather than UTC.
		if snapshot.BrokerOffset != 0 {
			p.ObserveMasterOffset(snapshot.BrokerOffset)
		}
		// Write directly to Valkey
		if p.valkeyCache != nil {
			p.valkeyCache.SetSnapshot(&snapshot)
			p.valkeyCache.SetLastSnapshot(&snapshot)
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
				// Carry the Master Node's live broker offset onto the synthetic
				// tick so ProcessTick also observes it (belt-and-suspenders).
				BrokerOffset: snapshot.BrokerOffset,
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
				DeviceID   string `json:"device_id"`
				NoLicense  bool   `json:"no_license"`
			}
			_ = json.Unmarshal(data, &initMsg)
			if initMsg.LicenseKey != "" && !initMsg.NoLicense {
				p.licenseValidateFn(agentID, initMsg.LicenseKey, initMsg.DeviceID)
			}
		}

	case "LICENSE_CHECK":
		// Signal EA's license check — forwarded by the Windows Agent pipe manager.
		// This is the PRIMARY license validation path because the Master Node EA
		// sends "no_license":true and no license_key in its MASTER_INIT message.
		// The signal EA's INIT/LICENSE_CHECK carries the real license_key.
		if p.agentConnectFn != nil {
			p.agentConnectFn(agentID, "LICENSE_CHECK")
		}
		if p.licenseValidateFn != nil {
			var licMsg struct {
				LicenseKey string `json:"license_key"`
				DeviceID   string `json:"device_id"`
			}
			_ = json.Unmarshal(data, &licMsg)
			if licMsg.LicenseKey != "" {
				log.Printf("[LICENSE_CHECK] agent=%s license_key=%s... — validating", agentID, licMsg.LicenseKey[:min(12, len(licMsg.LicenseKey))])
				p.licenseValidateFn(agentID, licMsg.LicenseKey, licMsg.DeviceID)
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

	case "EXECUTION_ACK":
		// EA sends EXECUTION_ACK after placing an order — contains the actual
		// entry, SL, TP, ticket, magic. The server verifies SL matches what was sent.
		log.Printf("[AGENT] Execution ACK from %s: %s", agentID, truncateForLog(data))
		if p.executionAckFn != nil {
			p.executionAckFn(agentID, data)
		}

	case "CLOSE_ACK":
		// NEW v1.06: Position close acknowledgement from EA
		log.Printf("[AGENT] Close ACK from %s: type=%s", agentID, msgType)

	case "TRADE_RESULT":
		// EA v1.08 exit reconciliation (prompt.md Bug 5). Persist the outcome
		// so expected-vs-actual can be measured per strategy. Must be handled
		// explicitly — the default branch would otherwise parse it as a tick.
		log.Printf("[AGENT] Trade result from %s: %s", agentID, truncateForLog(data))
		if p.tradeResultFn != nil {
			p.tradeResultFn(agentID, data)
		}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
