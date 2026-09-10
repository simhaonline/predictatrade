import { Module } from '@nestjs/common';
import { EmailCampaignController } from './email-campaigns.controller';
import { EmailCampaignService } from './email-campaigns.service';
import { DatabaseModule } from '../../common/database.module';

/**
 * EmailCampaignModule — admin email notifications to clients.
 *
 * EMAIL_SERVICE comes from the @Global MailModule (no import needed).
 */
@Module({
  imports: [DatabaseModule],
  controllers: [EmailCampaignController],
  providers: [EmailCampaignService],
})
export class EmailCampaignModule {}