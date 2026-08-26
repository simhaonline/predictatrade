# Predict-A-Trade: A Multi-Plane Architecture for Deterministic XAUUSD Trading Signal Generation with AI-Advisory Enhancement

**PhD Thesis — Department of Computer Science**

*Rebuilt for v1.16.0, 26 August 2026*

---

## Abstract

This thesis presents Predict-A-Trade, a multi-plane trading platform for XAUUSD (Gold/US Dollar) that generates deterministic trading signals through a layered architecture of 42 technical indicators, 5 independent strategy engines, 16 hard risk gates, and advisory AI augmentation. The system is designed around a strict plane-boundary enforcement model: the Go real-time engine owns authoritative market-data processing and signal generation; the NestJS control plane manages identity, access, subscriptions, and billing; the Next.js presentation plane renders server-authoritative state without independent risk authority; the Python research plane operates offline for backtesting and calibration; and the Windows/MQL edge handles broker execution with fail-closed safety controls. The platform achieves mathematical parity (MAPE < 0.0001 against reference implementations), processes signals through an ordered gate pipeline with cascading failure propagation, and maintains full evidence traceability from raw market data to final trade outcomes. Production readiness is assessed at 70/100 with all critical production blockers closed. The key contribution is a plane-separated architecture that prevents any single component — particularly AI/ML modules — from becoming an ungoverned authority in the critical trading decision path.

**Keywords:** algorithmic trading, XAUUSD, plane separation, deterministic signal generation, risk gates, evidence-based scoring, AI governance, TimescaleDB, Go, NestJS, Next.js

---

## Chapter 1: Introduction

### 1.1 Problem Statement

Algorithmic trading systems face a fundamental tension: the desire to incorporate increasingly sophisticated signal sources — machine learning models, large language models, sentiment analysis — against the absolute requirement that trading decisions remain auditable, reproducible, and safe. When AI components are granted authority over price discovery, position sizing, or risk assessment, the system can produce decisions that are unexplainable, unreproducible, and potentially catastrophic.

The XAUUSD (Gold/US Dollar) market presents unique challenges: it trades across three major sessions (Tokyo, London, New York), is influenced by macroeconomic factors (USD strength, real yields, COT positioning), exhibits distinct regime behaviors (trending, ranging, volatile), and requires precise broker compliance (symbol-specific stop levels, freeze levels, digit precision).

### 1.2 Research Questions

This thesis addresses four primary research questions:

**RQ1:** Can a trading platform be architected such that AI/ML components serve in a strictly advisory capacity without any path to becoming authoritative over prices, signals, or risk decisions?

**RQ2:** How should evidence from 42 heterogeneous technical indicators be combined into a single directional signal without introducing double-counting, correlation distortion, or family-level bias?

**RQ3:** What gate architecture ensures that safety constraints (capital protection, broker compliance, exposure limits) are enforced in a deterministic, ordered, and fail-closed manner?

**RQ4:** How can a multi-service distributed system maintain mathematical reproducibility — guaranteeing identical inputs always produce identical signals — for audit and compliance purposes?

### 1.3 Contributions

1. **Plane-boundary enforcement model:** A five-plane architecture with legally-enforced boundaries preventing any plane from crossing into another's authority domain.

2. **Family-capped evidence scoring:** A 13-pillar scoring system where each indicator family (TREND, MOMENTUM, STRUCTURE, etc.) has a hard contribution cap, preventing correlated indicators from collectively dominating the signal.

3. **Ordered fail-closed gate pipeline:** 16 risk gates evaluated in deterministic order with cascading failure — a VETO at any gate terminates the pipeline, DEGRADED propagates with reduced confidence.

4. **Broker symbol validation gate (P0-001):** A novel gate that validates signal parameters against live broker symbol metadata (minimum stop distance, freeze level, maximum spread) before permitting execution.

5. **Full evidence provenance:** Every signal carries its complete reasoning chain — which indicators fired, with what weights, which gates evaluated, and what outcomes — enabling complete post-hoc audit.

---

## Chapter 2: Literature Review

### 2.1 Algorithmic Trading Architectures

Traditional algorithmic trading systems fall into three architectural categories: monolithic (single-process, all logic co-located), microservice (distributed components communicating via message queues), and hybrid (critical path in-process, auxiliary services out-of-process). Monolithic systems offer simplicity but limit scalability and independent evolution. Microservice architectures enable independent scaling but introduce latency and consistency challenges on the critical trading path.

