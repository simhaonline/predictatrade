"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchFeatureFlags, updateFeatureFlag } from "@/lib/admin-api";
import { IconFlag, IconAlertTriangle } from "@tabler/icons-react";
import { toast } from "sonner";

const VALID_MODES = ["OFF", "SHADOW", "ACTIVE", "DISABLED", "UNSUPPORTED", "RESEARCH"];

interface FeatureFlag {
  id: string;
  module_name: string;
  mode: string;
  set_by: string | null;
  set_at: string | null;
  reason: string | null;
  created_at: string;
  updated_at: string;
}

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

export default function AdminFeatureFlagsPage() {
  const queryClient = useQueryClient();
  const { data, isLoading, error, refetch } = useQuery<FeatureFlag[]>({
    queryKey: ["admin-feature-flags"],
    queryFn: fetchFeatureFlags,
  });

  const mutate = useMutation({
    mutationFn: async ({ id, mode }: { id: string; mode: string }) =>
      updateFeatureFlag(id, { mode }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-feature-flags"] });
      toast.success("Feature flag updated");
    },
    onError: (err: unknown) => {
      toast.error(err instanceof Error ? err.message : "Failed to update feature flag");
    },
  });

  const flags = Array.isArray(data) ? data : [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Feature Flags &amp; Configuration</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Live PTB module flags from <code className="font-mono">trading.ptb_feature_flags</code>.
          Changing the mode updates the stored state.
        </p>
      </div>

      {error && (
        <DegradedBanner>
          Degraded — could not load feature flags from the backend. Showing no data rather than
          fabricating configuration. <span className="block mt-1 text-pat-text-muted">{(error as Error).message}</span>
        </DegradedBanner>
      )}

      {isLoading && <div className="text-sm text-pat-text-muted">Loading feature flags…</div>}

      {!isLoading && !error && flags.length === 0 && (
        <div className="text-center py-8 border border-pat-card-border rounded-lg bg-pat-card-bg text-sm text-pat-text-muted">
          No feature flags configured
        </div>
      )}

      {flags.length > 0 && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
            <IconFlag size={16} /> Flags
          </h2>
          <div className="space-y-2">
            {flags.map((f) => (
              <div
                key={f.id}
                className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2"
              >
                <div>
                  <span className="text-xs text-pat-text-primary font-mono">{f.module_name}</span>
                  <span className="text-xs text-pat-text-muted ml-2">[{f.mode}]</span>
                  <div className="text-xs text-pat-text-muted">
                    {f.reason || "No reason recorded"}
                    {f.set_by ? ` · set by ${f.set_by}` : ""}
                  </div>
                </div>
                <select
                  value={f.mode}
                  disabled={mutate.isPending}
                  onChange={(e) => mutate.mutate({ id: f.id, mode: e.target.value })}
                  className="text-xs px-2 py-1 rounded border border-pat-border bg-pat-input-bg text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary"
                >
                  {VALID_MODES.map((m) => (
                    <option key={m} value={m}>{m}</option>
                  ))}
                </select>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
