"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { getAccessToken } from "@/lib/auth";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "/api/v1";

interface ConnectivityAlert {
  alertKey: string;
  severity: string;
  scope: string;
  message: string;
  occurrences: number;
  firstSeenAt: string;
  lastSeenAt: string;
}

interface ConnectivityDevice {
  deviceId: string;
  deviceName: string;
  osName?: string;
  role?: string;
  email?: string;
  lastEquity?: number;
  lastSeenAt: string;
  secondsSincePoll: number;
}

interface ConnectivitySnapshot {
  healthy: boolean;
  openAlerts: ConnectivityAlert[];
  devices: ConnectivityDevice[];
  checkedAt: string;
}

const POLL_MS = 30_000;

// Live detection mirrors the platform standard (see memory/live-detection):
// GREATEST(device.last_seen, edge last_poll) < 5 min = ONLINE, otherwise
// OFFLINE. The terminal is a binary state; "last seen Xm/h ago" carries the
// freshness detail (a 3-state LIVE/RECENT/STALE ladder read like the terminal
// itself was in an unknown half-state and was removed on operator request —
// an offline terminal is simply OFFLINE, with its last-seen time shown).
function freshness(seconds: number): { label: string; cls: string; state: "ONLINE" | "OFFLINE" } {
  if (seconds < 300) {
    return { label: seconds < 5 ? "now" : `${seconds}s ago`, cls: "text-emerald-500", state: "ONLINE" };
  }
  if (seconds < 3600) {
    const mins = Math.max(1, Math.round(seconds / 60));
    return { label: `${mins}m ago`, cls: "text-slate-400", state: "OFFLINE" };
  }
  const hours = Math.round(seconds / 3600);
  return { label: `${hours}h ago`, cls: "text-slate-400", state: "OFFLINE" };
}

