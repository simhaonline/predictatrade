"use client";
// Astro Intelligence — Vedic + Western Astro-Financial Intelligence Engine
// (check.md 2026-08-30 v5.0). ELITE-plan exclusive; enforced upstream.
// Full state, interactive mind-map and per-factor signal screens.
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { customInstance } from "@/lib/axios-instance";
import { IconSparkles, IconRefresh, IconAlertTriangle, IconStar } from "@tabler/icons-react";

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
    lunar_phase?: string;
  };
  apocalypse?: { code: string; severity: number; action: string };
  note?: string;
  factors: Record<string, AstroFactor>;
}

function scoreColor(v: number): string {
  if (v >= 30) return "text-pat-success";
  if (v <= -30) return "text-pat-danger";
  return "text-pat-text-secondary";
}

export default function AstroPanel() {
  const [tab, setTab] = useState<"overview" | "mindmap" | "screens">("overview");
  const q = useQuery({
    queryKey: ["astro-state"],
    queryFn: async () => (await customInstance.get("/astro/state")).data as AstroState,
    refetchInterval: 60_000,
  });
  const st = q.data;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary flex items-center gap-2">
          <IconSparkles size={20} className="text-pat-info" /> Astro-Financial Intelligence
        </h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Vedic Jyotish (sidereal) + Western Tropical composite — Shadbala-weighted planetary
          strength, nakshatra bias, hora cadence, Vimshottari dasha and Gold natal transit aspects.
        </p>
        {st?.market_closed && (
          <div role="status" className="mt-3 rounded-lg border border-pat-warning/40 bg-pat-warning/5 px-4 py-2 text-xs text-pat-text-secondary">
            🕒 Market closed — ASTRO state is reference-only; no signals until re-open.
          </div>
        )}
        {st && !st.eligible_for_trade && !st.market_closed && st.ineligible_reason && (
          <div role="alert" className="mt-3 rounded-lg border border-pat-danger/40 bg-pat-danger/5 px-4 py-2 text-xs text-pat-danger">
            ⚠️ ASTRO signal suppressed — {st.ineligible_reason}
            {st.apocalypse && ` (apocalypse: ${st.apocalypse.code}, severity ${st.apocalypse.severity})`}
          </div>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-2">
        {(["overview", "mindmap", "screens"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={`rounded px-3 py-1.5 text-xs capitalize ${tab === t ? "bg-primary text-primary-foreground" : "border border-pat-border text-pat-text-secondary"}`}>
            {t === "mindmap" ? "Interactive Mind Map" : t}
          </button>
        ))}
      </div>

      {q.isLoading && <div className="text-sm text-pat-text-muted">Computing ASTRO state…</div>}
      {q.isError && <div className="text-sm text-pat-danger">Failed to load: {(q.error as any)?.message}</div>}
      {!st && !q.isLoading && !q.isLoading && null}

      {st && tab === "overview" && (
        <div className="space-y-4">
          {/* Composite */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">ASTRO Composite</div>
              <div className={`text-3xl font-mono font-bold ${st.composite_score >= 0 ? "text-pat-success" : "text-pat-danger"}`}>
                {st.composite_score.toFixed(1)}
              </div>
              <div className="text-[10px] text-pat-text-muted">of ±100</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">Nakshatra</div>
              <div className="text-lg font-bold text-pat-text-primary">{st.vedic.nakshatra_name}</div>
              <div className="text-[10px] text-pat-text-secondary">Pada {st.vedic.pada} · bias {st.vedic.nakshatra_bias > 0 ? "+" : ""}{st.vedic.nakshatra_bias}</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">Hora</div>
              <div className="text-lg font-bold text-pat-text-primary">{st.vedic.hora_lord}</div>
              <div className="text-[10px] text-pat-text-secondary">{st.vedic.hora_bias > 0 ? "+" : ""}{st.vedic.hora_bias} bias</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4">
              <div className="text-xs text-pat-text-muted uppercase mb-1">Dasha</div>
              <div className="text-lg font-bold text-pat-text-primary">{st.vedic.dasha_l1}</div>
              <div className="text-[10px] text-pat-text-secondary">L2: {st.vedic.dasha_l2} · {st.vedic.dasha_progress_pct.toFixed(0)}% progressed</div>
            </div>
          </div>

          {/* Factors */}
          <div className="rounded-lg border border-pat-border p-4">
            <h2 className="text-sm font-medium text-pat-text-primary mb-3">ASTRO Factor Decomposition</h2>
            <div className="space-y-2">
              {Object.entries(st.factors).map(([key, f]) => (
                <div key={key} className="flex items-center justify-between rounded bg-pat-bg-surface-secondary px-3 py-2">
                  <div>
                    <div className="text-xs text-pat-text-primary font-medium">{f.label}</div>
                    <div className="text-[10px] text-pat-text-muted">{f.detail}</div>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="w-16 bg-pat-bg-surface rounded h-1.5">
                      <div className="rounded h-1.5 bg-pat-primary" style={{ width: `${f.weight * 100}%` }} />
                    </div>
                    <span className={`text-xs font-medium w-16 text-right ${f.score >= 0 ? "text-pat-success" : "text-pat-danger"}`}>
                      {f.score > 0 ? "+" : ""}{f.score.toFixed(1)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Vedic detail */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-3">
              <div className="text-xs text-pat-text-muted mb-1">Dasha L2</div>
              <div className="text-sm text-pat-text-primary">{st.vedic.dasha_l2}</div>
              <div className="text-[10px] text-pat-text-secondary">{st.vedic.dasha_l2_bias > 0 ? "+" : ""}{st.vedic.dasha_l2_bias}</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-3">
              <div className="text-xs text-pat-text-muted mb-1">Eclipse</div>
              <div className="text-sm text-pat-text-primary">{st.vedic.eclipse ? `Yes — ${st.vedic.eclipse_type}` : "No"}</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-3">
              <div className="text-xs text-pat-text-muted mb-1">Western Score</div>
              <div className="text-sm text-pat-text-primary">{st.western.total_score.toFixed(1)}</div>
            </div>
            <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-3">
              <div className="text-xs text-pat-text-muted mb-1">Mercury Rx</div>
              <div className="text-sm text-pat-text-primary">{st.western.is_retrograde?.["Mercury"] ? "Yes" : "No"}</div>
            </div>
          </div>
        </div>
      )}

      {st && tab === "mindmap" && <AstroMindMap st={st} />}

      {st && tab === "screens" && (
        <div className="space-y-3 rounded-lg border border-pat-border p-4">
          <h2 className="text-sm font-medium text-pat-text-primary mb-2">Signal Screens</h2>
          {Object.entries(st.factors).map(([k, f]) => (
            <div key={k} className="flex justify-between border-b border-pat-border/50 pb-2">
              <div>
                <div className="text-sm text-pat-text-primary">{f.label}</div>
                {f.detail && <div className="text-xs text-pat-text-muted">{f.detail}</div>}
              </div>
              <div className={`font-bold text-sm ${f.score >= 0 ? "text-pat-success" : "text-pat-danger"}`}>
                {f.score > 0 ? "+" : ""}{f.score.toFixed(1)}
              </div>
            </div>
          ))}
        </div>
      )}

      {st && (
        <div className="flex items-center gap-2 text-xs text-pat-text-muted">
          <IconRefresh size={14} /> Refreshes every 60s · computed at {new Date(st.timestamp).toLocaleTimeString()}
        </div>
      )}
    </div>
  );
}

// Interactive Mind Map — collapsable tree of all factors + scores
function AstroMindMap({ st }: { st: AstroState }) {
  const [expanded, setExpanded] = useState<Record<string, boolean>>({ vedic: true, western: true });
  const node = (name: string, value?: string | number | boolean, score?: number) => (
    <div className="flex items-center justify-between py-1 px-2 rounded bg-pat-bg-surface-secondary/40">
      <span className="text-xs text-pat-text-primary">{name}</span>
      {value !== undefined && <span className="text-xs text-pat-text-muted">{String(value)}</span>}
      {score !== undefined && <span className={`text-xs font-medium ${score >= 0 ? "text-pat-success" : "text-pat-danger"}`}>{score > 0 ? "+" : ""}{score}</span>}
    </div>
  );
  return (
    <div className="rounded-lg border border-pat-border p-4">
      <h2 className="text-sm font-medium text-pat-text-primary mb-2 flex items-center gap-2">
        <IconStar size={14} /> Astro Mind Map
        <button onClick={() => setExpanded({})} className="ml-auto text-xs text-pat-info">↻ Reset</button>
      </h2>
      <div className="space-y-2">
        <div className="rounded-lg bg-pat-bg-surface border border-pat-border p-3">
          <div className="flex items-center justify-between">
            <span className="text-base font-bold text-pat-text-primary">✦ ASTRO</span>
            <span className={`text-lg font-bold ${st.composite_score >= 0 ? "text-pat-success" : "text-pat-danger"}`}>
              {st.composite_score.toFixed(1)}
            </span>
          </div>
          <p className="text-xs text-pat-text-muted mt-1">{st.note || "Live ASTRO composite"}</p>
        </div>

        {Object.entries({ vedic: "Vedic DI", western: "Western Tropical" } as const).map(([key, label]) => (
          <div key={key} className="border-l-2 border-pat-border pl-4">
            <button onClick={() => setExpanded(e => ({ ...e, [key]: !e[key] }))}
              className="text-sm text-pat-text-primary font-medium flex items-center gap-2 mt-3">
              <span className="text-pat-text-muted">{expanded[key] ? "▼" : "▶"}</span> {label}
            </button>
            {expanded[key] && key === "vedic" && (
              <div className="space-y-1 mt-2">
                {node("Nakshatra", `${st.vedic.nakshatra_name} pada ${st.vedic.pada}`, st.vedic.nakshatra_bias)}
                {node("Hora Lord", st.vedic.hora_lord, st.vedic.hora_bias)}
                {node("Dasha L1", st.vedic.dasha_l1, st.vedic.dasha_l1_bias)}
                {node("Dasha L2", st.vedic.dasha_l2, st.vedic.dasha_l2_bias)}
                {st.vedic.eclipse && node("Eclipse Window", st.vedic.eclipse_type)}
                {st.vedic.contamination?.map((c) => node("Contamination", c))}
              </div>
            )}
            {expanded[key] && key === "western" && (
              <div className="space-y-1 mt-2">
                {node("Western Total", st.western.total_score.toFixed(1))}
                {st.western.aspects.slice(0, 8).map((a, i) => node(a.planet_a + " → " + a.planet_b, a.type, a.bias))}
                {node("Mercury Retrograde", st.western.is_retrograde?.["Mercury"] ? "true" : "false")}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}