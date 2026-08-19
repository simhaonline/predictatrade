"use client";
import StatusBadge from "@/components/ui/status-badge";
import { IconTestPipe } from "@tabler/icons-react";

const STRATEGIES = [
  { name: "STANDARD_SCALPING", timeframes: "M1/M5 + M15/M30", threshold: "65", minRR: "1.2", cooldown: "15m" },
  { name: "ULTRA_SCALPING", timeframes: "M1 + M5", threshold: "85", minRR: "1.0", cooldown: "15m" },
  { name: "STANDARD_SWING", timeframes: "M15/M30/H1 + H4/D1", threshold: "55", minRR: "1.8", cooldown: "120m" },
  { name: "TREND_SWING", timeframes: "H1/H4 + D1/W1", threshold: "50", minRR: "2.5", cooldown: "360m" },
];

export default function AdminBacktestingPage() {

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Backtesting Reports</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Research plane backtesting framework with walk-forward analysis and Monte Carlo robustness.</p>
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4 flex items-center gap-2"><IconTestPipe size={16} /> Backtesting Framework</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Event-Driven Engine</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Walk-Forward Analysis</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Monte Carlo Robustness</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Parameter Sensitivity</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">No-Lookahead Guarantees</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Realistic Execution Model</span><StatusBadge status="ACTIVE" size="sm" /></div>
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Spread/Slippage</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Commission/Latency</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Partial Fills/Rejects</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Trailing Stop/Break-Even</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Time Exit</span><StatusBadge status="ACTIVE" size="sm" /></div>
            <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Deterministic Mode</span><StatusBadge status="ACTIVE" size="sm" /></div>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {STRATEGIES.map((s) => (
          <div key={s.name} className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
            <div className="flex items-center justify-between mb-3">
              <span className="text-sm font-semibold text-pat-text-primary">{s.name}</span>
              <StatusBadge status="AVAILABLE" size="sm" />
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
              <div><span className="text-pat-text-muted">Timeframes: </span><span className="text-pat-text-primary">{s.timeframes}</span></div>
              <div><span className="text-pat-text-muted">Threshold: </span><span className="text-pat-text-primary">{s.threshold}</span></div>
              <div><span className="text-pat-text-muted">Min RR: </span><span className="text-pat-text-primary">{s.minRR}</span></div>
              <div><span className="text-pat-text-muted">Cooldown: </span><span className="text-pat-text-primary">{s.cooldown}</span></div>
            </div>
          </div>
        ))}
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-2">CLI Usage</h2>
        <pre className="text-xs text-pat-text-secondary bg-pat-bg-surface rounded p-3 overflow-x-auto"><code>{`# Run a backtest
cd research && python3 -m patresearch.backtesting.cli run --strategy STANDARD_SCALPING --seed 42

# Walk-forward analysis
cd research && python3 -m patresearch.backtesting.cli walk-forward --strategy STANDARD_SCALPING

# Monte Carlo
cd research && python3 -m patresearch.backtesting.cli monte-carlo --runs 1000`}</code></pre>
      </div>
    </div>
  );
}
