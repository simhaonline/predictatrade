# PREDICT-A-TRADE

## SUBSCRIPTION BILLING, ENTITLEMENT, REFERRAL & COMMISSION FORENSIC AUDIT

You are acting as a:

* Senior FinTech Billing Architect
* Subscription Systems Engineer
* Payments Integration Auditor
* Financial Ledger Engineer
* Referral & Affiliate Systems Auditor
* Database Integrity Engineer
* PostgreSQL / TimescaleDB Auditor
* Backend API Engineer
* Security & Fraud Prevention Engineer
* Revenue Assurance Specialist
* Accounting Logic Auditor
* QA Automation Architect
* Production Reliability Engineer

Your task is to perform a **microscopic forensic audit of the entire Predict-A-Trade subscription, billing, entitlement, referral and commission system before production go-live**.

The central question is:

> **Can Predict-A-Trade accurately charge users, grant exactly the correct subscription benefits, enforce signal limits, attribute referrals, calculate commissions, prevent duplicates/fraud, preserve immutable financial history, process upgrades/downgrades/refunds correctly, and reconcile every financial amount mathematically from source transaction to final commission?**

This is not a UI review.

This is not simply checking whether subscription pages work.

You must prove the entire commercial system end-to-end.

---

# 1. PRIMARY OBJECTIVES

Independently verify:

```text
PLAN CONFIGURATION
        ↓
SUBSCRIPTION CREATION
        ↓
PAYMENT
        ↓
PAYMENT VERIFICATION
        ↓
SUBSCRIPTION ACTIVATION
        ↓
ENTITLEMENT GENERATION
        ↓
SIGNAL/FEATURE ACCESS
        ↓
USAGE/QUOTA ACCOUNTING
        ↓
UPGRADE / DOWNGRADE / RENEWAL
        ↓
REFERRAL ATTRIBUTION
        ↓
COMMISSION CALCULATION
        ↓
COMMISSION MATURITY
        ↓
PAYOUT / REVERSAL
        ↓
FINANCIAL RECONCILIATION
```

Every monetary amount and entitlement must be traceable.

---

# 2. AUDIT-FIRST RULE

Do NOT immediately repair production logic.

Do NOT:

* change subscription prices
* change commission rates
* change plan quotas
* alter referral levels
* modify billing provider settings
* activate dormant plans
* issue refunds
* create real payments
* pay real commissions
* modify financial records
* truncate financial tables
* silently recalculate balances
* manually correct users

First:

1. inspect,
2. reconstruct,
3. mathematically verify,
4. reproduce failures,
5. document findings,
6. propose safest remediation.

Any financial mutation requires separate operator authorization.

---

# 3. CODEBASE SAFETY

Before work:

* Read `AGENTS.md`.
* Inspect Git status.
* Record current branch and commit.
* Identify billing/referral migrations.
* Identify payment provider configuration.
* Identify current plan definitions.
* Identify previous/legacy subscription aliases.
* Identify financial database tables.
* Identify existing tests.
* Preserve all existing data.

Never use destructive commands.

---

# 4. CREATE AUDIT WORKSPACE

Create:

```text
docs/COMMERCIAL_SYSTEM_FORENSIC_AUDIT/
```

Create:

```text
00_EXECUTIVE_VERDICT.md
01_COMMERCIAL_ARCHITECTURE.md
02_PLAN_CATALOG_AUDIT.md
03_SUBSCRIPTION_LIFECYCLE_AUDIT.md
04_ENTITLEMENT_ACCESS_AUDIT.md
05_SIGNAL_QUOTA_AUDIT.md
06_BILLING_PAYMENT_AUDIT.md
07_PAYMENT_WEBHOOK_AUDIT.md
08_UPGRADE_DOWNGRADE_PRORATION_AUDIT.md
09_RENEWAL_EXPIRY_GRACE_PERIOD_AUDIT.md
10_REFUND_CHARGEBACK_AUDIT.md
11_REFERRAL_ATTRIBUTION_AUDIT.md
12_MULTI_LEVEL_REFERRAL_AUDIT.md
13_COMMISSION_MATHEMATICS_AUDIT.md
14_COMMISSION_LEDGER_AUDIT.md
15_PAYOUT_AUDIT.md
16_FRAUD_ABUSE_PREVENTION.md
17_DATABASE_FINANCIAL_INTEGRITY.md
18_API_BACKEND_AUDIT.md
19_ADMIN_CONTROL_AUDIT.md
20_USER_DASHBOARD_AUDIT.md
21_SECURITY_AUTHORIZATION_AUDIT.md
22_CONCURRENCY_IDEMPOTENCY_AUDIT.md
23_RECONCILIATION_REPORT.md
24_EDGE_CASE_TEST_MATRIX.md
25_PRODUCTION_BLOCKERS.md
26_REMEDIATION_PLAN.md
27_FINAL_CERTIFICATION.md
```

Diagnostic scripts may be placed under:

```text
tools/audit/commercial/
```

They must default to read-only.

---

# 5. MAP THE TRUE COMMERCIAL ARCHITECTURE

Find the actual implementations responsible for:

* plans
* prices
* billing cycles
* subscriptions
* subscription status
* entitlements
* feature permissions
* signal quotas
* quota counters
* billing
* checkout
* invoices
* payments
* payment verification
* webhooks
* refunds
* chargebacks
* upgrades
* downgrades
* renewals
* cancellations
* referrals
* referral attribution
* referral hierarchy
* commissions
* commission ledger
* payout eligibility
* payout processing
* administrative overrides

For each produce:

| Component | File | Function/Class | DB Table | API | Source of Truth | Status |
| --------- | ---- | -------------- | -------- | --- | --------------- | ------ |

---

# 6. IDENTIFY THE SINGLE SOURCE OF TRUTH

