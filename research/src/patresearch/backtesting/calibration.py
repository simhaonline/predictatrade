"""Probability calibration for strategy signals.

Maps a raw strategy score (0-100 scale) to a calibrated win probability for a
named prediction target / exit profile. Two methods are supported:

- ``logistic`` — Platt scaling: sigmoid(a * x + b) where x = clamp(score,0,100)/100.
  Fitted via ``sklearn.linear_model.LogisticRegression`` when scikit-learn is
  importable; otherwise a tiny dependency-free Newton-Raphson GLM fit is used.
- ``isotonic`` — monotonic step/linear function via
  ``sklearn.isotonic.IsotonicRegression`` when available; otherwise a PAVA
  (pool adjacent violators) binned monotonic fallback.

The module is dependency-light: it never crashes if scikit-learn is missing.

Canonical normalization (MUST match the Go consumer in
realtime/internal/calibration):
    x = clamp(score, 0, 100) / 100.0
so the same params reproduce the same probability on both planes.

Exported JSON schema (see ``export_json``):
    {
      "version": "1.0.0",
      "strategy": str,
      "target": str,            # e.g. "TP1_HIT"
      "exit_profile": str,      # e.g. "TP1" / "TRAILING"
      "oos_auc": float,         # AUC of calibrated probs vs labels
      "n_samples": int,
      "method": "logistic" | "isotonic",
      "params": {"a": float, "b": float},   # logistic
      "trained_at": str,
      "monotonic_bins": [{"x": float, "p": float}, ...],  # isotonic
      "x_scale": 100.0,
      "x_clip": [0.0, 100.0]
    }
"""
from __future__ import annotations

import json
import math
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Dict, List, Optional, Sequence, Tuple

SCHEMA_VERSION = "1.0.0"

SKLEARN_AVAILABLE = False
try:  # pragma: no cover - import guard
    from sklearn.isotonic import IsotonicRegression
    from sklearn.linear_model import LogisticRegression
    from sklearn.metrics import roc_auc_score

    SKLEARN_AVAILABLE = True
except Exception:  # pragma: no cover - fallback path
    SKLEARN_AVAILABLE = False


def _sigmoid(z: float) -> float:
    if z >= 0:
        ez = math.exp(-z)
        return 1.0 / (1.0 + ez)
    # numerically stable for large negative z
    ez = math.exp(z)
    return ez / (1.0 + ez)


def _clamp(v: float, lo: float, hi: float) -> float:
    return lo if v < lo else (hi if v > hi else v)


def _norm(score: float) -> float:
    """Canonical score -> [0,1] normalization shared with the Go consumer."""
    return _clamp(float(score), 0.0, 100.0) / 100.0


