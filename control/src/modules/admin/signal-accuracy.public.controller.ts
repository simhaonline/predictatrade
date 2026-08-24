import { Controller, Get, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminService } from './admin.service';

@Controller('signal-accuracy')
@UseGuards(JwtAuthGuard)
export class SignalAccuracyPublicController {
  constructor(private adminService: AdminService) {}

  @Get()
  get() {
    return this.adminService.getSignalAccuracy();
  }
}
