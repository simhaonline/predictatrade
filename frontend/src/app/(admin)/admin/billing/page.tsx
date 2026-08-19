"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";

interface AdminSub {
  id: string;
  user_email: string;
  plan_name: string;
  status: string;
  billing_cycle: string;
  current_period_start: string;
  current_period_end: string;
}

interface AdminCommission {
  id: string;
  recipient_email: string;
  source_email: string;
  commission_amount: string;
  commission_level: number;
  status: string;
  created_at: string;
}

interface AdminPayout {
  id: string;
  user_email: string;
  amount: string;
  status: string;
  created_at: string;
  approved_at: string | null;
}

export default function AdminBillingPage() {
  const [tab, setTab] = useState<"subscriptions" | "commissions" | "payouts">("subscriptions");
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();

  const subsQ = useQuery({
    queryKey: ["admin-subs-billing", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/subscriptions?page=${page}&limit=20`);
      return res.data as { items: AdminSub[]; total: number };
    },
    enabled: tab === "subscriptions",
  });

  const commissionsQ = useQuery({
    queryKey: ["admin-commissions-billing", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/commissions?page=${page}&limit=20`);
      return res.data as { items: AdminCommission[]; total: number };
    },
    enabled: tab === "commissions",
  });

  const payoutsQ = useQuery({
    queryKey: ["admin-payouts-billing", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/payouts?page=${page}&limit=20`);
      return res.data as { items: AdminPayout[]; total: number };
    },
    enabled: tab === "payouts",
  });

  const approvePayout = useMutation({
    mutationFn: async (payoutId: string) => {
      await customInstance.post(`/payouts/${payoutId}/approve`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-payouts-billing"] });
      toast.success("Payout approved");
    },
    onError: (err: unknown) => {
      toast.error(err instanceof Error ? err.message : "Failed to approve payout");
    },
  });

  const subCols: DataTableColumn<AdminSub>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-sm text-pat-text-primary">{row.plan_name || "—"}</span> },
    { key: "billing_cycle", header: "Cycle", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.billing_cycle || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "current_period_start", header: "Period Start", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_start ? format(new Date(row.current_period_start), "MMM d, yyyy") : "—"}</span> },
    { key: "current_period_end", header: "Period End", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_end ? format(new Date(row.current_period_end), "MMM d, yyyy") : "—"}</span> },
  ];

  const commissionCols: DataTableColumn<AdminCommission>[] = [
    { key: "recipient_email", header: "Recipient", cell: (row) => <span className="text-sm text-pat-text-primary">{row.recipient_email || "—"}</span> },
    { key: "source_email", header: "Source", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.source_email || "—"}</span> },
    { key: "commission_amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${parseFloat(row.commission_amount || "0").toFixed(2)}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Created", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
  ];

  const payoutCols: DataTableColumn<AdminPayout>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${parseFloat(row.amount || "0").toFixed(2)}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Requested", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
    { key: "actions", header: "Actions", cell: (row) => (
      row.status === "PENDING" ? (
        <button onClick={() => approvePayout.mutate(row.id)} disabled={approvePayout.isPending}
          className="text-xs bg-pat-success/10 text-pat-success hover:bg-pat-success/20 px-2 py-1 rounded transition-colors disabled:opacity-50">
          {approvePayout.isPending ? "Approving..." : "Approve"}
        </button>
      ) : <span className="text-xs text-pat-text-muted">—</span>
    )},
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Billing & Payouts</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Manage subscriptions, commissions, and payout approvals.</p>
      </div>
      <div className="flex gap-2">
        {(["subscriptions", "commissions", "payouts"] as const).map((t) => (
          <button key={t} onClick={() => { setTab(t); setPage(1); }} className={`text-xs px-3 py-1.5 rounded transition-colors capitalize ${tab === t ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {t}
          </button>
        ))}
      </div>

      {tab === "subscriptions" && (
        <DataTable data={subsQ.data?.items ?? []} columns={subCols} loading={subsQ.isLoading} error={subsQ.error as Error | null} onRetry={() => subsQ.refetch()} />
      )}
      {tab === "commissions" && (
        <DataTable data={commissionsQ.data?.items ?? []} columns={commissionCols} loading={commissionsQ.isLoading} error={commissionsQ.error as Error | null} onRetry={() => commissionsQ.refetch()} />
      )}
      {tab === "payouts" && (
        <DataTable data={payoutsQ.data?.items ?? []} columns={payoutCols} loading={payoutsQ.isLoading} error={payoutsQ.error as Error | null} onRetry={() => payoutsQ.refetch()} />
      )}

      {((tab === "subscriptions" ? subsQ.data?.total : tab === "commissions" ? commissionsQ.data?.total : payoutsQ.data?.total) ?? 0) > 20 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-secondary">Page {page}</span>
          <button onClick={() => setPage((p) => p + 1)} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded">Next</button>
        </div>
      )}
    </div>
  );
}
