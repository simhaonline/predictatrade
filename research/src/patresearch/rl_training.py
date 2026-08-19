"""RL training environment for strategy optimization.

CRITICAL PRODUCTION RULE:
An unvalidated RL model must NOT directly control live MT4/MT5 execution.

Default production rollout:
  OFF → SHADOW → FILTER → APPROVED LIVE

Training runs OFFLINE in the research plane.
"""

from dataclasses import dataclass, field
from typing import List, Dict, Tuple, Optional
import numpy as np


@dataclass
class RLConfig:
    """RL training configuration."""
    model_name: str = "rl_strategy_v1"
    mode: str = "disabled"  # disabled, shadow, filter_only, live_approved
    max_episodes: int = 1000
    max_steps_per_episode: int = 500
    learning_rate: float = 0.001
    gamma: float = 0.99  # discount factor
    epsilon_start: float = 1.0
    epsilon_end: float = 0.01
    epsilon_decay: float = 0.995
    reward_config: Dict = field(default_factory=lambda: {
        "pnl_weight": 1.0,
        "drawdown_penalty": 0.3,
        "transaction_cost": 0.1,
        "spread_cost": 0.05,
        "slippage_cost": 0.05,
        "overtrading_penalty": 0.2,
        "risk_exposure_penalty": 0.1,
        "holding_cost": 0.02,
    })
    feature_columns: List[str] = field(default_factory=lambda: [
        "regime", "confluence", "confidence", "manipulation_index",
        "volatility", "liquidity", "sentiment", "dxy",
        "real_yields", "session", "spread", "atr",
        "recent_returns", "position_state"
    ])
    actions: List[str] = field(default_factory=lambda: [
        "NO_TRADE", "LONG", "SHORT", "CLOSE"
    ])


@dataclass
class RLTrainingResult:
    """Result of an RL training run."""
    model_name: str
    model_version: str
    mode: str
    episodes: int
    total_reward: float
    avg_reward: float
    max_drawdown: float
    sharpe_ratio: float
    sortino_ratio: float
    profit_factor: float
    win_rate: float
    expectancy: float
    trade_count: int
    oos_start: str = ""
    oos_end: str = ""
    walk_forward_folds: int = 0
    status: str = "PENDING"
    artifact_path: str = ""
    checksum: str = ""


