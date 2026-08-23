"use client";
import { useQuery } from "@tanstack/react-query";
import React from "react";
import { customInstance } from "@/lib/axios-instance";
import { format } from "date-fns";
import { useState, useEffect, useRef, useMemo } from "react";
import { getGlobalWs, type WsMessage, type SignalEvent } from "@/lib/websocket";
import { IconChevronRight, IconChevronDown, IconBrain, IconShieldCheck, IconListDetails } from "@tabler/icons-react";

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
  Evidence?: Record<string, string> | string[];
  PillarContributions?: Record<string, number>;
  AiVerification?: string;
  RiskDecision?: string;
}

const STRATEGIES = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"];
const DIRECTIONS = ["BUY", "SELL", "BUY_CANDIDATE", "SELL_CANDIDATE", "NO-TRADE"];

function mapWs(s: SignalEvent): EngineSignal {
  return {
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
    ReasonCodes: [],
  };
}

function na(v: unknown): string {
  if (v === undefined || v === null || v === "" || (Array.isArray(v) && v.length === 0)) return "N/A";
  return String(v);
}

export default function UserSignalsPage() {
  const [liveSignals, setLiveSignals] = useState<EngineSignal[]>([]);
  const sigBuffer = useRef<EngineSignal[]>([]);
  const ws = getGlobalWs();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [filterStrategy, setFilterStrategy] = useState("ALL");
  const [filterDirection, setFilterDirection] = useState("ALL");
  const [filterRegime, setFilterRegime] = useState("ALL");

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
        const signal = mapWs(msg.payload);
        sigBuffer.current = [signal, ...sigBuffer.current].slice(0, 50);
        setLiveSignals(sigBuffer.current);
      }
    });
    return () => { unsub(); };
  }, [ws]);

  const restSignals = signalsData?.signals ?? [];
  const combinedSignals = liveSignals.length > 0 ? liveSignals : restSignals;

  const regimes = useMemo(() => Array.from(new Set(combinedSignals.map((s) => s.Regime).filter(Boolean) as string[])), [combinedSignals]);

  const filtered = useMemo(
    () =>
      combinedSignals.filter(
        (s) =>
          (filterStrategy === "ALL" || s.StrategyID === filterStrategy) &&
          (filterDirection === "ALL" || s.Direction === filterDirection) &&
          (filterRegime === "ALL" || (s.Regime ?? "") === filterRegime),
      ),
    [combinedSignals, filterStrategy, filterDirection, filterRegime],
  );

  const dirColor = (dir: string): string => {
    if (dir === "BUY") return "text-pat-success";
    if (dir === "SELL") return "text-pat-danger";
    if (dir === "BUY_CANDIDATE") return "text-pat-warning";
    if (dir === "SELL_CANDIDATE") return "text-pat-candidate-sell";
    return "text-pat-text-secondary";
  };

  const num = (v: string) => parseFloat(v || "0");

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

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <FilterSelect label="Strategy" value={filterStrategy} onChange={setFilterStrategy} options={["ALL", ...STRATEGIES]} />
        <FilterSelect label="Direction" value={filterDirection} onChange={setFilterDirection} options={["ALL", ...DIRECTIONS]} />
        <FilterSelect label="Regime" value={filterRegime} onChange={setFilterRegime} options={["ALL", ...regimes]} />
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-10 bg-pat-bg-surface-secondary/50 rounded animate-pulse" />
          ))}
        </div>
      ) : error ? (
        <div className="text-center py-12 border border-pat-table-border rounded-lg bg-pat-bg-surface/50">
          <div className="text-pat-danger text-sm mb-2">Failed to load data</div>
          <div className="text-pat-text-muted text-xs mb-4">{error.message}</div>
          <button onClick={() => refetch()} className="text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary px-3 py-1.5 rounded">Retry</button>
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 border border-pat-table-border rounded-lg bg-pat-bg-surface/50">
          <div className="text-pat-text-muted text-sm">No signals match the current filters</div>
        </div>
      ) : (
        <div className="overflow-x-auto border border-pat-table-border rounded-lg">
          <table className="w-full text-sm text-left">
            <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
              <tr>
                <th className="px-3 py-3"></th>
                <th className="px-3 py-3 font-medium">Direction</th>
                <th className="px-3 py-3 font-medium">Strategy</th>
                <th className="px-3 py-3 font-medium">Score</th>
                <th className="px-3 py-3 font-medium">Prob.</th>
                <th className="px-3 py-3 font-medium">Entry</th>
                <th className="px-3 py-3 font-medium">SL</th>
                <th className="px-3 py-3 font-medium">TP1</th>
                <th className="px-3 py-3 font-medium">Status</th>
                <th className="px-3 py-3 font-medium">Regime</th>
                <th className="px-3 py-3 font-medium">Date</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-800">
              {filtered.map((row) => {
                const isOpen = expanded === row.ID;
                return (
                  <React.Fragment key={row.ID}>
                    <tr className="hover:bg-pat-table-hover transition-colors cursor-pointer" onClick={() => setExpanded(isOpen ? null : row.ID)}>
                      <td className="px-3 py-3">
                        {isOpen ? <IconChevronDown size={14} className="text-pat-text-muted" /> : <IconChevronRight size={14} className="text-pat-text-muted" />}
                      </td>
                      <td className="px-3 py-3">
                        <div className="flex items-center gap-1.5">
                          <span className={`text-xs font-bold ${dirColor(row.Direction)}`}>{row.Direction}</span>
                          {row.Executable && <span className="text-[9px] px-1 py-0.5 rounded-full bg-pat-success/15 text-pat-success">EXEC</span>}
                        </div>
                      </td>
                      <td className="px-3 py-3 text-xs text-pat-text-secondary">{row.StrategyID?.replace(/_/g, " ")}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-primary tabular-nums">{num(row.RawScore) > 0 ? num(row.RawScore).toFixed(1) : "—"}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-primary" title="Calibrated probability — shows 'Pending' until calibration model is validated">
                        {num(row.CalibratedProbability) > 0 ? `${(num(row.CalibratedProbability) * 100).toFixed(1)}%` : "Pending"}
                      </td>
                      <td className="px-3 py-3 text-xs text-pat-text-primary tabular-nums">{num(row.EntryPrice) > 0 ? num(row.EntryPrice).toFixed(2) : "—"}</td>
                      <td className="px-3 py-3 text-xs text-pat-danger tabular-nums">{num(row.StopLoss) > 0 ? num(row.StopLoss).toFixed(2) : "—"}</td>
                      <td className="px-3 py-3 text-xs text-pat-success tabular-nums">{num(row.TP1) > 0 ? num(row.TP1).toFixed(2) : "—"}</td>
                      <td className="px-3 py-3"><StatusText status={row.Status} /></td>
                      <td className="px-3 py-3 text-[10px] text-pat-text-muted">{row.Regime || "—"}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-muted">
                        {row.CreatedAt && row.CreatedAt !== "0001-01-01T00:00:00Z" ? format(new Date(row.CreatedAt), "MMM d, HH:mm:ss") : "—"}
                      </td>
                    </tr>
                    {isOpen && (
                      <tr className="bg-pat-bg-surface-secondary/30">
                        <td colSpan={11} className="px-4 py-4">
                          <Explainability signal={row} />
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

function FilterSelect({ label, value, onChange, options }: { label: string; value: string; onChange: (v: string) => void; options: string[] }) {
  return (
    <label className="flex flex-col gap-1 text-xs text-pat-text-secondary">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-md border border-pat-input-border bg-pat-input-bg px-2 py-1.5 text-sm text-pat-input-text outline-none focus:border-primary"
      >
        {options.map((o) => (
          <option key={o} value={o}>{o === "ALL" ? "All" : o.replace(/_/g, " ")}</option>
        ))}
      </select>
    </label>
  );
}

function StatusText({ status }: { status: string }) {
  const cls =
    status === "ACTIVE" || status === "QUALIFIED" ? "text-pat-success"
      : status === "CLOSED" ? "text-pat-text-muted"
      : status === "EXPIRED" || status === "INVALIDATED" ? "text-pat-danger"
      : "text-pat-text-secondary";
  return <span className={`text-xs px-2 py-0.5 rounded-full border ${cls}`}>{status}</span>;
}

function Explainability({ signal }: { signal: EngineSignal }) {
  const pill: Record<string, number> = signal.PillarContributions ?? {};
  const evidence = signal.Evidence;
  const evidenceRows: [string, string][] = Array.isArray(evidence)
    ? evidence.map((e, i) => [`Item ${i + 1}`, e])
    : evidence && typeof evidence === "object"
    ? Object.entries(evidence as Record<string, string>)
    : [];

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <div className="space-y-2">
        <SectionTitle icon={<IconListDetails size={14} />} title="Reason codes" />
        <div className="flex flex-wrap gap-1.5">
          {(signal.ReasonCodes ?? []).length > 0 ? (
            signal.ReasonCodes!.map((rc, i) => (
              <span key={i} className="text-[10px] px-2 py-0.5 rounded-full bg-pat-info/10 text-pat-info border border-pat-info/20">{rc}</span>
            ))
          ) : (
            <span className="text-xs text-pat-text-muted">N/A</span>
          )}
        </div>

        <SectionTitle icon={<IconBrain size={14} />} title="Pillar contributions" />
        {Object.keys(pill).length > 0 ? (
          <div className="space-y-1">
            {Object.entries(pill).map(([k, v]) => (
              <div key={k} className="flex items-center gap-2 text-xs">
                <span className="w-32 text-pat-text-secondary truncate">{k}</span>
                <div className="flex-1 h-1.5 rounded bg-pat-bg-surface-secondary overflow-hidden">
                  <div className="h-full bg-pat-primary" style={{ width: `${Math.max(2, Math.min(100, Math.abs(v)))}%` }} />
                </div>
                <span className="w-10 text-right tabular-nums text-pat-text-primary">{v}</span>
              </div>
            ))}
          </div>
        ) : (
          <span className="text-xs text-pat-text-muted">N/A</span>
        )}
      </div>

      <div className="space-y-2">
        <SectionTitle icon={<IconShieldCheck size={14} />} title="AI verification & risk decision" />
        <Row label="AI verification" value={na(signal.AiVerification)} />
        <Row label="Risk decision" value={na(signal.RiskDecision)} />
        <Row label="Grade" value={na(signal.Grade)} />
        <Row label="Session" value={na(signal.Session)} />
        <Row label="Executable" value={signal.Executable === undefined ? "N/A" : signal.Executable ? "Yes" : "No"} />

        <SectionTitle title="Evidence" />
        {evidenceRows.length > 0 ? (
          <ul className="space-y-1 text-xs text-pat-text-secondary">
            {evidenceRows.map(([k, v], i) => (
              <li key={i}><span className="text-pat-text-muted">{k}:</span> {v}</li>
            ))}
          </ul>
        ) : (
          <span className="text-xs text-pat-text-muted">N/A</span>
        )}
      </div>
    </div>
  );
}

function SectionTitle({ title, icon }: { title: string; icon?: React.ReactNode }) {
  return (
    <div className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-pat-text-secondary mt-2">
      {icon} {title}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between text-xs">
      <span className="text-pat-text-muted">{label}</span>
      <span className="text-pat-text-primary">{value}</span>
    </div>
  );
}
