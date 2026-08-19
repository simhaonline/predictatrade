import { test, expect } from '@playwright/test';
import {
  setupApiMocking,
  ADMIN_TOKEN, ADMIN_USER,
  USER_TOKEN, USER_USER,
  SUPER_ADMIN_TOKEN, SUPER_ADMIN_USER,
} from './mock-helpers';

test.describe('Direct URL RBAC', () => {
  test('admin direct URL to /dashboard/live is redirected', async ({ page, context }) => {
    await context.addCookies([{
      name: 'pat_access_token',
      value: ADMIN_TOKEN,
      domain: 'localhost',
      path: '/',
    }]);

    setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);

    await page.goto('/dashboard/live');
    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });
    expect(page.url()).toContain('/admin/dashboard');
  });

  test('user direct URL to /admin/dashboard is redirected', async ({ page, context }) => {
    await context.addCookies([{
      name: 'pat_access_token',
      value: USER_TOKEN,
      domain: 'localhost',
      path: '/',
    }]);

    setupApiMocking(page, USER_TOKEN, USER_USER);

    await page.goto('/admin/dashboard');
    await page.waitForURL('**/dashboard/live', { timeout: 10000 });
    expect(page.url()).toContain('/dashboard/live');
  });

  test('super_admin login redirects to /admin/dashboard', async ({ page }) => {
    setupApiMocking(page, SUPER_ADMIN_TOKEN, SUPER_ADMIN_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'superadmin@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');

    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });
    expect(page.url()).toContain('/admin/dashboard');
    await expect(page.locator('[data-testid="panel-label"]')).toContainText('Admin Panel');
  });
});