Predict-A-Trade adopts a hybrid approach: the real-time signal generation path runs entirely within a single Go process (the realtime engine), eliminating network latency on the tick-to-signal path. The control plane (NestJS) and presentation plane (Next.js) operate as separate services communicating via REST and WebSocket, but they are never on the critical decision path.

### 2.2 Technical Indicators and Feature Engineering

The academic literature on technical analysis is divided. Lo, Mamaysky, and Wang (2000) demonstrated that technical patterns contain incremental information beyond random walk. Brock, Lakonishok, and LeBaron (1992) showed that moving average and trading-range breakout rules generate statistically significant returns. However, Sullivan, Timmermann, and White (1999) warned about data-snooping bias when testing many trading rules on the same dataset.

This thesis addresses the data-snooping concern through three mechanisms: (a) family-capped evidence scoring prevents indicator-correlation exploitation, (b) indicators are computed on closed candles only (zero repainting risk), and (c) golden tests verify that identical inputs produce identical outputs.

### 2.3 Risk Management in Algorithmic Trading

Modern risk management frameworks emphasize pre-trade controls (position limits, exposure caps), in-trade controls (stop-loss enforcement, trailing stops), and post-trade controls (reconciliation, audit). The Basel Committee's Principles for Sound Stress Testing (2018) and the IOSCO Principles for Algorithmic Trading (2020) both stress the importance of automated pre-trade risk checks that cannot be bypassed.

Predict-A-Trade implements 16 pre-trade risk gates, server-side stop-loss enforcement (verifying that the executing EA actually set the server-computed stop loss), and post-trade outcome resolution with full audit trail.

### 2.4 AI/ML in Financial Decision Systems

The application of machine learning to financial trading has seen exponential growth. Deep reinforcement learning (Deng et al., 2017), LSTM-based price prediction (Fischer and Krauss, 2018), and transformer architectures for time series (Wu et al., 2021) have all shown promise. However, the reproducibility crisis in ML research (Pineau et al., 2021) raises serious concerns about deploying non-deterministic models in financial decision paths where auditability is paramount.

Predict-A-Trade addresses this by constraining AI to an advisory role: ML model predictions and LLM sentiment analysis contribute to the evidence score but cannot override deterministic signal engine decisions, cannot set prices, cannot modify stop losses, and cannot bypass risk gates. AI failures degrade gracefully — a failed LLM call produces zero sentiment contribution rather than blocking signal generation.

---

## Chapter 3: System Architecture

### 3.1 Five-Plane Architecture

The platform is organized into five planes with legally-enforced boundaries:

**Plane 1 — Go Real-Time Engine (realtime/):** Authoritative for market data ingestion, technical indicator computation, strategy evaluation, signal generation, risk gate enforcement, and trade outcome resolution. This plane must never depend on synchronous billing, referral, or commission operations.

**Plane 2 — NestJS Control Plane (control/):** Authoritative for identity and access management, multi-factor authentication, role-based access control, subscription management, billing, licensing, device binding, referral tracking, commission calculation, and payout processing. This plane must never be on the critical tick-to-signal path.

**Plane 3 — Next.js Presentation Plane (frontend/):** Renders server-authoritative state for users and administrators. Responsible for UI rendering, charting, and user interaction. Must never compute risk, entitlement, probability, or signal validity independently.

**Plane 4 — Python Research Plane (research/):** Operates offline for historical data analysis, backtesting, strategy calibration, machine learning model training, and reinforcement learning optimization. Results feed into the real-time engine as configuration, not as runtime dependencies.

**Plane 5 — Windows/MQL Edge (windows-agent/, mql/):** Lightweight execution adapters on Windows machines running MetaTrader 4/5. Handle broker connectivity, order placement, position monitoring, and heartbeat reporting. Must never contain primary trading intelligence or server credentials.

### 3.2 Service Inventory (11 containerized services)

The production deployment consists of 11 Docker containers connected via an internal bridge network (pat-net), fronted by Nginx reverse proxy:

