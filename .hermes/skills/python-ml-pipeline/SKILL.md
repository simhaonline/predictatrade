---
name: python-ml-pipeline
description: "Train, validate, and export ML models as ONNX for Go."
---

# python-ml-pipeline

Use when training ML models or debugging the Predict-A-Trade ML pipeline.

## Pipeline
1. Data: research/scripts/train_ml_model.py
2. Features: 42 across 13 pillars
3. Training: XGBoost + LSTM
4. Export: models/xgb_model.onnx, models/lstm_model.onnx, scaler.json
5. Version: models/model_version.txt

## Commands
cd research && source .venv/bin/activate
python scripts/train_ml_model.py
python scripts/verify_math_parity.py
python scripts/oos_walkforward_calibrate.py

## Dependencies
numpy>=2.0, pandas>=2.2, scikit-learn>=1.5, scipy>=1.14
Dev: pytest>=8.0, pytest-cov>=5.0, ruff>=0.6

## Validation
42 features, no NaN, normalized per scaler.json
ONNX loads in Go via yalue/onnxruntime_go
Go output == Python output +/- 1e-6
OOS walk-forward with locked out-of-sample periods
