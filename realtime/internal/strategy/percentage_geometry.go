package strategy

import (
	"database/sql"
	"log"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ─── Percentage-based SL/TP configuration (database-driven) ───

type ExitProfileConfig struct {
	StrategyID       string
	CalculationMode  string  // "PERCENTAGE" or "ATR"
	StopPct          float64 // SL as fraction of entry price (0.0015 = 0.15%)
	TP1Pct           float64
	TP2Pct           float64
	TP3Pct           float64
	MinStopATRMult   float64 // SL must be >= this × ATR
	MaxStopATRMult   float64 // SL must be <= this × ATR
	MinTP1ATRMult    float64
	MaxTP1ATRMult    float64
	LoadedAt        time.Time
}

var (
	profileCache   map[string]*ExitProfileConfig
	profileCacheMu sync.RWMutex
	dbPool          *sql.DB
)

// ClearProfileCache clears all cached exit profiles (call after DB updates).
func ClearProfileCache() {
	profileCacheMu.Lock()
	defer profileCacheMu.Unlock()
	profileCache = make(map[string]*ExitProfileConfig)
}

// InitExitProfileDB sets the database connection for loading exit profiles.
func InitExitProfileDB(pool *sql.DB) {
	dbPool = pool
}

// LoadExitProfile reads the SL/TP configuration for a strategy from the database.
// Falls back to hardcoded ATR multipliers if database is not available.
func LoadExitProfile(strategyID string) *ExitProfileConfig {
	profileCacheMu.RLock()
	if cached, ok := profileCache[strategyID]; ok && time.Since(cached.LoadedAt) < 5*time.Minute {
		profileCacheMu.RUnlock()
		return cached
	}
	profileCacheMu.RUnlock()

	if dbPool == nil {
		return nil // No DB — caller falls back to ATR multipliers
	}

	query := `
		SELECT strategy_id, calculation_mode, 
		       stop_pct, tp1_pct, tp2_pct, tp3_pct,
		       min_stop_atr_mult, max_stop_atr_mult, min_tp1_atr_mult, max_tp1_atr_mult
		FROM trading.exit_profiles
		WHERE strategy_id = $1
		ORDER BY effective_from DESC LIMIT 1`

	var cfg ExitProfileConfig
	err := dbPool.QueryRow(query, strategyID).Scan(
		&cfg.StrategyID, &cfg.CalculationMode,
		&cfg.StopPct, &cfg.TP1Pct, &cfg.TP2Pct, &cfg.TP3Pct,
		&cfg.MinStopATRMult, &cfg.MaxStopATRMult, &cfg.MinTP1ATRMult, &cfg.MaxTP1ATRMult,
	)
	if err != nil {
		log.Printf("[EXIT_PROFILE] LoadExitProfile(%s): DB query error: %v", strategyID, err)
		return nil
	}
	cfg.LoadedAt = time.Now()
	log.Printf("[EXIT_PROFILE] Loaded %s: mode=%s SL=%.4f%% TP1=%.4f%% minATR=%.1f", strategyID, cfg.CalculationMode, cfg.StopPct*100, cfg.TP1Pct*100, cfg.MinStopATRMult)

	profileCacheMu.Lock()
	if profileCache == nil {
		profileCache = make(map[string]*ExitProfileConfig)
	}
	profileCache[strategyID] = &cfg
	profileCacheMu.Unlock()

	return &cfg
}

// computePercentageSLTP calculates SL/TP using percentage of entry price,
// with ATR guardrails to prevent extremes during very low/high volatility.
//
// SL = Entry × (1 ± stop_pct), clamped to [min_atr_mult × ATR, max_atr_mult × ATR]
// TP1 = Entry × (1 ± tp1_pct), clamped to [min_tp1_atr_mult × ATR, max_tp1_atr_mult × ATR]
// TP2 = Entry × (1 ± tp2_pct)
// TP3 = Entry × (1 ± tp3_pct)
//
// This is price-adaptive: when gold moves from $2000 to $4500, the SL/TP
// automatically scales proportionally — no need to change parameters.
func computePercentageSLTP(
	entry decimal.Decimal,
	direction types.Direction,
	atr decimal.Decimal,
	cfg *ExitProfileConfig,
) (sl, tp1, tp2, tp3 decimal.Decimal) {

	if entry.IsZero() || cfg == nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	}

	if cfg.CalculationMode == "ATR" {
		// Legacy ATR mode — use the old ATR multiplier approach
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero // signal to use ATR path
	}

	// ─── PERCENTAGE MODE ───
	// SL = Entry × stop_pct (as absolute distance)
	slDist := entry.Mul(decimal.NewFromFloat(cfg.StopPct))
	tp1Dist := entry.Mul(decimal.NewFromFloat(cfg.TP1Pct))
	tp2Dist := entry.Mul(decimal.NewFromFloat(cfg.TP2Pct))
	tp3Dist := entry.Mul(decimal.NewFromFloat(cfg.TP3Pct))

	// ─── ATR guardrails ───
	// Ensure SL distance is within [min_atr × ATR, max_atr × ATR]
	if !atr.IsZero() {
		minSL := atr.Mul(decimal.NewFromFloat(cfg.MinStopATRMult))
		maxSL := atr.Mul(decimal.NewFromFloat(cfg.MaxStopATRMult))
		if slDist.LessThan(minSL) {
			slDist = minSL // too tight — widen to minimum
		} else if slDist.GreaterThan(maxSL) {
			slDist = maxSL // too loose — tighten to maximum
		}

		// Ensure TP1 distance is within guardrails
		minTP1 := atr.Mul(decimal.NewFromFloat(cfg.MinTP1ATRMult))
		maxTP1 := atr.Mul(decimal.NewFromFloat(cfg.MaxTP1ATRMult))
		if tp1Dist.LessThan(minTP1) {
			tp1Dist = minTP1
		} else if tp1Dist.GreaterThan(maxTP1) {
			tp1Dist = maxTP1
		}
	}

	// Apply direction
	if direction == types.DirectionBuy {
		sl = entry.Sub(slDist)
		tp1 = entry.Add(tp1Dist)
		tp2 = entry.Add(tp2Dist)
		tp3 = entry.Add(tp3Dist)
	} else {
		sl = entry.Add(slDist)
		tp1 = entry.Sub(tp1Dist)
		tp2 = entry.Sub(tp2Dist)
		tp3 = entry.Sub(tp3Dist)
	}

	return sl, tp1, tp2, tp3
}

