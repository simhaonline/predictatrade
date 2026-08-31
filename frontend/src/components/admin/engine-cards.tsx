"use client";
import { useQuery } from "@tanstack/react-query";
import { fetchEnginesStatus, type EngineSnapshot } from "@/lib/engines-api";
import { formatDistanceToNowStrict, parseISO } from "date-fns";

const ENGINE_LABELS: Record<string, string> = {
  STANDARD_SCALPING: "Standard Scalping",
  ULTRA_SCALPING: "Ultra Scalping",
  STANDARD_SWING: "Standard Swing",
  TREND_SWING: "Trend Swing",
  MARNIE_FIB: "EQFE",
  ATEN: "ATEN",
  ARCANIST: "Arcanist (IMLR)",
};

const HEALTH_STYLES: Record<string, string> = {
  LIVE: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  WAITING: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  STALE: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  ERROR: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
};

// Quality grade color mapping (prompt.md Section 12)
const GRADE_COLORS: Record<string, string> = {
  "A+": "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  A: "bg-pat-badge-info-bg text-pat-badge-info-text border-pat-badge-info-bg",
  B: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  REJECTED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  "NO-TRADE": "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg",
};

function age(iso: string | undefined, now: Date): string {
  if (!iso) return "never";
  const t = parseISO(iso);
  if (Number.isNaN(t.getTime())) return "never";
  return `${formatDistanceToNowStrict(t, { addSuffix: false })} ago`;
}

function topRejectionReasons(counts: Record<string, number> | undefined): string {
  if (!counts || Object.keys(counts).length === 0) return "—";
  return Object.entries(counts)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 2)
    .map(([k, v]) => `${k}(${v})`)
    .join(", ");
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
        <div className="flex justify-between"><dt className="text-pat-text-muted">Last eval</dt><dd className="text-pat-text-primary">{age(e.last_evaluation, serverTime)}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Decision</dt><dd className={isCandidate ? "text-pat-warning font-medium" : "text-pat-text-secondary"}>{isCandidate ? decision : "NO-TRADE"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Score</dt><dd className="text-pat-text-primary tabular-nums">{e.current_score > 0 ? e.current_score.toFixed(1) : "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Expectancy</dt><dd className={(e.expectancy_score ?? 0) > 50 ? "text-pat-success tabular-nums" : "text-pat-text-secondary tabular-nums"}>{e.expectancy_score != null ? e.expectancy_score.toFixed(1) : "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Quality</dt><dd className={`text-[10px] px-1 py-0.5 rounded-full border font-medium ${GRADE_COLORS[e.current_decision === 'NO-TRADE' ? 'NO-TRADE' : 'B']}`}>{isCandidate ? (e.expectancy_score != null && e.expectancy_score > 60 ? "A" : "B") : "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Candidates / Qualified</dt><dd className="text-pat-text-primary tabular-nums">{e.candidates_today ?? e.signal_count} / {e.qualified_today ?? "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Rejection rate</dt><dd className={(e.rejection_rate ?? 0) > 80 ? "text-pat-danger tabular-nums" : "text-pat-text-primary tabular-nums"}>{(e.rejection_rate ?? 0) > 0 ? `${((e.rejection_rate ?? 0) * 100).toFixed(0)}%` : "—"}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Top reasons</dt><dd className="text-[11px] text-pat-text-secondary">{topRejectionReasons(e.rejection_counts)}</dd></div>
        <div className="flex justify-between"><dt className="text-pat-text-muted">Config v</dt><dd className="text-pat-text-muted">{e.config_version || "1.15.0"}</dd></div>
      </dl>
    </div>
  );
}

/**
 * Strategy engine cards fed by the Go engine's /engines/status endpoint.
 * Shows truthful liveness with quality, expectancy, rejection diagnostics.
 * Includes MARNIE_FIB as the 5th engine (prompt.md Sections 19, 69).
 */
export default function AdminEngineCards() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["engines-status"],
    queryFn: fetchEnginesStatus,
    refetchInterval: 10000,
  });

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-3">
        {Array.from({ length: 5 }).map((_, i) => (
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
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-3">
        {engines.map((e) => (
          <EngineCard key={e.engine} e={e} serverTime={serverTime} />
        ))}
      </div>
    </div>
  );
}
