"""Parameter sensitivity analysis.

Evaluates important parameters around their selected value (± configured percentages).
Measures impact on return, Sharpe, Sortino, max drawdown, profit factor, expectancy, trade count.

Highlights fragile configurations where small parameter changes cause large deterioration.

This analysis must NOT automatically alter production thresholds.
It is research evidence only unless an explicit promotion process exists.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Optional, Dict, Callable, Any
import numpy as np
import copy

from ..data.loader import HistoricalCandle
from ..engine.core import BacktestEngine, BacktestConfig, BacktestRunResult


@dataclass
class SensitivityConfig:
    """Sensitivity analysis configuration."""
    perturbation_pct: float = 0.10  # ±10%
    parameters: List[str] = field(default_factory=lambda: [
        "max_risk_per_trade", "min_rr", "max_spread_to_atr",
        "trailing_atr_multiplier", "break_even_trigger_r",
    ])


@dataclass
class SensitivityResult:
    """Sensitivity analysis result for a single parameter."""
    parameter: str
    base_value: float
    low_value: float
    high_value: float
    base_metrics: Dict = field(default_factory=dict)
    low_metrics: Dict = field(default_factory=dict)
    high_metrics: Dict = field(default_factory=dict)
    sensitivity_score: float = 0.0  # 0 = stable, 1 = very fragile


@dataclass
class SensitivityAnalysisResult:
    """Complete sensitivity analysis."""
    results: List[SensitivityResult] = field(default_factory=list)
    fragile_parameters: List[str] = field(default_factory=list)


class SensitivityAnalyzer:
    """Parameter sensitivity analyzer."""

    def __init__(self, config: SensitivityConfig):
        self.config = config

    def run(self, candles: List[HistoricalCandle],
            bt_config: BacktestConfig,
            strategy_factory: Callable[[], Any],
            higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None) -> SensitivityAnalysisResult:
        """Run sensitivity analysis."""
        result = SensitivityAnalysisResult()

        # Run base case
        base_engine = BacktestEngine(bt_config)
        base_engine.set_strategy(strategy_factory())
        base_result = base_engine.run(candles, higher_tf_data)
        base_metrics = self._extract_metrics(base_result)

        for param_name in self.config.parameters:
            base_val = getattr(bt_config, param_name, None)
            if base_val is None or not isinstance(base_val, (int, float)):
                continue

            low_val = base_val * (1 - self.config.perturbation_pct)
            high_val = base_val * (1 + self.config.perturbation_pct)

            # Run low case
            low_config = copy.deepcopy(bt_config)
            setattr(low_config, param_name, low_val)
            low_engine = BacktestEngine(low_config)
            low_engine.set_strategy(strategy_factory())
            low_result = low_engine.run(candles, higher_tf_data)
            low_metrics = self._extract_metrics(low_result)

            # Run high case
            high_config = copy.deepcopy(bt_config)
            setattr(high_config, param_name, high_val)
            high_engine = BacktestEngine(high_config)
            high_engine.set_strategy(strategy_factory())
            high_result = high_engine.run(candles, higher_tf_data)
            high_metrics = self._extract_metrics(high_result)

            # Compute sensitivity score
            base_sharpe = base_metrics.get("sharpe", 0)
            low_sharpe = low_metrics.get("sharpe", 0)
            high_sharpe = high_metrics.get("sharpe", 0)

            if base_sharpe != 0:
                sensitivity = max(
                    abs(base_sharpe - low_sharpe) / abs(base_sharpe),
                    abs(base_sharpe - high_sharpe) / abs(base_sharpe),
                )
            else:
                sensitivity = 0.0

            sr = SensitivityResult(
                parameter=param_name, base_value=base_val,
                low_value=low_val, high_value=high_val,
                base_metrics=base_metrics, low_metrics=low_metrics,
                high_metrics=high_metrics, sensitivity_score=sensitivity,
            )
            result.results.append(sr)

            if sensitivity > 0.5:
                result.fragile_parameters.append(param_name)

        return result

    def _extract_metrics(self, result: BacktestRunResult) -> Dict:
        """Extract key metrics from a backtest result."""
        if result.metrics is None:
            return {}
        return {
            "total_return_pct": result.metrics.total_return_pct,
            "sharpe": result.metrics.sharpe_ratio,
            "sortino": result.metrics.sortino_ratio,
            "max_drawdown_pct": result.metrics.max_drawdown_pct,
            "profit_factor": result.metrics.profit_factor,
            "expectancy": result.metrics.expectancy,
            "trade_count": result.metrics.total_trades,
        }