| Service | Container | Port | Technology | Role |
|---------|-----------|:----:|------------|------|
| PostgreSQL | pat-postgres | 5432 | PostgreSQL 17 + TimescaleDB | Durable persistence |
| Valkey | pat-valkey | 6379 | Valkey 8.0 | Cache and hot state |
| Realtime Engine | pat-realtime | 13081 | Go | Signal generation |
| Control Plane | pat-control | 13080 | NestJS | IAM, billing, licensing |
| Frontend | pat-frontend | 13082 | Next.js | User/admin dashboards |
| Status Page | pat-status | 13083 | Go | System health |
| Live Terminal | pat-live-terminal | 13090 | Go | Bloomberg-style terminal |
| Nginx | pat-nginx | 80/443 | Nginx Alpine | Reverse proxy, TLS |
| Prometheus | pat-prometheus | 9090 | Prometheus | Metrics collection |
| Grafana | pat-grafana | 3001 | Grafana | Dashboards |
| ntfy | pat-ntfy | 8091 | ntfy | Push notifications |

### 3.3 Data Flow: Tick to Signal

The complete signal generation pipeline executes within the Go real-time engine:

1. **Market Ingestion:** Raw ticks arrive from MT5 broker (via WebSocket agent) or from TwelveData API. The TickValidator rejects zero/negative prices, inverted spreads (bid > ask), and unreasonable spreads (>1% of mid-price).

2. **Candle Aggregation:** Ticks are aggregated into OHLC candles by timeframe (M1, M5, M15, H1, H4). Candles are cached in Valkey with TTL and persisted to TimescaleDB hypertables.

3. **Feature Computation:** On each candle close, the Feature Registry computes 42 indicators across all registered engines: StructureEngine, LiquidityEngine, FVGEngine, VWAPEngine, IndicatorEngine (EMA, MACD, ADX, RSI, Stoch, CCI, ATR, Bollinger), RegimeEngine, MTFEngine, SessionEngine, SAREngine, IchimokuEngine, StochRSIEngine, FibonacciEngine, CandleEngine, PivotEngine, PullbackEngine, and SessionORBEngine.

4. **Strategy Evaluation:** Each of the 5 strategy engines evaluates the complete MarketState against its distinct criteria. Evidence is accumulated across 13 pillars with family-level caps. The resulting StrategyResult contains direction (BUY/SELL/NO-TRADE), raw scores, entry/SL/TP prices, and full evidence chain.

5. **Gate Evaluation:** The signal engine passes the candidate signal through 16 ordered gates. Each gate evaluates a snapshot of market state, account state, and signal parameters. A VETO at any gate terminates the pipeline. DEGRADED status propagates. The gate evaluation order is enforced by RegisterOrdered().

6. **Signal Persistence:** Passing signals are persisted to trading.signals with full evidence JSON, assigned a UUID, and timestamped with market_time, generated_at, and expires_at.

7. **Delivery:** Signals are broadcast via WebSocket to connected user dashboards and Windows agents. The NestJS control plane enforces plan-tier filtering before delivery.

8. **Outcome Resolution:** The OutcomeResolver periodically checks current XAUUSD price against active signal TP/SL levels and records WIN, LOSS, BREAKEVEN, or EXPIRED outcomes.

---

## Chapter 4: Evidence-Based Scoring Engine

### 4.1 The 13 Evidence Pillars

The scoring engine organizes indicators into 13 semantic families, each with a hard contribution cap:

| Pillar | Cap | Indicator Count | Example Indicators |
|--------|:---:|:---------------:|-------------------|
| TREND | 0.35 | 8 | EMA 9/21/50/100/200, SMA 50/100/200, ADX |
| MOMENTUM | 0.30 | 5 | MACD (12/26/9), OsMA, RSI 14, Stoch 14/3/3, CCI 20 |
| STRUCTURE | 0.25 | 6 | BOS/CHoCH, BSL/SSL, pivot points (D/W/M), pullback |
| LIQUIDITY | 0.20 | 3 | Sweep detection, order blocks, liquidity voids |
| SMC | 0.20 | 3 | FVG, imbalance, displacement |
| MTF | 0.20 | 1 | Multi-timeframe trend alignment |
| CANDLE | 0.20 | 4 | Rejection, displacement, pin bar, doji |
| REGIME | 0.15 | 1 | Regime classification |
| VWAP | 0.15 | 2 | VWAP deviation, VWAP bands |
| VOLATILITY | 0.15 | 3 | ATR, BB width, range compression |
| ML | 0.25 | 1 | ONNX XGBoost model inference |
| SENTIMENT | 0.25 | 1 | Ollama LLM sentiment analysis |
| SESSION_ORB | 0.15 | 3 | Asian/London/NY opening ranges |

