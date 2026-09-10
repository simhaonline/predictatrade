import { Module } from '@nestjs/common';
import { TelegramAdminController } from './telegram-admin.controller';
import { TelegramBot } from './telegram-bot.service';
import { TelegramAssistantEngine } from './telegram-assistant.service';

/**
 * TelegramModule — the PAT Telegram AI (XAU/USD Gold Desk).
 *
 * Long-polling bot + assistant engine. Operator opt-in via
 * TELEGRAM_ASSISTANT_ENABLED=true + shared TELEGRAM_BOT_TOKEN.
 */
@Module({
  controllers: [TelegramAdminController],
  providers: [TelegramBot, TelegramAssistantEngine],
})
export class TelegramModule {}