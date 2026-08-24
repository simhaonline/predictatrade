"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import {
  IconWorld, IconActivity, IconShieldCheck, IconChartBar,
  IconAlertTriangle, IconCheck, IconX, IconClock,
} from "@tabler/icons-react";

interface CrossMarketDriver {
  name: string;
  direction: string;
  impact_score: number;
  confidence: number;
  freshness: number;
  quality: string;
  source: string;
  timeframe: string;
  reason: string;
  base_weight: number;
  effective_weight: number;
}

interface CrossMarketCurrent {
  score: number;
  direction: string;
  confidence: number;
  agreement: number;
  conflict: number;
  data_quality: string;
  regime: string;
  event_risk: string;
  correlation_regime: string;
  divergence: string;
  score_adjustment: number;
  mode: string;
  model_version: string;
  primary_drivers: string[] | null;
  opposing_drivers: string[] | null;
  missing_drivers: string[] | null;
  warnings: string[] | null;
  drivers: CrossMarketDriver[];
  reason: string;
  timestamp: string;
}

interface ValidationStatus {
  mode: string;
  calendar_days: number;
  usable_shadow_days: number;
  total_candidates: number;
  resolved_outcomes: number;
  minimum_days_required: number;
  ablation_ready: boolean;
  walk_forward_ready: boolean;
  activation_eligible: boolean;
  message?: string;
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    CONNECTED: "bg-pat-success/10 text-pat-success border border-pat-success/20",
    HEALTHY: "bg-pat-success/10 text-pat-success border border-pat-success/20",
    DEGRADED: "bg-pat-warning/10 text-pat-warning border border-pat-warning/20",
    STALE: "bg-pat-warning/10 text-pat-warning border border-pat-warning/20",
    MISSING: "bg-pat-text-muted/10 text-pat-text-muted border border-pat-border",
    DISABLED: "bg-pat-text-muted/10 text-pat-text-muted border border-pat-border",
    NOT_CONFIGURED: "bg-pat-text-muted/10 text-pat-text-muted border border-pat-border",
    ERROR: "bg-pat-danger/10 text-pat-danger border border-pat-danger/20",
    UNAVAILABLE: "bg-pat-danger/10 text-pat-danger border border-pat-danger/20",
  };
  const cls = colors[status] || colors["MISSING"];
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium ${cls}`}>
      {status}
    </span>
  );
}

function ScoreBadge({ score }: { score: number }) {
  const color = score > 15 ? "text-pat-success" : score < -15 ? "text-pat-danger" : "text-pat-text-secondary";
  const label = score > 40 ? "Strong Bullish" : score > 15 ? "Bullish" : score < -40 ? "Strong Bearish" : score < -15 ? "Bearish" : "Neutral";
  return (
    <div className="flex flex-col">
      <span className={`text-lg font-bold tabular-nums ${color}`}>{score > 0 ? "+" : ""}{score.toFixed(1)}</span>
      <span className="text-[10px] text-pat-text-muted">{label}</span>
    </div>
  );
}

export default function AdminMacroIntelligencePage() {
  const { data: macro, isLoading } = useQuery<CrossMarketCurrent>({
    queryKey: ["admin-macro-current"],
    queryFn: async () => (await customInstance.get("/cross-market/current")).data,
    refetchInterval: 10000,
  });

  const { data: validation } = useQuery<ValidationStatus>({
    queryKey: ["admin-macro-validation"],
    queryFn: async () => (await customInstance.get("/cross-market/validation")).data,
    refetchInterval: 30000,
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-bold text-pat-text-primary">Macro Intelligence</h1>
        <div className="h-64 bg-pat-bg-surface-secondary/50 rounded animate-pulse" />
      </div>
    );
  }

  const driverLabels: Record<string, string> = {
    dxy: "DXY", eurusd: "EURUSD", cot: "COT", real_yields: "Real Yield",
    vix: "VIX", btc: "BTC", oil: "Oil", fed_context: "Fed Context",
    usdjpy: "USD/JPY", usdchf: "USD/CHF", etf_flows: "ETF Flows",
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Macro Intelligence</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Cross-market confluence engine status, driver health, and shadow validation progress.</p>
      </div>

      {/* Top Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <div className="text-[10px] text-pat-text-muted uppercase mb-1">Mode</div>
          <div className="text-sm font-bold text-pat-text-primary">{macro?.mode || "—"}</div>
        </div>
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <div className="text-[10px] text-pat-text-muted uppercase mb-1">Macro Score</div>
          {macro && <ScoreBadge score={macro.score} />}
        </div>
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <div className="text-[10px] text-pat-text-muted uppercase mb-1">Confidence</div>
          <div className="text-lg font-bold text-pat-text-primary tabular-nums">{macro ? (macro.confidence * 100).toFixed(0) : "—"}%</div>
        </div>
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <div className="text-[10px] text-pat-text-muted uppercase mb-1">Data Quality</div>
          {macro && <StatusBadge status={macro.data_quality} />}
        </div>
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <div className="text-[10px] text-pat-text-muted uppercase mb-1">Regime</div>
          <div className="text-sm font-bold text-pat-text-primary">{macro?.regime || "—"}</div>
        </div>
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <div className="text-[10px] text-pat-text-muted uppercase mb-1">Activation</div>
          <div className="text-sm font-bold text-pat-danger">{validation?.activation_eligible ? "ELIGIBLE" : "NOT ELIGIBLE"}</div>
        </div>
      </div>

      {/* Driver Health Table */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Driver Status</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-[10px] text-pat-text-muted border-b border-pat-border">
                <th className="text-left py-2 px-2">Driver</th>
                <th className="text-left py-2 px-2">Source</th>
                <th className="text-left py-2 px-2">Status</th>
                <th className="text-right py-2 px-2">Impact</th>
                <th className="text-right py-2 px-2">Confidence</th>
                <th className="text-right py-2 px-2">Freshness</th>
                <th className="text-right py-2 px-2">Eff. Weight</th>
                <th className="text-left py-2 px-2">Direction</th>
              </tr>
            </thead>
            <tbody>
              {macro?.drivers.map((drv, i) => (
                <tr key={i} className="border-b border-pat-border/30">
                  <td className="py-2 px-2 text-pat-text-primary font-medium">{driverLabels[drv.name] || drv.name}</td>
                  <td className="py-2 px-2 text-pat-text-muted">{drv.source}</td>
                  <td className="py-2 px-2"><StatusBadge status={drv.quality} /></td>
                  <td className="py-2 px-2 text-right tabular-nums text-pat-text-secondary">{drv.impact_score.toFixed(1)}</td>
                  <td className="py-2 px-2 text-right tabular-nums text-pat-text-muted">{(drv.confidence * 100).toFixed(0)}%</td>
                  <td className="py-2 px-2 text-right tabular-nums text-pat-text-muted">{(drv.freshness * 100).toFixed(0)}%</td>
                  <td className="py-2 px-2 text-right tabular-nums text-pat-text-muted">{drv.effective_weight.toFixed(2)}</td>
                  <td className="py-2 px-2 text-pat-text-secondary">{drv.direction}</td>
                </tr>
              ))}
              {/* Missing drivers */}
              {macro?.missing_drivers?.map((name, i) => (
                <tr key={`missing-${i}`} className="border-b border-pat-border/30">
                  <td className="py-2 px-2 text-pat-text-muted font-medium">{driverLabels[name] || name}</td>
                  <td className="py-2 px-2 text-pat-text-muted">—</td>
                  <td className="py-2 px-2"><StatusBadge status="NOT_CONFIGURED" /></td>
                  <td className="py-2 px-2 text-right text-pat-text-muted">—</td>
                  <td className="py-2 px-2 text-right text-pat-text-muted">—</td>
                  <td className="py-2 px-2 text-right text-pat-text-muted">—</td>
                  <td className="py-2 px-2 text-right text-pat-text-muted">—</td>
                  <td className="py-2 px-2 text-pat-text-muted">—</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Score Breakdown + Regime */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Score Breakdown</h2>
          <div className="space-y-1">
            {macro?.drivers.map((drv, i) => (
              <div key={i} className="flex items-center justify-between text-xs">
                <span className="text-pat-text-muted">{driverLabels[drv.name] || drv.name}</span>
                <span className={`tabular-nums font-medium ${drv.impact_score > 0 ? "text-pat-success" : drv.impact_score < 0 ? "text-pat-danger" : "text-pat-text-muted"}`}>
                  {drv.impact_score > 0 ? "+" : ""}{drv.impact_score.toFixed(1)}
                </span>
              </div>
            ))}
            <div className="border-t border-pat-border/50 pt-2 flex items-center justify-between text-sm">
              <span className="text-pat-text-primary font-semibold">Cross-Market Score</span>
              <span className={`tabular-nums font-bold ${macro && macro.score > 0 ? "text-pat-success" : "text-pat-danger"}`}>
                {macro ? `${macro.score > 0 ? "+" : ""}${macro.score.toFixed(1)}` : "—"}
              </span>
            </div>
            {macro?.score_adjustment !== 0 && (
              <div className="flex items-center justify-between text-xs">
                <span className="text-pat-text-muted">Score Adjustment</span>
                <span className="tabular-nums text-pat-warning">{macro?.score_adjustment.toFixed(1) ?? "0.0"}</span>
              </div>
            )}
          </div>
          {macro?.reason && (
            <div className="mt-3 text-[11px] text-pat-text-muted leading-relaxed">{macro.reason}</div>
          )}
        </div>

        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Regime & Risk</h2>
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <span className="text-pat-text-muted">Macro Regime</span>
              <span className="text-pat-text-primary font-medium">{macro?.regime || "Unknown"}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-pat-text-muted">Event Risk</span>
              <span className="text-pat-text-primary font-medium">{macro?.event_risk || "—"}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-pat-text-muted">Correlation Regime</span>
              <span className="text-pat-text-primary font-medium">{macro?.correlation_regime || "—"}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-pat-text-muted">Divergence</span>
              <span className="text-pat-text-primary font-medium">{macro?.divergence || "NONE"}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-pat-text-muted">Agreement</span>
              <span className="text-pat-text-primary tabular-nums">{macro ? (macro.agreement * 100).toFixed(0) : 0}%</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-pat-text-muted">Conflict</span>
              <span className="text-pat-text-primary tabular-nums">{macro ? (macro.conflict * 100).toFixed(0) : 0}%</span>
            </div>
          </div>
        </div>
      </div>

      {/* Shadow Validation Panel */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Shadow Validation</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="text-[10px] text-pat-text-muted uppercase mb-1">Usable Shadow Days</div>
            <div className="text-sm font-bold text-pat-text-primary tabular-nums">
              {validation?.usable_shadow_days || 0} / {validation?.minimum_days_required || 30}
            </div>
          </div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="text-[10px] text-pat-text-muted uppercase mb-1">Candidates</div>
            <div className="text-sm font-bold text-pat-text-primary tabular-nums">{validation?.total_candidates || 0}</div>
          </div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="text-[10px] text-pat-text-muted uppercase mb-1">Resolved Outcomes</div>
            <div className="text-sm font-bold text-pat-text-primary tabular-nums">{validation?.resolved_outcomes || 0}</div>
          </div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3">
            <div className="text-[10px] text-pat-text-muted uppercase mb-1">Activation Eligible</div>
            <div className={`text-sm font-bold ${validation?.activation_eligible ? "text-pat-success" : "text-pat-danger"}`}>
              {validation?.activation_eligible ? "YES" : "NO"}
            </div>
          </div>
        </div>
        <div className="mt-3 grid grid-cols-2 md:grid-cols-3 gap-3">
          <div className="flex items-center gap-2 text-xs">
            {validation?.ablation_ready ? <IconCheck size={14} className="text-pat-success" /> : <IconClock size={14} className="text-pat-text-muted" />}
            <span className="text-pat-text-muted">Ablation: {validation?.ablation_ready ? "Ready" : "Waiting"}</span>
          </div>
          <div className="flex items-center gap-2 text-xs">
            {validation?.walk_forward_ready ? <IconCheck size={14} className="text-pat-success" /> : <IconClock size={14} className="text-pat-text-muted" />}
            <span className="text-pat-text-muted">Walk-Forward: {validation?.walk_forward_ready ? "Ready" : "Waiting"}</span>
          </div>
          <div className="flex items-center gap-2 text-xs">
            {validation?.activation_eligible ? <IconCheck size={14} className="text-pat-success" /> : <IconX size={14} className="text-pat-danger" />}
            <span className="text-pat-text-muted">Activation: {validation?.activation_eligible ? "Eligible" : "Not Eligible"}</span>
          </div>
        </div>
        {validation?.message && (
          <div className="mt-3 text-[11px] text-pat-text-muted">{validation.message}</div>
        )}
      </div>
    </div>
  );
}
