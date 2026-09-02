import { Injectable, Inject, Logger, BadRequestException, UnauthorizedException, ConflictException, NotFoundException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { JwtService } from '@nestjs/jwt';
import { Pool } from 'pg';
import * as crypto from 'crypto';
import { DB_POOL } from '../../common/database.module';

// H-fix: single source of truth for the JWT-derived secret, matching
// common/jwt.module.ts. Previously this service read process.env.JWT_SECRET
// directly, which could diverge from the ConfigService-backed signer.
const DEV_JWT_SECRET = 'pat_local_dev_secret_change_in_production';

const FINGERPRINT_WEIGHTS: Record<string, number> = {
  machine_guid: 25,
  system_uuid: 25,
  motherboard: 20,
  installation_id: 20,
  disk: 10,
};

const MATCH_THRESHOLD = 75;

@Injectable()
export class DeviceAuthService {
  private readonly logger = new Logger(DeviceAuthService.name);

  constructor(
    @Inject(DB_POOL) private pool: Pool,
    private config: ConfigService,
    private jwt: JwtService,
  ) {}

  /**
   * Mint a short-lived per-device JWT used to authenticate the agent WebSocket
   * against the realtime engine. The engine verifies it locally with JWT_SECRET
   * (the same secret this service signs with), so no synchronous cross-service
   * call is required on the hot path. Each client bootstraps this token from its
   * own license key at activation — nothing is manually distributed per client.
   */
  private mintWsToken(deviceId: string, role: 'data' | 'exec', sessionId?: string, licenseId?: string): string {
    return this.jwt.sign(
      {
        sub: deviceId,
        typ: 'agent-ws',
        role,
        ...(sessionId ? { sid: sessionId } : {}),
        ...(licenseId ? { lic: licenseId } : {}),
      },
      { expiresIn: '24h' },
    );
  }


  /**
   * Activate a device against a license key.
   * Implements: one license = one device, transactional binding, hardware fingerprint matching.
   */
  async activate(body: ActivationRequest, sourceIp?: string): Promise<ActivationResponse> {
    const { license_key, client_type, role, fingerprint, terminal, mt_account } = body;

    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      // Lock the license row
      const licenseResult = await client.query(
        `SELECT id, user_id, status, max_devices, expires_at FROM licensing.licenses
         WHERE license_key = $1 FOR UPDATE`,
        [license_key],
      );

      if (licenseResult.rows.length === 0) {
        await client.query('ROLLBACK');
        throw new NotFoundException('License not found');
      }

      const license = licenseResult.rows[0];

      if (license.status === 'REVOKED') throw new UnauthorizedException('License revoked');
      if (license.status === 'SUSPENDED') throw new UnauthorizedException('License suspended');
      if (license.status === 'EXPIRED' || (license.expires_at && new Date(license.expires_at) < new Date())) {
        throw new UnauthorizedException('License expired');
      }
      if (license.status !== 'ACTIVE' && license.status !== 'PENDING') {
        throw new UnauthorizedException(`License status: ${license.status}`);
      }

      // H4 fix: support MULTIPLE devices up to the license plan's max_devices.
      // Match the incoming fingerprint against ALL bound devices; reuse a match,
      // otherwise bind a new device if under the plan limit, else reject.
      const existingDevices = await client.query(
        `SELECT id, fingerprint_hash, fingerprint_components, installation_id, device_credential_hash
         FROM licensing.devices WHERE bound_license_id = $1 AND revoked_at IS NULL`,
        [license.id],
      );

      let deviceId: string;
      let isNewDevice = false;

      const bindNewDevice = async () => {
        isNewDevice = true;
        deviceId = crypto.randomUUID();
        const fpHash = this.computeFingerprintHash(fingerprint);
        const installationId = fingerprint.installation_id || crypto.randomUUID();
        const hashedComponents = this.hashFingerprintComponents(fingerprint);
        await client.query(
          `INSERT INTO licensing.devices (id, user_id, bound_license_id, installation_id, fingerprint_version,
             fingerprint_hash, fingerprint_components, device_name, windows_version, role, connection_status, last_activation_at, last_seen_at, created_at, updated_at)
            VALUES ($1, $2, $3, $4, 'hwfp-v1', $5, $6, $7, $8, $9, 'ONLINE', now(), now(), now(), now())`,
          [deviceId, license.user_id, license.id, installationId, fpHash, JSON.stringify(hashedComponents),
            terminal?.name || `${client_type} Terminal`, fingerprint.os || 'Windows', role],
        );
        await client.query(
          `UPDATE licensing.licenses SET status = 'ACTIVE', updated_at = now() WHERE id = $1`,
          [license.id],
        );
      };

      if (existingDevices.rows.length === 0) {
        await bindNewDevice();
      } else {
        let matched: any = null;
        let bestScore = -1;
        for (const d of existingDevices.rows) {
          const score = this.computeMatchScore(fingerprint, d.fingerprint_components || {});
          if (score > bestScore) { bestScore = score; matched = d; }
        }
        if (matched && bestScore >= MATCH_THRESHOLD) {
          deviceId = matched.id;
          // Persist the role of THIS activation — a device re-activating with a
          // different role flips its stored role (Master re-attach on a new
          // chart must stay 'data', not inherit a stale value).
          await client.query(
            `UPDATE licensing.devices SET last_seen_at = now(), last_activation_at = now(), role = $2, updated_at = now() WHERE id = $1`,
            [deviceId, role],
          );
        } else if (existingDevices.rows.length < (Number(license.max_devices) || 1)) {
          await bindNewDevice();
        } else {
          await client.query('ROLLBACK');
          this.logger.warn(`Device limit exceeded for license ${license.id}: ${existingDevices.rows.length}/${license.max_devices}`);
          throw new ConflictException('DEVICE_LIMIT_EXCEEDED');
        }
      }

      // Generate device credential (long-lived HMAC secret)
      const deviceSecret = crypto.randomBytes(32).toString('base64url');
      const credentialHash = this.hashSecret(deviceSecret);
      const tokenFamily = crypto.randomUUID();

      // Revoke old credentials
      await client.query(
        `UPDATE licensing.device_credentials SET revoked_at = now(), revocation_reason = ' superseded' WHERE device_id = $1 AND revoked_at IS NULL`,
        [deviceId],
      );

      // Insert new credential — store encrypted secret (AES-256-GCM) for HMAC verification
      const encryptedSecret = this.encryptSecret(deviceSecret);
      await client.query(
        `INSERT INTO licensing.device_credentials (id, device_id, credential_hash, token_family, issued_at, expires_at, created_at)
         VALUES ($1, $2, $3, $4, now(), now() + interval '30 days', now())`,
        [crypto.randomUUID(), deviceId, encryptedSecret, tokenFamily],
      );

      // Generate refresh token (rotating)
      const refreshToken = crypto.randomBytes(48).toString('base64url');
      const refreshTokenHash = this.hashSecret(refreshToken);

      await client.query(
        `INSERT INTO licensing.refresh_tokens (id, device_id, token_hash, token_family, issued_at, expires_at, created_at)
         VALUES ($1, $2, $3, $4, now(), now() + interval '30 days', now())`,
        [crypto.randomUUID(), deviceId, refreshTokenHash, tokenFamily],
      );

      // Create session lease
      const sessionId = crypto.randomUUID();
      const leaseExpires = new Date(Date.now() + 45 * 1000); // 45 seconds

      // Short-lived ingest JWT (HS256, sub=device, role=data|exec, 24h).
      // THE ENGINE REQUIRES A REAL JWT: POST /ingest/agent verifies the
      // Bearer locally with JWT_SECRET (validateJWTFull — 3 dot-separated
      // parts, HS256, exp). An opaque random string here makes EVERY EA
      // ingest 401 ("invalid token format: expected 3 parts") — the root
      // cause of the 2026-09-02 feed-stale incident. The EAs send
      // access_token as the ingest Bearer, so this field MUST be the JWT.
      const accessToken = this.mintWsToken(deviceId, role, sessionId, license.id);
      const accessTokenHash = this.hashSecret(accessToken);

      // Revoke old sessions for this license
      await client.query(
        `UPDATE licensing.session_leases SET status = 'EXPIRED', revoked_at = now(), revocation_reason = 'superseded by new activation' WHERE license_id = $1 AND status = 'ACTIVE'`,
        [license.id],
      );

      await client.query(
        `INSERT INTO licensing.session_leases (id, license_id, device_id, session_id, status, last_heartbeat_at, lease_expires_at, source_ip, created_at)
         VALUES ($1, $2, $3, $4, 'ACTIVE', now(), $5, $6, now())`,
        [crypto.randomUUID(), license.id, deviceId, sessionId, leaseExpires, sourceIp],
      );

      // Record activation
      await client.query(
        `INSERT INTO licensing.device_activations (id, license_id, device_id, client_type, terminal_build, ea_version,
         broker_name, broker_server, mt_account_login, installation_id, fingerprint_version, fingerprint_hash, fingerprint_components, activation_ip, activated_at, created_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'hwfp-v1', $11, $12, $13, now(), now())`,
        [crypto.randomUUID(), license.id, deviceId, client_type, terminal?.build, terminal?.ea_version,
         mt_account?.broker, mt_account?.server, mt_account?.login, fingerprint.installation_id,
         this.computeFingerprintHash(fingerprint), JSON.stringify(this.hashFingerprintComponents(fingerprint)), sourceIp],
      );

      // Audit log
      await client.query(
        `INSERT INTO licensing.license_events (license_id, event_type, reason, metadata, created_at)
         VALUES ($1, 'ACTIVATED', $2, $3, now())`,
        [license.id, isNewDevice ? 'First device activation' : 'Device re-activation',
         JSON.stringify({ device_id: deviceId, client_type, role, match_score: isNewDevice ? 100 : undefined })],
      );

      await client.query('COMMIT');

      return {
        device_id: deviceId,
        session_id: sessionId,
        device_secret: deviceSecret,
        refresh_token: refreshToken,
        access_token: accessToken,
        access_token_expires_in: 600,
        token_family: tokenFamily,
        ws_token: this.mintWsToken(deviceId, role, sessionId, license.id),
        fingerprint_match_score: isNewDevice ? 100 : undefined,
      };
    } catch (err) {
      try { await client.query("ROLLBACK"); } catch (e) { console.error("Rollback failed:", e.message); }
      throw err;
    } finally {
      client.release();
    }
  }

  /**
   * Refresh access token using rotating refresh token.
   * Implements: token rotation, family-based reuse detection.
   */
  async refresh(refreshToken: string, deviceId: string | undefined, role?: 'data' | 'exec'): Promise<RefreshResponse> {
    const tokenHash = this.hashSecret(refreshToken);

    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      const tokenResult = await client.query(
        `SELECT id, device_id, token_family, expires_at, used_at, revoked_at
         FROM licensing.refresh_tokens WHERE token_hash = $1 FOR UPDATE`,
        [tokenHash],
      );

      if (tokenResult.rows.length === 0) {
        await client.query('ROLLBACK');
        throw new UnauthorizedException('Invalid refresh token');
      }

      const token = tokenResult.rows[0];

      // Reuse detection — already used or revoked
      if (token.used_at || token.revoked_at) {
        // Revoke entire family
        await client.query(
          `UPDATE licensing.refresh_tokens SET revoked_at = now(), revoked_reason = 'reuse detected' WHERE token_family = $1 AND revoked_at IS NULL`,
          [token.token_family],
        );
        await client.query(
          `UPDATE licensing.device_credentials SET revoked_at = now(), revocation_reason = 'refresh reuse' WHERE token_family = $1 AND revoked_at IS NULL`,
          [token.token_family],
        );
        await client.query(
          `UPDATE licensing.session_leases SET status = 'REVOKED', revoked_at = now(), revocation_reason = 'refresh reuse' WHERE device_id = $1 AND status = 'ACTIVE'`,
          [token.device_id],
        );
        await client.query('COMMIT');
        this.logger.warn(`Refresh token reuse detected for device ${token.device_id}`);
        throw new UnauthorizedException('Session invalidated due to token reuse');
      }

      if (new Date(token.expires_at) < new Date()) {
        await client.query('ROLLBACK');
        throw new UnauthorizedException('Refresh token expired');
      }

      // Verify device is still active
      // deviceId may be undefined — MT4/MT5 EAs send only refresh_token; the
      // device is derived from the token row itself. When deviceId IS provided
      // it must match (prevents cross-device token use).
      if (deviceId !== undefined && token.device_id !== deviceId) {
        await client.query('ROLLBACK');
        throw new UnauthorizedException('Device mismatch');
      }

      // Rotate: mark old token as used
      await client.query(
        `UPDATE licensing.refresh_tokens SET used_at = now() WHERE id = $1`,
        [token.id],
      );

      // Issue new refresh token
      const newRefreshToken = crypto.randomBytes(48).toString('base64url');
      const newRefreshHash = this.hashSecret(newRefreshToken);

      await client.query(
        `INSERT INTO licensing.refresh_tokens (id, device_id, token_hash, token_family, issued_at, expires_at, created_at)
         VALUES ($1, $2, $3, $4, now(), now() + interval '30 days', now())`,
        [crypto.randomUUID(), token.device_id, newRefreshHash, token.token_family],
      );

      // Issue new access token — a REAL ingest JWT, not an opaque string.
      // The EAs send access_token as the POST /ingest/agent Bearer; the engine
      // verifies it locally (validateJWTFull). Role comes from the device row
      // (persisted at activation, migration 119) so a refreshed MASTER token
      // carries role='data' — with the old 'exec' default the engine dropped
      // every Master MARKET_SNAPSHOT after the first 24h token expired.
      const roleResult = await client.query(
        `SELECT role FROM licensing.devices WHERE id = $1`,
        [token.device_id],
      );
      const deviceRole = (roleResult.rows[0]?.role === 'data' ? 'data' : 'exec') as 'data' | 'exec';
      const newAccessToken = this.mintWsToken(token.device_id, deviceRole);

      // Renew session lease
      await client.query(
        `UPDATE licensing.session_leases SET last_heartbeat_at = now(), lease_expires_at = now() + interval '45 seconds' WHERE device_id = $1 AND status = 'ACTIVE'`,
        [token.device_id],
      );

      // Keep the device itself ONLINE so dashboards reflect a live, connected agent.
      // (Previously only the session lease was touched, leaving licensing.devices.connection_status
      //  stale as OFFLINE even when the agent was actively heartbeating.)
      // NOTE: token.device_id — EAs send refresh_token only, the deviceId arg
      // is undefined, and the old WHERE id = $1 silently no-op'd.
      await client.query(
        `UPDATE licensing.devices SET connection_status = 'ONLINE', last_seen_at = now(), updated_at = now()
         WHERE id = $1 AND deleted_at IS NULL`,
        [token.device_id],
      );

      await client.query('COMMIT');

      return {
        access_token: newAccessToken,
        refresh_token: newRefreshToken,
        access_token_expires_in: 600,
        ws_token: newAccessToken,
      };
    } catch (err) {
      try { await client.query("ROLLBACK"); } catch (e) { console.error("Rollback failed:", e.message); }
      throw err;
    } finally {
      client.release();
    }
  }

  /**
   * Process heartbeat from a connected device.
   * Renews session lease. Returns current trading/connection state separately.
   */
  async heartbeat(deviceId: string, sessionId: string, body: any, sourceIp?: string): Promise<HeartbeatResponse> {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      const leaseResult = await client.query(
        `SELECT sl.id, sl.license_id, sl.status, l.status as license_status
         FROM licensing.session_leases sl
         JOIN licensing.licenses l ON l.id = sl.license_id
         WHERE sl.device_id = $1 AND sl.session_id = $2 FOR UPDATE`,
        [deviceId, sessionId],
      );

      if (leaseResult.rows.length === 0) {
        await client.query('ROLLBACK');
        throw new UnauthorizedException('Session not found');
      }

      const lease = leaseResult.rows[0];

      // Update terminal account data from heartbeat (balance, equity, P&L, positions)
      if (body.terminals && Array.isArray(body.terminals)) {
        for (const term of body.terminals) {
          if (term.account && term.balance !== undefined) {
            const xau = term.xauusd && term.xauusd.available ? term.xauusd : null;
            await client.query(
              `UPDATE licensing.device_activations SET
                account_balance = $1, account_equity = $2, account_profit = $3,
                open_positions = $4, buy_positions = $5, sell_positions = $6,
                total_lots = $7, floating_pnl = $8, last_account_update = now(),
                terminal_connected = $9, terminal_version = $10,
                xauusd_available = $11, xauusd_bid = $12, xauusd_ask = $13,
                xauusd_spread = $14, xauusd_last_tick_time = $15,
                account_currency = COALESCE($16, account_currency),
                leverage = COALESCE($17, leverage),
                margin = COALESCE($18, margin),
                free_margin = COALESCE($19, free_margin),
                margin_level = COALESCE($20, margin_level),
                account_type = COALESCE($21, account_type),
                pending_orders_count = COALESCE($22, pending_orders_count)
               WHERE device_id = $23 AND mt_account_login = $24`,
              [term.balance || 0, term.equity || 0, term.profit || 0,
                term.open_positions || 0, term.buy_positions || 0, term.sell_positions || 0,
                term.total_lots || 0, term.floating_pnl || 0,
                term.connected === false ? false : true,
                term.terminal_version || null,
                xau ? true : false,
                xau ? (xau.bid ?? null) : null,
                xau ? (xau.ask ?? null) : null,
                xau ? (xau.spread ?? null) : null,
                xau ? (xau.last_tick_time || null) : null,
                term.currency || null,
                term.leverage != null ? term.leverage : null,
                term.margin != null ? term.margin : null,
                term.free_margin != null ? term.free_margin : null,
                term.margin_level != null ? term.margin_level : null,
                term.account_type || null,
                term.pending_orders_count != null ? term.pending_orders_count : null,
                deviceId, term.account],
            );
          }
        }
      }

      // Persist genuine agent/OS operational metadata (go-prompt §3-8)
      await client.query(
        `UPDATE licensing.devices SET
           agent_version = COALESCE($2, agent_version),
           os_name = COALESCE($3, os_name),
           architecture = COALESCE($4, architecture),
           agent_uptime_seconds = $5,
           service_status = COALESCE($6, service_status),
           health_status = COALESCE($7, health_status),
           hostname = COALESCE($8, hostname),
           agent_started_at = COALESCE($9, agent_started_at),
           updated_at = now()
         WHERE id = $1`,
        [deviceId,
          body.agent_version || null,
          body.os_name || null,
          body.architecture || null,
          body.agent_uptime_seconds != null ? body.agent_uptime_seconds : null,
          body.service_status || null,
          body.health_status || null,
          body.hostname || null,
          body.agent_started_at || null],
      );

      if (lease.status === 'REVOKED') {
        await client.query('ROLLBACK');
        throw new UnauthorizedException('Session revoked');
      }

      // Renew lease
      await client.query(
        `UPDATE licensing.session_leases SET last_heartbeat_at = now(), lease_expires_at = now() + interval '45 seconds',
         status = 'ACTIVE', source_ip = $2 WHERE id = $1`,
        [lease.id, sourceIp],
      );

      await client.query(
        `UPDATE licensing.devices SET last_seen_at = now(), connection_status = 'ONLINE' WHERE id = $1`,
        [deviceId],
      );

      await client.query('COMMIT');

      // Return independent states — NOT conflated
      return {
        connection: 'ONLINE',
        auth: 'AUTHENTICATED',
        license: lease.license_status || 'ACTIVE',
        device: 'AUTHORIZED',
        session: 'ACTIVE',
        trading: body.trading_mode || 'UNKNOWN',
        server_time: new Date().toISOString(),
        lease_expires_in: 45,
      };
    } catch (err) {
      try { await client.query("ROLLBACK"); } catch (e) { console.error("Rollback failed:", e.message); }
      throw err;
    } finally {
      client.release();
    }
  }

  /** Admin: revoke device */
  async revokeDevice(deviceId: string, reason: string) {
    await this.pool.query(
      `UPDATE licensing.devices SET revoked_at = now(), revocation_reason = $2, connection_status = 'OFFLINE' WHERE id = $1`,
      [deviceId, reason],
    );
    await this.pool.query(
      `UPDATE licensing.session_leases SET status = 'REVOKED', revoked_at = now(), revocation_reason = $2 WHERE device_id = $1 AND status = 'ACTIVE'`,
      [deviceId, reason],
    );
    await this.pool.query(
      `UPDATE licensing.device_credentials SET revoked_at = now(), revocation_reason = $2 WHERE device_id = $1 AND revoked_at IS NULL`,
      [deviceId, reason],
    );
    return { success: true };
  }

  /** Admin: get device details with session info */
  async getDeviceDetails(deviceId: string) {
    const r = await this.pool.query(
      `SELECT d.*, sl.session_id, sl.status as session_status, sl.last_heartbeat_at, sl.lease_expires_at,
        l.license_key, l.status as license_status
       FROM licensing.devices d
       LEFT JOIN licensing.session_leases sl ON sl.device_id = d.id AND sl.status = 'ACTIVE'
       LEFT JOIN licensing.licenses l ON l.id = d.bound_license_id
       WHERE d.id = $1`,
      [deviceId],
    );
    if (r.rows.length === 0) throw new NotFoundException('Device not found');
    return r.rows[0];
  }

  /** Admin: list active sessions */
  async listActiveSessions() {
    const r = await this.pool.query(
      `SELECT sl.*, d.device_name, l.license_key, u.email as user_email
       FROM licensing.session_leases sl
       JOIN licensing.devices d ON d.id = sl.device_id
       JOIN licensing.licenses l ON l.id = sl.license_id
       JOIN iam.users u ON u.id = l.user_id
       WHERE sl.status = 'ACTIVE' ORDER BY sl.last_heartbeat_at DESC`,
    );
    return r.rows;
  }

  /** Compute weighted match score between new and stored fingerprint */
  private computeMatchScore(newFp: any, storedFp: any): number {
    let totalWeight = 0;
    let matchedWeight = 0;
    let storedPresent = 0;

    // Score over the components the STORED device actually has. Legacy/EA
    // devices may only send a subset (MT4/MT5 EAs send machine_guid only);
    // requiring score >= 75 over the FULL weight table made them never match,
    // so every activation retry bound a brand-new device until slots ran out
    // (409 DEVICE_LIMIT_EXCEEDED on a fresh 2-slot license).
    for (const [component, weight] of Object.entries(FINGERPRINT_WEIGHTS)) {
      const storedVal = storedFp[component];
      if (!storedVal) continue; // component not collected for this device
      storedPresent += weight;
      const newVal = newFp[component] ? this.hashComponent(newFp[component]) : null;
      if (newVal && newVal === storedVal) {
        matchedWeight += weight;
      }
    }

    if (storedPresent === 0) return 0;
    return Math.round((matchedWeight / storedPresent) * 100);
  }

  /** Hash all fingerprint components with pepper for privacy-aware storage */
  private hashFingerprintComponents(fp: any): Record<string, string> {
    const result: Record<string, string> = {};
    for (const key of ['machine_guid', 'system_uuid', 'motherboard', 'disk', 'installation_id']) {
      if (fp[key]) {
        result[key] = this.hashComponent(fp[key]);
      }
    }
    return result;
  }

  private computeFingerprintHash(fp: any): string {
    const components = ['machine_guid', 'system_uuid', 'motherboard', 'disk', 'installation_id'];
    const hashes = components.map(c => fp[c] ? this.hashComponent(fp[c]) : '').join('|');
    return crypto.createHash('sha256').update(hashes).digest('hex');
  }

  private get jwtSecret(): string {
    return this.config.get<string>('JWT_SECRET') || DEV_JWT_SECRET;
  }

  private hashComponent(value: string): string {
    const pepper = this.jwtSecret;
    return crypto.createHmac('sha256', pepper).update(value).digest('hex');
  }

  private hashSecret(secret: string): string {
    return crypto.createHash('sha256').update(secret).digest('hex');
  }

  // ─── Device Secret Encryption (AES-256-GCM) ───
  // The device secret is stored encrypted so the server can decrypt it for HMAC verification.
  // The encryption key is derived from JWT_SECRET (never logged, never exposed).

  private getEncryptionKey(): Buffer {
    const baseKey = this.jwtSecret;
    // Derive a 32-byte key using SHA-256
    return crypto.createHash('sha256').update(baseKey + ':device_secret_encryption').digest();
  }

  private encryptSecret(plaintext: string): string {
    const key = this.getEncryptionKey();
    const iv = crypto.randomBytes(12); // AES-GCM standard IV size
    const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);
    const encrypted = Buffer.concat([cipher.update(plaintext, 'utf8'), cipher.final()]);
    const authTag = cipher.getAuthTag();
    // Format: base64(iv) + ':' + base64(authTag) + ':' + base64(encrypted)
    return `${iv.toString('base64')}:${authTag.toString('base64')}:${encrypted.toString('base64')}`;
  }

  private decryptSecret(ciphertext: string): string | null {
    try {
      const parts = ciphertext.split(':');
      if (parts.length !== 3) return null;
      const iv = Buffer.from(parts[0], 'base64');
      const authTag = Buffer.from(parts[1], 'base64');
      const encrypted = Buffer.from(parts[2], 'base64');
      const key = this.getEncryptionKey();
      const decipher = crypto.createDecipheriv('aes-256-gcm', key, iv);
      decipher.setAuthTag(authTag);
      const decrypted = Buffer.concat([decipher.update(encrypted), decipher.final()]);
      return decrypted.toString('utf8');
    } catch {
      return null;
    }
  }

  // ─── Complete HMAC Request Signature Verification ───
  // SOW Section 11: Proof-of-Device Request Signing
  // Canonical format: version\ntimestamp\nnonce\nmethod\npath\nbodyHash\ndeviceId
  // Signature: HMAC-SHA256(device_secret, canonical_request)

  async verifyRequestSignature(params: {
    deviceId: string;
    method: string;
    path: string;
    bodyHash: string;
    timestamp: string;
    nonce: string;
    signature: string;
    version?: string;
  }): Promise<{ valid: boolean; reason?: string }> {
    const { deviceId, method, path, bodyHash, timestamp, nonce, signature } = params;
    const version = params.version || 'v1';
    const clockSkewSeconds = 30;

    // 1. Verify timestamp window
    const ts = parseInt(timestamp, 10);
    if (isNaN(ts)) return { valid: false, reason: 'INVALID_TIMESTAMP' };
    const now = Date.now();
    const skew = Math.abs(now - ts);
    if (skew > clockSkewSeconds * 1000) {
      return { valid: false, reason: 'TIMESTAMP_OUT_OF_WINDOW' };
    }

    // 2. Look up active device credential with encrypted secret
    const credResult = await this.pool.query(
      `SELECT dc.id, dc.credential_hash, dc.token_family, dc.expires_at, dc.revoked_at,
              d.bound_license_id, d.revoked_at as device_revoked, d.connection_status,
              l.status as license_status
       FROM licensing.device_credentials dc
       JOIN licensing.devices d ON d.id = dc.device_id
       LEFT JOIN licensing.licenses l ON l.id = d.bound_license_id
       WHERE dc.device_id = $1 AND dc.revoked_at IS NULL
       ORDER BY dc.issued_at DESC LIMIT 1`,
      [deviceId],
    );

    if (credResult.rows.length === 0) {
      return { valid: false, reason: 'NO_ACTIVE_CREDENTIAL' };
    }

    const cred = credResult.rows[0];

    // 3. Verify device is not revoked
    if (cred.device_revoked) {
      return { valid: false, reason: 'DEVICE_REVOKED' };
    }

    // 4. Verify credential is not expired
    if (cred.expires_at && new Date(cred.expires_at) < new Date()) {
      return { valid: false, reason: 'CREDENTIAL_EXPIRED' };
    }

    // 5. Verify license is active
    if (cred.license_status && cred.license_status !== 'ACTIVE') {
      return { valid: false, reason: 'LICENSE_NOT_ACTIVE' };
    }

    // 6. Check nonce hasn't been used (replay protection)
    const nonceExists = await this.pool.query(
      `SELECT 1 FROM licensing.request_nonces WHERE nonce = $1`,
      [nonce],
    );
    if (nonceExists.rows.length > 0) {
      return { valid: false, reason: 'NONCE_ALREADY_USED' };
    }

    // Store nonce with TTL
    await this.pool.query(
      `INSERT INTO licensing.request_nonces (nonce, device_id, created_at, expires_at)
       VALUES ($1, $2, now(), now() + interval '120 seconds')
       ON CONFLICT DO NOTHING`,
      [nonce, deviceId],
    );

    // 7. Decrypt the device secret
    // The credential_hash column stores the encrypted secret (not just a hash)
    // This allows the server to decrypt and use it for HMAC verification
    const deviceSecret = this.decryptSecret(cred.credential_hash);
    if (!deviceSecret) {
      return { valid: false, reason: 'SECRET_DECRYPTION_FAILED' };
    }

    // 8. Build canonical request string
    const canonical = `${version}\n${timestamp}\n${nonce}\n${method.toUpperCase()}\n${path}\n${bodyHash}\n${deviceId}`;

    // 9. Compute expected signature
    const expectedSignature = crypto.createHmac('sha256', deviceSecret).update(canonical).digest('hex');

    // 10. Constant-time comparison (with length check for malformed signatures)
    const expectedBuf = Buffer.from(expectedSignature, 'hex');
    let sigBuf: Buffer;
    try {
      sigBuf = Buffer.from(signature, 'hex');
    } catch {
      return { valid: false, reason: 'MALFORMED_SIGNATURE' };
    }
    if (expectedBuf.length !== sigBuf.length) {
      return { valid: false, reason: 'SIGNATURE_MISMATCH' };
    }
    const valid = crypto.timingSafeEqual(expectedBuf, sigBuf);

    if (!valid) {
      return { valid: false, reason: 'SIGNATURE_MISMATCH' };
    }

    return { valid: true };
  }
}

export interface ActivationRequest {
  license_key: string;
  client_type: 'MT4' | 'MT5';
  role?: 'data' | 'exec';
  fingerprint: {
    machine_guid?: string;
    system_uuid?: string;
    motherboard?: string;
    disk?: string;
    installation_id?: string;
    os?: string;
  };
  terminal?: {
    build?: string;
    name?: string;
    ea_version?: string;
  };
  mt_account?: {
    broker?: string;
    server?: string;
    login?: string;
  };
}

export interface ActivationResponse {
  device_id: string;
  session_id: string;
  device_secret: string;
  refresh_token: string;
  access_token: string;
  access_token_expires_in: number;
  token_family: string;
  ws_token: string;
  fingerprint_match_score?: number;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
  access_token_expires_in: number;
  ws_token: string;
}

export interface HeartbeatResponse {
  connection: string;
  auth: string;
  license: string;
  device: string;
  session: string;
  trading: string;
  server_time: string;
  lease_expires_in: number;
}
