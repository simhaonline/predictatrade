"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconBroadcast } from "@tabler/icons-react";
import StatusBadge from "@/components/ui/status-badge";

interface AgentsStatusResponse {
  agents_connected: number;
  agents_online: boolean;
  data_agents_connected: number;
  snapshot_count: number;
  last_market_data_at: string;
  last_snapshot_at: string;
  data_stale_secs: number;
  data_health: "HEALTHY" | "STALE" | "CRITICAL" | "NO_DATA";
  market_closed: boolean;
  next_market_open_utc: string;
  mt4_connected: number;
  mt5_connected: number;
  timestamp: string;
  server_time: string;
}

/** Accent colours for the feed-health banner — PAT semantic tokens only. */
const HEALTH_BANNER: Record<string, { border: string; bg: string; text: string; dot: string }> = {
  HEALTHY: { border: "border-pat-success/40", bg: "bg-pat-success/10", text: "text-pat-text-primary", dot: "bg-pat-success" },
  STALE: { border: "border-pat-warning/40", bg: "bg-pat-warning/10", text: "text-pat-text-primary", dot: "bg-pat-warning" },
  CRITICAL: { border: "border-pat-danger/40", bg: "bg-pat-danger/10", text: "text-pat-text-primary", dot: "bg-pat-danger" },
  NO_DATA: { border: "border-pat-border", bg: "bg-pat-bg-surface", text: "text-pat-text-primary", dot: "bg-pat-text-muted" },
};

function Metric({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
      <div className="text-xs uppercase tracking-wide text-pat-text-muted">{label}</div>
      <div className="mt-1 text-xl font-semibold text-pat-text-primary">{value}</div>
    </div>
  );
}

export default function AdminAgentMeshPage() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["agents-status"],
    // Real endpoint served by the Go realtime engine (agent hub + provider).
    queryFn: async () => (await customInstance.get("/agents/status")).data as AgentsStatusResponse,
    refetchInterval: 10000,
  });

  const health = HEALTH_BANNER[data?.data_health || ""] || HEALTH_BANNER.NO_DATA;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Agent Mesh</h1>
          <p className="text-sm text-pat-text-secondary mt-1">
            Windows Agent bridge + AI Agent Mesh connectivity (real-time, sourced from the Go engine).
          </p>
        </div>
        {data?.timestamp && (
          <span className="text-xs text-pat-text-muted whitespace-nowrap">
            Updated {new Date(data.timestamp).toLocaleTimeString()}
          </span>
        )}
      </div>

      {isLoading && (
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-8 text-center text-sm text-pat-text-muted">
          Loading agent mesh state…
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-pat-danger/40 bg-pat-danger/10 p-4 text-sm text-pat-danger">
          Failed to load agent mesh: {(error as Error)?.message || "unknown error"}
          <button onClick={() => refetch()} className="ml-3 underline">
            retry
          </button>
        </div>
      )}

      {data && (
        <>
          <div className={`flex items-center gap-3 rounded-lg border p-4 ${health.border} ${health.bg}`}>
            <span className={`h-2.5 w-2.5 rounded-full ${health.dot}`} />
            <IconBroadcast size={24} className="text-pat-text-secondary" />
            <div>
              <div className="text-sm text-pat-text-secondary">Data Feed Health</div>
              <div className={`text-lg font-semibold ${health.text}`}>{data.data_health}</div>
            </div>
            <div className="ml-auto flex items-center gap-2">
              <StatusBadge status={data.data_health} size="sm" />
              {data.market_closed && (
                <span className="rounded-md bg-pat-bg-surface-secondary border border-pat-border px-2 py-1 text-xs text-pat-text-secondary">
                  Market closed — next open {new Date(data.next_market_open_utc).toUTCString()}
                </span>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <Metric label="Agents Connected" value={data.agents_connected} />
            <Metric label="Data Agents" value={data.data_agents_connected} />
            <Metric label="MT4 Connected" value={data.mt4_connected} />
            <Metric label="MT5 Connected" value={data.mt5_connected} />
            <Metric label="Snapshots" value={data.snapshot_count} />
            <Metric
              label="Stale (s)"
              value={data.data_stale_secs < 0 ? "—" : data.data_stale_secs}
            />
            <Metric
              label="Last Snapshot"
              value={
                data.last_snapshot_at ? new Date(data.last_snapshot_at).toLocaleTimeString() : "—"
              }
            />
            <Metric
              label="Last Market Data"
              value={
                data.last_market_data_at ? new Date(data.last_market_data_at).toLocaleTimeString() : "—"
              }
            />
          </div>

          <p className="text-xs text-pat-text-muted">
            Agents online: {String(data.agents_online)} · Server time:{" "}
            {new Date(data.server_time).toUTCString()}
          </p>
        </>
      )}
    </div>
  );
}