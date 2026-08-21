"use client";
import { useState } from "react";
import type { IndicatorLiveness } from "@/lib/use-indicator-liveness";
import { getLivenessColor } from "@/lib/use-indicator-liveness";

interface LivenessMatrixProps {
  liveness: IndicatorLiveness[];
  history: Map<string, { time: number; value: number }[]>;
}

function formatTimestamp(ts: number | null): string {
  if (!ts) return "—";
  return new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function formatValue(val: number | string | boolean | null): string {
  if (val === null || val === undefined) return "—";
  if (typeof val === "boolean") return val ? "Yes" : "No";
  if (typeof val === "number") {
    if (val === 0) return "0";
    if (Math.abs(val) < 0.01) return val.toExponential(2);
    return val.toFixed(4);
  }
  return String(val);
}

function formatInterval(ms: number): string {
  if (ms <= 1000) return "≤1s";
  if (ms <= 60000) return `${ms / 1000}s`;
  return `${Math.round(ms / 60000)}m`;
}

const TYPE_ICONS: Record<string, string> = {
  tick: "⚡", bar: "📊", session: "🕐",
};

export function LivenessMatrix({ liveness, history }: LivenessMatrixProps) {
  const [selected, setSelected] = useState<string | null>(null);
  const groups = [...new Set(liveness.map((i) => i.group))];
  const liveCount = liveness.filter(i => i.status === "live").length;
  const lateCount = liveness.filter(i => i.status === "late").length;
  const staleCount = liveness.filter(i => i.status === "stale").length;

  return (
    <div className="space-y-4">
      {/* Status legend bar */}
      <div className="flex items-center gap-4 rounded-lg border border-pat-border bg-pat-bg-surface px-4 py-2.5">
        <span className="text-xs font-medium text-pat-text-secondary">Status:</span>
        <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-success" /><span className="text-xs text-pat-text-secondary">Live ({liveCount})</span></div>
        <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-warning" /><span className="text-xs text-pat-text-secondary">Late ({lateCount})</span></div>
        <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-danger" /><span className="text-xs text-pat-text-secondary">Stale ({staleCount})</span></div>
        <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-text-muted" /><span className="text-xs text-pat-text-secondary">Off</span></div>
      </div>

      {/* Liveness cards grouped by category */}
      <div className="space-y-3">
        {groups.map((group) => {
          const groupInds = liveness.filter((i) => i.group === group);
          const groupLive = groupInds.filter(i => i.status === "live").length;
          return (
            <div key={group} className="rounded-xl border border-pat-border bg-pat-bg-surface overflow-hidden">
              {/* Group header */}
              <div className="flex items-center justify-between px-4 py-2.5 bg-pat-bg-surface-secondary/30 border-b border-pat-border">
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-semibold text-pat-text-primary">{group}</h3>
                  <span className="text-xs text-pat-text-muted">{groupLive}/{groupInds.length} live</span>
                </div>
                <span className="text-xs text-pat-text-muted">{groupInds.length} indicators</span>
              </div>
              {/* Indicator cards grid */}
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-2 p-3">
                {groupInds.map((ind) => {
                  const isSelected = selected === ind.key;
                  const histLen = history.get(ind.key)?.length ?? 0;
                  return (
                    <button
                      key={ind.key}
                      onClick={() => setSelected(isSelected ? null : ind.key)}
                      className={`rounded-lg border p-2.5 text-left transition-all ${
                        isSelected
                          ? "border-pat-success/40 bg-pat-success/5"
                          : "border-pat-border/60 bg-pat-bg-surface-secondary/30 hover:border-pat-border hover:bg-pat-bg-surface-secondary/50"
                      }`}
                    >
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-xs font-medium text-pat-text-primary truncate">{ind.label}</span>
                        <span className={`w-2 h-2 rounded-full shrink-0 ${getLivenessColor(ind.status)}`} />
                      </div>
                      <div className="font-mono text-sm text-pat-text-secondary tabular-nums">{formatValue(ind.currentValue)}</div>
                      <div className="flex items-center justify-between mt-1">
                        <span className="text-[10px] text-pat-text-muted">{formatTimestamp(ind.lastUpdated)}</span>
                        <span className="text-[10px] text-pat-text-muted flex items-center gap-0.5">
                          {TYPE_ICONS[ind.updateType]} {formatInterval(ind.expectedIntervalMs)}
                        </span>
                      </div>
                      {histLen > 0 && (
                        <div className="mt-1.5 flex items-center gap-1">
                          <span className="text-[10px] text-pat-success/70">{histLen} pts</span>
                          <div className="flex-1 h-0.5 rounded-full bg-pat-bg-surface-secondary overflow-hidden">
                            <div className="h-full bg-pat-success/40 rounded-full" style={{ width: `${Math.min(histLen / 200 * 100, 100)}%` }} />
                          </div>
                        </div>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>

      {/* Selected indicator detail */}
      {selected && (
        <div className="rounded-xl border border-pat-success/30 bg-pat-success/5 p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-pat-text-primary">
              {liveness.find(i => i.key === selected)?.label || selected}
            </h3>
            <button onClick={() => setSelected(null)} className="text-xs text-pat-text-muted hover:text-pat-text-primary">✕ Close</button>
          </div>
          <LivenessTimeline history={history.get(selected) || []} />
        </div>
      )}
    </div>
  );
}

function LivenessTimeline({ history }: { history: { time: number; value: number }[] }) {
  if (history.length === 0) {
    return (
      <div className="flex items-center justify-center py-6 text-xs text-pat-text-muted">
        No history available — indicator value is zero or not yet updating.
      </div>
    );
  }
  const latest = history[history.length - 1];
  const first = history[0];
  const change = latest.value - first.value;
  const changePct = first.value !== 0 ? (change / Math.abs(first.value)) * 100 : 0;
  
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-4 text-xs">
        <div><span className="text-pat-text-muted">Latest:</span> <span className="font-mono text-pat-text-primary">{latest.value.toFixed(4)}</span></div>
        <div><span className="text-pat-text-muted">Change:</span> <span className={`font-mono ${change >= 0 ? "text-pat-success" : "text-pat-danger"}`}>{change >= 0 ? "+" : ""}{change.toFixed(4)} ({changePct >= 0 ? "+" : ""}{changePct.toFixed(2)}%)</span></div>
        <div><span className="text-pat-text-muted">Samples:</span> <span className="font-mono text-pat-text-secondary">{history.length}</span></div>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-1.5">
        {history.slice(-24).reverse().map((point, i) => (
          <div key={i} className="flex items-center gap-1.5 text-[11px] rounded-md bg-pat-bg-surface-secondary/30 px-2 py-1">
            <span className="w-1.5 h-1.5 rounded-full bg-pat-success/60 shrink-0" />
            <span className="text-pat-text-muted tabular-nums">{new Date(point.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}</span>
            <span className="font-mono text-pat-text-secondary tabular-nums ml-auto">{point.value.toFixed(4)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
