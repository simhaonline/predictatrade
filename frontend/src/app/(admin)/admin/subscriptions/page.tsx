"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { fetchSubscriptionPayments, fetchSubscriptionRefunds, fetchSubscriptionChargebacks, fetchSubscriptionCoupons, fetchSubscriptionProvider } from "@/lib/admin-api";
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

interface PaymentRow {
  id: string;
  user_id: string;
  provider: string;
  amount: string | number;
  currency: string;
  payment_type: string;
  status: string;
  processed_at: string | null;
}

interface RefundRow {
  id: string;
  payment_id: string;
  amount: string | number;
  currency: string;
  reason: string;
  status: string;
  provider_refund_id: string | null;
  processed_at: string | null;
  created_at: string;
}

interface CouponRow {
  id: string;
  code: string;
  description: string | null;
  discount_type: string;
  discount_value: string | number;
  currency: string;
  max_redemptions: number | null;
  redemption_count: number;
  active: boolean;
  valid_from: string | null;
  valid_until: string | null;
}

type Tab = "subscriptions" | "invoices" | "payments" | "refunds" | "chargebacks" | "coupons" | "provider";

function DegradedNote({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
      <div className="text-xs text-pat-text-secondary">{children}</div>
    </div>
  );
}

function fmtDate(v: string | null | undefined) {
  return v ? format(new Date(v), "MMM d, yyyy HH:mm") : "—";
}

