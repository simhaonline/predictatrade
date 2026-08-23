# Data Retention Policy

## Configurable Retention
- AUDIT_LOG_RETENTION_DAYS=365 (default)
- CLIENT_TELEMETRY_RETENTION_DAYS=180 (default)
- SECURITY_EVENT_RETENTION_DAYS=730 (default)

## TimescaleDB Retention
Retention policies are disabled by default. Enable via:
```sql
SELECT add_retention_policy('compliance.client_event_log', INTERVAL '365 days');
```

## Legal Notice
Retention periods must be configured according to applicable law,
company policy, and regulatory requirements.
