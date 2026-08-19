import { Injectable, Inject, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class BillingService {
  private logger = new Logger(BillingService.name);
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async listInvoices(userId: string) {
    const r = await this.pool.query(
      `SELECT i.* FROM billing.invoices i
       JOIN billing.subscriptions s ON i.subscription_id = s.id
       WHERE s.user_id = $1 ORDER BY i.created_at DESC LIMIT 20`, [userId],
    );
    return r.rows;
  }

  async handleWebhook(body: any, headers: any) {
    this.logger.log(`Webhook received: ${body?.event_type || 'unknown'}`);
    return { received: true, eventType: body?.event_type || 'unknown' };
  }
}
