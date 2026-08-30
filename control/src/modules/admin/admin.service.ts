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

  constructor(
    @Inject(DB_POOL) private pool: Pool,
  ) {}

  /** System overview with real statistics from the database. */
  async getOverview() {
    const [usersResult, subsResult, commResult, payoutsResult, plansResult] = await Promise.all([
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN status = 'ACTIVE' THEN 1 END) as active,
                       count(CASE WHEN status = 'SUSPENDED' THEN 1 END) as suspended,
                       count(CASE WHEN created_at >= date_trunc('month', now()) THEN 1 END) as new_this_month
                       FROM iam.users WHERE deleted_at IS NULL`),
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN s.status = 'ACTIVE' THEN 1 END) as active,
                       COALESCE(SUM(CASE WHEN s.status = 'ACTIVE' AND s.billing_interval = 'ANNUAL'
                                         THEN p.annual_price / 12 ELSE p.monthly_price END), 0) as mrr
                       FROM billing.subscriptions s LEFT JOIN control.plans p ON s.plan_id = p.id`),
      this.pool.query(`SELECT count(*) as total_entries,
                       COALESCE(SUM(CASE WHEN status = 'PENDING' THEN commission_amount ELSE 0 END), 0) as pending_amount,
                       COALESCE(SUM(CASE WHEN status = 'CLEARED' THEN commission_amount ELSE 0 END), 0) as cleared_amount,
                       COALESCE(SUM(CASE WHEN status = 'AVAILABLE' THEN commission_amount ELSE 0 END), 0) as available_amount,
                       COALESCE(SUM(CASE WHEN status = 'PAID' THEN commission_amount ELSE 0 END), 0) as paid_amount,
                       COALESCE(SUM(CASE WHEN status = 'PAID' THEN commission_amount ELSE 0 END), 0) as confirmed_amount
                       FROM referral.commission_ledger`),
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN status = 'REQUESTED' THEN 1 END) as pending,
                       COALESCE(SUM(CASE WHEN status = 'REQUESTED' THEN requested_amount ELSE 0 END), 0) as pending_amount
                       FROM referral.payouts`),
      this.pool.query(`SELECT count(*) as total, count(CASE WHEN status = 'ACTIVE' AND visible = TRUE AND legacy = FALSE THEN 1 END) as active FROM control.plans`),
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
        `SELECT s.id, s.user_id, u.email as user_email, s.plan_id, p.code as plan_code,
                p.name as plan_name,
                p.monthly_price, p.annual_price, s.billing_interval as billing_cycle,
                s.status,
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
              count(CASE WHEN status = 'CLEARED' THEN 1 END) as cleared_count,
              COALESCE(SUM(CASE WHEN status = 'CLEARED' THEN commission_amount ELSE 0 END), 0) as cleared_amount,
              count(CASE WHEN status = 'AVAILABLE' THEN 1 END) as available_count,
              COALESCE(SUM(CASE WHEN status = 'AVAILABLE' THEN commission_amount ELSE 0 END), 0) as available_amount,
              count(CASE WHEN status = 'PAID' THEN 1 END) as paid_count,
              COALESCE(SUM(CASE WHEN status = 'PAID' THEN commission_amount ELSE 0 END), 0) as paid_amount,
              count(CASE WHEN status = 'REVERSED' THEN 1 END) as reversed_count,
              COALESCE(SUM(CASE WHEN status = 'REVERSED' THEN commission_amount ELSE 0 END), 0) as reversed_amount
       FROM referral.commission_ledger`,
    );
    // Defensive defaults — aggregate queries always return one row in Postgres,
    // but guard against empty-result mocks and unexpected edge cases.
    return {
      total_entries: 0,
      total_amount: 0,
      pending_count: 0,
      pending_amount: 0,
      cleared_count: 0,
      cleared_amount: 0,
      available_count: 0,
      available_amount: 0,
      paid_count: 0,
      paid_amount: 0,
      reversed_count: 0,
      reversed_amount: 0,
      ...(r.rows[0] || {}),
      confirmed_count: r.rows[0]?.confirmed_count ?? r.rows[0]?.paid_count ?? 0,
      confirmed_amount: r.rows[0]?.confirmed_amount ?? r.rows[0]?.paid_amount ?? 0,
    };
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
    // Defensive defaults — aggregate queries always return one row in Postgres,
    // but guard against empty-result mocks and unexpected edge cases.
    return {
      total: 0,
      pending: 0,
      approved: 0,
      rejected: 0,
      pending_amount: 0,
      approved_amount: 0,
      ...(r.rows[0] || {}),
    };
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
                d.fingerprint_hash, d.fingerprint_version, d.fingerprint_components,
                d.last_activation_at,
                l.license_key, l.status as license_status,
                l.max_devices, l.max_mt_accounts,
                (SELECT json_agg(json_build_object(
                    'id', da.id,
                    'client_type', da.client_type,
                    'terminal_build', da.terminal_build,
                    'ea_version', da.ea_version,
                    'broker_name', da.broker_name,
                    'broker_server', da.broker_server,
                    'mt_account_login', da.mt_account_login,
                    'installation_id', da.installation_id,
                    'fingerprint_hash', da.fingerprint_hash,
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

    const planCheck = await this.pool.query(
      'SELECT id, monthly_price, annual_price, currency, billing_interval FROM control.plans WHERE id = $1',
      [planId],
    );
    if (planCheck.rows.length === 0) throw new Error('Plan not found');
    const plan = planCheck.rows[0];

    const id = licenseKey ? crypto.randomUUID() : crypto.randomUUID();
    const key = licenseKey || `PAT-${id.slice(0, 8).toUpperCase()}-${id.slice(9, 13).toUpperCase()}-${id.slice(14, 18).toUpperCase()}-${id.slice(19, 23).toUpperCase()}-${id.slice(24, 36).toUpperCase()}`;

    // Copy allowed_strategies from the plan to the license
    const planStrategies = await this.pool.query(
      'SELECT allowed_strategies FROM control.plans WHERE id = $1',
      [planId],
    );
    // pg returns jsonb as a JS object already — stringify it for the INSERT
    const rawStrategies = planStrategies.rows[0]?.allowed_strategies;
    const allowedStrategies = typeof rawStrategies === 'string'
      ? rawStrategies
      : JSON.stringify(rawStrategies || []);

    const r = await this.pool.query(
      `INSERT INTO licensing.licenses (id, user_id, plan_id, status, license_key, issued_at, valid_from, max_devices, max_mt_accounts, allowed_strategies, created_by, created_at, updated_at)
       VALUES ($1, $2, $3, 'ACTIVE', $4, now(), now(), 2, 2, $5, $6, now(), now()) RETURNING *`,
      [id, userId, planId, key, allowedStrategies, actorId],
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

    // ─── COMMISSION CREDIT: NOT triggered on license assignment (audit 2.4) ───
    // Referral commission must be credited ONLY from VALIDATED revenue — i.e. a
    // NOWPayments payment that has been settled AND amount-verified in
    // NowPaymentsService.handleIPN -> CommissionsService.creditReferralForSettledRevenue.
    // Crediting merely because a license was assigned would fabricate commission
    // on money that may never be paid. The canonical trigger point therefore lives
    // in the settlement webhook, not here.
    // TODO(CONTROL-PLANE): if a new validated-revenue path is added, it must call
    // creditReferralForSettledRevenue with the settled payment id — never credit
    // from license assignment. No commission is credited in this method.

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

    // 2. Control Plane (NestJS) - self
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

    // 4. Valkey/Redis - perform an actual TCP connection check.
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

    // 6. Windows Agent - derived from Go engine agents endpoint
    try {
      const goAgentsUrl = process.env.GO_ENGINE_AGENTS_URL || 'http://127.0.0.1:13081/api/v1/agents/status';
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 3000);
      const res = await fetch(goAgentsUrl, { signal: controller.signal });
      clearTimeout(timeout);
      const data = await res.json();
      const agentCount = data.agents_connected ?? 0;
      const agentsOnline = data.agents_online ?? false;
      services.push({
        service: 'Windows Agent',
        status: agentsOnline ? 'HEALTHY' : agentCount > 0 ? 'DEGRADED' : 'OFFLINE',
        latency_ms: 0,
        last_check: now,
        details: agentsOnline
          ? `Agents connected, ${agentCount} agent(s)`
          : `No agents, ${agentCount} agent(s) connected`,
      });
    } catch {
      services.push({
        service: 'Windows Agent',
        status: 'UNKNOWN',
        latency_ms: 0,
        last_check: now,
        details: 'Cannot reach Go engine to determine agent status',
      });
    }

    return { services };
  }

  /** Regime diagnostics — distribution of signal regimes plus the latest observed regime. */
  async getRegimeDiagnostics() {
    const [regimeStats, latest, sessionStats] = await Promise.all([
      this.pool.query(`
        SELECT regime, count(*) as count
        FROM trading.signals WHERE regime IS NOT NULL
        GROUP BY regime ORDER BY count DESC`),
      this.pool.query(`
        SELECT regime, session, created_at
        FROM trading.signals
        WHERE regime IS NOT NULL
        ORDER BY created_at DESC LIMIT 1`),
      this.pool.query(`
        SELECT session, count(*) as count
        FROM trading.signals WHERE session IS NOT NULL
        GROUP BY session ORDER BY count DESC`),
    ]);

    const total = regimeStats.rows.reduce((acc, r) => acc + parseInt(r.count, 10), 0);
    const current = latest.rows[0]?.regime ?? null;

    return {
      current_regime: current,
      current_session: latest.rows[0]?.session ?? null,
      last_signal_at: latest.rows[0]?.created_at ?? null,
      by_regime: regimeStats.rows.map((r) => ({
        regime: r.regime,
        count: parseInt(r.count, 10),
        share_pct: total > 0 ? Math.round((parseInt(r.count, 10) / total) * 1000) / 10 : 0,
      })),
      by_session: sessionStats.rows.map((r) => ({
        session: r.session,
        count: parseInt(r.count, 10),
      })),
      total_classified: total,
    };
  }

  /** Get the global persistent risk configuration (defaults if no row exists). */
  async getRiskConfig() {
    const r = await this.pool.query(
      `SELECT id, config_key, kill_switches, limits, session_blackout, news_blackout,
              blackout_reason, updated_by, updated_at
       FROM control.risk_config WHERE config_key = 'GLOBAL'`,
    );
    if (r.rows.length === 0) {
      return {
        config_key: 'GLOBAL',
        kill_switches: {},
        limits: {},
        session_blackout: false,
        news_blackout: false,
        blackout_reason: null,
        updated_by: null,
        updated_at: null,
      };
    }
    return r.rows[0];
  }

  /** Persist the global risk configuration (UPSERT merging JSONB fields). */
  async saveRiskConfig(
    payload: {
      kill_switches?: Record<string, boolean>;
      limits?: Record<string, number>;
      session_blackout?: boolean;
      news_blackout?: boolean;
      blackout_reason?: string;
    },
    actorId?: string,
  ) {
    const killSwitches = payload.kill_switches ?? {};
    const limits = payload.limits ?? {};
    const sessionBlackout = payload.session_blackout ?? false;
    const newsBlackout = payload.news_blackout ?? false;
    const blackoutReason = payload.blackout_reason ?? null;

    const r = await this.pool.query(
      `INSERT INTO control.risk_config (config_key, kill_switches, limits, session_blackout, news_blackout, blackout_reason, updated_by, updated_at)
       VALUES ('GLOBAL',
               $1::jsonb,
               $2::jsonb,
               $3, $4, $5, $6, now())
       ON CONFLICT (config_key) DO UPDATE SET
         kill_switches = COALESCE(control.risk_config.kill_switches, '{}'::jsonb) || EXCLUDED.kill_switches,
         limits        = COALESCE(control.risk_config.limits, '{}'::jsonb) || EXCLUDED.limits,
         session_blackout = EXCLUDED.session_blackout,
         news_blackout    = EXCLUDED.news_blackout,
         blackout_reason  = EXCLUDED.blackout_reason,
         updated_by = EXCLUDED.updated_by,
         updated_at = now()
       RETURNING id, config_key, kill_switches, limits, session_blackout, news_blackout, blackout_reason, updated_by, updated_at`,
      [
        JSON.stringify(killSwitches),
        JSON.stringify(limits),
        sessionBlackout,
        newsBlackout,
        blackoutReason,
        actorId || null,
      ],
    );

    try {
      await this.pool.query(
        `INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, entity_id, new_value, reason, timestamp)
         VALUES (gen_random_uuid(), gen_random_uuid(), 'USER', $1, 'RISK_CONFIG_SAVED', 'risk_config', 'GLOBAL', $2, $3, now())`,
        [actorId || null, JSON.stringify(payload), `Admin saved global risk config`],
      );
    } catch {
      this.logger.warn('Failed to write audit event for risk config save');
    }

    return r.rows[0];
  }

  /**
   * Returns true when the given relation is present in the connected database.
   * Used so the financial tabs degrade to an honest empty payload instead of
   * throwing when a migration has not been applied yet.
   */
  private async tableExists(schema: string, table: string): Promise<boolean> {
    try {
      const r = await this.pool.query(
        `SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2`,
        [schema, table],
      );
      return r.rows.length > 0;
    } catch (err) {
      this.logger.warn(
        `table existence check failed for ${schema}.${table}: ${err instanceof Error ? err.message : err}`,
      );
      return false;
    }
  }

  /** Recent billing payments (admin subscriptions → Payments tab). Never 500s. */
  async getSubscriptionPayments(limit = 50) {
    if (!(await this.tableExists('billing', 'payments'))) {
      return { items: [], note: 'No payments recorded' };
    }
    try {
      const r = await this.pool.query(
        `SELECT id, user_id, provider, amount, currency, payment_type, status, processed_at
         FROM billing.payments
         ORDER BY COALESCE(processed_at, created_at) DESC
         LIMIT $1`,
        [limit],
      );
      if (r.rows.length === 0) return { items: [], note: 'No payments recorded' };
      return { items: r.rows };
    } catch (err) {
      this.logger.warn(`billing.payments read failed: ${err instanceof Error ? err.message : err}`);
      return { items: [], note: 'No payments recorded' };
    }
  }

  /** Refunds (admin subscriptions → Refunds tab). Returns real rows or an honest empty note. */
  async getSubscriptionRefunds() {
    if (!(await this.tableExists('billing', 'refunds'))) {
      return { items: [], note: 'No refunds recorded' };
    }
    try {
      const r = await this.pool.query(
        `SELECT id, payment_id, amount, currency, reason, status, provider_refund_id, processed_at, created_at
         FROM billing.refunds
         ORDER BY COALESCE(processed_at, created_at) DESC
         LIMIT 50`,
      );
      if (r.rows.length === 0) return { items: [], note: 'No refunds recorded' };
      return { items: r.rows };
    } catch (err) {
      this.logger.warn(`billing.refunds read failed: ${err instanceof Error ? err.message : err}`);
      return { items: [], note: 'No refunds recorded' };
    }
  }

  /**
   * Chargebacks (admin subscriptions → Chargebacks tab). There is no dedicated
   * chargeback table; the canonical source is billing.payments rows flagged as
   * CHARGED_BACK / CHARGEBACK by the provider webhook path (migration 003).
   */
  async getSubscriptionChargebacks() {
    if (!(await this.tableExists('billing', 'payments'))) {
      return { items: [], note: 'No chargebacks recorded' };
    }
    try {
      const r = await this.pool.query(
        `SELECT id, user_id, provider, amount, currency, payment_type, status, processed_at
         FROM billing.payments
         WHERE status = 'CHARGED_BACK' OR payment_type = 'CHARGEBACK'
         ORDER BY COALESCE(processed_at, created_at) DESC
         LIMIT 50`,
      );
      if (r.rows.length === 0) return { items: [], note: 'No chargebacks recorded' };
      return { items: r.rows };
    } catch (err) {
      this.logger.warn(`chargeback read failed: ${err instanceof Error ? err.message : err}`);
      return { items: [], note: 'No chargebacks recorded' };
    }
  }

  /** Coupons (admin subscriptions → Coupons tab). Returns real rows or an honest empty note. */
  async getSubscriptionCoupons() {
    if (!(await this.tableExists('billing', 'coupons'))) {
      return { items: [], note: 'No coupons configured' };
    }
    try {
      const r = await this.pool.query(
        `SELECT id, code, description, discount_type, discount_value, currency,
                max_redemptions, redemption_count, active, valid_from, valid_until, created_at
         FROM billing.coupons
         ORDER BY created_at DESC
         LIMIT 50`,
      );
      if (r.rows.length === 0) return { items: [], note: 'No coupons configured' };
      return { items: r.rows };
    } catch (err) {
      this.logger.warn(`billing.coupons read failed: ${err instanceof Error ? err.message : err}`);
      return { items: [], note: 'No coupons configured' };
    }
  }

  /**
   * Payment provider status (admin subscriptions → Provider tab).
   * Never invents a provider name: the only accepted evidence of a configured
   * provider is a real distinct `provider` value already recorded in
   * billing.payments. Absent that, it reports honestly as unconfigured.
   */
  async getSubscriptionProvider() {
    const unconfigured = {
      provider: null as string | null,
      configured: false,
      providers: [] as Array<{ provider: string; payment_count: number; last_payment_at: string | null }>,
      note: 'No payment provider configured',
    };

    if (!(await this.tableExists('billing', 'payments'))) return unconfigured;

    try {
      const r = await this.pool.query(
        `SELECT provider,
                count(*)::int AS payment_count,
                max(COALESCE(processed_at, created_at)) AS last_payment_at
         FROM billing.payments
         WHERE provider IS NOT NULL AND provider <> ''
         GROUP BY provider
         ORDER BY count(*) DESC`,
      );
      if (r.rows.length === 0) return unconfigured;

      const providers = r.rows.map((row) => ({
        provider: row.provider as string,
        payment_count: row.payment_count as number,
        last_payment_at: row.last_payment_at ?? null,
      }));

      return {
        provider: providers[0].provider,
        configured: true,
        providers,
        note: 'Provider(s) derived from recorded payments.',
      };
    } catch (err) {
      this.logger.warn(`provider read failed: ${err instanceof Error ? err.message : err}`);
      return unconfigured;
    }
  }

  /** Trading report statistics - signal counts, strategy breakdown, hourly trends. */
  async getTradingReport() {
    const [
      directionStats,
      strategyStats,
      hourlyTrend,
      recentSignals,
      regimeStats,
      sessionStats,
      totalSignals,
      last24h,
      gateVetoStats,
    ] = await Promise.all([
      // Signal counts by direction
      this.pool.query(`
        SELECT direction, count(*) as count, 
               round(avg(raw_score)::numeric, 2) as avg_score,
               round(avg(calibrated_probability)::numeric, 4) as avg_prob
        FROM trading.signals GROUP BY direction ORDER BY count DESC`),

      // Signal counts by strategy + direction
      this.pool.query(`
        SELECT strategy_id, direction, count(*) as count,
               round(avg(raw_score)::numeric, 2) as avg_score
        FROM trading.signals GROUP BY strategy_id, direction 
        ORDER BY strategy_id, count DESC`),

      // Hourly trend (last 24 hours)
      this.pool.query(`
        SELECT date_trunc('hour', created_at) as hour, direction, count(*) as count
        FROM trading.signals 
        WHERE created_at > now() - interval '24 hours'
        GROUP BY hour, direction ORDER BY hour DESC`),

      // Recent actionable signals (BUY/SELL/CANDIDATE)
      this.pool.query(`
        SELECT s.signal_id, s.strategy_id, s.direction, s.raw_score, 
               s.calibrated_probability, s.entry_price, s.stop_loss, 
               s.tp1, s.tp2, s.tp3, s.regime, s.session, s.status,
               s.created_at
        FROM trading.signals s
        WHERE s.direction IN ('BUY','SELL','BUY_CANDIDATE','SELL_CANDIDATE','BLOCKED')
        ORDER BY s.created_at DESC LIMIT 50`),

      // Regime distribution
      this.pool.query(`
        SELECT regime, count(*) as count 
        FROM trading.signals WHERE regime IS NOT NULL 
        GROUP BY regime ORDER BY count DESC`),

      // Session distribution
      this.pool.query(`
        SELECT session, count(*) as count 
        FROM trading.signals WHERE session IS NOT NULL 
        GROUP BY session ORDER BY count DESC`),

      // Total signal count
      this.pool.query(`SELECT count(*) as total FROM trading.signals`),

      // Last 24h summary
      this.pool.query(`
        SELECT count(*) as total,
               count(CASE WHEN direction = 'BUY' THEN 1 END) as buy,
               count(CASE WHEN direction = 'SELL' THEN 1 END) as sell,
               count(CASE WHEN direction = 'BUY_CANDIDATE' THEN 1 END) as buy_candidate,
               count(CASE WHEN direction = 'SELL_CANDIDATE' THEN 1 END) as sell_candidate,
               count(CASE WHEN direction = 'NO-TRADE' THEN 1 END) as no_trade,
               count(CASE WHEN direction = 'BLOCKED' THEN 1 END) as blocked
        FROM trading.signals WHERE created_at > now() - interval '24 hours'`),

      // Gate veto reasons (from reason_codes in BLOCKED signals — grade=BLOCKED, not direction)
      this.pool.query(`
        SELECT jsonb_array_elements_text(reason_codes) as reason, count(*) as count
        FROM trading.signals 
        WHERE grade = 'BLOCKED' AND reason_codes IS NOT NULL AND reason_codes::text != '[]'
        GROUP BY reason ORDER BY count DESC LIMIT 10`),
    ]);

    return {
      summary: {
        total_signals: parseInt(totalSignals.rows[0]?.total ?? '0', 10),
        last_24h: last24h.rows[0] ?? { total: '0', buy: '0', sell: '0', buy_candidate: '0', sell_candidate: '0', no_trade: '0', blocked: '0' },
      },
      by_direction: directionStats.rows,
      by_strategy: strategyStats.rows,
      hourly_trend: hourlyTrend.rows,
      by_regime: regimeStats.rows,
      by_session: sessionStats.rows,
      recent_signals: recentSignals.rows,
      gate_vetoes: gateVetoStats.rows,
    };
  }

  /**
   * Signal accuracy ranking computed from trading.signals outcome data.
   * Win = resolved signal with realized_pnl > 0. Honest: returns empty strategies
   * when no resolved signals exist (never fabricates accuracy).
   */
  async getSignalAccuracy() {
    // Accuracy is computed from the real executed-trade outcome ledger
    // (trading.trade_results), which carries per-trade win/loss + realized P&L.
    // trading.signals is only the signal emission log and is rarely closed, so
    // it cannot be the source of truth for win-rate.
    const q = await this.pool.query(`
      SELECT
        strategy_id,
        count(*) AS total,
        count(*) FILTER (WHERE is_win) AS wins,
        count(*) FILTER (WHERE is_loss) AS losses,
        COALESCE(SUM(pnl), 0) AS total_pnl,
        COALESCE(AVG(pnl) FILTER (WHERE is_win OR is_loss), 0) AS avg_pnl
      FROM trading.trade_results
      GROUP BY strategy_id
    `);
    const strategies = q.rows.map((r) => {
      const total = Number(r.total);
      const wins = Number(r.wins);
      const losses = Number(r.losses);
      const resolved = wins + losses;
      const winRate = resolved > 0 ? (wins / resolved) * 100 : null;
      return {
        strategyId: r.strategy_id,
        total,
        resolved,
        wins,
        losses,
        winRate,
        avgPnl: Number(r.avg_pnl),
        totalPnl: Number(r.total_pnl),
      };
    }).sort((a, b) => (b.winRate ?? -1) - (a.winRate ?? -1));
    return { generatedAt: new Date().toISOString(), strategies };
  }

}
