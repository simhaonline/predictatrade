import { Injectable, Inject, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import { AuditEventInput, ClientTelemetry } from './compliance.types';

// Redaction utility — strips sensitive data from any object
const SENSITIVE_KEYS = ['password', 'token', 'secret', 'authorization', 'cookie', 'api_key', 'apiKey', 'refresh_token', 'accessToken', 'private_key'];

function redact(obj: Record<string, any>): Record<string, any> {
  const result: Record<string, any> = {};
  for (const [key, value] of Object.entries(obj)) {
    if (SENSITIVE_KEYS.some(s => key.toLowerCase().includes(s.toLowerCase()))) {
      result[key] = '[REDACTED]';
    } else if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
      result[key] = redact(value);
    } else {
      result[key] = value;
    }
  }
  return result;
}

// Trusted proxy IP extraction
const TRUSTED_PROXY_CIDRS = (process.env.TRUSTED_PROXY_CIDRS || '172.16.0.0/12,10.0.0.0/8,127.0.0.0/8').split(',');

function isTrustedProxy(ip: string): boolean {
  // Check if IP is in trusted private/internal network ranges
  // Docker containers use 172.16-31.x.x, 10.x.x.x, 127.x.x.x
  if (!ip) return false;
  
  // Strip IPv6 prefix if present (e.g., ::ffff:172.18.0.1)
  const cleanIp = ip.replace(/^::ffff:/, '');
  
  for (const cidr of TRUSTED_PROXY_CIDRS) {
    const [network, prefix] = cidr.split('/');
    const prefixLen = parseInt(prefix || '32');
    
    // Simple check: compare first N octets for IPv4
    const netParts = network.split('.').map(Number);
    const ipParts = cleanIp.split('.').map(Number);
    
    if (netParts.length === 4 && ipParts.length === 4) {
      const octetsToCheck = Math.ceil(prefixLen / 8);
      let match = true;
      for (let i = 0; i < octetsToCheck; i++) {
        const mask = prefixLen % 8 === 0 || i < Math.floor(prefixLen / 8) ? 255 : (256 - Math.pow(2, 8 - (prefixLen % 8)));
        if ((netParts[i] & mask) !== (ipParts[i] & mask)) {
          match = false;
          break;
        }
      }
      if (match) return true;
    }
  }
  return false;
}

export function extractClientIp(headers: Record<string, string>, socketIp: string): { ip: string; proxyChain: string[] } {
  const proxyChain: string[] = [];

  // Check Cloudflare first (if from trusted proxy)
  const cfConnectingIp = headers['cf-connecting-ip'];
  const xRealIp = headers['x-real-ip'];
  const xForwardedFor = headers['x-forwarded-for'];

  if (isTrustedProxy(socketIp)) {
    // Request came through trusted proxy
    if (xForwardedFor) {
      proxyChain.push(...xForwardedFor.split(',').map(s => s.trim()));
    }
    // Use the leftmost non-trusted IP from X-Forwarded-For, or CF-Connecting-IP
    if (cfConnectingIp) {
      return { ip: cfConnectingIp, proxyChain };
    }
    if (xRealIp) {
      return { ip: xRealIp, proxyChain };
    }
    if (xForwardedFor) {
      const ips = xForwardedFor.split(',').map(s => s.trim());
      // Return the first IP (original client)
      return { ip: ips[0] || socketIp, proxyChain };
    }
  }

  // Direct connection — use socket IP
  return { ip: socketIp, proxyChain };
}

