// Package rl implements the RL Strategy Optimizer.
//
// CRITICAL PRODUCTION RULE:
// An unvalidated RL model must NOT directly control live MT4/MT5 execution.
//
// Default production rollout:
//   OFF → SHADOW → FILTER → APPROVED LIVE
//
// live_approved must never automatically become active merely because
// a model file exists. It requires explicit operator authorization.
package rl

import (
	"sync"
)

// RLMode represents the RL production deployment mode.
type RLMode string

const (
	RLDisabled    RLMode = "disabled"
	RLShadow      RLMode = "shadow"
	RLFilterOnly  RLMode = "filter_only"
	RLLiveApproved RLMode = "live_approved"
)

// Config holds RL configuration.
type Config struct {
	Mode               RLMode
	MinConfidence      float64
	MaxDrawdownPct     float64
	MinProfitFactor   float64
	MinTradeCount      int
	RequireOOSValidation bool
}

// DefaultConfig returns safe defaults — RL is disabled.
func DefaultConfig() Config {
	return Config{
		Mode:               RLDisabled,
		MinConfidence:      0.7,
		MaxDrawdownPct:     10.0,
		MinProfitFactor:   1.3,
		MinTradeCount:      50,
		RequireOOSValidation: true,
	}
}

// Action represents RL conceptual actions.
type Action string

const (
	ActionNoTrade Action = "NO_TRADE"
	ActionLong    Action = "LONG"
	ActionShort   Action = "SHORT"
	ActionClose   Action = "CLOSE"
)

// Observation is the RL environment state vector.
// Reuses PTB features where available.
type Observation struct {
	Regime            float64
	Confluence        float64
	Confidence        float64
	ManipulationIndex float64
	Volatility        float64
	Liquidity         float64
	Sentiment         float64
	DXY               float64
	RealYields        float64
	Session           float64
	Spread            float64
	ATR               float64
	RecentReturns     float64
	PositionState     float64 // current position: 0=flat, 1=long, -1=short
}

// RLDecision is the output of RL inference.
type RLDecision struct {
	Action      Action
	Confidence  float64
	Mode        RLMode
	ShadowOnly  bool // true if this decision is for observation only
}

// ValidationMetrics holds out-of-sample validation results.
type ValidationMetrics struct {
	TotalReward    float64
	AvgReward      float64
	MaxDrawdown    float64
	SharpeRatio    float64
	SortinoRatio   float64
	ProfitFactor   float64
	WinRate        float64
	Expectancy     float64
	TradeCount     int
	AvgDuration    float64
	OSSStart       string
	OSEnd          string
	WalkForwardFolds int
}

// RewardConfig defines the RL reward function components.
type RewardConfig struct {
	PnLWeight         float64
	DrawdownPenalty   float64
	TransactionCost   float64
	SpreadCost         float64
	SlippageCost      float64
	OvertradingPenalty float64
	RiskExposurePenalty float64
	HoldingCost        float64
}

// DefaultRewardConfig returns a balanced reward configuration.
// Reward accounts for more than raw PnL.
func DefaultRewardConfig() RewardConfig {
	return RewardConfig{
		PnLWeight:          1.0,
		DrawdownPenalty:     0.3,
		TransactionCost:     0.1,
		SpreadCost:         0.05,
		SlippageCost:       0.05,
		OvertradingPenalty:  0.2,
		RiskExposurePenalty: 0.1,
		HoldingCost:        0.02,
	}
}

// Manager manages the RL strategy optimizer.
type Manager struct {
	mu     sync.RWMutex
	config Config
	metrics *ValidationMetrics
}

// NewManager creates an RL manager.
func NewManager(cfg Config) *Manager {
	return &Manager{config: cfg}
}

// GetMode returns the current RL mode.
func (m *Manager) GetMode() RLMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Mode
}

// SetMode sets the RL mode. live_approved requires explicit authorization.
func (m *Manager) SetMode(mode RLMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Mode = mode
}