Determine the authoritative source for:

```text
plan name
plan ID
monthly price
annual price
currency
signal limit
strategy access
feature access
device limit
history depth
commission eligibility
referral rate
referral level
subscription state
renewal date
```

Detect duplicate definitions in:

* database
* backend constants
* frontend constants
* environment variables
* seed scripts
* admin configuration
* payment provider metadata

Any conflicting sources must be documented.

---

# 7. PLAN CATALOG AUDIT

Discover every active, inactive and legacy plan from actual code/database.

Do not assume names.

Possible examples may include:

```text
Free
Basic
Premium
Professional
VIP
Legacy
```

Use actual names.

Build:

| Plan | Price | Cycle | Signals | Features | Devices | Referral | Active |
| ---- | ----: | ----- | ------: | -------- | ------: | -------- | ------ |

Identify:

* duplicate plans
* legacy aliases
* orphan price IDs
* discontinued plans still purchasable
* frontend plans not existing in backend
* DB plans not visible in UI
* mismatched pricing

---

# 8. FREE PLAN VERIFICATION

Explicitly audit the Free package.

Verify:

* price = zero
* whether subscription record exists
* signal quota
* signal frequency
* allowed strategies
* restricted strategies
* historical data
* probability access
* score access
* AI analysis access
* alert access
* device restrictions
* referral eligibility

Ensure Free users cannot accidentally receive unrestricted daily signals merely because paid signal payloads are broadcast globally.

---

# 9. ENTITLEMENT MATRIX

Build the authoritative matrix from actual code/database.

Example:

| Feature              | Free | Tier 2 | Tier 3 | Tier 4 |
| -------------------- | ---: | -----: | -----: | -----: |
| Monthly signals      |      |        |        |        |
| Standard Scalping    |      |        |        |        |
| Ultra Scalping       |      |        |        |        |
| Standard Swing       |      |        |        |        |
| Trend Swing          |      |        |        |        |
| Live signals         |      |        |        |        |
| Historical signals   |      |        |        |        |
| Score                |      |        |        |        |
| Probability          |      |        |        |        |
| AI analysis          |      |        |        |        |
| Alerts               |      |        |        |        |
| Devices              |      |        |        |        |
| Referral eligibility |      |        |        |        |

Use actual configured tiers.

---

# 10. BACKEND ENTITLEMENT ENFORCEMENT

Prove that access restrictions are enforced server-side.

Test direct API requests using accounts from every tier.

Frontend hiding alone is NOT sufficient.

Attempt access to:

* premium signal API
* signal history
* specific strategy endpoints
* probability data
* AI analysis
* WebSocket subscription
* exports
* premium analytics

Expected behavior must be enforced by backend authorization.

---

# 11. WEBSOCKET ENTITLEMENT ENFORCEMENT

This is critical.

Ensure signal broadcasting does not send full premium payloads to every connected user.

Trace:

```text
Signal generated
↓
Audience determination
↓
Entitlement evaluation
↓
Quota evaluation
↓
Payload filtering
↓
WebSocket delivery
```

A Free user must not receive paid signals simply because frontend hides them.

---

# 12. MONTHLY SIGNAL QUOTA MATHEMATICS

Identify exact quota definition.

Determine whether quota counts:

* delivered signals
* viewed signals
* generated signals
* unique strategy signals
* notifications
* API retrieval
* signal unlocking

Document the intended business rule.

Then verify implementation matches.

---

# 13. QUOTA COUNTER ATOMICITY

Test concurrency.

Suppose plan allows:

```text
10 signals/month
```

and user currently used:

```text
9
```

Simultaneously attempt multiple signal deliveries.

The user must not receive:

```text
11
12
13...
```

due to race conditions.

Use:

* DB atomic update
* transaction
* distributed lock
* unique consumption record

as appropriate.

---

# 14. QUOTA RESET

Verify reset occurs at correct boundary.

Determine whether:

```text
calendar month
subscription anniversary
billing cycle
UTC month
local month
```

is intended.

Test:

* last minute before reset
* exact reset
* first minute after reset
* DST where applicable
* renewal date change

---

# 15. PLAN UPGRADE

Trace:

```text
User requests upgrade
↓
Payment confirmation
↓
Subscription update
↓
Entitlements regenerate
↓
Quota update
↓
Cache invalidation
↓
WebSocket permission update
↓
Dashboard update
```

Verify immediate or intended timing.

Do not grant upgraded access merely from successful frontend redirect.

---

# 16. PLAN DOWNGRADE

Verify:

* effective timing
* immediate vs next-cycle behavior
* reduced quotas
* restricted strategy access
* device reduction
* historical access
* WebSocket authorization
* cached entitlements

A downgraded user must not retain premium access indefinitely through stale cache/session data.

---

# 17. PRORATION MATHEMATICS

If prorated upgrades/downgrades exist, independently calculate them.

For example conceptually:

```text
UnusedValue =
OldPlanPrice × RemainingPeriodFraction

UpgradeCharge =
NewPlanRemainingValue - OldPlanUnusedValue
```

Use actual business/provider logic.

Verify:

* rounding
* tax
* currency precision
* credits
* negative amounts
* provider-generated proration

Compare:

```text
Expected
Provider
Database
Invoice
UI
```

---

# 18. RENEWAL

Verify complete renewal flow:

```text
renewal due
↓
provider charge
↓
webhook
↓
payment verification
↓
subscription extension
↓
new billing period
↓
quota reset
↓
invoice
```

Test duplicate webhook events.

A renewal must occur exactly once.

---

# 19. FAILED RENEWAL

Verify what happens if payment fails.

Possible states:

```text
ACTIVE
PAST_DUE
GRACE
SUSPENDED
EXPIRED
CANCELED
```

Use actual implementation.

