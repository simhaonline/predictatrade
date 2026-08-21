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
                  'activated_at', da.activated_at,
                  'balance', da.account_balance,
                  'equity', da.account_equity,
                  'profit', da.account_profit,
                  'currency', da.account_currency,
                  'open_positions', da.open_positions,
                  'buy_positions', da.buy_positions,
                  'sell_positions', da.sell_positions,
                  'total_lots', da.total_lots,
                  'floating_pnl', da.floating_pnl,
                  'last_account_update', da.last_account_update
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
    const lic = await this.pool.query(
      `SELECT l.max_mt_accounts FROM licensing.devices d
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
  async heartbeat(deviceId: string, body: { connectionStatus?: string; fingerprintHash?: string }) {
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET last_seen_at = now(), connection_status = $2, updated_at = now()
       WHERE id = $1 AND deleted_at IS NULL
       RETURNING id, connection_status, last_seen_at`,
      [deviceId, body.connectionStatus || 'ONLINE'],
    );
    if (r.rows.length === 0) {
      throw new NotFoundException('Device not found');
    }
    return r.rows[0];
  }

  /** Revoke a device (admin or user) */
  async revokeDevice(deviceId: string, reason: string) {
    const r = await this.pool.query(
      `UPDATE licensing.devices
       SET revoked_at = now(), revocation_reason = $2, connection_status = 'REVOKED', updated_at = now()
       WHERE id = $1 AND deleted_at IS NULL
       RETURNING *`,
      [deviceId, reason],
    );
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
}
