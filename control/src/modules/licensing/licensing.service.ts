import { Injectable, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class LicensingService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async listLicenses(userId: string) {
    const r = await this.pool.query(
      'SELECT * FROM licensing.licenses WHERE user_id = $1 ORDER BY created_at DESC', [userId],
    );
    return r.rows;
  }

  async listDevices(userId: string) {
    const r = await this.pool.query(
      'SELECT * FROM licensing.devices WHERE user_id = $1 ORDER BY created_at DESC', [userId],
    );
    return r.rows;
  }

  async registerDevice(userId: string, body: any) {
    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO licensing.devices (id, user_id, device_name, os, hardware_id, status, created_at)
       VALUES ($1, $2, $3, $4, $5, 'PENDING_ACTIVATION', now()) RETURNING *`,
      [id, userId, body.deviceName || 'Unknown', body.os || 'Windows', body.hardwareId || id],
    );
    return r.rows[0];
  }

  async listMtAccounts(userId: string) {
    const r = await this.pool.query(
      'SELECT * FROM licensing.mt_accounts WHERE user_id = $1 ORDER BY created_at DESC', [userId],
    );
    return r.rows;
  }

  async addMtAccount(userId: string, body: any) {
    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO licensing.mt_accounts (id, user_id, broker, platform, account_number, server, status, created_at)
       VALUES ($1, $2, $3, $4, $5, $6, 'PENDING_VERIFICATION', now()) RETURNING *`,
      [id, userId, body.broker, body.platform, body.accountNumber, body.server],
    );
    return r.rows[0];
  }
}
