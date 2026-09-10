import { Controller, Get, UseGuards, Query, Logger } from '@nestjs/common';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard, Roles, Role } from '../../common/guards/roles.guard';
import { TelegramBot } from './telegram-bot.service';
import { TelegramAssistantEngine } from './telegram-assistant.service';

/**
 * TelegramAdminController — operator surface for the Telegram AI desk.
 *
 * GET /api/v1/admin/telegram/status     bot config/arm state + poller health
 * POST /api/v1/admin/telegram/broadcast push a markdown alert to admin chats
 *
 * The bot itself talks directly to Telegram via long polling; these endpoints
 * exist for monitoring and for the notifications pipeline to route alerts.
 */
@Controller('admin/telegram')
@UseGuards(JwtAuthGuard, RolesGuard)
@Roles(Role.ADMIN, Role.SUPER_ADMIN)
export class TelegramAdminController {
  private readonly logger = new Logger(TelegramAdminController.name);

  constructor(
    private readonly bot: TelegramBot,
    private readonly engine: TelegramAssistantEngine,
  ) {}

  @Get('status')
  status() {
    return {
      botConfigured: this.bot.configured,
      assistantEnabled: this.bot.enabled,
      tradingArmed: this.engine.isArmed,
      note: this.engine.isArmed
        ? 'ARMED — approved plans emit gated execution intents (engine 14-gate registry holds authority)'
        : 'ADVISORY — plans only; execution locked until operator arms with TELEGRAM_TRADING_ARMED=true',
    };
  }

  @Get('broadcast')
  async broadcast(@Query('text') text: string) {
    if (!text?.trim()) return { sent: 0, note: 'text query param required' };
    await this.bot.broadcast(text.trim());
    return { sent: 1 };
  }
}