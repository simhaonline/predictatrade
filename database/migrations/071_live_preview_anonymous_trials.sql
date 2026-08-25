-- 071: Server-enforced anonymous 5-minute live preview trials
-- for live.predictatrade.com (prompt.md funnel spec).
--
-- Privacy: only HMAC digests of IP/User-Agent are stored — never raw IPs,
-- device fingerprints, or the raw trial token. The cookie holds a random
-- token; the database can only match its HMAC.

CREATE SCHEMA IF NOT EXISTS live_preview;

CREATE TABLE IF NOT EXISTS live_preview.anonymous_trials (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_token_hash    TEXT NOT NULL UNIQUE,        -- HMAC-SHA256(secret, token)
    trial_started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    trial_expires_at      TIMESTAMPTZ NOT NULL,
    status                TEXT NOT NULL DEFAULT 'ACTIVE'
                          CHECK (status IN ('ACTIVE','EXPIRED','CONVERTED','BLOCKED','REVOKED')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    ip_hash               TEXT,                        -- HMAC(secret, normalized_ip)
    user_agent_hash       TEXT,                        -- HMAC(secret, normalized_ua)
    browser_family        TEXT,
    device_class          TEXT,

    registration_wall_seen_at TIMESTAMPTZ,
    signup_started_at         TIMESTAMPTZ,

    registered_user_id    UUID,
    converted_at          TIMESTAMPTZ,

    expiration_reason     TEXT,
    abuse_score           INTEGER NOT NULL DEFAULT 0,
    metadata              JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_lp_trials_status_expiry
    ON live_preview.anonymous_trials (status, trial_expires_at);
CREATE INDEX IF NOT EXISTS idx_lp_trials_ip_hash
    ON live_preview.anonymous_trials (ip_hash, created_at);
CREATE INDEX IF NOT EXISTS idx_lp_trials_registered_user
    ON live_preview.anonymous_trials (registered_user_id)
    WHERE registered_user_id IS NOT NULL;

-- Funnel analytics events (one row per event; dedup handled by writer).
CREATE TABLE IF NOT EXISTS live_preview.trial_events (
    id          BIGSERIAL PRIMARY KEY,
    trial_id    UUID NOT NULL REFERENCES live_preview.anonymous_trials(id) ON DELETE CASCADE,
    event       TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_lp_events_trial ON live_preview.trial_events (trial_id, event);
CREATE INDEX IF NOT EXISTS idx_lp_events_time ON live_preview.trial_events (event, occurred_at);

-- Admin funnel view (real records only; no raw tokens/hashes exposed).
CREATE OR REPLACE VIEW live_preview.funnel_stats AS
SELECT
    (SELECT count(*) FROM live_preview.anonymous_trials)                                          AS unique_preview_visitors,
    (SELECT count(*) FROM live_preview.anonymous_trials WHERE status = 'ACTIVE'
        AND trial_expires_at > now())                                                             AS active_previews,
    (SELECT count(*) FROM live_preview.anonymous_trials)                                          AS preview_starts,
    (SELECT count(*) FROM live_preview.anonymous_trials
        WHERE last_seen_at - trial_started_at >= interval '1 minute')                             AS reached_1_minute,
    (SELECT count(*) FROM live_preview.anonymous_trials
        WHERE last_seen_at - trial_started_at >= interval '3 minutes')                            AS reached_3_minutes,
    (SELECT count(*) FROM live_preview.anonymous_trials
        WHERE last_seen_at - trial_started_at >= interval '5 minutes')                            AS reached_5_minutes,
    (SELECT count(DISTINCT trial_id) FROM live_preview.trial_events
        WHERE event = 'REGISTRATION_WALL_SHOWN')                                                  AS registration_wall_reached,
    (SELECT count(DISTINCT trial_id) FROM live_preview.trial_events
        WHERE event = 'SIGNUP_STARTED')                                                           AS signup_started,
    (SELECT count(*) FROM live_preview.anonymous_trials WHERE status = 'CONVERTED')               AS signups_completed,
    (SELECT count(*) FROM live_preview.anonymous_trials WHERE status = 'BLOCKED')                 AS repeat_attempt_blocks,
    (SELECT count(*) FROM live_preview.anonymous_trials
        WHERE expiration_reason = 'NATURAL')                                                      AS expired_naturally;
