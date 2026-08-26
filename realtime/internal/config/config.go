// Package config provides configuration loading for the realtime engine.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all realtime engine configuration.
type Config struct {
	// Database (internal only — never exposed publicly)
	DBURL string

	// Valkey/Redis (internal only)
	ValkeyAddr string

	// HTTP/WS listeners — bind to 127.0.0.1 only; Nginx is the public ingress
	HTTPHost string
	HTTPPort int
	WSPort   int

	// Market data
	ProviderMode string
	ReplayFile   string
	Symbols      []string
	BasePrice    float64

	// Engine
	TickRateMs     int
	FeatureWorkers int

	// Strategy
	MaxSpreadPips   float64
	MinRR           float64
	MaxCostToTarget float64
	MaxExposure     float64

	// Capital protection (R1-R7, EV1-EV3, PT/P&L)
	EnableShorts              bool               // ENABLE_SHORTS — false suppresses SELL at generation
	MaxRiskPerTradePct        float64            // MAX_RISK_PER_TRADE_PCT (default 1.5% of equity)
	MaxSameDirectionPositions int                // MAX_SAME_DIRECTION_POSITIONS
	MaxTotalPositions         int                // MAX_TOTAL_POSITIONS
	MaxPerStrategyPositions   int                // MAX_PER_STRATEGY_POSITIONS
	MaxDailyLossPct           float64            // MAX_DAILY_LOSS_PCT (negative halt threshold magnitude)
	MaxWeeklyLossPct          float64            // MAX_WEEKLY_LOSS_PCT
	MaxMonthlyLossPct         float64            // MAX_MONTHLY_LOSS_PCT
	MaxDailyProfitPct         float64            // MAX_DAILY_PROFIT_PCT profit lock
	MaxWeeklyProfitPct        float64            // MAX_WEEKLY_PROFIT_PCT profit lock
	MartingaleMaxLotRatio     float64            // MARTINGALE_MAX_LOT_RATIO vs per-strategy base lot
	BaseLots                  map[string]float64 // BASE_LOT_{STRATEGY}
	EdgeMinProfitFactor       float64            // EDGE_MIN_PROFIT_FACTOR
	EdgeMinExpectancyR        float64            // EDGE_MIN_EXPECTANCY_R
	EdgeMinSampleSize         int                // EDGE_MIN_SAMPLE_SIZE
	EdgeLookbackTrades        int                // EDGE_LOOKBACK_TRADES
	// Operator authorization for live auto-trading.
	// LIVE_TRADING_AUTHORIZED must be explicitly set true by an operator. It is
	// the master kill-switch for server-side EXECUTABLE signal emission. Without
	// it, edge_validation/position_caps remain fail-closed regardless of arming.
	LiveTradingAuthorized bool
	// EDGE_ARMED_STRATEGIES is the explicit operator list of strategies that are
	// qualified (backtest/walk-forward calibration on file) to emit EXECUTABLE
	// signals. Empty by default — nothing is armed unless the operator lists it.
	// Arming only takes effect when LiveTradingAuthorized is also true.
	EdgeArmedStrategies []string
	MaxMarginUsagePct         float64            // MAX_MARGIN_USAGE_PCT of free margin
	DefaultLeverage           int                // DEFAULT_LEVERAGE when broker snapshot lacks leverage
	CostToTP1MaxPct           float64            // COST_TO_TP1_MAX_PCT for scalping strategies
	SlippageCostPoints        float64            // SLIPPAGE_COST_POINTS (price units) added to round-trip cost
	CommissionCostPoints      float64            // COMMISSION_COST_POINTS (price units) added to round-trip cost

	// P0-001: Broker symbol metadata validation gate config
	BrokerMinStopPoints   float64 // BROKER_MIN_STOP_POINTS — symbol STOPS_LEVEL (0 = no constraint)
	BrokerMinFreezePoints float64 // BROKER_MIN_FREEZE_POINTS — symbol FREEZE_LEVEL (0 = no constraint)
	BrokerMinLot          float64 // BROKER_MIN_LOT — symbol volume_min (0 = no constraint)
	BrokerMaxLot          float64 // BROKER_MAX_LOT — symbol volume_max (0 = no constraint)
	BrokerLotStep         float64 // BROKER_LOT_STEP — symbol volume_step
	BrokerDigits          int     // BROKER_DIGITS — symbol digits for XAUUSD

	// CORS/origin validation
	AllowedOrigins []string

	// Logging
	LogLevel string

	// COT (Commitment of Traders) — optional macro/positioning data
	FMPAPIKey  string
	COTSymbol  string
	COTEnabled bool

	// DXY (US Dollar Index) — Twelve Data API
	TwelveDataAPIKey string
	DXYEnabled       bool

	// ML Inference Engine
	MLEnabled bool
	ModelsDir string

	// Cross-Market Macro Engine
	CrossMarketMode    string // disabled|shadow|active
	CrossMarketEnabled bool
	EURUSDEnabled      bool
	RealYieldEnabled   bool
	RealYieldProvider  string // fmp|fred|disabled
	FREDAPIKey         string // FRED API key for real yield data
	VIXEnabled         bool
	BTCEnabled         bool
	OilEnabled         bool

	// News / Economic Calendar
	NewsProvider            string // disabled|fmp|...
	NewsMode                string // OFF|PROTECT_ONLY|EVENT_BREAKOUT
	NewsFailPolicy          string // BLOCK_TRADING|ALLOW_TRADING
	NewsSyncIntervalSec     int
	NewsStaleAfterSec       int
	NewsPreBlackoutMinutes  int
	NewsPostBlackoutMinutes int
	NewsMinImpact           string // NONE|LOW|MEDIUM|HIGH|EXTREME
	NewsProviderAPIKey      string

	// News Breakout
	NewsBreakoutEnabled    bool
	NewsBreakoutPrepareSec int
	NewsBreakoutExpirySec  int
	NewsBreakoutEntryATR   float64
	NewsBreakoutMaxSpread  float64
	NewsBreakoutMaxRiskPct float64
	NewsBreakoutSLATR      float64
	NewsBreakoutTPATR      float64

	// Notifications
	NotifyEmailEnabled    bool
	NotifyTelegramEnabled bool
	NotifyPushEnabled     bool
	SMTPHost              string
	SMTPPort              int
	SMTPUsername          string
	SMTPPassword          string
	SMTPFrom              string
	SMTPTLS               bool
	TelegramBotToken      string
	TelegramChatID        string
	WhatsAppServerURL     string `json:"-"`
	WhatsAppToken         string `json:"-"`
	WhatsAppSession       string `json:"-"`
	WhatsAppPhone         string `json:"-"`
	NtfyServerURL         string `json:"-"`
	NtfyTopic             string `json:"-"`
	NtfyAccessToken       string `json:"-"`

	// Ollama (LLM sentiment analysis)
	OllamaEnabled bool
	OllamaHost    string
	OllamaModel   string
	OllamaTimeout string
}

