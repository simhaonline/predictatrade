import { Controller, Get, Query } from '@nestjs/common';
import { Throttle } from '@nestjs/throttler';
import { FeedbackService } from './feedback.service';

/**
 * FeedbackPublicController — unauthenticated marketing surface.
 *
 * GET /api/v1/feedback/public/featured
 *   Serves FEATURED + non-hidden feedback with FIRST NAME ONLY attribution
 *   (privacy: email/user_id never leave the database through this endpoint).
 *   Consumed by public pages (e.g. /feedback-showcase).
 */
@Controller('feedback/public')
export class FeedbackPublicController {
  constructor(private readonly feedback: FeedbackService) {}

  @Throttle({ default: { limit: 30, ttl: 60_000 } })
  @Get('featured')
  async featured(@Query('limit') limit?: string) {
    return this.feedback.listPublicFeatured(limit ? parseInt(limit, 10) : 10);
  }
}