"use client";
import { useQuery } from "@tanstack/react-query";
import { fetchEnginesStatus, type EngineSnapshot } from "@/lib/engines-api";
import { formatDistanceToNowStrict, parseISO } from "date-fns";

const ENGINE_LABELS: Record<string, string> = {
  STANDARD_SCALPING: "Standard Scalping",
  ULTRA_SCALPING: "Ultra Scalping",
  STANDARD_SWING: "Standard Swing",
  TREND_SWING: "Trend Swing",
  MARNIE_FIB: "Marnie Fib",
};

const HEALTH_STYLES: Record<string, string> = {
  LIVE: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  WAITING: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  STALE: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  ERROR: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
};

function age(iso: string | undefined, now: Date): string {
  if (!iso) return "never";
  const t = parseISO(iso);
  if (Number.isNaN(t.getTime())) return "never";
  return `${formatDistanceToNowStrict(t, { addSuffix: false })} ago`;
}

export function EngineCard({ e, serverTime }: { e: EngineSnapshot; serverTime: Date }) {
  const label = ENGINE_LABELS[e.engine] ?? e.engine;
  const decision = e.current_decision || "NO-TRADE";
  const isCandidate = decision === "BUY" || decision === "SELL" || decision === "BUY_CANDIDATE" || decision === "SELL_CANDIDATE";
  return (
    <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-3 shadow-sm">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-medium text-pat-text-primary">{label}</span>
        <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${HEALTH_STYLES[e.health] ?? HEALTH_STYLES.WAITING}`}>
          {e.health}
        </span>
      </div>
      <dl className="space-y-1 text-[11px]">
        <div className="flex justify-between"><dt className="text-pat-text-muted">Timeframes</dt><dd className="text-pat-text-primary">{(e.primary_timeframes ?? []).join(", ") || "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Last evaluation</dt><dd className="text-pat-text-primary">{age(e.last_evaluation, serverTime)}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Last signal</dt><dd className="text-pat-text-primary">{e.last_signal_reference || "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Decision</dt><dd className={isCandidate ? "text-pat-warning font-medium" : "text-pat-text-secondary"}>{isCandidate ? decision : "NO-TRADE"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Score</dt><dd className="text-pat-text-primary tabular-nums">{e.current_score > 0 ? e.current_score.toFixed(1) : "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Probability</dt><dd className="text-pat-text-primary tabular-nums">{e.has_calibrated_probability ? `${(e.calibrated_probability * 100).toFixed(1)}%` : "not calibrated"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Data quality</dt><dd className={e.data_quality === "GOOD" ? "text-pat-success" : "text-pat-warning"}>{e.data_quality || "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Evals / No-trade</dt><dd className="text-pat-text-primary tabular-nums">{e.evaluation_count} / {e.no_trade_count}</dd></div>
      </dl>
    </div>
  );
}

/**
 * Four strategy engine cards fed by the Go engine's /engines/status endpoint.
 * Shows truthful liveness only — NO-TRADE is displayed as NO-TRADE, never
 * dressed up as a signal (prompt.md Sections 44, 111).
 */
export default function AdminEngineCards() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["engines-status"],
    queryFn: fetchEnginesStatus,
    refetchInterval: 10000,
  });

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="bg-pat-card-bg border border-pat-card-border rounded-lg p-3 animate-pulse h-40" />
        ))}
      </div>
    );
  }
  if (isError || !data) {
    return (
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 text-sm text-pat-danger">
        DATA UNAVAILABLE — engine status endpoint unreachable.
      </div>
    );
  }

  const serverTime = data.server_time ? parseISO(data.server_time) : new Date();
  const engines = data.engines ?? [];
  const stale = engines.filter((e) => e.health === "STALE").map((e) => ENGINE_LABELS[e.engine] ?? e.engine);

  return (
    <div className="space-y-2">
      {stale.length > 0 && (
        <div className="rounded-md border border-pat-warning/30 bg-pat-warning/10 px-3 py-2 text-xs text-pat-warning flex items-center gap-2">
          STALE DATA — SIGNAL GENERATION PAUSED for: {stale.join(", ")}
        </div>
      )}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
        {engines.map((e) => (
          <EngineCard key={e.engine} e={e} serverTime={serverTime} />
        ))}
      </div>
    </div>
  );
}
