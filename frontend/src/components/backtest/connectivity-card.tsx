"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
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

// Live detection mirrors the platform standard: last poll < 5 min.
function stateOf(seconds: number): "LIVE" | "RECENT" | "STALE" {
  if (seconds < 300) return "LIVE";
  if (seconds < 3600) return "RECENT";
  return "STALE";
}

/**
 * Compact MT Client Connectivity strip for the Real-Time Console.
 * Shows live/total counts + critical alert count and links to the full
 * fleet view at /admin/mt-clients (dedicated tab — the console stays clean).
 */
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

  const devices = snap.devices ?? [];
  const states = devices.map((d) => stateOf(Number(d.secondsSincePoll ?? 0)));
  const live = states.filter((s) => s === "LIVE").length;
  const recent = states.filter((s) => s === "RECENT").length;
  const stale = states.filter((s) => s === "STALE").length;
  const critical = (snap.openAlerts ?? []).filter((a) => a.severity === "CRITICAL");

  return (
    <Link
      href="/admin/mt-clients"
      className="block rounded-lg border border-pat-border bg-pat-card p-4 hover:border-pat-primary/40 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-pat-text-primary">MT Client Connectivity</h3>
          <p className="text-[11px] text-pat-text-muted mt-0.5">
            {live} live · {recent} recent · {stale} stale · {devices.length} total terminals
            {critical.length > 0 && (
              <span className="text-red-500 font-semibold"> · {critical.length} CRITICAL alert{critical.length > 1 ? "s" : ""}</span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={`inline-flex items-center gap-1.5 text-xs font-medium ${
              critical.length > 0 ? "text-red-500" : "text-emerald-500"
            }`}
          >
            <span className={`h-2 w-2 rounded-full ${critical.length > 0 ? "bg-red-500" : "bg-emerald-500"}`} />
            {critical.length > 0 ? "Signal flow at risk" : "Fleet healthy"}
          </span>
          <span className="text-[11px] text-pat-text-muted">View all →</span>
        </div>
      </div>
    </Link>
  );
}