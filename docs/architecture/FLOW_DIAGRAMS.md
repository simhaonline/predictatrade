# Architecture Flow Diagrams
## v1.17.4 — 30 August 2026

Authoritative visual reference for the runtime flows of Predict-A-Trade. All diagrams are
[Mermaid](https://mermaid.js.org/) — rendered natively on the docs site (docsify-mermaid) and
GitHub. Truth sources: `realtime/cmd/realtime-engine/main.go`, `control/src/**`,
`nginx/sites-available/*.conf`, `windows-agent/deploy/install.ps1`.

---

## 1. System Plane Overview

```mermaid
flowchart TD
    subgraph Edge["Windows / MQL Edge"]
        EA_M[Master Node EA<br/>MARKET_SNAPSHOT + ticks]
        EA_C[Client EA<br/>order execution]
        WS[Windows Agent v1.2.40<br/>pat-master.exe + pat-agent.exe<br/>services :9001 / :9000]
        EA_M <-->|file IPC PAT_master_data.txt| WS
        WS <-->|file IPC / pipes| EA_C
    end

    subgraph Nginx["Nginx edge (80/443 TLS)"]
        NG[[api.predictatrade.com<br/>/api/v1 router]] 
        NG2[[live.predictatrade.com<br/>ws/v1/agent + ws/v1/data]]
        NG3[[platform.predictatrade.com<br/>frontend + /ws]]
        NG4[[downloads.predictatrade.com<br/>agent/EA artifacts]]
    end

    subgraph Go["Go Realtime Plane — pat-realtime :13081/:13091"]
        GW[Gateway HTTP+WS]
        ING[Ingest Bus<br/>DirectBus / NatsBus]
        FEAT[Feature Registry<br/>42 indicators]
        STRAT[Strategy Engines x5]
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

    EA_M -- "wss://live…/ws/v1/data" --> NG2 --> :13091 --> ING
    WS -- "wss://live…/ws/v1/agent" --> NG2 --> :13081 --> GW
    GW --> ING --> FEAT --> STRAT --> GATES --> SIG
    SIG --> PERS & VAL
    SIG -- "SIGNAL (per-client risk filter)" --> WS --> EA_C
    EA_C -- "EXECUTION_ACK / TRADE_RESULT" --> WS --> RECON
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
    participant WS as Windows Agent
    participant E as Go Engine (DirectBus)
    participant F as Feature Registry
    participant S as Strategy Engines
    participant G as 16 Hard Gates
    participant C as Calib Consumer
    participant R as Reconciler
    participant A as Agent Hub (WS)

    MT->>WS: MARKET_SNAPSHOT (file IPC, append-only)
    WS->>E: WS message
    E->>F: aggregate candle (M1..D1)
    F->>S: MarketState (42 indicators + regime)
    S->>G: candidate + geometry (SL/TP1..3, R:R)
    G-->>S: veto per gate (ordered, fail-closed) OR pass
    S->>C: RAW_SCORE → calibrated prob (VALIDATED models only)
    C-->>S: prob | "Pending" (never fabricated)
    alt all gates pass AND executable
        S->>A: SIGNAL payload (per-client margin filter)
        A->>R: RecordDelivery(signalID, agents:N)
    else
        S-->>E: NO-TRADE (first-class) persisted w/ reasons
    end
    Note over R: ACK TTL 2m · fill TTL 10m · deduped ntfy alerts
```

---

## 3. Execution Ack & Fill Reconciliation (BE-6)

```mermaid
sequenceDiagram
    autonumber
    participant EA as Client EA
    participant WS as Windows Agent
    participant E as Engine Ack Handler
    participant R as Reconciler
    participant P as Prometheus / ntfy

    EA->>E: EXECUTION_ACK {ticket, sl, tp, entry}
    E->>E: SL>0 AND |SL - server SL| <= tick_size*2 (digits-aware)
    alt SL violation
        E->>WS: CLOSE_POSITION (NO_SL / MISMATCH)
        Note over WS: 3 violations → agent suspended
    else valid
        E->>R: RecordAcknowledgement(signalID, agent)
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
    participant AG as Windows Agent
    participant CP as pat-control /licensing
    participant E as Go Engine (validate fn)

    EA->>AG: IPC license key (EA input)
    AG->>CP: POST /licensing/devices (fingerprint) / activate
    AG->>E: MASTER_INIT {license_key, device_id}
    E->>CP: SQL licensing.licenses ⋈ control.plans (revoked_at IS NULL)
    alt status ACTIVE
        E-->>AG: LICENSE_STATUS {valid, plan, allowed_strategies, risk caps}
    else REVOKED / SUSPENDED
        E-->>AG: LICENSE_STATUS {invalid}
        E->>AG: DisconnectAgent("license " + status)  // no executable signals
    end
    AG-->>EA: badge LICENSE ACTIVE / INVALID
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
        subgraph Svc["13 services (all healthy)"]
            PG[pat-postgres<br/>TimescaleDB pgvector]
            VK[pat-valkey]
            RT[pat-realtime<br/>13081 + 13091]
            PC[pat-control<br/>13080]
            PF[pat-frontend<br/>13082]
            PST[pat-status]
            PLT[pat-live-terminal]
            NATS[pat-nats optional bus]
            BAK[pat-backtest]
            NG[pat-nginx 80/443]
            PR[pat-prometheus]
            GF[pat-grafana]
            NF[pat-ntfy]
        end
    end
    DC --- PG & VK & RT & PC & PF & PST & PLT & NATS & BAK & NG & PR & GF & NF
    NG --> RT & PC & PF & PST
```

---

## 8. Update / Rollback Flow (Windows Agent)

```mermaid
flowchart TD
    UP[Auto-updater in agent] -->|poll| MF[update-manifest.json per role/arch]
    MF -->|version > local + checksum SHA256 match| DL[download exe]
    MF -->|checksum mismatch| ABORT[reject — log + keep old]
    DL[download ok] --> STP[stop service via PAT_SERVICE_NAME]
    STP --> SW[swap binary + Unblock + Defender exclusion re-apply]
    SW --> ST[restart service]
    ST --> HP{health :9000 or :9001 == 200?}
    HP -->|yes| OK[update complete]
    HP -->|no| RB[rollback to previous exe]
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
| Updates | `windows-agent/internal/updater.go`, `deploy/install.ps1` |
| Backup | `scripts/backup/*`, `infra/env` cron |