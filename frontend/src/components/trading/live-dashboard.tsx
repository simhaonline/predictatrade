"use client";
import { useEffect, useRef, useState } from "react";
import { getGlobalWs, type WsMessage, type MarketDataEvent, type SignalEvent } from "@/lib/websocket";
import { rafBatch, RingBuffer } from "@/lib/performance";

export default function LiveDashboard({ isAdmin }: { isAdmin: boolean }) {
  void isAdmin;
  const bidRef = useRef<HTMLSpanElement>(null);
  const askRef = useRef<HTMLSpanElement>(null);
  const spreadRef = useRef<HTMLSpanElement>(null);
  const [signals, setSignals] = useState<SignalEvent[]>([]);
  const [agentConnected, setAgentConnected] = useState(false);
  const sigBuffer = useRef(new RingBuffer<SignalEvent>(50));
  const ws = getGlobalWs();

  useEffect(() => {
    ws.connect();
    const unsub = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "market") {
        const d = msg.payload as MarketDataEvent;
        rafBatch(() => {
          if (bidRef.current) bidRef.current.innerText = d.bid.toFixed(2);
          if (askRef.current) askRef.current.innerText = d.ask.toFixed(2);
          if (spreadRef.current) spreadRef.current.innerText = d.spread.toFixed(2);
        });
      }
      if (msg.type === "signal") {
        sigBuffer.current.push(msg.payload as SignalEvent);
        setSignals(sigBuffer.current.toArray());
      }
      if (msg.type === "agent") {
        setAgentConnected(msg.payload.connected);
      }
    });
    return () => { unsub(); ws.disconnect(); };
  }, [ws]);

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4 shadow-sm">
          <div className="text-xs text-pat-text-muted uppercase">Bid</div>
          <div className="text-3xl font-mono mt-1">
            <span className="text-pat-text-secondary">2500</span>
            <span ref={bidRef} className="text-pat-success">.00</span>
          </div>
        </div>
        <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4 shadow-sm">
          <div className="text-xs text-pat-text-muted uppercase">Ask</div>
          <div className="text-3xl font-mono mt-1">
            <span className="text-pat-text-secondary">2500</span>
            <span ref={askRef} className="text-pat-danger">.50</span>
          </div>
        </div>
        <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4 shadow-sm">
          <div className="text-xs text-pat-text-muted uppercase">Spread</div>
          <div className="text-3xl font-mono mt-1">
            <span ref={spreadRef} className="text-pat-warning">0.50</span>
          </div>
        </div>
      </div>

      <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4 shadow-sm">
        <div className="text-sm font-medium text-pat-text-primary mb-2">Latest Signals</div>
        {signals.length === 0 && (
          <div className="text-xs text-pat-text-muted">No signals yet.</div>
        )}
        <div className="space-y-2">
          {signals.map((s) => (
            <div key={s.id} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
              <div className="text-sm font-medium text-pat-text-primary">{s.direction} <span className="text-pat-text-muted">{s.strategy}</span></div>
              <div className="text-xs text-pat-text-secondary">Prob: {(s.probability * 100).toFixed(1)}%</div>
            </div>
          ))}
        </div>
      </div>

      <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4 shadow-sm">
        <div className="text-sm font-medium text-pat-text-primary mb-2">Agent Status</div>
        <div className="flex items-center gap-2">
          <span className={"inline-block h-2 w-2 rounded-full " + (agentConnected ? "bg-pat-success" : "bg-pat-danger")} />
          <span className="text-sm text-pat-text-secondary">{agentConnected ? "Connected" : "Disconnected"}</span>
        </div>
      </div>
    </div>
  );
}
