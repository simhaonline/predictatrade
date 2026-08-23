"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import {
  IconChartBar, IconCoin, IconReceipt, IconActivity,
  IconTrendingUp, IconBolt, IconBrandWindows,
  IconTerminal2, IconInfoCircle,
} from "@tabler/icons-react";
import { format } from "date-fns";

interface EngineSignal {
  ID: string; Direction: string; StrategyID: string; Status: string;
  RawScore: string; CalibratedProbability: string;
  EntryPrice: string; StopLoss: string; TP1: string; TP2: string; TP3: string;
  RealizedPnL: string; RealizedR: string; ExitPrice: string; ExitReason: string;
  ClosedAt: string; CreatedAt: string; Regime: string; Session: string;
  Executable: boolean; Symbol: string;
}

interface TerminalActivation {
  client_type: string;
  terminal_build?: string;
  ea_version?: string;
  broker_name?: string;
  broker_server?: string;
  mt_account_login?: string;
  activated_at?: string;
  balance?: number;
  equity?: number;
  profit?: number;
  currency?: string;
  open_positions?: number;
  buy_positions?: number;
  sell_positions?: number;
  total_lots?: number;
  floating_pnl?: number;
  last_account_update?: string;
}

interface UserDevice {
  id: string;
  device_name: string;
  hostname: string;
  os: string;
  agent_version: string;
  status: string;
  installation_id: string;
  fingerprint_hash: string | null;
  license_key: string | null;
  max_devices?: number;
  max_mt_accounts?: number;
  activations: TerminalActivation[] | null;
}

interface AgentsStatus {
  agents_connected: number;
  master_node_connected: boolean;
  snapshot_count: number;
}

interface CommissionSummary {
  total_amount: string; pending_count: string; confirmed_count: string;
  pending_amount: string; confirmed_amount: string;
}

// Master Node account numbers that should NOT be shown to users (admin-only)
const MASTER_NODE_ACCOUNTS = ["1013700717"];

