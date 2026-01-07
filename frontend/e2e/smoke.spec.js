import { test, expect } from '@playwright/test';

/**
 * Smoke tests for critical application functionality
 * These tests verify that the application loads and basic navigation works
 */
test.describe('Application Smoke Tests', () => {
  test('homepage loads successfully', async ({ page }) => {
    await page.goto('/');

    // Check that page loads
    await expect(page).toHaveTitle(/Investment Portfolio Manager/i);

    // Check for navigation
    await expect(page.locator('nav')).toBeVisible();
  });

  test('can navigate to portfolios page', async ({ page }) => {
    await page.goto('/');

    // Click portfolios link
    await page.click('text=Portfolios');

    // Check URL changed
    await expect(page).toHaveURL(/.*portfolio/);

    // Check page content
    await expect(page.locator('h1')).toContainText('Portfolios');
  });

  test('can navigate to funds page', async ({ page }) => {
    await page.goto('/');

    await page.click('text=Funds');

    await expect(page).toHaveURL(/.*fund/);
    await expect(page.locator('h1')).toContainText('Funds');
  });

  test('can navigate to config page', async ({ page }) => {
    await page.goto('/');

    await page.click('text=Config');

    await expect(page).toHaveURL(/.*config/);
  });

  test('backend health check passes', async ({ page }) => {
    // Go to overview page which triggers health check via AppContext
    await page.goto('/');

    // Wait for AppContext to complete health check and version fetch
    await page.waitForTimeout(2000);

    // Check that we're not showing a health check error
    await expect(page.locator('[data-testid="health-check-error"]')).not.toBeVisible();

    // Check that we're not stuck on "Connecting to backend..."
    await expect(page.locator('text=Connecting to backend...')).not.toBeVisible();
  });

  test('application shows version information', async ({ page }) => {
    await page.goto('/config');

    // Wait for version info to load
    await page.waitForTimeout(1000);

    // Check for version section
    await expect(page.locator('text=Version Information')).toBeVisible();
  });
});
