import { Module } from '@nestjs/common';
import { EdgePollController } from './edge-poll.controller';
import { EdgePollService } from './edge-poll.service';
import { DatabaseModule } from '../../common/database.module';
import { DeviceAuthModule } from '../device-auth/device-auth.module';

@Module({
  imports: [DatabaseModule, DeviceAuthModule],
  controllers: [EdgePollController],
  providers: [EdgePollService],
  exports: [EdgePollService],
})
export class EdgePollModule {}