"use client";

import { useState, useEffect, useCallback } from "react";
import {
  IconPlayerPlay, IconDownload, IconHistory, IconChevronDown,
  IconChartBar, IconCoin, IconTrendingUp, IconTrendingDown, IconLoader
} from "@tabler/icons-react";
import {
  fetchAvailableData, fetchRuns, runBacktest, downloadCSV,
  type DataSummary, type BacktestRun, type RunBacktestResponse
} from "@/lib/backtest-api";

const STRATEGIES = [
  { id: "STANDARD_SCALPING", label: "Standard Scalping", desc: "M5 scalping, tight SL/TP" },
  { id: "ULTRA_SCALPING", label: "Ultra Scalping", desc: "M1 ultra-fast, very tight" },
  { id: "STANDARD_SWING", label: "Standard Swing", desc: "M15/H1 swing trading" },
  { id: "TREND_SWING", label: "Trend Swing", desc: "H1/H4 trend following" },
  { id: "MARNIE_FIB", label: "EQFE", desc: "H1 Fibonacci confluence (SHADOW)" },
];

const TIMEFRAMES = ["M1", "M5", "M15", "M30", "H1", "H4", "D1"];

export default function BacktestPanel({ isAdmin }: { isAdmin?: boolean }) {
  const [mounted, setMounted] = useState(false);
  const [strategy, setStrategy] = useState("STANDARD_SCALPING");
  const [timeframe, setTimeframe] = useState("M5");
  const [startDate, setStartDate] = useState("2025-06-01");
  const [endDate, setEndDate] = useState("2025-06-30");
  const [balance, setBalance] = useState(10000);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<RunBacktestResponse | null>(null);
  const [runs, setRuns] = useState<BacktestRun[]>([]);
  const [dataInfo, setDataInfo] = useState<DataSummary[]>([]);
  const [loadingData, setLoadingData] = useState(true);
  const [error, setError] = useState("");
  const [showResults, setShowResults] = useState(true);
  const [showHistory, setShowHistory] = useState(false);

  const loadData = useCallback(async () => {
    try {
      const [data, history] = await Promise.all([fetchAvailableData(), fetchRuns()]);
      setDataInfo(data);
      setRuns(history);
      const tf = data.find(d => d.timeframe === timeframe);
      if (tf) { setStartDate(tf.min_date); setEndDate(tf.max_date); }
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load data"); }
    finally { setLoadingData(false); }
  }, [timeframe]);

  useEffect(() => {
    const mountTimer = window.setTimeout(() => setMounted(true), 0);
    return () => window.clearTimeout(mountTimer);
  }, []);
  useEffect(() => {
    if (!mounted) return;
    const loadTimer = window.setTimeout(() => { void loadData(); }, 0);
    return () => window.clearTimeout(loadTimer);
  }, [loadData, mounted]);

  if (!mounted) return null;

  const handleRun = async () => {
    setRunning(true); setError(""); setResult(null);
    try {
      const res = await runBacktest({ strategy, timeframe, startDate, endDate, initialBalance: balance });
      setResult(res);
      const history = await fetchRuns(); setRuns(history);
    } catch (e) { setError(e instanceof Error ? e.message : "Backtest failed"); }
    finally { setRunning(false); }
  };

  const handleDownload = async (runId: string) => {
    try {
      await downloadCSV(runId);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to download CSV");
    }
  };

  const fmt = (v: string | number, suffix = "") => {
    const n = typeof v === "string" ? parseFloat(v) : v;
    return isNaN(n) ? "—" : `${n.toLocaleString(undefined, { maximumFractionDigits: 2 })}${suffix}`;
  };

  return (
    <div className="space-y-4">
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-4">
          <IconChartBar size={18} className="text-pat-primary" />
          <h2 className="text-sm font-semibold text-pat-text-primary">
            {isAdmin ? "Backtesting Engine" : "Run Backtest"}
          </h2>
          <span className="text-xs text-pat-text-muted ml-auto">Real Historical Data</span>
        </div>

        {loadingData ? (
          <div className="text-xs text-pat-text-muted flex items-center gap-1">
            <IconLoader size={12} className="animate-spin" /> Loading data...
          </div>
        ) : (
          <div className="mb-4 p-3 bg-pat-bg-surface-secondary rounded-md">
            <div className="text-xs text-pat-text-muted mb-2">Available Data (click to select):</div>
            <div className="flex flex-wrap gap-2">
              {dataInfo.map(d => (
                <button key={d.timeframe}
                  onClick={() => { setTimeframe(d.timeframe); setStartDate(d.min_date); setEndDate(d.max_date); }}
                  className={`text-xs px-2 py-1 rounded border transition-colors ${timeframe === d.timeframe
                    ? "border-pat-primary bg-pat-bg-surface text-pat-primary font-medium ring-1 ring-pat-primary/30"
                    : "border-pat-border text-pat-text-secondary hover:border-pat-primary/50"}`}
                >
                  {d.timeframe} ({parseInt(d.candle_count).toLocaleString()})
                </button>
              ))}
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
          <div>
            <label className="text-xs text-pat-text-muted mb-1 block">Strategy</label>
            <select value={strategy} onChange={e => setStrategy(e.target.value)}
              className="w-full bg-pat-bg-surface-secondary border border-pat-border rounded-md px-3 py-2 text-sm text-pat-text-primary focus:border-pat-primary outline-none">
              {STRATEGIES.map(s => <option key={s.id} value={s.id}>{s.label}</option>)}
            </select>
          </div>
          <div>
            <label className="text-xs text-pat-text-muted mb-1 block">Timeframe</label>
            <select value={timeframe} onChange={e => setTimeframe(e.target.value)}
              className="w-full bg-pat-bg-surface-secondary border border-pat-border rounded-md px-3 py-2 text-sm text-pat-text-primary focus:border-pat-primary outline-none">
              {TIMEFRAMES.map(tf => <option key={tf} value={tf}>{tf}</option>)}
            </select>
          </div>
          <div>
            <label className="text-xs text-pat-text-muted mb-1 block">Start Date</label>
            <input type="date" value={startDate} onChange={e => setStartDate(e.target.value)}
              className="w-full bg-pat-bg-surface-secondary border border-pat-border rounded-md px-3 py-2 text-sm text-pat-text-primary focus:border-pat-primary outline-none" />
          </div>
          <div>
            <label className="text-xs text-pat-text-muted mb-1 block">End Date</label>
            <input type="date" value={endDate} onChange={e => setEndDate(e.target.value)}
              className="w-full bg-pat-bg-surface-secondary border border-pat-border rounded-md px-3 py-2 text-sm text-pat-text-primary focus:border-pat-primary outline-none" />
          </div>
        </div>

        <div className="flex items-end gap-3 mt-3">
          <div>
            <label className="text-xs text-pat-text-muted mb-1 block">Balance ($)</label>
            <input type="number" value={balance} onChange={e => setBalance(parseInt(e.target.value) || 10000)}
              className="w-32 bg-pat-bg-surface-secondary border border-pat-border rounded-md px-3 py-2 text-sm text-pat-text-primary focus:border-pat-primary outline-none" />
          </div>
          <button onClick={handleRun} disabled={running}
            className="flex items-center gap-2 px-4 py-2 bg-pat-primary text-pat-primary-foreground rounded-md text-sm font-medium hover:bg-pat-primary-hover transition-colors disabled:opacity-50">
            {running ? <><IconLoader size={16} className="animate-spin" /> Running...</> : <><IconPlayerPlay size={16} /> Run Backtest</>}
          </button>
          {result?.status === "COMPLETED" && (
            <button onClick={() => handleDownload(result.runId)}
              className="flex items-center gap-2 px-4 py-2 bg-pat-bg-surface-secondary border border-pat-border text-pat-text-primary rounded-md text-sm hover:border-pat-primary hover:text-pat-primary">
              <IconDownload size={16} /> Download CSV
            </button>
          )}
          <button onClick={() => setShowHistory(!showHistory)}
            className="flex items-center gap-2 px-4 py-2 bg-pat-bg-surface-secondary border border-pat-border text-pat-text-primary rounded-md text-sm hover:border-pat-primary hover:text-pat-primary">
            <IconHistory size={16} /> History ({runs.length})
          </button>
        </div>
        {error && <div className="mt-3 p-3 bg-red-500/10 border border-red-500/30 rounded-md text-sm text-red-500">{error}</div>}
      </div>

      {result && (
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <button onClick={() => setShowResults(!showResults)} className="w-full flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-pat-text-primary">Results: {result.runId}</h2>
            <IconChevronDown size={16} className={`text-pat-text-muted transition-transform ${showResults ? "rotate-180" : ""}`} />
          </button>
          {result.status === "FAILED" || result.error ? (
            <div className="p-3 bg-red-500/10 border border-red-500/30 rounded-md text-sm text-red-500">
              <span className="font-semibold">Backtest failed:</span> {result.error || "The run did not complete successfully."}
            </div>
          ) : showResults && result.metrics && (
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
              {[
                { label: "Balance", val: `$${fmt(result.metrics.finalBalance)}` },
                { label: "Return", val: `${fmt(result.metrics.totalReturn)}%`, pos: parseFloat(result.metrics.totalReturn) >= 0, neg: parseFloat(result.metrics.totalReturn) < 0 },
                { label: "Win Rate", val: `${fmt(result.metrics.winRate)}%` },
                { label: "Profit Factor", val: fmt(result.metrics.profitFactor) },
                { label: "Sharpe", val: fmt(result.metrics.sharpe) },
                { label: "Max DD", val: `${fmt(result.metrics.maxDD)}%`, neg: true },
                { label: "Trades", val: result.metrics.totalTrades },
              ].map(m => (
                <div key={m.label} className="bg-pat-bg-surface-secondary border border-pat-border rounded-md p-3">
                  <div className="text-xs text-pat-text-muted mb-1">{m.label}</div>
                  <div className={`text-sm font-semibold ${m.pos ? "text-pat-success" : m.neg ? "text-pat-danger" : "text-pat-text-primary"}`}>{m.val}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {showHistory && (
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Backtest History</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead><tr className="text-pat-text-muted border-b border-pat-border">
                <th className="text-left py-2 px-2">Run ID</th><th className="text-left py-2 px-2">Strategy</th>
                <th className="text-left py-2 px-2">Period</th><th className="text-right py-2 px-2">Return</th>
                <th className="text-right py-2 px-2">Win</th><th className="text-right py-2 px-2">PF</th>
                <th className="text-right py-2 px-2">Sharpe</th><th className="text-right py-2 px-2">MaxDD</th>
                <th className="text-right py-2 px-2">Trades</th><th className="text-center py-2 px-2">CSV</th>
              </tr></thead>
              <tbody>
                {runs.map(r => (
                  <tr key={r.run_id} className="border-b border-pat-border/50 hover:bg-pat-bg-surface-secondary">
                    <td className="py-2 px-2 text-pat-text-muted font-mono">{r.run_id}</td>
                    <td className="py-2 px-2 text-pat-text-primary">{r.strategy_id}</td>
                    <td className="py-2 px-2 text-pat-text-secondary">{r.start_date}→{r.end_date}</td>
                    <td className={`py-2 px-2 text-right font-medium ${parseFloat(r.total_return_pct) >= 0 ? "text-pat-success" : "text-pat-danger"}`}>{fmt(r.total_return_pct)}%</td>
                    <td className="py-2 px-2 text-right text-pat-text-secondary">{fmt(r.win_rate)}%</td>
                    <td className="py-2 px-2 text-right text-pat-text-secondary">{fmt(r.profit_factor)}</td>
                    <td className="py-2 px-2 text-right text-pat-text-secondary">{fmt(r.sharpe)}</td>
                    <td className="py-2 px-2 text-right text-pat-danger">{fmt(r.max_drawdown)}%</td>
                    <td className="py-2 px-2 text-right text-pat-text-secondary">{r.trades_count}</td>
                    <td className="py-2 px-2 text-center"><button onClick={() => handleDownload(r.run_id)} className="text-pat-primary hover:text-pat-primary-hover"><IconDownload size={14} /></button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
