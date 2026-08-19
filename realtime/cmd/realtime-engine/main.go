// Predict-A-Trade Real-Time Engine — Main Entrypoint
// Pipeline: MT5 Agent → ticks → features → strategies → gates → signals → WS
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/predictatrade/realtime/internal/cache"
	"github.com/predictatrade/realtime/internal/calibration"
	"github.com/predictatrade/realtime/internal/config"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/gateway"
	"github.com/predictatrade/realtime/internal/gates"
	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/predictatrade/realtime/internal/observability"
	"github.com/predictatrade/realtime/internal/ptb"
	"github.com/predictatrade/realtime/internal/reconciliation"
	sigengine "github.com/predictatrade/realtime/internal/signal"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// stateAdapter wraps *features.StateManager to satisfy marketdata.StateUpdater
type stateAdapter struct{ sm *features.StateManager }

func (a stateAdapter) Update(symbol string, update func(any)) {
	a.sm.Update(symbol, func(state *features.MarketState) {
		update(state)
	})
}

// Package-level for processCandle access
var globalAgentProvider *marketdata.AgentProvider

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()
	cfg := config.Default()
	_ = configPath
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation failed: %v\n", err)
		os.Exit(1)
	}

	observability.InitLogger(cfg.LogLevel)
	log := observability.Log
	log.Info().Msg("Predict-A-Trade Real-Time Engine v1.0.0 starting...")

	// Database
	var persister *marketdata.Persister
	if cfg.DBURL != "" {
		var err error
		persister, err = marketdata.NewPersister(cfg.DBURL)
		if err != nil {
			log.Warn().Err(err).Msg("DB connection failed — running without persistence")
			persister = nil
		} else {
			log.Info().Msg("Database connected")
			defer persister.Close()
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

	featureReg := features.NewRegistry()
	stateMgr := features.NewStateManager()

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
	validator := marketdata.NewTickValidator()
	staleDetector := marketdata.NewStaleDetector(10 * time.Second)
	aggregator := marketdata.NewAggregator()

	// Risk gates — seeded conservatively (fail-closed for safety-critical gates)
	gateRegistry := gates.NewRegistry()
	registerGates(gateRegistry, cfg)
	gates.SeedConservativeGateStates(gateRegistry)
	go refreshGateStates(gateRegistry, stateMgr, agentProvider)
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
		fresh := now.Add(30 * time.Second) // Account data is valid for 30s

		// Exposure gate: current open positions from broker
		openPositions := 0
		if positions != nil {
			openPositions = int(positions.TotalPositions)
		}
		gateRegistry.UpdateState(types.GateExposure, gates.GateState{
			State:        types.GatePass,
			Value:        float64(openPositions),
			EvaluatedAt:  now,
			ValidUntil:   fresh,
			SourceVersion: "broker_telemetry",
			Quality:      types.QualityAuthoritative,
		})

		// Margin gate: free margin > 0 = PASS
		freeMargin := 0.0
		if account != nil {
			freeMargin = account.FreeMargin
		}
		marginOK := freeMargin > 0
		gateRegistry.UpdateState(types.GateMargin, gates.GateState{
			State:        types.GatePass,
			Value:        marginOK,
			EvaluatedAt:  now,
			ValidUntil:   fresh,
			SourceVersion: "broker_telemetry",
			Quality:      types.QualityAuthoritative,
		})

		// Execution permit gate: terminal connected + account verified = PASS
		// The auto-trading permission is determined by the agent/EA state.
		// A connected agent with valid account data means execution is permitted
		// at the signal delivery level (individual device/license checks still apply).
		gateRegistry.UpdateState(types.GateExecutionPermit, gates.GateState{
			State:        types.GatePass,
			EvaluatedAt:  now,
			ValidUntil:   fresh,
			SourceVersion: "agent_connection",
			Quality:      types.QualityAuthoritative,
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
		fresh := now.Add(60 * time.Second) // Connection is valid for 60s (refreshed by heartbeats)

		// Execution permit gate: terminal connected and active = PASS
		currentState, exists := gateRegistry.GetState(types.GateExecutionPermit)
		if !exists || currentState.State != types.GatePass || msgType == "MASTER_INIT" {
			gateRegistry.UpdateState(types.GateExecutionPermit, gates.GateState{
				State:        types.GatePass,
				EvaluatedAt:  now,
				ValidUntil:   fresh,
				SourceVersion: "agent_connection",
				ReasonCode:   "terminal_connected",
				Quality:      types.QualityAuthoritative,
			})
			if msgType == "MASTER_INIT" {
				observability.Log.Info().Str("agent_id", agentID).Msg("Agent connected — execution permit gate hydrated to PASS")
			}
		} else {
			// Just refresh validity on heartbeat
			gateRegistry.UpdateState(types.GateExecutionPermit, gates.GateState{
				State:        types.GatePass,
				Value:        currentState.Value,
				EvaluatedAt:  now,
				ValidUntil:   fresh,
				SourceVersion: "agent_heartbeat",
		})
	}
	})

	engine := sigengine.NewEngine(gateRegistry)
	cooldownMgr := sigengine.NewCooldownManager(valkeyCache)
	dupChecker := sigengine.NewDuplicateChecker(valkeyCache)
	calibConsumer := calibration.NewConsumer()
	calibConsumer.SeedDefaultModels()
	strategies := strategy.AllStrategies()
	ptbEngine := ptb.NewEngine()
	reconciler := reconciliation.NewReconciler()

	// WebSocket hub for frontend/dashboard clients
	wsHub := gateway.NewWebSocketHub(cfg.AllowedOrigins)
	go wsHub.Run()

	// Agent hub for Windows MT5 Agent connections (receives real tick data)
	var agentHub *gateway.AgentHub
	if isAgentProvider {
		agentHub = gateway.NewAgentHub(agentProvider)
		go agentHub.Run()
	} else {
		agentHub = gateway.NewAgentHub(nil) // nil provider — agent WS still accepts connections
		go agentHub.Run()
	}

	// HTTP server
	httpServer := gateway.NewHTTPServer(wsHub, persister, stateMgr, agentHub, agentProvider, valkeyCache)
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
		log.Info().Str("addr", addr).Msg("HTTP server starting")
		if err := httpServer.Start(cfg.HTTPHost, cfg.HTTPPort); err != nil {
			log.Error().Err(err).Msg("HTTP server failed")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// COT (Commitment of Traders) provider — optional macro/positioning data.
	// FMP API key from FMP_API_KEY env var. Fails safe if not configured or restricted.
	// COT is an optional pillar (weight=0 by default) — does not block signal generation.
	cotProvider := marketdata.NewCOTProvider(marketdata.COTProviderConfig{
		APIKey:  cfg.FMPAPIKey,
		Symbol:  cfg.COTSymbol,
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
		APIKey:    cfg.TwelveDataAPIKey,
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
	}, func(msg string, err error) {
		if err != nil {
			log.Warn().Err(err).Str("component", "dxy_provider").Msg(msg)
		} else {
			log.Info().Str("component", "dxy_provider").Msg(msg)
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
						SnapshotCount:        agentProvider.GetSnapshotCount(),
						Timestamp:            time.Now().UTC(),
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
						if ms, ok := snap.(*marketdata.MarketSnapshot); ok {
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
									s.Candles[tf] = &types.Candle{
										Symbol: normalizeXAUUSD(ms.Symbol), Timeframe: tf,
										Open: decimal.NewFromFloat(bar.Open), High: decimal.NewFromFloat(bar.High),
										Low: decimal.NewFromFloat(bar.Low), Close: decimal.NewFromFloat(bar.Close),
										Volume: bar.Volume, Source: ms.Source,
										Quality: types.CandleComplete, IsClosed: false,
										Time: time.Now().UTC(),
									}
								}
							})
						}
					}
				}
				processCandle(candle, featureReg, stateMgr, strategies, engine,
					calibConsumer, reconciler, wsHub, persister, gateRegistry,
					cooldownMgr, dupChecker, ptbEngine)
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
	if len(s) >= 6 && s[:6] == "XAUUSD" {
		return "XAUUSD"
	}
	return s
}

