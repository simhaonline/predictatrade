import { Injectable, Inject, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

export interface HealthServiceStatus {
  service: string;
  status: 'HEALTHY' | 'DEGRADED' | 'OFFLINE' | 'UNKNOWN';
  latency_ms?: number;
  last_check: string;
  version?: string;
  details?: string;
}

@Injectable()
export class AdminService {
  private readonly logger = new Logger(AdminService.name);

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /** System overview with real statistics from the database. */
  async getOverview() {
    const [usersResult, subsResult, commResult, payoutsResult, plansResult] = await Promise.all([
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN status = 'ACTIVE' THEN 1 END) as active,
                       count(CASE WHEN status = 'SUSPENDED' THEN 1 END) as suspended,
                       count(CASE WHEN created_at >= date_trunc('month', now()) THEN 1 END) as new_this_month
                       FROM iam.users WHERE deleted_at IS NULL`),
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN s.status = 'ACTIVE' THEN 1 END) as active,
                       COALESCE(SUM(CASE WHEN s.status = 'ACTIVE' THEN p.monthly_price ELSE 0 END), 0) as mrr
                       FROM billing.subscriptions s LEFT JOIN control.plans p ON s.plan_id = p.id`),
      this.pool.query(`SELECT count(*) as total_entries,
                       COALESCE(SUM(CASE WHEN status = 'PENDING' THEN commission_amount ELSE 0 END), 0) as pending_amount,
                       COALESCE(SUM(CASE WHEN status = 'CONFIRMED' THEN commission_amount ELSE 0 END), 0) as confirmed_amount
                       FROM referral.commission_ledger`),
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN status = 'REQUESTED' THEN 1 END) as pending,
                       COALESCE(SUM(CASE WHEN status = 'REQUESTED' THEN requested_amount ELSE 0 END), 0) as pending_amount
                       FROM referral.payouts`),
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN status = 'ACTIVE' THEN 1 END) as active FROM control.plans`),
    ]);

    return {
      users: usersResult.rows[0],
      subscriptions: subsResult.rows[0],
      commissions: commResult.rows[0],
      payouts: payoutsResult.rows[0],
      plans: plansResult.rows[0],
    };
  }

  /** List all users with pagination (admin only). */
  async listUsers(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT u.id, u.email, u.full_name, u.status, u.created_at, u.last_login_at,
                COALESCE(r.name, 'USER') as role
         FROM iam.users u
         LEFT JOIN iam.memberships m ON m.user_id = u.id
         LEFT JOIN iam.roles r ON m.role_id = r.id
         WHERE u.deleted_at IS NULL
         ORDER BY u.created_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM iam.users WHERE deleted_at IS NULL'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  /** Update user status (admin only). Audits the action. */
  async updateUserStatus(userId: string, status: string, actorId?: string) {
    const validStatuses = ['ACTIVE', 'SUSPENDED', 'LOCKED', 'DELETED'];
    if (!validStatuses.includes(status)) {
      throw new Error(`Invalid status: ${status}`);
    }
    const r = await this.pool.query(
      'UPDATE iam.users SET status = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL RETURNING id, email, full_name, status',
      [status, userId],
    );
    if (r.rows.length === 0) throw new Error('User not found');
    // Revoke sessions if suspended/locked/deleted
    if (status !== 'ACTIVE') {
      await this.pool.query(
        'UPDATE iam.sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL',
        [userId],
      );
    }
    // Audit the user status change
    const auditAction = status === 'SUSPENDED' ? 'USER_SUSPENDED' : status === 'ACTIVE' ? 'USER_REACTIVATED' : 'USER_UPDATED';
    try {
      await this.pool.query(
        `INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, entity_id, new_value, reason, timestamp)
         VALUES (gen_random_uuid(), gen_random_uuid(), 'USER', $1, $2, 'user', $3, $4, $5, now())`,
        [actorId || null, auditAction, userId, JSON.stringify({ status }), `Admin changed user status to ${status}`],
      );
    } catch {
      this.logger.warn('Failed to write audit event for user status change');
    }
    return r.rows[0];
  }

  /** List all subscriptions (admin only). Uses LEFT JOIN for plan (plan is NOT NULL but LEFT JOIN for safety). */
  async listAllSubscriptions(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT s.id, s.user_id, u.email as user_email, s.plan_id, p.name as plan_name,
                s.status, p.billing_interval as billing_cycle,
                s.billing_period_start as current_period_start,
                s.billing_period_end as current_period_end,
                s.auto_renew, s.created_at, s.updated_at,
                l.license_key, l.id as license_id
         FROM billing.subscriptions s
         JOIN iam.users u ON s.user_id = u.id
         LEFT JOIN control.plans p ON s.plan_id = p.id
         LEFT JOIN licensing.licenses l ON l.subscription_id = s.id
         ORDER BY s.created_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM billing.subscriptions'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  /** List all commissions (admin only). */
  async listAllCommissions(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT c.id, c.recipient_user_id, ru.email as recipient_email,
                c.source_user_id, su.email as source_email,
                c.commission_amount, c.level as commission_level, c.status, c.created_at
         FROM referral.commission_ledger c
         LEFT JOIN iam.users ru ON c.recipient_user_id = ru.id
         LEFT JOIN iam.users su ON c.source_user_id = su.id
         ORDER BY c.created_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM referral.commission_ledger'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  /** Commission summary (admin only). */
  async commissionSummary() {
    const r = await this.pool.query(
      `SELECT count(*) as total_entries,
              COALESCE(SUM(commission_amount), 0) as total_amount,
              count(CASE WHEN status = 'PENDING' THEN 1 END) as pending_count,
              COALESCE(SUM(CASE WHEN status = 'PENDING' THEN commission_amount ELSE 0 END), 0) as pending_amount,
              count(CASE WHEN status = 'CONFIRMED' THEN 1 END) as confirmed_count,
              COALESCE(SUM(CASE WHEN status = 'CONFIRMED' THEN commission_amount ELSE 0 END), 0) as confirmed_amount,
              count(CASE WHEN status = 'REVERSED' THEN 1 END) as reversed_count,
              COALESCE(SUM(CASE WHEN status = 'REVERSED' THEN commission_amount ELSE 0 END), 0) as reversed_amount
       FROM referral.commission_ledger`,
    );
    return r.rows[0];
  }

  /** Payout statistics (admin only). */
  async payoutStats() {
    const r = await this.pool.query(
      `SELECT count(*) as total,
              count(CASE WHEN status = 'REQUESTED' THEN 1 END) as pending,
              count(CASE WHEN status = 'APPROVED' THEN 1 END) as approved,
              count(CASE WHEN status = 'REJECTED' THEN 1 END) as rejected,
              COALESCE(SUM(CASE WHEN status = 'REQUESTED' THEN requested_amount ELSE 0 END), 0) as pending_amount,
              COALESCE(SUM(CASE WHEN status = 'APPROVED' THEN requested_amount ELSE 0 END), 0) as approved_amount
       FROM referral.payouts`,
    );
    return r.rows[0];
  }

  /** List all payouts (admin only). */
  async listAllPayouts(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT p.id, p.user_id, u.email as user_email,
                p.requested_amount as amount, p.status, p.requested_at as created_at, p.approved_at,
                p.currency, p.net_amount
         FROM referral.payouts p
         JOIN iam.users u ON p.user_id = u.id
         ORDER BY p.requested_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM referral.payouts'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  /** List all licenses (admin only). Uses LEFT JOINs for optional relations. */
  async listAllLicenses(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT l.id, l.user_id, u.email as user_email, l.license_key as key, l.status,
                l.issued_at as activated_at, l.expires_at, l.valid_from,
                l.max_devices, l.max_mt_accounts, l.allowed_strategies,
                p.name as plan_name, p.id as plan_id,
                s.status as subscription_status
         FROM licensing.licenses l
         JOIN iam.users u ON l.user_id = u.id
         LEFT JOIN control.plans p ON l.plan_id = p.id
         LEFT JOIN billing.subscriptions s ON l.subscription_id = s.id
         ORDER BY l.issued_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM licensing.licenses'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  /** List all devices (admin only). Uses LEFT JOIN for license (device may not be licensed). */
  async listAllDevices(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT d.id, d.user_id, u.email as user_email,
                d.device_name, d.windows_version as os, d.agent_version,
                d.hostname, d.connection_status as status,
                d.first_seen_at as registered_at, d.last_seen_at,
                d.bound_license_id, d.revoked_at, d.revocation_reason,
                d.security_state, d.installation_id,
                l.license_key, l.status as license_status,
                (SELECT json_agg(json_build_object(
                    'client_type', da.client_type,
                    'broker_name', da.broker_name,
                    'mt_account_login', da.mt_account_login,
                    'activated_at', da.activated_at
                 )) FROM licensing.device_activations da WHERE da.device_id = d.id) as activations
         FROM licensing.devices d
         JOIN iam.users u ON d.user_id = u.id
         LEFT JOIN licensing.licenses l ON d.bound_license_id = l.id
         WHERE d.deleted_at IS NULL
         ORDER BY d.first_seen_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM licensing.devices WHERE deleted_at IS NULL'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  /** List all device activations (admin only). */
  async listAllActivations(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT da.id, da.license_id, l.license_key, da.device_id,
                u.email as user_email, d.device_name,
                da.client_type, da.terminal_build, da.ea_version,
                da.broker_name, da.broker_server, da.mt_account_login,
                da.installation_id, da.activated_at, da.created_at,
                d.connection_status, d.last_seen_at,
                d.hostname
         FROM licensing.device_activations da
         JOIN licensing.licenses l ON da.license_id = l.id
         JOIN licensing.devices d ON da.device_id = d.id
         JOIN iam.users u ON l.user_id = u.id
         ORDER BY da.activated_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM licensing.device_activations'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  /** Assign/create a license for a user (admin only). */
  async assignLicense(userId: string, planId: string, actorId: string, licenseKey?: string) {
    const userCheck = await this.pool.query('SELECT id FROM iam.users WHERE id = $1 AND deleted_at IS NULL', [userId]);
    if (userCheck.rows.length === 0) throw new Error('User not found');

    const planCheck = await this.pool.query('SELECT id FROM control.plans WHERE id = $1', [planId]);
    if (planCheck.rows.length === 0) throw new Error('Plan not found');

    const id = licenseKey ? crypto.randomUUID() : crypto.randomUUID();
    const key = licenseKey || `PAT-${id.slice(0, 8).toUpperCase()}-${id.slice(9, 13).toUpperCase()}-${id.slice(14, 18).toUpperCase()}-${id.slice(19, 23).toUpperCase()}-${id.slice(24, 36).toUpperCase()}`;

    const r = await this.pool.query(
      `INSERT INTO licensing.licenses (id, user_id, plan_id, status, license_key, issued_at, valid_from, max_devices, max_mt_accounts, created_by, created_at, updated_at)
       VALUES ($1, $2, $3, 'ACTIVE', $4, now(), now(), 2, 2, $5, now(), now()) RETURNING *`,
      [id, userId, planId, key, actorId],
    );

    // Audit the license assignment
    await this.pool.query(
      `INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, entity_id, new_value, reason, timestamp)
       VALUES (gen_random_uuid(), gen_random_uuid(), 'USER', $1, 'LICENSE_ASSIGNED', 'license', $2, $3, 'Admin assigned license to user', now())`,
      [actorId, id, JSON.stringify({ user_id: userId, plan_id: planId, license_key: key })],
    );

    // Also record in license_events
    await this.pool.query(
      `INSERT INTO licensing.license_events (license_id, event_type, reason, metadata, created_at)
       VALUES ($1, 'ASSIGNED', 'Admin assigned license', $2, now())`,
      [id, JSON.stringify({ actor_id: actorId, user_id: userId })],
    );

    return r.rows[0];
  }

  /** Get user detail with subscription, license, device info (admin only). */
  async getUserDetail(userId: string) {
    const [user, subscription, licenses, devices, activations] = await Promise.all([
      this.pool.query(
        `SELECT u.id, u.email, u.full_name, u.status, u.created_at, u.last_login_at,
                COALESCE(r.name, 'USER') as role
         FROM iam.users u
         LEFT JOIN iam.memberships m ON m.user_id = u.id
         LEFT JOIN iam.roles r ON m.role_id = r.id
         WHERE u.id = $1 AND u.deleted_at IS NULL`,
        [userId],
      ),
      this.pool.query(
        `SELECT s.id, s.status, p.name as plan_name, s.billing_period_start, s.billing_period_end, s.auto_renew
         FROM billing.subscriptions s
         LEFT JOIN control.plans p ON s.plan_id = p.id
         WHERE s.user_id = $1 ORDER BY s.created_at DESC`,
        [userId],
      ),
      this.pool.query(
        `SELECT l.id, l.license_key, l.status, l.issued_at, l.expires_at, l.max_devices, l.max_mt_accounts,
                p.name as plan_name
         FROM licensing.licenses l
         LEFT JOIN control.plans p ON l.plan_id = p.id
         WHERE l.user_id = $1 ORDER BY l.issued_at DESC`,
        [userId],
      ),
      this.pool.query(
        `SELECT d.id, d.device_name, d.connection_status, d.hostname, d.last_seen_at, d.security_state,
                l.license_key
         FROM licensing.devices d
         LEFT JOIN licensing.licenses l ON d.bound_license_id = l.id
         WHERE d.user_id = $1 AND d.deleted_at IS NULL
         ORDER BY d.first_seen_at DESC`,
        [userId],
      ),
      this.pool.query(
        `SELECT da.id, da.client_type, da.broker_name, da.mt_account_login, da.activated_at
         FROM licensing.device_activations da
         JOIN licensing.licenses l ON da.license_id = l.id
         WHERE l.user_id = $1 ORDER BY da.activated_at DESC`,
        [userId],
      ),
    ]);

    if (user.rows.length === 0) throw new Error('User not found');

    return {
      ...user.rows[0],
      subscription: subscription.rows[0] || null,
      licenses: licenses.rows,
      devices: devices.rows,
      activations: activations.rows,
    };
  }

  /** System health check (admin only). Returns structured service health. */
  async systemHealth(): Promise<{ services: HealthServiceStatus[] }> {
    const services: HealthServiceStatus[] = [];
    const now = new Date().toISOString();

    // 1. Database (PostgreSQL/TimescaleDB)
    const dbStart = Date.now();
    try {
      await this.pool.query('SELECT 1');
      services.push({
        service: 'PostgreSQL/TimescaleDB',
        status: 'HEALTHY',
        latency_ms: Date.now() - dbStart,
        last_check: now,
        details: 'Connection successful',
      });
    } catch (err) {
      services.push({
        service: 'PostgreSQL/TimescaleDB',
        status: 'OFFLINE',
        latency_ms: Date.now() - dbStart,
        last_check: now,
        details: err instanceof Error ? err.message : 'Connection failed',
      });
    }

    // 2. Control Plane (NestJS) — self
    services.push({
      service: 'Control Plane (NestJS)',
      status: 'HEALTHY',
      latency_ms: 0,
      last_check: now,
      version: '1.0.0',
      details: 'Self-reporting',
    });

    // 3. Go Real-Time Engine
    const goStart = Date.now();
    try {
      const goHealthUrl = process.env.GO_ENGINE_HEALTH_URL || 'http://127.0.0.1:13081/health';
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 3000);
      const res = await fetch(goHealthUrl, { signal: controller.signal });
      clearTimeout(timeout);
      const data = await res.json();
      services.push({
        service: 'Go Real-Time Engine',
        status: data.status === 'ok' ? 'HEALTHY' : 'DEGRADED',
        latency_ms: Date.now() - goStart,
        last_check: now,
        version: data.version || 'unknown',
        details: `agents: ${data.agents ?? 0}, ws_clients: ${data.ws_clients ?? 0}`,
      });
    } catch (err) {
      services.push({
        service: 'Go Real-Time Engine',
        status: 'OFFLINE',
        latency_ms: Date.now() - goStart,
        last_check: now,
        details: err instanceof Error ? err.message : 'Connection refused',
      });
    }

    // 4. Valkey/Redis — perform an actual TCP connection check.
    // Previously this was hardcoded UNKNOWN, which showed a false "Unknown"
    // status on the System Health page even when Valkey was running.
    const valkeyAddr = process.env.VALKEY_ADDR || '127.0.0.1:6379';
    const valkeyStart = Date.now();
    try {
      const { Socket } = await import('net');
      const [host, portStr] = valkeyAddr.split(':');
      const port = parseInt(portStr || '6379', 10);
      await new Promise<void>((resolve, reject) => {
        const sock = new Socket();
        const timer = setTimeout(() => { sock.destroy(); reject(new Error('timeout')); }, 2000);
        sock.connect(port, host, () => { clearTimeout(timer); sock.destroy(); resolve(); });
        sock.on('error', (err) => { clearTimeout(timer); reject(err); });
      });
      services.push({
        service: 'Valkey/Redis',
        status: 'HEALTHY',
        latency_ms: Date.now() - valkeyStart,
        last_check: now,
        details: `TCP connection to ${valkeyAddr} successful`,
      });
    } catch (err) {
      services.push({
        service: 'Valkey/Redis',
        status: 'OFFLINE',
        latency_ms: Date.now() - valkeyStart,
        last_check: now,
        details: `Cannot connect to ${valkeyAddr}: ${err instanceof Error ? err.message : 'unknown'}`,
      });
    }

    // 5. Frontend (Next.js)
    const feStart = Date.now();
    try {
      const feUrl = process.env.FRONTEND_HEALTH_URL || 'http://127.0.0.1:13082';
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 3000);
      const res = await fetch(feUrl, { signal: controller.signal });
      clearTimeout(timeout);
      services.push({
        service: 'Frontend (Next.js)',
        status: res.ok ? 'HEALTHY' : 'DEGRADED',
        latency_ms: Date.now() - feStart,
        last_check: now,
        details: `HTTP ${res.status}`,
      });
    } catch (err) {
      services.push({
        service: 'Frontend (Next.js)',
        status: 'OFFLINE',
        latency_ms: Date.now() - feStart,
        last_check: now,
        details: err instanceof Error ? err.message : 'Connection refused',
      });
    }

    // 6. Windows Agent / Master Node — derived from Go engine agents endpoint
    try {
      const goAgentsUrl = process.env.GO_ENGINE_AGENTS_URL || 'http://127.0.0.1:13081/api/v1/agents/status';
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 3000);
      const res = await fetch(goAgentsUrl, { signal: controller.signal });
      clearTimeout(timeout);
      const data = await res.json();
      const agentCount = data.agents_connected ?? 0;
      const masterConnected = data.master_node_connected ?? false;
      services.push({
        service: 'Windows Agent / Master Node',
        status: masterConnected ? 'HEALTHY' : agentCount > 0 ? 'DEGRADED' : 'OFFLINE',
        latency_ms: 0,
        last_check: now,
        details: masterConnected
          ? `Master node connected, ${agentCount} agent(s)`
          : `No master node, ${agentCount} agent(s) connected`,
      });
    } catch {
      services.push({
        service: 'Windows Agent / Master Node',
        status: 'UNKNOWN',
        latency_ms: 0,
        last_check: now,
        details: 'Cannot reach Go engine to determine agent status',
      });
    }

    return { services };
  }
}
