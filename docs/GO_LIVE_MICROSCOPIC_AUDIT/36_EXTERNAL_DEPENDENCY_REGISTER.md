# 36 — External Dependency Register

| Dependency | Version (observed) | Purpose | Risk state |
|---|---|---|---|
| PostgreSQL/TimescaleDB image | timescale/timescaledb-ha:pg17, tsdb 2.29.1 | datastore | healthy; exposure P0 |
| Valkey | valkey/valkey:8.0 | cache/dedup | no-auth public P0 |
| Go toolchain | go.mod (repo) | realtime+agent builds | vet clean; targeted tests pass |
| Node 24 (control Dockerfile) | node:24 | NestJS runtime | current; passport-* deps unused |
| Next.js | 16.3.1 / React 19.2.8 | frontend | very new majors — pin policy + e2e recommended |
| onnxruntime shared lib | /usr/local/lib/libonnxruntime.so | ML inference | models absent → inert |
| Ollama | host service | sentiment | unreachable from container net |
| FMP API | key present | COT + economic calendar | LIVE; plaintext key |
| Twelve Data API | key present | DXY (+fallback price source) | LIVE; plaintext key |
| Stripe | **no SDK** | payments | claimed in data only — must integrate or remove |
| SMTP mail.predictatrade.com | :587 STARTTLS | email | unverified send |
| Telegram Bot API | token present | notifications | unverified send |
| ntfy | binwiederhier/ntfy:latest | push | port mismatch config |
| Grafana/Prometheus | latest tags | observability | unpinned versions; default creds committed |
| Let's Encrypt certs | mounted volume | TLS | valid; renewal automation presence UNVERIFIED (no certbot service found) |
| MetaTrader terminals | user-side MT4/MT5 | execution | EXTERNAL BLOCKER for qualification |

Unpinned `latest` images (grafana/prometheus/ntfy/nginx-alpine) violate reproducible-deploy guidance (P3).
