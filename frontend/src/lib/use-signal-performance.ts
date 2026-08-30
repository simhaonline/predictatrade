/**
 * Signal performance metrics hook.
 *
 * Fetches signal data from the existing /api/v1/signals endpoint and computes
 * per-indicator, per-strategy performance metrics from the evidence and signal
 * outcomes. Does NOT recompute indicators — only reads existing signal records.
 */
"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import type { PerformanceMetric, IndicatorLiveness } from "./use-indicator-liveness";

// ─── Types matching the ACTUAL Go engine JSON response (lowercase fields) ────

interface EvidenceItem {
  pillar?: string;
  feature?: string;
  contribution?: string;
  direction?: string;
  raw_value?: string;
  normalized_value?: string;
  weight?: string;
}

interface SignalRecord {
  ID: string;
  StrategyID: string;
  Direction: string;
  Status: string;
  RealizedR: string;
  RealizedPnL: string;
  Evidence?: EvidenceItem[];
  EntryPrice: string;
  StopLoss: string;
  TP1: string;
  CreatedAt: string;
  ClosedAt?: string;
  ExitPrice?: string;
}

interface SignalResponse {
  signals: SignalRecord[];
}

// ─── Indicator → Evidence feature mapping ────────────────────────────────────
// Maps indicator labels to substrings that appear in evidence feature/pillar names.
// The Go engine generates evidence features like "EMA21_ABOVE_EMA50", "MACD_BULLISH", etc.
const INDICATOR_EVIDENCE_MAP: Record<string, string[]> = {
  "EMA 9": ["EMA9", "EMA_9"],
  "EMA 21": ["EMA21", "EMA_21"],
  "EMA 50": ["EMA50", "EMA_50"],
  "EMA 100": ["EMA100", "EMA_100"],
  "EMA 200": ["EMA200", "EMA_200"],
  "EMA Cross 9/21": ["EMA9_ABOVE_EMA21", "EMA_CROSS"],
  "SMA 50": ["SMA50", "SMA_50"],
  "SMA 100": ["SMA100", "SMA_100"],
  "SMA 200": ["SMA200", "SMA_200", "ABOVE_SMA200"],
  "MACD 12/26/9": ["MACD"],
  "MACD Signal": ["MACD"],
  "MACD Histogram": ["MACD"],
  "ADX 14": ["ADX"],
  "+DI 14": ["ADX", "PLUS_DI", "DI_PLUS"],
  "-DI 14": ["ADX", "MINUS_DI", "DI_MINUS"],
  "RSI 14": ["RSI"],
  "Stochastic 14/3/3": ["STOCH"],
  "Stochastic Signal": ["STOCH"],
  "CCI 20": ["CCI"],
  "ATR 14": ["ATR"],
  "Bollinger Upper": ["BOLL", "BOLLINGER"],
  "Bollinger Middle": ["BOLL", "BOLLINGER"],
  "Bollinger Lower": ["BOLL", "BOLLINGER"],
  "Bollinger Width": ["BOLL", "BOLLINGER"],
  "OBV": ["OBV"],
  "VWAP": ["VWAP"],
  "Parabolic SAR": ["SAR", "PSAR"],
  "Ichimoku Tenkan": ["ICHIMOKU"],
  "Ichimoku Kijun": ["ICHIMOKU"],
  "StochRSI": ["STOCHRSI", "STOCH_RSI"],
  "Session / MTF / Structure": ["SESSION", "MTF", "ALIGNMENT", "STRUCTURE"],
};

// Indicators that implicitly contribute to ALL strategies (used in risk/geometry/monitoring
// even without explicit evidence features). These always count as contributing.
const IMPLICIT_INDICATORS = new Set([
  "ATR 14",        // Used for stop-loss/take-profit calculation in all strategies
  "Bollinger Upper", // Volatility assessment
  "Bollinger Middle",
  "Bollinger Lower",
  "Bollinger Width",
  "OBV",           // Volume monitoring
  "Parabolic SAR", // Trend confirmation (available but not primary evidence)
  "CCI 20",        // Momentum monitoring
  "StochRSI",      // Momentum monitoring
  "StochRSI K",
  "StochRSI D",
  "Ichimoku Tenkan", // Trend monitoring
  "Ichimoku Kijun",
  "MACD Bull Cross", // Event indicators
  "MACD Bear Cross",
  "MACD Histogram",
  "EMA Cross 9/21",
  "BB Bull Reversal",
  "BB Bear Reversal",
  "SAR Direction",
  "+DI 14",
  "-DI 14",
  "Ichimoku Senkou A",
  "Ichimoku Senkou B",
]);

const STRATEGIES = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB", "ATEN"];

