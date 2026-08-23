# Predict-A-Trade — Subscription Plans, Features & Limitations

> **Source of truth:** this document is derived directly from the deployment's
> database migrations, not from external assumptions.
>
> - `database/migrations/003_plans_billing_licensing.sql` — plan/subscription/license schema, v1 plan seeds and entitlements.
> - `database/migrations/024_subscription_referral_v3.sql` — FREE plan, revised pricing, per-plan feature/limit matrix, referral rates, and commercial feature flags.
> - `database/migrations/029_plan_based_test_users.sql` — plan-based seed users (Free/Standard/Pro/Elite).
>
> All monetary fields are `DECIMAL(18,8)`, all times `TIMESTAMPTZ` (UTC). Symbols: XAUUSD.

---

## 1. Deployment Status (read this first)

The deployment defines **four active plan codes** plus one legacy alias:

| Code | Name | Visible | Billing enabled | Notes |
|------|------|---------|----------------|-------|
| `FREE` | Free | Yes (table) | No | Gated by `FREE_PLAN_ENABLED = FALSE` |
| `STANDARD` | Standard | Yes | Yes | Default paid plan |
| `PRO` | Pro | Yes | Yes | |
| `ELITE` | Elite | Yes | Yes | |
| `BASIC` | Basic (Legacy Alias) | **No** | **No** | Hidden, `legacy = TRUE`; backward-compatible alias of `STANDARD` |

**Critical caveat — v3 commercial features are feature-flagged OFF.**
`control.commercial_feature_flags` is seeded with all flags disabled
(`024_subscription_referral_v3.sql:163`):

- `SUBSCRIPTION_V3_ENABLED = FALSE`
- `FREE_PLAN_ENABLED = FALSE`
- `ANNUAL_BILLING_ENABLED = FALSE`
- `REFERRAL_V3_ENABLED = FALSE`

Consequence: the **FREE plan, annual billing, referral commissions, and the
revised v3 pricing/entitlement matrix are configured in the database but NOT
enforced** until an operator enables these flags. The v3 migration's
unconditional `UPDATE`s did write new prices into `control.plans`
(STANDARD $99, PRO $299, ELITE $699 monthly; annual prices added), so the
stored rows reflect v3 prices, but runtime behavior depends on the flags.

Treat the **v1 (migration 003) structural model** as the enforced baseline, with
the **v3 rows present and ready to activate**.

---

## 2. Plan Pricing (as stored in `control.plans`)

| Plan | Setup fee | Monthly | Annual | Active strategy slots | Allowed strategies |
|------|-----------|---------|--------|-----------------------|--------------------|
| **Free** | $0 | $0 | — | 1 | `STANDARD_SCALPING` only |
| **Standard** | $0 *(v3 override; v1 seed was $19)* | $99 | $990 | 1 | `STANDARD_SCALPING`, `STANDARD_SWING` |
| **Pro** | $29 *(v1 seed; not changed by v3)* | $299 *(v3 override of $499)* | $2,990 | 2 | all four |
| **Elite** | $39 *(v1 seed; not changed by v3)* | $699 *(v3 override of $999)* | $6,990 | 4 | all four |
| **Basic (legacy)** | $19 | $99 | $990 | 1 | `STANDARD_SCALPING`, `STANDARD_SWING` |

> Pricing source: `003:481-491` (v1 seeds), `024:21-39` (FREE insert + v3 price updates).
> The four strategies are: `STANDARD_SCALPING`, `ULTRA_SCALPING`, `STANDARD_SWING`, `TREND_SWING`.

---

## 3. Feature Matrix

Entitlements are the union of `003` structural grants and `024` v3 feature
overrides (v3 wins on overlapping keys, e.g. `api.access` for Elite = `false`).