@Injectable()
export class ComplianceService {
  private readonly logger = new Logger(ComplianceService.name);

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /**
   * Record a compliance audit event to audit.client_events (TimescaleDB hypertable)
   * and optionally to audit.audit_events for backward compatibility.
   */
  async recordEvent(input: AuditEventInput): Promise<void> {
    try {
      const eventId = crypto.randomUUID();
      const now = new Date();
      const telemetry = input.client_telemetry || {};

      // Parse user agent
      const ua = input.user_agent || '';
      const browserInfo = this.parseUserAgent(ua);

      // Insert into audit.client_events
      await this.pool.query(
        `INSERT INTO audit.client_events (
          event_id, event_time, event_type, event_version, telemetry_schema_version,
          request_id, correlation_id, user_id, account_id, session_id,
          source, http_method, endpoint, http_status, latency_ms,
          client_ip, proxy_chain,
          geo_country_code, geo_region, geo_city, isp, asn, as_org,
          user_agent, browser_name, browser_version, os_name, os_version, device_type,
          language, languages, timezone, timezone_offset_minutes,
          screen_width, screen_height, screen_available_width, screen_available_height,
          viewport_width, viewport_height, device_pixel_ratio, color_depth, touch_points,
          client_hints, prediction_id, application_version, api_version,
          risk_flags, metadata
        ) VALUES (
          $1, $2, $3, $4, 1,
          $5, $6, $7, $8, $9,
          $10, $11, $12, $13, $14,
          $15, $16,
          $17, $18, $19, $20, $21, $22,
          $23, $24, $25, $26, $27, $28,
          $29, $30, $31, $32,
          $33, $34, $35, $36,
          $37, $38, $39, $40, $41,
          $42, $43, $44, $45,
          $46, $47
        )`,
        [
          eventId, now, input.event_type, input.event_version || 1,
          input.request_id, input.correlation_id, input.user_id, input.account_id, input.session_id,
          'nestjs', input.http_method, input.endpoint, input.http_status, input.latency_ms,
          input.client_ip, JSON.stringify(input.proxy_chain || []),
          input.geo?.country_code || null, input.geo?.region || null, input.geo?.city || null,
          input.geo?.isp || null, input.geo?.asn || null, input.geo?.as_org || null,
          ua, browserInfo.browser, browserInfo.version, browserInfo.os, browserInfo.osVersion, browserInfo.deviceType,
          telemetry.language || null, telemetry.languages ? JSON.stringify(telemetry.languages) : null,
          telemetry.timezone || null, telemetry.timezone_offset_minutes || null,
          telemetry.screen?.width || null, telemetry.screen?.height || null,
          telemetry.screen?.avail_width || null, telemetry.screen?.avail_height || null,
          telemetry.viewport?.width || null, telemetry.viewport?.height || null,
          telemetry.device_pixel_ratio || null, telemetry.color_depth || null, telemetry.touch_points || null,
          telemetry.client_hints ? JSON.stringify(telemetry.client_hints) : null,
          input.prediction_id || null,
          process.env.npm_package_version || '1.0.0', 'v1',
          input.risk_flags ? JSON.stringify(redact(input.risk_flags)) : null,
          input.metadata ? JSON.stringify(redact(input.metadata)) : null,
        ],
      );


    } catch (err) {
      // Audit failure must not crash the application (non-critical telemetry)
      this.logger.error(`Failed to record compliance event: ${err instanceof Error ? err.message : 'unknown'}`);
    }
  }

  /**
   * Validate client telemetry payload
   */
  validateTelemetry(telemetry: any): { valid: boolean; errors: string[]; sanitized: ClientTelemetry } {
    const errors: string[] = [];
    const sanitized: ClientTelemetry = {};

    if (typeof telemetry !== 'object' || telemetry === null) {
      return { valid: false, errors: ['Telemetry must be an object'], sanitized: {} };
    }

    // Validate string fields
    if (telemetry.user_agent && typeof telemetry.user_agent === 'string' && telemetry.user_agent.length <= 500) {
      sanitized.user_agent = telemetry.user_agent;
    }
    if (telemetry.language && typeof telemetry.language === 'string' && telemetry.language.length <= 20) {
      sanitized.language = telemetry.language;
    }
    if (telemetry.languages && Array.isArray(telemetry.languages) && telemetry.languages.length <= 10) {
      sanitized.languages = telemetry.languages.filter((l: any) => typeof l === 'string' && l.length <= 20).slice(0, 10);
    }
    if (telemetry.timezone && typeof telemetry.timezone === 'string' && telemetry.timezone.length <= 50) {
      sanitized.timezone = telemetry.timezone;
    }
    if (typeof telemetry.timezone_offset_minutes === 'number' && telemetry.timezone_offset_minutes >= -1440 && telemetry.timezone_offset_minutes <= 1440) {
      sanitized.timezone_offset_minutes = telemetry.timezone_offset_minutes;
    }
    if (telemetry.platform && typeof telemetry.platform === 'string' && telemetry.platform.length <= 100) {
      sanitized.platform = telemetry.platform;
    }

    // Validate screen dimensions
    if (telemetry.screen && typeof telemetry.screen === 'object') {
      sanitized.screen = {
        width: Math.min(Math.max(Number(telemetry.screen.width) || 0, 0), 100000),
        height: Math.min(Math.max(Number(telemetry.screen.height) || 0, 0), 100000),
        avail_width: Math.min(Math.max(Number(telemetry.screen.avail_width) || 0, 0), 100000),
        avail_height: Math.min(Math.max(Number(telemetry.screen.avail_height) || 0, 0), 100000),
      };
    }

    // Validate viewport dimensions
    if (telemetry.viewport && typeof telemetry.viewport === 'object') {
      sanitized.viewport = {
        width: Math.min(Math.max(Number(telemetry.viewport.width) || 0, 0), 100000),
        height: Math.min(Math.max(Number(telemetry.viewport.height) || 0, 0), 100000),
      };
    }

    // Validate device pixel ratio
    if (typeof telemetry.device_pixel_ratio === 'number' && telemetry.device_pixel_ratio > 0 && telemetry.device_pixel_ratio <= 10) {
      sanitized.device_pixel_ratio = telemetry.device_pixel_ratio;
    }

    // Validate color depth
    if (typeof telemetry.color_depth === 'number' && telemetry.color_depth > 0 && telemetry.color_depth <= 64) {
      sanitized.color_depth = telemetry.color_depth;
    }

    // Validate touch points
    if (typeof telemetry.touch_points === 'number' && telemetry.touch_points >= 0 && telemetry.touch_points <= 100) {
      sanitized.touch_points = telemetry.touch_points;
    }

    // Validate client hints (limited size)
    if (telemetry.client_hints && typeof telemetry.client_hints === 'object' && !Array.isArray(telemetry.client_hints)) {
      const hints: Record<string, string | boolean | number> = {};
      let hintCount = 0;
      for (const [k, v] of Object.entries(telemetry.client_hints)) {
        if (hintCount >= 20) break; // Max 20 hints
        if (typeof v === 'string' && v.length <= 200) hints[k] = v;
        else if (typeof v === 'boolean') hints[k] = v;
        else if (typeof v === 'number') hints[k] = v;
        hintCount++;
      }
      sanitized.client_hints = hints;
    }

    // Reject server-authoritative fields if client tries to send them
    const forbiddenFields = ['client_ip', 'country', 'isp', 'asn', 'user_id', 'account_id', 'authenticated', 'request_id', 'security_flags'];
    for (const field of forbiddenFields) {
      if (field in telemetry) {
        errors.push(`Field '${field}' is not accepted from client`);
      }
    }

    return { valid: errors.length === 0, errors, sanitized };
  }

