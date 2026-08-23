"use client";
import { IconPackage, IconAlertTriangle } from "@tabler/icons-react";

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

const SCHEMA_COLUMNS = ["Version", "Date", "Channel", "Checksum (SHA256)", "Signature", "Rollback"];

export default function AdminReleasesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Client Releases</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Release registry: versions, checksums, signatures and rollback. No backend registry is wired for this page.
        </p>
      </div>

      <DegradedBanner>
        Release registry backend pending. The table below is schema-only with no rows. Checksums, signatures and
        rollback states are NOT rendered because no authoritative source exists.
      </DegradedBanner>

      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconPackage size={16} /> Release Registry (Pending Backend)
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                {SCHEMA_COLUMNS.map((c) => (
                  <th key={c} className="px-3 py-2 font-medium">{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              <tr>
                <td colSpan={SCHEMA_COLUMNS.length} className="px-3 py-6 text-center text-xs text-pat-text-muted">
                  No releases available — registry backend pending.
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Publish Release (Pending Backend)</h2>
        <form className="grid grid-cols-1 md:grid-cols-4 gap-3" onSubmit={(e) => e.preventDefault()}>
          <input disabled placeholder="Version (e.g. 1.4.2)" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <input disabled placeholder="Channel (stable/beta)" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <input disabled placeholder="Artifact URL" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <button disabled className="px-3 py-2 text-sm rounded-md bg-pat-primary text-pat-primary-foreground opacity-50 cursor-not-allowed">
            Publish (disabled)
          </button>
        </form>
      </div>
    </div>
  );
}
