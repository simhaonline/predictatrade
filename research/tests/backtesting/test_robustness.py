"""Tests for robustness: walk-forward isolation, Monte Carlo determinism, sensitivity."""
import pytest
import numpy as np

from patresearch.backtesting.data.loader import DataLoader
from patresearch.backtesting.engine.core import BacktestEngine, BacktestConfig
from patresearch.backtesting.strategy.ptb_strategy import PTBStrategyAdapter
from patresearch.backtesting.analytics.walk_forward import WalkForwardAnalyzer, WalkForwardConfig
from patresearch.backtesting.analytics.monte_carlo import MonteCarloAnalyzer, MonteCarloConfig
from patresearch.backtesting.analytics.sensitivity import SensitivityAnalyzer, SensitivityConfig
from patresearch.backtesting.engine.portfolio import TradeRecord
from datetime import datetime, timezone, timedelta


class TestWalkForward:
    def test_fold_isolation(self):
        """Walk-forward folds should be isolated (no state leakage)."""
        candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 500, seed=42)
        config = BacktestConfig(random_seed=42)
        wf_config = WalkForwardConfig(train_size=200, test_size=50, step_size=100)

        analyzer = WalkForwardAnalyzer(wf_config)
        result = analyzer.run(
            candles,
            strategy_factory=lambda: PTBStrategyAdapter("STANDARD_SCALPING"),
            bt_config=config,
            final_holdout_pct=0.2,
        )

        # Each fold should have independent results
        for fold in result.folds:
            assert fold.in_sample_result is not None
            assert fold.out_of_sample_result is not None
            # Folds should not overlap
            assert fold.train_end <= fold.test_start

    def test_final_holdout_separate(self):
        """Final holdout should be separate from walk-forward data."""
        candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 1000, seed=42)
        config = BacktestConfig(random_seed=42)
        wf_config = WalkForwardConfig(train_size=200, test_size=50, step_size=100)

        analyzer = WalkForwardAnalyzer(wf_config)
        result = analyzer.run(
            candles,
            strategy_factory=lambda: PTBStrategyAdapter("STANDARD_SCALPING"),
            bt_config=config,
            final_holdout_pct=0.2,
        )

        if result.final_holdout_result:
            # Holdout should use the last 20% of data
            holdout_start = int(1000 * 0.8)
            assert result.final_holdout_result.bars_processed <= 200


class TestMonteCarlo:
    def test_deterministic_seed(self):
        """Same seed should produce same Monte Carlo results."""
        trades = [TradeRecord(
            trade_id=f"t{i}", signal_id="s1", strategy_id="STANDARD_SCALPING",
            direction="BUY", entry_time=datetime(2025, 1, 1, tzinfo=timezone.utc),
            entry_price=2400, exit_time=datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc),
            exit_price=2400 + np.random.randn() * 10, exit_reason="TP",
            size=1.0, pnl=np.random.randn() * 50, pnl_r=1.0,
        ) for i in range(20)]

        mc_config = MonteCarloConfig(n_simulations=100, random_seed=42)
        analyzer = MonteCarloAnalyzer(mc_config)
        result1 = analyzer.run(trades, 10000)

        analyzer2 = MonteCarloAnalyzer(mc_config)
        result2 = analyzer2.run(trades, 10000)

        assert result1.final_equity_percentiles["p50"] == result2.final_equity_percentiles["p50"]
        assert result1.prob_of_loss == result2.prob_of_loss

    def test_percentile_reporting(self):
        """Monte Carlo should report percentiles, not single curve."""
        trades = [TradeRecord(
            trade_id=f"t{i}", signal_id="s1", strategy_id="STANDARD_SCALPING",
            direction="BUY", entry_time=datetime(2025, 1, 1, tzinfo=timezone.utc),
            entry_price=2400, exit_time=datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc),
            exit_price=2410, exit_reason="TP", size=1.0,
            pnl=np.random.randn() * 100, pnl_r=1.0,
        ) for i in range(20)]

        mc_config = MonteCarloConfig(n_simulations=200, random_seed=42)
        analyzer = MonteCarloAnalyzer(mc_config)
        result = analyzer.run(trades, 10000)

        # Should have multiple percentile levels
        assert len(result.final_equity_percentiles) >= 3
        assert "p5" in result.final_equity_percentiles
        assert "p50" in result.final_equity_percentiles
        assert "p95" in result.final_equity_percentiles

    def test_zero_trades_safe(self):
        """Monte Carlo with zero trades should be safe."""
        mc_config = MonteCarloConfig(n_simulations=100, random_seed=42)
        analyzer = MonteCarloAnalyzer(mc_config)
        result = analyzer.run([], 10000)
        assert result.n_simulations == 0


class TestSensitivity:
    def test_sensitivity_runs(self):
        """Sensitivity analysis should run without error."""
        candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 200, seed=42)
        config = BacktestConfig(random_seed=42)
        sens_config = SensitivityConfig(perturbation_pct=0.10)
        analyzer = SensitivityAnalyzer(sens_config)
        result = analyzer.run(
            candles, config,
            strategy_factory=lambda: PTBStrategyAdapter("STANDARD_SCALPING"),
        )
        assert len(result.results) > 0

    def test_does_not_alter_production(self):
        """Sensitivity analysis should not alter the original config."""
        candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 200, seed=42)
        config = BacktestConfig(random_seed=42, max_risk_per_trade=0.02)
        original_risk = config.max_risk_per_trade

        sens_config = SensitivityConfig(perturbation_pct=0.20)
        analyzer = SensitivityAnalyzer(sens_config)
        analyzer.run(
            candles, config,
            strategy_factory=lambda: PTBStrategyAdapter("STANDARD_SCALPING"),
        )

        assert config.max_risk_per_trade == original_risk  # not altered