| Capability | Free | Standard | Pro | Elite |
|------------|------|----------|-----|-------|
| Strategy: Standard Scalping | ✅ | ✅ | ✅ | ✅ |
| Strategy: Standard Swing | ❌ | ✅ | ✅ | ✅ |
| Strategy: Ultra Scalping | ❌ | ❌ | ✅ | ✅ |
| Strategy: Trend Swing | ❌ | ❌ | ✅ | ✅ |
| Max active strategy slots | 1 | 1 | 2 | 4 |
| Realtime signals | ❌¹ | ✅ | ✅ | ✅ |
| Signal history | 7 days | 90 days | 365 days | Unlimited |
| Take-profits (TP1/TP2/TP3) | TP1 only | TP1/2/3 | TP1/2/3 | TP1/2/3 |
| Probability / Score / Confidence | Score + Confidence | Probability + Score | + Advanced evidence | + Advanced evidence |
| Trade trailing updates | ❌ | ❌ | ✅ | ✅ |
| Explainability | ❌ | ✅ (v1) | ✅ | ✅ |
| News & Macro signals | ❌ | ❌ | ✅ | ✅ |
| Windows Agent client | ❌¹ | ✅ | ✅ | ✅ |
| MT4 / MT5 client | ❌¹ | ✅ | ✅ | ✅ |
| Execution mode: Manual | ❌¹ | ✅ | ✅ | ✅ |
| Execution mode: Assisted | ❌ | ❌ | ✅ | ✅ |
| Execution mode: Auto | ❌ | ❌ | ❌ | ✅ |
| Max devices | 1¹ | 1 | 2 | 3 |
| Max MT accounts | 1¹ | 1 | 2 | 3 |
| Basic analytics | ❌¹ | ✅ | ✅ | ✅ |
| Advanced analytics | ❌ | ❌ | ✅ | ✅ |
| Strategy comparison analytics | ❌ | ❌ | ✅ | ✅ |
| Regime analytics | ❌ | ❌ | ❌ | ✅ |
| CSV export | ❌ | ❌ | ❌ | ✅ |
| API access | ❌ | ❌ | ✅ | ❌² |
| In-app notifications | ✅ | ✅ | ✅ | ✅ |
| Standard notifications | ❌ | ✅ | ✅ | ✅ |
| Push notifications | ❌ | ❌ | ✅ | ✅ |
| Realtime notifications | ❌ | ❌ | ❌ | ✅ |
| Elite support | ❌ | ❌ | ❌ | ✅ |
| Monthly signal delivery limit | **5** | Unlimited | Unlimited | Unlimited |

¹ Free-tier grants (realtime, clients, manual execution, devices/accounts, basic
analytics, max_active_slots) are defined in `024:126` only; they are **not
enforced** while `FREE_PLAN_ENABLED = FALSE`.
² Elite `api.access = false` per v3 override (`024:129`); Pro retains `api.access = true` (`003:531`).

Sources: `003:501-559` (v1 entitlements), `024:121-133` (v3 feature matrix).

---

## 4. Limitations & Constraints

### 4.1 Per-plan hard limits
- **Free:** capped at **5 signal deliveries per month** (`signal.monthly_delivery_limit = 5`, `024:126`), **7-day** history only, single strategy (`STANDARD_SCALPING`), in-app notifications only. No billing, no client download, no execution.
- **Standard:** **1 active strategy slot** — user may own two allowed strategies but run only one at a time (`max_active_strategy_slots = 1`). **1 device / 1 MT account**. Manual execution only (no assisted/auto). No news/macro signals. No API access. History capped at **90 days**.
- **Pro:** **2 active strategy slots**, **2 devices / 2 MT accounts**, manual + assisted execution (no auto). History **365 days**. No CSV export, no regime analytics.
- **Elite:** **4 active strategy slots**, **3 devices / 3 MT accounts**, full manual + assisted + auto execution. History unlimited. **API access disabled** for Elite (`api.access = false` v3 override).

