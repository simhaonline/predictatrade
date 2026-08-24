"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";

interface MacroCurrent {
  score: number;
  direction: string;
  confidence: number;
  data_quality: string;
  regime: string;
  drivers: { name: string; direction: string; impact_score: number; quality: string }[];
  missing_drivers: string[] | null;
  reason: string;
}

function biasLabel(score: number): string {
  if (score > 40) return "Strongly Bullish";
  if (score > 15) return "Bullish";
  if (score < -40) return "Strongly Bearish";
  if (score < -15) return "Bearish";
  return "Neutral";
}

function biasColor(score: number): string {
  if (score > 15) return "text-pat-success";
  if (score < -15) return "text-pat-danger";
  return "text-pat-text-secondary";
}

function driverContext(name: string, direction: string): string {
  const labels: Record<string, string> = {
    dxy: "US Dollar", eurusd: "EURUSD", cot: "Positioning", real_yields: "Real Yields",
    vix: "Risk Sentiment", btc: "Crypto", oil: "Oil / Inflation",
  };
  const label = labels[name] || name;
  if (direction === "BULLISH") {
    if (name === "dxy") return `${label}: Opposes Gold`;
    return `${label}: Supports Gold`;
  }
  if (direction === "BEARISH") {
    if (name === "dxy") return `${label}: Supports Gold`;
    return `${label}: Opposes Gold`;
  }
  return `${label}: Neutral`;
}

export function MarketContextPanel() {
  const { data: macro } = useQuery<MacroCurrent>({
    queryKey: ["user-market-context"],
    queryFn: async () => (await customInstance.get("/cross-market/current")).data,
    refetchInterval: 15000,
  });

  if (!macro) {
    return (
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="text-sm font-semibold text-pat-text-primary mb-2">Market Context</div>
        <div className="text-xs text-pat-text-muted">Loading...</div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-pat-text-primary">Market Context</h3>
        <span className={`text-xs font-medium ${biasColor(macro.score)}`}>
          {biasLabel(macro.score)}
        </span>
      </div>

      {/* Confidence bar */}
      <div className="mb-3">
        <div className="flex items-center justify-between text-[10px] text-pat-text-muted mb-1">
          <span>Confidence</span>
          <span>{(macro.confidence * 100).toFixed(0)}%</span>
        </div>
        <div className="h-1.5 rounded bg-pat-bg-surface-secondary overflow-hidden">
          <div
            className="h-full bg-pat-info transition-all"
            style={{ width: `${macro.confidence * 100}%` }}
          />
        </div>
      </div>

      {/* Driver context */}
      <div className="space-y-1">
        {macro.drivers.filter(d => d.quality === "CONNECTED").slice(0, 5).map((drv, i) => (
          <div key={i} className="flex items-center justify-between text-[11px]">
            <span className="text-pat-text-muted">{driverContext(drv.name, drv.direction)}</span>
            <span className={`tabular-nums ${drv.impact_score > 0 ? "text-pat-success" : drv.impact_score < 0 ? "text-pat-danger" : "text-pat-text-muted"}`}>
              {drv.impact_score > 0 ? "+" : ""}{drv.impact_score.toFixed(0)}
            </span>
          </div>
        ))}
      </div>

      {/* Regime */}
      {macro.regime && macro.regime !== "UNKNOWN" && (
        <div className="mt-3 pt-2 border-t border-pat-border/30">
          <div className="flex items-center justify-between text-[11px]">
            <span className="text-pat-text-muted">Regime</span>
            <span className="text-pat-text-secondary">{macro.regime.replace(/_/g, " ").toLowerCase()}</span>
          </div>
        </div>
      )}
    </div>
  );
}
