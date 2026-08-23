"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { fetchCommissionsAdminAll, exportRowsToCsv } from "@/lib/admin-commercial-api";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import { IconDownload, IconAlertTriangle, IconCoin } from "@tabler/icons-react";

interface CommissionLedgerRow {
  id: string;
  recipient_email?: string;
  source_email?: string;
  commission_amount?: string | number;
  commission_level?: number;
  status?: string;
  created_at?: string;
  plan_code?: string;
  billing_cycle?: string;
}

const PENDING_OPS: { label: string }[] = [
  { label: "Hold" },
  { label: "Release" },
  { label: "Reverse" },
  { label: "Adjust" },
];

export default function AdminCommissionOperationsPage() {
  const [page, setPage] = useState(1);

  const ledgerQ = useQuery<{ items: CommissionLedgerRow[]; total: number; page: number; limit: number }>({
    queryKey: ["commission-ledger-admin", page],
    queryFn: async () => {
      const res = await customInstance.get(`/commissions/admin/all?page=${page}&limit=20`);
      return res.data as { items: CommissionLedgerRow[]; total: number; page: number; limit: number };
    },
  });

  const cols: DataTableColumn<CommissionLedgerRow>[] = [
    { key: "recipient_email", header: "Recipient", cell: (row) => <span className="text-sm text-pat-text-primary">{row.recipient_email || "—"}</span> },
    { key: "source_email", header: "Source", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.source_email || "—"}</span> },
    { key: "commission_amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${Number(row.commission_amount || 0).toFixed(2)}</span> },
    { key: "commission_level", header: "Level", cell: (row) => <span className="text-xs text-pat-text-secondary">L{row.commission_level ?? "—"}</span> },
    { key: "plan_code", header: "Plan", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.plan_code || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status ?? "unknown"} /> },
    { key: "created_at", header: "Created", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
  ];

  const pendingOp = (label: string) => toast.error(`${label} operation pending backend — no commission operation endpoint available`);

  const totalPages = ledgerQ.data?.total ? Math.ceil(ledgerQ.data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Commission Operations</h1>
          <p className="text-sm text-pat-text-secondary mt-1">Commission ledger and lifecycle operations (hold / release / reverse / adjust).</p>
        </div>
        <button onClick={() => exportRowsToCsv((ledgerQ.data?.items ?? []) as unknown as Record<string, unknown>[], "commission-ledger.csv")} className="flex items-center gap-1 text-xs bg-pat-bg-surface-secondary text-pat-text-primary px-3 py-1.5 rounded hover:bg-pat-bg-surface-secondary transition-colors">
          <IconDownload size={14} /> Export CSV
        </button>
      </div>

      {ledgerQ.isError && (
        <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
          <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
          <div className="text-xs text-pat-text-secondary">
            Degraded — commission ledger endpoint (<code className="font-mono">GET /commissions/admin/all</code>) returned an error or is pending. No ledger data shown.
            <div className="mt-1 text-pat-text-muted">{(ledgerQ.error as Error).message}</div>
          </div>
        </div>
      )}

      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex items-center gap-2 mb-3">
          <IconCoin size={16} className="text-pat-info" />
          <h2 className="text-sm font-semibold text-pat-text-primary">Ledger Operations</h2>
        </div>
        <div className="flex flex-wrap gap-2">
          {PENDING_OPS.map((op) => (
            <button key={op.label} onClick={() => pendingOp(op.label)} disabled className="px-3 py-1.5 text-xs rounded-md bg-pat-bg-surface-secondary text-pat-text-muted cursor-not-allowed" title="Backend operation endpoint pending">
              {op.label} (pending backend)
            </button>
          ))}
        </div>
        <p className="text-[11px] text-pat-text-muted mt-2">These operations require backend commission lifecycle endpoints that are not yet available. They are disabled and non-functional to avoid fabricating financial state.</p>
      </div>

      <DataTable data={ledgerQ.data?.items ?? []} columns={cols} loading={ledgerQ.isLoading} error={ledgerQ.error as Error | null} onRetry={() => ledgerQ.refetch()} />

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-secondary">Page {page} of {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
        </div>
      )}
    </div>
  );
}
