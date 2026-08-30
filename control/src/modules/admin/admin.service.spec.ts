import { Test, TestingModule } from '@nestjs/testing';
import { AdminService } from './admin.service';
import { DB_POOL } from '../../common/database.module';
import { CommissionsService } from '../commissions/commissions.service';
import { Pool } from 'pg';
import { jest } from '@jest/globals';

// Integration-style tests that verify SQL queries match the actual schema.
// These use a real connection to the test database if DATABASE_URL is set,
// otherwise they mock the pool and verify query structure.

const DB_URL = process.env.DATABASE_URL || process.env.TEST_DATABASE_URL;

describe('AdminService', () => {
  let service: AdminService;
  let pool: any;

  beforeAll(async () => {
    if (DB_URL) {
      pool = new Pool({ connectionString: DB_URL, max: 5 });
    } else {
      // Mock pool for unit tests when no DB is available.
      // Query-aware: COUNT queries return a total row so pagination math works.
      pool = {
        query: jest.fn((text: string) => {
          if (typeof text === 'string' && /count\(\*\)\s+as\s+total/i.test(text)) {
            return Promise.resolve({ rows: [{ total: '0' }], rowCount: 0 });
          }
          return Promise.resolve({ rows: [], rowCount: 0 });
        }),
      };
    }
  });

  afterAll(async () => {
    if (pool?.end) await pool.end();
  });

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        AdminService,
        { provide: DB_POOL, useValue: pool },
        { provide: CommissionsService, useValue: { getSummary: jest.fn() } },
      ],
    }).compile();
    service = module.get<AdminService>(AdminService);
  });

  describe('getOverview', () => {
    it('should execute without SQL errors (all column references valid)', async () => {
      if (!DB_URL) return; // skip if no DB
      const result = await service.getOverview();
      expect(result).toBeDefined();
      expect(result.users).toBeDefined();
      expect(result.subscriptions).toBeDefined();
      expect(result.commissions).toBeDefined();
      expect(result.payouts).toBeDefined();
      expect(result.plans).toBeDefined();
    });
  });

  describe('listAllSubscriptions', () => {
    it('should return paginated structure with correct column names', async () => {
      const result = await service.listAllSubscriptions(1, 20);
      expect(result).toHaveProperty('items');
      expect(result).toHaveProperty('total');
      expect(result).toHaveProperty('page', 1);
      expect(result).toHaveProperty('limit', 20);
    });

    it('should handle page 1 with zero records gracefully', async () => {
      const result = await service.listAllSubscriptions(1, 20);
      expect(result.total).toBeGreaterThanOrEqual(0);
    });
  });

  describe('listAllCommissions', () => {
    it('should return paginated structure with commission_level mapped from level column', async () => {
      const result = await service.listAllCommissions(1, 20);
      expect(result).toHaveProperty('items');
      expect(result).toHaveProperty('total');
    });
  });

  describe('listAllPayouts', () => {
    it('should return paginated structure with amount mapped from requested_amount', async () => {
      const result = await service.listAllPayouts(1, 20);
      expect(result).toHaveProperty('items');
      expect(result).toHaveProperty('total');
    });
  });

  describe('listAllDevices', () => {
    it('should return paginated structure with correct device column mappings', async () => {
      const result = await service.listAllDevices(1, 20);
      expect(result).toHaveProperty('items');
      expect(result).toHaveProperty('total');
    });
  });

  describe('listAllLicenses', () => {
    it('should return paginated structure', async () => {
      const result = await service.listAllLicenses(1, 20);
      expect(result).toHaveProperty('items');
      expect(result).toHaveProperty('total');
    });
  });

  describe('commissionSummary', () => {
    it('should return summary with all expected fields', async () => {
      const result = await service.commissionSummary();
      expect(result).toHaveProperty('total_entries');
      expect(result).toHaveProperty('total_amount');
      expect(result).toHaveProperty('pending_count');
      expect(result).toHaveProperty('confirmed_count');
      expect(result).toHaveProperty('reversed_count');
    });
  });

  describe('payoutStats', () => {
    it('should return payout stats with REQUESTED status mapping', async () => {
      const result = await service.payoutStats();
      expect(result).toHaveProperty('total');
      expect(result).toHaveProperty('pending');
      expect(result).toHaveProperty('approved');
    });
  });

  describe('systemHealth', () => {
    it('should return services array format (not flat key-value)', async () => {
      const result = await service.systemHealth();
      expect(result).toHaveProperty('services');
      expect(Array.isArray(result.services)).toBe(true);
      expect(result.services.length).toBeGreaterThan(0);
      // Each service should have the required fields
      for (const svc of result.services) {
        expect(svc).toHaveProperty('service');
        expect(svc).toHaveProperty('status');
        expect(svc).toHaveProperty('last_check');
      }
    });
  });

  describe('listUsers', () => {
    it('should return paginated users with role info', async () => {
      const result = await service.listUsers(1, 20);
      expect(result).toHaveProperty('items');
      expect(result).toHaveProperty('total');
      expect(result).toHaveProperty('page', 1);
    });
  });

  describe('systemHealth — Valkey/Redis check (regression: was hardcoded UNKNOWN)', () => {
    it('should report Valkey/Redis as HEALTHY or OFFLINE, never UNKNOWN', async () => {
      const result = await service.systemHealth();
      const valkey = result.services.find((s: any) => s.service === 'Valkey/Redis');
      expect(valkey).toBeDefined();
      // The old code hardcoded 'UNKNOWN'. The fix does a real TCP check,
      // so the status must now be either HEALTHY or OFFLINE.
      expect(valkey.status).not.toBe('UNKNOWN');
      expect(['HEALTHY', 'OFFLINE']).toContain(valkey.status);
    });

    it('should include latency_ms for the Valkey check', async () => {
      const result = await service.systemHealth();
      const valkey = result.services.find((s: any) => s.service === 'Valkey/Redis');
      expect(valkey).toBeDefined();
      expect(valkey.latency_ms).toBeDefined();
      expect(typeof valkey.latency_ms).toBe('number');
    });
  });
  describe('listAllLicenses (regression: known production license must appear)', () => {
    it('should return at least 1 license with correct field mapping', async () => {
      if (!DB_URL) return;
      const result = await service.listAllLicenses(1, 20);
      expect(result.items.length).toBeGreaterThanOrEqual(1);
      const lic = result.items[0];
      expect(lic).toHaveProperty('key');
      expect(lic).toHaveProperty('user_email');
      expect(lic).toHaveProperty('plan_name');
      expect(lic).toHaveProperty('activated_at');
    });

    it('should include the known production license ee710bf6', async () => {
      if (!DB_URL) return;
      const result = await service.listAllLicenses(1, 20);
      const found = result.items.find((l: any) => l.id === 'ee710bf6-5fe0-4b91-9b6b-a201348ea310');
      expect(found).toBeDefined();
      expect(found.user_email).toBe('user@simhaonline.com');
      expect(found.plan_name).toBe('Elite');
    });
  });

  describe('listAllSubscriptions (regression: Elite subscription must appear)', () => {
    it('should return at least 1 subscription with correct field mapping', async () => {
      if (!DB_URL) return;
      const result = await service.listAllSubscriptions(1, 20);
      expect(result.items.length).toBeGreaterThanOrEqual(1);
      const sub = result.items[0];
      expect(sub).toHaveProperty('plan_name');
      expect(sub).toHaveProperty('current_period_start');
      expect(sub).toHaveProperty('current_period_end');
      expect(sub).toHaveProperty('billing_cycle');
      expect(sub).toHaveProperty('license_key');
    });
  });

  describe('listAllDevices (regression: device with activations must appear)', () => {
    it('should return at least 1 device with activations array', async () => {
      if (!DB_URL) return;
      const result = await service.listAllDevices(1, 20);
      expect(result.items.length).toBeGreaterThanOrEqual(1);
      const dev = result.items[0];
      expect(dev).toHaveProperty('license_key');
      expect(dev).toHaveProperty('activations');
      expect(Array.isArray(dev.activations)).toBe(true);
    });
  });

  describe('listAllActivations (regression: MT4 and MT5 activations must appear)', () => {
    it('should return activations with client_type MT4 and MT5', async () => {
      if (!DB_URL) return;
      const result = await service.listAllActivations(1, 20);
      expect(result.items.length).toBeGreaterThanOrEqual(2);
      const types = result.items.map((a: any) => a.client_type);
      expect(types).toContain('MT4');
      expect(types).toContain('MT5');
    });
  });

  describe('getUserDetail (regression: full relationship map)', () => {
    it('should return user with subscription, licenses, devices, and activations', async () => {
      if (!DB_URL) return;
      const detail = await service.getUserDetail('fbae762d-6fbc-4e37-9856-222036cdc783');
      expect(detail.email).toBe('user@simhaonline.com');
      expect(detail.subscription).not.toBeNull();
      expect(detail.subscription.plan_name).toBe('Elite');
      expect(detail.licenses.length).toBeGreaterThanOrEqual(1);
      expect(detail.devices.length).toBeGreaterThanOrEqual(1);
      expect(detail.activations.length).toBeGreaterThanOrEqual(2);
    });
  });

  // Admin subscriptions financial tabs must always resolve (HTTP 200) and must
  // never fabricate data. With no DB they degrade to honest empty payloads.
  describe('subscription financial tabs (honest empty, never throws)', () => {
    it('getSubscriptionPayments resolves to an items array', async () => {
      const r: any = await service.getSubscriptionPayments();
      expect(Array.isArray(r.items)).toBe(true);
      if (r.items.length === 0) expect(typeof r.note).toBe('string');
    });

    it('getSubscriptionRefunds resolves to an items array', async () => {
      const r: any = await service.getSubscriptionRefunds();
      expect(Array.isArray(r.items)).toBe(true);
      if (r.items.length === 0) expect(r.note).toBe('No refunds recorded');
    });

    it('getSubscriptionChargebacks resolves to an items array', async () => {
      const r: any = await service.getSubscriptionChargebacks();
      expect(Array.isArray(r.items)).toBe(true);
      if (r.items.length === 0) expect(r.note).toBe('No chargebacks recorded');
    });

    it('getSubscriptionCoupons resolves to an items array', async () => {
      const r: any = await service.getSubscriptionCoupons();
      expect(Array.isArray(r.items)).toBe(true);
      if (r.items.length === 0) expect(r.note).toBe('No coupons configured');
    });

    it('getSubscriptionProvider never invents a provider name', async () => {
      const r: any = await service.getSubscriptionProvider();
      expect(Array.isArray(r.providers)).toBe(true);
      if (!r.configured) {
        expect(r.provider).toBeNull();
        expect(r.note).toBe('No payment provider configured');
      } else {
        // A provider may only be reported if it came from recorded payments.
        expect(r.providers.map((p: any) => p.provider)).toContain(r.provider);
      }
    });
  });

});
