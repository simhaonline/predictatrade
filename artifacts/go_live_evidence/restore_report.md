# Backup/Restore Validation Report

**Generated:** 2026-08-22T11:12:20.715169+00:00
**Mode:** DRY_RUN
**RTO:** 14.1s (target: < 3600s) ✅ PASS
**RPO:** 0s (target: < 60s) ✅ PASS
**Errors:** 0

## Operations

| Step | Status | Time (ms) |
|------|--------|-----------|
| backup_create | OK | 1500 |
| backup_verify | OK | 200 |
| staging_restore | OK | 12000 |
| data_integrity_check | OK | 300 |
| consistency_verify | OK | 100 |

## Summary

- Total time: 0.0s
- RTO: 14.1s ✅
- RPO: 0s ✅
- Errors: 0
