import { Injectable, Inject, NotFoundException, BadRequestException } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class LicensingService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /** List user's licenses with plan info */
  async listLicenses(userId: string) {
    const r = await this.pool.query(
      `SELECT l.*, p.name as plan_name,
              (SELECT count(*) FROM licensing.devices d WHERE d.bound_license_id = l.id AND d.deleted_at IS NULL) as device_count
       FROM licensing.licenses l
       LEFT JOIN control.plans p ON l.plan_id = p.id
       WHERE l.user_id = $1 ORDER BY l.created_at DESC`,
      [userId],
    );
    return r.rows;
  }

  /** List user's devices with activations (terminal data) */
  async listDevices(userId: string) {
    const r = await this.pool.query(
      `SELECT d.id, d.user_id, d.device_name, d.windows_version as os, d.agent_version,
              d.hostname, d.connection_status as status, d.security_state,
              d.first_seen_at as registered_at, d.last_seen_at,
              d.installation_id, d.fingerprint_hash, d.fingerprint_version,
              d.bound_license_id, d.revoked_at, d.revocation_reason,
              d.os_name, d.architecture, d.agent_uptime_seconds,
              d.service_status, d.health_status,
              l.license_key, l.status as license_status,
              l.max_devices, l.max_mt_accounts,
              (SELECT json_agg(json_build_object(
                  'id', da.id,
                  'client_type', da.client_type,
                  'terminal_build', da.terminal_build,
                  'ea_version', da.ea_version,
                  'terminal_version', da.terminal_version,
                  'terminal_connected', da.terminal_connected,
                  'broker_name', da.broker_name,
                  'broker_server', da.broker_server,
                  'mt_account_login', da.mt_account_login,
                  'installation_id', da.installation_id,
                  'fingerprint_hash', da.fingerprint_hash,
                  'activated_at', da.activated_at,
                  'balance', da.account_balance,
                  'equity', da.account_equity,
                  'profit', da.account_profit,
                  'currency', da.account_currency,
                  'leverage', da.leverage,
                  'margin', da.margin,
                  'free_margin', da.free_margin,
                  'margin_level', da.margin_level,
                  'account_type', da.account_type,
                  'open_positions', da.open_positions,
                  'buy_positions', da.buy_positions,
                  'sell_positions', da.sell_positions,
                  'pending_orders_count', da.pending_orders_count,
                  'total_lots', da.total_lots,
                  'floating_pnl', da.floating_pnl,
                  'last_account_update', da.last_account_update,
                  'xauusd', json_build_object(
                    'available', da.xauusd_available,
                    'bid', da.xauusd_bid,
                    'ask', da.xauusd_ask,
                    'spread', da.xauusd_spread,
                    'last_tick_time', da.xauusd_last_tick_time
                  )
               )) FROM licensing.device_activations da WHERE da.device_id = d.id) as activations
       FROM licensing.devices d
       LEFT JOIN licensing.licenses l ON d.bound_license_id = l.id
       WHERE d.user_id = $1 AND d.deleted_at IS NULL
       ORDER BY d.first_seen_at DESC`,
      [userId],
    );
    return r.rows;
  }

  /** Register a new device with hardware fingerprint */
  async registerDevice(userId: string, body: {
    deviceName?: string;
    os?: string;
    agentVersion?: string;
    hostname?: string;
    installationId?: string;
    fingerprintHash?: string;
    fingerprintComponents?: Record<string, string>;
    licenseKey?: string;
  }) {
    // Check if device with same fingerprint already exists for this user
    if (body.fingerprintHash) {
      const existing = await this.pool.query(
        `SELECT id, connection_status FROM licensing.devices
         WHERE user_id = $1 AND fingerprint_hash = $2 AND deleted_at IS NULL`,
        [userId, body.fingerprintHash],
      );
      if (existing.rows.length > 0) {
        // Update existing device
        const r = await this.pool.query(
          `UPDATE licensing.devices
           SET device_name = $3, windows_version = $4, agent_version = $5,
               hostname = $6, installation_id = $7,
               fingerprint_components = $8,
               connection_status = 'ONLINE', last_seen_at = now(), updated_at = now()
           WHERE id = $1 AND user_id = $2
           RETURNING *`,
          [existing.rows[0].id, userId, body.deviceName || 'Windows Client',
           body.os || 'Windows', body.agentVersion || '1.0.0',
           body.hostname || '', body.installationId || '',
           JSON.stringify(body.fingerprintComponents || {})],
        );
        return r.rows[0];
      }
    }

    // Bind license if provided
    let licenseId: string | null = null;
    if (body.licenseKey) {
      const lic = await this.pool.query(
        `SELECT id, max_devices, status FROM licensing.licenses
         WHERE license_key = $1 AND user_id = $2 AND status = 'ACTIVE' AND revoked_at IS NULL`,
        [body.licenseKey, userId],
      );
      if (lic.rows.length === 0) {
        throw new BadRequestException('Invalid or inactive license key');
      }
      // Check device count limit
      const count = await this.pool.query(
        `SELECT count(*) as cnt FROM licensing.devices
         WHERE bound_license_id = $1 AND deleted_at IS NULL AND revoked_at IS NULL`,
        [lic.rows[0].id],
      );
      if (parseInt(count.rows[0].cnt, 10) >= lic.rows[0].max_devices) {
        throw new BadRequestException('Maximum device limit reached for this license');
      }
      licenseId = lic.rows[0].id;
    }

    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO licensing.devices
        (id, user_id, bound_license_id, device_name, windows_version, agent_version,
         hostname, installation_id, fingerprint_version, fingerprint_hash, fingerprint_components,
         connection_status, security_state, first_seen_at, last_seen_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'hwfp-v1', $9, $10, 'ONLINE', 'SECURE', now(), now())
       RETURNING *`,
      [id, userId, licenseId, body.deviceName || 'Windows Client',
       body.os || 'Windows', body.agentVersion || '1.0.0',
       body.hostname || '', body.installationId || '',
       body.fingerprintHash || '', JSON.stringify(body.fingerprintComponents || {})],
    );
    return r.rows[0];
  }

  /** Register/update a terminal activation (MT4/MT5 client terminal) */
  async registerTerminal(userId: string, body: {
    deviceId?: string;
    licenseKey?: string;
    clientType: string; // MT4 or MT5
    terminalBuild?: string;
    eaVersion?: string;
    brokerName?: string;
    brokerServer?: string;
    mtAccountLogin: string;
    installationId?: string;
    fingerprintHash?: string;
  }) {
    // Find or create device
    let deviceId = body.deviceId;

    // P0-CP4 fix: a client-supplied deviceId must belong to the caller
    if (deviceId) {
      const owned = await this.pool.query(
        `SELECT id FROM licensing.devices WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
        [deviceId, userId],
      );
      if (owned.rows.length === 0) {
        throw new BadRequestException('Device not found for this user');
      }
    }

    if (!deviceId && body.licenseKey) {
      const lic = await this.pool.query(
        `SELECT id FROM licensing.licenses WHERE license_key = $1 AND user_id = $2 AND status = 'ACTIVE'`,
        [body.licenseKey, userId],
      );
      if (lic.rows.length === 0) {
        throw new BadRequestException('Invalid license key');
      }
      // Find device bound to this license
      const dev = await this.pool.query(
        `SELECT id FROM licensing.devices WHERE bound_license_id = $1 AND user_id = $2 AND deleted_at IS NULL ORDER BY last_seen_at DESC LIMIT 1`,
        [lic.rows[0].id, userId],
      );
      if (dev.rows.length > 0) {
        deviceId = dev.rows[0].id;
      }
    }

    if (!deviceId) {
      throw new BadRequestException('Device not found. Register the device first.');
    }

    // Check if activation with same MT account already exists
    const existing = await this.pool.query(
      `SELECT id FROM licensing.device_activations
       WHERE device_id = $1 AND mt_account_login = $2 AND client_type = $3`,
      [deviceId, body.mtAccountLogin, body.clientType],
    );

    if (existing.rows.length > 0) {
      // Update existing activation
      const r = await this.pool.query(
        `UPDATE licensing.device_activations
         SET terminal_build = $3, ea_version = $4, broker_name = $5, broker_server = $6,
             fingerprint_hash = $7, activated_at = now()
         WHERE id = $1 AND device_id = $2
         RETURNING *`,
        [existing.rows[0].id, deviceId, body.terminalBuild, body.eaVersion,
         body.brokerName, body.brokerServer, body.fingerprintHash || ''],
      );
      return r.rows[0];
    }

    // Check max_mt_accounts limit
    // P1 fix: also select l.id so the activation insert gets a real license_id
    const lic = await this.pool.query(
      `SELECT l.id, l.max_mt_accounts FROM licensing.devices d
       JOIN licensing.licenses l ON d.bound_license_id = l.id
       WHERE d.id = $1`, [deviceId],
    );
    if (lic.rows.length > 0) {
      const count = await this.pool.query(
        `SELECT count(*) as cnt FROM licensing.device_activations WHERE device_id = $1`,
        [deviceId],
      );
      if (parseInt(count.rows[0].cnt, 10) >= lic.rows[0].max_mt_accounts) {
        throw new BadRequestException('Maximum MT account limit reached for this license');
      }
    }

    // Create new activation
    const id = crypto.randomUUID();
    const licId = lic.rows.length > 0 ? lic.rows[0].id : null;
    const r = await this.pool.query(
      `INSERT INTO licensing.device_activations
        (id, license_id, device_id, client_type, terminal_build, ea_version,
         broker_name, broker_server, mt_account_login, installation_id,
         fingerprint_hash, activated_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
       RETURNING *`,
      [id, licId, deviceId, body.clientType, body.terminalBuild, body.eaVersion,
       body.brokerName, body.brokerServer, body.mtAccountLogin,
       body.installationId || '', body.fingerprintHash || ''],
    );

    // Update device last_seen
    await this.pool.query(
      `UPDATE licensing.devices SET last_seen_at = now(), connection_status = 'ONLINE', last_activation_at = now() WHERE id = $1`,
      [deviceId],
    );

    return r.rows[0];
  }

  /** List user's MT account activations */
  async listMtAccounts(userId: string) {
    const r = await this.pool.query(
      `SELECT da.*, d.device_name, d.hostname, d.connection_status,
              l.license_key, l.status as license_status
       FROM licensing.device_activations da
       JOIN licensing.devices d ON da.device_id = d.id
       LEFT JOIN licensing.licenses l ON da.license_id = l.id
       WHERE d.user_id = $1 AND d.deleted_at IS NULL
       ORDER BY da.activated_at DESC`,
      [userId],
    );
    return r.rows;
  }

  /** Add MT account (alias for registerTerminal) */
  async addMtAccount(userId: string, body: any) {
    return this.registerTerminal(userId, body);
  }

  /** Update device heartbeat (called by Windows Agent periodically) */
  // P0-CP4 fix: ownerUserId scoping — cross-tenant heartbeats rejected
  async heartbeat(deviceId: string, body: { connectionStatus?: string; fingerprintHash?: string }, ownerUserId?: string) {
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET last_seen_at = now(), connection_status = $2, updated_at = now()
       WHERE id = $1 AND deleted_at IS NULL
         AND ($3::uuid IS NULL OR user_id = $3::uuid)
       RETURNING id, connection_status, last_seen_at`,
      [deviceId, body.connectionStatus || 'ONLINE', ownerUserId ?? null],
    );
    if (r.rows.length === 0) {
      throw new NotFoundException('Device not found');
    }
    return r.rows[0];
  }

  /** Revoke a device (admin or owning user) */
  // P0-CP4 fix: ownerUserId scoping — users can only revoke their own devices
  async revokeDevice(deviceId: string, reason: string, ownerUserId?: string) {
    if (!ownerUserId) {
      throw new BadRequestException('owner context required for device revocation');
    }
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET revoked_at = now(), revocation_reason = $2, connection_status = 'REVOKED', updated_at = now()
       WHERE id = $1 AND deleted_at IS NULL AND user_id = $2::uuid`,
      [deviceId, reason, ownerUserId],
    );
    if (r.rows.length === 0) {
      throw new NotFoundException('Device not found');
    }
    return r.rows[0];
  }

  /** Manually sync terminal account data */
  async syncTerminalAccount(userId: string, body: {
    client_type: string;
    mt_account_login: string;
    balance?: number;
    equity?: number;
    profit?: number;
    currency?: string;
    open_positions?: number;
    buy_positions?: number;
    sell_positions?: number;
    total_lots?: number;
    floating_pnl?: number;
  }) {
    // Find the device activation for this user's terminal
    const result = await this.pool.query(
      `UPDATE licensing.device_activations da SET
        account_balance = $1, account_equity = $2, account_profit = $3,
        account_currency = $4, open_positions = $5, buy_positions = $6,
        sell_positions = $7, total_lots = $8, floating_pnl = $9,
        last_account_update = now()
       FROM licensing.devices d
       WHERE da.device_id = d.id AND d.user_id = $10
       AND da.mt_account_login = $11 AND da.client_type = $12
       RETURNING da.id, da.mt_account_login, da.account_balance, da.account_equity`,
      [body.balance || 0, body.equity || 0, body.profit || 0, body.currency || 'USD',
       body.open_positions || 0, body.buy_positions || 0, body.sell_positions || 0,
       body.total_lots || 0, body.floating_pnl || 0,
       userId, body.mt_account_login, body.client_type],
    );
    if (result.rows.length === 0) {
      return { success: false, message: 'Terminal not found for this user' };
    }
    return { success: true, updated: result.rows[0] };
  }

  /** Create a license (admin). Body fields validated by caller. */
  async createLicense(body: {
    user_id: string;
    plan_id: string;
    max_devices?: number;
    max_mt_accounts?: number;
    allowed_strategies?: string[];
    allowed_execution_modes?: string[];
    valid_days?: number;
  }) {
    if (!body.user_id || !body.plan_id) {
      throw new BadRequestException('user_id and plan_id are required');
    }
    const validDays = Number.isFinite(body.valid_days) && body.valid_days! > 0 ? body.valid_days! : 365;

    // Validate user + plan exist (clean errors instead of FK failures)
    const user = await this.pool.query(`SELECT id FROM iam.users WHERE id = $1`, [body.user_id]);
    if (user.rows.length === 0) {
      throw new BadRequestException('user_id does not exist');
    }
    const plan = await this.pool.query(`SELECT id FROM control.plans WHERE id = $1`, [body.plan_id]);
    if (plan.rows.length === 0) {
      throw new BadRequestException('plan_id does not exist');
    }

    const licenseKey = `PAT-${crypto.randomUUID().replace(/-/g, '')}`;
    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO licensing.licenses
        (id, user_id, plan_id, license_key, status, issued_at, valid_from, expires_at,
         max_devices, max_mt_accounts, allowed_strategies, allowed_execution_modes)
       VALUES ($1, $2, $3, $4, 'ACTIVE', now(), now(), now() + ($5 || ' days')::interval,
               $6, $7, $8, $9)
       RETURNING *`,
      [id, body.user_id, body.plan_id, licenseKey,
       validDays,
       body.max_devices ?? 1, body.max_mt_accounts ?? 1,
       JSON.stringify(body.allowed_strategies || []),
       JSON.stringify(body.allowed_execution_modes || [])],
    );
    return r.rows[0];
  }

  /** Suspend a license (admin). Soft state; reversible. */
  async suspendLicense(id: string, reason: string) {
    const r = await this.pool.query(
      `UPDATE licensing.licenses
       SET status = 'SUSPENDED', suspended_at = now(), updated_at = now()
       WHERE id = $1 AND revoked_at IS NULL
       RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) {
      const exists = await this.pool.query(`SELECT id FROM licensing.licenses WHERE id = $1`, [id]);
      if (exists.rows.length === 0) {
        throw new NotFoundException('License not found');
      }
      throw new BadRequestException('License is revoked and cannot be suspended');
    }
    await this.logLicenseEvent(id, 'SUSPENDED', reason);
    return r.rows[0];
  }

  /** Revoke a license (admin). Terminal. */
  async revokeLicense(id: string, reason: string) {
    const r = await this.pool.query(
      `UPDATE licensing.licenses
       SET status = 'REVOKED', revoked_at = now(), revocation_reason = $2, updated_at = now()
       WHERE id = $1
       RETURNING *`,
      [id, reason || 'Admin revoked'],
    );
    if (r.rows.length === 0) {
      throw new NotFoundException('License not found');
    }
    await this.logLicenseEvent(id, 'REVOKED', reason);
    return r.rows[0];
  }

  /** Renew a license (admin). Extends expires_at; reactivates expired/revoked. */
  async renewLicense(id: string, validDays?: number) {
    const current = await this.pool.query(
      `SELECT expires_at FROM licensing.licenses WHERE id = $1`,
      [id],
    );
    if (current.rows.length === 0) {
      throw new NotFoundException('License not found');
    }
    const days = Number.isFinite(validDays) && validDays! > 0 ? validDays! : 365;
    const base = current.rows[0].expires_at && new Date(current.rows[0].expires_at) > new Date()
      ? 'expires_at'
      : 'now()';
    const r = await this.pool.query(
      `UPDATE licensing.licenses
       SET expires_at = GREATEST(${base}, now()) + ($2 || ' days')::interval,
           status = 'ACTIVE', revoked_at = NULL, revocation_reason = NULL,
           suspended_at = NULL, updated_at = now()
       WHERE id = $1
       RETURNING *`,
      [id, days],
    );
    await this.logLicenseEvent(id, 'RENEWED', `extended ${days} days`);
    return r.rows[0];
  }

  /** Reset a license (admin). Soft-deactivate activations and clear device bindings; reversible. */
  async resetLicense(id: string) {
    const lic = await this.pool.query(`SELECT id FROM licensing.licenses WHERE id = $1`, [id]);
    if (lic.rows.length === 0) {
      throw new NotFoundException('License not found');
    }
    // Soft-deactivate activations bound to this license
    await this.pool.query(
      `UPDATE licensing.device_activations SET deactivated_at = now() WHERE license_id = $1 AND deactivated_at IS NULL`,
      [id],
    );
    // Clear device bindings (mark for re-registration; do NOT delete)
    await this.pool.query(
      `UPDATE licensing.devices
       SET bound_license_id = NULL, connection_status = 'OFFLINE', updated_at = now()
       WHERE bound_license_id = $1`,
      [id],
    );
    await this.logLicenseEvent(id, 'RESET', 'activations soft-deactivated, device bindings cleared');
    return { success: true, license_id: id };
  }

  /** Force-logout all devices bound to a license (admin). Revokes devices. */
  async forceLogoutLicense(id: string) {
    const lic = await this.pool.query(`SELECT id FROM licensing.licenses WHERE id = $1`, [id]);
    if (lic.rows.length === 0) {
      throw new NotFoundException('License not found');
    }
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET revoked_at = now(), revocation_reason = 'Force logout by admin',
           connection_status = 'REVOKED', updated_at = now()
       WHERE bound_license_id = $1 AND deleted_at IS NULL
       RETURNING id`,
      [id],
    );
    await this.logLicenseEvent(id, 'FORCE_LOGOUT', `revoked ${r.rowCount} device(s)`);
    return { success: true, license_id: id, devices_revoked: r.rowCount };
  }

  /** List activations for a license (admin). */
  async fetchLicenseActivations(id: string) {
    const lic = await this.pool.query(`SELECT id FROM licensing.licenses WHERE id = $1`, [id]);
    if (lic.rows.length === 0) {
      throw new NotFoundException('License not found');
    }
    const r = await this.pool.query(
      `SELECT da.*, d.device_name, d.hostname, d.connection_status, d.revoked_at as device_revoked_at
       FROM licensing.device_activations da
       LEFT JOIN licensing.devices d ON da.device_id = d.id
       WHERE da.license_id = $1
       ORDER BY da.activated_at DESC`,
      [id],
    );
    return { items: r.rows, total: r.rowCount };
  }

  /** Reset a device (admin). Clears fingerprint/session, restores SECURE/ONLINE. Reversible, no delete. */
  async resetDevice(id: string) {
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET fingerprint_hash = NULL, fingerprint_components = '{}', installation_id = NULL,
           security_state = 'SECURE', connection_status = 'ONLINE',
           last_reset_at = now(), updated_at = now()
       WHERE id = $1 AND deleted_at IS NULL
       RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) {
      throw new NotFoundException('Device not found');
    }
    return r.rows[0];
  }

  /** Flag a device for forced upgrade (admin). Windows Agent must upgrade next lease. */
  async forceUpgradeDevice(id: string) {
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET force_upgrade_pending = TRUE, updated_at = now()
       WHERE id = $1 AND deleted_at IS NULL
       RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) {
      throw new NotFoundException('Device not found');
    }
    return r.rows[0];
  }

  /** Disable signal delivery for a device (admin). */
  async disableDeviceSignal(id: string) {
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET signal_enabled = FALSE, updated_at = now()
       WHERE id = $1 AND deleted_at IS NULL
       RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) {
      throw new NotFoundException('Device not found');
    }
    return r.rows[0];
  }

  /** Append a license lifecycle event (audit). Non-fatal if insert fails. */
  private async logLicenseEvent(licenseId: string, eventType: string, reason?: string) {
    try {
      await this.pool.query(
        `INSERT INTO licensing.license_events (license_id, event_type, reason, created_at)
         VALUES ($1, $2, $3, now())`,
        [licenseId, eventType, reason || null],
      );
    } catch {
      // audit best-effort; never block the primary mutation
    }
  }

  /** Public license validation by license key (no JWT — used by Windows Agent) */
  async validateLicenseKey(licenseKey: string, mtAccount?: string, brokerName?: string, terminalBuild?: string, eaVersion?: string) {
    // 1. Look up the license
    const r = await this.pool.query(
      `SELECT l.id, l.status, l.license_key, l.max_devices, l.max_mt_accounts,
              l.allowed_strategies, l.allowed_execution_modes, l.user_id,
              p.code as plan_code, p.name as plan_name
       FROM licensing.licenses l
       LEFT JOIN control.plans p ON l.plan_id = p.id
       WHERE l.license_key = $1 AND l.revoked_at IS NULL
       LIMIT 1`,
      [licenseKey],
    );
    if (r.rows.length === 0) {
      return { valid: false, status: 'NOT_FOUND', error: 'License key not found' };
    }
    const row = r.rows[0];

    if (row.status !== 'ACTIVE') {
      return { valid: false, status: row.status, error: 'License is ' + row.status };
    }

    // 2. Enforce max_mt_accounts — count existing activations for this license
    if (mtAccount) {
      // Check if this MT account is already activated on THIS license
      const existingAct = await this.pool.query(
        `SELECT id FROM licensing.device_activations
         WHERE license_id = $1 AND mt_account_login = $2
         ORDER BY activated_at DESC LIMIT 1`,
        [row.id, mtAccount],
      );

      if (existingAct.rows.length === 0) {
        // New MT account — check if we have room
        const countResult = await this.pool.query(
          `SELECT COUNT(DISTINCT mt_account_login) as count
           FROM licensing.device_activations
           WHERE license_id = $1`,
          [row.id],
        );
        const currentCount = parseInt(countResult.rows[0]?.count || '0');
        if (currentCount >= row.max_mt_accounts) {
          return {
            valid: false,
            status: 'MAX_MT_ACCOUNTS_EXCEEDED',
            error: `License allows ${row.max_mt_accounts} MT account(s), ${currentCount} already activated`,
            max_mt_accounts: row.max_mt_accounts,
            current_count: currentCount,
          };
        }

        // Check if this MT account is already bound to a DIFFERENT license
        const otherLicense = await this.pool.query(
          `SELECT l.license_key FROM licensing.device_activations da
           JOIN licensing.licenses l ON da.license_id = l.id
           WHERE da.mt_account_login = $1 AND da.license_id != $2 AND l.revoked_at IS NULL
           LIMIT 1`,
          [mtAccount, row.id],
        );
        if (otherLicense.rows.length > 0) {
          return {
            valid: false,
            status: 'ACCOUNT_BOUND_TO_OTHER_LICENSE',
            error: `MT account ${mtAccount} is already activated on a different license`,
          };
        }
      }

      // M3 fix: validate is a READ-ONLY policy/eligibility check. It must NOT
      // insert device_activations or mutate device liveness — activation is
      // performed by the authenticated device-auth activation flow. Writing here
      // on every unauthenticated call is an abuse/DoS vector and created
      // spurious activation rows. MT-account binding is still validated above
      // (read-only) using the activations recorded by the auth flow.
    }

    return {
      valid: true,
      status: row.status,
      plan: row.plan_code,
      plan_name: row.plan_name,
      max_devices: row.max_devices,
      max_mt_accounts: row.max_mt_accounts,
      allowed_strategies: row.allowed_strategies || [],
      allowed_execution_modes: row.allowed_execution_modes || [],
      license_key: row.license_key,
    };
  }}
