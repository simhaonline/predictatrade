---
name: compliance-legal
description: "GDPR consent, terms, privacy, audit logging."
---

# compliance-legal

## Implemented
- Terms of Service (frontend/app/(auth)/terms)
- Privacy Policy (frontend/app/(auth)/privacy)
- Data Processing & Security Agreement
- Signup with 6 consent checkboxes (marketing, data, performance, privacy, terms, age)
- Backend consent tracking with audit logging (migration 026)
- Unsubscribe page
- Cookie consent provider

## Data Protection
- No production secrets in repo (use .env files, env_file in docker-compose)
- JWT secret from env vars, not hardcoded
- PII audit trail in audit.client_events table
- Right to deletion / unsubscribe flows

## GDPR/GDPA Considerations
- Marketing consent: explicit opt-in, tracked per-user
- Data processing consent: explicit, revocable
- Performance analytics consent: explicit
- All consent changes logged to audit trail
