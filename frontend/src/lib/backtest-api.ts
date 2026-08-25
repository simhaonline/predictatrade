"use client";

import { getAccessToken } from "@/lib/auth";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "/api/v1";

function authHeaders(): Record<string, string> {
  const token = getAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// All fetch calls use credentials: 'include' to send cookies cross-origin
const fetchOpts = (extra?: RequestInit): RequestInit => ({
  credentials: "include",
  ...extra,
  headers: { ...authHeaders(), ...(extra?.headers || {}) },
});

export interface DataSummary {
  timeframe: string;
  candle_count: string;
  min_date: string;
  max_date: string;
  source: string;
}

export interface BacktestRun {
  run_id: string;
  strategy_id: string;
  strategy_mode: string;
  status: string;
  start_date: string;
  end_date: string;
  initial_balance: string;
  final_balance: string;
  total_return_pct: string;
  win_rate: string;
  profit_factor: string;
  sharpe: string;
  max_drawdown: string;
  trades_count: number;
  bars_processed: number;
  duration_seconds: string;
  created_at: string;
}

export interface BacktestTrade {
  direction: string;
  strategy_id: string;
  entry_price: string;
  exit_price: string;
  exit_reason: string;
  pnl: string;
  pnl_r: string;
  entry_time: string;
  exit_time: string;
  holding_bars: number;
}

export interface RunBacktestRequest {
  strategy: string;
  timeframe: string;
  startDate: string;
  endDate: string;
  initialBalance?: number;
}

export interface RunBacktestResponse {
  runId: string;
  status: string;
  strategy: string;
  timeframe: string;
  metrics: {
    finalBalance: string;
    totalReturn: string;
    winRate: string;
    profitFactor: string;
    sharpe: string;
    maxDD: string;
    totalTrades: string;
  };
  rawOutput?: string;
  error?: string;
}

export async function fetchAvailableData(): Promise<DataSummary[]> {
  const res = await fetch(`${API_BASE}/backtest/data`, fetchOpts());
  if (!res.ok) throw new Error("Failed to fetch data");
  return res.json();
}

export async function fetchRuns(): Promise<BacktestRun[]> {
  const res = await fetch(`${API_BASE}/backtest/runs`, fetchOpts());
  if (!res.ok) throw new Error("Failed to fetch runs");
  return res.json();
}

export async function fetchRunDetails(runId: string): Promise<{ run: BacktestRun; trades: BacktestTrade[] }> {
  const res = await fetch(`${API_BASE}/backtest/runs/${runId}`, fetchOpts());
  if (!res.ok) throw new Error("Failed to fetch run details");
  return res.json();
}

export async function runBacktest(req: RunBacktestRequest): Promise<RunBacktestResponse> {
  const res = await fetch(`${API_BASE}/backtest/run`, fetchOpts({
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  }));
  if (!res.ok) throw new Error("Failed to run backtest");
  return res.json();
}

export function downloadCSVUrl(runId: string): string {
  const token = getAccessToken();
  return `${API_BASE}/backtest/runs/${runId}/download?format=csv&token=${token || ""}`;
}

// F3 fix: download the CSV via fetch + blob so the JWT-bearing URL never
// lands in browser history, the address bar, or a Referer header. The token
// is still sent (backend reads ?token=), but only inside the network request.
export async function downloadCSV(runId: string): Promise<void> {
  const url = downloadCSVUrl(runId);
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error("Failed to download CSV");
  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition");
  let filename = `backtest-${runId}.csv`;
  if (disposition) {
    const match = /filename="?([^";]+)"?/.exec(disposition);
    if (match) filename = match[1];
  }
  const objectUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = objectUrl;
  a.download = filename;
  a.rel = "noopener noreferrer";
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(objectUrl);
}
