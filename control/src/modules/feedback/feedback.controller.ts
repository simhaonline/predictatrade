import { Controller, Get, Post, Body, Param, Query, UseGuards, Req } from '@nestjs/common';
import type { Request } from 'express';
import { Throttle } from '@nestjs/throttler';
import { FeedbackService } from './feedback.service';
import { CreateFeedbackDto } from './dto/create-feedback.dto';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { RolesGuard, Roles, Role, PermissionGuard, RequirePermissions, Permission } from '../../common/guards/roles.guard';

/**
 * FeedbackController — user-facing submissions (dashboard).
 * POST /api/v1/feedback        submit (rate-limited 3/10min)
 * GET  /api/v1/feedback/mine   own submissions
 */
@Controller('feedback')
@UseGuards(JwtAuthGuard, RolesGuard)
export class FeedbackController {
  constructor(private readonly feedback: FeedbackService) {}

  @Throttle({ default: { limit: 3, ttl: 600_000 } })
  @Post()
  async submit(@Body() dto: CreateFeedbackDto, @CurrentUser('sub') userId: string) {
    return this.feedback.submit(userId, dto);
  }

  @Get('mine')
  async mine(@CurrentUser('sub') userId: string) {
    return this.feedback.listMine(userId);
  }
}

/**
 * FeedbackAdminController — moderation + Featured toggle (admin console).
 * GET  /api/v1/admin/feedback            list (status filter, paging)
 * GET  /api/v1/admin/feedback/stats      aggregate stats
 * GET  /api/v1/admin/feedback/featured   public featured list
 * POST /api/v1/admin/feedback/:id/status    moderate
 * POST /api/v1/admin/feedback/:id/featured  Featured toggle (the requirement)
 * POST /api/v1/admin/feedback/:id/note      admin note
 */
@Controller('admin/feedback')
@UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
@Roles(Role.ADMIN, Role.SUPER_ADMIN)
export class FeedbackAdminController {
  constructor(private readonly feedback: FeedbackService) {}

  @Get()
  async list(@Query('status') status?: string, @Query('page') page?: string, @Query('limit') limit?: string) {
    return this.feedback.adminList(status, page ? parseInt(page, 10) : 1, limit ? parseInt(limit, 10) : 20);
  }

  @Get('stats')
  async stats() {
    return this.feedback.adminStats();
  }

  @Get('featured')
  async featured(@Query('limit') limit?: string) {
    return this.feedback.listPublicFeatured(limit ? parseInt(limit, 10) : 10);
  }

  @Post(':id/status')
  @RequirePermissions(Permission.USER_MANAGE)
  async setStatus(
    @Param('id') id: string,
    @Body() body: { status?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.feedback.adminSetStatus(actorId, id, body?.status ?? '');
  }

  /** THE Featured requirement toggle. */
  @Post(':id/featured')
  @RequirePermissions(Permission.USER_MANAGE)
  async setFeatured(
    @Param('id') id: string,
    @Body() body: { featured?: boolean },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.feedback.adminSetFeatured(actorId, id, body?.featured as unknown as boolean);
  }

  @Post(':id/note')
  @RequirePermissions(Permission.USER_MANAGE)
  async setNote(
    @Param('id') id: string,
    @Body() body: { note?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.feedback.adminSetNote(actorId, id, body?.note ?? '');
  }
}