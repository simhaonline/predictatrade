"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { IconBrain, IconHeartbeat, IconRefresh } from "@tabler/icons-react";
import { ProvidersCRUD } from "@/components/admin/providers-crud";
import {
  fetchAIModels,
  activateModel,
  deactivateModel,
  type AIModel,
} from "@/lib/admin-ai-providers-api";

function statusChip(status?: string) {
  const map: Record<string, string> = {
    ACTIVE: "bg-pat-success/15 text-pat-success",
    TRAINING: "bg-pat-warning/15 text-pat-warning",
    INACTIVE: "bg-gray-100 text-gray-500",
    ARCHIVED: "bg-gray-100 text-gray-500",
  };
  return (
    <span
      className={`text-xs px-2 py-0.5 rounded-full ${
        map[status ?? ""] ?? "bg-gray-100 text-gray-500"
      }`}
    >
      {status ?? "UNKNOWN"}
    </span>
  );
}

export default function AdminAiProvidersPage() {
  const queryClient = useQueryClient();
  const { data: models, isLoading, isFetching, refetch } = useQuery<AIModel[]>({
    queryKey: ["ai-models-providers"],
    queryFn: fetchAIModels,
    refetchInterval: 20000,
  });

  const toggleMutation = useMutation({
    mutationFn: async (m: AIModel) => {
      if (m.status === "ACTIVE") return deactivateModel(m.id);
      return activateModel(m.id);
    },
    onSuccess: (_d, m: AIModel) => {
      queryClient.invalidateQueries({ queryKey: ["ai-models-providers"] });
      toast.success(
        m.status === "ACTIVE"
          ? "Model deactivated — engine falls back to next ACTIVE model of the same type"
          : "Model activated",
      );
    },
    onError: () => toast.error("Failed to update model activation state"),
  });

  const activeCount = (models ?? []).filter((m) => m.status === "ACTIVE").length;
  const trainingCount = (models ?? []).filter((m) => m.status === "TRAINING").length;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">AI Providers</h1>
          <p className="text-sm text-pat-text-secondary mt-1">
            Inference provider registry + model activation — the live control surface
            for which models score predictions.
          </p>
        </div>
        <button
          onClick={() => refetch()}
          className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text hover:bg-pat-bg-surface-secondary"
        >
          <IconRefresh size={14} className={isFetching ? "animate-spin" : ""} /> Refresh
        </button>
      </div>

      {/* Summary strip */}
      <div className="grid grid-cols-3 gap-3">
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-3">
          <div className="text-xs text-pat-text-muted">Active models</div>
          <div className="text-lg font-semibold text-pat-success">{activeCount}</div>
        </div>
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-3">
          <div className="text-xs text-pat-text-muted">Training</div>
          <div className="text-lg font-semibold text-pat-warning">{trainingCount}</div>
        </div>
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-3">
          <div className="text-xs text-pat-text-muted">Registered providers</div>
          <div className="text-lg font-semibold text-pat-text-primary">
            {(models ?? []).length}
          </div>
        </div>
      </div>

      {/* LIVE: provider registry CRUD */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconHeartbeat size={16} /> Provider Registry (LIVE)
        </h2>
        <ProvidersCRUD />
      </div>

      {/* LIVE: model activation */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconBrain size={16} /> Configured Models (LIVE)
        </h2>
        {isLoading && <div className="text-xs text-pat-text-muted">Loading models...</div>}
        {!isLoading && (!models || models.length === 0) && (
          <div className="text-xs text-pat-text-muted">
            No models registered yet. The ai.models registry fills when the
            research pipeline registers a trained artifact (via the model
            registration job). The activation controls below activate once
            models exist. Provider connectivity is independent — test it above.
          </div>
        )}
        <div className="space-y-2">
          {models?.map((m) => (
            <div
              key={m.id}
              className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2"
            >
              <div className="min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-xs text-pat-text-primary font-medium">{m.name}</span>
                  <span className="text-xs text-pat-text-muted">v{m.version ?? "1"}</span>
                  {m.model_type && (
                    <span className="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded bg-pat-bg-surface text-pat-text-muted">
                      {m.model_type}
                    </span>
                  )}
                </div>
                {m.metrics && Object.keys(m.metrics).length > 0 && (
                  <div className="text-[10px] text-pat-text-muted font-mono mt-0.5 truncate">
                    {Object.entries(m.metrics)
                      .slice(0, 4)
                      .map(([k, v]) => `${k}=${typeof v === "number" ? Number(v).toFixed(4) : String(v)}`)
                      .join("  ")}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-3 shrink-0">
                {statusChip(m.status)}
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
    </div>
  );
}