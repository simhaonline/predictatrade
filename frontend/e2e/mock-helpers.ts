import type { Page, Route } from '@playwright/test';

/**
 * Create a mock JWT token with a given role.
 * The token is a valid JWT structure (header.payload.signature)
 * but the signature is fake — the frontend only decodes the payload,
 * it doesn't verify the signature (that's the backend's job).
 */
export function createMockJwt(role: string, userId = 'test-user-id'): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({
    sub: userId,
    email: `test-${role.toLowerCase()}@test.com`,
    role,
    purpose: 'access',
    exp: Math.floor(Date.now() / 1000) + 3600, // 1 hour from now
  })).toString('base64url');
  const signature = 'mock-signature';
  return `${header}.${payload}.${signature}`;
}

export const ADMIN_TOKEN = createMockJwt('ADMIN', 'admin-user-id');
export const SUPER_ADMIN_TOKEN = createMockJwt('SUPER_ADMIN', 'super-admin-user-id');
export const USER_TOKEN = createMockJwt('USER', 'regular-user-id');

interface MockUser {
  id: string;
  email: string;
  full_name: string;
  status: string;
  created_at: string;
}

export const ADMIN_USER: MockUser = {
  id: 'admin-user-id',
  email: 'admin@predictatrade.com',
  full_name: 'Simha Admin',
  status: 'ACTIVE',
  created_at: '2024-01-01T00:00:00Z',
};

export const SUPER_ADMIN_USER: MockUser = {
  id: 'super-admin-user-id',
  email: 'superadmin@predictatrade.com',
  full_name: 'Super Admin',
  status: 'ACTIVE',
  created_at: '2024-01-01T00:00:00Z',
};

export const USER_USER: MockUser = {
  id: 'regular-user-id',
  email: 'user@predictatrade.com',
  full_name: 'Simha User',
  status: 'ACTIVE',
  created_at: '2024-01-01T00:00:00Z',
};

/**
 * Set up API mocking for a page.
 * Intercepts all /api/v1/* calls and returns mock responses.
 */
export function setupApiMocking(page: Page, token: string, user: MockUser) {
  // Mock login
  page.route('**/api/v1/auth/login', async (route: Route) => {
    const body = route.request().postDataJSON();
    if (body?.email === 'admin@predictatrade.com') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          accessToken: token,
          user: { id: user.id, email: user.email, displayName: user.full_name },
        }),
      });
    } else if (body?.email === 'superadmin@predictatrade.com') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          accessToken: token,
          user: { id: user.id, email: user.email, displayName: user.full_name },
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          accessToken: token,
          user: { id: user.id, email: user.email, displayName: user.full_name },
        }),
      });
    }
  });

  // Mock /auth/me
  page.route('**/api/v1/auth/me', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(user),
    });
  });

  // Mock /auth/refresh
  page.route('**/api/v1/auth/refresh', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ accessToken: token }),
    });
  });

  // Mock /auth/logout
  page.route('**/api/v1/auth/logout', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true }),
    });
  });

  // Mock admin overview
  page.route('**/api/v1/admin/overview', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        users: { total: '10', active: '8', suspended: '1', new_this_month: '2' },
        subscriptions: { total: '5', active: '4', mrr: '500.00' },
        commissions: { total_entries: '20', pending_amount: '100.00', confirmed_amount: '500.00' },
        payouts: { total: '3', pending: '1', pending_amount: '50.00' },
        plans: { total: '3', active: '3' },
      }),
    });
  });

  // Mock admin health
  page.route('**/api/v1/admin/health', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        services: [
          { service: 'database', status: 'healthy', latency_ms: 5, last_check: new Date().toISOString() },
          { service: 'valkey', status: 'healthy', latency_ms: 2, last_check: new Date().toISOString() },
        ],
      }),
    });
  });

  // Mock Go engine signals
  page.route('**/api/v1/signals', async (route: Route) => {
    // This might be the Go engine or NestJS — check the URL pattern
    const url = route.request().url();
    if (url.includes('localhost:13082') || url.includes('platform.predictatrade.com')) {
      // NestJS control plane — return empty
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ signals: [] }),
      });
    } else {
      // Go engine
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ signals: [] }),
      });
    }
  });

  // Mock operations state
  page.route('**/api/v1/operations/state', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        trading_halted: false,
        signals_paused: false,
        active_strategies: ['STANDARD_SCALPING', 'ULTRA_SCALPING', 'STANDARD_SWING', 'TREND_SWING'],
        last_updated: new Date().toISOString(),
      }),
    });
  });

  // Mock billing invoices
  page.route('**/api/v1/billing/invoices', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock subscriptions
  page.route('**/api/v1/subscriptions', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock plans
  page.route('**/api/v1/plans', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock licensing devices
  page.route('**/api/v1/licensing/devices', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock licensing mt-accounts
  page.route('**/api/v1/licensing/mt-accounts', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock commissions
  page.route('**/api/v1/commissions', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock commissions summary
  page.route('**/api/v1/commissions/summary', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        total_amount: '0',
        pending_count: '0',
        confirmed_count: '0',
        pending_amount: '0',
        confirmed_amount: '0',
      }),
    });
  });

  // Mock admin endpoints (list responses)
  page.route('**/api/v1/admin/users*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, limit: 20 }),
    });
  });

  page.route('**/api/v1/admin/subscriptions*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, limit: 20 }),
    });
  });

  page.route('**/api/v1/admin/commissions*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, limit: 20 }),
    });
  });

  page.route('**/api/v1/admin/payouts*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, limit: 20 }),
    });
  });

  page.route('**/api/v1/admin/licenses*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, limit: 20 }),
    });
  });

  page.route('**/api/v1/admin/devices*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, limit: 20 }),
    });
  });

  // Mock audit
  page.route('**/api/v1/audit*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, limit: 20 }),
    });
  });

  // Mock devices sessions
  page.route('**/api/v1/devices/sessions', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock referrals
  page.route('**/api/v1/referrals*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock payouts
  page.route('**/api/v1/payouts*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  // Mock Go engine market state
  page.route('**/api/v1/market/state', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        symbol: 'XAUUSD',
        bid: 2500.00,
        ask: 2500.50,
        spread: 0.50,
        session: 'LONDON',
        regime: 'RANGE_BOUND',
        timestamp: new Date().toISOString(),
      }),
    });
  });

  // Mock Go engine market snapshot
  page.route('**/api/v1/market/snapshot', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        indicators: {},
        structure: { swing_highs: [], swing_lows: [], bos: false, choch: false },
      }),
    });
  });

  // Mock Go engine agents status
  page.route('**/api/v1/agents/status', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });
}
