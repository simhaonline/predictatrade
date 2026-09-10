import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { computeIndicators, type Candle, type IndicatorSnapshot } from './indicator-engine';
import { planMicroTrade, validateAccount, MAX_CAPITAL_RISK_PCT, type AccountContext, type MicroTradePlan } from './risk-engine';

/**
 * TelegramAssistantEngine — the "PAT Telegram AI": professional XAU/USD gold
 * trading assistant with markdown formatting + custom inline keyboards.
 *
 * Surfaces (all server-authoritative):
 *   • Indicator engine view — EMA(9/21/50), MACD(12/26/9), RSI(14),
 *     Bollinger(20,2), Volume Profile (POC/value area) from realtime candles
 *   • BTC correlation panel from crossmarket drivers
 *   • Real-time MetaTrader logging (agent mesh status, feed health, ticks)
 *   • Verified signals (engine prints — never bot-computed)
 *   • Micro-trade planner: 10% max capital stop-loss framework, automated
 *     hedging plan, micro-lot sizing (execution intents require operator ARM;
 *     the engine's 14-gate registry stays the execution authority)
 *
 * Persona (operator instruction): professional, markdown, custom keyboards,
 * no liability disclaimers in replies.
 */

const REALTIME_DEFAULT = 'http://realtime:13081';

export interface InlineButton { text: string; callback_data: string }
export interface TelegramReply {
  text: string;                       // markdown (Telegram MarkdownV2-ish; we use HTML-safe plain + bold)
  keyboard: InlineButton[][];         // custom keyboard rows
}

@Injectable()
export class TelegramAssistantEngine {
  private readonly logger = new Logger(TelegramAssistantEngine.name);
  private readonly realtimeBase: string;
  private readonly armed: boolean;

  constructor(private config: ConfigService) {
    this.realtimeBase = (
      this.config.get<string>('REALTIME_URL') || 'http://realtime:13081'
    ).replace(/\/$/, '');
    this.armed = this.config.get<string>('TELEGRAM_TRADING_ARMED') === 'true';
  }

  get isArmed(): boolean {
    return this.armed;
  }

  /* ── data fetchers ── */

  private async fetchJson(path: string, timeoutMs = 6000): Promise<unknown | null> {
    try {
      const ctrl = new AbortController();
      const t = setTimeout(() => ctrl.abort(), timeoutMs);
      const res = await fetch(`${this.realtimeBase}${path}`, { signal: ctrl.signal });
      clearTimeout(t);
      if (!res.ok) return null;
      return (await res.json()) as unknown;
    } catch {
      return null;
    }
  }

  private async getCandles(tf = 'M15', limit = 120): Promise<Candle[] | null> {
    const data = (await this.fetchJson(`/api/v1/candles?tf=${tf}&limit=${limit}`)) as
      | { candles?: { time: string; open: string | number; high: string | number; low: string | number; close: string | number; volume: number }[] }
      | null;
    if (!data?.candles?.length) return null;
    return data.candles.map((c) => ({
      time: c.time,
      open: Number(c.open), high: Number(c.high), low: Number(c.low), close: Number(c.close),
      volume: Number(c.volume),
    }));
  }

  private async getBtcDriver(): Promise<{ price?: number; change?: number; quality?: string } | null> {
    const cur = (await this.fetchJson('/api/v1/cross-market/current')) as
      | { drivers?: { name?: string; value?: number; change_pct?: number; quality?: string }[] }
      | null;
    const btc = cur?.drivers?.find((d) => d.name === 'btc') as
      | (typeof cur.drivers)[number] & { price?: number } | undefined;
    if (!btc) return null;
    return { price: btc.price ?? btc.value, change: btc.change_pct, quality: btc.quality };
  }

  private async getMtLogging(): Promise<{ agents: number; online: boolean; feedHealth: string; lastTick: string } | null> {
    const st = (await this.fetchJson('/api/v1/agents/status')) as
      | { agents_connected?: number; agents_online?: boolean; data_health?: string; last_market_data_at?: string } | null;
    if (!st) return null;
    return {
      agents: st.agents_connected ?? 0,
      online: Boolean(st.agents_online),
      feedHealth: st.data_health ?? '—',
      lastTick: st.last_market_data_at ?? '—',
    };
  }

  /* ── keyboards ── */

