"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useState } from "react";
import { IconServer, IconAlertTriangle } from "@tabler/icons-react";
import {
  fetchAllMtAccountsAdmin,
  fetchAllDevicesAdmin,
  createMtAccount,
  type AdminMtAccount,
  type AdminDevice,
  type CreateMtAccountBody,
} from "@/lib/admin-mt-accounts-api";

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

function timeAgo(iso?: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const secs = Math.max(0, Math.floor((Date.now() - then) / 1000));
  if (secs < 60) return `${secs}s ago`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

// Render a value, or a clean "—" when it's null/undefined/empty string.
function val(v?: string | null): string {
  return v && v.trim() !== "" ? v : "—";
}

// Balance is only as fresh as the last EA sync. Flag it stale past 1 hour so a
// frozen number is never mistaken for a live broker balance.
function isStale(iso?: string): boolean {
  if (!iso) return false;
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return false;
  return Date.now() - then > 60 * 60 * 1000;
}

export default function AdminMtAccountsPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateMtAccountBody>({
    deviceId: "",
    brokerName: "",
    brokerServer: "",
    mtAccountLogin: "",
    clientType: "MT5",
  });

  // Admin fleet-wide listing (check.md #4): one row per DEVICE (the backend
  // returns DISTINCT ON device via the freshest sync). This is the authoritative
  // view for the admin page — render it directly, never the user-scoped list
  // (which still carries multiple activation rows per device → duplicates).
  const adminQ = useQuery<AdminMtAccount[]>({
    queryKey: ["admin-mt-accounts"],
    queryFn: fetchAllMtAccountsAdmin,
    refetchInterval: 20000,
  });
  const devicesQ = useQuery<AdminDevice[]>({
    queryKey: ["admin-devices"],
    queryFn: fetchAllDevicesAdmin,
    refetchInterval: 60000,
  });

  const createMutation = useMutation({
    mutationFn: (body: CreateMtAccountBody) => createMtAccount(body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-mt-accounts"] });
      toast.success("MT account submitted");
      setForm({ deviceId: "", brokerName: "", brokerServer: "", mtAccountLogin: "", clientType: "MT5" });
    },
    onError: (e) => toast.error(`Failed to create MT account: ${e instanceof Error ? e.message : "unknown"}`),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">MT Accounts</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          MetaTrader accounts linked across the fleet — one entry per device. Create is POST /licensing/mt-accounts (requires a bound device id).
        </p>
      </div>

      {adminQ.error && (
        <DegradedBanner>
          Could not load MT accounts: {adminQ.error instanceof Error ? adminQ.error.message : "unknown error"}.
        </DegradedBanner>
      )}

      {/* LIVE fleet list — ONE row per device (adminQ is deduplicated server-side) */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconServer size={16} /> Linked Accounts (LIVE) — one entry per device
        </h2>
        {adminQ.isLoading && <div className="text-xs text-pat-text-muted">Loading accounts...</div>}
        {!adminQ.isLoading && (adminQ.data?.length ?? 0) === 0 && (
          <div className="text-xs text-pat-text-muted">No devices with linked MT accounts yet. Accounts appear when a device activates a bound MT account.</div>
        )}
        {!adminQ.isLoading && (adminQ.data?.length ?? 0) > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                  <th className="px-3 py-2 font-medium">Login</th>
                  <th className="px-3 py-2 font-medium">Broker</th>
                  <th className="px-3 py-2 font-medium">Server</th>
                  <th className="px-3 py-2 font-medium">Client</th>
                  <th className="px-3 py-2 font-medium">Device</th>
                  <th className="px-3 py-2 font-medium">Connection</th>
                  <th className="px-3 py-2 font-medium">Balance</th>
                  <th className="px-3 py-2 font-medium">License</th>
                  <th className="px-3 py-2 font-medium">User</th>
                </tr>
              </thead>
              <tbody>
                {adminQ.data!.map((a) => (
                  <tr key={a.id} className="border-b border-pat-border/50">
                    <td className="px-3 py-2 font-mono text-pat-text-primary">{val(a.mt_account_login)}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{val(a.broker_name)}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{val(a.broker_server)}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{val(a.client_type)}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{val(a.device_name)}</td>
                    <td className="px-3 py-2">
                      <span className={`text-xs px-2 py-0.5 rounded-full ${a.connection_status === "ONLINE" ? "bg-emerald-500/15 text-emerald-400" : "bg-pat-bg-surface-secondary text-pat-text-muted"}`}>
                        {a.connection_status === "ONLINE" ? "ONLINE" : "OFFLINE"}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-pat-text-primary">
                      {a.account_balance !== undefined && a.account_balance !== null && a.account_balance !== 0
                        ? `${a.account_balance.toLocaleString()} ${a.currency ?? ""}`.trim()
                        : "—"}
                      {a.account_balance === 0 && (
                        <span className="block text-[10px] text-pat-text-muted">not reported by EA</span>
                      )}
                      {a.last_account_update && (
                        <span className={`block text-[10px] ${isStale(a.last_account_update) ? "text-pat-warning" : "text-pat-text-muted"}`}>
                          synced {timeAgo(a.last_account_update)}
                        </span>
                      )}
                    </td>
                    <td className="px-3 py-2 font-mono text-xs text-pat-text-secondary">{a.license_key ?? "—"}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{a.user_email ?? "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create form (LIVE wiring, may be limited by backend validation) */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Register MT Account</h2>
        <DegradedBanner>
          This page lists every MetaTrader account linked across the whole fleet (the table above
          reads it from the backend's fleet-wide list). Registering a new account is intentionally
          tied to a specific device: you must supply a valid device id that already has a license
          bound to it, otherwise the backend rejects the request with a real error (no fake
          success). Enter the device id in the field below.
        </DegradedBanner>
        <form
          className="mt-3 grid grid-cols-1 md:grid-cols-5 gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            if (!form.deviceId || !form.mtAccountLogin) {
              toast.error("Device id and MT account login are required");
              return;
            }
            createMutation.mutate(form);
          }}
        >
          <select
            value={form.deviceId}
            onChange={(e) => setForm({ ...form, deviceId: e.target.value })}
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
          >
            <option value="">Select a licensed device…</option>
            {(devicesQ.data ?? []).map((d) => (
              <option key={d.id} value={d.id}>
                {d.device_name || d.hostname || d.id}
                {d.user_email ? ` — ${d.user_email}` : ""}
                {d.license_key ? ` (${d.license_key})` : ""}
              </option>
            ))}
          </select>
          <input
            value={form.mtAccountLogin}
            onChange={(e) => setForm({ ...form, mtAccountLogin: e.target.value })}
            placeholder="MT login"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
          />
          <input
            value={form.brokerName}
            onChange={(e) => setForm({ ...form, brokerName: e.target.value })}
            placeholder="Broker"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
          />
          <input
            value={form.brokerServer}
            onChange={(e) => setForm({ ...form, brokerServer: e.target.value })}
            placeholder="Server"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
          />
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="px-3 py-2 text-sm rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50"
          >
            Register
          </button>
        </form>
      </div>
    </div>
  );
}
