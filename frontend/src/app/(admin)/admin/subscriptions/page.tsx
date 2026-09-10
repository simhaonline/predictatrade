"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { fetchSubscriptionPayments, fetchSubscriptionRefunds, fetchSubscriptionChargebacks, fetchSubscriptionCoupons, fetchSubscriptionProvider, fetchSubscriptionInvoices, createSubscriptionCoupon } from "@/lib/admin-api";
import { approveEntitlement } from "@/lib/admin-commercial-api";
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
  cancelled_at?: string | null;
  cancel_reason?: string | null;
  user_status?: string;
  license_id?: string | null;
  expires_at?: string | null;
}

interface InvoiceRow {
  id: string;
  invoice_number: string | null;
  user_id: string;
  user_email: string | null;
  subscription_id: string;
  plan_name: string | null;
  amount: string | number;
  currency: string;
  status: string;
  invoice_url: string | null;
  billing_period_start: string | null;
  billing_period_end: string | null;
  created_at: string;
  paid_at: string | null;
}

interface UserNoSub {
  id: string;
  email: string;
  full_name: string;
  status: string;
  created_at: string;
  license_status: string | null;
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
  const [busyId, setBusyId] = useState<string | null>(null);
  const [showAllUsers, setShowAllUsers] = useState(false);
  const [planPick, setPlanPick] = useState<Record<string, string>>({});
  const [intervalPick, setIntervalPick] = useState<Record<string, "MONTHLY" | "ANNUAL">>({});
  const [couponForm, setCouponForm] = useState<{ code: string; description: string; discountType: "PERCENTAGE" | "FIXED"; discountValue: string; currency: string; maxRedemptions: string }>({
    code: "", description: "", discountType: "PERCENTAGE", discountValue: "", currency: "USD", maxRedemptions: "",
  });
  const [couponBusy, setCouponBusy] = useState(false);

  const completeSub = async (id: string) => {
    if (typeof window !== "undefined" && !window.confirm("Complete this INCOMPLETE subscription? This marks it ACTIVE (provisioning/entitlement confirmed) and is recorded in the audit log.")) return;
    setBusyId(id);
    try {
      await customInstance.post(`/admin/subscriptions/${id}/complete`, { reason: "Manual reconciliation — entitlement confirmed" });
      subsQ.refetch();
    } catch (e) {
      if (typeof window !== "undefined") window.alert("Failed to complete subscription: " + (e instanceof Error ? e.message : String(e)));
    } finally {
      setBusyId(null);
    }
  };

  // v1.31 unified manual approval — Approve (payment received out-of-band)
  // / Reject (cancel) for INCOMPLETE subscriptions, with a reason prompt.
  const approveSub = async (id: string, decision: "approve" | "reject") => {
    const what = decision === "approve"
      ? "APPROVE this subscription? Only click if payment was RECEIVED out-of-band (bank/USDT). It becomes ACTIVE, the billing period starts now, and the license is ensured."
      : "REJECT this subscription? It becomes CANCELLED (terminal). The user keeps their account.";
    const reason = typeof window !== "undefined" ? window.prompt(`${what}\n\nReason (recorded in the audit log):`, decision === "approve" ? "Payment received out-of-band" : "Payment not received / request refused") : null;
    if (reason === null) return;
    setBusyId(id);
    try {
      await approveEntitlement("subscription", id, decision, reason);
      subsQ.refetch();
      if (typeof window !== "undefined") window.alert(`Subscription ${decision}d successfully.`);
    } catch (e) {
      if (typeof window !== "undefined") window.alert("Approval failed: " + (e instanceof Error ? e.message : String(e)));
    } finally {
      setBusyId(null);
    }
  };

