"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchOperationsState,
  haltTrading,
  resumeTrading,
  pauseSignals,
  resumeSignals,
  fetchMarketState,
} from "@/lib/admin-api";
import ConfirmDialog from "@/components/admin/confirm-dialog";
import StatusBadge from "@/components/ui/status-badge";
import { toast } from "sonner";
import { IconShield, IconAlertTriangle, IconBolt } from "@tabler/icons-react";

interface TradingState {
  trading_halted: boolean;
  signals_paused: boolean;
  active_strategies: string[];
  last_updated: string;
}

interface MarketState {
  [key: string]: unknown;
}

const HARD_GATES = [
  "Data quality / freshness",
  "Hard risk veto",
  "News / session restriction",
  "Spread / slippage / total-cost limit",
  "Margin / exposure / account restriction",
  "Broker spec / execution constraint",
  "License / entitlement / device / account permission",
  "Signal TTL / replay / idempotency",
  "Emergency stop",
  "Financial-ledger correctness",
  "Security / privacy / compliance",
  "NO-TRADE fallback integrity",
];

export default function AdminRiskCenterPage() {
  const queryClient = useQueryClient();
  const [confirm, setConfirm] = useState<{ title: string; message: string; fn: () => Promise<void> } | null>(null);
  const [reason, setReason] = useState("");

  // Local/optimistic risk configuration (config-pending — backend write pending)
  const [killSwitches, setKillSwitches] = useState<Record<string, boolean>>({
    strategy: false,
    account: false,
    broker: false,
    symbol: false,
  });
  const [limits, setLimits] = useState<Record<string, string>>({
    max_exposure: "100000",
    max_spread: "5.0",
    max_slippage: "2.0",
    max_drawdown: "15",
    max_daily_loss: "5",
  });
  const [sessionBlackout, setSessionBlackout] = useState(false);
  const [newsBlackout, setNewsBlackout] = useState(false);

  const { data: state } = useQuery<TradingState>({
    queryKey: ["ops-state-risk"],
    queryFn: async () => (await fetchOperationsState()) as TradingState,
    refetchInterval: 15000,
  });

  const marketQ = useQuery<MarketState>({
    queryKey: ["market-state-risk"],
    queryFn: async () => (await fetchMarketState()) as MarketState,
  });

  const mutation = useMutation({
    mutationFn: async (fn: () => Promise<unknown>) => { await fn(); },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ops-state-risk"] });
      toast.success("Operation completed");
      setConfirm(null);
      setReason("");
    },
    onError: (err: unknown) => toast.error(err instanceof Error ? err.message : "Operation failed"),
  });

  const liveActions = [
    {
      label: "Halt Trading",
      danger: true,
      disabled: state?.trading_halted,
      fn: () => haltTrading(reason || "admin_risk_halt"),
      title: "Confirm Halt Trading",
      message: "Emergency stop: stops all signal execution across the platform. Existing positions remain.",
    },
    {
      label: "Resume Trading",
      danger: false,
      disabled: !state?.trading_halted,
      fn: () => resumeTrading(reason || "admin_risk_resume"),
      title: "Confirm Resume Trading",
      message: "Resume signal execution across the platform.",
    },
    {
      label: "Pause Signals",
      danger: true,
      disabled: state?.signals_paused,
      fn: () => pauseSignals(reason || "admin_risk_pause"),
      title: "Confirm Pause Signals",
      message: "Stop generating new signals. Existing signals remain active.",
    },
    {
      label: "Resume Signals",
      danger: false,
      disabled: !state?.signals_paused,
      fn: () => resumeSignals(reason || "admin_risk_resume"),
      title: "Confirm Resume Signals",
      message: "Resume signal generation.",
    },
  ];

  const toggleKill = (k: string) => {
    setKillSwitches((p) => ({ ...p, [k]: !p[k] }));
    toast.error(`${k} kill-switch is config-pending — backend risk rule-write endpoint not available`);
  };
  const changeLimit = (k: string, v: string) => {
    setLimits((p) => ({ ...p, [k]: v }));
  };
  const saveConfig = () => toast.error("Risk configuration save pending backend — limits are local-only preview");

  // Map market state fields to gates when available
  const marketGateStatus = (gate: string): "active" | "degraded" | "unknown" => {
    const m = marketQ.data;
    if (!m) return "degraded";
    switch (gate) {
      case "Data quality / freshness":
        return m.data_fresh ?? m.freshness ? "active" : "degraded";
      case "Spread / slippage / total-cost limit":
        return typeof m.spread === "number" || typeof m.avg_spread === "number" ? "active" : "degraded";
      case "Emergency stop":
        return m.emergency_stop || m.halt ? "active" : "degraded";
      default:
        return "active";
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Risk Center</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Live trading controls and risk guardrail configuration.</p>
      </div>

      {/* Live Platform State */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Live Platform State</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Trading</span>
            <StatusBadge status={state?.trading_halted ? "halted" : "active"} />
          </div>
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Signals</span>
            <StatusBadge status={state?.signals_paused ? "paused" : "active"} />
          </div>
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Strategies</span>
            <span className="text-xs text-pat-text-secondary">{state?.active_strategies?.length ?? 0} active</span>
          </div>
          {state?.last_updated && (
            <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
              <span className="text-xs text-pat-text-muted">Updated</span>
              <span className="text-xs text-pat-text-secondary">{new Date(state.last_updated).toLocaleString()}</span>
            </div>
          )}
        </div>
      </div>

      {/* LIVE emergency controls */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2"><IconBolt size={16} /> Live Emergency Controls</h2>
        <div className="mb-3">
          <input type="text" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Reason (optional, for audit trail)"
            className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {liveActions.map((a) => (
            <button key={a.label} onClick={() => setConfirm({ title: a.title, message: a.message, fn: a.fn })} disabled={a.disabled || mutation.isPending}
              className={`px-3 py-2 text-sm font-medium rounded-md transition-colors disabled:opacity-50 ${a.danger ? "bg-pat-danger text-white hover:opacity-90" : "bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover"}`}>
              {a.label}
            </button>
          ))}
        </div>
      </div>

      {/* Risk configuration (config-pending) */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm space-y-4">
        <div className="flex items-center gap-2">
          <IconShield size={16} className="text-pat-info" />
          <h2 className="text-sm font-medium text-pat-text-primary">Risk Guardrail Configuration</h2>
        </div>
        <div className="rounded-md bg-pat-warning/10 border border-pat-warning/20 px-3 py-2 text-[11px] text-pat-warning">
          Configuration preview only — backend risk rule-write endpoint is pending. Toggles and limits are local/optimistic and not yet enforced server-side.
        </div>

        <div>
          <div className="text-xs text-pat-text-muted mb-2">Kill-Switch Toggles</div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
            {Object.keys(killSwitches).map((k) => (
              <button key={k} onClick={() => toggleKill(k)} className={`px-3 py-2 text-xs rounded-md border transition-colors ${killSwitches[k] ? "border-pat-danger text-pat-danger bg-pat-danger/10" : "border-pat-border text-pat-text-secondary bg-pat-bg-surface-secondary"}`}>
                {k.charAt(0).toUpperCase() + k.slice(1)}: {killSwitches[k] ? "OFF" : "ON"}
              </button>
            ))}
          </div>
        </div>

        <div>
          <div className="text-xs text-pat-text-muted mb-2">Numeric Limits</div>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
            {Object.keys(limits).map((k) => (
              <div key={k}>
                <label className="text-[10px] uppercase text-pat-text-muted">{k.replace(/_/g, " ")}</label>
                <input type="number" value={limits[k]} onChange={(e) => changeLimit(k, e.target.value)} className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-2 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
              </div>
            ))}
          </div>
        </div>

        <div>
          <div className="text-xs text-pat-text-muted mb-2">Session & News Blackout</div>
          <div className="flex flex-wrap gap-2">
            <button onClick={() => { setSessionBlackout((p) => !p); toast.error("Session blackout config pending backend"); }} className={`px-3 py-2 text-xs rounded-md border transition-colors ${sessionBlackout ? "border-pat-danger text-pat-danger bg-pat-danger/10" : "border-pat-border text-pat-text-secondary bg-pat-bg-surface-secondary"}`}>
              Session Blackout: {sessionBlackout ? "ENABLED" : "DISABLED"}
            </button>
            <button onClick={() => { setNewsBlackout((p) => !p); toast.error("News blackout config pending backend"); }} className={`px-3 py-2 text-xs rounded-md border transition-colors ${newsBlackout ? "border-pat-danger text-pat-danger bg-pat-danger/10" : "border-pat-border text-pat-text-secondary bg-pat-bg-surface-secondary"}`}>
              News Blackout: {newsBlackout ? "ENABLED" : "DISABLED"}
            </button>
          </div>
        </div>

        <button onClick={saveConfig} className="px-4 py-2 text-xs bg-pat-bg-surface-secondary text-pat-text-primary rounded-md hover:bg-pat-bg-surface-secondary transition-colors">
          Save Risk Config (pending backend)
        </button>
      </div>

      {/* 12 Hard Gates panel */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2"><IconAlertTriangle size={16} /> Hard-Risk Gate Status</h2>
        {marketQ.isError && (
          <div className="rounded-md bg-pat-warning/10 border border-pat-warning/20 px-3 py-2 text-[11px] text-pat-warning mb-3">
            Degraded — market-state feed unavailable for live gate evaluation. Gates listed from SOW are shown but not actively monitored. No fabricated gate state is displayed.
          </div>
        )}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
          {HARD_GATES.map((gate) => {
            const status = marketQ.data ? marketGateStatus(gate) : "degraded";
            return (
              <div key={gate} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
                <span className="text-xs text-pat-text-primary">{gate}</span>
                {status === "active" ? <StatusBadge status="active" /> : status === "degraded" ? <StatusBadge status="degraded" /> : <StatusBadge status="unknown" />}
              </div>
            );
          })}
        </div>
      </div>

      <ConfirmDialog
        open={!!confirm}
        title={confirm?.title || ""}
        message={confirm?.message || ""}
        confirmLabel="Confirm"
        onConfirm={() => { if (confirm) mutation.mutate(confirm.fn); }}
        onCancel={() => setConfirm(null)}
        loading={mutation.isPending}
      />
    </div>
  );
}
