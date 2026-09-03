"use client";

import { useState, useEffect, useCallback } from "react";
import {
  IconChartPie,
  IconLoader,
  IconAlertTriangle,
  IconRefresh,
  IconTrendingUp,
  IconTrendingDown,
} from "@tabler/icons-react";
import {
  fetchPerformanceByPlan,
  type PerformanceByPlanResponse,
  type PlanStrategyPerformance,
} from "@/lib/backtest-api";

const usd = (n: number) =>
  `${n < 0 ? "-" : ""}$${Math.abs(n).toLocaleString("en-US", { minimumFractionDigits: 0, maximumFractionDigits: 2 })}`;

const pct = (n: number) => `${n >= 0 ? "+" : ""}${n.toFixed(2)}%`;

const pnlTone = (n: number) =>
  n > 0
    ? "text-pat-badge-success-text"
    : n < 0
      ? "text-pat-badge-danger-text"
      : "text-pat-text-muted";

function StrategyRow({ s }: { s: PlanStrategyPerformance }) {
  const positive = s.totalPnl > 0;
  return (
    <div className="flex items-center justify-between gap-3 py-2 border-b border-pat-border/50 last:border-b-0">
      <div className="min-w-0">
        <div className="text-xs font-medium text-pat-text-primary truncate">
          {s.strategyId.replace(/_/g, " ")}
        </div>
        <div className="text-xs text-pat-text-muted">
          {s.runs} run{s.runs === 1 ? "" : "s"} · win {s.avgWinRate.toFixed(1)}% · PF {s.avgProfitFactor.toFixed(2)} · {s.profitableRuns} profitable
        </div>
      </div>
      <div className="text-right shrink-0">
        <div className={`text-sm font-semibold tabular-nums ${pnlTone(s.totalPnl)}`}>
          {positive ? <IconTrendingUp size={12} className="inline mr-1 -mt-px" /> : s.totalPnl < 0 ? <IconTrendingDown size={12} className="inline mr-1 -mt-px" /> : null}
          {usd(s.totalPnl)}
        </div>
        <div className={`text-xs tabular-nums ${pnlTone(s.avgReturnPct)}`}>
          avg {pct(s.avgReturnPct)}
        </div>
      </div>
    </div>
  );
}

export default function PlanPerformanceCards() {
  const [data, setData] = useState<PerformanceByPlanResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setData(await fetchPerformanceByPlan());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to fetch plan performance");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <IconChartPie size={18} className="text-pat-primary" />
        <h2 className="text-sm font-semibold text-pat-text-primary">
          Backtest Performance by Plan
        </h2>
        <span className="text-xs text-pat-text-muted ml-auto">
          Aggregated P/L of backtested strategies per subscription plan
        </span>
        <button
          type="button"
          onClick={() => void load()}
          title="Refresh"
          className="text-pat-text-muted hover:text-pat-text-primary ml-1"
        >
          <IconRefresh size={14} />
        </button>
      </div>

      {loading && !data && (
        <div className="text-xs text-pat-text-muted flex items-center gap-1">
          <IconLoader size={12} className="animate-spin" /> Loading plan performance...
        </div>
      )}

      {error && !data && (
        <div className="flex items-center gap-2 text-xs text-pat-badge-danger-text bg-pat-badge-danger-bg border border-pat-border rounded-md p-3">
          <IconAlertTriangle size={14} /> {error}
        </div>
      )}

      {data && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {data.plans.map((p) => (
            <div
              key={p.planCode}
              className="border border-pat-border rounded-lg p-4 bg-pat-bg-surface-secondary flex flex-col gap-2"
            >
              <div className="flex items-baseline justify-between">
                <div className="text-sm font-semibold text-pat-text-primary">{p.planName}</div>
                <div className="text-xs text-pat-text-muted">
                  ${p.monthlyPrice.toLocaleString("en-US")}/mo
                </div>
              </div>
              <div className="flex items-center gap-3 text-xs text-pat-text-muted">
                <span>
                  {p.totals.runs} backtest{p.totals.runs === 1 ? "" : "s"}
                </span>
                <span>·</span>
                <span>win {p.totals.avgWinRate.toFixed(1)}%</span>
                <span>·</span>
                <span className={`tabular-nums font-medium ${pnlTone(p.totals.totalPnl)}`}>
                  net {usd(p.totals.totalPnl)}
                </span>
              </div>
              <div className="border-t border-pat-border pt-1">
                {p.strategies.length === 0 ? (
                  <div className="text-xs text-pat-text-muted py-2">
                    No backtested strategies in this plan yet.
                  </div>
                ) : (
                  p.strategies.map((s) => <StrategyRow key={s.strategyId} s={s} />)
                )}
              </div>
            </div>
          ))}
        </div>
      )}
      <p className="mt-3 text-xs text-pat-text-muted">
        Aggregated from COMPLETED backtest runs by strategy, mapped to each plan&apos;s
        allowed strategies. P/L = sum of (final − initial balance) across runs at the
        run&apos;s own initial balance. Past backtest results are not indicative of
        future live performance.
      </p>
    </div>
  );
}