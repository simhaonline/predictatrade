import { test, expect } from '@playwright/test';
import { setupApiMocking, ADMIN_TOKEN, ADMIN_USER, USER_TOKEN, USER_USER } from './mock-helpers';

test.describe('Navigation Separation', () => {
  test('admin sidebar has exactly 35 items with correct labels', async ({ page }) => {
    setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });

    const navLinks = page.locator('nav[aria-label="Main navigation"] a');
    await expect(navLinks).toHaveCount(35);

    const expectedLabels = [
      'Real-Time Console', 'Signal Monitor', 'Indicator Monitor', 'Strategy Panel',
      'Regime Diagnostics', 'Scoring Board', 'Risk Center', 'MT Accounts',
      'Device Auth', 'License Management', 'Activations', 'Users & Onboarding',
      'Subscription Management', 'Plans & Entitlements', 'Billing & Invoices',
      'Commission Operations', 'Payout Operations', 'Referrals & Affiliates',
      'Finance & Referral Reports', 'Market Data', 'Macro Calendar',
      'Macro Intelligence', 'AI Providers', 'Devil Liquidity',
      'Broker Qualification', 'Signal Accuracy', 'Releases', 'Backup & DR',
      'Feature Flags', 'Trading Reports', 'Backtesting', 'Platform Operations',
      'Logs & Audit', 'System Health', 'Settings',
    ];

    for (const label of expectedLabels) {
      await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText(label);
    }
  });

  test('user sidebar has exactly 17 items with correct labels', async ({ page }) => {
    setupApiMocking(page, USER_TOKEN, USER_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'user@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard/live', { timeout: 10000 });

    const navLinks = page.locator('nav[aria-label="Main navigation"] a');
    await expect(navLinks).toHaveCount(17);

    const expectedLabels = [
      'Real-Time Console', 'Signal Accuracy', 'Signals', 'MetaTrader Client',
      'Strategy Preferences', 'Trading Reports', 'Backtest', 'Devil Liquidity',
      'Referral & Earnings', 'Billing & Subscription', 'Payouts', 'License',
      'Security', 'Activity Log', 'Notifications', 'Settings', 'Support',
    ];

    for (const label of expectedLabels) {
      await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText(label);
    }
  });
});
