#!/usr/bin/env bash
# Predict-A-Trade — CODEBASE backup (separate from DB/WAL).
#
# Produces an off-host, server-migration-ready snapshot of the CODE + CONFIG:
#   1. Git bundle of the whole repo (all branches/tags) — full history, portable.
#   2. Tar of the working tree, EXCLUDING any *.env / secrets, so untracked
#      nginx sites / compose overrides are captured without leaking keys.
#   3. A migration manifest (commit, docker images, volumes, and a LIST of
#      secret files by PATH ONLY — never their contents) so a new server can be
#      reconstructed. Secret VALUES never leave the host.
#
# Pushed to R2 under predictatrade/code/ (separate from wal/ and db/).
#
# SOW Sections 81-91 (Backup). Safe to run on a schedule.
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/predictatrade}"
CODE_PREFIX="${BACKUP_S3_CODE_PREFIX:-predictatrade/code}"
TS=$(date -u +%Y%m%d_%H%M%S_UTC)
HOST=$(hostname)

# Load BACKUP_S3_* from the gitignored .env (single source of truth).
if [ -f "${REPO_DIR}/infra/env/.env" ]; then
  set -a; . "${REPO_DIR}/infra/env/.env"; set +a
fi
: "${BACKUP_S3_BUCKET:?set BACKUP_S3_BUCKET in infra/env/.env}"
: "${BACKUP_S3_ENDPOINT:?set BACKUP_S3_ENDPOINT in infra/env/.env}"
: "${BACKUP_S3_ACCESS_KEY:?set BACKUP_S3_ACCESS_KEY in infra/env/.env}"
: "${BACKUP_S3_SECRET_KEY:?set BACKUP_S3_SECRET_KEY in infra/env/.env}"

mkdir -p "${BACKUP_DIR}"
BUNDLE="${BACKUP_DIR}/code_bundle_${TS}.bundle"
TREE="${BACKUP_DIR}/code_tree_${TS}.tar.gz"
MANIFEST="${BACKUP_DIR}/code_manifest_${TS}.txt"

echo "[codebase-backup] repo=${REPO_DIR} ts=${TS}"

# 1. Git bundle (all refs -> portable single file)
echo "[codebase-backup] creating git bundle..."
git -C "${REPO_DIR}" bundle create "${BUNDLE}" --all 2>&1 | tail -2

# 2. Tar working tree EXCLUDING secrets/env + build artifacts
echo "[codebase-backup] archiving working tree (excluding secrets)..."
tar --exclude="./.git" \
    --exclude="*.env" \
    --exclude="infra/env/*.env" \
    --exclude="node_modules" \
    --exclude="*/node_modules" \
    --exclude="dist" --exclude=".next" --exclude="build" \
    --exclude="*.log" \
    -czf "${TREE}" -C "${REPO_DIR}" . 2>/dev/null || echo "[codebase-backup] tar completed with warnings"

# 3. Migration manifest (paths/metadata only — NO secret values)
echo "[codebase-backup] writing migration manifest..."
{
  echo "# Predict-A-Trade codebase backup manifest"
  echo "generated_utc: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo "host: ${HOST}"
  echo "repo_dir: ${REPO_DIR}"
  echo "git_commit: $(git -C "${REPO_DIR}" rev-parse HEAD)"
  echo "git_commit_short: $(git -C "${REPO_DIR}" rev-parse --short HEAD)"
  echo "git_branch: $(git -C "${REPO_DIR}" rev-parse --abbrev-ref HEAD)"
  echo "git_dirty: $(git -C "${REPO_DIR}" status --porcelain | wc -l) untracked/modified files"
  echo "docker_images:"; docker images --format '  - {{.Repository}}:{{.Tag}} ({{.ID}})' 2>/dev/null | head -40
  echo "docker_volumes:"; docker volume ls --format '  - {{.Name}}' 2>/dev/null | head -40
  echo "compose_project: $(docker compose -f "${REPO_DIR}/docker-compose.yml" ls 2>/dev/null | tail -1 || echo n/a)"
  echo ""
  echo "## SECRET FILES (recreate on new server — VALUES NOT included):"
  find "${REPO_DIR}" -name '*.env' -not -name '*.example' 2>/dev/null | sed 's#^#  - #'
  echo "  - infra/env/.env  (BACKUP_S3_*, POSTGRES_PASSWORD, DATABASE_URL, JWT_SECRET, etc.)"
  echo ""
  echo "## RESTORE STEPS (new server):"
  echo "  1. clone/pull repo OR: git clone <repo> && git bundle unbundle code_bundle_<ts>.bundle"
  echo "  2. docker compose pull && docker compose up -d"
  echo "  3. recreate secret files listed above (use infra/env/.env.example as template)"
  echo "  4. restore DB: dr-kit.sh restore-test  (or restore latest predictatrade/db dump)"
} > "${MANIFEST}"

# Checksums
echo "[codebase-backup] checksums..."
sha256sum "${BUNDLE}" | awk '{print $1}' > "${BUNDLE}.sha256"
sha256sum "${TREE}"   | awk '{print $1}' > "${TREE}.sha256"
sha256sum "${MANIFEST}" | awk '{print $1}' > "${MANIFEST}.sha256"

# Upload to R2 under predictatrade/code/
echo "[codebase-backup] uploading to s3://${BACKUP_S3_BUCKET}/${CODE_PREFIX}/ ..."
S3_EP="--endpoint-url ${BACKUP_S3_ENDPOINT}"
for f in "${BUNDLE}" "${BUNDLE}.sha256" "${TREE}" "${TREE}.sha256" "${MANIFEST}" "${MANIFEST}.sha256"; do
  docker run --rm \
    -v "${f}:/obj:ro" \
    -e AWS_ACCESS_KEY_ID="${BACKUP_S3_ACCESS_KEY}" \
    -e AWS_SECRET_ACCESS_KEY="${BACKUP_S3_SECRET_KEY}" \
    -e AWS_DEFAULT_REGION="${BACKUP_S3_REGION:-auto}" \
    amazon/aws-cli:latest s3 cp "/obj" "s3://${BACKUP_S3_BUCKET}/${CODE_PREFIX}/$(basename "${f}")" ${S3_EP} 2>&1 | tail -1
done

# Local retention (30d)
find "${BACKUP_DIR}" -name 'code_*' -mtime +30 -delete 2>/dev/null

echo "[codebase-backup] DONE -> s3://${BACKUP_S3_BUCKET}/${CODE_PREFIX}/"
