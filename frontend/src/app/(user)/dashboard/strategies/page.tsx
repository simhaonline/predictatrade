"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconBolt, IconLock, IconDeviceFloppy } from "@tabler/icons-react";
import { DegradedNote } from "@/components/ui/tabs";

const STRATEGIES = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"];
interface Entitlements { code: string; selected_strategies?: string[]; allowed_strategies?: string[]; }

export default function UserStrategiesPage() {
  const query = useQuery<Entitlements>({
    queryKey: ["subscription-entitlements"],
    queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data as Entitlements,
  });

  const [local, setLocal] = useState<string[] | null>(null);
  const [saved, setSaved] = useState(false);

  if (query.isLoading) return <div className="text-sm text-pat-text-secondary">Loading strategy preferences…</div>;
  if (query.isError) return <div className="rounded border border-pat-danger/30 p-4 text-sm text-pat-danger">Strategy preferences are unavailable.</div>;

  const allowed = query.data?.allowed_strategies ?? [];
  const initial = query.data?.selected_strategies ?? allowed;
  const selected = local ?? initial;

  const toggle = (s: string) => {
    if (!allowed.includes(s)) return;
    setSaved(false);
    setLocal((prev) => {
      const cur = prev ?? initial;
      return cur.includes(s) ? cur.filter((x) => x !== s) : [...cur, s];
    });
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
                <span className="text-sm text-pat-text-primary">{strategy.replaceAll("_", " ")}</span>
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

      <DegradedNote>
        The subscription service exposes <strong>GET /subscriptions/entitlements</strong> and <strong>POST /subscriptions</strong> but no
        PATCH endpoint to persist per-strategy enable/disable. Your toggles above are applied locally for this session only and are
        <strong> not yet saved to the server</strong>. A backend mutation endpoint is required to make selections authoritative.
      </DegradedNote>

      <div className="flex items-center gap-3">
        <button
          onClick={() => { setSaved(true); }}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground"
        >
          <IconDeviceFloppy size={16} /> Save selection (local)
        </button>
        {saved && <span className="text-xs text-pat-success">Saved locally only — pending backend support.</span>}
      </div>
    </div>
  );
}
