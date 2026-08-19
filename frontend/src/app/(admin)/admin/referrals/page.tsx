"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { IconCoin, IconUsers } from "@tabler/icons-react";

interface AdminCommission {
  id: string;
  recipient_user_id: string;
  recipient_email: string;
  source_user_id: string;
  source_email: string;
  commission_amount: string;
  commission_level: number;
  status: string;
  created_at: string;
}

interface CommissionSummary {
  total_entries: string;
  total_amount: string;
  pending_count: string;
  pending_amount: string;
  confirmed_count: string;
  confirmed_amount: string;
  reversed_count: string;
  reversed_amount: string;
}

interface AdminPayout {
  id: string;
  user_id: string;
  user_email: string;
  amount: string;
  status: string;
  created_at: string;
  approved_at: string | null;
}

export default function AdminReferralsPage() {
  const [tab, setTab] = useState<"commissions" | "payouts" | "summary">("commissions");
  const [page, setPage] = useState(1);

  const commissionsQ = useQuery({
    queryKey: ["admin-commissions", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/commissions?page=${page}&limit=20`);
      return res.data as { items: AdminCommission[]; total: number; page: number; limit: number };
    },
  });

  const summaryQ = useQuery<CommissionSummary>({
    queryKey: ["admin-commission-summary"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/commissions/summary");
      return res.data as CommissionSummary;
    },
  });

  const payoutsQ = useQuery({
    queryKey: ["admin-payouts", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/payouts?page=${page}&limit=20`);
      return res.data as { items: AdminPayout[]; total: number; page: number; limit: number };
    },
  });

  const payoutStatsQ = useQuery({
    queryKey: ["admin-payout-stats"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/payouts/stats");
      return res.data as { total: string; pending: string; approved: string; rejected: string; pending_amount: string; approved_amount: string };
    },
  });

  const commissionCols: DataTableColumn<AdminCommission>[] = [
    { key: "recipient_email", header: "Recipient", cell: (row) => <span className="text-sm text-pat-text-primary">{row.recipient_email || "—"}</span> },
    { key: "source_email", header: "Source", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.source_email || "—"}</span> },
    { key: "commission_amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${parseFloat(row.commission_amount || "0").toFixed(2)}</span> },
    { key: "commission_level", header: "Level", cell: (row) => <span className="text-xs text-pat-text-secondary">L{row.commission_level}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Date", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
  ];

  const payoutCols: DataTableColumn<AdminPayout>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${parseFloat(row.amount || "0").toFixed(2)}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Requested", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
    { key: "approved_at", header: "Approved", cell: (row) => <span className="text-xs text-pat-text-muted">{row.approved_at ? format(new Date(row.approved_at), "MMM d, yyyy") : "—"}</span> },
  ];

  const totalPages = (tab === "commissions" ? commissionsQ.data?.total : payoutsQ.data?.total) ?? 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Referrals & Commissions</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Platform-wide referral relationships, commission ledger, and payout management.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Total Commissions</span>
            <IconCoin size={18} className="text-pat-session" />
          </div>
          <div className="text-2xl font-bold text-pat-text-primary">{summaryQ.data?.total_entries ?? "—"}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Confirmed</span>
            <IconCoin size={18} className="text-pat-success" />
          </div>
          <div className="text-2xl font-bold text-pat-text-primary">${parseFloat(summaryQ.data?.confirmed_amount ?? "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Pending</span>
            <IconCoin size={18} className="text-pat-warning" />
          </div>
          <div className="text-2xl font-bold text-pat-text-primary">${parseFloat(summaryQ.data?.pending_amount ?? "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Payouts Pending</span>
            <IconUsers size={18} className="text-pat-badge-neutral-text" />
          </div>
          <div className="text-2xl font-bold text-pat-text-primary">{payoutStatsQ.data?.pending ?? "—"}</div>
        </div>
      </div>

      <div className="flex gap-2">
        {(["commissions", "payouts", "summary"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`text-xs px-3 py-1.5 rounded transition-colors capitalize ${tab === t ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {t === "summary" ? "Summary Details" : t}
          </button>
        ))}
      </div>

      {tab === "commissions" && (
        <>
          <DataTable data={commissionsQ.data?.items ?? []} columns={commissionCols} loading={commissionsQ.isLoading} error={commissionsQ.error as Error | null} onRetry={() => commissionsQ.refetch()} />
          {totalPages > 20 && (
            <div className="flex items-center justify-center gap-2 pt-2">
              <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
              <span className="text-xs text-pat-text-secondary">Page {page} of {Math.ceil(totalPages / 20)}</span>
              <button onClick={() => setPage((p) => p + 1)} disabled={page >= Math.ceil(totalPages / 20)} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
            </div>
          )}
        </>
      )}

      {tab === "payouts" && (
        <>
          <DataTable data={payoutsQ.data?.items ?? []} columns={payoutCols} loading={payoutsQ.isLoading} error={payoutsQ.error as Error | null} onRetry={() => payoutsQ.refetch()} />
          {totalPages > 20 && (
            <div className="flex items-center justify-center gap-2 pt-2">
              <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
              <span className="text-xs text-pat-text-secondary">Page {page} of {Math.ceil(totalPages / 20)}</span>
              <button onClick={() => setPage((p) => p + 1)} disabled={page >= Math.ceil(totalPages / 20)} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
            </div>
          )}
        </>
      )}

      {tab === "summary" && summaryQ.data && (
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-6">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Commission Summary</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div><div className="text-xs text-pat-text-muted">Total Entries</div><div className="text-lg font-semibold text-pat-text-primary">{summaryQ.data.total_entries}</div></div>
            <div><div className="text-xs text-pat-text-muted">Total Amount</div><div className="text-lg font-semibold text-pat-text-primary">${parseFloat(summaryQ.data.total_amount).toFixed(2)}</div></div>
            <div><div className="text-xs text-pat-text-muted">Pending Count</div><div className="text-lg font-semibold text-pat-text-primary">{summaryQ.data.pending_count}</div></div>
            <div><div className="text-xs text-pat-text-muted">Pending Amount</div><div className="text-lg font-semibold text-pat-warning">${parseFloat(summaryQ.data.pending_amount).toFixed(2)}</div></div>
            <div><div className="text-xs text-pat-text-muted">Confirmed Count</div><div className="text-lg font-semibold text-pat-success">{summaryQ.data.confirmed_count}</div></div>
            <div><div className="text-xs text-pat-text-muted">Confirmed Amount</div><div className="text-lg font-semibold text-pat-success">${parseFloat(summaryQ.data.confirmed_amount).toFixed(2)}</div></div>
            <div><div className="text-xs text-pat-text-muted">Reversed Count</div><div className="text-lg font-semibold text-pat-danger">{summaryQ.data.reversed_count}</div></div>
            <div><div className="text-xs text-pat-text-muted">Reversed Amount</div><div className="text-lg font-semibold text-pat-danger">${parseFloat(summaryQ.data.reversed_amount).toFixed(2)}</div></div>
          </div>
        </div>
      )}
    </div>
  );
}
