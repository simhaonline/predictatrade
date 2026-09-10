import { Controller, Get, Post, Param, Query, Body, UseGuards, Req } from '@nestjs/common';
import type { Request } from 'express';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard, Roles, Role, PermissionGuard, RequirePermissions, Permission } from '../../common/guards/roles.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { EmailCampaignService } from './email-campaigns.service';

/**
 * EmailCampaignController — admin email-notification campaigns.
 *
 * Endpoints (all admin-gated; sends require USER_MANAGE so only operators with
 * user-administration capability can mail the client base):
 *   GET  /admin/email-campaigns/audience      → live audience counts
 *   POST /admin/email-campaigns               → create draft
 *   POST /admin/email-campaigns/:id/send      → send (paced, retry-safe)
 *   GET  /admin/email-campaigns               → history
 *   GET  /admin/email-campaigns/:id           → detail + per-recipient status
 */
@Controller('admin/email-campaigns')
@UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
@Roles(Role.ADMIN, Role.SUPER_ADMIN)
export class EmailCampaignController {
  constructor(private readonly campaigns: EmailCampaignService) {}

  @Get('audience')
  async audience() {
    return this.campaigns.audienceCounts();
  }

  @Post()
  @RequirePermissions(Permission.USER_MANAGE)
  async create(
    @Body()
    body: { subject?: string; bodyHtml?: string; bodyText?: string; campaignType?: string; audience?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.campaigns.createDraft(actorId, {
      subject: body.subject ?? '',
      bodyHtml: body.bodyHtml ?? '',
      bodyText: body.bodyText ?? '',
      campaignType: body.campaignType ?? 'newsletter',
      audience: body.audience ?? 'all_clients',
    });
  }

  @Post(':id/send')
  @RequirePermissions(Permission.USER_MANAGE)
  async send(@Param('id') id: string, @CurrentUser('sub') actorId: string) {
    return this.campaigns.sendCampaign(actorId, id);
  }

  @Get()
  async list(@Query('limit') limit?: string) {
    return this.campaigns.listCampaigns(limit ? parseInt(limit, 10) : 50);
  }

  @Get(':id')
  async detail(@Param('id') id: string) {
    return this.campaigns.campaignDetail(id);
  }
}