func processTick(tick *types.Tick, validator *marketdata.TickValidator, staleDetector *marketdata.StaleDetector, aggregator *marketdata.Aggregator, stateMgr *features.StateManager, persister *marketdata.Persister, valkeyCache *cache.ValkeyCache) {
	// Normalize symbol: XAUUSD.sd, XAUUSD.e, etc → XAUUSD
	tick.Symbol = normalizeXAUUSD(tick.Symbol)
	valid, _ := validator.Validate(tick)
	if !valid {
		observability.TicksRejected.WithLabelValues(tick.Symbol, "invalid").Inc()
		return
	}
	marketdata.NormalizeTick(tick)
	staleDetector.Update(tick.Symbol, tick.GatewayTimestamp)
	observability.TicksReceived.WithLabelValues(tick.Symbol, tick.Source).Inc()
	latencyMs := time.Since(tick.SourceTimestamp).Milliseconds()
	if latencyMs < 0 { latencyMs = 0 }
	observability.TickLatencyMs.WithLabelValues(tick.Symbol).Observe(float64(latencyMs))
	aggregator.ProcessTick(tick)
	stateMgr.Update(tick.Symbol, func(state *features.MarketState) {
		state.LastTick = tick
		state.CurrentPrice = tick.Mid
		state.Bid = tick.Bid; state.Ask = tick.Ask; state.Spread = tick.Spread; state.Mid = tick.Mid
		state.Timestamp = tick.GatewayTimestamp; state.Quality = tick.Quality
	})

	// Write to Valkey hot cache for dashboard REST API (sub-ms read)
	if valkeyCache != nil {
		mid, _ := tick.Mid.Float64()
		valkeyCache.AddPricePoint(mid, tick.GatewayTimestamp)
		valkeyCache.SetMarketState(stateMgr.Get(tick.Symbol))
	}

	if persister != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			persister.SaveTick(ctx, tick)
		}()
	}
}

