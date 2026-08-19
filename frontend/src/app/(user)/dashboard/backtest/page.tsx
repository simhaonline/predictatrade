"use client";
import StatusBadge from "@/components/ui/status-badge";

const STRATEGIES = [
  { name: "STANDARD_SCALPING", timeframes: "M1/M5 + M15/M30", threshold: "65", minRR: "1.2" },
  { name: "ULTRA_SCALPING", timeframes: "M1 + M5", threshold: "85", minRR: "1.0" },
  { name: "STANDARD_SWING", timeframes: "M15/M30/H1 + H4/D1", threshold: "55", minRR: "1.8" },
  { name: "TREND_SWING", timeframes: "H1/H4 + D1/W1", threshold: "50", minRR: "2.5" },
];

export default function UserBacktestPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Backtest</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Available backtesting reports from the research plane.</p>
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Backtesting Framework</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Event-Driven Engine</span><StatusBadge status="ACTIVE" size="sm" /></div>
          <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Walk-Forward Analysis</span><StatusBadge status="ACTIVE" size="sm" /></div>
          <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">Monte Carlo Robustness</span><StatusBadge status="ACTIVE" size="sm" /></div>
          <div className="flex items-center justify-between"><span className="text-sm text-pat-text-secondary">No-Lookahead Guarantee</span><StatusBadge status="ACTIVE" size="sm" /></div>
        </div>
      </div>

      <div className="space-y-3">
        {STRATEGIES.map((s) => (
          <div key={s.name} className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-semibold text-pat-text-primary">{s.name}</span>
              <StatusBadge status="AVAILABLE" size="sm" />
            </div>
            <div className="grid grid-cols-3 gap-3 text-xs">
              <div><span className="text-pat-text-muted">Timeframes: </span><span className="text-pat-text-primary">{s.timeframes}</span></div>
              <div><span className="text-pat-text-muted">Threshold: </span><span className="text-pat-text-primary">{s.threshold}</span></div>
              <div><span className="text-pat-text-muted">Min RR: </span><span className="text-pat-text-primary">{s.minRR}</span></div>
            </div>
          </div>
        ))}
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-2">How to Run</h2>
        <pre className="text-xs text-pat-text-secondary bg-pat-bg-surface rounded p-3 overflow-x-auto"><code>{`cd research && python3 -m patresearch.backtesting.cli run --strategy STANDARD_SCALPING --seed 42`}</code></pre>
      </div>
    </div>
  );
}
