"use client";
import { useMemo, useState } from "react";
import type { PerformanceMetric } from "@/lib/use-indicator-liveness";
import { getPerformanceColor } from "@/lib/use-indicator-liveness";

interface PerformanceMatrixProps {
  performance: PerformanceMetric[];
  strategies: string[];
}

export function PerformanceMatrix({ performance }: PerformanceMatrixProps) {
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
              {sorted.map((m, i) => (
                <tr key={`${m.indicatorKey}-${m.strategy}-${i}`} className="border-b border-pat-border/50 hover:bg-pat-bg-surface-secondary/30">
                  <td className="py-2 px-3 text-pat-text-primary">{m.indicatorKey}</td>
                  <td className="py-2 px-3 text-xs text-pat-text-secondary">{m.strategy.replace(/_/g, " ")}</td>
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
          <RankingList metrics={sorted.filter((m) => m.hitRate !== null).slice(0, 5)} />
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
          <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Needs Attention</h3>
          <RankingList metrics={sorted.filter((m) => m.hitRate !== null).slice(-5).reverse()} />
        </div>
      </div>

      {performance.every((p) => p.performanceLevel === "no-data") && (
        <div className="rounded-lg border border-pat-warning/20 bg-pat-warning/10 p-4">
          <div className="text-sm text-pat-warning">No performance data available</div>
          <div className="text-xs text-pat-text-muted mt-1">
            Performance metrics require closed trade data with per-indicator evidence tracking.
            This will populate automatically as trades are closed and recorded by the engine.
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
          <span className="text-pat-text-secondary">{m.indicatorKey}</span>
          <span className={`font-mono ${getPerformanceColor(m.performanceLevel)}`}>
            {(m.hitRate ?? 0).toFixed(0)}% / {(m.avgRMultiple ?? 0).toFixed(2)}R
          </span>
        </div>
      ))}
    </div>
  );
}
