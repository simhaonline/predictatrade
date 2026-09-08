import { Module } from '@nestjs/common';
import { AdminController } from './admin.controller';
import { AdminService } from './admin.service';
import { SignalAccuracyPublicController } from './signal-accuracy.public.controller';
import { DatabaseModule } from '../../common/database.module';
import { CommissionsModule } from '../commissions/commissions.module';
import { LicensingModule } from '../licensing/licensing.module';
import { JwtModule } from '@nestjs/jwt';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { MailModule } from '../../common/mail/mail.module';

@Module({
  imports: [
    DatabaseModule,
    CommissionsModule,
    LicensingModule,
    MailModule,
    JwtModule.registerAsync({
      imports: [ConfigModule],
      useFactory: (config: ConfigService) => ({ secret: config.get<string>('JWT_SECRET') }),
      inject: [ConfigService],
    }),
  ],
  controllers: [AdminController, SignalAccuracyPublicController],
  providers: [AdminService],
})
export class AdminModule {}