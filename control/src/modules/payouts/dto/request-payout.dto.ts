import { IsString, IsIn, IsOptional, IsObject, Matches } from 'class-validator';
import { Transform } from 'class-transformer';

export class RequestPayoutDto {
  /**
   * Payout amount as an exact-decimal string. Kept as `string` so the value
   * travels through the control plane as decimal.js (never a JS float) and is
   * only coerced to integer minor-units at the external gateway boundary.
   * A JSON number is still accepted for backward compatibility and is coerced
   * to a string before validation.
   */
  @Transform(({ value }) => (value === undefined || value === null ? value : String(value)))
  @IsString()
  @Matches(/^\d{1,18}(\.\d{1,8})?$/, { message: 'amount must be a positive decimal with ≤8 fractional digits' })
  amount: string;

  // Only BANK_TRANSFER and USDT are supported payout rails.
  @IsIn(['BANK_TRANSFER', 'USDT']) method: 'BANK_TRANSFER' | 'USDT';
  @IsString() destination: string;
  /** Method-specific payout details (bank fields / USDT network). */
  @IsOptional() @IsObject() details?: Record<string, string>;
}
