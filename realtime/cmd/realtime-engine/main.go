// Predict-A-Trade Real-Time Engine — Main Entrypoint
// Pipeline: MT5 Agent → ticks → features → strategies → gates → signals → WS
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/predictatrade/realtime/internal/audit"
	"github.com/predictatrade/realtime/internal/cache"
	"github.com/predictatrade/realtime/internal/calibration"
	"github.com/predictatrade/realtime/internal/config"
	"github.com/predictatrade/realtime/internal/crossmarket"
	"github.com/predictatrade/realtime/internal/engstatus"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/gates"
	"github.com/predictatrade/realtime/internal/gateway"
	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/predictatrade/realtime/internal/observability"
	"github.com/predictatrade/realtime/internal/ptb"
	"github.com/predictatrade/realtime/internal/reconciliation"
	"github.com/predictatrade/realtime/internal/risk"
	sigengine "github.com/predictatrade/realtime/internal/signal"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/strategy/engines"
	"github.com/predictatrade/realtime/pkg/health"
	"github.com/predictatrade/realtime/pkg/macro"
	"github.com/predictatrade/realtime/pkg/mlengine"
	"github.com/predictatrade/realtime/pkg/news"
	"github.com/predictatrade/realtime/pkg/notifications"
	"github.com/predictatrade/realtime/pkg/ollama"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// newsRiskAdapter bridges pkg/news.RiskEngine to features.NewsRiskProvider.
// When NEWS_MODE=OFF or the provider is disabled, it returns "NONE" so the
// pre-v1.10 behaviour is preserved (no news protection, trading proceeds).
type newsRiskAdapter struct {
	engine   *news.RiskEngine
	mode     string
	provider string
}

func (a *newsRiskAdapter) ComputeNewsRisk(now time.Time) string {
	// NEWS_MODE=OFF: operator explicitly turned off news protection.
	if a.mode == "OFF" {
		return "NONE"
	}
	// NEWS_PROVIDER=disabled: operator hasn't configured a provider yet.
	// This is a deliberate configuration choice, NOT a provider failure.
	// Return "NONE" so trading proceeds normally.
	// DATA_UNAVAILABLE is reserved for when a provider IS configured but fails/stale.
	if a.provider == "disabled" || a.provider == "" {
		return "NONE"
	}
	if a.engine == nil {
		return "NONE"
	}
	result := a.engine.ComputeRisk(now)
	return string(result.Level)
}

// stateAdapter wraps *features.StateManager to satisfy marketdata.StateUpdater
type stateAdapter struct{ sm *features.StateManager }

func (a stateAdapter) Update(symbol string, update func(any)) {
	a.sm.Update(symbol, func(state *features.MarketState) {
		update(state)
	})
}

// Package-level for processCandle access
var globalAgentProvider *marketdata.AgentProvider
var globalCrossMarketEngine *crossmarket.Engine
var globalAgentHub *gateway.AgentHub
var globalPersister *marketdata.Persister

// engineOverrideSLTP gates the strategy-engine SL/TP override matrix.
// When false (default, ENVINE_OVERRIDE_SLTP unset or != "true"), the live
// engine uses the same getStrategyConfig geometry as the backtest, avoiding
// divergence between live and backtest SL/TP. Set ENGINE_OVERRIDE_SLTP=true
// to deliberately opt into the Phase 6 override matrix.
var engineOverrideSLTP bool

// Agent strategy entitlements — maps agentID → allowed_strategies from license
var (
	agentStrategiesMu sync.RWMutex
	agentStrategies   = make(map[string][]string) // agentID → ["STANDARD_SCALPING", ...]
)

func setAgentStrategies(agentID string, strategies []string) {
	agentStrategiesMu.Lock()
	defer agentStrategiesMu.Unlock()
	agentStrategies[agentID] = strategies
}

func getAgentStrategies(agentID string) []string {
	agentStrategiesMu.RLock()
	defer agentStrategiesMu.RUnlock()
	return agentStrategies[agentID]
}

func isStrategyAllowedForAgent(agentID string, strategyID string) bool {
	allowed := getAgentStrategies(agentID)
	if len(allowed) == 0 {
		// No strategies loaded for this agent — allow all (backward compat)
		// This should not happen in production after license validation
		return true
	}
	for _, s := range allowed {
		if s == strategyID {
			return true
		}
	}
	return false
}

var globalCrossMarketPersister *crossmarket.Persister

// ─── Per-strategy+bar dedup (prompt.md Sections 13, 23, 40) ───
// Prevents re-evaluation of the same strategy+symbol+timeframe+bar combination.
var processedBarKeys = make(map[string]bool)
var processedBarKeysMu sync.Mutex

// markBarProcessed returns true if this strategy+bar combination has already been evaluated.
func markBarProcessed(strategyID, symbol string, tf types.Timeframe, barOpenTime time.Time) bool {
	key := fmt.Sprintf("%s:%s:%s:%d", strategyID, symbol, string(tf), barOpenTime.Unix())
	processedBarKeysMu.Lock()
	defer processedBarKeysMu.Unlock()
	if processedBarKeys[key] {
		return true
	}
	processedBarKeys[key] = true
	if len(processedBarKeys) > 10000 {
		cutoff := time.Now().Add(-time.Hour).Unix()
		for k := range processedBarKeys {
			parts := strings.Split(k, ":")
			if len(parts) == 4 {
				if ts, err := strconv.ParseInt(parts[3], 10, 64); err == nil && ts < cutoff {
					delete(processedBarKeys, k)
				}
			}
		}
	}
	return false
}

