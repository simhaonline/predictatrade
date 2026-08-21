"use client";
import type { IndicatorLiveness, PerformanceMetric } from "@/lib/use-indicator-liveness";

interface SummaryCardsProps {
  liveness: IndicatorLiveness[];
  performance: PerformanceMetric[];
}

export function SummaryCards({ liveness, performance }: SummaryCardsProps) {
  const liveCount = liveness.filter((i) => i.status === "live").length;
  const activeCount = liveness.filter((i) => i.activeStatus === "active" || i.activeStatus === "armed").length;
  const staleCount = liveness.filter((i) => i.status === "stale").length;
  const disabledCount = liveness.filter((i) => i.status === "disabled").length;

  const perfWith = performance.filter((p) => p.performanceLevel !== "no-data" && p.signalFrequency !== null);
  const best = perfWith.length > 0
    ? perfWith.reduce((a, b) => (a.signalFrequency ?? 0) > (b.signalFrequency ?? 0) ? a : b)
    : null;
  const worst = perfWith.length > 0
    ? perfWith.reduce((a, b) => (a.signalFrequency ?? 999) < (b.signalFrequency ?? 999) ? a : b)
    : null;

  const cards = [
    {
      label: "Indicators Live", value: `${liveCount}`, sub: `/ ${liveness.length} total`,
      color: "text-pat-success", dot: "bg-pat-success",
    },
    {
      label: "Active / Armed", value: `${activeCount}`, sub: "triggering",
      color: "text-pat-warning", dot: "bg-pat-warning",
    },
    {
      label: "Stale", value: `${staleCount}`, sub: "need attention",
      color: "text-pat-danger", dot: "bg-pat-danger",
    },
    {
      label: "Disabled", value: `${disabledCount}`, sub: "offline",
      color: "text-pat-text-muted", dot: "bg-pat-text-muted",
    },
    {
      label: "Top Performer", value: best ? best.indicatorKey.slice(0, 12) : "—", sub: best ? `${(best.signalFrequency ?? 0).toFixed(0)}% freq` : "collecting",
      color: "text-pat-success", dot: "bg-pat-success",
    },
    {
      label: "Lowest Activity", value: worst ? worst.indicatorKey.slice(0, 12) : "—", sub: worst ? `${(worst.signalFrequency ?? 0).toFixed(0)}% freq` : "collecting",
      color: "text-pat-danger", dot: "bg-pat-danger",
    },
  ];

  return (
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
      {cards.map((card) => (
        <div
          key={card.label}
          className="rounded-xl border border-pat-border bg-pat-bg-surface p-4 hover:border-pat-border/80 transition-colors"
        >
          <div className="flex items-center gap-2 mb-1">
            <span className={`w-2 h-2 rounded-full ${card.dot}`} />
            <span className="text-[11px] text-pat-text-muted uppercase tracking-wide">{card.label}</span>
          </div>
          <div className={`text-lg font-bold ${card.color} tabular-nums`}>{card.value}</div>
          <div className="text-[11px] text-pat-text-muted mt-0.5">{card.sub}</div>
        </div>
      ))}
    </div>
  );
}
