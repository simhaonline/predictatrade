# Audit Logging Architecture

## Overview
Production-grade compliance audit logging integrated into the existing NestJS/PostgreSQL/TimescaleDB stack.

## Architecture
```
Browser → Frontend (telemetry.ts) → POST /api/v1/compliance/telemetry
  → NestJS ComplianceController (rate limited, validated)
  → ComplianceService.recordEvent()
  → compliance.client_event_log (TimescaleDB hypertable)
  + audit.audit_events (backward compatible)
```

## Server-Trusted Data (never from client)
- user_id (from JWT)
- client_ip (from socket/proxy chain)
- request_id, correlation_id (server-generated)
- timestamp (server UTC)
- http_method, endpoint, http_status
- geo/ISP/ASN (from server-side IP intelligence)

## Client-Reported Data (untrusted, validated)
- user_agent, language, timezone
- screen dimensions, viewport
- device_pixel_ratio, color_depth, touch_points
- client_hints (limited to 20 fields)

## Trusted Proxy Configuration
- TRUSTED_PROXY_CIDRS env var (default: 172.16.0.0/12,10.0.0.0/8,127.0.0.0/8)
- CF-Connecting-IP checked only from trusted proxies
- X-Forwarded-For leftmost IP used for trusted proxy chain
- Direct connections use socket IP

## Redaction
All metadata and risk_flags are redacted before storage:
- password, token, secret, authorization, cookie, api_key → [REDACTED]
