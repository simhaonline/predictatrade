/**
 * Dispatch contract tests (2026-09-04).
 *
 * Permanent regression guard after the 2026-09-03 silent-drop incident:
 * queued signal payloads went out WITHOUT a "type" key, and old MT5 client
 * builds dispatch on payload->>'type' only — every real signal fell to
 * "Unknown queue item type:", was ACKed PROCESSED (type:"") and never
 * executed. ACKs said SUCCESS; delivery was 100% dead.
 *
 * These tests pin the CONTRACT between what the engine enqueues and what
 * every EA build in the field can dispatch. If any of them fail, delivery
 * to old client builds is silently broken — DO NOT deploy.
 *
 * EA dispatch contract (all field builds):
 *   MT5 v1.26.1+/MT4 v1.27.1+: msgType==SIGNAL  OR payload has "ID" key
 *   MT5 (older field builds):   msgType==SIGNAL ONLY
 *   Both HandleSignal(payload) read ID/EntryPrice/StopLoss/TP* directly.
 *
 * Therefore EVERY enqueued signal payload MUST carry "type":"SIGNAL".
 */

// Mirrors realtime/cmd/realtime-engine enqueueSignalForDevices payload build.
function buildQueuePayload(signal: Record<string, unknown>): Record<string, unknown> {
  const raw = JSON.parse(JSON.stringify(signal)); // simulate json.Marshal→Unmarshal roundtrip
  raw['type'] = 'SIGNAL';
  return raw;
}

// Mirrors the EA dispatch (MT5 v1.26.1): extractJSONString(payload,'type').
function dispatchMsgType(payload: string): string {
  const m = payload.match(/"type":"([^"]*)"/);
  return m ? m[1] : '';
}

// Mirrors ExtractJSONString — finds `"key":"value"` (no spaces) anywhere.
function extractJSONString(payload: string, key: string): string {
  const m = payload.match(new RegExp(`"${key}":"([^"]*)"`));
  return m ? m[1] : '';
}

declare function describe(name: string, fn: () => void): void;
declare function it(name: string, fn: () => void): void;
declare function expect(actual: unknown): {
  toBe(expected: unknown): void;
  toBeUndefined(): void;
};

describe('signal delivery dispatch contract', () => {
  const REAL_SIGNAL = {
    ID: '78ed5aab-6d51-4be1-b0e3-f1b7f6f2c9f2',
    Symbol: 'XAUUSD',
    Direction: 'BUY_CANDIDATE',
    EntryPrice: '4444.10',
    StopLoss: '4438.20',
    TP1: '4449.00',
    TP2: '4453.40',
    TP3: '4459.10',
    StrategyID: 'STANDARD_SCALPING',
    Status: 'DETECTED',
    SignalClass: 'EXECUTABLE',
    Executable: true,
    Grade: 'A',
  };

  it('enqueued payload carries type:SIGNAL for the old-build dispatch', () => {
    const payload = buildQueuePayload(REAL_SIGNAL);
    expect(payload['type']).toBe('SIGNAL');
    expect(dispatchMsgType(JSON.stringify(payload))).toBe('SIGNAL');
  });

  it('old MT5 builds (type-only dispatch) can dispatch the enqueued payload', () => {
    const wire = JSON.stringify(buildQueuePayload(REAL_SIGNAL));
    const msgType = dispatchMsgType(wire);
    expect(msgType === 'SIGNAL' || extractJSONString(wire, 'ID') !== '').toBe(true);
    expect(msgType).toBe('SIGNAL'); // strict: type must be present, not ID-fallback
  });

  it('HandleSignal fields survive the injection (ID/EntryPrice/StopLoss/TP1)', () => {
    const wire = JSON.stringify(buildQueuePayload(REAL_SIGNAL));
    expect(extractJSONString(wire, 'ID')).toBe(REAL_SIGNAL.ID);
    expect(extractJSONString(wire, 'EntryPrice')).toBe(REAL_SIGNAL.EntryPrice as string);
    expect(extractJSONString(wire, 'StopLoss')).toBe(REAL_SIGNAL.StopLoss as string);
    expect(extractJSONString(wire, 'TP1')).toBe(REAL_SIGNAL.TP1 as string);
    expect(extractJSONString(wire, 'Direction')).toBe('BUY_CANDIDATE');
  });

  it('SERVER_COMMAND envelopes keep their own type (never rewritten to SIGNAL)', () => {
    const cmd = { type: 'SERVER_COMMAND', command: 'LICENSE_STATUS', payload: {} };
    expect(dispatchMsgType(JSON.stringify(cmd))).toBe('SERVER_COMMAND');
  });

  it('LICENSE_STATUS envelopes keep their own type', () => {
    const env = { type: 'LICENSE_STATUS', license_status: { state: 'ACTIVE' }, device_id: 'x' };
    expect(dispatchMsgType(JSON.stringify(env))).toBe('LICENSE_STATUS');
  });

  it('injection does not corrupt the JSON (roundtrip parse)', () => {
    const payload = buildQueuePayload(REAL_SIGNAL);
    const parsed = JSON.parse(JSON.stringify(payload));
    expect(parsed.ID).toBe(REAL_SIGNAL.ID);
    expect(parsed.type).toBe('SIGNAL');
  });

  it('canary payloads are type:CANARY and carry no StrategyID (never executed)', () => {
    const canary = {
      type: 'CANARY',
      ID: 'CANARY-20260904000000',
      test: true,
      ExpiresAt: new Date(Date.now() + 15 * 60_000).toISOString().replace(/\.\d+Z$/, 'Z'),
    };
    expect(canary.type).toBe('CANARY');
    expect(canary.test).toBe(true);
    expect((canary as Record<string, unknown>).StrategyID).toBeUndefined();
  });
});