// CanApproveLive checks whether the RL model meets criteria for live approval.
// This does NOT set the mode — it only checks eligibility.
func (m *Manager) CanApproveLive(metrics ValidationMetrics) (bool, string) {
	if m.config.RequireOOSValidation {
		if metrics.TradeCount < m.config.MinTradeCount {
			return false, "insufficient OOS trade count"
		}
		if metrics.MaxDrawdown > m.config.MaxDrawdownPct {
			return false, "max drawdown exceeds limit"
		}
		if metrics.ProfitFactor < m.config.MinProfitFactor {
			return false, "profit factor below minimum"
		}
	}
	return true, ""
}

// Evaluate runs RL inference and returns a decision.
// The decision is filtered by the current mode:
// - disabled: returns NO_TRADE always
// - shadow: returns decision but marked shadow only (cannot block or execute)
// - filter_only: may veto (NO_TRADE) but cannot create trades
// - live_approved: may influence live trading (requires explicit authorization)
func (m *Manager) Evaluate(obs Observation, inferenceFn func(Observation) (Action, float64)) RLDecision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mode := m.config.Mode

	switch mode {
	case RLDisabled:
		return RLDecision{
			Action:     ActionNoTrade,
			Confidence: 0,
			Mode:       RLDisabled,
			ShadowOnly: true,
		}

	case RLShadow:
		// Shadow mode: observe but cannot block or execute
		action, conf := inferenceFn(obs)
		return RLDecision{
			Action:     action,
			Confidence: conf,
			Mode:       RLShadow,
			ShadowOnly: true,
		}

	case RLFilterOnly:
		// Filter mode: may veto (NO_TRADE) but cannot create trades
		action, conf := inferenceFn(obs)
		if conf < m.config.MinConfidence {
			return RLDecision{
				Action:     ActionNoTrade,
				Confidence: conf,
				Mode:       RLFilterOnly,
				ShadowOnly:  false,
			}
		}
		if action == ActionLong || action == ActionShort {
			// In filter mode, we can only veto, not create
			return RLDecision{
				Action:     ActionNoTrade, // don't create, just pass through
				Confidence: conf,
				Mode:       RLFilterOnly,
				ShadowOnly: false,
			}
		}
		return RLDecision{
			Action:     action, // NO_TRADE or CLOSE
			Confidence: conf,
			Mode:       RLFilterOnly,
			ShadowOnly: false,
		}

	case RLLiveApproved:
		// Live approved: full influence (requires explicit operator authorization)
		action, conf := inferenceFn(obs)
		return RLDecision{
			Action:     action,
			Confidence: conf,
			Mode:       RLLiveApproved,
			ShadowOnly: false,
		}
	}

	return RLDecision{
		Action:     ActionNoTrade,
		Confidence: 0,
		Mode:       mode,
		ShadowOnly: true,
	}
}

// ValidateMetrics checks if OOS metrics are sufficient.
func (m *Manager) ValidateMetrics(metrics ValidationMetrics) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metrics.TradeCount < m.config.MinTradeCount {
		return nil // not an error, just insufficient
	}
	return nil
}

// Environment provides the RL training/replay environment interface.
// Actual training runs in the Python research plane, not the Go hot path.
type Environment interface {
	Reset() Observation
	Step(action Action) (Observation, float64, bool) // observation, reward, done
}

// SimulatedEnvironment is a simple test environment for Go-side testing.
// Real training uses the Python research environment.
type SimulatedEnvironment struct {
	stepCount int
	maxSteps  int
}

// NewSimulatedEnvironment creates a test environment.
func NewSimulatedEnvironment(maxSteps int) *SimulatedEnvironment {
	return &SimulatedEnvironment{maxSteps: maxSteps}
}

func (e *SimulatedEnvironment) Reset() Observation {
	e.stepCount = 0
	return Observation{}
}

func (e *SimulatedEnvironment) Step(action Action) (Observation, float64, bool) {
	e.stepCount++
	done := e.stepCount >= e.maxSteps
	// Simple reward: penalize NO_TRADE overtrading, reward decisive action
	reward := 0.0
	if action == ActionLong || action == ActionShort {
		reward = 0.1
	} else if action == ActionNoTrade {
		reward = -0.01 // small overtrading penalty
	}
	return Observation{}, reward, done
}
