import { test, expect } from '@playwright/test';
import { setupApiMocking, ADMIN_TOKEN, ADMIN_USER } from './mock-helpers';

test.describe('Admin Login Flow', () => {
  test('admin login redirects to /admin/dashboard with Admin Panel sidebar', async ({ page }) => {
    setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');

    // Should redirect to admin dashboard
    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });
    expect(page.url()).toContain('/admin/dashboard');

    // Topbar should show admin name
    await expect(page.locator('[data-testid="topbar-user-name"]')).toContainText('Simha Admin');

    // Sidebar footer should say "Admin Panel"
    await expect(page.locator('[data-testid="panel-label"]')).toContainText('Admin Panel');

    // Sidebar should contain admin-specific items
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('Signal Panel');
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('Scoring Board');
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('Platform Operations');
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('System Health');

    // Sidebar should NOT contain user-only items
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('MT4/MT5 Client');
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('Referral & Earnings');
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('Billing & Subscription');
  });

  test('admin navigating to /dashboard/live redirects to /admin/dashboard', async ({ page }) => {
    setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);

    // Login first
    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });

    // Try to navigate to user dashboard
    await page.goto('/dashboard/live');
    // Should be redirected back to admin dashboard
    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });
    expect(page.url()).toContain('/admin/dashboard');
  });
});
