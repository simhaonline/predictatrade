# Full Macroscopic Audit Prompt for Predict-A-Trade Project

**Objective:**  
Perform a comprehensive, end‑to‑end audit of the Predict‑A‑Trade system to confirm that:

- All trading signals are generated **in real‑time** using **genuine data from the Master Node feed**.
- All **candles are calculated properly** (OHLC, timeframes, volume, etc.).
- **Signal accuracy is ≥ 98%** when using all indicators and the signal engine.
- There are **no fake, stale, or circular data loops**.
- All **backend and frontend APIs** are functioning correctly.
- Signals are **delivered to MT clients properly** (MetaTrader terminals, mobile, web, etc.).
- There is **no unused logic, function, indicator, or engine** that is missing from the scoring mechanism or calibration pipeline.
- A **real‑time Signal Engine Calibration process** exists and is actively improving accuracy toward 100%.

This audit must be **macroscopic** – it should examine the entire system from data ingestion to signal delivery, including all internal components, data flows, and calibration mechanisms. The auditor must produce a detailed report with pass/fail status for each checklist item, evidence, and recommendations for any failures.

---

## 1. Real‑Time Signal Generation Verification

**1.1** Confirm that the signal generation pipeline runs **continuously** (or on a strictly scheduled tick‑by‑tick / candle‑close basis) and produces signals without manual intervention.

**1.2** Measure **end‑to‑end latency** from the moment a new tick/quote arrives from the Master Node feed to the moment a signal is generated and dispatched. Latency must be within the project’s defined real‑time threshold (e.g., < 50 ms for tick data, < 500 ms for candle‑close signals).

**1.3** Verify that the system does not use **historical data replay** or simulated feeds for live signal generation (unless explicitly in a backtesting mode). Live signals must be based on current market data.

**1.4** Check that all **timestamps** on generated signals match the actual time of data reception and processing. Discrepancies indicate stale or delayed data.

**1.5** Inspect the **scheduler / event loop** that triggers signal calculations. It must be event‑driven (e.g., on new tick, on new candle) and not based on fixed polling that may miss updates.

**1.6** Confirm that the system handles **data gaps** (e.g., broker disconnections) without producing signals based on missing or interpolated data that could be misleading.

---

## 2. Genuine Data from Master Node Feed

**2.1** Validate the **source of the Master Node feed**. Ensure it connects directly to the intended liquidity provider, exchange, or broker, and that no intermediate layer is altering or synthesizing data.

**2.2** Verify that the data received is **unmodified** by comparing a sample of ticks/candles against a trusted external reference (e.g., another data provider) for the same symbols and timeframes.

**2.3** Check for **data integrity** – all fields (symbol, bid, ask, volume, timestamp, etc.) are present and populated correctly. Missing or zero values in critical fields indicate feed problems.

**2.4** Examine the **connection health** to the Master Node. The system should have automatic reconnection logic, heartbeat monitoring, and failover to a secondary feed if necessary.

**2.5** Confirm that **no data is being generated internally** (e.g., random numbers, static values, or mock data) in the live environment. All live data must originate from the external Master Node.

**2.6** Inspect logs to ensure that every data packet consumed by the signal engine is traceable to the Master Node with proper sequence numbers or timestamps.

---

## 3. Candle Calculation Correctness

**3.1** Verify that **candle aggregation** is performed correctly for all supported timeframes (M1, M5, M15, H1, H4, D1, etc.). Compare a sample of generated candles against a known‑good reference (e.g., from the broker’s own historical data).

**3.2** Check that **OHLC values** (Open, High, Low, Close) are accurate. High must be the maximum price within the period, Low the minimum, Open the first tick, Close the last tick.

**3.3** Ensure that **volume** (if provided) is aggregated correctly (tick volume vs. real volume depending on the feed).

**3.4** Verify that candles are **closed and opened at the correct boundaries** (e.g., a new M5 candle starts exactly at 00:00:00, 00:05:00, etc., in the broker’s timezone).

**3.5** Test the **handling of incomplete candles** – the current forming candle must be updated in real time and not used in signal calculations until it is closed, unless the strategy explicitly uses forming candles.

**3.6** Check for **duplicate or missing candles** in the historical database. The sequence must be continuous without gaps or overlaps.

**3.7** Validate that the **candle calculation logic** is consistent across all parts of the system (signal engine, backtesting, UI displays) – no discrepancies between what the engine sees and what the user sees.

---

## 4. Signal Accuracy ≥ 98%

**4.1** Define the **accuracy metric** clearly. Typically, this is the percentage of signals that result in a profitable trade (or meet the strategy’s predefined success criteria) over a statistically significant sample.

**4.2** Review the **methodology** used to measure accuracy:
- Is it based on live forward testing, historical backtesting, or a combination?
- What is the sample size? It must be large enough (e.g., at least 1000 signals) to be statistically meaningful.
- Are all signals included, or only those that meet certain filters? All generated signals must be counted, not just cherry‑picked winners.

**4.3** Examine the **signal outcome tracking** system. Each signal must be tracked from generation to exit (take profit, stop loss, or timeout). The system must record:
- Entry price and time
- Exit price and time
- Profit/loss (in pips or percent)
- Whether the signal was successful according to the strategy’s definition (e.g., hit TP before SL)

