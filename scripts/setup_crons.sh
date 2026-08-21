#!/usr/bin/env bash
set -euo pipefail

# Add production health verification (every 5 minutes)
(crontab -l 2>/dev/null | grep -v "verify_live_production.sh"; echo "*/5 * * * * /srv/predictatrade/xauusd/scripts/verify_live_production.sh >> /srv/predictatrade/xauusd/logs/production_health.log 2>&1") | crontab -

# Add weekly retraining (Sunday 2 AM)
(crontab -l 2>/dev/null | grep -v "run_training.sh"; echo "0 2 * * 0 /srv/predictatrade/xauusd/scripts/run_training.sh >> /srv/predictatrade/xauusd/logs/training_cron.log 2>&1") | crontab -

echo "✅ Crons installed successfully"
echo "  - Health check: every 5 minutes"
echo "  - Weekly retraining: Sunday 2 AM"
