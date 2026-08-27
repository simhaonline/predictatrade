import { Module } from '@nestjs/common';
import { ComplianceService } from './compliance.service';
import { GdprService } from './gdpr.service';
import { ComplianceController } from './compliance.controller';
import { DatabaseModule } from '../../common/database.module';

@Module({
  imports: [DatabaseModule],
  controllers: [ComplianceController],
  providers: [ComplianceService, GdprService],
  exports: [ComplianceService, GdprService],
})
export class ComplianceModule {}
