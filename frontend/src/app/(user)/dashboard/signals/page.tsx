"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { useState, useEffect, useRef } from "react";
import { getGlobalWs, type WsMessage } from "@/lib/websocket";

interface Signal {
  id: string;
  direction: string;
  strategy: string;
  status: string;
  probability: number;
  entry_price: number;
  stop_loss: number;
  take_profit: number;
  created_at: string;
  symbol: string;
}

export default function UserSignalsPage() {
  const [liveSignals, setLiveSignals] = useState<Signal[]>([]);
  const sigBuffer = useRef<Signal[]>([]);
  const ws = getGlobalWs();

  const { data: signals, isLoading, error, refetch } = useQuery<{ signals: Signal[] }>({
    queryKey: ["engine-signals-user"],
    queryFn: async () => {
      const res = await customInstance.get("/api/v1/signals");
      return res.data as { signals: Signal[] };
    },
  });

  useEffect(() => {
    ws.connect();
    const unsub = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "signal") {
        const s = msg.payload;
        const entry: Signal = {
          id: s.id, direction: s.direction, strategy: s.strategy, status: s.status,
          probability: s.probability, entry_price: s.entryPrice, stop_loss: s.stopLoss,
          take_profit: s.takeProfit, created_at: s.timestamp, symbol: "XAUUSD",
        };
        sigBuffer.current = [entry, ...sigBuffer.current].slice(0, 50);
        setLiveSignals(sigBuffer.current);
      }
    });
    return () => { unsub(); };
  }, [ws]);

  const combinedSignals = liveSignals.length > 0 ? liveSignals : (signals?.signals ?? []);

  const columns: DataTableColumn<Signal>[] = [
    { key: "direction", header: "Direction", sortable: true, cell: (row) => (
      <span className={`text-xs font-bold ${
              row.direction === "BUY" ? "text-pat-success" :
              row.direction === "SELL" ? "text-pat-danger" :
              row.direction === "BUY_CANDIDATE" ? "text-pat-warning" :
              row.direction === "SELL_CANDIDATE" ? "text-pat-candidate-sell" :
              "text-pat-text-secondary"
            }`}>{row.direction}</span>
    )},
    { key: "strategy", header: "Strategy", sortable: true, cell: (row) => <span className="text-xs text-pat-text-secondary">{row.strategy}</span> },
    { key: "probability", header: "Probability", sortable: true, cell: (row) => <span className="text-xs text-pat-text-primary" title="Calibrated probability — shows 'Pending' until calibration model is validated (SOW §16, §36)">{row.probability ? `${(row.probability * 100).toFixed(1)}%` : "Pending"}</span> },
    { key: "entry_price", header: "Entry", cell: (row) => <span className="text-xs text-pat-text-primary">{row.entry_price ? row.entry_price.toFixed(2) : "—"}</span> },
    { key: "stop_loss", header: "SL", cell: (row) => <span className="text-xs text-pat-danger">{row.stop_loss ? row.stop_loss.toFixed(2) : "—"}</span> },
    { key: "take_profit", header: "TP", cell: (row) => <span className="text-xs text-pat-success">{row.take_profit ? row.take_profit.toFixed(2) : "—"}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} size="sm" /> },
    { key: "created_at", header: "Date", sortable: true, cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, HH:mm") : "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Signals</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Your XAUUSD trading signals from the real-time engine.</p>
      <div className="flex flex-wrap gap-3 text-[10px] text-pat-text-muted mt-2">
        <span><span className="text-pat-success font-bold">BUY</span> = Qualified signal</span>
        <span><span className="text-pat-warning font-bold">BUY_CANDIDATE</span> = Advisory (not yet qualified)</span>
        <span><span className="text-pat-danger font-bold">SELL</span> = Qualified short</span>
        <span><span className="text-pat-candidate-sell font-bold">SELL_CANDIDATE</span> = Advisory short</span>
        <span><span className="text-pat-text-secondary font-bold">NO-TRADE</span> = No signal</span>
      </div>
      </div>
      <DataTable data={combinedSignals} columns={columns} loading={isLoading} error={error as Error | null} onRetry={refetch} />
    </div>
  );
}
