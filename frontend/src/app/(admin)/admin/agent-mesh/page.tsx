"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconBroadcast } from "@tabler/icons-react";

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

const HEALTH_STYLES: Record<string, string> = {
  HEALTHY: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30",
  STALE: "bg-amber-500/15 text-amber-300 border-amber-500/30",
  CRITICAL: "bg-rose-500/15 text-rose-300 border-rose-500/30",
  NO_DATA: "bg-slate-500/15 text-slate-300 border-slate-500/30",
};

function Metric({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/50 p-4">
      <div className="text-xs uppercase tracking-wide text-slate-400">{label}</div>
      <div className="mt-1 text-xl font-semibold text-white">{value}</div>
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-white">Agent Mesh</h1>
          <p className="text-sm text-slate-400">
            Windows Agent bridge + AI Agent Mesh connectivity (real-time, sourced from the Go engine).
          </p>
        </div>
        {data?.timestamp && (
          <span className="text-xs text-slate-500">
            Updated {new Date(data.timestamp).toLocaleTimeString()}
          </span>
        )}
      </div>

      {isLoading && (
        <div className="rounded-lg border border-slate-700 bg-slate-900/50 p-8 text-center text-slate-400">
          Loading agent mesh state…
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-rose-500/40 bg-rose-500/10 p-4 text-rose-300">
          Failed to load agent mesh: {(error as Error)?.message || "unknown error"}
          <button onClick={() => refetch()} className="ml-3 underline">
            retry
          </button>
        </div>
      )}

      {data && (
        <>
          <div
            className={`flex items-center gap-3 rounded-xl border p-4 ${
              HEALTH_STYLES[data.data_health] || HEALTH_STYLES.NO_DATA
            }`}
          >
            <IconBroadcast size={28} />
            <div>
              <div className="text-sm font-medium">Data Feed Health</div>
              <div className="text-lg font-semibold">{data.data_health}</div>
            </div>
            {data.market_closed && (
              <span className="ml-auto rounded-md bg-slate-800 px-2 py-1 text-xs text-slate-300">
                Market closed — next open {new Date(data.next_market_open_utc).toUTCString()}
              </span>
            )}
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

          <p className="text-xs text-slate-500">
            Agents online: {String(data.agents_online)} · Server time:{" "}
            {new Date(data.server_time).toUTCString()}
          </p>
        </>
      )}
    </div>
  );
}
