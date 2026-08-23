"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";

interface HealthServiceStatus {
  service: string;
  status: 'HEALTHY' | 'DEGRADED' | 'OFFLINE' | 'UNKNOWN';
  latency_ms?: number;
  last_check: string;
  version?: string;
  details?: string;
}

export default function AdminHealthPage() {
  // Use Go engine's public system-health endpoint (no auth needed)
  const { data: goHealth, isLoading: goLoading, refetch } = useQuery({
    queryKey: ["go-system-health"],
    queryFn: async () => (await customInstance.get("/system-health")).data,
    refetchInterval: 30000,
  });

  // Also get NestJS health (public endpoint)
  const { data: nestHealth } = useQuery({
    queryKey: ["nestjs-health-public"],
    queryFn: async () => (await customInstance.get("/health")).data,
    refetchInterval: 30000,
  });

  // Build services list from both sources
  const services: HealthServiceStatus[] = [];
  const now = new Date().toISOString();

  // PostgreSQL/TimescaleDB
  services.push({
    service: 'PostgreSQL/TimescaleDB',
    status: goHealth?.postgresql?.healthy ? 'HEALTHY' : 'OFFLINE',
    last_check: now,
    details: goHealth?.timescaledb?.active ? 'TimescaleDB active' : 'TimescaleDB inactive',
  });

  // Control Plane (NestJS)
  services.push({
    service: 'Control Plane (NestJS)',
    status: nestHealth?.status === 'ok' ? 'HEALTHY' : 'OFFLINE',
    last_check: now,
    version: nestHealth?.version || 'unknown',
    details: nestHealth?.database === 'healthy' ? 'DB connected' : 'DB issue',
  });

  // Go Real-Time Engine
  services.push({
    service: 'Go Real-Time Engine',
    status: goHealth?.ready ? 'HEALTHY' : 'OFFLINE',
    last_check: now,
    details: goHealth?.ready_reason || 'Running',
  });

  // Valkey/Redis
  services.push({
    service: 'Valkey/Redis',
    status: goHealth?.valkey?.connected ? 'HEALTHY' : 'OFFLINE',
    last_check: now,
    details: goHealth?.valkey?.connected ? 'Connected' : 'Disconnected',
  });

  // Windows Agent / Master Node
  services.push({
    service: 'Windows Agent / Master Node',
    status: goHealth?.market_source?.agents_connected > 0 ? 'HEALTHY' : 'UNKNOWN',
    last_check: now,
    details: goHealth?.market_source?.master_node_connected ? 'Master node connected' : `Agents: ${goHealth?.market_source?.agents_connected ?? 0}`,
  });

  if (goLoading) return <div className="text-sm text-pat-text-secondary">Loading health status...</div>;

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
                svc.status === 'OFFLINE' ? 'bg-pat-danger' : 'bg-gray-400'
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
              <span className={`text-xs font-bold px-3 py-1 rounded-full ${
                svc.status === 'HEALTHY' ? 'bg-green-100 text-green-700 border border-green-200' :
                svc.status === 'DEGRADED' ? 'bg-yellow-100 text-yellow-700 border border-yellow-200' :
                svc.status === 'OFFLINE' ? 'bg-red-100 text-red-700 border border-red-200' :
                'bg-gray-100 text-gray-500 border border-gray-200'
              }`}>{svc.status}</span>
            </div>
          </div>
        ))}
        {services.length === 0 && <div className="text-sm text-pat-text-muted text-center py-8">No health data available</div>}
      </div>
      <div className="text-xs text-pat-text-muted text-center">
        Last updated: {now} · <button onClick={() => refetch()} className="text-pat-primary hover:underline">Refresh</button>
      </div>
    </div>
  );
}
