import { Injectable, Inject, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

export interface BackupComponent {
  config_key: string;
  config_value: string | null;
  is_configured: boolean;
  required_for_prod: boolean;
  description: string | null;
}

export interface BackupDrData {
  status: string;
  configured: boolean;
  last_archived_time: string | null;
  note: string;
  components: BackupComponent[];
}

export interface ReleaseRow {
  id: string;
  component: string;
  version: string;
  channel: string;
  download_url: string;
  sha256: string;
  signature_key_id: string | null;
  mandatory: boolean;
  published_at: string | null;
  active: boolean;
}

export interface BrokerRow {
  broker: string;
  server: string;
  platform: string;
  broker_symbol: string;
  typical_spread: number | null;
  spread_p95: number | null;
  contract_size: number | null;
  qualification_result: string;
  last_validated_at: string | null;
  last_observed_at: string | null;
}

export interface MacroNewsRow {
  id: number;
  event_id: string;
  provider: string;
  event_name: string;
  country: string;
  currency: string;
  impact: string;
  scheduled_at_utc: string | null;
  actual: string | null;
  forecast: string | null;
  previous: string | null;
  received_at: string | null;
}

@Injectable()
export class AdminExtrasService {
  private readonly logger = new Logger(AdminExtrasService.name);

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /**
   * Honest backup/DR status. Reads `system.backup_configuration` if the table
   * exists and always attempts `pg_stat_archiver` for the last archived time.
   * Any failure degrades to an honest NOT_CONFIGURED status (never 500).
   */
  async getBackupDr(): Promise<BackupDrData> {
    const components: BackupComponent[] = [];
    let configured = false;
    let lastArchivedTime: string | null = null;

    // 1. system.backup_configuration (only if present)
    try {
      const exists = await this.pool.query(
        `SELECT 1 FROM information_schema.tables WHERE table_schema = 'system' AND table_name = 'backup_configuration'`,
      );
      if (exists.rows.length > 0) {
        const r = await this.pool.query(
          `SELECT config_key, config_value, is_configured, required_for_prod, description
           FROM system.backup_configuration
           ORDER BY required_for_prod DESC, config_key ASC`,
        );
        for (const row of r.rows) {
          components.push({
            config_key: row.config_key,
            config_value: row.config_value ?? null,
            is_configured: !!row.is_configured,
            required_for_prod: !!row.required_for_prod,
            description: row.description ?? null,
          });
        }
        configured = components.some((c) => c.is_configured);
      }
    } catch (err) {
      this.logger.warn(`backup_configuration read failed: ${err instanceof Error ? err.message : err}`);
    }

    // 2. WAL archive status (best-effort, never fatal)
    try {
      const r = await this.pool.query(`SELECT last_archived_time FROM pg_stat_archiver`);
      if (r.rows.length > 0) {
        lastArchivedTime = r.rows[0].last_archived_time ?? null;
      }
    } catch (err) {
      this.logger.warn(`pg_stat_archiver read failed: ${err instanceof Error ? err.message : err}`);
    }

    let status: string;
    let note: string;
    if (components.length > 0) {
      status = configured ? 'CONFIGURED' : 'CONFIGURED_NO_ARCHIVE';
      note = configured
        ? 'Backup configuration present.'
        : 'Backup configuration present but required keys are not configured.';
    } else {
      status = 'NOT_CONFIGURED';
      note = 'No backup configuration table present.';
    }

    return {
      status,
      configured,
      last_archived_time: lastArchivedTime,
      note,
      components,
    };
  }

  /** Returns release registry rows from `licensing.client_releases` or an honest empty note. */
  async getReleases(): Promise<{ items: ReleaseRow[]; note?: string }> {
    try {
      const exists = await this.pool.query(
        `SELECT 1 FROM information_schema.tables WHERE table_schema = 'licensing' AND table_name = 'client_releases'`,
      );
      if (exists.rows.length === 0) {
        return { items: [], note: 'No release registry configured' };
      }
      const r = await this.pool.query(
        `SELECT id, component, version, channel, download_url, sha256, signature_key_id,
                mandatory, published_at, active
         FROM licensing.client_releases
         ORDER BY published_at DESC NULLS LAST, created_at DESC NULLS LAST
         LIMIT 100`,
      );
      const items: ReleaseRow[] = r.rows.map((row) => ({
        id: String(row.id),
        component: row.component,
        version: row.version,
        channel: row.channel,
        download_url: row.download_url,
        sha256: row.sha256,
        signature_key_id: row.signature_key_id ?? null,
        mandatory: !!row.mandatory,
        published_at: row.published_at ? new Date(row.published_at).toISOString() : null,
        active: !!row.active,
      }));
      return { items };
    } catch (err) {
      this.logger.warn(`releases read failed: ${err instanceof Error ? err.message : err}`);
      return { items: [], note: 'No release registry configured' };
    }
  }

  /** Returns broker qualification rows from `market.broker_execution_profiles` or an honest empty note. */
  async getBrokerQualification(): Promise<{ items: BrokerRow[]; note?: string }> {
    try {
      const exists = await this.pool.query(
        `SELECT 1 FROM information_schema.tables WHERE table_schema = 'market' AND table_name = 'broker_execution_profiles'`,
      );
      if (exists.rows.length === 0) {
        return { items: [], note: 'No broker qualification runs recorded' };
      }
      const r = await this.pool.query(
        `SELECT broker, server, platform, broker_symbol, typical_spread, spread_p95,
                contract_size, qualification_result, last_validated_at, last_observed_at
         FROM market.broker_execution_profiles
         ORDER BY broker ASC, server ASC, platform ASC
         LIMIT 200`,
      );
      const items: BrokerRow[] = r.rows.map((row) => ({
        broker: row.broker,
        server: row.server,
        platform: row.platform,
        broker_symbol: row.broker_symbol,
        typical_spread: row.typical_spread ?? null,
        spread_p95: row.spread_p95 ?? null,
        contract_size: row.contract_size ?? null,
        qualification_result: row.qualification_result ?? 'PENDING',
        last_validated_at: row.last_validated_at ? new Date(row.last_validated_at).toISOString() : null,
        last_observed_at: row.last_observed_at ? new Date(row.last_observed_at).toISOString() : null,
      }));
      return { items };
    } catch (err) {
      this.logger.warn(`broker qualification read failed: ${err instanceof Error ? err.message : err}`);
      return { items: [], note: 'No broker qualification runs recorded' };
    }
  }

  /** Returns recent macro/news rows from `trading.economic_events` or an honest empty note. */
  async getMacroNews(): Promise<{ items: MacroNewsRow[]; note?: string }> {
    try {
      const exists = await this.pool.query(
        `SELECT 1 FROM information_schema.tables WHERE table_schema = 'trading' AND table_name = 'economic_events'`,
      );
      if (exists.rows.length === 0) {
        return { items: [], note: 'No macro/news data source configured' };
      }
      const r = await this.pool.query(
        `SELECT id, event_id, provider, event_name, country, currency, impact,
                scheduled_at_utc, actual, forecast, previous, received_at
         FROM trading.economic_events
         ORDER BY scheduled_at_utc DESC NULLS LAST
         LIMIT 200`,
      );
      const items: MacroNewsRow[] = r.rows.map((row) => ({
        id: Number(row.id),
        event_id: row.event_id,
        provider: row.provider,
        event_name: row.event_name,
        country: row.country ?? '',
        currency: row.currency ?? '',
        impact: row.impact ?? 'NONE',
        scheduled_at_utc: row.scheduled_at_utc ? new Date(row.scheduled_at_utc).toISOString() : null,
        actual: row.actual ?? null,
        forecast: row.forecast ?? null,
        previous: row.previous ?? null,
        received_at: row.received_at ? new Date(row.received_at).toISOString() : null,
      }));
      return { items };
    } catch (err) {
      this.logger.warn(`macro/news read failed: ${err instanceof Error ? err.message : err}`);
      return { items: [], note: 'No macro/news data source configured' };
    }
  }
}
