"use client";

import { useCallback, useEffect, useState } from "react";
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

function freshness(seconds: number): { label: string; cls: string } {
  if (seconds < 30) return { label: "live", cls: "text-emerald-500" };
  if (seconds < 180) return { label: `${seconds}s ago`, cls: "text-amber-500" };
  const mins = Math.round(seconds / 60);
  return { label: `${mins}m ago`, cls: "text-red-500" };
}

export default function ConnectivityCard() {
  const [snap, setSnap] = useState<ConnectivitySnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);

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

  const critical = snap.openAlerts.filter((a) => a.severity === "CRITICAL");
  const warnings = snap.openAlerts.filter((a) => a.severity === "WARNING");

  return (
    <div className="rounded-lg border border-pat-border bg-pat-card p-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-pat-text-primary">MT Client Connectivity</h3>
        <span
          className={`inline-flex items-center gap-1.5 text-xs font-medium ${
            snap.healthy ? "text-emerald-500" : critical.length > 0 ? "text-red-500" : "text-amber-500"
          }`}
        >
          <span
            className={`h-2 w-2 rounded-full ${
              snap.healthy ? "bg-emerald-500" : critical.length > 0 ? "bg-red-500" : "bg-amber-500"
            }`}
          />
          {snap.healthy ? "All clients connected" : critical.length > 0 ? "Signal flow at risk" : "Attention needed"}
        </span>
      </div>

      {(critical.length > 0 || warnings.length > 0) && (
        <ul className="mt-3 space-y-1.5">
          {[...critical, ...warnings].map((a) => (
            <li key={a.alertKey} className="text-xs text-pat-text-secondary">
              <span className={a.severity === "CRITICAL" ? "text-red-500 font-semibold" : "text-amber-500 font-semibold"}>
                [{a.severity}]
              </span>{" "}
              {a.message}
            </li>
          ))}
        </ul>
      )}

      <div className="mt-3 overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="text-left text-pat-text-secondary">
              <th className="py-1 pr-3 font-medium">Device</th>
              <th className="py-1 pr-3 font-medium">Last edge-poll</th>
            </tr>
          </thead>
          <tbody>
            {snap.devices.map((d) => {
              const f = freshness(Number(d.secondsSincePoll ?? 0));
              return (
                <tr key={d.deviceId} className="border-t border-pat-border/50">
                  <td className="py-1 pr-3 text-pat-text-primary">{d.deviceName}</td>
                  <td className={`py-1 pr-3 ${f.cls}`}>{f.label}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <p className="mt-2 text-[11px] text-pat-text-secondary">
        Checked {new Date(snap.checkedAt).toLocaleTimeString()} · auto-refresh 30s · alerts also pushed to ntfy
      </p>
    </div>
  );
}