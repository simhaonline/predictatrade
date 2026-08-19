import { Injectable, NotFoundException, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class UsersService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async findById(id: string) {
    const r = await this.pool.query(
      'SELECT id, email, full_name, status, created_at FROM iam.users WHERE id = $1', [id],
    );
    if (r.rows.length === 0) throw new NotFoundException('User not found');
    return r.rows[0];
  }

  async update(id: string, dto: any) {
    const r = await this.pool.query(
      'UPDATE iam.users SET full_name = $1, updated_at = now() WHERE id = $2 RETURNING id, email, full_name, status',
      [dto.displayName, id],
    );
    return r.rows[0];
  }

  async list(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const r = await this.pool.query(
      'SELECT id, email, full_name, status, created_at FROM iam.users ORDER BY created_at DESC LIMIT $1 OFFSET $2',
      [limit, offset],
    );
    return r.rows;
  }
}
