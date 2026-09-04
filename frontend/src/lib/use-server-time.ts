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
 * The engine runs on the BROKER session timezone (collected live from the
 * Agents), not UTC — see broker_offset / time_mode in the snapshot API.
 * The frontend surfaces the authoritative broker-local time, not browser time.
 */

import { useEffect, useState, useCallback } from "react";
import { customInstance } from "@/lib/axios-instance";
import { format } from "date-fns";

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
  /** Active broker UTC offset in hours (engine runs on Broker TF, not UTC). 0 = UTC-aligned. */
  brokerOffset: number;
  /** Engine time-alignment mode: "BROKER_ALIGNED" or "UTC_ALIGNED". */
  brokerTimeMode: string;
}

const DRIFT_WARNING_MS = 30_000; // 30 seconds
const DRIFT_CRITICAL_MS = 120_000; // 2 minutes
const SYNC_INTERVAL_MS = 30_000; // 30 seconds

export function useServerTime() {
  const [state, setState] = useState<ServerTimeState>(() => ({
      serverTimeMs: Date.now(), driftMs: 0, driftWarning: false, driftCritical: false, lastSync: "",
      brokerOffset: 0, brokerTimeMode: "UTC_ALIGNED",
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
          const brokerOffset = Number(resp.data?.broker_offset) || 0;
          const brokerTimeMode = typeof resp.data?.time_mode === "string"
            ? resp.data.time_mode
            : (brokerOffset !== 0 ? "BROKER_ALIGNED" : "UTC_ALIGNED");
          setState({
            serverTimeMs: serverMs,
            driftMs: drift,
            driftWarning: Math.abs(drift) > DRIFT_WARNING_MS,
            driftCritical: Math.abs(drift) > DRIFT_CRITICAL_MS,
            lastSync: new Date().toISOString(),
            brokerOffset,
            brokerTimeMode,
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
 * Deprecated label: the engine now runs on Broker TF; prefer formatBrokerTime.
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
 * formatBrokerTime — Format the engine's authoritative time in the broker
 * session timezone. When brokerOffset is 0 it falls back to UTC. The label
 * reflects the actual alignment mode so the dashboard shows Broker TF, not UTC.
 */
export function formatBrokerTime(driftMs: number, brokerOffset: number): string {
  const serverMs = Date.now() + driftMs;
  const utc = new Date(serverMs);
  if (brokerOffset === 0) {
    return utc.toLocaleTimeString("en-GB", {
      timeZone: "UTC",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    }) + " UTC";
  }
  const label = `UTC${brokerOffset > 0 ? "+" : ""}${brokerOffset}`;
  return utc.toLocaleTimeString("en-GB", {
    timeZone: "UTC",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }) + ` (${label})`;
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

/**
 * formatBrokerInstant — Render a SERVER-issued timestamp (signal CreatedAt,
 * ExpiresAt, delivery sent_at …) on the broker session clock.
 *
 * All backend timestamps are UTC instants (RFC3339 with Z). The dashboard
 * MUST NOT render them with date-fns `format(new Date(s.CreatedAt))` — that
 * silently converts to the BROWSER's timezone, so a UTC 09:00 signal shows
 * 13:00 for a Dubai viewer while the EA's TimeCurrent() reads 12:00
 * (broker GMT+3) — three different clocks for the same event.
 *
 * Pass the brokerOffset from useServerTime() so the rendering follows the
 * same live Master-Node-reported offset the engine's session logic uses.
 * When brokerOffset is 0 (UTC-aligned mode) it falls back to UTC.
 */
export function formatBrokerTimestamp(
  iso: string | number | Date | null | undefined,
  brokerOffset: number,
  formatStr: string = "HH:mm:ss"
): string {
  if (iso === null || iso === undefined || iso === "") return "—";
  const d = iso instanceof Date ? iso : new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const shifted = brokerOffset !== 0 ? new Date(d.getTime() + brokerOffset * 3600_000) : d;
  return format(shifted, formatStr) + (brokerOffset !== 0 ? ` (B${brokerOffset > 0 ? "+" : ""}${brokerOffset})` : "");
}
