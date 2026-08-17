-- Predict-A-Trade v1.0.0 — Migration 004
-- Referral, Commission, Payout Tables (SOW Sections 62, 69)
-- All monetary fields use DECIMAL(18,8). All timestamps use TIMESTAMPTZ (UTC).
-- Financial records are immutable except through explicit reversal/adjustment entries.

-- ============================================================
-- Affiliate Profiles (SOW Section 62)
-- ============================================================
CREATE TABLE referral.affiliate_profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    -- ACTIVE, SUSPENDED, HOLD, BANNED
    hold_reason     TEXT,
    kyc_status      VARCHAR(20) NOT NULL DEFAULT 'NOT_REQUIRED',
    -- NOT_REQUIRED, PENDING, VERIFIED, REJECTED
    tax_status      JSONB NOT NULL DEFAULT '{}',
    payout_method_id UUID,
    total_earned    DECIMAL(18,8) NOT NULL DEFAULT 0,
    -- Cached/reconcilable summary
    total_paid      DECIMAL(18,8) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id)
);

-- ============================================================
-- Referral Codes (SOW Section 69.16)
-- ============================================================
CREATE TABLE referral.referral_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    code            VARCHAR(50) NOT NULL UNIQUE,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_referral_codes_user ON referral.referral_codes(user_id);
CREATE UNIQUE INDEX idx_referral_codes_code ON referral.referral_codes(code);

-- ============================================================
-- Referral Attributions (SOW Section 69.16)
-- ============================================================
CREATE TABLE referral.referral_attributions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referred_user_id UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    referrer_user_id UUID NOT NULL REFERENCES iam.users(id),
    referral_code_id UUID REFERENCES referral.referral_codes(id),
    attribution_method VARCHAR(30) NOT NULL DEFAULT 'LINK',
    -- LINK, MANUAL_ADMIN, IMPORTED
    clicked_at     TIMESTAMPTZ,
    attributed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    registered_at  TIMESTAMPTZ,
    status         VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, ATTRIBUTED, EXPIRED, OVERRIDDEN
    expires_at     TIMESTAMPTZ,
    metadata       JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_attribution_referred ON referral.referral_attributions(referred_user_id);
CREATE INDEX idx_attribution_referrer ON referral.referral_attributions(referrer_user_id);

-- ============================================================
-- Referral Relationships (SOW Section 69.7 — five-level sponsor tree)
-- ============================================================
CREATE TABLE referral.referral_relationships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_user_id   UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    parent_user_id  UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    level           INTEGER NOT NULL CHECK (level >= 1 AND level <= 5),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(child_user_id, level),
    -- Each user has exactly one parent at each level
    CHECK(parent_user_id <> child_user_id)
    -- No self-referral (SOW 69.26)
);

CREATE INDEX idx_refrel_child ON referral.referral_relationships(child_user_id);
CREATE INDEX idx_refrel_parent ON referral.referral_relationships(parent_user_id);
CREATE INDEX idx_refrel_level ON referral.referral_relationships(level);

-- ============================================================
-- Referral Events (SOW Section 62)
-- ============================================================
CREATE TABLE referral.referral_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      VARCHAR(50) NOT NULL,
    -- CLICK, REGISTER, VERIFY, FIRST_PAYMENT, SECOND_PAYMENT, RECURRING_PAYMENT,
    -- UPGRADE, DOWNGRADE, CANCEL
    referred_user_id UUID REFERENCES iam.users(id),
    referrer_user_id UUID REFERENCES iam.users(id),
    subscription_id UUID REFERENCES billing.subscriptions(id),
    payment_id      UUID REFERENCES billing.payments(id),
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ref_events_type ON referral.referral_events(event_type);
CREATE INDEX idx_ref_events_referred ON referral.referral_events(referred_user_id);