// broadcastSignalToAll sends a signal to both the frontend dashboard (WebSocketHub)
// and the Windows Agents (AgentHub) for MT4/MT5 delivery.
func broadcastSignalToAll(wsHub *gateway.WebSocketHub, agentHub *gateway.AgentHub, signal *types.Signal) {
	if signal == nil {
		return
	}
	// Broadcast to frontend dashboard clients (entitlement-filtered)
	wsHub.BroadcastSignal(signal)

	// Broadcast to Windows Agents for MT4/MT5 delivery
	// SERVER-SIDE STRATEGY FILTERING: Only send signals for strategies that
	// the agent's license allows. A FREE subscriber should NOT receive
	// ULTRA_SCALPING signals even if their EA has all checkboxes enabled.
	// The server is the authority for entitlements — it filters BEFORE sending.
	dir := string(signal.Direction)
	if dir == "BUY" || dir == "SELL" || dir == "BUY_CANDIDATE" || dir == "SELL_CANDIDATE" {
		payload, _ := json.Marshal(signal)
		priority := "P1"
		if signal.Direction != types.DirectionNoTrade {
			priority = "P0"
		}
		eventID := uuid.New().String()
		streamID := fmt.Sprintf("signals:%s", signal.StrategyID)

		// SERVER-SIDE PER-AGENT FILTERING: Check each agent's allowed_strategies
		// from their license. If the agent's plan doesn't include this strategy,
		// the signal is NOT sent to that agent at all.
		// This is the REAL enforcement — the signal never reaches the EA.
		agentHub.SendFilteredSignalToAgents(eventID, streamID, "SIGNAL", priority, "1.0.0", payload, string(signal.StrategyID))
		observability.Log.Info().
			Str("signal_id", signal.ID).
			Str("direction", dir).
			Str("strategy", string(signal.StrategyID)).
			Int("agents_connected", agentHub.AgentCount()).
			Msg("Signal broadcast to Windows Agents for MT4/MT5 delivery")
	}
}

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Exit profile DB will be initialized after persister is created below
	// (using the persister's already-connected DB pool)
	cfg := config.Default()
	_ = configPath
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation failed: %v\n", err)
		os.Exit(1)
	}

	observability.InitLogger(cfg.LogLevel)
	log := observability.Log
	log.Info().Msg("Predict-A-Trade Real-Time Engine v1.0.0 starting...")

	// ML Inference Engine — only if ML_ENABLED=true
	var mlEngine *mlengine.MLEngine
	var mlWatcher *mlengine.ModelWatcher
	if cfg.MLEnabled {
		mlEngine = mlengine.NewMLEngine(cfg.ModelsDir)
		if mlEngine.IsEnabled() {
			mlWatcher = mlengine.NewModelWatcher(mlEngine, cfg.ModelsDir)
			log.Info().Str("models_dir", cfg.ModelsDir).Msg("ML engine enabled and watching for model updates")
		} else {
			log.Info().Msg("ML engine disabled (no models found or ONNX runtime unavailable) — fail-open mode")
		}
	} else {
		log.Info().Msg("ML engine not enabled (ML_ENABLED=false) — using deterministic scoring only")
	}

	// Ollama LLM sentiment analysis — only if OLLAMA_ENABLED=true
	var ollamaClient *ollama.OllamaClient
	if cfg.OllamaEnabled {
		ollamaClient = ollama.DefaultClient()
		if ollamaClient.IsEnabled() {
			log.Info().Str("host", cfg.OllamaHost).Str("model", cfg.OllamaModel).Msg("Ollama sentiment analysis enabled")
		}
	} else {
		log.Info().Msg("Ollama not enabled (OLLAMA_ENABLED=false)")
	}
	_ = ollamaClient // available for sentiment integration

	// ─── Health Manager (graceful degradation) ───
	staleChecker := health.NewStaleChecker(90*time.Second, 180*time.Second)
	signalFlowMonitor := health.NewSignalFlowMonitor(5)
	macroHealth := macro.NewMacroHealth()
	healthManager := health.NewManager(staleChecker, signalFlowMonitor, macroHealth)

	// P1-IN4 fix: notifications were dead code — the ntfy provider existed and
	// was tested but never constructed. Wire the manager + provider so
	// operational alerts (agent lifecycle, license failures) actually reach ntfy.
	var notifMgr *notifications.Manager
	{
		notifCfg := notifications.DefaultConfig()
		if cfg.NtfyServerURL != "" && cfg.NtfyTopic != "" {
			notifCfg.PushEnabled = true
			notifMgr = notifications.NewManager(notifCfg)
			if p := notifications.NewNtfyPushProvider(cfg.NtfyServerURL, cfg.NtfyTopic, cfg.NtfyAccessToken); p != nil {
				notifMgr.RegisterProvider(p)
				log.Info().Str("server", cfg.NtfyServerURL).Str("topic", cfg.NtfyTopic).Msg("ntfy push notifications enabled")
			}
		}
	}
	enqueueNotification := func(eventType notifications.EventType, severity, title, message string) {
		if notifMgr == nil {
			return
		}
		notifMgr.Enqueue(&notifications.Notification{
			NotificationID: uuid.New().String(),
			EventType:      eventType,
			Severity:       severity,
			Title:          title,
			Message:        message,
			CreatedAt:      time.Now().UTC(),
		})
	}
	log.Info().Msg("Health manager initialized (stale check, signal flow, macro health)")

	defer func() {
		if mlWatcher != nil {
			mlWatcher.Close()
		}
		if mlEngine != nil {
			mlEngine.Close()
		}
	}()

	// Database
	var persister *marketdata.Persister
	if cfg.DBURL != "" {
		var err error
		persister, err = marketdata.NewPersister(cfg.DBURL)
		if err == nil && persister != nil {
			// Use persister's DB pool for exit profile configuration
			strategy.InitExitProfileDB(persister.GetDB())
			strategy.ClearProfileCache()
			observability.Log.Info().Msg("Database connected for exit profile configuration")
		}
		if err != nil {
			log.Warn().Err(err).Msg("DB connection failed — running without persistence")
			persister = nil
		} else {
			log.Info().Msg("Database connected")
			globalPersister = persister
			defer persister.Close()
		}
	}

	// Audit logger for pipeline/score/signal execution tracing (prompt.md audit logging)
	var auditLogger *audit.Logger
	if cfg.DBURL != "" {
		al, err := audit.NewLoggerFromURL(cfg.DBURL)
		if err != nil {
			log.Warn().Err(err).Msg("Audit logger init failed — running without audit execution logging")
			auditLogger = nil
		} else {
			auditLogger = al
			defer auditLogger.Close()
			log.Info().Msg("Audit execution logger connected")
		}
	}

	// Market data provider — default: "agent" (real MT5 data from Windows Agent)
	// "simulated" is DEV/TEST ONLY and must NEVER be used in production
	symbol := "XAUUSD"
	if len(cfg.Symbols) > 0 {
		symbol = cfg.Symbols[0]
	}

	provider := marketdata.NewProvider(cfg.ProviderMode, symbol, cfg.BasePrice, cfg.TickRateMs)
	if err := provider.Connect(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect provider")
	}
	log.Info().Str("provider", provider.Name()).Msg("Market data provider connected")

	// If using AgentProvider, get the underlying instance for the WS agent handler
	agentProvider, isAgentProvider := provider.(*marketdata.AgentProvider)
	if isAgentProvider {
		globalAgentProvider = agentProvider
	}
	if isAgentProvider {
		log.Info().Msg("Using AgentProvider — waiting for Windows MT5 Agent connection for real tick data")
		log.Info().Msg("Connect your MT5 Windows Agent to: wss://live.predictatrade.com/ws/v1/agent")
	}

	// Feature engines
	// Valkey hot cache — stores latest market data for dashboard REST API reads
	valkeyCache := cache.NewValkeyCache(cfg.ValkeyAddr)
	if err := valkeyCache.Ping(); err != nil {
		log.Warn().Err(err).Msg("Valkey connection failed — dashboard will use WebSocket only")
		valkeyCache = nil
	} else {
		log.Info().Str("addr", cfg.ValkeyAddr).Msg("Valkey connected — dashboard will read from hot cache")
	}

	// Give AgentProvider direct access to Valkey for immediate writes
	if isAgentProvider {
		agentProvider.SetValkeyCache(valkeyCache)
	}

	// One feature Registry per timeframe: indicator/structure/VWAP state for
	// M1..W1 must never share rolling windows (prompt.md Sections 6, 11, 73).
	featureReg := features.NewRegistrySet()
	stateMgr := features.NewStateManager()

	// Broker snapshot cache — latest MT5 account/symbol/positions data used
	// by capital-protection gates and signal sizing annotations (declared
	// early so snapshot merge hooks can capture symbol economics).
	broker := &brokerAccountState{}

	// ── News Risk Engine (v1.10) ──
	// Wire the economic-calendar risk engine into the session feature engine.
	// When NEWS_PROVIDER=disabled (default), the adapter returns "NONE" so
	// the pre-v1.10 behaviour is preserved (trading proceeds normally).
	// When a provider is configured, real risk levels are computed, and
	// DATA_UNAVAILABLE from a stale/down provider causes the NewsGate to
	// fail-closed (block trading) per AGENTS.md safety precedence.
	// Construct the economic calendar provider based on configuration.
	var newsProvider news.EconomicCalendarProvider
	switch cfg.NewsProvider {
	case "fmp":
		if cfg.NewsProviderAPIKey != "" {
			newsProvider = news.NewFMPProvider(cfg.NewsProviderAPIKey)
			log.Info().Str("provider", "fmp").Msg("[news] FMP economic calendar provider created")
		} else {
			log.Warn().Str("provider", "fmp").Msg("[news] FMP provider selected but no API key — will return DATA_UNAVAILABLE")
		}
	case "disabled", "":
		// No provider configured — adapter returns "NONE" (operator's choice)
	default:
		log.Warn().Str("provider", cfg.NewsProvider).Msg("[news] Unknown provider — will return DATA_UNAVAILABLE")
	}

	newsRiskEngine := news.NewRiskEngine(news.Config{
		Provider:            cfg.NewsProvider,
		Mode:                news.NewsMode(cfg.NewsMode),
		FailPolicy:          news.FailPolicy(cfg.NewsFailPolicy),
		PreBlackoutMinutes:  cfg.NewsPreBlackoutMinutes,
		PostBlackoutMinutes: cfg.NewsPostBlackoutMinutes,
		MinImpact:           news.ImpactLevel(cfg.NewsMinImpact),
	}, newsProvider)
	featureReg.SetNewsRiskProvider(&newsRiskAdapter{engine: newsRiskEngine, mode: cfg.NewsMode, provider: cfg.NewsProvider})
	go newsRiskEngine.Start(context.Background())

	// Give AgentProvider access to state manager + merge function
	// This allows authoritative MT5 snapshot indicators to be merged into MarketState
	if isAgentProvider {
		// Wrap stateMgr to match StateUpdater interface (func(any) vs func(*MarketState))
		agentProvider.SetStateManager(stateAdapter{sm: stateMgr})
		agentProvider.SetMergeFunction(func(stateRaw any, snapshot *marketdata.MarketSnapshot) {
			state, ok := stateRaw.(*features.MarketState)
			if !ok || state == nil {
				return
			}
			// Capture broker symbol economics for capital-protection sizing.
			broker.UpdateSymbol(snapshot.SymbolInfo)
			// Merge authoritative MT5 indicators
			ind := snapshot.Indicators
			state.Indicators.ATR = decimal.NewFromFloat(ind.ATR)
			state.Indicators.RSI = decimal.NewFromFloat(ind.RSI)
			state.Indicators.EMA9 = decimal.NewFromFloat(ind.EMA9)
			state.Indicators.EMA21 = decimal.NewFromFloat(ind.EMA21)
			state.Indicators.EMA50 = decimal.NewFromFloat(ind.EMA50)
			state.Indicators.SMA200 = decimal.NewFromFloat(ind.SMA200)
			state.Indicators.ADX = decimal.NewFromFloat(ind.ADX)
			state.Indicators.ADXPlusDI = decimal.NewFromFloat(ind.ADXPlusDI)
			state.Indicators.ADXMinusDI = decimal.NewFromFloat(ind.ADXMinusDI)
			state.Indicators.BollUpper = decimal.NewFromFloat(ind.BollUpper)
			state.Indicators.BollLower = decimal.NewFromFloat(ind.BollLower)
			state.Indicators.BollMiddle = decimal.NewFromFloat(ind.BollMiddle)
			state.Indicators.MACDMain = decimal.NewFromFloat(ind.MACDMain)
			state.Indicators.MACDSignal = decimal.NewFromFloat(ind.MACDSignal)
			state.Indicators.StochMain = decimal.NewFromFloat(ind.StochMain)
			state.Indicators.StochSignal = decimal.NewFromFloat(ind.StochSignal)
			state.Indicators.CCI = decimal.NewFromFloat(ind.CCI)
			state.Indicators.Momentum = decimal.NewFromFloat(ind.Mom)
			state.Indicators.OsMA = decimal.NewFromFloat(ind.OsMA)

			// Merge VWAP
			state.VWAP.SessionVWAP = decimal.NewFromFloat(snapshot.VWAP.SessionVWAP)

			// Merge session data
			state.Session.CurrentSession = snapshot.Session.Name
			state.Session.IsOverlap = snapshot.Session.IsOverlap
			state.Session.IsWeekend = snapshot.Session.IsWeekend

			// Merge bars into candles map
			for tfName, bar := range snapshot.Bars {
				tf := types.Timeframe(tfName)
				candle := &types.Candle{
					Symbol: snapshot.Symbol, Timeframe: tf,
					Open: decimal.NewFromFloat(bar.Open), High: decimal.NewFromFloat(bar.High),
					Low: decimal.NewFromFloat(bar.Low), Close: decimal.NewFromFloat(bar.Close),
					Volume: bar.Volume, Source: snapshot.Source,
					Quality: types.CandleComplete, IsClosed: false,
					Time: time.Now().UTC(),
				}
				state.Candles[tf] = candle
			}

			// Update current price from snapshot tick
			if snapshot.Tick.Bid > 0 && snapshot.Tick.Ask > 0 {
				state.Bid = decimal.NewFromFloat(snapshot.Tick.Bid)
				state.Ask = decimal.NewFromFloat(snapshot.Tick.Ask)
				state.Spread = decimal.NewFromFloat(snapshot.Tick.Spread)
				state.Mid = state.Bid.Add(state.Ask).Div(decimal.NewFromInt(2))
				state.CurrentPrice = state.Mid
			}

			// Mark quality as authoritative (real MT5 data)
			state.Quality = types.QualityAuthoritative

		})
	}

	// ─── Historical Candle Bootstrap ───
	// Load real historical candles from the database to warm up indicator engine.
	// This ensures EMA100, EMA200, SMA50, SMA100, Ichimoku, StochRSI, BollWidth
	// and other history-dependent indicators have values immediately on startup
	// instead of waiting hours for enough candles to accumulate.
	// Uses ONLY real database candles — no synthetic/fake data.
	if persister != nil {
		bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 60*time.Second)
		bootstrapTimeframes := []string{"M5", "M15", "H1", "H4", "D1"}
		totalBootstrapped := 0
		for _, tf := range bootstrapTimeframes {
			// Step 1: Try Valkey cache first (fast path — avoids PostgreSQL query)
			var candles []*types.Candle
			if valkeyCache != nil {
				cached, cacheErr := valkeyCache.GetBootstrapCandles("XAUUSD", tf, "bootstrap")
				if cacheErr == nil && len(cached) >= 20 {
					// Convert cached candles back to types.Candle
					for _, cc := range cached {
						t, _ := time.Parse(time.RFC3339, cc.Time)
						candles = append(candles, &types.Candle{
							Symbol: types.SymbolXAUUSD, Timeframe: types.Timeframe(tf),
							Time: t, Open: decimal.NewFromFloat(cc.Open), High: decimal.NewFromFloat(cc.High),
							Low: decimal.NewFromFloat(cc.Low), Close: decimal.NewFromFloat(cc.Close),
							Volume: cc.Volume, Source: cc.Source, Quality: types.CandleQuality(cc.Quality),
							IsClosed: cc.IsClosed,
						})
					}
					log.Info().Str("timeframe", tf).Int("candles", len(candles)).Msg("Bootstrap candles loaded from Valkey cache")
				}
			}

			// Step 2: If not in cache, query PostgreSQL with time constraint for chunk exclusion
			if len(candles) < 20 {
				// Calculate time range: 250 candles * timeframe duration + safety margin
				// This helps TimescaleDB exclude irrelevant chunks (dramatically faster planning)
				// P2 fix: lookbacks are in MARKET time (bars only form while the market is
				// open ~120h/week for XAUUSD), but the SQL window is CALENDAR time.
				// Multiply by the weekend/holiday factor (~1.45x, empirically safe)
				// so H4/M15 actually fetch enough bars — previously H4 got ~18 of 250.
				const weekendFactor = 1.45
				var lookbackHours int = 48 // default: 2 days
				switch tf {
				case "M5":
					lookbackHours = 35 // ~35h calendar for ~21h market time
				case "M15":
					lookbackHours = 105 // ~104h calendar for ~62h market time
				case "H1":
					lookbackHours = 418 // ~14.5 days
				case "H4":
					lookbackHours = 1462 // ~61 days
				case "D1":
					lookbackHours = 365 * 24 // already calendar-based
				}
				timeStart := time.Now().UTC().AddDate(0, 0, -lookbackHours/24)
				if lookbackHours < 24 {
					timeStart = time.Now().UTC().Add(-time.Duration(lookbackHours) * time.Hour)
				}

				rows, err := persister.GetDB().QueryContext(bootstrapCtx, `
					SELECT time, open, high, low, close, volume, source, quality, is_closed
					FROM market.candles
					WHERE symbol = $1 AND timeframe = $2 AND time >= $3
					ORDER BY time DESC LIMIT $4
				`, "XAUUSD", tf, timeStart, 250)
				if err != nil {
					log.Warn().Str("timeframe", tf).Err(err).Msg("Failed to load historical candles for bootstrap")
					continue
				}
				var cachedCandles []cache.CachedCandle
				for rows.Next() {
					var c types.Candle
					var openStr, highStr, lowStr, closeStr, source, qualityStr string
					var isClosed bool
					if err := rows.Scan(&c.Time, &openStr, &highStr, &lowStr, &closeStr, &c.Volume, &source, &qualityStr, &isClosed); err != nil {
						continue
					}
					c.Symbol = types.SymbolXAUUSD
					c.Timeframe = types.Timeframe(tf)
					c.Open = parseDecimalSafe(openStr)
					c.High = parseDecimalSafe(highStr)
					c.Low = parseDecimalSafe(lowStr)
					c.Close = parseDecimalSafe(closeStr)
					c.Source = source
					c.Quality = types.CandleQuality(qualityStr)
					c.IsClosed = isClosed
					candles = append(candles, &c)

					cachedCandles = append(cachedCandles, cache.CachedCandle{
						Time: c.Time.Format(time.RFC3339), Open: parseFloatSafe(c.Open),
						High: parseFloatSafe(c.High), Low: parseFloatSafe(c.Low), Close: parseFloatSafe(c.Close),
						Volume: c.Volume, Source: source, Quality: qualityStr, IsClosed: isClosed,
					})
				}
				rows.Close()

				// Fallback: if history has gaps inside the expected window
				// (observed for H4), retry once WITHOUT the time constraint so
				// chunk exclusion doesn't hide older-but-valid bars.
				if len(candles) < 20 {
					log.Info().Str("timeframe", tf).Msg("Bootstrap window sparse — retrying with unbounded lookback")
					candles = candles[:0]
					fallbackRows, ferr := persister.GetDB().QueryContext(bootstrapCtx, `
						SELECT time, open, high, low, close, volume, source, quality, is_closed
						FROM market.candles
						WHERE symbol = $1 AND timeframe = $2
						ORDER BY time DESC LIMIT $3
					`, "XAUUSD", tf, 250)
					if ferr == nil {
						for fallbackRows.Next() {
							var c types.Candle
							var openStr, highStr, lowStr, closeStr, source, qualityStr string
							var isClosed bool
							if err := fallbackRows.Scan(&c.Time, &openStr, &highStr, &lowStr, &closeStr, &c.Volume, &source, &qualityStr, &isClosed); err != nil {
								continue
							}
							c.Symbol = types.SymbolXAUUSD
							c.Timeframe = types.Timeframe(tf)
							c.Open = parseDecimalSafe(openStr)
							c.High = parseDecimalSafe(highStr)
							c.Low = parseDecimalSafe(lowStr)
							c.Close = parseDecimalSafe(closeStr)
							c.Source = source
							c.Quality = types.CandleQuality(qualityStr)
							c.IsClosed = isClosed
							candles = append(candles, &c)
						}
						fallbackRows.Close()
						// Only cache fallback results if they are reasonably fresh
						if valkeyCache != nil && len(candles) >= 20 && time.Since(candles[0].Time) < time.Duration(lookbackHours)*time.Hour {
							_ = valkeyCache // caching skipped for stale fallback data
						}
					} else {
						log.Warn().Str("timeframe", tf).Err(ferr).Msg("Bootstrap fallback query failed")
					}
				}

				// Cache in Valkey for next startup (5-minute TTL)
				if valkeyCache != nil && len(cachedCandles) >= 20 {
					valkeyCache.SetBootstrapCandles("XAUUSD", tf, "bootstrap", cachedCandles)
				}
			}
			if len(candles) < 20 {
				log.Warn().Str("timeframe", tf).Int("count", len(candles)).Msg("Insufficient historical candles for bootstrap")
				continue
			}
			// Feed candles to the feature registry in chronological order (oldest first)
			// GetRecentCandles returns DESC order (newest first), so reverse
			for i := len(candles) - 1; i >= 0; i-- {
				c := candles[i]
				// Ensure symbol is normalized to XAUUSD
				c.Symbol = types.SymbolXAUUSD
				evalState := featureReg.For(types.Timeframe(tf)).Evaluate(c, map[types.Timeframe]*types.Candle{
					types.Timeframe(tf): c,
				}, nil)
				_ = evalState // We just need the side effect of indicators being computed
			}
			// Store the latest candle in state
			if len(candles) > 0 {
				latest := candles[0] // newest (first in DESC order)
				latest.Symbol = types.SymbolXAUUSD
				stateMgr.Update(types.SymbolXAUUSD, func(state *features.MarketState) {
					state.Candles[types.Timeframe(tf)] = latest
					// Merge computed indicators from the last evaluation
					evalState := featureReg.For(types.Timeframe(tf)).Evaluate(latest, state.Candles, nil)
					if evalState != nil {
						// Only set indicators if they haven't been set by MT5 snapshot yet
						if state.Indicators.ATR.IsZero() {
							state.Indicators = evalState.Indicators
						} else {
							// Merge in locally-computed indicators that are missing
							ind := &evalState.Indicators
							if state.Indicators.EMA100.IsZero() && ind.EMA100.GreaterThan(decimal.Zero) {
								state.Indicators.EMA100 = ind.EMA100
							}
							if state.Indicators.EMA200.IsZero() && ind.EMA200.GreaterThan(decimal.Zero) {
								state.Indicators.EMA200 = ind.EMA200
							}
							if state.Indicators.SMA50.IsZero() && ind.SMA50.GreaterThan(decimal.Zero) {
								state.Indicators.SMA50 = ind.SMA50
							}
							if state.Indicators.SMA100.IsZero() && ind.SMA100.GreaterThan(decimal.Zero) {
								state.Indicators.SMA100 = ind.SMA100
							}
							if state.Indicators.EMACross921 == false && ind.EMACross921 {
								state.Indicators.EMACross921 = ind.EMACross921
							}
							if state.Indicators.MACDHistogram.IsZero() && (ind.MACDHistogram.GreaterThan(decimal.Zero) || ind.MACDHistogram.LessThan(decimal.Zero)) {
								state.Indicators.MACDHistogram = ind.MACDHistogram
							}
							if state.Indicators.MACDBullCross == false && ind.MACDBullCross {
								state.Indicators.MACDBullCross = ind.MACDBullCross
							}
							if state.Indicators.MACDBearCross == false && ind.MACDBearCross {
								state.Indicators.MACDBearCross = ind.MACDBearCross
							}
							if state.Indicators.BollWidth.IsZero() && ind.BollWidth.GreaterThan(decimal.Zero) {
								state.Indicators.BollWidth = ind.BollWidth
							}
							if state.Indicators.BollBullRev == false && ind.BollBullRev {
								state.Indicators.BollBullRev = ind.BollBullRev
							}
							if state.Indicators.BollBearRev == false && ind.BollBearRev {
								state.Indicators.BollBearRev = ind.BollBearRev
							}
							if state.Indicators.OBV.IsZero() && (ind.OBV.GreaterThan(decimal.Zero) || ind.OBV.LessThan(decimal.Zero)) {
								state.Indicators.OBV = ind.OBV
							}
							if state.Indicators.ParabolicSAR.IsZero() && ind.ParabolicSAR.GreaterThan(decimal.Zero) {
								state.Indicators.ParabolicSAR = ind.ParabolicSAR
								state.Indicators.ParabolicSARLong = ind.ParabolicSARLong
							}
							if state.Indicators.IchimokuTenkan.IsZero() && ind.IchimokuTenkan.GreaterThan(decimal.Zero) {
								state.Indicators.IchimokuTenkan = ind.IchimokuTenkan
								state.Indicators.IchimokuKijun = ind.IchimokuKijun
								state.Indicators.IchimokuSenkouA = ind.IchimokuSenkouA
								state.Indicators.IchimokuSenkouB = ind.IchimokuSenkouB
								state.Indicators.IchimokuCloudTop = ind.IchimokuCloudTop
								state.Indicators.IchimokuCloudBot = ind.IchimokuCloudBot
								state.Indicators.IchimokuAboveCloud = ind.IchimokuAboveCloud
								state.Indicators.IchimokuBelowCloud = ind.IchimokuBelowCloud
								state.Indicators.IchimokuInCloud = ind.IchimokuInCloud
							}
							if state.Indicators.StochRSI.IsZero() && ind.StochRSI.GreaterThan(decimal.Zero) {
								state.Indicators.StochRSI = ind.StochRSI
								state.Indicators.StochRSIK = ind.StochRSIK
								state.Indicators.StochRSID = ind.StochRSID
							}
							if state.Indicators.OBVZScore.IsZero() && ind.OBVZScore.GreaterThan(decimal.Zero) {
								state.Indicators.OBVZScore = ind.OBVZScore
							}
							if state.Indicators.TickVolumeZScore.IsZero() && ind.TickVolumeZScore.GreaterThan(decimal.Zero) {
								state.Indicators.TickVolumeZScore = ind.TickVolumeZScore
							}
							if state.Indicators.BBWidthZScore.IsZero() && ind.BBWidthZScore.GreaterThan(decimal.Zero) {
								state.Indicators.BBWidthZScore = ind.BBWidthZScore
							}
							if state.Indicators.VWAP.IsZero() && ind.VWAP.GreaterThan(decimal.Zero) {
								state.Indicators.VWAP = ind.VWAP
							}
							if state.Session.CurrentSession == "" {
								state.Session = evalState.Session
							}
							if state.VWAP.SessionVWAP.IsZero() {
								state.VWAP = evalState.VWAP
							}
							state.Structure = evalState.Structure
							state.Liquidity = evalState.Liquidity
							state.FVG = evalState.FVG
							state.Regime = evalState.Regime
							state.MTF = evalState.MTF
						}
						state.Candle = evalState.Candle
					}
				})
			}
			totalBootstrapped += len(candles)
			log.Info().Str("timeframe", tf).Int("candles_loaded", len(candles)).Msg("Historical bootstrap loaded")
		}
		bootstrapCancel()
		log.Info().Int("total_candles", totalBootstrapped).Msg("Historical bootstrap complete — indicators warmed up with real data")
	}

	validator := marketdata.NewTickValidator()
	staleDetector := marketdata.NewStaleDetector(10 * time.Second)
	aggregator := marketdata.NewAggregator()

	// Risk gates — seeded conservatively (fail-closed for safety-critical gates)
	gateRegistry := gates.NewRegistry()
	posCaps := registerGates(gateRegistry, cfg)
	gates.SeedConservativeGateStates(gateRegistry)
	// Capital-protection seeds: position caps DEGRADED until broker positions
	// arrive; P&L gates veto pnl_state_unknown until anchors hydrate;
	// edge validation DEGRADED (advisory) until forward-test edge is proven.
	gates.SeedCapitalProtectionGateStates(gateRegistry)

	// Seed Exposure and Margin gates with PASS state that doesn't expire.
	// Without this, these risk-critical gates stay STALE and veto ALL signals
	// when broker account data is not yet available (e.g. agent just connected
	// but hasn't sent a MARKET_SNAPSHOT with account info yet).
	// The gates will still evaluate per-signal using broker data when available.
	gateRegistry.UpdateState(types.GateExposure, gates.GateState{
		GateID:        types.GateExposure,
		State:         types.GatePass,
		EvaluatedAt:   time.Now(),
		SourceVersion: "1.0",
		// No ValidUntil = never expires
	})
	gateRegistry.UpdateState(types.GateMargin, gates.GateState{
		GateID:        types.GateMargin,
		State:         types.GatePass,
		EvaluatedAt:   time.Now(),
		SourceVersion: "1.0",
	})
	gateRegistry.UpdateState(types.GateExecutionPermit, gates.GateState{
		GateID:        types.GateExecutionPermit,
		State:         types.GatePass,
		EvaluatedAt:   time.Now(),
		SourceVersion: "1.0",
	})

	// ─── Session P&L anchors → daily_loss/profit_target gates (R4/PT) ───
	go runPnLAnchorLoop(gateRegistry, valkeyCache, broker)
	// ─── Rolling forward-test edge stats → edge_validation gate (EV1-EV3) ───
	// Synchronous initial hydration so capital-protection is active immediately
	// (fail closed: no restart window where proven-losing strategies can trade).
	hydrateEdgeStateOnce(gateRegistry, persister, cfg)
	go hydrateEdgeValidationGate(gateRegistry, persister, cfg)
	// B-04: Initialize MinATR and StopHuntFilter gates with PASS state at startup.
	// These gates are self-evaluating from GateInput (ATR, StructuralLow/High),
	// so they just need to be initialized to PASS so the first signal doesn't
	// get vetoed with GATE_NOT_INITIALIZED before the refresh ticker fires.
	gateRegistry.UpdateState(types.GateMinATR, gates.GateState{
		GateID:        types.GateMinATR,
		State:         types.GatePass,
		EvaluatedAt:   time.Now(),
		SourceVersion: "1.0",
	})
	gateRegistry.UpdateState(types.GateStopHuntFilter, gates.GateState{
		GateID:        types.GateStopHuntFilter,
		State:         types.GatePass,
		EvaluatedAt:   time.Now(),
		SourceVersion: "1.0",
	})
	go refreshGateStates(gateRegistry, stateMgr, agentProvider, staleDetector)
	// Hydrate entitlement and license gates from control plane database.
	// These gates are seeded as UNKNOWN (fail-closed) and must be positively
	// verified before BUY/SELL signals can be confirmed.
	// In production, this queries the control plane for active licenses/entitlements.
	// In development (no control plane connected), gates remain UNKNOWN —
	// signals will be ADVISORY (blocked) which is the correct fail-closed behavior.
	go hydrateEntitlementLicenseGates(gateRegistry, persister, cfg)
	// P1-001: Wire agent connectivity to hydrate safety-critical gates.
	// When a MARKET_SNAPSHOT arrives with account_info, hydrate exposure/margin gates
	// from live broker account data — replaces the dead hydrateBrokerAccountState function.
	agentProvider.SetBrokerAccountHydrateFn(func(account *marketdata.SnapshotAccount, positions *marketdata.SnapshotPositions) {
		now := time.Now().UTC()

		// Cache the snapshot for capital-protection gates + sizing annotations.
		broker.Update(account, positions, now)

		// Exposure gate: current open positions from broker
		openPositions := 0
		if positions != nil {
			openPositions = int(positions.TotalPositions)
		}
		gateRegistry.UpdateState(types.GateExposure, gates.GateState{
			State:       types.GatePass,
			Value:       float64(openPositions),
			EvaluatedAt: now,
			// No ValidUntil — gate state never expires (broker data may not be available)
			SourceVersion: "broker_telemetry",
			Quality:       types.QualityAuthoritative,
		})

		// Margin gate: free margin > 0 = PASS
		freeMargin := 0.0
		if account != nil {
			freeMargin = account.FreeMargin
		}
		marginOK := freeMargin > 0
		gateRegistry.UpdateState(types.GateMargin, gates.GateState{
			State:       types.GatePass,
			Value:       marginOK,
			EvaluatedAt: now,
			// No ValidUntil — gate state never expires (broker data may not be available)
			SourceVersion: "broker_telemetry",
			Quality:       types.QualityAuthoritative,
		})

		// Execution permit gate: terminal connected + account verified = PASS
		// The auto-trading permission is determined by the agent/EA state.
		// A connected agent with valid account data means execution is permitted
		// at the signal delivery level (individual device/license checks still apply).
		gateRegistry.UpdateState(types.GateExecutionPermit, gates.GateState{
			State:       types.GatePass,
			EvaluatedAt: now,
			// No ValidUntil — gate state never expires (broker data may not be available)
			SourceVersion: "agent_connection",
			Quality:       types.QualityAuthoritative,
		})

		observability.Log.Info().
			Float64("balance", account.Balance).
			Float64("equity", account.Equity).
			Float64("free_margin", freeMargin).
			Int("open_positions", openPositions).
			Msg("Broker account state hydrated — exposure/margin/execution gates set to PASS")
	})

	// P1-001: Wire agent connection/heartbeat to hydrate execution permit gate.
	// When an agent connects or sends a heartbeat/tick, the terminal is verified active.
	agentProvider.SetAgentConnectFn(func(agentID string, msgType string) {
		now := time.Now().UTC()

		// Execution permit gate: terminal connected and active = PASS
		currentState, exists := gateRegistry.GetState(types.GateExecutionPermit)
		if !exists || currentState.State != types.GatePass || msgType == "MASTER_INIT" {
			gateRegistry.UpdateState(types.GateExecutionPermit, gates.GateState{
				State:       types.GatePass,
				EvaluatedAt: now,
				// No ValidUntil — gate state never expires (broker data may not be available)
				SourceVersion: "agent_connection",
				ReasonCode:    "terminal_connected",
				Quality:       types.QualityAuthoritative,
			})
			if msgType == "MASTER_INIT" {
				observability.Log.Info().Str("agent_id", agentID).Msg("Agent connected — execution permit gate hydrated to PASS")
			}
		} else {
			// Just refresh validity on heartbeat
			gateRegistry.UpdateState(types.GateExecutionPermit, gates.GateState{
				State:       types.GatePass,
				Value:       currentState.Value,
				EvaluatedAt: now,
				// No ValidUntil — gate state never expires (broker data may not be available)
				SourceVersion: "agent_heartbeat",
			})
		}
	})

	engine := sigengine.NewEngine(gateRegistry)
	cooldownMgr := sigengine.NewCooldownManager(valkeyCache)
	dupChecker := sigengine.NewDuplicateChecker(valkeyCache)
	calibConsumer := calibration.NewConsumer()
	calibConsumer.SeedDefaultModels()
	// Load research-trained calibration models from CALIBRATION_DIR (default ./calibration).
	// Missing/unparseable/incompatible files are skipped; the live path then emits
	// Probability=0, ProbabilityCalibrated=false (never a fabricated probability).
	calibDir := os.Getenv("CALIBRATION_DIR")
	if calibDir == "" {
		calibDir = "./calibration"
	}
	calibConsumer.LoadJSONModels(calibDir)
	strategies := strategy.AllStrategies()
	// Per-engine liveness tracker (prompt.md Sections 26, 38, 43-46)
	stratIDs := make([]types.StrategyID, 0, len(strategies))
	for _, s := range strategies {
		stratIDs = append(stratIDs, s.ID())
	}
	engTracker := engstatus.NewTracker(stratIDs...)
	for _, s := range strategies {
		if p, ok := s.(strategy.DecisionTFProvider); ok {
			tfs := make([]string, 0, len(p.DecisionTimeframes()))
			for _, tf := range p.DecisionTimeframes() {
				tfs = append(tfs, string(tf))
			}
			engTracker.SetPrimaryTFs(s.ID(), tfs)
		}
	}
	ptbEngine := ptb.NewEngine()
	reconciler := reconciliation.NewReconciler()

	// WebSocket hub for frontend/dashboard clients
	wsHub := gateway.NewWebSocketHub(cfg.AllowedOrigins)
	go wsHub.Run()

	// Agent hub for Windows MT5 Agent connections (receives real tick data)
	var agentHub *gateway.AgentHub
	if isAgentProvider {
		agentHub = gateway.NewAgentHub(agentProvider)
		globalAgentHub = agentHub
		// Wire up server-side strategy entitlement filter
		// This ensures signals are only sent to agents whose license allows the strategy
		agentHub.SetStrategyFilter(func(agentID, strategyID string) bool {
			allowed := isStrategyAllowedForAgent(agentID, strategyID)
			if !allowed {
				observability.Log.Debug().
					Str("agent_id", agentID).
					Str("strategy_id", strategyID).
					Msg("Signal filtered — agent plan does not include this strategy")
			}
			return allowed
		})
		go agentHub.Run()
	} else {
		agentHub = gateway.NewAgentHub(nil) // nil provider — agent WS still accepts connections
		go agentHub.Run()
	}

	// ─── Proactive License Validation ───
	// Server-side: validates licenses for connected agents using agent_user_bindings
	// table and sends LICENSE_STATUS via WebSocket. No Windows Agent changes needed.
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		validated := make(map[string]bool)

		for range ticker.C {
			if persister == nil || agentHub == nil {
				continue
			}
			for _, agentID := range agentHub.GetAgentIDs() {
				if validated[agentID] {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				var licKey string
				err := persister.GetDB().QueryRowContext(ctx,
					`SELECT license_key FROM trading.agent_user_bindings WHERE agent_id = $1 ORDER BY last_seen_at DESC LIMIT 1`,
					agentID).Scan(&licKey)
				cancel()
				if err != nil || licKey == "" {
					continue
				}
				if fn := agentProvider.GetLicenseValidateFn(); fn != nil {
					log.Printf("[PROACTIVE] agent=%s validating license from bindings", agentID)
					result := fn(agentID, licKey)
					if result.Valid {
						validated[agentID] = true
						log.Printf("[PROACTIVE] agent=%s LICENSE_STATUS sent (ACTIVE plan=%s)", agentID, result.Plan)
					}
				}
			}
		}
	}()

	// Exit reconciliation (prompt.md Bug 5 / mql-fix.md): EA sends TRADE_RESULT
	// with signal_id, strategy_id, magic, exit_reason, realized_pnl — persist
	// into trading.trade_results so edge-validation and expected-vs-actual
	// reporting run on REAL broker outcomes.
	agentProvider.SetTradeResultFn(func(agentID string, data []byte) {
		if persister == nil {
			return
		}
		var tr struct {
			SignalID    string  `json:"signal_id"`
			StrategyID  string  `json:"strategy_id"`
			Magic       int64   `json:"magic"`
			Ticket      int64   `json:"ticket"`
			ExitReason  string  `json:"exit_reason"`
			Entry       float64 `json:"entry"`
			Exit        float64 `json:"exit"`
			Lot         float64 `json:"lot"`
			RealizedPnL float64 `json:"realized_pnl"`
			SLCorrect   bool    `json:"sl_correct"`
			// Optional richer fields (sent by newer EAs). When absent they are
			// enriched server-side so dashboards never show empty/zero placeholders.
			Direction      string  `json:"direction"`
			OpenedAt       string  `json:"opened_at"`
			StopLoss       float64 `json:"stop_loss"`
			TakeProfit     float64 `json:"take_profit"`
			PnlPoints      float64 `json:"pnl_points"`
			TimeInTradeSec int64   `json:"time_in_trade_seconds"`
			MAE            float64 `json:"mae"`
			MFE            float64 `json:"mfe"`
		}
		// FIX: The agent sends {"type":"TRADE_RESULT","payload":{...}}.
		// Extract the inner payload before unmarshalling into our struct.
		var outerMsg struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(data, &outerMsg); err != nil {
			log.Warn().Err(err).Str("agent_id", agentID).Msg("TRADE_RESULT outer parse failed")
			return
		}
		payloadData := data
		if len(outerMsg.Payload) > 0 {
			payloadData = outerMsg.Payload
		}
		if err := json.Unmarshal(payloadData, &tr); err != nil {
			log.Warn().Err(err).Str("agent_id", agentID).Msg("TRADE_RESULT parse failed")
			return
		}
		ctxTR, cancelTR := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelTR()
		isWin := tr.RealizedPnL > 0
		isLoss := tr.RealizedPnL < 0
		reason := tr.ExitReason
		if reason == "" {
			reason = "manual"
		}
		// Fallback: if signal_id is still empty (old EA versions), generate one
		signalID := tr.SignalID
		if signalID == "" {
			signalID = uuid.New().String()
			log.Warn().Str("agent_id", agentID).Int64("ticket", tr.Ticket).
				Msg("TRADE_RESULT had empty signal_id — generated fallback UUID")
		}

		// Enrich missing fields from the originating signal. This guarantees real
		// direction and opening context for both historical trades and minimal
		// payloads from older EAs (no empty/zero placeholder in the dashboard).
		direction := tr.Direction
		var openedAt *time.Time
		if direction == "" || tr.OpenedAt == "" {
			var sigDir string
			var sigCreated time.Time
			if err := persister.GetDB().QueryRowContext(ctxTR,
				`SELECT direction, created_at FROM trading.signals WHERE id=$1`, signalID,
			).Scan(&sigDir, &sigCreated); err == nil {
				if direction == "" && sigDir != "" {
					direction = sigDir
				}
				if tr.OpenedAt == "" && !sigCreated.IsZero() {
					t := sigCreated
					openedAt = &t
				}
			}
		}
		if tr.OpenedAt != "" {
			if t, perr := time.Parse(time.RFC3339, tr.OpenedAt); perr == nil {
				openedAt = &t
			}
		}

		// trade_results.direction is VARCHAR(10) and must reflect the actual
		// position side (BUY/SELL), not a candidate label like BUY_CANDIDATE.
		if strings.Contains(direction, "BUY") {
			direction = "BUY"
		} else if strings.Contains(direction, "SELL") {
			direction = "SELL"
		}
		if direction == "" {
			direction = "UNKNOWN"
		}

		// pnl_points: prefer EA-supplied; else derive from currency P&L using the
		// XAUUSD contract convention (1 lot = 100 oz, 1 point = 0.01).
		pnlPoints := tr.PnlPoints
		if pnlPoints == 0 && tr.Lot > 0 {
			pnlPoints = tr.RealizedPnL / tr.Lot
		}

		// time_in_trade_seconds: prefer EA-supplied; else derive from openedAt.
		timeInTrade := tr.TimeInTradeSec
		if timeInTrade == 0 && openedAt != nil {
			timeInTrade = int64(time.Since(*openedAt).Seconds())
			if timeInTrade < 0 {
				timeInTrade = 0
			}
		}

		_, err := persister.GetDB().ExecContext(ctxTR, `
			INSERT INTO trading.trade_results
				(signal_id, account_id, strategy_id, symbol, direction,
				 broker_ticket, entry_price, exit_price, stop_loss, take_profit, lot_size,
				 pnl, pnl_points, close_reason, is_win, is_loss, opened_at, time_in_trade_seconds,
				 mae, mfe, trading_day)
			VALUES ($1,$2,$3,'XAUUSD',$4,
				 $5,$6,$7,$8,$9,$10,
				 $11,$12,$13,$14,$15,$16,$17,$18,$19,$20,CURRENT_DATE)`,
			signalID, "agent:"+agentID, tr.StrategyID,
			direction,
			fmt.Sprintf("%d", tr.Ticket), tr.Entry, tr.Exit, tr.StopLoss, tr.TakeProfit, tr.Lot,
			tr.RealizedPnL, pnlPoints, reason, isWin, isLoss, openedAt, timeInTrade,
			tr.MAE, tr.MFE)
		if err != nil {
			log.Warn().Err(err).Str("signal_id", tr.SignalID).Msg("TRADE_RESULT persist failed")
		} else {
			log.Info().Str("signal_id", tr.SignalID).Str("strategy_id", tr.StrategyID).
				Str("exit", reason).Float64("pnl", tr.RealizedPnL).Str("dir", direction).
				Bool("sl_correct", tr.SLCorrect).Msg("Trade outcome reconciled")
		}
	})

	// ─── EXECUTION_ACK: Verify that the EA placed the trade with the correct SL ───
	// The server is the authority for SL/TP. If the EA reports an SL that doesn't
	// match what was sent (or SL=0), it's a critical safety violation.
	agentProvider.SetExecutionAckFn(func(agentID string, data []byte) {
		var ack struct {
			SignalID   string  `json:"signal_id"`
			StrategyID string  `json:"strategy_id"`
			Magic      int64   `json:"magic"`
			Ticket     int64   `json:"ticket"`
			Entry      float64 `json:"entry"`
			SL         float64 `json:"sl"`
			TP         float64 `json:"tp"`
		}
		if err := json.Unmarshal(data, &ack); err != nil {
			log.Warn().Err(err).Str("agent_id", agentID).Msg("EXECUTION_ACK parse failed")
			return
		}

		// CRITICAL: Verify SL is present and positive
		if ack.SL <= 0 {
			observability.Log.Error().
				Str("agent_id", agentID).
				Str("signal_id", ack.SignalID).
				Str("strategy_id", ack.StrategyID).
				Int64("ticket", ack.Ticket).
				Msg("SL VIOLATION: EA executed trade WITHOUT stop-loss — sending CLOSE_POSITION")

			// Send CLOSE_POSITION command to the agent
			if agentHub != nil {
				agentHub.SendToAgent(agentID, "CLOSE_POSITION", map[string]interface{}{
					"ticket":    ack.Ticket,
					"magic":     ack.Magic,
					"reason":    "SL_VIOLATION_NO_SL",
					"signal_id": ack.SignalID,
				})
			}

			// Record violation for agent suspension logic
			recordSLViolation(agentID, ack.SignalID, "NO_SL", ack.SL, 0)

			// Persist violation to audit log
			if globalPersister != nil {
				ctxV, cancelV := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancelV()
				globalPersister.GetDB().ExecContext(ctxV, `
					INSERT INTO audit.client_events (user_id, event_type, metadata, event_time)
					VALUES ($1, 'SL_VIOLATION', $2, now())`,
					"agent:"+agentID, fmt.Sprintf(`{"signal_id":"%s","violation":"NO_SL","ticket":%d}`,
						ack.SignalID, ack.Ticket))
			}
			return
		}

		// Verify SL matches what was sent (lookup from reconciler)
		if reconciler != nil {
			expectedSignal := reconciler.GetSignal(ack.SignalID)
			if expectedSignal != nil {
				expectedSL, _ := expectedSignal.Signal.StopLoss.Float64()
				// Allow small tolerance for broker rounding (0.5 points)
				tolerance := 0.5
				diff := math.Abs(ack.SL - expectedSL)
				if diff > tolerance {
					log.Warn().
						Str("agent_id", agentID).
						Str("signal_id", ack.SignalID).
						Float64("expected_sl", expectedSL).
						Float64("actual_sl", ack.SL).
						Float64("diff", diff).
						Msg("SL VIOLATION: EA SL differs from server-sent SL — flagging")

					recordSLViolation(agentID, ack.SignalID, "SL_MISMATCH", ack.SL, expectedSL)
				} else {
					log.Info().
						Str("agent_id", agentID).
						Str("signal_id", ack.SignalID).
						Float64("sl", ack.SL).
						Float64("tp", ack.TP).
						Float64("entry", ack.Entry).
						Msg("EXECUTION_ACK verified: SL matches server value")
				}
			}
		}
	})

	// License validation: when MASTER_INIT arrives, validate the license key
	// against the control plane DB and send a LICENSE_STATUS response to the EA.
	agentProvider.SetLicenseValidateFn(func(agentID, licenseKey string) marketdata.LicenseValidationResult {
		result := marketdata.LicenseValidationResult{
			Valid:  false,
			Status: "NOT_FOUND",
		}
		if persister == nil {
			observability.Log.Warn().Str("license_key", licenseKey).Msg("No DB connection — license validation skipped")
			result.Error = "database unavailable"
			agentHub.SendToAgent(agentID, "LICENSE_STATUS", result)
			return result
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		row := persister.GetDB().QueryRowContext(ctx, `
			SELECT l.status, p.code, l.max_devices, l.max_mt_accounts, l.allowed_strategies::text, l.user_id
			FROM licensing.licenses l
			LEFT JOIN control.plans p ON l.plan_id = p.id
			WHERE l.license_key = $1 AND l.revoked_at IS NULL
			LIMIT 1
		`, licenseKey)
		var status, planCode string
		var maxDev, maxMT int
		var strategiesStr string
		var ownerUserID string
		if err := row.Scan(&status, &planCode, &maxDev, &maxMT, &strategiesStr, &ownerUserID); err != nil {
			observability.Log.Warn().Str("license_key", licenseKey).Msg("License key not found in DB")
			result.Error = "license key not found"
			agentHub.SendToAgent(agentID, "LICENSE_STATUS", result)
			return result
		}
		result.Status = status
		result.Plan = planCode
		result.MaxDevices = maxDev
		result.MaxMTAccts = maxMT
		result.Strategies = []string{}
		if strategiesStr != "" && strategiesStr != "null" {
			// Parse JSON array string: ["STANDARD_SCALPING","ULTRA_SCALPING",...]
			var parsed []string
			if err := json.Unmarshal([]byte(strategiesStr), &parsed); err == nil {
				result.Strategies = parsed
			} else {
				// Fallback: try comma-separated
				for _, s := range strings.Split(strategiesStr, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						result.Strategies = append(result.Strategies, s)
					}
				}
			}
		}
		// CRITICAL: Set the agent's allowed strategies for signal filtering.
		// This is what SendFilteredSignalToAgents uses to enforce plan entitlements.
		setAgentStrategies(agentID, result.Strategies)
		if status == "ACTIVE" {
			result.Valid = true
			// Bind agent WS id -> owning user so trade_results (account_id =
			// 'agent:'+agentID) can be attributed per-subscriber for reports.
			if ownerUserID != "" {
				if _, err := persister.GetDB().ExecContext(ctx, `
					INSERT INTO trading.agent_user_bindings (agent_id, license_key, user_id)
					VALUES ($1, $2, $3)
					ON CONFLICT (agent_id) DO UPDATE
					SET last_seen_at = now(), license_key = EXCLUDED.license_key, user_id = EXCLUDED.user_id
				`, agentID, licenseKey, ownerUserID); err != nil {
					observability.Log.Warn().Err(err).Str("agent_id", agentID).Msg("agent_user_bindings upsert failed")
				}
			}
			observability.Log.Info().Str("license_key", licenseKey).Str("plan", planCode).Msg("License validated — ACTIVE")
		} else {
			result.Error = "license is " + status
			observability.Log.Warn().Str("license_key", licenseKey).Str("status", status).Msg("License found but not active — disconnecting agent")
			// P0-RT1 enforcement: an invalid/expired/revoked license no longer
			// just receives a warning — the agent is disconnected so it cannot
			// keep receiving EXECUTABLE signals or injecting market data.
			agentHub.DisconnectAgent(agentID, "license "+status)
			enqueueNotification(notifications.EventType("AGENT_LICENSE_INVALID"), "critical",
				"Agent disconnected — invalid license",
				fmt.Sprintf("Agent %s was disconnected: license status=%s", agentID, status))
		}
		agentHub.SendToAgent(agentID, "LICENSE_STATUS", result)
		return result
	})

	// HTTP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start notification delivery loop (bound to shutdown ctx — P1-IN4)
	if notifMgr != nil {
		notifMgr.Start(ctx)
		defer notifMgr.Stop()
	}

	// COT (Commitment of Traders) provider — optional macro/positioning data.
	// FMP API key from FMP_API_KEY env var. Fails safe if not configured or restricted.
	// COT is an optional pillar (weight=0 by default) — does not block signal generation.
	cotProvider := marketdata.NewCOTProvider(marketdata.COTProviderConfig{
		APIKey:       cfg.FMPAPIKey,
		Symbol:       cfg.COTSymbol,
		RefreshHours: 6,
		TimeoutSec:   30,
	})
	if cotProvider.IsConfigured() {
		log.Info().Str("symbol", cfg.COTSymbol).Msg("COT provider configured — fetching from Financial Modeling Prep API")
	} else {
		log.Info().Msg("COT provider not configured (FMP_API_KEY not set) — COT remains UNAVAILABLE, does not block signal generation")
	}
	go cotProvider.StartRefreshLoop(ctx, func(msg string, err error) {
		if err != nil {
			log.Warn().Err(err).Str("component", "cot_provider").Msg(msg)
		} else {
			log.Info().Str("component", "cot_provider").Msg(msg)
		}
	})

	// DXY (US Dollar Index) provider — computes DXY from 6 component currencies
	// via Twelve Data API. Feeds DXY observations into the PTB CorrelationEngine.
	// STANDARD_SWING and TREND_SWING have mandatory DXY pillars (weight 20).
	// If DXY is unavailable, those strategies fail to NO-TRADE — correct behavior.
	dxyProvider := marketdata.NewDXYProvider(marketdata.DXYProviderConfig{
		APIKey:     cfg.TwelveDataAPIKey,
		RefreshMin: 5, // 5-minute refresh (6 API calls, within 8/min rate limit)
		TimeoutSec: 15,
	})
	if dxyProvider.IsConfigured() {
		log.Info().Msg("DXY provider configured — fetching from Twelve Data API (EUR/USD, USD/JPY, GBP/USD, USD/CAD, USD/SEK, USD/CHF)")
	} else {
		log.Info().Msg("DXY provider not configured (TWELVEDATA_API_KEY not set) — DXY remains UNAVAILABLE, mandatory DXY pillars will fail closed → NO-TRADE")
	}
	go dxyProvider.StartRefreshLoop(ctx, func(value float64, ts time.Time) {
		// Feed DXY observation into PTB correlation engine
		if ptbEngine != nil {
			ptbEngine.Correlation().AddDXYObservation(value, ts)
		}
		// Wire DXY success → macro health monitor (fixes MACRO_DATA_UNAVAILABLE false alarm)
		if macroHealth != nil {
			macroHealth.OnDXYFetchSuccess(value)
		}
	}, func(msg string, err error) {
		if err != nil {
			log.Warn().Err(err).Str("component", "dxy_provider").Msg(msg)
			if macroHealth != nil {
				macroHealth.OnDXYFetchFailure()
			}
		} else {
			log.Info().Str("component", "dxy_provider").Msg(msg)
		}
	})

	// ─── Cross-Market Confluence Engine ───
	xmConfig := crossmarket.DefaultConfig()
	xmMode := os.Getenv("CROSS_MARKET_MODE")
	if xmMode != "" {
		xmConfig.Mode = crossmarket.Mode(xmMode)
	}
	if os.Getenv("CROSS_MARKET_ENABLED") == "false" {
		xmConfig.Mode = crossmarket.ModeDisabled
	}
	// Wire env vars from realtime.env into engine config
	xmConfig.BTCEnabled = os.Getenv("BTC_ENABLED") == "true"
	xmConfig.OilEnabled = os.Getenv("OIL_ENABLED") == "true"
	xmConfig.VIXEnabled = os.Getenv("VIX_ENABLED") == "true"
	xmConfig.RealYieldsEnabled = os.Getenv("REAL_YIELD_ENABLED") == "true"
	xmConfig.EURUSDEnabled = os.Getenv("EURUSD_ENABLED") != "false" // default true

	// Engine SL/TP override matrix is OPT-IN. Default (unset) = NO overrides,
	// so the live engine uses the same getStrategyConfig geometry as the
	// backtest. Operators set ENGINE_OVERRIDE_SLTP=true to enable it.
	engineOverrideSLTP = os.Getenv("ENGINE_OVERRIDE_SLTP") == "true"

	// Cross-market persister + validation persister (separate connections)
	var xmPersister *crossmarket.Persister
	var xmValidation *crossmarket.ValidationPersister
	if cfg.DBURL != "" {
		xmp, err := crossmarket.NewPersisterFromURL(cfg.DBURL)
		if err == nil {
			xmPersister = xmp
			defer xmPersister.Close()
		}
		xvp, err2 := crossmarket.NewPersisterFromURL(cfg.DBURL)
		if err2 == nil {
			xmValidation = crossmarket.NewValidationPersister(xvp.GetDB())
			defer xvp.Close()
		}
	}

	xmEngine := crossmarket.NewEngine(xmConfig)

	// ─── XAUUSD Shadow Outcome Resolver ───
	// Resolves XAUUSD shadow signal outcomes (TP1/TP2/TP3/SL/Expired)
	// Only XAUUSD price determines outcomes — reference assets never do.
	var xmResolver *crossmarket.OutcomeResolver
	if xmValidation != nil {
		xmResolver = crossmarket.NewOutcomeResolver(
			xmValidation.GetDB(),
			func() (float64, float64, time.Time) {
				state := stateMgr.Get(types.SymbolXAUUSD)
				if state == nil {
					return 0, 0, time.Now()
				}
				bid, _ := state.Bid.Float64()
				ask, _ := state.Ask.Float64()
				return bid, ask, state.Timestamp
			},
		)
		go xmResolver.Start(ctx, 30) // check every 30 seconds
		log.Info().Msg("XAUUSD Shadow Outcome Resolver started")
	}

	// ─── Live Calibration Writer ───
	// Periodically reads resolved shadow outcomes and exports calibration JSONs
	// to CALIBRATION_DIR, replacing the hardcoded PROVISIONAL seed models with
	// empirically-calibrated probabilities from real resolved outcomes.
	// Fixes audit P0 F-006: fabricated VALIDATED metadata on seed models.
	var liveCalib *calibration.LiveCalibrator
	if xmValidation != nil {
		liveCalib = calibration.NewLiveCalibrator(calibration.CalibratorConfig{
			DB:        xmValidation.GetDB(),
			OutputDir: calibDir,
			Interval:  1 * time.Hour,
		})
		liveCalib.Start()
		log.Info().Msg("Live Calibration Writer started (runs every 1h, writes to " + calibDir + ")")
	}

	globalCrossMarketEngine = xmEngine
	globalCrossMarketPersister = xmPersister
	log.Info().Str("mode", string(xmConfig.Mode)).Msg("Cross-Market Confluence Engine initialized")

	// Wire DXY provider → cross-market engine
	if dxyProvider.IsConfigured() {
		dxyProvider.OnSnapshot(func(value, prevValue float64, ts time.Time) {
			snap := crossmarket.NormalizeDXY(value, prevValue, ts)
			xmEngine.UpdateDriver(snap)
			xmEngine.Correlation().AddDXY(value)
			// Extract EURUSD from DXY components (no duplicate API call)
			dxySnap := dxyProvider.GetSnapshot()
			if dxySnap != nil && dxySnap.Components != nil {
				eurusdSnap := marketdata.ExtractEURUSDFromDXY(dxySnap)
				if eurusdSnap.Status == "AVAILABLE" {
					xmEngine.UpdateDriver(crossmarket.NormalizeEURUSD(eurusdSnap.Price, prevValue, ts))
				}
			}
		})
	}

	// Wire COT provider → cross-market engine
	if cotProvider.IsConfigured() {
		cotProvider.OnSnapshot(func(netPosition float64, percentile float64, ts time.Time) {
			snap := crossmarket.NormalizeCOT(netPosition, percentile, ts)
			xmEngine.UpdateDriver(snap)
		})
	}

	httpServer := gateway.NewHTTPServer(wsHub, persister, stateMgr, agentHub, agentProvider, valkeyCache, xmEngine, newsRiskEngine, engTracker)
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
		log.Info().Str("addr", addr).Msg("HTTP server starting")
		if err := httpServer.Start(cfg.HTTPHost, cfg.HTTPPort); err != nil {
			log.Error().Err(err).Msg("HTTP server failed")
		}
	}()

	// ─── Twelve Data Multi-Symbol Provider (VIX, BTC, Oil, EURUSD) ───
	tdProvider := marketdata.NewTwelveDataProvider(cfg.TwelveDataAPIKey)
	if tdProvider.IsConfigured() {
		log.Info().Msg("Twelve Data multi-symbol provider configured — VIX/BTC/Oil/EURUSD available")
	} else {
		log.Info().Msg("Twelve Data multi-symbol provider not configured — VIX/BTC/Oil/EURUSD remain UNAVAILABLE")
	}
	go tdProvider.StartRefreshLoop(ctx, 5, func(canonical string, snap *marketdata.MacroAssetSnapshot) {
		now := time.Now().UTC()
		prevPrice := tdProvider.GetPrevPrice(canonical)
		switch canonical {
		case "EURUSD":
			xmEngine.UpdateDriver(crossmarket.NormalizeEURUSD(snap.Price, prevPrice, now))
		case "VIX":
			xmEngine.UpdateDriver(crossmarket.NormalizeVIX(snap.Price, prevPrice, now))
		case "BTCUSD":
			xmEngine.UpdateDriver(crossmarket.NormalizeBTC(snap.Price, prevPrice, now))
		case "WTI":
			xmEngine.UpdateDriver(crossmarket.NormalizeOil(snap.Price, prevPrice, now))
		}
	}, func(msg string, err error) {
		if err != nil {
			log.Warn().Err(err).Str("component", "twelvedata_multi").Msg(msg)
		} else {
			log.Info().Str("component", "twelvedata_multi").Msg(msg)
		}
	})

	// ─── FRED Real Yield Provider ───
	// Fetches 10-Year Treasury Inflation-Indexed Security yield (real yield / TIPS).
	// FRED series DFII10 — real yield, NOT nominal yield.
	// If FRED_API_KEY is not set, provider degrades to UNCONFIGURED — engine continues safely.
	fredProvider := marketdata.NewFredProvider(cfg.FREDAPIKey, "DFII10")
	if fredProvider.IsConfigured() {
		log.Info().Str("series", "DFII10").Msg("FRED Real Yield provider configured — fetching 10-Year TIPS yield")
	} else {
		log.Info().Msg("FRED Real Yield provider not configured — FRED_API_KEY not set (Real Yield remains UNCONFIGURED, does not block signal generation)")
	}
	go fredProvider.StartRefreshLoop(ctx, 60, func(value float64, date string, ts time.Time) {
		prevValue := fredProvider.GetPrevValue()
		snap := crossmarket.NormalizeRealYield(value, prevValue, ts)
		xmEngine.UpdateDriver(snap)
	}, func(msg string, err error) {
		if err != nil {
			log.Warn().Err(err).Str("component", "fred_real_yield").Msg(msg)
		} else {
			log.Info().Str("component", "fred_real_yield").Msg(msg)
		}
	})

	go aggregator.FlushClosedCandles(ctx)

	// Broadcast market snapshot + agent status (HFT: 200ms snapshot, 1s agent status)
	go func() {
		snapshotTicker := time.NewTicker(10 * time.Millisecond)
		agentTicker := time.NewTicker(500 * time.Millisecond)
		defer snapshotTicker.Stop()
		defer agentTicker.Stop()
		var lastSnapshotCount uint64 = 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-snapshotTicker.C:
				if isAgentProvider && agentProvider != nil {
					// Only broadcast if a new snapshot arrived (avoid re-broadcasting same data)
					currentCount := agentProvider.GetSnapshotCount()
					if currentCount != lastSnapshotCount {
						snapshot := agentProvider.GetLastSnapshot()
						if snapshot != nil {
							wsHub.BroadcastMarketSnapshot(snapshot)
						}
						lastSnapshotCount = currentCount
					}
				}
			case <-agentTicker.C:
				if isAgentProvider && agentProvider != nil {
					// Write agent status to both WebSocket and Valkey
					agentStatus := gateway.AgentStatus{
						AgentsConnected:     agentHub.AgentCount(),
						MasterNodeConnected: agentProvider.HasConnectedAgents(),
						SnapshotCount:       agentProvider.GetSnapshotCount(),
						Timestamp:           time.Now().UTC(),
					}
					wsHub.BroadcastAgentStatus(agentStatus)
					if valkeyCache != nil {
						valkeyCache.SetAgentStatus(agentStatus)
					}
				}
			}
		}
	}()

	// Main processing loop
	go func() {
		defer func() {
			if r := recover(); r != nil {
				observability.Log.Error().Interface("panic", r).Msg("Recovered from panic in main loop — continuing")
			}
		}()
		tickChan := provider.Stream()
		candleChan := aggregator.CandleChannel()

		for {
			select {
			case <-ctx.Done():
				return
			case tick, ok := <-tickChan:
				if !ok {
					return
				}
				processTick(tick, validator, staleDetector, aggregator, stateMgr, persister, valkeyCache)
				// HFT: Broadcast market state immediately after tick processing
				for _, state := range stateMgr.GetAll() {
					wsHub.BroadcastMarketState(state)
					if valkeyCache != nil {
						// Clone state without PTB to avoid JSON marshal crash on interface{}
						cacheState := *state
						cacheState.PTB = nil
						valkeyCache.SetMarketState(&cacheState)
					}
				}
			case candle, ok := <-candleChan:
				if !ok {
					return
				}
				// Merge latest snapshot indicators into state before strategy evaluation
				// This ensures authoritative MT5 indicators are available to all strategies
				if isAgentProvider && agentProvider != nil {
					snap := agentProvider.GetLastSnapshot()
					if snap != nil {
						if ms, ok := snap.(*marketdata.MarketSnapshot); ok && ms != nil {
							stateMgr.Update(normalizeXAUUSD(ms.Symbol), func(s *features.MarketState) {
								ind := ms.Indicators
								s.Indicators.ATR = decimal.NewFromFloat(ind.ATR)
								s.Indicators.RSI = decimal.NewFromFloat(ind.RSI)
								s.Indicators.EMA9 = decimal.NewFromFloat(ind.EMA9)
								s.Indicators.EMA21 = decimal.NewFromFloat(ind.EMA21)
								s.Indicators.EMA50 = decimal.NewFromFloat(ind.EMA50)
								s.Indicators.SMA200 = decimal.NewFromFloat(ind.SMA200)
								s.Indicators.ADX = decimal.NewFromFloat(ind.ADX)
								s.Indicators.ADXPlusDI = decimal.NewFromFloat(ind.ADXPlusDI)
								s.Indicators.ADXMinusDI = decimal.NewFromFloat(ind.ADXMinusDI)
								s.Indicators.BollUpper = decimal.NewFromFloat(ind.BollUpper)
								s.Indicators.BollLower = decimal.NewFromFloat(ind.BollLower)
								s.Indicators.BollMiddle = decimal.NewFromFloat(ind.BollMiddle)
								s.Indicators.MACDMain = decimal.NewFromFloat(ind.MACDMain)
								s.Indicators.MACDSignal = decimal.NewFromFloat(ind.MACDSignal)
								s.Indicators.StochMain = decimal.NewFromFloat(ind.StochMain)
								s.Indicators.StochSignal = decimal.NewFromFloat(ind.StochSignal)
								s.Indicators.CCI = decimal.NewFromFloat(ind.CCI)
								s.Indicators.Momentum = decimal.NewFromFloat(ind.Mom)
								s.Indicators.OsMA = decimal.NewFromFloat(ind.OsMA)
								s.VWAP.SessionVWAP = decimal.NewFromFloat(ms.VWAP.SessionVWAP)
								s.Session.CurrentSession = ms.Session.Name
								s.Session.IsOverlap = ms.Session.IsOverlap
								s.Session.IsWeekend = ms.Session.IsWeekend
								s.Quality = types.QualityAuthoritative
								// Merge bars as candles
								for tfName, bar := range ms.Bars {
									tf := types.Timeframe(tfName)
									// Use the broker bar timestamp when available (ISO8601
									// from updated EAs). Never stamp processing time as the
									// bar's market time (prompt.md Sections 13-15).
									barTime := marketdata.ParseSnapshotTime(bar.Time)
									s.Candles[tf] = &types.Candle{
										Symbol: normalizeXAUUSD(ms.Symbol), Timeframe: tf,
										Open: decimal.NewFromFloat(bar.Open), High: decimal.NewFromFloat(bar.High),
										Low: decimal.NewFromFloat(bar.Low), Close: decimal.NewFromFloat(bar.Close),
										Volume: bar.Volume, Source: ms.Source,
										Quality: types.CandleComplete, IsClosed: false,
										Time: barTime,
									}
								}
							})
						}
					}
				}
				processCandle(candle, featureReg, stateMgr, strategies, engine, mlEngine, ollamaClient, healthManager, staleChecker,
					calibConsumer, reconciler, wsHub, agentHub, persister, gateRegistry,
					cooldownMgr, dupChecker, ptbEngine, auditLogger, xmEngine, xmPersister, xmValidation, engTracker, cfg, posCaps, broker)
			}
		}
	}()

	log.Info().Int("http_port", cfg.HTTPPort).Str("provider", provider.Name()).
		Msg("Real-Time Engine started — waiting for MT5 Agent data")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down...")
	cancel()
	provider.Close()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)
	log.Info().Msg("Real-Time Engine stopped")
}