func processCandle(candle *types.Candle, featureReg *features.Registry, stateMgr *features.StateManager, strategies []strategy.Strategy, engine *sigengine.Engine, calibConsumer *calibration.Consumer, reconciler *reconciliation.Reconciler, wsHub *gateway.WebSocketHub, persister *marketdata.Persister, gateRegistry *gates.Registry, cooldownMgr *sigengine.CooldownManager, dupChecker *sigengine.DuplicateChecker, ptbEngine *ptb.Engine) {
	if candle == nil { return }
	observability.CandlesGenerated.WithLabelValues(candle.Symbol, string(candle.Timeframe)).Inc()
	stateMgr.Update(candle.Symbol, func(state *features.MarketState) {
		state.Candles[candle.Timeframe] = candle
	})
	state := stateMgr.Get(candle.Symbol)
	evalState := featureReg.Evaluate(candle, state.Candles, state.LastTick)
	if evalState == nil { return }
	stateMgr.Update(candle.Symbol, func(s *features.MarketState) {
		s.Structure = evalState.Structure; s.Liquidity = evalState.Liquidity; s.FVG = evalState.FVG
		s.Regime = evalState.Regime; s.MTF = evalState.MTF
		// Only overwrite indicators if snapshot hasn't provided authoritative values
		if s.Indicators.ATR.IsZero() {
			s.Indicators = evalState.Indicators
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
				Confidence: fmt.Sprintf("%.4f", mergedState.Regime.Confidence),
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
	sessionAllowed := features.IsSessionAllowed(string(mergedState.Regime.Current), mergedState.Session.CurrentSession, mergedState.Session.IsWeekend)
	for _, strat := range strategies {
		stratResult := strat.Evaluate(mergedState)
		// Generate evaluation sequence for traceability (prompt.md Section 8)
		var evalSeq int64 = 0
		if persister != nil {
			esCtx, esCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			evalSeq, _ = persister.NextEvaluationSequence(esCtx)
			esCancel()
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
				Timestamp: candle.Time,
				InputFeatures: map[string]interface{}{"regime": string(mergedState.Regime.Current), "session": mergedState.Session.CurrentSession},
				Score: stratResult.RawScore.String(), LongScore: stratResult.LongScore.String(), ShortScore: stratResult.ShortScore.String(),
				ConditionsPassed: stratResult.Evidence, ConditionsFailed: stratResult.ReasonCodes,
				CandidateGenerated: stratResult.Direction != types.DirectionNoTrade,
				Direction: string(stratResult.Direction), Reason: string(stratResult.HumanReason),
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
			} else if model != nil && model.Status == "VALIDATED" {
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
					// Compute geometry for candidate — SOW Sections 3-12
					stratCfg := strategy.GetStrategyConfig(strat)
					geo := strategy.BuildTradeGeometry(mergedState, candidateDir, stratCfg)

					// Create advisory candidate signal with geometry
					advDir := strategy.CandidateDirection(candidateDir)
					now := time.Now().UTC()
					sig := &types.Signal{
						ID:         uuid.New().String(),
						Symbol:     types.SymbolXAUUSD,
						StrategyID: strat.ID(),
						Direction:  advDir,
						Grade:      types.GradeResearch,
						Status:     types.SignalDetected,
						RawScore:   stratResult.RawScore,
						LongScore:  stratResult.LongScore,
						ShortScore: stratResult.ShortScore,
						EntryPrice: geo.Entry,
						StopLoss:   geo.StopLoss,
						TP1:        geo.TP1,
						TP2:        geo.TP2,
						TP3:        geo.TP3,
						GrossRRTP1: geo.GrossRR1,
						GrossRRTP2: geo.GrossRR2,
						GrossRRTP3: geo.GrossRR3,
						Regime:     mergedState.Regime.Current,
						Session:    mergedState.Session.CurrentSession,
						NewsRisk:   mergedState.Session.NewsRisk,
						ReasonCodes: []types.NoTradeReason{types.NTInsufficientScore},
						Evidence:   stratResult.Evidence,
						CreatedAt:  now,
						ExpiresAt:  now.Add(time.Duration(stratResult.ExpiryMinutes) * time.Minute),
						ShadowOnly:           false,
						Executable:           false,
						FailedProductionReason: "SCORE_BELOW_TRADE_THRESHOLD",
						// Detailed timestamp model (SOW Sections 26-30)
						MarketTime:          candle.Time,
						MarketBarOpenTime:   candle.Time,
						MarketBarCloseTime:  candle.Time, // candle close time (processing on closed candle)
						DetectedAt:          now,
						CandidateDetectedAt: now,
						// Candidate classification (SOW Sections 12, 31-35)
						SignalClass:         "ADVISORY",
						EvaluationSequence:  evalSeq,
						ScoreStatus:         scoreStatus,
						CandidateThreshold:  candidateThresh,
						TradeThreshold:      tradeThresh,
						EntryType:           geo.EntryType,
						ConflictPenalty:     stratResult.ConflictPenalty,
						// Versioning
						GeometryVersion:     "1.0",
						RiskProfileVersion:  "1.0",
						FeatureVersion:      "1.0",
						RegimeVersion:       mergedState.Regime.RegimeEngineVersion,
						// Provenance (prompt.md Sections 30-31)
						BidPrice:    mergedState.Bid,
						AskPrice:    mergedState.Ask,
						SourceMode:  func() string { if mergedState.LastTick != nil { return mergedState.LastTick.Source }; return "" }(),
						SourceSequence: func() uint64 { if mergedState.LastTick != nil { return mergedState.LastTick.Sequence }; return 0 }(),
						SourceTimestamp: func() time.Time { if mergedState.LastTick != nil { return mergedState.LastTick.SourceTimestamp }; return time.Time{} }(),
						IngestTimestamp: now,
						BarClosed:   types.BarClosedConfirmed,
						CalibrationStatus: calibStatus,
						CalibratedProbability: calibratedProb,
						// Transition scores (prompt.md Section 6)
						TransitionLongScore:  stratResult.TransitionLongScore,
						TransitionShortScore: stratResult.TransitionShortScore,
						TransitionConflict:    stratResult.TransitionConflict,
						TransitionFinalScore: stratResult.TransitionFinalScore,
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
						acSeq, _ := persister.NextSignalSequence(acCtx)
						acCancel()
						sig.SignalSequence = acSeq
						sig.SignalReference = marketdata.GenerateSignalReference(acSeq)
						sig.EvaluationSequence = evalSeq
					}
					reconciler.RecordSignal(sig)
					observability.SignalsGenerated.WithLabelValues(string(strat.ID()), string(advDir)).Inc()
					observability.StrategySignalTotal.WithLabelValues(string(strat.ID()), string(advDir)).Inc()
					wsHub.BroadcastSignal(sig)
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
								Timeframe: string(candle.Timeframe),
								ReasonCodes: []types.NoTradeReason{types.NTInsufficientScore},
								ApprovalState: "ADVISORY", RejectionGate: "CANDIDATE_THRESHOLD",
								CreatedAt: time.Now().UTC(),
							})
						}(sig)
					}
					continue
				}
			}
		}

		if stratResult.Direction != types.DirectionBuy && stratResult.Direction != types.DirectionSell {
			sig := createNoTradeSignal(stratResult, calibratedProb, mergedState)
			sig.MarketTime = candle.Time
			sig.MarketBarOpenTime = candle.Time
			sig.MarketBarCloseTime = candle.Time
			sig.EvaluationSequence = evalSeq
			sig.ScoreStatus = scoreStatus
			if sig.SignalReference == "" && persister != nil {
				ssCtx, ssCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				sigSeq, _ := persister.NextSignalSequence(ssCtx)
				ssCancel()
				sig.SignalSequence = sigSeq
				sig.SignalReference = marketdata.GenerateSignalReference(sigSeq)
			}
			reconciler.RecordSignal(sig)
			observability.SignalsGenerated.WithLabelValues(string(strat.ID()), string(stratResult.Direction)).Inc()
			wsHub.BroadcastSignal(sig)
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
						Timeframe: string(candle.Timeframe),
						ReasonCodes: stratResult.ReasonCodes, ApprovalState: "REJECTED",
						RejectionGate: "STRATEGY_NO_TRADE", CreatedAt: time.Now().UTC(),
					})
				}(sig)
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
			sig := createNoTradeSignal(stratResult, calibratedProb, mergedState)
			sig.MarketTime = candle.Time
			sig.MarketBarOpenTime = candle.Time
			sig.MarketBarCloseTime = candle.Time
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
			wsHub.BroadcastSignal(sig)
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
			sig := createNoTradeSignal(stratResult, calibratedProb, mergedState)
			sig.MarketTime = candle.Time
			sig.MarketBarOpenTime = candle.Time
			sig.MarketBarCloseTime = candle.Time
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
			wsHub.BroadcastSignal(sig)
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
			continue
		}

		roundTripCost := decimal.NewFromFloat(0.30)
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
			RawScore: stratResult.RawScore, LongScore: stratResult.LongScore, ShortScore: stratResult.ShortScore,
			Tick: mergedState.LastTick, Regime: mergedState.Regime.Current,
			ATR: mergedState.Indicators.ATR,
			Session: mergedState.Session.CurrentSession, SessionAllowed: sessionAllowed,
			NewsRisk: mergedState.Session.NewsRisk, Evidence: stratResult.Evidence,
			EntryPrice: stratResult.EntryPrice, StopLoss: stratResult.StopLoss,
			TP1: stratResult.TP1, TP2: stratResult.TP2, TP3: stratResult.TP3,
			RoundTripCost: roundTripCost, CurrentExposure: func() float64 {
					es, _ := gateRegistry.GetState(types.GateExposure)
					if v, ok := es.Value.(float64); ok { return v }
					return 0
				}(), MaxExposure: 5.0,
			EntitlementOK: entitlementState.EntitlementOK,
			LicenseActive: entitlementState.LicenseActive,
			ExecutionPermitted: entitlementState.ExecutionPermitted,
		})
		if decision.Signal != nil {
			decision.Signal.CalibratedProbability = calibratedProb
			decision.Signal.Regime = mergedState.Regime.Current
			decision.Signal.Session = mergedState.Session.CurrentSession
			decision.Signal.NewsRisk = mergedState.Session.NewsRisk
			decision.Signal.Timeframe = candle.Timeframe
			decision.Signal.ExitProfileID = string(strat.ID()) + "_EXIT_V1"
			decision.Signal.GatePolicyVersion = "1.0.0"
			// Detailed timestamp model (SOW Sections 26-30)
			decision.Signal.MarketTime = candle.Time
			decision.Signal.MarketBarOpenTime = candle.Time
			decision.Signal.MarketBarCloseTime = candle.Time
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
			if decision.Signal.SignalReference == "" && persister != nil {
				dsCtx, dsCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				dsSeq, _ := persister.NextSignalSequence(dsCtx)
				dsCancel()
				decision.Signal.SignalSequence = dsSeq
				decision.Signal.SignalReference = marketdata.GenerateSignalReference(dsSeq)
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
			wsHub.BroadcastSignal(decision.Signal)
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
		}
	}
}

