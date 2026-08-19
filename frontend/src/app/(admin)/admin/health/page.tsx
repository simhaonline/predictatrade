"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";

interface HealthServiceStatus {
  service: string;
  status: 'HEALTHY' | 'DEGRADED' | 'OFFLINE' | 'UNKNOWN';
  latency_ms?: number;
  last_check: string;
  version?: string;
  details?: string;
}

export default function AdminHealthPage() {
  const { data, isLoading, error, refetch } = useQuery<{ services: HealthServiceStatus[] }>({
    queryKey: ["admin-health"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/health");
      return res.data as { services: HealthServiceStatus[] };
    },
    refetchInterval: 30000,
  });

  if (isLoading) return <div className="text-sm text-pat-text-secondary">Loading health status...</div>;
  if (error) return (
    <div className="text-center py-12">
      <div className="text-pat-danger text-sm mb-2">Failed to load health status</div>
      <div className="text-xs text-pat-text-muted mb-3">{error instanceof Error ? error.message : "Unknown error"}</div>
      <button onClick={() => refetch()} className="text-xs bg-pat-bg-surface-secondary px-3 py-1.5 rounded">Retry</button>
    </div>
  );

  const services = data?.services || [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">System Health</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Service health and status monitoring with real dependency checks.</p>
      </div>
      <div className="space-y-2">
        {services.map((svc) => (
          <div key={svc.service} className="flex items-center justify-between bg-pat-bg-surface border border-pat-border rounded-lg p-4">
            <div className="flex items-center gap-3">
              <div className={`w-2 h-2 rounded-full ${
                svc.status === 'HEALTHY' ? 'bg-pat-success' :
                svc.status === 'DEGRADED' ? 'bg-pat-warning' :
                svc.status === 'OFFLINE' ? 'bg-pat-danger' : 'bg-pat-badge-neutral-bg'
              }`} />
              <div>
                <span className="text-sm text-pat-text-primary font-medium">{svc.service}</span>
                {svc.version && <span className="text-xs text-pat-text-muted ml-2">v{svc.version}</span>}
                {svc.details && <div className="text-xs text-pat-text-muted mt-0.5">{svc.details}</div>}
              </div>
            </div>
            <div className="flex items-center gap-4">
              {svc.latency_ms !== undefined && svc.latency_ms > 0 && (
                <span className="text-xs text-pat-text-muted">{svc.latency_ms}ms</span>
              )}
              <StatusBadge status={svc.status} />
            </div>
          </div>
        ))}
        {services.length === 0 && <div className="text-sm text-pat-text-muted text-center py-8">No health data available</div>}
      </div>
    </div>
  );
}
