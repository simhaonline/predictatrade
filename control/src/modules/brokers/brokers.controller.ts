import { Controller, Get, Param, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { BrokersService } from './brokers.service';

@Controller('brokers')
@UseGuards(JwtAuthGuard)
export class BrokersController {
  constructor(private svc: BrokersService) {}

  @Get('account-types')
  async listAccountTypes() { return this.svc.listAccountTypes(); }

  @UseGuards(AdminGuard)
  @Get('admin/account-types')
  async listAll() { return this.svc.listAccountTypes(); }

  @Get('admin/strategy-gates')
  async listAllGates() { return this.svc.listAllGates(); }

  @Get('strategy-gates/:strategyId')
  async getGate(@Param('strategyId') id: string) { return this.svc.getStrategyGate(id); }
}