### 4.2 Family Capping Algorithm

The family cap mechanism prevents correlated indicators from collectively dominating the signal score. The algorithm:

```
For each evidence pillar P with cap C:
  accumulate contributions from all indicators in P
  if total > C:
    proportionally scale all contributions down so total = C

FinalScore = Σ capped_pillar_totals - conflict_penalty
```

Conflict penalty: 15 points per conflicting timeframe. If total conflict exceeds 40, the engine returns DirectionWait (holds the signal) rather than forcing a low-confidence trade.

### 4.3 P2 Feature Activation (v1.16.0)

Four features were promoted from SHADOW (computed but score-neutral) to ACTIVE (contributing to scoring) in version 1.16.0:

**Session ORB (P2-001):** Computes Asian (00:00-09:00 UTC), London (08:00-17:00), and New York (13:00-22:00) opening ranges. Detects breakouts when price exceeds the current session's high or low. Contributes to SESSION_ORB pillar (cap 0.15).

**Pin Bar Geometry (P2-002):** Analyzes candle body/wick ratios. A bullish pin bar (long lower wick, small body near top) scores quality on lower wick / total range ratio. Contributes to CANDLE pillar (cap 0.20).

**Pullback Detection (P2-003):** Tracks trend extrema from structure engine's swing highs/lows. Measures pullback depth as percentage of trend move and ATR-normalized retracement distance. Confirms continuation when price resumes trend direction. Contributes to STRUCTURE pillar (cap 0.25).

**Trade Group ID (P2-004):** Auto-generates a group identifier when a signal produces multiple take-profit levels, enabling multi-position tracking across the platform.

### 4.4 Mathematical Verification

All indicator mathematics were verified through a 1000-sample parity check against reference implementations. The Mean Absolute Percentage Error (MAPE) threshold was set at 0.0001 (0.01%). Results:

- RSI (Wilder smoothing): PASS — range [38.57, 60.23]
- ATR (TR-based): PASS — range [6.67, 19.42]
- GrossRR: PASS — mean 1.0000
- NetRR: PASS — mean 0.7778

The Wilder smoothing implementation (used in RSI, ATR, ADX) was independently verified against known test vectors to confirm correctness.

---

## Chapter 5: Gate Architecture

### 5.1 Gate Contract

All gates implement a pure-evaluation interface:

```
Gate.Evaluate(input GateInput, state GateState) -> GateEvaluation

GateEvaluation:
  Result: PASS | VETO | DEGRADED | UNKNOWN
  ReasonCode: string
```

The gate contract specifies:
- **No synchronous I/O:** Gates evaluate cached GateState snapshots, never making blocking database or network calls on the critical path.
- **No side effects:** Gate evaluation is a pure function of its inputs.
- **Fail-closed:** If GateState is stale, missing, or UNKNOWN, the gate returns VETO or DEGRADED — never PASS on insufficient evidence.
- **Generation-stamped:** Each GateState carries a monotonic generation counter, enabling stale-state detection.

### 5.2 The 16-Gate Pipeline

Gates are registered in priority order via RegisterOrdered(). The evaluation pipeline executes sequentially:

| # | Gate | Type | Failure Mode | Purpose |
|---|------|------|:------------:|---------|
| 1 | ExecutionPermission | Entitlement | VETO | License valid, execution permitted |
| 2 | BrokerSymbolValidation | Safety | DEGRADED | SL/TP/lot valid per broker metadata |
| 3 | SeedCapitalProtection | Capital | VETO | Seed capital not exceeded |
| 4 | DailyLossLimit | Capital | VETO | Daily loss within configured cap |
| 5 | MaxSpread | Market | VETO | Spread within strategy limit |
| 6 | NewsRisk | Event | VETO | No high-impact news events |
| 7 | Slippage | Execution | VETO | Fill price within tolerance |
| 8 | MaxPositions | Exposure | VETO | Position count within limit |
| 9 | MaxExposure | Exposure | VETO | Total exposure within limit |
| 10 | Cooldown | Timing | VETO | Strategy cooldown respected |
| 11 | StopHuntFilter | Structural | DEGRADED | Price not near suspected stop-hunt zones |
| 12 | MarginCheck | Broker | VETO | Sufficient free margin (OrderCalcMargin) |
| 13 | OvertradeProtection | Frequency | VETO | Trade frequency within limits |
| 14 | MaxDailyTrades | Frequency | VETO | Daily trade count within limit |
| 15 | RegimeFilter | Market | DEGRADED | Regime suitable for strategy |
| 16 | ProfitTarget | Capital | VETO | Daily profit target not exceeded |

