"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconUsers, IconReceipt, IconChartBar, IconActivity } from "@tabler/icons-react";

interface OverviewResponse {
  users: { total: string; active: string; suspended: string; new_this_month: string };
  subscriptions: { total: string; active: string; mrr: string };
  commissions: { total_entries: string; pending_amount: string; confirmed_amount: string };
  payouts: { total: string; pending: string; pending_amount: string };
}

export default function AdminTradingReportsPage() {
  const { data: overview } = useQuery<OverviewResponse>({
    queryKey: ["admin-overview"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/overview");
      return res.data as OverviewResponse;
    },
  });

  const { data: agents } = useQuery({
    queryKey: ["engine-agents"],
    queryFn: async () => {
      const res = await customInstance.get("/agents/status");
      return res.data;
    },
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Trading Reports</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Aggregate trading performance and system activity across all clients.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2"><span className="text-sm text-pat-text-secondary">Total Users</span><IconUsers size={18} className="text-pat-info" /></div>
          <div className="text-2xl font-bold text-pat-text-primary">{parseInt(overview?.users?.total ?? "0").toLocaleString()}</div>
          <div className="text-xs text-pat-text-muted mt-1">{overview?.users?.active ?? "0"} active</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2"><span className="text-sm text-pat-text-secondary">Active Subscriptions</span><IconReceipt size={18} className="text-pat-success" /></div>
          <div className="text-2xl font-bold text-pat-text-primary">{overview?.subscriptions?.active ?? "0"}</div>
          <div className="text-xs text-pat-text-muted mt-1">MRR ${parseFloat(overview?.subscriptions?.mrr ?? "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2"><span className="text-sm text-pat-text-secondary">Commissions</span><IconChartBar size={18} className="text-pat-session" /></div>
          <div className="text-2xl font-bold text-pat-text-primary">{overview?.commissions?.total_entries ?? "0"}</div>
          <div className="text-xs text-pat-text-muted mt-1">${parseFloat(overview?.commissions?.confirmed_amount ?? "0").toFixed(2)} confirmed</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2"><span className="text-sm text-pat-text-secondary">Connected Agents</span><IconActivity size={18} className="text-cyan-400" /></div>
          <div className="text-2xl font-bold text-pat-text-primary">{Number(agents?.agents_connected ?? 0)}</div>
          <div className="text-xs text-pat-text-muted mt-1">Windows Agent connections</div>
        </div>
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Signal Statistics</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div><div className="text-xs text-pat-text-muted">BUY Signals</div><div className="text-lg font-semibold text-pat-success">—</div></div>
          <div><div className="text-xs text-pat-text-muted">SELL Signals</div><div className="text-lg font-semibold text-pat-danger">—</div></div>
          <div><div className="text-xs text-pat-text-muted">NO-TRADE</div><div className="text-lg font-semibold text-pat-text-secondary">—</div></div>
          <div><div className="text-xs text-pat-text-muted">Blocked</div><div className="text-lg font-semibold text-pat-warning">—</div></div>
        </div>
        <p className="text-xs text-pat-text-muted mt-3">Signal statistics require the Go realtime engine database to be populated. Use the Signal Panel for live signal data.</p>
      </div>
    </div>
  );
}
