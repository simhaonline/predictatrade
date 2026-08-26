import { IsString, IsNumber, IsDateString, IsOptional, IsIn } from 'class-validator';

export class RunBacktestDto {
  @IsString()
  @IsIn(['STANDARD_SCALPING', 'ULTRA_SCALPING', 'STANDARD_SWING', 'TREND_SWING', 'MARNIE_FIB'])
  strategy: string;

  @IsString()
  @IsIn(['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'])
  timeframe: string;

  @IsDateString()
  startDate: string;

  @IsDateString()
  endDate: string;

  @IsOptional()
  @IsNumber()
  initialBalance?: number;

  @IsOptional()
  @IsString()
  higherTimeframes?: string;
}
