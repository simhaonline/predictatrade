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

  /** check.md 2026-08-30 #7 — admin user management (Approve/Suspend/Pending/Delete) */
  async setStatus(id: string, status: 'ACTIVE' | 'PENDING' | 'SUSPENDED' | 'DELETED', reason: string) {
    if (status === 'DELETED') return this.deleteUser(id);
    const r = await this.pool.query(
      `UPDATE iam.users SET status = $2, updated_at = now(), locked_until = NULL, failed_login_count = 0
       WHERE id = $1 RETURNING id, email, status`,
      [id, status],
    );
    if (r.rows.length === 0) throw new Error('user_not_found');
    await this.pool.query(
      `INSERT INTO audit.audit_events (actor_type, action, entity_type, entity_id, reason, new_value)
       VALUES ('system', 'iam.user.status_changed', 'user', $1, $2, $3::jsonb)`,
      [id, reason, JSON.stringify({ status })],
    );
    return r.rows[0];
  }

  async setRole(id: string, role: string) {
    // Role lives in iam.memberships ⋈ iam.roles. Create the membership if the
    // user has none (a user can be pre-approval / self-registered).
    const r = await this.pool.query(
      `UPDATE iam.memberships SET role_id = (SELECT id FROM iam.roles WHERE name = $2)
       WHERE user_id = $1 RETURNING role_id`,
      [id, role],
    );
    if (r.rowCount === 0) {
      // Insert the default org membership
      const defOrg = await this.pool.query(`SELECT id FROM iam.organizations ORDER BY created_at LIMIT 1`);
      if (defOrg.rowCount > 0) {
        const up = await this.pool.query(
          `INSERT INTO iam.memberships (id, user_id, organization_id, role_id, created_at)
           VALUES (gen_random_uuid(), $1, $2, (SELECT id FROM iam.roles WHERE name = $3), now())
           RETURNING role_id`,
          [id, defOrg.rows[0].id, role],
        );
        if (up.rowCount === 0) throw new Error('membership_create_failed');
        await this.pool.query(
          `INSERT INTO audit.audit_events (actor_type, action, entity_type, entity_id, reason, new_value)
           VALUES ('system', 'iam.user.role_changed', 'user', $1, 'admin set role (created membership)', $2::jsonb)`,
          [id, JSON.stringify({ role })],
        );
        return { id, role };
      }
      throw new Error('membership_not_found');
    }
    await this.pool.query(
      `INSERT INTO audit.audit_events (actor_type, action, entity_type, entity_id, reason, new_value)
       VALUES ('system', 'iam.user.role_changed', 'user', $1, 'admin set role', $2::jsonb)`,
      [id, JSON.stringify({ role })],
    );
    return { id, role };
  }

  async editUser(id: string, displayName?: string, email?: string) {
    const r = await this.pool.query(
      `UPDATE iam.users SET
         full_name = COALESCE($2, full_name),
         email = COALESCE($3, email),
         updated_at = now()
       WHERE id = $1 RETURNING id, email, full_name`,
      [id, displayName || null, email || null],
    );
    if (r.rows.length === 0) throw new Error('user_not_found');
    return r.rows[0];
  }

  async deleteUser(id: string) {
    // GDPR-friendly: anonymize + soft-delete, do not hard delete (financial FK)
    const r = await this.pool.query(
      `UPDATE iam.users SET
         status = 'DELETED',
         email = 'deleted_' || id::text || '@anonymized.local',
         password_hash = '',
         deleted_at = now(),
         updated_at = now()
       WHERE id = $1 RETURNING id`,
      [id],
    );
    if (r.rows.length === 0) throw new Error('user_not_found');
    await this.pool.query(
      `INSERT INTO audit.audit_events (actor_type, action, entity_type, entity_id, reason)
       VALUES ('system', 'iam.user.deleted', 'user', $1, 'admin deleted user')`,
      [id],
    );
    return { deleted: true, id };
  }
}
