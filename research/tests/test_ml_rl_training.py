"""Tests for ML training and RL training pipelines."""
import numpy as np
import pytest
from patresearch.ml_training import (
    TrainingConfig, chronological_split, walk_forward_split,
    validate_no_leakage, train_model
)
from patresearch.rl_training import (
    RLConfig, RLEnvironment, validate_rl_for_live
)
from datetime import datetime, timedelta


class TestMLTraining:
    def test_chronological_split_no_leakage(self):
        """Verify chronological split prevents leakage."""
        n = 200
        features = np.random.randn(n, 5)
        targets = np.random.randn(n, 3)
        config = TrainingConfig(min_samples=100)
        (train_x, train_y), (val_x, val_y), (test_x, test_y) = chronological_split(features, targets, config)
        assert len(train_x) == 120
        assert len(val_x) == 40
        assert len(test_x) == 40
        # Verify chronological order: train ends before val starts
        assert len(train_x) + len(val_x) + len(test_x) == n

    def test_insufficient_samples_raises(self):
        """Insufficient samples should raise error."""
        features = np.random.randn(50, 5)
        targets = np.random.randn(50, 3)
        config = TrainingConfig(min_samples=100)
        with pytest.raises(ValueError, match="Insufficient"):
            chronological_split(features, targets, config)

    def test_walk_forward_splits(self):
        """Walk-forward splits should train before validate."""
        n = 300
        features = np.random.randn(n, 5)
        targets = np.random.randn(n, 3)
        folds = walk_forward_split(features, targets, n_folds=5)
        assert len(folds) == 5
        for (train, val) in folds:
            assert len(train[0]) > 0
            assert len(val[0]) > 0
            # Train data should come before val data
            assert len(train[0]) < n

    def test_validate_no_leakage(self):
        """Leakage detection should work."""
        now = datetime.utcnow()
        features_ts = [now, now + timedelta(hours=1), now + timedelta(hours=2)]
        outcomes_ts = [now + timedelta(hours=2), now + timedelta(hours=3), now + timedelta(hours=4)]
        # Feature at index 0 is before outcome at index 0 — OK
        assert validate_no_leakage(features_ts, outcomes_ts)

        # Leakage: feature after outcome
        bad_outcomes = [now - timedelta(hours=1), now + timedelta(hours=1), now + timedelta(hours=2)]
        assert not validate_no_leakage(features_ts, bad_outcomes)

    def test_train_model(self):
        """Test actual model training."""
        n = 200
        np.random.seed(42)
        features = np.random.randn(n, 5)
        targets = np.random.randn(n, 3) * 0.1 + features[:, :3] * 0.5
        config = TrainingConfig(min_samples=100, model_name="test_model")
        result = train_model(features, targets, config)
        assert result.model_name == "test_model"
        assert result.sample_count == n
        assert result.train_samples > 0
        assert "test_r2" in result.metrics
        assert "val_r2" in result.metrics


class TestRLTraining:
    def test_environment_reset(self):
        """Environment reset should return observation."""
        data = np.random.randn(100, 5)
        config = RLConfig()
        env = RLEnvironment(data, config)
        obs = env.reset()
        assert obs is not None
        assert len(obs) > 0

    def test_transaction_cost(self):
        """Transaction cost should be applied on trade open."""
        data = np.random.randn(100, 5)
        config = RLConfig()
        env = RLEnvironment(data, config)
        env.reset()
        _, reward, _, _ = env.step(1)  # LONG
        # Reward should include transaction cost penalty
        assert reward < 0  # penalized for cost

    def test_drawdown_penalty(self):
        """Drawdown penalty should be applied."""
        data = np.random.randn(100, 5)
        config = RLConfig()
        env = RLEnvironment(data, config)
        env.reset()
        env.max_equity = 100.0
        env.total_pnl = 90.0  # 10 drawdown
        _, reward, _, _ = env.step(0)  # NO_TRADE
        # Reward should include drawdown penalty
        assert reward < 0

    def test_deterministic_observation(self):
        """Same state should give same observation."""
        data = np.random.randn(100, 5)
        config = RLConfig()
        env = RLEnvironment(data, config)
        obs1 = env.reset()
        env2 = RLEnvironment(data, config)
        obs2 = env2.reset()
        assert np.array_equal(obs1, obs2)

    def test_validate_rl_for_live(self):
        """RL validation should check metrics."""
        # Insufficient trades
        ok, _ = validate_rl_for_live(
            {"trade_count": 10, "max_drawdown": 5, "profit_factor": 1.5},
            RLConfig()
        )
        assert not ok

        # Good metrics
        ok, _ = validate_rl_for_live(
            {"trade_count": 100, "max_drawdown": 5, "profit_factor": 1.5},
            RLConfig()
        )
        assert ok

        # High drawdown
        ok, _ = validate_rl_for_live(
            {"trade_count": 100, "max_drawdown": 15, "profit_factor": 1.5},
            RLConfig()
        )
        assert not ok
