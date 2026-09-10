import { Module } from '@nestjs/common';
import { FeedbackController, FeedbackAdminController } from './feedback.controller';
import { FeedbackPublicController } from './feedback-public.controller';
import { FeedbackService } from './feedback.service';
import { DatabaseModule } from '../../common/database.module';

/**
 * FeedbackModule — customer feedback (user submit + admin moderate/feature + public showcase).
 */
@Module({
  imports: [DatabaseModule],
  controllers: [FeedbackController, FeedbackAdminController, FeedbackPublicController],
  providers: [FeedbackService],
})
export class FeedbackModule {}