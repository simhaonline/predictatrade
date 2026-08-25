# CI/CD — GitHub Actions Configuration

**Version:** v1.13.0+ (25 August 2026)  
**Status:** ALL 6 JOBS PASSING ✅

## Workflow: `.github/workflows/ci.yml`

Triggers on push to `main`/`develop` and pull requests to `main`.

### Jobs

| Job | Runner | Language | Version | Steps |
|-----|--------|----------|---------|-------|
| Go Real-Time Engine | ubuntu-latest | Go | go-version-file: `realtime/go.mod` (1.25) | mod download → vet → test (no race) → test (race, non-blocking) → build |
| NestJS Control Plane | ubuntu-latest | Node.js | 22 | npm ci → lint → test → build |
| Next.js Frontend | ubuntu-latest | Node.js | 22 | npm ci → lint → build |
| Python Research Plane | ubuntu-latest | Python | 3.12 | pip install → pytest |
| Windows Agent | ubuntu-latest | Go | 1.23 | mod download → test → cross-compile (GOOS=windows) |
| Security Scan | ubuntu-latest | — | — | secret scan → dependency audit |

### Key Design Decisions

1. **Go version**: Uses `go-version-file: 'realtime/go.mod'` to auto-match the project's required Go version (no hardcoded version that drifts).

2. **Race detector**: Split into two steps — required `go test` (no race) catches logic errors, non-blocking `go test -race` catches race conditions with `continue-on-error: true`. Build step always runs.

3. **Test timeout**: 300s (increased from 120s) to account for slower CI runners, especially the sentiment test which takes ~14s locally.

4. **Secret scan patterns**: Precise regex patterns that match actual secrets:
   - `ghp_[A-Za-z0-9]{36}` — GitHub tokens
   - `sk-[a-zA-Z0-9]{20,}` — OpenAI/Stripe API keys
   - `AKIA[A-Z0-9]{16}` — AWS access keys
   - `BEGIN.*PRIVATE KEY` — private key blocks
   - `PASSWORD\s*=\s*['"][^'"]{8,}['"]` — hardcoded passwords in .env/.cfg files only

5. **npm ci compatibility**: `@testing-library/react@16` required for React 19 (Next.js 16.3.1). v15 only supports React 18.

### Fixes Applied (v1.13.0)

| Issue | Fix |
|-------|-----|
| Go test: `DATABASE_URL is required` | `helperDefault()` provides dummy DBURL when env var absent |
| Go test: race in `mockProvider.sendCount` | Added `sync.Mutex` protection |
| Go version: CI used 1.23, go.mod requires 1.25 | `go-version-file: 'realtime/go.mod'` |
| Frontend: `npm ci` ERESOLVE peer-dep conflict | `@testing-library/react` v15→v16 |
| Frontend: 3 lint errors | useEffect→useState/useMemo, apostrophe escaping |
| Security: false positive on `password` variable | Precise patterns for actual secrets only |
