import { Controller, Get, Param, Post, Body, UseGuards } from '@nestjs/common';
import { PlansService } from './plans.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';

@Controller('plans')
@UseGuards(JwtAuthGuard)
export class PlansController {
  constructor(private plansService: PlansService) {}

  @Get()
  async list() { return this.plansService.listActive(); }

  @Get(':id')
  async findById(@Param('id') id: string) { return this.plansService.findById(id); }

  @Patch(':id')
  @UseGuards(AdminGuard)
  async update(@Param('id') id: string, @Body() body: Record<string, any>) {
    return this.plansService.update(id, body);
  }
}
