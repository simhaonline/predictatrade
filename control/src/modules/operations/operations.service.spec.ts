import { Test, TestingModule } from '@nestjs/testing';
import { OperationsService } from './operations.service';
import { DB_POOL } from '../../common/database.module';

const DB_URL = process.env.DATABASE_URL || process.env.TEST_DATABASE_URL;

describe('OperationsService', () => {
  let service: OperationsService;
  let pool: any;

  beforeAll(async () => {
    if (DB_URL) {
      const { Pool } = require('pg');
      pool = new Pool({ connectionString: DB_URL, max: 5 });
    } else {
      // Mock pool for unit tests when no DB is available
      pool = {
        query: jest.fn().mockResolvedValue({ rows: [], rowCount: 0 }),
      };
    }
  });

  afterAll(async () => {
    if (pool?.end) await pool.end();
  });

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        OperationsService,
        { provide: DB_POOL, useValue: pool },
      ],
    }).compile();
    service = module.get<OperationsService>(OperationsService);
  });

  describe('getTradingState', () => {
    it('should return active_strategies array (not undefined)', async () => {
      const state = await service.getTradingState();
      expect(state).toBeDefined();
      expect(state.active_strategies).toBeDefined();
      expect(Array.isArray(state.active_strategies)).toBe(true);
    });

    it('should return all 4 canonical strategies when none are disabled', async () => {
      if (!DB_URL) return; // skip if no DB
      const state = await service.getTradingState();
      expect(state.active_strategies).toContain('STANDARD_SCALPING');
      expect(state.active_strategies).toContain('ULTRA_SCALPING');
      expect(state.active_strategies).toContain('STANDARD_SWING');
      expect(state.active_strategies).toContain('TREND_SWING');
    });

    it('should return last_updated field (not undefined)', async () => {
      const state = await service.getTradingState();
      expect(state).toHaveProperty('last_updated');
    });

    it('should return trading_halted and signals_paused as booleans', async () => {
      const state = await service.getTradingState();
      expect(typeof state.trading_halted).toBe('boolean');
      expect(typeof state.signals_paused).toBe('boolean');
    });

    it('should return disabled_strategies array', async () => {
      const state = await service.getTradingState();
      expect(Array.isArray(state.disabled_strategies)).toBe(true);
    });
  });

  describe('getActiveOperations', () => {
    it('should not return stale RESUME_TRADING operations as ACTIVE', async () => {
      if (!DB_URL) return; // skip if no DB
      const ops = await service.getActiveOperations();
      const staleResume = ops.filter(
        (op: any) => op.operation_type === 'RESUME_TRADING' && op.status === 'ACTIVE'
      );
      expect(staleResume).toHaveLength(0);
    });

    it('should not return stale RESUME_SIGNALS operations as ACTIVE', async () => {
      if (!DB_URL) return; // skip if no DB
      const ops = await service.getActiveOperations();
      const staleResume = ops.filter(
        (op: any) => op.operation_type === 'RESUME_SIGNALS' && op.status === 'ACTIVE'
      );
      expect(staleResume).toHaveLength(0);
    });

    it('should not return stale ENABLE_STRATEGY operations as ACTIVE', async () => {
      if (!DB_URL) return; // skip if no DB
      const ops = await service.getActiveOperations();
      const staleEnable = ops.filter(
        (op: any) => op.operation_type === 'ENABLE_STRATEGY' && op.status === 'ACTIVE'
      );
      expect(staleEnable).toHaveLength(0);
    });
  });

  describe('cleanupStaleOperations', () => {
    it('should execute without error and return a number', async () => {
      const count = await service.cleanupStaleOperations();
      expect(typeof count).toBe('number');
      expect(count).toBeGreaterThanOrEqual(0);
    });

    it('should leave no stale RESUME_*/ENABLE_* operations ACTIVE after cleanup', async () => {
      if (!DB_URL) return; // skip if no DB
      await service.cleanupStaleOperations();
      const ops = await service.getActiveOperations();
      const stale = ops.filter(
        (op: any) =>
          ['RESUME_TRADING', 'RESUME_SIGNALS', 'ENABLE_STRATEGY'].includes(op.operation_type) &&
          op.status === 'ACTIVE'
      );
      expect(stale).toHaveLength(0);
    });
  });
});