class ProbabilityCalibrator:
    """Fit and apply a probability calibrator for a single (strategy, target)."""

    def __init__(self, method: str = "logistic"):
        if method not in ("logistic", "isotonic"):
            raise ValueError(f"unknown calibration method: {method!r}")
        self.method = method
        self.a: float = 0.0
        self.b: float = 0.0
        self.bins: List[Tuple[float, float]] = []  # (x, p) sorted by x, monotonic
        self.oos_auc: float = float("nan")
        self.n_samples: int = 0
        self.trained_at: str = ""
        self.version: str = SCHEMA_VERSION

    # ─── Fitting ──────────────────────────────────────────────────────────

    def fit(self, scores: Sequence[float], labels: Sequence[int]) -> "ProbabilityCalibrator":
        scores = [float(s) for s in scores]
        labels = [int(l) for l in labels]
        if len(scores) != len(labels):
            raise ValueError("scores and labels length mismatch")
        self.n_samples = len(scores)
        xs = [_norm(s) for s in scores]
        if self.method == "logistic":
            self._fit_logistic(xs, labels)
        else:
            self._fit_isotonic(xs, labels)
        try:
            self.oos_auc = self._auc(xs, labels)
        except Exception:
            self.oos_auc = float("nan")
        self.trained_at = datetime.now(timezone.utc).isoformat()
        return self

    def _fit_logistic(self, xs: List[float], labels: List[int]) -> None:
        if SKLEARN_AVAILABLE:
            model = LogisticRegression(C=1e6, fit_intercept=True, max_iter=1000)
            model.fit([[x] for x in xs], labels)
            self.a = float(model.coef_[0][0])
            self.b = float(model.intercept_[0])
            return
        # Dependency-free Newton-Raphson GLM (logistic regression w/ intercept).
        a, b = 0.0, 0.0
        for _ in range(100):
            g_a = 0.0
            g_b = 0.0
            h_aa = 0.0
            h_ab = 0.0
            h_bb = 0.0
            for x, y in zip(xs, labels):
                p = _sigmoid(a * x + b)
                g_a += (p - y) * x
                g_b += (p - y)
                w = p * (1.0 - p)
                h_aa += w * x * x
                h_ab += w * x
                h_bb += w
            det = h_aa * h_bb - h_ab * h_ab
            if det <= 1e-12:
                break  # separable -> Hessian singular; coefficients already finite
            da = (h_bb * g_a - h_ab * g_b) / det
            db = (-h_ab * g_a + h_aa * g_b) / det
            a -= da
            b -= db
            if abs(da) < 1e-8 and abs(db) < 1e-8:
                break
        self.a = a
        self.b = b

    def _fit_isotonic(self, xs: List[float], labels: List[int]) -> None:
        if SKLEARN_AVAILABLE:
            model = IsotonicRegression(
                y_min=0.0, y_max=1.0, increasing=True, out_of_bounds="clip"
            )
            model.fit(xs, labels)
            # Use unique input x-values and the model's predictions as knots.
            # (Avoids reliance on sklearn-internal attributes removed in 1.9+.)
            ux = sorted(set(xs))
            pairs = [(float(x), float(model.predict([x])[0])) for x in ux]
            self.bins = _dedupe_bins(pairs)
            return
        # PAVA fallback on binned empirical positive rates.
        n_bins = 20
        edges = [i / n_bins for i in range(n_bins + 1)]
        sums = [0.0] * n_bins
        counts = [0] * n_bins
        for x, y in zip(xs, labels):
            idx = min(n_bins - 1, int(x * n_bins))
            sums[idx] += y
            counts[idx] += 1
        centers = [(edges[i] + edges[i + 1]) / 2.0 for i in range(n_bins)]
        rates = [
            (sums[i] / counts[i]) if counts[i] > 0 else 0.5 for i in range(n_bins)
        ]
        self.bins = _pava(list(zip(centers, rates)))

    # ─── Prediction ───────────────────────────────────────────────────────

    def predict_proba(self, score: float) -> float:
        """Calibrated probability in [0,1] for a raw score. Monotonic in score."""
        x = _norm(score)
        if self.method == "logistic":
            return _sigmoid(self.a * x + self.b)
        return self._interp_bins(x)

    def _interp_bins(self, x: float) -> float:
        if not self.bins:
            return 0.5
        if x <= self.bins[0][0]:
            return _clamp(self.bins[0][1], 0.0, 1.0)
        if x >= self.bins[-1][0]:
            return _clamp(self.bins[-1][1], 0.0, 1.0)
        for i in range(1, len(self.bins)):
            x0, p0 = self.bins[i - 1]
            x1, p1 = self.bins[i]
            if x <= x1:
                if x1 == x0:
                    return _clamp(p0, 0.0, 1.0)
                t = (x - x0) / (x1 - x0)
                return _clamp(p0 + t * (p1 - p0), 0.0, 1.0)
        return _clamp(self.bins[-1][1], 0.0, 1.0)

    # ─── AUC ──────────────────────────────────────────────────────────────

    def _auc(self, xs: List[float], labels: List[int]) -> float:
        probs = [self._proba_from_x(x) for x in xs]
        if len(set(labels)) < 2:
            return float("nan")
        if SKLEARN_AVAILABLE:
            return float(roc_auc_score(labels, probs))
        return _manual_auc(labels, probs)

    def _proba_from_x(self, x: float) -> float:
        if self.method == "logistic":
            return _sigmoid(self.a * x + self.b)
        return self._interp_bins(x)

    # ─── Serialization ────────────────────────────────────────────────────

    def export_json(self, path: str, metadata: Dict) -> None:
        data = {
            "version": self.version,
            "strategy": metadata.get("strategy", ""),
            "target": metadata.get("target", ""),
            "exit_profile": metadata.get("exit_profile", ""),
            "oos_auc": self.oos_auc,
            "n_samples": self.n_samples,
            "method": self.method,
            "params": {"a": self.a, "b": self.b},
            "trained_at": self.trained_at,
            "monotonic_bins": [{"x": x, "p": p} for x, p in self.bins],
            "x_scale": 100.0,
            "x_clip": [0.0, 100.0],
            "metadata": metadata,
        }
        with open(path, "w") as f:
            json.dump(data, f, indent=2)

    @classmethod
    def load_json(cls, path: str) -> "ProbabilityCalibrator":
        with open(path) as f:
            data = json.load(f)
        method = data.get("method", "logistic")
        cal = cls(method=method)
        cal.version = data.get("version", SCHEMA_VERSION)
        cal.oos_auc = float(data.get("oos_auc", "nan"))
        cal.n_samples = int(data.get("n_samples", 0))
        cal.trained_at = data.get("trained_at", "")
        params = data.get("params", {}) or {}
        cal.a = float(params.get("a", 0.0))
        cal.b = float(params.get("b", 0.0))
        bins = data.get("monotonic_bins") or []
        cal.bins = [(float(b["x"]), float(b["p"])) for b in bins]
        return cal


