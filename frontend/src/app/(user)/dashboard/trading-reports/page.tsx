"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconChartBar, IconCoin, IconReceipt } from "@tabler/icons-react";

export default function UserTradingReportsPage() {
  const { data: subscriptions } = useQuery({
    queryKey: ["user-subscriptions"],
    queryFn: async () => {
      const res = await customInstance.get("/subscriptions");
      return res.data;
    },
  });

  const { data: commissionSummary } = useQuery({
    queryKey: ["user-commission-summary"],
    queryFn: async () => {
      const res = await customInstance.get("/commissions/summary");
      return res.data as { total_amount: string; pending_count: string; confirmed_count: string; pending_amount: string; confirmed_amount: string };
    },
  });

  const { data: invoices } = useQuery({
    queryKey: ["user-invoices"],
    queryFn: async () => {
      const res = await customInstance.get("/billing/invoices");
      return res.data;
    },
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Trading Reports</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Your personal trading performance and financial overview.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2"><span className="text-sm text-pat-text-secondary">Active Subscriptions</span><IconReceipt size={18} className="text-pat-success" /></div>
          <div className="text-2xl font-bold text-pat-text-primary">{Array.isArray(subscriptions) ? subscriptions.length : 0}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2"><span className="text-sm text-pat-text-secondary">Total Earnings</span><IconCoin size={18} className="text-pat-session" /></div>
          <div className="text-2xl font-bold text-pat-text-primary">${parseFloat(commissionSummary?.total_amount ?? "0").toFixed(2)}</div>
          <div className="text-xs text-pat-text-muted mt-1">{commissionSummary?.confirmed_count ?? "0"} confirmed commissions</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2"><span className="text-sm text-pat-text-secondary">Total Invoices</span><IconChartBar size={18} className="text-pat-info" /></div>
          <div className="text-2xl font-bold text-pat-text-primary">{Array.isArray(invoices) ? invoices.length : 0}</div>
        </div>
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Commission Breakdown</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div><div className="text-xs text-pat-text-muted">Confirmed</div><div className="text-lg font-semibold text-pat-success">${parseFloat(commissionSummary?.confirmed_amount ?? "0").toFixed(2)}</div></div>
          <div><div className="text-xs text-pat-text-muted">Pending</div><div className="text-lg font-semibold text-pat-warning">${parseFloat(commissionSummary?.pending_amount ?? "0").toFixed(2)}</div></div>
          <div><div className="text-xs text-pat-text-muted">Confirmed Count</div><div className="text-lg font-semibold text-pat-text-primary">{commissionSummary?.confirmed_count ?? "0"}</div></div>
          <div><div className="text-xs text-pat-text-muted">Pending Count</div><div className="text-lg font-semibold text-pat-text-primary">{commissionSummary?.pending_count ?? "0"}</div></div>
        </div>
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Live Signal Statistics</h2>
        <p className="text-xs text-pat-text-muted">Real-time signal statistics are available on the Live Dashboard and Signals page.</p>
      </div>
    </div>
  );
}
