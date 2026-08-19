// Package strategy — Shadow evaluation path.
// SOW Phase 2 Section 8-9: Shadow diagnostic computation for regime-mismatched strategies.
//
// When a strategy rejects due to REGIME_MISMATCH, production correctly returns NO-TRADE.
// However, for diagnostic purposes, we continue evaluating evidence to determine
// whether the regime classification is the ONLY thing suppressing otherwise strong setups.
//
// Shadow signals are NEVER delivered to clients and NEVER executed.
// They are persisted as SHADOW_ONLY=true, EXECUTABLE=false for counterfactual analysis.
package strategy

import (
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ShadowResult is the output of shadow evaluation.
// It contains the hypothetical computation that would have occurred
// if the regime gate had not rejected.
type ShadowResult struct {
	StrategyID         types.StrategyID
	Timestamp          time.Time
	Regime             types.Regime
	HypotheticalDirection types.Direction
	HypotheticalScore  decimal.Decimal
	HypotheticalLong   decimal.Decimal
	HypotheticalShort  decimal.Decimal
	HypotheticalEntry   decimal.Decimal
	HypotheticalSL      decimal.Decimal
	HypotheticalTP1     decimal.Decimal
	HypotheticalTP2     decimal.Decimal
	HypotheticalTP3     decimal.Decimal
	HypotheticalRR      decimal.Decimal
	Evidence            []types.EvidenceContribution
	FailedProductionReason string
	ShadowOnly          bool
	Executable          bool
}

// EvaluateShadow runs the strategy evaluation bypassing the regime gate.
// This computes what the strategy WOULD have produced if the regime were acceptable.
//
// IMPORTANT: This does NOT change the production decision. The production
// NO-TRADE result remains. This is purely diagnostic.
func EvaluateShadow(s Strategy, state *features.MarketState) *ShadowResult {
	if state == nil {
		return nil
	}

	// First, check if the strategy would reject on regime
	// We do this by examining the strategy's accepted regimes
	cfg := getStrategyConfig(s)
	if cfg == nil {
		return nil
	}

	// Check if regime is actually mismatched
	regimeMatched := false
	for _, r := range cfg.AcceptedRegimes {
		if state.Regime.Current == r {
			regimeMatched = true
			break
		}
	}

	if regimeMatched {
		// Regime matches — no shadow evaluation needed
		return nil
	}

	// Temporarily bypass regime by creating a modified state copy
	// We evaluate the strategy's evidence computation without the regime gate
	shadowResult := &ShadowResult{
		StrategyID:            s.ID(),
		Timestamp:             state.Timestamp,
		Regime:                state.Regime.Current,
		ShadowOnly:            true,
		Executable:            false,
		FailedProductionReason: "REGIME_MISMATCH: regime=" + string(state.Regime.Current) + " not in accepted=" + regimesToString(cfg.AcceptedRegimes),
	}

	// Run the evidence computation by calling the strategy's evaluate
	// but with a modified state where the regime is set to an accepted one
	shadowState := *state
	// Set regime to the first accepted regime to bypass the gate
	if len(cfg.AcceptedRegimes) > 0 {
		shadowState.Regime = features.RegimeFeatures{
			Current:             cfg.AcceptedRegimes[0],
			Previous:            state.Regime.Current,
			Confidence:          state.Regime.Confidence,
			Volatility:          state.Regime.Volatility,
			EnteredAt:           state.Regime.EnteredAt,
			Age:                 state.Regime.Age,
			EntryReason:         state.Regime.EntryReason,
			RawRegime:           state.Regime.RawRegime,
			RegimeEngineVersion: state.Regime.RegimeEngineVersion,
		}
	}

	// Evaluate with shadow state
	result := s.Evaluate(&shadowState)

	shadowResult.HypotheticalDirection = result.Direction
	shadowResult.HypotheticalScore = result.RawScore
	shadowResult.HypotheticalLong = result.LongScore
	shadowResult.HypotheticalShort = result.ShortScore
	shadowResult.HypotheticalEntry = result.EntryPrice
	shadowResult.HypotheticalSL = result.StopLoss
	shadowResult.HypotheticalTP1 = result.TP1
	shadowResult.HypotheticalTP2 = result.TP2
	shadowResult.HypotheticalTP3 = result.TP3
	shadowResult.Evidence = result.Evidence

	// Compute hypothetical R:R
	if !result.StopLoss.IsZero() && !result.EntryPrice.IsZero() {
		risk := result.EntryPrice.Sub(result.StopLoss).Abs()
		reward := result.TP1.Sub(result.EntryPrice).Abs()
		if !risk.IsZero() {
			shadowResult.HypotheticalRR = reward.Div(risk)
		}
	}

	return shadowResult
}

// getStrategyConfig extracts the StrategyConfig from a strategy instance.
func getStrategyConfig(s Strategy) *StrategyConfig {
	switch v := s.(type) {
	case *StandardScalping:
		return &v.cfg
	case *UltraScalping:
		return &v.cfg
	case *StandardSwing:
		return &v.cfg
	case *TrendSwing:
		return &v.cfg
	}
	return nil
}

// regimesToString converts a regime list to a comma-separated string.
func regimesToString(regimes []types.Regime) string {
	result := ""
	for i, r := range regimes {
		if i > 0 {
			result += ","
		}
		result += string(r)
	}
	return result
}

// EvaluateAllShadows runs shadow evaluation for all strategies that would
// reject on regime mismatch. Returns only non-nil results.
func EvaluateAllShadows(strategies []Strategy, state *features.MarketState) []*ShadowResult {
	var results []*ShadowResult
	for _, s := range strategies {
		shadow := EvaluateShadow(s, state)
		if shadow != nil {
			results = append(results, shadow)
		}
	}
	return results
}