// Default returns a config suitable for local development/testing.
// Production values come from environment variables.
func Default() *Config {
	return &Config{
		DBURL: func() string {
			url := getEnv("DATABASE_URL", "")
			if url == "" {
				url = loadDatabaseURLFromFile()
			}
			return url
		}(),
		ValkeyAddr:      getEnv("VALKEY_ADDR", "127.0.0.1:6379"),
		HTTPHost:        getEnv("HTTP_HOST", "127.0.0.1"),
		HTTPPort:        getEnvInt("HTTP_PORT", 13081),
		WSPort:          getEnvInt("WS_PORT", 13081),
		ProviderMode:    getEnv("PROVIDER_MODE", "agent"),
		ReplayFile:      getEnv("REPLAY_FILE", ""),
		Symbols:         strings.Split(getEnv("SYMBOLS", "XAUUSD"), ","),
		BasePrice:       getEnvFloat("BASE_PRICE", 2430.0),
		TickRateMs:      getEnvInt("TICK_RATE_MS", 500),
		FeatureWorkers:  getEnvInt("FEATURE_WORKERS", 4),
		MaxSpreadPips:   getEnvFloat("MAX_SPREAD_PIPS", 3.0),
		MinRR:           getEnvFloat("MIN_RR", 1.0),
		MaxCostToTarget: getEnvFloat("MAX_COST_TO_TARGET", 0.35),
		MaxExposure:     getEnvFloat("MAX_EXPOSURE", 5.0),

		// Capital protection — fail-safe defaults; nesting validated at startup.
		EnableShorts:              getEnvBoolDefaultTrue("ENABLE_SHORTS"),
		MaxRiskPerTradePct:        getEnvFloat("MAX_RISK_PER_TRADE_PCT", 1.5),
		MaxSameDirectionPositions: getEnvInt("MAX_SAME_DIRECTION_POSITIONS", 1),
		MaxTotalPositions:         getEnvInt("MAX_TOTAL_POSITIONS", 2),
		MaxPerStrategyPositions:   getEnvInt("MAX_PER_STRATEGY_POSITIONS", 1),
		MaxDailyLossPct:           getEnvFloat("MAX_DAILY_LOSS_PCT", 2.0),
		MaxWeeklyLossPct:          getEnvFloat("MAX_WEEKLY_LOSS_PCT", 4.0),
		MaxMonthlyLossPct:         getEnvFloat("MAX_MONTHLY_LOSS_PCT", 5.0),
		MaxDailyProfitPct:         getEnvFloat("MAX_DAILY_PROFIT_PCT", 5.0),
		MaxWeeklyProfitPct:        getEnvFloat("MAX_WEEKLY_PROFIT_PCT", 12.0),
		MartingaleMaxLotRatio:     getEnvFloat("MARTINGALE_MAX_LOT_RATIO", 1.0),
		BaseLots: map[string]float64{
			"STANDARD_SCALPING": getEnvFloat("BASE_LOT_STANDARD_SCALPING", 0.01),
			"ULTRA_SCALPING":    getEnvFloat("BASE_LOT_ULTRA_SCALPING", 0.01),
			"STANDARD_SWING":    getEnvFloat("BASE_LOT_STANDARD_SWING", 0.01),
			"TREND_SWING":       getEnvFloat("BASE_LOT_TREND_SWING", 0.01),
			"MARNIE_FIB":        getEnvFloat("BASE_LOT_MARNIE_FIB", 0.01),
		},
		EdgeMinProfitFactor:  getEnvFloat("EDGE_MIN_PROFIT_FACTOR", 1.2),
		EdgeMinExpectancyR:   getEnvFloat("EDGE_MIN_EXPECTANCY_R", 0.2),
		EdgeMinSampleSize:    getEnvInt("EDGE_MIN_SAMPLE_SIZE", 50),
		EdgeLookbackTrades:   getEnvInt("EDGE_LOOKBACK_TRADES", 50),
		LiveTradingAuthorized: getEnvBool("LIVE_TRADING_AUTHORIZED", false),
		EdgeArmedStrategies:   splitComma(getEnv("EDGE_ARMED_STRATEGIES", "")),
		MaxMarginUsagePct:    getEnvFloat("MAX_MARGIN_USAGE_PCT", 30.0),
		DefaultLeverage:      getEnvInt("DEFAULT_LEVERAGE", 500),
		CostToTP1MaxPct:      getEnvFloat("COST_TO_TP1_MAX_PCT", 0.30),
		SlippageCostPoints:   getEnvFloat("SLIPPAGE_COST_POINTS", 0.10),
		CommissionCostPoints: getEnvFloat("COMMISSION_COST_POINTS", 0.06),
		// P0-001: Broker symbol validation — zero means "no constraint" (gate degrades, not vetoes)
		BrokerMinStopPoints:   getEnvFloat("BROKER_MIN_STOP_POINTS", 0),
		BrokerMinFreezePoints: getEnvFloat("BROKER_MIN_FREEZE_POINTS", 0),
		BrokerMinLot:          getEnvFloat("BROKER_MIN_LOT", 0.01),
		BrokerMaxLot:          getEnvFloat("BROKER_MAX_LOT", 0),
		BrokerLotStep:         getEnvFloat("BROKER_LOT_STEP", 0.01),
		BrokerDigits:          getEnvInt("BROKER_DIGITS", 2),
		AllowedOrigins:       strings.Split(getEnv("ALLOWED_ORIGINS", "https://platform.predictatrade.com,https://predictatrade.com"), ","),
		LogLevel:             getEnv("LOG_LEVEL", "info"),

		// Cross-Market Macro Engine
		CrossMarketMode:    getEnv("CROSS_MARKET_MODE", "shadow"),
		CrossMarketEnabled: getEnvBool("CROSS_MARKET_ENABLED", true),
		EURUSDEnabled:      getEnvBool("EURUSD_ENABLED", true),
		RealYieldEnabled:   getEnvBool("REAL_YIELD_ENABLED", false),
		RealYieldProvider:  getEnv("REAL_YIELD_PROVIDER", "disabled"),
		FREDAPIKey:         getEnv("FRED_API_KEY", ""),
		VIXEnabled:         getEnvBool("VIX_ENABLED", false),
		BTCEnabled:         getEnvBool("BTC_ENABLED", false),
		OilEnabled:         getEnvBool("OIL_ENABLED", false),

		// COT provider — optional, fails safe if not configured
		FMPAPIKey:  getEnv("FMP_API_KEY", ""),
		COTSymbol:  getEnv("COT_SYMBOL", "GC"), // Gold futures
		COTEnabled: getEnvBool("COT_ENABLED", false),

		// DXY provider — Twelve Data API for DXY computation
		TwelveDataAPIKey: getEnv("TWELVEDATA_API_KEY", ""),
		DXYEnabled:       getEnvBool("DXY_ENABLED", false),
		MLEnabled:        getEnvBool("ML_ENABLED", false),
		ModelsDir:        getEnv("MODELS_DIR", "models"),

		// News / Economic Calendar — PROTECT_ONLY by default, disabled provider
		NewsProvider:            getEnv("NEWS_PROVIDER", "disabled"),
		NewsMode:                getEnv("NEWS_MODE", "PROTECT_ONLY"),
		NewsFailPolicy:          getEnv("NEWS_FAIL_POLICY", "BLOCK_TRADING"),
		NewsSyncIntervalSec:     getEnvInt("NEWS_SYNC_INTERVAL_SEC", 300),
		NewsStaleAfterSec:       getEnvInt("NEWS_STALE_AFTER_SEC", 900),
		NewsPreBlackoutMinutes:  getEnvInt("NEWS_PRE_BLACKOUT_MINUTES", 15),
		NewsPostBlackoutMinutes: getEnvInt("NEWS_POST_BLACKOUT_MINUTES", 15),
		NewsMinImpact:           getEnv("NEWS_MIN_IMPACT", "MEDIUM"),
		NewsProviderAPIKey:      getEnv("NEWS_PROVIDER_API_KEY", ""),

		// News Breakout — DISABLED BY DEFAULT
		NewsBreakoutEnabled:    getEnvBool("NEWS_BREAKOUT_ENABLED", false),
		NewsBreakoutPrepareSec: getEnvInt("NEWS_BREAKOUT_PREPARE_SECONDS", 120),
		NewsBreakoutExpirySec:  getEnvInt("NEWS_BREAKOUT_EXPIRY_SECONDS", 300),
		NewsBreakoutEntryATR:   getEnvFloat("NEWS_BREAKOUT_ENTRY_ATR_MULTIPLIER", 0.5),
		NewsBreakoutMaxSpread:  getEnvFloat("NEWS_BREAKOUT_MAX_SPREAD", 3.0),
		NewsBreakoutMaxRiskPct: getEnvFloat("NEWS_BREAKOUT_MAX_RISK_PCT", 1.0),
		NewsBreakoutSLATR:      getEnvFloat("NEWS_BREAKOUT_SL_ATR_MULTIPLIER", 1.0),
		NewsBreakoutTPATR:      getEnvFloat("NEWS_BREAKOUT_TP_ATR_MULTIPLIER", 2.0),

		// Notifications — all DISABLED BY DEFAULT
		NotifyEmailEnabled:    getEnvBool("NOTIFICATION_EMAIL_ENABLED", false),
		NotifyTelegramEnabled: getEnvBool("NOTIFICATION_TELEGRAM_ENABLED", false),
		NotifyPushEnabled:     getEnvBool("NOTIFICATION_PUSH_ENABLED", false),
		SMTPHost:              getEnv("SMTP_HOST", ""),
		SMTPPort:              getEnvInt("SMTP_PORT", 587),
		SMTPUsername:          getEnv("SMTP_USERNAME", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:              getEnv("SMTP_FROM", ""),
		SMTPTLS:               getEnvBool("SMTP_TLS", true),
		TelegramBotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:        getEnv("TELEGRAM_CHAT_ID", ""),
		// Push (self-hosted ntfy)
		NtfyServerURL:   getEnv("NTFY_SERVER_URL", ""),
		NtfyTopic:       getEnv("NTFY_TOPIC", ""),
		NtfyAccessToken: getEnv("NTFY_ACCESS_TOKEN", ""),

		// Ollama (LLM Sentiment Analysis)
		OllamaEnabled: getEnvBool("OLLAMA_ENABLED", false),
		OllamaHost:    getEnv("OLLAMA_HOST", "http://localhost:11434"),
		OllamaModel:   getEnv("OLLAMA_MODEL", "deepseek-v4-pro:cloud"),
		OllamaTimeout: getEnv("OLLAMA_TIMEOUT", "2s"),
	}
}

// IsProduction returns true when the runtime is configured for production.
func (c *Config) IsProduction() bool {
	return os.Getenv("NODE_ENV") == "production" || os.Getenv("APP_ENV") == "production"
}

// KnownInsecureSecrets are placeholder values that must never be accepted in production.
var knownInsecureSecrets = map[string]bool{
	"CHANGE_ME_IN_PRODUCTION":                   true,
	"CHANGE_ME_IN_PRODUCTION_USE_SECRET_FILE":   true,
	"pat_local_dev_secret_change_in_production": true,
	"change_this_to_a_long_random_secret":       true,
	"changeme":                                  true,
	"placeholder":                               true,
	"development":                               true,
	"secret":                                    true,
	"":                                          true,
}

// IsInsecureSecret returns true if the value is empty or a known placeholder.
func IsInsecureSecret(v string) bool {
	return knownInsecureSecrets[v]
}

// KnownInsecureDBPasswords are dev/test credentials that must not be used in production.
var knownInsecureDBPasswords = []string{"pat_local_dev_only", "change_me", "changeme", "password"}

func containsInsecureDBPassword(url string) bool {
	for _, p := range knownInsecureDBPasswords {
		if strings.Contains(url, ":"+p+"@") {
			return true
		}
	}
	return false
}

// loadDatabaseURLFromFile reads DATABASE_URL from the secret file
// /srv/predictatrade/xauusd/database_url.txt (gitignored, chmod 600).
// Used as fallback when DATABASE_URL env var is not set.
func loadDatabaseURLFromFile() string {
	data, err := os.ReadFile("/srv/predictatrade/xauusd/database_url.txt")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (c *Config) Validate() error {
	if c.DBURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.WSPort <= 0 || c.WSPort > 65535 {
		return fmt.Errorf("invalid WS port: %d", c.WSPort)
	}

	// P1-002: simulated provider mode is forbidden in production
	if c.IsProduction() && c.ProviderMode == "simulated" {
		return fmt.Errorf("PROVIDER_MODE=simulated is forbidden in production — use 'agent' for live MT5 data")
	}

	// P2-002: hardcoded/insecure DB credentials are forbidden in production
	// unless DB_ALLOW_INSECURE_DEV=true is explicitly set (for local Docker deployments
	// behind a firewall where the DB is not exposed externally)
	if c.IsProduction() && containsInsecureDBPassword(c.DBURL) {
		if os.Getenv("DB_ALLOW_INSECURE_DEV") != "true" {
			return fmt.Errorf("DATABASE_URL contains an insecure hardcoded password — supply credentials via production secret or set DB_ALLOW_INSECURE_DEV=true for local Docker")
		}
	}

	// Capital-protection nesting validation (R4): loss caps must nest
	// daily < weekly < monthly, and profit locks must be >= loss magnitudes.
	// Unsafe config refuses to start (fail-closed at boot).
	// An entirely-unset block (all zeros — legacy/manual Config literals)
	// skips these checks; any partial configuration is validated strictly.
	capitalConfigured := c.MaxDailyLossPct > 0 || c.MaxWeeklyLossPct > 0 ||
		c.MaxMonthlyLossPct > 0 || c.MaxDailyProfitPct > 0 || c.MaxWeeklyProfitPct > 0 ||
		c.MaxRiskPerTradePct > 0
	if capitalConfigured {
		if c.MaxDailyLossPct <= 0 || c.MaxWeeklyLossPct <= 0 || c.MaxMonthlyLossPct <= 0 {
			return fmt.Errorf("capital protection: MAX_DAILY_LOSS_PCT/MAX_WEEKLY_LOSS_PCT/MAX_MONTHLY_LOSS_PCT must be > 0 (got %v/%v/%v)", c.MaxDailyLossPct, c.MaxWeeklyLossPct, c.MaxMonthlyLossPct)
		}
		if !(c.MaxDailyLossPct < c.MaxWeeklyLossPct && c.MaxWeeklyLossPct < c.MaxMonthlyLossPct) {
			return fmt.Errorf("capital protection: loss caps must nest daily < weekly < monthly (got daily=%v weekly=%v monthly=%v)", c.MaxDailyLossPct, c.MaxWeeklyLossPct, c.MaxMonthlyLossPct)
		}
		if c.MaxDailyProfitPct <= 0 || c.MaxWeeklyProfitPct <= 0 {
			return fmt.Errorf("capital protection: MAX_DAILY_PROFIT_PCT/MAX_WEEKLY_PROFIT_PCT must be > 0 (got %v/%v)", c.MaxDailyProfitPct, c.MaxWeeklyProfitPct)
		}
		if !(c.MaxDailyProfitPct >= c.MaxDailyLossPct && c.MaxWeeklyProfitPct >= c.MaxWeeklyLossPct) {
			return fmt.Errorf("capital protection: profit locks must be >= loss magnitudes (daily %v>=%v, weekly %v>=%v)", c.MaxDailyProfitPct, c.MaxDailyLossPct, c.MaxWeeklyProfitPct, c.MaxWeeklyLossPct)
		}
		if c.MaxRiskPerTradePct <= 0 || c.MaxRiskPerTradePct > 100 {
			return fmt.Errorf("capital protection: MAX_RISK_PER_TRADE_PCT must be in (0,100], got %v", c.MaxRiskPerTradePct)
		}
	}
	if c.MartingaleMaxLotRatio != 0 && c.MartingaleMaxLotRatio < 1.0 {
		return fmt.Errorf("capital protection: MARTINGALE_MAX_LOT_RATIO must be >= 1.0 (anti-martingale), got %v", c.MartingaleMaxLotRatio)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// ─── Advanced configuration sections ───

// AdvancedConfig holds configuration for advanced risk/adaptation/hedging/ML/RL/sentiment.
type AdvancedConfig struct {
	LossRecovery lossRecoveryConfig
	Adaptation   adaptationConfig
	Hedging      hedgingConfig
	MLAdaptation mlAdaptationConfig
	RLStrategy   rlStrategyConfig
	Sentiment    sentimentConfig
}

type lossRecoveryConfig struct {
	Enabled                 bool
	MaxDailyLossPercent     float64
	MaxDailyLossCount       int
	MaxConsecutiveLosses    int
	RecoverySizeMultiplier  float64
	RecoveryMinConfluence   float64
	RecoveryMinSetupGrade   string
	RecoveryMinConfidence   float64
	RecoveryMaxTrades       int
	RecoveryExitAfterWins   int
	NormalCooldownMinutes   int
	RecoveryCooldownMinutes int
	HaltCooldownMinutes     int
}

type adaptationConfig struct {
	Enabled           bool
	MaxRiskMultiplier float64
	GlobalHardMaxRisk float64
}

type hedgingConfig struct {
	Enabled               bool
	MinLossThreshold      float64
	MaxLossThreshold      float64
	HedgeSizeCap          float64
	MaxSimultaneousHedges int
	MaxAggregateExposure  float64
	MaxHedgeDurationMin   int
	ManipulationThreshold float64
	VolatilityThreshold   float64
	GridEnabled           bool
	OptionsEnabled        bool
}

type mlAdaptationConfig struct {
	Enabled                bool
	MinimumTrainingSamples int
	MinConfidence          float64
	ModelStaleMinutes      int
}

type rlStrategyConfig struct {
	Mode                 string // disabled, shadow, filter_only, live_approved
	MinConfidence        float64
	MaxDrawdownPct       float64
	MinProfitFactor      float64
	MinTradeCount        int
	RequireOOSValidation bool
}

type sentimentConfig struct {
	Enabled                bool
	RefreshIntervalSec     int
	TimeoutSec             int
	MaxRetries             int
	StaleThresholdSec      int
	MinConfidenceThreshold float64
}

// DefaultAdvancedConfig returns safe defaults for all advanced features.
// High-risk/new functionality is DISABLED by default.
func DefaultAdvancedConfig() AdvancedConfig {
	return AdvancedConfig{
		LossRecovery: lossRecoveryConfig{
			Enabled:                 true,
			MaxDailyLossPercent:     2.0,
			MaxDailyLossCount:       3,
			MaxConsecutiveLosses:    2,
			RecoverySizeMultiplier:  0.50,
			RecoveryMinConfluence:   80,
			RecoveryMinSetupGrade:   "A",
			RecoveryMinConfidence:   75,
			RecoveryMaxTrades:       3,
			RecoveryExitAfterWins:   2,
			NormalCooldownMinutes:   5,
			RecoveryCooldownMinutes: 30,
			HaltCooldownMinutes:     60,
		},
		Adaptation: adaptationConfig{
			Enabled:           true,
			MaxRiskMultiplier: 1.0,
			GlobalHardMaxRisk: 0.02,
		},
		Hedging: hedgingConfig{
			Enabled:               false, // DISABLED BY DEFAULT
			MinLossThreshold:      0.5,
			MaxLossThreshold:      3.0,
			HedgeSizeCap:          0.5,
			MaxSimultaneousHedges: 2,
			MaxAggregateExposure:  5.0,
			MaxHedgeDurationMin:   120,
			ManipulationThreshold: 70,
			VolatilityThreshold:   0.005,
			GridEnabled:           false, // OFF by default
			OptionsEnabled:        false, // OFF by default
		},
		MLAdaptation: mlAdaptationConfig{
			Enabled:                false, // disabled — research/offline
			MinimumTrainingSamples: 100,
			MinConfidence:          0.65,
			ModelStaleMinutes:      1440,
		},
		RLStrategy: rlStrategyConfig{
			Mode:                 "disabled",
			MinConfidence:        0.7,
			MaxDrawdownPct:       10.0,
			MinProfitFactor:      1.3,
			MinTradeCount:        50,
			RequireOOSValidation: true,
		},
		Sentiment: sentimentConfig{
			Enabled:                false, // disabled — requires API credentials
			RefreshIntervalSec:     300,
			TimeoutSec:             10,
			MaxRetries:             3,
			StaleThresholdSec:      600,
			MinConfidenceThreshold: 0.5,
		},
	}
}

// LoadAdvancedConfig loads advanced config from environment variables.
func LoadAdvancedConfig() AdvancedConfig {
	cfg := DefaultAdvancedConfig()

	// Loss Recovery
	cfg.LossRecovery.Enabled = getEnvBool("LOSS_RECOVERY_ENABLED", cfg.LossRecovery.Enabled)
	cfg.LossRecovery.MaxDailyLossPercent = getEnvFloat("MAX_DAILY_LOSS_PERCENT", cfg.LossRecovery.MaxDailyLossPercent)
	cfg.LossRecovery.MaxConsecutiveLosses = getEnvInt("MAX_CONSECUTIVE_LOSSES", cfg.LossRecovery.MaxConsecutiveLosses)

	// Adaptation
	cfg.Adaptation.Enabled = getEnvBool("ADAPTATION_ENABLED", cfg.Adaptation.Enabled)

	// Hedging
	cfg.Hedging.Enabled = getEnvBool("HEDGING_ENABLED", cfg.Hedging.Enabled)
	cfg.Hedging.GridEnabled = getEnvBool("GRID_HEDGING_ENABLED", cfg.Hedging.GridEnabled)
	cfg.Hedging.OptionsEnabled = getEnvBool("OPTIONS_HEDGING_ENABLED", cfg.Hedging.OptionsEnabled)

	// ML
	cfg.MLAdaptation.Enabled = getEnvBool("ML_ADAPTATION_ENABLED", cfg.MLAdaptation.Enabled)

	// RL
	cfg.RLStrategy.Mode = getEnv("RL_MODE", cfg.RLStrategy.Mode)

	// Sentiment
	cfg.Sentiment.Enabled = getEnvBool("SENTIMENT_ENABLED", cfg.Sentiment.Enabled)

	return cfg
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "TRUE"
}

// getEnvBoolDefaultTrue returns true unless the env var is explicitly false/0.
func getEnvBoolDefaultTrue(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return true
	}
	return !(v == "false" || v == "0" || v == "FALSE")
}

// splitComma parses a comma-separated env value into a trimmed, non-empty slice.
func splitComma(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
