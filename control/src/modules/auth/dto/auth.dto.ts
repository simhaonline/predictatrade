import { IsEmail, IsString, MinLength, MaxLength, IsOptional, IsBoolean, Matches, IsDefined } from 'class-validator';

export class RegisterDto {
  @IsEmail() email: string;
  @IsString() @MinLength(8) @MaxLength(72) password: string;
  @IsOptional() @IsString() @MaxLength(100) displayName?: string;
  @IsOptional() @IsString() referralCode?: string;

  // ── Required consent fields (must be true) ──
  @IsDefined() @IsBoolean() agreeToTerms: boolean;
  @IsDefined() @IsBoolean() acknowledgePrivacyPolicy: boolean;
  @IsDefined() @IsBoolean() acknowledgeDataProcessing: boolean;

  // ── Optional marketing opt-in fields (default false) ──
  @IsOptional() @IsBoolean() optInEmailMarketing?: boolean;
  @IsOptional() @IsBoolean() optInSmsMarketing?: boolean;
  @IsOptional() @IsBoolean() optInPhoneMarketing?: boolean;
}

export class LoginDto {
  @IsEmail() email: string;
  @IsString() @MinLength(1) password: string;
  @IsOptional() @IsString() mfaCode?: string;
  @IsOptional() @IsBoolean() trustDevice?: boolean;
}

export class MfaSetupDto {
  @IsString() code: string;
}

export class VerifyOtpDto {
  @IsString() challengeId: string;
  @IsString() @Matches(/^\d{6}$/, { message: 'Code must be exactly 6 digits' }) code: string;
  @IsOptional() @IsBoolean() trustDevice?: boolean;
}

export class ForgotPasswordDto {
  @IsEmail() email: string;
}

export class ResetPasswordDto {
  @IsString() token: string;
  @IsString() @MinLength(8) @MaxLength(72) password: string;
}
