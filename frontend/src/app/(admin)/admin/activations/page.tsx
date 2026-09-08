"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import { IconActivity, IconHistory } from "@tabler/icons-react";

interface Activation {
  id: string;
  license_id: string;
  license_key: string;
  device_id: string;
  user_email: string;
  device_name: string;
  client_type: string;
  terminal_build: string | null;
  ea_version: string | null;
  broker_name: string | null;
  broker_server: string | null;
  mt_account_login: string | null;
  installation_id: string | null;
  activated_at: string;
  created_at: string;
  connection_status: string;
  last_seen_at: string | null;
  terminal_connected?: boolean;
  last_account_update?: string | null;
  hostname: string | null;
}

type Scope = "live" | "recent" | "history";

export default function AdminActivationsPage() {
  const queryClient = useQueryClient();
  const [scope, setScope] = useState<Scope>("live");
  const [page, setPage] = useState(1);
  const limit = 20;

  const { data, isLoading, error, refetch } = useQuery<{ items: Activation[]; total: number; page: number; limit: number; scope: string }>({
    queryKey: ["admin-activations", scope, page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/activations?page=${page}&limit=${limit}&scope=${scope}`);
      return res.data as { items: Activation[]; total: number; page: number; limit: number; scope: string };
    },
    refetchInterval: scope === "live" ? 15000 : scope === "recent" ? 60000 : false,
  });

  const revokeMutation = useMutation({
    mutationFn: async (deviceId: string) => {
      await customInstance.post(`/licensing/devices/${deviceId}/revoke`, { reason: "admin_revoke" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-device-sessions"] });
      toast.success("Device revoked");
    },
    onError: () => toast.error("Failed to revoke device"),
  });

  const columns: DataTableColumn<Activation>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "license_key", header: "License", cell: (row) => <span className="text-xs text-pat-text-muted font-mono">{row.license_key ? row.license_key.slice(0, 20) + "..." : "—"}</span> },
    { key: "device_name", header: "Device", cell: (row) => (
      <div>
        <div className="text-sm text-pat-text-primary">{row.device_name || "—"}</div>
        {row.hostname && <div className="text-xs text-pat-text-muted">{row.hostname}</div>}
      </div>
    )},
    { key: "client_type", header: "Terminal", cell: (row) => (
      <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${row.client_type === "MT5" ? "bg-pat-info/10 text-pat-info" : "bg-pat-badge-neutral-bg/10 text-pat-badge-neutral-text"}`}>{row.client_type}</span>
    )},
    { key: "broker_name", header: "Broker", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.broker_name || "—"}</span> },
    { key: "mt_account_login", header: "Account", cell: (row) => <span className="text-xs text-pat-text-muted font-mono">{row.mt_account_login || "—"}</span> },
    ...(scope === "live"
      ? [
          { key: "status", header: "Status", sortable: true, cell: (row: Activation) => {
            const seen = row.last_seen_at ? new Date(row.last_seen_at).getTime() : 0;
            const mins = Math.floor((Date.now() - seen) / 60000);
            if (mins < 5) return <span className="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-pat-success/10 text-pat-success border border-pat-success/20"><span className="inline-block h-1.5 w-1.5 rounded-full bg-pat-success animate-pulse" />LIVE</span>;
            if (mins < 60) return <span className="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-pat-warning/10 text-pat-warning border border-pat-warning/20">IDLE {mins}m</span>;
            return <span className="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-pat-badge-neutral-bg/10 text-pat-badge-neutral-text border border-pat-border">OFFLINE</span>;
          } },
          { key: "last_seen_at", header: "Last Poll", cell: (row: Activation) => {
            const seen = row.last_seen_at ? new Date(row.last_seen_at).getTime() : 0;
            const mins = Math.floor((Date.now() - seen) / 60000);
            const label = mins < 1 ? "just now" : mins < 60 ? `${mins}m ago` : `${Math.floor(mins / 60)}h ${mins % 60}m ago`;
            return <span className="text-xs text-pat-text-muted">{label}</span>;
          } },
        ]
      : scope === "recent"
        ? [
            { key: "status", header: "Last Poll", sortable: true, cell: (row: Activation) => {
              const seen = row.last_seen_at ? new Date(row.last_seen_at).getTime() : 0;
              const mins = Math.floor((Date.now() - seen) / 60000);
              const label = mins < 60 ? `${mins}m ago` : `${Math.floor(mins / 60)}h ${mins % 60}m ago`;
              return <span className="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-pat-warning/10 text-pat-warning border border-pat-warning/20">{label}</span>;
            } },
          ]
        : [
            { key: "connection_status", header: "Connection", cell: (row: Activation) => <StatusBadge status={row.connection_status} /> },
          ]),
    { key: "activated_at", header: scope === "live" ? "Since" : "Activated", cell: (row) => <span className="text-xs text-pat-text-muted">{row.activated_at ? format(new Date(row.activated_at), "MMM d, yyyy HH:mm") : "—"}</span> },
  ];

  const totalPages = data?.total ? Math.ceil(data.total / limit) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Activations</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          {scope === "live"
            ? "Terminals currently running (polled within 5 min) — one row per terminal, auto-refreshed every 15s. IDLE = open but not polling (market closed / no ticks)."
            : scope === "recent"
              ? "Terminals active in the last 24 hours but not polling right now — clients whose MT terminal is closed or idle, with how long ago they were last seen."
              : "Full activation history — every connect and disconnect event, newest first. Rows are never deleted."}
        </p>
      </div>

      {/* Scope tabs */}
      <div className="flex items-center gap-2">
        <button
          onClick={() => { setScope("live"); setPage(1); }}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg border transition-colors ${
            scope === "live"
              ? "bg-pat-success/10 text-pat-success border-pat-success/30 font-medium"
              : "bg-pat-bg-surface-secondary text-pat-text-secondary border-pat-border hover:text-pat-text-primary"
          }`}
        >
          <IconActivity size={14} /> Live Connections {data?.scope === "live" && data.total > 0 && <span className="px-1.5 py-0.5 rounded bg-pat-success/20">{data.total}</span>}
        </button>
        <button
          onClick={() => { setScope("recent"); setPage(1); }}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg border transition-colors ${
            scope === "recent"
              ? "bg-pat-warning/10 text-pat-warning border-pat-warning/30 font-medium"
              : "bg-pat-bg-surface-secondary text-pat-text-secondary border-pat-border hover:text-pat-text-primary"
          }`}
        >
          Recent (24h)
        </button>
        <button
          onClick={() => { setScope("history"); setPage(1); }}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg border transition-colors ${
            scope === "history"
              ? "bg-pat-info/10 text-pat-info border-pat-info/30 font-medium"
              : "bg-pat-bg-surface-secondary text-pat-text-secondary border-pat-border hover:text-pat-text-primary"
          }`}
        >
          <IconHistory size={14} /> Connection History
        </button>
      </div>

      <DataTable
        key={scope}
        data={data?.items || []}
        columns={columns}
        loading={isLoading}
        error={error as Error | null}
        onRetry={refetch}
        pageSize={limit}
        hidePager
      />

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded border border-pat-border disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-muted">Page {page} of {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded border border-pat-border disabled:opacity-30">Next</button>
        </div>
      )}
    </div>
  );
}