Verify entitlement changes at each state.

---

# 20. GRACE PERIOD

If grace exists, determine:

```text
duration
feature access
signal access
quota access
payment retry
final expiry
```

No undefined grace behavior.

---

# 21. CANCELLATION

Audit:

```text
cancel now
cancel at period end
admin cancel
payment-provider cancel
```

Verify entitlement timing.

---

# 22. EXPIRATION

At expiration verify:

* premium signal access revoked
* WebSocket premium subscription revoked
* admin status updated
* user status updated
* quota behavior defined
* device/licensing behavior updated if linked

---

# 23. PAYMENT PROVIDER INVENTORY

Identify actual provider(s).

Document:

```text
Provider
API endpoint
Webhook endpoint
Price/product IDs
Currency
Secret management
Signature method
Retry behavior
Idempotency
```

Do not print secrets.

---

# 24. CHECKOUT SECURITY

Audit checkout creation.

Ensure the client cannot manipulate:

```text
plan price
currency
quantity
discount
subscription duration
user ID
referral commission
```

Authoritative values must originate server-side.

---

# 25. PAYMENT VERIFICATION

Never trust:

```text
success=true
```

from frontend redirect.

Subscription activation must be based on trusted provider/server confirmation.

Trace exact code.

---

# 26. WEBHOOK SIGNATURE VERIFICATION

Audit:

* signature
* timestamp
* replay protection
* event ID
* raw body handling
* secret
* invalid signature response

Safely test invalid signatures.

---

# 27. WEBHOOK IDEMPOTENCY

Payment providers frequently send duplicate events.

Store provider event ID or equivalent.

Replay identical event multiple times.

Must NOT create:

* duplicate subscription
* duplicate invoice
* duplicate commission
* duplicate referral credit
* duplicate ledger entry

---

# 28. OUT-OF-ORDER WEBHOOKS

Test:

```text
subscription.updated
payment.succeeded
subscription.created
```

arriving unexpectedly.

Financial state must converge correctly.

---

# 29. INVOICE MATHEMATICS

For every invoice verify:

```text
subtotal
discount
tax
credits
proration
total
amount paid
balance
currency
```

Independently recompute.

---

# 30. UNIQUE INVOICE ID

Ensure invoice numbers/IDs cannot collide.

Historical invoices must remain immutable.

---

# 31. REFUNDS

Audit:

```text
full refund
partial refund
multiple partial refunds
```

Determine impact on:

* subscription
* revenue
* invoice
* referral commission
* partner balance

---

# 32. CHARGEBACKS

Audit chargeback behavior.

Commission related to reversed revenue should be:

```text
reversed
held
clawed back
```

according to business rules.

It must not remain permanently payable if underlying revenue is reversed unless explicitly intended.

---

# 33. FINANCIAL DECIMAL PRECISION

Monetary calculations must use appropriate decimal/fixed precision.

Search for dangerous:

```text
float
double
```

usage for money.

Verify precision for:

* USD
* AED
* INR
* supported currencies

Follow currency minor-unit rules.

---

# 34. ROUNDING POLICY

Define one rounding policy.

Examples:

```text
round half up
banker's rounding
provider-specific
```

Commission calculations must be consistent.

Test values such as:

```text
0.005
0.015
1.005
```

and currency-specific cases.

---

# 35. REFERRAL ATTRIBUTION FLOW

Trace:

```text
Referral Link
↓
Referral Code
↓
Visitor
↓
Cookie/session
↓
Signup
↓
Referrer assignment
↓
Subscription
↓
Payment
↓
Commission
```

Identify exact attribution point.

---

# 36. REFERRAL CODE UNIQUENESS

Verify referral codes are:

* unique
* collision resistant
* immutable where appropriate
* case behavior defined

Test concurrent creation.

---

# 37. REFERRAL ATTRIBUTION IMMUTABILITY

Determine whether users can change referrer after signup.

Prevent unauthorized manipulation such as:

```text
User signs up through A
then changes referrer to B
```

unless business rules permit this.

---

# 38. REFERRAL COOKIE SECURITY

If cookies are used, verify:

* TTL
* SameSite
* Secure
* HttpOnly if appropriate
* tamper resistance
* server validation

Client-supplied referral IDs must not be blindly trusted.

---

# 39. SELF-REFERRAL

Test:

```text
same email
same account
same device
same payment identity where available
```

Prevent self-referral if prohibited by policy.

Document exact rule.

---

# 40. CIRCULAR REFERRALS

Explicitly prevent:

```text
A refers B
B refers A
```

or deeper cycles:

```text
A → B → C → A
```

Referral tree must remain acyclic.

---

# 41. MULTI-LEVEL REFERRAL TREE

If multi-level referrals exist, reconstruct hierarchy.

Example conceptually:

```text
Customer
↓
Level 1
↓
Level 2
↓
Level 3
```

Use actual configured depth.

For each level determine:

* commission %
* qualification
* active subscription requirement
* minimum balance
* maturity
* payout status

---

# 42. COMMISSION BASE

Determine exactly what commission is calculated on.

Possible bases:

```text
gross payment
net payment
pre-tax subtotal
post-discount subtotal
post-tax amount
payment-provider net
company net revenue
```

Do not assume.

This must be explicitly documented.

---

# 43. COMMISSION EQUATION

For every level derive exact formula.

Conceptually:

```text
Commission =
CommissionBase × CommissionRate
```

If level-specific:

```text
L1 = Base × L1Rate
L2 = Base × L2Rate
L3 = Base × L3Rate
```

Use actual production logic.

---

# 44. INDEPENDENT COMMISSION RECOMPUTATION

Do NOT verify commission by calling the same production calculation.

Create independent audit calculator.

For sampled transactions compare:

