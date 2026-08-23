import { Controller, Post, Get, Body, Query, UseGuards, Req, HttpCode, HttpStatus } from '@nestjs/common';
import { ComplianceService } from './compliance.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { Throttle } from '@nestjs/throttler';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { ComplianceService as svc, extractClientIp } from './compliance.service';

@Controller('compliance')
export class ComplianceController {
  constructor(private complianceService: ComplianceService) {}

  /**
   * POST /api/v1/compliance/telemetry
   * Accept client telemetry from authenticated users.
   * Rate limited to prevent flooding.
   * Server-authoritative fields are never accepted from client.
   */
  @Post('telemetry')
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
   * GET /api/v1/compliance/events
   * Admin-only: query audit events
   */
  @Get('events')
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
      endpoint: '/api/v1/compliance/events',
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
}
