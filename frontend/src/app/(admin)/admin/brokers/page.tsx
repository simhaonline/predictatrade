"use client";
// Broker Account Types + Strategy Cost Gates — check.md playbook §8
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { customInstance } from "@/lib/axios-instance";
import { toast } from "sonner";
import { IconBuildingBank, IconTrendingUp, IconX } from "@tabler/icons-react";
import StatusBadge from "@/components/ui/status-badge";

interface BrokerType {
  id: string; code: string; label: string; execution_model: string;
  typical_spread_pips: string | number; commission_per_side: string | number;
  commission_per_lot_r?: string | number; min_deposit?: string | number;
  mt4_supported: boolean; mt5_supported: boolean; webtrader: boolean;
  best_for?: string; is_active: boolean;
}
interface StrategyGate {
  strategy_id: string; broker_account_code: string; label: string;
  cost_as_pct_of_1r: string | number; suitability: string; allowed: boolean;
}

export default function AdminBrokersPage() {
  const [selectedStrategy, setSelectedStrategy] = useState("ULTRA_SCALPING");
  const qc = useQueryClient();

  const typesQ = useQuery({
    queryKey: ["broker-types"],
    queryFn: async () => (await customInstance.get("/brokers/admin/account-types")).data,
  });
  const gatesQ = useQuery({
    queryKey: ["strategy-gates", selectedStrategy],
    queryFn: async () => (await customInstance.get(`/brokers/strategy-gates/${selectedStrategy}`)).data,
  });

  const strategies = ["ULTRA_SCALPING", "STANDARD_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB"];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Broker Account Types</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Account structure affects cost-per-trade and strategy suitability (playbook §8).
        </p>
      </div>

      {/* Account types grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {(typesQ.data ?? []).map((bt: BrokerType) => (
          <div key={bt.id} className="rounded-lg border border-pat-border p-4">
            <div className="flex items-center justify-between mb-2">
              <div className="font-medium text-pat-text-primary">{bt.label}</div>
              <StatusBadge status={bt.is_active ? "active" : "inactive"} size="sm" />
            </div>
            <div className="text-xs text-pat-text-muted space-y-1">
              <div>Model: {bt.execution_model}</div>
              <div>Typical spread: {bt.typical_spread_pips} pips</div>
              <div>Commission: ${bt.commission_per_side}/side/lot</div>
              <div>Min deposit: ${bt.min_deposit}</div>
              <div>MT4: {bt.mt4_supported ? "✅" : "❌"} · MT5: {bt.mt5_supported ? "✅" : "❌"}</div>
              <div>Best for: {bt.best_for || "—"}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Strategy cost gates selector */}
      <div className="rounded-lg border border-pat-border p-4">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconTrendingUp size={16} /> Strategy Cost Gates
        </h2>
        <div className="flex gap-2 mb-3 flex-wrap">
          {strategies.map((s) => (
            <button key={s} onClick={() => setSelectedStrategy(s)}
              className={`rounded px-3 py-1.5 text-xs ${selectedStrategy === s ? "bg-primary text-primary-foreground" : "border border-pat-border text-pat-text-secondary"}`}>
              {s.replace(/_/g, " ")}
            </button>
          ))}
        </div>
        {gatesQ.isLoading && <div className="text-xs text-pat-text-muted">Loading gates…</div>}
        {gatesQ.data && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                  <th className="px-3 py-2 font-medium">Account Type</th>
                  <th className="px-3 py-2 font-medium">Cost % of 1R</th>
                  <th className="px-3 py-2 font-medium">Suitability</th>
                  <th className="px-3 py-2 font-medium">Allowed</th>
                </tr>
              </thead>
              <tbody>
                {(gatesQ.data as StrategyGate[]).map((g) => (
                  <tr key={g.id} className="border-b border-pat-border/50">
                    <td className="px-3 py-2 text-pat-text-primary">{g.label}</td>
                    <td className="px-3 py-2 text-pat-text-secondary">{g.cost_as_pct_of_1r}%</td>
                    <td className="px-3 py-2"><span className={g.suitability === "best" ? "text-pat-success" : g.suitability === "avoid" ? "text-pat-danger" : "text-pat-text-secondary"}>{g.suitability}</span></td>
                    <td className="px-3 py-2">
                      <span className={`text-xs px-2 py-0.5 rounded-full ${g.allowed ? "bg-pat-success/15 text-pat-success" : "bg-pat-danger/15 text-pat-danger"}`}>
                        {g.allowed ? "ALLOWED" : "BLOCKED"}
                      </span>
                    </td>
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
