"use client";
import { useState, useMemo, useEffect, useRef } from "react";
import {
  ScatterChart, Scatter, BarChart, Bar, Cell, XAxis, YAxis,
  CartesianGrid, Tooltip, ResponsiveContainer, Legend,
  RadarChart, Radar, PolarGrid, PolarAngleAxis, PolarRadiusAxis,
  ZAxis,
} from "recharts";
import {
  createChart, ColorType, CrosshairMode, LineStyle, LineSeries,
  type IChartApi, type ISeriesApi, type Time,
} from "lightweight-charts";
import type { IndicatorLiveness, PerformanceMetric } from "@/lib/use-indicator-liveness";

interface IndicatorChartsProps {
  liveness: IndicatorLiveness[];
  history: Map<string, { time: number; value: number }[]>;
  performance: PerformanceMetric[];
}

const INDICATOR_COLORS: Record<string, string> = {
  ema9: "#10B981", ema21: "#3B82F6", ema50: "#F59E0B", ema100: "#EF4444",
  ema200: "#8B5CF6", sma50: "#EC4899", sma100: "#06B6D4", sma200: "#84CC16",
  rsi: "#F97316", atr: "#6366F1", adx: "#10B981", macd_main: "#3B82F6",
  stoch_main: "#F59E0B", cci: "#EF4444", boll_upper: "#8B5CF6",
  boll_middle: "#EC4899", boll_lower: "#06B6D4", obv: "#F97316",
  psar: "#6366F1", vwap: "#84CC16",
};
const FALLBACK_COLORS = ["#10B981", "#3B82F6", "#F59E0B", "#EF4444", "#8B5CF6", "#EC4899"];

const TOOLTIP_STYLE = {
  background: "#0F172A",
  border: "1px solid #334155",
  borderRadius: "10px",
  color: "#94A3B8",
  fontSize: "12px",
  boxShadow: "0 4px 12px rgba(0,0,0,0.5)",
};

function getColor(key: string, idx: number): string {
  return INDICATOR_COLORS[key] || FALLBACK_COLORS[idx % FALLBACK_COLORS.length];
}