**4.4** Compare the **measured accuracy** against the 98% target. If below, identify the root cause (e.g., poor indicator inputs, overfitting, data issues, execution slippage) and document required fixes.

**4.5** Verify that **accuracy is not inflated** by:
- Excluding losing signals from the stats
- Using hindsight or look‑ahead bias in backtesting
- Using a different time period for calibration than for live
- Applying filters after the fact that would not have been known at signal time

**4.6** Check the **confidence intervals** for the accuracy measurement. At 98% claimed accuracy, the sample must be large enough to have a narrow confidence interval.

**4.7** Ensure that the **accuracy calculation** is performed by an independent module or script that cannot be tampered with.

---

## 5. All Indicators & Engine Utilised (No Missing Components)

**5.1** Obtain the **complete list of indicators and engine components** that are supposed to be part of the signal generation process (from design documents, configuration files, or code).

**5.2** Verify that **every indicator** listed is actually:
- Implemented in the codebase
- Executed during signal generation
- Contributing to the final signal score (i.e., its output is used in the scoring formula)

**5.3** Check for **orphaned or commented‑out indicator code** that may have been left over from development but is no longer active. Such code should be removed or explicitly disabled.

**5.4** Confirm that **all engine components** (e.g., signal combiner, risk filter, trend filter) are called in the correct order and their outputs are incorporated into the final decision.

**5.5** Verify that no indicator is being bypassed due to configuration errors, incorrect parameter passing, or conditional logic that skips it.

**5.6** Inspect the **scoring function** (or final signal generation logic) to ensure it uses the outputs from all intended indicators. A missing indicator contribution would reduce the robustness and potentially the accuracy.

**5.7** Perform a **static code analysis** to identify functions or modules that are never called in the live path but are still present. These may indicate dead code or incomplete integration.

---

## 6. No Fake or Stale Data Loop

**6.1** Trace the **data flow** from the Master Node to the signal engine and then to the client delivery. Ensure there is no point where data is fed back into the input (which would create a loop).

**6.2** Check for **circular references** in databases or caches where stale data could be served repeatedly.

**6.3** Verify that **time‑sensitive data** (e.g., current price) is always fresh. Any data older than a few seconds (or the defined freshness threshold) must be flagged as stale and not used for live signals.

**6.4** Inspect the **caching mechanisms**:
- Are caches invalidated properly on new data?
- Is there a TTL (time‑to‑live) for cached items?
- Could a stale cache be served instead of live data?

**6.5** Examine the **message queues / event bus** to ensure that old events are not being replayed or processed out of order.

**6.6** Confirm that **no data is generated from the system’s own output** (e.g., using past signals to create future signals in a self‑referential way) unless it is an explicitly designed feedback loop for calibration, which must be controlled and not affect live signal generation directly.

**6.7** Test the system under **network disconnection** scenarios. The system should halt signal generation (or clearly flag that data is unavailable) rather than continuing with stale data.

---

## 7. Backend and Frontend API Health

**7.1** Enumerate all **backend API endpoints** (internal and external) and verify they respond correctly:
- Return expected status codes (200, 404, etc.)
- Response times within SLAs
- Proper error handling and logging

**7.2** Check **frontend API integration** – all calls from the frontend (web dashboard, mobile app) to the backend are successful and return the correct data.

**7.3** Test **authentication and authorisation** for API endpoints. Unauthorised access must be rejected.

**7.4** Verify **data consistency** between frontend and backend. For example, the same signal shown in the dashboard must match the signal sent to MT clients.

**7.5** Perform **load testing** (if applicable) to ensure the API can handle the expected number of concurrent clients without performance degradation.

**7.6** Monitor **API error rates** over a period (e.g., 24 hours). Any persistent errors indicate a problem.

**7.7** Confirm that **API versions** are compatible across all components.

---

## 8. Signal Delivery to MT Clients

**8.1** Identify all **delivery channels** to MetaTrader clients (e.g., push notifications via MetaQuotes ID, EA (Expert Advisor) that polls an API, copy trading server, etc.).

**8.2** Verify that **signals are being sent** to the correct client accounts and terminals. There must be a mapping between user accounts and their MT credentials.

**8.3** Check **delivery latency** – the time from signal generation to the client receiving the signal. This should be within acceptable limits (e.g., < 1 second).

**8.4** Ensure that the **signal payload** contains all necessary information: symbol, direction (buy/sell), entry price (or market), stop loss, take profit, lot size (if fixed), and expiry time if applicable.

**8.5** Test the **delivery mechanism** with a few test clients to confirm that signals are received and can be acted upon (e.g., an EA can automatically place trades).

**8.6** Verify that **failed deliveries** are logged and retried. The system should have a retry policy and alerting for persistent failures.

**8.7** Check that **client acknowledgements** (if implemented) are processed correctly to avoid duplicate signal sending.

**8.8** Confirm that **permissions** are respected – only authorised clients receive signals for the symbols they are subscribed to.

---

