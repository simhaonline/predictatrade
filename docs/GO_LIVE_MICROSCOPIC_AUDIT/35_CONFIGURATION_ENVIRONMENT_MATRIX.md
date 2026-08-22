# 35 — Configuration / Environment Matrix (key variables)

| Variable | Service | Used by code | Secret | Prod value present | Startup-validated | Finding |
|---|---|---|---|---|---|---|
| DATABASE_URL | realtime, control | yes | YES | superuser URL | partial | rotate + least-privilege (B-01) |
| VALKEY_ADDR | realtime | dedup/cooldown/cache | (needs auth added) | addr only, no password option set | no | B-02 |
| PROVIDER_MODE=agent | realtime | provider select | no | yes | yes | simulated provider prod-guarded OK |
| SYMBOLS/TICK_RATE_MS/FEATURE_WORKERS | realtime | engine tuning | no | yes | defaults | OK |
| MAX_SPREAD_PIPS/MIN_RR/MAX_COST_TO_TARGET/MAX_EXPOSURE | realtime | gates | no | 5.0/2.0/0.15/5.0 | parsed | exposure semantics coarse (13) |
| ML_ENABLED=true + ONNX_RUNTIME_PATH | realtime | mlengine | no | true | logs show load FAIL → fail-open | models absent in container — config drift (16) |
| OLLAMA_ENABLED/HOST/MODEL/TIMEOUT | realtime | sentiment | no | localhost:11434 unreachable from container | no (silent) | inert path (16) |
| NEWS_PROVIDER/MODE/FAIL_POLICY/*MINUTES | realtime | news gate | key=YES | fmp/PROTECT_ONLY/BLOCK_TRADING | policy constants | LIVE, fail-closed OK |
| COT_ENABLED+FMP_API_KEY; DXY+TWELVEDATA_API_KEY | realtime | macro loops | YES keys | yes | no | plaintext secrets on disk (V-05) |
| NEWS_BREAKOUT_ENABLED=true | realtime | **nothing** | no | true | n/a | dead config advertised as enabled |
| SMTP_*/TELEGRAM_* / NTFY_SERVER_URL | realtime notifier | notifications | YES | smtp+tg set; ntfy port WRONG (8090 vs 8091) | no | silent notification loss |
| JWT_SECRET (+fallback string in code) | control | auth+device crypto | YES | env set at runtime; fallback reachable in device paths | guard covers jwt.module only | V-08 |
| device credential AES key = derived(JWT_SECRET) | control | licensing crypto | derived | active | no | key separation violation |
| compose POSTGRES_PASSWORD / GRAFANA admin | infra | containers | YES | committed to git | n/a | V-05 rotation required |
| NEXT_PUBLIC_API_BASE_URL / WS URL | frontend | endpoints | no | prod URLs in .env.local | no | OK (no secrets client-side) |
| SUBSCRIPTION_V3_ENABLED etc. (DB flags) | none | **unread** | no | all FALSE | n/a | flag gating is documentation-only |

Unused-but-defined env: ADAPTATION_ENABLED, HEDGING_ENABLED(false default), RL_MODE, ML_ADAPTATION_ENABLED — parsed into config structs with zero consumers.
