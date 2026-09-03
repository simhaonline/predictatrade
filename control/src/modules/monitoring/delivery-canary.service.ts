import * as http from 'http';
import { Injectable, Inject, Logger, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

/**
 * Delivery canary (2026-09-04).
 *
 * Every CANNERY_INTERVAL_MS, enqueue one `type:"CANARY"` queue item per
 * active device and verify it reaches ACKED within the window. The canary
 * is a non-trade, non-signal payload (no StrategyID → skipped by the
 * entitlement clause; `type:"CANARY"` → dispatched by both old and new EA
 * builds; `test:true` → never executed). It exercises the REAL delivery
 * path: edge_signal_queue → edge-poll claim → EA WebRequest poll → ACK.
 *
 * If the canary does not ACK within CANARY_TIMEOUT_MS:
 *   - WARNING: "delivery pipe degraded" — poll path or EA queue processing
 *     is broken for that device (distinct from the connectivity watchdog,
 *     which only proves the device POLLS, not that items flow through).
 *
 * Canary rows carry signal_id 'CANARY-<ts>' and are auto-cleaned after
 * ack/expiry so they never pollute reconciliation counters.
 */

const CANNERY_INTERVAL_MS = 10 * 60_000; // every 10 minutes
const CANNERY_TTL_SECS = 15 * 60; // canary dies if not polled in 15 min
const CANNERY_MAX_ATTEMPTS = 3;
const NOTIFY_COOLDOWN_MS = 30 * 60_000;

const NTFY_BASE = process.env.NTFY_URL || 'http://ntfy:80';
const NTFY_TOPIC = process.env.NTFY_DELIVERY_TOPIC || 'pat-connectivity';

@Injectable()
export class DeliveryCanaryService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(DeliveryCanaryService.name);
  private timer: NodeJS.Timeout | null = null;
  private running = false;

  constructor(@Inject(DB_POOL) private readonly pool: Pool) {}

  onModuleInit() {
    this.timer = setInterval(() => void this.runCycle(), CANNERY_INTERVAL_MS);
    setTimeout(() => void this.runCycle(), 45_000);
    this.logger.log('Delivery canary started (10 min interval)');
  }

  onModuleDestroy() {
    if (this.timer) clearInterval(this.timer);
  }

  private async runCycle() {
    if (this.running) return;
    this.running = true;
    try {
      await this.verifyPrevious();
      await this.enqueue();
    } catch (e) {
      this.logger.error(`Canary cycle failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      this.running = false;
    }
  }

  /** Enqueue one canary item for every device seen in the last 30 minutes. */
  private async enqueue() {
    const res = await this.pool.query(
      `INSERT INTO licensing.edge_signal_queue (device_id, signal_id, payload)
       SELECT d.id,
              'CANARY-' || to_char(now(), 'YYYYMMDDHH24MISS'),
              jsonb_build_object(
                'type', 'CANARY',
                'ID', 'CANARY-' || to_char(now(), 'YYYYMMDDHH24MISS'),
                'test', true,
                'ExpiresAt', to_char(now() + interval '15 minutes', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
              )
       FROM licensing.devices d
       WHERE d.revoked_at IS NULL
         AND d.last_seen_at > now() - interval '30 minutes'
         AND EXISTS (SELECT 1 FROM licensing.licenses l
                     WHERE l.id IN (COALESCE(d.bound_license_id, d.license_id))
                       AND l.status IN ('ACTIVE','TRIALING'))
       RETURNING signal_id`,
    );
    if (res.rows.length > 0) {
      this.logger.log(`Canary enqueued for ${res.rows.length} device(s): ${res.rows[0].signal_id}`);
    }
  }

  /** Check canaries from the previous window: ACKED = pipe healthy; stale = alert. */
  private async verifyPrevious() {
    const res = await this.pool.query(
      `SELECT q.device_id, q.signal_id, q.status, q.attempts,
              EXTRACT(EPOCH FROM (now() - q.created_at))::int AS age_secs
         FROM licensing.edge_signal_queue q
        WHERE q.signal_id LIKE 'CANARY-%'
          AND q.signal_id <> 'CANARY-' || to_char(now(), 'YYYYMMDDHH24MISS')
          AND q.created_at > now() - interval '2 hours'
        ORDER BY q.created_at DESC`,
    );
    const stale: string[] = [];
    const acked = new Set<string>();
    const seen = new Set<string>();
    for (const r of res.rows) {
      if (r.status === 'ACKED' || r.status === 'PROCESSED') acked.add(r.device_id);
      else if (
        Number(r.age_secs) > CANNERY_TTL_SECS &&
        Number(r.attempts) >= CANNERY_MAX_ATTEMPTS &&
        r.status !== 'EXPIRED'
      ) {
        // Polled at least 3 times over the TTL window but never acked —
        // the device receives items but its EA never processes them.
        stale.push(`${r.device_id} (signal ${r.signal_id}, attempts ${r.attempts})`);
      }
      seen.add(r.device_id);
    }
    if (stale.length > 0) {
      await this.raiseCanaryAlert(
        `Delivery canary UNACKED for ${stale.length} device(s): ${stale.join('; ')}. ` +
          `The device polls but its EA never processes queue items — the delivery pipe is broken end-to-end.`,
      );
    } else {
      await this.resolveCanaryAlert();
    }
    // Housekeeping: delete old canary rows (they are not signals).
    await this.pool.query(
      `DELETE FROM licensing.edge_signal_queue
        WHERE signal_id LIKE 'CANARY-%'
          AND (status IN ('ACKED','PROCESSED','EXPIRED')
               AND acked_at < now() - interval '1 hour'
               OR created_at < now() - interval '2 hours')`,
    );
    void acked; void seen;
  }

  private async raiseCanaryAlert(message: string) {
    await this.pool.query(
      `INSERT INTO system.connectivity_alerts
         (alert_key, severity, scope, message, status, occurrences, last_seen_at)
       VALUES ('DELIVERY:canary-stale', 'CRITICAL', 'DEVICE', $1, 'OPEN', 1, now())
       ON CONFLICT (alert_key) DO UPDATE
         SET message = EXCLUDED.message, severity = 'CRITICAL', status = 'OPEN',
             occurrences = system.connectivity_alerts.occurrences + 1,
             last_seen_at = now()
       RETURNING notified_at`,
      [message],
    );
    await this.pool.query(
      `UPDATE system.connectivity_alerts SET notified_at = now()
        WHERE alert_key = 'DELIVERY:canary-stale'
          AND (notified_at IS NULL OR notified_at < now() - ($1::text || ' milliseconds')::interval)`,
      [String(NOTIFY_COOLDOWN_MS)],
    );
    this.logger.warn(`ALERT [CRITICAL] DELIVERY:canary-stale: ${message}`);
    await this.notify(message);
  }

  private async resolveCanaryAlert() {
    await this.pool.query(
      `UPDATE system.connectivity_alerts
          SET status = 'RESOLVED', resolved_at = now()
        WHERE alert_key = 'DELIVERY:canary-stale' AND status = 'OPEN'`,
    );
  }

  private notify(message: string) {
    return new Promise<void>((resolve) => {
      const u = new URL(`/${NTFY_TOPIC}`, NTFY_BASE);
      const payload = JSON.stringify({
        topic: NTFY_TOPIC,
        title: '[CRITICAL] Predict-A-Trade delivery canary',
        message,
        priority: 'high',
        tags: ['rotating_light'],
      });
      const req = http.request(
        { hostname: u.hostname, port: u.port || 80, path: u.pathname, method: 'POST',
          headers: { 'Content-Type': 'application/json' }, timeout: 5000 },
        (res) => { res.resume(); resolve(); },
      );
      req.on('error', () => resolve());
      req.on('timeout', () => { req.destroy(); resolve(); });
      req.end(payload);
    });
  }
}