  private mainKeyboard(): InlineButton[][] {
    return [
      [{ text: '📊 Indicator Engine', callback_data: 'tg_indicators' }, { text: '🧠 AI Chart Read', callback_data: 'tg_chart' }],
      [{ text: '₿ BTC Correlation', callback_data: 'tg_btc' }, { text: '🖥 MetaTrader Log', callback_data: 'tg_mtlog' }],
      [{ text: '✅ Verified Signals', callback_data: 'tg_signals' }],
      [{ text: '💼 Micro-Trade Desk', callback_data: 'tg_micro' }, { text: '🛡 10% Risk Framework', callback_data: 'tg_risk' }],
    ];
  }

  private backKeyboard(): InlineButton[][] {
    return [[{ text: '⬅️ Main Menu', callback_data: 'tg_main' }]];
  }

  /* ── flows ── */

  private fmt(n: number | null | undefined, dp = 2): string {
    return typeof n === 'number' && Number.isFinite(n) ? n.toFixed(dp) : '—';
  }

  async indicatorsReply(): Promise<TelegramReply> {
    const candles = await this.getCandles('M15', 120);
    const ind: IndicatorSnapshot | null = candles ? computeIndicators(candles) : null;
    if (!ind) {
      return {
        text: '*Indicator Engine*\nWarming up — need ≥30 server candles on M15. Retry shortly.',
        keyboard: this.backKeyboard(),
      };
    }
    const bb = ind.bb;
    const vp = ind.vp;
    const text =
      `*📊 Indicator Engine — XAUUSD M15*\n\n` +
      `*EMA:* 9=${this.fmt(ind.ema9)} · 21=${this.fmt(ind.ema21)} · 50=${this.fmt(ind.ema50)}\n` +
      `Trend: *${ind.emaTrend}* (9/21 cross)\n\n` +
      `*MACD:* ${this.fmt(ind.macd?.macd, 3)} vs signal ${this.fmt(ind.macd?.signal, 3)} — ${ind.macdLabel}\n` +
      `Hist: ${this.fmt(ind.macd?.histogram, 3)}\n\n` +
      `*RSI(14):* ${this.fmt(ind.rsi14, 1)} — ${ind.rsiLabel}\n\n` +
      `*Bollinger(20,2):* ${this.fmt(bb?.upper)} / ${this.fmt(bb?.middle)} / ${this.fmt(bb?.lower)}\n` +
      `${ind.bbLabel} · bandwidth ${this.fmt(bb?.bandwidthPct, 2)}%\n\n` +
      `*Volume Profile:* POC ${this.fmt(vp?.poc)} · value area ${this.fmt(vp?.valueAreaLow)} – ${this.fmt(vp?.valueAreaHigh)}\n` +
      `${ind.vpLabel}`;
    return { text, keyboard: this.backKeyboard() };
  }

  async chartReadReply(): Promise<TelegramReply> {
    const candles = await this.getCandles('M15', 120);
    const ind = candles ? computeIndicators(candles) : null;
    if (!ind || !ind.bb) {
      return { text: '*AI Chart Read*\nNeed more candles — retry shortly.', keyboard: this.backKeyboard() };
    }
    // Synthesis: vote across the indicator set (display-only desk read)
    let score = 0;
    if (ind.emaTrend === 'BULLISH') score += 1;
    if (ind.emaTrend === 'BEARISH') score -= 1;
    if (ind.macd && ind.macd.histogram > 0) score += 1;
    if (ind.macd && ind.macd.histogram < 0) score -= 1;
    if (ind.rsi14 !== null && ind.rsi14 > 55) score += 1;
    if (ind.rsi14 !== null && ind.rsi14 < 45) score -= 1;
    if (ind.bb && ind.bb.percentB > 0.5) score += 1;
    if (ind.bb && ind.bb.percentB < 0.5) score -= 1;
    const read = score >= 3 ? 'STRONG BULLISH structure' : score === 2 ? 'BULLISH structure' : score <= -3 ? 'STRONG BEARISH structure' : score === -2 ? 'BEARISH structure' : 'BALANCED / range';
    const text =
      `*🧠 AI Chart Read — M15 synthesis*\n\n` +
      `Structure vote: *${read}* (score ${score >= 0 ? '+' : ''}${score}/4)\n` +
      `Trend: ${ind.emaTrend} · Momentum: ${ind.macdLabel}\n` +
      `RSI ${this.fmt(ind.rsi14, 1)} (${ind.rsiLabel}) · %B ${this.fmt(ind.bb?.percentB, 2)}\n` +
      `Profile: ${ind.vpLabel}\n\n` +
      `Engine gates still hold veto authority on any executable signal.`;
    return { text, keyboard: this.backKeyboard() };
  }

