---
name: rate-limiting-security
description: "Implement rate limiting on auth and API endpoints."
---

# rate-limiting-security

Use when implementing rate limits on PAT endpoints.

## Limits
/login: 5/min per IP + 3/min per email
/register: 2/min per IP
/verify-otp: 10/min per IP
/reset-password: 3/min per email
/refresh: 30/min per user
API: tiered (free 30/min, pro 300/min, enterprise 1000/min)

## Implementation
NestJS: @nestjs/throttler with @Throttle decorator
Go: tollbooth middleware
Distributed: Valkey-backed rate limiter for multi-instance

## Audit
Rate limit headers in responses
Burst allowance
No bypass via header manipulation
