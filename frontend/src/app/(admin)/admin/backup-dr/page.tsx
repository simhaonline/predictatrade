"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconDatabase } from "@tabler/icons-react";
import { DegradedBanner } from "@/components/ui/degraded-banner";

interface BackupComponent {
  config_key: string;
  config_value: string | null;
  is_configured: boolean;
  required_for_prod: boolean;
  description: string | null;
}

interface BackupDrData {
  status: string;
  configured: boolean;
  last_archived_time: string | null;
  note: string;
  components?: BackupComponent[];
}

function StatusPill({ status }: { status: string }) {
  const tone =
    status === "CONFIGURED"
      ? "bg-emerald-500/15 text-emerald-400 border-emerald-500/30"
      : status === "CONFIGURED_NO_ARCHIVE"
        ? "bg-pat-warning/15 text-pat-warning border-pat-warning/30"
        : "bg-pat-muted/15 text-pat-text-muted border-pat-border";
  return <span className={`px-2 py-0.5 rounded-full border text-xs font-medium ${tone}`}>{status}</span>;
}

export default function AdminBackupDrPage() {
  const { data, isLoading, isError, error } = useQuery<BackupDrData>({
    queryKey: ["admin-backup-dr"],
    queryFn: async () => (await customInstance.get("/admin/backup-dr")).data as BackupDrData,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Backup &amp; DR</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Backup last-run, status, RPO/RTO and restore-test status. Honest read-only view from system configuration.
        </p>
      </div>

      {isLoading && (
        <DegradedBanner>Loading backup/DR status from backend…</DegradedBanner>
      )}

      {isError && (
        <DegradedBanner>
          Backup/DR backend degraded: {error instanceof Error ? error.message : "unable to reach endpoint"}.
          No last-run times, RPO/RTO values, or restore-test results are rendered.
        </DegradedBanner>
      )}

      {data && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
            <div className="text-sm font-medium text-pat-text-primary mb-2">Status</div>
            <StatusPill status={data.status} />
          </div>
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
            <div className="text-sm font-medium text-pat-text-primary mb-2">Configured</div>
            <div className="text-xs text-pat-text-muted">{data.configured ? "Yes" : "No"}</div>
          </div>
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
            <div className="text-sm font-medium text-pat-text-primary mb-2">Last Archived (WAL)</div>
            <div className="text-xs text-pat-text-muted">{data.last_archived_time ?? "Not recorded"}</div>
          </div>
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
            <div className="text-sm font-medium text-pat-text-primary mb-2">Note</div>
            <div className="text-xs text-pat-text-muted">{data.note}</div>
          </div>
        </div>
      )}

      {data?.components && data.components.length > 0 && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3">Configuration Components</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                  <th className="px-3 py-2 font-medium">Key</th>
                  <th className="px-3 py-2 font-medium">Value</th>
                  <th className="px-3 py-2 font-medium">Configured</th>
                  <th className="px-3 py-2 font-medium">Required for Prod</th>
                  <th className="px-3 py-2 font-medium">Description</th>
                </tr>
              </thead>
              <tbody>
                {data.components.map((c) => (
                  <tr key={c.config_key} className="border-b border-pat-border/60">
                    <td className="px-3 py-2 text-pat-text-primary">{c.config_key}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{c.config_value || "—"}</td>
                    <td className="px-3 py-2">{c.is_configured ? "Yes" : "No"}</td>
                    <td className="px-3 py-2">{c.required_for_prod ? "Yes" : "No"}</td>
                    <td className="px-3 py-2 text-pat-text-muted">{c.description || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Trigger Restore Test (Pending Backend)</h2>
        <form className="grid grid-cols-1 md:grid-cols-3 gap-3" onSubmit={(e) => e.preventDefault()}>
          <input disabled placeholder="Target (db/snapshot)" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <input disabled placeholder="Restore point" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <button disabled className="px-3 py-2 text-sm rounded-md bg-pat-primary text-pat-primary-foreground opacity-50 cursor-not-allowed">
            Run Test (disabled)
          </button>
        </form>
      </div>
    </div>
  );
}
