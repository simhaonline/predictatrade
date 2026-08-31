import { Module } from '@nestjs/common';
import { PlansService } from './plans.service';
import { PlansController } from './plans.controller';
import { PlansPublicController } from './plans-public.controller';
import { DatabaseModule } from '../../common/database.module';

@Module({
  imports: [DatabaseModule],
  controllers: [PlansController, PlansPublicController],
  providers: [PlansService],
})
export class PlansModule {}
