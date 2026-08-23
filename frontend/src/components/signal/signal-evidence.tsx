"use client";
import { IconChevronUp, IconShieldCheck, IconBrain, IconListDetails, IconTarget } from "@tabler/icons-react";

/** Loose source shape — both admin (DiagnosticRecord[]) and user (PillarContributions) shapes are accepted. */
export interface SignalEvidenceSource {
  ID?: string;
  Symbol?: string;
  StrategyID?: string;
  Direction?: string;
  Regime?: string;
  Session?: string;
  Timeframe?: string;
  Grade?: string;
  RawScore?: string;
  CalibratedProbability?: string;
  Status?: string;
  EntryPrice?: string;
  StopLoss?: string;
  TP1?: string;
  TP2?: string;
  TP3?: string;
  ReasonCodes?: string[] | null;
  Evidence?: unknown;
  GateResults?: unknown[] | null;
  PillarContributions?: Record<string, number>;
  AiVerification?: string;
  RiskDecision?: string;
  Executable?: boolean;
  CreatedAt?: string;
  ExpiresAt?: string;
}

interface Pillar { name: string; contribution: number; direction: "BUY" | "SELL" | "NEUTRAL"; }
interface Gate { id: string; result: string; reason: string; }
interface Diag { key: string; value: string; }