-- ============================================================
-- Commission Rules (SOW Section 69.8, 69.28)
-- ============================================================
CREATE TABLE referral.commission_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id             UUID NOT NULL REFERENCES control.plans(id),
    level               INTEGER NOT NULL CHECK (level >= 1 AND level <= 5),
    base_rate           DECIMAL(8,4) NOT NULL,
    -- e.g. 0.10 = 10%
    effective_from      TIMESTAMPTZ NOT NULL,
    effective_until     TIMESTAMPTZ,
    rule_version        INTEGER NOT NULL DEFAULT 1,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    approved_by         UUID REFERENCES iam.users(id),
    approved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(plan_id, level, effective_from)
);

-- ============================================================
-- Purchase Commission Rules (SOW Section 69.9-69.12)
-- ============================================================
CREATE TABLE referral.purchase_commission_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_type       VARCHAR(30) NOT NULL,
    -- FIRST_PURCHASE, SECOND_PURCHASE, RECURRING_PURCHASE
    multiplier          DECIMAL(8,4) NOT NULL,
    -- 1.0, 0.75, 0.50
    max_referral_level  INTEGER NOT NULL CHECK (max_referral_level >= 1 AND max_referral_level <= 5),
    -- FIRST: 5, SECOND: 1, RECURRING: 5
    effective_from      TIMESTAMPTZ NOT NULL,
    effective_until     TIMESTAMPTZ,
    rule_version        INTEGER NOT NULL DEFAULT 1,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    approved_by         UUID REFERENCES iam.users(id),
    approved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(purchase_type, effective_from)
);

-- ============================================================
-- Commission Ledger (SOW Section 69.20 — immutable)
-- ============================================================
CREATE TABLE referral.commission_ledger (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id           UUID NOT NULL REFERENCES iam.users(id),
    source_user_id              UUID NOT NULL REFERENCES iam.users(id),
    source_subscription_id      UUID REFERENCES billing.subscriptions(id),
    purchase_id                 UUID REFERENCES billing.payments(id),
    invoice_id                  UUID REFERENCES billing.invoices(id),
    plan_id                     UUID REFERENCES control.plans(id),
    plan_version                INTEGER NOT NULL,
    purchase_number             INTEGER NOT NULL,
    -- Eligible recurring payment number
    purchase_type               VARCHAR(30) NOT NULL,
    -- FIRST_PURCHASE, SECOND_PURCHASE, RECURRING_PURCHASE
    level                       INTEGER NOT NULL CHECK (level >= 1 AND level <= 5),
    base_commission_rate        DECIMAL(8,4) NOT NULL,
    purchase_multiplier         DECIMAL(8,4) NOT NULL,
    effective_commission_rate   DECIMAL(8,4) NOT NULL,
    -- base_rate × multiplier
    commissionable_amount       DECIMAL(18,8) NOT NULL,
    commission_amount           DECIMAL(18,8) NOT NULL,
    currency                    VARCHAR(3) NOT NULL DEFAULT 'USD',
    status                      VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, CLEARED, AVAILABLE, PAID, CANCELLED, REVERSED, CHARGEBACK, FRAUD_HOLD
    commission_rule_id          UUID REFERENCES referral.commission_rules(id),
    commission_rule_snapshot    JSONB NOT NULL DEFAULT '{}',
    -- Immutable snapshot of the rule at creation time
    purchase_rule_id            UUID REFERENCES referral.purchase_commission_rules(id),
    purchase_rule_snapshot      JSONB NOT NULL DEFAULT '{}',
    payment_event_id            VARCHAR(255),
    cleared_at                  TIMESTAMPTZ,
    available_at                TIMESTAMPTZ,
    paid_at                     TIMESTAMPTZ,
    reversed_at                 TIMESTAMPTZ,
    reversal_reason             TEXT,
    reversed_by                 UUID REFERENCES iam.users(id),
    correlation_id              UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_commission_recipient ON referral.commission_ledger(recipient_user_id);
CREATE INDEX idx_commission_source ON referral.commission_ledger(source_user_id);
CREATE INDEX idx_commission_status ON referral.commission_ledger(status);
CREATE INDEX idx_commission_purchase ON referral.commission_ledger(purchase_id);
CREATE INDEX idx_commission_level ON referral.commission_ledger(level);
-- SOW 62: UNIQUE(purchase_id, recipient_user_id, level, commission_rule_snapshot_id)
-- We use a composite uniqueness to prevent duplicate commission
CREATE UNIQUE INDEX idx_commission_dedup ON referral.commission_ledger(purchase_id, recipient_user_id, level)
    WHERE status NOT IN ('REVERSED', 'CANCELLED');

-- ============================================================
-- Commission Adjustments (SOW Section 69.20 — additive, not edits)
-- ============================================================
CREATE TABLE referral.commission_adjustments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_commission_id UUID NOT NULL REFERENCES referral.commission_ledger(id),
    adjustment_type     VARCHAR(30) NOT NULL,
    -- REVERSAL, PARTIAL_REVERSAL, CORRECTION, MANUAL_ADJUSTMENT
    amount              DECIMAL(18,8) NOT NULL,
    -- Negative for reversal
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    reason              TEXT NOT NULL,
    adjusted_by         UUID NOT NULL REFERENCES iam.users(id),
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_commission_adj_original ON referral.commission_adjustments(original_commission_id);

-- ============================================================
-- Affiliate Wallets (SOW Section 69.23 — ledger-derived buckets)
-- ============================================================
CREATE TABLE referral.affiliate_wallets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    pending_balance     DECIMAL(18,8) NOT NULL DEFAULT 0,
    cleared_balance     DECIMAL(18,8) NOT NULL DEFAULT 0,
    available_balance   DECIMAL(18,8) NOT NULL DEFAULT 0,
    paid_balance        DECIMAL(18,8) NOT NULL DEFAULT 0,
    reversed_balance    DECIMAL(18,8) NOT NULL DEFAULT 0,
    on_hold_balance     DECIMAL(18,8) NOT NULL DEFAULT 0,
    -- Cached/reconcilable summary — ledger is authoritative
    last_reconciled_at  TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, currency)
);

