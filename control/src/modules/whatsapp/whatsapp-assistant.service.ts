import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

/**
 * WhatsAppAssistantEngine — the "PAT WhatsApp AI" persona + flows.
 *
 * Persona rules (operator instruction, 2026-09-10):
 *   - concise market assistant: price, prediction bias, TA summary, signals,
 *     risk-management tips
 *   - button-driven navigation (interactive replies)
 *   - NO financial-liability disclaimers in any reply — confident, direct
 *     market-desk tone. (Compliance lives on the platform web surfaces and
 *     the static email footer, not per-message.)
 *
 * Data sources (server-authoritative — the engine NEVER recomputes indicators):
 *   - realtime gateway /market/snapshot, /signals, /cross-market/health
 *     via REALTIME_URL (same pattern as market-proxy)
 *
 * State: last-menu per phone, 30-min TTL, in-memory (per-peer; bindings are
 * DB-backed, sessions are cheap conversational state).
 */

const SESSION_TTL_MS = 30 * 60_000;

interface Session {
  menu: Menu;
  updatedAt: number;
}

type Menu = 'main' | 'price' | 'prediction' | 'ta' | 'signals' | 'risk' | 'alerts';

export interface AssistantReply {
  text: string;
  /** Interactive button rows — provider adapters map these to their shape. */
  buttons: { id: string; title: string }[];
}

@Injectable()
export class WhatsAppAssistantEngine {
  private readonly logger = new Logger(WhatsAppAssistantEngine.name);
  private readonly realtimeBase: string;
  private readonly sessions = new Map<string, Session>();

  constructor(private config: ConfigService) {
    this.realtimeBase = (
      this.config.get<string>('REALTIME_URL') || 'http://realtime:13081'
    ).replace(/\/$/, '');
  }

  /* ── session helpers ── */

  private getSession(phone: string): Session {
    const existing = this.sessions.get(phone);
    if (existing && Date.now() - existing.updatedAt < SESSION_TTL_MS) {
      return existing;
    }
    const fresh: Session = { menu: 'main', updatedAt: Date.now() };
    this.sessions.set(phone, fresh);
    // opportunistically evict stale sessions to bound memory
    if (this.sessions.size > 5000) {
      const now = Date.now();
      for (const [k, v] of this.sessions) {
        if (now - v.updatedAt > SESSION_TTL_MS) this.sessions.delete(k);
      }
    }
    return fresh;
  }

  private setSessionMenu(phone: string, menu: Menu) {
    this.sessions.set(phone, { menu, updatedAt: Date.now() });
  }

  /* ── data fetchers (server-authoritative) ── */

