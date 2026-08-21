#!/usr/bin/env python3
"""
Predict-A-Trade — Bootstrap ML Artifacts (No Training Required)

Creates production-ready ML artifacts so the Go engine can initialize
with ML_ENABLED=true, even before full training is run.

1. Scaler Conversion: scaler.pkl → scaler.json (or mock if no pkl exists)
2. Dummy ONNX Models: minimal XGBoost + LSTM ONNX files for fallback
3. Feature Columns: writes feature_columns.json if missing

Usage: python3 scripts/bootstrap_artifacts.py
"""
import json
import os
import sys
import pickle
import logging
import numpy as np

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger("bootstrap")

MODELS_DIR = os.environ.get("MODELS_DIR", os.path.join(os.path.dirname(__file__), "..", "models"))
MODELS_DIR = os.path.abspath(MODELS_DIR)

# ─── 1. Scaler Conversion ─────────────────────────────────────────────────────

def convert_or_create_scaler(models_dir: str) -> bool:
    """
    Create models/scaler.json from scaler.pkl (if exists) or a mock scaler.
    Returns True on success.
    """
    scaler_json_path = os.path.join(models_dir, "scaler.json")
    scaler_pkl_path = os.path.join(models_dir, "scaler.pkl")

    # Determine feature count
    num_features = 42  # default
    feature_cols_path = os.path.join(models_dir, "feature_columns.json")
    if os.path.exists(feature_cols_path):
        with open(feature_cols_path) as f:
            cols = json.load(f)
        num_features = len(cols)
        log.info(f"Feature count from feature_columns.json: {num_features}")

    if os.path.exists(scaler_pkl_path):
        # Load pickle scaler and extract mean_ and scale_
        log.info(f"Found scaler.pkl — converting to JSON...")
        try:
            with open(scaler_pkl_path, "rb") as f:
                scaler = pickle.load(f)

            mean = list(scaler.mean_) if hasattr(scaler, "mean_") else [0.0] * num_features
            scale = list(scaler.scale_) if hasattr(scaler, "scale_") else [1.0] * num_features

            # Pad or truncate to num_features
            mean = (mean + [0.0] * num_features)[:num_features]
            scale = (scale + [1.0] * num_features)[:num_features]

            scaler_data = {"mean": mean, "scale": scale, "n_features": num_features}
            with open(scaler_json_path, "w") as f:
                json.dump(scaler_data, f, indent=2)
            log.info(f"  scaler.json written ({num_features} features) from trained scaler.pkl")
            return True
        except Exception as e:
            log.warning(f"  scaler.pkl load failed: {e} — creating mock scaler")

    # No scaler.pkl — create mock scaler (mean=0, scale=1)
    log.info(f"No scaler.pkl found — creating mock scaler ({num_features} features)...")
    scaler_data = {
        "mean": [0.0] * num_features,
        "scale": [1.0] * num_features,
        "n_features": num_features,
    }
    with open(scaler_json_path, "w") as f:
        json.dump(scaler_data, f, indent=2)
    log.info(f"  scaler.json written (mock: mean=0, scale=1, {num_features} features)")
    return True


# ─── 2. Dummy ONNX Models ────────────────────────────────────────────────────

def create_dummy_xgb_onnx(models_dir: str, num_features: int) -> bool:
    """
    Create a minimal 3-class classifier and export as ONNX.
    Uses sklearn GradientBoostingClassifier (native skl2onnx support).
    This is a placeholder — real training replaces this file.
    """
    onnx_path = os.path.join(models_dir, "xgb_model.onnx")
    if os.path.exists(onnx_path):
        log.info(f"  xgb_model.onnx already exists — skipping")
        return True

    log.info("Creating dummy XGBoost ONNX model (via sklearn GradientBoosting)...")
    try:
        from skl2onnx import to_onnx
        from sklearn.ensemble import GradientBoostingClassifier

        # Generate random training data
        np.random.seed(42)
        X = np.random.randn(200, num_features).astype(np.float32)
        y = np.random.randint(0, 3, 200)  # 3 classes: HOLD=0, BUY=1, SELL=2

        # Train a minimal sklearn GradientBoosting model (native ONNX export support)
        model = GradientBoostingClassifier(
            n_estimators=10, max_depth=3, learning_rate=0.1,
            random_state=42,
        )
        model.fit(X, y)

        # Export to ONNX via skl2onnx with explicit input name
        from skl2onnx.common.data_types import FloatTensorType
        onnx_model = to_onnx(
            model,
            initial_types=[("input", FloatTensorType([1, num_features]))],
            target_opset=15,
            name="xgb_model",
        )
        # Rename outputs to 'output' if needed
        for out in onnx_model.graph.output:
            if out.name != "output":
                out.name = "output"

        with open(onnx_path, "wb") as f:
            f.write(onnx_model.SerializeToString())
        log.info(f"  xgb_model.onnx written ({os.path.getsize(onnx_path)} bytes)")
        return True
    except Exception as e:
        log.warning(f"  XGBoost ONNX creation failed: {e}")
        return False


