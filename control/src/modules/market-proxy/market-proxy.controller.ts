import {
  Controller,
  Get,
  HttpException,
  HttpStatus,
  Logger,
} from '@nestjs/common';

const REALTIME_BASE =
  process.env.REALTIME_URL && process.env.REALTIME_URL.trim().length > 0
    ? process.env.REALTIME_URL.replace(/\/$/, '')
    : 'http://realtime:13081';

const UPSTREAM_TIMEOUT_MS = 5000;

@Controller('market')
export class MarketProxyController {
  private readonly logger = new Logger(MarketProxyController.name);

  private async proxy(path: string): Promise<unknown> {
    const url = `${REALTIME_BASE}${path}`;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), UPSTREAM_TIMEOUT_MS);

    try {
      const res = await fetch(url, {
        signal: controller.signal,
        headers: { Accept: 'application/json' },
      });

      if (!res.ok) {
        throw new HttpException(
          { error: 'market_source_unavailable' },
          HttpStatus.BAD_GATEWAY,
        );
      }

      return await res.json();
    } catch (err) {
      if (err instanceof HttpException) {
        throw err;
      }
      this.logger.warn(`Market upstream unavailable: ${url} — ${err}`);
      throw new HttpException(
        { error: 'market_source_unavailable' },
        HttpStatus.BAD_GATEWAY,
      );
    } finally {
      clearTimeout(timer);
    }
  }

  @Get('snapshot')
  async snapshot(): Promise<unknown> {
    return this.proxy('/api/v1/market/snapshot');
  }

  @Get('state')
  async state(): Promise<unknown> {
    return this.proxy('/api/v1/market/state');
  }

  @Get('candles')
  async candles(): Promise<unknown> {
    return this.proxy('/api/v1/candles');
  }
}
