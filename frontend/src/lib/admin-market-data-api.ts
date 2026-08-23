import { customInstance } from "@/lib/axios-instance";
import { fetchMarketSnapshot } from "@/lib/admin-api";

export interface MarketSnapshotIndicators {
  symbol?: string;
  source?: string;
  broker?: string;
  node?: string;
  timestamp?: string;
  tick?: { bid: number; ask: number; spread: number; volume: number; time: string };
  indicators?: Record<string, number | string | boolean>;
  vwap?: { session_vwap: number; upper_band: number; lower_band: number };
  session?: { name: string; is_overlap: boolean; is_weekend: boolean };
}

export async function fetchLiveMarketSnapshot(): Promise<MarketSnapshotIndicators> {
  return fetchMarketSnapshot() as Promise<MarketSnapshotIndicators>;
}
