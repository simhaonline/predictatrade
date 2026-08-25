# Legal Compliance — Terms, Privacy & Consent Tracking

**Version:** v1.12.0+ (25 August 2026)  
**Status:** ACTIVE — market-standard legal documents + backend consent tracking

## Legal Documents

### Terms of Service (`/terms`)
- 18 sections covering: acceptance, service description, eligibility, accounts, trading risk disclaimer, subscriptions/billing, referral program, prohibited conduct, IP, MT integration, disclaimers, limitation of liability, indemnification, termination, governing law, severability
- Last updated: 25 August 2026

### Privacy Policy (`/privacy`)
- 16 sections: PDPL/GDPR compliant
- Covers: data controller, data categories (direct/automatic/special), legal basis, usage, sharing, international transfers, retention schedule, data subject rights, cookies, children's privacy, marketing consent, security measures, UAE PDPL compliance, breach notification
- Last updated: 25 August 2026

### Data Processing and Security Agreement (`/data-processing-agreement`)
- 14 sections: technical + organizational measures
- Covers: roles/responsibilities, data categories, processing purposes, encryption (TLS 1.2+, bcrypt 12, JWT RS256), access control (RBAC, MFA), network security (WAF, rate limiting), application security (input validation, CSRF, idempotency), infrastructure (PostgreSQL+TimescaleDB, Valkey, Docker), sub-processors, breach notification (72h), data subject rights, audit
- Last updated: 25 August 2026

## Consent Tracking

### Required Consents (must be true to register)
| Field | Consent Text |
|------|--------------|
| `agreeToTerms` | "I agree to the Terms of Use and Privacy Policy" |
| `acknowledgePrivacyPolicy` | "I confirm that I have read and acknowledge the Simha FinTech Terms of Service" |
| `acknowledgeDataProcessing` | "I confirm that I have read and acknowledge the Privacy Policy and Data Processing and Security Agreement" |

### Optional Marketing Preferences
| Field | Consent Text | Default |
|------|--------------|---------|
| `optInEmailMarketing` | "I want to receive news and promotional offers by email" | false |
| `optInSmsMarketing` | "I want to receive news and promotional offers by SMS" | false |
| `optInPhoneMarketing` | "I want to receive news and promotional offers by phone call" | false |

### Backend Storage
- **RegisterDto**: 6 consent fields validated by class-validator
- **AuthService.register()**: throws BadRequestException if required consents are false
- **audit.client_events**: all 6 consents logged with type, accepted, textVersion, consentText
- **iam.users**: marketing_email_optin, marketing_sms_optin, marketing_phone_optin, consent_version, consent_timestamp
- **Migration 071**: forward-only, additive columns with safe defaults

### Frontend
- Login page: email/password, remember me, registration success banner
- Signup page: full name, password strength meter (5-level), 6 consent checkboxes with links to legal documents
- Submit disabled until all required consents checked + valid form
- Footer: links to all 3 legal documents
