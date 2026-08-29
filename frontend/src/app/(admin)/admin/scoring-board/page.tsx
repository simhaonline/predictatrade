"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { strategyLabel } from "@/lib/strategy-labels";

interface GoSignal {
  ID: string;
  Symbol: string;
  StrategyID: string;
  Direction: string;
  Grade: string;
  RawScore: string;
  LongScore: string;
  ShortScore: string;
  CalibratedProbability: string;
  EntryPrice: string;
  StopLoss: string;
  TP1: string;
  TP2: string;
  TP3: string;
  Regime: string;
  Session: string;
  NewsRisk: string;
  Timeframe: string;
  Status: string;
  ReasonCodes: string[] | null;
  Evidence: unknown;
  GateResults: unknown;
  CreatedAt: string;
  ExpiresAt: string;
}

interface MarketState {
  Symbol: string;
  Bid: string;
  Ask: string;
  Spread: string;
  LastTick: { Source: string; SourceTimestamp: string };
  Regime: { Current: string; Volatility: string; Confidence: number };
  Session: { CurrentSession: string; IsOverlap: boolean };
  Indicators: Record<string, string>;
  Timestamp: string;
}

export default function AdminScoringBoardPage() {
  const { data: marketState } = useQuery<MarketState>({
    queryKey: ["engine-market-state-scoring"],
    queryFn: async () => (await customInstance.get("/market/state")).data,
    refetchInterval: 5000,
  });

  const { data: signalsData, isLoading: signalsLoading } = useQuery<{ signals: GoSignal[] }>({
    queryKey: ["engine-signals-scoring"],
    queryFn: async () => (await customInstance.get("/signals")).data,
    refetchInterval: 10000,
  });

  const signals = signalsData?.signals ?? [];
  const latestByStrategy = signals.reduce((acc, s) => {
    if (!acc[s.StrategyID] || new Date(s.CreatedAt) > new Date(acc[s.StrategyID].CreatedAt)) {
      acc[s.StrategyID] = s;
    }
    return acc;
  }, {} as Record<string, GoSignal>);

  const strategies = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB"];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Scoring Board</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Live market state, scoring metrics, and hard gate status from the Go engine.</p>
      </div>

      {/* Market State */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted uppercase mb-1">Bid</div>
          <div className="text-2xl font-mono text-neonGreen">{marketState?.Bid ? parseFloat(marketState.Bid).toFixed(2) : "—"}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted uppercase mb-1">Ask</div>
          <div className="text-2xl font-mono text-neonRed">{marketState?.Ask ? parseFloat(marketState.Ask).toFixed(2) : "—"}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted uppercase mb-1">Spread</div>
          <div className={`text-2xl font-mono ${(marketState ? parseFloat(marketState.Spread) : 0) > 0.5 ? "text-pat-warning" : "text-pat-text-primary"}`}>{marketState?.Spread ? parseFloat(marketState.Spread).toFixed(2) : "—"}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted uppercase mb-1">Session</div>
          <div className="text-lg font-medium text-pat-text-primary">{marketState?.Session?.CurrentSession ?? "—"}</div>
          {marketState?.Session?.IsOverlap && <div className="text-xs text-pat-warning">London/NY Overlap</div>}
        </div>
      </div>

      {/* Market Regime */}
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Market Regime</h2>
        <div className="flex items-center gap-3">
          <StatusBadge status={marketState?.Regime?.Current ?? "UNKNOWN"} />
          <span className="text-xs text-pat-text-muted">
            Volatility: {marketState?.Regime?.Volatility ?? "—"} · Confidence: {marketState?.Regime?.Confidence ? `${(marketState.Regime.Confidence * 100).toFixed(0)}%` : "—"}
          </span>
        </div>
        {marketState?.LastTick?.Source && (
          <div className="text-xs text-pat-text-muted mt-2">
            Source: {marketState.LastTick.Source} · Last tick: {marketState.LastTick.SourceTimestamp ? format(new Date(marketState.LastTick.SourceTimestamp), "HH:mm:ss") : "—"}
          </div>
        )}
      </div>

      {/* Strategy Scoring Results */}
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Strategy Evaluation Results</h2>
        <div className="space-y-3">
          {strategies.map((strat) => {
            const sig = latestByStrategy[strat];
            return (
              <div key={strategyLabel(strat)} className="rounded-md bg-pat-bg-surface-secondary/50 px-4 py-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-pat-text-primary">{strategyLabel(strat)}</span>
                  <span className={`text-xs font-bold px-2 py-0.5 rounded ${
                    sig?.Direction === "BUY" ? "bg-pat-success/10 text-pat-success" :
                    sig?.Direction === "SELL" ? "bg-pat-danger/10 text-pat-danger" :
                    "bg-pat-badge-neutral-bg/10 text-pat-badge-neutral-text"
                  }`}>{sig?.Direction ?? "NO-TRADE"}</span>
                </div>
                {sig ? (
                  <div className="grid grid-cols-2 md:grid-cols-6 gap-2 text-xs">
                    <div><div className="text-pat-text-muted">Raw Score</div><div className="font-mono text-pat-text-primary">{sig.RawScore}</div></div>
                    <div><div className="text-pat-text-muted">Long Score</div><div className="font-mono text-pat-success">{sig.LongScore}</div></div>
                    <div><div className="text-pat-text-muted">Short Score</div><div className="font-mono text-pat-danger">{sig.ShortScore}</div></div>
                    <div><div className="text-pat-text-muted">Probability</div><div className="font-mono text-pat-text-primary">{sig.CalibratedProbability ? `${(parseFloat(sig.CalibratedProbability) * 100).toFixed(1)}%` : "—"}</div></div>
                    <div><div className="text-pat-text-muted">Regime</div><div className="text-pat-text-secondary">{sig.Regime || "—"}</div></div>
                    <div><div className="text-pat-text-muted">Timeframe</div><div className="text-pat-text-secondary">{sig.Timeframe || "—"}</div></div>
                    {sig.EntryPrice !== "0" && (
                      <div><div className="text-pat-text-muted">Entry</div><div className="font-mono text-pat-text-primary">{parseFloat(sig.EntryPrice).toFixed(2)}</div></div>
                    )}
                    {sig.StopLoss !== "0" && (
                      <div><div className="text-pat-text-muted">SL</div><div className="font-mono text-pat-danger">{parseFloat(sig.StopLoss).toFixed(2)}</div></div>
                    )}
                    {sig.TP1 !== "0" && (
                      <div><div className="text-pat-text-muted">TP1</div><div className="font-mono text-pat-success">{parseFloat(sig.TP1).toFixed(2)}</div></div>
                    )}
                    <div><div className="text-pat-text-muted">Evaluated</div><div className="text-pat-text-muted">{sig.CreatedAt ? format(new Date(sig.CreatedAt), "HH:mm:ss") : "—"}</div></div>
                    <div><div className="text-pat-text-muted">Status</div><div className="text-pat-text-secondary">{sig.Status}</div></div>
                  </div>
                ) : (
                  <div className="text-xs text-pat-text-muted">No evaluation data</div>
                )}
              </div>
            );
          })}
        </div>
        <p className="text-xs text-pat-text-muted mt-3">
          NO-TRADE is a valid outcome. Hard gates fail closed when data is unavailable. Scoring is not weakened to produce signals.
          {signals.length > 0 && ` · Total evaluations: ${signals.length}`}
        </p>
      </div>

      {/* 12 Hard Gates */}
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">12 Hard Gates</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {["Data Quality", "Spread", "News Restriction", "Session Restriction", "Risk Per Trade", "Daily Loss", "Exposure", "Margin", "Cooldown", "Signal TTL", "License/Entitlement", "Emergency Stop"].map((gate) => (
            <div key={gate} className="flex items-center gap-2 rounded-md bg-pat-bg-surface-secondary/50 px-3 py-2">
              <div className="w-2 h-2 rounded-full bg-neutral-600" />
              <span className="text-xs text-pat-text-secondary">{gate}</span>
            </div>
          ))}
        </div>
        <p className="text-xs text-pat-text-muted mt-3">Gate status is determined by the Go engine at evaluation time. Gates fail closed when data is unavailable.</p>
      </div>

      {signalsLoading && <div className="text-xs text-pat-text-muted">Loading scoring data from Go engine...</div>}
    </div>
  );
}
