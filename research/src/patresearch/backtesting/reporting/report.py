"""Report generator and run manifest for backtest results.

Generates:
- summary.json — overview metrics
- trades.csv — trade records
- equity.csv — equity curve
- metrics.json — full metrics
- configuration.json — run config
- data_quality.json — data quality report
- run_manifest.json — full reproducibility manifest
"""
from __future__ import annotations

from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from typing import List, Optional, Dict
import json
import csv
import os
import uuid

from ..engine.core import BacktestRunResult
from ..engine.portfolio import TradeRecord
from ..analytics.metrics import PerformanceMetrics


@dataclass
class RunManifest:
    """Complete reproducibility manifest for a backtest run."""
    run_id: str
    symbol: str
    strategy: str
    strategy_mode: str
    timeframe: str
    start_timestamp: str
    end_timestamp: str
    initial_balance: float
    configuration: Dict
    random_seed: int
    execution_assumptions: Dict
    spread_model: str
    slippage_model: str
    commission: float
    latency_assumptions: Dict
    partial_fill_assumptions: Dict
    data_source: str
    data_hash: str
    feature_version: str
    model_version: str
    git_commit_sha: str
    application_version: str
    start_time: str
    completion_time: str
    status: str
    metrics: Dict
    artifact_locations: Dict

    def to_dict(self) -> Dict:
        return asdict(self)

    def to_json(self, filepath: str):
        with open(filepath, "w") as f:
            json.dump(self.to_dict(), f, indent=2, default=str)


