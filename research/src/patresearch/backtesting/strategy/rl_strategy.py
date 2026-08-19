"""RL strategy support — standalone and PTB confirmation filter modes.

STANDALONE: RL chooses actions (NO_TRADE, LONG, SHORT, CLOSE)
PTB_CONFIRMATION_FILTER: PTB candidate → risk gates → RL confirm/veto → final action

RL must NEVER bypass mandatory risk controls.
Feature schema safety: feature names, order, dtype must be verified.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import List, Optional, Dict, Callable, Any
import hashlib
import json
import numpy as np

from ..data.loader import HistoricalCandle
from ..data.alignment import TimeframeAlignment
from ..engine.events import SignalEvent
from ..engine.portfolio import Portfolio
from .base import BaseStrategy
from .ptb_strategy import PTBStrategyAdapter


@dataclass
class FeatureSchema:
    """RL feature schema for safety verification."""
    feature_names: List[str]
    feature_order: List[str]
    dtypes: List[str]  # float32, float64, etc.
    normalization: Dict[str, str]  # feature -> normalization type
    observation_dim: int
    model_version: str

    def fingerprint(self) -> str:
        """Compute a fingerprint of the schema."""
        data = json.dumps({
            "names": self.feature_names,
            "order": self.feature_order,
            "dtypes": self.dtypes,
            "dim": self.observation_dim,
            "version": self.model_version,
        }, sort_keys=True)
        return hashlib.sha256(data.encode()).hexdigest()[:16]

    def validate(self, features: Dict[str, float]) -> tuple:
        """Validate that features match the schema.

        Returns (valid, error_message).
        """
        if len(features) != self.observation_dim:
            return False, f"Feature count mismatch: {len(features)} != {self.observation_dim}"

        for name in self.feature_names:
            if name not in features:
                return False, f"Missing feature: {name}"

            val = features[name]
            if not isinstance(val, (int, float, np.floating)):
                return False, f"Wrong type for {name}: {type(val)}"

            if np.isnan(val) or np.isinf(val):
                return False, f"NaN/Inf for {name}: {val}"

        return True, ""


class RLStandaloneStrategy(BaseStrategy):
    """RL strategy in standalone mode.

    RL model chooses actions: NO_TRADE, LONG, SHORT, CLOSE.
    The model's decisions are mapped to existing production signal/order semantics.
    Risk gates still apply — RL cannot bypass them.
    """

    def __init__(self, strategy_id: str = "RL_STANDALONE",
                 model_fn: Optional[Callable] = None,
                 schema: Optional[FeatureSchema] = None,
                 ptb_adapter: Optional[PTBStrategyAdapter] = None):
        self._strategy_id = strategy_id
        self._model_fn = model_fn  # inference function: (observation) -> (action, confidence)
        self._schema = schema
        self._ptb_adapter = ptb_adapter  # for feature extraction

    @property
    def strategy_id(self) -> str:
        return self._strategy_id

    def initialize(self, candles: List[HistoricalCandle],
                   higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None):
        if self._ptb_adapter:
            self._ptb_adapter.initialize(candles, higher_tf_data)

    def evaluate(self, alignment: TimeframeAlignment, portfolio: Portfolio,
                 engine=None) -> SignalEvent:
        """Evaluate RL model and produce a signal."""
        candle = alignment.primary_candle

        if self._model_fn is None:
            return SignalEvent(
                timestamp=candle.timestamp, direction="NO_TRADE",
                strategy_id=self._strategy_id,
                reason_codes=["NO_RL_MODEL"],
            )

        # Extract features
        features = self._extract_features(alignment, portfolio)

        # Validate schema
        if self._schema:
            valid, err = self._schema.validate(features)
            if not valid:
                return SignalEvent(
                    timestamp=candle.timestamp, direction="NO_TRADE",
                    strategy_id=self._strategy_id,
                    reason_codes=["RL_FEATURE_SCHEMA_MISMATCH", err],
                )

        # Run inference
        try:
            action, confidence = self._model_fn(features)
        except Exception as e:
            return SignalEvent(
                timestamp=candle.timestamp, direction="NO_TRADE",
                strategy_id=self._strategy_id,
                reason_codes=["RL_INFERENCE_ERROR", str(e)[:100]],
            )

        # Map action to direction
        action_map = {
            "NO_TRADE": "NO_TRADE",
            "LONG": "BUY",
            "SHORT": "SELL",
            "CLOSE": "NO_TRADE",  # CLOSE would close existing positions
        }
        direction = action_map.get(action, "NO_TRADE")

        if direction == "NO_TRADE":
            return SignalEvent(
                timestamp=candle.timestamp, direction="NO_TRADE",
                strategy_id=self._strategy_id,
                confidence=confidence,
                reason_codes=[f"RL_ACTION_{action}"],
            )

        # Compute entry/SL/TP using simple ATR-based method
        atr = features.get("atr", 1.0)
        entry = candle.close
        if direction == "BUY":
            stop_loss = entry - atr * 1.5
            tp1 = entry + atr * 1.0
        else:
            stop_loss = entry + atr * 1.5
            tp1 = entry - atr * 1.0

        return SignalEvent(
            timestamp=candle.timestamp, direction=direction,
            strategy_id=self._strategy_id,
            confidence=confidence,
            entry_price=entry, stop_loss=stop_loss, tp1=tp1,
        )

    def _extract_features(self, alignment: TimeframeAlignment, portfolio: Portfolio) -> Dict:
        """Extract RL observation features."""
        candle = alignment.primary_candle
        return {
            "regime": 0.0,
            "confluence": 0.0,
            "confidence": 0.0,
            "manipulation_index": 0.0,
            "volatility": 0.0,
            "liquidity": 0.0,
            "sentiment": 0.0,
            "dxy": 0.0,
            "real_yields": 0.0,
            "session": 0.0,
            "spread": 0.3,
            "atr": max(1.0, candle.high - candle.low),
            "recent_returns": (candle.close - candle.open) / candle.open if candle.open > 0 else 0.0,
            "position_state": 1.0 if portfolio.open_position_count > 0 else 0.0,
        }

    def reset(self):
        if self._ptb_adapter:
            self._ptb_adapter.reset()


class RLConfirmationFilter(BaseStrategy):
    """RL as a confirmation filter on PTB candidates.

    Flow: PTB candidate → PTB/risk gates → RL observation → RL confirm/veto → final action

    RL can only VETO a PTB signal — it cannot create new signals.
    RL must NEVER bypass mandatory risk controls.
    """

    def __init__(self, ptb_adapter: PTBStrategyAdapter,
                 model_fn: Optional[Callable] = None,
                 schema: Optional[FeatureSchema] = None,
                 min_confidence: float = 0.5):
        self._ptb = ptb_adapter
        self._model_fn = model_fn
        self._schema = schema
        self._min_confidence = min_confidence
        self._strategy_id = f"{ptb_adapter.strategy_id}_RL_CONFIRM"

    @property
    def strategy_id(self) -> str:
        return self._strategy_id

    def initialize(self, candles: List[HistoricalCandle],
                   higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None):
        self._ptb.initialize(candles, higher_tf_data)

    def evaluate(self, alignment: TimeframeAlignment, portfolio: Portfolio,
                 engine=None) -> SignalEvent:
        """Evaluate PTB first, then apply RL confirmation filter."""
        # Step 1: Get PTB signal
        ptb_signal = self._ptb.evaluate(alignment, portfolio, engine)

        # If PTB says NO_TRADE, no need for RL confirmation
        if ptb_signal.direction not in ("BUY", "SELL"):
            return ptb_signal

        # Step 2: Extract RL features
        features = self._extract_features(alignment, portfolio, ptb_signal)

        # Step 3: Validate schema
        if self._schema:
            valid, err = self._schema.validate(features)
            if not valid:
                # Schema mismatch → fail safe (veto)
                return SignalEvent(
                    timestamp=alignment.primary_candle.timestamp,
                    direction="BLOCKED",
                    strategy_id=self._strategy_id,
                    reason_codes=["RL_SCHEMA_MISMATCH", err],
                )

        # Step 4: Run RL inference
        if self._model_fn is None:
            # No model → pass through PTB signal
            ptb_signal.strategy_id = self._strategy_id
            return ptb_signal

        try:
            action, confidence = self._model_fn(features)
        except Exception:
            # Inference error → fail safe (veto)
            return SignalEvent(
                timestamp=alignment.primary_candle.timestamp,
                direction="BLOCKED",
                strategy_id=self._strategy_id,
                reason_codes=["RL_INFERENCE_ERROR"],
            )

        # Step 5: Apply RL confirmation
        # RL can only veto (NO_TRADE/CLOSE), not change direction
        if action in ("NO_TRADE", "CLOSE"):
            return SignalEvent(
                timestamp=alignment.primary_candle.timestamp,
                direction="BLOCKED",
                strategy_id=self._strategy_id,
                raw_score=ptb_signal.raw_score,
                confluence=ptb_signal.confluence,
                confidence=confidence,
                reason_codes=["RL_VETO"],
            )

        # RL confirms → pass through PTB signal with RL confidence
        if confidence >= self._min_confidence:
            ptb_signal.strategy_id = self._strategy_id
            ptb_signal.confidence = confidence
            return ptb_signal
        else:
            # Low confidence → veto
            return SignalEvent(
                timestamp=alignment.primary_candle.timestamp,
                direction="BLOCKED",
                strategy_id=self._strategy_id,
                reason_codes=["RL_LOW_CONFIDENCE"],
            )

    def _extract_features(self, alignment: TimeframeAlignment,
                          portfolio: Portfolio, ptb_signal: SignalEvent) -> Dict:
        """Extract features for RL confirmation."""
        candle = alignment.primary_candle
        return {
            "regime": 0.0,
            "confluence": ptb_signal.confluence,
            "confidence": ptb_signal.confidence,
            "manipulation_index": 0.0,
            "volatility": 0.0,
            "liquidity": 0.0,
            "sentiment": 0.0,
            "dxy": 0.0,
            "real_yields": 0.0,
            "session": 0.0,
            "spread": 0.3,
            "atr": max(1.0, candle.high - candle.low),
            "recent_returns": (candle.close - candle.open) / candle.open if candle.open > 0 else 0.0,
            "position_state": 1.0 if portfolio.open_position_count > 0 else 0.0,
        }

    def reset(self):
        self._ptb.reset()
