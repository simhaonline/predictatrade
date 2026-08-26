"use client";
import { useQuery } from "@tanstack/react-query";
import React from "react";
import { customInstance } from "@/lib/axios-instance";
import { visibleStrategies, type SubscriptionContext } from "@/lib/subscription-access";
import { fetchLicenses } from "@/lib/user-licensing-api";
import { format } from "date-fns";
import { useState, useEffect, useRef, useMemo } from "react";
import { getGlobalWs, type WsMessage, type SignalEvent } from "@/lib/websocket";
import { IconChevronRight, IconChevronDown } from "@tabler/icons-react";
import SignalEvidencePanel from "@/components/signal/signal-evidence";

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
  // Capital-protection sizing (engine-annotated; 0/absent = not yet computed)
  SuggestedLot?: string | number;
  RiskDollars?: string | number;
  RiskPctOfEquity?: string | number;
  SLDistancePoints?: string | number;
  // prompt.md Sections 12-14: Quality grade + Expectancy
  QualityGrade?: string;
  ExpectancyR?: string;
  ExpectancyScore?: number;
  // prompt.md Section 18: Rejection
  PrimaryRejectionReason?: string;
}

const STRATEGIES = ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB"];
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

export default function UserSignalsPage() {
  const [liveSignals, setLiveSignals] = useState<EngineSignal[]>([]);
  const sigBuffer = useRef<EngineSignal[]>([]);
  const ws = getGlobalWs();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [filterStrategy, setFilterStrategy] = useState("ALL");
  const [filterDirection, setFilterDirection] = useState("ALL");
  const [filterRegime, setFilterRegime] = useState("ALL");

  // Fetch user's license to determine allowed strategies
  const { data: licenses } = useQuery({
    queryKey: ["user-licenses-signals"],
    queryFn: async () => fetchLicenses(),
  });
  const userLicense = licenses?.[0];
  const allowedStrategies: string[] = (userLicense as any)?.allowed_strategies || [];
  const userPlan: string = (userLicense as any)?.plan || "FREE";

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
  const allSignals = liveSignals.length > 0 ? liveSignals : restSignals;
  // CRITICAL: Filter signals by user's subscription plan
  // FREE users only see signals for strategies their plan includes
  // While license is loading, show all signals (don't show empty state)
  const combinedSignals = licenses === undefined
    ? allSignals // Still loading — show everything
    : allowedStrategies.length > 0
      ? allSignals.filter(s => allowedStrategies.includes(s.StrategyID))
      : allSignals; // License loaded but no strategies configured — fallback to all

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

  const num = (v: string | number | undefined | null) => parseFloat(String(v ?? "0")) || 0;

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
        <FilterSelect label="Strategy" value={filterStrategy} onChange={setFilterStrategy} options={["ALL", ...(allowedStrategies.length > 0 ? STRATEGIES.filter(s => allowedStrategies.includes(s)) : STRATEGIES)]} />
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
          {allowedStrategies.length === 0 ? (
            <>
              <div className="text-pat-text-muted text-sm mb-2">No license found</div>
              <div className="text-pat-text-muted text-xs">Subscribe to a plan to access trading signals.</div>
            </>
          ) : combinedSignals.length === 0 ? (
            <>
              <div className="text-pat-text-muted text-sm mb-2">No signals available</div>
              <div className="text-pat-text-muted text-xs">Signals will appear when the engine generates them for your entitled strategies.</div>
            </>
          ) : (
            <>
              <div className="text-pat-text-muted text-sm mb-2">No signals match the current filters</div>
              <div className="text-pat-text-muted text-xs">Try adjusting your strategy, direction, or regime filters.</div>
            </>
          )}
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
                <th className="px-3 py-3 font-medium">Quality</th>
                <th className="px-3 py-3 font-medium">Status</th>
                <th className="px-3 py-3 font-medium">Regime</th>
                <th className="px-3 py-3 font-medium">Date</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-pat-border">
              {filtered.map((row) => {
                const isOpen = expanded === row.ID;
                return (
                  <React.Fragment key={row.ID}>
                    <tr className="hover:bg-pat-table-hover transition-colors">
                      <td className="px-3 py-3">
                        <button
                          onClick={() => setExpanded(isOpen ? null : row.ID)}
                          aria-expanded={isOpen}
                          aria-label={`Expand signal ${row.ID}`}
                          className="p-1 rounded hover:bg-pat-bg-surface-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-pat-primary"
                        >
                          {isOpen ? <IconChevronDown size={14} className="text-pat-text-muted" /> : <IconChevronRight size={14} className="text-pat-text-muted" />}
                        </button>
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
                      <td className="px-3 py-3">
                      {row.QualityGrade ? (
                        <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${
                          row.QualityGrade === "A+" ? "bg-pat-success/15 text-pat-success" :
                          row.QualityGrade === "A" ? "bg-pat-info/15 text-pat-info" :
                          row.QualityGrade === "B" ? "bg-pat-warning/15 text-pat-warning" :
                          "bg-pat-danger/15 text-pat-danger"
                        }`}>{row.QualityGrade}</span>
                      ) : <span className="text-xs text-pat-text-muted">—</span>}
                    </td>
                    <td className="px-3 py-3"><StatusText status={row.Status} /></td>
                      <td className="px-3 py-3 text-[10px] text-pat-text-muted">{row.Regime || "—"}</td>
                      <td className="px-3 py-3 text-xs text-pat-text-muted">
                        {row.CreatedAt && row.CreatedAt !== "0001-01-01T00:00:00Z" ? format(new Date(row.CreatedAt), "MMM d, HH:mm:ss") : "—"}
                      </td>
                    </tr>
                    {isOpen && (
                      <tr className="bg-pat-bg-surface-secondary/30">
                        <td colSpan={13} className="px-4 py-4 space-y-3">
                          <div className="flex flex-wrap gap-4 text-xs text-pat-text-secondary">
                            <span title="Engine-recommended lot (risk-capped, margin-aware)">
                              Lot: <b className="text-pat-text-primary">{num(row.SuggestedLot) > 0 ? Number(row.SuggestedLot).toFixed(2) : "—"}</b>
                            </span>
                            <span title="Risk at stop distance, USD">
                              Risk: <b className="text-pat-text-primary">{num(row.RiskDollars) > 0 ? `$${Number(row.RiskDollars).toFixed(2)}` : "—"}</b>
                            </span>
                            <span title="Risk as % of account equity">
                              Equity %: <b className="text-pat-text-primary">{num(row.RiskPctOfEquity) > 0 ? `${Number(row.RiskPctOfEquity).toFixed(2)}%` : "—"}</b>
                            </span>
                            <span title="Stop distance in points">
                              SL pts: <b className="text-pat-text-primary">{num(row.SLDistancePoints) > 0 ? Number(row.SLDistancePoints).toFixed(0) : "—"}</b>
                            </span>
                          </div>
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
