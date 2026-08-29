"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconPackage } from "@tabler/icons-react";
import { DegradedBanner } from "@/components/ui/degraded-banner";

interface ReleaseRow {
  id: string;
  component: string;
  version: string;
  channel: string;
  download_url: string;
  sha256: string;
  signature_key_id: string | null;
  mandatory: boolean;
  published_at: string | null;
  active: boolean;
}

interface ReleasesData {
  items: ReleaseRow[];
  note?: string;
}

const SCHEMA_COLUMNS = ["Version", "Date", "Channel", "Checksum (SHA256)", "Signature", "Rollback", "Status"];

export default function AdminReleasesPage() {
  const { data, isLoading, isError, error } = useQuery<ReleasesData>({
    queryKey: ["admin-releases"],
    queryFn: async () => (await customInstance.get("/admin/releases")).data as ReleasesData,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Client Releases</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Release registry: versions, checksums, signatures and rollback. Honest read-only view from the registry.
        </p>
      </div>

      {isLoading && <DegradedBanner>Loading release registry from backend…</DegradedBanner>}
      {isError && (
        <DegradedBanner>
          Release registry backend degraded: {error instanceof Error ? error.message : "unable to reach endpoint"}.
          Checksums, signatures and rollback states are NOT rendered.
        </DegradedBanner>
      )}

      {data && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
            <IconPackage size={16} /> Release Registry
          </h2>
          {data.note && !data.items.length ? (
            <div className="px-3 py-6 text-center text-xs text-pat-text-muted">{data.note}</div>
          ) : (
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
                  {data.items.map((r) => (
                    <tr key={r.id} className="border-b border-pat-border/60">
                      <td className="px-3 py-2 text-pat-text-primary">
                        {r.component} {r.version}
                      </td>
                      <td className="px-3 py-2 text-pat-text-secondary">
                        {r.published_at ? new Date(r.published_at).toISOString().slice(0, 19).replace("T", " ") : "—"}
                      </td>
                      <td className="px-3 py-2">{r.channel}</td>
                      <td className="px-3 py-2 font-mono text-xs text-pat-text-muted">{r.sha256 ? `${r.sha256.slice(0, 12)}…` : "—"}</td>
                      <td className="px-3 py-2 text-pat-text-muted">{r.signature_key_id ?? "—"}</td>
                      <td className="px-3 py-2">{r.mandatory ? "Mandatory" : "Optional"}</td>
                      <td className="px-3 py-2">{r.active ? "Active" : "Inactive"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Publish Release</h2>
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
