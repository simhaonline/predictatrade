"use client";
// ATEN — Aetherial Technical Engine Node
// Astro-Financial Intelligence Engine (Vedic DI + Western Tropical)
// ELITE-plan exclusive. Signal screens, interactive mind map, full factor breakdown.
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { customInstance } from "@/lib/axios-instance";
import { toast } from "sonner";
import { IconSparkles, IconRefresh, IconAlertTriangle, IconStar, IconChevronDown, IconChevronRight } from "@tabler/icons-react";
import StatusBadge from "@/components/ui/status-badge";
import { strategyLabel } from "@/lib/strategy-labels";

interface AstroFactor {
  label: string;
  score: number;
  detail?: string;
  weight: number;
}

interface AstroState {
  timestamp: string;
  composite_score: number;
  confidence: number;
  eligible_for_trade: boolean;
  ineligible_reason?: string;
  market_closed: boolean;
  vedic: {
    nakshatra_name: string;
    pada: number;
    nakshatra_bias: number;
    hora_lord: string;
    hora_bias: number;
    dasha_l1: string;
    dasha_l1_bias: number;
    dasha_l2: string;
    dasha_l2_bias: number;
    dasha_progress_pct: number;
    contamination?: string[];
    eclipse: boolean;
    eclipse_type?: string;
  };
  western: {
    total_score: number;
    is_retrograde: Record<string, boolean>;
    aspects: { planet_a: string; planet_b: string; type: string; orb: number; bias: number }[];
  };
  apocalypse: { code: string; severity: number; action: string } | null;
  factors: Record<string, AstroFactor>;
  note?: string;
}

interface MindMapNode {
  name: string;
  children?: MindMapNode[];
  value?: string | number | boolean;
  score?: number;
  count?: number;
}