| Payment | Level | Expected Rate | Expected Commission | Stored Commission | Match |
| ------- | ----- | ------------: | ------------------: | ----------------: | ----- |

---

# 45. TOTAL COMMISSION CAP

Verify:

```text
Σ all commission levels
<=
maximum distributable commission
```

unless documented business rules explicitly permit otherwise.

Example:

```text
L1 20%
L2 10%
L3 5%

Total = 35%
```

Ensure configuration cannot accidentally total 120%.

---

# 46. COMMISSION RATE SOURCE

Find all places commission rates are defined.

Detect rates duplicated across:

```text
frontend
backend
DB
environment
admin settings
seed scripts
```

Use one authoritative source.

---

# 47. COMMISSION VERSIONING

Historical commissions should retain the rate used at creation.

If admin changes:

```text
10% → 15%
```

past transactions should not automatically become 15% unless intentionally recalculated.

Store:

```text
rate
rate version
policy version
```

where appropriate.

---

# 48. COMMISSION MATURITY

If commissions mature after a waiting period:

verify:

```text
created_at
maturity_at
refund window
chargeback window
eligible_at
```

No premature payout.

---

# 49. COMMISSION STATES

Define actual state machine.

Example:

```text
PENDING
HELD
MATURED
PAYABLE
PAID
REVERSED
CANCELED
DISPUTED
```

Verify legal transitions.

No:

```text
REVERSED → PAID
```

without explicit correction workflow.

---

# 50. IMMUTABLE COMMISSION LEDGER

Do not rely solely on:

```text
user.balance = 1250
```

Financial balances should be derivable from ledger events where appropriate.

Audit ledger structure.

Each transaction should record:

```text
ledger_entry_id
user
source
source_transaction
type
debit
credit
currency
status
timestamp
reference
```

---

# 51. BALANCE RECOMPUTATION

For sampled referral users independently calculate:

```text
Opening
+ matured credits
- reversals
- payouts
+ adjustments
=
Current Balance
```

Compare with stored dashboard balance.

---

# 52. AVAILABLE VS PENDING BALANCE

Ensure distinction:

```text
Pending Commission
Available Commission
Paid Commission
Reversed Commission
Lifetime Commission
```

Do not mix values.

---

# 53. COMMISSION DUPLICATION

Test repeated processing of same qualifying payment.

Commission uniqueness should include something like:

```text
payment_id
referrer_id
level
commission_type
```

or equivalent.

One qualifying event must create one legitimate commission per permitted level.

---

# 54. UPGRADE COMMISSION

Determine whether referral receives commission on:

* initial subscription
* upgrade difference
* full upgraded plan
* renewal

Follow documented business rules.

Verify mathematics.

---

# 55. DOWNGRADE COMMISSION

Determine impact of downgrade.

Prevent overpayment if business rule only pays on actual collected revenue.

---

# 56. RECURRING COMMISSION

If referrers earn recurring commission:

trace every renewal independently.

Verify:

```text
renewal 1
renewal 2
renewal 3
```

each generates exactly one allowed commission.

---

# 57. REFERRAL QUALIFICATION

Determine whether referrer must satisfy conditions such as:

* active subscription
* KYC
* account age
* minimum sales
* approved affiliate
* paid subscription

Verify enforcement server-side.

---

# 58. COMMISSION ON FREE USERS

Confirm that Free subscriptions do not unintentionally create monetary commission.

---

# 59. REFERRAL SIGNUP WITHOUT PAYMENT

Separate:

```text
referral
conversion
qualifying payment
commission
```

A signup alone should not create paid commission unless business policy explicitly says so.

---

# 60. REFUND COMMISSION REVERSAL

Example:

```text
Payment = $100
Commission = $20
Full refund = $100
```

Expected commission reversal must follow policy.

For partial refund:

```text
Payment = $100
Refund = $40
```

determine whether commission becomes:

```text
20 → 12
```

under proportional policy.

Use actual rules.

---

# 61. CHARGEBACK AFTER PAYOUT

If commission was already paid and underlying payment later chargebacks:

define handling.

Possible:

```text
negative balance
clawback
future-offset
manual debt
```

No silent loss of accounting integrity.

---

# 62. PAYOUT REQUEST AUDIT

Audit:

```text
request
approval
processing
payment
completion
```

Verify server-side available balance.

Never trust requested amount from UI without validation.

---

# 63. DOUBLE PAYOUT PREVENTION

Two admins or duplicate requests must not pay same commission twice.

Use atomic state transition/idempotency.

---

# 64. PAYOUT FAILURE

If external payout fails:

ledger must not incorrectly mark money as paid.

Define:

```text
PROCESSING
FAILED
RETRY
PAID
```

---

# 65. ADMIN MANUAL ADJUSTMENTS

If admin can modify:

* subscription
* commission
* balance
* payout
* referral assignment

require:

* authorization
* reason
* before value
* after value
* audit log
* timestamp
* admin identity

No invisible balance edits.

---

# 66. ADMIN PLAN CONTROL

Verify admin can correctly control intended attributes:

```text
price
active/inactive
signal quota
strategies
features
device count
referral eligibility
```

Check whether change affects:

* new subscribers only
* current subscribers
* next renewal
* immediately

This behavior must be explicit.

---

# 67. ADMIN COMMISSION CONTROL

If admin can modify referral rates:

test:

* validation
* maximum values
* total-level cap
* effective date
* historical preservation

Prevent negative or >100% accidental values unless explicitly supported.

---

# 68. ADMIN SUBSCRIPTION OVERRIDE

If admin manually upgrades a user:

ensure:

* correct entitlement
* correct expiry
* no fake payment record
* correct audit classification

Distinguish:

```text
PAID
COMPLIMENTARY
MANUAL
PROMOTIONAL
```

---

