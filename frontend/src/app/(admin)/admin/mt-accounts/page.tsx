"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useState } from "react";
import { IconServer, IconAlertTriangle } from "@tabler/icons-react";
import {
  fetchMtAccounts,
  createMtAccount,
  type MtAccountDevice,
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

export default function AdminMtAccountsPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateMtAccountBody>({
    deviceId: "",
    brokerName: "",
    brokerServer: "",
    mtAccountLogin: "",
    clientType: "MT5",
  });

  const { data: accounts, isLoading, error } = useQuery<MtAccountDevice[]>({
    queryKey: ["admin-mt-accounts"],
    queryFn: fetchMtAccounts,
    refetchInterval: 20000,
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

  const rows = (accounts ?? []).flatMap((dev) =>
    (dev.activations ?? []).map((a) => ({ device: dev, activation: a }))
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">MT Accounts</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          MetaTrader accounts linked via licensing (GET /licensing/mt-accounts). Create is POST /licensing/mt-accounts.
        </p>
      </div>

      {error && (
        <DegradedBanner>
          Could not load MT accounts: {error instanceof Error ? error.message : "unknown error"}.
        </DegradedBanner>
      )}

      {/* LIVE list */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconServer size={16} /> Linked Accounts (LIVE)
        </h2>
        {isLoading && <div className="text-xs text-pat-text-muted">Loading accounts...</div>}
        {!isLoading && rows.length === 0 && (
          <div className="text-xs text-pat-text-muted">
            No MT accounts returned for the current session. (Endpoint is user-scoped; admin-wide listing depends on a future admin endpoint.)
          </div>
        )}
        {rows.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                  <th className="px-3 py-2 font-medium">Login</th>
                  <th className="px-3 py-2 font-medium">Broker</th>
                  <th className="px-3 py-2 font-medium">Server</th>
                  <th className="px-3 py-2 font-medium">Client</th>
                  <th className="px-3 py-2 font-medium">Device</th>
                  <th className="px-3 py-2 font-medium">Status</th>
                  <th className="px-3 py-2 font-medium">Balance</th>
                </tr>
              </thead>
              <tbody>
                {rows.map(({ device, activation }) => (
                  <tr key={activation.id} className="border-b border-pat-border/50">
                    <td className="px-3 py-2 font-mono text-pat-text-primary">{activation.mt_account_login ?? "—"}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{activation.broker_name ?? "—"}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{activation.broker_server ?? "—"}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{activation.client_type ?? "—"}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{device.device_name ?? device.id}</td>
                    <td className="px-3 py-2">
                      <span className="text-xs px-2 py-0.5 rounded-full bg-pat-bg-surface-secondary text-pat-text-muted">
                        {device.connection_status ?? "UNKNOWN"}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-pat-text-primary">
                      {activation.balance !== undefined
                        ? `${activation.balance} ${activation.currency ?? ""}`.trim()
                        : "—"}
                    </td>
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
          The create endpoint is user-scoped and requires a valid bound device id. If the current session has no
          eligible device, the request will be rejected by the backend (honest error shown, no fake success).
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
          <input
            value={form.deviceId}
            onChange={(e) => setForm({ ...form, deviceId: e.target.value })}
            placeholder="Device id"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
          />
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
