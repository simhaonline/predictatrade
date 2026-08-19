export interface MarketTick {
  symbol: string;
  bid: number;
  ask: number;
  spread: number;
  timestamp: string;
  direction: 'up' | 'down' | 'flat';
}

export interface WorkerUpdate {
  type: 'market' | 'signal' | 'agent' | 'gate';
  payload: unknown;
}

let lastBid: number | null = null;

function parseMarket(raw: unknown): MarketTick | null {
  if (typeof raw !== 'object' || raw === null) return null;
  const r = raw as Record<string, unknown>;
  const bid = typeof r.bid === 'number' ? r.bid : NaN;
  const ask = typeof r.ask === 'number' ? r.ask : NaN;
  if (Number.isNaN(bid) || Number.isNaN(ask)) return null;
  const spread = typeof r.spread === 'number' ? r.spread : ask - bid;
  const direction: 'up' | 'down' | 'flat' = lastBid === null ? 'flat' : bid > lastBid ? 'up' : bid < lastBid ? 'down' : 'flat';
  lastBid = bid;
  return {
    symbol: typeof r.symbol === 'string' ? r.symbol : 'XAUUSD',
    bid,
    ask,
    spread,
    timestamp: typeof r.timestamp === 'string' ? r.timestamp : new Date().toISOString(),
    direction,
  };
}

self.onmessage = (event: MessageEvent<{ type: string; payload: unknown }>) => {
  const { type, payload } = event.data;
  if (type === 'market') {
    const tick = parseMarket(payload);
    if (tick) {
      self.postMessage({ type: 'market', payload: tick });
    }
  } else {
    self.postMessage({ type, payload });
  }
};
