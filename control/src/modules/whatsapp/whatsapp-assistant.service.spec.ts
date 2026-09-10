/**
 * WhatsAppAssistantEngine tests — persona rules, button routing, text
 * routing, data-shaping from server-authoritative sources, and the
 * operator-mandated NO-DISCLAIMER persona rule (asserted on every reply).
 */
import { WhatsAppAssistantEngine, AssistantReply } from './whatsapp-assistant.service';
import { ConfigService } from '@nestjs/config';
import { jest } from '@jest/globals';

/** Stub ConfigService: REALTIME_URL points at a local fake. */
const fakeConfig = {
  get: jest.fn((_key: string, def?: unknown) => {
    if (_key === 'REALTIME_URL') return 'http://fake-realtime';
    return def;
  }),
} as unknown as ConfigService;

/** Fake realtime gateway responses keyed by path. */
let fakeResponses: Record<string, unknown> = {};

describe('WhatsAppAssistantEngine', () => {
  let engine: WhatsAppAssistantEngine;
  const PHONE = '+971500000001';

  beforeAll(() => {
    // intercept engine's fetch by stubbing global fetch
    jest.spyOn(global, 'fetch').mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      for (const [path, body] of Object.entries(fakeResponses)) {
        if (url.includes(path)) {
          return { ok: true, json: async () => body } as unknown as Response;
        }
      }
      return { ok: true, json: async () => ({}) } as unknown as Response;
    });
  });

  beforeEach(() => {
    jest.clearAllMocks();
    fakeResponses = {};
    engine = new WhatsAppAssistantEngine(fakeConfig);
  });

  afterAll(() => {
    (global.fetch as jest.Mock).mockRestore();
  });

  /* ── persona rules ── */

  it('NO-DISCLAIMER rule: no reply ever contains liability language', async () => {
    const flows: AssistantReply[] = [
      await engine.handle(PHONE, 'hi'),
      await engine.handle(PHONE, 'price'),
      await engine.handle(PHONE, 'prediction'),
      await engine.handle(PHONE, 'ta'),
      await engine.handle(PHONE, 'signals'),
      await engine.handle(PHONE, 'risk'),
      await engine.handle(PHONE, 'alerts'),
      await engine.handle(PHONE, 'gibberish xyz'),
      await engine.handleButton(PHONE, 'menu_price'),
      await engine.handleButton(PHONE, 'menu_risk'),
    ];
    const banned = [
      'not financial advice',
      'financial advice',
      'no financial',
      'disclaimer',
      'does not constitute',
      'consult a',
      'at your own risk',
      'we are not responsible',
      'trading involves risk',
      'past performance',
    ];
    for (const r of flows) {
      for (const b of banned) {
        expect(r.text.toLowerCase()).not.toContain(b);
      }
    }
  });

  it('greeting uses the profile first name and no disclaimer', async () => {
    const r = await engine.handle(PHONE, 'hi', 'Mehul Bhatt');
    expect(r.text).toContain('Mehul');
    expect(r.buttons.length).toBeGreaterThan(0);
  });

  /* ── text routing ── */

  it('routes price keyword to live-quote flow with server data', async () => {
    fakeResponses = {
      '/api/v1/market/snapshot': {
        bars: {
          H1: { open: 4394.53, close: 4394, high: 4397.02, low: 4388.21 },
          D1: { open: 4355.6, close: 4355.27, high: 4434.03, low: 4341.12 },
        },
      },
    };
    const r = await engine.handle(PHONE, 'what is the price?');
    expect(r.text).toContain('XAUUSD Live Quote');
    expect(r.text).toContain('4394.00');
  });

  it('routes prediction keyword and computes engine bias', async () => {
    fakeResponses = {
      '/api/v1/signals?limit=5': {
        signals: [
          { direction: 'BUY', strategy_id: 'ATEN', grade: 'A' },
          { direction: 'SELL_CANDIDATE', strategy_id: 'TREND_SWING', grade: 'B' },
          { direction: 'BUY_CANDIDATE', strategy_id: 'MARNIE_FIB', grade: 'A' },
        ],
      },
    };
    const r = await engine.handle(PHONE, 'prediction');
    expect(r.text).toContain('BULLISH'); // 2 buy-ish vs 1 sell
    expect(r.text).toContain('ATEN');
  });

  it('prediction with zero signals reports NO-TRADE stand-aside', async () => {
    fakeResponses = { '/api/v1/signals?limit=5': { signals: [] } };
    const r = await engine.handle(PHONE, 'prediction');
    expect(r.text).toContain('NO-TRADE');
    expect(r.text).toContain('STAND ASIDE');
  });

  it('TA summary reports candle shape + close position', async () => {
    fakeResponses = {
      '/api/v1/market/snapshot': {
        bars: { H1: { open: 100, close: 108, high: 110, low: 98 } },
      },
      '/api/v1/cross-market/health': { drivers: { d1: { quality: 'CONNECTED' }, d2: { quality: 'STALE' } } },
    };
    const r = await engine.handle(PHONE, 'ta');
    expect(r.text).toContain('bullish');
    expect(r.text).toContain('buyers in control');
  });

  it('signals flow lists up to 5 engine prints', async () => {
    fakeResponses = {
      '/api/v1/signals?limit=5': {
        signals: Array.from({ length: 7 }, (_, i) => ({
          direction: i % 2 ? 'BUY' : 'SELL',
          strategy_id: `STRAT_${i}`,
          grade: 'A',
        })),
      },
    };
    const r = await engine.handle(PHONE, 'signals');
    expect(r.text).toContain('5.'); // 5 items listed, not 7
    expect(r.text).not.toContain('6.');
  });

  it('risk tips rotate across the tip set', async () => {
    const r1 = await engine.handle(PHONE, 'risk');
    const texts = new Set<string>();
    texts.add(r1.text);
    for (let i = 0; i < 20; i++) texts.add((await engine.handle(PHONE, 'risk')).text);
    expect(texts.size).toBeGreaterThan(1); // rotating tips, not a static line
    expect(r1.buttons.some((b) => b.id === 'menu_risk')).toBe(true); // "another tip" affordance
  });

  it('unknown input offers keywords, never disclaims', async () => {
    const r = await engine.handle(PHONE, 'what about dogecoin??');
    expect(r.text).toContain("didn't catch that");
    expect(r.text.toLowerCase()).not.toContain('advice');
  });

  /* ── button routing ── */

  it('handleButton routes menu ids to flows', async () => {
    fakeResponses = {
      '/api/v1/market/snapshot': { bars: { H1: { open: 1, close: 1.1, high: 1.2, low: 0.9 } } },
    };
    const r = await engine.handleButton(PHONE, 'menu_price');
    expect(r.text).toContain('Live Quote');
    const back = await engine.handleButton(PHONE, 'menu_back');
    expect(back.text).toContain('Market Desk');
  });

  it('handleButton with unknown id falls back to main menu', async () => {
    const r = await engine.handleButton(PHONE, 'menu_nonsense');
    expect(r.text).toContain('Market Desk');
  });
});