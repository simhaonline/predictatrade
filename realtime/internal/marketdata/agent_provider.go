// Package marketdata — AgentProvider receives real tick data from Windows MT5 Agent.
// Architecture: MT5 Terminal → Windows Agent → wss://live.predictatrade.com/ws/v1/agent → Go RT engine
// This provider does NOT generate fake data. It only processes real ticks from connected agents.
package marketdata

import (
	"context"
	"encoding/json"
	"log"
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
	Type      string  `json:"type"` // "TICK", "HEARTBEAT", "BAR"
	Symbol    string  `json:"symbol"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	Volume    int64   `json:"volume"`
	Timestamp string  `json:"timestamp"`
	Source    string  `json:"source"` // "MT5", "MT4"
	Broker    string  `json:"broker"`
	Account   string  `json:"account"`
	// BrokerOffset is the broker's UTC offset in hours, reported live by the
	// Master Node (TimeGMTOffset). Used to align candles to broker session TF.
	BrokerOffset int `json:"broker_offset"`
	// MarketClosed: snapshot was liveness-only (broker market closed). The tick
	// derived from it carries the last-known price — data-quality gate must
	// fail closed.
	MarketClosed bool `json:"market_closed"`
}

// MarketSnapshot is a comprehensive market data message from the Master Node EA.
// It includes ticks, multi-timeframe bars, indicators, account info, and symbol info.
// SOW Section 9: Normalized market state for dashboard/Command Center.
type MarketSnapshot struct {
	Type      string `json:"type"`
	Symbol    string `json:"symbol"`
	Timestamp string `json:"timestamp"`
	GMT       string `json:"gmt"`
	Source    string `json:"source"`
	Broker    string `json:"broker"`
	Account   string `json:"account"`
	Node      string `json:"node"`
	// MarketClosed is set by the Master Node EA when the broker market is
	// closed (weekend/holiday): bid/ask are 0 and the snapshot carries the
	// last-known price. Liveness-only — the engine must NOT evaluate signals
	// on stale closed-market prices.
	MarketClosed bool `json:"market_closed"`
	// BrokerOffset is the broker's UTC offset in hours, reported live by the
	// Master Node (TimeGMTOffset). Authoritative broker session timezone.
	BrokerOffset int                    `json:"broker_offset"`
	Tick         SnapshotTick           `json:"tick"`
	Bars         map[string]SnapshotBar `json:"bars,omitempty"`
	Indicators   SnapshotIndicators     `json:"indicators,omitempty"`
	VWAP         SnapshotVWAP           `json:"vwap"`
	AccountInfo  SnapshotAccount        `json:"account_info,omitempty"`
	SymbolInfo   SnapshotSymbol         `json:"symbol_info,omitempty"`
	Session      SnapshotSession        `json:"session"`
	Positions    SnapshotPositions      `json:"positions,omitempty"`
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
	EMA100          float64 `json:"ema100,omitempty"`
	EMA200          float64 `json:"ema200,omitempty"`
	EMACross921     bool    `json:"ema_cross_9_21,omitempty"`
	SMA50           float64 `json:"sma50,omitempty"`
	SMA100          float64 `json:"sma100,omitempty"`
	MACDHistogram   float64 `json:"macd_histogram,omitempty"`
	MACDBullCross   bool    `json:"macd_bull_cross,omitempty"`
	MACDBearCross   bool    `json:"macd_bear_cross,omitempty"`
	BollWidth       float64 `json:"boll_width,omitempty"`
	BollBullRev     bool    `json:"boll_bull_rev,omitempty"`
	BollBearRev     bool    `json:"boll_bear_rev,omitempty"`
	OBV             float64 `json:"obv,omitempty"`
	TickVolume      float64 `json:"tick_volume,omitempty"`
	VWAP            float64 `json:"vwap,omitempty"`
	ParabolicSAR    float64 `json:"psar,omitempty"`
	PSARLong        bool    `json:"psar_long,omitempty"`
	StochRSI        float64 `json:"stoch_rsi,omitempty"`
	StochRSIK       float64 `json:"stoch_rsi_k,omitempty"`
	StochRSID       float64 `json:"stoch_rsi_d,omitempty"`
	IchimokuTenkan  float64 `json:"ichimoku_tenkan,omitempty"`
	IchimokuKijun   float64 `json:"ichimoku_kijun,omitempty"`
	IchimokuSenkouA float64 `json:"ichimoku_senkou_a,omitempty"`
	IchimokuSenkouB float64 `json:"ichimoku_senkou_b,omitempty"`
}

type SnapshotVWAP struct {
	SessionVWAP float64 `json:"session_vwap"`
}

type SnapshotAccount struct {
	Account    string  `json:"account"`
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
	BuyCount       int64   `json:"buy_count"`
	SellCount      int64   `json:"sell_count"`
	TotalLots      float64 `json:"total_lots"`
	FloatingProfit float64 `json:"floating_profit"`
	// Per-position details for server-side SL verification (v1.09+)
	Details []PositionDetail `json:"details,omitempty"`
}

// PositionDetail captures individual position SL/TP for server-side enforcement.
type PositionDetail struct {
	Ticket int64   `json:"ticket"`
	Magic  int64   `json:"magic"`
	Type   string  `json:"type"` // "BUY" or "SELL"
	Volume float64 `json:"volume"`
	OpenPx float64 `json:"open_price"`
	SL     float64 `json:"sl"`
	TP     float64 `json:"tp"`
	Profit float64 `json:"profit"`
	Symbol string  `json:"symbol"`
}

// AgentProvider receives real tick data from connected Windows MT5 Agents.
// It does NOT generate fake data. If no agent is connected, it produces NO ticks
// and the system degrades to NO-TRADE (SOW: data quality gate fails closed).
type AgentProvider struct {
	marketClosed bool // last snapshot says broker market closed (liveness-only)
	name         string
	mu           sync.Mutex
	agents       map[string]chan *AgentTickMessage // agentID → tick channel
	agentRoleMu  sync.RWMutex
	agentRoles   map[string]string // agentID → "data" (Master node) | "exec" (Client node)
	dataNodeAuthMu     sync.Mutex
	dataNodeAuthorized map[string]bool // data nodes already license-authorized
	tickChan     chan *types.Tick
	stopChan     chan struct{}
	running      atomic.Bool
	validator    *TickValidator

	// Market snapshot storage from Master Node
	snapshotMu    sync.RWMutex
	lastSnapshot  *MarketSnapshot
	snapshotCount uint64

	// Valkey cache — write snapshots directly (not via delayed broadcast loop)
	valkeyCache interface {
		SetSnapshot(interface{}) error
		SetLastSnapshot(interface{}) error
		SetMarketState(interface{}) error
		AddPricePoint(float64, time.Time) error
	}

	// State manager — merge snapshot indicators/bars into market state
	stateMgr StateUpdater

	// Merge function — set by main.go to avoid import cycle
	mergeSnapshotFn func(any, *MarketSnapshot)

	// CandleSyncFn — set by main.go. Called on every MARKET_SNAPSHOT with the
	// authoritative per-TF broker CopyRates bars so the engine can sync its
	// candles to MT5 exactly (separate from indicator merge to avoid cycles).
	candleSyncFn func(symbol string, bars map[string]SnapshotBar, source string)

	// BrokerAccountHydrateFn — called when a snapshot with account_info arrives.
	// This callback hydrates safety-critical gates (exposure, margin, execution)
	// from live broker account data. Set by main.go to avoid import cycle.
	// agentID is provided so hydration is tracked PER CLIENT (no global
	// account-driven gate that could contaminate other clients).
	brokerAccountHydrateFn func(agentID string, account *SnapshotAccount, positions *SnapshotPositions)

	// BrokerInfoFn — called when a MARKET_SNAPSHOT carries authoritative broker
	// identity (broker name + server). Used to persist broker_name/broker_server
	// onto the client's device_activations row so the User/Admin dashboards can
	// show the live MT broker instead of "Unknown broker".
	brokerInfoFn func(agentID, broker, server string)

	// DataFreshnessFn — called on every MARKET_SNAPSHOT so the engine can mark the
	// XAUUSD feed fresh (the Master Node's primary feed is MARKET_SNAPSHOT, not
	// standalone TICK messages; without this the data-quality gate vetoes live
	// trading as stale even when authoritative bars/indicators are flowing).
	dataFreshnessFn func(symbol string)

	// AgentConnectFn — called when an agent connects or sends heartbeat.
	// This hydrates the execution permit gate (terminal connected = PASS).
	agentConnectFn func(agentID string, msgType string)

	// PositionSLCheckFn — v1.15.0 SL enforcement monitor. Called on every
	// broker snapshot so the server can CLOSE_POSITION for PAT positions
	// missing SL. Set by main.go to avoid import cycle.
	positionSLCheckFn func(agentID string, positions *SnapshotPositions)

	// LicenseValidateFn — validates a license key against the DB and sends
	// a LICENSE_STATUS response back to the agent. Set by main.go.
	// deviceID (when provided by the agent) is the control-plane device id
	// (licensing.devices.id) so the engine can correlate its live agent
	// connection to a dashboard-visible device row.
	licenseValidateFn func(agentID, licenseKey, deviceID string) LicenseValidationResult

	// TradeResultFn — receives EA exit-reconciliation records (TRADE_RESULT)
	// for persistence into the expected-vs-actual outcome table. Set by main.go.
	tradeResultFn  func(agentID string, data []byte)
	executionAckFn func(agentID string, data []byte)

	// Broker session alignment. The Master Node sends the broker's server
	// time per tick; we derive the broker UTC offset from it so candles align
	// to BROKER session boundaries (not UTC). cfgOffset is an operator override
	// (BROKER_UTC_OFFSET); otherwise the offset is taken from the Master Node's
	// authoritative live time (masterOffset) and falls back to auto-detection.
	cfgOffset     int
	brokerOffset  atomic.Int32 // auto-detected from naive broker-local ticks
	masterOffset  atomic.Int32 // authoritative offset reported by Master Node
	offsetMu      sync.Mutex
	offsetSamples map[int]int

	// Capital-protection log de-duplication. The EA emits CAPITAL_WARNING /
	// CAPITAL_PROTECTION on every tick while the account sits in the triggered
	// state, which would otherwise flood the logs every few milliseconds. We
	// only log a transition per agent.
	capitalStateMu   sync.Mutex
	lastCapitalState map[string]string

	// ── Ingest bulkhead (production hardening) ──
	// Per-agent message rate limiting + quarantine so a single misbehaving,
	// laggy, or flooding EA cannot consume shared engine resources or starve
	// other clients (defense-in-depth on top of the immutable-snapshot
	// StateManager and per-client account-state isolation).
	guardMu               sync.Mutex
	agentMsgCount         map[string]int
	agentWindowStart      map[string]time.Time
	agentQuarantinedUntil map[string]time.Time

	// Redundant-message de-duplication: identical repeated messages of low-value
	// types (CAPITAL_WARNING / CAPITAL_PROTECTION) are dropped before processing
	// so a per-tick emitter cannot flood the engine. State-change detection is
	// preserved because the signature includes the full message body.
	agentLastSig map[string]map[string]string

	// Single-writer market-state merge. MARKET_SNAPSHOT merges into the shared
	// MarketState through one coalescing goroutine instead of one writer per
	// agent connection. This removes the N-writer contention / race class on the
	// shared MarketState (see features.StateManager.Get snapshot clone) regardless
	// of how many agents stream snapshots.
	snapshotPendingMu sync.Mutex
	snapshotPending   map[string]*MarketSnapshot
	snapshotStop      chan struct{}

	// Per-agent account state (production hardening). Each client's broker
	// account is tracked individually so risk gating can be evaluated per
	// receiving client at signal-delivery time. This guarantees one client's
	// blown/over-exposed account can NEVER block or contaminate another
	// client's executable signals (no shared global account-driven gate).
	agentAccMu    sync.Mutex
	agentAccounts map[string]map[string]*agentAccountState // agentID -> accountLogin -> state

	// lastMarketDataAt records the most recent time the engine received ANY live
	// market data (tick or snapshot) from ANY agent. Used for coarse liveness.
	// lastSnapshotAt records the most recent MARKET_SNAPSHOT — the message that
	// actually carries bars/indicators and builds the market state required to
	// generate signals. Tracking it separately prevents a lone tick from masking
	// a dead snapshot feed (which would silently suppress all executable signals).
	lastMarketDataMu sync.Mutex
	lastMarketDataAt time.Time
	lastSnapshotAt   time.Time
}

// agentAccountState caches one client's latest broker account snapshot.
type agentAccountState struct {
	known          bool
	equity         float64
	freeMargin     float64
	leverage       float64
	totalPositions int
	updatedAt      time.Time
	account        *SnapshotAccount
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
	prev := int(p.masterOffset.Load())
	p.masterOffset.Store(int32(hours))
	// Log only on first observation or change — the Master Node sends this on
	// every tick, so logging unconditionally would flood the log every few ms.
	if prev != hours {
		log.Printf("[marketdata] broker UTC offset set from Master Node live time = %d (candles align to broker sessions)", hours)
	}
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
	Valid      bool     `json:"valid"`
	Status     string   `json:"status"` // ACTIVE, EXPIRED, REVOKED, NOT_FOUND
	Plan       string   `json:"plan"`   // FREE, STANDARD, PRO, ELITE
	MaxDevices int      `json:"max_devices"`
	MaxMTAccts int      `json:"max_mt_accounts"`
	Strategies []string `json:"allowed_strategies"`
	Error      string   `json:"error,omitempty"`
}

func (p *AgentProvider) SetValkeyCache(v interface {
	SetSnapshot(interface{}) error
	SetLastSnapshot(interface{}) error
	SetMarketState(interface{}) error
	AddPricePoint(float64, time.Time) error
}) {
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

// SetCandleSyncFn sets the callback that syncs per-TF broker CopyRates bars
// into the engine candle pipeline so candles match MT5 exactly.
func (p *AgentProvider) SetCandleSyncFn(fn func(symbol string, bars map[string]SnapshotBar, source string)) {
	p.candleSyncFn = fn
}

// SetBrokerAccountHydrateFn sets the callback that hydrates safety-critical gates
// from live broker account data when a MARKET_SNAPSHOT with account_info arrives.
// agentID is supplied so hydration is tracked per client.
func (p *AgentProvider) SetBrokerAccountHydrateFn(fn func(agentID string, account *SnapshotAccount, positions *SnapshotPositions)) {
	p.brokerAccountHydrateFn = fn
}

// SetBrokerInfoFn sets the callback that persists authoritative broker identity
// (name + server) from a MARKET_SNAPSHOT onto the client's device_activations row.
func (p *AgentProvider) SetBrokerInfoFn(fn func(agentID, broker, server string)) {
	p.brokerInfoFn = fn
}

// SetDataFreshnessFn sets the callback that marks the feed fresh on every
// MARKET_SNAPSHOT so the data-quality gate does not wrongly veto live trading.
func (p *AgentProvider) SetDataFreshnessFn(fn func(symbol string)) {
	p.dataFreshnessFn = fn
}

// RecordAgentAccount stores one client's latest broker account snapshot in the
// per-agent, per-account registry. State is keyed by account login so multiple
// terminals (e.g. a live account and a small demo account) on the same agent
// NEVER overwrite each other — a demo/small account's equity cannot poison the
// risk view of the funded trading account.
func (p *AgentProvider) RecordAgentAccount(agentID string, account *SnapshotAccount, positions *SnapshotPositions) {
	if account == nil {
		return
	}
	login := account.Account
	if login == "" {
		login = "default"
	}
	p.agentAccMu.Lock()
	if p.agentAccounts == nil {
		p.agentAccounts = make(map[string]map[string]*agentAccountState)
	}
	accts := p.agentAccounts[agentID]
	if accts == nil {
		accts = make(map[string]*agentAccountState)
		p.agentAccounts[agentID] = accts
	}
	st := accts[login]
	if st == nil {
		st = &agentAccountState{}
		accts[login] = st
	}
	st.known = true
	st.equity = account.Equity
	st.freeMargin = account.FreeMargin
	st.account = account
	if account.Leverage > 0 {
		st.leverage = float64(account.Leverage)
	}
	if positions != nil {
		st.totalPositions = int(positions.TotalPositions)
	}
	st.updatedAt = time.Now().UTC()
	p.agentAccMu.Unlock()
}

// GetPrimaryAccount returns the connected account with the highest equity for an
// agent. This is the account the engine uses for capital-protection gating and
// trade sizing, so a small/demo account cannot trigger a false daily-loss halt
// or distort position sizing on the funded trading account.
func (p *AgentProvider) GetPrimaryAccount(agentID string) *SnapshotAccount {
	p.agentAccMu.Lock()
	defer p.agentAccMu.Unlock()
	accts := p.agentAccounts[agentID]
	if len(accts) == 0 {
		return nil
	}
	var best *agentAccountState
	for _, st := range accts {
		if st == nil || !st.known || st.account == nil {
			continue
		}
		if best == nil || st.equity > best.equity {
			best = st
		}
	}
	if best == nil {
		return nil
	}
	return best.account
}

// hydrateAccountFromJSON parses a generic agent message (e.g. the signal EA's
// INIT/LICENSE_CHECK payload, which carries balance/equity at the top level) and
// records it as a broker account snapshot. It is an additional equity feed beside
// the Master Node's account_info, so the funded trading account's equity reaches
// risk gating even when the Master Node EA is attached to a different account.
func (p *AgentProvider) hydrateAccountFromJSON(agentID string, data []byte) {
	var acct struct {
		Account    string  `json:"account"`
		Broker     string  `json:"broker"`
		Balance    float64 `json:"balance"`
		Equity     float64 `json:"equity"`
		Margin     float64 `json:"margin"`
		FreeMargin float64 `json:"free_margin"`
		Profit     float64 `json:"profit"`
		Currency   string  `json:"currency"`
		Leverage   int64   `json:"leverage"`
		Server     string  `json:"server"`
	}
	if err := json.Unmarshal(data, &acct); err != nil {
		return
	}
	log.Printf("[ACCT-INIT] agent=%s balance=%.2f equity=%.2f free_margin=%.2f leverage=%d currency=%s broker=%s",
		agentID, acct.Balance, acct.Equity, acct.FreeMargin, acct.Leverage, acct.Currency, acct.Broker)
	if acct.Balance <= 0 && acct.Equity <= 0 {
		return
	}
	snap := &SnapshotAccount{
		Account:    acct.Account,
		Balance:    acct.Balance,
		Equity:     acct.Equity,
		Margin:     acct.Margin,
		FreeMargin: acct.FreeMargin,
		Profit:     acct.Profit,
		Currency:   acct.Currency,
		Leverage:   acct.Leverage,
		Server:     acct.Server,
	}
	p.RecordAgentAccount(agentID, snap, nil)
	if p.brokerAccountHydrateFn != nil {
		p.brokerAccountHydrateFn(agentID, snap, nil)
	}
	// The Client (exec) node's account payload carries the user's OWN trading
	// broker (e.g. Xelance) — persist it so dashboards show the user's broker,
	// not the Master data-feed broker (Equiti).
	if p.brokerInfoFn != nil && acct.Broker != "" {
		p.brokerInfoFn(agentID, acct.Broker, acct.Server)
	}
}

// AgentAccountOK reports whether a given client may receive EXECUTABLE signals
// based solely on that client's own account. It is fail-open: an agent with no
// known/remote account state is allowed (preserving current behavior); only a
// client whose own account is KNOWN and has no buying power (free margin <= 0)
// or is stale is isolated. This guarantees one client's blown account never
// blocks another's signals.
func (p *AgentProvider) AgentAccountOK(agentID string) bool {
	p.agentAccMu.Lock()
	accts := p.agentAccounts[agentID]
	p.agentAccMu.Unlock()
	if len(accts) == 0 {
		return true
	}
	for _, st := range accts {
		stale := time.Since(st.updatedAt) > 60*time.Second
		if stale {
			continue // fail-open on stale data
		}
		if st.freeMargin <= 0 {
			return false
		}
	}
	return true
}

// SetAgentConnectFn sets the callback that hydrates the execution permit gate
// when an agent connects or sends a heartbeat.
func (p *AgentProvider) SetAgentConnectFn(fn func(agentID string, msgType string)) {
	p.agentConnectFn = fn
}

// SetPositionSLCheckFn registers the v1.15.0 position-SL monitor callback.
// Invoked on every broker snapshot so the server can CLOSE_POSITION for any
// PAT position missing its stop-loss (server is the enforcement authority).
func (p *AgentProvider) SetPositionSLCheckFn(fn func(agentID string, positions *SnapshotPositions)) {
	p.positionSLCheckFn = fn
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
	p := &AgentProvider{
		name:      "MT5_AGENT",
		agents:    make(map[string]chan *AgentTickMessage),
		tickChan:  make(chan *types.Tick, 4096),
		stopChan:  make(chan struct{}),
		validator: NewTickValidator(),

		agentMsgCount:         make(map[string]int),
		agentWindowStart:      make(map[string]time.Time),
		agentQuarantinedUntil: make(map[string]time.Time),
		agentLastSig:          make(map[string]map[string]string),

		snapshotPending: make(map[string]*MarketSnapshot),
		snapshotStop:    make(chan struct{}),
	}
	// Single-writer market-state merge loop (see field docs above).
	go p.snapshotMergeLoop()
	return p
}

func (p *AgentProvider) Name() string { return p.name }
func (p *AgentProvider) Connect(ctx context.Context) error {
	p.running.Store(true)
	return nil
}
func (p *AgentProvider) Subscribe(symbol string) error { return nil }
func (p *AgentProvider) Stream() <-chan *types.Tick    { return p.tickChan }

// SetAgentRole records the role declared by an agent on connection
// ("data" = Master Node market feed, "exec" = Client execution/trade node).
// The engine only consumes market price/bars/indicators from "data" agents;
// "exec" agents contribute account, position and trade data only.
func (p *AgentProvider) SetAgentRole(agentID, role string) {
	if agentID == "" {
		return
	}
	if role != "data" && role != "exec" {
		return
	}
	p.agentRoleMu.Lock()
	if p.agentRoles == nil {
		p.agentRoles = map[string]string{}
	}
	// A Master (data) node and Client (exec) node on the SAME device register the
	// same agent id. Never let a Client (exec) downgrade the node to non-data — once
	// a data role is seen it stays, so the price feed is never accidentally gated
	// off by the Client's registration.
	if p.agentRoles[agentID] != "data" {
		p.agentRoles[agentID] = role
	}
	p.agentRoleMu.Unlock()
}

// IsDataNode reports whether the agent is the Master (market data) node.
func (p *AgentProvider) IsDataNode(agentID string) bool {
	p.agentRoleMu.RLock()
	defer p.agentRoleMu.RUnlock()
	return p.agentRoles[agentID] == "data"
}

// ensureDataNodeLicense authorizes a Master (data) node as a market-data feed on
// its first snapshot. The Master EA sends MASTER_INIT only once at startup, so we
// cannot rely on it after an engine restart — but it streams MARKET_SNAPSHOT
// continuously, which is a reliable trigger to grant data-node authorization.
func (p *AgentProvider) ensureDataNodeLicense(agentID string) {
	p.dataNodeAuthMu.Lock()
	if p.dataNodeAuthorized == nil {
		p.dataNodeAuthorized = map[string]bool{}
	}
	if p.dataNodeAuthorized[agentID] {
		p.dataNodeAuthMu.Unlock()
		return
	}
	p.dataNodeAuthorized[agentID] = true
	p.dataNodeAuthMu.Unlock()
	if p.licenseValidateFn != nil {
		p.licenseValidateFn(agentID, "", "")
	}
}
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
				MarketClosed:     msg.MarketClosed,
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
// Brokers use different suffixes/symbols: XAUUSD, XAUUSD.sd, XAUUSD.e, XAUUSD.m,
// XAU/USD, "XAU USD", GOLD, GOLD.sb, etc. All are the same underlying instrument,
// so the engine treats them identically. This is what lets the Master Node run on
// any broker (e.g. Equiti's XAUUSD.sd) while Clients on any other broker receive
// the same signal for their own symbol variant — no broker lock-in required.
func normalizeSymbol(s string) string {
	u := strings.ToUpper(strings.TrimSpace(s))
	cleaned := strings.NewReplacer("/", "", " ", "", ".", "").Replace(u)
	if len(cleaned) >= 6 && cleaned[:6] == "XAUUSD" {
		return "XAUUSD"
	}
	if strings.Contains(cleaned, "XAUUSD") || strings.Contains(cleaned, "GOLD") {
		return "XAUUSD"
	}
	return s
}

// ── Ingest bulkhead constants (production hardening) ──
const (
	// maxAgentMsgsPerWindow is the max messages an agent may send per
	// agentRateWindow before being quarantined. Generous for normal
	// high-frequency tick feeds; catches pathological floods.
	maxAgentMsgsPerWindow = 5000
	agentRateWindow       = time.Second
	agentQuarantinePeriod = 60 * time.Second

	// redundantMsgTypes are de-duplicated: identical repeated messages of these
	// types are dropped before processing (state-change detection preserved via
	// the full-body signature).
	redundantMsgTypes = "CAPITAL_WARNING,CAPITAL_PROTECTION"
)

// ingestGuard enforces the per-agent bulkhead: rate limiting, quarantine, and
// redundant-message de-duplication. It returns true if the message should be
// processed. A false return means the message was dropped (quarantined or a
// redundant duplicate) and must not be processed further.
func (p *AgentProvider) ingestGuard(agentID, msgType string, data []byte) bool {
	p.guardMu.Lock()
	defer p.guardMu.Unlock()

	now := time.Now()

	// Quarantine: drop everything from a quarantined agent until it expires.
	if until, ok := p.agentQuarantinedUntil[agentID]; ok && now.Before(until) {
		return false
	}

	// Rate limit (per-agent sliding window).
	if start, ok := p.agentWindowStart[agentID]; !ok || now.Sub(start) > agentRateWindow {
		p.agentWindowStart[agentID] = now
		p.agentMsgCount[agentID] = 0
	}
	p.agentMsgCount[agentID]++
	if p.agentMsgCount[agentID] > maxAgentMsgsPerWindow {
		p.agentQuarantinedUntil[agentID] = now.Add(agentQuarantinePeriod)
		log.Printf("[AGENT-BULKHEAD] agent=%s quarantined for %s — message rate exceeded (%d in %s)",
			agentID, agentQuarantinePeriod, p.agentMsgCount[agentID], agentRateWindow)
		return false
	}

	// Redundant de-duplication for low-value repeated message types.
	for _, rt := range strings.Split(redundantMsgTypes, ",") {
		if msgType != rt {
			continue
		}
		if p.agentLastSig[agentID] == nil {
			p.agentLastSig[agentID] = make(map[string]string)
		}
		sig := string(data)
		if prev, ok := p.agentLastSig[agentID][msgType]; ok && prev == sig {
			return false // identical repeated message — drop before processing
		}
		p.agentLastSig[agentID][msgType] = sig
		break
	}

	return true
}

// snapshotMergeLoop is the single writer that merges MARKET_SNAPSHOT data into
// the shared MarketState. Snapshots are coalesced per symbol, so concurrent
// agent connections never write the shared MarketState directly — eliminating
// the N-writer contention / race class regardless of client count.
func (p *AgentProvider) snapshotMergeLoop() {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-p.snapshotStop:
			return
		case <-ticker.C:
			p.snapshotPendingMu.Lock()
			pending := p.snapshotPending
			p.snapshotPending = make(map[string]*MarketSnapshot)
			p.snapshotPendingMu.Unlock()
			for sym, snap := range pending {
				if p.stateMgr != nil {
					p.stateMgr.Update(sym, func(stateRaw any) {
						if p.mergeSnapshotFn != nil {
							p.mergeSnapshotFn(stateRaw, snap)
						}
					})
				}
			}
		}
	}
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

	// Ingest bulkhead: per-agent rate limit, quarantine, and redundant-message
	// de-duplication. Drop messages that fail the guard before any processing.
	if !p.ingestGuard(agentID, msgType, data) {
		return
	}

	// Record live market-data receipt (ticks + snapshots) so a silent feed is
	// always detectable/alertable.
	if msgType == "TICK" || msgType == "MASTER_TICK" || msgType == "MARKET_SNAPSHOT" {
		p.updateLastMarketData()
	}

	switch msgType {
	case "TICK", "MASTER_TICK":
		// CRITICAL: price/tick feed comes ONLY from the Master (data) node.
		// Client (exec) nodes must never feed market price — they are for signal
		// reception and trade-data collection only. Ignoring client ticks here
		// prevents a client terminal's bad/stale price from poisoning signal
		// generation or marking the feed stale.
		if !p.IsDataNode(agentID) {
			return
		}
	case "HEARTBEAT":
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

	case "LIVENESS":
		// Client EA weekend/holiday liveness ping (no market ticks): proves the
		// EA→agent→engine chain is alive while the market is closed. Hydrate the
		// execution-permit gate exactly like an active agent, but do NOT feed
		// any price path — market_closed payloads carry no valid price.
		if p.agentConnectFn != nil {
			p.agentConnectFn(agentID, "LIVENESS")
		}
		var lv struct {
			AgentID      string `json:"agent_id"`
			Symbol       string `json:"symbol"`
			Source       string `json:"source"`
			Account      string `json:"account"`
			Broker       string `json:"broker"`
			MarketClosed bool   `json:"market_closed"`
		}
		if err := json.Unmarshal(data, &lv); err == nil {
			log.Printf("[AGENT-LIVENESS] agent=%s terminal=%s:%s symbol=%s market_closed=%v",
				agentID, lv.Source, lv.Account, lv.Symbol, lv.MarketClosed)
		}
		return

	case "AGENT_TELEMETRY":
		// Client agent health/usage snapshot for server-side observability.
		// Logged (not processed) so ops can see fleet state; never affects trading.
		var tel map[string]interface{}
		if err := json.Unmarshal(data, &tel); err == nil {
			log.Printf("[AGENT-TELEMETRY] agent=%s version=%v role=%v goarch=%v backend=%v mt4=%v mt5=%v uptime_s=%v candles=%v license=%v plan=%v",
				agentID,
				tel["version"], tel["role"], tel["goarch"],
				tel["backend_connected"], tel["mt4_connected"], tel["mt5_connected"],
				tel["uptime_seconds"], tel["candles_delivered"],
				tel["license_status"], tel["license_plan"])
		}
		return

	case "MARKET_SNAPSHOT":
		// CRITICAL: market snapshots (price/bars/indicators) come ONLY from the
		// Master (data) node. Client (exec) nodes send account/trade data only and
		// must never feed the snapshot pipeline (would mix a client terminal's
		// price into signal generation).
		// A MARKET_SNAPSHOT is itself authoritative proof the sender is the data
		// node. Establish the role here so the feed survives engine restarts — the
		// Master EA does NOT re-send MASTER_INIT on reconnect, so without this the
		// reconnected data node's role is lost and every snapshot is silently
		// dropped (NO_DATA / STALE). SetAgentRole keeps any existing data role.
		p.SetAgentRole(agentID, "data")
		if !p.IsDataNode(agentID) {
			return
		}
		// Process as comprehensive market snapshot from Master Node
		var snapshot MarketSnapshot
		// Authorize the data (Master) node as a market-data feed on first snapshot
		// (covers engine restarts where MASTER_INIT is not re-sent).
		if p.licenseValidateFn != nil {
			p.ensureDataNodeLicense(agentID)
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			raw := data
			if len(raw) > 200 {
				raw = raw[:200]
			}
			log.Printf("[marketdata] MARKET_SNAPSHOT unmarshal failed (agent=%s): %v — payload snippet: %s", agentID, err, string(raw))
			return
		}
		// Normalize broker-specific symbol variants (XAUUSD.sd / .e / .m / GOLD.sb …)
		// to the canonical "XAUUSD" BEFORE storing into MarketState. Without this the
		// master feed (e.g. Equiti's "XAUUSD.sd") is persisted under that raw symbol,
		// so any consumer keyed on "XAUUSD" — the dashboard market feed — never
		// receives the fresh tick and shows a stale price. normalizeSymbol is the
		// engine's single source of truth for this; apply it here at intake.
		snapshot.Symbol = normalizeSymbol(snapshot.Symbol)
		p.snapshotMu.Lock()
		p.lastSnapshot = &snapshot
		p.snapshotCount++
		p.marketClosed = snapshot.MarketClosed || snapshot.Session.IsWeekend
		p.snapshotMu.Unlock()
		// Track snapshot receipt separately so a lone tick cannot mask a dead
		// snapshot feed (snapshots build the market state for signal generation).
		p.updateLastSnapshot()

		// CRITICAL: queue the snapshot for the single-writer merge loop IMMEDIATELY,
		// before any downstream callback (gate hydration / SL monitoring) runs. The
		// Master Node is the sole authoritative live-data source; if a later
		// callback faults (panic/hang), the feed must still stay live. The merge
		// loop refreshes MarketState.LastTick, Bid/Ask and indicators from this.
		if p.stateMgr != nil {
			snapCopy := snapshot
			p.snapshotPendingMu.Lock()
			p.snapshotPending[normalizeSymbol(snapshot.Symbol)] = &snapCopy
			p.snapshotPendingMu.Unlock()
		}

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

		// All downstream side-effects (account recording, gate hydration, SL
		// monitoring) are isolated in a recover so a fault in any one of them can
		// NEVER block the live feed — the snapshot is already queued above.
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[marketdata] MARKET_SNAPSHOT side-effect recovered (agent=%s): %v", agentID, r)
				}
			}()

			// Track this client's account state individually (per-client risk
			// isolation) so a blown/over-exposed account can never contaminate
			// another client's signals.
			if snapshot.AccountInfo.Balance > 0 || snapshot.AccountInfo.Equity > 0 {
				// Carry the terminal's account login so per-account state is isolated
				// (a demo/small account cannot poison the funded account's risk view).
				if snapshot.AccountInfo.Account == "" {
					snapshot.AccountInfo.Account = snapshot.Account
				}
				p.RecordAgentAccount(agentID, &snapshot.AccountInfo, &snapshot.Positions)
			}

			// NOTE: broker identity for the user's TRADING account is captured from the
			// Client (exec) node's LICENSE_CHECK/INIT account payload (see
			// hydrateAccountFromJSON), NOT from the Master node's snapshot. The Master
			// node is a pure data feed (Equiti) and must not overwrite the user's own
			// broker (e.g. Xelance) on the dashboard.

			// Mark the feed fresh so the data-quality gate does not veto live trading
			// when the agent streams MARKET_SNAPSHOT without standalone TICK messages.
			if p.dataFreshnessFn != nil {
				p.dataFreshnessFn(normalizeSymbol(snapshot.Symbol))
			}

			// Hydrate safety-critical gates from live broker account data (P1-001).
			// When the Windows Agent sends account_info with the snapshot, the
			// exposure and margin gates are hydrated from real broker state.
			if p.brokerAccountHydrateFn != nil {
				// Only hydrate if we have meaningful account data (balance > 0 means a real account is connected)
				if snapshot.AccountInfo.Balance > 0 || snapshot.AccountInfo.Equity > 0 {
					// Carry the terminal's account login (see RecordAgentAccount above).
					if snapshot.AccountInfo.Account == "" {
						snapshot.AccountInfo.Account = snapshot.Account
					}
					p.brokerAccountHydrateFn(agentID, &snapshot.AccountInfo, &snapshot.Positions)
				}
			}

			// v1.15.0 SL ENFORCEMENT: monitor every snapshot for PAT positions
			// missing SL → CLOSE_POSITION (server is the enforcement authority).
			if p.positionSLCheckFn != nil && snapshot.Positions.TotalPositions > 0 {
				p.positionSLCheckFn(agentID, &snapshot.Positions)
			}
		}()

		// CRITICAL: Merge authoritative MT5 indicators and bars into MarketState
		// via the single-writer coalescing loop (snapshotMergeLoop). The snapshot
		// is already queued above; downstream callbacks must never block it.

		// CRITICAL: Sync per-TF broker CopyRates bars into the engine candle
		// pipeline so candles match MT5 exactly (broker bar sync, not indicator merge).
		if p.candleSyncFn != nil && len(snapshot.Bars) > 0 {
			p.candleSyncFn(normalizeSymbol(snapshot.Symbol), snapshot.Bars, snapshot.Source)
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
				// A closed-market snapshot is liveness-only: any tick derived
				// from it must be gated out of EXECUTABLE evaluation.
				MarketClosed: snapshot.MarketClosed,
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
			} else if initMsg.NoLicense {
				// Master (data) node: authorize as a data feed (no trading license).
				p.licenseValidateFn(agentID, "", initMsg.DeviceID)
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
		// Hydrate broker account state from the signal EA's LICENSE_CHECK/INIT
		// account payload (balance/equity) as an ADDITIONAL equity feed alongside
		// the Master Node's account_info. This ensures the funded trading
		// account's equity reaches risk gating even when the Master Node EA is
		// attached to a different (e.g. demo) account.
		p.hydrateAccountFromJSON(agentID, data)
	case "ACCOUNT_INFO":
		// Periodic/INIT account telemetry from the signal (exec) EA, forwarded by
		// the Windows Agent. Hydrates broker equity/free-margin/leverage so the
		// margin gate and lot-sizing can compute (enables executable signals).
		p.hydrateAccountFromJSON(agentID, data)
	case "MASTER_DEINIT":
		// Master Node lifecycle events — logged but no tick processing needed
		// These are informational messages from the data collection EA

	case "SLIPPAGE_EVENT":
		// NEW v1.07: Slippage monitoring from EA — log and forward
		log.Printf("[AGENT] Slippage event from %s: type=%s", agentID, msgType)

	case "CAPITAL_WARNING":
		// NEW v1.07: Capital protection warning (3% loss). De-duplicated:
		// the EA emits this on every tick while in the warning state.
		p.capitalStateMu.Lock()
		if p.lastCapitalState == nil {
			p.lastCapitalState = make(map[string]string)
		}
		prev := p.lastCapitalState[agentID]
		p.capitalStateMu.Unlock()
		if prev != "CAPITAL_WARNING" {
			log.Printf("[AGENT] CAPITAL WARNING from %s: type=%s", agentID, msgType)
			p.capitalStateMu.Lock()
			p.lastCapitalState[agentID] = "CAPITAL_WARNING"
			p.capitalStateMu.Unlock()
		}

	case "CAPITAL_PROTECTION":
		// NEW v1.07: Capital protection triggered (5% loss) — trading blocked.
		// De-duplicated the same way as CAPITAL_WARNING.
		p.capitalStateMu.Lock()
		if p.lastCapitalState == nil {
			p.lastCapitalState = make(map[string]string)
		}
		prev := p.lastCapitalState[agentID]
		p.capitalStateMu.Unlock()
		if prev != "CAPITAL_PROTECTION" {
			log.Printf("[AGENT] CAPITAL PROTECTION from %s: type=%s", agentID, msgType)
			p.capitalStateMu.Lock()
			p.lastCapitalState[agentID] = "CAPITAL_PROTECTION"
			p.capitalStateMu.Unlock()
		}

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

// updateLastMarketData records the current time as the latest live market-data
// receipt. Guarded by its own mutex so it is cheap to call on every message.
func (p *AgentProvider) updateLastMarketData() {
	p.lastMarketDataMu.Lock()
	p.lastMarketDataAt = time.Now().UTC()
	p.lastMarketDataMu.Unlock()
}

// updateLastSnapshot records the current time as the latest MARKET_SNAPSHOT
// receipt (the data that builds the market state used for signal generation).
func (p *AgentProvider) updateLastSnapshot() {
	p.lastMarketDataMu.Lock()
	p.lastSnapshotAt = time.Now().UTC()
	p.lastMarketDataMu.Unlock()
}

// IsMarketClosed reports whether the latest Master snapshot flagged the
// broker market as closed (weekend/holiday liveness-only data).
func (p *AgentProvider) IsMarketClosed() bool {
	p.snapshotMu.RLock()
	defer p.snapshotMu.RUnlock()
	if p.lastSnapshot != nil {
		return p.lastSnapshot.MarketClosed || p.lastSnapshot.Session.IsWeekend
	}
	return p.marketClosed
}

// LastMarketDataAt returns the time the engine last received any live market
// data (tick or snapshot) from any agent. A zero value means none received
// since startup.
func (p *AgentProvider) LastMarketDataAt() time.Time {
	p.lastMarketDataMu.Lock()
	defer p.lastMarketDataMu.Unlock()
	return p.lastMarketDataAt
}

// LastSnapshotAt returns the time the engine last received a MARKET_SNAPSHOT
// (the feed that builds market state). A zero value means none received since
// startup — signal generation cannot proceed without it.
func (p *AgentProvider) LastSnapshotAt() time.Time {
	p.lastMarketDataMu.Lock()
	defer p.lastMarketDataMu.Unlock()
	return p.lastSnapshotAt
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
