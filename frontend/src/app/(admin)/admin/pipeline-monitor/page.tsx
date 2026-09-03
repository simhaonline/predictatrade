"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";

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

/** Dot colour per lifecycle status — PAT semantic tokens only. */
const STATUS_DOTS: Record<string, string> = {
  healthy: "bg-pat-success",
  live: "bg-pat-success",
  connected: "bg-pat-success",
  idle: "bg-pat-text-muted",
  db_error: "bg-pat-danger",
  degraded: "bg-pat-warning",
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
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Pipeline Monitor</h1>
          <p className="text-sm text-pat-text-secondary mt-1">
            Live Signal → Risk → Execution → Review pipeline (real-time, sourced from the Go engine).
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
          Loading pipeline state…
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-pat-danger/40 bg-pat-danger/10 p-4 text-sm text-pat-danger">
          Failed to load pipeline monitor: {(error as Error)?.message || "unknown error"}
          <button onClick={() => refetch()} className="ml-3 underline">
            retry
          </button>
        </div>
      )}

      {data?.pipeline?.map((stage) => (
        <div key={stage.stage} className="rounded-lg border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <span
                className={`h-2 w-2 rounded-full ${
                  STATUS_DOTS[stage.status] || "bg-pat-text-muted"
                }`}
              />
              <span className="text-base font-semibold text-pat-text-primary">{stage.stage}</span>
              <StatusBadge status={stage.status} size="sm" />
            </div>
            <span className="text-sm text-pat-text-secondary">{stage.name}</span>
          </div>

          <p className="mt-2 text-sm text-pat-text-secondary">{stage.detail || stage.note}</p>

          <div className="mt-3 flex flex-wrap gap-x-8 gap-y-2 text-sm text-pat-text-secondary">
            {typeof stage.count_5m === "number" && (
              <span>
                Signals (5m): <span className="font-semibold text-pat-text-primary">{stage.count_5m}</span>
              </span>
            )}
            {stage.last_at && (
              <span>
                Last at: <span className="font-semibold text-pat-text-primary">{stage.last_at}</span>
              </span>
            )}
            {typeof stage.vetoed_5m === "number" && (
              <span>
                Vetoed (5m):{" "}
                <span className="font-semibold text-pat-text-primary">{stage.vetoed_5m}</span>
              </span>
            )}
            {typeof stage.backfill === "number" && (
              <span>
                Backfill: <span className="font-semibold text-pat-text-primary">{stage.backfill}%</span>
              </span>
            )}
          </div>

          {stage.engines && stage.engines.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-2">
              {stage.engines.map((e) => (
                <span
                  key={e}
                  className="rounded-md bg-pat-bg-surface-secondary border border-pat-border px-2 py-0.5 text-xs text-pat-text-secondary"
                >
                  {e}
                </span>
              ))}
            </div>
          )}
        </div>
      ))}

      <p className="text-xs text-pat-text-muted">
        The Intelligence Engine roster above is returned live by the engine. IGS (Institutional Gold
        Signal) is a shadow/research engine and is intentionally excluded from the live pipeline until
        its validation gate is promoted — see the architecture design doc.
      </p>
    </div>
  );
}