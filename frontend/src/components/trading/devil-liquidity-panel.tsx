"use client";

import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconDroplet, IconAlertTriangle } from "@tabler/icons-react";

export interface DevilMark {
  id: string;
  symbol: string;
  timeframe: string;
  direction: "BULLISH" | "BEARISH";
  mark_price: number;
  status: string;
  mark_quality_score: number;
  combined_score: number;
  priority_score: number;
  distance_atr: number;
  atr: number;
  volume_ratio: number;
  detected_at: string;
  first_touch_at?: string;
  first_sweep_at?: string;
  reversal_confirmed_at?: string;
  formation_session?: string;
  formation_regime?: string;
}

interface DevilResponse {
  enabled: boolean;
  count: number;
  marks: DevilMark[];
  stats: {
    candles_processed: number;
    marks_created: number;
    active_marks: number;
    last_candle_time: string;
    symbols_seen: string[];
  };
}

export function DevilLiquidityPanel({ title = "Devil Liquidity / Devil's Mark Engine" }: { title?: string }) {
  const { data, isLoading, error, refetch } = useQuery<DevilResponse>({
    queryKey: ["devil-liquidity-marks"],
    queryFn: async () => (await customInstance.get("/devil-liquidity/marks")).data,
    refetchInterval: 5000,
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary flex items-center gap-2">
            <IconDroplet size={20} className="text-pat-accent" /> {title}
          </h1>
          <p className="text-sm text-pat-text-secondary mt-1">
            Detects institutional-style displacement candles with a flat (wickless) edge — a potential
            <span className="font-semibold"> Devil&apos;s Mark</span>. Tracks the full lifecycle: approach → touch →
            sweep → reclaim → reversal confirmation. Runs in <strong>SHADOW</strong> mode (observation only, no live signal gating yet).
          </p>
        </div>
        <button
          onClick={() => refetch()}
          className="text-xs bg-pat-bg-surface-secondary px-3 py-1.5 rounded hover:bg-pat-bg-surface"
        >
          ↻ Refresh
        </button>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-xs">
        <span className="px-2 py-1 rounded bg-pat-bg-surface-secondary border border-pat-border">
          Engine: {data?.enabled ? <span className="text-pat-success">ENABLED</span> : <span className="text-pat-text-muted">OFFLINE</span>}
        </span>
        <span className="px-2 py-1 rounded bg-pat-bg-surface-secondary border border-pat-border">
          Active marks: {data?.count ?? 0}
        </span>
        <span className="px-2 py-1 rounded bg-pat-bg-surface-secondary border border-pat-border">
          Candles processed: {data?.stats?.candles_processed ?? 0}
        </span>
        <span className="px-2 py-1 rounded bg-pat-bg-surface-secondary border border-pat-border">
          Marks created: {data?.stats?.marks_created ?? 0}
        </span>
        <span className="px-2 py-1 rounded bg-pat-bg-surface-secondary border border-pat-border">
          Symbols: {(data?.stats?.symbols_seen ?? []).join(", ") || "—"}
        </span>
        {data?.stats?.last_candle_time && (
          <span className="px-2 py-1 rounded bg-pat-bg-surface-secondary border border-pat-border">
            Last candle: {new Date(data.stats.last_candle_time).toLocaleTimeString()}
          </span>
        )}
      </div>

      {isLoading ? (
        <div className="h-40 bg-pat-bg-surface-secondary/50 rounded animate-pulse" />
      ) : error ? (
        <div className="text-sm text-pat-danger flex items-center gap-2">
          <IconAlertTriangle size={16} /> Failed to load Devil Liquidity marks.
        </div>
      ) : !data || data.marks.length === 0 ? (
        <div className="text-sm text-pat-text-muted border border-pat-border rounded-lg p-6 space-y-2">
          <p>
            No active Devil&apos;s Marks right now. Marks are created only on completed candles that show a flat-edged
            displacement with sufficient body dominance, ATR expansion and volume. They expire if untested.
          </p>
          {data?.stats && (
            <p className="text-xs text-pat-text-secondary">
              Engine diagnostics — candles processed: {data.stats.candles_processed ?? 0} · marks created:{" "}
              {data.stats.marks_created ?? 0} · symbols seen: {data.stats.symbols_seen?.join(", ") || "—"}
              {data.stats.last_candle_time && (
                <>
                  {" "}· last candle: {new Date(data.stats.last_candle_time).toLocaleTimeString()}
                </>
              )}
            </p>
          )}
        </div>
      ) : (
        <div className="overflow-x-auto border border-pat-border rounded-lg">
          <table className="w-full text-sm">
            <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
              <tr>
                <th className="px-3 py-2 text-left">Symbol</th>
                <th className="px-3 py-2 text-left">TF</th>
                <th className="px-3 py-2 text-left">Dir</th>
                <th className="px-3 py-2 text-right">Mark Price</th>
                <th className="px-3 py-2 text-left">State</th>
                <th className="px-3 py-2 text-right">Quality</th>
                <th className="px-3 py-2 text-right">Combined</th>
                <th className="px-3 py-2 text-right">Dist (ATR)</th>
                <th className="px-3 py-2 text-left">Detected</th>
              </tr>
            </thead>
            <tbody>
              {data.marks.map((m) => (
                <tr key={m.id} className="border-t border-pat-border hover:bg-pat-bg-surface-secondary/40">
                  <td className="px-3 py-2">{m.symbol}</td>
                  <td className="px-3 py-2">{m.timeframe}</td>
                  <td className="px-3 py-2">
                    <span className={m.direction === "BULLISH" ? "text-pat-success" : "text-pat-danger"}>
                      {m.direction}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-right font-mono">
                    {typeof m.mark_price === "number" ? m.mark_price.toFixed(2) : "—"}
                  </td>
                  <td className="px-3 py-2">{m.status ?? "—"}</td>
                  <td className="px-3 py-2 text-right">
                    {typeof m.mark_quality_score === "number" ? m.mark_quality_score.toFixed(1) : "—"}
                  </td>
                  <td className="px-3 py-2 text-right">
                    {typeof m.combined_score === "number" ? m.combined_score.toFixed(1) : "—"}
                  </td>
                  <td className="px-3 py-2 text-right">
                    {typeof m.distance_atr === "number" ? m.distance_atr.toFixed(2) : "—"}
                  </td>
                  <td className="px-3 py-2 text-xs text-pat-text-muted">
                    {m.detected_at ? new Date(m.detected_at).toLocaleTimeString() : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
