"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconBolt, IconLock } from "@tabler/icons-react";

const strategies = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"];
interface Entitlements { code: string; selected_strategies?: string[]; allowed_strategies?: string[]; }

export default function UserStrategiesPage() {
  const query = useQuery({ queryKey: ["subscription-entitlements"], queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data as Entitlements });
  if (query.isLoading) return <div className="text-sm text-pat-text-secondary">Loading strategy preferences...</div>;
  if (query.isError) return <div className="rounded border border-pat-danger/30 p-4 text-sm text-pat-danger">Strategy preferences are unavailable.</div>;
  const allowed = query.data?.allowed_strategies ?? [];
  const selected = query.data?.selected_strategies ?? [];
  return <div className="space-y-6"><div><h1 className="text-xl font-bold text-pat-text-primary">Strategy Preferences</h1><p className="mt-1 text-sm text-pat-text-secondary">Your {query.data?.code ?? "current"} subscription controls which strategies may be selected. Plan changes are managed under Plans & Subscription.</p></div><div className="grid grid-cols-1 gap-3 md:grid-cols-2">{strategies.map((strategy) => { const enabled = allowed.includes(strategy); const active = selected.includes(strategy); return <div key={strategy} className="flex items-center justify-between rounded-lg border border-pat-border bg-pat-bg-surface p-4"><div className="flex items-center gap-3"><IconBolt size={18} className={active ? "text-pat-success" : "text-pat-text-muted"}/><span className="text-sm text-pat-text-primary">{strategy.replaceAll("_", " ")}</span></div>{enabled ? <span className="text-xs text-pat-success">{active ? "Selected" : "Available"}</span> : <span className="flex items-center gap-1 text-xs text-pat-text-muted"><IconLock size={14}/> Locked</span>}</div>; })}</div></div>;
}
