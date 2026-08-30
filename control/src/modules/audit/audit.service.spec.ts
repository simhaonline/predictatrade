import { Test, TestingModule } from '@nestjs/testing';
import { jest } from '@jest/globals';
import { AuditService } from './audit.service';
import { DB_POOL } from '../../common/database.module';
import { Pool } from 'pg';

const DB_URL = process.env.DATABASE_URL || process.env.TEST_DATABASE_URL;

describe('AuditService', () => {
  let service: AuditService;
  let pool: any;

  beforeAll(async () => {
    if (DB_URL) {
      pool = new Pool({ connectionString: DB_URL, max: 5 });
    } else {
      // Query-aware mock: COUNT queries return a total row for pagination math.
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
        AuditService,
        { provide: DB_POOL, useValue: pool },
      ],
    }).compile();
    service = module.get<AuditService>(AuditService);
  });

  describe('list', () => {
    it('should return paginated audit events from audit.audit_events table', async () => {
      const result = await service.list(1, 20);
      expect(result).toHaveProperty('items');
      expect(result).toHaveProperty('total');
      expect(result).toHaveProperty('page', 1);
      expect(result).toHaveProperty('limit', 20);
    });

    it('should map audit_events columns to frontend-expected field names', async () => {
      if (!DB_URL) return;
      const result = await service.list(1, 5);
      if (result.items.length > 0) {
        const item = result.items[0];
        expect(item).toHaveProperty('event_type');
        expect(item).toHaveProperty('user_id');
        expect(item).toHaveProperty('created_at');
        expect(item).toHaveProperty('metadata');
      }
    });

    it('should handle empty audit table without error', async () => {
      const result = await service.list(1, 20);
      expect(result.total).toBeGreaterThanOrEqual(0);
    });
  });
});