## 9. No Unused Logic, Function, Indicator, or Engine

**9.1** Perform a **code coverage analysis** (static or dynamic) to identify functions, classes, and modules that are never executed in the live environment.

**9.2** List all **indicators** and **engines** in the codebase and cross‑reference with the live configuration. Any indicator/engine that is not active must be either:
- Removed
- Explicitly disabled with documentation
- Kept for future use but clearly marked as inactive (and not affecting the audit)

**9.3** Check for **dead code** that may still be imported or compiled but has no effect on the output. This can lead to confusion during maintenance and may hide bugs.

**9.4** Ensure that **every active function** in the signal generation path is contributing to the final signal. There should be no “no‑op” functions that are called but produce no output.

**9.5** Review the **calibration pipeline** – all indicators and engine parameters that are supposed to be calibrated must actually be updated during the calibration process. Missing parameter updates indicate a broken calibration.

**9.6** Verify that the **scoring formula** is complete and uses all intended variables. Any variable that is declared but not used in the score calculation is a sign of incomplete integration.

---

## 10. Real‑Time Signal Engine Calibration Process

**10.1** Confirm that a **calibration process** exists and runs **automatically** (or on a schedule) to adjust engine parameters and indicator weights based on recent performance.

**10.2** Determine the **frequency** of calibration – it should be frequent enough to adapt to changing market conditions (e.g., daily, hourly, or after a certain number of signals). The audit must verify that calibration is not a one‑time event.

**10.3** Check the **inputs to the calibration** process:
- Recent signal outcomes (profit/loss)
- Market conditions
- Indicator performance metrics
- Any user‑defined targets

**10.4** Verify that the **calibration logic** actually updates the parameters used by the live signal engine. There should be a direct link between the calibration output and the engine’s configuration.

**10.5** Ensure that calibration does **not introduce look‑ahead bias**. It must only use data available up to the calibration time.

**10.6** Test the **effectiveness** of calibration by comparing pre‑calibration and post‑calibration accuracy over a rolling window. The accuracy should trend upward toward 100% (or at least remain above 98%).

**10.7** Check for **safe‑guards** in calibration – e.g., limits on parameter changes to avoid extreme values, rollback mechanism if accuracy degrades, and validation before deployment.

**10.8** Inspect the **feedback loop** from signal outcomes back into the calibration engine. It must be robust to missing or delayed outcome data.

**10.9** Confirm that the calibration process is **real‑time** in the sense that it can run concurrently with live signal generation without causing performance issues or data races.

---

## 11. Signal Engine Calibration for 100% Accuracy (Aspirational)

**11.1** Understand the project’s claim of achieving **100% accuracy**. In practical financial markets, 100% accuracy is extremely rare; the audit must assess whether the calibration process is realistically moving toward that goal or if it is an unrealistic target.

**11.2** Evaluate the **current accuracy trend** over time. If accuracy is already at 98% and calibration is active, the rate of improvement should be measurable.

**11.3** Check whether the calibration algorithm is **overfitting** to past data. If it adjusts parameters too aggressively to fit historical outcomes, live performance may suffer. Overfitting can temporarily boost backtested accuracy but degrade live accuracy.

**11.4** Verify that **risk management constraints** are not being sacrificed for higher accuracy (e.g., increasing stop loss excessively to avoid losses would artificially inflate win rate but reduce profitability).

**11.5** Ensure that the calibration process includes **cross‑validation** or walk‑forward analysis to prevent overfitting.

**11.6** Document any **limitations** that prevent reaching 100% accuracy (e.g., market noise, execution slippage, latency) and confirm that the system acknowledges them.

---

## 12. Overall System Coherence and Integration

**12.1** Produce a **system architecture diagram** (if not already available) showing all components: data feed, storage, signal engine, calibration engine, API server, frontend, and MT delivery. Verify that all components are connected as intended.

**12.2** Check for **single points of failure** that could disrupt real‑time operation (e.g., one server handling everything). Recommend redundancy if missing.

**12.3** Review **logging and monitoring** – the system should have comprehensive logs for all critical operations, and monitoring alerts for anomalies (data feed drop, high latency, accuracy drop, API errors).

**12.4** Perform a **security audit** – ensure that the Master Node connection, API endpoints, and client delivery are protected against unauthorised access and data tampering.

**12.5** Verify that **configuration management** is robust; all parameters (indicator settings, engine weights, calibration schedule) are stored in a centralised, version‑controlled configuration and not hard‑coded in multiple places.

**12.6** Check **documentation** – the project should have up‑to‑date documentation describing each component, its role, and how it contributes to signal generation and calibration.

---

## Audit Methodology and Reporting

- The auditor must use a combination of **static code analysis**, **dynamic testing**, **log review**, **performance monitoring**, and **comparison with external data sources**.
- All findings must be **evidenced** with logs, code snippets, screenshots, or test results.
- The final report must include a **summary table** with each checklist item, its status (Pass / Fail / Partial), and a brief explanation.
- For any failure or partial status, provide **specific recommendations** and a severity rating (Critical, Major, Minor).
- The audit should be repeatable – include the steps taken and data samples used.

---

**End of Audit Prompt**