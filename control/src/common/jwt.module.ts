import { Module, Global, Logger } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { JwtModule } from '@nestjs/jwt';

const DEV_JWT_SECRET = 'pat_local_dev_secret_change_in_production';

// P2-001: Known insecure placeholder secrets that must never be accepted in production.
const INSECURE_SECRETS = new Set<string>([
  '',
  DEV_JWT_SECRET,
  'CHANGE_ME_IN_PRODUCTION',
  'CHANGE_ME_IN_PRODUCTION_USE_SECRET_FILE',
  'change_this_to_a_long_random_secret',
  'changeme',
  'secret',
  'placeholder',
  'development',
]);

@Global()
@Module({
  imports: [
    JwtModule.registerAsync({
      imports: [ConfigModule],
      inject: [ConfigService],
      useFactory: (config: ConfigService) => {
        const logger = new Logger('GlobalJwtModule');
        const secret = config.get<string>('JWT_SECRET', DEV_JWT_SECRET);
        const isProduction = config.get<string>('NODE_ENV') === 'production';

        if (isProduction) {
          // P2-001: Reject empty, placeholder, or weak JWT secrets in production.
          if (!secret || INSECURE_SECRETS.has(secret)) {
            const msg =
              'FATAL: JWT_SECRET is missing, empty, or uses a known insecure placeholder. ' +
              'Refusing to start in production. Supply a strong secret (min 32 chars) via the production secret mechanism.';
            logger.error(msg);
            throw new Error(msg);
          }
          if (secret.length < 32) {
            const msg =
              'FATAL: JWT_SECRET is too short for production (min 32 characters required). ' +
              'Refusing to start with a weak secret.';
            logger.error(msg);
            throw new Error(msg);
          }
        } else if (secret === DEV_JWT_SECRET || INSECURE_SECRETS.has(secret)) {
          logger.warn(
            'Using insecure default JWT_SECRET for development. Set JWT_SECRET in production.',
          );
        }

        return {
          secret,
          signOptions: { expiresIn: '15m' },
        };
      },
    }),
  ],
  exports: [JwtModule],
})
export class GlobalJwtModule {}