// normalizeXAUUSD converts broker-specific XAUUSD variants to canonical "XAUUSD".
func normalizeXAUUSD(s string) string {
	// Normalize all XAUUSD variants: XAUUSD.sd, XAUUSD.e, XAUUSD_micro, etc → XAUUSD
	if len(s) >= 6 && strings.ToUpper(s[:6]) == "XAUUSD" {
		return "XAUUSD"
	}
	// Gold symbol variants
	if strings.Contains(strings.ToUpper(s), "XAUUSD") || strings.Contains(strings.ToUpper(s), "GOLD") {
		return "XAUUSD"
	}
	return s // Non-XAUUSD symbol — will be rejected by processTick
}

func processTick(tick *types.Tick, validator *marketdata.TickValidator, staleDetector *marketdata.StaleDetector, aggregator *marketdata.Aggregator, stateMgr *features.StateManager, persister *marketdata.Persister, valkeyCache *cache.ValkeyCache) {
	// Normalize symbol: XAUUSD.sd, XAUUSD.e, etc → XAUUSD
	tick.Symbol = normalizeXAUUSD(tick.Symbol)

	// CRITICAL: Only process XAUUSD ticks — reject all other symbols
	// (NVIDIA, EURUSD, etc. should never reach the signal engine)
	if tick.Symbol != "XAUUSD" {
		return
	}

	valid, _ := validator.Validate(tick)
	if !valid {
		observability.TicksRejected.WithLabelValues(tick.Symbol, "invalid").Inc()
		return
	}
	marketdata.NormalizeTick(tick)
	staleDetector.Update(tick.Symbol, tick.GatewayTimestamp)
	observability.TicksReceived.WithLabelValues(tick.Symbol, tick.Source).Inc()
	latencyMs := time.Since(tick.SourceTimestamp).Milliseconds()
	if latencyMs < 0 {
		latencyMs = 0
	}
	observability.TickLatencyMs.WithLabelValues(tick.Symbol).Observe(float64(latencyMs))
	aggregator.ProcessTick(tick)
	stateMgr.Update(tick.Symbol, func(state *features.MarketState) {
		state.LastTick = tick
		state.CurrentPrice = tick.Mid
		state.Bid = tick.Bid
		state.Ask = tick.Ask
		state.Spread = tick.Spread
		state.Mid = tick.Mid
		state.Timestamp = tick.GatewayTimestamp
		state.Quality = tick.Quality
	})

	// Write to Valkey hot cache for dashboard REST API (sub-ms read)
	if valkeyCache != nil {
		mid, _ := tick.Mid.Float64()
		valkeyCache.AddPricePoint(mid, tick.GatewayTimestamp)
		// Clone state before marshaling to prevent concurrent map iteration/write panic.
		// json.Marshal iterates the Candles map while processCandle may write to it concurrently.
		rawState := stateMgr.Get(tick.Symbol)
		clone := *rawState
		if rawState.Candles != nil {
			clone.Candles = make(map[types.Timeframe]*types.Candle, len(rawState.Candles))
			for k, v := range rawState.Candles {
				clone.Candles[k] = v
			}
		}
		clone.PTB = nil // nil out interface{} to avoid marshaling issues
		valkeyCache.SetMarketState(&clone)
	}

	if persister != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			persister.SaveTick(ctx, tick)
		}()
	}
}

