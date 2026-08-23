import { Controller, Get, Query, UseGuards } from '@nestjs/common';
import { AuditService } from './audit.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('audit')
@UseGuards(JwtAuthGuard)
export class AuditController {
  constructor(private auditService: AuditService) {}

  @UseGuards(AdminGuard)
  @Get()
  async list(@Query('page') page = '1', @Query('limit') limit = '50') {
    return this.auditService.list(parseInt(page), parseInt(limit));
  }

  /**
   * Client-scoped activity log. Returns only events belonging to the
   * authenticated client (actor_id = current user). No admin role required.
   */
  @Get('client')
  async listClient(
    @CurrentUser('sub') userId: string,
    @Query('page') page = '1',
    @Query('limit') limit = '50',
  ) {
    return this.auditService.listForClient(userId, parseInt(page), parseInt(limit));
  }
}
