import { Controller, Get, Put, Param, Body, UseGuards } from '@nestjs/common';
import { FeatureFlagsService } from './feature-flags.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';

@Controller('admin/feature-flags')
@UseGuards(JwtAuthGuard, AdminGuard)
export class FeatureFlagsController {
  constructor(private service: FeatureFlagsService) {}

  @Get()
  async list() {
    return this.service.listFlags();
  }

  @Put(':id')
  async update(
    @Param('id') id: string,
    @Body() body: { mode?: string; reason?: string; is_enabled?: boolean; set_by?: string },
  ) {
    return this.service.updateFlag(id, body);
  }
}
