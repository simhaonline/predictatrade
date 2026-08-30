/* eslint-disable @typescript-eslint/no-explicit-any */
import { DeviceAuthService } from './device-auth.service';
import { jest } from '@jest/globals';
import { UnauthorizedException, ConflictException, NotFoundException } from '@nestjs/common';
import * as crypto from 'crypto';

describe('DeviceAuthService', () => {
  let service: DeviceAuthService;
  let pool: any;

  beforeEach(() => {
    pool = {
      connect: jest.fn(),
      query: jest.fn(),
    };
    service = new DeviceAuthService(pool, { get: (k: string) => process.env[k] ?? 'pat_local_dev_secret_change_in_production' } as any);
  });

  describe('activate', () => {
    it('rejects missing license key', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      client.query.mockImplementation(async (sql: string) => {
        if (sql.trim().toUpperCase().startsWith('BEGIN') || sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY') || sql.includes('ROLLBACK') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, user_id')) return { rows: [] };
        return { rows: [] };
      });
      await expect(service.activate({ license_key: '', client_type: 'MT5', fingerprint: {} }, '1.2.3.4'))
        .rejects.toThrow(NotFoundException);
    });

    it('rejects revoked license', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      let callCount = 0;
      client.query.mockImplementation(async (sql: string) => {
        callCount++;
        if (sql.includes('BEGIN') || sql.includes('ROLLBACK') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, user_id') && callCount === 2) {
          return { rows: [{ id: 'lic-1', user_id: 'u-1', status: 'REVOKED', max_devices: 1, expires_at: null }] };
        }
        return { rows: [] };
      });
      await expect(service.activate({ license_key: 'TEST-KEY', client_type: 'MT5', fingerprint: { installation_id: 'test' } }, '1.2.3.4'))
        .rejects.toThrow(UnauthorizedException);
    });

    it('rejects expired license', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      let callCount = 0;
      client.query.mockImplementation(async (sql: string) => {
        callCount++;
        if (sql.includes('BEGIN') || sql.includes('ROLLBACK') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, user_id') && callCount === 2) {
          return { rows: [{ id: 'lic-1', user_id: 'u-1', status: 'ACTIVE', max_devices: 1, expires_at: new Date(Date.now() - 10000) }] };
        }
        return { rows: [] };
      });
      await expect(service.activate({ license_key: 'TEST-KEY', client_type: 'MT5', fingerprint: { installation_id: 'test' } }, '1.2.3.4'))
        .rejects.toThrow(UnauthorizedException);
    });

    it('creates device on first activation', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      let callCount = 0;
      client.query.mockImplementation(async (sql: string) => {
        callCount++;
        if (sql.includes('BEGIN') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, user_id')) return { rows: [{ id: 'lic-1', user_id: 'u-1', status: 'ACTIVE', max_devices: 1, expires_at: null }] };
        if (sql.includes('SELECT id, fingerprint_hash')) return { rows: [] }; // no existing device
        return { rows: [] };
      });
      const result = await service.activate({ license_key: 'TEST-KEY', client_type: 'MT5', fingerprint: { machine_guid: 'mg1', installation_id: 'inst1' } }, '1.2.3.4');
      expect(result.device_id).toBeDefined();
      expect(result.session_id).toBeDefined();
      expect(result.device_secret).toHaveLength(43); // 32 bytes base64url
      expect(result.refresh_token).toHaveLength(64); // 48 bytes base64url
      expect(result.access_token).toHaveLength(43);
      expect(result.access_token_expires_in).toBe(600);
      expect(result.token_family).toBeDefined();
      expect(result.fingerprint_match_score).toBe(100);
    });

    it('rejects different device on same license (DEVICE_ALREADY_BOUND)', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      let callCount = 0;
      client.query.mockImplementation(async (sql: string) => {
        callCount++;
        if (sql.includes('BEGIN') || sql.includes('ROLLBACK') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, user_id')) return { rows: [{ id: 'lic-1', user_id: 'u-1', status: 'ACTIVE', max_devices: 1, expires_at: null }] };
        if (sql.includes('SELECT id, fingerprint_hash')) return { rows: [{ id: 'dev-1', fingerprint_hash: 'old', fingerprint_components: { machine_guid: 'different' } }] };
        return { rows: [] };
      });
      await expect(service.activate({ license_key: 'TEST-KEY', client_type: 'MT5', fingerprint: { machine_guid: 'completely_different', installation_id: 'new_install' } }, '1.2.3.4'))
        .rejects.toThrow(ConflictException);
    });
  });

  describe('refresh', () => {
    it('rejects invalid refresh token', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      client.query.mockImplementation(async (sql: string) => {
        if (sql.includes('BEGIN') || sql.includes('ROLLBACK') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, device_id')) return { rows: [] }; // not found
        return { rows: [] };
      });
      await expect(service.refresh('invalid_token', 'dev-1')).rejects.toThrow(UnauthorizedException);
    });

    it('detects reuse of already-rotated token and revokes family', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      let callCount = 0;
      client.query.mockImplementation(async (sql: string) => {
        callCount++;
        if (sql.includes('BEGIN') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, device_id') && callCount === 2) {
          return { rows: [{ id: 'tok-1', device_id: 'dev-1', token_family: 'fam-1', expires_at: new Date(Date.now() + 10000), used_at: new Date(), revoked_at: null }] };
        }
        return { rows: [] };
      });
      await expect(service.refresh('reused_token', 'dev-1')).rejects.toThrow(UnauthorizedException);
    });
  });

  describe('heartbeat', () => {
    it('rejects unknown session', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      client.query.mockImplementation(async (sql: string) => {
        if (sql.includes('BEGIN') || sql.includes('ROLLBACK') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT sl.id')) return { rows: [] }; // not found
        return { rows: [] };
      });
      await expect(service.heartbeat('dev-1', 'sess-1', {}, '1.2.3.4')).rejects.toThrow(UnauthorizedException);
    });

    it('returns independent states (not conflated)', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      let callCount = 0;
      client.query.mockImplementation(async (sql: string) => {
        callCount++;
        if (sql.includes('BEGIN') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT sl.id') && callCount === 2) {
          return { rows: [{ id: 'lease-1', license_id: 'lic-1', status: 'ACTIVE', license_status: 'ACTIVE' }] };
        }
        return { rows: [] };
      });
      const result = await service.heartbeat('dev-1', 'sess-1', { trading_mode: 'HALTED' }, '1.2.3.4');
      expect(result.connection).toBe('ONLINE');
      expect(result.auth).toBe('AUTHENTICATED');
      expect(result.license).toBe('ACTIVE');
      expect(result.device).toBe('AUTHORIZED');
      expect(result.session).toBe('ACTIVE');
      expect(result.trading).toBe('HALTED'); // Trading halted ≠ disconnected
      expect(result.server_time).toBeDefined();
      expect(result.lease_expires_in).toBe(45);
    });
  });

  describe('fingerprint matching', () => {
    it('same fingerprint components should match', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      let callCount = 0;
      client.query.mockImplementation(async (sql: string) => {
        callCount++;
        if (sql.includes('BEGIN') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, user_id')) return { rows: [{ id: 'lic-1', user_id: 'u-1', status: 'ACTIVE', max_devices: 1, expires_at: null }] };
        if (sql.includes('SELECT id, fingerprint_hash')) return { rows: [] }; // first activation
        return { rows: [] };
      });
      const fp = { machine_guid: 'mg1', system_uuid: 'uuid1', motherboard: 'mb1', disk: 'd1', installation_id: 'inst1' };
      const result = await service.activate({ license_key: 'TEST-KEY', client_type: 'MT5', fingerprint: fp }, '1.2.3.4');
      expect(result.fingerprint_match_score).toBe(100);
    });
  });

  describe('secret redaction', () => {
    it('activation response contains device_secret (for client, not logged)', async () => {
      const client = { query: jest.fn(), release: jest.fn() };
      pool.connect.mockResolvedValue(client);
      client.query.mockImplementation(async (sql: string) => {
        if (sql.includes('BEGIN') || sql.includes('COMMIT')) return { rows: [] };
        if (sql.includes('SELECT id, user_id')) return { rows: [{ id: 'lic-1', user_id: 'u-1', status: 'ACTIVE', max_devices: 1, expires_at: null }] };
        if (sql.includes('SELECT id, fingerprint_hash')) return { rows: [] };
        return { rows: [] };
      });
      const result = await service.activate({ license_key: 'TEST', client_type: 'MT5', fingerprint: { installation_id: 'x' } }, '1.2.3.4');
      // Device secret is returned to the CLIENT for HMAC signing, but must never be logged
      expect(result.device_secret).toBeDefined();
      expect(result.device_secret).not.toContain(' '); // base64url
    });
  });
});

