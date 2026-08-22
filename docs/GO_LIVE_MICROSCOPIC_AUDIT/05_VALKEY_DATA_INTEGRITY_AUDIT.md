# 05 — Valkey / Cache Integrity Audit

Instance: Valkey 8.0 (docker `pat-valkey`), **published 0.0.0.0:6379, no auth, no TLS** — remote unauthenticated PING verified (`+PONG`) → F-002 P0.

## Live key inventory at audit time

```
db0 keys=5: pat:agent_status (+TTL), backup1..4
```

Namespaces observed in code but absent at audit time due to TTL expiry during weekend close (consistent with design): `pat:latest_signals` (10s TTL REST cache), `signal:cooldown:{symbol}:{strategy}`, duplicate-fingerprint SETNX keys (30min), candle-cache keys.

## Authoritative vs cached classification

| Data | Store | Recoverable? |
|---|---|---|
| Ticks/candles/market states/signals/finance | PostgreSQL/Timescale | YES (durable truth) |
| Latest-signals REST cache | Valkey 10s TTL | yes (rebuilds from DB) |
| Cooldown + duplicate fingerprints | **Valkey ONLY** | **NO** — restart/flush clears dedup state; with public write access an attacker can DELETE them to force duplicates, or SET false positives to block strategies |
| Agent status | Valkey `pat:agent_status` | rebuild on heartbeat |
| Entitlement/license caches | none implemented in control | n/a |

## Findings

- **05-1 (P0):** Public unauthenticated write access to the dedup/cooldown namespace defeats §86 duplicate-signal prevention and can silently suppress all BUY/SELL via planted keys. Fail-open behavior when Valkey is down (`cooldown.go:108-112`) compounds this: flush ⇒ dedup off.
- **05-2 (P2):** No distributed locks/leader election for engine singleton; acceptable single-instance today, undocumented for scale-out.
- **05-3 (PASS):** Durable financial/trading truth correctly lives in Postgres, not Valkey (SOW-compliant separation).
- **05-4 (UNVERIFIED):** Pub/sub usage — none found for WS fan-out (in-process hubs only); horizontal scaling of WS would silently break envelope sequencing.
