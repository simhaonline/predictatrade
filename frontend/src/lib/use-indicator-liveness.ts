/**
 * Indicator liveness tracking hook.
 *
 * Consumes the existing /market/snapshot REST endpoint and the existing
 * WebSocket market events to track when each indicator was last updated.
 * Does NOT recompute indicators — only observes update timestamps.
 */
"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { getGlobalWs, type WsMessage } from "@/lib/websocket";

// ─── Types ───────────────────────────────────────────────────────────────────

export interface MarketSnapshot {
  type?: string;
  symbol?: string;
  timestamp?: string;
  source?: string;
  indicators?: Record<string, number | string | boolean>;
  tick?: { bid: number; ask: number; spread: number; time: string };
  session?: { name: string; is_overlap: boolean; is_weekend: boolean };
}

export type LivenessStatus = "live" | "late" | "stale" | "disabled";
export type ActiveStatus = "active" | "armed" | "reactive" | "inactive";

export interface IndicatorLiveness {
  key: string;
  label: string;
  group: string;
  updateType: "tick" | "bar" | "session";
  lastUpdated: number | null; // epoch ms
  currentValue: number | string | boolean | null;
  status: LivenessStatus;
  activeStatus: ActiveStatus;
  expectedIntervalMs: number;
}

export interface PerformanceMetric {
  indicatorKey: string;
  strategy: string;
  hitRate: number | null;
  avgRMultiple: number | null;
  contributionScore: number | null;
  signalFrequency: number | null;
  signalAccuracy: number | null;
  performanceLevel: "excellent" | "good" | "neutral" | "poor" | "no-data";
  tradeCount: number;
}

// ─── Indicator definitions (matches Go engine IndicatorFeatures) ─────────────

interface IndicatorDef {
  key: string;
  label: string;
  group: string;
  updateType: "tick" | "bar" | "session";
  expectedIntervalMs: number;
}

const INDICATOR_DEFS: IndicatorDef[] = [
  // Trend
  { key: "ema9", label: "EMA 9", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ema21", label: "EMA 21", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ema50", label: "EMA 50", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ema100", label: "EMA 100", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ema200", label: "EMA 200", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ema_cross_9_21", label: "EMA Cross 9/21", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "sma50", label: "SMA 50", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "sma100", label: "SMA 100", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "sma200", label: "SMA 200", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "macd_main", label: "MACD 12/26/9", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "macd_signal", label: "MACD Signal", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "macd_histogram", label: "MACD Histogram", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "macd_bull_cross", label: "MACD Bull Cross", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "macd_bear_cross", label: "MACD Bear Cross", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "adx", label: "ADX 14", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "adx_plus_di", label: "+DI 14", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "adx_minus_di", label: "-DI 14", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "psar", label: "Parabolic SAR", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "psar_long", label: "SAR Direction", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ichimoku_tenkan", label: "Ichimoku Tenkan", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ichimoku_kijun", label: "Ichimoku Kijun", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ichimoku_senkou_a", label: "Ichimoku Senkou A", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "ichimoku_senkou_b", label: "Ichimoku Senkou B", group: "Trend", updateType: "bar", expectedIntervalMs: 300000 },
  // Momentum
  { key: "rsi", label: "RSI 14", group: "Momentum", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "stoch_main", label: "Stochastic 14/3/3", group: "Momentum", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "stoch_signal", label: "Stochastic Signal", group: "Momentum", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "stoch_rsi", label: "StochRSI", group: "Momentum", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "stoch_rsi_k", label: "StochRSI K", group: "Momentum", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "stoch_rsi_d", label: "StochRSI D", group: "Momentum", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "cci", label: "CCI 20", group: "Momentum", updateType: "bar", expectedIntervalMs: 300000 },
  // Volatility
  { key: "atr", label: "ATR 14", group: "Volatility", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "boll_upper", label: "Bollinger Upper", group: "Volatility", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "boll_middle", label: "Bollinger Middle", group: "Volatility", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "boll_lower", label: "Bollinger Lower", group: "Volatility", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "boll_width", label: "Bollinger Width", group: "Volatility", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "boll_bull_rev", label: "BB Bull Reversal", group: "Volatility", updateType: "bar", expectedIntervalMs: 300000 },
  { key: "boll_bear_rev", label: "BB Bear Reversal", group: "Volatility", updateType: "bar", expectedIntervalMs: 300000 },
  // Volume
  { key: "obv", label: "OBV", group: "Volume", updateType: "tick", expectedIntervalMs: 1000 },
  { key: "vwap", label: "VWAP", group: "Volume", updateType: "tick", expectedIntervalMs: 1000 },
  // Session / MTF / Structure
  { key: "session", label: "Session / MTF / Structure", group: "Context", updateType: "session", expectedIntervalMs: 300000 },
];

const STRATEGIES = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB", "ATEN"];

// ─── Hook ────────────────────────────────────────────────────────────────────

