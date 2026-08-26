---
name: git-workflow-cicd
description: "Git hooks, CI pipelines, Docker builds, releases."
---

# git-workflow-cicd

Use for PAT git workflow, CI, releases.

## Auto-Push
git add -A, git commit -m "...", git push origin main — always.

## Branches
main (protected), feature/*, hotfix/*

## CI (6 GitHub Actions)
Go: go test -race, go vet, golangci-lint
NestJS: npm test, npm lint, npm test:e2e
Frontend: tsc, eslint, test, build
Python: pytest, ruff
Windows Agent: go build, checksum
Security: govulncheck, npm audit, secrets

## Docker
docker compose build <svc> && docker compose up -d <svc>
Windows: scripts/build-windows-agent.sh --bump

## Commands
make test, make lint, make build, make format
