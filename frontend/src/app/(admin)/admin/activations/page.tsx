"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";

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
  hostname: string | null;
}

export default function AdminActivationsPage() {
  const queryClient = useQueryClient();

  const { data, isLoading, error, refetch } = useQuery<{ items: Activation[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-activations"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/activations?page=1&limit=20");
      return res.data as { items: Activation[]; total: number; page: number; limit: number };
    },
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
    { key: "connection_status", header: "Connection", cell: (row) => <StatusBadge status={row.connection_status} /> },
    { key: "activated_at", header: "Activated", cell: (row) => <span className="text-xs text-pat-text-muted">{row.activated_at ? format(new Date(row.activated_at), "MMM d, yyyy HH:mm") : "—"}</span> },
    { key: "last_seen_at", header: "Last Seen", cell: (row) => <span className="text-xs text-pat-text-muted">{row.last_seen_at ? format(new Date(row.last_seen_at), "MMM d, yyyy HH:mm") : "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Activations</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Active device sessions and activation management.</p>
      </div>
      <DataTable data={data?.items || []} columns={columns} loading={isLoading} error={error as Error | null} onRetry={refetch} />
    </div>
  );
}
