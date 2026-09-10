/**
 * IndicatorEngine — presentation-grade technical indicators computed FROM
 * server-authoritative candles (realtime /candles, MT4_MASTER source).
 *
 * Plane note: these values are DISPLAY/desk-support derivations for the
 * Telegram assistant. The realtime engine remains the sole authority for
 * strategy scoring, gates, and execution decisions.
 */

export interface Candle {
  time: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

export interface EmaPoint { period: number; value: number }

/** Standard EMA (seed = SMA of first `period` values). */
export function ema(values: number[], period: number): number | null {
  if (values.length < period || period <= 0) return null;
  const k = 2 / (period + 1);
  let acc = 0;
  for (let i = 0; i < period; i++) acc += values[i];
  let prev = acc / period;
  for (let i = period; i < values.length; i++) {
    prev = values[i] * k + prev * (1 - k);
  }
  return prev;
}

/** RSI (Wilder smoothing). Returns 0-100. */
export function rsi(values: number[], period = 14): number | null {
  if (values.length < period + 1) return null;
  let gains = 0;
  let losses = 0;
  for (let i = 1; i <= period; i++) {
    const d = values[i] - values[i - 1];
    if (d >= 0) gains += d; else losses -= d;
  }
  let avgGain = gains / period;
  let avgLoss = losses / period;
  for (let i = period + 1; i < values.length; i++) {
    const d = values[i] - values[i - 1];
    avgGain = (avgGain * (period - 1) + Math.max(d, 0)) / period;
    avgLoss = (avgLoss * (period - 1) + Math.max(-d, 0)) / period;
  }
  if (avgLoss === 0) return 100;
  const rs = avgGain / avgLoss;
  return 100 - 100 / (1 + rs);
}

export interface MacdResult { macd: number; signal: number; histogram: number }

/** MACD (12/26/9) with EMA smoothing on MACD line for the signal. */
export function macd(values: number[], fast = 12, slow = 26, signalPeriod = 9): MacdResult | null {
  if (values.length < slow + signalPeriod) return null;
  // EMA series for fast and slow
  const emaSeries = (period: number): (number | null)[] => {
    const k = 2 / (period + 1);
    const out: (number | null)[] = [];
    let prev: number | null = null;
    for (let i = 0; i < values.length; i++) {
      if (i < period - 1) { out.push(null); continue; }
      if (prev === null) {
        let acc = 0;
        for (let j = i - period + 1; j <= i; j++) acc += values[j];
        prev = acc / period;
      } else {
        prev = values[i] * k + prev * (1 - k);
      }
      out.push(prev);
    }
    return out;
  };
  const fastS = emaSeries(fast);
  const slowS = emaSeries(slow);
  const macdLine: number[] = [];
  for (let i = 0; i < values.length; i++) {
    const f = fastS[i];
    const s = slowS[i];
    if (f !== null && s !== null) macdLine.push(f - s);
  }
  if (macdLine.length < signalPeriod) return null;
  const kS = 2 / (signalPeriod + 1);
  let sig = 0;
  for (let i = 0; i < signalPeriod; i++) sig += macdLine[i];
  sig /= signalPeriod;
  for (let i = signalPeriod; i < macdLine.length; i++) {
    sig = macdLine[i] * kS + sig * (1 - kS);
  }
  const m = macdLine[macdLine.length - 1];
  return { macd: m, signal: sig, histogram: m - sig };
}

export interface BollingerResult { upper: number; middle: number; lower: number; bandwidthPct: number; percentB: number }

/** Bollinger Bands (SMA20 ± 2σ). */
export function bollinger(values: number[], period = 20, mult = 2): BollingerResult | null {
  if (values.length < period) return null;
  const slice = values.slice(-period);
  const mean = slice.reduce((a, b) => a + b, 0) / period;
  const variance = slice.reduce((acc, v) => acc + (v - mean) ** 2, 0) / period;
  const sd = Math.sqrt(variance);
  const upper = mean + mult * sd;
  const lower = mean - mult * sd;
  const last = values[values.length - 1];
  return {
    upper, middle: mean, lower,
    bandwidthPct: mean !== 0 ? ((upper - lower) / mean) * 100 : 0,
    percentB: upper !== lower ? (last - lower) / (upper - lower) : 0.5,
  };
}

export interface VolumeNode { priceBucket: number; volume: number }
export interface VolumeProfile {
  nodes: VolumeNode[];
  poc: number;               // point of control (highest-volume bucket)
  valueAreaHigh: number;
  valueAreaLow: number;
}

/** Volume profile: bucketed traded volume by price (70% value area). */
export function volumeProfile(candles: Candle[], buckets = 12): VolumeProfile | null {
  if (candles.length < 5) return null;
  let hi = -Infinity, lo = Infinity;
  for (const c of candles) { if (c.high > hi) hi = c.high; if (c.low < lo) lo = c.low; }
  if (hi <= lo) return null;
  const step = (hi - lo) / buckets;
  const nodes: VolumeNode[] = Array.from({ length: buckets }, (_, i) => ({
    priceBucket: lo + (i + 0.5) * step,
    volume: 0,
  }));
  for (const c of candles) {
    // distribute candle volume across the buckets its range touches
    const from = Math.max(0, Math.floor(((c.low - lo) / (hi - lo)) * buckets));
    const to = Math.min(buckets - 1, Math.floor(((c.high - lo) / (hi - lo)) * buckets));
    const span = to - from + 1;
    const per = c.volume / span;
    for (let i = from; i <= to; i++) nodes[i].volume += per;
  }
  const total = nodes.reduce((a, b) => a + b.volume, 0);
  const pocNode = nodes.reduce((a, b) => (b.volume > a.volume ? b : a), nodes[0]);
  // 70% value area around POC
  let acc = pocNode.volume;
  let lowIdx = nodes.indexOf(pocNode);
  let highIdx = lowIdx;
  while (acc < total * 0.7 && (lowIdx > 0 || highIdx < buckets - 1)) {
    const below = lowIdx > 0 ? nodes[lowIdx - 1].volume : -1;
    const above = highIdx < buckets - 1 ? nodes[highIdx + 1].volume : -1;
    if (below >= above) { lowIdx--; acc += Math.max(below, 0); } else { highIdx++; acc += Math.max(above, 0); }
  }
  return {
    nodes,
    poc: pocNode.priceBucket,
    valueAreaLow: nodes[lowIdx].priceBucket - step / 2,
    valueAreaHigh: nodes[highIdx].priceBucket + step / 2,
  };
}

export interface IndicatorSnapshot {
  ema9: number | null;
  ema21: number | null;
  ema50: number | null;
  emaTrend: 'BULLISH' | 'BEARISH' | 'FLAT';
  rsi14: number | null;
  rsiLabel: string;
  macd: MacdResult | null;
  macdLabel: string;
  bb: BollingerResult | null;
  bbLabel: string;
  vp: VolumeProfile | null;
  vpLabel: string;
}

/** Compute the full indicator snapshot from server candles (closes in time order). */
export function computeIndicators(candles: Candle[]): IndicatorSnapshot | null {
  if (candles.length < 30) return null;
  const closes = candles.map((c) => c.close);
  const e9 = ema(closes, 9);
  const e21 = ema(closes, 21);
  const e50 = ema(closes, 50);
  const emaTrend =
    e9 !== null && e21 !== null ? (e9 > e21 ? 'BULLISH' : e9 < e21 ? 'BEARISH' : 'FLAT') : 'FLAT';
  const r14 = rsi(closes, 14);
  const rsiLabel =
    r14 === null ? '—'
    : r14 >= 70 ? 'overbought' : r14 <= 30 ? 'oversold' : 'neutral';
  const m = macd(closes);
  const macdLabel =
    m === null ? '—'
    : m.histogram > 0.01 ? 'bullish momentum'
    : m.histogram < -0.01 ? 'bearish momentum'
    : 'flat momentum';
  const bb = bollinger(closes);
  const bbLabel =
    bb === null ? '—'
    : bb.percentB > 1 ? 'above upper band (extended)'
    : bb.percentB < 0 ? 'below lower band (extended)'
    : bb.percentB > 0.8 ? 'upper half stretch' : bb.percentB < 0.2 ? 'lower half stretch' : 'mid-band';
  const vp = volumeProfile(candles);
  const vpLabel = vp
    ? (closes[closes.length - 1] > vp.poc
        ? 'price above POC (acceptance above value)'
        : closes[closes.length - 1] < vp.poc
          ? 'price below POC (acceptance below)'
          : 'at POC (balance)')
    : '—';
  return {
    ema9: e9, ema21: e21, ema50: e50, emaTrend,
    rsi14: r14, rsiLabel,
    macd: m, macdLabel,
    bb, bbLabel,
    vp, vpLabel,
  };
}
