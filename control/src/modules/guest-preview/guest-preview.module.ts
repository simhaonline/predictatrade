import { Module } from '@nestjs/common';
import { GuestPreviewService } from './guest-preview.service';
import { GuestPreviewController } from './guest-preview.controller';
import { DatabaseModule } from '../../common/database.module';
import { MailModule } from '../../common/mail/mail.module';

@Module({
  imports: [DatabaseModule, MailModule],
  controllers: [GuestPreviewController],
  providers: [GuestPreviewService],
  exports: [GuestPreviewService],
})
export class GuestPreviewModule {}
