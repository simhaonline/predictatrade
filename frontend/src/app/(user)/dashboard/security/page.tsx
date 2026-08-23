"use client";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/providers/auth-provider";
import { toast } from "sonner";
import {
  IconShieldLock,
  IconDeviceDesktop,
  IconHistory,
  IconKey,
  IconCircleCheck,
  IconAlertTriangle,
} from "@tabler/icons-react";
import { Tabs, DegradedNote, EmptyState, type TabItem } from "@/components/ui/tabs";
import StatusBadge from "@/components/ui/status-badge";
import {
  mfaSetup,
  mfaVerify,
  fetchTrustedDevices,
  revokeTrustedDevice,
  fetchSessions,
  fetchLoginHistory,
  type TrustedDevice,
} from "@/lib/user-security-api";
import { format } from "date-fns";

type TabId = "mfa" | "sessions" | "devices" | "history";

export default function UserSecurityPage() {
  const { user, refreshUser } = useAuth();
  const [tab, setTab] = useState<TabId>("mfa");

  const tabs: TabItem[] = [
    { id: "mfa", label: "MFA", icon: IconShieldLock },
    { id: "sessions", label: "Sessions", icon: IconHistory },
    { id: "devices", label: "Trusted Devices", icon: IconDeviceDesktop },
    { id: "history", label: "Login History", icon: IconKey },
  ];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Security</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Manage multi-factor authentication, active sessions, trusted devices and login history.
        </p>
      </div>
      <Tabs tabs={tabs} active={tab} onChange={(id) => setTab(id as TabId)} />
      {tab === "mfa" && <MfaTab mfaEnabled={!!user?.mfaEnabled} onEnabled={refreshUser} />}
      {tab === "sessions" && <SessionsTab />}
      {tab === "devices" && <DevicesTab />}
      {tab === "history" && <HistoryTab />}
    </div>
  );
}

function MfaTab({ mfaEnabled, onEnabled }: { mfaEnabled: boolean; onEnabled: () => void }) {
  const [secret, setSecret] = useState<string | null>(null);
  const [otpauth, setOtpauth] = useState<string | null>(null);
  const [code, setCode] = useState("");

  const setup = useMutation({
    mutationFn: async () => {
      const r = await mfaSetup();
      setSecret(r.secret);
      setOtpauth(r.otpauth);
    },
    onError: () => toast.error("Could not start MFA setup."),
  });

  const verify = useMutation({
    mutationFn: async () => {
      if (!code) throw new Error("Enter the 6-digit code");
      await mfaVerify(code);
    },
    onSuccess: () => {
      toast.success("MFA enabled successfully.");
      setSecret(null);
      setOtpauth(null);
      setCode("");
      onEnabled();
    },
    onError: (e: Error) => toast.error(e.message || "Invalid code."),
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 rounded-lg border border-pat-border bg-pat-bg-surface p-4">
        <div className={`flex items-center justify-center w-10 h-10 rounded-lg ${mfaEnabled ? "bg-pat-success/10" : "bg-pat-warning/10"}`}>
          {mfaEnabled ? <IconCircleCheck size={20} className="text-pat-success" /> : <IconAlertTriangle size={20} className="text-pat-warning" />}
        </div>
        <div className="flex-1">
          <div className="text-sm font-medium text-pat-text-primary">Multi-Factor Authentication</div>
          <div className="text-xs text-pat-text-muted">
            {mfaEnabled ? "TOTP MFA is enabled on your account." : "TOTP MFA is not yet enabled."}
          </div>
        </div>
        <StatusBadge status={mfaEnabled ? "ENABLED" : "DISABLED"} size="sm" />
      </div>

      {!mfaEnabled && !secret && (
        <button
          onClick={() => setup.mutate()}
          disabled={setup.isPending}
          className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground disabled:opacity-50"
        >
          {setup.isPending ? "Preparing…" : "Enable MFA"}
        </button>
      )}

      {secret && (
        <div className="space-y-3 rounded-lg border border-pat-border bg-pat-bg-surface p-4">
          <div className="text-sm font-medium text-pat-text-primary">Scan or enter the secret</div>
          <div className="text-xs text-pat-text-muted">
            Add this to your authenticator app (e.g. Google Authenticator). A QR code is encoded in the otpauth URI below.
          </div>
          <div className="rounded-md border border-pat-border bg-pat-bg-surface-secondary px-3 py-2 font-mono text-xs text-pat-text-primary break-all">
            {secret}
          </div>
          <div className="rounded-md border border-pat-border bg-pat-bg-surface-secondary px-3 py-2 font-mono text-[10px] text-pat-text-secondary break-all">
            {otpauth}
          </div>
          <div className="flex items-center gap-2">
            <input
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
              placeholder="123456"
              className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
            />
            <button
              onClick={() => verify.mutate()}
              disabled={verify.isPending || code.length !== 6}
              className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground disabled:opacity-50"
            >
              {verify.isPending ? "Verifying…" : "Verify & Enable"}
            </button>
          </div>
        </div>
      )}

      {mfaEnabled && (
        <DegradedNote>
          Disabling MFA is not available from this page yet. Use account recovery / admin support if you need to reset it.
        </DegradedNote>
      )}
    </div>
  );
}

function SessionsTab() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["user-sessions"],
    queryFn: fetchSessions,
    retry: false,
  });

  if (isLoading) return <div className="text-sm text-pat-text-secondary">Loading sessions…</div>;
  if (error) {
    return (
      <DegradedNote>
        Live session listing is not available to end users — the session endpoint is restricted to administrators.
        You can still review and revoke your registered devices under the <strong>Trusted Devices</strong> tab.
      </DegradedNote>
    );
  }
  return <pre className="text-xs text-pat-text-muted">{JSON.stringify(data, null, 2)}</pre>;
}

