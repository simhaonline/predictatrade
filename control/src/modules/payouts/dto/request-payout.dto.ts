import { IsNumber, IsString, Min } from 'class-validator';

export class RequestPayoutDto {
  @IsNumber() @Min(10) amount: number;
  @IsString() method: string; // BANK_TRANSFER, PAYPAL, etc.
  @IsString() destination: string;
}
