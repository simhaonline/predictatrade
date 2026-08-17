-- Predict-A-Trade v1.0.0 — Migration 003
-- Plans, Billing, Licensing Tables (SOW Sections 37-43, 55, 62, 66, 69)
-- All monetary fields use DECIMAL(18,8). All timestamps use TIMESTAMPTZ (UTC).

-- ============================================================
-- Plans (SOW Section 69.1)
-- ============================================================
CREATE TABLE control.plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(50) NOT NULL UNIQUE,
    -- STANDARD, PRO, ELITE
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    setup_fee           DECIMAL(18,8) NOT NULL DEFAULT 0,
    monthly_price       DECIMAL(18,8) NOT NULL DEFAULT 0,
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    billing_interval    VARCHAR(20) NOT NULL DEFAULT 'MONTHLY',
    max_active_strategy_slots INTEGER NOT NULL DEFAULT 1,
    allowed_strategies  JSONB NOT NULL DEFAULT '[]',
    -- e.g. ["STANDARD_SCALPING", "STANDARD_SWING"]
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    effective_from      TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_until     TIMESTAMPTZ,
    upgrade_rules       JSONB NOT NULL DEFAULT '{}',
    downgrade_rules     JSONB NOT NULL DEFAULT '{}',
    cancellation_rules  JSONB NOT NULL DEFAULT '{}',
    grace_period_days   INTEGER NOT NULL DEFAULT 7,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Plan versioning — track price/entitlement changes (SOW Section 66)
CREATE TABLE control.plan_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id         UUID NOT NULL REFERENCES control.plans(id),
    version         INTEGER NOT NULL,
    setup_fee       DECIMAL(18,8) NOT NULL,
    monthly_price   DECIMAL(18,8) NOT NULL,
    allowed_strategies JSONB NOT NULL,
    max_active_strategy_slots INTEGER NOT NULL,
    effective_from  TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,
    changed_by      UUID REFERENCES iam.users(id),
    change_reason   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(plan_id, version)
);

-- ============================================================
-- Plan Entitlements (SOW Section 38)
-- ============================================================
CREATE TABLE control.plan_entitlements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id         UUID NOT NULL REFERENCES control.plans(id) ON DELETE CASCADE,
    entitlement_key VARCHAR(100) NOT NULL,
    -- e.g. strategy.standard_scalping, signals.realtime, execution.auto, devices.max
    entitlement_value JSONB NOT NULL DEFAULT 'true',
    -- boolean, number, or array
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(plan_id, entitlement_key)
);

-- ============================================================
-- Subscriptions (SOW Section 69.3)
-- ============================================================
CREATE TABLE billing.subscriptions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES iam.users(id),
    plan_id             UUID NOT NULL REFERENCES control.plans(id),
    status              VARCHAR(30) NOT NULL DEFAULT 'INCOMPLETE',
    -- INCOMPLETE, TRIAL, ACTIVE, PAST_DUE, GRACE, SUSPENDED,
    -- CANCEL_AT_PERIOD_END, CANCELLED, EXPIRED
    billing_period_start TIMESTAMPTZ NOT NULL,
    billing_period_end   TIMESTAMPTZ NOT NULL,
    next_billing_date    TIMESTAMPTZ,
    auto_renew          BOOLEAN NOT NULL DEFAULT TRUE,
    selected_strategies JSONB NOT NULL DEFAULT '[]',
    -- User-selected strategies within plan limits
    cancel_reason       TEXT,
    cancelled_at        TIMESTAMPTZ,
    setup_fee_paid      BOOLEAN NOT NULL DEFAULT FALSE,
    eligible_payment_count INTEGER NOT NULL DEFAULT 0,
    -- Counter of validated eligible recurring subscription payments (SOW 69.9)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_user ON billing.subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON billing.subscriptions(status);
CREATE INDEX idx_subscriptions_plan ON billing.subscriptions(plan_id);

