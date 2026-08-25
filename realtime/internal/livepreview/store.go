package livepreview

import (
	"database/sql"
	"time"
)

// DBStore persists trials to live_preview.anonymous_trials (migration 071).
type DBStore struct{ DB *sql.DB }

func (d *DBStore) GetByTokenHash(hash string) (*Trial, error) {
	row := d.DB.QueryRow(`
		SELECT visitor_token_hash, trial_started_at, trial_expires_at, status,
		       ip_hash, user_agent_hash, browser_family, device_class,
		       registration_wall_seen_at, signup_started_at, abuse_score,
		       expiration_reason, last_seen_at
		FROM live_preview.anonymous_trials WHERE visitor_token_hash = $1`, hash)
	var stored Trial
	var wallSeen, signupStarted sql.NullTime
	var ipHash, uaHash, browser, device, expReason sql.NullString
	var abuse int
	if err := row.Scan(&stored.TokenHash, &stored.StartedAt, &stored.ExpiresAt, &stored.Status,
		&ipHash, &uaHash, &browser, &device,
		&wallSeen, &signupStarted, &abuse, &expReason, &stored.lastSeenSynced); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	stored.IPHash, stored.UAHash = ipHash.String, uaHash.String
	stored.BrowserFamily, stored.DeviceClass = browser.String, device.String
	stored.AbuseScore = abuse
	stored.ExpirationReasonDB = expReason.String
	if wallSeen.Valid {
		w := wallSeen.Time
		stored.WallSeenAt = &w
	}
	if signupStarted.Valid {
		su := signupStarted.Time
		stored.SignupStartedAt = &su
	}
	return &stored, nil
}

func (d *DBStore) Insert(t *Trial) error {
	_, err := d.DB.Exec(`
		INSERT INTO live_preview.anonymous_trials
			(visitor_token_hash, trial_started_at, trial_expires_at, status,
			 ip_hash, user_agent_hash, browser_family, device_class, abuse_score, last_seen_at)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9,$2)`,
		t.TokenHash, t.StartedAt, t.ExpiresAt, t.Status,
		t.IPHash, t.UAHash, t.BrowserFamily, t.DeviceClass, t.AbuseScore)
	return err
}

func (d *DBStore) Save(t *Trial) error {
	_, err := d.DB.Exec(`
		UPDATE live_preview.anonymous_trials SET
			status = $2, updated_at = now(), last_seen_at = $3,
			registration_wall_seen_at = $4, signup_started_at = $5,
			expiration_reason = NULLIF($6,'')
		WHERE visitor_token_hash = $1`,
		t.TokenHash, t.Status, time.Now().UTC(),
		t.WallSeenAt, t.SignupStartedAt, t.ExpirationReasonDB)
	return err
}

// CountRecent counts prior trials from the same coarse signal window — used
// only as a scored repeat-visitor signal, never as a sole blocker (§26).
func (d *DBStore) CountRecent(ipHash, uaHash string, since time.Time) (int, error) {
	var n int
	err := d.DB.QueryRow(`
		SELECT count(*) FROM live_preview.anonymous_trials
		WHERE created_at > $3 AND ip_hash = NULLIF($1,'') AND user_agent_hash = NULLIF($2,'')`,
		ipHash, uaHash, since).Scan(&n)
	return n, err
}

func (d *DBStore) RecordEvent(tokenHash, event string) error {
	_, err := d.DB.Exec(`
		INSERT INTO live_preview.trial_events (trial_id, event)
		SELECT id, $2 FROM live_preview.anonymous_trials WHERE visitor_token_hash = $1`,
		tokenHash, event)
	return err
}