export default function MtClientsPage() {
  const [snap, setSnap] = useState<ConnectivitySnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<"ALL" | "LIVE" | "RECENT" | "STALE">("ALL");
  const [roleFilter, setRoleFilter] = useState<"ALL" | "exec" | "data">("ALL");

  const load = useCallback(async () => {
    try {
      const token = getAccessToken();
      const res = await fetch(`${API_BASE}/monitoring/connectivity`, {
        credentials: "include",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setSnap(await res.json());
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), POLL_MS);
    return () => clearInterval(t);
  }, [load]);

  const { live, recent, stale, critical, warnings } = useMemo(() => {
    if (!snap) return { live: 0, recent: 0, stale: 0, critical: [], warnings: [] };
    const devices = snap.devices ?? [];
    const states = devices.map((d) => freshness(Number(d.secondsSincePoll ?? 0)).state);
    return {
      live: states.filter((s) => s === "LIVE").length,
      recent: states.filter((s) => s === "RECENT").length,
      stale: states.filter((s) => s === "STALE").length,
      critical: (snap.openAlerts ?? []).filter((a) => a.severity === "CRITICAL"),
      warnings: (snap.openAlerts ?? []).filter((a) => a.severity === "WARNING"),
    };
  }, [snap]);

  const devices = useMemo(() => {
    if (!snap) return [];
    return (snap.devices ?? [])
      .filter((d) => (roleFilter === "ALL" ? true : (d.role ?? "exec") === roleFilter))
      .filter((d) => {
        if (filter === "ALL") return true;
        return freshness(Number(d.secondsSincePoll ?? 0)).state === filter;
      });
  }, [snap, filter, roleFilter]);

  if (error) {
    return (
      <div className="rounded-lg border border-pat-border bg-pat-card p-4 text-sm text-pat-text-secondary">
        Connectivity monitor unavailable ({error})
      </div>
    );
  }
  if (!snap) {
    return (
      <div className="rounded-lg border border-pat-border bg-pat-card p-4 text-sm text-pat-text-secondary">
        Loading connectivity…
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">MT Client Connectivity</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          MT4/MT5 client terminals polling the signal edge (edge-poll liveness).
          LIVE = polled within 5 minutes; alerts also push to ntfy.
        </p>
      </div>

      {/* Summary tiles */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">LIVE (&lt;5 min)</div>
          <div className="text-2xl font-bold text-emerald-500">{live}</div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">RECENT (5-60 min)</div>
          <div className="text-2xl font-bold text-amber-500">{recent}</div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">STALE (&gt;1 h)</div>
          <div className="text-2xl font-bold text-red-500">{stale}</div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">Open alerts</div>
          <div className={`text-2xl font-bold ${critical.length > 0 ? "text-red-500" : warnings.length > 0 ? "text-amber-500" : "text-emerald-500"}`}>
            {critical.length + warnings.length}
          </div>
        </div>
      </div>

      {(critical.length > 0 || warnings.length > 0) && (
        <ul className="space-y-1.5 rounded-lg border border-pat-border bg-pat-card p-4">
          {[...critical, ...warnings].map((a) => (
            <li key={a.alertKey} className="text-xs text-pat-text-secondary">
              <span className={`${a.severity === "CRITICAL" ? "text-red-500" : "text-amber-500"} font-semibold`}>
                [{a.severity}]
              </span>{" "}
              {a.message}
            </li>
          ))}
        </ul>
      )}

      {/* Filters */}
      <div className="flex flex-wrap gap-2">
        {(["ALL", "LIVE", "RECENT", "STALE"] as const).map((f) => (
          <button key={f} onClick={() => setFilter(f)}
            className={`text-xs px-3 py-1.5 rounded transition-colors ${filter === f ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {f}
          </button>
        ))}
        <span className="mx-2 w-px self-stretch bg-pat-border" />
        {(["ALL", "exec", "data"] as const).map((r) => (
          <button key={r} onClick={() => setRoleFilter(r)}
            className={`text-xs px-3 py-1.5 rounded transition-colors ${roleFilter === r ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {r === "ALL" ? "All roles" : r === "exec" ? "Exec (trading)" : "Data (masters)"}
          </button>
        ))}
        <span className="text-xs text-pat-text-muted ml-auto self-center">{devices.length} devices</span>
      </div>

      {/* Device table */}
      <div className="overflow-x-auto border border-pat-border rounded-lg">
        <table className="w-full text-sm text-left">
          <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
            <tr>
              <th className="px-3 py-3 font-medium">Terminal</th>
              <th className="px-3 py-3 font-medium">Platform</th>
              <th className="px-3 py-3 font-medium">Role</th>
              <th className="px-3 py-3 font-medium">Account</th>
              <th className="px-3 py-3 font-medium">Equity</th>
              <th className="px-3 py-3 font-medium">Last poll</th>
              <th className="px-3 py-3 font-medium">State</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-pat-border">
            {devices.length === 0 ? (
              <tr><td colSpan={7} className="px-3 py-8 text-center text-pat-text-muted text-sm">No devices match the current filters</td></tr>
            ) : (
              devices.map((d) => {
                const f = freshness(Number(d.secondsSincePoll ?? 0));
                const brokerShort = (d.deviceName || "Unknown broker").replace(/\s*\(.*?\)\s*$/, "");
                return (
                  <tr key={d.deviceId} className="hover:bg-pat-table-hover transition-colors">
                    <td className="px-3 py-3 text-xs text-pat-text-primary">{brokerShort}</td>
                    <td className="px-3 py-3 text-xs text-pat-text-secondary">{d.osName || "—"}</td>
                    <td className="px-3 py-3 text-xs text-pat-text-secondary">{d.role === "data" ? "Data (master)" : "Exec"}</td>
                    <td className="px-3 py-3 text-xs text-pat-text-secondary">{d.email || "—"}</td>
                    <td className="px-3 py-3 text-xs text-pat-text-primary tabular-nums">{(d.lastEquity ?? 0) > 0 ? `$${Number(d.lastEquity).toFixed(0)}` : "—"}</td>
                    <td className={`px-3 py-3 text-xs tabular-nums ${f.cls}`}>
                      {Number(d.secondsSincePoll ?? 0) < 60 ? `${d.secondsSincePoll}s ago` : f.label}
                    </td>
                    <td className="px-3 py-3">
                      <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${
                        f.state === "LIVE" ? "bg-emerald-500/10 text-emerald-500 border-emerald-500/30"
                        : f.state === "RECENT" ? "bg-amber-500/10 text-amber-500 border-amber-500/30"
                        : "bg-red-500/10 text-red-500 border-red-500/30"
                      }`}>{f.state}</span>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      <p className="text-[11px] text-pat-text-secondary">
        Checked {new Date(snap.checkedAt).toLocaleTimeString()} · auto-refresh 30s · alerts also pushed to ntfy
      </p>
    </div>
  );
}