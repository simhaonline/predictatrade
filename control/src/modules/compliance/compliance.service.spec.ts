import { extractClientIp, ComplianceService } from './compliance.service';
import { jest } from '@jest/globals';

describe('IP Extraction (Security)', () => {
  it('uses socket IP for direct connection (no trusted proxy)', () => {
    const result = extractClientIp({}, '203.0.113.10');
    expect(result.ip).toBe('203.0.113.10');
    expect(result.proxyChain).toEqual([]);
  });

  it('ignores spoofed X-Forwarded-For from untrusted direct client', () => {
    const result = extractClientIp(
      { 'x-forwarded-for': '8.8.8.8' },
      '203.0.113.10'
    );
    // Should NOT use 8.8.8.8 — client is not a trusted proxy
    expect(result.ip).toBe('203.0.113.10');
  });

  it('ignores spoofed CF-Connecting-IP from untrusted direct client', () => {
    const result = extractClientIp(
      { 'cf-connecting-ip': '8.8.8.8' },
      '203.0.113.10'
    );
    expect(result.ip).toBe('203.0.113.10');
  });

  it('uses X-Real-IP from trusted proxy (172.18.x.x)', () => {
    const result = extractClientIp(
      { 'x-real-ip': '198.51.100.5' },
      '172.18.0.1'
    );
    expect(result.ip).toBe('198.51.100.5');
  });

  it('uses CF-Connecting-IP from trusted proxy', () => {
    const result = extractClientIp(
      { 'cf-connecting-ip': '198.51.100.5', 'x-forwarded-for': '198.51.100.5, 172.68.0.1' },
      '172.18.0.1'
    );
    expect(result.ip).toBe('198.51.100.5');
    expect(result.proxyChain.length).toBeGreaterThan(0);
  });

  it('uses first IP from X-Forwarded-For through trusted proxy', () => {
    const result = extractClientIp(
      { 'x-forwarded-for': '198.51.100.5, 172.68.0.1' },
      '172.18.0.1'
    );
    expect(result.ip).toBe('198.51.100.5');
  });
});

describe('Telemetry Validation (Security)', () => {
  let service: any;

  beforeAll(async () => {
    const mockPool = { query: jest.fn().mockResolvedValue({ rows: [] }) };
    service = new ComplianceService(mockPool as any);
  });

  it('rejects client-provided user_id', () => {
    const result = service.validateTelemetry({ user_id: 'fake-uuid' });
    expect(result.valid).toBe(false);
    expect(result.errors).toContain("Field 'user_id' is not accepted from client");
  });

  it('rejects client-provided client_ip', () => {
    const result = service.validateTelemetry({ client_ip: '8.8.8.8' });
    expect(result.valid).toBe(false);
    expect(result.errors).toContain("Field 'client_ip' is not accepted from client");
  });

  it('rejects client-provided country', () => {
    const result = service.validateTelemetry({ country: 'US' });
    expect(result.valid).toBe(false);
  });

  it('rejects client-provided asn', () => {
    const result = service.validateTelemetry({ asn: 12345 });
    expect(result.valid).toBe(false);
  });

  it('accepts valid telemetry payload', () => {
    const result = service.validateTelemetry({
      user_agent: 'Mozilla/5.0',
      language: 'en-US',
      timezone: 'Asia/Dubai',
      timezone_offset_minutes: -240,
      screen: { width: 1920, height: 1080, avail_width: 1920, avail_height: 1040 },
      viewport: { width: 1365, height: 900 },
      device_pixel_ratio: 1,
      color_depth: 24,
      touch_points: 0,
      platform: 'Win32',
    });
    expect(result.valid).toBe(true);
    expect(result.sanitized.user_agent).toBe('Mozilla/5.0');
    expect(result.sanitized.timezone).toBe('Asia/Dubai');
  });

  it('handles missing browser fields gracefully', () => {
    const result = service.validateTelemetry({});
    expect(result.valid).toBe(true);
    expect(result.sanitized).toEqual({});
  });

  it('rejects oversized user_agent (>500 chars)', () => {
    const result = service.validateTelemetry({ user_agent: 'A'.repeat(501) });
    expect(result.sanitized.user_agent).toBeUndefined();
  });

  it('rejects impossible screen dimensions (>100000)', () => {
    const result = service.validateTelemetry({ screen: { width: 999999 } });
    expect(result.sanitized.screen?.width).toBe(100000);
  });

  it('limits client_hints to 20 fields', () => {
    const hints: Record<string, string> = {};
    for (let i = 0; i < 25; i++) hints[`key${i}`] = 'value';
    const result = service.validateTelemetry({ client_hints: hints });
    expect(Object.keys(result.sanitized.client_hints || {}).length).toBe(20);
  });
});
