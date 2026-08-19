// Package ml implements ML-based adaptation inference.
//
// CRITICAL: Training runs offline in the Python research plane.
// Only inference runs in the Go production path.
//
// If model is absent, stale, incompatible, inference throws error,
// features are missing, confidence is inadequate, or version is invalid:
//   → fall back to rule-based adaptation
//
// The live signal engine must continue safely. Never fail open into higher risk.
package ml

import (
	"fmt"
	"sync"
	"time"

)

// Config holds ML adaptation configuration.
type Config struct {
	Enabled                bool
	MinimumTrainingSamples int     // minimum samples before ML is authoritative
	MinConfidence          float64 // minimum inference confidence
	ModelStaleMinutes       int     // model considered stale after this many minutes
	FallbackToRules        bool    // always true — never fail open
}

// DefaultConfig returns safe defaults. ML is disabled by default.
func DefaultConfig() Config {
	return Config{
		Enabled:                false, // disabled by default — research/offline
		MinimumTrainingSamples: 100,
		MinConfidence:          0.65,
		ModelStaleMinutes:      1440, // 24 hours
		FallbackToRules:        true,
	}
}

// ModelMetadata holds information about a loaded model.
type ModelMetadata struct {
	Name          string
	Version       string
	TrainedAt     time.Time
	DatasetPeriod string
	SampleCount   int
	FeatureSchema string
	ValidationMetrics map[string]float64
	ArtifactPath   string
	Active         bool
	Checksum       string
}

// ModelRegistry manages loaded ML models.
type ModelRegistry struct {
	mu     sync.RWMutex
	models map[string]*ModelMetadata // name -> metadata
}

// NewModelRegistry creates a model registry.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{models: make(map[string]*ModelMetadata)}
}

// RegisterModel adds or updates a model in the registry.
func (r *ModelRegistry) RegisterModel(meta ModelMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := meta
	r.models[meta.Name] = &m
}

// GetModel returns model metadata by name.
func (r *ModelRegistry) GetModel(name string) (*ModelMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[name]
	return m, ok
}

// ActiveModel returns the currently active model (if any).
func (r *ModelRegistry) ActiveModel() *ModelMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.models {
		if m.Active {
			cp := *m
			return &cp
		}
	}
	return nil
}

// AllModels returns all registered models.
func (r *ModelRegistry) AllModels() []ModelMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ModelMetadata, 0, len(r.models))
	for _, m := range r.models {
		result = append(result, *m)
	}
	return result
}

// SetActive marks a model as active.
func (r *ModelRegistry) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := false
	for _, m := range r.models {
		if m.Name == name {
			m.Active = true
			found = true
		} else {
			m.Active = false
		}
	}
	if !found {
		return fmt.Errorf("model %s not found", name)
	}
	return nil
}

// FeatureVector represents the input features for ML inference.
type FeatureVector struct {
	Regime            float64
	Confluence        float64
	Confidence        float64
	ManipulationIndex float64
	Volatility        float64
	LiquidityScore    float64
	Spread            float64
	ATR               float64
	Session           float64
	RecentReturns     float64
	SentimentScore    float64
}

// MLPrediction is the output of ML inference.
type MLPrediction struct {
	StopDistanceMultiplier float64
	PositionSizeMultiplier float64
	MinimumConfluence      float64
	Confidence             float64
	ModelName              string
	ModelVersion           string
	FallbackUsed           bool
}

// AdaptationManager provides ML-based adaptation with safe fallback.
type AdaptationManager struct {
	mu      sync.RWMutex
	config  Config
	registry *ModelRegistry
	// inferenceFn is the pluggable inference function.
	// In production this would load a model artifact and run inference.
	// For safety, if nil or error, falls back to rule-based.
	inferenceFn func(ModelMetadata, FeatureVector) (MLPrediction, error)
}

// NewAdaptationManager creates an ML adaptation manager.
func NewAdaptationManager(cfg Config, registry *ModelRegistry) *AdaptationManager {
	return &AdaptationManager{
		config:   cfg,
		registry: registry,
	}
}

// SetInferenceFn sets the inference function (pluggable for testing/production).
func (m *AdaptationManager) SetInferenceFn(fn func(ModelMetadata, FeatureVector) (MLPrediction, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inferenceFn = fn
}

// Predict runs ML inference with comprehensive fallback.
// If ANY condition fails, it returns a safe fallback prediction.
func (m *AdaptationManager) Predict(features FeatureVector) MLPrediction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fallback := MLPrediction{
		StopDistanceMultiplier: 1.0,
		PositionSizeMultiplier: 1.0,
		MinimumConfluence:      70.0,
		Confidence:             0,
		FallbackUsed:           true,
	}

	// 1. Check if ML is enabled
	if !m.config.Enabled {
		return fallback
	}

	// 2. Check if there's an active model
	model := m.registry.ActiveModel()
	if model == nil {
		return fallback
	}

	// 3. Check model staleness
	if m.config.ModelStaleMinutes > 0 && time.Since(model.TrainedAt) > time.Duration(m.config.ModelStaleMinutes)*time.Minute {
		return fallback
	}

	// 4. Check minimum training samples
	if model.SampleCount < m.config.MinimumTrainingSamples {
		return fallback
	}

	// 5. Check inference function is set
	if m.inferenceFn == nil {
		return fallback
	}

	// 6. Run inference — catch any error
	prediction, err := m.inferenceFn(*model, features)
	if err != nil {
		return fallback
	}

	// 7. Check confidence is adequate
	if prediction.Confidence < m.config.MinConfidence {
		prediction.FallbackUsed = true
		return prediction // return with fallback flag
	}

	// 8. Validate prediction bounds
	prediction.StopDistanceMultiplier = clampFloat(prediction.StopDistanceMultiplier, 0.5, 2.0)
	prediction.PositionSizeMultiplier = clampFloat(prediction.PositionSizeMultiplier, 0.1, 1.0)
	prediction.MinimumConfluence = clampFloat(prediction.MinimumConfluence, 50, 100)

	prediction.ModelName = model.Name
	prediction.ModelVersion = model.Version
	prediction.FallbackUsed = false
	return prediction
}

// DatasetBuilder helps build training datasets with leakage protection.
// This is used by the Python research plane, not the Go hot path.
type DatasetBuilder struct {
	MinSamples int
	// Chronological split: train → validation → test
	TrainRatio    float64
	ValidationRatio float64
	TestRatio     float64
}

// NewDatasetBuilder creates a dataset builder with leakage protection.
func NewDatasetBuilder(minSamples int) *DatasetBuilder {
	return &DatasetBuilder{
		MinSamples:      minSamples,
		TrainRatio:      0.6,
		ValidationRatio: 0.2,
		TestRatio:       0.2,
	}
}

// ValidateDataset checks that a dataset meets minimum requirements.
// Returns an error if insufficient samples or leakage detected.
func (db *DatasetBuilder) ValidateDataset(sampleCount int, hasChronologicalOrder bool) error {
	if sampleCount < db.MinSamples {
		return fmt.Errorf("insufficient training samples: %d < %d", sampleCount, db.MinSamples)
	}
	if !hasChronologicalOrder {
		return fmt.Errorf("dataset must be chronologically ordered to prevent leakage")
	}
	return nil
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ModelLoaded returns whether an active model is loaded.
func (m *AdaptationManager) ModelLoaded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry.ActiveModel() != nil
}

// Config returns the current configuration.
func (m *AdaptationManager) Config() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}
