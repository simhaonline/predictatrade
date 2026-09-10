/**
 * IndicatorEngine tests — EMA/RSI/MACD/Bollinger/VolumeProfile math against
 * known-answer fixtures (hand-computed), plus the full computeIndicators
 * snapshot from synthetic candle series.
 */
import { ema, rsi, macd, bollinger, volumeProfile, computeIndicators, type Candle } from './indicator-engine';

function candlesFrom(closes: number[], volume = 100): Candle[] {
  return closes.map((c, i) => ({
    time: new Date(Date.UTC(2026, 8, 10, 12, i)).toISOString(),
    open: c - 0.5, high: c + 1, low: c - 1, close: c, volume,
  }));
}

describe('IndicatorEngine', () => {
  it('ema seeds with SMA then smooths (known 5-period case)', () => {
    const values = [2, 4, 6, 8, 10, 12];
    // SMA(5) of first 5 = 6; k = 2/6; next = 12*(0.3333) + 6*(0.6667) = 8
    expect(ema(values, 5)).toBeCloseTo(8, 6);
  });

  it('ema returns null for insufficient data', () => {
    expect(ema([1, 2], 5)).toBeNull();
  });

  it('rsi is 100 when all gains (no losses)', () => {
    expect(rsi([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16], 14)).toBe(100);
  });

  it('rsi is 0 when all losses', () => {
    expect(rsi([16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1], 14)).toBe(0);
  });

  it('rsi sits mid-range on alternating series', () => {
    const r = rsi([10, 11, 10, 11, 10, 11, 10, 11, 10, 11, 10, 11, 10, 11, 10, 11], 14);
    expect(r).toBeGreaterThan(45);
    expect(r).toBeLessThan(55);
  });

  it('macd histogram is ~0 (flat momentum) on a perfectly linear ramp — float noise tolerated', () => {
    const up = Array.from({ length: 40 }, (_, i) => 100 + i);
    const m = macd(up);
    expect(m).not.toBeNull();
    expect(m!.macd).toBeGreaterThan(0);            // macd line positive on the ramp
    expect(Math.abs(m!.histogram)).toBeLessThan(0.01); // converged signal ≈ macd
  });

  it('bollinger: price above upper → percentB > 1', () => {
    const flat = Array.from({ length: 30 }, () => 100);
    const values = [...flat, 110];
    const bb = bollinger(values, 20, 2);
    expect(bb).not.toBeNull();
    expect(bb!.percentB).toBeGreaterThan(1);
  });

  it('bollinger symmetric bands on flat series', () => {
    const flat = Array.from({ length: 25 }, () => 100);
    const bb = bollinger(flat, 20, 2);
    expect(bb!.upper).toBeCloseTo(100, 6);
    expect(bb!.lower).toBeCloseTo(100, 6);
    expect(bb!.bandwidthPct).toBeCloseTo(0, 6);
  });

  it('volume profile: POC sits at the highest-volume price bucket', () => {
    const candles: Candle[] = [];
    for (let i = 0; i < 30; i++) {
      // everything trades around 100 except a big-volume cluster at 105
      const price = i % 5 === 0 ? 105 : 95;
      candles.push({ time: '', open: price - 0.5, high: price + 1, low: price - 1, close: price + 0.5, volume: i % 5 === 0 ? 10000 : 10 });
    }
    const vp = volumeProfile(candles, 10);
    expect(vp).not.toBeNull();
    // POC should be in the upper half (the 105 cluster)
    expect(vp!.poc).toBeGreaterThan(100);
  });

  it('computeIndicators returns null under 30 candles', () => {
    expect(computeIndicators(candlesFrom([1, 2, 3]))).toBeNull();
  });

  it('computeIndicators full snapshot labels a strong uptrend', () => {
    // accelerating series so MACD histogram is clearly positive (linear ramps
    // converge histogram→0 by construction)
    const closes = Array.from({ length: 60 }, (_, i) => 4000 + i * i * 0.5);
    const ind = computeIndicators(candlesFrom(closes));
    expect(ind).not.toBeNull();
    expect(ind!.emaTrend).toBe('BULLISH');
    expect(ind!.macdLabel).toContain('bullish');
    expect(ind!.rsi14).toBeGreaterThan(60);
  });

  it('computeIndicators labels a strong downtrend', () => {
    // accelerating decline → clearly negative MACD histogram (linear declines
    // converge histogram→0 by construction)
    const closes = Array.from({ length: 60 }, (_, i) => 4000 - i * i * 0.5);
    const ind = computeIndicators(candlesFrom(closes));
    expect(ind!.emaTrend).toBe('BEARISH');
    expect(ind!.rsi14).toBeLessThan(40);
    expect(ind!.macdLabel).toContain('bearish');
  });
});