export function useIndicatorLiveness(refreshMs: number = 500) {
  const [liveness, setLiveness] = useState<IndicatorLiveness[]>(
    INDICATOR_DEFS.map((d) => ({
      key: d.key,
      label: d.label,
      group: d.group,
      updateType: d.updateType,
      lastUpdated: null,
      currentValue: null,
      status: "disabled" as LivenessStatus,
      activeStatus: "inactive" as ActiveStatus,
      expectedIntervalMs: d.expectedIntervalMs,
    }))
  );
  const [history, setHistory] = useState<Map<string, { time: number; value: number }[]>>(
    new Map(INDICATOR_DEFS.map((d) => [d.key, []]))
  );
  const updateTimesRef = useRef<Map<string, number>>(new Map());
  const ws = getGlobalWs();

  // REST polling for snapshot (reuse existing endpoint)
  const { data: snapshot } = useQuery<MarketSnapshot>({
    queryKey: ["engine-market-snapshot-monitor"],
    queryFn: async () => {
      const res = await customInstance.get("/market/snapshot");
      return res.data as MarketSnapshot;
    },
    refetchInterval: 2000, // poll every 2s for REST fallback
    staleTime: 1000,
  });

  // Process snapshot updates
  const processSnapshot = useCallback((snap: MarketSnapshot) => {
    const now = Date.now();
    const indicators = snap.indicators || {};
    const newTimes = new Map(updateTimesRef.current);

    setLiveness((prev) =>
      prev.map((ind) => {
        const val = indicators[ind.key];
        // Key IS present in the API response — mark as live regardless of value.
        // Zero/false values mean "building candle history", not "disabled".
        // Only keys completely missing from the response are "disabled".
        if (val !== undefined && val !== null) {
          newTimes.set(ind.key, now);
          // Track history for charts (only non-zero numeric values)
          if (typeof val === "number" && val !== 0) {
            setHistory((h) => {
              const arr = h.get(ind.key) || [];
              const updated = [...arr, { time: now, value: val }].slice(-200);
              return new Map(h).set(ind.key, updated);
            });
          }
          return {
            ...ind,
            currentValue: val,
            lastUpdated: now,
            status: "live" as LivenessStatus,
            activeStatus: val === 0 || val === false ? "reactive" : determineActiveStatus(ind.key, val),
          };
        }
        // No update — compute staleness
        const lastTime = newTimes.get(ind.key);
        if (lastTime) {
          const elapsed = now - lastTime;
          const expected = ind.expectedIntervalMs;
          if (elapsed > expected * 2) {
            return { ...ind, status: "stale" as LivenessStatus };
          } else if (elapsed > expected) {
            return { ...ind, status: "late" as LivenessStatus };
          }
        }
        return ind;
      })
    );
    updateTimesRef.current = newTimes;
  }, []);

  // Process REST snapshots
  useEffect(() => {
    if (snapshot) {
      processSnapshot(snapshot);
    }
  }, [snapshot, processSnapshot]);

  // Subscribe to WebSocket for real-time updates
  useEffect(() => {
    ws.connect();
    const unsub = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "market") {
        // Market tick — update tick-based indicators (e.g. OBV)
        const now = Date.now();
        setLiveness((prev) =>
          prev.map((ind) => {
            if (ind.updateType === "tick") {
              updateTimesRef.current.set(ind.key, now);
              return { ...ind, lastUpdated: now, status: "live" as LivenessStatus };
            }
            return ind;
          })
        );
      }
    });
    return () => { unsub(); };
  }, [ws]);

  // Periodic staleness check
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now();
      setLiveness((prev) =>
        prev.map((ind) => {
          if (ind.status === "disabled") return ind;
          const lastTime = updateTimesRef.current.get(ind.key);
          if (!lastTime) return ind;
          const elapsed = now - lastTime;
          const expected = ind.expectedIntervalMs;
          if (elapsed > expected * 2) {
            return { ...ind, status: "stale" as LivenessStatus };
          } else if (elapsed > expected) {
            return { ...ind, status: "late" as LivenessStatus };
          }
          return { ...ind, status: "live" as LivenessStatus };
        })
      );
    }, refreshMs);
    return () => clearInterval(interval);
  }, [refreshMs]);

  return { liveness, history, strategies: STRATEGIES, snapshot };
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function determineActiveStatus(key: string, value: number | string | boolean): ActiveStatus {
  // Heuristic: indicators with non-zero/non-neutral values are "reactive"
  // Indicators in trigger states are "armed"
  // This is an observability layer — it does NOT affect trading logic.
  if (typeof value === "boolean") {
    return value ? "armed" : "reactive";
  }
  if (typeof value === "number") {
    // RSI in oversold/overbought → armed
    if (key === "rsi" && (value < 30 || value > 70)) return "armed";
    // ADX above 25 → armed (trending)
    if (key === "adx" && value > 25) return "armed";
    // MACD above 0 → armed
    if (key === "macd_main" && value > 0) return "armed";
    // CCI extreme → armed
    if (key === "cci" && (value < -100 || value > 100)) return "armed";
    // StochRSI extreme → armed
    if (key === "stoch_rsi" && (value < 20 || value > 80)) return "armed";
    return "reactive";
  }
  return "reactive";
}

export function getLivenessColor(status: LivenessStatus): string {
  switch (status) {
    case "live": return "bg-pat-success";
    case "late": return "bg-pat-warning";
    case "stale": return "bg-pat-danger";
    case "disabled": return "bg-pat-text-muted";
  }
}

export function getActiveColor(status: ActiveStatus): string {
  switch (status) {
    case "active": return "bg-pat-success";
    case "armed": return "bg-pat-warning";
    case "reactive": return "bg-pat-info";
    case "inactive": return "bg-pat-text-muted";
  }
}

export function getPerformanceColor(level: PerformanceMetric["performanceLevel"]): string {
  switch (level) {
    case "excellent": return "text-pat-success";
    case "good": return "text-pat-success";
    case "neutral": return "text-pat-text-secondary";
    case "poor": return "text-pat-danger";
    case "no-data": return "text-pat-text-muted";
  }
}