-- ============================================================
-- Payout Methods (SOW Section 69.24)
-- ============================================================
CREATE TABLE referral.payout_methods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    method_type     VARCHAR(30) NOT NULL,
    -- BANK_TRANSFER, PAYPAL, CRYPTO, WIRE
    details_encrypted TEXT NOT NULL,
    -- Encrypted payout destination details
    details_tokenized VARCHAR(255),
    -- Tokenized/masked representation for display
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    verified        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payout_methods_user ON referral.payout_methods(user_id);

-- ============================================================
-- Payouts (SOW Section 69.24)
-- ============================================================
CREATE TABLE referral.payouts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    payout_method_id UUID REFERENCES referral.payout_methods(id),
    requested_amount DECIMAL(18,8) NOT NULL,
    approved_amount  DECIMAL(18,8) NOT NULL DEFAULT 0,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    status          VARCHAR(20) NOT NULL DEFAULT 'REQUESTED',
    -- REQUESTED, UNDER_REVIEW, APPROVED, PROCESSING, PAID, FAILED, CANCELLED
    provider_reference VARCHAR(255),
    failure_reason  TEXT,
    fee_amount      DECIMAL(18,8) NOT NULL DEFAULT 0,
    net_amount      DECIMAL(18,8) NOT NULL DEFAULT 0,
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at     TIMESTAMPTZ,
    approved_at     TIMESTAMPTZ,
    approved_by     UUID REFERENCES iam.users(id),
    processed_at    TIMESTAMPTZ,
    paid_at         TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payouts_user ON referral.payouts(user_id);
CREATE INDEX idx_payouts_status ON referral.payouts(status);

-- ============================================================
-- Payout Items (links payout to specific commission ledger entries)
-- ============================================================
CREATE TABLE referral.payout_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payout_id       UUID NOT NULL REFERENCES referral.payouts(id) ON DELETE CASCADE,
    commission_id   UUID NOT NULL REFERENCES referral.commission_ledger(id),
    amount          DECIMAL(18,8) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(payout_id, commission_id)
);

