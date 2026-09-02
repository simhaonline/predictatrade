"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { createLicense, suspendLicense, revokeLicense, renewLicense, resetLicense, forceLogoutLicense, fetchLicenseActivations } from "@/lib/admin-commercial-api";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import { IconAlertTriangle, IconHistory } from "@tabler/icons-react";

interface License { id: string; user_id: string; user_email: string; key: string; plan_name: string; status: string; activated_at: string | null; expires_at: string | null; max_devices: number; max_mt_accounts: number; subscription_status: string | null; }
interface Activation { id?: string; device_name?: string; activated_at?: string; ip?: string; [key: string]: unknown; }

export default function AdminLicensesPage() {
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<License | null>(null);
  const [history, setHistory] = useState<Activation[] | null>(null);
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-licenses", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/licenses?page=${page}&limit=20`);
      return res.data as { items: License[]; total: number; page: number; limit: number };
    },
  });

  // Plans for the Create-license picker (backend contract: user_id + plan_id UUIDs)
  const { data: plans } = useQuery({
    queryKey: ["admin-plans"],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/plans`);
      return res.data as { id: string; code: string; name: string; monthly_price: number; currency: string }[];
    },
  });
  const [planId, setPlanId] = useState("");

  // License management endpoints are implemented in the backend (/licensing/licenses/*).
  const mgmtMutation = useMutation({
    mutationFn: async (p: { action: string; fn: () => Promise<unknown> }) => { await p.fn(); },
    onSuccess: (_d, v) => {
      queryClient.invalidateQueries({ queryKey: ["admin-licenses", page] });
      toast.success(`${v.action} succeeded`);
    },
    onError: (err: unknown, v) => {
      toast.error(`${v.action} failed — ${err instanceof Error ? err.message : "not available"}`);
    },
  });

  const doAction = (action: string, fn: () => Promise<unknown>) => {
    toast.message(`${action} requested…`);
    mgmtMutation.mutate({ action, fn });
  };

  const viewHistory = async (lic: License) => {
    try {
      const data = await fetchLicenseActivations(lic.id) as { items?: Activation[]; activations?: Activation[] } | Activation[];
      const rows = Array.isArray(data) ? data : (data.items ?? data.activations ?? []);
      setHistory(rows);
    } catch (err) {
      toast.error(`Activation history failed — ${err instanceof Error ? err.message : "not available"}`);
      setHistory([]);
    }
  };

  const columns: DataTableColumn<License>[] = [
    { key: "user_email", header: "User", cell: (row) => <button onClick={() => setSelected(row)} className="text-sm text-pat-text-primary hover:underline">{row.user_email || "—"}</button> },
    { key: "key", header: "License Key", cell: (row) => <span className="text-xs text-pat-text-muted font-mono">{row.key || "—"}</span> },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-sm text-pat-text-primary">{row.plan_name || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "max_devices", header: "Max Devices", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.max_devices ?? "—"}</span> },
    { key: "activated_at", header: "Issued", cell: (row) => <span className="text-xs text-pat-text-muted">{row.activated_at ? format(new Date(row.activated_at), "MMM d, yyyy") : "—"}</span> },
    { key: "expires_at", header: "Expires", cell: (row) => <span className="text-xs text-pat-text-muted">{row.expires_at ? format(new Date(row.expires_at), "MMM d, yyyy") : "—"}</span> },
    { key: "actions", header: "Manage", cell: (row) => (
      <button onClick={() => { setSelected(row); }} className="text-xs bg-pat-bg-surface-secondary text-pat-text-primary px-2 py-1 rounded hover:bg-pat-bg-surface-secondary transition-colors">
        Manage
      </button>
    )},
  ];

  const totalPages = data?.total ? Math.ceil(data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">License Management</h1>
        <p className="text-sm text-pat-text-secondary mt-1">View and manage all platform licenses.</p>
      </div>

      <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 px-4 py-3 flex items-start gap-2">
        <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
        <div className="text-xs text-pat-text-secondary">
          License management actions (create, suspend, revoke, renew, reset, force-logout, activation-history) are wired to live backend endpoints under <span className="font-mono">/licensing/licenses/*</span>. Each action requires admin authorization and returns a real result from the licensing database. The license list below is live.
        </div>
      </div>

      <div className="flex flex-wrap gap-2 items-center">
        {/* Backend contract (POST /licensing/licenses): user_id + plan_id are required.
            The Create action issues a license for the SELECTED row's user, under the
            picked plan, and displays the returned PAT-… key. */}
        <select
          value={planId}
          onChange={(e) => setPlanId(e.target.value)}
          className="px-2 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border"
        >
          <option value="">Plan…</option>
          {(plans ?? []).map((p) => (
            <option key={p.id} value={p.id}>
              {p.name} ({p.currency} {p.monthly_price}/mo)
            </option>
          ))}
        </select>
        <button
          onClick={() => {
            if (!selected) { toast.error("Select a license row first — the new license is issued to that user"); return; }
            if (!planId) { toast.error("Pick a plan first"); return; }
            doAction(`Create license for ${selected.user_email}`, async () => {
              const created = await createLicense({ user_id: selected.user_id, plan_id: planId, max_devices: 2, max_mt_accounts: 2, valid_days: 365 }) as { license_key?: string; key?: string };
              setCreatedKey(created?.license_key ?? created?.key ?? null);
            });
          }}
          className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors"
        >
          Create
        </button>
        {createdKey && (
          <span className="text-xs font-mono text-pat-text-primary bg-pat-bg-surface-secondary px-2 py-1 rounded">
            New key: {createdKey}
          </span>
        )}
        <button onClick={() => selected ? doAction(`Suspend ${selected.user_email}`, () => suspendLicense(selected.id, "admin")) : toast.error("Select a license first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Suspend</button>
        <button onClick={() => selected ? doAction(`Revoke ${selected.user_email}`, () => revokeLicense(selected.id, "admin")) : toast.error("Select a license first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Revoke</button>
        <button onClick={() => selected ? doAction(`Renew ${selected.user_email}`, () => renewLicense(selected.id)) : toast.error("Select a license first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Renew</button>
        <button onClick={() => selected ? doAction(`Reset ${selected.user_email}`, () => resetLicense(selected.id)) : toast.error("Select a license first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Reset</button>
        <button onClick={() => selected ? doAction(`Force logout ${selected.user_email}`, () => forceLogoutLicense(selected.id)) : toast.error("Select a license first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40">Force Logout</button>
        <button onClick={() => selected ? viewHistory(selected) : toast.error("Select a license first")} disabled={!selected} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded hover:bg-pat-bg-surface-secondary transition-colors disabled:opacity-40 flex items-center gap-1"><IconHistory size={12} /> Activation History</button>
        {selected && <span className="text-xs text-pat-text-muted self-center">Selected: {selected.user_email}</span>}
      </div>

      <DataTable data={data?.items || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-secondary">Page {page} of {totalPages}</span>
          <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
        </div>
      )}

      {history && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setHistory(null)}>
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg shadow-xl max-w-lg w-full mx-4 p-5" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Activation History — {selected?.user_email}</h3>
            {history.length === 0 ? (
              <div className="text-xs text-pat-text-muted">No activation history returned (empty or license not yet activated).</div>
            ) : (
              <div className="space-y-2 max-h-80 overflow-auto">
                {history.map((a, i) => (
                  <div key={a.id ?? i} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2 text-xs">
                    <span className="text-pat-text-primary">{a.device_name ?? "—"}</span>
                    <span className="text-pat-text-muted">{a.ip ?? ""}</span>
                    <span className="text-pat-text-muted">{a.activated_at ? format(new Date(a.activated_at), "MMM d, yyyy") : "—"}</span>
                  </div>
                ))}
              </div>
            )}
            <div className="flex justify-end mt-4">
              <button onClick={() => setHistory(null)} className="px-3 py-1.5 text-xs border border-pat-border-strong rounded-md text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors">Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
