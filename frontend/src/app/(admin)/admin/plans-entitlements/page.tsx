"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { fetchEntitlements, updatePlan, exportRowsToCsv } from "@/lib/admin-commercial-api";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import { IconSettings, IconDownload, IconAlertTriangle } from "@tabler/icons-react";

interface Plan {
  id?: string;
  code?: string;
  name?: string;
  plan_name?: string;
  monthly_price?: string | number;
  annual_price?: string | number | null;
  billing_cycle?: string;
  features?: string[] | Record<string, unknown>;
  strategies?: string[];
  strategy_availability?: string[];
  status?: string;
  is_active?: boolean;
  referral_rates?: Record<string, string | number>;
  // Tier-limit columns (migration 067)
  max_signals_per_day?: number | null;
  max_active_strategy_slots?: number | null;
  feature_access_level?: string | null;
  per_trade_risk_pct?: string | number | null;
  daily_loss_cap_pct?: string | number | null;
  weekly_loss_cap_pct?: string | number | null;
  monthly_loss_cap_pct?: string | number | null;
  monthly_profit_target_pct?: string | number | null;
}

interface Entitlement {
  plan_code?: string;
  plan_name?: string;
  feature_key?: string;
  feature?: string;
  enabled?: boolean;
  limit?: number | string | null;
}

// Shape returned by GET /subscriptions/entitlements (caller-scoped object)
interface EntitlementsContext {
  code?: string;
  name?: string;
  max_active_strategy_slots?: number | null;
  max_signals_per_day?: number | null;
  feature_access_level?: string | null;
  allowed_strategies?: string[];
  selected_strategies?: string[];
  entitlements?: Record<string, unknown>;
}

