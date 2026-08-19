import { Injectable, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class HealthService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async check() {
    const checks: any = { service: 'control-plane', version: '1.0.0', timestamp: new Date().toISOString() };
    try {
      const r = await this.pool.query('SELECT 1 as ok');
      checks.database = r.rows[0].ok === 1 ? 'healthy' : 'unhealthy';
    } catch (e) {
      checks.database = 'unavailable';
    }
    checks.status = checks.database === 'healthy' ? 'ok' : 'degraded';
    return checks;
  }
}
