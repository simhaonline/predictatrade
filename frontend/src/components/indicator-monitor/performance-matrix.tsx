"use client";
import { useMemo, useState } from "react";
import type { PerformanceMetric } from "@/lib/use-indicator-liveness";
import { getPerformanceColor } from "@/lib/use-indicator-liveness";
import { strategyLabel } from "@/lib/strategy-labels";

interface PerformanceMatrixProps {
  performance: PerformanceMetric[];
  strategies: string[];
}

export function PerformanceMatrix({ performance, marketClosed }: { performance: PerformanceMetric[]; marketClosed?: boolean }) {
  const [sortBy, setSortBy] = useState<"hitRate" | "avgRMultiple" | "signalFrequency">("hitRate");

  const sorted = useMemo(() => {
    return [...performance].sort((a, b) => {
      const aVal = a[sortBy] ?? -999;
      const bVal = b[sortBy] ?? -999;
      return bVal - aVal;
    });
  }, [performance, sortBy]);

  return (
    <div className="space-y-4">
      {/* Performance Matrix */}
      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-pat-text-primary">Performance Matrix</h2>
          <div className="flex gap-2 text-xs">
            <SortButton active={sortBy === "hitRate"} onClick={() => setSortBy("hitRate")}>Hit Rate</SortButton>
            <SortButton active={sortBy === "avgRMultiple"} onClick={() => setSortBy("avgRMultiple")}>Avg R</SortButton>
            <SortButton active={sortBy === "signalFrequency"} onClick={() => setSortBy("signalFrequency")}>Frequency</SortButton>
          </div>
        </div>
        {/* Show only indicator×strategy pairs that actually have contributions (tradeCount > 0).
            164 rows of mostly no-data is unreadable noise. */}
        {sorted.filter(m => m.tradeCount > 0).length === 0 && (
          <div className="rounded border border-pat-warning/20 bg-pat-warning/5 p-4 mb-3">
            <div className="text-sm text-pat-warning mb-1">No performance data available</div>
            <div className="text-xs text-pat-text-muted">
              Performance metrics require closed trade data or evidence-matched signals.
              This will populate automatically as trades are closed and recorded by the engine.
            </div>
          </div>
        )}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-pat-text-muted border-b border-pat-border">
                <th className="text-left py-2 px-3">Indicator</th>
                <th className="text-left py-2 px-3">Strategy</th>
                <th className="text-right py-2 px-3">Hit Rate</th>
                <th className="text-right py-2 px-3">Avg R</th>
                <th className="text-right py-2 px-3">Contrib.</th>
                <th className="text-right py-2 px-3">Freq.</th>
                <th className="text-right py-2 px-3">Accuracy</th>
                <th className="text-right py-2 px-3">Trades</th>
                <th className="text-center py-2 px-3">Level</th>
              </tr>
            </thead>
            <tbody>
              {sorted.filter(m => m.tradeCount > 0).map((m, i) => (
                <tr key={`${m.indicatorKey}-${m.strategy}-${i}`} className="border-b border-pat-border/50 hover:bg-pat-bg-surface-secondary/30">
                  <td className="py-2 px-3 text-pat-text-primary">{m.indicatorKey}</td>
                  <td className="py-2 px-3 text-xs text-pat-text-secondary">{m.strategy}</td>
                  <td className="py-2 px-3 text-right font-mono text-xs">{m.hitRate !== null ? `${m.hitRate.toFixed(1)}%` : "—"}</td>
                  <td className="py-2 px-3 text-right font-mono text-xs">{m.avgRMultiple !== null ? m.avgRMultiple.toFixed(2) : "—"}</td>
                  <td className="py-2 px-3 text-right font-mono text-xs">{m.contributionScore !== null ? m.contributionScore.toFixed(2) : "—"}</td>
                  <td className="py-2 px-3 text-right font-mono text-xs">{m.signalFrequency !== null ? m.signalFrequency.toFixed(1) : "—"}</td>
                  <td className="py-2 px-3 text-right font-mono text-xs">{m.signalAccuracy !== null ? `${m.signalAccuracy.toFixed(1)}%` : "—"}</td>
                  <td className="py-2 px-3 text-right font-mono text-xs text-pat-text-muted">{m.tradeCount}</td>
                  <td className={`py-2 px-3 text-center text-xs ${getPerformanceColor(m.performanceLevel)}`}>{m.performanceLevel}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Ranking list */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
          <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Top Performers</h3>
          <RankingList metrics={sorted.filter((m) => m.tradeCount > 0 && m.hitRate !== null).slice(0, 5)} />
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
          <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Needs Attention</h3>
          <NeedsAttention performance={performance} marketClosed={marketClosed} />
        </div>
      </div>

      {performance.every((p) => p.tradeCount === 0) && (
        <div className="rounded-lg border border-pat-warning/20 bg-pat-warning/10 p-4">
          <div className="text-sm text-pat-warning">No closed-trade performance data yet</div>
          <div className="text-xs text-pat-text-muted mt-1">
            Performance metrics populate as signals resolve with realized P&amp;L.
            The engine is generating signals across all active strategies — check the Signals page for live activity.
          </div>
        </div>
      )}
    </div>
  );
}

function SortButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`px-2 py-1 rounded ${active ? "bg-pat-bg-surface-secondary text-pat-text-primary" : "text-pat-text-muted"}`}
    >
      {children}
    </button>
  );
}

