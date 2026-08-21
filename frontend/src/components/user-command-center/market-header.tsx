"use client";
import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { getGlobalWs, type WsMessage, type MarketDataEvent } from "@/lib/websocket";
import { rafBatch } from "@/lib/performance";

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
  const [lastUpdate, setLastUpdate] = useState<number>(0);
  const ws = getGlobalWs();

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
    const unsubState = ws.subscribeState((s) => setWsConnected(s === "CONNECTED"));
    return () => { unsub(); unsubState(); };
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

  const metaItems = [
    { label: "Regime", value: regime, color: regime.includes("BULLISH") ? "text-pat-success" : regime.includes("BEARISH") ? "text-pat-danger" : "text-pat-text-secondary" },
    { label: "Session", value: session, color: "text-pat-text-secondary" },
    { label: "Trend", value: trendDir, color: trendDir === "BULLISH" ? "text-pat-success" : trendDir === "BEARISH" ? "text-pat-danger" : "text-pat-text-muted" },
    { label: "ATR", value: atr > 0 ? atr.toFixed(2) : "—", color: "text-pat-text-secondary" },
    { label: "ADX", value: adx > 0 ? adx.toFixed(1) : "—", color: adx > 25 ? "text-pat-success" : "text-pat-text-muted" },
    { label: "RSI", value: rsi > 0 ? rsi.toFixed(1) : "—", color: rsi > 70 ? "text-pat-danger" : rsi < 30 ? "text-pat-success" : "text-pat-text-secondary" },
  ];

  return (
    <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
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
          <span className={`inline-block h-2 w-2 rounded-full ${wsConnected ? "bg-pat-success animate-pulse" : "bg-pat-warning"}`} />
          <span className="text-[10px] text-pat-text-muted">{wsConnected ? "LIVE" : lastUpdate > 0 ? "REST 3s" : "CONNECTING"}</span>
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