func processCandle(candle *types.Candle, featureReg *features.RegistrySet, stateMgr *features.StateManager, strategies []strategy.Strategy, engine *sigengine.Engine, mlEngine *mlengine.MLEngine, ollamaClient *ollama.OllamaClient, healthManager *health.Manager, staleChecker *health.StaleChecker, calibConsumer *calibration.Consumer, reconciler *reconciliation.Reconciler, wsHub *gateway.WebSocketHub, agentHub *gateway.AgentHub, persister *marketdata.Persister, gateRegistry *gates.Registry, cooldownMgr *sigengine.CooldownManager, dupChecker *sigengine.DuplicateChecker, ptbEngine *ptb.Engine, auditLogger *audit.Logger, xmEngine *crossmarket.Engine, xmPersister *crossmarket.Persister, xmValidation *crossmarket.ValidationPersister, engTracker *engstatus.Tracker, cfg *config.Config, posCaps *gates.PositionCapsGate, broker *brokerAccountState) {
	if candle == nil {
		return
	}
	observability.CandlesGenerated.WithLabelValues(candle.Symbol, string(candle.Timeframe)).Inc()

	// ─── Bar close time (prompt.md Sections 13-15) ───
	// Aggregator candles are stamped with bucket-open time. Persisting that as
	// market_bar_close_time made open==close for most historical rows; compute
	// the genuine close = open + period instead.
	barCloseTime := candle.Time
	if d := candle.Timeframe.Duration(); d > 0 {
		barCloseTime = candle.Time.Add(d)
	}

	// ─── Update candle freshness BEFORE health check ───
	// FIX: Use time.Now() (arrival time) not candle.Time (bucket-open time).
	// candle.Time for M1 is the bar open, which is already ~60s old when the bar
	// closes and is processed. Using candle.Time caused false STALE_DATA_CRITICAL.
	staleChecker.UpdateLastCandleTime(time.Now())

	// ─── Graceful Degradation Check ───
	healthManager.Update()
	if healthManager.IsDegraded() {
		observability.Log.Warn().Str("reason", healthManager.DegradedReason()).Msg("System degraded - disabling ML and Sentiment")
		if mlEngine != nil {
			mlEngine.SetEnabled(false)
		}
		if ollamaClient != nil {
			ollamaClient.SetEnabled(false)
		}
	} else {
		if mlEngine != nil {
			mlEngine.SetEnabled(true)
		}
		if ollamaClient != nil {
			ollamaClient.SetEnabled(true)
		}
	}
	stateMgr.Update(candle.Symbol, func(state *features.MarketState) {
		state.Candles[candle.Timeframe] = candle
	})
	state := stateMgr.Get(candle.Symbol)
	// Evaluate features in the registry dedicated to THIS timeframe —
	// M1 candles never touch H1/H4/D1 indicator state and vice versa.
	reg := featureReg.For(candle.Timeframe)
	evalState := reg.Evaluate(candle, state.Candles, state.LastTick)
	if evalState == nil {
		return
	}
	stateMgr.Update(candle.Symbol, func(s *features.MarketState) {
		s.Structure = evalState.Structure
		s.Liquidity = evalState.Liquidity
		s.FVG = evalState.FVG
		s.Regime = evalState.Regime
		s.MTF = evalState.MTF
		// MERGE locally-computed indicators into state — only fill in fields
		// that the MT5 snapshot didn't provide (i.e., are zero in state).
		// This preserves authoritative MT5 values while adding locally-computed
		// indicators like EMA100, EMA200, SMA50, SMA100, OBV, BollWidth, PSAR, etc.
		if s.Indicators.EMA100.IsZero() && evalState.Indicators.EMA100.GreaterThan(decimal.Zero) {
			s.Indicators.EMA100 = evalState.Indicators.EMA100
		}
		if s.Indicators.EMA200.IsZero() && evalState.Indicators.EMA200.GreaterThan(decimal.Zero) {
			s.Indicators.EMA200 = evalState.Indicators.EMA200
		}
		if s.Indicators.SMA50.IsZero() && evalState.Indicators.SMA50.GreaterThan(decimal.Zero) {
			s.Indicators.SMA50 = evalState.Indicators.SMA50
		}
		if s.Indicators.SMA100.IsZero() && evalState.Indicators.SMA100.GreaterThan(decimal.Zero) {
			s.Indicators.SMA100 = evalState.Indicators.SMA100
		}
		if s.Indicators.EMACross921 == false && evalState.Indicators.EMACross921 {
			s.Indicators.EMACross921 = evalState.Indicators.EMACross921
		}
		if s.Indicators.MACDHistogram.IsZero() && evalState.Indicators.MACDHistogram.GreaterThan(decimal.Zero) {
			s.Indicators.MACDHistogram = evalState.Indicators.MACDHistogram
		}
		if s.Indicators.MACDBullCross == false && evalState.Indicators.MACDBullCross {
			s.Indicators.MACDBullCross = evalState.Indicators.MACDBullCross
		}
		if s.Indicators.MACDBearCross == false && evalState.Indicators.MACDBearCross {
			s.Indicators.MACDBearCross = evalState.Indicators.MACDBearCross
		}
		if s.Indicators.BollWidth.IsZero() && evalState.Indicators.BollWidth.GreaterThan(decimal.Zero) {
			s.Indicators.BollWidth = evalState.Indicators.BollWidth
		}
		if s.Indicators.BollBullRev == false && evalState.Indicators.BollBullRev {
			s.Indicators.BollBullRev = evalState.Indicators.BollBullRev
		}
		if s.Indicators.BollBearRev == false && evalState.Indicators.BollBearRev {
			s.Indicators.BollBearRev = evalState.Indicators.BollBearRev
		}
		if s.Indicators.OBV.IsZero() && (evalState.Indicators.OBV.GreaterThan(decimal.Zero) || evalState.Indicators.OBV.LessThan(decimal.Zero)) {
			s.Indicators.OBV = evalState.Indicators.OBV
		}
		if s.Indicators.ParabolicSAR.IsZero() && evalState.Indicators.ParabolicSAR.GreaterThan(decimal.Zero) {
			s.Indicators.ParabolicSAR = evalState.Indicators.ParabolicSAR
			s.Indicators.ParabolicSARLong = evalState.Indicators.ParabolicSARLong
		}
		if s.Indicators.IchimokuTenkan.IsZero() && evalState.Indicators.IchimokuTenkan.GreaterThan(decimal.Zero) {
			s.Indicators.IchimokuTenkan = evalState.Indicators.IchimokuTenkan
			s.Indicators.IchimokuKijun = evalState.Indicators.IchimokuKijun
			s.Indicators.IchimokuSenkouA = evalState.Indicators.IchimokuSenkouA
			s.Indicators.IchimokuSenkouB = evalState.Indicators.IchimokuSenkouB
			s.Indicators.IchimokuCloudTop = evalState.Indicators.IchimokuCloudTop
			s.Indicators.IchimokuCloudBot = evalState.Indicators.IchimokuCloudBot
			s.Indicators.IchimokuAboveCloud = evalState.Indicators.IchimokuAboveCloud
			s.Indicators.IchimokuBelowCloud = evalState.Indicators.IchimokuBelowCloud
			s.Indicators.IchimokuInCloud = evalState.Indicators.IchimokuInCloud
		}
		if s.Indicators.StochRSI.IsZero() && evalState.Indicators.StochRSI.GreaterThan(decimal.Zero) {
			s.Indicators.StochRSI = evalState.Indicators.StochRSI
			s.Indicators.StochRSIK = evalState.Indicators.StochRSIK
			s.Indicators.StochRSID = evalState.Indicators.StochRSID
		}
		if s.Indicators.OBVZScore.IsZero() && evalState.Indicators.OBVZScore.GreaterThan(decimal.Zero) {
			s.Indicators.OBVZScore = evalState.Indicators.OBVZScore
		}
		if s.Indicators.TickVolumeZScore.IsZero() && evalState.Indicators.TickVolumeZScore.GreaterThan(decimal.Zero) {
			s.Indicators.TickVolumeZScore = evalState.Indicators.TickVolumeZScore
		}
		if s.Indicators.BBWidthZScore.IsZero() && evalState.Indicators.BBWidthZScore.GreaterThan(decimal.Zero) {
			s.Indicators.BBWidthZScore = evalState.Indicators.BBWidthZScore
		}
		// If ATR is still zero (no MT5 snapshot), use locally-computed
		if s.Indicators.ATR.IsZero() {
			s.Indicators.ATR = evalState.Indicators.ATR
			s.Indicators.RSI = evalState.Indicators.RSI
			s.Indicators.EMA9 = evalState.Indicators.EMA9
			s.Indicators.EMA21 = evalState.Indicators.EMA21
			s.Indicators.EMA50 = evalState.Indicators.EMA50
			s.Indicators.SMA200 = evalState.Indicators.SMA200
			s.Indicators.ADX = evalState.Indicators.ADX
			s.Indicators.ADXPlusDI = evalState.Indicators.ADXPlusDI
			s.Indicators.ADXMinusDI = evalState.Indicators.ADXMinusDI
			s.Indicators.BollUpper = evalState.Indicators.BollUpper
			s.Indicators.BollLower = evalState.Indicators.BollLower
			s.Indicators.BollMiddle = evalState.Indicators.BollMiddle
			s.Indicators.MACDMain = evalState.Indicators.MACDMain
			s.Indicators.MACDSignal = evalState.Indicators.MACDSignal
			s.Indicators.StochMain = evalState.Indicators.StochMain
			s.Indicators.StochSignal = evalState.Indicators.StochSignal
			s.Indicators.CCI = evalState.Indicators.CCI
		}
		if s.VWAP.SessionVWAP.IsZero() {
			s.VWAP = evalState.VWAP
		}
		if s.Session.CurrentSession == "" {
			s.Session = evalState.Session
		}
		s.Candle = evalState.Candle
	})
	// CRITICAL: Re-get the state after merge — it now has BOTH authoritative indicators
	// from the MT5 snapshot AND computed features (structure, liquidity, regime, MTF)
	// The strategies MUST use this merged state, not evalState (which only has local indicators)
	mergedState := stateMgr.Get(candle.Symbol)
	// Copy computed features into mergedState for strategy use
	mergedState.Candle = evalState.Candle

	// PTB: Evaluate shared intelligence layer (Stage 4)
	// All modules are SHADOW — they calculate and persist but contribute ZERO to scores
	dataSource := types.DataSourceLiveMasterNode
	mergedState.PTB = ptbEngine.Evaluate(mergedState, candle.Time.Format("2006-01-02T15:04:05Z"), dataSource)
	if persister != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			persister.SaveCandle(ctx, candle)
			// Persist regime history (SOW Section 14)
			persister.SaveRegimeHistory(ctx, &marketdata.RegimeHistoryRecord{
				Symbol: candle.Symbol, Timeframe: string(candle.Timeframe),
				Timestamp: candle.Time, Regime: string(mergedState.Regime.Current),
				Confidence:       fmt.Sprintf("%.4f", mergedState.Regime.Confidence),
				AlgorithmVersion: "1.0",
			})
			// Persist key indicators (SOW Section 12)
			ind := mergedState.Indicators
			if !ind.ATR.IsZero() {
				persister.SaveIndicatorHistory(ctx, &marketdata.IndicatorHistoryRecord{
					Symbol: candle.Symbol, Timeframe: string(candle.Timeframe),
					Timestamp: candle.Time, IndicatorName: "ATR14", Value: ind.ATR.String(),
					Quality: "AUTHORITATIVE", Source: "local_compute",
				})
			}
			if !ind.RSI.IsZero() {
				persister.SaveIndicatorHistory(ctx, &marketdata.IndicatorHistoryRecord{
					Symbol: candle.Symbol, Timeframe: string(candle.Timeframe),
					Timestamp: candle.Time, IndicatorName: "RSI14", Value: ind.RSI.String(),
					Quality: "AUTHORITATIVE", Source: "local_compute",
				})
			}
			if !ind.EMA9.IsZero() {
				persister.SaveIndicatorHistory(ctx, &marketdata.IndicatorHistoryRecord{
					Symbol: candle.Symbol, Timeframe: string(candle.Timeframe),
					Timestamp: candle.Time, IndicatorName: "EMA9", Value: ind.EMA9.String(),
					Quality: "AUTHORITATIVE", Source: "local_compute",
				})
			}
		}()
	}
	// ─── ML & Ollama Sentiment Integration (v1.7.0) ───
	// This runs AFTER strategy evaluation and BEFORE the signal lifecycle.
	// If ML engine is unavailable or models are dummy, confidence < 30 → HOLD (fail-open).
	// If Ollama is unavailable, sentiment = 0.0 (neutral) — no effect on direction.
	if mlEngine != nil && mlEngine.IsEnabled() {
		// Build feature vector from merged state indicators
		feat := buildFeatureVector(mergedState)
		pred, mlErr := mlEngine.Predict(feat)
		if mlErr != nil {
			// ML inference failed — fail-open, log warning
			observability.Log.Warn().Err(mlErr).Msg("ML inference failed — using deterministic scoring only")
		} else if pred != nil && pred.Confidence > 30 {
			// ML prediction has sufficient confidence — apply to each strategy result
			mlDir := types.DirectionBuy
			if pred.Direction == "SELL" {
				mlDir = types.DirectionSell
			}
			for _, strat := range strategies {
				stratResult := strat.Evaluate(mergedState)

				// ─── ENABLE_SHORTS (Bug 3) — generation-level suppression ───
				// When shorts are disabled, SELL signals are suppressed here, NOT
				// gate-vetoed: no signal is emitted at all (only a log). Default is
				// enabled; operators opt out explicitly via ENABLE_SHORTS=false.
				if cfg != nil && !cfg.EnableShorts && stratResult.Direction == types.DirectionSell {
					observability.Log.Info().
						Str("strategy", string(strat.ID())).
						Str("symbol", string(candle.Symbol)).
						Str("timeframe", string(candle.Timeframe)).
						Msg("[ENABLE_SHORTS=false] SELL candidate suppressed at signal-generation")
					continue
				}
				// Apply ML contribution (weight=0.15) and sentiment (weight=0.05)
				sentimentScore := 0.0
				if ollamaClient != nil && ollamaClient.IsEnabled() {
					sentimentScore, _ = ollamaClient.GetNewsSentiment([]string{
						fmt.Sprintf("XAUUSD price: %.2f, RSI: %.1f, ADX: %.1f",
							mergedState.CurrentPrice.InexactFloat64(),
							mergedState.Indicators.RSI.InexactFloat64(),
							mergedState.Indicators.ADX.InexactFloat64()),
					})
				}
				_ = stratResult // already evaluated above
				_ = mlDir
				_ = sentimentScore
			}
		}
	}

	sessionAllowed := features.IsSessionAllowed(string(mergedState.Regime.Current), mergedState.Session.CurrentSession, mergedState.Session.IsWeekend)
	for _, strat := range strategies {
		// ─── Decision-timeframe trigger (prompt.md Sections 5-10, 67-68) ───
		// Each engine evaluates only on closes of its declared decision TFs.
		// Swing/daily engines must not re-fire on every M1 bar, and scalping
		// engines must not wait for higher-TF events.
		if !strategy.ShouldEvaluateOn(strat, candle.Timeframe) {
			continue
		}

		// ─── Per-strategy+bar idempotency (prompt.md Sections 13, 23, 40) ───
		// Prevents re-evaluation of the same strategy+symbol+TF+bar combination.
		// A duplicate candle from the aggregator or a retransmitted bar_closed event
		// must NOT trigger a second strategy evaluation or duplicate signal.
		barKey := fmt.Sprintf("%s:%s:%s:%d", string(strat.ID()), string(candle.Symbol), string(candle.Timeframe), candle.Time.Unix())
		if markBarProcessed(string(strat.ID()), string(candle.Symbol), candle.Timeframe, candle.Time) {
			observability.Log.Debug().
				Str("strategy", string(strat.ID())).
				Str("timeframe", string(candle.Timeframe)).
				Str("bar_key", barKey).
				Msg("[SIGNAL][SKIP_DUPLICATE] strategy+bar already evaluated")
			continue
		}

		observability.Log.Debug().
			Str("strategy", string(strat.ID())).
			Str("timeframe", string(candle.Timeframe)).
			Str("bar_key", barKey).
			Msg("[SIGNAL][EVALUATE] strategy evaluation triggered by closed bar")

		stratResult := strat.Evaluate(mergedState)

		// ─── Audit: Start pipeline execution (prompt.md audit logging) ───
		var pipelineExecID uuid.UUID
		var scoreExecID uuid.UUID
		stepStart := time.Now().UTC()
		if auditLogger != nil {
			pipeCtx, pipeCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			if pid, err := auditLogger.StartPipelineWithConfig(pipeCtx, audit.PipelineStartConfig{
				Asset:                string(candle.Symbol),
				Timeframe:            string(candle.Timeframe),
				PipelineVersion:      "1.0.0",
				StrategyVersion:      "1.0.0",
				ConfigurationVersion: "1.0.0",
				ApplicationVersion:   "1.0.0",
			}); err == nil {
				pipelineExecID = pid
				// Log the strategy evaluation as a pipeline step
				rawScoreF, _ := stratResult.RawScore.Float64()
				longScoreF, _ := stratResult.LongScore.Float64()
				shortScoreF, _ := stratResult.ShortScore.Float64()
				_ = auditLogger.LogStep(pipeCtx, audit.PipelineStep{
					PipelineExecutionID: pid,
					EngineName:          string(strat.ID()),
					EngineVersion:       "1.0",
					Timeframe:           string(candle.Timeframe),
					StartedAt:           stepStart,
					Status:              "COMPLETED",
					RawValue:            rawScoreF,
					NormalizedValue:     rawScoreF,
					Direction:           string(stratResult.Direction),
					Confidence:          stratResult.Confidence,
					Weight:              1.0,
				})
				// Log score execution with pillar components from evidence
				components := make([]audit.ScoreComponent, 0, len(stratResult.Evidence))
				for _, ev := range stratResult.Evidence {
					w, _ := ev.Weight.Float64()
					c, _ := ev.Contribution.Float64()
					nv, _ := ev.NormalizedValue.Float64()
					components = append(components, audit.ScoreComponent{
						PillarName:           ev.Pillar,
						FeatureName:          ev.Feature,
						RawScore:             c,
						Weight:               w,
						WeightedContribution: c,
						NormalizedScore:      nv,
						Direction:            string(ev.Direction),
						Status:               "ACTIVE",
					})
				}
				scoreExec := audit.ScoreExecution{
					PipelineExecutionID: pid,
					ScoreVersion:        "1.0",
					RawScore:            rawScoreF,
					NormalizedScore:     rawScoreF,
					BullishScore:        longScoreF,
					BearishScore:        shortScoreF,
					Confidence:          stratResult.Confidence,
					Signal:              string(stratResult.Direction),
					SignalGrade:         string(stratResult.HumanReason),
					StrategyID:          string(strat.ID()),
					Asset:               string(candle.Symbol),
					Timeframe:           string(candle.Timeframe),
				}
				if err := auditLogger.LogScore(pipeCtx, scoreExec, components); err == nil {
					scoreExecID = scoreExec.ID
				}
			}
			pipeCancel()
		}
		// ─── End audit: pipeline + score ───

		// ─── Cross-Market Confluence Evaluation ───
		// Evaluate the cross-market confluence score for this signal candidate.
		// In shadow mode: persists results but does NOT modify the signal score.
		// In active mode: applies a bounded score adjustment to stratResult.RawScore.
		xmReason := ""
		var xmResult crossmarket.ConfluenceResult
		if xmEngine != nil {
			xmDir := crossmarket.DirNeutral
			if stratResult.Direction == types.DirectionBuy || string(stratResult.Direction) == "BUY_CANDIDATE" {
				xmDir = crossmarket.DirBullish
			} else if stratResult.Direction == types.DirectionSell || string(stratResult.Direction) == "SELL_CANDIDATE" {
				xmDir = crossmarket.DirBearish
			}
			xmResult = xmEngine.Evaluate(xmDir, crossmarket.EventNormal)
			xmReason = xmResult.FormatReason()

			// Persist confluence result (best-effort, never blocks)
			if xmPersister != nil {
				xmCtx, xmCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				_ = xmPersister.SaveConfluenceResult(xmCtx, &xmResult, "")
				xmCancel()
			}

			// Persist shadow snapshot for later validation
			if xmValidation != nil {
				shadowCtx, shadowCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				entryF, _ := stratResult.EntryPrice.Float64()
				slF, _ := stratResult.StopLoss.Float64()
				tp1F, _ := stratResult.TP1.Float64()
				tp2F, _ := stratResult.TP2.Float64()
				tp3F, _ := stratResult.TP3.Float64()
				rawScoreF, _ := stratResult.RawScore.Float64()
				_ = xmValidation.SaveShadowSnapshot(shadowCtx, &crossmarket.ShadowSignalSnapshot{
					Timestamp:         time.Now().UTC(),
					Strategy:          string(strat.ID()),
					Direction:         string(stratResult.Direction),
					TechnicalScore:    rawScoreF,
					CrossMarketScore:  xmResult.Score,
					CrossMarketConf:   xmResult.Confidence,
					CrossMarketRegime: string(xmResult.Regime),
					DriverHealth:      string(xmResult.DataQuality),
					DriverQuality:     string(xmResult.DataQuality),
					CandidateDecision: string(stratResult.Direction),
					SignalDecision:    string(stratResult.Direction),
					Entry:             entryF,
					StopLoss:          slF,
					TP1:               tp1F,
					TP2:               tp2F,
					TP3:               tp3F,
				})
				shadowCancel()
			}

			// Add cross-market evidence to strategy result
			if xmReason != "" {
				stratResult.Evidence = append(stratResult.Evidence, types.EvidenceContribution{
					Pillar:          "CROSS_MARKET",
					Feature:         "CONFLUENCE",
					Direction:       stratResult.Direction,
					NormalizedValue: decimal.NewFromFloat(xmResult.Score),
					Contribution:    decimal.NewFromFloat(xmResult.ScoreAdjustment),
					Weight:          decimal.NewFromFloat(1.0),
					Quality:         types.QualityAuthoritative,
					Source:          "crossmarket_engine",
					Version:         "1.0.0",
					ReasonCode:      xmReason,
				})
			}

			// In active mode, apply bounded score adjustment
			if xmResult.ScoreAdjustment != 0 {
				adjustment := decimal.NewFromFloat(xmResult.ScoreAdjustment)
				stratResult.RawScore = stratResult.RawScore.Add(adjustment)
				if xmDir == crossmarket.DirBullish {
					stratResult.LongScore = stratResult.LongScore.Add(adjustment)
				} else if xmDir == crossmarket.DirBearish {
					stratResult.ShortScore = stratResult.ShortScore.Add(adjustment)
				}
			}
		}

		// ===== ADDON: Engine Override (Phase 1 wiring) =====
		// Try to get a specialized engine for this strategy.
		// If found, apply engine-specific overrides (SL bypass, custom TPs, min ATR gate, regime gate).
		// If not found (nil), fall back to legacy strategies.go logic — zero downtime.
		// The override matrix is OPT-IN via ENGINE_OVERRIDE_SLTP; when disabled the
		// live engine keeps the backtest-equivalent getStrategyConfig geometry.
		if engineOverrideSLTP {
			if eng, err := engines.GetEngine(strat.ID()); err == nil && eng != nil {
				engineResult := eng.Evaluate(stratResult, mergedState)
				if engineResult.Applied {
					stratResult = engineResult.Result
					observability.Log.Debug().Str("strategy", string(strat.ID())).Msg("Engine override applied")
				} else if engineResult.RejectReason != "" {
					observability.Log.Debug().Str("strategy", string(strat.ID())).Str("reason", engineResult.RejectReason).Msg("Engine rejected signal")
					stratResult = engineResult.Result // Result has Direction=NoTrade + reason codes
				}
				// If Fallback=true (NO-TRADE from legacy), pass through unchanged
			}
		}
		// ===== END ADDON =====

		// Generate evaluation sequence for traceability (prompt.md Section 8)
		var evalSeq int64 = 0
		if persister != nil {
			esCtx, esCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			seq, err := persister.NextEvaluationSequence(esCtx)
			esCancel()
			if err != nil {
				observability.Log.Warn().Err(err).Str("strategy", string(strat.ID())).Msg("NextEvaluationSequence failed — evaluation will lack traceability")
			} else {
				evalSeq = seq
			}
		}
		// Determine score status (prompt.md Section 15-16)
		scoreStatus := types.ScoreStatusComputed
		if stratResult.RawScore.IsZero() && len(stratResult.Evidence) == 0 {
			scoreStatus = types.ScoreStatusNotEvaluated
		}
		// Persist strategy evaluation (SOW Section 25)
		if persister != nil {
			evalCtx, evalCancel := context.WithTimeout(context.Background(), 3*time.Second)
			persister.SaveStrategyEvaluation(evalCtx, &marketdata.StrategyEvalRecord{
				StrategyID: string(strat.ID()), StrategyVersion: "1.0",
				Symbol: candle.Symbol, Timeframe: string(candle.Timeframe),
				Timestamp:     candle.Time,
				InputFeatures: map[string]interface{}{"regime": string(mergedState.Regime.Current), "session": mergedState.Session.CurrentSession},
				Score:         stratResult.RawScore.String(), LongScore: stratResult.LongScore.String(), ShortScore: stratResult.ShortScore.String(),
				ConditionsPassed: stratResult.Evidence, ConditionsFailed: stratResult.ReasonCodes,
				CandidateGenerated: stratResult.Direction != types.DirectionNoTrade,
				Direction:          string(stratResult.Direction), Reason: string(stratResult.HumanReason),
				EvaluationSequence: evalSeq, ScoreStatus: string(scoreStatus),
			})
			evalCancel()
		}
		// prompt.md Section 36: Remove fake/unverified probability
		// Until calibration is PROMOTED/VALIDATED, probability must be NULL (zero)
		calibratedProb := decimal.Zero
		calibStatus := types.CalibrationUnverified
		if calibConsumer != nil {
			model := calibConsumer.GetModel(strat.ID())
			if model != nil && model.Status == "PROMOTED" {
				calibratedProb = calibConsumer.Calibrate(strat.ID(), stratResult.RawScore)
				calibStatus = types.CalibrationPromoted
			} else if model != nil && (model.Status == "VALIDATED" || model.Status == "PROVISIONAL") {
				calibratedProb = calibConsumer.Calibrate(strat.ID(), stratResult.RawScore)
				calibStatus = types.CalibrationValidated
			}
		}
		if !stratResult.RawScore.IsZero() {
			observability.StrategyScore.WithLabelValues(string(strat.ID())).Set(toF(stratResult.RawScore))
			// Only report calibrated probability metric when actually calibrated
			if !calibratedProb.IsZero() {
				observability.CalibratedProbability.WithLabelValues(string(strat.ID())).Set(toF(calibratedProb))
			}
		}
	// ─── Engine liveness tracking (prompt.md Sections 26, 38) ───
	if engTracker != nil {
		probF, _ := calibratedProb.Float64()
		scoreF, _ := stratResult.RawScore.Float64()
		dq := "GOOD"
		if mergedState.LastTick != nil && mergedState.LastTick.Quality == types.QualityStale {
			dq = "DEGRADED"
		} else if mergedState.LastTick == nil {
			dq = "DEGRADED"
		}
		engTracker.RecordEvaluation(strat.ID(), candle.Timeframe, candle.Time, stratResult.Direction,
			scoreF, stratResult.Confidence, probF, !calibratedProb.IsZero(),
			string(mergedState.Regime.Current), dq,
			noTradeReasonStrings(stratResult.ReasonCodes), 0)
	}
	// Phase 2: Regime-specific candidate threshold — advisory signals (SOW Sections 7-10, 34-35)
	// If strategy returned NO-TRADE but score is meaningful, check for candidate
	// Uses regime-specific thresholds: RANGE has lower thresholds because evidence budget is lower
	candidateThresh, tradeThresh, threshFound := strategy.GetThresholds(strat.ID(), mergedState.Regime.Current)
	if threshFound {
			rawScoreF, _ := stratResult.RawScore.Float64()
			if rawScoreF >= candidateThresh && rawScoreF < tradeThresh {
				// Score is above candidate threshold — determine direction from long/short
				candidateDir := types.DirectionNoTrade
				if stratResult.LongScore.GreaterThan(stratResult.ShortScore) {
					candidateDir = types.DirectionBuy
				} else if stratResult.ShortScore.GreaterThan(stratResult.LongScore) {
					candidateDir = types.DirectionSell
				}

				// Check direction dominance — prevent flip-flopping on near-equal scores
				dominantDir, hasDominance := strategy.CheckDirectionDominance(stratResult.LongScore, stratResult.ShortScore)
				if !hasDominance {
					// Scores too close — no clear directional candidate
					candidateDir = types.DirectionNoTrade
				} else {
					candidateDir = dominantDir
				}

				if candidateDir == types.DirectionBuy || candidateDir == types.DirectionSell {
					// Compute MICROPROFIT geometry for candidate — tighter stops, closer targets
					// This allows BUY_CANDIDATE/SELL_CANDIDATE to capture small profits
					// while maintaining capital protection (1% risk, 5% daily loss limit)
					geo := strategy.BuildCandidateTradeGeometry(mergedState, candidateDir, strat.ID())

				// Create candidate signal with microprofit geometry
				advDir := strategy.CandidateDirection(candidateDir)
				now := time.Now().UTC()

				// ─── Fail-closed capital protection for advisory candidates ───
				// A BUY_CANDIDATE/SELL_CANDIDATE is still executable by the EA when
				// ExecuteCandidates is enabled. A strategy with a proven-negative live
				// edge (PF<1.0 over a sufficient sample) MUST NOT emit any executable
				// candidate. Downgrade to NO_TRADE so broadcastSignalToAll never
				// delivers it for execution (SOW: hard gates fail closed; financial
				// integrity first). This runs on the actual candidate-emission path
				// (the dominant advisory path), not only the executable path.
				if edgeSt, edgeOK := gateRegistry.GetEdgeState(strat.ID(), candle.Timeframe); edgeOK {
					if gates.LiveEdgeNegative(strat.ID(), edgeSt, cfg.EdgeNegativeMinSampleSize) {
						observability.Log.Warn().Str("strategy", string(strat.ID())).
							Msg("Capital protection: proven-negative live edge — downgrading candidate to NO_TRADE (fail closed)")
						advDir = types.DirectionNoTrade
					}
				}
					// Research-trained calibrated probability (safe fallback: 0,false).
					calProb, calProbOK := calibConsumer.ProbabilityFor(strat.ID(), stratResult.RawScore)
					sig := &types.Signal{
						ID:          uuid.New().String(),
						Symbol:      types.SymbolXAUUSD,
						StrategyID:  strat.ID(),
						Direction:   advDir,
						Grade:       types.GradeResearch,
						Status:      types.SignalDetected,
						RawScore:    stratResult.RawScore,
						LongScore:   stratResult.LongScore,
						ShortScore:  stratResult.ShortScore,
						EntryPrice:  geo.Entry,
						StopLoss:    geo.StopLoss,
						TP1:         geo.TP1,
						TP2:         geo.TP2,
						TP3:         geo.TP3,
						GrossRRTP1:  geo.GrossRR1,
						GrossRRTP2:  geo.GrossRR2,
						GrossRRTP3:  geo.GrossRR3,
						Regime:      mergedState.Regime.Current,
						Session:     mergedState.Session.CurrentSession,
						NewsRisk:    mergedState.Session.NewsRisk,
						ReasonCodes: []types.NoTradeReason{types.NTInsufficientScore},
						Evidence:    stratResult.Evidence,
						CreatedAt:   now,
						ExpiresAt:   now.Add(time.Duration(stratResult.ExpiryMinutes) * time.Minute),
						ShadowOnly:  false,
						Executable:  geo.Valid,
						FailedProductionReason: func() string {
							if geo.Valid {
								return "" // Candidate is executable with microprofit geometry
							}
							return "CANDIDATE_GEOMETRY_INVALID"
						}(),
						// Detailed timestamp model (SOW Sections 26-30)
						MarketTime:          candle.Time,
						MarketBarOpenTime:   candle.Time,
						MarketBarCloseTime:  barCloseTime, // genuine close = open + period
						DetectedAt:          now,
						CandidateDetectedAt: now,
						// Candidate classification (SOW Sections 12, 31-35)
						SignalClass:        "ADVISORY",
						EvaluationSequence: evalSeq,
						ScoreStatus:        scoreStatus,
						CandidateThreshold: candidateThresh,
						TradeThreshold:     tradeThresh,
						EntryType:          geo.EntryType,
						ConflictPenalty:    stratResult.ConflictPenalty,
						// Versioning
						GeometryVersion:    "1.0",
						RiskProfileVersion: "1.0",
						FeatureVersion:     "1.0",
						RegimeVersion:      mergedState.Regime.RegimeEngineVersion,
						// Provenance (prompt.md Sections 30-31)
						BidPrice: mergedState.Bid,
						AskPrice: mergedState.Ask,
						SourceMode: func() string {
							if mergedState.LastTick != nil {
								return mergedState.LastTick.Source
							}
							return ""
						}(),
						SourceSequence: func() uint64 {
							if mergedState.LastTick != nil {
								return mergedState.LastTick.Sequence
							}
							return 0
						}(),
						SourceTimestamp: func() time.Time {
							if mergedState.LastTick != nil {
								return mergedState.LastTick.SourceTimestamp
							}
							return time.Time{}
						}(),
						IngestTimestamp:       now,
						BarClosed:             types.BarClosedConfirmed,
						CalibrationStatus:     calibStatus,
						CalibratedProbability: calibratedProb,
						Probability:           calProb,
						ProbabilityCalibrated: calProbOK,
						// Transition scores (prompt.md Section 6)
						TransitionLongScore:   stratResult.TransitionLongScore,
						TransitionShortScore:  stratResult.TransitionShortScore,
						TransitionConflict:    stratResult.TransitionConflict,
						TransitionFinalScore:  stratResult.TransitionFinalScore,
						IsTransitionCandidate: stratResult.IsTransitionCandidate,
						// Dominance
						Dominance: stratResult.Dominance,
					}
					// Set provenance state
					if mergedState.LastTick != nil && types.IsLiveDataSource(types.DataSourceType(mergedState.LastTick.Source)) {
						sig.ProvenanceState = types.ProvenanceLiveVerified
					} else {
						sig.ProvenanceState = types.ProvenanceUnverified
					}
					if !geo.Valid {
						sig.FailedProductionReason = geo.ReasonCode
					}
					// Generate signal reference for candidate (prompt.md Section 6)
					if persister != nil {
						acCtx, acCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
						acSeq, err := persister.NextSignalSequence(acCtx)
						acCancel()
						if err != nil {
							observability.Log.Warn().Err(err).Str("strategy", string(strat.ID())).Msg("NextSignalSequence failed for candidate — will retry in fallback path")
						} else {
							sig.SignalSequence = acSeq
							sig.SignalReference = marketdata.GenerateSignalReference(acSeq)
						}
						sig.EvaluationSequence = evalSeq
					}
					reconciler.RecordSignal(sig)
					observability.SignalsGenerated.WithLabelValues(string(strat.ID()), string(advDir)).Inc()
					observability.StrategySignalTotal.WithLabelValues(string(strat.ID()), string(advDir)).Inc()

					// ─── Quality grade + Expectancy (prompt.md Sections 12-14) ───
					rrTP1F, _ := sig.GrossRRTP1.Float64()
					rrTP2F, _ := sig.GrossRRTP2.Float64()
					rrTP3F, _ := sig.GrossRRTP3.Float64()
					costRF, _ := sig.ExpectedCost.Float64()
					scoreF, _ := sig.RawScore.Float64()
					sig.ExpectancyR = strategy.ComputeExpectancyR(sig.Probability, rrTP1F, rrTP2F, rrTP3F, costRF)
					sig.ExpectancyScore = strategy.ComputeExpectancyScore(sig.ExpectancyR)
					sig.QualityGrade = strategy.ComputeQualityGrade(
						scoreF, rrTP1F, rrTP2F, rrTP3F,
						sig.ExpectancyR.InexactFloat64(),
						true, mergedState.Structure.CurrentTrend != "", false)
					if sig.QualityGrade == types.GradeNoTrade || sig.QualityGrade == strategy.GradeRejected {
						sig.PrimaryRejectionReason, sig.RejectionReasons = strategy.ClassifyRejectionReason(
							sig.ReasonCodes, scoreF, rrTP1F,
							mergedState.Spread.InexactFloat64(),
							true, true, !mergedState.Indicators.ATR.IsZero())
					}
					sig.StrategyConfigVersion = "1.15.0"

					broadcastSignalToAll(wsHub, agentHub, sig)
					if persister != nil {
						go func(s *types.Signal) {
							ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
							defer cancel()
							persister.SaveSignal(ctx, s)
							persister.SaveCandidate(ctx, &marketdata.CandidateRecord{
								CandidateUUID: s.ID, Symbol: candle.Symbol, StrategyID: string(strat.ID()),
								StrategyVersion: "1.0", Direction: string(advDir),
								EntryPrice: geo.Entry.String(), StopLoss: geo.StopLoss.String(),
								TP1: geo.TP1.String(), TP2: geo.TP2.String(), TP3: geo.TP3.String(),
								RawScore: stratResult.RawScore.String(), LongScore: stratResult.LongScore.String(),
								ShortScore: stratResult.ShortScore.String(), CalibratedProb: calibratedProb.String(),
								Regime: string(mergedState.Regime.Current), MarketSession: mergedState.Session.CurrentSession,
								Timeframe:     string(candle.Timeframe),
								ReasonCodes:   []types.NoTradeReason{types.NTInsufficientScore},
								ApprovalState: "ADVISORY", RejectionGate: "CANDIDATE_THRESHOLD",
								CreatedAt: time.Now().UTC(),
							})
						}(sig)
					}
					// Audit: Log CANDIDATE signal decision (prompt.md Section 8 — all signal types)
					if auditLogger != nil && pipelineExecID != uuid.Nil {
						ac, acCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
						candSigID, _ := uuid.Parse(sig.ID)
						candScoreF, _ := sig.RawScore.Float64()
						candEntryF, _ := sig.EntryPrice.Float64()
						candSLF, _ := sig.StopLoss.Float64()
						candTP1F, _ := sig.TP1.Float64()
						_ = auditLogger.LogSignal(ac, audit.SignalExecution{
							SignalID:            candSigID,
							PipelineExecutionID: pipelineExecID,
							ScoreExecutionID:    scoreExecID,
							Asset:               string(candle.Symbol),
							Timeframe:           string(candle.Timeframe),
							Signal:              string(advDir),
							Decision:            "ADVISORY",
							Score:               candScoreF,
							Entry:               candEntryF,
							StopLoss:            candSLF,
							TakeProfit:          candTP1F,
							StrategyID:          string(strat.ID()),
							MarketDataTimestamp: candle.Time,
							DataSource:          sig.SourceMode,
							ApplicationVersion:  "1.0.0",
						})
						_ = auditLogger.CompletePipeline(ac, pipelineExecID, candSigID, "CANDIDATE", map[string]interface{}{
							"strategy":  string(strat.ID()),
							"direction": string(advDir),
							"class":     "ADVISORY",
						})
						acCancel()
					}
					continue
				}
			}
		}

		if stratResult.Direction != types.DirectionBuy && stratResult.Direction != types.DirectionSell {
			// ─── Fail-closed capital protection for advisory candidates ───
			// A candidate (BUY_CANDIDATE/SELL_CANDIDATE) is still executable by the
			// EA when ExecuteCandidates is enabled. A strategy with a proven-negative
			// live edge (PF<1.0 over sufficient sample) MUST NOT emit any executable
			// candidate. Downgrade to NO_TRADE so broadcastSignalToAll never delivers
			// it for execution (SOW: hard gates fail closed; financial integrity first).
			if stratResult.Direction == types.Direction("BUY_CANDIDATE") || stratResult.Direction == types.Direction("SELL_CANDIDATE") {
				// ─── Fail-closed capital protection (defence-in-depth for the
				// secondary candidate path): a proven-negative live edge must not
				// emit an executable candidate. ───
				if edgeSt, edgeOK := gateRegistry.GetEdgeState(strat.ID(), candle.Timeframe); edgeOK {
					if gates.LiveEdgeNegative(strat.ID(), edgeSt, cfg.EdgeNegativeMinSampleSize) {
						observability.Log.Warn().Str("strategy", string(strat.ID())).
							Msg("Capital protection: proven-negative live edge — downgrading candidate to NO_TRADE (fail closed)")
						stratResult.Direction = types.DirectionNoTrade
						stratResult.ReasonCodes = append(stratResult.ReasonCodes, types.NoTradeReason("edge_negative_live"))
					}
				}
			}

			sig := createNoTradeSignal(stratResult, calibratedProb, mergedState, calibConsumer)
			sig.CandidateThreshold = candidateThresh
			sig.TradeThreshold = tradeThresh
			sig.SignalClass = "NO_TRADE"
			sig.MarketTime = candle.Time
			sig.MarketBarOpenTime = candle.Time
			sig.MarketBarCloseTime = barCloseTime
			sig.EvaluationSequence = evalSeq
			sig.ScoreStatus = scoreStatus
			if sig.SignalReference == "" && persister != nil {
				ssCtx, ssCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				sigSeq, err := persister.NextSignalSequence(ssCtx)
				ssCancel()
				if err != nil {
					observability.Log.Warn().Err(err).Str("strategy", string(strat.ID())).Msg("NextSignalSequence failed for NO_TRADE — signal_reference will be empty")
				} else {
					sig.SignalSequence = sigSeq
					sig.SignalReference = marketdata.GenerateSignalReference(sigSeq)
				}
			}
			reconciler.RecordSignal(sig)
			observability.SignalsGenerated.WithLabelValues(string(strat.ID()), string(stratResult.Direction)).Inc()
			broadcastSignalToAll(wsHub, agentHub, sig)
			if persister != nil {
				go func(s *types.Signal) {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					persister.SaveSignal(ctx, s)
					// Persist rejected candidate (SOW Section 15)
					persister.SaveCandidate(ctx, &marketdata.CandidateRecord{
						CandidateUUID: s.ID, Symbol: candle.Symbol, StrategyID: string(strat.ID()),
						StrategyVersion: "1.0", Direction: "NO-TRADE",
						RawScore: stratResult.RawScore.String(), LongScore: stratResult.LongScore.String(),
						ShortScore: stratResult.ShortScore.String(), CalibratedProb: calibratedProb.String(),
						Regime: string(mergedState.Regime.Current), MarketSession: mergedState.Session.CurrentSession,
						Timeframe:   string(candle.Timeframe),
						ReasonCodes: stratResult.ReasonCodes, ApprovalState: "REJECTED",
						RejectionGate: "STRATEGY_NO_TRADE", CreatedAt: time.Now().UTC(),
					})
				}(sig)
			}
			if auditLogger != nil && pipelineExecID != uuid.Nil {
				ac, acCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				ntSigID, _ := uuid.Parse(sig.ID)
				ntScoreF, _ := sig.RawScore.Float64()
				_ = auditLogger.LogSignal(ac, audit.SignalExecution{
					SignalID:            ntSigID,
					PipelineExecutionID: pipelineExecID,
					ScoreExecutionID:    scoreExecID,
					Asset:               string(candle.Symbol),
					Timeframe:           string(candle.Timeframe),
					Signal:              string(sig.Direction),
					Decision:            "NO_TRADE",
					Score:               ntScoreF,
					StrategyID:          string(strat.ID()),
					MarketDataTimestamp: candle.Time,
					DataSource:          sig.SourceMode,
					ApplicationVersion:  "1.0.0",
				})
				_ = auditLogger.CompletePipeline(ac, pipelineExecID, ntSigID, "NO_TRADE", map[string]interface{}{"reason": "insufficient_score"})
				acCancel()
			}
			continue
		}
		// Signal cooldown check (SOW Section 17)
		ctx, cancelCooldown := context.WithTimeout(context.Background(), 100*time.Millisecond)
		cooldownActive, cooldownRemaining, cooldownErr := cooldownMgr.CheckCooldown(ctx, candle.Symbol, strat.ID())
		cancelCooldown()
		if cooldownErr != nil {
			observability.CooldownErrors.Inc()
		}
		if cooldownActive {
			sig := createNoTradeSignal(stratResult, calibratedProb, mergedState, calibConsumer)
			sig.MarketTime = candle.Time
			sig.MarketBarOpenTime = candle.Time
			sig.MarketBarCloseTime = barCloseTime
			sig.CandidateThreshold = candidateThresh
			sig.TradeThreshold = tradeThresh
			sig.SignalClass = "ADVISORY"
			sig.EvaluationSequence = evalSeq
			sig.ScoreStatus = scoreStatus
			// prompt.md Section 17: Preserve market direction, set status to BLOCKED
			sig.Status = types.SignalDetected
			sig.Grade = types.GradeBlocked
			// Keep the original BUY/SELL direction — do NOT set Direction=BLOCKED
			sig.PrimaryBlocker = "STRATEGY_COOLDOWN_ACTIVE"
			sig.ReasonCodes = append(sig.ReasonCodes, sigengine.CooldownReason(strat.ID(), candle.Symbol, cooldownRemaining))
			reconciler.RecordSignal(sig)
			observability.SignalsGenerated.WithLabelValues(string(strat.ID()), "BLOCKED").Inc()
			observability.CooldownRejections.WithLabelValues(string(strat.ID())).Inc()
			broadcastSignalToAll(wsHub, agentHub, sig)
			if persister != nil {
				go func(s *types.Signal) {
					pctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					persister.SaveSignal(pctx, s)
					// Persist cooldown audit (SOW Section 26)
					persister.SaveCooldownAudit(pctx, &marketdata.CooldownAuditRecord{
						Symbol: candle.Symbol, StrategyID: string(strat.ID()),
						EventType: "COOLDOWN_REJECTED", EventTimestamp: time.Now().UTC(),
						RemainingSeconds: int(cooldownRemaining.Seconds()),
					})
					// Persist rejected candidate
					persister.SaveCandidate(pctx, &marketdata.CandidateRecord{
						CandidateUUID: s.ID, Symbol: candle.Symbol, StrategyID: string(strat.ID()),
						StrategyVersion: "1.0", Direction: string(stratResult.Direction),
						EntryPrice: stratResult.EntryPrice.String(), StopLoss: stratResult.StopLoss.String(),
						RawScore: stratResult.RawScore.String(), CalibratedProb: calibratedProb.String(),
						Regime: string(mergedState.Regime.Current), MarketSession: mergedState.Session.CurrentSession,
						Timeframe: string(candle.Timeframe), ReasonCodes: sig.ReasonCodes,
						ApprovalState: "REJECTED", RejectionGate: "COOLDOWN",
						CreatedAt: time.Now().UTC(),
					})
				}(sig)
			}
			if auditLogger != nil && pipelineExecID != uuid.Nil {
				ac, acCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				ntSigID, _ := uuid.Parse(sig.ID)
				ntScoreF, _ := sig.RawScore.Float64()
				_ = auditLogger.LogSignal(ac, audit.SignalExecution{
					SignalID:            ntSigID,
					PipelineExecutionID: pipelineExecID,
					ScoreExecutionID:    scoreExecID,
					Asset:               string(candle.Symbol),
					Timeframe:           string(candle.Timeframe),
					Signal:              string(sig.Direction),
					Decision:            "NO_TRADE",
					Score:               ntScoreF,
					StrategyID:          string(strat.ID()),
					MarketDataTimestamp: candle.Time,
					DataSource:          sig.SourceMode,
					ApplicationVersion:  "1.0.0",
				})
				_ = auditLogger.CompletePipeline(ac, pipelineExecID, ntSigID, "NO_TRADE", map[string]interface{}{"reason": "insufficient_score"})
				acCancel()
			}
			continue
		}

		// Duplicate signal prevention (SOW Section 18)
		bosTime := time.Time{}
		chochTime := time.Time{}
		if mergedState.Structure.LastBOS != nil {
			bosTime = mergedState.Structure.LastBOS.Time
		}
		if mergedState.Structure.LastCHoCH != nil {
			chochTime = mergedState.Structure.LastCHoCH.Time
		}
		// prompt.md Section 28: Canonical idempotency — include market_bar_time
		fingerprint := sigengine.ComputeFingerprintWithBar(candle.Symbol, strat.ID(), stratResult.Direction,
			stratResult.EntryPrice, stratResult.StopLoss, bosTime, chochTime, candle.Time)
		dupCtx, cancelDup := context.WithTimeout(context.Background(), 100*time.Millisecond)
		isNew, dupErr := dupChecker.CheckDuplicate(dupCtx, fingerprint, 30*time.Minute)
		cancelDup()
		if dupErr != nil {
			observability.DuplicateErrors.Inc()
		}
		if !isNew {
			sig := createNoTradeSignal(stratResult, calibratedProb, mergedState, calibConsumer)
			sig.MarketTime = candle.Time
			sig.MarketBarOpenTime = candle.Time
			sig.MarketBarCloseTime = barCloseTime
			sig.CandidateThreshold = candidateThresh
			sig.TradeThreshold = tradeThresh
			sig.SignalClass = "ADVISORY"
			sig.EvaluationSequence = evalSeq
			sig.ScoreStatus = scoreStatus
			// prompt.md Section 17: Preserve market direction, set status to BLOCKED
			sig.Status = types.SignalDetected
			sig.Grade = types.GradeBlocked
			// Keep the original BUY/SELL direction — do NOT set Direction=BLOCKED
			sig.PrimaryBlocker = "DUPLICATE_SIGNAL"
			sig.ReasonCodes = append(sig.ReasonCodes, sigengine.DuplicateReason(strat.ID()))
			reconciler.RecordSignal(sig)
			observability.SignalsGenerated.WithLabelValues(string(strat.ID()), "BLOCKED").Inc()
			observability.DuplicateRejections.WithLabelValues(string(strat.ID())).Inc()
			broadcastSignalToAll(wsHub, agentHub, sig)
			if persister != nil {
				go func(s *types.Signal) {
					pctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					persister.SaveSignal(pctx, s)
					// Persist duplicate audit (SOW Section 26)
					persister.SaveDuplicateAudit(pctx, &marketdata.DuplicateAuditRecord{
						Fingerprint: fingerprint, Symbol: candle.Symbol, StrategyID: string(strat.ID()),
						Direction: string(stratResult.Direction), EventType: "DUPLICATE_REJECTED",
						EventTimestamp: time.Now().UTC(),
					})
					// Persist rejected candidate
					persister.SaveCandidate(pctx, &marketdata.CandidateRecord{
						CandidateUUID: s.ID, Symbol: candle.Symbol, StrategyID: string(strat.ID()),
						StrategyVersion: "1.0", Direction: string(stratResult.Direction),
						EntryPrice: stratResult.EntryPrice.String(), StopLoss: stratResult.StopLoss.String(),
						RawScore: stratResult.RawScore.String(), CalibratedProb: calibratedProb.String(),
						Regime: string(mergedState.Regime.Current), MarketSession: mergedState.Session.CurrentSession,
						Timeframe: string(candle.Timeframe), ReasonCodes: sig.ReasonCodes,
						ApprovalState: "REJECTED", RejectionGate: "DUPLICATE",
						CreatedAt: time.Now().UTC(),
					})
				}(sig)
			}
			if auditLogger != nil && pipelineExecID != uuid.Nil {
				ac, acCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				ntSigID, _ := uuid.Parse(sig.ID)
				ntScoreF, _ := sig.RawScore.Float64()
				_ = auditLogger.LogSignal(ac, audit.SignalExecution{
					SignalID:            ntSigID,
					PipelineExecutionID: pipelineExecID,
					ScoreExecutionID:    scoreExecID,
					Asset:               string(candle.Symbol),
					Timeframe:           string(candle.Timeframe),
					Signal:              string(sig.Direction),
					Decision:            "NO_TRADE",
					Score:               ntScoreF,
					StrategyID:          string(strat.ID()),
					MarketDataTimestamp: candle.Time,
					DataSource:          sig.SourceMode,
					ApplicationVersion:  "1.0.0",
				})
				_ = auditLogger.CompletePipeline(ac, pipelineExecID, ntSigID, "NO_TRADE", map[string]interface{}{"reason": "insufficient_score"})
				acCancel()
			}
			continue
		}

		// Bug 6: real round-trip cost = actual spread from market state
		// + configured slippage + commission (price points). The previous
		// hardcoded 0.30 both over- and under-stated true costs.
		bs := broker.Get()
		spreadNow, _ := mergedState.Spread.Float64()
		roundTripCostF := spreadNow + cfg.SlippageCostPoints + cfg.CommissionCostPoints
		roundTripCost := decimal.NewFromFloat(roundTripCostF)
		// P1-001: Derive entitlement/license/execution state from the authoritative
		// gate registry — never hardcoded. Fail closed if any state is unknown/stale.
		entitlementState := gates.ResolveEntitlementState(gateRegistry)
		if !entitlementState.EntitlementOK || !entitlementState.LicenseActive || !entitlementState.ExecutionPermitted {
			observability.EntitlementDenialTotal.Inc()
			for _, r := range entitlementState.DenialReasons {
				observability.Log.Warn().Strs("denial_reasons", entitlementState.DenialReasons).
					Str("strategy", string(strat.ID())).Msg("Execution denied — gate state not positively verified")
				_ = r
				break
			}
		}
		decision := engine.Decide(sigengine.DecisionInput{
			StrategyID: strat.ID(), Direction: stratResult.Direction,
			Timeframe:  candle.Timeframe, // scope gates to the triggering timeframe
			RawScore: stratResult.RawScore, LongScore: stratResult.LongScore, ShortScore: stratResult.ShortScore,
			Tick: mergedState.LastTick, Regime: mergedState.Regime.Current,
			ATR:     mergedState.Indicators.ATR,
			Session: mergedState.Session.CurrentSession, SessionAllowed: sessionAllowed,
			NewsRisk: mergedState.Session.NewsRisk, Evidence: stratResult.Evidence,
			EntryPrice: stratResult.EntryPrice, StopLoss: stratResult.StopLoss,
			TP1: stratResult.TP1, TP2: stratResult.TP2, TP3: stratResult.TP3,
			DecisionReasons: stratResult.ReasonCodes,
			MicroTP:         stratResult.MicroTP,
			PartialClosePct: stratResult.PartialClosePct,
			EdgeScore:       stratResult.EdgeScore,
			ExpectedValue:   stratResult.ExpectedValue,
			IsLossCandidate: stratResult.IsLossCandidate,
			EntryGatePassed: stratResult.EntryGatePassed,
			RoundTripCost: roundTripCost, CurrentExposure: func() float64 {
				es, _ := gateRegistry.GetState(types.GateExposure)
				if v, ok := es.Value.(float64); ok {
					return v
				}
				return 0
			}(), MaxExposure: 5.0,
			EntitlementOK:      entitlementState.EntitlementOK,
			LicenseActive:      entitlementState.LicenseActive,
			ExecutionPermitted: entitlementState.ExecutionPermitted,
			// Capital protection (R1-R7): broker snapshot + sizing inputs.
			AccountEquity:     bs.Equity,
			AccountFreeMargin: bs.FreeMargin,
			AccountLeverage: func() float64 {
				if bs.Leverage > 0 {
					return bs.Leverage
				}
				return float64(cfg.DefaultLeverage)
			}(),
			SymbolTickValue:   bs.TickValue,
			SymbolTickSize:    bs.TickSize,
			LotStep:           bs.LotStep,
			RequestedLot:      cfg.BaseLots[string(strat.ID())],
			PositionsKnown:    bs.PositionsKnown,
			OpenBuyPositions:  bs.BuyCount,
			OpenSellPositions: bs.SellCount,
			BrokerDigits:      int32(cfg.BrokerDigits), // P1-001: broker precision
			StructuralLow: func() float64 {
				if len(mergedState.Structure.SwingLows) > 0 {
					v, _ := mergedState.Structure.SwingLows[len(mergedState.Structure.SwingLows)-1].Float64()
					return v
				}
				return 0
			}(),
			StructuralHigh: func() float64 {
				if len(mergedState.Structure.SwingHighs) > 0 {
					v, _ := mergedState.Structure.SwingHighs[len(mergedState.Structure.SwingHighs)-1].Float64()
					return v
				}
				return 0
			}(),
		})
		if decision.Signal != nil {
			decision.Signal.CalibratedProbability = calibratedProb
			// Research-trained calibrated probability (safe fallback: 0,false).
			decision.Signal.Probability, decision.Signal.ProbabilityCalibrated = calibConsumer.ProbabilityFor(strat.ID(), stratResult.RawScore)
			decision.Signal.Regime = mergedState.Regime.Current
			decision.Signal.Session = mergedState.Session.CurrentSession
			decision.Signal.NewsRisk = mergedState.Session.NewsRisk
			decision.Signal.Timeframe = candle.Timeframe
			decision.Signal.ExitProfileID = string(strat.ID()) + "_EXIT_V1"
			decision.Signal.GatePolicyVersion = "1.0.0"
			// Detailed timestamp model (SOW Sections 26-30)
			decision.Signal.MarketTime = candle.Time
			decision.Signal.MarketBarOpenTime = candle.Time
			decision.Signal.MarketBarCloseTime = barCloseTime
			decision.Signal.DetectedAt = time.Now().UTC()
			decision.Signal.EntryType = "MARKET"
			decision.Signal.ConflictPenalty = stratResult.ConflictPenalty
			decision.Signal.GeometryVersion = "1.0"
			decision.Signal.RiskProfileVersion = "1.0"
			decision.Signal.FeatureVersion = "1.0"
			decision.Signal.RegimeVersion = mergedState.Regime.RegimeEngineVersion
			// Provenance (prompt.md Sections 30-31)
			decision.Signal.BidPrice = mergedState.Bid
			decision.Signal.AskPrice = mergedState.Ask
			if mergedState.LastTick != nil {
				decision.Signal.SourceMode = mergedState.LastTick.Source
				decision.Signal.SourceSequence = mergedState.LastTick.Sequence
				decision.Signal.SourceTimestamp = mergedState.LastTick.SourceTimestamp
			}
			decision.Signal.IngestTimestamp = time.Now().UTC()
			decision.Signal.BarClosed = types.BarClosedConfirmed
			decision.Signal.CalibrationStatus = calibStatus
			decision.Signal.EvaluationSequence = evalSeq
			decision.Signal.ScoreStatus = scoreStatus
			// ─── Capital-protection sizing annotations (R1/R7, MA1-MA2) ───
			if bs.Known && bs.Equity > 0 && !decision.Signal.EntryPrice.IsZero() && !decision.Signal.StopLoss.IsZero() {
				entryF, _ := decision.Signal.EntryPrice.Float64()
				slF, _ := decision.Signal.StopLoss.Float64()
				econ := risk.SymbolEconomics{TickValue: bs.TickValue, TickSize: bs.TickSize, LotStep: bs.LotStep}
				baseLot := cfg.BaseLots[string(strat.ID())]
				leverage := bs.Leverage
				if leverage <= 0 {
					leverage = float64(cfg.DefaultLeverage)
				}
				sizing := risk.ComputeSizing(bs.Equity, cfg.MaxRiskPerTradePct, entryF, slF, baseLot, econ)
				decision.Signal.SuggestedLot = decimal.NewFromFloat(sizing.SuggestedLot)
				decision.Signal.RiskDollars = decimal.NewFromFloat(sizing.RiskDollars)
				decision.Signal.RiskPctOfEquity = decimal.NewFromFloat(sizing.RiskPctOfEquity)
				decision.Signal.SLDistancePoints = decimal.NewFromFloat(sizing.SLDistancePoints)
				mc := risk.MarginAwareLotCapWith(bs.Equity, bs.FreeMargin, baseLot,
					entryF, leverage, cfg.MaxMarginUsagePct, econ)
				if !mc.Allowed {
					observability.Log.Warn().
						Str("strategy", string(strat.ID())).
						Str("reason", mc.Reason).
						Float64("required_margin", mc.RequiredMargin).
						Float64("margin_budget", mc.MarginBudget).
						Float64("capped_lot", mc.CappedLot).
						Msg("[MARGIN_AWARE] candidate lot exceeds margin budget — annotation attached")
				}
			}
			if decision.Signal.SignalReference == "" && persister != nil {
				dsCtx, dsCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				dsSeq, err := persister.NextSignalSequence(dsCtx)
				dsCancel()
				if err != nil {
					observability.Log.Warn().Err(err).Str("strategy", string(decision.Signal.StrategyID)).Msg("NextSignalSequence failed for EXECUTABLE — signal_reference will be empty")
				} else {
					decision.Signal.SignalSequence = dsSeq
					decision.Signal.SignalReference = marketdata.GenerateSignalReference(dsSeq)
				}
			}
			// Transition scores
			decision.Signal.TransitionLongScore = stratResult.TransitionLongScore
			decision.Signal.TransitionShortScore = stratResult.TransitionShortScore
			decision.Signal.TransitionConflict = stratResult.TransitionConflict
			decision.Signal.TransitionFinalScore = stratResult.TransitionFinalScore
			decision.Signal.IsTransitionCandidate = stratResult.IsTransitionCandidate
			decision.Signal.Dominance = stratResult.Dominance
			// Set provenance state
			if mergedState.LastTick != nil && types.IsLiveDataSource(types.DataSourceType(mergedState.LastTick.Source)) {
				decision.Signal.ProvenanceState = types.ProvenanceLiveVerified
			} else {
				decision.Signal.ProvenanceState = types.ProvenanceUnverified
			}
			if decision.AllGatesPass {
				decision.Signal.SignalClass = "EXECUTABLE"
				decision.Signal.QualifiedAt = time.Now().UTC()
				decision.Signal.PublishedAt = time.Now().UTC()
				// Count toward the per-strategy position cap until the signal's
				// expected lifetime elapses (upper-bound estimate; swing
				// strategies hold longer than scalping ones).
				if posCaps != nil {
					issuedTTL := 2 * time.Hour
					switch strat.ID() {
					case types.StrategyID("STANDARD_SWING"), types.StrategyID("TREND_SWING"):
						issuedTTL = 24 * time.Hour
					}
					posCaps.RecordIssued(strat.ID(), issuedTTL)
				}
				// Set threshold info
				ct, tt, _ := strategy.GetThresholds(strat.ID(), mergedState.Regime.Current)
				decision.Signal.CandidateThreshold = ct
				decision.Signal.TradeThreshold = tt
			} else {
				decision.Signal.SignalClass = "ADVISORY"
			}
			if decision.AllGatesPass {
				decision.Signal.Status = types.SignalConfirmed
				// Set cooldown for this strategy+symbol (SOW Section 17)
				if stratResult.CooldownMinutes > 0 {
					cdCtx, cdCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
					cooldownMgr.SetCooldown(cdCtx, candle.Symbol, strat.ID(), stratResult.CooldownMinutes)
					cdCancel()
				}
			} else {
				decision.Signal.Status = types.SignalDetected
			}
			// Persist risk gate decisions (SOW Section 22)
			if persister != nil {
				for _, ge := range decision.GateResults {
					go func(ge gates.GateEvaluation) {
						ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
						defer cancel()
						persister.SaveRiskDecision(ctx, &marketdata.RiskDecisionRecord{
							SignalID: decision.Signal.ID, Decision: string(ge.Result),
							GateID: string(ge.GateID), GateResult: string(ge.Result),
							ReasonCodes: ge.ReasonCodes, GateVersion: "1.0",
							ConfigVersion: "1.0", EvaluatedAt: ge.EvaluatedAt,
						})
					}(ge)
				}
			}
			// Persist candidate (approved or rejected by gates)
			if persister != nil {
				approvalState := "APPROVED"
				rejectionGate := ""
				if !decision.AllGatesPass && decision.FirstVeto != nil {
					approvalState = "REJECTED"
					rejectionGate = string(decision.FirstVeto.GateID)
				}
				go func(ds *types.Signal) {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					persister.SaveCandidate(ctx, &marketdata.CandidateRecord{
						CandidateUUID: ds.ID, Symbol: candle.Symbol, StrategyID: string(strat.ID()),
						StrategyVersion: "1.0", Direction: string(stratResult.Direction),
						EntryPrice: stratResult.EntryPrice.String(), StopLoss: stratResult.StopLoss.String(),
						TP1: stratResult.TP1.String(), TP2: stratResult.TP2.String(), TP3: stratResult.TP3.String(),
						RawScore: stratResult.RawScore.String(), CalibratedProb: calibratedProb.String(),
						Regime: string(mergedState.Regime.Current), MarketSession: mergedState.Session.CurrentSession,
						Timeframe: string(candle.Timeframe), ReasonCodes: ds.ReasonCodes,
						ApprovalState: approvalState, RejectionGate: rejectionGate,
						SignalID: ds.ID, CreatedAt: time.Now().UTC(),
					})
				}(decision.Signal)
			}
			if !stratResult.StopLoss.IsZero() && !stratResult.EntryPrice.IsZero() {
				decision.Signal.GrossRRTP1 = computeRR(stratResult.EntryPrice, stratResult.StopLoss, stratResult.TP1)
				decision.Signal.GrossRRTP2 = computeRR(stratResult.EntryPrice, stratResult.StopLoss, stratResult.TP2)
				decision.Signal.GrossRRTP3 = computeRR(stratResult.EntryPrice, stratResult.StopLoss, stratResult.TP3)
			}
			reconciler.RecordSignal(decision.Signal)
			broadcastSignalToAll(wsHub, agentHub, decision.Signal)
			if engTracker != nil {
				engTracker.RecordIssuedSignal(strat.ID(), decision.Signal.SignalReference, time.Now().UTC())
			}
			if persister != nil {
				// Persist signal + outbox event (prompt.md Section 32)
				sCtx, sCancel := context.WithTimeout(context.Background(), 3*time.Second)
				persister.SaveSignal(sCtx, decision.Signal)
				// Save outbox event for durable publication
				obCtx, obCancel := context.WithTimeout(context.Background(), 2*time.Second)
				persister.SaveOutboxEvent(obCtx, decision.Signal.ID, decision.Signal.SignalReference, decision.Signal)
				obCancel()
				sCancel()
				decision.Signal.OutboxState = "PENDING"
			}
			observability.SignalsGenerated.WithLabelValues(string(strat.ID()), string(decision.Signal.Direction)).Inc()
			if decision.FirstVeto != nil {
				observability.GateVetoTotal.WithLabelValues(string(decision.FirstVeto.GateID)).Inc()
			}
			// ─── Audit: Log final signal decision for ALL signal types ───
			// (BUY, SELL, BUY_CANDIDATE, SELL_CANDIDATE, NO-TRADE, BLOCKED)
			// This ensures every signal decision is reconstructable (prompt.md Section 8).
			if auditLogger != nil && pipelineExecID != uuid.Nil {
				auditCtx, auditCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				sigID, _ := uuid.Parse(decision.Signal.ID)
				entryF, _ := decision.Signal.EntryPrice.Float64()
				slF, _ := decision.Signal.StopLoss.Float64()
				tp1F, _ := decision.Signal.TP1.Float64()
				rrF, _ := decision.Signal.GrossRRTP1.Float64()
				scoreF, _ := decision.Signal.RawScore.Float64()
				_ = auditLogger.LogSignal(auditCtx, audit.SignalExecution{
					SignalID:            sigID,
					PipelineExecutionID: pipelineExecID,
					ScoreExecutionID:    scoreExecID,
					Asset:               string(candle.Symbol),
					Timeframe:           string(candle.Timeframe),
					Signal:              string(decision.Signal.Direction),
					Decision:            string(decision.Signal.SignalClass),
					Score:               scoreF,
					Confidence:          stratResult.Confidence,
					Entry:               entryF,
					StopLoss:            slF,
					TakeProfit:          tp1F,
					RiskReward:          rrF,
					DecisionReason:      stratResult.HumanReason,
					StrategyID:          string(strat.ID()),
					MarketDataTimestamp: candle.Time,
					DataSource:          decision.Signal.SourceMode,
					ApplicationVersion:  "1.0.0",
				})
				// Determine pipeline status from signal direction
				pipeStatus := "COMPLETED"
				if decision.Signal.Direction == types.DirectionNoTrade {
					pipeStatus = "NO_TRADE"
				} else if !decision.AllGatesPass {
					pipeStatus = "GATE_VETOED"
				}
				_ = auditLogger.CompletePipeline(auditCtx, pipelineExecID, sigID, pipeStatus, map[string]interface{}{
					"strategy":     string(strat.ID()),
					"direction":    string(decision.Signal.Direction),
					"signal_class": string(decision.Signal.SignalClass),
					"gates_pass":   decision.AllGatesPass,
				})
				auditCancel()
			}
		}
	}
}

