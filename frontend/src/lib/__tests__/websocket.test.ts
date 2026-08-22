import { WebSocketManager, normalizeWsMessage } from '@/lib/websocket';

const ORIGINAL_FEBSOCKET = global.WebSocket;

class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  static instances: MockWebSocket[] = [];
  readyState = 0;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  close() { this.readyState = 3; this.onclose?.(); }
  send() {}
  constructor() { MockWebSocket.instances.push(this); }
}

(MockWebSocket as unknown as Record<string, number>).CONNECTING = 0;
(MockWebSocket as unknown as Record<string, number>).OPEN = 1;
(MockWebSocket as unknown as Record<string, number>).CLOSING = 2;
(MockWebSocket as unknown as Record<string, number>).CLOSED = 3;

global.WebSocket = MockWebSocket as unknown as typeof WebSocket;

describe('WebSocketManager', () => {
  beforeEach(() => { MockWebSocket.instances = []; });
  afterAll(() => { global.WebSocket = ORIGINAL_FEBSOCKET; });

  it('should create instance', () => {
    const ws = new WebSocketManager('ws://test');
    expect(ws).toBeDefined();
  });

  it('should subscribe and unsubscribe', () => {
    const ws = new WebSocketManager('ws://test');
    const cb = jest.fn();
    const unsub = ws.subscribe(cb);
    expect(typeof unsub).toBe('function');
    unsub();
  });

  it('should handle disconnect', () => {
    const ws = new WebSocketManager('ws://test');
    ws.disconnect();
    expect(ws.readyState).toBe(WebSocket.CLOSED);
  });

  it('should not throw when sending while disconnected', () => {
    const ws = new WebSocketManager('ws://test');
    expect(() => ws.send({ type: 'ping' })).not.toThrow();
  });

  it('should have a global ws factory', async () => {
    const { getGlobalWs } = await import('@/lib/websocket');
    const ws1 = getGlobalWs('ws://test');
    const ws2 = getGlobalWs('ws://test');
    expect(ws1).toBe(ws2);
  });
});

describe('normalizeWsMessage — Go EventEnvelope to frontend WsMessage', () => {
  it('should normalize SIGNAL type to signal with camelCase fields', () => {
    const msg = normalizeWsMessage({
      event_id: 'evt-1',
      stream_id: 'signals:STANDARD_SCALPING',
      type: 'SIGNAL',
      priority: 'P1',
      payload: {
        ID: 'sig-123',
        Direction: 'BUY',
        StrategyID: 'STANDARD_SCALPING',
        CalibratedProbability: 0.75,
        EntryPrice: 4357.80,
        StopLoss: 4343.76,
        TP1: 4380.14,
        CreatedAt: '2026-08-19T13:04:00Z',
        Status: 'ACTIVE',
      },
    });

    expect(msg).not.toBeNull();
    expect(msg!.type).toBe('signal');
    expect(msg!.payload).toMatchObject({ id: 'sig-123', direction: 'BUY', strategy: 'STANDARD_SCALPING', probability: 0.75, entryPrice: 4357.80, stopLoss: 4343.76, takeProfit: 4380.14, timestamp: '2026-08-19T13:04:00Z' });
  });

  it('should normalize NO_TRADE signals', () => {
    const msg = normalizeWsMessage({
      type: 'SIGNAL',
      payload: {
        ID: 'sig-notrade',
        Direction: 'NO_TRADE',
        StrategyID: 'TREND_SWING',
        CalibratedProbability: 0.45,
        CreatedAt: '2026-08-19T13:03:00Z',
        Status: 'NO_TRADE',
      },
    });

    expect(msg!.type).toBe('signal');
    expect(msg!.payload).toMatchObject({ direction: 'NO_TRADE', strategy: 'TREND_SWING' });
  });

  it('should normalize MARKET_STATE type to market', () => {
    const msg = normalizeWsMessage({
      type: 'MARKET_STATE',
      payload: {
        Symbol: 'XAUUSD',
        Bid: 4357.38,
        Ask: 4357.80,
        Spread: 0.42,
        LastTick: { GatewayTimestamp: '2026-08-19T13:04:10Z' },
      },
    });

    expect(msg!.type).toBe('market');
    expect(msg!.payload).toMatchObject({ bid: 4357.38, ask: 4357.80, spread: 0.42 });
  });

  it('should normalize SNAPSHOT connection message', () => {
    const msg = normalizeWsMessage({
      type: 'SNAPSHOT',
      payload: { status: 'connected', client_id: 'test-123' },
    });

    expect(msg).not.toBeNull();
  });

  it('should normalize AGENT_STATUS type to agent', () => {
    const msg = normalizeWsMessage({
      type: 'AGENT_STATUS',
      payload: {
        AgentsConnected: 1,
        MasterNodeConnected: true,
        Timestamp: '2026-08-19T13:00:00Z',
      },
    });

    expect(msg!.type).toBe('agent');
    expect(msg!.payload).toMatchObject({ connected: true });
  });

  it('should return null for invalid input', () => {
    expect(normalizeWsMessage(null)).toBeNull();
    expect(normalizeWsMessage(undefined)).toBeNull();
    expect(normalizeWsMessage('not an object')).toBeNull();
  });
});
