import { Injectable, Inject, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class AuditService {
  private readonly logger = new Logger(AuditService.name);

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /**
   * List audit events with pagination.
   * Maps audit.audit_events columns to the frontend-expected shape.
   */
  async list(page: number, limit: number) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT e.id, e.event_id, e.actor_type, e.actor_id as user_id,
                e.action as event_type, e.entity_type, e.entity_id,
                e.request_id, e.timestamp as created_at,
                e.source_ip, e.user_agent,
                e.old_value, e.new_value, e.reason, e.correlation_id
         FROM audit.audit_events e
         ORDER BY e.timestamp DESC
         LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM audit.audit_events'),
    ]);

    // Build metadata field from old_value/new_value/reason for frontend compatibility
    const items = data.rows.map((row) => ({
      ...row,
      metadata: {
        old_value: row.old_value,
        new_value: row.new_value,
        reason: row.reason,
        entity_type: row.entity_type,
        entity_id: row.entity_id,
        request_id: row.request_id,
        correlation_id: row.correlation_id,
      },
    }));

    return {
      items,
      total: parseInt(count.rows[0].total, 10),
      page,
      limit,
    };
  }

  /**
   * List audit events belonging to a single authenticated client (actor_id = user).
   * Used by the client-facing Activity Log so a user only sees their own events.
   */
  async listForClient(userId: string, page = 1, limit = 50) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT e.id, e.event_id, e.actor_type, e.action as event_type, e.entity_type, e.entity_id,
                e.request_id, e.timestamp as created_at, e.source_ip,
                e.new_value, e.reason, e.correlation_id
         FROM audit.audit_events e
         WHERE e.actor_id = $1
         ORDER BY e.timestamp DESC
         LIMIT $2 OFFSET $3`,
        [userId, limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM audit.audit_events WHERE actor_id = $1', [userId]),
    ]);

    const items = data.rows.map((row) => ({
      ...row,
      metadata: {
        new_value: row.new_value,
        reason: row.reason,
        entity_type: row.entity_type,
        entity_id: row.entity_id,
      },
    }));

    return {
      items,
      total: parseInt(count.rows[0].total, 10),
      page,
      limit,
    };
  }

  /**
   * Log an audit event.
   * Uses the actual audit.audit_events table schema.
   */
  async log(userId: string, action: string, resource: string, details: any = {}) {
    const id = crypto.randomUUID();
    const eventId = crypto.randomUUID();
    await this.pool.query(
      `INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, entity_id, new_value, reason, timestamp)
       VALUES ($1, $2, 'USER', $3, $4, $5, $6, $7, $8, now())`,
      [
        id,
        eventId,
        userId,
        action,
        resource.split(':')[0] || 'unknown',
        resource.split(':')[1] || null,
        JSON.stringify(details),
        details.reason || null,
      ],
    );
    return { id, eventId };
  }

  /**
   * Log a system-level audit event (no user actor).
   */
  async logSystem(action: string, entityType: string, details: any = {}) {
    const id = crypto.randomUUID();
    const eventId = crypto.randomUUID();
    await this.pool.query(
      `INSERT INTO audit.audit_events (id, event_id, actor_type, action, entity_type, new_value, timestamp)
       VALUES ($1, $2, 'SYSTEM', $3, $4, $5, now())`,
      [id, eventId, action, entityType, JSON.stringify(details)],
    );
    return { id, eventId };
  }
}
