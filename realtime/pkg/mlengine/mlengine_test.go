package mlengine

import (
	"os"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestMLEngineDisabledWhenNoModels(t *testing.T) {
	// Create empty temp directory
	tmpDir, err := os.MkdirTemp("", "mlengine_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewMLEngine(tmpDir)
	if engine.IsEnabled() {
		t.Error("Engine should be disabled when no models exist")
	}

	// Predict should return HOLD with 0 confidence (fail-open)
	pred, err := engine.Predict(make([]float64, 42))
	if err != nil {
		t.Errorf("Disabled engine Predict should not error, got: %v", err)
	}
	if pred.Direction != "HOLD" {
		t.Errorf("Disabled engine should return HOLD, got %s", pred.Direction)
	}
	if pred.Confidence != 0 {
		t.Errorf("Disabled engine should return 0 confidence, got %f", pred.Confidence)
	}
}

func TestMLEngineDisabledWhenNil(t *testing.T) {
	var engine *MLEngine // nil engine
	if engine.IsEnabled() {
		t.Error("Nil engine should not be enabled")
	}
	pred, err := engine.Predict(nil)
	if err != nil {
		t.Errorf("Nil engine Predict should not error, got: %v", err)
	}
	if pred.Direction != "HOLD" {
		t.Errorf("Nil engine should return HOLD, got %s", pred.Direction)
	}
}

func TestScalerLoading(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mlengine_scaler")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a scaler.json file
	scalerData := map[string]interface{}{
		"mean":       []float64{1.0, 2.0, 3.0},
		"scale":      []float64{0.5, 1.0, 1.5},
		"n_features": 3,
	}
	data, err := json.Marshal(scalerData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "scaler.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load it
	loaded, err := loadScaler(filepath.Join(tmpDir, "scaler.json"))
	if err != nil {
		t.Fatalf("loadScaler failed: %v", err)
	}
	if len(loaded.Mean) != 3 || loaded.Mean[0] != 1.0 || loaded.Scale[1] != 1.0 {
		t.Errorf("Scaler values mismatch: %+v", loaded)
	}
}

func TestFeatureColumnsLoading(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mlengine_features")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cols := []string{"ema9", "rsi", "atr", "adx", "macd_main"}
	data, _ := json.Marshal(cols)
	os.WriteFile(filepath.Join(tmpDir, "feature_columns.json"), data, 0644)

	loaded, err := loadFeatureColumns(filepath.Join(tmpDir, "feature_columns.json"))
	if err != nil {
		t.Fatalf("loadFeatureColumns failed: %v", err)
	}
	if len(loaded) != 5 || loaded[0] != "ema9" {
		t.Errorf("Feature columns mismatch: %+v", loaded)
	}
}

func TestScaleFeatures(t *testing.T) {
	engine := &MLEngine{
		scaler: &Scaler{
			Mean:  []float64{10, 20, 30},
			Scale: []float64{2, 5, 10},
		},
	}

	features := []float64{12, 30, 80}
	scaled := engine.scaleFeatures(features)

	// (12-10)/2 = 1.0, (30-20)/5 = 2.0, (80-30)/10 = 5.0
	if scaled[0] != 1.0 || scaled[1] != 2.0 || scaled[2] != 5.0 {
		t.Errorf("ScaleFeatures mismatch: %+v expected [1,2,5]", scaled)
	}
}

func TestScaleFeaturesNoScaler(t *testing.T) {
	engine := &MLEngine{scaler: nil}
	features := []float64{1, 2, 3}
	scaled := engine.scaleFeatures(features)

	// Without scaler, features should pass through unchanged
	for i, v := range features {
		if scaled[i] != v {
			t.Errorf("Without scaler, features should pass through: got %f expected %f", scaled[i], v)
		}
	}
}

func TestPredictionStruct(t *testing.T) {
	pred := &Prediction{
		Direction:    "BUY",
		Confidence:   85.5,
		Probabilities: [3]float64{0.05, 0.855, 0.095},
		LatencyMs:    3.2,
		ModelVersion: "v1.0.0",
	}
	if pred.Direction != "BUY" || pred.Confidence != 85.5 {
		t.Error("Prediction struct values mismatch")
	}
	if pred.Probabilities[1] != 0.855 {
		t.Error("Probability array mismatch")
	}
}

func TestModelWatcherNil(t *testing.T) {
	// ModelWatcher should return nil for disabled engine
	var engine *MLEngine
	w := NewModelWatcher(engine, "/tmp/test")
	if w != nil {
		t.Error("ModelWatcher should be nil for nil engine")
	}
}

func TestFailOpenBehavior(t *testing.T) {
	// Engine with enabled=false should always fail-open (HOLD, 0 confidence)
	engine := &MLEngine{enabled: false}

	pred, err := engine.Predict(make([]float64, 42))
	if err != nil {
		t.Errorf("Fail-open should not error: %v", err)
	}
	if pred.Direction != "HOLD" {
		t.Errorf("Fail-open should return HOLD: got %s", pred.Direction)
	}
	if pred.Confidence != 0 {
		t.Errorf("Fail-open should return 0 confidence: got %f", pred.Confidence)
	}
}
