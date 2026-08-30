# PAT ML Models — Provenance Notice

The ONNX files in this directory (xgb_model.onnx, lstm_model.onnx) are **bootstrap placeholders** (size ~710 bytes, not production-trained weights). They exist so the ML inference path initializes in sandbox/paper mode and emits honest NON_AI_VERIFIED / NOT_AI_VERIFIED signals.

They are NOT real-v1.0.0 production models. Replace with genuinely trained, validated (OOS AUC > 0.52, n >= 100) models before any live promotion. The engine exposes the version string only as a metric; it does not hard-match, so relabeling here is safe.
