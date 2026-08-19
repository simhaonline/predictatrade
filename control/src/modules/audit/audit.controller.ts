import { Controller, Get, Query, UseGuards } from '@nestjs/common';
import { AuditService } from './audit.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';

@Controller('audit')
@UseGuards(JwtAuthGuard)
export class AuditController {
  constructor(private auditService: AuditService) {}

  @UseGuards(AdminGuard)
  @Get()
  async list(@Query('page') page = '1', @Query('limit') limit = '50') {
    return this.auditService.list(parseInt(page), parseInt(limit));
  }
}