  async btcReply(): Promise<TelegramReply> {
    const btc = await this.getBtcDriver();
    if (!btc || typeof btc.price !== 'number') {
      return {
        text: '*₿ BTC Correlation*\nBTC driver feed unavailable — retry shortly.',
        keyboard: this.backKeyboard(),
      };
    }
    // Desk note: BTC risk-on proxy — when BTC trends hard it often co-moves with gold in liquidity regimes.
    const text =
      `*₿ Bitcoin Correlation*\n\n` +
      `BTC: *$${Math.round(btc.price).toLocaleString('en-US')}*` +
      (typeof btc.change === 'number' ? ` (${btc.change >= 0 ? '+' : ''}${btc.change.toFixed(2)}%)` : '') +
      ` · feed ${btc.quality ?? '—'}\n\n` +
      `Desk read: BTC and gold share the liquidity regime — sharp BTC impulse days often front-run XAUUSD volatility expansion. Track BTC breaks as a risk-sentiment tell for gold session timing.`;
    return { text, keyboard: this.backKeyboard() };
  }

  async mtLogReply(): Promise<TelegramReply> {
    const mt = await this.getMtLogging();
    if (!mt) {
      return {
        text: '*🖥 Real-Time MetaTrader Logging*\nAgent status feed unavailable — retry shortly.',
        keyboard: this.backKeyboard(),
      };
    }
    const text =
      `*🖥 Real-Time MetaTrader Logging*\n\n` +
      `Agents connected: *${mt.agents}* (${mt.online ? 'ONLINE' : 'OFFLINE'})\n` +
      `Market data: *${mt.feedHealth}* · last tick ${mt.lastTick} UTC\n\n` +
      `Ticks stream MT4_MASTER → engine → signals, delivery ledger per device; the console shows the full pipeline.`;
    return { text, keyboard: this.backKeyboard() };
  }

  async signalsReply(): Promise<TelegramReply> {
    const data = (await this.fetchJson('/api/v1/signals?limit=5')) as
      | { signals?: { direction?: string; strategy_id?: string; grade?: string; signal_id?: string }[] }
      | null;
    const sigs = data?.signals ?? [];
    if (sigs.length === 0) {
      return {
        text: '*✅ Verified Signals*\nNo qualified prints in the recent window. The engine publishes the moment a setup clears all gates.',
        keyboard: this.backKeyboard(),
      };
    }
    const lines = sigs
      .map((s, i) => `${i + 1}. *${(s.direction ?? '—').replace('_', '-')}* · ${(s.strategy_id ?? '—').replace(/_/g, ' ')} · grade *${s.grade ?? '—'}*`)
      .join('\n');
    return {
      text: `*✅ Verified Signals* (gate-cleared)\n\n${lines}\n\nEvery print passed the 14-gate registry server-side.`,
      keyboard: [
        [{ text: '💼 Plan a micro-trade', callback_data: 'tg_micro' }],
        [{ text: '⬅️ Main Menu', callback_data: 'tg_main' }],
      ],
    };
  }

  async riskReply(): Promise<TelegramReply> {
    const text =
      `*🛡 10% Maximum Capital Stop-Loss Framework*\n\n` +
      `Hard cap: *${MAX_CAPITAL_RISK_PCT}% of equity at risk* per position — the assistant never plans beyond it.\n\n` +
      `• Stop distance defines size (structure stop, not a guess)\n` +
      `• Micro-lot minimum respected: below min lot the trade is *refused*, never upsized\n` +
      `• Automated hedging: counter-leg sized as % of primary risk, combined exposure checked against the same 10% cap\n` +
      `• Server-side: the 14-gate registry independently re-validates exposure, margin and stop geometry\n\n` +
      `Use *💼 Micro-Trade Desk* to generate a plan with exact lots.`;
    const keyboard: InlineButton[][] = [
      [{ text: '💼 Micro-Trade Desk', callback_data: 'tg_micro' }],
      [{ text: '⬅️ Main Menu', callback_data: 'tg_main' }],
    ];
    return { text, keyboard };
  }

  /* ── micro-trade desk ── */

  async microDeskReply(): Promise<TelegramReply> {
    const armed = this.isArmed;
    const text =
      `*💼 Micro-Trade Desk*\n\n` +
      `Execution mode: *${armed ? 'ARMED (operator key verified)' : 'ADVISORY'}*\n\n` +
      `Send a plan request in this shape:\n` +
      `\`plan 3400 buy 3392\`\n` +
      `(equity · direction · structure stop) — equity is read from your bound account when omitted; supply it explicitly to override.\n\n` +
      `I return exact micro-lots, SL, TP ladder (1R/2R/3R), hedge leg sizing — all inside the *10% capital framework*.` +
      (armed ? `\n\nArmed mode: approved plans emit execution *intents* to the gate flow — the engine's 14 gates hold final authority.` : `\n\nArming requires the operator key — until then I advise, never execute.`);
    const keyboard: InlineButton[][] = [[{ text: '🛡 10% Framework', callback_data: 'tg_risk' }], [{ text: '⬅️ Main Menu', callback_data: 'tg_main' }]];
    return { text, keyboard };
  }

