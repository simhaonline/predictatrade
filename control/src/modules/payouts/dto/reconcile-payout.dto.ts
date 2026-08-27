import { IsOptional, IsString } from 'class-validator';
import { Transform } from 'class-transformer';

export class ReconcilePayoutDto {
  @IsOptional() @IsString() provider_reference?: string;
  /**
   * Exact-decimal strings (a JSON number is still accepted for backward
   * compatibility and coerced to a string before validation). The service keeps
   * these as decimal.js through to the DB write.
   */
  @IsOptional()
  @Transform(({ value }) => (value === undefined || value === null ? value : String(value)))
  @IsString()
  net_amount?: string;

  @IsOptional()
  @Transform(({ value }) => (value === undefined || value === null ? value : String(value)))
  @IsString()
  fee_amount?: string;
}
