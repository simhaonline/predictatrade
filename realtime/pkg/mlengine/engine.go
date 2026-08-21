// Package mlengine provides ONNX-based ML inference for the Predict-A-Trade
// strategy scorer. It loads XGBoost and LSTM ONNX models, runs concurrent
// inference, and combines predictions with a weighted ensemble.
//
// The engine is fail-open: if ML inference fails or confidence is below
// threshold, ML contribution is set to zero and trading continues normally.
package mlengine

import (
	"context"
	"strings"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yalue/onnxruntime_go"
)

// Prediction is the output of a single ML inference call.
type Prediction struct {
	Direction     string     `json:"direction"`
	Confidence    float64    `json:"confidence"`
	Probabilities [3]float64 `json:"probabilities"`
	LatencyMs     float64    `json:"latency_ms"`
	ModelVersion  string     `json:"model_version"`
}

// modelSession wraps an ONNX session with its persistent input/output tensors.
// A mutex protects concurrent access during Run().
type modelSession struct {
	mu           sync.Mutex
	session      *onnxruntime_go.AdvancedSession
	inputTensor  *onnxruntime_go.Tensor[float32]
	outputTensor *onnxruntime_go.Tensor[float32]
}

func (ms *modelSession) predict(features []float64) ([3]float64, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	inputData := ms.inputTensor.GetData()
	n := len(features)
	if n > len(inputData) {
		n = len(inputData)
	}
	for i := 0; i < n; i++ {
		inputData[i] = float32(features[i])
	}

	if err := ms.session.Run(); err != nil {
		return [3]float64{}, fmt.Errorf("inference: %w", err)
	}

	var probs [3]float64
	outData := ms.outputTensor.GetData()
	for i := 0; i < 3 && i < len(outData); i++ {
		probs[i] = float64(outData[i])
	}
	return probs, nil
}

func (ms *modelSession) destroy() {
	if ms == nil {
		return
	}
	if ms.session != nil {
		ms.session.Destroy()
	}
	if ms.inputTensor != nil {
		ms.inputTensor.Destroy()
	}
	if ms.outputTensor != nil {
		ms.outputTensor.Destroy()
	}
}

// MLEngine holds ONNX sessions, scaler, feature metadata, and provides
// thread-safe inference with hot-reload support.
type MLEngine struct {
	mu          sync.RWMutex
	xgbSession  *modelSession
	lstmSession *modelSession
	scaler      *Scaler
	featureCols []string
	modelsDir   string
	enabled     bool
	xgbWeight   float64
	lstmWeight  float64
	modelVersion string
}

// Scaler is a StandardScaler loaded from JSON (scaler.json).
type Scaler struct {
	Mean  []float64
	Scale []float64
}

// scalerJSON is the JSON format produced by Python training/bootstrap.
type scalerJSON struct {
	Mean      []float64 `json:"mean"`
	Scale     []float64 `json:"scale"`
	NFeatures int       `json:"n_features"`
}