-- ============================================================
-- Subscription Events
-- ============================================================
CREATE TABLE billing.subscription_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES billing.subscriptions(id) ON DELETE CASCADE,
    event_type      VARCHAR(50) NOT NULL,
    -- CREATED, ACTIVATED, RENEWED, PAST_DUE, SUSPENDED, CANCELLED, UPGRADED, DOWNGRADED, GRACE
    metadata        JSONB NOT NULL DEFAULT '{}',
    actor_id        UUID REFERENCES iam.users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sub_events_sub ON billing.subscription_events(subscription_id);

-- ============================================================
-- Invoices (SOW Section 62)
-- ============================================================
CREATE TABLE billing.invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id     UUID NOT NULL REFERENCES billing.subscriptions(id),
    user_id             UUID NOT NULL REFERENCES iam.users(id),
    invoice_number      VARCHAR(50) NOT NULL UNIQUE,
    plan_id             UUID NOT NULL REFERENCES control.plans(id),
    plan_version        INTEGER NOT NULL,
    billing_period_start TIMESTAMPTZ NOT NULL,
    billing_period_end   TIMESTAMPTZ NOT NULL,
    subtotal            DECIMAL(18,8) NOT NULL,
    discounts           DECIMAL(18,8) NOT NULL DEFAULT 0,
    taxes               DECIMAL(18,8) NOT NULL DEFAULT 0,
    total               DECIMAL(18,8) NOT NULL,
    commissionable_amount DECIMAL(18,8) NOT NULL DEFAULT 0,
    -- SOW 69.5: eligible recurring subtotal minus eligible discounts
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    status              VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    -- DRAFT, OPEN, PAID, VOID, UNCOLLECTIBLE
    due_date            TIMESTAMPTZ,
    paid_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invoices_sub ON billing.invoices(subscription_id);
CREATE INDEX idx_invoices_user ON billing.invoices(user_id);
CREATE INDEX idx_invoices_status ON billing.invoices(status);

-- ============================================================
-- Invoice Items
-- ============================================================
CREATE TABLE billing.invoice_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id      UUID NOT NULL REFERENCES billing.invoices(id) ON DELETE CASCADE,
    description     VARCHAR(255) NOT NULL,
    item_type       VARCHAR(30) NOT NULL,
    -- SETUP_FEE, SUBSCRIPTION, PRORATION, ADJUSTMENT, ADD_ON
    quantity        DECIMAL(18,8) NOT NULL DEFAULT 1,
    unit_price      DECIMAL(18,8) NOT NULL,
    amount          DECIMAL(18,8) NOT NULL,
    commissionable  BOOLEAN NOT NULL DEFAULT FALSE,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invoice_items_invoice ON billing.invoice_items(invoice_id);

-- ============================================================
-- Payments (SOW Section 62, 69.19)
-- ============================================================
CREATE TABLE billing.payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id          UUID REFERENCES billing.invoices(id),
    subscription_id     UUID REFERENCES billing.subscriptions(id),
    user_id             UUID NOT NULL REFERENCES iam.users(id),
    provider            VARCHAR(50) NOT NULL,
    -- STRIPE, PAYPAL, MANUAL, etc.
    provider_payment_id VARCHAR(255),
    provider_event_id   VARCHAR(255),
    amount              DECIMAL(18,8) NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    payment_type        VARCHAR(30) NOT NULL,
    -- SETUP_FEE, SUBSCRIPTION, PRORATION, REFUND, CHARGEBACK
    payment_number      INTEGER NOT NULL DEFAULT 0,
    -- Eligible recurring payment number (SOW 69.9)
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, SUCCEEDED, FAILED, REFUNDED, PARTIALLY_REFUNDED, CHARGED_BACK
    commissionable_amount DECIMAL(18,8) NOT NULL DEFAULT 0,
    processed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_user ON billing.payments(user_id);
CREATE INDEX idx_payments_sub ON billing.payments(subscription_id);
CREATE INDEX idx_payments_status ON billing.payments(status);
CREATE UNIQUE INDEX idx_payments_provider_event ON billing.payments(provider, provider_event_id)
    WHERE provider_event_id IS NOT NULL;

-- ============================================================
-- Payment Events (SOW Section 69.19 — idempotency)
-- ============================================================
CREATE TABLE billing.payment_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id      UUID REFERENCES billing.payments(id),
    provider        VARCHAR(50) NOT NULL,
    provider_event_id VARCHAR(255) NOT NULL,
    event_type      VARCHAR(50) NOT NULL,
    raw_payload     JSONB NOT NULL,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_event_id)
);

