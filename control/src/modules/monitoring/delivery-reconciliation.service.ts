import * as http from 'http';
import { Injectable, Inject, Logger, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

/**
 * Delivery-reconciliation watchdog (2026-09-04).
 *
 * PERMANENT guardrail after the 2026-09-03 silent-drop incident: old MT5
 * client builds ACKed every queued signal PROCESSED with ack type:"" but
 * never executed them — ACKs reported SUCCESS while delivery was 100% dead.
 *
 * Lesson institutionalized here: **an ACK is not proof of delivery.**
 * This watchdog tracks, per active device:
 *   1. Dispatch contract — the last ACKed signal item's ack type. Old/healthy
 *      builds ack type:"SIGNAL"; an empty type on a signal item means the
 *      device could not dispatch it (silent drop) → CRITICAL alert.
 *   2. End-to-end throughput — delivered (SIGNAL-acked) vs dropped
 *      (empty-acked) counts per 24h per device.
 *   3. Enqueue backlog — PENDING items aging >5 min mean the device is not
 *      polling (or the poll path broke) → WARNING/CRITICAL.
 *
 * Alerts dedupe via system.connectivity_alerts (same lifecycle as the
 * connectivity watchdog): OPEN until the condition clears, ntfy push at
 * most once per cooldown per key.
 */

const CHECK_INTERVAL_MS = 60_000;
const NOTIFY_COOLDOWN_MS = 10 * 60_000;
const PENDING_WARN_SECS = 5 * 60; // PENDING item older than 5 min = suspect
const EMPTY_ACK_DROP_THRESHOLD = 3; // >3 empty-type signal ACKs/24h = dropping

const NTFY_BASE = process.env.NTFY_URL || 'http://ntfy:80';
const NTFY_TOPIC = process.env.NTFY_DELIVERY_TOPIC || 'pat-connectivity';

interface RaiseInput {
  alertKey: string;
  severity: 'INFO' | 'WARNING' | 'CRITICAL';
  message: string;
  deviceId?: string | null;
}

@Injectable()
export class DeliveryReconciliationService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(DeliveryReconciliationService.name);
  private timer: NodeJS.Timeout | null = null;
  private running = false;

  constructor(@Inject(DB_POOL) private readonly pool: Pool) {}

  onModuleInit() {
    this.timer = setInterval(() => void this.runChecks(), CHECK_INTERVAL_MS);
    setTimeout(() => void this.runChecks(), 30_000);
    this.logger.log('Delivery reconciliation watchdog started (60s interval)');
  }

  onModuleDestroy() {
    if (this.timer) clearInterval(this.timer);
  }

  private async runChecks() {
    if (this.running) return; // never overlap cycles
    this.running = true;
    try {
      await this.reconcile();
    } catch (e) {
      this.logger.error(`Delivery reconciliation cycle failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      this.running = false;
    }
  }

  /** One reconciliation pass over every device that received queue items recently. */
  async reconcile(): Promise<void> {
    const res = await this.pool.query(
      `WITH recent AS (
         SELECT q.device_id,
                max(q.acked_at) FILTER (
                  WHERE COALESCE(q.ack_result->>'type','') = 'SIGNAL'
                     OR (q.payload->>'type' = 'SIGNAL' AND COALESCE(q.ack_result->>'type','') <> '')) AS last_sig_ack_at,
                count(*) FILTER (WHERE q.ack_result->>'status' = 'PROCESSED'
                                  AND COALESCE(q.ack_result->>'type','') = ''
                                  AND (q.payload->>'type' = 'SIGNAL' OR (q.payload->>'ID' IS NOT NULL AND q.payload->>'test' IS NULL))
                                  AND q.acked_at > now() - interval '24 hours') AS empty_ack_24h,
                count(*) FILTER (WHERE COALESCE(q.ack_result->>'type','') = 'SIGNAL'
                                  AND q.acked_at > now() - interval '24 hours') AS sig_acked_24h,
                count(*) FILTER (WHERE q.status IN ('PENDING','IN_FLIGHT')) AS pending_count,
                EXTRACT(EPOCH FROM (now() - min(q.created_at) FILTER (WHERE q.status IN ('PENDING','IN_FLIGHT'))))::int
                  AS pending_oldest_secs
         FROM licensing.edge_signal_queue q
         WHERE q.created_at > now() - interval '24 hours'
         GROUP BY q.device_id
       ), last_sig AS (
         SELECT DISTINCT ON (q.device_id) q.device_id, q.ack_result->>'type' AS last_sig_ack_type
         FROM licensing.edge_signal_queue q
         WHERE q.ack_result->>'status' = 'PROCESSED'
           AND COALESCE(q.ack_result->>'type','') = 'SIGNAL'
         ORDER BY q.device_id, q.acked_at DESC NULLS LAST
       )
       SELECT r.*, COALESCE(l.last_sig_ack_type, '') AS last_sig_ack_type,
              d.device_name, u.email
       FROM recent r
       JOIN licensing.devices d ON d.id = r.device_id
       LEFT JOIN last_sig l ON l.device_id = r.device_id
       LEFT JOIN iam.users u ON u.id = d.user_id`,
    );

    for (const r of res.rows) {
      const deviceId: string = r.device_id;
      const emptyAck24h = Number(r.empty_ack_24h ?? 0);
      const sigAcked24h = Number(r.sig_acked_24h ?? 0);
      const pendingCount = Number(r.pending_count ?? 0);
      const pendingOldest = Number(r.pending_oldest_secs ?? 0);
      const lastAckType: string | null = r.last_sig_ack_type ?? null;

      // Upsert the reconciliation row (source of truth for the admin card).
      await this.pool.query(
        `INSERT INTO system.delivery_reconciliation
           (device_id, last_signal_ack_at, last_signal_ack_type,
            empty_ack_count_24h, pending_count, pending_oldest_secs,
            dropped_count_24h, delivered_count_24h, updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now())
         ON CONFLICT (device_id) DO UPDATE SET
           last_signal_ack_at = EXCLUDED.last_signal_ack_at,
           last_signal_ack_type = EXCLUDED.last_signal_ack_type,
           empty_ack_count_24h = EXCLUDED.empty_ack_count_24h,
           pending_count = EXCLUDED.pending_count,
           pending_oldest_secs = EXCLUDED.pending_oldest_secs,
           dropped_count_24h = EXCLUDED.dropped_count_24h,
           delivered_count_24h = EXCLUDED.delivered_count_24h,
           updated_at = now()`,
        [
          deviceId,
          r.last_sig_ack_at ?? null,
          lastAckType,
          emptyAck24h,
          pendingCount,
          pendingOldest ?? 0,
          emptyAck24h,
          sigAcked24h,
        ],
      );

      // 1) Silent-drop detector: signal items PROCESSED with an empty ack type.
      if (emptyAck24h > EMPTY_ACK_DROP_THRESHOLD) {
        await this.raiseAlert({
          alertKey: `DELIVERY:empty-acks:${deviceId}`,
          severity: 'CRITICAL',
          deviceId,
          message: `Device "${r.device_name}" (${r.email ?? 'unknown'}) ACKed ${emptyAck24h} signals with an EMPTY dispatch type in 24h — its EA build is not dispatching signals (silent drop). Recompile the EA (MT5 v1.26.1+/MT4 v1.27.1+) or the payload contract changed.`,
        });
      } else {
        await this.resolveAlert(`DELIVERY:empty-acks:${deviceId}`);
      }

      // 2) Enqueue backlog: items undelivered too long.
      if (pendingCount > 0 && pendingOldest > PENDING_WARN_SECS * 4) {
        await this.raiseAlert({
          alertKey: `DELIVERY:backlog:${deviceId}`,
          severity: 'CRITICAL',
          deviceId,
          message: `Device "${r.device_name}" has ${pendingCount} undelivered queue items, oldest ${Math.round(pendingOldest / 60)} min — it stopped polling or the edge-poll path broke. Signals are piling up unsent.`,
        });
      } else if (pendingCount > 0 && pendingOldest > PENDING_WARN_SECS) {
        await this.raiseAlert({
          alertKey: `DELIVERY:backlog:${deviceId}`,
          severity: 'WARNING',
          deviceId,
          message: `Device "${r.device_name}" has ${pendingCount} undelivered queue items, oldest ${Math.round(pendingOldest / 60)} min — polling may be stalled.`,
        });
      } else {
        await this.resolveAlert(`DELIVERY:backlog:${deviceId}`);
      }
    }

    // Auto-resolve backlog/empty-ack alerts for devices that no longer have
    // any recent queue activity at all (device decommissioned / idle market).
    const open = await this.pool.query(
      `SELECT alert_key FROM system.connectivity_alerts
       WHERE status = 'OPEN' AND alert_key LIKE 'DELIVERY:%'`,
    );
    const active = new Set<string>(res.rows.map((r: Record<string, unknown>) => String(r.device_id)));
    for (const row of open.rows) {
      const key = String(row.alert_key);
      const id = key.slice(key.lastIndexOf(':') + 1);
      if (!active.has(id)) await this.resolveAlert(key);
    }
  }

  /** Admin dashboard payload: per-device delivery reconciliation rows. */
  async getDeliverySnapshot() {
    const rows = await this.pool.query(
      `SELECT r.device_id, r.updated_at, r.last_signal_ack_at, r.last_signal_ack_type,
              r.empty_ack_count_24h, r.pending_count, r.pending_oldest_secs,
              r.dropped_count_24h, r.delivered_count_24h,
              d.device_name, u.email
       FROM system.delivery_reconciliation r
       JOIN licensing.devices d ON d.id = r.device_id
       LEFT JOIN iam.users u ON u.id = d.user_id
       ORDER BY r.dropped_count_24h DESC, r.pending_oldest_secs DESC
       LIMIT 50`,
    );
    const open = await this.pool.query(
      `SELECT alert_key, severity, message, occurrences, last_seen_at
       FROM system.connectivity_alerts
       WHERE status = 'OPEN' AND alert_key LIKE 'DELIVERY:%'
       ORDER BY severity DESC, last_seen_at DESC`,
    );
    return {
      healthy: open.rows.length === 0,
      devices: rows.rows.map((r: Record<string, unknown>) => ({
        deviceId: r.device_id,
        deviceName: r.device_name,
        email: r.email,
        lastSignalAckAt: r.last_signal_ack_at,
        lastSignalAckType: r.last_signal_ack_type,
        emptyAcks24h: Number(r.empty_ack_count_24h ?? 0),
        pendingCount: Number(r.pending_count ?? 0),
        pendingOldestSecs: Number(r.pending_oldest_secs ?? 0),
        dropped24h: Number(r.dropped_count_24h ?? 0),
        delivered24h: Number(r.delivered_count_24h ?? 0),
        updatedAt: r.updated_at,
      })),
      openAlerts: open.rows,
      checkedAt: new Date().toISOString(),
    };
  }

  /** Deduped raise (mirrors ConnectivityWatchdogService.raiseAlert). */
  private async raiseAlert(a: RaiseInput) {
    const res = await this.pool.query(
      `INSERT INTO system.connectivity_alerts
         (alert_key, severity, scope, device_id, message, status, occurrences, last_seen_at)
       VALUES ($1, $2, 'DEVICE', $3, $4, 'OPEN', 1, now())
       ON CONFLICT (alert_key) DO UPDATE
         SET message = EXCLUDED.message,
             severity = EXCLUDED.severity,
             status = 'OPEN',
             occurrences = system.connectivity_alerts.occurrences + 1,
             last_seen_at = now()
       RETURNING notified_at`,
      [a.alertKey, a.severity, a.deviceId ?? null, a.message],
    );
    const notifiedAt = res.rows[0]?.notified_at ? new Date(res.rows[0].notified_at).getTime() : 0;
    if (!notifiedAt || Date.now() - notifiedAt > NOTIFY_COOLDOWN_MS) {
      await this.pool.query(
        `UPDATE system.connectivity_alerts SET notified_at = now() WHERE alert_key = $1`,
        [a.alertKey],
      );
      await this.notify('Predict-A-Trade signal delivery', a.message, a.severity);
      this.logger.warn(`ALERT [${a.severity}] ${a.alertKey}: ${a.message}`);
    }
  }

  private async resolveAlert(alertKey: string) {
    const res = await this.pool.query(
      `UPDATE system.connectivity_alerts
       SET status = 'RESOLVED', resolved_at = now()
       WHERE alert_key = $1 AND status = 'OPEN'
       RETURNING id`,
      [alertKey],
    );
    if (res.rows.length > 0) this.logger.log(`RESOLVED ${alertKey}`);
  }

  private notify(title: string, body: string, severity: string) {
    return new Promise<void>((resolve) => {
      const u = new URL(`/${NTFY_TOPIC}`, NTFY_BASE);
      const payload = JSON.stringify({
        topic: NTFY_TOPIC,
        title: `[${severity}] ${title}`,
        message: body,
        priority: severity === 'CRITICAL' ? 'high' : 'default',
        tags: severity === 'CRITICAL' ? ['rotating_light'] : ['signal_status'],
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