# 69. PROMO / COUPON AUDIT

If discount codes exist, verify:

* validity
* expiry
* usage limit
* plan restrictions
* user restrictions
* recurring vs one-time
* combination rules

---

# 70. DISCOUNT COMMISSION BASE

Determine whether referral commission is based on:

```text
original price
or
actual amount collected
```

Use documented policy.

---

# 71. MULTI-CURRENCY

If multiple currencies are accepted:

verify:

* charge currency
* settlement currency
* commission currency
* conversion rate
* conversion timestamp
* rounding
* FX source

Never add balances from different currencies without proper conversion.

---

# 72. TAX

If tax/VAT/GST exists:

verify:

```text
tax-exclusive
tax-inclusive
tax jurisdiction
tax rate
tax rounding
```

Determine whether commissions include or exclude tax.

---

# 73. DATABASE SCHEMA AUDIT

Inspect all tables for:

* plans
* prices
* subscriptions
* subscription_history
* entitlements
* quota_usage
* payments
* payment_events
* invoices
* refunds
* referrals
* referral_tree
* commissions
* commission_ledger
* payouts
* audit_logs

Use actual names.

---

# 74. FINANCIAL FOREIGN KEYS

Every monetary record should trace to source entities.

Example:

```text
commission
→ payment
→ subscription
→ referred user
→ referrer
```

Detect orphan financial records.

---

# 75. UNIQUE CONSTRAINTS

Audit DB constraints preventing:

* duplicate payment event
* duplicate invoice
* duplicate commission
* duplicate referral
* duplicate payout
* duplicate quota consumption

Do not rely solely on application code.

---

# 76. TRANSACTION BOUNDARIES

Critical operations should be atomic.

Example:

```text
payment confirmed
+
subscription activated
+
invoice stored
+
commission generated
```

Determine appropriate transaction boundaries.

Avoid half-completed financial states.

---

# 77. CONCURRENCY TESTING

Safely test:

* duplicate checkout completion
* duplicate webhook
* simultaneous upgrade
* simultaneous quota use
* simultaneous payout request
* two admins approving payout
* concurrent referral commission workers

---

# 78. EVENTUAL CONSISTENCY

Where asynchronous jobs exist, determine expected delay.

Dashboard should distinguish:

```text
PROCESSING
```

from failure.

No user should be double-paid because retry jobs overlap.

---

# 79. JOB QUEUE AUDIT

If billing/referrals use workers:

verify:

* job ID
* retries
* dead letter behavior
* idempotency
* visibility
* logging

---

# 80. API AUDIT

Inventory endpoints covering:

```text
plans
subscriptions
checkout
billing
invoices
referrals
commissions
payouts
admin commercial controls
```

Document:

| Endpoint | Auth | RBAC | Input | DB | Idempotency | Consumer |
| -------- | ---- | ---- | ----- | -- | ----------- | -------- |

---

# 81. USER DATA ISOLATION

User A must not see:

* User B subscription
* User B invoice
* User B commission
* User B referral tree
* User B payout
* User B billing information

Test IDOR attacks safely.

---

# 82. ADMIN RBAC

Normal users must never access:

```text
/admin/subscriptions
/admin/billing
/admin/referrals
/admin/commissions
/admin/payouts
```

through direct API requests.

---

# 83. PAYMENT DATA SECURITY

Do not store prohibited raw payment information.

Audit for:

* full card number
* CVV
* payment secrets
* raw provider tokens

Use provider-hosted secure mechanisms where applicable.

---

# 84. WEBHOOK SECURITY

Audit against:

* forged events
* replay
* expired signatures
* event modification
* cross-environment events

Production must not accept test-environment payment events.

---

# 85. TEST VS PRODUCTION ISOLATION

Ensure:

```text
TEST payment
```

cannot activate:

```text
PRODUCTION subscription
```

unless intentionally configured for staging.

Separate provider keys and product IDs.

---

# 86. REFERRAL FRAUD SIGNALS

Identify protections against:

* self-referral
* account farms
* duplicate identity
* rapid referral loops
* mass free signups
* same payment source
* manipulated cookies
* fake conversions

Do not make unsupported fraud accusations; identify technical risk indicators.

---

# 87. SIGNAL ACCESS AFTER PAYMENT

Perform end-to-end entitlement test:

```text
Free user
↓
purchase paid plan
↓
provider confirms payment
↓
subscription active
↓
premium entitlement granted
↓
allowed signal delivered
```

Verify automatically.

---

# 88. PAYMENT FAILURE ACCESS TEST

Perform inverse test:

```text
payment failed
↓
no valid activation
↓
premium entitlement NOT granted
```

---

# 89. EXPIRATION ACCESS TEST

Test:

```text
active subscription expires
↓
premium access revoked
↓
Free/default behavior applied
```

without waiting for manual administrator action.

---

# 90. CACHE INVALIDATION

Subscription/referral state may be cached.

Verify cache invalidates on:

* upgrade
* downgrade
* renewal
* cancellation
* expiry
* admin override

Do not allow stale premium access.

---

# 91. BILLING PERIOD TIMEZONE

Determine canonical timestamp system.

Billing calculations should generally use unambiguous UTC timestamps internally.

Verify:

```text
period_start
period_end
renewal_at
grace_until
commission_maturity
payout_date
```

No off-by-one-day behavior.

---

# 92. CALENDAR MONTH EDGE CASES

Test:

* Jan 31
* Feb 28
* leap-year Feb 29
* 30-day months
* 31-day months

Subscription anniversary logic must behave deterministically.

---

# 93. FREE TRIAL

If trial exists verify:

* start
* end
* payment method requirement
* conversion
* cancellation
* entitlement
* referral commission eligibility

No commission before qualifying revenue unless specifically intended.

---

