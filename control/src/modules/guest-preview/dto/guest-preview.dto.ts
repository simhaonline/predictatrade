import {
  IsEmail, IsString, MinLength, MaxLength, IsOptional, IsBoolean, Matches,
} from 'class-validator';

/**
 * Registration form for the guest-preview gate (passwordless, email-OTP).
 * Fields in the exact order required by the spec:
 *   Full name (required) → Email (required) → Phone (optional) → Broker (optional).
 *
 * Email is trimmed + lowercased server-side. The three consent checkboxes are
 * distinct, default UNCHECKED, and validated independently (never combined).
 */
export class GuestRegisterDto {
  @IsString() @MinLength(1) @MaxLength(255)
  fullName: string;

  @IsEmail() @MaxLength(320)
  email: string;

  @IsOptional() @IsString() @MaxLength(50)
  phone?: string;

  @IsOptional() @IsString() @MaxLength(100)
  broker?: string;

  // ─── Three distinct consent flags (all default unchecked on the client) ───
  /** REQUIRED: Terms & Conditions and Privacy Policy. */
  @IsBoolean()
  termsAccepted: boolean;

  /** REQUIRED: Risk acknowledgment (informational/educational, not investment advice). */
  @IsBoolean()
  riskAcknowledged: boolean;

  /** OPTIONAL: Marketing opt-in (email + WhatsApp). User may register WITHOUT this. */
  @IsBoolean()
  marketingOptIn: boolean;
}

export class GuestOtpVerifyDto {
  @IsEmail() @MaxLength(320)
  email: string;

  @IsString() @Matches(/^\d{6}$/, { message: 'Code must be exactly 6 digits' })
  code: string;
}

export class GuestOtpResendDto {
  @IsEmail() @MaxLength(320)
  email: string;
}

export class GuestUnsubscribeDto {
  @IsString() @MinLength(1)
  token: string;
}
