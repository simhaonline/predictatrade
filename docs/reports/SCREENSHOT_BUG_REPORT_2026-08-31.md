# Predict-A-Trade — Screenshot Bug & Improvement Report

- **Date:** 2026-08-31
- **Source:** `screenshot/Screenshot_1.png` … `Screenshot_22.png` (22 captures of the live admin + Windows Agent UI)
- **Method:** OCR extraction (tesseract) cross-referenced with source code (`frontend/src`, `control/src`, `realtime/`) for root-cause accuracy.
- **Scope:** Bug identification + fix/improvement recommendations **only**. No code changes were made.
- **Severity key:** 🔴 P0 (safety/security/financial/data-quality/acceptance gate) · 🟠 P1 (functional breakage / wrong data) · 🟡 P2 (UX/layout / polish / feature-gap).

---

## Executive Summary

22 screenshots were reviewed. The platform is broadly online (Go engine, control plane, frontend, Windows Agent all reachable), but the review surfaced **1 P0 crash**, **several P0/P1 production-readiness gaps**, and a consistent set of layout/overflow defects across admin tables.

The most serious items:

1. 🔴 **`/admin/macro-news` page crashes** with `Cannot read properties of undefined (reading 'map')` — the frontend calls the wrong backend endpoint (`/news` instead of `/admin/macro-news`).
2. 🔴 **Backup & DR is `CONFIGURED_NO_ARCHIVE`** — WAL archiving off, no off-host backup, all required keys unconfigured. No production DR posture.
3. 🔴 **Risk Center shows Emergency Stop and Data-quality/freshness as `Degraded`** — two P0 safety gates degraded.
4. 🟠 **Signal Accuracy lists a phantom strategy `EQFE`** (an intelligence engine, not one of the 4 canonical strategies) and shows catastrophic real win rates (0–32.5%).
5. 🟠 **Duplicate commission ledger rows** (identical $125.82 entries, both `Pending`) — idempotency concern.
6. 🟡 **Recurring table/text overflow** across Scoring Board, License Management, Device Auth, Payments, Market Data, Macro Intelligence.

---

## Per-Screenshot Findings & Fixes

### Screenshot 1 — `/admin/dashboard` (Real-Time Console) 🟠 P1
**Observed:** Score / Expectancy / Quality all show `—` or `=`. Candidates/Qualified = `0/—`. Scoring board candidates `0`. "Last eval 3 minutes ago" (stale). All strategy panels `NO-TRADE` with `Score 12.5` / `47.6`.

**Root-cause hypothesis:** Realtime WS either not hydrating these metric fields, or frontend field mapping is incomplete (empty-string → `—`).

**Fix / Improve:**
- Verify the Go engine is emitting Score/Expectancy/Quality fields on the admin stream and that the frontend maps them (not just renders placeholder `—`).
- "Last eval 3 minutes ago" with 0 candidates → confirm engine is actually receiving ticks; if genuine NO-TRADE, show the rejection reason(s) (the "Top reasons" panel is blank).
- Add a "stale" indicator when last-eval age > threshold (e.g. > 60s) per the SOW honest-state requirement.

---

### Screenshot 2 — `/admin/pipeline-monitor` 🟠 P1
**Observed:** Signals (5m) = **43**, Vetoed (5m) = **0**. Yet Screenshot 1 shows NO-TRADE everywhere.

**Root-cause hypothesis:** Either the hard-risk veto counter is not being incremented (gate instrumentation broken) — 43 candidates with **zero** vetoes is implausible when every strategy returns NO-TRADE — or "vetoed" is defined differently than "rejected by a hard gate".

**Fix / Improve:**
- Audit the `vetoed` metric source. NO-TRADE outcomes must be attributable to a named gate (spread, news/session, margin, TTL, etc.).
- Ensure Vetoed count + "Top reasons" reconcile with the NO-TRADE decisions shown on the console.

---

### Screenshot 3 — `/admin/agent-mesh` 🟡 P2
**Observed:** "Data Feed Health: HEALTHY", "Agents online: true", server time present. Content is very sparse.

**Fix / Improve:** Likely an honest minimal state, but confirm the mesh detail (per-agent identity, role, last-heartbeat) is intentionally compact vs. missing. Add per-agent heartbeat age if absent.

---

### Screenshot 4 — `/admin/scoring-board` 🟡 P2
**Observed:** BID 4421.37 / ASK 4421.71 → spread 0.34; Regime RANGE; NO-TRADE in London/NY overlap. Right side of card is garbled/cut off (overflow).