# 94. LIFETIME / MANUAL PLANS

If special or lifetime plans exist, identify them explicitly.

Ensure ordinary subscription expiry processes do not incorrectly cancel them.

---

# 95. LEGACY USERS

Audit legacy plan mappings.

Ensure legacy alias such as an old `Basic` identifier does not accidentally receive incorrect Free or paid entitlements.

---

# 96. PLAN ID VS PLAN NAME

Entitlement logic should preferably use immutable IDs/keys rather than UI labels.

Detect logic such as:

```python
if plan.name == "Premium":
```

where plan renaming can break access.

---

# 97. COMMERCIAL FEATURE FLAGS

Identify flags affecting:

```text
billing
subscriptions
referrals
commission
payout
free plan
```

Verify production values.

---

# 98. MOCK / SIMULATED COMMERCIAL DATA

Search for:

```text
mock subscription
fake billing
dummy commission
sample invoice
test referral
simulated payment
```

No production dashboard should present mock financial information as real.

---

# 99. HISTORICAL IMMUTABILITY

Changes to current:

* prices
* plans
* referral percentages

must not retroactively mutate old invoices/commissions.

Verify snapshots exist where required.

---

# 100. PLAN PRICE HISTORY

Determine whether historical subscription invoices retain:

```text
price actually paid
currency
discount
tax
plan at time
```

not merely current plan price.

---

# 101. REVENUE RECONCILIATION

For a selected period independently calculate:

```text
Successful payments
- refunds
- chargebacks
=
Net collected revenue
```

Then compare system totals.

---

# 102. COMMISSION RECONCILIATION

Calculate:

```text
Generated commissions
- reversed commissions
=
net commission liability
```

Then:

```text
net commission liability
- paid commissions
=
remaining payable/pending liability
```

Compare database/dashboard.

---

# 103. SUBSCRIPTION RECONCILIATION

Calculate counts:

```text
active
trial
grace
past_due
expired
canceled
free
```

Compare:

* DB
* API
* admin dashboard

---

# 104. PAYMENT RECONCILIATION

For sampled transactions compare:

| Field        | Provider | DB | Invoice | UI | Match |
| ------------ | -------- | -- | ------- | -- | ----- |
| Payment ID   |          |    |         |    |       |
| Amount       |          |    |         |    |       |
| Currency     |          |    |         |    |       |
| Status       |          |    |         |    |       |
| Customer     |          |    |         |    |       |
| Subscription |          |    |         |    |       |

---

# 105. COMMISSION SAMPLE TRACE

For every audited commission produce:

```text
COMMISSION ID:
PAYMENT ID:
SUBSCRIBER:
REFERRER:
REFERRAL LEVEL:
PLAN:
AMOUNT PAID:
DISCOUNT:
TAX:
COMMISSION BASE:
RATE:
EXPECTED COMMISSION:
ACTUAL COMMISSION:
ROUNDING:
STATE:
MATURITY:
PAYOUT ID:
VERDICT:
```

---

# 106. REFERRAL TREE TRACE

For representative conversion:

```text
Subscriber
↓
L1 Referrer
↓
L2 Referrer
↓
L3 Referrer
```

Show exact commission allocation.

---

# 107. BUSINESS RULE TEST CASE

For each configured plan create deterministic scenario.

Example:

```text
Plan Price = $100
L1 = 20%
L2 = 5%
L3 = 2%
```

Expected:

```text
L1 = $20
L2 = $5
L3 = $2
Total = $27
```

Then reproduce using production code/database.

Use actual configured rates, not this example.

---

# 108. COMPLEX COMMISSION TEST

Test:

```text
price
- coupon
+ tax
- credit
= collected amount
```

Then apply actual commission policy.

This catches incorrect commission bases.

---

# 109. PARTIAL REFUND TEST

Example conceptually:

```text
Collected = 100
Commission rate = 20%
Commission = 20

Refund = 25%
```

Determine mathematically expected remaining commission under production policy.

Verify.

---

# 110. DUPLICATE EVENT TEST

Replay same successful payment event 10 times in safe test environment.

Expected:

```text
1 payment
1 subscription state transition
1 invoice
1 commission per legitimate referral level
```

not ten.

---

# 111. OUT-OF-ORDER EVENT TEST

Replay events in unusual sequence and prove final state converges correctly.

---

# 112. RECOVERY AFTER FAILURE

Simulate safe application failure between:

```text
payment persisted
and
commission generated
```

Determine whether retry can finish processing without duplication.

---

# 113. DATABASE FAILURE

Determine behavior if DB unavailable when payment provider webhook arrives.

Provider retry or internal queue must preserve event.

Financial events must not disappear.

---

# 114. ADMIN DASHBOARD DATA

Verify Admin commercial dashboard values originate from real DB calculations.

Audit:

* MRR
* ARR if shown
* active subscribers
* revenue
* refunds
* commissions
* referral conversions
* pending payouts

No placeholder metrics.

---

# 115. USER DASHBOARD BILLING

Verify user sees only their:

```text
current plan
renewal
quota
usage
invoices
referral code
referral earnings
pending commissions
paid commissions
```

Values must reconcile with backend.

---

# 116. MRR MATHEMATICS

If Monthly Recurring Revenue is displayed, independently verify normalization.

For annual plans:

```text
AnnualPrice / 12
```

may contribute to MRR depending on chosen methodology.

Document exact formula.

---

# 117. ARR MATHEMATICS

If ARR is displayed, derive from accepted methodology.

Do not simply multiply unreliable MRR.

---

# 118. LIFETIME VALUE

If LTV is displayed, identify exact methodology.

Do not present arbitrary calculation as accounting fact.

---

# 119. REFERRAL ANALYTICS

Verify:

```text
clicks
signups
paid conversions
conversion rate
commission
```