class ReportGenerator:
    """Generates complete backtest reports."""

    def __init__(self, output_dir: str = "backtest_reports"):
        self.output_dir = output_dir
        os.makedirs(output_dir, exist_ok=True)

    def generate(self, result: BacktestRunResult,
                 git_commit_sha: str = "",
                 application_version: str = "1.0.0") -> Dict:
        """Generate all report files for a backtest run.

        Returns dict of artifact locations.
        """
        run_dir = os.path.join(self.output_dir, result.run_id)
        os.makedirs(run_dir, exist_ok=True)

        artifacts = {}

        # summary.json
        summary = {
            "run_id": result.run_id,
            "status": result.status,
            "bars_processed": result.bars_processed,
            "trades": len(result.trades),
            "no_trade_decisions": result.no_trade_count,
            "blocked_signals": result.blocked_count,
            "duration_seconds": result.duration_seconds,
        }
        if result.metrics:
            summary.update({
                "initial_balance": result.metrics.initial_balance,
                "final_balance": result.metrics.final_balance,
                "total_return_pct": result.metrics.total_return_pct,
                "sharpe_ratio": result.metrics.sharpe_ratio,
                "sortino_ratio": result.metrics.sortino_ratio,
                "max_drawdown_pct": result.metrics.max_drawdown_pct,
                "win_rate_pct": result.metrics.win_rate_pct,
                "profit_factor": result.metrics.profit_factor,
                "total_trades": result.metrics.total_trades,
            })
        artifacts["summary"] = self._write_json(summary, run_dir, "summary.json")

        # trades.csv
        if result.trades:
            artifacts["trades"] = self._write_trades_csv(result.trades, run_dir)

        # equity.csv
        if result.equity_curve:
            artifacts["equity"] = self._write_equity_csv(result.equity_curve, run_dir)

        # metrics.json
        if result.metrics:
            metrics_dict = asdict(result.metrics)
            artifacts["metrics"] = self._write_json(metrics_dict, run_dir, "metrics.json")

        # configuration.json
        config_dict = {
            "symbol": result.config.symbol,
            "strategy_id": result.config.strategy_id,
            "strategy_mode": result.config.strategy_mode,
            "primary_timeframe": result.config.primary_timeframe,
            "initial_balance": result.config.initial_balance,
            "random_seed": result.config.random_seed,
            "max_risk_per_trade": result.config.max_risk_per_trade,
            "max_daily_loss_percent": result.config.max_daily_loss_percent,
            "min_rr": result.config.min_rr,
            "trailing_stop_enabled": result.config.trailing_stop_enabled,
            "trailing_atr_multiplier": result.config.trailing_atr_multiplier,
            "break_even_enabled": result.config.break_even_enabled,
            "conservative_sl_tp": result.config.conservative_sl_tp,
        }
        artifacts["configuration"] = self._write_json(config_dict, run_dir, "configuration.json")

        # data_quality.json
        dq_summary = result.data_quality.summary()
        artifacts["data_quality"] = self._write_json(dq_summary, run_dir, "data_quality.json")

        # filter_contribution.json (per-filter edge reporting)
        if result.filter_contribution:
            artifacts["filter_contribution"] = self._write_json(
                result.filter_contribution, run_dir, "filter_contribution.json"
            )

        # run_manifest.json
        if result.manifest:
            manifest = RunManifest(
                run_id=result.run_id,
                symbol=result.config.symbol,
                strategy=result.config.strategy_id,
                strategy_mode=result.config.strategy_mode,
                timeframe=result.config.primary_timeframe,
                start_timestamp=result.manifest.get("start_time", ""),
                end_timestamp=result.manifest.get("end_time", ""),
                initial_balance=result.config.initial_balance,
                configuration=config_dict,
                random_seed=result.config.random_seed,
                execution_assumptions=result.manifest.get("execution_assumptions", {}),
                spread_model=result.config.execution_config.spread_model,
                slippage_model=result.config.execution_config.slippage_model,
                commission=result.config.execution_config.commission_per_lot,
                latency_assumptions={"latency_ms": result.config.execution_config.latency_ms},
                partial_fill_assumptions={"probability": result.config.execution_config.partial_fill_probability},
                data_source=result.manifest.get("data_source", "UNKNOWN"),
                data_hash=result.manifest.get("data_hash", ""),
                feature_version="1.0",
                model_version="1.0.0",
                git_commit_sha=git_commit_sha,
                application_version=application_version,
                start_time=datetime.now(timezone.utc).isoformat(),
                completion_time=datetime.now(timezone.utc).isoformat(),
                status=result.status,
                metrics=summary,
                artifact_locations=artifacts,
            )
            manifest_path = os.path.join(run_dir, "run_manifest.json")
            manifest.to_json(manifest_path)
            artifacts["run_manifest"] = manifest_path

        return artifacts

    def _write_json(self, data: Dict, dirpath: str, filename: str) -> str:
        filepath = os.path.join(dirpath, filename)
        with open(filepath, "w") as f:
            json.dump(data, f, indent=2, default=str)
        return filepath

    def _write_trades_csv(self, trades: List[TradeRecord], dirpath: str) -> str:
        filepath = os.path.join(dirpath, "trades.csv")
        if not trades:
            with open(filepath, "w") as f:
                f.write("no_trades\n")
            return filepath

        fields = ["trade_id", "strategy_id", "direction", "entry_time", "entry_price",
                  "exit_time", "exit_price", "exit_reason", "size", "pnl", "pnl_r",
                  "commission", "duration_bars", "regime", "session", "confluence",
                  "confidence", "setup_grade"]

        with open(filepath, "w", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=fields, extrasaction="ignore")
            writer.writeheader()
            for t in trades:
                row = {
                    "trade_id": t.trade_id, "strategy_id": t.strategy_id,
                    "direction": t.direction,
                    "entry_time": t.entry_time.isoformat() if t.entry_time else "",
                    "entry_price": t.entry_price,
                    "exit_time": t.exit_time.isoformat() if t.exit_time else "",
                    "exit_price": t.exit_price, "exit_reason": t.exit_reason,
                    "size": t.size, "pnl": t.pnl, "pnl_r": t.pnl_r,
                    "commission": t.commission, "duration_bars": t.duration_bars,
                    "regime": t.regime, "session": t.session,
                    "confluence": t.confluence, "confidence": t.confidence,
                    "setup_grade": t.setup_grade,
                }
                writer.writerow(row)
        return filepath

    def _write_equity_csv(self, equity_curve: List[Dict], dirpath: str) -> str:
        filepath = os.path.join(dirpath, "equity.csv")
        with open(filepath, "w", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=["timestamp", "equity", "balance", "drawdown"])
            writer.writeheader()
            for ep in equity_curve:
                writer.writerow({
                    "timestamp": ep.get("timestamp", ""),
                    "equity": ep.get("equity", 0),
                    "balance": ep.get("balance", 0),
                    "drawdown": ep.get("drawdown", 0),
                })
        return filepath