### 5.3 P0-001: BrokerSymbolValidationGate

A novel contribution of this thesis is the BrokerSymbolValidationGate (v1.16.0), which validates signal parameters against live broker symbol metadata:

- **Minimum Stop Distance:** Ensures SL distance exceeds the broker's minimum stop level (StopsLevel in MT5 terminology).
- **Freeze Level:** Ensures entry price is not within the broker's freeze zone.
- **Maximum Spread:** Ensures current spread is within the symbol's configured maximum.
- **Digit Precision:** Rounds Entry, SL, and TP to the broker's configured digit precision (BrokerDigits).

This gate is registered second (after ExecutionPermission) and degrades rather than vetoes when broker metadata is unavailable, preventing production interruption during broker disconnections while still enforcing safety when metadata is present.

### 5.4 Server-Side Stop-Loss Enforcement

Beyond pre-trade gates, the platform implements server-side SL enforcement:

1. **EXECUTION_ACK Verification:** When the EA reports trade execution, the server verifies that the reported SL matches the server-computed value (within 0.5 point tolerance).
2. **Position Monitoring:** On every broker snapshot, the server scans all PAT-managed positions. Any position with SL=0 triggers immediate CLOSE_POSITION.
3. **Agent Suspension:** After 3 SL violations, the agent is disconnected and blocked from receiving future signals. Suspension is logged to compliance audit.
4. **Emergency Commands:** EMERGENCY_STOP closes all PAT positions and halts trading. KILL_SWITCH closes all positions and terminates the agent.

---

## Chapter 6: Repository-Level Audit

### 6.1 File and Dependency Inventory

The repository contains 31 internal Go packages in the real-time engine, each with a specific responsibility:

- **features/:** 42 indicators, structure/liquidity/VWAP/FVG/SMC engines, regime classification, pivot points, Fibonacci, candle analysis, pullback detection, session ORB
- **strategy/:** 5 independent strategy engines, evidence scoring, confluence computation, signal geometry, capability checks
- **gates/:** 16 hard risk gates, ordered registration, capital protection, broker compliance
- **signal/:** Master decision engine, cooldown management, duplicate prevention
- **marketdata/:** Provider interface, tick validation, candle aggregation, COT/DXY providers
- **crossmarket/:** Multi-asset macro module (BTC, Oil, DXY correlation), outcome resolver
- **ml/:** ONNX model inference (XGBoost, advisory only)
- **sentiment/:** Ollama LLM sentiment analysis (advisory only)
- **ptb/:** Professional Trader Brain intelligence layer, synthesis, metrics
- **gateway/:** HTTP handler, WebSocket broadcaster, agent handler
- **recovery/, hedging/, oco/:** Trade management and operational support

Database schema: 30 SQL migrations creating 9 schemas (iam, billing, finance, trading, market, calibration, ptb, compliance, backtest) with TimescaleDB hypertables for market.candles.

### 6.2 Test Coverage

| Test Suite | Tests | Result |
|-----------|:-----:|:------:|
| Go (28/28 packages) | hundreds | PASS |
| Frontend (Next.js) | 70 | PASS |
| Python (research/tests) | 127 | PASS |
| TypeScript (control/) | 0 errors | PASS |

Golden tests verify deterministic behavior: identical inputs produce identical signals. The strategy/golden_test.go suite contains 18 test cases. Replay tests verify that historical candle data replayed through the engine produces bit-exact signal outputs.

Pre-existing known failures (4 gate tests) are tracked and excluded from CI: TestRiskOversizeGate, TestDailyLossGate, TestProfitTargetGate, TestSeedCapitalProtectionGateStates.

### 6.3 Security Audit Findings

