"""Event-driven backtest engine core.

Reproduces the production signal pipeline:
  historical market update → strategy/PTB evaluation → signal
  → risk decision → order → simulated execution → fill
  → position/equity update → stop/target/trade-management → analytics

The engine is deterministic: same data + config + seed = same results.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import List, Optional, Dict, Callable
import uuid
import hashlib
import json
import time as time_module

from ..data.loader import HistoricalCandle, DatasetMetadata
from ..data.quality import DataQualityValidator, DataQualityReport
from ..data.alignment import MultiTimeframeAligner, TimeframeAlignment
from ..data.session_calendar import SessionCalendar, SessionInfo
from .events import MarketEvent, SignalEvent, OrderEvent, FillEvent, ExitEvent
from .execution import ExecutionSimulator, ExecutionConfig
from .portfolio import Portfolio, TradeRecord, EquityPoint
from ..strategy.base import BaseStrategy
from ..analytics.metrics import calculate_metrics, PerformanceMetrics


@dataclass
class BacktestConfig:
    """Configuration for a backtest run."""
    symbol: str = "XAUUSD"
    strategy_id: str = "STANDARD_SCALPING"
    strategy_mode: str = "ptb"  # ptb, precomputed_ptb, rl_standalone, rl_confirmation
    primary_timeframe: str = "M5"
    higher_timeframes: List[str] = field(default_factory=lambda: ["M15", "H1", "H4", "D1"])
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    initial_balance: float = 10000.0
    random_seed: int = 42

    # Execution
    execution_config: ExecutionConfig = field(default_factory=ExecutionConfig)

    # Risk gates (reuse production thresholds)
    max_risk_per_trade: float = 0.02  # 2% of equity
    max_daily_loss_percent: float = 2.0
    max_consecutive_losses: int = 2
    max_exposure: float = 5.0
    max_positions: int = 3
    min_rr: float = 1.0
    max_spread_to_atr: float = 0.5

    # Exit management
    trailing_stop_enabled: bool = True
    trailing_atr_multiplier: float = 2.0
    break_even_enabled: bool = True
    break_even_trigger_r: float = 1.0
    max_holding_bars: int = 0  # 0 = no time limit

    # Same-bar SL/TP policy
    conservative_sl_tp: bool = True  # assume SL hit first when ambiguous

    # Data quality
    min_quality_score: float = 0.8

    # Walk-forward / Monte Carlo
    walk_forward_enabled: bool = False
    monte_carlo_enabled: bool = False
    monte_carlo_runs: int = 1000
    sensitivity_enabled: bool = False


@dataclass
class BacktestRunResult:
    """Complete result of a single backtest run."""
    run_id: str
    config: BacktestConfig
    data_quality: DataQualityReport
    trades: List[TradeRecord] = field(default_factory=list)
    equity_curve: List[Dict] = field(default_factory=list)
    metrics: Optional[PerformanceMetrics] = None
    manifest: Optional[Dict] = None
    no_trade_count: int = 0
    blocked_count: int = 0
    error_count: int = 0
    bars_processed: int = 0
    duration_seconds: float = 0.0
    status: str = "COMPLETED"  # COMPLETED, FAILED, DATA_QUALITY_FAILED


class BacktestEngine:
    """Event-driven backtest engine.

    Processes historical data bar by bar, invoking the strategy at each bar,
    checking risk gates, simulating execution, and tracking the portfolio.
    """

    def __init__(self, config: BacktestConfig):
        self.config = config
        self.run_id = str(uuid.uuid4())
        self.portfolio = Portfolio(
            initial_balance=config.initial_balance,
            trailing_stop_enabled=config.trailing_stop_enabled,
            trailing_atr_multiplier=config.trailing_atr_multiplier,
            break_even_enabled=config.break_even_enabled,
            break_even_trigger_r=config.break_even_trigger_r,
            max_holding_bars=config.max_holding_bars,
            conservative_sl_tp=config.conservative_sl_tp,
        )
        self.execution_sim = ExecutionSimulator(config.execution_config)
        self.quality_validator = DataQualityValidator(min_quality_score=config.min_quality_score)
        self.aligner = MultiTimeframeAligner(config.primary_timeframe, config.higher_timeframes)
        self.session_cal = SessionCalendar()
        self.strategy: Optional[BaseStrategy] = None

        # Risk state (reuse production recovery logic concept)
        self.consecutive_losses = 0
        self.daily_loss_count = 0
        self.daily_pnl = 0.0
        self.daily_pnl_percent = 0.0
        self.current_trading_day: Optional[datetime] = None
        self.recovery_mode = False
        self.recovery_size_multiplier = 1.0

        # ATR tracking for trailing stops and slippage
        self.atr_history: List[float] = []
        self.atr_period = 14

        # Counters
        self.no_trade_count = 0
        self.blocked_count = 0
        self.error_count = 0
        self.blocked_signals: List[Dict] = []
        self.no_trade_signals: List[Dict] = []

        # Blocked signals audit
        self.blocked_signals: List[Dict] = []
        self.no_trade_signals: List[Dict] = []

    def set_strategy(self, strategy: BaseStrategy):
        """Set the strategy to use for this backtest."""
        self.strategy = strategy

    def run(self, candles: List[HistoricalCandle],
            higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None,
            data_meta: Optional[DatasetMetadata] = None) -> BacktestRunResult:
        """Run the backtest.

        Args:
            candles: Primary timeframe candles
            higher_tf_data: Higher timeframe data dict
            data_meta: Dataset metadata

        Returns:
            BacktestRunResult with all trades, metrics, and manifest
        """
        start_time = time_module.time()

        # 1. Data quality validation
        dq_report = self.quality_validator.validate(candles, self.config.symbol,
                                                     self.config.primary_timeframe)
        if not dq_report.passed:
            return BacktestRunResult(
                run_id=self.run_id, config=self.config,
                data_quality=dq_report, status="DATA_QUALITY_FAILED",
                duration_seconds=time_module.time() - start_time,
            )

        # 2. Multi-timeframe alignment (no look-ahead)
        if higher_tf_data:
            alignments = self.aligner.align(candles, higher_tf_data)
            # Verify no look-ahead
            if not self.aligner.verify_no_lookahead(alignments):
                return BacktestRunResult(
                    run_id=self.run_id, config=self.config,
                    data_quality=dq_report, status="FAILED",
                    duration_seconds=time_module.time() - start_time,
                    manifest={"error": "Look-ahead bias detected in timeframe alignment"},
                )
        else:
            alignments = [TimeframeAlignment(
                timestamp=c.timestamp, primary_candle=c,
                higher_tf_candles={}, primary_index=i,
            ) for i, c in enumerate(candles)]

        # 3. Initialize ATR history
        self._init_atr(candles)

        # 4. Strategy initialization
        if self.strategy:
            self.strategy.initialize(candles, higher_tf_data)

        # 5. Main event loop
        for i, align in enumerate(alignments):
            candle = align.primary_candle

            # Check for new trading day
            self._check_trading_day(candle.timestamp)

            # Update existing positions (SL/TP, trailing, time exits)
            atr = self._get_current_atr()
            exits = self.portfolio.update_positions(candle, atr)
            for exit_event in exits:
                self._process_exit(exit_event)

            # Strategy evaluation
            if self.strategy:
                signal = self.strategy.evaluate(align, self.portfolio, self)

                if signal.direction in ("BUY", "SELL"):
                    # Risk gate check
                    allowed, reason = self._check_risk_gates(signal, candle)

                    if allowed:
                        # Create order
                        order = self._create_order(signal, candle, atr)

                        # Execute
                        fill = self.execution_sim.execute(order, candle, atr)

                        if fill.fill_status != "REJECTED":
                            # Open position
                            pos_id = self.portfolio.open_position(
                                fill=fill, signal_id=signal.strategy_id,
                                strategy_id=signal.strategy_id,
                                stop_loss=signal.stop_loss, take_profit=signal.tp1,
                                tp1=signal.tp1, tp2=signal.tp2, tp3=signal.tp3,
                                regime=signal.regime, session=signal.session,
                                confluence=signal.confluence, confidence=signal.confidence,
                                setup_grade=signal.setup_grade,
                            )
                    else:
                        self.blocked_count += 1
                        self.blocked_signals.append({
                            "timestamp": candle.timestamp.isoformat(),
                            "reason": reason,
                            "direction": signal.direction,
                            "strategy_id": signal.strategy_id,
                        })

                elif signal.direction == "NO_TRADE":
                    self.no_trade_count += 1
                    self.no_trade_signals.append({
                        "timestamp": candle.timestamp.isoformat(),
                        "reason_codes": signal.reason_codes,
                    })
                elif signal.direction == "BLOCKED":
                    self.blocked_count += 1

            self._update_atr(candle)

        # 6. Close all positions at end of backtest
        if self.portfolio.positions:
            self.portfolio.close_all_positions(candles[-1])

        # 7. Calculate metrics
        metrics = calculate_metrics(
            self.portfolio.closed_trades,
            self.portfolio.equity_curve,
            self.config.initial_balance,
        )

        # 8. Build equity curve dict
        equity_curve = [
            {"timestamp": ep.timestamp.isoformat(), "equity": ep.equity,
             "balance": ep.balance, "drawdown": ep.drawdown}
            for ep in self.portfolio.equity_curve
        ]

        # 9. Build manifest
        manifest = self._build_manifest(candles, data_meta, dq_report)

        duration = time_module.time() - start_time

        return BacktestRunResult(
            run_id=self.run_id, config=self.config,
            data_quality=dq_report,
            trades=self.portfolio.closed_trades,
            equity_curve=equity_curve,
            metrics=metrics, manifest=manifest,
            no_trade_count=self.no_trade_count,
            blocked_count=self.blocked_count,
            error_count=self.error_count,
            bars_processed=len(candles),
            duration_seconds=duration,
        )

    def _check_risk_gates(self, signal: SignalEvent, candle: HistoricalCandle) -> tuple:
        """Check risk gates (reuse production logic concept).

        Returns (allowed, reason).
        """
        # Recovery mode check
        if self.recovery_mode:
            if signal.confluence < 80:
                return False, "RECOVERY_LOW_CONFLUENCE"
            if signal.setup_grade not in ("A", "A+"):
                return False, "RECOVERY_LOW_QUALITY"
            if signal.confidence < 75:
                return False, "RECOVERY_LOW_CONFIDENCE"

        # Daily loss limit
        if self.daily_pnl_percent <= -self.config.max_daily_loss_percent:
            return False, "DAILY_LOSS_LIMIT"

        # Max consecutive losses → recovery mode
        if self.consecutive_losses >= self.config.max_consecutive_losses and not self.recovery_mode:
            self.recovery_mode = True
            self.recovery_size_multiplier = 0.5

        # Max positions
        if self.portfolio.open_position_count >= self.config.max_positions:
            return False, "MAX_POSITIONS"

        # Max exposure
        if self.portfolio.total_exposure >= self.config.max_exposure:
            return False, "MAX_EXPOSURE"

        # Min R:R
        if signal.stop_loss and signal.tp1:
            risk = abs(signal.entry_price - signal.stop_loss)
            reward = abs(signal.tp1 - signal.entry_price)
            if risk > 0 and reward / risk < self.config.min_rr:
                return False, "POOR_RR"

        # Spread to ATR check
        atr = self._get_current_atr()
        if atr > 0:
            spread = self.config.execution_config.fixed_spread
            if spread / atr > self.config.max_spread_to_atr:
                return False, "HIGH_SPREAD_TO_ATR"

        # Session check
        session_info = self.session_cal.get_session(candle.timestamp)
        if not session_info.session_allowed:
            return False, "SESSION_NOT_ALLOWED"

        return True, ""

    def _create_order(self, signal: SignalEvent, candle: HistoricalCandle,
                      atr: float) -> OrderEvent:
        """Create an order from a signal with position sizing."""
        # Calculate position size from risk
        risk_amount = self.portfolio.equity * self.config.max_risk_per_trade
        risk_per_unit = abs(signal.entry_price - signal.stop_loss) * self.portfolio.contract_size

        if risk_per_unit > 0:
            size = risk_amount / risk_per_unit
        else:
            size = 0.01  # minimum lot

        # Apply recovery size multiplier
        size *= self.recovery_size_multiplier

        # Clamp to reasonable bounds
        size = max(0.01, min(size, 10.0))  # 0.01 to 10 lots

        return OrderEvent(
            timestamp=candle.timestamp,
            direction=signal.direction,
            size=size,
            stop_loss=signal.stop_loss,
            take_profit=signal.tp1,
            signal_id=signal.strategy_id,
        )

    def _process_exit(self, exit_event: ExitEvent):
        """Process a position exit for risk state tracking."""
        pnl = exit_event.realized_pnl
        self.daily_pnl += pnl

        # Update daily PnL percent
        if self.portfolio.initial_balance > 0:
            self.daily_pnl_percent = (self.daily_pnl / self.portfolio.initial_balance) * 100

        # Track consecutive losses
        if pnl < 0:
            self.consecutive_losses += 1
            self.daily_loss_count += 1
        elif pnl > 0:
            self.consecutive_losses = 0
            if self.recovery_mode:
                # Recovery exit after wins
                pass  # simplified — production recovery manager handles this

    def _check_trading_day(self, ts: datetime):
        """Check for new trading day reset."""
        trading_day = ts.date()
        if self.current_trading_day is None:
            self.current_trading_day = trading_day
        elif trading_day != self.current_trading_day:
            # New day — reset daily counters
            self.current_trading_day = trading_day
            self.daily_loss_count = 0
            self.daily_pnl = 0.0
            self.daily_pnl_percent = 0.0

    def _init_atr(self, candles: List[HistoricalCandle]):
        """Initialize ATR history from candle data."""
        if len(candles) < 2:
            return
        atrs = []
        for i in range(1, min(len(candles), self.atr_period + 1)):
            high = candles[i].high
            low = candles[i].low
            prev_close = candles[i-1].close
            tr = max(high - low, abs(high - prev_close), abs(low - prev_close))
            atrs.append(tr)
        if atrs:
            self.atr_history = atrs

    def _update_atr(self, candle: HistoricalCandle):
        """Update ATR with new candle.

        Uses the PREVIOUS candle's close to compute true range. Using the
        current close (as the prior implementation did) collapses TR to
        high-low and ignores overnight gaps, biasing ATR low.
        """
        prev_close = getattr(self, '_prev_close', None)
        if prev_close is None or not self.atr_history:
            tr = candle.high - candle.low
        else:
            tr = max(candle.high - candle.low,
                     abs(candle.high - prev_close),
                     abs(candle.low - prev_close))

        self._prev_close = candle.close
        self.atr_history.append(tr)
        if len(self.atr_history) > self.atr_period:
            self.atr_history = self.atr_history[-self.atr_period:]

    def _get_current_atr(self) -> float:
        """Get current ATR value."""
        if not self.atr_history:
            return 0.0
        return sum(self.atr_history) / len(self.atr_history)

    def _build_manifest(self, candles: List[HistoricalCandle],
                        data_meta: Optional[DatasetMetadata],
                        dq_report: DataQualityReport) -> Dict:
        """Build the run manifest for reproducibility."""
        return {
            "run_id": self.run_id,
            "symbol": self.config.symbol,
            "strategy_id": self.config.strategy_id,
            "strategy_mode": self.config.strategy_mode,
            "primary_timeframe": self.config.primary_timeframe,
            "higher_timeframes": self.config.higher_timeframes,
            "start_time": candles[0].timestamp.isoformat() if candles else "",
            "end_time": candles[-1].timestamp.isoformat() if candles else "",
            "initial_balance": self.config.initial_balance,
            "random_seed": self.config.random_seed,
            "execution_assumptions": {
                "spread_model": self.config.execution_config.spread_model,
                "fixed_spread": self.config.execution_config.fixed_spread,
                "slippage_model": self.config.execution_config.slippage_model,
                "fixed_slippage": self.config.execution_config.fixed_slippage,
                "commission_per_lot": self.config.execution_config.commission_per_lot,
                "latency_ms": self.config.execution_config.latency_ms,
                "partial_fill_probability": self.config.execution_config.partial_fill_probability,
                "conservative_sl_tp": self.config.conservative_sl_tp,
            },
            "risk_config": {
                "max_risk_per_trade": self.config.max_risk_per_trade,
                "max_daily_loss_percent": self.config.max_daily_loss_percent,
                "max_consecutive_losses": self.config.max_consecutive_losses,
                "max_exposure": self.config.max_exposure,
                "min_rr": self.config.min_rr,
            },
            "exit_config": {
                "trailing_stop_enabled": self.config.trailing_stop_enabled,
                "trailing_atr_multiplier": self.config.trailing_atr_multiplier,
                "break_even_enabled": self.config.break_even_enabled,
                "max_holding_bars": self.config.max_holding_bars,
            },
            "data_source": data_meta.source if data_meta else "UNKNOWN",
            "data_hash": data_meta.data_hash if data_meta else "",
            "data_quality_score": dq_report.quality_score,
            "bars_processed": len(candles),
            "trades_count": len(self.portfolio.closed_trades),
            "no_trade_count": self.no_trade_count,
            "blocked_count": self.blocked_count,
            "status": "COMPLETED",
        }
