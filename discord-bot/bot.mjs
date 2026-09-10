/**
 * Predict-A-Trade Discord Bot — real-time alerts, portfolio tracking, and
 * price-prediction commands for the XAU/USD platform.
 *
 * discord.js v14 · slash commands · rich embeds
 *
 * Architecture (plane-safe, mirrors PAT patterns):
 *   - Reads PUBLIC realtime gateway endpoints (server-authoritative data):
 *       /market/snapshot · /candles · /signals · /cross-market/current
 *       /agents/status · /engines/status
 *   - Reads the platform DB read-only for /portfolio (trading.trade_results)
 *   - Pushes platform alerts to a Discord channel via webhook (watchdog mirror)
 *   - NEVER executes trades, never recomputes strategy scores. The realtime
 *     engine's 14-gate registry stays the sole execution authority. This bot
 *     is a presentation surface, same as the dashboards.
 *
 * Env:
 *   DISCORD_BOT_TOKEN      bot token (required)
 *   DISCORD_CLIENT_ID      application id (required, for slash registration)
 *   DISCORD_GUILD_ID       optional — register guild-scoped (instant) instead of global
 *   REALTIME_URL           default http://realtime:13081
 *   DATABASE_URL           postgres read-only connection (for /portfolio)
 *   ALERT_CHANNEL_ID       channel id the alert-poll loop posts into
 *   PAT_POLL_INTERVAL_MS   alert poll cadence (default 30000)
 */

import {
  Client,
  GatewayIntentBits,
  EmbedBuilder,
  REST,
  Routes,
  SlashCommandBuilder,
  ActionRowBuilder,
  ButtonBuilder,
  ButtonStyle,
  Events,
  time as discordTime,
  PermissionFlagsBits,
} from 'discord.js';
import pg from 'pg';

// ─────────────────────────────────────────────────────────────────────────────
// Configuration
// ─────────────────────────────────────────────────────────────────────────────

const TOKEN = process.env.DISCORD_BOT_TOKEN || '';
const CLIENT_ID = process.env.DISCORD_CLIENT_ID || '';
const GUILD_ID = process.env.DISCORD_GUILD_ID || ''; // optional: instant guild registration
const REALTIME = (process.env.REALTIME_URL || 'http://realtime:13081').replace(/\/$/, '');
const POLL_MS = parseInt(process.env.PAT_POLL_INTERVAL_MS || '30000', 10);
const ALERT_CHANNEL_ID = process.env.ALERT_CHANNEL_ID || '';

const BRAND = { blue: 0x2362ea, green: 0x22c55e, red: 0xef4444, amber: 0xe8a33d, slate: 0x77828f };

function log(level, msg) { console.log(`[${new Date().toISOString()}] [${level}] ${msg}`); }

if (!TOKEN || !CLIENT_ID) {
  log('FATAL', 'DISCORD_BOT_TOKEN and DISCORD_CLIENT_ID are required');
  process.exit(1);
}

// ─────────────────────────────────────────────────────────────────────────────
// Data layer — server-authoritative fetchers + read-only portfolio pool
// ─────────────────────────────────────────────────────────────────────────────

