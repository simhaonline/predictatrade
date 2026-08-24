"use client";
import { useState, useMemo, useEffect, useRef } from "react";
import type { IndicatorLiveness, PerformanceMetric } from "@/lib/use-indicator-liveness";

interface IndicatorChartsProps {
  liveness: IndicatorLiveness[];
  history: Map<string, { time: number; value: number }[]>;
  performance: PerformanceMetric[];
}

const INDICATOR_COLORS: Record<string, string> = {
  ema9: "#16A36A", ema21: "#2563EB", ema50: "#D97706", ema100: "#D64550",
  ema200: "#7C3AED", sma50: "#0F8B8D", sma100: "#0F8B8D", sma200: "#16A36A",
  rsi: "#D97706", atr: "#7C3AED", adx: "#16A36A", macd_main: "#2563EB",
  stoch_main: "#D97706", cci: "#D64550", boll_upper: "#7C3AED",
  boll_middle: "#0F8B8D", boll_lower: "#2563EB", obv: "#D97706",
  psar: "#7C3AED", vwap: "#16A36A",
};
const FALLBACK_COLORS = ["#16A36A", "#2563EB", "#D97706", "#D64550", "#7C3AED", "#0F8B8D"];

function getColor(key: string, idx: number): string {
  return INDICATOR_COLORS[key] || FALLBACK_COLORS[idx % FALLBACK_COLORS.length];
}

/** Reusable canvas hook with devicePixelRatio scaling. */
function useCanvas(draw: (ctx: CanvasRenderingContext2D, w: number, h: number) => void, deps: unknown[]) {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const parent = canvas.parentElement;
    const w = parent ? parent.clientWidth : 600;
    const h = canvas.clientHeight || 360;
    const dpr = typeof window !== "undefined" ? window.devicePixelRatio || 1 : 1;
    canvas.width = Math.max(1, Math.floor(w * dpr));
    canvas.height = Math.max(1, Math.floor(h * dpr));
    canvas.style.width = w + "px";
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);
    draw(ctx, w, h);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return ref;
}

function EmptyChart({ message }: { message: string }) {
  return (
    <div className="flex items-center justify-center h-[300px] rounded-lg bg-pat-bg-surface-secondary/20">
      <div className="text-center">
        <p className="text-xs text-pat-text-muted max-w-xs">{message}</p>
      </div>
    </div>
  );
}

/** Multi-series line chart (no external deps). */
function CanvasLineChart({
  selectedIndicators, history, liveness,
}: { selectedIndicators: string[]; history: Map<string, { time: number; value: number }[]>; liveness: IndicatorLiveness[]; }) {
  const ref = useCanvas((ctx, w, h) => {
    const padL = 44, padR = 12, padT = 12, padB = 24;
    const plotW = w - padL - padR, plotH = h - padT - padB;
    const series = selectedIndicators
      .map((key, idx) => ({ key, color: getColor(key, idx), pts: (history.get(key) || []) as { time: number; value: number }[] }))
      .filter((s) => s.pts.length > 0);
    if (series.length === 0) return;
    let min = Infinity, max = -Infinity;
    for (const s of series) for (const p of s.pts) { if (p.value < min) min = p.value; if (p.value > max) max = p.value; }
    if (!isFinite(min) || !isFinite(max)) return;
    if (min === max) { min -= 1; max += 1; }
    const maxLen = Math.max(...series.map((s) => s.pts.length));
    const xAt = (i: number) => padL + (maxLen <= 1 ? plotW / 2 : (i / (maxLen - 1)) * plotW);
    const yAt = (v: number) => padT + plotH - ((v - min) / (max - min)) * plotH;
    // grid
    ctx.strokeStyle = "#2A3850"; ctx.lineWidth = 1;
    for (let g = 0; g <= 4; g++) {
      const y = padT + (g / 4) * plotH;
      ctx.beginPath(); ctx.moveTo(padL, y); ctx.lineTo(w - padR, y); ctx.stroke();
      const val = max - (g / 4) * (max - min);
      ctx.fillStyle = "#74829A"; ctx.font = "9px monospace";
      ctx.fillText(val.toFixed(1), 4, y + 3);
    }
    for (const s of series) {
      ctx.strokeStyle = s.color; ctx.lineWidth = 1.6; ctx.beginPath();
      s.pts.forEach((p, i) => { const x = xAt(i), y = yAt(p.value); if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y); });
      ctx.stroke();
      const last = s.pts[s.pts.length - 1];
      ctx.fillStyle = s.color; ctx.beginPath(); ctx.arc(xAt(s.pts.length - 1), yAt(last.value), 2.2, 0, Math.PI * 2); ctx.fill();
    }
  }, [selectedIndicators, history, liveness]);
  const hasData = selectedIndicators.some((k) => (history.get(k)?.length ?? 0) > 0);
  if (!hasData) return <EmptyChart message="No historical data yet. The chart populates as indicator values update in real time." />;
  return <canvas ref={ref} style={{ width: "100%", height: 360, display: "block" }} />;
}

