// Package strategy — ML and Sentiment score injection for the strategy scorer.
//
// This file provides the ApplyMLAndSentiment function that safely injects
// ML and sentiment contributions into a StrategyResult AFTER the existing
// evidence-based scoring is complete. This preserves all existing test
// behavior because new fields default to 0.
package strategy

import (
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ApplyMLAndSentiment injects ML and sentiment contributions into a
// StrategyResult after the existing scoring is complete.
//
// This is called AFTER strategy.Evaluate() returns, so it doesn't affect
// the existing evidence-based directional scoring. It only:
//   1. Adds ML contribution to the total score (directional)
//   2. Adds sentiment contribution to confidence (not directional)
//
// If mlScore is 0 or sentimentScore is 0, the respective injection is skipped.
// This is fail-open: no ML/sentiment = no change to the result.
func ApplyMLAndSentiment(
	result *strategy.StrategyResult,
	mlScore float64,
	mlDirection types.Direction,
	sentimentScore float64,
) *strategy.StrategyResult {
	if result == nil {
		return result
	}

	// ML contribution: affects total score (directional)
	if mlScore != 0 {
		contribution := mlScore * MLWeight
		result.MLContribution = contribution

		// Add to raw score based on ML direction
		scoreDelta := decimal.NewFromFloat(contribution * 100) // scale to 0-100 like other evidence
		if mlDirection == types.DirectionBuy {
			result.LongScore = result.LongScore.Add(scoreDelta)
			result.RawScore = result.RawScore.Add(scoreDelta)
		} else if mlDirection == types.DirectionSell {
			result.ShortScore = result.ShortScore.Add(scoreDelta)
			result.RawScore = result.RawScore.Add(scoreDelta)
		}

		// Add evidence item for audit trail
		result.Evidence = append(result.Evidence, types.EvidenceContribution{
			Pillar:      "ML",
			Feature:     "ONNX_INFERENCE",
			Direction:   mlDirection,
			Contribution: decimal.NewFromFloat(contribution),
			NormalizedValue: decimal.NewFromFloat(contribution),
			Weight:      decimal.NewFromFloat(MLWeight),
			ML:          mlScore,
			Quality:     types.QualityAuthoritative,
			Source:      "ml_engine",
			Version:     "1.0",
		})
	}

	// Sentiment contribution: affects confidence only (NOT direction)
	if sentimentScore != 0 {
		sentimentContribution := sentimentScore * SentimentWeight
		result.SentimentContribution = sentimentContribution
		result.Confidence += sentimentContribution

		// Add evidence item for audit trail
		dir := types.DirectionBuy
		if sentimentScore < 0 {
			dir = types.DirectionSell
		}
		result.Evidence = append(result.Evidence, types.EvidenceContribution{
			Pillar:      "SENTIMENT",
			Feature:     "OLLAMA_LLM",
			Direction:   dir,
			Contribution: decimal.NewFromFloat(sentimentContribution),
			NormalizedValue: decimal.NewFromFloat(sentimentContribution),
			Weight:      decimal.NewFromFloat(SentimentWeight),
			Sentiment:   sentimentScore,
			Quality:     types.QualityAuthoritative,
			Source:      "ollama_llm",
			Version:     "1.0",
		})
	}

	return result
}

// IsMLDirectionValid returns true if the ML direction is a valid trade direction.
func IsMLDirectionValid(dir types.Direction) bool {
	return dir == types.DirectionBuy || dir == types.DirectionSell
}