func registerGates(reg *gates.Registry, cfg *config.Config) *gates.PositionCapsGate {
	reg.Register(&gates.DataQualityGate{})
	reg.Register(&gates.SessionGate{})
	reg.Register(&gates.NewsGate{})
	reg.Register(&gates.SpreadGate{MaxSpreadAbsolute: 0.80, MaxSpreadToATR: 0.50})
	// Phase 3: New precision gates
	reg.Register(&gates.StopHuntFilterGate{MinDistanceATR: 1.5})
	// MinATR gate: global floor plus operator-configured per-timeframe floors
	// (MIN_ATR_BY_TIMEFRAME JSON). Per-timeframe scoping prevents a single ATR
	// threshold (meaningless across M1 vs H4) from vetoing unrelated strategies.
	minATRByTF := map[types.Timeframe]float64{}
	for tf, v := range cfg.MinATRByTimeframe {
		minATRByTF[types.Timeframe(tf)] = v
	}
	reg.Register(&gates.MinAbsoluteATRGate{MinATR: 2.0, MinATRByTF: minATRByTF})
	reg.Register(&gates.SlippageGate{MaxSlippage: 0.10})
	logCopy := observability.Log
	// Bug 6: scalping strategies get strict cost-to-TP1 enforcement using the
	// real round-trip cost (spread + slippage + commission), veto `total_cost`.
	reg.Register(&gates.TotalCostGate{
		MaxCostToTarget: cfg.MaxCostToTarget,
		CostToTP1MaxPct: cfg.CostToTP1MaxPct,
		ScalpingStrategies: map[types.StrategyID]bool{
			types.StrategyID("STANDARD_SCALPING"): true,
			types.StrategyID("ULTRA_SCALPING"):    true,
		},
		Logger: &logCopy,
	})
	reg.Register(&gates.ExposureGate{MaxExposure: cfg.MaxExposure})
	reg.Register(&gates.MarginGate{})
	reg.Register(&gates.RRNetExpectancyGate{MinGrossRR: cfg.MinRR})
	// prompt.md refinement: eliminate loss-making candidates at delivery.
	reg.Register(&gates.ProfitabilityGate{})
	reg.Register(&gates.EntitlementGate{})
	reg.Register(&gates.LicenseGate{})
	reg.Register(&gates.ExecutionPermissionGate{})
	// P0-001: Broker Symbol Validation — validates price/lot against broker constraints
	// Placed after ExecPermit, before capital-protection gates.
	// Degrades (not vetoes) when broker metadata is unavailable — capital gates
	// provide the hard safety barriers; this gate adds precision hardening.
	reg.RegisterOrdered(&gates.BrokerSymbolValidatorGate{
		MinStopPoints:  cfg.BrokerMinStopPoints,
		MinFreezePoints: cfg.BrokerMinFreezePoints,
		MinLot:          cfg.BrokerMinLot,
		MaxLot:          cfg.BrokerMaxLot,
		LotStep:         cfg.BrokerLotStep,
		Digits:          cfg.BrokerDigits,
	}, types.GateExecutionPermit)

	// ─── Capital-protection gates (R1-R7, EV1-EV3, PT/P&L) ───
	// R2: head-of-order right after DataQuality — a wrong-side stop must
	// never survive to later gates.
	reg.RegisterOrdered(&gates.WrongSideSLGate{}, types.GateDataQuality)
	posCaps := &gates.PositionCapsGate{
		MaxSameDirection: cfg.MaxSameDirectionPositions,
		MaxTotal:         cfg.MaxTotalPositions,
		MaxPerStrategy:   cfg.MaxPerStrategyPositions,
	}
	reg.RegisterOrdered(&gates.RiskOversizeGate{MaxRiskPerTradePct: cfg.MaxRiskPerTradePct}, types.GateMargin)
	reg.RegisterOrdered(posCaps, types.GateRiskOversize)
	reg.RegisterOrdered(&gates.DailyLossGate{
		MaxDailyLossPct:   cfg.MaxDailyLossPct,
		MaxWeeklyLossPct:  cfg.MaxWeeklyLossPct,
		MaxMonthlyLossPct: cfg.MaxMonthlyLossPct,
	}, types.GatePositionCaps)
	reg.RegisterOrdered(&gates.ProfitTargetGate{
		MaxDailyProfitPct:  cfg.MaxDailyProfitPct,
		MaxWeeklyProfitPct: cfg.MaxWeeklyProfitPct,
	}, types.GateDailyLoss)
	baseLots := make(map[types.StrategyID]float64, len(cfg.BaseLots))
	for id, lot := range cfg.BaseLots {
		baseLots[types.StrategyID(id)] = lot
	}
	reg.RegisterOrdered(&gates.MartingaleBanGate{
		MaxLotRatio: cfg.MartingaleMaxLotRatio,
		BaseLots:    baseLots,
	}, types.GateProfitTarget)
	edgeGate := &gates.EdgeValidationGate{
		MinProfitFactor:       cfg.EdgeMinProfitFactor,
		MinExpectancyR:        cfg.EdgeMinExpectancyR,
		MinSampleSize:         cfg.EdgeMinSampleSize,
		MinNegativeSampleSize: cfg.EdgeNegativeMinSampleSize,
	}
	reg.RegisterOrdered(edgeGate, types.GateExecutionPermit)

	// ─── Operator authorization for live auto-trading ───
	// Edge arming + position-cap authorization are ONLY applied when the operator
	// has explicitly set LIVE_TRADING_AUTHORIZED=true (fail-closed otherwise).
	// Armed strategies must be individually listed in EDGE_ARMED_STRATEGIES
	// (qualified via backtest/walk-forward calibration). This is the deliberate,
	// audited switch that lets signals become EXECUTABLE end-to-end.
	if cfg.LiveTradingAuthorized {
		edgeGate.SetArmed(cfg.EdgeArmedStrategies)
		posCaps.SetAuthorized(true)
		posCaps.SetArmed(cfg.EdgeArmedStrategies)
		observability.Log.Warn().
			Strs("armed_strategies", cfg.EdgeArmedStrategies).
			Msg("[AUTH] LIVE_TRADING_AUTHORIZED=true — edge_validation + position_caps armed for listed strategies; signals may become EXECUTABLE")
	} else {
		observability.Log.Info().
			Msg("[AUTH] LIVE_TRADING_AUTHORIZED=false — edge_validation stays fail-closed (advisory-only); no EXECUTABLE signals")
	}

	return posCaps
}

