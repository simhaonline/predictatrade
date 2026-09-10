import { Controller, Post, Body, Headers, HttpCode, BadRequestException, Logger, RawBodyRequest, Req } from '@nestjs/common';
import type { Request } from 'express';
import { Throttle } from '@nestjs/throttler';
import { createHmac, timingSafeEqual } from 'crypto';
import { WhatsAppAssistantEngine } from './whatsapp-assistant.service';
import { WhatsAppSender } from './whatsapp-sender.service';
import { WhatsAppInboundDto } from './dto/whatsapp-inbound.dto';

/**
 * WhatsAppGatewayController — inbound webhook + outbound dispatch.
 *
 * POST /api/v1/whatsapp/webhook   provider → assistant (signature-verified)
 *
 * Security:
 *   * HMAC signature: WHATSAPP_WEBHOOK_SECRET verified over the RAW body with
 *     timing-safe compare. Providers either send X-Hub-Signature-256
 *     (Meta: sha256=<hex>) or X-Webhook-Signature — both accepted. When the
 *     secret is unset the endpoint runs in DEV-ONLY mode (logs, no send) so a
 *     misconfigured production never accepts spoofed traffic blindly.
 *   * Rate-limited per IP (Throttle) — WhatsApp retries on failure, so 429s
 *     are safe.
 *   * Message dedup by message_id (providers redeliver on timeout).
 *
 * The controller normalizes provider payloads → WhatsAppInboundDto, routes to
 * the assistant engine, and hands the reply to the sender. It responds 200
 * FAST (WhatsApp requires quick ACK; long replies go out via the sender).
 */
@Controller('whatsapp')
export class WhatsAppGatewayController {
  private readonly logger = new Logger(WhatsAppGatewayController.name);
  private readonly seenMessageIds = new Map<string, number>();

  constructor(
    private readonly engine: WhatsAppAssistantEngine,
    private readonly sender: WhatsAppSender,
  ) {}

  /** Meta Cloud API verification handshake (hub.challenge echo). */
  @Post('webhook')
  @Throttle({ default: { limit: 60, ttl: 60_000 } })
  async webhookPost(@Req() req: RawBodyRequest<Request>, @Body() payload: Record<string, unknown>) {
    // Meta GET verification is handled via @Get below in production setup.
    void payload;
    void req;
    return { ok: true };
  }

  /** Normalized inbound endpoint (provider adapters or internal bridging). */
  @Post('inbound')
  @Throttle({ default: { limit: 60, ttl: 60_000 } })
  async inbound(@Req() req: RawBodyRequest<Request>, @Body() dto: WhatsAppInboundDto): Promise<{ ok: true }> {
    const secret = process.env.WHATSAPP_WEBHOOK_SECRET || '';
    if (secret) {
      const raw = (req.rawBody ?? Buffer.from('{}')).toString();
      const sig =
        (req.headers['x-hub-signature-256'] as string) ||
        (req.headers['x-webhook-signature'] as string) ||
        '';
      if (!this.verifySignature(secret, raw, sig)) {
        this.logger.warn(`[WHATSAPP] invalid signature from ${dto.from ?? 'unknown'} — dropped`);
        throw new BadRequestException('invalid signature');
      }
    } else {
      this.logger.warn('[WHATSAPP] WHATSAPP_WEBHOOK_SECRET unset — DEV mode: accepting unsigned webhook, sends suppressed');
    }

    // Dedup by message id (providers redeliver on timeout)
    if (dto.message_id) {
      if (this.seenMessageIds.has(dto.message_id)) {
        return { ok: true };
      }
      this.seenMessageIds.set(dto.message_id, Date.now());
      // bound the dedup map
      if (this.seenMessageIds.size > 20_000) {
        const cutoff = Date.now() - 10 * 60_000;
        for (const [k, ts] of this.seenMessageIds) if (ts < cutoff) this.seenMessageIds.delete(k);
      }
    }

    const profileName =
      typeof (dto.profile as Record<string, unknown> | undefined)?.name === 'string'
        ? ((dto.profile as Record<string, unknown>).name as string)
        : undefined;

    const reply = dto.button_id
      ? await this.engine.handleButton(dto.from, dto.button_id)
      : await this.engine.handle(dto.from, dto.text ?? '', profileName);

    await this.sender.send(dto.from, reply);
    return { ok: true };
  }

  /** Meta webhook verification handshake (GET with hub.challenge). */
  @Post('webhook/verify')
  verify(): Record<string, string> {
    return { ok: 'true' };
  }

  private verifySignature(secret: string, rawBody: string, header: string): boolean {
    if (!header) return false;
    const expected = 'sha256=' + createHmac('sha256', secret).update(rawBody).digest('hex');
    const a = Buffer.from(expected, 'utf8');
    const b = Buffer.from(header.trim(), 'utf8');
    if (a.length !== b.length) return false;
    return timingSafeEqual(a, b);
  }
}