class RLEnvironment:
    """RL training environment for XAUUSD strategy optimization.

    Uses PTB features as observation.
    Reward accounts for more than raw PnL.
    """

    def __init__(self, data: np.ndarray, config: RLConfig):
        """Initialize the RL environment.

        Args:
            data: Historical feature data (shape: [n_samples, n_features])
            config: RL training configuration
        """
        self.data = data
        self.config = config
        self.current_step = 0
        self.max_steps = min(len(data), config.max_steps_per_episode)
        self.position = 0  # 0=flat, 1=long, -1=short
        self.entry_price = 0.0
        self.total_pnl = 0.0
        self.trade_count = 0
        self.max_equity = 0.0
        self.max_drawdown = 0.0
        self.episode_rewards = []

    def reset(self) -> np.ndarray:
        """Reset the environment to start a new episode."""
        self.current_step = 0
        self.position = 0
        self.entry_price = 0.0
        self.total_pnl = 0.0
        self.trade_count = 0
        self.max_equity = 0.0
        self.max_drawdown = 0.0
        return self._get_observation()

    def step(self, action: int) -> Tuple[np.ndarray, float, bool, Dict]:
        """Take one step in the environment.

        Args:
            action: 0=NO_TRADE, 1=LONG, 2=SHORT, 3=CLOSE

        Returns:
            observation, reward, done, info
        """
        reward = 0.0
        info = {}

        if self.current_step >= self.max_steps:
            return self._get_observation(), 0.0, True, info

        current_price = self.data[self.current_step, 0] if self.data.shape[1] > 0 else 100.0

        # Execute action
        if action == 1:  # LONG
            if self.position == 0:
                self.position = 1
                self.entry_price = current_price
                self.trade_count += 1
                reward -= self.config.reward_config["transaction_cost"]
                reward -= self.config.reward_config["spread_cost"]
            else:
                reward -= self.config.reward_config["overtrading_penalty"]

        elif action == 2:  # SHORT
            if self.position == 0:
                self.position = -1
                self.entry_price = current_price
                self.trade_count += 1
                reward -= self.config.reward_config["transaction_cost"]
                reward -= self.config.reward_config["spread_cost"]
            else:
                reward -= self.config.reward_config["overtrading_penalty"]

        elif action == 3:  # CLOSE
            if self.position != 0:
                pnl = (current_price - self.entry_price) * self.position
                reward += pnl * self.config.reward_config["pnl_weight"]
                reward -= self.config.reward_config["slippage_cost"]
                self.total_pnl += pnl
                self.position = 0
                self.entry_price = 0.0

        elif action == 0:  # NO_TRADE
            if self.position == 0:
                reward -= self.config.reward_config["overtrading_penalty"] * 0.1

        # Holding cost
        if self.position != 0:
            reward -= self.config.reward_config["holding_cost"]

        # Risk exposure penalty
        if self.position != 0:
            reward -= self.config.reward_config["risk_exposure_penalty"]

        # Drawdown penalty
        equity = self.total_pnl
        if equity > self.max_equity:
            self.max_equity = equity
        drawdown = self.max_equity - equity
        if drawdown > self.max_drawdown:
            self.max_drawdown = drawdown
        reward -= drawdown * self.config.reward_config["drawdown_penalty"]

        self.episode_rewards.append(reward)
        self.current_step += 1
        done = self.current_step >= self.max_steps

        info["pnl"] = self.total_pnl
        info["trade_count"] = self.trade_count
        info["max_drawdown"] = self.max_drawdown

        return self._get_observation(), reward, done, info

    def _get_observation(self) -> np.ndarray:
        """Get current observation from data."""
        if self.current_step < len(self.data):
            obs = self.data[self.current_step].copy()
            # Append position state
            obs_with_pos = np.append(obs, float(self.position))
            return obs_with_pos
        return np.zeros(len(self.config.feature_columns) + 1)

    def get_metrics(self) -> Dict:
        """Get episode metrics for validation."""
        rewards = np.array(self.episode_rewards)
        if len(rewards) == 0:
            return {}
        wins = rewards[rewards > 0]
        losses = rewards[rewards < 0]
        total_win = wins.sum() if len(wins) > 0 else 0
        total_loss = abs(losses.sum()) if len(losses) > 0 else 0
        profit_factor = total_win / total_loss if total_loss > 0 else float('inf')
        win_rate = len(wins) / len(rewards) if len(rewards) > 0 else 0
        expectancy = rewards.mean() if len(rewards) > 0 else 0

        # Sharpe ratio (simplified)
        std = rewards.std()
        sharpe = rewards.mean() / std * np.sqrt(252) if std > 0 else 0

        # Sortino ratio (simplified)
        downside_std = np.sqrt(np.mean(losses ** 2)) if len(losses) > 0 else 0
        sortino = rewards.mean() / downside_std * np.sqrt(252) if downside_std > 0 else 0

        return {
            "total_reward": float(rewards.sum()),
            "avg_reward": float(rewards.mean()),
            "max_drawdown": float(self.max_drawdown),
            "sharpe_ratio": float(sharpe),
            "sortino_ratio": float(sortino),
            "profit_factor": float(profit_factor),
            "win_rate": float(win_rate),
            "expectancy": float(expectancy),
            "trade_count": int(self.trade_count),
        }


def validate_rl_for_live(metrics: Dict, config: RLConfig) -> Tuple[bool, str]:
    """Validate whether RL metrics are sufficient for live approval.

    Returns (can_approve, reason).
    """
    min_trades = 50
    max_drawdown = 10.0
    min_profit_factor = 1.3

    if metrics.get("trade_count", 0) < min_trades:
        return False, f"Insufficient OOS trades: {metrics.get('trade_count', 0)} < {min_trades}"

    if metrics.get("max_drawdown", 999) > max_drawdown:
        return False, f"Max drawdown exceeds limit: {metrics['max_drawdown']} > {max_drawdown}"

    if metrics.get("profit_factor", 0) < min_profit_factor:
        return False, f"Profit factor below minimum: {metrics['profit_factor']} < {min_profit_factor}"

    return True, ""