export function IndicatorCharts({ liveness, history, performance }: IndicatorChartsProps) {
  const [selectedIndicators, setSelectedIndicators] = useState<string[]>(["rsi", "atr", "adx", "cci"]);
  const [chartType, setChartType] = useState<"value" | "scatter" | "distribution" | "radar">("value");

  const availableIndicators = liveness.filter((i) => {
    const h = history.get(i.key);
    return h && h.length > 0;
  });

  const toggleIndicator = (key: string) => {
    setSelectedIndicators((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key].slice(-6)
    );
  };

  // ─── Scatter data: signal frequency vs accuracy ───────────────────────────
  const scatterData = useMemo(() => {
    return performance
      .filter((p) => p.signalFrequency !== null && p.signalAccuracy !== null)
      .map((p) => ({
        x: p.signalFrequency!,
        y: p.signalAccuracy!,
        z: 120,
        label: p.indicatorKey,
        strategy: p.strategy.replace(/_/g, " "),
        fullLabel: `${p.indicatorKey} — ${p.strategy.replace(/_/g, " ")}`,
      }));
  }, [performance]);

  // ─── Distribution data ──────────────────────────────────────────────────────
  const distributionData = useMemo(() => {
    if (selectedIndicators.length === 0) return [];
    const key = selectedIndicators[0];
    const h = history.get(key);
    if (!h || h.length === 0) return [];
    const values = h.map((p) => p.value);
    const min = Math.min(...values);
    const max = Math.max(...values);
    const bins = 20;
    const binSize = (max - min) / bins || 1;
    return Array.from({ length: bins }, (_, i) => ({
      range: `${(min + i * binSize).toFixed(2)}`,
      count: values.filter((v) => v >= min + i * binSize && v < min + (i + 1) * binSize).length,
    }));
  }, [selectedIndicators, history]);

  // ─── Radar data ─────────────────────────────────────────────────────────────
  const radarIndicators = useMemo(() => {
    return [...new Set(performance.filter(p => p.signalFrequency !== null).map(p => p.indicatorKey))].slice(0, 6);
  }, [performance]);

  const radarData = useMemo(() => {
    if (radarIndicators.length === 0) return [];
    const metrics = ["Frequency", "Accuracy", "Contribution"];
    return metrics.map(metric => {
      const point: Record<string, number | string> = { metric };
      for (const ind of radarIndicators) {
        const vals = performance.filter(p => p.indicatorKey === ind);
        if (metric === "Frequency") {
          const avg = vals.reduce((s, m) => s + (m.signalFrequency ?? 0), 0) / (vals.length || 1);
          point[ind] = parseFloat(avg.toFixed(1));
        } else if (metric === "Accuracy") {
          const valid = vals.filter(m => m.signalAccuracy !== null);
          const avg = valid.length > 0 ? valid.reduce((s, m) => s + (m.signalAccuracy ?? 0), 0) / valid.length : 0;
          point[ind] = parseFloat(avg.toFixed(1));
        } else {
          const valid = vals.filter(m => m.contributionScore !== null);
          const avg = valid.length > 0 ? valid.reduce((s, m) => s + (m.contributionScore ?? 0), 0) / valid.length : 0;
          point[ind] = parseFloat((avg * 100).toFixed(1));
        }
      }
      return point;
    });
  }, [performance, radarIndicators]);

  const chartTypes = [
    { id: "value" as const, label: "Value Timeline" },
    { id: "scatter" as const, label: "Freq vs Accuracy" },
    { id: "distribution" as const, label: "Distribution" },
    { id: "radar" as const, label: "Radar" },
  ];

  return (
    <div className="space-y-4">
      {/* Chart type selector */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap gap-1.5">
            {chartTypes.map((t) => (
              <button
                key={t.id}
                onClick={() => setChartType(t.id)}
                className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
                  chartType === t.id
                    ? "bg-pat-success/15 text-pat-success border border-pat-success/30"
                    : "text-pat-text-muted hover:text-pat-text-secondary hover:bg-pat-bg-surface-secondary/50 border border-transparent"
                }`}
              >
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
                    <button
                      key={ind.key}
                      onClick={() => toggleIndicator(ind.key)}
                      className={`px-2 py-0.5 text-[11px] rounded-md border transition-all ${
                        isActive
                          ? "text-pat-text-primary"
                          : "text-pat-text-muted border-transparent hover:border-pat-border/50"
                      }`}
                      style={isActive ? { backgroundColor: `${color}20`, borderColor: `${color}50` } : {}}
                    >
                      {isActive && <span className="inline-block w-1.5 h-1.5 rounded-full mr-1" style={{ backgroundColor: color }} />}
                      {ind.label}
                    </button>
                  );
                })}
                {availableIndicators.length === 0 && (
                  <span className="text-xs text-pat-text-muted">Collecting data — indicators appear as values update</span>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Value Timeline — using lightweight-charts for real-time */}
      {chartType === "value" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">Indicator Values Over Time</h3>
            <span className="text-xs text-pat-text-muted">{selectedIndicators.length} indicators · real-time</span>
          </div>
          <RealtimeLineChart
            selectedIndicators={selectedIndicators}
            history={history}
            liveness={liveness}
          />
        </div>
      )}

      {/* Scatter — Freq vs Accuracy */}
      {chartType === "scatter" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">Signal Frequency vs Accuracy</h3>
            <span className="text-xs text-pat-text-muted">{scatterData.length} data points · each dot = indicator × strategy</span>
          </div>
          {scatterData.length > 0 ? (
            <ResponsiveContainer width="100%" height={450}>
              <ScatterChart margin={{ top: 20, right: 40, left: 20, bottom: 50 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1E293B" />
                <XAxis
                  type="number" dataKey="x" name="Signal Frequency"
                  unit="%" domain={[0, 100]}
                  tick={{ fill: "#475569", fontSize: 11 }}
                  stroke="#1E293B" tickLine={false} axisLine={false}
                  label={{ value: "Signal Frequency (%)", position: "bottom", fill: "#94A3B8", fontSize: 12, offset: 15 }}
                />
                <YAxis
                  type="number" dataKey="y" name="Signal Accuracy"
                  unit="%" domain={[0, 100]}
                  tick={{ fill: "#475569", fontSize: 11 }}
                  stroke="#1E293B" tickLine={false} axisLine={false}
                  label={{ value: "Signal Accuracy (%)", angle: -90, position: "insideLeft", fill: "#94A3B8", fontSize: 12, offset: 0 }}
                />
                <ZAxis type="number" dataKey="z" range={[80, 200]} />
                <Tooltip
                  contentStyle={TOOLTIP_STYLE}
                  cursor={{ strokeDasharray: "3 3", stroke: "#334155" }}
                  formatter={(value, _name, props) => {
                    if (props && props.payload) {
                      const p = props.payload;
                      return [
                        <div key="tip" style={{ lineHeight: 1.5 }}>
                          <div style={{ color: "#F1F5F9", fontWeight: 600, marginBottom: 2 }}>{p.fullLabel}</div>
                          <div style={{ color: "#94A3B8" }}>Frequency: {Number(p.x).toFixed(1)}%</div>
                          <div style={{ color: "#94A3B8" }}>Accuracy: {Number(p.y).toFixed(1)}%</div>
                        </div>,
                        "",
                      ];
                    }
                    return [`${Number(value).toFixed(1)}%`, ""];
                  }}
                />
                <Scatter data={scatterData} shape="circle">
                  {scatterData.map((d, i) => {
                    // Color by accuracy: green > 50, yellow 20-50, red < 20
                    const color = d.y > 50 ? "#10B981" : d.y > 20 ? "#F59E0B" : "#EF4444";
                    return <Cell key={i} fill={color} fillOpacity={0.75} stroke={color} strokeWidth={1} />;
                  })}
                </Scatter>
              </ScatterChart>
            </ResponsiveContainer>
          ) : (
            <EmptyChart message="No scatter data available. Data populates as the engine generates signals with evidence." />
          )}
          {/* Legend */}
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-success" /><span className="text-xs text-pat-text-muted">High accuracy ({" > 50%"})</span></div>
            <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-warning" /><span className="text-xs text-pat-text-muted">Medium (20-50%)</span></div>
            <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-pat-danger" /><span className="text-xs text-pat-text-muted">Low ({"< 20%"})</span></div>
          </div>
        </div>
      )}

      {/* Distribution */}
      {chartType === "distribution" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">
              Value Distribution {selectedIndicators.length > 0 && <span className="text-pat-text-muted font-normal">— {liveness.find(i => i.key === selectedIndicators[0])?.label || selectedIndicators[0]}</span>}
            </h3>
            <span className="text-xs text-pat-text-muted">{distributionData.reduce((s, d) => s + d.count, 0)} data points · 20 bins</span>
          </div>
          {distributionData.length > 0 ? (
            <ResponsiveContainer width="100%" height={400}>
              <BarChart data={distributionData} margin={{ top: 10, right: 20, left: 0, bottom: 30 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1E293B" vertical={false} />
                <XAxis dataKey="range" tick={{ fill: "#475569", fontSize: 9 }} stroke="#1E293B" tickLine={false} axisLine={false} angle={-40} textAnchor="end" height={50} interval={1} />
                <YAxis tick={{ fill: "#475569", fontSize: 10 }} stroke="#1E293B" tickLine={false} axisLine={false} />
                <Tooltip contentStyle={TOOLTIP_STYLE} cursor={{ fill: "#1E293B40" }} />
                <Bar dataKey="count" fill="#F59E0B" radius={[3, 3, 0, 0]} barSize={24} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <EmptyChart message="No distribution data available. Select an indicator with historical values." />
          )}
        </div>
      )}

      {/* Radar */}
      {chartType === "radar" && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-pat-text-primary">Indicator Performance Radar</h3>
            <span className="text-xs text-pat-text-muted">Top {radarIndicators.length} indicators across 3 metrics</span>
          </div>
          {radarData.length > 0 && radarIndicators.length > 0 ? (
            <ResponsiveContainer width="100%" height={400}>
              <RadarChart data={radarData} margin={{ top: 20, right: 40, left: 40, bottom: 20 }}>
                <PolarGrid stroke="#1E293B" />
                <PolarAngleAxis dataKey="metric" tick={{ fill: "#94A3B8", fontSize: 11 }} />
                <PolarRadiusAxis tick={{ fill: "#475569", fontSize: 9 }} stroke="#1E293B" angle={90} />
                {radarIndicators.map((ind, i) => (
                  <Radar key={ind} name={ind} dataKey={ind} stroke={getColor(ind, i)} fill={getColor(ind, i)} fillOpacity={0.12} strokeWidth={2} />
                ))}
                <Legend wrapperStyle={{ fontSize: "11px" }} iconType="circle" />
                <Tooltip contentStyle={TOOLTIP_STYLE} />
              </RadarChart>
            </ResponsiveContainer>
          ) : (
            <EmptyChart message="No radar data available. Requires signal performance metrics." />
          )}
        </div>
      )}
    </div>
  );
}

// ─── Real-time line chart using lightweight-charts ────────────────────────────
function RealtimeLineChart({
  selectedIndicators,
  history,
  liveness,
}: {
  selectedIndicators: string[];
  history: Map<string, { time: number; value: number }[]>;
  liveness: IndicatorLiveness[];
}) {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<Map<string, ISeriesApi<"Line">>>(new Map());
  const lastUpdateRef = useRef<number>(0);

  // Initialize chart
  useEffect(() => {
    if (!chartContainerRef.current) return;

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: "transparent" },
        textColor: "#94A3B8",
        fontSize: 11,
        fontFamily: "monospace",
      },
      grid: {
        vertLines: { color: "#1E293B", style: LineStyle.Dashed },
        horzLines: { color: "#1E293B", style: LineStyle.Dashed },
      },
      crosshair: {
        mode: CrosshairMode.Normal,
        vertLine: { color: "#334155", labelBackgroundColor: "#0F172A" },
        horzLine: { color: "#334155", labelBackgroundColor: "#0F172A" },
      },
      rightPriceScale: { borderColor: "#1E293B" },
      timeScale: {
        borderColor: "#1E293B",
        timeVisible: true,
        secondsVisible: true,
        rightOffset: 2,
      },
      width: chartContainerRef.current.clientWidth,
      height: 400,
    });

    chartRef.current = chart;

    // Resize observer
    const resizeObserver = new ResizeObserver((entries) => {
      if (entries.length > 0 && chartRef.current) {
        chartRef.current.applyOptions({ width: entries[0].contentRect.width });
      }
    });
    resizeObserver.observe(chartContainerRef.current);

    return () => {
      resizeObserver.disconnect();
      chart.remove();
      chartRef.current = null;
      // eslint-disable-next-line react-hooks/exhaustive-deps
      seriesRef.current.clear();
    };
  }, []);

  // Update series when selected indicators change
  useEffect(() => {
    if (!chartRef.current) return;

    // Remove old series
    for (const [key, series] of seriesRef.current) {
      if (!selectedIndicators.includes(key)) {
        chartRef.current.removeSeries(series);
        seriesRef.current.delete(key);
      }
    }

    // Add new series
    selectedIndicators.forEach((key, idx) => {
      if (!seriesRef.current.has(key)) {
        const color = getColor(key, idx);
        const series = chartRef.current!.addSeries(LineSeries, {
          color,
          lineWidth: 2,
          priceLineVisible: false,
          lastValueVisible: true,
          title: liveness.find(i => i.key === key)?.label || key,
        });
        seriesRef.current.set(key, series);

        // Load existing history
        const hist = history.get(key) || [];
        if (hist.length > 0) {
          const data = hist.map(p => ({
            time: Math.floor(p.time / 1000) as Time,
            value: p.value,
          })).filter((d, i, arr) => i === 0 || d.time !== arr[i-1].time);
          series.setData(data);
        }
      }
    });
  }, [selectedIndicators, history, liveness]);

  // Real-time update on history changes
  useEffect(() => {
    if (!chartRef.current) return;

    for (const key of selectedIndicators) {
      const series = seriesRef.current.get(key);
      if (!series) continue;

      const hist = history.get(key) || [];
      if (hist.length === 0) continue;

      const latest = hist[hist.length - 1];
      const latestTime = Math.floor(latest.time / 1000) as Time;

      // Only update if new data point
      if ((latestTime as number) > lastUpdateRef.current) {
        lastUpdateRef.current = latestTime as number;
        for (const k of selectedIndicators) {
          const s = seriesRef.current.get(k);
          const h = history.get(k);
          if (s && h && h.length > 0) {
            const last = h[h.length - 1];
            s.update({
              time: Math.floor(last.time / 1000) as Time,
              value: last.value,
            });
          }
        }
      }
    }
  }, [history, selectedIndicators]);

  if (selectedIndicators.length === 0 || availableIndicatorsCount(history, selectedIndicators) === 0) {
    return <EmptyChart message="No historical data yet. The chart will populate as indicator values update in real time. This takes a few seconds after the page loads." />;
  }

  return <div ref={chartContainerRef} className="w-full" style={{ height: 400 }} />;
}

function availableIndicatorsCount(history: Map<string, { time: number; value: number }[]>, keys: string[]): number {
  return keys.filter(k => (history.get(k)?.length ?? 0) > 0).length;
}

function EmptyChart({ message }: { message: string }) {
  return (
    <div className="flex items-center justify-center h-[300px] rounded-lg bg-pat-bg-surface-secondary/20">
      <div className="text-center">
        <svg className="mx-auto h-10 w-10 text-pat-text-muted/40 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
        </svg>
        <p className="text-xs text-pat-text-muted max-w-xs">{message}</p>
      </div>
    </div>
  );
}
