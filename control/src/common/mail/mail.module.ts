/**
 * MailModule — provides the EmailService based on EMAIL_PROVIDER env var.
 *
 * Currently supports:
 *   - "smtp" (default) → NodemailerEmailProvider
 *   - "console"        → logs to console (dev only)
 *
 * To add a new provider (SendGrid, SES, etc.), implement EmailService
 * and add a case in the useFactory below.
 */

import { Module, Global } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { EMAIL_SERVICE } from './email.service';
import type { EmailService } from './email.service';
import { NodemailerEmailProvider } from './nodemailer-email.provider';

@Global()
@Module({
  imports: [ConfigModule],
  providers: [
    {
      provide: EMAIL_SERVICE,
      inject: [ConfigService],
      useFactory: (config: ConfigService): EmailService => {
        const provider = config.get<string>('EMAIL_PROVIDER', 'smtp');
        switch (provider) {
          case 'smtp':
          case 'console':
          default:
            return new NodemailerEmailProvider(config);
        }
      },
    },
  ],
  exports: [EMAIL_SERVICE],
})
export class MailModule {}
