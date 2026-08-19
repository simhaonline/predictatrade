"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import { format } from "date-fns";

interface AuditLog {
  id: string;
  event_id: string;
  actor_type: string;
  user_id: string | null;
  event_type: string;
  entity_type: string;
  entity_id: string | null;
  created_at: string;
  source_ip: string | null;
  metadata: Record<string, unknown>;
}

export default function AdminLogsPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, error, refetch } = useQuery<{ items: AuditLog[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-audit", page],
    queryFn: async () => {
      const res = await customInstance.get(`/audit?page=${page}&limit=20`);
      return res.data as { items: AuditLog[]; total: number; page: number; limit: number };
    },
  });

  const columns: DataTableColumn<AuditLog>[] = [
    { key: "event_type", header: "Event", cell: (row) => <span className="text-xs font-medium text-pat-text-primary">{row.event_type}</span> },
    { key: "actor_type", header: "Actor", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.actor_type}</span> },
    { key: "user_id", header: "User", cell: (row) => <span className="text-xs text-pat-text-muted">{row.user_id?.slice(0, 8) || "System"}</span> },
    { key: "entity_type", header: "Entity", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.entity_type || "—"}</span> },
    { key: "metadata", header: "Details", cell: (row) => {
      const md = row.metadata;
      const summary = String(md?.reason || md?.entity_type || "");
      return <span className="text-xs text-pat-text-muted font-mono truncate max-w-xs inline-block">{summary || JSON.stringify(md).slice(0, 60)}</span>;
    }},
    { key: "source_ip", header: "IP", cell: (row) => <span className="text-xs text-pat-text-muted font-mono">{row.source_ip || "—"}</span> },
    { key: "created_at", header: "Timestamp", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy HH:mm:ss") : "—"}</span> },
  ];

  const totalPages = data?.total ? Math.ceil(data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Logs & Audit</h1>
        <p className="text-sm text-pat-text-secondary mt-1">System audit logs and events.</p>
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
