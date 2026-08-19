"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";

interface MarketSnapshot {
  type?: string;
  symbol?: string;
  timestamp?: string;
  source?: string;
  broker?: string;
  node?: string;
  tick?: { bid: number; ask: number; spread: number; volume: number; time: string };
  bars?: Record<string, { open: number; high: number; low: number; close: number; volume: number }>;
  indicators?: Record<string, number | string | boolean>;
  vwap?: { session_vwap: number; upper_band: number; lower_band: number };
  session?: { name: string; is_overlap: boolean; is_weekend: boolean };
  positions?: unknown[];
}

export default function AdminIndicatorsPage() {
  const { data: snapshot, isLoading, error, refetch } = useQuery<MarketSnapshot>({
    queryKey: ["engine-market-snapshot"],
    queryFn: async () => {
      const res = await customInstance.get("/market/snapshot");
      return res.data as MarketSnapshot;
    },
    refetchInterval: 10000,
  });

  // Map Go engine indicator keys to display names
  const indicatorGroups = [
    { name: "Trend", indicators: [
      { key: "ema9", label: "EMA 9" }, { key: "ema21", label: "EMA 21" },
      { key: "ema50", label: "EMA 50" }, { key: "ema100", label: "EMA 100" },
      { key: "ema200", label: "EMA 200" }, { key: "ema_cross_9_21", label: "EMA Cross 9/21" },
      { key: "sma50", label: "SMA 50" }, { key: "sma100", label: "SMA 100" },
      { key: "sma200", label: "SMA 200" },
      { key: "macd_main", label: "MACD Main" }, { key: "macd_signal", label: "MACD Signal" },
      { key: "adx", label: "ADX 14" }, { key: "adx_plus_di", label: "+DI" }, { key: "adx_minus_di", label: "-DI" },
      { key: "psar", label: "Parabolic SAR" }, { key: "psar_long", label: "SAR Direction" },
      { key: "ichimoku_tenkan", label: "Ichimoku Tenkan" }, { key: "ichimoku_kijun", label: "Ichimoku Kijun" },
      { key: "ichimoku_senkou_a", label: "Ichimoku Senkou A" }, { key: "ichimoku_senkou_b", label: "Ichimoku Senkou B" },
    ]},
    { name: "Momentum", indicators: [
      { key: "rsi", label: "RSI 14" },
      { key: "stoch_main", label: "Stochastic Main" }, { key: "stoch_signal", label: "Stochastic Signal" },
      { key: "stoch_rsi", label: "Stochastic RSI" }, { key: "stoch_rsi_k", label: "StochRSI K" }, { key: "stoch_rsi_d", label: "StochRSI D" },
      { key: "cci", label: "CCI 20" },
    ]},
    { name: "Volatility", indicators: [
      { key: "atr", label: "ATR 14" },
      { key: "boll_upper", label: "BB Upper" }, { key: "boll_middle", label: "BB Middle" }, { key: "boll_lower", label: "BB Lower" },
      { key: "boll_width", label: "BB Width" },
    ]},
    { name: "Volume", indicators: [
      { key: "obv", label: "OBV" }, { key: "tick_volume", label: "Tick Volume" },
      { key: "vwap", label: "VWAP" },
    ]},
    { name: "Session / Context", indicators: [
      { key: "session", label: "Session" }, { key: "is_overlap", label: "London/NY Overlap" },
    ]},
  ];

  const formatValue = (val: unknown): string => {
    if (val === null || val === undefined || val === 0) return "—";
    if (typeof val === "boolean") return val ? "Yes" : "No";
    if (typeof val === "number") return val.toFixed(4);
    return String(val);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Indicator Panel</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Live indicator values from the Go real-time engine.</p>
      </div>

      {error && (
        <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-4">
          <div className="text-sm text-pat-warning">Go engine API unavailable</div>
          <div className="text-xs text-pat-text-muted mt-1">
            {error instanceof Error ? error.message : "Live indicator values require the Go realtime engine to be running."}
          </div>
          <button onClick={() => refetch()} className="text-xs bg-pat-bg-surface-secondary px-3 py-1.5 rounded mt-2">Retry</button>
        </div>
      )}

      {/* Live tick and source info */}
      {snapshot?.tick && (
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold text-pat-text-primary">Live Market Data</h2>
            <span className={`text-xs px-2 py-0.5 rounded-full ${snapshot.source === 'MT5_MASTER' ? 'bg-pat-success/10 text-pat-success' : 'bg-pat-warning/10 text-pat-warning'}`}>
              {snapshot.source || 'UNKNOWN'}
            </span>
          </div>
          <div className="grid grid-cols-3 md:grid-cols-6 gap-3 text-sm">
            <div><div className="text-xs text-pat-text-muted">Bid</div><div className="font-mono text-pat-success">{snapshot.tick.bid?.toFixed(2)}</div></div>
            <div><div className="text-xs text-pat-text-muted">Ask</div><div className="font-mono text-pat-danger">{snapshot.tick.ask?.toFixed(2)}</div></div>
            <div><div className="text-xs text-pat-text-muted">Spread</div><div className="font-mono text-pat-text-primary">{snapshot.tick.spread?.toFixed(2)}</div></div>
            <div><div className="text-xs text-pat-text-muted">Volume</div><div className="font-mono text-pat-text-primary">{snapshot.tick.volume}</div></div>
            <div><div className="text-xs text-pat-text-muted">Broker</div><div className="text-xs text-pat-text-secondary">{snapshot.broker || '—'}</div></div>
            <div><div className="text-xs text-pat-text-muted">Node</div><div className="text-xs text-pat-text-secondary">{snapshot.node || '—'}</div></div>
          </div>
        </div>
      )}

      {/* Live indicator values */}
      {snapshot?.indicators && Object.keys(snapshot.indicators).length > 0 && (
        <div className="space-y-4">
          {indicatorGroups.map((group) => {
            const availableIndicators = group.indicators.filter(ind => {
              const val = snapshot.indicators?.[ind.key];
              return val !== undefined && val !== null;
            });
            if (availableIndicators.length === 0) return null;
            return (
              <div key={group.name} className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
                <h2 className="text-sm font-semibold text-pat-text-primary mb-3">{group.name}</h2>
                <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-2">
                  {availableIndicators.map((ind) => {
                    const val = snapshot.indicators?.[ind.key];
                    return (
                      <div key={ind.key} className="rounded-md bg-pat-bg-surface-secondary/50 px-3 py-2">
                        <div className="text-xs text-pat-text-secondary">{ind.label}</div>
                        <div className="text-sm font-mono text-pat-text-primary">{formatValue(val)}</div>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Fallback: static inventory when no live data */}
      {(!snapshot?.indicators || Object.keys(snapshot.indicators).length === 0) && !error && (
        <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-4">
          <div className="text-sm text-pat-warning">No live indicator values available</div>
          <div className="text-xs text-pat-text-muted mt-1">
            The Go engine is connected but no indicator data is being returned. This may indicate market data is not flowing.
          </div>
        </div>
      )}

      {isLoading && <div className="text-xs text-pat-text-muted">Loading indicator data from Go engine...</div>}
    </div>
  );
}