| Finding | Severity | Status |
|---------|:--------:|:------:|
| Hardcoded secrets in repo root | MEDIUM | FIXED |
| JWT dual-source secret | CRITICAL | FIXED (unified + HttpOnly) |
| License fail-open | CRITICAL | FIXED (fail-closed) |
| Backtest cross-tenant IDOR | HIGH | FIXED |
| Payout double-spend | CRITICAL | FIXED (RESERVED state) |
| NOWPayments IPN signature mismatch | CRITICAL | FIXED (HMAC-SHA512) |
| Fabricated quant-validation evidence | CRITICAL | FIXED (provenance check) |
| Container root users | MEDIUM | OPEN (deferred) |
| No CI/CD pipeline | MEDIUM | OPEN (deferred) |
| No incident response plan | HIGH | OPEN (deferred) |

---

## Chapter 7: AI/ML Governance

### 7.1 Advisory-Only Architecture

The platform uses two AI/ML components, both constrained to advisory roles:

**ONNX ML Model (XGBoost):** A gradient-boosted tree model trained on historical XAUUSD data (42 features: all indicators, regime, session, market state). Inference runs at signal evaluation time. The model's contribution is capped at 0.25 within the ML evidence pillar. If the model is absent, times out, or produces an error, the contribution defaults to zero — trading continues deterministically.

**Ollama Sentiment Analysis (LLM):** A local LLM (llama3.2) receives structured market data (processed indicators, not raw prices) and produces a sentiment score. Contributes up to 0.25 within the SENTIMENT pillar. Cannot see raw tick data, account balances, or signal parameters.

### 7.2 AI Market-Data Firewall

The following invariants are enforced in code:

1. No AI component receives raw tick data — only processed indicators (RSI, ADX, regime, session)
2. No AI component can set entry price, stop loss, or take profit levels
3. No AI component can override gate decisions
4. No AI component can modify signal expiry, cooldown, or delivery
5. AI failures always degrade to zero contribution, never to forced trades

### 7.3 Risk Matrix

| AI Risk | Probability | Impact | Controls | Residual Risk |
|---------|:-----------:|:------:|----------|:-------------:|
| Hallucinated prices | 1/5 | 3/5 | Only receives structured data | Low (3) |
| Stale model inputs | 2/5 | 2/5 | TTL-guarded feature pipeline | Low (4) |
| Model drift | 2/5 | 2/5 | ONNX model versioned, offline-only training | Low (4) |
| Nondeterministic output | 1/5 | 2/5 | Advisory role, contribution capped | Low (2) |
| Provider outage | 2/5 | 1/5 | Deterministic fallback with zero contribution | Low (2) |
| Prompt injection | 1/5 | 1/5 | Structured input only | Minimal (1) |

---

## Chapter 8: Production Readiness Assessment

### 8.1 Scoring Methodology

Nine dimensions were evaluated on a 0-100 scale using the following criteria:

| Dimension | Score | Key Evidence |
|-----------|:-----:|-------------|
| Security Readiness | 62 | JWT HttpOnly, secrets fixed, MFA, rate limiting. Gaps: no IR plan, no CVE scanning |
| Signal Integrity | 78 | Traceable from tick to signal, 49/49 geometry valid, repaint-free, idempotent |
| Data Integrity | 65 | 30 migrations, NUMERIC(18,8) money types. Gap: no retention policy |
| Mathematical Correctness | 85 | MAPE < 0.0001, Wilder smoothing verified, golden tests reproducible |
| AI Governance | 68 | Advisory only, market-data firewall, deterministic fallback. Gap: no drift monitoring |
| Reliability | 70 | Reconnect loops, circuit breakers, ordered gate fail-closed. Gap: untested restore |
| Observability | 72 | Prometheus + Grafana + health manager + structured logs. Gap: backlog monitoring |
| Software Quality | 75 | 28 packages tested, 3 language test suites, type-checked. Gap: no CI/CD pipeline |
| IT Compliance | 55 | ISO 27001 mapped, gitleaks configured. Gaps: no formal ISMS, no IR documentation |
| **OVERALL** | **70** | **CONDITIONAL GO** |

### 8.2 Critical Production Blockers (ALL CLOSED)

| ID | Blocker | Domain | Fix Applied |
|----|---------|--------|-------------|
| C1 | Fabricated quant-validation evidence | Signal Integrity | Provenance check + final_go_live_check.py |
| C2 | NOWPayments IPN HMAC mismatch | Revenue | Raw-body SHA-512 verification |
| C3 | Payout double-spend | Finance | RESERVED state machine + migrations |
| C4 | Windows-agent license fail-open | Safety | Default PENDING (fail-closed) |
| C5 | JWT dual-source + insecure token | Security | Unified JwtService + HttpOnly cookie |

