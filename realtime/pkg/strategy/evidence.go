// Package strategy provides ML and Sentiment evidence injection for the
// Predict-A-Trade strategy scorer.
//
// This is a non-invasive layer: it adds ML and Sentiment contributions
// AFTER the existing evidence-based scoring completes. Existing tests
// are unaffected because all new fields default to 0.
package strategy

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// MLWeight is the weight applied to ML pillar contribution (0.15 = 15%).
const MLWeight = 0.15

// SentimentWeight is the weight applied to sentiment (0.05 = 5%).
// Sentiment only affects confidence, NOT direction.
const SentimentWeight = 0.05

// InjectMLContribution adds an ML evidence contribution to the evidence list.
// The ML score (from ONNX inference) is multiplied by MLWeight and added
// as a directional evidence contribution. If ML score is 0, this is a no-op.
//
// direction: BUY or SELL (the ML model's predicted direction)
// mlScore: the ML confidence (0.0 to 1.0)
func InjectMLContribution(evidence *[]types.EvidenceContribution, direction types.Direction, mlScore float64) {
	if mlScore == 0 {
		return
	}
	contribution := mlScore * MLWeight
	*evidence = append(*evidence, types.EvidenceContribution{
		Pillar:      "ML",
		Feature:     "ONNX_INFERENCE",
		Direction:   direction,
		Contribution: decimal.NewFromFloat(contribution),
		NormalizedValue: decimal.NewFromFloat(contribution),
		Weight:      decimal.NewFromFloat(MLWeight),
		ML:          mlScore,
		Quality:     types.QualityAuthoritative,
		Source:      "ml_engine",
		Version:     "1.0",
	})
}

// InjectSentimentContribution adds a sentiment evidence contribution.
// Sentiment only affects confidence (not direction), so it's added as a
// non-directional (NEUTRAL) evidence with a separate weight.
//
// sentimentScore: from Ollama LLM (-1.0 bearish to 1.0 bullish)
func InjectSentimentContribution(evidence *[]types.EvidenceContribution, sentimentScore float64) {
	if sentimentScore == 0 {
		return
	}
	contribution := sentimentScore * SentimentWeight
	dir := types.DirectionBuy
	if sentimentScore < 0 {
		dir = types.DirectionSell
	}
	*evidence = append(*evidence, types.EvidenceContribution{
		Pillar:      "SENTIMENT",
		Feature:     "OLLAMA_LLM",
		Direction:   dir,
		Contribution: decimal.NewFromFloat(contribution),
		NormalizedValue: decimal.NewFromFloat(contribution),
		Weight:      decimal.NewFromFloat(SentimentWeight),
		Sentiment:   sentimentScore,
		Quality:     types.QualityAuthoritative,
		Source:      "ollama_llm",
		Version:     "1.0",
	})
}

// GetMLContribution extracts the total ML contribution from evidence.
func GetMLContribution(evidence []types.EvidenceContribution) float64 {
	total := 0.0
	for _, e := range evidence {
		if e.Pillar == "ML" {
			c, _ := e.Contribution.Float64()
			total += c
		}
	}
	return total
}

// GetSentimentContribution extracts the total sentiment contribution from evidence.
func GetSentimentContribution(evidence []types.EvidenceContribution) float64 {
	total := 0.0
	for _, e := range evidence {
		if e.Pillar == "SENTIMENT" {
			c, _ := e.Contribution.Float64()
			total += c
		}
	}
	return total
}
