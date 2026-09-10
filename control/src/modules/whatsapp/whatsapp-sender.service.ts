import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import type { AssistantReply } from './whatsapp-assistant.service';

/**
 * WhatsAppSender — provider-agnostic outbound sender.
 *
 * Providers differ; the adapter speaks the Meta Cloud API message shape by
 * default (the industry norm) and can be swapped via WHATSAPP_PROVIDER.
 * Outbound failures NEVER throw to the caller path (delivery is best-effort,
 * logged) — mirroring the notifications.Manager philosophy.
 */
@Injectable()
export class WhatsAppSender {
  private readonly logger = new Logger(WhatsAppSender.name);
  private readonly apiUrl: string;
  private readonly token: string;
  private readonly phoneNumberId: string;

  constructor(private config: ConfigService) {
    this.apiUrl = (
      this.config.get<string>('WHATSAPP_API_URL') || 'https://graph.facebook.com/v21.0'
    ).replace(/\/$/, '');
    this.token = this.config.get<string>('WHATSAPP_TOKEN') || '';
    this.phoneNumberId = this.config.get<string>('WHATSAPP_PHONE_NUMBER_ID') || '';
  }

  get configured(): boolean {
    return Boolean(this.token && this.phoneNumberId);
  }

  /**
   * Send a text + interactive-buttons message.
   * Buttons are capped at 3 by Meta's interactive button spec — extra rows
   * collapse to a numbered list appended to the body text.
   */
  async send(to: string, reply: AssistantReply): Promise<void> {
    if (!this.configured) {
      this.logger.warn(`[WHATSAPP] not configured — would send to ${to}: ${reply.text.slice(0, 80)}…`);
      return;
    }
    try {
      const { text, buttons } = this.shapeReply(reply);
      const body: Record<string, unknown> = {
        messaging_product: 'whatsapp',
        to,
        type: 'text',
        text: { preview_url: false, body: text },
      };
      if (buttons.length > 0) {
        body.type = 'interactive';
        delete body.text;
        body.interactive = {
          type: 'button',
          body: { text },
          action: { buttons: buttons.map((b) => ({ type: 'reply', reply: { id: b.id, title: b.title } })) },
        };
      }
      const res = await fetch(`${this.apiUrl}/${this.phoneNumberId}/messages`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${this.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const errText = await res.text().catch(() => '');
        this.logger.error(`[WHATSAPP] send to ${to} failed (${res.status}): ${errText.slice(0, 200)}`);
        return;
      }
      this.logger.log(`[WHATSAPP] sent to ${to} (${buttons.length} buttons)`);
    } catch (e) {
      this.logger.error(`[WHATSAPP] send to ${to} error: ${e instanceof Error ? e.message : e}`);
    }
  }

  /** Meta allows max 3 buttons; collapse overflow into a numbered list. */
  private shapeReply(reply: AssistantReply): { text: string; buttons: { id: string; title: string }[] } {
    if (reply.buttons.length <= 3) return reply;
    const [a, b, c, ...rest] = reply.buttons;
    const list = rest.map((r, i) => `${i + 4}. ${r.title}`).join('\n');
    return {
      text: `${reply.text}\n\nReply with:\n${rest.map((r, i) => `${i + 4}. *${r.title.split(' ').slice(1).join(' ') || r.title}*`).join('\n')}`,
      buttons: [a, b, c].filter(Boolean),
    };
  }
}