export default function AtenPage() {
  const [tab, setTab] = useState<"overview" | "mindmap" | "screens">("overview");
  const [expanded, setExpanded] = useState<Record<string, boolean>>({ vedic: true, western: true, factors: true });
  const q = useQuery({
    queryKey: ["astro-state-aten"],
    queryFn: async () => (await customInstance.get("/astro/state")).data as AstroState,
    refetchInterval: 60_000,
  });
  const mmQ = useQuery({
    queryKey: ["astro-mindmap"],
    queryFn: async () => (await customInstance.get("/astro/mindmap")).data as MindMapNode,
    enabled: tab === "mindmap",
  });
  const screensQ = useQuery({
    queryKey: ["astro-screens"],
    queryFn: async () => (await customInstance.get("/astro/screens")).data,
    enabled: tab === "screens",
  });
  const st = q.data;

  const scoreColor = (v?: number) => (v === undefined || v === null) ? "" : v >= 30 ? "text-pat-success" : v <= -30 ? "text-pat-danger" : "text-pat-text-secondary";

  const renderNode = (node: MindMapNode, depth: number) => (
    <div key={node.name + depth} className={depth > 0 ? "border-l-2 border-pat-border pl-4 ml-2 mt-2" : ""}>
      <div className="flex items-center justify-between rounded bg-pat-bg-surface-secondary/40 px-3 py-2">
        <span className={depth === 0 ? "text-sm font-bold text-pat-text-primary" : "text-xs text-pat-text-primary"}>
          {depth === 0 && <IconStar size={14} className="inline mr-2 text-pat-accent" />}
          {node.name}
        </span>
        <div className="flex gap-2 items-center">
          {node.value !== undefined && <span className="text-xs text-pat-text-muted">{String(node.value)}</span>}
          {node.score !== undefined && <span className={`text-xs font-medium ${scoreColor(node.score)}`}>{node.score > 0 ? "+" : ""}{typeof node.score === "number" ? node.score.toFixed(1) : node.score}</span>}
        </div>
      </div>
      {node.children && expanded[node.name] !== false && node.children.map((child) => renderNode(child, depth + 1))}
      {node.children && node.children.length > 0 && (
        <button onClick={() => setExpanded(e => ({ ...e, [node.name]: !e[node.name] }))} className="text-xs text-pat-info mt-1">
          {expanded[node.name] === false ? <IconChevronRight size={12} className="inline" /> : <IconChevronDown size={12} className="inline" />}
          {expanded[node.name] === false ? " expand" : " collapse"}
        </button>
      )}
    </div>
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary flex items-center gap-2">
          <IconSparkles size={22} className="text-pat-accent" />
          ATEN — Aetherial Technical Engine Node
        </h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Vedic Jyotish + Western Tropical Astro-Financial Intelligence.
          Nakshatra bias, hora cadence, Vimshottari dasha, Shadbala strength, Gold natal transit aspects.
          <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-pat-accent/15 text-pat-accent font-medium">ELITE only</span>
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1.5">
        {(["overview", "mindmap", "screens"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={`rounded px-4 py-1.5 text-xs capitalize ${tab === t ? "bg-primary text-primary-foreground" : "border border-pat-border text-pat-text-secondary hover:bg-pat-bg-surface"}`}>
            {t === "mindmap" ? "Interactive Mind Map" : t === "screens" ? "Signal Screens" : t}
          </button>
        ))}
      </div>

      {q.isLoading && <div className="text-sm text-pat-text-muted">Computing ASTRO state…</div>}
      {q.isError && <div className="text-sm text-pat-danger">Failed to load ATEN state: {(q.error as any)?.message}</div>}

      {st && tab === "overview" && (
        <div className="space-y-4">
          {/* Score + eligibility status */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">ATEN Composite</div>
              <div className={`text-3xl font-mono font-bold ${st.composite_score >= 0 ? "text-pat-success" : "text-pat-danger"}`}>
                {st.composite_score > 0 ? "+" : ""}{st.composite_score.toFixed(1)}
              </div>
              <div className="text-[10px] text-pat-text-muted">of ±100 · 5th intelligence engine</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">Nakshatra</div>
              <div className="text-lg font-bold text-pat-text-primary">{st.vedic.nakshatra_name}</div>
              <div className="text-[10px] text-pat-text-secondary">Pada {st.vedic.pada} · {st.vedic.nakshatra_bias > 0 ? "+" : ""}{st.vedic.nakshatra_bias} bias</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">Hora</div>
              <div className="text-lg font-bold text-pat-text-primary">{st.vedic.hora_lord}</div>
              <div className="text-[10px] text-pat-text-secondary">{st.vedic.hora_bias > 0 ? "+" : ""}{st.vedic.hora_bias} bias</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">Vimshottari Dasha</div>
              <div className="text-lg font-bold text-pat-text-primary">{st.vedic.dasha_l1} / {st.vedic.dasha_l2}</div>
              <div className="text-[10px] text-pat-text-secondary">{st.vedic.dasha_progress_pct.toFixed(0)}% progressed · {st.vedic.dasha_l1_bias > 0 ? "+" : ""}{st.vedic.dasha_l1_bias}</div>
            </div>
          </div>

          {/* Market closed / eligibility notice */}
          {st.market_closed && (
            <div role="status" className="rounded-lg border border-pat-warning/40 bg-pat-warning/5 px-4 py-2 text-xs text-pat-text-secondary">
              🕒 Market closed — ATEN state is reference-only; no signals until re-open.
            </div>
          )}
          {!st.eligible_for_trade && !st.market_closed && st.ineligible_reason && (
            <div role="alert" className="rounded-lg border border-pat-danger/40 bg-pat-danger/5 px-4 py-2 text-xs text-pat-danger flex items-center gap-2">
              <IconAlertTriangle size={14} /> ATEN signal suppressed — {st.ineligible_reason}
              {st.apocalypse && <span className="ml-1">({st.apocalypse.code}, severity {st.apocalypse.severity})</span>}
            </div>
          )}

          {/* Western aspects table */}
          <div className="rounded-lg border border-pat-border overflow-hidden">
            <div className="bg-pat-bg-surface px-4 py-2 text-sm font-medium text-pat-text-primary border-b border-pat-border flex items-center gap-2">
              <IconStar size={16} className="text-pat-accent" /> Western Tropical Aspects (Gold Natal)
            </div>
            {st.western.aspects.length === 0 ? (
              <div className="px-4 py-4 text-xs text-pat-text-muted">No major aspects within orb right now.</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border bg-pat-bg-surface">
                      <th className="px-3 py-2 font-medium">Transit</th>
                      <th className="px-3 py-2 font-medium">Natal</th>
                      <th className="px-3 py-2 font-medium">Aspect</th>
                      <th className="px-3 py-2 font-medium">Orb</th>
                      <th className="px-3 py-2 font-medium">Bias</th>
                    </tr>
                  </thead>
                  <tbody>
                    {st.western.aspects.map((a, i) => (
                      <tr key={i} className="border-b border-pat-border/50">
                        <td className="px-3 py-2 text-pat-text-primary">{a.planet_a}</td>
                        <td className="px-3 py-2 text-pat-text-secondary">{a.planet_b}</td>
                        <td className="px-3 py-2 text-pat-text-secondary capitalize">{a.type}</td>
                        <td className="px-3 py-2 font-mono text-pat-text-secondary">{a.orb.toFixed(1)}°</td>
                        <td className={`px-3 py-2 font-medium ${a.bias >= 0 ? "text-pat-success" : "text-pat-danger"}`}>{a.bias > 0 ? "+" : ""}{a.bias}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Mind Map */}
      {st && tab === "mindmap" && st && (
        <div className="rounded-lg border border-pat-border p-4">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
            <IconSparkles size={14} className="text-pat-accent" /> ATEN Interactive Mind Map
            <span className="text-xs font-normal text-pat-text-muted ml-3">click ▼/▶ to expand/collapse</span>
          </h2>
          {mmQ.isLoading && <div className="text-xs text-pat-text-muted">Building mind map…</div>}
          {mmQ.data && (
            <div className="rounded-lg bg-pat-bg-surface border border-pat-border p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-base font-bold text-pat-text-primary flex items-center gap-2">
                  <IconSparkles size={16} className="text-pat-accent" /> ATEN
                </span>
                <span className={`text-xl font-mono font-bold ${Number(mmQ.data?.score ?? 0) >= 0 ? "text-pat-success" : "text-pat-danger"}`}>
                  {Number(mmQ.data?.score ?? 0).toFixed(1)}
                </span>
              </div>
              {(mmQ.data.children ?? []).map((child) => (
                <div key={child.name} className="ml-2 mt-3">
                  <button onClick={() => setExpanded(e => ({ ...e, [child.name]: e[child.name] === false }))}
                    className="text-sm text-pat-text-primary font-medium flex items-center gap-2">
                    {expanded[child.name] === false ? <IconChevronRight size={14} /> : <IconChevronDown size={14} />}
                    {child.name}
                  </button>
                  {expanded[child.name] !== false && (child.children ?? []).map((c) => (
                    <div key={c.name} className="ml-4 border-l-2 border-pat-border/50 pl-3 mt-1.5">
                      <div className="flex items-center justify-between rounded bg-pat-bg-surface-secondary/30 px-2 py-1.5">
                        <span className="text-xs text-pat-text-primary">{c.name}</span>
                        <div className="flex gap-3 items-center">
                          {c.value !== undefined && <span className="text-xs text-pat-text-secondary">{String(c.value)}</span>}
                          {c.score !== undefined && <span className={`text-xs font-medium ${scoreColor(c.score)}`}>{typeof c.score === "number" ? (c.score > 0 ? "+" : "") + c.score.toFixed(1) : c.score}</span>}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Signal Screens */}
      {st && tab === "screens" && (
        <div className="space-y-3">
          <div className="rounded-lg border border-pat-border bg-pat-card-bg p-4">
            <h2 className="text-sm font-medium text-pat-text-primary mb-2">ATEN Signal Screens</h2>
            <p className="text-xs text-pat-text-secondary mb-3">Per-factor breakdown for decision transparency (check.md signal screens requirement). Every factor is shown with its weight and score — no hidden computation.</p>
            {screensQ.isLoading && <div className="text-xs text-pat-text-muted">Loading screens…</div>}
            {screensQ.data && (
              <div className="space-y-2">
                <div className="flex justify-between items-center rounded bg-pat-bg-surface-secondary px-3 py-2">
                  <span className="text-sm font-bold text-pat-text-primary">ATEN Composite Score</span>
                  <span className={`text-base font-bold font-mono ${scoreColor(screensQ.data.composite_score)}`}>
                    {screensQ.data.composite_score > 0 ? "+" : ""}{screensQ.data.composite_score.toFixed(1)}
                  </span>
                </div>
                {(screensQ.data.factors ?? []).map((f: any, i: number) => (
                  <div key={i} className="flex items-center justify-between rounded border border-pat-border/40 px-3 py-2">
                    <div>
                      <div className="text-sm text-pat-text-primary">{f.label}</div>
                      {f.detail && <div className="text-xs text-pat-text-muted">{f.detail}</div>}
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-pat-text-muted">weight {f.weight}</span>
                      <span className={`text-sm font-bold font-mono ${f.score >= 0 ? "text-pat-success" : "text-pat-danger"}`}>{f.score > 0 ? "+" : ""}{f.score.toFixed(1)}</span>
                    </div>
                  </div>
                ))}
                {screensQ.data.ineligible_reason && (
                  <div className="rounded border border-pat-danger/30 bg-pat-danger/5 px-3 py-2 text-xs text-pat-danger">
                    ASTRO signal suppressed — {screensQ.data.ineligible_reason}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
