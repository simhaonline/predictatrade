import { IsString, IsOptional } from 'class-validator';

export class RejectPayoutDto {
  @IsString() reason: string;
}

export class CancelPayoutDto {
  @IsString() reason: string;
}
