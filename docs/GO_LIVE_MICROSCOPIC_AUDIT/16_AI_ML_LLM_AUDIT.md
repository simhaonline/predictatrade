# 16 — AI / ML / LLM Audit

## Inventory & classification

| Component | Files | Class | Evidence |
|---|---|---|---|
| ONNX inference (XGB+LSTM) | `pkg/mlengine/*` | **WIRED-BUT-INERT** | Runtime log: "ML engine disabled (no models found or ONNX runtime unavailable) — fail-open mode"; `/app/models` absent in container while repo `/models` has artifacts; metric `pat_ml_model_loaded 0`. Even when loaded, results discarded (`main.go:1143-1145`). |
| Ollama sentiment | `pkg/ollama/client.go` | **INERT** | Enabled in env; host `localhost:11434` unreachable from inside container → every call times out → neutral 0.0 fallback. Only consumed by the discarded ML block anyway. |
| Sentiment engine (internal) | `internal/sentiment/engine.go` | DEAD | zero importers |
| RL optimizer | `internal/rl/optimizer.go` | DEAD | no cmd importers; RL_MODE=disabled default |
| Adaptation manager | `internal/adaptation/manager.go` | DEAD | env parsed, never consumed |
| Hedging manager | `internal/hedging/manager.go` | DEAD | HEDGING_ENABLED=false; no importers |
| ML adaptation | `internal/ml/adaptation.go` | DEAD | no importers |
| PTB intelligence (19 modules) | `internal/ptb/*` | WIRED — SHADOW, weight 0 | honest `ScoreContrib: decimal.Zero // ALWAYS ZERO in SHADOW mode`; capability labels `UNSUPPORTED_BY_DATA_SOURCE` |
| Breakout/OCO engines | pkg | DEAD | `NEWS_BREAKOUT_ENABLED=true` parsed but nothing instantiated |
| Replay research engine | `internal/replay/engine.go` | OFFLINE ONLY | SYNTHETIC_REPLAY labeled; not imported live |
| Calibration models | internal/calibration | SIMULATED metadata | untrained sigmoids stamped VALIDATED (F-006) |

## Model validation (§27)

`models/` contains xgb/lstm onnx + scaler + feature_columns + version file; training pipeline exists (`research/ml_training.py`, weekly cron Sun 02:00). However: **no walk-forward/OOS/calibration evidence files were found for the shipped artifacts**; feature vector (42 cols) matches `feature_columns.json`; train/infer skew unprovable without training manifests ⇒ UNVERIFIED. Since outputs are discarded, live impact is nil — but dashboards/config advertise AI activity that does not influence signals (claim-validation failure, §114).

## LLM specifics

Provider: local Ollama (`deepseek-v4-pro:cloud` label). No external LLM API keys found. Prompt-injection surface: sentiment input includes news text — output clamped ±1.0 and currently discarded; if ever wired, structured validation exists (JSON extraction + clamp) but no schema enforcement beyond numeric clamp. No PII sent. Cost controls n/a (local).

**Verdict: AI/ML = SIMULATED/INERT with misleading configuration.** Financial decisions do not depend on it today — which is the only reason this is P1 not P0.