def create_dummy_lstm_onnx(models_dir: str, num_features: int) -> bool:
    """
    Create a minimal LSTM model and export as ONNX.
    Uses a 1-layer LSTM with random weights — placeholder for testing.
    """
    onnx_path = os.path.join(models_dir, "lstm_model.onnx")
    if os.path.exists(onnx_path):
        log.info(f"  lstm_model.onnx already exists — skipping")
        return True

    log.info("Creating dummy LSTM ONNX model...")
    try:
        import torch
        import torch.nn as nn

        class MiniLSTM(nn.Module):
            def __init__(self, input_size, hidden_size=16, num_classes=3):
                super().__init__()
                self.lstm = nn.LSTM(input_size, hidden_size, batch_first=True)
                self.fc = nn.Linear(hidden_size, num_classes)

            def forward(self, x):
                out, _ = self.lstm(x)
                return self.fc(out[:, -1, :])

        model = MiniLSTM(num_features)
        model.eval()

        # Export: input shape [1, 1, num_features] (batch=1, seq=1, features=N)
        dummy_input = torch.randn(1, 1, num_features)
        torch.onnx.export(
            model, dummy_input, onnx_path,
            export_params=True, opset_version=14,
            input_names=["input"], output_names=["output"],
            dynamic_axes={"input": {0: "batch_size"}, "output": {0: "batch_size"}},
            dynamo=False,
        )
        log.info(f"  lstm_model.onnx written ({os.path.getsize(onnx_path)} bytes)")
        return True
    except Exception as e:
        log.warning(f"  LSTM ONNX creation failed: {e}")
        return False


# ─── 3. Feature Columns ──────────────────────────────────────────────────────

def ensure_feature_columns(models_dir: str) -> int:
    """
    Create feature_columns.json if it doesn't exist.
    Returns the number of features.
    """
    path = os.path.join(models_dir, "feature_columns.json")
    if os.path.exists(path):
        with open(path) as f:
            cols = json.load(f)
        log.info(f"  feature_columns.json exists ({len(cols)} features)")
        return len(cols)

    # Default 42 features matching Go engine indicators
    default_features = [
        "ema9", "ema21", "ema50", "ema100", "ema200", "ema_cross_9_21",
        "sma50", "sma100", "sma200",
        "macd_main", "macd_signal", "macd_histogram", "macd_bull_cross", "macd_bear_cross",
        "adx", "adx_plus_di", "adx_minus_di",
        "rsi", "stoch_main", "stoch_signal",
        "stoch_rsi", "stoch_rsi_k", "stoch_rsi_d", "cci",
        "atr", "boll_upper", "boll_middle", "boll_lower", "boll_width",
        "boll_bull_rev", "boll_bear_rev", "obv", "vwap",
        "psar", "psar_long",
        "ichimoku_tenkan", "ichimoku_kijun", "ichimoku_senkou_a", "ichimoku_senkou_b",
        "session", "is_overlap",
    ]
    # Pad to 42
    while len(default_features) < 42:
        default_features.append(f"feature_{len(default_features)}")

    with open(path, "w") as f:
        json.dump(default_features[:42], f, indent=2)
    log.info(f"  feature_columns.json written ({len(default_features[:42])} features)")
    return len(default_features[:42])


# ─── Main ────────────────────────────────────────────────────────────────────

def main():
    log.info("=" * 60)
    log.info("Predict-A-Trade ML Artifact Bootstrap")
    log.info(f"Models dir: {MODELS_DIR}")
    log.info("=" * 60)

    os.makedirs(MODELS_DIR, exist_ok=True)

    # Step 1: Feature columns
    log.info("Step 1: Ensuring feature_columns.json...")
    num_features = ensure_feature_columns(MODELS_DIR)

    # Step 2: Scaler
    log.info("Step 2: Creating scaler.json...")
    convert_or_create_scaler(MODELS_DIR)

    # Step 3: Dummy ONNX models
    log.info("Step 3: Creating dummy ONNX models...")
    create_dummy_xgb_onnx(MODELS_DIR, num_features)
    create_dummy_lstm_onnx(MODELS_DIR, num_features)

    # Step 4: Model version
    version_path = os.path.join(MODELS_DIR, "model_version.txt")
    with open(version_path, "w") as f:
        f.write("bootstrap-v1.0.0\n")
    log.info(f"  model_version.txt written")

    # Verify
    log.info("")
    log.info("Verification:")
    for fname in ["feature_columns.json", "scaler.json", "xgb_model.onnx", "lstm_model.onnx", "model_version.txt"]:
        fpath = os.path.join(MODELS_DIR, fname)
        exists = os.path.exists(fpath)
        size = os.path.getsize(fpath) if exists else 0
        status = "✅" if exists else "❌"
        log.info(f"  {status} {fname}: {size} bytes")

    log.info("")
    log.info("Bootstrap complete. Go engine can now start with ML_ENABLED=true.")
    log.info("=" * 60)


if __name__ == "__main__":
    main()
