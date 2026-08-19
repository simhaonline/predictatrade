import { Controller, Get, Query, UseGuards } from '@nestjs/common';
import { CommissionsService } from './commissions.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('commissions')
@UseGuards(JwtAuthGuard)
export class CommissionsController {
  constructor(private commissionsService: CommissionsService) {}

  @Get()
  async list(@CurrentUser('sub') userId: string) { return this.commissionsService.findByRecipient(userId); }

  @Get('summary')
  async summary(@CurrentUser('sub') userId: string) { return this.commissionsService.getSummary(userId); }

  @UseGuards(AdminGuard)
  @Get('admin/all')
  async listAll(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.commissionsService.listAll(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @UseGuards(AdminGuard)
  @Get('admin/summary')
  async adminSummary() { return this.commissionsService.getGlobalSummary(); }
}
