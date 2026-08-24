"use client";
import { useQuery } from "@tanstack/react-query";
import React from "react";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { useState, useEffect, useRef } from "react";
import { getGlobalWs, type WsMessage } from "@/lib/websocket";
import { IconChevronDown, IconChevronUp } from "@tabler/icons-react";
import SignalEvidencePanel from "@/components/signal/signal-evidence";

interface GoSignal {
  ID: string;
  Symbol: string;
  StrategyID: string;
  Direction: string;
  Grade: string;
  RawScore: string;
  LongScore: string;
  ShortScore: string;
  CalibratedProbability: string;
  EntryPrice: string;
  StopLoss: string;
  TP1: string;
  TP2: string;
  TP3: string;
  Regime: string;
  Session: string;
  Timeframe: string;
  Status: string;
  ReasonCodes: string[] | null;
  Evidence: DiagnosticRecord[] | null;
  GateResults: DiagnosticRecord[] | null;
  CreatedAt: string;
  ExpiresAt: string;
}

type DiagnosticRecord = Record<string, string | number | string[] | null | undefined>;

const STRATEGY_TABS = ["ALL", "STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"] as const;
const DIRECTION_FILTERS = ["ALL", "BUY", "BUY_CANDIDATE", "SELL", "SELL_CANDIDATE", "NO-TRADE"] as const;

