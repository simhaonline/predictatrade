"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconUsers, IconReceipt, IconChartBar, IconActivity, IconTrendingUp, IconTrendingDown, IconMinus, IconBan } from "@tabler/icons-react";

interface TradingReport {
  summary: {
    total_signals: number;
    last_24h: {
      total: string; buy: string; sell: string; buy_candidate: string;
      sell_candidate: string; no_trade: string; blocked: string;
    };
  };
  by_direction: { direction: string; count: string; avg_score: string; avg_prob: string }[];
  by_strategy: { strategy_id: string; direction: string; count: string; avg_score: string }[];
  hourly_trend: { hour: string; direction: string; count: string }[];
  by_regime: { regime: string; count: string }[];
  by_session: { session: string; count: string }[];
  recent_signals: {
    signal_id: string; strategy_id: string; direction: string; raw_score: string;
    calibrated_probability: string; entry_price: string; stop_loss: string;
    tp1: string; tp2: string; tp3: string; regime: string; session: string;
    status: string; created_at: string;
  }[];
  gate_vetoes: { reason: string; count: string }[];
}

interface OverviewResponse {
  users: { total: string; active: string; suspended: string; new_this_month: string };
  subscriptions: { total: string; active: string; mrr: string };
  commissions: { total_entries: string; pending_amount: string; confirmed_amount: string };
  payouts: { total: string; pending: string; pending_amount: string };
}

const dirColor = (dir: string) => {
  switch (dir) {
    case "BUY": return "text-pat-success";
    case "SELL": return "text-pat-danger";
    case "BUY_CANDIDATE": return "text-pat-candidate-buy";
    case "SELL_CANDIDATE": return "text-pat-candidate-sell";
    case "NO-TRADE": return "text-pat-text-muted";
    case "BLOCKED": return "text-pat-warning";
    default: return "text-pat-text-secondary";
  }
};

const dirIcon = (dir: string) => {
  switch (dir) {
    case "BUY": return <IconTrendingUp size={16} className="text-pat-success" />;
    case "SELL": return <IconTrendingDown size={16} className="text-pat-danger" />;
    case "BUY_CANDIDATE": return <IconTrendingUp size={16} className="text-pat-candidate-buy" />;
    case "SELL_CANDIDATE": return <IconTrendingDown size={16} className="text-pat-candidate-sell" />;
    case "NO-TRADE": return <IconMinus size={16} className="text-pat-text-muted" />;
    case "BLOCKED": return <IconBan size={16} className="text-pat-warning" />;
    default: return null;
  }
};

