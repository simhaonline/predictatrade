"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { strategyLabel } from "@/lib/strategy-labels";

interface MarketState {
  MTF?: { Score?: number; States?: Record<string, number> };
  Indicators?: Record<string, string | number | boolean>;
}

interface MtfPulseProps {
  snapshot?: { indicators?: Record<string, number | string | boolean>; bars?: Record<string, { close: number; high: number; low: number; open: number }> };
}

export function MtfPulse({ snapshot }: MtfPulseProps) {
  const { data: marketState } = useQuery<MarketState>({
    queryKey: ["user-mtf-state"],
    queryFn: async () => (await customInstance.get("/market/state")).data,
    refetchInterval: 5000,
  });

  const mtfStates = marketState?.MTF?.States ?? {};
  const mtfScore = marketState?.MTF?.Score ?? 0;
  const indicators = snapshot?.indicators ?? {};
  const bars = snapshot?.bars ?? {};

  const timeframes = ["M1", "M5", "M15", "M30", "H1", "H4", "D1"];

  const stateLabel = (s: number | undefined): string => {
    if (s === undefined) return "—";
    if (s > 0) return "BULL";
    if (s < 0) return "BEAR";
    return "NEUT";
  };

  const stateColor = (s: number | undefined): string => {
    if (s === undefined) return "text-pat-text-muted";
    if (s > 0) return "text-pat-success";
    if (s < 0) return "text-pat-danger";
    return "text-pat-text-secondary";
  };

  const consensus = mtfScore > 20 ? "STRONG_BULLISH" : mtfScore > 5 ? "BULLISH" : mtfScore < -20 ? "STRONG_BEARISH" : mtfScore < -5 ? "BEARISH" : mtfScore !== 0 ? "NEUTRAL" : "INSUFFICIENT_DATA";
  const consensusColor = consensus.includes("BULLISH") ? "text-pat-success" : consensus.includes("BEARISH") ? "text-pat-danger" : "text-pat-text-secondary";

  return (
    <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-pat-text-primary">Multi-Timeframe Pulse</h3>
        <div className="flex items-center gap-2">
          <span className="text-[10px] text-pat-text-muted">Consensus:</span>
          <span className={`text-xs font-bold ${consensusColor}`}>{strategyLabel(consensus)}</span>
          <span className="text-[10px] text-pat-text-muted">({mtfScore.toFixed(1)})</span>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="text-[10px] text-pat-text-muted border-b border-pat-border">
              <th className="text-left py-1.5 px-2">TF</th>
              <th className="text-center py-1.5 px-2">Trend</th>
              <th className="text-right py-1.5 px-2">Close</th>
              <th className="text-right py-1.5 px-2">EMA9</th>
              <th className="text-right py-1.5 px-2">EMA21</th>
              <th className="text-right py-1.5 px-2">RSI</th>
            </tr>
          </thead>
          <tbody>
            {timeframes.map((tf) => {
              const state = mtfStates[tf];
              const bar = bars[tf];
              const close = bar?.close;
              const ema9 = Number(indicators["ema9"] ?? 0);
              const ema21 = Number(indicators["ema21"] ?? 0);
              const rsi = Number(indicators["rsi"] ?? 0);
              return (
                <tr key={tf} className="border-b border-pat-border/30 hover:bg-pat-bg-surface-secondary/30">
                  <td className="py-1.5 px-2 text-pat-text-primary font-medium">{tf}</td>
                  <td className="py-1.5 px-2 text-center">
                    <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-bold ${stateColor(state)} bg-pat-bg-surface-secondary/50`}>
                      {stateLabel(state)}
                    </span>
                  </td>
                  <td className="py-1.5 px-2 text-right font-mono text-pat-text-secondary tabular-nums">{close ? close.toFixed(2) : "—"}</td>
                  <td className="py-1.5 px-2 text-right font-mono text-pat-text-muted tabular-nums">{ema9 > 0 ? ema9.toFixed(2) : "—"}</td>
                  <td className="py-1.5 px-2 text-right font-mono text-pat-text-muted tabular-nums">{ema21 > 0 ? ema21.toFixed(2) : "—"}</td>
                  <td className="py-1.5 px-2 text-right font-mono text-pat-text-muted tabular-nums">{rsi > 0 ? rsi.toFixed(1) : "—"}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