export default function AdminPlansEntitlementsPage() {
  const [tab, setTab] = useState<"plans" | "entitlements">("plans");
  const queryClient = useQueryClient();
  const [editPlan, setEditPlan] = useState<Plan | null>(null);
  const [editForm, setEditForm] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);

  const plansQ = useQuery<Plan[]>({
    queryKey: ["commercial-plans-ent"],
    queryFn: async () => (await customInstance.get("/plans")).data as Plan[],
    enabled: tab === "plans",
  });

  const entitlementsQ = useQuery<EntitlementsContext | Entitlement[]>({
    queryKey: ["entitlements"],
    queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data,
    enabled: tab === "entitlements",
  });

  const savePlan = useMutation({
    mutationFn: async (payload: { id: string; data: Record<string, unknown> }) => {
      return updatePlan(payload.id, payload.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["commercial-plans-ent"] });
      toast.success("Plan saved");
      setEditPlan(null);
      setSaving(false);
    },
    onError: (err: unknown) => {
      setSaving(false);
      toast.error(err instanceof Error ? err.message : "Failed to save plan");
    },
  });

  const openEdit = (plan: Plan) => {
    setEditPlan(plan);
    setEditForm({
      name: String(plan.name ?? plan.plan_name ?? ""),
      monthly_price: String(plan.monthly_price ?? ""),
      annual_price: String(plan.annual_price ?? ""),
      billing_cycle: String(plan.billing_cycle ?? ""),
    });
  };

  const planCols: DataTableColumn<Plan>[] = [
    { key: "code", header: "Code", cell: (row) => <span className="text-xs font-mono text-pat-text-muted">{row.code || row.plan_name || "—"}</span> },
    { key: "name", header: "Name", cell: (row) => <span className="text-sm text-pat-text-primary">{row.name ?? row.plan_name ?? "—"}</span> },
    { key: "price", header: "Price", cell: (row) => <span className="text-xs text-pat-text-secondary">${Number(row.monthly_price || 0).toFixed(0)}/mo{row.annual_price ? ` · $${Number(row.annual_price).toFixed(0)}/yr` : ""}</span> },
    { key: "billing_cycle", header: "Cycle", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.billing_cycle || "—"}</span> },
    { key: "strategies", header: "Strategies", cell: (row) => (
      <div className="flex flex-wrap gap-1">
        {(row.strategy_availability ?? row.strategies ?? []).map((s) => (
          <span key={s} className="text-[10px] px-1.5 py-0.5 rounded bg-pat-bg-surface-secondary text-pat-text-secondary">{s}</span>
        ))}
      </div>
    )},
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status ?? (row.is_active ? "active" : "inactive")} /> },
    { key: "slots", header: "Strategy Slots", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.max_active_strategy_slots ?? "—"}</span> },
    { key: "signals_day", header: "Signals/Day", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.max_signals_per_day ?? "unlimited"}</span> },
    { key: "features", header: "Feature Tier", cell: (row) => <span className="text-xs font-mono text-pat-text-muted">{row.feature_access_level ?? "core"}</span> },
    { key: "risk", header: "Risk/Trade", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.per_trade_risk_pct != null ? `${Number(row.per_trade_risk_pct).toFixed(2)}%` : "—"}</span> },
    { key: "actions", header: "Actions", cell: (row) => (
      <button onClick={() => openEdit(row)} className="text-xs bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary px-2 py-1 rounded transition-colors">
        Edit
      </button>
    )},
  ];

  // GET /subscriptions/entitlements returns a SINGLE OBJECT for the calling
  // admin (their own entitlement context), not an array. Flatten its
  // `entitlements` map into display rows and show tier limits as a card.
  const ent = entitlementsQ.data && !Array.isArray(entitlementsQ.data) ? entitlementsQ.data : null;
  const entitlementRows: Entitlement[] = Array.isArray(entitlementsQ.data)
    ? (entitlementsQ.data as Entitlement[])
    : ent?.entitlements
      ? Object.entries(ent.entitlements).map(([featureKey, value]) => {
          const v = value as { enabled?: boolean; limit?: number | string | null } | string | number | boolean | null;
          if (v && typeof v === "object") {
            return { feature_key: featureKey, enabled: Boolean(v.enabled), limit: v.limit ?? null };
          }
          return { feature_key: featureKey, enabled: Boolean(v), limit: typeof v === "number" ? v : null };
        })
      : [];

  const entitlementCols: DataTableColumn<Entitlement>[] = [
    { key: "plan", header: "Plan", cell: () => <span className="text-sm text-pat-text-primary">{ent?.code ?? "—"}</span> },
    { key: "feature", header: "Feature", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.feature_key ?? row.feature ?? "—"}</span> },
    { key: "enabled", header: "Enabled", cell: (row) => <StatusBadge status={row.enabled ? "active" : "inactive"} /> },
    { key: "limit", header: "Limit", cell: (row) => <span className="text-xs text-pat-text-muted">{row.limit ?? "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Plans & Entitlements</h1>
          <p className="text-sm text-pat-text-secondary mt-1">Commercial plan catalog and subscription entitlements.</p>
        </div>
        <button onClick={() => exportRowsToCsv((plansQ.data ?? []) as unknown as Record<string, unknown>[], "plans.csv")} className="flex items-center gap-1 text-xs bg-pat-bg-surface-secondary text-pat-text-primary px-3 py-1.5 rounded hover:bg-pat-bg-surface-secondary transition-colors">
          <IconDownload size={14} /> Export CSV
        </button>
      </div>

      <div className="flex gap-2">
        {(["plans", "entitlements"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`text-xs px-3 py-1.5 rounded transition-colors capitalize ${tab === t ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {t}
          </button>
        ))}
      </div>

      {tab === "plans" && (
        <>
          <DataTable data={plansQ.data ?? []} columns={planCols} loading={plansQ.isLoading} error={plansQ.error as Error | null} onRetry={() => plansQ.refetch()} />
          {plansQ.data && plansQ.data.length === 0 && (
            <div className="text-center py-8 border border-pat-card-border rounded-lg bg-pat-card-bg text-sm text-pat-text-muted">No plans returned</div>
          )}
        </>
      )}

      {tab === "entitlements" && (
        <>
          {entitlementsQ.isLoading && <div className="text-sm text-pat-text-muted">Loading entitlements…</div>}
          {entitlementsQ.error && (
            <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
              <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
              <div className="text-xs text-pat-text-secondary">
                Degraded — entitlement endpoint (<code className="font-mono">GET /subscriptions/entitlements</code>) returned an error or is pending. Showing no data rather than fabricating entitlements.
                <div className="mt-1 text-pat-text-muted">{(entitlementsQ.error as Error).message}</div>
              </div>
            </div>
          )}
          {/* Caller's own entitlement context — tier limits from migration 067 */}
          {ent && (
            <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4">
              <div className="text-sm font-medium text-pat-text-primary mb-2">
                Calling admin context — plan: <span className="font-mono">{ent.code ?? "—"}</span>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
                <div><span className="text-pat-text-muted">Strategy slots:</span> <b className="text-pat-text-primary">{ent.max_active_strategy_slots ?? "—"}</b></div>
                <div><span className="text-pat-text-muted">Signals/day:</span> <b className="text-pat-text-primary">{ent.max_signals_per_day ?? "unlimited"}</b></div>
                <div><span className="text-pat-text-muted">Feature tier:</span> <b className="text-pat-text-primary font-mono">{ent.feature_access_level ?? "core"}</b></div>
                <div>
                  <span className="text-pat-text-muted">Allowed strategies:</span>{" "}
                  <span className="text-pat-text-secondary">{(ent.allowed_strategies ?? []).join(", ") || "—"}</span>
                </div>
              </div>
            </div>
          )}
          {entitlementRows.length > 0 && (
            <DataTable data={entitlementRows} columns={entitlementCols} loading={false} error={null} onRetry={() => entitlementsQ.refetch()} />
          )}
          {!entitlementsQ.isLoading && !entitlementsQ.error && entitlementRows.length === 0 && (
            <div className="text-center py-8 border border-pat-card-border rounded-lg bg-pat-card-bg text-sm text-pat-text-muted">No entitlements found</div>
          )}
        </>
      )}

      {editPlan && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditPlan(null)}>
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg shadow-xl max-w-md w-full mx-4 p-5" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-2 mb-3">
              <IconSettings size={18} className="text-pat-info" />
              <h3 className="text-sm font-semibold text-pat-text-primary">Edit Plan — {editPlan.code || editPlan.plan_name}</h3>
            </div>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-pat-text-muted">Name</label>
                <input className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" value={editForm.name} onChange={(e) => setEditForm({ ...editForm, name: e.target.value })} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-pat-text-muted">Monthly Price</label>
                  <input className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" value={editForm.monthly_price} onChange={(e) => setEditForm({ ...editForm, monthly_price: e.target.value })} />
                </div>
                <div>
                  <label className="text-xs text-pat-text-muted">Annual Price</label>
                  <input className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" value={editForm.annual_price} onChange={(e) => setEditForm({ ...editForm, annual_price: e.target.value })} />
                </div>
              </div>
            </div>
            <div className="mt-3 rounded-md bg-pat-info/10 border border-pat-info/20 px-3 py-2 text-[11px] text-pat-info">
              Submission writes provided fields via POST /plans/:id (name, monthly/annual price, setup fee, slots, strategies, status, billing, visibility, grace period).
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button onClick={() => setEditPlan(null)} className="px-3 py-1.5 text-xs border border-pat-border-strong rounded-md text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors">Cancel</button>
              <button
                onClick={() => { setSaving(true); savePlan.mutate({ id: String(editPlan.id ?? editPlan.code ?? ""), data: editForm }); }}
                disabled={saving}
                className="px-3 py-1.5 text-xs bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover disabled:opacity-50 transition-opacity">
                {saving ? "Saving..." : "Save"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
