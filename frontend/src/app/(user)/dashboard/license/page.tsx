"use client";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import {
  IconLicense,
  IconClipboard,
  IconCheck,
  IconDeviceDesktop,
  IconShieldCheck,
  IconCalendar,
} from "@tabler/icons-react";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { fetchLicenses, fetchMtAccounts, type LicenseRecord, type MtAccount } from "@/lib/user-licensing-api";
import { DegradedNote, EmptyState } from "@/components/ui/tabs";

function val(v: unknown): string {
  if (v === null || v === undefined || v === "") return "N/A";
  return String(v);
}

export default function UserLicensePage() {
  const [copied, setCopied] = useState(false);

  const { data: licenses, isLoading, error } = useQuery<LicenseRecord[]>({
    queryKey: ["user-licenses"],
    queryFn: fetchLicenses,
  });

  const { data: accounts } = useQuery<MtAccount[]>({
    queryKey: ["user-mt-accounts"],
    queryFn: fetchMtAccounts,
  });

  const copyKey = (key: string) => {
    if (!key || key === "N/A") return;
    navigator.clipboard.writeText(key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (isLoading) return <div className="text-sm text-pat-text-secondary">Loading license…</div>;
  if (error) return <div className="rounded border border-pat-danger/30 p-4 text-sm text-pat-danger">License details unavailable.</div>;
  if (!licenses || licenses.length === 0) {
    return (
      <div className="space-y-4">
        <h1 className="text-xl font-bold text-pat-text-primary">License</h1>
        <EmptyState title="No license found." hint="A license is provisioned when you subscribe to a paid plan." />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">License</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Your Predict-A-Trade license key, activation limits and linked terminals.</p>
      </div>

      {licenses.map((lic, i) => (
        <div key={lic.id || i} className="rounded-xl border border-pat-border bg-pat-bg-surface p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <IconLicense size={18} className="text-pat-success" />
              <span className="text-sm font-semibold text-pat-text-primary">{val(lic.plan_name)} License</span>
            </div>
            <StatusBadge status={val(lic.status)} size="sm" />
          </div>

          <div className="flex items-center gap-2">
            <div className="flex-1 rounded-lg border border-pat-border bg-pat-bg-surface-secondary/30 px-3 py-2 font-mono text-sm text-pat-text-primary break-all">
              {val(lic.license_key)}
            </div>
            <button
              onClick={() => copyKey(val(lic.license_key))}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg border border-pat-border bg-pat-bg-surface-secondary text-xs text-pat-text-secondary hover:text-pat-text-primary"
            >
              {copied ? <IconCheck size={14} className="text-pat-success" /> : <IconClipboard size={14} />}
              {copied ? "Copied" : "Copy"}
            </button>
          </div>

          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <Stat icon={<IconDeviceDesktop size={14} />} label="Max devices" value={val(lic.max_devices)} />
            <Stat icon={<IconDeviceDesktop size={14} />} label="Max MT accounts" value={val(lic.max_mt_accounts)} />
            <Stat icon={<IconShieldCheck size={14} />} label="Devices active" value={val(lic.device_count)} />
            <Stat icon={<IconCalendar size={14} />} label="Expires" value={lic.expires_at ? format(new Date(lic.expires_at), "MMM d, yyyy") : "No expiry"} />
          </div>

          {lic.revoked_at && (
            <div className="rounded border border-pat-danger/20 bg-pat-danger/5 px-3 py-1.5 text-[11px] text-pat-danger">
              Revoked {format(new Date(lic.revoked_at), "MMM d, yyyy")}{lic.revocation_reason ? ` — ${lic.revocation_reason}` : ""}
            </div>
          )}

          <div className="text-[10px] text-pat-text-muted">
            Created {lic.created_at ? format(new Date(lic.created_at), "MMM d, yyyy HH:mm") : "—"}
          </div>
        </div>
      ))}

      {accounts && accounts.length > 0 && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Linked Terminals</h2>
          <div className="space-y-2">
            {accounts.map((a, i) => (
              <div key={a.id || i} className="flex items-center justify-between rounded-md border border-pat-border/60 bg-pat-bg-surface-secondary/20 px-3 py-2">
                <div>
                  <div className="text-xs font-medium text-pat-text-primary">{val(a.client_type)} — {val(a.broker_name)}</div>
                  <div className="text-[10px] text-pat-text-muted">Account: {val(a.mt_account_login)} | Server: {val(a.broker_server)}</div>
                </div>
                <StatusBadge status={val(a.status)} size="sm" />
              </div>
            ))}
          </div>
        </div>
      )}

      <DegradedNote>
        License keys are hardware-bound per the Windows Agent. If you believe your license details are incorrect, contact support.
      </DegradedNote>
    </div>
  );
}

function Stat({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-lg border border-pat-border bg-pat-bg-surface-secondary/30 p-3">
      <div className="flex items-center gap-1.5 text-[10px] text-pat-text-muted">{icon} {label}</div>
      <div className="mt-1 text-lg font-bold text-pat-text-primary">{value}</div>
    </div>
  );
}
