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
	ProviderMode   string
	ReplayFile     string
	Symbols        []string
	BasePrice      float64

	// Engine
	TickRateMs     int
	FeatureWorkers int

	// Strategy
	MaxSpreadPips   float64
	MinRR           float64
	MaxCostToTarget float64
	MaxExposure     float64

	// CORS/origin validation
	AllowedOrigins []string

	// Logging
	LogLevel string

	// COT (Commitment of Traders) — optional macro/positioning data
	FMPAPIKey   string
	COTSymbol   string
	COTEnabled  bool

	// DXY (US Dollar Index) — Twelve Data API
	TwelveDataAPIKey string
	DXYEnabled       bool

	// ML Inference Engine
	MLEnabled   bool
	ModelsDir    string

	// Cross-Market Macro Engine
	CrossMarketMode      string // disabled|shadow|active
	CrossMarketEnabled   bool
	EURUSDEnabled        bool
	RealYieldEnabled     bool
	RealYieldProvider    string // fmp|fred|disabled
	FREDAPIKey           string // FRED API key for real yield data
	VIXEnabled           bool
	BTCEnabled           bool
	OilEnabled           bool

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
	NewsBreakoutEnabled     bool
	NewsBreakoutPrepareSec  int
	NewsBreakoutExpirySec   int
	NewsBreakoutEntryATR    float64
	NewsBreakoutMaxSpread   float64
	NewsBreakoutMaxRiskPct  float64
	NewsBreakoutSLATR       float64
	NewsBreakoutTPATR       float64

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
	OllamaTimeout  string
}

// Default returns a config suitable for local development/testing.
// Production values come from environment variables.
func Default() *Config {
	return &Config{
		DBURL:           func() string {
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
		MinRR:           getEnvFloat("MIN_RR", 1.5),
		MaxCostToTarget: getEnvFloat("MAX_COST_TO_TARGET", 0.35),
		MaxExposure:     getEnvFloat("MAX_EXPOSURE", 5.0),
		AllowedOrigins:  strings.Split(getEnv("ALLOWED_ORIGINS", "https://platform.predictatrade.com,https://predictatrade.com"), ","),
		LogLevel:        getEnv("LOG_LEVEL", "info"),

		// Cross-Market Macro Engine
		CrossMarketMode:      getEnv("CROSS_MARKET_MODE", "shadow"),
		CrossMarketEnabled:   getEnvBool("CROSS_MARKET_ENABLED", true),
		EURUSDEnabled:        getEnvBool("EURUSD_ENABLED", true),
		RealYieldEnabled:     getEnvBool("REAL_YIELD_ENABLED", false),
		RealYieldProvider:    getEnv("REAL_YIELD_PROVIDER", "disabled"),
		FREDAPIKey:           getEnv("FRED_API_KEY", ""),
		VIXEnabled:           getEnvBool("VIX_ENABLED", false),
		BTCEnabled:           getEnvBool("BTC_ENABLED", false),
		OilEnabled:           getEnvBool("OIL_ENABLED", false),

		// COT provider — optional, fails safe if not configured
		FMPAPIKey:  getEnv("FMP_API_KEY", ""),
		COTSymbol:  getEnv("COT_SYMBOL", "GC"), // Gold futures
		COTEnabled: getEnvBool("COT_ENABLED", false),

		// DXY provider — Twelve Data API for DXY computation
		TwelveDataAPIKey: getEnv("TWELVEDATA_API_KEY", ""),
		DXYEnabled:      getEnvBool("DXY_ENABLED", false),
		MLEnabled:       getEnvBool("ML_ENABLED", false),
		ModelsDir:       getEnv("MODELS_DIR", "models"),

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
		NewsBreakoutEnabled:     getEnvBool("NEWS_BREAKOUT_ENABLED", false),
		NewsBreakoutPrepareSec:  getEnvInt("NEWS_BREAKOUT_PREPARE_SECONDS", 120),
		NewsBreakoutExpirySec:   getEnvInt("NEWS_BREAKOUT_EXPIRY_SECONDS", 300),
		NewsBreakoutEntryATR:    getEnvFloat("NEWS_BREAKOUT_ENTRY_ATR_MULTIPLIER", 0.5),
		NewsBreakoutMaxSpread:   getEnvFloat("NEWS_BREAKOUT_MAX_SPREAD", 3.0),
		NewsBreakoutMaxRiskPct:  getEnvFloat("NEWS_BREAKOUT_MAX_RISK_PCT", 1.0),
		NewsBreakoutSLATR:       getEnvFloat("NEWS_BREAKOUT_SL_ATR_MULTIPLIER", 1.0),
		NewsBreakoutTPATR:       getEnvFloat("NEWS_BREAKOUT_TP_ATR_MULTIPLIER", 2.0),

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
		NtfyServerURL:         getEnv("NTFY_SERVER_URL", ""),
		NtfyTopic:             getEnv("NTFY_TOPIC", ""),
		NtfyAccessToken:       getEnv("NTFY_ACCESS_TOKEN", ""),

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
	"CHANGE_ME_IN_PRODUCTION_USE_SECRET_FILE":    true,
	"pat_local_dev_secret_change_in_production":  true,
	"change_this_to_a_long_random_secret":         true,
	"changeme":                                    true,
	"placeholder":                                 true,
	"development":                                true,
	"secret":                                      true,
	"":                                            true,
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
	if c.IsProduction() && containsInsecureDBPassword(c.DBURL) {
		return fmt.Errorf("DATABASE_URL contains an insecure hardcoded password — supply credentials via production secret")
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
	RecoveryMinSetupGrade    string
	RecoveryMinConfidence    float64
	RecoveryMaxTrades       int
	RecoveryExitAfterWins   int
	NormalCooldownMinutes   int
	RecoveryCooldownMinutes int
	HaltCooldownMinutes     int
}

type adaptationConfig struct {
	Enabled                bool
	MaxRiskMultiplier      float64
	GlobalHardMaxRisk      float64
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
	MaxDrawdownPct        float64
	MinProfitFactor      float64
	MinTradeCount        int
	RequireOOSValidation bool
}

type sentimentConfig struct {
	Enabled             bool
	RefreshIntervalSec   int
	TimeoutSec           int
	MaxRetries           int
	StaleThresholdSec    int
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
