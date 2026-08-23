"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { fetchCommissionSummary as fetchSummaryFromApi } from "@/lib/admin-api";
import StatCard from "@/components/admin/stat-card";
import { IconCoin, IconSettings, IconAlertTriangle } from "@tabler/icons-react";

interface CommissionSummary {
  total_entries?: string | number;
  total_amount?: string | number;
  pending_count?: string | number;
  pending_amount?: string | number;
  confirmed_count?: string | number;
  confirmed_amount?: string | number;
  available_amount?: string | number;
  paid_count?: string | number;
  paid_amount?: string | number;
  reversed_count?: string | number;
  reversed_amount?: string | number;
}

const LEVELS = [1, 2, 3, 4, 5] as const;

export default function AdminCommissionControlCenterPage() {
  const [baseRate, setBaseRate] = useState("10");
  const [levelRates, setLevelRates] = useState<Record<number, string>>({ 1: "10", 2: "5", 3: "3", 4: "2", 5: "1" });
  const [planRates, setPlanRates] = useState<Record<string, string>>({ FREE: "0", STANDARD: "10", PRO: "15", ELITE: "20" });

  const summaryQ = useQuery<CommissionSummary>({
    queryKey: ["commission-control-summary"],
    queryFn: async () => {
      try {
        return (await customInstance.get("/commissions/admin/summary")).data as CommissionSummary;
      } catch {
        return (await fetchSummaryFromApi()) as CommissionSummary;
      }
    },
  });

  const s = summaryQ.data ?? {};

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Commission Control Center</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Commission rule configuration and live commission summary.</p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Total Commissions" value={String(s.total_entries ?? "—")} icon={IconCoin} color="text-pat-text-secondary" />
        <StatCard label="Total Amount" value={`$${Number(s.total_amount ?? 0).toFixed(2)}`} icon={IconCoin} color="text-pat-success" />
        <StatCard label="Pending" value={`$${Number(s.pending_amount ?? 0).toFixed(2)}`} icon={IconCoin} color="text-pat-warning" />
        <StatCard label="Paid" value={`$${Number(s.paid_amount ?? s.confirmed_amount ?? 0).toFixed(2)}`} icon={IconCoin} color="text-pat-success" />
      </div>

      {summaryQ.isError && (
        <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 flex items-start gap-2">
          <IconAlertTriangle size={16} className="text-pat-warning shrink-0 mt-0.5" />
          <div className="text-xs text-pat-text-secondary">
            Degraded — commission summary endpoint returned an error or is pending. Values above reflect no data rather than fabricated figures.
            <div className="mt-1 text-pat-text-muted">{(summaryQ.error as Error).message}</div>
          </div>
        </div>
      )}

      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5 space-y-5">
        <div className="flex items-center gap-2">
          <IconSettings size={16} className="text-pat-info" />
          <h2 className="text-sm font-semibold text-pat-text-primary">Commission Rule Configuration</h2>
        </div>
        <div className="rounded-md bg-pat-warning/10 border border-pat-warning/20 px-3 py-2 text-[11px] text-pat-warning">
          Configuration preview only — backend commission rule-write endpoint is pending. No rules are saved or applied by this form.
        </div>

        <div className="space-y-4">
          <div>
            <label className="text-xs text-pat-text-muted">Base Commission Rate (%)</label>
            <input type="number" value={baseRate} onChange={(e) => setBaseRate(e.target.value)} className="mt-1 w-40 rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
          </div>

          <div>
            <div className="text-xs text-pat-text-muted mb-2">Multi-Level Referral Rates (L1–L5, %)</div>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
              {LEVELS.map((lvl) => (
                <div key={lvl}>
                  <label className="text-[10px] uppercase text-pat-text-muted">L{lvl}</label>
                  <input type="number" value={levelRates[lvl]} onChange={(e) => setLevelRates({ ...levelRates, [lvl]: e.target.value })} className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-2 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
                </div>
              ))}
            </div>
          </div>

          <div>
            <div className="text-xs text-pat-text-muted mb-2">Plan Base Rates (%)</div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
              {Object.keys(planRates).map((plan) => (
                <div key={plan}>
                  <label className="text-[10px] uppercase text-pat-text-muted">{plan}</label>
                  <input type="number" value={planRates[plan]} onChange={(e) => setPlanRates({ ...planRates, [plan]: e.target.value })} className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-2 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
                </div>
              ))}
            </div>
          </div>
        </div>

        <button
          onClick={() => alert("Backend commission rule-write endpoint pending — configuration not saved.")}
          className="px-4 py-2 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded-md hover:bg-pat-bg-surface-secondary transition-colors"
        >
          Save Rules (pending backend)
        </button>
      </div>
    </div>
  );
}
