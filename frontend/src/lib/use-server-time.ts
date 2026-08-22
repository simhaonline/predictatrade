/**
 * useServerTime — Server-authoritative time synchronization hook.
 *
 * The Go realtime engine returns `server_time` in RFC3339 format (UTC) on
 * every API response. This hook fetches the server time periodically and
 * calculates the clock drift between the browser and the server.
 *
 * This is critical for trading systems where a 22-minute clock skew between
 * the MT5 terminal and the dashboard can cause incorrect signal timing,
 * stale data misjudgment, and wrong session/news gate evaluation.
 *
 * SOW: "Internal time truth is UTC" — the frontend must display
 * server-authoritative UTC time, not browser local time.
 */

import { useEffect, useState, useCallback } from "react";
import { customInstance } from "@/lib/axios-instance";

interface ServerTimeState {
  /** Server UTC time in milliseconds since epoch */
  serverTimeMs: number;
  /** Clock drift in milliseconds (server - browser). Positive = browser is behind. */
  driftMs: number;
  /** Whether the drift exceeds the warning threshold (30 seconds) */
  driftWarning: boolean;
  /** Whether the drift exceeds the critical threshold (2 minutes) */
  driftCritical: boolean;
  /** ISO string of the last server time sync */
  lastSync: string;
}

const DRIFT_WARNING_MS = 30_000; // 30 seconds
const DRIFT_CRITICAL_MS = 120_000; // 2 minutes
const SYNC_INTERVAL_MS = 30_000; // 30 seconds

export function useServerTime() {
  const [state, setState] = useState<ServerTimeState>(() => ({
      serverTimeMs: Date.now(), driftMs: 0, driftWarning: false, driftCritical: false, lastSync: "",
    }));

  const syncTime = useCallback(async () => {
    try {
      const browserBefore = Date.now();
      const resp = await customInstance.get("/market/snapshot", {
        timeout: 5000,
      });
      const browserAfter = Date.now();
      const serverTimeStr = resp.data?.server_time || resp.data?.timestamp;
      if (serverTimeStr) {
        const serverMs = new Date(serverTimeStr).getTime();
        if (!isNaN(serverMs)) {
          // Account for network latency: server time corresponds to the
          // midpoint between request send and response receive
          const browserMid = (browserBefore + browserAfter) / 2;
          const drift = serverMs - browserMid;
          setState({
            serverTimeMs: serverMs,
            driftMs: drift,
            driftWarning: Math.abs(drift) > DRIFT_WARNING_MS,
            driftCritical: Math.abs(drift) > DRIFT_CRITICAL_MS,
            lastSync: new Date().toISOString(),
          });
        }
      }
    } catch {
      // Server unreachable — keep last known state
    }
  }, []);

  useEffect(() => {
    const initialSync = window.setTimeout(() => { void syncTime(); }, 0);
    const interval = setInterval(() => { void syncTime(); }, SYNC_INTERVAL_MS);
    return () => { window.clearTimeout(initialSync); clearInterval(interval); };
  }, [syncTime]);

  return state;
}

/**
 * getServerTimeMs — Returns the current server time in ms, accounting for drift.
 * Use this instead of Date.now() when displaying server-authoritative time.
 */
export function getServerTimeMs(driftMs: number): number {
  return Date.now() + driftMs;
}

/**
 * formatServerTime — Format server time as HH:mm:ss UTC.
 */
export function formatServerTime(driftMs: number): string {
  const serverMs = Date.now() + driftMs;
  return new Date(serverMs).toLocaleTimeString("en-GB", {
    timeZone: "UTC",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }) + " UTC";
}

/**
 * formatDrift — Human-readable drift label.
 */
export function formatDrift(driftMs: number): string {
  const abs = Math.abs(driftMs);
  if (abs < 1000) return "±0s";
  if (abs < 60_000) return `±${(abs / 1000).toFixed(0)}s`;
  return `±${(abs / 60_000).toFixed(1)}min`;
}
