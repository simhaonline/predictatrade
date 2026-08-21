// Package mlengine — Atomic session reload for hot model updates.
package mlengine

import (
	"fmt"
	"path/filepath"
	"sync"
)

var reloadMu sync.Mutex

func (e *MLEngine) Reload() {
	if e == nil || !e.enabled {
		return
	}

	reloadMu.Lock()
	defer reloadMu.Unlock()

	// Load new sessions outside the write lock
	newXgb, xgbErr := createModelSession(filepath.Join(e.modelsDir, "xgb_model.onnx"), len(e.featureCols))
	newLstm, lstmErr := createModelSession(filepath.Join(e.modelsDir, "lstm_model.onnx"), len(e.featureCols))
	if lstmErr != nil {
		newLstm = nil
	}
	if xgbErr != nil {
		fmt.Printf("[mlengine] Reload failed: XGBoost error: %v\n", xgbErr)
		return
	}

	scaler, scalerErr := loadScaler(filepath.Join(e.modelsDir, "scaler.json"))
	if scalerErr != nil {
		scaler = e.scaler
	}

	featureCols, fcErr := loadFeatureColumns(filepath.Join(e.modelsDir, "feature_columns.json"))
	if fcErr != nil {
		featureCols = e.featureCols
	}

	// Atomic swap
	e.mu.Lock()
	oldXgb := e.xgbSession
	oldLstm := e.lstmSession
	e.xgbSession = newXgb
	e.lstmSession = newLstm
	e.scaler = scaler
	e.featureCols = featureCols
	e.mu.Unlock()

	oldXgb.destroy()
	oldLstm.destroy()

	fmt.Printf("[mlengine] Models reloaded: XGB=%v LSTM=%v features=%d\n",
		newXgb != nil, newLstm != nil, len(featureCols))
}
