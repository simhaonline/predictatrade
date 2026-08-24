import { Injectable, Inject, NotFoundException } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import * as ExcelJS from 'exceljs';
import PDFDocument from 'pdfkit';

export type ReportFormat = 'csv' | 'xlsx' | 'pdf';

export interface ReportFile {
  filename: string;
  contentType: string;
  buffer: Buffer;
  rowCount: number;
}

interface TradeRow {
  signal_id: string | null;
  account_id: string;
  strategy_id: string;
  symbol: string;
  direction: string;
  entry_price: string | null;
  exit_price: string | null;
  lot_size: string | null;
  pnl: string;
  close_reason: string | null;
  is_win: boolean;
  is_loss: boolean;
  opened_at: Date | null;
  closed_at: Date;
}

const MAX_TRADES = 5000;

/** RFC4180 field escaping: quote fields containing commas, quotes or newlines. */
function csvField(value: unknown): string {
  const s = value === null || value === undefined ? '' : String(value);
  if (/[",\r\n]/.test(s)) {
    return '"' + s.replace(/"/g, '""') + '"';
  }
  return s;
}

/** DECIMAL columns arrive as strings; format to 2 decimals for display only. */
function fmtMoney(value: string | number | null | undefined): string {
  if (value === null || value === undefined || value === '') return '';
  const n = typeof value === 'number' ? value : parseFloat(value);
  return Number.isFinite(n) ? n.toFixed(2) : '';
}

function fmtPrice(value: string | null): string {
  if (!value) return '';
  const n = parseFloat(value);
  return Number.isFinite(n) ? n.toFixed(2) : '';
}

@Injectable()
export class ReportsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /** Resolve agent ids bound to a user. */
  async getAgentIds(userId: string): Promise<string[]> {
    const r = await this.pool.query(
      `SELECT agent_id FROM trading.agent_user_bindings WHERE user_id = $1 ORDER BY bound_at ASC`,
      [userId],
    );
    return r.rows.map((row) => row.agent_id as string);
  }

  async getTrades(agentIds: string[]): Promise<TradeRow[]> {
    const accounts = agentIds.map((id) => `agent:${id}`);
    const r = await this.pool.query(
      `SELECT signal_id, account_id, strategy_id, symbol, direction,
              entry_price::text AS entry_price, exit_price::text AS exit_price,
              lot_size::text AS lot_size, pnl::text AS pnl,
              close_reason, is_win, is_loss, opened_at, closed_at
         FROM trading.trade_results
        WHERE account_id = ANY($1::varchar[])
        ORDER BY created_at DESC
        LIMIT $2`,
      [accounts, MAX_TRADES],
    );
    return r.rows as TradeRow[];
  }

  /**
   * Per-user aggregates for one subscriber. All money math happens in SQL over
   * NUMERIC columns; SUM(pnl)::text preserves exact decimals end-to-end.
   */
  async getUserSummary(userId: string) {
    const r = await this.pool.query(
      `SELECT count(*)::int AS total_trades,
              count(*) FILTER (WHERE tr.is_win)::int AS wins,
              count(*) FILTER (WHERE tr.is_loss)::int AS losses,
              CASE WHEN count(*) > 0
                   THEN round(100.0 * count(*) FILTER (WHERE tr.is_win) / count(*), 2)
                   ELSE 0 END AS win_rate_pct,
              COALESCE(SUM(tr.pnl), 0)::text AS total_pnl,
              min(tr.closed_at) AS first_trade_at,
              max(tr.closed_at) AS last_trade_at
         FROM trading.trade_results tr
        WHERE tr.account_id = ANY(
          SELECT 'agent:' || b.agent_id FROM trading.agent_user_bindings b WHERE b.user_id = $1)`,
      [userId],
    );
    const byStrategy = await this.pool.query(
      `SELECT tr.strategy_id,
              count(*)::int AS trades,
              count(*) FILTER (WHERE tr.is_win)::int AS wins,
              COALESCE(SUM(tr.pnl), 0)::text AS total_pnl
         FROM trading.trade_results tr
        WHERE tr.account_id = ANY(
          SELECT 'agent:' || b.agent_id FROM trading.agent_user_bindings b WHERE b.user_id = $1)
        GROUP BY tr.strategy_id ORDER BY count(*) DESC`,
      [userId],
    );
    const u = await this.pool.query(`SELECT email FROM iam.users WHERE id = $1`, [userId]);
    return {
      user_id: userId,
      email: u.rows[0]?.email ?? null,
      ...r.rows[0],
      by_strategy: byStrategy.rows,
    };
  }

  /** Aggregates for ALL users that have agent bindings. Money summed in SQL. */
  async getAllUsersSummary() {
    const r = await this.pool.query(
      `SELECT b.user_id, u.email,
              count(DISTINCT b.agent_id)::int AS agents,
              count(tr.id)::int AS total_trades,
              count(tr.id) FILTER (WHERE tr.is_win)::int AS wins,
              count(tr.id) FILTER (WHERE tr.is_loss)::int AS losses,
              CASE WHEN count(tr.id) > 0
                   THEN round(100.0 * count(tr.id) FILTER (WHERE tr.is_win) / count(tr.id), 2)
                   ELSE 0 END AS win_rate_pct,
              COALESCE(SUM(tr.pnl), 0)::text AS total_pnl,
              max(b.last_seen_at) AS last_agent_seen_at
         FROM trading.agent_user_bindings b
         JOIN iam.users u ON u.id = b.user_id
         LEFT JOIN trading.trade_results tr ON tr.account_id = 'agent:' || b.agent_id
        GROUP BY b.user_id, u.email
        ORDER BY COALESCE(SUM(tr.pnl), 0) DESC`,
    );
    return r.rows;
  }

  /** Fail-closed: no bindings or zero rows -> 404 no_trading_data. */
  private async loadUserData(userId: string): Promise<{ trades: TradeRow[]; email: string }> {
    const agentIds = await this.getAgentIds(userId);
    let trades: TradeRow[] = [];
    if (agentIds.length > 0) {
      trades = await this.getTrades(agentIds);
    }
    if (trades.length === 0) {
      throw new NotFoundException({ error: 'no_trading_data' });
    }
    const emailRow = await this.pool.query(`SELECT email FROM iam.users WHERE id = $1`, [userId]);
    const email = emailRow.rows[0]?.email ?? userId;
    return { trades, email };
  }

  async generateReport(userId: string, format: ReportFormat): Promise<ReportFile> {
    const { trades, email } = await this.loadUserData(userId);
    const dateStamp = new Date().toISOString().slice(0, 10);
    const safeEmail = email.replace(/[^a-zA-Z0-9._@-]/g, '_');
    switch (format) {
      case 'csv':
        return {
          filename: `trading_report_${safeEmail}_${dateStamp}.csv`,
          contentType: 'text/csv; charset=utf-8',
          buffer: Buffer.from(this.buildCsv(trades), 'utf8'),
          rowCount: trades.length,
        };
      case 'xlsx':
        return {
          filename: `trading_report_${safeEmail}_${dateStamp}.xlsx`,
          contentType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
          buffer: await this.buildXlsx(trades),
          rowCount: trades.length,
        };
      case 'pdf':
        return {
          filename: `trading_report_${safeEmail}_${dateStamp}.pdf`,
          contentType: 'application/pdf',
          buffer: await this.buildPdf(trades, email),
          rowCount: trades.length,
        };
    }
  }

  private headerRow(): string[] {
    return [
      'closed_at',
      'signal_id',
      'strategy',
      'symbol',
      'direction',
      'entry_price',
      'exit_price',
      'lot_size',
      'pnl',
      'result',
      'exit_reason',
    ];
  }

  private dataRows(trades: TradeRow[]): string[][] {
    return trades.map((t) => [
      new Date(t.closed_at).toISOString(),
      t.signal_id ?? '',
      t.strategy_id,
      t.symbol,
      t.direction,
      fmtPrice(t.entry_price),
      fmtPrice(t.exit_price),
      t.lot_size ?? '',
      fmtMoney(t.pnl),
      t.is_win ? 'WIN' : t.is_loss ? 'LOSS' : 'BREAKEVEN/OTHER',
      t.close_reason ?? '',
    ]);
  }

  /** RFC4180 CSV: CRLF line endings, quoted fields — opens natively in Excel. */
  private buildCsv(trades: TradeRow[]): string {
    const lines = [this.headerRow(), ...this.dataRows(trades)];
    return lines.map((row) => row.map(csvField).join(',')).join('\r\n') + '\r\n';
  }

  private async buildXlsx(trades: TradeRow[]): Promise<Buffer> {
    const wb = new ExcelJS.Workbook();
    wb.creator = 'Predict-A-Trade';
    wb.created = new Date();
    const ws = wb.addWorksheet('Trading Report');
    ws.addRow(this.headerRow());
    ws.getRow(1).font = { bold: true };
    ws.columns = [
      { width: 24 }, // closed_at
      { width: 38 }, // signal_id
      { width: 18 }, // strategy
      { width: 10 }, // symbol
      { width: 10 }, // direction
      { width: 12 }, // entry
      { width: 12 }, // exit
      { width: 10 }, // lot
      { width: 14, numFmt: '#,##0.00' }, // pnl
      { width: 16 }, // result
      { width: 18 }, // exit reason
    ];
    for (const row of this.dataRows(trades)) ws.addRow(row);
    return Buffer.from(await wb.xlsx.writeBuffer());
  }

  private buildPdf(trades: TradeRow[], email: string): Promise<Buffer> {
    const wins = trades.filter((t) => t.is_win).length;
    const losses = trades.filter((t) => t.is_loss).length;
    // NOTE: display-only float aggregation for the rendered summary block.
    // Exact decimal sums are computed in SQL (getUserSummary/getAllUsersSummary);
    // these JS floats are never persisted nor used for financial decisions.
    const totalPnl = trades.reduce((acc, t) => acc + parseFloat(t.pnl), 0);

    const doc = new PDFDocument({
      size: 'A4',
      margin: 36,
      info: { Title: `Trading Report — ${email}`, Author: 'Predict-A-Trade' },
    });
    const chunks: Buffer[] = [];
    doc.on('data', (c: Buffer) => chunks.push(c));
    const done = new Promise<Buffer>((resolve) => doc.on('end', () => resolve(Buffer.concat(chunks))));

    const left = doc.page.margins.left;
    const tableWidth = doc.page.width - left - doc.page.margins.right;

    // Report header
    doc.fontSize(16).font('Helvetica-Bold').text('Predict-A-Trade — Trading Report');
    doc.moveDown(0.3);
    doc.fontSize(9).font('Helvetica').fillColor('#444444');
    doc.text(`User: ${email}`);
    doc.text(
      `Period: ${new Date(trades[trades.length - 1].closed_at).toISOString().slice(0, 10)} to ` +
        `${new Date(trades[0].closed_at).toISOString().slice(0, 10)} (UTC)`,
    );
    doc.text(`Generated at: ${new Date().toISOString()} (UTC)`);
    doc.text(`Trades listed: ${trades.length} (most recent first, limit ${MAX_TRADES})`);
    doc.fillColor('#000000');

    // Summary block
    doc.moveDown();
    doc.fontSize(11).font('Helvetica-Bold').text('Summary');
    doc.fontSize(9).font('Helvetica');
    const winRate = ((wins / trades.length) * 100).toFixed(2);
    doc.text(`Total trades: ${trades.length}    Wins: ${wins}    Losses: ${losses}    Win rate: ${winRate}%`);
    doc.text(`Total P&L: ${fmtMoney(totalPnl)} USD`);

    // Per-strategy subtotals (same render-only float caveat as above).
    const byStrategy = new Map<string, { n: number; wins: number; pnl: number }>();
    for (const t of trades) {
      const s = byStrategy.get(t.strategy_id) ?? { n: 0, wins: 0, pnl: 0 };
      s.n += 1;
      s.wins += t.is_win ? 1 : 0;
      s.pnl += parseFloat(t.pnl);
      byStrategy.set(t.strategy_id, s);
    }
    for (const [strategy, s] of byStrategy) {
      doc.text(`  • ${strategy}: ${s.n} trades, ${((s.wins / s.n) * 100).toFixed(1)}% win, P&L ${fmtMoney(s.pnl)}`);
    }

    // Paginated trade table: date, strategy, direction, entry, exit, lot, pnl, reason
    const cols = [
      { label: 'Date', x: 0, w: 88 },
      { label: 'Strategy', x: 88, w: 76 },
      { label: 'Dir', x: 164, w: 30 },
      { label: 'Entry', x: 194, w: 54 },
      { label: 'Exit', x: 248, w: 54 },
      { label: 'Lot', x: 302, w: 34 },
      { label: 'P&L', x: 336, w: 56 },
      { label: 'Exit Reason', x: 392, w: tableWidth - 392 },
    ];

    const drawTableHeader = () => {
      const y = doc.y;
      doc.rect(left, y, tableWidth, 16).fill('#e8e8e8').fillColor('#000000');
      doc.fontSize(8).font('Helvetica-Bold');
      for (const c of cols) {
        doc.text(c.label, left + c.x + 2, y + 4, { width: c.w - 4, lineBreak: false, ellipsis: true });
      }
      doc.y = y + 16;
      doc.font('Helvetica');
    };

    doc.moveDown();
    drawTableHeader();

    const rowH = 13;
    doc.fontSize(8);
    for (const t of trades) {
      // Page break handling: start a fresh page and repeat the header row.
      if (doc.y + rowH + 2 > doc.page.height - doc.page.margins.bottom) {
        doc.addPage();
        drawTableHeader();
        doc.fontSize(8);
      }
      const y = doc.y;
      const pnlNum = parseFloat(t.pnl);
      const values = [
        new Date(t.closed_at).toISOString().slice(0, 16).replace('T', ' '),
        t.strategy_id,
        t.direction || '—',
        fmtPrice(t.entry_price),
        fmtPrice(t.exit_price),
        t.lot_size ?? '',
        fmtMoney(t.pnl),
        t.close_reason ?? '',
      ];
      doc.rect(left, y - 1, tableWidth, rowH).stroke('#dddddd');
      cols.forEach((c, i) => {
        doc.fillColor(i === 6 && Number.isFinite(pnlNum) && pnlNum !== 0 ? (pnlNum > 0 ? '#0a7d32' : '#b3261e') : '#000000');
        doc.text(values[i], left + c.x + 2, y + 2, { width: c.w - 4, lineBreak: false, ellipsis: true });
      });
      doc.y = y + rowH;
      doc.fillColor('#000000');
    }

    doc.end();
    return done;
  }

  async auditReportGeneration(actorId: string, subjectUserId: string, format: ReportFormat, rowCount: number) {
    try {
      await this.pool.query(
        `INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, entity_id, new_value, timestamp)
         VALUES (gen_random_uuid(), gen_random_uuid(), $1, $2, $3, 'user', $4, $5, now())`,
        [
          actorId === subjectUserId ? 'USER' : 'ADMIN',
          actorId,
          'TRADING_REPORT_GENERATED',
          subjectUserId,
          JSON.stringify({ format, subject_user_id: subjectUserId, row_count: rowCount }),
        ],
      );
    } catch {
      // Audit failure must not block report delivery to an authorized caller.
    }
  }
}
