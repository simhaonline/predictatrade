"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
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

  const columns: DataTableColumn<GoSignal>[] = [
    { key: "CreatedAt", header: "Time", sortable: true, cell: (row) => (
      <span className="text-xs text-pat-text-muted whitespace-nowrap">{row.CreatedAt ? format(new Date(row.CreatedAt), "MMM d, HH:mm:ss") : "—"}</span>
    )},
    { key: "Direction", header: "Direction", sortable: true, cell: (row) => (
      <span className={`text-xs font-bold ${
              row.Direction === "BUY" ? "text-pat-success" :
              row.Direction === "SELL" ? "text-pat-danger" :
              row.Direction === "BUY_CANDIDATE" ? "text-pat-warning" :
              row.Direction === "SELL_CANDIDATE" ? "text-pat-candidate-sell" :
              "text-pat-text-secondary"
            }`}>{row.Direction}</span>
    )},
    { key: "StrategyID", header: "Strategy", sortable: true, cell: (row) => <span className="text-xs text-pat-text-secondary whitespace-nowrap">{row.StrategyID}</span> },
    { key: "Symbol", header: "Symbol", cell: (row) => <span className="text-xs text-pat-text-primary">{row.Symbol}</span> },
    { key: "CalibratedProbability", header: "Prob", sortable: true, cell: (row) => <span className="text-xs text-pat-text-primary">{fmtProb(row.CalibratedProbability)}</span> },
    { key: "RawScore", header: "Score", sortable: true, cell: (row) => <span className="text-xs text-pat-text-secondary">{fmtScore(row.RawScore)}</span> },
    { key: "EntryPrice", header: "Entry", cell: (row) => <span className="text-xs text-pat-text-primary">{fmtPrice(row.EntryPrice)}</span> },
    { key: "StopLoss", header: "SL", cell: (row) => <span className="text-xs text-pat-danger">{fmtPrice(row.StopLoss)}</span> },
    { key: "TP1", header: "TP1", cell: (row) => <span className="text-xs text-pat-success">{fmtPrice(row.TP1)}</span> },
    { key: "TP2", header: "TP2", cell: (row) => <span className="text-xs text-pat-success">{fmtPrice(row.TP2)}</span> },
    { key: "TP3", header: "TP3", cell: (row) => <span className="text-xs text-pat-success">{fmtPrice(row.TP3)}</span> },
    { key: "Regime", header: "Regime", cell: (row) => <span className="text-xs text-pat-text-muted">{row.Regime || "—"}</span> },
    { key: "Session", header: "Session", cell: (row) => <span className="text-xs text-pat-text-muted">{row.Session || "—"}</span> },
    { key: "Status", header: "Status", cell: (row) => <StatusBadge status={row.Status} size="sm" /> },
    { key: "expand", header: "", cell: (row) => (
      <button onClick={() => setExpandedRow(expandedRow === row.ID ? null : row.ID)} className="text-pat-text-muted hover:text-pat-text-primary">
        {expandedRow === row.ID ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />}
      </button>
    )},
  ];

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

      <DataTable data={filteredSignals} columns={columns} loading={isLoading} error={error as Error | null} onRetry={refetch} />

      {/* Expanded Row Details */}
      {expandedRow && (() => {
        const sig = filteredSignals.find((s) => s.ID === expandedRow);
        return sig ? (
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
            <SignalEvidencePanel sig={sig} />
          </div>
        ) : null;
      })()}
    </div>
  );
}