export default function AdminSubscriptionsPage() {
  const [page, setPage] = useState(1);
  const [tab, setTab] = useState<Tab>("subscriptions");

  const subsQ = useQuery<{ items: Subscription[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-subscriptions", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/subscriptions?page=${page}&limit=20`);
      return res.data as { items: Subscription[]; total: number; page: number; limit: number };
    },
    enabled: tab === "subscriptions",
  });

  const paymentsQ = useQuery<{ items: PaymentRow[] }>({
    queryKey: ["admin-sub-payments"],
    queryFn: fetchSubscriptionPayments,
    enabled: tab === "payments",
  });

  const refundsQ = useQuery<{ items: RefundRow[]; note?: string }>({
    queryKey: ["admin-sub-refunds"],
    queryFn: fetchSubscriptionRefunds,
    enabled: tab === "refunds",
  });

  const chargebacksQ = useQuery<{ items: unknown[]; note?: string }>({
    queryKey: ["admin-sub-chargebacks"],
    queryFn: fetchSubscriptionChargebacks,
    enabled: tab === "chargebacks",
  });

  const couponsQ = useQuery<{ items: CouponRow[]; note?: string }>({
    queryKey: ["admin-sub-coupons"],
    queryFn: fetchSubscriptionCoupons,
    enabled: tab === "coupons",
  });

  const providerQ = useQuery<{ provider: string | null; configured: boolean; note?: string }>({
    queryKey: ["admin-sub-provider"],
    queryFn: fetchSubscriptionProvider,
    enabled: tab === "provider",
  });

  const subsCols: DataTableColumn<Subscription>[] = [
    { key: "user_email", header: "User", cell: (row) => <span className="text-sm text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-sm text-pat-text-primary">{row.plan_code === "BASIC" ? "Legacy" : row.plan_name || "—"}</span> },
    { key: "monthly_price", header: "Fee", cell: (row) => <span className="text-xs text-pat-text-secondary">${Number(row.monthly_price || 0).toFixed(0)}/mo{row.annual_price ? ` · $${Number(row.annual_price).toFixed(0)}/yr` : ""}</span> },
    { key: "billing_cycle", header: "Cycle", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.billing_cycle || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "current_period_start", header: "Period Start", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_start ? format(new Date(row.current_period_start), "MMM d, yyyy") : "—"}</span> },
    { key: "current_period_end", header: "Period End", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_end ? format(new Date(row.current_period_end), "MMM d, yyyy") : "—"}</span> },
    { key: "auto_renew", header: "Auto-Renew", cell: (row) => <span className={`text-xs ${row.auto_renew ? "text-pat-success" : "text-pat-text-muted"}`}>{row.auto_renew ? "Yes" : "No"}</span> },
  ];

  const paymentsCols: DataTableColumn<PaymentRow>[] = [
    { key: "user_id", header: "User", cell: (row) => <span className="text-xs text-pat-text-primary font-mono">{row.user_id.slice(0, 8)}</span> },
    { key: "provider", header: "Provider", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.provider}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="text-xs text-pat-text-primary">{(Number(row.amount) || 0).toFixed(2)} {row.currency}</span> },
    { key: "payment_type", header: "Type", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.payment_type}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "processed_at", header: "Processed", cell: (row) => <span className="text-xs text-pat-text-muted">{fmtDate(row.processed_at)}</span> },
  ];

  const refundsCols: DataTableColumn<RefundRow>[] = [
    { key: "payment_id", header: "Payment", cell: (row) => <span className="text-xs text-pat-text-primary font-mono">{row.payment_id.slice(0, 8)}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="text-xs text-pat-text-primary">{(Number(row.amount) || 0).toFixed(2)} {row.currency}</span> },
    { key: "reason", header: "Reason", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.reason}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "processed_at", header: "Processed", cell: (row) => <span className="text-xs text-pat-text-muted">{fmtDate(row.processed_at)}</span> },
  ];

  const couponsCols: DataTableColumn<CouponRow>[] = [
    { key: "code", header: "Code", cell: (row) => <span className="text-xs text-pat-text-primary font-mono">{row.code}</span> },
    { key: "discount", header: "Discount", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.discount_type === "PERCENTAGE" ? `${row.discount_value}%` : `${(Number(row.discount_value) || 0).toFixed(2)} ${row.currency}`}</span> },
    { key: "active", header: "Active", cell: (row) => <StatusBadge status={row.active ? "active" : "inactive"} /> },
    { key: "redemption_count", header: "Redeemed", cell: (row) => <span className="text-xs text-pat-text-muted">{row.redemption_count}{row.max_redemptions ? ` / ${row.max_redemptions}` : ""}</span> },
    { key: "valid_until", header: "Valid Until", cell: (row) => <span className="text-xs text-pat-text-muted">{fmtDate(row.valid_until)}</span> },
  ];

  const totalPages = subsQ.data?.total ? Math.ceil(subsQ.data.total / 20) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Subscription Management</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Manage all user subscriptions and billing records.</p>
      </div>

      <div className="flex gap-2 flex-wrap">
        {(["subscriptions", "invoices", "payments", "refunds", "chargebacks", "coupons", "provider"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`text-xs px-3 py-1.5 rounded transition-colors capitalize ${tab === t ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {t === "provider" ? "Provider Refs" : t}
          </button>
        ))}
      </div>

      {tab === "subscriptions" && (
        <DataTable data={subsQ.data?.items || []} columns={subsCols} loading={subsQ.isLoading} error={subsQ.error as Error | null} onRetry={() => subsQ.refetch()} />
      )}

      {tab === "invoices" && (
        <DegradedNote>
          Subscription invoices are not available here — see Billing &amp; Payouts → Invoices. No backend
          subscription-invoice endpoint is wired to this tab.
        </DegradedNote>
      )}

      {tab === "payments" && (
        paymentsQ.isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading payments…</div>
        ) : paymentsQ.error ? (
          <DegradedNote>Degraded — payments endpoint returned an error. Showing no data rather than fabricating records. {(paymentsQ.error as Error).message}</DegradedNote>
        ) : (paymentsQ.data?.items?.length ?? 0) === 0 ? (
          <DegradedNote>No payments recorded.</DegradedNote>
        ) : (
          <DataTable data={paymentsQ.data?.items || []} columns={paymentsCols} loading={false} error={null} onRetry={() => paymentsQ.refetch()} />
        )
      )}

      {tab === "refunds" && (
        refundsQ.isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading refunds…</div>
        ) : refundsQ.error ? (
          <DegradedNote>Degraded — refunds endpoint returned an error. Showing no data rather than fabricating records. {(refundsQ.error as Error).message}</DegradedNote>
        ) : (refundsQ.data?.items?.length ?? 0) === 0 ? (
          <DegradedNote>{refundsQ.data?.note || "No refunds recorded."}</DegradedNote>
        ) : (
          <DataTable data={refundsQ.data?.items || []} columns={refundsCols} loading={false} error={null} onRetry={() => refundsQ.refetch()} />
        )
      )}

      {tab === "chargebacks" && (
        chargebacksQ.isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading chargebacks…</div>
        ) : chargebacksQ.error ? (
          <DegradedNote>Degraded — chargebacks endpoint returned an error. {(chargebacksQ.error as Error).message}</DegradedNote>
        ) : (
          <DegradedNote>{chargebacksQ.data?.note || "No chargebacks recorded."}</DegradedNote>
        )
      )}

      {tab === "coupons" && (
        couponsQ.isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading coupons…</div>
        ) : couponsQ.error ? (
          <DegradedNote>Degraded — coupons endpoint returned an error. Showing no data rather than fabricating records. {(couponsQ.error as Error).message}</DegradedNote>
        ) : (couponsQ.data?.items?.length ?? 0) === 0 ? (
          <DegradedNote>{couponsQ.data?.note || "No coupons configured."}</DegradedNote>
        ) : (
          <DataTable data={couponsQ.data?.items || []} columns={couponsCols} loading={false} error={null} onRetry={() => couponsQ.refetch()} />
        )
      )}

      {tab === "provider" && (
        providerQ.isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading provider reference…</div>
        ) : providerQ.error ? (
          <DegradedNote>Degraded — provider endpoint returned an error. {(providerQ.error as Error).message}</DegradedNote>
        ) : (
          <DegradedNote>{providerQ.data?.note || "No payment provider configured."}</DegradedNote>
        )
      )}

      {tab === "subscriptions" && totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-secondary">Page {page} of {totalPages}</span>
          <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary rounded disabled:opacity-30">Next</button>
        </div>
      )}
    </div>
  );
}
