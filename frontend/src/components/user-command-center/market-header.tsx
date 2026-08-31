"use client";
import { useEffect, useRef, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { strategyLabel } from "@/lib/strategy-labels";
import { getGlobalWs, type WsMessage, type MarketDataEvent, type FeedStatus } from "@/lib/websocket";
import { rafBatch } from "@/lib/performance";
import { useServerTime, formatBrokerTime, formatDrift } from "@/lib/use-server-time";

interface MarketState {
  Regime?: { Current?: string; Volatility?: string; Confidence?: number };
  Session?: { CurrentSession?: string; IsOverlap?: boolean; NewsRisk?: string };
  Indicators?: Record<string, string | number | boolean>;
  Bid?: string; Ask?: string; Spread?: string; CurrentPrice?: string;
  LastTick?: { Bid?: number; Ask?: number; Spread?: number; Source?: string };
}

export function MarketHeader() {
  const bidRef = useRef<HTMLSpanElement>(null);
  const askRef = useRef<HTMLSpanElement>(null);
  const spreadRef = useRef<HTMLSpanElement>(null);
  const [wsConnected, setWsConnected] = useState(false);
  const [wsLost, setWsLost] = useState(false);
  const [feedStatus, setFeedStatus] = useState<FeedStatus>("UNKNOWN");
  const [lastUpdate, setLastUpdate] = useState<number>(0);
  const [clockTick, setClockTick] = useState<number>(0);
  const [mounted, setMounted] = useState(false);
  const ws = getGlobalWs();
  const { driftMs, driftWarning, driftCritical, brokerOffset, brokerTimeMode } = useServerTime();

  // Update clock display every second
  useEffect(() => {
    const interval = setInterval(() => setClockTick(t => t + 1), 1000);
    return () => clearInterval(interval);
  }, []);

  const { data: marketState } = useQuery<MarketState>({
    queryKey: ["user-market-state"],
    queryFn: async () => (await customInstance.get("/market/state")).data,
    refetchInterval: 3000,
  });

  const { data: snapshot } = useQuery<{
    tick?: { bid: number; ask: number; spread: number; time: string };
    source?: string; session?: { name: string; is_overlap: boolean; is_weekend: boolean };
    indicators?: Record<string, number | string | boolean>;
  }>({
    queryKey: ["user-market-snapshot"],
    queryFn: async () => (await customInstance.get("/market/snapshot")).data,
    refetchInterval: 3000,
  });

  // Server-authoritative active (selected) strategy for this subscriber.
  // The engine is single-tenant for market state, so a user's "active strategy"
  // is their subscription's selected strategy(s), sourced from entitlements.
  const { data: entitlements } = useQuery<{ selected_strategies?: string[]; allowed_strategies?: string[] }>({
    queryKey: ["user-active-strategy-header"],
    queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data,
    refetchInterval: 15000,
  });
  const activeStrategies = (
    entitlements?.selected_strategies && entitlements.selected_strategies.length
      ? entitlements.selected_strategies
      : entitlements?.allowed_strategies ?? []
  ) as string[];
  const activeStrategyLabel = activeStrategies.length
    ? activeStrategies.map((s) => strategyLabel(s)).join(", ")
    : "—";

  useEffect(() => {
    ws.connect();
    const unsub = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "market") {
        const d = msg.payload as MarketDataEvent;
        rafBatch(() => {
          if (bidRef.current) bidRef.current.innerText = d.bid.toFixed(2);
          if (askRef.current) askRef.current.innerText = d.ask.toFixed(2);
          if (spreadRef.current) spreadRef.current.innerText = d.spread.toFixed(2);
          setLastUpdate(Date.now());
        });
      }
    });
    const unsubState = ws.subscribeState((s) => {
      setWsConnected(s === "CONNECTED");
      // prompt.md Section 59: never keep displaying stale values as live.
      setWsLost(s === "DISCONNECTED" || s === "RECONNECTING");
    });
    const unsubFeed = ws.subscribeFeedStatus((s) => setFeedStatus(s));
    return () => { unsub(); unsubState(); unsubFeed(); };
  }, [ws]);

  const bid = snapshot?.tick?.bid ?? Number(marketState?.LastTick?.Bid ?? marketState?.Bid ?? 0);
  const ask = snapshot?.tick?.ask ?? Number(marketState?.LastTick?.Ask ?? marketState?.Ask ?? 0);
  const spread = snapshot?.tick?.spread ?? Number(marketState?.LastTick?.Spread ?? marketState?.Spread ?? 0);
  const regime = marketState?.Regime?.Current ?? "—";
  const session = marketState?.Session?.CurrentSession ?? snapshot?.session?.name ?? "—";
  const atr = Number(marketState?.Indicators?.ATR ?? snapshot?.indicators?.atr ?? 0);
  const adx = Number(marketState?.Indicators?.ADX ?? snapshot?.indicators?.adx ?? 0);
  const rsi = Number(marketState?.Indicators?.RSI ?? snapshot?.indicators?.rsi ?? 0);
  const source = snapshot?.source ?? "—";
  const isOverlap = marketState?.Session?.IsOverlap ?? snapshot?.session?.is_overlap ?? false;
  const newsRisk = marketState?.Session?.NewsRisk ?? "NONE";
  const ema50 = Number(marketState?.Indicators?.EMA50 ?? 0);
  const ema200 = Number(marketState?.Indicators?.EMA200 ?? 0);
  const trendDir = ema50 > 0 && ema200 > 0 ? (ema50 > ema200 ? "BULLISH" : "BEARISH") : "—";

  const items = [
    { label: "BID", value: bid, refEl: bidRef, color: "text-pat-success", fixed: true },
    { label: "ASK", value: ask, refEl: askRef, color: "text-pat-danger", fixed: true },
    { label: "SPREAD", value: spread, refEl: spreadRef, color: spread > 0.5 ? "text-pat-warning" : "text-pat-text-primary", fixed: true },
  ];

  // clockTick forces re-render every second for the UTC clock display
  void clockTick;
  const metaItems = [
    { label: "Regime", value: regime, color: regime.includes("BULLISH") ? "text-pat-success" : regime.includes("BEARISH") ? "text-pat-danger" : "text-pat-text-secondary" },
    { label: "Session", value: session, color: "text-pat-text-secondary" },
    { label: "Trend", value: trendDir, color: trendDir === "BULLISH" ? "text-pat-success" : trendDir === "BEARISH" ? "text-pat-danger" : "text-pat-text-muted" },
    { label: "Active Strategy", value: activeStrategyLabel, color: activeStrategies.length ? "text-pat-success" : "text-pat-text-muted" },
    { label: "ATR", value: atr > 0 ? atr.toFixed(2) : "—", color: "text-pat-text-secondary" },
    { label: "ADX", value: adx > 0 ? adx.toFixed(1) : "—", color: adx > 25 ? "text-pat-success" : "text-pat-text-muted" },
    { label: "RSI", value: rsi > 0 ? rsi.toFixed(1) : "—", color: rsi > 70 ? "text-pat-danger" : rsi < 30 ? "text-pat-success" : "text-pat-text-secondary" },
  ];

  return (
    <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
      {/* Live connection lost banner (prompt.md Section 59) */}
      {wsLost && (
        <div role="alert" className="mb-3 flex items-center gap-2 rounded-md border border-pat-warning/30 bg-pat-warning/10 px-3 py-2 text-xs font-medium text-pat-warning">
          <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-pat-warning" />
          LIVE CONNECTION LOST — showing last known values; reconnecting…
        </div>
      )}
      {/* Price bar */}
      <div className="flex flex-wrap items-center gap-4 mb-3">
        <div className="flex items-center gap-2">
          <span className="text-sm font-bold text-pat-text-primary">XAUUSD</span>
          <span className={`text-[10px] px-1.5 py-0.5 rounded-full border ${
            source.includes("MT5") || source.includes("LOCAL")
              ? "bg-pat-success/10 text-pat-success border-pat-success/20"
              : "bg-pat-warning/10 text-pat-warning border-pat-warning/20"
          }`}>{source.includes("LOCAL") ? "LOCAL" : source.includes("MT5") ? "MT5" : source || "—"}</span>
        </div>
        {items.map((item) => (
          <div key={item.label} className="flex items-center gap-1.5">
            <span className="text-[10px] text-pat-text-muted uppercase">{item.label}</span>
            <span className={`text-lg font-mono font-bold ${item.color} tabular-nums`}>
              {item.fixed && item.refEl ? <span ref={item.refEl}>{item.value > 0 ? item.value.toFixed(2) : "—"}</span> : item.value > 0 ? item.value.toFixed(2) : "—"}
            </span>
          </div>
        ))}
        <div className="ml-auto flex items-center gap-2">
          {/* Engine-authoritative clock — Broker TF (collected live from Agents), not UTC */}
          <span className="text-[10px] font-mono tabular-nums text-pat-text-secondary" title={brokerTimeMode === "BROKER_ALIGNED" ? "Engine time aligned to broker session timezone" : "Engine time UTC-aligned"}>
            {/* Clock is Date.now()-derived; render a stable placeholder until
                mounted to avoid a server/client text mismatch (#418 hydration). */}
            {mounted ? formatBrokerTime(driftMs, brokerOffset) : "—"}
          </span>
          {driftCritical ? (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-pat-danger/10 text-pat-danger border border-pat-danger/20 font-semibold" title="Clock drift > 2min — check NTP sync on all machines">
              ⚠ CLOCK DRIFT {formatDrift(driftMs)}
            </span>
          ) : driftWarning ? (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-pat-warning/10 text-pat-warning border border-pat-warning/20" title="Clock drift > 30s — minor skew detected">
              ⏰ {formatDrift(driftMs)}
            </span>
          ) : null}
          <span className={`inline-block h-2 w-2 rounded-full ${
            feedStatus === "LIVE" ? "bg-pat-success animate-pulse"
            : feedStatus === "DEGRADED" ? "bg-pat-warning animate-pulse"
            : feedStatus === "STALE" ? "bg-pat-danger"
            : feedStatus === "REPLAY" ? "bg-pat-text-muted"
            : wsConnected ? "bg-pat-success animate-pulse" : "bg-pat-warning"
          }`} />
          <span className="text-[10px] text-pat-text-muted">{
            feedStatus !== "UNKNOWN" ? feedStatus
            : wsConnected ? "LIVE" : lastUpdate > 0 ? "REST 3s" : "CONNECTING"
          }</span>
          {newsRisk !== "NONE" && (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-pat-danger/10 text-pat-danger border border-pat-danger/20">{newsRisk}</span>
          )}
          {isOverlap && (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-pat-success/10 text-pat-success border border-pat-success/20">LONDON/NY OVERLAP</span>
          )}
        </div>
      </div>
      {/* Meta strip */}
      <div className="flex flex-wrap items-center gap-4 border-t border-pat-border/50 pt-2">
        {metaItems.map((item) => (
          <div key={item.label} className="flex items-center gap-1">
            <span className="text-[10px] text-pat-text-muted uppercase">{item.label}</span>
            <span className={`text-xs font-medium ${item.color}`}>{item.value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