-- ============================================================
-- Affiliate Risk Flags (SOW Section 69.26)
-- ============================================================
CREATE TABLE referral.affiliate_risk_flags (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    flag_type       VARCHAR(50) NOT NULL,
    -- SELF_REFERRAL, DUPLICATE_ACCOUNT, CIRCULAR_REFERRAL, COMMISSION_FARMING,
    -- PAYMENT_ANOMALY, HIGH_VELOCITY_SIGNUP, IP_CORRELATION, DEVICE_CORRELATION
    severity        VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    -- LOW, MEDIUM, HIGH, CRITICAL
    status          VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    -- OPEN, UNDER_REVIEW, RESOLVED, DISMISSED
    evidence        JSONB NOT NULL DEFAULT '{}',
    resolved_by     UUID REFERENCES iam.users(id),
    resolved_at     TIMESTAMPTZ,
    resolution_note TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_risk_flags_user ON referral.affiliate_risk_flags(user_id);
CREATE INDEX idx_risk_flags_status ON referral.affiliate_risk_flags(status);

-- ============================================================
-- Commission Caps (SOW Section 69.25)
-- ============================================================
CREATE TABLE referral.commission_caps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope           VARCHAR(50) NOT NULL,
    -- PER_TRANSACTION, PER_DAY, PER_MONTH, PER_AFFILIATE_BALANCE, PER_NETWORK_PAYOUT
    cap_amount      DECIMAL(18,8) NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    effective_from  TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Seed Commission Rules (SOW Section 69.8, 69.34)
-- ============================================================
-- STANDARD: [10%, 4%, 2%, 1%, 0.5%]
INSERT INTO referral.commission_rules (plan_id, level, base_rate, effective_from, rule_version, approved_by, approved_at)
SELECT p.id, lvl.level, lvl.rate, now(), 1, NULL, now()
FROM control.plans p
CROSS JOIN (VALUES (1, 0.10), (2, 0.04), (3, 0.02), (4, 0.01), (5, 0.005)) AS lvl(level, rate)
WHERE p.code = 'STANDARD'
ON CONFLICT (plan_id, level, effective_from) DO NOTHING;

-- PRO: [15%, 5%, 3%, 2%, 1%]
INSERT INTO referral.commission_rules (plan_id, level, base_rate, effective_from, rule_version, approved_by, approved_at)
SELECT p.id, lvl.level, lvl.rate, now(), 1, NULL, now()
FROM control.plans p
CROSS JOIN (VALUES (1, 0.15), (2, 0.05), (3, 0.03), (4, 0.02), (5, 0.01)) AS lvl(level, rate)
WHERE p.code = 'PRO'
ON CONFLICT (plan_id, level, effective_from) DO NOTHING;

-- ELITE: [20%, 6%, 4%, 2%, 1%]
INSERT INTO referral.commission_rules (plan_id, level, base_rate, effective_from, rule_version, approved_by, approved_at)
SELECT p.id, lvl.level, lvl.rate, now(), 1, NULL, now()
FROM control.plans p
CROSS JOIN (VALUES (1, 0.20), (2, 0.06), (3, 0.04), (4, 0.02), (5, 0.01)) AS lvl(level, rate)
WHERE p.code = 'ELITE'
ON CONFLICT (plan_id, level, effective_from) DO NOTHING;

-- ============================================================
-- Seed Purchase Commission Rules (SOW Section 69.9-69.12, 69.34)
-- ============================================================
INSERT INTO referral.purchase_commission_rules (purchase_type, multiplier, max_referral_level, effective_from, rule_version, approved_by, approved_at) VALUES
    ('FIRST_PURCHASE', 1.00, 5, now(), 1, NULL, now()),
    ('SECOND_PURCHASE', 0.75, 1, now(), 1, NULL, now()),
    ('RECURRING_PURCHASE', 0.50, 5, now(), 1, NULL, now())
ON CONFLICT (purchase_type, effective_from) DO NOTHING;
