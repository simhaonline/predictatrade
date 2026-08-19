import { Controller, Get, Param, UseGuards } from '@nestjs/common';
import { PlansService } from './plans.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';

@Controller('plans')
@UseGuards(JwtAuthGuard)
export class PlansController {
  constructor(private plansService: PlansService) {}

  @Get()
  async list() { return this.plansService.listActive(); }

  @Get(':id')
  async findById(@Param('id') id: string) { return this.plansService.findById(id); }
}
