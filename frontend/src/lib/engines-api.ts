import { customInstance } from "@/lib/axios-instance";

export interface EngineSnapshot {
  engine: string;
  enabled: boolean;
  running: boolean;
  health: "LIVE" | "WAITING" | "STALE" | "ERROR";
  primary_timeframes: string[];
  last_market_event: string;
  last_market_timeframe: string;
  last_evaluation: string;
  last_candidate: string;
  last_signal_at: string;
  last_signal_reference: string;
  current_decision: string;
  current_score: number;
  confidence: number;
  calibrated_probability: number;
  has_calibrated_probability: boolean;
  data_quality: string;
  regime: string;
  evaluation_count: number;
  candidate_count: number;
  signal_count: number;
  no_trade_count: number;
  error_count: number;

  // prompt.md Sections 17-18: Rejection diagnostics
  rejection_counts?: Record<string, number>;
  last_rejection_reason?: string;
  candidates_today?: number;
  qualified_today?: number;
  rejection_rate?: number;

  // prompt.md Section 14: Expectancy
  expectancy_score?: number;

  // prompt.md Section 32: Config version
  config_version?: string;
}

export interface EnginesStatusResponse {
  engines: EngineSnapshot[];
  server_time: string;
}

/** Backend-authoritative per-engine liveness (Go realtime plane). */
export async function fetchEnginesStatus(): Promise<EnginesStatusResponse> {
  const res = await customInstance.get<EnginesStatusResponse>("/engines/status");
  return res.data ?? { engines: [], server_time: new Date().toISOString() };
}