function titleCase(s: string): string {
  return s.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function num(v: unknown): number {
  const n = parseFloat(typeof v === "string" ? v : String(v ?? "0"));
  return isNaN(n) ? 0 : n;
}

function getPillars(sig: SignalEvidenceSource): Pillar[] {
  const out: Pillar[] = [];
  if (Array.isArray(sig.Evidence)) {
    for (const e of sig.Evidence as Record<string, unknown>[]) {
      const name = String(e.Pillar || e.pillar || e.Feature || e.feature || "Factor");
      const c = num(e.Contribution ?? e.contribution ?? 0);
      const d = String(e.Direction || e.direction || "").toUpperCase();
      out.push({ name, contribution: c, direction: d === "BUY" ? "BUY" : d === "SELL" ? "SELL" : c >= 0 ? "BUY" : "SELL" });
    }
  } else if (sig.PillarContributions && typeof sig.PillarContributions === "object") {
    for (const [k, v] of Object.entries(sig.PillarContributions)) {
      const c = Number(v) || 0;
      out.push({ name: k, contribution: c, direction: c >= 0 ? "BUY" : "SELL" });
    }
  }
  return out.sort((a, b) => Math.abs(b.contribution) - Math.abs(a.contribution));
}

function getGates(sig: SignalEvidenceSource): Gate[] {
  const out: Gate[] = [];
  if (Array.isArray(sig.GateResults)) {
    for (const g of sig.GateResults as Record<string, unknown>[]) {
      const rc = g.reason_codes ?? g.reasonCodes;
      out.push({
        id: String(g.gate_id || g.GateID || "Gate"),
        result: String(g.result || g.Result || "—").toUpperCase(),
        reason: Array.isArray(rc) ? rc.join(", ") : String(rc || ""),
      });
    }
  }
  return out;
}

function getDiagnostics(sig: SignalEvidenceSource): Diag[] {
  const out: Diag[] = [];
  const ev = sig.Evidence;
  if (ev && !Array.isArray(ev) && typeof ev === "object") {
    for (const [k, v] of Object.entries(ev as Record<string, unknown>)) {
      if (k.toLowerCase() === "pillar" || k.toLowerCase() === "contribution" || k.toLowerCase() === "direction" || k.toLowerCase() === "feature") continue;
      out.push({ key: titleCase(k), value: typeof v === "object" ? JSON.stringify(v) : String(v) });
    }
  } else if (Array.isArray(ev) && ev.every((x) => typeof x === "string")) {
    for (const s of ev as string[]) out.push({ key: "Note", value: s });
  }
  return out;
}

const DIR_STYLES: Record<string, { text: string; chip: string; bar: string; label: string }> = {
  BUY: { text: "text-pat-success", chip: "bg-pat-success/10 text-pat-success border border-pat-success/20", bar: "bg-pat-success", label: "Bullish" },
  SELL: { text: "text-pat-danger", chip: "bg-pat-danger/10 text-pat-danger border border-pat-danger/20", bar: "bg-pat-danger", label: "Bearish" },
  NEUTRAL: { text: "text-pat-text-secondary", chip: "bg-pat-bg-surface-secondary text-pat-text-secondary border border-pat-border", bar: "bg-pat-text-muted", label: "Neutral" },
};

function dirClass(d: string): string {
  return DIR_STYLES[d] ? d : "NEUTRAL";
}

function SectionTitle({ icon, title }: { icon?: React.ReactNode; title: string }) {
  return (
    <div className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-pat-text-secondary mt-4 mb-2">
      {icon}
      <span>{title}</span>
    </div>
  );
}

export default function SignalEvidencePanel({ sig }: { sig: SignalEvidenceSource }) {
  const pillars = getPillars(sig);
  const gates = getGates(sig);
  const diagnostics = getDiagnostics(sig);
  const reasonCodes = sig.ReasonCodes ?? [];
  const maxAbs = Math.max(0.0001, ...pillars.map((p) => Math.abs(p.contribution)));
  const pct = (v: number) => Math.max(2, Math.min(100, (Math.abs(v) / maxAbs) * 100));

  const dir = sig.Direction || "NO-TRADE";
  const dirText =
    dir === "BUY" ? "text-pat-success" : dir === "SELL" ? "text-pat-danger"
      : dir === "BUY_CANDIDATE" ? "text-pat-warning" : dir === "SELL_CANDIDATE" ? "text-pat-candidate-sell"
        : "text-pat-text-secondary";

  const prob = num(sig.CalibratedProbability);
  const score = num(sig.RawScore);

  const level = (label: string, val: string | undefined, cls: string) => (
    <div className="flex flex-col">
      <span className="text-[10px] text-pat-text-muted">{label}</span>
      <span className={`text-sm font-semibold ${cls}`}>{num(val) > 0 ? num(val).toFixed(2) : "—"}</span>
    </div>
  );

  return (
    <div className="space-y-1">
      {/* Header */}
      <div className="flex flex-wrap items-center gap-3">
        <span className={`text-base font-bold ${dirText}`}>{dir}</span>
        <span className="text-xs text-pat-text-secondary">{sig.StrategyID?.replace(/_/g, " ")}</span>
        <span className="text-xs text-pat-text-muted">{sig.Symbol || "XAUUSD"}</span>
        <span className="ml-auto flex items-center gap-3 text-xs">
          <span><span className="text-pat-text-muted">Score </span><span className="text-pat-text-primary font-medium">{score > 0 ? score.toFixed(1) : "—"}</span></span>
          <span><span className="text-pat-text-muted">Prob </span><span className="text-pat-text-primary font-medium">{prob > 0 ? `${(prob * 100).toFixed(1)}%` : "Pending"}</span></span>
          <span className="px-2 py-0.5 rounded-full bg-pat-bg-surface-secondary text-pat-text-secondary border border-pat-border">{sig.Status || "—"}</span>
        </span>
      </div>

      {/* Trade levels */}
      <div className="flex flex-wrap gap-4 rounded-lg border border-pat-border bg-pat-bg-surface-secondary/30 px-3 py-2">
        {level("Entry", sig.EntryPrice, "text-pat-text-primary")}
        {level("Stop Loss", sig.StopLoss, "text-pat-danger")}
        {level("TP1", sig.TP1, "text-pat-success")}
        {level("TP2", sig.TP2, "text-pat-success")}
        {level("TP3", sig.TP3, "text-pat-success")}
        <div className="flex flex-col">
          <span className="text-[10px] text-pat-text-muted">Regime / Session</span>
          <span className="text-sm font-medium text-pat-text-primary">{sig.Regime || "—"} <span className="text-pat-text-muted">/ {sig.Session || "—"}</span></span>
        </div>
      </div>

      {/* Decision rationale */}
      {reasonCodes.length > 0 && (
        <>
          <SectionTitle title="Decision rationale" />
          <div className="flex flex-wrap gap-1.5">
            {reasonCodes.map((rc, i) => (
              <span key={i} className="text-[10px] px-2 py-0.5 rounded-full bg-pat-danger/10 text-pat-danger border border-pat-danger/20">{rc}</span>
            ))}
          </div>
        </>
      )}

      {/* Gate results */}
      {gates.length > 0 && (
        <>
          <SectionTitle icon={<IconShieldCheck size={14} />} title="Hard gates" />
          <div className="space-y-1">
            {gates.map((g, i) => {
              const pass = g.result === "PASS";
              const veto = g.result === "VETO" || g.result === "FAIL";
              return (
                <div key={i} className="flex items-center gap-2 text-xs">
                  <span className={`w-14 shrink-0 text-[10px] font-semibold px-1.5 py-0.5 rounded-full text-center ${pass ? "bg-pat-success/10 text-pat-success" : veto ? "bg-pat-danger/10 text-pat-danger" : "bg-pat-bg-surface-secondary text-pat-text-secondary"}`}>{g.result}</span>
                  <span className="text-pat-text-secondary w-40 truncate">{g.id}</span>
                  <span className="text-pat-text-muted truncate">{g.reason || "—"}</span>
                </div>
              );
            })}
          </div>
        </>
      )}

      {/* Evidence pillars */}
      {pillars.length > 0 && (
        <>
          <SectionTitle icon={<IconListDetails size={14} />} title="Evidence pillars" />
          <div className="space-y-1.5">
            {pillars.map((p, i) => {
              const d = dirClass(p.direction);
              return (
                <div key={i} className="flex items-center gap-3">
                  <span className="w-36 shrink-0 truncate text-xs text-pat-text-secondary" title={p.name}>{p.name}</span>
                  <div className="flex-1 h-2 rounded bg-pat-bg-surface-secondary overflow-hidden">
                    <div className={`h-full ${DIR_STYLES[d].bar}`} style={{ width: `${pct(p.contribution)}%` }} />
                  </div>
                  <span className={`w-16 shrink-0 text-right text-xs font-medium ${DIR_STYLES[d].text}`}>{p.contribution.toFixed(2)}</span>
                  <span className={`w-16 shrink-0 text-[10px] text-right ${DIR_STYLES[d].text}`}>{DIR_STYLES[d].label}</span>
                </div>
              );
            })}
          </div>
        </>
      )}

      {/* Verification & diagnostics */}
      <SectionTitle icon={<IconBrain size={14} />} title="Verification & diagnostics" />
      <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-xs md:grid-cols-3">
        <KV k="AI Verification" v={sig.AiVerification} />
        <KV k="Risk Decision" v={sig.RiskDecision} />
        <KV k="Grade" v={sig.Grade} />
        <KV k="Executable" v={sig.Executable === undefined ? undefined : sig.Executable ? "Yes" : "No"} />
        <KV k="Timeframe" v={sig.Timeframe} />
      </div>
      {diagnostics.length > 0 && (
        <div className="mt-2 grid grid-cols-1 gap-x-6 gap-y-1 text-xs sm:grid-cols-2">
          {diagnostics.map((d, i) => (
            <div key={i} className="flex items-center justify-between gap-2 border-b border-pat-border/40 pb-0.5">
              <span className="text-pat-text-muted truncate">{d.key}</span>
              <span className="text-pat-text-primary truncate text-right">{d.value}</span>
            </div>
          ))}
        </div>
      )}

      {pillars.length === 0 && gates.length === 0 && diagnostics.length === 0 && reasonCodes.length === 0 && (
        <div className="text-xs text-pat-text-muted mt-2">No detailed evidence available for this signal.</div>
      )}
    </div>
  );
}

function KV({ k, v }: { k: string; v?: string }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="text-pat-text-muted">{k}</span>
      <span className="text-pat-text-primary truncate text-right">{v && v !== "" ? v : "N/A"}</span>
    </div>
  );
}
