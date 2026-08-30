"use client";
// MarketStatusBanner — shared weekend/closed-market banner (check.md #1):
// shows the last closing price with a "market closed" notice + live countdown
// to the next FX re-open (Sun 22:00 UTC). Rendered on admin + user consoles.
import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { fetchAgentsStatus } from "@/lib/admin-api";

interface AgentsStatus {
  market_closed?: boolean;
  next_market_open_utc?: string;
  data_health?: string;
  last_snapshot_at?: string;
}

function useCountdown(target?: string) {
  const [, force] = useState(0);
  useEffect(() => {
    const t = setInterval(() => force((x) => x + 1), 1000);
    return () => clearInterval(t);
  }, []);
  if (!target) return null;
  const ms = new Date(target).getTime() - Date.now();
  if (ms <= 0) return "opening…";
  const s = Math.floor(ms / 1000);
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60), sec = s % 60;
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m ${sec}s`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
}

export function MarketStatusBanner() {
  const [, force] = useState(0);
  useEffect(() => { const t = setInterval(() => force((x) => x + 1), 1000); return () => clearInterval(t); }, []);
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
            Re-opens in <b className="text-pat-text-primary">{d.next_market_open_utc ? fmt(d.next_market_open_utc) : "…"}</b>{" "}
            <span className="opacity-70">({new Date(d.next_market_open_utc).toISOString()})</span>
          </div>
        )}
      </div>
    </div>
  );
}

function fmt(target: string): string {
  const ms = new Date(target).getTime() - Date.now();
  if (ms <= 0) return "now";
  const s = Math.floor(ms / 1000);
  const dd = Math.floor(s / 86400), hh = Math.floor((s % 86400) / 3600), mm = Math.floor((s % 3600) / 60);
  return `${dd > 0 ? dd + "d " : ""}${hh}h ${mm}m`;
}