  private parseUserAgent(ua: string): { browser: string; version: string; os: string; osVersion: string; deviceType: string } {
    let browser = 'unknown';
    let version = '';
    let os = 'unknown';
    let osVersion = '';
    let deviceType = 'desktop';

    if (!ua) return { browser, version, os, osVersion, deviceType };

    // Browser detection
    if (ua.includes('Edg/')) { browser = 'Edge'; version = ua.match(/Edg\/([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Chrome/')) { browser = 'Chrome'; version = ua.match(/Chrome\/([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Firefox/')) { browser = 'Firefox'; version = ua.match(/Firefox\/([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Safari/') && !ua.includes('Chrome')) { browser = 'Safari'; version = ua.match(/Version\/([\d.]+)/)?.[1] || ''; }

    // OS detection
    if (ua.includes('Windows')) { os = 'Windows'; osVersion = ua.match(/Windows NT ([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Mac OS')) { os = 'macOS'; osVersion = ua.match(/Mac OS X ([\d_]+)/)?.[1]?.replace(/_/g, '.') || ''; }
    else if (ua.includes('Linux')) { os = 'Linux'; }
    else if (ua.includes('Android')) { os = 'Android'; osVersion = ua.match(/Android ([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('iPhone') || ua.includes('iPad')) { os = 'iOS'; osVersion = ua.match(/OS ([\d_]+)/)?.[1]?.replace(/_/g, '.') || ''; }

    // Device type
    if (ua.includes('Mobile') || ua.includes('iPhone')) deviceType = 'mobile';
    else if (ua.includes('iPad') || ua.includes('Tablet')) deviceType = 'tablet';

    return { browser, version, os, osVersion, deviceType };
  }

  /**
   * Query audit events (admin only)
   */
  async listEvents(page: number, limit: number, filter?: { userId?: string; eventType?: string }) {
    const offset = (page - 1) * limit;
    let query = `SELECT * FROM audit.client_events WHERE 1=1`;
    const params: any[] = [];
    let paramIdx = 1;

    if (filter?.userId) {
      query += ` AND user_id = $${paramIdx++}`;
      params.push(filter.userId);
    }
    if (filter?.eventType) {
      query += ` AND event_type = $${paramIdx++}`;
      params.push(filter.eventType);
    }

    query += ` ORDER BY event_time DESC LIMIT $${paramIdx++} OFFSET $${paramIdx++}`;
    params.push(limit, offset);

    const [data, count] = await Promise.all([
      this.pool.query(query, params),
      this.pool.query(`SELECT count(*) as total FROM audit.client_events`),
    ]);

    return { items: data.rows, total: count.rows[0].total, page, limit };
  }
}
