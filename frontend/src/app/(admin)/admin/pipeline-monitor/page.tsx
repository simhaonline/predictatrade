"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconListCheck } from "@tabler/icons-react";

interface PipelineStage {
  stage: string;
  name: string;
  status: string;
  count_5m?: number;
  last_at?: string;
  engines?: string[];
  note?: string;
  vetoed_5m?: number;
  detail?: string;
  backfill?: number;
}

interface PipelineMonitorResponse {
  pipeline: PipelineStage[];
  timestamp: string;
}

const STATUS_STYLES: Record<string, string> = {
  healthy: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30",
  idle: "bg-slate-500/15 text-slate-300 border-slate-500/30",
  live: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30",
  connected: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30",
  db_error: "bg-rose-500/15 text-rose-300 border-rose-500/30",
  degraded: "bg-amber-500/15 text-amber-300 border-amber-500/30",
};

export default function AdminPipelineMonitorPage() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["pipeline-monitor"],
    // Real endpoint served by the Go realtime engine.
    queryFn: async () => (await customInstance.get("/pipeline/monitor")).data as PipelineMonitorResponse,
    refetchInterval: 15000,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-white">Pipeline Monitor</h1>
          <p className="text-sm text-slate-400">
            Live Signal → Risk → Execution → Review pipeline (real-time, sourced from the Go engine).
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
          Loading pipeline state…
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-rose-500/40 bg-rose-500/10 p-4 text-rose-300">
          Failed to load pipeline monitor: {(error as Error)?.message || "unknown error"}
          <button onClick={() => refetch()} className="ml-3 underline">
            retry
          </button>
        </div>
      )}

      {data?.pipeline?.map((stage) => {
        const badge = STATUS_STYLES[stage.status] || "bg-slate-500/15 text-slate-300 border-slate-500/30";
        return (
          <div key={stage.stage} className="rounded-xl border border-slate-700 bg-slate-900/50 p-5">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <span className="text-lg font-semibold text-white">{stage.stage}</span>
                <span className={`rounded-full border px-2.5 py-0.5 text-xs font-medium ${badge}`}>
                  {stage.status}
                </span>
              </div>
              <span className="text-sm text-slate-300">{stage.name}</span>
            </div>

            <p className="mt-2 text-sm text-slate-400">{stage.detail || stage.note}</p>

            <div className="mt-3 flex flex-wrap gap-x-8 gap-y-2 text-sm">
              {typeof stage.count_5m === "number" && (
                <span className="text-slate-300">
                  Signals (5m): <span className="font-semibold text-white">{stage.count_5m}</span>
                </span>
              )}
              {stage.last_at && (
                <span className="text-slate-300">
                  Last at: <span className="font-semibold text-white">{stage.last_at}</span>
                </span>
              )}
              {typeof stage.vetoed_5m === "number" && (
                <span className="text-slate-300">
                  Vetoed (5m): <span className="font-semibold text-white">{stage.vetoed_5m}</span>
                </span>
              )}
              {typeof stage.backfill === "number" && (
                <span className="text-slate-300">
                  Backfill: <span className="font-semibold text-white">{stage.backfill}%</span>
                </span>
              )}
            </div>

            {stage.engines && stage.engines.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-2">
                {stage.engines.map((e) => (
                  <span key={e} className="rounded-md bg-slate-800 px-2 py-0.5 text-xs text-slate-200">
                    {e}
                  </span>
                ))}
              </div>
            )}
          </div>
        );
      })}

      <p className="text-xs text-slate-500">
        The Intelligence Engine roster above is returned live by the engine. IGS (Institutional Gold
        Signal) is a shadow/research engine and is intentionally excluded from the live pipeline until
        its validation gate is promoted — see the architecture design doc.
      </p>
    </div>
  );
}
