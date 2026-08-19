# Predict-A-Trade — Remaining External Dependencies

**Updated:** 2026-08-18

This document lists only genuine external dependencies that cannot be resolved through code changes.
Software blockers have been resolved; remaining items are either external data, external configuration, or require physical runtime validation.

---

## 1. Windows Runtime Validation

**Status:** WINDOWS_RUNTIME_VALIDATION_REQUIRED

**What's Done:**
- Windows Agent source code complete (service, installer, updater)
- Cross-compilation succeeds: `GOOS=windows GOARCH=amd64 go build -o bin/PredictATradeAgent.exe`
- Windows Service registration/unregistration/start/stop code implemented
- Secure updater with checksum validation, atomic replacement, rollback
- Validation package generated (PowerShell script + checklist)

**What's Needed:**
- Execute the validation script on a real Windows 10/11 or Windows Server machine
- Test with real MT4/MT5 instances
- Verify service installation, startup, reboot persistence
- Verify backend connectivity, signal receipt, MT5 pipe communication
- Test update flow and rollback

**NOT a Software Blocker:** All code is implemented and compiles. This is a runtime validation gap only.

---

## 2. COT (Commitment of Traders) Report Provider

**Status:** IMPLEMENTED — FMP API key configured (HTTP 402 on current subscription tier)

**What's Done:**
- COT provider adapter implemented (`realtime/internal/marketdata/cot_provider.go`)
- Fetches COT data from Financial Modeling Prep API (stable + legacy v4 endpoints)
- Computes net positioning, percentile, z-score from historical reports
- Background refresh loop (every 6 hours — COT is weekly data)
- Fails safe: if API unavailable or HTTP 402, marks COT as UNAVAILABLE — never fabricates
- 8 unit tests covering not-configured, restricted, valid response, empty response, percentile, z-score, caching, redaction
- API key `FMP_API_KEY` set in env config

**What's Needed:**
- FMP subscription tier upgrade to include COT endpoints (current tier returns HTTP 402)
- OR use alternative COT data source (adapter is extensible)

**NOT a Software Blocker:** COT is an optional pillar (weight=0 by default). Does not block signal generation.

---

## 3. DXY (US Dollar Index) Provider

**Status:** IMPLEMENTED — Twelve Data API key configured, DXY computation verified

**What's Done:**
- DXY provider adapter implemented (`realtime/internal/marketdata/dxy_provider.go`)
- Computes ICE US Dollar Index from 6 component currencies via Twelve Data API:
  DXY = 50.14348112 × EUR/USD^(-0.576) × USD/JPY^(0.136) × GBP/USD^(-0.119) × USD/CAD^(0.091) × USD/SEK^(0.042) × USD/CHF^(0.036)
- Feeds DXY observations into PTB CorrelationEngine via `AddDXYObservation()`
- Background refresh loop (every 5 minutes — 6 API calls, within 8/min rate limit)
- Fails safe: if API unavailable, rate-limited (429), or components missing → DXY UNAVAILABLE
- 6 unit tests covering not-configured, rate limit, computation, missing component, caching, math
- API key `TWELVEDATA_API_KEY` set in env config

**Strategy Impact:**
- STANDARD_SWING: `macro_dxy_yield` is mandatory (weight 20) — DXY unavailable → NO-TRADE (correct)
- TREND_SWING: `macro_real_yield_dxy` is mandatory (weight 20) — DXY unavailable → NO-TRADE (correct)

**NOT a Software Blocker:** DXY is now wired and will produce real data once the API is accessible.

---

## 4. SMTP (Email — Password Reset, Verification)

**Status:** VERIFIED — SMTP connection tested and working

**What's Done:**
- SMTP configured: `mail.predictatrade.com:587` (STARTTLS), user `no-reply@predictatrade.com`
- Test email sent successfully via Python smtplib
- Password reset API returns correct generic response (no enumeration leak)
- Nodemailer provider validates insecure placeholder passwords in production
- Connection timeouts configured (10s connect, 30s socket)
- `verifyConnection()` method for startup health check

**What's Needed:**
- Cloud provider (netcup) must allow outbound SMTP — firewall was disabled for testing
- For production: ensure netcup firewall allows outbound 587/465 permanently

**NOT a Software Blocker:** SMTP is working. Emails are being sent.

---

## 5. True Volume / Order Flow Provider

**Status:** UNSUPPORTED_BY_DATA_SOURCE

**What's Done:**
- Volume Profile correctly labeled UNAVAILABLE (not fabricated from tick volume)
- Cumulative Delta correctly labeled UNAVAILABLE (not fabricated from price direction)
- Tick volume is clearly labeled as `tick_volume` (never mislabeled as real exchange volume)
- Feature readiness states expose: `UNSUPPORTED_BY_DATA_SOURCE`
- External provider interface architected for future activation

**What's Needed:**
- An approved external real-volume feed (e.g., CME/COMEX data subscription)
- An order flow/aggressor-side data provider
- These are NOT available from standard MT4/MT5 broker feeds

**NOT a Software Blocker:** These features have weight=0 and are correctly represented as unavailable.

---

## 4. Code-Signing Certificate (Optional)

**Status:** EXTERNAL_CONFIGURATION_REQUIRED (optional)

**What's Needed:**
- Windows code-signing certificate for agent binary distribution
- Used for updater signature verification (architecture supports it)
- Without it, the updater uses SHA-256 checksum validation only

**NOT a Software Blocker:** Checksum validation provides integrity. Code signing adds trust level.

---

## Items NOT Listed Here (Because They Are Resolved)

- ~~POOR_RR~~ — CORRECT_SAFETY_BEHAVIOR (risk gate working correctly)
- ~~UNCLEAR_STRUCTURE~~ — RESOLVED_BY_HISTORY_BOOTSTRAP
- ~~Parabolic SAR~~ — FIXED (implemented + tested)
- ~~Ichimoku~~ — FIXED (implemented + tested)
- ~~Stochastic RSI~~ — FIXED (implemented + tested)
- ~~Fibonacci~~ — FIXED (implemented + tested)
- ~~Daily/Weekly Pivots~~ — FIXED (implemented + tested)
- ~~OBV/Tick Volume/BB Width Z-scores~~ — FIXED (rolling stats engine)
- ~~Signal Cooldown~~ — FIXED (Valkey-based)
- ~~Duplicate Signal Prevention~~ — FIXED (fingerprinting)
- ~~Market History Bootstrap~~ — FIXED
- ~~Structure Engine Look-ahead Bias~~ — FIXED (fractal-based)
- ~~Feature Readiness States~~ — FIXED
- ~~Windows Agent Code~~ — FIXED (cross-compiles, needs runtime validation)