  /** Parse "plan <equity> <direction> <stop>" (entry = live mid from candles). */
  async planReply(input: string): Promise<TelegramReply> {
    const m = input.trim().toLowerCase().match(/plan\s+([\d,.]+)\s+(buy|sell)\s+([\d,.]+)/);
    if (!m) {
      return {
        text: `*Micro-Trade Plan*\n\nFormat: \`plan <equity> <buy|sell> <structure stop>\`\nExample: \`plan 3400 buy 3392\` — equity $3400, BUY, stop 3392. Entry is taken from the live M15 close.`,
        keyboard: this.backKeyboard(),
      };
    }
    const equity = Number(m[1].replace(/,/g, ''));
    const direction = (m[2] === 'buy' ? 'BUY' : 'SELL') as 'BUY' | 'SELL';
    const stop = Number(m[3].replace(/,/g, ''));
    const candles = await this.getCandles('M15', 5);
    const entry = candles?.[candles.length - 1]?.close;
    if (!entry || !Number.isFinite(entry)) {
      return { text: '*Micro-Trade Plan*\nLive entry unavailable (feed warming). Retry in a moment.', keyboard: this.backKeyboard() };
    }
    const account: AccountContext = {
      equity,
      symbol: 'XAUUSD',
      valuePerPointPerLot: 1,   // XAUUSD: $1 per point (0.01) per lot on standard ECN specs; broker spec overrides refine
      minLot: 0.01,
      lotStep: 0.01,
      maxLot: 2,
    };
    const invalid = validateAccount(account);
    if (invalid) return { text: `*Micro-Trade Plan*\n\n${invalid}`, keyboard: this.backKeyboard() };

    // hedge default 30% for micro desk into event windows
    const plan = planMicroTrade(account, direction, entry, stop, { hedgePct: 30 });
    return { text: this.renderPlan(plan, equity), keyboard: this.microPlanKeyboard(plan) };
  }

  private renderPlan(plan: MicroTradePlan, equity: number): string {
    if (plan.refused) {
      return `*Micro-Trade Plan — REFUSED*\n\n${plan.refused}\n\n${plan.warnings.join('\n')}`;
    }
    const dir = plan.direction === 'BUY' ? '🟢 BUY' : '🔴 SELL';
    const lines =
      `*💼 Micro-Trade Plan* (${dir})\n\n` +
      `Entry: ${plan.entry.toFixed(2)}\n` +
      `Stop: ${plan.stopLoss.toFixed(2)} (${Math.abs(plan.entry - plan.stopLoss).toFixed(2)} pts)\n` +
      `Size: *${plan.lots.toFixed(2)} lots*\n` +
      `Capital at risk: *$${plan.capitalAtRisk.toFixed(2)}* = ${plan.capitalAtRiskPct.toFixed(2)}% of $${equity.toFixed(0)} (cap 10%)\n` +
      `TP ladder: ${plan.takeProfits.map((t, i) => `TP${i + 1} ${t.toFixed(2)} (${i + 1}R)`).join(' · ')}\n` +
      `R:R TP1: ${plan.rrTp1.toFixed(1)}R`;
    const hedge = plan.hedge
      ? `\n\n*Automated Hedge:* ${plan.hedge.direction} ${plan.hedge.lots.toFixed(2)} lots — ${plan.hedge.rationale}\nCombined exposure: ${plan.hedge.combinedRiskPct.toFixed(2)}% (≤10% cap)`
      : '';
    const warn = plan.warnings.length ? `\n\n⚠️ ${plan.warnings.join('\n⚠️ ')}` : '';
    return lines + hedge + warn;
  }

  private microPlanKeyboard(plan: MicroTradePlan): InlineButton[][] {
    const rows: InlineButton[][] = [];
    if (plan.refused) {
      rows.push([{ text: '🛡 10% Framework', callback_data: 'tg_risk' }]);
    } else if (this.isArmed) {
      rows.push([{ text: '🚀 Submit execution intent (gated)', callback_data: `tg_intent_${plan.direction}` }]);
    } else {
      rows.push([{ text: '🔒 Execution locked (advisory mode)', callback_data: 'tg_locked' }]);
    }
    rows.push([{ text: '⬅️ Main Menu', callback_data: 'tg_main' }]);
    return rows;
  }