-- ============================================================
-- Refunds & Chargebacks (SOW Section 62, 69.22)
-- ============================================================
CREATE TABLE billing.refunds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id      UUID NOT NULL REFERENCES billing.payments(id),
    amount          DECIMAL(18,8) NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    reason          VARCHAR(50) NOT NULL,
    -- FULL_REFUND, PARTIAL_REFUND, CHARGEBACK, SUBSCRIPTION_CANCELLED
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    provider_refund_id VARCHAR(255),
    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refunds_payment ON billing.refunds(payment_id);

-- ============================================================
-- Coupons & Credits (SOW Section 62)
-- ============================================================
CREATE TABLE billing.coupons (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(50) NOT NULL UNIQUE,
    description     TEXT,
    discount_type   VARCHAR(20) NOT NULL,
    -- PERCENTAGE, FIXED_AMOUNT
    discount_value  DECIMAL(18,8) NOT NULL,
    currency        VARCHAR(3) DEFAULT 'USD',
    max_redemptions INTEGER,
    redemption_count INTEGER NOT NULL DEFAULT 0,
    valid_from      TIMESTAMPTZ,
    valid_until     TIMESTAMPTZ,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE billing.credits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    amount          DECIMAL(18,8) NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    reason          VARCHAR(100) NOT NULL,
    expires_at      TIMESTAMPTZ,
    used_amount     DECIMAL(18,8) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Licenses (SOW Section 37)
-- ============================================================
CREATE TABLE licensing.licenses (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID REFERENCES iam.organizations(id),
    user_id             UUID NOT NULL REFERENCES iam.users(id),
    plan_id             UUID NOT NULL REFERENCES control.plans(id),
    subscription_id     UUID REFERENCES billing.subscriptions(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, ACTIVE, SUSPENDED, EXPIRED, REVOKED, GRACE
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    valid_from          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ,
    grace_until         TIMESTAMPTZ,
    max_devices         INTEGER NOT NULL DEFAULT 1,
    max_mt_accounts     INTEGER NOT NULL DEFAULT 1,
    allowed_features    JSONB NOT NULL DEFAULT '[]',
    allowed_execution_modes JSONB NOT NULL DEFAULT '[]',
    -- SIGNAL_ONLY, MANUAL_EXECUTION, ASSISTED_EXECUTION, AUTO_EXECUTION
    allowed_strategies  JSONB NOT NULL DEFAULT '[]',
    created_by          UUID REFERENCES iam.users(id),
    revoked_at          TIMESTAMPTZ,
    revocation_reason   TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_licenses_user ON licensing.licenses(user_id);
CREATE INDEX idx_licenses_status ON licensing.licenses(status);
CREATE INDEX idx_licenses_sub ON licensing.licenses(subscription_id);

-- ============================================================
-- License Entitlements (derived from plan + overrides)
-- ============================================================
CREATE TABLE licensing.license_entitlements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id      UUID NOT NULL REFERENCES licensing.licenses(id) ON DELETE CASCADE,
    entitlement_key VARCHAR(100) NOT NULL,
    entitlement_value JSONB NOT NULL DEFAULT 'true',
    is_override     BOOLEAN NOT NULL DEFAULT FALSE,
    -- TRUE if manually set, FALSE if derived from plan
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(license_id, entitlement_key)
);

-- ============================================================
-- License Events (SOW Section 62)
-- ============================================================
CREATE TABLE licensing.license_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id      UUID NOT NULL REFERENCES licensing.licenses(id) ON DELETE CASCADE,
    event_type      VARCHAR(50) NOT NULL,
    -- ISSUED, ACTIVATED, SUSPENDED, REVOKED, EXPIRED, RENEWED, RESET
    actor_id        UUID REFERENCES iam.users(id),
    reason          TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_license_events_license ON licensing.license_events(license_id);

-- ============================================================
-- Devices & Device Keys (SOW Sections 40, 42, 62)
-- ============================================================
CREATE TABLE licensing.devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    license_id      UUID REFERENCES licensing.licenses(id),
    device_name     VARCHAR(255),
    windows_version VARCHAR(100),
    agent_version   VARCHAR(50),
    hostname        VARCHAR(255),
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ,
    last_ip         INET,
    connection_status VARCHAR(20) NOT NULL DEFAULT 'OFFLINE',
    -- ONLINE, OFFLINE, EXPIRED
    security_state  VARCHAR(20) NOT NULL DEFAULT 'UNKNOWN',
    revoked_at      TIMESTAMPTZ,
    revocation_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_devices_user ON licensing.devices(user_id);
CREATE INDEX idx_devices_license ON licensing.devices(license_id);

CREATE TABLE licensing.device_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL REFERENCES licensing.devices(id) ON DELETE CASCADE,
    public_key      TEXT NOT NULL,
    key_algorithm   VARCHAR(20) NOT NULL DEFAULT 'ed25519',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_rotated_at TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    UNIQUE(device_id)
);

-- ============================================================
-- License-Device Bindings (SOW Section 39)
-- ============================================================
CREATE TABLE licensing.license_devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id      UUID NOT NULL REFERENCES licensing.licenses(id) ON DELETE CASCADE,
    device_id       UUID NOT NULL REFERENCES licensing.devices(id) ON DELETE CASCADE,
    activated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at  TIMESTAMPTZ,
    UNIQUE(license_id, device_id)
);

