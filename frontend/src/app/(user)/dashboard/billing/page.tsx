"use client";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { toast } from "sonner";

interface Plan { id: string; code: "FREE" | "STANDARD" | "PRO" | "ELITE"; name: string; monthly_price: string; annual_price: string | null; allowed_strategies: string[]; annual_savings_percent: number | null; }
interface Invoice { id: string; amount: number; status: string; description: string; created_at: string; due_date: string; }
interface Entitlements { code?: string; }

const planCopy: Record<Plan["code"], string> = {
  FREE: "Explore Predict-A-Trade with five qualified real-time signals per month.",
  STANDARD: "Focused XAUUSD trading with one Standard strategy.",
  PRO: "Advanced intelligence with any two strategies.",
  ELITE: "Complete Predict-A-Trade intelligence across all four strategies.",
};

export default function UserBillingPage() {
  const [interval, setInterval] = useState<"MONTHLY" | "ANNUAL">("MONTHLY");
  const queryClient = useQueryClient();
  const plansQ = useQuery({ queryKey: ["plans"], queryFn: async () => (await customInstance.get("/plans")).data as Plan[] });
  const entitlementsQ = useQuery({ queryKey: ["subscription-entitlements"], queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data as Entitlements });
  const invoicesQ = useQuery({ queryKey: ["user-invoices"], queryFn: async () => ((await customInstance.get("/billing/invoices")).data as Invoice[]) || [] });
  const requestSubscription = useMutation({
    mutationFn: async (plan: Plan) => {
      const count = plan.code === "ELITE" ? 4 : plan.code === "PRO" ? 2 : 1;
      await customInstance.post("/subscriptions", { planId: plan.id, billingInterval: plan.code === "FREE" ? "MONTHLY" : interval, selectedStrategies: plan.allowed_strategies.slice(0, count) });
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["subscription-entitlements"] }); toast.success("Subscription request recorded; activation awaits validated payment."); },
    onError: () => toast.error("Subscription request could not be recorded."),
  });
  const columns: DataTableColumn<Invoice>[] = [
    { key: "description", header: "Description", cell: (row) => <span className="text-sm text-pat-text-primary">{row.description || "Invoice"}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="font-medium text-pat-text-primary">${parseFloat(String(row.amount || 0)).toFixed(2)}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Created", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
    { key: "due_date", header: "Due", cell: (row) => <span className="text-xs text-pat-text-muted">{row.due_date ? format(new Date(row.due_date), "MMM d, yyyy") : "—"}</span> },
  ];
  const currentCode = entitlementsQ.data?.code;
  return <div className="space-y-6">
    <div className="flex flex-wrap items-end justify-between gap-3"><div><h1 className="text-xl font-bold text-pat-text-primary">Plans & Subscription</h1><p className="mt-1 text-sm text-pat-text-secondary">Fees and access are read from the server-authoritative plan configuration.</p></div><div className="flex rounded-lg border border-pat-border p-1"><button onClick={() => setInterval("MONTHLY")} className={`rounded px-3 py-1 text-xs ${interval === "MONTHLY" ? "bg-primary text-primary-foreground" : "text-pat-text-secondary"}`}>Monthly</button><button onClick={() => setInterval("ANNUAL")} className={`rounded px-3 py-1 text-xs ${interval === "ANNUAL" ? "bg-primary text-primary-foreground" : "text-pat-text-secondary"}`}>Annual</button></div></div>
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">{(plansQ.data ?? []).map((plan) => { const isCurrent = currentCode === plan.code; const annual = interval === "ANNUAL" && plan.annual_price !== null; const price = annual ? plan.annual_price : plan.monthly_price; return <div key={plan.id} className="rounded-lg border border-pat-border bg-pat-bg-surface p-5"><div className="flex items-center justify-between"><h2 className="font-semibold text-pat-text-primary">{plan.name}</h2><StatusBadge status={isCurrent ? "CURRENT" : "AVAILABLE"} size="sm" /></div><div className="mt-3 text-2xl font-bold text-pat-text-primary">${Number(price || 0).toFixed(0)}<span className="text-xs font-normal text-pat-text-muted">/{annual ? "year" : "month"}</span></div>{annual && plan.annual_savings_percent !== null && <div className="mt-1 text-xs text-pat-success">Save {plan.annual_savings_percent}%</div>}<p className="mt-3 min-h-10 text-xs text-pat-text-secondary">{planCopy[plan.code]}</p><div className="mt-3 text-xs text-pat-text-muted">{plan.allowed_strategies.length} permitted strateg{plan.allowed_strategies.length === 1 ? "y" : "ies"}</div><button onClick={() => requestSubscription.mutate(plan)} disabled={isCurrent || requestSubscription.isPending} className="mt-4 w-full rounded bg-primary px-3 py-2 text-xs text-primary-foreground disabled:cursor-not-allowed disabled:opacity-50">{isCurrent ? "Current Plan" : plan.code === "FREE" ? "Start Free" : "Select Plan"}</button></div>; })}</div>
    <div><h2 className="mb-3 text-sm font-semibold text-pat-text-primary">Invoice history</h2><DataTable data={invoicesQ.data || []} columns={columns} loading={invoicesQ.isLoading} error={invoicesQ.error as Error|null} onRetry={() => invoicesQ.refetch()} /></div>
  </div>;
}