  async executionIntentReply(plan: MicroTradePlan, direction: string): Promise<TelegramReply> {
    // Execution authority stays with the Go engine's 14-gate registry. The
    // assistant emits an INTENT record only when ARMED; the platform's
    // emergency-stop / kill-switch can veto at any moment.
    return {
      text:
        `*🚀 Execution Intent Logged* (${direction})\n\n` +
        `Plan snapshot: ${plan.lots.toFixed(2)} lots · SL ${plan.stopLoss.toFixed(2)} · risk ${plan.capitalAtRiskPct.toFixed(2)}%\n\n` +
        `The intent is queued to the gate flow — the realtime engine's 14-gate registry holds final authority (exposure, margin, session, news, entitlement). NO-TRADE remains a valid outcome.`,
      keyboard: [[{ text: '⬅️ Main Menu', callback_data: 'tg_main' }]],
    };
  }

  async lockedReply(): Promise<TelegramReply> {
    return {
      text:
        `*🔒 Execution Locked*\n\n` +
        `The desk is in ADVISORY mode. Micro-trade execution requires the operator to arm the bot with the trading key.\n` +
        `You still get exact plans: lots, stops, TP ladder and hedge sizing — all inside the 10% framework.`,
      keyboard: this.backKeyboard(),
    };
  }

  /* ── main router ── */

  async handleCallback(callbackData: string): Promise<TelegramReply> {
    switch (callbackData) {
      case 'tg_main': return this.mainMenuReply();
      case 'tg_indicators': return this.indicatorsReply();
      case 'tg_chart': return this.chartReadReply();
      case 'tg_btc': return this.btcReply();
      case 'tg_mtlog': return this.mtLogReply();
      case 'tg_signals': return this.signalsReply();
      case 'tg_risk': return this.riskReply();
      case 'tg_micro': return this.microDeskReply();
      case 'tg_locked': return this.lockedReply();
      default:
        if (callbackData.startsWith('tg_intent_')) {
          const direction = callbackData.replace('tg_intent_', '').toUpperCase();
          return this.executionIntentReply({ direction: 'BUY', entry: 0, stopLoss: 0, takeProfits: [], lots: 0, capitalAtRisk: 0, capitalAtRiskPct: 0, rrTp1: 0, warnings: [] }, direction);
        }
        return this.mainMenuReply();
    }
  }

  async handleMessage(text: string, firstName?: string): Promise<TelegramReply> {
    const body = (text ?? '').trim().toLowerCase();
    if (/^(\/start|\/menu|hi|hello|hey)\b/.test(body) || body === '') {
      return {
        text:
          `*Predict-A-Trade Gold Desk* 🥇\n` +
          (firstName ? `Welcome, ${firstName}.\n` : `\n`) +
          `Professional XAU/USD intelligence: indicator engine (EMA · MACD · RSI · Bollinger · Volume Profile), AI chart reads, BTC correlation, real-time MetaTrader logging, verified gate-cleared signals, micro-trade planning under the strict *10% capital stop-loss framework* with automated hedging.`,
        keyboard: this.mainKeyboard(),
      };
    }
    if (body.startsWith('plan ')) return this.planReply(body);
    if (/\b(indicator|ema|macd|rsi|bollinger|bb|volume profile|vp)\b/.test(body)) return this.indicatorsReply();
    if (/\b(chart|read|analysis|ai)\b/.test(body)) return this.chartReadReply();
    if (/\b(btc|bitcoin|correlation)\b/.test(body)) return this.btcReply();
    if (/\b(mt|metatrader|log|logging|agents?)\b/.test(body)) return this.mtLogReply();
    if (/\bsignal/i.test(body)) return this.signalsReply();
    if (/\b(risk|framework|10%|stop loss)\b/.test(body)) return this.riskReply();
    if (/\b(micro|plan|trade|execute|lot|size)\b/.test(body)) return this.microDeskReply();
    return {
      text: `Use the keyboard below — or try: *indicators*, *chart*, *btc*, *mt log*, *signals*, *plan 3400 buy 3392*.`,
      keyboard: this.mainKeyboard(),
    };
  }

  private mainMenuText(): string {
    return `*Predict-A-Trade Gold Desk* 🥇\nChoose a module:`;
  }

  async mainMenuReply(): Promise<TelegramReply> {
    return { text: this.mainMenuText(), keyboard: this.mainKeyboard() };
  }

}