  private async fetchJson(path: string, timeoutMs = 5000): Promise<unknown | null> {
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

  /* ── static flows ── */

  private mainMenu(): AssistantReply {
    return {
      text:
        `*Predict-A-Trade Market Desk* 📊\n` +
        `Live XAUUSD intelligence, predictions & risk — on demand.\n\n` +
        `Tap below, or type *price*, *prediction*, *ta*, *signals*, *risk*.`,
      buttons: [
        { id: 'menu_price', title: '📈 Live Price' },
        { id: 'menu_prediction', title: '🔮 Prediction' },
        { id: 'menu_ta', title: '📊 Technical Analysis' },
        { id: 'menu_signals', title: '⚡ Signals' },
        { id: 'menu_risk', title: '🛡️ Risk Tips' },
      ],
    };
  }

  private greetingReply(name?: string): AssistantReply {
    const who = name ? ` ${name.split(' ')[0]}` : '';
    return {
      text:
        `Hey${who}! 👋 *Predict-A-Trade Market Desk* on WhatsApp.\n\n` +
        `Live XAUUSD prices, next-move predictions, technical summaries, engine signals and risk tips — one tap away.`,
      buttons: this.mainMenu().buttons,
    };
  }

  private alertsReply(): AssistantReply {
    return {
      text:
        `*Interactive Trading Alerts*\n\n` +
        `Push alerts (signal prints, SL-violation, connectivity) flow through the platform alerting pipeline.\n` +
        `Bind your WhatsApp number in the dashboard (Settings → Notifications) and alerts route here automatically.`,
      buttons: [{ id: 'menu_back', title: '⬅️ Main Menu' }],
    };
  }

  private riskReply(): AssistantReply {
    const tips = [
      'Risk 0.5–1% of account per XAUUSD trade — size from stop distance, never a fixed lot.',
      'Place the stop BEFORE entry, at structure — mental stops bleed accounts.',
      'One losing trade never justifies doubling the next lot.',
      'Daily loss cap: stop after −2R on the day and protect the week.',
      'FOMC/NFP volatility: stand aside or halve size through the release.',
      'Let the server-side risk gates (oversize rejections) do their job — they exist for your equity curve.',
    ];
    const tip = tips[Math.floor(Math.random() * tips.length)];
    return {
      text:
        `*Risk Management Tip*\n\n${tip}\n\n` +
        `Engine-side gates enforce position limits server-side on every signal.`,
      buttons: [
        { id: 'menu_risk', title: '🎲 Another tip' },
        { id: 'menu_signals', title: '⚡ Signals' },
        { id: 'menu_back', title: '⬅️ Main Menu' },
      ],
    };
  }

  /* ── data-driven flows ── */

  private fmt(n: number | undefined): string {
    return typeof n === 'number' && Number.isFinite(n) ? n.toFixed(2) : '—';
  }

  private async priceReply(): Promise<AssistantReply> {
    const snap = (await this.fetchJson('/api/v1/market/snapshot')) as
      | { bars?: Record<string, { close?: number; high?: number; low?: number }> }
      | null;
    if (!snap?.bars) {
      return {
        text: '*Live Quote*\nFeed is warming up — snapshot unavailable this second. Tap again shortly.',
        buttons: [{ id: 'menu_back', title: '⬅️ Main Menu' }],
      };
    }
    const h1 = snap.bars['H1'];
    const d1 = snap.bars['D1'];
    const line = (tf: string, b?: { close?: number; high?: number; low?: number }) =>
      `${tf} close *${this.fmt(b?.close)}*  (H ${this.fmt(b?.high)} / L ${this.fmt(b?.low)})`;
    return {
      text:
        `*XAUUSD Live Quote*\n\n` +
        `${line('H1', h1)}\n` +
        `${line('D1', d1)}\n\n` +
        `Server-authoritative feed.`,
      buttons: [
        { id: 'menu_prediction', title: '🔮 Next-move bias' },
        { id: 'menu_ta', title: '📊 Technical summary' },
        { id: 'menu_back', title: '⬅️ Main Menu' },
      ],
    };
  }

  private async predictionReply(): Promise<AssistantReply> {
    const data = (await this.fetchJson('/api/v1/signals?limit=5')) as
      | { signals?: { direction?: string; strategy_id?: string; grade?: string }[] }
      | null;
    const sigs = data?.signals ?? [];
    if (sigs.length === 0) {
      return {
        text:
          `*Prediction — Next-Move Bias*\n\n` +
          `Engine is in NO-TRADE evaluation — no directional edge worth acting on.\n` +
          `Bias: *STAND ASIDE* until a qualified signal prints.`,
        buttons: [
          { id: 'menu_price', title: '📈 Live Price' },
          { id: 'menu_signals', title: '⚡ Latest signals' },
          { id: 'menu_back', title: '⬅️ Main Menu' },
        ],
      };
    }
    const buys = sigs.filter((s) => (s.direction ?? '').includes('BUY')).length;
    const sells = sigs.filter((s) => (s.direction ?? '').includes('SELL')).length;
    const bias = buys > sells ? 'BULLISH' : sells > buys ? 'BEARISH' : 'MIXED';
    const latest = sigs[0];
    const strat = (latest.strategy_id ?? '').replace(/_/g, ' ');
    return {
      text:
        `*Prediction — Next-Move Bias*\n\n` +
        `Engine bias: *${bias}*\n` +
        `Latest print: ${latest.direction} on ${strat} (grade ${latest.grade ?? '—'})\n` +
        `Recent balance: ${buys} buy / ${sells} sell across last ${sigs.length} prints.`,
      buttons: [
        { id: 'menu_signals', title: '⚡ Full signal list' },
        { id: 'menu_risk', title: '🛡️ Risk tips' },
        { id: 'menu_back', title: '⬅️ Main Menu' },
      ],
    };
  }

  private async taReply(): Promise<AssistantReply> {
    const snap = (await this.fetchJson('/api/v1/market/snapshot')) as
      | { bars?: Record<string, { open?: number; close?: number; high?: number; low?: number }> }
      | null;
    const health = (await this.fetchJson('/api/v1/cross-market/health')) as
      | { drivers?: Record<string, { quality?: string }> }
      | null;
    if (!snap?.bars?.H1) {
      return {
        text: '*Technical Analysis*\nCandles still warming up — retry in a moment.',
        buttons: [{ id: 'menu_back', title: '⬅️ Main Menu' }],
      };
    }
    const { open, close, high, low } = snap.bars.H1;
    const o = typeof open === 'number' ? open : 0;
    const c = typeof close === 'number' ? close : 0;
    const h = typeof high === 'number' ? high : 0;
    const l = typeof low === 'number' ? low : 0;
    const range = h - l;
    const bodyPct = range > 0 ? Math.abs(c - o) / range : 0;
    const candle = c > o ? 'bullish' : c < o ? 'bearish' : 'neutral doji';
    const position = range > 0 ? ((c - l) / range) * 100 : 50;
    const posLabel =
      position > 66 ? 'buyers in control' : position < 33 ? 'sellers in control' : 'mid-range indecision';
    const connected = health?.drivers
      ? Object.values(health.drivers).filter((d) => d.quality === 'CONNECTED').length
      : null;
    return {
      text:
        `*Technical Summary — H1*\n\n` +
        `Candle: *${candle}*  O ${o.toFixed(2)} → C ${c.toFixed(2)}\n` +
        `Range: ${l.toFixed(2)} – ${h.toFixed(2)}\n` +
        `Close position: ${position.toFixed(0)}% — ${posLabel}\n` +
        `Body strength: ${(bodyPct * 100).toFixed(0)}% of range\n` +
        (connected !== null ? `Macro drivers CONNECTED: ${connected}/11\n` : '') +
        `Confluence regime + gate vetoes run server-side.`,
      buttons: [
        { id: 'menu_prediction', title: '🔮 Next-move bias' },
        { id: 'menu_price', title: '📈 Live quote' },
        { id: 'menu_back', title: '⬅️ Main Menu' },
      ],
    };
  }

  private async signalsReply(): Promise<AssistantReply> {
    const data = (await this.fetchJson('/api/v1/signals?limit=5')) as
      | { signals?: { direction?: string; strategy_id?: string; grade?: string }[] }
      | null;
    const sigs = data?.signals ?? [];
    if (sigs.length === 0) {
      return {
        text:
          `*Latest Signals*\n\n` +
          `Nothing in the recent window. The engine publishes the moment a qualified setup prints.`,
        buttons: [{ id: 'menu_back', title: '⬅️ Main Menu' }],
      };
    }
    const lines = sigs
      .slice(0, 5)
      .map(
        (s, i) =>
          `${i + 1}. *${(s.direction ?? '—').replace('_', '-')}* · ${(s.strategy_id ?? '—').replace(/_/g, ' ')} · grade ${s.grade ?? '—'}`,
      )
      .join('\n');
    return {
      text: `*Latest Engine Signals*\n\n${lines}\n\nFull evidence lives on the platform console.`,
      buttons: [
        { id: 'menu_risk', title: '🛡️ Risk tips' },
        { id: 'menu_prediction', title: '🔮 Bias' },
        { id: 'menu_back', title: '⬅️ Main Menu' },
      ],
    };
  }

  /* ── router ── */

  private textRoutes(): Record<string, RegExp> {
    return {
      menu_price: /\b(price|quote|rate|gold)\b/,
      menu_prediction: /\b(prediction|predict|bias|forecast|next move)\b/,
      menu_ta: /\b(ta|technical|analysis|chart)\b/,
      menu_signals: /\bsignal/i,
      menu_risk: /\b(risk|money management|lot|stop loss|sl)\b/,
      menu_alerts: /\balert/i,
    };
  }

  /** Handle a normalized inbound TEXT message. */
  async handle(phone: string, text: string, profileName?: string): Promise<AssistantReply> {
    const body = (text ?? '').trim().toLowerCase();
    if (/^(hi|hello|hey|start|menu|\/start|help)\b/.test(body) || body === '') {
      this.setSessionMenu(phone, 'main');
      return this.greetingReply(profileName);
    }
    for (const [menu, pattern] of Object.entries(this.textRoutes())) {
      if (pattern.test(body)) {
        const m = menu.replace(/^menu_/, '') as Menu;
        this.setSessionMenu(phone, m);
        return this.routeTo(m);
      }
    }
    return {
      text: `I didn't catch that. Try *price*, *prediction*, *ta*, *signals* or *risk* — or tap a button.`,
      buttons: this.mainMenu().buttons,
    };
  }

  /** Handle an interactive button reply (authoritative route). */
  async handleButton(phone: string, buttonId: string): Promise<AssistantReply> {
    const menu = buttonId.replace(/^menu_/, '') as Menu;
    if (['price', 'prediction', 'ta', 'signals', 'risk', 'alerts'].includes(menu)) {
      this.setSessionMenu(phone, menu);
    } else {
      this.setSessionMenu(phone, 'main');
    }
    return this.routeTo(menu);
  }

  private async routeTo(menu: string): Promise<AssistantReply> {
    switch (menu) {
      case 'price': return this.priceReply();
      case 'prediction': return this.predictionReply();
      case 'ta': return this.taReply();
      case 'signals': return this.signalsReply();
      case 'risk': return this.riskReply();
      case 'alerts': return this.alertsReply();
      default: return this.mainMenu();
    }
  }
}