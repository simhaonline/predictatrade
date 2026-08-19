"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";

interface License { id: string; user_id: string; user_email: string; key: string; plan_name: string; status: string; activated_at: string | null; expires_at: string | null; max_devices: number; max_mt_accounts: number; subscription_status: string | null; }

export default function AdminLicensesPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-licenses", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/licenses?page=${page}&limit=20`);
      return res.data as { items: License[]; total: number; page: number; limit: number };
    },
  });

  const columns: DataTableColumn<License>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "key", header: "License Key", cell: (row) => <span className="text-xs text-pat-text-muted font-mono">{row.key || "—"}</span> },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-sm text-pat-text-primary">{row.plan_name || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "max_devices", header: "Max Devices", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.max_devices ?? "—"}</span> },
    { key: "activated_at", header: "Issued", cell: (row) => <span className="text-xs text-pat-text-muted">{row.activated_at ? format(new Date(row.activated_at), "MMM d, yyyy") : "—"}</span> },
    { key: "expires_at", header: "Expires", cell: (row) => <span className="text-xs text-pat-text-muted">{row.expires_at ? format(new Date(row.expires_at), "MMM d, yyyy") : "—"}</span> },
  ];

  const totalPages = data?.total ? Math.ceil(data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">License Management</h1>
        <p className="text-sm text-pat-text-secondary mt-1">View and manage all platform licenses.</p>
      </div>
      <DataTable data={data?.items || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-secondary">Page {page} of {totalPages}</span>
          <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
        </div>
      )}
    </div>
  );
}