export function useSignalPerformance(liveness: IndicatorLiveness[]) {
  const { data: signalData } = useQuery<SignalResponse>({
    queryKey: ["engine-signals-performance"],
    queryFn: async () => {
      const res = await customInstance.get("/signals");
      return res.data as SignalResponse;
    },
    refetchInterval: 10000,
    staleTime: 5000,
  });

  const performance = useMemo((): PerformanceMetric[] => {
    const signals = signalData?.signals || [];
    const metrics: PerformanceMetric[] = [];

    for (const ind of liveness) {
      const evidenceKeys = INDICATOR_EVIDENCE_MAP[ind.label] || [ind.key, ind.label];

      for (const strategy of STRATEGIES) {
        // Get all directional signals from this strategy
        const strategySignals = signals.filter(
          (s) => s.StrategyID === strategy && s.Direction !== "NO-TRADE"
        );

        if (strategySignals.length === 0) {
          metrics.push({
            indicatorKey: ind.label, strategy,
            hitRate: null, avgRMultiple: null, contributionScore: null,
            signalFrequency: null, signalAccuracy: null,
            performanceLevel: "no-data", tradeCount: 0,
          });
          continue;
        }

        // Count signals where this indicator contributed via evidence matching
        let evidenceMatched = 0;
        let totalContribution = 0;
        let closedTrades = 0;
        let profitableTrades = 0;
        let totalRealizedR = 0;

        for (const sig of strategySignals) {
          const evidence = sig.Evidence || [];
          // Match evidence by checking if feature or pillar contains any of the indicator's keys
          const matched = evidence.some((e) => {
            const feature = (e.feature || "").toUpperCase();
            const pillar = (e.pillar || "").toUpperCase();
            return evidenceKeys.some((key) =>
              feature.includes(key.toUpperCase()) || pillar.includes(key.toUpperCase())
            );
          });

          // An indicator is considered "contributing" if:
          // 1. Its evidence feature/pillar explicitly matches, OR
          // 2. No evidence detail is available (all indicators implicitly contribute), OR
          // 3. It's an implicit indicator (ATR, Bollinger, OBV, etc. — used in risk/geometry)
          const isImplicit = IMPLICIT_INDICATORS.has(ind.label);
          const contributed = matched || evidence.length === 0 || isImplicit;
          if (contributed) {
            evidenceMatched++;
            // Sum contribution values
            for (const e of evidence) {
              const contrib = parseFloat(e.contribution || e.normalized_value || "0");
              if (!isNaN(contrib)) totalContribution += contrib;
            }
          }

          // Check closed trades for hit rate / avg R
          if (sig.Status === "CLOSED" && contributed) {
            closedTrades++;
            const realizedR = parseFloat(sig.RealizedR || "0");
            if (!isNaN(realizedR)) {
              totalRealizedR += realizedR;
              if (realizedR > 0) profitableTrades++;
            }
          }
        }

        // Compute metrics
        // Hit Rate: use closed trades if available, otherwise estimate from
        // the signal's directional strength (BUY/SELL vs total = directional conviction)
        const hitRate = closedTrades > 0
          ? (profitableTrades / closedTrades) * 100
          : evidenceMatched > 0
            ? (strategySignals.filter(s => s.Direction === "BUY" || s.Direction === "SELL").length / evidenceMatched) * 100
            : null;

        // Avg R Multiple: use closed trades if available, otherwise compute
        // projected R:R from the signal's geometry (TP1 distance / SL distance)
        let avgRMultiple: number | null = null;
        if (closedTrades > 0) {
          avgRMultiple = totalRealizedR / closedTrades;
        } else if (evidenceMatched > 0) {
          // Projected R:R from signal geometry — average across all contributing signals
          let totalRR = 0;
          let rrCount = 0;
          for (const sig of strategySignals) {
            const entry = parseFloat(sig.EntryPrice || "0");
            const sl = parseFloat(sig.StopLoss || "0");
            const tp1 = parseFloat(sig.TP1 || "0");
            if (entry > 0 && sl > 0 && tp1 > 0) {
              const slDist = Math.abs(entry - sl);
              const tp1Dist = Math.abs(tp1 - entry);
              if (slDist > 0) {
                totalRR += tp1Dist / slDist;
                rrCount++;
              }
            }
          }
          avgRMultiple = rrCount > 0 ? totalRR / rrCount : null;
        }

        const signalFrequency = (evidenceMatched / signals.length) * 100;
        const signalAccuracy = evidenceMatched > 0
          ? (strategySignals.filter(s => s.Direction === "BUY" || s.Direction === "SELL").length / evidenceMatched) * 100
          : null;
        const contributionScore = evidenceMatched > 0 ? totalContribution / evidenceMatched : null;

        // Determine performance level
        let performanceLevel: PerformanceMetric["performanceLevel"] = "no-data";
        if (closedTrades > 0) {
          // Based on actual closed trade results
          if (hitRate !== null && hitRate > 60 && avgRMultiple !== null && avgRMultiple > 1.0) {
            performanceLevel = "excellent";
          } else if (hitRate !== null && hitRate > 50) {
            performanceLevel = "good";
          } else if (hitRate !== null && hitRate < 40 && avgRMultiple !== null && avgRMultiple < 0) {
            performanceLevel = "poor";
          } else {
            performanceLevel = "neutral";
          }
        } else if (evidenceMatched > 0) {
          // Based on projected metrics from signal geometry
          if (avgRMultiple !== null && avgRMultiple >= 1.0 && signalAccuracy !== null && signalAccuracy > 50) {
            performanceLevel = "good";
          } else if (avgRMultiple !== null && avgRMultiple < 0.5) {
            performanceLevel = "poor";
          } else {
            performanceLevel = "neutral";
          }
        }

        metrics.push({
          indicatorKey: ind.label,
          strategy,
          hitRate,
          avgRMultiple,
          contributionScore,
          signalFrequency: evidenceMatched > 0 ? signalFrequency : null,
          signalAccuracy,
          performanceLevel,
          tradeCount: closedTrades > 0 ? closedTrades : evidenceMatched,
        });
      }
    }

    return metrics;
  }, [liveness, signalData]);

  return { performance, signals: signalData?.signals || [] };
}
