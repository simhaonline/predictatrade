import { test, expect } from '@playwright/test';
import { setupApiMocking, ADMIN_TOKEN, ADMIN_USER, USER_TOKEN, USER_USER } from './mock-helpers';

test.describe('Sequential Account Test', () => {
  test('admin login → logout → user login changes navigation', async ({ page, context }) => {
    // Step 1: Login as admin
    setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);

    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/dashboard', { timeout: 10000 });

    // Verify admin session
    await expect(page.locator('nav[aria-label="Main navigation"]')).toContainText('Signal Monitor');
    await expect(page.locator('[data-testid="topbar-user-name"]')).toContainText('Simha Admin');

    // Step 2: Logout
    await page.click('button[aria-label="Logout"]');
    await page.waitForURL('**/login', { timeout: 10000 });

    // Step 3: Login as user
    // Need to re-setup mocking for the new token
    setupApiMocking(page, USER_TOKEN, USER_USER);

    await page.fill('input[type="email"]', 'user@predictatrade.com');
    await page.fill('input[type="password"]', 'TestPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard/live', { timeout: 10000 });

    // Verify user session — admin items absent
    await expect(page.locator('[data-testid="topbar-user-name"]')).toContainText('Simha User');

    // Verify no admin items
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('Signal Monitor');
    await expect(page.locator('nav[aria-label="Main navigation"]')).not.toContainText('Risk Center');
  });
});
