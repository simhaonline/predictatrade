-- 142_customer_feedback.sql — customer feedback with admin moderation + Featured toggle
--
-- Product spec:
--   * Any authenticated client submits feedback from the user dashboard:
--     category, rating (1-5), message. Rate-limited at the controller.
--   * Admin sees everything in the admin console (Customer Management section),
--     can hide abusive/spam items (moderation) and mark items FEATURED.
--   * `featured` is the requirement toggle: featured feedback can be surfaced
--     on public marketing surfaces later. One flag, admin-controlled, audited.
--   * Feedback is immutable user-side after submission (no edit/delete API) —
--     moderation is admin-only. Soft-hide instead of hard delete keeps history.

CREATE TABLE IF NOT EXISTS control.customer_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    email TEXT NOT NULL,
    full_name TEXT,
    category TEXT NOT NULL
        CHECK (category IN ('bug', 'feature_request', 'usability', 'performance', 'billing', 'other')),
    rating INTEGER NOT NULL
        CHECK (rating BETWEEN 1 AND 5),
    message TEXT NOT NULL
        CHECK (length(message) BETWEEN 10 AND 2000),
    status TEXT NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'reviewed', 'hidden')),
    featured BOOLEAN NOT NULL DEFAULT false,
    featured_at TIMESTAMPTZ,
    featured_by UUID,
    admin_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_customer_feedback_status
    ON control.customer_feedback (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_customer_feedback_featured
    ON control.customer_feedback (featured, created_at DESC) WHERE featured = true;
CREATE INDEX IF NOT EXISTS idx_customer_feedback_user
    ON control.customer_feedback (user_id, created_at DESC);

COMMENT ON TABLE control.customer_feedback IS
'Customer feedback submissions (category + rating + message). Admin moderation: status new/reviewed/hidden (soft-hide, never deleted) and a Featured toggle for surfacing on marketing surfaces.';

-- Register in the migration ledger (idempotent).
INSERT INTO audit.migration_history (filename, status, started_at, completed_at)
VALUES ('142_customer_feedback.sql', 'applied', now(), now())
ON CONFLICT (filename) DO NOTHING;