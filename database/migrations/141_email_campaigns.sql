-- 141_email_campaigns.sql — admin email-notification campaigns
--
-- Powers the admin dashboard "Email Notifications" page: client alerts,
-- newsletters, and marketing sends. Design rules:
--   * Campaign = one composed email + audience filter + send batch. Immutable
--     after send (history is evidence; no silent edits).
--   * Per-recipient status tracking (sent/failed/skipped-unsubscribed) so
--     partial-failure retries never double-send.
--   * Compliance (CAN-SPAM/GDPR): recipients marked unsubscribed in
--     iam.marketing_unsubscribes are EXCLUDED from marketing/newsletter
--     campaigns at send time (transactional 'alert' sends go to all clients —
--     they are service communications, not marketing).
--   * Every campaign is audit-logged (actor, subject, audience, counts).

CREATE TABLE IF NOT EXISTS control.email_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by UUID NOT NULL,
    subject TEXT NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT NOT NULL,
    campaign_type TEXT NOT NULL DEFAULT 'newsletter'
        CHECK (campaign_type IN ('alert', 'newsletter', 'marketing')),
    audience TEXT NOT NULL DEFAULT 'all_clients'
        CHECK (audience IN ('all_clients', 'active_subscribers', 'marketing_opt_in')),
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'sending', 'sent', 'failed', 'cancelled')),
    total_recipients INTEGER NOT NULL DEFAULT 0,
    sent_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS control.email_campaign_recipients (
    id BIGSERIAL PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES control.email_campaigns(id) ON DELETE CASCADE,
    user_id UUID,
    email TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed', 'skipped_unsubscribed', 'skipped_bounced')),
    error TEXT,
    sent_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_email_campaign_recipients_campaign
    ON control.email_campaign_recipients (campaign_id, status);
CREATE INDEX IF NOT EXISTS idx_email_campaigns_created
    ON control.email_campaigns (created_at DESC);

COMMENT ON TABLE control.email_campaigns IS
'Admin email campaigns (alerts/newsletters/marketing). Immutable history; per-recipient delivery tracking lives in email_campaign_recipients.';
COMMENT ON TABLE control.email_campaign_recipients IS
'Per-recipient delivery rows for admin email campaigns — status tracks sent/failed/skipped so retries never double-send.';

-- Register in the migration ledger (idempotent).
INSERT INTO audit.migration_history (filename, status, started_at, completed_at)
VALUES ('141_email_campaigns.sql', 'applied', now(), now())
ON CONFLICT (filename) DO NOTHING;