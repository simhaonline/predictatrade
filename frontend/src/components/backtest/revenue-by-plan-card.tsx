"use client";

import { useState, useEffect, useCallback } from "react";
import { IconCoin, IconLoader, IconAlertTriangle, IconRefresh } from "@tabler/icons-react";
import {
  fetchRevenueByPlan,
  type RevenueByPlanResponse,
} from "@/lib/backtest-api";

const usd = (n: number) =>
  `$${n.toLocaleString("en-US", { minimumFractionDigits: 0, maximumFractionDigits: 2 })}`;

export default function RevenueByPlanCard() {
  const [data, setData] = useState<RevenueByPlanResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setData(await fetchRevenueByPlan());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to fetch revenue by plan");
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
        <IconCoin size={18} className="text-pat-primary" />
        <h2 className="text-sm font-semibold text-pat-text-primary">
          Revenue by Subscription Plan
        </h2>
        <span className="text-xs text-pat-text-muted ml-auto">
          Subscriptions · MRR · Collected · Backtest usage
        </span>
        <button
          type="button"
          onClick={() => void load()}
          title="Refresh"
          className="text-pat-text-muted hover:text-pat-text-primary ml-1"
        >
          <IconRefresh size={14} />
        </button>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="text-xs text-pat-text-muted hover:text-pat-text-primary"
        >
          {open ? "Hide" : "Show"}
        </button>
      </div>

      {loading && !data && (
        <div className="text-xs text-pat-text-muted flex items-center gap-1">
          <IconLoader size={12} className="animate-spin" /> Loading revenue attribution...
        </div>
      )}

      {error && !data && (
        <div className="flex items-center gap-2 text-xs text-pat-badge-danger-text bg-pat-badge-danger-bg border border-pat-border rounded-md p-3">
          <IconAlertTriangle size={14} /> {error}
        </div>
      )}

      {data && open && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3 mb-4">
            <div className="p-3 bg-pat-bg-surface-secondary rounded-md">
              <div className="text-xs text-pat-text-muted">Monthly Recurring Revenue</div>
              <div className="text-lg font-semibold text-pat-text-primary tabular-nums">
                {usd(data.totals.mrr)}
              </div>
            </div>
            <div className="p-3 bg-pat-bg-surface-secondary rounded-md">
              <div className="text-xs text-pat-text-muted">Collected Revenue</div>
              <div className="text-lg font-semibold text-pat-text-primary tabular-nums">
                {usd(data.totals.collectedRevenue)}
              </div>
            </div>
            <div className="p-3 bg-pat-bg-surface-secondary rounded-md">
              <div className="text-xs text-pat-text-muted">Attributed Backtest Runs</div>
              <div className="text-lg font-semibold text-pat-text-primary tabular-nums">
                {data.totals.backtestRuns.toLocaleString("en-US")}
              </div>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left text-pat-text-muted border-b border-pat-border">
                  <th className="py-2 pr-3 font-medium">Plan</th>
                  <th className="py-2 pr-3 font-medium text-right">Price / mo</th>
                  <th className="py-2 pr-3 font-medium text-right">Active subs</th>
                  <th className="py-2 pr-3 font-medium text-right">MRR</th>
                  <th className="py-2 pr-3 font-medium text-right">Collected</th>
                  <th className="py-2 pr-3 font-medium text-right">Backtests</th>
                  <th className="py-2 font-medium">Strategies used</th>
                </tr>
              </thead>
              <tbody>
                {data.plans.map((p) => (
                  <tr key={p.planCode} className="border-b border-pat-border/50">
                    <td className="py-2 pr-3">
                      <span className="font-medium text-pat-text-primary">{p.planName}</span>
                      <span className="text-pat-text-muted ml-1.5">({p.planCode})</span>
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums text-pat-text-secondary">
                      {usd(p.monthlyPrice)}
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums text-pat-text-secondary">
                      {p.activeSubscriptions}
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums text-pat-text-secondary">
                      {usd(p.mrr)}
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums text-pat-text-secondary">
                      {usd(p.collectedRevenue)}
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums text-pat-text-secondary">
                      {p.backtestRuns}
                    </td>
                    <td className="py-2 text-pat-text-muted">
                      {p.strategiesUsed.length === 0
                        ? "—"
                        : p.strategiesUsed.join(", ")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="mt-3 text-xs text-pat-text-muted">
            MRR = sum of plan prices across ACTIVE/TRIALING subscriptions. Collected =
            SUCCEEDED payments. New backtest runs are attributed to the runner&apos;s
            subscription automatically; platform-era runs (before attribution shipped)
            are not counted.
          </p>
        </>
      )}
    </div>
  );
}
