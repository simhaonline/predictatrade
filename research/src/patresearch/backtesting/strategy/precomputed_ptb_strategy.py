"""Precomputed PTB replay strategy.

Uses pre-computed PTB features to avoid re-running the full PTB on every bar.
Validated against live-style PTB run for equivalence.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import List, Optional, Dict
import json
import hashlib

from ..data.loader import HistoricalCandle
from ..data.alignment import TimeframeAlignment
from ..engine.events import SignalEvent
from ..engine.portfolio import Portfolio
from .base import BaseStrategy


@dataclass
class PrecomputedFeatures:
    """Pre-computed PTB features for a single bar."""
    timestamp: datetime
    regime: str = ""
    volatility_state: str = ""
    manipulation_index: float = 0.0
    confluence_score: float = 0.0
    bias: str = "NEUTRAL"
    bias_strength: float = 0.0
    confidence: float = 0.0
    setup_quality: str = ""
    action: str = ""
    stop_distance_multiplier: float = 1.0
    position_size_multiplier: float = 1.0
    entry_zone_low: float = 0.0
    entry_zone_high: float = 0.0
    stop_loss: float = 0.0
    take_profit: float = 0.0
    atr: float = 0.0


@dataclass
class FeatureDatasetMetadata:
    """Metadata for a precomputed feature dataset."""
    symbol: str = "XAUUSD"
    timeframe: str = "M5"
    start_timestamp: str = ""
    end_timestamp: str = ""
    data_version_hash: str = ""
    ptb_model_version: str = "1.0.0"
    config_hash: str = ""
    source_commit_sha: str = ""
    creation_timestamp: str = ""
    feature_schema_version: str = "1.0"
    feature_count: int = 0


class PrecomputedPTBStrategy(BaseStrategy):
    """Replay strategy using precomputed PTB features.

    Avoids re-running expensive PTB calculations on every bar.
    Must be validated against live PTB for equivalence.
    """

    def __init__(self, strategy_id: str = "STANDARD_SCALPING",
                 features: Optional[List[PrecomputedFeatures]] = None,
                 metadata: Optional[FeatureDatasetMetadata] = None):
        self._strategy_id = strategy_id
        self._features: Dict[datetime, PrecomputedFeatures] = {}
        self._metadata = metadata or FeatureDatasetMetadata()
        if features:
            self.load_features(features)

    @property
    def strategy_id(self) -> str:
        return self._strategy_id

    def load_features(self, features: List[PrecomputedFeatures]):
        """Load precomputed features indexed by timestamp."""
        self._features = {f.timestamp: f for f in features}

    def load_from_json(self, filepath: str):
        """Load features from JSON file."""
        with open(filepath, "r") as f:
            data = json.load(f)

        meta = data.get("metadata", {})
        self._metadata = FeatureDatasetMetadata(**meta)

        features = []
        for item in data.get("features", []):
            ts = datetime.fromisoformat(item["timestamp"])
            features.append(PrecomputedFeatures(
                timestamp=ts,
                regime=item.get("regime", ""),
                volatility_state=item.get("volatility_state", ""),
                manipulation_index=item.get("manipulation_index", 0.0),
                confluence_score=item.get("confluence_score", 0.0),
                bias=item.get("bias", "NEUTRAL"),
                bias_strength=item.get("bias_strength", 0.0),
                confidence=item.get("confidence", 0.0),
                setup_quality=item.get("setup_quality", ""),
                action=item.get("action", ""),
                stop_distance_multiplier=item.get("stop_distance_multiplier", 1.0),
                position_size_multiplier=item.get("position_size_multiplier", 1.0),
                entry_zone_low=item.get("entry_zone_low", 0.0),
                entry_zone_high=item.get("entry_zone_high", 0.0),
                stop_loss=item.get("stop_loss", 0.0),
                take_profit=item.get("take_profit", 0.0),
                atr=item.get("atr", 0.0),
            ))
        self.load_features(features)

    def initialize(self, candles: List[HistoricalCandle],
                   higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None):
        """No initialization needed — features are pre-loaded."""
        pass

    def evaluate(self, alignment: TimeframeAlignment, portfolio: Portfolio,
                 engine=None) -> SignalEvent:
        """Evaluate using precomputed features."""
        candle = alignment.primary_candle
        feat = self._features.get(candle.timestamp)

        if feat is None:
            return SignalEvent(
                timestamp=candle.timestamp, direction="NO_TRADE",
                strategy_id=self._strategy_id,
                reason_codes=["NO_PRECOMPUTED_FEATURES"],
            )

        # Map bias to direction
        if feat.bias in ("STRONG_LONG", "LONG") and feat.action == "ENTER":
            direction = "BUY"
        elif feat.bias in ("STRONG_SHORT", "SHORT") and feat.action == "ENTER":
            direction = "SELL"
        elif feat.action == "AVOID":
            direction = "NO_TRADE"
        elif feat.action in ("WAIT", "EXIT"):
            direction = "NO_TRADE"
        else:
            direction = "NO_TRADE"

        if direction == "NO_TRADE":
            return SignalEvent(
                timestamp=candle.timestamp, direction="NO_TRADE",
                strategy_id=self._strategy_id,
                raw_score=feat.confluence_score,
                confluence=feat.confluence_score,
                confidence=feat.confidence,
                setup_grade=feat.setup_quality,
                regime=feat.regime,
                reason_codes=["PTB_NO_ACTION"],
            )

        # Compute entry/SL/TP from precomputed data
        entry = candle.close
        atr = feat.atr if feat.atr > 0 else 1.0
        sl_mult = feat.stop_distance_multiplier

        if direction == "BUY":
            stop_loss = entry - atr * sl_mult
            tp1 = entry + atr * 1.0
        else:
            stop_loss = entry + atr * sl_mult
            tp1 = entry - atr * 1.0

        return SignalEvent(
            timestamp=candle.timestamp, direction=direction,
            strategy_id=self._strategy_id,
            raw_score=feat.confluence_score,
            confluence=feat.confluence_score,
            confidence=feat.confidence,
            setup_grade=feat.setup_quality,
            regime=feat.regime,
            entry_price=entry, stop_loss=stop_loss,
            tp1=tp1,
        )

    def reset(self):
        """Reset for fold isolation."""
        pass

    def validate_parity(self, live_signals: List[SignalEvent],
                         tolerance: float = 0.05) -> Dict:
        """Validate parity with live PTB run.

        Compares precomputed replay decisions with live-style decisions.
        Returns a parity report.
        """
        matches = 0
        mismatches = 0
        for live_sig in live_signals:
            feat = self._features.get(live_sig.timestamp)
            if feat is None:
                continue

            # Compare direction
            replay_dir = "NO_TRADE"
            if feat.bias in ("STRONG_LONG", "LONG") and feat.action == "ENTER":
                replay_dir = "BUY"
            elif feat.bias in ("STRONG_SHORT", "SHORT") and feat.action == "ENTER":
                replay_dir = "SELL"

            if live_sig.direction == replay_dir:
                matches += 1
            else:
                mismatches += 1

            # Compare confluence within tolerance
            confluence_diff = abs(live_sig.confluence - feat.confluence_score)
            if confluence_diff > tolerance * 100:
                mismatches += 1

        total = matches + mismatches
        return {
            "total_compared": total,
            "matches": matches,
            "mismatches": mismatches,
            "parity_rate": matches / total if total > 0 else 0.0,
            "tolerance": tolerance,
        }
