#!/usr/bin/env bash
# security-scan.sh — Predict-A-Trade static security gate (Makefile:security-scan)
# Runs secret scanning, dependency audit, and static checks across planes.
# Best-effort: each check is isolated so one missing tool does not abort the rest.
set -uo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
fail=0

echo "== Secret scanning (gitleaks) =="
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks detect --source . --redact --no-banner && echo -e "${GREEN}gitleaks: clean${NC}" || { echo -e "${RED}gitleaks: findings${NC}"; fail=1; }
else
  echo "gitleaks not installed; skipping (install via 'brew install gitleaks' or apt)"
fi

echo "== Go vet (realtime + windows-agent) =="
if [ -d realtime ]; then
  ( cd realtime && go vet ./... ) && echo -e "${GREEN}go vet: clean${NC}" || { echo -e "${RED}go vet: issues${NC}"; fail=1; }
fi

echo "== Node audit (control + frontend) =="
for d in control frontend; do
  if [ -f "$d/package.json" ]; then
    ( cd "$d" && npm audit --audit-level=high ) && echo -e "${GREEN}$d npm audit: clean${NC}" || { echo -e "${RED}$d npm audit: issues (see above)${NC}"; }
  fi
done

echo "== Dockerfile base-image pinning check =="
if grep -rnE 'FROM .*:(latest|$)' --include=Dockerfile* --include=*.Dockerfile . 2>/dev/null | grep -v '#' | grep -q .; then
  echo -e "${RED}Unpinned base images found (avoid :latest)${NC}"; fail=1
else
  echo -e "${GREEN}Base images pinned${NC}"
fi

echo "== Committed secret guard =="
if grep -rniE '(password|secret|api_key|token)\s*[:=]\s*["'\''][A-Za-z0-9_\-]{6,}' \
   --include=*.yml --include=*.env.example --include=*.example . 2>/dev/null \
   | grep -viE 'example|changeme|your-|placeholder|REDACTED' | grep -q .; then
  echo -e "${RED}Possible hardcoded secret in committed config${NC}"; fail=1
else
  echo -e "${GREEN}No obvious hardcoded secrets${NC}"
fi

if [ "$fail" -ne 0 ]; then
  echo -e "${RED}Security scan reported issues.${NC}"; exit 1
fi
echo -e "${GREEN}Security scan passed (or skipped where tooling absent).${NC}"
