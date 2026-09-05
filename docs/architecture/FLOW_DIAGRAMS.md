# Architecture Flow Diagrams
## v1.29.0 — 05 September 2026

Authoritative visual reference for the runtime flows of Predict-A-Trade. All diagrams are
[Mermaid](https://mermaid.js.org/) — rendered natively on the docs site (docsify-mermaid) and
GitHub. Truth sources: `realtime/cmd/realtime-engine/main.go`, `control/src/**`,
`nginx/sites-available/*.conf`, `mql/` EA sources. All diagrams reflect the v1.19+ Option B
EA-direct transport (Windows Agent removed).

---

## 1. System Plane Overview

```mermaid
flowchart TD
    subgraph Edge["Windows / MQL Edge"]
        EA_M[Master Node EA<br/>POST /ingest/agent<br/>MARKET_SNAPSHOT + ticks]
        EA_C[Client EA<br/>HMAC edge-poll + order execution]
    end

    subgraph Nginx["Nginx edge (80/443 TLS)"]
        NG[[api.predictatrade.com<br/>/api/v1 router + /ingest/agent]] 
        NG2[[platform.predictatrade.com<br/>frontend + /ws]]
        NG4[[downloads.predictatrade.com<br/>EA .mq4/.mq5 sources]]
    end

    subgraph Go["Go Realtime Plane — pat-realtime :13081 (single port)"]
        GW[Gateway HTTP+WS]
        ING[Ingest Bus<br/>DirectBus / NatsBus]
        FEAT[Feature Registry<br/>42 indicators]
        STRAT[Strategy Engines x7]
        GATES[16 Hard Gates<br/>fail-closed]
        SIG[Signal Engine + Calibration]
        RECON[Reconciliation<br/>ACK + fill legs]
        PERS[(TimescaleDB)]
        VAL[(Valkey)]
    end

    subgraph Nest["NestJS Control Plane — pat-control :13080"]
        IAM[IAM / MFA / RBAC]
        SUB[Subscriptions / Billing<br/>Stripe + NOWPayments]
        LIC[Licensing / Devices]
        FIN[Commissions / Payouts]
        OPS[Operations / Feature flags]
        AUD[Audit / GDPR]
    end

    subgraph FE["Next.js Presentation — pat-frontend :13082"]
        ADMIN[Admin Console 25 pages]
        USER[User Portal 19 pages]
    end

    subgraph Data[(Datastores)]
        PG[(PostgreSQL 17 +<br/>TimescaleDB, pgvector)]
        VK[(Valkey 8)]
    end

    EA_M -- "POST /ingest/agent (Bearer device JWT)" --> NG --> :13081 --> GW --> ING
    GW --> ING --> FEAT --> STRAT --> GATES --> SIG
    SIG --> PERS & VAL
    EA_C -- "POST /api/v1/devices/edge-poll (HMAC, every ~3s)" --> NG
    NG -- "SIGNAL / CLOSE_POSITION / EMERGENCY_STOP / KILL_SWITCH / LICENSE_STATUS / REQUEST_SNAPSHOT" --> EA_C
    EA_C -- "EXECUTION_ACK / TRADE_RESULT (via /ingest/agent)" --> ING
    SIG -- "executable signals → licensing.edge_signal_queue (fail-closed, plan-filtered)" --> PERS
    RECON -- "gap alerts" --> NTFY[ntfy + Prometheus]

    ADMIN & USER -- "https://api…/api/v1 (JWT)" --> NG --> IAM & SUB & LIC & FIN & OPS & AUD
    IAM & SUB & LIC & FIN & OPS & AUD --> PG
    PERS --- PG
    VAL --- VK
```

---

## 2. Tick → Signal Lifecycle (realtime hot path)

```mermaid
sequenceDiagram
    autonumber
    participant MT as MT5 Master Node EA
    participant E as Go Engine (DirectBus)
    participant F as Feature Registry
    participant S as Strategy Engines
    participant G as 16 Hard Gates
    participant C as Calib Consumer
    participant R as Reconciler

    MT->>E: POST /ingest/agent — MASTER_TICK / MARKET_SNAPSHOT (Bearer device JWT)
    E->>F: aggregate candle (M1..MN)
    F->>S: MarketState (42 indicators + regime)
    S->>G: candidate + geometry (SL/TP1..3, R:R)
    G-->>S: veto per gate (ordered, fail-closed) OR pass
    S->>C: RAW_SCORE → calibrated prob (VALIDATED models only)
    C-->>S: prob | "Pending" (never fabricated)
    alt all gates pass AND executable
        S->>R: enqueue licensing.edge_signal_queue (plan-filtered in SQL)
        Note over R: per-device delivery tracking
    else
        S-->>E: NO-TRADE (first-class) persisted w/ reasons
    end
    Note over E: broker TimeCurrent() is the sole trading clock;<br/>time_mode BROKER_ALIGNED via live DST-adaptive bridge
```

---

## 3. Execution Ack & Fill Reconciliation (BE-6)

```mermaid
sequenceDiagram
    autonumber
    participant EA as Client EA
    participant CP as pat-control (edge-poll API)
    participant E as Go Engine Ack Handler
    participant R as Reconciler
    participant P as Prometheus / ntfy

    EA->>CP: POST /devices/edge-poll (HMAC, always-ACK)
    CP-->>EA: SIGNAL / CLOSE_POSITION / EMERGENCY_STOP / KILL_SWITCH / LICENSE_STATUS / REQUEST_SNAPSHOT
    EA->>E: EXECUTION_ACK via POST /ingest/agent {ticket, sl, tp, entry}
    E->>E: SL>0 AND |SL - server SL| <= tick_size*2 (digits-aware)
    alt SL violation
        E->>CP: CLOSE_POSITION command (NO_SL / MISMATCH)
        Note over EA: 3 violations → device suspended
    else valid
        E->>R: RecordAcknowledgement(signalID, device)
    end
    loop every 30s (startReconciliationMonitor)
        R->>R: UnacknowledgedOlderThan(2m) / UnfilledOlderThan(10m)
        R->>P: gauges pat_reconciliation_*
        R-->>P: ntfy SIGNAL_ACK_TIMEOUT / SIGNAL_FILL_TIMEOUT (deduped)
    end
    EA->>E: TRADE_RESULT {signal_id, ticket, pnl, exit}
    E->>R: RecordFill(signalID, ticket)  // closes fill leg
    E->>P: edge-validation + recovery manager update
```

---

## 4. Authentication & Session Flow

```mermaid
flowchart TD
    L[Login page] -->|POST /auth/login| C2{mfaRequired?}
    C2 -->|yes| OTP[/verify-otp page/] -->|POST /auth/verify-otp| OK2
    C2 -->|no| OK2[JWT access + HttpOnly refresh cookie]
    OK2 -->|setAccessToken memory + pat_access_token cookie| DASH{Role?}
    DASH -->|ADMIN / SUPER_ADMIN| AD[/admin/* console/]
    DASH -->|USER| UD[/dashboard/* portal/]
    AD & UD -.->|every request| AX[axios interceptor<br/>Bearer + 401→refresh single-flight]
    AX -->|refresh ok| CONT[retry original]
    AX -->|refresh fail| OUT[force-logout once, pat:auth-changed]
    subgraph Guards
        JW[JwtAuthGuard] --> RO[RolesGuard] --> PE[PermissionGuard]
    end
    AD --- JW
```

Notes: privileged roles (`ADMIN`, `SUPER_ADMIN`, `OPERATOR`) must enroll MFA before login
completes (`mfaEnrollmentRequired`).

---

## 5. Licensing / Device Activation Flow

```mermaid
sequenceDiagram
    participant EA as Client EA
    participant CP as pat-control /licensing + /devices
    participant E as Go Engine (validate fn)

    EA->>CP: POST /api/v1/devices/activate {license key, fingerprint}
    CP->>CP: SQL licensing.licenses ⋈ control.plans (revoked_at IS NULL)
    alt status ACTIVE
        CP-->>EA: device_id + device_secret + refresh token → Bearer device JWT
        EA->>E: MASTER_INIT via POST /ingest/agent {license_key, device_id}
        E->>CP: SQL validate
        E-->>EA: LICENSE_STATUS {valid, plan, allowed_strategies, risk caps} on next edge-poll
    else REVOKED / SUSPENDED
        CP-->>EA: activation DENIED — no executable signals
        Note over EA: EA fails closed (signal-only display)
    end
    Note over EA,CP: entitlement re-checked at every poll — a license revoked<br/>between enqueue and poll expires the queued signal
```

---

## 6. Payment → Entitlement Flow

```mermaid
flowchart LR
    U[User] -->|choose plan + strategies| S[POST /subscriptions<br/>server-side entitlement validation]
    S --> F[plan FREE → ACTIVE invoice paid<br/>paid → INCOMPLETE]
    F --> SP[Stripe / NOWPayments webhook<br/>HMAC verified, raw body]
    S2{webhook paid?}
    S2 -->|yes| ENT[entitlements: allowed_strategies<br/>risk caps, device limits]
    S2 -->|no| PEND[stay INCOMPLETE]
    ENT --> LIC[licensing.licenses ACTIVE row<br/>PAT-XXXXXXXX key]
    LIC --> E[Engine serves executable signals]
```

---

## 7. Deployment Topology (Docker-first, no systemd)

```mermaid
flowchart TD
    subgraph Host["Single VPS host"]
        DC[docker compose — infra/env/.env]
        subgraph Svc["16 services"]
            PG[pat-postgres<br/>TimescaleDB pgvector]
            VK[pat-valkey]
            RT[pat-realtime<br/>13081 single port]
            PC[pat-control<br/>13080]
            PCB[pat-control-b<br/>blue/green twin]
            PF[pat-frontend<br/>13082]
            PST[pat-status]
            PLT[pat-live-terminal]
            NATS[pat-nats optional bus]
            BAK[pat-backtest]
            MR[pat-mail-relay + spool]
            BUS[pat-backup-sync]
            NG[pat-nginx 80/443]
            PR[pat-prometheus]
            GF[pat-grafana]
            NF[pat-ntfy]
        end
    end
    DC --- PG & VK & RT & PC & PCB & PF & PST & PLT & NATS & BAK & MR & BUS & NG & PR & GF & NF
    NG --> RT & PC & PCB & PF & PST
```

---

## 8. EA Update / Rollback Flow (Option B — no local binaries)

```mermaid
flowchart TD
    UP[New EA source pushed to repo] -->|CI / deploy| MF[downloads.predictatrade.com/mql/<br/>PredictATrade*.mq4/.mq5 + MasterNode variants]
    TR{Trader terminal}
    TR -->|auto-notice or dashboard prompt| DL[download .mq5/.mq4]
    DL --> C[compile in MetaEditor F7 — 0 errors]
    C --> RE[re-attach EA to XAUUSD chart]
    RE --> ACT[EA re-activates device via refresh token<br/>state file per platform in Common\\Files]
    ACT --> HP{edge-poll returns LICENSE_STATUS valid?}
    HP -->|yes| OK[update complete — version banner on dashboard]
    HP -->|no| RB[re-download previous source / restore state file backup]
```

---

## 9. Backup / DR Flow

```mermaid
flowchart LR
    CR[cron/pg_cron] --> BK[pg_dump --format=custom<br/>+ Valkey RDB + /var/www artifacts]
    BK --> OFH[(off-host storage)]
    OFH --> VAL[restore_test.sh drill<br/>row-count + latest timestamp assertions]
    VAL -->|PASS| READY[dr-ready]
    VAL -->|FAIL| ALERT[ntfy DR_VALIDATION_FAILED]
```

---

## Traceability

| Flow | Source of truth |
|---|---|
| Signal lifecycle | `realtime/internal/reconciliation/reconciler.go`, `cmd/realtime-engine/main.go` |
| Ack enforcement | `main.go` EXECUTION_ACK handler (SL verify + CLOSE_POSITION) |
| Auth | `control/src/modules/auth/*`, `frontend/src/lib/auth.ts`, `frontend/src/proxy.ts` |
| Licensing | `control/src/modules/licensing/*`, engine `SetLicenseValidateFn` |
| Payments | `control/src/modules/billing/*`, `subscriptions.service.ts` |
| Updates | `mql/` EA sources → `downloads.predictatrade.com/mql/` (nginx `mql/` mount); MetaEditor F7 recompile |
| Backup | `scripts/backup/*`, `infra/env` cron |