// NewMLEngine initializes the ML inference engine (fail-open if unavailable).
func NewMLEngine(modelsDir string) *MLEngine {
	engine := &MLEngine{
		modelsDir:  modelsDir,
		xgbWeight:  0.6,
		lstmWeight: 0.4,
		enabled:    false,
	}

	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		return engine
	}

	if !onnxruntime_go.IsInitialized() {
		// Set shared library path from ONNXRUNTIME_LIB env var
		if libPath := os.Getenv("ONNXRUNTIME_LIB"); libPath != "" {
			onnxruntime_go.SetSharedLibraryPath(libPath)
		}
		if err := onnxruntime_go.InitializeEnvironment(); err != nil {
			fmt.Printf("[mlengine] ONNX init failed: %v — ML disabled\n", err)
			return engine
		}
	}

	scaler, err := loadScaler(filepath.Join(modelsDir, "scaler.json"))
	if err != nil {
		fmt.Printf("[mlengine] Scaler load failed: %v — ML disabled\n", err)
		return engine
	}
	engine.scaler = scaler

	featureCols, err := loadFeatureColumns(filepath.Join(modelsDir, "feature_columns.json"))
	if err != nil {
		fmt.Printf("[mlengine] Feature cols load failed: %v — ML disabled\n", err)
		return engine
	}
	engine.featureCols = featureCols

	xgbSess, err := createModelSession(filepath.Join(modelsDir, "xgb_model.onnx"), len(featureCols))
	if err != nil {
		fmt.Printf("[mlengine] XGBoost load failed: %v — ML disabled\n", err)
		return engine
	}
	engine.xgbSession = xgbSess

	lstmSess, err := createModelSession(filepath.Join(modelsDir, "lstm_model.onnx"), len(featureCols))
	if err != nil {
		fmt.Printf("[mlengine] LSTM load failed: %v — XGBoost-only mode\n", err)
		engine.lstmSession = nil
	} else {
		engine.lstmSession = lstmSess
	}

	if data, err := os.ReadFile(filepath.Join(modelsDir, "model_version.txt")); err == nil {
		engine.modelVersion = string(data)
	} else {
		engine.modelVersion = "unknown"
	}

	engine.enabled = true
	setModelVersionMetric(engine.modelVersion)
	fmt.Printf("[mlengine] Enabled: XGB=%v LSTM=%v features=%d version=%s\n",
		engine.xgbSession != nil, engine.lstmSession != nil, len(featureCols), engine.modelVersion)
	return engine
}

// IsEnabled returns true if ML inference is active.
func (e *MLEngine) IsEnabled() bool {
	return e != nil && e.enabled
}

// SetEnabled enables or disables ML inference at runtime.
// When disabled, Predict() returns HOLD/0 (fail-open).
func (e *MLEngine) SetEnabled(enabled bool) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// Predict runs concurrent XGBoost+LSTM inference, enforces 15ms timeout.
// Fail-open: returns HOLD/0 if inference fails or confidence < 30%.
func (e *MLEngine) Predict(features []float64) (*Prediction, error) {
	if !e.IsEnabled() {
		return &Prediction{Direction: "HOLD", Confidence: 0}, nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	start := time.Now()

	if len(features) != len(e.featureCols) {
		return &Prediction{Direction: "HOLD", Confidence: 0},
			fmt.Errorf("feature count mismatch: got %d, expected %d", len(features), len(e.featureCols))
	}

	scaled := e.scaleFeatures(features)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()

	var xgbProbs, lstmProbs [3]float64
	var xgbErr, lstmErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			xgbErr = ctx.Err()
		default:
			xgbProbs, xgbErr = e.runXGBoost(scaled)
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			lstmErr = ctx.Err()
		default:
			lstmProbs, lstmErr = e.runLSTM(scaled)
		}
	}()

	wg.Wait()
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	pred := &Prediction{LatencyMs: latency, ModelVersion: e.modelVersion}

	if xgbErr != nil {
		pred.Direction = "HOLD"
		pred.Confidence = 0
		recordMetrics(pred, xgbErr)
		return pred, xgbErr
	}

	// Weighted ensemble
	effXgbW, effLstmW := e.xgbWeight, e.lstmWeight
	if e.lstmSession == nil || lstmErr != nil {
		effXgbW, effLstmW = 1.0, 0
	}
	var combined [3]float64
	for i := 0; i < 3; i++ {
		combined[i] = effXgbW*xgbProbs[i] + effLstmW*lstmProbs[i]
	}

	maxIdx, maxProb := 0, combined[0]
	for i := 1; i < 3; i++ {
		if combined[i] > maxProb {
			maxProb = combined[i]
			maxIdx = i
		}
	}

	direction := "HOLD"
	if maxIdx == 1 {
		direction = "BUY"
	} else if maxIdx == 2 {
		direction = "SELL"
	}
	confidence := maxProb * 100

	pred.Direction = direction
	pred.Confidence = confidence
	pred.Probabilities = combined

	if confidence < 30.0 {
		pred.Direction = "HOLD"
		pred.Confidence = 0
	}

	recordMetrics(pred, nil)
	return pred, nil
}

func (e *MLEngine) runXGBoost(features []float64) ([3]float64, error) {
	if e.xgbSession == nil {
		return [3]float64{0.33, 0.33, 0.34}, nil
	}
	return e.xgbSession.predict(features)
}

