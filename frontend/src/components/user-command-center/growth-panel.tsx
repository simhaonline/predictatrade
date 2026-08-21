"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconUsers, IconCoin, IconReceipt, IconTrendingUp } from "@tabler/icons-react";

export function GrowthPanel() {
  const { data: commissionSummary } = useQuery({
    queryKey: ["user-growth-commission"],
    queryFn: async () => (await customInstance.get("/commissions/summary")).data as {
      total_amount: string; pending_count: string; confirmed_count: string;
      pending_amount: string; confirmed_amount: string;
    },
  });

  const { data: subscriptions } = useQuery({
    queryKey: ["user-growth-subs"],
    queryFn: async () => (await customInstance.get("/subscriptions")).data,
  });

  const { data: commissions } = useQuery({
    queryKey: ["user-growth-commissions"],
    queryFn: async () => (await customInstance.get("/commissions")).data as Array<{
      id: string; commission_amount: string; status: string; created_at: string;
    }>,
  });

  const totalEarnings = parseFloat(commissionSummary?.total_amount ?? "0");
  const confirmedEarnings = parseFloat(commissionSummary?.confirmed_amount ?? "0");
  const pendingEarnings = parseFloat(commissionSummary?.pending_amount ?? "0");
  const subCount = Array.isArray(subscriptions) ? subscriptions.length : 0;
  const commissionCount = Array.isArray(commissions) ? commissions.length : 0;

  const cards = [
    { label: "Total Earnings", value: `$${totalEarnings.toFixed(2)}`, icon: IconCoin, color: "text-pat-success" },
    { label: "Confirmed", value: `$${confirmedEarnings.toFixed(2)}`, icon: IconTrendingUp, color: "text-pat-success" },
    { label: "Pending", value: `$${pendingEarnings.toFixed(2)}`, icon: IconReceipt, color: "text-pat-warning" },
    { label: "Subscriptions", value: `${subCount}`, icon: IconUsers, color: "text-pat-info" },
  ];

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {cards.map((card) => (
          <div key={card.label} className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[10px] text-pat-text-muted uppercase">{card.label}</span>
              <card.icon size={16} className={card.color} />
            </div>
            <div className={`text-lg font-bold ${card.color} tabular-nums`}>{card.value}</div>
          </div>
        ))}
      </div>

      {/* Recent commissions */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Recent Commissions</h3>
        {commissionCount === 0 ? (
          <div className="text-xs text-pat-text-muted py-4 text-center">No commissions yet. Share your referral link to start earning.</div>
        ) : (
          <div className="space-y-1.5">
            {(commissions ?? []).slice(0, 8).map((c) => (
              <div key={c.id} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary/30 px-3 py-1.5">
                <span className="text-xs text-pat-text-secondary">${parseFloat(c.commission_amount || "0").toFixed(2)}</span>
                <div className="flex items-center gap-2">
                  <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${
                    c.status === "CONFIRMED" ? "bg-pat-success/10 text-pat-success" : "bg-pat-warning/10 text-pat-warning"
                  }`}>{c.status}</span>
                  <span className="text-[10px] text-pat-text-muted">{c.created_at ? new Date(c.created_at).toLocaleDateString() : "—"}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