**Fix / Improve:**
- Consistent with risk gates (range + spread) — **not a logic bug**, but the table/card overflows its container on this viewport. Constrain column widths / add `overflow-x-auto` + `truncate` so values don't bleed.
- Display the rejection reason for the NO-TRADE (currently only "Status: NO-TRADE" with no reason).

---

### Screenshot 5 — `/admin/devil-liquidity` 🟡 P2
**Observed:** Engine ENABLED, **SHADOW mode** (observation only, no live signal gating yet). Candles processed 9, marks 0.

**Fix / Improve:**
- SHADOW mode is acknowledged, but per SOW the Devil Liquidity feature needs a defined graduation path from shadow → live gating (with golden fixtures + replay validation). Document the criteria/owner and a target state.
- "Candles processed: 9" after extended uptime suggests low candle throughput on the monitored symbol/timeframe — verify the candle pipeline is fed correctly.

---

### Screenshot 6 — `/admin/signal-accuracy` 🟠 P1
**Observed:**

| # | Strategy | Total | Resolved | Wins | Losses | Win% | Avg P&L | Total P&L |
|---|---|---|---|---|---|---|---|---|
| 1 | Ultra Scalping | 40 | 40 | 13 | 27 | 32.5% | 0.11 | 4.58 |
| 2 | Standard Scalping | 47 | 47 | 6 | 41 | 12.8% | -2.36 | -111.08 |
| 3 | Standard Swing | 8 | 8 | 0 | 8 | 0.0% | -5.07 | -40.63 |
| 4 | **EQFE** | 5 | 5 | 0 | 5 | 0.0% | -1.05 | 5.27 |

**Bugs:**
1. **`EQFE` is not a strategy.** The 4 canonical products are `STANDARD_SCALPING`, `ULTRA_SCALPING`, `STANDARD_SWING`, `TREND_SWING`. `EQFE` is an intelligence engine/regime label. `trading.trade_results.strategy_id` is carrying a non-strategy value, or the strategy-id→display-name map is leaking engine names. **Data-integrity bug.**
2. **`TREND_SWING` is missing** from the table entirely (only 4 rows, one of which is the bogus `EQFE`).
3. Win rates are catastrophically low (real outcomes). Honest, but flags strategy-quality / calibration problems the SOW requires to be addressed (not relabeled as limitations).

**Fix / Improve:**
- Constrain `strategy_id` to the 4 canonical enum values; add a DB CHECK constraint + reject/redirect any trade_result with an unknown strategy_id.
- Fix the strategy display-name mapping so engines/regimes never appear as strategies.
- Investigate why `TREND_SWING` has zero trade_results.
- Treat the 0–12.8% win rates as a quant-validation trigger: walk-forward/OOS + calibration review per `xauusd-quant-validation` skill before any live enablement.

---

### Screenshot 7 — `/admin/risk-center` 🔴 P0
**Observed:** Two gates marked **Degraded**: **Data quality / freshness** and **Emergency stop**. All others Active.

**Root-cause hypothesis:**
- *Emergency stop degraded* is a P0 safety gate (AGENTS.md non-regression precedence). A degraded emergency-stop path means a KILL_SWITCH/EMERGENCY_STOP may not reach all agents/EAs.
- *Data quality/freshness degraded* — stale or low-quality feed flagged by the engine.

**Fix / Improve:**
- Investigate and restore Emergency Stop to Active: verify Windows Agent command channel (CLOSE_POSITION / EMERGENCY_STOP / KILL_SWITCH) end-to-end with a paper drill.
- Restore Data quality/freshness: trace the degraded signal to its feed/lag/freshness threshold and remediate.
- Per SOW, a P0 gate in Degraded state blocks production GO — do not mark the system PASS while these are degraded.

---

### Screenshot 8 — `/admin/subscriptions` 🟠 P1
**Observed:** `forexdks@gmail.com` shows status **"Incomplete"** with no Period Start; other rows show `Se` (truncated) for period end and mixed auto-renew.

**Bugs:**
1. **"Incomplete" is a non-standard subscription state** with no visible explanation. Likely a provisioning/billing-webhook that didn't complete (charge succeeded but entitlement not provisioned).
2. Period-end column renders `Se` (truncated "Sep 30") — column too narrow.