/** Scatter chart (freq vs accuracy). */
function CanvasScatter({ points }: { points: { x: number; y: number; color: string }[] }) {
  const ref = useCanvas((ctx, w, h) => {
    const padL = 44, padR = 16, padT = 16, padB = 36;
    const plotW = w - padL - padR, plotH = h - padT - padB;
    ctx.strokeStyle = "#2A3850"; ctx.lineWidth = 1;
    for (let g = 0; g <= 4; g++) {
      const y = padT + (g / 4) * plotH; ctx.beginPath(); ctx.moveTo(padL, y); ctx.lineTo(w - padR, y); ctx.stroke();
      const val = 100 - (g / 4) * 100; ctx.fillStyle = "#74829A"; ctx.font = "9px monospace"; ctx.fillText(val.toFixed(0), 8, y + 3);
      const x = padL + (g / 4) * plotW; ctx.beginPath(); ctx.moveTo(x, padT); ctx.lineTo(x, h - padB); ctx.stroke();
      ctx.fillText((g / 4 * 100).toFixed(0), x - 6, h - padB + 12);
    }
    for (const p of points) {
      ctx.fillStyle = p.color; ctx.beginPath();
      ctx.arc(padL + (p.x / 100) * plotW, padT + (1 - p.y / 100) * plotH, 4, 0, Math.PI * 2); ctx.fill();
    }
  }, [points]);
  if (points.length === 0) return <EmptyChart message="No scatter data available. Data populates as the engine generates signals with evidence." />;
  return <canvas ref={ref} style={{ width: "100%", height: 400, display: "block" }} />;
}

/** Bar chart (distribution). */
function CanvasBar({ data, color = "#D97706" }: { data: { range: string; count: number }[]; color?: string }) {
  const ref = useCanvas((ctx, w, h) => {
    const padL = 40, padR = 12, padT = 12, padB = 40;
    const plotW = w - padL - padR, plotH = h - padT - padB;
    const max = Math.max(1, ...data.map((d) => d.count));
    const bw = plotW / data.length;
    ctx.strokeStyle = "#2A3850"; ctx.fillStyle = color;
    data.forEach((d, i) => {
      const bh = (d.count / max) * plotH;
      const x = padL + i * bw, y = padT + plotH - bh;
      ctx.fillRect(x + 2, y, Math.max(1, bw - 4), bh);
      if (i % 2 === 0) { ctx.fillStyle = "#74829A"; ctx.font = "8px monospace"; ctx.save(); ctx.translate(x + bw / 2, h - padB + 6); ctx.rotate(-Math.PI / 4); ctx.fillText(d.range, 0, 0); ctx.restore(); ctx.fillStyle = color; }
    });
    ctx.strokeStyle = "#2A3850"; ctx.beginPath(); ctx.moveTo(padL, padT + plotH); ctx.lineTo(w - padR, padT + plotH); ctx.stroke();
  }, [data, color]);
  if (data.length === 0) return <EmptyChart message="No distribution data available. Select an indicator with historical values." />;
  return <canvas ref={ref} style={{ width: "100%", height: 400, display: "block" }} />;
}

