import { Controller, Get, Post, Body, UseGuards, Param, Query } from '@nestjs/common';
import { OperationsService } from './operations.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('operations')
@UseGuards(JwtAuthGuard, AdminGuard)
export class OperationsController {
  constructor(private opsService: OperationsService) {}

  @Get('state')
  async getTradingState() { return this.opsService.getTradingState(); }

  @Get('active')
  async getActiveOperations() { return this.opsService.getActiveOperations(); }

  @Post('halt-trading')
  async haltTrading(@CurrentUser('sub') userId: string, @Body() body: { reason: string }) {
    return this.opsService.haltTrading(userId, body.reason || 'admin_halt');
  }

  @Post('resume-trading')
  async resumeTrading(@CurrentUser('sub') userId: string, @Body() body: { reason: string }) {
    return this.opsService.resumeTrading(userId, body.reason || 'admin_resume');
  }

  @Post('pause-signals')
  async pauseSignals(@CurrentUser('sub') userId: string, @Body() body: { reason: string }) {
    return this.opsService.pauseSignals(userId, body.reason || 'admin_pause');
  }

  @Post('resume-signals')
  async resumeSignals(@CurrentUser('sub') userId: string, @Body() body: { reason: string }) {
    return this.opsService.resumeSignals(userId, body.reason || 'admin_resume');
  }

  @Post('strategy/:id/enable')
  async enableStrategy(@Param('id') id: string, @CurrentUser('sub') userId: string, @Body() body: { reason: string }) {
    return this.opsService.enableStrategy(id, userId, body.reason || 'admin_enable');
  }

  @Post('strategy/:id/disable')
  async disableStrategy(@Param('id') id: string, @CurrentUser('sub') userId: string, @Body() body: { reason: string }) {
    return this.opsService.disableStrategy(id, userId, body.reason || 'admin_disable');
  }

  // AI/ML endpoints
  @Get('ai/models')
  async listModels() { return this.opsService.listModels(); }

  @Get('ai/training-jobs')
  async listTrainingJobs() { return this.opsService.listTrainingJobs(); }

  @Get('ai/inference')
  async listInference(@Query('limit') limit?: string) {
    return this.opsService.listInferenceHistory(limit ? parseInt(limit, 10) : 50);
  }

  @Post('ai/model/:id/activate')
  async activateModel(@Param('id') id: string, @CurrentUser('sub') userId: string) {
    return this.opsService.activateModel(id, userId);
  }

  @Post('ai/model/:id/deactivate')
  async deactivateModel(@Param('id') id: string, @CurrentUser('sub') userId: string) {
    return this.opsService.deactivateModel(id, userId);
  }
}
