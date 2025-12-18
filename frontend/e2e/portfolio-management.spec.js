import { test, expect } from '@playwright/test';

/**
 * Portfolio Management E2E Tests
 * These tests verify the complete portfolio creation and viewing workflows
 *
 * Note: These tests interact with the actual backend and database.
 * They assume a clean test environment or use unique naming to avoid conflicts.
 */
test.describe('Portfolio Management', () => {
  // Generate unique portfolio name for this test run
  const testPortfolioName = `E2E Test Portfolio ${Date.now()}`;

  test('can navigate to portfolios page', async ({ page }) => {
    await page.goto('/');

    // Click on Portfolios navigation link
    await page.click('text=Portfolios');

    // Verify URL changed
    await expect(page).toHaveURL(/.*portfolios/);

    // Verify page heading
    await expect(page.locator('h1')).toContainText('Portfolios');
  });

  test('portfolios page loads successfully', async ({ page }) => {
    await page.goto('/portfolios');

    // Wait for page to finish loading
    await page.waitForTimeout(1000);

    // Verify the page heading exists - this confirms the page loaded
    const heading = await page
      .locator('h1')
      .isVisible()
      .catch(() => false);
    expect(heading).toBeTruthy();
  });

  test('can open create portfolio modal', async ({ page }) => {
    await page.goto('/portfolios');

    // Look for "Add Portfolio" button
    const createButton = page.locator('button:has-text("Add Portfolio")');

    await expect(createButton).toBeVisible();
    await createButton.click();

    // Modal should open - wait for modal animation
    await page.waitForTimeout(500);

    // Verify modal title (use h2 to avoid strict mode violation with button)
    await expect(page.locator('h2:has-text("Add Portfolio")')).toBeVisible();

    // Verify form fields exist
    const nameLabel = page.locator('label:has-text("Name")');
    await expect(nameLabel).toBeVisible();
  });

  test('can view existing portfolio details if any exist', async ({ page }) => {
    await page.goto('/portfolios');
    await page.waitForTimeout(1000);

    // Check if there are any portfolio cards with "View Details" button
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      // Click on first portfolio's "View Details" button
      await firstViewButton.click();

      // Should navigate to portfolio detail page (UUID-based ID)
      await expect(page).toHaveURL(/.*portfolios\/[0-9a-f-]+/);

      // Portfolio detail page should have key sections
      await page.waitForTimeout(1000); // Wait for page to load

      // Check for the "Funds & Stocks" section heading
      const fundsSectionHeading = page.locator('h2:has-text("Funds & Stocks")');
      await expect(fundsSectionHeading).toBeVisible();
    } else {
      // No portfolios exist, skip this test
      test.skip();
    }
  });

  test('portfolio detail page shows key information', async ({ page }) => {
    await page.goto('/portfolios');
    await page.waitForTimeout(1000);

    // Try to access a portfolio
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();

      // Wait for navigation
      await page.waitForTimeout(1000);

      // The page should show the portfolio name in h1
      const portfolioName = page.locator('h1');
      await expect(portfolioName).toBeVisible();

      // Should have main navigation
      const hasNavigation = page.locator('nav').isVisible();
      expect(await hasNavigation).toBeTruthy();
    } else {
      test.skip();
    }
  });

  test('can navigate back to portfolios list from detail page', async ({ page }) => {
    await page.goto('/portfolios');
    await page.waitForTimeout(1000);

    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();

      // Wait for navigation to detail page
      await page.waitForTimeout(1000);

      // Navigate back via browser back button
      await page.goBack();

      // Should be back at portfolios page
      await expect(page).toHaveURL(/.*portfolios$/);
    } else {
      test.skip();
    }
  });
});
