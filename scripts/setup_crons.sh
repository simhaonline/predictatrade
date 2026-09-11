#!/usr/bin/env bash
set -euo pipefail

# Production health verification (every 5 minutes) — now includes the Cloudflare
# edge posture (non-blocking) via verify_live_production.sh → verify-cloudflare.sh.
(crontab -l 2>/dev/null | grep -v "verify_live_production.sh"; echo "*/5 * * * * /srv/predictatrade/xauusd/scripts/verify_live_production.sh >> /srv/predictatrade/xauusd/logs/production_health.log 2>&1") | crontab -

# Weekly retraining (Sunday 2 AM)
(crontab -l 2>/dev/null | grep -v "run_training.sh"; echo "0 2 * * 0 /srv/predictatrade/xauusd/scripts/run_training.sh >> /srv/predictatrade/xauusd/logs/training_cron.log 2>&1") | crontab -

# DR kit — daily logical backup + restore self-test (off-host sync handled by the
# backup-sync sidecar; this guarantees a restorable dump exists every day).
(crontab -l 2>/dev/null | grep -v "dr-kit.sh backup"; echo "13 3 * * * /srv/predictatrade/xauusd/scripts/dr-kit.sh backup >> /srv/predictatrade/xauusd/logs/dr_backup.log 2>&1") | crontab -
(crontab -l 2>/dev/null | grep -v "dr-kit.sh restore-test"; echo "43 3 * * * /srv/predictatrade/xauusd/scripts/dr-kit.sh restore-test >> /srv/predictatrade/xauusd/logs/dr_restore_test.log 2>&1") | crontab -

# DR kit — daily SEPARATE codebase+config snapshot -> R2 predictatrade/code/
# (git bundle + working tree minus secrets + migration manifest). Distinct from
# the DB/WAL backups so a server migration has code, configs, AND database.
(crontab -l 2>/dev/null | grep -v "dr-kit.sh codebase"; echo "23 3 * * * /srv/predictatrade/xauusd/scripts/dr-kit.sh codebase >> /srv/predictatrade/xauusd/logs/dr_codebase.log 2>&1") | crontab -

# DR kit — weekly migration status report (Monday 4 AM)
(crontab -l 2>/dev/null | grep -v "dr-kit.sh migrate-status"; echo "17 4 * * 1 /srv/predictatrade/xauusd/scripts/dr-kit.sh migrate-status >> /srv/predictatrade/xauusd/logs/dr_migrate.log 2>&1") | crontab -

echo "✅ Crons installed successfully"
echo "  - Health check (incl. Cloudflare edge): every 5 minutes"
echo "  - Weekly retraining: Sunday 2 AM"
echo "  - DR backup: daily 03:13"
echo "  - DR restore self-test: daily 03:43"
echo "  - DR codebase snapshot (R2 predictatrade/code/): daily 03:23"
echo "  - DR migration status: Monday 04:17"
