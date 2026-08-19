"""Predict-A-Trade Backtesting Framework.

A production-grade event-driven backtesting framework that reuses the
production strategy/PTB/risk gate logic through faithful Python adapters.

Key principles:
- Live/backtest parity: same decision logic, different execution layer
- No look-ahead bias: multi-timeframe alignment enforces information causality
- Realistic execution: spread, slippage, commission, latency, partial fills
- Conservative same-bar SL/TP: assumes worst case when order ambiguous
- Deterministic: same data + config + seed = same results
- Reproducible: run manifests with full provenance
"""
from .data.loader import DataLoader, HistoricalCandle
from .data.quality import DataQualityValidator, DataQualityReport
from .data.alignment import MultiTimeframeAligner, TimeframeAlignment
from .data.session_calendar import SessionCalendar
from .engine.core import BacktestEngine, BacktestConfig, BacktestRunResult
from .engine.events import MarketEvent, SignalEvent, OrderEvent, FillEvent
from .engine.execution import ExecutionSimulator, ExecutionConfig
from .engine.portfolio import Portfolio, Position, TradeRecord
from .strategy.base import BaseStrategy
from .strategy.ptb_strategy import PTBStrategyAdapter
from .strategy.precomputed_ptb_strategy import PrecomputedPTBStrategy
from .strategy.rl_strategy import RLStandaloneStrategy, RLConfirmationFilter
from .analytics.metrics import PerformanceMetrics, calculate_metrics
from .analytics.walk_forward import WalkForwardAnalyzer
from .analytics.monte_carlo import MonteCarloAnalyzer
from .analytics.sensitivity import SensitivityAnalyzer
from .reporting.report import ReportGenerator, RunManifest
