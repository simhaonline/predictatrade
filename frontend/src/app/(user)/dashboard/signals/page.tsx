"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { useState, useEffect, useRef } from "react";
import { getGlobalWs, type WsMessage } from "@/lib/websocket";

// Interface matches the actual Go engine API response (PascalCase)
interface EngineSignal {
  ID: string;
  Direction: string;
  StrategyID: string;
  Status: string;
  RawScore: string;
  CalibratedProbability: string;
  EntryPrice: string;
  StopLoss: string;
  TP1: string;
  TP2: string;
  TP3: string;
  CreatedAt: string;
  Symbol: string;
  Regime?: string;
  Session?: string;
  Executable?: boolean;
  Grade?: string;
  ReasonCodes?: string[];
}

export default function UserSignalsPage() {
  const [liveSignals, setLiveSignals] = useState<EngineSignal[]>([]);
  const sigBuffer = useRef<EngineSignal[]>([]);
  const ws = getGlobalWs();

  const { data: signalsData, isLoading, error, refetch } = useQuery<{ signals: EngineSignal[] }>({
    queryKey: ["engine-signals-user"],
    queryFn: async () => {
      const res = await customInstance.get("/signals");
      return res.data as { signals: EngineSignal[] };
    },
    refetchInterval: 10000,
  });

  useEffect(() => {
    ws.connect();
    const unsub = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "signal") {
        const s = msg.payload;
        // Map WebSocket signal to EngineSignal format
        const signal: EngineSignal = {
          ID: s.id,
          Direction: s.direction,
          StrategyID: s.strategy,
          Status: s.status,
          RawScore: "0",
          CalibratedProbability: String(s.probability || 0),
          EntryPrice: String(s.entryPrice || 0),
          StopLoss: String(s.stopLoss || 0),
          TP1: String(s.takeProfit || 0),
          TP2: "0",
          TP3: "0",
          CreatedAt: s.timestamp,
          Symbol: "XAUUSD",
        };
        sigBuffer.current = [signal, ...sigBuffer.current].slice(0, 50);
        setLiveSignals(sigBuffer.current);
      }
    });
    return () => { unsub(); };
  }, [ws]);

  const restSignals = signalsData?.signals ?? [];
  const combinedSignals = liveSignals.length > 0 ? liveSignals : restSignals;

  const dirColor = (dir: string): string => {
    if (dir === "BUY") return "text-pat-success";
    if (dir === "SELL") return "text-pat-danger";
    if (dir === "BUY_CANDIDATE") return "text-pat-warning";
    if (dir === "SELL_CANDIDATE") return "text-pat-candidate-sell";
    return "text-pat-text-secondary";
  };

  const columns: DataTableColumn<EngineSignal>[] = [
    {
      key: "Direction",
      header: "Direction",
      sortable: true,
      cell: (row) => (
        <div className="flex items-center gap-1.5">
          <span className={`text-xs font-bold ${dirColor(row.Direction)}`}>{row.Direction}</span>
          {row.Executable && (
            <span className="text-[9px] px-1 py-0.5 rounded-full bg-pat-success/15 text-pat-success">EXEC</span>
          )}
        </div>
      ),
    },
    {
      key: "StrategyID",
      header: "Strategy",
      sortable: true,
      cell: (row) => <span className="text-xs text-pat-text-secondary">{row.StrategyID?.replace(/_/g, " ")}</span>,
    },
    {
      key: "RawScore",
      header: "Score",
      sortable: true,
      cell: (row) => {
        const score = parseFloat(row.RawScore || "0");
        return <span className="text-xs text-pat-text-primary tabular-nums">{score > 0 ? score.toFixed(1) : "—"}</span>;
      },
    },
    {
      key: "CalibratedProbability",
      header: "Probability",
      sortable: true,
      cell: (row) => {
        const prob = parseFloat(row.CalibratedProbability || "0");
        return (
          <span className="text-xs text-pat-text-primary" title="Calibrated probability — shows 'Pending' until calibration model is validated">
            {prob > 0 ? `${(prob * 100).toFixed(1)}%` : "Pending"}
          </span>
        );
      },
    },
    {
      key: "EntryPrice",
      header: "Entry",
      cell: (row) => {
        const v = parseFloat(row.EntryPrice || "0");
        return <span className="text-xs text-pat-text-primary tabular-nums">{v > 0 ? v.toFixed(2) : "—"}</span>;
      },
    },
    {
      key: "StopLoss",
      header: "SL",
      cell: (row) => {
        const v = parseFloat(row.StopLoss || "0");
        return <span className="text-xs text-pat-danger tabular-nums">{v > 0 ? v.toFixed(2) : "—"}</span>;
      },
    },
    {
      key: "TP1",
      header: "TP1",
      cell: (row) => {
        const v = parseFloat(row.TP1 || "0");
        return <span className="text-xs text-pat-success tabular-nums">{v > 0 ? v.toFixed(2) : "—"}</span>;
      },
    },
    {
      key: "TP2",
      header: "TP2",
      cell: (row) => {
        const v = parseFloat(row.TP2 || "0");
        return <span className="text-xs text-pat-success/70 tabular-nums">{v > 0 ? v.toFixed(2) : "—"}</span>;
      },
    },
    {
      key: "TP3",
      header: "TP3",
      cell: (row) => {
        const v = parseFloat(row.TP3 || "0");
        return <span className="text-xs text-pat-success/50 tabular-nums">{v > 0 ? v.toFixed(2) : "—"}</span>;
      },
    },
    {
      key: "Status",
      header: "Status",
      cell: (row) => <StatusBadge status={row.Status} size="sm" />,
    },
    {
      key: "Regime",
      header: "Regime",
      cell: (row) => <span className="text-[10px] text-pat-text-muted">{row.Regime || "—"}</span>,
    },
    {
      key: "CreatedAt",
      header: "Date",
      sortable: true,
      cell: (row) => (
        <span className="text-xs text-pat-text-muted">
          {row.CreatedAt && row.CreatedAt !== "0001-01-01T00:00:00Z" ? format(new Date(row.CreatedAt), "MMM d, HH:mm:ss") : "—"}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Signals</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Your XAUUSD trading signals from the real-time engine.</p>
        <div className="flex flex-wrap gap-3 text-[10px] text-pat-text-muted mt-2">
          <span><span className="text-pat-success font-bold">BUY</span> = Qualified signal</span>
          <span><span className="text-pat-warning font-bold">BUY_CANDIDATE</span> = Advisory (microprofit)</span>
          <span><span className="text-pat-danger font-bold">SELL</span> = Qualified short</span>
          <span><span className="text-pat-candidate-sell font-bold">SELL_CANDIDATE</span> = Advisory short</span>
          <span><span className="text-pat-text-secondary font-bold">NO-TRADE</span> = No signal</span>
        </div>
      </div>
      <DataTable data={combinedSignals} columns={columns} loading={isLoading} error={error as Error | null} onRetry={refetch} />
    </div>
  );
}
