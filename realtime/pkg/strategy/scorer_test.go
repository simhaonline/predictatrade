package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func TestApplyMLAndSentiment_NilResult(t *testing.T) {
	result := ApplyMLAndSentiment(nil, 0.8, types.DirectionBuy, 0.5)
	if result != nil {
		t.Error("nil result should return nil")
	}
}

func TestApplyMLAndSentiment_ZeroScores(t *testing.T) {
	// Zero ML and sentiment = no change to result
	r := &strategy.StrategyResult{
		RawScore:   decimal.NewFromInt(50),
		LongScore:  decimal.NewFromInt(50),
		ShortScore: decimal.NewFromInt(30),
	}
	original := *r
	ApplyMLAndSentiment(r, 0, types.DirectionBuy, 0)

	if r.RawScore.Cmp(original.RawScore) != 0 {
		t.Errorf("Zero scores should not change RawScore: got %s, expected %s", r.RawScore, original.RawScore)
	}
	if r.MLContribution != 0 {
		t.Errorf("Zero ML should not set MLContribution: got %f", r.MLContribution)
	}
	if r.SentimentContribution != 0 {
		t.Errorf("Zero sentiment should not set SentimentContribution: got %f", r.SentimentContribution)
	}
}

func TestApplyMLAndSentiment_MLBuy(t *testing.T) {
	r := &strategy.StrategyResult{
		RawScore:   decimal.NewFromInt(50),
		LongScore:  decimal.NewFromInt(50),
		ShortScore: decimal.NewFromInt(30),
	}
	ApplyMLAndSentiment(r, 0.8, types.DirectionBuy, 0)

	// ML contribution = 0.8 * 0.15 = 0.12
	// Score delta = 0.12 * 100 = 12 (scaled to 0-100 range)
	expectedML := 0.8 * MLWeight
	if r.MLContribution != expectedML {
		t.Errorf("MLContribution: got %f, expected %f", r.MLContribution, expectedML)
	}

	// LongScore should increase by 12
	expectedLong := decimal.NewFromInt(50).Add(decimal.NewFromFloat(expectedML * 100))
	if r.LongScore.Cmp(expectedLong) != 0 {
		t.Errorf("LongScore: got %s, expected %s", r.LongScore, expectedLong)
	}

	// Evidence should have ML entry
	mlFound := false
	for _, e := range r.Evidence {
		if e.Pillar == "ML" {
			mlFound = true
			if e.ML != 0.8 {
				t.Errorf("ML field: got %f, expected 0.8", e.ML)
			}
		}
	}
	if !mlFound {
		t.Error("ML evidence item should be present")
	}
}

func TestApplyMLAndSentiment_MLSell(t *testing.T) {
	r := &strategy.StrategyResult{
		RawScore:   decimal.NewFromInt(40),
		LongScore:  decimal.NewFromInt(30),
		ShortScore: decimal.NewFromInt(40),
	}
	ApplyMLAndSentiment(r, 0.7, types.DirectionSell, 0)

	// ShortScore should increase
	expectedShort := decimal.NewFromInt(40).Add(decimal.NewFromFloat(0.7*MLWeight*100))
	if r.ShortScore.Cmp(expectedShort) != 0 {
		t.Errorf("ShortScore: got %s, expected %s", r.ShortScore, expectedShort)
	}
}

func TestApplyMLAndSentiment_SentimentOnly(t *testing.T) {
	r := &strategy.StrategyResult{
		RawScore:   decimal.NewFromInt(50),
		LongScore:  decimal.NewFromInt(50),
		ShortScore: decimal.NewFromInt(30),
	}
	ApplyMLAndSentiment(r, 0, types.DirectionBuy, 0.6)

	// Sentiment contribution = 0.6 * 0.05 = 0.03
	expectedSent := 0.6 * SentimentWeight
	if r.SentimentContribution != expectedSent {
		t.Errorf("SentimentContribution: got %f, expected %f", r.SentimentContribution, expectedSent)
	}

	// Confidence should increase
	if r.Confidence != expectedSent {
		t.Errorf("Confidence: got %f, expected %f", r.Confidence, expectedSent)
	}

	// RawScore should NOT change (sentiment doesn't affect direction)
	if r.RawScore.Cmp(decimal.NewFromInt(50)) != 0 {
		t.Errorf("RawScore should not change with sentiment only: got %s", r.RawScore)
	}

	// Evidence should have SENTIMENT entry
	sentFound := false
	for _, e := range r.Evidence {
		if e.Pillar == "SENTIMENT" {
			sentFound = true
			if e.Sentiment != 0.6 {
				t.Errorf("Sentiment field: got %f, expected 0.6", e.Sentiment)
			}
		}
	}
	if !sentFound {
		t.Error("Sentiment evidence item should be present")
	}
}

