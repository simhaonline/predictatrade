import {
  Injectable, NestInterceptor, ExecutionContext, CallHandler, Inject, Optional,
} from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { Observable, tap } from 'rxjs';
import { Pool } from 'pg';
import { DB_POOL } from '../database.module';
import { ComplianceService, extractClientIp } from '../../modules/compliance/compliance.service';

// Metadata key to mark endpoints for compliance logging
export const COMPLIANCE_EVENT_TYPE = 'compliance:event_type';

// Set via decorator: @ComplianceLog('AUTH_LOGIN')
export function ComplianceLog(eventType: string): MethodDecorator {
  return (target: any, propertyKey: string | symbol, descriptor: PropertyDescriptor) => {
    Reflect.defineMetadata(COMPLIANCE_EVENT_TYPE, eventType, descriptor.value);
    return descriptor;
  };
}

/**
 * Compliance Interceptor — automatically records audit.client_events for
 * important API endpoints decorated with @ComplianceLog.
 * 
 * Captures: client IP (server-side, trusted-proxy aware), HTTP method, 
 * endpoint, status, latency, user agent, and user ID from JWT.
 * 
 * Never blocks the request — failures are logged but not thrown.
 */
@Injectable()
export class ComplianceInterceptor implements NestInterceptor {
  constructor(
    private reflector: Reflector,
    @Optional() @Inject(DB_POOL) private pool: Pool,
  ) {}

  intercept(context: ExecutionContext, next: CallHandler): Observable<any> {
    const handler = context.getHandler();
    const eventType = this.reflector.get<string>(COMPLIANCE_EVENT_TYPE, handler);

    // Only log endpoints marked with @ComplianceLog
    if (!eventType || !this.pool) {
      return next.handle();
    }

    const request = context.switchToHttp().getRequest();
    const startTime = Date.now();

    return next.handle().pipe(
      tap({
        next: () => this.recordEvent(eventType, request, Date.now() - startTime, 200),
        error: (err) => {
          const status = err?.status || 500;
          this.recordEvent(eventType, request, Date.now() - startTime, status);
        },
      }),
    );
  }

  private async recordEvent(
    eventType: string,
    request: any,
    latencyMs: number,
    httpStatus: number,
  ): Promise<void> {
    try {
      const headers = request.headers || {};
      const socketIp = request.socket?.remoteAddress || request.ip || '';
      const { ip, proxyChain } = extractClientIp(headers, socketIp);
      const userId = request.user?.sub || null;

      const eventId = crypto.randomUUID();
      const now = new Date();
      const ua = headers['user-agent'] || '';

      // Parse user agent
      const browserInfo = this.parseUA(ua);

      await this.pool.query(
        `INSERT INTO audit.client_events (
          event_id, event_time, event_type, event_version,
          request_id, user_id,
          http_method, endpoint, http_status, latency_ms,
          client_ip, user_agent,
          browser_name, browser_version, os_name, os_version, device_type
        ) VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
        [
          eventId, now, eventType,
          headers['x-request-id'] || null,
          userId,
          request.method, request.path, httpStatus, latencyMs,
          ip, ua,
          browserInfo.browser, browserInfo.version, browserInfo.os, browserInfo.osVersion, browserInfo.deviceType,
        ],
      );
    } catch {
      // Audit failure must never block the request
    }
  }

  private parseUA(ua: string): { browser: string; version: string; os: string; osVersion: string; deviceType: string } {
    let browser = 'unknown', version = '', os = 'unknown', osVersion = '', deviceType = 'desktop';
    if (!ua) return { browser, version, os, osVersion, deviceType };
    if (ua.includes('Edg/')) { browser = 'Edge'; version = ua.match(/Edg\/([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Chrome/')) { browser = 'Chrome'; version = ua.match(/Chrome\/([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Firefox/')) { browser = 'Firefox'; version = ua.match(/Firefox\/([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Safari/') && !ua.includes('Chrome')) { browser = 'Safari'; version = ua.match(/Version\/([\d.]+)/)?.[1] || ''; }
    if (ua.includes('Windows')) { os = 'Windows'; osVersion = ua.match(/Windows NT ([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('Mac OS')) { os = 'macOS'; osVersion = ua.match(/Mac OS X ([\d_]+)/)?.[1]?.replace(/_/g, '.') || ''; }
    else if (ua.includes('Android')) { os = 'Android'; osVersion = ua.match(/Android ([\d.]+)/)?.[1] || ''; }
    else if (ua.includes('iPhone') || ua.includes('iPad')) { os = 'iOS'; osVersion = ua.match(/OS ([\d_]+)/)?.[1]?.replace(/_/g, '.') || ''; }
    else if (ua.includes('Linux')) { os = 'Linux'; }
    if (ua.includes('Mobile') || ua.includes('iPhone')) deviceType = 'mobile';
    else if (ua.includes('iPad') || ua.includes('Tablet')) deviceType = 'tablet';
    return { browser, version, os, osVersion, deviceType };
  }
}
