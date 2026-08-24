import { IsNumber, IsString, IsObject, IsOptional, IsIn, Min } from 'class-validator';

export class RequestPayoutDto {
  @IsNumber() @Min(50) amount: number;
  // Only BANK_TRANSFER and USDT are supported payout rails.
  @IsIn(['BANK_TRANSFER', 'USDT']) method: 'BANK_TRANSFER' | 'USDT';
  @IsString() destination: string;
  /** Method-specific payout details (bank fields / USDT network). */
  @IsOptional() @IsObject() details?: Record<string, string>;
}
