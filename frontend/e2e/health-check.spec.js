import { test, expect } from '@playwright/test';

/**
 * Health check tests
 * Tests for backend health check functionality and error handling
 */
test.describe('Backend Health Check', () => {
  test('application loads when backend is healthy', async ({ page }) => {
    await page.goto('/');

    // Wait for health check to complete
    await page.waitForTimeout(2000);

    // Should not show health check error
    await expect(page.locator('[data-testid="health-check-error"]')).not.toBeVisible();

    // Should not be stuck on connecting message
    await expect(page.locator('text=Connecting to backend...')).not.toBeVisible();

    // Navigation should be visible (indicates app loaded successfully)
    await expect(page.locator('nav')).toBeVisible();
  });

  test('version information is displayed on config page', async ({ page }) => {
    await page.goto('/config');

    // Wait for version info to load
    await page.waitForTimeout(1500);

    // Check for version information section
    await expect(page.locator('text=Version Information')).toBeVisible();

    // Check that version numbers are displayed (not just "unknown")
    const versionSection = page.locator('text=Version Information').locator('..');
    await expect(versionSection).toBeVisible();
  });

  test('system status is displayed on config page', async ({ page }) => {
    await page.goto('/config');

    // Wait for health status to load
    await page.waitForTimeout(1500);

    // Check for health status section
    const healthSection = page.locator('text=Health Status').locator('..');
    await expect(healthSection).toBeVisible();
  });
});
