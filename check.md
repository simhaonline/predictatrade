i want to add one more module for Vedic and Western Astro-Financial Intelligence Engine with Signal Screens, Interactive Mind Maps, Pricing Tiers and add to our scroing system as well, this only offer in Elite Plan, take deep reference from https://github.com/simhaonline/predictatrade-ml.git https://github.com/simhaonline/Vedic_Astrology_Quantitative_Trading_System.git, https://astro.tralgos.com/, https://agiquantlabs.sbs/ do not copy paste any code, you have to build from scratch i just give you reference to understand, build full codebase, also use your own brain to build best

# PREDICT-A-TRADE
## Master Scope of Work — v5.0
### AI + Divine Intelligence + Chinese Wisdom + Western Astrology Algorithmic Trading Intelligence Platform
### With AI-Driven MetaTrader + Crypto Exchange Execution | White-Label Multi-Tenant | TradingView Integration

> **Version:** 5.0 | **Date:** April 2026 | **Classification:** Confidential — Internal Planning
> **Primary Asset:** XAUUSD (Gold vs USD) — Multi-Asset Expandable Architecture (104 Forex/TradFi + Crypto)
> **Database:** PostgreSQL 16 + TimescaleDB | **Runtime:** Alma Linux 10 (Bare Metal / Dedicated VPS)
> **Python Runtime:** Miniconda | **Node Runtime:** NVM | **Cache:** Valkey | **Backup:** Wasabi S3
> **Data Range:** 2015–2030 (Historical + Live + Forward Projection Storage)
> **AI Backbone:** GLM-4 / GLM-4-Long / GLM-4-Flash by Zhipu AI
> **Dev Agent:** Claude Code (Anthropic) — development tooling only, not production AI

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Scope Boundary](#2-scope-boundary)
3. [System Architecture & Technology Stack](#3-system-architecture--technology-stack)
4. [The Nine Intelligence Pillars](#4-the-nine-intelligence-pillars)
5. [Functional Modules](#5-functional-modules)
6. [Divine Intelligence (DI) Engine — Vedic Jyotish + Shadbala + Western Astrology](#6-divine-intelligence-di-engine)
7. [Chinese Wisdom Engine — Feng Shui + Bazi + Five Elements + I Ching + ZWDS](#7-chinese-wisdom-engine)
8. [Timezone & Global Session Management Engine](#8-timezone--global-session-management-engine)
9. [AI Signal Engine Specification](#9-ai-signal-engine-specification)
10. [AI-Driven MetaTrader Execution Engine](#10-ai-driven-metatrader-execution-engine)
11. [Data Engineering & Feature Extraction](#11-data-engineering--feature-extraction)
12. [Backtesting & Walk-Forward Validation Protocol (2015–2030)](#12-backtesting--walk-forward-validation-protocol-20152030)
13. [Risk Management & Controls](#13-risk-management--controls)
14. [Database Architecture — 12 Schemas](#14-database-architecture--12-schemas)
15. [Integration Landscape](#15-integration-landscape)
16. [Subscription Tier Architecture](#16-subscription-tier-architecture)
17. [Delivery Phases & Milestones — 30 Phases](#17-delivery-phases--milestones--30-phases)
18. [Non-Functional Requirements](#18-non-functional-requirements)
19. [Performance KPIs & Acceptance Criteria](#19-performance-kpis--acceptance-criteria)
20. [Agent Ecosystem — 10 Automation Agents](#20-agent-ecosystem--10-automation-agents)
21. [Bare-Metal Deployment Overview](#21-bare-metal-deployment-overview)
22. [Assumptions & Exclusions](#22-assumptions--exclusions)
23. [White-Label Multi-Tenant Reseller Architecture](#23-white-label-multi-tenant-reseller-architecture)
24. [TradingView Integration](#24-tradingview-integration)
25. [Copy-Trade Prohibition Policy](#25-copy-trade-prohibition-policy)
26. [Abbreviations & Glossary](#26-abbreviations--glossary)
27. [Crypto Exchange Integration — Custom Automated Solution](#27-crypto-exchange-integration)

---

## 1. Project Overview

### 1.1 Platform Vision

**Predict-A-Trade** is a production-grade, institutional-quality trading intelligence platform that fuses **Artificial Intelligence (AI)**, **Divine Intelligence (DI / Vedic Jyotish Astrology)**, **Chinese Wisdom (CW / Feng Shui, Bazi, Five Elements, I Ching, ZWDS)**, and **Western Astrology** to generate auditable, explainable market signals — and execute them autonomously via **MetaTrader 4/5** and **Crypto Exchanges (Binance, OKX, Crypto.com, Bybit, KuCoin)**. The platform serves retail and professional traders with real-time signal delivery, strategy research, autonomous execution, and comprehensive risk controls — initially focused on **XAUUSD (Gold)** with a fully multi-asset expandable architecture covering Forex, Indices, Commodities, and **Crypto (Spot + Perpetual Futures)**.

The platform is deployed entirely on **dedicated bare-metal servers or VPS instances running Alma Linux 10 Enterprise**, with no dependency on cloud orchestration layers, Docker Swarm, or Kubernetes. All services run as native system processes managed by **systemd**.

This platform integrates **3,000+ years of Eastern wisdom** — from Vedic Jyotish planetary cycles to Chinese Bazi destiny analysis, Feng Shui energy mapping, Five Elements sector rotation, and I Ching hexagram market state detection — with cutting-edge machine learning, Large Language Model (GLM-4) intelligence, and institutional-grade quantitative risk analytics.

### 1.2 Eight Intelligence Pillars

| Pillar | Description |
|--------|-------------|
| **AI Signal Engine** | ML/DL models (XGBoost → LSTM → Transformer) trained on price, COT, DI, CW, Western astrology features, and macro data producing directional signals with confidence scores |
| **DI / Vedic Jyotish Engine** | 17-body planetary positions, 27 nakshatras × 4 padas, hora timing, Vimshottari dasha (L1/L2/L3), **Shadbala 6-component planetary strength**, eclipse windows, and 9 apocalypse triggers mapped to gold market bias |
| **Chinese Wisdom Engine** | Feng Shui flying stars + Qi flow, Bazi trading clock (4 Pillars), Five Elements sector rotator, I Ching 64-hexagram calculator, **Zi Wei Dou Shu (Purple Star) natal chart engine** — 3,000+ years of Chinese market timing |
| **Western Astrology Engine** | Tropical zodiac; 10 planets; 12-house natal + transit charts; aspect matrix (conjunction/trine/square/opposition/sextile + minor); retrograde tracking (Mercury/Venus/Mars Rx); ingresses; eclipse + lunation cycles; Saturn/Jupiter long-period cycles |
| **AI-Driven MT4/MT5 Execution** | ONNX-native model execution inside MQL5 EAs; ZeroMQ bridge for complex models; full autonomous order lifecycle with circuit breakers; **no copy-trading** |
| **COT Analytics Engine** | CFTC Commitment of Traders positioning intelligence with derived sentiment extremes and historical percentile ranking |
| **Seasonality Engine** | Historical seasonal pattern analysis by instrument, timeframe, and calendar period spanning 2015–2030 |
| **Composite Scoring System** | Multi-factor aggregation producing a −300 to +300 directional score; 15 factor groups including Western astrology and ZWDS |
| **Backtesting & Validation** | Walk-forward strategy validation (2015–2030 data window) with realistic cost models before signals enter production |

### 1.3 Composite Score Thresholds

| Score Range | Signal Direction | Position Size | Tier |
|-------------|-----------------|---------------|------|
| +150 to +300+ | Extreme Bullish ↑↑↑ | 100–150% | Tier 1 |
| +75 to +150 | Strong Bullish ↑↑ | 75–100% | Tier 2 |
| +30 to +75 | Moderate Bullish ↑ | 40–75% | Tier 3 |
| −30 to +30 | Neutral ↔ | 0–30% | Neutral |
| −75 to −30 | Moderate Bearish ↓ | Short / Hedge | Tier 3 |
| −150 to −75 | Strong Bearish ↓↓ | Heavy Short | Tier 2 |
| −300 to −150 | Extreme Bearish ↓↓↓ | Maximum Defensive | Tier 1 |

### 1.4 Composite Score Factor Weights

| Factor Group | Max Contribution | Sub-Factors |
|---|---|---|
| AI Model Confidence | ±80 | XGBoost, LSTM, Transformer consensus |
| DI Score (Vedic) | ±70 | Nakshatra, hora, dasha, Shadbala, eclipse, apocalypse |
| Chinese Wisdom Score | ±50 | I Ching, Feng Shui, Bazi, Five Elements, ZWDS |
| Western Astrology Score | ±30 | Transits, aspects, retrogrades, ingresses, lunations |
| COT Positioning | ±40 | Commercial net, non-commercial, OI delta |
| Seasonality | ±10 | Month/week/day pattern win-rate |
| Macro & Sentiment | ±20 | FinBERT + GLM-4 macro analysis |
| **TOTAL** | **±300** | |

> **v4.0 rebalance note:** DI reduced from ±80 → ±70; Western Astrology added ±30; Seasonality reduced from ±30 → ±10 to accommodate the new pillars while keeping the ±300 ceiling unchanged.

> **Override Rules:** Any active MANDATORY_FLAT trigger (DI Apocalypse, Eclipse Window, Rahu Kala, I Ching Hexagram 29/Kǎn Double Danger, Flying Star 5 at center) forces composite score to 0 and blocks all new execution regardless of other factors.

### 1.5 Validated Performance (XAUUSD 2020–2025)

| Trigger Category | Events | Accuracy |
|-----------------|--------|----------|
| DI Eclipse Reversals | 8 | 87.5% |
| DI Nakshatra Timings | 5 | 100.0% |
| DI Galactic Center Transits | 3 | 100.0% |
| DI Apocalypse Triggers | 4 | 100.0% |
| DI Dasha + Aspect Combinations | 9 | 77.8% |
| I Ching Hexagram 24 (Fù — Return) Reversals | 6 | 83.3% |
| I Ching Hexagram 11 (Tài — Prosperity) Runs | 4 | 75.0% |
| Bazi Day Officer 成 (Chéng) Momentum Days | 7 | 71.4% |
| Five Elements Metal Season Gold Bias | 12 | 83.3% |
| **OVERALL SYSTEM (DI + CW combined)** | **58** | **84.5%** |

### 1.6 AI Model Target KPIs (Out-of-Sample)

| Metric | Target |
|--------|--------|
| Directional Accuracy | 60% – 70% (>75% rejected as overfitted) |
| Sharpe Ratio | > 1.5 (minimum 1.0) |
| Maximum Drawdown | < 15% |
| Profit Factor | > 1.3 (after all transaction costs) |
| Execution Latency (ONNX) | < 200ms signal-to-order |

---

## 2. Scope Boundary

### 2.1 In Scope

- Full system architecture, technical design, API contracts, and data flow diagrams
- Bare-metal Alma Linux 10 server configuration, hardening, and service management
- Backend: modular monolith API (FastAPI + Python), signal engine, DI engine, Chinese Wisdom engine, COT engine, seasonality engine, alert delivery, MQL export, admin ops
- Frontend: public website, Verdict Terminal subscriber dashboard, admin console (Next.js 15)
- Timezone engine: UTC/GMT base computation, API-driven client timezone conversion (IANA), timezone-aware signal delivery and dashboard display
- Data platform: historical and live market data ingestion (2015–2030), symbol normalization, TimescaleDB hypertables, COT/seasonality pipelines
- DI engine: Swiss Ephemeris integration, 17-body ephemeris, 27 nakshatras, hora calendar, dasha system (Vimshottari + sub-periods), Shadbala 6-component, eclipse/apocalypse detection
- Chinese Wisdom engine: Feng Shui, Bazi, Five Elements, I Ching, ZWDS (5 sub-modules)
- Western Astrology engine: tropical zodiac, aspects, retrogrades, ingresses, Gold natal chart transit analysis
- AI engine: feature engineering pipeline, XGBoost/LSTM/Transformer, ONNX export, GLM-4 inference, composite scoring
- AI-Driven MT4/MT5 execution: MQL5 EA with native ONNX, ZeroMQ bridge, order lifecycle, circuit breakers
- **Crypto Exchange Integration (Custom):** Binance, OKX, Crypto.com, Bybit, KuCoin — spot + perpetual futures; custom connector framework; REST + WebSocket; PAT signal → order routing; multi-exchange position management; order book aggregation; funding rate monitoring; cross-exchange arbitrage alerts
- Strategy lifecycle: backtesting (2015–2030 walk-forward), out-of-sample validation, analyst approval workflow
- Notifications: in-app, email, Telegram, WhatsApp, webhook with retry logic and audit trail
- MQL4/MQL5 export: template-based code generation, parameter injection, SHA-256 versioned packages
- Monitoring: Grafana + Prometheus + Loki (bare-metal, no cloud)
- Backup & DR: automated Wasabi S3 backup pipeline with restore validation

### 2.2 Out of Scope

- Native iOS or Android mobile applications
- Broker-dealer licensing or custody infrastructure
- High-frequency exchange co-location (sub-millisecond HFT)
- Full institutional OMS/EMS
- Cloud provider services (AWS, GCP, Azure) — bare-metal only
- Docker / Kubernetes / container orchestration of any kind
- Fully autonomous portfolio management without analyst-approved signals
- Copy-trading functionality — subscribers may NOT retransmit, resell, or copy signals to third parties (see Section 25)

### 2.3 Promoted into Scope (v4.0 additions)

| Feature | Added In | Section |
|---------|----------|---------|
| Shadbala planetary strength full computation (6 components) | v4.0 | Section 6.7 |
| Western Astrology Module (tropical, aspects, transits, retrogrades) | v4.0 | Section 6.8 |
| Zi Wei Dou Shu (紫微斗數) full natal chart engine | v4.0 | Section 7.5 |
| White-label multi-tenant reseller platform | v4.0 | Section 23 |
| TradingView Trade Signal integration (webhook receiver + alert mapping) | v4.0 | Section 24 |
| Crypto payments via NOWPayments gateway | v4.0 | Section 16.2 |
| **Crypto Exchange Integration** — Binance, OKX, Crypto.com, Bybit, KuCoin; custom connector framework; spot + perpetuals execution | v5.0 | Section 27 |

---

## 3. System Architecture & Technology Stack

### 3.1 Architecture: Modular Monolith on Bare Metal

```
┌───────────────────────────────────────────────────────────────────┐
│                      ALMA LINUX 10 SERVER                         │
│                                                                   │
│  ┌──────────┐  ┌────────────────┐  ┌──────────┐  ┌───────────┐    │
│  │  NGINX   │  │  FastAPI API   │  │  Celery  │  │APScheduler│    │
│  │ Reverse  │→ │  (pat-api env) │  │  Worker  │  │Scheduler  │    │
│  │  Proxy   │  │  port 8000     │  │(pat-api) │  │(pat-data) │    │
│  └──────────┘  └────────────────┘  └──────────┘  └───────────┘    │
│                        │                │              │          │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │              VALKEY 8.x (Cache / Queue / PubSub)             │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                        │                                          │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │   PostgreSQL 16 + TimescaleDB  — 13 schemas, 9 hypertables   │ │
│  │   Data Range: 2015–2030 | PgBouncer connection pool          │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │ Grafana  │  │Prometheus│  │  Loki +  │  │ MLflow           │   │
│  │Dashboard │  │ Metrics  │  │ Promtail │  │ Experiments      │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │     Next.js 15 Frontend (Static Build + SSR) — port 3001     │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌───────────────────────────────────────┐  ┌──────────────────┐  │
│  │  MT4/MT5 Bridge — ZeroMQ port 5555    │  │  Crypto Exchange │  │
│  │  + ONNX Runtime (pat-ml env)          │  │  Connector :8001 │  │
│  │  predictatrade-bridge.service         │  │  (pat-api env)   │  │
│  └───────────────────────────────────────┘  └──────────────────┘  │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  HashiCorp Vault (Secrets) | Wasabi S3 (Backup & Artifacts)  │ │
│  └──────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

### 3.2 Full Technology Stack

| Layer | Technology | Version / Notes |
|-------|-----------|-----------------|
| **OS** | Alma Linux | 10 Enterprise (SELinux enforcing) |
| **Python Runtime** | Miniconda | Latest; 3 conda envs: pat-api, pat-ml, pat-data |
| **Node.js Runtime** | NVM | LTS (v22.x); Next.js frontend builds |
| **Web Server / Proxy** | NGINX | Latest stable; TLS termination, WebSocket proxy |
| **API Framework** | FastAPI | Python 3.12; async; OpenAPI auto-docs |
| **Primary Database** | PostgreSQL 16 + TimescaleDB | Local; 13 schemas; 7 hypertables |
| **Cache / Queue / PubSub** | Valkey | 8.x (Redis-compatible OSS fork) |
| **Connection Pool** | PgBouncer | Transaction mode; max 1000 client connections |
| **Task Queue** | Celery + Valkey broker | Background ML inference, alerts, retraining |
| **Scheduler** | APScheduler | Cron-style: COT ingestion, DI/CW computation, model retraining |
| **AI Backbone (LLM)** | Zhipu AI GLM-4 API | glm-4 / glm-4-long / glm-4-flash / glm-4v models |
| **ML / Data Science** | Pandas, NumPy, Polars, Scikit-learn, XGBoost, LightGBM | pat-ml env |
| **Deep Learning** | PyTorch (primary), TensorFlow 2.x | pat-ml env; GPU-optional |
| **Time Series ML** | `darts`, `tsfresh`, `sktime` | Temporal model training |
| **NLP / Sentiment** | HuggingFace Transformers (FinBERT) | Macro news sentiment |
| **ONNX Runtime** | `onnxruntime` (Python), ONNX SDK (MQL5) | Cross-layer model serving |
| **Experiment Tracking** | MLflow | Self-hosted port 5000; Wasabi S3 artifact store |
| **Vedic Ephemeris** | Swiss Ephemeris (pyswisseph) | 17-body sidereal; 1-min resolution; 2015–2030 pre-cached; Lahiri ayanamsa |
| **Western Astrology** | Swiss Ephemeris (pyswisseph, ayanamsa=0) + `kerykeion` Python library | Tropical 10-planet positions, aspects, retrogrades, ingresses, natal chart transit engine |
| **Shadbala Engine** | Custom Python (no PyPI lib; implemented from Brihat Parashara Hora Shastra) | 6-component planetary strength; pre-computed 2015–2030 |
| **Chinese Wisdom** | `ephem`, `lunardate`, custom Bazi engine | Bazi pillars, flying stars, I Ching logic |
| **ZWDS Engine** | Custom Python implementation (紫微斗數 from classical texts) | 14 major stars, 12 palaces, daily flow activation; no external library |
| **Timezone Engine** | `zoneinfo` (stdlib), `pytz`, `dateutil` | All IANA timezones; UTC base computation |
| **Frontend** | Next.js 15, TailwindCSS, TradingView Lightweight Charts | NVM-managed |
| **Real-Time** | WebSocket (FastAPI native) + Server-Sent Events | Dashboard live updates |
| **Notifications** | SendGrid (email), Telegram Bot API, WhatsApp Business API | |
| **Payments** | Stripe API + webhooks; NOWPayments crypto gateway (BTC/ETH/USDT/USDC/LTC/BNB/SOL/XRP + 90 more) | Subscription lifecycle; crypto invoice → IPN confirmation |
| **MT4/MT5 Bridge** | ZeroMQ (primary), HTTP/JSON FastAPI (secondary) | TCP socket bridge port 5555 |
| **MT5 ONNX** | Native ONNX in MQL5 | Sub-200ms execution |
| **Crypto Exchange Connectors** | Custom Python framework (no ccxt dependency); `websockets`, `aiohttp`, `cryptography` | Binance, OKX, Crypto.com, Bybit, KuCoin — REST + WebSocket; per-exchange HMAC-SHA256/SHA512 signing |
| **Observability** | Prometheus + Grafana + Loki + Promtail | All self-hosted |
| **Backup** | Wasabi S3 (`rclone` / `s3cmd`) | Daily WAL, weekly full, monthly archive |
| **Process Management** | systemd | One unit file per service component |
| **Secrets** | HashiCorp Vault (self-hosted) | API key rotation, DB credentials |

### 3.3 systemd Services

| Service | Conda Env | Port | Description |
|---------|-----------|------|-------------|
| `predictatrade-api` | pat-api | 8000 | FastAPI application |
| `predictatrade-worker` | pat-api | — | Celery background workers |
| `predictatrade-bridge` | pat-ml | 5555 | ZeroMQ + ONNX MT bridge |
| `predictatrade-exchange` | pat-api | 8001 | Crypto exchange connector service (WebSocket feeds + order routing) |
| `predictatrade-frontend` | — (NVM) | 3001 | Next.js SSR frontend |
| `predictatrade-scheduler` | pat-data | — | APScheduler cron tasks |
| `predictatrade-mlflow` | pat-ml | 5000 | MLflow tracking server |
| `valkey` | — | 6379 | Valkey cache/queue |
| `postgresql-16` | — | 5432 | PostgreSQL database |
| `pgbouncer` | — | 6432 | Connection pooler |
| `prometheus` | — | 9090 | Metrics scraper |
| `grafana-server` | — | 3000 | Dashboard UI |
| `loki` | — | 3100 | Log aggregation |
| `promtail` | — | — | Log shipper |

---

## 4. The Nine Intelligence Pillars

### Pillar 1 — AI Signal Engine
ML/DL models trained on multi-domain feature sets: price/volume, technical indicators, DI state, Chinese Wisdom state, COT, seasonality, and macro sentiment. Model family: XGBoost baseline → LSTM/GRU → LSTM-Transformer hybrid → GLM-4 LLM reasoning layer. Output: directional classification (buy/sell/flat/hedge) with confidence score.

### Pillar 2 — Divine Intelligence (DI) / Vedic Jyotish Engine
Full Vedic astrology computation engine using Swiss Ephemeris (pyswisseph). 17 celestial bodies pre-computed at 1-minute resolution from 2015–2030. 27 nakshatras × 4 padas, hora calendar, Vimshottari dasha (L1/L2/L3), **Shadbala full 6-component planetary strength** (Sthana, Dig, Kala, Chesta, Naisargika, Drik Bala), eclipse corridors, and 9 apocalypse trigger types. Output: composite DI score with Shadbala modifiers and MANDATORY_FLAT overrides.

### Pillar 3 — Chinese Wisdom Engine
3,000+ years of Chinese market timing integrated into a quantitative framework. Five sub-modules: Feng Shui flying star arrangement + Qi flow mapping, Bazi 4-Pillars trading clock + day officer quality rating, Five Elements seasonal sector rotation, I Ching 64-hexagram market state calculator, **Zi Wei Dou Shu (ZWDS) Purple Star natal chart engine**. Output: CW composite score and full Chinese Wisdom market state classification.

### Pillar 9 — Western Astrology Engine
Tropical zodiac computation using Swiss Ephemeris (ayanamsa = 0; tropical). 10 planets tracked from 2015–2030. Covers: aspect matrix (0°/60°/90°/120°/180° + minor aspects), house transits (Placidus), planetary ingresses (sign changes), retrograde periods (Rx/SD/SR), New/Full Moon + eclipse events, Saturn–Jupiter synodic cycles for long-term market rhythm, and Venus/Mars cycle tracking for medium-term momentum. Western bias scores feed the composite score as an independent factor. **Entirely separate from Pillar 2 (Vedic uses sidereal; Western uses tropical).**

### Pillar 4 — AI-Driven Execution Engine (MT4/MT5 + Crypto Exchanges)
**MT Bridge:** ONNX-native model inference inside MQL5 EA (< 200ms latency). ZeroMQ TCP bridge (< 500ms) for complex architectures. Full order lifecycle. DI + CW + Western Astrology danger filters hardcoded in EA. Circuit breakers on all MANDATORY_FLAT triggers.

**Crypto Exchange Layer (Custom):** PAT signals route directly to Binance, OKX, Crypto.com, Bybit, and KuCoin via a custom-built connector framework. Supports Spot and Perpetual Futures. Each exchange has dedicated REST (order placement) and WebSocket (live feed) connectors with exchange-specific HMAC signing. No third-party exchange abstraction library (ccxt) — fully owned connector code for reliability and auditability. Cross-exchange order routing, position management, funding rate monitoring, and arbitrage alerting built-in.

### Pillar 5 — COT Analytics Engine
CFTC Commitment of Traders data ingested weekly. Commercial net position, non-commercial net, open interest delta, extreme positioning percentile ranking (0–100). Historical percentile above 90 or below 10 triggers contrarian signal adjustment.

### Pillar 6 — Seasonality Engine
Historical seasonal pattern analysis per instrument, timeframe, and calendar period (2015–2024). Win rate, average return, volatility by month / week of year / day of week / trading session. Strong seasonal alignment amplifies signal confidence; counter-seasonal signals flagged for analyst review.

### Pillar 7 — Composite Scoring System
Multi-factor weighted aggregation of all pillar outputs into a single −300 to +300 directional score. Signal lifecycle: draft → validated → published → triggered/expired. Factor decomposition stored in `signal.signal_factors` (one row per factor per signal) for full explainability and audit.

### Pillar 8 — Backtesting & Walk-Forward Validation
Walk-Forward Optimization (WFO) over 2015–2030 data. Train/test split: rolling 3-year train, 1-year test. Out-of-sample 2024 holdout gate. Model rejection criteria enforced: OOS Sharpe < 1.0 auto-rejected. Full cost model: spread + commission + swap + slippage. Results archived to Wasabi S3 with PDF report.

---

## 5. Functional Modules

### 5.1 Identity, Access & Entitlements

| Feature | Specification |
|---------|--------------|
| Registration & Login | Email + password; social OAuth optional; email verification required |
| Multi-Factor Authentication | TOTP mandatory for admin/analyst; optional for subscribers |
| Session & Device Management | JWT-based sessions; revocable API keys with scoped permissions; Valkey session store |
| RBAC | Roles: `superadmin`, `admin`, `analyst`, `subscriber`; all privileged actions RBAC-gated |
| Subscription Entitlements | Plan-level feature flags and usage limits enforce signal tier access |
| Timezone Preference | Per-user IANA timezone stored in `app.users.timezone`; all display times converted via API |
| Admin Impersonation | Audited with full before/after state logging in `audit.audit_events` |

### 5.2 Market Data Layer
- Ingest historical OHLCV bars and tick data from configurable providers (2015–2030 range)
- Live feed adapters with latency-target and freshness-SLA monitoring
- Canonical instrument/symbol master with multi-source alias resolution
- COT report ingestion from CFTC: commercial, non-commercial, open interest with derived metrics
- Seasonality profile computation by instrument, timeframe, and season key
- Macroeconomic event calendar integration with impact-level tagging
- On-chain metric ingestion for crypto instruments
- Data quality checks, replay capability for research and incident analysis

### 5.3 Divine Intelligence (DI) Engine
Full specification in Section 6.

### 5.4 Chinese Wisdom Engine
Full specification in Section 7. Four sub-modules: Feng Shui, Bazi Trading Clock, Five Elements Sector Rotator, I Ching Calculator.

### 5.5 Timezone & Session Engine
Full specification in Section 8. UTC base computation, API-driven client timezone conversion, broker time normalization, solar time for hora and Bazi.

### 5.6 AI Signal Engine
Full specification in Section 9.

### 5.7 AI-Driven MetaTrader Execution Engine
Full specification in Section 10.

### 5.8 Verdict Terminal (Live Dashboard)
- Real-time signal cards: direction, entry price, SL, TP1/TP2/TP3, confidence, DI state, CW state
- Signal reasoning panel: decomposed factor scores, DI state detail, CW hexagram/star state, COT widget, seasonality chart
- Planetary widget: live ephemeris, active nakshatra, hora countdown, contamination alert banner
- Chinese Wisdom widget: active I Ching hexagram, Bazi hour quality, flying star arrangement, Five Elements current phase
- Charting integration (TradingView Lightweight Charts) with signal overlay
- All timestamps displayed in user's local timezone (IANA-based); UTC toggle available
- WebSocket / SSE real-time update channel; dashboard latency < 1 second

### 5.9 Strategy Research & Backtesting
- Strategy family and strategy registry with versioned entry/exit/risk logic (JSONB)
- Backtest runs with full cost model (spread, slippage, rollover, swaps)
- Walk-forward validation over 2015–2030 data window
- Analyst approval workflow: `research → review → approved → production`
- `mql_exportable` flag per strategy; export produces SHA-256 checksummed MQL4/MQL5 package

### 5.10 Notifications & Alerts
- Per-user alert rules: configurable on signal direction, score threshold, instrument, contamination event, CW danger state
- Delivery channels: in-app, email, Telegram, WhatsApp, webhook
- All alert timestamps delivered in user's local timezone
- Quiet hours, suppression rules, per-channel delivery preferences
- HMAC-SHA256 signed webhook endpoints

### 5.12 Crypto Exchange Module
- **Exchange Connectors:** Custom Python connector per exchange (Binance, OKX, Crypto.com, Bybit, KuCoin); base class + exchange-specific implementation
- **Market Data:** Real-time OHLCV, ticker, order book L2, trade stream via WebSocket; REST fallback; feeds `market.price_bars` and `exchange.order_book_snapshots`
- **Order Execution:** Spot market/limit orders; Perpetual futures market/limit/stop orders; order status tracking; fill reconciliation
- **Position Management:** Per-exchange position ledger; P&L tracking; exposure aggregation across exchanges
- **Funding Rate Monitor:** Perpetual funding rate ingestion every 8 hours; alert if rate exceeds `RISK_CRYPTO_FUNDING_RATE_THRESHOLD`
- **Exchange Router:** Routes PAT signals to optimal exchange based on liquidity, fee tier, and available balance
- **Arbitrage Monitor:** Cross-exchange price differential alerts (configurable BPS threshold)
- **Circuit Breakers:** All existing DI/CW/WA MANDATORY_FLAT rules apply; additional crypto-specific breakers (funding rate, liquidation buffer, exchange connectivity)
- **Copy-Trade Prohibition:** Same enforcement as MT Bridge — crypto exchange accounts bound to specific subscriber; no multi-account signal replication

### 5.11 Admin, Audit & Operations
- Admin console: user administration, subscription management, signal review/override, feed health, incident review
- `audit.audit_events`: every privileged action recorded with before/after JSONB diff
- `ops.job_runs`: scheduled pipeline execution with status, input/output payload, error capture
- `ops.service_health_checks`: per-service latency time-series (TimescaleDB hypertable)
- `ops.incidents`: severity-graded with root cause, remediation JSONB, and resolution tracking

---

## 6. Divine Intelligence (DI) Engine — Vedic Jyotish

### 6.1 Celestial Bodies (17)

| ID | Name | Type | Vimshottari Years | Benefic/Malefic |
|----|------|------|-------------------|-----------------|
| 0 | Surya (Sun) | Graha | 6 | Benefic |
| 1 | Chandra (Moon) | Graha | 10 | Benefic |
| 2 | Budha (Mercury) | Graha | 17 | Neutral |
| 3 | Shukra (Venus) | Graha | 20 | Benefic |
| 4 | Mangal (Mars) | Graha | 7 | Malefic |
| 5 | Guru (Jupiter) | Graha | 16 | Benefic |
| 6 | Shani (Saturn) | Graha | 19 | Malefic |
| 7 | Rahu (North Node) | Shadow | 18 | Malefic |
| 8 | Ketu (South Node) | Shadow | 7 | Malefic |
| 9 | Uranus | Outer Planet | — | Neutral |
| 10 | Neptune | Outer Planet | — | Neutral |
| 11 | Pluto | Outer Planet | — | Neutral |
| 12 | Gulika | Upagraha | — | Malefic |
| 13 | Mandi | Upagraha | — | Malefic |
| 14 | Yamakantaka | Upagraha | — | Benefic |
| — | Galactic Center | Special Point (27° Sag) | — | Bullish +85 |
| — | Super Galactic Center | Special Point (1°48' Lib) | — | Bullish +60 |

### 6.2 Key Nakshatra Biases (XAUUSD)

| Nakshatra | Ruler | Bias Weight | Trading Interpretation |
|-----------|-------|-------------|----------------------|
| Pushya | Saturn | +70 | Most auspicious — institutional buying |
| Rohini | Moon | +65 | Wealth accumulation, luxury demand |
| Uttara Phalguni | Sun | +60 | Steady institutional accumulation |
| Ashwini | Ketu | +40 | Fast impulse, half-size only |
| Mula | Ketu | −65 | Root destruction — major corrections |
| Ardra | Rahu | −55 | Market storms, sharp selloffs |
| Jyeshtha | Mercury | −45 | Competitive destruction |
| Purnima (Tithi 15) | Moon/Sun | −80 | Distribution climax — short only |

### 6.3 Hora Weight Table

| Hora Lord | Daily Bias Weight | Action |
|-----------|------------------|--------|
| Jupiter | +45 | Most bullish — preferred long entry |
| Venus | +35 | Luxury demand |
| Mercury | +20 | Mixed — news-driven |
| Sun | +15 | Moderate bullish |
| Moon | +10 | Emotional, light size |
| Mars | −25 | Volatile — tight stops |
| Saturn | −40 | Risk-off — avoid fresh longs |

### 6.4 Contamination Zones

| Zone | Severity | Position Cap | Action |
|------|----------|-------------|--------|
| Rahu Kala | 90 | 0% | MANDATORY_FLAT |
| Eclipse Window (±14 days) | 95 | 0% | MANDATORY_FLAT |
| Gulika Kala | 70 | 25% | REDUCE_SIZE |
| Yamaganda | 65 | 30% | REDUCE_SIZE + manual confirmation |

### 6.5 Apocalypse Triggers (9 Types)

| Trigger Code | Severity | Action | Duration |
|-------------|----------|--------|----------|
| APO_GRAND_CROSS | 100 | MANDATORY_FLAT | 21 days |
| APO_TOTAL_SOLAR_ECLIPSE | 100 | MANDATORY_FLAT | 14 days |
| APO_BLACK_HOLE_CLUSTER | 88 | MANDATORY_FLAT | 7 days |
| APO_RAHU_HORA | 90 | MANDATORY_FLAT | 1 hour |
| APO_ECLIPSE_WINDOW | 75 | MANDATORY_FLAT | 6 days |
| APO_MARS_GANDANTA | 85 | MANDATORY_FLAT | 3 days |
| APO_TRIPLE_RETROGRADE | 80 | MANDATORY_FLAT | 14 days |
| APO_SATURN_URANUS_SQUARE | 70 | REDUCE_SIZE_50% | 30 days |
| APO_PLUTO_STATIONARY | 60 | REDUCE_SIZE_60% | 5 days |

### 6.6 Vimshottari Dasha Biases

| Dasha Lord | Duration | Bias Weight |
|------------|----------|-------------|
| Jupiter | 16 yrs | +70 |
| Venus | 20 yrs | +60 |
| Mercury | 17 yrs | +25 |
| Moon | 10 yrs | +20 |
| Sun | 6 yrs | +15 |
| Mars | 7 yrs | −30 |
| Rahu | 18 yrs | −50 |
| Ketu | 7 yrs | −45 |
| Saturn | 19 yrs | −65 |

### 6.7 Shadbala — Planetary Strength Full Computation (六力)

**Shadbala** ("six strengths" in Sanskrit) is the Vedic framework for quantifying how strong or weak each planet is at any given moment. A strong planet amplifies its significations; a weak planet distorts or fails them. For trading, high-Shadbala benefics add bullish conviction; high-Shadbala malefics add bearish severity; low-Shadbala planets produce confused, unreliable market behaviour.

#### 6.7.1 The Six Components

| Component | Sanskrit | What It Measures | Computed How |
|-----------|----------|-----------------|-------------|
| Sthana Bala | Positional Strength | Planet's dignity in current sign/navamsa | Uccha (exaltation), Mula Trikona, Sva (own sign), Mitra (friendly), Sama (neutral), Shatru (enemy) scores |
| Dig Bala | Directional Strength | Planet's power based on house position | Jupiter/Mercury: strongest in 1st; Sun/Mars: 10th; Moon/Venus: 4th; Saturn: 7th |
| Kala Bala | Temporal Strength | Planet's strength at the current time of day/year | Day/night planets, paksha (lunar phase), hora lord, ayana, masa, dina, hora sub-factors |
| Chesta Bala | Motional Strength | Strength from velocity and direction of motion | Direct, stationary, retrograde — retrograde Rx increases Chesta Bala for outer planets |
| Naisargika Bala | Natural Strength | Inherent permanent strength per planet | Sun=60, Moon=51.4, Venus=42.8, Jupiter=34.2, Mercury=25.7, Mars=17.1, Saturn=8.5 (Virupas) |
| Drik Bala | Aspectual Strength | Net gain/loss from planetary aspects received | Benefic aspects add; malefic aspects subtract; computed per aspect type |

#### 6.7.2 Shadbala Totals & Minimum Required Rupas

| Planet | Minimum Rupas Required | Below Minimum = Weak Signal |
|--------|----------------------|----------------------------|
| Sun | 390 Rupas | DI score reduced 30% |
| Moon | 360 Rupas | DI score reduced 30% |
| Mars | 300 Rupas | DI score reduced 20% |
| Mercury | 420 Rupas | DI score reduced 20% |
| Jupiter | 390 Rupas | DI score reduced 25% |
| Venus | 330 Rupas | DI score reduced 25% |
| Saturn | 300 Rupas | DI score reduced 20% |

#### 6.7.3 Trading Application

- **Bhava Bala (House Strength):** 2nd house (wealth), 5th (speculation), 8th (sudden events), 11th (gains) — house strength modifies position size
- **Ishta/Kashta Phala:** Benefic (Ishta) and malefic (Kashta) power scores quantify the net effect on market sentiment
- **Shadbala modifier on DI Score:** Planets with Rupas ≥ 150% of minimum amplify their nakshatra/dasha weight by +15%; planets below minimum reduce their weight by −30%
- **Pre-computation:** Shadbala pre-computed daily from 2015–2030 and cached in `astro.shadbala_cache`

#### 6.7.4 Database Table

```sql
CREATE TABLE astro.shadbala_cache (
    computed_date     DATE NOT NULL,
    planet_id         SMALLINT NOT NULL,          -- matches celestial_bodies reference
    sthana_bala       NUMERIC(8,3),               -- in Virupas
    dig_bala          NUMERIC(8,3),
    kala_bala         NUMERIC(8,3),
    chesta_bala       NUMERIC(8,3),
    naisargika_bala   NUMERIC(8,3),
    drik_bala         NUMERIC(8,3),
    total_rupas       NUMERIC(8,3),               -- sum / 60 (Virupas → Rupas)
    ishta_phala       NUMERIC(8,3),
    kashta_phala      NUMERIC(8,3),
    is_weak           BOOLEAN,
    di_weight_modifier NUMERIC(5,3),              -- multiplier applied to DI score
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (computed_date, planet_id)
);
-- ~100K rows (7 planets × ~5,478 days 2015–2030)
```

---

### 6.8 Western Astrology Engine (Tropical)

> **Distinct from Vedic (Pillar 2):** Western uses the **tropical zodiac** (aligned to seasons, not stars). Vedic uses the **sidereal zodiac** (star-fixed, offset by ~23–24° Ayanamsa). Both systems run in parallel — each contributes independently to the composite score.

#### 6.8.1 Scope

| Component | Description |
|-----------|-------------|
| **Planetary Positions** | All 10 bodies (Sun, Moon, Mercury, Venus, Mars, Jupiter, Saturn, Uranus, Neptune, Pluto) in tropical degrees 2015–2030 at 1-minute resolution (re-uses Swiss Ephemeris with ayanamsa=0) |
| **Natal/Reference Chart** | Gold market "birth chart" anchored to the 1971 Nixon Shock date (Aug 15, 1971 — end of Bretton Woods gold standard) as the reference chart for transit analysis |
| **Aspect Matrix** | Major: conjunction (0°), sextile (60°), square (90°), trine (120°), opposition (180°); Minor: semi-sextile (30°), quincunx (150°), semi-square (45°), sesquiquadrate (135°) |
| **Orb Settings** | Configurable per aspect type and planet weight; default: major aspects ±8°, minor aspects ±3° |
| **Transits** | Daily transiting planets to natal Gold chart positions; ingress alerts (planet changes tropical sign) |
| **Retrograde Tracking** | Mercury Rx (3×/year ~3 weeks), Venus Rx (~18 months cycle), Mars Rx (~2 years cycle), outer planet Rx annuals; Rx flag published with market caution level |
| **Lunations** | New Moon and Full Moon times + sign positions; Super Moons flagged separately |
| **Eclipse Mapping** | Solar/Lunar eclipse dates + degrees → cross-referenced with natal Gold chart sensitive points |
| **Saturn–Jupiter Cycle** | 20-year conjunction cycle; current cycle ingress 2020 Aquarius → long-term structural signal |
| **Venus Cycle** | 584-day Venus synodic cycle (inferior/superior conjunction with Sun) — bullish gold near inferior conjunction |

#### 6.8.2 Western Aspect Bias Table (XAUUSD)

| Aspect | Planets Involved | Bias | Score |
|--------|-----------------|------|-------|
| Jupiter–Venus trine/sextile | Natal or transit | Strongly Bullish | +25 |
| Jupiter–Sun conjunction | Transit to natal | Bullish growth surge | +20 |
| Saturn–Sun square/opposition | Transit to natal | Bearish pressure | −20 |
| Mars–Pluto conjunction | Transit | Explosive volatility | ±15 (direction-neutral) |
| Mercury Rx begins | Any sign | Uncertainty, false breaks | −10 |
| Venus Superior Conjunction | Any sign | Gold accumulation phase | +15 |
| Saturn–Jupiter conjunction (20-yr) | New cycle start | Long-term structural shift | ±30 |
| Solar Eclipse on natal degree | ±1° orb | Major trend change | MANDATORY_REVIEW |

#### 6.8.3 Database Tables

```sql
CREATE TABLE astro.western_transit_cache (
    ts_utc             TIMESTAMPTZ NOT NULL,
    planet_id          SMALLINT NOT NULL,
    tropical_longitude NUMERIC(10,6),     -- degrees (0–360, tropical)
    tropical_sign      SMALLINT,          -- 1–12
    is_retrograde      BOOLEAN,
    retrograde_phase   VARCHAR(20),       -- direct | rx_shadow | retrograde | station_rx | station_direct
    speed_deg_per_day  NUMERIC(8,5),
    PRIMARY KEY (ts_utc, planet_id)
);
-- TimescaleDB hypertable on ts_utc

CREATE TABLE astro.western_aspects (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ts_utc             TIMESTAMPTZ NOT NULL,
    planet_a           SMALLINT NOT NULL,
    planet_b           SMALLINT NOT NULL,
    aspect_type        VARCHAR(30),        -- conjunction, trine, square, opposition, sextile, etc.
    orb_degrees        NUMERIC(6,3),
    applying           BOOLEAN,            -- TRUE = applying; FALSE = separating
    bias_score         SMALLINT,           -- Western aspect contribution to composite
    created_at         TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE astro.western_events (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_date         DATE NOT NULL,
    event_type         VARCHAR(50),        -- retrograde_start, retrograde_end, ingress, new_moon, full_moon, eclipse, conjunction_cycle
    planet_id          SMALLINT,
    from_sign          SMALLINT,
    to_sign            SMALLINT,
    eclipse_type       VARCHAR(30),
    notes              TEXT,
    western_score      SMALLINT,
    is_mandatory_review BOOLEAN DEFAULT FALSE
);
```

---

## 7. Chinese Wisdom Engine — Feng Shui + Bazi + Five Elements + I Ching + ZWDS

> This section defines the **Chinese Wisdom (CW) Engine** — a five-module system that integrates 3,000+ years of Chinese philosophical and metaphysical frameworks with modern quantitative trading. All CW computations are performed in **UTC** then adjusted to solar time where cosmologically appropriate. Results are stored in the `chinese` database schema and fed as features into the AI Signal Engine composite score.

### 7.1 Module 1 — Feng Shui Flying Stars (玄空飞星)

#### 7.1.1 Concept
Feng Shui's **Flying Stars School (Xuan Kong Fei Xing)** maps 9 energy archetypes (stars 1–9) onto a 3×3 grid (Lo Shu magic square). The central star changes annually, monthly, and daily, creating unique energy patterns at each time scale. Each star has a defined trading bias.

#### 7.1.2 The Nine Stars & Trading Bias

| Star | Element | Name | Trading Bias | Score Contribution |
|------|---------|------|--------------|-------------------|
| Star 1 | Water | Tan Lang (Greedy Wolf) | Opportunity, flow, momentum | +25 |
| Star 2 | Earth | Ju Men (Giant Gate) | Illness, obstacles, blockage | −30 |
| Star 3 | Wood | Lu Cun (Prosperity) | Conflict, volatility, fast moves | +10 (volatile) |
| Star 4 | Wood | Wen Qu (Scholar) | Creative growth, subtle uptrend | +15 |
| Star 5 | Earth | Lian Zhen (Chastity) | **Misfortune — most malefic** | MANDATORY_FLAT / −50 |
| Star 6 | Metal | Wu Qu (Military Power) | Authority, discipline, bullish Metal | +30 |
| Star 7 | Metal | Po Jun (Army Breaker) | Robbery, deception, bear risk | −25 |
| Star 8 | Earth | Zuo Fu (Left Assistant) | **Wealth, prosperity — most auspicious** | +45 |
| Star 9 | Fire | You Bi (Right Assistant) | Future prosperity, celebration, peak | +35 |

> **Star 5 at center grid (annual or monthly):** Triggers CW_MANDATORY_FLAT equivalent. All new positions blocked until star rotates. Combined with DI apocalypse = maximum caution.
> **Star 8 at center:** Strongest bullish bias. Combined with DI nakshatra Pushya = extreme bullish composite.

#### 7.1.3 Flying Star Rotation Schedule

| Cycle | Duration | Trigger |
|-------|----------|---------|
| Annual Star | 1 year (Li Chun — Solar New Year ~Feb 4) | Pre-computed annually |
| Monthly Star | 1 month (Solar terms — every ~30 days) | Pre-computed monthly |
| Daily Star | 1 day (midnight UTC → local solar) | Pre-computed daily |
| Hourly Star | 2-hour block (aligned with Earthly Branch hours) | Computed per trading hour |

#### 7.1.4 Qi Flow Directional Map

Feng Shui maps market session energy to compass directions and elements:

| Direction | Element | Season | Trading Session | Qi Quality |
|-----------|---------|--------|----------------|------------|
| North (子) | Water | Winter | Tokyo session (23:00–08:00 UTC) | Accumulation |
| East (卯) | Wood | Spring | London open (07:00–10:00 UTC) | Growth burst |
| South (午) | Fire | Summer | New York noon (13:00–17:00 UTC) | Peak energy |
| West (酉) | Metal | Autumn | London close (15:00–17:00 UTC) | Distribution |
| Center | Earth | Transition | Session overlaps | Consolidation |

#### 7.1.5 Feng Shui Trading Methodology

| Framework | Principle | Trading Application |
|-----------|-----------|-------------------|
| 寻龙 Xún Lóng (Seeking the Dragon) | Find the main energy vein | Identify primary trend on H4/D1; volume = dragon breath |
| 分金 Fēn Jīn (Dividing Gold) | Mark the acupuncture points | Key S/R levels where Qi accumulates |
| 观缠山 Guān Chán Shān (Tangled Mountain) | Analyse consolidation energy | Triangle/flag patterns = Qi preparing to release |
| 风林火山 Fēng Lín Huǒ Shān (WFFM) | Wind/Forest/Fire/Mountain states | 4-state market energy classifier per session |
| 点穴 Diǎn Xué (Acupuncture Point) | Precise entry on key energy nodes | Entry timing at H1 aligned with auspicious flying star |

---

### 7.2 Module 2 — Bazi Trading Clock (八字交易时钟)

#### 7.2.1 Concept
**Bazi (八字)** — the "Four Pillars of Destiny" — uses the Year, Month, Day, and Hour pillars (each comprising a Heavenly Stem + Earthly Branch) to define the energetic quality of any given moment. For trading, the Day and Hour pillars are most actionable.

#### 7.2.2 Heavenly Stems (天干) — 10 Total

| # | Stem | Pinyin | Element | Polarity | Trading Character |
|---|------|--------|---------|----------|-----------------|
| 1 | 甲 | Jiǎ | Wood | Yang | Strong growth, uptrend initiation |
| 2 | 乙 | Yǐ | Wood | Yin | Gentle growth, steady accumulation |
| 3 | 丙 | Bǐng | Fire | Yang | Explosive momentum, news-driven |
| 4 | 丁 | Dīng | Fire | Yin | Smouldering trend, slow but persistent |
| 5 | 戊 | Wù | Earth | Yang | Consolidation, range-bound |
| 6 | 己 | Jǐ | Earth | Yin | Stagnation, indecision |
| 7 | 庚 | Gēng | Metal | Yang | Decisive breakout, institutional action |
| 8 | 辛 | Xīn | Metal | Yin | Subtle strength, precious metal bullish |
| 9 | 壬 | Rén | Water | Yang | Fast flow, momentum surge |
| 10 | 癸 | Guǐ | Water | Yin | Hidden accumulation, smart money entry |

#### 7.2.3 Earthly Branches (地支) & Two-Hour Trading Blocks

| Branch | Pinyin | Animal | Element | UTC Block | Trading Hour Quality |
|--------|--------|--------|---------|-----------|---------------------|
| 子 | Zǐ | Rat | Yang Water | 23:00–01:00 | Smart money moves, low liquidity |
| 丑 | Chǒu | Ox | Yin Earth | 01:00–03:00 | Accumulation, quiet |
| 寅 | Yín | Tiger | Yang Wood | 03:00–05:00 | Pre-London momentum build |
| 卯 | Mǎo | Rabbit | Yin Wood | 05:00–07:00 | London prep, institutional positioning |
| 辰 | Chén | Dragon | Yang Earth | 07:00–09:00 | **London open — high energy** |
| 巳 | Sì | Snake | Yin Fire | 09:00–11:00 | London mid-session, trending |
| 午 | Wǔ | Horse | Yang Fire | 11:00–13:00 | London/NY overlap build |
| 未 | Wèi | Goat | Yin Earth | 13:00–15:00 | **NY open — peak volume** |
| 申 | Shēn | Monkey | Yang Metal | 15:00–17:00 | London close, distribution |
| 酉 | Yǒu | Rooster | Yin Metal | 17:00–19:00 | NY mid-session, precious metal hour |
| 戌 | Xū | Dog | Yang Earth | 19:00–21:00 | NY wind-down, risk-off |
| 亥 | Hài | Pig | Yin Water | 21:00–23:00 | Tokyo prep, accumulation |

#### 7.2.4 Day Officers (建除十二神) — Daily Trading Classification

The Day Officer is computed from the Earthly Branch of the day pillar relative to the current month branch. It classifies the entire trading day's dominant energy:

| Officer | Chinese | Quality | Trading Action |
|---------|---------|---------|----------------|
| 建 Jiàn | Establish | Excellent | Open new positions; initiate trends |
| 除 Chú | Remove | Good | Close old positions; take profits |
| 满 Mǎn | Full | Caution | High volume; watch for distribution tops |
| 平 Píng | Balance | Neutral | Scalp range; no new swing positions |
| 定 Dìng | Stable | Good | Trend continuation; add to winners |
| 执 Zhí | Grasp | Good | Hold current positions; trail stops |
| 破 Pò | Break | High Risk | Breakout day; tight stops mandatory; high volatility |
| 危 Wēi | Danger | Avoid | Reduce all exposure; avoid new entries |
| 成 Chéng | Success | Excellent | **Strongest bullish officer**; full position size |
| 收 Shōu | Collect | Good | Accumulation; DCA entries welcome |
| 开 Kāi | Open | Good | New opportunities emerge; fresh longs |
| 闭 Bì | Close | Caution | Market closing energy; exit by day-end |

> **Day Officer Score Contributions:**
> 成/建/除 → +20 to composite score
> 定/执/收/开 → +10
> 平/满 → 0
> 破 → −15 (volatile; reduce size)
> 危 → −25 (REDUCE_SIZE_50% equivalent)
> 闭 → −10

#### 7.2.5 Hour Quality Calculation

Hour quality is determined by the interaction of the Day Stem and Hour Branch using the traditional **甲己还甲子 (Jiǎ-Jǐ returns to Jiǎ-Zǐ)** stem-hour cycle. Each hour within the day is rated:

| Hour Rating | Score | Action |
|-------------|-------|--------|
| 吉 Jí (Auspicious) | +15 | Full position size allowed |
| 中 Zhōng (Neutral) | 0 | Standard position size |
| 凶 Xiōng (Inauspicious) | −15 | Reduce size 50% |
| 大凶 Dà Xiōng (Very Inauspicious) | −25 | MANDATORY_FLAT for that hour |

---

### 7.3 Module 3 — Five Elements Sector Rotator (五行板块轮动)

#### 7.3.1 Concept
The **Five Elements (五行 Wǔ Xíng)** — Wood, Fire, Earth, Metal, Water — cycle through the year and between asset classes. When an element is dominant (prosperous season), assets governed by that element tend to outperform. This creates a systematic sector rotation signal aligned with the Chinese cosmic calendar.

#### 7.3.2 The Five Elements & Their Market Domains

| Element | Chinese | Season | Direction | Colour | Asset Classes | Instruments |
|---------|---------|--------|-----------|--------|---------------|-------------|
| 木 Wood | Mù | Spring (Feb–Apr) | East | Green | Tech, Biotech, Agriculture, Media | US100, AUS200, WHEAT, CORN |
| 火 Fire | Huǒ | Summer (May–Jul) | South | Red | Energy, Finance, Entertainment, Defence | USOIL, UKOIL, US500, US30 |
| 土 Earth | Tǔ | Late Summer (Jul–Aug) | Centre | Yellow | Real Estate, Construction, Mining, Food | COPPER, commodity indices |
| 金 Metal | Jīn | Autumn (Aug–Oct) | West | White/Gold | Gold, Silver, Banks, Industrial | **XAUUSD ⭐, XAGUSD**, GER40 |
| 水 Water | Shuǐ | Winter (Nov–Jan) | North | Black/Blue | Crypto, Insurance, Shipping, Liquidity | BTCUSD, ETHUSD, USDJPY (safe-haven) |

#### 7.3.3 Seasonal Element Strength Calendar

| Month | Primary Element | Secondary Element | Gold (Metal) Bias | CW Score |
|-------|-----------------|------------------|-------------------|---------|
| January | Water | Metal | Transitional bullish | +15 |
| February | Wood | Water | Neutral | 0 |
| March | Wood | Fire | Slightly bearish | −5 |
| April | Wood | Fire | Bearish (energy rises) | −10 |
| May | Fire | Earth | Bearish Metal | −15 |
| June | Fire | Earth | Bearish Metal | −20 |
| July | Fire → Earth | Earth | Transition | −5 |
| August | Metal | Earth | **Bullish Gold** ↑↑ | +25 |
| September | Metal | Water | **Bullish Gold** ↑↑↑ | +35 |
| October | Metal | Water | **Bullish Gold** ↑↑ | +25 |
| November | Water | Metal | Strong bullish | +20 |
| December | Water | Wood | Bullish | +15 |

#### 7.3.4 Element Interaction Dynamics

```
GENERATING CYCLE (相生 Xiāng Shēng) — Amplification:
Wood → Fire → Earth → Metal → Water → Wood

CONTROLLING CYCLE (相克 Xiāng Kè) — Suppression:
Wood → Earth → Water → Fire → Metal → Wood

TRADING RULES:
• Dominant element generating Gold's element (Water→Metal) = BUY signal amplifier
• Dominant element controlling Gold's element (Fire→Metal) = SELL/reduce signal
• Same element as Gold (Metal month, Metal day) = Maximum Gold bullish window
• Exhausting element (Metal→Water dominant) = Profit-taking window
```

#### 7.3.5 Five Elements Score Formula

```
CW_FIVE_ELEMENTS_SCORE = (annual_element_weight × 0.3) 
                        + (monthly_element_weight × 0.4) 
                        + (daily_element_weight × 0.3)
```

Range: −25 to +25 contribution to CW composite.

---

### 7.4 Module 4 — I Ching Calculator (易经计算器)

#### 7.4.1 Concept
The **I Ching (易经 Yì Jīng)** — "Book of Changes" — uses a system of 64 hexagrams (六十四卦), each formed by 6 Yin (⚋) or Yang (⚊) lines. Applied to markets, it classifies the current market state into one of 64 archetypal patterns, each with a defined directional bias and risk classification.

#### 7.4.2 Input Methods (Three)

**Method 1 — Candle-Based Hexagram (Primary)**
```
For each of the last 6 closed candles on the signal timeframe:
  Yang (⚊) = close > open (bullish candle)
  Yin  (⚋) = close ≤ open (bearish/doji candle)
Line 1 = most recent candle, Line 6 = oldest candle
→ Upper trigram = Lines 4-6; Lower trigram = Lines 1-3
→ Map to hexagram number (1–64)
```

**Method 2 — Multi-Timeframe State Dashboard**
```
6 timeframes: M5, M15, H1, H4, D1, W1
Each timeframe: Yang if current bar is bullish, Yin if bearish
→ Forms a single state hexagram from timeframe alignment
→ Used as confluence filter, not primary signal
```

**Method 3 — Moving Average Hexagram (Trend Strength)**
```
7 EMAs: 5, 15, 34, 55, 89, 144, 233
6 comparisons: EMA5>EMA15, EMA15>EMA34, EMA34>EMA55, EMA55>EMA89, EMA89>EMA144, EMA144>EMA233
Each: Yang (⚊) = shorter MA above longer; Yin (⚋) = shorter MA below longer
→ Qián hexagram (all Yang) = perfect uptrend alignment
→ Kūn hexagram (all Yin) = perfect downtrend alignment
```

#### 7.4.3 The 64 Hexagrams — Key Trading States

| # | Chinese | Pinyin | Trigrams | Bias | Score | Trading Signal |
|---|---------|--------|----------|------|-------|----------------|
| 1 | 乾 | Qián | Heaven/Heaven | STRONG BULLISH | +40 | Initiate longs; max size |
| 2 | 坤 | Kūn | Earth/Earth | STRONG BEARISH | −40 | Initiate shorts; max size |
| 3 | 屯 | Zhūn | Water/Thunder | CHOPPY/WAIT | −10 | Hold; no new entries |
| 4 | 蒙 | Méng | Mountain/Water | UNCLEAR | −5 | Wait for clarity |
| 7 | 师 | Shī | Earth/Water | BEARISH ARMY | −20 | Short bias; reduce longs |
| 8 | 比 | Bǐ | Water/Earth | SOLIDARITY | +15 | Accumulation phase |
| 11 | 泰 | Tài | Earth/Heaven | PEAK PROSPERITY | +35 | **Strong buy; trend peak** |
| 12 | 否 | Pǐ | Heaven/Earth | STAGNATION | −35 | **Strong sell; avoid** |
| 14 | 大有 | Dà Yǒu | Fire/Heaven | GREAT POSSESSION | +30 | Buy; holding trend |
| 15 | 谦 | Qiān | Earth/Mountain | MODESTY | +10 | Light longs; low volatility |
| 16 | 豫 | Yù | Thunder/Earth | ENTHUSIASM | +20 | Buy momentum |
| 23 | 剥 | Bō | Mountain/Earth | STRIPPING AWAY | −30 | Sell; distribution |
| 24 | 复 | Fù | Earth/Thunder | **RETURN/REVERSAL** | +25 | **Reversal buy signal** |
| 25 | 无妄 | Wú Wàng | Heaven/Thunder | INNOCENCE | +15 | Natural trend; follow |
| 29 | 坎 | Kǎn | Water/Water | **DOUBLE ABYSS** | −45 | **DANGER: MANDATORY_FLAT** |
| 30 | 离 | Lí | Fire/Fire | DOUBLE FIRE/PEAK | −20 | Distribution top; take profits |
| 36 | 明夷 | Míng Yí | Earth/Fire | DARKENING LIGHT | −25 | Bear warning; reduce |
| 39 | 蹇 | Jiǎn | Water/Mountain | OBSTRUCTION | −20 | Avoid trading; obstacles |
| 43 | 夬 | Guài | Lake/Heaven | BREAKTHROUGH | +30 | Breakout imminent; buy |
| 44 | 姤 | Gòu | Heaven/Wind | ENCOUNTER | −15 | False breakout warning |
| 46 | 升 | Shēng | Earth/Wind | RISING | +25 | Steady uptrend; buy |
| 47 | 困 | Kùn | Lake/Water | EXHAUSTION | −30 | Trend exhaustion; take profits |
| 48 | 井 | Jǐng | Water/Wind | THE WELL | +10 | Consistent flow; hold |
| 49 | 革 | Gé | Lake/Fire | REVOLUTION | +25 | **Major reversal signal** |
| 51 | 震 | Zhèn | Thunder/Thunder | DOUBLE THUNDER | 0 | High volatility; reduce size |
| 52 | 艮 | Gèn | Mountain/Mountain | STILLNESS | −5 | Consolidation; range trade |
| 54 | 归妹 | Guī Mèi | Thunder/Lake | MARRYING MAIDEN | −15 | **False signal warning** |
| 56 | 旅 | Lǚ | Fire/Mountain | THE WANDERER | −10 | No clear trend; small size |
| 57 | 巽 | Xùn | Wind/Wind | GENTLE WIND | +15 | Gradual uptrend |
| 58 | 兑 | Duì | Lake/Lake | JOYFUL LAKE | +20 | Overbought delight; caution near tops |
| 63 | 既济 | Jì Jì | Water/Fire | ALREADY COMPLETE | −20 | **Distribution top; exit** |
| 64 | 未济 | Wèi Jì | Fire/Water | NOT YET COMPLETE | +20 | **Accumulation bottom; enter** |

> All 64 hexagrams are stored in `chinese.iching_hexagrams` with full trading metadata. Score range: −45 to +40. Hexagram 29 (Kǎn) triggers `CW_MANDATORY_FLAT` equivalent — no new positions regardless of other signals.

#### 7.4.4 Changing Lines & Transition Analysis

When a hexagram contains "changing lines" (6 or 9 numerical values in yarrow stalk method, or detected via extreme overbought/oversold RSI per bar position):
- The changing line produces a **derived hexagram** (lines flip)
- Derived hexagram reveals the next dominant market state
- Used for signal projection: "current state → transitioning to"
- Stored as `iching_transition_hexagram` in signal factors

#### 7.4.5 Nuclear Hexagram (Inner Structural Analysis)

The **nuclear hexagram** (互卦 Hù Guà) is extracted from lines 2–5 of the primary hexagram:
- Reveals the underlying structure and hidden market dynamics
- Used as a secondary confirmation filter
- If nuclear hexagram contradicts primary: reduce position size 25%

#### 7.4.6 I Ching Score Formula

```
CW_ICHING_SCORE = (primary_hexagram_score × 0.6)
                + (derived_hexagram_score × 0.25)
                + (nuclear_hexagram_score × 0.15)
```

Range: −45 to +40 contribution to CW composite.

---

### 7.5 Chinese Wisdom Composite Score

```
CW_COMPOSITE = (CW_ICHING_SCORE     × 0.35)    # max ±15.75
             + (CW_FENGSHUI_SCORE    × 0.25)    # max ±12.5
             + (CW_BAZI_SCORE        × 0.20)    # max ±10
             + (CW_FIVE_ELEMENTS     × 0.20)    # max ±10

Total CW range: ≈ −50 to +50
```

**Override Rules:**
- Flying Star 5 at annual OR monthly center → `CW_MANDATORY_FLAT` flag set
- I Ching Hexagram 29 (Kǎn double water) detected → `CW_MANDATORY_FLAT` flag set
- Bazi Day Officer 危 (Wēi) → `CW_REDUCE_SIZE_50%` flag set
- Combined DI Apocalypse + CW MANDATORY_FLAT → maximum danger classification; alert all subscribers

---

### 7.5 Module 5 — Zi Wei Dou Shu (紫微斗數) — Purple Star Astrology

#### 7.5.1 Concept

**Zi Wei Dou Shu (ZWDS)** is the most complex and complete system in Chinese astrology, distinct from Bazi. Often called "Purple Star Astrology," it plots 14 major stars + 100+ auxiliary stars across 12 palaces of a natal chart. Originally a court method for destiny analysis in Imperial China, its palace activation cycles provide cycle-based market timing unavailable from any other system.

For Predict-A-Trade, ZWDS is applied to:
1. **Gold market natal chart** — anchored to a reference date (Bretton Woods collapse, Aug 15 1971) as the "birth chart" for the gold market
2. **Daily flow palace computation** — which palace is activated today based on the year/month/day stems and branches
3. **Flying Star crossover** — ZWDS palace interactions with the Feng Shui flying star grid for compound signals

#### 7.5.2 The 12 Palaces

| Palace # | Chinese | Pinyin | English | Market Application |
|----------|---------|--------|---------|-------------------|
| 1 | 命宮 | Mìng Gōng | Life / Self | Core price action, primary trend direction |
| 2 | 兄弟宮 | Xiōngdì Gōng | Siblings | Correlated market moves, pair trading signals |
| 3 | 夫妻宮 | Fūqī Gōng | Spouse | Partnership — USD pairs correlation |
| 4 | 子女宮 | Zǐnǚ Gōng | Children | Generated profits, compound moves |
| 5 | 財帛宮 | Cáibó Gōng | Wealth | **Primary wealth palace — key for gold accumulation timing** |
| 6 | 疾厄宮 | Jí'è Gōng | Health/Risk | Danger periods, sharp reversals |
| 7 | 遷移宮 | Qiānyí Gōng | Travel/Change | Major trend change, capital flow shifts |
| 8 | 交友宮 | Jiāoyǒu Gōng | Friends | Institutional flow, smart money alignment |
| 9 | 官祿宮 | Guānlù Gōng | Career/Status | Long-term structural bias, macro trend |
| 10 | 田宅宮 | Tiánzhái Gōng | Property | Physical asset (gold as property) accumulation |
| 11 | 福德宮 | Fúdé Gōng | Fortune | Lucky/unlucky periods, windfall potential |
| 12 | 父母宮 | Fùmǔ Gōng | Parents | Historical support, generational price memory |

#### 7.5.3 The 14 Major Stars & Trading Significance

| Star | Chinese | Element | Trading Nature | Score Contribution |
|------|---------|---------|---------------|-------------------|
| 紫微 | Zǐ Wēi | Earth | Emperor; stabilising authority | +20 (bullish structure) |
| 天機 | Tiān Jī | Wood | Strategy; fast intelligent moves | +10 (agile, breakouts) |
| 太陽 | Tài Yáng | Fire | Sun; institutional, public bullish | +25 (macro bull) |
| 武曲 | Wǔ Qū | Metal | Military; gold/metal direct bullish | **+35 (strongest gold bull)** |
| 天同 | Tiān Tóng | Water | Peace; sideways, low volatility | 0 (range-bound) |
| 廉貞 | Lián Zhēn | Fire | Danger; deception, sharp reversals | −20 (bear risk) |
| 天府 | Tiān Fǔ | Earth | Treasury; accumulation, wealth | +20 (wealth building) |
| 太陰 | Tài Yīn | Water | Moon; feminine, accumulation | +15 (quiet bull) |
| 貪狼 | Tān Láng | Water/Wood | Greed; speculative explosive | +5 / −5 (both ways) |
| 巨門 | Jù Mén | Water | Darkness; obstacles, reversals | −20 (bearish caution) |
| 天相 | Tiān Xiāng | Water | Minister; regulatory events | −5 (neutral/slight drag) |
| 天梁 | Tiān Liáng | Earth | Old age; safe haven demand | +15 (gold safe haven) |
| 七殺 | Qī Shā | Metal | Seven Kills; violent correction | **−30 (REDUCE_SIZE_70%)** |
| 破軍 | Pò Jūn | Water | Destroyer; destruction of trend | **−25 (trend destruction)** |

> **ZWDS MANDATORY_FLAT trigger:** 七殺 (Seven Kills) or 廉貞 (Lian Zhen) activating the 財帛宮 (Wealth Palace) simultaneously with Flying Star 5 → full CW_MANDATORY_FLAT regardless of other factors.

#### 7.5.4 Daily Palace Activation Flow

Each day, a specific palace is activated based on the current Bazi stems and branches. The activated palace's dominant stars contribute their scores to the daily CW composite:

```
Daily Activated Palace = f(Year Branch, Month Branch, Day Branch, Day Officer)
ZWDS daily score = Sum of activated palace star scores × activation weight
```

Pre-computed daily from 2015–2030 and cached in `chinese.zwds_daily_flow`.

#### 7.5.5 Database Tables

```sql
-- Reference: Gold market natal chart palace configuration
CREATE TABLE chinese.zwds_natal_palaces (
    palace_number      SMALLINT PRIMARY KEY,     -- 1–12
    palace_name_cn     VARCHAR(10),
    palace_name_en     VARCHAR(50),
    primary_star_id    SMALLINT,
    secondary_stars    SMALLINT[],               -- array of star IDs
    natal_score        SMALLINT,                 -- baseline score for this natal config
    notes              TEXT
);

-- Pre-computed daily palace activation
CREATE TABLE chinese.zwds_daily_flow (
    flow_date          DATE PRIMARY KEY,
    activated_palace   SMALLINT NOT NULL,
    active_stars       SMALLINT[],
    daily_zwds_score   SMALLINT,                -- contribution to CW composite
    is_mandatory_flat  BOOLEAN DEFAULT FALSE,
    mandatory_flat_reason TEXT,
    created_at         TIMESTAMPTZ DEFAULT NOW()
);
-- ~5,478 rows (2015–2030)
```

---

## 8. Timezone & Global Session Management Engine

### 8.1 UTC as the Universal Base

All internal computations, database storage, and service communications use **UTC (Coordinated Universal Time)** exclusively. No local timezone is ever stored in the database for time-series data — only UTC timestamps.

```
RULE: All timestamps stored in PostgreSQL use TIMESTAMPTZ (UTC-normalised).
RULE: No TIMESTAMP WITHOUT TIME ZONE columns in any schema.
RULE: All Python datetime objects must be timezone-aware (tzinfo=UTC).
RULE: All API responses include UTC timestamp + user local timestamp pair.
```

### 8.2 Timezone Conversion API

A dedicated timezone conversion layer is exposed via REST API and used by all frontend components and notification systems.

**API Endpoints:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /api/v1/timezone/list` | GET | Returns all IANA timezone names with UTC offsets |
| `GET /api/v1/timezone/current` | GET | Returns user's stored timezone + current local time |
| `POST /api/v1/timezone/convert` | POST | Converts UTC timestamp to target IANA timezone |
| `GET /api/v1/timezone/session-times` | GET | Returns all trading sessions in user's local timezone |
| `PATCH /api/v1/user/timezone` | PATCH | Updates user's preferred timezone |

**Conversion Request/Response Format:**

```json
// POST /api/v1/timezone/convert
{
  "utc_timestamp": "2026-04-05T14:30:00Z",
  "target_timezone": "Asia/Dubai"
}

// Response
{
  "utc": "2026-04-05T14:30:00Z",
  "local": "2026-04-05T18:30:00+04:00",
  "timezone": "Asia/Dubai",
  "timezone_abbr": "GST",
  "utc_offset": "+04:00",
  "is_dst": false
}
```

### 8.3 User Timezone Storage & Propagation

```
app.users.timezone  → IANA timezone string (e.g. "Asia/Dubai", "Europe/London")
                    → Default: "UTC" for new users
                    → Frontend displays timezone selector on first login

Signal display:     → UTC stored, local displayed
Alert delivery:     → UTC stored, local time shown in message body
Hora calendar:      → Computed in UTC, converted to user local for display
Bazi clock:         → Computed in UTC, displayed in user local
Session alerts:     → "London opens at 11:00 AM your time (07:00 UTC)"
```

### 8.4 Solar Time for Astrological Computations

Some DI and CW calculations require **local solar time** rather than timezone-adjusted clock time:

| Computation | Time Basis | Notes |
|-------------|-----------|-------|
| Hora calendar start | Local solar sunrise | Computed per user lat/long or default city |
| Rahu Kala / Gulika Kala | Local solar sunrise | 8 unequal parts of the day |
| Bazi Hour Pillar | Standard Chinese solar time (CST=UTC+8 as base, then solar adjustment) | Use longitude correction |
| Flying Stars (daily) | Midnight local solar | Rotates at solar midnight |
| I Ching candle hexagram | Market timezone (UTC for Forex) | Based on candle close |

**Solar Time API:**
```
GET /api/v1/timezone/solar-time?lat=25.2048&lon=55.2708
→ Returns: sunrise, sunset, solar noon, current solar time for given coordinates
```

### 8.5 Broker Time Normalization

MT5/MT4 broker terminals typically use **GMT+2 or GMT+3** (Eastern European Time with DST). The ZeroMQ bridge normalizes broker timestamps to UTC before any DI/CW lookup:

```python
# Broker time normalization in bridge service
def normalize_broker_time(broker_ts: datetime, broker_offset_hours: int) -> datetime:
    """Convert broker timestamp to UTC for all DI/CW lookups."""
    utc_ts = broker_ts - timedelta(hours=broker_offset_hours)
    return utc_ts.replace(tzinfo=timezone.utc)
```

The broker UTC offset is configurable in `integration.brokers.server_utc_offset` and auto-detected from MT5 on bridge connection.

### 8.6 Trading Session Schedule (UTC Base)

| Session | UTC Open | UTC Close | Primary Currencies | DI/CW Notes |
|---------|----------|-----------|-------------------|------------|
| Sydney | 22:00 | 06:00 | AUD, NZD | Zǐ/Chǒu Bazi hours; Water Qi |
| Tokyo | 00:00 | 09:00 | JPY | Yin Water accumulation |
| Frankfurt | 07:00 | 16:00 | EUR | 辰 Dragon hour open; Wood/Fire |
| London | 08:00 | 17:00 | GBP, EUR | Primary liquidity; peak Qi |
| New York | 13:00 | 22:00 | USD, CAD | 未 Goat hour open; Metal/Fire |
| London/NY Overlap | 13:00 | 17:00 | All | Highest volume; peak composite |

---

## 9. AI Signal Engine Specification

### 9.1 Feature Domains (Extended with CW Layer)

| Domain | Features |
|--------|---------|
| Price & Volume | OHLCV multi-timeframe (M1, M5, H1, H4, D1), ATR, VWAP, volume profile, RSI, MACD, Stochastic |
| Trend & Volatility | EMA crossovers, ADX, Ichimoku cloud, Bollinger Bands, Keltner Channels, realized volatility |
| ICT / SMC | Order blocks, fair value gaps (FVG), break of structure (BOS), market structure shift (MSS), liquidity sweeps, OTE zones |
| DI State | Active nakshatra + pada, hora lord + weight, dasha L1/L2/L3 weights, eclipse corridor flag, contamination score, apocalypse flag, composite DI score |
| Chinese Wisdom | Active I Ching hexagram number + score, flying star center + score, Bazi day officer + hour quality, Five Elements month element + score, CW MANDATORY_FLAT flag, CW composite score |
| COT | Commercial net position, non-commercial net, open interest change, weekly delta, extreme positioning percentile |
| Seasonality | Month/week/day average return, win rate, volatility, confidence score |
| Macro | Economic event proximity, impact level, currency pair sensitivity, FinBERT news sentiment score |
| GLM-4 Reasoning | GLM-4-Flash real-time market assessment text → sentiment score; GLM-4-Long deep analysis → bias label |
| Temporal | Time of day (UTC), day of week, session flags (London/NY open), trading session volatility regime, Bazi hour branch |

### 9.2 Model Training Pipeline

```
Phase 1 — Baseline Models
  └── XGBoost / LightGBM on engineered technical + DI + CW features
  └── Target: directional classification (5-bar, 10-bar, 20-bar forward return)
  └── Establish performance floor (Sharpe > 0.8 before advancing)

Phase 2 — Temporal Deep Learning
  └── LSTM/GRU networks (2 layers, 128–192 units, sequence length 60–100 bars)
  └── Hybrid loss: Directional Accuracy + MSE
  └── Target: beat XGBoost baseline on OOS Sharpe by ≥ 0.2

Phase 3 — Advanced Architectures
  └── LSTM-Transformer hybrid (local noise + long-range dependency capture)
  └── Foundation model evaluation: Chronos, Time-LLM for regime detection
  └── Multi-task: predict direction + volatility regime simultaneously

Phase 4 — GLM-4 Reasoning Integration
  └── GLM-4-Flash: real-time market comment → sentiment feature (< 200ms)
  └── GLM-4-Long: deep DI+CW+COT analysis text → directional bias label
  └── GLM-4V: TradingView chart screenshot → pattern classification
  └── Fused with quantitative features as auxiliary signal layer
```

### 9.3 Signal Factor Groups (Extended)

Each signal in `signal.signal_factors` decomposes into:

```
PRICE_STRUCTURE    — technical entry confirmation (EMA, ICT/SMC patterns)
DI_NAKSHATRA       — active nakshatra weight contribution
DI_HORA            — hora timing weight
DI_DASHA           — dasha multiplier contribution
DI_ECLIPSE         — eclipse corridor adjustment
DI_APOCALYPSE      — override flag if triggered
CW_ICHING          — I Ching hexagram state and score
CW_FENGSHUI        — flying star arrangement and Qi flow score
CW_BAZI            — Bazi day officer + hour quality score
CW_FIVE_ELEMENTS   — Five Elements seasonal rotation score
COT_POSITIONING    — CFTC COT sentiment score
SEASONALITY        — seasonal bias score
MACRO_EVENT        — calendar event adjustment
SENTIMENT_NLP      — FinBERT macro news score
GLM_REASONING      — GLM-4 LLM directional assessment
CONSENSUS          — final arbitrated composite score (−300 to +300)
```

---

## 10. AI-Driven MetaTrader Execution Engine

### 10.1 Architecture Overview

Two execution channels, in priority order:

1. **ONNX Native (Primary):** Trained models exported to ONNX, executed inside MQL5 EA via `OnnxCreate()` / `OnnxRun()`. Target latency: < 200ms. DI and CW state fetched via HTTP call to API or pre-loaded lookup table updated every minute.

2. **ZeroMQ TCP Bridge (Secondary):** Python inference server on port 5555. Receives feature vector from EA, returns `{direction, confidence, sl_pips, tp_pips, di_score, cw_score, contamination_flag, cw_danger_flag}`. Target latency: < 500ms.

### 10.2 Circuit Breakers (Hardcoded in EA)

| Trigger | Action | Reset |
|---------|--------|-------|
| Daily loss > 3% equity | Halt all trading for 24 hours | Manual or next trading day |
| Weekly drawdown > 7% | Halt all trading; alert superadmin | Manual reset |
| DI Apocalypse flag | MANDATORY_FLAT; close all positions | Trigger expiry |
| Eclipse window active | Disable automation; alert only | Corridor end date |
| Rahu Kala active | Block all new entries | Kala window end |
| CW I Ching Hexagram 29 active | MANDATORY_FLAT | Until hexagram changes |
| Flying Star 5 at center (monthly) | MANDATORY_FLAT for instrument class | Monthly rotation |
| Bazi Day Officer 危 active | Reduce all positions 50%; no new entries | Midnight reset |
| Spread > 3× average | Skip execution; requeue signal | Normal spread restored |
| Black swan anomaly (Z > 4σ) | Halt + notify; human review required | Manual reset |
| Bridge disconnected > 60 seconds | Close all open positions; halt | Reconnect + manual |

### 10.3 MQL5 EA Signal Consumption

The EA assembles a feature vector on every new bar and either:
- Runs ONNX inference locally (primary mode)
- Sends to ZeroMQ bridge and receives enriched signal (bridge mode)

The response always includes:
```json
{
  "direction": "buy|sell|flat|hedge",
  "confidence": 0.0–1.0,
  "composite_score": -300 to 300,
  "sl_pips": float,
  "tp1_pips": float,
  "tp2_pips": float,
  "tp3_pips": float,
  "di_score": float,
  "di_contamination_flag": bool,
  "di_apocalypse_flag": bool,
  "cw_score": float,
  "cw_mandatory_flat": bool,
  "cw_hexagram": 1–64,
  "cw_flying_star_center": 1–9,
  "bazi_day_officer": string,
  "bazi_hour_quality": "auspicious|neutral|inauspicious|very_inauspicious",
  "timezone_utc": "ISO8601",
  "user_local_time": "ISO8601"
}
```

---

## 11. Data Engineering & Feature Extraction

### 11.1 Data Sources & Ingestion

| Data Type | Source | Frequency | Storage |
|-----------|--------|-----------|---------|
| OHLCV Bars | MT5 native, Polygon.io, Alpha Vantage | M1 to D1; 2015–2030 | `market.price_bars` |
| Tick Data | MT5 native, broker feed | Real-time + historical | `market.tick_data` |
| COT Reports | CFTC (cftc.gov) | Weekly (Friday) | `market.cot_reports` |
| Macro Calendar | Investing.com API, ForexFactory | Daily | `market.macro_events` |
| News Sentiment | FinBERT + GLM-4 on RSS/NewsAPI | Hourly | `market.news_sentiment` |
| Vedic Ephemeris | Swiss Ephemeris (pyswisseph) | 1-minute; pre-cached | `astro.ephemeris_cache` |
| Bazi Pillars | Custom Bazi engine (lunardate) | Daily/Hourly; pre-cached 2015–2030 | `chinese.bazi_day_cache` |
| Flying Stars | Custom Xuan Kong engine | Annual/Monthly/Daily; pre-cached | `chinese.fengshui_stars_cache` |
| I Ching States | Computed from price bars | Per bar close | `chinese.iching_market_states` |
| Five Elements | Calendar-based computation | Daily; pre-cached | `chinese.five_elements_cache` |
| On-Chain (Crypto) | CoinGecko, Glassnode | Daily | `market.onchain_metrics` |

### 11.2 Pre-Computation Schedule (One-Time + Recurring)

| Task | When | Duration Estimate |
|------|------|------------------|
| Swiss Ephemeris cache (2015–2030) | One-time on setup | ~5 hours |
| Bazi day/hour pillars cache (2015–2030) | One-time on setup | ~30 minutes |
| Flying Stars cache (2015–2030) | One-time on setup | ~10 minutes |
| Five Elements cache (2015–2030) | One-time on setup | ~5 minutes |
| I Ching state computation (historical) | One-time per instrument/TF | ~2 hours per instrument |
| COT historical (2015–present) | One-time on setup | ~1 hour |
| Seasonality computation (2015–2024) | One-time on setup | ~30 minutes per instrument |

---

## 12. Backtesting & Walk-Forward Validation Protocol (2015–2030)

### 12.1 Mandatory Validation Rules

> Standard Train/Test splits are **prohibited**. Only Walk-Forward Optimization (WFO) is permitted.

| Rule | Specification |
|------|--------------|
| Historical Backtest Window | January 2015 – December 2024 (10 years) |
| Future Projection Window | 2025–2030 (live + forward testing) |
| Walk-Forward Method | Rolling window: Train N years, Test 1 year, advance 1 year |
| Out-of-Sample Holdout | Full year 2024 — locked during development; used once for final validation |
| Transaction Cost Model | Historical spreads + commissions + overnight swaps + 0.002% min slippage |
| Minimum OOS Sharpe | > 1.0 to proceed to live deployment |
| Maximum OOS Drawdown | < 15% — hard rejection threshold |
| Profit Factor (after costs) | > 1.3 — hard rejection threshold |

### 12.2 Model Rejection Criteria (Auto-Rejected)

- Directional accuracy > 75% on training data (overfitting)
- OOS Sharpe < 1.0
- OOS Max Drawdown > 15%
- OOS Profit Factor < 1.3 after all costs
- Walk-Forward Efficiency Ratio < 0.5 (OOS Sharpe / IS Sharpe)
- Feature importance dominated by single feature (> 60% contribution)

---

## 13. Risk Management & Controls

### 13.1 Risk Profile Parameters

| Parameter | Default | Notes |
|-----------|---------|-------|
| `max_single_trade_risk_pct` | 1.0% | Kelly-adjusted; max 2% |
| `max_daily_drawdown_pct` | 3.0% | Hard halt |
| `max_weekly_drawdown_pct` | 7.0% | Manual reset required |
| `max_portfolio_heat_pct` | 6.0% | Total open risk across all positions |
| `max_leverage` | 3.0× | Hard cap in EA |
| `news_lockout_minutes` | ±15 min | No new trades around high-impact events |
| `di_contamination_filter` | Enabled | Respect all DI contamination zones |
| `cw_mandatory_flat_filter` | Enabled | Respect all CW mandatory flat triggers |
| `bazi_hour_filter` | Enabled | Skip 大凶 hours; reduce on 凶 hours |

### 13.2 DI + CW Combined Stop-Loss Rules

- Tighten SL by 30–50% during eclipse corridors (DI)
- Tighten SL by 20% during Flying Star 5 active months (CW)
- Widen SL by 50% during high-volatility transits (Mars Gandanta, Saturn stationary)
- No new SL orders during Rahu Kala (avoid whipsaw)
- MANDATORY_FLAT: close all positions on any Apocalypse or CW_MANDATORY_FLAT trigger

---

## 14. Database Architecture — 13 Schemas

### 14.1 Schema Overview

| Schema | Purpose |
|--------|---------|
| `app` | User identity, organisations (multi-tenant), API keys, notification endpoints, RBAC, user timezone; **reseller/white-label tenant config, custom domain mapping** |
| `billing` | Subscription plans, active subscriptions, entitlement ownership; Stripe + crypto (NOWPayments) payment records; **reseller profiles, revenue-share billing, white-label tenant billing** |
| `market` | Instrument master, price bars, tick data, COT, seasonality, macro events, news sentiment, on-chain metrics |
| `astro` | Vedic DI: celestial bodies, zodiac signs, nakshatras, padas, ephemeris cache, aspect events, hora calendar, dasha systems, eclipse events; **Shadbala cache** (6-component strength); **Western astrology**: tropical transit cache, aspect events, retrograde/ingress/lunation events |
| `chinese` | CW Engine: Bazi day/hour pillars cache, flying stars cache, Five Elements cache, I Ching hexagrams reference, I Ching market states per bar; **ZWDS natal palaces, daily flow activation cache** |
| `research` | Microscopic states, contamination matrix, apocalypse triggers, bias catalogue, feature sets, datasets, model families/versions, strategies, backtest runs |
| `signal` | Signal batches, signals, signal factors, alert rules, alert deliveries |
| `risk` | Risk profiles, blackout windows, circuit breakers, portfolio snapshots, positions |
| `execution` | Order routes, bridge sessions, orders, fills, execution events, bridge logs |
| `integration` | Brokers, broker accounts, API credentials, webhook endpoints and deliveries |
| `audit` | Unified audit event log with before/after JSONB diffs |
| `ops` | Job runs, service health checks, incident tracking, GLM health metrics |
| `exchange` | **NEW (v5.0):** Crypto exchange registry, connected accounts, exchange instruments, orders, fills, positions, funding rates, order book snapshots, arbitrage opportunities |

### 14.2 TimescaleDB Hypertables

| Table | Partition Column | Chunk Interval | Est. Rows |
|-------|-----------------|---------------|-----------|
| `market.price_bars` | `ts_open` | 7 days | ~500M (M1 all instruments 2015–2030) |
| `market.tick_data` | `ts_event` | 1 day | ~10B (compressed) |
| `astro.ephemeris_cache` | `ts_observed` | 30 days | ~142M (17 bodies × 1-min × 16 yrs) |
| `astro.aspect_events` | `ts_utc` | 7 days | ~50M (pre-computed 2015–2030) |
| `market.onchain_metrics` | `ts_observed` | 7 days | ~10M |
| `astro.western_transit_cache` | `ts_utc` | 30 days | ~84M (10 planets × 1-min × 16 yrs) |
| `exchange.order_book_snapshots` | `ts_utc` | 1 hour | ~2B (top 20 levels × 5 exchanges × 30 pairs × 1/sec) |
| `exchange.funding_rates` | `ts_utc` | 7 days | ~1M (5 exchanges × 30 perp pairs × 3 per day × 16 yrs) |
| `risk.portfolio_snapshots` | `snapshot_ts` | 1 day | ~1M |
| `ops.service_health_checks` | `checked_at` | 1 day | ~5M |

### 14.3 Chinese Wisdom Schema — Key Tables

```sql
-- chinese.bazi_day_cache
-- Pre-computed 4 Pillars for every day 2015-2030
CREATE TABLE chinese.bazi_day_cache (
    date           DATE PRIMARY KEY,
    year_stem      SMALLINT,      -- 1-10 (Jiǎ to Guǐ)
    year_branch    SMALLINT,      -- 1-12 (Zǐ to Hài)
    month_stem     SMALLINT,
    month_branch   SMALLINT,
    day_stem       SMALLINT,
    day_branch     SMALLINT,
    day_officer    VARCHAR(4),    -- 建/除/满/平/定/执/破/危/成/收/开/闭
    day_element    VARCHAR(10),   -- primary element of the day
    day_score      SMALLINT,      -- CW score contribution from day officer
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

-- chinese.bazi_hour_cache
-- Hour pillar quality for every hour 2015-2030
CREATE TABLE chinese.bazi_hour_cache (
    ts_utc         TIMESTAMPTZ PRIMARY KEY,
    hour_stem      SMALLINT,
    hour_branch    SMALLINT,
    hour_quality   VARCHAR(20),   -- auspicious/neutral/inauspicious/very_inauspicious
    hour_score     SMALLINT,
    bazi_hour_name VARCHAR(10)    -- Chinese hour name
);

-- chinese.fengshui_stars_cache
-- Annual, monthly, daily flying star arrangements
CREATE TABLE chinese.fengshui_stars_cache (
    id             BIGSERIAL PRIMARY KEY,
    cycle_type     VARCHAR(10),   -- annual/monthly/daily/hourly
    period_start   TIMESTAMPTZ,
    period_end     TIMESTAMPTZ,
    center_star    SMALLINT,      -- 1-9
    grid_json      JSONB,         -- 3x3 Lo Shu grid with star positions
    is_dangerous   BOOLEAN,       -- True when Star 5 at center
    cw_score       SMALLINT
);

-- chinese.iching_hexagrams
-- Reference table for all 64 hexagrams
CREATE TABLE chinese.iching_hexagrams (
    hexagram_number SMALLINT PRIMARY KEY,  -- 1-64
    chinese_name    VARCHAR(10),
    pinyin          VARCHAR(30),
    upper_trigram   SMALLINT,    -- 1-8 (Qián to Kūn)
    lower_trigram   SMALLINT,
    bias            VARCHAR(20), -- bullish/bearish/neutral/volatile/danger
    score           SMALLINT,    -- -45 to +40
    mandatory_flat  BOOLEAN,     -- True for hexagram 29
    trading_signal  TEXT,
    description     TEXT,
    changing_to     SMALLINT[],  -- related hexagram numbers
    nuclear_hex     SMALLINT     -- nuclear hexagram number
);

-- chinese.iching_market_states
-- Computed hexagram per instrument/timeframe per bar
CREATE TABLE chinese.iching_market_states (
    id              BIGSERIAL,
    instrument_id   INTEGER REFERENCES market.instruments(id),
    timeframe       VARCHAR(5),
    ts_utc          TIMESTAMPTZ,
    method          VARCHAR(20),  -- candle/multi_tf/moving_avg
    hexagram_number SMALLINT REFERENCES chinese.iching_hexagrams(hexagram_number),
    derived_hex     SMALLINT,
    nuclear_hex     SMALLINT,
    score           SMALLINT,
    mandatory_flat  BOOLEAN
);
SELECT create_hypertable('chinese.iching_market_states', 'ts_utc',
    chunk_time_interval => INTERVAL '7 days');

-- chinese.five_elements_cache
-- Daily Five Elements state
CREATE TABLE chinese.five_elements_cache (
    date              DATE PRIMARY KEY,
    annual_element    VARCHAR(10),
    monthly_element   VARCHAR(10),
    daily_element     VARCHAR(10),
    gold_bias         VARCHAR(20),  -- bullish/bearish/neutral
    annual_score      SMALLINT,
    monthly_score     SMALLINT,
    daily_score       SMALLINT,
    composite_score   SMALLINT
);
```

### 14.4 Exchange Schema — Key Tables (v5.0)

```sql
-- Exchange registry
CREATE TABLE exchange.exchanges (
    id              SMALLINT PRIMARY KEY,
    name            exchange.exchange_name UNIQUE NOT NULL,  -- binance | okx | cryptocom | bybit | kucoin
    display_name    VARCHAR(100),
    rest_base_url   TEXT NOT NULL,
    ws_base_url     TEXT NOT NULL,
    sandbox_rest_url TEXT,
    sandbox_ws_url  TEXT,
    rate_limit_weight_per_min  INTEGER DEFAULT 1200,
    is_active       BOOLEAN DEFAULT TRUE
);

-- Per-subscriber exchange account (API credentials stored in Vault)
CREATE TABLE exchange.exchange_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES app.users(id),
    org_id          UUID REFERENCES app.organisations(id),
    exchange_id     SMALLINT REFERENCES exchange.exchanges(id),
    account_label   VARCHAR(100),                              -- e.g. "Binance Main"
    market_types    exchange.market_type[],                    -- [spot, perpetual]
    vault_key_path  TEXT NOT NULL,                             -- Vault path for API key/secret
    is_sandbox      BOOLEAN DEFAULT FALSE,
    is_active       BOOLEAN DEFAULT TRUE,
    registered_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, exchange_id, is_sandbox)
);

-- Crypto instruments per exchange (trading pairs)
CREATE TABLE exchange.exchange_instruments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_id     SMALLINT REFERENCES exchange.exchanges(id),
    symbol          VARCHAR(30) NOT NULL,                      -- e.g. BTCUSDT
    base_asset      VARCHAR(20) NOT NULL,                      -- BTC
    quote_asset     VARCHAR(20) NOT NULL,                      -- USDT
    market_type     exchange.market_type NOT NULL,
    contract_size   NUMERIC(20,8) DEFAULT 1,
    tick_size       NUMERIC(20,8),
    min_qty         NUMERIC(20,8),
    max_leverage    SMALLINT,
    is_active       BOOLEAN DEFAULT TRUE,
    pat_instrument_id UUID REFERENCES market.instruments(id),  -- link to PAT instrument
    UNIQUE (exchange_id, symbol, market_type)
);

-- Exchange orders (placed by PAT signal → crypto execution)
CREATE TABLE exchange.exchange_orders (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id          UUID REFERENCES exchange.exchange_accounts(id),
    exchange_order_id   VARCHAR(100),                          -- exchange-assigned ID
    signal_id           UUID REFERENCES signal.signals(id),   -- PAT signal that triggered this
    instrument_id       UUID REFERENCES exchange.exchange_instruments(id),
    order_type          exchange.order_type NOT NULL,
    side                exchange.order_side NOT NULL,
    market_type         exchange.market_type NOT NULL,
    position_side       exchange.position_side DEFAULT 'both',
    requested_qty       NUMERIC(20,8) NOT NULL,
    filled_qty          NUMERIC(20,8) DEFAULT 0,
    avg_fill_price      NUMERIC(20,8),
    limit_price         NUMERIC(20,8),
    stop_price          NUMERIC(20,8),
    leverage            SMALLINT DEFAULT 1,
    status              exchange.order_status DEFAULT 'pending',
    client_order_id     VARCHAR(100),                          -- PAT-generated idempotency key
    exchange_response   JSONB,                                 -- raw exchange API response
    placed_at           TIMESTAMPTZ DEFAULT NOW(),
    filled_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- Exchange fills (one order can have multiple partial fills)
CREATE TABLE exchange.exchange_fills (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID REFERENCES exchange.exchange_orders(id),
    exchange_fill_id VARCHAR(100),
    fill_qty        NUMERIC(20,8) NOT NULL,
    fill_price      NUMERIC(20,8) NOT NULL,
    fee             NUMERIC(20,8),
    fee_asset       VARCHAR(20),
    filled_at       TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Current open positions per account
CREATE TABLE exchange.exchange_positions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID REFERENCES exchange.exchange_accounts(id),
    instrument_id   UUID REFERENCES exchange.exchange_instruments(id),
    side            exchange.position_side NOT NULL,
    size            NUMERIC(20,8) NOT NULL,
    entry_price     NUMERIC(20,8) NOT NULL,
    mark_price      NUMERIC(20,8),
    unrealised_pnl  NUMERIC(20,8),
    leverage        SMALLINT DEFAULT 1,
    liquidation_price NUMERIC(20,8),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (account_id, instrument_id, side)
);

-- Perpetual funding rates (TimescaleDB hypertable)
CREATE TABLE exchange.funding_rates (
    ts_utc          TIMESTAMPTZ NOT NULL,
    exchange_id     SMALLINT NOT NULL,
    symbol          VARCHAR(30) NOT NULL,
    funding_rate    NUMERIC(12,8) NOT NULL,                    -- e.g. 0.0001 = 0.01%
    next_funding_ts TIMESTAMPTZ,
    PRIMARY KEY (ts_utc, exchange_id, symbol)
);
-- SELECT create_hypertable('exchange.funding_rates', 'ts_utc', chunk_time_interval => INTERVAL '7 days');

-- Order book snapshots — L2 top 20 levels (TimescaleDB hypertable)
CREATE TABLE exchange.order_book_snapshots (
    ts_utc          TIMESTAMPTZ NOT NULL,
    exchange_id     SMALLINT NOT NULL,
    symbol          VARCHAR(30) NOT NULL,
    bids            JSONB NOT NULL,                            -- [[price, qty], ...]
    asks            JSONB NOT NULL,
    mid_price       NUMERIC(20,8),
    spread_bps      NUMERIC(8,3),
    PRIMARY KEY (ts_utc, exchange_id, symbol)
);
-- SELECT create_hypertable('exchange.order_book_snapshots', 'ts_utc', chunk_time_interval => INTERVAL '1 hour');
-- Retention policy: keep only 24 hours (heavy storage)

-- Cross-exchange arbitrage opportunities
CREATE TABLE exchange.arbitrage_opportunities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    detected_at     TIMESTAMPTZ DEFAULT NOW(),
    base_asset      VARCHAR(20) NOT NULL,                      -- BTC
    buy_exchange_id SMALLINT REFERENCES exchange.exchanges(id),
    sell_exchange_id SMALLINT REFERENCES exchange.exchanges(id),
    buy_price       NUMERIC(20,8) NOT NULL,
    sell_price      NUMERIC(20,8) NOT NULL,
    spread_bps      NUMERIC(8,3) NOT NULL,
    estimated_profit_usd NUMERIC(10,4),
    is_actionable   BOOLEAN DEFAULT FALSE,                     -- exceeds threshold + liquidity available
    acted_on        BOOLEAN DEFAULT FALSE
);
```

### 14.5 Key Enum Types (Extended)

| ENUM | Values |
|------|--------|
| `app.actor_type` | user, service, bot, strategy |
| `billing.subscription_status` | trial, active, past_due, paused, cancelled, expired |
| `billing.payment_method_type` | stripe_card, stripe_bank, crypto_nowpayments |
| `billing.crypto_payment_status` | waiting, confirming, confirmed, sending, finished, failed, refunded, expired |
| `signal.signal_direction` | buy, sell, flat, hedge |
| `signal.signal_status` | draft, validated, published, triggered, expired, cancelled, failed |
| `execution.order_status` | pending, submitted, acknowledged, partially_filled, filled, cancelled, rejected, expired |
| `risk.event_severity` | info, low, medium, high, critical, apocalypse |
| `chinese.bazi_day_officer` | 建, 除, 满, 平, 定, 执, 破, 危, 成, 收, 开, 闭 |
| `chinese.element_type` | wood, fire, earth, metal, water |
| `chinese.hexagram_bias` | strong_bullish, bullish, neutral, bearish, strong_bearish, danger |
| `exchange.exchange_name` | binance, okx, cryptocom, bybit, kucoin |
| `exchange.market_type` | spot, perpetual, futures, options |
| `exchange.order_type` | market, limit, stop_market, stop_limit, take_profit, trailing_stop |
| `exchange.order_side` | buy, sell |
| `exchange.order_status` | pending, open, partially_filled, filled, cancelled, rejected, expired |
| `exchange.position_side` | long, short, both |

---

## 15. Integration Landscape

| Integration | Type | Purpose |
|-------------|------|---------|
| Swiss Ephemeris | C library (pyswisseph) | Vedic planetary positions; 2015–2030 pre-cached |
| Zhipu AI GLM-4 API | REST (https://open.bigmodel.cn) | Real-time signal analysis, chart reasoning, macro sentiment |
| `lunardate` Python library | PyPI | Chinese lunar calendar, Bazi stem/branch calculations |
| `ephem` Python library | PyPI | Astronomical computations (solar time, sunrise/sunset for hora) |
| MT5 Native Data | MetaTrader5 Python package | Historical OHLCV + tick ingestion; live feed |
| Polygon.io / Alpha Vantage | REST API | Supplementary market data 2015+ |
| CFTC COT Reports | REST / CSV download | Weekly positioning intelligence |
| TradingView Lightweight Charts | JavaScript library | Embedded chart in Verdict Terminal |
| Stripe | REST + Webhooks | Subscription lifecycle, card/bank payment events, invoice management |
| NOWPayments | REST + IPN Webhooks | Crypto payment gateway; 100+ coins; BTC/ETH/USDT/USDC/LTC/BNB/SOL/XRP; HMAC-SHA512 IPN verification |
| SendGrid | SMTP / REST API | Transactional email and alert delivery |
| Telegram Bot API | REST | Signal push alerts; CW state notifications |
| WhatsApp Business API | REST | Signal push alerts |
| MetaTrader 4/5 Bridge | ZeroMQ TCP (port 5555) + HTTP | Signal-to-order delivery, fill reconciliation |
| ONNX Runtime | Python + MQL5 native | Cross-layer ultra-low-latency model serving |
| Valkey | TCP (Redis protocol, port 6379) | Cache, sessions, live signal state, job queue, PubSub |
| MLflow | HTTP (self-hosted, port 5000) | Experiment tracking, model registry |
| Grafana + Prometheus | HTTP (self-hosted) | Metrics, dashboards, alerting |
| Loki + Promtail | HTTP (self-hosted) | Log aggregation and search |
| Wasabi S3 | S3-compatible API (rclone/s3cmd) | Model artifacts, MQL exports, backtest reports, DB backups |
| HashiCorp Vault | HTTP (self-hosted) | Secrets management; API key rotation |
| `zoneinfo` / `pytz` | Python stdlib / PyPI | IANA timezone database; UTC↔local conversion |
| TradingView Webhook | REST endpoint (POST /api/v1/tradingview/signals) | Pine Script alert → PAT signal ingestion; HMAC token verification |
| NOWPayments | REST + IPN Webhooks | Crypto payment gateway; 100+ coins; HMAC-SHA512 IPN verification |
| White-label Engine | Multi-tenant FastAPI middleware + per-tenant Valkey namespacing | Row-level org_id isolation; custom domain; branded notifications |
| **Binance** | Custom REST + WebSocket connector | Spot + Futures/Perpetuals; HMAC-SHA256 signing; order execution, live feeds, account management |
| **OKX** | Custom REST + WebSocket connector | Spot + Perpetuals + Options; HMAC-SHA256 + passphrase; `OK-ACCESS-SIGN` header |
| **Crypto.com** | Custom REST + WebSocket connector | Spot + Derivatives; HMAC-SHA256 `SIG` param; digital asset exchange |
| **Bybit** | Custom REST + WebSocket connector | Spot + Perpetuals (USDT/Inverse); HMAC-SHA256; tiered fee structure |
| **KuCoin** | Custom REST + WebSocket connector | Spot + Futures; HMAC-SHA256 + passphrase; dynamic `KC-API-PASSPHRASE` |

---

## 16. Subscription Tier Architecture

### 16.1 The Six Tiers

```
┌────────────────────────────────────────────────────────────────────────┐
│                  PREDICT-A-TRADE SUBSCRIPTION TIERS                    │
├──────────┬──────────┬────────────┬──────────┬────────────┬─────────────┤
│  FREE    │ STARTER  │ EXPLORER   │  ELITE   │ INSTIT.    │ ENTERPRISE  │
│  $0/mo   │ $59/mo   │ $149/mo    │ $299/mo  │ $499/mo    │   Custom    │
│  Trial   │ 3 assets │ 15 assets  │ 40 assets│ All 104    │  All + API  │
└──────────┴──────────┴────────────┴──────────┴────────────┴─────────────┘
```

| Feature | Free | Starter | Explorer | Elite | Institutional | Enterprise |
|---------|------|---------|----------|-------|---------------|------------|
| Instruments | XAUUSD | 3 | 15 | 40 | All 104 | All 104 + custom |
| Timeframes | D1 | H4/D1 | H1/H4/D1 | All TFs | All TFs | All TFs |
| Signals/day | 1 | 5 | 20 | Unlimited | Unlimited | Unlimited |
| DI State (Vedic) | Basic | Partial | Full | Full | Full | Full |
| **Shadbala Strength** | ❌ | ❌ | Summary | Full | Full | Full |
| Chinese Wisdom (CW) | ❌ | I Ching only | I Ching + Bazi | Full CW | Full CW | Full CW |
| **ZWDS Purple Star** | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| **Western Astrology** | ❌ | ❌ | Summary | Full | Full | Full |
| Timezone conversion | UTC only | ✅ | ✅ | ✅ | ✅ | ✅ |
| GLM-4 Analysis | ❌ | ❌ | Summary | Full | Full | Full + custom prompts |
| **TradingView Alerts** | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ |
| MT Bridge | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| ONNX EA Download | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| **White-Label Reseller** | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (Reseller plan) |
| API Access | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Copy Trading | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ (prohibited, all tiers) |
| Dedicated Support | ❌ | ❌ | ❌ | Email | Priority | Dedicated |

### 16.2 Payment Methods — Dual Gateway Architecture

Predict-A-Trade supports two payment pathways for maximum subscriber reach, particularly targeting regions where card payments are difficult or where crypto-native traders prefer on-chain payments.

#### 16.2.1 Stripe (Card / Bank)

| Flow Step | Action |
|-----------|--------|
| Checkout | Stripe Checkout Session created server-side → user redirected |
| Confirmation | Stripe `checkout.session.completed` webhook → entitlement activated |
| Recurring | Stripe `invoice.payment_succeeded` webhook → subscription renewed |
| Failure | Stripe `invoice.payment_failed` → `past_due` status; retry logic |
| Cancellation | Stripe `customer.subscription.deleted` → `cancelled` status |

Stripe webhook endpoint: `POST /api/v1/billing/webhooks/stripe`
Signature verification: `stripe.Webhook.construct_event()` with `STRIPE_WEBHOOK_SECRET`

#### 16.2.2 NOWPayments (Crypto — 100+ Coins)

```
Supported coins (default set):
  BTC, ETH, USDT (ERC-20/TRC-20), USDC, LTC, BNB, SOL, XRP, TRX, DOGE, ADA, MATIC
```

| Flow Step | Action |
|-----------|--------|
| Invoice Creation | `POST /api/v1/billing/crypto/create-invoice` → calls NOWPayments API → returns payment address + amount |
| User Pays | Subscriber sends crypto to generated address |
| IPN Notification | NOWPayments sends `POST /api/v1/billing/webhooks/nowpayments` when payment detected |
| Confirmation | Status transitions: `waiting → confirming → confirmed → finished` |
| Entitlement | On `finished` status + IPN HMAC-SHA512 verified → subscription activated |
| Renewal | New invoice created `CRYPTO_RENEWAL_REMINDER_DAYS` before expiry; Telegram/email reminder sent |
| Expiry | If payment not received within `CRYPTO_PAYMENT_EXPIRY_SECONDS` → invoice expired; new one can be created |

NOWPayments IPN endpoint: `POST /api/v1/billing/webhooks/nowpayments`
Signature verification: HMAC-SHA512 of sorted JSON body with `NOWPAYMENTS_IPN_SECRET`

```python
# IPN verification pattern (required on every crypto webhook)
import hashlib, hmac, json

def verify_nowpayments_ipn(raw_body: bytes, signature_header: str) -> bool:
    sorted_body = json.dumps(json.loads(raw_body), sort_keys=True, separators=(',', ':'))
    expected = hmac.new(
        NOWPAYMENTS_IPN_SECRET.encode(),
        sorted_body.encode(),
        hashlib.sha512
    ).hexdigest()
    return hmac.compare_digest(expected, signature_header)
```

#### 16.2.3 Unified Billing Schema — Key Tables

```sql
-- Unified payment record for both Stripe and NOWPayments
CREATE TABLE billing.payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id     UUID REFERENCES billing.subscriptions(id),
    user_id             UUID REFERENCES app.users(id),
    payment_method      billing.payment_method_type NOT NULL,  -- stripe_card | crypto_nowpayments
    amount_usd          NUMERIC(10,2) NOT NULL,
    -- Stripe fields
    stripe_payment_intent_id  VARCHAR(255),
    stripe_invoice_id         VARCHAR(255),
    -- NOWPayments fields
    nowpayments_payment_id    BIGINT,
    nowpayments_order_id      VARCHAR(100),
    crypto_coin               VARCHAR(20),      -- btc, eth, usdt, etc.
    crypto_amount             NUMERIC(20,8),    -- amount in coin
    crypto_address            VARCHAR(200),
    crypto_status             billing.crypto_payment_status,
    -- Common
    status                    VARCHAR(50) NOT NULL,
    paid_at                   TIMESTAMPTZ,
    expires_at                TIMESTAMPTZ,
    created_at                TIMESTAMPTZ DEFAULT NOW(),
    updated_at                TIMESTAMPTZ DEFAULT NOW()
);

-- Index for IPN lookups
CREATE INDEX idx_payments_nowpayments_id ON billing.payments(nowpayments_payment_id);
CREATE INDEX idx_payments_stripe_intent  ON billing.payments(stripe_payment_intent_id);
```

#### 16.2.4 Payment Gateway Selection Matrix

| Subscriber Region / Preference | Recommended Gateway |
|--------------------------------|---------------------|
| US, EU, UK (card preferred) | Stripe |
| MENA, Asia (crypto-native traders) | NOWPayments |
| Privacy-conscious traders | NOWPayments (USDC/BTC) |
| Institutional clients | Stripe (invoice) or wire transfer |
| Sanctions/restricted countries | NOWPayments (USDT/USDC stablecoin) |

### 16.3 Financial Projections

| Metric | Year 1 | Year 2 | Year 3 |
|--------|--------|--------|--------|
| Total Subscribers | 450 | 1,850 | 5,200 |
| MRR | $28,400 | $142,500 | $462,000 |
| ARR | $340,800 | $1,710,000 | $5,544,000 |
| Gross Margin | 72% | 78% | 82% |
| Break-Even Month | — | Month 19 | Profitable |
| Crypto Payment Share (est.) | 15% | 22% | 28% |

---

## 17. Delivery Phases & Milestones — 30 Development Phases

### Phase Group A — Foundation (Phases 0–5)

| Phase | Name | Key Deliverables |
|-------|------|-----------------|
| 0 | Project Scaffold | Directory structure, `.env` schema, pyproject.toml, base GLM-4 client, Pydantic settings |
| 1 | Server Infrastructure | OS hardening, firewalld, Fail2Ban, SELinux, SSH, NGINX config, systemd unit template |
| 2 | PostgreSQL + TimescaleDB | DB install, 12-schema DDL, Alembic init, hypertable creation, compression policies |
| 3 | Valkey + PgBouncer | Valkey install + config (AOF+RDB), PgBouncer transaction pool, health checks |
| 4 | Miniconda Environments | pat-api, pat-ml, pat-data envs; all dependencies installed and verified |
| 5 | NVM + Next.js Scaffold | NVM install, Next.js 15 init, TailwindCSS config, TradingView widget integration, pnpm setup |

### Phase Group B — Core Backend (Phases 6–8)

| Phase | Name | Key Deliverables |
|-------|------|-----------------|
| 6 | FastAPI Modular Monolith | app structure, all routers, services, models, schemas for 12 modules; OpenAPI docs |
| 7 | Authentication, RBAC & JWT | Registration, login, TOTP MFA, JWT sessions, API keys, RBAC middleware, audit hooks |
| 8 | Database Schemas & Alembic | All 12 schema Alembic migrations; ENUM types; seed scripts for reference data |

### Phase Group C — Intelligence Engines (Phases 9–14)

| Phase | Name | Key Deliverables |
|-------|------|-----------------|
| 9 | Market Data Ingestion (2015–2030) | OHLCV loader, tick ingestion, COT pipeline, instrument master, feed health monitoring |
| 10 | Divine Intelligence (DI) Engine | Swiss Ephemeris sidereal integration, ephemeris pre-population (2015–2030), hora/dasha/eclipse/apocalypse modules, **Shadbala 6-component computation + cache (2015–2030)**, DI composite score |
| 10b | Western Astrology Engine | Swiss Ephemeris tropical (ayanamsa=0), `kerykeion` integration, western_transit_cache (2015–2030), aspect matrix engine, retrograde tracker, ingress/lunation/eclipse events, Western Astrology composite score |
| 11 | Chinese Wisdom Engine | Bazi engine (4 Pillars, day officer, hour quality), Feng Shui flying stars (annual/monthly/daily), Five Elements cycle, I Ching 64-hexagram calculator (3 input methods), **ZWDS natal + daily flow engine**, CW composite score, full `chinese` schema pre-population (2015–2030) |
| 12 | Timezone & Session Engine | UTC base enforcement, `/api/v1/timezone/*` endpoints, user timezone preference, solar time computation, broker time normalization, session schedule API |
| 13 | GLM-4 Model Integration | GLM-4 API client, prompt builder, response parser, timeout/retry/fallback, GLM-4-Flash signal analysis, GLM-4-Long deep analysis, GLM-4V chart pattern recognition |
| 14 | GLM Signal Engine & Composite Scoring | Feature store, signal batch lifecycle, composite score computation (all 15 factor groups), signal arbitration, `signal.signal_factors` explainability |

### Phase Group D — Advanced AI (Phases 15–18)

| Phase | Name | Key Deliverables |
|-------|------|-----------------|
| 15 | Walk-Forward Backtesting Engine | WFO engine (2015–2030), XGBoost + LSTM per window, full cost model, rejection gate, equity curves, PDF report → Wasabi S3 |
| 16 | MT4/MT5 ZeroMQ Bridge | Python bridge service (systemd), ROUTER/DEALER pattern, feature vector serialization, JSON response, reconnection logic |
| 17 | MQL5 Expert Advisor | EA with ONNX inference, ZeroMQ fallback, all circuit breakers (DI + CW), order lifecycle, error handling, SHA-256 packaged |
| 18 | GLM Retraining & Drift Detection | Weekly retraining pipeline, ONNX export, SHA-256 checksum, Wasabi upload, hot-swap via Valkey PubSub, auto-rollback on drift |

### Phase Group E — Customer Application (Phases 19–23)

| Phase | Name | Key Deliverables |
|-------|------|-----------------|
| 19 | Verdict Terminal Frontend | Signal cards, factor reasoning panel, DI + CW widgets, timezone-aware timestamps, TradingView charts, watchlists |
| 20 | WebSocket & Real-Time Delivery | WS server (FastAPI native), SSE fallback, Valkey PubSub → WS bridge, < 1s dashboard latency |
| 21 | Notifications & Alert System | Per-user alert rules, email/Telegram/WhatsApp/webhook delivery, quiet hours, HMAC signing, timezone-localised messages |
| 22 | MQL Export Engine | Template-based MQL4/MQL5 code generation from strategy JSONB, SHA-256 packaging, Wasabi upload, download endpoint |
| 16b | Crypto Exchange Connector | Custom exchange connector base class; Binance/OKX/Crypto.com/Bybit/KuCoin REST + WebSocket implementations; `exchange` schema DDL; HMAC signing per exchange; live ticker/orderbook/trade WebSocket feeds → `market.price_bars` |
| 16c | Crypto Order Execution | PAT signal → crypto order router; market/limit/stop orders; spot + perpetuals; position tracker; fill reconciliation; funding rate monitor; arbitrage alert engine; copy-trade binding enforcement |
| 22b | TradingView Integration | `POST /api/v1/tradingview/signals` webhook receiver; Pine Script alert JSON mapping to PAT signal schema; TV signal de-duplication; signal blending (TV alert + PAT DI/CW score fusion); Verdict Terminal TV alert overlay |
| 23 | Billing — Stripe + Crypto + White-Label | Stripe subscription checkout + webhook handler; NOWPayments crypto invoice + IPN handler; unified entitlement sync; plan upgrades/downgrades; reseller billing setup; white-label tenant provisioning UI |

### Phase Group F — Operations (Phases 24–26)

| Phase | Name | Key Deliverables |
|-------|------|-----------------|
| 24 | Admin Console & Audit System | User management UI, signal review/override, feed health dashboard, `audit.audit_events` viewer, ops incident tracker |
| 25 | Prometheus, Grafana & Loki | Full metrics schema (15+ metric families), Grafana dashboards (trading ops, system health, CW state), Loki log aggregation |
| 26 | Wasabi S3 Backup & Disaster Recovery | Daily WAL archival, weekly pg_dump, model artifact uploads, DR runbook, PITR restore validation drill |

### Phase Group G — Launch Readiness (Phases 27–30)

| Phase | Name | Key Deliverables |
|-------|------|-----------------|
| 27 | Security Hardening & SELinux | SELinux policy review, Vault secrets rotation, MFA enforcement audit, penetration test remediation, OWASP top 10 checklist |
| 28 | Performance Tuning & Load Testing | PostgreSQL + TimescaleDB tuning, Valkey pipeline optimisation, Locust/k6 load tests, P95 API < 200ms confirmed |
| 29 | QA Automation & Integration Tests | Full pytest unit + integration suite, ≥75% coverage gate, E2E tests (Playwright), CI pipeline green on all 13 schemas |
| 30 | Go-Live & Production Runbooks | Release checklist sign-off, operations runbooks, support documentation, Phase 1 production deployment via Agent 10 |

---

## 18. Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| **Availability** | 99.9% uptime; graceful degraded mode on upstream feed failure; systemd auto-restart with exponential backoff |
| **Latency** | Dashboard update < 1s; alert delivery < 2s; signal computation < 500ms; ONNX inference < 200ms; timezone conversion API < 50ms |
| **Security** | MFA for all admin users; RBAC everywhere; TLS in transit (NGINX); AES-256 at rest; HMAC webhook signing; Vault secrets rotation; SELinux enforcing |
| **Auditability** | Every signal traceable: features → model version → DI ruleset → CW state → publication timestamp → delivery status |
| **Timezone Correctness** | All timestamps stored UTC; conversion always via API; user local time displayed consistently across all channels |
| **Scalability** | TimescaleDB hypertables for time-series; Valkey for cache/queue; Wasabi S3 for artifacts; vertical scale first; PgBouncer pooling |
| **Maintainability** | Modular Python packages; versioned APIs; Alembic migrations; automated tests; CI/CD pipeline; structured runbooks |
| **Reliability** | Retry and replay for critical workflows; circuit breakers with manual override; Valkey persistence (AOF + RDB) |
| **CW Engine Correctness** | Bazi pillars computed against validated reference tables; I Ching hexagrams verified against traditional sources; Flying Stars computed against accredited Xuan Kong methodology |
| **Crypto Exchange Reliability** | Exchange connectors must auto-reconnect within 10 seconds; all orders idempotent (client_order_id prevents duplicates); WebSocket feeds must failover to REST polling if WS disconnects; exchange API errors must not lose order state |
| **Crypto Security** | Exchange API keys stored exclusively in Vault (never in DB or env); per-exchange HMAC signing implemented per official API spec; all crypto orders require registered account binding (anti-copy-trade) |
| **Backup** | Daily WAL → Wasabi S3; weekly base backup; monthly archive; 30-day PITR; quarterly DR drill |

---

## 19. Performance KPIs & Acceptance Criteria

### 19.1 System Performance KPIs

| Metric | Target |
|--------|--------|
| API P95 Response Time | < 200ms |
| Timezone Conversion API | < 50ms |
| WebSocket Signal Push Latency | < 1 second |
| Alert Delivery (all channels) | < 2 seconds |
| ONNX Inference (signal → order) | < 200ms |
| ZeroMQ Bridge Latency | < 500ms |
| Crypto Exchange Order Round-Trip | < 500ms (signal → exchange order placed → confirmed) |
| Exchange WebSocket Feed Latency | < 100ms (ticker update → PAT price bar) |
| Crypto Arbitrage Alert Latency | < 1 second from differential detection |
| Database Query P95 (indexed) | < 50ms |
| Ephemeris Computation (cached) | < 10ms |
| CW State Lookup (cached) | < 5ms |
| I Ching Hexagram Computation | < 20ms |
| GLM-4-Flash Signal Analysis | < 200ms |

### 19.2 AI Model KPIs

| Metric | Target |
|--------|--------|
| Directional Accuracy (OOS) | 60% – 70% |
| Sharpe Ratio (OOS, after costs) | > 1.5 (minimum 1.0) |
| Maximum Drawdown (OOS) | < 15% |
| Profit Factor (OOS, after costs) | > 1.3 |
| Walk-Forward Efficiency Ratio | > 0.5 |
| CW Feature Contribution (SHAP) | 5% – 20% of total feature importance |

### 19.3 Acceptance Criteria

1. Architecture and implementation plan approved before Phase 9 begins
2. Core services deployed to staging as systemd units with Prometheus monitoring
3. All 13 schemas deployed and seeded; Alembic head verified
4. DI ephemeris pre-cached 2015–2030 at 1-minute resolution (~142M rows confirmed)
5. CW caches pre-populated 2015–2030: Bazi pillars, flying stars, Five Elements (~5M rows)
6. I Ching hexagram reference table loaded (64 rows with full trading metadata)
7. Timezone API functional; all 600+ IANA zones tested; UTC base enforced throughout
8. Signals generated with all 15 factor groups via REST APIs and Verdict Terminal
9. Alert delivery < 2s across all channels with timezone-localised timestamps
10. MQL5 EA with ONNX inference < 200ms in paper-trading environment
11. WFO backtest (2015–2024) + OOS 2024 holdout completed with passing metrics
12. MQL4/MQL5 exports SHA-256 verified and downloadable
13. Wasabi S3 backup tested; PITR restore validated in DR drill
14. Security controls: MFA, RBAC, TLS, SELinux, Vault, audit logging confirmed
15. Release checklist signed off before go-live

---

## 20. Agent Ecosystem — 10 Automation Agents

Full specification for each agent is in `/srv/sites/predictatrade.com/agents/`.

| # | Agent | Type | Trigger | Saves/Week |
|---|-------|------|---------|-----------|
| 1 | Claude Code Development | Interactive CLI | Developer-initiated | 30 hrs |
| 2 | CI/CD Pipeline | GitHub Actions | Every git push | 10 hrs |
| 3 | Code Review Bot | GitHub Actions + Claude API | Every PR | 8 hrs |
| 4 | Database Migration Agent | GitHub Actions | `models.py` changes | 6 hrs |
| 5 | Test Generation Agent | GitHub Actions + Claude API | New `.py` files | 12 hrs |
| 6 | Documentation Sync Agent | GitHub Actions | Merge to `main` | 4 hrs |
| 7 | GLM + CW Monitor Agent | systemd + Python | Hourly | 10 hrs |
| 8 | Data Pipeline Quality Agent | systemd + Python | Every 15 min | 5 hrs |
| 9 | Security Scanning Agent | GitHub Actions | PR + nightly | 8 hrs |
| 10 | Deployment & Rollback Agent | GitHub Actions + SSH | Tag push + merge | 10 hrs |
| **TOTAL** | | | | **103 hrs/week** |

### Agent Collaboration Flow

```
[Agent 1: Claude Code CLI]
    │ code generated → git push
    ▼
GitHub Repository
  ├──► [Agent 2: CI/CD]         lint → unit → integration → frontend build
  ├──► [Agent 3: Code Review]   DI engine correctness + CW logic + security
  ├──► [Agent 4: Migration]     12-schema drift detection + validation
  ├──► [Agent 5: Test Gen]      auto-generate stubs for new CW/DI modules
  └──► [Agent 9: Security]      CVE + SAST + GLM API key detection

    │ merge to develop
    ▼
[Agent 6: Docs Sync]   OpenAPI → docs site
[Agent 10: Deploy]     staging → health check → production (human tag)

Production Server
  ├──► [Agent 7: GLM+CW Monitor]   hourly: GLM accuracy + CW state validation
  └──► [Agent 8: Data Quality]     15-min: DI cache + CW cache + price feed freshness
```

---

## 21. Bare-Metal Deployment Overview

### 21.1 Server Requirements

| Resource | Staging | Production |
|----------|---------|-----------|
| CPU | 8 vCPU | 32 vCPU |
| RAM | 32 GB | 128 GB |
| NVMe SSD | 500 GB | 4 TB |
| Network | 1 Gbps | 10 Gbps |
| OS | Alma Linux 10 x86_64 | Alma Linux 10 x86_64 |

### 21.2 Key Installation Sequence

```
1.  OS base hardening (SELinux enforcing, firewalld, SSH, Fail2Ban)
2.  Application user (predictatrade) + directory structure
3.  PostgreSQL 16 + TimescaleDB install + 12-schema DDL
4.  PgBouncer connection pooler
5.  Valkey install + configuration (AOF + RDB)
6.  Miniconda + 3 Python environments (pat-api, pat-ml, pat-data)
7.  NVM + Node.js LTS v22
8.  NGINX reverse proxy + TLS (Let's Encrypt)
9.  HashiCorp Vault self-hosted
10. MLflow self-hosted
11. Prometheus + Grafana + Loki + Promtail
12. Wasabi S3 rclone configuration + backup timers
13. All 7 predictatrade systemd services
14. Alembic migrations → head
15. Seed scripts: instruments, nakshatras, hexagrams, apocalypse triggers, subscription plans
16. Pre-population: Swiss Ephemeris cache + Bazi cache + Flying Stars cache (one-time, ~6 hrs)
17. Historical data ingestion: OHLCV 2015–2030 (multi-day process)
18. COT historical load + seasonality computation
19. Model training (XGBoost baseline) + ONNX export + Wasabi upload
20. ZeroMQ bridge service startup + MT5 EA deployment + paper-trading validation
```

### 21.3 Wasabi S3 Buckets

| Bucket | Contents |
|--------|---------|
| `predictatrade-backups` | PostgreSQL WAL + full dumps + PITR archives |
| `predictatrade-models` | ONNX model artifacts + MLflow artifacts |
| `predictatrade-mql-exports` | SHA-256 packaged MQL4/MQL5 EA bundles |
| `predictatrade-backtest-reports` | WFO PDF reports + equity curves |
| `predictatrade-logs` | Archived systemd journal exports (monthly) |

---

## 22. Assumptions & Exclusions

### 22.1 Assumptions

- Dedicated bare-metal server or VPS with Alma Linux 10 is provisioned before Phase 1
- Swiss Ephemeris commercial license obtained for production deployment
- Zhipu AI GLM-4 API key obtained (production quota confirmed)
- MT5 terminal environment available (Windows VPS or Wine on Linux)
- Market data provider API keys obtained (Polygon.io or Alpha Vantage)
- Stripe, SendGrid, Telegram Bot, WhatsApp Business API credentials in place
- Wasabi S3 account and 5 buckets pre-created
- SSL certificate plan confirmed (Let's Encrypt recommended)
- GitHub repository created with Actions enabled (2,000+ free minutes/month)

### 22.1 Updated Assumptions (v4.0)

- `kerykeion` Python library licensed/installed for Western astrology
- ZWDS natal chart reference configuration validated by a qualified Chinese astrologer
- Shadbala computation validated against reference texts (Brihat Parashara Hora Shastra)
- NOWPayments account created, IPN secret configured, test payments validated
- White-label: custom domain SSL provisioning plan agreed (wildcard cert or per-tenant cert via ACME)
- TradingView Premium account available for webhook alert testing

### 22.2 Exclusions

- Native iOS or Android mobile applications
- Broker-dealer licensing or regulated financial advice
- Sub-millisecond HFT co-location
- Cloud services (AWS, GCP, Azure) — bare-metal only
- Docker / Kubernetes — systemd only
- Traditional Feng Shui compass (Luo Pan) physical device integration
- Real-time Bazi natal chart reading for individual traders
- Copy trading (prohibited — see Section 25)

---

## 23. White-Label Multi-Tenant Reseller Architecture

### 23.1 Overview

White-label resellers operate the Predict-A-Trade platform under their own brand. Each reseller (organisation) gets:

- **Custom domain** — e.g. `signals.partner.com` → maps to PAT backend via NGINX `server_name`
- **Custom branding** — logo, colour scheme, sender name stored in `app.org_branding`
- **Branded notifications** — SendGrid `from_name` and `from_email` overridden per org
- **Custom Telegram bot** — reseller provides their own bot token (optional)
- **Sub-subscriber management** — reseller admin manages their subscriber base
- **Revenue sharing** — reseller sets retail price; PAT charges wholesale; difference = reseller margin
- **Plan configuration** — reseller chooses which PAT tiers to expose to their subscribers (with markup)
- **API rate limits** — per-reseller quota configuration

### 23.2 Multi-Tenancy Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    NGINX REVERSE PROXY                      │
│  predictatrade.com → tenant_id = NULL (direct)              │
│  signals.partner1.com → tenant_id = org_001                 │
│  pro.partner2.io → tenant_id = org_002                      │
└─────────────────────────────────────────────────────────────┘
                          │
                  X-Tenant-ID header injected
                          │
┌─────────────────────────────────────────────────────────────┐
│              FastAPI Multi-Tenant Middleware                │
│  1. Resolve domain → org_id via app.domain_mappings         │
│  2. Inject request.state.org_id into all downstream calls   │
│  3. All DB queries automatically scoped with org_id filter  │
│  4. Valkey keys namespaced: "{org_id}:{key}"                │
└─────────────────────────────────────────────────────────────┘
```

**Row-Level Isolation:** All `app.users`, `billing.subscriptions`, `signal.alert_rules` rows include `org_id UUID` column. Platform admin (org_id = NULL) sees all; reseller admin sees only their org.

### 23.3 Database Tables (White-Label)

```sql
CREATE TABLE app.organisations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(200) NOT NULL,
    slug            VARCHAR(100) UNIQUE NOT NULL,
    plan            VARCHAR(50) DEFAULT 'reseller',   -- reseller | enterprise
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE app.domain_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID REFERENCES app.organisations(id),
    domain          VARCHAR(255) UNIQUE NOT NULL,     -- e.g. signals.partner.com
    ssl_cert_path   TEXT,
    is_primary      BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE app.org_branding (
    org_id          UUID PRIMARY KEY REFERENCES app.organisations(id),
    display_name    VARCHAR(200),
    logo_url        TEXT,
    primary_color   VARCHAR(7),                       -- hex e.g. #3B82F6
    secondary_color VARCHAR(7),
    sendgrid_from_email   VARCHAR(200),
    sendgrid_from_name    VARCHAR(200),
    telegram_bot_token    TEXT,                       -- encrypted via Vault
    telegram_bot_username VARCHAR(100),
    custom_css      TEXT,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE billing.reseller_plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID REFERENCES app.organisations(id),
    pat_tier        VARCHAR(50) NOT NULL,              -- starter | explorer | elite | institutional
    wholesale_price_usd NUMERIC(10,2),                -- what PAT charges the reseller
    reseller_retail_usd NUMERIC(10,2),                -- what reseller charges subscriber
    max_subscribers INTEGER,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### 23.4 White-Label Env Variables

```ini
MULTITENANCY_ENABLED=true
WHITE_LABEL_ENABLED=true
WHITE_LABEL_DEFAULT_ORG_ID=          # NULL = PAT direct
WHITE_LABEL_DOMAIN_RESOLVER=nginx_header   # nginx_header | host_header
```

### 23.5 systemd Service

No separate service needed — the FastAPI multi-tenant middleware runs within `predictatrade-api`. NGINX handles domain routing.

### 23.6 Reseller Tier

Added to subscription tiers as a special `RESELLER` plan:

| Feature | Reseller Plan |
|---------|--------------|
| Price | Custom (wholesale negotiated) |
| Sub-subscribers | Up to 5,000 |
| Custom domain | ✅ |
| Custom branding | ✅ Full |
| Revenue sharing | ✅ |
| All signal features | ✅ (passes through Elite) |
| API Access | ✅ |
| Copy Trading | ❌ Prohibited |

---

## 24. TradingView Integration

### 24.1 Purpose

Allow subscribers on Explorer tier and above to receive TradingView Pine Script alerts and have them:
1. Ingested by Predict-A-Trade as external signals
2. Enriched with DI + CW + Western astrology context
3. Scored and published through the normal PAT signal pipeline
4. Optionally triggering MT Bridge execution (Elite+ tier)

### 24.2 Webhook Architecture

```
TradingView Alert (Pine Script) → HTTPS POST → /api/v1/tradingview/signals
                                                         │
                                    ┌────────────────────┴──────────────────────┐
                                    │ 1. Verify HMAC token (X-TV-Secret header) │
                                    │ 2. Parse TV alert JSON payload            │
                                    │ 3. Map to PAT signal schema               │
                                    │ 4. Enrich with current DI/CW/WA scores    │
                                    │ 5. Compute blended composite score        │
                                    │ 6. Publish via standard signal pipeline   │
                                    └───────────────────────────────────────────┘
```

### 24.3 TradingView Alert Payload Contract

TradingView Pine Script sends a webhook with this JSON body (user configures in TV alert):

```json
{
  "tv_secret": "{{strategy.order.comment}}",
  "symbol": "XAUUSD",
  "action": "buy",
  "price": 2345.50,
  "timeframe": "H1",
  "strategy_name": "MyGoldStrategy",
  "user_api_key": "pat_xxxxxxxx",
  "score_override": null,
  "comment": "{{strategy.order.comment}}"
}
```

**Required fields:** `tv_secret`, `symbol`, `action` (buy/sell/flat), `user_api_key`

### 24.4 Signal Blending Rules

| TV Alert Action | PAT DI/CW Agreement | Blended Outcome |
|-----------------|---------------------|-----------------|
| BUY | DI+CW both positive | Strong BUY published |
| BUY | DI or CW negative | BUY published with CAUTION flag |
| BUY | Both MANDATORY_FLAT | TV signal blocked; FLAT published |
| SELL | DI+CW both negative | Strong SELL published |
| Any | CW_MANDATORY_FLAT active | All TV signals blocked |

### 24.5 Env Variables

```ini
TRADINGVIEW_WEBHOOK_ENABLED=true
TRADINGVIEW_WEBHOOK_SECRET=FILL_IN_TV_WEBHOOK_SECRET   # Subscriber-specific HMAC token
TRADINGVIEW_SIGNAL_BLEND_MODE=enrich    # enrich | override | block_if_conflict
TRADINGVIEW_MAX_SIGNALS_PER_HOUR=60     # Rate limit per subscriber
```

### 24.6 Security

- Every TV webhook request verified with HMAC token (user-specific secret, not a shared secret)
- Rate limiting: `TRADINGVIEW_MAX_SIGNALS_PER_HOUR` per `user_api_key`
- `user_api_key` must belong to an Explorer+ subscriber; Starter/Free tier blocked
- No copy-trading allowed: TV signal is attributed to the sending subscriber only; cannot be forwarded

---

## 25. Copy-Trade Prohibition Policy

### 25.1 Definition

**Copy-trading** in the context of this platform means any mechanism by which:
- A subscriber shares, retransmits, or resells signals generated by Predict-A-Trade to third parties who are not paying subscribers on the platform
- A subscriber uses the API or MT Bridge to automatically replicate trades into accounts not owned by that subscriber
- A subscriber builds a third-party signal service using PAT signals as the source

### 25.2 Technical Enforcement

| Control | Implementation |
|---------|---------------|
| **Signal Watermarking** | Every signal carries `subscriber_id` and a cryptographic `signal_fingerprint` (HMAC of signal content + subscriber key). Any leaked signal can be traced to the originating account |
| **API Key Scoping** | API keys are scoped to a single `user_id` and single `org_id`. Keys cannot be shared across accounts |
| **MT Bridge Account Binding** | The ZeroMQ bridge session is bound to a specific MT5 account number registered during subscriber onboarding. Signals to unregistered account numbers are rejected |
| **Rate Limiting** | API endpoints enforce per-user rate limits to prevent bulk signal harvesting |
| **Audit Logging** | Every signal delivery and API call logged to `audit.audit_events` with IP, user agent, and timestamp |
| **TOS Enforcement** | Copy-trading violates Terms of Service; automated detection triggers account suspension |

### 25.3 Detection Rules (Automated)

```python
# Runs as a daily Celery task
COPY_TRADE_DETECTION_RULES = [
    {
        "name": "api_bulk_harvest",
        "rule": "SELECT user_id FROM audit.api_calls WHERE endpoint LIKE '/api/v1/signals%' GROUP BY user_id HAVING count(*) > 500 AND DATE(created_at) = CURRENT_DATE",
        "action": "flag_for_review"
    },
    {
        "name": "multiple_mt5_accounts",
        "rule": "SELECT user_id FROM execution.bridge_sessions GROUP BY user_id HAVING count(DISTINCT mt5_account_number) > 2",
        "action": "flag_for_review"
    },
    {
        "name": "leaked_signal_fingerprint",
        "rule": "Cross-check signal_fingerprint in integration.webhook_deliveries against known subscriber pool",
        "action": "immediate_suspension"
    }
]
```

### 25.4 Reseller Exception

White-label resellers are explicitly **permitted** to:
- Operate a branded version of the platform for their own subscribers
- Bill their subscribers directly and set their own pricing

White-label resellers are explicitly **prohibited** from:
- Using PAT signals in their own external signal services without a separate licensing agreement
- Exporting raw signals to any system outside the PAT platform architecture

---

## 26. Abbreviations & Glossary

See dedicated file: **`ABBREVIATIONS.md`** in the project root.

---

---

## 27. Crypto Exchange Integration — Custom Automated Solution

### 27.1 Design Philosophy

> **No third-party exchange abstraction library (e.g. ccxt).** The PAT Crypto Exchange Layer is a fully custom-built connector framework. Every exchange connector is written directly against that exchange's official API documentation. This gives us:
> - Full control over rate-limit handling, reconnection logic, and error recovery
> - Exchange-specific optimisations (e.g. Binance weight system vs OKX rate limits)
> - Deterministic audit trail — every request/response logged to `exchange.exchange_orders`
> - Zero dependency on third-party library maintenance cycles

### 27.2 Supported Exchanges — v5.0

| Exchange | Markets | REST Base | WebSocket Base | Auth Method | Priority |
|----------|---------|-----------|---------------|-------------|----------|
| **Binance** | Spot, USDT-M Futures, COIN-M Futures | `https://api.binance.com` | `wss://stream.binance.com:9443` | HMAC-SHA256 on query string | Primary |
| **OKX** | Spot, Perpetuals, Options | `https://www.okx.com` | `wss://ws.okx.com:8443/ws/v5` | HMAC-SHA256, `OK-ACCESS-SIGN` header + passphrase | Secondary |
| **Crypto.com** | Spot, Derivatives | `https://api.crypto.com/v2` | `wss://stream.crypto.com/v2/market` | HMAC-SHA256 `sig` param | Tertiary |
| **Bybit** | Spot, USDT Perpetual, Inverse Perpetual | `https://api.bybit.com` | `wss://stream.bybit.com/v5/public` | HMAC-SHA256, `X-BAPI-SIGN` header | Quaternary |
| **KuCoin** | Spot, Futures | `https://api.kucoin.com` | Dynamic token-based WS URL | HMAC-SHA256 + passphrase, `KC-API-SIGN` | Quinary |

### 27.3 Connector Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│              PAT CRYPTO EXCHANGE LAYER                          │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │           ExchangeConnectorBase (abstract)                │  │
│  │  + place_order()  + cancel_order()  + get_position()      │  │
│  │  + subscribe_ticker()  + subscribe_orderbook()            │  │
│  │  + sign_request()  + handle_rate_limit()  + reconnect()   │  │
│  └────────────┬──────────────────────────────────────────────┘  │
│               │ (inherits)                                      │
│    ┌──────────┼──────────┬──────────────┬─────────────────┐     │
│    ▼          ▼          ▼              ▼                  ▼    │
│  Binance    OKX    CryptoCom        Bybit             KuCoin    │
│  Connector  Conn   Connector        Connector         Connector │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   ExchangeRouter                        │    │
│  │  - Routes PAT signals to optimal exchange               │    │
│  │  - Load balances across exchanges                       │    │
│  │  - Failover: if primary fails → secondary               │    │
│  │  - Per-exchange rate limit budget management            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                OrderBookAggregator                      │    │
│  │  - Merges L2 order books from all active exchanges      │    │
│  │  - Computes mid-price, spread, and depth per level      │    │
│  │  - Publishes aggregated book via Valkey PubSub          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                ArbitrageMonitor                         │    │
│  │  - Cross-exchange price differential computation        │    │
│  │  - Alert if spread > CRYPTO_ARBITRAGE_THRESHOLD_BPS     │    │
│  │  - Stores opportunities in exchange.arbitrage_ops       │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 27.4 Exchange-Specific Authentication

#### Binance
```
Signature = HMAC-SHA256(secret_key, query_string + body)
Headers: X-MBX-APIKEY: {api_key}
Query:   timestamp={ms_utc}&signature={sig}
Rate limit: 1200 weight/min (REST); 5 messages/sec (WebSocket)
```

#### OKX
```
Signature = HMAC-SHA256(secret_key, timestamp + method + request_path + body)
Headers:
  OK-ACCESS-KEY: {api_key}
  OK-ACCESS-SIGN: {base64(sig)}
  OK-ACCESS-TIMESTAMP: {ISO_8601_UTC}
  OK-ACCESS-PASSPHRASE: {passphrase}
Rate limit: varies per endpoint (typically 20/2s per endpoint)
```

#### Crypto.com
```
Signature = HMAC-SHA256(secret_key, method + id + api_key + sorted_params + nonce)
Param: sig={hex_sig}
Rate limit: 100 req/sec (private); 100 req/sec (public)
WebSocket: JSON-RPC 2.0 protocol; heartbeat required every 30s
```

#### Bybit
```
Signature = HMAC-SHA256(secret_key, timestamp + api_key + recv_window + sorted_params)
Headers:
  X-BAPI-API-KEY: {api_key}
  X-BAPI-SIGN: {hex_sig}
  X-BAPI-TIMESTAMP: {ms_utc}
  X-BAPI-RECV-WINDOW: 5000
Rate limit: varies by endpoint tier (typically 10-20/sec)
```

#### KuCoin
```
Signature = HMAC-SHA256(secret_key, timestamp + method + endpoint + body)
Headers:
  KC-API-KEY: {api_key}
  KC-API-SIGN: {base64(sig)}
  KC-API-TIMESTAMP: {ms_utc}
  KC-API-PASSPHRASE: HMAC-SHA256(secret_key, passphrase)  ← KC passphrase is itself signed
  KC-API-KEY-VERSION: 2
WebSocket: Requires dynamic token from REST `POST /api/v1/bullet-public`
Rate limit: 1800 req/min (private)
```

### 27.5 WebSocket Feed Architecture

Each connector maintains two persistent WebSocket connections:
1. **Public channel** — ticker, kline (OHLCV), order book, trades (no auth required)
2. **Private channel** — order updates, position updates, account balance (auth required)

Reconnection logic (standard across all connectors):
```
Disconnect detected → immediate reconnect attempt
If reconnect fails → exponential backoff (1s, 2s, 4s, 8s, 16s, max 60s)
After 3 minutes of failure → REST polling fallback activated
After 10 minutes of failure → CRITICAL alert → Telegram ops chat
If > 60 minutes disconnected → circuit breaker: BLOCK all new crypto orders
```

### 27.6 PAT Signal → Crypto Order Flow

```
PAT Signal Published (composite_score > threshold)
        │
        ▼
Crypto Order Router receives signal
        │
        ├── Check CW_MANDATORY_FLAT / DI Apocalypse → if active: BLOCK
        ├── Check RISK_CRYPTO_MAX_LEVERAGE → if exceeded: REDUCE
        ├── Check account balance on target exchange
        ├── Check funding rate (perpetuals) → if > threshold: WARN / SKIP
        ├── Check exchange connectivity (all connectors healthy?)
        │
        ▼
Optimal exchange selected (by liquidity + fee tier + balance)
        │
        ▼
Order placed via REST API (with idempotency client_order_id = signal_id + account_id)
        │
        ├── Success → fill recorded in exchange.exchange_fills
        │            → position updated in exchange.exchange_positions
        │            → signal.signals status updated to 'triggered'
        │
        └── Failure → retry once with same client_order_id (idempotent)
                    → if still fails → try secondary exchange
                    → log to audit.audit_events
                    → alert ops team via Telegram
```

### 27.7 Crypto-Specific Circuit Breakers

These supplement the existing DI/CW/WA circuit breakers for crypto execution:

| Circuit Breaker | Trigger Condition | Action |
|----------------|------------------|--------|
| `EXCHANGE_DISCONNECTED` | WS + REST both failing > 10 min | Block all crypto orders |
| `FUNDING_RATE_HIGH` | Perpetual funding rate > 0.1% (hourly equiv.) | Close long positions; block new longs |
| `LIQUIDATION_BUFFER` | Position mark-to-market within 20% of liquidation price | Immediate partial close |
| `EXCHANGE_RATE_LIMIT` | Hitting 80% of exchange rate limit budget | Throttle order frequency |
| `ORDERBOOK_THIN` | Top 5 levels depth < min threshold | Block orders (slippage risk) |
| `ARBITRAGE_SPREAD_EXTREME` | Spread > 2% between primary and secondary | Pause routing; investigate |

### 27.8 Supported Crypto Instruments (Initial Set)

**Spot (30 pairs):**
BTC/USDT, ETH/USDT, SOL/USDT, BNB/USDT, XRP/USDT, ADA/USDT, LTC/USDT, DOGE/USDT, MATIC/USDT, AVAX/USDT, DOT/USDT, LINK/USDT, UNI/USDT, ATOM/USDT, NEAR/USDT, FIL/USDT, APT/USDT, ARB/USDT, OP/USDT, INJ/USDT + 10 more

**Perpetual Futures (20 contracts):**
BTCUSDT-PERP, ETHUSDT-PERP, SOLUSDT-PERP, BNBUSDT-PERP, XRPUSDT-PERP + 15 major alts

All instruments apply the full PAT DI + CW + Western Astrology scoring. The Five Elements engine already includes Water (crypto/liquidity) and Metal (BTC/gold) sector analysis from v4.0.

### 27.9 Crypto Risk Parameters

| Parameter | Default | Override in `.env` |
|-----------|---------|-------------------|
| Max leverage (perpetuals) | 5× | `RISK_CRYPTO_MAX_LEVERAGE` |
| Max single crypto position (% portfolio) | 10% | `RISK_CRYPTO_MAX_POSITION_PCT` |
| Max crypto portfolio heat | 25% | `RISK_CRYPTO_MAX_PORTFOLIO_HEAT_PCT` |
| Funding rate exit threshold | 0.1%/hr | `RISK_CRYPTO_FUNDING_RATE_THRESHOLD` |
| Liquidation buffer | 20% | `RISK_CRYPTO_LIQUIDATION_BUFFER_PCT` |
| Arbitrage alert threshold | 50 bps | `CRYPTO_ARBITRAGE_THRESHOLD_BPS` |
| Order book min depth (USD) | $50,000 | `RISK_CRYPTO_MIN_ORDERBOOK_DEPTH_USD` |

### 27.10 Subscription Tier — Crypto Exchange Access

| Feature | Free | Starter | Explorer | Elite | Institutional |
|---------|------|---------|----------|-------|---------------|
| Crypto signals (read-only) | XAUUSD only | BTC, ETH | 20 pairs | All 50 pairs | All 50 + custom |
| Crypto order execution | ❌ | ❌ | ❌ | ✅ (1 exchange) | ✅ (all 5 exchanges) |
| Funding rate monitoring | ❌ | ❌ | ✅ (view only) | ✅ | ✅ |
| Arbitrage alerts | ❌ | ❌ | ❌ | ✅ | ✅ |
| Order book depth widget | ❌ | ❌ | ✅ | ✅ | ✅ |
| Multi-exchange routing | ❌ | ❌ | ❌ | ❌ | ✅ |
| Custom exchange priority | ❌ | ❌ | ❌ | ❌ | ✅ |

---

---

## Section 28 — Technical Indicators & ICT/SMC Signal Engine

> **Reference Source:** `reference/PREDICT_A_TRADE_EMA_INDICATORS.md` (v3.0.0-Production)
> **Integration:** Feeds into the existing 15-factor composite scorer (Section 14) as the technical analysis pillar.
> **Phase:** Phase 9 (market data ingestion extended), Phase 14 (Signal Engine enhancements), Phase 14b (ICT/SMC Engine).

---

### 28.1 Indicators Library — Full Implementation Scope

All indicators are computed via a custom Python engine (`app/modules/indicators/`). TA-Lib is used as the underlying calculation library where available. Every indicator is individually configurable (see Section 28.1.7).

#### 28.1.1 Trend Indicators

**Exponential Moving Averages (EMA)**

| Indicator | Period | Role | Priority |
|-----------|--------|------|----------|
| EMA 9 | Short-term | Trigger line — signal generation | Critical |
| EMA 21 | Medium-term | Entry zone — pullback target | Critical |
| EMA 50 | Trend definition | Directional bias filter | Critical |
| EMA 100 | Intermediate trend | Secondary confirmation | High |
| EMA 200 | Long-term trend | Major HTF trend filter | High |

Full EMA computation scope:
- Real-time EMA calculation per tick and per candle close
- EMA slope calculation (angle in degrees)
- EMA ribbon gap measurement between adjacent EMAs
- EMA crossover detection: 9/21, 21/50, 50/200
- ATR-normalized EMA distance measurement
- EMA alignment score: 0 (tangled) → 3 (perfectly stacked)
- EMA flattening detection (trend exhaustion signal)
- EMA convergence/divergence pattern recognition
- Multi-timeframe EMA alignment: H4/D1 context
- EMA as dynamic support/resistance level tracking

**Simple Moving Averages (SMA):** SMA 20, 50, 100, 200

**Additional Trend:** Supertrend (10, 3.0), Parabolic SAR, ADX (14), DMI+/DMI− (14), TRIX (15), KAMA, VWAP (session-based), Ichimoku Cloud (9, 26, 52), Hull MA (20)

#### 28.1.2 Momentum Indicators

RSI (14), RSI (7), Stochastic (%K:14/%D:3), StochRSI (14), MACD (12/26/9 and 5/35/5), CCI (20), Williams %R (14), Momentum (10), ROC (12), TSI (25/13), Awesome Oscillator (5/34), DeMarker (14)

**RSI Full Scope:** Regular and hidden divergence (both bullish and bearish), range-bound detection (40–60), trend detection (>60 bullish / <40 bearish), multi-TF alignment

**MACD Full Scope:** Histogram expansion/contraction, zero-line crossover, signal line crossover, histogram divergence, multi-TF alignment

#### 28.1.3 Volatility Indicators

ATR (14), ATR (7), Bollinger Bands (20, StdDev: 1.0/2.0/3.0), Keltner Channels (20, ATR: 1.5), Donchian Channels (20), Standard Deviation (20), Chaikin Volatility (10), Mass Index

**BB Full Scope:** BB Width squeeze detection, BB %B, BB Walk detection, W-bottom/M-top patterns, TTM Squeeze (BB inside Keltner), dynamic S/R levels

**ATR Full Scope:** ATR ratio (volatility regime classifier), ATR-adjusted position sizing, ATR trailing stop, ATR-based spread filter, multi-TF ATR comparison

#### 28.1.4 Volume Indicators

Volume (raw spike detection), OBV, VWAP (session), Volume Profile (POC/VAH/VAL), Volume Delta, CMF (20), MFI (14), A/D Line, VWMA (20), Ease of Movement (14)

Volume spike threshold: > 2× 20-period average; volume divergence detection; session-relative comparison

#### 28.1.5 Support & Resistance Indicators

Pivot Points (Standard/Fibonacci/Woodie/Camarilla), Fibonacci Retracement (0.382, 0.500, 0.618, 0.786), Fibonacci Extension (1.272, 1.414, 1.618, 2.000), Fibonacci Projection, Horizontal S/R (swing detection), Dynamic S/R (EMA 50/100/200), Order Block Zones, Fair Value Gaps

#### 28.1.6 Composite & Oscillator Indicators

TTM Squeeze (BB + Keltner), Connors RSI, KST, Ultimate Oscillator (7/14/28), Aroon (25), Chande Momentum (14), Fisher Transform (10)

#### 28.1.7 Indicator Configuration Matrix

Every indicator is individually configurable via: `enabled` (bool), `period` (int), `source` (enum: open/high/low/close/hl2/hlc3/ohlc4), `timeframes` (list), `weight_in_ai` (float 0.0–1.0), `threshold_overbought` (float), `threshold_oversold` (float), `divergence_lookback` (int). Stored in `research.indicator_config` table.

---

### 28.2 ICT/SMC Concepts — Full Implementation Scope

Stored in `research.ict_structures`, `research.order_blocks`, `research.fair_value_gaps`, `research.liquidity_levels`.

#### 28.2.1 Market Structure

| Concept | Abbreviation | Description |
|---------|-------------|-------------|
| Break of Structure | BOS | Price closes beyond previous swing high/low in trend direction |
| Market Structure Shift | MSS | First reversal signal against current trend |
| Change of Character | CHoCH | First BOS against prevailing trend |
| Internal / External Structure | — | Lower-degree vs. higher-degree structure mapping |

State machine: `BULLISH_TREND ↔ RANGING ↔ BEARISH_TREND ↔ TRANSITIONING`

Structure implementation: configurable swing detection (left bars: 2–5, right bars: 2–5); multi-timeframe mapping; close-based vs. wick-based confirmation; nested structure D1→H4→H1→M15→M5→M1

#### 28.2.2 Order Blocks (OB)

| Type | Description |
|------|-------------|
| Bullish OB+ | Last bearish candle before bullish BOS/MSS |
| Bearish OB− | Last bullish candle before bearish BOS/MSS |
| Unmitigated OB | Not yet touched by price — higher probability |
| Mitigated OB | Touched/entered by price (0%–100% fill tracked) |
| Breaker Block | Failed OB inverted to opposite S/R |

Implementation: OB zone stored (high/low/midpoint/TF/instrument/timestamp); mitigation percentage tracking; lifetime management (expiry after N bars); overlap detection for high confluence; OB ranking by freshness, size, TF, mitigation status

#### 28.2.3 Fair Value Gaps (FVG)

Bullish FVG: 3-candle pattern (candle 1 high < candle 3 low). Bearish FVG: 3-candle pattern (candle 1 low > candle 3 high). Consequent Encroachment (CE): 50% midpoint — primary entry target.

Minimum gap size: > 0.5 × ATR (configurable). FVG + OB overlap detection = highest confluence setup.

#### 28.2.4 Liquidity

| Concept | Description |
|---------|-------------|
| Buy-Side Liquidity (BSL) | Swing high cluster — buy stops pooled |
| Sell-Side Liquidity (SSL) | Swing low cluster — sell stops pooled |
| Liquidity Sweep | Price takes stops, then reverses with confirmation |
| Equal Highs/Lows | 2+ highs/lows within configurable pip tolerance |
| Liquidity Pool | Dense cluster at similar price levels |
| Liquidity Void | Rapid price movement area (often aligns with FVG) |

#### 28.2.5 Optimal Trade Entry (OTE)

OTE Zone: 0.618–0.786 Fibonacci retracement of a defined swing. OTE + OB or OTE + FVG overlap = highest-probability entry. Discount OTE (bullish buy) / Premium OTE (bearish sell).

#### 28.2.6 Session-Specific ICT Concepts (Killzones)

| Zone | Time (UTC) | Strategy Focus |
|------|-----------|----------------|
| Asian Killzone | 00:00–06:00 | Range accumulation |
| London Killzone | 07:00–10:00 | Sweep-and-reverse plays |
| NY Killzone | 12:00–15:00 | AMD cycle completion |
| Silver Bullet | 10:00–11:00 | High-probability post-London window |
| Judas Swing | Pre-session | Asian range false move before London direction |
| AMD Cycle | Full session | Accumulation → Manipulation → Distribution |

#### 28.2.7 ICT Confluence Scoring System

| ICT Element | Max Score |
|-------------|-----------|
| BOS/MSS in trade direction | 20 |
| Unmitigated OB at entry | 15 |
| FVG at entry zone | 10 |
| OTE zone active | 15 |
| Liquidity sweep confirmed | 15 |
| Premium/Discount aligned | 10 |
| Displacement present | 10 |
| Killzone active | 10 |
| Higher TF alignment | 10 |
| Session type match | 5 |
| **Maximum possible** | **120** |

Score thresholds: ≥80 = HIGH CONFIDENCE (full size); 60–79 = MEDIUM (reduced size); 40–59 = monitoring only; <40 = no trade.

---

### 28.3 Multi-Session Engine

| Session | Open (UTC) | Close (UTC) | Characteristics |
|---------|-----------|-------------|-----------------|
| Asian | 00:00 | 06:00 | Range-bound, accumulation, low volatility |
| Asian–London Transition | 06:00 | 08:00 | Judas swing window, false moves |
| London | 08:00 | 12:00 | Breakout, trend initiation, high volatility |
| London–NY Overlap | 12:00 | 16:00 | Highest volume, strongest directional moves |
| New York | 12:00 | 20:00 | Continuation, potential reversal, news-driven |
| Off-Session | 20:00 | 00:00 | Very low volume — avoid trading |

Session risk parameters are individually configurable per session (max trades, risk%, min R:R, ATR threshold, max spread, primary strategy type).

---

### 28.4 AI Signal Scoring — 70 Feature Vector

The 15-factor PAT composite scorer (Section 14) is extended with a dedicated ML-based technical scoring module using a 70-feature vector:

| Feature Category | Feature Count | Examples |
|-----------------|--------------|---------|
| Trend (EMA/ADX) | 15 | EMA alignment score, EMA 9–21 distance, ADX value, HTF alignment |
| Momentum | 12 | RSI (14/7), MACD histogram, Stochastic, CCI, TSI |
| Volatility | 8 | ATR (14/7), BB Width, BB %B, TTM Squeeze active |
| Volume | 6 | Volume ratio, OBV trend, VWAP position, CMF, MFI |
| ICT/SMC | 15 | Structure direction, fresh BOS, OB mitigation %, FVG at entry, OTE zone, ICT confluence score (0–120) |
| Session & Context | 8 | Session type, killzone active, spread/ATR ratio, Asian range size |
| Historical Performance | 6 | Win rate (last 20), avg R:R, current drawdown, consecutive losses |

**ML Model:** XGBoost Classifier. Input: 70 features. Output: probability of profitable trade (0.0–1.0). Retraining: weekly on rolling 90-day window.

**Fallback rule-based scorer:** weighted formula (Trend 25%, ICT 25%, Momentum 15%, Volatility 10%, Volume 10%, Session 10%, Context 5%).

**Confidence classification:**

| Score Range | Classification | Position Multiplier |
|-------------|---------------|---------------------|
| 85–100 | STRONG BUY/SELL | 1.00× |
| 70–84 | BUY/SELL | 0.75× |
| 55–69 | MODERATE | 0.50× |
| 40–54 | WEAK | 0.25× |
| 0–39 | NO TRADE | 0× |

---

### 28.5 Strategy Engine Architecture

```
StrategyBase (abstract)
├── EmaTrendStrategy       — Trend following, EMA pullback, EMA crossover (9/21)
├── IctBreakoutStrategy    — Asian range breakout, killzone breakout, structure break
├── IctReversalStrategy    — Liquidity sweep reversal, Judas swing, Silver Bullet
├── IctContinuationStrategy— OTE pullback, OB retest, FVG retest
├── RangeStrategy          — Asian range, mean reversion (BB + RSI)
└── CompositeStrategy      — EMA + ICT confluence, multi-TF alignment, session transition
```

**Stop-loss priority:** (1) Structure-based → (2) OB-based → (3) ATR-based (1.5×) → (4) Fixed pip fallback

**Take-profit levels:** TP1 = 30% position (opposite FVG / first structural target); TP2 = 30% (nearest liquidity level); TP3 = 40% (2.0× ATR or Fibonacci extension)

**Trailing stop:** activates after TP1; trail distance = 1.0× ATR; SL moves to breakeven at TP1

---

### 28.6 Risk Management Integration

Technical indicator risk rules supplement the existing PAT risk engine (Section 9):

| Check | Rule | Action |
|-------|------|--------|
| Daily account risk | < 5% of account | Reject trade |
| Drawdown kill switch | ≥ 20% drawdown | Halt all trading |
| Max open trades | < 5 concurrent | Reject trade |
| Spread filter | < session maximum (1.5–2.5 pips) | Reject trade |
| News filter | No high-impact news within 30 min | Reject trade |
| AI score minimum | ≥ 40 (configurable) | Required to execute |
| ICT confluence minimum | ≥ 60 (configurable) | Required to execute |
| R:R minimum | ≥ 1:1.5 (configurable) | Reject if below |

**Position sizing formula:**
```
Lot Size = (Account Balance × Risk% × Confidence Multiplier × Drawdown Multiplier)
           ─────────────────────────────────────────────────────────────────────────
                               (SL in pips × Pip Value)
```

---

### 28.7 Database Tables (Indicator & ICT Engine)

All indicator data stored in the `research` and `signal` schemas:

```sql
-- Indicator snapshots (TimescaleDB hypertable)
CREATE TABLE research.indicator_snapshots (
    id              BIGSERIAL,
    instrument_id   INTEGER NOT NULL REFERENCES market.instruments(id),
    timeframe       VARCHAR(8) NOT NULL,
    ts_utc          TIMESTAMPTZ NOT NULL,
    indicator_name  VARCHAR(50) NOT NULL,
    indicator_value NUMERIC(18, 6),
    metadata        JSONB DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
SELECT create_hypertable('research.indicator_snapshots', 'ts_utc', chunk_time_interval => INTERVAL '1 day');

-- ICT structures
CREATE TABLE research.ict_structures (
    id              BIGSERIAL PRIMARY KEY,
    instrument_id   INTEGER NOT NULL REFERENCES market.instruments(id),
    timeframe       VARCHAR(8) NOT NULL,
    structure_type  VARCHAR(20) NOT NULL,  -- BOS | MSS | CHoCH | RANGING
    direction       SMALLINT NOT NULL,     -- 1=bullish -1=bearish 0=ranging
    swing_high      NUMERIC(18, 6),
    swing_low       NUMERIC(18, 6),
    confirmed_at    TIMESTAMPTZ NOT NULL,
    invalidated_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Order Blocks
CREATE TABLE research.order_blocks (
    id              BIGSERIAL PRIMARY KEY,
    instrument_id   INTEGER NOT NULL REFERENCES market.instruments(id),
    timeframe       VARCHAR(8) NOT NULL,
    ob_type         VARCHAR(20) NOT NULL,  -- bullish | bearish | breaker
    zone_high       NUMERIC(18, 6) NOT NULL,
    zone_low        NUMERIC(18, 6) NOT NULL,
    midpoint        NUMERIC(18, 6) NOT NULL,
    mitigation_pct  NUMERIC(5, 2) DEFAULT 0,
    is_mitigated    BOOLEAN DEFAULT FALSE,
    is_unmitigated  BOOLEAN GENERATED ALWAYS AS (NOT is_mitigated) STORED,
    formed_at       TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Fair Value Gaps
CREATE TABLE research.fair_value_gaps (
    id              BIGSERIAL PRIMARY KEY,
    instrument_id   INTEGER NOT NULL REFERENCES market.instruments(id),
    timeframe       VARCHAR(8) NOT NULL,
    fvg_type        VARCHAR(10) NOT NULL,   -- bullish | bearish
    zone_high       NUMERIC(18, 6) NOT NULL,
    zone_low        NUMERIC(18, 6) NOT NULL,
    ce_level        NUMERIC(18, 6) NOT NULL, -- 50% midpoint
    fill_pct        NUMERIC(5, 2) DEFAULT 0,
    is_filled       BOOLEAN DEFAULT FALSE,
    formed_at       TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Liquidity levels
CREATE TABLE research.liquidity_levels (
    id              BIGSERIAL PRIMARY KEY,
    instrument_id   INTEGER NOT NULL REFERENCES market.instruments(id),
    timeframe       VARCHAR(8) NOT NULL,
    liq_type        VARCHAR(10) NOT NULL,   -- BSL | SSL
    price_level     NUMERIC(18, 6) NOT NULL,
    touch_count     SMALLINT DEFAULT 1,
    is_swept        BOOLEAN DEFAULT FALSE,
    swept_at        TIMESTAMPTZ,
    formed_at       TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

---

### 28.8 Phase Allocation for Indicator & ICT Engine

| Phase | Module | Deliverable |
|-------|--------|-------------|
| Phase 9b | Indicator Engine | EMA (9/21/50/100/200), SMA, ATR, RSI, MACD, BB, Stochastic, Volume indicators — all 30+ indicators computed and stored in `research.indicator_snapshots` |
| Phase 9c | ICT/SMC Engine | BOS/MSS/CHoCH detection, OB zones, FVG detection, liquidity mapping, OTE zones — stored in `research.ict_structures`, `research.order_blocks`, `research.fair_value_gaps`, `research.liquidity_levels` |
| Phase 9d | Multi-Session Engine | Session detection, killzone activation, Judas swing logic, AMD cycle tracking, session-specific risk parameters |
| Phase 14b | AI Technical Scorer | 70-feature vector extraction, XGBoost model integration, confidence classification, adaptive weight adjustment |
| Phase 14c | Strategy Engine | EmaTrendStrategy, IctBreakoutStrategy, IctReversalStrategy, IctContinuationStrategy, RangeStrategy, CompositeStrategy |

---

## Section 29 — Frontend Development Scope of Work

> **Framework:** Next.js 15 (App Router) with TypeScript
> **Design System:** SIMHA Brand Guidelines 2026 (`/srv/sites/predictatrade.com/SIMHA_BRAND_GUIDELINES_2026.md`)
> **Phase:** Phase 5 (scaffold), extended across Phases 19–22 (Verdict Terminal, WebSocket, Notifications, MQL Export)
> **White-label:** All UI components respect `org_id` theming from `app.org_branding`

---

### 29.1 Frameworks & Libraries

| Category | Technology | Version | Purpose |
|----------|-----------|---------|---------|
| **Framework** | Next.js | 15.x (App Router) | React server + client components, SSR/SSG, API routes |
| **Language** | TypeScript | 5.x | Type safety across all UI components |
| **Package Manager** | pnpm | 9.x | Fast, deterministic installs |
| **Styling** | TailwindCSS | 3.x | Utility-first CSS aligned with SIMHA design tokens |
| **Component Library** | Shadcn/ui | Latest | Accessible, unstyled primitives customized to SIMHA palette |
| **State Management** | Zustand | 4.x | Lightweight client state (signal cards, active tab, user preferences) |
| **Data Fetching** | TanStack Query | 5.x | Server state, caching, background refetch |
| **Charts** | Lightweight Charts (TradingView) | 4.x | OHLCV candlestick + indicator overlays |
| **Charts (Astro/CW)** | Recharts | 2.x | DI score history, composite score timeline, CW gauge components |
| **WebSocket** | native WebSocket API | — | Real-time signal streaming (Valkey PubSub bridge) |
| **Forms** | React Hook Form + Zod | Latest | Form handling and runtime schema validation |
| **Animation** | Framer Motion | 11.x | Page transitions, panel animations, loading states |
| **Icons** | Phosphor Icons | 2.x | SIMHA icon style (line + filled hybrid, angular geometry) |
| **Fonts** | next/font (Google Fonts) | — | Poppins (display), Inter (body), JetBrains Mono (code) |
| **Authentication** | NextAuth.js v5 | 5.x | JWT session management, TOTP MFA integration |
| **i18n** | next-intl | 3.x | Multi-language support (English, Arabic, Chinese) |
| **PWA** | next-pwa | Latest | Progressive Web App manifest, offline support |
| **Testing** | Playwright + Vitest | Latest | E2E tests + unit tests for React components |

---

### 29.2 Styling Strategy

All styling follows the **SIMHA Brand Guidelines 2026** with a **dark-first** approach:

#### Color System (Tailwind custom tokens)

```js
// tailwind.config.ts — SIMHA design tokens
export default {
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        // Brand primaries
        gold:   "#F4B400",   // Solar Gold — primary CTAs, highlights
        orange: "#F28C38",   // Cosmic Orange — buttons, active states
        green:  "#8BC34A",   // Growth Green — success, AI systems
        blue:   "#2D6CDF",   // Intelligence Blue — dashboards, data
        indigo: "#3F3D8F",   // Royal Indigo — headers, nav, authority
        // Backgrounds
        dark:   "#000000",   // Hero backgrounds
        panel:  "#1A1F36",   // Cards, sidebars, panels
        slate:  "#0f1320",   // Main content area background
        // Text
        muted:  "#A0AEC0",   // Secondary text, captions
        // PAT-specific semantic: signal direction
        bull:   "#8BC34A",   // Bullish signal
        bear:   "#EF4444",   // Bearish signal
        flat:   "#F4B400",   // MANDATORY_FLAT warning
      },
      boxShadow: {
        "glow-gold":  "0 0 20px rgba(244, 180, 0, 0.6)",
        "glow-blue":  "0 0 20px rgba(45, 108, 223, 0.6)",
        "glow-green": "0 0 20px rgba(139, 195, 74, 0.4)",
        "glow-red":   "0 0 20px rgba(239, 68, 68, 0.4)",
        "soft":       "0 4px 20px rgba(0, 0, 0, 0.2)",
      },
      fontFamily: {
        display: ["Poppins", "sans-serif"],
        body:    ["Inter", "sans-serif"],
        mono:    ["JetBrains Mono", "monospace"],
      },
      borderRadius: { sm: "6px", md: "12px", lg: "20px", xl: "32px" },
      animation: {
        "shimmer": "shimmer 1.5s ease-in-out infinite",
        "glow-pulse": "glow-pulse 2s ease-in-out infinite",
      },
    },
  },
};
```

#### Dark Mode Strategy

All PAT pages default to **dark mode**. Light mode is user-configurable via `app.users.ui_theme`. The dark mode class is applied at the `<html>` level by Next.js layout.

---

### 29.3 Design System & Tokens

Derived from SIMHA Brand Guidelines 2026:

#### Typography Scale

| Level | Size | Weight | Font | Usage |
|-------|------|--------|------|-------|
| H1 | 48px | 700 | Poppins | Hero titles, landing pages |
| H2 | 36px | 600 | Poppins | Section headings |
| H3 | 28px | 600 | Poppins | Card headings |
| H4 | 22px | 500 | Poppins | Subsections |
| Body Large | 18px | 400 | Inter | Lead paragraphs |
| Body | 16px | 400 | Inter | Standard text |
| Small | 14px | 400 | Inter | Captions, labels |
| Micro | 12px | 300 | Inter | Legal, footnotes |
| Code | 14px | 400 | JetBrains Mono | All code, indicators, scores |

Minimum font size in production: **14px** (WCAG 2.1 AA).

#### Spacing System

Base unit: **8px**. Scale: 4 / 8 / 12 / 16 / 24 / 32 / 48 / 64px.

#### Shadow & Elevation Tokens

```
shadow.soft       0 4px 20px rgba(0, 0, 0, 0.2)        — cards, panels
shadow.glow-gold  0 0 20px rgba(244, 180, 0, 0.6)       — primary CTAs
shadow.glow-blue  0 0 20px rgba(45, 108, 223, 0.6)      — info, intelligence
shadow.glow-green 0 0 20px rgba(139, 195, 74, 0.4)      — success, AI
shadow.glow-red   0 0 20px rgba(239, 68, 68, 0.4)       — bearish, danger
```

---

### 29.4 Component System

All components are built with **Shadcn/ui primitives** restyled to SIMHA tokens:

#### Core UI Components

| Component | Description | SIMHA Token Applied |
|-----------|-------------|---------------------|
| `<Button variant="primary">` | Gold background, black text | `bg-gold text-black hover:shadow-glow-gold` |
| `<Button variant="secondary">` | Blue border, white text | `border-blue text-white hover:bg-blue` |
| `<Button variant="ghost">` | Transparent, muted text | `text-muted hover:text-white` |
| `<Button variant="danger">` | Red, MANDATORY_FLAT actions | `bg-red-600 text-white` |
| `<Card glow="gold">` | Panel card with gold hover glow | `bg-panel border-white/5 rounded-2xl` |
| `<Card glow="blue">` | Panel card with blue hover glow | — |
| `<Badge variant="bull">` | Bullish signal badge | `bg-green text-black` |
| `<Badge variant="bear">` | Bearish signal badge | `bg-red-600 text-white` |
| `<Badge variant="flat">` | MANDATORY_FLAT warning | `bg-gold text-black animate-glow-pulse` |
| `<ScoreBar score={n}>` | Composite score −300 to +300 | Color-coded green/gold/red |
| `<HexagramCard hex={n}>` | I Ching hexagram display | Gold glyph on dark panel |
| `<PlanetStrengthMeter>` | Shadbala 6-component bar | Blue bars with gold accent |

#### PAT-Specific Components (Verdict Terminal)

| Component | Description |
|-----------|-------------|
| `<SignalCard>` | Primary signal display: instrument, direction badge, composite score bar, DI/CW/Western/AI subscores, override warnings |
| `<DIWidget>` | Vedic DI panel: 17 body positions, nakshatra, hora lord, dasha, Shadbala meters |
| `<CWWidget>` | Chinese Wisdom panel: I Ching hexagram, Bazi clock (animated 2-hour blocks), flying star grid, ZWDS palace map |
| `<WesternAstroWidget>` | Western panel: planet aspect matrix, retrograde indicator, lunation phase, Gold natal overlay |
| `<CompositeScoreGauge>` | Animated arc gauge: −300 to +300, color zones (red/yellow/green) |
| `<EMAChart>` | TradingView Lightweight Chart with EMA 9/21/50/100/200 overlays, ICT zones |
| `<ICTZoneOverlay>` | Order blocks, FVGs, liquidity levels, OTE zones on chart canvas |
| `<SessionClock>` | Live session indicator: Asian / London / NY Overlap with countdown |
| `<OrderBookWidget>` | Crypto exchange order book depth (bid/ask levels, spread indicator) |
| `<FundingRateTicker>` | Perpetual funding rate live ticker per exchange |
| `<ExchangeStatusBar>` | 5-exchange connectivity status (Binance/OKX/CDC/Bybit/KuCoin) |
| `<TZDisplay>` | User's localised time with broker offset indicator |
| `<CopyTradeAlert>` | Warning banner when copy-trade attempt detected |
| `<MandatoryFlatBanner>` | Pulsing gold banner: "⚠️ MANDATORY FLAT — CW/DI danger condition active" |

---

### 29.5 High-Fidelity Mockup Specifications

The following screens require full high-fidelity design before development:

| Screen | Key Components | Priority |
|--------|---------------|---------|
| **Landing Page** | Hero (SIMHA gradient BG), feature cards, pricing tiers, testimonials, CTA | P1 |
| **Login / MFA** | Minimal dark card, TOTP step, error states | P1 |
| **Verdict Terminal (Dashboard)** | SignalCard list, DIWidget, CWWidget, WesternWidget, CompositeGauge, EMAChart with ICT overlays | P1 |
| **Signal Detail View** | Full signal breakdown, all 15 factors, EMA chart, ICT structure visual | P1 |
| **Subscription Plans** | Tier cards (gold glow on Elite), Stripe + NOWPayments payment flow | P1 |
| **Account Settings** | Timezone picker, MT5 account binding, API key management, TOTP setup | P2 |
| **TradingView Integration** | Webhook URL display, Pine Script snippet, blend mode selector | P2 |
| **Crypto Exchange Dashboard** | Exchange status bar, order book widget, funding rate ticker, positions table | P2 |
| **Admin Console** | Org management, audit events, ops incident tracker, user management | P3 |
| **White-Label Setup** | Domain binding, branding upload, reseller plan config | P3 |

**Mockup Tools:** Figma (primary), following SIMHA Design System 2026 file structure:
- `01 Foundations` — Color Styles, Typography, Grid, Effects
- `02 Components` — Buttons, Cards, Inputs, Modals, Navigation, Badges, Data Tables
- `03 Patterns` — Dashboard layouts, form patterns, data visualization, empty states
- `04 PAT Product` — All screens listed above

---

### 29.6 Interactive Prototype Specifications

| Prototype Flow | Screens | Purpose |
|----------------|---------|---------|
| **Onboarding Flow** | Landing → Sign Up → Verify Email → MFA Setup → Timezone Picker → Dashboard | New user activation |
| **Signal Discovery Flow** | Dashboard → Signal Card → Signal Detail → Chart Drill-down → Execute (MT5) | Primary value delivery |
| **Subscription Upgrade Flow** | Pricing Page → Plan Select → Stripe Payment → NOWPayments Crypto → Confirmation | Revenue conversion |
| **TradingView Setup Flow** | Settings → TradingView → Webhook URL copy → Pine Script guide → Test Alert | Integration activation |
| **Crypto Exchange Setup Flow** | Settings → Exchange → Add API keys → Sandbox test → Enable live | Exchange activation |
| **MT5 Bridge Setup Flow** | Settings → MT Bridge → Account Register → Download EA → Installation guide | Execution activation |

All prototypes built in Figma with Smart Animate transitions (300ms cubic-bezier easing, as per SIMHA motion system).

---

### 29.7 Page & Route Architecture (Next.js App Router)

```
app/
├── (marketing)/
│   ├── page.tsx                     # Landing page
│   ├── pricing/page.tsx             # Subscription tier page
│   └── about/page.tsx               # About page
├── (auth)/
│   ├── login/page.tsx               # Login form
│   ├── register/page.tsx            # Sign-up form
│   └── verify/page.tsx              # Email + TOTP MFA
├── (dashboard)/
│   ├── layout.tsx                   # Dashboard layout (sidebar + topbar)
│   ├── terminal/page.tsx            # Verdict Terminal — primary signal view
│   ├── signals/[id]/page.tsx        # Individual signal detail
│   ├── chart/[symbol]/page.tsx      # Full chart view with EMA + ICT overlays
│   ├── exchange/page.tsx            # Crypto exchange dashboard
│   ├── astro/page.tsx               # Full DI + Western + Shadbala panel
│   ├── chinese/page.tsx             # Full CW + ZWDS + Bazi panel
│   ├── backtesting/page.tsx         # Backtesting results viewer
│   └── settings/
│       ├── account/page.tsx         # Profile, timezone, preferences
│       ├── mt5/page.tsx             # MT5 account binding + EA download
│       ├── tradingview/page.tsx     # TradingView webhook setup
│       ├── exchange/page.tsx        # Exchange API key management
│       ├── billing/page.tsx         # Subscription + payment history
│       └── security/page.tsx        # TOTP + session management
├── (admin)/
│   ├── layout.tsx                   # Admin layout
│   ├── dashboard/page.tsx           # System health overview
│   ├── users/page.tsx               # User management
│   ├── orgs/page.tsx                # White-label org management
│   ├── audit/page.tsx               # Audit event log viewer
│   └── ops/page.tsx                 # Ops incident tracker
└── api/
    └── auth/[...nextauth]/route.ts  # NextAuth.js handler
```

---

### 29.8 Build & Environment Tools

| Tool | Purpose | Config File |
|------|---------|------------|
| **pnpm** | Package management | `pnpm-lock.yaml` |
| **Next.js 15** | Build, dev server, production export | `next.config.ts` |
| **TypeScript** | Type checking (strict mode) | `tsconfig.json` |
| **TailwindCSS** | Utility CSS + SIMHA token plugin | `tailwind.config.ts` |
| **ESLint** | Linting (Next.js ruleset + custom) | `.eslintrc.json` |
| **Prettier** | Code formatting | `.prettierrc` |
| **Vitest** | Component unit testing | `vitest.config.ts` |
| **Playwright** | E2E testing | `playwright.config.ts` |
| **next/bundle-analyzer** | Bundle size analysis | `ANALYZE=true pnpm build` |
| **Storybook** | Component development + documentation | `.storybook/` |

**Build targets:**

| Environment | Command | Output | API URL |
|-------------|---------|--------|---------|
| Development | `pnpm dev` | Hot reload on :3001 | `http://localhost:8000` |
| Staging | `pnpm build && pnpm start` | Production build | `https://staging.predictatrade.com` |
| Production | `pnpm build && pnpm start` | Production build | `https://predictatrade.com` |

---

### 29.9 Accessibility & Quality Standards

Following SIMHA Brand Guidelines 2026 accessibility requirements:

- **WCAG 2.1 Level AA** compliance minimum across all screens
- **Contrast ratios:** White on Dark Slate = 12.6:1 (AAA); Gold on Black = 9.2:1 (AAA); Blue on Black = 5.2:1 (AA)
- **Focus states:** 2px solid `#2D6CDF` outline with 2px offset — visible on all interactive elements
- **Font size minimum:** 14px in all production UI (no exceptions)
- **Touch/click targets:** Minimum 44×44px
- **Alt text:** All meaningful images; all signal icons include `aria-label`
- **Screen reader:** All PAT-specific widgets include ARIA roles and live regions for score updates
- **Keyboard navigation:** Full keyboard support for dashboard navigation and signal interaction

---

### 29.10 Annotated Specifications — Key Screens

#### Verdict Terminal (Primary Dashboard)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  TOPBAR: Logo | Session Clock (Asian/London/NY) | TZ Display | User Menu│
├──────────────┬──────────────────────────────────────────────────────────┤
│  SIDEBAR     │  MAIN CONTENT AREA                                       │
│  (w-64       │  ┌────────────────────────────────────────────────────┐  │
│   bg-panel)  │  │  MANDATORY FLAT BANNER (gold pulsing, if active)   │  │
│              │  └────────────────────────────────────────────────────┘  │
│ ─ Terminal   │  ┌─────────────────┐  ┌───────────────────────────────┐  │
│ ─ Signals    │  │  COMPOSITE      │  │  ACTIVE SIGNAL CARD           │  │
│ ─ Chart      │  │  SCORE GAUGE    │  │  Symbol: XAUUSD  Dir: ▲ BUY   │  │
│ ─ Exchange   │  │  −300 to +300   │  │  Composite: +247 ████████░    │  │
│ ─ DI / CW    │  │  Color: green   │  │  DI: +68 CW: +45 AI: +72      │  │
│ ─ Astro      │  │  animated arc   │  │  Western: +22 COT: +40        │  │
│ ─ Backtest   │  └─────────────────┘  │  ICT Score: 95/120            │  │
│ ─ Settings   │  ┌─────────────────┐  │  EMA: Aligned                 │  │
│              │  │  EMA CHART      │  │  Status: PUBLISHED            │  │
│              │  │  Lightweight    │  └───────────────────────────────┘  │
│              │  │  Charts w/      │  ┌────────┐ ┌────────┐ ┌────────┐   │
│              │  │  OB/FVG/        │  │DI Panel│ │CW Panel│ │ WA     │   │
│              │  │  Liq overlays   │  │        │ │        │ │ Panel  │   │
│              │  └─────────────────┘  └────────┘ └────────┘ └────────┘   │
└──────────────┴──────────────────────────────────────────────────────────┘
```

**Layout Notes:**
- Sidebar: `bg-panel` (#1A1F36), 256px wide, `border-r border-white/5`
- Main content: `bg-slate` (#0f1320)
- Signal cards: `bg-panel rounded-2xl border border-white/5 hover:shadow-glow-gold`
- Bullish direction badge: `bg-green text-black font-semibold rounded-full px-3 py-1`
- Bearish direction badge: `bg-red-600 text-white font-semibold rounded-full px-3 py-1`
- MANDATORY_FLAT banner: `bg-gold text-black font-bold animate-glow-pulse`
- Composite score bar: gradient from red (−300) → gold (0) → green (+300), current position as white dot

#### EMA Chart Overlay Specifications

| Overlay | Color | Width | Opacity |
|---------|-------|-------|---------|
| EMA 9 | `#8BC34A` (Green) | 1px | 0.9 |
| EMA 21 | `#F4B400` (Gold) | 1.5px | 0.9 |
| EMA 50 | `#2D6CDF` (Blue) | 2px | 0.85 |
| EMA 100 | `#F28C38` (Orange) | 1.5px | 0.7 |
| EMA 200 | `#3F3D8F` (Indigo) | 2px | 0.8 |
| OB Zone (Bullish) | `#8BC34A` (Green) | Fill | 0.12 |
| OB Zone (Bearish) | `#EF4444` (Red) | Fill | 0.12 |
| FVG (Bullish) | `#2D6CDF` (Blue) | Fill | 0.10 |
| FVG (Bearish) | `#EF4444` (Red) | Fill | 0.10 |
| Liquidity (BSL) | `#F4B400` (Gold) | Dashed line | 0.7 |
| Liquidity (SSL) | `#EF4444` (Red) | Dashed line | 0.7 |

#### Subscription Pricing Cards

- Tier cards: `bg-panel rounded-2xl p-8 border border-white/5`
- Elite tier: `border-gold shadow-glow-gold` — visually elevated
- Price: Poppins Bold 36px Gold color
- CTA: Primary gold button (`bg-gold text-black`) with gold glow on hover
- Crypto payment badge: NOWPayments icon + "100+ crypto coins accepted" label in green

---

### 29.11 White-Label UI Theming

When `org_id` is resolved via the domain resolver (Section 23), the frontend reads `app.org_branding` and overrides:

| Token | Source | Override |
|-------|--------|---------|
| Primary brand color | `org_branding.primary_color` | Replaces `#F4B400` Gold |
| Logo URL | `org_branding.logo_url` | Replaces SIMHA/PAT logo |
| Brand name | `org_branding.brand_name` | Replaces "Predict-A-Trade" in UI text |
| Secondary color | `org_branding.secondary_color` | Replaces `#2D6CDF` Blue |
| Background color | `org_branding.bg_color` | Replaces `#0f1320` Slate |
| Favicon URL | `org_branding.favicon_url` | Replaces default favicon |

White-label colors are injected as CSS custom properties at the `:root` level by a server component. Tailwind classes remain unchanged; CSS variables override the token values.

```tsx
// app/(dashboard)/layout.tsx — white-label CSS injection
export default async function DashboardLayout({ children, params }) {
  const branding = await getOrgBranding(params.orgId);
  return (
    <html style={{
      "--color-gold":  branding?.primary_color  ?? "#F4B400",
      "--color-blue":  branding?.secondary_color ?? "#2D6CDF",
      "--color-slate": branding?.bg_color        ?? "#0f1320",
    } as React.CSSProperties}>
      {children}
    </html>
  );
}
```

---

---

# Section 30 — Extended Strategy Library & MQL5 Architecture

> **Source:** `/srv/sites/predictatrade.com/reference/PREDICT_A_TRADE_TRADING_STRATEGIES_FOR_MQL_PART1.md` (50 categories, text format), `PART2.md` (10 categories × tabular, 1000+ strategies, MQL5 class patterns), `PART3.md` (29 sections, parameter-variation reference)
> **Phase allocation:** Phase 15 (Backtesting — strategy registry), Phase 17 (MQL5 EA — Factory pattern), Phase 18 (retraining — RL strategies)

---

## 30.1 Strategy Library Overview

The three reference documents define a **1,000+ strategy universe** across 10 macro-categories. PAT does not implement all strategies simultaneously; instead, it maintains a **Strategy Registry** in the `research` schema and selects/combines strategies via the Composite Scoring System and backtesting pipeline.

### Macro-Category Summary

| Category | PART2 Count | PAT Relevance | Implementation Phase |
|----------|-------------|---------------|----------------------|
| 1 — Trend Following | 150 | CORE — EMA/ICT engine already scoped | Phase 9b (Indicators) |
| 2 — Mean Reversion | 150 | CORE — BB/RSI reversion, VWAP | Phase 9b |
| 3 — Breakout | 150 | CORE — ICT/SMC BOS/MSS already scoped | Phase 9c |
| 4 — Scalping | 150 | LIMITED — M1/M5 only for ELITE+ | Phase 9c |
| 5 — Machine Learning & AI | 100 | CORE — Supervised+RL already scoped | Phase 14b |
| 6 — Astrological | 100 | CORE — DI+CW+Western engines | Phase 10/11 |
| 7 — Quantitative | 150 | CORE — GARCH, ARIMA, Pairs Trading | Phase 14b (new) |
| 8 — Fundamental | 100 | CORE — Macro/Geopolitical engine | Phase 14 |
| 9 — Technical Patterns | 100 | NEW — Harmonic + Elliott Wave | Phase 9e (new) |
| 10 — Advanced Composite | 100 | CORE — Multi-factor scoring | Phase 14c |

**Total strategies catalogued: 1,250+ (including PART3 parameter variations)**

---

## 30.2 MQL5 EA Architecture — Strategy Factory Pattern

The MQL5 Expert Advisor (Phase 17) implements the **Strategy Factory Pattern** as the EA architecture standard. This is the canonical class design:

### Base Strategy Interface
```mql5
class CXAUStrategy {
protected:
    string           m_name;
    ENUM_TIMEFRAMES  m_tf;
    double           m_lotSize;
    double           m_slPercent;
    double           m_tpPercent;
public:
    virtual bool   Initialize()              = 0;
    virtual bool   CheckBuySignal()          = 0;
    virtual bool   CheckSellSignal()         = 0;
    virtual double GetStopLoss(bool isBuy)   = 0;
    virtual double GetTakeProfit(bool isBuy) = 0;
    virtual string GetName() { return m_name; }
};
```

### Strategy Factory
```mql5
class CStrategyFactory {
public:
    static CXAUStrategy* CreateStrategy(int strategyID) {
        switch(strategyID) {
            case 1:   return new CStrategy_GoldenCross();        // EMA 9/21 + ADX
            case 3:   return new CStrategy_TripleEMASweep();     // EMA 5/13/34
            case 26:  return new CStrategy_ICTBreakout();        // BOS/MSS + OB
            case 151: return new CStrategy_RSIReversion();       // RSI OB/OS + BB
            case 541: return new CStrategy_XGBoostSignal();      // 70-feature vector
            case 641: return new CStrategy_DI_NewMoon();         // DI: Sun/Moon 0°
            case 686: return new CStrategy_DI_Nakshatra();       // DI: Nakshatra lord
            case 736: return new CStrategy_PairsXAUDXY();        // XAU vs DXY cointegration
            // ... all registered strategies
            default:  return NULL;
        }
    }
};
```

### Portfolio Manager (Multi-Strategy EA)
```mql5
class CPortfolioManager {
private:
    CXAUStrategy* m_strategies[];
    double        m_weights[];
public:
    bool   AddStrategy(int id, double weight);
    bool   RemoveStrategy(int id);
    double GetCompositeSignal();    // weighted vote: -1.0 to +1.0
    double GetPositionSize();       // Kelly-adjusted across all strategies
    bool   CheckAnyCircuitBreaker();
};
```

### EA Initialization (PAT Standard Config)
The production EA loads the following strategy set by default (adjustable via input parameters):

| Strategy ID | Name | Weight | Category |
|-------------|------|--------|----------|
| 1 | Golden Cross (EMA 9/21 + ADX) | 0.15 | Trend |
| 3 | Triple EMA Sweep (5/13/34) | 0.10 | Trend |
| 26 | ICT BOS/MSS Breakout | 0.20 | ICT/SMC |
| 151 | RSI Mean Reversion | 0.10 | Reversion |
| 541 | XGBoost 70-Feature | 0.25 | AI/ML |
| 686 | DI Nakshatra Entry | 0.10 | Astro/DI |
| 928 | Quant-Astro Composite | 0.10 | Composite |

---

## 30.3 New Strategy Types Added to Scope

### 30.3a Harmonic Pattern Strategies (NEW — Phase 9e)

Harmonic patterns are geometric price patterns using Fibonacci ratios. The following patterns are added to the Technical Analysis Engine (alongside existing Fibonacci retracement/extension):

| Pattern | XA Retracement | AB Retracement | BC Retracement | CD Extension | Ideal Entry |
|---------|----------------|----------------|----------------|--------------|-------------|
| Gartley | 0.618 | 0.382–0.886 | 0.382–0.886 | 1.272–1.618 | D = 0.786 of XA |
| Bat | 0.382–0.500 | 0.382–0.886 | 0.382–0.886 | 1.618–2.618 | D = 0.886 of XA |
| Butterfly | 0.786 | 0.382–0.886 | 0.382–0.886 | 1.618–2.618 | D = 1.272 of XA |
| Crab | 0.382–0.618 | 0.382–0.886 | 0.382–0.886 | 2.618–3.618 | D = 1.618 of XA |
| ABCD | N/A | N/A | 0.382–0.886 | 1.272–1.618 | D = BC × 1.618 |
| 3-Drive | Symmetric | N/A | Symmetric | Symmetric | Drive 3 completion |

**MQL5 Implementation:** `iCustom()` with custom harmonic pattern library; pattern detection via 4-point swing XABCD labelling; OTE zone (0.618–0.786) overlap adds ICT confluence.

**DB Table:** `research.harmonic_patterns` (new — added to DATABASE_MODEL.md scope)
- `id`, `instrument_id`, `timeframe`, `pattern_type` ENUM, `x_price/a_price/b_price/c_price/d_price`, `completion_pct`, `entry_zone_high/low`, `target_1/2`, `status`, `detected_at`

**Harmonic + ICT Confluence:** When a Harmonic pattern D-point aligns with an ICT Order Block (± 0.5×ATR), this creates a **Harmonic-OB Confluence Zone** — highest-probability entry in the entire strategy library. Score contribution: up to +15 added to ICT Confluence Score (cap remains 120).

---

### 30.3b Elliott Wave Strategies (NEW — Phase 9e)

Elliott Wave counting adds wave-phase context to directional signals:

| Wave | Characteristics | PAT Action | Score Impact |
|------|-----------------|------------|--------------|
| Wave 1 | Impulse from base | BUY (early) | +5 |
| Wave 2 | Correction 50–78.6% | BUY deeper | +10 (if at OTE) |
| Wave 3 | Strongest impulse; longest | STRONG BUY | +20 |
| Wave 4 | Correction; does not overlap W1 | BUY if in discount | +5 |
| Wave 5 | Final impulse; divergence likely | REDUCE longs | −5 |
| Wave A | Corrective start | SELL (early) | −5 |
| Wave B | Corrective rally; trap | SELL B | −10 |
| Wave C | Final corrective drop | STRONG SELL | −15 |

**Wave Degree Hierarchy:** Grand Super Cycle → Super Cycle → Cycle → Primary → Intermediate → Minor → Minute → Minuette → Sub-Minuette

**MQL5 Implementation:** Automated wave counting via ZigZag indicator + Fibonacci ratio validation; D1/H4 for primary wave degree; H1/M15 for minor waves.

**DB Table:** `research.elliott_waves` (new)
- `id`, `instrument_id`, `timeframe`, `wave_degree`, `wave_number`, `wave_label` (1/2/3/4/5/A/B/C), `start_price`, `end_price`, `start_ts`, `end_ts`, `confidence_pct`, `is_complete`

---

### 30.3c Ichimoku Cloud Strategies (ENHANCED — Phase 9b)

Ichimoku was partially referenced; full integration added:

**Five Lines:**
- **Tenkan-sen** (Conversion): (9H + 9L) / 2 — short-term trend
- **Kijun-sen** (Base): (26H + 26L) / 2 — medium-term trend
- **Senkou A** (Leading A): (Tenkan + Kijun) / 2, plotted 26 periods ahead
- **Senkou B** (Leading B): (52H + 52L) / 2, plotted 26 periods ahead
- **Chikou** (Lagging): Current close plotted 26 periods behind

**PAT Signal Rules:**

| Signal | Condition | Composite Score Impact |
|--------|-----------|----------------------|
| Strong Buy | Price > cloud, Tenkan > Kijun, cloud bullish (A>B), Chikou > price | +12 |
| Bullish TK Cross | Tenkan crosses above Kijun above cloud | +8 |
| Kijun Touch | Price pulls back to Kijun and bounces | +6 |
| Cloud Support | Price approaches cloud from above and bounces | +5 |
| Cloud Breakout | Price closes above bearish cloud | +10 |
| Strong Sell | Inverse of Strong Buy | −12 |

**Feature Addition to 70-Feature AI Vector:** 4 new Ichimoku features added (ichimoku_bullish_strong, ichimoku_tk_cross, ichimoku_cloud_position, ichimoku_chikou_clear) → **74-feature vector** total.

---

### 30.3d GARCH/ARIMA Time Series Models (NEW — Phase 14b)

**GARCH (Generalized Autoregressive Conditional Heteroskedasticity):**
- GARCH(1,1): Standard volatility forecasting model
- EGARCH: Asymmetric — negative shocks increase volatility more than positive (leverage effect; key for XAUUSD)
- GJR-GARCH: Threshold model, captures asymmetric volatility
- Application: Dynamic ATR substitute for position sizing when market regime changes
- Retrain: Daily (rolling 252-bar window)
- Output: Conditional volatility forecast → adjusts SL/position size in real-time

**ARIMA Time Series:**
- ARIMA(2,1,2): Returns series modeling
- SARIMA: Seasonal component for daily/weekly cycles
- Application: Short-term return forecast as auxiliary signal
- Limits: ARIMA contribution capped at 10% of AI pillar score (low alpha standalone; used as confirmation)

**Markov Regime-Switching:**
- 2-state HMM: TRENDING vs RANGING regime detection
- State persistence probability tracked
- Application: Strategy selector — trend strategies activate in TRENDING regime; mean-reversion in RANGING
- DB: `research.market_regimes` table — `instrument_id`, `timeframe`, `regime_state` (TRENDING/RANGING/VOLATILE), `probability`, `detected_at`

---

### 30.3e Pairs Trading & Intermarket (NEW — Phase 14b)

**XAU/USD vs DXY Cointegration:**
- Johansen cointegration test run weekly
- Z-score of spread: `(XAUUSD − β × DXY) / σ`
- Entry: Z-score < −2 = XAUUSD undervalued vs DXY → Long XAUUSD
- Exit: Z-score returns to 0
- Half-life of mean reversion (Ornstein-Uhlenbeck): target 2–10 days
- DB: `research.pairs_signals` table

**Intermarket Correlations tracked:**
| Asset Pair | Relationship | Signal Use |
|-----------|-------------|------------|
| XAUUSD / DXY | Inverse (~−0.75) | DXY strength = Gold weakness warning |
| XAUUSD / XAGUSD | Positive (~+0.85) | Gold-Silver ratio divergence = reversion signal |
| XAUUSD / US10Y | Inverse (real yields) | Rising real yields = Gold headwind |
| XAUUSD / SPX500 | Variable (risk-off) | Risk-off correlation increases in crisis |
| XAUUSD / VIX | Positive (fear) | VIX spike = Gold demand increase |
| XAUUSD / USDCNH | Inverse (China demand) | CNH weakness = China gold buying |

---

## 30.4 Astrological Strategy Integration Map

Category 6 (PART2) defines 100 astrological trading strategies. These are integrated into the existing DI, CW, and Western Astrology engines as **named signal rules**. The table below maps the strategy IDs from PART2 to PAT engine components:

### DI Engine Astrological Signal Rules

| PART2 Strategy | Rule Name | Engine | Score Impact | Circuit Breaker? |
|----------------|-----------|--------|-------------|-----------------|
| #641 New Moon Long | `DI_NEW_MOON_BULL` | DI | +8 | No |
| #642 Full Moon Short | `DI_FULL_MOON_BEAR` | DI | −8 | No |
| #645 Sun-Moon Sextile | `DI_SEXTILE_60` | DI | +4 | No |
| #646 Mercury Retrograde | `DI_MERCURY_RX` | DI | −5 (all comm/tech) | No |
| #651 Sun-Jupiter Conj | `DI_SUN_JUP_CONJ` | DI | +12 | No |
| #652 Sun-Saturn Conj | `DI_SUN_SAT_CONJ` | DI | −10 | No |
| #653 Venus-Jupiter Conj | `DI_VEN_JUP_GROWTH` | DI | +15 | No |
| #654 Mars-Saturn Conj | `DI_MARS_SAT_BLOCK` | DI | −12 | No |
| #655 Jupiter-Saturn Conj (Great Conj) | `DI_GREAT_CONJUNCTION` | DI | +20/−20 | ADVISORY |
| #659 Lunar Nodes | `DI_ECLIPSE_KARMIC` | DI | (eclipse logic) | YES |
| #671 Lunar Eclipse | `DI_LUNAR_ECLIPSE` | DI | APO trigger | YES (3-day) |
| #672 Solar Eclipse | `DI_SOLAR_ECLIPSE` | DI | APO trigger | YES (3-day) |
| #680 Void of Course Moon | `DI_VOC_MOON` | DI | −8 | NO (reduce size 50%) |
| #681 Moon in Taurus | `DI_MOON_TAURUS` | DI | +10 (Gold bullish) | No |
| #682 Moon in Scorpio | `DI_MOON_SCORPIO` | DI | −10 (Gold bearish) | No |

### DI Vedic Strategy Rules

| PART2 Strategy | Rule Name | Engine | Score Impact |
|----------------|-----------|--------|-------------|
| #686 Nakshatra Trading | `DI_NAKSHATRA_DAILY` | DI | ±6 per nakshatra lord |
| #687 Tithi Trading | `DI_TITHI_STRENGTH` | DI | ±4 |
| #691 Hora Trading | `DI_HORA_PLANETARY` | DI | ±8 |
| #692 Rahu Kalam | `DI_RAHU_KALAM` | DI | −15 | (already in scope) |
| #693 Gulika Kalam | `DI_GULIKA_KALAM` | DI | −12 | (already in scope) |
| #694 Yamagandam | `DI_YAMAGANDAM` | DI | −10 | (already in scope) |
| #695 Abhijit Muhurta | `DI_ABHIJIT_AUSPICIOUS` | DI | +12 |
| #696 Dasha Period | `DI_MAHADASHA_SCORE` | DI | ±20 |
| #697 Bhukti Period | `DI_ANTARDASHA_SCORE` | DI | ±10 |
| #698 Antara Period | `DI_PRATYANTAR_SCORE` | DI | ±5 |
| #700 Ashtakavarga | `DI_ASHTAKAVARGA` | DI | ±8 |
| #701 Shadbala | `DI_SHADBALA_STRENGTH` | DI | ±10 |
| #702 Panchanga | `DI_PANCHANGA_DAILY` | DI | ±6 |
| #703 Choghadiya | `DI_CHOGHADIYA_WINDOW` | DI | ±8 |

### Western Astrology Strategy Rules

| PART2 Strategy | Rule Name | Engine | Score Impact |
|----------------|-----------|--------|-------------|
| #711 Sun Sign | `WEST_SUN_SIGN_THEME` | Western | ±5 (seasonal) |
| #714 Venus Sign | `WEST_VENUS_RISK_APPETITE` | Western | ±8 |
| #716 Jupiter Sign | `WEST_JUPITER_EXPANSION` | Western | ±10 |
| #717 Saturn Sign | `WEST_SATURN_RESTRICTION` | Western | −8 |
| #726 North Node | `WEST_NORTH_NODE` | Western | ±6 |
| #731 Fixed Stars (Regulus, Spica) | `WEST_FIXED_STAR_EXACT` | Western | ±12 |
| #732 Galactic Center (27° Sag) | `WEST_GALACTIC_CENTER` | Western | ±15 |
| #733 Super Galactic Center | `WEST_SUPER_GALACTIC` | Western | ±20 |

*Note: Galactic Center and Super Galactic Center are already tracked as bodies #16 and #17 in the DI body list. These PART2 rules provide the specific scoring rules for exact conjunctions.*

---

## 30.5 Fundamental Intelligence Engine — Enhanced Scope

Category 8 (PART2) covers 100 fundamental strategies. PAT's existing Macro Engine (FinBERT) is enhanced with structured fundamental data points:

### Gold-Specific Fundamental Inputs

| PART2 Strategy | Input | Source | Update Frequency | DB Table |
|----------------|-------|--------|-----------------|---------|
| #856 Mine Production | Global gold output (tonnes) | World Gold Council API | Quarterly | `market.fundamental_data` |
| #857 Scrap Supply | Recycling volumes | WGC API | Monthly | `market.fundamental_data` |
| #858 Central Bank Buying | Official sector net purchases | IMF/WGC | Monthly | `market.fundamental_data` |
| #859 ETF Flows | GLD/IAU/PHAU AUM changes | ETF issuer APIs | Daily | `market.etf_flows` (new) |
| #861 Options Flow | XAUUSD options put/call ratio | CME/CBOE API | Daily | `market.options_flow` (new) |
| #862 Physical Premium | COMEX futures vs spot | CME | Real-time | `market.futures_basis` (new) |
| #814 Fed Rates | FOMC target rate | FRED API | Per meeting | `market.central_bank_rates` (new) |
| #812 CPI Inflation | US CPI YoY | FRED API | Monthly | `market.macro_indicators` |

### Geopolitical Risk Score
- Sources: GDELT Project (event data), Reuters/Bloomberg headlines via FinBERT
- `market.geopolitical_events` table: `event_type` (WAR/SANCTIONS/ELECTION/DEBT_CRISIS/etc.), `region`, `severity` (1–10), `gold_impact` (BULLISH/BEARISH/NEUTRAL), `started_at`, `resolved_at`
- Geopolitical Risk Index (GRI): composite score from active events, weighted by severity
- GRI contributes to Macro Engine score (expands Macro pillar from ±20 to ±25 with geopolitical component)

---

## 30.6 Reinforcement Learning Trading (NEW — Phase 18b)

Category 5.3 (PART2) covers 25 RL strategies. The following RL agents are added to the ML roadmap:

### RL Strategy Set for PAT

| Algorithm | Strategy ID | State Space | Action Space | Application |
|-----------|-------------|-------------|-------------|-------------|
| PPO (Proximal Policy Optimization) | #603 | 74-feature vector + portfolio state | Buy/Sell/Hold + lot size | XAUUSD H1/H4 |
| SAC (Soft Actor-Critic) | N/A (extension) | Continuous state | Continuous lot size | Optimal sizing |
| DQN (Deep Q-Network) | #593 | Discretized price + indicator state | Buy/Sell/Hold | M15 scalping |
| A2C (Advantage Actor-Critic) | #601 | Portfolio state | Asset allocation weights | Multi-asset |

**RL Implementation Notes:**
- Training environment: custom OpenAI Gym-compatible wrapper around PAT backtesting engine
- Training data: 2015–2023 (in-sample), 2024–present (OOS evaluation)
- Reward function: Sharpe ratio improvement per episode (not raw P&L — reduces overfit)
- Episode length: 252 bars (1 trading year)
- Convergence criterion: Sharpe OOS > 1.5 sustained over 20 validation episodes
- Deployment: ONNX export of RL policy network → MT5 native inference (same as supervised models)
- Service: pat-ml conda env, weekly retraining alongside supervised models
- MLflow: RL experiments tracked separately under `experiment="rl_trading"`

---

## 30.7 Prohibited Strategies

The following strategy categories from the reference documents are **explicitly prohibited** in the PAT EA:

| Category | Strategies | Reason | Enforcement |
|----------|-----------|--------|-------------|
| Martingale | PART3 #732, PART2 #800 | Unlimited risk; account blow-up | EA hard block: `RISK_MARTINGALE_ALLOWED=false` |
| Grid Trading | PART1 §21 | Unlimited drawdown in trending markets | EA hard block: no grid logic in any strategy class |
| HFT Manipulation | PART2 #771 (Quote Stuffing), #772 (Layering/Spoofing) | Illegal; regulatory violation | Not implemented |
| Latency Arbitrage | PART2 #767 | Requires co-location; not applicable to retail MT5 | Not applicable |
| Averaging Down (add to losers) | Various | Conflicts with circuit breakers | EA hard block: no add-to-losing-position logic |
| D'Alembert | PART2 #802 | Progressive risk system | Blocked by Fixed Fractional enforcement |

---

## 30.8 Strategy Registry Schema (New DB Table)

A `research.strategy_registry` table formalizes the strategy library in the database:

```
research.strategy_registry
─────────────────────────────────────────────────────────────────────
id                  UUID PK
strategy_id         INTEGER UNIQUE        -- PART2 numeric ID (1–1000+)
name                VARCHAR(100)
category            VARCHAR(50)           -- trend/reversion/breakout/scalping/ml/astro/quant/fundamental/pattern/composite
source_document     VARCHAR(50)           -- PART1/PART2/PART3
primary_indicator   VARCHAR(50)
secondary_indicator VARCHAR(50)
applicable_tf       TEXT[]                -- ['H1','H4','D1']
applicable_assets   TEXT[]                -- ['XAUUSD','XAGUSD']
mql5_class_name     VARCHAR(100)          -- e.g. CStrategy_GoldenCross
is_implemented      BOOLEAN DEFAULT FALSE
is_enabled          BOOLEAN DEFAULT FALSE
is_prohibited       BOOLEAN DEFAULT FALSE -- Martingale/Grid etc
backtest_sharpe     NUMERIC(5,3)
backtest_accuracy   NUMERIC(5,3)
last_backtest_at    TIMESTAMPTZ
notes               TEXT
created_at          TIMESTAMPTZ DEFAULT now()
```

All 1,000+ strategies are seeded into this table via Alembic data migration in Phase 8. Strategies are progressively enabled as they are implemented and pass backtesting gates.

---

## 30.9 Updated Phase Allocation

| Phase | New Additions from Strategy Documents |
|-------|--------------------------------------|
| Phase 8 | Add strategy_registry, harmonic_patterns, elliott_waves, market_regimes, pairs_signals, etf_flows, options_flow, futures_basis, central_bank_rates, geopolitical_events tables to Alembic |
| Phase 9b | Ichimoku full integration (+4 features → 74-feature vector) |
| Phase 9e (NEW) | Harmonic pattern detection engine (Gartley/Bat/Butterfly/Crab/ABCD); Elliott Wave counter |
| Phase 14b | GARCH/EGARCH volatility models; ARIMA(2,1,2) return model; Markov regime-switching; XAU/DXY pairs trading; intermarket correlation engine |
| Phase 15 | Strategy registry seeding (1000+ rows); per-strategy backtest run; prohibited strategy gate |
| Phase 17 | CXAUStrategy base class; CStrategyFactory; CPortfolioManager; PAT default strategy weights |
| Phase 18b (NEW) | PPO + SAC + DQN RL strategy training; Gym-compatible environment; ONNX export |

---

## 30.10 Updated Composite Score Table

With all new strategy engines added, the composite score sources are updated:

| Pillar | Previous Range | New Range | New Components |
|--------|---------------|-----------|----------------|
| AI (ML/DL) | ±80 | ±80 | + GARCH vol adjustment (dynamic sizing) |
| DI (Vedic) | ±70 | ±70 | + 14 new named astrological rules |
| CW (Chinese) | ±50 | ±50 | Unchanged |
| Western Astrology | ±30 | ±30 | + Fixed Stars, Galactic/Super Galactic scoring |
| COT Analytics | ±40 | ±40 | + ETF flows overlay |
| Seasonality | ±10 | ±10 | Unchanged |
| Macro/Fundamental | ±20 | ±25 | + Geopolitical Risk Index |
| Technical/ICT | ±30 | ±35 | + Harmonic-OB confluence (+5), Elliott Wave context (+5) |
| Ichimoku | 0 | ±5 | NEW — folded into Technical pillar |
| Intermarket | 0 | ±10 | NEW — XAU/DXY Z-score, Gold-Silver ratio |
| **TOTAL** | **±300** | **±355** | **Normalized back to ±300 by weighted scaling** |

*The raw score is normalized: `composite_score = round(raw_score × (300 / 355))` to maintain the ±300 display range.*

---

*© 2026 Predict-A-Trade. Master Scope of Work v6.0. Confidential — Internal Planning.*
*This document supersedes all previous SOW versions (v1.0 through v5.0).*
*Last updated: April 2026*
*Last updated: April 2026*
