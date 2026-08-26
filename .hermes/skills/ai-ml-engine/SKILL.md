---
name: ai-ml-engine
description: "ML inference, sentiment analysis, RL optimization."
---

# ai-ml-engine

## Components
- ONNX Runtime (via yalue/onnxruntime_go): XGBoost + LSTM models in models/
- Ollama local LLM (real-time sentiment analysis on port 11434)
- RL optimizer in realtime/internal/rl/
- Sentiment engine in realtime/internal/sentiment/
- ML engine pkg in realtime/pkg/mlengine/

## Models
- models/xgb_model.onnx (gradient boosting)
- models/lstm_model.onnx (deep learning)
- models/scaler.json, models/feature_columns.json, models/model_version.txt

## ML Features (42 total)
Trend, momentum, volatility, VWAP, structure, liquidity, SMC, MTF, candle, regime, ML, sentiment pillars.

## Wired in main.go
- MLEngine via pkg/mlengine
- Ollama client via pkg/ollama
- MLContribution + SentimentContribution fields in StrategyResult
