"use client";
// MarketStatusBanner — shared weekend/closed-market banner (check.md #1):
// shows the last closing price with a "market closed" notice + live countdown
// to the next FX re-open (Sun 22:00 UTC). Rendered on admin + user consoles.
import { useQuery } from "@tanstack/react-query";
import { useState, useSyncExternalStore } from "react";
import { fetchAgentsStatus } from "@/lib/admin-api";

interface AgentsStatus {
  market_closed?: boolean;
  next_market_open_utc?: string;
  data_health?: string;
  last_snapshot_at?: string;
}

// ─── useNow(): external store holding a 1s-ticking timestamp ───
// React-19 sanctioned way to read a moving clock during render without
// "impure function during render" or setState-in-effect: the clock is an
// external store; useSyncExternalStore subscribes and re-renders per tick.
const clockListeners = new Set<() => void>();
let clockNow = 0;
let clockTimer: ReturnType<typeof setInterval> | null = null;
function startClock() {
  clockNow = Date.now();
  clockTimer = setInterval(() => {
    clockNow = Date.now();
    for (const l of clockListeners) l();
  }, 1000);
}
function subscribeClock(l: () => void) {
  clockListeners.add(l);
  if (clockTimer === null) startClock();
  return () => {
    clockListeners.delete(l);
    if (clockListeners.size === 0 && clockTimer !== null) {
      clearInterval(clockTimer);
      clockTimer = null;
    }
  };
}
function useNow(): number {
  return useSyncExternalStore(
    subscribeClock,
    () => clockNow,
    () => 0, // server snapshot: neutral (no countdown on SSR)
  );
}

function fmtCountdown(target: string, now: number): string {
  const ms = new Date(target).getTime() - now;
  if (ms <= 0) return "opening…";
  const s = Math.floor(ms / 1000);
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60), sec = s % 60;
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m ${sec}s`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
}

export function MarketStatusBanner() {
  const now = useNow();
  const q = useQuery({
    queryKey: ["agents-status-banner"],
    queryFn: async () => (await fetchAgentsStatus()) as AgentsStatus,
    refetchInterval: 30_000,
  });
  const d = q.data;
  if (!d?.market_closed) return null;
  return (
    <div role="status" data-testid="market-closed-banner"
      className="rounded-lg border border-pat-warning/40 bg-pat-warning/5 px-4 py-3 text-sm">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <span className="text-pat-text-primary">🕒 Market closed — weekend.</span>{" "}
          <span className="text-pat-text-secondary">
            Showing last closing prices; no signals are generated until the next market re-opens.
          </span>
        </div>
        {d.next_market_open_utc && (
          <div className="rounded bg-pat-bg-surface px-3 py-1.5 text-xs text-pat-text-secondary">
            Re-opens in <b className="text-pat-text-primary">{now > 0 ? fmtCountdown(d.next_market_open_utc, now) : "…"}</b>{" "}
            <span className="opacity-70">({new Date(d.next_market_open_utc).toISOString()})</span>
          </div>
        )}
      </div>
    </div>
  );
}
