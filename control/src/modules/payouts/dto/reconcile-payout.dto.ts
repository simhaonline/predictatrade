import { IsOptional, IsString, IsNumber } from 'class-validator';

export class ReconcilePayoutDto {
  @IsOptional() @IsString() provider_reference?: string;
  @IsOptional() @IsNumber() net_amount?: number;
  @IsOptional() @IsNumber() fee_amount?: number;
}
