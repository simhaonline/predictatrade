"use client";
import { IconDatabase, IconAlertTriangle } from "@tabler/icons-react";

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

function PendingCard({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4 opacity-80">
      <div className="text-sm font-medium text-pat-text-primary mb-2">{title}</div>
      <div className="text-xs text-pat-text-muted">{hint}</div>
      <div className="text-xs text-pat-warning mt-2">Status pending — placeholder only</div>
    </div>
  );
}

export default function AdminBackupDrPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Backup &amp; DR</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Backup last-run, status, RPO/RTO and restore-test status. No backend data source is wired for this page.
        </p>
      </div>

      <DegradedBanner>
        Backup/DR backend pending. No last-run times, RPO/RTO values, or restore-test results are rendered.
        Do not infer backup health from this page.
      </DegradedBanner>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
        <PendingCard title="Last Backup" hint="Timestamp + duration of most recent successful backup." />
        <PendingCard title="Backup Status" hint="HEALTHY / DEGRADED / FAILED." />
        <PendingCard title="RPO / RTO" hint="Recovery point / time objectives." />
        <PendingCard title="Restore Test" hint="Last validated restore drill result." />
      </div>

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
