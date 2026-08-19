# RBAC Routing Fix Report

## Observed Defect

An authenticated Admin user ("Simha Admin") was landing at `/dashboard/live` (User Panel) instead of `/admin/dashboard` (Admin Panel). The topbar correctly showed the admin's name, but the sidebar displayed the User Panel menu with "User Panel" footer label.

## Actual Root Cause

Multiple bugs in the authentication → role resolution → redirect → layout → sidebar chain:

### Bug 1: Login redirect only checked `role === 'ADMIN'`, missing `SUPER_ADMIN`

**File:** `src/providers/auth-provider.tsx`

```typescript
// OLD (buggy):
if (userFromLogin?.role === 'ADMIN') {
  router.push('/admin/dashboard');
} else {
  router.push('/dashboard/live');  // ← SUPER_ADMIN goes here!
}
```

The backend `AdminGuard` accepts both `ADMIN` and `SUPER_ADMIN`. The backend `generateTokens` method looks up role from `iam.roles` table, which can return `SUPER_ADMIN`. The login redirect only checked `=== 'ADMIN'`, so `SUPER_ADMIN` users were sent to `/dashboard/live`.

### Bug 2: `fetchMe()` used stale token for role extraction

**File:** `src/providers/auth-provider.tsx`

```typescript
// OLD (buggy):
const token = getAccessToken();  // May be expired
const res = await customInstance.get('/auth/me');  // Interceptor refreshes token
const fetchedUser = normalizeUser(res.data, token);  // Uses OLD expired token!
```

When the access token expired (15 min TTL) and the axios interceptor refreshed it, `normalizeUser` still used the old expired token for role extraction. `getRoleFromToken(expiredToken)` returns `null`, so `role` defaulted to `'USER'`, causing admin users to appear as regular users after token refresh.

### Bug 3: Proxy did not redirect admins away from `/dashboard/*`

**File:** `src/proxy.ts` (formerly `src/middleware.ts`)

The proxy only blocked non-admins from `/admin/*`. It did not block admins from `/dashboard/*`. An admin could navigate to `/dashboard/live` and stay there indefinitely.

### Bug 4: Sidebar defaulted to User Panel when `user === null` (session loading)

**File:** `src/components/layout/sidebar.tsx`

```typescript
// OLD (buggy):
const isAdmin = user?.role === "ADMIN" || user?.role === "SUPER_ADMIN";
const items = isAdmin ? adminItems : userItems;  // ← user null → userItems!
```

During session hydration (page refresh), `user` was `null` while `fetchMe()` was in progress. The sidebar showed User Panel items by default, even for admin users.

### Bug 5: No loading boundary in AppShell

**File:** `src/components/layout/app-shell.tsx`

Both admin and user layouts rendered `AppShell` immediately, even while session was loading. The sidebar rendered with `user = null`, showing User Panel.

### Bug 6: `isAdmin()` in `auth.ts` only checked `ADMIN`

**File:** `src/lib/auth.ts`

```typescript
// OLD (buggy):
export function isAdmin(user: User | null): boolean {
  return user?.role === 'ADMIN';  // ← Missing SUPER_ADMIN!
}
```

### Bug 7: No client-side cross-role redirect

There was no client-side guard to redirect an admin away from `/dashboard/*` or a user away from `/admin/*` if they somehow landed on the wrong route (e.g., via direct URL entry with a valid token).

## Files Involved

1. `src/providers/auth-provider.tsx` — Auth provider, login redirect, session hydration
2. `src/lib/auth.ts` — Token management, role extraction, `isAdmin()` helper
3. `src/components/layout/sidebar.tsx` — Sidebar menu selection, panel label
4. `src/components/layout/app-shell.tsx` — Application shell, loading boundary
5. `src/proxy.ts` — Server-side route guard (middleware)
6. `src/lib/axios-instance.ts` — Axios interceptor, refresh logic
7. `src/components/layout/topbar.tsx` — Topbar user display
8. `src/providers/query-provider.tsx` — React Query cache clearing