function DevicesTab() {
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery<TrustedDevice[]>({
    queryKey: ["user-trusted-devices"],
    queryFn: fetchTrustedDevices,
  });

  const revoke = useMutation({
    mutationFn: (id: string) => revokeTrustedDevice(id),
    onSuccess: () => {
      toast.success("Device revoked.");
      queryClient.invalidateQueries({ queryKey: ["user-trusted-devices"] });
    },
    onError: () => toast.error("Could not revoke device."),
  });

  if (isLoading) return <div className="text-sm text-pat-text-secondary">Loading devices…</div>;
  if (error) return <div className="rounded border border-pat-danger/30 p-4 text-sm text-pat-danger">Trusted devices unavailable.</div>;
  if (!data || data.length === 0) return <EmptyState title="No trusted devices registered." hint="Install the Windows Agent to register a device." />;

  return (
    <div className="space-y-3">
      {data.map((d) => {
        const revoked = !!d.revoked_at;
        return (
          <div key={d.id} className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="flex items-center justify-center w-9 h-9 rounded-lg bg-pat-info/10">
                  <IconDeviceDesktop size={18} className="text-pat-info" />
                </div>
                <div>
                  <div className="text-sm font-medium text-pat-text-primary">{d.device_name || "Unnamed device"}</div>
                  <div className="text-[10px] text-pat-text-muted">
                    {d.os || "—"} | Host: {d.hostname || "—"} | Agent: {d.agent_version || "—"}
                  </div>
                </div>
              </div>
              {!revoked && (
                <button
                  onClick={() => revoke.mutate(d.id)}
                  disabled={revoke.isPending}
                  className="rounded-md border border-pat-danger/30 px-3 py-1.5 text-xs text-pat-danger hover:bg-pat-danger/10 disabled:opacity-50"
                >
                  Revoke
                </button>
              )}
            </div>
            <div className="mt-2 flex flex-wrap items-center gap-3 text-xs">
              <span className="text-pat-text-muted">
                Status: <StatusBadge status={(d.status as string) || "UNKNOWN"} size="sm" />
              </span>
              {d.license_key && <span className="text-pat-text-muted">License: <span className="font-mono text-pat-text-secondary">{d.license_key}</span></span>}
              <span className="text-pat-text-muted">
                Registered: {d.registered_at ? format(new Date(d.registered_at), "MMM d, yyyy") : "—"}
              </span>
              <span className="text-pat-text-muted">
                Last seen: {d.last_seen_at ? format(new Date(d.last_seen_at), "MMM d, yyyy HH:mm") : "—"}
              </span>
            </div>
            {revoked && (
              <div className="mt-2 rounded border border-pat-danger/20 bg-pat-danger/5 px-3 py-1.5 text-[11px] text-pat-danger">
                Revoked {d.revoked_at ? format(new Date(d.revoked_at), "MMM d, yyyy") : ""}{d.revocation_reason ? ` — ${d.revocation_reason}` : ""}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function HistoryTab() {
  const { isLoading, error } = useQuery({
    queryKey: ["user-login-history"],
    queryFn: fetchLoginHistory,
    retry: false,
  });

  if (isLoading) return <div className="text-sm text-pat-text-secondary">Loading login history…</div>;
  if (error) {
    return (
      <DegradedNote>
        Login history is available to administrators through the audit log and is not exposed to end users in this build.
        For a copy of your access history, contact support.
      </DegradedNote>
    );
  }
  return <EmptyState title="No login history to display." />;
}
