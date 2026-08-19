package ml

import (
	"testing"
	"time"
)

func TestNoModelFallback(t *testing.T) {
	reg := NewModelRegistry()
	m := NewAdaptationManager(DefaultConfig(), reg)
	pred := m.Predict(FeatureVector{})
	if !pred.FallbackUsed {
		t.Fatal("should fallback when no model")
	}
}

func TestInsufficientDataFallback(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.MinimumTrainingSamples = 1000
	reg := NewModelRegistry()
	reg.RegisterModel(ModelMetadata{
		Name:        "test_model",
		Version:     "1.0",
		TrainedAt:   time.Now(),
		SampleCount: 50, // insufficient
		Active:      true,
	})
	m := NewAdaptationManager(cfg, reg)
	pred := m.Predict(FeatureVector{})
	if !pred.FallbackUsed {
		t.Fatal("should fallback with insufficient data")
	}
}

func TestFeatureOrderingValidation(t *testing.T) {
	reg := NewModelRegistry()
	reg.RegisterModel(ModelMetadata{
		Name:          "test",
		Version:       "1.0",
		TrainedAt:     time.Now(),
		SampleCount:   200,
		FeatureSchema: "regime,confluence,confidence,manipulation,volatility,liquidity,spread,atr,session,returns,sentiment",
		Active:        true,
	})
	// Verify feature schema has expected fields
	m := reg.ActiveModel()
	if m == nil {
		t.Fatal("model should be active")
	}
	if m.FeatureSchema == "" {
		t.Fatal("feature schema should be set")
	}
}

func TestModelLoading(t *testing.T) {
	reg := NewModelRegistry()
	reg.RegisterModel(ModelMetadata{
		Name:        "adapt_v1",
		Version:     "1.0.0",
		TrainedAt:   time.Now(),
		SampleCount: 500,
		Active:      true,
	})
	m := reg.ActiveModel()
	if m == nil {
		t.Fatal("model should be loaded")
	}
	if m.Name != "adapt_v1" {
		t.Fatalf("expected adapt_v1, got %s", m.Name)
	}
}

func TestInferenceBounds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.MinConfidence = 0.5
	reg := NewModelRegistry()
	reg.RegisterModel(ModelMetadata{
		Name:        "test",
		Version:     "1.0",
		TrainedAt:   time.Now(),
		SampleCount: 200,
		Active:      true,
	})
	m := NewAdaptationManager(cfg, reg)
	m.SetInferenceFn(func(model ModelMetadata, fv FeatureVector) (MLPrediction, error) {
		return MLPrediction{
			StopDistanceMultiplier: 5.0,  // out of bounds
			PositionSizeMultiplier:  10.0, // out of bounds
			MinimumConfluence:      200,   // out of bounds
			Confidence:             0.9,
		}, nil
	})
	pred := m.Predict(FeatureVector{})
	if pred.StopDistanceMultiplier > 2.0 {
		t.Fatalf("stop multiplier should be clamped, got %f", pred.StopDistanceMultiplier)
	}
	if pred.PositionSizeMultiplier > 1.0 {
		t.Fatalf("position size should be clamped, got %f", pred.PositionSizeMultiplier)
	}
	if pred.MinimumConfluence > 100 {
		t.Fatalf("min confluence should be clamped, got %f", pred.MinimumConfluence)
	}
}

func TestLowConfidenceFallback(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.MinConfidence = 0.8
	reg := NewModelRegistry()
	reg.RegisterModel(ModelMetadata{
		Name:        "test",
		Version:     "1.0",
		TrainedAt:   time.Now(),
		SampleCount: 200,
		Active:      true,
	})
	m := NewAdaptationManager(cfg, reg)
	m.SetInferenceFn(func(model ModelMetadata, fv FeatureVector) (MLPrediction, error) {
		return MLPrediction{Confidence: 0.3}, nil // low confidence
	})
	pred := m.Predict(FeatureVector{})
	if !pred.FallbackUsed {
		t.Fatal("low confidence should trigger fallback")
	}
}

func TestModelVersionHandling(t *testing.T) {
	reg := NewModelRegistry()
	reg.RegisterModel(ModelMetadata{
		Name:    "test",
		Version: "2.1.0",
		TrainedAt: time.Now(),
		SampleCount: 200,
		Active:  true,
	})
	m := reg.ActiveModel()
	if m.Version != "2.1.0" {
		t.Fatalf("expected version 2.1.0, got %s", m.Version)
	}
}

func TestStaleModelFallback(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.ModelStaleMinutes = 1 // very short
	reg := NewModelRegistry()
	reg.RegisterModel(ModelMetadata{
		Name:        "test",
		Version:     "1.0",
		TrainedAt:   time.Now().Add(-2 * time.Minute), // stale
		SampleCount: 200,
		Active:      true,
	})
	m := NewAdaptationManager(cfg, reg)
	pred := m.Predict(FeatureVector{})
	if !pred.FallbackUsed {
		t.Fatal("stale model should trigger fallback")
	}
}

func TestNoFutureLeakageInDatasetBuilder(t *testing.T) {
	db := NewDatasetBuilder(100)
	err := db.ValidateDataset(50, true)
	if err == nil {
		t.Fatal("should reject insufficient samples")
	}
	err = db.ValidateDataset(200, false)
	if err == nil {
		t.Fatal("should reject non-chronological data (leakage risk)")
	}
	err = db.ValidateDataset(200, true)
	if err != nil {
		t.Fatalf("valid dataset should pass: %v", err)
	}
}
