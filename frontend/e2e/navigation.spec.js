import { test, expect } from '@playwright/test';

/**
 * Navigation tests
 * Tests for application navigation and routing
 */
test.describe('Navigation', () => {
  test.beforeEach(async ({ page }) => {
    // Start at home page before each test
    await page.goto('/');
  });

  test('navigation bar is visible on all pages', async ({ page }) => {
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    // Navigate to different pages and verify nav is still visible
    await page.click('text=Portfolios');
    await expect(nav).toBeVisible();

    await page.click('text=Funds');
    await expect(nav).toBeVisible();

    await page.click('text=Config');
    await expect(nav).toBeVisible();
  });

  test('navigation links are clickable', async ({ page }) => {
    // Test Overview link
    await page.click('text=Overview');
    await expect(page).toHaveURL(/.*\//);

    // Test Portfolios link
    await page.click('text=Portfolios');
    await expect(page).toHaveURL(/.*portfolios/);
    await expect(page.locator('h1')).toContainText('Portfolios');

    // Test Funds link
    await page.click('text=Funds');
    await expect(page).toHaveURL(/.*funds/);
    await expect(page.locator('h1')).toContainText('Funds');

    // Test Config link
    await page.click('text=Config');
    await expect(page).toHaveURL(/.*config/);
  });

  test('browser back button works', async ({ page }) => {
    // Navigate through pages
    await page.click('text=Portfolios');
    await page.click('text=Funds');

    // Go back
    await page.goBack();
    await expect(page).toHaveURL(/.*portfolios/);

    // Go back again
    await page.goBack();
    await expect(page).toHaveURL(/.*\//);
  });

  test('browser forward button works', async ({ page }) => {
    // Navigate through pages
    await page.click('text=Portfolios');
    await page.click('text=Funds');

    // Go back twice
    await page.goBack();
    await page.goBack();

    // Go forward
    await page.goForward();
    await expect(page).toHaveURL(/.*portfolios/);

    // Go forward again
    await page.goForward();
    await expect(page).toHaveURL(/.*funds/);
  });

  test('direct URL navigation works', async ({ page }) => {
    // Navigate directly to portfolios page
    await page.goto('/portfolios');
    await expect(page.locator('h1')).toContainText('Portfolios');

    // Navigate directly to funds page
    await page.goto('/funds');
    await expect(page.locator('h1')).toContainText('Funds');

    // Navigate directly to config page
    await page.goto('/config');
    await expect(page).toHaveURL(/.*config/);
  });
});