export default function UserTradingReportsPage() {
  const { data: devices } = useQuery<UserDevice[]>({
    queryKey: ["user-trading-devices"],
    queryFn: async () => (await customInstance.get("/licensing/devices")).data,
    refetchInterval: 5000,
  });

  const { data: commissionSummary } = useQuery<CommissionSummary>({
    queryKey: ["user-trading-commission"],
    queryFn: async () => (await customInstance.get("/commissions/summary")).data,
  });

  const { data: subscriptions } = useQuery({
    queryKey: ["user-trading-subs"],
    queryFn: async () => (await customInstance.get("/subscriptions")).data,
  });
  const { data: invoices } = useQuery({
    queryKey: ["user-trading-invoices"],
    queryFn: async () => (await customInstance.get("/billing/invoices")).data,
  });

  const { data: signalsData } = useQuery<{ signals: EngineSignal[] }>({
    queryKey: ["user-trading-signals"],
    queryFn: async () => (await customInstance.get("/signals")).data,
    refetchInterval: 15000,
  });

  const { data: agentsStatus } = useQuery<AgentsStatus>({
    queryKey: ["user-trading-agents"],
    queryFn: async () => (await customInstance.get("/agents/status")).data,
    refetchInterval: 10000,
  });

  const signals = signalsData?.signals ?? [];

  // Filter out Master Node terminals — those are admin-only, not for client dashboard
  const allTerminals = (devices?.flatMap(d =>
    (d.activations || []).map(a => ({ ...a, deviceName: d.device_name, deviceStatus: d.status }))
  ) ?? []).filter(t => !MASTER_NODE_ACCOUNTS.includes(t.mt_account_login || ""));

  const totalSignals = signals.length;
  const directional = signals.filter(s => s.Direction !== "NO-TRADE");
  const buySignals = signals.filter(s => s.Direction === "BUY");
  const sellSignals = signals.filter(s => s.Direction === "SELL");
  const buyCandidates = signals.filter(s => s.Direction === "BUY_CANDIDATE");
  const sellCandidates = signals.filter(s => s.Direction === "SELL_CANDIDATE");

  // Aggregate account data from CLIENT terminals only
  const totalBalance = allTerminals.reduce((sum, t) => sum + (t.balance ?? 0), 0);
  const totalEquity = allTerminals.reduce((sum, t) => sum + (t.equity ?? 0), 0);
  const totalProfit = allTerminals.reduce((sum, t) => sum + (t.profit ?? 0), 0);
  const totalPositions = allTerminals.reduce((sum, t) => sum + (t.open_positions ?? 0), 0);
  const totalBuy = allTerminals.reduce((sum, t) => sum + (t.buy_positions ?? 0), 0);
  const totalSell = allTerminals.reduce((sum, t) => sum + (t.sell_positions ?? 0), 0);



  const strategies = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"];
  const strategyStats = strategies.map(strat => {
    const ss = signals.filter(s => s.StrategyID === strat);
    const sd = ss.filter(s => s.Direction !== "NO-TRADE");
    const avgScore = sd.length > 0 ? sd.reduce((sum, s) => sum + parseFloat(s.RawScore || "0"), 0) / sd.length : 0;
    const rrVals = sd.map(s => {
      const e = parseFloat(s.EntryPrice || "0"), sl = parseFloat(s.StopLoss || "0"), tp1 = parseFloat(s.TP1 || "0");
      const slD = e && sl ? Math.abs(e - sl) : 0;
      return slD && tp1 ? Math.abs(tp1 - e) / slD : 0;
    }).filter(r => r > 0);
    const avgRR = rrVals.length > 0 ? rrVals.reduce((a, b) => a + b, 0) / rrVals.length : 0;
    return { strategy: strat, total: ss.length, directional: sd.length, noTrade: ss.length - sd.length, avgScore: avgScore.toFixed(1), avgRR: avgRR.toFixed(2) };
  });

  const dirColor = (dir: string): string => {
    if (dir === "BUY") return "text-pat-success";
    if (dir === "SELL") return "text-pat-danger";
    if (dir.includes("BUY_CANDIDATE")) return "text-pat-warning";
    if (dir.includes("SELL_CANDIDATE")) return "text-pat-candidate-sell";
    return "text-pat-text-muted";
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Trading Reports</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Your XAUUSD trading performance and connected terminals.</p>
      </div>

      {/* Connection status */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-3">
            <div className={`flex items-center justify-center w-10 h-10 rounded-lg ${(agentsStatus?.agents_connected ?? 0) > 0 ? "bg-pat-success/10" : "bg-pat-danger/10"}`}>
              <IconBrandWindows size={20} className={(agentsStatus?.agents_connected ?? 0) > 0 ? "text-pat-success" : "text-pat-danger"} />
            </div>
            <div>
              <div className="text-sm font-semibold text-pat-text-primary">Platform Connection</div>
              <div className="text-xs text-pat-text-muted">
                {agentsStatus?.agents_connected ?? 0} agent(s) · {allTerminals.length} client terminal(s)
                {agentsStatus?.master_node_connected ? " · Master Node: ONLINE" : " · Master Node: OFFLINE"}
              </div>
            </div>
          </div>
          <span className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium ${
            (agentsStatus?.agents_connected ?? 0) > 0
              ? "bg-pat-success/10 text-pat-success border border-pat-success/20"
              : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
          }`}>
            <span className={`inline-block h-2 w-2 rounded-full ${(agentsStatus?.agents_connected ?? 0) > 0 ? "bg-pat-success animate-pulse" : "bg-pat-danger"}`} />
            {(agentsStatus?.agents_connected ?? 0) > 0 ? "LIVE" : "OFFLINE"}
          </span>
        </div>
      </div>

      {/* Summary cards — aggregated from CLIENT terminals */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          { label: "Total Signals", value: totalSignals, sub: `${directional.length} directional`, icon: IconActivity, color: "text-pat-text-primary" },
          { label: "Active Signals", value: directional.length, sub: `${buySignals.length} BUY · ${sellSignals.length} SELL`, icon: IconTrendingUp, color: "text-pat-success" },
          { label: "Candidates", value: buyCandidates.length + sellCandidates.length, sub: "advisory signals", icon: IconBolt, color: "text-pat-warning" },
          { label: "Open Positions", value: totalPositions, sub: `${totalBuy} BUY · ${totalSell} SELL`, icon: IconChartBar, color: "text-pat-info" },
        ].map((c) => (
          <div key={c.label} className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[10px] text-pat-text-muted uppercase">{c.label}</span>
              <c.icon size={16} className={c.color} />
            </div>
            <div className={`text-lg font-bold ${c.color} tabular-nums`}>{c.value}</div>
            <div className="text-[10px] text-pat-text-muted mt-0.5">{c.sub}</div>
          </div>
        ))}
      </div>

      {/* Account cards — aggregated from CLIENT terminals */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          { label: "Account Balance", value: `$${totalBalance.toFixed(2)}`, sub: `Equity: $${totalEquity.toFixed(2)}`, icon: IconCoin, color: "text-pat-text-primary" },
          { label: "Floating P/L", value: `$${totalProfit.toFixed(2)}`, sub: totalProfit >= 0 ? "In profit" : "In loss", icon: IconTrendingUp, color: totalProfit >= 0 ? "text-pat-success" : "text-pat-danger" },
          { label: "Total Earnings", value: `$${parseFloat(commissionSummary?.total_amount ?? "0").toFixed(2)}`, sub: `${commissionSummary?.confirmed_count ?? "0"} confirmed`, icon: IconCoin, color: "text-pat-success" },
          { label: "Subscriptions", value: Array.isArray(subscriptions) ? subscriptions.length : 0, sub: `${Array.isArray(invoices) ? invoices.length : 0} invoices`, icon: IconReceipt, color: "text-pat-info" },
        ].map((c) => (
          <div key={c.label} className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[10px] text-pat-text-muted uppercase">{c.label}</span>
              <c.icon size={16} className={c.color} />
            </div>
            <div className={`text-lg font-bold ${c.color} tabular-nums`}>{c.value}</div>
            <div className="text-[10px] text-pat-text-muted mt-0.5">{c.sub}</div>
          </div>
        ))}
      </div>

      {/* Client terminals with account data */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Your Connected Client Terminals</h2>
        {allTerminals.length === 0 ? (
          <div className="text-xs text-pat-text-muted py-4 text-center">No client terminals registered. Install the MT4/MT5 EA to connect.</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {allTerminals.map((t, i) => (
              <div key={i} className="rounded-lg border border-pat-border/60 bg-pat-bg-surface-secondary/30 p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <IconTerminal2 size={16} className="text-pat-text-secondary" />
                    <span className="text-sm font-medium text-pat-text-primary">{t.client_type}</span>
                  </div>
                  <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium ${
                    t.deviceStatus === "ONLINE"
                      ? "bg-pat-success/10 text-pat-success border border-pat-success/20"
                      : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
                  }`}>
                    <span className={`inline-block h-1.5 w-1.5 rounded-full ${t.deviceStatus === "ONLINE" ? "bg-pat-success animate-pulse" : "bg-pat-danger"}`} />
                    {t.deviceStatus === "ONLINE" ? "Online" : "Offline"}
                  </span>
                </div>
                <div className="space-y-0.5 text-xs">
                  <div><span className="text-pat-text-muted">Broker:</span> <span className="text-pat-text-secondary">{t.broker_name || "—"}</span></div>
                  <div><span className="text-pat-text-muted">Account:</span> <span className="text-pat-text-secondary font-mono">{t.mt_account_login || "—"}</span></div>
                  {t.broker_server && <div><span className="text-pat-text-muted">Server:</span> <span className="text-pat-text-secondary">{t.broker_server}</span></div>}
                  {t.terminal_build && <div><span className="text-pat-text-muted">Build:</span> <span className="text-pat-text-secondary">{t.terminal_build}</span></div>}
                  <div><span className="text-pat-text-muted">EA Version:</span> <span className="text-pat-text-secondary">{t.ea_version || "—"}</span></div>
                </div>
                {/* Account data */}
                <div className="mt-2 pt-2 border-t border-pat-border/30 grid grid-cols-3 gap-2 text-xs">
                  <div>
                    <span className="text-pat-text-muted">Balance</span>
                    <div className="font-mono font-medium text-pat-text-primary tabular-nums">${(t.balance ?? 0).toFixed(2)}</div>
                  </div>
                  <div>
                    <span className="text-pat-text-muted">Equity</span>
                    <div className="font-mono font-medium text-pat-text-primary tabular-nums">${(t.equity ?? 0).toFixed(2)}</div>
                  </div>
                  <div>
                    <span className="text-pat-text-muted">P/L</span>
                    <div className={`font-mono font-medium tabular-nums ${(t.profit ?? 0) >= 0 ? "text-pat-success" : "text-pat-danger"}`}>${(t.profit ?? 0).toFixed(2)}</div>
                  </div>
                  <div>
                    <span className="text-pat-text-muted">Positions</span>
                    <div className="font-mono font-medium text-pat-text-secondary tabular-nums">{t.open_positions ?? 0} ({t.buy_positions ?? 0}B/{t.sell_positions ?? 0}S)</div>
                  </div>
                  <div>
                    <span className="text-pat-text-muted">Lots</span>
                    <div className="font-mono font-medium text-pat-text-secondary tabular-nums">{(t.total_lots ?? 0).toFixed(2)}</div>
                  </div>
                  <div>
                    <span className="text-pat-text-muted">Float P/L</span>
                    <div className={`font-mono font-medium tabular-nums ${(t.floating_pnl ?? 0) >= 0 ? "text-pat-success" : "text-pat-danger"}`}>${(t.floating_pnl ?? 0).toFixed(2)}</div>
                  </div>
                </div>
                {t.last_account_update && (
                  <div className="text-[9px] text-pat-text-muted/60 mt-1">Updated: {new Date(t.last_account_update).toLocaleString()}</div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Strategy performance */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Strategy Performance</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[10px] text-pat-text-muted border-b border-pat-border">
                <th className="text-left py-2 px-3">Strategy</th>
                <th className="text-center py-2 px-3">Total</th>
                <th className="text-center py-2 px-3">Directional</th>
                <th className="text-center py-2 px-3">No-Trade</th>
                <th className="text-center py-2 px-3">Avg Score</th>
                <th className="text-center py-2 px-3">Projected R:R</th>
              </tr>
            </thead>
            <tbody>
              {strategyStats.map((stat) => (
                <tr key={stat.strategy} className="border-b border-pat-border/30 hover:bg-pat-bg-surface-secondary/20">
                  <td className="py-2 px-3 text-pat-text-primary font-medium text-xs">{stat.strategy.replace(/_/g, " ")}</td>
                  <td className="py-2 px-3 text-center text-pat-text-secondary tabular-nums">{stat.total}</td>
                  <td className="py-2 px-3 text-center text-pat-success tabular-nums">{stat.directional}</td>
                  <td className="py-2 px-3 text-center text-pat-text-muted tabular-nums">{stat.noTrade}</td>
                  <td className="py-2 px-3 text-center text-pat-text-secondary tabular-nums">{stat.avgScore}</td>
                  <td className="py-2 px-3 text-center">
                    <span className={`text-xs font-medium tabular-nums ${parseFloat(stat.avgRR) >= 1.0 ? "text-pat-success" : "text-pat-text-secondary"}`}>{stat.avgRR}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Recent signal setups */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Recent Signal Trade Setups</h2>
        {directional.length === 0 ? (
          <div className="text-xs text-pat-text-muted py-4 text-center">No directional signals detected yet.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-[10px] text-pat-text-muted border-b border-pat-border">
                  <th className="text-left py-2 px-2">Direction</th>
                  <th className="text-left py-2 px-2">Strategy</th>
                  <th className="text-right py-2 px-2">Score</th>
                  <th className="text-right py-2 px-2">Entry</th>
                  <th className="text-right py-2 px-2">SL</th>
                  <th className="text-right py-2 px-2">TP1</th>
                  <th className="text-right py-2 px-2">R:R</th>
                  <th className="text-left py-2 px-2">Time</th>
                </tr>
              </thead>
              <tbody>
                {directional.slice(0, 15).map((s) => {
                  const entry = parseFloat(s.EntryPrice || "0");
                  const sl = parseFloat(s.StopLoss || "0");
                  const tp1 = parseFloat(s.TP1 || "0");
                  const slDist = entry && sl ? Math.abs(entry - sl) : 0;
                  const rr1 = slDist && tp1 ? Math.abs(tp1 - entry) / slDist : 0;
                  return (
                    <tr key={s.ID} className="border-b border-pat-border/30 hover:bg-pat-bg-surface-secondary/20">
                      <td className="py-1.5 px-2"><span className={`font-bold ${dirColor(s.Direction)}`}>{s.Direction}</span>{s.Executable && <span className="ml-1 text-[9px] text-pat-success">EXEC</span>}</td>
                      <td className="py-1.5 px-2 text-pat-text-secondary">{s.StrategyID?.replace(/_/g, " ")}</td>
                      <td className="py-1.5 px-2 text-right text-pat-text-primary tabular-nums">{parseFloat(s.RawScore || "0").toFixed(1)}</td>
                      <td className="py-1.5 px-2 text-right font-mono text-pat-text-primary tabular-nums">{entry > 0 ? entry.toFixed(2) : "—"}</td>
                      <td className="py-1.5 px-2 text-right font-mono text-pat-danger tabular-nums">{sl > 0 ? sl.toFixed(2) : "—"}</td>
                      <td className="py-1.5 px-2 text-right font-mono text-pat-success tabular-nums">{tp1 > 0 ? tp1.toFixed(2) : "—"}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums"><span className={rr1 >= 1.0 ? "text-pat-success" : "text-pat-text-secondary"}>{rr1 > 0 ? rr1.toFixed(2) : "—"}</span></td>
                      <td className="py-1.5 px-2 text-pat-text-muted">{s.CreatedAt && s.CreatedAt !== "0001-01-01T00:00:00Z" ? format(new Date(s.CreatedAt), "MMM d, HH:mm") : "—"}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Commission */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-3">Commission Breakdown</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3"><div className="text-xs text-pat-text-muted">Confirmed</div><div className="text-lg font-semibold text-pat-success">${parseFloat(commissionSummary?.confirmed_amount ?? "0").toFixed(2)}</div></div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3"><div className="text-xs text-pat-text-muted">Pending</div><div className="text-lg font-semibold text-pat-warning">${parseFloat(commissionSummary?.pending_amount ?? "0").toFixed(2)}</div></div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3"><div className="text-xs text-pat-text-muted">Total</div><div className="text-lg font-semibold text-pat-text-primary">${parseFloat(commissionSummary?.total_amount ?? "0").toFixed(2)}</div></div>
          <div className="rounded-lg bg-pat-bg-surface-secondary/30 p-3"><div className="text-xs text-pat-text-muted">Invoices</div><div className="text-lg font-semibold text-pat-text-primary">{Array.isArray(invoices) ? invoices.length : 0}</div></div>
        </div>
      </div>

      {/* Info */}
      <div className="rounded-xl border border-pat-info/20 bg-pat-info/5 p-4">
        <div className="flex items-start gap-2">
          <IconInfoCircle size={16} className="text-pat-info shrink-0 mt-0.5" />
          <div className="text-[11px] text-pat-text-muted leading-relaxed">
            Account balance, equity, and P/L are captured from your MT4/MT5 terminals via the Windows Agent.
            If values show $0.00, ensure you have the latest EA version (v1.08+) installed — download it from the
            MetaTrader Client page. The EA sends account data (balance, equity, P&L, positions) to the platform
            on initialization and during license checks. {(agentsStatus?.agents_connected ?? 0) > 0 && `${agentsStatus?.agents_connected} agent(s) are connected and sending data.`}
          </div>
        </div>
      </div>
    </div>
  );
}