/** Radar chart (indicator performance). */
function CanvasRadar({ axes, series }: { axes: string[]; series: { name: string; color: string; values: number[] }[] }) {
  const ref = useCanvas((ctx, w, h) => {
    const cx = w / 2, cy = h / 2, R = Math.min(w, h) / 2 - 36;
    const n = axes.length;
    const angleAt = (i: number) => -Math.PI / 2 + (i / n) * Math.PI * 2;
    ctx.strokeStyle = "#2A3850"; ctx.lineWidth = 1;
    for (let r = 1; r <= 4; r++) {
      ctx.beginPath();
      for (let i = 0; i <= n; i++) { const a = angleAt(i % n), rr = (r / 4) * R; const x = cx + Math.cos(a) * rr, y = cy + Math.sin(a) * rr; if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y); }
      ctx.stroke();
    }
    ctx.fillStyle = "#94A3B8"; ctx.font = "10px monospace";
    axes.forEach((ax, i) => { const a = angleAt(i); ctx.fillText(ax, cx + Math.cos(a) * (R + 10) - 10, cy + Math.sin(a) * (R + 10)); });
    for (const s of series) {
      ctx.strokeStyle = s.color; ctx.fillStyle = s.color + "22"; ctx.lineWidth = 2; ctx.beginPath();
      s.values.forEach((v, i) => { const a = angleAt(i), rr = (Math.max(0, Math.min(100, v)) / 100) * R; const x = cx + Math.cos(a) * rr, y = cy + Math.sin(a) * rr; if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y); });
      ctx.closePath(); ctx.fill(); ctx.stroke();
    }
  }, [axes, series]);
  if (axes.length === 0 || series.length === 0) return <EmptyChart message="No radar data available. Requires signal performance metrics." />;
  return <canvas ref={ref} style={{ width: "100%", height: 400, display: "block" }} />;
}

