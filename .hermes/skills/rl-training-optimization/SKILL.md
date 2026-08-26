---
name: rl-training-optimization
description: "RL training, optimization, and ONNX deployment."
---

# rl-training-optimization

Use for PAT reinforcement learning.

Location: research/src/patresearch/rl_training.py, scripts/run_training.sh

Pipeline: state (42 features + regime) → action (hold/buy/sell/close) → reward (risk-adjusted return) → train (PPO/SAC) → validate (OOS) → export (ONNX)

Validation: no leakage, Sharpe > 1.0 OOS, max DD < 20%, realistic frequency

Deploy: ONNX → yalue/onnxruntime_go, parity verified
