import { Injectable, Inject, Logger, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { Pool } from 'pg';
import * as http from 'http';
import { DB_POOL } from '../../common/database.module';

/**
 * Connectivity watchdog (2026-09-04).
 *
 * Server-side guarantee for MT clients: if an edge-poll device goes quiet or
 * the realtime engine dies, the user AND the admin are notified — clients
 * must never silently lose trade signals.
 *
 * Checks every CHECK_INTERVAL_MS:
 *   1. Realtime engine health (GET realtime:13081/health) — CRITICAL if down
 *      or agents == 0.
 *   2. Per-device freshness — a device bound to an ACTIVE license that has
 *      not edge-polled for > DEVICE_STALE_MS is WARNING (user is missing
 *      signals).
 *   3. Conditions are deduped rows in system.connectivity_alerts
 *      (OPEN → RESOLVED when the condition clears) and pushed to ntfy
 *      (pat-connectivity topic) at most once per NOTIFY_COOLDOWN_MS per key.
 */

const CHECK_INTERVAL_MS = 60_000;
const DEVICE_STALE_MS = 3 * 60_000; // edge-poll cadence ~2s; 3 min = hard down
const NOTIFY_COOLDOWN_MS = 10 * 60_000; // re-notify at most every 10 min
const TICK_STALE_SECS = 180; // no tick receipt for 3 min = feed down
const SOURCE_TS_DRIFT_SECS = 300; // master source clock >5 min behind = warn

const NTFY_BASE = process.env.NTFY_URL || 'http://ntfy:80';
const NTFY_TOPIC = process.env.NTFY_TOPIC || 'pat-connectivity';
const REALTIME_HEALTH = process.env.REALTIME_HEALTH_URL || 'http://realtime:13081/health';

interface RaiseInput {
  alertKey: string;
  severity: 'INFO' | 'WARNING' | 'CRITICAL';
  scope: 'DEVICE' | 'ENGINE' | 'API';
  message: string;
  deviceId?: string | null;
}

@Injectable()
export class ConnectivityWatchdogService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(ConnectivityWatchdogService.name);
  private timer: NodeJS.Timeout | null = null;
  private running = false;

  constructor(@Inject(DB_POOL) private readonly pool: Pool) {}

  onModuleInit() {
    this.timer = setInterval(() => void this.runChecks(), CHECK_INTERVAL_MS);
    setTimeout(() => void this.runChecks(), 15_000);
    this.logger.log('Connectivity watchdog started (60s interval)');
  }

  onModuleDestroy() {
    if (this.timer) clearInterval(this.timer);
  }

  private async runChecks() {
    if (this.running) return; // never overlap cycles
    this.running = true;
    try {
      await this.checkRealtimeEngine();
      await this.checkDeviceFreshness();
    } catch (e) {
      this.logger.error(`Watchdog cycle failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      this.running = false;
    }
  }

  /** Realtime engine liveness — the source of all trade signals. */
  private async checkRealtimeEngine(): Promise<void> {
    const alive = await this.probeRealtime();
    if (!alive.ok) {
      await this.raiseAlert({
        alertKey: 'ENGINE:realtime-down',
        severity: 'CRITICAL',
        scope: 'ENGINE',
        message: `Realtime engine unreachable: ${alive.error}. MT clients receive no signals — check pat-realtime.`,
      });
      return;
    }
    if (alive.agents === 0) {
      await this.raiseAlert({
        alertKey: 'ENGINE:no-agents',
        severity: 'CRITICAL',
        scope: 'ENGINE',
        message: 'Realtime engine UP but zero agents connected (no market-data master) — signals will be stale.',
      });
    } else {
      await this.resolveAlert('ENGINE:realtime-down');
      await this.resolveAlert('ENGINE:no-agents');
    }
    // Feed health: ticks must keep landing. gateway_receipt_time is the
    // transport truth (source `time` lags when the master PC clock drifts —
    // seen live 2026-09-03: 15+ min source lag with 0.2s receipt lag).
    await this.checkTickFeed();
  }

  /** market.ticks must show a fresh gateway_receipt within TICK_STALE. */
  private async checkTickFeed(): Promise<void> {
    const res = await this.pool.query(
      `SELECT EXTRACT(EPOCH FROM (now() - max(gateway_receipt_time)))::int AS receipt_lag_secs,
              EXTRACT(EPOCH FROM (now() - max(time)))::int AS source_lag_secs
       FROM market.ticks
       WHERE gateway_receipt_time > now() - interval '1 hour'`,
    );
    const r = res.rows[0] ?? {};
    const receiptLag = Number(r.receipt_lag_secs ?? 9999);
    if (receiptLag > TICK_STALE_SECS) {
      await this.raiseAlert({
        alertKey: 'ENGINE:tick-feed-stale',
        severity: 'CRITICAL',
        scope: 'ENGINE',
        message: `No ticks received for ${Math.round(receiptLag / 60)} min — the market-data master stopped streaming. MT clients will receive no fresh signals.`,
      });
    } else {
      await this.resolveAlert('ENGINE:tick-feed-stale');
      // Informational: master clock drift (source timestamps lag real time).
      const sourceLag = Number(r.source_lag_secs ?? 0);
      if (sourceLag > SOURCE_TS_DRIFT_SECS) {
        await this.raiseAlert({
          alertKey: 'ENGINE:master-clock-drift',
          severity: 'WARNING',
          scope: 'ENGINE',
          message: `Market-data master clock is ${Math.round(sourceLag / 60)} min behind the server. Signals still flow (transport fresh), but candle timestamps may lag — sync the master terminal's clock.`,
        });
      } else {
        await this.resolveAlert('ENGINE:master-clock-drift');
      }
    }
  }

  /** Devices bound to active licenses that stopped polling. */
  private async checkDeviceFreshness(): Promise<void> {
    const res = await this.pool.query(
      `SELECT d.id, d.device_name, d.last_seen_at, u.email
       FROM licensing.devices d
       JOIN iam.users u ON u.id = d.user_id
       WHERE d.revoked_at IS NULL
         AND EXISTS (
           SELECT 1 FROM licensing.licenses l
           WHERE l.id = d.license_id AND l.status IN ('ACTIVE','TRIALING')
         )
         AND d.last_seen_at < now() - ($1::text || ' milliseconds')::interval
         AND d.last_seen_at > now() - interval '30 days'`,
      [String(DEVICE_STALE_MS)],
    );
    const staleKeys = new Set<string>();
    for (const d of res.rows) {
      const mins = Math.max(1, Math.round((Date.now() - new Date(d.last_seen_at).getTime()) / 60000));
      staleKeys.add(`DEVICE:${d.id}`);
      await this.raiseAlert({
        alertKey: `DEVICE:${d.id}`,
        severity: 'WARNING',
        scope: 'DEVICE',
        deviceId: d.id,
        message: `Device "${d.device_name}" (${d.email}) has not polled for ${mins} min — it is missing trade signals. Last seen ${new Date(d.last_seen_at).toISOString().slice(0, 16)}Z.`,
      });
    }
    // Auto-resolve device alerts whose device is fresh again.
    const open = await this.pool.query(
      `SELECT alert_key FROM system.connectivity_alerts
       WHERE status = 'OPEN' AND scope = 'DEVICE'`,
    );
    for (const row of open.rows) {
      if (!staleKeys.has(row.alert_key)) await this.resolveAlert(row.alert_key);
    }
  }

  /** Deduped raise: bumps occurrences; re-notifies at most once per cooldown. */
  private async raiseAlert(a: RaiseInput) {
    const res = await this.pool.query(
      `INSERT INTO system.connectivity_alerts
         (alert_key, severity, scope, device_id, message, status, occurrences, last_seen_at)
       VALUES ($1, $2, $3, $4, $5, 'OPEN', 1, now())
       ON CONFLICT (alert_key) DO UPDATE
         SET message = EXCLUDED.message,
             severity = EXCLUDED.severity,
             status = 'OPEN',
             occurrences = system.connectivity_alerts.occurrences + 1,
             last_seen_at = now()
       RETURNING id, notified_at, first_seen_at`,
      [a.alertKey, a.severity, a.scope, a.deviceId ?? null, a.message],
    );
    const row = res.rows[0];
    const notifiedAt = row?.notified_at ? new Date(row.notified_at).getTime() : 0;
    if (!notifiedAt || Date.now() - notifiedAt > NOTIFY_COOLDOWN_MS) {
      await this.pool.query(
        `UPDATE system.connectivity_alerts SET notified_at = now() WHERE alert_key = $1`,
        [a.alertKey],
      );
      await this.notify(
        a.scope === 'ENGINE' ? 'Predict-A-Trade engine down' : 'MT client disconnected',
        a.message,
        a.severity,
      );
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
    if (res.rows.length > 0) {
      this.logger.log(`RESOLVED ${alertKey}`);
    }
  }

  /** Admin dashboard payload: open alerts + per-device freshness. */
  async getConnectivitySnapshot() {
    const open = await this.pool.query(
      `SELECT alert_key, severity, scope, device_id, message, occurrences,
              first_seen_at, last_seen_at, notified_at
       FROM system.connectivity_alerts
       WHERE status = 'OPEN'
       ORDER BY severity DESC, last_seen_at DESC
       LIMIT 50`,
    );
    const recent = await this.pool.query(
      `SELECT alert_key, severity, scope, message, status, first_seen_at, resolved_at
       FROM system.connectivity_alerts
       WHERE last_seen_at > now() - interval '24 hours'
       ORDER BY last_seen_at DESC
       LIMIT 100`,
    );
    const devices = await this.pool.query(
      `SELECT d.id, d.device_name, d.last_seen_at, u.email,
              EXTRACT(EPOCH FROM (now() - d.last_seen_at))::int AS secs_since_poll
       FROM licensing.devices d
       JOIN iam.users u ON u.id = d.user_id
       WHERE d.revoked_at IS NULL
       ORDER BY d.last_seen_at DESC
       LIMIT 50`,
    );
    return {
      healthy: open.rows.length === 0,
      openAlerts: open.rows.map((r: Record<string, unknown>) => ({
        alertKey: r.alert_key,
        severity: r.severity,
        scope: r.scope,
        message: r.message,
        occurrences: r.occurrences,
        firstSeenAt: r.first_seen_at,
        lastSeenAt: r.last_seen_at,
      })),
      recentAlerts: recent.rows,
      devices: devices.rows.map((d: Record<string, unknown>) => ({
        deviceId: d.id,
        deviceName: d.device_name,
        lastSeenAt: d.last_seen_at,
        secondsSincePoll: d.secs_since_poll,
      })),
      checkedAt: new Date().toISOString(),
    };
  }

  /** GET realtime/health with a 5s timeout; returns agents count. */
  private probeRealtime(): Promise<{ ok: boolean; agents?: number; error?: string }> {
    return new Promise((resolve) => {
      const u = new URL(REALTIME_HEALTH);
      const req = http.request(
        { hostname: u.hostname, port: u.port || 80, path: u.pathname, method: 'GET', timeout: 5000 },
        (res) => {
          let body = '';
          res.on('data', (c) => { body += c; if (body.length > 16384) res.destroy(); });
          res.on('end', () => {
            if (res.statusCode !== 200) {
              resolve({ ok: false, error: `HTTP ${res.statusCode}` });
              return;
            }
            try {
              const j = JSON.parse(body) as { agents?: number; ws_clients?: number };
              resolve({ ok: true, agents: Number(j.agents ?? j.ws_clients ?? 0) });
            } catch {
              resolve({ ok: true, agents: 0 }); // up but unparsable — treat as unknown
            }
          });
        },
      );
      req.on('timeout', () => { req.destroy(); resolve({ ok: false, error: 'timeout (5s)' }); });
      req.on('error', (e) => resolve({ ok: false, error: e.message }));
      req.end();
    });
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
        { hostname: u.hostname, port: u.port || 80, path: u.pathname, method: 'POST', headers: { 'Content-Type': 'application/json' }, timeout: 5000 },
        (res) => { res.resume(); resolve(); },
      );
      req.on('error', () => resolve());
      req.on('timeout', () => { req.destroy(); resolve(); });
      req.end(payload);
    });
  }
}