func (e *MLEngine) runLSTM(features []float64) ([3]float64, error) {
	if e.lstmSession == nil {
		return [3]float64{0.33, 0.33, 0.34}, nil
	}
	return e.lstmSession.predict(features)
}

func (e *MLEngine) scaleFeatures(features []float64) []float64 {
	if e.scaler == nil {
		return features
	}
	scaled := make([]float64, len(features))
	for i := 0; i < len(features); i++ {
		if i < len(e.scaler.Scale) && e.scaler.Scale[i] != 0 {
			scaled[i] = (features[i] - e.scaler.Mean[i]) / e.scaler.Scale[i]
		} else {
			scaled[i] = features[i]
		}
	}
	return scaled
}

func createModelSession(modelPath string, numFeatures int) (*modelSession, error) {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model not found: %s", modelPath)
	}

	// Inspect ONNX model to auto-detect input/output names
	inputInfos, outputInfos, err := onnxruntime_go.GetInputOutputInfo(modelPath)
	if err != nil {
		return nil, fmt.Errorf("inspect ONNX model: %w", err)
	}

	// Auto-detect input name: prefer "input", fallback to first input
	inputName := "input"
	if len(inputInfos) > 0 {
		found := false
		for _, info := range inputInfos {
			if info.Name == "input" {
				found = true
				break
			}
		}
		if !found {
			inputName = inputInfos[0].Name
		}
	}

	// Auto-detect output name: prefer "output", then "output_probability", then first output
	outputName := "output"
	if len(outputInfos) > 0 {
		found := false
		for _, info := range outputInfos {
			if info.Name == "output" {
				found = true
				break
			}
		}
		if !found {
			for _, info := range outputInfos {
				if info.Name == "output_probability" {
					outputName = "output_probability"
					found = true
					break
				}
			}
		}
		if !found {
			outputName = outputInfos[0].Name
		}
	}

	inputShape := onnxruntime_go.NewShape(1, int64(numFeatures))
	inputData := make([]float32, numFeatures)
	inputTensor, err := onnxruntime_go.NewTensor(inputShape, inputData)
	if err != nil {
		return nil, fmt.Errorf("input tensor: %w", err)
	}

	outputShape := onnxruntime_go.NewShape(1, 3)
	outputData := make([]float32, 3)
	outputTensor, err := onnxruntime_go.NewTensor(outputShape, outputData)
	if err != nil {
		inputTensor.Destroy()
		return nil, fmt.Errorf("output tensor: %w", err)
	}

	inputs := []onnxruntime_go.Value{inputTensor}
	outputs := []onnxruntime_go.Value{outputTensor}
	session, err := onnxruntime_go.NewAdvancedSession(
		modelPath,
		[]string{inputName}, []string{outputName},
		inputs, outputs, nil,
	)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("session: %w (input=%s output=%s)", err, inputName, outputName)
	}

	return &modelSession{
		session:      session,
		inputTensor:  inputTensor,
		outputTensor: outputTensor,
	}, nil
}

// loadScaler reads scaler.json (JSON format, not gob).
func loadScaler(path string) (*Scaler, error) {
	// Try scaler.json first (preferred format)
	jsonPath := strings.Replace(path, ".gob", ".json", 1)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		// Fallback: try the exact path provided
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read scaler: %w", err)
		}
	}

	var sj scalerJSON
	if err := json.Unmarshal(data, &sj); err != nil {
		return nil, fmt.Errorf("parse scaler JSON: %w", err)
	}
	if len(sj.Mean) == 0 || len(sj.Scale) == 0 {
		return nil, fmt.Errorf("scaler JSON has empty mean/scale arrays")
	}
	return &Scaler{Mean: sj.Mean, Scale: sj.Scale}, nil
}

func loadFeatureColumns(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var cols []string
	if err := json.Unmarshal(data, &cols); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return cols, nil
}

// Close releases all ONNX sessions and resources.
func (e *MLEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.xgbSession.destroy()
	e.lstmSession.destroy()
	e.xgbSession = nil
	e.lstmSession = nil
	e.enabled = false
}