### 4.2 Billing & lifecycle
- **Currency:** USD only (`003:16`).
- **Billing interval:** `MONTHLY` primary; annual prices stored (`annual_price`) but **annual billing is disabled** (`ANNUAL_BILLING_ENABLED = FALSE`) until enabled.
- **Subscription statuses:** `INCOMPLETE, TRIAL, ACTIVE, PAST_DUE, GRACE, SUSPENDED, CANCEL_AT_PERIOD_END, CANCELLED, EXPIRED` (`003:71`).
- **Grace period:** default **7 days** (`grace_period_days`, `003:27`). Past-due subscriptions move to `GRACE` then `SUSPENDED`.
- **Trial:** a `TRIAL` status exists in the schema but **no trial plan, duration, or automated trial-issuance logic is defined** in the migrations reviewed. Trial is not a separately priced/offered plan.
- **Auto-renew:** default `TRUE` (`003:76`).
- **Setup fees:** applied once; `setup_fee_paid` flag tracked per subscription (`003:81`).

### 4.3 Licensing & device binding (applies to paid plans)
- Licenses derive entitlements from the plan (`licensing.licenses`, `003:254`).
- Execution modes per license: `SIGNAL_ONLY, MANUAL_EXECUTION, ASSISTED_EXECUTION, AUTO_EXECUTION` (`003:270`).
- Device binding is enforced: `max_devices` / `max_mt_accounts` per license (`003:266-267`); activations, device keys (ed25519), and entitlement leases are tracked (`003:316-402`).
- Exceeding device/account limits requires a plan upgrade; over-limit activation is rejected.

### 4.4 Financial / referral constraints
- Commissions derive only from **eligible validated recurring revenue** (`commissionable_amount` on invoices/payments, `003:123,177`); non-recurring and refunded amounts are excluded.
- **Referral rates (v3, gated by `REFERRAL_V3_ENABLED = FALSE`):**

  | Plan | L1 | L2 | L3 | L4+ |
  |------|----|----|----|-----|
  | Standard | 10% | 3% | 1% | 0% |
  | Pro | 15% | 4% | 2% | 0% |
  | Elite | 18% | 5% | 2% | 0% |

  Purchase multipliers: first purchase 1.00 (max L3), second 0.75 (max L1), recurring 0.50 (max L3) (`024:149-155`). **Not active** until the flag is enabled.
- Payouts, chargebacks, and refunds are ledger-backed with compensating records (`003:206-249`); history is never rewritten.

---

## 5. Honest Status Summary

| Item | State |
|------|-------|
| Paid plans (Standard/Pro/Elite) schema, subscriptions, licensing | **Deployed & enforced** |
| v1 pricing (STANDARD $99 / PRO $499→$299 / ELITE $999→$699) | Stored rows updated by v3; effective enforcement depends on flags |
| FREE plan | Configured in DB, **disabled** (`FREE_PLAN_ENABLED = FALSE`) |
| Annual billing | Prices stored, **disabled** (`ANNUAL_BILLING_ENABLED = FALSE`) |
| Referral / commission v3 | Rules stored, **disabled** (`REFERRAL_V3_ENABLED = FALSE`) |
| Trial offering | Status enum exists; **no defined trial plan/flow** |
| BASIC legacy alias | Hidden, billing disabled |

**Bottom line:** The live deployment runs the v1 subscription/licensing model
with Standard/Pro/Elite. The richer v3 commercial model (Free tier, annual
billing, referral network, revised pricing) is fully scripted in
`024_subscription_referral_v3.sql` but dormant behind disabled feature flags and
requires an explicit operator activation (with corresponding pricing/entitlement
communication to subscribers) before it takes effect.

---

## 6. Evidence References

- Plan & subscription schema + v1 seeds/entitlements: `database/migrations/003_plans_billing_licensing.sql:8-61,66-86,481-559`
- FREE plan, v3 pricing, feature matrix, referral rules, feature flags: `database/migrations/024_subscription_referral_v3.sql:21-39,121-133,136-155,163-165`
- Plan-based seed users (Free/Standard/Pro/Elite): `database/migrations/029_plan_based_test_users.sql`
- License/device/entitlement enforcement: `database/migrations/003_plans_billing_licensing.sql:254-402`