function RankingList({ metrics }: { metrics: PerformanceMetric[] }) {
  if (metrics.length === 0) {
    return <div className="text-xs text-pat-text-muted">No data available.</div>;
  }
  return (
    <div className="space-y-2">
      {metrics.map((m, i) => (
        <div key={i} className="flex items-center justify-between text-xs">
          <span className="text-pat-text-secondary">{m.indicatorKey} · {m.strategy}</span>
          <span className={`font-mono ${getPerformanceColor(m.performanceLevel)}`}>
            {(m.hitRate ?? 0).toFixed(0)}% / {(m.avgRMultiple ?? 0).toFixed(2)}R
          </span>
        </div>
      ))}
    </div>
  );
}

/**
 * Needs Attention panel — shows actionable items beyond raw performance numbers:
 * 1. Strategies with zero directional signals (TREND_SWING gap, etc.)
 * 2. Indicators contributing to signals but performing poorly
 * 3. Indicators flagged as stale/disabled by liveness
 */
function NeedsAttention({ performance, marketClosed }: { performance: PerformanceMetric[]; marketClosed?: boolean }) {
  const items: string[] = [];

  // 1. Strategies with zero directional signals — EXPECTED during closed
  // market (check.md 2026-08-30 #3): say so once, auto-calibration note.
  if (marketClosed) {
    items.push("Market closed — no directional signals expected until re-open. Auto-calibration (walk-forward refresh) runs automatically at re-open.");
  } else {
    const strategies = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB"];
    const stratSignalCounts = new Map<string, number>();
    for (const m of performance) {
      stratSignalCounts.set(m.strategy, (stratSignalCounts.get(m.strategy) ?? 0) + m.tradeCount);
    }
    for (const st of strategies) {
      const count = stratSignalCounts.get(st) ?? 0;
      if (count === 0) {
        items.push(`No directional signals from ${strategyLabel(st)} — auto-calibration queued (walk-forward refresh at re-open); otherwise market conditions are unfavorable`);
      }
    }
  }

  // 2. Indicators with poor performance (low hit rate where we have data)
  for (const m of performance) {
    if (m.tradeCount > 0 && m.hitRate !== null && m.hitRate < 40) {
      items.push(`${m.indicatorKey} on ${m.strategy}: ${m.hitRate.toFixed(0)}% hit rate over ${m.tradeCount} trades`);
    }
    if (m.tradeCount > 0 && m.avgRMultiple !== null && m.avgRMultiple < 0) {
      items.push(`${m.indicatorKey} on ${m.strategy}: negative avg R (${m.avgRMultiple.toFixed(2)}) over ${m.tradeCount} trades`);
    }
  }

  // 3. Show evidence-matched indicators and their projected R:R
  const withProjectedRR = performance
    .filter(m => m.tradeCount > 0 && m.avgRMultiple !== null)
    .sort((a, b) => (a.avgRMultiple ?? 0) - (b.avgRMultiple ?? 0));
  for (const m of withProjectedRR.slice(0, 3)) {
    if (m.avgRMultiple !== null && m.avgRMultiple < 0.5 && m.avgRMultiple > 0) {
      items.push(`${m.indicatorKey} on ${m.strategy}: projected R:R only ${m.avgRMultiple.toFixed(2)}`);
    }
  }

  if (items.length === 0) {
    return <div className="text-xs text-pat-text-muted">All active indicators are performing within expected parameters.</div>;
  }

  return (
    <div className="space-y-2 max-h-60 overflow-y-auto">
      {items.slice(0, 8).map((item, i) => (
        <div key={i} className="flex items-start gap-2 text-xs">
          <span className="text-pat-warning mt-0.5 shrink-0">⚠</span>
          <span className="text-pat-text-secondary">{item}</span>
        </div>
      ))}
    </div>
  );
}
