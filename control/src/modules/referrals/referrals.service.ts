import { Injectable, BadRequestException, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class ReferralsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  buildSponsorChain(userId: string, relationships: Map<string, string>): string[] {
    const chain: string[] = [];
    let current = userId;
    const visited = new Set<string>([userId]);
    for (let level = 1; level <= 5; level++) {
      const parent = relationships.get(current);
      if (!parent) break;
      if (visited.has(parent)) throw new BadRequestException('Circular referral detected');
      visited.add(parent); chain.push(parent); current = parent;
    }
    return chain;
  }

  async getReferralNetwork(userId: string) {
    const direct = await this.pool.query(
      `SELECT r.child_user_id, u.email, u.full_name, r.level, r.created_at
       FROM referral.referral_relationships r JOIN iam.users u ON r.child_user_id = u.id
       WHERE r.parent_user_id = $1 ORDER BY r.level, r.created_at`, [userId],
    );
    return { referrals: direct.rows, count: direct.rows.length };
  }

  async getCommissions(userId: string) {
    const r = await this.pool.query(
      `SELECT c.*, u.email as source_email FROM referral.commission_ledger c
       LEFT JOIN iam.users u ON c.source_user_id = u.id
       WHERE c.recipient_user_id = $1 ORDER BY c.created_at DESC LIMIT 50`, [userId],
    );
    return r.rows;
  }
}
