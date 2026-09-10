import { Module } from '@nestjs/common';
import { FeedbackController, FeedbackAdminController } from './feedback.controller';
import { FeedbackService } from './feedback.service';
import { DatabaseModule } from '../../common/database.module';

/**
 * FeedbackModule — customer feedback (user submit + admin moderate/feature).
 */
@Module({
  imports: [DatabaseModule],
  controllers: [FeedbackController, FeedbackAdminController],
  providers: [FeedbackService],
})
export class FeedbackModule {}