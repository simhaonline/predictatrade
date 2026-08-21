"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";

interface SnapshotData {
  indicators?: Record<string, number | string | boolean>;
  source?: string;
}

interface IndicatorCardsProps {
  snapshot?: SnapshotData;
}

interface CardDef {
  key: string;
  label: string;
  group: string;
  interpret: (v: number, all: Record<string, number | string | boolean>) => { state: string; color: string };
}

const INDICATOR_CARDS: CardDef[] = [
  { key: "rsi", label: "RSI 14", group: "Momentum", interpret: (v) => v > 70 ? { state: "Overbought", color: "text-pat-danger" } : v < 30 ? { state: "Oversold", color: "text-pat-success" } : v > 50 ? { state: "Bullish bias", color: "text-pat-success" } : { state: "Bearish bias", color: "text-pat-danger" } },
  { key: "macd_main", label: "MACD", group: "Momentum", interpret: (v) => v > 0 ? { state: "Above zero", color: "text-pat-success" } : { state: "Below zero", color: "text-pat-danger" } },
  { key: "macd_histogram", label: "MACD Hist", group: "Momentum", interpret: (v) => v > 0 ? { state: "Rising", color: "text-pat-success" } : v < 0 ? { state: "Falling", color: "text-pat-danger" } : { state: "Flat", color: "text-pat-text-muted" } },
  { key: "adx", label: "ADX 14", group: "Trend", interpret: (v) => v > 25 ? { state: "Strong trend", color: "text-pat-success" } : v > 20 ? { state: "Trending", color: "text-pat-text-secondary" } : { state: "Ranging", color: "text-pat-text-muted" } },
  { key: "atr", label: "ATR 14", group: "Volatility", interpret: (v) => v > 20 ? { state: "High vol", color: "text-pat-warning" } : v > 10 ? { state: "Normal", color: "text-pat-text-secondary" } : { state: "Low vol", color: "text-pat-text-muted" } },
  { key: "cci", label: "CCI 20", group: "Momentum", interpret: (v) => v > 100 ? { state: "Overbought", color: "text-pat-danger" } : v < -100 ? { state: "Oversold", color: "text-pat-success" } : { state: "Neutral", color: "text-pat-text-secondary" } },
  { key: "stoch_main", label: "Stochastic", group: "Momentum", interpret: (v) => v > 80 ? { state: "Overbought", color: "text-pat-danger" } : v < 20 ? { state: "Oversold", color: "text-pat-success" } : { state: "Neutral", color: "text-pat-text-secondary" } },
  { key: "boll_width", label: "BB Width", group: "Volatility", interpret: (v) => v > 0.01 ? { state: "Expanding", color: "text-pat-warning" } : v > 0 ? { state: "Compressing", color: "text-pat-text-secondary" } : { state: "—", color: "text-pat-text-muted" } },
  { key: "obv", label: "OBV", group: "Volume", interpret: (v) => v > 0 ? { state: "Accumulation", color: "text-pat-success" } : v < 0 ? { state: "Distribution", color: "text-pat-danger" } : { state: "Neutral", color: "text-pat-text-muted" } },
  { key: "vwap", label: "VWAP", group: "Volume", interpret: (v) => v > 0 ? { state: "Active", color: "text-pat-text-secondary" } : { state: "Unavailable", color: "text-pat-text-muted" } },
  { key: "psar", label: "Parabolic SAR", group: "Trend", interpret: (v) => v > 0 ? { state: "Active", color: "text-pat-text-secondary" } : { state: "—", color: "text-pat-text-muted" } },
  { key: "ichimoku_tenkan", label: "Ichimoku", group: "Trend", interpret: (v) => v > 0 ? { state: "Active", color: "text-pat-text-secondary" } : { state: "Building", color: "text-pat-text-muted" } },
];

const EMA_CARDS = [
  { key: "ema9", label: "EMA 9" },
  { key: "ema21", label: "EMA 21" },
  { key: "ema50", label: "EMA 50" },
  { key: "ema100", label: "EMA 100" },
  { key: "ema200", label: "EMA 200" },
  { key: "sma50", label: "SMA 50" },
  { key: "sma100", label: "SMA 100" },
  { key: "sma200", label: "SMA 200" },
];

export function IndicatorCards({ snapshot }: IndicatorCardsProps) {
  const { data: fetchedSnapshot } = useQuery<SnapshotData>({
    queryKey: ["user-indicator-cards"],
    queryFn: async () => (await customInstance.get("/market/snapshot")).data,
    refetchInterval: 5000,
  });

  const all = snapshot?.indicators ?? fetchedSnapshot?.indicators ?? {};
  

  const fmtVal = (v: number | string | boolean | undefined): string => {
    if (v === undefined || v === null) return "—";
    if (typeof v === "boolean") return v ? "Yes" : "No";
    if (typeof v === "number") {
      if (v === 0) return "0";
      if (Math.abs(v) < 0.01) return v.toExponential(2);
      return v.toFixed(2);
    }
    return String(v);
  };

  return (
    <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
      <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Indicator Intelligence</h3>
      
      {/* EMA/SMA alignment strip */}
      <div className="flex flex-wrap gap-1.5 mb-3 pb-3 border-b border-pat-border/50">
        {EMA_CARDS.map((ema) => {
          const val = Number(all[ema.key] ?? 0);
          const active = val > 0;
          return (
            <div key={ema.key} className={`rounded-md px-2 py-1 text-[11px] border ${active ? "border-pat-border/60 bg-pat-bg-surface-secondary/40" : "border-transparent bg-pat-bg-surface-secondary/20 opacity-50"}`}>
              <span className="text-pat-text-muted">{ema.label}: </span>
              <span className="font-mono text-pat-text-secondary tabular-nums">{active ? val.toFixed(2) : "—"}</span>
            </div>
          );
        })}
      </div>

      {/* Indicator cards grid */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2">
        {INDICATOR_CARDS.map((card) => {
          const rawVal = all[card.key];
          const numVal = typeof rawVal === "number" ? rawVal : 0;
          const interp = card.interpret(numVal, all);
          const active = numVal !== 0 || rawVal === true;
          return (
            <div key={card.key} className={`rounded-lg border p-2.5 transition-all ${active ? "border-pat-border/60 bg-pat-bg-surface-secondary/30" : "border-pat-border/30 bg-pat-bg-surface-secondary/10 opacity-60"}`}>
              <div className="flex items-center justify-between mb-1">
                <span className="text-[10px] text-pat-text-muted uppercase">{card.label}</span>
                <span className="text-[9px] text-pat-text-muted">{card.group}</span>
              </div>
              <div className="font-mono text-sm text-pat-text-primary tabular-nums">{fmtVal(rawVal)}</div>
              <div className={`text-[10px] mt-0.5 ${interp.color}`}>{interp.state}</div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