export default function AdminSignalsPage() {
  const [activeTab, setActiveTab] = useState<typeof STRATEGY_TABS[number]>("ALL");
  const [directionFilter, setDirectionFilter] = useState<typeof DIRECTION_FILTERS[number]>("ALL");
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const [liveSignals, setLiveSignals] = useState<GoSignal[]>([]);
  const sigBuffer = useRef<GoSignal[]>([]);
  const ws = getGlobalWs();

  const { data: signalsData, isLoading, error, refetch } = useQuery<{ signals: GoSignal[] }>({
    queryKey: ["engine-signals"],
    queryFn: async () => {
      const res = await customInstance.get("/signals");
      return res.data as { signals: GoSignal[] };
    },
    refetchInterval: 30000,
  });

  useEffect(() => {
    ws.connect();
    const unsub = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "signal") {
        const s = msg.payload as unknown as Record<string, unknown>;
        const entry: GoSignal = {
          ID: s.id as string || s.ID as string || crypto.randomUUID(),
          Symbol: (s.symbol as string) || (s.Symbol as string) || "XAUUSD",
          StrategyID: (s.strategy as string) || (s.StrategyID as string) || "",
          Direction: (s.direction as string) || (s.Direction as string) || "NO-TRADE",
          Grade: "", RawScore: String(s.probability ? (Number(s.probability) * 100).toFixed(2) : "0"),
          LongScore: "0", ShortScore: "0",
          CalibratedProbability: String(s.probability ?? "0"),
          EntryPrice: String(s.entryPrice ?? s.entry_price ?? "0"),
          StopLoss: String(s.stopLoss ?? s.stop_loss ?? "0"),
          TP1: String(s.takeProfit ?? s.take_profit ?? s.tp1 ?? "0"),
          TP2: "0", TP3: "0", Regime: "", Session: "", Timeframe: "",
          Status: (s.status as string) || "DETECTED", ReasonCodes: null,
          Evidence: null, GateResults: null,
          CreatedAt: (s.timestamp as string) || new Date().toISOString(), ExpiresAt: "",
        };
        sigBuffer.current = [entry, ...sigBuffer.current].slice(0, 100);
        setLiveSignals(sigBuffer.current);
      }
    });
    return () => { unsub(); };
  }, [ws]);

  const allSignals = liveSignals.length > 0 ? liveSignals : (signalsData?.signals ?? []);

  // Apply filters
  const filteredSignals = allSignals.filter((s) => {
    if (activeTab !== "ALL" && s.StrategyID !== activeTab) return false;
    if (directionFilter !== "ALL" && s.Direction !== directionFilter) return false;
    return true;
  });

  const fmtScore = (val: string) => {
    const n = parseFloat(val);
    if (isNaN(n) || n === 0) return "0";
    return n.toFixed(2);
  };

  const fmtPrice = (val: string) => {
    const n = parseFloat(val);
    if (isNaN(n) || n === 0) return "—";
    return n.toFixed(2);
  };

  const fmtProb = (val: string) => {
    const n = parseFloat(val);
    if (isNaN(n) || n === 0) return "Pending";
    return `${(n * 100).toFixed(1)}%`;
  };

  const dirClass = (dir: string) => (
    dir === "BUY" ? "text-pat-success" :
    dir === "SELL" ? "text-pat-danger" :
    dir === "BUY_CANDIDATE" ? "text-pat-warning" :
    dir === "SELL_CANDIDATE" ? "text-pat-candidate-sell" :
    "text-pat-text-secondary"
  );

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Signal Panel</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Live and historical trading signals from the Go real-time engine, including NO-TRADE decisions.</p>
      <div className="flex flex-wrap gap-3 text-[10px] text-pat-text-muted mt-2">
        <span><span className="text-pat-success font-bold">BUY</span> = Qualified (score ≥ trade threshold + all gates passed)</span>
        <span><span className="text-pat-warning font-bold">BUY_CANDIDATE</span> = Advisory (score ≥ candidate threshold, below trade threshold)</span>
        <span><span className="text-pat-danger font-bold">SELL</span> = Qualified short</span>
        <span><span className="text-pat-candidate-sell font-bold">SELL_CANDIDATE</span> = Advisory short</span>
        <span><span className="text-pat-text-secondary font-bold">NO-TRADE</span> = Insufficient score or gate veto</span>
      </div>
      <div className="text-[10px] text-pat-text-muted mt-1">
        <span className="font-medium">Prob</span> = Calibrated probability. Shows &quot;Pending&quot; until calibration model is validated (SOW §16, §36). Raw score is shown in the Score column.
      </div>
      </div>

      {/* Strategy Tabs */}
      <div className="flex flex-wrap gap-2">
        {STRATEGY_TABS.map((tab) => (
          <button key={tab} onClick={() => setActiveTab(tab)}
            className={`text-xs px-3 py-1.5 rounded transition-colors whitespace-nowrap ${activeTab === tab ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"}`}>
            {tab === "ALL" ? "All" : tab.replace(/_/g, " ")}
          </button>
        ))}
      </div>

      {/* Direction Filter */}
      <div className="flex gap-2">
        {DIRECTION_FILTERS.map((f) => (
          <button key={f} onClick={() => setDirectionFilter(f)}
            className={`text-[10px] px-2 py-1 rounded transition-colors ${directionFilter === f ? "bg-pat-primary text-pat-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-muted hover:bg-pat-bg-surface-secondary"}`}>
            {f}
          </button>
        ))}
        <span className="text-xs text-pat-text-muted ml-auto self-center">{filteredSignals.length} signals</span>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-10 bg-pat-bg-surface-secondary/50 rounded animate-pulse" />
          ))}
        </div>
      ) : error ? (
        <div className="text-center py-12 border border-pat-border rounded-lg bg-pat-bg-surface/50">
          <div className="text-pat-danger text-sm mb-2">Failed to load data</div>
          <div className="text-pat-text-muted text-xs mb-4">{(error as Error).message}</div>
          <button onClick={() => refetch()} className="text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary px-3 py-1.5 rounded">Retry</button>
        </div>
      ) : filteredSignals.length === 0 ? (
        <div className="text-center py-12 border border-pat-border rounded-lg bg-pat-bg-surface/50">
          <div className="text-pat-text-muted text-sm">No signals match the current filters</div>
        </div>
      ) : (
        <div className="overflow-x-auto border border-pat-border rounded-lg">
          <table className="w-full text-sm text-left">
            <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
              <tr>
                <th className="px-3 py-3"></th>
                <th className="px-3 py-3 font-medium">Time</th>
                <th className="px-3 py-3 font-medium">Direction</th>
                <th className="px-3 py-3 font-medium">Strategy</th>
                <th className="px-3 py-3 font-medium">Symbol</th>
                <th className="px-3 py-3 font-medium">Prob</th>
                <th className="px-3 py-3 font-medium">Score</th>
                <th className="px-3 py-3 font-medium">Entry</th>
                <th className="px-3 py-3 font-medium">SL</th>
                <th className="px-3 py-3 font-medium">TP1</th>
                <th className="px-3 py-3 font-medium">TP2</th>
                <th className="px-3 py-3 font-medium">TP3</th>
                <th className="px-3 py-3 font-medium">Regime</th>
                <th className="px-3 py-3 font-medium">Session</th>
                <th className="px-3 py-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-800">
              {filteredSignals.map((row) => {
                const isOpen = expandedRow === row.ID;
                return (
                  <React.Fragment key={row.ID}>
                    <tr
                      className="hover:bg-pat-table-hover transition-colors cursor-pointer"
                      onClick={() => setExpandedRow(isOpen ? null : row.ID)}
                    >
                      <td className="px-3 py-3">
                        {isOpen ? <IconChevronUp size={14} className="text-pat-text-muted" /> : <IconChevronDown size={14} className="text-pat-text-muted" />}
                      </td>
                      <td className="px-3 py-3 text-xs text-pat-text-muted whitespace-nowrap">{row.CreatedAt ? format(new Date(row.CreatedAt), "MMM d, HH:mm:ss") : "—"}</td>
                      <td className="px-3 py-3">
                        <span className={`text-xs font-bold ${dirClass(row.Direction)}`}>{row.Direction}</span>
                      </td>
                      <td className="px-3 py-3 text-xs text-pat-text-secondary whitespace-nowrap">{row.StrategyID}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-primary">{row.Symbol}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-primary">{fmtProb(row.CalibratedProbability)}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-secondary">{fmtScore(row.RawScore)}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-primary">{fmtPrice(row.EntryPrice)}</td>
                      <td className="px-3 py-3 text-xs text-pat-danger">{fmtPrice(row.StopLoss)}</td>
                      <td className="px-3 py-3 text-xs text-pat-success">{fmtPrice(row.TP1)}</td>
                      <td className="px-3 py-3 text-xs text-pat-success">{fmtPrice(row.TP2)}</td>
                      <td className="px-3 py-3 text-xs text-pat-success">{fmtPrice(row.TP3)}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-muted">{row.Regime || "—"}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-muted">{row.Session || "—"}</td>
                      <td className="px-3 py-3"><StatusBadge status={row.Status} size="sm" /></td>
                    </tr>
                    {isOpen && (
                      <tr className="bg-pat-bg-surface-secondary/30">
                        <td colSpan={15} className="px-4 py-4">
                          <SignalEvidencePanel sig={row} />
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
