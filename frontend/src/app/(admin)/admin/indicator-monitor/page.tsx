"use client";
import { useState } from "react";
import { useIndicatorLiveness } from "@/lib/use-indicator-liveness";
import { useSignalPerformance } from "@/lib/use-signal-performance";
import { SummaryCards } from "@/components/indicator-monitor/summary-cards";
import { LivenessMatrix } from "@/components/indicator-monitor/liveness-matrix";
import { ActiveReactiveTable } from "@/components/indicator-monitor/active-reactive-table";
import { PerformanceMatrix } from "@/components/indicator-monitor/performance-matrix";
import { IndicatorCharts } from "@/components/indicator-monitor/indicator-charts";

type Tab = "overview" | "liveness" | "active" | "performance" | "charts";

export default function IndicatorMonitorPage() {
  const [tab, setTab] = useState<Tab>("overview");
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [refreshMs, setRefreshMs] = useState(500);
  const { liveness, history, strategies, snapshot } = useIndicatorLiveness(autoRefresh ? refreshMs : 60000);

  // Compute performance metrics from actual signal data
  const { performance, signals } = useSignalPerformance(liveness);

  const tabs: { id: Tab; label: string }[] = [
    { id: "overview", label: "Overview" },
    { id: "liveness", label: "Liveness" },
    { id: "active", label: "Active / Reactive" },
    { id: "performance", label: "Performance" },
    { id: "charts", label: "Charts" },
  ];

  // Source info
  const source = snapshot?.source || "—";
  const symbol = snapshot?.symbol || "XAUUSD";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Indicator Monitor</h1>
          <p className="text-sm text-pat-text-secondary mt-1">
            Real-time liveness, status, and performance monitoring for all {liveness.length} indicators.
            {signals.length > 0 && <span className="text-pat-text-muted"> · {signals.length} signals tracked</span>}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="text-xs text-pat-text-muted">
            <span className="text-pat-text-secondary">{symbol}</span> · {source}
          </div>
          <label className="flex items-center gap-1 text-xs text-pat-text-secondary cursor-pointer">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="accent-pat-success"
            />
            Auto-refresh
          </label>
          <select
            value={refreshMs}
            onChange={(e) => setRefreshMs(Number(e.target.value))}
            className="bg-pat-bg-surface-secondary text-xs text-pat-text-primary border border-pat-border rounded px-2 py-1"
            disabled={!autoRefresh}
          >
            <option value={500}>500ms</option>
            <option value={1000}>1s</option>
            <option value={2000}>2s</option>
            <option value={5000}>5s</option>
          </select>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-pat-border">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-4 py-2 text-sm transition-colors ${
              tab === t.id
                ? "text-pat-text-primary border-b-2 border-pat-success"
                : "text-pat-text-muted hover:text-pat-text-secondary"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === "overview" && (
        <div className="space-y-4">
          <SummaryCards liveness={liveness} performance={performance} />
          <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
            <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Quick Status</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <h3 className="text-xs text-pat-text-muted mb-2">Liveness Summary</h3>
                <div className="space-y-1">
                  {liveness.slice(0, 8).map((ind) => (
                    <div key={ind.key} className="flex items-center justify-between text-xs">
                      <span className="text-pat-text-secondary">{ind.label}</span>
                      <span className="flex items-center gap-1">
                        <span className={`inline-block w-2 h-2 rounded-full ${
                          ind.status === "live" ? "bg-pat-success" :
                          ind.status === "late" ? "bg-pat-warning" :
                          ind.status === "stale" ? "bg-pat-danger" : "bg-pat-text-muted"
                        }`} />
                        <span className="text-pat-text-muted">{ind.status}</span>
                      </span>
                    </div>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-xs text-pat-text-muted mb-2">Active / Armed</h3>
                <div className="space-y-1">
                  {liveness.filter((i) => i.activeStatus === "armed" || i.activeStatus === "active").slice(0, 8).map((ind) => (
                    <div key={ind.key} className="flex items-center justify-between text-xs">
                      <span className="text-pat-text-secondary">{ind.label}</span>
                      <span className="text-pat-warning">{ind.activeStatus}</span>
                    </div>
                  ))}
                  {liveness.filter((i) => i.activeStatus === "armed" || i.activeStatus === "active").length === 0 && (
                    <div className="text-xs text-pat-text-muted">No indicators currently armed or active.</div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === "liveness" && (
        <LivenessMatrix liveness={liveness} history={history} />
      )}

      {tab === "active" && (
        <ActiveReactiveTable liveness={liveness} />
      )}

      {tab === "performance" && (
        <PerformanceMatrix performance={performance} strategies={strategies} />
      )}

      {tab === "charts" && (
        <IndicatorCharts liveness={liveness} history={history} performance={performance} />
      )}

      {/* Data source notice */}
      <div className="text-xs text-pat-text-muted text-center">
        Observability layer only — does not affect trading or strategy logic.
        Data source: {snapshot?.source ? `Go engine (${snapshot.source})` : "No live connection"}
      </div>
    </div>
  );
}
