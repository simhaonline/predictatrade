"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { approvePayout } from "@/lib/admin-api";
import { exportRowsToCsv } from "@/lib/admin-commercial-api";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import { IconDownload, IconAlertTriangle, IconEye } from "@tabler/icons-react";

interface AdminPayout {
  id: string;
  user_id?: string;
  user_email: string;
  amount: string | number;
  status: string;
  method?: string;
  destination?: string;
  created_at: string;
  approved_at: string | null;
  processed_at?: string | null;
  notes?: string | null;
}

const PENDING_OPS = ["Reject", "Process", "Reconcile", "Retry", "Cancel"] as const;

export default function AdminPayoutOperationsPage() {
  const [page, setPage] = useState(1);
  const [review, setReview] = useState<AdminPayout | null>(null);
  const queryClient = useQueryClient();

  const payoutsQ = useQuery<{ items: AdminPayout[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-payouts-ops", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/payouts?page=${page}&limit=20`);
      return res.data as { items: AdminPayout[]; total: number; page: number; limit: number };
    },
  });

  const statsQ = useQuery<{ total?: string | number; pending?: string | number; approved?: string | number; rejected?: string | number; pending_amount?: string | number; approved_amount?: string | number }>({
    queryKey: ["admin-payout-stats-ops"],
    queryFn: async () => (await customInstance.get("/admin/payouts/stats")).data,
  });

  const approveMutation = useMutation({
    mutationFn: async (id: string) => approvePayout(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-payouts-ops"] });
      queryClient.invalidateQueries({ queryKey: ["admin-payout-stats-ops"] });
      toast.success("Payout approved");
      setReview(null);
    },
    onError: (err: unknown) => toast.error(err instanceof Error ? err.message : "Failed to approve payout"),
  });

  const pendingOp = (label: string) => toast.error(`${label} endpoint pending backend — operation unavailable`);

  const cols: DataTableColumn<AdminPayout>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${Number(row.amount || 0).toFixed(2)}</span> },
    { key: "method", header: "Method", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.method || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Requested", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
    { key: "actions", header: "Actions", cell: (row) => (
      <div className="flex items-center gap-2">
        <button onClick={() => setReview(row)} className="flex items-center gap-1 text-xs bg-pat-bg-surface-secondary text-pat-text-primary px-2 py-1 rounded hover:bg-pat-bg-surface-secondary transition-colors">
          <IconEye size={12} /> Review
        </button>
        {row.status === "PENDING" && (
          <button onClick={() => approveMutation.mutate(row.id)} disabled={approveMutation.isPending} className="text-xs bg-pat-success/10 text-pat-success hover:bg-pat-success/20 px-2 py-1 rounded transition-colors disabled:opacity-50">
            {approveMutation.isPending ? "Approving..." : "Approve"}
          </button>
        )}
      </div>
    )},
  ];

  const totalPages = payoutsQ.data?.total ? Math.ceil(payoutsQ.data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Payout Operations</h1>
          <p className="text-sm text-pat-text-secondary mt-1">Review, approve, and reconcile affiliate payouts.</p>
        </div>
        <button onClick={() => exportRowsToCsv((payoutsQ.data?.items ?? []) as unknown as Record<string, unknown>[], "payouts.csv")} className="flex items-center gap-1 text-xs bg-pat-bg-surface-secondary text-pat-text-primary px-3 py-1.5 rounded hover:bg-pat-bg-surface-secondary transition-colors">
          <IconDownload size={14} /> Export CSV
        </button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted">Pending</div>
          <div className="text-2xl font-bold text-pat-text-primary">{statsQ.data?.pending ?? "—"}</div>
          <div className="text-xs text-pat-warning mt-1">${Number(statsQ.data?.pending_amount ?? 0).toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted">Approved</div>
          <div className="text-2xl font-bold text-pat-text-primary">{statsQ.data?.approved ?? "—"}</div>
          <div className="text-xs text-pat-success mt-1">${Number(statsQ.data?.approved_amount ?? 0).toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted">Rejected</div>
          <div className="text-2xl font-bold text-pat-text-primary">{statsQ.data?.rejected ?? "—"}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted">Total</div>
          <div className="text-2xl font-bold text-pat-text-primary">{statsQ.data?.total ?? "—"}</div>
        </div>
      </div>

      {statsQ.isError && (
        <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
          <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
          <div className="text-xs text-pat-text-secondary">
            Degraded — payout stats endpoint (<code className="font-mono">GET /admin/payouts/stats</code>) returned an error or is pending. Summary cards reflect no data.
            <div className="mt-1 text-pat-text-muted">{(statsQ.error as Error).message}</div>
          </div>
        </div>
      )}

      {payoutsQ.isError && (
        <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
          <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
          <div className="text-xs text-pat-text-secondary">
            Degraded — payout list endpoint (<code className="font-mono">GET /admin/payouts</code>) returned an error or is pending.
            <div className="mt-1 text-pat-text-muted">{(payoutsQ.error as Error).message}</div>
          </div>
        </div>
      )}

      <DataTable data={payoutsQ.data?.items ?? []} columns={cols} loading={payoutsQ.isLoading} error={payoutsQ.error as Error | null} onRetry={() => payoutsQ.refetch()} />

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-secondary">Page {page} of {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
        </div>
      )}

      {review && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setReview(null)}>
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg shadow-xl max-w-md w-full mx-4 p-5" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-pat-text-primary mb-4">Review Payout</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between"><span className="text-pat-text-muted">User</span><span className="text-pat-text-primary">{review.user_email}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Amount</span><span className="text-pat-text-primary font-medium">${Number(review.amount || 0).toFixed(2)}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Status</span><StatusBadge status={review.status} /></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Method</span><span className="text-pat-text-secondary">{review.method || "—"}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Destination</span><span className="text-pat-text-secondary text-xs">{review.destination || "—"}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Requested</span><span className="text-pat-text-secondary">{review.created_at ? format(new Date(review.created_at), "MMM d, yyyy") : "—"}</span></div>
              {review.notes && <div className="flex justify-between"><span className="text-pat-text-muted">Notes</span><span className="text-pat-text-secondary text-xs">{review.notes}</span></div>}
            </div>

            <div className="mt-4 space-y-2">
              <button
                onClick={() => approveMutation.mutate(review.id)}
                disabled={approveMutation.isPending || review.status !== "PENDING"}
                className="w-full px-3 py-2 text-xs bg-pat-success text-white rounded-md hover:opacity-90 disabled:opacity-50 transition-opacity"
              >
                {approveMutation.isPending ? "Approving..." : "Approve Payout"}
              </button>
              <div className="flex flex-wrap gap-2">
                {PENDING_OPS.map((op) => (
                  <button key={op} onClick={() => pendingOp(op)} disabled title="Backend endpoint pending" className="flex-1 px-2 py-1.5 text-[11px] rounded-md bg-pat-bg-surface-secondary text-pat-text-muted cursor-not-allowed">
                    {op}
                  </button>
                ))}
              </div>
              <p className="text-[11px] text-pat-text-muted">Only Approve is wired (<code className="font-mono">POST /payouts/:id/approve</code>). Reject / Process / Reconcile / Retry / Cancel are pending backend endpoints and are disabled.</p>
            </div>

            <div className="flex justify-end mt-3">
              <button onClick={() => setReview(null)} className="px-3 py-1.5 text-xs border border-pat-border-strong rounded-md text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors">Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
