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
  // prompt.md Sections 12-14: Quality grade + Expectancy
  qualityGrade?: string;
  expectancyR?: number;
  expectancyScore?: number;
  // prompt.md Section 18: Rejection diagnostics
  primaryRejectionReason?: string;
  rejectionReasons?: string[];
  // RR fields
  grossRRTP1?: number;
  grossRRTP2?: number;
  grossRRTP3?: number;
  // Multi-TP + market context (forwarded from Go engine signal)
  tp2?: number;
  tp3?: number;
  regime?: string;
  session?: string;
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

// Honest feed status reported by the relay/provider (never assume LIVE).
export type FeedStatus = 'LIVE' | 'DEGRADED' | 'STALE' | 'REPLAY' | 'UNKNOWN';

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
        tp2: Number(p.TP2 || p.tp2 || 0),
        tp3: Number(p.TP3 || p.tp3 || 0),
        regime: String(p.Regime || p.regime || ''),
        session: String(p.Session || p.session || ''),
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
  private feedStatus: FeedStatus = 'UNKNOWN';
  private feedStatusListeners = new Set<(status: FeedStatus) => void>();

  constructor(url: string) {
    this.url = url;
  }

  private setState(state: ConnectionState) {
    this.connectionState = state;
    this.stateListeners.forEach(cb => cb(state));
  }

  private setFeedStatus(status: FeedStatus) {
    this.feedStatus = status;
    this.feedStatusListeners.forEach(cb => cb(status));
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
          
          // ── Resilient Relay Format ──
          // The relay sends a combined JSON: {status, engine_online, market_snapshot, 
          // system_health, agents_status, signals}
          // Parse and dispatch as individual typed events.
          if (raw.status === "LIVE" || raw.status === "DEGRADED" || raw.status === "STALE" || raw.status === "REPLAY") {
            // F4 fix: thread the parsed feed status so the UI can render an
            // honest LIVE/DEGRADED/STALE/REPLAY badge instead of assuming LIVE.
            this.setFeedStatus(raw.status as FeedStatus);
            // Market snapshot → emit as "market" event
            if (raw.market_snapshot) {
              const ms = raw.market_snapshot;
              if (ms.tick || ms.indicators) {
                const tick = ms.tick || {};
                this.listeners.forEach(cb => cb({
                  type: "market",
                  payload: {
                    symbol: String(ms.Symbol || tick.symbol || "XAUUSD"),
                    bid: Number(tick.bid || ms.Bid || 0),
                    ask: Number(tick.ask || ms.Ask || 0),
                    spread: Number(tick.spread || ms.Spread || 0),
                    timestamp: String(ms.timestamp || tick.timestamp || new Date().toISOString()),
                    session: String(ms.session || ms.indicators?.session || ""),
                  }
                }));
              }
            }
            // System health → emit as "agent" event
            if (raw.system_health) {
              const sh = raw.system_health;
              const ms = sh.market_source || {};
              this.listeners.forEach(cb => cb({
                type: "agent",
                payload: {
                  agentId: "relay",
                  connected: true,
                  lastSeen: String(sh.timestamp || new Date().toISOString()),
                }
              }));
            }
            // Signals → emit as "signal" events
            if (raw.signals) {
              const sigs = raw.signals.signals || raw.signals;
              if (Array.isArray(sigs)) {
                for (const s of sigs.slice(0, 5)) {
                  const dir = String(s.Direction || s.direction || "NO-TRADE");
                  if (dir === "BUY" || dir === "SELL" || dir === "BUY_CANDIDATE" || dir === "SELL_CANDIDATE") {
                    this.listeners.forEach(cb => cb({
                      type: "signal",
                      payload: {
                        id: String(s.ID || s.id || ""),
                        direction: dir as any,
                        probability: Number(s.CalibratedProbability || s.calibratedProbability || 0),
                        entryPrice: Number(s.EntryPrice || s.entryPrice || 0),
                        stopLoss: Number(s.StopLoss || s.stopLoss || 0),
                        takeProfit: Number(s.TP1 || s.tp1 || 0),
                        strategy: String(s.StrategyID || s.strategy || ""),
                        timestamp: String(s.CreatedAt || s.created_at || s.timestamp || new Date().toISOString()),
                        status: String(s.Status || s.status || "ACTIVE") as any,
                        tp2: Number(s.TP2 || s.tp2 || 0),
                        tp3: Number(s.TP3 || s.tp3 || 0),
                        regime: String(s.Regime || s.regime || ""),
                        session: String(s.Session || s.session || ""),
                      }
                    }));
                  }
                }
              }
            }
            return;
          }
          
          // ── Legacy format (direct engine WebSocket) ──
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
        this.setFeedStatus('UNKNOWN');
        if (this.shouldReconnect) this.scheduleReconnect();
      };

      this.ws.onerror = () => {
        this.setState('DISCONNECTED');
        this.setFeedStatus('UNKNOWN');
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

  subscribeFeedStatus(callback: (status: FeedStatus) => void) {
    this.feedStatusListeners.add(callback);
    callback(this.feedStatus);
    return () => { this.feedStatusListeners.delete(callback); };
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
