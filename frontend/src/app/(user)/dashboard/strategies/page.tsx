"use client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import { toast } from "sonner";
import { IconBolt } from "@tabler/icons-react";

interface Plan {
  id: string;
  name: string;
  description: string;
  monthly_price: string;
  status: string;
  strategy_ids: string[];
}

interface UserSubscription {
  id: string;
  plan_id: string;
  status: string;
}

export default function UserStrategiesPage() {
  const queryClient = useQueryClient();

  const { data: plans, isLoading: plansLoading } = useQuery<Plan[]>({
    queryKey: ["plans"],
    queryFn: async () => {
      const res = await customInstance.get("/plans");
      return (res.data as Plan[]) || [];
    },
  });

  const { data: subscriptions } = useQuery<UserSubscription[]>({
    queryKey: ["user-subscriptions"],
    queryFn: async () => {
      const res = await customInstance.get("/subscriptions");
      return (res.data as UserSubscription[]) || [];
    },
  });

  const subscribeMutation = useMutation({
    mutationFn: async ({ planId, strategyIds }: { planId: string; strategyIds: string }) => {
      await customInstance.post("/subscriptions", { planId, strategyIds });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["user-subscriptions"] });
      toast.success("Strategy preference updated");
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : "Failed to update strategy preference";
      toast.error(msg);
    },
  });

  const subscribedPlanIds = (subscriptions ?? []).map((s) => s.plan_id);

  if (plansLoading) return <div className="text-sm text-pat-text-secondary">Loading strategies...</div>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Strategy Preferences</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Choose your trading strategy subscription plan.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {(plans ?? []).map((plan) => {
          const isSubscribed = subscribedPlanIds.includes(plan.id);
          return (
            <div key={plan.id} className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <IconBolt size={18} className={isSubscribed ? "text-pat-success" : "text-pat-text-muted"} />
                  <span className="text-sm font-semibold text-pat-text-primary">{plan.name}</span>
                </div>
                <StatusBadge status={isSubscribed ? "ACTIVE" : "INACTIVE"} size="sm" />
              </div>
              <p className="text-xs text-pat-text-muted mb-3">{plan.description || "Trading strategy plan"}</p>
              <div className="text-lg font-bold text-pat-text-primary mb-3">${parseFloat(plan.monthly_price ?? "0").toFixed(2)}<span className="text-xs text-pat-text-muted font-normal">/month</span></div>
              {plan.strategy_ids && plan.strategy_ids.length > 0 && (
                <div className="mb-3">
                  <div className="text-xs text-pat-text-muted mb-1">Included Strategies:</div>
                  <div className="flex flex-wrap gap-1">
                    {plan.strategy_ids.map((sid) => (
                      <span key={sid} className="text-xs bg-pat-bg-surface-secondary text-pat-text-primary px-2 py-0.5 rounded">{sid}</span>
                    ))}
                  </div>
                </div>
              )}
              <button
                onClick={() => subscribeMutation.mutate({ planId: plan.id, strategyIds: "STANDARD_SCALPING,ULTRA_SCALPING" })}
                disabled={isSubscribed || subscribeMutation.isPending}
                className={`text-xs px-3 py-1.5 rounded transition-colors ${
                  isSubscribed
                    ? "bg-pat-bg-surface-secondary text-pat-text-muted cursor-default"
                    : "bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                }`}
              >
                {isSubscribed ? "Subscribed" : subscribeMutation.isPending ? "Subscribing..." : "Subscribe"}
              </button>
            </div>
          );
        })}
      </div>

      {(!plans || plans.length === 0) && !plansLoading && (
        <div className="text-center py-12 border border-pat-border rounded-lg bg-pat-bg-surface/50">
          <div className="text-pat-text-muted text-sm">No plans available. Please contact support.</div>
        </div>
      )}
    </div>
  );
}
