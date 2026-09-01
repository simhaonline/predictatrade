import { Injectable, Logger, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

/**
 * EdgePollService — EA-direct signal delivery (Option B).
 *
 * Customer MT4/MT5 EAs poll POST /api/v1/devices/edge-poll using their device
 * HMAC signature (same Proof-of-Device scheme as the Windows agent). The
 * realtime engine enqueues EXECUTABLE signals into licensing.edge_signal_queue
 * for devices not connected via the agent WebSocket hub; this service hands
 * them to the EA and takes back execution ACKs — no local binaries required.
 */
@Injectable()
export class EdgePollService {
  private readonly logger = new Logger(EdgePollService.name);

  constructor(@Inject(DB_POOL) private readonly pool: Pool) {}

  /**
   * Poll: hand PENDING signals to the device and mark them IN_FLIGHT.
   * Stale IN_FLIGHT rows (EA crashed mid-batch) return to PENDING; signals
   * past their TTL are EXPIRED. Delivery is at-least-once; the EA dedupes by
   * signal ID.
   */
  async poll(deviceId: string, body: Record<string, any>) {
    const maxSignals = Math.min(Math.max(parseInt(body?.max_signals, 10) || 10, 1), 20);

    // Reclaim stale IN_FLIGHT (EA crashed mid-batch) back to PENDING.
    await this.pool.query(
      `UPDATE licensing.edge_signal_queue
          SET status = 'PENDING', last_error = COALESCE(last_error,'') || ' reclaimed;'
        WHERE device_id = $1::uuid AND status = 'IN_FLIGHT'
          AND in_flight_at < now() - interval '30 seconds'`,
      [deviceId],
    );

    // Expire signals past their TTL — never execute a stale read.
    // NOTE: the queue payload is json.Marshal(signal) of the Go struct, so
    // keys are PascalCase ("ExpiresAt"); accept the lowercase spelling too
    // in case the payload shape ever gains json tags.
    await this.pool.query(
      `UPDATE licensing.edge_signal_queue
          SET status = 'EXPIRED'
        WHERE device_id = $1::uuid AND status = 'PENDING'
          AND COALESCE(payload->>'ExpiresAt', payload->>'expires_at') IS NOT NULL
          AND COALESCE(payload->>'ExpiresAt', payload->>'expires_at')::timestamptz < now()`,
      [deviceId],
    );

    // Entitlement re-check at poll time (defense in depth — the engine already
    // filtered at enqueue): a license revoked or plan downgraded BETWEEN
    // enqueue and poll must not receive the signal. Fail-closed: if the
    // device's license/plan cannot be resolved, nothing is claimed.
    // Strategy key is PascalCase in the marshalled Signal ("StrategyID").
    await this.pool.query(
      `UPDATE licensing.edge_signal_queue q
          SET status = 'EXPIRED',
              last_error = COALESCE(q.last_error,'') || ' entitlement-revoked;'
        FROM licensing.devices d
        LEFT JOIN licensing.licenses l ON l.id = d.bound_license_id
        LEFT JOIN control.plans p ON p.id = l.plan_id
       WHERE q.device_id = d.id
         AND q.device_id = $1::uuid
         AND q.status = 'PENDING'
         AND COALESCE(q.payload->>'StrategyID', q.payload->>'strategy_id') IS NOT NULL
         AND (d.revoked_at IS NOT NULL
              OR l.id IS NULL
              OR l.status NOT IN ('ACTIVE', 'PENDING')
              OR (l.allowed_strategies IS NOT NULL
                  AND l.allowed_strategies::text NOT IN ('', 'null', '[]')
                  AND l.allowed_strategies::text NOT LIKE '%' || q.payload->>'StrategyID' || '%')
              OR (p.allowed_strategies IS NOT NULL
                  AND p.allowed_strategies::text NOT IN ('', 'null', '[]')
                  AND p.allowed_strategies::text NOT LIKE '%' || q.payload->>'StrategyID' || '%'))`,
      [deviceId],
    );

    // Claim the next batch atomically.
    const claimed = await this.pool.query(
      `UPDATE licensing.edge_signal_queue
          SET status = 'IN_FLIGHT', in_flight_at = now(), attempts = attempts + 1
        WHERE id IN (
          SELECT id FROM licensing.edge_signal_queue
           WHERE device_id = $1::uuid AND status = 'PENDING'
           ORDER BY created_at ASC
           LIMIT $2
           FOR UPDATE SKIP LOCKED
        )
        RETURNING id, signal_id, payload, created_at`,
      [deviceId, maxSignals],
    );

    // Record liveness + delivery counters.
    await this.pool.query(
      `INSERT INTO licensing.edge_device_state (device_id, last_poll_at, polls_total, signals_delivered, updated_at)
       VALUES ($1::uuid, now(), 1, $2, now())
       ON CONFLICT (device_id) DO UPDATE
         SET last_poll_at     = now(),
             polls_total      = licensing.edge_device_state.polls_total + 1,
             signals_delivered = licensing.edge_device_state.signals_delivered + $2,
             updated_at       = now()`,
      [deviceId, claimed.rows.length],
    ).catch(() => {});

    return {
      ok: true,
      server_time: new Date().toISOString(),
      pending: claimed.rows.map((r: any) => ({
        queue_id: r.id,
        signal_id: r.signal_id,
        created_at: r.created_at,
        signal: r.payload,
      })),
    };
  }

  /** Execution ACK: device reports fill/reject per queue item. */
  async ack(deviceId: string, body: Record<string, any>) {
    const queueId = body?.queue_id;
    if (!queueId) return { ok: false, error: 'queue_id required' };

    const result = await this.pool.query(
      `UPDATE licensing.edge_signal_queue
          SET status = 'ACKED', acked_at = now(), ack_result = $2::jsonb
        WHERE id = $1::uuid AND device_id = $3::uuid
        RETURNING signal_id`,
      [String(queueId), JSON.stringify(body?.result || body || {}), deviceId],
    );

    if (result.rows.length === 0) {
      return { ok: false, error: 'NOT_FOUND_OR_WRONG_DEVICE' };
    }

    await this.pool.query(
      `UPDATE licensing.edge_device_state
          SET last_ack_at = now(), signals_acked = signals_acked + 1, updated_at = now()
        WHERE device_id = $1::uuid`,
      [deviceId],
    ).catch(() => {});

    this.logger.log(`[EDGE-ACK] device=${deviceId} queue=${queueId} result=${body?.status || 'unknown'}`);
    return { ok: true };
  }

  /** Lightweight heartbeat (also carries EA-side terminal/account metadata). */
  async heartbeat(deviceId: string, body: Record<string, any>, ip?: string) {
    await this.pool.query(
      `INSERT INTO licensing.edge_device_state (device_id, last_heartbeat_at, updated_at)
       VALUES ($1::uuid, now(), now())
       ON CONFLICT (device_id) DO UPDATE
         SET last_heartbeat_at = now(), updated_at = now()`,
      [deviceId],
    );
    // Mirror into devices.last_seen_at so admin dashboards stay truthful.
    await this.pool.query(
      `UPDATE licensing.devices SET last_seen_at = now(), last_ip = COALESCE($2::inet, last_ip)
        WHERE id = $1::uuid`,
      [deviceId, ip || null],
    ).catch(() => {});

    return { ok: true, server_time: new Date().toISOString() };
  }

  /**
   * Enqueue an EXECUTABLE signal for every EA-direct device. Called by the
   * realtime engine when an executable signal is published. Entitlement
   * enforcement happens at delivery time (the poll handler filters by the
   * device's license plan) AND here cheaply (license ACTIVE only) — the
   * strategy-level entitlement check lives in the poll handler where the
   * device's plan is known.
   */
  // v1.19.0 (Option B): enqueueForAllDevices was REMOVED — the realtime engine
  // enqueues EXECUTABLE signals into licensing.edge_signal_queue directly
  // (in-process DB), so the HTTP enqueue proxy is dead code. Entitlement at
  // enqueue time lives in the engine's enqueueSignalForDevices SQL filter.
}