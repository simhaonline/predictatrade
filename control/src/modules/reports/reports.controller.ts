import {
  BadRequestException,
  Controller,
  Get,
  Param,
  Query,
  Res,
  UseGuards,
} from '@nestjs/common';
import { Response } from 'express';
import { ReportsService, ReportFormat } from './reports.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function parseFormat(raw?: string): ReportFormat {
  const format = (raw ?? 'csv').toLowerCase();
  if (format === 'csv' || format === 'xlsx' || format === 'pdf') return format;
  throw new BadRequestException({ error: 'invalid_format', supported: ['csv', 'xlsx', 'pdf'] });
}

@Controller('reports')
@UseGuards(JwtAuthGuard)
export class ReportsController {
  constructor(private reportsService: ReportsService) {}

  /** GET /api/v1/reports/trading/self?format=csv|xlsx|pdf — caller's own trades. */
  @Get('trading/self')
  async self(
    @CurrentUser('sub') userId: string,
    @Query('format') format?: string,
    @Res() res?: Response,
  ) {
    const fmt = parseFormat(format);
    const file = await this.reportsService.generateReport(userId, fmt);
    await this.reportsService.auditReportGeneration(userId, userId, fmt, file.rowCount);
    res!.setHeader('Content-Type', file.contentType);
    res!.setHeader('Content-Disposition', `attachment; filename="${file.filename}"`);
    res!.send(file.buffer);
  }

  /** GET /api/v1/reports/admin/reports/trading/summary — per-user aggregates. */
  @UseGuards(AdminGuard)
  @Get('admin/reports/trading/summary')
  async adminSummary(@CurrentUser('sub') actorId: string) {
    const rows = await this.reportsService.getAllUsersSummary();
    return { generated_at: new Date().toISOString(), users: rows };
  }

  /** GET /api/v1/reports/admin/reports/trading/:userId?format=csv|xlsx|pdf */
  @UseGuards(AdminGuard)
  @Get('admin/reports/trading/:userId')
  async adminUser(
    @CurrentUser('sub') actorId: string,
    @Param('userId') userId: string,
    @Query('format') format?: string,
    @Res() res?: Response,
  ) {
    if (!UUID_RE.test(userId)) {
      throw new BadRequestException({ error: 'invalid_user_id' });
    }
    const fmt = parseFormat(format);
    const file = await this.reportsService.generateReport(userId, fmt);
    await this.reportsService.auditReportGeneration(actorId, userId, fmt, file.rowCount);
    res!.setHeader('Content-Type', file.contentType);
    res!.setHeader('Content-Disposition', `attachment; filename="${file.filename}"`);
    res!.send(file.buffer);
  }
}