async function fetchJson(path, timeoutMs = 6000) {
  try {
    const ctrl = new AbortController();
    const t = setTimeout(() => ctrl.abort(), timeoutMs);
    const res = await fetch(`${REALTIME}${path}`, { signal: ctrl.signal });
    clearTimeout(t);
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

const pgPool = process.env.DATABASE_URL
  ? new pg.Pool({ connectionString: process.env.DATABASE_URL, max: 3 })
  : null;

async function query(sql, params = []) {
  if (!pgPool) throw new Error('DATABASE_URL not configured — /portfolio unavailable');
  const res = await pgPool.query(sql, params);
  return res.rows;
}

const fmt = (n, dp = 2) => (typeof n === 'number' && Number.isFinite(n) ? n.toFixed(dp) : '—');
const pct = (n) => `${n >= 0 ? '+' : ''}${Number(n).toFixed(2)}%`;

// ─────────────────────────────────────────────────────────────────────────────
// Indicator engine (presentation-grade, from server candles)
// ─────────────────────────────────────────────────────────────────────────────

function ema(values, period) {
  if (values.length < period || period <= 0) return null;
  const k = 2 / (period + 1);
  let acc = 0;
  for (let i = 0; i < period; i++) acc += values[i];
  let prev = acc / period;
  for (let i = period; i < values.length; i++) prev = values[i] * k + prev * (1 - k);
  return prev;
}

function rsi(values, period = 14) {
  if (values.length < period + 1) return null;
  let gains = 0, losses = 0;
  for (let i = 1; i <= period; i++) {
    const d = values[i] - values[i - 1];
    if (d >= 0) gains += d; else losses -= d;
  }
  let avgGain = gains / period, avgLoss = losses / period;
  for (let i = period + 1; i < values.length; i++) {
    const d = values[i] - values[i - 1];
    avgGain = (avgGain * (period - 1) + Math.max(d, 0)) / period;
    avgLoss = (avgLoss * (period - 1) + Math.max(-d, 0)) / period;
  }
  if (avgLoss === 0) return 100;
  return 100 - 100 / (1 + avgGain / avgLoss);
}

function macd(values, fast = 12, slow = 26, signalPeriod = 9) {
  if (values.length < slow + signalPeriod) return null;
  const emaSeries = (period) => {
    const k = 2 / (period + 1);
    const out = []; let prev = null;
    for (let i = 0; i < values.length; i++) {
      if (i < period - 1) { out.push(null); continue; }
      if (prev === null) {
        let acc = 0;
        for (let j = i - period + 1; j <= i; j++) acc += values[j];
        prev = acc / period;
      } else prev = values[i] * k + prev * (1 - k);
      out.push(prev);
    }
    return out;
  };
  const f = emaSeries(fast), s = emaSeries(slow);
  const line = [];
  for (let i = 0; i < values.length; i++) {
    if (f[i] !== null && s[i] !== null) line.push(f[i] - s[i]);
  }
  if (line.length < signalPeriod) return null;
  const kS = 2 / (signalPeriod + 1);
  let sig = 0;
  for (let i = 0; i < signalPeriod; i++) sig += line[i];
  sig /= signalPeriod;
  for (let i = signalPeriod; i < line.length; i++) sig = line[i] * kS + sig * (1 - kS);
  const m = line[line.length - 1];
  return { macd: m, signal: sig, histogram: m - sig };
}

function bollinger(values, period = 20, mult = 2) {
  if (values.length < period) return null;
  const slice = values.slice(-period);
  const mean = slice.reduce((a, b) => a + b, 0) / period;
  const sd = Math.sqrt(slice.reduce((a, v) => a + (v - mean) ** 2, 0) / period);
  const upper = mean + mult * sd, lower = mean - mult * sd;
  const last = values[values.length - 1];
  return {
    upper, middle: mean, lower,
    bandwidthPct: mean !== 0 ? ((upper - lower) / mean) * 100 : 0,
    percentB: upper !== lower ? (last - lower) / (upper - lower) : 0.5,
  };
}

function volumeProfile(candles, buckets = 12) {
  if (candles.length < 5) return null;
  let hi = -Infinity, lo = Infinity;
  for (const c of candles) { if (c.high > hi) hi = c.high; if (c.low < lo) lo = c.low; }
  if (hi <= lo) return null;
  const step = (hi - lo) / buckets;
  const nodes = Array.from({ length: buckets }, (_, i) => ({ price: lo + (i + 0.5) * step, volume: 0 }));
  for (const c of candles) {
    const from = Math.max(0, Math.floor(((c.low - lo) / (hi - lo)) * buckets));
    const to = Math.min(buckets - 1, Math.floor(((c.high - lo) / (hi - lo)) * buckets));
    const per = c.volume / (to - from + 1);
    for (let i = from; i <= to; i++) nodes[i].volume += per;
  }
  const total = nodes.reduce((a, b) => a + b.volume, 0);
  const pocIdx = nodes.reduce((best, n, i) => (n.volume > nodes[best].volume ? i : best), 0);
  let loI = pocIdx, hiI = pocIdx, acc = nodes[pocIdx].volume;
  while (acc < total * 0.7 && (loI > 0 || hiI < buckets - 1)) {
    const below = loI > 0 ? nodes[loI - 1].volume : -1;
    const above = hiI < buckets - 1 ? nodes[hiI + 1].volume : -1;
    if (below >= above) { loI--; acc += Math.max(below, 0); } else { hiI++; acc += Math.max(above, 0); }
  }
  return {
    poc: nodes[pocIdx].price,
    valueAreaLow: nodes[loI].price - step / 2,
    valueAreaHigh: nodes[hiI].price + step / 2,
  };
}

function computeIndicators(candles) {
  if (candles.length < 30) return null;
  const closes = candles.map((c) => c.close);
  const e9 = ema(closes, 9), e21 = ema(closes, 21), e50 = ema(closes, 50);
  const r14 = rsi(closes, 14);
  const m = macd(closes);
  const bb = bollinger(closes);
  const vp = volumeProfile(candles);
  return {
    closes, e9, e21, e50,
    emaTrend: e9 !== null && e21 !== null ? (e9 > e21 ? 'BULLISH' : e9 < e21 ? 'BEARISH' : 'FLAT') : 'FLAT',
    rsi14: r14,
    rsiLabel: r14 === null ? '—' : r14 >= 70 ? 'overbought' : r14 <= 30 ? 'oversold' : 'neutral',
    macd: m,
    macdLabel: m === null ? '—' : m.histogram > 0.01 ? 'bullish momentum' : m.histogram < -0.01 ? 'bearish momentum' : 'flat momentum',
    bb,
    bbLabel: bb === null ? '—' : bb.percentB > 1 ? 'above upper band' : bb.percentB < 0 ? 'below lower band' : bb.percentB > 0.8 ? 'upper stretch' : bb.percentB < 0.2 ? 'lower stretch' : 'mid-band',
    vp,
    vpLabel: vp === null ? '—' : closes[closes.length - 1] > vp.poc ? 'above POC' : closes[closes.length - 1] < vp.poc ? 'below POC' : 'at POC',
  };
}

// ─────────────────────────────────────────────────────────────────────────────
// Prediction engine — weighted synthesis of server-authoritative inputs
// ─────────────────────────────────────────────────────────────────────────────

async function buildPrediction() {
  const [candles, signals, cross] = await Promise.all([
    fetchJson('/api/v1/candles?tf=M15&limit=120').then((d) => d?.candles?.map((c) => ({ ...c, open: +c.open, high: +c.high, low: +c.low, close: +c.close, volume: +c.volume })) ?? null),
    fetchJson('/api/v1/signals?limit=5'),
    fetchJson('/api/v1/cross-market/current'),
  ]);
  const ind = candles ? computeIndicators(candles) : null;
  if (!ind) return null;

  const votes = [];
  if (ind.emaTrend === 'BULLISH') votes.push(['EMA 9/21 cross', +1]);
  if (ind.emaTrend === 'BEARISH') votes.push(['EMA 9/21 cross', -1]);
  if (ind.macd?.histogram > 0.01) votes.push(['MACD histogram', +1]);
  if (ind.macd?.histogram < -0.01) votes.push(['MACD histogram', -1]);
  if (ind.rsi14 > 55) votes.push(['RSI stance', +1]);
  if (ind.rsi14 < 45) votes.push(['RSI stance', -1]);
  if (ind.bb?.percentB > 0.5) votes.push(['Bollinger %B', +1]);
  if (ind.bb?.percentB < 0.5) votes.push(['Bollinger %B', -1]);

  // Engine signal balance (strongest input — gate-cleared prints)
  const sigs = signals?.signals ?? [];
  let sigBias = 0;
  for (const s of sigs) {
    const d = (s.Direction ?? '').toUpperCase();
    if (d.includes('BUY')) sigBias += 1;
    if (d.includes('SELL')) sigBias -= 1;
  }
  if (sigs.length) votes.push(['Engine signals (5-print balance)', sigBias * 2]);

  // BTC driver sentiment (risk-proxy)
  const btc = cross?.drivers?.find((d) => d.name === 'btc');
  let btcLine = 'BTC driver unavailable';
  if (btc) {
    const imp = btc.impact_score ?? 0;
    votes.push(['BTC risk sentiment', imp >= 0.05 ? 1 : imp <= -0.05 ? -1 : 0]);
    btcLine = `BTC ${fmt(btc.raw_value, 0)} · impact ${imp.toFixed(2)} (${btc.direction})`;
  }

  const score = votes.reduce((a, [, v]) => a + v, 0);
  const maxScore = votes.reduce((a, [, v]) => a + Math.max(Math.abs(v), 1), 0) || 1;
  const confidence = Math.min(95, Math.round((Math.abs(score) / maxScore) * 100));
  const bias = score >= 3 ? 'STRONG BULLISH' : score >= 1 ? 'BULLISH' : score <= -3 ? 'STRONG BEARISH' : score <= -1 ? 'BEARISH' : 'NEUTRAL / RANGE';
  const direction = score >= 1 ? 'BUY-side' : score <= -1 ? 'SELL-side' : 'STAND ASIDE';

  const last = candles[candles.length - 1];
  const stopFor = (dir) => dir === 'BUY-side' ? fmt(last.close - (ind.e21 - last.close > 0 ? (last.high - last.low) : (last.high - last.low)), 2) : null;

  return { ind, votes, score, confidence, bias, direction, btcLine, sigs, last };
}

// ─────────────────────────────────────────────────────────────────────────────
// Embed builders — rich, brand-colored, server-truth fields
// ─────────────────────────────────────────────────────────────────────────────

function embedBase(title, color = BRAND.blue) {
  return new EmbedBuilder()
    .setColor(color)
    .setTitle(title)
    .setAuthor({ name: 'Predict-A-Trade', iconURL: 'https://platform.predictatrade.com/predict-a-trade_icon-only.png' })
    .setTimestamp(new Date())
    .setFooter({ text: 'Server-authoritative feed · MT4_MASTER · 14-gate verified' });
}

function priceEmbed() {
  return embedBase('🥇 XAU/USD Live Price').setDescription('Server snapshot below — not a broker quote.');
}

async function buildPriceEmbed() {
  const snap = await fetchJson('/api/v1/market/snapshot');
  const e = priceEmbed();
  if (!snap?.bars) {
    e.setColor(BRAND.amber).setDescription('Feed warming up — snapshot unavailable.');
    return e;
  }
  const h1 = snap.bars.H1 ?? {}, d1 = snap.bars.D1 ?? {};
  const tick = snap.LastTick ?? {};
  e.addFields(
    { name: 'Last Tick', value: `Bid **${fmt(+tick.Bid)}** · Ask **${fmt(+tick.Ask)}**`, inline: true },
    { name: 'H1', value: `C **${fmt(h1.close)}**\nH ${fmt(h1.high)} · L ${fmt(h1.low)}`, inline: true },
    { name: 'D1', value: `C **${fmt(d1.close)}**\nH ${fmt(d1.high)} · L ${fmt(d1.low)}`, inline: true },
  );
  return e;
}

async function buildPredictionEmbed() {
  const p = await buildPrediction();
  if (!p) {
    return embedBase('🔮 Price Prediction — XAU/USD', BRAND.amber)
      .setDescription('Need ≥30 server candles on M15 — the feed is still warming up.');
  }
  const color = p.score >= 1 ? BRAND.green : p.score <= -1 ? BRAND.red : BRAND.amber;
  const e = embedBase('🔮 Price Prediction — XAU/USD', color)
    .setDescription(`**Bias: ${p.bias}** → ${p.direction}\nConfidence: **${p.confidence}%** (weighted vote across server-truth inputs)`)
    .addFields({ name: 'Live', value: `M15 close **${fmt(p.last.close)}**`, inline: true });

  const voteLines = p.votes.map(([name, v]) => `${v > 0 ? '🟢' : v < 0 ? '🔴' : '⚪'} ${name}: ${v > 0 ? '+' : ''}${v}`);
  e.addFields({ name: 'Vote ledger', value: voteLines.join('\n').slice(0, 1020) || '—', inline: false });

  if (p.ind) {
    e.addFields(
      { name: 'EMA trend', value: `${p.ind.emaTrend} (9 ${fmt(p.ind.e9)} / 21 ${fmt(p.ind.e21)})`, inline: true },
      { name: 'RSI(14)', value: `${fmt(p.ind.rsi14, 1)} — ${p.ind.rsiLabel}`, inline: true },
      { name: 'MACD', value: `${p.ind.macdLabel}\nhist ${fmt(p.ind.macd?.histogram, 3)}`, inline: true },
      { name: 'Bollinger', value: `${fmt(p.ind.bb?.upper)} / ${fmt(p.ind.bb?.middle)} / ${fmt(p.ind.bb?.lower)}\n${p.ind.bbLabel}`, inline: true },
      { name: 'Volume Profile', value: `POC **${fmt(p.ind.vp?.poc)}** · VA ${fmt(p.ind.vp?.valueAreaLow)} – ${fmt(p.ind.vp?.valueAreaHigh)}\n${p.ind.vpLabel}`, inline: true },
      { name: 'BTC', value: p.btcLine, inline: true },
    );
  }
  if (p.sigs.length) {
    const lines = p.sigs.slice(0, 3).map((s) =>
      `${(s.Direction ?? '').replace('_', '-')} · ${(s.StrategyID ?? '').replace(/_/g, ' ')} · grade ${s.Grade ?? '—'}`);
    e.addFields({ name: 'Latest verified prints', value: lines.join('\n'), inline: false });
  }
  return e;
}

async function buildIndicatorsEmbed() {
  const candles = await fetchJson('/api/v1/candles?tf=M15&limit=120')
    .then((d) => d?.candles?.map((c) => ({ ...c, open: +c.open, high: +c.high, low: +c.low, close: +c.close, volume: +c.volume })) ?? null);
  const ind = candles ? computeIndicators(candles) : null;
  if (!ind) {
    return embedBase('📊 Indicator Engine — XAU/USD M15', BRAND.amber).setDescription('Need ≥30 server candles — warming up.');
  }
  const vp = ind.vp;
  const e = embedBase('📊 Indicator Engine — XAU/USD M15')
    .addFields(
      { name: 'EMA', value: `9 **${fmt(ind.e9)}**\n21 **${fmt(ind.e21)}**\n50 **${fmt(ind.e50)}**\n→ ${ind.emaTrend}`, inline: true },
      { name: 'RSI(14)', value: `**${fmt(ind.rsi14, 1)}**\n${ind.rsiLabel}`, inline: true },
      { name: 'MACD(12,26,9)', value: `**${fmt(ind.macd?.macd, 3)}**\nsig ${fmt(ind.macd?.signal, 3)}\nhist ${fmt(ind.macd?.histogram, 3)}\n${ind.macdLabel}`, inline: true },
      { name: 'Bollinger(20,2)', value: `U ${fmt(ind.bb?.upper)}\nM ${fmt(ind.bb?.middle)}\nL ${fmt(ind.bb?.lower)}\n%B ${fmt(ind.bb?.percentB, 2)} · ${ind.bbLabel}`, inline: true },
      { name: 'Volume Profile', value: `POC **${fmt(vp?.poc)}**\nVA ${fmt(vp?.valueAreaLow)} – ${fmt(vp?.valueAreaHigh)}\n${ind.vpLabel}`, inline: true },
      { name: 'Read', value: `Structure: ${ind.emaTrend} + ${ind.macdLabel} + RSI ${ind.rsiLabel}`, inline: true },
    );
  return e;
}

async function buildSignalsEmbed() {
  const data = await fetchJson('/api/v1/signals?limit=5');
  const sigs = data?.signals ?? [];
  const e = embedBase('✅ Verified Signals — 14-Gate Cleared');
  if (!sigs.length) {
    e.setColor(BRAND.amber).setDescription('No qualified prints in the recent window. NO-TRADE is a valid first-class outcome.');
    return e;
  }
  for (const s of sigs.slice(0, 5)) {
    const dir = (s.Direction ?? '').replace('_', '-');
    const col = dir.includes('BUY') ? '🟢' : dir.includes('SELL') ? '🔴' : '⚪';
    e.addFields({
      name: `${col} ${dir} · ${(s.StrategyID ?? '').replace(/_/g, ' ')} · ${s.Grade ?? '—'}`,
      value:
        `Entry ${fmt(+s.EntryPrice)} · SL ${fmt(+s.StopLoss)}\n` +
        `TP1 ${fmt(+s.TP1)} (${fmt(+s.GrossRRTP1, 1)}R) · TP2 ${fmt(+s.TP2)} · TP3 ${fmt(+s.TP3)}\n` +
        `Regime: ${s.Regime ?? '—'}`,
      inline: false,
    });
  }
  return e;
}

async function buildBtcEmbed() {
  const cross = await fetchJson('/api/v1/cross-market/current');
  const btc = cross?.drivers?.find((d) => d.name === 'btc');
  const e = embedBase('₿ Bitcoin Correlation — XAU/USD');
  if (!btc) {
    e.setColor(BRAND.amber).setDescription('BTC driver unavailable.');
    return e;
  }
  e.setColor((btc.impact_score ?? 0) >= 0 ? BRAND.green : BRAND.red)
    .setDescription(`BTC **$${fmt(btc.raw_value, 0)}** · impact **${(btc.impact_score ?? 0).toFixed(2)}** (${btc.direction})`)
    .addFields(
      { name: 'Reason', value: btc.reason ?? '—', inline: false },
      { name: 'Desk read', value: 'BTC and gold share the liquidity regime — BTC impulse days front-run XAUUSD volatility expansion. Track BTC breaks as a risk-sentiment tell for gold sessions.', inline: false },
      { name: 'Freshness', value: `${((btc.freshness ?? 0) * 100).toFixed(0)}% · ${btc.quality}`, inline: true },
      { name: 'Weight', value: `${(btc.effective_weight ?? 0).toFixed(2)} effective (base ${btc.base_weight})`, inline: true },
    );
  return e;
}

async function buildMtLogEmbed() {
  const st = await fetchJson('/api/v1/agents/status');
  const eng = await fetchJson('/api/v1/engines/status');
  const e = embedBase('🖥 Real-Time MetaTrader Logging');
  if (!st) {
    e.setColor(BRAND.amber).setDescription('Agent status feed unavailable.');
    return e;
  }
  const online = st.agents_online ? BRAND.green : BRAND.red;
  e.setColor(online)
    .setDescription(`Agent mesh: **${st.agents_connected} connected** · ${st.agents_online ? 'ONLINE' : 'OFFLINE'}`)
    .addFields(
      { name: 'Market data', value: `${st.data_health ?? '—'} · last tick ${st.last_market_data_at ?? '—'}`, inline: false },
      { name: 'Edge devices', value: `${st.edge_devices_online ? 'ONLINE' : 'OFFLINE'}`, inline: true },
      { name: 'Feed source', value: 'MT4_MASTER → engine → signals (per-device delivery ledger)', inline: true },
    );
  if (eng?.engines) {
    const rows = Object.values(eng.engines).slice(0, 6)
      .map((en) => `${en.health === 'LIVE' ? '🟢' : en.health === 'WAITING' ? '🟡' : '🔴'} ${(en.engine ?? '').replace(/_/g, ' ')}: ${en.health}`);
    if (rows.length) e.addFields({ name: 'Strategy engines', value: rows.join('\n').slice(0, 1020), inline: false });
  }
  return e;
}

// ─────────────────────────────────────────────────────────────────────────────
// Portfolio (read-only DB — trading.trade_results)
// ─────────────────────────────────────────────────────────────────────────────

async function buildPortfolioEmbed(accountId) {
  if (!pgPool) {
    return embedBase('💼 Portfolio', BRAND.amber).setDescription('DATABASE_URL not configured — portfolio tracking unavailable.');
  }
  let rows;
  try {
    rows = await query(
      `SELECT strategy_id, direction, count(*)::int AS trades,
              ROUND(SUM(pnl::numeric), 2) AS total_pnl,
              ROUND(AVG(pnl::numeric), 2) AS avg_pnl,
              count(*) FILTER (WHERE is_win)::int AS wins,
              count(*) FILTER (WHERE is_loss)::int AS losses,
              MAX(closed_at) AS last_close
         FROM trading.trade_results
        WHERE ($1::text IS NULL OR account_id = $1::text)
        GROUP BY strategy_id, direction
        ORDER BY total_pnl DESC`,
      [accountId || null],
    );
  } catch (err) {
    return embedBase('💼 Portfolio', BRAND.red).setDescription(`Query failed: ${String(err.message).slice(0, 180)}`);
  }
  if (!rows.length) {
    return embedBase('💼 Portfolio', BRAND.amber)
      .setDescription(accountId ? `No executed trades found for account \`${accountId}\`.` : 'No executed trades recorded yet.');
  }
  const totalPnl = rows.reduce((a, r) => a + Number(r.total_pnl), 0);
  const totalTrades = rows.reduce((a, r) => a + r.trades, 0);
  const wins = rows.reduce((a, r) => a + r.wins, 0);
  const losses = rows.reduce((a, r) => a + r.losses, 0);
  const winRate = wins + losses > 0 ? (wins / (wins + losses)) * 100 : 0;
  const e = embedBase(accountId ? `💼 Portfolio — account ${accountId}` : '💼 Portfolio — all accounts', totalPnl >= 0 ? BRAND.green : BRAND.red)
    .setDescription(
      `**Net P&L: $${fmt(totalPnl)}** across **${totalTrades}** executed trades\n` +
      `Win rate: **${winRate.toFixed(1)}%** (${wins}W / ${losses}L)`,
    );
  for (const r of rows.slice(0, 10)) {
    const col = Number(r.total_pnl) >= 0 ? '🟢' : '🔴';
    e.addFields({
      name: `${col} ${(r.strategy_id ?? '').replace(/_/g, ' ')} — ${r.direction}`,
      value: `${r.trades} trades · P&L **$${fmt(Number(r.total_pnl))}** · avg $${fmt(Number(r.avg_pnl))}`,
      inline: true,
    });
  }
  return e;
}

// ─────────────────────────────────────────────────────────────────────────────
// Slash command registration
// ─────────────────────────────────────────────────────────────────────────────

const commands = [
  new SlashCommandBuilder().setName('price').setDescription('Live XAU/USD price from the server snapshot'),
  new SlashCommandBuilder().setName('predict').setDescription('AI price prediction: weighted bias from indicators, engine signals, BTC'),
  new SlashCommandBuilder().setName('indicators').setDescription('EMA, MACD, RSI, Bollinger Bands, Volume Profile — M15'),
  new SlashCommandBuilder().setName('signals').setDescription('Latest 14-gate-verified engine signals'),
  new SlashCommandBuilder().setName('btc').setDescription('Bitcoin correlation + liquidity-regime desk read'),
  new SlashCommandBuilder().setName('mtlog').setDescription('Real-time MetaTrader logging: agent mesh, feed health, engines'),
  new SlashCommandBuilder()
    .setName('portfolio')
    .setDescription('Executed-trade portfolio tracking (net P&L, win rate, per-strategy)')
    .addStringOption((o) => o.setName('account').setDescription('Broker account id filter (optional)')),
  new SlashCommandBuilder()
    .setName('alerts')
    .setDescription('Show which alert streams mirror into Discord + how to subscribe')
    .addStringOption((o) =>
      o.setName('stream').setDescription('Alert stream').addChoices(
        { name: 'signals', value: 'signals' },
        { name: 'system-health', value: 'system' },
      )),
].map((c) => c.toJSON());

async function registerCommands() {
  const rest = new REST({ version: '10' }).setToken(TOKEN);
  if (GUILD_ID) {
    await rest.put(Routes.applicationGuildCommands(CLIENT_ID, GUILD_ID), { body: commands });
    log('BOOT', `Registered ${commands.length} guild slash commands (instant)`);
  } else {
    await rest.put(Routes.applicationCommands(CLIENT_ID), { body: commands });
    log('BOOT', `Registered ${commands.length} global slash commands (may take up to 1h to propagate)`);
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Client
// ─────────────────────────────────────────────────────────────────────────────

const client = new Client({
  intents: [GatewayIntentBits.Guilds],
});

client.once(Events.ClientReady, async (c) => {
  log('BOOT', `Logged in as ${c.user.tag}`);
  try { await registerCommands(); } catch (e) { log('ERROR', `slash registration failed: ${e.message}`); }
  log('BOOT', `Polling alert sources every ${POLL_MS}ms — alert channel ${ALERT_CHANNEL_ID || '(unset)'}`);
});

client.on(Events.InteractionCreate, async (interaction) => {
  if (!interaction.isChatInputCommand()) return;
  try { await interaction.deferReply(); } catch { /* already acked */ }
  try {
    let embed;
    switch (interaction.commandName) {
      case 'price': embed = await buildPriceEmbed(); break;
      case 'predict': embed = await buildPredictionEmbed(); break;
      case 'indicators': embed = await buildIndicatorsEmbed(); break;
      case 'signals': embed = await buildSignalsEmbed(); break;
      case 'btc': embed = await buildBtcEmbed(); break;
      case 'mtlog': embed = await buildMtLogEmbed(); break;
      case 'portfolio': embed = await buildPortfolioEmbed(interaction.options.getString('account')); break;
      case 'alerts': {
        embed = embedBase('🔔 Alert Streams → Discord')
          .setDescription(
            'Platform alert mirrors (watchdog → Discord webhook):\n' +
            '🟢 **Signal prints** — every gate-cleared signal\n' +
            '🟠 **System health** — feed staleness, agent disconnects, disk\n' +
            '🔴 **Risk events** — SL violations, gate failures, kill-switch\n\n' +
            `Current cadence: every ${POLL_MS / 1000}s from the realtime gateway.`)
          .addFields({ name: 'Channel', value: ALERT_CHANNEL_ID ? `<#${ALERT_CHANNEL_ID}>` : 'Not configured (set ALERT_CHANNEL_ID)', inline: false });
        break;
      }
      default:
        embed = embedBase('Unknown command').setDescription('Try /price, /predict, /indicators, /signals, /btc, /mtlog, /portfolio, /alerts.');
    }
    const rows = new ActionRowBuilder().addComponents(
      new ButtonBuilder().setCustomId('refresh:price').setLabel('📈 Price').setStyle(ButtonStyle.Primary),
      new ButtonBuilder().setCustomId('refresh:predict').setLabel('🔮 Predict').setStyle(ButtonStyle.Primary),
      new ButtonBuilder().setCustomId('refresh:signals').setLabel('✅ Signals').setStyle(ButtonStyle.Secondary),
      new ButtonBuilder().setCustomId('refresh:indicators').setLabel('📊 Indicators').setStyle(ButtonStyle.Secondary),
    );
    await interaction.editReply({ embeds: [embed], components: [rows] });
  } catch (err) {
    log('ERROR', `command ${interaction.commandName} failed: ${err.message}`);
    try {
      await interaction.editReply({ content: 'Command failed — check bot logs.', embeds: [] });
    } catch { /* interaction expired */ }
  }
});

// Refresh buttons — re-run the flow for the message
client.on(Events.InteractionCreate, async (interaction) => {
  if (!interaction.isButton()) return;
  const [kind, arg] = interaction.customId.split(':');
  if (kind !== 'refresh') return;
  try { await interaction.deferUpdate(); } catch { return; }
  try {
    let embed;
    switch (arg) {
      case 'price': embed = await buildPriceEmbed(); break;
      case 'predict': embed = await buildPredictionEmbed(); break;
      case 'signals': embed = await buildSignalsEmbed(); break;
      case 'indicators': embed = await buildIndicatorsEmbed(); break;
      case 'btc': embed = await buildBtcEmbed(); break;
      case 'mtlog': embed = await buildMtLogEmbed(); break;
      default: return;
    }
    await interaction.editReply({ embeds: [embed] });
  } catch (e) {
    log('ERROR', `refresh ${interaction.customId} failed: ${e.message}`);
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Alert stream — new gate-cleared EXECUTABLE signals mirror into the channel
// ─────────────────────────────────────────────────────────────────────────────

let lastSignalId = null;
let firstPoll = true;

async function pollAlerts() {
  if (!ALERT_CHANNEL_ID) return;
  try {
    const data = await fetchJson('/api/v1/signals?limit=5');
    const sigs = data?.signals ?? [];
    const latest = sigs[0];
    if (!latest) return;
    if (firstPoll) { lastSignalId = latest.ID; firstPoll = false; return; }
    if (latest.ID === lastSignalId) return;
    lastSignalId = latest.ID;

    const dir = (latest.Direction ?? '').replace('_', '-');
    const executable = dir === 'BUY' || dir === 'SELL';
    const e = embedBase('🚨 New Signal Print', executable ? BRAND.green : BRAND.slate)
      .setDescription(
        `**${dir}** · ${(latest.StrategyID ?? '').replace(/_/g, ' ')} · grade **${latest.Grade ?? '—'}**`)
      .addFields(
        { name: 'Entry', value: fmt(+latest.EntryPrice), inline: true },
        { name: 'Stop', value: fmt(+latest.StopLoss), inline: true },
        { name: 'TP1 / TP2 / TP3', value: `${fmt(+latest.TP1)} / ${fmt(+latest.TP2)} / ${fmt(+latest.TP3)}`, inline: true },
        { name: 'R:R', value: `${fmt(+latest.GrossRRTP1, 1)}R / ${fmt(+latest.GrossRRTP2, 1)}R / ${fmt(+latest.GrossRRTP3, 1)}R`, inline: true },
        { name: 'Regime', value: latest.Regime ?? '—', inline: true },
      );
    const ch = await client.channels.fetch(ALERT_CHANNEL_ID).catch(() => null);
    if (ch?.isTextBased?.()) await ch.send({ embeds: [e] });
    log('ALERT', `posted new signal ${latest.ID} (${dir})`);
  } catch (e) {
    log('ERROR', `alert poll failed: ${e.message}`);
  }
}

setInterval(pollAlerts, POLL_MS);

// ─────────────────────────────────────────────────────────────────────────────
// Graceful shutdown
// ─────────────────────────────────────────────────────────────────────────────

process.on('SIGTERM', async () => {
  log('BOOT', 'SIGTERM — shutting down');
  if (pgPool) await pgPool.end().catch(() => {});
  client.destroy();
  process.exit(0);
});

client.login(TOKEN).catch((e) => {
  log('FATAL', `login failed: ${e.message}`);
  process.exit(1);
});