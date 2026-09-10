import { IsString, IsNotEmpty, MaxLength, IsOptional, IsObject } from 'class-validator';

/**
 * WhatsApp inbound webhook payload (provider-agnostic envelope).
 *
 * Providers differ in shape (Meta Cloud API, Twilio, 360dialog, WAHA). The
 * gateway controller normalizes each provider into THIS envelope before the
 * assistant engine sees it:
 *   from        — the sender phone (E.164)
 *   text        — the message body (may be empty for button clicks)
 *   button_id   — interactive button/list reply id (may be empty)
 *   message_id  — provider message id (for dedup)
 *   profile     — optional provider profile payload (name etc.)
 */
export class WhatsAppInboundDto {
  @IsString() @IsNotEmpty() @MaxLength(32)
  from!: string;

  @IsOptional() @IsString() @MaxLength(4096)
  text?: string;

  @IsOptional() @IsString() @MaxLength(255)
  button_id?: string;

  @IsOptional() @IsString() @MaxLength(255)
  message_id?: string;

  @IsOptional() @IsObject()
  profile?: Record<string, unknown>;
}