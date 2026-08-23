"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { IconAlertTriangle } from "@tabler/icons-react";

interface Subscription {
  id: string;
  user_id: string;
  user_email: string;
  plan_id: string;
  plan_code: string;
  plan_name: string;
  monthly_price: string;
  annual_price: string | null;
  status: string;
  billing_cycle: string;
  current_period_start: string;
  current_period_end: string;
  auto_renew: boolean;
  created_at: string;
}

export default function AdminSubscriptionsPage() {
  const [page, setPage] = useState(1);
  const [tab, setTab] = useState<"subscriptions" | "invoices" | "payments" | "refunds" | "chargebacks" | "coupons" | "provider">("subscriptions");
  const { data, isLoading, error, refetch } = useQuery<{ items: Subscription[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-subscriptions", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/subscriptions?page=${page}&limit=20`);
      return res.data as { items: Subscription[]; total: number; page: number; limit: number };
    },
    enabled: tab === "subscriptions",
  });

  const columns: DataTableColumn<Subscription>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-sm text-pat-text-primary">{row.plan_code === "BASIC" ? "Legacy" : row.plan_name || "—"}</span> },
    { key: "monthly_price", header: "Fee", cell: (row) => <span className="text-xs text-pat-text-secondary">${Number(row.monthly_price || 0).toFixed(0)}/mo{row.annual_price ? ` · $${Number(row.annual_price).toFixed(0)}/yr` : ""}</span> },
    { key: "billing_cycle", header: "Cycle", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.billing_cycle || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "current_period_start", header: "Period Start", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_start ? format(new Date(row.current_period_start), "MMM d, yyyy") : "—"}</span> },
    { key: "current_period_end", header: "Period End", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_end ? format(new Date(row.current_period_end), "MMM d, yyyy") : "—"}</span> },
    { key: "auto_renew", header: "Auto-Renew", cell: (row) => <span className={`text-xs ${row.auto_renew ? "text-pat-success" : "text-pat-text-muted"}`}>{row.auto_renew ? "Yes" : "No"}</span> },
  ];

  const totalPages = data?.total ? Math.ceil(data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Subscription Management</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Manage all user subscriptions.</p>
      </div>

      <div className="flex gap-2 flex-wrap">
        {(["subscriptions", "invoices", "payments", "refunds", "chargebacks", "coupons", "provider"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`text-xs px-3 py-1.5 rounded transition-colors capitalize ${tab === t ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {t === "provider" ? "Provider Refs" : t}
          </button>
        ))}
      </div>

      {tab === "subscriptions" && (
        <DataTable data={data?.items || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
      )}

      {tab !== "subscriptions" && (
        <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
          <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
          <div className="text-xs text-pat-text-secondary">
            {tab === "invoices" && "Subscription invoices are not available here — see Billing & Payouts → Invoices. No backend subscription-invoice endpoint is wired to this tab."}
            {tab === "payments" && "Payments ledger is not yet available — no backend subscription-payments endpoint exists. This panel is intentionally empty."}
            {tab === "refunds" && "Refunds are not yet available — no backend refunds endpoint exists. This panel is intentionally empty."}
            {tab === "chargebacks" && "Chargebacks are not yet available — no backend chargeback endpoint exists. This panel is intentionally empty."}
            {tab === "coupons" && "Coupons are not yet available — no backend coupon endpoint exists. This panel is intentionally empty."}
            {tab === "provider" && "Provider references (Stripe/PayPal/etc.) are not yet available — no backend provider-reference endpoint exists. This panel is intentionally empty."}
          </div>
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-secondary">Page {page} of {totalPages}</span>
          <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
        </div>
      )}
    </div>
  );
}