## Why Admin Was Routed to User Panel

1. Admin logs in with role `SUPER_ADMIN`
2. Login redirect: `userFromLogin?.role === 'ADMIN'` → false (it's `SUPER_ADMIN`)
3. Admin is sent to `/dashboard/live` instead of `/admin/dashboard`
4. Proxy does not redirect admin away from `/dashboard/*`
5. User layout renders `AppShell` → `Sidebar` with `user` initially null
6. Sidebar defaults to User Panel (because `user === null → isAdmin = false`)
7. `fetchMe()` completes, role is extracted from JWT
8. If token was refreshed during `fetchMe()`, role extraction uses stale token → `USER`
9. Sidebar remains User Panel

## Authentication Role Received From Backend

The backend JWT contains a `role` claim populated from the `iam.roles` table:
- `SUPER_ADMIN` — highest privilege
- `ADMIN` — admin access
- `USER` — default

The `AdminGuard` (backend) accepts both `ADMIN` and `SUPER_ADMIN`.

## Role Normalization (New)

Created `src/lib/roles.ts` with canonical helpers:
- `isAdminRole(role)` — returns true for `ADMIN` and `SUPER_ADMIN`
- `isUserRole(role)` — returns true for non-admin roles
- `homeRouteForRole(role)` — returns `/admin/dashboard` or `/dashboard/live`
- `panelLabelForRole(role)` — returns "Admin Panel" or "User Panel"

## Old Redirect Behavior

- Login: `role === 'ADMIN'` → `/admin/dashboard`, else → `/dashboard/live`
- Proxy: only blocked non-admins from `/admin/*`
- No client-side cross-role redirect

## New Redirect Behavior

- Login: `isAdminRole(role)` → `/admin/dashboard` (via `homeRouteForRole()`)
- Proxy: admins on `/dashboard/*` → redirect to `/admin/dashboard`
- Proxy: non-admins on `/admin/*` → redirect to `/dashboard/live`
- Client-side: `AppShell` also redirects cross-role as a safety net
- `router.replace()` used instead of `router.push()` to avoid history entries

## Old Layout Behavior

- Both admin and user layouts used the same `AppShell`
- `AppShell` rendered `Sidebar` immediately, even during loading
- `Sidebar` defaulted to User Panel when `user === null`

## New Layout Behavior

- `AppShell` shows a loading spinner while `sessionState === 'LOADING'`
- `Sidebar` uses route + role to determine navigation, not just role
- Separate navigation configs: `admin-navigation.ts`, `user-navigation.ts`
- Loading state shows "Loading…" label, not "User Panel"

## Admin Navigation

Live Dashboard, Signal Panel, Indicator Panel, Strategy Panel, Scoring Board, Activations, License Management, User Onboarding, Subscription Management, Billing & Payouts, Referral & Commissions, Device Auth, Trading Reports, Backtesting Reports, Logs & Audit, Platform Operations, System Health, Settings

## User Navigation

Live Dashboard, Signals, MT4/MT5 Client, Strategy Preferences, Trading Reports, Backtest, Referral & Earnings, Billing & Subscription, Settings

## Route Guard Behavior

| Scenario | Proxy (server) | AppShell (client) |
|----------|----------------|-------------------|
| Unauthenticated → /admin/* | Redirect to /login | N/A (proxy handles) |
| Unauthenticated → /dashboard/* | Redirect to /login | N/A (proxy handles) |
| Admin → /dashboard/* | Redirect to /admin/dashboard | Redirect to /admin/dashboard |
| User → /admin/* | Redirect to /dashboard/live | Redirect to /dashboard/live |
| Admin → /admin/* | Allow | Allow |
| User → /dashboard/* | Allow | Allow |

## Tests Added

1. `src/lib/__tests__/roles.test.ts` — Role resolver tests (ADMIN, SUPER_ADMIN, USER → correct routes)
2. `src/components/__tests__/sidebar.test.tsx` — Admin sidebar shows correct items, no user-only items
3. `src/components/__tests__/navigation-separation.test.tsx` — User sidebar shows correct items, no admin-only items
4. `src/components/__tests__/loading-state.test.tsx` — Loading state shows "Loading…" not "User Panel"

## Test Results

- Lint: 0 errors, 1 warning (cosmetic)
- TypeScript: PASS (0 errors)
- Unit tests: 13 suites, 47 tests — all PASS
- Production build: PASS (39 routes)

## Production Verification

- Frontend server: active on port 13082
- All public routes: HTTP 200
- Protected routes (no auth): HTTP 307 (redirect to login)
- Static CSS/JS chunks: HTTP 200 (no 500 errors)
- Root redirect: HTTP 307 (to login or dashboard based on auth)

## Token Expiry Edge Case Fix

### Problem

When an access token expired (15-minute TTL) and the axios interceptor refreshed it, `normalizeUser` in the auth provider used the old expired token for role extraction. `getRoleFromToken(expiredToken)` returned `null` (due to expiry check), causing the role to default to `'USER'` — even for admin users.

### Fix

Created `getRoleFromTokenUnchecked()` in `src/lib/auth.ts` that extracts the `role` claim from the JWT payload WITHOUT checking expiry. This is safe because:

1. The role claim is baked into the JWT at signing time and does not change
2. The backend already validated the token during the API call
3. The axios interceptor will refresh the token for subsequent API calls
4. This is only used for UI display (sidebar, routing), not for API authorization

The fallback chain in `normalizeUser`:
```typescript
const role = getRoleFromToken(token)           // 1. Try current token (may be refreshed)
  || getRoleFromTokenUnchecked(token)           // 2. Fallback: extract role ignoring expiry
  || 'USER';                                    // 3. Last resort: default to USER
```

### Tests

11 dedicated tests in `src/lib/__tests__/token-expiry.test.ts`:
- ADMIN role preserved for expired token
- SUPER_ADMIN role preserved for expired token
- USER role preserved for expired token
- Invalid token returns null
- Null token returns null
- Token without role claim returns USER
- Fallback chain works correctly for all cases

## E2E Browser Tests

### Implementation

10 Playwright E2E tests using API mocking (no real credentials needed):

1. **admin-login.spec.ts**: Admin login → /admin/dashboard with Admin Panel sidebar, admin cannot stay on /dashboard/live
2. **user-login.spec.ts**: User login → /dashboard/live with User Panel sidebar, user cannot access /admin/dashboard
3. **role-routing.spec.ts**: Direct URL RBAC (admin on /dashboard → redirect, user on /admin → redirect, SUPER_ADMIN login)
4. **navigation-separation.spec.ts**: Admin sidebar has exactly 18 correct items, user sidebar has exactly 9 correct items
5. **sequential-login.spec.ts**: Admin login → logout → user login changes navigation correctly

### E2E Test Results

```
✓  1  admin login redirects to /admin/dashboard with Admin Panel sidebar
✓  2  admin navigating to /dashboard/live redirects to /admin/dashboard
✓  3  admin sidebar has exactly 18 items with correct labels
✓  4  user sidebar has exactly 9 items with correct labels
✓  5  admin direct URL to /dashboard/live is redirected
✓  6  user direct URL to /admin/dashboard is redirected
✓  7  super_admin login redirects to /admin/dashboard
✓  8  admin login → logout → user login changes navigation
✓  9  user login redirects to /dashboard/live with User Panel sidebar
✓ 10  user navigating to /admin/dashboard redirects to /dashboard/live

10 passed (3.9s)
```

## Final Test Results

| Check | Result |
|-------|--------|
| Lint | 0 errors, 2 warnings (cosmetic) |
| TypeScript | PASS (0 errors) |
| Unit tests | 14 suites, 58 tests — all PASS |
| E2E tests | 10 tests — all PASS |
| Production build | 39 routes — PASS |
| Server | active on port 13082 |
| Smoke tests | All routes correct HTTP codes |