-- ============================================================
-- Activations (SOW Section 39)
-- ============================================================
CREATE TABLE licensing.activations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id      UUID NOT NULL REFERENCES licensing.licenses(id),
    device_id       UUID REFERENCES licensing.devices(id),
    activation_code VARCHAR(255) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, COMPLETED, FAILED, EXPIRED
    activated_at    TIMESTAMPTZ,
    ip_address      INET,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_activations_license ON licensing.activations(license_id);

-- ============================================================
-- Entitlement Leases (SOW Section 41 — signed short-lived)
-- ============================================================
CREATE TABLE licensing.entitlement_leases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id      UUID NOT NULL REFERENCES licensing.licenses(id),
    device_id       UUID REFERENCES licensing.devices(id),
    token_id        VARCHAR(255) NOT NULL UNIQUE,
    subject         VARCHAR(255) NOT NULL,
    features        JSONB NOT NULL DEFAULT '[]',
    execution_modes JSONB NOT NULL DEFAULT '[]',
    strategies      JSONB NOT NULL DEFAULT '[]',
    issuer          VARCHAR(100) NOT NULL,
    audience        VARCHAR(100) NOT NULL,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_leases_license ON licensing.entitlement_leases(license_id);
CREATE INDEX idx_leases_token ON licensing.entitlement_leases(token_id);
CREATE INDEX idx_leases_expires ON licensing.entitlement_leases(expires_at) WHERE revoked_at IS NULL;

-- ============================================================
-- MT Accounts (SOW Section 43)
-- ============================================================
CREATE TABLE licensing.mt_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    license_id      UUID REFERENCES licensing.licenses(id),
    broker          VARCHAR(100) NOT NULL,
    broker_server   VARCHAR(255),
    platform        VARCHAR(10) NOT NULL,
    -- MT4 or MT5
    account_reference VARCHAR(255) NOT NULL,
    account_currency VARCHAR(3),
    symbol_mapping  JSONB NOT NULL DEFAULT '{}',
    authorized      BOOLEAN NOT NULL DEFAULT FALSE,
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mt_accounts_user ON licensing.mt_accounts(user_id);