export function IndicatorCharts({ liveness, history, performance }: IndicatorChartsProps) {
  const [selectedIndicators, setSelectedIndicators] = useState<string[]>(["rsi", "atr", "adx", "cci"]);
  const [chartType, setChartType] = useState<"value" | "scatter" | "distribution" | "radar">("value");

  const availableIndicators = liveness.filter((i) => { const h = history.get(i.key); return h && h.length > 0; });
  const toggleIndicator = (key: string) => {
    setSelectedIndicators((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key].slice(-6)));
  };

  const scatterData = useMemo(() => performance
    .filter((p) => p.signalFrequency !== null && p.signalAccuracy !== null)
    .map((p) => ({ x: p.signalFrequency!, y: p.signalAccuracy!, color: (p.signalAccuracy! > 50 ? "#16A36A" : p.signalAccuracy! > 20 ? "#D97706" : "#D64550") })), [performance]);

  const distributionData = useMemo(() => {
    if (selectedIndicators.length === 0) return [];
    const h = history.get(selectedIndicators[0]);
    if (!h || h.length === 0) return [];
    const values = h.map((p) => p.value);
    const min = Math.min(...values), max = Math.max(...values);
    const bins = 20, binSize = (max - min) / bins || 1;
    return Array.from({ length: bins }, (_, i) => ({ range: (min + i * binSize).toFixed(2), count: values.filter((v) => v >= min + i * binSize && v < min + (i + 1) * binSize).length }));
  }, [selectedIndicators, history]);

  const radarAxes = useMemo(() => [...new Set(performance.filter((p) => p.signalFrequency !== null).map((p) => p.indicatorKey))].slice(0, 6), [performance]);
  const radarSeries = useMemo(() => {
    if (radarAxes.length === 0) return [];
    const metrics = ["Frequency", "Accuracy", "Contribution"];
    return metrics.map((metric, mi) => {
      const values = radarAxes.map((ind) => {
        const vals = performance.filter((p) => p.indicatorKey === ind);
        if (metric === "Frequency") { const v = vals.reduce((s, m) => s + (m.signalFrequency ?? 0), 0) / (vals.length || 1); return v; }
        if (metric === "Accuracy") { const v2 = vals.filter((m) => m.signalAccuracy !== null); return v2.length ? v2.reduce((s, m) => s + (m.signalAccuracy ?? 0), 0) / v2.length : 0; }
        const v3 = vals.filter((m) => m.contributionScore !== null); return v3.length ? v3.reduce((s, m) => s + (m.contributionScore ?? 0), 0) / v3.length * 100 : 0;
      });
      return { name: metric, color: getColor(radarAxes[mi] || "x", mi), values };
    });
  }, [performance, radarAxes]);

  const chartTypes = [
    { id: "value" as const, label: "Value Timeline" },
    { id: "scatter" as const, label: "Freq vs Accuracy" },
    { id: "distribution" as const, label: "Distribution" },
    { id: "radar" as const, label: "Radar" },
  ];

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap gap-1.5">
            {chartTypes.map((t) => (
              <button key={t.id} onClick={() => setChartType(t.id)}
                className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${chartType === t.id ? "bg-pat-success/15 text-pat-success border border-pat-success/30" : "text-pat-text-muted hover:text-pat-text-secondary hover:bg-pat-bg-surface-secondary/50 border border-transparent"}`}>
                {t.label}
              </button>
            ))}
          </div>
          {(chartType === "value" || chartType === "distribution") && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-pat-text-muted">Indicators:</span>
              <div className="flex flex-wrap gap-1.5 max-w-lg">
                {availableIndicators.map((ind) => {
                  const isActive = selectedIndicators.includes(ind.key);
                  const colorIdx = selectedIndicators.indexOf(ind.key);
                  const color = getColor(ind.key, colorIdx);
                  return (
                    <button key={ind.key} onClick={() => toggleIndicator(ind.key)}
                      className={`px-2 py-0.5 text-[11px] rounded-md border transition-all ${isActive ? "text-pat-text-primary" : "text-pat-text-muted border-transparent hover:border-pat-border/50"}`}
                      style={isActive ? { backgroundColor: `${color}20`, borderColor: `${color}50` } : {}}>
                      {isActive && <span className="inline-block w-1.5 h-1.5 rounded-full mr-1" style={{ backgroundColor: color }} />}
                      {ind.label}
                    </button>
                  );
                })}
                {availableIndicators.length === 0 && <span className="text-xs text-pat-text-muted">Collecting data — indicators appear as values update</span>}
              </div>
            </div>
          )}
        </div>
      </div>

      {chartType === "value" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">Indicator Values Over Time</h3>
            <span className="text-xs text-pat-text-muted">{selectedIndicators.length} indicators · real-time</span>
          </div>
          <CanvasLineChart selectedIndicators={selectedIndicators} history={history} liveness={liveness} />
        </div>
      )}

      {chartType === "scatter" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">Signal Frequency vs Accuracy</h3>
            <span className="text-xs text-pat-text-muted">{scatterData.length} data points · each dot = indicator × strategy</span>
          </div>
          <CanvasScatter points={scatterData} />
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-success" /><span className="text-xs text-pat-text-muted">High accuracy (&gt; 50%)</span></div>
            <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-warning" /><span className="text-xs text-pat-text-muted">Medium (20-50%)</span></div>
            <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-danger" /><span className="text-xs text-pat-text-muted">Low (&lt; 20%)</span></div>
          </div>
        </div>
      )}

      {chartType === "distribution" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">
              Value Distribution {selectedIndicators.length > 0 && <span className="text-pat-text-muted font-normal">— {liveness.find((i) => i.key === selectedIndicators[0])?.label || selectedIndicators[0]}</span>}
            </h3>
            <span className="text-xs text-pat-text-muted">{distributionData.reduce((s, d) => s + d.count, 0)} data points · 20 bins</span>
          </div>
          <CanvasBar data={distributionData} />
        </div>
      )}

      {chartType === "radar" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">Indicator Performance Radar</h3>
            <span className="text-xs text-pat-text-muted">Top {radarAxes.length} indicators across 3 metrics</span>
          </div>
          <CanvasRadar axes={radarAxes} series={radarSeries} />
        </div>
      )}
    </div>
  );
}