are not mixed.

Define denominators precisely.

---

# 120. COMMISSION RATE VALIDATION

Add constraints preventing:

```text
negative rates
NaN
infinite
> configured maximum
```

and invalid level counts.

---

# 121. PLAN PRICE VALIDATION

Prevent:

```text
negative price
zero-priced paid plan unintentionally
currency mismatch
wrong decimal precision
```

---

# 122. MANUAL DATABASE EDIT RISK

Identify whether financial records can be casually altered.

Recommend strong controls and audit trails where absent.

Do not perform schema changes yet.

---

# 123. P0 NO-GO CONDITIONS

Treat as P0/production NO-GO if confirmed:

* user receives paid access without valid entitlement
* payment marked successful without trusted verification
* duplicate webhook produces duplicate commission
* commission calculation mathematically incorrect
* user can manipulate price client-side
* user can alter referral attribution
* self/circular referral produces unauthorized commission
* financial ledger cannot reconcile
* paid-out commissions can be duplicated
* user can access another user's billing/referral data
* refunds do not reverse required commissions
* failed payments activate subscriptions
* expired users retain paid signal access
* subscription quotas can be bypassed
* monetary calculations lose precision
* production financial records rely on mock data

---

# 124. SEVERITY

Use:

```text
P0 CRITICAL
P1 HIGH
P2 MEDIUM
P3 LOW
INFO
```

---

# 125. FINDING FORMAT

Every material finding must contain:

```text
Finding ID:
Severity:
Subsystem:
Affected Plan/User/Transaction:
Expected Rule:
Observed Rule:
Expected Amount:
Actual Amount:
Difference:
Code Path:
DB Evidence:
API Evidence:
Provider Evidence:
Root Cause:
Financial Impact:
Access Impact:
Fraud Risk:
Recommended Fix:
Required Regression Test:
```

Redact personal information and secrets.

---

# 126. TEST MATRIX

At minimum test:

### Subscription

```text
Free
Paid signup
Upgrade
Downgrade
Renew
Cancel
Expire
Failed payment
Grace period
```

### Billing

```text
successful payment
duplicate webhook
invalid webhook
refund
partial refund
chargeback
```

### Referral

```text
valid referral
no referral
self referral
circular referral
multi-level referral
```

### Commission

```text
initial payment
renewal
upgrade
refund
chargeback
duplicate processing
payout
```

---

# 127. REQUIRED SQL / DATA EVIDENCE

Use read-only database queries to verify real state.

Do not assume table names.

Inspect actual schema first.

Collect evidence for:

```text
plans
subscriptions
payments
invoices
referrals
commissions
payouts
entitlements
quota usage
audit logs
```

---

# 128. MATHEMATICAL INDEPENDENCE

For all money calculations:

Do not verify production formula with itself.

Create separate audit formulas for:

```text
invoice totals
proration
commission
refund adjustment
commission reversal
balances
MRR
quota usage
```

---

# 129. CROSS-LAYER RECONCILIATION

For sampled subscription compare:

```text
PAYMENT PROVIDER
=
PAYMENT TABLE
=
SUBSCRIPTION
=
ENTITLEMENT
=
API
=
ADMIN UI
=
USER UI
```

For commission compare:

```text
PAYMENT
=
REFERRAL TREE
=
COMMISSION ENGINE
=
COMMISSION TABLE
=
LEDGER
=
USER BALANCE
=
ADMIN REPORT
```

Every discrepancy is a finding.

---

# 130. FINAL SUBSCRIPTION CERTIFICATION TABLE

| Area                   | Status | Evidence |
| ---------------------- | ------ | -------- |
| Plan Catalog           |        |          |
| Pricing                |        |          |
| Subscription Lifecycle |        |          |
| Entitlements           |        |          |
| Signal Quotas          |        |          |
| Upgrade                |        |          |
| Downgrade              |        |          |
| Renewal                |        |          |
| Expiration             |        |          |
| Billing                |        |          |
| Webhooks               |        |          |
| Invoices               |        |          |
| Refunds                |        |          |
| Chargebacks            |        |          |

Use:

```text
VERIFIED
PARTIAL
FAILED
UNVERIFIED
```

---

# 131. FINAL REFERRAL CERTIFICATION TABLE

| Area                   | Status | Evidence |
| ---------------------- | ------ | -------- |
| Referral Attribution   |        |          |
| Referral Code          |        |          |
| Anti Self-Referral     |        |          |
| Anti Circular Referral |        |          |
| Multi-Level Tree       |        |          |
| Commission Base        |        |          |
| Commission Rate        |        |          |
| Commission Formula     |        |          |
| Commission Maturity    |        |          |
| Refund Reversal        |        |          |
| Ledger                 |        |          |
| Payout                 |        |          |

---

# 132. FINAL COMMISSION CALCULATION TABLE

For representative payments:

| Payment | Revenue Base | L1 | L2 | L3 | Expected Total | Stored Total | Difference |
| ------- | -----------: | -: | -: | -: | -------------: | -----------: | ---------: |

Use actual supported referral levels.

---

# 133. FINAL COMMERCIAL VERDICT

Begin and end the final report with:

