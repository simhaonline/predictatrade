import { Module, Global, Logger } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { Pool } from 'pg';
import * as fs from 'fs';
import * as path from 'path';

export const DB_POOL = 'DB_POOL';

/**
 * Loads the DATABASE_URL from environment variable, falling back to
 * /srv/predictatrade/xauusd/database_url.txt (gitignored secret file).
 *
 * Priority:
 *   1. DATABASE_URL environment variable (set by systemd EnvironmentFile)
 *   2. /srv/predictatrade/xauusd/database_url.txt (secret file, chmod 600)
 *   3. Empty (app fails with clear message)
 */
function loadDatabaseUrl(config: ConfigService): string {
  // 1. Try environment variable first
  let dbUrl = config.get<string>('DATABASE_URL', '');

  // Strip comments (env files may have inline comments)
  if (dbUrl && dbUrl.includes('#')) {
    dbUrl = dbUrl.split('#')[0].trim();
  }

  // 2. Fall back to secret file if env var is empty or placeholder
  if (!dbUrl || dbUrl.startsWith('  #')) {
    const secretPath = '/srv/predictatrade/xauusd/database_url.txt';
    try {
      if (fs.existsSync(secretPath)) {
        dbUrl = fs.readFileSync(secretPath, 'utf-8').trim();
      }
    } catch {
      // ignore read errors — will be caught below
    }
  }

  return dbUrl;
}

@Global()
@Module({
  imports: [ConfigModule],
  providers: [
    {
      provide: DB_POOL,
      inject: [ConfigService],
      useFactory: (config: ConfigService) => {
        const logger = new Logger('DatabaseModule');
        const dbUrl = loadDatabaseUrl(config);
        const isProduction = config.get<string>('NODE_ENV') === 'production';

        if (!dbUrl) {
          const msg =
            'FATAL: DATABASE_URL is not set and no database_url.txt found. ' +
            'Refusing to start. Create /srv/predictatrade/xauusd/database_url.txt ' +
            'with the connection string, or set DATABASE_URL env var.';
          logger.error(msg);
          throw new Error(msg);
        }

        if (isProduction) {
          // Log a sanitized version (never the password)
          const safeUrl = dbUrl.replace(/:\/\/([^:]+):([^@]+)@/, '://$1:***@');
          logger.log(`Database URL: ${safeUrl}`);
        }

        return new Pool({
          connectionString: dbUrl,
          max: 20,
          idleTimeoutMillis: 30000,
          connectionTimeoutMillis: 5000,
        });
      },
    },
  ],
  exports: [DB_POOL],
})
export class DatabaseModule {}