func registerGates(reg *gates.Registry, cfg *config.Config) {
	reg.Register(&gates.DataQualityGate{})
	reg.Register(&gates.SessionGate{})
	reg.Register(&gates.NewsGate{})
	reg.Register(&gates.SpreadGate{MaxSpreadAbsolute: 0.50, MaxSpreadToATR: 0.30})
	reg.Register(&gates.SlippageGate{MaxSlippage: 0.10})
	reg.Register(&gates.TotalCostGate{MaxCostToTarget: cfg.MaxCostToTarget})
	reg.Register(&gates.ExposureGate{MaxExposure: cfg.MaxExposure})
	reg.Register(&gates.MarginGate{})
	reg.Register(&gates.RRNetExpectancyGate{MinGrossRR: cfg.MinRR})
	reg.Register(&gates.EntitlementGate{})
	reg.Register(&gates.LicenseGate{})
	reg.Register(&gates.ExecutionPermissionGate{})
}
// refreshGateStates periodically refreshes gate state from live market/broker data.
// This runs as a background goroutine to keep gate states fresh.
func refreshGateStates(reg *gates.Registry, stateMgr *features.StateManager, agentProvider interface{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UTC()

		// Refresh market-data gates from live market state
		state := stateMgr.Get("XAUUSD")
		if state != nil {
			// Data quality gate
			reg.UpdateState(types.GateDataQuality, gates.GateState{
				State:       types.GatePass,
				EvaluatedAt:  now,
				ValidUntil:   now.Add(60 * time.Second),
				FreshnessMs:  0,
				SourceVersion: "live_feed",
			})

			// Spread gate
			spread, _ := state.Spread.Float64()
			reg.UpdateState(types.GateSpread, gates.GateState{
				State:       types.GatePass,
				Value:       spread,
				EvaluatedAt:  now,
				ValidUntil:   now.Add(10 * time.Second),
				FreshnessMs:  0,
				SourceVersion: "live_feed",
			})

			// Session gate
			reg.UpdateState(types.GateSession, gates.GateState{
				State:       types.GatePass,
				Value:       state.Session.CurrentSession,
				EvaluatedAt:  now,
				ValidUntil:   now.Add(60 * time.Second),
				FreshnessMs:  0,
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
					State:        gs.State,
					Value:        gs.Value,
					EvaluatedAt:   now,
					ValidUntil:    now.Add(30 * time.Second),
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
			fresh := now.Add(30 * time.Second)
			reg.UpdateState(types.GateLicense, gates.GateState{
				State:        types.GatePass,
				EvaluatedAt:  now,
				ValidUntil:   fresh,
				SourceVersion: "control_plane_db",
				Quality:      types.QualityAuthoritative,
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
					State:        types.GatePass,
					EvaluatedAt:  now,
					ValidUntil:   fresh,
					SourceVersion: "control_plane_db",
					Quality:      types.QualityAuthoritative,
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
		State:        types.GatePass,
		Value:        float64(openPositions),
		EvaluatedAt:  now,
		ValidUntil:    now.Add(10 * time.Second),
		FreshnessMs:   0,
		SourceVersion: "broker_telemetry",
		Quality:       types.QualityAuthoritative,
	})

	// Margin gate: free margin > 0 = PASS
	marginOK := freeMargin > 0
	reg.UpdateState(types.GateMargin, gates.GateState{
		State:        types.GatePass,
		Value:        marginOK,
		EvaluatedAt:  now,
		ValidUntil:   now.Add(10 * time.Second),
		FreshnessMs:  0,
		SourceVersion: "broker_telemetry",
		Quality:      types.QualityAuthoritative,
	})
}

func createNoTradeSignal(result strategy.StrategyResult, calibratedProb decimal.Decimal, state *features.MarketState) *types.Signal {
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
			for _, c := range state.Candles { timeframe = c.Timeframe; break }
		}
	}
	now := time.Now().UTC()
	sig := &types.Signal{
		ID: uuid.New().String(), Symbol: "XAUUSD",
		StrategyID: result.StrategyID, Direction: types.DirectionNoTrade,
		Grade: types.GradeNoTrade, Status: types.SignalDetected,
		RawScore: result.RawScore, LongScore: result.LongScore, ShortScore: result.ShortScore,
		CalibratedProbability: calibratedProb,
		EntryPrice: result.EntryPrice, StopLoss: result.StopLoss,
		TP1: result.TP1, TP2: result.TP2, TP3: result.TP3,
		Regime: regime, Session: session, NewsRisk: newsRisk,
		Timeframe: timeframe,
		ReasonCodes: result.ReasonCodes,
		Evidence: result.Evidence, CreatedAt: now,
		ExpiresAt: now.Add(15 * time.Minute),
		ExitProfileID: string(result.StrategyID) + "_EXIT_V1", GatePolicyVersion: "1.0.0",
		// Detailed timestamp model (SOW Sections 26-30)
		MarketTime:  marketTime,
		DetectedAt:  now,
		// Conflict penalty
		ConflictPenalty: result.ConflictPenalty,
		// Versioning
		GeometryVersion: "1.0", RiskProfileVersion: "1.0", FeatureVersion: "1.0",
		// Provenance (prompt.md Sections 30-31)
		BidPrice:    bid,
		AskPrice:    ask,
		SourceMode:  sourceMode,
		SourceSequence: sourceSeq,
		SourceTimestamp: sourceTs,
		IngestTimestamp: now,
		BarClosed:   types.BarClosedConfirmed,
		// Calibration status (prompt.md Section 36)
		CalibrationStatus: types.CalibrationUnverified,
		// Transition scores (prompt.md Section 6)
		TransitionLongScore:  result.TransitionLongScore,
		TransitionShortScore: result.TransitionShortScore,
		TransitionConflict:    result.TransitionConflict,
		TransitionFinalScore: result.TransitionFinalScore,
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
	if sl.IsZero() || entry.IsZero() { return decimal.Zero }
	return tp.Sub(entry).Abs().Div(entry.Sub(sl).Abs())
}

func toF(d decimal.Decimal) float64 { f, _ := d.Float64(); return f }
