export type WsMessage =
  | { type: 'signal'; payload: SignalEvent }
  | { type: 'market'; payload: MarketDataEvent }
  | { type: 'agent'; payload: AgentStatusEvent }
  | { type: 'gate'; payload: GateRiskEvent };

export type SignalDirection = 'BUY' | 'SELL' | 'NO_TRADE';

export interface SignalEvent {
  id: string;
  direction: SignalDirection;
  probability: number;
  entryPrice: number;
  stopLoss: number;
  takeProfit: number;
  strategy: string;
  timestamp: string;
  status: 'ACTIVE' | 'CLOSED' | 'EXPIRED';
}

export interface MarketDataEvent {
  symbol: string;
  bid: number;
  ask: number;
  spread: number;
  timestamp: string;
  session?: string;
}

export interface AgentStatusEvent {
  agentId: string;
  connected: boolean;
  lastSeen: string;
  version?: string;
}

export interface GateRiskEvent {
  gate: string;
  status: 'PASS' | 'FAIL' | 'WARN';
  timestamp: string;
  details?: string;
}

export type ConnectionState = 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'RECONNECTING';

// Normalize Go engine EventEnvelope to frontend WsMessage format.
// The Go engine sends: { type: "SIGNAL", payload: { ID, Direction, StrategyID, ... } }
// The frontend expects: { type: "signal", payload: { id, direction, strategy, ... } }
type RawRecord = Record<string, unknown>;

function asRecord(value: unknown): RawRecord | null {
  return value !== null && typeof value === 'object' ? value as RawRecord : null;
}

export function normalizeWsMessage(input: unknown): WsMessage | null {
  const raw = asRecord(input);
  if (!raw || typeof raw.type !== 'string') return null;

  const typeMap: Record<string, string> = {
    'SIGNAL': 'signal',
    'MARKET_STATE': 'market',
    'MARKET_SNAPSHOT': 'market',
    'AGENT_STATUS': 'agent',
    'GATE': 'gate',
    'SNAPSHOT': 'market', // initial connection snapshot
  };

  const lowerType = typeMap[raw.type] || raw.type.toLowerCase();

  const payload = asRecord(raw.payload);

  if (lowerType === 'signal' && payload) {
    const p = payload;
    // F4 fix: never fabricate values — skip malformed events instead of
    // inventing direction/timestamps (no-fabrication rule).
    const direction = p.Direction || p.direction;
    const ts = p.CreatedAt || p.created_at || p.timestamp;
    if (!ts) return null; // no trustworthy timestamp — do not fabricate "now"
    if (direction !== 'BUY' && direction !== 'SELL' && direction !== 'NO_TRADE') {
      return null; // unknown direction — never guess
    }
    return {
      type: 'signal',
      payload: {
        id: String(p.ID || p.id || ''),
        direction,
        probability: Number(p.CalibratedProbability || p.calibratedProbability || p.probability || 0),
        entryPrice: Number(p.EntryPrice || p.entryPrice || 0),
        stopLoss: Number(p.StopLoss || p.stopLoss || 0),
        takeProfit: Number(p.TP1 || p.tp1 || p.takeProfit || 0),
        strategy: String(p.StrategyID || p.strategy || p.Strategy || ''),
        timestamp: String(ts),
        status: (p.Status || p.status || 'ACTIVE') as 'ACTIVE' | 'CLOSED' | 'EXPIRED',
      },
    };
  }

  if (lowerType === 'market' && payload) {
    const p = payload;
    // MarketState or MarketSnapshot — extract bid/ask/spread
    const tick = asRecord(p.LastTick) || asRecord(p.tick) || p;
    return {
      type: 'market',
      payload: {
        symbol: String(p.Symbol || tick.Symbol || p.symbol || 'XAUUSD'),
        bid: Number(tick.Bid || p.Bid || p.bid || 0),
        ask: Number(tick.Ask || p.Ask || p.ask || 0),
        spread: Number(tick.Spread || p.Spread || p.spread || 0),
        timestamp: String(tick.GatewayTimestamp || p.Timestamp || p.timestamp || new Date().toISOString()),
        session: String((asRecord(p.Session)?.CurrentSession || p.Session || p.session || '')),
      },
    };
  }

  if (lowerType === 'agent' && payload) {
    const p = payload;
    return {
      type: 'agent',
      payload: {
        agentId: String(p.AgentID || p.agentId || p.agent_id || ''),
        connected: Boolean(p.Connected ?? p.connected ?? (Number(p.AgentsConnected) > 0)),
        lastSeen: String(p.LastSeen || p.last_seen || p.Timestamp || new Date().toISOString()),
        version: p.Version ? String(p.Version) : undefined,
      },
    };
  }

  if (lowerType === 'gate' && payload) {
    const p = payload;
    return {
      type: 'gate',
      payload: {
        gate: String(p.Gate || p.gate || p.GateID || ''),
        status: (p.Result || p.result || p.Status || 'WARN') as 'PASS' | 'FAIL' | 'WARN',
        timestamp: String(p.Timestamp || p.timestamp || new Date().toISOString()),
        details: p.Details ? String(p.Details) : undefined,
      },
    };
  }

  // Unknown type — pass through with lowercase
  return null;
}

