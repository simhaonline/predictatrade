import { test, expect } from '@playwright/test';
import { setupApiMocking, ADMIN_TOKEN, ADMIN_USER, USER_TOKEN, USER_USER } from './mock-helpers';

test.describe('Navigation Separation', () => {
  test('admin sidebar has exactly 18 items with correct labels', async ({ page }) => {
    setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });

    const navLinks = page.locator('nav[aria-label="Main navigation"] a');
    await expect(navLinks).toHaveCount(18);

    const expectedLabels = [
      'Live Dashboard', 'Signal Panel', 'Indicator Panel', 'Strategy Panel',
      'Scoring Board', 'Activations', 'License Management', 'User Onboarding',
      'Subscription Management', 'Billing & Payouts', 'Referral & Commissions',
      'Device Auth', 'Trading Reports', 'Backtesting Reports', 'Logs & Audit',
      'Platform Operations', 'System Health', 'Settings',
    ];

    for (const label of expectedLabels) {
      await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText(label);
    }
  });

  test('user sidebar has exactly 9 items with correct labels', async ({ page }) => {
    setupApiMocking(page, USER_TOKEN, USER_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'user@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard/live', { timeout: 10000 });

    const navLinks = page.locator('nav[aria-label="Main navigation"] a');
    await expect(navLinks).toHaveCount(9);

    const expectedLabels = [
      'Live Dashboard', 'Signals', 'MT4/MT5 Client', 'Strategy Preferences',
      'Trading Reports', 'Backtest', 'Referral & Earnings', 'Billing & Subscription',
      'Settings',
    ];

    for (const label of expectedLabels) {
      await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText(label);
    }
  });
});