**Fix / Improve:**
- Define and document all subscription statuses; surface a reason/recovery action for `Incomplete` (retry provisioning, link to invoice).
- Widen the date columns / use responsive truncation.
- Add reconciliation: ensure a paid invoice always results in an Active entitlement or a flagged `Incomplete` with an owner action.

---

### Screenshot 9 — `/admin/licenses` 🟡 P2
**Observed:** Table truncated: header `EXPIR…`, `MAX DEVICES` empty, `EXPIRY` shows only `Aug 3`. Five Elite/Free rows visible.

**Fix / Improve:**
- Fix column overflow (expiry/max-devices cut off). Use `whitespace-nowrap` + horizontal scroll or a details drawer.
- Ensure `MAX DEVICES` is populated for every license (currently blank).

---

### Screenshot 10 — `/admin/device-auth` 🟡 P2
**Observed:** Garbled header text, mixed Online/Offline/Online connection states, "Limited" P/L $0.00 / floating P/L $0.00, `ITS_LMASTER+LOCAL_COMPUTE` label.

**Fix / Improve:**
- Fix the header/label rendering (looks like overlapping or unescaped text).
- Verify device connection state accuracy (a device showing Online then Offline then Online across rows suggests flapping or stale snapshots).
- Clarify the "Limited" P/L panel semantics (is this per-device trade P&L? label it explicitly).

---

### Screenshot 11 — `/admin/mt-accounts` 🟡 P2
**Observed:** "No MT accounts linked anywhere yet."

**Fix / Improve:** Honest empty state (accounts appear only when a device activates with a bound MT account). Verify the MT-account binding flow actually works with a real device activation; until then this is expected-but-unverified.

---

### Screenshot 12 — `/admin/payments` 🟠 P1
**Observed:** Info banner text is garbled/overlapping: "Live view Tor FREETSeTS ts tte=garet tO 1100-13:00 GMT+3 on live.predictatrade.com." This is a maintenance-window / live-window banner whose text is unreadable.

**Fix / Improve:**
- Fix the banner text rendering (likely an unescaped template/variable interpolation or CSS overflow collision). The intended message (live USDT payment window 11:00–13:00 GMT+3 on live.predictatrade.com) must render cleanly.
- Confirm NOWPayments status, Stripe disabled-by-decision, and the ledger section below are all wired (ledger appears empty in the capture).

---

### Screenshot 13 — `/admin/payout-operations` 🟡 P2
**Observed:** Pending/Approved/Rejected/Total all `$0.00` / `0`; "No data found".

**Fix / Improve:** With commissions still `Pending` (Screenshot 14), zero payouts is plausibly correct (nothing has cleared→available→paid yet). Verify the payout eligibility pipeline (pending→cleared→available→paid) and that "Approved/Rejected" counts are wired, not just defaulted to 0.

---

### Screenshot 14 — `/admin/commission-operations` 🟠 P1
**Observed:** Two **identical** rows: `standard@predictatrade.com → neo.trade007@gmail.com`, `$125.82`, both `Pending`, dated **Aug 29** and **Aug 23**.

**Bugs:**
1. **Duplicate commission ledger entries** for the same recipient/source/amount — idempotency failure in commission creation (a referral event likely processed twice).
2. Both are `Pending` for 2–8 days — no hold→release lifecycle progression.

**Fix / Improve:**
- Add idempotency keys / dedupe on commission creation (source_event_id + recipient + period) so a replayed webhook cannot double-post.
- Reconcile: reverse one of the duplicate rows with a compensating record (do **not** rewrite history — per AGENTS.md financial-integrity rules).
- Drive the lifecycle: define when Pending commissions are reviewed for hold/release.

---

### Screenshot 15 — `/admin/market-data` 🟠 P1
**Observed:** Top indicator table is garbled/cut off (`No 29.2000 LUNDUI_NEWTURK_UVERLHHH4O…`). Feed Monitoring panels (Divergence, Tick Rate, Latency, Candle Health, Backfill, Macro Calendar) all **"Monitoring pending"**. The page itself states: *"No dedicated feed-health / divergence / tick-rate / latency / candle-health / backfill endpoint exists. These panels show the intended schema only and must not be interpreted as live metrics."*

**Bugs:**
1. Indicator snapshot table overflows / mis-renders.
2. Feed-health monitoring is **not implemented** (acknowledged placeholder).