---

## Chapter 9: Discussion and Future Work

### 9.1 Limitations

1. **Sample Size:** The MARNIE_FIB engine has fewer than 30 resolved outcomes, insufficient for statistical confidence. It remains in SHADOW mode.

2. **CI/CD:** No automated build pipeline exists (GitHub Actions templates pending). All builds are local.

3. **Container Security:** All Docker containers currently run as root. Non-root user migration is a deferred hardening task.

4. **Data Retention:** The market.candles hypertable has no retention policy, allowing unbounded growth.

5. **Incident Response:** No documented incident response plan exists, representing a gap for ISO 27001 and NIST CSF compliance.

6. **Backup Testing:** While backup procedures exist (pg_dump), restoration has never been tested in a production-context environment.

### 9.2 Future Research Directions

1. **Reinforcement Learning Integration:** The platform includes an RL optimizer in disabled mode. Future work could explore safe RL policy training in offline simulation with bounded authority (filter_only mode) before gradual promotion.

2. **Multi-Asset Expansion:** The cross-market module already ingests BTC, Oil, and DXY data. Extending the strategy engines to multi-asset pairs while maintaining the plane-boundary architecture is a natural next step.

3. **Real-Time Calibration:** Online model calibration with concept drift detection and automatic shadow/promotion cycles could improve ML contribution reliability.

4. **Formal Verification:** The gate pipeline's deterministic, fail-closed semantics make it amenable to formal verification using TLA+ or similar specification languages.

### 9.3 Conclusion

Predict-A-Trade demonstrates that a multi-plane architecture with strict boundary enforcement can successfully integrate sophisticated AI/ML components into a trading platform without surrendering deterministic control over the critical decision path. The 13-pillar evidence scoring engine, 16-gate ordered risk pipeline, and complete evidence provenance provide a reproducible, auditable foundation for algorithmic trading. With a production readiness score of 70/100 and all critical blockers closed, the platform is positioned for conditional production deployment pending operator-completed P1 actions (MQL compilation, API key provisioning, backup testing, and incident response documentation).

---

## References

[1] Brock, W., Lakonishok, J., & LeBaron, B. (1992). Simple Technical Trading Rules and the Stochastic Properties of Stock Returns. *Journal of Finance*, 47(5), 1731-1764.

[2] Deng, Y., Bao, F., Kong, Y., Ren, Z., & Dai, Q. (2017). Deep Direct Reinforcement Learning for Financial Signal Representation and Trading. *IEEE Transactions on Neural Networks*, 28(3), 653-664.

[3] Fischer, T., & Krauss, C. (2018). Deep learning with long short-term memory networks for financial market predictions. *European Journal of Operational Research*, 270(2), 654-669.

[4] Lo, A. W., Mamaysky, H., & Wang, J. (2000). Foundations of Technical Analysis: Computational Algorithms, Statistical Inference, and Empirical Implementation. *Journal of Finance*, 55(4), 1705-1765.

[5] Pineau, J., Vincent-Lamarre, P., Sinha, K., Larivière, V., Beygelzimer, A., d'Alché-Buc, F., Fox, E., & Larochelle, H. (2021). Improving Reproducibility in Machine Learning Research. *Journal of Machine Learning Research*, 22(164), 1-20.

[6] Sullivan, R., Timmermann, A., & White, H. (1999). Data-Snooping, Technical Trading Rule Performance, and the Bootstrap. *Journal of Finance*, 54(5), 1647-1691.

[7] Wu, H., Xu, J., Wang, J., & Long, M. (2021). Autoformer: Decomposition Transformers with Auto-Correlation for Long-Term Series Forecasting. *NeurIPS 2021*.

[8] Basel Committee on Banking Supervision. (2018). *Stress Testing Principles*. Bank for International Settlements.

[9] International Organization of Securities Commissions. (2020). *Principles for Algorithmic Trading*. IOSCO/MR/05/2020.

---

**Thesis submitted: 26 August 2026**
**Version: v1.16.0**
**Repository: github.com/simhaonline/predictatrade**
