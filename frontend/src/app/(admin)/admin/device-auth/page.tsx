"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { revokeDevice } from "@/lib/admin-api";
import { resetDevice, forceUpgradeDevice, disableDeviceSignal } from "@/lib/admin-commercial-api";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import {
  IconBrandWindows, IconBroadcast, IconServer,
  IconActivity, IconInfoCircle, IconAlertTriangle,
} from "@tabler/icons-react";

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
  const [selected, setSelected] = useState<Device | null>(null);
  const queryClient = useQueryClient();

  const deviceAction = useMutation({
    mutationFn: async (p: { action: string; fn: () => Promise<unknown> }) => { await p.fn(); },
    onSuccess: (_d, v) => {
      queryClient.invalidateQueries({ queryKey: ["admin-devices", page] });
      toast.success(`${v.action} succeeded`);
    },
    onError: (err: unknown, v) => {
      toast.error(`${v.action}: backend endpoint pending — ${err instanceof Error ? err.message : "not available"}`);
    },
  });

  const doDeviceAction = (action: string, fn: () => Promise<unknown>) => {
    toast.message(`${action} requested…`);
    deviceAction.mutate({ action, fn });
  };

  // Live Go engine agents status
  const { data: agentsStatus } = useQuery<{ agents_connected: number; master_node_connected: boolean; snapshot_count: number }>({
    queryKey: ["admin-device-auth-agents"],
    queryFn: async () => (await customInstance.get("/agents/status")).data,
    refetchInterval: 10000,
  });

  // Live Go engine market snapshot for connected terminal details
  const { data: liveSnapshot } = useQuery<{
    broker?: string; account?: string; node?: string; source?: string; symbol?: string;
    account_info?: { balance: number; equity: number; profit: number; currency: string; server: string; leverage: number };
    positions?: { total_positions: number; buy_count: number; sell_count: number; total_lots: number; floating_profit: number };
    symbol_info?: { digits: number; spread: number; contract_size: number; tick_value: number; tick_size: number };
  }>({
    queryKey: ["admin-device-auth-snapshot"],
    queryFn: async () => (await customInstance.get("/market/snapshot")).data,
    refetchInterval: 10000,
  });
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
    { key: "actions", header: "Actions", cell: (row) => (
      <button onClick={() => setSelected(row)} className="text-xs bg-pat-bg-surface-secondary text-pat-text-primary px-2 py-1 rounded hover:bg-pat-bg-surface-secondary transition-colors">
        Manage
      </button>
    )},
  ];

  const totalPages = data?.total ? Math.ceil(data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Device Auth</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Manage registered devices, activations, and heartbeat state.</p>
      </div>
      {/* Live Go Engine Connection Status */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-pat-text-primary">Live Engine Connections</h2>
          <div className="flex items-center gap-2">
            <span className={`inline-block h-2 w-2 rounded-full ${(agentsStatus?.agents_connected ?? 0) > 0 ? "bg-pat-success animate-pulse" : "bg-pat-danger"}`} />
            <span className="text-xs text-pat-text-secondary">{(agentsStatus?.agents_connected ?? 0) > 0 ? "Agents Connected" : "No Connections"}</span>
          </div>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="flex items-center gap-2 mb-1"><IconBrandWindows size={14} className="text-pat-info" /><span className="text-[10px] text-pat-text-muted uppercase">Agents</span></div>
            <div className="text-lg font-bold text-pat-text-primary tabular-nums">{agentsStatus?.agents_connected ?? 0}</div>
            <div className="text-[10px] text-pat-text-muted">Windows Agent(s)</div>
          </div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="flex items-center gap-2 mb-1"><IconBroadcast size={14} className={agentsStatus?.master_node_connected ? "text-pat-success" : "text-pat-danger"} /><span className="text-[10px] text-pat-text-muted uppercase">Master Node</span></div>
            <div className={`text-lg font-bold tabular-nums ${agentsStatus?.master_node_connected ? "text-pat-success" : "text-pat-danger"}`}>{agentsStatus?.master_node_connected ? "ONLINE" : "OFFLINE"}</div>
            <div className="text-[10px] text-pat-text-muted">MT5 Data Feed</div>
          </div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="flex items-center gap-2 mb-1"><IconServer size={14} className="text-pat-info" /><span className="text-[10px] text-pat-text-muted uppercase">Snapshots</span></div>
            <div className="text-lg font-bold text-pat-text-primary tabular-nums">{(agentsStatus?.snapshot_count ?? 0).toLocaleString()}</div>
            <div className="text-[10px] text-pat-text-muted">Total received</div>
          </div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="flex items-center gap-2 mb-1"><IconActivity size={14} className="text-pat-success" /><span className="text-[10px] text-pat-text-muted uppercase">Positions</span></div>
            <div className="text-lg font-bold text-pat-text-primary tabular-nums">{liveSnapshot?.positions?.total_positions ?? 0}</div>
            <div className="text-[10px] text-pat-text-muted">{liveSnapshot?.positions?.buy_count ?? 0} BUY · {liveSnapshot?.positions?.sell_count ?? 0} SELL</div>
          </div>
        </div>

        {/* Connected terminal details */}
        {liveSnapshot?.broker && (
          <div className="rounded-lg bg-pat-bg-surface-secondary/20 p-3 space-y-2">
            <div className="text-xs font-medium text-pat-text-primary mb-2">Connected Master Node Terminal</div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
              <div><span className="text-pat-text-muted">Broker:</span> <span className="text-pat-text-secondary">{liveSnapshot.broker}</span></div>
              <div><span className="text-pat-text-muted">Server:</span> <span className="text-pat-text-secondary">{liveSnapshot?.account_info?.server || "—"}</span></div>
              <div><span className="text-pat-text-muted">Symbol:</span> <span className="text-pat-text-secondary">{liveSnapshot?.symbol || "—"}</span></div>
              <div><span className="text-pat-text-muted">Leverage:</span> <span className="text-pat-text-secondary">1:{liveSnapshot?.account_info?.leverage || "—"}</span></div>
              <div><span className="text-pat-text-muted">Balance:</span> <span className="text-pat-text-secondary">${(liveSnapshot?.account_info?.balance ?? 0).toFixed(2)}</span></div>
              <div><span className="text-pat-text-muted">Equity:</span> <span className="text-pat-text-secondary">${(liveSnapshot?.account_info?.equity ?? 0).toFixed(2)}</span></div>
              <div><span className="text-pat-text-muted">Floating P/L:</span> <span className={(liveSnapshot?.account_info?.profit ?? 0) >= 0 ? "text-pat-success" : "text-pat-danger"}>${(liveSnapshot?.account_info?.profit ?? 0).toFixed(2)}</span></div>
              <div><span className="text-pat-text-muted">Source:</span> <span className="text-pat-text-secondary">{liveSnapshot?.source || "—"}</span></div>
            </div>
            {liveSnapshot?.symbol_info && (
              <div className="text-[10px] text-pat-text-muted pt-2 border-t border-pat-border/30">
                Contract: {liveSnapshot.symbol_info.contract_size} | Tick: ${liveSnapshot.symbol_info.tick_value}/{liveSnapshot.symbol_info.tick_size} | Spread: {liveSnapshot.symbol_info.spread} | Digits: {liveSnapshot.symbol_info.digits}
              </div>
            )}
          </div>
        )}

        {/* Info note */}
        {(agentsStatus?.agents_connected ?? 0) > 0 && (data?.total ?? 0) === 0 && (
          <div className="mt-3 rounded-lg border border-pat-info/20 bg-pat-info/5 p-3">
            <div className="flex items-start gap-2">
              <IconInfoCircle size={14} className="text-pat-info shrink-0 mt-0.5" />
              <div className="text-[11px] text-pat-text-muted leading-relaxed">
                {agentsStatus?.agents_connected} Windows Agent(s) are connected to the Go engine and sending live data,
                but no devices are registered in the licensing database yet. This means the agents connected to the
                real-time engine but have not completed device registration with the control plane. Device registration
                occurs when the Windows Agent sends its first heartbeat with a valid license key to the NestJS API.
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Registered Devices Table */}
      <div>
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Registered Devices (Licensing Database)</h2>

        <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 px-4 py-3 flex items-start gap-2 mb-3">
          <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
          <div className="text-xs text-pat-text-secondary">
            Device write actions: <strong>Revoke</strong> is wired via the licensing device-revoke endpoint. <strong>Reset / Force Upgrade / Disable Signal</strong> are pending backend endpoints and degrade honestly with a &quot;backend endpoint pending&quot; toast.
          </div>
        </div>

        <div className="flex flex-wrap gap-2 mb-3">
          <button onClick={() => selected ? doDeviceAction(`Revoke ${selected.device_name}`, () => revokeDevice(selected.id, "admin")) : toast.error("Select a device first (click Manage)")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Revoke</button>
          <button onClick={() => selected ? doDeviceAction(`Reset ${selected.device_name}`, () => resetDevice(selected.id)) : toast.error("Select a device first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Reset</button>
          <button onClick={() => selected ? doDeviceAction(`Force upgrade ${selected.device_name}`, () => forceUpgradeDevice(selected.id)) : toast.error("Select a device first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Force Upgrade</button>
          <button onClick={() => selected ? doDeviceAction(`Disable signal ${selected.device_name}`, () => disableDeviceSignal(selected.id)) : toast.error("Select a device first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Disable Signal</button>
          {selected && <span className="text-xs text-pat-text-muted self-center">Selected: {selected.device_name}</span>}
        </div>

        <DataTable data={data?.items || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
      </div>
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
