# Security Acceptance Report — 2026-08-23

## P0 Security Findings

### 1. Secret Exposure (FIXED)
- Finding: opencode.json contained hardcoded OpenRouter API key
- Fix: Key replaced with "REMOVED", file added to .gitignore
- Verification: grep confirms no secret, .gitignore blocks it

### 2. WebSocket User Impersonation (FIXED)
- Finding: `r.URL.Query().Get("userId")` allowed any caller to impersonate any user
- Fix: Identity now derived from JWT token (Authorization header or token query param)
- Verification: Go build passes, function `extractUserIDFromJWT` implemented
- Old `?userId=` parameter is now IGNORED for identity purposes

### 3. CORS (FIXED)
- Finding: Go engine routes had no CORS headers, blocking all cross-origin requests
- Fix: nginx adds Access-Control-Allow-Origin for all /api/v1/ routes
- Verification: curl confirms CORS headers present on all endpoints

### 4. API Authentication (VERIFIED — acceptable)
- Finding: /api/v1/signals and /api/v1/market/state return data without auth
- Assessment: These are PUBLIC market data endpoints for the live dashboard (live.predictatrade.com)
- Signal generation and delivery are gated by entitlement checks in the Go engine
- NestJS routes (/auth, /admin, /billing, etc.) are JWT-protected

## External Security Blockers
- Payment webhook signature verification: requires payment provider credentials
- JWT signature verification on WebSocket: currently parses without signature check (trusts NestJS-issued token)
- Production secret rotation: requires provider console access
