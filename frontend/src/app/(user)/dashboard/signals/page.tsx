"use client";
import { useQuery } from "@tanstack/react-query";
import React from "react";
import { customInstance } from "@/lib/axios-instance";
import { format } from "date-fns";
import { useState, useEffect, useMemo } from "react";
import { getGlobalWs, type WsMessage } from "@/lib/websocket";
import { IconChevronRight, IconChevronDown, IconChevronLeft } from "@tabler/icons-react";
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
const PAGE_SIZE = 15;

export default function UserSignalsPage() {
  const ws = getGlobalWs();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [filterStrategy, setFilterStrategy] = useState("ALL");
  const [filterDirection, setFilterDirection] = useState("ALL");
  const [filterRegime, setFilterRegime] = useState("ALL");
  const [page, setPage] = useState(0);

  // Server-authoritative allowed strategies (entitlements from control plane)
  const { data: entitlements } = useQuery<{ selected_strategies?: string[] }>({
    queryKey: ["user-entitlements-signals"],
    queryFn: async () => (await customInstance.get("/subscriptions/entitlements")).data,
  });
  const allowedStrategies: string[] = Array.isArray(entitlements?.selected_strategies)
    ? entitlements!.selected_strategies!
    : [];

  const { data: signalsData, isLoading, error, refetch } = useQuery<{ signals: EngineSignal[] }>({
    queryKey: ["engine-signals-user"],
    queryFn: async () => {
      const res = await customInstance.get("/signals?limit=200");
      return res.data as { signals: EngineSignal[] };
    },
    refetchInterval: 10000,
  });

  // WS is used ONLY to prompt a refresh of the canonical REST list.
  // Raw WS payloads are never rendered, so real signal fields (RawScore,
  // probabilities, evidence) are never overwritten by placeholder values.
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
  const restSignals = signalsData?.signals ?? [];
  // Fail-closed subscription gating: once entitlements are resolved, only the
  // user's entitled strategies are shown. While still loading, show all to
  // avoid flicker; once resolved with no entitlements, show none (no loophole).
  const combinedSignals = entitlements === undefined
    ? restSignals
    : allowedStrategies.length > 0
      ? restSignals.filter(s => allowedStrategies.includes(s.StrategyID))
      : [];

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

  // Reset page when filters change
  useEffect(() => { setPage(0); }, [filterStrategy, filterDirection, filterRegime]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const pagedSignals = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

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
              <div className="text-pat-text-muted text-sm mb-2">No entitled strategies</div>
              <div className="text-pat-text-muted text-xs">Your current plan does not include any strategies. Upgrade your subscription to view signals.</div>
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
        <>
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
                  <th className="px-3 py-3 font-medium">TP2</th>
                  <th className="px-3 py-3 font-medium">TP3</th>
                  <th className="px-3 py-3 font-medium">Quality</th>
                  <th className="px-3 py-3 font-medium">Status</th>
                  <th className="px-3 py-3 font-medium">Regime</th>
                  <th className="px-3 py-3 font-medium">Date</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-pat-border">
                {pagedSignals.map((row) => {
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
                        <td className="px-3 py-3 text-xs text-pat-success tabular-nums">{num(row.TP2) > 0 ? num(row.TP2).toFixed(2) : "—"}</td>
                        <td className="px-3 py-3 text-xs text-pat-success tabular-nums">{num(row.TP3) > 0 ? num(row.TP3).toFixed(2) : "—"}</td>
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
                          <td colSpan={15} className="px-4 py-4 space-y-3">
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

          {/* Pagination controls */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 text-xs">
              <button
                onClick={() => setPage(p => Math.max(0, p - 1))}
                disabled={page === 0}
                className="px-2 py-1 rounded border border-pat-border bg-pat-bg-surface hover:bg-pat-bg-surface-secondary disabled:opacity-30 disabled:cursor-not-allowed text-pat-text-secondary"
              >
                <IconChevronLeft size={14} className="inline" /> Prev
              </button>
              <span className="text-pat-text-muted">
                Page {page + 1} of {totalPages} ({filtered.length} signals)
              </span>
              <button
                onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
                disabled={page >= totalPages - 1}
                className="px-2 py-1 rounded border border-pat-border bg-pat-bg-surface hover:bg-pat-bg-surface-secondary disabled:opacity-30 disabled:cursor-not-allowed text-pat-text-secondary"
              >
                Next <IconChevronRight size={14} className="inline" />
              </button>
            </div>
          )}
        </>
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
