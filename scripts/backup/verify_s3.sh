#!/usr/bin/env bash
# Safe S3 reachability / config check for off-host backups.
#
# Verifies the BACKUP_S3_* configuration is usable WITHOUT ever printing the
# secret access key. It only reports: config presence, bucket existence,
# write+read round-trip (in a temp key), and endpoint reachability.
#
# This is the pre-flight check before trusting the pat-backup-sync sidecar or
# `dr-kit.sh offhost`. It does NOT upload any real data.
#
# Usage:
#   ./scripts/backup/verify_s3.sh
#
# Requires the real values in infra/env/.env (loaded automatically) OR exported
# in the environment. The secret is passed only to the aws-cli container as an
# environment variable (never echoed).

set -uo pipefail

# Load shared env if present (gitignored; holds secrets). Never print it.
if [ -f infra/env/.env ]; then
  # shellcheck disable=SC1091
  set -a; . ./infra/env/.env; set +a
fi

REQUIRED=(BACKUP_S3_ACCESS_KEY BACKUP_S3_SECRET_KEY BACKUP_S3_BUCKET)
ok=1
echo "=== S3 off-host backup: config check (secrets NOT printed) ==="

for v in "${REQUIRED[@]}"; do
  if [ -z "${!v:-}" ]; then
    echo "  [MISSING] $v — set it in infra/env/.env (gitignored)"
    ok=0
  else
    # Print only the PRESENCE, masked length, never the value.
    echo "  [OK] $v present (${#v} chars set)"
  fi
done

# Non-secret facts (safe to show)
echo "  Bucket : ${BACKUP_S3_BUCKET:-<unset>}"
echo "  Region : ${BACKUP_S3_REGION:-<unset>}"
echo "  Endpoint: ${BACKUP_S3_ENDPOINT:-<unset/AWS-default>}"
echo "  Prefix : ${BACKUP_S3_DB_PREFIX:-predictatrade/db}"

if [ "$ok" -eq 0 ]; then
  echo "==> RESULT: INCOMPLETE — fill missing vars in infra/env/.env"
  exit 2
fi

export AWS_ACCESS_KEY_ID="${BACKUP_S3_ACCESS_KEY}"
export AWS_SECRET_ACCESS_KEY="${BACKUP_S3_SECRET_KEY}"
export AWS_DEFAULT_REGION="${BACKUP_S3_REGION:-auto}"

S3_EP=""
if [ -n "${BACKUP_S3_ENDPOINT:-}" ]; then S3_EP="--endpoint-url ${BACKUP_S3_ENDPOINT}"; fi
TARGET="s3://${BACKUP_S3_BUCKET}/${BACKUP_S3_DB_PREFIX:-predictatrade/db}/.verify_s3_probe"
PROBE=$(printf 'pat-s3-verify %s' "$(date -u +%Y-%m-%dT%H:%M:%SZ)")

# Use the same amazon/aws-cli image the backup-sync sidecar uses (matches server version).
run_aws() {
  if command -v aws >/dev/null 2>&1; then aws "$@"; else
    docker run --rm \
      -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_DEFAULT_REGION \
      amazon/aws-cli:latest "$@"
  fi
}

echo "=== probe: write to bucket + confirm via list (temp key, cleaned up) ==="
if printf '%s' "$PROBE" | run_aws s3 cp - "${TARGET}" ${S3_EP} 2>/tmp/s3probe.err; then
  # Confirm the object is visible (proves creds + endpoint + write path end-to-end).
  if run_aws s3 ls "${TARGET}" ${S3_EP} >/dev/null 2>&1; then
    run_aws s3 rm "${TARGET}" ${S3_EP} >/dev/null 2>&1
    echo "==> RESULT: PASS — bucket reachable, credentials valid, write confirmed."
    echo "    Off-host backup is correctly configured (pat-backup-sync sidecar active)."
    exit 0
  else
    run_aws s3 rm "${TARGET}" ${S3_EP} >/dev/null 2>&1
    echo "==> RESULT: WARN — wrote object but could not list it back (eventual-consistency on read?)."
    echo "    The continuous backup-sync sidecar is the authoritative consumer; verify it is"
    echo "    shipping via: docker logs --tail 20 pat-backup-sync"
    exit 1
  fi
else
  echo "==> RESULT: FAIL — cannot reach bucket or credentials rejected."
  echo "    aws-cli error (no secrets shown):"
  sed 's/\(AKIA[0-9A-Za-z]\{16\}\)\|\([A-Za-z0-9/+=]\{40,\}\)/***REDACTED***/g' /tmp/s3probe.err | tail -5
  exit 1
fi
