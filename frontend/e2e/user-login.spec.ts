import { test, expect } from '@playwright/test';
import { setupApiMocking, USER_TOKEN, USER_USER } from './mock-helpers';

test.describe('User Login Flow', () => {
  test('user login redirects to /dashboard/live with User Panel sidebar', async ({ page }) => {
    setupApiMocking(page, USER_TOKEN, USER_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'user@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');

    // Should redirect to user dashboard
    await page.waitForURL('**/dashboard/live', { timeout: 10000 });
    expect(page.url()).toContain('/dashboard/live');

    // Topbar should show user name
    await expect(page.locator('[data-testid="topbar-user-name"]')).toContainText('Simha User');

    // Sidebar footer panel label is intentionally empty (removed by design)
    await expect(page.locator('[data-testid="panel-label"]')).toHaveText('');

    // Sidebar should contain user-specific items
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('MetaTrader Client');
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('Referral & Earnings');
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('Billing & Subscription');

    // Sidebar should NOT contain admin-only items
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('Signal Monitor');
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('Risk Center');
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('Platform Operations');
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('System Health');
  });

  test('user navigating to /admin/dashboard redirects to /dashboard/live', async ({ page }) => {
    setupApiMocking(page, USER_TOKEN, USER_USER);

    // Login first
    await page.goto('/login');
    await page.fill('input[type="email"]', 'user@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard/live', { timeout: 10000 });

    // Try to navigate to admin dashboard
    await page.goto('/admin/dashboard');
    // Should be redirected back to user dashboard
    await page.waitForURL('**/dashboard/live', { timeout: 10000 });
    expect(page.url()).toContain('/dashboard/live');
  });
});
