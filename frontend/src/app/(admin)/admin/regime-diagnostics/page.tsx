"use client";

import { useEffect, useState } from "react";
import { fetchRegimeDiagnostics } from "@/lib/admin-api";


interface RegimeDiagnostics {
  status: string;
  symbol: string;
  timestamp: string;
  current_regime: string;
  previous_regime: string;
  regime_age: string;
  confidence: number;
  entry_reason: string;
  raw_regime: string;
  entered_at: string;
  current_rsi: number;
  current_adx: number;
  current_atr: number;
  volatility: string;
  hold_reason: string;
  regime_engine_version: string;
  ema_alignment: string;
  structure_trend: string;
  last_bos_direction?: string;
  transition_candidate?: string;
  transition_confidence?: number;
  confirmation_count?: number;
  required_confirmations?: number;
}

export default function RegimeDiagnosticsPage() {
  const [data, setData] = useState<RegimeDiagnostics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadData = async () => {
      try {
        const result = await fetchRegimeDiagnostics();
        setData(result);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Failed to load regime diagnostics");
      } finally {
        setLoading(false);
      }
    };
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-muted-foreground">Loading regime diagnostics...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-destructive">Error: {error}</div>
      </div>
    );
  }

  if (!data || data.status === "NO_DATA") {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-muted-foreground">No market data available. Engine may not be running.</div>
      </div>
    );
  }

  const regimeColor = (regime: string) => {
    switch (regime) {
      case "TRENDING_BULLISH": return "text-pat-success";
      case "TRENDING_BEARISH": return "text-pat-danger";
      case "MEAN_REVERSION": return "text-pat-badge-neutral-text";
      case "RANGE": return "text-pat-info";
      case "HIGH_VOLATILITY": return "text-pat-candidate-sell";
      case "BREAKOUT": return "text-cyan-500";
      default: return "text-muted-foreground";
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Regime Diagnostics</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Real-time regime state machine diagnostics from the Go trading engine.
          Engine version: {data.regime_engine_version}
        </p>
      </div>

      {/* Current Regime Card */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="rounded-lg border p-4">
          <div className="text-xs text-muted-foreground mb-1">Current Regime</div>
          <div className={`text-xl font-bold ${regimeColor(data.current_regime)}`}>
            {data.current_regime}
          </div>
          <div className="text-xs text-muted-foreground mt-1">
            Confidence: {(data.confidence * 100).toFixed(1)}%
          </div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-xs text-muted-foreground mb-1">Previous Regime</div>
          <div className={`text-lg font-semibold ${regimeColor(data.previous_regime)}`}>
            {data.previous_regime}
          </div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-xs text-muted-foreground mb-1">Regime Age</div>
          <div className="text-lg font-semibold">{data.regime_age}</div>
          <div className="text-xs text-muted-foreground mt-1">
            Entered: {new Date(data.entered_at).toLocaleString()}
          </div>
        </div>
      </div>

      {/* Market Indicators */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4">
          <div className="text-xs text-muted-foreground mb-1">RSI</div>
          <div className="text-lg font-bold">{data.current_rsi?.toFixed(2)}</div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-xs text-muted-foreground mb-1">ADX</div>
          <div className="text-lg font-bold">{data.current_adx?.toFixed(2)}</div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-xs text-muted-foreground mb-1">ATR</div>
          <div className="text-lg font-bold">{data.current_atr?.toFixed(4)}</div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-xs text-muted-foreground mb-1">Volatility</div>
          <div className="text-lg font-bold">{data.volatility}</div>
        </div>
      </div>

      {/* Regime Analysis */}
      <div className="rounded-lg border p-4 space-y-3">
        <h2 className="text-lg font-semibold">Regime Analysis</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-muted-foreground">Raw Regime (from indicators):</span>{" "}
            <span className={`font-semibold ${regimeColor(data.raw_regime)}`}>{data.raw_regime}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Entry Reason:</span>{" "}
            <span className="font-mono text-xs">{data.entry_reason}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Hold Reason:</span>{" "}
            <span className="font-mono text-xs">{data.hold_reason}</span>
          </div>
          <div>
            <span className="text-muted-foreground">EMA Alignment:</span>{" "}
            <span className="font-semibold">{data.ema_alignment}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Structure Trend:</span>{" "}
            <span className="font-semibold">{data.structure_trend}</span>
          </div>
          {data.last_bos_direction && (
            <div>
              <span className="text-muted-foreground">Last BOS:</span>{" "}
              <span className="font-semibold">{data.last_bos_direction}</span>
            </div>
          )}
        </div>
      </div>

      {/* Transition Candidate */}
      {data.transition_candidate && (
        <div className="rounded-lg border border-yellow-500/50 p-4 space-y-2">
          <h2 className="text-lg font-semibold text-pat-session">Transition Candidate</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
            <div>
              <span className="text-muted-foreground">Candidate Regime:</span>{" "}
              <span className={`font-semibold ${regimeColor(data.transition_candidate)}`}>
                {data.transition_candidate}
              </span>
            </div>
            <div>
              <span className="text-muted-foreground">Confidence:</span>{" "}
              <span className="font-semibold">{((data.transition_confidence || 0) * 100).toFixed(1)}%</span>
            </div>
            <div>
              <span className="text-muted-foreground">Confirmation:</span>{" "}
              <span className="font-semibold">{data.confirmation_count}/{data.required_confirmations}</span>
            </div>
          </div>
        </div>
      )}

      {/* Timestamp */}
      <div className="text-xs text-muted-foreground">
        Last updated: {data.timestamp ? new Date(data.timestamp).toLocaleString() : "N/A"}
      </div>
    </div>
  );
}
