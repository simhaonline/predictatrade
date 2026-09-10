import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { TelegramAssistantEngine, type TelegramReply } from './telegram-assistant.service';

/**
 * TelegramBot — long-polling bot driving the PAT Gold Desk assistant.
 *
 * Uses the shared TELEGRAM_BOT_TOKEN (the realtime engine only SENDS via the
 * same token — it never calls getUpdates, so no polling conflict).
 * Inline keyboards ride on every reply; callback_query updates route to the
 * assistant engine; dedup by update_id.
 */
@Injectable()
export class TelegramBot implements OnModuleInit {
  private readonly logger = new Logger(TelegramBot.name);
  private readonly token: string;
  private readonly adminChatIds: string[];
  private readonly api: string;
  private lastUpdateId = 0;
  private running = false;

  constructor(
    private config: ConfigService,
    private readonly engine: TelegramAssistantEngine,
  ) {
    this.token = this.config.get<string>('TELEGRAM_BOT_TOKEN') || '';
    this.api = `https://api.telegram.org/bot${this.token}`;
    const ids = this.config.get<string>('TELEGRAM_ASSISTANT_CHAT_IDS') || '';
    this.adminChatIds = ids.split(',').map((s) => s.trim()).filter(Boolean);
  }

  get configured(): boolean {
    return Boolean(this.token);
  }

  /** Enabled when TELEGRAM_ASSISTANT_ENABLED=true (operator opt-in). */
  get enabled(): boolean {
    return this.config.get<string>('TELEGRAM_ASSISTANT_ENABLED') === 'true' && this.configured;
  }

  onModuleInit() {
    if (!this.enabled) {
      this.logger.log('[TG] assistant disabled (TELEGRAM_ASSISTANT_ENABLED!=true or no token)');
      return;
    }
    this.running = true;
    this.logger.log('[TG] assistant poller starting');
    void this.pollLoop();
  }

  private sleep(ms: number) { return new Promise((r) => setTimeout(r, ms)); }

  private async pollLoop(): Promise<void> {
    while (this.running) {
      try {
        const res = await fetch(
          `${this.api}/getUpdates?timeout=25&offset=${this.lastUpdateId + 1}&allowed_updates=${encodeURIComponent(JSON.stringify(['message', 'callback_query']))}`,
        );
        if (!res.ok) {
          await this.sleep(5000);
          continue;
        }
        const data = (await res.json()) as {
          result?: Array<{
            update_id: number;
            message?: { chat?: { id?: number }; from?: { first_name?: string }; text?: string };
            callback_query?: { id?: string; from?: { first_name?: string }; data?: string; message?: { chat?: { id?: number } } };
          }>;
        };
        for (const upd of data.result ?? []) {
          this.lastUpdateId = Math.max(this.lastUpdateId, upd.update_id);
          try {
            if (upd.message?.text && upd.message.chat?.id) {
              const reply = await this.engine.handleMessage(upd.message.text, upd.message.from?.first_name);
              await this.send(upd.message.chat.id, reply);
            } else if (upd.callback_query?.data && upd.callback_query.message?.chat?.id) {
              const reply = await this.engine.handleCallback(upd.callback_query.data);
              await this.send(upd.callback_query.message.chat.id, reply);
              // answer the callback to clear the spinner
              await fetch(`${this.api}/answerCallbackQuery`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ callback_query_id: upd.callback_query.id }),
              });
            }
          } catch (e) {
            this.logger.error(`[TG] update handling failed: ${e instanceof Error ? e.message : e}`);
          }
        }
        await this.sleep(300);
      } catch (e) {
        this.logger.warn(`[TG] poll error: ${e instanceof Error ? e.message : e}`);
        await this.sleep(5000);
      }
    }
  }

  /** Send markdown text + inline keyboard (custom keyboard). */
  async send(chatId: number | string, reply: TelegramReply): Promise<void> {
    try {
      await fetch(`${this.api}/sendMessage`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chat_id: chatId,
          text: reply.text,
          parse_mode: 'Markdown',
          disable_web_page_preview: true,
          reply_markup: { inline_keyboard: reply.keyboard },
        }),
      });
    } catch (e) {
      this.logger.error(`[TG] send failed: ${e instanceof Error ? e.message : e}`);
    }
  }

  /** Broadcast an alert to admin chats (from the notifications pipeline). */
  async broadcast(text: string): Promise<void> {
    for (const id of this.adminChatIds) {
      await this.send(id, { text, keyboard: [] });
    }
  }
}