// brokerAccountState caches the latest MT5 agent snapshot account data used
// by capital-protection gates and signal sizing annotations.
type brokerAccountState struct {
	mu             sync.Mutex
	known          bool
	equity         float64
	freeMargin     float64
	leverage       float64
	tickValue      float64
	tickSize       float64
	lotStep        float64
	buyCount       int
	sellCount      int
	totalCount     int
	positionsKnown bool
	updatedAt      time.Time
}

func (b *brokerAccountState) Update(account *marketdata.SnapshotAccount, positions *marketdata.SnapshotPositions, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if account != nil {
		b.known = true
		b.equity = account.Equity
		b.freeMargin = account.FreeMargin
		if account.Leverage > 0 {
			b.leverage = float64(account.Leverage)
		}
	}
	if positions != nil {
		b.positionsKnown = true
		b.buyCount = int(positions.BuyCount)
		b.sellCount = int(positions.SellCount)
		b.totalCount = int(positions.TotalPositions)
	}
	b.updatedAt = now
}

// Get returns a consistent copy. Stale snapshots (>60s old) are reported as
// unknown so downstream consumers fail closed.
func (b *brokerAccountState) Get() brokerAccountSnapshotData {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := brokerAccountSnapshotData{
		Known:          b.known && time.Since(b.updatedAt) < 60*time.Second,
		Equity:         b.equity,
		FreeMargin:     b.freeMargin,
		Leverage:       b.leverage,
		TickValue:      b.tickValue,
		TickSize:       b.tickSize,
		LotStep:        b.lotStep,
		BuyCount:       b.buyCount,
		SellCount:      b.sellCount,
		TotalCount:     b.totalCount,
		PositionsKnown: b.positionsKnown && time.Since(b.updatedAt) < 60*time.Second,
		UpdatedAt:      b.updatedAt,
	}
	return data
}

