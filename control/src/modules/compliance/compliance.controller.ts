import { Controller, Post, Get, Body, Query, UseGuards, Req, HttpCode, HttpStatus } from '@nestjs/common';
import { ComplianceService } from './compliance.service';
import { GdprService } from './gdpr.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { Throttle } from '@nestjs/throttler';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { ComplianceLog } from '../../common/interceptors/compliance.interceptor';
import { ComplianceService as svc, extractClientIp } from './compliance.service';

@Controller('')
export class ComplianceController {
  constructor(
    private complianceService: ComplianceService,
    private gdprService: GdprService,
  ) {}

  /**
   * POST /api/v1/telemetry/client
   * Accept client telemetry from authenticated users.
   * Rate limited to prevent flooding.
   * Server-authoritative fields are never accepted from client.
   */
  @Post('telemetry/client')
  @UseGuards(JwtAuthGuard)
  @Throttle({ default: { limit: 10, ttl: 60000 } })
  @HttpCode(HttpStatus.CREATED)
  async receiveTelemetry(
    @Body() body: any,
    @Req() req: any,
    @CurrentUser('sub') userId: string,
  ) {
    // Validate telemetry payload
    const { valid, errors, sanitized } = this.complianceService.validateTelemetry(body);

    // Extract server-side IP (never trust client-provided IP)
    const headers = req.headers as Record<string, string>;
    const socketIp = req.socket?.remoteAddress || req.ip || '';
    const { ip, proxyChain } = extractClientIp(headers, socketIp);

    // Record the telemetry event
    await this.complianceService.recordEvent({
      event_type: 'TELEMETRY_RECEIVED',
      user_id: userId,
      request_id: req.headers['x-request-id'] || undefined,
      correlation_id: req.correlationId || req.headers['x-correlation-id'] || undefined,
      http_method: req.method,
      endpoint: req.path,
      client_ip: ip,
      proxy_chain: proxyChain,
      user_agent: req.headers['user-agent'] || '',
      client_telemetry: sanitized,
      metadata: { validation_errors: errors },
      risk_flags: valid ? undefined : { invalid_telemetry: true },
    });

    return { status: 'recorded', valid };
  }

  /**
   * GET /api/v1/audit/events
   * Admin-only: query audit events
   */
  @Get('audit/events')
  @UseGuards(JwtAuthGuard, AdminGuard)
  async listEvents(
    @Query('page') page = '1',
    @Query('limit') limit = '50',
    @Req() req: any,
    @Query('userId') userId?: string,
    @Query('eventType') eventType?: string,
  ) {
    // Audit the admin viewing audit data
    const headers = req.headers as Record<string, string>;
    const socketIp = req.socket?.remoteAddress || '';
    const { ip, proxyChain } = extractClientIp(headers, socketIp);

    await this.complianceService.recordEvent({
      event_type: 'ADMIN_AUDIT_VIEW',
      user_id: req.user?.sub,
      http_method: 'GET',
      endpoint: '/api/v1/audit/events',
      client_ip: ip,
      proxy_chain: proxyChain,
      user_agent: req.headers['user-agent'] || '',
      metadata: { query: { page, limit, userId, eventType } },
    });

    return this.complianceService.listEvents(
      parseInt(page),
      Math.min(parseInt(limit), 200),
      { userId, eventType },
    );
  }

  /**
   * POST /api/v1/compliance/gdpr/anonymize
   * Admin-only: anonymize PII for a single user (account stays usable).
   */
  @Post('compliance/gdpr/anonymize')
  @UseGuards(JwtAuthGuard, AdminGuard)
  @ComplianceLog('GDPR_USER_ANONYMIZED')
  @HttpCode(HttpStatus.OK)
  async anonymizeUser(
    @Body('userId') userId: string,
    @CurrentUser('sub') actorId: string,
  ) {
    return this.gdprService.anonymizeUser(userId, actorId);
  }

  /**
   * POST /api/v1/compliance/gdpr/erase
   * Admin-only: erase (anonymize + lock) a single user's PII.
   */
  @Post('compliance/gdpr/erase')
  @UseGuards(JwtAuthGuard, AdminGuard)
  @ComplianceLog('GDPR_USER_ERASURE')
  @HttpCode(HttpStatus.OK)
  async eraseUser(
    @Body('userId') userId: string,
    @CurrentUser('sub') actorId: string,
  ) {
    return this.gdprService.eraseUser(userId, actorId);
  }

  /**
   * POST /api/v1/compliance/gdpr/retention
   * Admin-only: anonymize audit/client telemetry PII older than `days`
   * (default 365). Run on a schedule for GDPR retention compliance.
   */
  @Post('compliance/gdpr/retention')
  @UseGuards(JwtAuthGuard, AdminGuard)
  @ComplianceLog('GDPR_RETENTION_RUN')
  @HttpCode(HttpStatus.OK)
  async applyRetention(
    @Body('days') days?: number,
    @CurrentUser('sub') actorId?: string,
  ) {
    return this.gdprService.applyRetention(days, actorId);
  }
}