  const subsQ = useQuery<{ items: Subscription[]; total: number; page: number; limit: number }>({
    queryKey: ["admin-subscriptions", page],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/subscriptions?page=${page}&limit=20`);
      return res.data as { items: Subscription[]; total: number; page: number; limit: number };
    },
    enabled: tab === "subscriptions",
  });

  const noSubQ = useQuery<{ items: UserNoSub[]; total: number }>({
    queryKey: ["admin-users-without-sub"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/users-without-subscription");
      return res.data as { items: UserNoSub[]; total: number };
    },
    enabled: tab === "subscriptions" && showAllUsers,
  });

  const plansQ = useQuery<{ id: string; name: string; monthly_price: string; currency: string }[]>({
    queryKey: ["admin-sub-plans"],
    queryFn: async () => {
      const res = await customInstance.get("/plans");
      return res.data as { id: string; name: string; monthly_price: string; currency: string }[];
    },
  });

  const startSub = async (userId: string, email: string) => {
    const planId = planPick[userId];
    if (!planId) { if (typeof window !== "undefined") window.alert("Pick a plan first"); return; }
    if (typeof window !== "undefined" && !window.confirm(`Start a ${intervalPick[userId] || "MONTHLY"} subscription on this plan for ${email}? Use only when payment was confirmed out-of-band (bank transfer, in-person).`)) return;
    setBusyId(userId);
    try {
      await customInstance.post(`/admin/users/${userId}/start-subscription`, { planId, billingInterval: intervalPick[userId] || "MONTHLY" });
      noSubQ.refetch(); subsQ.refetch();
    } catch (e) {
      if (typeof window !== "undefined") window.alert("Failed to start subscription: " + (e instanceof Error ? e.message : String(e)));
    } finally {
      setBusyId(null);
    }
  };

  const createCouponHandler = async () => {
    if (!couponForm.code || !couponForm.discountValue) {
      if (typeof window !== "undefined") window.alert("Coupon code and discount value are required.");
      return;
    }
    setCouponBusy(true);
    try {
      await createSubscriptionCoupon({
        code: couponForm.code,
        description: couponForm.description || null,
        discountType: couponForm.discountType,
        discountValue: Number(couponForm.discountValue),
        currency: couponForm.currency || "USD",
        maxRedemptions: couponForm.maxRedemptions ? Number(couponForm.maxRedemptions) : null,
      });
      setCouponForm({ code: "", description: "", discountType: "PERCENTAGE", discountValue: "", currency: "USD", maxRedemptions: "" });
      couponsQ.refetch();
    } catch (e) {
      if (typeof window !== "undefined") window.alert("Failed to create coupon: " + (e instanceof Error ? e.message : String(e)));
    } finally {
      setCouponBusy(false);
    }
  };

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

  const invoicesQ = useQuery<{ items: InvoiceRow[]; note?: string }>({
    queryKey: ["admin-sub-invoices"],
    queryFn: fetchSubscriptionInvoices,
    enabled: tab === "invoices",
  });

  const subsCols: DataTableColumn<Subscription>[] = [
    { key: "user_email", header: "User", cell: (row) => (
      <span className="text-sm">
        <span className={row.user_status === "DELETED" ? "text-pat-text-muted line-through" : "text-pat-text-primary"}>
          {row.user_email || "—"}
        </span>
        {row.user_status === "DELETED" && (
          <span className="ml-1.5 text-[9px] px-1 py-0.5 rounded-full border border-pat-border text-pat-text-muted align-middle">
            user deleted
          </span>
        )}
      </span>
    ) },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-sm text-pat-text-primary">{row.plan_name || "—"}</span> },
    { key: "monthly_price", header: "Fee", cell: (row) => <span className="text-xs text-pat-text-secondary">${Number(row.monthly_price || 0).toFixed(0)}/mo{row.annual_price ? ` · $${Number(row.annual_price).toFixed(0)}/yr` : ""}</span> },
    { key: "billing_cycle", header: "Cycle", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.billing_cycle || "—"}</span> },
    { key: "status", header: "Status", cell: (row) => (
      <span>
        <StatusBadge status={row.status} />
        {row.status === "CANCELLED" && (row.cancel_reason || row.cancelled_at) && (
          <span className="block mt-1 text-[10px] text-pat-text-muted max-w-[180px] leading-tight">
            {row.cancel_reason
              ? row.cancel_reason
              : `cancelled ${row.cancelled_at ? format(new Date(row.cancelled_at), "MMM d, yyyy") : ""}`}
          </span>
        )}
      </span>
    ) },
    { key: "current_period_start", header: "Period Start", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_start ? format(new Date(row.current_period_start), "MMM d, yyyy") : "—"}</span> },
    { key: "current_period_end", header: "Period End", cell: (row) => <span className="text-xs text-pat-text-muted">{row.current_period_end ? format(new Date(row.current_period_end), "MMM d, yyyy") : "—"}</span> },
    { key: "expires_at", header: "Expires", cell: (row) => <span className="text-xs text-pat-text-muted">{row.expires_at ? format(new Date(row.expires_at), "MMM d, yyyy") : "—"}</span> },
    { key: "auto_renew", header: "Auto-Renew", cell: (row) => <span className={`text-xs ${row.auto_renew ? "text-pat-success" : "text-pat-text-muted"}`}>{row.auto_renew ? "Yes" : "No"}</span> },
    { key: "action", header: "Approval", cell: (row) => ["INCOMPLETE", "FAILED", "PAST_DUE"].includes(row.status) ? (
      <div className="flex items-center gap-1">
        <button onClick={() => approveSub(row.id, "approve")} disabled={busyId === row.id}
          className="text-xs px-2 py-1 rounded bg-pat-success/20 text-pat-success hover:bg-pat-success/30 disabled:opacity-40 font-medium">
          {busyId === row.id ? "…" : "Approve"}
        </button>
        <button onClick={() => approveSub(row.id, "reject")} disabled={busyId === row.id}
          className="text-xs px-2 py-1 rounded bg-pat-danger/10 text-pat-danger hover:bg-pat-danger/20 disabled:opacity-40">
          Reject
        </button>
        <button onClick={() => completeSub(row.id)} disabled={busyId === row.id}
          className="text-xs px-2 py-1 rounded bg-pat-bg-surface-secondary text-pat-text-secondary hover:bg-pat-bg-surface disabled:opacity-40"
          title="Legacy: mark ACTIVE without license sync">
          Complete
        </button>
      </div>
    ) : <span className="text-xs text-pat-text-muted">—</span> },
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

  const invoiceCols: DataTableColumn<InvoiceRow>[] = [
    { key: "invoice_number", header: "Invoice #", cell: (row) => <span className="text-xs text-pat-text-primary font-mono">{row.invoice_number || row.id.slice(0, 8)}</span> },
    { key: "user_email", header: "User", cell: (row) => <span className="text-xs text-pat-text-primary">{row.user_email || "—"}</span> },
    { key: "plan_name", header: "Plan", cell: (row) => <span className="text-xs text-pat-text-secondary">{row.plan_name || "—"}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="text-xs text-pat-text-primary">{(Number(row.amount) || 0).toFixed(2)} {row.currency}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "billing_period_start", header: "Period Start", cell: (row) => <span className="text-xs text-pat-text-muted">{fmtDate(row.billing_period_start)}</span> },
    { key: "billing_period_end", header: "Period End", cell: (row) => <span className="text-xs text-pat-text-muted">{fmtDate(row.billing_period_end)}</span> },
    { key: "created_at", header: "Issued", cell: (row) => <span className="text-xs text-pat-text-muted">{fmtDate(row.created_at)}</span> },
    { key: "invoice_url", header: "Link", cell: (row) => row.invoice_url ? (
      <a href={row.invoice_url} target="_blank" rel="noreferrer" className="text-xs text-pat-info underline">open</a>
    ) : <span className="text-xs text-pat-text-muted">—</span> },
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
        <>
          {subsQ.data?.items?.some((s) => s.status === "INCOMPLETE") && (
            <DegradedNote>
              <strong>INCOMPLETE</strong> means provisioning/billing-webhook did not complete (e.g. payment confirmed out-of-band or entitlement already granted via an ACTIVE license). Use <em>Complete</em> to reconcile it to ACTIVE — this is recorded in the audit log and never rewrites history.
            </DegradedNote>
          )}
          <div className="flex items-center justify-between flex-wrap gap-2">
            <label className="flex items-center gap-2 text-xs text-pat-text-secondary cursor-pointer select-none">
              <input type="checkbox" checked={showAllUsers} onChange={(e) => setShowAllUsers(e.target.checked)} className="accent-pat-primary" />
              Show all users without a subscription ({noSubQ.data?.total ?? "…"})
            </label>
            <span className="text-xs text-pat-text-muted">Subscriptions: {subsQ.data?.total ?? "…"}</span>
          </div>
          <DataTable data={subsQ.data?.items || []} pageSize={20} hidePager columns={subsCols} loading={subsQ.isLoading} error={subsQ.error as Error | null} onRetry={() => subsQ.refetch()} />

          {showAllUsers && (
            <div className="space-y-2">
              <div className="text-xs text-pat-text-muted">
                Registered users who never started a plan. Start a subscription only after payment was confirmed out-of-band — it activates immediately and auto-issues the plan&apos;s license.
              </div>
              <DataTable
                data={noSubQ.data?.items || []}
                pageSize={20}
                hidePager
                columns={[
                  { key: "email", header: "User", cell: (row: UserNoSub) => (
                    <div>
                      <div className="text-sm text-pat-text-primary">{row.email}</div>
                      {row.full_name && <div className="text-xs text-pat-text-muted">{row.full_name}</div>}
                    </div>
                  )},
                  { key: "status", header: "Account", cell: (row: UserNoSub) => <StatusBadge status={row.status} /> },
                  { key: "license_status", header: "License", cell: (row: UserNoSub) => row.license_status
                      ? <StatusBadge status={row.license_status} />
                      : <span className="text-xs text-pat-warning">none</span> },
                  { key: "created_at", header: "Registered", cell: (row: UserNoSub) => <span className="text-xs text-pat-text-muted">{format(new Date(row.created_at), "MMM d, yyyy")}</span> },
                  { key: "plan", header: "Plan", cell: (row: UserNoSub) => (
                    <select
                      value={planPick[row.id] || ""}
                      onChange={(e) => setPlanPick((m) => ({ ...m, [row.id]: e.target.value }))}
                      className="px-2 py-1 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border"
                    >
                      <option value="">Pick plan…</option>
                      {(plansQ.data || []).filter((p) => p.monthly_price !== undefined).map((p) => (
                        <option key={p.id} value={p.id}>{p.name}</option>
                      ))}
                    </select>
                  )},
                  { key: "interval", header: "Cycle", cell: (row: UserNoSub) => (
                    <select
                      value={intervalPick[row.id] || "MONTHLY"}
                      onChange={(e) => setIntervalPick((m) => ({ ...m, [row.id]: e.target.value as "MONTHLY" | "ANNUAL" }))}
                      className="px-2 py-1 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border"
                    >
                      <option value="MONTHLY">Monthly</option>
                      <option value="ANNUAL">Annual</option>
                    </select>
                  )},
                  { key: "action", header: "Action", cell: (row: UserNoSub) => (
                    <button
                      onClick={() => startSub(row.id, row.email)}
                      disabled={busyId === row.id || !planPick[row.id]}
                      className="text-xs px-2 py-1 rounded bg-pat-success/20 text-pat-success hover:bg-pat-success/30 disabled:opacity-40"
                    >
                      {busyId === row.id ? "Working…" : "Start Subscription"}
                    </button>
                  )},
                ] as never}
                loading={noSubQ.isLoading}
                error={noSubQ.error as Error | null}
                onRetry={() => noSubQ.refetch()}
              />
            </div>
          )}
        </>
      )}

      {tab === "invoices" && (
        invoicesQ.isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading invoices…</div>
        ) : invoicesQ.error ? (
          <DegradedNote>Degraded — invoices endpoint returned an error. {(invoicesQ.error as Error).message}</DegradedNote>
        ) : (invoicesQ.data?.items?.length ?? 0) === 0 ? (
          <DegradedNote>{invoicesQ.data?.note || "No invoices recorded."}</DegradedNote>
        ) : (
          <DataTable data={invoicesQ.data?.items || []} pageSize={20} hidePager columns={invoiceCols} loading={false} error={null} onRetry={() => invoicesQ.refetch()} />
        )
      )}
      {tab === "payments" && (
        paymentsQ.isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading payments…</div>
        ) : paymentsQ.error ? (
          <DegradedNote>Degraded — payments endpoint returned an error. Showing no data rather than fabricating records. {(paymentsQ.error as Error).message}</DegradedNote>
        ) : (paymentsQ.data?.items?.length ?? 0) === 0 ? (
          <DegradedNote>No payments recorded.</DegradedNote>
        ) : (
          <DataTable data={paymentsQ.data?.items || []} pageSize={20} hidePager columns={paymentsCols} loading={false} error={null} onRetry={() => paymentsQ.refetch()} />
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
        <div className="space-y-4">
          <div className="rounded-lg border border-pat-border bg-pat-card p-4">
            <div className="text-sm font-medium text-pat-text-primary mb-3">Create coupon</div>
            <div className="grid grid-cols-2 md:grid-cols-6 gap-3">
              <input placeholder="Code (e.g. LAUNCH20)" value={couponForm.code} onChange={(e) => setCouponForm((c) => ({ ...c, code: e.target.value.toUpperCase() }))}
                className="px-2 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border" />
              <select value={couponForm.discountType} onChange={(e) => setCouponForm((c) => ({ ...c, discountType: e.target.value as "PERCENTAGE" | "FIXED" }))}
                className="px-2 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border">
                <option value="PERCENTAGE">% off</option>
                <option value="FIXED">Fixed</option>
              </select>
              <input placeholder={couponForm.discountType === "PERCENTAGE" ? "Value (e.g. 20)" : "Amount"} value={couponForm.discountValue} onChange={(e) => setCouponForm((c) => ({ ...c, discountValue: e.target.value }))}
                className="px-2 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border" inputMode="decimal" />
              <input placeholder="Currency (USD)" value={couponForm.currency} onChange={(e) => setCouponForm((c) => ({ ...c, currency: e.target.value }))}
                className="px-2 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border" />
              <input placeholder="Max redeems (blank = ∞)" value={couponForm.maxRedemptions} onChange={(e) => setCouponForm((c) => ({ ...c, maxRedemptions: e.target.value }))}
                className="px-2 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border" inputMode="numeric" />
              <button onClick={createCouponHandler} disabled={couponBusy}
                className="text-xs px-3 py-1.5 rounded bg-pat-success/20 text-pat-success hover:bg-pat-success/30 disabled:opacity-40 font-medium">
                {couponBusy ? "Creating…" : "Create"}
              </button>
            </div>
            <input placeholder="Description (optional)" value={couponForm.description} onChange={(e) => setCouponForm((c) => ({ ...c, description: e.target.value }))}
              className="mt-2 w-full px-2 py-1.5 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded border border-pat-border" />
          </div>

          {couponsQ.isLoading ? (
            <div className="text-sm text-pat-text-muted">Loading coupons…</div>
          ) : couponsQ.error ? (
            <DegradedNote>Degraded — coupons endpoint returned an error. Showing no data rather than fabricating records. {(couponsQ.error as Error).message}</DegradedNote>
          ) : (couponsQ.data?.items?.length ?? 0) === 0 ? (
            <DegradedNote>{couponsQ.data?.note || "No coupons configured."}</DegradedNote>
          ) : (
            <DataTable data={couponsQ.data?.items || []} columns={couponsCols} loading={false} error={null} onRetry={() => couponsQ.refetch()} />
          )}
        </div>
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
