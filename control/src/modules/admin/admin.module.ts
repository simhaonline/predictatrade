import { Module } from '@nestjs/common';
import { AdminController } from './admin.controller';
import { AdminService } from './admin.service';
import { SignalAccuracyPublicController } from './signal-accuracy.public.controller';
import { DatabaseModule } from '../../common/database.module';
import { CommissionsModule } from '../commissions/commissions.module';
import { LicensingModule } from '../licensing/licensing.module';

@Module({
  imports: [DatabaseModule, CommissionsModule, LicensingModule],
  controllers: [AdminController, SignalAccuracyPublicController],
  providers: [AdminService],
})
export class AdminModule {}