export default function AdminTradingReportsPage() {
  const { data: report, isLoading } = useQuery<TradingReport>({
    queryKey: ["trading-reports"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/trading-reports");
      return res.data as TradingReport;
    },
    refetchInterval: 30000,
  });

  const { data: overview } = useQuery<OverviewResponse>({
    queryKey: ["admin-overview"],
    queryFn: async () => {
      const res = await customInstance.get("/admin/overview");
      return res.data as OverviewResponse;
    },
  });

  const { data: agents } = useQuery({
    queryKey: ["engine-agents"],
    queryFn: async () => {
      const res = await customInstance.get("/agents/status");
      return res.data;
    },
  });

  const last24 = report?.summary?.last_24h;
  const total24 = parseInt(last24?.total ?? "0");
  const buy24 = parseInt(last24?.buy ?? "0");
  const sell24 = parseInt(last24?.sell ?? "0");
  const candidate24 = parseInt(last24?.buy_candidate ?? "0") + parseInt(last24?.sell_candidate ?? "0");
  const noTrade24 = parseInt(last24?.no_trade ?? "0");
  const blocked24 = parseInt(last24?.blocked ?? "0");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Trading Reports</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Aggregate trading performance and signal activity across all strategies and clients.
        </p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Total Signals</span>
            <IconChartBar size={18} className="text-pat-info" />
          </div>
          <div className="text-2xl font-bold text-pat-text-primary">
            {report?.summary?.total_signals?.toLocaleString() ?? "—"}
          </div>
          <div className="text-xs text-pat-text-muted mt-1">{total24.toLocaleString()} in last 24h</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Actionable (BUY/SELL)</span>
            <IconTrendingUp size={18} className="text-pat-success" />
          </div>
          <div className="text-2xl font-bold text-pat-success">
            {(buy24 + sell24).toLocaleString()}
          </div>
          <div className="text-xs text-pat-text-muted mt-1">{buy24} BUY · {sell24} SELL (24h)</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Candidates</span>
            <IconActivity size={18} className="text-pat-candidate-buy" />
          </div>
          <div className="text-2xl font-bold text-pat-candidate-buy">
            {candidate24.toLocaleString()}
          </div>
          <div className="text-xs text-pat-text-muted mt-1">Advisory signals (24h)</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-pat-text-secondary">Connected Agents</span>
            <IconActivity size={18} className="text-cyan-400" />
          </div>
          <div className="text-2xl font-bold text-pat-text-primary">
            {Number(agents?.agents_connected ?? 0)}
          </div>
          <div className="text-xs text-pat-text-muted mt-1">Windows Agent connections</div>
        </div>
      </div>

      {/* Signal Statistics by Direction */}
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Signal Statistics by Direction</h2>
        {isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading...</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-pat-border">
                  <th className="text-left py-2 px-3 text-pat-text-muted font-medium">Direction</th>
                  <th className="text-right py-2 px-3 text-pat-text-muted font-medium">Count</th>
                  <th className="text-right py-2 px-3 text-pat-text-muted font-medium">Avg Score</th>
                  <th className="text-right py-2 px-3 text-pat-text-muted font-medium">Avg Probability</th>
                  <th className="text-right py-2 px-3 text-pat-text-muted font-medium">% of Total</th>
                </tr>
              </thead>
              <tbody>
                {report?.by_direction?.map((row) => {
                  const count = parseInt(row.count);
                  const total = report?.summary?.total_signals ?? 1;
                  const pct = ((count / total) * 100).toFixed(1);
                  const prob = parseFloat(row.avg_prob);
                  return (
                    <tr key={row.direction} className="border-b border-pat-border/50 hover:bg-pat-bg-page/50">
                      <td className="py-2 px-3">
                        <div className="flex items-center gap-2">
                          {dirIcon(row.direction)}
                          <span className={`font-medium ${dirColor(row.direction)}`}>{row.direction}</span>
                        </div>
                      </td>
                      <td className="text-right py-2 px-3 text-pat-text-primary font-semibold">{count.toLocaleString()}</td>
                      <td className="text-right py-2 px-3 text-pat-text-secondary">{row.avg_score}</td>
                      <td className="text-right py-2 px-3 text-pat-text-secondary">
                        {prob > 0 ? `${(prob * 100).toFixed(2)}%` : "Pending"}
                      </td>
                      <td className="text-right py-2 px-3 text-pat-text-muted">{pct}%</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Strategy Breakdown */}
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Strategy Performance Breakdown</h2>
        {isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading...</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-pat-border">
                  <th className="text-left py-2 px-3 text-pat-text-muted font-medium">Strategy</th>
                  <th className="text-left py-2 px-3 text-pat-text-muted font-medium">Direction</th>
                  <th className="text-right py-2 px-3 text-pat-text-muted font-medium">Count</th>
                  <th className="text-right py-2 px-3 text-pat-text-muted font-medium">Avg Score</th>
                </tr>
              </thead>
              <tbody>
                {report?.by_strategy?.map((row, i) => (
                  <tr key={i} className="border-b border-pat-border/50 hover:bg-pat-bg-page/50">
                    <td className="py-2 px-3 text-pat-text-primary font-medium">{row.strategy_id}</td>
                    <td className="py-2 px-3">
                      <span className={dirColor(row.direction)}>{row.direction}</span>
                    </td>
                    <td className="text-right py-2 px-3 text-pat-text-primary">{parseInt(row.count).toLocaleString()}</td>
                    <td className="text-right py-2 px-3 text-pat-text-secondary">{row.avg_score}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Two column: Regime + Session */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Regime Distribution</h2>
          <div className="space-y-2">
            {report?.by_regime?.map((row) => {
              const count = parseInt(row.count);
              const total = report?.summary?.total_signals ?? 1;
              const pct = (count / total) * 100;
              return (
                <div key={row.regime} className="flex items-center gap-3">
                  <span className="text-sm text-pat-text-secondary w-40 truncate">{row.regime}</span>
                  <div className="flex-1 bg-pat-bg-page rounded-full h-5 overflow-hidden">
                    <div className="bg-pat-info h-full rounded-full flex items-center justify-end px-2" style={{ width: `${Math.max(pct, 2)}%` }}>
                      <span className="text-xs text-white font-medium">{count.toLocaleString()}</span>
                    </div>
                  </div>
                  <span className="text-xs text-pat-text-muted w-12 text-right">{pct.toFixed(1)}%</span>
                </div>
              );
            })}
          </div>
        </div>

        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Session Distribution</h2>
          <div className="space-y-2">
            {report?.by_session?.map((row) => {
              const count = parseInt(row.count);
              const total = report?.summary?.total_signals ?? 1;
              const pct = (count / total) * 100;
              return (
                <div key={row.session} className="flex items-center gap-3">
                  <span className="text-sm text-pat-text-secondary w-40 truncate">{row.session}</span>
                  <div className="flex-1 bg-pat-bg-page rounded-full h-5 overflow-hidden">
                    <div className="bg-pat-session h-full rounded-full flex items-center justify-end px-2" style={{ width: `${Math.max(pct, 2)}%` }}>
                      <span className="text-xs text-black font-medium">{count.toLocaleString()}</span>
                    </div>
                  </div>
                  <span className="text-xs text-pat-text-muted w-12 text-right">{pct.toFixed(1)}%</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Recent Actionable Signals */}
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Recent Actionable Signals</h2>
        {isLoading ? (
          <div className="text-sm text-pat-text-muted">Loading...</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-pat-border">
                  <th className="text-left py-2 px-2 text-pat-text-muted font-medium">Time</th>
                  <th className="text-left py-2 px-2 text-pat-text-muted font-medium">Strategy</th>
                  <th className="text-left py-2 px-2 text-pat-text-muted font-medium">Dir</th>
                  <th className="text-right py-2 px-2 text-pat-text-muted font-medium">Score</th>
                  <th className="text-right py-2 px-2 text-pat-text-muted font-medium">Entry</th>
                  <th className="text-right py-2 px-2 text-pat-text-muted font-medium">SL</th>
                  <th className="text-right py-2 px-2 text-pat-text-muted font-medium">TP1</th>
                  <th className="text-left py-2 px-2 text-pat-text-muted font-medium">Regime</th>
                  <th className="text-left py-2 px-2 text-pat-text-muted font-medium">Session</th>
                </tr>
              </thead>
              <tbody>
                {report?.recent_signals?.slice(0, 20).map((sig, i) => (
                  <tr key={i} className="border-b border-pat-border/30 hover:bg-pat-bg-page/50">
                    <td className="py-1.5 px-2 text-pat-text-muted">
                      {sig.created_at ? new Date(sig.created_at).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : "—"}
                    </td>
                    <td className="py-1.5 px-2 text-pat-text-secondary">{sig.strategy_id?.replace("_", " ")}</td>
                    <td className="py-1.5 px-2">
                      <span className={`font-medium ${dirColor(sig.direction)}`}>{sig.direction}</span>
                    </td>
                    <td className="text-right py-1.5 px-2 text-pat-text-primary font-semibold">{sig.raw_score}</td>
                    <td className="text-right py-1.5 px-2 text-pat-text-secondary">{sig.entry_price ? parseFloat(sig.entry_price).toFixed(2) : "—"}</td>
                    <td className="text-right py-1.5 px-2 text-pat-danger">{sig.stop_loss ? parseFloat(sig.stop_loss).toFixed(2) : "—"}</td>
                    <td className="text-right py-1.5 px-2 text-pat-success">{sig.tp1 ? parseFloat(sig.tp1).toFixed(2) : "—"}</td>
                    <td className="py-1.5 px-2 text-pat-text-muted">{sig.regime?.replace("_", " ")}</td>
                    <td className="py-1.5 px-2 text-pat-text-muted">{sig.session?.replace("_", " ")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Gate Vetoes + Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Gate Veto Reasons</h2>
          {report?.gate_vetoes && report.gate_vetoes.length > 0 ? (
            <div className="space-y-2">
              {report.gate_vetoes.map((v, i) => (
                <div key={i} className="flex items-center justify-between">
                  <span className="text-sm text-pat-text-secondary">{v.reason}</span>
                  <span className="text-sm font-semibold text-pat-warning">{v.count}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-sm text-pat-text-muted">No gate veto data available</div>
          )}
        </div>

        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Account Overview</h2>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <div className="text-xs text-pat-text-muted">Total Users</div>
              <div className="text-lg font-bold text-pat-text-primary">{parseInt(overview?.users?.total ?? "0").toLocaleString()}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">Active Subs</div>
              <div className="text-lg font-bold text-pat-text-primary">{overview?.subscriptions?.active ?? "0"}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">Commissions</div>
              <div className="text-lg font-bold text-pat-text-primary">{overview?.commissions?.total_entries ?? "0"}</div>
            </div>
            <div>
              <div className="text-xs text-pat-text-muted">MRR</div>
              <div className="text-lg font-bold text-pat-success">${parseFloat(overview?.subscriptions?.mrr ?? "0").toFixed(2)}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