// LoadStrategyConfigFromDB loads the full strategy configuration (confluence thresholds,
// accepted regimes/sessions, R:R) from strategy_config_versions table.
type StrategyDBConfig struct {
	MinConfluence    float64
	MinMTFAlignment  float64
	MinADX           float64
	MinRR            float64
	CalculationMode  string
	AcceptedRegimes  []string
	AcceptedSessions []string
}

func LoadStrategyConfig(strategyID string) *StrategyDBConfig {
	if dbPool == nil {
		return nil
	}

	query := `
		SELECT values FROM trading.strategy_config_versions
		WHERE strategy_id = $1 AND effective_to IS NULL
		ORDER BY effective_from DESC LIMIT 1`

	var valuesJSON json.RawMessage
	err := dbPool.QueryRow(query, strategyID).Scan(&valuesJSON)
	if err != nil {
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(valuesJSON, &raw); err != nil {
		return nil
	}

	cfg := &StrategyDBConfig{}
	if v, ok := raw["min_confluence"].(float64); ok {
		cfg.MinConfluence = v
	}
	if v, ok := raw["min_mtf_alignment"].(float64); ok {
		cfg.MinMTFAlignment = v
	}
	if v, ok := raw["min_adx"].(float64); ok {
		cfg.MinADX = v
	}
	if v, ok := raw["min_rr"].(float64); ok {
		cfg.MinRR = v
	}
	if v, ok := raw["calculation_mode"].(string); ok {
		cfg.CalculationMode = v
	}
	if arr, ok := raw["accepted_regimes"].([]interface{}); ok {
		for _, r := range arr {
			cfg.AcceptedRegimes = append(cfg.AcceptedRegimes, fmt.Sprintf("%v", r))
		}
	}
	if arr, ok := raw["accepted_sessions"].([]interface{}); ok {
		for _, s := range arr {
			cfg.AcceptedSessions = append(cfg.AcceptedSessions, fmt.Sprintf("%v", s))
		}
	}

	return cfg
}
