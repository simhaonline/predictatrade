"use client";
import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import { format } from "date-fns";
import { IconSearch } from "@tabler/icons-react";

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

const FILTER_FIELDS = ["actor", "action", "entity", "ip", "state", "reason"] as const;

export default function AdminLogsPage() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [field, setField] = useState<typeof FILTER_FIELDS[number]>("action");

  const { data, isLoading, error, refetch } = useQuery<{ items: AuditLog[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-audit", page],
    queryFn: async () => {
      const res = await customInstance.get(`/audit?page=${page}&limit=20`);
      return res.data as { items: AuditLog[]; total: number; page: number; limit: number };
    },
  });

  const filtered = useMemo(() => {
    const rows = data?.items ?? [];
    if (!search.trim()) return rows;
    const q = search.toLowerCase();
    return rows.filter((r) => {
      switch (field) {
        case "actor": return (r.actor_type ?? "").toLowerCase().includes(q);
        case "action": return (r.event_type ?? "").toLowerCase().includes(q);
        case "entity": return `${r.entity_type ?? ""} ${r.entity_id ?? ""}`.toLowerCase().includes(q);
        case "ip": return (r.source_ip ?? "").toLowerCase().includes(q);
        case "state": return String(r.metadata?.state ?? "").toLowerCase().includes(q);
        case "reason": return String(r.metadata?.reason ?? "").toLowerCase().includes(q);
        default: return true;
      }
    });
  }, [data, search, field]);

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

      <div className="flex flex-wrap items-center gap-2 rounded-lg border border-pat-border bg-pat-bg-surface p-3">
        <div className="relative flex-1 min-w-[200px]">
          <IconSearch size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-pat-text-muted" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={`Filter by ${field}…`}
            className="w-full rounded-md border border-pat-input-border bg-pat-input-bg pl-9 pr-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary"
          />
        </div>
        <select
          value={field}
          onChange={(e) => setField(e.target.value as typeof field)}
          className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary"
        >
          {FILTER_FIELDS.map((f) => <option key={f} value={f} className="capitalize">{f}</option>)}
        </select>
      </div>
      <p className="text-[11px] text-pat-text-muted">Client-side filtering over the currently loaded page. Server-side search/filter is pending backend support — full-history search is not yet available.</p>

      <DataTable data={filtered} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
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
