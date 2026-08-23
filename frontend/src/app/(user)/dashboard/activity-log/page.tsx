"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconHistory, IconInfoCircle } from "@tabler/icons-react";
import { format } from "date-fns";

interface ClientEvent {
  id: string;
  event_id: string;
  actor_type: string;
  event_type: string;
  entity_type: string | null;
  entity_id: string | null;
  created_at: string;
  source_ip: string | null;
  reason: string | null;
  metadata?: Record<string, unknown>;
}

interface AuditResponse {
  items: ClientEvent[];
  total: number;
  page: number;
  limit: number;
}

export default function UserActivityLogPage() {
  const { data, isLoading, isError } = useQuery<AuditResponse>({
    queryKey: ["user-activity-log"],
    queryFn: async () => (await customInstance.get("/audit/client?limit=100")).data,
    refetchInterval: 15000,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <IconHistory className="h-7 w-7 text-pat-accent" />
        <div>
          <h1 className="text-xl font-semibold text-pat-text">Activity Log</h1>
          <p className="text-sm text-pat-text-muted">
            Security and account events for your own account only.
          </p>
        </div>
      </div>

      <div className="rounded-lg border border-pat-border bg-pat-surface p-4">
        {isLoading && (
          <p className="text-sm text-pat-text-muted">Loading your activity…</p>
        )}
        {isError && (
          <p className="text-sm text-pat-danger">
            Unable to load activity log. Please try again later.
          </p>
        )}
        {!isLoading && !isError && (!data || data.items.length === 0) && (
          <div className="flex items-center gap-2 text-sm text-pat-text-muted">
            <IconInfoCircle className="h-4 w-4" />
            No activity recorded yet.
          </div>
        )}
        {!isLoading && !isError && data && data.items.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-pat-text-muted border-b border-pat-border">
                  <th className="py-2 pr-4 font-medium">Time</th>
                  <th className="py-2 pr-4 font-medium">Event</th>
                  <th className="py-2 pr-4 font-medium">Resource</th>
                  <th className="py-2 pr-4 font-medium">Detail</th>
                </tr>
              </thead>
              <tbody>
                {data.items.map((e) => (
                  <tr key={e.id} className="border-b border-pat-border/50">
                    <td className="py-2 pr-4 whitespace-nowrap text-pat-text-secondary">
                      {e.created_at ? format(new Date(e.created_at), "yyyy-MM-dd HH:mm:ss") : "—"}
                    </td>
                    <td className="py-2 pr-4 text-pat-text-secondary">{e.event_type}</td>
                    <td className="py-2 pr-4 text-pat-text-secondary">
                      {e.entity_type || "—"}
                      {e.entity_id ? `:${e.entity_id}` : ""}
                    </td>
                    <td className="py-2 pr-4 text-pat-text-muted">
                      {e.reason || (e.metadata?.new_value ? JSON.stringify(e.metadata.new_value) : "—")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="mt-3 text-xs text-pat-text-muted">
              Showing {data.items.length} of {data.total} events.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
