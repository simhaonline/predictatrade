"""Tests for the probability calibration module.

Core honesty rules (AGENTS.md):
- Calibration is computed ONLY from real scores/labels.
- We never assert exact probabilities for fabricated data; we assert the
  structural guarantees the math must satisfy (separability + monotonicity).
"""
from __future__ import annotations

import json
import os
import sys
import tempfile

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "..", "src"))

from patresearch.backtesting.calibration import ProbabilityCalibrator


def _separable_dataset():
    """A perfectly separable synthetic dataset: label 1 scores ~90, label 0 ~10."""
    scores = []
    labels = []
    for _ in range(50):
        scores.append(90.0)
        labels.append(1)
    for _ in range(50):
        scores.append(10.0)
        labels.append(0)
    for i in range(50):
        scores[i] += (i % 5) - 2.0  # 88..92
    for i in range(50):
        scores[50 + i] += (i % 5) - 2.0  # 8..12
    return scores, labels


def test_logistic_separable_extremes():
    scores, labels = _separable_dataset()
    cal = ProbabilityCalibrator(method="logistic").fit(scores, labels)
    p_high = cal.predict_proba(90.0)
    p_low = cal.predict_proba(10.0)
    assert p_high >= 0.95, f"label=1 score should map near 1.0, got {p_high}"
    assert p_low <= 0.05, f"label=0 score should map near 0.0, got {p_low}"


def test_isotonic_separable_extremes():
    scores, labels = _separable_dataset()
    cal = ProbabilityCalibrator(method="isotonic").fit(scores, labels)
    p_high = cal.predict_proba(90.0)
    p_low = cal.predict_proba(10.0)
    assert p_high >= 0.9, f"label=1 score should map near 1.0, got {p_high}"
    assert p_low <= 0.1, f"label=0 score should map near 0.0, got {p_low}"


def test_predict_proba_monotonic():
    scores, labels = _separable_dataset()
    for method in ("logistic", "isotonic"):
        cal = ProbabilityCalibrator(method=method).fit(scores, labels)
        grid = [float(s) for s in range(0, 101, 5)]
        probs = [cal.predict_proba(s) for s in grid]
        for i in range(1, len(probs)):
            assert probs[i] >= probs[i - 1] - 1e-9, (
                f"{method} probability not monotonic at score {grid[i]}"
            )


def test_export_load_roundtrip():
    scores, labels = _separable_dataset()
    cal = ProbabilityCalibrator(method="logistic").fit(scores, labels)
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "cal.json")
        cal.export_json(
            path,
            metadata={"strategy": "STANDARD_SCALPING", "target": "TP1_HIT", "exit_profile": "TP1"},
        )
        assert os.path.exists(path)
        with open(path) as f:
            raw = json.load(f)
        assert raw["version"] == "1.0.0"
        assert raw["method"] == "logistic"
        assert "a" in raw["params"] and "b" in raw["params"]

        loaded = ProbabilityCalibrator.load_json(path)
        assert abs(loaded.predict_proba(90.0) - cal.predict_proba(90.0)) < 1e-9
        assert abs(loaded.predict_proba(10.0) - cal.predict_proba(10.0)) < 1e-9
