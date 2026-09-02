"use client";
import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconBolt, IconLock, IconDeviceFloppy } from "@tabler/icons-react";
import { toast } from "sonner";
import { strategyLabel } from "@/lib/strategy-labels";

const STRATEGIES = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB", "ATEN", "ARCANIST"];
interface Entitlements { code: string; selected_strategies?: string[]; allowed_strategies?: string[]; max_active_strategy_slots?: number; }
// Free/preview plans: all strategies viewable in the dashboard (selection gated separately),
// live.predictatrade.com is time-gated 11:00–13:00 broker time (GMT+3) / 08:00–10:00 UTC.
const VIEWING_ALL_PLANS = new Set(["FREE", "TRIAL"]);

export default function UserStrategiesPage() {
  const queryClient = useQueryClient();
  const query = useQuery<Entitlements>({
    queryKey: ["subscription-entitlements"],
    queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data as Entitlements,
  });

  const [local, setLocal] = useState<string[] | null>(null);
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);

  if (query.isLoading) return <div className="text-sm text-pat-text-secondary">Loading strategy preferences…</div>;
  if (query.isError) return <div className="rounded border border-pat-danger/30 p-4 text-sm text-pat-danger">Strategy preferences are unavailable.</div>;

  const code = query.data?.code;
  const allowed = VIEWING_ALL_PLANS.has(code ?? "") || !query.data?.allowed_strategies
    ? ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB"]  // view-only set
    : (query.data?.allowed_strategies ?? query.data?.selected_strategies ?? ["STANDARD_SCALPING"]);
  const initial = query.data?.selected_strategies ?? query.data?.allowed_strategies ?? ["STANDARD_SCALPING"];
  const selected = local ?? initial;

  const toggle = (s: string) => {
    if (!allowed.includes(s)) return;
    setSaved(false);
    setLocal((prev) => {
      const cur = prev ?? initial;
      return cur.includes(s) ? cur.filter((x) => x !== s) : [...cur, s];
    });
  };

  const onSave = async () => {
    setSaving(true);
    try {
      // P1-FE1 fix: use the authenticated axios instance (token lives in
      // memory+cookie, not localStorage — the raw fetch silently failed).
      await customInstance.patch("/subscriptions/strategies", {
        selectedStrategies: selected,
      });
      setLocal(null); // snap back to server state
      setSaved(true);
      queryClient.invalidateQueries({ queryKey: ["subscription-entitlements"] });
      toast.success("Strategy preferences saved.");
    } catch (e) {
      setSaved(false);
      const status = (e as { response?: { status?: number } })?.response?.status;
      const reason = (e as { response?: { data?: { message?: string } } })?.response?.data?.message;
      if (status === 400 && reason === "STRATEGY_NOT_ENTITLED") {
        toast.error("One or more selected strategies are not included in your plan.");
      } else if (status === 400 && reason === "plan_strategy_limit") {
        toast.error(`Your ${code ?? ""} plan allows at most ${query.data?.max_active_strategy_slots ?? "a limited number of"} active strategies.`);
      } else if (status === 400 && reason === "AT_LEAST_ONE_STRATEGY_REQUIRED") {
        toast.error("Select at least one strategy.");
      } else {
        toast.error("Could not save strategy preferences.");
      }
      queryClient.invalidateQueries({ queryKey: ["subscription-entitlements"] });
      setLocal(null); // discard un-entitled local edits
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Strategy Preferences</h1>
        <p className="mt-1 text-sm text-pat-text-secondary">
          Your {query.data?.code ?? "current"} subscription controls which strategies may be selected. Plan changes are managed under Plans & Subscription.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {STRATEGIES.map((strategy) => {
          const enabled = allowed.includes(strategy);
          const active = selected.includes(strategy);
          return (
            <div key={strategy} className="flex items-center justify-between rounded-lg border border-pat-border bg-pat-bg-surface p-4">
              <div className="flex items-center gap-3">
                <IconBolt size={18} className={active ? "text-pat-success" : "text-pat-text-muted"} />
                <span className="text-sm text-pat-text-primary">{strategyLabel(strategy)}</span>
              </div>
              {enabled ? (
                <button
                  onClick={() => toggle(strategy)}
                  role="switch"
                  aria-checked={active}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    active ? "bg-pat-success" : "bg-pat-bg-surface-secondary border border-pat-border"
                  }`}
                >
                  <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${active ? "translate-x-6" : "translate-x-1"}`} />
                </button>
              ) : (
                <span className="flex items-center gap-1 text-xs text-pat-text-muted"><IconLock size={14} /> Locked</span>
              )}
            </div>
          );
        })}
      </div>

      <div className="flex items-center gap-3">
        <button
          onClick={onSave}
          disabled={saving}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground disabled:opacity-50"
        >
          <IconDeviceFloppy size={16} /> {saving ? "Saving…" : "Save selection"}
        </button>
        {saved && <span className="text-xs text-pat-success">Saved to server.</span>}
      </div>
    </div>
  );
}