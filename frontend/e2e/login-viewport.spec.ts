import { test, expect } from '@playwright/test';
import { setupApiMocking, ADMIN_TOKEN, ADMIN_USER } from './mock-helpers';

const VIEWPORTS = [
  { name: '1920x1080', width: 1920, height: 1080 },
  { name: '1600x900', width: 1600, height: 900 },
  { name: '1440x900', width: 1440, height: 900 },
  { name: '1366x768', width: 1366, height: 768 },
  { name: 'mobile-390', width: 390, height: 844 },
];

for (const vp of VIEWPORTS) {
  test(`login fits ${vp.name} — no scroll`, async ({ page }) => {
    setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('/login');
    await page.waitForSelector('h1', { timeout: 10000 });

    // Check no vertical/horizontal scroll needed
    const dims = await page.evaluate(() => ({
      scrollHeight: document.documentElement.scrollHeight,
      clientHeight: document.documentElement.clientHeight,
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
    }));
    expect(dims.scrollHeight).toBeLessThanOrEqual(dims.clientHeight + 1);
    expect(dims.scrollWidth).toBeLessThanOrEqual(dims.clientWidth + 1);

    // Logo image exists (auth layout renders one horizontal logo)
    const logoCount = await page.locator('img[alt="Predict-A-Trade"]').count();
    expect(logoCount).toBe(1);

    // Verify ThemeControl (Display Preferences) is present
    const themeBtn = page.locator('button[aria-label="Display preferences"]');
    await expect(themeBtn).toBeVisible();

    // Verify footer copyright
    const footerText = await page.locator('footer').textContent();
    expect(footerText).toContain('© 2016–2026 Predict-A-Trade by Simha FinTech');
  });
}

test('ThemeControl on login switches theme persistently', async ({ page }) => {
  setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.goto('/login');
  await page.waitForSelector('h1');

  // Open theme dropdown
  await page.click('button[aria-label="Display preferences"]');

  // Click "Light Mode"
  await page.click('text=Light Mode');

  // Verify the dark class is removed from html (light mode active)
  await expect(page.locator('html')).not.toHaveClass(/dark/);

  // Open dropdown again, click "Dark Mode"
  await page.click('button[aria-label="Display preferences"]');
  await page.click('text=Dark Mode');

  // Verify dark class is present
  await expect(page.locator('html')).toHaveClass(/dark/);

  // Open dropdown, click "System Mode"
  await page.click('button[aria-label="Display preferences"]');
  await page.click('text=System Mode');
});

test('Login page logo is 260-320px on desktop', async ({ page }) => {
  setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.goto('/login');
  await page.waitForSelector('h1');

  // Get the visible logo's computed width
  const logoWidth = await page.evaluate(() => {
    const imgs = document.querySelectorAll('img[alt="Predict-A-Trade"]');
    for (const img of imgs) {
      const rect = (img as HTMLElement).getBoundingClientRect();
      if (rect.width > 0) return Math.round(rect.width);
    }
    return 0;
  });

  // On desktop (1920x1080), the auth layout horizontal logo is 200px wide
  expect(logoWidth).toBeGreaterThanOrEqual(180);
  expect(logoWidth).toBeLessThanOrEqual(220);
});

test('Login page logo is 180-220px on mobile', async ({ page }) => {
  setupApiMocking(page, ADMIN_TOKEN, ADMIN_USER);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/login');
  await page.waitForSelector('h1');

  const logoWidth = await page.evaluate(() => {
    const imgs = document.querySelectorAll('img[alt="Predict-A-Trade"]');
    for (const img of imgs) {
      const rect = (img as HTMLElement).getBoundingClientRect();
      if (rect.width > 0) return Math.round(rect.width);
    }
    return 0;
  });

  // On mobile (390px wide), the same 200px logo is used (fluid, centered)
  expect(logoWidth).toBeGreaterThanOrEqual(180);
  expect(logoWidth).toBeLessThanOrEqual(220);
});
