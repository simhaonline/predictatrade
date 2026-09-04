"use client";
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { strategyLabel } from "@/lib/strategy-labels";
import { getGlobalWs, type WsMessage } from "@/lib/websocket";
import { useServerTime, formatBrokerTimestamp } from "@/lib/use-server-time";

interface EngineSignal {
  ID: string; StrategyID: string; Direction: string; Status: string;
  RawScore: string; CalibratedProbability: string;
  EntryPrice: string; StopLoss: string; TP1: string; TP2: string; TP3: string;
  GrossRRTP1: string; GrossRRTP2: string; GrossRRTP3: string;
  Evidence?: Array<{ pillar?: string; feature?: string; contribution?: string; direction?: string }>;
  ReasonCodes?: string[]; CreatedAt: string; ExpiresAt?: string;
  Regime?: string; Session?: string; Executable?: boolean;
  SignalClass?: string; // ADVISORY | EXECUTABLE — server-authoritative classification
}

export function SignalPipeline() {
  const ws = getGlobalWs();
  const { brokerOffset } = useServerTime();

  const { data: engineData, refetch } = useQuery<{ signals: EngineSignal[] }>({
    queryKey: ["user-signal-pipeline"],
    queryFn: async () => (await customInstance.get("/signals")).data,
    refetchInterval: 10000,
  });

  // WS is used ONLY to prompt a refresh of the canonical REST list.
  // Raw WS payloads are never rendered, so real signal fields are never
  // overwritten by placeholder values.
  useEffect(() => {
    ws.connect();
    let pending: ReturnType<typeof setTimeout> | null = null;
    const unsub = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "signal") {
        if (pending) clearTimeout(pending);
        pending = setTimeout(() => { refetch(); }, 800);
      }
    });
    return () => { if (pending) clearTimeout(pending); unsub(); };
  }, [ws, refetch]);

  // REST is the single source of truth — always fresh, never frozen.
  const restSignals = (engineData?.signals ?? []).filter(s => s.Direction !== "NO-TRADE").slice(0, 10);
  // Entitlement enforcement is authoritative server-side (realtime /signals
  // already filters by the user's allowed strategies via the gate registry).
  // The client must NOT re-filter by subscription.selected_strategies, which
  // previously hid entitled strategies (e.g. only ULTRA_SCALPING showed).
  const displaySignals = restSignals;

  const dirColor = (dir: string): string => {
    if (dir === "BUY") return "text-pat-success";
    if (dir === "SELL") return "text-pat-danger";
    if (dir.includes("BUY_CANDIDATE")) return "text-pat-warning";
    if (dir.includes("SELL_CANDIDATE")) return "text-pat-candidate-sell";
    return "text-pat-text-muted";
  };

  const dirBg = (dir: string): string => {
    if (dir === "BUY") return "bg-pat-success/10 border-pat-success/20";
    if (dir === "SELL") return "bg-pat-danger/10 border-pat-danger/20";
    if (dir.includes("CANDIDATE")) return "bg-pat-warning/10 border-pat-warning/20";
    return "bg-pat-bg-surface-secondary/50 border-pat-border/30";
  };

  return (
    <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-pat-text-primary">Live Signal Pipeline</h3>
        <div className="flex items-center gap-2">
          <span className="text-[10px] text-pat-text-muted">{displaySignals.length} active</span>
          <button onClick={() => refetch()} className="text-[10px] text-pat-text-muted hover:text-pat-text-primary">↻ Refresh</button>
        </div>
      </div>
      {displaySignals.length === 0 ? (
        <div className="text-xs text-pat-text-muted py-6 text-center">No directional signals detected. The engine is scanning market conditions.</div>
      ) : (
        <div className="space-y-2 max-h-[400px] overflow-y-auto">
          {displaySignals.map((s) => {
            const entry = parseFloat(s.EntryPrice || "0");
            const sl = parseFloat(s.StopLoss || "0");
            const tp1 = parseFloat(s.TP1 || "0");
            const tp2 = parseFloat(s.TP2 || "0");
            const tp3 = parseFloat(s.TP3 || "0");
            const slDist = entry && sl ? Math.abs(entry - sl) : 0;
            const rr1 = slDist && tp1 ? Math.abs(tp1 - entry) / slDist : 0;
            const rr2 = slDist && tp2 ? Math.abs(tp2 - entry) / slDist : 0;
            const rr3 = slDist && tp3 ? Math.abs(tp3 - entry) / slDist : 0;
            const score = parseFloat(s.RawScore || "0");
            const prob = parseFloat(s.CalibratedProbability || "0");
            const evidence = s.Evidence ?? [];
            const keyEvidence = evidence.slice(0, 3).map(e => e.feature || e.pillar || "").filter(Boolean);
            const isCandidate = s.Direction.includes("CANDIDATE");
            const isExecutable = s.Executable || s.SignalClass === "EXECUTABLE";
            const isAdvisory = s.SignalClass === "ADVISORY" || (isCandidate && !isExecutable);

            return (
              <div key={s.ID} className={`rounded-lg border p-3 ${dirBg(s.Direction)}`}>
                {/* Header row */}
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className={`text-xs font-bold ${dirColor(s.Direction)}`}>{s.Direction}</span>
                    <span className="text-xs text-pat-text-secondary">{strategyLabel(s.StrategyID)}</span>
                    {isAdvisory && (
                      <span className="text-[9px] px-1.5 py-0.5 rounded-full bg-pat-info/20 text-pat-info" title="Advisory-only — not routed to your terminal for execution">ADVISORY</span>
                    )}
                    {isCandidate && !isAdvisory && (
                      <span className="text-[9px] px-1.5 py-0.5 rounded-full bg-pat-warning/20 text-pat-warning">MICROPROFIT</span>
                    )}
                    {isExecutable && (
                      <span className="text-[9px] px-1.5 py-0.5 rounded-full bg-pat-success/20 text-pat-success">EXECUTABLE</span>
                    )}
                  </div>
                  <div className="flex items-center gap-2 text-[10px] text-pat-text-muted">
                    {score > 0 && <span>Score: {score.toFixed(1)}</span>}
                    {prob > 0 && <span>Prob: {(prob * 100).toFixed(1)}%</span>}
                    {/* Broker-clock render: same clock as the EAs' TimeCurrent(), not the browser's TZ */}
                    <span>{formatBrokerTimestamp(s.CreatedAt, brokerOffset, "HH:mm:ss")}</span>
                  </div>
                </div>

                {/* Geometry row */}
                {entry > 0 && sl > 0 && (
                  <div className="grid grid-cols-5 gap-2 text-[11px] mb-1.5">
                    <div>
                      <span className="text-pat-text-muted">Entry</span>
                      <div className="font-mono text-pat-text-primary tabular-nums">{entry.toFixed(2)}</div>
                    </div>
                    <div>
                      <span className="text-pat-text-muted">SL</span>
                      <div className="font-mono text-pat-danger tabular-nums">{sl.toFixed(2)}</div>
                    </div>
                    <div>
                      <span className="text-pat-text-muted">TP1</span>
                      <div className="font-mono text-pat-success tabular-nums">{tp1 > 0 ? tp1.toFixed(2) : "—"}</div>
                      {rr1 > 0 && <span className="text-[9px] text-pat-text-muted">R:R {rr1.toFixed(2)}</span>}
                    </div>
                    <div>
                      <span className="text-pat-text-muted">TP2</span>
                      <div className="font-mono text-pat-success tabular-nums">{tp2 > 0 ? tp2.toFixed(2) : "—"}</div>
                      {rr2 > 0 && <span className="text-[9px] text-pat-text-muted">R:R {rr2.toFixed(2)}</span>}
                    </div>
                    <div>
                      <span className="text-pat-text-muted">TP3</span>
                      <div className="font-mono text-pat-success tabular-nums">{tp3 > 0 ? tp3.toFixed(2) : "—"}</div>
                      {rr3 > 0 && <span className="text-[9px] text-pat-text-muted">R:R {rr3.toFixed(2)}</span>}
                    </div>
                  </div>
                )}

                {/* Evidence + regime */}
                <div className="flex flex-wrap items-center gap-2 text-[10px]">
                  {s.Regime && <span className="text-pat-text-muted">Regime: {s.Regime}</span>}
                  {s.Session && <span className="text-pat-text-muted">Session: {s.Session}</span>}
                  {keyEvidence.length > 0 && (
                    <span className="text-pat-text-muted">Evidence: {keyEvidence.join(", ")}</span>
                  )}
                  {s.ReasonCodes && s.ReasonCodes.length > 0 && (
                    <span className="text-pat-text-muted">Reasons: {s.ReasonCodes.join(", ")}</span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
