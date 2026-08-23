import { Injectable, Inject, NotFoundException, BadRequestException } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

const VALID_MODES = ['OFF', 'SHADOW', 'ACTIVE', 'DISABLED', 'UNSUPPORTED', 'RESEARCH'];

@Injectable()
export class FeatureFlagsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /** List all PTB feature flags ordered by module name. Returns [] if none. */
  async listFlags() {
    const r = await this.pool.query(
      `SELECT id, module_name, mode, set_by, set_at, reason, created_at, updated_at
       FROM trading.ptb_feature_flags
       ORDER BY module_name`,
    );
    return r.rows;
  }

  /**
   * Update a single feature flag. The table has no is_enabled column, so
   * toggling is performed via the `mode` field. An `is_enabled` boolean in the
   * body is mapped to ACTIVE/DISABLED for ergonomic frontends. Returns the row.
   */
  async updateFlag(
    id: string,
    body: { mode?: string; reason?: string; is_enabled?: boolean; set_by?: string },
  ) {
    const sets: string[] = [];
    const values: any[] = [];
    let idx = 1;

    let mode = body.mode;
    if (typeof body.is_enabled === 'boolean') {
      mode = body.is_enabled ? 'ACTIVE' : 'DISABLED';
    }
    if (mode !== undefined) {
      if (!VALID_MODES.includes(mode)) {
        throw new BadRequestException(`Invalid mode: ${mode}. Allowed: ${VALID_MODES.join(', ')}`);
      }
      sets.push(`mode = $${idx}`);
      values.push(mode);
      idx++;
      // Record when the mode was last changed.
      sets.push(`set_at = now()`);
    }

    if (typeof body.reason === 'string') {
      sets.push(`reason = $${idx}`);
      values.push(body.reason);
      idx++;
    }

    if (typeof body.set_by === 'string') {
      sets.push(`set_by = $${idx}`);
      values.push(body.set_by);
      idx++;
    }

    if (sets.length === 0) {
      const current = await this.pool.query(
        'SELECT * FROM trading.ptb_feature_flags WHERE id = $1',
        [id],
      );
      if (current.rows.length === 0) throw new NotFoundException('Feature flag not found');
      return current.rows[0];
    }

    sets.push(`updated_at = now()`);
    values.push(id);

    const r = await this.pool.query(
      `UPDATE trading.ptb_feature_flags
       SET ${sets.join(', ')}
       WHERE id = $${idx}
       RETURNING id, module_name, mode, set_by, set_at, reason, created_at, updated_at`,
      values,
    );
    if (r.rows.length === 0) throw new NotFoundException('Feature flag not found');
    return r.rows[0];
  }
}
