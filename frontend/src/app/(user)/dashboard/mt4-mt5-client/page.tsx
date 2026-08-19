"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";

interface Device { id: string; client_type: string; fingerprint: string; status: string; last_seen_at: string | null; }

export default function UserMtClientPage() {
  const { data: devices, isLoading } = useQuery({
    queryKey: ["user-devices"],
    queryFn: async () => {
      const res = await customInstance.get("/licensing/devices");
      return (res.data as Device[]) || [];
    },
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">MT4/MT5 Client</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Download the Windows Agent and manage your devices.</p>
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Download</h2>
        <div className="flex gap-3">
          <a href="/downloads/Predict-A-Trade-Agent-Setup.exe" className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded text-sm font-medium hover:bg-primary/90 transition-colors">
            Download Windows Agent
          </a>
        </div>
        <p className="text-xs text-pat-text-muted mt-3">Run the installer, enter your license key, and the agent will auto-connect to the live signal stream.</p>
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Your Devices</h2>
        {isLoading && <div className="text-sm text-pat-text-secondary">Loading...</div>}
        <div className="space-y-2">
          {devices?.map((d) => (
            <div key={d.id} className="flex items-center justify-between border-b border-pat-border pb-2">
              <div>
                <div className="text-sm text-pat-text-primary">{d.client_type}</div>
                <div className="text-xs text-pat-text-muted">{d.fingerprint?.slice(0, 16)}...</div>
              </div>
              <StatusBadge status={d.status} size="sm" />
            </div>
          )) || (
            !isLoading && <div className="text-sm text-pat-text-muted">No devices registered.</div>
          )}
        </div>
      </div>
    </div>
  );
}