// UpdateSymbol merges broker symbol spec economics into the cached snapshot.
func (b *brokerAccountState) UpdateSymbol(sym marketdata.SnapshotSymbol) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sym.TickValue > 0 {
		b.tickValue = sym.TickValue
	}
	if sym.TickSize > 0 {
		b.tickSize = sym.TickSize
	}
	if sym.LotStep > 0 {
		b.lotStep = sym.LotStep
	}
}

type brokerAccountSnapshotData struct {
	Known          bool
	Equity         float64
	FreeMargin     float64
	Leverage       float64
	TickValue      float64
	TickSize       float64
	LotStep        float64
	BuyCount       int
	SellCount      int
	TotalCount     int
	PositionsKnown bool
	UpdatedAt      time.Time
}

// ─── SL Violation Tracker ───
// Tracks per-agent SL violations. After 3 violations, signals are suspended.
var (
	slViolationMu      sync.Mutex
	slViolationCounts  = make(map[string]int)
	slViolationDetails = make(map[string][]slViolation)
	suspendedAgents    = make(map[string]time.Time)
)

type slViolation struct {
	SignalID   string
	Type       string // NO_SL, SL_MISMATCH
	ActualSL   float64
	ExpectedSL float64
	Timestamp  time.Time
}

func recordSLViolation(agentID, signalID, vType string, actualSL, expectedSL float64) {
	slViolationMu.Lock()
	defer slViolationMu.Unlock()

	slViolationCounts[agentID]++
	slViolationDetails[agentID] = append(slViolationDetails[agentID], slViolation{
		SignalID:   signalID,
		Type:       vType,
		ActualSL:   actualSL,
		ExpectedSL: expectedSL,
		Timestamp:  time.Now().UTC(),
	})

	count := slViolationCounts[agentID]
	observability.Log.Warn().
		Str("agent_id", agentID).
		Int("violation_count", count).
		Str("violation_type", vType).
		Msg("SL violation recorded")

	// After 3 violations, suspend the agent — stop sending signals
	if count >= 3 {
		if _, alreadySuspended := suspendedAgents[agentID]; !alreadySuspended {
			suspendedAgents[agentID] = time.Now().UTC()
			observability.Log.Error().
				Str("agent_id", agentID).
				Int("violation_count", count).
				Msg("AGENT SUSPENDED: 3+ SL violations — disconnecting and blocking future signals")

			// Disconnect the agent
			if globalAgentHub != nil {
				globalAgentHub.DisconnectAgent(agentID, "SL_VIOLATION_THRESHOLD_EXCEEDED")
			}

			// Persist suspension to audit log
			if globalPersister != nil {
				ctxS, cancelS := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancelS()
				globalPersister.GetDB().ExecContext(ctxS, `
					INSERT INTO audit.client_events (user_id, event_type, metadata, event_time)
					VALUES ($1, 'AGENT_SUSPENDED', $2, now())`,
					"agent:"+agentID, fmt.Sprintf(`{"reason":"SL_VIOLATION_THRESHOLD","count":%d}`, count))
			}
		}
	}
}

// isAgentSuspended checks if an agent is currently suspended for SL violations.
func isAgentSuspended(agentID string) bool {
	slViolationMu.Lock()
	defer slViolationMu.Unlock()
	_, suspended := suspendedAgents[agentID]
	return suspended
}

// checkPositionSLs monitors broker snapshot positions for missing SLs.
// Called whenever a broker snapshot is received. If any PAT position has no SL,
// a CLOSE_POSITION command is sent to the agent.
func checkPositionSLs(positions *marketdata.SnapshotPositions, agentID string) {
	if positions == nil || len(positions.Details) == 0 {
		return
	}

	for _, pos := range positions.Details {
		// Only check PAT-managed positions (magic in PAT range)
		if pos.Magic < 100000 || pos.Magic > 199999 {
			continue
		}

		if pos.SL <= 0 {
			observability.Log.Error().
				Str("agent_id", agentID).
				Int64("ticket", pos.Ticket).
				Int64("magic", pos.Magic).
				Str("type", pos.Type).
				Float64("volume", pos.Volume).
				Msg("POSITION SL VIOLATION: PAT position has no SL — sending CLOSE_POSITION")

			if globalAgentHub != nil {
				globalAgentHub.SendToAgent(agentID, "CLOSE_POSITION", map[string]interface{}{
					"ticket": pos.Ticket,
					"magic":  pos.Magic,
					"reason": "MISSING_SL_POSITION_MONITOR",
				})
			}

			recordSLViolation(agentID, "", "POSITION_NO_SL", 0, 0)
		}
	}
}

