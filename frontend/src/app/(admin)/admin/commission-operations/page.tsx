"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import {
  fetchCommissionsAdminAll,
  exportRowsToCsv,
  holdCommission,
  releaseCommission,
  reverseCommission,
  adjustCommission,
  clearEligibleCommissions,
} from "@/lib/admin-commercial-api";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import { IconDownload, IconAlertTriangle, IconCoin, IconCheck } from "@tabler/icons-react";

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

const OPS = ["Hold", "Release", "Reverse", "Adjust"] as const;
type Op = (typeof OPS)[number];

export default function AdminCommissionOperationsPage() {
  const [page, setPage] = useState(1);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

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
    {
      key: "select",
      header: "Action",
      cell: (row) => (
        <button
          onClick={(e) => { e.stopPropagation(); setSelectedId(row.id); }}
          className={`px-2 py-1 text-xs rounded-md ${selectedId === row.id ? "bg-pat-primary text-white" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}
        >
          {selectedId === row.id ? "Selected" : "Select"}
        </button>
      ),
    },
  ];

  const runOp = async (op: Op) => {
    if (!selectedId) {
      toast.error("Select a commission row first");
      return;
    }
    const reason = window.prompt(`Reason for ${op}:`);
    if (reason === null) return;
    const finalReason = reason.trim() || "Admin operation";
    try {
      setBusy(true);
      if (op === "Hold") {
        await holdCommission(selectedId, finalReason);
      } else if (op === "Release") {
        await releaseCommission(selectedId, finalReason);
      } else if (op === "Reverse") {
        const a = window.prompt("Reversal amount (optional, full commission if blank):");
        if (a === null) return;
        const amt = a.trim() ? Number(a) : undefined;
        await reverseCommission(selectedId, finalReason, amt);
      } else if (op === "Adjust") {
        const a = window.prompt("Adjustment amount (signed, e.g. -5 or 5):");
        if (a === null) return;
        const amt = Number(a);
        if (Number.isNaN(amt)) {
          toast.error("Invalid adjustment amount");
          return;
        }
        await adjustCommission(selectedId, amt, finalReason);
      }
      toast.success(`${op} applied`);
      setSelectedId(null);
      ledgerQ.refetch();
    } catch (e) {
      toast.error((e as Error).message || `${op} failed`);
    } finally {
      setBusy(false);
    }
  };

  const runClearEligible = async () => {
    const confirm = window.confirm(
      "Run clear-eligible? This moves PENDING→CLEARED (>=14d) and CLEARED→AVAILABLE (>=30d) in bulk.",
    );
    if (!confirm) return;
    try {
      setBusy(true);
      const res = await clearEligibleCommissions();
      toast.success(`Cleared ${res.cleared ?? 0}, made available ${res.available ?? 0}`);
      ledgerQ.refetch();
    } catch (e) {
      toast.error((e as Error).message || "clear-eligible failed");
    } finally {
      setBusy(false);
    }
  };

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
          <span className="text-[11px] text-pat-text-muted">
            {selectedId ? `Selected: ${selectedId.slice(0, 8)}…` : "No row selected"}
          </span>
        </div>
        <div className="flex flex-wrap gap-2">
          {OPS.map((op) => (
            <button
              key={op}
              onClick={() => runOp(op)}
              disabled={busy || !selectedId}
              className="px-3 py-1.5 text-xs rounded-md bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary disabled:opacity-40 disabled:cursor-not-allowed"
              title={selectedId ? `Apply ${op}` : "Select a row first"}
            >
              {op}
            </button>
          ))}
          <button
            onClick={runClearEligible}
            disabled={busy}
            className="px-3 py-1.5 text-xs rounded-md bg-pat-info/10 text-pat-info hover:bg-pat-info/20 disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-1"
          >
            <IconCheck size={14} /> Clear Eligible
          </button>
        </div>
        <p className="text-[11px] text-pat-text-muted mt-2">
          Select a ledger row, then apply a lifecycle operation. All operations are auditable and move amounts between wallet buckets (pending → cleared → available → paid, or hold/reverse). Clear Eligible is admin-bulk and not auto-run.
        </p>
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