```text
=========================================================

PREDICT-A-TRADE
SUBSCRIPTION / BILLING / REFERRAL CERTIFICATION

PLAN CONFIGURATION:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

SUBSCRIPTION LIFECYCLE:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

ENTITLEMENT ENFORCEMENT:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

SIGNAL QUOTAS:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

PAYMENT PROCESSING:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

PAYMENT WEBHOOKS:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

INVOICE MATHEMATICS:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

UPGRADE / DOWNGRADE:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

REFUNDS / CHARGEBACKS:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

REFERRAL ATTRIBUTION:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

MULTI-LEVEL REFERRALS:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

COMMISSION CALCULATION:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

COMMISSION LEDGER:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

PAYOUT:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

FINANCIAL RECONCILIATION:
VERIFIED / PARTIAL / FAILED / UNVERIFIED


CAN USERS BYPASS PAID ACCESS?
YES / NO / UNVERIFIED

CAN COMMISSIONS DUPLICATE?
YES / NO / UNVERIFIED

CAN REFERRAL ATTRIBUTION BE MANIPULATED?
YES / NO / UNVERIFIED

ARE ALL COMMISSION VALUES MATHEMATICALLY REPRODUCIBLE?
YES / NO / PARTIAL

CAN FINANCIAL BALANCES BE RECONSTRUCTED FROM SOURCE EVENTS?
YES / NO / PARTIAL

SAFE FOR PRODUCTION BILLING?
YES / CONDITIONAL / NO

SAFE FOR REAL COMMISSION PAYOUT?
YES / CONDITIONAL / NO


FINAL VERDICT:

GO
/
CONDITIONAL GO
/
NO-GO

P0:
P1:
P2:
P3:

=========================================================
```

---

# 134. FINAL QUESTIONS — ANSWER ALL

1. What plans actually exist?
2. Which plans are active?
3. Which are legacy?
4. Are prices consistent everywhere?
5. Are plan limits sourced centrally?
6. Can Free users retrieve premium signals?
7. Are monthly signal limits server-side?
8. Can quota races exceed the plan limit?
9. Do upgrades automatically unlock correct features?
10. Do downgrades correctly restrict them?
11. Do expired subscriptions lose access?
12. Can failed payment activate a subscription?
13. Is payment confirmation provider-authoritative?
14. Are webhook signatures verified?
15. Are duplicate webhooks idempotent?
16. Are invoices mathematically correct?
17. Is proration correct?
18. Are refunds handled correctly?
19. Are chargebacks handled correctly?
20. Is referral attribution deterministic?
21. Can referral attribution be manipulated?
22. Is self-referral prevented where required?
23. Are circular referral trees impossible?
24. Are referral levels correct?
25. What exactly is the commission base?
26. Are commission rates correctly applied?
27. Can total commission exceed allowed amount?
28. Can a payment create duplicate commission?
29. Are renewal commissions correct?
30. Are upgrade commissions correct?
31. Are refund commission reversals correct?
32. Are chargeback reversals correct?
33. Are commissions versioned historically?
34. Are monetary calculations using safe decimal precision?
35. Can referral balances be reconstructed from ledger records?
36. Are Pending, Available and Paid balances separated?
37. Can payout be duplicated?
38. Are admin adjustments fully audited?
39. Can one user access another user's commercial data?
40. Are test and production payment environments isolated?
41. Are commercial dashboards displaying genuine database data?
42. Can every dollar of commission be traced to a qualifying payment?
43. Can every entitlement be traced to a valid subscription state?
44. Are subscription and commission calculations reproducible independently?
45. Is the commercial system genuinely safe for real paying users?

---

# 135. COMMERCIAL TRUTH PRINCIPLE

Never conclude:

```text
payment page works → billing works

subscription says ACTIVE → payment valid

button hidden → access restricted

commission displayed → calculation correct

balance displayed → ledger correct

webhook returned 200 → event safely processed
```

Require:

```text
VALID PAYMENT
+
TRUSTED VERIFICATION
+
IDEMPOTENT EVENT PROCESSING
+
CORRECT SUBSCRIPTION STATE
+
CORRECT ENTITLEMENTS
+
ATOMIC QUOTA ENFORCEMENT
+
VALID REFERRAL ATTRIBUTION
+
CORRECT COMMISSION BASE
+
CORRECT COMMISSION RATE
+
DECIMAL-SAFE MATHEMATICS
+
IMMUTABLE LEDGER
+
REVERSAL SUPPORT
+
PAYOUT SAFETY
+
CROSS-LAYER RECONCILIATION
=
PRODUCTION-READY COMMERCIAL SYSTEM
```

---

# 136. DO NOT OPTIMIZE FOR PASS

The purpose of this audit is not to prove the subscription/referral system is good.

The purpose is to discover whether it is **financially correct**.

If something cannot be verified:

```text
UNVERIFIED
```

If external payment credentials or sandbox access are missing:

```text
EXTERNAL BLOCKER
```

If logic exists but is not wired:

```text
UNWIRED
```

If frontend simulates data:

```text
SIMULATED
```

Do not hide these states.

---

# 137. STOP AFTER AUDIT

After completing this audit:

1. Produce the final certification.
2. Produce exact P0–P3 findings.
3. Identify financial leakage risk.
4. Identify subscription-access leakage.
5. Identify commission overpayment/underpayment risk.
6. Identify duplicate processing risks.
7. Identify database/ledger defects.
8. Propose exact remediation sequence.
9. Define regression tests.
10. STOP.

Do NOT implement broad commercial or financial changes until separately authorized.

---

# 138. BEGIN

Start from repository root.

First:

1. read `AGENTS.md`,
2. inspect existing commercial architecture,
3. inspect plan/price definitions,
4. inspect subscription models,
5. inspect entitlement enforcement,
6. inspect signal quotas,
7. inspect billing provider integration,
8. inspect webhooks,
9. inspect referral architecture,
10. inspect commission formulas,
11. inspect financial ledger,
12. independently recompute real sample transactions,
13. reconcile provider → DB → API → dashboards,
14. identify every discrepancy,
15. issue the final production certification.

**Prefer accounting truth over dashboard appearance.**

**Prefer immutable financial evidence over mutable balances.**

**Prefer backend entitlement enforcement over frontend hiding.**

**Prefer idempotency over duplicate revenue events.**

**Prefer exact mathematical reconciliation over assumptions.**
