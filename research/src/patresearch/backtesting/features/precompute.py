"""PTB feature precomputation pipeline.

Iterates through historical data chronologically, builds valid historical
snapshots, invokes the actual PTB, and stores outputs for later replay.

Every feature dataset records:
- symbol, timeframe, start/end timestamp
- data version/hash, PTB/model version, configuration hash
- source commit SHA, creation timestamp, feature schema/version
"""
from __future__ import annotations

from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from typing import List, Optional, Dict
import hashlib
import json

from ..data.loader import HistoricalCandle
from ..data.alignment import MultiTimeframeAligner, TimeframeAlignment
from ..strategy.ptb_strategy import PTBStrategyAdapter
from ..strategy.precomputed_ptb_strategy import PrecomputedFeatures, FeatureDatasetMetadata


@dataclass
class PrecomputeConfig:
    """Configuration for feature precomputation."""
    symbol: str = "XAUUSD"
    timeframe: str = "M5"
    higher_timeframes: List[str] = field(default_factory=lambda: ["M15", "H1", "H4", "D1"])
    ptb_model_version: str = "1.0.0"
    config_hash: str = ""
    source_commit_sha: str = ""


class FeaturePrecomputer:
    """Precomputes PTB features for later replay."""

    def __init__(self, config: PrecomputeConfig):
        self.config = config

    def precompute(self, candles: List[HistoricalCandle],
                   higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None,
                   strategy: Optional[PTBStrategyAdapter] = None) -> tuple:
        """Precompute PTB features for all candles.

        Returns (features_list, metadata).
        """
        if strategy is None:
            strategy = PTBStrategyAdapter()

        # Initialize strategy with full data
        strategy.initialize(candles, higher_tf_data)

        # Align timeframes
        aligner = MultiTimeframeAligner(self.config.timeframe, self.config.higher_timeframes)
        if higher_tf_data:
            alignments = aligner.align(candles, higher_tf_data)
        else:
            alignments = [
                TimeframeAlignment(timestamp=c.timestamp, primary_candle=c,
                                   higher_tf_candles={}, primary_index=i)
                for i, c in enumerate(candles)
            ]

        features_list = []
        from ..engine.events import SignalEvent
        from ..engine.portfolio import Portfolio

        portfolio = Portfolio()

        for align in alignments:
            signal = strategy.evaluate(align, portfolio, None)

            feat = PrecomputedFeatures(
                timestamp=align.timestamp,
                regime=signal.regime,
                volatility_state="",
                manipulation_index=0.0,
                confluence_score=signal.confluence,
                bias="LONG" if signal.direction == "BUY" else ("SHORT" if signal.direction == "SELL" else "NEUTRAL"),
                bias_strength=signal.raw_score / 100.0 if signal.raw_score > 0 else 0.0,
                confidence=signal.confidence,
                setup_quality=signal.setup_grade,
                action="ENTER" if signal.direction in ("BUY", "SELL") else "WAIT",
                stop_distance_multiplier=1.0,
                position_size_multiplier=1.0,
                atr=strategy._get_current_atr() if hasattr(strategy, '_get_current_atr') else 0.0,
            )
            features_list.append(feat)

        # Compute data hash
        h = hashlib.sha256()
        for c in candles:
            h.update(f"{c.timestamp.isoformat()}{c.open}{c.high}{c.low}{c.close}".encode())
        data_hash = h.hexdigest()[:16]

        metadata = FeatureDatasetMetadata(
            symbol=self.config.symbol,
            timeframe=self.config.timeframe,
            start_timestamp=candles[0].timestamp.isoformat() if candles else "",
            end_timestamp=candles[-1].timestamp.isoformat() if candles else "",
            data_version_hash=data_hash,
            ptb_model_version=self.config.ptb_model_version,
            config_hash=self.config.config_hash,
            source_commit_sha=self.config.source_commit_sha,
            creation_timestamp=datetime.now(timezone.utc).isoformat(),
            feature_schema_version="1.0",
            feature_count=len(features_list),
        )

        return features_list, metadata

    def save_to_json(self, features: List[PrecomputedFeatures],
                     metadata: FeatureDatasetMetadata, filepath: str):
        """Save precomputed features to JSON file."""
        data = {
            "metadata": asdict(metadata),
            "features": [
                {
                    "timestamp": f.timestamp.isoformat(),
                    "regime": f.regime,
                    "volatility_state": f.volatility_state,
                    "manipulation_index": f.manipulation_index,
                    "confluence_score": f.confluence_score,
                    "bias": f.bias,
                    "bias_strength": f.bias_strength,
                    "confidence": f.confidence,
                    "setup_quality": f.setup_quality,
                    "action": f.action,
                    "stop_distance_multiplier": f.stop_distance_multiplier,
                    "position_size_multiplier": f.position_size_multiplier,
                    "atr": f.atr,
                }
                for f in features
            ],
        }
        with open(filepath, "w") as f:
            json.dump(data, f, indent=2)
