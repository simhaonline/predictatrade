// Typed bindings for the pat-engine REST API (cmd/gateway /api/v1).
// The frontend only renders server-authoritative truth; it never recomputes
// indicators, probability, risk or execution eligibility.

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";

async function getJSON<T>(path: string): Promise<T | null> {
  try {
    const res = await fetch(`${API_BASE}${path}`, { cache: "no-store" });
    if (!res.ok) return null;
    return (await res.json()) as T;
  } catch {
    return null;
  }
}

export type BrokerProfile = {
  symbol: string;
  digits: number;
  tick_size: number;
  contract_size: number;
  commission_per_lot: number;
  typical_spread: number;
  leverage: number;
  swap_long: number;
  swap_short: number;
  timezone_offset: number;
  sessions: { name: string; start_h: number; end_h: number; overlap: boolean }[];
};

export type RiskConfig = {
  Equity: number;
  FreeMargin: number;
  RiskPerTradePct: number;
  MaxDailyLossPct: number;
  MaxPositions: number;
  MaxLeverage: number;
  MinRR: number;
};

export type SessionInfo = {
  session: string;
  overlap: boolean;
  tz_offset: number;
};

export type SignalRecord = {
  id: string;
  ts: string;
  strategy_id: string;
  direction: string;
  entry: number;
  sl: number;
  tp1: number;
  raw_score: number;
  grade: string;
  status: string;
};

export type BarRecord = {
  time: string;
  open: number;
  high: number;
  low: number;
  close: number;
  spread: number;
};

export const api = {
  broker: () => getJSON<BrokerProfile>("/api/v1/broker"),
  risk: () => getJSON<RiskConfig>("/api/v1/risk"),
  session: () => getJSON<SessionInfo>("/api/v1/session"),
  strategies: () => getJSON<{ strategies: string[]; license_plan: string }>("/api/v1/strategies"),
  signals: (n = 25) => getJSON<SignalRecord[]>(`/api/v1/signals?limit=${n}`),
  bars: (n = 50) => getJSON<BarRecord[]>(`/api/v1/bars?limit=${n}`),
};
