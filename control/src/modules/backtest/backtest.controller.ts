import { Controller, Post, Get, Body, Param, Query, Res, UseGuards, HttpStatus, UnauthorizedException } from '@nestjs/common';
import { Response } from 'express';
import { ConfigService } from '@nestjs/config';
import { BacktestService } from './backtest.service';
import { RunBacktestDto } from './backtest.dto';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import * as jwt from 'jsonwebtoken';

@Controller('backtest')
export class BacktestController {
  constructor(
    private backtestService: BacktestService,
    private config: ConfigService,
  ) {}

  @UseGuards(JwtAuthGuard)
  @Get('data')
  async getAvailableData() {
    return this.backtestService.getAvailableData();
  }

  @UseGuards(JwtAuthGuard)
  @Get('runs')
  async listRuns(@Query('limit') limit?: string) {
    return this.backtestService.listRuns(limit ? parseInt(limit) : 20);
  }

  @UseGuards(JwtAuthGuard)
  @Get('runs/:runId')
  async getRunDetails(@Param('runId') runId: string) {
    const result = await this.backtestService.getRunDetails(runId);
    if (!result) {
      return { error: 'Run not found' };
    }
    return result;
  }

  @UseGuards(JwtAuthGuard)
  @Post('run')
  async runBacktest(
    @Body() dto: RunBacktestDto,
    @CurrentUser('sub') userId: string,
    @CurrentUser('role') role: string,
  ) {
    const isAdmin = role === 'ADMIN' || role === 'SUPER_ADMIN';
    return this.backtestService.runBacktest(dto, userId, isAdmin);
  }

  @Get('runs/:runId/download')
  async downloadRun(
    @Param('runId') runId: string,
    @Query('format') format: string,
    @Query('token') token: string,
    @Res() res: Response,
  ) {
    // Downloads are triggered via window.open() which cannot set an Authorization
    // header, so the frontend passes the JWT as a ?token= query param. Validate it
    // here instead of relying on JwtAuthGuard (which only reads the header).
    try {
      if (!token) throw new Error('missing token');
      jwt.verify(token, this.config.get<string>('JWT_SECRET') || '');
    } catch {
      throw new UnauthorizedException('Invalid or missing token');
    }
    if (format === 'csv') {
      const csv = await this.backtestService.getRunTradesCSV(runId);
      res.setHeader('Content-Type', 'text/csv');
      res.setHeader('Content-Disposition', `attachment; filename="backtest_${runId}_trades.csv"`);
      return res.send(csv);
    }
    return res.status(HttpStatus.BAD_REQUEST).json({ error: 'Unsupported format. Use ?format=csv' });
  }
}