**Fix / Improve:**
- Implement the feed-health endpoints (divergence, tick-rate, latency, candle-health, backfill) in the Go engine, or keep a visible "NOT IMPLEMENTED — schema only" badge (currently only a single inline note; the tiles still imply live data).
- Fix the indicator table layout so symbol/regime/indicator values are legible.
- Per SOW market-data-truth: capability absence must degrade quality or cause NO-TRADE — make the absence explicit on the dashboard.

---

### Screenshot 16 & 17 — `/admin/macro-news` 🔴 P0 (CRASH)
**Observed (both captures):** `Something went wrong — Cannot read properties of undefined (reading 'map')`. The page throws and the error boundary takes over.

**Root cause (confirmed in source):**
- `frontend/src/app/(admin)/admin/macro-news/page.tsx` calls `customInstance.get("/news")`, which resolves to `/api/v1/news`.
- The NestJS control plane route is `/api/v1/admin/macro-news` (`AdminExtrasController @Get('macro-news')` under global prefix `api/v1`). There is **no `/api/v1/news` route in the control plane.**
- The realtime engine has a `/api/v1/news` handler but it is a different service/host and returns a different shape — so the frontend receives a payload without an `items` array, then `data.items.map(...)` throws.

**Fix / Improve:**
- Change the frontend call to the correct endpoint: `/admin/macro-news`.
- Add a defensive guard: `(data?.items ?? []).map(...)` and render `data?.note` when `items` is empty, so a shape mismatch degrades gracefully instead of crashing the whole page.
- Add a contract test asserting the frontend query path matches a real control-plane route (catches this class of bug).

---

### Screenshot 18 — `/admin/macro-intelligence` 🟠 P1
**Observed:** Macro score +45, confidence 0%, data quality Connected, regime MIXED, activation NOT ELIGIBLE. Driver table: `DXY` row has **blank Status** (others "Connected"), impact `-100.0`, eff. weight `0.00`, direction BEARISH. Bottom row garbled (`A400 RULLISH`).

**Bugs:**
1. **DXY driver Status is blank** — inconsistent with other connected drivers; either the driver is down (should show a status) or the status field isn't populated.
2. `confidence 0%` but `data quality Connected` and macro score `+45` — confidence 0 with a non-zero score is contradictory unless confidence is unbuilt; label it "unbuilt" not "0%".
3. Layout overflow at the bottom (`A400 RULLISH` = likely ATR + Bullish colliding).

**Fix / Improve:**
- Populate/validate driver Status for every row; fail closed (show Disconnected) rather than blank.
- Resolve the confidence=0 vs score=+45 contradiction (calibration not yet built → say so).
- Fix bottom-row table overflow.

---

### Screenshot 19 — `/admin/backup-dr` 🔴 P0
**Observed:** Status `CONFIGURED_NO_ARCHIVE` — "Backup configuration present but required keys are not configured." All components unconfigured:

| Key | Value | Configured |
|---|---|---|
| archive_command | (empty) | No |
| archive_mode | off | No |
| max_wal_senders | 0 | No |
| off_host_backup_bucket | (empty) | No |
| off_host_backup_encryption | none | No |
| off_host_backup_provider | (empty) | No |

**Bugs:** No WAL archiving, no off-host backup, no encryption, no restore-test record. This is a **P0 production reliability gap** (AGENTS.md: backup/restore is mandatory for production GO).

**Fix / Improve:**
- Enable `archive_mode = on`, set a real `archive_command` (e.g. `wal-g`/`pgBackRest` to object storage), bump `max_wal_senders`.
- Configure off-host backup provider + bucket + encryption (SSE/CSE).
- Run and record a restore test; surface "Last Archived (WAL)" timestamp + last restore-test result on this page.
- Until then, this page is a NO-GO evidence item.

---

### Screenshot 20 — Windows Agent status page 🟡 P2
**Observed:** `MT4 OFFLINE`, `MT5 CONNECTED`, Backend data WS CONNECTED, candles delivered 630580, **clock drift 1158 ms**, auto-refresh 5s.

**Fix / Improve:**
- **Clock drift 1158 ms** is > 1s — investigate NTP sync on the Windows host; high drift can affect candle/session/fix timing (internal time truth is UTC per SOW).
- MT4 OFFLINE is acceptable if only MT5 is in use; confirm MT4 is intentionally decommissioned vs. an unplugged terminal.

---

### Screenshot 21 — `/admin/releases` 🟢 OK (minor)
**Observed:** `windows-agent-master` 1.2.44 STABLE, `windows-agent-client` 1.2.44 STABLE, both Active, rollback Optional.

