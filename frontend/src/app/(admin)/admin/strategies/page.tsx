"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import { toast } from "sonner";
import { IconBolt } from "@tabler/icons-react";
import { strategyLabel } from "@/lib/strategy-labels";

interface TradingState {
  trading_halted: boolean;
  signals_paused: boolean;
  active_strategies: string[];
  last_updated: string;
}

const STRATEGY_NAMES = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB"];

export default function AdminStrategiesPage() {
  const queryClient = useQueryClient();

  const { data: state, isLoading } = useQuery<TradingState>({
    queryKey: ["ops-state"],
    queryFn: async () => {
      const res = await customInstance.get("/operations/state");
      return res.data as TradingState;
    },
  });

  const enableMutation = useMutation({
    mutationFn: async (strategyId: string) => {
      await customInstance.post(`/operations/strategy/${strategyId}/enable`, { reason: "admin_enable" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ops-state"] });
      toast.success("Strategy enabled");
    },
    onError: () => toast.error("Failed to enable strategy"),
  });

  const disableMutation = useMutation({
    mutationFn: async (strategyId: string) => {
      await customInstance.post(`/operations/strategy/${strategyId}/disable`, { reason: "admin_disable" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ops-state"] });
      toast.success("Strategy disabled");
    },
    onError: () => toast.error("Failed to disable strategy"),
  });

  if (isLoading) return <div className="text-sm text-pat-text-secondary">Loading strategies...</div>;

  const activeStrategies = state?.active_strategies ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Strategy Panel</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Manage the five trading strategy engines (EQFE runs SHADOW).</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {STRATEGY_NAMES.map((name) => {
          const isActive = activeStrategies.includes(name);
          return (
            <div key={strategyLabel(name)} className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <IconBolt size={18} className={isActive ? "text-pat-success" : "text-pat-text-muted"} />
                  <span className="text-sm font-semibold text-pat-text-primary">{strategyLabel(name)}</span>
                </div>
                <StatusBadge status={isActive ? "ACTIVE" : "INACTIVE"} size="sm" />
              </div>
              <p className="text-xs text-pat-text-muted mb-4">
                {name === "STANDARD_SCALPING" && "M1/M5 + M15/M30 · Threshold 65 · Min RR 1.2 · Cooldown 15m"}
                {name === "ULTRA_SCALPING" && "M1 + M5 · Threshold 85 · Min RR 1.0 · Cooldown 15m"}
                {name === "STANDARD_SWING" && "M15/M30/H1 + H4/D1 · Threshold 55 · Min RR 1.8 · Cooldown 120m"}
                {name === "TREND_SWING" && "H1/H4 + D1/W1 · Threshold 50 · Min RR 2.5 · Cooldown 360m"}
                {name === "MARNIE_FIB" && "EQFE · Equilibrium Fibonacci Engine · H1 · SHADOW"}
              </p>
              <button
                onClick={() => isActive ? disableMutation.mutate(name) : enableMutation.mutate(name)}
                disabled={disableMutation.isPending || enableMutation.isPending}
                className={`text-xs px-3 py-1.5 rounded transition-colors ${
                  isActive
                    ? "bg-pat-danger/10 text-pat-danger hover:bg-pat-danger/20"
                    : "bg-pat-success/10 text-pat-success hover:bg-pat-success/20"
                } disabled:opacity-50`}
              >
                {isActive ? "Disable" : "Enable"}
              </button>
            </div>
          );
        })}
      </div>

      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Platform State</h2>
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm text-pat-text-secondary">Trading</span>
            <StatusBadge status={state?.trading_halted ? "SUSPENDED" : "ACTIVE"} size="sm" />
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-pat-text-secondary">Signal Generation</span>
            <StatusBadge status={state?.signals_paused ? "PAUSED" : "ACTIVE"} size="sm" />
          </div>
          {state?.last_updated && (
            <div className="flex items-center justify-between">
              <span className="text-sm text-pat-text-secondary">Last Updated</span>
              <span className="text-xs text-pat-text-muted">{new Date(state.last_updated).toLocaleString()}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
