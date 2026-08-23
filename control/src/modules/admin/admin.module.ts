import { Module } from '@nestjs/common';
import { AdminController } from './admin.controller';
import { AdminService } from './admin.service';
import { DatabaseModule } from '../../common/database.module';
import { CommissionsModule } from '../commissions/commissions.module';

@Module({
  imports: [DatabaseModule, CommissionsModule],
  controllers: [AdminController],
  providers: [AdminService],
})
export class AdminModule {}
