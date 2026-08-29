"use client";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";
import { IconCalendar, IconCheck, IconLock } from "@tabler/icons-react";

interface Plan { id: string; code: "FREE" | "STANDARD" | "PRO" | "ELITE"; name: string; monthly_price: string; annual_price: string | null; allowed_strategies: string[]; annual_savings_percent: number | null; legacy?: boolean; }
interface Invoice { id: string; invoice_number: string; total: number | string; items_total?: number | string; status: string; created_at: string; due_date: string; }
interface PaymentStatus {
  id: string;
  subscription_id?: string;
  status: string;
  display_status: "awaiting_payment" | "confirmed" | "underpaid" | "failed";
  gateway_event?: string | null;
  amount: string | number;
  currency: string;
  hosted_url?: string | null;
  created_at: string;
  processed_at?: string | null;
}
interface Entitlements { code?: string; }
interface Subscription {
  id: string;
  plan_id: string;
  plan_name?: string;
  plan_code?: string;
  status: string;
  billing_interval: string;
  billing_period_start?: string;
  billing_period_end?: string;
  selected_strategies?: string[];
}

const planCopy: Record<Plan["code"], string> = {
  FREE: "Explore Predict-A-Trade with one scalping strategy.",
  STANDARD: "Focused XAUUSD trading with Standard Scalping + Standard Swing.",
  PRO: "Advanced intelligence with all four strategies, up to 3 devices.",
  ELITE: "Complete Predict-A-Trade intelligence across all five strategies including MarnieFib, 5 devices, full auto.",
};

// Tier hierarchy for upgrade/downgrade logic
const PLAN_TIERS: Record<string, number> = { FREE: 0, STANDARD: 1, PRO: 2, ELITE: 3 };

