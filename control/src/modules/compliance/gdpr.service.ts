import { Injectable, Inject, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import { createHash } from 'crypto';

export interface GdprResult {
  affectedUserId: string | null;
  userRows: number;
  clientEventRows: number;
}

/**
 * GDPR service — erasure / anonymization / retention for personal data.
 *
 * PII stores handled:
 *   - iam.users            (email, username, full_name, last_login_ip, password_hash)
 *   - audit.client_events  (client_ip, geo_*, isp, asn, as_org, user_agent, browser/os/device, languages, client_hints)
 *   - compliance.client_event_log (mirror PII store)
 *
 * Every operation writes a compliance-log entry to compliance.gdpr_operations
 * (and, as a fallback, audit.audit_events) so erasures are auditable.
 */
@Injectable()
export class GdprService {
  private readonly logger = new Logger(GdprService.name);

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /**
   * Anonymize PII for a single user but keep the account intact/usable.
   * Use this for "right to be forgotten" where hard deletion would break
   * financial/audit referential integrity.
   */
  async anonymizeUser(userId: string, actorId?: string): Promise<GdprResult> {
    const hashedEmail = this.anonymizedEmail(userId);
    const userRes = await this.pool.query(
      `UPDATE iam.users
         SET email = $2,
             email_verified = false,
             username = NULL,
             full_name = NULL,
             last_login_ip = NULL,
             updated_at = now()
       WHERE id = $1`,
      [userId, hashedEmail],
    );
    const clientEventRows = await this.anonymizeClientEventsForUser(userId);
    await this.logOperation('GDPR_USER_ANONYMIZED', userId, actorId, {
      userRows: userRes.rowCount ?? 0,
      clientEventRows,
    });
    return {
      affectedUserId: userId,
      userRows: userRes.rowCount ?? 0,
      clientEventRows,
    };
  }

  /**
   * Full erasure: anonymize PII AND lock the account (status DELETED,
   * password invalidated) so it can never be used to log in again.
   */
  async eraseUser(userId: string, actorId?: string): Promise<GdprResult> {
    const hashedEmail = this.anonymizedEmail(userId);
    const userRes = await this.pool.query(
      `UPDATE iam.users
         SET email = $2,
             email_verified = false,
             username = NULL,
             full_name = NULL,
             last_login_ip = NULL,
             password_hash = 'ERASED',
             status = 'DELETED',
             updated_at = now()
       WHERE id = $1`,
      [userId, hashedEmail],
    );
    const clientEventRows = await this.anonymizeClientEventsForUser(userId);
    await this.logOperation('GDPR_USER_ERASURE', userId, actorId, {
      userRows: userRes.rowCount ?? 0,
      clientEventRows,
    });
    return {
      affectedUserId: userId,
      userRows: userRes.rowCount ?? 0,
      clientEventRows,
    };
  }

  /**
   * Retention policy: anonymize PII in audit/client telemetry rows older than
   * `days` (default 365). Defensive-in-depth — TimescaleDB chunk retention may
   * later drop the rows entirely, but PII is scrubbed first.
   */
  async applyRetention(days: number = 365, actorId?: string): Promise<{ anonymizedRows: number }> {
    const cutoff = Number.isFinite(days) && days > 0 ? days : 365;
    const res = await this.pool.query(
      `UPDATE audit.client_events
         SET client_ip = NULL,
             proxy_chain = NULL,
             geo_country_code = NULL,
             geo_region = NULL,
             geo_city = NULL,
             isp = NULL,
             asn = NULL,
             as_org = NULL,
             user_agent = NULL,
             browser_name = NULL,
             browser_version = NULL,
             os_name = NULL,
             os_version = NULL,
             device_type = NULL,
             languages = NULL,
             client_hints = NULL
       WHERE event_time < now() - ($1::text || ' days')::interval`,
      [cutoff],
    );

    // Mirror store (best-effort; table may not exist in every deployment).
    try {
      await this.pool.query(
        `UPDATE compliance.client_event_log
           SET client_ip = NULL,
               proxy_chain = NULL,
               geo_country_code = NULL,
               geo_region = NULL,
               geo_city = NULL,
               isp = NULL,
               asn = NULL,
               as_org = NULL,
               user_agent = NULL,
               browser_name = NULL,
               browser_version = NULL,
               os_name = NULL,
               os_version = NULL,
               device_type = NULL,
               languages = NULL,
               client_hints = NULL
         WHERE event_time < now() - ($1::text || ' days')::interval`,
        [cutoff],
      );
    } catch {
      // compliance.client_event_log optional
    }

    const anonymizedRows = res.rowCount ?? 0;
    await this.logOperation('GDPR_RETENTION_RUN', null, actorId, {
      days: cutoff,
      anonymizedRows,
    });
    return { anonymizedRows };
  }

  private async anonymizeClientEventsForUser(userId: string): Promise<number> {
    const res = await this.pool.query(
      `UPDATE audit.client_events
         SET client_ip = NULL,
             proxy_chain = NULL,
             geo_country_code = NULL,
             geo_region = NULL,
             geo_city = NULL,
             isp = NULL,
             asn = NULL,
             as_org = NULL,
             user_agent = NULL,
             browser_name = NULL,
             browser_version = NULL,
             os_name = NULL,
             os_version = NULL,
             device_type = NULL,
             languages = NULL,
             client_hints = NULL
       WHERE user_id = $1`,
      [userId],
    );

    try {
      await this.pool.query(
        `UPDATE compliance.client_event_log
           SET client_ip = NULL,
               proxy_chain = NULL,
               geo_country_code = NULL,
               geo_region = NULL,
               geo_city = NULL,
               isp = NULL,
               asn = NULL,
               as_org = NULL,
               user_agent = NULL,
               browser_name = NULL,
               browser_version = NULL,
               os_name = NULL,
               os_version = NULL,
               device_type = NULL,
               languages = NULL,
               client_hints = NULL
         WHERE user_id = $1`,
        [userId],
      );
    } catch {
      // compliance.client_event_log optional
    }

    return res.rowCount ?? 0;
  }

  private anonymizedEmail(userId: string): string {
    const h = createHash('sha256').update(userId).digest('hex').slice(0, 16);
    return `gdpr-erased-${h}@anonymized.local`;
  }

  /**
   * Write a compliance-log entry for the operation. Primary target is the
   * dedicated compliance.gdpr_operations table; on failure we fall back to the
   * canonical audit.audit_events table so an erasure is never silent.
   */
  private async logOperation(
    operation: string,
    targetUserId: string | null,
    actorId: string | undefined,
    details: Record<string, unknown>,
  ): Promise<void> {
    try {
      await this.pool.query(
        `INSERT INTO compliance.gdpr_operations
           (operation, target_user_id, actor_id, details, rows_affected)
         VALUES ($1, $2, $3, $4, $5)`,
        [
          operation,
          targetUserId,
          actorId || null,
          JSON.stringify(details),
          (details.clientEventRows ?? details.anonymizedRows ?? 0) as number,
        ],
      );
    } catch (err) {
      this.logger.warn(
        `compliance.gdpr_operations unavailable, falling back to audit.audit_events: ${
          err instanceof Error ? err.message : 'unknown'
        }`,
      );
      try {
        await this.pool.query(
          `INSERT INTO audit.audit_events
             (actor_type, actor_id, action, entity_type, entity_id, new_value)
           VALUES ('ADMIN', $1, $2, 'gdpr', $3, $4)`,
          [actorId || null, operation, targetUserId, JSON.stringify(details)],
        );
      } catch (fallbackErr) {
        this.logger.error(
          `Failed to write GDPR compliance log: ${
            fallbackErr instanceof Error ? fallbackErr.message : 'unknown'
          }`,
        );
      }
    }
  }
}
