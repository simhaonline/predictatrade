"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { IconBrain, IconPlus, IconAlertTriangle } from "@tabler/icons-react";
import {
  fetchAIModels,
  activateModel,
  deactivateModel,
  type AIModel,
} from "@/lib/admin-ai-providers-api";

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

export default function AdminAiProvidersPage() {
  const queryClient = useQueryClient();
  const { data: models, isLoading } = useQuery<AIModel[]>({
    queryKey: ["ai-models-providers"],
    queryFn: fetchAIModels,
    refetchInterval: 20000,
  });

  const toggleMutation = useMutation({
    mutationFn: async (m: AIModel) => {
      if (m.status === "ACTIVE") return deactivateModel(m.id);
      return activateModel(m.id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-models-providers"] });
      toast.success("Model activation state updated");
    },
    onError: () => toast.error("Failed to update model activation state"),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">AI Providers</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Live AI/ML model activation state (source: GET /operations/ai/models). Provider-level CRUD is not yet backed by an endpoint.
        </p>
      </div>

      {/* LIVE: model activation */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconBrain size={16} /> Configured Models (LIVE)
        </h2>
        {isLoading && <div className="text-xs text-pat-text-muted">Loading models...</div>}
        {!isLoading && (!models || models.length === 0) && (
          <div className="text-xs text-pat-text-muted">No models returned from /operations/ai/models.</div>
        )}
        <div className="space-y-2">
          {models?.map((m) => (
            <div
              key={m.id}
              className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2"
            >
              <div>
                <span className="text-xs text-pat-text-primary">{m.name}</span>
                <span className="text-xs text-pat-text-muted ml-2">v{m.version ?? "1"}</span>
                {m.model_type && (
                  <span className="text-xs text-pat-text-muted ml-2">· {m.model_type}</span>
                )}
              </div>
              <div className="flex items-center gap-3">
                <span
                  className={`text-xs px-2 py-0.5 rounded-full ${
                    m.status === "ACTIVE"
                      ? "bg-green-100 text-green-700"
                      : "bg-gray-100 text-gray-500"
                  }`}
                >
                  {m.status ?? "UNKNOWN"}
                </span>
                <button
                  onClick={() => toggleMutation.mutate(m)}
                  disabled={toggleMutation.isPending}
                  className="text-xs px-3 py-1 rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50"
                >
                  {m.status === "ACTIVE" ? "Deactivate" : "Activate"}
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* DEGRADED: add provider */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconPlus size={16} /> Add Provider (Pending Backend)
        </h2>
        <DegradedBanner>
          No dedicated provider-management endpoint exists. The form below is schema-only and disabled.
          Model activation above is the only LIVE capability for this page.
        </DegradedBanner>
        <form className="mt-3 grid grid-cols-1 md:grid-cols-3 gap-3" onSubmit={(e) => e.preventDefault()}>
          <input
            disabled
            placeholder="Provider name"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60"
          />
          <input
            disabled
            placeholder="API base URL"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60"
          />
          <input
            disabled
            placeholder="Default model"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60"
          />
          <button
            disabled
            className="md:col-span-3 px-3 py-2 text-sm rounded-md bg-pat-primary text-pat-primary-foreground opacity-50 cursor-not-allowed"
          >
            Create Provider (disabled — backend pending)
          </button>
        </form>
      </div>
    </div>
  );
}