**Fix / Improve:** No defect. Consider distinguishing `master` vs `client` channel/rollback policy if they should differ.

---

### Screenshot 22 — `/admin/settings` (MFA) 🔴 P0
**Observed:** Authentication **"Inactive"**, banner "MFA setup initiated". Admin MFA not yet enrolled.

**Fix / Improve:**
- Per AGENTS.md, **MFA is required** for admin. Complete MFA enrollment for `admin@predictatrade.com` and flip Authentication to Active.
- Until MFA is active, admin access is below the security baseline — a NO-GO item.

---

## Cross-Cutting Issues

### CC-1 — Admin table/card text overflow (recurring) 🟡 P2
Affects: Scoring Board (4), License Management (9), Device Auth (10), Payments (12), Market Data (15), Macro Intelligence (18).
**Fix:** Apply a shared admin-table style: fixed min-widths, `overflow-x-auto`, `whitespace-nowrap`, `truncate` with tooltips, and responsive column hiding. This single fix resolves most "garbled" captures.

### CC-2 — Honest empty/degraded states vs. silent placeholders 🟠 P1
Several panels (feed monitoring, payout counts, MT accounts) show zeros/"pending" without distinguishing "not implemented" from "implemented-but-empty". Per SOW, these must be unmistakably labeled (demo/replay/pending/NOT-IMPLEMENTED) and unable to be misread as live metrics.

### CC-3 — Endpoint contract drift 🟠 P1
The macro-news crash is the visible symptom of frontend↔backend route drift. Recommend a generated/contract-tested route map (frontend query paths ↔ control-plane `@Get/@Post` handlers) so mismatches fail in CI, not in production.

### CC-4 — Financial ledger integrity 🟠 P1
Duplicate commission rows (14) + zero payout reconciliation (13) + "Incomplete" subscription (8) together indicate the financial ledger needs an idempotency + reconciliation pass (exact-decimal, compensating records, no history rewrites).

---

## Recommended Fix Priority

| Pri | Item | Screens | Gate impact |
|---|---|---|---|
| 🔴 P0 | Fix macro-news endpoint mismatch + defensive guard | 16, 17 | Acceptance (page crash) |
| 🔴 P0 | Configure Backup & DR (WAL archive + off-host + restore test) | 19 | Production GO |
| 🔴 P0 | Restore Emergency Stop + Data-quality gates to Active | 7 | Safety non-regression |
| 🔴 P0 | Complete admin MFA enrollment | 22 | Security baseline |
| 🟠 P1 | Fix strategy_id integrity (remove EQFE, add TREND_SWING, enum CHECK) | 6 | Quant/data integrity |
| 🟠 P1 | Dedupe + reconcile commission ledger (idempotency) | 14 | Financial integrity |
| 🟠 P1 | Investigate "Vetoed 0" vs all-NO-TRADE gate instrumentation | 2 | Risk-gate correctness |
| 🟠 P1 | Fix payments banner text rendering | 12 | Functional |
| 🟠 P1 | Implement/feed-health endpoints OR label NOT-IMPLEMENTED clearly | 15 | Market-data truth |
| 🟠 P1 | Macro-intelligence: blank DXY status + confidence=0 contradiction | 18 | Data quality |
| 🟠 P1 | Resolve "Incomplete" subscription provisioning | 8 | Financial integrity |
| 🟡 P2 | Shared admin-table overflow fix (CC-1) | 4,9,10,12,15,18 | UX |
| 🟡 P2 | Windows Agent clock-drift remediation (NTP) | 20 | Time-truth |
| 🟡 P2 | Devil Liquidity shadow→gating graduation criteria | 5 | Feature completeness |

---

## GO / NO-GO Summary

**NO-GO.** Four P0 items block a production GO under the AGENTS.md definition-of-done / safety-precedence rules:

1. macro-news page crash (acceptance gate failed — visible broken page),
2. Backup & DR `CONFIGURED_NO_ARCHIVE` (no production DR posture),
3. Emergency Stop + Data-quality gates `Degraded` (P0 safety gates not green),
4. Admin MFA `Inactive` (security baseline not met).

All four must be remediated and re-verified before any PASS status. P1 items (strategy_id integrity, commission idempotency, gate-veto instrumentation, feed-health) should follow on the same release but do not, by themselves, block — except where they touch financial-integrity or data-quality gates.

*No code was modified in the production of this report.*