-- ============================================================
-- MT Connections (heartbeats)
-- ============================================================
CREATE TABLE licensing.mt_connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mt_account_id   UUID NOT NULL REFERENCES licensing.mt_accounts(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL,
    -- CONNECTED, DISCONNECTED, RECONNECTING
    last_heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Client Releases (SOW Section 53)
-- ============================================================
CREATE TABLE licensing.client_releases (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component           VARCHAR(50) NOT NULL,
    -- WINDOWS_AGENT, MT4_EA, MT5_EA
    version             VARCHAR(50) NOT NULL,
    channel             VARCHAR(20) NOT NULL DEFAULT 'STABLE',
    -- STABLE, BETA
    download_url        TEXT NOT NULL,
    sha256              VARCHAR(64) NOT NULL,
    signature           TEXT,
    signature_key_id    VARCHAR(100),
    release_notes       TEXT,
    minimum_server_version VARCHAR(50),
    minimum_client_version VARCHAR(50),
    mandatory           BOOLEAN NOT NULL DEFAULT FALSE,
    published_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(component, version, channel)
);

-- ============================================================
-- Download Events (SOW Section 62)
-- ============================================================
CREATE TABLE licensing.download_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES iam.users(id),
    release_id      UUID NOT NULL REFERENCES licensing.client_releases(id),
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_download_events_release ON licensing.download_events(release_id);

-- ============================================================
-- Seed initial plans (SOW Section 69.1, 69.34)
-- ============================================================
INSERT INTO control.plans (code, name, setup_fee, monthly_price, currency, billing_interval, max_active_strategy_slots, allowed_strategies, description) VALUES
    ('STANDARD', 'Standard', 19, 99, 'USD', 'MONTHLY', 1,
     '["STANDARD_SCALPING","STANDARD_SWING"]'::jsonb,
     'Exactly 1 active strategy: Standard Scalping OR Standard Swing'),
    ('PRO', 'Pro', 29, 499, 'USD', 'MONTHLY', 2,
     '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING"]'::jsonb,
     'Any 2 active strategies from all four'),
    ('ELITE', 'Elite', 39, 999, 'USD', 'MONTHLY', 4,
     '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING"]'::jsonb,
     'All 4 strategies active')
ON CONFLICT (code) DO NOTHING;

-- Backward-compatible alias for BASIC -> STANDARD (SOW 69.1)
INSERT INTO control.plans (code, name, setup_fee, monthly_price, currency, billing_interval, max_active_strategy_slots, allowed_strategies, description) VALUES
    ('BASIC', 'Basic (Legacy Alias)', 19, 99, 'USD', 'MONTHLY', 1,
     '["STANDARD_SCALPING","STANDARD_SWING"]'::jsonb,
     'Legacy alias for STANDARD plan')
ON CONFLICT (code) DO NOTHING;

-- Seed plan entitlements (SOW Section 38)
INSERT INTO control.plan_entitlements (plan_id, entitlement_key, entitlement_value) VALUES
    -- STANDARD
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'signals.realtime', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'signals.history', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'signals.explainability', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'client.windows', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'client.mt4', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'client.mt5', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'execution.manual', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'devices.max', '1'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'accounts.max', '1'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'analytics.basic', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'strategy.standard_scalping', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'strategy.standard_swing', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'STANDARD', 'strategy.max_active_slots', '1'::jsonb),
    -- PRO
    (SELECT id FROM control.plans WHERE code = 'PRO', 'signals.realtime', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'signals.history', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'signals.explainability', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'signals.news', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'signals.macro', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'client.windows', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'client.mt4', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'client.mt5', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'execution.manual', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'execution.assisted', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'devices.max', '2'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'accounts.max', '2'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'analytics.basic', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'analytics.advanced', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'api.access', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'strategy.standard_scalping', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'strategy.ultra_scalping', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'strategy.standard_swing', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'strategy.trend_swing', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'PRO', 'strategy.max_active_slots', '2'::jsonb),
    -- ELITE
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'signals.realtime', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'signals.history', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'signals.explainability', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'signals.news', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'signals.macro', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'client.windows', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'client.mt4', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'client.mt5', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'execution.manual', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'execution.assisted', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'execution.auto', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'devices.max', '3'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'accounts.max', '3'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'analytics.basic', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'analytics.advanced', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'api.access', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'notifications.realtime', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'strategy.standard_scalping', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'strategy.ultra_scalping', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'strategy.standard_swing', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'strategy.trend_swing', 'true'::jsonb),
    (SELECT id FROM control.plans WHERE code = 'ELITE', 'strategy.max_active_slots', '4'::jsonb)
ON CONFLICT (plan_id, entitlement_key) DO NOTHING;
