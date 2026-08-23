import { Module } from '@nestjs/common';
import { MarketProxyController } from './market-proxy.controller';

@Module({
  controllers: [MarketProxyController],
})
export class MarketProxyModule {}
