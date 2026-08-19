# Database Migration Report

## Version: v1.0.0 — Stage 4 PTB

## Migration History

| # | Migration | Status | Type |
|---|-----------|--------|------|
| 001 | Schemas and roles | Applied | Foundation |
| 002 | IAM tables | Applied | Core |
| 003 | Plans/billing/licensing | Applied | Core |
| 004 | Referral/commission/payout | Applied | Core |
| 005 | Trading/market tables | Applied | Core |
| 006 | Session token rotation | Applied | Security |
| 007 | Auth hardening | Applied | Security |
| 008 | Device activation sessions | Applied | Security |
| 009 | Signal delivery replay | Applied | Reliability |
| 010 | Database completion audit | Applied | Audit |
| 011 | COT capability WAL | Applied | Feature |
| 012 | PTB intelligence tables | Pending | Stage 4 PTB |
| 013 | PTB synthesis performance | Pending | Stage 4 PTB |

## Migrations 012-013 Details

### Migration 012: PTB Intelligence Tables
- `trading.ptb_feature_flags` — Module activation states with seed data
- `trading.ptb_evidence_snapshots` — Per-evaluation module results (JSONB)
- `trading.data_provenance_log` — Data source authenticity tracking
- All additive, non-destructive, idempotent

### Migration 013: PTB Synthesis Performance
- `trading.ptb_analysis_history` — Full synthesis records (hypertable)
- `trading.signal_performance` — Signal outcome feedback (hypertable)
- All additive, non-destructive, idempotent
- TimescaleDB hypertable creation wrapped in DO$$ block (fails silently if extension absent)

## Safety

- All migrations are **additive** — no DROP, TRUNCATE, or ALTER of existing tables
- All migrations are **idempotent** where possible (IF NOT EXISTS, ON CONFLICT)
- No migration history resets
- No production data modification outside migration framework
