import { IsArray, IsIn, IsOptional, IsString, IsUUID } from 'class-validator';

export class CreateSubscriptionDto {
  @IsUUID() planId: string;
  @IsOptional() @IsString() strategyIds?: string;
  @IsOptional() @IsArray() @IsString({ each: true }) selectedStrategies?: string[];
  @IsOptional() @IsIn(['MONTHLY', 'ANNUAL']) billingInterval?: 'MONTHLY' | 'ANNUAL';
}
