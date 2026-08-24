import 'reflect-metadata';
import { Pool } from 'pg';
import * as fs from 'fs';
import { ReportsService } from './src/modules/reports/reports.service';

async function main() {
  const dbUrl = fs.readFileSync('/srv/predictatrade/xauusd/database_url.txt', 'utf-8').trim();
  const pool = new Pool({ connectionString: dbUrl });
  const svc = new ReportsService(pool);

  // Pick an existing user
  const u = await pool.query(`SELECT id, email FROM iam.users LIMIT 1`);
  if (u.rows.length === 0) {
    console.log('NO_USERS_IN_DB');
    await pool.end();
    return;
  }
  const userId = u.rows[0].id;
  console.log('test user:', u.rows[0].email);

  // 1) Fail-closed: user with no bindings
  try {
    await svc.generateReport(userId, 'csv');
    console.log('FAIL: expected 404 no_trading_data');
    process.exitCode = 1;
  } catch (e: any) {
    console.log('fail-closed ok:', e.status, JSON.stringify(e.response?.body ?? e.response));
  }

  // 2) Seed inside transaction, generate all formats, roll back
  const client = await pool.connect();
  try {
    await client.query('BEGIN');
    await client.query(
      `INSERT INTO trading.agent_user_bindings (agent_id, license_key, user_id)
       VALUES ('smoke-test-agent', 'SMOKE-LICENSE-KEY', $1) ON CONFLICT (agent_id) DO UPDATE SET user_id = EXCLUDED.user_id`,
      [userId],
    );
    for (let i = 0; i < 60; i++) {
      const pnl = (i % 3 === 0 ? -1 : 1) * (12.5 + i);
      await client.query(
        `INSERT INTO trading.trade_results
           (account_id, strategy_id, symbol, direction, entry_price, exit_price, lot_size, pnl, close_reason, is_win, is_loss, trading_day, closed_at)
         VALUES ('agent:smoke-test-agent', $1, 'XAUUSD', $2, 2350.25, 2352.75, 0.10, $3, $4, $5, $6, (now() - interval '2 hours')::date, now() - ($7 || ' hours')::interval)`,
        [
          i % 2 ? 'STANDARD_SCALPING' : 'ULTRA_SCALPING',
          i % 2 ? 'BUY' : 'SELL',
          pnl,
          i % 3 === 0 ? 'sl' : 'tp',
          pnl > 0,
          pnl < 0,
          i,
        ],
      );
    }
    // Temporarily swap pool so service sees uncommitted data on same connection:
    // simplest is to commit-free read via same client — instead just commit-less
    // generation won't see rows across connections. Use SAVEPOINT trick: commit.
    await client.query('COMMIT');
  } catch (err) {
    await client.query('ROLLBACK');
    throw err;
  } finally {
    client.release();
  }

  try {
    for (const fmt of ['csv', 'xlsx', 'pdf'] as const) {
      const file = await svc.generateReport(userId, fmt);
      fs.writeFileSync(`/tmp/opencode/smoke.${fmt}`, file.buffer);
      console.log(fmt, '->', file.filename, file.contentType, file.rowCount, 'rows,', file.buffer.length, 'bytes',
        'magic:', file.buffer.subarray(0, 4).toString('hex'));
    }
    const summary = await svc.getUserSummary(userId);
    console.log('userSummary:', summary.total_trades, 'trades, win_rate', summary.win_rate_pct, 'pnl', summary.total_pnl, 'email', summary.email);
    const all = await svc.getAllUsersSummary();
    console.log('allUsersSummary users:', all.length, 'first:', JSON.stringify(all[0]));
    await svc.auditReportGeneration(userId, userId, 'csv', 60);
    const audit = await pool.query(
      `SELECT action, actor_type, entity_type, new_value FROM audit.audit_events WHERE action='TRADING_REPORT_GENERATED' ORDER BY timestamp DESC LIMIT 1`,
    );
    console.log('audit row:', JSON.stringify(audit.rows[0]));
  } finally {
    // Cleanup seeded rows (leave DB as found)
    const c2 = await pool.connect();
    try {
      await c2.query("DELETE FROM trading.trade_results WHERE account_id = 'agent:smoke-test-agent'");
      await c2.query("DELETE FROM trading.agent_user_bindings WHERE agent_id = 'smoke-test-agent'");

      console.log('cleanup done');
    } finally {
      c2.release();
      await pool.end();
    }
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