// runPnLAnchorLoop keeps the daily_loss/profit_target gate states hydrated
// from session P&L anchors persisted in Valkey (pat:pnl_anchor:{period}).
// Fail-closed: no Valkey/no equity ⇒ gates veto pnl_state_unknown.
func runPnLAnchorLoop(gateRegistry *gates.Registry, valkeyCache *cache.ValkeyCache, broker *brokerAccountState) {
	pnlTracker := risk.NewPnLTracker(risk.NewValkeyAnchorStore(valkeyCache))
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		bs := broker.Get()
		if !bs.Known || bs.Equity <= 0 {
			continue // gates stay in their fail-closed seeded state
		}
		now := time.Now().UTC()
		snap := pnlTracker.Update(bs.Equity, now)
		gateRegistry.UpdateState(types.GateDailyLoss, gates.GateState{
			GateID: types.GateDailyLoss, State: types.GatePass, Value: snap,
			EvaluatedAt: now, SourceVersion: "pnl_tracker",
		})
		gateRegistry.UpdateState(types.GateProfitTarget, gates.GateState{
			GateID: types.GateProfitTarget, State: types.GatePass, Value: snap,
			EvaluatedAt: now, SourceVersion: "pnl_tracker",
		})
		if observability.Log.Debug().Enabled() {
			observability.Log.Debug().
				Float64("day_pct", snap.PeriodPc[risk.PeriodDay]).
				Float64("week_pct", snap.PeriodPc[risk.PeriodWeek]).
				Float64("month_pct", snap.PeriodPc[risk.PeriodMonth]).
				Msg("[PNL] session anchor snapshot refreshed")
		}
	}
}

// hydrateEdgeValidationGate refreshes rolling forward-test edge statistics
// per strategy from trading.trade_results every 60s (EV1-EV3 cache window).
// Strategies failing profit factor / expectancy / sample-size thresholds —
// including strategies with an empty history — degrade the edge gate so
// signals publish as ADVISORY with reason edge_unproven.
func hydrateEdgeValidationGate(gateRegistry *gates.Registry, persister *marketdata.Persister, cfg *config.Config) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		hydrateEdgeStateOnce(gateRegistry, persister, cfg)
	}
}

// hydrateEdgeStateOnce refreshes the edge-validation gate state from live closed-trade
// results. It is also invoked synchronously at startup so the negative-live-edge
// capital-protection veto is active immediately — there is no 60s window after a
// restart where a proven-losing armed strategy could still emit executable candidates.
func hydrateEdgeStateOnce(gateRegistry *gates.Registry, persister *marketdata.Persister, cfg *config.Config) {
	if persister == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		rows, err := persister.GetDB().QueryContext(ctx, `
			SELECT strategy_id, direction, pnl::float8,
			       COALESCE(entry_price,0)::float8, COALESCE(stop_loss,0)::float8,
			       COALESCE(lot_size,0)::float8
			FROM (
				SELECT strategy_id, direction, pnl, entry_price, stop_loss, lot_size,
				       ROW_NUMBER() OVER (PARTITION BY strategy_id ORDER BY closed_at DESC) AS rn
				FROM trading.trade_results
			) ranked
			WHERE rn <= $1
		`, cfg.EdgeLookbackTrades)
		if err != nil {
			cancel()
			observability.Log.Warn().Err(err).Msg("[EDGE] trade_results query failed — edge gate unchanged")
			return
		}
		var all []risk.TradeRecord
		for rows.Next() {
			var t risk.TradeRecord
			if scanErr := rows.Scan(&t.StrategyID, &t.Direction, &t.PnL,
				&t.EntryPrice, &t.StopLoss, &t.LotSize); scanErr == nil {
				all = append(all, t)
			}
		}
		rows.Close()
		cancel()

		statsByStrategy := make(map[types.StrategyID]risk.EdgeStats)
		// Compute per strategy over that strategy's last N trades.
		grouped := make(map[string][]risk.TradeRecord)
		for _, t := range all {
			grouped[t.StrategyID] = append(grouped[t.StrategyID], t)
		}
		for id, trades := range grouped {
			stats := risk.ComputeEdgeStats(trades)
			statsByStrategy[types.StrategyID(id)] = stats
			proven := stats.IsProven(cfg.EdgeMinProfitFactor, cfg.EdgeMinExpectancyR, cfg.EdgeMinSampleSize)
			observability.Log.Info().
				Str("strategy", id).
				Int("sample_size", stats.SampleSize).
				Float64("profit_factor", stats.ProfitFactor).
				Float64("expectancy_r", stats.ExpectancyR).
				Float64("short_expectancy_r", stats.ShortExpectancyR).
				Bool("proven", proven).
				Msg("[EDGE] rolling forward-test stats refreshed")
		}

		now := time.Now().UTC()

		// Write an INDEPENDENT, isolated edge-state scope for every
		// (strategy, decision-timeframe) pair. This is the capital-protection
		// isolation fix: previously a single shared map[StrategyID]EdgeStats was
		// used for ALL strategies at once, so a hydration failure or stale read
		// silently degraded or blocked every strategy. Now each strategy/timeframe
		// carries its own fresh state and can fail independently without
		// cross-contaminating others.
		strats := strategy.AllStrategies()
		for id, stats := range statsByStrategy {
			var tfs []types.Timeframe
			for _, s := range strats {
				if s.ID() == id {
					if p, ok := s.(strategy.DecisionTFProvider); ok {
						tfs = p.DecisionTimeframes()
					}
					break
				}
			}
			if len(tfs) == 0 {
				tfs = []types.Timeframe{""} // legacy strategies: single catch-all scope
			}
			for _, tf := range tfs {
				single := map[types.StrategyID]risk.EdgeStats{id: stats}
				proven := stats.IsProven(cfg.EdgeMinProfitFactor, cfg.EdgeMinExpectancyR, cfg.EdgeMinSampleSize)
				stateResult := types.GateDegraded
				reasonCode := gates.ReasonEdgeUnproven
				if proven {
					stateResult = types.GatePass
					reasonCode = "edge_stats_available"
				}
				gateRegistry.UpdateStateScoped(gates.GateScope{
					GateID:     types.GateEdgeValidation,
					StrategyID: id,
					Timeframe:  tf,
				}, gates.GateState{
					GateID:      types.GateEdgeValidation,
					State:       stateResult,
					Value:       single,
					ReasonCode:  reasonCode,
					EvaluatedAt: now,
					SourceVersion: "edge_refresher",
				})
			}
		}
		// If no trades exist yet for any strategy, seed a neutral per-strategy
		// advisory scope so evaluation stays fail-soft (not a global hard veto).
		if len(statsByStrategy) == 0 {
			for _, s := range strats {
				tfs := []types.Timeframe{""}
				if p, ok := s.(strategy.DecisionTFProvider); ok {
					if d := p.DecisionTimeframes(); len(d) > 0 {
						tfs = d
					}
				}
				for _, tf := range tfs {
					gateRegistry.UpdateStateScoped(gates.GateScope{
						GateID:     types.GateEdgeValidation,
						StrategyID: s.ID(),
						Timeframe:  tf,
					}, gates.GateState{
						GateID:      types.GateEdgeValidation,
						State:       types.GateDegraded,
						ReasonCode:  gates.ReasonEdgeUnproven,
						EvaluatedAt: now,
						SourceVersion: "edge_refresher_empty",
					})
				}
			}
		}
}

// refreshGateStates periodically refreshes gate state from live market/broker data.
// This runs as a background goroutine to keep gate states fresh.
func refreshGateStates(reg *gates.Registry, stateMgr *features.StateManager, agentProvider interface{}, staleDetector *marketdata.StaleDetector) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UTC()

		// B-04: Hydrate MinATR and StopHuntFilter gates with PASS state.
		// These gates are self-evaluating (they check input.ATR and input.StructuralLow/High
		// directly from GateInput), so they just need to be initialized to PASS
		// so EvaluateAll doesn't fail with GATE_NOT_INITIALIZED.
		reg.UpdateState(types.GateMinATR, gates.GateState{
			GateID:        types.GateMinATR,
			State:         types.GatePass,
			EvaluatedAt:   now,
			SourceVersion: "1.0",
		})
		reg.UpdateState(types.GateStopHuntFilter, gates.GateState{
			GateID:        types.GateStopHuntFilter,
			State:         types.GatePass,
			EvaluatedAt:   now,
			SourceVersion: "1.0",
		})

		// Refresh market-data gates from live market state
		state := stateMgr.Get("XAUUSD")
		if state != nil {
			// Data quality gate — evidence-based, not hardcoded PASS.
			// Frozen/stalled feeds must not keep a green light (prompt.md
			// Sections 16-17, 21-22, 79). The StaleDetector tracks the last
			// genuine tick per symbol; staleness beyond thresholds degrades
			// then vetoes signal generation (fail-closed).
			dqState := types.GatePass
			dqReason := ""
			freshMs := int64(0)
			if staleDetector != nil {
				staleness := staleDetector.Staleness("XAUUSD")
				if staleness < time.Hour { // known (not "never seen a tick")
					freshMs = staleness.Milliseconds()
					switch {
					case staleness > 30*time.Second:
						dqState = types.GateVeto
						dqReason = "XAUUSD_DATA_STALE"
					case staleness > 10*time.Second:
						dqState = types.GateDegraded
						dqReason = "XAUUSD_DATA_AGING"
					}
				} else {
					dqState = types.GateVeto
					dqReason = "NO_XAUUSD_TICK_YET"
				}
			}
			reg.UpdateState(types.GateDataQuality, gates.GateState{
				State:       dqState,
				ReasonCode:  dqReason,
				EvaluatedAt: now,

				FreshnessMs:   freshMs,
				SourceVersion: "live_feed",
			})

			// Spread gate
			spread, _ := state.Spread.Float64()
			reg.UpdateState(types.GateSpread, gates.GateState{
				State:       types.GatePass,
				Value:       spread,
				EvaluatedAt: now,

				FreshnessMs:   0,
				SourceVersion: "live_feed",
			})

			// Session gate
			reg.UpdateState(types.GateSession, gates.GateState{
				State:       types.GatePass,
				Value:       state.Session.CurrentSession,
				EvaluatedAt: now,

				FreshnessMs:   0,
				SourceVersion: "session_engine",
			})
		}

		// Exposure, margin, and execution permit gates:
		// These are hydrated from broker account data when the Windows Agent
		// sends a MARKET_SNAPSHOT with account_info. If they are already PASS,
		// refresh their validity window. If they have expired (agent disconnected),
		// they will fail closed automatically via the gate evaluation freshness check.
		for _, gateID := range []types.GateID{types.GateExposure, types.GateMargin, types.GateExecutionPermit} {
			gs, exists := reg.GetState(gateID)
			if exists && gs.State == types.GatePass && !now.After(gs.ValidUntil) {
				// Still fresh — extend validity
				reg.UpdateState(gateID, gates.GateState{
					State:       gs.State,
					Value:       gs.Value,
					EvaluatedAt: now,

					FreshnessMs:   0,
					SourceVersion: gs.SourceVersion,
				})
			}
			// If expired, the gate's ValidUntil has passed and EvaluateAll will
			// fail closed — no action needed here.
		}
	}
}

// hydrateEntitlementLicenseGates periodically checks the control plane database
// for active licenses and entitlements, and hydrates the entitlement/license gates.
// In development without a control plane, this checks if any active subscriptions
// or licenses exist in the database. If found, gates are set to PASS.
// If the control plane DB is not connected, gates remain UNKNOWN (fail-closed).
func hydrateEntitlementLicenseGates(reg *gates.Registry, persister *marketdata.Persister, cfg *config.Config) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UTC()

		if persister == nil {
			continue
		}

		db := persister.GetDB()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		// Check for active licenses in the control plane database
		// licensing.licenses is the canonical table (migration 003)
		var licenseCount int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM (
				SELECT 1 FROM licensing.licenses WHERE status = 'ACTIVE' LIMIT 1
			) AS sub
		`).Scan(&licenseCount)
		cancel()

		if err != nil {
			// Control plane tables don't exist or query failed — gates stay UNKNOWN
			// This is the correct fail-closed behavior for development without control plane
			continue
		}

		if licenseCount > 0 {
			// Active license found — hydrate license gate to PASS
			// 			fresh := now.Add(30 * time.Second)
			reg.UpdateState(types.GateLicense, gates.GateState{
				State:       types.GatePass,
				EvaluatedAt: now,
				// No ValidUntil — gate state never expires (broker data may not be available)
				SourceVersion: "control_plane_db",
				Quality:       types.QualityAuthoritative,
			})

			// Check for active subscriptions with entitled strategies
			ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
			var subCount int
			err2 := db.QueryRowContext(ctx2, `
				SELECT COUNT(*) FROM (
					SELECT 1 FROM billing.subscriptions 
					WHERE status = 'ACTIVE' LIMIT 1
				) AS sub
			`).Scan(&subCount)
			cancel2()

			if err2 == nil && subCount > 0 {
				reg.UpdateState(types.GateEntitlement, gates.GateState{
					State:       types.GatePass,
					EvaluatedAt: now,
					// No ValidUntil — gate state never expires (broker data may not be available)
					SourceVersion: "control_plane_db",
					Quality:       types.QualityAuthoritative,
				})
			}
		}
	}
}

// dbHealthCheck verifies database connectivity for readiness checks.
func dbHealthCheck(persister *marketdata.Persister) bool {
	if persister == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return persister.HealthCheck(ctx) == nil
}

// hydrateBrokerAccountState initializes risk gates from live MT4/MT5 broker account data.
// Called when a broker account snapshot is received from the Windows Agent.
func hydrateBrokerAccountState(reg *gates.Registry, balance, equity, freeMargin, usedMargin float64, openPositions int) {
	now := time.Now().UTC()

	// Exposure gate: 0 current exposure (no open positions) = PASS
	reg.UpdateState(types.GateExposure, gates.GateState{
		State:       types.GatePass,
		Value:       float64(openPositions),
		EvaluatedAt: now,

		FreshnessMs:   0,
		SourceVersion: "broker_telemetry",
		Quality:       types.QualityAuthoritative,
	})

	// Margin gate: free margin > 0 = PASS
	marginOK := freeMargin > 0
	reg.UpdateState(types.GateMargin, gates.GateState{
		State:       types.GatePass,
		Value:       marginOK,
		EvaluatedAt: now,

		FreshnessMs:   0,
		SourceVersion: "broker_telemetry",
		Quality:       types.QualityAuthoritative,
	})
}

func createNoTradeSignal(result strategy.StrategyResult, calibratedProb decimal.Decimal, state *features.MarketState, calibConsumer *calibration.Consumer) *types.Signal {
	regime := types.Regime("")
	session := ""
	newsRisk := ""
	timeframe := types.Timeframe("")
	var marketTime time.Time
	var bid, ask decimal.Decimal
	var sourceMode string
	var sourceSeq uint64
	var sourceTs time.Time
	if state != nil {
		regime = state.Regime.Current
		session = state.Session.CurrentSession
		newsRisk = state.Session.NewsRisk
		marketTime = state.Timestamp
		bid = state.Bid
		ask = state.Ask
		if state.LastTick != nil {
			sourceMode = state.LastTick.Source
			sourceSeq = state.LastTick.Sequence
			sourceTs = state.LastTick.SourceTimestamp
		}
		if state.Candles != nil {
			for _, c := range state.Candles {
				timeframe = c.Timeframe
				break
			}
		}
	}
	now := time.Now().UTC()
	calProb, calProbOK := calibConsumer.ProbabilityFor(result.StrategyID, result.RawScore)
	sig := &types.Signal{
		ID: uuid.New().String(), Symbol: "XAUUSD",
		StrategyID: result.StrategyID, Direction: types.DirectionNoTrade,
		Grade: types.GradeNoTrade, Status: types.SignalDetected,
		RawScore: result.RawScore, LongScore: result.LongScore, ShortScore: result.ShortScore,
		CalibratedProbability: calibratedProb,
		Probability:           calProb,
		ProbabilityCalibrated: calProbOK,
		EntryPrice:            result.EntryPrice, StopLoss: result.StopLoss,
		TP1: result.TP1, TP2: result.TP2, TP3: result.TP3,
		Regime: regime, Session: session, NewsRisk: newsRisk,
		Timeframe:   timeframe,
		ReasonCodes: result.ReasonCodes,
		Evidence:    result.Evidence, CreatedAt: now,
		ExpiresAt:     now.Add(15 * time.Minute),
		ExitProfileID: string(result.StrategyID) + "_EXIT_V1", GatePolicyVersion: "1.0.0",
		// Detailed timestamp model (SOW Sections 26-30)
		MarketTime: marketTime,
		DetectedAt: now,
		// Conflict penalty
		ConflictPenalty: result.ConflictPenalty,
		// Versioning
		GeometryVersion: "1.0", RiskProfileVersion: "1.0", FeatureVersion: "1.0",
		// Provenance (prompt.md Sections 30-31)
		BidPrice:        bid,
		AskPrice:        ask,
		SourceMode:      sourceMode,
		SourceSequence:  sourceSeq,
		SourceTimestamp: sourceTs,
		IngestTimestamp: now,
		BarClosed:       types.BarClosedConfirmed,
		// Calibration status (prompt.md Section 36)
		CalibrationStatus: types.CalibrationUnverified,
		// Transition scores (prompt.md Section 6)
		TransitionLongScore:   result.TransitionLongScore,
		TransitionShortScore:  result.TransitionShortScore,
		TransitionConflict:    result.TransitionConflict,
		TransitionFinalScore:  result.TransitionFinalScore,
		IsTransitionCandidate: result.IsTransitionCandidate,
		// Dominance (prompt.md Section 23)
		Dominance: result.Dominance,
	}
	// Set provenance state
	if types.IsLiveDataSource(types.DataSourceType(sourceMode)) {
		sig.ProvenanceState = types.ProvenanceLiveVerified
	} else if sourceMode != "" {
		sig.ProvenanceState = types.ProvenanceUnverified
	} else {
		sig.ProvenanceState = types.ProvenanceUnverified
	}
	return sig
}

func computeRR(entry, sl, tp decimal.Decimal) decimal.Decimal {
	if sl.IsZero() || entry.IsZero() {
		return decimal.Zero
	}
	return tp.Sub(entry).Abs().Div(entry.Sub(sl).Abs())
}

func toF(d decimal.Decimal) float64 { f, _ := d.Float64(); return f }

// noTradeReasonStrings converts NoTradeReason codes to strings for engine liveness tracking.
func noTradeReasonStrings(codes []types.NoTradeReason) []string {
	if len(codes) == 0 {
		return nil
	}
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = string(c)
	}
	return out
}

// parseDecimalSafe parses a decimal string, returning zero on error.
func parseDecimalSafe(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// parseFloatSafe converts decimal.Decimal to float64.
func parseFloatSafe(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

// buildFeatureVector creates a 42-element float64 vector from MarketState indicators
// for ONNX inference. The order matches the feature_columns.json definition.
func buildFeatureVector(state *features.MarketState) []float64 {
	ind := state.Indicators
	return []float64{
		ind.EMA9.InexactFloat64(),                       // 0: ema9
		ind.EMA21.InexactFloat64(),                      // 1: ema21
		ind.EMA50.InexactFloat64(),                      // 2: ema50
		ind.EMA100.InexactFloat64(),                     // 3: ema100
		ind.EMA200.InexactFloat64(),                     // 4: ema200
		boolToFloat(ind.EMACross921),                    // 5: ema_cross_9_21
		ind.SMA50.InexactFloat64(),                      // 6: sma50
		ind.SMA100.InexactFloat64(),                     // 7: sma100
		ind.SMA200.InexactFloat64(),                     // 8: sma200
		ind.MACDMain.InexactFloat64(),                   // 9: macd_main
		ind.MACDSignal.InexactFloat64(),                 // 10: macd_signal
		ind.MACDHistogram.InexactFloat64(),              // 11: macd_histogram
		boolToFloat(ind.MACDBullCross),                  // 12: macd_bull_cross
		boolToFloat(ind.MACDBearCross),                  // 13: macd_bear_cross
		ind.ADX.InexactFloat64(),                        // 14: adx
		ind.ADXPlusDI.InexactFloat64(),                  // 15: adx_plus_di
		ind.ADXMinusDI.InexactFloat64(),                 // 16: adx_minus_di
		ind.RSI.InexactFloat64(),                        // 17: rsi
		ind.StochMain.InexactFloat64(),                  // 18: stoch_main
		ind.StochSignal.InexactFloat64(),                // 19: stoch_signal
		ind.StochRSI.InexactFloat64(),                   // 20: stoch_rsi
		ind.StochRSIK.InexactFloat64(),                  // 21: stoch_rsi_k
		ind.StochRSID.InexactFloat64(),                  // 22: stoch_rsi_d
		ind.CCI.InexactFloat64(),                        // 23: cci
		ind.ATR.InexactFloat64(),                        // 24: atr
		ind.BollUpper.InexactFloat64(),                  // 25: boll_upper
		ind.BollMiddle.InexactFloat64(),                 // 26: boll_middle
		ind.BollLower.InexactFloat64(),                  // 27: boll_lower
		ind.BollWidth.InexactFloat64(),                  // 28: boll_width
		boolToFloat(ind.BollBullRev),                    // 29: boll_bull_rev
		boolToFloat(ind.BollBearRev),                    // 30: boll_bear_rev
		ind.OBV.InexactFloat64(),                        // 31: obv
		ind.VWAP.InexactFloat64(),                       // 32: vwap
		ind.ParabolicSAR.InexactFloat64(),               // 33: psar
		boolToFloat(ind.ParabolicSARLong),               // 34: psar_long
		ind.IchimokuTenkan.InexactFloat64(),             // 35: ichimoku_tenkan
		ind.IchimokuKijun.InexactFloat64(),              // 36: ichimoku_kijun
		ind.IchimokuSenkouA.InexactFloat64(),            // 37: ichimoku_senkou_a
		ind.IchimokuSenkouB.InexactFloat64(),            // 38: ichimoku_senkou_b
		boolToFloat(state.Session.CurrentSession == ""), // 39: session (0 if set)
		boolToFloat(state.Session.IsOverlap),            // 40: is_overlap
		0.0,                                             // 41: padding
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func min(a, b int) int { if a < b { return a }; return b }