# ─── Helpers ───────────────────────────────────────────────────────────────


def _dedupe_bins(pairs: List[Tuple[float, float]]) -> List[Tuple[float, float]]:
    out: List[Tuple[float, float]] = []
    for x, p in pairs:
        if out and abs(out[-1][0] - x) < 1e-12:
            continue
        out.append((x, _clamp(p, 0.0, 1.0)))
    if not out:
        out = [(0.0, 0.5), (1.0, 0.5)]
    return out


def _pava(pairs: List[Tuple[float, float]]) -> List[Tuple[float, float]]:
    """Pool adjacent violators: enforce non-decreasing p over sorted x."""
    items = sorted(pairs)
    merged: List[Tuple[float, float]] = []
    for x, p in items:
        if merged and abs(merged[-1][0] - x) < 1e-12:
            merged[-1] = (x, (merged[-1][1] + p) / 2.0)
        else:
            merged.append((x, p))
    blocks = [[m] for m in merged]
    while True:
        changed = False
        i = 0
        while i < len(blocks) - 1:
            cur = sum(p for _, p in blocks[i]) / len(blocks[i])
            nxt = sum(p for _, p in blocks[i + 1]) / len(blocks[i + 1])
            if cur > nxt:
                blocks[i] += blocks[i + 1]
                del blocks[i + 1]
                changed = True
            else:
                i += 1
        if not changed:
            break
    out = []
    for blk in blocks:
        avg = sum(p for _, p in blk) / len(blk)
        for x, _ in blk:
            out.append((x, _clamp(avg, 0.0, 1.0)))
    return sorted(out)


def _manual_auc(labels: List[int], probs: List[float]) -> float:
    """Rank-based AUC (Harrell's C) without scikit-learn."""
    pos = [p for p, y in zip(probs, labels) if y == 1]
    neg = [p for p, y in zip(probs, labels) if y == 0]
    if not pos or not neg:
        return float("nan")
    total = 0.0
    for pp in pos:
        for np_ in neg:
            if pp > np_:
                total += 1.0
            elif pp == np_:
                total += 0.5
    return total / (len(pos) * len(neg))
