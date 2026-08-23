"use client";
import { useQuery } from "@tanstack/react-query";
import { IconActivity, IconAlertTriangle } from "@tabler/icons-react";
import { fetchLiveMarketSnapshot } from "@/lib/admin-market-data-api";

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

function MonitorCard({ title, note }: { title: string; note: string }) {
  return (
    <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4 opacity-80">
      <div className="text-sm font-medium text-pat-text-primary mb-2">{title}</div>
      <div className="text-xs text-pat-warning">Monitoring pending</div>
      <div className="text-xs text-pat-text-muted mt-1">{note}</div>
    </div>
  );
}

export default function AdminMarketDataPage() {
  const { data: snapshot, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-market-data-snapshot"],
    queryFn: fetchLiveMarketSnapshot,
    refetchInterval: 10000,
  });

  const indicators = snapshot?.indicators ?? {};
  const indicatorEntries = Object.entries(indicators).filter(
    ([, v]) => v !== undefined && v !== null
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Market Data Feed Health</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Live snapshot values are REAL (GET /market/snapshot). Sub-feed monitoring metrics are not yet exposed by a backend endpoint.
        </p>
      </div>

      {error && (
        <DegradedBanner>
          Go engine market snapshot unavailable:{" "}
          {error instanceof Error ? error.message : "unknown error"}. Live values cannot be shown.
        </DegradedBanner>
      )}

      {/* REAL live snapshot */}
      {snapshot?.tick && (
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold text-pat-text-primary flex items-center gap-2">
              <IconActivity size={16} /> Live Market Data
            </h2>
            <span className="text-xs px-2 py-0.5 rounded-full bg-pat-success/10 text-pat-success">
              {snapshot.source || "UNKNOWN"}
            </span>
          </div>
          <div className="grid grid-cols-3 md:grid-cols-6 gap-3 text-sm">
            <div>
              <div className="text-xs text-pat-text-muted">Bid</div>
              <div className="font-mono text-pat-success">{snapshot.tick.bid?.toFixed(2)}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">Ask</div>
              <div className="font-mono text-pat-danger">{snapshot.tick.ask?.toFixed(2)}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">Spread</div>
              <div className="font-mono text-pat-text-primary">{snapshot.tick.spread?.toFixed(2)}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">Volume</div>
              <div className="font-mono text-pat-text-primary">{snapshot.tick.volume}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">Broker</div>
              <div className="text-xs text-pat-text-secondary">{snapshot.broker || "—"}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">Node</div>
              <div className="text-xs text-pat-text-secondary">{snapshot.node || "—"}</div>
            </div>
          </div>
        </div>
      )}

      {indicatorEntries.length > 0 && (
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Live Indicator Values</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-2">
            {indicatorEntries.map(([key, val]) => (
              <div key={key} className="rounded-md bg-pat-bg-surface-secondary/50 px-3 py-2">
                <div className="text-xs text-pat-text-secondary">{key}</div>
                <div className="text-sm font-mono text-pat-text-primary">
                  {typeof val === "boolean" ? (val ? "Yes" : "No") : typeof val === "number" ? val.toFixed(4) : String(val)}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {isLoading && <div className="text-xs text-pat-text-muted">Loading market snapshot...</div>}

      {/* DEGRADED monitoring panels */}
      <div>
        <h2 className="text-sm font-medium text-pat-text-primary mb-2">Feed Monitoring (Pending Backend)</h2>
        <DegradedBanner>
          No dedicated feed-health / divergence / tick-rate / latency / candle-health / backfill endpoint exists.
          These panels show the intended schema only and must not be interpreted as live metrics.
        </DegradedBanner>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mt-3">
          <MonitorCard title="Divergence" note="Compares engine vs broker feed; endpoint pending." />
          <MonitorCard title="Tick Rate" note="Ticks/sec; endpoint pending." />
          <MonitorCard title="Latency" note="Feed-to-engine p50/p95/p99; endpoint pending." />
          <MonitorCard title="Candle Health" note="Gap/stale detection; endpoint pending." />
          <MonitorCard title="Backfill" note="History completeness; endpoint pending." />
          <button
            onClick={() => refetch()}
            className="bg-pat-bg-surface-secondary border border-pat-border rounded-lg px-4 py-2 text-sm text-pat-text-primary hover:opacity-90"
          >
            Retry live snapshot
          </button>
        </div>
      </div>
    </div>
  );
}
