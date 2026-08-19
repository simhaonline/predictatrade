"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";

interface DeviceActivation {
  client_type: string;
  broker_name: string;
  mt_account_login: string;
  activated_at: string;
}

interface Device {
  id: string;
  user_id: string;
  user_email: string;
  device_name: string;
  os: string;
  agent_version: string;
  hostname: string;
  status: string;
  registered_at: string;
  last_seen_at: string | null;
  bound_license_id: string | null;
  license_key: string | null;
  license_status: string | null;
  installation_id: string | null;
  revoked_at: string | null;
  revocation_reason: string | null;
  security_state: string;
  activations: DeviceActivation[] | null;
}

export default function AdminDeviceAuthPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, error, refetch } = useQuery<{ items: Device[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-devices", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/devices?page=${page}&limit=20`);
      return res.data as { items: Device[]; total: number; page: number; limit: number };
    },
  });

  const columns: DataTableColumn<Device>[] = [
    { key: "device_name", header: "Device", cell: (row) => (
      <div>
        <div className="text-sm text-pat-text-primary">{row.device_name || "—"}</div>
        {row.hostname && <div className="text-xs text-pat-text-muted">{row.hostname}</div>}
      </div>
    )},
    { key: "user_email", header: "User", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.user_email || "—"}</span> },
    { key: "license_key", header: "License", cell: (row) => <span className="text-xs text-pat-text-muted font-mono">{row.license_key ? row.license_key.slice(0, 20) + "..." : "—"}</span> },
    { key: "activations", header: "Terminals", cell: (row) => (
      <div className="flex flex-wrap gap-1">
        {row.activations && row.activations.length > 0 ? row.activations.map((a, i) => (
          <span key={i} className="text-[10px] px-1.5 py-0.5 rounded bg-pat-bg-surface-secondary text-pat-text-secondary">{a.client_type}</span>
        )) : <span className="text-xs text-pat-text-muted">—</span>}
      </div>
    )},
    { key: "os", header: "OS", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.os || "—"}</span> },
    { key: "status", header: "Connection", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "last_seen_at", header: "Last Seen", cell: (row) => <span className="text-xs text-pat-text-muted">{row.last_seen_at ? format(new Date(row.last_seen_at), "MMM d, yyyy HH:mm") : "—"}</span> },
    { key: "revoked_at", header: "Revoked", cell: (row) => row.revoked_at ? (
      <span className="text-xs text-pat-danger">{format(new Date(row.revoked_at), "MMM d, yyyy")}</span>
    ) : <span className="text-xs text-pat-text-muted">—</span> },
  ];

  const totalPages = data?.total ? Math.ceil(data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Device Auth</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Manage registered devices, activations, and heartbeat state.</p>
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