// ─── HMAC Request Signature Tests ───

describe('DeviceAuthService HMAC Verification', () => {
  let service: any;
  let pool: any;

  beforeEach(() => {
    pool = { connect: jest.fn(), query: jest.fn() };
    service = new DeviceAuthService(pool, { get: (k: string) => process.env[k] ?? 'pat_local_dev_secret_change_in_production' } as any);
  });

  describe('verifyRequestSignature', () => {
    const validTimestamp = Date.now().toString();
    const validNonce = 'test-nonce-' + Date.now();
    const validMethod = 'POST';
    const validPath = '/api/v1/devices/heartbeat';
    const validBodyHash = 'abc123';
    const validDeviceId = '00000000-0000-0000-0000-000000000001';

    // Helper to set up a valid credential in the mock
    function setupValidCredential() {
      // Mock the credential lookup to return a valid encrypted secret
      pool.query.mockImplementation(async (sql: string, params?: any[]) => {
        if (sql.includes('SELECT') && sql.includes('device_credentials')) {
          return { rows: [{
            id: 'cred-1',
            credential_hash: service.encryptSecret('test-device-secret'),
            token_family: 'fam-1',
            expires_at: new Date(Date.now() + 1000000),
            revoked_at: null,
            device_revoked: null,
            connection_status: 'ONLINE',
            license_status: 'ACTIVE',
          }] };
        }
        if (sql.includes('request_nonces') && sql.includes('SELECT 1')) {
          return { rows: [] }; // nonce not used yet
        }
        if (sql.includes('INSERT INTO licensing.request_nonces')) {
          return { rows: [] };
        }
        return { rows: [] };
      });
    }

    function computeValidSignature(secret: string, version: string, ts: string, nonce: string, method: string, path: string, bodyHash: string, deviceId: string): string {
      const canonical = `${version}\n${ts}\n${nonce}\n${method.toUpperCase()}\n${path}\n${bodyHash}\n${deviceId}`;
      return crypto.createHmac('sha256', secret).update(canonical).digest('hex');
    }

    it('accepts a valid signature', async () => {
      setupValidCredential();
      const signature = computeValidSignature('test-device-secret', 'v1', validTimestamp, validNonce, validMethod, validPath, validBodyHash, validDeviceId);
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature,
      });
      expect(result.valid).toBe(true);
    });

    it('rejects invalid signature (mismatch)', async () => {
      setupValidCredential();
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature: 'wrong-signature',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('SIGNATURE_MISMATCH');
    });

    it('rejects modified body hash', async () => {
      setupValidCredential();
      const signature = computeValidSignature('test-device-secret', 'v1', validTimestamp, validNonce, validMethod, validPath, validBodyHash, validDeviceId);
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: 'TAMPERED',
        timestamp: validTimestamp, nonce: validNonce, signature,
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('SIGNATURE_MISMATCH');
    });

    it('rejects modified path', async () => {
      setupValidCredential();
      const signature = computeValidSignature('test-device-secret', 'v1', validTimestamp, validNonce, validMethod, validPath, validBodyHash, validDeviceId);
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: '/different/path', bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature,
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('SIGNATURE_MISMATCH');
    });

    it('rejects modified method', async () => {
      setupValidCredential();
      const signature = computeValidSignature('test-device-secret', 'v1', validTimestamp, validNonce, validMethod, validPath, validBodyHash, validDeviceId);
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: 'GET', path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature,
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('SIGNATURE_MISMATCH');
    });

    it('rejects expired timestamp', async () => {
      setupValidCredential();
      const expiredTs = (Date.now() - 60000).toString(); // 60s ago
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: expiredTs, nonce: validNonce, signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('TIMESTAMP_OUT_OF_WINDOW');
    });

    it('rejects future timestamp outside tolerance', async () => {
      setupValidCredential();
      const futureTs = (Date.now() + 60000).toString(); // 60s in future
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: futureTs, nonce: validNonce, signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('TIMESTAMP_OUT_OF_WINDOW');
    });

    it('rejects unknown device (no credential)', async () => {
      pool.query.mockResolvedValue({ rows: [] });
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('NO_ACTIVE_CREDENTIAL');
    });

    it('rejects revoked device', async () => {
      pool.query.mockImplementation(async (sql: string) => {
        if (sql.includes('device_credentials')) {
          return { rows: [{ device_revoked: new Date(), credential_hash: 'x', expires_at: null, revoked_at: null, license_status: 'ACTIVE' }] };
        }
        return { rows: [] };
      });
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('DEVICE_REVOKED');
    });

    it('rejects expired credential', async () => {
      pool.query.mockImplementation(async (sql: string) => {
        if (sql.includes('device_credentials')) {
          return { rows: [{ device_revoked: null, credential_hash: 'x', expires_at: new Date(Date.now() - 1000), revoked_at: null, license_status: 'ACTIVE' }] };
        }
        return { rows: [] };
      });
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('CREDENTIAL_EXPIRED');
    });

    it('rejects when license is not active', async () => {
      pool.query.mockImplementation(async (sql: string) => {
        if (sql.includes('device_credentials')) {
          return { rows: [{ device_revoked: null, credential_hash: 'x', expires_at: null, revoked_at: null, license_status: 'SUSPENDED' }] };
        }
        return { rows: [] };
      });
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: validNonce, signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('LICENSE_NOT_ACTIVE');
    });

    it('rejects duplicate nonce (replay attack)', async () => {
      pool.query.mockImplementation(async (sql: string) => {
        if (sql.includes('device_credentials')) {
          return { rows: [{ device_revoked: null, credential_hash: service.encryptSecret('secret'), expires_at: null, revoked_at: null, license_status: 'ACTIVE' }] };
        }
        if (sql.includes('SELECT 1') && sql.includes('request_nonces')) {
          return { rows: [{ nonce: 'used' }] }; // nonce already used
        }
        return { rows: [] };
      });
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: validTimestamp, nonce: 'used-nonce', signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('NONCE_ALREADY_USED');
    });

    it('rejects malformed timestamp', async () => {
      const result = await service.verifyRequestSignature({
        deviceId: validDeviceId, method: validMethod, path: validPath, bodyHash: validBodyHash,
        timestamp: 'not-a-number', nonce: validNonce, signature: 'any',
      });
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('INVALID_TIMESTAMP');
    });

    it('encrypt and decrypt secret round-trip works', () => {
      const original = 'my-secret-device-key-12345';
      const encrypted = service.encryptSecret(original);
      const decrypted = service.decryptSecret(encrypted);
      expect(decrypted).toBe(original);
    });

    it('decrypt of tampered ciphertext fails', () => {
      const encrypted = service.encryptSecret('test');
      const tampered = encrypted.slice(0, -2) + 'XX'; // tamper last bytes
      const result = service.decryptSecret(tampered);
      expect(result).toBeNull();
    });
  });
});
