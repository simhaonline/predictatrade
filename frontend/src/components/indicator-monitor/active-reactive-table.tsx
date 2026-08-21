"use client";
import { useState } from "react";
import type { IndicatorLiveness } from "@/lib/use-indicator-liveness";
import { getActiveColor } from "@/lib/use-indicator-liveness";

interface ActiveReactiveTableProps {
  liveness: IndicatorLiveness[];
}

export function ActiveReactiveTable({ liveness }: ActiveReactiveTableProps) {
  const [selected, setSelected] = useState<string | null>(null);
  const groups = [...new Set(liveness.map((i) => i.group))];

  const activeCount = liveness.filter((i) => i.activeStatus === "active").length;
  const armedCount = liveness.filter((i) => i.activeStatus === "armed").length;
  const reactiveCount = liveness.filter((i) => i.activeStatus === "reactive").length;
  const inactiveCount = liveness.filter((i) => i.activeStatus === "inactive").length;

  return (
    <div className="space-y-4">
      {/* Summary bar */}
      <div className="flex gap-4">
        <StatusBadge label="Active" count={activeCount} color="bg-pat-success" />
        <StatusBadge label="Armed" count={armedCount} color="bg-pat-warning" />
        <StatusBadge label="Reactive" count={reactiveCount} color="bg-pat-info" />
        <StatusBadge label="Inactive" count={inactiveCount} color="bg-pat-text-muted" />
      </div>

      {/* Table */}
      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Active / Reactive / Armed Status</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-pat-text-muted border-b border-pat-border">
                <th className="text-left py-2 px-3">Indicator</th>
                <th className="text-center py-2 px-3">Status</th>
                <th className="text-left py-2 px-3">Current Value</th>
                <th className="text-left py-2 px-3">Used By</th>
              </tr>
            </thead>
            <tbody>
              {groups.map((group) => (
                <ActiveGroup
                  key={group}
                  group={group}
                  liveness={liveness.filter((i) => i.group === group)}
                  onSelect={setSelected}
                />
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {selected && (
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
          <h3 className="text-sm font-semibold text-pat-text-primary mb-2">{selected}</h3>
          <div className="text-xs text-pat-text-muted">
            Recent trade history for this indicator is not available from the current data source.
            Trade-level indicator participation requires backend evidence tracking.
          </div>
        </div>
      )}
    </div>
  );
}

function ActiveGroup({ group, liveness, onSelect }: {
  group: string;
  liveness: IndicatorLiveness[];
  onSelect: (key: string) => void;
}) {
  return (
    <>
      <tr className="bg-pat-bg-surface-secondary/30">
        <td colSpan={4} className="text-xs font-semibold text-pat-text-secondary py-2 px-3">{group}</td>
      </tr>
      {liveness.map((ind) => (
        <tr
          key={ind.key}
          onClick={() => onSelect(ind.key)}
          className="cursor-pointer hover:bg-pat-bg-surface-secondary/50 border-b border-pat-border/50"
        >
          <td className="py-2 px-3 text-pat-text-primary">{ind.label}</td>
          <td className="py-2 px-3 text-center">
            <span className={`inline-flex items-center gap-1 text-xs ${getActiveColor(ind.activeStatus).replace("bg-", "text-")}`}>
              <span className={`inline-block w-2.5 h-2.5 rounded-full ${getActiveColor(ind.activeStatus)}`} />
              {ind.activeStatus}
            </span>
          </td>
          <td className="py-2 px-3 font-mono text-xs text-pat-text-secondary">
            {ind.currentValue === null ? "—" : typeof ind.currentValue === "number" ? ind.currentValue.toFixed(4) : String(ind.currentValue)}
          </td>
          <td className="py-2 px-3 text-xs text-pat-text-muted">
            {ind.activeStatus === "armed" ? "All strategies (monitoring)" : "—"}
          </td>
        </tr>
      ))}
    </>
  );
}

function StatusBadge({ label, count, color }: { label: string; count: number; color: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className={`inline-block w-3 h-3 rounded-full ${color}`} />
      <span className="text-xs text-pat-text-secondary">{label}: {count}</span>
    </div>
  );
}
