import { Module } from '@nestjs/common';
import { WhatsAppGatewayController } from './whatsapp-gateway.controller';
import { WhatsAppAssistantEngine } from './whatsapp-assistant.service';
import { WhatsAppSender } from './whatsapp-sender.service';

/**
 * WhatsAppModule — the PAT WhatsApp AI assistant.
 *
 * Inbound: POST /api/v1/whatsapp/inbound (signature-verified, provider-agnostic
 * envelope) → assistant engine (server-authoritative data via REALTIME_URL)
 * → outbound sender (Meta Cloud API shape by default).
 */
@Module({
  controllers: [WhatsAppGatewayController],
  providers: [WhatsAppAssistantEngine, WhatsAppSender],
})
export class WhatsAppModule {}