import { Module } from '@nestjs/common';
import { DeviceAuthController } from './device-auth.controller';
import { DeviceAuthService } from './device-auth.service';
import { DatabaseModule } from '../../common/database.module';

@Module({
  imports: [DatabaseModule],
  controllers: [DeviceAuthController],
  providers: [DeviceAuthService],
})
export class DeviceAuthModule {}
