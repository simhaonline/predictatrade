"use client";
// Payments Admin — check.md 2026-08-30 #8: gateway status + USDT settings
// view + payment event ledger (live from backend).
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { toast } from "sonner";
import { IconStatusChange, IconAlertTriangle, IconCircleCheck } from "@tabler/icons-react";
import StatusBadge from "@/components/ui/status-badge";

interface PaymentRow {
  id: string;
  user_id?: string;
  status: string;
  amount: string;
  currency: string;
  provider: string;
  created_at: string;
  processed_at?: string | null;
}

export default function AdminPaymentsPage() {
  const q = useQuery({
    queryKey: ["admin-payments"],
    queryFn: async () => (await customInstance.get("/admin/subscriptions/payments")).data,
    refetchInterval: 30_000,
  });
  const nwStatus = useQuery({
    queryKey: ["nowpayments-status"],
    queryFn: async () => (await customInstance.get("/operations/state")).data,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Payments Configuration</h1>
        <p className="mt-1 text-sm text-pat-text-secondary">Live USDT payment status, gateway health and the full payment ledger.</p>
      </div>

      <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Payment Gateway Status</h2>
        <div className="flex flex-wrap gap-4 text-sm">
          <div className="flex items-center gap-2 rounded-lg bg-pat-bg-surface-secondary px-3 py-2">
            <IconCircleCheck size={16} className="text-pat-success" />
            <div>
              <div className="font-medium text-pat-text-primary">NOWPayments (USDT)</div>
              <div className="text-xs text-pat-text-secondary">Sole active gateway · pay_currency=USDT · HMAC-SHA512 verified IPN</div>
            </div>
          </div>
          <div className="flex items-center gap-2 rounded-lg bg-pat-bg-surface-secondary px-3 py-2">
            <IconAlertTriangle size={16} className="text-pat-text-muted" />
            <div>
              <div className="font-medium text-pat-text-secondary">Stripe</div>
              <div className="text-xs text-pat-text-muted">Disabled by product decision — re-enable via PAT_ENABLE_STRIPE env</div>
            </div>
          </div>
        </div>
    <div className="mt-3 rounded bg-pat-info/5 border border-pat-info/20 p-3 text-xs text-pat-text-secondary">
      <p>Anti-scam policy active: signature + replay dedupe + amount verification + one-shot settlement. Users see payment status in real-time on their billing page.</p>
      <p className="mt-1">Live view for FREE users is time-gated to 11:00-13:00 GMT+3 on live.predictatrade.com.</p>
    </div>
      </div>

      <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Payment Ledger</h2>
        {q.isLoading && <div className="text-xs text-pat-text-muted">Loading payments…</div>}
        {q.isError && <div className="text-xs text-pat-danger">Failed to load payments: {(q.error as any)?.message}</div>}
        {q.data && Array.isArray(q.data) && q.data.length === 0 && (
          <div className="text-xs text-pat-text-muted">No payments yet.</div>
        )}
        {q.data && Array.isArray(q.data) && q.data.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                  <th className="px-3 py-2 font-medium">Payment ID</th>
                  <th className="px-3 py-2 font-medium">User</th>
                  <th className="px-3 py-2 font-medium">Amount</th>
                  <th className="px-3 py-2 font-medium">Status</th>
                  <th className="px-3 py-2 font-medium">Created</th>
                  <th className="px-3 py-2 font-medium">Processed</th>
                </tr>
              </thead>
              <tbody>
                {q.data.map((p: any) => (
                  <tr key={p.id} className="border-b border-pat-border/50">
                    <td className="px-3 py-2 font-mono text-xs text-pat-text-primary">{(p.provider_payment_id ?? p.id).slice(0, 14)}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{p.user_id?.slice(0, 8)}</td>
                    <td className="px-3 py-2 text-pat-text-primary">{p.amount} {p.currency}</td>
                    <td className="px-3 py-2"><StatusBadge status={p.status} /></td>
                    <td className="px-3 py-2 text-xs text-pat-text-muted">{p.created_at?.slice(0, 10)}</td>
                    <td className="px-3 py-2 text-xs text-pat-text-muted">{p.processed_at?.slice(0, 10) ?? "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}