export default function UserBillingPage() {
  const [interval, setInterval] = useState<"MONTHLY" | "ANNUAL">("MONTHLY");
  const [autoRenew, setAutoRenew] = useState(true);
  const [usdtNotice, setUsdtNotice] = useState<string | null>(null);
  const queryClient = useQueryClient();
  // USDT payment verification status — server-truth from billing.payments.
  // The banner mirrors settlement state exactly (no optimistic lies): the only
  // "confirmed" path is a signature-verified, amount-verified gateway event.
  const paymentsQ = useQuery({
    queryKey: ["usdt-payments"],
    queryFn: async () => ((await customInstance.get("/billing/payments")).data as PaymentStatus[]) || [],
    refetchInterval: 30_000,
  });
  const latestPayment = paymentsQ.data?.[0];
  const plansQ = useQuery({ queryKey: ["plans"], queryFn: async () => (await customInstance.get("/plans")).data as Plan[] });
  const entitlementsQ = useQuery({ queryKey: ["subscription-entitlements"], queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data as Entitlements });
  const invoicesQ = useQuery({ queryKey: ["user-invoices"], queryFn: async () => ((await customInstance.get("/billing/invoices")).data as Invoice[]) || [] });
  const subsQ = useQuery({ queryKey: ["user-subscriptions"], queryFn: async () => ((await customInstance.get("/subscriptions")).data as Subscription[]) || [] });

  const requestSubscription = useMutation({
    mutationFn: async (plan: Plan) => {
      const count = plan.code === "ELITE" ? 5 : plan.code === "PRO" ? 2 : 1;
      await customInstance.post("/subscriptions", { planId: plan.id, billingInterval: plan.code === "FREE" ? "MONTHLY" : interval, selectedStrategies: plan.allowed_strategies.slice(0, count) });
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["subscription-entitlements"] }); toast.success("Plan upgrade request recorded; activation awaits validated payment."); },
    onError: () => toast.error("Plan upgrade request could not be recorded."),
  });

  // USDT (NOWPayments) checkout: server creates a pending payment + hosted
  // invoice URL; browser is redirected to it. 503 means the gateway is not
  // configured yet — shown honestly as a degraded state, not an error toast.
  const payWithUsdt = useMutation({
    mutationFn: async (plan: Plan) => {
      setUsdtNotice(null);
      const res = await customInstance.post("/billing/nowpayments/create-invoice", {
        plan_id: plan.id,
        billing_interval: plan.code === "FREE" ? "MONTHLY" : interval,
      });
      return res.data as { payment_url: string };
    },
    onSuccess: (data) => {
      if (data?.payment_url) window.location.href = data.payment_url;
      else toast.error("Payment gateway returned no checkout URL");
    },
    onError: (err: unknown) => {
      const status = (err as { response?: { status?: number } } | undefined)?.response?.status;
      if (status === 503) {
        setUsdtNotice("USDT payments are temporarily unavailable — the crypto payment gateway is not configured yet. Card/other methods and support remain unaffected.");
      } else {
        toast.error("Could not start USDT checkout");
      }
    },
  });

  const current = subsQ.data?.find((s) => ["ACTIVE", "TRIAL", "GRACE", "CANCEL_AT_PERIOD_END"].includes(s.status)) || subsQ.data?.[0];
  const currentCode = entitlementsQ.data?.code || current?.plan_code;
  const currentTier = currentCode ? (PLAN_TIERS[currentCode] ?? 0) : 0;

  const columns: DataTableColumn<Invoice>[] = [
    { key: "invoice_number", header: "Invoice", cell: (row) => <span className="text-sm text-pat-text-primary">{row.invoice_number || "—"}</span> },
    { key: "total", header: "Amount", cell: (row) => <span className="font-medium text-pat-text-primary">${parseFloat(String(row.total || 0)).toFixed(2)}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Created", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
    { key: "due_date", header: "Due", cell: (row) => <span className="text-xs text-pat-text-muted">{row.due_date ? format(new Date(row.due_date), "MMM d, yyyy") : "—"}</span> },
    {
      key: "actions",
      header: "Actions",
      cell: (row) => (
        <button
          onClick={() => openInvoice(row.id)}
          className="text-xs text-primary hover:underline"
        >
          View / Print
        </button>
      ),
    },
  ];

  const openInvoice = async (invoiceId: string) => {
    try {
      const res = await customInstance.get<string>(`/billing/invoices/${invoiceId}/html`, { responseType: "text" });
      const html = typeof res.data === "string" ? res.data : String(res.data);
      const blob = new Blob([html], { type: "text/html" });
      const url = URL.createObjectURL(blob);
      window.open(url, "_blank", "noopener,noreferrer");
    } catch {
      toast.error("Could not open invoice");
    }
  };

  const generateInvoice = useMutation({
    mutationFn: async (subscriptionId: string) => {
      await customInstance.post("/billing/invoices/generate", { subscription_id: subscriptionId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["user-invoices"] });
      toast.success("Invoice generated");
    },
    onError: () => toast.error("Could not generate invoice"),
  });

  const hasInvoices = (invoicesQ.data?.length ?? 0) > 0;

  return (
    <div className="space-y-6">
      {/* USDT payment verification status — server-truth banner (anti-scam
          transparency: state changes only after signature+amount verified) */}
      {paymentsQ.data && latestPayment && (
        latestPayment.display_status === "awaiting_payment" ? (
          <div role="status" className="rounded-lg border border-pat-warning/40 bg-pat-warning/5 px-4 py-3 text-sm">
            <div className="flex items-center justify-between gap-3 flex-wrap">
              <span className="text-pat-text-primary">⏳ Awaiting your USDT payment — expected {latestPayment.amount} {latestPayment.currency}</span>
              <div className="flex gap-2">
                {latestPayment.hosted_url && (
                  <a href={latestPayment.hosted_url} className="rounded bg-primary px-3 py-1.5 text-xs text-primary-foreground">Resume payment</a>
                )}
                <button onClick={() => queryClient.invalidateQueries({ queryKey: ["usdt-payments"] })} className="rounded border border-pat-border px-3 py-1.5 text-xs">Refresh status</button>
              </div>
            </div>
            <p className="mt-1 text-xs text-pat-text-secondary">Crypto payments need network confirmations; this banner updates automatically once verified.</p>
          </div>
        ) : latestPayment.display_status === "underpaid" ? (
          <div role="alert" className="rounded-lg border border-pat-danger/40 bg-pat-danger/5 px-4 py-3 text-sm">
            <div className="flex items-center justify-between gap-3 flex-wrap">
              <span className="text-pat-danger">⚠️ Payment underpaid — received less than the required {latestPayment.amount} USD equivalent. Subscription not activated.</span>
              <div className="flex gap-2">
                {latestPayment.hosted_url && (
                  <a href={latestPayment.hosted_url} className="rounded bg-primary px-3 py-1.5 text-xs text-primary-foreground">Pay remaining</a>
                )}
                <a href="/dashboard/support" className="rounded border border-pat-border px-3 py-1.5 text-xs">Contact support</a>
              </div>
            </div>
          </div>
        ) : latestPayment.display_status === "failed" ? (
          <div role="alert" className="rounded-lg border border-pat-danger/40 bg-pat-danger/5 px-4 py-3 text-sm text-pat-danger">
            Payment failed. <a href="/dashboard/support" className="underline">Contact support</a> if you were charged.
          </div>
        ) : latestPayment.display_status === "confirmed" ? (
          <div role="status" className="rounded-lg border border-pat-success/40 bg-pat-success/5 px-4 py-3 text-sm text-pat-text-secondary">
            ✅ USDT payment confirmed ({latestPayment.amount} USD-equivalent){latestPayment.processed_at ? ` — verified ${new Date(latestPayment.processed_at).toLocaleString()}` : ""}. Subscription active.
          </div>
        ) : null
      )}
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Plans & Subscription</h1>
          <p className="mt-1 text-sm text-pat-text-secondary">Manage your subscription plan. Only upgrade options are available.</p>
        </div>
        <div className="flex rounded-lg border border-pat-border p-1">
          <button onClick={() => setInterval("MONTHLY")} className={`rounded px-3 py-1 text-xs ${interval === "MONTHLY" ? "bg-primary text-primary-foreground" : "text-pat-text-secondary"}`}>Monthly</button>
          <button onClick={() => setInterval("ANNUAL")} className={`rounded px-3 py-1 text-xs ${interval === "ANNUAL" ? "bg-primary text-primary-foreground" : "text-pat-text-secondary"}`}>Annual</button>
        </div>
      </div>

      {/* Current subscription summary */}
      {subsQ.isLoading ? <div className="text-sm text-pat-text-secondary">Loading subscription…</div> : current ? (
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5 space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <IconCalendar size={18} className="text-pat-info" />
              <span className="text-sm font-medium text-pat-text-primary">Current subscription</span>
            </div>
            <StatusBadge status={current.status} size="sm" />
          </div>
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <Field label="Plan" value={current.plan_name || current.plan_code || currentCode || "—"} />
            <Field label="Billing" value={(current.billing_interval || "—").toLowerCase()} />
            <Field label="Next billing" value={current.billing_period_end ? format(new Date(current.billing_period_end), "MMM d, yyyy") : "—"} />
            <Field label="Period start" value={current.billing_period_start ? format(new Date(current.billing_period_start), "MMM d, yyyy") : "—"} />
          </div>
          <div className="flex items-center justify-between rounded-md border border-pat-border bg-pat-bg-surface-secondary/30 px-3 py-2">
            <div>
              <div className="text-xs text-pat-text-secondary">Auto-renew</div>
              <div className="text-[10px] text-pat-text-muted">Local preference — no server endpoint persists this yet.</div>
            </div>
            <button
              onClick={() => setAutoRenew((v) => !v)}
              role="switch"
              aria-checked={autoRenew}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${autoRenew ? "bg-pat-success" : "bg-pat-bg-surface-secondary border border-pat-border"}`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${autoRenew ? "translate-x-6" : "translate-x-1"}`} />
            </button>
          </div>
        </div>
      ) : null}

      {/* Honest degraded state: USDT gateway not configured (server 503) */}
      {usdtNotice && (
        <div role="status" className="rounded-lg border border-pat-warning/40 bg-pat-warning/5 px-4 py-3 text-xs text-pat-text-secondary">
          {usdtNotice}{" "}
          <button onClick={() => setUsdtNotice(null)} className="underline hover:text-pat-text-primary">Dismiss</button>
        </div>
      )}

      {/* Plan cards — only upgrade options shown */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        {(plansQ.data ?? []).filter(p => !p.legacy).map((plan) => {
          const isCurrent = currentCode === plan.code;
          const planTier = PLAN_TIERS[plan.code] ?? 0;
          const isUpgrade = planTier > currentTier;
          const isDowngrade = planTier < currentTier;
          const annual = interval === "ANNUAL" && plan.annual_price !== null;
          const price = annual ? plan.annual_price : plan.monthly_price;

          return (
            <div key={plan.id} className={`rounded-lg border p-5 ${isCurrent ? "border-pat-success/40 bg-pat-success/5" : "border-pat-border bg-pat-bg-surface"}`}>
              <div className="flex items-center justify-between">
                <h2 className="font-semibold text-pat-text-primary">{plan.name}</h2>
                {isCurrent ? (
                  <span className="flex items-center gap-1 text-xs font-medium text-pat-success">
                    <IconCheck size={14} /> Active
                  </span>
                ) : isDowngrade ? (
                  <span className="flex items-center gap-1 text-xs text-pat-text-muted">
                    <IconLock size={12} /> Lower tier
                  </span>
                ) : null}
              </div>
              <div className="mt-3 text-2xl font-bold text-pat-text-primary">
                ${Number(price || 0).toFixed(0)}
                <span className="text-xs font-normal text-pat-text-muted">/{annual ? "year" : "month"}</span>
              </div>
              {annual && plan.annual_savings_percent !== null && (
                <div className="mt-1 text-xs text-pat-success">Save {plan.annual_savings_percent}%</div>
              )}
              <p className="mt-3 min-h-10 text-xs text-pat-text-secondary">{planCopy[plan.code]}</p>
              <div className="mt-3 text-xs text-pat-text-muted">
                {plan.allowed_strategies.length} permitted strateg{plan.allowed_strategies.length === 1 ? "y" : "ies"}
              </div>
              <button
                onClick={() => isUpgrade && requestSubscription.mutate(plan)}
                disabled={isCurrent || isDowngrade || requestSubscription.isPending}
                className={`mt-4 w-full rounded px-3 py-2 text-xs ${
                  isCurrent
                    ? "bg-pat-success/10 text-pat-success cursor-default"
                    : isDowngrade
                    ? "bg-pat-bg-surface-secondary text-pat-text-muted cursor-not-allowed border border-pat-border"
                    : "bg-primary text-primary-foreground hover:bg-primary/90"
                }`}
              >
                {isCurrent ? "Current Plan" : isDowngrade ? "Not Available" : "Upgrade"}
              </button>
              {!isCurrent && !isDowngrade && plan.code !== "FREE" && (
                <button
                  onClick={() => payWithUsdt.mutate(plan)}
                  disabled={payWithUsdt.isPending}
                  className="mt-2 w-full rounded border border-pat-border bg-pat-bg-surface px-3 py-2 text-xs text-pat-text-secondary hover:bg-pat-bg-surface-secondary disabled:opacity-50"
                >
                  {payWithUsdt.isPending ? "Redirecting to USDT checkout…" : "Pay with USDT"}
                </button>
              )}
            </div>
          );
        })}
      </div>

      <div>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-pat-text-primary">Invoice history</h2>
          {current && !hasInvoices && (
            <button
              onClick={() => generateInvoice.mutate(current.id)}
              disabled={generateInvoice.isPending}
              className="rounded px-3 py-1.5 text-xs bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {generateInvoice.isPending ? "Generating…" : "Generate invoice"}
            </button>
          )}
        </div>
        <DataTable data={invoicesQ.data || []} columns={columns} loading={invoicesQ.isLoading} error={invoicesQ.error as Error|null} onRetry={() => invoicesQ.refetch()} />
      </div>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[10px] text-pat-text-muted">{label}</div>
      <div className="text-sm font-medium text-pat-text-primary">{value}</div>
    </div>
  );
}
