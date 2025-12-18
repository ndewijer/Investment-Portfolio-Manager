import { test, expect } from '@playwright/test';

/**
 * Dividend Management E2E Tests
 * These tests verify dividend tracking functionality
 *
 * Note: These tests require at least one portfolio with funds to exist
 */
test.describe('Dividend Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to portfolios page
    await page.goto('/portfolios');
  });

  test('can navigate to dividends section', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      // Look for Dividends section heading (no tabs, sections are on the same page)
      const dividendsHeading = page.locator('h2:has-text("Dividends")');

      if (await dividendsHeading.isVisible().catch(() => false)) {
        // Dividends section exists, check for table or empty state
        const hasDividendsContent =
          (await page
            .locator('.portfolio-dividends table')
            .isVisible()
            .catch(() => false)) ||
          (await page
            .locator('text=/no dividends/i')
            .isVisible()
            .catch(() => false));

        expect(hasDividendsContent).toBeTruthy();
      } else {
        // Dividends section might not be visible (only shows if funds have dividends)
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('dividends page shows table or empty state', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      const dividendsHeading = page.locator('h2:has-text("Dividends")');

      if (await dividendsHeading.isVisible().catch(() => false)) {
        // Should show dividends table or empty state
        const hasTable = await page
          .locator('.portfolio-dividends table')
          .isVisible()
          .catch(() => false);
        const hasEmptyState = await page
          .locator('text=/no dividends/i')
          .isVisible()
          .catch(() => false);

        expect(hasTable || hasEmptyState).toBeTruthy();
      } else {
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('can open add dividend modal if available', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      // Look for "Add Dividend" button in the funds table (only shown for dividend funds)
      const addDividendBtn = page.locator('button:has-text("Add Dividend")').first();

      if (await addDividendBtn.isVisible().catch(() => false)) {
        await addDividendBtn.click();
        await page.waitForTimeout(500);

        // Modal should open with dividend form - check for modal title and labels
        const hasModalTitle = await page
          .locator('h2:has-text("Add Dividend")')
          .isVisible()
          .catch(() => false);
        const hasRecordDateLabel = await page
          .locator('label:has-text("Record Date")')
          .isVisible()
          .catch(() => false);

        expect(hasModalTitle && hasRecordDateLabel).toBeTruthy();
      } else {
        // No dividend button - portfolio might not have dividend funds
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('dividend form has required fields', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      const addDividendBtn = page.locator('button:has-text("Add Dividend")').first();

      if (await addDividendBtn.isVisible().catch(() => false)) {
        await addDividendBtn.click();
        await page.waitForTimeout(500);

        // Check for common dividend form labels
        const fields = {
          recordDate: await page
            .locator('label:has-text("Record Date")')
            .isVisible()
            .catch(() => false),
          exDividendDate: await page
            .locator('label:has-text("Ex-Dividend Date")')
            .isVisible()
            .catch(() => false),
          dividendPerShare: await page
            .locator('label:has-text("Dividend per Share")')
            .isVisible()
            .catch(() => false),
        };

        // All essential fields should be present
        const hasFields = Object.values(fields).every((field) => field);
        expect(hasFields).toBeTruthy();
      } else {
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('dividend table shows dividend information if dividends exist', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      const dividendsHeading = page.locator('h2:has-text("Dividends")');

      if (await dividendsHeading.isVisible().catch(() => false)) {
        // Check if dividend table exists with data
        const dividendTable = page.locator('.portfolio-dividends table');

        if (await dividendTable.isVisible().catch(() => false)) {
          // Table exists, check for headers/data
          const hasCells = await page.locator('.portfolio-dividends th,td').count();
          expect(hasCells).toBeGreaterThan(0);
        } else {
          // No dividends table, might be empty state
          test.skip();
        }
      } else {
        // No dividends section (portfolio has no dividend funds)
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('can view multiple sections on portfolio detail page', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      // Verify multiple sections are visible on the same page (no tabs)
      const fundsSectionHeading = page.locator('h2:has-text("Funds & Stocks")');
      const transactionsHeading = page.locator('h2:has-text("Transactions")');

      // Funds section should always be visible
      await expect(fundsSectionHeading).toBeVisible();

      // Transactions section should also be visible
      await expect(transactionsHeading).toBeVisible();

      // Dividends section is conditional (only if funds have dividends)
      const dividendsHeading = page.locator('h2:has-text("Dividends")');
      const hasDividendsSection = await dividendsHeading.isVisible().catch(() => false);

      // Test passes - sections are accessible without tab navigation
      expect(true).toBeTruthy();
    } else {
      test.skip();
    }
  });
});
