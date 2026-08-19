"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { toast } from "sonner";
import { useState } from "react";
import ConfirmDialog from "@/components/admin/confirm-dialog";
import StatusBadge from "@/components/admin/status-badge";
import { IconBolt, IconAlertTriangle } from "@tabler/icons-react";

interface TradingState {
  trading_halted: boolean;
  signals_paused: boolean;
  active_strategies: string[];
  last_updated: string;
}

interface ActiveOp {
  id: string;
  operation_type: string;
  status: string;
  actor_id: string;
  reason: string;
  created_at: string;
}

export default function AdminOperationsPage() {
  const queryClient = useQueryClient();
  const [confirm, setConfirm] = useState<{ action: string; title: string; message: string; fn: () => Promise<void> } | null>(null);
  const [reason, setReason] = useState("");

  const { data: state } = useQuery<TradingState>({
    queryKey: ["ops-state"],
    queryFn: async () => (await customInstance.get("/operations/state")).data,
    refetchInterval: 15000,
  });

  const { data: activeOps } = useQuery<ActiveOp[]>({
    queryKey: ["ops-active"],
    queryFn: async () => (await customInstance.get("/operations/active")).data,
    refetchInterval: 10000,
  });

  const { data: aiModels } = useQuery({
    queryKey: ["ai-models"],
    queryFn: async () => (await customInstance.get("/operations/ai/models")).data,
  });

  const mutation = useMutation({
    mutationFn: async (fn: () => Promise<void>) => { await fn(); },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ops-state"] });
      queryClient.invalidateQueries({ queryKey: ["ops-active"] });
      toast.success("Operation completed");
      setConfirm(null);
      setReason("");
    },
    onError: () => toast.error("Operation failed"),
  });

  const actions = [
    {
      label: "Halt Trading",
      title: "Confirm Halt Trading",
      message: "This will stop all signal execution across the platform. Existing positions remain.",
      danger: true,
      fn: () => customInstance.post("/operations/halt-trading", { reason: reason || "admin_halt" }),
      disabled: state?.trading_halted,
      status: state?.trading_halted ? "halted" : "active",
    },
    {
      label: "Resume Trading",
      title: "Confirm Resume Trading",
      message: "This will resume signal execution across the platform.",
      danger: false,
      fn: () => customInstance.post("/operations/resume-trading", { reason: reason || "admin_resume" }),
      disabled: !state?.trading_halted,
      status: state?.trading_halted ? "halted" : "active",
    },
    {
      label: "Pause Signals",
      title: "Confirm Pause Signal Generation",
      message: "This will stop generating new signals. Existing signals remain active.",
      danger: true,
      fn: () => customInstance.post("/operations/pause-signals", { reason: reason || "admin_pause" }),
      disabled: state?.signals_paused,
      status: state?.signals_paused ? "paused" : "active",
    },
    {
      label: "Resume Signals",
      title: "Confirm Resume Signal Generation",
      message: "This will resume signal generation.",
      danger: false,
      fn: () => customInstance.post("/operations/resume-signals", { reason: reason || "admin_resume" }),
      disabled: !state?.signals_paused,
      status: state?.signals_paused ? "paused" : "active",
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Platform Operations</h1>
        <p className="text-sm text-pat-text-secondary mt-1">High-security trading and signal controls. All actions are audited.</p>
      </div>

      {/* Current State */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Current Platform State</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Trading</span>
            <StatusBadge status={state?.trading_halted ? "halted" : "active"} />
          </div>
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Signals</span>
            <StatusBadge status={state?.signals_paused ? "paused" : "active"} />
          </div>
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Strategies</span>
            <span className="text-xs text-pat-text-secondary">{state?.active_strategies?.length ?? 0} active</span>
          </div>
          {state?.last_updated && (
            <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
              <span className="text-xs text-pat-text-muted">Updated</span>
              <span className="text-xs text-pat-text-secondary">{new Date(state.last_updated).toLocaleString()}</span>
            </div>
          )}
        </div>
      </div>

      {/* Trading Controls */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconBolt size={16} /> Trading Controls
        </h2>
        <div className="mb-3">
          <input type="text" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Reason (optional, for audit trail)"
            className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {actions.map((action) => (
            <button key={action.label}
              onClick={() => setConfirm({ action: action.label, title: action.title, message: action.message, fn: async () => { await action.fn(); } })}
              disabled={action.disabled || mutation.isPending}
              className={`px-3 py-2 text-sm font-medium rounded-md transition-colors disabled:opacity-50 ${
                action.danger
                  ? "bg-pat-danger text-white hover:opacity-90"
                  : "bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover"
              }`}>
              {action.label}
            </button>
          ))}
        </div>
      </div>

      {/* AI Models */}
      {aiModels && Array.isArray(aiModels) && aiModels.length > 0 && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3">AI/ML Models</h2>
          <div className="space-y-2">
            {aiModels.map((model: Record<string, unknown>) => (
              <div key={String(model.id)} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
                <div>
                  <span className="text-xs text-pat-text-primary">{String(model.name ?? "Unknown")}</span>
                  <span className="text-xs text-pat-text-muted ml-2">v{String(model.version ?? "1")}</span>
                </div>
                <StatusBadge status={String(model.status ?? "unknown")} />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Active Operations History */}
      {activeOps && activeOps.length > 0 && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
            <IconAlertTriangle size={16} /> Active Operations
          </h2>
          <div className="space-y-2">
            {activeOps.map((op) => (
              <div key={op.id} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2 text-xs">
                <span className="text-pat-text-primary">{op.operation_type}</span>
                <span className="text-pat-text-muted">{op.reason}</span>
                <StatusBadge status={op.status} />
              </div>
            ))}
          </div>
        </div>
      )}

      <ConfirmDialog
        open={!!confirm}
        title={confirm?.title || ""}
        message={confirm?.message || ""}
        confirmLabel={confirm?.action || "Confirm"}
        onConfirm={() => { if (confirm) mutation.mutate(confirm.fn); }}
        onCancel={() => setConfirm(null)}
        loading={mutation.isPending}
      />
    </div>
  );
}