export class WebSocketManager {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private maxReconnectAttempts = 10;
  private reconnectDelay = 1000;
  private listeners = new Set<(msg: WsMessage) => void>();
  private stateListeners = new Set<(state: ConnectionState) => void>();
  private shouldReconnect = true;
  private connectionState: ConnectionState = 'DISCONNECTED';

  constructor(url: string) {
    this.url = url;
  }

  private setState(state: ConnectionState) {
    this.connectionState = state;
    this.stateListeners.forEach(cb => cb(state));
  }

  connect() {
    if (typeof window === 'undefined') return;
    if (this.ws?.readyState === WebSocket.OPEN) return;
    if (this.ws?.readyState === WebSocket.CONNECTING) return;

    this.shouldReconnect = true;
    this.setState('CONNECTING');

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.setState('CONNECTED');
      };

      this.ws.onmessage = (ev) => {
        try {
          const raw = JSON.parse(ev.data);
          // The Go engine sends EventEnvelope objects with uppercase type names
          // (e.g. "SIGNAL", "MARKET_STATE", "AGENT_STATUS", "SNAPSHOT", "GATE")
          // and PascalCase payload fields. Normalize to the frontend's expected
          // lowercase type names and camelCase payload fields.
          const msg = normalizeWsMessage(raw);
          if (msg) {
            this.listeners.forEach(cb => cb(msg));
          }
        } catch {
          /* ignore malformed */
        }
      };

      this.ws.onclose = () => {
        this.setState('DISCONNECTED');
        if (this.shouldReconnect) this.scheduleReconnect();
      };

      this.ws.onerror = () => {
        this.setState('DISCONNECTED');
        if (this.shouldReconnect) this.scheduleReconnect();
      };
    } catch {
      this.setState('DISCONNECTED');
      this.scheduleReconnect();
    }
  }

  disconnect() {
    this.shouldReconnect = false;
    // F3 fix: cancel any pending reconnect timer so an explicit disconnect
    // cannot be undone by a stale setTimeout firing later (zombie sockets).
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.ws?.close();
    this.ws = null;
    this.setState('DISCONNECTED');
  }

  private scheduleReconnect() {
    if (!this.shouldReconnect) return;
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;
    this.setState('RECONNECTING');
    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }

  subscribe(callback: (msg: WsMessage) => void) {
    this.listeners.add(callback);
    return () => { this.listeners.delete(callback); };
  }

  subscribeState(callback: (state: ConnectionState) => void) {
    this.stateListeners.add(callback);
    callback(this.connectionState);
    return () => { this.stateListeners.delete(callback); };
  }

  send(data: unknown) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  get readyState() {
    return this.ws?.readyState ?? WebSocket.CLOSED;
  }

  get state() {
    return this.connectionState;
  }
}

let globalWs: WebSocketManager | null = null;

/**
 * Resolve the realtime WS URL:
 *  1. NEXT_PUBLIC_WS_URL (inlined at build time — set it in the Docker build args)
 *  2. same-origin wss/ws + /ws/v1 (works when the host proxies /ws to the engine)
 */
export function defaultWsUrl(): string {
  const fromEnv = process.env.NEXT_PUBLIC_WS_URL;
  if (fromEnv) return fromEnv;
  if (typeof window === 'undefined') return '';
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws/v1`;
}

export function getGlobalWs(url?: string): WebSocketManager {
  if (!globalWs) {
    globalWs = new WebSocketManager(url || defaultWsUrl());
  }
  return globalWs;
}