func TestApplyMLAndSentiment_NegativeSentiment(t *testing.T) {
	r := &strategy.StrategyResult{
		RawScore:   decimal.NewFromInt(50),
		LongScore:  decimal.NewFromInt(50),
		ShortScore: decimal.NewFromInt(30),
	}
	ApplyMLAndSentiment(r, 0, types.DirectionBuy, -0.5)

	// Negative sentiment should reduce confidence
	expectedSent := -0.5 * SentimentWeight
	if r.SentimentContribution != expectedSent {
		t.Errorf("SentimentContribution: got %f, expected %f", r.SentimentContribution, expectedSent)
	}
	if r.Confidence != expectedSent {
		t.Errorf("Confidence with negative sentiment: got %f, expected %f", r.Confidence, expectedSent)
	}
}

func TestApplyMLAndSentiment_BothMLAndSentiment(t *testing.T) {
	r := &strategy.StrategyResult{
		RawScore:   decimal.NewFromInt(50),
		LongScore:  decimal.NewFromInt(50),
		ShortScore: decimal.NewFromInt(30),
	}
	ApplyMLAndSentiment(r, 0.9, types.DirectionBuy, 0.7)

	// Both contributions should be set
	if r.MLContribution != 0.9*MLWeight {
		t.Errorf("MLContribution: got %f", r.MLContribution)
	}
	if r.SentimentContribution < 0.034 || r.SentimentContribution > 0.036 {
		t.Errorf("SentimentContribution: got %f", r.SentimentContribution)
	}

	// Both evidence items should be present
	mlFound, sentFound := false, false
	for _, e := range r.Evidence {
		if e.Pillar == "ML" {
			mlFound = true
		}
		if e.Pillar == "SENTIMENT" {
			sentFound = true
		}
	}
	if !mlFound || !sentFound {
		t.Errorf("Both ML and SENTIMENT evidence should be present: ml=%v sent=%v", mlFound, sentFound)
	}
}

func TestInjectMLContribution(t *testing.T) {
	var evidence []types.EvidenceContribution
	InjectMLContribution(&evidence, types.DirectionBuy, 0.8)

	if len(evidence) != 1 {
		t.Fatalf("Expected 1 evidence item, got %d", len(evidence))
	}
	if evidence[0].Pillar != "ML" {
		t.Errorf("Pillar: got %s, expected ML", evidence[0].Pillar)
	}
	if evidence[0].ML != 0.8 {
		t.Errorf("ML field: got %f, expected 0.8", evidence[0].ML)
	}
}

func TestInjectMLContribution_Zero(t *testing.T) {
	var evidence []types.EvidenceContribution
	InjectMLContribution(&evidence, types.DirectionBuy, 0)
	if len(evidence) != 0 {
		t.Error("Zero ML score should not add evidence")
	}
}

func TestInjectSentimentContribution(t *testing.T) {
	var evidence []types.EvidenceContribution
	InjectSentimentContribution(&evidence, -0.5)

	if len(evidence) != 1 {
		t.Fatalf("Expected 1 evidence item, got %d", len(evidence))
	}
	if evidence[0].Pillar != "SENTIMENT" {
		t.Errorf("Pillar: got %s, expected SENTIMENT", evidence[0].Pillar)
	}
	if evidence[0].Sentiment != -0.5 {
		t.Errorf("Sentiment field: got %f, expected -0.5", evidence[0].Sentiment)
	}
	if evidence[0].Direction != types.DirectionSell {
		t.Errorf("Negative sentiment should have SELL direction: got %s", evidence[0].Direction)
	}
}

func TestGetMLContribution(t *testing.T) {
	evidence := []types.EvidenceContribution{
		{Pillar: "ML", Contribution: decimal.NewFromFloat(0.12)},
		{Pillar: "TREND", Contribution: decimal.NewFromFloat(0.25)},
		{Pillar: "ML", Contribution: decimal.NewFromFloat(0.03)},
	}
	total := GetMLContribution(evidence)
	if total != 0.15 {
		t.Errorf("GetMLContribution: got %f, expected 0.15", total)
	}
}

func TestGetSentimentContribution(t *testing.T) {
	evidence := []types.EvidenceContribution{
		{Pillar: "SENTIMENT", Contribution: decimal.NewFromFloat(0.03)},
		{Pillar: "TREND", Contribution: decimal.NewFromFloat(0.25)},
	}
	total := GetSentimentContribution(evidence)
	if total != 0.03 {
		t.Errorf("GetSentimentContribution: got %f, expected 0.03", total)
	}
}

func TestIsMLDirectionValid(t *testing.T) {
	if !IsMLDirectionValid(types.DirectionBuy) {
		t.Error("BUY should be valid ML direction")
	}
	if !IsMLDirectionValid(types.DirectionSell) {
		t.Error("SELL should be valid ML direction")
	}
	if IsMLDirectionValid(types.DirectionNoTrade) {
		t.Error("NO_TRADE should not be valid ML direction")
	}
}
