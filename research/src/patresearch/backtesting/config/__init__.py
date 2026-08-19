"""Backtesting configuration."""
from dataclasses import dataclass, field
from typing import List, Optional
from datetime import datetime

from ..engine.core import BacktestConfig
from ..engine.execution import ExecutionConfig


def config_from_env(env: dict) -> BacktestConfig:
    """Load backtest configuration from environment variables."""
    cfg = BacktestConfig()
    cfg.symbol = env.get("BACKTEST_SYMBOL", "XAUUSD")
    cfg.strategy_id = env.get("BACKTEST_STRATEGY", "STANDARD_SCALPING")
    cfg.primary_timeframe = env.get("BACKTEST_PRIMARY_TIMEFRAME", "M5")
    cfg.initial_balance = float(env.get("BACKTEST_INITIAL_BALANCE", "10000"))
    cfg.random_seed = int(env.get("BACKTEST_RANDOM_SEED", "42"))

    # Execution
    cfg.execution_config.spread_model = env.get("BACKTEST_SPREAD_MODEL", "fixed")
    cfg.execution_config.fixed_spread = float(env.get("BACKTEST_FIXED_SPREAD", "0.30"))
    cfg.execution_config.slippage_model = env.get("BACKTEST_SLIPPAGE_MODEL", "fixed")
    cfg.execution_config.fixed_slippage = float(env.get("BACKTEST_FIXED_SLIPPAGE", "0.05"))
    cfg.execution_config.commission_per_lot = float(env.get("BACKTEST_COMMISSION", "7.0"))
    cfg.execution_config.latency_ms = int(env.get("BACKTEST_LATENCY_MS", "0"))
    cfg.execution_config.partial_fill_probability = float(env.get("BACKTEST_PARTIAL_FILL_PROBABILITY", "0.0"))

    # Exit management
    cfg.trailing_stop_enabled = env.get("BACKTEST_TRAILING_STOP_ENABLED", "true").lower() == "true"
    cfg.trailing_atr_multiplier = float(env.get("BACKTEST_TRAILING_ATR_MULTIPLIER", "2.0"))
    cfg.break_even_enabled = env.get("BACKTEST_BREAK_EVEN_ENABLED", "true").lower() == "true"
    cfg.break_even_trigger_r = float(env.get("BACKTEST_BREAK_EVEN_TRIGGER", "1.0"))
    cfg.max_holding_bars = int(env.get("BACKTEST_MAX_HOLDING_TIME", "0"))

    # Walk-forward / Monte Carlo
    cfg.walk_forward_enabled = env.get("BACKTEST_WALK_FORWARD_ENABLED", "false").lower() == "true"
    cfg.monte_carlo_enabled = env.get("BACKTEST_MONTE_CARLO_ENABLED", "false").lower() == "true"
    cfg.monte_carlo_runs = int(env.get("BACKTEST_MONTE_CARLO_RUNS", "1000"))
    cfg.sensitivity_enabled = env.get("BACKTEST_SENSITIVITY_ENABLED", "false").lower() == "true"

    return cfg
